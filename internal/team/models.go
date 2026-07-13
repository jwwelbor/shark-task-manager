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
	"regexp"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

var (
	ErrInvalidPlanInput      = errors.New("invalid team plan input")
	ErrDependencyCycle       = errors.New("team plan dependency cycle")
	ErrMissingDependency     = errors.New("team plan missing dependency")
	ErrMalformedDependency   = errors.New("team plan malformed dependency")
	ErrUnresolvedWorkflow    = errors.New("team plan unresolved workflow")
	ErrCapabilityUnavailable = errors.New("team plan capability unavailable")
	ErrDuplicateChild        = errors.New("team plan duplicate child")
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
