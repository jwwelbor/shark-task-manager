---
feature_key: E20-F01-template-configuration-externalization
epic_key: E20
title: Template Configuration Externalization
description: Move workflow configuration from .sharkconfig.json into a dedicated .sharkworkflow.config file and standardize task_workflow to match other entity workflow structures.
related-docs:
  - docs/plan/changes/CC-001.md
---

# Template Configuration Externalization

**Feature Key**: E20-F01-template-configuration-externalization

---

## Epic

- **Epic**: [E20 - Shark Templates](../epic.md)

---

## Goal

### Problem

Currently, `.sharkconfig.json` mixes workflow/template configuration with runtime settings (database, viewer, sync). Task workflow uses legacy top-level `status_flow` and `status_metadata` keys while epic, feature, bug, and change workflows each have a dedicated `{entity}_workflow` block. This inconsistency complicates config loading and maintenance.

### Solution

1. Add a `workflow_config` key to `.sharkconfig.json` pointing to a new config file (default: `shark-templates/.sharkworkflow.config`)
2. The new file contains the same workflow settings using the existing structure — `{entity}_workflow` blocks with `status_flow` and `status_metadata` sub-keys
3. Standardize task workflow: create a `task_workflow` block matching the pattern used by `epic_workflow`, `feature_workflow`, `bug_workflow`, `change_workflow` — moving the current top-level `status_flow`, `status_metadata`, and `special_statuses` into this block
4. Backward compatible: if `workflow_config` is absent from `.sharkconfig.json`, fall back to reading workflow config from `.sharkconfig.json` as today

### Impact

- Consistent per-entity workflow structure across all 5 entity types
- Clean separation: `.sharkconfig.json` for runtime settings, `.sharkworkflow.config` for all workflow definitions
- No breaking changes — existing setups continue working without modification

---

## Current State (What Exists Today)

### .sharkconfig.json top-level keys (workflow-related):

```
epic_workflow:       { status_flow: {...}, status_metadata: {...} }
feature_workflow:    { status_flow: {...}, status_metadata: {...} }
bug_workflow:        { status_flow: {...}, status_metadata: {...} }
change_workflow:     { status_flow: {...}, status_metadata: {...} }
status_flow:         { ... }          ← task workflow (legacy top-level)
status_metadata:     { ... }          ← task workflow (legacy top-level)
special_statuses:    { ... }          ← task workflow (legacy top-level)
status_flow_version: "1.0"
```

### Target State (.sharkworkflow.config):

```json
{
  "status_flow_version": "1.0",
  "task_workflow": {
    "status_flow": { ... },
    "status_metadata": { ... },
    "special_statuses": { ... }
  },
  "epic_workflow": {
    "status_flow": { ... },
    "status_metadata": { ... }
  },
  "feature_workflow": {
    "status_flow": { ... },
    "status_metadata": { ... }
  },
  "bug_workflow": {
    "status_flow": { ... },
    "status_metadata": { ... }
  },
  "change_workflow": {
    "status_flow": { ... },
    "status_metadata": { ... }
  }
}
```

### .sharkconfig.json (after):

```json
{
  "workflow_config": "shark-templates/.sharkworkflow.config",
  "color_enabled": true,
  "database": { ... },
  "interactive_mode": false,
  "json_output": false,
  "last_sync_time": "...",
  "require_rejection_reason": true
}
```

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want workflow config in a separate file so that I can edit workflow definitions without risking changes to database or runtime settings.

**Acceptance Criteria**:
- [ ] `.sharkconfig.json` supports a `workflow_config` key pointing to the workflow config file
- [ ] All shark commands read workflow config from the external file when the key is present
- [ ] All shark commands fall back to `.sharkconfig.json` when the key is absent

**Story 2**: As a developer, I want task workflow to use the same `task_workflow` block structure as other entities so that config loading is consistent.

**Acceptance Criteria**:
- [ ] New config file has `task_workflow` block with `status_flow`, `status_metadata`, `special_statuses`
- [ ] Legacy top-level `status_flow`/`status_metadata`/`special_statuses` keys still work in fallback mode
- [ ] Config loading code handles both formats transparently

**Story 3**: As a developer, I want `shark init` to generate the new config file so that new projects get the externalized structure by default.

**Acceptance Criteria**:
- [ ] `shark init` creates `.sharkworkflow.config` in the template directory
- [ ] `shark init` sets the `workflow_config` key in `.sharkconfig.json`
- [ ] `shark init update --workflow=advanced|basic` updates the external config file

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Config Loading with External File
   - Config loader checks for `workflow_config` key in `.sharkconfig.json`
   - If present, loads workflow config from the referenced file
   - If absent, loads from `.sharkconfig.json` as today (full backward compat)

2. **REQ-F-002**: Task Workflow Standardization
   - New config file uses `task_workflow` block (not top-level keys)
   - Fallback mode supports both top-level keys (legacy) and `task_workflow` block
   - All workflow consumers (workflow.Service, status.CalculationService, template renderer) work with both formats

3. **REQ-F-003**: Init/Profile System Updates
   - `shark init` generates `.sharkworkflow.config` with all entity workflows
   - `shark init update --workflow=<profile>` updates the external file
   - Profile system writes to external file when `workflow_config` is set

4. **REQ-F-004**: Config Validation
   - `shark config validate` checks both files when `workflow_config` is set
   - Reports errors with file-specific context (which file has the issue)

### Non-Functional Requirements

1. **REQ-NF-001**: Zero Breaking Changes
   - Existing `.sharkconfig.json` files without `workflow_config` continue to work identically
   - No migration required — externalization is opt-in

2. **REQ-NF-002**: Performance
   - Config loading should not add measurable latency (single additional file read)

---

## Key Files to Modify

- `internal/config/config.go` — Add `TemplateConfig` field, update `GetTemplateDirectory()`
- `internal/config/manager.go` — Load external config file when `workflow_config` is set
- `internal/config/workflow_parser.go` — Support `task_workflow` block alongside legacy top-level keys
- `internal/workflow/service.go` — No changes expected (already receives parsed config)
- `internal/init/profile_service.go` — Write to external file when generating profiles
- `internal/cli/commands/init.go` — Create external config file during init
- `internal/cli/commands/config.go` — Validate both files

---

## Out of Scope

1. **Removing workflow config from .sharkconfig.json** — The legacy format remains supported indefinitely. No migration tool needed.
2. **Per-entity config files** — All workflows stay in one `.sharkworkflow.config` file. Splitting per entity type is a future consideration.
3. **Template file structure changes** — This feature only addresses config; template directory layout is unchanged.

---

## Dependencies

- E07-F30 (template engine) — completed, provides the template rendering foundation
- E07-F34 (template variable enrichment) — completed, provides enriched template variables
- Current workflow config structure in `.sharkconfig.json` — source of truth for migration

---

*Last Updated*: 2026-03-18
