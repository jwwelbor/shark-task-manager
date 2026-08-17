# Test Plan: E40-F09 - Calibrated evaluation and comparison identity

**Created:** 2026-08-17
**Feature PRD:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F09-calibrated-evaluation-and-comparison-identity/feature.md`
**Task Spec:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F09-calibrated-evaluation-and-comparison-identity/spec.md`
**Parent UAT Plan:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md`
**Status:** APPROVED

## Spec Drift Analysis

### Drift Findings

- `spec.md` refines the feature boundary with the I-08 schema, offline shell
  entrypoints, exact contract-test pointers, and invalidity vocabulary. These
  are implementation detail and verification refinements, not scope drift.
- The feature and task specifications agree that F09 consumes immutable I-05
  and I-07 artifacts, owns evaluation/finding adjudication/identity/eligibility,
  and does not add a database table, provider runtime, workflow engine, or
  publication surface.
- The spec explicitly narrows direct cross-feature interactions to I-05, I-07,
  and I-08 and direct cross-epic integration to X-12. It correctly carries I-04
  and I-06 identity through upstream artifacts rather than inventing new edges.
- No unresolved scope, semantic, schema, or conversion drift was found.

### Traceability Matrix

| Feature/architecture requirement | Task AC | Covered? | Notes |
|---|---|---:|---|
| I-08 schema and one joined evaluation record | AC-001, AC-002 | Yes | TC-067 is the shared contract pointer; TC-067/TC-068 exercise the real evaluator. |
| Separate structural, judge, and execution-oracle truth | AC-002, AC-003 | Yes | Terminal status and exit code substitution are negative cases. |
| Held-back oracle ordering and isolation | AC-003 | Yes | Pre-terminal and failed-access paths require zero adapter calls and cleanup. |
| Calibrated, disjoint judge evidence | AC-004 | Yes | Missing/mutated rubric, judge, usage, and overlap cases are enumerated. |
| Complete evaluation, candidate, and workflow-policy identity | AC-005, AC-010 | Yes | One-field mutation matrix and X-12 content mutations reject pairs. |
| Raw and normalized review findings | AC-006 | Yes | Reviewer fields remain evidence; confirmation is independent. |
| Independent and sequential comparison modes | AC-007 | Yes | Frozen candidate and intervening lineage are separately asserted. |
| Deterministic offline replay and invalid retention | AC-008, AC-009 | Yes | Provider/network/DB/live-tree writes are denied. |
| File-backed, bounded, language-neutral implementation | AC-011, AC-012 | Yes | Static scan, 100 MB performance fixture, adapter matrix, and full gates. |
| Quality/time/cost remain separate | AC-006, AC-007, AC-009 | Yes | No composite efficiency score is accepted. |

## Acceptance Criteria Review

### Ambiguity Findings

None. Each AC names a test pointer, input family, expected result, or command
and the architecture supplies the missing cross-component semantics. “Complete”
identity is enumerated field-by-field in AC-005 and the requirements it cites.

### Missing Coverage

None. UAT-13 is covered by TC-067/TC-068, UAT-14 by TC-070/TC-074/TC-075,
UAT-17 by TC-071/TC-072, and UAT-19 by TC-070. The feature’s only direct
cross-epic row, X-12, is covered by TC-075 and the shared TC-067 contract.

## ISTQB Technique Application (per AC)

The plan uses `contract-surface enumeration` as equivalence partitioning over
schema and command fields, `attack-class enumeration` for fail-closed and
disclosure threats, `boundary-value analysis` for terminal/size/performance
boundaries, `decision-table testing` for eligibility and comparison modes, and
`state-transition testing` for oracle ordering and independent/sequential
lineage. The specific partitions are listed in each test case; no AC is left
with an unenumerated robustness assertion.

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-001 | Equivalence partitioning + contract-surface enumeration + BVA | TC-067 | Required fields, types, vocabularies, digest forms, unknown fields, and malformed paths are schema partitions. |
| AC-002 | Decision table + equivalence partitioning | TC-067 | Structural/judge/oracle applicability and missing-truth substitutions form mutually exclusive truth rows. |
| AC-003 | State transition + attack-class enumeration | TC-068 | Pre-terminal, terminal, authorized, denied, hidden-file, and cleanup states are ordered transitions. |
| AC-004 | Equivalence partitioning + BVA + decision table | TC-069 | Disjoint/overlapping sets, present/missing/malformed provenance, and applicability determine eligibility. |
| AC-005 | Contract-surface enumeration + one-factor-at-a-time mutation + decision table | TC-070 | Every named identity field and branch/HEAD-only shortcut is mutated independently. |
| AC-006 | Equivalence partitioning + attack-class enumeration | TC-071 | Empty, duplicate, recurrent, seeded, clean-control, and unconfirmed findings are distinct classes. |
| AC-007 | State transition + decision table | TC-072 | Frozen independent and candidate-changing sequential paths have different attribution rules. |
| AC-008 | Retest/reproducibility testing + attack-class enumeration | TC-073 | Same retained inputs, denied egress, canonical JSON, and no scenario rerun define the replay partition. |
| AC-009 | Equivalence partitioning + decision table | TC-074 | Each invalidity class is retained and excluded; valid uniform records are the control. |
| AC-010 | Contract-surface enumeration + one-factor mutation | TC-075 | Workflow, prompt, skill, and agent content mutations must alter one installed-bundle digest. |
| AC-011 | Structural/configuration testing + compatibility testing + BVA | TC-076 | Static forbidden-path/language scans and the 100 MB/30-second boundary prove non-functional constraints. |
| AC-012 | State-transition/retest + regression testing | TC-077 | Registered suite, full repository gates, and preservation of F01-F08 tests are release conditions. |

## ISO 25010 Coverage Matrix

| AC | Functional suitability | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | TC-067 schema and path diagnostics | N/A: no AC latency target | TC-067 reads committed YAML/JSON | TC-067 checks actionable field paths for operators | TC-067 rejects malformed/partial records | TC-067 rejects secrets, copied evaluator payloads, and invalid digests | TC-067 makes schema the vocabulary owner | TC-067 uses the repository’s Go contract harness |
| AC-002 | TC-067/068 assert three independent truth blocks | N/A: truth separation is not a latency SLA | TC-067/068 use file-backed I-05/I-07 shapes | N/A: internal artifact contract; bounded reasons are the interface | TC-068 rejects terminal/exit-code substitution | TC-068 prevents false implementation truth | TC-067/068 catch schema and evaluator drift | TC-068 uses POSIX shell and adapter contract fixtures |
| AC-003 | TC-068 asserts ordering, access event, oracle, cleanup | TC-068 records zero-call pre-terminal boundary | TC-068 invokes existing I-05 guard and I-04 adapter | N/A: machine-only safety gate | TC-068 covers denied access, failed cleanup, and no hidden residue | TC-068 scans both agent-visible roots and traversal/symlink attacks | TC-068 verifies reuse of the existing guard | TC-068 uses file-backed roots and no live DB |
| AC-004 | TC-069 validates calibrated evidence and eligibility | N/A: no judge latency requirement; usage/cost retained | TC-069 validates model/configuration identity fields | N/A: supplied judge artifact, not interactive UX | TC-069 makes missing calibration ineligible without a default | TC-069 verifies held-out/calibration disjointness and bounded evidence | TC-069 pins rubric/prompt/schema provenance | TC-069 uses deterministic JSON fixtures |
| AC-005 | TC-070 rejects every one-field identity mutation | N/A: comparator has no stated latency target | TC-070 hashes installed content and deep-review files | N/A: comparison is an internal artifact | TC-070 rejects incomplete or mixed identities | TC-070 catches content/policy drift and branch/HEAD shortcuts | TC-070 reads the declared bundle rather than duplicating its list | TC-070 uses canonical lowercase SHA-256 on repository files |
| AC-006 | TC-071 preserves raw and normalized findings and counts | N/A: normalization target is determinism, not SLA | TC-071 consumes I-07 review-gate shape and note fields | N/A: downstream reporting owns presentation | TC-071 distinguishes zero, failed, unreached, and unconfirmed | TC-071 prevents reviewer-supplied disposition from becoming truth | TC-071 keeps F09 identity separate from raw note schema | TC-071 is offline and file-backed |
| AC-007 | TC-072 asserts both comparison semantics and lineage | N/A: no comparison latency target | TC-072 consumes declared mode vocabulary | N/A: no end-user presentation in F09 | TC-072 retains every sequential candidate and rejects causal inference | TC-072 prevents fix erasure and misleading gate attribution | TC-072 keeps mode rules explicit and testable | TC-072 uses portable JSONL fixtures |
| AC-008 | TC-073 asserts byte-identical verdict/inventory and no rerun | TC-076 measures validator completion separately | TC-073 varies locale/key/filesystem ordering | N/A: offline reproducibility contract | TC-073 denies provider/network calls and live DB/tree writes | TC-073 rejects transcript/reference/credential copying | TC-073 verifies canonical serialization and replay lineage | TC-073 uses repository shell plus Go contract tests |
| AC-009 | TC-074 retains every invalid record and excludes it from aggregation | N/A: aggregation has no new performance SLA | TC-074 uses existing aggregate-runs interface | N/A: machine diagnostics are the operator surface | TC-074 distinguishes invalid inventory from eligible aggregate | TC-074 ensures failed evidence cannot be silently averaged | TC-074 reuses invalid-retention discipline without changing v1 | TC-074 is file-backed and database-free |
| AC-010 | TC-075 changes digest on each X-12 mutation | N/A: digest calculation has no feature SLA | TC-075 covers workflows, prompts, skills, and agents | N/A: internal operator evidence is handled by F10 | TC-075 rejects mixed installed bundles | TC-075 prevents cross-content comparison | TC-075 uses E32 canonical-content shape, not a second API | TC-075 uses deterministic tree hashing |
| AC-011 | TC-076 verifies bounded artifact and language-neutral behavior | TC-076 requires the committed 100 MB fixture within 30 seconds | TC-076 covers Python/Go adapter declarations without language branches | N/A: static/runtime quality gate, no product UX | TC-076 scans for evaluator-only leakage and forbidden writes | TC-076 checks no credentials/transcripts in I-08 | TC-076 scans maintainability boundaries and forbidden product files | TC-076 uses the repository CI shell environment |
| AC-012 | TC-077 runs full quality and F09 suites and protects prior tests | TC-077 records suite completion and failures | TC-077 runs Go and shell test surfaces | N/A: development quality gate | TC-077 rejects removed/weakened F01-F08 coverage | TC-077 keeps provider/network/DB protections active | TC-077 registers tests deterministically | TC-077 uses documented `make` and shell commands |

## Observability Design (per behavior)

F09 adds no E23 product-observability row. Its runtime evidence is the retained
benchmark artifact and bounded stderr diagnostics, as required by the epic
maps. These are hard implementation requirements and QA assertions.

| Behavior | Metric | Log | Trace/span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| I-05/I-07 join and evaluation | `eligibility.aggregate_eligible` and `eligibility.publication_eligible` in `bench/evaluations/<evaluation_id>/evaluation.jsonl` | Mandatory bounded stderr `evaluation_invalid` with evaluation ID, ordered reason code, source refs | Retained `identity`/`source_artifacts` fields; no optional trace required | Any missing/duplicate/contradictory join | TC-067/074 assert exact path, one record, named reason, and inventory retention |
| Held-back oracle ordering | `execution_oracle.observed_result` and `execution_oracle.access_event` in the same evaluation artifact | Mandatory bounded `oracle_access` with terminal status, access event, adapter, result | Retained `execution_oracle` block; no optional trace required | Any pre-terminal or unauthorized adapter call | TC-068 asserts zero/one calls, access event, result, pre-dispatch scans, and cleanup |
| Calibrated judge provenance | `judge.applicability`, `judge.usage`, and `judge.cost` in `evaluation.jsonl` | Mandatory bounded `judge_invalid` with missing/overlap/malformed field | Retained `judge` block; no optional trace required | Any score without calibration or judge identity | TC-069 asserts provenance, usage, cost, and no fabricated score |
| Identity comparison | `comparison.accepted` / `comparison.rejected` and divergence list | Mandatory `comparison_divergence` with exact field and both digests | Retained `comparison` and `identity` blocks; no optional trace required | Any acceptance after one-factor mutation | TC-070/075 assert exact divergence and rejected pair |
| Finding normalization | `review_findings.derived_counts` in `evaluation.jsonl` | Mandatory `finding_adjudication` with gate, identity, source, disposition | Retained `review_findings` block; no optional trace required | Any raw disposition treated as confirmed truth | TC-071 asserts raw preservation and independent normalized fields |
| Replay and invalid retention | `eligibility.invalidity_reasons` and retained evaluation count | Mandatory `evaluation_replay` and `aggregate_exclusion` with stable IDs | Retained evaluation artifact; no optional trace required | Any provider call, rerun, dropped invalid record, or aggregate inclusion | TC-073/074 assert byte identity, zero egress, and retained exclusions |

Pure canonical digest helpers have `internal — no independent observability`;
their output is covered by mandatory `identity`, `comparison`, and source
artifact fields. No optional trace or metric is an implementation escape hatch.

## Cross-feature contract tests (I-##)

The staged values below are copied exactly from `E40-interaction-map.md` and
the feature spec. Counterpart status is not copied into this plan; the parent
loop must read it live from Shark at review/UAT time.

| I-## | Producer | Consumer | Shape source | Gate mode / activation / closure | Review basis / disposition | Shared test pointer | TC |
|---|---|---|---|---|---|---|---|
| I-05 | E40-F06 | E40-F08/F09/F10 | `architecture.md#stage-evidence-and-isolation-contract` | `contract-only`; activation owners E40-F08, E40-F09, E40-F10; closure keys E40-F08, E40-F09, E40-F10 at each UAT; counterpart status live | F06 completed `spec.md` + map row; `pending-integration` | `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042` | TC-067 (F09 slice; full owner set asserted) |
| I-07 | E40-F08 | E40-F09/F10 | `architecture.md#lifecycle-run-record-contract` | `contract-only`; activation owners E40-F09, E40-F10; closure keys E40-F09, E40-F10 at each UAT; counterpart status live | F08 completed `spec.md` + map row; `pending-integration` | `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061` | TC-067 (F09 slice; full owner set asserted) |
| I-08 | E40-F09 | E40-F10 | `architecture.md#lifecycle-evaluation-record-contract` | `contract-only` until F10 proves live use; activation owner E40-F10; closure key E40-F10 at its UAT; counterpart status live | F09 completed `spec.md` + map row; `pending-integration` | `tests/contracts/e40_i08_lifecycle_evaluation_contract_test.go#TC-067` | TC-067 |

These are one shared proof per contract, not twin tests. The contract test must
assert the shape source, exact pointer, `gate_mode`, activation owner, closure
key, live status read, review basis, and `pending-integration` disposition. TC-067
must read real committed schemas and fixtures rather than a hand-built object.

## Cross-epic integration tests (X-##)

| X-## | Boundary | Verification | Test coverage pointer | TC |
|---|---|---|---|---|
| X-12 | E32-F04 canonical installed Shark-data content -> E40-F09 | Hash the installed workflows, prompts, skills, and agents; mutate one file in each class; reject the paired comparison with both differing content digests and never report a quality delta | `tc075_content-identity_x12_test.sh`, `tests/contracts/e40_i08_lifecycle_evaluation_contract_test.go#TC-067`, UAT-14 | TC-075 and TC-067 |

No other X-## is declared by the feature. X-09 provider usage reaches F09 via
I-05 and is not redefined here.

## Caller-Path Contracts (per test case)

| TC | Production entrypoint and argument shape | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-067 | `go test ./tests/contracts -run TestTC067_I08LifecycleEvaluationContract`; test reads `bench/evaluation/i08-schema.yaml` and `tests/contracts/testdata/e40_i08/{valid,invalid}` via real filesystem/parser | None; committed schema/fixtures are the seam | Do not mock schema loading, field enumeration, canonical JSON, or validator | A hand-built in-memory record would stay green after the real schema or fixture is broken. |
| TC-068 | `bench/scripts/evaluate-lifecycle.sh --i05 <bundle> --i07 <lifecycle.jsonl> --scenario <package.yaml> --output <evaluation.jsonl> --judge-result <judge.json>` invokes the nested `bench/scripts/run-heldback-oracle.sh --scenario <package.yaml> --i07 <lifecycle.jsonl> --stage-bundle <bundle> --checkout <dir> --output <oracle.json>` path | Only the registered I-04 adapter executable may be a deterministic fixture; use the real I-05 guard and filesystem roots | Do not mock evaluator argument construction, terminal-status validation, evaluator-access authorization, cleanup, or provider/network denial | A missing evaluator flag, pre-terminal, or unauthorized implementation would invoke the adapter or leave hidden material. |
| TC-069 | `bench/scripts/evaluate-lifecycle.sh --i05 <bundle> --i07 <lifecycle.jsonl> --scenario <package.yaml> --output <evaluation.jsonl> --judge-result <judge.json>` | None above file-backed judge input; provider calls remain denied | Do not mock judge provenance validation, set disjointness, usage parsing, or eligibility | A missing/overlapping calibration set would receive a fabricated score or eligible verdict. |
| TC-070 | `bench/scripts/compare-lifecycle-evaluations.sh --left <evaluation.jsonl> --right <evaluation.jsonl> --mode <mode> --output <comparison.json>` | None; mutate real temporary copies of retained JSONL and repository bundle files | Do not mock digest calculation, field projection, mode validation, or branch/HEAD rejection | A comparator using branch/HEAD only would accept one mutated identity field. |
| TC-071 | `bench/scripts/normalize-review-findings.sh --i07 <lifecycle.jsonl> --output <findings.json>` | None; raw I-07 JSONL and normalizer are real | Do not mock raw finding preservation, identity assignment, confirmation, or truth-set detection | A normalizer trusting reviewer fingerprint/severity/disposition would misclassify duplicates or confirmation. |
| TC-072 | `bench/scripts/compare-lifecycle-evaluations.sh --left <left-evaluation.jsonl> --right <right-evaluation.jsonl> --mode independent_frozen_candidate --output <comparison.json>` and the same exact command with `--mode sequential_delivery` | None; use retained candidate JSONL fixtures | Do not mock candidate lineage, intervening-fix detection, or newly-confirmed attribution | Stage order alone would be reported as causal value or a fix edge would disappear. |
| TC-073 | `verify-lifecycle-evaluation.sh <evaluation.jsonl> --schema bench/evaluation/i08-schema.yaml` followed by the real evaluator replay over retained I-05/I-07 files | Provider/network/DB/live-tree denial process only | Do not mock canonical serialization, replay lineage, or scenario-not-rerun assertion | A replay that reruns a scenario or depends on provider output would evade the byte-identity check. |
| TC-074 | `bench/scripts/aggregate-runs.sh` on the evaluator’s retained output plus `verify-lifecycle-evaluation.sh` | None; real invalid inventory and aggregator | Do not mock invalid classification, exclusion, or output retention | Invalid records would be silently dropped or averaged into a headline. |
| TC-075 | `tc075_content-identity_x12_test.sh` invokes the real installed-tree digest path and then `compare-lifecycle-evaluations.sh` | None; use a temporary copy of the installed canonical tree | Do not mock file discovery, tree ordering, digest computation, or comparator | A missing prompt/skill/agent file from the X-12 digest would permit mixed bundles. |
| TC-076 | `bench/scripts/verify-lifecycle-evaluation.sh` plus static scan and 100 MB fixture under `tc076_*` | None; shell process and committed fixture are real | Do not mock streaming, file-size behavior, forbidden-path scan, or adapter dispatch | A validator loading an unbounded transcript or branching on Python/Go would pass a shallow test. |
| TC-077 | `make fmt && make lint && make test` and `bench/scripts/tests/run-all.sh` | None beyond explicitly declared provider/network denial fixtures | Do not replace the full suite with selected F09 tests or delete prior registrations | A green F09-only run would conceal a regression or removed F01-F08 test. |

## Acceptance Test Cases

### TC-067: I-08 schema, truth separation, and shared contract

**Feature requirements:** REQ-F-001/002/003/006/013, REQ-NF-001/002/003; AC-001,
AC-002; I-05, I-07, I-08; UAT-13/UAT-14.
**Technique:** Equivalence partitioning, contract-surface enumeration, BVA,
decision table. **ISO:** Functional suitability, reliability, security,
maintainability, compatibility, portability.

**Setup/input:** Run the named Go contract test against the committed schema
and valid/invalid fixtures. Include missing/null/wrong-type/empty/unknown-field,
malformed digest, duplicate join, contradictory reference, secret/evaluator
payload, each structural-check outcome, missing judge, terminal-only oracle,
and complete eligible records. Read staged I-05/I-07/I-08 metadata and the live
counterpart status at review/UAT time.

**Expected:** Exactly one versioned I-08 object has schema-owned fields and
diagnostics naming the failing path. Structural, judge, and oracle blocks remain
independent; terminal status, process exit, and self-report never fill gaps.
Every invalid case is ineligible with a machine-readable reason. The shared
pointer is exactly `...e40_i08...#TC-067`; I-05 is `TC-042`, I-07 is `TC-061`.
Staged metadata retains `contract-only`, the F09/F10 activation and closure
values, live status handling, review basis, and `pending-integration`.

**Negative/edge:** Empty but present judge/oracle, duplicate join, unknown
vocabulary, uppercase/non-hex digest, and an otherwise matching branch/HEAD
must all fail with a named path/reason.

**Required I-05/I-07 field enumeration:** The contract fixture matrix asserts
the presence and join identity of the agent-visible roots, evaluator-only roots,
and scratch Shark project root; non-overlapping interval ledger; candidate
base/tree/binary-diff/changed-path/dirty-untracked/test-suite fields; artifact
producer/consumer/access events; keyed dispatch responses; claims, heartbeats,
releases, semantic outcomes, and resulting statuses; dispatch and scenario
lineage; Questions and replay decisions; workflow policy; prompt/input/replay/
output; provider usage/model/effort; cost/error/rework; resource limits and
observed consumption; stop outcome; named scenario outcome; and aggregate/
publication eligibility. For each field the invalid partition is missing, null,
wrong type, empty, malformed digest, contradictory cross-reference, and
duplicate where uniqueness is required.

**Truth decision table:** TC-067 has explicit rows for structural-pass/judge-
pass/oracle-pass (eligible), structural-fail/judge-pass/oracle-pass,
structural-pass/judge-fail/oracle-pass, structural-pass/judge-pass/oracle-fail,
missing structural, missing applicable judge, missing oracle, and
not-applicable judge. A legitimately not-applicable judge is valid only when
the artifact applicability field says `not_applicable`; it may remain aggregate
eligible if structural and oracle truth pass, while a missing judge for an
applicable artifact is ineligible. Every other invalid row retains all three
blocks, applicability, ordered reason, source references, and false eligibility.

### TC-068: Held-back oracle ordering and evaluator isolation

**Feature requirements:** REQ-F-004/REQ-NF-003; AC-003; UAT-09/UAT-13.
**Technique:** State transition + attack-class enumeration. **ISO:** Functional
suitability, reliability, security, compatibility, maintainability.

**Setup/input:** Use pre-terminal, terminal-but-unauthorized, authorized
post-terminal, adapter-failure, hidden-test injection, symlink, traversal,
broken-link, renamed-root, and cleanup fixtures through the real guard and I-04
adapter. Deny provider/network access loudly and record attempted calls.

**Expected:** Pre-terminal and failed-isolation paths make zero adapter/provider
calls and retain named invalidity. Authorized evaluation records one access
event, adapter/test/reference digests, bounded result evidence, and no
evaluator-only bytes in either agent-visible root after cleanup.

**Negative/edge:** Access before terminal, hidden file introduced after
admission, cleanup failure, and a non-empty evaluator root must not be treated
as a successful oracle.

The harness scans both agent-visible roots immediately before every dispatch,
records the scan/access event in the retained I-08 source references, then
scans after cleanup. A late disclosure, symlink, traversal, broken link, or
renamed-root observation is a named invalidity and is never inferred from the
final process status.

### TC-069: Calibration boundary and judge provenance

**Feature requirements:** REQ-F-005; AC-004; UAT-13.
**Technique:** Equivalence partitioning + BVA + decision table. **ISO:**
Functional suitability, reliability, security, maintainability, compatibility.

**Setup/input:** Supply a valid calibrated judge result, disjoint calibration
and held-out sets, missing calibration, overlapping example, changed rubric or
prompt digest, changed judge model/configuration, missing reference, malformed
usage/cost, and non-applicable planning artifact.

**Expected:** Only the complete calibrated applicable case is eligible. The
record retains rubric/prompt/model/configuration/calibration/reference
identity, rationale, score, usage, cost, and applicability. Every other case
retains a named invalidity reason and no default/fabricated score.

**Negative/edge:** Empty calibration set, duplicate example ID, held-out item
present in calibration, negative/unknown usage, and judge identity omitted.

The fixture matrix also enumerates score/rationale empty and out-of-range
partitions, zero-cost and zero-usage boundaries, unknown judge fields,
applicability/result contradictions, malformed reference-set membership,
duplicate calibration IDs, and missing/invalid reference digests.

### TC-070: Complete evaluation, candidate, and workflow-policy identity

**Feature requirements:** REQ-F-006/007/008; AC-005; UAT-14/UAT-19.
**Technique:** Contract-surface enumeration + one-factor mutation + decision
table. **ISO:** Functional suitability, reliability, security, maintainability,
compatibility, portability.

**Setup/input:** Start with an exact matching pair. Mutate one at a time:
scenario/version, fixture, adapter, toolchain, Shark binary, installed content,
rendered prompt, provider/model/effort, judge, reference, resource policy;
candidate base/tree/binary diff/changed paths/ordered dirty-untracked
manifest/test-suite; enabled gates/order/reviewer provider/model/effort/review
prompt/deep-review bundle/fix policy.

**Expected:** Each mutation rejects the pair with the exact differing field and
digest. Exact match accepts. Branch-only, HEAD-only, base-only, reordered
manifest, and reordered prompt/stage list shortcuts reject.

**Negative/edge:** Missing field versus unequal field are distinct reasons;
one changed angle prompt, consolidator, `get_diff.sh`, or one agent file is
also sufficient to reject.

The matrix explicitly includes every rendered prompt digest and every source
artifact digest (I-05, I-07, scenario, oracle test/reference, judge,
calibration, evaluator-reference, resource-policy, candidate, and workflow
bundle), not merely one representative prompt or source digest.

### TC-071: Review-finding normalization and confirmation

**Feature requirements:** REQ-F-009/011; AC-006; UAT-17.
**Technique:** Equivalence partitioning + attack-class enumeration. **ISO:**
Functional suitability, reliability, security, maintainability, compatibility.

**Setup/input:** Feed multiple findings, zero findings, duplicate fingerprints,
recurrence after candidate change, confirmed seeded defect, clean control,
unconfirmed observation, missing truth set, and malformed raw fields.

**Expected:** Raw fields remain byte-preserved evidence. F09 creates its own
identity namespace with confirmation source, first-seen gate, duplicate or
recurrence link, resolution candidate, and final disposition. Precision/recall
appear only with a retained truth set; otherwise truth-set-unavailable and
non-truth-set measures appear. Quality, time, cost, rework, and artifact-use
remain separate.

**Negative/edge:** Reviewer severity/fingerprint/disposition alone cannot
confirm a finding; duplicate and recurrence are not counted as new confirmed
findings.

Raw-field attack partitions are enumerated independently for gate, round,
severity, defect class, fingerprint, criterion, disposition, confirmation,
resolution candidate, and first-seen gate: missing, null, wrong type, empty,
unknown vocabulary, duplicate, contradictory candidate, and invalid digest.
Conflicting seeded truth-set evidence is also a separate ineligible partition.

### TC-072: Independent frozen-candidate and sequential delivery comparison

**Feature requirements:** REQ-F-010/011/012; AC-007; UAT-17/UAT-19.
**Technique:** State transition + decision table. **ISO:** Functional
suitability, reliability, maintainability, compatibility.

**Setup/input:** Compare QA and deep review over one identical candidate and
test snapshot with no fix edge in `independent_frozen_candidate`. Then compare
sequential candidates with every intervening candidate and finding lineage
retained.

**Expected:** Independent mode reports a shared frozen candidate and no fix.
Sequential mode attributes only newly confirmed findings to the later gate and
retains all candidate identities. Stage order alone produces no causal claim;
quality/time/cost are separate.

**Negative/edge:** A hidden intervening fix, changed test snapshot, missing
candidate lineage, or wrong mode vocabulary rejects the comparison.

Both accepted and rejected outputs must retain explicit candidate identity and
workflow-policy digests, and sequential output must retain every intervening
candidate artifact and finding link. A mode label without those digests cannot
qualify a comparison.

### TC-073: Deterministic offline evaluation replay

**Feature requirements:** REQ-NF-002/003; AC-008; UAT-09/UAT-14.
**Technique:** Retest/reproducibility testing + attack-class enumeration. **ISO:**
Functional suitability, reliability, security, compatibility, portability.

**Setup/input:** Evaluate the same retained I-05/I-07 inputs twice with provider,
network, live DB, `.sharkconfig.json`, and working-tree writes denied. Vary
filesystem order, locale, JSON key order, and shell environment while keeping
declared inputs identical.

**Expected:** Verdict and ordered invalidity inventory are byte-identical;
scenario and provider are not rerun; output contains references/digests and
bounded evidence only. Any attempted forbidden egress/write fails loudly.

Negative replay partitions include missing/corrupt source artifact, duplicate
JSONL record, truncated JSONL, clock/randomness dependence, duplicate
evaluation ID, output-root collision, and changed source digest. These retain a
stable evaluation ID, ordered invalidity reasons, and source references rather
than being compared as valid replays.

### TC-074: Invalid retention and aggregation exclusion

**Feature requirements:** REQ-F-013; AC-009; UAT-14.
**Technique:** Equivalence partitioning + decision table. **ISO:** Functional
suitability, reliability, security, maintainability, compatibility.

**Setup/input:** Supply incomplete, mixed, failed-oracle, failed-calibration,
isolation-violating, candidate-mismatched, and policy-mismatched records plus
one complete valid control to the evaluator and existing aggregate path.

**Expected:** Every invalid record remains under the declared output root with
all reasons; none contributes to an aggregate or disappears. The valid control
alone contributes, and no composite efficiency score is emitted.

For every invalid case, assert stable `evaluation_id`, original I-05/I-07
source references, ordered complete invalidity inventory, and the concrete
path `bench/evaluations/<evaluation_id>/evaluation.jsonl`; the aggregate input
must reference the retained record while excluding its measures.

### TC-075: X-12 installed canonical content identity

**Feature requirements:** REQ-F-006/008; AC-010; X-12; UAT-14.
**Technique:** Contract-surface enumeration + one-factor mutation. **ISO:**
Functional suitability, reliability, security, maintainability, portability.

**Setup/input:** Hash the installed canonical Shark-data tree, then mutate one
workflow, prompt, skill, and agent file in isolated copies.

**Expected:** Each mutation changes the canonical content digest and rejects a
paired comparison with both digests and the exact differing source class.
Mixed bundles cannot produce a quality delta.

Inventory partitions also cover empty tree, missing file, extra file, symlink,
duplicate logical path, reordered manifest, and non-canonical path spelling.
The test proves the inventory is complete, not just that four representative
files affect a digest.

### TC-076: Static safety, language neutrality, and 100 MB verifier bound

**Feature requirements:** REQ-NF-003/004/005; AC-011. **Technique:** Structural
testing + compatibility testing + BVA. **ISO:** Functional suitability,
performance, security, maintainability, portability.

**Setup/input:** Scan all F09 scripts for writes to product/database paths,
provider calls, evaluator-only copies, language-specific branches, and
unbounded transcript loading. Run the committed 100 MB synthetic fixture.

**Expected:** No forbidden path or branch exists; Python/Go/package-manager,
test, lint, build, and oracle behavior reaches the I-04 adapter. Streaming
verification completes within 30 seconds on the repository CI runner and does
not retain unbounded payloads in I-08.

Runtime denial fixtures replace static assumptions: attempted writes to the
live DB, `.sharkconfig.json`, working tree, provider, or network fail loudly
and are retained as bounded attempted-operation evidence.

### TC-077: Full regression and registered F09 contract suite

**Feature requirements:** AC-012 and all listed test registrations. **Technique:**
Regression testing + retest/state transition. **ISO:** Functional suitability,
reliability, maintainability, compatibility, portability.

**Setup/input:** Run `make fmt && make lint && make test`, then
`bench/scripts/tests/run-all.sh`; verify TC-067 through TC-075 are registered
and all existing F01-F08 tests remain present and effective.

**Expected:** All commands pass, F09 cases execute in deterministic order, no
prior test is removed or weakened, and provider/network/live-DB protections
remain active.

The test first enumerates the pre-F09 F01-F08 registrations from the existing
`run-all.sh`, runs each by name, and asserts each produces its prior pass
marker; a full-suite exit code alone is insufficient evidence of preservation.

## Usability and operator diagnostics

Although F09 has no product UI, its file-backed shell commands are operator
surfaces. For every non-applicable usability cell in the matrix, the named TC
must assert non-zero authoring/usage errors, bounded stderr with evaluation ID,
source path, reason code, remediation-oriented field/path, and output path;
successful records must print or retain the exact artifact path. These are
covered by TC-067 through TC-077, including TC-071/072 normalization and
comparison diagnostics and TC-077 suite-preservation diagnostics.

## Test Infrastructure

### Existing patterns to follow

- `tests/contracts/e40_i02_artifact_contract_test.go` for schema-backed Go
  contract tests and committed valid/invalid fixtures.
- `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` and the F06/F08
  shared contract tests for file-backed staged-edge contracts.
- `bench/scripts/aggregate-runs.sh` for strict invalid-input retention and
  no fabricated aggregate output.
- `bench/scripts/verify-stage-evidence.sh` and
  `bench/scripts/verify-evidence-roots.sh` for existing isolation seams.
- `bench/runs/i07-schema.yaml` and `bench/evidence/i05-schema.yaml` as
  upstream schema owners; do not duplicate their vocabularies in tests.
- `skills/shark-rider/skills/deep-review/references/` plus
  `skills/shark-rider/skills/deep-review/scripts/get_diff.sh` as the bundle
  files whose combined digest must be exercised.

### New fixtures/helpers required

- `bench/evaluation/i08-schema.yaml` and
  `tests/contracts/testdata/e40_i08/{valid,invalid}/` with field-level attacks.
- `bench/scripts/tests/tc067_*` through `tc075_*` and their registered
  `run-all.sh` entries; TC-076/077 cover the non-functional and suite gates.
- File-backed I-04 adapter, provider/network denial, evaluator-root, clock,
  and canonical-tree mutation fixtures. No real database is needed: F09 is
  file-backed/offline; if a repository test is added, it must use the project’s
  real-DB repository-test rule. No mocks/stubs are allowed for validators,
  comparators, normalization, aggregation, identity calculation, or committed
  schema/fixture reads; only the explicitly declared I-04 adapter and
  provider/network denial process seams may be deterministic fixtures.

## Recommendations

- [x] Ready for development (no unresolved drift; post-remediation Codex
  red-team returned PASS)
- [ ] Needs BA refinement
- [ ] Needs tech refinement

## Codex Test-Plan Red-Team

**Verdict:** PASS
**Issues raised:** 21 across the initial and follow-up reviews
**Issues addressed before dev:** 21
**Issues deferred:** 0

The red-team must check open-endedness, technique fit, exhaustive selected
partitions, ISO coverage, runtime evidence, negative cases, exact production
argument shapes, forbidden mocks, staged I-## metadata, and X-12 coverage. Any
The initial long-budget review identified the following issues; all are
addressed above and must be checked again by the fresh review:

1. AC-001 needed explicit I-05/I-07 field-level enumeration and AC-002 needed
   a complete truth decision table; added to TC-067.
2. AC-003 needed an immediate pre-dispatch scan in both agent-visible roots;
   added to TC-068.
3. AC-004 needed calibration IDs, score/rationale, zero-cost, unknown fields,
   applicability, and reference-membership partitions; added to TC-069.
4. AC-005 needed every prompt and source-artifact digest; added to TC-070.
5. AC-006 needed per-field raw-finding attacks and conflicting truth evidence;
   added to TC-071.
6. AC-007 needed retained candidate/policy digests and intervening artifacts;
   added to TC-072.
7. AC-008 needed corrupt/duplicate/truncated/clock/randomness/output-collision
   replay partitions; added to TC-073.
8. AC-009 needed stable IDs, ordered reasons, source references, and exact
   evaluation-root assertions; added to TC-074.
9. AC-010 needed complete content inventory attacks; added to TC-075.
10. AC-011 needed runtime denial fixtures, not only static scans; added to
    TC-076.
11. AC-012 needed executable prior-registration enumeration; added to TC-077.
12. Operator usability diagnostics were made explicit in the new diagnostics
    section.
13. Observability is mandatory through retained I-08 paths and bounded
    diagnostics; optional `if present` evidence is not accepted.
14. Infrastructure mock guidance was narrowed to the two explicitly allowed
    seams.
15. Full staged owner/closure sets are asserted below, not only the F09 slice.
16. Status and recommendation remained pending until the fresh review returned
    PASS; they are now approved.
17. The follow-up review required the scratch Shark root and exact keyed
    dispatch/claim/heartbeat/release/outcome/status/Question/replay/resource/
    scenario fields; added to TC-067.
18. The follow-up review required explicit `not_applicable` judge eligibility;
    added to the TC-067 truth table.
19. The follow-up review required complete evaluator/oracle and comparator argv;
    added to TC-068 and TC-072 caller contracts.
20. The follow-up review required operator diagnostics for TC-071/072/077;
    expanded the diagnostics assertion to TC-067 through TC-077.

### Final Codex review output (verbatim)

```text
PASS

1. All four prior blockers are resolved: complete I-05/I-07 fields, judge eligibility distinction, exact TC-068/TC-072 argv, and operator diagnostics coverage.
2. Every AC has technique, ISO mapping, negative cases, mandatory observability, and caller-path contracts.
3. I-05/I-07/I-08 staged metadata and X-12 match the source maps; no blocking drift found.
```

### Initial Codex output (verbatim)

The initial review returned `FAIL` with the 17 numbered findings summarized
above. The full command output is retained in the execution transcript for
this planning run; the remediation mapping above is the incorporated response.
