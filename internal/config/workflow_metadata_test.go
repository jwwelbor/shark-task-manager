package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoadWorkflowMetadata tests that metadata is loaded correctly from config
func TestLoadWorkflowMetadata(t *testing.T) {
	// Create temp config with metadata
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configJSON := `{
  "status_flow_version": "1.0",
  "status_flow": {
    "todo": ["in_progress"],
    "in_progress": ["completed"],
    "completed": []
  },
  "status_metadata": {
    "todo": {
      "color": "gray",
      "description": "Task is ready to be started",
      "phase": "planning",
      "agent_types": ["business-analyst", "project-manager"]
    },
    "in_progress": {
      "color": "blue",
      "description": "Task is actively being worked on",
      "phase": "development",
      "agent_types": ["developer", "backend", "frontend"]
    },
    "completed": {
      "color": "green",
      "description": "Task approved and merged",
      "phase": "done",
      "agent_types": []
    }
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}`

	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	assert.NoError(t, err)

	// Clear cache before testing
	ClearWorkflowCache()

	// Load workflow config
	workflow, err := LoadWorkflowConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)

	// Verify metadata loaded correctly
	assert.NotNil(t, workflow.StatusMetadata)
	assert.Equal(t, 3, len(workflow.StatusMetadata))

	// Check todo metadata
	todoMeta := workflow.StatusMetadata["todo"]
	assert.Equal(t, "gray", todoMeta.Color)
	assert.Equal(t, "Task is ready to be started", todoMeta.Description)
	assert.Equal(t, "planning", todoMeta.Phase)
	assert.Equal(t, []string{"business-analyst", "project-manager"}, todoMeta.AgentTypes)

	// Check in_progress metadata
	inProgressMeta := workflow.StatusMetadata["in_progress"]
	assert.Equal(t, "blue", inProgressMeta.Color)
	assert.Equal(t, "development", inProgressMeta.Phase)
	assert.Equal(t, 3, len(inProgressMeta.AgentTypes))

	// Check completed metadata
	completedMeta := workflow.StatusMetadata["completed"]
	assert.Equal(t, "green", completedMeta.Color)
	assert.Equal(t, "done", completedMeta.Phase)
	assert.Equal(t, 0, len(completedMeta.AgentTypes))
}

// TestLoadWorkflowMetadata_MissingFields tests graceful defaults for missing metadata
func TestLoadWorkflowMetadata_MissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Config with partial metadata (missing fields)
	configJSON := `{
  "status_flow_version": "1.0",
  "status_flow": {
    "todo": ["completed"],
    "completed": []
  },
  "status_metadata": {
    "todo": {
      "color": "gray"
    },
    "completed": {}
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}`

	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	assert.NoError(t, err)

	ClearWorkflowCache()

	workflow, err := LoadWorkflowConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)

	// Verify partial metadata loaded without errors
	todoMeta := workflow.StatusMetadata["todo"]
	assert.Equal(t, "gray", todoMeta.Color)
	assert.Equal(t, "", todoMeta.Description) // Missing field defaults to empty
	assert.Equal(t, "", todoMeta.Phase)
	assert.Nil(t, todoMeta.AgentTypes) // Missing field defaults to nil

	completedMeta := workflow.StatusMetadata["completed"]
	assert.Equal(t, "", completedMeta.Color)
	assert.Equal(t, "", completedMeta.Description)
	assert.Equal(t, "", completedMeta.Phase)
	assert.Nil(t, completedMeta.AgentTypes)
}

// TestLoadWorkflowMetadata_NoMetadataSection tests that missing metadata section is handled
func TestLoadWorkflowMetadata_NoMetadataSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Config without status_metadata section
	configJSON := `{
  "status_flow_version": "1.0",
  "status_flow": {
    "todo": ["completed"],
    "completed": []
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}`

	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	assert.NoError(t, err)

	ClearWorkflowCache()

	workflow, err := LoadWorkflowConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)

	// Verify empty metadata map created
	assert.NotNil(t, workflow.StatusMetadata)
	assert.Equal(t, 0, len(workflow.StatusMetadata))
}

// TestGetStatusMetadata tests helper for getting metadata for a status
func TestGetStatusMetadata(t *testing.T) {
	workflow := &WorkflowConfig{
		StatusMetadata: map[string]StatusMetadata{
			"todo": {
				Color:       "gray",
				Description: "Ready to start",
				Phase:       "planning",
				AgentTypes:  []string{"developer"},
			},
		},
	}

	// Test existing status
	meta, found := workflow.GetStatusMetadata("todo")
	assert.True(t, found)
	assert.Equal(t, "gray", meta.Color)

	// Test non-existent status - should return empty metadata
	meta, found = workflow.GetStatusMetadata("nonexistent")
	assert.False(t, found)
	assert.Equal(t, "", meta.Color)
	assert.Equal(t, "", meta.Description)
}

// TestGetStatusesByAgentType tests filtering statuses by agent type
func TestGetStatusesByAgentType(t *testing.T) {
	workflow := &WorkflowConfig{
		StatusMetadata: map[string]StatusMetadata{
			"todo": {
				AgentTypes: []string{"business-analyst", "developer"},
			},
			"in_progress": {
				AgentTypes: []string{"developer", "backend"},
			},
			"ready_for_review": {
				AgentTypes: []string{"qa", "tech-lead"},
			},
			"completed": {
				AgentTypes: []string{},
			},
		},
	}

	// Test developer agent
	devStatuses := workflow.GetStatusesByAgentType("developer")
	assert.Equal(t, 2, len(devStatuses))
	assert.Contains(t, devStatuses, "todo")
	assert.Contains(t, devStatuses, "in_progress")

	// Test QA agent
	qaStatuses := workflow.GetStatusesByAgentType("qa")
	assert.Equal(t, 1, len(qaStatuses))
	assert.Contains(t, qaStatuses, "ready_for_review")

	// Test unknown agent type
	unknownStatuses := workflow.GetStatusesByAgentType("unknown")
	assert.Equal(t, 0, len(unknownStatuses))
}

// TestGetStatusesByPhase tests filtering statuses by phase
func TestGetStatusesByPhase(t *testing.T) {
	workflow := &WorkflowConfig{
		StatusMetadata: map[string]StatusMetadata{
			"todo": {
				Phase: "planning",
			},
			"in_progress": {
				Phase: "development",
			},
			"code_review": {
				Phase: "development",
			},
			"ready_for_review": {
				Phase: "review",
			},
			"completed": {
				Phase: "done",
			},
		},
	}

	// Test development phase (multiple statuses)
	devStatuses := workflow.GetStatusesByPhase("development")
	assert.Equal(t, 2, len(devStatuses))
	assert.Contains(t, devStatuses, "in_progress")
	assert.Contains(t, devStatuses, "code_review")

	// Test planning phase (single status)
	planningStatuses := workflow.GetStatusesByPhase("planning")
	assert.Equal(t, 1, len(planningStatuses))
	assert.Contains(t, planningStatuses, "todo")

	// Test unknown phase
	unknownStatuses := workflow.GetStatusesByPhase("unknown")
	assert.Equal(t, 0, len(unknownStatuses))
}
