---
epic_key: E19
title: E19 Sprint Management & Planning System — Feature Decomposition Review
verdict: PASS
date: 2026-05-05
---

# E19 Feature Decomposition Review

**Epic**: [E19 — Sprint Management & Planning System](../docs/plan/E19-sprint-management-planning-system/epic.md)

**Verdict**: **PASS**

The decomposition into five features cleanly covers every Must-Have requirement (REQ-F-001 through REQ-F-010), every Should-Have requirement (REQ-F-011 through REQ-F-015), and explicitly addresses the optional Could-Haves (REQ-F-016 and REQ-F-017) under F05's stretch scope. Boundaries are clear, ordering is correct (foundation first, then lifecycle, assignment, analytics, and planning view last), and there are no overlaps or scope-creep features.

The feature-PRD bodies (problem/solution, personas, user stories, requirements, acceptance criteria, success metrics) are still in template-stub state. **That is expected at this stage** — the feature decomposition step produces titles + frontmatter descriptions that scope each feature. Filling out the per-feature PRD body is the next workflow step (`ready_for_specification` -> PRD authoring per feature). This review is gating the *decomposition*, not the PRD bodies.

---

## 1. Requirements Coverage Matrix

### Must-Have Requirements

| Req ID | Requirement | Owning Feature | Coverage |
|---|---|---|---|
| REQ-F-001 | Sprint Creation (`shark sprint create` with dates and goal, auto S### key, date validation, --json) | E19-F02 (Sprint Lifecycle Management) | Full |
| REQ-F-002 | Sprint Status Transitions (planning -> active -> closing -> completed, single-active enforcement, audit history) | E19-F02 (Sprint Lifecycle Management) — close-with-carryover terminal step shared with F03 | Full |
| REQ-F-003 | Sprint CRUD Operations (get, list with --status filter, update, delete planning-only, S### auto-detection in `shark get`) | E19-F02 (Sprint Lifecycle Management) | Full |
| REQ-F-004 | Individual Task Assignment (`shark sprint add/remove`, single-active-sprint-per-task constraint, capacity warning) | E19-F03 (Sprint Task Assignment & Backlog) | Full |
| REQ-F-005 | Sprint Backlog View (`shark sprint backlog`, status grouping, --blocked filter, completion %) | E19-F03 (Sprint Task Assignment & Backlog) | Full |
| REQ-F-006 | Sprint Close with Carryover (`--carryover=next\|backlog`, auto next-sprint creation, completion record, configurable default) | E19-F03 (Sprint Task Assignment & Backlog) — atomic carryover transaction explicitly called out as highest-complexity operation | Full |
| REQ-F-007 | Velocity Calculation (Σ size per sprint, trailing average, unsized_completed reporting, "insufficient data" message) | E19-F04 (Sprint Analytics) | Full |
| REQ-F-008 | Sprint Burndown (ideal + actual lines, daily reconstruction, unsized_remaining, mid-sprint scope changes, ASCII chart) | E19-F04 (Sprint Analytics) — burndown reconstructed from task_history per F04 description | Full |
| REQ-F-009 | Sprint Summary Report (planned/completed Σ size, velocity comparison, unsized counts, --detailed cycle-time-by-phase, agent utilization) | E19-F04 (Sprint Analytics) | Full |
| REQ-F-010 | Sprint Database Tables (sprints, sprint_assignments, sprint_capacity, FKs, indexes, idempotent migration, schema version bump) | E19-F01 (Sprint Database Schema & Core Entity Foundation) | Full |

### Should-Have Requirements

| Req ID | Requirement | Owning Feature | Coverage |
|---|---|---|---|
| REQ-F-011 | Sprint Planning Command (composite view: backlog + capacity + readiness in one command, --json) | E19-F05 (Sprint Planning View & Capacity Management) | Full |
| REQ-F-012 | Bulk Task Assignment (`shark sprint add --bulk <feature-key>`, capacity warnings) | E19-F05 (Sprint Planning View & Capacity Management) | Full |
| REQ-F-013 | Sprint Readiness Score (0-100, capacity utilization / dependency satisfaction / task count / agent balance / sizing coverage / oversized-entity factors) | E19-F05 (Sprint Planning View & Capacity Management) — F05 description enumerates all six factors | Full |
| REQ-F-014 | Agent Capacity Configuration (`shark sprint capacity set/show`, allocated computed at query time, unsized_assigned reporting) | E19-F05 (Sprint Planning View & Capacity Management) | Full |
| REQ-F-015 | Default Capacity Configuration (sprint_defaults.capacity in .sharkconfig.json, per-sprint override) | E19-F05 (Sprint Planning View & Capacity Management) | Full |

### Could-Have Requirements

| Req ID | Requirement | Owning Feature | Coverage |
|---|---|---|---|
| REQ-F-016 | Sprint Auto-Creation (sprint_defaults.auto_create) | E19-F05 (stretch scope, explicitly called out in description) | Optional / scoped |
| REQ-F-017 | `shark status` dashboard sprint integration | E19-F05 (stretch scope, explicitly called out) | Optional / scoped |
| REQ-F-018 | Sprint History per Entity (`shark sprint history <task-key>`) | Not explicitly assigned to any feature | **Gap (minor) — see Recommendations** |

### Non-Functional Requirements

| Req ID | Requirement | Owning Feature(s) | Coverage |
|---|---|---|---|
| REQ-NF-001 | Command response time targets | All five features (each command implementer's responsibility) | Cross-cutting; addressed implicitly via existing repository/service patterns |
| REQ-NF-002 | Indexed query lookups | E19-F01 (indexes defined in migration) + each query implementer | Full |
| REQ-NF-003 | Sprint assignment referential integrity (FKs, partial unique index, status CHECK) | E19-F01 (constraints defined at schema layer) — partial unique index for one-active-sprint-per-entity called out in F03 description | Full |
| REQ-NF-004 | Backward compatibility (additive only) | E19-F01 (additive migration design) | Full |
| REQ-NF-005 | --json / --field consistency on every command | E19-F02, F03, F04, F05 | Full (consistent with existing CLI patterns) |
| REQ-NF-006 | Sprint data inherits existing access model | E19-F01, F02 (no new auth surfaces introduced) | Full |

---

## 2. Gaps Identified

### Minor Gap (Non-Blocking)

- **REQ-F-018 (Sprint History per Entity)** is a Could-Have requirement (`shark sprint history <task-key>` showing all sprints a task was assigned to historically). It is not explicitly named in any of the five feature descriptions. The data is preserved by the schema (sprint_assignments has `removed_at` for soft-delete history per F01), so the *capability* is enabled — but the command surface is not assigned to a feature.

  **Severity**: Minor / Could-Have. This is a deferrable Could-Have and the underlying data is captured. It can be added to F03 (assignment commands) or deferred to a future feature without affecting the decomposition validity. **Not gating PASS.**

### Template-Stub Bodies (Acknowledged, Not Gating)

- All five feature.md bodies (problem/solution, personas, user stories, requirements, acceptance criteria, success metrics) are template scaffolding. The frontmatter `description` is the substantive decomposition artifact and is sufficient at this stage for review. **PRD authoring is the next workflow phase**, not part of this gate.

---

## 3. Overlaps Identified

**None.** Boundaries are crisp:

- **F01** owns *only* schema, migration, and key-format parsing — no commands.
- **F02** owns *only* sprint-entity CRUD and lifecycle state machine — no task assignment.
- **F03** owns *only* task-to-sprint linkage commands and the carryover transaction (which is correctly placed because carryover is fundamentally an assignment-table operation that *triggers* the F02 lifecycle close transition; the description correctly identifies this as a shared responsibility where F03 implements the transactional logic and calls into F02's lifecycle service).
- **F04** owns *only* read-side analytics — no writes.
- **F05** owns *only* the planning workflow surface (composite views, bulk operations, readiness scoring, capacity config). Capacity config is correctly placed here rather than in F02 because it's tightly coupled to the planning workflow and the readiness score.

**Note on F02/F03 boundary**: The shared close-with-carryover behavior is the only cross-feature interaction. F02's `shark sprint close` command initiates the lifecycle transition; F03's carryover service performs the atomic reassignment + status advance. The F03 description explicitly acknowledges this ("atomically reassigns incomplete entities to the next sprint... and advances sprint status to completed") and assigns the implementation to F03 since the complexity is in the assignment-table mutation. This is a clean dependency, not an overlap.

---

## 4. Ordering Issues

**None.** The execution order is correct and dependency-driven:

```
F01 (schema)  ->  F02 (lifecycle)  ->  F03 (assignment + carryover)  ->  F04 (analytics)
                                                                     \-> F05 (planning view)
```

| Feature | Depends On | Justification |
|---|---|---|
| F01 | (none — foundational) | Schema must exist before any sprint data can be persisted |
| F02 | F01 | Sprint CRUD requires `sprints` table and S### key parsing |
| F03 | F01, F02 | Assignment requires `sprint_assignments` table AND a sprint to assign to (F02 produces sprints in `planning` status); carryover invokes lifecycle close from F02 |
| F04 | F01, F02, F03 | Analytics read assignment data; "completed" sprints exist only after F03's close-with-carryover runs |
| F05 | F01, F02, F03 | Planning view reads backlog (assignment data from F03), capacity (F01 schema), and computes readiness over assigned entities |

**No circular dependencies.** F05 does not depend on F04 (planning view uses raw assignment data, not analytics output), which means F04 and F05 can be developed in parallel after F03 — a useful property the decomposition preserves.

**Foundation-first principle**: Satisfied. F01 is purely additive schema with no behavior, sized as the smallest feature, and unblocks the four feature streams above it.

---

## 5. Scope Alignment

### Stays Within Epic Scope

All five features map directly to capabilities described in the epic's "Solution" section (lifecycle commands, assignment, carryover, analytics, planning view, capacity model). No feature introduces functionality outside the epic.

### Scope Exclusions Respected

The features correctly exclude items from scope.md:

- No web UI (CLI-first; F03/F04 explicitly use text-based output)
- No estimation ceremonies (E19 *consumes* size from E07-F42; no feature provides estimation tooling)
- No cross-team coordination (single-database scope)
- No notifications (data-providing only)
- No time tracking (capacity is in abstract points)
- No sprint templates beyond the cadence-only auto-creation in F05's stretch

### E07-F42 Dependency Handling

The epic correctly declares E07-F42 (entity `size` field) as a dependency. F04 and F05 both depend on size for their core logic and the descriptions reference graceful handling of unsized entities (separate `unsized_*` counts rather than 0-distortion of metrics) — matching the epic's "unsized entities contribute 0 and are surfaced separately" rule.

### Granularity

Feature sizing looks appropriate:

- F01: Smallest, schema + migration + key parsing only (XS-S work).
- F02: Medium, ~7 CLI commands plus state machine and one-active-sprint constraint.
- F03: Medium-large, ~4 commands but one (carryover) is the highest-complexity operation in the epic, correctly flagged in the description.
- F04: Medium-large, three analytics commands with non-trivial reconstruction logic for burndown.
- F05: Largest by surface area, ~5 commands plus config schema, but each command is a relatively thin composition over F02/F03 reads.

No feature is too small to deliver standalone value (each adds a coherent CLI surface or unblocks downstream work), and no feature is so large that it should be split further.

---

## 6. Recommendations

### Non-Blocking (Optional)

1. **Assign REQ-F-018 (`shark sprint history <entity-key>`) to a feature.** Lowest-friction landing spot is F03 (where the assignment commands live), since the data is already captured by F01's `sprint_assignments.removed_at` column. Alternatively, defer explicitly by noting it as out-of-scope for E19 in a follow-up. Either is acceptable; the schema already enables it.

2. **Note for PRD authoring (next phase)**: When each feature's PRD body is filled in, ensure:
   - F03's PRD calls out the partial unique index `WHERE removed_at IS NULL` from REQ-NF-003 as a hard acceptance criterion.
   - F04's PRD specifies the burndown reconstruction algorithm (using task_history end-of-day snapshots) since this is the most algorithmically novel piece in the epic.
   - F05's PRD documents the readiness-score factor weights (the requirements list factors but not weights — a PRD-level decision).
   - F02's PRD enumerates the *exact* terminal-state semantics (planning -> active -> closing -> completed vs. cancelled vs. archived); the epic shows only the happy path.

3. **Consider explicit cross-feature interface contracts** for F02<->F03 carryover and F02<->F05 capacity defaults. This is a PRD-phase concern, not a decomposition issue, but worth flagging for the spec authors.

### Blocking

**None.** No issues require returning the epic to `ready_for_decomposition`.

---

## Decision

**Verdict: PASS** — All Must-Have and Should-Have requirements are covered with clean boundaries and correct dependency-driven ordering. The single Could-Have gap (REQ-F-018) is non-gating and the data foundation supports it.

**Action**: Advance E19 to the next workflow status.
