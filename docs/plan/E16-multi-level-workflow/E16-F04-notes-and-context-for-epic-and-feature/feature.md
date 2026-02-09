---
feature_key: E16-F04-notes-and-context-for-epic-and-feature
epic_key: E16
title: Notes and Context for Epic and Feature
description: Extend note and context commands to epic and feature entities
priority: P2
---

# Notes and Context for Epic and Feature

**Feature Key**: E16-F04-notes-and-context-for-epic-and-feature
**Priority**: P2

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

The note and context system currently only works at the task level. Agents working on epic-level research or feature-level refinement have no way to persistently record decisions, findings, blockers, or context. A BA refining a feature PRD needs to record decisions; a researcher analyzing market feasibility needs to log findings. Without this, context is lost between agent sessions.

### Solution

Extend the existing note and context commands to support epic and feature entities. Same note types, same context fields, same resume command -- just at higher entity levels.

### Impact

- Agents can record persistent context at all entity levels
- Context is preserved across agent sessions for epic/feature work
- `resume` commands provide full context for picking up planning work

---

## User Stories

### Must-Have Stories

**Story 1**: As an agent, I want to add notes to epics and features so that decisions and findings are recorded.

**Acceptance Criteria**:
- [ ] `shark epic note add <key> --type <type> "<content>"` works
- [ ] `shark feature note add <key> --type <type> "<content>"` works
- [ ] Same note types as tasks: `comment`, `decision`, `blocker`, `solution`, `reference`, `implementation`, `testing`, `future`, `question`

**Story 2**: As an agent, I want to set context on epics and features so that key-value metadata is persisted.

**Acceptance Criteria**:
- [ ] `shark epic context set <key> --field <field> --value "<value>"` works
- [ ] `shark feature context set <key> --field <field> --value "<value>"` works

**Story 3**: As an agent, I want `shark epic resume <key>` and `shark feature resume <key>` to get full context for resuming work.

**Acceptance Criteria**:
- [ ] Shows notes, context, history, current status, and workflow position
- [ ] Same format as existing `shark task resume`

---

## Requirements

### Functional Requirements

**Category: Notes for Epic/Feature (FR-10)**

1. **REQ-F-001**: Epic note add command
   - **Description**: `shark epic note add <key> --type <type> "<content>"`
   - **Priority**: Must-Have

2. **REQ-F-002**: Feature note add command
   - **Description**: `shark feature note add <key> --type <type> "<content>"`
   - **Priority**: Must-Have

3. **REQ-F-003**: Epic note list command
   - **Description**: `shark epic note list <key>` -- list all notes for an epic
   - **Priority**: Must-Have

4. **REQ-F-004**: Feature note list command
   - **Description**: `shark feature note list <key>` -- list all notes for a feature
   - **Priority**: Must-Have

**Category: Context for Epic/Feature**

5. **REQ-F-005**: Epic context set command
   - **Description**: `shark epic context set <key> --field <field> --value "<value>"`
   - **Priority**: Must-Have

6. **REQ-F-006**: Feature context set command
   - **Description**: `shark feature context set <key> --field <field> --value "<value>"`
   - **Priority**: Must-Have

7. **REQ-F-007**: Epic context get command
   - **Description**: `shark epic context get <key>` -- show all context fields
   - **Priority**: Must-Have

8. **REQ-F-008**: Feature context get command
   - **Description**: `shark feature context get <key>` -- show all context fields
   - **Priority**: Must-Have

**Category: Resume Command**

9. **REQ-F-009**: Epic resume command
   - **Description**: `shark epic resume <key>` -- full context dump for resuming work
   - **Priority**: Must-Have

10. **REQ-F-010**: Feature resume command
    - **Description**: `shark feature resume <key>` -- full context dump
    - **Priority**: Must-Have

---

### Non-Functional Requirements

1. **REQ-NF-001**: Database schema supports notes/context for all entity types
   - Notes table uses `entity_type` + `entity_id` pattern (or separate tables per entity)

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Add Note to Epic**
- **Given** epic E16 exists
- **When** `shark epic note add E16 --type decision "Chose config-driven approach over code-driven"`
- **Then** note is persisted and visible in `shark epic note list E16`

**Scenario 2: Add Context to Feature**
- **Given** feature E16-F01 exists
- **When** `shark feature context set E16-F01 --field "architect" --value "claude-architect-agent"`
- **Then** context field is persisted and visible in `shark feature context get E16-F01`

**Scenario 3: Resume Feature Work**
- **Given** feature E16-F01 has notes, context, and status history
- **When** `shark feature resume E16-F01`
- **Then** shows all notes, context, current status, workflow position, and history

---

## Out of Scope

1. **Note search/filtering across entities** - Future enhancement
2. **Note types specific to epic/feature** - Uses same types as tasks initially

---

## Dependencies & Integrations

### Dependencies

- **E16-F01**: Core Workflow Engine (provides workflow status/position context for resume command)
- **Existing note/context system**: Database schema and CLI patterns to extend

---

*Last Updated*: 2026-02-08
