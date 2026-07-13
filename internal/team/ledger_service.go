package team

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	teamrunrepo "github.com/jwwelbor/shark-task-manager/internal/repository/teamrun"
)

// LedgerRepository is the persistence seam owned by the team domain. The
// concrete repository remains SQL-only; validation and idempotency stay here.
type LedgerRepository interface {
	FindRunByRoot(ctx context.Context, rootType, rootKey string) (*teamrunrepo.TeamRun, error)
	CreateRunWithItems(ctx context.Context, run *teamrunrepo.TeamRun, items []*teamrunrepo.TeamRunItem) error
	CreateRunWithItemsIfAbsent(ctx context.Context, run *teamrunrepo.TeamRun, items []*teamrunrepo.TeamRunItem) (*teamrunrepo.TeamRun, bool, error)
	GetRun(ctx context.Context, runID int64) (*teamrunrepo.TeamRun, error)
	ListItems(ctx context.Context, runID int64) ([]*teamrunrepo.TeamRunItem, error)
	UpdateRun(ctx context.Context, run *teamrunrepo.TeamRun) error
	CompareAndSetItem(ctx context.Context, item *teamrunrepo.TeamRunItem, expectedStatus string, expectedAttempt int) (bool, error)
}

// Ledger is the consumer-facing service contract for durable team-run state.
type Ledger interface {
	PersistConfirmedPlan(ctx context.Context, plan *TeamPlan, rootSessionID string) (*TeamRun, error)
	GetRun(ctx context.Context, runID int64) (*TeamRun, error)
	ListItems(ctx context.Context, runID int64) ([]*TeamRunItem, error)
	UpdateRun(ctx context.Context, update RunUpdate) (*TeamRun, error)
	RecordItemResult(ctx context.Context, update ItemResultUpdate) (*TeamRunItem, error)
}

// LedgerService owns confirmed-plan and item-result semantics without
// scheduling or Shark entity lifecycle mutations.
type LedgerService struct {
	repo             LedgerRepository
	artifactBasePath string
}

// NewLedger constructs a ledger service with an injected repository.
func NewLedger(repo LedgerRepository, artifactBasePath ...string) *LedgerService {
	base := "."
	if len(artifactBasePath) > 0 && strings.TrimSpace(artifactBasePath[0]) != "" {
		base = artifactBasePath[0]
	}
	return &LedgerService{repo: repo, artifactBasePath: base}
}

// NewLedgerService is the descriptive constructor alias used by service wiring.
func NewLedgerService(repo LedgerRepository, artifactBasePath ...string) *LedgerService {
	return NewLedger(repo, artifactBasePath...)
}

var _ LedgerRepository = (*teamrunrepo.Repository)(nil)
var _ Ledger = (*LedgerService)(nil)

// PersistConfirmedPlan inserts a complete plan before any worker claim. A
// matching root/hash returns the existing run; a changed hash is drift.
func (l *LedgerService) PersistConfirmedPlan(ctx context.Context, plan *TeamPlan, rootSessionID string) (*TeamRun, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if l == nil || l.repo == nil {
		return nil, errors.New("team ledger repository is required")
	}
	if err := validateConfirmedPlan(plan, rootSessionID); err != nil {
		return nil, fmt.Errorf("validate confirmed team plan: %w", err)
	}

	rootKey := strings.ToUpper(strings.TrimSpace(plan.RootKey))
	session := rootSessionID
	run := &teamrunrepo.TeamRun{RootKey: rootKey, RootType: string(plan.RootType), Status: string(RunStatusPlanned), ExecutionMode: string(plan.ExecutionMode), ConcurrencyLimit: plan.ConcurrencyLimit, PlanHash: plan.PlanHash, RootSessionID: &session}
	items := make([]*teamrunrepo.TeamRunItem, 0, len(plan.Items))
	for _, planItem := range plan.Items {
		item, err := planItemToRepository(planItem)
		if err != nil {
			return nil, fmt.Errorf("convert planned item %s: %w", planItem.ChildKey, err)
		}
		items = append(items, item)
	}
	confirmed, _, err := l.repo.CreateRunWithItemsIfAbsent(ctx, run, items)
	if err != nil {
		return nil, fmt.Errorf("persist confirmed team plan for %s: %w", rootKey, err)
	}
	if confirmed.PlanHash != plan.PlanHash {
		return nil, &PlanDriftError{RootKey: rootKey, ExistingHash: confirmed.PlanHash, RequestedHash: plan.PlanHash}
	}
	if err := validateRepositoryRunIdentity(confirmed); err != nil {
		return nil, err
	}
	return toDomainRun(confirmed), nil
}

// GetRun retrieves a durable run by ID.
func (l *LedgerService) GetRun(ctx context.Context, runID int64) (*TeamRun, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if runID <= 0 {
		return nil, fmt.Errorf("get team run: %w", ErrInvalidPlanInput)
	}
	run, err := l.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get team run %d: %w", runID, err)
	}
	if err := validateRepositoryRunIdentity(run); err != nil {
		return nil, fmt.Errorf("validate team run %d: %w", runID, err)
	}
	return toDomainRun(run), nil
}

// ListItems retrieves a run's complete deterministic item list.
func (l *LedgerService) ListItems(ctx context.Context, runID int64) ([]*TeamRunItem, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if runID <= 0 {
		return nil, fmt.Errorf("list team run items: %w", ErrInvalidPlanInput)
	}
	items, err := l.repo.ListItems(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("list team run items for %d: %w", runID, err)
	}
	return toDomainItems(items)
}

// GetRunResult reads a run and all items into the shared I-01 result shape.
func (l *LedgerService) GetRunResult(ctx context.Context, runID int64) (*TeamRunResult, error) {
	run, err := l.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	items, err := l.ListItems(ctx, runID)
	if err != nil {
		return nil, err
	}
	return NewTeamRunResult(run, items)
}

// UpdateRun validates and persists coordinator-owned run lifecycle fields.
func (l *LedgerService) UpdateRun(ctx context.Context, update RunUpdate) (*TeamRun, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if update.RunID <= 0 || !validRunStatus(update.Status) || update.ConcurrencyLimit <= 0 || (update.ExecutionMode != ExecutionModeParallel && update.ExecutionMode != ExecutionModeSequential) || !sha256Hex.MatchString(update.PlanHash) {
		return nil, fmt.Errorf("update team run %d: %w", update.RunID, ErrInvalidPlanInput)
	}
	run, err := l.repo.GetRun(ctx, update.RunID)
	if err != nil {
		return nil, fmt.Errorf("load team run %d for update: %w", update.RunID, err)
	}
	if err := validateRepositoryRunIdentity(run); err != nil {
		return nil, fmt.Errorf("validate team run %d for update: %w", update.RunID, err)
	}
	if update.PlanHash != run.PlanHash {
		return nil, &PlanDriftError{RootKey: run.RootKey, ExistingHash: run.PlanHash, RequestedHash: update.PlanHash}
	}
	if update.ExecutionMode != ExecutionMode(run.ExecutionMode) || update.ConcurrencyLimit != run.ConcurrencyLimit {
		return nil, fmt.Errorf("update team run %d: %w", update.RunID, ErrImmutablePlanSnapshot)
	}
	currentStatus := RunStatus(run.Status)
	if !validRunTransition(currentStatus, update.Status) {
		return nil, fmt.Errorf("update team run %d from %s to %s: %w", update.RunID, currentStatus, update.Status, ErrInvalidRunTransition)
	}
	run.Status = string(update.Status)
	run.AggregateOutcome = update.AggregateOutcome
	run.NextAction = update.NextAction
	run.RootSessionID = update.RootSessionID
	run.StartedAt = update.StartedAt
	run.CompletedAt = update.CompletedAt
	if err := l.repo.UpdateRun(ctx, run); err != nil {
		return nil, fmt.Errorf("update team run %d: %w", update.RunID, err)
	}
	return toDomainRun(run), nil
}

// RecordItemResult validates a terminal result and applies it idempotently.
// ExplicitRetry is the only path that advances an item's attempt.
func (l *LedgerService) RecordItemResult(ctx context.Context, update ItemResultUpdate) (*TeamRunItem, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	validated, err := l.validateResult(update)
	if err != nil {
		return nil, err
	}
	current, err := l.loadResultItem(ctx, update.RunID, update.ItemID)
	if err != nil {
		return nil, err
	}
	decision, err := classifyResult(current, validated)
	if err != nil {
		return nil, err
	}
	if decision.idempotent {
		return toDomainItem(current)
	}
	return l.applyResult(ctx, current, validated)
}

type validatedResult struct {
	update         ItemResultUpdate
	status         ItemStatus
	storedEvidence string
}

func (l *LedgerService) validateResult(update ItemResultUpdate) (validatedResult, error) {
	if err := update.ValidateWithArtifactBase(l.artifactBasePath); err != nil {
		return validatedResult{}, fmt.Errorf("validate result for item %d: %w", update.ItemID, err)
	}
	normalizedRefs, err := update.NormalizeArtifactRefs(l.artifactBasePath)
	if err != nil {
		return validatedResult{}, fmt.Errorf("normalize result artifacts for item %d: %w", update.ItemID, err)
	}
	update.ArtifactRefs = normalizedRefs
	storedEvidence, err := encodeEvidence(update.Evidence, normalizedRefs)
	if err != nil {
		return validatedResult{}, fmt.Errorf("encode result evidence for item %d: %w", update.ItemID, err)
	}
	return validatedResult{update: update, status: update.effectiveStatus(), storedEvidence: storedEvidence}, nil
}

func (l *LedgerService) loadResultItem(ctx context.Context, runID, itemID int64) (*teamrunrepo.TeamRunItem, error) {
	items, err := l.repo.ListItems(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load item %d for result: %w", itemID, err)
	}
	for _, item := range items {
		if item != nil && item.ID == itemID {
			if item.TeamRunID != runID {
				break
			}
			if err := validateEntityIdentity(item.ChildKey, models.EntityType(item.ChildType)); err != nil {
				return nil, fmt.Errorf("validate result item %d identity: %w", itemID, err)
			}
			return item, nil
		}
	}
	return nil, fmt.Errorf("load item %d for result: %w", itemID, ErrRepositoryNotFound)
}

type resultDecision struct{ idempotent bool }

func classifyResult(current *teamrunrepo.TeamRunItem, result validatedResult) (resultDecision, error) {
	if terminalItemStatus(ItemStatus(current.ItemStatus)) && result.update.Attempt == current.Attempt {
		if sameTerminalResult(current, result.status, result.update, result.storedEvidence) {
			return resultDecision{idempotent: true}, nil
		}
		return resultDecision{}, &ConflictingTerminalResultError{RunID: result.update.RunID, ItemID: result.update.ItemID, Attempt: result.update.Attempt}
	}
	currentStatus := ItemStatus(current.ItemStatus)
	if result.update.ExplicitRetry && terminalItemStatus(currentStatus) {
		if result.update.Attempt != current.Attempt+1 {
			return resultDecision{}, fmt.Errorf("record result for item %d: %w", result.update.ItemID, ErrInvalidAttempt)
		}
		if err := validateItemOwnership(current, result.update); err != nil {
			return resultDecision{}, err
		}
		return resultDecision{}, nil
	}
	if !validItemTransition(currentStatus, result.status) {
		return resultDecision{}, fmt.Errorf("record result for item %d from %s to %s: %w", result.update.ItemID, currentStatus, result.status, ErrInvalidItemTransition)
	}
	if err := validateItemOwnership(current, result.update); err != nil {
		return resultDecision{}, err
	}
	if result.update.Attempt != current.Attempt {
		return resultDecision{}, fmt.Errorf("record result for item %d: %w", result.update.ItemID, ErrInvalidAttempt)
	}
	return resultDecision{}, nil
}

func validateItemOwnership(current *teamrunrepo.TeamRunItem, update ItemResultUpdate) error {
	claimSession := stringValue(current.ClaimSessionID)
	if claimSession == "" || update.ClaimSessionID == "" || update.ClaimSessionID != claimSession {
		return fmt.Errorf("record result for item %d: %w", update.ItemID, ErrInvalidItemOwnership)
	}
	workerSession := stringValue(current.WorkerSessionID)
	if ItemStatus(current.ItemStatus) == ItemStatusRunning && (workerSession == "" || update.WorkerSessionID == "" || update.WorkerSessionID != workerSession) {
		return fmt.Errorf("record result for item %d: %w", update.ItemID, ErrInvalidItemOwnership)
	}
	if workerSession != "" && update.WorkerSessionID != workerSession {
		return fmt.Errorf("record result for item %d: %w", update.ItemID, ErrInvalidItemOwnership)
	}
	return nil
}

func (l *LedgerService) applyResult(ctx context.Context, current *teamrunrepo.TeamRunItem, result validatedResult) (*TeamRunItem, error) {
	update := result.update
	expectedStatus, expectedAttempt := current.ItemStatus, current.Attempt
	current.ItemStatus, current.Attempt = string(result.status), update.Attempt
	current.Outcome, current.SkipReason = optionalString(update.Outcome), optionalString(update.SkipReason)
	current.Evidence = optionalString(result.storedEvidence)
	current.ClaimSessionID, current.WorkerSessionID = optionalString(update.ClaimSessionID), optionalString(update.WorkerSessionID)
	current.StartedAt = update.StartedAt
	current.CompletedAt = update.CompletedAt
	if current.CompletedAt == nil {
		now := time.Now().UTC()
		current.CompletedAt = &now
	}
	updated, err := l.repo.CompareAndSetItem(ctx, current, expectedStatus, expectedAttempt)
	if err != nil {
		return nil, fmt.Errorf("record result for item %d: %w", update.ItemID, err)
	}
	if updated {
		return toDomainItem(current)
	}
	latest, err := l.findItem(ctx, update.RunID, update.ItemID)
	if err != nil {
		return nil, fmt.Errorf("reload item %d after concurrent result: %w", update.ItemID, err)
	}
	if terminalItemStatus(ItemStatus(latest.ItemStatus)) && update.Attempt == latest.Attempt {
		if sameTerminalResult(latest, result.status, update, result.storedEvidence) {
			return toDomainItem(latest)
		}
		return nil, &ConflictingTerminalResultError{RunID: update.RunID, ItemID: update.ItemID, Attempt: update.Attempt}
	}
	return nil, fmt.Errorf("record result for item %d: %w", update.ItemID, ErrInvalidAttempt)
}

func (l *LedgerService) findItem(ctx context.Context, runID, itemID int64) (*teamrunrepo.TeamRunItem, error) {
	items, err := l.repo.ListItems(ctx, runID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item != nil && item.ID == itemID {
			if err := validateEntityIdentity(item.ChildKey, models.EntityType(item.ChildType)); err != nil {
				return nil, fmt.Errorf("validate item %d identity: %w", itemID, err)
			}
			return item, nil
		}
	}
	return nil, ErrRepositoryNotFound
}

// PlanDriftError identifies an incompatible repeated confirmation.
type PlanDriftError struct{ RootKey, ExistingHash, RequestedHash string }

func (e *PlanDriftError) Error() string {
	return fmt.Sprintf("team plan drift for %s: existing hash %s, requested hash %s", e.RootKey, e.ExistingHash, e.RequestedHash)
}
func (e *PlanDriftError) Unwrap() error { return ErrPlanDrift }

// ConflictingTerminalResultError identifies a different terminal result for
// an already recorded run/item/attempt tuple.
type ConflictingTerminalResultError struct {
	RunID, ItemID int64
	Attempt       int
}

func (e *ConflictingTerminalResultError) Error() string {
	return fmt.Sprintf("conflicting terminal result for run %d item %d attempt %d", e.RunID, e.ItemID, e.Attempt)
}
func (e *ConflictingTerminalResultError) Unwrap() error { return ErrConflictingTerminalResult }

type storedEvidence struct {
	Summary      string   `json:"summary,omitempty"`
	ArtifactRefs []string `json:"artifact_refs,omitempty"`
}

func validateConfirmedPlan(plan *TeamPlan, rootSessionID string) error {
	if err := plan.Validate(); err != nil {
		return err
	}
	if err := validateEntityIdentity(plan.RootKey, plan.RootType); err != nil {
		return fmt.Errorf("validate plan root identity: %w", err)
	}
	if strings.TrimSpace(rootSessionID) == "" || len([]byte(rootSessionID)) > maxBoundedText {
		return fmt.Errorf("%w: root session is required", ErrInvalidPlanInput)
	}
	for _, item := range plan.Items {
		if err := validateEntityIdentity(item.ChildKey, item.ChildType); err != nil {
			return fmt.Errorf("validate plan item identity: %w", err)
		}
		if item.Wave < 0 || item.ExecutionOrder < 0 {
			return fmt.Errorf("%w: invalid item %s", ErrInvalidPlanInput, item.ChildKey)
		}
		for _, edge := range item.Dependencies {
			if err := validateEntityIdentity(edge.ChildKey, edge.ChildType); err != nil {
				return fmt.Errorf("validate dependency child identity: %w", err)
			}
			if err := validateEntityIdentity(edge.DependencyKey, edge.DependencyType); err != nil {
				return fmt.Errorf("validate dependency target identity: %w", err)
			}
		}
	}
	return nil
}

func planItemToRepository(item TeamPlanItem) (*teamrunrepo.TeamRunItem, error) {
	deps := append([]string(nil), item.DependencyKeys...)
	sort.Strings(deps)
	dependencyJSON, err := json.Marshal(deps)
	if err != nil {
		return nil, err
	}
	return &teamrunrepo.TeamRunItem{ChildKey: item.ChildKey, ChildType: string(item.ChildType), Wave: item.Wave, ExecutionOrder: item.ExecutionOrder, DependencyKeys: string(dependencyJSON), PlannedRole: optionalString(item.Planned.AgentType), PlannedAction: optionalString(item.Planned.Action), PlannedAgentType: optionalString(item.Planned.AgentType), PlannedProvider: optionalString(item.Planned.Provider), PlannedModel: optionalString(item.Planned.Model), PlannedEffort: optionalString(item.Planned.Effort), ItemStatus: string(ItemStatusPlanned), Attempt: 0}, nil
}

func sameTerminalResult(item *teamrunrepo.TeamRunItem, status ItemStatus, update ItemResultUpdate, encodedEvidence string) bool {
	return item.ItemStatus == string(status) && stringValue(item.Outcome) == update.Outcome && stringValue(item.SkipReason) == update.SkipReason && stringValue(item.Evidence) == encodedEvidence && stringValue(item.ClaimSessionID) == update.ClaimSessionID && stringValue(item.WorkerSessionID) == update.WorkerSessionID
}

func encodeEvidence(summary string, refs []string) (string, error) {
	if summary == "" && len(refs) == 0 {
		return "", nil
	}
	ordered := append([]string(nil), refs...)
	sort.Strings(ordered)
	body, err := json.Marshal(storedEvidence{Summary: summary, ArtifactRefs: ordered})
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func toDomainRun(run *teamrunrepo.TeamRun) *TeamRun {
	if run == nil {
		return nil
	}
	return &TeamRun{ID: run.ID, RootKey: run.RootKey, RootType: models.EntityType(run.RootType), Status: RunStatus(run.Status), ExecutionMode: ExecutionMode(run.ExecutionMode), ConcurrencyLimit: run.ConcurrencyLimit, PlanHash: run.PlanHash, AggregateOutcome: run.AggregateOutcome, NextAction: run.NextAction, RootSessionID: run.RootSessionID, StartedAt: run.StartedAt, CompletedAt: run.CompletedAt, CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt}
}

func validateRepositoryRunIdentity(run *teamrunrepo.TeamRun) error {
	if run == nil {
		return fmt.Errorf("%w: nil run", ErrInvalidPlanInput)
	}
	if err := validateEntityIdentity(run.RootKey, models.EntityType(run.RootType)); err != nil {
		return fmt.Errorf("validate persisted root identity: %w", err)
	}
	return nil
}

func toDomainItems(items []*teamrunrepo.TeamRunItem) ([]*TeamRunItem, error) {
	result := make([]*TeamRunItem, 0, len(items))
	for _, item := range items {
		converted, err := toDomainItem(item)
		if err != nil {
			return nil, fmt.Errorf("convert team-run item: %w", err)
		}
		result = append(result, converted)
	}
	return result, nil
}

func toDomainItem(item *teamrunrepo.TeamRunItem) (*TeamRunItem, error) {
	if item == nil {
		return nil, fmt.Errorf("%w: nil item", ErrInvalidPlanInput)
	}
	if err := validateEntityIdentity(item.ChildKey, models.EntityType(item.ChildType)); err != nil {
		return nil, fmt.Errorf("validate item %d identity: %w", item.ID, err)
	}
	refs, summary, err := decodeEvidence(stringValue(item.Evidence))
	if err != nil {
		return nil, fmt.Errorf("%w: decode item %d evidence: %v", ErrInvalidEvidence, item.ID, err)
	}
	deps := []string{}
	if err := json.Unmarshal([]byte(item.DependencyKeys), &deps); err != nil {
		return nil, fmt.Errorf("%w: decode item %d dependencies: %v", ErrMalformedDependency, item.ID, err)
	}
	return &TeamRunItem{ID: item.ID, TeamRunID: item.TeamRunID, ChildKey: item.ChildKey, ChildType: models.EntityType(item.ChildType), Wave: item.Wave, ExecutionOrder: item.ExecutionOrder, DependencyKeys: deps, PlannedRole: item.PlannedRole, PlannedAction: item.PlannedAction, PlannedAgentType: item.PlannedAgentType, PlannedProvider: item.PlannedProvider, PlannedModel: item.PlannedModel, PlannedEffort: item.PlannedEffort, ItemStatus: ItemStatus(item.ItemStatus), ClaimSessionID: item.ClaimSessionID, WorkerSessionID: item.WorkerSessionID, Outcome: item.Outcome, SkipReason: item.SkipReason, Evidence: summary, ArtifactRefs: refs, Attempt: item.Attempt, StartedAt: item.StartedAt, CompletedAt: item.CompletedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}

func decodeEvidence(raw string) ([]string, string, error) {
	if raw == "" {
		return nil, "", nil
	}
	var stored storedEvidence
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil, "", err
	}
	return stored.ArtifactRefs, stored.Summary, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	copy := value
	return &copy
}
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("team ledger context is nil")
	}
	return ctx.Err()
}
