package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ConfigureWorkflowInput is the only F02 input that may create pending
// responders. It intentionally contains no raw JSON representation.
type ConfigureWorkflowInput struct {
	Key             string
	ResolutionOwner string
	Responders      []string
}

// RecordQuestionResponseInput is the bounded successful-response shape. The
// lease session is checked before this value reaches the Question transaction.
type RecordQuestionResponseInput struct {
	Key             string
	SessionID       string
	Responder       string
	Summary         string
	EvidencePointer string
}

// ResolveQuestionInput closes a fully answered Question with its classified,
// already-local resolution evidence. It does not carry any linked-work mutation.
type ResolveQuestionInput struct{ Key, Owner, Kind, Pointer string }

// WithdrawQuestionInput closes an eligible Question with a bounded reason.
type WithdrawQuestionInput struct{ Key, Owner, Reason string }

// SupersedeQuestionInput closes an eligible Question in favor of another
// existing Question. The superseding Question is read only.
type SupersedeQuestionInput struct{ Key, Owner, Reason, SupersededBy string }

// ConfigureWorkflow validates and initializes the single serial responder
// state for an open Question. A newly created F01-compatible draft is promoted
// to open atomically with its first F02 configuration. The repository commits state, concise typed note,
// and history together; no invalid or repeat configuration reaches a write.
func (s *QuestionService) ConfigureWorkflow(ctx context.Context, input ConfigureWorkflowInput) (*models.Question, error) {
	question, err := s.GetQuestion(ctx, input.Key)
	if err != nil {
		return nil, err
	}
	if question.Status != models.QuestionStatusDraft && question.Status != models.QuestionStatusOpen {
		return nil, fmt.Errorf("configure Question workflow %s: %w", question.Key, questionConflictError(errors.New("Question must be draft or open")))
	}
	if existing, err := models.DecodeQuestionState(question.ContextData); err != nil {
		return nil, fmt.Errorf("configure Question workflow %s: %w", question.Key, err)
	} else if existing != nil {
		return nil, fmt.Errorf("configure Question workflow %s: %w", question.Key, questionConflictError(errors.New("Question is already configured")))
	}
	responders := make([]models.QuestionResponder, len(input.Responders))
	for i, identity := range input.Responders {
		responders[i] = models.QuestionResponder{Identity: identity, Status: models.QuestionResponderPending}
	}
	state := models.QuestionState{ResolutionOwner: input.ResolutionOwner, Responders: responders}
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("configure Question workflow %s: %w", question.Key, questionValidationError(err))
	}
	encoded, err := models.EncodeQuestionState(question.ContextData, state)
	if err != nil {
		return nil, fmt.Errorf("encode configured Question workflow %s: %w", question.Key, err)
	}
	if err := s.repo.ConfigureWorkflow(ctx, question.ID, question.Status, question.ContextData, encoded, state.ResolutionOwner); err != nil {
		return nil, fmt.Errorf("configure Question workflow %s: %w", question.Key, classifyQuestionTransitionError(err))
	}
	question.ContextData = encoded
	question.Status = models.QuestionStatusOpen
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, question.ID); err != nil {
		return nil, err
	}
	return question, nil
}

// RecordResponse records one response from exactly the currently routed
// responder. It never changes the claim: the parent loop releases that lease
// only after this successful Question-owned transaction has committed.
func (s *QuestionService) RecordResponse(ctx context.Context, input RecordQuestionResponseInput) (*models.Question, error) {
	if s.claimReader == nil {
		return nil, errors.New("record Question response: claim reader is required")
	}
	question, err := s.GetQuestion(ctx, input.Key)
	if err != nil {
		return nil, err
	}
	state, err := models.DecodeQuestionState(question.ContextData)
	if err != nil || state == nil {
		if err != nil {
			return nil, fmt.Errorf("record Question response %s: %w", question.Key, err)
		}
		return nil, fmt.Errorf("record Question response %s: Question workflow is not configured", question.Key)
	}
	if responseReplayMatches(*state, input) {
		return question, nil
	}
	if question.Status != models.QuestionStatusOpen && question.Status != models.QuestionStatusAnswering {
		return nil, fmt.Errorf("record Question response %s: %w", question.Key, questionConflictError(errors.New("Question must be open or answering")))
	}
	if input.SessionID == "" || input.Responder == "" {
		return nil, questionValidationError(errors.New("record Question response: session and responder are required"))
	}
	claim, err := s.claimReader.Get(ctx, string(models.EntityTypeQuestion), question.Key)
	if err != nil {
		return nil, fmt.Errorf("record Question response %s: load claim: %w", question.Key, err)
	}
	if claim == nil || claim.SessionID != input.SessionID || claim.ClaimedBy != input.Responder {
		return nil, fmt.Errorf("record Question response %s: %w", question.Key, questionConflictError(errors.New("active claim does not match responder session")))
	}
	if state.CurrentResponder() != input.Responder {
		return nil, fmt.Errorf("record Question response %s: %w", question.Key, questionConflictError(errors.New("responder is not current")))
	}
	response := models.QuestionResponse{SessionID: input.SessionID, Responder: input.Responder, Summary: input.Summary, EvidencePointer: input.EvidencePointer, RecordedAt: time.Now().UTC()}
	state.Responses = append(state.Responses, response)
	for i := range state.Responders {
		if state.Responders[i].Identity == input.Responder {
			state.Responders[i].Status = models.QuestionResponderCompleted
			break
		}
	}
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("record Question response %s: %w", question.Key, questionValidationError(err))
	}
	encoded, err := models.EncodeQuestionState(question.ContextData, *state)
	if err != nil {
		return nil, fmt.Errorf("record Question response %s: %w", question.Key, err)
	}
	nextStatus := models.QuestionStatusAnswering
	if state.CurrentResponder() == "" {
		nextStatus = models.QuestionStatusReadyForResolution
	}
	if err := s.repo.RecordResponse(ctx, question.ID, question.Status, nextStatus, question.ContextData, encoded, input.Responder); err != nil {
		return nil, fmt.Errorf("record Question response %s: %w", question.Key, classifyQuestionTransitionError(err))
	}
	question.ContextData, question.Status = encoded, nextStatus
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, question.ID); err != nil {
		return nil, err
	}
	return question, nil
}

// responseReplayMatches recognizes only an exact record already committed by
// the same lease session. It intentionally runs before the active-claim and
// current-responder checks because a successful parent release makes the
// completed responder ineligible for new work but must not turn a retry into
// a duplicate write or an error.
func responseReplayMatches(state models.QuestionState, input RecordQuestionResponseInput) bool {
	for _, response := range state.Responses {
		if response.SessionID == input.SessionID && response.Responder == input.Responder && response.Summary == input.Summary && response.EvidencePointer == input.EvidencePointer {
			return true
		}
	}
	return false
}

// Resolve records classified provenance and transitions only this Question to
// resolved. Destination validation completes before the repository begins its
// all-or-nothing state, note, history, and status transaction.
func (s *QuestionService) Resolve(ctx context.Context, input ResolveQuestionInput) (*models.Question, error) {
	question, state, err := s.loadClosableQuestion(ctx, input.Key, input.Owner)
	if err != nil {
		return nil, fmt.Errorf("resolve Question: %w", err)
	}
	if question.Status != models.QuestionStatusReadyForResolution || state.CurrentResponder() != "" {
		return nil, fmt.Errorf("resolve Question %s: %w", question.Key, questionConflictError(errors.New("Question must be ready for resolution with all responders completed")))
	}
	kind, pointer := strings.TrimSpace(input.Kind), strings.TrimSpace(input.Pointer)
	if kind != input.Kind || pointer != input.Pointer {
		return nil, questionValidationError(errors.New("resolve Question: resolution kind and pointer must be trimmed"))
	}
	state.ResolutionKind, state.ResolutionPointer = kind, pointer
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("resolve Question %s: %w", question.Key, questionValidationError(err))
	}
	if err := s.validateResolutionDestination(ctx, kind, pointer); err != nil {
		return nil, fmt.Errorf("resolve Question %s: validate destination: %w", question.Key, err)
	}
	encoded, err := models.EncodeQuestionState(question.ContextData, *state)
	if err != nil {
		return nil, fmt.Errorf("resolve Question %s: %w", question.Key, err)
	}
	if err := s.repo.Resolve(ctx, question.ID, question.Status, models.QuestionStatusResolved, question.ContextData, encoded, input.Owner, kind); err != nil {
		return nil, fmt.Errorf("resolve Question %s: %w", question.Key, classifyQuestionTransitionError(err))
	}
	question.ContextData, question.Status = encoded, models.QuestionStatusResolved
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, question.ID); err != nil {
		return nil, err
	}
	return question, nil
}

func (s *QuestionService) validateResolutionDestination(ctx context.Context, kind, pointer string) error {
	switch kind {
	case "no_lasting_consequence":
		return nil
	case "follow_up_work":
		found, err := s.repo.FollowUpWorkExists(ctx, pointer)
		if err != nil {
			return err
		}
		if !found {
			return questionValidationError(fmt.Errorf("follow-up work destination %q does not exist", pointer))
		}
		return nil
	case "local_clarification":
		if !strings.HasPrefix(pointer, "note:") {
			return questionValidationError(errors.New("local clarification pointer must reference note:<id>"))
		}
		found, err := s.repo.NoteExists(ctx, strings.TrimPrefix(pointer, "note:"))
		if err != nil {
			return err
		}
		if !found {
			return questionValidationError(fmt.Errorf("local clarification note %q does not exist", pointer))
		}
		return nil
	case "product_decision":
		if !strings.HasPrefix(pointer, "docs/product/progress.md#") || strings.TrimPrefix(pointer, "docs/product/progress.md#") == "" {
			return questionValidationError(errors.New("product decision pointer must be a docs/product/progress.md anchor"))
		}
		return s.validateResolutionDocument("docs/product/progress.md")
	}
	paths := strings.Split(pointer, ";")
	if kind == "architecture_decision" && len(paths) < 2 {
		return questionValidationError(errors.New("architecture decision pointer must include an ADR and affected reference"))
	}
	for _, path := range paths {
		if err := s.validateResolutionDocument(strings.TrimSpace(path)); err != nil {
			return err
		}
	}
	return nil
}

func (s *QuestionService) validateResolutionDocument(pointer string) error {
	if pointer == "" || filepath.IsAbs(pointer) || strings.HasPrefix(filepath.Clean(pointer), "..") {
		return questionValidationError(fmt.Errorf("invalid local document pointer %q", pointer))
	}
	root, err := filepath.Abs(s.projectRoot)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}
	path := filepath.Join(root, pointer)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return fmt.Errorf("resolve document path %q: %w", pointer, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return questionValidationError(fmt.Errorf("document pointer %q escapes project root", pointer))
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return questionValidationError(fmt.Errorf("document destination %q does not exist: %w", pointer, err))
		}
		return fmt.Errorf("stat document destination %q: %w", pointer, err)
	}
	return nil
}

// Withdraw preserves an owner-authored bounded reason as terminal provenance.
func (s *QuestionService) Withdraw(ctx context.Context, input WithdrawQuestionInput) (*models.Question, error) {
	return s.closeWithReason(ctx, input.Key, input.Owner, input.Reason, models.QuestionStatusWithdrawn, "")
}

// Supersede preserves an owner-authored reason and existing distinct Question
// pointer; it reads but never mutates the superseding Question.
func (s *QuestionService) Supersede(ctx context.Context, input SupersedeQuestionInput) (*models.Question, error) {
	if input.SupersededBy == input.Key {
		return nil, questionValidationError(errors.New("supersede Question: superseding Question must differ from target"))
	}
	if err := models.ValidateQuestionKey(input.SupersededBy); err != nil {
		return nil, fmt.Errorf("supersede Question: superseding Question: %w", questionValidationError(err))
	}
	if _, err := s.repo.GetByKey(ctx, input.SupersededBy); err != nil {
		return nil, fmt.Errorf("supersede Question: load superseding Question: %w", err)
	}
	return s.closeWithReason(ctx, input.Key, input.Owner, input.Reason, models.QuestionStatusSuperseded, input.SupersededBy)
}

func (s *QuestionService) closeWithReason(ctx context.Context, key, owner, reason string, status models.QuestionStatus, supersededBy string) (*models.Question, error) {
	question, _, err := s.loadClosableQuestion(ctx, key, owner)
	if err != nil {
		return nil, fmt.Errorf("%s Question: %w", status, err)
	}
	if isQuestionTerminal(question.Status) {
		return nil, fmt.Errorf("%s Question %s: %w", status, question.Key, questionConflictError(errors.New("Question is already terminal")))
	}
	if err := validateTerminalReason(reason); err != nil {
		return nil, fmt.Errorf("%s Question %s: %w", status, question.Key, questionValidationError(err))
	}
	encoded, err := encodeQuestionTerminalProvenance(question.ContextData, string(status), reason, supersededBy)
	if err != nil {
		return nil, fmt.Errorf("%s Question %s: %w", status, question.Key, err)
	}
	if err := s.repo.Withdraw(ctx, question.ID, question.Status, status, question.ContextData, encoded, owner, reason); err != nil {
		return nil, fmt.Errorf("%s Question %s: %w", status, question.Key, classifyQuestionTransitionError(err))
	}
	question.ContextData, question.Status = encoded, status
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeQuestion, question.ID); err != nil {
		return nil, err
	}
	return question, nil
}

func (s *QuestionService) loadClosableQuestion(ctx context.Context, key, owner string) (*models.Question, *models.QuestionState, error) {
	question, err := s.GetQuestion(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	state, err := models.DecodeQuestionState(question.ContextData)
	if err != nil || state == nil {
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, errors.New("Question workflow is not configured")
	}
	if owner != state.ResolutionOwner {
		return nil, nil, questionConflictError(errors.New("resolution owner does not match configured owner"))
	}
	return question, state, nil
}

func validateTerminalReason(reason string) error {
	return models.ValidateQuestionBoundedText("reason", reason, 1, 1000)
}

func isQuestionTerminal(status models.QuestionStatus) bool {
	return status == models.QuestionStatusResolved || status == models.QuestionStatusWithdrawn || status == models.QuestionStatusSuperseded || status == models.QuestionStatusArchived
}

func encodeQuestionTerminalProvenance(contextData *string, status, reason, supersededBy string) (*string, error) {
	fields := map[string]json.RawMessage{}
	if contextData != nil && strings.TrimSpace(*contextData) != "" {
		if err := json.Unmarshal([]byte(*contextData), &fields); err != nil {
			return nil, fmt.Errorf("decode Question context data: %w", err)
		}
	}
	provenance := struct {
		Status       string `json:"status"`
		Reason       string `json:"reason"`
		SupersededBy string `json:"superseded_by,omitempty"`
	}{status, reason, supersededBy}
	encodedProvenance, err := json.Marshal(provenance)
	if err != nil {
		return nil, fmt.Errorf("encode terminal Question provenance: %w", err)
	}
	fields["question_terminal_provenance"] = encodedProvenance
	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encode Question context data: %w", err)
	}
	value := string(encoded)
	return &value, nil
}
