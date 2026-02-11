# UAT Test Guide - Orchestrator Actions

**Feature:** E16-F02 - Orchestrator Actions
**Epic:** E16 - Multi-Level Workflow System
**Generated:** 2026-02-09
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Extend shark's configurable workflow engine from task-only to epic, feature, and task levels with level-specific status flows, orchestrator actions, and cascading state management.

**This Feature's Role:** E16-F02 provides orchestrator action support at all entity levels - when an entity transitions status, the system returns structured action metadata telling the orchestrator what agent to spawn next. This is the config-driven dispatch mechanism that eliminates hardcoded agent routing.

**Related Features:**
- E16-F01 Core Workflow Engine (completed) - Provides the workflow parsing and transition infrastructure that F02 builds on
- E16-F03 Display and Aggregation Threshold (completed) - Handles display of planning vs aggregation states
- E16-F04 Notes and Context for Epic and Feature (active) - Extends context tracking to epic/feature levels
- E16-F05 Backward Transition and Escalation (draft) - Will add backward transition guards
- E16-F06 Workflow Visualization (draft) - Will add workflow list visualization

**Integration Points:**
- F02 depends on F01's workflow parsing (epic_workflow, feature_workflow config sections)
- F02's `PopulateTemplate` is used by the service layer (`epic_service.go`, `feature_service.go`, `task_repository.go`) to resolve action instructions

---

## Design Intent

**From Epic PRD:**
> Orchestrator becomes config-driven at all levels: No hardcoded dispatch logic; shark tells the orchestrator what to do at every status transition, for every entity type

**From Feature PRD:**
> Include `orchestrator_action` in status metadata for epic and feature workflows, returned in transition responses. Extend `shark workflow show-actions` to display actions for all three entity levels.

**Key Design Decisions:**
- All four placeholder names (`{id}`, `{task_id}`, `{epic_id}`, `{feature_id}`) resolve to the same entity key value - names exist for template readability
- Template validation warns on unknown placeholders but doesn't error (non-blocking)
- Backward compatibility: existing `{task_id}` templates work identically to before

---

## UAT Focus: T-E16-F02-001

**Task:** Expand PopulateTemplate and template validation for multi-level placeholders

This task extends the template system from task-only (`{task_id}`) to multi-level support (`{id}`, `{epic_id}`, `{feature_id}`, `{task_id}`).

---

## Test Scenarios

### Scenario 1: PopulateTemplate - Multi-Level Placeholder Replacement

**Tasks covered:** T-E16-F02-001 (FR-1 through FR-5)

**Spec Requirement:** PopulateTemplate replaces `{id}`, `{epic_id}`, `{feature_id}`, and `{task_id}` placeholders with the entity key.

**Success Criteria:**
- [ ] `{id}` placeholder replaced with entity key
- [ ] `{epic_id}` placeholder replaced with entity key
- [ ] `{feature_id}` placeholder replaced with entity key
- [ ] `{task_id}` backward compatibility preserved
- [ ] Mixed placeholders in same template all replaced
- [ ] Template with no placeholders returned unchanged

### Scenario 2: Template Validation - Expanded Known Placeholders

**Tasks covered:** T-E16-F02-001 (FR-6 through FR-8)

**Spec Requirement:** `validateTemplateSyntax` accepts all four placeholders without warnings and warns on unknown placeholders.

**Success Criteria:**
- [ ] `{id}` produces zero validation warnings
- [ ] `{epic_id}` produces zero validation warnings
- [ ] `{feature_id}` produces zero validation warnings
- [ ] Template with no known placeholder produces warning
- [ ] Unknown placeholder like `{custom_field}` produces warning

### Scenario 3: ValidateWithContext - Updated Suggestion Message

**Tasks covered:** T-E16-F02-001 (FR-9)

**Spec Requirement:** `ValidateWithContext` suggests `{id}` as the recommended placeholder in missing-template error message.

**Success Criteria:**
- [ ] Error message includes `{id}` as recommended placeholder
- [ ] Error message mentions all four supported placeholders

### Scenario 4: Backward Compatibility - Existing Tests Unmodified

**Tasks covered:** T-E16-F02-001 (NFR-1)

**Spec Requirement:** All existing tests pass without modification (except one intentional update to `TestValidateTemplate_UnknownPlaceholder`).

**Success Criteria:**
- [ ] All tests in `go test ./internal/config/...` pass
- [ ] Full test suite `make test` passes
- [ ] Quality gate `make fmt && make lint && make test` passes

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-02-09 |
| Result | PASS (4/4 scenarios) |
| Results File | docs/uat/E16/results/2026-02-09-22-30-E16-F02.md |

**Previous Sessions:**
- 2026-02-09: PASS (4/4 scenarios) - T-E16-F02-001 placeholder expansion UAT
