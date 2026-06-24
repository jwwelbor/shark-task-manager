package workflow

import (
	"strings"
	"testing"
	"time"

	cfgtemplate "github.com/jwwelbor/shark-task-manager/internal/config/template"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestWorkflowWalk_TaskHappyPath walks through the full task workflow happy path,
// showing which statuses have OrchestratorActions and what the populated instructions look like.
func TestWorkflowWalk_TaskHappyPath(t *testing.T) {
	t.Cleanup(ClearWorkflowCache)
	configPath := findConfigPath(t)
	multi := LoadMultiLevelWorkflowOrDefault(configPath)
	taskWf := multi.GetWorkflowForLevel("task")

	if taskWf == nil {
		t.Fatal("task workflow is nil")
	}

	happyPath := traceHappyPath(taskWf, "draft")
	if len(happyPath) < 2 {
		t.Fatalf("happy path too short: %v", happyPath)
	}

	// Mock task data for template population
	mockTask := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F01-001",
		Title: "Implement JWT Token Validation",

		CreatedAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)}, Status: "draft",
		Priority: 5,
	}
	filePath := "docs/plan/E07-enhancements/E07-F01-auth/tasks/T-E07-F01-001.md"
	mockTask.FilePath = &filePath
	slug := "implement-jwt-token-validation"
	mockTask.Slug = &slug
	agentType := "developer"
	mockTask.AgentType = &agentType

	t.Logf("")
	t.Logf("========================================================================")
	t.Logf("  TASK WORKFLOW  —  %d statuses in happy path", len(happyPath))
	t.Logf("========================================================================")
	t.Logf("  Entity:  TASK   key=%s", mockTask.Key)
	t.Logf("  Title:   %q", mockTask.Title)
	t.Logf("  File:    %s", filePath)
	t.Logf("------------------------------------------------------------------------")

	actionsFound := 0
	for i, status := range happyPath {
		mockTask.Status = models.TaskStatus(status)
		placeholders := cfgtemplate.TaskPlaceholders(mockTask)
		meta, hasMeta := taskWf.GetStatusMetadata(status)

		if !hasMeta {
			t.Logf("  [TASK %2d/%-2d] %-28s  (no metadata)", i+1, len(happyPath), status)
			continue
		}

		if meta.OrchestratorAction == nil {
			t.Logf("  [TASK %2d/%-2d] %-28s  phase=%-12s  — no action (work in progress)",
				i+1, len(happyPath), status, meta.Phase)
			continue
		}

		actionsFound++
		action := meta.OrchestratorAction
		instruction := action.PopulateTemplate(placeholders)

		t.Logf("  [TASK %2d/%-2d] %-28s  phase=%-12s  ACTION: %s",
			i+1, len(happyPath), status, meta.Phase, action.Action)
		if action.AgentType != "" {
			t.Logf("               -> spawn %s  skills=%v", action.AgentType, action.Skills)
		}
		t.Logf("               -> instruction: %.200s", instruction)

		// Verify common template variables were replaced
		for _, placeholder := range []string{"{task_id}", "{title}"} {
			if strings.Contains(instruction, placeholder) {
				t.Errorf("  FAIL: %s not replaced in status %s", placeholder, status)
			}
		}

		if err := action.Validate(); err != nil {
			t.Errorf("  FAIL: action validation failed for status %s: %v", status, err)
		}
	}

	t.Logf("------------------------------------------------------------------------")
	t.Logf("  TASK SUMMARY: %d total | %d with action | %d without (in-progress states)",
		len(happyPath), actionsFound, len(happyPath)-actionsFound)
	t.Logf("========================================================================")

	lastStatus := happyPath[len(happyPath)-1]
	if lastStatus != "completed" {
		t.Errorf("happy path should end at 'completed', got %q", lastStatus)
	}
	if actionsFound == 0 {
		t.Error("expected at least one OrchestratorAction in the task workflow")
	}
}

// TestWorkflowWalk_EpicHappyPath walks through the epic workflow happy path.
func TestWorkflowWalk_EpicHappyPath(t *testing.T) {
	t.Cleanup(ClearWorkflowCache)
	configPath := findConfigPath(t)
	multi := LoadMultiLevelWorkflowOrDefault(configPath)
	epicWf := multi.GetWorkflowForLevel("epic")

	if epicWf == nil {
		t.Fatal("epic workflow is nil")
	}

	happyPath := traceHappyPath(epicWf, "draft")
	if len(happyPath) < 2 {
		t.Fatalf("epic happy path too short: %v", happyPath)
	}

	mockEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E07",
		Title: "User Management Enhancements",

		CreatedAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)}, Status: "draft",
		Priority: models.PriorityHigh,
	}
	filePath := "docs/plan/E07-enhancements/epic.md"
	mockEpic.FilePath = &filePath
	slug := "user-management-enhancements"
	mockEpic.Slug = &slug

	t.Logf("")
	t.Logf("========================================================================")
	t.Logf("  EPIC WORKFLOW  —  %d statuses in happy path", len(happyPath))
	t.Logf("========================================================================")
	t.Logf("  Entity:  EPIC   key=%s", mockEpic.Key)
	t.Logf("  Title:   %q", mockEpic.Title)
	t.Logf("  File:    %s", filePath)
	t.Logf("------------------------------------------------------------------------")

	actionsFound := 0
	for i, status := range happyPath {
		mockEpic.Status = models.EpicStatus(status)
		placeholders := cfgtemplate.EpicPlaceholders(mockEpic)
		meta, hasMeta := epicWf.GetStatusMetadata(status)

		if !hasMeta || meta.OrchestratorAction == nil {
			phase := ""
			if hasMeta {
				phase = meta.Phase
			}
			t.Logf("  [EPIC %2d/%-2d] %-28s  phase=%-12s  — no action",
				i+1, len(happyPath), status, phase)
			continue
		}

		actionsFound++
		action := meta.OrchestratorAction
		instruction := action.PopulateTemplate(placeholders)

		t.Logf("  [EPIC %2d/%-2d] %-28s  phase=%-12s  ACTION: %s",
			i+1, len(happyPath), status, meta.Phase, action.Action)
		if action.AgentType != "" {
			t.Logf("               -> spawn %s  skills=%v", action.AgentType, action.Skills)
		}
		t.Logf("               -> instruction: %.200s", instruction)

		if err := action.Validate(); err != nil {
			t.Errorf("  FAIL: action validation failed for status %s: %v", status, err)
		}
	}

	t.Logf("------------------------------------------------------------------------")
	t.Logf("  EPIC SUMMARY: %d total | %d with action | %d without",
		len(happyPath), actionsFound, len(happyPath)-actionsFound)
	t.Logf("========================================================================")
}

// TestWorkflowWalk_FeatureHappyPath walks through the feature workflow happy path.
func TestWorkflowWalk_FeatureHappyPath(t *testing.T) {
	t.Cleanup(ClearWorkflowCache)
	configPath := findConfigPath(t)
	multi := LoadMultiLevelWorkflowOrDefault(configPath)
	featureWf := multi.GetWorkflowForLevel("feature")

	if featureWf == nil {
		t.Fatal("feature workflow is nil")
	}

	happyPath := traceHappyPath(featureWf, "draft")
	if len(happyPath) < 2 {
		t.Fatalf("feature happy path too short: %v", happyPath)
	}

	mockFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01",
		Title: "User Authentication",

		CreatedAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)}, Status: "draft",
	}
	filePath := "docs/plan/E07-enhancements/E07-F01-auth/feature.md"
	mockFeature.FilePath = &filePath
	slug := "user-authentication"
	mockFeature.Slug = &slug

	t.Logf("")
	t.Logf("========================================================================")
	t.Logf("  FEATURE WORKFLOW  —  %d statuses in happy path", len(happyPath))
	t.Logf("========================================================================")
	t.Logf("  Entity:  FEATURE   key=%s", mockFeature.Key)
	t.Logf("  Title:   %q", mockFeature.Title)
	t.Logf("  File:    %s", filePath)
	t.Logf("------------------------------------------------------------------------")

	actionsFound := 0
	for i, status := range happyPath {
		mockFeature.Status = models.FeatureStatus(status)
		placeholders := cfgtemplate.FeaturePlaceholders(mockFeature)
		meta, hasMeta := featureWf.GetStatusMetadata(status)

		if !hasMeta || meta.OrchestratorAction == nil {
			phase := ""
			if hasMeta {
				phase = meta.Phase
			}
			t.Logf("  [FEAT %2d/%-2d] %-28s  phase=%-12s  — no action",
				i+1, len(happyPath), status, phase)
			continue
		}

		actionsFound++
		action := meta.OrchestratorAction
		instruction := action.PopulateTemplate(placeholders)

		t.Logf("  [FEAT %2d/%-2d] %-28s  phase=%-12s  ACTION: %s",
			i+1, len(happyPath), status, meta.Phase, action.Action)
		if action.AgentType != "" {
			t.Logf("               -> spawn %s  skills=%v", action.AgentType, action.Skills)
		}
		t.Logf("               -> instruction: %.200s", instruction)

		if err := action.Validate(); err != nil {
			t.Errorf("  FAIL: action validation failed for status %s: %v", status, err)
		}
	}

	t.Logf("------------------------------------------------------------------------")
	t.Logf("  FEATURE SUMMARY: %d total | %d with action | %d without",
		len(happyPath), actionsFound, len(happyPath)-actionsFound)
	t.Logf("========================================================================")
}

// TestWorkflowWalk_AllTaskStatusesHaveMetadata verifies every status in the
// task status_flow has corresponding metadata defined.
func TestWorkflowWalk_AllTaskStatusesHaveMetadata(t *testing.T) {
	t.Cleanup(ClearWorkflowCache)
	configPath := findConfigPath(t)
	multi := LoadMultiLevelWorkflowOrDefault(configPath)
	taskWf := multi.GetWorkflowForLevel("task")

	for status := range taskWf.StatusFlow {
		if _, found := taskWf.GetStatusMetadata(status); !found {
			t.Errorf("status %q is in status_flow but has no metadata", status)
		}
	}
}

// TestWorkflowWalk_ActionCoverage reports which statuses have actions vs not
// across all three entity types.
func TestWorkflowWalk_ActionCoverage(t *testing.T) {
	t.Cleanup(ClearWorkflowCache)
	configPath := findConfigPath(t)
	multi := LoadMultiLevelWorkflowOrDefault(configPath)

	levels := []struct {
		name string
		wf   *WorkflowConfig
	}{
		{"task", multi.GetWorkflowForLevel("task")},
		{"epic", multi.GetWorkflowForLevel("epic")},
		{"feature", multi.GetWorkflowForLevel("feature")},
	}

	for _, level := range levels {
		t.Run(level.name, func(t *testing.T) {
			withAction := 0
			withoutAction := 0
			for status, meta := range level.wf.StatusMetadata {
				if meta.OrchestratorAction != nil {
					withAction++
					if err := meta.OrchestratorAction.Validate(); err != nil {
						t.Errorf("status %q has invalid action: %v", status, err)
					}
				} else {
					withoutAction++
				}
			}
			t.Logf("%s workflow: %d statuses with actions, %d without (%d total)",
				level.name, withAction, withoutAction, withAction+withoutAction)
		})
	}
}

// --- helpers ---

// findConfigPath locates .sharkconfig.json, skipping the test if not found.
func findConfigPath(t *testing.T) string {
	t.Helper()
	// Try project root (tests run from package dir, config is at repo root)
	candidates := []string{
		"../../../.sharkconfig.json",    // from internal/config/workflow/
		"../../../../.sharkconfig.json", // fallback
		".sharkconfig.json",             // if running from root
	}
	for _, p := range candidates {
		if _, err := LoadMultiLevelWorkflow(p); err == nil {
			return p
		}
	}
	t.Skip("Skipping: .sharkconfig.json not found (test requires real config)")
	return ""
}

// traceHappyPath follows the forward transition from each status to build the
// happy path through the workflow. Stops at terminal statuses or cycles.
//
// For route-based (steps:) workflows it follows the semantic `pass` outcome:
// the derived StatusFlow targets are sorted alphabetically for determinism, so
// StatusFlow[current][0] is no longer the forward transition. For legacy
// workflows it falls back to the first listed transition (authors order the
// happy path first).
func traceHappyPath(wf *WorkflowConfig, startStatus string) []string {
	path := []string{startStatus}
	visited := map[string]bool{startStatus: true}

	current := startStatus
	for {
		var next string
		if wf.HasSteps() {
			target, ok := wf.ResolveOutcome(current, OutcomePass)
			if !ok || target == "" {
				break // terminal/parking step or no pass outcome
			}
			next = target
		} else {
			targets, ok := wf.StatusFlow[current]
			if !ok || len(targets) == 0 {
				break // terminal status
			}
			next = targets[0] // first transition = happy path
		}
		if visited[next] {
			break // cycle detected
		}
		visited[next] = true
		path = append(path, next)
		current = next
	}
	return path
}
