---
research_schema: 2
entity_key: E40-F10
entity_type: feature
recipe: universal
rigor: standard
categories:
  - backend
  - data
  - workflow_operations
  - documentation
related_work: true
---

# Research report: Operator workflow and retained lifecycle baseline

## Scope

E40-F10 is the operator-facing layer over lifecycle v2. It adds no-provider
preview, explicit spend acknowledgement and positive resource ceilings,
retained-pilot inspection, a lifecycle headline view and a subordinate stage
diagnostic view, machine-readable aggregates, noise bands, and the
independent-versus-sequential QA-vs-deep-review comparison operation. It
consumes I-07 (`E40-F08`'s lifecycle run record) and I-08 (`E40-F09`'s
evaluation record) as read-only inputs: F10 formats and publishes those
verdicts but must not recompute or weaken them
(`E40-interaction-map.md` boundary rules). It does not implement PR-comment,
CI, merge, or branch-cleanup phases of finish-feature, and it does not
relabel the completed Phase-1 F02/F03 run and report surfaces as the
lifecycle v2 baseline.

This report finds F10's implementation surface substantially unstarted:
`bench/scripts/` has no script, and
`docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-.../` has no
`spec.md`, `test-plan.md`, or `tasks/` yet, confirmed by directory listing
against the seven other lifecycle-v2 feature folders which all carry those
files. F10 does, however, inherit several close operator-facing patterns
already built for F02/F03/F08 that materially narrow its design space.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/feature.md` (Scope, Acceptance boundary, Contracts, Out of scope, Workflow handoff, 2026-08-13 amendment) and `architecture.md#workflow-value-attribution-contract` define the dry-run/spend-acknowledgement/pilot/lifecycle-headline/stage-diagnostic/noise-band/review-comparison vocabulary this feature must implement.
- [x] `affected_implementation_or_contract` — Evidence: no `bench/scripts/*.sh` file, `spec.md`, `test-plan.md`, or `tasks/` directory exists yet under the F10 feature folder (confirmed by `find docs/plan/.../E40-F10-.../` returning only `feature.md`), so F10's concrete "affected implementation" is the *contracts it must read*: `bench/runs/i07-schema.yaml` (I-07 top-level fields: `identity`, `entity_graph`, `dispatches`, `stages`, `workflow_policy`, `review_gates`, `questions`, `limits`, `outcome`) and `bench/evaluation/i08-schema.yaml` (I-08 top-level fields: `identity`, `source_artifacts`, `structural`, `judge`, `execution_oracle`, `eligibility` with `aggregate_eligible`/`publication_eligible`/`invalidity_reasons`, `candidate_snapshots`, `workflow_policy`, `comparison`, `review_findings`, `metrics`).
- [x] `related_work` — Evidence: upstream `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md`; sibling reports `E40-F08-canonical-multi-entity-lifecycle-runner/research-report.md` and `E40-F09-calibrated-evaluation-and-comparison-identity/research-report.md` (I-07/I-08 producer decisions this feature must not recompute); `E40-interaction-map.md` I-07/I-08 rows, staged edges, and boundary rules; `E40-cross-epic-map.md`; and `uat-plan.md` UAT-15 through UAT-18 (this feature's acceptance scenarios).
- [x] `pattern_contract` — Evidence: `bench/scripts/run-batch.sh` (`--dry-run` flag; `dispatch_pair()` short-circuits to `append_summary` with zero subprocess calls, lines ~458-462) and `bench/scripts/run-lifecycle.sh` (`--mode contract|dry-run`; `parse_limits()` already rejects non-positive `max_cost_usd`/`max_wall_clock_seconds`/`max_generated_tasks`, lines ~300-314; `adapter_result()` returns a synthetic offline result in `contract`/`dry-run` mode without invoking the adapter, lines ~335-337) establish the existing zero-provider-call preview idiom this feature must extend, not replace. `bench/scripts/report-baseline.sh` (pure function of one `aggregate.json`, no clock/subprocess/network access) and `bench/scripts/compare-lifecycle-evaluations.sh` (fail-closed I-08 comparison keyed on `MODES = {independent_frozen_candidate, sequential_delivery}` and candidate/policy digest fields) establish the report-purity and comparison-identity patterns F10's reports and review-comparison operation must match.

## Capability map

| Capability | Brownfield evidence | Decision | E40-F10 responsibility |
|---|---|---|---|
| I-07 lifecycle run record | `E40-F08` research report; `bench/runs/i07-schema.yaml` | REUSE (read-only) | Consume dispatches, stages, `workflow_policy`, `review_gates`, `limits`, `outcome`; never adjudicate a raw finding or infer completion. |
| I-08 lifecycle evaluation record | `E40-F09` research report; `bench/evaluation/i08-schema.yaml` | REUSE (read-only) | Consume `eligibility.aggregate_eligible`/`publication_eligible`/`invalidity_reasons`, `comparison`, `review_findings`, `metrics` verbatim; format and publish, never recompute or weaken the verdict (`E40-interaction-map.md` boundary rules). |
| Zero-provider-call dry run / `--mode contract\|dry-run` | `bench/scripts/run-batch.sh` `--dry-run`; `bench/scripts/run-lifecycle.sh` `--mode dry-run` | EXTEND | Reuse the established "short-circuit before subprocess dispatch" idiom for F10's operator preview, but extend it to surface the scenario matrix, applicable stages, planned provider calls, and output root — none of which the existing dry-run paths currently print; they only suppress dispatch. |
| Positive resource-ceiling enforcement | `bench/scripts/run-lifecycle.sh` `parse_limits()` (already rejects non-positive `max_cost_usd`/`max_wall_clock_seconds`/`max_generated_tasks`) | REUSE | The ceiling-positivity check F10's acceptance boundary requires already exists in the lifecycle runner; F10 must not reimplement it, only surface it in preview and require it before any pilot/baseline command starts. |
| Explicit provider-spend acknowledgement gate | Not found anywhere in `bench/scripts/*.sh` (`grep` for `acknowledge`/`--i-acknowledge`/`--confirm` returns no matches) | NEW | F10 must add the first explicit spend-acknowledgement flag/prompt in the codebase; no existing pattern to extend. |
| Retained pilot / raw-evidence inspection layout | Not found; `bench/runs/`, `bench/evidence/`, `bench/evaluation/` hold schema and UAT fixtures, not an operator retention convention | NEW | F10 defines the retention layout (raw scenario packages, stage evidence, run records, evaluation records, transcripts, entity history, oracle results) under an explicit output root; no prior art to reuse beyond the existing artifact *shapes* (I-05/I-07/I-08) it must retain unmodified. |
| Report purity (pure function of retained artifacts, no clock/subprocess/network) | `bench/scripts/report-baseline.sh` header contract; `bench/scripts/aggregate-runs.sh` header contract | REUSE | F10's lifecycle headline and stage diagnostic reports should follow the same purity discipline: read only the retained I-07/I-08 records named by the operator, compute no new statistic outside what those upstream contracts already carry. |
| Independent-frozen-candidate vs. sequential-delivery comparison | `architecture.md#workflow-value-attribution-contract`; `bench/scripts/compare-lifecycle-evaluations.sh` (`MODES = {independent_frozen_candidate, sequential_delivery}`, candidate/policy digest fields) | REUSE | F09 already implements the comparison-identity fail-closed check; F10 adds the *operator* half — preview candidates/policies/expected provider calls/truth-set availability/fix rules before spend, and publish/report the already-computed comparison, not a second implementation of it. |
| Phase-1 F02/F03 run and report surfaces | `bench/scripts/run-batch.sh`, `aggregate-runs.sh`, `report-baseline.sh` | REUSE, non-relabeled | Feature contract explicitly requires these remain low-cost regression/compatibility inputs; F10 must not present them as the lifecycle v2 baseline. |

## Findings

1. **F10's own implementation surface is unstarted.** Every other lifecycle-v2
   feature folder (F05 through F09) already has `spec.md`, `test-plan.md`, and
   a populated `tasks/` directory; F10's folder contains only `feature.md`.
   This is expected given execution order (F10 is last, gated on I-07/I-08),
   not a gap in prior work.

2. **The zero-provider-call preview idiom already exists twice, but neither
   instance satisfies F10's preview content requirement.** `run-batch.sh
   --dry-run` and `run-lifecycle.sh --mode dry-run` both suppress the
   subprocess/adapter call, but neither prints the scenario matrix,
   applicable stages, planned provider calls, output root, or resource
   ceilings that F10's acceptance boundary requires a dry run to show. F10
   extends the *mechanism* (short-circuit before dispatch) but must add new
   preview-content logic; it should not fork a third dry-run flag convention.

3. **Positive resource-ceiling validation is already implemented once, for
   the lifecycle runner only.** `run-lifecycle.sh`'s `parse_limits()` rejects
   `max_cost_usd`/`max_wall_clock_seconds`/`max_generated_tasks` ≤ 0
   unconditionally — for every mode, not gated on an acknowledgement flag.
   F10's acceptance boundary ("provider-backed commands refuse to start
   without explicit spend acknowledgement and positive resource ceilings")
   is therefore half-satisfied structurally by an existing lower-layer
   check; the missing half — the spend-acknowledgement gate itself — has no
   precedent anywhere in `bench/scripts/`.

4. **No spend-acknowledgement pattern exists in this codebase.** A targeted
   search of `bench/scripts/*.sh` for `acknowledge`, `--i-acknowledge`, and
   `--confirm` returns zero matches. This is genuinely new surface for F10,
   not an extension of an existing flag family.

5. **The report-purity discipline F10 should inherit is explicit and
   testable.** `report-baseline.sh`'s header states it is "a pure function of
   one `aggregate.json` document... consults no clock, invokes no
   subprocess, and prints one markdown document to stdout" and that "every
   value in the report traces to a field already present in the aggregate...
   this script computes no new statistic." `compare-lifecycle-evaluations.sh`
   similarly loads exactly one retained I-08 record per side and fails
   closed on unrecognized comparison mode or missing digest fields. F10's
   lifecycle-headline and stage-diagnostic reports, and its review-comparison
   preview/report, should hold the same purity and fail-closed discipline
   against retained I-07/I-08 records rather than re-deriving metrics.

6. **F09 already owns the comparison-identity mechanics F10 must not
   duplicate.** `compare-lifecycle-evaluations.sh` already enforces
   `independent_frozen_candidate`/`sequential_delivery` mode validity and
   candidate/workflow-policy digest presence. F10's "independent frozen-
   candidate and actual sequential QA-versus-finish-feature deep-review
   operations" scope item is therefore an *operator wrapper* — preview,
   spend gate, and report — around an evaluation F09 has already built the
   comparison engine for, not a second comparison implementation.

7. **No retention-layout convention exists to reuse or extend.** `bench/runs/`,
   `bench/evidence/`, and `bench/evaluation/` currently hold schema files and
   UAT fixture data (e.g. `bench/runs/e40-f08-uat/*.json`), not an
   operator-facing "retained pilot" directory convention. F10 must design
   this layout new, while keeping the I-05/I-07/I-08 artifact *shapes* it
   retains exactly as F06/F08/F09 defined them.

## Decisions

1. Extend, not fork, the existing `--dry-run`/`--mode dry-run` idiom from
   `run-batch.sh`/`run-lifecycle.sh`: add scenario-matrix, applicable-stage,
   planned-provider-call, output-root, and ceiling-preview output to a
   consistent operator-facing preview surface, rather than inventing a third
   flag convention.
2. Build the spend-acknowledgement gate as new surface; there is no local
   pattern to reuse, so its design should be settled explicitly in F10's
   workflow handoff (spec.md) before task decomposition, per the feature's
   own "Workflow handoff" requirement.
3. Reuse `run-lifecycle.sh`'s existing positive-ceiling check as the
   ceiling half of the acceptance boundary; do not reimplement ceiling
   validation. Wire the new acknowledgement check alongside it rather than
   replacing it.
4. Treat I-07 and I-08 as strictly read-only inputs. F10's lifecycle
   headline, stage diagnostic view, and review-comparison report must format
   and publish `eligibility`/`comparison`/`review_findings`/`metrics` fields
   already present in the retained records; any apparent gap in those
   records is an F08/F09 defect to flag, not an F10 workaround to compute
   around.
5. Model F10's report scripts on `report-baseline.sh`'s and
   `compare-lifecycle-evaluations.sh`'s purity contract: read only the
   retained artifact(s) an operator names, touch no clock/subprocess/network,
   and derive no new statistic the upstream I-07/I-08 contract does not
   already carry.
6. Design the retained-pilot output-root layout as new work, but pin it to
   preserve the I-05/I-07/I-08 shapes verbatim (raw scenario packages, stage
   evidence, run records, evaluation records, transcripts, entity history,
   oracle results) rather than re-serializing them.
7. Keep F02/F03's existing run and report surfaces reachable as low-cost
   regression/compatibility tooling in F10's operator command set, but never
   label their output as the lifecycle v2 baseline (feature contract,
   `E40-interaction-map.md` boundary rules).

## Sources

- `internal/sharkdata/default_data/research/recipes.yaml` (universal v2 modules and standard-rigor rule).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/feature.md` (scope, acceptance boundary, contracts, out of scope, workflow handoff).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (epic-level brownfield findings, cited not duplicated).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (`#lifecycle-run-record-contract`, `#lifecycle-evaluation-record-contract`, `#workflow-value-attribution-contract`, ADR-008/ADR-009/ADR-010, Delivery boundaries and traceability).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` (I-07/I-08 rows, staged edges, boundary rules).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md` (UAT-15 through UAT-18).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/research-report.md` (I-07 producer decision).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F09-calibrated-evaluation-and-comparison-identity/research-report.md` (I-08 producer decision, comparison-identity boundary).
- `bench/runs/i07-schema.yaml` and `bench/evaluation/i08-schema.yaml` (I-07/I-08 field vocabularies).
- `bench/scripts/run-batch.sh` (`--dry-run` short-circuit).
- `bench/scripts/run-lifecycle.sh` (`--mode contract|dry-run`, `parse_limits()` positive-ceiling check, `adapter_result()` offline short-circuit).
- `bench/scripts/report-baseline.sh` and `bench/scripts/aggregate-runs.sh` (pure-report and pure-aggregation header contracts).
- `bench/scripts/compare-lifecycle-evaluations.sh` (I-08 comparison-identity fail-closed check, `MODES`).
- Directory listings confirming F10 has no `spec.md`/`test-plan.md`/`tasks/` and no `bench/scripts/*.sh` file names it yet, versus F05-F09 which all do.
