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
	Source           string            `json:"source,omitempty"`
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
	Items            []*TeamRunItem    `json:"items"`
}

// NewTeamRunResult converts a persisted run and its complete item list into
// the shared result shape without introducing prompt or worker-output fields.
func NewTeamRunResult(run *TeamRun, items []*TeamRunItem) (*TeamRunResult, error) {
	if err := run.Validate(); err != nil {
		return nil, err
	}
	result := &TeamRunResult{RunID: run.ID, RootKey: run.RootKey, RootType: run.RootType, Status: run.Status, ExecutionMode: run.ExecutionMode, ConcurrencyLimit: run.ConcurrencyLimit, PlanHash: run.PlanHash, AggregateOutcome: run.AggregateOutcome, NextAction: run.NextAction, Items: make([]*TeamRunItem, 0, len(items))}
	for _, item := range items {
		if err := item.Validate(); err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

// Validate checks the persisted run invariants before repository access.
func (r *TeamRun) Validate() error {
	if r == nil {
		return fmt.Errorf("%w: run is nil", ErrInvalidPlanInput)
	}
	if err := validateEntityKey(r.RootKey); err != nil {
		return fmt.Errorf("validate run root key: %w", err)
	}
	if r.RootType != models.EntityTypeEpic && r.RootType != models.EntityTypeFeature {
		return fmt.Errorf("%w: %q", ErrInvalidPlanInput, r.RootType)
	}
	if !validRunStatus(r.Status) {
		return fmt.Errorf("%w: %q", ErrInvalidRunStatus, r.Status)
	}
	if r.ExecutionMode != ExecutionModeParallel && r.ExecutionMode != ExecutionModeSequential {
		return fmt.Errorf("%w: %q", ErrInvalidPlanInput, r.ExecutionMode)
	}
	if r.ConcurrencyLimit <= 0 || !sha256Hex.MatchString(r.PlanHash) {
		return fmt.Errorf("%w: run limit and plan hash are invalid", ErrInvalidPlanInput)
	}
	return nil
}

// Validate checks item identity, lifecycle, dependency, and attempt bounds.
func (i *TeamRunItem) Validate() error {
	if i == nil {
		return fmt.Errorf("%w: item is nil", ErrInvalidPlanInput)
	}
	if err := validateEntityKey(i.ChildKey); err != nil {
		return fmt.Errorf("validate item key: %w", err)
	}
	if !models.ValidEntityTypes[i.ChildType] || i.Wave < 0 || i.ExecutionOrder < 0 || i.Attempt < 0 {
		return fmt.Errorf("%w: item identity, wave, order, or attempt is invalid", ErrInvalidPlanInput)
	}
	if !validItemStatus(i.ItemStatus) {
		return fmt.Errorf("%w: %q", ErrInvalidItemStatus, i.ItemStatus)
	}
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
	seen := make(map[string]struct{}, len(u.ArtifactRefs))
	for _, ref := range u.ArtifactRefs {
		if err := validateArtifactRef(ref); err != nil {
			return err
		}
		if _, exists := seen[ref]; exists {
			return fmt.Errorf("%w: duplicate artifact reference %q", ErrInvalidArtifactPath, ref)
		}
		seen[ref] = struct{}{}
	}
	return nil
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

func validateArtifactRef(ref string) error {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" || len([]byte(trimmed)) > maxArtifactPath || filepath.IsAbs(trimmed) || strings.HasPrefix(trimmed, "\\") {
		return fmt.Errorf("%w: %q", ErrInvalidArtifactPath, ref)
	}
	for _, part := range strings.FieldsFunc(trimmed, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == ".." {
			return fmt.Errorf("%w: %q", ErrInvalidArtifactPath, ref)
		}
	}
	if filepath.Clean(trimmed) == "." || strings.Contains(trimmed, "\x00") {
		return fmt.Errorf("%w: %q", ErrInvalidArtifactPath, ref)
	}
	return nil
}

func validRunStatus(status RunStatus) bool {
	switch status {
	case RunStatusPlanned, RunStatusRunning, RunStatusPaused, RunStatusFailed, RunStatusCompleted, RunStatusCancelled:
		return true
	}
	return false
}

func validItemStatus(status ItemStatus) bool {
	switch status {
	case ItemStatusPlanned, ItemStatusClaimed, ItemStatusRunning, ItemStatusCompleted, ItemStatusFailed, ItemStatusBlocked, ItemStatusPaused, ItemStatusSkipped, ItemStatusCancelled:
		return true
	}
	return false
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
	if p.RootType != models.EntityTypeEpic && p.RootType != models.EntityTypeFeature {
		return fmt.Errorf("%w: root type %q must be epic or feature", ErrInvalidPlanInput, p.RootType)
	}
	if strings.TrimSpace(p.RootKey) == "" || p.ConcurrencyLimit <= 0 {
		return fmt.Errorf("%w: root key and positive concurrency limit are required", ErrInvalidPlanInput)
	}
	if p.ExecutionMode != ExecutionModeParallel && p.ExecutionMode != ExecutionModeSequential {
		return fmt.Errorf("%w: invalid execution mode %q", ErrInvalidPlanInput, p.ExecutionMode)
	}
	if !sha256Hex.MatchString(p.PlanHash) {
		return fmt.Errorf("%w: plan hash must be lowercase SHA-256", ErrInvalidPlanInput)
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
}

func (p *TeamPlan) computeHash() (string, error) {
	items := make([]canonicalPlanItem, 0, len(p.Items))
	for _, item := range p.Items {
		edges := make([]canonicalEdge, 0, len(item.Dependencies))
		for _, edge := range item.Dependencies {
			edges = append(edges, canonicalEdge{ChildKey: edge.ChildKey, ChildType: edge.ChildType, DependencyKey: edge.DependencyKey, DependencyType: edge.DependencyType, DependencyStatus: edge.DependencyStatus, External: edge.External, Satisfied: edge.Satisfied})
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
