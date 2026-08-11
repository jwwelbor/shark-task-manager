---
epic: E40
title: Shark Bench UAT plan
date: 2026-08-11
---

# E40 UAT plan

Use observable retained evidence, not implementation claims. UAT-01, UAT-02,
UAT-05, UAT-06, and UAT-07 preserve the completed Phase 1 contract. UAT-08
through UAT-15 gate lifecycle v2. UAT-03 and UAT-04 remain the configuration
comparison scenarios and now have owners in E40-F09 and E40-F10.

## Phase 1 regression scenarios

| ID | Scenario | Verify | Owner and criteria |
|---|---|---|---|
| UAT-01 | Unattended baseline batch produces a report with a noise band | Start the 10 x 3 v1 batch and leave it. Every run yields a record, already-completed pairs are not duplicated, and the report shows a value and observed spread for each headline metric. | E40-F02/E40-F03; G2, G4, G5 |
| UAT-02 | A broken corpus item is rejected at admission | Reject a candidate when the reference patch leaves F2P red or the base P2P set is red. Name the failing check and reproduce the verdict on a clean checkout. | E40-F01; G1 |
| UAT-05 | A stuck run is bounded and recorded | Kill a v1 stage at its cap, record `outcome=timeout` and the stalled stage, and continue the batch. | E40-F02/E40-F04; G2, G4 |
| UAT-06 | An in-flight `shark run` is observable | stderr or `run.log` shows child-correct stage progress and heartbeats while stdout remains exactly one JSON `RunResult`. | E40-F04; G3 |
| UAT-07 | A stored manifest reproduces its result | Replay the pinned fixture, models, and bundle from artifacts alone and reproduce metrics within the stored noise band. | E40-F03; G7 |

## Configuration comparison scenarios

| ID | Scenario | Verify | Owner and criteria |
|---|---|---|---|
| UAT-03 | A config delta inside the band is reported as no effect | Compare uniform paired runs whose deltas remain inside the matching band. Report `no detectable effect`; do not call the variant better or worse. | E40-F09/E40-F10; G6, G14 |
| UAT-04 | A config delta outside the band is reported as a measured change | Compare uniform paired runs whose delta clears the band. Name the metric, direction, magnitude, per-scenario pairs, exact identities, and reproducibility result. | E40-F09/E40-F10; G6, G14 |

E40-F03 remains complete and does not acquire UAT-03 or UAT-04. Lifecycle v2
implements the comparison and publication behavior from new I-07/I-08 records.

## Lifecycle v2 scenarios

| ID | Scenario | Verify | Owner and criteria |
|---|---|---|---|
| UAT-08 | All four lifecycle families are admitted correctly | Load one versioned feature, bug, change-card, and tech-debt package on the controlled Python fixture. Verify stable identity, adapter selection, final predicate, and the exact applicable/non-applicable stage matrix. Reject a malformed or non-runnable package with the failing field named. | E40-F05; G8; I-04 |
| UAT-09 | Evaluator truth is isolated and stage evidence replays | Inspect both agent-visible roots immediately before every dispatch and prove that references, answer keys, patches, and hidden tests are absent. After the stage or run, grant recorded evaluator access and replay the snapshot without rerunning the worker. | E40-F06; G9; I-05; X-09 |
| UAT-10 | Product design replays without live input | Run a feature scenario through the existing Rider D01-D05 action using only its versioned response bundle. Record each consumed response and artifact lineage. Disable live research and human input; a missing response stops as `unresolved_gate`. Verify other families record D01-D05 as non-applicable. | E40-F07; G10; I-06; X-10 |
| UAT-11 | The canonical keyed lifecycle executes every eligible entity | Exercise a hierarchy fork and agent-generated child tasks. Preserve each dispatch response, record the scheduling choice, claim and heartbeat the concrete entity, pass its prompt unchanged, persist the semantic outcome, apply the configured transition, release on every path, and execute every eligible task. | E40-F08; G11; I-07; X-11 |
| UAT-12 | Questions, failures, and safety stops remain truthful | Exercise Question routing, missing replay input, lease loss, worker failure, missing outcome, cancellation, pause, archive, error, and each resource ceiling. Verify the named stop outcome, retained partial evidence, released lease where owned, and baseline ineligibility. | E40-F08; G12; I-07; X-13 |
| UAT-13 | Structural, judge, and execution truth stay separate | Supply artifacts with a structural defect, a calibrated-judge disagreement, and an implementation that reaches terminal workflow status but fails the held-back oracle. Verify all three results remain distinct and each blocks publication as configured. | E40-F09; G13; I-08 |
| UAT-14 | Comparison identity fails closed | Omit each required identity field and mix one field at a time across otherwise valid runs. Reject every aggregate, retain the invalid inventory and divergence reason, and accept only a fully uniform set. | E40-F09; G14; I-08; X-12 |
| UAT-15 | Operator commands prevent accidental spend and gate publication | Run preview and report operations with provider access monitored and verify zero calls. Verify pilot and baseline operations refuse missing acknowledgement or non-positive limits. Retain and inspect one real pilot per family, including raw artifacts and oracle evidence, before publishing a repeated baseline. | E40-F10; G15; I-07, I-08 |

## Interaction coverage

- **I-01-I-03:** preserve the completed v1 corpus, record, and liveness shapes
  through UAT-01, UAT-02, UAT-05, UAT-06, and UAT-07.
- **I-04:** UAT-08 validates the lifecycle scenario package before E40-F06,
  E40-F07, or E40-F08 consumes it.
- **I-05:** UAT-09 validates the stage snapshot and three-root isolation before
  E40-F09 or E40-F10 trusts it.
- **I-06:** UAT-10 validates product-design replay before E40-F08 starts the
  feature entity lifecycle.
- **I-07:** UAT-11 and UAT-12 validate complete and partial lifecycle run
  records before evaluation or reporting.
- **I-08:** UAT-13 and UAT-14 validate the evaluation and identity verdict before
  UAT-15 publishes it.
- **X-07-X-09:** preserve and extend runner/usage compatibility through the
  named Phase 1 and v2 scenarios.
- **X-10-X-13:** UAT-10, UAT-11, UAT-12, and UAT-14 cover product design, Rider
  execution, Shark-data identity, and Questions respectively.

## Non-functional evidence

**Integrity:** Retain scenario packages, stage snapshots, run records,
evaluation records, transcripts, claims, transitions, and oracle results under
an explicit output root. Never use terminal status or worker self-report as the
implementation oracle.

**Isolation:** Keep the scratch Shark project, agent-visible fixture checkout,
and evaluator-only root separate. Inspect actual dispatch-time visibility, not
only the intended directory layout.

**Cost and safety:** Require explicit provider-spend acknowledgement and
positive cost, wall-time, and generated-task ceilings. Preserve partial evidence
when a ceiling stops a scenario and exclude that scenario from publication.

**Reproducibility:** Pin the complete identity listed in I-08. Reject rather
than normalize missing or mixed identities.
