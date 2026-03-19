---
feature_key: E20-F04-workflow-file-loading-and-precedence
epic_key: E20
title: Workflow File Loading and Precedence
description: Introduce .sharkworkflow.json as a dedicated workflow configuration file with per-entity precedence over .sharkconfig.json inline data.
---

# Workflow File Loading and Precedence

**Feature Key**: E20-F04-workflow-file-loading-and-precedence

---

## Epic

- **Epic PRD**: [Shark Templates](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)

---

## Goal

### Problem

All workflow configuration (status flows, metadata, orchestrator actions, and special statuses for five entity types) is embedded inline in `.sharkconfig.json`, a 1,759-line file that also contains unrelated runtime settings (database backend, auth tokens, sync timestamps, viewer preferences). This coupling creates risk of accidental edits -- a developer modifying `database.url` must scroll past hundreds of lines of workflow definitions -- and prevents future capabilities like workflow preset sharing or per-project template overrides.

### Solution

Introduce a dedicated `.sharkworkflow.json` file that the config subsystem detects and loads alongside `.sharkconfig.json`. The workflow file contains all five `{entity}_workflow` blocks plus template-related settings (`template_directory`). A per-entity precedence chain resolves conflicts: workflow file definitions win over `.sharkconfig.json` inline definitions, which win over built-in defaults. The workflow file path is configurable via a `workflow_config` key in `.sharkconfig.json`. When no workflow file exists, the system falls back to reading from `.sharkconfig.json` with no change in behavior.

### Impact

- `.sharkconfig.json` shrinks from 1,759 lines to approximately 50 lines of runtime settings.
- Workflow definitions are isolated in a purpose-built file, reducing risk of accidental edits.
- Per-entity precedence enables gradual migration: move one entity workflow at a time.
- The `workflow_config` key enables non-standard file paths for projects with custom directory structures.
- Zero breaking changes: existing projects work identically without a workflow file.

---

## Epic Requirement Mapping

| Epic Requirement | Coverage |
|------------------|----------|
| **REQ-F-001** (Dedicated Workflow Configuration File) | Full |
| **REQ-F-002** (Backward-Compatible Fallback) | Full |
| **REQ-F-003** (Configuration Precedence Chain) | Full |
| **REQ-F-004** (Configurable Workflow File Path) | Full |
| **REQ-F-007** (Unified Config Loading Path) | Full |
| **REQ-NF-001** (Zero Breaking Changes) | Full |
| **REQ-NF-002** (Performance <5ms overhead) | Full |
| **REQ-NF-003** (Single Config Loading Code Path) | Full |
| **REQ-NF-004** (Test Isolation) | Partial -- tests for this feature use fixtures |

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want workflow configuration in a separate `.sharkworkflow.json` file so that I can edit runtime settings in `.sharkconfig.json` without risk of accidentally modifying workflow definitions.

**Acceptance Criteria**:
- [ ] A `.sharkworkflow.json` file in the project root is detected and loaded by the config subsystem
- [ ] The file contains all five entity workflow blocks: `epic_workflow`, `feature_workflow`, `task_workflow`, `bug_workflow`, `change_workflow`
- [ ] The file also supports `template_directory` and template rendering options
- [ ] The file follows JSON format consistent with `.sharkconfig.json`

**Story 2**: As a developer with an existing project, I want the system to work identically without a `.sharkworkflow.json` file so that I do not need to change anything to upgrade.

**Acceptance Criteria**:
- [ ] A project without `.sharkworkflow.json` behaves identically to the current system
- [ ] All existing commands, status transitions, and template rendering continue to work
- [ ] No error or warning is emitted when the workflow file is absent
- [ ] Existing `.sharkconfig.json` files with inline workflow blocks remain valid

**Story 3**: As a developer, I want the workflow file to take precedence over inline config on a per-entity basis so that I can migrate entity workflows one at a time.

**Acceptance Criteria**:
- [ ] If `.sharkworkflow.json` defines `task_workflow` but not `bug_workflow`, the system reads `task_workflow` from the file and `bug_workflow` from `.sharkconfig.json`
- [ ] Precedence chain is: workflow file > `.sharkconfig.json` inline > built-in defaults
- [ ] Per-entity precedence resolution happens once in the config layer, not scattered across consumers

**Story 4**: As a developer with a non-standard project layout, I want to configure the workflow file path so that I can place it in a custom location.

**Acceptance Criteria**:
- [ ] `.sharkconfig.json` supports a top-level `workflow_config` key whose value is a file path
- [ ] When `workflow_config` is set, the system loads from that path instead of the default `.sharkworkflow.json`
- [ ] When the configured path does not exist, the system falls back to inline data with no error

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Workflow file detected and loaded**
- **Given** both `.sharkconfig.json` (runtime settings only) and `.sharkworkflow.json` (all five entity workflows) exist
- **When** any shark command is run
- **Then** workflow definitions are loaded from `.sharkworkflow.json`
- **And** no warnings or errors about missing workflow data

**Scenario 2: Backward compatibility -- no workflow file**
- **Given** only `.sharkconfig.json` exists with inline workflow blocks
- **When** any shark command is run
- **Then** behavior is identical to pre-E20 system
- **And** no workflow-file-related messages on stderr

**Scenario 3: Per-entity precedence**
- **Given** `.sharkworkflow.json` defines `epic_workflow` and `task_workflow` only; `.sharkconfig.json` defines all five entity workflows
- **When** config is loaded
- **Then** epic and task use workflow file definitions; feature, bug, and change use inline definitions

**Scenario 4: Custom workflow file path**
- **Given** `.sharkconfig.json` contains `"workflow_config": "config/workflows.json"`
- **When** `config/workflows.json` exists with workflow data
- **Then** the system loads from that path

**Scenario 5: Missing configured path -- graceful fallback**
- **Given** `.sharkconfig.json` contains `"workflow_config": "nonexistent.json"`
- **When** any shark command is run
- **Then** the system falls back to inline data with no error

**Scenario 6: Performance**
- **Given** both config files exist
- **When** `LoadMultiLevelWorkflow()` is benchmarked
- **Then** additional latency from the workflow file is less than 5ms

---

## Out of Scope

1. **Generating `.sharkworkflow.json`** -- Handled by E20-F05. This feature only reads the file.
2. **Automatic removal of inline workflow data from `.sharkconfig.json`** -- See epic scope.md. The precedence chain makes coexistence safe.
3. **Workflow file validation** -- Structural validation is handled by E20-F06.
4. **YAML/TOML format support** -- Explicitly excluded in epic scope.

---

## Dependencies & Integrations

### Dependencies

- **E20-F03 (Task Workflow Standardization)**: Must be completed first. This feature assumes all five entity types use the `{entity}_workflow` block pattern, including task.
- **E16 (Multi-Level Workflow)**: Completed. Provides `MultiLevelWorkflow` struct and `parseWorkflowSection()`.

### Downstream Dependents

- **E20-F05 (Init Update)**: Generates the workflow file that this feature loads.
- **E20-F06 (Config Validation)**: Validates the workflow file that this feature introduces.

---

## Implementation Notes

### Key Files to Modify

- `internal/config/workflow_parser.go` -- Add workflow file detection, loading, and per-entity precedence resolution in `LoadMultiLevelWorkflow()`
- `internal/config/config.go` -- Add `WorkflowConfig *string` field to `Config` struct
- `internal/config/manager.go` -- Extract `workflow_config` key from raw JSON

### Estimated Scope

- ~80-120 lines of new code in `workflow_parser.go`, `config.go`, `manager.go`
- ~200-250 lines of new tests (two-file scenarios, precedence, fallback, cache behavior)
- Complexity: M (Medium)

### UAT Scenarios

Maps to UAT acceptance plan scenarios: AS-A01, AS-A02, AS-A03 (Area A), AS-B01, AS-B02, AS-B03 (Area B), AS-C01, AS-C02, AS-C03 (Area C), AS-F01 (Area F)

### Cache Strategy

The existing `multiLevelCache` keys on `configPath`. With two files, the cache key should incorporate both file paths. Since CLI commands are short-lived (one load per process), a simple dual-path key is sufficient. The existing `ClearWorkflowCache()` function already clears both caches.

---

## Data Contracts

### Workflow File Schema

The `.sharkworkflow.json` file uses the same JSON structure as the per-entity workflow blocks already defined in `.sharkconfig.json`. The top-level keys are:

```json
{
  "epic_workflow": { ... },
  "feature_workflow": { ... },
  "task_workflow": { ... },
  "bug_workflow": { ... },
  "change_workflow": { ... },
  "template_directory": "shark-templates"
}
```

Each `{entity}_workflow` block follows the existing `WorkflowConfig` struct shape:

```json
{
  "version": "1.0",
  "status_flow": { "<status>": ["<next_status>", ...] },
  "status_metadata": { "<status>": { "color": "...", "phase": "...", ... } },
  "orchestrator_actions": { "<status>": { "action": "...", ... } },
  "special_statuses": { "<group>": ["<status>", ...] }
}
```

All five entity workflow blocks are optional in the workflow file. Any block present in the workflow file takes precedence over the corresponding block in `.sharkconfig.json` for that entity type. Blocks absent from the workflow file fall through to `.sharkconfig.json`.

### Config Struct Addition

The `Config` struct in `config.go` gains one new field:

```go
WorkflowConfig *string `json:"workflow_config,omitempty"`
```

This field holds the path to the workflow file. When `nil`, the system checks for `.sharkworkflow.json` in the project root. When set, the system loads from the specified path. The path is resolved relative to the project root.

### LoadMultiLevelWorkflow Updated Signature

The function signature of `LoadMultiLevelWorkflow(configPath string)` does not change. Internally, the function:

1. Reads `.sharkconfig.json` at `configPath` (existing behavior).
2. Extracts `workflow_config` key from the raw JSON (new step).
3. Determines workflow file path: `workflow_config` value if present, otherwise `<projectRoot>/.sharkworkflow.json`.
4. If the workflow file exists, reads and parses it.
5. For each entity level, uses the workflow file definition if present, otherwise falls through to `.sharkconfig.json` inline definition.
6. Falls through to built-in defaults for any level still nil (existing behavior via `GetWorkflowForLevel()`).

The return type `*MultiLevelWorkflow` is unchanged. Consumers see no difference.

---

## Error Handling

### File Read Errors

| Scenario | Behavior |
|----------|----------|
| `.sharkworkflow.json` does not exist (default path) | Silent fallback to `.sharkconfig.json` inline data. No error, no warning. |
| Configured `workflow_config` path does not exist | Silent fallback to `.sharkconfig.json` inline data. No error, no warning. |
| `.sharkworkflow.json` exists but contains invalid JSON | Return error from `LoadMultiLevelWorkflow()`. `LoadMultiLevelWorkflowOrDefault()` logs warning to stderr and returns empty `MultiLevelWorkflow{}` (falling through to defaults). |
| `.sharkworkflow.json` exists but is unreadable (permission denied) | Return error with `os.ErrPermission` context. Same fallback behavior as invalid JSON via `LoadMultiLevelWorkflowOrDefault()`. |
| `.sharkworkflow.json` is empty file (0 bytes) | Treated as `{}` (empty JSON object). All entity levels nil, fall through to `.sharkconfig.json` inline data. |

### Parse Errors

| Scenario | Behavior |
|----------|----------|
| Workflow file has valid JSON but an `{entity}_workflow` block has invalid structure | Error returned with entity context: `"invalid epic_workflow in workflow file: ..."`. Other entity levels from the workflow file are not loaded (fail-fast for the entire file). |
| Workflow file has an unknown top-level key (e.g., `"sprint_workflow"`) | Ignored silently. Only recognized keys (`epic_workflow`, `feature_workflow`, `task_workflow`, `bug_workflow`, `change_workflow`, `template_directory`) are processed. |
| Both `.sharkworkflow.json` and `.sharkconfig.json` define `template_directory` | Workflow file value wins (consistent with entity-level precedence). |

### Error Message Format

All errors from the workflow file loading path include the file path for diagnosability:

```
failed to read workflow file /path/to/.sharkworkflow.json: <OS error>
invalid JSON in /path/to/.sharkworkflow.json at byte offset 42: <parse error>
invalid epic_workflow in /path/to/.sharkworkflow.json: failed to parse epic_workflow: <detail>
```

---

## Edge Cases

### Precedence Edge Cases

1. **Workflow file defines entity with empty object `{}`**: Treated as "not defined" (nil). The `.sharkconfig.json` inline definition is used for that entity. This matches existing behavior where `parseWorkflowSection()` returns nil for `{}`.

2. **Workflow file defines entity with `status_flow: {}` (empty map)**: Treated as "not defined" (nil). The `parseWorkflowSection()` function returns nil when `len(wf.StatusFlow) == 0`.

3. **`.sharkconfig.json` has no inline workflow data AND no workflow file exists**: All entity levels are nil. `GetWorkflowForLevel()` returns built-in defaults. This is the existing behavior for a fresh project.

4. **`workflow_config` points to `.sharkconfig.json` itself (circular reference)**: The system loads `.sharkconfig.json` twice (once for runtime settings, once as the "workflow file"). Entity workflow blocks are resolved from the same file. This is harmless but redundant. No special handling needed.

5. **`workflow_config` is an absolute path**: Supported. The path is used as-is. If relative, it is resolved relative to the directory containing `.sharkconfig.json` (the project root).

6. **`workflow_config` is an empty string `""`**: Treated as absent. The system checks for `.sharkworkflow.json` in the project root.

### Cache Edge Cases

7. **Two different commands in the same process load different config paths**: Not a realistic scenario for CLI (each process loads once), but the cache correctly keys on `configPath`. The workflow file path is derived from the config path, so different config paths produce different cache entries.

8. **Workflow file is modified between two `LoadMultiLevelWorkflow()` calls in the same process**: The cache returns the first load result. This is acceptable because CLI commands are short-lived. `ClearWorkflowCache()` can be called in tests to reset.

### Template Directory Edge Cases

9. **`template_directory` in workflow file but not in `.sharkconfig.json`**: The workflow file value is used.

10. **`template_directory` in `.sharkconfig.json` but not in workflow file**: The `.sharkconfig.json` value is used (workflow file does not override to empty).

11. **`template_directory` in neither file**: Default `"shark-templates"` is used (existing behavior via `GetTemplateDirectory()`).

---

## Cache Strategy Detail

### Current Cache Design

The existing cache uses two global variables with double-check locking:

- `multiLevelCache *MultiLevelWorkflow` / `multiLevelCachePath string` -- keyed on `configPath`
- `workflowCache *WorkflowConfig` / `workflowCachePath string` -- legacy single-level cache, populated from `result.Task`

### Updated Cache Design

With two files, the cache key must account for the workflow file path. The approach:

1. The `multiLevelCachePath` continues to key on `configPath` (the `.sharkconfig.json` path).
2. Because the workflow file path is deterministic from `configPath` (either `workflow_config` value or default `.sharkworkflow.json` in the same directory), a single `configPath` key is sufficient. If `configPath` changes, the entire cache is invalidated, and the new workflow file path is derived from the new config.
3. `ClearWorkflowCache()` already clears both caches. No changes needed.

This design is correct because:
- CLI commands are short-lived (one config load per process).
- The workflow file path is a function of the config file path and its contents.
- No hot-reloading is supported (out of scope per epic scope.md).

### Cache Invalidation in Tests

Tests that modify `.sharkworkflow.json` or `.sharkconfig.json` between assertions must call `ClearWorkflowCache()` before the next `LoadMultiLevelWorkflow()` call. This is the existing pattern used in `workflow_multilevel_test.go`.

---

*Last Updated*: 2026-03-18
