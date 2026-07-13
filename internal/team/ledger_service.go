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
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	teamrunrepo "github.com/jwwelbor/shark-task-manager/internal/repository/teamrun"
)

// LedgerRepository is the persistence seam owned by the team domain. The
// concrete repository remains SQL-only; validation and idempotency stay here.
type LedgerRepository interface {
	FindRunByRoot(ctx context.Context, rootType, rootKey string) (*teamrunrepo.TeamRun, error)
	CreateRunWithItems(ctx context.Context, run *teamrunrepo.TeamRun, items []*teamrunrepo.TeamRunItem) error
	GetRun(ctx context.Context, runID int64) (*teamrunrepo.TeamRun, error)
	ListItems(ctx context.Context, runID int64) ([]*teamrunrepo.TeamRunItem, error)
	UpdateRun(ctx context.Context, run *teamrunrepo.TeamRun) error
	UpdateItem(ctx context.Context, item *teamrunrepo.TeamRunItem) error
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
	repo LedgerRepository
}

// NewLedger constructs a ledger service with an injected repository.
func NewLedger(repo LedgerRepository) *LedgerService { return &LedgerService{repo: repo} }

// NewLedgerService is the descriptive constructor alias used by service wiring.
func NewLedgerService(repo LedgerRepository) *LedgerService { return NewLedger(repo) }

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
	existing, err := l.repo.FindRunByRoot(ctx, string(plan.RootType), rootKey)
	if err == nil {
		if existing.PlanHash != plan.PlanHash {
			return nil, &PlanDriftError{RootKey: rootKey, ExistingHash: existing.PlanHash, RequestedHash: plan.PlanHash}
		}
		return toDomainRun(existing), nil
	}
	if !isNotFound(err) {
		return nil, fmt.Errorf("find confirmed team run for %s: %w", rootKey, err)
	}

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
	if err := l.repo.CreateRunWithItems(ctx, run, items); err != nil {
		return nil, fmt.Errorf("persist confirmed team plan for %s: %w", rootKey, err)
	}
	return toDomainRun(run), nil
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
	run.Status = string(update.Status)
	run.ExecutionMode = string(update.ExecutionMode)
	run.ConcurrencyLimit = update.ConcurrencyLimit
	run.PlanHash = update.PlanHash
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
	if err := update.Validate(); err != nil {
		return nil, fmt.Errorf("validate result for item %d: %w", update.ItemID, err)
	}
	items, err := l.repo.ListItems(ctx, update.RunID)
	if err != nil {
		return nil, fmt.Errorf("load item %d for result: %w", update.ItemID, err)
	}
	var current *teamrunrepo.TeamRunItem
	for _, item := range items {
		if item != nil && item.ID == update.ItemID {
			current = item
			break
		}
	}
	if current == nil {
		return nil, fmt.Errorf("load item %d for result: %w", update.ItemID, ErrRepositoryNotFound)
	}
	if current.TeamRunID != update.RunID {
		return nil, fmt.Errorf("record result for item %d: %w", update.ItemID, ErrRepositoryNotFound)
	}

	status := update.effectiveStatus()
	storedEvidence, err := encodeEvidence(update.Evidence, update.ArtifactRefs)
	if err != nil {
		return nil, fmt.Errorf("encode result evidence for item %d: %w", update.ItemID, err)
	}
	if terminalItemStatus(ItemStatus(current.ItemStatus)) && update.Attempt == current.Attempt {
		if sameTerminalResult(current, status, update, storedEvidence) {
			return toDomainItem(current), nil
		}
		return nil, &ConflictingTerminalResultError{RunID: update.RunID, ItemID: update.ItemID, Attempt: update.Attempt}
	}
	if update.Attempt != current.Attempt {
		if !update.ExplicitRetry || update.Attempt != current.Attempt+1 || !terminalItemStatus(ItemStatus(current.ItemStatus)) {
			return nil, fmt.Errorf("record result for item %d: %w", update.ItemID, ErrInvalidAttempt)
		}
	}

	current.ItemStatus = string(status)
	current.Attempt = update.Attempt
	current.Outcome = optionalString(update.Outcome)
	current.SkipReason = optionalString(update.SkipReason)
	current.Evidence = optionalString(storedEvidence)
	current.ClaimSessionID = optionalString(update.ClaimSessionID)
	current.WorkerSessionID = optionalString(update.WorkerSessionID)
	current.StartedAt = update.StartedAt
	completedAt := update.CompletedAt
	if completedAt == nil {
		now := time.Now().UTC()
		completedAt = &now
	}
	current.CompletedAt = completedAt
	if err := l.repo.UpdateItem(ctx, current); err != nil {
		return nil, fmt.Errorf("record result for item %d: %w", update.ItemID, err)
	}
	return toDomainItem(current), nil
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
	if err := validateEntityKey(plan.RootKey); err != nil {
		return fmt.Errorf("validate plan root key: %w", err)
	}
	if strings.TrimSpace(rootSessionID) == "" || len([]byte(rootSessionID)) > maxBoundedText {
		return fmt.Errorf("%w: root session is required", ErrInvalidPlanInput)
	}
	for _, item := range plan.Items {
		if err := validateEntityKey(item.ChildKey); err != nil {
			return err
		}
		if !models.ValidEntityTypes[item.ChildType] || item.Wave < 0 || item.ExecutionOrder < 0 {
			return fmt.Errorf("%w: invalid item %s", ErrInvalidPlanInput, item.ChildKey)
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

func toDomainItems(items []*teamrunrepo.TeamRunItem) ([]*TeamRunItem, error) {
	result := make([]*TeamRunItem, 0, len(items))
	for _, item := range items {
		converted := toDomainItem(item)
		if converted == nil {
			return nil, fmt.Errorf("%w: nil item", ErrInvalidPlanInput)
		}
		result = append(result, converted)
	}
	return result, nil
}

func toDomainItem(item *teamrunrepo.TeamRunItem) *TeamRunItem {
	if item == nil {
		return nil
	}
	refs, summary := decodeEvidence(stringValue(item.Evidence))
	deps := []string{}
	_ = json.Unmarshal([]byte(item.DependencyKeys), &deps)
	return &TeamRunItem{ID: item.ID, TeamRunID: item.TeamRunID, ChildKey: item.ChildKey, ChildType: models.EntityType(item.ChildType), Wave: item.Wave, ExecutionOrder: item.ExecutionOrder, DependencyKeys: deps, PlannedRole: item.PlannedRole, PlannedAction: item.PlannedAction, PlannedAgentType: item.PlannedAgentType, PlannedProvider: item.PlannedProvider, PlannedModel: item.PlannedModel, PlannedEffort: item.PlannedEffort, ItemStatus: ItemStatus(item.ItemStatus), ClaimSessionID: item.ClaimSessionID, WorkerSessionID: item.WorkerSessionID, Outcome: item.Outcome, SkipReason: item.SkipReason, Evidence: summary, ArtifactRefs: refs, Attempt: item.Attempt, StartedAt: item.StartedAt, CompletedAt: item.CompletedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func decodeEvidence(raw string) ([]string, string) {
	if raw == "" {
		return nil, ""
	}
	var stored storedEvidence
	if err := json.Unmarshal([]byte(raw), &stored); err == nil {
		return stored.ArtifactRefs, stored.Summary
	}
	return nil, raw
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
func isNotFound(err error) bool {
	return errors.Is(err, ErrRepositoryNotFound) || errors.Is(err, repoerr.ErrNotFound)
}
