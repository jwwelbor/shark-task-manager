# E18 UAT Acceptance Plan: Bug and Change-Card Management System

**Date**: 2026-03-02
**Author**: QA Agent
**Epic**: E18 -- Bug and Change-Card Management System
**Status**: Complete

---

## Table of Contents

1. [Acceptance Scenarios from User Journeys](#1-acceptance-scenarios-from-user-journeys)
2. [Success Metrics Validation Plan](#2-success-metrics-validation-plan)
3. [Cross-Epic Integration Test Scenarios](#3-cross-epic-integration-test-scenarios)
4. [Risk-Based Test Priorities](#4-risk-based-test-priorities)
5. [Persona-Based Acceptance Matrix](#5-persona-based-acceptance-matrix)

---

## 1. Acceptance Scenarios from User Journeys

Each user journey from `user-journeys.md` is decomposed into concrete, executable acceptance scenarios. Scenarios cover the happy path and all documented alternative paths.

---

### Journey 1: Bug Lifecycle -- Report, Triage, Fix, Verify

#### Scenario J1-HP: Bug Happy Path -- Report Through Verified Resolution

**Preconditions**:
- Shark is initialized with the advanced workflow profile
- Bug entity type is enabled (bugs table exists, bug workflow configured in `.sharkconfig.json`)
- Epic E07 and feature E07-F01 exist in the database
- No bugs exist in the database (clean state)

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Developer | `shark bug create "Login fails on Safari when 2FA is enabled" --severity=high --link=E07-F01` | Bug B001 created with status `reported`, severity `high`, linked to E07-F01. Markdown file generated at `docs/bugs/B001.md`. CLI outputs success with key B001. |
| 2 | Developer | `shark bug list` | B001 appears in list with status `reported`, severity `high`. |
| 3 | QA Engineer | `shark bug list --status=reported` | B001 appears (filtered to reported bugs only). |
| 4 | QA Engineer | `shark bug triage B001 --severity=high --assign=developer` | B001 advances to `triaged`. Severity confirmed as `high`. Agent assignment recorded. |
| 5 | Developer | `shark bug list --status=triaged --agent=developer` | B001 appears in developer's assigned work. |
| 6 | Developer | `shark bug get B001` | Full bug details displayed: title, severity, status (`triaged`), linked entity (E07-F01), file path. |
| 7 | Developer | `shark status advance B001` | B001 advances to `in_fix`. |
| 8 | Developer | `shark status advance B001` | B001 advances to `in_verification`. |
| 9 | QA Engineer | `shark bug list --status=in_verification` | B001 appears in verification queue. |
| 10 | QA Engineer | `shark status advance B001` | B001 advances to `resolved`. |
| 11 | Any User | `shark status history B001` | Full audit trail: reported -> triaged -> in_fix -> in_verification -> resolved with timestamps for each transition. |

**Pass Criteria**: All 11 steps execute without error. Bug progresses through the complete lifecycle. Status history contains all transitions with timestamps.

---

#### Scenario J1-ALT-A: Bug Marked as Duplicate

**Preconditions**:
- Bug B001 exists in `triaged` status
- Bug B002 exists in `reported` status (the duplicate)

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | QA Engineer | `shark status set B002 duplicate` | B002 status changes to `duplicate`. |
| 2 | QA Engineer | `shark bug note add B002 --type=comment "Duplicate of B001"` | Note added to B002 referencing B001. |
| 3 | Any User | `shark bug list --status=duplicate` | B002 appears. B001 does not appear. |
| 4 | Any User | `shark bug list` | B002 shown with status `duplicate`. It does not inflate active bug counts. |

**Pass Criteria**: Duplicate status is a valid terminal state accessible from any non-terminal status. Notes preserve the link to the original bug.

---

#### Scenario J1-ALT-B: Bug Marked as Won't Fix

**Preconditions**:
- Bug B001 exists in `triaged` status

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Developer | `shark status set B001 wont_fix` | B001 status changes to `wont_fix`. |
| 2 | Developer | `shark bug note add B001 --type=decision "Behavior is intentional per spec section 3.2"` | Decision note recorded. |
| 3 | Any User | `shark bug get B001` | Status shows `wont_fix`. Notes include the decision rationale. |

**Pass Criteria**: `wont_fix` is a valid terminal state. The decision rationale is preserved in notes.

---

#### Scenario J1-ALT-C: Verification Fails -- Fix Returned to Developer

**Preconditions**:
- Bug B001 exists in `in_verification` status

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | QA Engineer | `shark status set B001 in_fix` | B001 status returns to `in_fix`. |
| 2 | QA Engineer | `shark bug note add B001 --type=comment "Fix does not cover edge case: empty username"` | Note explains what failed verification. |
| 3 | Developer | `shark bug get B001` | Status shows `in_fix`. Notes include the QA feedback. |
| 4 | Developer | `shark status advance B001` | B001 advances back to `in_verification`. |
| 5 | QA Engineer | `shark status advance B001` | B001 advances to `resolved`. |

**Pass Criteria**: Bug can cycle between `in_fix` and `in_verification` until the fix passes. Each cycle is recorded in status history.

---

#### Scenario J1-ERR-1: Invalid Workflow Transition Rejected

**Preconditions**:
- Bug B001 exists in `reported` status

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Developer | `shark status set B001 resolved` | Command fails with clear error: cannot transition directly from `reported` to `resolved`. |
| 2 | Developer | `shark status options B001` | Shows valid next statuses: `triaged`, `duplicate`, `wont_fix`. |

**Pass Criteria**: Invalid transitions are rejected with a human-readable error message. `shark status options` shows only valid transitions.

---

#### Scenario J1-ERR-2: Bug Creation with Invalid Link

**Preconditions**:
- Epic E99 does not exist in the database

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Developer | `shark bug create "Some bug" --link=E99` | Command fails with error: linked entity E99 not found. No bug created. No orphaned files. |

**Pass Criteria**: Link validation prevents bugs from referencing non-existent entities. The operation is atomic -- no partial state.

---

### Journey 2: Change-Card Lifecycle -- Propose, Approve, Implement

#### Scenario J2-HP: Change-Card Happy Path -- Propose Through Completion

**Preconditions**:
- Shark is initialized with the advanced workflow profile
- Change-card entity type is enabled
- Epic E17 exists in the database
- No change-cards exist in the database (clean state)

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Developer | `shark change create "Add keyboard shortcuts for common actions" --link=E17` | Change-card C001 created with status `proposed`, linked to E17. Markdown file generated at `docs/changes/C001.md`. |
| 2 | Product Owner | `shark change list --status=proposed` | C001 appears in proposed list. |
| 3 | Product Owner | `shark change get C001` | Full details: title, status (`proposed`), linked entity (E17), file path. |
| 4 | Product Owner | `shark change approve C001` | C001 advances from `proposed` to `approved`. |
| 5 | Developer | `shark status advance C001` | C001 advances to `in_progress`. |
| 6 | Developer | `shark status advance C001` | C001 advances to `completed`. |
| 7 | Any User | `shark status history C001` | Full audit trail: proposed -> approved -> in_progress -> completed with timestamps. |

**Pass Criteria**: All 7 steps execute without error. Change-card progresses through the complete lifecycle.

---

#### Scenario J2-ALT-A: Change-Card Declined

**Preconditions**:
- Change-card C001 exists in `proposed` status

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Product Owner | `shark status set C001 declined` | C001 status changes to `declined`. |
| 2 | Product Owner | `shark change note add C001 --type=decision "Not aligned with Q2 priorities"` | Decision note recorded. |
| 3 | Developer | `shark change get C001` | Status shows `declined`. Notes include the rationale. |

**Pass Criteria**: `declined` is a valid terminal state from `proposed`. Decision rationale is preserved.

---

#### Scenario J2-ALT-B: Change-Card Promoted to Feature (Manual Process)

**Preconditions**:
- Change-card C001 exists in `proposed` status
- Epic E17 exists

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Product Owner | `shark feature create E17 "Keyboard Shortcuts System"` | New feature created under E17. |
| 2 | Product Owner | `shark status set C001 declined` | C001 declined. |
| 3 | Product Owner | `shark change note add C001 --type=decision "Promoted to feature E17-F03"` | Link to the new feature documented. |

**Pass Criteria**: The manual promotion workflow is coherent. The change-card serves as documented context for the new feature.

---

#### Scenario J2-ERR-1: Approve Already-Approved Change-Card

**Preconditions**:
- Change-card C001 exists in `approved` status

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Product Owner | `shark change approve C001` | Command fails with error: C001 is already in `approved` status, not `proposed`. |

**Pass Criteria**: The approve command enforces that the change-card must be in `proposed` status.

---

#### Scenario J2-ERR-2: Decline from Non-Proposed Status

**Preconditions**:
- Change-card C001 exists in `in_progress` status

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Product Owner | `shark status set C001 declined` | Command fails with error: `declined` is only valid from `proposed` status. |

**Pass Criteria**: Workflow enforcement prevents invalid terminal state transitions.

---

### Journey 3: Cross-Entity Discovery and Linking

#### Scenario J3-HP: Query Related Bugs for a Feature

**Preconditions**:
- Feature E07-F01 exists
- Bug B001 is linked to E07-F01 (status: `in_fix`)
- Bug B002 is linked to E07-F01 (status: `resolved`)
- Bug B003 is linked to E07-F02 (different feature)

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Any User | `shark bug list --link=E07-F01` | Returns B001 and B002. Does NOT return B003. |
| 2 | Any User | `shark bug list --link=E07` | Returns B001, B002, and B003 (all bugs linked to entities under epic E07). |
| 3 | Any User | `shark search "login" --type=bug` | Returns bugs matching "login" in title or description. |

**Pass Criteria**: Linked-entity filtering returns correct results. Epic-level filtering aggregates across features. Search with type filter works.

---

#### Scenario J3-DASH: Feature Status with Bug Context

**Preconditions**:
- Feature E07-F01 exists with tasks
- 2 bugs are linked to E07-F01 (1 `in_fix`, 1 `resolved`)

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Any User | `shark status E07-F01` | Feature status includes a "Related Bugs" section showing: 2 total bugs, 1 open, 1 resolved. |

**Pass Criteria**: Feature status view includes linked bug summary without requiring a separate command.

---

#### Scenario J3-ANALYTICS: Project-Wide Bug and Change-Card Metrics

**Preconditions**:
- 5 bugs exist: 2 `reported`, 1 `in_fix`, 1 `resolved`, 1 `wont_fix`
- 3 change-cards exist: 1 `proposed`, 1 `in_progress`, 1 `completed`

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Product Owner | `shark analytics` | Output includes: Bug section (total: 5, by status, by severity, avg resolution time). Change-Card section (total: 3, by status, approval rate, avg time-to-completion). |
| 2 | Product Owner | `shark analytics --type=bug` | Bug-specific analytics only. No change-card data. |
| 3 | Product Owner | `shark analytics --type=change` | Change-card-specific analytics only. No bug data. |

**Pass Criteria**: Analytics output includes all specified metrics. Type filter isolates entity-specific analytics.

---

#### Scenario J3-DASHBOARD: Project Dashboard with All Entity Types

**Preconditions**:
- Epics, features, tasks, bugs, and change-cards all exist in the database

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Any User | `shark status` | Project dashboard shows: Epics section, Features section, Tasks section, Bugs section (counts by status, severity breakdown), Change Cards section (counts by status). |
| 2 | Any User | Verify sections are conditionally displayed | If no bugs exist, the Bugs section should not appear. If no change-cards exist, the Change Cards section should not appear. |

**Pass Criteria**: Dashboard includes all 5 entity types. Sections are conditionally displayed. Bug severity breakdown is visible.

---

#### Scenario J3-UNIFIED: Unified Commands Auto-Detect Bug and Change-Card Keys

**Preconditions**:
- Bug B001 and change-card C001 exist

**Steps**:

| Step | Actor | Action | Expected Outcome |
|------|-------|--------|------------------|
| 1 | Any User | `shark get B001` | Returns bug details (auto-detected from B prefix). |
| 2 | Any User | `shark get C001` | Returns change-card details (auto-detected from C prefix). |
| 3 | Any User | `shark get B001 --json` | Returns JSON output with all bug fields. |
| 4 | Any User | `shark get B001 --field status` | Returns just the status value. |
| 5 | Any User | `shark status advance B001` | Advances the bug workflow. |
| 6 | Any User | `shark status advance C001` | Advances the change-card workflow. |

**Pass Criteria**: All unified commands (`get`, `status`, `search`) auto-detect B### and C### key formats without requiring entity-specific subcommands.

---

## 2. Success Metrics Validation Plan

Each metric from `success-metrics.md` has a concrete measurement method, pass/fail criteria, and the scenarios that validate measurement capability.

---

### Metric 1: Bug Tracking Adoption Rate

**What**: Percentage of bugs tracked in Shark versus external tools

| Attribute | Value |
|-----------|-------|
| **Target** | 90% of bugs tracked in Shark within 8 weeks |
| **Minimum Viable** | 50% within 4 weeks |
| **Measurement Method** | `shark analytics --type=bug --json` to extract total bug count per week. Compare with team's historical volume from prior tools. |
| **Validation Scenario** | J3-ANALYTICS (Step 2) proves the analytics command outputs bug counts by status. |
| **Pass/Fail for UAT** | PASS if `shark analytics --type=bug` outputs total bug count, bugs by status, and creation date data that enables weekly aggregation. Adoption rate itself is measured post-launch, not during UAT. |
| **UAT Validation** | Verify the data infrastructure exists to measure this metric. Create 10 bugs over 3 simulated "weeks" (via timestamp manipulation or by verifying created_at field is present in JSON output) and confirm `shark analytics --type=bug --json` returns data sufficient to calculate weekly creation rate. |

---

### Metric 2: Average Bug Resolution Time

**What**: Elapsed time from `reported` to resolution

| Attribute | Value |
|-----------|-------|
| **Target** | 20% reduction by week 8 compared to week 2 baseline |
| **Minimum Viable** | No increase versus baseline |
| **Measurement Method** | `shark analytics --type=bug` to report average resolution time. Calculated from status history timestamps: `resolution_timestamp - creation_timestamp`. |
| **Validation Scenario** | J1-HP (Step 11) proves status history contains all transitions with timestamps. |
| **Pass/Fail for UAT** | PASS if: (a) status history records timestamps for each transition, (b) `shark analytics --type=bug` includes an "average resolution time" metric, (c) the metric calculation correctly uses creation and resolution timestamps. |
| **UAT Validation** | Create a bug, advance it through the full lifecycle (reported -> resolved), then verify `shark analytics --type=bug --json` includes a `resolution_time_avg` (or equivalent) field that is non-zero and plausible. |

---

### Metric 3: Change-Card Throughput

**What**: Number of change-cards completed per month

| Attribute | Value |
|-----------|-------|
| **Target** | 5+ completed per month by month 2 |
| **Minimum Viable** | 2+ per month |
| **Measurement Method** | `shark analytics --type=change` to report completed change-card count. |
| **Validation Scenario** | J3-ANALYTICS (Step 3) proves change-card analytics exist. |
| **Pass/Fail for UAT** | PASS if `shark analytics --type=change` outputs total change-cards by status, including a count of `completed` change-cards. Throughput rate itself is measured post-launch. |
| **UAT Validation** | Create 3 change-cards, advance 2 to `completed`, then verify `shark analytics --type=change --json` shows completed count of 2. |

---

### Metric 4: Bug Creation Speed

**What**: Time to create a bug via CLI

| Attribute | Value |
|-----------|-------|
| **Target** | Under 500ms local, under 2s cloud |
| **Minimum Viable** | Under 1s local, under 3s cloud |
| **Measurement Method** | `time shark bug create "Test bug" --severity=medium` measured via shell timing. |
| **Validation Scenario** | J1-HP (Step 1) exercises bug creation. |
| **Pass/Fail for UAT** | PASS if bug creation completes in under 500ms on local SQLite (measured as wall clock time from `time` command). |
| **UAT Validation** | Run `time shark bug create "Performance test bug" --severity=low` 5 times on local SQLite. All 5 runs must complete in under 500ms. Calculate average and p95 latency. |

---

### Metric 5: Dashboard Visibility Score

**What**: Whether bugs and change-cards appear in `shark status` dashboard

| Attribute | Value |
|-----------|-------|
| **Target** | All 3 checklist items pass: (1) bug counts by status visible, (2) bug severity breakdown visible, (3) change-card counts by status visible |
| **Minimum Viable** | Items 1 and 2 pass |
| **Measurement Method** | Manual inspection of `shark status` output. |
| **Validation Scenario** | J3-DASHBOARD (Steps 1-2) directly validates this. |
| **Pass/Fail for UAT** | PASS if `shark status` output contains all three items when bugs and change-cards exist in the database. |
| **UAT Validation** | Create at least 2 bugs (different severities and statuses) and 2 change-cards (different statuses). Run `shark status` and visually confirm: (1) "Bugs" section with counts, (2) severity breakdown (e.g., "high: 1, low: 1"), (3) "Change Cards" section with counts. |

---

## 3. Cross-Epic Integration Test Scenarios

These scenarios verify that E18's capabilities work correctly with capabilities delivered by other epics.

---

### CE-1: E17 (CLI Simplification) -- Unified Command Dispatch

**What**: E18's B### and C### keys are recognized by all unified commands established in E17.

**Preconditions**: Bug B001 and change-card C001 exist.

| Step | Command | Expected | Validates |
|------|---------|----------|-----------|
| 1 | `shark get B001` | Returns bug details | Key auto-detection works for B prefix |
| 2 | `shark get C001` | Returns change-card details | Key auto-detection works for C prefix |
| 3 | `shark status advance B001` | Advances bug status | Status command dispatches to bug workflow |
| 4 | `shark status advance C001` | Advances change-card status | Status command dispatches to change workflow |
| 5 | `shark status set B001 wont_fix` | Sets bug status directly | Direct status set dispatches correctly |
| 6 | `shark search "login" --type=bug` | Filters search to bugs | Search type filter includes new entity types |
| 7 | `shark delete B001 --force` | Deletes bug | Delete dispatch includes bug entity type |

**Risk Reference**: Research Risk 1 (Cross-Cutting Entity Type Changes). This is the highest-risk integration point.

**Pass Criteria**: All 7 unified commands correctly dispatch to bug/change-card handlers. No "unknown entity type" errors.

---

### CE-2: E16 (Multi-Level Workflow) -- Workflow Engine Extension

**What**: Bug and change-card workflows are registered and enforced by the multi-level workflow engine.

**Preconditions**: `.sharkconfig.json` contains bug and change-card workflow definitions.

| Step | Command | Expected | Validates |
|------|---------|----------|-----------|
| 1 | `shark status options B001` (status: `reported`) | Shows: `triaged`, `duplicate`, `wont_fix` | Bug workflow transitions registered |
| 2 | `shark status options C001` (status: `proposed`) | Shows: `approved`, `declined` | Change-card workflow transitions registered |
| 3 | `shark status set B001 completed` | Error: invalid transition | Bug workflow enforces valid transitions |
| 4 | `shark status set C001 in_fix` | Error: invalid transition | Change-card workflow rejects bug statuses |
| 5 | `shark init update --workflow=advanced --dry-run` | Output includes bug and change-card workflow definitions | Workflow profiles include new entity types |

**Risk Reference**: Research Risk 2 (Workflow Engine Extension), Risk 4 (E16 Dependency).

**Pass Criteria**: Workflow engine correctly enforces bug and change-card status flows independently. `ForLevel()` dispatches to the correct workflow for each entity type.

---

### CE-3: E15 (Service Layer) -- Service Pattern Compliance

**What**: Bug and change-card services follow E15 service layer patterns.

**Validation Method**: Code review (not runtime test). Verify:

| Check | What to Verify |
|-------|---------------|
| Constructor injection | `NewBugService(repo, workflowSvc, ...)` pattern |
| Interface-based repos | `BugRepository` is an interface, not concrete type |
| No business logic in CLI | `shark bug create` command is a thin wrapper |
| No repo calls from CLI | CLI commands call services, not repositories |
| Context first parameter | All service methods accept `context.Context` as first param |
| Error wrapping | Service errors wrap repository errors with business context |

**Pass Criteria**: All 6 checks pass during code review. This is a structural quality gate, not a runtime test.

---

### CE-4: E08 (Idea System) -- Pattern Consistency

**What**: Bug and change-card entities follow the same standalone entity patterns established by ideas.

**Preconditions**: Ideas, bugs, and change-cards all exist.

| Step | Command | Expected | Validates |
|------|---------|----------|-----------|
| 1 | `shark bug get B001 --json` | JSON structure follows same patterns as `shark idea get 1 --json` | Output consistency |
| 2 | `shark change get C001 --json` | JSON structure follows same patterns | Output consistency |
| 3 | `shark bug note add B001 --type=comment "test"` | Note added | Notes system works for bugs |
| 4 | `shark change note add C001 --type=comment "test"` | Note added | Notes system works for change-cards |
| 5 | `shark bug context set B001 --field env --value "Safari"` | Context set | Context system works for bugs |
| 6 | `shark change context set C001 --field effort --value "small"` | Context set | Context system works for change-cards |

**Pass Criteria**: All commands work. JSON output structures are consistent with existing entity patterns.

---

### CE-5: E11 (Configurable Workflow) -- Configuration Persistence

**What**: Bug and change-card workflows are persisted in `.sharkconfig.json` and survive restarts.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `shark init update --workflow=advanced` | Config file updated with bug and change-card workflow sections |
| 2 | Inspect `.sharkconfig.json` | `bug_workflow` and `change_workflow` sections present with status metadata, flow, and special statuses |
| 3 | Restart shark (new process) | `shark status options B001` still returns correct transitions |
| 4 | Run `shark config validate` | No validation errors for bug or change-card workflow sections |

**Pass Criteria**: Workflow definitions persist across process restarts and pass config validation.

---

## 4. Risk-Based Test Priorities

Scenarios are prioritized based on the risk assessment from the research report and feasibility reviews. **Priority 1 scenarios must pass before any lower-priority testing begins.**

---

### Priority 1: CRITICAL -- Must Pass First

These address the highest-risk areas identified in the research and feasibility reviews.

| Priority | Scenario | Risk Source | Why Critical |
|----------|----------|-------------|--------------|
| P1-1 | CE-1 (Unified Command Dispatch) | Research Risk 1: Cross-cutting entity type changes (HIGH probability, MEDIUM impact) | A missed dispatch point causes silent failures. Every unified command must recognize B### and C### keys. |
| P1-2 | CE-2 (Workflow Engine Extension) | Research Risk 2: Workflow engine extension (MEDIUM probability, MEDIUM impact) | Workflow enforcement is foundational. If `ForLevel()` does not work for bugs/changes, all status operations fail. |
| P1-3 | J1-HP (Bug Happy Path) | Core value proposition | If the basic bug lifecycle does not work end-to-end, the epic fails to deliver its primary value. |
| P1-4 | J2-HP (Change-Card Happy Path) | Core value proposition | If the change-card lifecycle does not work end-to-end, the epic is only half-delivered. |
| P1-5 | J1-ERR-1 (Invalid Transition Rejected) | REQ-F-004 workflow enforcement | Workflow enforcement is a quality gate. If invalid transitions are accepted, data integrity is compromised. |

---

### Priority 2: HIGH -- Must Pass Before Release

These address medium-risk areas and secondary functionality.

| Priority | Scenario | Risk Source | Why High |
|----------|----------|-------------|----------|
| P2-1 | J1-ALT-A (Duplicate Bug) | Bug workflow completeness | Duplicate handling is a core triage operation. |
| P2-2 | J1-ALT-B (Won't Fix) | Bug workflow completeness | Won't fix is a common terminal state. |
| P2-3 | J1-ALT-C (Verification Fails) | Bug workflow completeness | Fix/verify cycles are essential for quality assurance. |
| P2-4 | J2-ALT-A (Change-Card Declined) | Change-card workflow completeness | Decline is the primary alternative outcome. |
| P2-5 | J3-DASHBOARD (Dashboard Visibility) | Research Risk 3: Dashboard overload (MEDIUM probability) | Dashboard must handle 5 entity types without overwhelming users. |
| P2-6 | J3-UNIFIED (Unified Commands) | BA Review: Cross-cutting change risk (MEDIUM severity) | Validates that all unified commands work, not just status. |
| P2-7 | J1-ERR-2 (Invalid Link Rejected) | REQ-NF-003 (atomic operations) + REQ-F-003 (link validation) | Prevents orphaned records and invalid references. |
| P2-8 | CE-5 (Config Persistence) | E16 dependency; config file growth concern | Ensures workflows survive restarts and config validation. |

---

### Priority 3: MEDIUM -- Should Pass Before Release

These validate secondary features and metrics infrastructure.

| Priority | Scenario | Risk Source | Why Medium |
|----------|----------|-------------|-----------|
| P3-1 | J3-ANALYTICS (Metrics Output) | Success Metrics 1-3 measurement infrastructure | Analytics data must be available for post-launch measurement. |
| P3-2 | Metric 4 Validation (Creation Speed) | REQ-NF-001 (performance) | Performance must meet targets to encourage adoption. |
| P3-3 | J3-HP (Linked Entity Filtering) | REQ-F-018 (Should Have) | Important for QA workflow but has CLI workarounds. |
| P3-4 | CE-4 (Pattern Consistency) | Technical Review: E15 pattern compliance | Ensures long-term maintainability. |
| P3-5 | J2-ALT-B (Promotion to Feature) | Manual process; Could Have for automated promotion | Validates that the manual workaround is viable. |

---

### Priority 4: LOW -- Nice to Have

| Priority | Scenario | Risk Source |
|----------|----------|-------------|
| P4-1 | J2-ERR-1 (Re-approve) | Edge case error handling |
| P4-2 | J2-ERR-2 (Decline from wrong status) | Edge case error handling |
| P4-3 | CE-3 (Service Pattern Code Review) | Structural quality; no runtime impact |

---

## 5. Persona-Based Acceptance Matrix

For each persona defined in `personas.md`, this matrix maps which scenarios validate their specific needs. A persona's needs are considered met when all MUST scenarios pass and at least 80% of SHOULD scenarios pass.

---

### Developer (AI Agent or Human)

**Goals from personas.md**:
1. Report bugs immediately when discovered (under 30s, without leaving terminal)
2. View bugs assigned for fixing with clear reproduction steps, severity, and linked context
3. Propose small enhancements as change-cards

| Goal | Scenario(s) | Priority | Requirement |
|------|-------------|----------|-------------|
| Report bug quickly | J1-HP Step 1, Metric 4 (creation speed < 500ms) | P1-3, P3-2 | MUST |
| Report with severity | J1-HP Step 1 (`--severity=high`) | P1-3 | MUST |
| Report with link | J1-HP Step 1 (`--link=E07-F01`) | P1-3 | MUST |
| Invalid link rejected | J1-ERR-2 | P2-7 | MUST |
| View assigned bugs | J1-HP Step 5, Step 6 | P1-3 | MUST |
| See reproduction steps | J1-HP Step 6 (bug markdown file) | P1-3 | SHOULD |
| Advance bug to in_fix | J1-HP Step 7 | P1-3 | MUST |
| Complete fix (advance to verification) | J1-HP Step 8 | P1-3 | MUST |
| Mark won't fix | J1-ALT-B | P2-2 | SHOULD |
| Propose change-card | J2-HP Step 1 | P1-4 | MUST |
| Implement change-card | J2-HP Steps 5-6 | P1-4 | MUST |
| Use unified commands | CE-1, J3-UNIFIED | P1-1, P2-6 | MUST |
| JSON output for AI agents | J3-UNIFIED Step 3 (`--json`) | P2-6 | MUST |

**Acceptance**: Developer persona is validated when all MUST scenarios pass. Minimum: J1-HP, J2-HP, CE-1, Metric 4.

---

### QA Engineer

**Goals from personas.md**:
1. Triage incoming bugs (set severity, assign, add notes)
2. Verify fixes before marking resolved
3. Monitor bug metrics for quality trends

| Goal | Scenario(s) | Priority | Requirement |
|------|-------------|----------|-------------|
| List reported bugs | J1-HP Step 3 (`--status=reported`) | P1-3 | MUST |
| Triage bug (severity + assign) | J1-HP Step 4 (triage command) | P1-3 | MUST |
| Verify fix | J1-HP Steps 9-10 | P1-3 | MUST |
| Return fix to developer | J1-ALT-C | P2-3 | MUST |
| Mark as duplicate | J1-ALT-A | P2-1 | MUST |
| Add notes during triage | J1-ALT-A Step 2, CE-4 Step 3 | P2-1, P3-4 | SHOULD |
| View bug metrics | J3-ANALYTICS Step 2 | P3-1 | MUST |
| Filter bugs by severity | J1-HP Step 3 (implied: `--severity=critical`) | P1-3 | SHOULD |
| Filter bugs by linked entity | J3-HP Step 1 (`--link=E07-F01`) | P3-3 | SHOULD |
| Feature status with bug context | J3-DASH | P2-5 | SHOULD |
| Dashboard with bug counts | J3-DASHBOARD | P2-5 | MUST |

**Acceptance**: QA Engineer persona is validated when all MUST scenarios pass. Minimum: J1-HP, J1-ALT-A, J1-ALT-C, J3-ANALYTICS, J3-DASHBOARD.

---

### Product Owner

**Goals from personas.md**:
1. Review and approve/decline proposed change-cards
2. Understand planned vs. reactive work balance
3. Identify when change-cards should be promoted to features

| Goal | Scenario(s) | Priority | Requirement |
|------|-------------|----------|-------------|
| List proposed change-cards | J2-HP Step 2 | P1-4 | MUST |
| Review change-card details | J2-HP Step 3 | P1-4 | MUST |
| Approve change-card | J2-HP Step 4 | P1-4 | MUST |
| Decline change-card | J2-ALT-A | P2-4 | MUST |
| Decline with rationale | J2-ALT-A Step 2 (note with decision type) | P2-4 | SHOULD |
| Promote to feature (manual) | J2-ALT-B | P3-5 | SHOULD |
| View planned vs reactive balance | J3-ANALYTICS Step 1 | P3-1 | MUST |
| Dashboard with all entity types | J3-DASHBOARD Step 1 | P2-5 | MUST |
| Change-card analytics | J3-ANALYTICS Step 3 | P3-1 | SHOULD |

**Acceptance**: Product Owner persona is validated when all MUST scenarios pass. Minimum: J2-HP, J2-ALT-A, J3-ANALYTICS, J3-DASHBOARD.

---

### Secondary Persona: CI/CD Pipeline Agent

**Goals**: Create bugs programmatically when tests fail.

| Goal | Scenario(s) | Priority | Requirement |
|------|-------------|----------|-------------|
| Create bug with `--json` output | J1-HP Step 1 with `--json` flag | P1-3 | MUST |
| Machine-parseable output | J3-UNIFIED Step 3 | P2-6 | MUST |
| Field extraction | J3-UNIFIED Step 4 (`--field status`) | P2-6 | SHOULD |

**Acceptance**: CI/CD persona is validated when bug creation with `--json` output works and returns a parseable key field.

---

## Exit Gate Verification

| Gate Criterion | Status | Evidence |
|----------------|--------|----------|
| Every user journey has at least one acceptance scenario | PASS | Journey 1: 6 scenarios (J1-HP, J1-ALT-A/B/C, J1-ERR-1/2). Journey 2: 4 scenarios (J2-HP, J2-ALT-A/B, J2-ERR-1/2). Journey 3: 4 scenarios (J3-HP, J3-DASH, J3-ANALYTICS, J3-DASHBOARD, J3-UNIFIED). |
| Every success metric has a validation method | PASS | All 5 metrics (Adoption Rate, Resolution Time, Throughput, Creation Speed, Dashboard Visibility) have measurement method, UAT validation procedure, and pass/fail criteria defined in Section 2. |
| Risk areas from research and feasibility reviews have targeted scenarios | PASS | Risk 1 (cross-cutting dispatch) -> CE-1, P1-1. Risk 2 (workflow extension) -> CE-2, P1-2. Risk 3 (dashboard overload) -> J3-DASHBOARD, P2-5. Risk 4 (E16 dependency) -> CE-2, CE-5. Risk 5 (key format) -> documented but LOW risk, no scenario needed. Risk 6 (test suite) -> CE-3, P4-3. |
| Plan is actionable for feature-level decomposition | PASS | Each scenario specifies preconditions, concrete CLI commands, expected outcomes, and pass criteria. Scenarios are prioritized (P1-P4) enabling phased test execution aligned with phased implementation. |

---

*This UAT Acceptance Plan is ready for feature-level decomposition into test strategies.*
