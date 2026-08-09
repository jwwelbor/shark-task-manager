---
feature_key: E40-F03-baseline-report-and-noise-band
epic_key: E40
title: Baseline report and noise band
description: Run the Phase 1 matrix (~10 tasks x default config x 3 reps), aggregate JSONL artifacts, and produce the baseline report: pass rate, rejection rate per gate, tokens/cost per step, wall-clock, LOC — with run-to-run spread per metric. The published noise band is the deliverable: it defines the threshold any future config delta must clear in paired per-task comparison. Also owns G7/UAT-07 reproducibility: given a stored manifest, re-invokes E40-F02's single-run command with that manifest's pinned fixture SHA, model IDs, and variant bundle, and verifies the replayed run's metrics fall within the published noise band. Consumes: I-02.
---

# Baseline report and noise band

**Feature Key**: E40-F03

See [shark-bench-design.md](../shark-bench-design.md) §5 (comparison method).

---

## Goal

Turn 30 run artifacts (~10 tasks × default workflow config × 3 reps) into the Phase 1 deliverable: a baseline report whose central output is the **noise band** — per-metric run-to-run spread on an unchanged config. Every future config comparison (Phase 2) is judged against this band; deltas inside it are "no detectable effect."

F03 also owns **G7/UAT-07 (reproducibility)**, the Phase 1 exit criterion that a stored manifest's result can be reproduced. Trigger: an operator invokes the replay command against a stored artifact's manifest block. Execution path: F03 reads the manifest (pinned fixture SHA, pinned exact model IDs, variant bundle id) from the stored JSONL record, re-invokes E40-F02's single-run command with those exact pinned inputs to produce a fresh artifact, and compares its metrics against the originally published noise band. Observable result: a pass/fail verification stating whether every headline metric of the replayed run falls inside the published band, computed from the manifest and artifact directory alone.

## Scope

1. **Batch runner**: iterate the corpus × reps using the E40-F02 single-run command; resumable (skip already-completed (task, rep) pairs by scanning existing artifacts).
2. **Aggregator**: read JSONL artifacts → per-task and overall: oracle pass rate, rejection rate per gate, rework loops, tokens/cost per stage and total, wall-clock (stage/API/harness), LOC, quality-gate pass rates.
3. **Spread**: per metric, per task: min/median/max across reps; flag metrics where spread exceeds the mean (unusable for comparison at 3 reps).
4. **Report**: one markdown report + the machine-readable aggregate; records exact model IDs, corpus SHA, bundle version.
5. **Replay verification (G7/UAT-07)**: given a stored manifest, re-run E40-F02's single-run command pinned to that manifest's fixture SHA, model IDs, and variant bundle; diff the replayed run's per-metric values against the originally published noise band; report pass/fail per metric.

## Acceptance Criteria

- [ ] Full 30-run baseline completes (resumably) and aggregates without manual editing
- [ ] Report shows per-metric spread, not just means; noise band stated explicitly per metric
- [ ] Tasks that every rep aces or every rep fails are flagged as non-discriminative (corpus feedback to E40-F01)
- [ ] Report is reproducible from artifacts alone (no state outside the artifact dir)
- [ ] Given a stored manifest, replay reproduces a fresh artifact whose metrics fall within the published noise band (G7/UAT-07); the check runs from the manifest and artifact directory alone

## Out of Scope

- Variant comparison / A-B delta reports (Phase 2 — needs variant bundles and the paired-comparison method)
- Statistical significance testing beyond min/median/max spread (revisit if 3-rep bands prove too coarse)

## Success Metric

A deliberate no-op re-run of the baseline lands inside its own published noise band for every reported metric.

---

*Last Updated*: 2026-08-05
