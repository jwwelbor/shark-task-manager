---
type: interaction-map
epic: E40
last_updated: 2026-08-11
---

# E40 cross-feature interaction map

E40 has two delivery phases under one epic. E40-F01 through E40-F04 preserve
the completed v1 corpus, collector, report, and `shark run` liveness contracts.
E40-F05 through E40-F10 define the active lifecycle v2 tranche. Every
interaction below has a named producer, named consumer, authoritative shape,
and observable downstream use.

| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |
|---|---|---|---|---|---|
| I-01 | E40-F01 Benchmark corpus v1 | E40-F02 Bench harness; E40-F05 Lifecycle scenario corpus | [Corpus and oracle contract](architecture.md#corpus-and-oracle-contract) | V1 manifest, entity seed, held-back F2P tests, P2P set, reference patch, fixture SHA, and base ledgers. F05 consumes the admission and oracle principles without making the Go manifest the global schema. | File artifact |
| I-02 | E40-F02 Bench harness | E40-F03 Baseline report and noise band | [Metric collection and artifact schema](architecture.md#metric-collection-and-artifact-schema) | One JSONL record per v1 run with manifest, per-stage, post-run, and rollup data | File artifact |
| I-03 | E40-F04 `shark run` live progress and per-run log | E40-F02 Bench harness | [Run liveness contract](architecture.md#run-liveness-contract) | stderr NDJSON progress and `.shark/runs/<run_id>/run.log` used to diagnose and attribute a killed v1 run | Process stream and file artifact |
| I-04 | E40-F05 Lifecycle scenario corpus and adapter contract | E40-F06 Stage evidence and evaluator isolation; E40-F07 Replayable product-design prelude; E40-F08 Canonical multi-entity lifecycle runner | [Lifecycle scenario package contract](architecture.md#lifecycle-scenario-package-contract) | Versioned scenario identity, family, stage matrix, fixture and adapter, visible input, replay and evaluator references, resource policy, final predicate, and admission result | File artifact |
| I-05 | E40-F06 Stage evidence and evaluator isolation | E40-F08 Canonical multi-entity lifecycle runner; E40-F09 Calibrated evaluation and comparison identity; E40-F10 Operator workflow and retained lifecycle baseline | [Stage evidence and isolation contract](architecture.md#stage-evidence-and-isolation-contract) | Three-root access policy and immutable stage snapshot with prompt, input, replay, output, usage, cost, elapsed, error, rework, digest, and evaluator-access lineage | File artifact and access policy |
| I-06 | E40-F07 Replayable product-design prelude | E40-F08 Canonical multi-entity lifecycle runner | [Product-design replay contract](architecture.md#product-design-replay-contract) | Authorized replay response sequence, D01-D05 artifact references and digests, consumption lineage, and terminal prelude outcome | File artifact |
| I-07 | E40-F08 Canonical multi-entity lifecycle runner | E40-F09 Calibrated evaluation and comparison identity; E40-F10 Operator workflow and retained lifecycle baseline | [Lifecycle run record contract](architecture.md#lifecycle-run-record-contract) | Entity graph; dispatches; scheduling decisions; claims, heartbeats, transitions, releases; evidence references; Questions; usage; limits; stop outcome; aggregate eligibility | File artifact |
| I-08 | E40-F09 Calibrated evaluation and comparison identity | E40-F10 Operator workflow and retained lifecycle baseline | [Lifecycle evaluation record contract](architecture.md#lifecycle-evaluation-record-contract) | Structural results, calibrated judge evidence, held-back execution-oracle result, complete comparison identity, eligibility verdict, and invalidity reasons | File artifact |

All rows use the default `live` gate mode. A feature workflow may refine a shape
only if it updates the architecture source, this row, every named consumer, and
the matching UAT scenario before passing feature review.

## Execution order

| Order | Feature(s) | Reason |
|---:|---|---|
| 1-4 | E40-F01 through E40-F04 | Completed v1 delivery; preserved as historical foundation |
| 5 | E40-F05 | Establish I-04 before any v2 consumer specifies its implementation |
| 6 | E40-F06 | Defines stage evidence and evaluator isolation after I-04 |
| 7 | E40-F07 | Defines product-design replay after I-04; F06 and F07 are dependency-independent, but Shark records a unique portfolio order |
| 8 | E40-F08 | Requires I-04, I-05, and I-06 plus the external Rider and Question contracts |
| 9 | E40-F09 | Requires captured stage evidence and a complete lifecycle run |
| 10 | E40-F10 | Requires the lifecycle and evaluation record shapes before operator publication behavior is specified |

## Boundary rules

- Completed v1 features do not acquire new acceptance criteria. New behavior
  belongs to E40-F05 through E40-F10.
- E40-F08 drives public Shark contracts but does not own generic dispatch,
  claims, Questions, or prompt assembly. Those seams are X-11 and X-13.
- E40-F09 owns aggregate eligibility. E40-F10 may format or publish the verdict,
  but it may not weaken or recompute it.
- E40-F10 retains both the lifecycle headline and stage diagnostic view from the
  same I-07/I-08 inputs. A stage view is never a separate product baseline.

Cross-epic seams use X-07 through X-13 in
[E40-cross-epic-map.md](E40-cross-epic-map.md). Keep the I and X namespaces
separate in feature specs, task specs, reviews, QA, and UAT.
