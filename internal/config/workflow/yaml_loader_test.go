package workflow

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// canonicalTaskYAMLDir resolves the embedded canonical workflow directory by
// walking up from this test file to the repository root, then descending into
// internal/sharkdata/default_data/workflow. This is the same path used by
// canonical_provider_test.go and mirrors what shark-data/workflow/ holds in
// the project root (the two trees are identical after `shark upgrade`).
func canonicalTaskYAMLDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile: <repo>/internal/config/workflow/yaml_loader_test.go
	// target:   <repo>/internal/sharkdata/default_data/workflow
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(repoRoot, "internal", "sharkdata", "default_data", "workflow")
}

// TestLoadCanonicalTaskYAML_RoundTripParity is the E2 proof-entity test.
//
// It loads the canonical shark-data/workflow/task.yaml via
// LoadMultiLevelWorkflowFromYAMLDir and verifies:
//  1. Load succeeds without error.
//  2. All statuses declared in status_flow are also present in status_metadata
//     (semantic self-consistency — no orphan transitions).
//  3. All statuses declared in status_metadata are reachable from status_flow
//     or listed as start/complete in special_statuses.
//  4. Every status_metadata entry carries the minimum required fields:
//     color, phase, progress_weight (≥ 0), and responsibility.
//  5. special_statuses._start_ and ._complete_ are non-empty.
//  6. Per-file error accumulation: loading the workflow directory (which
//     contains epic.yaml, feature.yaml, etc.) does not abort even when one
//     entity file is not the focus of this test.  The task slot is populated
//     and the other loaded slots are non-nil (or nil only when absent from the
//     dir) — no error returned.
//
// This test intentionally does NOT assert that the YAML status set matches the
// legacy .sharkworkflow.json task_workflow statuses: task.yaml represents the
// Shark 2.0 simplified task workflow (7 statuses: draft, ready_for_development,
// in_development, completed, cancelled, blocked, on_hold), whereas the legacy
// JSON carries the full TDD pipeline (16 statuses). They are different workflow
// profiles. The YAML loader's job is to faithfully reproduce the YAML file's
// content in the in-memory model — that is what this test verifies.
func TestLoadCanonicalTaskYAML_RoundTripParity(t *testing.T) {
	workflowDir := canonicalTaskYAMLDir(t)
	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		t.Skipf("canonical workflow dir not found at %s; skipping E2 round-trip test", workflowDir)
	}

	mlw, err := LoadMultiLevelWorkflowFromYAMLDir(workflowDir, "")
	require.NoError(t, err, "loading canonical task.yaml must not return an error")
	require.NotNil(t, mlw, "LoadMultiLevelWorkflowFromYAMLDir must return a non-nil MultiLevelWorkflow")

	// ── 1. task slot is populated ──────────────────────────────────────────
	require.NotNil(t, mlw.Task, "task slot must be non-nil after loading canonical task.yaml")
	cfg := mlw.Task

	// ── 2. status_flow self-consistency ───────────────────────────────────
	// Every status that appears as a source in status_flow must have metadata.
	for src, targets := range cfg.StatusFlow {
		assert.Contains(t, cfg.StatusMetadata, src,
			"status %q appears in status_flow but has no entry in status_metadata", src)
		for _, tgt := range targets {
			assert.Contains(t, cfg.StatusMetadata, tgt,
				"transition target %q (from %q) has no entry in status_metadata", tgt, src)
		}
	}

	// ── 3. status_metadata reachability ───────────────────────────────────
	// Every status in metadata must be reachable: either it's a source in
	// status_flow, or it's listed in special_statuses (_start_ / _complete_).
	reachable := make(map[string]bool)
	for src := range cfg.StatusFlow {
		reachable[src] = true
	}
	for _, statuses := range cfg.SpecialStatuses {
		for _, s := range statuses {
			reachable[s] = true
		}
	}
	for status := range cfg.StatusMetadata {
		assert.True(t, reachable[status],
			"status %q has metadata but is not reachable from status_flow or special_statuses", status)
	}

	// ── 4. minimum required metadata fields ───────────────────────────────
	for status, meta := range cfg.StatusMetadata {
		assert.NotEmpty(t, meta.Color,
			"status %q metadata missing color", status)
		assert.NotEmpty(t, meta.Phase,
			"status %q metadata missing phase", status)
		// progress_weight is a float64 default zero value is valid (planning statuses),
		// so only assert it is in the expected [0, 1] range.
		assert.GreaterOrEqual(t, meta.ProgressWeight, float64(0),
			"status %q progress_weight must be >= 0", status)
		assert.LessOrEqual(t, meta.ProgressWeight, float64(1),
			"status %q progress_weight must be <= 1.0", status)
		assert.NotEmpty(t, meta.Responsibility,
			"status %q metadata missing responsibility", status)
	}

	// ── 5. special_statuses are non-empty ─────────────────────────────────
	startStatuses, hasStart := cfg.SpecialStatuses[StartStatusKey]
	assert.True(t, hasStart, "task.yaml must declare _start_ in special_statuses")
	assert.NotEmpty(t, startStatuses, "task.yaml _start_ must be non-empty")

	completeStatuses, hasComplete := cfg.SpecialStatuses[CompleteStatusKey]
	assert.True(t, hasComplete, "task.yaml must declare _complete_ in special_statuses")
	assert.NotEmpty(t, completeStatuses, "task.yaml _complete_ must be non-empty")

	// ── 5a. specific values from the canonical task.yaml ──────────────────
	// The canonical task.yaml is now route-based (E35): the legacy
	// ready_for_development / in_development pair collapses into a single
	// `development` step, with the old names preserved in its aliases:. These
	// assertions encode the expected content of the collapsed file and serve as
	// a regression guard against accidental breaking edits. The derived
	// StatusFlow target sets must match the collapsed original targets.
	require.True(t, cfg.HasSteps(), "canonical task.yaml must use the route-based steps: schema")
	assert.Equal(t, "draft", cfg.Start, "canonical task.yaml start step must be draft")

	expectedStatuses := []string{
		"draft", "development", "completed", "cancelled", "blocked", "on_hold",
	}
	for _, s := range expectedStatuses {
		assert.Contains(t, cfg.StatusFlow, s,
			"expected collapsed status %q in canonical task.yaml status_flow", s)
		assert.Contains(t, cfg.StatusMetadata, s,
			"expected collapsed status %q in canonical task.yaml status_metadata", s)
	}

	// The old ready_for_development / in_development names must resolve to the
	// collapsed `development` step via the alias map (input compat + migration).
	aliasMap, aliasErrs := cfg.AliasMap()
	assert.Empty(t, aliasErrs, "task.yaml alias map must be collision-free")
	assert.Equal(t, "development", aliasMap["ready_for_development"],
		"ready_for_development must alias to the collapsed development step")
	assert.Equal(t, "development", aliasMap["in_development"],
		"in_development must alias to the collapsed development step")

	// Derived transitions: draft advances to development (pass outcome), and
	// development advances to completed (pass) and falls back to draft (fail).
	assert.Contains(t, cfg.StatusFlow["draft"], "development",
		"draft must have development as a derived transition")
	assert.Contains(t, cfg.StatusFlow["development"], "completed",
		"development must reach completed via its pass outcome")
	assert.Contains(t, cfg.StatusFlow["development"], "draft",
		"development must fall back to draft via its fail outcome")

	// completed is a terminal step (no outgoing transitions in the derived map).
	assert.Empty(t, cfg.StatusFlow["completed"],
		"completed is terminal in the route-based task workflow")

	// _start_ must include draft.
	assert.Contains(t, cfg.SpecialStatuses[StartStatusKey], "draft",
		"_start_ must include draft")

	// _complete_ must include completed and cancelled.
	assert.Contains(t, cfg.SpecialStatuses[CompleteStatusKey], "completed",
		"_complete_ must include completed")
	assert.Contains(t, cfg.SpecialStatuses[CompleteStatusKey], "cancelled",
		"_complete_ must include cancelled")

	// ── 6. per-file error accumulation ────────────────────────────────────
	// The canonical workflow dir also contains epic.yaml, feature.yaml, etc.
	// Loading should populate multiple slots without error.
	assert.NotNil(t, mlw.Epic, "epic slot should be populated from the canonical dir")
	assert.NotNil(t, mlw.Feature, "feature slot should be populated from the canonical dir")
	assert.NotNil(t, mlw.Bug, "bug slot should be populated from the canonical dir")
	assert.NotNil(t, mlw.Change, "change slot should be populated from the canonical dir")

	// Source tracking: task slot must record the file it was loaded from.
	taskSrc, hasSrc := mlw.Sources["task"]
	assert.True(t, hasSrc, "Sources map must have an entry for 'task'")
	assert.True(t, strings.HasSuffix(taskSrc, "task.yaml"),
		"task source path must end in task.yaml, got %q", taskSrc)
}

// TestLoadMultiLevelWorkflowFromYAML_LogsDebugWhenFileMissing exercises the
// TD-024 fix: when a per-entity workflow YAML is absent, the loader silently
// falls back to the embedded default but must emit a verbose-only (slog.Debug)
// trace so operators can diagnose divergence between expected and actual
// dispatch behavior.
func TestLoadMultiLevelWorkflowFromYAML_LogsDebugWhenFileMissing(t *testing.T) {
	// Install an in-memory JSON slog handler at Debug level so we can capture
	// the trace messages produced by the loader. Restore the previous default
	// after the test.
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	old := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(old)

	dataDir := t.TempDir()
	// Provide only task.yaml; epic/feature/bug/change/tech-debt/sprint slots
	// will be missing and should each emit a debug trace.
	writeYAMLFixture(t, dataDir, "task.yaml", taskYAML)

	_, err := LoadMultiLevelWorkflowFromYAML(dataDir)
	require.NoError(t, err)

	// Parse captured log lines into structured records.
	var records []map[string]interface{}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var rec map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("failed to parse log line %q: %v", line, err)
		}
		records = append(records, rec)
	}

	// Collect entity_type values from records whose message matches the
	// fallback trace.
	const wantMsg = "workflow yaml not found; using built-in default"
	missingSlots := map[string]string{}
	for _, rec := range records {
		if rec["msg"] != wantMsg {
			continue
		}
		// Verify level is DEBUG so this stays a verbose-only trace.
		assert.Equal(t, "DEBUG", rec["level"], "fallback trace must be DEBUG-level")
		// Capture entity_type → path for cross-checks below.
		entity, _ := rec["entity_type"].(string)
		path, _ := rec["path"].(string)
		missingSlots[entity] = path
	}

	// task.yaml was provided, so it must NOT have emitted a fallback trace.
	assert.NotContains(t, missingSlots, "task")

	// Every other slot must have emitted a fallback trace naming its expected
	// default path under <dataDir>/workflow/.
	for _, slot := range []string{"epic", "feature", "bug", "change", "tech_debt", "sprint"} {
		path, ok := missingSlots[slot]
		assert.True(t, ok, "expected fallback trace for slot %q", slot)
		assert.True(t, strings.HasPrefix(path, filepath.Join(dataDir, YAMLWorkflowDir)),
			"trace path for slot %q should reference the workflow dir, got %q", slot, path)
	}
}
