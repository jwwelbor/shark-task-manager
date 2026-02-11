# UAT Test Guide - Orchestrator Actions

**Feature:** E16-F02 - Orchestrator Actions
**Epic:** E16 - Multi-Level Workflow System
**Generated:** 2026-02-09
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Extend shark's configurable workflow engine from task-only to epic, feature, and task levels with level-specific status flows, orchestrator actions, and cascading state management.

**This Feature's Role:** E16-F02 adds orchestrator action resolution to epic and feature status transitions, enabling config-driven agent dispatch at all entity levels. It extends `show-actions` and `validate-actions` commands for multi-level display/validation.

**Related Features:**
- E16-F01: Core Workflow Engine (completed) - Provides workflow parsing, transition validation, and `next-status` commands for epics/features
- E16-F03: Display and Aggregation Threshold (completed) - Planning vs aggregation mode display
- E16-F04: Notes and Context for Epic and Feature (active) - Note/context system for epics/features
- E16-F05 Backward Transition and Escalation (draft) - Will add backward transition guards
- E16-F06 Workflow Visualization (draft) - Will add workflow list visualization

**Integration Points:**
- E16-F01 provides the workflow engine, transition types, and service layer that F02 extends with action resolution
- `PopulateTemplate` is shared across all entity types for template variable substitution
- `show-actions` and `validate-actions` CLI commands are multi-level extensions of existing task-only commands

---

## Design Intent

**From Epic PRD:**
> Orchestrator becomes config-driven at all levels: No hardcoded dispatch logic; shark tells the orchestrator what to do at every status transition, for every entity type

**From Feature PRD:**
> Include orchestrator_action in status metadata for epic and feature workflows, returned in transition responses for `update --status` and `next-status` commands. Extend `shark workflow show-actions` to display actions for all three entity levels.

**Key Design Decisions:**
- Template placeholders expanded from `{task_id}` only to `{id}`, `{task_id}`, `{epic_id}`, `{feature_id}` for multi-level support
- Per-transition orchestrator actions on `TransitionInfoWithAction` type (replacing top-level `NextStatusInfo.OrchestratorAction`)
- `resolveAction` private method pattern replicated on both EpicService and FeatureService
- `displayOrchestratorAction` extracted to shared file for reuse across entity types
- Multi-level `--level` flag added to both `show-actions` and `validate-actions`

---

## Cross-Feature Integration Tests

### Integration Scenario 1: F01 Engine + F02 Actions
**Features:** E16-F01 + E16-F02
**Scenario:** Epic/feature transition via `next-status` command returns orchestrator action from workflow config

Steps:
1. Verify `epic next-status` transitions work (F01 engine)
2. Verify transition response includes `orchestrator_action` when configured (F02 addition)

Expected Result: Transition succeeds AND action is populated with resolved template

---

## Epic Acceptance Validation

| Epic AC | Description | Feature Contribution | Status |
|---------|-------------|---------------------|--------|
| SC-2 | Orchestrator actions returned in transition responses for all entity levels | Primary deliverable | [ ] |
| SC-4 | `shark workflow list` / `show-actions` shows all three workflow levels | `show-actions` multi-level extension | [ ] |
| SC-5 | `shark workflow validate` validates all three levels | `validate-actions` multi-level extension | [ ] |

---

## Feature Acceptance Validation

| Feature AC | Description | Status |
|------------|-------------|--------|
| AC-S1 | `shark epic next-status` response includes `orchestrator_action` with resolved `{id}` | [ ] |
| AC-S1 | `shark feature next-status` response includes `orchestrator_action` | [ ] |
| AC-S2 | `shark workflow show-actions` displays epic, feature, task sections | [ ] |
| AC-S2 | `show-actions --json` includes all three levels in structured format | [ ] |
| AC-S3 | `shark workflow validate-actions` validates all three levels | [ ] |
| AC-S3 | Validates `agent_type` present for `spawn_agent` actions | [ ] |
| AC-S4 | No action defined -> `orchestrator_action` is null in JSON | [ ] |

---

## Test Scenarios

### Scenario 1: PopulateTemplate Multi-Level Placeholders (T-E16-F02-001)
**Tasks covered:** T-E16-F02-001

**Steps:**
1. Review `PopulateTemplate` code for `{id}`, `{task_id}`, `{epic_id}`, `{feature_id}` replacement
2. Review `validateTemplateSyntax` for expanded known placeholder set
3. Run unit tests for template population and validation

**Success Criteria:**
- [ ] `{id}` placeholder replaced with entity key
- [ ] `{task_id}` backward compatibility preserved
- [ ] `{epic_id}` and `{feature_id}` placeholders replaced
- [ ] Template validation accepts all four known placeholders
- [ ] Template validation warns on unknown placeholders

### Scenario 2: TransitionInfoWithAction Type (T-E16-F02-002)
**Tasks covered:** T-E16-F02-002

**Steps:**
1. Review `TransitionInfoWithAction` struct definition with embedded `TransitionInfo`
2. Review `NextStatusInfo.AvailableTransitions` type change
3. Verify `TransitionResult.OrchestratorAction` field unchanged from F01
4. Run serialization tests

**Success Criteria:**
- [ ] `TransitionInfoWithAction` exists with embedded `workflow.TransitionInfo` + `OrchestratorAction`
- [ ] `NextStatusInfo` does NOT have top-level `OrchestratorAction` field
- [ ] JSON serialization correctly flattens embedded fields
- [ ] Services compile with new type

### Scenario 3: EpicService Action Resolution (T-E16-F02-003)
**Tasks covered:** T-E16-F02-003

**Steps:**
1. Review `resolveAction` method on `EpicService`
2. Review integration into `TransitionStatus` and `GetNextStatus`
3. Run epic service tests

**Success Criteria:**
- [ ] `resolveAction` returns populated action when configured
- [ ] `resolveAction` returns nil safely on missing config/metadata
- [ ] `TransitionStatus` includes action in response
- [ ] `GetNextStatus` enriches each transition with per-status action
- [ ] `{id}` template replaced with epic key

### Scenario 4: FeatureService Action Resolution (T-E16-F02-004)
**Tasks covered:** T-E16-F02-004

**Steps:**
1. Review `resolveAction` method on `FeatureService`
2. Review integration into `TransitionStatus` and `GetNextStatus`
3. Run feature service tests

**Success Criteria:**
- [ ] `resolveAction` mirrors EpicService pattern
- [ ] Feature key (e.g., "E16-F01") substituted for `{id}`
- [ ] Nil-safe on missing workflow/metadata
- [ ] `TransitionStatus` and `GetNextStatus` include actions

### Scenario 5: Shared Orchestrator Action Display (T-E16-F02-005)
**Tasks covered:** T-E16-F02-005

**Steps:**
1. Verify `displayOrchestratorAction` extracted to `orchestrator_display.go`
2. Verify function removed from `task.go`
3. Verify `performEntityTransition` calls `displayOrchestratorAction`
4. Run CLI tests

**Success Criteria:**
- [ ] Function lives in `orchestrator_display.go`, not `task.go`
- [ ] Epic and feature `next-status` show action details in human-readable mode
- [ ] Nil action shows "Next Action: None configured"
- [ ] Task `next-status` behavior unchanged

### Scenario 6: Multi-Level Show-Actions (T-E16-F02-006)
**Tasks covered:** T-E16-F02-006

**Steps:**
1. Run `shark workflow show-actions` and verify three sections
2. Test `--level` flag filtering
3. Test JSON output structure
4. Run show-actions tests

**Success Criteria:**
- [ ] Three sections displayed (epic, feature, task)
- [ ] `--level=epic` filters to epic only
- [ ] `--level=task` backward compatible
- [ ] JSON output contains `epic_actions`, `feature_actions`, `task_actions`
- [ ] Unconfigured levels show default message

### Scenario 7: Multi-Level Validate-Actions (T-E16-F02-007)
**Tasks covered:** T-E16-F02-007

**Steps:**
1. Run `shark workflow validate-actions` and verify three levels validated
2. Test `--level` flag filtering
3. Test JSON output structure
4. Run validate-actions tests

**Success Criteria:**
- [ ] All three levels validated independently
- [ ] `--level=task` backward compatible
- [ ] Overall `valid` is false if ANY level fails
- [ ] `--strict` mode applies per-level
- [ ] Default workflows pass validation

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-02-09 22:30 |
| Result | FAIL (6/7 pass, T-E16-F02-001 failed) |
| Results File | docs/uat/E16/results/2026-02-09-22-30-E16-F02.md |

**Previous Sessions:**
- 2026-02-09: 6/7 PASS. T-E16-F02-001 (PopulateTemplate) not implemented - only `{task_id}` supported, `{id}`, `{epic_id}`, `{feature_id}` missing.
