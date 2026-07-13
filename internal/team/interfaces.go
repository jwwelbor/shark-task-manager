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
	Children     ChildSnapshotReader
	Dependencies DependencyReader
	Dispatch     DispatchStepResolver
	Claims       ClaimDiagnosticReader
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
}
