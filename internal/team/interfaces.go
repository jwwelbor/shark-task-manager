package team

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
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
