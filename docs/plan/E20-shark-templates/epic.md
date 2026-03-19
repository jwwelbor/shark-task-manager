---
epic_key: E20
title: Shark Templates
description: Externalize and standardize template and workflow configuration, extending the template engine foundation from E07-F30 and E07-F34.
---

# Shark Templates

**Epic Key**: E20

---

## Goal

### Problem

Shark's workflow configuration has grown to dominate `.sharkconfig.json`. The file is currently 1,759 lines, with workflow definitions for five entity types (epic, feature, task, bug, change) consuming the vast majority of that space. This workflow data sits alongside unrelated runtime settings such as database backend, authentication tokens, sync timestamps, and viewer preferences. The coupling creates three concrete problems:

1. **Risk of accidental edits.** A developer modifying `template_directory` or `database.url` must scroll past hundreds of lines of status-flow definitions. Any accidental keystroke in a workflow block can silently break status transitions for an entire entity type.

2. **Inconsistent task workflow structure.** Epic, feature, bug, and change workflows each use a dedicated `{entity}_workflow` block with versioned `status_flow`, `status_metadata`, `orchestrator_actions`, and `special_statuses` sub-keys. Task workflow, however, still uses legacy top-level keys (`status_flow`, `status_metadata`) that predate the per-entity pattern introduced in E16. This inconsistency forces the config loading code in `internal/config/manager.go` to handle two different structures, and makes it harder to write generic tooling that operates on "any entity workflow."

3. **No dedicated home for template configuration.** The template system built in E07-F30 and enriched with database-sourced variables in E07-F34 reads workflow metadata from `.sharkconfig.json`. The `template_directory` setting and the `shark-templates/` file tree (entity templates, status-based orchestrator prompts, partials) have no first-class configuration surface. This limits future capabilities such as template set sharing, per-project template overrides, or workflow presets.

### Solution

Introduce a dedicated workflow and template configuration file that separates workflow definitions from runtime settings:

- **New file: `.sharkworkflow.json`** (or configurable name) that contains all five `{entity}_workflow` blocks in a consistent per-entity structure, plus template-related settings (`template_directory`, template rendering options).
- **Standardize task workflow** to use the same `task_workflow` block pattern as other entities, eliminating the legacy top-level `status_flow` / `status_metadata` keys.
- **Reference from `.sharkconfig.json`** via a `workflow_config` key that points to the workflow file. When the key is absent or the file does not exist, the system falls back to reading workflow data from `.sharkconfig.json` itself, preserving backward compatibility.
- **Update config loading code** in `internal/config/` to support the two-file model with a clear precedence chain: workflow file (if present) takes priority over inline definitions in `.sharkconfig.json`.

### Impact

- `.sharkconfig.json` shrinks from 1,759 lines to approximately 50 lines of runtime settings, making it safe and quick to edit.
- All five entity types use an identical `{entity}_workflow` block structure, eliminating the task workflow special case.
- The template system gains a dedicated configuration surface, enabling future features such as workflow presets, template set sharing, and per-project overrides.
- Backward compatibility is preserved: projects that have not adopted the new file continue to work without changes.

---

## Business Value

**Rating**: Medium

This epic reduces configuration complexity and maintenance burden for anyone who modifies workflow definitions or template settings. The consistent per-entity structure simplifies the config loading code path (currently bifurcated for task vs. other entities) and removes a source of subtle bugs. It also establishes the foundation for higher-value future capabilities: workflow config sharing across projects, template marketplace concepts, and onboarding presets that ship as a single file rather than requiring manual `.sharkconfig.json` edits. The risk is low because the change is backward-compatible and the blast radius is limited to configuration loading, not core business logic.

---

## Epic Components

This epic is documented across multiple interconnected files:

- **[Requirements](./requirements.md)** - Functional and non-functional requirements catalog
- **[Scope Boundaries](./scope.md)** - Out of scope items, rejected alternatives, and future considerations

---

## Quick Reference

**Primary Users**: AI agents (template consumers), developers (config maintainers)

**Key Features**:
- Dedicated `.sharkworkflow.json` file for all workflow and template configuration
- Consistent `{entity}_workflow` block structure across all five entity types (epic, feature, task, bug, change)
- Task workflow standardization: migrate from legacy top-level keys to `task_workflow` block
- Backward-compatible fallback: system reads from `.sharkconfig.json` when workflow file is absent
- Configurable workflow file path via `workflow_config` key in `.sharkconfig.json`

**Related Work**:
- E07-F30: Template engine (completed) -- built the Go template rendering foundation
- E07-F34: Template variable enrichment (completed) -- added database-sourced variables to templates
- E16: Multi-level workflow (completed) -- introduced per-entity workflow blocks for epic/feature/bug/change
- E11: Configurable status workflow system (completed) -- established the config-driven workflow pattern
- CC-001: Original change-card proposal (declined -- scope too broad for a change-card)

**Success Criteria**:
- All entity workflows use consistent `{entity}_workflow` block structure in the new file
- `.sharkworkflow.json` is optional -- system works identically without it via fallback to `.sharkconfig.json`
- Existing commands, tests, and workflows continue to function without modification after migration
- `shark init` and `shark init update --workflow=` commands generate the new file format

**Current Features**:
- E20-F01: Template Configuration Externalization (cancelled -- absorbed into E20-F02)
- E20-F02: Enhancements and Maintenance (draft -- consolidates externalization and standardization work)

---

## Open Questions & Assumptions

No open questions -- all epic-level decisions are resolved.

**Resolved decisions:**
1. **File format**: JSON (`.sharkworkflow.json`), consistent with `.sharkconfig.json`. YAML was considered but rejected for consistency with the existing config ecosystem.
2. **Fallback behavior**: When `workflow_config` key is absent from `.sharkconfig.json`, the system reads workflow data inline from `.sharkconfig.json` itself. This preserves backward compatibility with zero migration effort for existing users.
3. **Task workflow migration**: The legacy top-level `status_flow` and `status_metadata` keys will be deprecated in favor of a `task_workflow` block. A migration path via `shark init update` will handle the conversion automatically.

---

*Last Updated*: 2026-03-18
