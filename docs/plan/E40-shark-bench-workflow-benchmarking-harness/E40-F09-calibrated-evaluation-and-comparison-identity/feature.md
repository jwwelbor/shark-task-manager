---
feature_key: E40-F09-calibrated-evaluation-and-comparison-identity
epic_key: E40
title: Calibrated evaluation and comparison identity
description: Evaluate applicable stage artifacts and final implementations with deterministic structural checks, held-back execution oracles, and a versioned LLM judge calibrated against human-scored examples. Pin complete comparison identity, reject mixed or incomplete aggregates, and retain invalid-run reasons. Consumes I-05, I-07, and X-12. Produces I-08.
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
- Reject mixed or incomplete aggregates and retain every invalid run and reason.

## Acceptance boundary

- A terminal Shark status or worker pass cannot substitute for structural or
  execution-oracle evidence.
- The judge cannot gate publication until calibration evidence exists and is
  separate from the held-out evaluation set.
- Changing any required identity field prevents the runs from entering the same
  aggregate.
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

## Workflow handoff

The feature workflow must specify deterministic checks and calibration evidence
before it specifies aggregate scoring. It must preserve raw disagreements for
later rubric refinement.

*Last updated: 2026-08-11*
