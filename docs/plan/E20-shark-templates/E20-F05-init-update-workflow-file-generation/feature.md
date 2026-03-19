---
feature_key: E20-F05-init-update-workflow-file-generation
epic_key: E20
title: Init Update Workflow File Generation
description: Update shark init and shark init update commands to generate .sharkworkflow.json with the new file format.
---

# Init Update Workflow File Generation

**Feature Key**: E20-F05-init-update-workflow-file-generation

---

## Epic

- **Epic PRD**: [Shark Templates](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)

---

## Goal

### Problem

The `shark init update --workflow=` command currently writes all workflow data inline into `.sharkconfig.json`, producing the 1,759-line monolithic file that E20 aims to separate. Additionally, the task workflow output uses legacy top-level keys (`status_flow`, `status_metadata`) rather than the standardized `task_workflow` block introduced by E20-F03. There is no mechanism to generate a standalone `.sharkworkflow.json` file.

### Solution

Update the `shark init update --workflow=advanced|basic` command to generate a `.sharkworkflow.json` file containing all five entity workflow blocks in the consistent `{entity}_workflow` format. The command writes workflow data to the workflow file and leaves `.sharkconfig.json` with only runtime settings. The `--dry-run` flag previews the generated file without writing it. Updating an existing `.sharkworkflow.json` creates a timestamped backup before overwriting.

### Impact

- Developers get a one-command migration path from the monolithic config to the separated two-file model.
- New projects (`shark init`) start with the clean separated format from day one.
- The generated `.sharkworkflow.json` serves as a reference for the correct file format.
- Timestamped backups ensure no data loss during updates.

---

## Epic Requirement Mapping

| Epic Requirement | Coverage |
|------------------|----------|
| **REQ-F-006** (Automatic Migration via `shark init update`) | Full |
| **REQ-NF-001** (Zero Breaking Changes) | Partial -- the command is opt-in; existing projects are not affected |

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want `shark init update --workflow=advanced` to generate a `.sharkworkflow.json` file so that I can adopt the separated config model with one command.

**Acceptance Criteria**:
- [ ] Running `shark init update --workflow=advanced` creates a `.sharkworkflow.json` file in the project root
- [ ] The file contains all five entity workflow blocks: `epic_workflow`, `feature_workflow`, `task_workflow`, `bug_workflow`, `change_workflow`
- [ ] `task_workflow` uses the block format (not legacy top-level keys)
- [ ] The file is valid JSON that passes `shark config validate`

**Story 2**: As a developer, I want `shark init update --workflow=basic` to generate a basic profile in the workflow file.

**Acceptance Criteria**:
- [ ] Running `shark init update --workflow=basic` creates `.sharkworkflow.json` with basic profile
- [ ] All five entity types have basic-level status flows

**Story 3**: As a developer, I want `--dry-run` to preview the workflow file without creating it so that I can review before committing.

**Acceptance Criteria**:
- [ ] `shark init update --workflow=advanced --dry-run` shows what would be written
- [ ] No file is created or modified when `--dry-run` is active

**Story 4**: As a developer updating an existing workflow file, I want a backup created automatically so that I do not lose my customizations.

**Acceptance Criteria**:
- [ ] Running `shark init update --workflow=advanced` on a project that already has `.sharkworkflow.json` creates a timestamped backup (e.g., `.sharkworkflow.json.backup.20260318-143022`)
- [ ] The backup contains the previous file contents

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Generate advanced workflow file**
- **Given** a project initialized with basic profile, no `.sharkworkflow.json`
- **When** `shark init update --workflow=advanced` is run
- **Then** `.sharkworkflow.json` is created with all five entity workflow blocks in advanced profile
- **And** `task_workflow` uses block format
- **And** `shark config validate` passes

**Scenario 2: Generate basic workflow file**
- **Given** a project with advanced profile
- **When** `shark init update --workflow=basic` is run
- **Then** `.sharkworkflow.json` contains basic profile for all entity types
- **And** status options reflect basic statuses

**Scenario 3: Dry run**
- **Given** no `.sharkworkflow.json` exists
- **When** `shark init update --workflow=advanced --dry-run` is run
- **Then** no file is created
- **And** preview output shows the workflow file contents

**Scenario 4: Update with backup**
- **Given** existing `.sharkworkflow.json` with basic profile
- **When** `shark init update --workflow=advanced` is run
- **Then** backup file is created with timestamp
- **And** new file has advanced profile content

---

## Out of Scope

1. **Removing inline workflow data from `.sharkconfig.json`** -- See epic scope.md. The precedence chain (workflow file wins) makes coexistence safe.
2. **`shark admin workflow export` command** -- Handled by E20-F06.
3. **Partial profile updates** -- The command regenerates all five entity workflows. Partial updates (e.g., only task workflow) are not supported.

---

## Dependencies & Integrations

### Dependencies

- **E20-F03 (Task Workflow Standardization)**: Must be completed. The generated file uses `task_workflow` block format.
- **E20-F04 (Workflow File Loading)**: Must be completed or concurrent. The generated file must be loadable by the new file-loading code.

### Downstream Dependents

- **E20-F06 (Config Validation)**: Validates the files generated by this feature.

---

## Implementation Notes

### Key Files to Modify

- `internal/cli/commands/init.go` -- Update `runInitUpdate()` to write workflow data to `.sharkworkflow.json` instead of (or in addition to) `.sharkconfig.json`
- Workflow profile generation functions -- Ensure task profile output uses `task_workflow` block format

### Estimated Scope

- ~50-80 lines of modified code in `init.go`
- ~80-120 lines of new tests (table-driven tests comparing generated output against known-good fixtures)
- Complexity: S (Small)

### UAT Scenarios

Maps to UAT acceptance plan scenarios: AS-E01, AS-E02, AS-E03, AS-E04 (Area E: Init Update)

---

*Last Updated*: 2026-03-18
