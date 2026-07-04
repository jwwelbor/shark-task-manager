package workflow

import (
	"testing"
)

// Level constants (same values as workflow.Level* constants, but defined here to avoid import cycle)
const (
	intTestLevelBug     = "bug"
	intTestLevelChange  = "change"
	intTestLevelTask    = "task"
	intTestLevelEpic    = "epic"
	intTestLevelFeature = "feature"
)

// INT-03: Pre-E18 config loads without error; bug/change workflows fall back to defaults
// Verifies that MultiLevelWorkflow with nil Bug/Change fields returns default workflows.
//
// NOTE: bug/change/task workflows are now route-based (loaded from embedded
// workflow/*.yaml). The old hardcoded status names ("reported", "proposed",
// "todo") survive only as backward-compat aliases (or not at all, for
// "todo") -- see internal/config/workflow/steps.go ResolveAlias. Assertions
// below check the real step names and, where a legacy name is meant to keep
// working as an alias, verify it via ResolveAlias.
func TestE18F01_INT03_WorkflowEngineForwardCompatibility(t *testing.T) {
	// Simulate a config without bug_workflow/change_workflow (pre-E18 config)
	// MultiLevelWorkflow with nil Bug and Change fields should fall back to defaults
	multi := &MultiLevelWorkflow{
		// Bug and Change are nil (not set) -- simulates pre-E18 config
	}

	// Bug level should return DefaultBugWorkflow()
	bugWorkflow := multi.GetWorkflowForLevel(intTestLevelBug)
	if bugWorkflow == nil {
		t.Fatal("INT-03: GetWorkflowForLevel('bug') returned nil for pre-E18 config")
	}

	// Verify it's the default bug workflow (has 'draft' status, and the old
	// 'reported' name still resolves to it via alias)
	if _, ok := bugWorkflow.StatusMetadata["draft"]; !ok {
		t.Error("INT-03: Default bug workflow missing 'draft' status")
	}
	if got := bugWorkflow.ResolveAlias("reported"); got != "draft" {
		t.Errorf("INT-03: Default bug workflow's 'reported' alias should resolve to 'draft', got %q", got)
	}

	// Change level should return DefaultChangeCardWorkflow()
	changeWorkflow := multi.GetWorkflowForLevel(intTestLevelChange)
	if changeWorkflow == nil {
		t.Fatal("INT-03: GetWorkflowForLevel('change') returned nil for pre-E18 config")
	}

	// Verify it's the default change workflow (has 'draft' status, and the old
	// 'proposed' name still resolves to it via alias)
	if _, ok := changeWorkflow.StatusMetadata["draft"]; !ok {
		t.Error("INT-03: Default change workflow missing 'draft' status")
	}
	if got := changeWorkflow.ResolveAlias("proposed"); got != "draft" {
		t.Errorf("INT-03: Default change workflow's 'proposed' alias should resolve to 'draft', got %q", got)
	}

	// Verify existing levels still work
	taskWorkflow := multi.GetWorkflowForLevel(intTestLevelTask)
	if taskWorkflow == nil {
		t.Fatal("INT-03: GetWorkflowForLevel('task') returned nil")
	}
	if _, ok := taskWorkflow.StatusMetadata["draft"]; !ok {
		t.Error("INT-03: Task workflow missing 'draft' status (regression)")
	}
}

// INT-04: Profile update + workflow resolution
// After setting profile-defined Bug/Change workflows, GetWorkflowForLevel returns them.
func TestE18F01_INT04_ProfileUpdateWorkflowResolution(t *testing.T) {
	// Build custom workflows (simulating profile-loaded config)
	customBugWorkflow := DefaultBugWorkflow()
	customBugWorkflow.StatusMetadata["custom_status"] = StatusMetadata{
		Color: "purple",
		Phase: "planning",
	}

	customChangeWorkflow := DefaultChangeCardWorkflow()

	multi := &MultiLevelWorkflow{
		Bug:    customBugWorkflow,
		Change: customChangeWorkflow,
	}

	// Bug workflow should return profile-defined (with custom status)
	bugWorkflow := multi.GetWorkflowForLevel(intTestLevelBug)
	if bugWorkflow == nil {
		t.Fatal("INT-04: GetWorkflowForLevel('bug') returned nil for profile-defined workflow")
	}
	if _, ok := bugWorkflow.StatusMetadata["custom_status"]; !ok {
		t.Error("INT-04: Expected profile-defined bug workflow with custom_status, got default")
	}

	// Change workflow should return profile-defined
	changeWorkflow := multi.GetWorkflowForLevel(intTestLevelChange)
	if changeWorkflow == nil {
		t.Fatal("INT-04: GetWorkflowForLevel('change') returned nil")
	}
	// "proposed" is now a backward-compat alias for "draft" (change.yaml),
	// not a StatusMetadata key.
	if got := changeWorkflow.ResolveAlias("proposed"); got != "draft" {
		t.Errorf("INT-04: Change workflow's 'proposed' alias should resolve to 'draft', got %q", got)
	}
	if _, ok := changeWorkflow.StatusMetadata["draft"]; !ok {
		t.Error("INT-04: Change workflow missing 'draft' status")
	}

	// Other levels should still return their own defaults
	epicWorkflow := multi.GetWorkflowForLevel(intTestLevelEpic)
	if epicWorkflow == nil {
		t.Fatal("INT-04: GetWorkflowForLevel('epic') returned nil")
	}
	featureWorkflow := multi.GetWorkflowForLevel(intTestLevelFeature)
	if featureWorkflow == nil {
		t.Fatal("INT-04: GetWorkflowForLevel('feature') returned nil")
	}
}

// INT-06: Regression -- existing entity workflows unchanged
// Verifies that GetWorkflowForLevel still works correctly for epic, feature, task levels.
//
// NOTE: under the route-based schema, "draft" is common vocabulary shared by
// epic/feature/task/bug/change (each entity's start step), so its mere
// presence across entity types is no longer evidence of cross-contamination
// -- each GetWorkflowForLevel call returns its own independently-loaded
// WorkflowConfig. The isolation checks below instead verify: (a) each
// workflow still carries its own entity-specific steps the others don't
// define (epic's "decomposition" vs. task's "development"), and (b) a step
// name shared across entities (e.g. "development") is populated from that
// entity's own YAML, not another entity's (checked via the orchestrator
// prompt path, which is entity-namespaced: "task/development.md" vs.
// "bug/development.md").
func TestE18F01_INT06_ExistingEntityWorkflowsUnchanged(t *testing.T) {
	// Use nil MultiLevelWorkflow (defaults only, simulates production config)
	multi := &MultiLevelWorkflow{}

	// Epic workflow
	epicWorkflow := multi.GetWorkflowForLevel(intTestLevelEpic)
	if epicWorkflow == nil {
		t.Fatal("INT-06: Epic workflow is nil (regression)")
	}
	if _, ok := epicWorkflow.StatusMetadata["draft"]; !ok {
		t.Error("INT-06: Epic workflow missing 'draft' status (regression)")
	}
	if _, ok := epicWorkflow.StatusMetadata["active"]; !ok {
		t.Error("INT-06: Epic workflow missing 'active' status (regression)")
	}

	// Feature workflow
	featureWorkflow := multi.GetWorkflowForLevel(intTestLevelFeature)
	if featureWorkflow == nil {
		t.Fatal("INT-06: Feature workflow is nil (regression)")
	}
	if _, ok := featureWorkflow.StatusMetadata["draft"]; !ok {
		t.Error("INT-06: Feature workflow missing 'draft' status (regression)")
	}

	// Task workflow
	taskWorkflow := multi.GetWorkflowForLevel(intTestLevelTask)
	if taskWorkflow == nil {
		t.Fatal("INT-06: Task workflow is nil (regression)")
	}
	if _, ok := taskWorkflow.StatusMetadata["draft"]; !ok {
		t.Error("INT-06: Task workflow missing 'draft' status (regression)")
	}
	if _, ok := taskWorkflow.StatusMetadata["development"]; !ok {
		t.Error("INT-06: Task workflow missing 'development' status (regression)")
	}

	// Verify workflow isolation: epic-only steps aren't in task, and vice versa.
	if _, ok := taskWorkflow.StatusMetadata["decomposition"]; ok {
		t.Error("INT-06: Task workflow should not have 'decomposition' status (epic-only, regression)")
	}
	if _, ok := epicWorkflow.StatusMetadata["development"]; ok {
		t.Error("INT-06: Epic workflow should not have 'development' status (task-only, regression)")
	}

	// Verify bug/change dispatch does NOT interfere with task/epic/feature
	bugWorkflow := multi.GetWorkflowForLevel(intTestLevelBug)
	if bugWorkflow == nil {
		t.Fatal("INT-06: Bug workflow is nil")
	}

	// "development" is a shared step name (task and bug both define it), but
	// each entity's step is populated from its own YAML: verify the task
	// workflow's "development" step still points at the task-specific prompt,
	// not the bug workflow's.
	taskDevMeta, ok := taskWorkflow.StatusMetadata["development"]
	if !ok || taskDevMeta.OrchestratorAction == nil {
		t.Fatal("INT-06: Task workflow 'development' status missing orchestrator action")
	}
	if taskDevMeta.OrchestratorAction.InstructionTemplate != "task/development.md" {
		t.Errorf("INT-06: Task workflow 'development' should use 'task/development.md', got %q (isolation regression)",
			taskDevMeta.OrchestratorAction.InstructionTemplate)
	}
	bugDevMeta, ok := bugWorkflow.StatusMetadata["development"]
	if !ok || bugDevMeta.OrchestratorAction == nil {
		t.Fatal("INT-06: Bug workflow 'development' status missing orchestrator action")
	}
	if bugDevMeta.OrchestratorAction.InstructionTemplate != "bug/development.md" {
		t.Errorf("INT-06: Bug workflow 'development' should use 'bug/development.md', got %q (isolation regression)",
			bugDevMeta.OrchestratorAction.InstructionTemplate)
	}
}
