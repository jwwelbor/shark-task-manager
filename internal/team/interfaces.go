package team

import (
	"context"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// Planner is the public read-only planning contract.
type Planner interface {
	Plan(ctx context.Context, input PlanInput) (*TeamPlan, error)
}

// ChildSnapshotReader supplies every direct child, not merely currently
// dispatchable children.
type ChildSnapshotReader interface {
	ListChildren(ctx context.Context, rootType models.EntityType, rootKey string) ([]ChildSnapshot, error)
}

// DependencyReader supplies normalized prerequisite edges for one child.
type DependencyReader interface {
	ListDependencies(ctx context.Context, child ChildIdentity) ([]DependencyEdge, error)
}

// DispatchStepResolver is re-exported at the team boundary so consumers can
// provide the canonical resolver without importing a CLI package.
type DispatchStepResolver = dispatch.DispatchStepResolver

type ClaimDiagnosticReader interface {
	Diagnose(ctx context.Context, child ChildIdentity) (ClaimDiagnostic, error)
}

type PlannerDeps struct {
	Children         ChildSnapshotReader
	Dependencies     DependencyReader
	Dispatch         DispatchStepResolver
	Claims           ClaimDiagnosticReader
	SuccessfulStatus func(models.EntityType, string) bool
}

// LegacyDependencySource is the compatibility input for tasks.depends_on.
// The source returns the raw JSON so malformed data cannot be silently
// discarded by the adapter.
type LegacyDependencySource interface {
	ListLegacyDependencies(ctx context.Context, child ChildIdentity) (string, error)
}

type RelationshipDependencySource interface {
	ListRelationshipDependencies(ctx context.Context, child ChildIdentity) ([]DependencyEdge, error)
}

// SchedulerLedger is the optional execution extension of Ledger. F01 remains
// the source of truth for reads and terminal results; these CAS operations are
// deliberately narrow so workers never run inside a database transaction.
type SchedulerLedger interface {
	Ledger
	// RecordPreClaimResult is the coordinator-only CAS path for terminal
	// diagnostics discovered before a child claim exists.
	RecordPreClaimResult(ctx context.Context, update ItemResultUpdate, coordinatorSessionID string) (*TeamRunItem, error)
	ClaimItem(ctx context.Context, runID, itemID int64, attempt int, claimSessionID string) (bool, error)
	StartItem(ctx context.Context, runID, itemID int64, attempt int, claimSessionID, workerSessionID string) (bool, error)
}

// TeamClaims is the scheduler's session-safe claim boundary.
type TeamClaims interface {
	Claim(ctx context.Context, input services.ClaimInput) (*models.EntityClaim, error)
	Heartbeat(ctx context.Context, entityType, entityKey, sessionID string, progress *float64, note string) error
	Release(ctx context.Context, entityType, entityKey, sessionID, outcome string, force bool) (bool, error)
}

// ResourcePolicy reports the maximum safe execution mode for the confirmed
// snapshot. Unknown or overlapping ownership must be conservative.
type ResourcePolicy interface {
	Select(ctx context.Context, run *TeamRun, items []*TeamRunItem) (ExecutionMode, int, string, error)
}

// EventSink receives safe, structured scheduler events. Implementations must
// not include prompts, credentials, or unrestricted worker output.
type SchedulerEvent struct {
	RunID         int64
	RootKey       string
	ChildKey      string
	Wave          int
	ItemStatus    ItemStatus
	Provider      string
	Duration      time.Duration
	ClaimSession  string
	WorkerSession string
	Outcome       string
	// Communication carries the bounded council context that informed this
	// execution. It is metadata only; claims and workflow remain authoritative.
	Communication *CouncilCommunication
}

// CouncilCommunication is the scheduler's read-only projection of the F04
// council contract. Keeping it here lets execution consume the shared shape
// without making roster YAML a mutation or routing authority.
type CouncilCommunication struct {
	SenderRole      string           `json:"sender_role"`
	RecipientRole   string           `json:"recipient_role"`
	RootKey         string           `json:"root_key"`
	ChildKey        string           `json:"child_key"`
	Subject         string           `json:"subject"`
	RequestedAction string           `json:"requested_action"`
	Urgency         string           `json:"urgency"`
	EvidenceLinks   []string         `json:"evidence_links,omitempty"`
	Handoff         *CouncilHandoff  `json:"handoff,omitempty"`
	Decision        *CouncilDecision `json:"decision,omitempty"`
}

type CouncilHandoff struct {
	Summary       string   `json:"summary,omitempty"`
	OpenQuestions []string `json:"open_questions,omitempty"`
}

type CouncilDecision struct {
	Outcome   string `json:"outcome,omitempty"`
	Rationale string `json:"rationale,omitempty"`
}

type EventSink interface {
	Emit(context.Context, SchedulerEvent)
}

type SchedulerDeps struct {
	Ledger     SchedulerLedger
	Claims     TeamClaims
	Resolver   DispatchStepResolver
	Dispatcher runner.AgentDispatcher
	Resource   ResourcePolicy
	Events     EventSink
	Now        func() time.Time
	// HeartbeatInterval controls coordinator lease renewal. A non-positive
	// value disables periodic renewal while preserving the initial heartbeat.
	HeartbeatInterval time.Duration
	// CleanupTimeout bounds release/result/final-run persistence after the
	// caller cancels. Zero uses the scheduler default.
	CleanupTimeout time.Duration
	// Communication is optional execution metadata supplied by the council
	// protocol. The scheduler never derives workflow authority from it.
	Communication *CouncilCommunication
	// ExpectedPlanHash is captured at confirmation and checked before mutation.
	ExpectedPlanHash string
	// CoordinatorTransition performs promptless workflow transitions such as
	// advance_status. It is coordinator-owned and never delegated to workers.
	CoordinatorTransition func(context.Context, models.EntityType, string, *services.NextStatusInfo) error
}
