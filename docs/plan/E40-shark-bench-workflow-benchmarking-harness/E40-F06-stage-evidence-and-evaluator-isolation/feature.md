---
feature_key: E40-F06-stage-evidence-and-evaluator-isolation
epic_key: E40
title: Stage evidence and evaluator isolation
description: Define the three-root isolation model and the durable evidence emitted after every applicable lifecycle stage. Capture prompt and artifact lineage, decomposed time, exact candidate identity, artifact producer and consumer records, admission checks, and the held-back execution-oracle boundary without exposing evaluator-only material to workers. Consumes I-04 and X-09. Produces I-05.
---

# Stage evidence and evaluator isolation

**Feature Key**: E40-F06 · **Size**: M · **Execution order**: 6

## Outcome

Every applicable lifecycle stage leaves enough durable evidence to reproduce
and evaluate it, while reference artifacts, answer keys, patches, and hidden
tests remain unavailable to workers until their authorized evaluation phase.

## Scope

- Define the agent-visible fixture root, scratch Shark project root, and
  evaluator-only root, including which artifacts may cross each boundary.
- Define a stage snapshot with scenario and entity identity, rendered-prompt
  digest, input and replay lineage, output artifact paths and digests, tokens,
  cost, elapsed time, errors, and rework count.
- Classify each stage as discovery, specification, planning, code, review, QA,
  UAT, or shipping. Record a non-overlapping time ledger for provider-active,
  tool and test, queue or claim wait, replay or human-gate wait, retry or
  backoff, and unclassified intervals.
- For every code-producing or review stage, record the base commit, candidate
  tree and binary-diff digests, changed-path digest, dirty and untracked
  manifest, and test-suite digest. Do not treat `HEAD` alone as candidate
  identity.
- Record each generated artifact's type, path, digest, size, producer stage,
  and downstream consumption or access events. Preserve an explicit empty
  consumer set so an orphan artifact is distinguishable from missing telemetry.
- Add admission and dispatch-boundary checks that fail when evaluator-only
  material appears in either agent-visible root.
- Permit evaluator-only material to become readable only by a post-stage or
  post-run evaluator, and record that access in the evidence bundle.
- Preserve partial evidence for named stop outcomes without treating it as a
  valid baseline contribution.

## Acceptance boundary

- Every applicable stage produces exactly one addressable snapshot or a named
  missing-stage failure.
- A dispatch fails before provider spend if any evaluator-only file is visible
  to the worker.
- The post-run execution oracle can read its hidden inputs without copying them
  into the worker checkout before execution completes.
- The same captured evidence can be evaluated again without rerunning the
  worker or calling a provider.
- The stage-time ledger reconciles to stage wall time without double counting,
  and an unknown interval remains explicit rather than being assigned to model
  work.
- Replaying a candidate snapshot detects a changed tracked file, changed
  untracked file, changed test suite, or missing artifact-consumption record.

## Contracts

- **Consumes I-04**: use the scenario package to determine applicable stages,
  identities, roots, and referenced inputs.
- **Produces I-05 — Stage evidence bundle**: the immutable snapshot and
  isolation contract written by E40-F08 and consumed by E40-F09 and E40-F10.
  The authoritative shape lives in
  [architecture.md](../architecture.md#stage-evidence-and-isolation-contract).
- **Consumes X-09** when available: reuse the audited provider-usage field
  mapping from E27-F15; fail closed on missing comparison identity instead of
  guessing provider metadata.

## Out of scope

- Scheduling or dispatching lifecycle work.
- Scoring artifact quality.
- Designing report layouts or baseline commands.

## Workflow handoff

The feature workflow must specify the evidence schema, filesystem boundaries,
and negative isolation tests together. It must not accept a schema that records
only final workflow status or worker self-report.

## 2026-08-13 amendment: value-attribution evidence

I-05 now supplies the raw evidence required to distinguish coding work from
coordination cost and artifact production from artifact use. E40-F08 populates
the runtime records; E40-F09 validates comparison identity; E40-F10 derives
reports. F06 does not decide whether a gate or artifact was valuable.

*Last updated: 2026-08-13*
