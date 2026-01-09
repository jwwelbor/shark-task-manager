package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestConfig creates a temporary workflow config file for testing
func createTestConfig(t *testing.T, cfg map[string]interface{}) string {
	t.Helper()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Clear cache before test
	config.ClearWorkflowCache()

	return tempDir
}

func TestNewService_ValidConfig(t *testing.T) {
	// Create config with custom workflow
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    {"draft", "ready_for_development"},
			"_complete_": {"completed", "cancelled"},
		},
		"status_flow": map[string][]string{
			"draft":                 {"ready_for_development"},
			"ready_for_development": {"in_development"},
			"in_development":        {"ready_for_review"},
			"ready_for_review":      {"completed"},
			"completed":             {},
			"cancelled":             {},
		},
		"status_metadata": map[string]interface{}{
			"draft": map[string]interface{}{
				"color":       "gray",
				"description": "Initial draft",
				"phase":       "planning",
			},
			"ready_for_development": map[string]interface{}{
				"color":       "yellow",
				"description": "Ready to implement",
				"phase":       "development",
				"agent_types": []string{"developer", "backend"},
			},
			"in_development": map[string]interface{}{
				"color":       "blue",
				"description": "Work in progress",
				"phase":       "development",
			},
			"ready_for_review": map[string]interface{}{
				"color":       "magenta",
				"description": "Awaiting code review",
				"phase":       "review",
			},
			"completed": map[string]interface{}{
				"color":       "green",
				"description": "Finished",
				"phase":       "done",
			},
			"cancelled": map[string]interface{}{
				"color":       "gray",
				"description": "Cancelled",
				"phase":       "done",
			},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	assert.NotNil(t, svc)
	assert.NotNil(t, svc.GetWorkflow())
}

func TestNewService_MissingConfig(t *testing.T) {
	// Create temp directory without config file
	tempDir := t.TempDir()
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	svc := NewService(tempDir)

	// Should fall back to default workflow
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.GetWorkflow())
}

func TestNewService_InvalidConfig(t *testing.T) {
	// Create config with invalid JSON
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte("{invalid json}"), 0644)
	require.NoError(t, err)

	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	svc := NewService(tempDir)

	// Should fall back to default workflow
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.GetWorkflow())
}

func TestService_GetInitialStatus(t *testing.T) {
	// Test with custom entry status
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    {"draft", "ready_for_development"},
			"_complete_": {"completed"},
		},
		"status_flow": map[string][]string{
			"draft":                 {"ready_for_development"},
			"ready_for_development": {"completed"},
			"completed":             {},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Should return first entry status
	assert.Equal(t, models.TaskStatus("draft"), svc.GetInitialStatus())
}

func TestService_GetInitialStatus_Fallback(t *testing.T) {
	// Test fallback when no entry statuses defined
	tempDir := t.TempDir()
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	svc := NewService(tempDir)

	// Should fall back to "todo"
	assert.Equal(t, models.TaskStatusTodo, svc.GetInitialStatus())
}

func TestService_GetEntryStatuses(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    {"draft", "backlog", "ready_for_development"},
			"_complete_": {"completed"},
		},
		"status_flow": map[string][]string{
			"draft":                 {},
			"backlog":               {},
			"ready_for_development": {},
			"completed":             {},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	statuses := svc.GetEntryStatuses()
	assert.Equal(t, []string{"draft", "backlog", "ready_for_development"}, statuses)
}

func TestService_GetTerminalStatuses(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    {"draft"},
			"_complete_": {"completed", "cancelled", "archived"},
		},
		"status_flow": map[string][]string{
			"draft":     {},
			"completed": {},
			"cancelled": {},
			"archived":  {},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	statuses := svc.GetTerminalStatuses()
	assert.Equal(t, []string{"completed", "cancelled", "archived"}, statuses)
}

func TestService_IsTerminalStatus(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    {"draft"},
			"_complete_": {"completed", "cancelled"},
		},
		"status_flow": map[string][]string{
			"draft":       {"in_progress"},
			"in_progress": {"completed"},
			"completed":   {},
			"cancelled":   {},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Terminal statuses
	assert.True(t, svc.IsTerminalStatus("completed"))
	assert.True(t, svc.IsTerminalStatus("cancelled"))
	assert.True(t, svc.IsTerminalStatus("COMPLETED")) // Case insensitive

	// Non-terminal statuses
	assert.False(t, svc.IsTerminalStatus("draft"))
	assert.False(t, svc.IsTerminalStatus("in_progress"))
}

func TestService_GetAllStatusesOrdered(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    {"draft"},
			"_complete_": {"completed"},
		},
		"status_flow": map[string][]string{
			"draft":                 {"ready_for_development"},
			"ready_for_development": {"in_development"},
			"in_development":        {"ready_for_review"},
			"ready_for_review":      {"ready_for_qa"},
			"ready_for_qa":          {"completed"},
			"completed":             {},
			"blocked":               {"draft"},
		},
		"status_metadata": map[string]interface{}{
			"draft":                 map[string]interface{}{"phase": "planning"},
			"ready_for_development": map[string]interface{}{"phase": "development"},
			"in_development":        map[string]interface{}{"phase": "development"},
			"ready_for_review":      map[string]interface{}{"phase": "review"},
			"ready_for_qa":          map[string]interface{}{"phase": "qa"},
			"completed":             map[string]interface{}{"phase": "done"},
			"blocked":               map[string]interface{}{"phase": "any"},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	statuses := svc.GetAllStatusesOrdered()

	// Verify ordering by phase: planning < development < review < qa < done < any
	phaseMap := map[string]string{
		"draft":                 "planning",
		"ready_for_development": "development",
		"in_development":        "development",
		"ready_for_review":      "review",
		"ready_for_qa":          "qa",
		"completed":             "done",
		"blocked":               "any",
	}

	phaseOrder := map[string]int{
		"planning":    0,
		"development": 1,
		"review":      2,
		"qa":          3,
		"done":        5,
		"any":         6,
	}

	// Verify statuses are ordered correctly
	for i := 0; i < len(statuses)-1; i++ {
		phase1 := phaseMap[statuses[i]]
		phase2 := phaseMap[statuses[i+1]]
		order1 := phaseOrder[phase1]
		order2 := phaseOrder[phase2]

		// Phase order should be non-decreasing
		assert.LessOrEqual(t, order1, order2,
			"Status %s (phase %s) should come before or with %s (phase %s)",
			statuses[i], phase1, statuses[i+1], phase2)
	}
}

func TestService_GetStatusMetadata(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"in_development": {},
		},
		"status_metadata": map[string]interface{}{
			"in_development": map[string]interface{}{
				"color":       "blue",
				"description": "Work in progress",
				"phase":       "development",
				"agent_types": []string{"developer", "backend"},
			},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	info := svc.GetStatusMetadata("in_development")

	assert.Equal(t, "in_development", info.Name)
	assert.Equal(t, "blue", info.Color)
	assert.Equal(t, "Work in progress", info.Description)
	assert.Equal(t, "development", info.Phase)
	assert.Equal(t, []string{"developer", "backend"}, info.AgentTypes)
}

func TestService_GetStatusMetadata_NotFound(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"draft": {},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	info := svc.GetStatusMetadata("unknown_status")

	// Should return StatusInfo with just the name
	assert.Equal(t, "unknown_status", info.Name)
	assert.Empty(t, info.Color)
	assert.Empty(t, info.Description)
}

func TestService_GetStatusesByPhase(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"draft":          {},
			"in_refinement":  {},
			"in_development": {},
			"in_review":      {},
			"completed":      {},
		},
		"status_metadata": map[string]interface{}{
			"draft":          map[string]interface{}{"phase": "planning"},
			"in_refinement":  map[string]interface{}{"phase": "planning"},
			"in_development": map[string]interface{}{"phase": "development"},
			"in_review":      map[string]interface{}{"phase": "review"},
			"completed":      map[string]interface{}{"phase": "done"},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	planningStatuses := svc.GetStatusesByPhase("planning")
	assert.Contains(t, planningStatuses, "draft")
	assert.Contains(t, planningStatuses, "in_refinement")
	assert.Len(t, planningStatuses, 2)

	devStatuses := svc.GetStatusesByPhase("development")
	assert.Contains(t, devStatuses, "in_development")
	assert.Len(t, devStatuses, 1)
}

func TestService_GetValidTransitions(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"draft":       {"in_progress", "cancelled"},
			"in_progress": {"ready_for_review", "blocked"},
			"completed":   {}, // Terminal - no transitions
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Has transitions
	transitions := svc.GetValidTransitions("draft")
	assert.Equal(t, []string{"in_progress", "cancelled"}, transitions)

	// Terminal status - no transitions
	transitions = svc.GetValidTransitions("completed")
	assert.Empty(t, transitions)

	// Case insensitive
	transitions = svc.GetValidTransitions("DRAFT")
	assert.Equal(t, []string{"in_progress", "cancelled"}, transitions)
}

func TestService_GetTransitionInfo(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"in_development": {"ready_for_review", "blocked"},
		},
		"status_metadata": map[string]interface{}{
			"ready_for_review": map[string]interface{}{
				"color":       "magenta",
				"description": "Awaiting code review",
				"phase":       "review",
				"agent_types": []string{"tech-lead"},
			},
			"blocked": map[string]interface{}{
				"color":       "red",
				"description": "Blocked by dependency",
				"phase":       "any",
			},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	infos := svc.GetTransitionInfo("in_development")

	assert.Len(t, infos, 2)

	// First transition
	assert.Equal(t, "ready_for_review", infos[0].TargetStatus)
	assert.Equal(t, "magenta", infos[0].Color)
	assert.Equal(t, "Awaiting code review", infos[0].Description)
	assert.Equal(t, "review", infos[0].Phase)
	assert.Equal(t, []string{"tech-lead"}, infos[0].AgentTypes)

	// Second transition
	assert.Equal(t, "blocked", infos[1].TargetStatus)
	assert.Equal(t, "red", infos[1].Color)
}

func TestService_IsValidTransition(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"draft":       {"in_progress"},
			"in_progress": {"completed"},
			"completed":   {},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Valid transitions
	assert.True(t, svc.IsValidTransition("draft", "in_progress"))
	assert.True(t, svc.IsValidTransition("in_progress", "completed"))

	// Case insensitive
	assert.True(t, svc.IsValidTransition("DRAFT", "IN_PROGRESS"))

	// Invalid transitions
	assert.False(t, svc.IsValidTransition("draft", "completed")) // Must go through in_progress
	assert.False(t, svc.IsValidTransition("completed", "draft")) // Terminal status
}

func TestService_IsValidStatus(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"draft":       {"in_progress"},
			"in_progress": {"completed"},
			"completed":   {},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Valid statuses
	assert.True(t, svc.IsValidStatus("draft"))
	assert.True(t, svc.IsValidStatus("in_progress"))
	assert.True(t, svc.IsValidStatus("completed"))

	// Case insensitive
	assert.True(t, svc.IsValidStatus("DRAFT"))
	assert.True(t, svc.IsValidStatus("Draft"))

	// Invalid status
	assert.False(t, svc.IsValidStatus("unknown"))
	assert.False(t, svc.IsValidStatus(""))
}

func TestService_NormalizeStatus(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"ready_for_development": {},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Normalizes to canonical case
	assert.Equal(t, "ready_for_development", svc.NormalizeStatus("READY_FOR_DEVELOPMENT"))
	assert.Equal(t, "ready_for_development", svc.NormalizeStatus("Ready_For_Development"))
	assert.Equal(t, "ready_for_development", svc.NormalizeStatus("ready_for_development"))

	// Unknown status returned unchanged
	assert.Equal(t, "unknown", svc.NormalizeStatus("unknown"))
}

func TestService_GetPhases(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"draft":        {},
			"in_dev":       {},
			"in_review":    {},
			"in_qa":        {},
			"completed":    {},
			"blocked":      {},
			"custom_phase": {},
		},
		"status_metadata": map[string]interface{}{
			"draft":        map[string]interface{}{"phase": "planning"},
			"in_dev":       map[string]interface{}{"phase": "development"},
			"in_review":    map[string]interface{}{"phase": "review"},
			"in_qa":        map[string]interface{}{"phase": "qa"},
			"completed":    map[string]interface{}{"phase": "done"},
			"blocked":      map[string]interface{}{"phase": "any"},
			"custom_phase": map[string]interface{}{"phase": "custom"},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	phases := svc.GetPhases()

	// Standard phases should be in order
	assert.Contains(t, phases, "planning")
	assert.Contains(t, phases, "development")
	assert.Contains(t, phases, "review")
	assert.Contains(t, phases, "qa")
	assert.Contains(t, phases, "done")
	assert.Contains(t, phases, "any")

	// Custom phase should be included
	assert.Contains(t, phases, "custom")

	// Verify standard phases come first and in order
	planningIdx := indexOf(phases, "planning")
	devIdx := indexOf(phases, "development")
	reviewIdx := indexOf(phases, "review")
	qaIdx := indexOf(phases, "qa")
	doneIdx := indexOf(phases, "done")
	anyIdx := indexOf(phases, "any")

	assert.Less(t, planningIdx, devIdx)
	assert.Less(t, devIdx, reviewIdx)
	assert.Less(t, reviewIdx, qaIdx)
	assert.Less(t, qaIdx, doneIdx)
	assert.Less(t, doneIdx, anyIdx)
}

// Helper function
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func TestService_FormatStatusForDisplay(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"in_development": {},
		},
		"status_metadata": map[string]interface{}{
			"in_development": map[string]interface{}{
				"color":       "blue",
				"description": "Work in progress",
				"phase":       "development",
			},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// With color enabled
	formatted := svc.FormatStatusForDisplay("in_development", true)
	assert.Equal(t, "in_development", formatted.Status)
	assert.Equal(t, "Work in progress", formatted.Description)
	assert.Equal(t, "development", formatted.Phase)
	assert.Equal(t, "blue", formatted.ColorName)
	assert.Contains(t, formatted.Colored, "\033[34m") // Blue color code
	assert.Contains(t, formatted.Colored, "in_development")
	assert.Contains(t, formatted.Colored, "\033[0m") // Reset code

	// With color disabled
	formatted = svc.FormatStatusForDisplay("in_development", false)
	assert.Equal(t, "in_development", formatted.Colored) // No color codes
}

func TestService_FormatStatusForDisplay_UnknownStatus(t *testing.T) {
	tempDir := t.TempDir()
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	svc := NewService(tempDir)

	// Unknown status should return basic info
	formatted := svc.FormatStatusForDisplay("unknown_status", true)
	assert.Equal(t, "unknown_status", formatted.Status)
	assert.Equal(t, "unknown_status", formatted.Colored) // No color applied
	assert.Empty(t, formatted.ColorName)
}

func TestService_FormatStatusCount(t *testing.T) {
	tempDir := t.TempDir()
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	svc := NewService(tempDir)

	// With color
	sc := StatusCount{
		Status: "in_development",
		Count:  5,
		Color:  "blue",
		Phase:  "development",
	}
	formatted := svc.FormatStatusCount(sc, true)
	assert.Contains(t, formatted, "\033[34m") // Blue color code
	assert.Contains(t, formatted, "in_development")

	// Without color
	formatted = svc.FormatStatusCount(sc, false)
	assert.Equal(t, "in_development", formatted)

	// With empty color
	sc.Color = ""
	formatted = svc.FormatStatusCount(sc, true)
	assert.Equal(t, "in_development", formatted) // No color applied
}

func TestService_GetColorForStatus(t *testing.T) {
	projectRoot := createTestConfig(t, map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string][]string{
			"in_development": {},
			"draft":          {},
		},
		"status_metadata": map[string]interface{}{
			"in_development": map[string]interface{}{
				"color": "blue",
			},
			"draft": map[string]interface{}{
				// No color defined
			},
		},
	})

	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Has color
	assert.Equal(t, "blue", svc.GetColorForStatus("in_development"))

	// No color defined
	assert.Empty(t, svc.GetColorForStatus("draft"))

	// Unknown status
	assert.Empty(t, svc.GetColorForStatus("unknown"))
}

func TestColorizeStatus(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		colorName string
		wantCode  string
	}{
		{"red", "blocked", "red", "\033[31m"},
		{"green", "completed", "green", "\033[32m"},
		{"yellow", "in_progress", "yellow", "\033[33m"},
		{"blue", "in_development", "blue", "\033[34m"},
		{"magenta", "ready_for_review", "magenta", "\033[35m"},
		{"cyan", "ready_for_qa", "cyan", "\033[36m"},
		{"gray", "draft", "gray", "\033[90m"},
		{"orange", "warning", "orange", "\033[38;5;208m"},
		{"purple", "special", "purple", "\033[38;5;141m"},
		{"unknown color", "status", "unknown", ""}, // No color code
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorizeStatus(tt.status, tt.colorName)
			if tt.wantCode == "" {
				// Should return status unchanged
				assert.Equal(t, tt.status, result)
			} else {
				// Should contain color code and reset
				assert.Contains(t, result, tt.wantCode)
				assert.Contains(t, result, tt.status)
				assert.Contains(t, result, "\033[0m")
			}
		})
	}
}
