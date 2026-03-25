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

	// Verify it's the default bug workflow (has 'reported' status)
	if _, ok := bugWorkflow.StatusMetadata["reported"]; !ok {
		t.Error("INT-03: Default bug workflow missing 'reported' status")
	}

	// Change level should return DefaultChangeCardWorkflow()
	changeWorkflow := multi.GetWorkflowForLevel(intTestLevelChange)
	if changeWorkflow == nil {
		t.Fatal("INT-03: GetWorkflowForLevel('change') returned nil for pre-E18 config")
	}

	// Verify it's the default change workflow (has 'proposed' status)
	if _, ok := changeWorkflow.StatusMetadata["proposed"]; !ok {
		t.Error("INT-03: Default change workflow missing 'proposed' status")
	}

	// Verify existing levels still work
	taskWorkflow := multi.GetWorkflowForLevel(intTestLevelTask)
	if taskWorkflow == nil {
		t.Fatal("INT-03: GetWorkflowForLevel('task') returned nil")
	}
	if _, ok := taskWorkflow.StatusMetadata["todo"]; !ok {
		t.Error("INT-03: Task workflow missing 'todo' status (regression)")
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
	if _, ok := changeWorkflow.StatusMetadata["proposed"]; !ok {
		t.Error("INT-04: Change workflow missing 'proposed' status")
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
	if _, ok := taskWorkflow.StatusMetadata["todo"]; !ok {
		t.Error("INT-06: Task workflow missing 'todo' status (regression)")
	}
	if _, ok := taskWorkflow.StatusMetadata["in_progress"]; !ok {
		t.Error("INT-06: Task workflow missing 'in_progress' status (regression)")
	}

	// Verify workflow isolation (epic statuses not in task, task statuses not in epic)
	if _, ok := taskWorkflow.StatusMetadata["draft"]; ok {
		t.Error("INT-06: Task workflow should not have 'draft' status (epic-only, regression)")
	}
	if _, ok := epicWorkflow.StatusMetadata["todo"]; ok {
		t.Error("INT-06: Epic workflow should not have 'todo' status (task-only, regression)")
	}

	// Verify bug/change dispatch does NOT interfere with task/epic/feature
	bugWorkflow := multi.GetWorkflowForLevel(intTestLevelBug)
	if bugWorkflow == nil {
		t.Fatal("INT-06: Bug workflow is nil")
	}

	// Bug statuses should not appear in task workflow
	for status := range bugWorkflow.StatusMetadata {
		if _, inTask := taskWorkflow.StatusMetadata[status]; inTask {
			// Only fail if there's a cross-contamination of the specific bug-only statuses
			if status == "reported" || status == "triaged" || status == "wont_fix" || status == "duplicate" {
				t.Errorf("INT-06: Task workflow should not contain bug-specific status '%s' (isolation regression)", status)
			}
		}
	}
}
