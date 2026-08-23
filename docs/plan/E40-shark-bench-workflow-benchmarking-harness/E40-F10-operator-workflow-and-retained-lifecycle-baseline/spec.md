---
feature_key: E40-F10-operator-workflow-and-retained-lifecycle-baseline
epic_key: E40
title: Operator workflow and retained lifecycle baseline
---

# E40-F10 Operator workflow and retained lifecycle baseline

This specification is incremental over the E40 epic. See [the epic PRD](../epic.md)
for business context and goals G6 and G15-G19, and [the epic architecture](../architecture.md)
for the system-level decisions — especially
[lifecycle run record](../architecture.md#lifecycle-run-record-contract),
[lifecycle evaluation record](../architecture.md#lifecycle-evaluation-record-contract),
[workflow value-attribution](../architecture.md#workflow-value-attribution-contract),
and ADR-002, ADR-006, ADR-008, ADR-009, ADR-010.

The validated research report for this feature is
[research-report.md](research-report.md). Its Capability map is authoritative for
reuse. F10 **reuses read-only**: the I-05 stage-evidence bundle, the I-07
lifecycle run record, the I-08 evaluation record, the positive-ceiling check
already implemented in `bench/scripts/run-lifecycle.sh` `parse_limits()`, the
zero-provider-call short-circuit idiom in `run-batch.sh --dry-run` and
`run-lifecycle.sh --mode dry-run`, the report-purity discipline of
`bench/scripts/report-baseline.sh` and `bench/scripts/aggregate-runs.sh`, and
the fail-closed comparison engine in
`bench/scripts/compare-lifecycle-evaluations.sh`. F10 **adds**: the operator
batch and review-comparison drivers, the explicit provider-spend acknowledgement
gate (no local precedent exists), the retained-pilot retention-root layout and
inspection ledger, the lifecycle aggregate with noise bands and invalid-run
inventory, and the lifecycle headline plus subordinate stage-diagnostic reports.
F10 **does not re-implement** evaluation, finding adjudication, comparison
identity, oracle execution, stage-evidence capture, workflow routing, claims,
prompts, or Questions.

## Requirements

### Functional requirements

| ID | Requirement | Traceability |
|---|---|---|
| REQ-F-001 | F10 MUST provide one operator preview that makes zero provider calls and prints, for the requested batch: the scenario matrix (scenario id, version, family, reps), the applicable stages per scenario resolved from the admitted I-04 stage matrix, the planned provider-call inventory per scenario and stage (including which stages are replayed and therefore make none), the resolved retention root, the declared resource ceilings, and the current pilot-ledger state per family. The preview MUST extend the existing `--dry-run` / `--mode dry-run` idiom rather than introduce a third flag convention. | Epic G15; feature Scope 1; research Decision 1 |
| REQ-F-002 | Every provider-backed operator command (`--mode pilot`, `--mode baseline`) MUST refuse to start unless all of: an explicit `--acknowledge-provider-spend` flag is present on the command line; `--max-cost-usd`, `--max-wall-clock-seconds`, and `--max-generated-tasks` are all supplied and strictly positive; and a declared `--retention-root` is given. The refusal MUST occur before any subprocess, scenario checkout, or Shark invocation, MUST name the missing condition in a machine-readable refusal reason, and MUST exit with a refusal status distinct from a usage-error status. Acknowledgement MUST NOT be satisfiable by an environment variable, a config default, or a stored prior acknowledgement. | Epic G15; feature Acceptance boundary 2; research Findings 3-4 |
| REQ-F-003 | Ceiling positivity MUST be enforced by delegating to the existing `run-lifecycle.sh` `parse_limits()` check for the per-scenario run, and re-checked by the batch driver before dispatch so that a refusal costs zero subprocesses. F10 MUST NOT add a second, divergent ceiling-validation implementation. | Feature Acceptance boundary 2; research Decision 3 |
| REQ-F-004 | F10 MUST retain, under one explicit operator-declared retention root (`--retention-root`), for each (scenario, rep): the admitted I-04 scenario package, the I-05 stage-evidence bundle, stage transcripts, the scratch-project entity history, the I-07 lifecycle record, the I-08 evaluation record, the held-back oracle result, and any review-comparison record. Retained artifacts MUST be byte-preserved copies verified by digest against their source; F10 MUST NOT re-serialize, normalize, reorder, or summarize an upstream artifact in place of retaining it. | Epic G15; feature Scope 3; research Decision 6 |
| REQ-F-005 | F10 MUST record one operator pilot-inspection attestation per scenario family, containing the inspected run reference, the checklist item results, the inspecting operator identity, and the digests of the artifacts inspected. `--mode baseline` MUST refuse to start when any scenario family in the requested matrix lacks a verified attestation whose inspected-artifact digests still match the retained artifacts. | Epic G15; feature Scope 12, Acceptance boundary 2 |
| REQ-F-006 | The retained-pilot layout MUST expose enough raw evidence to diagnose a workflow that reached a terminal completed status while its held-back execution oracle failed: the I-08 `execution_oracle` result, the I-07 dispatch and gate lineage, the per-stage I-05 evidence, and the stage transcripts MUST all be reachable from the scenario's retention directory without rerunning the scenario or contacting a provider. | Feature Acceptance boundary 4; UAT-15 |
| REQ-F-007 | F10 MUST produce a machine-readable lifecycle aggregate from retained I-07/I-08 pairs under one retention root. The aggregate MUST carry: batch and provenance identity, per-scenario eligibility verdicts copied verbatim from I-08 `eligibility`, an explicit invalid-run inventory carrying every `invalidity_reasons` entry, per-(scenario, metric) noise bands, and the rep count with an `insufficient_reps` flag when reps fall below the batch policy's declared minimum. Aggregation MUST be a pure function of the named retention root: no clock, no network, no provider, no subprocess dispatch, no Shark database access. | Epic G15, G6; feature Scope 5; research Decision 5 |
| REQ-F-008 | The aggregate MUST refuse to mark a scenario publication-eligible when its applicable-stage lineage is incomplete, its identity is incompatible, its structural results are missing, its judge calibration is missing, or its execution-oracle result is missing. F10 MUST take these verdicts verbatim from I-08 `eligibility.publication_eligible` and `eligibility.invalidity_reasons`; it MUST NOT recompute, override, soften, or add to an upstream verdict. An apparent gap in I-07/I-08 MUST be reported as an upstream contract defect, not worked around. | Epic G14, G15; feature Acceptance boundary 3; interaction-map boundary rules; research Decision 4 |
| REQ-F-009 | F10 MUST render a **lifecycle headline view** as the primary product result and a **stage diagnostic view** as an explicitly subordinate view derived from the same retained records. The stage view MUST refuse to render without the headline eligibility verdict for the same scenario and MUST carry that verdict in its header, so that a stage view can never circulate as a standalone product baseline. | Feature Scope 4; interaction-map boundary rules |
| REQ-F-010 | Reports MUST present lifecycle wall time and provider cost twice: once partitioned by the eight I-05 `stage_category` values, and once partitioned by the six I-05 `interval_category` values. Each partition MUST reconcile to total lifecycle wall time, and any residual MUST be printed as an explicit `unattributed` line even when it is zero. F10 MUST NOT introduce, rename, or extend either vocabulary. | Epic G16; feature Scope 6; UAT-16 |
| REQ-F-011 | Reports MUST show pre-code, review, rework, wait, and shipping shares as a single non-overlapping partition of lifecycle wall time, computed over disjoint (`stage_category`, `interval_category`) cells: `wait` is every cell whose interval category is `queue_or_claim_wait`, `replay_or_human_gate_wait`, or `retry_or_backoff`; among remaining cells, `pre_code` is stage category `discovery`/`specification`/`planning`, `review` is `review`/`qa`/`uat`, `shipping` is `shipping`, `rework` is stage category `code` on a stage whose retained I-07 `/stages[]/rework` boolean is `true`, and `first_pass_code` is the remaining `code` cells. F10 MUST read that per-stage flag as retained and MUST NOT infer rework from gate rounds, dispatch ordinals, or status re-entry. The six shares MUST sum to lifecycle wall time; any unmappable cell MUST appear in the `unattributed` line. The count of stages contributing to the `rework` share MUST reconcile against the I-08 `metrics.rework` rollup, and a mismatch MUST be reported as an upstream contract defect under REQ-F-008 rather than resolved locally. | Feature Scope 6; epic G16; architecture workflow value-attribution contract; `bench/runs/i07-schema.yaml` `/stages[]/rework` |
| REQ-F-012 | Per review gate, reports MUST show emitted, unique, duplicate, recurrent, confirmed, unconfirmed, and downstream-escape finding counts broken out by severity and defect class, together with elapsed time, provider cost, and resolution cost. Every value MUST be read from I-08 `review_findings` and `metrics`. When I-08 reports no seeded truth set, the report MUST print `precision: unavailable` and `recall: unavailable` and MUST NOT print `0`, an empty value, or an inferred substitute. A gate that reported zero findings MUST render distinctly from a gate whose collection failed. | Epic G17; feature Scope 7, Acceptance boundary 5; UAT-17 |
| REQ-F-013 | Reports MUST show artifact production, downstream consumption, reuse, and orphan counts, and the replayed product-design interaction proxies (request/response counts, payload size, revision count, unresolved gates), from the I-05/I-06 lineage carried in I-08 `metrics.artifact_use`. Every replay proxy MUST be labelled as a replayed proxy. No report field, heading, unit, or caption may express a replay proxy as observed human effort, human minutes, or human hours. | Epic G18; feature Scope 8; UAT-18 |
| REQ-F-014 | F10 MUST provide an operator review-comparison operation supporting both architecture modes, `independent_frozen_candidate` and `sequential_delivery`, for the feature-QA gate versus the finish-feature deep-review gate. Its preview MUST make zero provider calls and MUST print the candidate identities, the workflow-policy identities, the expected provider-call inventory, the truth-set availability, and the fix rules (whether fixes are permitted between gates) before any spend. | Epic G17; feature Scope 9, Acceptance boundary 5 |
| REQ-F-015 | The review-comparison operation MUST delegate identity adjudication to `bench/scripts/compare-lifecycle-evaluations.sh` and MUST refuse to publish a comparison the comparator rejected, preserving the comparator's divergence reasons verbatim. F10 MUST NOT re-implement candidate or workflow-policy identity comparison, and MUST NOT accept a branch name or matching `HEAD` as candidate identity. | Epic G19; architecture ADR-009; research Finding 6, Decision 4 |
| REQ-F-016 | Reports MUST present quality, elapsed time, and provider cost as three separate dimensions and MUST present paired comparisons as paired deltas per dimension. F10 MUST NOT emit any composite, weighted, or blended efficiency, value, or ROI score. A paired delta that does not clear the published noise band for its metric MUST be reported as "no detectable effect", never as an improvement or regression. | Epic G6, G17; feature Scope 11; architecture ADR-010 |
| REQ-F-017 | The completed Phase 1 surfaces `bench/scripts/run-batch.sh`, `bench/scripts/aggregate-runs.sh`, and `bench/scripts/report-baseline.sh` MUST remain reachable and unmodified as low-cost regression and compatibility tooling. F10 artifacts and reports MUST carry an explicit phase label distinguishing lifecycle v2 output from v1 output, and F10's aggregator MUST refuse a Phase 1 `record.jsonl` input with a named reason rather than coercing it into a lifecycle aggregate. | Feature Contracts 3; interaction-map boundary rules; research Decision 7 |
| REQ-F-018 | Every F10 refusal, invalidity, and unattributed condition MUST use a closed, schema-owned vocabulary carried in a committed F10 schema file, so that operator tooling and tests read the vocabulary rather than maintaining private copies. | Architecture ADR-008; F05/F06/F08/F09 schema-ownership precedent |

### Non-functional requirements

| ID | Requirement | Traceability |
|---|---|---|
| REQ-NF-001 | F10 MUST add no file under `internal/` or `cmd/`, no Shark database table, migration, workflow engine, claim store, Question store, prompt assembler, or product service. Its artifacts are file-backed under the operator's declared retention root. | Epic constraints; architecture ADR-002, ADR-006 |
| REQ-NF-002 | Preview, aggregate, report, verify, and pilot-ledger commands MUST make zero provider calls, zero network calls, zero writes to the live Shark database, `.sharkconfig.json`, or the live working tree. This MUST be provable under a provider-denial and network-denial harness rather than asserted by inspection. | Epic G15; feature Acceptance boundary 1; `report-baseline.sh` purity precedent |
| REQ-NF-003 | Reports and aggregates MUST be deterministic pure functions of their declared inputs: same retained root in, byte-identical aggregate and report out. They MUST consult no clock and invoke no subprocess. Timestamps that appear in output MUST originate from the retained records. | Epic G7; `report-baseline.sh` and `aggregate-runs.sh` header contracts |
| REQ-NF-004 | Retention, aggregation, and reporting MUST stream retained JSONL and MUST NOT load transcripts, evidence payloads, or evaluator artifacts into memory. Aggregation over the committed synthetic 100 MB retention fixture MUST complete within 60 seconds on the repository CI runner. | Epic G15; F09 REQ-NF-004 precedent |
| REQ-NF-005 | No credential, rendered prompt body, evaluator-only path content, or unbounded transcript may be copied into an aggregate or a report. Bounded paths, sizes, digests, counts, and bounded excerpts are permitted. | Epic G9; architecture ADR-007 |
| REQ-NF-006 | Generic F10 code MUST remain fixture-language-neutral. Any language-specific behavior MUST be reached through the I-04 adapter already registered by the scenario package; F10 MUST add no Python- or Go-specific branch. | Epic G8; F05 adapter boundary |
| REQ-NF-007 | Retention MUST be append-and-verify: F10 MUST NOT delete, overwrite, or rewrite a previously retained scenario directory. A repeat of an already-retained (scenario, rep) MUST be classified and skipped, or quarantined under an explicit reclaim flag, following the existing `run-batch.sh` classification discipline. | `run-batch.sh` ADR-F03-07 classification precedent; feature Scope 3 |

### Acceptance criteria

| ID | Verification | Expected result |
|---|---|---|
| AC-001 | `tests/contracts/e40_f10_operator_baseline_contract_test.go#TC-078` validates `bench/reports/lifecycle-baseline-schema.yaml` and committed valid/invalid aggregate, retention-manifest, and pilot-attestation fixtures. | Required fields, retention layout, refusal-reason vocabulary, noise-band rule names, share-partition cells, and view names are schema-owned; a malformed fixture fails with the failing path named. |
| AC-002 | `tc079_operator_preview_zero_spend_test.sh` runs `run-lifecycle-batch.sh --mode preview` and `run-review-comparison.sh --mode preview` under a PATH shim that fails any provider or network invocation. | Both previews exit 0, make zero denied invocations, and print the scenario matrix, applicable stages, planned provider-call inventory, retention root, ceilings, pilot-ledger state, and (for the comparison) candidates, policies, truth-set availability, and fix rules. |
| AC-003 | `tc080_spend_gate_refusal_test.sh` invokes both provider-backed drivers with, one at a time: no acknowledgement flag; acknowledgement plus a zero, negative, and absent value for each of the three ceilings; no `--retention-root`; and an acknowledgement supplied only through an environment variable or config file. | Every case refuses with the refusal exit status and a schema-owned refusal reason, before any subprocess, checkout, or Shark call; only the fully-satisfied invocation proceeds. |
| AC-004 | `tc081_pilot_ledger_gate_test.sh` runs `--mode baseline` against a matrix whose families have: no attestation; an attestation with a stale artifact digest; and a verified current attestation. | The first two refuse naming the family; only the third proceeds. `pilot-ledger.sh --record` produces an attestation that `--verify` accepts and that a mutated retained artifact then invalidates. |
| AC-005 | `tc082_retention_layout_test.sh` retains one scenario, then runs `verify-retention-root.sh` over complete, missing-artifact, digest-mismatched, and re-serialized-artifact roots. | A complete root verifies; each damaged root fails naming the artifact and reason. Retained artifacts are byte-identical to their sources. A repeat of a retained (scenario, rep) is classified and skipped, and quarantined only under the explicit reclaim flag. |
| AC-006 | `tc083_failed_oracle_diagnosis_test.sh` retains a fixture whose I-07 terminal outcome is a completed workflow and whose I-08 `execution_oracle` result is `fail`, then diagnoses it offline from the retention root only. | The oracle failure, gate lineage, per-stage evidence, and transcripts are all reachable from the scenario directory; the headline reports the workflow as not correct; no provider or scenario rerun occurs. |
| AC-007 | `tc084_time_reconciliation_test.sh` aggregates a scenario carrying provider work, tool/test time, a replayed gate, retry time, wait time, and unclassified time, then renders both report views. | Stage-category and interval-category partitions each reconcile to lifecycle wall time; the `unattributed` line is printed even at zero; the six shares of REQ-F-011 sum to lifecycle wall time with no cell counted twice; an injected unmappable cell appears in `unattributed` rather than being absorbed. |
| AC-008 | `tc085_review_value_report_test.sh` reports I-08 fixtures containing findings, an explicit zero-finding gate, a collection-failure gate, duplicates, recurrences, confirmed and unconfirmed findings, a downstream escape, a seeded truth set, and no truth set. | All seven finding measures render by severity and defect class. Per-gate elapsed/provider/resolution cost render verbatim when the I-08 contract supplies them; until the named upstream extension is activated, the report emits the schema-owned upstream-contract-gap reason. Zero-findings and collection-failure render distinctly; precision and recall render as `unavailable` without a truth set and never as `0`. |
| AC-009 | `tc086_artifact_use_and_replay_proxy_test.sh` reports a fixture with one consumed artifact, one orphan, and replayed D01-D05 proxies, and runs a static scan of every F10 script and template. | Consumed and orphaned artifacts are distinguished; replay proxies render with counts, sizes, revisions, and unresolved gates under a replayed-proxy label; the static scan finds no human-minute, human-hour, or human-effort framing of any proxy field. |
| AC-010 | `tc087_review_comparison_operator_test.sh` runs the comparison operation in both modes against identity-compatible and one-field-divergent retained I-08 pairs. | The comparator's accept/reject verdict and divergence reasons are preserved verbatim; a rejected pair is never published; a branch-name-only or `HEAD`-only match is rejected; independent and sequential modes render distinct candidate lineage. |
| AC-011 | `tc088_dimension_separation_and_noise_band_test.sh` renders paired deltas where one metric clears its band and another does not, plus a matrix below the declared minimum reps, and statically scans report output fields. | Quality, time, and cost render as three separate blocks. Each dimension renders its upstream paired delta and noise-band classification when supplied; before the named I-08 extension, time/cost render the schema-owned upstream-contract-gap reason. The sub-band delta renders as "no detectable effect"; the low-rep aggregate carries `insufficient_reps`; no composite efficiency, value, or ROI field exists in schema or output. |
| AC-012 | `tc089_phase_separation_test.sh` feeds a Phase 1 `record.jsonl` to `aggregate-lifecycle.sh`, runs the unmodified `run-batch.sh`/`aggregate-runs.sh`/`report-baseline.sh` path, and diffs those three files against `HEAD`. | The lifecycle aggregator refuses the v1 record with a named reason; the Phase 1 path still runs and is byte-unmodified; every F10 artifact and report carries the lifecycle-v2 phase label and no v1 output is relabelled. |
| AC-013 | `tc090_offline_determinism_and_scale_test.sh` aggregates and reports the same retention root twice under provider, network, database, and live-tree denial, including the committed 100 MB retention fixture. | Both runs produce byte-identical aggregate and report output; zero denied calls occur; the 100 MB run meets REQ-NF-004; peak memory shows the retained payloads were streamed. |
| AC-014 | `tc091_static_safety_language_neutrality_test.sh` statically scans all F10 scripts and the schema. | No write outside the declared retention root; no `internal/`, `cmd/`, migration, or `.sharkconfig.json` change; no fixture-language branch; no credential, prompt body, evaluator-only content, or unbounded transcript copied into an aggregate or report. |
| AC-015 | `make fmt && make lint && make test` and `bench/scripts/tests/run-all.sh` with TC-078 through TC-092 registered. | Repository quality gates and the complete F10 suite pass; no existing F01-F09 test is removed, skipped, or weakened (`tc092_full-regression-registration_test.sh`). |

### Out of scope for this feature

- A hosted dashboard, scheduled service, CI-triggered provider spend, or any always-on operator surface.
- Publishing an aggregate from a truncated, incompatible, or incompletely evaluated scenario, or overriding an I-08 eligibility verdict.
- Expanding the scenario corpus beyond the admitted v2 families, or admitting a new scenario (E40-F05).
- Recomputing evaluation truth, adjudicating findings, executing the held-back oracle, deriving comparison identity, or capturing stage evidence (E40-F06, E40-F08, E40-F09).
- The PR-comment, CI, merge, and branch-cleanup phases of finish-feature; F10 compares its deep-review gate only.
- Modifying `bench/scripts/run-batch.sh`, `aggregate-runs.sh`, or `report-baseline.sh`, or retrofitting Phase 1 records into the lifecycle aggregate.
- Statistical inference beyond the published noise band, and any causal claim derived from observational gate order alone.

## Architecture

### Component changes

| Path | Change |
|---|---|
| `bench/reports/lifecycle-baseline-schema.yaml` | **New**, in a **new** `bench/reports/` directory. Machine-readable owner of the F10 schema version, retention-layout inventory, refusal-reason vocabulary, invalidity-inventory shape, aggregate field inventory, noise-band derivation-rule names, the REQ-F-011 share-partition cell map, report view names (`headline`, `stage_diagnostic`), phase label values, and digest rules. It **references** `bench/evidence/i05-schema.yaml` for `stage_category` and `interval_category` and `bench/evaluation/i08-schema.yaml` for `invalidity_reason`; it does not restate or extend either vocabulary. |
| `bench/scripts/lib/spend-gate.sh` | **New** sourced library. Implements the acknowledgement check, ceiling presence/positivity pre-check, `--retention-root` presence check, and pilot-ledger check, and emits one schema-owned refusal reason plus the refusal exit status. It is the single owner of refusal semantics for both provider-backed drivers. |
| `bench/scripts/run-lifecycle-batch.sh` | **New** operator batch driver, modelled on `run-batch.sh`'s classification and non-aborting discipline. Accepts `--mode preview\|pilot\|baseline` (with `--dry-run` retained as an alias for `--mode preview`), enumerates the (scenario, rep) matrix, sources `spend-gate.sh`, and for non-preview modes invokes the existing `bench/scripts/run-lifecycle.sh` once per pair and `bench/scripts/evaluate-lifecycle.sh` once per completed pair, then retains their outputs. In preview mode it invokes `run-lifecycle.sh --mode dry-run` for stage resolution and dispatches nothing else. |
| `bench/scripts/run-review-comparison.sh` | **New** operator review-comparison driver. Same three modes and the same `spend-gate.sh` library. Drives the QA and finish-feature deep-review gates for a declared candidate through `run-lifecycle.sh`, then delegates the paired verdict to `bench/scripts/compare-lifecycle-evaluations.sh` and retains the accepted or rejected comparison record unchanged. |
| `bench/scripts/pilot-ledger.sh` | **New.** `--record` writes one operator pilot-inspection attestation (run reference, checklist results, operator identity, inspected-artifact digests) into `<root>/pilot-ledger.jsonl`; `--verify` re-checks the recorded digests against the retained artifacts and reports per-family pass/fail. Offline, no provider. |
| `bench/scripts/verify-retention-root.sh` | **New** offline retention validator. Checks layout completeness, per-artifact digest equality against the recorded source digests, upstream schema validity via the existing `verify-lifecycle-run.sh` and `verify-lifecycle-evaluation.sh`, and pair-level lineage. Emits a bounded verdict; it never repairs a root. |
| `bench/scripts/aggregate-lifecycle.sh` | **New** pure aggregator over one retention root, modelled on `aggregate-runs.sh`. Emits `aggregate.json` with batch provenance, per-scenario verbatim I-08 eligibility, the invalid-run inventory, per-(scenario, metric) noise bands with the `insufficient_reps` flag, stage-category and interval-category rollups, the REQ-F-011 share partition, review-value rollups, and artifact-use rollups. Refuses a Phase 1 `record.jsonl` input with a named reason. |
| `bench/scripts/report-lifecycle.sh` | **New** pure renderer. `--aggregate <aggregate.json> --view headline\|stage_diagnostic` prints one markdown document to stdout. The stage view refuses to render without the headline eligibility verdict and prints it in its header. Every printed value traces to a field already present in the aggregate; this script computes no new statistic. |
| `bench/scripts/tests/tc079_*.sh` through `bench/scripts/tests/tc092_*.sh` | **New** offline shell contract fixtures for AC-002 through AC-015, using stubs only at declared provider/network seams. |
| `tests/contracts/e40_f10_operator_baseline_contract_test.go` | **New** test-only Go contract validator, `package contracts`, `TC-078`; reads the F10 schema and committed valid/invalid fixtures only. |
| `tests/contracts/testdata/e40_f10/{valid,invalid}/` | **New** static fixtures for the aggregate, retention manifest, pilot attestation, share partition, and refusal-reason cases. |
| `bench/scripts/tests/run-all.sh` | **Modified only** to register TC-079 through TC-092 in deterministic order. |
| `bench/README.md` | **Modified** with the operator command set, refusal behavior, retention layout, pilot inspection checklist, publication gate, report views, and phase-label rules. |

**No file under `internal/` or `cmd/` is modified.** The following are read-only
inputs, invoked but never changed: `bench/scripts/run-lifecycle.sh`,
`bench/scripts/evaluate-lifecycle.sh`,
`bench/scripts/compare-lifecycle-evaluations.sh`,
`bench/scripts/verify-lifecycle-run.sh`,
`bench/scripts/verify-lifecycle-evaluation.sh`,
`bench/scripts/verify-stage-evidence.sh`, `bench/evidence/i05-schema.yaml`,
`bench/runs/i07-schema.yaml`, `bench/evaluation/i08-schema.yaml`, and the
admitted I-04 scenario packages. `bench/scripts/run-batch.sh`,
`bench/scripts/aggregate-runs.sh`, and `bench/scripts/report-baseline.sh` are
neither invoked by nor modified by F10 (REQ-F-017).

### Data model changes

There is no Shark schema, table, or migration (REQ-NF-001). F10 writes only
under the operator-declared retention root (`--retention-root`):

```
<retention_root>/
  batch.json                        # batch id, phase label, policy digest, ceilings,
                                    #   acknowledgement record, matrix, provenance
  pilot-ledger.jsonl                # one attestation per scenario family
  scenarios/<scenario_id>/<rep>/
    package.yaml                    # I-04 admitted package, byte-preserved
    evidence/                       # I-05 stage evidence bundle, byte-preserved
    transcripts/                    # stage transcripts, byte-preserved
    entity-history.json             # scratch-project entity history export
    lifecycle.jsonl                 # I-07 record, byte-preserved
    evaluation.jsonl                # I-08 record, byte-preserved
    oracle.json                     # held-back oracle result, byte-preserved
    comparison.json                 # optional review-comparison record, byte-preserved
    manifest.json                   # source paths + sha256 for every file above
  invalid/index.jsonl               # one row per ineligible run: scenario, rep,
                                    #   verbatim I-08 invalidity_reasons, retention path
  aggregate.json                    # machine-readable lifecycle aggregate
  reports/headline.md
  reports/stage-diagnostic.md
```

`aggregate.json` blocks:

| Block | Required contents |
|---|---|
| `identity` | F10 schema version, batch id, phase label (`lifecycle_v2`), retention-root digest, batch-policy digest, ceilings, acknowledgement record reference, and the declared minimum reps. |
| `scenarios[]` | Scenario id/version/family, rep, retention path, verbatim I-08 `eligibility` (`aggregate_eligible`, `publication_eligible`, `invalidity_reasons`), verbatim I-07 `outcome`, and source-artifact digests. |
| `time` | Per scenario and rolled up: lifecycle wall time; the eight-cell `stage_category` partition; the six-cell `interval_category` partition; the REQ-F-011 six-share partition; and the `unattributed` residual for each partition. |
| `cost` | The same three partitions expressed in provider cost, plus observed-versus-ceiling consumption from I-07 `limits`. |
| `quality` | Verbatim I-08 `structural`, `judge`, and `execution_oracle` verdicts per scenario; first-pass yield derived from the retained gate rounds. |
| `review_value` | Per gate: emitted, unique, duplicate, recurrent, confirmed, unconfirmed, downstream-escape by severity and defect class; elapsed, provider, and resolution cost; `truth_set` availability; precision/recall only when available. |
| `artifact_use` | Produced, consumed, reused, and orphan counts with typed producer/consumer edges, and the replayed-interaction proxy block under its replayed-proxy label. |
| `noise_bands` | Per (scenario, metric): min, median, max, spread, acceptance interval, the named derivation rule applied, rep count, and `insufficient_reps`. |
| `comparisons` | Verbatim `compare-lifecycle-evaluations.sh` verdicts and divergence reasons, with mode, candidate identity references, and policy identity references. |
| `invalid` | The invalid-run inventory, mirroring `invalid/index.jsonl`. |

Canonical digests are SHA-256 over compact sorted-key UTF-8 JSON, matching the
I-05/I-07/I-08 `digest_rules`. There is deliberately no composite score field
anywhere in the schema (REQ-F-016).

### API / interface contracts

F10 exposes file-backed shell interfaces, not a Go API:

- `run-lifecycle-batch.sh --batch <batch-policy.yaml> --retention-root <retention_root> --mode preview|pilot|baseline [--dry-run] [--acknowledge-provider-spend] [--max-cost-usd <n>] [--max-wall-clock-seconds <n>] [--max-generated-tasks <n>] [--reps <n>] [--scenarios <id[,id...]>] [--reclaim-incomplete]`. Exit `0` success, `2` usage error, `3` spend-gate refusal, `4` batch completed with at least one failed pair.
- `run-review-comparison.sh --candidate <candidate-ref> --mode preview|pilot|baseline --comparison-mode independent_frozen_candidate|sequential_delivery --retention-root <retention_root> [--acknowledge-provider-spend] [ceiling flags]`. Same exit codes.
- `pilot-ledger.sh --retention-root <retention_root> {--record --scenario <id> --rep <n> --operator <identity> --checklist <checklist.json> | --verify [--family <f>]}`.
- `verify-retention-root.sh --retention-root <retention_root> --schema bench/reports/lifecycle-baseline-schema.yaml`.
- `aggregate-lifecycle.sh --retention-root <retention_root>` prints one `aggregate.json` document to stdout, diagnostics to stderr.
- `report-lifecycle.sh --aggregate <aggregate.json> --view headline|stage_diagnostic` prints one markdown document to stdout.

Operator flow:

```mermaid
sequenceDiagram
    participant Op as Operator
    participant Batch as run-lifecycle-batch.sh
    participant Gate as spend-gate.sh
    participant Run as run-lifecycle.sh (F08, I-07)
    participant Eval as evaluate-lifecycle.sh (F09, I-08)
    participant Root as retention root
    participant Agg as aggregate-lifecycle.sh
    participant Rep as report-lifecycle.sh
    Op->>Batch: --mode preview
    Batch->>Run: --mode dry-run (stage resolution only)
    Batch-->>Op: matrix, stages, planned calls, root, ceilings, pilot state
    Op->>Batch: --mode pilot --acknowledge-provider-spend + ceilings
    Batch->>Gate: check acknowledgement, ceilings, root
    Gate-->>Batch: proceed or refusal reason (exit 3, zero subprocesses)
    Batch->>Run: one scenario, one rep
    Run-->>Root: I-07 lifecycle record + I-05 evidence
    Batch->>Eval: evaluate retained I-05/I-07
    Eval-->>Root: I-08 evaluation record
    Op->>Root: inspect pilot; pilot-ledger.sh --record
    Op->>Batch: --mode baseline (gate now also checks pilot ledger)
    Op->>Agg: aggregate retained root
    Agg-->>Op: aggregate.json (verbatim eligibility + noise bands)
    Op->>Rep: headline view, then stage_diagnostic view
```

### Key technical decisions

**ADR-F10-01 — Extend the existing dry-run idiom; add preview content at the
batch layer.** `run-batch.sh --dry-run` and `run-lifecycle.sh --mode dry-run`
already own the zero-dispatch short-circuit, but neither prints the matrix,
stages, planned calls, root, or ceilings (research Finding 2). Adding a third
flag convention would fragment the operator surface, and pushing preview text
into `run-lifecycle.sh` would give a single-scenario runner batch-level
knowledge. The batch driver therefore owns preview content and calls
`run-lifecycle.sh --mode dry-run` for stage resolution.

**ADR-F10-02 — The acknowledgement gate is a command-line flag only.** No
acknowledgement pattern exists in `bench/scripts/` (research Finding 4), so the
design is chosen rather than inherited. A command-line flag is auditable in the
retained `batch.json`, works in non-interactive contexts, and cannot be
satisfied by ambient state. Environment variables and config defaults are
explicitly rejected because they make accidental spend a one-time setup mistake
rather than a per-invocation decision. An interactive prompt is rejected because
the harness must run unattended.

**ADR-F10-03 — Delegate ceiling positivity, re-check presence early.**
`run-lifecycle.sh` `parse_limits()` already rejects non-positive ceilings for
every mode (research Finding 3). Re-implementing that check would create two
divergent validators. The batch driver re-checks only presence and positivity
before dispatch so that a refusal costs zero subprocesses, and `parse_limits()`
remains the single authority for the per-run semantics.

**ADR-F10-04 — I-05, I-07, and I-08 are strictly read-only.** The interaction
map's boundary rules give F09 ownership of aggregate eligibility, finding
normalization, and comparison identity, and permit F10 only to format and
publish. F10 copies `eligibility`, `comparison`, `review_findings`, and
`metrics` verbatim. Any apparent gap is reported as an upstream contract defect
(research Decision 4). This is what keeps the published baseline auditable back
to a single producer.

**ADR-F10-05 — Byte-preserving retention, never re-serialization.** The
retention root stores upstream artifacts unchanged with a digest manifest
(research Decision 6). Re-serializing would silently create a second, divergent
copy of an I-07/I-08 record and destroy the ability to re-verify a published
baseline with the producers' own validators.

**ADR-F10-06 — Reports are pure functions of one aggregate.** This follows
`report-baseline.sh`'s stated contract and `aggregate-runs.sh`'s pure-function
discipline. Purity is what makes a published baseline reproducible from the
retention root alone, which is what G7-style replay verification checks.

**ADR-F10-07 — The stage diagnostic view is structurally subordinate.** The
interaction map states a stage view is never a separate product baseline. Making
the stage renderer refuse without the headline verdict, and printing that
verdict in its header, enforces the rule structurally rather than by convention,
so a stage report cannot be circulated on its own.

**ADR-F10-08 — Shares are a partition over (stage_category, interval_category)
cells.** "Without double counting" cannot be satisfied by five overlapping
categories, because rework is a property of a `code` stage while wait is a
property of an interval. Defining the shares as disjoint cells of the two
existing closed vocabularies (REQ-F-011) makes the partition exhaustive,
non-overlapping, and reconcilable to lifecycle wall time, and it adds no new
vocabulary that F06 would have to accept. The partition is computable entirely
from retained fields: `bench/runs/i07-schema.yaml` already makes
`/stages[]/rework` a required per-stage boolean, so the one cell that is not a
pure vocabulary lookup is still read rather than derived. Inferring rework from
gate rounds or status re-entry is explicitly rejected because that would be F10
adjudicating a fact I-07 already asserts, which ADR-F10-04 forbids; I-08's
`metrics.rework` rollup is used only as a reconciliation cross-check.

**ADR-F10-09 — The pilot gate is an artifact-digest attestation, not a flag.**
"One inspected pilot per family" is only meaningful if the inspection is tied to
the evidence inspected. Recording the inspected-artifact digests means a later
change to the retained evidence invalidates the attestation, which a boolean
flag or a run-id reference alone could not detect.

**ADR-F10-10 — Absent measures render as `unavailable`, never as zero.** ADR-010
separates emitted findings from confirmed value; printing `0` for precision when
no truth set exists would present an absence of evidence as evidence of a value.
The same rule applies to a zero-finding gate versus a collection-failure gate,
which must render distinctly (feature Acceptance boundary 5).

**ADR-F10-11 — Phase labels, not shared aggregators.** Rather than teaching
`aggregate-runs.sh` about lifecycle records or teaching F10 about v1 records,
each phase keeps its own aggregator and both label their output. The lifecycle
aggregator refuses a v1 `record.jsonl` with a named reason, which makes an
accidental relabel a loud failure instead of a silent one (REQ-F-017).

**ADR-F10-12 — The noise-band minimum rep count is operator policy, not
architecture.** F10 fixes the *mechanism* — the band carries its rep count, its
named derivation rule, and an `insufficient_reps` flag against the declared
minimum in the batch policy, reusing `aggregate-runs.sh`'s existing flag
semantics — but not the number. Provider-backed lifecycle scenarios cost far
more per rep than v1 tasks, so the right minimum is a cost decision the operator
makes per batch and the aggregate then records; hard-coding one would either
over-spend or publish an unsupportable band.

### Integration with existing code

- `bench/scripts/run-lifecycle.sh` is invoked once per (scenario, rep) with its
  existing signature `--scenario <package.yaml> --run-id <id> --root <key>
  --scratch-root <dir> [--output <lifecycle.jsonl>] [--limits <policy.yaml>]
  [--mode contract|dry-run]`, and with `--mode dry-run` in preview. Its
  `--root` is the root *entity key*; F10's own retention directory is a
  separate `--retention-root` flag, deliberately named to avoid the collision.
  Its `parse_limits()` remains the authority
  for ceiling semantics and its `adapter_result()` offline short-circuit remains
  the authority for zero-dispatch preview. F10 changes neither.
- `bench/scripts/evaluate-lifecycle.sh` is invoked with `--i05`, `--i07`,
  `--scenario`, `--output`, and the operator's `--judge-result` /
  `--review-findings` inputs. F10 retains its output verbatim and reads
  `eligibility`, `comparison`, `review_findings`, and `metrics` without
  modification.
- `bench/scripts/compare-lifecycle-evaluations.sh` is invoked with `--left`,
  `--right`, `--mode`, and `--output`. Its `MODES`, `CANDIDATE_FIELDS`, and
  `POLICY_DIGEST_FIELDS` remain the single identity authority; F10 preserves its
  accept/reject verdict and divergence reasons verbatim (REQ-F-015).
- `bench/scripts/verify-lifecycle-run.sh` and
  `bench/scripts/verify-lifecycle-evaluation.sh` are invoked by
  `verify-retention-root.sh` for upstream schema validity. F10 duplicates
  neither validator.
- `bench/evidence/i05-schema.yaml` supplies `stage_category` (eight values),
  `interval_category` (six values), `artifact_type`, `edge_kind`, and
  `stop_outcome`. `bench/runs/i07-schema.yaml` supplies `limits`,
  `review_gates`, `outcome`, and terminal outcomes.
  `bench/evaluation/i08-schema.yaml` supplies `invalidity_reason`,
  `comparison_modes`, `truth_result`, and `oracle_result`. F10's schema
  references all three and restates none.
- `bench/scripts/run-batch.sh` is the structural model for matrix enumeration,
  per-pair classification (`skipped_complete`, `incomplete_prior_attempt`,
  `quarantined_and_rerun`, `pending_run`, `failed`), non-aborting batch
  behavior, and `--reclaim-incomplete` quarantine-never-delete semantics
  (REQ-NF-007). F10 copies the discipline into a new driver; it does not modify
  or extend `run-batch.sh`.
- `bench/scripts/report-baseline.sh` and `bench/scripts/aggregate-runs.sh`
  supply the purity contract, the derivation-rule sentence discipline, and the
  `insufficient_reps` / invalid-retention semantics F10 reuses. Both remain
  byte-unmodified (AC-012).
- `bench/README.md` is the operator-facing document extended with the F10
  command set, refusal behavior, retention layout, and inspection checklist.

## Cross-feature interactions

### Consumes

- **I-05 — Stage evidence and evaluator isolation**; producer E40-F06; F10 retains the stage-evidence bundle byte-for-byte and reads its `stage_category`, `interval_category`, artifact producer/consumer, and candidate vocabularies for the time, cost, and artifact-use partitions. **Shape source:** `../architecture.md#stage-evidence-and-isolation-contract`. **Contract test pointer:** `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042`. The map-assigned gate mode is `contract-only` until E40-F10 proves live production-path use; activation owner and closure key are E40-F10 at its own UAT; counterpart status is read live from Shark at review/UAT time; review basis is F06's completed `spec.md` and the map row; disposition remains `pending-integration` until activation.
- **I-07 — Lifecycle run record**; producer E40-F08; F10 retains the record byte-for-byte and reads the entity graph, dispatch and gate lineage, stage intervals, workflow-policy identity, resource `limits` (declared and observed), and stop `outcome` for the preview, retention, headline, stage-diagnostic, and review-value views. **Shape source:** `../architecture.md#lifecycle-run-record-contract`. **Contract test pointer:** `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061`. The map-assigned gate mode is `contract-only` until E40-F10 proves live production-path use; activation owner and closure key are E40-F10 at its own UAT; counterpart status is read live from Shark at review/UAT time; review basis is F08's completed `spec.md` and the map row; disposition remains `pending-integration` until activation.
- **I-08 — Lifecycle evaluation record**; producer E40-F09; F10 retains the record byte-for-byte and reads `eligibility.aggregate_eligible`, `eligibility.publication_eligible`, `eligibility.invalidity_reasons`, `structural`, `judge`, `execution_oracle`, `candidate_snapshots`, `workflow_policy`, `comparison`, `review_findings`, and `metrics` verbatim. F10 formats and publishes these verdicts and never recomputes or weakens them. **Shape source:** `../architecture.md#lifecycle-evaluation-record-contract`. **Contract test pointer:** `tests/contracts/e40_i08_lifecycle_evaluation_contract_test.go#TC-067` (the single shared contract proof named by E40-F09; F10 reuses this pointer and creates no twin test). The map-assigned gate mode is `contract-only` until E40-F10 proves live production-path use; activation owner is E40-F10; closure key is E40-F10 at its own UAT; counterpart status is read live from Shark at review/UAT time; review basis is F09's completed `spec.md` and the map row; disposition remains `pending-integration` until F10 closes the handoff.

### Produces

F10 is the terminal consumer in the E40 interaction map. It produces no I-##
row: no E40 feature consumes its aggregate, retention root, or reports. Its
outputs are operator deliverables, not a cross-feature contract.

F10 does not declare I-04 or I-06 as direct interactions. The scenario/adapter
identity of I-04 and the product-design replay lineage of I-06 reach F10 through
the I-05, I-07, and I-08 producer contracts named above. F10 invents no I-## ID
and alters no map-assigned shape source, gate mode, activation owner, closure
key, counterpart status, or review basis.

## Cross-epic integrations

F10 produces, consumes, and validates **no X-## row**. Every row in
[E40-cross-epic-map.md](../E40-cross-epic-map.md) and
[docs/product/cross-epic-integration-map.md](../../../product/cross-epic-integration-map.md)
names a different owning feature: X-07 (E40-F02), X-08 (E40-F04), X-09
(E40-F06), X-10 (E40-F07), X-11 and X-13 (E40-F08), X-12 (E40-F09). The epic
UAT plan's interaction-coverage section likewise assigns X-10 through X-13 to
UAT-10, UAT-11, UAT-12, and UAT-14 — none of which are F10's scenarios (F10 owns
UAT-15 through UAT-18).

Three of those seams reach F10 only transitively and are deliberately not
re-declared here: X-09's audited provider-usage mapping arrives inside I-05 and
I-07 usage fields; X-11's keyed Rider loop is executed by F08 and arrives as
I-07 dispatch lineage; X-12's installed-content identity is computed by F09 and
arrives as an I-08 identity field. Re-declaring any of them at F10 would create
a second owner for a contract whose adaptation and failure behavior another
feature already owns.

## Durable unresolved decisions

No material unresolved decision remains for this specification. The three
closest calls were examined and settled without a Question record:

1. **Minimum reps for a publishable lifecycle noise band.** Settled as operator
   policy rather than architecture by ADR-F10-12: the mechanism (recorded rep
   count, named derivation rule, `insufficient_reps` flag against the batch
   policy's declared minimum) is fixed by REQ-F-007, and the number is a
   per-batch cost decision the aggregate records. Non-material because no
   contract, acceptance criterion, or sequencing depends on the value.
2. **Whether F10 owns G6's paired configuration-change delta report.** Settled
   affirmatively from existing sources: Q002 (resolved) places G6 in Phase 2,
   the epic architecture gives G6 a durable home in E40-F09 and E40-F10, E40-F09
   explicitly out-of-scopes operator reporting and publication, and this
   feature's own scope requires paired deltas and noise bands. REQ-F-016
   therefore carries the G6 "no detectable effect" rule. Non-material because it
   confirms an existing assignment rather than changing one.
3. **Executing the finish-feature deep-review gate in isolation.** Settled by
   the feature's own Out-of-scope boundary and the interaction map's boundary
   rule that finish-feature scope in lifecycle v2 stops at its controlled
   deep-review gate; the repository deep-review bundle is invocable at that gate
   without the PR, CI, merge, or cleanup phases. Non-material because it
   restates an approved boundary.

Two inherited open Questions are dependencies of upstream features, not F10
decisions, and are referenced rather than duplicated: **Q003** (unverified
provider-usage fields) is an E40-F06/X-09 obligation — if a required usage field
is absent, I-08 marks the record ineligible and F10 refuses publication under
REQ-F-008, so F10's design is unchanged by its outcome. **Q004** (cascade child
stage attribution in `shark run`) is superseded for lifecycle v2 by F08's
per-dispatch I-07 evidence and does not affect F10's partitions. F10 creates no
new Question record.

## Verification traceability

| Requirement group | Proof |
|---|---|
| REQ-F-001, REQ-F-014 | AC-002; UAT-15 and UAT-17 preview halves |
| REQ-F-002, REQ-F-003 | AC-003; epic G15 |
| REQ-F-004, REQ-F-005, REQ-F-006 | AC-004, AC-005, AC-006; UAT-15 |
| REQ-F-007, REQ-F-008 | AC-001, AC-011, AC-013; UAT-15 |
| REQ-F-009, REQ-F-010, REQ-F-011 | AC-007; UAT-16 |
| REQ-F-012, REQ-F-015 | AC-008, AC-010; UAT-17 |
| REQ-F-013 | AC-009; UAT-18 |
| REQ-F-016 | AC-011; epic G6 and G17 |
| REQ-F-017 | AC-012; interaction-map boundary rules |
| REQ-F-018 | AC-001; architecture ADR-008 |
| REQ-NF-001, REQ-NF-005, REQ-NF-006 | AC-014 |
| REQ-NF-002, REQ-NF-003 | AC-002, AC-013 |
| REQ-NF-004 | AC-013 |
| REQ-NF-007 | AC-005 |
| All | AC-015 (`make fmt && make lint && make test`, full `run-all.sh`) |
