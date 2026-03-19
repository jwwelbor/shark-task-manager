# Test Plan: E20-F04 Workflow File Loading and Precedence

**Feature**: [Workflow File Loading and Precedence](./feature.md)
**Requirements**: [Requirements](./requirements.md)
**Architecture**: [Architecture](./02-architecture.md)
**Epic UAT Plan**: [UAT Acceptance Plan](../uat-acceptance-plan.md)
**Date**: 2026-03-18
**Author**: QA Agent (Claude Opus 4.6, 1M context)

---

## 1. Acceptance Criteria Test Matrix

All tests target `internal/config/workflow_parser.go` and use `t.TempDir()` with fixture files per REQ-NF04-004. Tests follow the codebase pattern established in `workflow_test.go` and `workflow_multilevel_test.go`: create temp dir, write fixture JSON, call `ClearWorkflowCache()`, invoke `LoadMultiLevelWorkflow()`, assert results.

### AC Group 1: Workflow File Detection (REQ-F04-001)

Maps to UAT: AS-A01

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-001 | `TestLoadMultiLevelWorkflow_WorkflowFileDetected` | `.sharkconfig.json` (runtime only) + `.sharkworkflow.json` (all 5 entity blocks with distinct `version` markers) | `result.Epic.Version == "file-epic"`, etc. for all 5 entities | No |
| TC-002 | `TestLoadMultiLevelWorkflow_WorkflowFileOnlyPartialEntities` | `.sharkworkflow.json` with only `epic_workflow` | `result.Epic` from file; `result.Task`, `result.Feature` etc. nil (defaults via `GetWorkflowForLevel()`) | Yes |
| TC-003 | `TestLoadMultiLevelWorkflow_UnknownKeysIgnored` | `.sharkworkflow.json` with `sprint_workflow` and `unknown_key` alongside `task_workflow` | `result.Task` populated from file; no error returned; unknown keys silently ignored | Yes -- forward compat |

### AC Group 2: Backward-Compatible Fallback (REQ-F04-002)

Maps to UAT: AS-B01, AS-B02, AS-B03

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-010 | `TestLoadMultiLevelWorkflow_NoWorkflowFile` | `.sharkconfig.json` with inline epic/feature/task/bug/change workflow blocks. No `.sharkworkflow.json`. | All 5 entity levels populated from inline config. Identical to pre-E20 behavior. | No |
| TC-011 | `TestLoadMultiLevelWorkflow_RegressionNoWorkflowFile` | Production-shaped `.sharkconfig.json` with advanced profile. No workflow file. | Compare `result` field-by-field against a known-good baseline built from the same config file parsed by the existing code path. Zero drift. | No -- regression |
| TC-012 | `TestLoadMultiLevelWorkflow_NoConfigNoWorkflowFile` | Empty `.sharkconfig.json` (`{}`). No `.sharkworkflow.json`. | All entity levels nil. `GetWorkflowForLevel()` returns built-in defaults (task: 5 statuses, epic: 4, feature: 4, bug: 7, change: 5). | Yes -- fresh project |

### AC Group 3: Per-Entity Precedence (REQ-F04-003)

Maps to UAT: AS-C01, AS-C02, AS-C03

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-020 | `TestLoadMultiLevelWorkflow_PerEntityPrecedence` | `.sharkworkflow.json` defines `epic_workflow` (version "file-epic") and `task_workflow` (version "file-task"). `.sharkconfig.json` defines all 5 (version "config-*"). | Epic and Task from file; Feature, Bug, Change from config. | No |
| TC-021 | `TestLoadMultiLevelWorkflow_FullPrecedenceChain` | `.sharkworkflow.json` defines `epic_workflow` only. `.sharkconfig.json` defines `feature_workflow` only. Neither defines task/bug/change. | Epic from file. Feature from config. Task, Bug, Change from built-in defaults. | No |
| TC-022 | `TestLoadMultiLevelWorkflow_WorkflowFileOverridesInline` | Both files define `task_workflow` with different `status_flow` maps. | `result.Task.StatusFlow` matches workflow file, not config. | No |
| TC-023 | `TestLoadMultiLevelWorkflow_EmptyEntityInWorkflowFile` | `.sharkworkflow.json` has `"task_workflow": {}`. `.sharkconfig.json` has `task_workflow` with valid flow. | `result.Task` populated from `.sharkconfig.json` (empty block treated as "not defined"). | Yes -- edge case 1 from feature.md |
| TC-024 | `TestLoadMultiLevelWorkflow_EmptyStatusFlowInWorkflowFile` | `.sharkworkflow.json` has `"task_workflow": {"status_flow": {}}`. `.sharkconfig.json` has valid task. | `result.Task` from config (empty status_flow treated as nil per `parseWorkflowSection()` behavior). | Yes -- edge case 2 |
| TC-025 | `TestLoadMultiLevelWorkflow_ResolutionHappensOnce` | Verify that only `workflow_parser.go` contains file-source awareness. | `grep -r ".sharkworkflow.json" internal/ --include="*.go" -l` returns only files in `internal/config/`. | NFR -- REQ-NF04-003 |

### AC Group 4: Configurable Workflow File Path (REQ-F04-004)

Maps to UAT: AS-A02, AS-A03

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-030 | `TestLoadMultiLevelWorkflow_CustomPath` | `.sharkconfig.json` has `"workflow_config": "config/workflows.json"`. `config/workflows.json` exists with `epic_workflow`. | Epic from `config/workflows.json`. | No -- AC-1 |
| TC-031 | `TestLoadMultiLevelWorkflow_MissingCustomPath` | `.sharkconfig.json` has `"workflow_config": "nonexistent.json"`. Config has inline task workflow. | Task from inline config. No error returned. | No -- AC-2 |
| TC-032 | `TestLoadMultiLevelWorkflow_AbsentWorkflowConfigKey` | `.sharkconfig.json` with no `workflow_config` key. `.sharkworkflow.json` in same dir. | Workflow file at default path is loaded. | No -- AC-3 |
| TC-033 | `TestLoadMultiLevelWorkflow_EmptyWorkflowConfig` | `"workflow_config": ""` in `.sharkconfig.json`. `.sharkworkflow.json` in same dir. | Empty string treated as absent; default `.sharkworkflow.json` loaded. | Yes -- AC-4 |
| TC-034 | `TestLoadMultiLevelWorkflow_AbsolutePath` | `"workflow_config": "/tmp/xxx/workflows.json"` (absolute path via `t.TempDir()`). File exists. | Loaded from absolute path. | Yes -- AC-5 |
| TC-035 | `TestLoadMultiLevelWorkflow_RelativePathResolution` | `"workflow_config": "subdir/wf.json"`. File at `<configDir>/subdir/wf.json`. | Path resolved relative to config directory. | No -- AC-6 |

### AC Group 5: Template Directory Precedence (REQ-F04-006)

Maps to UAT: INT-04

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-040 | `TestLoadMultiLevelWorkflow_TemplateDirFromWorkflowFile` | `.sharkworkflow.json` has `"template_directory": "custom-templates"`. `.sharkconfig.json` has `"template_directory": "shark-templates"`. | `result.TemplateDirectory` == `"custom-templates"`. | No -- AC-1 |
| TC-041 | `TestLoadMultiLevelWorkflow_TemplateDirFallbackToConfig` | `.sharkworkflow.json` has no `template_directory`. `.sharkconfig.json` has `"template_directory": "shark-templates"`. | `result.TemplateDirectory` is nil (Config layer handles fallback). | No -- AC-2 |
| TC-042 | `TestLoadMultiLevelWorkflow_TemplateDirNeitherFile` | Neither file has `template_directory`. | `result.TemplateDirectory` is nil. Default `"shark-templates"` applied by `GetTemplateDirectory()`. | Yes -- AC-3 |

### AC Group 6: Error Handling (REQ-F04-011, ADR-F04-005)

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-050 | `TestLoadMultiLevelWorkflow_InvalidWorkflowFileJSON` | `.sharkworkflow.json` contains `{invalid}`. | Error returned. Error message contains file path. | No |
| TC-051 | `TestLoadMultiLevelWorkflow_InvalidEntityBlock` | `.sharkworkflow.json` has `"epic_workflow": "not-an-object"`. | Error returned. Message contains `"epic_workflow"` and file path. | No |
| TC-052 | `TestLoadMultiLevelWorkflow_EmptyWorkflowFile` | `.sharkworkflow.json` is 0 bytes. | No error. All entities nil from file (fall through to config/defaults). | Yes -- edge case 4 |
| TC-053 | `TestLoadMultiLevelWorkflow_CircularWorkflowConfig` | `"workflow_config"` points to `.sharkconfig.json` itself. | No error. Config loaded twice harmlessly; entity blocks resolved from the config file. | Yes -- edge case 4 from feature.md |

### AC Group 7: Task Workflow Block Precedence (ADR-F04-004)

Maps to UAT: AS-D01, AS-D02, AS-D03

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-060 | `TestLoadMultiLevelWorkflow_TaskWorkflowBlock` | `.sharkconfig.json` has `task_workflow` block with `version: "block"` AND legacy top-level `status_flow`/`status_metadata` with different statuses. | `result.Task.Version == "block"`. Legacy keys ignored. | No -- AS-D03 |
| TC-061 | `TestLoadMultiLevelWorkflow_LegacyTaskKeysOnly` | `.sharkconfig.json` has legacy top-level `status_flow`/`status_metadata` only. No `task_workflow` block. | `result.Task` populated from legacy keys. | No -- AS-D02 |
| TC-062 | `TestLoadMultiLevelWorkflow_WorkflowFileTaskOverridesConfigBlock` | `.sharkworkflow.json` has `task_workflow` (version "file"). `.sharkconfig.json` has `task_workflow` block (version "config-block") AND legacy keys. | `result.Task.Version == "file"`. Workflow file wins over both config block and legacy. | No |

### AC Group 8: Cache Behavior (REQ-F04-010)

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-070 | `TestLoadMultiLevelWorkflow_LegacyCacheSync` | Load with both files. Task from workflow file. | `GetWorkflowOrDefault()` returns same task workflow as `result.Task`. Legacy `workflowCache` is in sync. | No |
| TC-071 | `TestLoadMultiLevelWorkflow_CacheClearedBetweenLoads` | Load config A. `ClearWorkflowCache()`. Load config B with different workflows. | Second load returns config B data, not stale config A. | No |

### AC Group 9: Performance (REQ-NF04-002)

| TC ID | Test Name | Input | Expected Output | Edge Case? |
|-------|-----------|-------|-----------------|------------|
| TC-080 | `BenchmarkLoadMultiLevelWorkflow_TwoFiles` | Both files with full advanced profile (~1500 lines). | Benchmark result. Target: <5ms per operation. Report ns/op. | No |
| TC-081 | `BenchmarkLoadMultiLevelWorkflow_ConfigOnly` | Config-only baseline (no workflow file). | Benchmark result. Compare against TC-080 to measure overhead. | No |

---

## 2. Component Test Strategy

### Target Component: `workflow_parser.go`

This is the sole file containing workflow file detection, loading, and precedence logic. All changes are confined here per ADR-F04-001.

**Functions Under Test:**

| Function | Type | Coverage Strategy |
|----------|------|-------------------|
| `LoadMultiLevelWorkflow(configPath)` | Modified (public) | All TC-* tests above exercise this function. Table-driven tests for precedence scenarios (TC-020 through TC-025). |
| `resolveWorkflowFilePath(configPath, rawConfig)` | New (private) | Tested indirectly through TC-030 to TC-035. Consider a focused sub-test if path resolution logic exceeds 15 lines. |
| `loadWorkflowFile(path)` | New (private) | Tested indirectly through TC-050 to TC-053 (error scenarios). Happy path via TC-001. |
| `parseWorkflowSection(raw, name)` | Existing (reused) | No changes needed to this function. Existing tests in `workflow_test.go` cover it. New tests verify it is called correctly for workflow file blocks. |

**Test Fixture Pattern (matching codebase conventions):**

```go
func TestLoadMultiLevelWorkflow_PerEntityPrecedence(t *testing.T) {
    dir := t.TempDir()

    // Write .sharkconfig.json with all 5 entity workflows using "config-*" version markers
    configData := buildFullConfig("config")
    writeJSON(t, filepath.Join(dir, ".sharkconfig.json"), configData)

    // Write .sharkworkflow.json with only epic and task using "file-*" version markers
    workflowData := map[string]interface{}{
        "epic_workflow": buildWorkflowBlock("file-epic"),
        "task_workflow": buildWorkflowBlock("file-task"),
    }
    writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

    ClearWorkflowCache()
    result, err := LoadMultiLevelWorkflow(filepath.Join(dir, ".sharkconfig.json"))
    require.NoError(t, err)

    // Workflow file wins for epic and task
    assert.Equal(t, "file-epic", result.Epic.Version)
    assert.Equal(t, "file-task", result.Task.Version)

    // Config wins for feature, bug, change
    assert.Equal(t, "config-feature", result.Feature.Version)
    assert.Equal(t, "config-bug", result.Bug.Version)
    assert.Equal(t, "config-change", result.Change.Version)
}
```

**Test Helpers to Create:**

- `buildWorkflowBlock(version string) map[string]interface{}` -- builds a minimal valid `{entity}_workflow` block with a distinguishable `version` field and a non-empty `status_flow`.
- `buildFullConfig(prefix string) map[string]interface{}` -- builds a `.sharkconfig.json` body with all 5 entity workflow blocks using `prefix` in version strings.
- `writeJSON(t *testing.T, path string, data interface{})` -- marshals and writes JSON to path. Calls `t.Fatal` on error.

These helpers follow the pattern in `workflow_test.go` where fixture files are written inline with `os.WriteFile`.

### Supporting Files

| File | Change | Test Impact |
|------|--------|-------------|
| `config.go` | Add `WorkflowConfig *string` field | Tested by `manager_test.go` (existing pattern for new fields) + TC-030 through TC-035 |
| `manager.go` | Extract `workflow_config` from raw JSON | Tested by existing `TestManager_Load` pattern + TC-030 |
| `workflow_multilevel.go` | Add `TemplateDirectory *string` to `MultiLevelWorkflow` | TC-040 through TC-042 |

---

## 3. Integration Scenarios

### Integration Point 1: workflow.Service Initialization (INT-03 from UAT)

**What**: `workflow.Service` calls `LoadMultiLevelWorkflowOrDefault(configPath)`. After E20-F04, this function must return correct results whether workflow data comes from one file or two.

**Test Approach**: Not a unit test in `workflow_parser_test.go`. Verified by:
- TC-010 (no workflow file -- existing path unchanged)
- TC-020 (two files -- new path produces correct `MultiLevelWorkflow`)
- The `LoadMultiLevelWorkflowOrDefault()` wrapper is a thin error-swallowing layer; if `LoadMultiLevelWorkflow()` is correct, the wrapper is correct.

**Regression Signal**: `make test` passes with no `.sharkworkflow.json` in the project root. This validates that `workflow.Service` consumers (all CLI commands, status calculation, template rendering) are unaffected.

### Integration Point 2: Template Rendering (INT-04 from UAT)

**What**: `GetTemplateDirectory()` must return the correct directory regardless of which file contains `template_directory`.

**Test Approach**: TC-040 through TC-042 validate `MultiLevelWorkflow.TemplateDirectory` is set correctly. The `GetTemplateDirectory()` function reads this field and falls back. A focused integration test:

```
TC-INT-01: TestGetTemplateDirectory_WorkflowFileSource
  Setup: Load config with workflow file containing template_directory
  Call:  GetTemplateDirectory() or equivalent
  Assert: Returns workflow file value, not config value
```

### Integration Point 3: `shark init update --workflow=` (E20-F05 downstream)

**What**: E20-F05 generates `.sharkworkflow.json`. F04 must correctly load whatever F05 produces.

**Test Approach**: Not tested in F04's test suite (F05 is downstream). However, the data contract (JSON schema in feature.md section "Data Contracts") establishes the file format. F04's tests use fixtures matching this schema.

**Contract Tests**: TC-001 and TC-020 verify that a file matching the documented schema is loaded correctly. If F05 generates a file that differs from the schema, F05's tests should catch it.

### Integration Point 4: Legacy `LoadWorkflowConfig()` (REQ-F04-010)

**What**: The legacy single-level API (`GetWorkflowOrDefault()`) must return the task workflow from whichever source won precedence.

**Test Approach**: TC-070 validates this explicitly. After loading with both files, `GetWorkflowOrDefault()` is called and compared against `result.Task`.

### Integration Point 5: Config Validation (E20-F06 downstream)

**What**: E20-F06 validates the workflow file. F04 introduces the file; F06 validates it.

**Test Approach**: F04 does not validate workflow file structure beyond JSON syntax and entity block parsing (which is done by `parseWorkflowSection()`). TC-050 and TC-051 verify that parse errors from the workflow file are surfaced correctly. F06 builds on this with deeper structural validation.

---

## 4. Requirement Traceability

| Requirement | Test Cases | UAT Scenario |
|-------------|------------|--------------|
| REQ-F04-001 (File Detection) | TC-001, TC-002, TC-003 | AS-A01 |
| REQ-F04-002 (Backward Fallback) | TC-010, TC-011, TC-012 | AS-B01, AS-B02, AS-B03 |
| REQ-F04-003 AC-1 (Per-Entity Override) | TC-020, TC-022, TC-023, TC-024 | AS-C01, AS-C02 |
| REQ-F04-003 AC-2 (Full Chain) | TC-021 | AS-C03 |
| REQ-F04-003 AC-3 (Single Resolution) | TC-025 | -- (NFR verification) |
| REQ-F04-004 AC-1 (Custom Path) | TC-030 | AS-A02 |
| REQ-F04-004 AC-2 (Missing Custom) | TC-031 | AS-A03 |
| REQ-F04-004 AC-3 (Absent Key) | TC-032 | -- |
| REQ-F04-004 AC-4 (Empty String) | TC-033 | -- |
| REQ-F04-004 AC-5 (Absolute Path) | TC-034 | -- |
| REQ-F04-004 AC-6 (Relative Path) | TC-035 | -- |
| REQ-F04-005 (Config Struct) | TC-030 (implicitly) | -- |
| REQ-F04-006 (Template Dir) | TC-040, TC-041, TC-042 | INT-04 |
| REQ-F04-007 (Unified Path) | TC-025 | AS-F01 |
| REQ-F04-010 (Legacy Cache) | TC-070, TC-071 | -- |
| REQ-F04-011 (Error Context) | TC-050, TC-051 | -- |
| REQ-NF04-001 (Zero Breaking) | TC-010, TC-011 + `make test` | SC-3 |
| REQ-NF04-002 (Performance) | TC-080, TC-081 | -- |
| REQ-NF04-003 (Single Code Path) | TC-025 | -- |
| REQ-NF04-004 (Test Isolation) | All tests use `t.TempDir()` | -- |
| ADR-F04-004 (Task Block) | TC-060, TC-061, TC-062 | AS-D01, AS-D02, AS-D03 |

---

## 5. TDD Execution Guide

### Recommended Test-First Order

Developers should write tests in this order, which follows the implementation task sequence from the architecture doc:

**Task 1 -- Task workflow block parsing (TC-060, TC-061, TC-062)**
Write these first since they cover the prerequisite work (adding `task_workflow` block support to `LoadMultiLevelWorkflow()`). Red-green-refactor before moving to Task 2.

**Task 2 -- Workflow file loading and precedence (TC-001, TC-010, TC-020, TC-021, TC-022, TC-050, TC-052)**
Core feature tests. Start with TC-010 (no-change baseline to ensure refactoring does not regress), then TC-001 (happy path with workflow file), then precedence tests.

**Task 3 -- Configurable path and template directory (TC-030, TC-031, TC-033, TC-034, TC-040, TC-041)**
Path resolution tests. TC-030 first (happy path), then fallback/edge cases.

**Task 4 -- Performance and regression (TC-080, TC-081, TC-011, TC-025, TC-070)**
Non-functional tests. Run `make test` as the final regression gate.

### Running Tests

```bash
# Run all E20-F04 tests (convention: test names start with TestLoadMultiLevelWorkflow_)
go test ./internal/config/ -run "TestLoadMultiLevelWorkflow_" -v

# Run benchmarks
go test ./internal/config/ -run "^$" -bench "BenchmarkLoadMultiLevelWorkflow" -benchmem

# Full regression
make fmt && make lint && make test
```

---

*Last Updated*: 2026-03-18
*Author*: QA Agent (Claude Opus 4.6, 1M context)
