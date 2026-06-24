# Route-Based Workflow Redesign (Shark 2.x)

**Status:** Design — agreed, not yet broken into shark entities
**Date:** 2026-06-22
**Builds on:** E32 (single-artifact consolidation — YAML workflows, `shark next`, embedded FS, `{{include:}}`)
**Supersedes:** the `ready_for_X`/`in_X` status model and the `status_flow` + `status_metadata` two-map split

---

## 1. Why

E32 moved workflows from one JSON file to per-entity YAML, but three frictions remain:

1. **Two parallel maps per entity.** Each YAML still keeps `status_flow` (the transition graph) *separate from* `status_metadata` (color, phase, weight, responsibility, `orchestrator_action`), both keyed by status name. Metadata should live *with* the step.
2. **`ready_for_X` + `in_X` doubles every stage.** The pair only exists to let a dispatch loop tell "waiting" from "being worked." It bloats the workflow (epic has a full `ready_for_refinement → in_refinement → ready_for_research → in_research → …` ladder).
3. **Routing is engine-pushed and forward-only.** `shark next` dispatches the current step, but nothing lets a *skill report an outcome* that decides the next step. Skills can't stay out of orchestration if they have no way to hand routing back to it.

The redesign is one coherent change, not three: an **outcome-routing contract** is the keystone, and the marker-collapse and file-consolidation fall out of it.

---

## 2. Decisions (locked)

| # | Decision |
|---|----------|
| D1 | **Skills are topology-blind.** A skill does its craft, then returns a release payload `{ outcome, reason, log-data }`. It never names a status. |
| D2 | **The workflow YAML owns routing.** Each step has an `outcomes:` map (`pass: approval`, `fail: development`, …). This *replaces* `status_flow` entirely. |
| D3 | **Status is a pure phase; the claim is the lease.** An agent *claims* an entity (session = who + when); `shark next` only hands out *unclaimed* entities. `in_X` → the claim. `ready_for_X` → the bare phase name. |
| D4 | **`advance` becomes `release(outcome)`.** The caller never picks a target; the engine resolves `step.outcomes[outcome]`. `set-status --force` remains the human escape hatch. |
| D5 | **One block per step.** `status_flow` + `status_metadata` merge into a single per-step block; `instruction_template` → `prompt`; terminal steps flagged; `start:` at the top. |
| D6 | **Master index, bundle-rooted resolution.** `.sharkconfig.json` → one index file (absolute path allowed). The shark-data bundle root is the resolution base; prompts/skills/agents resolve relative to it; local `overrides/` layer on top. |
| D7 | **Outcome vocabulary = core + extras.** Core `pass`/`fail`/`blocked` every non-terminal step must define; steps may add extras (`dead-end`, …). Engine routing is a map lookup; `validate` enforces the core exists. |

Consistency check: nothing here hardcodes statuses. Phases and outcomes both come from YAML — config-driven end to end.

---

## 3. The contract

A skill claims an entity, does its work knowing nothing about the graph, then **releases** a payload:

```
ROUTE: fail
REASON: 3 acceptance criteria unmet
LOG:    { ...anything the skill wants recorded... }
```

The engine looks up `step.outcomes[fail]` and advances. Skills are portable because they emit *semantic outcomes*, not target statuses — reordering or renaming steps never touches a skill.

- **Status** = the phase the entity is parked at (`qa`, `research`).
- **Claim** = the in-flight lease (session id + timestamp), orthogonal to status. Prevents double-grab.
- **Release** = the moment the outcome comes back; the engine resolves the target and advances atomically. `shark advance` is effectively `release(outcome)`.

This single contract dissolves both markers: `in_X` becomes the claim, `ready_for_X` becomes the phase name. The epic ladder collapses to `draft → refinement → research → design → decomposition → feature_review → active`.

---

## 4. Consolidated step schema (D5)

One block per step. `status_flow` and `status_metadata` both gone — `outcomes` replaces the first, the merge absorbs the second.

```yaml
# workflow/feature.yaml
version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    progress_weight: 0.0
    action: advance_status          # auto, no agent
    outcomes: { pass: refinement }

  qa:                               # was ready_for_qa + in_qa
    phase: qa
    color: cyan
    progress_weight: 0.85
    responsibility: agent
    action: spawn_agent
    agent: qa
    provider: anthropic
    model: sonnet
    skills: [quality]
    prompt: feature/qa.md           # was instruction_template
    aliases: [ready_for_qa, in_qa]  # migration + compat (see §7)
    outcomes:
      pass: approval
      fail: development             # route back
      blocked: on_hold

  blocked:
    phase: blocked
    parking: true                   # resume target computed from history, not a static outcome
  on_hold:
    phase: paused
    parking: true

  completed:
    phase: done
    progress_weight: 1.0
    terminal: true                  # no outcomes; was special_statuses._complete_
```

---

## 5. Master index & resolution (D6)

```jsonc
// .sharkconfig.json
{ "workflow_config": "shark-data/workflow.yaml" }   // absolute path also allowed
```

```yaml
# shark-data/workflow.yaml  (the index)
entities:
  task:      workflow/task.short.yaml   # mix profiles per entity
  feature:   workflow/feature.yaml
  epic:      workflow/epic.yaml
  bug:       workflow/bug.yaml
  change:    workflow/change.yaml
  tech-debt: workflow/tech-debt.yaml
```

- The **shark-data bundle root is the resolution base.** Relative paths resolve against the index file's location; absolute paths point anywhere on the filesystem (this is all "remote" means — outside the project folder; no URL fetch, no cache, no pinning).
- The unit of sharing is the **whole bundle** (workflow + prompts + skills + agents). A project consumes a shared bundle by pointing at it and customizes only through its local `overrides/`. Everything the workflow references is guaranteed to resolve because it resolves from the same root.

---

## 6. Claim / release / crash recovery

One lease primitive serves both dispatch modes:

- **Sync dispatch** (loop spawns agent → blocks → releases): the orchestrator releases on *every* exit — success → `outcome`, crash/timeout → `release(blocked, reason="agent died")`. The TTL almost never fires.
- **Parallel / detached dispatch**: the agent (or its runner) emits periodic **updates** that renew the lease *and* carry progress. Miss too many → `shark next` reclaims and requeues. The TTL is the primary safety here.
- **TTL is the universal backstop** — neither mode can wedge permanently, even if the orchestrator itself dies. Express it as **K missed updates**, not a per-step absolute (one global cadence, no per-step timeout guessing).

**Updates do triple duty** — lease renewal + progress reporting + telemetry. Route them through the same `events.jsonl`/span pipeline that **E32-F07** adds; do not build a separate heartbeat channel. The loop's "is it alive / how far along" question and F07's "measure efficiency" question are the same data.

---

## 7. Resolved TBDs

| TBD | Resolution |
|-----|------------|
| **Parking resume** | Derive the return target from `task_history` (most recent `from_status` that isn't itself a parking state — walk back past chained parks). No stored field; history is the single source of truth. Parking steps (`parking: true`) are the one case whose target is *computed*, not a static `outcomes` entry. Because steps are atomic, "resume" = re-enter the phase and re-claim; there is no mid-step state to reconstruct. |
| **Crash recovery** | See §6 — one lease, sync-releases / detached-heartbeats, TTL = K missed updates, fed through F07. |
| **Remote bundles** | "Remote" = absolute filesystem paths (shared mount, monorepo, git submodule). The path model already covers it. No fetch/cache/pin/trust machinery. |
| **Status migration** | `aliases:` on each new step list the old names that collapse into it (per-entity, since steps are per-entity). The alias map does triple duty: (1) **migration source** — the one-shot DB migration rewrites the live `status` column; (2) **input compat shim** — the loader accepts an old status name during the deprecation window so hooks/scripts/muscle-memory don't break at cutover; (3) **history-read resolution** — for an entity parked before migration and resumed after. **Rewrite the live `status` column once; leave `task_history` untouched** (audit trails record what actually happened — alias-resolve old names on the rare read instead). Mid-flight rows (`in_X`) become *unclaimed* phase rows and get safely re-dispatched, because steps are atomic. |

### Status mapping (epic, illustrative — rule generalizes: strip `ready_for_`/`in_`, collapse the pair)

| Old | New |
|-----|-----|
| `draft` | `draft` |
| `ready_for_refinement`, `in_refinement` | `refinement` |
| `ready_for_research`, `in_research` | `research` |
| `ready_for_design`, `in_design` | `design` |
| `ready_for_decomposition`, `in_decomposition` | `decomposition` |
| `ready_for_feature_review`, `in_feature_review` | `feature_review` |
| `active`, `completed`, `cancelled`, `blocked`, `on_hold` | unchanged |

---

## 8. Code blast radius

The precise, line-level inventory is a tracing task; this is the known surface (current readers of the workflow config):

- **Schema / loader** — `internal/config/workflow/{schema,yaml_loader,parser,multilevel}.go`: new per-step `Step` struct with `outcomes`/`aliases`/`parking`/`terminal`; drop the `status_flow` + `status_metadata` split; add index-file loading + absolute-path resolution rooted at the bundle.
- **Service** — `internal/workflow/service.go`: `ValidateTransition` / `GetValidTransitions` derive from `outcomes` instead of `status_flow`; add outcome-resolution + parking-resume (history walk-back).
- **Dispatch + advance** — `internal/cli/commands/next.go` (reads the merged step, unchanged wire shape where possible) and the advance / `next-status` / `set-status` commands (release-with-outcome semantics; `set-status --force` retained).
- **Persistence** — claim fields (`claimed_by`/`session`/`claimed_at`, or reuse the existing sessions table) + the one-shot status-value migration; bump `CurrentSchemaVersion`.
- **Validate** — `shark validate`: outcome maps resolve, core outcomes present, every old status has an alias home, prompts/skills/agents resolve from the bundle root.
- **Telemetry** — fold lease updates into the E32-F07 `events.jsonl`/span pipeline.
- **Harness** — `shark/SKILL.md` dispatch loop becomes `claim → run → release`; docs (`CLAUDE.md`, `workflow-configuration.md`, `workflow-profiles.md`) update vocabulary.

---

## 9. Suggested decomposition (when broken into shark)

Rough feature shape — keys assigned by shark on creation:

1. **Step-schema consolidation** — merged per-step block; loader/schema; drop the two-map split (D5).
2. **Outcome routing + release** — `outcomes` map; `release(outcome)` advance; service resolution; validate (D1, D2, D4, D7).
3. **Claim / session lease** — claim fields, `shark next` unclaimed-only, TTL backstop, updates via F07 (D3, §6).
4. **Master index + bundle resolution** — index file, absolute paths, bundle-rooted resolution (D6).
5. **Migration + compat** — `aliases:`, status-column rewrite, input shim, history-read resolution (§7).
6. **Harness + docs** — dispatch loop rewrite, vocabulary updates.

---

## 10. Out of scope / open

- `{{augment:}}` semantics (carried over as TBD from E32-F02) — unrelated to this redesign.
- True remote (URL) bundles — explicitly *not* in scope; absolute filesystem paths only.
- Semantic-rot validation (golden-output diff) — already an E32 follow-up; this redesign adds structural checks only.
