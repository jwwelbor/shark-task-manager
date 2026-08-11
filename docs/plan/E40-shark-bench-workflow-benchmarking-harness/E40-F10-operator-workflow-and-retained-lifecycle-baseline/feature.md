---
feature_key: E40-F10-operator-workflow-and-retained-lifecycle-baseline
epic_key: E40
title: Operator workflow and retained lifecycle baseline
description: Provide no-spend dry runs, explicit provider-spend and resource gates, retained pilots, lifecycle and stage reports, machine-readable aggregates, noise bands, and publication checks. Require one inspected provider-backed pilot per scenario family before repeated baseline collection. Consumes I-07 and I-08.
---

# Operator workflow and retained lifecycle baseline

**Feature Key**: E40-F10 · **Size**: M · **Execution order**: 10

## Outcome

An operator can preview, pilot, run, inspect, and report a lifecycle benchmark
without accidental provider spend, and can publish a baseline only when every
selected scenario has complete, comparable, oracle-backed evidence.

## Scope

- Add a no-provider dry run that shows the scenario matrix, applicable stages,
  planned provider calls, output root, and resource ceilings.
- Require explicit provider-spend acknowledgement and positive cost, wall-time,
  and generated-task ceilings for pilot and baseline commands.
- Retain raw scenario packages, stage evidence, run records, evaluation records,
  transcripts, entity history, and oracle results under an explicit output root.
- Publish a lifecycle headline view and a stage diagnostic view from the same
  retained records. Keep stage diagnostics subordinate to the lifecycle result.
- Produce machine-readable aggregates, human-readable reports, invalid-run
  inventories, and scenario/stage noise bands.
- Require one inspected provider-backed pilot per scenario family before any
  repeated baseline batch.

## Acceptance boundary

- Dry-run and report commands make no provider calls.
- Provider-backed commands refuse to start without explicit spend acknowledgement
  and positive resource ceilings.
- The report rejects scenarios with incomplete applicable-stage lineage,
  incompatible identity, missing structural results, missing judge calibration,
  or missing execution-oracle results.
- A retained pilot exposes enough raw evidence to diagnose a completed workflow
  whose held-back execution oracle failed.

## Contracts

- **Consumes I-07**: lifecycle execution, resource, and validity records.
- **Consumes I-08**: evaluation, comparison identity, and aggregate-eligibility
  verdicts.
- Existing E40-F02/F03 run and report surfaces remain low-cost regression and
  compatibility inputs; they are not silently relabeled as the lifecycle v2
  baseline.

## Out of scope

- A hosted dashboard, scheduled service, or CI-triggered provider spend.
- Publishing aggregates from truncated, incompatible, or incompletely evaluated
  scenarios.
- Expanding the corpus beyond the admitted v2 scenario families.

## Workflow handoff

The feature workflow must specify the operator commands, refusal behavior,
retention layout, pilot inspection checklist, and publication gate before it
generates implementation tasks.

*Last updated: 2026-08-11*
