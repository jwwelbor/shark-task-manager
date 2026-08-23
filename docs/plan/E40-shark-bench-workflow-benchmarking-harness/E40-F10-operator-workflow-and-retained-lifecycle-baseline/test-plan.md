# Test Plan: E40-F10 - Operator workflow and retained lifecycle baseline

**Created:** 2026-08-19
**Feature Spec:** docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/spec.md
**Parent UAT Plan:** docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md
**Feature Research Report:** docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/research-report.md
**Status:** APPROVED

## Acceptance Criteria Review

### Ambiguity Findings

None outstanding. Every AC in spec.md already names its verifying test file
(`tc0NN_*_test.sh` or the Go contract test), its exact command or invocation,
and a concrete expected result — this is unusually far along for pre-development
test planning, and this plan inherits those names as TC-078 through TC-092
rather than renumbering. "Correctly", "gracefully", and "follows the pattern"
do not appear anywhere in spec.md's AC table; every AC instead names a
schema-owned vocabulary term, an exit code, or a byte-identity check.

One residual ambiguity was found and closed here: AC-011's "statically scans
report output fields" (for composite/ROI/efficiency score absence) does not by
itself prove no such field is *computed and merely unprinted*. TC-088 below
closes this by asserting absence in both the printed report and the
`aggregate.json` schema, not the report text alone (see TC-088 Negative/edge).

### Missing Coverage

None. Every REQ-F/REQ-NF row in spec.md's Verification traceability table
resolves to at least one AC, and every AC resolves to exactly one TC (078-092)
below. UAT-15 through UAT-18 (F10's owned scenarios) each map to at least one
TC; UAT-19 is E40-F06/F08/F09 owned and is only transitively touched by TC-087
comparison-verdict preservation (see Cross-feature contract tests).

## ISTQB Technique Application (per AC)

Every runtime AC applies `contract-surface enumeration` for the schema and
CLI-flag partitions (this feature is almost entirely a closed-vocabulary,
flag-and-file-shape surface), `decision table` where a refusal or view depends
on several independent conditions (spend gate, pilot gate, eligibility
publication), `attack-class enumeration` for the zero-provider-call and
retention-integrity safety properties, `boundary-value analysis` for ceiling
positivity and rep-count thresholds, and `state transition` for the
preview→pilot→baseline command sequencing and the headline/stage-diagnostic
view dependency. AC-012 (phase separation) is `regression testing` against a
byte-unmodified Phase 1 surface.

| AC | Technique(s) Applied | Test Cases Generated | Rationale |
|---|---|---|---|
| AC-001 | Contract-surface enumeration + BVA | TC-078 | Schema-owned vocabulary, fixture matrix, malformed-path failure are equivalence partitions over a closed schema. |
| AC-002 | Attack-class enumeration + state transition | TC-079 | Zero-spend proof requires a denial harness (attack class) over the preview→pilot state boundary. |
| AC-003 | Decision table + BVA | TC-080 | Refusal depends on five independent conditions (ack flag, three ceilings, retention root); BVA covers zero/negative/absent ceiling values. |
| AC-004 | Decision table + state transition | TC-081 | Pilot-ledger gate is a 3-state decision (no attestation / stale digest / verified) gating a state transition into `--mode baseline`. |
| AC-005 | Attack-class enumeration + equivalence partitioning | TC-082 | Complete/missing-artifact/digest-mismatched/re-serialized roots are damage-class partitions; reclaim-vs-skip is an equivalence split. |
| AC-006 | State transition + attack-class enumeration | TC-083 | Completed-workflow-but-failed-oracle is a specific terminal state; offline-diagnosability is a "no live surface" attack-class property. |
| AC-007 | Equivalence partitioning + decision table | TC-084 | Six disjoint shares over two closed vocabularies (stage_category x interval_category) form a decision-table partition that must reconcile. |
| AC-008 | Equivalence partitioning + attack-class enumeration | TC-085 | Seven finding measures x severity x defect class is an equivalence-partition matrix; zero-findings-vs-collection-failure is an attack-class-style ambiguity trap. The staged I-08 measurement fields are copied when present; otherwise TC-085 asserts the schema-owned upstream-contract-gap reason. |
| AC-009 | Equivalence partitioning + attack-class enumeration | TC-086 | Consumed/orphan artifact classes and a static forbidden-language scan (human-minute framing) are equivalence and attack-class checks. |
| AC-010 | Decision table + one-factor mutation | TC-087 | Independent vs. sequential mode x identity-compatible vs. divergent pair is a 2x2 decision table. |
| AC-011 | Equivalence partitioning + BVA + static scan | TC-088 | Above/below noise-band and above/below minimum reps are boundary partitions; ROI-field absence is a static-scan equivalence check. Time/cost remain explicitly staged until the I-08 producer fields and closure evidence exist. |
| AC-012 | Regression testing + equivalence partitioning | TC-089 | Phase 1 vs. lifecycle v2 input is an equivalence partition; byte-unmodified Phase 1 path is a regression assertion. |
| AC-013 | Retest/reproducibility testing + BVA | TC-090 | Byte-identical double-run under denial is retest testing; the 100 MB/60-second bound is BVA. |
| AC-014 | Structural/static scan + attack-class enumeration | TC-091 | Forbidden-write and forbidden-content scans are attack-class enumeration over the complete production write/read surface, including retain_pair, path-safety.sh, and verify_pair_retention. |
| AC-015 | Regression testing | TC-092 | Full-suite and prior-registration preservation is classic regression testing. |

## ISO 25010 Coverage Matrix

| AC | Functional Suitability | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | TC-078 required fields/vocab schema-owned | N/A: no AC latency target | TC-078 reads committed YAML/JSON only | TC-078 asserts a named failing path in diagnostics | TC-078 rejects malformed fixtures with the failing path | TC-078 excludes secret/credential fields from schema | TC-078 makes schema the single vocabulary owner (REQ-F-018) | TC-078 uses the repository Go contract harness |
| AC-002 | TC-079 both previews print full required content | N/A: preview has no stated latency SLA | TC-079 exercises both `--mode preview` drivers | TC-079 asserts human-readable matrix/stage/call/root/ceiling output | TC-079 proves preview stays correct and completes with zero provider/network reachable at all (fault-tolerant to their absence, not merely to their success) | TC-079 proves zero denied provider/network calls under an enumerated PATH-shim binary list (§Test Infrastructure) | TC-079 reuses the existing dry-run idiom rather than forking a third flag | TC-079 runs as a POSIX shell fixture |
| AC-003 | TC-080 refuses on any missing/non-positive condition | N/A: refusal has no latency target | TC-080 covers both provider-backed drivers identically | TC-080 asserts the refusal names the missing condition | TC-080 asserts refusal occurs before any subprocess (zero side effects) | TC-080 proves env-var/config-default acknowledgement cannot satisfy the gate | TC-080 asserts one refusal owner (`spend-gate.sh`) for both drivers | TC-080 runs as a POSIX shell fixture |
| AC-004 | TC-081 gates `--mode baseline` on verified attestation | N/A: no latency target | TC-081 exercises `pilot-ledger.sh --record/--verify` | TC-081 asserts the refusal names the failing family | TC-081 asserts a mutated artifact invalidates a previously-verified attestation | TC-081 ties inspection to artifact digest, not a boolean flag (ADR-F10-09) | TC-081 keeps ledger format schema-owned | TC-081 runs as a POSIX shell fixture |
| AC-005 | TC-082 verifies byte-identical retained artifacts | N/A: no latency target | TC-082 exercises `verify-retention-root.sh` | TC-082 asserts each damaged root names the artifact and reason | TC-082 distinguishes complete/missing/mismatched/re-serialized roots | TC-082 detects digest tampering and re-serialization | TC-082 reuses `run-batch.sh` classification discipline (REQ-NF-007) | TC-082 runs as a POSIX shell fixture over a temp retention root |
| AC-006 | TC-083 proves offline reachability of all named evidence | N/A: no latency target | TC-083 reads only retained I-05/I-07/I-08 files | TC-083 asserts the diagnosing operator can resolve every named evidence path (oracle result, dispatch/gate lineage, per-stage evidence, transcripts) directly from the scenario's retention directory without a lookup tool or documentation, per REQ-F-006's explicit "reachable... without rerunning" usability requirement | TC-083 asserts no provider or scenario rerun occurs | TC-083 proves a completed-workflow / failed-oracle fixture is not silently published | TC-083 keeps oracle/gate/evidence read-only per ADR-F10-04 | TC-083 runs as a POSIX shell fixture |
| AC-007 | TC-084 reconciles both partitions to lifecycle wall time | TC-084 asserts reconciliation arithmetic only, no runtime SLA | TC-084 consumes the closed i05 stage/interval vocabularies unmodified | TC-084 asserts both report views print human-readable partition tables with named categories and an explicit `unattributed` line, so an operator reading the report (not just a machine) can see the full reconciliation, per REQ-F-010's explicit report-presentation requirement | TC-084 asserts `unattributed` prints even at zero (no silent absorption) | TC-084 proves rework attribution reads a retained fact rather than an inferable-and-spoofable signal (gate rounds/status re-entry) | TC-084 asserts REQ-F-011's rework-flag-not-inferred rule | TC-084 runs as a POSIX shell fixture |
| AC-008 | TC-085 renders all seven finding measures correctly | N/A: no latency target | TC-085 reads I-08 `review_findings`/`metrics` unmodified | TC-085 asserts zero-findings and collection-failure render visibly distinct | TC-085 distinguishes a real zero-finding gate from a failed collection | TC-085 asserts `unavailable` (never `0`) without a seeded truth set (ADR-F10-10) | TC-085 keeps finding-measure vocabulary I-08-owned | TC-085 runs as a POSIX shell fixture |
| AC-009 | TC-086 distinguishes consumed/orphan artifacts and replay proxies | N/A: no latency target | TC-086 reads I-08 `metrics.artifact_use` unmodified | TC-086 asserts every replay-proxy field carries a replayed-proxy label | N/A: artifact-lineage reporting has no failure-recovery property | TC-086's static scan finds no human-minute/human-hour/human-effort framing | TC-086 keeps D01-D05 proxy vocabulary I-06/I-08-owned | TC-086 runs as a POSIX shell fixture plus a static grep scan |
| AC-010 | TC-087 preserves the comparator's verdict verbatim | N/A: no latency target | TC-087 exercises both comparison-mode drivers | TC-087 shows candidate identities/policies/truth-set/fix-rules before spend | TC-087 never publishes a comparator-rejected pair | TC-087 rejects a branch-name-only or `HEAD`-only match (delegated, not re-implemented) | TC-087 delegates identity to `compare-lifecycle-evaluations.sh`, adds no second implementation (REQ-F-015) | TC-087 runs as a POSIX shell fixture |
| AC-011 | TC-088 separates quality/time/cost into three blocks | N/A: no latency target for report rendering | TC-088 consumes the existing noise-band mechanism (`aggregate-runs.sh` semantics) | TC-088 asserts "no detectable effect" renders in place of a false improvement/regression claim | TC-088 asserts `insufficient_reps` flags a below-minimum matrix rather than silently publishing | TC-088's static scan proves no composite/ROI/value field exists in schema or output | TC-088 keeps the noise-band derivation-rule name schema-owned | TC-088 runs as a POSIX shell fixture plus a schema/static scan |
| AC-012 | TC-089 refuses a v1 record with a named reason | N/A: no latency target | TC-089 proves Phase 1 scripts remain runnable unmodified | TC-089 asserts every F10 artifact/report carries an explicit phase label | TC-089 diffs the three Phase 1 scripts against the pinned `PRE_F10_BASELINE_SHA` (byte-unmodified) | N/A: phase separation is not a disclosure/security property | TC-089 asserts each phase keeps its own aggregator (ADR-F10-11) | TC-089 runs as a POSIX shell fixture with a `git diff` check |
| AC-013 | TC-090 asserts byte-identical double-run output | TC-090 asserts the 100 MB fixture completes within 60s (REQ-NF-004) | TC-090 runs under provider/network/DB/live-tree denial | N/A: determinism is a machine-verifiable, not end-user, property | TC-090 asserts zero denied calls occur on both runs | TC-090 denies provider/network/DB/live-tree access and proves it | TC-090 asserts reports/aggregates consult no clock (REQ-NF-003) | TC-090 asserts peak memory shows streaming, not full-payload load (REQ-NF-004) |
| AC-014 | TC-091 statically proves the safety/scope boundary | N/A: static scan has no runtime latency target | TC-091 scans all F10 scripts and the schema file | N/A: static scan output is a machine/CI diagnostic | N/A: static scan is not a failure-recovery property | TC-091 proves no write outside retention root and no credential/prompt/transcript copy | TC-091 proves no `internal/`/`cmd/`/migration/`.sharkconfig.json` change and no language branch | TC-091 runs as a static grep/AST-style scan over committed shell/schema files |
| AC-015 | TC-092 runs the full quality gate and F10 suite | TC-092 records suite completion/failure only, no new SLA | TC-092 runs `make lint`/`make test` across the whole Go/shell surface | N/A: development quality gate, no product UX | TC-092 asserts no existing F01-F09 test is removed, skipped, or weakened | TC-092 keeps provider/network/DB denial fixtures active throughout the full run | TC-092 registers TC-078 through TC-092 deterministically in `run-all.sh` | TC-092 uses documented `make` and shell commands |

### Coverage Gaps

None identified. Every cell above is a deliberate decision; `N/A` cells are
justified inline (no latency SLA is declared by spec.md for a given behavior,
or the ISO characteristic does not structurally apply to a machine-only
diagnostic surface).

## Observability Design (per behavior)

F10 adds no Shark product-observability row (REQ-NF-001: no `internal/`/`cmd/`
change). Its runtime evidence is entirely the retained file artifacts named in
spec.md's Data model section plus bounded stderr diagnostics, matching the
F08/F09 precedent.

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Operator preview | `batch.json`-shaped preview payload printed to stdout (scenario matrix, stages, planned calls, root, ceilings, pilot-ledger state) | Mandatory bounded stderr diagnostic naming any unresolved I-04 stage matrix or scenario | N/A: preview is a single synchronous CLI invocation | Any denied provider/network call during preview | TC-079 asserts exact printed fields and zero denied calls |
| Spend-gate refusal | `refusal_reason` field (schema-owned vocabulary) in the refusal output | Mandatory stderr line naming the exact missing condition (flag/ceiling/root) | N/A: refusal short-circuits before any dispatch | Any refusal that occurs after a subprocess/checkout/Shark call | TC-080 asserts pre-dispatch refusal and the named reason |
| Pilot-ledger attestation | `pilot-ledger.jsonl` row per family: inspected run reference, checklist results, operator identity, inspected-artifact digests | Mandatory stderr on `--verify` mismatch naming the family and stale digest | N/A: ledger is a file artifact, not a live trace | Any `--mode baseline` proceeding without a currently-valid attestation | TC-081 asserts refusal naming the family and digest-mismatch detection |
| Retention verification | `verify-retention-root.sh` bounded verdict (per-artifact pass/fail, reason) | Mandatory stderr naming the artifact and damage class (missing/mismatched/re-serialized) | N/A: offline verifier, no live trace | Any damaged root that verifies as complete | TC-082 asserts a complete root verifies and each damage class fails with a named reason |
| Failed-oracle diagnosis | `execution_oracle` result plus reachable `dispatches`/gate lineage in the retention directory | N/A: diagnosis reads retained records only, produces no new log | N/A: offline, no live trace | Headline reporting a not-correct workflow as publication-eligible | TC-083 asserts the headline reports the workflow as not correct and evidence is offline-reachable |
| Time reconciliation | `time` block in `aggregate.json`: stage-category, interval-category, and REQ-F-011 six-share partitions, each with `unattributed` | N/A: pure aggregation, no subprocess log | N/A: pure function, no trace | Any partition that fails to reconcile to lifecycle wall time, or a nonzero cell silently dropped | TC-084 asserts reconciliation and a printed `unattributed` line even at zero |
| Review-value reporting | `review_value` block per gate: seven finding measures by severity/defect class, elapsed/provider/resolution cost | N/A: pure rendering from retained I-08 | N/A: pure function | `0` printed for precision/recall without a seeded truth set | TC-085 asserts `unavailable` rendering and visible zero-vs-failure distinction |
| Artifact-use / replay-proxy reporting | `artifact_use` block: produced/consumed/reused/orphan counts, replayed D01-D05 proxy counts under a replayed-proxy label | N/A: pure rendering | N/A: pure function | Any replay proxy rendered without its replayed-proxy label, or as human effort | TC-086 asserts labelling and the static human-effort-framing scan |
| Review-comparison operation | `comparisons` block: verbatim comparator verdict, divergence reasons, mode, candidate/policy identity references | Mandatory stderr on comparator rejection naming the divergence reason (delegated, not re-derived) | N/A: preview/report are synchronous CLI calls | Publishing a comparator-rejected pair | TC-087 asserts verbatim verdict preservation and rejection-blocks-publication |
| Dimension separation / noise bands | `noise_bands` block: min/median/max/spread/acceptance interval/derivation rule/rep count/`insufficient_reps` | N/A: pure aggregation | N/A: pure function | Any composite/blended score field appearing anywhere in schema or output | TC-088 asserts three separate blocks, "no detectable effect" rendering, and the static composite-score scan |
| Phase separation | `phase` label (`lifecycle_v2`) on every F10 artifact/report; refusal reason when fed a v1 `record.jsonl` | Mandatory stderr naming the v1-input refusal reason | N/A: pure aggregation | Any F10 output missing its phase label, or a v1 record silently coerced | TC-089 asserts refusal, byte-unmodified Phase 1 diff, and phase-label presence |
| Offline determinism / scale | Byte-identical `aggregate.json`/report output across two runs; peak-memory evidence of streaming | Mandatory stderr on any denied provider/network/DB/live-tree attempt | N/A: pure function, no trace | Any run consulting a clock, subprocess, or exceeding the 60s/100MB bound | TC-090 asserts byte identity, zero denied calls, and the timing/memory bound |

Pure canonical-digest and partition-arithmetic helpers (share-partition
computation, noise-band statistics) are `internal — no independent
observability`; their correctness is proven by the deterministic output they
produce, asserted directly by TC-084/TC-088/TC-090. This is not an escape
hatch: every one of their outputs is a required `aggregate.json` field checked
by a TC above.

## Cross-feature contract tests (I-##)

The staged values below are copied verbatim from `E40-interaction-map.md` and
`spec.md`'s Cross-feature interactions section. Counterpart status is not
copied into this plan; the parent loop must read it live from Shark at
review/UAT time.

| I-## | Producer | Consumer(s) | Shape source | Gate mode / activation / closure | Review basis / disposition | Shared contract test pointer | TC |
|---|---|---|---|---|---|---|---|
| I-05 | E40-F06 | E40-F08/F09/F10 | `architecture.md#stage-evidence-and-isolation-contract` | `contract-only` until F10 proves live production-path use; activation owner E40-F10 (its slice); closure key E40-F10 at its own UAT; counterpart status live | F06 completed `spec.md` + map row; `pending-integration` | `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042` | TC-078 (schema/fixture proof); TC-084/TC-086 (F10's own runtime slice: time partitions, artifact-use) |
| I-07 | E40-F08 | E40-F09/F10 | `architecture.md#lifecycle-run-record-contract` | `contract-only` until F10 proves live production-path use; activation owner E40-F10; closure key E40-F10 at its own UAT; counterpart status live | F08 completed `spec.md` + map row; `pending-integration` | `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061` | TC-078 (schema/fixture proof); TC-083 (dispatch/gate lineage reachability) |
| I-08 | E40-F09 | E40-F10 | `architecture.md#lifecycle-evaluation-record-contract` | `contract-only` until F10 proves live production-path use; activation owner E40-F10; closure key E40-F10 at its own UAT; counterpart status live | F09 completed `spec.md` + map row; `pending-integration` | `tests/contracts/e40_i08_lifecycle_evaluation_contract_test.go#TC-067` (the single shared contract proof named by E40-F09; F10 reuses this pointer and creates no twin test) | TC-078 (schema/fixture proof, same pointer as F09's TC-067); TC-085/TC-087 (F10's own runtime slice: review-value, comparison) |

These are shared proofs, not twin tests: TC-078 is F10's own Go contract test
(`e40_f10_operator_baseline_contract_test.go`) validating F10's *own* schema
(`lifecycle-baseline-schema.yaml`) and fixtures — it references the I-05/I-07/I-08
shape sources by pointer per REQ-F-018 and spec.md's Component-changes row, and
does not restate or re-implement `TC-042`/`TC-061`/`TC-067`. Per ADR-F10-04, F10
never re-validates I-05/I-07/I-08 field-level correctness itself; that remains
F06/F08/F09's obligation proven by their own contract tests.

## Cross-epic integration tests (X-##)

None. spec.md's "Cross-epic integrations" section states explicitly that F10
produces, consumes, and validates no X-## row: every X-## in
`E40-cross-epic-map.md` and `docs/product/cross-epic-integration-map.md` names
a different owning feature (X-07 E40-F02, X-08 E40-F04, X-09 E40-F06, X-10
E40-F07, X-11/X-13 E40-F08, X-12 E40-F09). X-09's provider-usage mapping,
X-11's Rider loop, and X-12's installed-content identity reach F10 only
transitively inside I-05/I-07/I-08 fields, which are already covered above by
TC-078 and F10's runtime slices; re-declaring any of them at F10 would create a
second owner for a contract another feature already owns. This finding is
explicit, not a silent omission (Rule 12 / fail loud).

## Caller-Path Contracts (per test case)

Per spec.md's Architecture section, none of F10's driver scripts exist yet
(`run-lifecycle-batch.sh`, `run-review-comparison.sh`, `pilot-ledger.sh`,
`verify-retention-root.sh`, `aggregate-lifecycle.sh`, `report-lifecycle.sh` are
all listed **New** in the Component-changes table). Each Caller-Path Contract
below cites the exact production entrypoint and argument shape spec.md's API
section already commits to, so the developer implements against a fixed
signature rather than a convenience helper.

| TC | Entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-078 | `go test ./tests/contracts -run TestTC078_F10OperatorBaselineContract`; reads `bench/reports/lifecycle-baseline-schema.yaml` and `tests/contracts/testdata/e40_f10/{valid,invalid}` via real filesystem/parser | None; committed schema/fixtures are the seam | Do not mock schema loading, field enumeration, canonical JSON, or the validator | A hand-built in-memory record would stay green after the real schema or fixture is broken. |
| TC-079 | `bench/scripts/run-lifecycle-batch.sh --batch <policy.yaml> --retention-root <root> --mode preview` and `bench/scripts/run-review-comparison.sh --candidate <ref> --mode preview --comparison-mode <mode> --retention-root <root>`, run under a PATH shim that replaces the enumerated binary list (§Test Infrastructure: every provider CLI and network tool the drivers or `run-lifecycle.sh` can invoke) with a script that exits non-zero and logs the call | Only the enumerated PATH-shim provider/network denial process; `run-lifecycle.sh --mode dry-run` invocation itself is real | Do not mock preview-content assembly, stage resolution, or the dry-run delegation call | A preview that shells out for real provider pricing or scenario state would leak a call the shim should have caught. |
| TC-080 | Baseline fully-satisfied invocation: `run-lifecycle-batch.sh --batch <policy.yaml> --retention-root <root> --mode pilot --acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10` and `run-review-comparison.sh --candidate <ref> --mode pilot --comparison-mode independent_frozen_candidate --retention-root <root> --acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10`; each refusal case removes or corrupts exactly one required condition from this baseline | None; real `spend-gate.sh` sourced library and real CLI parsing | Do not mock flag parsing, `spend-gate.sh` refusal logic, or the pre-dispatch short-circuit | A gate that checked ceilings after the first subprocess spawn would cost a live call on a doomed run. |
| TC-081 | Baseline: `pilot-ledger.sh --retention-root <root> --record --scenario <id> --rep <n> --operator <identity> --checklist <checklist.json>` then `--verify [--family <f>]`; gate check: `run-lifecycle-batch.sh --batch <single-family-policy.yaml> --retention-root <root> --mode baseline --acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10`, invoked once per single-family batch policy (see TC-081 below for why the matrix is single-family per invocation) | None; real ledger file and real digest computation over a temp retention root | Do not mock digest comparison, family lookup, or the baseline-mode gate check | A ledger keyed on a boolean flag instead of an artifact digest would not catch a mutated retained artifact. |
| TC-082 | `verify-retention-root.sh --retention-root <root> --schema bench/reports/lifecycle-baseline-schema.yaml` over a root produced by one real `run-lifecycle-batch.sh --mode pilot` retention | None; real retained files, real `verify-lifecycle-run.sh`/`verify-lifecycle-evaluation.sh` delegation | Do not mock digest verification, layout completeness checks, or upstream schema-validity delegation | A verifier trusting file *presence* without a digest check would accept a re-serialized (silently mutated) artifact. |
| TC-083 | Retain a fixture I-07/I-08 pair (terminal `completed` outcome, `execution_oracle` = `fail`) under a temp retention root, then `report-lifecycle.sh --aggregate <aggregate.json> --view headline` with no provider/network access available | None; real aggregator and reporter over the fixture root | Do not mock `eligibility` verbatim pass-through, oracle-result reading, or dispatch-lineage reachability | A headline that inferred correctness from terminal status alone would publish a functionally broken workflow as eligible. |
| TC-084 | `aggregate-lifecycle.sh --retention-root <root>` over a fixture with provider work, tool/test time, a replayed gate, retry time, wait time, and unclassified time, then `report-lifecycle.sh --aggregate <root>/aggregate.json --view headline` and `--view stage_diagnostic` | None; pure aggregator/reporter over fixture I-05/I-07 data | Do not mock the stage_category/interval_category lookup, the REQ-F-011 cell-partition function, or `unattributed` residual computation | A partition that summed shares independently instead of walking disjoint cells could double-count or silently drop a cell. |
| TC-085 | `report-lifecycle.sh --aggregate <aggregate.json> --view headline` (review_value block) over an aggregate built from I-08 fixtures containing findings, an explicit zero-finding gate, a collection-failure gate, duplicates/recurrences, confirmed/unconfirmed findings, a downstream escape, seeded-truth and no-truth-set cases | None; real aggregator/reporter reading I-08 `review_findings`/`metrics` verbatim | Do not mock precision/recall availability logic, or the zero-vs-collection-failure rendering distinction | A reporter defaulting an absent measure to `0` would misrepresent an untested gate as a clean one. |
| TC-086 | `report-lifecycle.sh --aggregate <aggregate.json> --view headline` (artifact_use block) over a fixture with one consumed artifact, one orphan, and replayed D01-D05 proxies, plus a static `grep`/scan of every F10 script/template for human-effort language | None for the report path; the static scan is a real grep/text scan over committed files | Do not mock producer/consumer edge typing or the replayed-proxy label attachment | A report omitting the replayed-proxy label would let a reader mistake a replay count for observed human effort. |
| TC-087 | `run-review-comparison.sh --candidate <ref> --mode pilot\|baseline --comparison-mode independent_frozen_candidate\|sequential_delivery --retention-root <root> --acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10`, which internally invokes `bench/scripts/compare-lifecycle-evaluations.sh --left <e1> --right <e2> --mode <mode> --output <comparison.json>` unchanged | None; real comparator invocation over identity-compatible and one-field-divergent retained I-08 pairs | Do not mock the comparator call, its accept/reject verdict, or divergence-reason preservation | A wrapper re-deriving its own identity check instead of delegating would risk accepting a branch-name-only match the comparator would reject. |
| TC-088 | `report-lifecycle.sh --aggregate <root>/aggregate.json --view headline` over an aggregate with one metric clearing its noise band and one not, plus a matrix below the declared minimum reps, and a static scan of `bench/reports/lifecycle-baseline-schema.yaml` and the enumerated F10 report-template file list (§Test Infrastructure) for the enumerated composite/ROI/value field-name list | None; real aggregator/reporter and a real static scan | Do not mock noise-band comparison, the `insufficient_reps` flag computation, or the three-block dimension separation | A report blending quality/time/cost into one score would violate REQ-F-016's flat prohibition, undetectable by inspection alone. |
| TC-089 | `aggregate-lifecycle.sh --retention-root <root-with-v1-record.jsonl>` (expect named refusal), then run the unmodified `run-batch.sh`/`aggregate-runs.sh`/`report-baseline.sh` path and `git diff <pre-F10-baseline-commit> -- bench/scripts/run-batch.sh bench/scripts/aggregate-runs.sh bench/scripts/report-baseline.sh` (baseline commit pinned at F10 task-decomposition time, not a moving `HEAD`) | None; real aggregator, real Phase 1 scripts, real `git diff` | Do not mock the v1-record-detection refusal or the byte-unmodified diff check | A "coerce and continue" aggregator would silently mix v1 and v2 semantics into one aggregate. |
| TC-090 | `aggregate-lifecycle.sh --retention-root <root>` and `report-lifecycle.sh --aggregate <root>/aggregate.json --view headline\|stage_diagnostic`, run twice, under a provider/network/DB/live-tree denial harness, including once over the committed 100 MB synthetic retention fixture with peak-memory measurement | Only the denial-harness process boundary; the aggregator/reporter under test are real | Do not mock canonical JSON serialization, streaming I/O, or the clock/subprocess-absence assertion | An aggregator reading a system clock or loading full transcripts into memory would fail determinism or the 60s/100MB bound under real load. |
| TC-091 | Static scan (`grep`/path-walk) of every committed F10 script (`bench/scripts/run-lifecycle-batch.sh`, `run-review-comparison.sh`, `pilot-ledger.sh`, `verify-retention-root.sh`, `aggregate-lifecycle.sh`, `report-lifecycle.sh`, `lib/spend-gate.sh`) and `bench/reports/lifecycle-baseline-schema.yaml` | None; real static scan over committed source | Do not replace the scan with a documentation claim; the scan must read actual script bytes | A script that wrote outside the declared retention root, or branched on fixture language, would pass a doc-only review but fail a real scan. |
| TC-092 | `make fmt && make lint && make test` and `bench/scripts/tests/run-all.sh` with TC-078 through TC-092 registered | None beyond the explicitly declared provider/network denial fixtures already required by TC-079/080/090 | Do not replace the full suite with only F10's new tests, and do not delete or skip an existing F01-F09 registration | A green F10-only run would conceal a regression in F05-F09's tests or an accidentally weakened prior assertion. |

## Acceptance Test Cases

### TC-078: F10 schema ownership and fixture validation

**Feature Requirement:** REQ-F-018; component-changes row for
`bench/reports/lifecycle-baseline-schema.yaml`.
**Acceptance Criterion:** AC-001.
**Technique Applied:** Contract-surface enumeration + Boundary-value analysis.
**ISO 25010 Characteristic(s):** Functional Suitability, Security,
Maintainability, Reliability.

**Caller-Path Contract:** see table above.

**Preconditions:** `bench/reports/lifecycle-baseline-schema.yaml` and
`tests/contracts/testdata/e40_f10/{valid,invalid}/` are committed with
aggregate, retention-manifest, and pilot-attestation fixtures.

**Input:** Each valid fixture (complete aggregate, complete retention
manifest, verified pilot attestation) and each invalid fixture (missing
required field, wrong type, unknown refusal reason, malformed noise-band
derivation-rule name, malformed share-partition cell name, unrecognized view
name, wrong phase-label value, malformed digest).

**Expected Output:** Every valid fixture validates cleanly against the
schema. Every invalid fixture fails with the specific failing JSON path named
in the diagnostic (not a generic "invalid" message). Required fields,
retention layout, refusal-reason vocabulary, noise-band rule names,
REQ-F-011 share-partition cell names, and report view names (`headline`,
`stage_diagnostic`) are all schema-owned — no test or script maintains a
private copy of this vocabulary.

**Observability Evidence:** Schema-validation diagnostics name the failing
path per invalid fixture (asserted directly, not inferred).

**Edge Cases:** Unknown/extra field beyond the schema (must fail unless the
schema explicitly allows additive fields); an aggregate that references
`bench/evidence/i05-schema.yaml`/`bench/evaluation/i08-schema.yaml`
vocabulary terms not present in those upstream schemas (must fail — F10's
schema references, never restates, those vocabularies per the Component
changes table).

**Negative Cases:** A fixture using a private (non-schema-owned) refusal
reason string must fail; a fixture restating a `stage_category`/
`interval_category` value inconsistent with `i05-schema.yaml` must fail.

**Required field enumeration:** Per spec.md's `aggregate.json` blocks table,
the invalid-fixture matrix has one row per required block —
`identity`, `scenarios[]`, `time`, `cost`, `quality`, `review_value`,
`artifact_use`, `noise_bands`, `comparisons`, `invalid` — each exercised
missing, null, wrong-type, and empty, plus one row per named sub-field
explicitly called out in that table (`eligibility.aggregate_eligible`,
`eligibility.publication_eligible`, `eligibility.invalidity_reasons`,
`insufficient_reps`, `phase` = `lifecycle_v2`, and the digest fields). The
retention-manifest fixture matrix covers every artifact named in the Data-model
layout (same eight-item list TC-082 exercises for `verify-retention-root.sh`).
The pilot-attestation fixture matrix covers the four fields REQ-F-005 names:
inspected run reference, checklist item results, inspecting operator identity,
and inspected-artifact digests.

---

### TC-079: Zero-provider-call operator preview

**Feature Requirement:** REQ-F-001, REQ-F-014, REQ-NF-002.
**Acceptance Criterion:** AC-002.
**Technique Applied:** Attack-class enumeration + State transition.
**ISO 25010 Characteristic(s):** Functional Suitability, Usability,
Compatibility, Security.

**Caller-Path Contract:** see table above.

**Preconditions:** A PATH shim replaces every provider/network binary with a
script that exits non-zero and records any invocation attempt. A batch policy
YAML and a candidate reference are prepared.

**Input:** `run-lifecycle-batch.sh --batch <policy.yaml> --retention-root
<root> --mode preview` and `run-review-comparison.sh --candidate <ref> --mode
preview --comparison-mode independent_frozen_candidate --retention-root
<root>` (and again with `--comparison-mode sequential_delivery`).

**Expected Output:** Both previews exit `0`. The denial-attempt log is empty
(zero denied invocations). The batch preview prints: scenario matrix (id,
version, family, reps), applicable stages per scenario resolved from the
admitted I-04 stage matrix, planned provider-call inventory per scenario/stage
(including which stages are replayed and make none), resolved retention root,
declared resource ceilings, and current pilot-ledger state per family. The
comparison preview prints: candidate identities, workflow-policy identities,
expected provider-call inventory, truth-set availability, and fix rules
(whether fixes are permitted between gates).

**Observability Evidence:** PATH-shim invocation log asserted empty after
both commands.

**Edge Cases:** A scenario whose stages are entirely replayed (zero planned
calls) must still appear in the matrix with an explicit zero-call line, not be
omitted.

**Negative Cases:** A third, divergent preview flag convention (i.e., not
`--mode preview` / `--dry-run` alias) must not exist — asserted by checking
`run-lifecycle-batch.sh --help` output surface only exposes the documented
alias per REQ-F-001's "MUST extend... rather than introduce a third flag
convention."

---

### TC-080: Provider-spend gate refusal

**Feature Requirement:** REQ-F-002, REQ-F-003.
**Acceptance Criterion:** AC-003.
**Technique Applied:** Decision table + Boundary-value analysis.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Security, Maintainability.

**Caller-Path Contract:** see table above.

**Preconditions:** `spend-gate.sh` is sourced by both provider-backed drivers.
A subprocess/checkout/Shark-call spy records whether any occurred.

**Input:** For both `run-lifecycle-batch.sh --mode pilot|baseline` and
`run-review-comparison.sh --mode pilot|baseline`, one invocation each for:
(a) no `--acknowledge-provider-spend`; (b) acknowledgement present but
`--max-cost-usd` zero, negative, and absent (three sub-cases); same three
sub-cases for `--max-wall-clock-seconds` and `--max-generated-tasks`; (c) no
`--retention-root`; (d) acknowledgement supplied only via an environment
variable (e.g. `SHARK_BENCH_ACK=1`) or only via a config default, with no CLI
flag.

**Expected Output:** Every case above refuses with the schema-owned refusal
exit status (distinct from the usage-error exit status `2`) and a
machine-readable refusal reason naming the specific missing condition. The
subprocess/checkout/Shark-call spy shows zero calls for every refused case.
Only the fully-satisfied invocation (ack flag present + all three ceilings
positive + retention root present) proceeds past the gate.

**Observability Evidence:** Refusal reason string asserted equal to the
schema-owned vocabulary term for that specific missing condition (not a
generic "refused").

**Edge Cases:** Ceiling supplied as `0` vs. `-1` vs. absent must all refuse,
each potentially with the same or a distinguishable reason (assert per
schema); a ceiling supplied as a non-numeric string must also refuse (usage
error, not spend-gate refusal — distinguish exit codes).

**Negative Cases:** Acknowledgement via `SHARK_BENCH_ACK=1` env var alone
must refuse identically to no acknowledgement at all; acknowledgement via a
`.sharkconfig.json`-adjacent bench config default must refuse identically. A
stored prior acknowledgement (e.g. from a previous successful run against the
same retention root) must not satisfy the gate on a new invocation.

---

### TC-081: Pilot-ledger attestation gate

**Feature Requirement:** REQ-F-005.
**Acceptance Criterion:** AC-004.
**Technique Applied:** Decision table + State transition.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Security, Maintainability.

**Caller-Path Contract:** see table above.

**Preconditions:** A retention root with three scenario families: family A
has no pilot attestation; family B has an attestation whose recorded
inspected-artifact digest no longer matches the retained artifact (simulated
by mutating a retained file after attestation); family C has a verified,
current attestation.

**Input:** Per REQ-F-005, `--mode baseline` MUST refuse to start when *any*
requested family lacks a verified attestation — it is a whole-command
precondition, not a per-family dispatch filter. TC-081 therefore issues three
separate `--mode baseline` invocations, each against a single-family batch
policy requesting only that family: (1) family-A-only matrix, (2)
family-B-only matrix, (3) family-C-only matrix. Then `pilot-ledger.sh
--record ...` for family A followed by `--verify`, then a mutation of a
retained artifact for family A followed by `--verify` again.

**Expected Output:** Invocation (1) and (2) each refuse before any dispatch,
naming the specific requesting family (A or B respectively) and the specific
condition (no attestation vs. stale digest). Invocation (3) proceeds.
`pilot-ledger.sh --record` for family A produces an attestation that
`--verify` accepts. Mutating a retained artifact for family A after
attestation causes a subsequent `--verify` to reject it. A follow-up
multi-family matrix requesting A, B, and C together (a fourth invocation) MUST
also refuse as a whole command, naming both A and B — this asserts REQ-F-005's
whole-command refusal is not silently downgraded to per-family skipping when
multiple families are requested in one matrix.

**Observability Evidence:** Refusal names the specific family per REQ-F-005;
`--verify` output distinguishes per-family pass/fail.

**Edge Cases:** An attestation recorded against a retention root that is later
`--reclaim-incomplete`-quarantined and re-run must be treated as stale (digest
mismatch), not silently carried forward.

**Negative Cases:** A pilot attestation recorded for the wrong family (typo'd
`--scenario`/family mismatch) must not satisfy a different family's gate.

---

### TC-082: Retention-root layout and byte-preservation verification

**Feature Requirement:** REQ-F-004, REQ-NF-007.
**Acceptance Criterion:** AC-005.
**Technique Applied:** Attack-class enumeration + Equivalence partitioning.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Security, Maintainability.

**Caller-Path Contract:** see table above.

**Preconditions:** One scenario retained via a real `--mode pilot` run
against a temp retention root, producing the full layout: `package.yaml`,
`evidence/`, `transcripts/`, `entity-history.json`, `lifecycle.jsonl`,
`evaluation.jsonl`, `oracle.json`, `manifest.json`.

**Input:** `verify-retention-root.sh` run over: (a) the complete, undamaged
root; (b)-(i) eight copies, each with exactly one of the eight retained
artifacts from spec.md's Data-model layout deleted in turn (`package.yaml`,
`evidence/`, `transcripts/`, `entity-history.json`, `lifecycle.jsonl`,
`evaluation.jsonl`, `oracle.json`, `manifest.json` itself) — not one
representative artifact; (j) a copy with one artifact's bytes mutated so its
digest no longer matches `manifest.json`; (k) a copy where an artifact was
re-serialized (re-encoded JSON, same logical content, different bytes) rather
than byte-copied. Then repeat retention of the same (scenario, rep) against
the same root, once plainly and once with `--reclaim-incomplete`.

**Expected Output:** (a) verifies cleanly. Each of (b)-(i) fails naming the
specific missing artifact; (j) fails naming the artifact and
`digest-mismatch`; (k) fails naming the artifact and `re-serialized`. Retained
artifacts in the complete root are byte-identical (digest-equal) to their
sources. The plain repeat of an already-retained
(scenario, rep) is classified and skipped (no overwrite); the
`--reclaim-incomplete` repeat is quarantined-and-rerun, never silently
deleting the prior directory.

**Observability Evidence:** Verifier output names artifact + reason per
damaged case.

**Edge Cases:** A manifest listing a source path that no longer exists on
disk (upstream artifact moved) must fail distinctly from a digest mismatch.

**Negative Cases:** A root missing `manifest.json` entirely must fail
layout-completeness before any digest check is attempted.

---

### TC-083: Offline diagnosis of a completed-but-failed-oracle run

**Feature Requirement:** REQ-F-006.
**Acceptance Criterion:** AC-006.
**Technique Applied:** State transition + Attack-class enumeration.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Security, Maintainability.

**Caller-Path Contract:** see table above.

**Preconditions:** A fixture I-07 record whose `outcome` is a terminal
`completed` status, paired with a fixture I-08 record whose
`execution_oracle.observed_result` is `fail`, retained under a temp root with
provider/network access unavailable.

**Input:** `report-lifecycle.sh --aggregate <aggregate.json> --view headline`
after `aggregate-lifecycle.sh --retention-root <root>`, with no provider or
scenario-rerun path reachable.

**Expected Output:** The `execution_oracle` fail result, the I-07 dispatch and
gate lineage, the per-stage I-05 evidence, and the stage transcripts are all
reachable from the scenario's retention directory without rerunning the
scenario or contacting a provider. The headline view reports the workflow as
not correct (publication-ineligible per its verbatim I-08 `eligibility`), not
as a false pass inferred from terminal status alone.

**Observability Evidence:** Reachability is asserted by direct file-path
resolution from the retention directory, not by re-invoking any script.

**Edge Cases:** A record where I-07 `outcome` is terminal-completed but I-08
`eligibility.publication_eligible` is `true` despite the failed oracle is a
contract violation upstream (should not occur if I-08 is well-formed); F10
must still surface the oracle-fail evidence even if such an inconsistent
fixture were supplied, and REQ-F-008 requires F10 report the inconsistency as
an upstream contract defect rather than silently trusting either field alone.

**Negative Cases:** A missing `execution_oracle` block entirely must be
distinguished from an oracle that ran and passed — the headline must not
collapse "oracle absent" and "oracle passed" into the same rendering.

---

### TC-084: Stage-category / interval-category time reconciliation and share partition

**Feature Requirement:** REQ-F-009, REQ-F-010, REQ-F-011.
**Acceptance Criterion:** AC-007.
**Technique Applied:** Equivalence partitioning + Decision table.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Maintainability, Compatibility.

**Caller-Path Contract:** see table above.

**Preconditions:** A fixture scenario whose I-05/I-07 records carry, for every
one of the eight `stage_category` values (`discovery`, `specification`,
`planning`, `code`, `review`, `qa`, `uat`, `shipping`) crossed with every one
of the six `interval_category` values (`provider_active`, `tool_and_test`,
`queue_or_claim_wait`, `replay_or_human_gate_wait`, `retry_or_backoff`,
`unclassified`), at least one populated cell — a full 8x6 fixture matrix, not
a handful of representative cells — including at least one `code` stage with
`/stages[]/rework = true` and one `code` stage with `rework = false`.

**Input:** `aggregate-lifecycle.sh --retention-root <root>` then
`report-lifecycle.sh --view headline` and `--view stage_diagnostic`.

**Expected Output:** The stage-category (eight-cell) partition sums to
lifecycle wall time; the interval-category (six-cell) partition sums to
lifecycle wall time; each prints an explicit `unattributed` line even when
zero. The REQ-F-011 six shares (`pre_code`, `review`, `rework`,
`first_pass_code`, `wait`, `shipping`) are computed over disjoint
(`stage_category`, `interval_category`) cells per the exact rule in REQ-F-011,
sum to lifecycle wall time with no cell counted twice, and the `rework` share
reads the retained `/stages[]/rework` boolean rather than inferring rework
from gate rounds or status re-entry. The `rework` share's contributing-stage
count reconciles against I-08's `metrics.rework` rollup.

**Observability Evidence:** All three partitions' totals are asserted equal
to the independently-computed lifecycle wall time from I-07.

**Edge Cases:** "Unmappable cell" per REQ-F-011 means a *valid* `(stage_category,
interval_category)` pair — both terms drawn from the closed i05 vocabularies,
so the cell itself never fails schema validation — that REQ-F-011's explicit
share-assignment rule does not route to `pre_code`/`review`/`rework`/
`first_pass_code`/`wait`/`shipping`. Re-reading REQ-F-011's rule against the
full 8x6 matrix in Preconditions, the only such cell under the current rule is
`(shipping, unclassified)` (the rule assigns `shipping` category to the
`shipping` share regardless of interval category, so this is in fact mapped —
TC-084 must therefore also assert that REQ-F-011's rule, applied literally,
leaves **zero** genuinely unmappable cells for today's 8x6 matrix, and that
the `unattributed` line still prints explicitly at zero per REQ-F-011's "any
residual MUST be printed... even when it is zero"). To exercise a populated
`unattributed` line, inject one synthetic cell using a `stage_category` value
outside the eight-value vocabulary would be a schema violation, not a
same-vocabulary edge case — the correct synthetic edge is a `code` stage with
`rework` field *absent* on the raw fixture, which per REQ-F-011's own text
must fail loudly (see Negative Cases), not silently fall through to
`unattributed`. TC-084 therefore proves `unattributed` is a live, working line
in the report template (zero-value assertion above) without needing a
schema-invalid cell to force a nonzero value.

**Negative Cases:** A stage whose `rework` flag is absent (schema violation,
since `i07-schema.yaml` makes it required) must fail loudly, not default to
`false`. A mismatch between the `rework` share's stage count and I-08's
`metrics.rework` rollup must be reported as an upstream contract defect under
REQ-F-008, not silently resolved by trusting either side.

---

### TC-085: Review-value reporting per gate

**Feature Requirement:** REQ-F-012, REQ-F-015.
**Acceptance Criterion:** AC-008.
**Technique Applied:** Equivalence partitioning + Attack-class enumeration.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Security, Maintainability.

**Caller-Path Contract:** see table above.

**Preconditions:** I-08 fixtures covering: a gate with multiple findings
(mixed severity/defect class), an explicit zero-finding gate, a
collection-failure gate, duplicate findings, recurrent findings, confirmed
findings, unconfirmed findings, a downstream escape, one gate with a seeded
truth set, one gate with no truth set.

**Input:** `report-lifecycle.sh --view headline` (review_value block) over
the aggregate built from these fixtures.

**Expected Output:** All seven finding measures (emitted, unique, duplicate,
recurrent, confirmed, unconfirmed, downstream-escape) render broken out by
severity and defect class, each read verbatim from I-08 `review_findings` and
`metrics`, alongside elapsed time, provider cost, and resolution cost. The
truth-set gate shows precision/recall as numeric values; the no-truth-set gate
prints exactly `precision: unavailable` and `recall: unavailable` — never `0`,
an empty string, or an inferred substitute. The zero-finding gate and the
collection-failure gate render visibly distinct from each other.

**Observability Evidence:** `unavailable` rendering asserted by exact string
match, not by absence-of-output.

**Edge Cases:** A gate with zero findings AND no truth set must render both
"zero findings" and "precision/recall unavailable" simultaneously without
conflating them into one ambiguous "no data" line.

**Negative Cases:** A collection-failure gate must not silently render as
"zero findings" (that would hide a broken review run as a clean one).

---

### TC-086: Artifact-use and replayed-interaction-proxy reporting

**Feature Requirement:** REQ-F-013.
**Acceptance Criterion:** AC-009.
**Technique Applied:** Equivalence partitioning + Attack-class enumeration.
**ISO 25010 Characteristic(s):** Functional Suitability, Security,
Maintainability, Compatibility.

**Caller-Path Contract:** see table above.

**Preconditions:** A fixture with one artifact produced-and-consumed
downstream, one orphan (produced, never consumed), and replayed D01-D05
product-design proxy data (request/response counts, payload size, revision
count, unresolved gates) from I-05/I-06 lineage carried in I-08
`metrics.artifact_use`.

**Input:** `report-lifecycle.sh --aggregate <root>/aggregate.json --view
headline` (artifact_use block) over the built aggregate, plus a static scan
over the enumerated file list `bench/scripts/run-lifecycle-batch.sh`,
`run-review-comparison.sh`, `pilot-ledger.sh`, `verify-retention-root.sh`,
`aggregate-lifecycle.sh`, `report-lifecycle.sh`, `lib/spend-gate.sh`, and
every file matching `bench/reports/*.md`/`bench/reports/templates/*` (F10's
committed report-template set), grepping the closed, committed pattern list
`bench/reports/lifecycle-baseline-schema.yaml` MUST define under a
`forbidden_effort_language` key: literal case-insensitive matches for
`human minute`, `human hour`, `human effort`, `effort saved`, `time
equivalent`, `time saved`, `person-hour`, and `FTE`. The schema owns this list
(REQ-F-018) so the scan reads it rather than hard-coding a private copy.

**Expected Output:** The consumed artifact and the orphan are distinguished
with typed producer/consumer edges. Replay proxies render with counts, sizes,
revisions, and unresolved-gate counts, every field carrying a visible
replayed-proxy label. The static scan finds zero matches for any term in the
schema-owned `forbidden_effort_language` list, across the enumerated file set
above.

**Observability Evidence:** Static scan result asserted as an explicit
zero-match count per enumerated file, not "scan ran" alone.

**Edge Cases:** A proxy field whose caption is ambiguous (e.g. "Response
time") without any listed forbidden term must still carry its replayed-proxy
label per the artifact_use schema block, checked independently of the
forbidden-language scan.

**Negative Cases:** A report heading using any schema-listed forbidden term
— including the non-obvious ones (`effort saved`, `time equivalent`,
`person-hour`, `FTE`) — must fail the scan. This closed, schema-owned list is
the enumerated model the AC's open-ended "no report field... may express a
replay proxy as observed human effort" requirement resolves to; if a future
report field introduces a new human-effort synonym not yet in the schema
list, that is a schema-maintenance gap to fix at the schema (REQ-F-018), not
evidence the scan itself is incomplete.

---

### TC-087: Operator review-comparison operation (both architecture modes)

**Feature Requirement:** REQ-F-014, REQ-F-015.
**Acceptance Criterion:** AC-010.
**Technique Applied:** Decision table + One-factor-at-a-time mutation.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Security, Maintainability.

**Caller-Path Contract:** see table above.

**Preconditions:** Two retained I-08 pairs for `independent_frozen_candidate`
(one identity-compatible, one diverging in exactly one field, e.g. one
differing candidate tree digest) plus a three-candidate sequential chain
(candidate 1 -> fix -> candidate 2 -> fix -> candidate 3, each with a retained
I-08 record) for `sequential_delivery`.

**Input:** `run-review-comparison.sh` run with `--comparison-mode
independent_frozen_candidate` against each of the two pairs, and with
`--comparison-mode sequential_delivery` against the three-candidate chain
(exercising every intervening candidate, not just the first-and-last pair).

**Expected Output:** The comparator's (`compare-lifecycle-evaluations.sh`)
accept/reject verdict and divergence reasons are preserved verbatim in F10's
output — the compatible pair is accepted and published, the one-field-
divergent pair is rejected and never published, with the comparator's own
divergence reason surfaced unchanged. A branch-name-only or `HEAD`-only match
is rejected (delegated rejection, not an F10 special-case). Independent and
sequential modes render distinct candidate lineage in the output (sequential
retains every intervening candidate; independent shows the single frozen
pair).

**Observability Evidence:** Divergence reason string asserted identical to
the comparator's own output, byte-for-byte.

**Edge Cases:** A pair identical in every field except workflow-policy
identity (not candidate identity) must also reject, with the reason
distinguishing policy divergence from candidate divergence.

**Negative Cases:** F10 must not implement a second, independent identity
check — asserted by verifying F10's comparison driver contains no field-by-
field identity comparison logic of its own (static grep for candidate/policy
digest field names inside `run-review-comparison.sh` outside the delegated
call), per REQ-F-015.

---

### TC-088: Dimension separation and noise-band reporting

**Feature Requirement:** REQ-F-016.
**Acceptance Criterion:** AC-011.
**Technique Applied:** Equivalence partitioning + Boundary-value analysis +
Static scan.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Maintainability, Security.

**Caller-Path Contract:** see table above.

**Preconditions:** An aggregate with paired deltas for two metrics: one whose
delta clears its published noise band, one whose delta does not; and a
separate matrix whose rep count is below the batch policy's declared minimum.

**Input:** `report-lifecycle.sh --aggregate <root>/aggregate.json --view
headline` over the built aggregate, plus a static scan against the closed,
schema-owned `forbidden_composite_fields` list F10's schema MUST define
(literal terms: `efficiency`, `value_score`, `roi`, `composite`,
`weighted_score`, `blended`). The scan has two distinct targets with two
distinct rules: (i) `bench/reports/lifecycle-baseline-schema.yaml`'s
**field/property definitions** (the `aggregate.json` block schemas: `identity`,
`scenarios[]`, `time`, `cost`, `quality`, `review_value`, `artifact_use`,
`noise_bands`, `comparisons`, `invalid`) must contain zero properties whose
*name* matches a `forbidden_composite_fields` term — the schema's own
`forbidden_composite_fields` list declaration itself is explicitly excluded
from this check, since the list necessarily names the terms it forbids; (ii)
the enumerated F10 report-template file set (same list as TC-086) must contain
zero matches anywhere, with no exclusion.

**Expected Output:** Quality, elapsed time, and provider cost render as three
separate blocks, each showing paired deltas per dimension. The sub-band delta
renders as exactly "no detectable effect" — never as an improvement or
regression claim. The low-rep aggregate carries `insufficient_reps: true`. The
static scan finds zero forbidden-term property names in the aggregate schema
(target (i)) and zero forbidden-term matches anywhere in report templates or
rendered output (target (ii)).

**Observability Evidence:** Static scan asserted as an explicit zero-match
result over both the schema file and the report templates (not the report
output text alone — closes the Ambiguity Finding noted above).

**Edge Cases:** spec.md does not define a noise-band boundary-inclusivity
rule (inside vs. exactly-on-the-boundary), so TC-088 does not assert a
specific resolution for an exact-boundary delta — asserting one here would
create a test oracle the spec does not authorize. Instead, TC-088 requires
`bench/reports/lifecycle-baseline-schema.yaml` to state the boundary rule
explicitly (open vs. closed interval) as part of its noise-band derivation-rule
documentation (REQ-F-018), and TC-078 (schema contract test) is the place that
then locks the stated rule down; a task-decomposition follow-up must confirm
this schema field exists before TC-088 is implemented, or flag it as an
upstream question rather than silently picking a rule inside the test.

**Negative Cases:** A report that renders a sub-band delta as "improved by
X%" (a plausible bug: reporting the raw delta regardless of band) must fail
this test.

---

### TC-089: Phase separation from completed Phase 1 tooling

**Feature Requirement:** REQ-F-017.
**Acceptance Criterion:** AC-012.
**Technique Applied:** Regression testing + Equivalence partitioning.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Compatibility, Portability.

**Caller-Path Contract:** see table above.

**Preconditions:** A Phase 1 `record.jsonl` fixture (valid v1 shape) and a git
checkout of `bench/scripts/run-batch.sh`, `aggregate-runs.sh`,
`report-baseline.sh` at `HEAD` before F10 development begins.

**Input:** `aggregate-lifecycle.sh --retention-root <root-containing-v1-record.jsonl>`;
separately, run the unmodified `run-batch.sh` / `aggregate-runs.sh` /
`report-baseline.sh` path end-to-end; then diff those three files against
the commit SHA recorded when F10 task-decomposition begins (recorded once, in
this test's fixture setup, as `PRE_F10_BASELINE_SHA` — not a moving `HEAD`,
which would silently pass after later, unrelated Phase-1-touching commits):
`git diff $PRE_F10_BASELINE_SHA -- bench/scripts/run-batch.sh
bench/scripts/aggregate-runs.sh bench/scripts/report-baseline.sh`.

**Expected Output:** The lifecycle aggregator refuses the v1 record with a
named reason (not a crash, not silent coercion). The Phase 1 path still runs
successfully end-to-end and is byte-unmodified per the `git diff` (empty
diff) against the pinned baseline SHA. Every F10 artifact and report carries
an explicit lifecycle-v2 phase label, and no Phase 1 (v1) output anywhere is
relabelled as lifecycle v2.

**Observability Evidence:** `git diff` output asserted empty for the three
named Phase 1 scripts against `PRE_F10_BASELINE_SHA`.

**Edge Cases:** spec.md's AC-012 and REQ-F-017 specify refusal of a Phase 1
`record.jsonl` input but do not define behavior for a retention root
containing *both* a v1 `record.jsonl` and valid v2 pairs side by side. TC-089
does not assert a specific resolution for that mixed case — doing so would
invent a rule the spec does not authorize. Instead, TC-089 records this as an
explicit open question for task-decomposition to resolve (either "refuse the
whole root" or "refuse the v1 file and aggregate the v2 pairs", named
explicitly in the task spec) before the mixed-root behavior is implemented or
tested.

**Negative Cases:** A phase label accidentally applied to Phase 1 output (a
plausible regression if F10 shares code with the v1 path) must fail this
test — asserted by scanning v1 output files for the `lifecycle_v2` label
string.

---

### TC-090: Offline determinism and 100 MB scale bound

**Feature Requirement:** REQ-NF-002, REQ-NF-003, REQ-NF-004.
**Acceptance Criterion:** AC-013.
**Technique Applied:** Retest/reproducibility testing + Boundary-value
analysis.
**ISO 25010 Characteristic(s):** Functional Suitability, Performance,
Reliability, Security, Portability.

**Caller-Path Contract:** see table above.

**Preconditions:** A provider-denial, network-denial, live-Shark-DB-denial,
and live-working-tree-write-denial harness. The committed 100 MB synthetic
retention fixture referenced in REQ-NF-004.

**Input:** `aggregate-lifecycle.sh --retention-root <root>` and
`report-lifecycle.sh --aggregate <root>/aggregate.json --view headline` and
`--view stage_diagnostic`, each run twice under the denial harness against the
same retention root, including once against the 100 MB fixture with
peak-RSS measurement via `/usr/bin/time -v` (`Maximum resident set size`).

**Expected Output:** Both runs produce byte-identical `aggregate.json` and
report markdown output. Zero denied calls occur across all four denial
surfaces (provider, network, DB, live tree). The 100 MB fixture run completes
within 60 seconds on the repository CI runner (REQ-NF-004). Peak RSS for the
100 MB fixture run does not exceed 25 MB (25% of total fixture size) —
a concrete, numeric proxy for "streamed, not fully loaded"; if the largest
single retained file in the fixture exceeds 25 MB, the bound is instead the
larger of 25 MB or (largest single file size + 5 MB), so a single large file
read whole under a documented exception is still distinguished from loading
the entire 100 MB retention tree into memory.

**Observability Evidence:** Peak-RSS measurement asserted numerically against
the 25 MB (or largest-file-plus-5-MB) bound above, not just "completed".

**Edge Cases:** A retention root with a very large single evidence file
(e.g. a multi-MB transcript) within the 100 MB fixture must not cause a
memory spike proportional to that one file's size.

**Negative Cases:** Any timestamp appearing in report output must trace to a
retained-record field, not `date`/`time.Now()` — asserted by injecting a
distinguishable fake system clock during the test run and confirming report
output is unaffected.

---

### TC-091: Static safety and scope-boundary scan

**Feature Requirement:** REQ-NF-001, REQ-NF-005, REQ-NF-006.
**Acceptance Criterion:** AC-014.
**Technique Applied:** Attack-class enumeration (static).
**ISO 25010 Characteristic(s):** Security, Maintainability, Portability,
Compatibility.

**Caller-Path Contract:** see table above.

**Preconditions:** All committed F10 scripts and the F10 schema file.

**Input:** Four enumerated, mechanically-checkable static scans over the file
list `bench/scripts/run-lifecycle-batch.sh`, `run-review-comparison.sh`,
`pilot-ledger.sh`, `verify-retention-root.sh`, `aggregate-lifecycle.sh`,
`report-lifecycle.sh`, `lib/spend-gate.sh`, and
`bench/reports/lifecycle-baseline-schema.yaml`:

(a) **Write-path scan:** every shell redirection (`>`, `>>`, `tee`, `cp
... <dest>`, `mv ... <dest>`, `mkdir`) whose destination argument is a
variable is traced to its assignment; the scan asserts every such variable is
either a literal constant under a path prefixed by the script's
`$RETENTION_ROOT` (or `$root`/`$retention_root`, the script's own declared
name for the `--retention-root` value) or is itself derived, by direct
assignment chain, from that flag's parsed value — no write-destination
variable may originate from an unrelated input (a scenario field, a fixture
path, or an unvalidated argument).

(b) **Forbidden-path scan:** a literal grep for the path-prefixes `internal/`,
`cmd/`, `.sharkconfig.json`, and `migrations/` anywhere in the seven files
above; zero matches required.

(c) **Language-branch scan:** a literal grep for `python`, `\.py\b`, `go run`,
`go build`, `go test`, and `\.go\b` outside the single sanctioned call site
that invokes the I-04-registered adapter executable (named explicitly in each
script's `--scenario <package.yaml>` handling); any other match is a
violation.

(d) **Content-disclosure scan:** grep for the credential-pattern list
`sk-[A-Za-z0-9]{20,}`, `AKIA[0-9A-Z]{16}`, `ANTHROPIC_API_KEY`,
`OPENAI_API_KEY`, `TURSO_AUTH_TOKEN`; grep for the literal directory names
`evaluator_only` and `prompt_body`/`rendered_prompt` appearing as a *read
source feeding a write into* the retention root or a report (a script merely
referencing the isolation-guard's own root-name constant, without reading its
contents, does not violate this); and grep for any file-read call
(`cat`, `read`, shell `<`) against a transcript file that lacks an adjacent
size-cap check (`wc -c`, `head -c`, `du`) before the read.

**Expected Output:** Zero matches/violations for all four scans (a)-(d) as
defined above. This enumerated model is F10's concrete resolution of REQ-NF-001/
REQ-NF-005/REQ-NF-006's broader prose; any behavior outside this enumerated
model (e.g. a genuinely novel disclosure vector not covered by (d)'s pattern
list) is a documented scope limit of TC-091, to be closed by extending the
enumerated list at task-decomposition or code-review time, not treated as
silently covered by this test.

**Observability Evidence:** Each of the four scans reports an explicit
zero-match count with the scanned file list enumerated (proving the scan
covered every committed F10 script, not a partial set).

**Edge Cases:** A write-destination variable built by string concatenation
across two assignment statements (not a single line) must still be traced by
scan (a)'s assignment-chain walk.

**Negative Cases:** A script that logs a rendered-prompt-body variable to
stderr "for debugging" is caught by scan (d)'s read-then-output tracing, not
only by a file-write check — stdout/stderr are treated as output surfaces
equally with file writes.

---

### TC-092: Full regression and complete F10 suite registration

**Feature Requirement:** All (final release gate).
**Acceptance Criterion:** AC-015.
**Technique Applied:** Regression testing.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability,
Maintainability, Compatibility, Portability.

**Caller-Path Contract:** see table above.

**Preconditions:** `bench/scripts/tests/run-all.sh` currently registers
TC-003 through TC-077 (F01-F09). At F10 task-decomposition time, this test's
fixture setup commits `bench/scripts/tests/testdata/pre-f10-registration.txt`
— the literal, ordered list of registered test names extracted from
`run-all.sh` before any F10 change, one name per line. This file is the
concrete baseline artifact TC-092 diffs against; it is not regenerated after
F10 lands.

**Input:** `make fmt && make lint && make test`, then
`bench/scripts/tests/run-all.sh` with TC-078 through TC-092 newly registered
in deterministic order.

**Expected Output:** Repository quality gates (`make fmt`, `make lint`,
`make test`) pass with zero failures. The complete F10 suite (TC-078 through
TC-092) executes and passes. Every name in
`pre-f10-registration.txt` is still present, in the same relative order, in
the post-F10 `run-all.sh`, and each still executes and produces its
documented prior pass marker (the marker string each pre-F10 test itself
prints on success, not merely a nonzero exit code from `run-all.sh` as a
whole) — this is the concrete, enumerable resolution of "not removed, skipped,
or weakened."

**Observability Evidence:** A line-by-line diff of `pre-f10-registration.txt`
against the post-F10 `run-all.sh` registration list is asserted empty
(modulo ordinal position shift from new insertions, not name removal), and
each pre-F10 test's documented pass-marker string is asserted present in the
post-F10 run's captured output.

**Edge Cases:** A test renamed rather than removed (e.g. TC-061 renamed to
something else while keeping the same body) must still be caught by name-based
enumeration, not exit-code counting alone.

**Negative Cases:** A `run-all.sh` change that silently reduces parallelism or
skips a slow test under a new flag (e.g. `--fast`) without that flag being the
one `make test` actually invokes must fail this test.

## Test Infrastructure

### Existing patterns to follow

- `tests/contracts/e40_i08_lifecycle_evaluation_contract_test.go` and its
  `testdata/e40_i08/{valid,invalid}/` layout — the direct structural model for
  TC-078's new `e40_f10_operator_baseline_contract_test.go` and
  `testdata/e40_f10/{valid,invalid}/`.
- `bench/scripts/run-batch.sh`'s `--dry-run` short-circuit (`dispatch_pair()`
  zero-subprocess path) and classification discipline (`skipped_complete`,
  `incomplete_prior_attempt`, `quarantined_and_rerun`, `pending_run`,
  `failed`) — the structural model for `run-lifecycle-batch.sh`'s preview mode
  and TC-082's retention-repeat classification.
- `bench/scripts/run-lifecycle.sh`'s `parse_limits()` positive-ceiling check
  and `--mode dry-run` — TC-080 and TC-079 must prove F10 delegates to these
  rather than reimplementing them.
- `bench/scripts/report-baseline.sh` and `bench/scripts/aggregate-runs.sh`
  header purity contracts ("pure function... consults no clock, invokes no
  subprocess") — the direct model for TC-090's determinism assertions on
  `aggregate-lifecycle.sh`/`report-lifecycle.sh`.
- `bench/scripts/compare-lifecycle-evaluations.sh`'s fail-closed
  `MODES`/candidate/policy digest check — TC-087 must prove F10 delegates to
  this unchanged rather than re-implementing identity comparison.
- `bench/scripts/tests/run-all.sh` registration pattern used by TC-060
  through TC-077 — the direct model for registering TC-078 through TC-092.

### New fixtures/helpers required

- `bench/reports/lifecycle-baseline-schema.yaml` and
  `tests/contracts/testdata/e40_f10/{valid,invalid}/` (TC-078). The schema
  MUST additionally define the `forbidden_effort_language` key (TC-086) and
  the `forbidden_composite_fields` key (TC-088) as closed, versioned lists —
  not private copies inside test scripts — per REQ-F-018.
- A PATH-shim provider/network-denial harness reusable across TC-079,
  TC-080, and TC-090, backed by one committed, enumerated binary list (every
  provider CLI and network tool reachable from `run-lifecycle-batch.sh`,
  `run-review-comparison.sh`, and the `run-lifecycle.sh`/
  `evaluate-lifecycle.sh` scripts they delegate to) rather than three
  divergent implementations or an unenumerated wildcard shim.
- `bench/scripts/tests/testdata/pre-f10-registration.txt` (TC-092): the
  literal ordered `run-all.sh` test-name list committed once at F10
  task-decomposition time, and the `PRE_F10_BASELINE_SHA` constant (TC-089)
  recorded the same way — both are fixed baselines, never regenerated after
  F10 lands, so later drift is caught rather than silently re-baselined.
- Fixture I-05/I-07/I-08 triples covering: replayed-vs-live stages
  (TC-079), missing/mismatched/re-serialized retained artifacts (TC-082),
  completed-workflow-with-failed-oracle (TC-083), the full time-partition
  matrix including at least one `rework=true` and one `rework=false` `code`
  stage (TC-084), the seven-finding-measure matrix with and without a truth
  set (TC-085), producer/consumer/orphan artifact and D01-D05 replay proxy
  data (TC-086), identity-compatible and one-field-divergent I-08 pairs in
  both comparison modes (TC-087), noise-band-clearing and sub-band paired
  deltas plus a below-minimum-reps matrix (TC-088).
- A committed Phase 1 `record.jsonl` fixture for TC-089's refusal case.
- The committed 100 MB synthetic retention fixture referenced by REQ-NF-004
  for TC-090 (shared with F09's precedent per the research report's citation
  of "F09 REQ-NF-004 precedent" — confirm whether F09's existing fixture is
  directly reusable or F10 needs its own retention-shaped variant before
  task decomposition).
- A static-scan helper (grep/AST-style) reusable across TC-086 (human-effort
  framing), TC-088 (composite-score absence), and TC-091 (write-path/
  language-branch/credential scan) rather than three divergent scanners.
- No repository-level Go test in this feature touches the live Shark
  database; TC-078 is the only Go contract test and it is file/schema-backed
  only (consistent with CLAUDE.md's "repo tests use real DB, everything else
  uses mocks" rule — F10 has no service/repository layer, so neither applies,
  and TC-078 correctly uses committed fixtures rather than either a real DB
  or a mock).

## Recommendations

- [x] Ready for development (post-remediation codex red-team returned PASS
  below)
- [ ] Needs BA refinement
- [ ] Needs tech refinement

## Codex Test-Plan Red-Team

**Final verdict (round 3):** PASS
**Issues raised across three rounds:** 8 round-1 blockers + 5 round-1
non-blocking findings + 3 round-2 blockers (introduced by round-1's own
remediation) = 16 total findings
**Issues addressed:** 14, remediated below and re-verified by codex
**Issues deferred:** 2 explicit open task-decomposition questions (not
silently resolved): TC-088/TC-078's noise-band boundary-inclusivity rule, and
TC-089's mixed v1/v2 retention-root behavior

The initial review found: (1) AC-002's zero-call proof lacked an enumerated
provider/network binary model — fixed by naming the committed binary list in
Test Infrastructure and TC-079's caller-path row. (2) AC-004/TC-081 contained
a genuine contract contradiction with REQ-F-005 (whole-command refusal vs.
per-family dispatch) — fixed by rewriting TC-081 as three single-family
invocations plus a fourth multi-family confirmation. (3) AC-007/TC-084's
"future unmappable cell" edge case conflicted with the closed i05 vocabularies
— fixed by clarifying "unmappable" means share-assignment gap, not vocabulary
violation, walking REQ-F-011's rule against the full 8x6 matrix, and replacing
the invalid synthetic cell with a schema-valid `unattributed`-at-zero
assertion. (4) AC-009/TC-086's human-effort-language scan was lexical-only
against an open-ended claim — fixed by making the forbidden-term list
schema-owned (`forbidden_effort_language`) rather than test-private. (5)
AC-011/TC-088 asserted an undefined noise-band boundary-inclusivity rule —
removed; replaced with a requirement that the schema state the rule
explicitly, deferred to TC-078/task-decomposition rather than invented here.
(6) AC-013/TC-090's "well below 100 MB" was not a numeric bound — fixed with
a concrete 25 MB (or largest-file+5MB) peak-RSS bound. (7) AC-014/TC-091's
static-scan claims were open-ended robustness assertions — fixed by
enumerating four concrete, mechanically-checkable scans (write-path
assignment-chain tracing, forbidden-path grep, language-branch grep,
content-disclosure pattern list) with an explicit documented-scope-limit
statement for anything outside them. (8) AC-015/TC-092's "not removed,
skipped, or weakened" lacked a concrete baseline — fixed with a committed
`pre-f10-registration.txt` snapshot and per-test pass-marker assertion.
Non-blocking findings addressed: missing `--aggregate` flag in several
Caller-Path Contract rows (TC-084, TC-090) and body text; missing spend-gate
arguments in TC-080/TC-081/TC-087 caller-path rows; weak ISO Usability `N/A`
justifications for AC-006/AC-007 (now non-N/A with a stated rationale);
thin AC-005/AC-010 enumeration (TC-082 now covers all eight retained
artifacts individually; TC-087 now includes a three-candidate sequential
chain); `HEAD`-relative diffs in TC-089/TC-092 replaced with pinned baselines
(`PRE_F10_BASELINE_SHA`, `pre-f10-registration.txt`) so later unrelated
commits cannot silently pass a regression check.

### Second-round Codex output (verbatim)

```text
FAIL

1. TC-088 scans the schema for forbidden composite terms while requiring that same schema to define them, making its zero-match assertion impossible (test-plan.md:692).
2. The ISO matrix still says TC-089 diffs against moving `HEAD`, contradicting the pinned `PRE_F10_BASELINE_SHA` fix (test-plan.md:81).
3. TC-089 introduces an unspecified mixed v1/v2 "refuse-vs-partial-aggregate" rule not defined by the spec (test-plan.md:761).
```

All three second-round findings were addressed: TC-088's scan now excludes
the schema's own `forbidden_composite_fields` list declaration and splits
into two explicit targets (schema property names vs. report-template text) so
the assertion is no longer self-contradictory; the ISO matrix's AC-012 row now
cites the pinned `PRE_F10_BASELINE_SHA` instead of `HEAD`; TC-089's
mixed-root case now records an explicit open task-decomposition question
instead of asserting an unauthorized rule.

### Third-round Codex output (verbatim)

```text
PASS

1. TC-088 correctly separates schema-property and report-template scans, excluding only the schema's list declaration.
2. ISO AC-012 now references `PRE_F10_BASELINE_SHA`.
3. TC-089 treats mixed v1/v2 retention behavior as an open task-decomposition question.
4. No additional blockers found. No builds or tests run.
```

Verdict across all three rounds: FAIL (8 blockers) -> FAIL (3 blockers,
all newly introduced by round-1 remediation) -> **PASS**. Total distinct
issues raised: 11. Addressed: 11. Deferred: 2 explicit open
task-decomposition questions (TC-088/TC-078's noise-band boundary-inclusivity
rule; TC-089's mixed v1/v2 retention-root behavior) — both are documented in
their respective test cases as spec gaps to close before implementation, not
silently assumed.
