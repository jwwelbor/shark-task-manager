package models

import "time"

// PortfolioAdviceMode identifies the internal portfolio evidence mode.
type PortfolioAdviceMode string

const (
	// PortfolioAdviceModePortfolioAdvice is the read-only portfolio advice mode.
	PortfolioAdviceModePortfolioAdvice PortfolioAdviceMode = "portfolio_advice"
)

// PortfolioEligibility describes whether an epic is eligible for recommendation.
type PortfolioEligibility string

const (
	// PortfolioEligibilityEligible means the available evidence permits the epic.
	PortfolioEligibilityEligible PortfolioEligibility = "eligible"
	// PortfolioEligibilityIneligible means the available evidence blocks the epic.
	PortfolioEligibilityIneligible PortfolioEligibility = "ineligible"
	// PortfolioEligibilityUnknown means required eligibility evidence is unavailable.
	PortfolioEligibilityUnknown PortfolioEligibility = "unknown"
)

// PortfolioBlockerKind identifies the source of portfolio blocker evidence.
type PortfolioBlockerKind string

const (
	// PortfolioBlockerWorkflowBlocked identifies an entity in a workflow-blocked status.
	PortfolioBlockerWorkflowBlocked PortfolioBlockerKind = "workflow_blocked"
	// PortfolioBlockerHardDependency identifies an unresolved depends-on prerequisite.
	PortfolioBlockerHardDependency PortfolioBlockerKind = "hard_dependency"
	// PortfolioBlockerIncomingBlock identifies an unresolved incoming blocks relationship.
	PortfolioBlockerIncomingBlock PortfolioBlockerKind = "incoming_block"
)

// PortfolioWarningCode identifies a stable portfolio evidence or ordering diagnostic.
type PortfolioWarningCode string

const (
	// PortfolioWarningHardOrderCycle identifies a cycle in unresolved hard precedence.
	PortfolioWarningHardOrderCycle PortfolioWarningCode = "HARD_ORDER_CYCLE"
	// PortfolioWarningRoadmapOrderCycle identifies a cycle after advisory order is included.
	PortfolioWarningRoadmapOrderCycle PortfolioWarningCode = "ROADMAP_ORDER_CYCLE"
	// PortfolioWarningContradictoryOrder identifies precedence in both directions.
	PortfolioWarningContradictoryOrder PortfolioWarningCode = "CONTRADICTORY_ORDER"
	// PortfolioWarningMissingOrdering identifies incomparable eligible first-layer epics.
	PortfolioWarningMissingOrdering PortfolioWarningCode = "MISSING_ORDERING"
	// PortfolioWarningChildStateUnavailable identifies unavailable descendant evidence.
	PortfolioWarningChildStateUnavailable PortfolioWarningCode = "CHILD_STATE_UNAVAILABLE"
	// PortfolioWarningRelationshipStateUnavailable identifies unavailable relationship evidence.
	PortfolioWarningRelationshipStateUnavailable PortfolioWarningCode = "RELATIONSHIP_STATE_UNAVAILABLE"
	// PortfolioWarningClaimStateUnavailable identifies unavailable active-work evidence.
	PortfolioWarningClaimStateUnavailable PortfolioWarningCode = "CLAIM_STATE_UNAVAILABLE"
	// PortfolioWarningUnknownWorkflowStatus identifies a status absent from its workflow.
	PortfolioWarningUnknownWorkflowStatus PortfolioWarningCode = "UNKNOWN_WORKFLOW_STATUS"
	// PortfolioWarningDanglingRelationship identifies an unresolved relationship endpoint.
	PortfolioWarningDanglingRelationship PortfolioWarningCode = "DANGLING_RELATIONSHIP"
)

// PortfolioAdviceEnvelope is the read-only evidence assembled for bare
// `shark plan` epic selection.
type PortfolioAdviceEnvelope struct {
	Mode             PortfolioAdviceMode         `json:"mode"`
	EvidenceComplete bool                        `json:"evidence_complete"`
	Epics            []PortfolioEpicEvidence     `json:"epics"`
	Relationships    []PortfolioEpicRelationship `json:"relationships"`
	Ordering         PortfolioOrdering           `json:"ordering"`
	Warnings         []PortfolioWarning          `json:"warnings"`
	Prompt           string                      `json:"prompt"`
}

// PortfolioEpicEvidence is the advisory projection of one non-terminal epic.
type PortfolioEpicEvidence struct {
	Key                string                 `json:"key"`
	Title              string                 `json:"title"`
	Status             string                 `json:"status"`
	Priority           string                 `json:"priority"`
	BusinessValue      *string                `json:"business_value"`
	ProgressPct        float64                `json:"progress_pct"`
	Eligibility        PortfolioEligibility   `json:"eligibility"`
	EligibilityReasons []string               `json:"eligibility_reasons"`
	BlockedItems       []PortfolioBlockedItem `json:"blocked_items"`
	ActiveWork         []PortfolioActiveWork  `json:"active_work"`
}

// PortfolioBlockedItem describes one workflow or relationship blocker.
type PortfolioBlockedItem struct {
	Kind       PortfolioBlockerKind `json:"kind"`
	EntityType string               `json:"entity_type"`
	EntityKey  string               `json:"entity_key"`
	Title      string               `json:"title"`
	Status     string               `json:"status"`
}

// PortfolioActiveWork is the sanitized public projection of an entity claim.
type PortfolioActiveWork struct {
	EntityType    string    `json:"entity_type"`
	EntityKey     string    `json:"entity_key"`
	ClaimedBy     string    `json:"claimed_by"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	Progress      *float64  `json:"progress"`
}

// PortfolioEpicRelationship describes one supported epic-to-epic relationship.
type PortfolioEpicRelationship struct {
	FromKey          string                 `json:"from_key"`
	FromStatus       string                 `json:"from_status"`
	RelationshipType EntityRelationshipType `json:"relationship_type"`
	ToKey            string                 `json:"to_key"`
	ToStatus         string                 `json:"to_status"`
	Hard             bool                   `json:"hard"`
	Satisfied        *bool                  `json:"satisfied"`
}

// PortfolioWarning is an actionable portfolio diagnostic.
type PortfolioWarning struct {
	Code     PortfolioWarningCode `json:"code"`
	Message  string               `json:"message"`
	EpicKeys []string             `json:"epic_keys"`
}

// PortfolioOrdering contains deterministic dependency and roadmap graph output.
type PortfolioOrdering struct {
	DependencyLayers [][]string         `json:"dependency_layers"`
	RoadmapLayers    [][]string         `json:"roadmap_layers"`
	UnlayeredEpics   []string           `json:"unlayered_epics"`
	Warnings         []PortfolioWarning `json:"warnings"`
}

// NewPortfolioAdviceEnvelope creates an empty portfolio-advice envelope whose
// collection fields marshal as JSON arrays rather than null.
func NewPortfolioAdviceEnvelope() *PortfolioAdviceEnvelope {
	return &PortfolioAdviceEnvelope{
		Mode:          PortfolioAdviceModePortfolioAdvice,
		Epics:         []PortfolioEpicEvidence{},
		Relationships: []PortfolioEpicRelationship{},
		Ordering: PortfolioOrdering{
			DependencyLayers: [][]string{},
			RoadmapLayers:    [][]string{},
			UnlayeredEpics:   []string{},
			Warnings:         []PortfolioWarning{},
		},
		Warnings: []PortfolioWarning{},
	}
}
