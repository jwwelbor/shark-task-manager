package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for workflow-aware feature display

// TestWorkflowService_Integration verifies WorkflowService loads correctly
func TestWorkflowService_Integration(t *testing.T) {
	// Clear the global workflow cache before test
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	// Create temp directory with workflow config
	// Config format matches actual .sharkconfig.json format
	tempDir := t.TempDir()
	configContent := `{
		"status_flow": {
			"todo": ["in_progress"],
			"in_progress": ["completed", "blocked"],
			"blocked": ["in_progress"],
			"completed": []
		},
		"status_metadata": {
			"todo": {"color": "gray", "phase": "planning", "description": "Not started"},
			"in_progress": {"color": "blue", "phase": "development", "description": "In progress"},
			"blocked": {"color": "red", "phase": "any", "description": "Blocked"},
			"completed": {"color": "green", "phase": "done", "description": "Completed"}
		},
		"special_statuses": {
			"_start_": ["todo"],
			"_complete_": ["completed"]
		}
	}`

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create WorkflowService
	svc := workflow.NewService(tempDir)
	require.NotNil(t, svc)

	// Test status ordering
	statuses := svc.GetAllStatusesOrdered()
	assert.NotEmpty(t, statuses)
	assert.Contains(t, statuses, "todo")
	assert.Contains(t, statuses, "completed")

	// Test status metadata
	meta := svc.GetStatusMetadata("in_progress")
	assert.Equal(t, "blue", meta.Color)
	assert.Equal(t, "development", meta.Phase)
	assert.Equal(t, "In progress", meta.Description)

	// Test format for display with color
	formatted := svc.FormatStatusForDisplay("in_progress", true)
	assert.Equal(t, "in_progress", formatted.Status)
	assert.Contains(t, formatted.Colored, "\033[34m") // Blue ANSI code
	assert.Equal(t, "development", formatted.Phase)

	// Test format for display without color
	formattedNoColor := svc.FormatStatusForDisplay("in_progress", false)
	assert.Equal(t, "in_progress", formattedNoColor.Colored) // No ANSI codes
}

// TestWorkflowService_DefaultConfig verifies default config is used when no config file
func TestWorkflowService_DefaultConfig(t *testing.T) {
	// Clear the global workflow cache before test
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	// Use empty temp directory (no config file)
	tempDir := t.TempDir()

	// Create WorkflowService - should fall back to defaults
	svc := workflow.NewService(tempDir)
	require.NotNil(t, svc)

	// Should have default statuses
	statuses := svc.GetAllStatuses()
	assert.NotEmpty(t, statuses)

	// Should have default initial status
	initial := svc.GetInitialStatus()
	assert.NotEmpty(t, string(initial))
}

// TestStatusCount_Formatting verifies StatusCount formatting
func TestStatusCount_Formatting(t *testing.T) {
	// Clear the global workflow cache before test
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	// Create temp directory with workflow config
	// Config format matches actual .sharkconfig.json format
	tempDir := t.TempDir()
	configContent := `{
		"status_flow": {
			"todo": ["in_progress"],
			"in_progress": ["completed"],
			"completed": []
		},
		"status_metadata": {
			"todo": {"color": "gray", "phase": "planning"},
			"in_progress": {"color": "yellow", "phase": "development"},
			"completed": {"color": "green", "phase": "done"}
		}
	}`

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	svc := workflow.NewService(tempDir)

	// Create StatusCount
	sc := workflow.StatusCount{
		Status: "in_progress",
		Count:  5,
		Phase:  "development",
		Color:  "yellow",
	}

	// Test formatting with color
	formatted := svc.FormatStatusCount(sc, true)
	assert.Contains(t, formatted, "\033[33m") // Yellow ANSI code
	assert.Contains(t, formatted, "in_progress")

	// Test formatting without color
	formattedNoColor := svc.FormatStatusCount(sc, false)
	assert.Equal(t, "in_progress", formattedNoColor)
}

// TestGetColorForStatus verifies color retrieval
func TestGetColorForStatus(t *testing.T) {
	// Clear the global workflow cache before test
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	// Create temp directory with workflow config
	// Config format matches actual .sharkconfig.json format
	tempDir := t.TempDir()
	configContent := `{
		"status_flow": {
			"draft": ["ready"],
			"ready": ["done"],
			"done": []
		},
		"status_metadata": {
			"draft": {"color": "gray", "phase": "planning"},
			"ready": {"color": "cyan", "phase": "development"},
			"done": {"color": "green", "phase": "done"}
		}
	}`

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	svc := workflow.NewService(tempDir)

	// Test color retrieval for known statuses
	assert.Equal(t, "gray", svc.GetColorForStatus("draft"))
	assert.Equal(t, "cyan", svc.GetColorForStatus("ready"))
	assert.Equal(t, "green", svc.GetColorForStatus("done"))

	// Test color retrieval for unknown status
	assert.Empty(t, svc.GetColorForStatus("unknown"))
}

// TestPhaseOrdering verifies statuses are ordered by phase
func TestPhaseOrdering(t *testing.T) {
	// Clear the global workflow cache before test
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	// Create temp directory with workflow config
	// Config format matches actual .sharkconfig.json format
	tempDir := t.TempDir()
	configContent := `{
		"status_flow": {
			"draft": ["ready_for_dev"],
			"ready_for_dev": ["in_progress"],
			"in_progress": ["in_review"],
			"in_review": ["in_qa"],
			"in_qa": ["done"],
			"done": []
		},
		"status_metadata": {
			"draft": {"phase": "planning"},
			"ready_for_dev": {"phase": "planning"},
			"in_progress": {"phase": "development"},
			"in_review": {"phase": "review"},
			"in_qa": {"phase": "qa"},
			"done": {"phase": "done"}
		}
	}`

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	svc := workflow.NewService(tempDir)

	// Get ordered statuses
	ordered := svc.GetAllStatusesOrdered()
	require.NotEmpty(t, ordered)

	// Verify phases are in correct order
	// We expect: planning statuses, then development, then review, then qa, then done
	phaseOrder := map[string]int{"planning": 0, "development": 1, "review": 2, "qa": 3, "done": 5}

	lastPhaseOrder := -1
	for _, status := range ordered {
		meta := svc.GetStatusMetadata(status)
		currentPhaseOrder, ok := phaseOrder[meta.Phase]
		if ok {
			// Phase order should never decrease
			assert.GreaterOrEqual(t, currentPhaseOrder, lastPhaseOrder,
				"Status %s (phase %s) should not come before previous phase (order %d vs %d)",
				status, meta.Phase, currentPhaseOrder, lastPhaseOrder)
			lastPhaseOrder = currentPhaseOrder
		}
	}
}
