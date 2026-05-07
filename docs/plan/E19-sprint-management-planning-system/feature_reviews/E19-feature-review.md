---
epic_key: E19
title: E19 Feature Decomposition Review
review_type: feature_review
date: 2026-05-05
reviewer: uat-agent (claude opus, ready_for_feature_review template)
verdict: PASS
---

# E19 Feature Decomposition Review

**Epic**: [E19 — Sprint Management & Planning System](../epic.md)
**Date**: 2026-05-05
**Verdict**: **PASS**
**Reviewer**: ready_for_feature_review template (epic_short workflow)

---

## Summary

The five generated features cleanly cover all Must-Have and Should-Have requirements, address the polymorphic-assignment fix that triggered the prior rejection, and respect a coherent execution order (schema → lifecycle → assignment → analytics → planning/capacity). One Could-Have (REQ-F-018, sprint history per entity) is not explicitly assigned to any feature, but it is a Could-Have stretch and the schema delivered by F01 is sufficient to add it later without rework — recorded as a recommendation, not a gate failure.

The features are appropriately sized, have clear boundaries (no overlap between F03 assignment and F05 bulk assignment because F05 is feature-level/group bulk), and each description is specific enough to drive specification and assessment in the next phase.

---

## Features Reviewed

| Order | Key | Title | Status |
|------:|-----|-------|--------|
| 1 | E19-F01 | Sprint Database Schema & Core Entity Foundation | draft |
| 2 | E19-F02 | Sprint Lifecycle Management | draft |
| 3 | E19-F03 | Sprint Entity Assignment & Backlog | draft |
| 4 | E19-F04 | Sprint Analytics: Velocity, Burndown & Summary | draft |
| 5 | E19-F05 | Sprint Planning View & Capacity Management | draft |

All five features are in `draft` status (expected at this gate). Feature-body PRDs are template skeletons — substantive scope lives in the frontmatter `description` field, which is the contract this review evaluates. Detailed feature PRDs are produced in the subsequent decomposition phase.

---

## 1. Requirements Coverage Matrix

Functional requirements (REQ-F-###) sourced from `requirements.md`. Mapped by reading each feature's `description` frontmatter against the requirement's acceptance criteria.

### Must-Have Functional Requirements

| Req | Title | Primary Feature(s) | Coverage |
|-----|-------|--------------------|----------|
| REQ-F-001 | Sprint Creation | E19-F02 | Full — `description` explicitly names `shark sprint create` and references REQ-F-001 |
| REQ-F-002 | Sprint Status Transitions | E19-F02 | Full — `description` enumerates the full state machine (planning → active → closing → completed, plus cancelled/archived) and the single-active-sprint constraint |
| REQ-F-003 | Sprint CRUD Operations | E19-F02 (CRUD) + E19-F01 (S### key auto-detection) | Full — F02 covers `get/list/update/delete`; `shark get S024` auto-detect comes from F01's key-parsing-service extension |
| REQ-F-004 | Individual Entity Assignment (task/bug/CC/TD) | E19-F03 | Full — `description` explicitly enumerates all four entity types and the partial unique index that enforces single-active-sprint membership |
| REQ-F-005 | Sprint Backlog View | E19-F03 | Full — `description` explicitly states "the backlog view groups all entity types by status so PMs see ALL in-flight work — tasks, bugs, change-cards, and tech-debt — in one place" |
| REQ-F-006 | Sprint Close with Carryover | E19-F03 | Full — `description` calls out the carryover transaction (`--carryover=next|backlog`) as the highest-complexity operation in the epic |
| REQ-F-007 | Velocity Calculation | E19-F04 | Full — `description` names "shark sprint velocity (historical Σ size completed per sprint with trailing average)" and unsized-entity reporting |
| REQ-F-008 | Sprint Burndown | E19-F04 | Full — `description` names "day-by-day ideal vs. actual remaining Σ size reconstructed from task_history, rendered as a text table" |
| REQ-F-009 | Sprint Summary Report | E19-F04 | Full — `description` covers planned/completed size, task counts, velocity comparison, unsized counts, and `--detailed` cycle-time-by-phase |
| REQ-F-010 | Sprint Database Tables | E19-F01 | Full — `description` enumerates all three tables (sprints, sprint_assignments, sprint_capacity), the polymorphic (entity_type, entity_id) pattern, partial unique index, schema version bump, and migration in `internal/db/db.go` |

### Should-Have Functional Requirements

| Req | Title | Primary Feature(s) | Coverage |
|-----|-------|--------------------|----------|
| REQ-F-011 | Sprint Planning Command | E19-F05 | Full — `description` names "shark sprint plan (composite view of unassigned backlog sorted by priority/dependency, current capacity utilization per agent type, and readiness score)" |
| REQ-F-012 | Bulk Entity Assignment | E19-F05 | Full — `description` explicitly mentions "shark sprint add --bulk for feature-level bulk assignment". *Note*: feature description focuses on the feature-level bulk path; `--bulk-bugs / --bulk-tech-debt / --bulk-changes` from the AC list also live here by virtue of being part of REQ-F-012 — flagged as a clarification recommendation below, not a coverage gap. |
| REQ-F-013 | Sprint Readiness Score | E19-F05 | Full — `description` enumerates all six scoring factors (capacity utilization, dependency satisfaction, task count, agent balance, sizing coverage, oversized-entity flag) |
| REQ-F-014 | Agent Capacity Configuration | E19-F05 | Full — `description` names "shark sprint capacity set/show for per-sprint agent capacity CRUD" |
| REQ-F-015 | Default Capacity Configuration | E19-F05 | Full — `description` names "the sprint_defaults config section in .sharkconfig.json for team-level defaults" |

### Could-Have Functional Requirements

| Req | Title | Primary Feature(s) | Coverage |
|-----|-------|--------------------|----------|
| REQ-F-016 | Sprint Auto-Creation | E19-F05 (stretch) | Conditional — F05 `description` says "also enables the Could Have auto-creation (REQ-F-016) and dashboard integration (REQ-F-017) if time permits" |
| REQ-F-017 | `shark status` Dashboard Integration | E19-F05 (stretch) | Conditional — same stretch clause as REQ-F-016 |
| REQ-F-018 | Sprint History for Entities | **(unassigned)** | **Gap (Could-Have)** — no feature description references "sprint history" for an entity. Schema in F01 supports it (sprint_assignments retains `assigned_at`/`removed_at` rows), but no command surface owns it. See Recommendations §1. |

### Non-Functional Requirements

| Req | Title | Coverage |
|-----|-------|----------|
| REQ-NF-001 | Sprint Command Response Time | Cross-cutting — implicit in F02–F05 service implementations; F01 indexes (REQ-F-010 AC) enable the targets. No dedicated feature, which is appropriate. |
| REQ-NF-002 | Database Query Efficiency | Cross-cutting — F01 explicitly delivers the indexes named in REQ-F-010 (status, (entity_type, entity_id), sprint_id). |
| REQ-NF-003 | Sprint Assignment Consistency | F01 — partial unique index, FK to sprints, CHECK on entity_type, CHECK on sprint status. |
| REQ-NF-004 | Backward Compatibility | F01 (migration is additive; idempotent, version-bumped). |
| REQ-NF-005 | JSON Output Consistency | Cross-cutting — every F02–F05 description references `--json` and follows existing patterns. |
| REQ-NF-006 | Sprint Data Access | Cross-cutting — sprint commands inherit the existing model (local SQLite or Turso); no new auth surface needed. |

**Coverage verdict**: All Must-Have requirements are explicitly owned by a feature. All Should-Have requirements are explicitly owned by F05. One Could-Have (REQ-F-018) is unassigned but the schema supports a future drop-in — non-blocking for this gate.

---

## 2. Feature Quality

| Check | Result |
|-------|--------|
| Each feature is a cohesive, independently deliverable capability | Pass — F01 is data-only, F02/F03/F04/F05 are command-surface increments that each compose into a usable subset |
| Descriptions are specific enough for assessment & specification | Pass — every description names CLI commands, integration points (e.g. `internal/db/db.go`, `internal/keys/service.go`, `services_global.go`, `internal/status/`), and the requirement IDs it covers |
| No overlapping scope between features | Pass — F03 owns single-entity add/remove + close/carryover; F05 owns bulk add + readiness + capacity + planning view. The boundary is clean: F03 = "wire one entity into a sprint"; F05 = "plan the sprint as a whole." |
| Feature titles accurately reflect content | Pass — titles match the work surface; F03 was correctly retitled "Sprint Entity Assignment & Backlog" (reflecting polymorphic scope, not just tasks) |

**Note on body content**: The feature.md *body* sections (User Personas, User Stories, Requirements with REQ-F-001 placeholders, Out of Scope, etc.) are template scaffolding in all five files. This is the expected state at the feature-review gate — the body PRDs are produced after this gate passes, in the per-feature spec phase. The review evaluates the contract in the frontmatter `description`, which is substantive in all five files.

---

## 3. Ordering & Dependencies

Execution order in shark: **F01 → F02 → F03 → F04 → F05**.

| From | To | Dependency type | Validity |
|------|----|-----------------|----------|
| F01 | F02 | F02 needs `sprints` table + `S###` key parsing | Valid — schema must exist before lifecycle commands |
| F01 | F03 | F03 needs `sprint_assignments` (polymorphic schema) and partial unique index | Valid |
| F02 | F03 | F03's `shark sprint close --carryover` advances sprint status; depends on the lifecycle service from F02 | Valid |
| F01,F02,F03 | F04 | Analytics consume sprint data, lifecycle history, and assignment rows | Valid |
| F01,F02,F03 | F05 | Planning view & capacity consume backlog, sprint, and assignment data; readiness uses size from F03 assignments | Valid |

- No circular dependencies.
- Foundation feature (F01, schema) is correctly ordered first.
- F02 and F03 are sequenced correctly (lifecycle commands before assignment, because `shark sprint close --carryover` calls into the lifecycle state machine).
- F04 (analytics) and F05 (planning) both depend on F01–F03 but are independent of each other; running them as 4 then 5 (or in parallel after F03) is workable. The sequencing in shark reflects "Must-Have analytics first, Should-Have planning second" which matches the priority framework in `requirements.md`.

**Verdict**: Ordering is sound and matches the actual dependency graph.

---

## 4. Scope Alignment

| Check | Result |
|-------|--------|
| Features stay within epic scope | Pass — all five features map directly to requirements in `requirements.md`; nothing introduces capabilities outside the epic |
| No features that don't map to requirements | Pass — every feature description cross-references the REQ-F-### IDs it covers |
| Granularity is appropriate | Pass — F01 (schema only) is correctly small; F05 is the largest (covers REQ-F-011 through REQ-F-015 plus stretch F-016/F-017) but cohesive ("planning workflow surface"); none feel oversized for tier-2 decomposition |
| Out-of-scope items from `scope.md` are not slipped in | Pass — no feature mentions web UI, story-point ceremonies, cross-team coordination, notifications, time-tracking, or sprint templates |

---

## 5. Polymorphic-Assignment Fix Verification

The prior decomposition was rejected (2026-05-05 00:44:54) for not handling polymorphic assignment across all four entity types. Verifying the fix:

- `requirements.md` REQ-F-004 acceptance criteria explicitly enumerate `task / bug / change-card / tech-debt` add/remove (lines 80–87). ✓
- `requirements.md` REQ-F-005 acceptance criteria require entity-type tagging in backlog output and `--type` filter for all four types. ✓
- `requirements.md` REQ-F-010 specifies polymorphic `(entity_type, entity_id)` pattern, partial unique index, no `task_id` FK column, mirroring `entity_notes`. ✓
- `requirements.md` REQ-F-012 adds `--bulk-bugs / --bulk-tech-debt / --bulk-changes` flags. ✓
- E19-F01 description explicitly states all four entity types and the polymorphic pattern. ✓
- E19-F03 description explicitly enumerates "tasks (E##-F##-###), bugs (B###), change-cards (CC-###), AND tech-debt items (TD-###)" and emphasizes "ALL in-flight work … in one place." ✓
- E19-F05 description references `--bulk` for feature-level bulk; the bulk-by-entity-type variants (REQ-F-012 ACs) live in this feature too — see Recommendation §2 for an explicit call-out.

The rejection driver is fully addressed.

---

## 6. Gaps Identified

**G1 — REQ-F-018 (Sprint History for Entities) has no owning feature.**
- Severity: Low (Could-Have)
- Impact: A user asking "which sprints was T-E07-F01-001 in?" has no command path until this is added.
- Schema readiness: F01's `sprint_assignments` table preserves `assigned_at` / `removed_at` history rows, so the data is captured even without a command — the missing piece is the read surface.
- Disposition: **Non-blocking.** Recorded as a recommendation; can be added either as a small extension to F02 or as a future feature without schema change.

No other gaps identified.

---

## 7. Overlaps Identified

None. F03 and F05 both touch sprint assignment, but the line is clean:
- **F03**: per-entity add/remove (`shark sprint add S024 E07-F01-001`), backlog view, close-with-carryover.
- **F05**: composite planning view (`shark sprint plan`), bulk assignment from a feature or by entity type, capacity, readiness scoring, capacity defaults config.

---

## 8. Ordering Issues

None.

---

## 9. Recommendations (non-blocking)

These are recorded for the team to consider in the per-feature spec phase. They do **not** affect the PASS verdict.

1. **Decide who owns REQ-F-018 before starting F02.** Two reasonable options: (a) extend F02 to include `shark sprint history <entity-key>` since F02 already owns sprint read surfaces, or (b) defer to a future Could-Have feature. Option (a) is small, schema-ready, and avoids leaving a Could-Have orphaned. Either choice is fine — just record the decision in F02's spec or in the epic notes.

2. **Make F05's description reference REQ-F-012's bulk-by-entity-type flags explicitly.** The current description mentions "shark sprint add --bulk for feature-level bulk assignment" but REQ-F-012 also names `--bulk-bugs`, `--bulk-tech-debt`, `--bulk-changes`. These belong in F05 by inheritance, but spelling them out in the description avoids ambiguity when F05 enters spec-writing.

3. **F01 spec should record the exact `entity_type` enum values** (`task`, `bug`, `change_card`, `tech_debt`) and any CHECK constraint. REQ-F-010 and REQ-NF-003 imply this, but the F01 description names the values once — make sure the spec phase captures the exact strings (consistent with `entity_notes` precedent) and adds the new types to the project-wide entity-type allowlist (see CLAUDE.md memory `feedback_entity_type_check_constraints.md`).

4. **F04 spec should clarify the unsized-entity reporting path.** REQ-F-007 and REQ-F-008 both require unsized counts to surface. F04 description mentions this but the per-day burndown reporting of `unsized_remaining: N` (REQ-F-008 AC) is the subtle one — call it out so the spec writer doesn't reduce it to a single end-of-sprint number.

---

## 10. Decision

**Verdict: PASS**

All Must-Have and Should-Have requirements are owned by a feature, the polymorphic-assignment fix that triggered the prior rejection is fully reflected in both the requirements and the feature descriptions, ordering follows the actual dependency graph, no scope creep, no overlaps. The single Could-Have gap (REQ-F-018) is non-blocking and the supporting schema is in F01.

**Action**: `shark status advance E19` to move E19 from `in_feature_review` to `active`.
