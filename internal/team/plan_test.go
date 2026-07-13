package team

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

type plannerChildrenMock struct {
	children []ChildSnapshot
}

func (m plannerChildrenMock) ListChildren(context.Context, models.EntityType, string) ([]ChildSnapshot, error) {
	return append([]ChildSnapshot(nil), m.children...), nil
}

type plannerDependenciesMock struct {
	byKey map[string][]DependencyEdge
}

func (m plannerDependenciesMock) ListDependencies(_ context.Context, child ChildIdentity) ([]DependencyEdge, error) {
	return append([]DependencyEdge(nil), m.byKey[child.Key]...), nil
}

type plannerDispatchMock struct {
	byKey map[string]dispatch.DispatchStep
}

func (m plannerDispatchMock) Resolve(_ context.Context, _ models.EntityType, key string) (dispatch.DispatchStep, error) {
	step, ok := m.byKey[key]
	if !ok {
		return dispatch.DispatchStep{}, errors.New("missing workflow step")
	}
	return step, nil
}

type plannerClaimsMock struct {
	byKey map[string]ClaimDiagnostic
}

func (m plannerClaimsMock) Diagnose(_ context.Context, child ChildIdentity) (ClaimDiagnostic, error) {
	return m.byKey[child.Key], nil
}

func plannerFixture() PlannerDeps {
	return PlannerDeps{
		Children: plannerChildrenMock{children: []ChildSnapshot{
			{Key: "T-E38-F01-001", EntityType: models.EntityTypeTask, Status: "completed", ExecutionOrder: 1, Priority: 1},
			{Key: "T-E38-F01-002", EntityType: models.EntityTypeTask, Status: "todo", ExecutionOrder: 2, Priority: 2},
			{Key: "T-E38-F01-003", EntityType: models.EntityTypeTask, Status: "todo", ExecutionOrder: 3, Priority: 3},
			{Key: "T-E38-F01-004", EntityType: models.EntityTypeTask, Status: "awaiting_approval", ExecutionOrder: 4, Priority: 4},
			{Key: "T-E38-F01-005", EntityType: models.EntityTypeTask, Status: "todo", ExecutionOrder: 5, Priority: 5},
		}},
		Dependencies: plannerDependenciesMock{byKey: map[string][]DependencyEdge{
			"T-E38-F01-003": {
				{ChildKey: "T-E38-F01-003", DependencyKey: "T-E38-F01-001", DependencyType: models.EntityTypeTask},
				{ChildKey: "T-E38-F01-003", DependencyKey: "T-E38-F01-002", DependencyType: models.EntityTypeTask},
			},
			"T-E38-F01-005": {{ChildKey: "T-E38-F01-005", DependencyKey: "T-EXTERNAL-001", DependencyType: models.EntityTypeTask, External: true, DependencyStatus: "blocked"}},
		}},
		Dispatch: plannerDispatchMock{byKey: map[string]dispatch.DispatchStep{
			"T-E38-F01-001": {EntityKey: "T-E38-F01-001", EntityType: models.EntityTypeTask, Status: "completed", Action: "archive", GateClassification: dispatch.GateTerminal},
			"T-E38-F01-002": {EntityKey: "T-E38-F01-002", EntityType: models.EntityTypeTask, Status: "todo", Action: "spawn_agent", AgentType: "developer", Provider: "anthropic", Model: "claude-sonnet", Effort: "medium"},
			"T-E38-F01-003": {EntityKey: "T-E38-F01-003", EntityType: models.EntityTypeTask, Status: "todo", Action: "spawn_agent", AgentType: "developer", Provider: "anthropic", Model: "claude-sonnet", Effort: "medium"},
			"T-E38-F01-004": {EntityKey: "T-E38-F01-004", EntityType: models.EntityTypeTask, Status: "awaiting_approval", Action: "check_or_resume", GateClassification: dispatch.GateHuman, UnresolvedPlaceholders: []string{"<approval_note>"}},
			"T-E38-F01-005": {EntityKey: "T-E38-F01-005", EntityType: models.EntityTypeTask, Status: "todo", Action: "spawn_agent", AgentType: "developer", Provider: "anthropic", Model: "claude-sonnet", Effort: "medium"},
		}},
		Claims: plannerClaimsMock{byKey: map[string]ClaimDiagnostic{
			"T-E38-F01-002": {Claimed: true, Reason: "claimed_by_other_session"},
		}},
	}
}

// TestPlanner_ReadOnlyCompleteSnapshot_TC001 drives Planner.Plan through its
// public entrypoint and verifies every child partition is retained once while
// no mutation seam is available to the planner.
func TestPlanner_ReadOnlyCompleteSnapshot_TC001(t *testing.T) {
	planner, err := NewPlanner(plannerFixture())
	if err != nil {
		t.Fatalf("NewPlanner() error = %v", err)
	}

	plan, err := planner.Plan(context.Background(), PlanInput{
		RootType:             models.EntityTypeFeature,
		RootKey:              "E38-F01",
		RequestedConcurrency: 2,
		Capabilities:         CapabilityFacts{TeamExecutionAvailable: true, SingleWorkerAvailable: true, WorktreeIsolationAvailable: true, ResourceOwnershipKnown: true, MaxConcurrency: 2},
	})
	if err != nil {
		t.Fatalf("Planner.Plan() error = %v", err)
	}
	if len(plan.Items) != 5 {
		t.Fatalf("Planner.Plan() returned %d items, want 5", len(plan.Items))
	}
	seen := make(map[string]bool)
	for _, item := range plan.Items {
		if seen[item.ChildKey] {
			t.Fatalf("child %q appears more than once", item.ChildKey)
		}
		seen[item.ChildKey] = true
	}
	if plan.Items[0].ChildKey != "T-E38-F01-001" {
		t.Errorf("terminal child should remain in deterministic plan: %+v", plan.Items[0])
	}
	byKey := make(map[string]TeamPlanItem, len(plan.Items))
	for _, item := range plan.Items {
		byKey[item.ChildKey] = item
	}
	if byKey["T-E38-F01-001"].ExclusionReason != ExclusionTerminal {
		t.Errorf("terminal exclusion = %q", byKey["T-E38-F01-001"].ExclusionReason)
	}
	if byKey["T-E38-F01-002"].ExclusionReason != ExclusionClaimed {
		t.Errorf("claimed exclusion = %q", byKey["T-E38-F01-002"].ExclusionReason)
	}
	if byKey["T-E38-F01-004"].ExclusionReason != ExclusionHumanGate {
		t.Errorf("gate exclusion = %q", byKey["T-E38-F01-004"].ExclusionReason)
	}
	if byKey["T-E38-F01-005"].ExclusionReason != ExclusionDependencyIneligible {
		t.Errorf("external dependency exclusion = %q", byKey["T-E38-F01-005"].ExclusionReason)
	}
	if byKey["T-E38-F01-003"].Wave <= byKey["T-E38-F01-002"].Wave {
		t.Errorf("dependent child wave = %d, predecessor wave = %d", byKey["T-E38-F01-003"].Wave, byKey["T-E38-F01-002"].Wave)
	}
	if byKey["T-E38-F01-002"].Planned.Provider != "anthropic" || byKey["T-E38-F01-002"].Planned.Model != "claude-sonnet" {
		t.Errorf("planned dispatch metadata missing: %+v", byKey["T-E38-F01-002"].Planned)
	}
	if plan.ExecutionMode != ExecutionModeParallel || plan.ConcurrencyLimit != 2 || len(plan.PlanHash) != 64 {
		t.Errorf("plan execution/hash = %q/%d/%q", plan.ExecutionMode, plan.ConcurrencyLimit, plan.PlanHash)
	}
}

func TestPlanner_ClassifiesResolvedRelationshipOutsideRootAsExternal_TC003(t *testing.T) {
	planner, err := NewPlanner(PlannerDeps{
		Children: plannerChildrenMock{children: []ChildSnapshot{
			{Key: "T-E38-F01-001", EntityType: models.EntityTypeTask, Status: "todo"},
		}},
		Dependencies: plannerDependenciesMock{byKey: map[string][]DependencyEdge{
			"T-E38-F01-001": {{ChildKey: "T-E38-F01-001", DependencyKey: "T-E37-F01-001", DependencyType: models.EntityTypeTask, Resolved: true, DependencyStatus: "completed", Source: "relationship"}},
		}},
		Dispatch: plannerDispatchMock{byKey: map[string]dispatch.DispatchStep{
			"T-E38-F01-001": {EntityKey: "T-E38-F01-001", EntityType: models.EntityTypeTask, Status: "todo", Action: "spawn_agent"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planner.Plan(context.Background(), PlanInput{RootType: models.EntityTypeFeature, RootKey: "E38-F01", RequestedConcurrency: 1, Capabilities: CapabilityFacts{SingleWorkerAvailable: true}})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(plan.Items) != 1 || len(plan.Items[0].Dependencies) != 1 {
		t.Fatalf("unexpected plan dependencies: %+v", plan.Items)
	}
	edge := plan.Items[0].Dependencies[0]
	if !edge.External {
		t.Fatalf("resolved relationship outside root labeled internal: %+v", edge)
	}
	if plan.Items[0].ExclusionReason != "" {
		t.Fatalf("satisfied external prerequisite excluded: %q", plan.Items[0].ExclusionReason)
	}
}

// TestPlanner_RejectsInvalidGraph_TC002 verifies cycle, missing-reference,
// and unresolved-workflow failures are typed and never become partial plans.
func TestPlanner_RejectsInvalidGraph_TC002(t *testing.T) {
	tests := []struct {
		name  string
		cause error
	}{
		{"cycle", ErrDependencyCycle},
		{"missing reference", ErrMissingDependency},
		{"unresolved workflow", ErrUnresolvedWorkflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := plannerFixture()
			children := deps.Children.(plannerChildrenMock)
			children.children = children.children[:2]
			deps.Children = children
			depMock := plannerDependenciesMock{byKey: map[string][]DependencyEdge{
				"T-E38-F01-002": {{ChildKey: "T-E38-F01-002", DependencyKey: "T-E38-F01-001", DependencyType: models.EntityTypeTask}},
			}}
			if tt.cause == ErrDependencyCycle {
				depMock.byKey["T-E38-F01-001"] = []DependencyEdge{{ChildKey: "T-E38-F01-001", DependencyKey: "T-E38-F01-002", DependencyType: models.EntityTypeTask}}
			} else if tt.cause == ErrMissingDependency {
				depMock.byKey["T-E38-F01-002"] = []DependencyEdge{{ChildKey: "T-E38-F01-002", DependencyKey: "T-MISSING-999", DependencyType: models.EntityTypeTask}}
			} else {
				steps := deps.Dispatch.(plannerDispatchMock)
				steps.byKey["T-E38-F01-002"] = dispatch.DispatchStep{EntityKey: "T-E38-F01-002", Error: "unresolved placeholder workflow"}
				deps.Dispatch = steps
			}
			deps.Dependencies = depMock
			planner, err := NewPlanner(deps)
			if err != nil {
				t.Fatalf("NewPlanner() error = %v", err)
			}
			_, err = planner.Plan(context.Background(), PlanInput{RootType: models.EntityTypeFeature, RootKey: "E38-F01", RequestedConcurrency: 1, Capabilities: CapabilityFacts{SingleWorkerAvailable: true}})
			if err == nil || !errors.Is(err, tt.cause) {
				t.Fatalf("Planner.Plan() error = %v, want %v", err, tt.cause)
			}
		})
	}
}

// TestPlanner_SelectsBoundedParallelMode_TC004 verifies the requested limit is
// capped by host capability facts and planning never starts a worker.
func TestPlanner_SelectsBoundedParallelMode_TC004(t *testing.T) {
	deps := plannerFixture()
	planner, err := NewPlanner(deps)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planner.Plan(context.Background(), PlanInput{RootType: models.EntityTypeFeature, RootKey: "E38-F01", RequestedConcurrency: 4, Capabilities: CapabilityFacts{TeamExecutionAvailable: true, SingleWorkerAvailable: true, WorktreeIsolationAvailable: true, ResourceOwnershipKnown: true, MaxConcurrency: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionMode != ExecutionModeParallel || plan.ConcurrencyLimit != 2 {
		t.Fatalf("mode/limit = %q/%d, want parallel/2", plan.ExecutionMode, plan.ConcurrencyLimit)
	}
}

// TestPlanner_UsesExplicitSequentialFallback_TC005 verifies unsafe parallel
// capability facts produce a safe, actionable sequential plan.
func TestPlanner_UsesExplicitSequentialFallback_TC005(t *testing.T) {
	planner, err := NewPlanner(plannerFixture())
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planner.Plan(context.Background(), PlanInput{RootType: models.EntityTypeFeature, RootKey: "E38-F01", RequestedConcurrency: 4, Capabilities: CapabilityFacts{SingleWorkerAvailable: true, WorktreeIsolationAvailable: false}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionMode != ExecutionModeSequential || plan.ConcurrencyLimit != 1 || plan.DegradedReason != DegradedReasonParallelUnavailable {
		t.Fatalf("fallback = %q/%d/%q", plan.ExecutionMode, plan.ConcurrencyLimit, plan.DegradedReason)
	}
}

// TestPlanHash_DetectsMaterialDrift_TC008 verifies material child state changes
// drift the hash while equivalent reader ordering does not.
func TestPlanHash_DetectsMaterialDrift_TC008(t *testing.T) {
	deps := plannerFixture()
	planner, err := NewPlanner(deps)
	if err != nil {
		t.Fatal(err)
	}
	input := PlanInput{RootType: models.EntityTypeFeature, RootKey: "E38-F01", RequestedConcurrency: 2, Capabilities: CapabilityFacts{TeamExecutionAvailable: true, SingleWorkerAvailable: true, WorktreeIsolationAvailable: true, ResourceOwnershipKnown: true, MaxConcurrency: 2}}
	first, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	children := deps.Children.(plannerChildrenMock)
	children.children[1], children.children[2] = children.children[2], children.children[1]
	deps.Children = children
	planner, err = NewPlanner(deps)
	if err != nil {
		t.Fatal(err)
	}
	second, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanHash != second.PlanHash {
		t.Fatalf("equivalent permutation changed plan hash: %s != %s", first.PlanHash, second.PlanHash)
	}
	children.children[0].Status = "in_progress"
	deps.Children = children
	planner, _ = NewPlanner(deps)
	third, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanHash == third.PlanHash {
		t.Fatal("material status change did not change plan hash")
	}
}
