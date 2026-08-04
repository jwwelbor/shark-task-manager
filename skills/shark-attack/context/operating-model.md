# Coordination-level and execution-topology selection model

Two axes select how work runs. They are chosen **independently**: changing
one axis in a fixture never changes the other. `Direct` is the default
coordination level; `Sequential` is the default execution topology.

## Axis 1 — coordination level

- **`Direct`** — one entity is bounded, acceptance is clear, the edit
  surface is small, and no material cross-entity decision is expected. No
  standing council, inbox, or decision artifact is required. This is the
  default level.
- **`Batch`** — related entities benefit from one scope and conflict
  analysis but do not need sustained specialist adjudication. Research runs
  in parallel where read-only; implementation runs in conflict-aware waves;
  one consolidated observation record is created.
- **`Council`** — use when any of these is true: architecture or product
  intent is materially unclear; two specialists disagree; a cross-feature or
  cross-epic contract is missing or inconsistent; implementation crosses
  ownership boundaries or has high blast radius; security, data migration,
  cost, or irreversible behavior is at risk; acceptance requires independent
  technical and product judgment; a worker cannot proceed safely from
  project evidence.

## Axis 2 — execution topology

- **`Sequential`** — the default topology. Use for overlapping files,
  dependent edits, one shared database/environment, producer-before-consumer
  contracts, or hosts without safe isolation.
- **`Parallel with ownership`** — use only when the chair has recorded
  exclusive file/component ownership, workers have disjoint write sets,
  shared contracts are frozen, and the parent can integrate without
  speculative conflict resolution. Ownership evidence authorizes this
  topology only — it does not authorize `Parallel with isolation`.
- **`Parallel with isolation`** — use separate worktrees or equivalent
  isolated sessions when write sets may overlap but work can be integrated
  as explicit commits/tranches. The parent controls base revision,
  integration order, conflict resolution, and the post-integration gate.
  Isolation evidence authorizes this topology only — it does not authorize
  `Parallel with ownership`.

## Degradation rule

`Sequential` is the default topology. A parallel topology (`Parallel with
ownership` or `Parallel with isolation`) requires captured, recorded
evidence matching that specific topology. Whenever the required evidence
cannot be produced, the request **MUST degrade to `Sequential`** —
regardless of the requested coordination level. Ownership evidence alone
does not authorize `Parallel with isolation`, and isolation evidence alone
does not authorize `Parallel with ownership`; each parallel topology needs
its own matching evidence.

**Isolation does not make logically dependent work parallel.** Isolation
evidence never authorizes running logically dependent (producer/consumer
contract-ordered) work in parallel. Regardless of which topology is
selected, producer/consumer contract order still applies.

## Illustrative examples

| Work | Coordination | Topology | Why |
|---|---|---|---|
| One localized task with clear tests | Direct | Sequential | One worker is cheaper and clearer. |
| Seven related tech-debt items | Batch | Parallel research, then Sequential/ownership waves | Shared files and gates limit safe mutation despite easy parallel discovery. |
| Cross-cutting feature with API, storage, CLI, and UAT | Council | Mixed: research parallel; isolated/disjoint implementation waves; controlled integration | Requirements, contracts, review, and acceptance require live specialist routing. |

## Decision table — two-axis independence

The coordination-level column and the resolved-topology column classify
independently of each other. `Direct` classifies with both `Sequential`
(row 1) and `Parallel with ownership` (row 9) — coordination level does
not itself forbid a parallel-topology classification; the axes are
independent in **both** directions, not just in the `Batch`/`Council` →
parallel direction. That independence is a classification-time property
only: only `Batch` and `Council` dispatch procedures (`batch.md`,
`council.md`, via `execute-wave.md`) actually consult the resolved
topology to shape a wave. `Direct`'s own dispatch procedure (`direct.md`)
never consults it — a single bounded entity has no wave to shape, so
`direct.md` always dispatches exactly one worker regardless of which
topology row 9 classifies.

| # | Requested coordination | Requested topology | Ownership evidence recorded? | Isolation evidence recorded? | Resolved topology |
|---|---|---|---|---|---|
| 1 | Direct | Sequential | n/a | n/a | Sequential |
| 2 | Batch | Parallel with ownership | yes | no | Parallel with ownership |
| 3 | Batch | Parallel with ownership | no | no | Sequential (degraded — missing ownership evidence) |
| 4 | Batch | Parallel with isolation | no | yes | Parallel with isolation |
| 5 | Batch | Parallel with isolation | no | no | Sequential (degraded — missing isolation evidence) |
| 6 | Batch | Parallel with isolation | yes | yes | Parallel with isolation (isolation still required — ownership alone is insufficient for an isolation-requesting row) |
| 7 | Council | Sequential | n/a | n/a | Sequential |
| 8 | Council | Parallel with ownership | yes | no | Parallel with ownership |
| 9 | Direct | Parallel with ownership | yes | no | Parallel with ownership (classification only — proves the axes are independent in both directions; `direct.md` still dispatches exactly one worker, since a single bounded entity has no wave to shape) |

Rows 3 and 5 are the two independent degrade-to-`Sequential` paths: row 3
degrades because ownership evidence is missing, row 5 degrades because
isolation evidence is missing. A fix that repairs one path can silently
leave the other broken, so both are asserted separately. Row 6 shows that
ownership evidence alone is insufficient when the requested topology is
`Parallel with isolation` — the evidence must match the requested topology,
not merely exist.

## Runtime consequence of no recorded evidence

Keyed dispatch (documented in the route-based workflow guide) never
dispatches more than one entity from a single fork call: a fork's
candidate listing is a read-only response with no worker prompt, and each
candidate requires its own separate keyed call to receive a dispatch
step. With no ownership or isolation evidence recorded, nothing in that
mechanism produces concurrent dispatch — a parallel-topology request
resolves the same way a `Sequential` request would: one entity dispatched
per call, in whatever order the chair issues those calls.
