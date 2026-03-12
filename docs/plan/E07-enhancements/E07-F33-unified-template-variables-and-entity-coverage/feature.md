---
feature_key: E07-F33-unified-template-variables-and-entity-coverage
epic_key: E07
title: Unified Template Variables and Entity Coverage
description: Standardize template variables across all entity types and add bug/change-card template support
---

# Unified Template Variables and Entity Coverage

**Feature Key**: E07-F33

---

## Goal

### Problem

The template system has three issues:

1. **Inconsistent variable names**: Documentation says `{{.task_id}}`, `{{.epic_id}}`, `{{.feature_id}}` but code uses `{{.Key}}`, `{{.Epic}}`, `{{.Feature}}`. This confuses template authors.

2. **Different fields per entity type**: Tasks have `AgentType`, `Priority`, `DependsOn` while Epics/Features lack them. Features have `ExecutionOrder` while tasks don't. There's no common superset — template authors must know which entity they're targeting.

3. **No bug or change-card support**: The template system (`renderer.go`) and orchestrator action template population (`action_service.go`) only handle epics, features, and tasks. Bugs (`B###`) and change-cards (`CC-###`) have their own models, workflows, and statuses but cannot use templates.

### Solution

1. Create a **unified `EntityTemplateData` struct** with a common superset of fields available to ALL entity types (epic, feature, task, bug, change-card). Entity-specific fields are empty/zero when not applicable.
2. Add **`BugTemplateData`** and **`ChangeCardTemplateData`** population logic.
3. Update `action_service.go` `GetStatusActionPopulated()` to accept entity type and populate the full variable set (not just 4 vars all set to the same key).
4. Fix the **doc/code variable name mismatch** — align on canonical names.

### Impact

- Template authors can use one consistent variable set regardless of entity type
- Bugs and change-cards can use orchestrator action templates with full variable support
- Documentation matches actual code behavior

---

## Requirements

### Functional Requirements

**Category: Unified Template Data**

1. **REQ-F-001**: Unified EntityTemplateData Struct
   - **Description**: Create a single `EntityTemplateData` struct with ALL fields from all entity types. Fields not applicable to a given entity are empty/zero-valued.
   - **Priority**: Must-Have
   - **Fields (superset)**:
     - Common: `Key`, `Title`, `Description`, `Status`, `FilePath`, `Slug`, `CreatedAt`, `UpdatedAt`
     - Hierarchy: `EpicKey`, `FeatureKey`, `TaskKey` (populated based on entity and its parents)
     - Task-specific: `AgentType`, `Priority`, `DependsOn`, `ExecutionOrder`
     - Bug-specific: `Severity`, `LinkedEntityType`, `LinkedEntityKey`
     - Change-card-specific: `RequestedBy`, `AssignedTo`, `Justification`, `ImpactAnalysis`, `RollbackPlan`

2. **REQ-F-002**: Bug Template Population
   - **Description**: Add `PopulateBugTemplateData(bug *models.Bug) EntityTemplateData` function
   - **Priority**: Must-Have

3. **REQ-F-003**: Change-Card Template Population
   - **Description**: Add `PopulateChangeCardTemplateData(cc *models.ChangeCard) EntityTemplateData` function
   - **Priority**: Must-Have

4. **REQ-F-004**: Orchestrator Action Variable Expansion
   - **Description**: Update `GetStatusActionPopulated()` to accept entity type + entity data and populate full variable set instead of just 4 vars all set to the same key
   - **Priority**: Must-Have

5. **REQ-F-005**: Documentation Update
   - **Description**: Update `docs/cli-reference/template-system.md` to reflect actual variable names and document bug/change-card support
   - **Priority**: Must-Have

6. **REQ-F-006**: Breaking Change Documentation
   - **Description**: Document old→new variable name mappings so the 3-4 existing template usage sites can be updated. No backward-compatibility aliases — just replace.
   - **Priority**: Must-Have

---

## Out of Scope

1. **External .tmpl file support for bugs/change-cards** — E07-F30 handles the template engine; this feature only ensures bugs/change-cards get template variables populated
2. **New template files** — No new .tmpl files are created; this provides the data plumbing
3. **UI changes** — No CLI output changes

---

## Dependencies

- **E07-F30** (Template Engine): This feature builds on top of F30's template engine. F30 provides the rendering engine; F33 provides the data.
- **Bug model** (`internal/models/bug.go`): Already exists
- **Change-card model** (`internal/models/change_card.go`): Already exists

---

*Last Updated*: 2026-03-11
