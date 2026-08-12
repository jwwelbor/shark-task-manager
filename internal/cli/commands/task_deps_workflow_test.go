package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintBlockedByUsesConfiguredWorkflowStatuses(t *testing.T) {
	setupDependencyDisplayWorkflow(t)

	output, err := captureStdoutForTest(t, func() error {
		return printBlockedBy("T-001", "Blocked task", []services.RelationshipWithTask{
			{TaskKey: "T-002", TaskTitle: "Implementation", TaskStatus: "building"},
			{TaskKey: "T-003", TaskTitle: "Release", TaskStatus: "shipped"},
			{TaskKey: "T-004", TaskTitle: "External dependency", TaskStatus: "waiting"},
		})
	})
	require.NoError(t, err)

	assert.Contains(t, output, "• T-002: Implementation")
	assert.Contains(t, output, "✓ T-003: Release")
	assert.Contains(t, output, "✗ T-004: External dependency")
	assert.Contains(t, output, "Legend: ○ queued | • building | ⊙ verifying | ✓ shipped | ✗ waiting")
	assert.NotContains(t, output, "completed")
	assert.NotContains(t, output, "in_progress")
}

func TestDependencyDisplaysUseConfiguredWorkflowStatuses(t *testing.T) {
	setupDependencyDisplayWorkflow(t)

	depsOutput, err := captureStdoutForTest(t, func() error {
		return printTaskDeps("T-001", "Release task", []services.RelationshipWithTask{
			{Direction: "outgoing", RelationshipType: models.RelDependsOn, TaskKey: "T-002", TaskTitle: "Implementation", TaskStatus: "building"},
			{Direction: "outgoing", RelationshipType: models.RelDependsOn, TaskKey: "T-003", TaskTitle: "Release", TaskStatus: "shipped"},
			{Direction: "outgoing", RelationshipType: models.RelDependsOn, TaskKey: "T-004", TaskTitle: "External dependency", TaskStatus: "waiting"},
		})
	})
	require.NoError(t, err)
	assert.Contains(t, depsOutput, "• T-002: Implementation")
	assert.Contains(t, depsOutput, "✓ T-003: Release")
	assert.Contains(t, depsOutput, "✗ T-004: External dependency")
	assert.Contains(t, depsOutput, "Legend: ○ queued | • building | ⊙ verifying | ✓ shipped | ✗ waiting")
	assert.NotContains(t, depsOutput, "in_progress")

	treeOutput, err := captureStdoutForTest(t, func() error {
		return printDepsTree(context.Background(), &models.Task{
			BaseEntity: models.BaseEntity{Key: "T-001", Title: "Release task"},
			Status:     models.TaskStatus("shipped"),
		}, nil, nil, false, false, 1)
	})
	require.NoError(t, err)
	assert.Contains(t, treeOutput, "✓ T-001: Release task")
	assert.Contains(t, treeOutput, "Legend: ○ queued | • building | ⊙ verifying | ✓ shipped | ✗ waiting")
	assert.NotContains(t, treeOutput, "completed")

	blocksOutput, err := captureStdoutForTest(t, func() error {
		return printBlocks("T-001", "Release task", "shipped", []services.RelationshipWithTask{
			{TaskKey: "T-002", TaskTitle: "Implementation", TaskStatus: "building"},
			{TaskKey: "T-003", TaskTitle: "Release", TaskStatus: "shipped"},
			{TaskKey: "T-004", TaskTitle: "External dependency", TaskStatus: "waiting"},
		})
	})
	require.NoError(t, err)
	assert.Contains(t, blocksOutput, "• T-002: Implementation (unblocked)")
	assert.Contains(t, blocksOutput, "✓ T-003: Release (unblocked)")
	assert.Contains(t, blocksOutput, "✗ T-004: External dependency (unblocked)")
	assert.Contains(t, blocksOutput, "Legend: ○ queued | • building | ⊙ verifying | ✓ shipped | ✗ waiting")
	assert.Contains(t, blocksOutput, "This task is terminal - all downstream tasks are unblocked.")
	assert.NotContains(t, blocksOutput, "This task is completed")
	assert.NotContains(t, blocksOutput, "in_progress")
}

func setupDependencyDisplayWorkflow(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	configJSON := `{
		"status_flow": {
			"queued": ["building", "waiting"],
			"building": ["verifying", "waiting"],
			"verifying": ["shipped"],
			"waiting": ["queued"],
			"shipped": []
		},
		"status_metadata": {
			"queued": {"phase": "planning"},
			"building": {"phase": "development"},
			"verifying": {"phase": "review"},
			"waiting": {"phase": "blocked"},
			"shipped": {"phase": "done"}
		},
		"special_statuses": {"_start_": ["queued"], "_complete_": ["shipped"]}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(root, ".sharkconfig.json"), []byte(configJSON), 0644))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(root))
	config.ClearWorkflowCache()
	cli.ResetWorkflowService()
	t.Cleanup(func() {
		cli.ResetWorkflowService()
		config.ClearWorkflowCache()
		require.NoError(t, os.Chdir(originalWD))
	})
}
