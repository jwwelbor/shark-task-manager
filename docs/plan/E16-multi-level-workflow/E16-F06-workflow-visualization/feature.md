---
feature_key: E16-F06-workflow-visualization
epic_key: E16
title: Workflow Visualization
description: Update workflow list to show all three levels, planning vs aggregation distinction
priority: P3
---

# Workflow Visualization

**Feature Key**: E16-F06-workflow-visualization
**Priority**: P3

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

`shark workflow list` currently only shows the task workflow. With the addition of epic and feature workflows, users need a way to see all three workflow levels, understand the planning vs aggregation distinction, and visualize where entities are in their respective workflows.

### Solution

Extend `shark workflow list` to display all configured workflows (epic, feature, task). Visually distinguish planning statuses from aggregation statuses. Show the aggregation threshold clearly.

### Impact

- Users can see the full workflow landscape at a glance
- Clear visual distinction between planning and execution phases
- Better understanding of how the three levels relate

---

## User Stories

### Must-Have Stories

**Story 1**: As a user, I want `shark workflow list` to show all three workflow levels.

**Acceptance Criteria**:
- [ ] Epic workflow displayed (or "default" if using hardcoded)
- [ ] Feature workflow displayed (or "default" if using hardcoded)
- [ ] Task workflow displayed (existing behavior)
- [ ] Each level clearly labeled

**Story 2**: As a user, I want to see the aggregation threshold visually distinguished in workflow displays.

**Acceptance Criteria**:
- [ ] Aggregation statuses marked with `[brackets]` or similar
- [ ] Legend explains the notation: `[active] = aggregation threshold (progress derived from children)`

**Story 3**: As a user, I want `--json` output of `shark workflow list` to include all three levels.

**Acceptance Criteria**:
- [ ] JSON output includes `epic_workflow`, `feature_workflow`, `task_workflow` keys
- [ ] Each includes status list, flow, and metadata

---

### Should-Have Stories

**Story 4**: As a user, I want to see which workflow level I'm viewing when running `shark workflow list --level epic`.

**Acceptance Criteria**:
- [ ] `--level` flag filters to a specific workflow level
- [ ] Default shows all levels

---

## Requirements

### Functional Requirements

**Category: Workflow List Extension (FR-8)**

1. **REQ-F-001**: Show epic workflow in `shark workflow list`
   - **Description**: Display epic workflow statuses in order, with flow arrows
   - **Priority**: Must-Have

2. **REQ-F-002**: Show feature workflow in `shark workflow list`
   - **Description**: Display feature workflow statuses in order, with flow arrows
   - **Priority**: Must-Have

3. **REQ-F-003**: Distinguish planning vs aggregation statuses
   - **Description**: Aggregation statuses marked with `[brackets]`, planning statuses unmarked
   - **Priority**: Must-Have

4. **REQ-F-004**: Default vs custom label
   - **Description**: Show "(custom)" for configured workflows, "(default)" for hardcoded defaults
   - **Priority**: Must-Have

5. **REQ-F-005**: Legend/footer
   - **Description**: Include legend: `[active] = aggregation threshold (progress derived from children)`
   - **Priority**: Must-Have

6. **REQ-F-006**: `--level` filter
   - **Description**: Optional `--level` flag to show only one workflow level
   - **Priority**: Should-Have

7. **REQ-F-007**: JSON output
   - **Description**: `--json` returns structured workflow data for all three levels
   - **Priority**: Must-Have

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Full Workflow List**
- **Given** configured epic, feature, and task workflows
- **When** `shark workflow list`
- **Then** output shows all three levels with status flows and aggregation markers

**Scenario 2: Default Workflows**
- **Given** no custom epic or feature workflows configured
- **When** `shark workflow list`
- **Then** epic and feature show "(default)" with hardcoded statuses
- **And** task shows current workflow

**Scenario 3: Level Filter**
- **Given** all workflows configured
- **When** `shark workflow list --level feature`
- **Then** only feature workflow is displayed

---

## Out of Scope

1. **Interactive workflow editor** - Future enhancement
2. **Graphical workflow visualization** - Text-based only

---

## Dependencies & Integrations

### Dependencies

- **E16-F01**: Core Workflow Engine (provides multi-level workflow data)
- **E16-F03**: Display & Aggregation (provides `_aggregation_` and `is_planning` concepts)

---

*Last Updated*: 2026-02-08
