---
type: interaction-map
epic: E40
last_updated: 2026-08-13
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
| I-05 | E40-F06 Stage evidence and evaluator isolation | E40-F08 Canonical multi-entity lifecycle runner; E40-F09 Calibrated evaluation and comparison identity; E40-F10 Operator workflow and retained lifecycle baseline | [Stage evidence and isolation contract](architecture.md#stage-evidence-and-isolation-contract) | Three-root access policy; immutable stage snapshot; non-overlapping time ledger; exact candidate snapshot; typed artifact producer, consumer, and access records; prompt, input, replay, output, usage, cost, error, rework, digest, and evaluator-access lineage | File artifact and access policy |
| I-06 | E40-F07 Replayable product-design prelude | E40-F08 Canonical multi-entity lifecycle runner | [Product-design replay contract](architecture.md#product-design-replay-contract) | Authorized replay response sequence, D01-D05 artifact references and digests, request and response volume, revision and unresolved-gate counts, downstream artifact-consumption lineage, and terminal prelude outcome | File artifact |
| I-07 | E40-F08 Canonical multi-entity lifecycle runner | E40-F09 Calibrated evaluation and comparison identity; E40-F10 Operator workflow and retained lifecycle baseline | [Lifecycle run record contract](architecture.md#lifecycle-run-record-contract) | Entity graph; dispatches; scheduling; claims and transitions; stage intervals; candidate and artifact references; workflow-policy identity; explicit review-gate results and raw structured findings; Questions; usage; limits; stop outcome; aggregate eligibility | File artifact |
| I-08 | E40-F09 Calibrated evaluation and comparison identity | E40-F10 Operator workflow and retained lifecycle baseline | [Lifecycle evaluation record contract](architecture.md#lifecycle-evaluation-record-contract) | Structural, calibrated-judge, and held-back-oracle results; normalized and confirmed review findings; independent and sequential gate-comparison evidence; complete candidate and workflow-policy identity; eligibility verdict; invalidity reasons | File artifact |

All rows use the default `live` gate mode except I-04 through I-08, staged
below. A feature workflow may refine a shape only if it updates the
architecture source, this row, every named consumer, and the matching UAT
scenario before passing feature review.

I-04 through I-08 share one structural cause: each producer is the feature
that establishes a new v2 contract, and the execution order (below) always
sequences that producer before the consumer(s) who will read it — F05 before
F06/F07/F08, F06 before F08/F09/F10, F07 before F08, F08 before F09/F10, F09
before F10. A `live` gate at any producer's task-decomposition review can
therefore never be satisfied by that producer alone: it would require
consumer task files for a feature that has not yet reached its own
`task_generation`. Each row below stages the same predeclared `contract-only`
handoff I-04 uses, discovered and fixed first at F05's task_review
(2026-08-12).

## I-04 staged edge

I-04's producer (E40-F05, execution order 5) necessarily runs before any of
its three named consumers (E40-F06/F07/F08, execution order 6-8) are
decomposed — that is the documented reason F05 is ordered first (see
Execution order below). A `live` gate at F05's task-decomposition review can
never be satisfied by F05 alone: it requires consumer task files that cannot
exist yet. I-04 is therefore a predeclared `contract-only` handoff: E40-F05
produces the documented shape and each of E40-F06, E40-F07, and E40-F08 is the
activation owner for its own slice of read-only consumption (per spec.md's
Consumer split: F07 reads `stage_matrix.prelude`/`replay_reference`; F06 reads
`evaluator_only`/`toolchain_identity`/both stage-matrix halves; F08 reads
`stage_matrix.lifecycle`/`adapter`/`fixture`/`resource_policy`).

| Field | Assigned value |
|-------|----------------|
| `gate_mode` | `contract-only` until each consumer proves live production-path use |
| `activation_owner` | E40-F06 (its slice); E40-F07 (its slice); E40-F08 (its slice) — each closes its own consumption independently |
| `closure_key` | E40-F06 / E40-F07 / E40-F08, respectively, at each feature's own UAT |
| `counterpart_status` | Read live from Shark at review/UAT time; this map intentionally contains no copied current-state snapshot |
| `review_basis` | E40-F05's completed specification (`spec.md`) and this map row, present together at F05 task_review |
| `demonstrability_disposition` | `pending-integration` until each consumer's live wiring closes; no override makes it demonstrated-now |

The map table above supplies the counterpart identities (E40-F05 producer;
E40-F06/F07/F08 consumers) and the shared contract evidence (shape source,
payload, shared contract-test pointer `tests/contracts/e40_i04_scenario_contract_test.go#TC-030`).
An internal activation obligation blocks epic completion until each of
E40-F06, E40-F07, and E40-F08 closes it during its own UAT with a real caller
chain, shared-contract evidence, a production-path integration test, and a
wiring-removal counterfactual — a future named owner is not itself live
integration evidence.

## I-05 staged edge

I-05's producer (E40-F06, order 6) runs before its consumers E40-F08/F09/F10
(order 8-10) are decomposed. Predeclared `contract-only` handoff:

| Field | Assigned value |
|-------|----------------|
| `gate_mode` | `contract-only` until each consumer proves live production-path use |
| `activation_owner` | E40-F08; E40-F09; E40-F10 — each closes its own consumption independently |
| `closure_key` | E40-F08 / E40-F09 / E40-F10, respectively, at each feature's own UAT |
| `counterpart_status` | Read live from Shark at review/UAT time; not copied here |
| `review_basis` | E40-F06's completed specification and this map row, present together at F06 task_review |
| `demonstrability_disposition` | `pending-integration` until each consumer's live wiring closes |

Shared contract evidence is the map row above (shape source
[Stage evidence and isolation contract](architecture.md#stage-evidence-and-isolation-contract));
F06's spec.md must name the shared contract-test pointer at specification
time, the same way F05's spec.md named TC-030 for I-04.

## I-06 staged edge

I-06's producer (E40-F07, order 7) runs before its consumer E40-F08 (order 8)
is decomposed. Predeclared `contract-only` handoff:

| Field | Assigned value |
|-------|----------------|
| `gate_mode` | `contract-only` until E40-F08 proves live production-path use |
| `activation_owner` | E40-F08 |
| `closure_key` | E40-F08, at its own UAT |
| `counterpart_status` | Read live from Shark at review/UAT time; not copied here |
| `review_basis` | E40-F07's completed specification and this map row, present together at F07 task_review |
| `demonstrability_disposition` | `pending-integration` until E40-F08's live wiring closes |

Shared contract evidence is the map row above (shape source
[Product-design replay contract](architecture.md#product-design-replay-contract));
F07's spec.md must name the shared contract-test pointer at specification
time.

## I-07 staged edge

I-07's producer (E40-F08, order 8) runs before its consumers E40-F09/F10
(order 9-10) are decomposed. Predeclared `contract-only` handoff:

| Field | Assigned value |
|-------|----------------|
| `gate_mode` | `contract-only` until each consumer proves live production-path use |
| `activation_owner` | E40-F09; E40-F10 — each closes its own consumption independently |
| `closure_key` | E40-F09 / E40-F10, respectively, at each feature's own UAT |
| `counterpart_status` | Read live from Shark at review/UAT time; not copied here |
| `review_basis` | E40-F08's completed specification and this map row, present together at F08 task_review |
| `demonstrability_disposition` | `pending-integration` until each consumer's live wiring closes |

Shared contract evidence is the map row above (shape source
[Lifecycle run record contract](architecture.md#lifecycle-run-record-contract));
F08's spec.md must name the shared contract-test pointer at specification
time.

## I-08 staged edge

I-08's producer (E40-F09, order 9) runs before its consumer E40-F10 (order 10)
is decomposed. Predeclared `contract-only` handoff:

| Field | Assigned value |
|-------|----------------|
| `gate_mode` | `contract-only` until E40-F10 proves live production-path use |
| `activation_owner` | E40-F10 |
| `closure_key` | E40-F10, at its own UAT |
| `counterpart_status` | Read live from Shark at review/UAT time; not copied here |
| `review_basis` | E40-F09's completed specification and this map row, present together at F09 task_review |
| `demonstrability_disposition` | `pending-integration` until E40-F10's live wiring closes |

Shared contract evidence is the map row above (shape source
[Lifecycle evaluation record contract](architecture.md#lifecycle-evaluation-record-contract));
F09's spec.md must name the shared contract-test pointer at specification
time.

For every staged row I-04 through I-08, the activation obligation blocks epic
completion until its named owner(s) close it during their own UAT with a real
caller chain, shared-contract evidence, a production-path integration test,
and a wiring-removal counterfactual — a future named owner is not itself live
integration evidence.

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
- E40-F09 owns aggregate eligibility, finding normalization and confirmation,
  and the independent-versus-sequential review comparison contract. E40-F10 may
  format or publish those results, but it may not weaken or recompute them.
- E40-F10 retains both the lifecycle headline and stage diagnostic view from the
  same I-07/I-08 inputs. A stage view is never a separate product baseline.
- Finish-feature scope in lifecycle v2 stops at its controlled deep-review gate.
  PR feedback, CI, merge, and cleanup remain deferred delivery-tail scenarios.

Cross-epic seams use X-07 through X-13 in
[E40-cross-epic-map.md](E40-cross-epic-map.md). Keep the I and X namespaces
separate in feature specs, task specs, reviews, QA, and UAT.
