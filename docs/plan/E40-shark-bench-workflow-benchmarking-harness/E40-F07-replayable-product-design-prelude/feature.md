---
feature_key: E40-F07-replayable-product-design-prelude
epic_key: E40
title: Replayable product-design prelude
description: Run the real D01 through D05 Shark Rider product-design prelude for feature scenarios from versioned stakeholder and research responses. Route human questions and research through a replay adapter, block live input during scored runs, record response lineage, and preserve explicit non-applicable stages for other entity families. Consumes I-04 and X-10. Produces I-06.
---

# Replayable product-design prelude

**Feature Key**: E40-F07 · **Size**: M · **Execution order**: 7

## Outcome

A feature-delivery scenario reproducibly executes the real D01-D05 Shark Rider
product-design path without depending on a live stakeholder, mutable web result,
or unrecorded operator decision.

## Scope

- Wrap the existing Shark Rider product-design action; do not copy or fork its
  D01-D05 methodology.
- Define a versioned replay bundle for scripted stakeholder answers, interview
  or proxy-research evidence, and frozen research-tool responses.
- Route human questions and research requests through the replay adapter during
  scored runs. Disable live network research and unrecorded human input.
- Record which response each stage consumed and connect every generated D01-D05
  artifact to its input, response, prompt, and output digests.
- Stop with `unresolved_gate` when the bundle lacks an authorized answer.

## Acceptance boundary

- Two runs with the same scenario and replay bundle consume the same response
  sequence and produce complete D01-D05 lineage.
- A scored run cannot reach live research or an interactive question surface.
- Missing replay input stops the scenario and never invents an answer.
- Non-feature scenarios bypass the prelude and retain explicit non-applicable
  stage records.

## Contracts

- **Consumes I-04**: use the feature scenario's stage matrix and replay
  references.
- **Consumes X-10**: invoke the existing E36-F02 product-design action and
  progress-record contract rather than defining a second product-design flow.
- **Produces I-06 — Product-design replay result**: D01-D05 artifact references,
  response lineage, digests, and terminal prelude outcome consumed by E40-F08.
  The authoritative shape lives in
  [architecture.md](../architecture.md#product-design-replay-contract).

## Out of scope

- D06-D14 product-design artifacts.
- New product-design methodology or production Rider behavior.
- The keyed Shark entity lifecycle after the prelude completes.

## Workflow handoff

The feature workflow must verify the live Rider adapter and bundled methodology
before specifying the replay seam. Any required generic Rider change must be
triaged under its owning epic and linked; benchmark-only replay remains here.

*Last updated: 2026-08-11*
