package workflow

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestLevelConstants(t *testing.T) {
	if LevelEpic != "epic" {
		t.Errorf("expected LevelEpic = 'epic', got %q", LevelEpic)
	}
	if LevelFeature != "feature" {
		t.Errorf("expected LevelFeature = 'feature', got %q", LevelFeature)
	}
	if LevelTask != "task" {
		t.Errorf("expected LevelTask = 'task', got %q", LevelTask)
	}
}

// newTestService creates a Service with default workflows for testing.
// Does not require a project root or config file.
func newTestService() *Service {
	multi := &config.MultiLevelWorkflow{} // all nil => defaults
	return &Service{
		workflow:    multi.GetWorkflowForLevel(LevelTask),
		projectRoot: "",
		level:       LevelTask,
		multiLevel:  multi,
	}
}

func TestForLevel_Epic(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)

	if epicSvc.GetLevel() != LevelEpic {
		t.Errorf("expected level %q, got %q", LevelEpic, epicSvc.GetLevel())
	}

	// Epic default workflow has "draft" as a valid status
	if !epicSvc.IsValidStatus("draft") {
		t.Error("expected 'draft' to be a valid epic status")
	}
	if !epicSvc.IsValidStatus("active") {
		t.Error("expected 'active' to be a valid epic status")
	}
	// "todo" is a task-only status
	if epicSvc.IsValidStatus("todo") {
		t.Error("expected 'todo' to NOT be a valid epic status")
	}
}

func TestForLevel_Feature(t *testing.T) {
	svc := newTestService()
	featureSvc := svc.ForLevel(LevelFeature)

	if featureSvc.GetLevel() != LevelFeature {
		t.Errorf("expected level %q, got %q", LevelFeature, featureSvc.GetLevel())
	}

	if !featureSvc.IsValidStatus("draft") {
		t.Error("expected 'draft' to be a valid feature status")
	}
	if !featureSvc.IsValidStatus("active") {
		t.Error("expected 'active' to be a valid feature status")
	}
}

func TestForLevel_Task(t *testing.T) {
	svc := newTestService()
	taskSvc := svc.ForLevel(LevelTask)

	if taskSvc.GetLevel() != LevelTask {
		t.Errorf("expected level %q, got %q", LevelTask, taskSvc.GetLevel())
	}

	// Task workflow has "todo" and "in_progress"
	if !taskSvc.IsValidStatus("todo") {
		t.Error("expected 'todo' to be a valid task status")
	}
	if !taskSvc.IsValidStatus("in_progress") {
		t.Error("expected 'in_progress' to be a valid task status")
	}
}

func TestForLevel_Isolation(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)
	taskSvc := svc.ForLevel(LevelTask)

	// Epic: "draft" -> "active" should be valid
	if !epicSvc.IsValidTransition("draft", "active") {
		t.Error("expected epic transition 'draft' -> 'active' to be valid")
	}

	// Epic: "draft" -> "in_progress" should NOT be valid (task-only transition)
	if epicSvc.IsValidTransition("draft", "in_progress") {
		t.Error("expected epic transition 'draft' -> 'in_progress' to be invalid")
	}

	// Task: "todo" -> "in_progress" should be valid
	if !taskSvc.IsValidTransition("todo", "in_progress") {
		t.Error("expected task transition 'todo' -> 'in_progress' to be valid")
	}

	// Task: "draft" -> "active" should NOT be valid (epic-only status)
	if taskSvc.IsValidStatus("draft") {
		t.Error("expected 'draft' to NOT be a valid task status (default task workflow)")
	}
}

func TestNewService_BackwardCompatible(t *testing.T) {
	// NewService with empty project root should give a task-level service
	// that works with default workflow (it will fail to find config, fallback to defaults)
	svc := NewService("")

	if svc.GetLevel() != LevelTask {
		t.Errorf("expected default level %q, got %q", LevelTask, svc.GetLevel())
	}

	// Should have task statuses
	if !svc.IsValidStatus("todo") {
		t.Error("expected 'todo' to be valid in default task service")
	}

	// GetInitialStatus should return "todo" for task level
	status := svc.GetInitialStatus()
	if string(status) != "todo" {
		t.Errorf("expected initial status 'todo', got %q", status)
	}
}

func TestGetInitialStatusString_Epic(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)

	initial := epicSvc.GetInitialStatusString()
	if initial != "draft" {
		t.Errorf("expected epic initial status 'draft', got %q", initial)
	}
}

func TestGetInitialStatusString_Feature(t *testing.T) {
	svc := newTestService()
	featureSvc := svc.ForLevel(LevelFeature)

	initial := featureSvc.GetInitialStatusString()
	if initial != "draft" {
		t.Errorf("expected feature initial status 'draft', got %q", initial)
	}
}

func TestGetInitialStatusString_Task(t *testing.T) {
	svc := newTestService()
	taskSvc := svc.ForLevel(LevelTask)

	initial := taskSvc.GetInitialStatusString()
	if initial != "todo" {
		t.Errorf("expected task initial status 'todo', got %q", initial)
	}
}

func TestGetLevel(t *testing.T) {
	svc := newTestService()

	tests := []struct {
		level    string
		expected string
	}{
		{LevelEpic, "epic"},
		{LevelFeature, "feature"},
		{LevelTask, "task"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			levelSvc := svc.ForLevel(tt.level)
			if levelSvc.GetLevel() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, levelSvc.GetLevel())
			}
		})
	}
}

func TestValidateTransition_ValidEpic(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)

	// "draft" -> "active" is valid in default epic workflow
	err := epicSvc.ValidateTransition("draft", "active")
	if err != nil {
		t.Errorf("expected valid transition, got error: %v", err)
	}
}

func TestValidateTransition_InvalidEpic(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)

	// "draft" -> "in_progress" is NOT valid in epic workflow
	err := epicSvc.ValidateTransition("draft", "in_progress")
	if err == nil {
		t.Error("expected error for invalid epic transition, got nil")
	}
}

func TestValidateTransition_ValidTask(t *testing.T) {
	svc := newTestService()
	taskSvc := svc.ForLevel(LevelTask)

	// "todo" -> "in_progress" is valid in default task workflow
	err := taskSvc.ValidateTransition("todo", "in_progress")
	if err != nil {
		t.Errorf("expected valid transition, got error: %v", err)
	}
}

func TestValidateTransition_InvalidTask(t *testing.T) {
	svc := newTestService()
	taskSvc := svc.ForLevel(LevelTask)

	// "todo" -> "completed" is NOT valid (must go through in_progress)
	err := taskSvc.ValidateTransition("todo", "completed")
	if err == nil {
		t.Error("expected error for invalid task transition, got nil")
	}
}

func TestForLevel_SharedMultiLevel(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)
	featureSvc := svc.ForLevel(LevelFeature)

	// Both should share the same multiLevel pointer
	if epicSvc.multiLevel != featureSvc.multiLevel {
		t.Error("expected ForLevel instances to share same multiLevel config")
	}
}

func TestForLevel_WithCustomWorkflow(t *testing.T) {
	customEpic := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"new":      {"planning"},
			"planning": {"active"},
			"active":   {"done"},
			"done":     {},
		},
		StatusMetadata: make(map[string]config.StatusMetadata),
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"new"},
			config.CompleteStatusKey: {"done"},
		},
	}

	multi := &config.MultiLevelWorkflow{
		Epic: customEpic,
		// Feature and Task are nil => defaults
	}

	svc := &Service{
		workflow:    multi.GetWorkflowForLevel(LevelTask),
		projectRoot: "",
		level:       LevelTask,
		multiLevel:  multi,
	}

	epicSvc := svc.ForLevel(LevelEpic)

	// Should use custom epic workflow
	if !epicSvc.IsValidStatus("new") {
		t.Error("expected 'new' to be valid in custom epic workflow")
	}
	if !epicSvc.IsValidStatus("planning") {
		t.Error("expected 'planning' to be valid in custom epic workflow")
	}
	if epicSvc.IsValidStatus("draft") {
		t.Error("expected 'draft' to NOT be valid in custom epic workflow")
	}

	// Initial status should come from custom config
	initial := epicSvc.GetInitialStatusString()
	if initial != "new" {
		t.Errorf("expected custom epic initial status 'new', got %q", initial)
	}

	// Feature service should still use defaults
	featureSvc := svc.ForLevel(LevelFeature)
	if !featureSvc.IsValidStatus("draft") {
		t.Error("expected 'draft' to be valid in default feature workflow")
	}
}

func TestGetValidTransitions_EpicLevel(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)

	transitions := epicSvc.GetValidTransitions("draft")
	if len(transitions) == 0 {
		t.Fatal("expected transitions from 'draft' in epic workflow")
	}

	// Default epic: draft -> [active, archived]
	foundActive := false
	for _, tr := range transitions {
		if tr == "active" {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("expected 'active' in transitions from 'draft', got %v", transitions)
	}
}

func TestGetTerminalStatuses_EpicLevel(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)

	terminal := epicSvc.GetTerminalStatuses()
	if len(terminal) == 0 {
		t.Fatal("expected terminal statuses in epic workflow")
	}

	// Default epic has "completed" and "archived" as complete statuses
	foundCompleted := false
	for _, s := range terminal {
		if s == "completed" {
			foundCompleted = true
		}
	}
	if !foundCompleted {
		t.Errorf("expected 'completed' in terminal statuses, got %v", terminal)
	}
}

func TestGetStatusMetadata_EpicLevel(t *testing.T) {
	svc := newTestService()
	epicSvc := svc.ForLevel(LevelEpic)

	meta := epicSvc.GetStatusMetadata("draft")
	if meta.Name != "draft" {
		t.Errorf("expected status name 'draft', got %q", meta.Name)
	}
	if meta.Color != "gray" {
		t.Errorf("expected color 'gray' for draft, got %q", meta.Color)
	}
	if meta.Phase != "planning" {
		t.Errorf("expected phase 'planning' for draft, got %q", meta.Phase)
	}
}
