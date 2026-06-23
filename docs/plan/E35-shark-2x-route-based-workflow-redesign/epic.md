---
epic_key: E35
title: Shark 2.x — Route-Based Workflow Redesign
description: Replace the ready_for_X/in_X status model and the status_flow+status_metadata two-map split with an outcome-routing contract. Skills become topology-blind and return semantic {outcome, reason, log}; the workflow YAML owns routing via a per-step outcomes: map; status collapses to a pure phase with a claim/session lease tracking in-flight work; advance becomes release(outcome). Adds a master index + bundle-rooted resolution and a one-shot status migration via per-step aliases. Builds on E32 (YAML workflows, shark next, embedded FS). Design: docs/plan/route-based-workflow-redesign.md. Six features.
size: XL
---

# Shark 2.x — Route-Based Workflow Redesign

**Epic Key**: E35

> The **single source of design truth** for this epic is
> [`docs/plan/route-based-workflow-redesign.md`](../route-based-workflow-redesign.md).
> Features under this epic REFERENCE that document for the locked decisions
> (D1–D7), the contract, the consolidated step schema, and the resolved TBDs —
> they must not restate them. Each feature carves out one slice of the design and
> records only its own acceptance criteria and verification.

---

## Goal

### Problem

E32 moved workflows from one JSON file to per-entity YAML, but three frictions remain:

1. **Two parallel maps per entity.** Each YAML keeps `status_flow` (the transition
   graph) separate from `status_metadata` (color, phase, weight, responsibility,
   `orchestrator_action`), both keyed by status name. Metadata should live *with*
   the step.
2. **`ready_for_X` + `in_X` doubles every stage.** The pair exists only to let a
   dispatch loop tell "waiting" from "being worked." It bloats every workflow (the
   epic has a full `ready_for_refinement → in_refinement → ready_for_research → …`
   ladder).
3. **Routing is engine-pushed and forward-only.** `shark next` dispatches the
   current step, but nothing lets a *skill report an outcome* that decides the next
   step. Skills cannot stay out of orchestration if they have no way to hand
   routing back to it.

### Solution

One coherent change — an **outcome-routing contract** is the keystone, and the
marker-collapse and file-consolidation fall out of it:

- Skills are **topology-blind**: they do their craft and return a release payload
  `{ outcome, reason, log }`, never naming a status (D1).
- The workflow YAML **owns routing** via a per-step `outcomes:` map that replaces
  `status_flow` entirely (D2).
- **Status is a pure phase; the claim is the lease.** `in_X` becomes the claim;
  `ready_for_X` becomes the bare phase name (D3).
- **`advance` becomes `release(outcome)`** — the engine resolves
  `step.outcomes[outcome]`; `set-status --force` remains the human escape hatch (D4).
- **One block per step** — `status_flow` + `status_metadata` merge; metadata lives
  with the step (D5).
- **Master index + bundle-rooted resolution** — `.sharkconfig.json` points at one
  index file; the shark-data bundle root is the resolution base (D6).
- **Outcome vocabulary = core + extras** — every non-terminal step defines
  `pass`/`fail`/`blocked`; steps may add extras; `validate` enforces the core (D7).

The epic ladder collapses from the marker pairs to
`draft → refinement → research → design → decomposition → feature_review → active`.

### Impact

- Skills become portable across reordered/renamed workflows — they emit semantic
  outcomes, not target statuses.
- Workflow YAML shrinks roughly by half (one block per step, no marker pairs).
- Routing is data-driven and bidirectional (a `fail` can route back to an earlier
  step) without any skill knowing the graph.
- Nothing hardcodes statuses — phases and outcomes both come from YAML, config-driven
  end to end (preserves the no-hardcoded-statuses rule).

---

## Business Value

**Rating**: High

Foundational engine work that removes the largest remaining sources of workflow
friction after E32: the duplicated marker statuses, the two-map split, and the
engine-only forward routing. It makes skills genuinely portable (the prerequisite
for multi-harness support), halves workflow-file size, and enables back-routing
(`fail → development`) that the current model cannot express. It is debt paydown
that compounds every time a new entity workflow or harness is added.

---

## Scope

### In Scope

- Consolidated per-step schema; drop the `status_flow` + `status_metadata` split (F01).
- `outcomes:` routing map + `release(outcome)` advance semantics + service
  resolution + validate (F02).
- Claim/session lease, `shark next` unclaimed-only, TTL backstop, updates folded
  into the E32-F07 telemetry pipeline (F03).
- Master index file + absolute-path / bundle-rooted resolution (F04).
- One-shot status-column migration via per-step `aliases:`, input compat shim,
  history-read resolution (F05).
- Harness dispatch-loop rewrite (`claim → run → release`), `validate` checks, and
  doc/vocabulary updates (F06).

### Out of Scope

- `{{augment:}}` semantics (carried over as a TBD from E32-F02) — unrelated to this
  redesign.
- True remote (URL) bundles — absolute filesystem paths only; no fetch/cache/pin/trust.
- Semantic-rot validation (golden-output diff) — an E32 follow-up; this epic adds
  structural checks only.

---

## Features & Sequence

| Order | Feature | Size | Design refs | Depends on |
|------:|---------|:----:|-------------|------------|
| 1 | **E35-F01** — Step-schema consolidation | M | D5, §4, §8 | — |
| 2 | **E35-F02** — Outcome routing and release | L | D1, D2, D4, D7, §3, §8 | F01 |
| 3 | **E35-F03** — Claim / session lease | L | D3, §6, §8 | F02 |
| 4 | **E35-F04** — Master index and bundle resolution | M | D6, §5, §8 | — (parallel with F02) |
| 5 | **E35-F05** — Status migration and compat shim | M | §7, §8 | F01, F02, F03 |
| 6 | **E35-F06** — Harness and docs | M | §3, §8, §9 | F02, F03, F04 |

Critical path: **F01 → F02 → F03 → F05 → F06**. F04 is independent and can run in
parallel with F02. F05 is the migration capstone; F06 is final integration.

---

## Code Blast Radius

Known surface (current readers of the workflow config); precise line-level inventory
is a per-feature tracing task. See design §8.

- **Schema / loader** — `internal/config/workflow/{schema,yaml_loader,parser,multilevel}.go` (F01, F02, F04).
- **Service** — `internal/workflow/service.go`: transitions derive from `outcomes`;
  outcome-resolution + parking-resume history walk-back (F02, F05).
- **Dispatch + advance** — `internal/cli/commands/next.go` and the
  advance / `next-status` / `set-status` commands (F02, F03).
- **Persistence** — claim fields + one-shot status migration; bump
  `CurrentSchemaVersion` (F03, F05).
- **Validate** — `shark validate` outcome/alias/bundle-root checks (F02, F06).
- **Telemetry** — fold lease updates into the E32-F07 `events.jsonl`/span pipeline (F03).
- **Harness** — `shark/SKILL.md` dispatch loop; `CLAUDE.md`,
  `workflow-configuration.md`, `workflow-profiles.md` (F06).

---

## Success Criteria

| # | Criterion | How it is measured |
|---|-----------|--------------------|
| SC1 | A workflow YAML has one block per step; no `status_flow`/`status_metadata` maps remain. | Loader rejects the two-map form; canonical YAMLs parse with merged steps. |
| SC2 | A skill releases `{outcome, reason, log}` and the engine routes without the skill naming a status. | `shark advance <key> --outcome=fail` resolves `step.outcomes[fail]` and parks the entity at the target phase. |
| SC3 | `shark next` only hands out unclaimed entities; claimed ones are skipped until lease expiry. | Claim an entity, confirm `shark next` skips it; let the lease lapse (K missed updates), confirm it is reclaimable. |
| SC4 | Routing and resolution are config-driven from a single bundle root. | `.sharkconfig.json` → index file; prompts/skills/agents resolve from the bundle root with `overrides/` winning. |
| SC5 | Existing entities migrate without data loss. | One-shot migration rewrites the live `status` column via `aliases:`; `task_history` is untouched and alias-resolves on read; mid-flight `in_X` rows become unclaimed phase rows. |
| SC6 | `shark validate` catches structural defects. | Fails when core outcomes are missing, an outcome target does not exist, an old status has no alias home, or a referenced prompt/skill/agent does not resolve. |

---

*Last Updated*: 2026-06-23
