---
feature_key: E35-F02-outcome-routing-and-release
epic_key: E35
title: Outcome routing and release
description: Keystone. Per-step outcomes: map replaces status_flow. advance becomes release(outcome): caller emits a semantic outcome, engine resolves step.outcomes[outcome] and advances atomically; set-status --force retained as human escape hatch. Service ValidateTransition/GetValidTransitions derive from outcomes. Core outcome vocabulary pass/fail/blocked required per non-terminal step; extras allowed. Decisions D1, D2, D4, D7.
size: L
---

# Outcome routing and release

**Feature Key**: E35-F02

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Design (single source of truth)**: [route-based-workflow-redesign.md](../../route-based-workflow-redesign.md) — D1, D2, D4, D7, §3, §8

---

## Goal

### Problem

Routing today is engine-pushed and forward-only: `shark next` dispatches the
current step but nothing lets a skill *report an outcome* that decides the next
step. Skills therefore can't be topology-blind — to move work along they must know
target status names, which couples them to one workflow's shape.

### Solution

This is the **keystone** of the redesign (design §3). A skill does its craft and
returns a semantic release payload — `{ outcome, reason, log }` — never naming a
status (D1). The workflow YAML owns routing: each non-terminal step has an
`outcomes:` map that the engine looks up (D2):

```yaml
qa:
  outcomes:
    pass: approval
    fail: development      # routes back — impossible in the old forward-only model
    blocked: on_hold
```

`advance` becomes `release(outcome)` (D4): the caller emits the outcome, the engine
resolves `step.outcomes[outcome]` and advances **atomically**. `set-status --force`
stays as the human escape hatch. The service's `ValidateTransition` /
`GetValidTransitions` derive from `outcomes` instead of `status_flow`.

Outcome vocabulary is **core + extras** (D7): every non-terminal step must define
`pass`, `fail`, and `blocked`; steps may add extras (`dead-end`, …). Engine routing
is a pure map lookup; `validate` enforces that the core outcomes exist and that
every outcome target names a defined step.

### Impact

- Skills become portable — reordering or renaming steps never touches a skill.
- Back-routing (`fail → development`) is expressible for the first time.
- The transition graph has a single source: the per-step `outcomes:` map.

---

## Scope

- Wire `outcomes:` semantics on the `Step` struct delivered by F01 (resolution,
  not just parsing).
- `internal/workflow/service.go`: `ValidateTransition` / `GetValidTransitions`
  derive from `outcomes`; add outcome resolution (`Resolve(step, outcome) → target`).
- Advance path: `release(outcome)` — `internal/cli/commands` advance /
  `next-status` gain `--outcome`; resolve + advance atomically; `set-status --force`
  retained unchanged.
- `internal/cli/commands/next.go`: read the merged step; emit the same wire shape
  where possible (the release payload contract documented for the harness in F06).
- `shark validate`: core outcomes present on every non-terminal step; every
  outcome target resolves to a defined step; terminal steps carry no outcomes.

### Out of Scope

- The claim/lease (who holds the entity while it's being worked) — F03.
- Parking-resume history walk-back — F05 (depends on migration + history reads).
- Harness `SKILL.md` rewrite and doc vocabulary — F06.

---

## Acceptance Criteria

1. `shark advance <key> --outcome=pass` resolves `step.outcomes[pass]` and parks
   the entity at the target phase; an unknown outcome is a clear error.
2. `--outcome=fail` can route to an *earlier* step (back-routing works).
3. `set-status <key> <status> --force` still sets status directly, bypassing
   outcome resolution.
4. Service transition checks derive from `outcomes` (no `status_flow` references
   remain in `internal/workflow/service.go`).
5. `shark validate` fails when a non-terminal step omits a core outcome
   (`pass`/`fail`/`blocked`) or names an outcome target that isn't a defined step.
6. No status names hardcoded in Go — routing is a YAML map lookup throughout.

---

## Verification

- Service tests (mocked) cover `Resolve`, core-outcome validation, and
  unknown-outcome / unknown-target error paths.
- CLI tests (mocked services) cover `advance --outcome`, back-routing, and
  `set-status --force`.
- End-to-end on a canonical workflow: claim-free advance through `pass` and a
  `fail` back-route lands at the expected phases.
- `make fmt && make lint && make test` pass.

---

## Dependencies

- **Blocked by**: F01 (merged `Step` struct).
- **Blocks**: F03 (release is half of claim/release), F05, F06.
- Can run in parallel with F04.

---

*Last Updated*: 2026-06-23
