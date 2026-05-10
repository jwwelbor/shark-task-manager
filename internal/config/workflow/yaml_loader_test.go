package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// E2 — YAML workflow loader tests
// ============================================================================

const taskYAML = `status_flow_version: "1.0"
status_flow:
  todo:
    - in_development
    - blocked
  in_development:
    - ready_for_code_review
    - blocked
  ready_for_code_review:
    - in_code_review
  in_code_review:
    - ready_for_qa
    - ready_for_development
  ready_for_qa:
    - in_qa
  in_qa:
    - ready_for_approval
    - ready_for_development
  ready_for_approval:
    - in_approval
  in_approval:
    - completed
    - ready_for_development
  completed: []
  blocked:
    - todo
    - in_development
status_metadata:
  todo:
    color: gray
    description: Task ready to start
    phase: planning
    agent_types:
      - developer
  in_development:
    color: blue
    description: Active development
    phase: development
    agent_types:
      - developer
      - backend
      - frontend
    progress_weight: 0.5
special_statuses:
  _start_:
    - todo
  _complete_:
    - completed
require_rejection_reason: true
`

const epicYAML = `status_flow_version: "1.0"
status_flow:
  draft:
    - active
  active:
    - completed
  completed: []
status_metadata:
  draft:
    color: gray
    description: Epic created, not yet started
    phase: planning
  active:
    color: blue
    description: Epic in active development
    phase: development
  completed:
    color: green
    description: Epic completed
    phase: done
    progress_weight: 1.0
special_statuses:
  _start_:
    - draft
  _complete_:
    - completed
`

func writeYAMLFixture(t *testing.T, dataDir, filename, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, YAMLWorkflowDir), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, YAMLWorkflowDir, filename), []byte(content), 0644))
}

func TestLoadMultiLevelWorkflowFromYAML_EmptyDataDir(t *testing.T) {
	mlw, err := LoadMultiLevelWorkflowFromYAML("")
	require.NoError(t, err)
	assert.NotNil(t, mlw)
	assert.Empty(t, mlw.Sources)
}

func TestLoadMultiLevelWorkflowFromYAML_NoWorkflowDir(t *testing.T) {
	dataDir := t.TempDir()
	mlw, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.NoError(t, err)
	assert.NotNil(t, mlw)
	assert.Empty(t, mlw.Sources)
}

func TestLoadMultiLevelWorkflowFromYAML_LoadsTask(t *testing.T) {
	dataDir := t.TempDir()
	writeYAMLFixture(t, dataDir, "task.yaml", taskYAML)

	mlw, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.NoError(t, err)
	require.NotNil(t, mlw.Task)
	assert.Equal(t, "1.0", mlw.Task.Version)

	// Status flow round-trip
	assert.ElementsMatch(t,
		[]string{"in_development", "blocked"},
		mlw.Task.StatusFlow["todo"],
	)
	assert.Empty(t, mlw.Task.StatusFlow["completed"], "terminal status has empty next list")

	// Metadata round-trip
	require.Contains(t, mlw.Task.StatusMetadata, "in_development")
	dev := mlw.Task.StatusMetadata["in_development"]
	assert.Equal(t, "blue", dev.Color)
	assert.Equal(t, "development", dev.Phase)
	assert.ElementsMatch(t, []string{"developer", "backend", "frontend"}, dev.AgentTypes)
	assert.InDelta(t, 0.5, dev.ProgressWeight, 0.001)

	// Special statuses
	assert.ElementsMatch(t, []string{"todo"}, mlw.Task.SpecialStatuses["_start_"])
	assert.ElementsMatch(t, []string{"completed"}, mlw.Task.SpecialStatuses["_complete_"])

	assert.True(t, mlw.Task.RequireRejectionReason)

	// Source recorded
	assert.Equal(t, filepath.Join(dataDir, YAMLWorkflowDir, "task.yaml"), mlw.Sources["task"])

	// Other slots remain nil
	assert.Nil(t, mlw.Epic)
	assert.Nil(t, mlw.Feature)
}

func TestLoadMultiLevelWorkflowFromYAML_LoadsMultiple(t *testing.T) {
	dataDir := t.TempDir()
	writeYAMLFixture(t, dataDir, "task.yaml", taskYAML)
	writeYAMLFixture(t, dataDir, "epic.yaml", epicYAML)

	mlw, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.NoError(t, err)
	require.NotNil(t, mlw.Task)
	require.NotNil(t, mlw.Epic)
	assert.Nil(t, mlw.Feature)
	assert.ElementsMatch(t, []string{"draft"}, mlw.Epic.SpecialStatuses["_start_"])
}

func TestLoadMultiLevelWorkflowFromYAML_OverrideWins(t *testing.T) {
	dataDir := t.TempDir()
	writeYAMLFixture(t, dataDir, "task.yaml", taskYAML)

	// Override at <dataDir>/overrides/workflow/task.yaml replaces the default.
	overridePath := filepath.Join(dataDir, "overrides", YAMLWorkflowDir, "task.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(overridePath), 0755))
	overrideYAML := `status_flow_version: "2.0"
status_flow:
  custom_start:
    - completed
  completed: []
special_statuses:
  _start_:
    - custom_start
  _complete_:
    - completed
`
	require.NoError(t, os.WriteFile(overridePath, []byte(overrideYAML), 0644))

	mlw, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.NoError(t, err)
	require.NotNil(t, mlw.Task)
	assert.Equal(t, "2.0", mlw.Task.Version, "override file should fully replace the default")
	assert.Contains(t, mlw.Task.StatusFlow, "custom_start")
	assert.NotContains(t, mlw.Task.StatusFlow, "in_development", "default status flow must not bleed through")
	assert.Equal(t, overridePath, mlw.Sources["task"])
}

func TestLoadMultiLevelWorkflowFromYAML_TechDebtKebabCase(t *testing.T) {
	dataDir := t.TempDir()
	// Filename uses kebab-case; slot key uses snake_case.
	writeYAMLFixture(t, dataDir, "tech-debt.yaml", taskYAML)

	mlw, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.NoError(t, err)
	require.NotNil(t, mlw.TechDebt)
	assert.Equal(t, filepath.Join(dataDir, YAMLWorkflowDir, "tech-debt.yaml"), mlw.Sources["tech_debt"])
}

func TestLoadMultiLevelWorkflowFromYAML_InvalidYAML(t *testing.T) {
	dataDir := t.TempDir()
	writeYAMLFixture(t, dataDir, "task.yaml", "this: is: not: valid: yaml\n  - mixed\n: badly")

	_, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task.yaml")
}

func TestLoadMultiLevelWorkflowFromYAML_DefaultsApplied(t *testing.T) {
	dataDir := t.TempDir()
	// Minimal YAML — version omitted, defaults to "1.0".
	minimal := `status_flow:
  draft:
    - active
  active: []
special_statuses:
  _start_:
    - draft
  _complete_:
    - active
`
	writeYAMLFixture(t, dataDir, "epic.yaml", minimal)

	mlw, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.NoError(t, err)
	require.NotNil(t, mlw.Epic)
	assert.Equal(t, "1.0", mlw.Epic.Version, "missing version field should default to 1.0")
}

func TestGetWorkflowForLevel_FallsBackWhenSlotNil(t *testing.T) {
	// MultiLevelWorkflow with nothing loaded should still hand out the
	// hardcoded defaults — no panic, no nil.
	mlw := &MultiLevelWorkflow{}
	assert.NotNil(t, mlw.GetWorkflowForLevel("task"))
	assert.NotNil(t, mlw.GetWorkflowForLevel("epic"))
	assert.NotNil(t, mlw.GetWorkflowForLevel("tech_debt"))
}

func TestGetWorkflowForLevel_UsesYAMLWhenLoaded(t *testing.T) {
	dataDir := t.TempDir()
	writeYAMLFixture(t, dataDir, "task.yaml", taskYAML)

	mlw, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.NoError(t, err)

	cfg := mlw.GetWorkflowForLevel("task")
	require.NotNil(t, cfg)
	assert.ElementsMatch(t, []string{"in_development", "blocked"}, cfg.StatusFlow["todo"])
}
