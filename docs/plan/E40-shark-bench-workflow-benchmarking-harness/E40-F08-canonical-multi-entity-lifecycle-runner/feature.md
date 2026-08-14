---
feature_key: E40-F08-canonical-multi-entity-lifecycle-runner
epic_key: E40
title: Canonical multi-entity lifecycle runner
description: Drive each scenario root and every eligible descendant through the canonical keyed Shark lifecycle. Preserve dispatch, lease, prompt, outcome, stage-time, candidate, artifact-use, workflow-policy, and structured review-finding evidence; route Questions; and stop with named resource or human-gate outcomes. Consumes I-04, I-05, I-06, X-11, and X-13. Produces I-07.
---

# Canonical multi-entity lifecycle runner

**Feature Key**: E40-F08 · **Size**: M · **Execution order**: 8

## Outcome

The benchmark drives a scenario root and every eligible descendant through the
real Shark lifecycle with deterministic scheduling, bounded resource use, and a
durable record of every dispatch, lease, outcome, transition, and stop reason.

## Scope

- Call the keyed dispatch API for the current root and preserve each returned
  response before selecting work.
- Resolve hierarchy forks through a recorded deterministic policy and execute
  every eligible generated task unless a scenario-level safety ceiling stops
  the entire run.
- Claim the concrete returned entity, pass the rendered prompt unchanged to a
  host worker, heartbeat the lease, persist the semantic outcome, apply the
  configured transition, and release on every exit path.
- Route Questions and human decisions only through authorized scenario replay.
- Record `pause`, `archive`, `error`, lease loss, missing outcome,
  `unresolved_gate`, and `resource_limit` as named outcomes with partial evidence.
- Enforce positive cost, wall-time, and generated-task ceilings for every
  provider-backed scenario.
- Populate I-05 stage categories, time intervals, candidate snapshots, and
  artifact consumption events from observed dispatch and tool activity.
- Record workflow-policy identity: enabled gates, gate order, reviewer provider,
  model, effort, prompt digest, full review-bundle digest, and whether fixes are
  allowed between gates.
- Capture every `review-finding` note with its raw gate, round, severity, defect
  class, fingerprint, affected criterion or test, and disposition. Record an
  explicit zero-finding gate result; do not infer zero from an absent collector.
- Link each finding to the candidate it reviewed and to the later candidate that
  claims to resolve it, while preserving the original note unchanged for F09.

## Acceptance boundary

- Tests cover fork scheduling, claim, heartbeat, exact prompt handoff, outcome
  persistence, transition, and release on success and every failure path.
- Every eligible generated task executes or has a durable ineligibility reason.
- Reaching a safety ceiling invalidates and stops the whole scenario; the runner
  never publishes a silently truncated plan.
- Workers cannot claim, advance, or release Shark entities themselves.
- A gate with no findings, a gate whose finding collector failed, and a gate
  that was not reached produce three different records.
- Every review finding references the exact candidate snapshot and workflow
  policy active when the reviewer emitted it.

## Contracts

- **Consumes I-04**: scenario roots, family, applicable stages, adapter, and
  resource policy.
- **Consumes I-05**: write the required snapshot after each applicable stage.
- **Consumes I-06**: begin feature scenarios from the recorded D01-D05 prelude.
- **Consumes X-11**: preserve the E38-F07/E38-F09 Rider execution contract for
  keyed dispatch, leases, prompt provenance, semantic outcomes, and resume.
- **Consumes X-13**: use the E39-F04 Question lifecycle as the durable human-gate
  surface.
- **Produces I-07 — Lifecycle run record**: entity graph, dispatch and evidence
  references, claims, transitions, outcomes, usage, cost, elapsed time, limits,
  and final validity consumed by E40-F09 and E40-F10. The authoritative shape
  lives in [architecture.md](../architecture.md#lifecycle-run-record-contract).

## Out of scope

- Reconstructing prompts or workflow routing outside Shark.
- Adding a second claim store, workflow engine, or status model.
- Deciding whether generated artifacts are correct.

## Workflow handoff

The feature workflow must treat the host loop as a benchmark adapter over
public Shark contracts, not as a fork of `shark run`. Production defects found
in those contracts must be triaged under E38, E39, E22, or the owning core epic.

## 2026-08-13 amendment: runtime value evidence

I-07 now carries the observed events needed to measure review and coordination
cost. F08 captures evidence but does not confirm findings, score gates, or claim
that a later passing oracle proves every review finding was valid.

*Last updated: 2026-08-13*
