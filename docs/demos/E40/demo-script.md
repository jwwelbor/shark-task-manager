# Run the E40 Shark Bench demo

**Target:** E40
**Prepared:** 2026-08-23
**Scope source:** live `shark get E40 --json`; all ten returned features are `completed`.
**Evidence boundary:** this is a traceable walkthrough of retained repository evidence. It is not UAT, approval, deployment proof, or a lifecycle transition.

## Goal

Use this runbook to validate and operate the complete E40 benchmark, from
corpus admission through v1 baseline publication and lifecycle-v2 comparison.
The epic has two related but separate flows:

1. **Phase 1 / v1:** admit the Go corpus, run the `shark run` matrix, aggregate
   the records, publish a baseline/noise band, and replay a stored manifest.
2. **Lifecycle v2:** admit the four-family scenario corpus, validate stage
   isolation and replay, run the canonical lifecycle, evaluate retained
   artifacts, inspect pilots, aggregate, report, and compare review policies.

Use `bench/scripts/e40-benchmark.sh` for the normal local operator path. It
prepares the scratch Shark project, seeded roots, batch policy, manifests,
retention paths, aggregation, and comparison report. Use the component
commands later in this runbook when you need to inspect one F01–F10 boundary.

## Prerequisites

Run from the repository root with the required local toolchain, initialized
fixture submodules, and a working `shark` binary. Keep all generated output in
explicit temporary or retention roots. Do not use the live Shark database as a
benchmark scratch project.

```bash
git submodule update --init
```

Before provider-backed work, choose a private output root and confirm the
scenario package's `resource_policy` and the batch policy's positive ceilings.
Provider-backed commands also require the literal
`--acknowledge-provider-spend` flag; an environment variable or prior
acknowledgement is not sufficient.

## Run the local baseline and prompt-variant path

### 1. Prepare and inspect the no-spend inputs

Build Shark and create an operator root outside the repository:

```bash
make shark

E40_ROOT=/tmp/e40-demo
bench/scripts/e40-benchmark.sh setup --out "$E40_ROOT"
bench/scripts/e40-benchmark.sh preflight \
  --config "$E40_ROOT/e40-demo.yaml"
```

Open these files before any provider-backed command:

- `$E40_ROOT/setup-result.json`
- `$E40_ROOT/e40-demo.yaml`
- `$E40_ROOT/preflight/baseline/batch-policy.yaml`
- `$E40_ROOT/preflight/baseline/preflight-result.json`
- `$E40_ROOT/preflight/baseline/preview.txt`

Show the four seeded families and the recorded Shark, content, prompt,
workflow, gate, scenario, fixture, adapter, toolchain, and installed
provider/model/effort route identities. Confirm that `provider_calls` and
`live_database_mutations` are both `0`.

Point out that preflight uses pinned repository helpers and the setup-owned
Shark digest. Environment-selected batch drivers and config-selected provider
commands are stripped, while a replaced `shark_binary` path or digest is
refused without executing it.

The generated config is not a successful provider fixture. It names two
missing production inputs: an executable F08 lifecycle adapter and run-matched
I-05 evidence for every scenario. The current repository does not wire those
inputs into a repeatable provider caller chain. If preview records
`pass_with_dry_run_limitations`, show the named F08 artifact-production
limitation; do not relabel it as a fully resolved dry run.

### 2. Supply real runtime inputs

Continue only when an authorized runtime owner supplies both inputs. Set
`runtime.lifecycle_adapter` and every
`scenario_roots.<scenario-id>.i05_bundle_dir` in
`$E40_ROOT/e40-demo.yaml`. An existing directory does not prove that its I-05
records match the I-07 run. F09 checks the run and dispatch identities and
rejects reused or mixed evidence.

The top-level operator always enters provider-backed work through the pinned
repository `run-lifecycle-batch.sh`; an environment-selected replacement
cannot bypass F10's acknowledgement and cap enforcement.

Do not invent an adapter, evidence bundle, provider credential, endpoint, or
result for this demo. If the inputs remain unavailable, stop the
provider-backed portion here. The completed offline setup and preflight are
the truthful demo result.

### 3. Pilot and promote the baseline

Use the same run ID for the pilot and baseline. F10 stores pilot attestations
in that retention root:

```bash
BASELINE_ID=baseline-prompt-v1

bench/scripts/e40-benchmark.sh pilot \
  --config "$E40_ROOT/e40-demo.yaml" \
  --run-id "$BASELINE_ID" \
  --reps 1 \
  --acknowledge-provider-spend \
  --max-cost-usd 5 \
  --max-wall-clock-seconds 900 \
  --max-generated-tasks 20
```

Inspect one retained pair from each selected family. Record each family
attestation with `pilot-ledger.sh --record`, then run:

```bash
BASELINE_ROOT="$E40_ROOT/runs/$BASELINE_ID"
bench/scripts/pilot-ledger.sh --retention-root "$BASELINE_ROOT" --verify
```

After all selected families pass verification, promote the same root:

```bash
bench/scripts/e40-benchmark.sh baseline \
  --config "$E40_ROOT/e40-demo.yaml" \
  --run-id "$BASELINE_ID" \
  --reps 3 \
  --acknowledge-provider-spend \
  --max-cost-usd 15 \
  --max-wall-clock-seconds 2700 \
  --max-generated-tasks 60
```

Show `pilot-benchmark-manifest.json` and `benchmark-manifest.json` as separate
identities. Then inspect `aggregate.json`, `reports/headline.md`,
`retention-verification.jsonl`, and the retained scenario directories. A
completed Shark status is not publication evidence.

### 4. Validate, pilot, and promote the prompt variant

Prepare a versioned prompt directory. Validate the copied overlay and inspect
both its source and installed digests:

```bash
bench/scripts/e40-benchmark.sh validate-variant \
  --config "$E40_ROOT/e40-demo.yaml" \
  --variant-name prompt-v2 \
  --prompt-root /path/to/prompt-v2

python3 -m json.tool \
  "$E40_ROOT/variants/prompt-v2/variant-definition.json"
```

For a provider, model, or effort variant, edit a copied workflow bundle and
pass `--workflow-root`. The operator derives the changed axes from the actual
installed routes; it does not accept a free-form model label.

Run the variant pilot, inspect and attest every selected family under the
variant root, then promote that same run ID:

```bash
VARIANT_ID=prompt-v2-run

bench/scripts/e40-benchmark.sh pilot \
  --config "$E40_ROOT/e40-demo.yaml" \
  --variant-name prompt-v2 \
  --run-id "$VARIANT_ID" \
  --reps 1 \
  --acknowledge-provider-spend \
  --max-cost-usd 5 \
  --max-wall-clock-seconds 900 \
  --max-generated-tasks 20

# Inspect, record, and verify pilot attestations in
# "$E40_ROOT/runs/$VARIANT_ID" before promotion.

bench/scripts/e40-benchmark.sh variant \
  --config "$E40_ROOT/e40-demo.yaml" \
  --variant-name prompt-v2 \
  --run-id "$VARIANT_ID" \
  --reps 3 \
  --acknowledge-provider-spend \
  --max-cost-usd 15 \
  --max-wall-clock-seconds 2700 \
  --max-generated-tasks 60
```

### 5. Compare and interpret the retained pair

Render both comparison formats:

```bash
bench/scripts/e40-benchmark.sh compare \
  --baseline "$E40_ROOT/runs/$BASELINE_ID" \
  --variant "$E40_ROOT/runs/$VARIANT_ID" \
  --out "$E40_ROOT/comparisons/prompt-v2"
```

Open `comparison.json` first, then `comparison.md`. Show the retained raw
evidence links, authorized prompt-only boundary, held-back oracle result,
pass rate, invalid and incomplete rows, per-scenario noise-band result, time
partitions, cost, review-gate findings, artifact use, and replayed-interaction
proxies. The proxy values are not observed human time. Tokens, generated LOC,
and changed paths remain `unavailable` when F10 exposes only their digests;
the report never converts missing data to zero.

The command re-derives each aggregate from its retained root with the pinned
F10 aggregator before publication. A copied or edited aggregate, mismatched
batch/policy identity, or changed retained source fails closed rather than
being accepted because its JSON shape looks valid. Open the matching
`batch-authorities/<batch-id>.json` to show the manifest digest, acknowledged
attempt digest, and complete batch authority captured after the driver exits.
Authority records and their recorded attempt paths are read through validated
directory descriptors without following symlinks. A preplaced authority
symlink prevents publication eligibility, and a comparison refuses any
attempt path redirected through a symlinked parent.
The same stable, no-follow read boundary covers each retained manifest,
aggregate, batch identity, cached JSON/report, and completion marker, including
replacement of a final file after its parent directory was opened.
The lifecycle, I-05, and I-07 schema paths are repository-pinned during this
derivation; environment overrides do not change the result.
The retained manifests must be regular files, not symlinks. Their execution
identity binds the adapter bytes, provider-command digest, and every selected
I-05 evidence-tree digest.
Each scenario is additionally pinned to the setup-owned scratch template (or
its validated variant derivative); a live-repository or symlinked scratch path
is refused before the provider driver can copy it.

Quality controls the interpretation. A faster or cheaper variant that fails
the held-back oracle is a regression, not a better result. The report does not
combine quality, time, and cost into one score.

### 6. Resume without overwriting evidence

Repeat a completed command with the same run ID to reuse verified retained
pairs. The operator appends an attempt record and refuses an immutable
identity change. It re-digests the config, lifecycle adapter, provider command,
and selected I-05 trees before calling the provider driver, so mutated resume
inputs fail before spend. Add `--retry-incomplete` only after you inspect the partial
pair; F10 moves that pair into its quarantine area before rerunning it.

Repeat `compare` with the same pair and output root to get
`already_compared`. If either manifest or aggregate digest changed, the
command refuses the existing output instead of overwriting it. It also
recomputes the full comparison and Markdown report, so a hand-edited or
symlinked cached result cannot be accepted by copying the four input digests.
Both cache inspection and publication are serialized by one output lock, and
`comparison.complete.json` is written last. An exact interrupted partial can
be recovered; mismatched partial content fails closed.
The same parent-chain guard rejects symlinked retained directories such as
`operator-attempts/` and `reports/` before an external target can be written.
The guard uses no-follow directory descriptors for atomic publication, so a
parent swapped to a symlink between validation and write is also refused.
Setup/variant staging and execution/comparison locks use the same held-parent
discipline, including descriptor-relative PID ownership and final rename.
For provider execution, the manifest and policy stay anchored to the held run
descriptor, the driver receives the policy through that descriptor, and both
layers revalidate the lexical run-root inode around delegated work. Replacing
the named run directory is a bounded refusal and cannot redirect retained
writes into the replacement.
Recorded retention roots remain lexical authority: even a self-digested batch
that names a symlink alias to the real root fails comparison.

## Inspect the full E40 component flow

### 1. Validate the v1 corpus and fixture

Run the curator sequence whenever `bench/corpus/corpus.yaml` or the pinned Go
fixture changes:

```bash
bench/scripts/admit.sh bench/corpus/corpus.yaml
bench/scripts/checkout-fixture.sh <fixture.base_sha> /tmp/e40-v1-fixture
bench/scripts/build-ledgers.sh /tmp/e40-v1-fixture /tmp/e40-v1-ledgers
bench/scripts/diff-ledgers.sh --toolchain-guard --base=/tmp/e40-v1-ledgers/tests.json
bench/scripts/verify-clean-checkout.sh /tmp/e40-v1-fixture bench/corpus/corpus.yaml
```

`admit.sh` must admit every entry in `items:` and must not admit
`negative_items:`. The ledgers pin the base-SHA test and lint state used by
later run records.

### 2. Run and aggregate the v1 baseline

First preview the matrix without dispatching provider work:

```bash
bench/scripts/run-batch.sh --out /tmp/e40-v1-runs --reps 3 --timeout 900 --dry-run
```

Then run the default Phase 1 matrix. `run-batch.sh` enumerates admitted items
× variant × repetitions, provisions an isolated scratch project per pair, and
retains one `record.jsonl` per completed pair:

```bash
bench/scripts/run-batch.sh --out /tmp/e40-v1-runs --reps 3 --timeout 900
```

Aggregate the records and render the baseline report:

```bash
bench/scripts/aggregate-runs.sh --root /tmp/e40-v1-runs --variant default --reps 3 \
  > /tmp/e40-v1-aggregate.json
bench/scripts/report-baseline.sh --aggregate /tmp/e40-v1-aggregate.json \
  > /tmp/e40-v1-baseline.md
```

The baseline is established by the aggregate's per-item, per-metric observed
spread and acceptance intervals. Inspect `baseline_id`, provenance, excluded
or anomalous records, and the noise-band derivation before using it to judge a
configuration change. A timeout or unexplained missing metric does not become
a passing baseline measurement.

### 3. Replay one v1 result

Choose a retained record and replay it against the published aggregate:

```bash
bench/scripts/replay-manifest.sh \
  --record /tmp/e40-v1-runs/<item>/<variant>/rep-<n>/record.jsonl \
  --band /tmp/e40-v1-aggregate.json \
  --out /tmp/e40-v1-replay
```

The replay must pin the stored fixture SHA, corpus schema, variant, model
identity, and repetition. It returns `invalid` for identity drift, `fail` for
an in-band identity match whose metrics leave the published interval, and
`pass` only when both identity and metrics match.

### 4. Validate the lifecycle-v2 scenario corpus

Validate the controlled Python fixture, then admit each registered scenario:

```bash
bench/scripts/verify-fixture-py-base.sh 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9
for package in \
  bench/scenarios/packages/py-bug-due-date-boundary/package.yaml \
  bench/scenarios/packages/py-change-priority-scale/package.yaml \
  bench/scenarios/packages/py-techdebt-consolidate-validation/package.yaml \
  bench/scenarios/packages/py-feature-recurring-tasks/package.yaml
do
  bench/scripts/admit-scenario.sh "$package"
done
```

Admission proves the base fixture is runnable, the package's P2P selection is
green, the family stage matrix is valid, the final predicate is false at base,
the reference outcome is true, and all resource limits are positive. A
feature scenario alone uses D01–D05; the bug, change-card, and tech-debt
scenarios record those stages as non-applicable.

### 5. Test the full offline and contract surface

Run the registered suite. It covers F01 through F10's contract, isolation,
replay, lifecycle, evaluation, retention, reporting, and safety cases:

```bash
bench/scripts/tests/run-all.sh
```

For a focused walkthrough, the major slices are:

```bash
# F01–F04: corpus, run driver, baseline, and liveness
bench/scripts/tests/tc003_clean_checkout_test.sh
bench/scripts/tests/tc004_admit_full_set_test.sh
bench/scripts/tests/tc014_run_one_smoke_test.sh
bench/scripts/tests/tc018_aggregate_report_test.sh
bench/scripts/tests/tc019_replay_manifest_test.sh

# F05–F08: scenario admission, isolation, replay, and canonical lifecycle
bench/scripts/tests/tc031_adapter_conformance_test.sh
bench/scripts/tests/tc043_root_policy_isolation_test.sh
bench/scripts/tests/tc053_live_egress_denial_test.sh
bench/scripts/tests/tc060_lifecycle_runner_contract_test.sh
bench/scripts/tests/tc061_lifecycle_runner_loop_test.sh

# F09–F10: evaluation identity, retention, aggregate, and reports
bench/scripts/tests/tc067_lifecycle_evaluation_truth_test.sh
bench/scripts/tests/tc070_comparison_identity_test.sh
bench/scripts/tests/tc074_invalid-retention-and-aggregation_test.sh
bench/scripts/tests/tc079_operator_preview_zero_spend_test.sh
bench/scripts/tests/tc081_pilot_ledger_gate_test.sh
bench/scripts/tests/tc082_retention_layout_test.sh
bench/scripts/tests/tc088_dimension_separation_and_noise_band_test.sh
bench/scripts/tests/tc094_e40_benchmark_operator_test.sh
```

Use the exact registered names in `bench/scripts/tests/run-all.sh` if a test
has been renamed; the list above is an operator tour, not a replacement for
the full wrapper.

### 6. Prepare and preview a lifecycle batch

Create an operator-owned batch policy that follows the contract in the header
of `bench/scripts/run-lifecycle-batch.sh`. It must declare `schema_version`, a
positive `min_reps`, and, for each selected scenario, an existing `root_key`, a
pre-built `scratch_root`, and—when evaluation is required—an `i05_bundle_dir`.
The scratch template must already contain the root entity; F10 does not create
that entity or bootstrap the Shark project.

Preview the matrix before any provider call:

```bash
bench/scripts/run-lifecycle-batch.sh \
  --batch <batch-policy.yaml> \
  --retention-root /tmp/e40-lifecycle-retention \
  --mode preview \
  --reps 1
```

Confirm the scenario matrix, D01–D05 applicability, planned provider calls,
retention root, ceilings, and pilot-ledger state. Preview mode must make zero
provider calls.

### 7. Run one pilot per lifecycle family

Run a bounded pilot only after preview succeeds:

```bash
bench/scripts/run-lifecycle-batch.sh \
  --batch <batch-policy.yaml> \
  --retention-root /tmp/e40-lifecycle-retention \
  --mode pilot \
  --acknowledge-provider-spend \
  --max-cost-usd <positive-limit> \
  --max-wall-clock-seconds <positive-limit> \
  --max-generated-tasks <positive-limit> \
  --reps 1
```

Inspect the retained package, stage evidence, transcripts, entity history,
lifecycle record, evaluation, and oracle for one scenario in each family.
Record the inspection attestation with the documented checklist shape, then
verify the digests:

```bash
bench/scripts/pilot-ledger.sh --retention-root /tmp/e40-lifecycle-retention \
  --record --scenario <scenario-id> --rep 1 \
  --operator <operator-identity> --checklist <checklist.json>
bench/scripts/pilot-ledger.sh --retention-root /tmp/e40-lifecycle-retention --verify
```

### 8. Run the lifecycle baseline

After every requested family has a verified pilot attestation, run the
repeated baseline matrix:

```bash
bench/scripts/run-lifecycle-batch.sh \
  --batch <batch-policy.yaml> \
  --retention-root /tmp/e40-lifecycle-retention \
  --mode baseline \
  --acknowledge-provider-spend \
  --max-cost-usd <positive-limit> \
  --max-wall-clock-seconds <positive-limit> \
  --max-generated-tasks <positive-limit> \
  --reps <minimum-from-policy>
```

Aggregate only the retained root, then render the lifecycle headline. The
stage diagnostic is subordinate and must not be presented without the
headline eligibility verdict:

```bash
bench/scripts/aggregate-lifecycle.sh \
  --retention-root /tmp/e40-lifecycle-retention \
  > /tmp/e40-lifecycle-aggregate.json
bench/scripts/report-lifecycle.sh \
  --aggregate /tmp/e40-lifecycle-aggregate.json --view headline \
  > /tmp/e40-lifecycle-headline.md
bench/scripts/report-lifecycle.sh \
  --aggregate /tmp/e40-lifecycle-aggregate.json --view stage_diagnostic \
  > /tmp/e40-lifecycle-stage-diagnostic.md
```

Before calling the baseline publishable, verify the retention root and each
I-07/I-08 record independently:

```bash
bench/scripts/verify-retention-root.sh \
  --retention-root /tmp/e40-lifecycle-retention \
  --schema bench/reports/lifecycle-baseline-schema.yaml
bench/scripts/verify-lifecycle-run.sh \
  /tmp/e40-lifecycle-retention/scenarios/<scenario-id>/<rep>/lifecycle.jsonl \
  --schema bench/runs/i07-schema.yaml
bench/scripts/verify-lifecycle-evaluation.sh \
  /tmp/e40-lifecycle-retention/scenarios/<scenario-id>/<rep>/evaluation.jsonl \
  --schema bench/evaluation/i08-schema.yaml
```

Publish only scenarios whose evaluation says `publication_eligible: true`,
whose identity is uniform, whose oracle and required evidence are present,
and whose aggregate has no disqualifying invalidity or insufficient-repetition
condition.

### 9. Compare review policies

Prepare a candidate declaration with the two existing scratch roots and the
same frozen candidate, then preview both comparison modes:

```bash
bench/scripts/run-review-comparison.sh \
  --candidate <candidate.yaml> \
  --retention-root /tmp/e40-review-comparison \
  --mode preview \
  --comparison-mode independent_frozen_candidate

bench/scripts/run-review-comparison.sh \
  --candidate <candidate.yaml> \
  --retention-root /tmp/e40-review-comparison \
  --mode preview \
  --comparison-mode sequential_delivery
```

Run `pilot` first, then `baseline`, with the same explicit spend and positive
ceiling flags used by the lifecycle batch. The independent mode keeps one
candidate frozen between feature QA and deep review; the sequential mode
retains each intervening candidate and fix. Do not combine their results or
publish precision/recall without a seeded truth set.

## Full-epic demo walkthrough

Use one terminal and one read-only presentation view. Walk the audience
through this order:

1. Open `bench/corpus/corpus.yaml` and show the admitted items, held-back F2P
   tests, P2P sets, fixture SHA, and base ledgers.
2. Show the v1 `run-batch.sh --dry-run` matrix, then open one retained
   `record.jsonl` and its `run.log`/transcript evidence.
3. Open `aggregate.json` and the baseline report. Point to provenance,
   per-item spread, acceptance interval, anomaly/exclusion handling, and the
   replay command.
4. Open `bench/scenarios/scenarios.yaml` and one package from each family.
   Show the D01–D05 applicability difference and the evaluator-only boundary.
5. Show lifecycle preview output, then one retained lifecycle directory with
   `package.yaml`, stage evidence, transcripts, entity history, lifecycle,
   evaluation, oracle, and manifest.
6. Show the pilot attestation and digest verification, then the aggregate and
   headline report. Explain why `publication_eligible` comes from I-08 rather
   than terminal Shark status.
7. Show the independent and sequential review-comparison previews and their
   candidate/policy identity boundary.
8. Finish with the complete `run-all.sh` result and the known evidence
   limitations below.

The audience should leave knowing how E40 establishes a baseline, what each
feature contributes, which commands are pure/offline, where provider spend is
gated, and which artifacts make a result eligible for publication.

## Demonstrated now

## Scenario: Prepare and compare with the top-level operator

- **Stakeholder value:** An operator can prepare inspectable four-family inputs, verify zero-spend behavior, pin prompt or policy variants, and reject unfair comparisons without assembling F10 paths by hand.
- **Source requirement or acceptance criterion:** `docs/demos/E40/sol-session-prompt.md`, required deliverables 1–5 and testing requirements 1–12.
- **Prerequisites and demo data:** A local Shark binary and an explicit operator root outside the repository. The acceptance test uses deterministic comparison fixtures and never claims provider execution.
- **Presenter actions:** Run `bench/scripts/tests/tc094_e40_benchmark_operator_test.sh`, then run `e40-benchmark.sh setup` and `preflight` against a new temporary root. Open the generated setup, preflight, variant-definition, manifest, and comparison shapes.
- **Expected observable result:** The test proves zero provider calls and no live-database mutation, prompt and policy digest changes, pilot-to-baseline manifest separation, fair identity boundaries, quality-dominant interpretation, unavailable truth-set metrics, no-overwrite reruns, concurrency safety, and time reconciliation.
- **Evidence type and path:** Test-backed contract and integration evidence — `bench/scripts/tests/tc094_e40_benchmark_operator_test.sh` and its generated temporary artifacts.
- **Evidence environment and date:** Local repository checkout; current implementation on 2026-08-23.
- **Acceptance/readiness classification:** Demonstrated now for the offline operator and comparison contract.
- **Reset or recovery instructions:** The test removes its temporary root. For a manual run, retain or archive the explicit operator root; do not delete the live database.
- **Known limitations:** The test does not call a provider and does not prove the missing lifecycle-adapter-to-run-matched-I-05 production caller chain. Provider-backed UAT and publication eligibility remain separate gates.

## Scenario: Inspect the retained canonical lifecycle-runner result

- **Stakeholder value:** An operator can see that the benchmark uses the public keyed Shark lifecycle and records a terminal result.
- **Source requirement or acceptance criterion:** E40-F08 `feature.md`, “Acceptance boundary”; E40 UAT plan, UAT-08/UAT-09 coverage.
- **Prerequisites and demo data:** Clean access to the committed E40-F08 retained UAT summary and the repository's `bench` scripts.
- **Presenter actions:** Open `bench/runs/e40-f08-uat/README.md` and `loop-complete.json`; point out the public commands, provider seam, terminal outcome, publication flag, and resume safety.
- **Expected observable result:** The record shows `next`, `claim`, `heartbeat`, `status advance`, and `release`; terminal outcome `complete`; `publication_eligible: true`; and `summary_only: true`.
- **Evidence type and path:** Pipeline artifact/data — `bench/runs/e40-f08-uat/README.md`, `bench/runs/e40-f08-uat/loop-complete.json`.
- **Evidence environment and date:** Isolated E40-F08 worktree; captured 2026-08-17.
- **Acceptance/readiness classification:** Demonstrated now.
- **Reset or recovery instructions:** N/A; read-only retained evidence.
- **Known limitations:** The summary explicitly says detailed dispatch, Question, and integrity artifacts are not retained; this is not a full independent UAT transcript.

## Scenario: Run the retained E40-F10 quality and regression checks

- **Stakeholder value:** An operator can verify the retained baseline tooling and its registration checks without treating a green command as production proof.
- **Source requirement or acceptance criterion:** E40-F10 `feature.md`, “Acceptance boundary”; E40-F10 current-HEAD follow-up evidence README.
- **Prerequisites and demo data:** A clean clone with initialized bench submodules, a local Shark binary, and a private temporary root as documented in the evidence.
- **Presenter actions:** From the documented environment, run `make fmt && make lint && make test`, then `bench/scripts/tests/run-all.sh`, then `TC092_RUN_LOG=run-all.log bench/scripts/tests/tc092_full-regression-registration_test.sh`.
- **Expected observable result:** The retained follow-up records exit 0 for all three commands, 77 wrapper passes, and successful TC-092 registration checks.
- **Evidence type and path:** Pipeline artifact/data — `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/evidence/verification-a76ae83f-followup/README.md` and its logs.
- **Evidence environment and date:** Primary checkout with unrelated dirty files preserved; captured 2026-08-23.
- **Acceptance/readiness classification:** Demonstrated now.
- **Reset or recovery instructions:** Use the documented clean-clone/private-`TMPDIR` setup; do not run against shared concurrent scratch state.
- **Known limitations:** This evidence does not independently clear prior UAT findings or the upstream time/cost contract gap. Older retained bundles include failures and must remain visible in the risk section.

## Scenario: Show the v1-to-v2 contract map and operator flow

- **Stakeholder value:** Reviewers can trace how corpus, liveness, lifecycle, evaluation, and reporting artifacts are intended to connect.
- **Source requirement or acceptance criterion:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md`; E40 architecture and UAT plan.
- **Prerequisites and demo data:** The committed E40 planning documents.
- **Presenter actions:** Open the interaction map and walk I-01 through I-08, then compare the feature order and each feature's producer/consumer contract with the live completed statuses.
- **Expected observable result:** Each interaction has a named producer, consumer, shape, payload, and evidence boundary; staged I-04 through I-08 edges remain labeled `contract-only` until live wiring is proven.
- **Evidence type and path:** Pipeline artifact/data — `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md`.
- **Evidence environment and date:** Repository documentation last updated 2026-08-13; counterpart statuses re-read from Shark on 2026-08-23.
- **Acceptance/readiness classification:** Demonstrated now.
- **Reset or recovery instructions:** N/A; read-only documentation.
- **Known limitations:** This demonstrates the documented contract map, not every downstream production caller chain.

## Not demonstrated / pending integration

## Scenario: Claim a complete production-path lifecycle baseline

- **Stakeholder value:** A maintainer should be able to publish a comparable, oracle-backed lifecycle baseline.
- **Source requirement or acceptance criterion:** E40-F09 and E40-F10 acceptance boundaries; I-07/I-08 staged interaction rows.
- **Prerequisites and demo data:** A retained run, evaluation record, complete identity set, held-back oracle, and publication report for every selected scenario.
- **Presenter actions:** Do not present this as complete. Show the staged I-07/I-08 rows and the F10 evidence limitations.
- **Expected observable result:** The current materials do not establish a complete production-path publication claim.
- **Evidence type and path:** Pipeline artifact/data — F10 evidence READMEs under `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/evidence/`.
- **Evidence environment and date:** Current retained evidence reviewed 2026-08-23.
- **Acceptance/readiness classification:** Not demonstrated / pending integration.
- **Reset or recovery instructions:** Supply the missing live caller-chain, complete evaluation, and publication evidence before reclassifying.
- **Known limitations:** Completed Shark status is not publication evidence; per-gate time/cost and paired time/cost deltas remain documented upstream-contract limitations.

## Scenario: Demonstrate replayed product-design and stage-evidence consumption end to end

- **Stakeholder value:** Feature scenarios should consume replayed D01-D05 and immutable stage evidence through the real downstream caller chain.
- **Source requirement or acceptance criterion:** E40-F06 and E40-F07 acceptance boundaries; staged I-05 and I-06 rows in the interaction map.
- **Prerequisites and demo data:** A retained replay bundle, stage snapshots, artifact-consumption records, and a live F08 caller-chain capture.
- **Presenter actions:** Show the contract and F08 summary-only evidence, but mark the missing end-to-end retained proof.
- **Expected observable result:** The contract shapes are documented; the selected evidence does not contain the complete downstream production-path records required by the readiness handoff.
- **Evidence type and path:** Pipeline artifact/data — E40-F06/F07 specs and `bench/runs/e40-f08-uat/README.md`.
- **Evidence environment and date:** Documents and retained summary reviewed 2026-08-23.
- **Acceptance/readiness classification:** Not demonstrated / pending integration.
- **Reset or recovery instructions:** Add the originating UAT's complete replay, stage, and caller-chain records.
- **Known limitations:** The F08 retained summary explicitly omits detailed dispatch, Question, and integrity artifacts.

## Accepted risks and overrides

The independent assessor verdict, owner decision, and open conditions are kept separate. For I-04 through I-08, the E40 interaction map declares `gate_mode: contract-only`, names activation owners and closure keys, requires live counterpart status, and sets `demonstrability_disposition: pending-integration`. E40's selected documents do not record a separate assessor verdict or owner override for these rows; no owner decision is inferred from `completed` status.

| Readiness field | E40 demo treatment |
|---|---|
| `assessor_verdict` | Not recorded in the selected E40 guidance; do not infer one. |
| `owner_decision` | Not recorded as an override; completion is not substituted. |
| `open_conditions` | Live caller-chain, production-path, and retained-evidence obligations remain visible. |
| `gate_mode` | `contract-only` for I-04 through I-08. |
| `activation_owner` | Read from each staged interaction row: E40-F06/F07/F08, E40-F08/F09/F10, E40-F08, E40-F09/F10, and E40-F10 respectively. |
| `closure_key` | The corresponding consumer feature's own UAT, as specified in each row. |
| `counterpart_status` | Re-read live on 2026-08-23: all ten E40 features are `completed`; this does not close the evidence obligation. |
| `review_basis` | E40 interaction map, feature specifications, UAT plan, and retained evidence READMEs. |
| `demonstrability_disposition` | `pending-integration` for the staged handoffs until the named owners provide closed live proof. |

Known retained risks include the F08 summary-only limitation, older negative/stale F10 evidence bundles, and the documented upstream-contract gaps for per-gate cost/time and paired time/cost deltas. These remain risks and evidence qualifiers, not demonstrated delivery.

## Presenter close

E40 is demonstrable as a completed benchmark implementation with retained F08 lifecycle and F10 operator verification surfaces. The demo must end by distinguishing those artifacts from independent UAT, production-path integration, and publication readiness.
