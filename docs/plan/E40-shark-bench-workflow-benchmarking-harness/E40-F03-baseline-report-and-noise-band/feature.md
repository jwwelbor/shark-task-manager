---
feature_key: E40-F03-baseline-report-and-noise-band
epic_key: E40
title: Baseline report and noise band
description: Run the Phase 1 matrix (~10 tasks x default config x 3 reps), aggregate JSONL artifacts, and produce the baseline report: pass rate, rejection rate per gate, tokens/cost per step, wall-clock, LOC — with run-to-run spread per metric. The published noise band is the deliverable: it defines the threshold any future config delta must clear in paired per-task comparison.
---

# Baseline report and noise band

**Feature Key**: E40-F03

See [shark-bench-design.md](../shark-bench-design.md) §5 (comparison method).

---

## Goal

Turn 30 run artifacts (~10 tasks × default workflow config × 3 reps) into the Phase 1 deliverable: a baseline report whose central output is the **noise band** — per-metric run-to-run spread on an unchanged config. Every future config comparison (Phase 2) is judged against this band; deltas inside it are "no detectable effect."

## Scope

1. **Batch runner**: iterate the corpus × reps using the E40-F02 single-run command; resumable (skip already-completed (task, rep) pairs by scanning existing artifacts).
2. **Aggregator**: read JSONL artifacts → per-task and overall: oracle pass rate, rejection rate per gate, rework loops, tokens/cost per stage and total, wall-clock (stage/API/harness), LOC, quality-gate pass rates.
3. **Spread**: per metric, per task: min/median/max across reps; flag metrics where spread exceeds the mean (unusable for comparison at 3 reps).
4. **Report**: one markdown report + the machine-readable aggregate; records exact model IDs, corpus SHA, bundle version.

## Acceptance Criteria

- [ ] Full 30-run baseline completes (resumably) and aggregates without manual editing
- [ ] Report shows per-metric spread, not just means; noise band stated explicitly per metric
- [ ] Tasks that every rep aces or every rep fails are flagged as non-discriminative (corpus feedback to E40-F01)
- [ ] Report is reproducible from artifacts alone (no state outside the artifact dir)

## Out of Scope

- Variant comparison / A-B delta reports (Phase 2 — needs variant bundles and the paired-comparison method)
- Statistical significance testing beyond min/median/max spread (revisit if 3-rep bands prove too coarse)

## Success Metric

A deliberate no-op re-run of the baseline lands inside its own published noise band for every reported metric.

---

*Last Updated*: 2026-08-05
