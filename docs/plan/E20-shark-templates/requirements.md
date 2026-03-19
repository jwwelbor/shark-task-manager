# Requirements

**Epic**: [Shark Templates](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for the Shark Templates epic (E20). The epic externalizes workflow and template configuration from `.sharkconfig.json` into a dedicated file, standardizes the task workflow block structure, and preserves backward compatibility.

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for launch; epic fails without these
- **Should Have**: Important but workarounds exist; target for initial release
- **Could Have**: Valuable but deferrable; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

### Must Have Requirements

#### Workflow File Introduction

**REQ-F-001**: Dedicated Workflow Configuration File
- **Description**: The system must support reading workflow and template configuration from a dedicated file (`.sharkworkflow.json`) located in the project root alongside `.sharkconfig.json`.
- **Acceptance Criteria**:
  - [ ] A `.sharkworkflow.json` file in the project root is detected and loaded by the config subsystem
  - [ ] The file contains all five entity workflow blocks: `epic_workflow`, `feature_workflow`, `task_workflow`, `bug_workflow`, `change_workflow`
  - [ ] The file also contains template-related settings: `template_directory` and any future template rendering options
  - [ ] The file follows the same JSON format conventions as `.sharkconfig.json`

**REQ-F-002**: Backward-Compatible Fallback
- **Description**: When no `.sharkworkflow.json` exists or when `.sharkconfig.json` does not contain a `workflow_config` reference, the system must fall back to reading workflow data from `.sharkconfig.json` itself with no change in behavior.
- **Acceptance Criteria**:
  - [ ] A project without `.sharkworkflow.json` behaves identically to the current system
  - [ ] All existing commands, status transitions, and template rendering continue to work without modification
  - [ ] No error or warning is emitted when the workflow file is absent
  - [ ] Existing `.sharkconfig.json` files with inline workflow blocks continue to be valid

**REQ-F-003**: Configuration Precedence Chain
- **Description**: When both the workflow file and `.sharkconfig.json` contain workflow definitions, the workflow file takes precedence.
- **Acceptance Criteria**:
  - [ ] If `.sharkworkflow.json` exists and contains `task_workflow`, that definition is used even if `.sharkconfig.json` also contains task workflow data
  - [ ] Precedence is per-entity: if the workflow file defines `epic_workflow` but not `bug_workflow`, the system reads `bug_workflow` from `.sharkconfig.json`
  - [ ] The precedence chain is: workflow file > `.sharkconfig.json` inline > built-in defaults

**REQ-F-004**: Configurable Workflow File Path
- **Description**: The path to the workflow file must be configurable via a `workflow_config` key in `.sharkconfig.json`.
- **Acceptance Criteria**:
  - [ ] `.sharkconfig.json` supports a top-level `workflow_config` key whose value is a file path (relative to project root or absolute)
  - [ ] When `workflow_config` is set, the system loads from that path instead of the default `.sharkworkflow.json`
  - [ ] When `workflow_config` is absent, the system checks for `.sharkworkflow.json` in the project root
  - [ ] If the configured path does not exist, the system falls back to `.sharkconfig.json` inline data with no error

#### Task Workflow Standardization

**REQ-F-005**: Consistent Task Workflow Block
- **Description**: Task workflow must use the same `task_workflow` block structure as epic, feature, bug, and change workflows, replacing the legacy top-level `status_flow` and `status_metadata` keys.
- **Acceptance Criteria**:
  - [ ] The workflow file contains a `task_workflow` block with `version`, `status_flow`, `status_metadata`, `orchestrator_actions`, and `special_statuses` sub-keys
  - [ ] The structure is identical in shape to `epic_workflow`, `feature_workflow`, etc.
  - [ ] Config loading code in `internal/config/manager.go` handles both legacy (top-level keys) and new (`task_workflow` block) formats during the transition period
  - [ ] When `task_workflow` block is present, legacy top-level keys are ignored for task workflow resolution

**REQ-F-006**: Automatic Migration via `shark init update`
- **Description**: The `shark init update --workflow=` command must generate the new workflow file format and optionally clean up inline workflow data from `.sharkconfig.json`.
- **Acceptance Criteria**:
  - [ ] `shark init update --workflow=advanced` generates a `.sharkworkflow.json` file with all five entity workflow blocks
  - [ ] `shark init update --workflow=basic` generates a `.sharkworkflow.json` file with the basic profile for all entity types
  - [ ] The generated file includes a `task_workflow` block (not legacy top-level keys)
  - [ ] `shark init update` with `--dry-run` previews the generated workflow file without writing it
  - [ ] Running `shark init update` on a project that already has `.sharkworkflow.json` updates it in place (with timestamped backup)

#### Config Loading Updates

**REQ-F-007**: Unified Config Loading Path
- **Description**: The config loading code must be updated to support the two-file model with a single, well-defined loading sequence.
- **Acceptance Criteria**:
  - [ ] `internal/config/manager.go` implements a loading sequence: (1) load `.sharkconfig.json` for runtime settings, (2) check for `workflow_config` key, (3) load workflow file if present, (4) fall back to inline workflow data
  - [ ] The `Config` struct in `internal/config/config.go` gains a `WorkflowConfig` field (pointer to string, omitempty)
  - [ ] `workflow.Service` initialization accepts workflow data from either source transparently
  - [ ] Template rendering in `internal/templates/orchestrator_renderer.go` reads template directory from the workflow file when available

---

### Should Have Requirements

#### Developer Experience

**REQ-F-010**: Config Validation for Workflow File
- **Description**: The `shark config validate` command should validate the workflow file in addition to `.sharkconfig.json`.
- **Acceptance Criteria**:
  - [ ] `shark config validate` checks `.sharkworkflow.json` for valid JSON structure if the file exists
  - [ ] Validation reports which entity workflows are defined in the workflow file vs. `.sharkconfig.json`
  - [ ] Validation warns about duplicate definitions (same entity workflow in both files)
  - [ ] Validation reports missing required sub-keys within each `{entity}_workflow` block

**REQ-F-011**: Config Show Includes Workflow Source
- **Description**: The `shark config show` command should indicate the source of each workflow configuration (workflow file vs. `.sharkconfig.json`).
- **Acceptance Criteria**:
  - [ ] Output includes a "Workflow Source" indicator for each entity type
  - [ ] When using JSON output (`--json`), each workflow block includes a `_source` metadata field

---

### Could Have Requirements

**REQ-F-020**: Legacy Key Deprecation Warnings
- **Description**: When legacy top-level `status_flow` and `status_metadata` keys are found in `.sharkconfig.json` alongside a `task_workflow` block (either inline or in the workflow file), emit a deprecation warning.
- **Acceptance Criteria**:
  - [ ] Warning is emitted once per CLI invocation, not per command
  - [ ] Warning message includes guidance: "Migrate task workflow to task_workflow block. Run `shark init update` to auto-migrate."
  - [ ] Warning is suppressed when `--json` output is active (to avoid polluting machine-readable output)

**REQ-F-021**: Workflow File Generation from Current Config
- **Description**: A `shark admin workflow export` command that extracts all workflow data from the current `.sharkconfig.json` into a new `.sharkworkflow.json` file.
- **Acceptance Criteria**:
  - [ ] Command reads all `{entity}_workflow` blocks plus legacy task keys from `.sharkconfig.json`
  - [ ] Command writes a complete `.sharkworkflow.json` with all five entity workflows
  - [ ] Task workflow is converted from legacy format to `task_workflow` block during export
  - [ ] Command supports `--dry-run` to preview without writing

---

## Non-Functional Requirements

### Backward Compatibility

**REQ-NF-001**: Zero-Breaking-Change Migration
- **Description**: Existing projects must continue to work without any configuration changes after upgrading to a version that supports the workflow file.
- **Measurement**: All existing integration tests pass without modification after the config loading changes
- **Target**: 100% backward compatibility -- no existing `.sharkconfig.json` file requires edits
- **Justification**: Shark is used by multiple projects and AI agent workflows; breaking config changes would disrupt active development

### Performance

**REQ-NF-002**: Config Loading Overhead
- **Description**: Loading configuration from two files must not introduce perceptible latency compared to the current single-file approach.
- **Measurement**: Benchmark config loading time before and after the change
- **Target**: Less than 5ms additional latency for the two-file path (JSON parse of workflow file)
- **Justification**: Every shark CLI command loads config; added latency multiplies across all operations

### Maintainability

**REQ-NF-003**: Single Config Loading Code Path
- **Description**: The config loading code must not have separate branches for "workflow file present" vs. "workflow file absent" scattered throughout the codebase. The resolution should happen once in the config layer and present a unified interface to consumers.
- **Measurement**: Only `internal/config/manager.go` (and optionally a new `workflow_loader.go`) contains file-source logic; all other packages receive a fully-resolved `Config` struct
- **Target**: Zero workflow-file-awareness in service, repository, CLI, or template packages
- **Justification**: Prevents the two-file model from creating widespread code complexity

### Testability

**REQ-NF-004**: Config Test Isolation
- **Description**: Tests for the config loading changes must not depend on the project's actual `.sharkconfig.json` or `.sharkworkflow.json`. All config loading tests must use in-memory or temporary file fixtures.
- **Measurement**: Config tests run successfully in a clean environment with no pre-existing config files
- **Target**: 100% of new config tests use test fixtures, not production config
- **Justification**: Prevents flaky tests that depend on developer-specific config state

---

*See also*: [Scope Boundaries](./scope.md)
