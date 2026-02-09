---
feature_key: E16-F02-orchestrator-actions
epic_key: E16
title: Orchestrator Actions
description: Return orchestrator_action in epic/feature transition responses, extend show-actions
priority: P1
---

# Orchestrator Actions

**Feature Key**: E16-F02-orchestrator-actions
**Priority**: P1

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

The `orchestrator_action` mechanism (implemented for tasks in E13) tells the orchestrator what agent to spawn when a task reaches a given status. Epics and features have no equivalent, so orchestration tools like `/build` and `/feature` must hardcode all dispatch logic. This creates a tight coupling between orchestration skills and entity lifecycle management.

### Solution

Include `orchestrator_action` in status metadata for epic and feature workflows, returned in transition responses for `update --status` and `next-status` commands. Extend `shark workflow show-actions` to display actions for all three entity levels.

### Impact

- Orchestrator becomes fully config-driven at all levels
- No hardcoded dispatch logic needed in orchestration skills
- Foundation for autonomous pipelines: epic decomposition, feature refinement, and task execution can each be automated

---

## User Stories

### Must-Have Stories

**Story 1**: As an orchestrator, I want transition responses to include `orchestrator_action` when defined so that I know what agent to spawn next.

**Acceptance Criteria**:
- [ ] `shark epic update --status` response includes `orchestrator_action` when defined in config
- [ ] `shark feature update --status` response includes `orchestrator_action` when defined
- [ ] `shark epic next-status` response includes `orchestrator_action`
- [ ] `shark feature next-status` response includes `orchestrator_action`
- [ ] `{id}` placeholder in `instruction_template` is replaced with entity key

**Story 2**: As an orchestrator, I want `shark workflow show-actions` to display actions for all three levels so I can see the full dispatch map.

**Acceptance Criteria**:
- [ ] Output shows Epic Workflow Actions, Feature Workflow Actions, and Task Workflow Actions
- [ ] Each section lists status -> action type (agent type)
- [ ] `--json` output includes all three levels in structured format

**Story 3**: As a developer, I want `shark workflow validate-actions` to validate orchestrator actions for all three levels.

**Acceptance Criteria**:
- [ ] Validates action types are from allowed set: `spawn_agent`, `pause`, `wait_for_triage`, `archive`
- [ ] Validates `agent_type` is present when action is `spawn_agent`
- [ ] Reports per-level validation results

---

## Requirements

### Functional Requirements

**Category: Orchestrator Action in Transitions (FR-5)**

1. **REQ-F-001**: Return `orchestrator_action` in epic transition responses
   - **Description**: When `epic_workflow.status_metadata.<status>.orchestrator_action` is defined, include it in the JSON response of `update --status` and `next-status`
   - **Priority**: Must-Have

2. **REQ-F-002**: Return `orchestrator_action` in feature transition responses
   - **Description**: Same as REQ-F-001 but for `feature_workflow`
   - **Priority**: Must-Have

3. **REQ-F-003**: Template variable replacement
   - **Description**: Replace `{id}` in `instruction_template` with the entity key (e.g., `E16`, `E16-F01`)
   - **Priority**: Must-Have

4. **REQ-F-004**: Action types
   - **Description**: Support same action types as task level: `spawn_agent`, `pause`, `wait_for_triage`, `archive`
   - **Priority**: Must-Have

**Category: Show-Actions Extension**

5. **REQ-F-005**: Extend `shark workflow show-actions` for all levels
   - **Description**: Display epic, feature, and task workflow actions in organized sections
   - **Priority**: Must-Have

6. **REQ-F-006**: JSON output for show-actions
   - **Description**: `--json` returns structured output with actions grouped by entity level
   - **Priority**: Must-Have

**Category: Validate-Actions Extension**

7. **REQ-F-007**: Extend `shark workflow validate-actions` for all levels
   - **Description**: Validate orchestrator_action entries in epic and feature workflow configs
   - **Priority**: Must-Have

---

### Non-Functional Requirements

1. **REQ-NF-001**: Action resolution adds < 5ms to transition commands

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Epic Transition with Action**
- **Given** an epic at `draft` with `ready_for_research` having an orchestrator_action
- **When** `shark epic next-status E16 --json`
- **Then** JSON includes `orchestrator_action` with `action: "spawn_agent"`, `agent_type: "researcher"`, and resolved `instruction`

**Scenario 2: Feature Transition with Action**
- **Given** a feature at `in_refinement_ba` transitioning to `ready_for_refinement_tech`
- **When** `shark feature next-status E16-F01 --json`
- **Then** JSON includes `orchestrator_action` with `agent_type: "architect"` and instruction containing "E16-F01"

**Scenario 3: Show-Actions All Levels**
- **Given** configured epic, feature, and task workflows
- **When** `shark workflow show-actions`
- **Then** output displays three sections with action mappings for each level

**Scenario 4: No Action Defined**
- **Given** a status with no `orchestrator_action` in metadata
- **When** transition occurs
- **Then** `orchestrator_action` is `null` in JSON response

---

## Out of Scope

1. **Automatic agent spawning** - Shark returns the action; the orchestrator executes it
2. **Cross-level action dependencies** - Actions are independent per level

---

## Dependencies & Integrations

### Dependencies

- **E16-F01**: Core Workflow Engine (must be complete - provides workflow parsing and transition infrastructure)
- **E13**: Existing task-level orchestrator action implementation (pattern to extend)

---

*Last Updated*: 2026-02-08
