---
feature_key: E16-F05-backward-transition-and-escalation
epic_key: E16
title: Backward Transition and Escalation
description: set-status with --reason, backward transition guards, upward escalation tooling
priority: P2
---

# Backward Transition and Escalation

**Feature Key**: E16-F05-backward-transition-and-escalation
**Priority**: P2

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

When task-level work discovers a design flaw or architecture conflict, there's no mechanism to push the parent feature back to a refinement state. Currently, the only option is to manually update the status, but there's no guardrail requiring a reason and no audit trail for why a backward transition happened.

### Solution

Add `shark feature set-status` and `shark epic set-status` commands that allow any valid status transition (with workflow validation) but require a `--reason` flag when moving backward (from `active` or a later phase to an earlier planning phase). Log the reason in entity history/notes. Do NOT automatically change child entity statuses.

### Impact

- Provides explicit, audited backward transition path
- Prevents accidental backward transitions without justification
- Enables design-flaw discovery at task level to surface at feature level
- Child entities remain unchanged (orchestrator decides what to do with them)

---

## User Stories

### Must-Have Stories

**Story 1**: As a tech lead, I want to move a feature backward to a planning state with a documented reason when a design flaw is discovered.

**Acceptance Criteria**:
- [ ] `shark feature set-status E16-F01 ready_for_refinement_tech --reason "..."` works
- [ ] Transition validates against workflow
- [ ] Reason is logged in feature history/notes
- [ ] Child tasks remain in their current states

**Story 2**: As a project manager, I want backward transitions to require a reason so that accidental regressions are prevented.

**Acceptance Criteria**:
- [ ] Moving from a later phase to an earlier phase requires `--reason`
- [ ] Omitting `--reason` on backward transition produces an error
- [ ] Forward transitions do NOT require `--reason`

**Story 3**: As a project manager, I want `shark epic set-status` with the same backward transition guards.

**Acceptance Criteria**:
- [ ] Same behavior as feature set-status for epic entities
- [ ] Reason logged in epic history

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when I try to move a feature backward without a reason, I want a clear error explaining why `--reason` is required.

**Acceptance Criteria**:
- [ ] Error message: "Backward transition from 'active' to 'ready_for_refinement_tech' requires --reason flag"
- [ ] Lists the required flag usage

---

## Requirements

### Functional Requirements

**Category: Backward Transition (FR-9)**

1. **REQ-F-001**: `shark feature set-status <key> <status>` command
   - **Description**: Sets feature to any valid status with workflow validation
   - **Priority**: Must-Have

2. **REQ-F-002**: `shark epic set-status <key> <status>` command
   - **Description**: Sets epic to any valid status with workflow validation
   - **Priority**: Must-Have

3. **REQ-F-003**: Backward transition detection
   - **Description**: Compare current phase with target phase; if target is earlier, it's a backward transition
   - **Priority**: Must-Have

4. **REQ-F-004**: `--reason` required for backward transitions
   - **Description**: Backward transitions MUST include `--reason` flag
   - **Priority**: Must-Have

5. **REQ-F-005**: Reason logging
   - **Description**: Log the reason in entity history/notes with timestamp
   - **Priority**: Must-Have

6. **REQ-F-006**: Child entities unchanged
   - **Description**: Moving a feature backward does NOT change child task statuses. Display warning: "N tasks remain in current states."
   - **Priority**: Must-Have

7. **REQ-F-007**: `--force` overrides reason requirement
   - **Description**: `--force` flag allows backward transition without `--reason` (for emergencies)
   - **Priority**: Should-Have

---

### Non-Functional Requirements

1. **REQ-NF-001**: Phase ordering is derived from workflow config (not hardcoded)
   - Phases have implicit ordering based on position in status_flow

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Backward Transition with Reason**
- **Given** feature E16-F01 at status `active`
- **When** `shark feature set-status E16-F01 ready_for_refinement_tech --reason "API contract conflicts with database schema"`
- **Then** feature transitions to `ready_for_refinement_tech`
- **And** reason is logged in feature notes/history
- **And** warning: "Feature moved backward to planning phase. 15 tasks remain in current states."

**Scenario 2: Backward Transition without Reason (Error)**
- **Given** feature E16-F01 at status `active`
- **When** `shark feature set-status E16-F01 ready_for_refinement_tech`
- **Then** error: "Backward transition requires --reason flag"

**Scenario 3: Forward Transition (No Reason Needed)**
- **Given** feature E16-F01 at status `draft`
- **When** `shark feature set-status E16-F01 ready_for_refinement_ba`
- **Then** transition succeeds without requiring `--reason`

---

## Out of Scope

1. **Automatic child status changes** - Orchestrator decides what to do with children
2. **Automatic escalation** - This is explicit tooling, not automatic propagation

---

## Dependencies & Integrations

### Dependencies

- **E16-F01**: Core Workflow Engine (provides workflow validation and phase metadata)
- **E16-F04**: Notes & Context (provides note storage for logging reasons) -- soft dependency, can use basic history if F04 isn't complete

---

*Last Updated*: 2026-02-08
