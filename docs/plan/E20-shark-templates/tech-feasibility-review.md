# Technical Feasibility Review: E20 -- Shark Templates

**Review Date**: 2026-03-18
**Reviewer**: Architect Agent (Claude Opus 4.6, 1M context)
**Epic**: E20 -- Shark Templates
**Status at Review**: in_feasibility_review_tech

---

## Executive Summary

E20 proposes externalizing workflow and template configuration from `.sharkconfig.json` into a dedicated `.sharkworkflow.json` file and standardizing the task workflow block structure to match the other four entity types. After reviewing the epic PRD, requirements, research report, BA feasibility review, and performing independent codebase analysis of the config loading infrastructure (`internal/config/`), workflow service (`internal/workflow/`), and init commands, I confirm the epic is technically feasible with low risk.

The existing infrastructure provides strong foundations: `MultiLevelWorkflow` already holds all five entity levels, `parseWorkflowSection()` handles per-entity block parsing for four of five entity types, and the `workflow.Service` consumer layer is fully abstracted from the config source. The changes are additive (new file source), not structural (no new models, services, APIs, or database changes).

**Overall Assessment: APPROVED**

---

## (1) Technical Feasibility by Requirement Area

### Must Have Requirements

**REQ-F-001 (Dedicated Workflow Configuration File)** -- FEASIBLE

*Codebase verification*: `LoadMultiLevelWorkflow()` in `workflow_parser.go` (lines 166-263) currently reads a single file path and parses entity workflow blocks. Adding a second file source requires:

1. A file-existence check for `.sharkworkflow.json` (or configured path) before the current `.sharkconfig.json` parsing.
2. Calling the same `parseWorkflowSection()` function (line 281) on blocks from either file.
3. Merging results into the existing `MultiLevelWorkflow` struct (line 205).

The `parseWorkflowSection()` function is already entity-agnostic -- it accepts any `json.RawMessage` and returns a `*WorkflowConfig`. No changes to this function are needed.

*Estimated change*: ~40-60 lines added to `LoadMultiLevelWorkflow()`.

*Risk*: Low. The parsing logic is reused, not duplicated.

---

**REQ-F-002 (Backward-Compatible Fallback)** -- FEASIBLE

*Codebase verification*: The current code already handles missing config gracefully:
- `LoadMultiLevelWorkflow()` returns `&MultiLevelWorkflow{}` when the config file does not exist (line 188-191).
- `GetWorkflowForLevel()` in `workflow_multilevel.go` returns default workflows for nil levels (lines 22-51).
- `LoadMultiLevelWorkflowOrDefault()` catches all errors and returns an empty struct (lines 267-277).

The fallback chain (workflow file > `.sharkconfig.json` > built-in defaults) naturally extends this pattern. When `.sharkworkflow.json` does not exist, the loading flow falls through to `.sharkconfig.json` exactly as it does today.

*Estimated change*: Zero lines for the fallback itself -- the existing nil-means-default pattern handles it.

*Risk*: Negligible.

---

**REQ-F-003 (Configuration Precedence Chain)** -- FEASIBLE

*Codebase verification*: The `MultiLevelWorkflow` struct uses nullable pointers for each entity level. The precedence logic is:
1. Parse workflow file, populate levels that are defined there.
2. For any level still nil, parse from `.sharkconfig.json`.
3. `GetWorkflowForLevel()` fills in defaults for any remaining nil levels.

This is a straightforward two-pass fill. Per-entity precedence is achieved because each level is independently populated.

*Estimated change*: ~15-20 lines of nil-check-and-fill logic.

*Risk*: Low. The nullable pointer pattern already supports this.

---

**REQ-F-004 (Configurable Workflow File Path)** -- FEASIBLE

*Codebase verification*: The `Config` struct in `config.go` already supports adding new optional fields (pattern: pointer type with `omitempty` tag, lines 22-28). Adding `WorkflowConfig *string` follows the exact pattern used by `TemplateDirectory`, `Viewer`, and `InteractiveMode`.

The `Manager.Load()` method in `manager.go` (line 28) parses `rawData` and extracts known fields. Adding extraction for `workflow_config` follows the existing `template_directory` extraction pattern (line 78-79).

*Estimated change*: 5 lines (struct field + extraction + getter method).

*Risk*: Negligible.

---

**REQ-F-005 (Consistent Task Workflow Block)** -- FEASIBLE, MODERATE COMPLEXITY

*Codebase verification*: This is the most impactful change. Currently:
- Epic, feature, bug, change: Parsed via `parseWorkflowSection()` (lines 208-241).
- Task: Parsed via `parseTopLevelTaskWorkflow()` (lines 314-360), which reads legacy top-level keys (`status_flow`, `status_metadata`, etc.).

The `.sharkconfig.json` file currently has **zero** occurrences of `task_workflow` (confirmed by grep). All task workflow config is in legacy top-level keys.

The migration path:
1. Add `task_workflow` block parsing in `LoadMultiLevelWorkflow()` using the same `parseWorkflowSection()` call used for the other four entities (~5 lines).
2. Give `task_workflow` block precedence over legacy top-level keys (check `task_workflow` first, fall back to `parseTopLevelTaskWorkflow()` if nil).
3. Retain `parseTopLevelTaskWorkflow()` as fallback for existing configs.

*Key observation*: The legacy single-level cache (`workflowCache`) is populated from `result.Task` regardless of source (lines 251-256). This means consumers of the legacy `LoadWorkflowConfig()` API will automatically receive the task workflow from whichever source wins the precedence check, with no additional changes needed.

*Estimated change*: ~15 lines in `LoadMultiLevelWorkflow()`.

*Risk*: Moderate. The legacy cache synchronization (lines 251-256) must remain correct. The existing `ClearWorkflowCache()` function (lines 136-146) already clears both caches, which is sufficient for two-file scenarios.

**Specific concern**: The `LoadWorkflowConfig()` function (lines 39-132) is a completely independent code path that reads `.sharkconfig.json` directly and looks for top-level `status_flow`. It does **not** delegate to `LoadMultiLevelWorkflow()`. If any consumer calls `LoadWorkflowConfig()` directly (bypassing the multi-level path), they will not see `task_workflow` blocks or workflow file data. However, the primary API `GetWorkflowOrDefault()` (line 150) already delegates to `LoadMultiLevelWorkflowOrDefault()`, which is the multi-level path. Auditing the 16 files that reference these functions confirms the multi-level path is the standard entry point. The legacy `LoadWorkflowConfig()` is only called indirectly via `GetWorkflowOrDefault()`.

---

**REQ-F-006 (Automatic Migration via `shark init update`)** -- FEASIBLE

*Codebase verification*: The init update command already generates complete workflow profiles for all five entity types. The change is primarily **where** the output is written (`.sharkworkflow.json` vs. inline in `.sharkconfig.json`). The backup logic (timestamped backups) already exists in the init update code path.

The task workflow output currently uses legacy top-level keys. Converting to `task_workflow` block format is a straightforward restructuring of the JSON output -- the data content is identical, only the JSON nesting changes.

*Estimated change*: ~50-80 lines in the init command to write the new file format.

*Risk*: Low. Table-driven tests comparing generated output against known-good fixtures will catch format errors.

---

**REQ-F-007 (Unified Config Loading Path)** -- FEASIBLE

*Codebase verification*: The `MultiLevelWorkflow` struct already serves as the abstraction boundary. Consumers (workflow.Service, status.CalculationService, template renderer) call `GetWorkflowForLevel()` and receive a `*WorkflowConfig` without knowing the source. The only file that needs file-source awareness is `workflow_parser.go`, satisfying REQ-NF-003 (single code path).

The `workflow.Service` in `internal/workflow/service.go` (lines 38-48) creates itself by calling `config.LoadMultiLevelWorkflowOrDefault(configPath)` and then `multi.GetWorkflowForLevel()`. This pattern requires no changes -- the multi-level loader will transparently handle the two-file model.

*Estimated change*: Zero lines outside `workflow_parser.go` and `config.go`.

*Risk*: Low.

---

### Should Have and Could Have Requirements

**REQ-F-010 (Config Validation)** -- FEASIBLE, Low Effort. The existing `workflow_validator.go` provides structural validation that can be applied to either file. Extending `shark config validate` to check `.sharkworkflow.json` requires loading the workflow file and running the same validation functions.

**REQ-F-011 (Config Show Source)** -- FEASIBLE, Low Effort. Adding a `_source` metadata field is a display-layer change in the config show command.

**REQ-F-020 (Deprecation Warnings)** -- FEASIBLE, Trivial. A single check in `LoadMultiLevelWorkflow()` for co-existence of `task_workflow` block and legacy top-level keys.

**REQ-F-021 (Workflow Export Command)** -- FEASIBLE, Low Effort. A new CLI command that reads existing config and writes `.sharkworkflow.json`. Follows the standard command pattern (parse args, call service/helper, format output).

---

### Non-Functional Requirements

**REQ-NF-001 (Zero Breaking Changes)** -- ACHIEVABLE. Confirmed by codebase analysis: the fallback chain preserves all existing behavior. No consumer code needs changes.

**REQ-NF-002 (Performance <5ms overhead)** -- ACHIEVABLE. JSON parsing a ~1,500-line file takes 1-2ms on modern hardware. The existing cache (`multiLevelCache`) means the overhead is only on the first invocation per process. Shark CLI commands are short-lived processes, so each invocation loads config once. The second file adds at most one additional `os.ReadFile()` + `json.Unmarshal()` call, well within the 5ms budget.

**REQ-NF-003 (Single Config Loading Code Path)** -- ACHIEVABLE. Only `workflow_parser.go` needs file-source logic. Confirmed: no other package references file paths for workflow loading.

**REQ-NF-004 (Test Isolation)** -- ACHIEVABLE. Existing tests in `workflow_test.go`, `workflow_multilevel_test.go`, and `workflow_metadata_test.go` all use temporary files and in-memory fixtures. The same patterns extend naturally to two-file scenarios.

---

## (2) Architectural Concerns Assessment

### Integration Complexity

**Finding: LOW.**

The change is confined to a single layer (config loading) with a well-defined abstraction boundary (`MultiLevelWorkflow` struct). No cross-layer changes are needed. The `workflow.Service`, status calculation, template rendering, and all CLI commands interact only with the abstracted `*WorkflowConfig`, never with file paths or file-source logic.

### Performance Risks

**Finding: NONE.**

The worst case adds one file read and one JSON parse per CLI invocation. With caching already in place, this is a one-time cost of ~2ms. The cache invalidation concern raised in the research report (two files, stale cache) is mitigated by the existing `ClearWorkflowCache()` function and the fact that CLI commands are short-lived (config is loaded once per process, not reloaded mid-execution).

### Scaling Issues

**Finding: NOT APPLICABLE.**

This epic addresses configuration management, not runtime scalability. The workflow file is read once at startup and cached. The file size is bounded (workflow definitions for five entity types, each with ~200-300 lines of JSON, totaling ~1,500 lines max). There is no scenario where the workflow file grows unboundedly.

---

## (3) Dependency and Integration Risk Assessment

### Cross-Epic Dependencies

| Epic | Interaction | Risk |
|------|------------|------|
| E21 (Entity Polymorphism) | SYNERGISTIC. E20's consistent `{entity}_workflow` structure removes the task special case that would complicate E21's generic entity service. | None. E20 should complete first. |
| E19 (Sprint Management) | SUPPORTIVE. If sprints need `sprint_workflow`, E20's externalized file is the natural home. | None. No dependency. |
| E15 (Service Layer Refactoring) | COMPATIBLE. `workflow.Service` initialization already abstracted from config source. | None. |
| E16 (Multi-Level Workflow) | ALIGNED. E20 extends E16's `parseWorkflowSection()` to cover task workflow. | None. |
| E11 (Configurable Status Workflow) | ALIGNED. E20 preserves E11's config-driven pattern while separating physical storage. | None. |

### External Dependencies

**Finding: NONE.** All technologies are Go standard library (`encoding/json`, `os`, `sync`). No new external packages required.

### Cross-Epic Technical Conflicts

**Finding: NONE.** E20 modifies only `internal/config/` (workflow_parser.go, config.go, manager.go) and `internal/cli/commands/init.go`. No other epic currently modifies these files. The git branch `file-path-updates` has changes to `internal/config/template_helpers.go` and `internal/config/template_helpers_test.go`, but these are in the template helpers module, not the workflow parser module. No merge conflicts expected.

---

## (4) Technical Debt Assessment

### Will E20 Create Technical Debt?

**Finding: NET REDUCTION.**

E20 **reduces** technical debt in two specific ways:

1. **Eliminates the task workflow bifurcation.** Currently, `LoadMultiLevelWorkflow()` has two parsing paths: `parseWorkflowSection()` for epic/feature/bug/change, and `parseTopLevelTaskWorkflow()` for task. After E20, all five entities use the same `parseWorkflowSection()` function via `{entity}_workflow` blocks. The `parseTopLevelTaskWorkflow()` function becomes a backward-compatibility shim that can be deprecated and eventually removed.

2. **Separates concerns in configuration.** The current 1,759-line `.sharkconfig.json` mixes runtime settings (database, viewer, sync timestamps) with workflow definitions (status flows, metadata, orchestrator actions for five entity types). After E20, runtime settings remain in `.sharkconfig.json` (~50 lines) and workflow definitions move to `.sharkworkflow.json`. This separation reduces the risk of accidental edits and makes each file easier to understand and maintain.

### Will E20 Exacerbate Existing Technical Debt?

**Finding: NO.** The changes are additive (new file source, new config field) and do not add complexity to existing consumer code. The abstraction boundary (`MultiLevelWorkflow.GetWorkflowForLevel()`) shields all consumers from the two-file model. The temporary coexistence of legacy top-level task keys and the new `task_workflow` block is managed by a clear precedence rule (block wins over legacy), and the legacy path can be removed in a future cleanup.

---

## (5) Implementation Recommendations

### Recommended Implementation Order

1. **Task workflow standardization (REQ-F-005)**: Add `task_workflow` block parsing to `LoadMultiLevelWorkflow()`. This is the highest-value change and can be validated independently by adding a `task_workflow` block to `.sharkconfig.json` alongside the existing legacy keys.

2. **Workflow file loading (REQ-F-001 + REQ-F-003 + REQ-F-004)**: Modify `LoadMultiLevelWorkflow()` to check for an external file. This builds on step 1 because the task workflow block is now consistent.

3. **Backward compatibility verification (REQ-F-002)**: Run the full test suite with both configurations (with and without `.sharkworkflow.json`).

4. **Init update changes (REQ-F-006 + REQ-F-007)**: Update `shark init update` to generate `.sharkworkflow.json`.

5. **Should Have and Could Have**: Validation, show source, deprecation warnings, export command.

### Testing Strategy

- Extend `workflow_multilevel_test.go` with two-file test scenarios (workflow file present, absent, partial, conflicting).
- Add a regression test that loads the current `.sharkconfig.json` and verifies workflow resolution matches the pre-E20 behavior exactly.
- Performance benchmark: measure config loading time before and after (target: <5ms additional).
- Use temporary directories for all test fixtures (following existing patterns in `workflow_test.go`).

### Cache Strategy for Two-File Model

The current cache keys on `configPath` (a single string). With two files, the cache should be invalidated when either file changes. Two options:

1. **Simple**: Key cache on both paths concatenated. When `LoadMultiLevelWorkflow()` is called, compare both the `.sharkconfig.json` path and the workflow file path against the cached values. This is the recommended approach given that CLI commands are short-lived (one load per process).

2. **Content-based**: Key cache on a hash of both file contents. This is more robust but adds unnecessary complexity for CLI commands that load config once.

Recommendation: Option 1 (path-based, two keys).

---

## (6) Risk Summary

| Risk | Probability | Impact | Mitigation | Assessment |
|------|------------|--------|------------|------------|
| Regression in workflow loading for any entity | Medium | High | Existing test suite + new two-file tests + regression test against current `.sharkconfig.json` | MANAGEABLE |
| Cache stale with two files | Medium | Medium | Dual-path cache key; `ClearWorkflowCache()` already clears both caches | MANAGEABLE |
| Init update generates incorrect format | Low | Medium | Table-driven tests with known-good fixtures | MANAGEABLE |
| Legacy `LoadWorkflowConfig()` callers bypass two-file path | Low | Medium | Audit confirms all callers go through `GetWorkflowOrDefault()` -> multi-level path | MANAGEABLE |
| Template directory resolution breaks if moved to workflow file | Low | Medium | Fallback chain ensures setting found in either file; `GetTemplateDirectory()` on Config struct unchanged | MANAGEABLE |

**No showstopper risks identified.**

---

## Conclusion

E20 is technically feasible with low risk and moderate effort (~400-600 lines of new/modified code). The existing infrastructure provides strong foundations: `MultiLevelWorkflow` for multi-entity support, `parseWorkflowSection()` for entity-agnostic block parsing, nullable pointer pattern for fallback semantics, and double-check locking cache for thread-safe config loading.

The changes are confined to the config loading layer (`internal/config/`) and the init command (`internal/cli/commands/init.go`). No database changes, no new services, no new API endpoints, and no changes to consumer code. The abstraction boundary (`MultiLevelWorkflow.GetWorkflowForLevel()`) shields all consumers from the two-file model.

The epic reduces technical debt by eliminating the task workflow bifurcation and separating concerns in configuration files. It is synergistic with E21 (Entity Polymorphism) and supportive of E19 (Sprint Management).

**Assessment: APPROVED**

The epic should advance to `ready_for_tech_check`.

---

*Review completed: 2026-03-18*
*Reviewer: Architect Agent (Claude Opus 4.6, 1M context)*
