---
feature_key: E20-F06-config-validation-and-developer-experience
epic_key: E20
title: Config Validation and Developer Experience
description: Extend config validation and display commands for the two-file model, add deprecation warnings.
---

# Config Validation and Developer Experience

**Feature Key**: E20-F06-config-validation-and-developer-experience

---

## Epic

- **Epic PRD**: [Shark Templates](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)

---

## Goal

### Problem

After E20-F03/F04/F05 introduce the two-file workflow model, developers need tooling to understand and manage the new configuration:
1. `shark config validate` only checks `.sharkconfig.json` -- it does not validate `.sharkworkflow.json`.
2. `shark config show` does not indicate whether a workflow definition came from the workflow file or from inline config.
3. There is no warning when deprecated legacy top-level task workflow keys coexist with the new `task_workflow` block, making it unclear which takes effect.
4. ~~There is no tool to extract workflow data from `.sharkconfig.json` into a standalone `.sharkworkflow.json` without running `shark init update` (which regenerates a profile rather than preserving customizations).~~ **(DESCOPED)**

### Solution

Extend the developer experience around the two-file model:
- **Config validation**: `shark config validate` checks `.sharkworkflow.json` for valid JSON structure, reports which entity workflows are defined in which file, and warns about duplicates.
- **Config show source**: `shark config show` indicates the source file for each entity workflow.
- **Deprecation warnings**: When legacy top-level task keys coexist with a `task_workflow` block, emit a one-time deprecation warning in human output (suppressed in `--json` mode).
- ~~**Workflow export**: A `shark admin workflow export` command that extracts workflow data from `.sharkconfig.json` into `.sharkworkflow.json`, converting task workflow from legacy format to block format.~~ **(DESCOPED -- see Out of Scope)**

### Impact

- Developers can validate the workflow file alongside the main config, catching structural errors early.
- Developers can quickly see where each entity workflow is defined, reducing confusion.
- Deprecation warnings guide developers toward the new block format.
- ~~The export command provides a customization-preserving migration path (vs. `shark init update` which regenerates from a profile).~~ **(DESCOPED)**

---

## Epic Requirement Mapping

| Epic Requirement | Coverage |
|------------------|----------|
| **REQ-F-010** (Config Validation for Workflow File) | Full |
| **REQ-F-011** (Config Show Includes Workflow Source) | Full |
| **REQ-F-020** (Legacy Key Deprecation Warnings) | Full |
| **REQ-F-021** (Workflow File Generation from Current Config) | **DESCOPED** -- Export command not implemented; `shark init update` covers migration. |

---

## User Stories

### Should-Have Stories

**Story 1**: As a developer, I want `shark config validate` to check `.sharkworkflow.json` so that I can catch structural errors in the workflow file.

**Acceptance Criteria**:
- [ ] `shark config validate` checks `.sharkworkflow.json` for valid JSON if the file exists
- [ ] Validation reports which entity workflows are defined in the workflow file vs. `.sharkconfig.json`
- [ ] Validation warns about duplicate definitions (same entity workflow in both files)
- [ ] Validation reports missing required sub-keys within each `{entity}_workflow` block

**Story 2**: As a developer, I want `shark config show` to indicate the source of each workflow so that I can understand my active configuration.

**Acceptance Criteria**:
- [ ] Human output includes a "Source" indicator per entity type (e.g., "epic_workflow: .sharkworkflow.json")
- [ ] JSON output includes a `_source` metadata field per workflow block

### Could-Have Stories

**Story 3**: As a developer, I want a deprecation warning when legacy task keys coexist with a `task_workflow` block so that I know to clean up the old format.

**Acceptance Criteria**:
- [ ] Warning is emitted once per CLI invocation
- [ ] Warning includes guidance: "Migrate task workflow to task_workflow block. Run `shark init update` to auto-migrate."
- [ ] Warning is suppressed when `--json` output is active

**Story 4 (DESCOPED)**: ~~As a developer with customized workflow config, I want `shark admin workflow export` to extract my current config into `.sharkworkflow.json` so that I can adopt the two-file model without losing my customizations.~~

> **DESCOPED**: The export command was removed from scope during implementation. The existing `shark init update --workflow=<profile>` command provides sufficient migration capability. A dedicated export command may be revisited in a future epic if demand warrants it.

~~**Acceptance Criteria**:~~
- ~~Command reads all `{entity}_workflow` blocks plus legacy task keys from `.sharkconfig.json`~~
- ~~Command writes a complete `.sharkworkflow.json` with all five entity workflows~~
- ~~Task workflow is converted from legacy format to `task_workflow` block during export~~
- ~~Command supports `--dry-run` to preview without writing~~

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Validate workflow file**
- **Given** `.sharkworkflow.json` exists with an intentional structural error
- **When** `shark config validate` is run
- **Then** the error is reported with the file name and offending field

**Scenario 2: Duplicate definition warning**
- **Given** `task_workflow` is defined in both `.sharkworkflow.json` and `.sharkconfig.json`
- **When** `shark config validate` is run
- **Then** a warning about duplicate definition is emitted

**Scenario 3: Config show with source**
- **Given** `.sharkworkflow.json` defines epic and task; `.sharkconfig.json` defines feature
- **When** `shark config show` is run
- **Then** output indicates source per entity type

**Scenario 4: Deprecation warning**
- **Given** both legacy top-level `status_flow` and `task_workflow` block present
- **When** `shark status` is run (human output)
- **Then** deprecation warning is emitted once
- **When** `shark status --json` is run
- **Then** no warning appears in output

**Scenario 5: Workflow export (DESCOPED)**
- ~~**Given** all workflow data is inline in `.sharkconfig.json` with legacy task keys~~
- ~~**When** `shark admin workflow export` is run~~
- ~~**Then** `.sharkworkflow.json` is created with all five entity workflows~~
- ~~**And** task workflow is in `task_workflow` block format~~

> **DESCOPED**: See Story 4 note above.

---

## Out of Scope

1. **Automatic removal of inline workflow data** -- Users manually clean up `.sharkconfig.json` after migration.
2. **Merge logic for conflicting definitions** -- Precedence chain resolves conflicts deterministically.
3. **REQ-F-021 Workflow Export Command** (`shark admin workflow export`) -- Descoped during implementation. The existing `shark init update --workflow=<profile>` command provides sufficient migration capability for moving to the two-file model. A dedicated customization-preserving export may be revisited in a future epic.

---

## Dependencies & Integrations

### Dependencies

- **E20-F04 (Workflow File Loading)**: Must be completed. This feature extends the validation and display of the two-file model introduced by F04.
- **E20-F03 (Task Workflow Standardization)**: Must be completed. Deprecation warnings and validation depend on the block format introduced by F03.

---

## Implementation Notes

### Key Files to Modify

- `internal/cli/commands/config.go` -- Extend validate and show subcommands
- `internal/config/workflow_parser.go` -- Add deprecation warning logic in the loading path
- ~~New: `internal/cli/commands/workflow_export.go` (or add to `admin.go`) -- Export command~~ **(DESCOPED)**

### Estimated Scope

- ~60-100 lines for validation extensions
- ~30-50 lines for config show source
- ~20-30 lines for deprecation warnings
- ~~~80-120 lines for export command~~ **(DESCOPED)**
- ~150-200 lines of tests
- Complexity: M (Medium)

### UAT Scenarios

Maps to UAT acceptance plan scenarios: AS-G01, AS-G02, AS-G03 (Area G), AS-H01 (Area H), AS-I01 (Area I)

---

*Last Updated*: 2026-03-18
