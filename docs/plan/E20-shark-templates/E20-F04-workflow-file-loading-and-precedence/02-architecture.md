# Technical Architecture: E20-F04 Workflow File Loading and Precedence

**Feature**: [Workflow File Loading and Precedence](./feature.md)
**Requirements**: [Requirements](./requirements.md)
**Epic**: [Shark Templates](../epic.md)
**Date**: 2026-03-18
**Author**: Architect Agent (Claude Opus 4.6, 1M context)

---

## 1. Overview

This document defines the technical architecture for loading workflow configuration from a dedicated `.sharkworkflow.json` file with per-entity precedence over `.sharkconfig.json` inline data. The changes are confined to `internal/config/` (three files) with zero impact on consumers.

### Scope

- **Modified files**: `workflow_parser.go`, `config.go`, `manager.go`
- **New files**: None (all changes fit within existing files)
- **No changes to**: database, services, repositories, CLI commands, HTTP handlers, models
- **Estimated code delta**: ~80-120 lines of production code, ~200-250 lines of tests

---

## 2. Key Architecture Decisions

### ADR-F04-001: Workflow File Loading Location

**Decision**: All workflow file detection, reading, and precedence resolution logic lives inside `LoadMultiLevelWorkflow()` in `workflow_parser.go`.

**Rationale**: REQ-F04-007 requires that file-source logic not leak beyond `internal/config/`. The `LoadMultiLevelWorkflow()` function is the single entry point for all workflow config loading. Placing the two-file logic here means:
- `workflow.Service` initialization is unchanged (calls `LoadMultiLevelWorkflowOrDefault(configPath)`)
- No consumer package needs to know about `.sharkworkflow.json`
- The existing `parseWorkflowSection()` function is reused for blocks from either file

**Alternatives considered**:
- A new `workflow_loader.go` helper file: Rejected because the logic is ~40-60 lines and fits naturally inside `LoadMultiLevelWorkflow()`. Extracting to a separate file adds indirection without benefit. If the logic grows beyond 80 lines in the future, extraction can be revisited.
- Loading in `Manager.Load()`: Rejected because `Manager` handles runtime settings, not workflow parsing. Mixing concerns would violate the existing separation.

---

### ADR-F04-002: Precedence Resolution Algorithm

**Decision**: Two-pass fill with per-entity granularity.

**Algorithm** (inside `LoadMultiLevelWorkflow()`):

```
1. Read .sharkconfig.json at configPath (existing behavior).
2. Extract workflow_config value from raw JSON (new step).
3. Determine workflow file path:
   a. If workflow_config is a non-empty string, resolve it relative to configPath directory.
   b. Else, use <configPath directory>/.sharkworkflow.json.
4. Attempt to read the workflow file:
   a. If file does not exist: skip (no error, no warning).
   b. If file exists but is unreadable: return error.
   c. If file contains invalid JSON: return error.
   d. If file is empty (0 bytes): treat as {} (all entities nil).
5. Parse workflow file into workflowFileConfig map[string]json.RawMessage.
6. For each entity level (epic, feature, task, bug, change):
   a. If workflow file defines {entity}_workflow with non-empty status_flow:
      use workflow file definition (call parseWorkflowSection()).
   b. Else if .sharkconfig.json defines {entity}_workflow (or legacy top-level keys for task):
      use .sharkconfig.json definition (existing behavior).
   c. Else: level remains nil (GetWorkflowForLevel() provides default).
7. For template_directory:
   a. If workflow file defines template_directory: use it.
   b. Else: fall through to .sharkconfig.json value (handled by Config.GetTemplateDirectory()).
8. Populate legacy workflowCache from result.Task (existing behavior, unchanged).
9. Cache and return.
```

**Rationale**: This algorithm satisfies REQ-F04-003 (per-entity precedence) with minimal code. The two-pass approach (workflow file first, config file second) is simple because `MultiLevelWorkflow` uses nullable pointers -- a nil level means "not defined", and the fill logic just skips levels that are already non-nil.

**Alternatives considered**:
- Deep merge of workflow blocks (merge individual fields within a workflow block from both sources): Rejected per feature PRD -- precedence is at the entity-workflow-block level, not at the field level. A `task_workflow` from the workflow file completely replaces a `task_workflow` from `.sharkconfig.json`.
- Loading workflow file in a separate function called before `LoadMultiLevelWorkflow()`: Rejected because it would require a new public API and change the call pattern for all consumers.

---

### ADR-F04-003: Cache Strategy for Two-File Model

**Decision**: Keep the existing single-key cache (`multiLevelCachePath` keyed on `configPath`). Do not add a second cache key for the workflow file path.

**Rationale**: The workflow file path is deterministic from `configPath`:
- If `workflow_config` is set, it is resolved relative to the `configPath` directory.
- If not set, the default is `<configPath directory>/.sharkworkflow.json`.

Therefore, a given `configPath` always produces the same workflow file path. If `configPath` changes, the cache is invalidated, and the new workflow file path is derived from the new config. The existing `ClearWorkflowCache()` clears both `multiLevelCache` and `workflowCache`, which is sufficient.

Additionally, CLI commands are short-lived (one config load per process), so cache staleness from workflow file modifications between calls is not a concern.

**Alternatives considered**:
- Dual-path cache key (concatenating both paths): Adds complexity for no practical benefit in CLI context. Would be relevant for a long-running HTTP server, but that scenario can be addressed when needed.
- Content-hash cache key: Over-engineered for the use case. Adds `crypto/sha256` dependency and file-read overhead on every cache check.

---

### ADR-F04-004: Task Workflow Block Precedence Over Legacy Keys

**Decision**: When both `task_workflow` block and legacy top-level keys (`status_flow`, `status_metadata`, etc.) exist in the same file, `task_workflow` wins.

**Implementation**: In `LoadMultiLevelWorkflow()`, parse `task_workflow` block first (via `parseWorkflowSection()`), then only fall back to `parseTopLevelTaskWorkflow()` if `task_workflow` was not found or was empty.

This applies to both `.sharkworkflow.json` and `.sharkconfig.json`:
- Workflow file: Only `task_workflow` blocks are recognized (no legacy top-level keys).
- `.sharkconfig.json`: `task_workflow` block checked first, legacy top-level keys as fallback.

**Rationale**: This enables gradual migration from legacy format to block format within `.sharkconfig.json`, and ensures the workflow file (which only uses block format) naturally wins via the entity-level precedence chain.

**Note**: T-E20-F04-001 (already exists as a task) handles the prerequisite work of adding `task_workflow` block parsing. E20-F04's main feature work builds on top of that.

---

### ADR-F04-005: Error Handling Strategy

**Decision**: Fail-fast on parse errors in the workflow file; silent fallback on file-not-found.

| Scenario | Behavior | Rationale |
|----------|----------|-----------|
| File does not exist | Silent fallback | REQ-F04-002: backward compatibility |
| File exists, invalid JSON | Return error from `LoadMultiLevelWorkflow()` | Invalid config is a developer mistake; failing loudly helps debugging |
| File exists, valid JSON, invalid entity block | Return error with entity name and file path | Partial parse is dangerous; fail-fast prevents silent misconfiguration |
| File exists, empty (0 bytes) | Treat as `{}`, all entities nil, fall through | Empty file is a reasonable "no overrides" state |
| File exists, unknown top-level keys | Ignored silently | Forward compatibility; allows future keys |

Error messages always include the file path for diagnosability (REQ-F04-011).

---

### ADR-F04-006: Template Directory Precedence

**Decision**: `template_directory` in the workflow file takes precedence over `.sharkconfig.json`, following the same pattern as entity workflows.

**Implementation**: After parsing the workflow file, if it contains a `template_directory` key, store it as a package-level variable or pass it through the existing `Config.TemplateDirectory` field. The simplest approach is to let `LoadMultiLevelWorkflow()` return the template directory alongside the `MultiLevelWorkflow` result, or to have the caller extract it from the raw workflow file data.

**Recommended approach**: Add a `TemplateDirectory *string` field to `MultiLevelWorkflow`. This keeps the template directory coupled with the workflow data it belongs with, and avoids package-level state.

```go
type MultiLevelWorkflow struct {
    Epic              *WorkflowConfig
    Feature           *WorkflowConfig
    Task              *WorkflowConfig
    Bug               *WorkflowConfig
    Change            *WorkflowConfig
    TemplateDirectory *string  // NEW: from workflow file, if present
}
```

The `GetTemplateDirectoryFromConfig()` function in `config.go` can then check the `MultiLevelWorkflow.TemplateDirectory` first, falling back to `Config.TemplateDirectory`.

---

## 3. Code Structure

### 3.1 Changes to `workflow_parser.go`

**Modified function**: `LoadMultiLevelWorkflow(configPath string) (*MultiLevelWorkflow, error)`

New internal steps (inserted after step "Parse full config as raw JSON" at line 197):

```go
// --- NEW: Workflow file loading ---
// 1. Extract workflow_config path from .sharkconfig.json raw data
workflowFilePath := resolveWorkflowFilePath(configPath, rawConfig)

// 2. Attempt to load workflow file
workflowFileData, err := loadWorkflowFile(workflowFilePath)
if err != nil {
    return nil, err // Parse errors are fatal
}
// workflowFileData is nil if file does not exist (silent fallback)

// 3. Parse entity blocks from workflow file (if loaded)
if workflowFileData != nil {
    // For each entity, parse from workflow file first
    for entity, field := range entityFieldMap {
        if raw, ok := workflowFileData[entity+"_workflow"]; ok {
            wf, err := parseWorkflowSection(raw, entity+"_workflow")
            if err != nil {
                return nil, fmt.Errorf("invalid %s_workflow in %s: %w", entity, workflowFilePath, err)
            }
            setEntityLevel(result, entity, wf)
        }
    }
    // Extract template_directory if present
    if tdRaw, ok := workflowFileData["template_directory"]; ok {
        var td string
        if json.Unmarshal(tdRaw, &td) == nil && td != "" {
            result.TemplateDirectory = &td
        }
    }
}

// 4. Fill remaining nil levels from .sharkconfig.json (existing code, unchanged)
// ... existing epic_workflow, feature_workflow, bug_workflow, change_workflow parsing ...
// ... existing parseTopLevelTaskWorkflow() for legacy task keys ...
// But now: only set a level if it is still nil (workflow file did not provide it)
```

**New helper functions** (private, in `workflow_parser.go`):

```go
// resolveWorkflowFilePath determines the workflow file path from config.
// Returns the resolved absolute path to check.
func resolveWorkflowFilePath(configPath string, rawConfig map[string]json.RawMessage) string

// loadWorkflowFile reads and parses the workflow file.
// Returns nil, nil if the file does not exist.
// Returns nil, error if the file exists but cannot be parsed.
func loadWorkflowFile(path string) (map[string]json.RawMessage, error)
```

### 3.2 Changes to `config.go`

**One new field on `Config` struct**:

```go
WorkflowConfig *string `json:"workflow_config,omitempty"`
```

**One new field on `MultiLevelWorkflow` struct** (in `workflow_multilevel.go`):

```go
TemplateDirectory *string // From workflow file, if present
```

### 3.3 Changes to `manager.go`

**In `Manager.Load()`**, add extraction for the new field (follows existing `template_directory` pattern):

```go
if workflowConfig, ok := rawData["workflow_config"].(string); ok && workflowConfig != "" {
    config.WorkflowConfig = &workflowConfig
}
```

### 3.4 Modifications to Existing Parsing Logic

The existing entity-level parsing in `LoadMultiLevelWorkflow()` (lines 207-248) must be guarded with nil checks to avoid overwriting workflow-file-sourced values:

```go
// Parse epic_workflow from .sharkconfig.json ONLY if not already set from workflow file
if result.Epic == nil {
    if epicRaw, ok := rawConfig["epic_workflow"]; ok {
        // ... existing parsing ...
    }
}
```

This pattern repeats for feature, bug, change, and task.

For task specifically, the existing `task_workflow` block parsing (ADR-F04-004) must also be added before the legacy `parseTopLevelTaskWorkflow()` call:

```go
// Parse task_workflow block (new, consistent with other entities)
if result.Task == nil {
    if taskRaw, ok := rawConfig["task_workflow"]; ok {
        taskWf, err := parseWorkflowSection(taskRaw, "task_workflow")
        if err != nil {
            return nil, fmt.Errorf("invalid task_workflow: %w", err)
        }
        result.Task = taskWf
    }
}

// Fall back to legacy top-level keys (existing behavior)
if result.Task == nil {
    taskWf, err := parseTopLevelTaskWorkflow(rawConfig)
    // ...
}
```

---

## 4. Precedence Resolution Summary

For each entity level independently:

```
Priority 1 (highest): .sharkworkflow.json {entity}_workflow block
Priority 2:           .sharkconfig.json {entity}_workflow block
Priority 3:           .sharkconfig.json legacy top-level keys (task only)
Priority 4 (lowest):  Built-in defaults (via GetWorkflowForLevel())
```

For `template_directory`:

```
Priority 1 (highest): .sharkworkflow.json template_directory
Priority 2:           .sharkconfig.json template_directory
Priority 3 (lowest):  DefaultTemplateDir constant ("shark-templates")
```

---

## 5. Testing Strategy

### 5.1 Test Categories

All tests use `t.TempDir()` and fixture files. No test depends on the project's actual config files.

**Unit tests** (in `workflow_parser_test.go`):

| Test | Description | Validates |
|------|-------------|-----------|
| `TestLoadMultiLevelWorkflow_WorkflowFileOnly` | Only `.sharkworkflow.json` exists (no inline config) | REQ-F04-001 |
| `TestLoadMultiLevelWorkflow_NoWorkflowFile` | Only `.sharkconfig.json` with inline blocks | REQ-F04-002 |
| `TestLoadMultiLevelWorkflow_PerEntityPrecedence` | Workflow file defines 2 of 5 entities; config defines all 5 | REQ-F04-003 AC-1 |
| `TestLoadMultiLevelWorkflow_FullPrecedenceChain` | All three tiers (workflow file, config, defaults) for different entities | REQ-F04-003 AC-2 |
| `TestLoadMultiLevelWorkflow_CustomPath` | `workflow_config` points to non-default location | REQ-F04-004 AC-1 |
| `TestLoadMultiLevelWorkflow_MissingCustomPath` | `workflow_config` points to nonexistent file | REQ-F04-004 AC-2 |
| `TestLoadMultiLevelWorkflow_EmptyWorkflowConfig` | `workflow_config: ""` treated as absent | REQ-F04-004 AC-4 |
| `TestLoadMultiLevelWorkflow_AbsolutePath` | `workflow_config` with absolute path | REQ-F04-004 AC-5 |
| `TestLoadMultiLevelWorkflow_TaskWorkflowBlock` | `task_workflow` block wins over legacy top-level keys | ADR-F04-004 |
| `TestLoadMultiLevelWorkflow_TemplateDirPrecedence` | template_directory from workflow file wins | REQ-F04-006 |
| `TestLoadMultiLevelWorkflow_InvalidWorkflowFileJSON` | Malformed JSON in workflow file returns error | Error handling |
| `TestLoadMultiLevelWorkflow_EmptyWorkflowFile` | 0-byte file treated as `{}` | Edge case 4 |
| `TestLoadMultiLevelWorkflow_WorkflowFileEmptyEntity` | Entity block `{}` treated as not-defined | Edge case 1 |
| `TestLoadMultiLevelWorkflow_UnknownKeys` | Unknown keys in workflow file ignored | Forward compat |
| `TestLoadMultiLevelWorkflow_LegacyCacheSync` | `workflowCache` populated from winning task source | REQ-F04-010 |

**Benchmark test**:

| Test | Description | Target |
|------|-------------|--------|
| `BenchmarkLoadMultiLevelWorkflow_TwoFiles` | Both files present, measure overhead | <5ms (REQ-NF04-002) |

**Regression test**:

| Test | Description | Validates |
|------|-------------|-----------|
| `TestLoadMultiLevelWorkflow_RegressionNoWorkflowFile` | Load with config-only setup, compare result to known-good baseline | REQ-NF04-001 |

### 5.2 Test Fixture Pattern

```go
func TestLoadMultiLevelWorkflow_PerEntityPrecedence(t *testing.T) {
    dir := t.TempDir()

    // Write .sharkconfig.json with all 5 entity workflows
    configData := map[string]interface{}{
        "epic_workflow":    buildWorkflow("config-epic"),
        "feature_workflow": buildWorkflow("config-feature"),
        "task_workflow":    buildWorkflow("config-task"),
        "bug_workflow":     buildWorkflow("config-bug"),
        "change_workflow":  buildWorkflow("config-change"),
    }
    writeJSON(t, filepath.Join(dir, ".sharkconfig.json"), configData)

    // Write .sharkworkflow.json with only epic and task
    workflowData := map[string]interface{}{
        "epic_workflow": buildWorkflow("file-epic"),
        "task_workflow": buildWorkflow("file-task"),
    }
    writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

    ClearWorkflowCache()
    result, err := LoadMultiLevelWorkflow(filepath.Join(dir, ".sharkconfig.json"))
    require.NoError(t, err)

    // Epic and task from workflow file
    assert.Equal(t, "file-epic", result.Epic.Version)
    assert.Equal(t, "file-task", result.Task.Version)

    // Feature, bug, change from .sharkconfig.json
    assert.Equal(t, "config-feature", result.Feature.Version)
    assert.Equal(t, "config-bug", result.Bug.Version)
    assert.Equal(t, "config-change", result.Change.Version)
}
```

---

## 6. Implementation Sequence

### Task 1: Standardize task_workflow block parsing (T-E20-F04-001, exists)

Add `task_workflow` block support in `LoadMultiLevelWorkflow()`. The `task_workflow` block is parsed via `parseWorkflowSection()` (same as the other four entities), with precedence over legacy top-level keys via `parseTopLevelTaskWorkflow()`.

**Files**: `workflow_parser.go`
**Tests**: Add tests for `task_workflow` block parsing, fallback to legacy keys, and both-present precedence.
**Complexity**: S

### Task 2: Add workflow file loading and per-entity precedence

Implement the two-file loading model in `LoadMultiLevelWorkflow()`:
- Add `resolveWorkflowFilePath()` and `loadWorkflowFile()` helpers
- Parse entity blocks from workflow file before `.sharkconfig.json`
- Guard existing `.sharkconfig.json` parsing with nil checks

**Files**: `workflow_parser.go`
**Tests**: Two-file scenarios, per-entity precedence, fallback, error handling
**Complexity**: M

### Task 3: Add WorkflowConfig field and template_directory precedence

- Add `WorkflowConfig *string` to `Config` struct
- Extract `workflow_config` in `Manager.Load()`
- Add `TemplateDirectory *string` to `MultiLevelWorkflow`
- Update `GetTemplateDirectoryFromConfig()` to check workflow-sourced value

**Files**: `config.go`, `manager.go`, `workflow_multilevel.go`
**Tests**: Custom path resolution, template directory precedence
**Complexity**: S

### Task 4: Performance benchmark and regression tests

- Add `BenchmarkLoadMultiLevelWorkflow_TwoFiles`
- Add regression test comparing config-only loading to known baseline
- Verify `make test` passes with no modifications to existing tests

**Files**: `workflow_parser_test.go`
**Complexity**: XS

---

## 7. Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Regression in workflow loading | Existing test suite runs unchanged; new regression test compares output |
| Legacy `LoadWorkflowConfig()` bypassed | Audit confirms all callers use `GetWorkflowOrDefault()` which delegates to multi-level path. `LoadWorkflowConfig()` is not called directly by any consumer. |
| Cache stale with two files | Cache keys on `configPath` which deterministically maps to workflow file path. CLI commands are short-lived. `ClearWorkflowCache()` used in tests. |
| `parseTopLevelTaskWorkflow()` conflicts with `task_workflow` block | Clear precedence: block wins over legacy keys. Both paths are tested. |
| Template directory resolution breaks | Three-tier fallback chain (workflow file > config > default) with explicit tests for each tier. |

---

## 8. Cross-Feature Consistency

### Relationship to Other E20 Features

| Feature | Relationship to F04 |
|---------|---------------------|
| E20-F03 (Task Workflow Standardization) | **Cancelled**; T-E20-F04-001 absorbed the prerequisite work of adding `task_workflow` block support |
| E20-F05 (Init Update) | **Downstream**; writes `.sharkworkflow.json` that F04 reads. F05 depends on F04's file format and precedence rules being established. |
| E20-F06 (Config Validation) | **Downstream**; validates `.sharkworkflow.json` that F04 introduces. F06 uses the same `parseWorkflowSection()` function for validation. |

### Backward Compatibility Contract

After F04 is complete:
1. Every existing `.sharkconfig.json` with inline workflows produces identical `MultiLevelWorkflow` results.
2. No consumer code outside `internal/config/` references `.sharkworkflow.json` or `workflow_config`.
3. `make test` passes with zero test modifications.

---

*Last Updated*: 2026-03-18
