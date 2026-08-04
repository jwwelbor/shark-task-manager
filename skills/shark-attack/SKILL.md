---
name: shark-attack
description: Chair-led, evidence-based team protocol for Shark work without adding a runtime, workflow engine, or claim store.
---

# Shark Attack

## Purpose

Shark Attack is a durable collaboration protocol for work that benefits from a
chair-led council. It coordinates judgment and evidence; Shark workflow metadata,
the owning dispatch loop, and the existing claim service remain authoritative.
It does not create an AI runtime, provider configuration, credential store,
workflow engine, or second lease store.

## Two axes and coordination routing

Classify coordination level and execution topology independently — see
`context/operating-model.md` for both axes' definitions, the illustrative
examples, and the degradation rule that governs a parallel-topology request
with no matching evidence.

`workflows/coordinate.md` applies the classified coordination level to select
exactly one destination procedure:

| Coordination level | Scenario | Destination |
|---|---|---|
| `Direct` | one bounded entity, clear acceptance, small edit surface | `workflows/direct.md` |
| `Batch` | several related entities needing one scope/conflict pass | `workflows/batch.md` |
| `Council` | cross-entity ambiguity, high blast radius, or specialist disagreement | `workflows/council.md` |

The selected procedure then applies the topology already classified above to
decide whether it dispatches one worker, a Sequential wave, or a parallel wave
through `workflows/execute-wave.md`.

## Parent loop

The parent — the Rider loop or an equivalent dispatching coordinator — holds
all Shark workflow authority for a dispatched entity; a worker never mutates
that state directly. `context/authority.md` is the single source for the
parent/worker/council authority boundary and its consultation and
lease-loss rules.

Rider re-entry (`workflows/pull-by-role.md`'s "Sanctioned path: Rider
re-entry") is the only sanctioned claim path. The worker-owned child mode is
not `/shark-rider run`; it is retired and documented only as a historical /
compatibility reference in `context/worker-ownership.md` — never as a live
claim instruction.

On a live-consultation follow-up or any other worker refresh, the parent
follows `workflows/resume.md`, which selects the active host's follow-up,
interrupt, and isolation capability from `providers/codex.md` or
`providers/claude-code.md` before choosing a resume path.

## Question loop

A dispatched worker that cannot proceed returns a `question` control envelope
(`context/worker-control-schema.yaml`) instead of guessing. A routine
question — scope-bounded, single-role, no cross-entity impact — routes
through the full mint-through-resolve E39 `Q###` loop in
`workflows/route-question.md`. A question that trips `workflows/council.md`'s
material threshold instead routes through that file — never through a second
escalation format or destination.

## Setup and roster

Use `context/roster-schema.yaml` as the canonical roster template; its chair
facilitates a decision, not a new workflow authority, and `model_tier` is an
optional preference that cannot select work, override workflow metadata, or
affect a claim. Follow `workflows/setup.md` to prepare a project's durable
`docs/council/` layout and its replace-only override subtree.

## Links

- Two axes: `context/operating-model.md`
- Coordination routing: `workflows/coordinate.md`, `workflows/direct.md`,
  `workflows/batch.md`, `workflows/council.md`, `workflows/execute-wave.md`
- Authority and roles: `context/authority.md`, `context/worker-ownership.md`
- Role-aware selection and Rider re-entry: `workflows/pull-by-role.md`
- Questions and consultation: `workflows/route-question.md`,
  `context/worker-control-schema.yaml`
- Resume and providers: `workflows/resume.md`, `providers/codex.md`,
  `providers/claude-code.md`
- Council artifacts: `context/message-schema.md`
- Setup and roster: `workflows/setup.md`, `context/roster-schema.yaml`
