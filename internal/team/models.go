// Package team contains the read-only planning domain for multi-child Shark
// execution. It intentionally has no database, Cobra, claim mutation, or
// worker-dispatch dependency.
package team

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

var (
	ErrInvalidPlanInput          = errors.New("invalid team plan input")
	ErrDependencyCycle           = errors.New("team plan dependency cycle")
	ErrMissingDependency         = errors.New("team plan missing dependency")
	ErrMalformedDependency       = errors.New("team plan malformed dependency")
	ErrUnresolvedWorkflow        = errors.New("team plan unresolved workflow")
	ErrCapabilityUnavailable     = errors.New("team plan capability unavailable")
	ErrDuplicateChild            = errors.New("team plan duplicate child")
	ErrRepositoryNotFound        = errors.New("team ledger record not found")
	ErrPlanDrift                 = errors.New("confirmed team plan drifted")
	ErrConflictingTerminalResult = errors.New("conflicting terminal item result")
	ErrInvalidRunStatus          = errors.New("invalid team run status")
	ErrInvalidItemStatus         = errors.New("invalid team run item status")
	ErrInvalidRunTransition      = errors.New("invalid team run status transition")
	ErrInvalidItemTransition     = errors.New("invalid team run item status transition")
	ErrImmutablePlanSnapshot     = errors.New("confirmed team plan snapshot is immutable")
	ErrInvalidItemOwnership      = errors.New("team run item result is not owned by the submitting session")
	ErrInvalidAttempt            = errors.New("invalid team run attempt")
	ErrInvalidEvidence           = errors.New("invalid team result evidence")
	ErrEvidenceTooLarge          = errors.New("team result evidence exceeds bound")
	ErrSensitiveEvidence         = errors.New("team result evidence contains sensitive content")
	ErrInvalidArtifactPath       = errors.New("invalid team result artifact path")
	ErrInvalidEntityKey          = errors.New("invalid team ledger entity key")
)

// PlanValidationError identifies the root and offending child/reference that
// made a plan invalid while retaining a typed cause for errors.Is.
type PlanValidationError struct {
	Cause     error
	RootKey   string
	ChildKey  string
	Reference string
	Detail    string
}

func (e *PlanValidationError) Error() string {
	parts := []string{"team plan validation"}
	if e.RootKey != "" {
		parts = append(parts, "root="+e.RootKey)
	}
	if e.ChildKey != "" {
		parts = append(parts, "child="+e.ChildKey)
	}
	if e.Reference != "" {
		parts = append(parts, "reference="+e.Reference)
	}
	if e.Detail != "" {
		parts = append(parts, e.Detail)
	}
	if e.Cause != nil {
		return strings.Join(parts, " ") + ": " + e.Cause.Error()
	}
	return strings.Join(parts, " ")
}

func (e *PlanValidationError) Unwrap() error { return e.Cause }

// CapabilityError is returned before any persistence or mutation when the
// host cannot safely execute even one required worker.
type CapabilityError struct {
	RootKey string
	Reason  string
}

func (e *CapabilityError) Error() string {
	return fmt.Sprintf("team plan capability unavailable for root %s: %s", e.RootKey, e.Reason)
}

func (e *CapabilityError) Unwrap() error { return ErrCapabilityUnavailable }

type ExecutionMode string

const (
	ExecutionModeParallel   ExecutionMode = "parallel"
	ExecutionModeSequential ExecutionMode = "sequential"
)

const (
	ExclusionTerminal             = "terminal"
	ExclusionClaimed              = "claimed"
	ExclusionHumanGate            = "human_gate"
	ExclusionPause                = "pause"
	ExclusionDependencyIneligible = "dependency_ineligible"
	ExclusionCapability           = "capability_excluded"
)

const (
	DegradedReasonParallelUnavailable      = "parallel_unavailable"
	DegradedReasonUnknownResourceOwnership = "unknown_resource_ownership"
	DegradedReasonOverlappingOwnership     = "overlapping_resource_ownership"
)

// ChildIdentity is the typed canonical identity used at every planner seam.
type ChildIdentity struct {
	Key        string            `json:"key"`
	EntityType models.EntityType `json:"entity_type"`
}

// ChildSnapshot is the complete direct-child read model supplied by the
// cascade/repository adapter. It is not limited to dispatchable children.
type ChildSnapshot struct {
	Key            string            `json:"key"`
	EntityType     models.EntityType `json:"entity_type"`
	Status         string            `json:"status"`
	ExecutionOrder int               `json:"execution_order"`
	Priority       int               `json:"priority,omitempty"`
	// LegacyDependencies is retained as compatibility input for adapters that
	// read the old tasks.depends_on JSON column directly.
	LegacyDependencies string `json:"-"`
}

// DependencyEdge is a normalized prerequisite edge. ChildKey depends on
// DependencyKey. External edges refer to an entity outside the root's direct
// child set and are retained in the plan.
type DependencyEdge struct {
	ChildKey         string            `json:"child_key"`
	ChildType        models.EntityType `json:"child_type"`
	DependencyKey    string            `json:"dependency_key"`
	DependencyType   models.EntityType `json:"dependency_type"`
	DependencyStatus string            `json:"dependency_status,omitempty"`
	External         bool              `json:"external,omitempty"`
	Satisfied        bool              `json:"satisfied"`
	// Resolved means the source successfully looked up the prerequisite entity.
	// The planner uses this to classify the edge against the root roster: a
	// resolved target outside the roster is an external prerequisite.
	Resolved bool   `json:"resolved,omitempty"`
	Source   string `json:"source,omitempty"`
}

// DispatchMetadata is the durable subset of a resolved dispatch step. Prompt,
// vars, and worker output are intentionally absent.
type DispatchMetadata struct {
	Action                 string                      `json:"action,omitempty"`
	AgentType              string                      `json:"agent_type,omitempty"`
	Provider               string                      `json:"provider,omitempty"`
	Model                  string                      `json:"model,omitempty"`
	Effort                 string                      `json:"effort,omitempty"`
	GateClassification     dispatch.GateClassification `json:"gate_classification,omitempty"`
	UnresolvedPlaceholders []string                    `json:"unresolved_placeholders,omitempty"`
}

func metadataFromStep(step dispatch.DispatchStep) DispatchMetadata {
	placeholders := append([]string(nil), step.UnresolvedPlaceholders...)
	sort.Strings(placeholders)
	return DispatchMetadata{
		Action:                 step.Action,
		AgentType:              step.AgentType,
		Provider:               step.Provider,
		Model:                  step.Model,
		Effort:                 step.Effort,
		GateClassification:     step.GateClassification,
		UnresolvedPlaceholders: placeholders,
	}
}

type ClaimDiagnostic struct {
	Claimed        bool   `json:"claimed"`
	ClaimSessionID string `json:"claim_session_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// CapabilityFacts describe host facts available at plan time. The aliases
// keep the input compatible with both the host precondition vocabulary and
// the domain vocabulary used by later scheduler code.
type CapabilityFacts struct {
	TeamExecutionAvailable       bool
	SingleWorkerAvailable        bool
	WorktreeIsolationAvailable   bool
	ResourceOwnershipKnown       bool
	ResourceOwnershipOverlap     bool
	MaxConcurrency               int
	SafeTeamExecution            bool
	SafeSingleWorkerExecution    bool
	WorktreeIsolation            bool
	UnknownResourceOwnership     bool
	OverlappingResourceOwnership bool
	SafeWorkerExecutionAvailable bool
	MaxParallelism               int
}

type PlanInput struct {
	RootType             models.EntityType
	RootKey              string
	RequestedConcurrency int
	Capabilities         CapabilityFacts
}

type TeamPlanItem struct {
	ChildKey        string            `json:"child_key"`
	ChildType       models.EntityType `json:"child_type"`
	Status          string            `json:"status"`
	ExecutionOrder  int               `json:"execution_order"`
	Priority        int               `json:"priority,omitempty"`
	Wave            int               `json:"wave"`
	DependencyKeys  []string          `json:"dependency_keys,omitempty"`
	Dependencies    []DependencyEdge  `json:"dependencies,omitempty"`
	Planned         DispatchMetadata  `json:"planned"`
	Claim           ClaimDiagnostic   `json:"claim,omitempty"`
	Eligible        bool              `json:"eligible"`
	ExclusionReason string            `json:"exclusion_reason,omitempty"`
}

type TeamPlan struct {
	RootKey              string            `json:"root_key"`
	RootType             models.EntityType `json:"root_type"`
	Items                []TeamPlanItem    `json:"items"`
	ExecutionMode        ExecutionMode     `json:"execution_mode"`
	ConcurrencyLimit     int               `json:"concurrency_limit"`
	DegradedReason       string            `json:"degraded_reason,omitempty"`
	CapabilityExclusions []string          `json:"capability_exclusions,omitempty"`
	PlanHash             string            `json:"plan_hash"`
}

// RunStatus is the lifecycle of a confirmed team run.
type RunStatus string

const (
	RunStatusPlanned   RunStatus = "planned"
	RunStatusRunning   RunStatus = "running"
	RunStatusPaused    RunStatus = "paused"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCompleted RunStatus = "completed"
	RunStatusCancelled RunStatus = "cancelled"
)

// ItemStatus is the lifecycle of a child membership in a team run.
type ItemStatus string

const (
	ItemStatusPlanned   ItemStatus = "planned"
	ItemStatusClaimed   ItemStatus = "claimed"
	ItemStatusRunning   ItemStatus = "running"
	ItemStatusCompleted ItemStatus = "completed"
	ItemStatusFailed    ItemStatus = "failed"
	ItemStatusBlocked   ItemStatus = "blocked"
	ItemStatusPaused    ItemStatus = "paused"
	ItemStatusSkipped   ItemStatus = "skipped"
	ItemStatusCancelled ItemStatus = "cancelled"
)

const (
	// MaxEvidenceBytes bounds the UTF-8 encoded evidence summary stored in the ledger.
	MaxEvidenceBytes = 4096
	maxArtifactRefs  = 32
	maxArtifactPath  = 512
	maxBoundedText   = 1024
)

var sensitiveEvidencePattern = regexp.MustCompile(`(?is)(?:bearer\s+[a-z0-9._-]+|\bsk-[a-z0-9_-]+\b|\bAKIA[0-9A-Z]{16}\b|-----BEGIN [^-\r\n]*PRIVATE KEY-----|rendered\s+prompt\s*:|system\s+prompt\s*:)`)

// TeamRun is the domain representation of a durable team run.
type TeamRun struct {
	ID               int64             `json:"id"`
	RootKey          string            `json:"root_key"`
	RootType         models.EntityType `json:"root_type"`
	Status           RunStatus         `json:"status"`
	ExecutionMode    ExecutionMode     `json:"execution_mode"`
	ConcurrencyLimit int               `json:"concurrency_limit"`
	PlanHash         string            `json:"plan_hash"`
	AggregateOutcome *string           `json:"aggregate_outcome,omitempty"`
	NextAction       *string           `json:"next_action,omitempty"`
	RootSessionID    *string           `json:"root_session_id,omitempty"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// TeamRunItem is the domain representation of a persisted child membership.
type TeamRunItem struct {
	ID               int64             `json:"id"`
	TeamRunID        int64             `json:"team_run_id"`
	ChildKey         string            `json:"child_key"`
	ChildType        models.EntityType `json:"child_type"`
	Wave             int               `json:"wave"`
	ExecutionOrder   int               `json:"execution_order"`
	DependencyKeys   []string          `json:"dependency_keys,omitempty"`
	PlannedRole      *string           `json:"planned_role,omitempty"`
	PlannedAction    *string           `json:"planned_action,omitempty"`
	PlannedAgentType *string           `json:"planned_agent_type,omitempty"`
	PlannedProvider  *string           `json:"planned_provider,omitempty"`
	PlannedModel     *string           `json:"planned_model,omitempty"`
	PlannedEffort    *string           `json:"planned_effort,omitempty"`
	ItemStatus       ItemStatus        `json:"item_status"`
	ClaimSessionID   *string           `json:"claim_session_id,omitempty"`
	WorkerSessionID  *string           `json:"worker_session_id,omitempty"`
	Outcome          *string           `json:"outcome,omitempty"`
	SkipReason       *string           `json:"skip_reason,omitempty"`
	Evidence         string            `json:"evidence,omitempty"`
	ArtifactRefs     []string          `json:"artifact_refs,omitempty"`
	Attempt          int               `json:"attempt"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// ItemResultUpdate is the validated, bounded result accepted from a worker.
type ItemResultUpdate struct {
	RunID           int64      `json:"run_id"`
	ItemID          int64      `json:"item_id"`
	Attempt         int        `json:"attempt"`
	Status          ItemStatus `json:"status"`
	ItemStatus      ItemStatus `json:"item_status,omitempty"`
	Outcome         string     `json:"outcome,omitempty"`
	SkipReason      string     `json:"skip_reason,omitempty"`
	Evidence        string     `json:"evidence,omitempty"`
	ArtifactRefs    []string   `json:"artifact_refs,omitempty"`
	ClaimSessionID  string     `json:"claim_session_id,omitempty"`
	WorkerSessionID string     `json:"worker_session_id,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ExplicitRetry   bool       `json:"explicit_retry,omitempty"`
}

// RunUpdate contains mutable run fields supplied by the coordinator.
type RunUpdate struct {
	RunID            int64
	Status           RunStatus
	ExecutionMode    ExecutionMode
	ConcurrencyLimit int
	PlanHash         string
	AggregateOutcome *string
	NextAction       *string
	RootSessionID    *string
	StartedAt        *time.Time
	CompletedAt      *time.Time
}

// TeamRunCounts is the operator-facing aggregate for a complete item
// snapshot. ByStatus is keyed by the durable item-status string so the JSON
// contract remains stable for configured consumers.
type TeamRunCounts struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
}

// TeamRunResult is the stable I-01 read shape shared with later E38 features.
type TeamRunResult struct {
	RunID            int64             `json:"run_id"`
	RootKey          string            `json:"root_key"`
	RootType         models.EntityType `json:"root_type"`
	Status           RunStatus         `json:"status"`
	ExecutionMode    ExecutionMode     `json:"execution_mode"`
	ConcurrencyLimit int               `json:"concurrency_limit"`
	PlanHash         string            `json:"plan_hash"`
	AggregateOutcome *string           `json:"aggregate_outcome,omitempty"`
	NextAction       *string           `json:"next_action,omitempty"`
	Counts           TeamRunCounts     `json:"counts"`
	Items            []*TeamRunItem    `json:"items"`
}

// NewTeamRunResult converts a persisted run and its complete item list into
// the shared result shape without introducing prompt or worker-output fields.
func NewTeamRunResult(run *TeamRun, items []*TeamRunItem) (*TeamRunResult, error) {
	if err := run.Validate(); err != nil {
		return nil, err
	}
	result := &TeamRunResult{RunID: run.ID, RootKey: run.RootKey, RootType: run.RootType, Status: run.Status, ExecutionMode: run.ExecutionMode, ConcurrencyLimit: run.ConcurrencyLimit, PlanHash: run.PlanHash, AggregateOutcome: run.AggregateOutcome, NextAction: run.NextAction, Counts: newTeamRunCounts(), Items: make([]*TeamRunItem, 0, len(items))}
	for _, item := range items {
		if err := item.Validate(); err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
		result.Counts.Total++
		result.Counts.ByStatus[string(item.ItemStatus)]++
	}
	return result, nil
}

func newTeamRunCounts() TeamRunCounts {
	counts := TeamRunCounts{ByStatus: make(map[string]int, 9)}
	for _, status := range []ItemStatus{
		ItemStatusPlanned, ItemStatusClaimed, ItemStatusRunning,
		ItemStatusCompleted, ItemStatusFailed, ItemStatusBlocked,
		ItemStatusPaused, ItemStatusSkipped, ItemStatusCancelled,
	} {
		counts.ByStatus[string(status)] = 0
	}
	return counts
}

// Validate checks the persisted run invariants before repository access.
func (r *TeamRun) Validate() error {
	if r == nil {
		return fmt.Errorf("%w: run is nil", ErrInvalidPlanInput)
	}
	if err := validateRunIdentity(r); err != nil {
		return err
	}
	if !validRunStatus(r.Status) {
		return fmt.Errorf("%w: %q", ErrInvalidRunStatus, r.Status)
	}
	if !validExecutionMode(r.ExecutionMode) {
		return fmt.Errorf("%w: %q", ErrInvalidPlanInput, r.ExecutionMode)
	}
	if r.ConcurrencyLimit <= 0 || !sha256Hex.MatchString(r.PlanHash) {
		return fmt.Errorf("%w: run limit and plan hash are invalid", ErrInvalidPlanInput)
	}
	return nil
}

func validateRunIdentity(r *TeamRun) error {
	if r.RootType != models.EntityTypeEpic && r.RootType != models.EntityTypeFeature {
		return fmt.Errorf("%w: %q", ErrInvalidPlanInput, r.RootType)
	}
	if err := validateEntityIdentity(r.RootKey, r.RootType); err != nil {
		return fmt.Errorf("validate run root identity: %w", err)
	}
	return nil
}

// Validate checks item identity, lifecycle, dependency, and attempt bounds.
func (i *TeamRunItem) Validate() error {
	if i == nil {
		return fmt.Errorf("%w: item is nil", ErrInvalidPlanInput)
	}
	if err := validateItemIdentity(i); err != nil {
		return err
	}
	if !validItemStatus(i.ItemStatus) {
		return fmt.Errorf("%w: %q", ErrInvalidItemStatus, i.ItemStatus)
	}
	return validateStoredEvidence(i)
}

func validateItemIdentity(i *TeamRunItem) error {
	if err := validateEntityIdentity(i.ChildKey, i.ChildType); err != nil {
		return fmt.Errorf("validate item identity: %w", err)
	}
	if i.Wave < 0 || i.ExecutionOrder < 0 || i.Attempt < 0 {
		return fmt.Errorf("%w: item identity, wave, order, or attempt is invalid", ErrInvalidPlanInput)
	}
	return nil
}

func validateStoredEvidence(i *TeamRunItem) error {
	if len([]byte(i.Evidence)) > MaxEvidenceBytes || sensitiveEvidencePattern.MatchString(i.Evidence) {
		return fmt.Errorf("%w: item evidence is not safe", ErrInvalidEvidence)
	}
	for _, ref := range i.ArtifactRefs {
		if err := validateArtifactRef(ref); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks worker result boundaries before any persistence call.
func (u ItemResultUpdate) Validate() error {
	return u.ValidateWithArtifactBase(".")
}

// ValidateWithArtifactBase validates a worker result against the configured
// project root before any repository access. Artifact references are stored as
// canonical project-relative paths, never as caller-provided path strings.
func (u ItemResultUpdate) ValidateWithArtifactBase(artifactBase string) error {
	if err := validateResultIdentity(u); err != nil {
		return err
	}
	if err := validateResultContent(u); err != nil {
		return err
	}
	normalized, err := u.NormalizeArtifactRefs(artifactBase)
	if err != nil {
		return err
	}
	return rejectDuplicateArtifactRefs(normalized)
}

func validateResultIdentity(u ItemResultUpdate) error {
	if u.RunID <= 0 || u.ItemID <= 0 {
		return fmt.Errorf("%w: run and item IDs must be positive", ErrInvalidPlanInput)
	}
	if u.Attempt < 0 {
		return ErrInvalidAttempt
	}
	if u.Status != "" && u.ItemStatus != "" && u.Status != u.ItemStatus {
		return fmt.Errorf("%w: conflicting result status fields", ErrInvalidItemStatus)
	}
	status := u.effectiveStatus()
	if !validItemStatus(status) || !terminalItemStatus(status) {
		return fmt.Errorf("%w: result status %q", ErrInvalidItemStatus, status)
	}
	return nil
}

func validateResultContent(u ItemResultUpdate) error {
	if len([]byte(u.Evidence)) > MaxEvidenceBytes {
		return ErrEvidenceTooLarge
	}
	if sensitiveEvidencePattern.MatchString(u.Evidence) || sensitiveEvidencePattern.MatchString(u.Outcome) || sensitiveEvidencePattern.MatchString(u.SkipReason) {
		return ErrSensitiveEvidence
	}
	if len([]byte(u.Outcome)) > maxBoundedText || len([]byte(u.SkipReason)) > maxBoundedText {
		return ErrInvalidEvidence
	}
	if len(u.ArtifactRefs) > maxArtifactRefs {
		return fmt.Errorf("%w: too many artifact references", ErrInvalidArtifactPath)
	}
	return nil
}

func rejectDuplicateArtifactRefs(refs []string) error {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if _, exists := seen[ref]; exists {
			return fmt.Errorf("%w: duplicate artifact reference %q", ErrInvalidArtifactPath, ref)
		}
		seen[ref] = struct{}{}
	}
	return nil
}

// NormalizeArtifactRefs validates and canonicalizes artifact references under
// an allowed project base. It is intentionally separate so the service can
// persist exactly the validated representation.
func (u ItemResultUpdate) NormalizeArtifactRefs(artifactBase string) ([]string, error) {
	base, err := filepath.Abs(strings.TrimSpace(artifactBase))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid artifact base %q", ErrInvalidArtifactPath, artifactBase)
	}
	base = filepath.Clean(base)
	refs := make([]string, 0, len(u.ArtifactRefs))
	for _, ref := range u.ArtifactRefs {
		normalized, err := normalizeArtifactRef(base, ref)
		if err != nil {
			return nil, err
		}
		refs = append(refs, normalized)
	}
	return refs, nil
}

func validateArtifactRef(ref string) error {
	_, err := normalizeArtifactRef(".", ref)
	return err
}

func (u ItemResultUpdate) effectiveStatus() ItemStatus {
	if u.ItemStatus != "" {
		return u.ItemStatus
	}
	return u.Status
}

func validateEntityKey(key string) error {
	if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n\t") || len([]byte(key)) > maxBoundedText {
		return ErrInvalidEntityKey
	}
	if keys.NewKeyService().Parse(key).EntityType == keys.EntityTypeUnknown {
		return ErrInvalidEntityKey
	}
	return nil
}

// validateEntityIdentity verifies that a key's syntax agrees with its
// declared entity type. Keys are case-insensitive and may include slugs, but
// a valid key for one entity type must never cross a typed planner or ledger
// boundary as another type.
func validateEntityIdentity(key string, declaredType models.EntityType) error {
	if !models.ValidEntityTypes[declaredType] {
		return fmt.Errorf("%w: unsupported declared entity type %q", ErrInvalidEntityKey, declaredType)
	}
	if err := validateEntityKey(key); err != nil {
		return fmt.Errorf("%w: key %q", err, key)
	}
	parsed := keys.NewKeyService().Parse(strings.TrimSpace(key))
	actualType := models.EntityType(parsed.EntityType)
	if actualType != declaredType {
		return fmt.Errorf("%w: key %q identifies %s, declared %s", ErrInvalidEntityKey, key, actualType, declaredType)
	}
	return nil
}

func normalizeArtifactRef(base, ref string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if err := rejectArtifactLexicalPath(trimmed, ref); err != nil {
		return "", err
	}
	candidate := filepath.Clean(filepath.Join(base, filepath.FromSlash(trimmed)))
	relative, err := filepath.Rel(base, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q", ErrInvalidArtifactPath, ref)
	}
	return filepath.ToSlash(relative), nil
}

func rejectArtifactLexicalPath(trimmed, original string) error {
	if artifactPathHasForbiddenShape(trimmed) || filepath.Clean(trimmed) == "." || strings.Contains(trimmed, "\x00") {
		return fmt.Errorf("%w: %q", ErrInvalidArtifactPath, original)
	}
	if strings.Contains(trimmed, "..") && artifactPathHasParentSegment(trimmed) {
		return fmt.Errorf("%w: %q", ErrInvalidArtifactPath, original)
	}
	return nil
}

func artifactPathHasForbiddenShape(path string) bool {
	return path == "" || len([]byte(path)) > maxArtifactPath || filepath.IsAbs(path) || strings.HasPrefix(path, "\\") || strings.Contains(path, "\\") || strings.HasPrefix(path, "~") || isWindowsAbsolute(path)
}

func artifactPathHasParentSegment(path string) bool {
	for _, part := range strings.Split(path, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func isWindowsAbsolute(path string) bool {
	return len(path) >= 3 && ((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) && path[1] == ':' && path[2] == '/'
}

func validRunStatus(status RunStatus) bool {
	switch status {
	case RunStatusPlanned, RunStatusRunning, RunStatusPaused, RunStatusFailed, RunStatusCompleted, RunStatusCancelled:
		return true
	}
	return false
}

// runTransitions is the only allowed lifecycle graph for a confirmed run.
// Repeating the current status is handled as an idempotent update by the
// service; it is intentionally not represented as a state change here.
var runTransitions = map[RunStatus]map[RunStatus]struct{}{
	RunStatusPlanned: {RunStatusRunning: {}, RunStatusPaused: {}},
	RunStatusRunning: {RunStatusPaused: {}, RunStatusFailed: {}, RunStatusCompleted: {}, RunStatusCancelled: {}},
	RunStatusPaused:  {RunStatusRunning: {}, RunStatusFailed: {}, RunStatusCancelled: {}},
	RunStatusFailed:  {RunStatusRunning: {}, RunStatusCancelled: {}},
}

func validItemStatus(status ItemStatus) bool {
	switch status {
	case ItemStatusPlanned, ItemStatusClaimed, ItemStatusRunning, ItemStatusCompleted, ItemStatusFailed, ItemStatusBlocked, ItemStatusPaused, ItemStatusSkipped, ItemStatusCancelled:
		return true
	}
	return false
}

// itemTransitions separates claim/worker lifecycle from terminal result
// recording. In particular, planned items cannot be completed by a result
// submission that bypasses claim ownership.
var itemTransitions = map[ItemStatus]map[ItemStatus]struct{}{
	ItemStatusPlanned: {ItemStatusClaimed: {}},
	ItemStatusClaimed: {
		ItemStatusRunning: {}, ItemStatusCompleted: {}, ItemStatusFailed: {}, ItemStatusBlocked: {},
		ItemStatusPaused: {}, ItemStatusSkipped: {}, ItemStatusCancelled: {},
	},
	ItemStatusRunning: {
		ItemStatusCompleted: {}, ItemStatusFailed: {}, ItemStatusBlocked: {}, ItemStatusPaused: {},
		ItemStatusSkipped: {}, ItemStatusCancelled: {},
	},
}

func validRunTransition(from, to RunStatus) bool {
	if from == to {
		return true
	}
	_, ok := runTransitions[from][to]
	return ok
}

func validItemTransition(from, to ItemStatus) bool {
	_, ok := itemTransitions[from][to]
	return ok
}

func terminalItemStatus(status ItemStatus) bool {
	switch status {
	case ItemStatusCompleted, ItemStatusFailed, ItemStatusBlocked, ItemStatusPaused, ItemStatusSkipped, ItemStatusCancelled:
		return true
	}
	return false
}

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (p *TeamPlan) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: plan is nil", ErrInvalidPlanInput)
	}
	if err := validatePlanIdentity(p); err != nil {
		return err
	}
	if p.ConcurrencyLimit <= 0 {
		return fmt.Errorf("%w: root key and positive concurrency limit are required", ErrInvalidPlanInput)
	}
	if !validExecutionMode(p.ExecutionMode) {
		return fmt.Errorf("%w: invalid execution mode %q", ErrInvalidPlanInput, p.ExecutionMode)
	}
	if !sha256Hex.MatchString(p.PlanHash) {
		return fmt.Errorf("%w: plan hash must be lowercase SHA-256", ErrInvalidPlanInput)
	}
	return validatePlanItems(p.Items)
}

func validatePlanIdentity(p *TeamPlan) error {
	if p.RootType != models.EntityTypeEpic && p.RootType != models.EntityTypeFeature {
		return fmt.Errorf("%w: root type %q must be epic or feature", ErrInvalidPlanInput, p.RootType)
	}
	if err := validateEntityIdentity(p.RootKey, p.RootType); err != nil {
		return fmt.Errorf("validate plan root identity: %w", err)
	}
	return nil
}

func validatePlanItems(items []TeamPlanItem) error {
	for _, item := range items {
		if err := validateEntityIdentity(item.ChildKey, item.ChildType); err != nil {
			return fmt.Errorf("validate plan item identity: %w", err)
		}
		for _, edge := range item.Dependencies {
			if err := validateEntityIdentity(edge.ChildKey, edge.ChildType); err != nil {
				return fmt.Errorf("validate plan dependency child identity: %w", err)
			}
			if err := validateEntityIdentity(edge.DependencyKey, edge.DependencyType); err != nil {
				return fmt.Errorf("validate plan dependency target identity: %w", err)
			}
		}
	}
	return nil
}

type canonicalPlan struct {
	RootKey          string              `json:"root_key"`
	RootType         models.EntityType   `json:"root_type"`
	ExecutionMode    ExecutionMode       `json:"execution_mode"`
	ConcurrencyLimit int                 `json:"concurrency_limit"`
	DegradedReason   string              `json:"degraded_reason"`
	Capability       []string            `json:"capability_exclusions"`
	Items            []canonicalPlanItem `json:"items"`
}

type canonicalPlanItem struct {
	ChildKey       string            `json:"child_key"`
	ChildType      models.EntityType `json:"child_type"`
	Status         string            `json:"status"`
	ExecutionOrder int               `json:"execution_order"`
	Priority       int               `json:"priority"`
	Wave           int               `json:"wave"`
	Dependencies   []canonicalEdge   `json:"dependencies"`
	Planned        DispatchMetadata  `json:"planned"`
	Eligible       bool              `json:"eligible"`
	Exclusion      string            `json:"exclusion"`
}

type canonicalEdge struct {
	ChildKey         string            `json:"child_key"`
	ChildType        models.EntityType `json:"child_type"`
	DependencyKey    string            `json:"dependency_key"`
	DependencyType   models.EntityType `json:"dependency_type"`
	DependencyStatus string            `json:"dependency_status"`
	External         bool              `json:"external"`
	Satisfied        bool              `json:"satisfied"`
	Resolved         bool              `json:"resolved"`
}

func (p *TeamPlan) computeHash() (string, error) {
	items := make([]canonicalPlanItem, 0, len(p.Items))
	for _, item := range p.Items {
		edges := make([]canonicalEdge, 0, len(item.Dependencies))
		for _, edge := range item.Dependencies {
			edges = append(edges, canonicalEdge{ChildKey: edge.ChildKey, ChildType: edge.ChildType, DependencyKey: edge.DependencyKey, DependencyType: edge.DependencyType, DependencyStatus: edge.DependencyStatus, External: edge.External, Satisfied: edge.Satisfied, Resolved: edge.Resolved})
		}
		sort.Slice(edges, func(i, j int) bool {
			if edges[i].DependencyType != edges[j].DependencyType {
				return edges[i].DependencyType < edges[j].DependencyType
			}
			return edges[i].DependencyKey < edges[j].DependencyKey
		})
		planned := item.Planned
		sort.Strings(planned.UnresolvedPlaceholders)
		items = append(items, canonicalPlanItem{ChildKey: item.ChildKey, ChildType: item.ChildType, Status: item.Status, ExecutionOrder: item.ExecutionOrder, Priority: item.Priority, Wave: item.Wave, Dependencies: edges, Planned: planned, Eligible: item.Eligible, Exclusion: item.ExclusionReason})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Wave != items[j].Wave {
			return items[i].Wave < items[j].Wave
		}
		if items[i].ExecutionOrder != items[j].ExecutionOrder {
			return items[i].ExecutionOrder < items[j].ExecutionOrder
		}
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		if items[i].ChildType != items[j].ChildType {
			return items[i].ChildType < items[j].ChildType
		}
		return items[i].ChildKey < items[j].ChildKey
	})
	capability := append([]string(nil), p.CapabilityExclusions...)
	sort.Strings(capability)
	encoded, err := json.Marshal(canonicalPlan{RootKey: p.RootKey, RootType: p.RootType, ExecutionMode: p.ExecutionMode, ConcurrencyLimit: p.ConcurrencyLimit, DegradedReason: p.DegradedReason, Capability: capability, Items: items})
	if err != nil {
		return "", fmt.Errorf("marshal canonical team plan: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
