# Test Plan: E40-F08 - Canonical multi-entity lifecycle runner

**Created:** 2026-08-16
**Feature PRD:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/feature.md`
**Task Spec:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/spec.md`
**Status:** APPROVED FOR IMPLEMENTATION PLANNING (not implementation completion)

## Spec Drift Analysis

### Drift Findings

- `feature.md` is the feature-level intent document and has an acceptance
  boundary, while `spec.md` is the combined requirements/architecture/task
  specification. The spec adds concrete I-07 fields, command ordering, test
  pointers, and non-functional constraints; these refine rather than change
  the feature outcome.
- The spec narrows the implementation to `bench/` and explicitly excludes
  Shark product code, a second workflow engine, claim store, Question store,
  prompt reconstruction, and evaluation. This is consistent with the feature
  boundary and is tested below.
- No scope, semantic, schema, or conversion drift remains unresolved.

### Traceability Matrix

| Feature requirement | Task acceptance criterion(s) | Covered? | Notes |
|---|---|---:|---|
| Drive root and all eligible descendants through keyed dispatch | AC-002, AC-003, AC-004, AC-012 | Yes | Real `run-lifecycle.sh` with stubbed public commands; retained UAT path |
| Parent-owned claim/heartbeat/transition/release and unchanged prompt | AC-002, AC-003 | Yes | Caller-path tests assert exact command arguments and authority boundary |
| Named stops, limits, partial evidence, and publication ineligibility | AC-005, AC-007, AC-009 | Yes | Each stop class and first-ceiling behavior is enumerated |
| I-05 evidence and isolation | AC-008, AC-009 | Yes | Existing guards are invoked; missing evidence fails closed |
| Review-gate and raw finding evidence | AC-006, AC-009 | Yes | Findings, zero findings, collection failure, and not-reached are distinct |
| Deterministic, offline, file-backed I-07 record | AC-001, AC-010, AC-011 | Yes | Schema validator, byte-identical reruns, no internal/DB changes |

## Acceptance Criteria Review

### Ambiguity Findings

None. The specification names exact command surfaces, fields, vocabularies,
fixtures, test locations, stop outcomes, and publication rules. The test plan
uses the production command argument shapes rather than helper-only signatures.

### Missing Coverage

None. UAT-09 is covered by TC-008; UAT-11 and UAT-12 by TC-012/TC-013;
UAT-16 by TC-003/TC-009/TC-014; UAT-17 by TC-006/TC-012; UAT-18 by
TC-009/TC-010; and UAT-19 by TC-009/TC-012. F08 does not own
UAT-13/UAT-14 evaluation decisions, so those are intentionally out of scope.

## ISTQB Technique Application (per AC)

The plan uses the workflow's attack/contract terms as coverage descriptors.
Their ISTQB mapping is: `contract-surface enumeration` → equivalence
partitioning over interface fields and decision-table combinations;
`attack-class enumeration` → equivalence partitions of invalid/adversarial
inputs plus boundary-value analysis where a limit is involved;
`ordering/uniqueness partitioning` → equivalence partitioning plus state
transition; `reproducibility/replay testing` → state-transition/retest
testing; `environment partitioning` → compatibility testing; and `resume
testing` → state-transition testing. Every row below therefore includes a
recognized ISTQB technique, with the workflow descriptor as an explicit
sub-technique.

| AC | Technique(s) applied | Test cases generated | Rationale |
|---|---|---|---|
| AC-001 | Equivalence partitioning + decision table (contract-surface/attack-class descriptors) | TC-001 | Every I-07 field has missing/null/wrong-type/empty/oversized/extra and cross-field-invalid partitions, plus vocabulary and ordinal attacks |
| AC-002 | Equivalence partitioning + state transition (contract-surface descriptor) | TC-002 | Enumerates every public command boundary and exact prompt/authority transition |
| AC-003 | State transition + decision table + boundary-value analysis | TC-003 | Every command failure, TTL/cadence boundary, cancellation point, and cleanup outcome has an exact order/session expectation |
| AC-004 | Boundary-value analysis + equivalence partitioning + state transition (ordering/uniqueness descriptor) | TC-004 | Zero/one/two/three/many candidates, nested forks, duplicates, cycles, changed sets, and canonical completion are enumerated |
| AC-005 | Boundary-value analysis + decision table | TC-005 | Equality, first exceed, simultaneous exceed, overshoot, root counting, and unavailable observations are enumerated |
| AC-006 | Equivalence partitioning + decision table (attack-class descriptor) | TC-006 | Four gate states plus duplicate/malformed/missing identity and empty-collector partitions remain distinct |
| AC-007 | State transition + equivalence partitioning + decision table (contract-surface descriptor) | TC-007 | Complete I-04/I-06 payload, replay sequence, request matching, and authorization partitions are explicit |
| AC-008 | Equivalence partitioning + boundary-value analysis (attack-class/path descriptor) | TC-008 | Files, symlinks, nested paths, renamed roots, traversal, broken links, late disclosure, and both visible roots are tested |
| AC-009 | Equivalence partitioning + decision table (attack-class/contract-surface descriptors) | TC-009 | Every I-05 lineage, identity, timing, access, usage, policy, and artifact join field fails closed |
| AC-010 | State transition/retest + compatibility testing (replay/environment descriptors) | TC-010 | Same inputs are run twice with loud egress denial and filesystem/locale/shell/key-order variation |
| AC-011 | Structural/configuration testing + equivalence partitioning (contract-surface descriptor) | TC-011 | Verifies registration, full quality gate, forbidden changes, and adapter-only family dispatch |
| AC-012 | State transition + decision table (contract/resume descriptors) | TC-012, TC-013 | Real keyed Rider loop, bounded resume, and durable Question handoff are exercised |

## ISO 25010 Coverage Matrix

| AC | Functional suitability | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | TC-001 validates every schema field/vocabulary | N/A: no performance requirement for offline validation; command completion is recorded | TC-001 runs committed YAML/JSON on the repository shell | N/A: machine-only contract; field diagnostics are the operator surface | TC-001 rejects malformed/partial records | TC-001 rejects secret/prompt fields and invalid eligibility | TC-001 derives fields from one schema, preventing drift | TC-001 uses POSIX shell, `yq`, and `jq` fixtures |
| AC-002 | TC-002 compares complete request/result fields | N/A: provider latency is out of scope; canary records process duration | TC-002 uses the public `NextResponse` JSON contract | TC-002 asserts bounded evidence and actionable redaction failures | TC-002 requires persist-before-dispatch and exact bytes | TC-002 proves no worker mutation authority or secret leakage | TC-002 pins source line and adapter contract | TC-002 uses file paths and JSON only |
| AC-003 | TC-003 asserts lifecycle state/order and release | TC-003 measures file-clock cadence and strict pre-TTL boundary | TC-003 uses public claim/heartbeat/advance/release argv | N/A: no end-user interaction; failure reason is machine-visible | TC-003 covers every injected failure and cleanup path | TC-003 binds all mutations to returned session | TC-003's decision table prevents cleanup duplication | TC-003 runs with the real binary seam and shell fixture |
| AC-004 | TC-004 asserts all eligible descendants exactly once | N/A: serial scheduling has no latency SLA; canary duration is recorded | TC-004 consumes actual `parallel_candidates` envelope | N/A: internal scheduler, not a user-facing workflow | TC-004 rejects duplicate/cycle/changed graph inputs | TC-004 treats duplicate/path/selection metadata as untrusted | TC-004 centralizes canonical ordering and generated guard | TC-004 uses the same adapter across fixture families |
| AC-005 | TC-005 asserts first-exceeded outcome and retained evidence | TC-005 records exact cost/wall/task observations and ceiling decision | TC-005 reads I-04 policy and file-backed usage/clock | N/A: named machine outcome, no interaction design | TC-005 asserts stop-before-next and release-on-stop | TC-005 prevents uncontrolled provider spend and post-stop work | TC-005 has a table for equality, exceed, unavailable, simultaneous | TC-005 uses portable NDJSON/YAML fixtures |
| AC-006 | TC-006 asserts one reached-gate state and raw finding identity | N/A: local collector has no latency target | TC-006 uses typed review-note fields and file collector | N/A: machine-only gate record; diagnostics name the invalid field | TC-006 distinguishes absent, empty, failed, and unreached | TC-006 preserves raw bytes and prevents false zero-findings | TC-006 keeps adjudication in F09 via explicit raw fields | TC-006 uses committed JSON fixtures |
| AC-007 | TC-007/013 assert terminal prelude and Question mappings | N/A: blocked replay must make zero provider calls | TC-007 invokes shared I-04/I-06 contract tests and public Question CLI | TC-013 leaves an explicit durable unresolved gate | TC-007 consumes each authorized entry exactly once | TC-007 checks responder/owner authorization and no transcript decision | TC-007 reuses producer validators rather than copying rules | TC-007 uses file-backed replay independent of provider |
| AC-008 | TC-008 asserts guard-before-each-dispatch and zero starts | TC-008 records time-to-fail-before-spend, not provider latency | TC-008 uses the existing I-05 guard and three-root paths | N/A: pre-dispatch machine guard | TC-008 covers late disclosure, links, traversal, and broken paths | TC-008 prevents evaluator disclosure in both agent roots | TC-008 calls one existing guard rather than duplicating policy | TC-008 uses filesystem primitives available in the test shell |
| AC-009 | TC-009 asserts controller-written I-05/I-07 joins before mutation | N/A: validator has no product latency SLA | TC-009 consumes real retained I-05 files and path/digest joins | N/A: internal evidence join; invalidity reason is the operator output | TC-009 distinguishes absent evidence from empty evidence | TC-009 protects evaluator access ordering and candidate identity | TC-009 tests every join field through copied mutations | TC-009 uses file-backed artifacts, no database |
| AC-010 | TC-010 compares canonical verdict bytes and zero egress | TC-010 records bounded rerun duration but claims no benchmark threshold | TC-010 varies shell, locale, filesystem, and key ordering | N/A: offline machine reproducibility contract | TC-010 pins seed/timezone/run ID and rejects provider starts | TC-010 uses an executable denial guard and scans egress | TC-010 makes canonical projection/schema ownership explicit | TC-010 runs in contract and dry-run modes |
| AC-011 | TC-011 asserts registration, matrix reachability, and forbidden paths | TC-011 runs the full repository quality gate within CI's existing budget; no new SLA | TC-011 exercises Python/Go/package-manager/test/lint adapter declarations | N/A: build/operator contract, not product UX | TC-011 requires `run-all.sh` plus full quality commands | TC-011 scans internal/cmd/migration/generated/evaluator paths | TC-011's structural scan prevents future language branching | TC-011 verifies shell fixture portability and no DB dependency |
| AC-012 | TC-012/013 assert real keyed loop, resume, and durable Question state | TC-012 records bounded retained-run duration and stop timing | TC-012/013 use public X-11/X-13 commands and real scratch Shark | TC-013 exposes unresolved/authorized outcomes durably | TC-012 asserts exact wire events and resume cleanup | TC-013 enforces authorization and no advance after unacknowledged worker | TC-012's resume decision table is reusable across adapters | TC-012/013 retain JSONL evidence for offline inspection |

There are no unqualified checkmarks. Each applicable cell names a parser or
assertion; each `N/A` states why that characteristic is outside this
file-backed, offline test-plan boundary.

## Observability Design (per behavior)

| Behavior | Emitter and sink | Parser/assertion | QA-only alert condition |
|---|---|---|---|
| Keyed dispatch and prompt handoff | Controller writes `bench/runs/<run_id>/events.ndjson` (`dispatch`) and `metrics.ndjson` (`lifecycle.dispatch.count`); trace goes to `trace.ndjson` | `jq -s` parser asserts one dispatch per ordinal, exact `entity_key`, `entity_type`, `status`, `action`, `agent_type`, `provider`, `model`, `effort`, `prompt_sha256`, `prompt_bytes`, `resolved_via`, `unresolved_placeholders`, and no `prompt` value | Any digest/byte mismatch, prompt leak, or duplicate ordinal is a hard failure |
| Lease and worker lifecycle | Controller writes `claim`, `heartbeat`, `advance`, and `release` events with `session_id` and command argv | `awk`/`jq` parser asserts command order, returned key/session reuse, heartbeat timestamps strictly before TTL, one release per owned lease, and parent span ID | Missing release, wrong session, post-TTL heartbeat, or worker mutation attempt |
| Stage/time/artifact evidence | Controller writes I-05 references to `lifecycle.jsonl` and stage/interval/access events to `events.ndjson` | Parser asserts stage categories, non-overlapping intervals, usage/model identity, candidate digest, artifact `path`/`digest`/`size_bytes`/producer and every access edge; absent `consumers` is distinct from `consumers: []` | Any overlap, missing identity, evaluator access before terminal stage, or missing edge |
| Safety ceilings and stops | Controller writes `stop` event and `lifecycle.stop.count{outcome}` with bounded labels | Parser asserts `first_exceeded`, policy/observed values, `partial_evidence`, `publication_eligible`, non-empty `reason`, and release events for all owned leases | Any stop without reason, later dispatch, or unreleased lease |
| Review-gate capture | Controller writes one `review_gate` event per reached gate and `lifecycle.review_gate.count{state}` | Parser asserts exactly one state, gate/round, raw finding bytes digest plus field-preserving payload, candidate/policy references, and `not_reached` for skipped gates | Duplicate reached-gate record, missing collector result, or raw-byte mutation |
| Question replay | Controller writes `question` events and `lifecycle.question.count{outcome}` | Parser asserts key, claim session, responder, owner, authorized entry digest, request match, evidence pointer, and durable terminal result | Unauthorized/missing response, unused-entry mismatch, or transcript-only resolution |
| Contract validation | `verify-lifecycle-run.sh` writes `validation` events, `metrics.ndjson`, and named stderr diagnostics | `jq -s` parser asserts bounded `field` labels, run ID, source path, reason, and `publication_eligible=false` for invalidity | Any validator acceptance of an unknown field/vocabulary or eligibility conflict |

The parsers read the event names, required fields, bounded-label policy, and
canonical timestamp rules from `bench/runs/i07-schema.yaml`; they do not infer
success from a process exit code. `TC-002`, `TC-003`, `TC-005`, `TC-006`,
`TC-007`, `TC-009`, `TC-012`, and `TC-014` invoke the parsers against the
controller's actual `bench/runs/<run_id>/{lifecycle.jsonl,events.ndjson,
metrics.ndjson,trace.ndjson}` outputs. Heartbeat acceptance is
`claim_time <= heartbeat_time < claim_time + effective_ttl`; equality and later
timestamps are rejected, regardless of cadence tolerance.

Pure serialization helpers may have no independent metric, but their output is
covered by the validator and deterministic verdict assertions. The developer
must add the listed bounded evidence before provider-backed execution; tests
inspect emitted records/logs rather than infer behavior from exit code.

## Cross-feature contract tests (I-##)

All staged rows preserve the map-assigned `gate_mode=contract-only` until live
consumer activation. At review/UAT time, counterpart status must be read live
from Shark; this plan does not copy a stale status. Activation owner, closure
key, and review basis are retained exactly as assigned by the interaction map.

| I-## | Producer | Consumer | Shape source | Gate fields | Contract test pointer | TC |
|---|---|---|---|---|---|---|
| I-04 | E40-F05 | E40-F08 | `architecture.md#lifecycle-scenario-package-contract` | `gate_mode: contract-only`; activation owners F06/F07/F08 by slice; closure F06/F07/F08 UAT; `counterpart_status`: read live from Shark; `review_basis`: F05 spec + map row; `demonstrability_disposition: pending-integration` | `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` | TC-007 asserts fields |
| I-05 | E40-F06 | E40-F08 | `architecture.md#stage-evidence-and-isolation-contract` | `gate_mode: contract-only`; activation owners F08/F09/F10 by slice; closure F08/F09/F10 UAT; `counterpart_status`: read live from Shark; `review_basis`: F06 spec + map row; `demonstrability_disposition: pending-integration` | `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042` | TC-009 asserts fields |
| I-06 | E40-F07 | E40-F08 | `architecture.md#product-design-replay-contract` | `gate_mode: contract-only`; activation owner F08; closure F08 UAT; `counterpart_status`: read live from Shark; `review_basis`: F07 spec + map row; `demonstrability_disposition: pending-integration` | `tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052` | TC-007 asserts fields |
| I-07 | E40-F08 | E40-F09/F10 | `architecture.md#lifecycle-run-record-contract` | `gate_mode: contract-only`; activation owners F09/F10; closure F09/F10 UAT; `counterpart_status`: read live from Shark; `review_basis`: F08 spec + map row; `demonstrability_disposition: pending-integration` | `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061` | TC-001/009 assert fields |

The same named contract tests are referenced by producer and consumer plans;
this plan does not create twin tests. X-11 is covered by TC-002/TC-003 and
X-13 by TC-007/TC-013, matching the F08 spec pointers and UAT-11/UAT-12.
Plan TC-001 is the QA design case for shared implementation contract TC-061;
the mapping is explicit and is not a second contract test.

**Executable staged-edge read (I-04 through I-07):** Before the retained UAT,
`bench/scripts/tests/tc060_lifecycle_runner_contract_test.sh` reads the live
counterpart rows with `shark get E40-F05 --json`, `shark get E40-F06 --json`,
`shark get E40-F07 --json`, and `shark get E40-F08 --json` (read-only; no
status/claim/release mutation). It also reads the map and spec files named in
the table, computes `sha256sum` for each shared contract-test source, and
asserts these exact fields in the staged metadata:

| Edge | Live fields and exact expected values |
|---|---|
| I-04 | `gate_mode=contract-only`; activation owners `E40-F06`, `E40-F07`, `E40-F08` by slice; closure keys `E40-F06`, `E40-F07`, `E40-F08`; `counterpart_status` equals the live `shark get E40-F05 --json` value; `review_basis` names E40-F05 `spec.md` and the I-04 map row; `demonstrability_disposition=pending-integration`; pointer `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` and its computed digest |
| I-05 | `gate_mode=contract-only`; activation owners `E40-F08`, `E40-F09`, `E40-F10`; closure keys `E40-F08`, `E40-F09`, `E40-F10`; live E40-F06 counterpart status; `review_basis` names E40-F06 `spec.md` and the I-05 map row; `demonstrability_disposition=pending-integration`; pointer `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042` and its computed digest |
| I-06 | `gate_mode=contract-only`; activation owner `E40-F08`; closure key `E40-F08`; live E40-F07 counterpart status; `review_basis` names E40-F07 `spec.md` and the I-06 map row; `demonstrability_disposition=pending-integration`; pointer `tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052` and its computed digest |
| I-07 | `gate_mode=contract-only`; activation owners `E40-F09`, `E40-F10`; closure keys `E40-F09`, `E40-F10`; live E40-F08 counterpart status; `review_basis` names this F08 `spec.md` and the I-07 map row; `demonstrability_disposition=pending-integration`; pointer `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061` and its computed digest |

The test fails if a copied status, missing pointer/digest, incomplete owner or
closure slice, altered review basis, or any disposition other than the exact
map value is observed. It records the live read and all four digests in
`bench/runs/tc060/staged-contracts.json` for the I-07 join.

## Cross-epic integration tests (X-##)

| X-## | Boundary | Verification | Test coverage pointer | TC |
|---|---|---|---|---|
| X-11 | E38 Rider keyed loop → F08 host controller | Exact dispatch response, claim session, unchanged prompt, heartbeat, semantic outcome, transition, release, and bounded result; workers have no Shark mutation authority | F08 `tc060`/`tc061`, UAT-11/UAT-12 | TC-002, TC-003, TC-012 |
| X-13 | E39 Question lifecycle → F08 replay gate | Authorized response is durable with key/owner/evidence/terminal result; absent or unauthorized response is `unresolved_gate`, never inferred from prose | F08 AC-007 in `tc061`, UAT-12 | TC-007, TC-013 |

## Caller-Path Contracts (per test case)

| TC | Production entrypoint and argument shape | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `bench/scripts/verify-lifecycle-run.sh bench/runs/<run_id>/lifecycle.jsonl --schema bench/runs/i07-schema.yaml` | none; committed files are the seam | Do not mock schema loading, field enumeration, canonical digest, or validator | A validator that accepts an unknown outcome, missing schema-derived field, or eligibility conflict fails |
| TC-002 | `bench/scripts/run-lifecycle.sh --scenario bench/scenarios/packages/py-bug-due-date-boundary/package.yaml --run-id tc002 --root E40-F08-ROOT --scratch-root /tmp/e40-f08-tc002-scratch`; exact child argv is `shark next E40-F08-ROOT --json --prompt-out /tmp/e40-f08-tc002-scratch/prompts/0001`, then `shark claim E40-F08-ROOT --by bench-tc002 --json` | PATH-stub only the external `shark` executable; real controller and real `lifecycle-worker-adapter.sh`; provider is a recording fixture | Do not mock controller ordering, response fields, prompt digest, redaction, or worker authority | A prompt reconstruction, missing `NextResponse` field, prompt leak, or worker status mutation fails |
| TC-003 | Exact child argv includes `shark heartbeat TASK-002 --session SID-002 --progress 0.5 --note tc003`, `shark status advance TASK-002 --outcome pass --session SID-002 --from-status development --agent bench@fixture`, and `shark release TASK-002 --session SID-002 --outcome pass` | PATH-stub only external `shark`; file-backed clock; real controller cleanup | Do not mock cleanup, returned IDs, clock reads, command order, or release failure | Any wrong key/session/status, post-TTL heartbeat, duplicate/missing release, or exit-code-as-complete fails |
| TC-004 | Same production runner with `shark next E40-F08-ROOT --json` returning the complete `parallel_candidates` envelope | PATH-stub only external `shark`; real scheduler and canary adapter | Do not preselect candidates, sort the response in the fixture, or mock scheduler | A dropped, duplicated, cyclic, malformed, or response-order-dependent descendant fails |
| TC-005 | `bench/scripts/run-lifecycle.sh --scenario bench/scripts/testdata/lifecycle/scenario-fork.json --run-id tc005 --root E40-F08-ROOT --scratch-root /tmp/e40-f08-tc005 --limits bench/scripts/testdata/lifecycle/limits/first-exceed.yaml --clock-file bench/scripts/testdata/lifecycle/clock/tc005.ndjson --usage-file bench/scripts/testdata/lifecycle/usage/tc005.ndjson` | Mock only provider process; use file-backed clock/usage and real I-04 policy read | Do not mock stop-record creation, release-on-stop, observation reads, or eligibility | A post-exceed dispatch, wrong first ceiling, zeroed unavailable observation, or unreleased lease fails |
| TC-006 | `run-lifecycle.sh --review-fixture bench/scripts/testdata/lifecycle/review/{findings,zero-findings,collector-failure,duplicate-gate}.json` followed by the real validator | Mock only external reviewer/provider process | Do not mock I-07 gate serialization, raw bytes, candidate/policy join, or collector state | Missing/duplicate/invalid gate data, or changed raw finding bytes, fails |
| TC-007 | Run real `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` and `tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052`; then execute `bench/scripts/run-lifecycle.sh --scenario bench/scenarios/packages/py-feature-recurring-tasks/package.yaml --replay bench/scripts/testdata/lifecycle/prelude/valid-d01-d05/result.json --run-id tc007 --root E40-F08-FEATURE --scratch-root /tmp/e40-f08-tc007` through the public Question commands | Mock only provider process after the file-backed I-04/I-06 chain | Do not mock prelude gating, replay matching, Question authorization, terminal mapping, or output persistence | A missing/unauthorized/unused/mismatched replay entry reaches provider work or invents a decision |
| TC-008 | Before each dispatch, real controller executes `bench/scripts/verify-evidence-roots.sh "$fixture_root" "$scratch_root" "$evaluator_root"` | no mocks; plant only named filesystem attack fixtures | Do not replace the guard with a test-only path check or mock preflight success | Any disclosure visible through either agent root, symlink, traversal, rename, or late admission fails before provider start |
| TC-009 | Real `bench/scripts/verify-stage-evidence.sh`, `bench/scripts/replay-stage-evidence.sh`, and `bench/scripts/verify-lifecycle-run.sh` consume a retained controller run, then copied mutations are applied | no mocks in validators/joins; mutate copied files only after the retained run | Do not mock controller-written evidence, identity, artifact access, or interval joins | Missing or disagreeing I-05/I-07 evidence is never synthesized as valid |
| TC-010 | `bench/scripts/run-lifecycle.sh --mode contract --scenario bench/scripts/testdata/lifecycle/scenario-complete.json --run-id tc010 --root E40-F08-ROOT --scratch-root /tmp/e40-f08-tc010` and the same argv with `--mode dry-run`, each run twice; denial executable is `bench/scripts/testdata/lifecycle/bin/provider-deny` | real denial executable records argv and exits 97; no success stub | Do not mock canonical projection, filesystem ordering, locale, shell, or map ordering | Any adapter/provider start or non-identical canonical verdict fails |
| TC-011 | `bench/scripts/tests/run-all.sh`; `make fmt`; `make lint`; `make test`; structural `git diff --name-only -- internal cmd migrations`; adapter-family matrix command | none | Do not replace full commands with package subsets or infer forbidden-change safety from a branch label | Unregistered tests, forbidden paths, or language-specific branching fails |
| TC-012 | Retained real run uses public `shark next`, `claim`, `heartbeat`, `status advance`, `release`; resume cases use the adapter capability fixtures below | Stub only provider output; real Shark binary and scratch project for retained run | Do not mock the Shark command sequence, I-07 persistence, resume decision, or wire event parser | A terminal status/exit code without semantic outcome, release, or exact wire evidence fails |
| TC-013 | Exact argv: `shark next Q-E40-F08-001 --json`; `shark claim Q-E40-F08-001 --by bench-tc013 --json`; `shark question respond Q-E40-F08-001 --session SID-Q --responder responder-a --summary approved --evidence-pointer runs/tc013/answer.json`; `shark question resolve Q-E40-F08-001 --owner owner-a --resolution-kind accepted --resolution-pointer runs/tc013/resolution.json` | Stub only external replay/provider response source; real Question CLI path and scratch state | Do not inject an answer into I-07 or bypass `question respond`/`resolve` | Transcript-only, unauthorized owner/responder, duplicate, or unused-entry response fails |

## Acceptance Test Cases

### TC-001: I-07 schema and closed-vocabulary validation

**Feature requirement:** I-07 lifecycle run record and `REQ-F-015`.
**Task acceptance criteria:** AC-001, AC-009.

**Technique Applied:** Equivalence partitioning + attack-class enumeration.
**ISO 25010:** Functional suitability, compatibility, reliability, security,
maintainability, portability.

**Preconditions/Input:** Run `verify-lifecycle-run.sh` against valid fixtures,
then for every top-level and nested I-07 field enumerate missing, JSON null,
wrong type, empty string/list/object, oversized value, unexpected extra field,
malformed digest, and cross-field mismatch. Also test unsupported outcome/gate
vocabulary, duplicate/non-monotonic ordinals, missing stop reason, prompt
digest mismatch, missing model/usage identity, and stop with
`publication_eligible: true`.

**Executable field inventory:** `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061`
loads `bench/runs/i07-schema.yaml` with `yq`, walks every declared
`required_fields` and `properties` entry, and creates one named fixture under
`tests/contracts/testdata/e40_i07/invalid/field-attacks/<json-pointer>/` for
missing, null, wrong type, empty/boundary-size, and unexpected-field cases.
The explicit field pointers are `/identity/{schema_version,run_id,scenario_id,
scenario_version,fixture_id,fixture_digest,adapter_id,adapter_version,
shark_binary_digest,shark_content_digest,roots}`;
`/entity_graph/{root_key,root_type,resolved_via,fork_candidates,selected_keys,
selected_types,ordinals,ineligible}`; `/dispatches[]/{ordinal,requested_key,
response,claim,worker,heartbeats,outcome,transition,release,started_at,ended_at,
evidence_refs}` where `response` is the complete `NextResponse` and `worker`
is `{worker_id,session_id,kind,recommended_outcome,evidence}`;
`/stages[]/{stage,category,snapshot_digest,prompt_digest,input_lineage,
replay_lineage,output_paths,output_digests,usage,cost_usd,elapsed_seconds,
errors,rework,intervals,candidate,artifacts,access_events}` where each
artifact is `{artifact_type,path,digest,size_bytes,producer_stage,consumers}`;
`/workflow_policy/{enabled_gates,gate_order,reviewer,prompt_digest,
review_bundle_digest,fixes_allowed_between_gates}`;
`/review_gates[]/{gate_id,state,round,findings,candidate_ref,policy_ref}` where
each raw finding is `{gate,round,severity,defect_class,fingerprint,criterion,
test,disposition,metadata,raw_bytes}`; `/questions[]/{question_key,
current_responder,owner,authorized_entry_digest,request_digest,response_digest,
evidence_pointer,terminal_result}`; `/limits/{max_cost_usd,
max_wall_clock_seconds,max_generated_tasks,observed_cost_usd,
observed_wall_clock_seconds,observed_generated_tasks,first_exceeded}`; and
`/outcome/{terminal,reason,partial_evidence,publication_eligible}`.

The test runs `jq -e` assertions for every schema-declared closed vocabulary,
integer ordinal uniqueness/monotonicity, SHA-256 recomputation, digest/byte
equality, interval containment, candidate/policy/finding/Question joins,
stop/release consistency, and `publication_eligible=false` for each invalid
fixture. Every failure must name the JSON pointer, observed value class, source
fixture, and reason in `events.ndjson` and stderr.

**Auditable partition matrix:**

| I-07 block | Fields covered by missing/null/type/empty/oversized/extra/digest or cross-field partitions |
|---|---|
| `identity` | schema version, run ID, scenario/version, fixture identity, adapter identity, Shark binary/content identity, roots |
| `entity_graph` | root, `resolved_via`, fork candidates, selected keys/types, ordinals, ineligibility reasons |
| `dispatches` | response identity/type/status/action/agent/provider/model/effort, prompt/digest/bytes, claim session, worker identity, outcome, transition, release, timing, evidence references |
| `stages` | snapshot, category, usage/cost, interval ledger, candidate reference, artifact producer/consumer/access events |
| `workflow_policy` | enabled gates, order, reviewer provider/model/effort, prompt digest, review-bundle digest, fix policy |
| `review_gates` | gate identity/state, round, raw finding gate/severity/class/fingerprint/criterion/disposition/metadata, candidate/policy references |
| `questions` | Question key, owner/responder handoff, authorized response reference, evidence pointer, terminal result |
| `limits` | positive cost/wall/task ceilings, observed consumption, first exceeded ceiling |
| `outcome` | terminal vocabulary, partial-evidence flag, publication eligibility, non-empty reason |

Each row is tested with the common malformed partitions plus block-specific
cross-field pairs (digest↔bytes, ordinal↔dispatch count, stop↔eligibility,
finding↔gate, stage intervals↔wall time, candidate↔policy, and Question
response↔authorization). This closes the field enumeration without treating an
unbounded fuzz corpus as the acceptance model.

**Expected outcome:** The valid fixture passes. Every invalid fixture fails,
names the field/value, retains the reason in the I-07 invalidity evidence, and
cannot be publication eligible.

**Observability:** `lifecycle.validation.failures{field}` and a structured
validation log include the run ID and named reason.

**Edge/negative cases:** Empty `findings: []` is valid only for
`zero_findings`; absent `review_gates` for a reached gate fails. A `complete`
run may be eligible; every named stop must be ineligible with a non-empty reason.
The shared implementation contract is
`tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061`; this plan's
TC-001 is its QA design case, not a twin contract test.

### TC-002: Contract boundary and exact prompt handoff (X-11)

**Feature requirement:** `REQ-F-002`, `REQ-F-004`, `REQ-F-006`; X-11.
**Task acceptance criteria:** AC-002.

**Technique Applied:** Contract-surface enumeration + state transition.
**ISO 25010:** Functional suitability, compatibility, usability, security,
reliability, maintainability, portability.

**Input:** One admitted root whose stubbed
`shark next <root> --json --prompt-out <path>` response contains
`entity_key=E40-F08-CHILD-001`, status `development`, action `spawn_agent`,
prompt bytes `run exact bytes\n`, digest and byte count, provider/model/effort,
`resolved_via`, `unresolved_placeholders`, and every Question handoff field.

**Expected outcome:** The response is persisted before worker dispatch; the
worker receives byte-for-byte `run exact bytes\n`; stored digest/bytes match;
the worker request has no claim/advance/release/heartbeat authority. The
record preserves all named routing and Question fields.

**Executable assertions:** `bench/scripts/testdata/lifecycle/next-response-complete.json`
contains every `NextResponse` wire field from
`internal/cli/commands/next.go:143-205`: `entity_key`, `entity_type`, `status`,
`action`, `agent_type`, `provider`, `model`, `effort`, `prompt`,
`prompt_sha256`, `prompt_bytes`, `resolved_via`, `unresolved_placeholders`,
`error`, `question_block`, and `current_responder`. The real controller passes
that JSON to `bench/scripts/lifecycle-worker-adapter.sh`; the real-binary
canary records the argv array and request bytes in
`bench/runs/tc002/adapter-argv.ndjson`. Assertions compare the `--prompt-out`
file, request `prompt`, SHA-256, and byte count, then scan every file under
`bench/runs/tc002/` for sentinel prompt text, `credential-sentinel`,
`provider-secret-sentinel`, and `transcript-line-999`. Only hashes, bounded
evidence, paths, sizes, and access metadata may remain. `worker_id`,
`session_id`, `kind`, conditional `recommended_outcome`, and bounded `evidence`
must round-trip; any mutation authority field in the adapter request fails.

**Edge/negative cases:** Digest mismatch and byte-count mismatch stop with
`error` before provider invocation. A missing required identity is invalid.
Absent prompt and `--prompt-out` are separate partitions; when present, file
bytes and response bytes must match exactly.

### TC-003: Parent-owned lease and semantic transition cleanup (X-11)

**Feature requirement:** `REQ-F-005`; X-11.
**Task acceptance criteria:** AC-003.

**Technique Applied:** State transition + decision table.
**ISO 25010:** Functional suitability, performance, reliability, security,
compatibility, maintainability, portability.

**Input:** Run the real controller for success, worker failure, cancellation at
each command boundary, exception, transition failure, heartbeat failure,
missing worker outcome, release failure, and duplicate cleanup attempt. Use
returned key `TASK-002`, session `SID-002`, original status `development`,
claim TTLs of 1 and 180 seconds, and cadence `max(1s,min(60s,TTL/3))`.

**Expected outcome:** Claim uses returned key; heartbeat uses `SID-002` at the
configured cadence; advance uses returned outcome, original status, and same
session; release uses same key/session on every path. A failed heartbeat maps
to `lease_loss`; missing result maps to `missing_outcome`; no worker mutates
Shark state.

**Edge/negative cases:** Claim failure emits no worker call; release failure is
retained as evidence; heartbeat at the TTL boundary fails closed; release is
once-only on every path; a process exit code alone never becomes `complete`.

**Planned executable boundary matrix:** `tc061_lifecycle_runner_loop_test.sh` is intended to set
`LIFECYCLE_CLOCK_FILE=bench/scripts/testdata/lifecycle/clock/{before,equal,after}-ttl.ndjson`
and `SHARK_CLAIM_TTL_SECONDS=3`. The file-backed clock records claim at
`2026-08-17T12:00:00Z`, an allowed heartbeat at `12:00:00.999Z`, an equal
boundary at `12:00:03Z`, and a late heartbeat at `12:00:03.001Z`. Assertions
require `heartbeat_time < claim_time+ttl`; no cadence tolerance may admit an
equal or later heartbeat. For injected fixtures `claim-failure`,
`adapter-failure`, `heartbeat-failure`, `advance-failure`, `release-failure`,
`cancel-before-next`, and `missing-outcome`, parse `events.ndjson` and assert
exact argv order/count, key, session, original `from-status`, outcome, and
exactly one release for every successful claim.

### TC-004: Deterministic fork scheduling and complete descendant execution

**Feature requirement:** `REQ-F-003`, `REQ-F-007`; UAT-11.
**Task acceptance criteria:** AC-004.

**Technique Applied:** Boundary-value analysis + ordering/uniqueness
partitioning.
**ISO 25010:** Functional suitability, compatibility, reliability,
maintainability, portability.

**Input:** Partition `parallel_candidates` into zero, one, two, three, and
many candidates. For the many case return `TASK-003`, `TASK-001`, and
`TASK-002` in that order, then test nested forks, duplicate keys, duplicate
descendants across responses, malformed selection metadata, changed candidate
sets, and a cycle.

**Expected outcome:** The fork response and all three candidates are retained;
dispatch ordinals follow canonical key order `TASK-001`, `TASK-002`,
`TASK-003`; each executes exactly once and no sibling is silently discarded.

**Edge/negative cases:** One ineligible candidate gets a durable reason; zero
and one candidates are valid boundaries; duplicate/cyclic/changed sets fail
validation. Every eligible descendant appears exactly once or the run is
explicitly ineligible.

**Planned executable fork canary:** `bench/scripts/testdata/lifecycle/bin/argv-canary`
is a real executable invoked by the adapter; the PATH stub replaces only the
external `shark` binary. It writes JSON arrays, not a shell-joined string, to
`bench/runs/tc004/canary-argv.ndjson`, including `argv`, `RUNNER_ID`,
`response_schema`, `fork_parent`, `resolved_via`, and the complete candidate
envelope. The test feeds `fork/{zero,one,two,three,four,nested,duplicate-key,
duplicate-descendant,malformed-selection,changed-set,cycle}.json`; `jq` asserts
canonical key order, one ordinal per eligible key, complete fork-response
preservation, no fixture-order selection, and a named `ineligible` reason for
every candidate not dispatched.

### TC-005: First-exceeded resource ceilings stop the whole scenario

**Feature requirement:** `REQ-F-008`, `REQ-F-015`; UAT-12.
**Task acceptance criteria:** AC-005.

**Technique Applied:** Boundary-value analysis + decision table.
**ISO 25010:** Functional suitability, performance, compatibility, reliability,
maintainability, portability.

**Input:** Three otherwise eligible descendants and positive limits, tested
independently: `max_cost_usd=0.01`, `max_wall_clock_seconds=1`, and
`max_generated_tasks=1`. Arrange consumption to exceed exactly one limit after
the current stage. Also test equality, zero/one/many task limits, simultaneous
breaches, one-stage overshoot, root-only counting, and unavailable clock/usage
observations.

**Expected outcome:** The first exceeded limit emits `resource_limit`, retains
prior/current partial I-05/I-07 evidence, sets `publication_eligible=false`,
and prevents the later sibling dispatch. The root is not counted as a
generated task.

**Edge/negative cases:** At exactly the limit execution remains allowed;
non-positive/missing policy is rejected before provider work; a partial record
never claims `complete`; unavailable observations are explicit invalid stops,
not zero consumption.

**Planned executable observation matrix:** `tc062_lifecycle_runner_limits_test.sh`
reads I-04 `resource_policy` from
`bench/scripts/testdata/lifecycle/limits/{cost,wall,tasks,equality,first-exceed,
simultaneous,overshoot,unavailable}.yaml`, usage from
`usage/{equal,exceed,read-error}.ndjson`, and time only from the named clock
file. Equality dispatches; the first strict exceed writes `first_exceeded` in
policy order, `terminal=resource_limit`, a non-empty `reason`,
`partial_evidence=true`, and `publication_eligible=false`, then stops before
the next sibling. Simultaneous breaches retain all observations plus one
`first_exceeded`. For every prior `claim` event, the parser requires a matching
`release` event after the stop, including provider-failure and release-failure
paths; a release error is retained and cannot reopen dispatch.

### TC-006: Review-gate state and raw finding preservation

**Feature requirement:** `REQ-F-011`–`REQ-F-013`; UAT-17.
**Task acceptance criteria:** AC-006.

**Technique Applied:** Equivalence partitioning.
**ISO 25010:** Functional suitability, reliability, security, maintainability,
portability.

**Input:** Four reached/unreached gate fixtures: two raw findings with gate,
round, severity, defect class, fingerprint, criterion/test, disposition and
candidate/policy identity; empty collector; collector error; and unreached
gate. Add duplicate gate records, missing gate identity, malformed finding
metadata, multiple rounds/findings, missing candidate/policy references, and
an empty-but-present collector.

**Expected outcome:** States are respectively `findings`, `zero_findings`,
`collection_failure`, and `not_reached`; raw finding bytes/metadata remain
unchanged and join to the exact candidate and workflow-policy identity.

**Negative case:** An absent collector result must not be recorded as
`zero_findings`; raw bytes remain unchanged and confirmation/deduplication is
not performed by F08.

**Planned executable invalidity matrix:** `tc063_review_finding_capture_test.sh` is intended to run
`review/{findings,zero-findings,collector-failure,unreached,duplicate-gate,
missing-gate-id,malformed-finding,missing-candidate-ref,missing-policy-ref,
multiple-rounds}.json` through the real collector file path and then
`verify-lifecycle-run.sh`. It asserts one state per reached gate, distinguishes
absent collector from `findings: []`, rejects duplicate gate records and every
malformed raw field, and compares SHA-256 plus byte-for-byte contents of each
`raw_bytes` value before and after serialization. Finding fields `gate`,
`round`, `severity`, `defect_class`, `fingerprint`, `criterion`, `test`,
`disposition`, and `metadata` must remain unchanged and join the exact
`candidate_ref` and `policy_ref`.

### TC-007: I-04 admission and I-06 replay gating (I-04/I-06)

**Feature requirement:** `REQ-F-001`, `REQ-F-009`, `REQ-F-014`; UAT-12.
**Task acceptance criteria:** AC-007.

**Technique Applied:** State transition + equivalence partitioning.
**ISO 25010:** Functional suitability, compatibility, usability, reliability,
security, maintainability, portability.

**Input:** Validate the complete I-04 package through
`e40_i04_scenario_contract_test.go#TC-030`, including identity/version/family,
stage matrix, fixture/adapter, visible input, replay/evaluator references,
resource policy, final predicate, and admission. Then use a valid admitted
feature package with D01-D05 replay; a missing replay entry; unauthorized
response; and valid bug, change-card, and tech-debt packages. Exercise I-06
`TC-052` for request/action matching, unused/duplicate entries, malformed
response, response counts/sizes, artifact digests, revision counts, and
terminal prelude outcome.

**Expected outcome:** Feature lifecycle dispatch starts only after D01-D05;
missing/unauthorized replay produces `unresolved_gate`, no provider call, and
durable partial evidence. Non-feature families receive explicit
`not_applicable` prelude records and continue to their roots. Missing or
non-admitted I-04 package is rejected before dispatch.

**Negative case:** Worker prose cannot answer or resolve a Question.

**Executable chain:** TC-007 first runs the real shared contract tests
`tests/contracts/e40_i04_scenario_contract_test.go#TC-030` and
`tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052` against
their committed fixture trees. It then copies a valid I-04 package and I-06
bundle into the scratch project and invokes the real controller. The feature
fixture `prelude/valid-d01-d05` consumes each unused entry once and emits entry
digest, request/response counts and sizes, revision count, artifact digests,
and terminal prelude outcome before the first lifecycle `next` event.
`prelude/{missing-entry,duplicate-entry,wrong-request,malformed-response,
unauthorized-response}` must emit `unresolved_gate` with no adapter/provider
start. For an authorized Question, the controller must execute the exact
`next`, `claim`, `question respond`, and `question resolve` argv in the caller
table; `shark question get Q-E40-F08-001 --json` must show the authorized
responder, configured owner, evidence pointer, and terminal resolution. An
unused authorized entry, wrong responder/owner, duplicate response, or missing
resolution is a failure, not a successful replay.

### TC-008: Evaluator disclosure guard before provider dispatch (I-05)

**Feature requirement:** `REQ-NF-003`; UAT-09.
**Task acceptance criteria:** AC-008.

**Technique Applied:** Attack-class enumeration.
**ISO 25010:** Functional suitability, performance, compatibility, reliability,
security, maintainability, portability.

**Input:** Plant evaluator-only material as a regular file, nested file,
symlink, broken symlink, traversal path, renamed evaluator root, and material
introduced after admission, separately in fixture and scratch visible roots.

**Expected outcome:** Existing I-05 disclosure guard fails before provider
invocation, records the preflight failure and named root, and emits no worker
call or provider spend.

**Negative case:** A guard that checks only the evaluator root, or runs only at
scenario start, or misses symlink/path-traversal visibility, fails this test.

**Executable guard matrix:** `tc060_lifecycle_runner_contract_test.sh` plants
`evaluator-attacks/{regular/note.txt,nested/a/b/note.txt,symlink-to-file,
symlink-to-dir,broken-symlink,../../evaluator-only/late.txt,renamed-root/answer,
late-admission/answer}` separately under both
`bench/scripts/testdata/lifecycle/roots/fixture/` and `roots/scratch/`. The
controller must invoke the existing `bench/scripts/verify-evidence-roots.sh`
immediately before *each* `spawn_agent`; the canary log must contain zero
provider starts and the I-07 preflight event must name the visible root,
resolved path, attack kind, and non-empty reason. A late-created file is
checked on the second dispatch, not only admission.

### TC-009: I-05 evidence, candidate/policy identity, and artifact joins

**Feature requirement:** `REQ-F-010`–`REQ-F-013`; UAT-16/UAT-18/UAT-19.
**Task acceptance criteria:** AC-009.

**Technique Applied:** Attack-class enumeration + contract-surface
enumeration.
**ISO 25010:** Functional suitability, compatibility, reliability, security,
maintainability, portability.

**Input:** Valid retained stage snapshots with non-overlapping category
intervals, candidate snapshot, artifact producer/consumer access, usage/model,
and workflow policy. Mutate one at a time: prompt/input/replay/output lineage,
cost/error/rework/digest, evaluator-access ordering, three-root identity,
base/tree/binary-diff/changed-path/dirty-untracked/test-suite candidate
identity, enabled gates/order/reviewer/provider/model/effort/prompt digest/
review-bundle digest/fix policy, missing usage/model, prompt digest, candidate,
artifact evidence, time interval, or policy field; use unknown stop outcome and
cross-reference mismatches.

**Expected outcome:** Valid joins reconcile stage/lifecycle time and pass.
Each mutation fails closed, names the exact divergence, retains the invalidity
reason, and cannot enter publication aggregates.

**Negative case:** Missing evidence is never synthesized as zero, success, or
an empty artifact-consumption list. The test runs existing I-05 validators and
joins a real file-backed retained artifact before applying copied-file mutations.

**Executable retained-run gate:** First run the controller with
`bench/scripts/testdata/lifecycle/e2e/retained-i05-input/` and assert that the
controller itself writes, before any mutation test, `prompt_digest`, input/
replay/output lineage, usage/model, cost/error/rework/digest, non-overlapping
intervals, evaluator-access events, all three root identity digests, and the
candidate fields `base_commit`, `tree_digest`, `binary_diff_digest`,
`changed_path_digest`, `dirty_untracked_manifest`, and `test_suite_digest`.
It must also write workflow-policy identity and the full review-bundle digest.
`jq` and `sha256sum` compare those fields to the retained I-05 files and the
controller's `lifecycle.jsonl` before copying it to
`testdata/e2e/mutations/`. One mutation per field, plus wrong run ID, ordinal,
path, digest, interval, access phase, candidate/policy reference, and missing
`consumers` versus `consumers: []`, must fail with a named invalidity reason.

### TC-010: Offline contract/dry-run determinism

**Feature requirement:** `REQ-NF-002`, `REQ-NF-008`; UAT-18.
**Task acceptance criteria:** AC-010.

**Technique Applied:** Reproducibility/replay testing.
**ISO 25010:** Functional suitability, performance, compatibility, reliability,
security, maintainability, portability.

**Input:** Run `run-lifecycle.sh --mode dry-run` and `--mode contract` twice
against identical committed fixtures. Provider denial is a real executable
that records an attempted invocation and exits non-zero; assert zero adapter/
provider process starts. Repeat with filesystem ordering, locale, shell, and
JSON/map key ordering varied.

**Expected outcome:** Both repeated canonical verdict projections are
byte-identical: the test pins run ID, fixture paths, timezone, random seed,
PID-independent serialization, and a file-backed clock. Volatile timestamps
are excluded only by the declared canonical projection; dispatch ordinals and
canonical JSON are stable, and provider invocation count is zero.

**Negative case:** Provider output, filesystem ordering, locale, shell,
timezone, PID, random seed, environment variables, or map iteration must not
alter the canonical projection; any provider attempt fails the test.

**Executable determinism guard:** `bench/scripts/testdata/lifecycle/bin/provider-deny`
is executable, appends its exact argv and `PROVIDER_DENIED=1` to
`bench/runs/tc064/provider-attempts.ndjson`, and exits 97. `tc064` runs each
mode twice with the same `--run-id`, `TZ=UTC`, `LC_ALL=C`, fixed seed, sorted
and reversed fixture directory entries, two shells, and reordered JSON maps.
It compares `lifecycle.verdict.json` after the schema-declared canonical
projection and asserts the denial log is empty, no adapter process starts,
and no provider command appears in any `events.ndjson`.

### TC-011: Quality gate and boundary-file registration

**Feature requirement:** `REQ-NF-001`, `REQ-NF-006`–`REQ-NF-008`; UAT-11.
**Task acceptance criteria:** AC-011.

**Technique Applied:** Structural/configuration testing.
**ISO 25010:** Functional suitability, performance, compatibility, reliability,
security, maintainability, portability.

**Input:** Run `make fmt && make lint && make test`, then
`bench/scripts/tests/run-all.sh`; inspect the diff and registration for
TC-060 through TC-066.

**Expected outcome:** All commands pass; new tests are registered; no
`internal/`, `cmd/`, database, migration, workflow-engine, or second-store
files change. A structural scan plus Python, Go, package-manager, test, and
lint fixture-family matrix proves those operations reach the I-04 adapter;
generic scripts do not branch on fixture language.

**Negative case:** A green isolated test without `run-all.sh` registration, or
any forbidden product/database change, fails the gate.

**Executable matrix:** `tc060` through `tc064` are each named in
`bench/scripts/tests/run-all.sh`; the test runs `git diff --name-only --
internal cmd migrations` and fails on any output, then scans generated files
under `bench/` and `docs/plan/` for evaluator-only paths. The adapter-family
fixture matrix is `adapter-families/{python,go,package-manager,test,lint}/`;
each package has `package.yaml`, a family-specific adapter declaration, and a
recorded command. The parser asserts that all five reach the same I-04
`adapter` field and that `run-lifecycle.sh`, `verify-lifecycle-run.sh`, and
the generic test runner contain no `case`/`if` branch on `python`, `go`, or
package-manager names. `make fmt`, `make lint`, `make test`, and
`bench/scripts/tests/run-all.sh` remain required; no product code change is
made by this plan.

### TC-012: Retained real keyed lifecycle UAT (X-11)

**Feature requirement:** Feature acceptance boundary and `REQ-F-002`–`REQ-F-007`;
UAT-11.

**Technique Applied:** End-to-end state transition + contract testing.
**ISO 25010:** Functional suitability, performance, compatibility, reliability,
maintainability, portability.

**Input:** A retained UAT scenario with one root, at least three eligible
descendants, real keyed Rider dispatch, bounded worker result, and stage
evidence.

**Expected outcome:** Every eligible entity has dispatch, claim, heartbeat,
unchanged prompt handoff, semantic outcome, configured transition, release,
and I-07 evidence. The retained artifact is sufficient to inspect scheduling,
candidate identity, and stage categories.

The retained run also exercises bounded resume: same-worker follow-up,
immutable replacement-worker fallback, unsupported retirement, and refusal to
advance after an unacknowledged background worker.

**Planned executable X-11 decision table:** `tc061` is intended to run four fixtures:
`resume/same-worker`, `resume/immutable-replacement`,
`resume/unsupported-retirement`, and `resume/unacknowledged-background`.
The first sends the same `worker_id` and immutable handoff to the documented
follow-up argv; the second starts exactly one replacement from the saved
handoff digest; the third records `retirement_unsupported`, releases, and
stops; the fourth records the terminal worker envelope, releases, and proves
no `shark status advance` argv was emitted. For each case, the parser asserts
ordered I-07 wire events `dispatch -> claim -> heartbeat* -> worker_result ->
(advance|no_advance) -> release`, exact `entity_key`/`session_id`, handoff
digest, worker identity, and terminal reason.

**Negative case:** A terminal Shark status or worker exit code without the
held-back execution oracle is not treated as proof of correctness/publication.

### TC-013: Durable Question authorized/blocked lifecycle (X-13)

**Feature requirement:** `REQ-F-009`, `REQ-F-014`, `REQ-F-015`; UAT-12.

**Technique Applied:** State transition + equivalence partitioning.
**ISO 25010:** Functional suitability, compatibility, usability, reliability,
security, maintainability, portability.

**Input:** One authorized replay response with Question key, owner/responder,
evidence pointer, and terminal result; repeat with missing and unauthorized
responses.

**Expected outcome:** Authorized response is recorded through the public
Question lifecycle and joins I-07. Missing/unauthorized input produces named
`unresolved_gate`, partial evidence, `publication_eligible=false`, and no
invented worker decision. Assertions use exact public Question dispatch,
response, and resolution arguments, including owner/responder authorization,
unused-entry matching, and durable terminal resolution.

The exact command path is `shark next <question-key> --json`,
`shark claim <question-key> --by <runner> --json`,
`shark question respond <question-key> --session <sid> --responder <id>
--summary <summary> --evidence-pointer <pointer>`, then
`shark question resolve <question-key> --owner <owner> --resolution-kind
<kind> [--resolution-pointer <pointer>]`.

**Negative case:** Transcript-only text or a worker recommendation cannot close
the Question.

**Planned executable X-13 assertions:** The authorized fixture must show the exact
`question_block.question_key`, `current_responder`, claim `session_id`,
`--responder`, `--owner`, `--evidence-pointer`, `--resolution-kind`, and
`--resolution-pointer` in `events.ndjson`, plus the post-resolution Question
JSON with terminal status. Missing, unauthorized, duplicate, wrong-request,
and unused authorized entries must each produce one `unresolved_gate` event,
`publication_eligible=false`, no `status advance`, and no worker decision.

### TC-014: Named-stop and resume/incomplete-record matrix

**Feature requirement:** `REQ-F-005`, `REQ-F-007`, `REQ-F-015`, `REQ-NF-005`;
UAT-12/UAT-16.

**Technique Applied:** State-transition matrix + boundary-value analysis.

**ISO 25010:** Functional suitability, reliability, security, maintainability,
portability.

**Input:** Inject `pause`, `archive`, `error`, `lease_loss`, `missing_outcome`,
`cancellation`, `worker_failure`, and `timeout` at each dispatch boundary,
including interruption after a flush and before the next dispatch.

**Expected outcome:** Exactly one named terminal outcome, non-empty reason,
partial evidence retained, publication ineligible, and no incomplete JSONL
record presented as complete. A resumable diagnostic artifact identifies the
last flushed ordinal.

**Negative case:** Stop is not relabeled `completed`, and a later sibling is
not dispatched after a scenario-level stop.

**Executable stop/resume assertion:** `tc062` and `tc061` parse every
`events.ndjson`/`lifecycle.jsonl` record after injecting each stop at
`before-next`, `after-next`, `after-claim`, `after-heartbeat`, `after-worker`,
`after-advance`, and `after-release`. Each run has exactly one terminal
`outcome.terminal`, a non-empty `reason`, `partial_evidence=true`,
`publication_eligible=false`, a flush marker for the last ordinal, and no
subsequent dispatch. `verify-lifecycle-run.sh` rejects truncated JSONL and
accepts a diagnostic resume only when the saved ordinal, entity key, and
evidence digest match.

## Test Infrastructure

Existing patterns to follow:

- `bench/scripts/tests/tc043_root_policy_isolation_test.sh` through
  `tc059_replay_offline_determinism_test.sh` for shell fixtures, PATH-stubbed
  commands, offline guards, evidence joins, and determinism.
- `bench/scripts/tests/run-all.sh` for registration and the bench test tier.
- `tests/contracts/e40_i04_scenario_contract_test.go`,
  `e40_i05_stage_evidence_contract_test.go`, and
  `e40_i06_product_design_replay_contract_test.go` for contract-test naming,
  committed fixtures, and producer/consumer shape checks.
- `internal/cli/commands/next.go`, `claim.go`,
  `internal/services/claim_service.go`, and
  `skills/shark-rider/context/host-adapter-contract.md` for production caller
  arguments and lease/prompt/result semantics.

New test utilities/fixtures required:

- `bench/scripts/tests/tc060_lifecycle_runner_contract_test.sh` through
  `tc064_lifecycle_runner_offline_determinism_test.sh` and their registered
  entries in `run-all.sh`.
- `bench/scripts/testdata/lifecycle/` with stub Shark, worker, heartbeat,
  transition, release, Question commands and complete/fork/stop/finding/
  malformed fixtures. Keep evaluator-only material outside agent-visible roots.
- `tests/contracts/e40_i07_lifecycle_run_contract_test.go` and
  `tests/contracts/testdata/e40_i07/{valid,invalid}/`.
- No real database is needed: F08 is file-backed bench code. If a test reaches
  a repository implementation, it must follow the repository real-DB rule;
  controller and CLI boundaries use stubs/mocks at the command seam.

## Codex Test-Plan Red-Team

**Verdict:** superseded historical review record
**Issues raised:** 29 across three red-team passes
**Issues addressed before dev:** 29
**Issues deferred:** 0

The review checked open-endedness, technique fit and enumeration, ISO coverage,
runtime observability, negative cases, and every caller-path contract. The
initial result was FAIL with 19 findings; all were addressed in the expanded
partitions, exact command shapes, real-join seams, observability assertions,
staged disposition fields, and status correction above.

### Codex output (verbatim)

```text
FAIL

1. AC-001/TC-001: “Each malformed field” is open-ended but the attack model is not enumerated. Add a field-by-field matrix covering missing, null, wrong type, empty, oversized, extra, malformed digest, duplicate, and cross-field inconsistency cases for every I-07 field and vocabulary.
2. AC-002/TC-002: The test does not enumerate all `NextResponse` fields required by the spec, including unresolved placeholders, all Question handoff fields, absent-prompt behavior, and `--prompt-out` byte equivalence. Add exact assertions for the complete response before dispatch.
3. AC-003/TC-003: Lease coverage omits exact TTL-boundary behavior, minimum one-second cadence, heartbeat timing drift, cancellation during every command, duplicate cleanup, and cleanup when release itself fails. Add a decision table for each lifecycle command failure and assert exact command order, count, key, session, and status arguments.
4. AC-004/TC-004: Three candidates are not sufficient enumeration for fork behavior. Missing are zero/one/two candidates, more-than-three candidates, nested forks, duplicate keys, duplicate descendants across responses, malformed selection metadata, changed candidate sets, and cycles. Replace “three is the minimum” with explicit boundary partitions and a completion invariant for every eligible descendant.
5. AC-005/TC-005: Boundary-value analysis is incomplete. It lacks zero/one/many generated-task limits, exact cumulative cost/time values, simultaneous ceiling breaches, overshoot caused by a single stage, root-versus-descendant counting, and clock/usage read failures. Add equality, first-exceeded, simultaneous, and unavailable-observation cases.
6. AC-006/TC-006: Review-gate enumeration does not cover duplicate gate records, missing gate identity, malformed raw finding metadata, multiple rounds, multiple findings, missing candidate/policy references, or a reached gate with an empty-but-present collector result. Add those partitions and verify raw bytes plus every required metadata field remain unchanged.
7. AC-007/TC-007: I-04/I-06 coverage is too shallow. It does not validate the complete I-04 payload or I-06 replay sequence, request/action matching, unused-entry consumption, duplicate entries, wrong request, malformed response, response-count/size metadata, artifact digests, revision counts, or terminal prelude outcome. Invoke the declared I-04 `TC-030` and I-06 `TC-052` contract tests explicitly, or add equivalent assertions.
8. AC-008/TC-008: The evaluator-disclosure attack model only plants a file. Add symlinks, nested paths, renamed evaluator roots, path traversal, broken symlinks, evaluator material introduced after admission, and visibility through both the fixture and scratch roots. Assert the real guard runs immediately before every dispatch and fails closed.
9. AC-009/TC-009: I-05 evidence enumeration omits prompt/input/replay/output lineage, cost/error/rework/digest fields, evaluator-access ordering, three-root identity, dirty/untracked/tree/binary-diff/test-suite candidate identity, and full review-bundle identity. The test also says “none beyond synthetic committed fixtures” while the requirement is to exercise existing I-05 joins; this can validate the validator while missing production wiring. Add retained real I-05 artifacts and one mutation per required field plus cross-reference mismatch cases.
10. AC-010/TC-010: Provider egress denial is described as a stub, which can hide an accidental provider call. Make the denial fail loudly, record attempted invocations, and assert zero adapter/provider process starts. Also vary filesystem ordering, locale, shell, and map/key ordering to test the stated determinism claim.
11. AC-011/TC-011: “No generic scripts branch on fixture language” has no concrete test method. Add a structural scan or fixture-family matrix proving Python, Go, package-manager, test, and lint behavior reaches the I-04 adapter without F08 language branches. Also reconcile the plan’s `TC-061` Go contract pointer with the test-plan’s `TC-001` naming.
12. AC-012/TC-012/TC-013: X-11 coverage does not test bounded resume: same-worker follow-up, immutable replacement-worker fallback, unsupported retirement, or the rule forbidding advance after an unacknowledged background worker. X-13 coverage does not assert the exact public Question request/response/resolution arguments, unused authorized-entry matching, responder/owner authorization, or durable terminal resolution. Add caller-path tests for those contracts.
13. Caller-path table, TC-002/TC-003/TC-007/TC-013: Several argument shapes remain placeholders or aliases rather than exact production invocations. Specify the resolved commands and flags for `shark claim`, `heartbeat`, `status advance`, `release`, Question dispatch, response, and resolution. Clarify whether PATH stubs replace one `shark` binary or separate subcommands; the current description is internally inconsistent.
14. Caller-path table, TC-005/TC-006/TC-009/TC-010: The lowest mock seams can hide wiring defects. Clock/usage stubs bypass I-05 collection, review collector stubs bypass gate serialization, replay/Question stubs bypass X-13 wiring, and synthetic committed fixtures bypass real artifact joins. Keep mocks only at external provider/process boundaries and run one real file-backed path through each existing validator/guard.
15. Observability matrix, AC-003/005/006/009: Metrics, logs, trace spans, and alert thresholds are named but no emitter, sink, schema, cardinality policy, or concrete assertion is specified. Add exact evidence locations and assertions for metric count/labels, structured log fields, span parentage, heartbeat cadence, stop reason, gate state, and validation failure. “Tests must inspect emitted records/logs” is not implemented by the current TC assertions.
16. ISO 25010 matrix, all ACs: Several `N/A` cells are unsupported, while applicable evidence is asserted only as a checkmark. In particular, AC-004 security, AC-005 security, and multiple usability/compatibility/performance omissions need explicit rationale; AC-011 performance and portability have no measurable evidence. Replace checkmarks with named assertions or justify each N/A against the actual feature boundary.
17. I-04/I-05/I-06/I-07 staged rows: The plan preserves `gate_mode`, activation owner, closure key, counterpart-status handling, and review basis, but omits the map-assigned `demonstrability_disposition: pending-integration`. Add that field and assert its exact value for every staged row. Also verify the full activation-owner/closure-key sets: I-04 has F06/F07/F08 slices, I-05 has F08/F09/F10 slices, I-06 has F08, and I-07 has F09/F10.
18. I-07/TC-001/TC-009: The plan treats publication eligibility as a validator concern but does not test all required identity components or the distinction between invalidity, partial evidence, and F09 aggregate eligibility. Add explicit cases for missing/disagreeing base commit, tree digest, binary diff, changed paths, dirty/untracked manifest, test-suite digest, and review-bundle identity.
19. Test-plan status, lines 6, 479–488, 492–508: The document claimed approval and “Ready for development” while its own red-team section was pending and the exit gate left Codex output unincorporated. Set the status and recommendation to pending until these findings are addressed and the red-team result is recorded.
```

**Issues raised:** 19
**Issues addressed before dev:** 19
**Issues deferred:** 0
**Final disposition:** The initial Codex verdict was FAIL; all findings are
addressed above. No finding remains deferred.

### Final Codex review result

The earlier review record is retained for traceability only. Its conclusion is
not an approval of this revision; the fresh rejection below required a
whole-surface sweep.

## Defect-Class Sweep

**Class sentence:** The test plan asserted broad lifecycle and evidence
contracts without an executable, field-level, production-wired verification
surface.

**Surface enumerated:** All 12 categories below were swept across every
relevant test case, not sampled: TC-001 through TC-014, the shared I-04/I-05/
I-06/I-07 contract pointers, X-11/X-13, the staged metadata rows, and the
observability and ISO matrices.

| # | Affected category and complete TC coverage | Concrete fix, source, fixture, guard, and assertion |
|---:|---|---|
| 1 | Schema-derived I-07 field attacks: TC-001, TC-004, TC-005, TC-006, TC-007, TC-009, TC-014 | `TC-061` loads every field from `bench/runs/i07-schema.yaml`; `tests/contracts/testdata/e40_i07/invalid/field-attacks/` covers missing/null/type/empty/size/extra and cross-field attacks for identity, graph, dispatch, stage, policy, gate, Question, limits, and outcome pointers. `jq -e` asserts field names, vocabularies, ordinals, digests, joins, stop eligibility, and named invalidity. |
| 2 | Complete host-adapter/NextResponse artifact redaction: TC-002, TC-003, TC-009, TC-010, TC-012, TC-013 | Source is `internal/cli/commands/next.go:143-205` and `skills/shark-rider/context/host-adapter-contract.md`. `next-response-complete.json`, `adapter-argv.ndjson`, and redaction sentinels are exercised through the real controller and adapter; `jq`, `sha256sum`, and recursive `rg` assert all 16 response fields, five bounded result fields, exact prompt transport, and no prompt/credential/secret/unbounded transcript in any emitted artifact. |
| 3 | Real-binary argv/fork canary: TC-002, TC-003, TC-004, TC-012, TC-013 | `bench/scripts/testdata/lifecycle/bin/argv-canary` is a real executable; PATH stubs replace only `shark`. It records JSON argv arrays, runner identity, response schema, and fork envelope. `tc061` asserts exact argv, no shell joining, canonical scheduling, complete candidate set, and no worker mutation authority. |
| 4 | File-backed clock and pre-TTL boundary: TC-003, TC-005, TC-010, TC-014 | `LIFECYCLE_CLOCK_FILE` fixtures `before-ttl`, `equal-ttl`, and `after-ttl` feed the controller. Assertions require `claim_time <= heartbeat_time < claim_time+ttl`; equal/later values yield `lease_loss`, with no tolerance that admits a post-TTL heartbeat. Clock and usage are real file reads, not in-process mocks. |
| 5 | First-exceeded ceilings and release-on-stop: TC-003, TC-005, TC-012, TC-014 | I-04 policy fixtures cover equality, strict first exceed, simultaneous breaches, overshoot, unavailable reads, root exclusion, and zero/one/many descendants. `tc062` asserts first-exceeded policy order, current/prior evidence, no later dispatch, `resource_limit`, ineligibility, and a release event for every owned session, including release failure. |
| 6 | Review-gate invalidity and raw bytes: TC-001, TC-006, TC-009, TC-012 | `tc063` uses findings/zero/collector-failure/unreached/duplicate/malformed/missing-reference/multi-round fixtures. The real collector and validator assert exactly one state per reached gate, absent versus empty collection, all raw finding fields, raw byte digest/equality, and exact candidate/policy references; F08 does not adjudicate. |
| 7 | Real I-04/I-06 chain and Question authorization: TC-007, TC-013, TC-012 | Real `TC-030` and `TC-052` run before the controller's file-backed chain. Prelude fixtures cover unused/duplicate/wrong/malformed/unauthorized entries and terminal mappings. Public `next`, `claim`, `question respond`, `question resolve`, and `question get` argv/output are asserted for responder/owner authorization, evidence pointer, durable resolution, and no transcript-only decision. |
| 8 | Live staged I-04..I-07 status/pointer/digest/owner/closure/review/gate/disposition evidence: TC-001, TC-007, TC-009, TC-011, TC-012 | `tc060` performs read-only `shark get E40-F05/F06/F07/F08 --json`, reads `E40-interaction-map.md`, computes contract-test `sha256sum`s, and writes `staged-contracts.json`. It asserts live counterpart status, exact pointers/digests, all owner/closure slices, `review_basis`, `gate_mode=contract-only`, and `demonstrability_disposition=pending-integration`. |
| 9 | Controller-written end-to-end I-05/I-07 evidence before mutation tests: TC-008, TC-009, TC-010, TC-012, TC-014 | Retained fixture `testdata/lifecycle/e2e/retained-i05-input/` runs the real controller and existing I-05 guards/joins first. `jq`/`sha256sum` assert every lineage, usage, cost, interval, access, three-root, dirty/untracked, test-suite, candidate, policy, and review-bundle field exists before copied-file mutations. |
| 10 | Adapter-family and forbidden-change matrix: TC-004, TC-008, TC-010, TC-011, TC-012 | `adapter-families/{python,go,package-manager,test,lint}/` reaches one I-04 adapter field and records its command. `run-all.sh` registration, `git diff --name-only -- internal cmd migrations`, generated-file scan, and source scan for language branches are executable guards. |
| 11 | All X-11 resume/retirement cases and wire events: TC-003, TC-004, TC-012, TC-014 | `resume/{same-worker,immutable-replacement,unsupported-retirement,unacknowledged-background}` is a decision table. Event parser requires `dispatch -> claim -> heartbeat* -> worker_result -> advance/no_advance -> release`, exact IDs, handoff digest, terminal reason, and no advance for an unacknowledged background worker. |
| 12 | Concrete observability parsers/assertions and characteristic-specific ISO/N-A rationale: TC-001, TC-002, TC-003, TC-005, TC-006, TC-007, TC-009, TC-010, TC-011, TC-012, TC-013, TC-014 | `jq -s` parsers read `events.ndjson`, `metrics.ndjson`, `trace.ndjson`, and `lifecycle.jsonl`; assertions cover bounded labels/counts, event fields/order, parent span IDs, timestamps, and error/alert records. A QA-only alert is explicitly a failing condition when the corresponding emitted artifact is absent. ISO cells below name the assertion or justify N/A against the file-backed/offline boundary. |

**Instances fixed:** The field matrix, exact public argv, real-binary canary,
file-backed clock, strict threshold semantics, release-on-stop, gate invalidity
matrix, raw-byte preservation, real I-04/I-06/X-13 chain, live staged-edge
read, retained controller-written evidence, adapter-family/forbidden-change
guards, all X-11 resume cases, and concrete observability/ISO assertions are
now attached to named TCs and artifacts above. The former stale verdict claims
and the “11 deferred” refinement disposition are removed.

**Structural guard:** `TC-061` is the single checklist/validator guard. It
derives required fields and vocabularies from `bench/runs/i07-schema.yaml`,
then validates every I/X staged metadata row and every emitted I-07/event/
metric/trace evidence file for required fields, pointers, digests, owner and
closure slices, live status, gate/disposition metadata, redaction, ordering,
and publication eligibility. `run-all.sh` must invoke this validator and all
TC-060..TC-066; any schema addition without a corresponding generated attack
fixture fails the contract test.

**No out-of-scope items:** This sweep changes only this test-plan document. It
does not implement the controller, alter Shark product code/database/workflow
state, adjudicate findings, score candidates, publish baselines, or run
provider-backed execution. Production defects discovered at those seams remain
owned by their existing epics.

## Recommendations

- [x] Ready for implementation planning
- [ ] Needs BA refinement
- [ ] Needs tech refinement

**Final recommendation:** APPROVED FOR IMPLEMENTATION PLANNING. The named defect class was swept across
all 12 categories and all relevant TCs; the structural guard is part of the
planned acceptance surface. This approval is for the test plan only, not for
implementation or shipped behavior.

## Exit-Gate Checklist

- [x] Feature PRD/feature intent and combined specification read and compared
- [x] Every AC has at least one concrete test case
- [x] Every runtime case has entrypoint, mock seam, forbidden mocks, and
  counter-factual
- [x] Edge and negative cases are enumerated
- [x] I-04/I-05/I-06/I-07 shared contract tests use declared pointers
- [x] X-11 and X-13 integration boundaries are covered
- [x] UAT-09, UAT-11, UAT-12, UAT-16, UAT-17, UAT-18, and UAT-19 are mapped
- [x] Test infrastructure and new helpers are named
- [x] Fresh Codex red-team blockers addressed and re-reviewed by this defect-class sweep
