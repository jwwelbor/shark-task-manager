# E16-F06: Workflow Visualization - Architecture

## Overview

Update `shark workflow list` to display all three workflow levels (epic, feature, task) with visual distinction between planning and aggregation statuses.

## Current State

- `runWorkflowList()` in `workflow.go` calls `config.LoadWorkflowConfig()` returning only the **task-level** workflow
- `displayWorkflowHumanReadable()` renders a single-level workflow display
- Multi-level infrastructure already exists and is used by `workflow validate`, `workflow show-actions`, and `workflow validate-actions`

### Existing Infrastructure (No Changes Needed)

| Component | Location | Purpose |
|-----------|----------|---------|
| `MultiLevelWorkflow` | `internal/config/workflow_multilevel.go` | Container for epic/feature/task workflows |
| `GetWorkflowForLevel()` | `internal/config/workflow_multilevel.go` | Level-specific workflow with default fallback |
| `LoadMultiLevelWorkflow()` | `internal/config/workflow_parser.go` | Parses multi-level from `.sharkconfig.json` |
| `LoadMultiLevelWorkflowOrDefault()` | `internal/config/workflow_parser.go` | Never-fail variant |
| `DefaultEpicWorkflow()` | `internal/config/workflow_default.go` | Default epic workflow with IsPlanning/AggregatesFrom |
| `DefaultFeatureWorkflow()` | `internal/config/workflow_default.go` | Default feature workflow with IsPlanning/AggregatesFrom |
| `StatusMetadata.IsPlanning` | `internal/config/workflow_schema.go:131` | Planning phase marker |
| `StatusMetadata.AggregatesFrom` | `internal/config/workflow_schema.go:136` | Aggregation source marker |
| `AggregationStatusKey` | `internal/config/workflow_schema.go:150` | `_aggregation_` special status key |

## Design

### Data Flow

```
runWorkflowList()
  -> cli.GetConfigPath()
  -> config.LoadMultiLevelWorkflow(configPath)     // detect custom vs default per level
  -> buildMultiLevelWorkflowDisplay(multi, path)   // build display structs
  -> cli.OutputJSON(display)                        // JSON path
  -> displayMultiLevelWorkflowHumanReadable(display) // human-readable path
```

### JSON Output Structure

```go
type MultiLevelWorkflowDisplay struct {
    EpicWorkflow    *LevelWorkflowDisplay `json:"epic_workflow"`
    FeatureWorkflow *LevelWorkflowDisplay `json:"feature_workflow"`
    TaskWorkflow    *LevelWorkflowDisplay `json:"task_workflow"`
    ConfigPath      string                `json:"config_path"`
}

type LevelWorkflowDisplay struct {
    Level           string              `json:"level"`
    Source          string              `json:"source"` // "custom" or "default"
    Version         string              `json:"version"`
    Statuses        []StatusDisplay     `json:"statuses"`
    SpecialStatuses map[string][]string `json:"special_statuses"`
    StatusCount     int                 `json:"status_count"`
    TransitionCount int                 `json:"transition_count"`
}

type StatusDisplay struct {
    Name           string   `json:"name"`
    Description    string   `json:"description,omitempty"`
    Phase          string   `json:"phase,omitempty"`
    Color          string   `json:"color,omitempty"`
    IsPlanning     bool     `json:"is_planning,omitempty"`
    AggregatesFrom string   `json:"aggregates_from,omitempty"`
    Transitions    []string `json:"transitions"`
    AgentTypes     []string `json:"agent_types,omitempty"`
}
```

### Human-Readable Display

```
Workflow Configuration
================================================================

--- Epic Workflow (default) ---
  Version: 1.0

  Special Statuses:
    _start_ (entry points):  draft
    _complete_ (exit points): completed, archived
    _aggregation_ (threshold): active

  Status Transitions:
    draft (Epic created, not yet started)  [planning]
      -> active
      -> archived
      [phase: planning | color: gray]

    [active] (Epic in progress, aggregating features)  [aggregates: features]
      -> completed
      -> archived
      [phase: execution | color: blue]

    completed (All features complete)
      -> archived
      [phase: done | color: green]

    archived (Epic archived)
      -> (terminal - no transitions)
      [phase: done | color: gray]

--- Feature Workflow (default) ---
  ...similar structure...

--- Task Workflow (custom) ---
  ...task workflow statuses and transitions...

Legend:
  [status] = aggregation threshold (progress derived from children)
  [planning] = entity has its own workflow status (not aggregating)
```

### Visual Markers (REQ-F-003)

| Marker | Condition | Example |
|--------|-----------|---------|
| `[name]` (brackets) | Status has `AggregatesFrom != ""` | `[active]` |
| `[planning]` suffix | Status has `IsPlanning == true` | `draft ... [planning]` |
| `[aggregates: X]` suffix | Status has `AggregatesFrom` set | `[active] ... [aggregates: features]` |

### Source Labels (REQ-F-004)

- `(custom)` when `MultiLevelWorkflow.Epic/Feature/Task` is non-nil (configured in `.sharkconfig.json`)
- `(default)` when nil (using hardcoded defaults)

### Legend (REQ-F-005)

Footer section explaining the visual markers.

## Files Changed

| File | Change | Size |
|------|--------|------|
| `internal/cli/commands/workflow.go` | Rewrite `runWorkflowList()`, add display structs and rendering functions, remove `displayWorkflowHumanReadable()` | M |
| `internal/cli/commands/workflow_test.go` | Update `TestWorkflowListCommand` for multi-level output, add planning/aggregation/JSON tests | M |
| `docs/plan/E16.../feature.md` | Remove Story 4 / REQ-F-006 / Scenario 3 (--level filter) | S |

## Functions Added/Modified

### New Functions in `workflow.go`

- `buildMultiLevelWorkflowDisplay(multi *config.MultiLevelWorkflow, configPath string) *MultiLevelWorkflowDisplay`
- `buildLevelWorkflowDisplay(level string, raw *config.WorkflowConfig, resolved *config.WorkflowConfig) *LevelWorkflowDisplay`
- `displayMultiLevelWorkflowHumanReadable(display *MultiLevelWorkflowDisplay) error`
- `displayWorkflowLevelSection(level *LevelWorkflowDisplay)`
- `displaySpecialStatuses(specials map[string][]string)`
- `displayStatusWithMarkers(status StatusDisplay)`

### Modified Functions

- `runWorkflowList()` - complete rewrite to use multi-level loading

### Removed Functions

- `displayWorkflowHumanReadable(workflow *config.WorkflowConfig)` - replaced by multi-level variant

## Breaking Changes

**JSON output format change**: `shark workflow list --json` currently returns a single `WorkflowConfig`. After this change, it returns a `MultiLevelWorkflowDisplay` with `epic_workflow`, `feature_workflow`, `task_workflow` keys. This is an intentional breaking change as part of E16's multi-level workflow system.

## Test Strategy

Tests use temp config files and captured stdout (no database). Key scenarios:
1. All defaults - verify 3 section headers with "(default)" labels
2. Custom task workflow - verify "(custom)" label for task, "(default)" for epic/feature
3. Planning/aggregation markers - verify `[active]`, `[planning]`, `[aggregates: features]` in output
4. JSON structure - parse `MultiLevelWorkflowDisplay`, verify all 3 levels present with correct fields
5. Legend present in human-readable output
