---
feature_key: E19-F09-sprint-selection-root-on-shark-plan
epic_key: E19
title: Sprint Selection Root on shark plan
description: Unify sprint selection under the plan surface: add the active sprint as a selection root (shark plan sprint) returning the standard read-only selection JSON — parallel_candidates capped by max_parallel_items, claim-aware and question-gate-aware like other collection roots — ordered by the sprint four-tier order (sprint_order, execution_order, priority, assigned_at). Keyed planning-state form (shark plan S###) provides a read-only execution preview for the sprint planning ceremony. Demote shark sprint next to a compatibility alias returning the top candidate (equivalent to --sequential collapse) through a deprecation window. Supersedes the minimal claim-aware exclusion-flag ask. Design context: docs/plan/E38-shark-attack-team-orchestration/proposal-parallel-team-integration.md §5.
---

# Sprint Selection Root on shark plan

**Feature Key**: E19-F09-sprint-selection-root-on-shark-plan

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Design context**: [Parallel Team Integration Proposal §5](../../E38-shark-attack-team-orchestration/proposal-parallel-team-integration.md) (E38, 2026-08-02)

---

## Goal

### Problem

Shark has two sanctioned selection/dispatch surfaces (E35): `shark plan` for
read-only selection (hierarchy roots and collection roots — claim-aware,
question-gate-aware, returning `parallel_candidates` capped by
`max_parallel_items`) and `shark next <key>` for keyed dispatch. `shark sprint
next` predates that split and is a second selector with divergent semantics:
it returns a single item, inspects no claims, and is not question-gate-aware.
Verified in code (`GetNextTask`, `internal/services/sprint_service.go:1512`):
candidates are filtered only by non-terminal status and workflow-role
eligibility, then four-tier sorted. Because claiming an item does not change
its status, repeated calls return the *same top item* for the entire duration
of its craft — a parallel coordinator can never see rank 2 while rank 1 is in
flight. The E38 parallel-team coordinator is therefore forced into client-side
backlog enumeration (proposal §5 interim), re-deriving ordering shark already
owns — the same duplication-of-selection defect class that previously
afflicted `/run-agent-team`'s hand-rolled DAG.

### Solution

Make the sprint a **collection root on `shark plan`** and retire `sprint
next`'s independent selection semantics:

- `shark plan sprint` — selects from the **active** sprint's backlog,
  returning the standard read-only selection JSON: `parallel_candidates`
  capped by `max_parallel_items`, excluding claimed and question-gated items,
  ordered by the sprint's four-tier order (`sprint_order` → `execution_order`
  → `priority` → `assigned_at`), with `--agent` filtering by the workflow
  step's role (same semantics as today's `sprint next --agent`).
- `shark plan S###` — keyed form; for a **planning-state** sprint it is a
  read-only execution preview for the planning ceremony (what would run, in
  what order), with no dispatch implied.
- `shark sprint next` — demoted to a **compatibility alias** that delegates to
  the same selection path and returns the top candidate (equivalent to a
  `--sequential` collapse), preserving its current output contract through a
  deprecation window. Existing consumers (`skills/shark-attack` execute.md
  role-aware pull, `/run-sprint`) keep working unchanged.

This supersedes the earlier minimal ask (a claim-aware `--exclude` flag on
`sprint next`): one selection surface instead of two selectors, per the
project's surface-conflicts-pick-one rule.

### Impact

- The E38 team coordinator's sprint-mode selection collapses from client-side
  backlog enumeration to a single `shark plan sprint` call (proposal §5 marks
  the enumeration as interim pending this feature).
- Sprint selection gains claim- and question-gate-awareness for free, closing
  the double-dispatch window inherent in the current selector.
- The planning ceremony (council) gains a read-only preview of a staged
  sprint before the owner activates it.

---

## User Stories

**Story 1**: As a team coordinator running a parallel sprint wave, I want
`shark plan sprint` to return the ordered, unclaimed, unblocked candidate set
so that I can top up idle teammates without re-deriving sprint order myself.

**Acceptance Criteria**:
- [ ] Returns standard selection JSON identical in shape to other collection
      roots, capped by `max_parallel_items`.
- [ ] After a candidate is claimed, a re-invocation excludes it and reveals
      the next-ranked item (the property `sprint next` lacks today).
- [ ] Question-blocked items are excluded while their Question is open.

**Story 2**: As the sprint-planning council, I want `shark plan S###` on a
planning-state sprint so that we can preview execution order before proposing
activation to the owner.

**Acceptance Criteria**:
- [ ] Works for planning-state sprints without requiring activation.
- [ ] Response is clearly a preview: read-only, no dispatch prompt, no claim.

**Story 3**: As an existing harness/skill using `shark sprint next --agent`,
I want unchanged behavior so that nothing breaks before I migrate.

**Acceptance Criteria**:
- [ ] `sprint next` returns the same top candidate the plan root ranks first.
- [ ] `--agent` filtering behavior and output fields are preserved.
- [ ] Help text carries a deprecation pointer to `shark plan sprint`.

---

## Requirements

### Functional Requirements

1. **REQ-F-001 — Sprint collection root**: `shark plan sprint` selects from
   the active sprint's backlog and returns the standard selection JSON
   (`parallel_candidates` / single-candidate shapes, `max_parallel_items`
   cap). **Must-Have**.
2. **REQ-F-002 — Eligibility filtering**: candidates are non-terminal,
   unclaimed, and not question-gated, consistent with existing collection
   roots. **Must-Have**.
3. **REQ-F-003 — Sprint ordering**: candidates are stably ordered
   `sprint_order` → `execution_order` → `priority` (1 = highest) →
   `assigned_at`, matching the documented `sprint next` order. **Must-Have**.
4. **REQ-F-004 — Role filter**: `--agent <type>` filters by the entity's
   current workflow-step role (not roster identity), matching today's
   `sprint next --agent` semantics. **Must-Have**.
5. **REQ-F-005 — All entity types**: task, bug, change-card, and tech-debt
   members are directly dispatchable candidates; feature and epic members are
   returned as candidates marked for expansion (caller expands via
   `shark plan <key>`) — never auto-expanded, never silently skipped.
   **Must-Have**.
6. **REQ-F-006 — Keyed sprint form**: `shark plan S###` accepts active and
   planning-state sprints; planning-state responses are previews with no
   dispatch metadata. **Should-Have**.
7. **REQ-F-007 — Compatibility alias**: `shark sprint next` delegates to the
   same selection path, returns the top candidate, preserves its current
   output contract, and emits a deprecation notice in `--help` (not in
   JSON output). **Must-Have**.

### Non-Functional Requirements

1. **REQ-NF-001 — Read-only guarantee**: no `shark plan` invocation (any
   root) claims, heartbeats, releases, mutates status, or spawns anything.
   Verified by test asserting no DB writes occur during selection.
2. **REQ-NF-002 — No hardcoded statuses**: terminal/eligibility checks use
   `workflow.Service` per entity level; no status-name literals in the
   selection path.
3. **REQ-NF-003 — Layering**: selection logic lives in the service layer
   (plan/sprint services); the command remains a thin wrapper; no repository
   access from commands.
4. **REQ-NF-004 — No schema changes**: reads existing sprint-assignment and
   claim tables; no new tables, columns, or migrations.

---

## Acceptance Criteria

**Scenario 1: Wave selection with claims and gates**
- **Given** an active sprint with 6 eligible items, `max_parallel_items` = 5,
  one item claimed, one item question-blocked
- **When** `shark plan sprint --json` is invoked
- **Then** 4 candidates are returned in four-tier order
- **And** the claimed and question-blocked items are absent.

**Scenario 2: Rank 2 becomes visible (the defect this feature fixes)**
- **Given** a plan-sprint response whose top candidate is then claimed by a
  teammate (status unchanged)
- **When** `shark plan sprint --json` is invoked again
- **Then** the previously top candidate is excluded and the next-ranked item
  is returned.

**Scenario 3: Parent members are expandable, not expanded**
- **Given** an active sprint containing a feature key
- **When** `shark plan sprint --json` is invoked
- **Then** the feature appears as a candidate marked for expansion
- **And** none of that feature's tasks appear as sprint-root candidates.

**Scenario 4: Planning preview**
- **Given** a sprint in a planning state with staged entities
- **When** `shark plan S### --json` is invoked
- **Then** an ordered read-only preview is returned with no dispatch prompt
- **And** the database is byte-unchanged.

**Scenario 5: Alias back-compat**
- **Given** the same sprint fixture
- **When** `shark sprint next --agent=<role> --json` is invoked
- **Then** it returns exactly the top candidate `shark plan sprint
  --agent=<role>` ranks first, with the current output fields preserved.

---

## Out of Scope

1. **Concurrent active sprints** — single-active constraint stays
   (`StartSprint` enforcement unchanged). Noted: `GetNextTask` already
   iterates all execution-phase sprints, so a future lift is smaller than it
   looks; deliberately not pre-built.
2. **Claim/lease semantic changes** — selection filters on claims; it never
   modifies them.
3. **E38-F12 skill-layer work** — the team coordinator procedure consumes
   this feature but is specified separately (see design-context proposal).
4. **Changes to `sprint add` / `reorder` / `backlog` surfaces** — staging and
   ordering inputs are untouched.

---

## Success Metrics

1. **Coordinator simplification** — E38 proposal §5's client-side enumeration
   is replaced by a single `shark plan sprint` call (proposal updated on
   ship; enumeration remains documented only as a legacy-version fallback).
2. **No consumer regressions** — back-compat tests for `sprint next` (fields,
   `--agent` behavior) pass unchanged; `skills/shark-attack` execute.md and
   `/run-sprint` need no edits during the deprecation window.
3. **Selection correctness** — Scenario 2 (claim-exclusion reveals rank 2)
   is covered by an automated test; it is the property that motivated the
   feature.

---

## References

- Design context and consumer contract: `docs/plan/E38-shark-attack-team-orchestration/proposal-parallel-team-integration.md` §5 (recorded decisions §9.4)
- Code finding motivating the fix: `GetNextTask`, `internal/services/sprint_service.go:1512` (no claim inspection; same-top-item behavior) — architect consult, 2026-08-02
- Plan surface semantics: `shark plan --help` (collection roots, `max_parallel_items`, read-only guarantees); route-based workflow guide §4

*Last Updated*: 2026-08-02
