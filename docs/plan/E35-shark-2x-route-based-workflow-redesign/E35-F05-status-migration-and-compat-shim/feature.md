---
feature_key: E35-F05-status-migration-and-compat-shim
epic_key: E35
title: Status migration and compat shim
description: Per-step aliases: list old status names that collapse into each step. Alias map does triple duty: (1) one-shot DB migration rewrites the live status column; (2) input compat shim accepts old status names during deprecation window; (3) history-read resolution for entities parked before migration. Rewrite live status once; leave task_history untouched (alias-resolve on read). Mid-flight in_X rows become unclaimed phase rows, safely re-dispatched. Bump CurrentSchemaVersion. Design section 7.
size: M
---

# Status migration and compat shim

**Feature Key**: E35-F05

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Design (single source of truth)**: [route-based-workflow-redesign.md](../../route-based-workflow-redesign.md) — §7, §8

---

## Goal

### Problem

Collapsing `ready_for_X` + `in_X` into a single phase (F02/F03) renames most
statuses. Existing entities carry the old status values in the live DB and in
`task_history`; hooks, scripts, and muscle memory still use the old names. A hard
cutover would orphan in-flight rows and break every external caller.

### Solution

The per-step `aliases:` field (parsed in F01) drives the whole migration (design §7).
Each new step lists the old names that collapse into it; the alias map does **triple
duty**:

1. **Migration source** — a one-shot DB migration rewrites the live `status` column
   to the new phase names.
2. **Input compat shim** — the loader accepts an old status name during a
   deprecation window so hooks/scripts/muscle-memory don't break at cutover.
3. **History-read resolution** — for an entity parked before migration and resumed
   after, old names in reads resolve through the alias map.

**Rewrite the live `status` column once; leave `task_history` untouched** — audit
trails record what actually happened, so alias-resolve old names on the rare read
instead of rewriting history. Mid-flight `in_X` rows become **unclaimed** phase rows
and get safely re-dispatched (steps are atomic, so "resume" = re-enter the phase and
re-claim — no mid-step state to reconstruct).

This feature also delivers **parking resume** (a §7 TBD): for `parking: true` steps
(`blocked`, `on_hold`) the return target is *computed* by walking back through
`task_history` to the most recent `from_status` that isn't itself a parking state
(walking past chained parks). No stored field — history is the single source of truth.

### Impact

- Existing entities migrate with zero data loss; audit history preserved.
- Old status names keep working during the deprecation window.
- Parked entities resume to the right phase from history alone.

---

## Scope

- One-shot migration: rewrite the live `status` column using the per-entity alias
  maps; bump `CurrentSchemaVersion`; idempotent and safe under the skip_migrations
  path (per database-critical rules).
- Input compat shim in the loader/service: accept aliased (old) status names and
  resolve to the canonical phase during the deprecation window.
- History-read resolution: alias-resolve old `task_history` values on read; do not
  rewrite `task_history`.
- Parking-resume: `internal/workflow/service.go` walk-back over `task_history` to
  compute the resume target for `parking: true` steps.
- Mid-flight `in_X` rows land as unclaimed phase rows (coordinate with F03 claim
  fields).

### Out of Scope

- Defining the `aliases:` field schema — F01 (this feature consumes it).
- Claim-field schema itself — F03 (this feature coordinates the combined version bump).
- Doc/vocabulary updates for the new status names — F06.

---

## Acceptance Criteria

1. After migration, the live `status` column holds new phase names per the §7
   mapping (e.g. `ready_for_research`/`in_research` → `research`).
2. `task_history` is unchanged; old values in history reads alias-resolve to the new
   phase.
3. An old status name supplied as input is accepted during the deprecation window
   and resolves to its canonical phase.
4. A mid-flight `in_X` row becomes an unclaimed `X`-phase row and is re-dispatchable.
5. A parked entity (`blocked`/`on_hold`) resumes to the most recent non-parking
   phase computed from `task_history` (chained parks walked past).
6. Migration is idempotent and `CurrentSchemaVersion` is bumped; re-running is a no-op.
7. `shark validate` (checks authored in F06) can confirm every old status has an
   alias home — this feature ensures the alias maps are complete.

---

## Verification

- Migration test (real DB): seed old statuses incl. an `in_X` and a chained park,
  run migration, assert live column rewritten and history intact.
- Service tests: input shim resolution, history-read resolution, parking walk-back
  (incl. chained parks).
- `make fmt && make lint && make test` pass.

---

## Dependencies

- **Blocked by**: F01 (`aliases:` field), F02 (new phase names / routing),
  F03 (claim fields + combined schema bump).
- **Blocks**: F06 (validate alias-coverage check, docs).

---

*Last Updated*: 2026-06-23
