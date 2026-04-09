---
epic_key: E26
title: Auto-reopen parent entities on child regression
description: When a child entity regresses from a terminal state (or a new child is added under a terminal parent), automatically walk the parent chain back to the previous non-terminal status from history.
---

# Epic E26: Auto-reopen parent entities on child regression

**Epic Key**: E26
**Business Value**: High
**Last Updated**: 2026-04-06

---

## 1. Problem Statement and Business Justification

### Problem

Parent entities (features, epics) frequently end up in `completed` prematurely and then get stranded there when the underlying work regresses. The dominant scenario: a feature is marked complete, UAT rejects the work, tasks are sent back to `in_development`, but the feature and epic stay `completed` and won't come back on their own. A developer must manually hunt down every parent in the chain and set the status back — a step that is easy to forget and leaves the project dashboard misrepresenting what is actually done.

A partial fix exists today: adding a new task to a completed feature reopens the feature. But that only handles one trigger (new child creation) at one level (feature) and does not address the UAT rejection flow, which is where the operational pain is concentrated.

### Business Justification

- **Operational pain**: Every team using the advanced workflow profile hits this bug during the UAT rejection cycle. Manual parent cleanup is the single most common manual intervention against shark today.
- **Dashboard trust**: Stale `completed` features and epics erode confidence in the shark dashboard as a source of truth, which in turn pushes teams to track status outside the system.
- **Automation enablement**: Downstream orchestrator actions assume parent status is consistent with child reality. Stale parents cause incorrect routing and missed agent dispatches.
- **Narrow blast radius, high leverage**: The fix touches a localized area (status transition logic + history lookup) but eliminates a recurring class of bugs that affects every workflow profile.

---

## 2. Goals and Success Criteria

### Goals

1. Eliminate manual parent-status cleanup after child regressions (especially UAT rejection).
2. Restore the shark dashboard as an accurate source of truth without developer intervention.
3. Unify the existing "new task reopens feature" behavior with a general cascade rule that works across all workflow profiles and at all parent levels (feature and epic).
4. Preserve workflow context: a feature that was in `in_qa` when prematurely completed returns to `in_qa`, not a generic "reopened" state.

### Measurable Success Criteria

| # | Criterion | How Measured |
|---|-----------|--------------|
| SC1 | A child backward transition out of a terminal status reopens all terminal ancestors | Integration test: regress task → assert feature and epic are no longer terminal |
| SC2 | A new child created under a terminal parent reopens all terminal ancestors | Integration test: create task under completed feature/epic → assert both reopen |
| SC3 | Each ancestor returns to its own previous non-terminal status from `status_history` | Unit + integration tests: assert restored status matches the most recent non-terminal entry per ancestor |
| SC4 | Cascade walks the full chain task → feature → epic in a single transition operation | Integration test: single task transition triggers ≥2 parent updates |
| SC5 | Behavior works on basic, advanced, and custom workflow profiles with zero config changes | Profile-parameterized tests (basic + advanced + custom fixture) all pass |
| SC6 | Bugs and change-cards never trigger parent reopens | Negative tests: regress a bug/change-card linked to a completed feature → feature stays completed |
| SC7 | Auto-reopen transitions are recorded in `status_history` with an `auto_reopen` reason referencing the triggering child key | Repository-level test on history rows |
| SC8 | Reopening an already non-terminal ancestor is a no-op | Idempotency test: trigger cascade twice; only the first writes history |
| SC9 | Existing "new task reopens feature" tests continue to pass under the unified implementation | Existing test suite green; no duplicated code paths |
| SC10 | Manual parent-cleanup interventions drop to zero in dogfood usage post-rollout | Tracked via absence of manual `status set` commands targeting features/epics in regression scenarios |

---

## 3. Scope

### In Scope

- **Triggers**:
  - Backward transition: a task, feature, or epic moves from a terminal status to a non-terminal status.
  - New child creation: a new task is created under a terminal feature, or a new feature is created under a terminal epic.
- **Cascade levels**: task → feature → epic. Full chain walk on every trigger.
- **Reopen target**: per-ancestor lookup in that ancestor's `status_history` for the most recent non-terminal status the ancestor previously held.
- **Fallback**: when an ancestor has no prior non-terminal history, fall back to the workflow's default initial status.
- **Audit trail**: every auto-reopen writes a `status_history` row with a distinct reason identifying the triggering child.
- **Workflow profile compatibility**: implementation reads terminal status classification from existing config (`status_metadata` / phase grouping). Works on basic, advanced, and custom profiles.
- **Idempotency**: triggering an auto-reopen on an already non-terminal ancestor is a no-op (no history row, no status change).
- **Unification**: existing "new task reopens feature" behavior is refactored into the new unified cascade path. No two parallel implementations.
- **Refinement-time decisions** to be locked during BA/tech refinement (not deferred indefinitely):
  - Whether to add an explicit `is_terminal: true` flag to `status_metadata` or rely on existing phase classification.
  - Whether to add a per-profile `auto_reopen_enabled` opt-out flag.
  - Whether to add a per-workflow `reopen_fallback` status override.

### Out of Scope

- **Bugs and change-cards**: standalone entities; never trigger parent reopens even when linked to a feature or epic.
- **Forward-direction cascades**: this epic does not auto-complete parents when all children become terminal. That is a separate concern handled by existing rollup logic.
- **Cross-epic reopens**: a task only reopens its own feature and epic. There is no concept of reopening sibling or upstream-dependency entities.
- **Notification/alerting**: no email, webhook, or external notification when an auto-reopen fires. History row is the only artifact.
- **UI changes**: this epic targets the CLI/service/repository layers. No HTTP API or dashboard UI work is in scope (downstream consumers benefit automatically once the data is correct).
- **Retroactive reopen of pre-existing stale parents**: the change applies only to transitions that occur after the feature ships. No data migration or backfill scan.
- **Disabling cascades for orchestrator-driven transitions**: orchestrator-triggered child regressions cascade exactly like manual ones (this is desired behavior, not an exclusion).

---

## 4. Constraints and Assumptions

### Constraints

- **Workflow-agnostic**: implementation must not hardcode any status name (`completed`, `in_progress`, `in_qa`, etc.). Terminal classification must come from config metadata.
- **Single transaction**: the original child transition and all cascading parent transitions must occur within one database transaction. A partial cascade (child reopened but parent missed) is unacceptable.
- **Backward compatible**: existing CLI commands, JSON output schemas, and exit codes do not change. The cascade is observable only via additional `status_history` rows and updated parent statuses.
- **Performance**: the full cascade (task → feature → epic) plus history walks must complete within the existing transition latency budget — target ≤50ms additional overhead per transition on a typical project.
- **No new external dependencies**: implementation lives in existing service/repository layers (`internal/services/`, `internal/repository/`, `internal/workflow/`).
- **Database schema**: prefer to avoid new tables. May add a column to `status_history` if needed to record the auto-reopen reason and triggering child key. Any schema change must follow the migration protocol in `.claude/rules/database-critical.md` (bump `CurrentSchemaVersion`, notify developer to flip `skip_migrations`).

### Assumptions

- `status_history` already records every status transition for tasks, features, and epics with a timestamp ordering sufficient to reconstruct prior status. (Verified during refinement.)
- The existing `workflow.Service` exposes (or can expose) a method to classify a status as terminal vs non-terminal based on the active workflow profile.
- All entity status transitions (manual, CLI, orchestrator, HTTP) flow through a common service-layer entry point where the cascade hook can be installed without duplication.
- Bugs and change-cards do not currently participate in parent rollups, so excluding them does not require defensive code paths beyond the entity-type check at the cascade entry point.
- Auto-reopen is desirable behavior for all current users; default is **on**. An opt-out config flag may be added during refinement but is not the default state.

---

## 5. Stakeholder Impact

### Developers (primary beneficiaries)

- **Before**: After a UAT rejection, manually run `shark status set E07-F01 in_qa` and `shark status set E07 in_development` (or whichever statuses apply). Easy to forget; easy to set the wrong status.
- **After**: Regress the task; parents reopen automatically to their correct historical statuses. Zero manual parent cleanup.

### QA / Product Owners

- **Before**: Dashboard shows features as `completed` even after rejection, causing confusion about what is actually ready for sign-off.
- **After**: Dashboard reflects reality immediately. `feature list` and `epic status` rollups are trustworthy.

### Orchestrator and Automation Consumers

- **Before**: Orchestrator actions tied to specific statuses miss reopened work because parent status is stale. Workflow routing breaks silently.
- **After**: Reopening a parent fires the same orchestrator side-effects as a manual transition into the same status. Routing is consistent with reality.

### AI Agents (DevAgent, BA, Tech Lead, QA, PO)

- **Before**: Agents querying parent status for routing decisions get incorrect data and may skip work or dispatch to the wrong phase.
- **After**: Parent status always matches child reality, so agent dispatch is correct without defensive double-checks.

### Existing Users / No-Op Stakeholders

- **Basic profile users**: see no behavior change unless they actively use terminal-status regression. When they do, parents reopen correctly with no setup.
- **Custom profile users**: behavior adapts to their config automatically as long as terminal statuses are classifiable from `status_metadata`.

### Shark Maintainers

- One unified cascade code path replaces the existing ad-hoc "new task reopens feature" branch. Reduced surface area for future bugs.
- New schema column (if added) is documented in the migration protocol.

---

## 6. High-Level Acceptance Criteria (UAT Scenarios)

### UAT-1: UAT rejection cycle (the primary scenario)

**Given** epic `E07` is `completed` (history shows it was previously `in_development`)
**And** feature `E07-F01` is `completed` (history shows it was previously `in_qa`)
**And** task `E07-F01-003` is `completed`
**When** a developer runs `shark status set E07-F01-003 in_development`
**Then** task `E07-F01-003` moves to `in_development`
**And** feature `E07-F01` moves to `in_qa`
**And** epic `E07` moves to `in_development`
**And** `status_history` for the feature and epic each contain a new row with reason indicating auto-reopen triggered by `E07-F01-003`
**And** all three updates are committed atomically (no partial state visible to other readers).

### UAT-2: New task under a completed feature

**Given** feature `E07-F01` is `completed` and epic `E07` is `completed`
**When** a developer runs `shark task create E07 F01 "Follow-up fix"`
**Then** the new task is created in the workflow's default initial status
**And** feature `E07-F01` reopens to its previous non-terminal status from history
**And** epic `E07` reopens to its previous non-terminal status from history
**And** the existing "new task reopens feature" test suite still passes against the unified implementation.

### UAT-3: Idempotency under concurrent regressions

**Given** feature `E07-F01` is `completed` and epic `E07` is `completed`
**When** two tasks `E07-F01-001` and `E07-F01-002` are regressed in rapid succession
**Then** the feature and epic each reopen exactly once
**And** `status_history` contains exactly one auto-reopen row per ancestor (not two)
**And** the second regression observes the parents already non-terminal and writes no additional history.

### UAT-4: Workflow profile compatibility

**Given** a project using the **basic** workflow profile with feature in `completed`
**When** a child task regresses to `in_progress`
**Then** the feature reopens to its previous non-terminal status (`in_progress` per basic profile history)

**And given** a project using a **custom** workflow profile with non-standard status names
**When** a child regresses
**Then** the cascade still fires correctly using the profile's terminal classification, with no code changes or hardcoded status names involved.

### UAT-5: Bugs and change-cards do not cascade

**Given** bug `B042` is linked to feature `E07-F01` (which is `completed`)
**When** the bug regresses from `closed` to `open`
**Then** feature `E07-F01` remains `completed`
**And** epic `E07` remains `completed`
**And** no auto-reopen history rows are written.

### UAT-6: Fallback when no prior non-terminal history exists

**Given** feature `E07-F01` is `completed` but its `status_history` contains only the transition into `completed` (no prior non-terminal entries)
**When** a child task regresses
**Then** feature `E07-F01` reopens to the workflow's default initial status (or the configured `reopen_fallback` if added during refinement)
**And** the auto-reopen history row records that the fallback path was used.

### UAT-7: Audit trail is distinguishable from manual transitions

**Given** any auto-reopen occurs
**When** a developer runs `shark status history E07-F01`
**Then** the auto-reopen row is clearly labeled (e.g., reason = `auto_reopen: triggered by E07-F01-003 regression`)
**And** the row is visually distinguishable from manual `shark status set` transitions in both human and `--json` output.

### UAT-8: Existing dashboard rollups reflect reopened state immediately

**Given** an auto-reopen has just fired
**When** a developer runs `shark status` (project dashboard) or `shark epic status E07`
**Then** the rollup shows the epic and feature in their reopened (non-terminal) state
**And** task counts and progress percentages reflect the regressed work as in-flight, not completed.

---
