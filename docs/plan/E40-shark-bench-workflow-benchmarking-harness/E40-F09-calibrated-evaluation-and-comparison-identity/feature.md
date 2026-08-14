---
feature_key: E40-F09-calibrated-evaluation-and-comparison-identity
epic_key: E40
title: Calibrated evaluation and comparison identity
description: Evaluate applicable stage artifacts, structured review findings, review-gate policies, and final implementations with deterministic checks, held-back execution oracles, and calibrated judgment. Pin exact candidate and policy identity, reject mixed or incomplete aggregates, and retain invalid-run reasons. Consumes I-05, I-07, and X-12. Produces I-08.
---

# Calibrated evaluation and comparison identity

**Feature Key**: E40-F09 · **Size**: M · **Execution order**: 9

## Outcome

Maintainers can distinguish workflow completion from artifact quality and
implementation correctness, and can compare runs only when every input that
could change the result has the same recorded identity.

## Scope

- Define deterministic structural checks for required artifacts, ownership,
  links, dependencies, status transitions, traceability, and executable tasks.
- Run held-back execution oracles after implementation without exposing them to
  workers.
- Define a versioned LLM-judge rubric for applicable planning and decomposition
  artifacts and calibrate it against a small human-scored set.
- Record the judge model and configuration, rubric and prompt digests, reference
  digests, rationale, score, usage, and cost.
- Require scenario, fixture, adapter, Shark binary, installed content, rendered
  prompt, provider/model/effort, judge, reference, and resource-policy identity.
- Require exact candidate identity and workflow-policy identity for every
  review comparison. A matching branch or `HEAD` is insufficient.
- Preserve reviewer-provided finding fields, then assign a separate normalized
  identity, confirmation source, first-seen gate, duplicate or recurrence link,
  resolution candidate, and final disposition. Never treat the reviewer's own
  fingerprint or severity as adjudicated truth.
- Define an independent comparison in which feature QA and finish-feature
  deep review inspect the same frozen candidate with no fixes between them, and
  a sequential comparison in which the real gate order and intervening fixes
  remain visible.
- Use the repository-owned deep-review bundle and pin one digest over its skill,
  six angle prompts, consolidator prompt, and diff-selection script. Treat a
  bundle change as workflow-policy drift.
- With seeded defects and clean controls, measure confirmed precision and recall
  by defect class. Without a truth set, report unique confirmed yield, overlap,
  recurrence, downstream escapes, and unconfirmed findings without inventing
  precision or recall.
- Derive first-pass yield, review and rework cost, artifact utilization, and
  quality-time-cost tradeoffs without collapsing them into one composite score.
- Reject mixed or incomplete aggregates and retain every invalid run and reason.

## Acceptance boundary

- A terminal Shark status or worker pass cannot substitute for structural or
  execution-oracle evidence.
- The judge cannot gate publication until calibration evidence exists and is
  separate from the held-out evaluation set.
- Changing any required identity field prevents the runs from entering the same
  aggregate.
- Changing the candidate tree, diff, untracked manifest, test suite, enabled
  gates, gate order, reviewer configuration, or fix policy invalidates a review
  comparison.
- An independent QA-versus-deep-review result uses one frozen candidate; a
  sequential result identifies each intervening candidate and attributes only
  newly confirmed findings to the later gate.
- Evaluation can be replayed from retained I-05 and I-07 artifacts without
  rerunning the scenario.

## Contracts

- **Consumes I-05**: immutable stage snapshots and evaluator access boundary.
- **Consumes I-07**: complete lifecycle execution and validity record.
- **Consumes X-12**: use the E32-F04 canonical Shark-data bundle as the source
  for installed workflow, prompt, skill, and agent content identity.
- **Produces I-08 — Lifecycle evaluation record**: structural results, judge
  evidence, execution-oracle result, comparison identity, aggregate eligibility,
  and invalidity reasons consumed by E40-F10. The authoritative shape lives in
  [architecture.md](../architecture.md#lifecycle-evaluation-record-contract).

## Out of scope

- Tuning the rubric against the held-out evaluation set.
- Treating an LLM judge as a replacement for executable truth.
- Running provider-backed batches or publishing operator reports.
- Treating observational stage order as causal evidence of gate value without a
  paired frozen-candidate or controlled policy comparison.

## Workflow handoff

The feature workflow must specify deterministic checks and calibration evidence
before it specifies aggregate scoring. It must preserve raw disagreements for
later rubric refinement.

## 2026-08-13 amendment: review-gate value

I-08 now owns the distinction between emitted findings and confirmed value. It
defines the fair comparison boundary for feature QA and finish-feature deep
review; E40-F10 owns provider-backed execution and reporting of that comparison.

*Last updated: 2026-08-13*
