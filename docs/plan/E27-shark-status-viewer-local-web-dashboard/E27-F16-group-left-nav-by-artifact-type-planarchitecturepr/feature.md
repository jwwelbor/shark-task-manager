---
feature_key: E27-F16-group-left-nav-by-artifact-type-planarchitecturepr
epic_key: E27
title: Group Left Nav by Artifact Type (Plan/Architecture/Product) + Configurable Browsable Folders
description: Reorganize shark web's left nav into collapsible top-level groups mirroring known artifact locations (Plan, Architecture, Product), plus a .sharkconfig.json section for user-registered browsable folders.
---

# Group Left Nav by Artifact Type (Plan/Architecture/Product) + Configurable Browsable Folders

**Feature Key**: E27-F16-group-left-nav-by-artifact-type-planarchitecturepr

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
Today the `shark web` left nav is one flat tree of plan entities (epics → features → tasks, plus bugs/change-cards/tech-debt). It has no browsable view onto `docs/architecture/*` or product docs, and no way for a user to register their own project folders to browse from the sidebar. Users have to leave the dashboard and use a file browser to reach architecture/product documentation.

### Solution
Group the left nav into three top-level, independently collapsible sections that mirror where artifacts actually live:
- **Plan** — every tracked entity family: epic→feature→task hierarchy, change-cards, tech-debt, bugs, ideas, questions, and sprints.
- **Architecture** — `docs/architecture/*`, clickable and browsable.
- **Product** — product docs, clickable and browsable.

Each group remembers its expand/collapse state the same way the existing tree does, and participates in the existing expand-all/collapse-all control.

Separately, add a config section to `.sharkconfig.json` that lets a user register additional browsable folders in the left bar. Folder paths must be relative to the local shark project root, must reject `../` traversal, and must reuse shark's existing path-safety check (see `.claude/rules/go/input-sanitization.md` patterns) rather than introducing a new one.

### Impact
Makes `shark web` a single place to browse everything relevant to a project (plan entities + architecture + product docs + user-defined folders), removing the need to leave the dashboard for documentation.

---

## User Stories

### Must-Have Stories

**Story 1**: As a [user persona], I want to [perform an action] so that I can [achieve a benefit].

**Acceptance Criteria**:
- [ ] [Specific testable criterion 1]
- [ ] [Specific testable criterion 2]
- [ ] [Specific testable criterion 3]

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: [Requirement Title]
   - **Description**: [Clear, specific, testable requirement statement]
   - **Priority**: [Must-Have | Should-Have | Could-Have]
   - **Acceptance Criteria**:
     - [ ] [Specific criterion 1]
     - [ ] [Specific criterion 2]

### Non-Functional Requirements

1. **REQ-NF-001**: [Non-functional requirement]
   - **Description**: [Specific target or constraint]
   - **Measurement**: [How it will be measured]

---

## Acceptance Criteria

**Scenario 1: [Primary Use Case]**
- **Given** [initial context/state]
- **When** [user action is performed]
- **Then** [expected outcome]
- **And** [additional outcome]

---

## Out of Scope

1. **[Feature/Capability]**
   - **Why**: [Reasoning]
   - **Future**: [Whether this may be addressed later]

---

## Success Metrics

1. **[Metric Name]**
   - **What**: [What data point is tracked]
   - **Target**: [Specific goal]
   - **Measurement**: [How to measure]

---

*Last Updated*: 2026-08-10
