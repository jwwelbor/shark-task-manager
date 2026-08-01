package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	questionrepo "github.com/jwwelbor/shark-task-manager/internal/repository/question"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// QuestionRepository is the direct persistence seam for QuestionService.
// The F01 transport slice owns its read, update, and status methods; this
// service owns direct creation and deletion orchestration.
type QuestionRepository interface {
	Create(ctx context.Context, question *models.Question) error
	GetByKey(ctx context.Context, key string) (*models.Question, error)
	GetByID(ctx context.Context, id int64) (*models.Question, error)
	List(ctx context.Context, filter questionrepo.QuestionListFilter) ([]*models.Question, error)
	ListOpenCandidates(ctx context.Context, limit, offset int) ([]*models.Question, error)
	Update(ctx context.Context, question *models.Question) error
	UpdateStatus(ctx context.Context, id int64, status models.QuestionStatus) error
	ConfigureWorkflow(ctx context.Context, id int64, expectedStatus models.QuestionStatus, expectedContextData, contextData *string, resolutionOwner string) error
	RecordResponse(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, responder string) error
	FollowUpWorkExists(ctx context.Context, key string) (bool, error)
	NoteExists(ctx context.Context, noteID string) (bool, error)
	Resolve(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, owner, kind string) error
	Withdraw(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, owner, reason string) error
	Delete(ctx context.Context, id int64) error
}

// QuestionUpdates is the finite mutable base-record surface. A nil field means
// leave the stored value unchanged; status is intentionally excluded because
// workflow routing owns status transitions.
type QuestionUpdates struct {
	Title       *string
	Summary     *string
	Requester   *string
	Description *string
	Blocking    *bool
}

// QuestionListFilter is the service-owned finite query shape for Question
// list transports. Repository adapters may translate it to their SQL filter.
type QuestionListFilter struct {
	Status    *models.QuestionStatus
	Requester *string
	Blocking  *bool
	Limit     int
	Offset    int
}

// QuestionFullReadDeniedError is the typed policy result for an actor that is
// not the Question's current responder or configured resolution owner.
type QuestionFullReadDeniedError struct{ Key string }

func (e *QuestionFullReadDeniedError) Error() string {
	return fmt.Sprintf("full Question read for %s is not authorized", e.Key)
}

// CreateQuestionInput is the bounded base-record shape for direct Question
// creation. It intentionally excludes workflow, responder, and provenance
// fields owned by later features.
type CreateQuestionInput struct {
	Title       string
	Summary     string
	Requester   string
	Description string
	Blocking    bool
	Status      string
}

// QuestionClaimReader is the read-only portion of ClaimService needed by the
// Question domain. It intentionally cannot claim, release, or heartbeat.
type QuestionClaimReader interface {
	Get(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error)
}

// QuestionFocusedRelationshipReader is the bounded direct-edge seam used only
// by the focused blocking read. It cannot create, mutate, or traverse edges.
type QuestionFocusedRelationshipReader interface {
	GetIncomingPage(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType, limit, offset int) ([]*models.EntityRelationship, error)
}

// QuestionService owns direct creation of the Question base record.
type QuestionService struct {
	repo            QuestionRepository
	historyRepo     EntityHistoryRecorder // optional: records the initial draft status
	searchIndexer   SearchIndexer         // optional: keeps Question discovery current
	claimReader     QuestionClaimReader
	focusedEdges    QuestionFocusedRelationshipReader
	focusedRegistry QuestionBlockerRegistry
	projectRoot     string
	entitySvc       *EntityService // optional: shared config-driven transition engine, see SetEntityTransitioner
	entityRepo      EntityRepository
}

// NewQuestionService constructs the Question lifecycle service.
func NewQuestionService(repo QuestionRepository) (*QuestionService, error) {
	if repo == nil {
		return nil, fmt.Errorf("QuestionService requires a non-nil QuestionRepository")
	}
	return &QuestionService{repo: repo, projectRoot: "."}, nil
}

// SetProjectRoot sets the canonical root used to validate resolution documents.
func (s *QuestionService) SetProjectRoot(projectRoot string) { s.projectRoot = projectRoot }

// SetHistoryRepo sets the optional shared entity-history recorder. Creation
// history follows the existing non-blocking audit convention: a recorder
// failure is logged by recordEntityHistory but does not undo a successfully
// persisted Question.
func (s *QuestionService) SetHistoryRepo(repo EntityHistoryRecorder) {
	s.historyRepo = repo
}

// SetSearchIndexer wires the optional unified-search indexer used after
// Question writes. Search indexing is best effort, matching the existing
// standalone entity services.
func (s *QuestionService) SetSearchIndexer(indexer SearchIndexer) {
	s.searchIndexer = indexer
}

// SetClaimReader wires the existing single-lease authority into response
// validation without giving the Question domain lifecycle authority.
func (s *QuestionService) SetClaimReader(reader QuestionClaimReader) { s.claimReader = reader }

// SetFocusedReadDependencies wires the read-only target registry and bounded
// incoming-edge reader required by ListQuestionsBlocking. Keeping this seam
// explicit lets CLI and HTTP share the same service without granting either
// transport relationship mutation authority.
func (s *QuestionService) SetFocusedReadDependencies(edges QuestionFocusedRelationshipReader, registry QuestionBlockerRegistry) {
	s.focusedEdges = edges
	s.focusedRegistry = registry
}

// SetEntityTransitioner wires the shared, config-driven transition engine
// every sibling entity service (Bug, ChangeCard, TechDebt, Task, Feature,
// Epic) uses for TransitionStatus, via the level-scoped workflow.Service and
// the existing Question EntityRepository adapter. entitySvc is stored scoped
// to workflow.LevelQuestion. Optional: TransitionStatus falls back to a
// hand-rolled check when unset, matching prior behavior for callers (tests)
// that don't wire it.
func (s *QuestionService) SetEntityTransitioner(entitySvc *EntityService, entityRepo EntityRepository) {
	s.entitySvc = entitySvc.ForLevel(workflow.LevelQuestion)
	s.entityRepo = entityRepo
}

// CreateQuestion validates the finite F01 input contract before persistence.
// The repository atomically allocates and persists the canonical Q### identity.
// It then reloads that record so callers receive database-assigned fields such
// as created_at and updated_at instead of the pre-insert model.
func (s *QuestionService) CreateQuestion(ctx context.Context, input CreateQuestionInput) (*models.Question, error) {
	title := strings.TrimSpace(input.Title)
	summary := strings.TrimSpace(input.Summary)
	requester := strings.TrimSpace(input.Requester)
	if title == "" {
		return nil, fmt.Errorf("question title is required")
	}
	if summary == "" {
		return nil, fmt.Errorf("question summary is required")
	}
	if requester == "" {
		return nil, fmt.Errorf("question requester is required")
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status == "" {
		status = string(models.QuestionStatusDraft)
	}
	if status != string(models.QuestionStatusDraft) {
		return nil, fmt.Errorf("question status must be %q at creation, got %q", models.QuestionStatusDraft, input.Status)
	}

	slug := utils.GenerateSlug(title)
	question := &models.Question{
		BaseEntity: models.BaseEntity{
			Title: title,
			Slug:  &slug,
		},
		Status:    models.QuestionStatusDraft,
		Summary:   summary,
		Blocking:  input.Blocking,
		Requester: requester,
	}
	if input.Description != "" {
		description := input.Description
		question.Description = &description
	}

	if err := s.repo.Create(ctx, question); err != nil {
		return nil, fmt.Errorf("create question: %w", err)
	}
	// A newly persisted Question begins its auditable lifecycle in draft. The
	// repository assigns its identity before this call, so the generic history
	// row refers to the durable Question rather than a caller-supplied model.
	recordEntityHistory(ctx, s.historyRepo, models.EntityTypeQuestion, question.ID,
		"", string(question.Status), false, EntityHistoryOpts{})
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, question.ID); err != nil {
		return nil, err
	}
	persisted, err := s.repo.GetByKey(ctx, question.Key)
	if err != nil {
		return nil, fmt.Errorf("reload created question %s: %w", question.Key, err)
	}
	if persisted == nil {
		return nil, fmt.Errorf("reload created question %s: repository returned no question", question.Key)
	}
	return persisted, nil
}

// GetQuestion returns a canonical persisted Question.
func (s *QuestionService) GetQuestion(ctx context.Context, key string) (*models.Question, error) {
	question, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get question %s: %w", key, err)
	}
	return question, nil
}

// GetQuestionByID returns a persisted Question by its relationship source ID.
// It is a read-only seam for cross-cutting consumers such as QuestionBlocker.
func (s *QuestionService) GetQuestionByID(ctx context.Context, id int64) (*models.Question, error) {
	question, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get question %d: %w", id, err)
	}
	if question == nil {
		return nil, fmt.Errorf("get question %d: repository returned no question", id)
	}
	return question, nil
}

// ListQuestions returns a bounded Question page using the repository's finite
// filter contract.
func (s *QuestionService) ListQuestions(ctx context.Context, filter QuestionListFilter) ([]*models.Question, error) {
	questions, err := s.repo.List(ctx, questionrepo.QuestionListFilter{
		Status: filter.Status, Requester: filter.Requester, Blocking: filter.Blocking,
		Limit: filter.Limit, Offset: filter.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	return questions, nil
}

// ListOpenQuestionsByResponder returns the compact page for one exact current
// responder. Candidate status filtering is intentionally delegated to the
// bounded repository read; this service validates each persisted state before
// deriving identity, so claims or response text cannot influence selection.
func (s *QuestionService) ListOpenQuestionsByResponder(ctx context.Context, responder string, limit, offset int) ([]models.QuestionProjection, error) {
	if err := ValidateQuestionReadIdentity("responder", responder); err != nil {
		return nil, err
	}
	limit, err := normalizeQuestionReadPage(limit, offset)
	if err != nil {
		return nil, err
	}
	return s.collectOpenQuestionsByResponder(ctx, responder, limit, offset)
}

func (s *QuestionService) collectOpenQuestionsByResponder(ctx context.Context, responder string, limit, offset int) ([]models.QuestionProjection, error) {
	result := make([]models.QuestionProjection, 0, limit)
	candidateOffset, skipped := 0, offset
	for {
		questions, err := s.repo.ListOpenCandidates(ctx, limit, candidateOffset)
		if err != nil {
			return nil, fmt.Errorf("list open Questions for responder: %w", err)
		}
		for _, question := range questions {
			if question == nil {
				return nil, errors.New("list open Questions for responder: repository returned a nil Question")
			}
			if question.Status != models.QuestionStatusOpen && question.Status != models.QuestionStatusAnswering {
				continue
			}
			state, decodeErr := models.DecodeQuestionState(question.ContextData)
			if decodeErr != nil {
				return nil, fmt.Errorf("list open Questions for responder: decode state for %s: %w", question.Key, decodeErr)
			}
			if state == nil {
				return nil, fmt.Errorf("list open Questions for responder: Question workflow state is required for %s", question.Key)
			}
			if state.CurrentResponder() != responder {
				continue
			}
			if skipped > 0 {
				skipped--
				continue
			}
			result = append(result, models.ProjectQuestion(question))
			if len(result) == limit {
				return result, nil
			}
		}
		if len(questions) < limit {
			return result, nil
		}
		candidateOffset += len(questions)
	}
}

// ListQuestionsBlocking returns the compact I-03 handoffs for direct,
// qualifying Question blockers of one non-Question target. Qualification is
// shared with QuestionBlocker so focused reads and dispatch preflight cannot
// disagree about source status, blocking flag, or Question state validity.
func (s *QuestionService) ListQuestionsBlocking(ctx context.Context, targetType models.EntityType, targetKey string, limit, offset int) ([]*QuestionBlock, error) {
	if targetType == models.EntityTypeQuestion {
		return nil, errors.New("list Questions blocking target: Question targets are not supported")
	}
	if s.focusedEdges == nil || s.focusedRegistry == nil {
		return nil, errors.New("list Questions blocking target: focused read dependencies are not configured")
	}
	limit, err := normalizeQuestionReadPage(limit, offset)
	if err != nil {
		return nil, err
	}
	repository, err := s.focusedRegistry.GetRepository(targetType)
	if err != nil {
		return nil, fmt.Errorf("list Questions blocking target: resolve %s: %w", targetType, err)
	}
	target, err := repository.GetByKey(ctx, targetKey)
	if err != nil {
		return nil, fmt.Errorf("list Questions blocking target: load %s: %w", targetKey, err)
	}
	if target == nil {
		return nil, errors.New("list Questions blocking target: target repository returned no entity")
	}
	return s.collectQuestionsBlocking(ctx, targetType, target.GetID(), limit, offset)
}

func (s *QuestionService) collectQuestionsBlocking(ctx context.Context, targetType models.EntityType, targetID int64, limit, offset int) ([]*QuestionBlock, error) {
	matches := make([]questionBlockMatch, 0, limit)
	edgeOffset, skipped := 0, offset
	for {
		edges, err := s.focusedEdges.GetIncomingPage(ctx, targetType, targetID, []models.EntityRelationshipType{models.EntityRelQuestionBlocks}, limit, edgeOffset)
		if err != nil {
			return nil, fmt.Errorf("list Questions blocking target: load direct edges: %w", err)
		}
		for _, edge := range edges {
			if edge == nil || edge.RelationshipType != models.EntityRelQuestionBlocks || edge.FromEntityType != models.EntityTypeQuestion {
				continue
			}
			question, readErr := s.GetQuestionByID(ctx, edge.FromEntityID)
			if readErr != nil {
				return nil, fmt.Errorf("list Questions blocking target: load source Question %d: %w", edge.FromEntityID, readErr)
			}
			block, qualifyErr := QualifyQuestionBlock(question)
			if qualifyErr != nil {
				return nil, fmt.Errorf("list Questions blocking target: %w", qualifyErr)
			}
			if block != nil {
				if skipped > 0 {
					skipped--
					continue
				}
				matches = append(matches, questionBlockMatch{edge: edge, block: block})
				if len(matches) == limit {
					break
				}
			}
		}
		if len(matches) == limit || len(edges) < limit {
			break
		}
		edgeOffset += len(edges)
	}
	sortQuestionBlockMatches(matches)
	blocks := make([]*QuestionBlock, 0, len(matches))
	for _, match := range matches {
		blocks = append(blocks, match.block)
	}
	return blocks, nil
}

// ReadQuestionFull returns the intentionally separate full projection only to
// the current responder or resolution owner. It performs no fallback to the
// persisted model and performs no writes.
func (s *QuestionService) ReadQuestionFull(ctx context.Context, key, actor string) (*models.QuestionFullProjection, error) {
	if err := ValidateQuestionReadIdentity("actor", actor); err != nil {
		return nil, err
	}
	question, err := s.GetQuestion(ctx, key)
	if err != nil {
		return nil, err
	}
	state, err := models.DecodeQuestionState(question.ContextData)
	if err != nil {
		return nil, fmt.Errorf("read full Question %s: decode state: %w", key, err)
	}
	if state == nil {
		return nil, fmt.Errorf("read full Question %s: Question workflow state is required", key)
	}
	if actor != state.CurrentResponder() && actor != state.ResolutionOwner {
		return nil, &QuestionFullReadDeniedError{Key: question.Key}
	}
	projection := models.ProjectQuestionFull(question, *state)
	return &projection, nil
}

// ValidateQuestionReadIdentity applies the finite identity contract shared by
// focused Question read transports and their service policy seam.
func ValidateQuestionReadIdentity(field, identity string) error {
	if err := models.ValidateQuestionBoundedText(field, identity, 1, 256); err != nil {
		return fmt.Errorf("Question read %s: %w", field, err)
	}
	return nil
}

func normalizeQuestionReadPage(limit, offset int) (int, error) {
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return 0, fmt.Errorf("Question read limit must be between 1 and 100, got %d", limit)
	}
	if offset < 0 {
		return 0, fmt.Errorf("Question read offset must be zero or greater, got %d", offset)
	}
	return limit, nil
}

// UpdateQuestion applies only the finite F01 mutable fields. Required text
// values are trimmed once and rejected before the repository is called.
func (s *QuestionService) UpdateQuestion(ctx context.Context, key string, updates QuestionUpdates) (*models.Question, error) {
	if updates.Title == nil && updates.Summary == nil && updates.Requester == nil && updates.Description == nil && updates.Blocking == nil {
		return nil, fmt.Errorf("question update requires at least one supported field")
	}
	question, err := s.GetQuestion(ctx, key)
	if err != nil {
		return nil, err
	}
	if updates.Title != nil {
		question.Title = strings.TrimSpace(*updates.Title)
	}
	if updates.Summary != nil {
		question.Summary = strings.TrimSpace(*updates.Summary)
	}
	if updates.Requester != nil {
		question.Requester = strings.TrimSpace(*updates.Requester)
	}
	if updates.Description != nil {
		question.Description = updates.Description
	}
	if updates.Blocking != nil {
		question.Blocking = *updates.Blocking
	}
	if err := question.Validate(); err != nil {
		return nil, fmt.Errorf("validate question update: %w", err)
	}
	if err := s.repo.Update(ctx, question); err != nil {
		return nil, fmt.Errorf("update question %s: %w", key, err)
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, question.ID); err != nil {
		return nil, err
	}
	return question, nil
}

// SetQuestionStatus is the narrow persistence seam used by the workflow
// adapter. It deliberately does not validate transition policy here.
func (s *QuestionService) SetQuestionStatus(ctx context.Context, key, status string) (*models.Question, error) {
	question, err := s.GetQuestion(ctx, key)
	if err != nil {
		return nil, err
	}
	next := models.QuestionStatus(strings.ToLower(strings.TrimSpace(status)))
	if next == "" {
		return nil, fmt.Errorf("question status cannot be empty")
	}
	if err := s.repo.UpdateStatus(ctx, question.ID, next); err != nil {
		return nil, fmt.Errorf("update question %s status: %w", key, err)
	}
	question.Status = next
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, question.ID); err != nil {
		return nil, err
	}
	return question, nil
}

// GetNextStatus implements runner.EntityTransitioner's read side for the
// deliberately minimal F01 Question workflow. The workflow fixture supplies
// the draft pause action; this service only reports persisted state and the
// terminal archived boundary, so keyed-next cannot mutate a Question while it
// resolves a dispatch response.
//
// This intentionally does not delegate to EntityService.GetNextStatus (unlike
// TransitionStatus above): question.yaml's ready_for_resolution/resolved/
// withdrawn/superseded steps declare non-empty `outcomes:` for Resolve/
// Withdraw/Supersede's typed use, but those are human-owned resolution-owner
// actions, not machine-dispatchable transitions. The generic engine would
// read that same config and report ready_for_resolution as non-terminal with
// available transitions, which would make `shark next`/`shark run` treat a
// human-owned checkpoint as dispatchable again -- the exact regression this
// method's IsTerminal override below exists to prevent.
func (s *QuestionService) GetNextStatus(ctx context.Context, key string) (*NextStatusInfo, error) {
	question, err := s.GetQuestion(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get next status for question %s: %w", key, err)
	}
	status := string(question.Status)
	if status == string(models.QuestionStatusOpen) || status == string(models.QuestionStatusAnswering) {
		state, err := models.DecodeQuestionState(question.ContextData)
		if err != nil {
			return nil, fmt.Errorf("get next status for question %s: %w", key, err)
		}
		if state == nil || state.CurrentResponder() == "" {
			return &NextStatusInfo{EntityType: models.EntityTypeQuestion, EntityKey: question.Key, CurrentStatus: status, IsTerminal: true}, nil
		}
		info := &NextStatusInfo{
			EntityType:    models.EntityTypeQuestion,
			EntityKey:     question.Key,
			CurrentStatus: status,
			// A parent loop advances the worker stage while it owns the lease.
			// Answering intentionally self-transitions: RecordResponse alone
			// establishes whether all responders are complete and moves to
			// ready_for_resolution.
			AvailableTransitions: []TransitionInfoWithAction{{
				TransitionInfo: workflow.TransitionInfo{TargetStatus: string(models.QuestionStatusAnswering)},
			}},
			Outcomes: map[string]string{"pass": string(models.QuestionStatusAnswering)},
		}
		if s.claimReader == nil {
			return info, nil
		}
		claim, err := s.claimReader.Get(ctx, string(models.EntityTypeQuestion), question.Key)
		if err != nil {
			return nil, fmt.Errorf("get next status for question %s: load claim: %w", key, err)
		}
		info.IsClaimed = claim != nil
		return info, nil
	}
	return &NextStatusInfo{
		EntityType:           models.EntityTypeQuestion,
		EntityKey:            question.Key,
		CurrentStatus:        status,
		AvailableTransitions: []TransitionInfoWithAction{},
		// A persisted F01 draft has no configured responder state. It must use
		// the normal keyed-next pause compatibility path instead of reaching the
		// responder prompt renderer without a durable identity.
		// ready_for_resolution is not a terminal durable status, but it is a
		// non-dispatching checkpoint: all responders are complete and the next
		// operation belongs to the resolution owner. Mark it terminal to the
		// responder dispatch adapters so keyed next, run, and cascades stop before
		// attempting to render a nonexistent current responder.
		IsTerminal: status == string(models.QuestionStatusDraft) || status == string(models.QuestionStatusArchived) || status == string(models.QuestionStatusReadyForResolution) || status == string(models.QuestionStatusResolved) || status == string(models.QuestionStatusWithdrawn) || status == string(models.QuestionStatusSuperseded),
	}, nil
}

// TransitionStatus satisfies runner.EntityTransitioner for the bounded
// Question workflow. GetNextStatus above only ever offers "answering" (from
// open/answering) and "archived" (from draft) as targets, so this only ever
// needs to validate and persist that narrow pair -- but it does so through
// the same shared, config-driven EntityService.TransitionStatus every
// sibling entity (Bug, ChangeCard, TechDebt, Task, Feature, Epic) uses, when
// SetEntityTransitioner has wired one in, so the transition is validated
// against question.yaml instead of a hardcoded Go check and gets
// entity_history recording for free. Falls back to the direct, hand-rolled
// path when unset (e.g. mock-backed unit tests that don't wire a full
// EntityService/workflow.Service).
func (s *QuestionService) TransitionStatus(ctx context.Context, key, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
	if s.entitySvc != nil && s.entityRepo != nil {
		result, err := s.entitySvc.TransitionStatus(ctx, s.entityRepo, models.EntityTypeQuestion, key, targetStatus, opts, SimpleTransitionFeatures(), nil)
		if err != nil {
			return nil, fmt.Errorf("transition question %s: %w", key, err)
		}
		if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, result.EntityID); err != nil {
			return nil, err
		}
		return result, nil
	}

	question, err := s.GetQuestion(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("transition question %s: %w", key, err)
	}
	target := strings.ToLower(strings.TrimSpace(targetStatus))
	valid := (target == string(models.QuestionStatusAnswering) && (question.Status == models.QuestionStatusOpen || question.Status == models.QuestionStatusAnswering)) ||
		(target == string(models.QuestionStatusArchived) && question.Status == models.QuestionStatusDraft)
	if !valid {
		return nil, fmt.Errorf("cannot transition question %s from %q to %q", key, question.Status, targetStatus)
	}
	if _, err := s.SetQuestionStatus(ctx, key, target); err != nil {
		return nil, fmt.Errorf("transition question %s to %s: %w", key, target, err)
	}
	return &TransitionResult{
		EntityType:   models.EntityTypeQuestion,
		EntityKey:    key,
		EntityID:     question.ID,
		FromStatus:   string(question.Status),
		ToStatus:     target,
		Transitioned: true,
	}, nil
}

// DeleteQuestion resolves a Question by its canonical key and delegates the
// single atomic delete to the typed repository. The repository delete executes
// the Question table's cleanup triggers in the same database statement, so a
// cleanup failure leaves both the base row and its generic associations intact.
func (s *QuestionService) DeleteQuestion(ctx context.Context, key string) error {
	question, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("get question %s for deletion: %w", key, err)
	}
	if question == nil {
		return fmt.Errorf("get question %s for deletion: repository returned no question", key)
	}
	if err := s.repo.Delete(ctx, question.ID); err != nil {
		return fmt.Errorf("delete question %s: %w", key, err)
	}
	if err := removeEntityFromIndexIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, question.ID); err != nil {
		return err
	}
	return nil
}
