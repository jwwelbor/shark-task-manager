# Requirements: E20-F04 Workflow File Loading and Precedence

**Feature**: [Workflow File Loading and Precedence](./feature.md)
**Epic**: [Shark Templates](../epic.md)
**Epic Requirements**: [E20 Requirements](../requirements.md)

---

## Overview

This document defines the functional and non-functional requirements for E20-F04, which introduces `.sharkworkflow.json` as a dedicated workflow configuration file with per-entity precedence over `.sharkconfig.json` inline data. All requirements trace to parent epic requirements.

This feature handles **reading** the workflow file only. Generating the file is E20-F05. Validating the file is E20-F06.

---

## Functional Requirements

### Priority Framework

**MoSCoW prioritization** per epic conventions:
- **Must Have**: Feature cannot ship without these
- **Should Have**: Important, include if schedule permits
- **Could Have**: Deferrable to a follow-up
- **Won't Have**: Explicitly out of scope for this feature

---

### Must Have Requirements

#### REQ-F04-001: Workflow File Detection

**Traces to**: Epic REQ-F-001 (Dedicated Workflow Configuration File)

**Description**: `LoadMultiLevelWorkflow()` must detect and load a `.sharkworkflow.json` file in the project root directory (the directory containing `.sharkconfig.json`).

**Acceptance Criteria**:

- Given `.sharkworkflow.json` exists in the same directory as `.sharkconfig.json`
- When `LoadMultiLevelWorkflow(configPath)` is called
- Then the function reads and parses `.sharkworkflow.json`
- And entity workflow blocks defined in the file are populated into the `MultiLevelWorkflow` result

**Constraints**:
- Detection uses `os.ReadFile()` on the derived path (no directory scanning).
- The workflow file must be valid JSON. Invalid JSON causes `LoadMultiLevelWorkflow()` to return an error.
- Only the following top-level keys are recognized in the workflow file: `epic_workflow`, `feature_workflow`, `task_workflow`, `bug_workflow`, `change_workflow`, `template_directory`. Unknown keys are ignored silently.

---

#### REQ-F04-002: Backward-Compatible Fallback

**Traces to**: Epic REQ-F-002 (Backward-Compatible Fallback)

**Description**: When no `.sharkworkflow.json` exists, the system must behave identically to the pre-E20-F04 system with no errors, no warnings, and no behavioral changes.

**Acceptance Criteria**:

- Given only `.sharkconfig.json` exists with inline workflow blocks
- When any shark command is run
- Then behavior is identical to the pre-E20-F04 system
- And no workflow-file-related messages are emitted on stdout or stderr
- And `LoadMultiLevelWorkflow()` returns the same `MultiLevelWorkflow` result as before

**Verification**: Run the existing test suite (`make test`) with no `.sharkworkflow.json` present. All tests must pass without modification.

---

#### REQ-F04-003: Per-Entity Precedence Resolution

**Traces to**: Epic REQ-F-003 (Configuration Precedence Chain)

**Description**: When both `.sharkworkflow.json` and `.sharkconfig.json` contain workflow definitions, the workflow file takes precedence on a per-entity basis.

**Acceptance Criteria**:

AC-1: Per-entity override
- Given `.sharkworkflow.json` defines `epic_workflow` and `task_workflow`
- And `.sharkconfig.json` defines all five entity workflows
- When config is loaded
- Then `MultiLevelWorkflow.Epic` and `MultiLevelWorkflow.Task` contain workflow file definitions
- And `MultiLevelWorkflow.Feature`, `MultiLevelWorkflow.Bug`, and `MultiLevelWorkflow.Change` contain `.sharkconfig.json` inline definitions

AC-2: Full precedence chain
- The three-tier precedence chain is: workflow file > `.sharkconfig.json` inline > built-in defaults
- For each entity level independently:
  1. If the workflow file defines the entity workflow block with non-empty `status_flow`, use it.
  2. Else if `.sharkconfig.json` defines the entity workflow block (or legacy top-level keys for task), use it.
  3. Else `GetWorkflowForLevel()` returns the built-in default for that entity type.

AC-3: Resolution happens once
- Precedence resolution occurs inside `LoadMultiLevelWorkflow()` in `workflow_parser.go`.
- No other package in the codebase contains file-source awareness or precedence logic.

---

#### REQ-F04-004: Configurable Workflow File Path

**Traces to**: Epic REQ-F-004 (Configurable Workflow File Path)

**Description**: The path to the workflow file must be configurable via a `workflow_config` top-level key in `.sharkconfig.json`.

**Acceptance Criteria**:

AC-1: Custom path
- Given `.sharkconfig.json` contains `"workflow_config": "config/workflows.json"`
- And `config/workflows.json` exists with valid workflow data
- When config is loaded
- Then workflow definitions are read from `config/workflows.json`

AC-2: Missing custom path
- Given `.sharkconfig.json` contains `"workflow_config": "nonexistent.json"`
- When any shark command is run
- Then the system falls back to `.sharkconfig.json` inline data
- And no error or warning is emitted

AC-3: Absent key
- Given `.sharkconfig.json` does not contain a `workflow_config` key
- When config is loaded
- Then the system checks for `.sharkworkflow.json` in the project root (default path)

AC-4: Empty string value
- Given `.sharkconfig.json` contains `"workflow_config": ""`
- When config is loaded
- Then the system treats it as absent and checks for `.sharkworkflow.json` in the project root

AC-5: Absolute path
- Given `.sharkconfig.json` contains `"workflow_config": "/absolute/path/to/workflows.json"`
- When the absolute path exists
- Then the system loads from that absolute path

AC-6: Relative path resolution
- Relative paths in `workflow_config` are resolved relative to the directory containing `.sharkconfig.json` (the project root)

---

#### REQ-F04-005: Config Struct Field

**Traces to**: Epic REQ-F-007 (Unified Config Loading Path)

**Description**: The `Config` struct in `config.go` must include a `WorkflowConfig` field for the workflow file path.

**Acceptance Criteria**:

- The `Config` struct has a `WorkflowConfig *string` field with JSON tag `json:"workflow_config,omitempty"`
- `Manager.Load()` in `manager.go` extracts the `workflow_config` value from raw JSON and populates this field
- The field is nil when `workflow_config` is absent from `.sharkconfig.json`
- The field is ignored during config file writes (it is a read-only directive, not written back by `UpdateLastSyncTime()` or other mutators)

---

#### REQ-F04-006: Template Directory Precedence

**Traces to**: Epic REQ-F-001 (Dedicated Workflow Configuration File)

**Description**: The `template_directory` setting in the workflow file takes precedence over the same setting in `.sharkconfig.json`.

**Acceptance Criteria**:

AC-1: Workflow file wins
- Given `.sharkworkflow.json` contains `"template_directory": "custom-templates"`
- And `.sharkconfig.json` contains `"template_directory": "shark-templates"`
- When `GetTemplateDirectory()` is called
- Then it returns `"custom-templates"`

AC-2: Fallback to .sharkconfig.json
- Given `.sharkworkflow.json` does not contain `template_directory`
- And `.sharkconfig.json` contains `"template_directory": "shark-templates"`
- When `GetTemplateDirectory()` is called
- Then it returns `"shark-templates"`

AC-3: Fallback to default
- Given neither file contains `template_directory`
- When `GetTemplateDirectory()` is called
- Then it returns `"shark-templates"` (the `DefaultTemplateDir` constant)

---

#### REQ-F04-007: Unified Loading Path

**Traces to**: Epic REQ-F-007 (Unified Config Loading Path), Epic REQ-NF-003 (Single Config Loading Code Path)

**Description**: File-source logic (detecting, reading, and merging the workflow file) must be contained within `internal/config/workflow_parser.go`. No other package in the codebase may contain workflow file path detection or precedence logic.

**Acceptance Criteria**:

- Only `workflow_parser.go` (and optionally a new `workflow_loader.go` helper in the same package) contains file-existence checks for the workflow file
- `workflow.Service` initialization is unchanged: it calls `LoadMultiLevelWorkflowOrDefault(configPath)` and receives a fully-resolved `MultiLevelWorkflow`
- Service, repository, CLI command, and template packages have zero awareness of the two-file model
- `GetWorkflowForLevel()` continues to return `*WorkflowConfig` regardless of file source

---

### Should Have Requirements

#### REQ-F04-010: Legacy Cache Synchronization

**Traces to**: Epic REQ-NF-003 (Single Config Loading Code Path)

**Description**: The legacy single-level cache (`workflowCache`) must be correctly populated from the winning task workflow source (workflow file or inline) after precedence resolution.

**Acceptance Criteria**:

- After `LoadMultiLevelWorkflow()` completes, `workflowCache` contains the task workflow from whichever source won precedence
- `GetWorkflowOrDefault()` returns the same task workflow regardless of whether it came from the workflow file or `.sharkconfig.json`
- No consumer of the legacy API observes a difference after E20-F04

---

#### REQ-F04-011: Error Context in Workflow File Failures

**Traces to**: Epic REQ-NF-001 (Zero Breaking Changes -- diagnosability)

**Description**: Errors from the workflow file loading path must include the file path for diagnosability.

**Acceptance Criteria**:

- Error messages from workflow file read failures include the full file path
- Error messages from workflow file parse failures include the file path and byte offset (for JSON syntax errors)
- Error messages from entity block parse failures include the file path and entity name (e.g., `"invalid epic_workflow in /path/.sharkworkflow.json"`)

---

### Could Have Requirements

#### REQ-F04-020: Verbose Logging for File Detection

**Description**: When `--verbose` is enabled, log which workflow file path was checked and whether it was found.

**Acceptance Criteria**:

- With `--verbose`, a debug message indicates: `"Checking workflow file: /path/.sharkworkflow.json [found]"` or `"Checking workflow file: /path/.sharkworkflow.json [not found, using inline]"`
- Without `--verbose`, no file detection messages are emitted
- When a custom `workflow_config` path is used, the message includes the custom path

---

### Won't Have (Out of Scope for E20-F04)

| Item | Reason | Handled By |
|------|--------|------------|
| Generating `.sharkworkflow.json` | Write responsibility belongs to init commands | E20-F05 |
| Structural validation of workflow file contents | Validation is a separate concern | E20-F06 |
| Deprecation warnings for legacy task keys | Low priority, deferred | Epic Could Have REQ-F-020 |
| Automatic cleanup of inline workflow data | Risk of data loss; precedence makes coexistence safe | Epic scope.md |
| YAML/TOML format support | JSON only per epic decision | Epic scope.md |
| Hot-reloading of workflow file changes | CLI is short-lived; no use case | Epic scope.md |

---

## Non-Functional Requirements

### REQ-NF04-001: Zero Breaking Changes

**Traces to**: Epic REQ-NF-001

**Description**: All existing tests, commands, and workflows must continue to function without modification after this feature is implemented.

**Measurement**: `make test` passes with no test modifications. The existing `.sharkconfig.json` (with inline workflow blocks, no workflow file present) produces identical `MultiLevelWorkflow` results.

**Target**: 100% backward compatibility.

---

### REQ-NF04-002: Performance Budget

**Traces to**: Epic REQ-NF-002

**Description**: The additional file I/O from checking for and loading `.sharkworkflow.json` must not introduce perceptible latency.

**Measurement**: Benchmark `LoadMultiLevelWorkflow()` with and without a workflow file present.

**Target**: Less than 5ms additional latency for the two-file path. The worst case is one `os.ReadFile()` call (~1ms for a 1,500-line JSON file) plus one `json.Unmarshal()` call (~1-2ms).

**Verification**: Add a benchmark test `BenchmarkLoadMultiLevelWorkflow_TwoFiles` in `workflow_parser_test.go`.

---

### REQ-NF04-003: Single Code Path

**Traces to**: Epic REQ-NF-003

**Description**: File-source logic must not leak beyond `internal/config/`. The two-file model is an implementation detail of the config layer.

**Measurement**: Grep for `.sharkworkflow.json` or `workflow_config` outside of `internal/config/` returns zero results (excluding documentation and test fixtures).

**Target**: Zero workflow-file-awareness outside `internal/config/`.

---

### REQ-NF04-004: Test Isolation

**Traces to**: Epic REQ-NF-004

**Description**: All tests for E20-F04 must use temporary directories and fixture files. No test may depend on the project's actual `.sharkconfig.json` or `.sharkworkflow.json`.

**Measurement**: Tests create temp directories with `t.TempDir()`, write fixture files, and clean up automatically.

**Target**: 100% of new tests use test fixtures.

---

## Requirement Traceability Matrix

| Feature Requirement | Epic Requirement | Priority | Status |
|---------------------|-----------------|----------|--------|
| REQ-F04-001 (File Detection) | REQ-F-001 | Must Have | Draft |
| REQ-F04-002 (Backward Fallback) | REQ-F-002 | Must Have | Draft |
| REQ-F04-003 (Per-Entity Precedence) | REQ-F-003 | Must Have | Draft |
| REQ-F04-004 (Configurable Path) | REQ-F-004 | Must Have | Draft |
| REQ-F04-005 (Config Struct Field) | REQ-F-007 | Must Have | Draft |
| REQ-F04-006 (Template Dir Precedence) | REQ-F-001 | Must Have | Draft |
| REQ-F04-007 (Unified Loading Path) | REQ-F-007, REQ-NF-003 | Must Have | Draft |
| REQ-F04-010 (Legacy Cache Sync) | REQ-NF-003 | Should Have | Draft |
| REQ-F04-011 (Error Context) | REQ-NF-001 | Should Have | Draft |
| REQ-F04-020 (Verbose Logging) | -- | Could Have | Draft |
| REQ-NF04-001 (Zero Breaking Changes) | REQ-NF-001 | Must Have | Draft |
| REQ-NF04-002 (Performance Budget) | REQ-NF-002 | Must Have | Draft |
| REQ-NF04-003 (Single Code Path) | REQ-NF-003 | Must Have | Draft |
| REQ-NF04-004 (Test Isolation) | REQ-NF-004 | Must Have | Draft |

---

*Last Updated*: 2026-03-18
