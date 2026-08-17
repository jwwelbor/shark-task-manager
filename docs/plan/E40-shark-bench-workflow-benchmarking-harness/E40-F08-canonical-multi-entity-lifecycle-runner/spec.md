---
feature_key: E40-F08-canonical-multi-entity-lifecycle-runner
epic_key: E40
title: Canonical multi-entity lifecycle runner — combined requirements and architecture
date: 2026-08-16
---

# E40-F08 specification: Canonical multi-entity lifecycle runner

Business context is not restated here. See the E40 epic PRD sections
“Goals and Success Criteria” (G11, G12, G14, G16–G19), “Active scope”, and
“Constraints and Assumptions”, plus [feature.md](feature.md). System decisions
are in [the parent architecture](../architecture.md), especially the
[Lifecycle scenario package contract](../architecture.md#lifecycle-scenario-package-contract),
[Stage evidence and isolation contract](../architecture.md#stage-evidence-and-isolation-contract),
[Product-design replay contract](../architecture.md#product-design-replay-contract), and
[Lifecycle run record contract](../architecture.md#lifecycle-run-record-contract).

The validated feature research and Capability map are in
[research-report.md](research-report.md). F08 reuses I-04, I-05, I-06, X-11,
and X-13; it does not reimplement their owning capabilities.

## Requirements

### Functional requirements

| ID | Requirement | Trace |
|---|---|---|
| REQ-F-001 | The runner MUST accept one admitted I-04 scenario package, its declared roots, the scratch Shark project, and a versioned run identity. It MUST reject a missing or non-admitted package before provider dispatch. | Epic G8, G11 |
| REQ-F-002 | For each requested root, the runner MUST call `shark next <key> --json`, persist the complete response before scheduling work, and preserve `entity_key`, `entity_type`, `status`, `action`, `agent_type`, `provider`, `model`, `effort`, `prompt_sha256`, `prompt_bytes`, `resolved_via`, `unresolved_placeholders`, and Question handoff fields when present. | Epic G11, G14; X-11 |
| REQ-F-003 | For `parallel_candidates`, the runner MUST record the complete candidate set and the Shark selection metadata, then process every eligible candidate exactly once in a deterministic canonical-key order. It MUST re-enter keyed dispatch for each selected candidate and MUST NOT silently choose one candidate or use `--sequential` to discard siblings. | Epic G11 |
| REQ-F-004 | For every `spawn_agent` response, the runner MUST claim the concrete returned entity, pass the exact `response.prompt` bytes to the host adapter, and record the prompt digest and byte count independently before dispatch. A digest or byte-count mismatch MUST stop the scenario as `error`. | Epic G11, G14, G19; X-11 |
| REQ-F-005 | The parent controller MUST own the entity lease and workflow mutation. It MUST send session-scoped heartbeats while a worker runs, persist the worker's bounded semantic result, advance with the returned configured outcome and original status, and release the same session on success, failure, cancellation, and exception paths. Workers MUST NOT claim, heartbeat, advance, or release Shark entities. | Epic G11; X-11 |
| REQ-F-006 | The runner MUST preserve the worker adapter request/result fields from `skills/shark-rider/context/host-adapter-contract.md`, including worker identity, Shark session identity, control-envelope kind, recommended outcome, and bounded evidence. It MUST not reconstruct a prompt, status route, or provider persona. | Epic G11, G19; X-11 |
| REQ-F-007 | The runner MUST execute every eligible generated descendant or record a durable `ineligible` entry naming the concrete reason. A provider or worker process exit code MUST NOT substitute for the semantic outcome or execution oracle. | Epic G11, G13 |
| REQ-F-008 | The runner MUST enforce the I-04 positive `max_cost_usd`, `max_wall_clock_seconds`, and `max_generated_tasks` ceilings. It MUST check consumption after each stage and stop the entire scenario with `resource_limit` when any ceiling is reached, retaining partial I-05/I-07 evidence and marking the run ineligible. Generated-task count means dispatched non-root task descendants; the scenario root is not counted as a generated task. | Epic G12 |
| REQ-F-009 | A feature scenario MUST consume the completed I-06 prelude result before lifecycle dispatch. A bug, change-card, or tech-debt scenario MUST retain explicit non-applicable prelude records and continue to its lifecycle root. Missing or unauthorized replay input MUST stop the scenario as `unresolved_gate`. | Epic G10, G11; X-10, X-13 |
| REQ-F-010 | After each applicable dispatch, the runner MUST write or reference the I-05 stage snapshot, time ledger, candidate snapshot, artifact producer/consumer events, provider usage, and evaluator-access ordering. Missing evidence MUST remain explicit and MUST invalidate publication rather than being synthesized as zero or success. | Epic G9, G13, G16, G18 |
| REQ-F-011 | The run record MUST capture the workflow-policy identity active at each review-capable stage: enabled gates, gate order, reviewer provider/model/effort, prompt digest, full review-bundle digest, and whether fixes are allowed between gates. | Epic G14, G17, G19 |
| REQ-F-012 | For every reached review gate, the runner MUST record exactly one gate state: `findings`, `zero_findings`, or `collection_failure`; an unreached gate MUST be recorded separately as `not_reached`. It MUST preserve each raw `review-finding` note's gate, round, severity, defect class, fingerprint, affected criterion/test, disposition, and metadata without normalizing or adjudicating it. | Epic G17; I-07 |
| REQ-F-013 | Each captured finding MUST reference the exact candidate snapshot and workflow-policy identity that produced it, plus a later candidate reference only when the Shark record explicitly claims resolution. Finding confirmation, deduplication, recurrence, and truth comparison belong to F09. | Epic G17, G19 |
| REQ-F-014 | The runner MUST route a durable Question only through the authorized X-13 lifecycle and scenario replay. It MUST retain the Question key, responder/owner handoff, response evidence pointer, and terminal result. It MUST never infer an answer from worker prose or create a transcript-only decision. | Epic G11; X-13 |
| REQ-F-015 | The runner MUST terminate with one named scenario outcome: `complete`, or one I-05 stop outcome (`resource_limit`, `lease_loss`, `missing_outcome`, `unresolved_gate`, `pause`, `archive`, `error`, `cancellation`, `worker_failure`, or `timeout`). Every non-complete outcome MUST retain partial evidence, set `publication_eligible: false`, and include a non-empty reason. | Epic G11–G14 |

### Non-functional requirements

| ID | Requirement | Trace |
|---|---|---|
| REQ-NF-001 | F08 MUST add no Shark product code, database table, column, migration, workflow engine, claim store, Question store, or second status model. The runner is a file-backed bench adapter under `bench/`. | ADR-002, ADR-006; architecture Delivery boundaries |
| REQ-NF-002 | The runner MUST be deterministic for scheduling and serialization: stable candidate ordering, monotonically increasing dispatch ordinals, stable JSON key serialization for digests, and byte-comparable contract verdicts for identical inputs. Provider output remains observed evidence, not a scheduling input. | Epic G7, G11 |
| REQ-NF-003 | All provider dispatches MUST occur inside the I-05 agent-visible fixture and scratch roots. The runner MUST invoke the existing evaluator-disclosure guard immediately before each dispatch and fail before provider spend if evaluator-only material is visible. | Epic G9; ADR-007 |
| REQ-NF-004 | The controller MUST heartbeat before the configured Shark claim TTL expires. Its default cadence MUST be the smaller of 60 seconds and one third of the effective claim TTL, with a minimum of 1 second; a failed heartbeat MUST produce `lease_loss` and stop the scenario. | Epic G11, G16 |
| REQ-NF-005 | The run record MUST be append-safe and resumable for diagnosis: flush each dispatch and stop record before the next dispatch, preserve partial records on interruption, and never report an incomplete JSONL record as a completed run. | Epic G9, G12 |
| REQ-NF-006 | Generic lifecycle and evidence code MUST remain language-neutral. Python, Go, package-manager, test, and lint behavior MUST be reached through the I-04 adapter contract; no F08 generic script may branch on fixture language. | F05 Capability map; Epic G8 |
| REQ-NF-007 | Logs and I-07 artifacts MUST exclude rendered prompts, credentials, provider secrets, and unbounded worker transcripts. They may retain hashes, bounded result evidence, paths, sizes, and access-event metadata. | Epic G9, G14; X-11 |
| REQ-NF-008 | Contract validation MUST run offline against committed artifacts and synthetic fixtures. Provider-backed execution MUST be opt-in through the operator workflow owned by F10; F08's dry-run and contract modes MUST make zero provider calls. | Epic G15; ADR-002 |

### Acceptance criteria

| ID | Testable acceptance criterion |
|---|---|
| AC-001 | `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061` validates every required I-07 field and closed vocabulary against `bench/runs/i07-schema.yaml`; malformed, unsupported, duplicate-ordinal, missing-stop-reason, and publication-eligibility conflicts fail with the field or value named. |
| AC-002 | `tc060_lifecycle_runner_contract_test.sh` proves the real worker adapter preserves exact prompt bytes and bounded semantic results without Shark mutation authority. `tc061_lifecycle_runner_loop_test.sh` exercises the real `run-lifecycle.sh` controller against stubbed public Shark commands. |
| AC-003 | `tc061_lifecycle_runner_loop_test.sh` proves the controller's successful lease/transition path: claim uses the returned entity key; heartbeat uses the returned session; the semantic outcome, `--from-status`, and session reach `status advance`; release uses the same session. Failure/cleanup variants are covered by the registered lifecycle contract suite and are not claimed as a single-test result. |
| AC-004 | Planned controller coverage supplies a `parallel_candidates` response with at least three eligible descendants in non-canonical order. The target I-07 record contains all three exactly once, in canonical-key scheduling order, with the fork response and `resolved_via` preserved. |
| AC-005 | Planned controller coverage reaches each positive ceiling independently and proves the scenario stops at the first exceeded ceiling, retains prior and current partial evidence, emits `resource_limit`, sets `publication_eligible: false`, and does not dispatch a later sibling. |
| AC-006 | `tc063_review_finding_capture_test.sh` proves distinct records for a gate with findings, a gate with zero findings, a collector failure, and an unreached gate. It verifies raw metadata and exact candidate/policy references are retained. |
| AC-007 | A feature fixture with a valid I-06 result completes D01–D05 before lifecycle dispatch; a missing replay entry produces `unresolved_gate` without a provider call. Non-feature fixtures produce explicit non-applicable prelude records. |
| AC-008 | An evaluator-only file planted in either agent-visible root causes a pre-dispatch failure and zero worker invocations. The check reads the existing I-05 isolation guard rather than reimplementing its path rules. |
| AC-009 | The run validator rejects missing usage/model identity, prompt digest mismatch, changed candidate snapshot, missing artifact-consumption evidence, and an unknown stop outcome. It retains each rejection reason in the I-07 record. |
| AC-010 | Repeating dry-run and contract-mode execution twice over the same committed fixtures produces byte-identical I-07 verdicts and zero provider invocations. |
| AC-011 | `make fmt && make lint && make test` passes; no file under `internal/` or `cmd/` changes; no database schema or migration changes; and `bench/scripts/tests/run-all.sh` invokes the new F08 tests. |
| AC-012 | Retained UAT summary artifacts identify the exercised TC-061/TC-065 surfaces and their terminal predicates. Full descendant scheduling and durable Question records remain inspectable in the originating run artifacts; the committed JSON summaries are explicitly not the authoritative run record. |

### Out of scope for this feature

- Reconstructing Shark prompts, workflow routing, status semantics, claims, leases, or Question state.
- Modifying `internal/runner`, `internal/cli`, `internal/services`, or the Shark database. Generic defects found at those seams are triaged under E22, E38, E39, E27-F15, or the owning epic.
- Scoring artifacts, calibrating an LLM judge, confirming review findings, normalizing fingerprints, computing precision/recall, or deciding aggregate eligibility beyond the raw I-07 invalidity flag. Those responsibilities belong to E40-F09.
- Operator spend approval, pilot selection, baseline publication, report layouts, noise bands, and QA-versus-deep-review comparison reports. Those responsibilities belong to E40-F10.
- D06–D14 product design, sprint workflows, PR feedback, CI/merge/cleanup, and epics as scenario roots.
- Adding a generic provider runtime or claiming that replayed request counts represent observed human time.

## Architecture

### Component changes

| Path | Change |
|---|---|
| `bench/runs/i07-schema.yaml` | New single machine-readable owner for I-07 schema version, terminal outcomes, dispatch/gate states, required field inventory, and canonical digest rules. F09 and F10 read this file rather than duplicating vocabularies. |
| `bench/scripts/run-lifecycle.sh` | New host-side controller. Reads I-04, consumes I-06 when applicable, calls keyed Shark commands, invokes the provider-neutral host adapter, schedules all fork candidates, enforces ceilings, and writes I-07. It never calls `shark run`. |
| `bench/scripts/lifecycle-worker-adapter.sh` | New narrow adapter boundary implementing the request/result shape from `skills/shark-rider/context/host-adapter-contract.md`. It owns provider invocation and bounded control-envelope extraction; it does not own Shark state. |
| `bench/scripts/verify-lifecycle-run.sh` | New offline validator and guard for I-07 joins, dispatch ordinals, stop eligibility, candidate/policy references, isolation preflight results, and explicit review-gate states. It invokes existing I-05/I-06 validators instead of re-deriving their rules. |
| `bench/scripts/tests/tc060_lifecycle_runner_contract_test.sh` | New contract and exact-prompt handoff test. |
| `bench/scripts/tests/tc061_lifecycle_runner_loop_test.sh` | New claim, heartbeat, semantic outcome, transition, release, fork, and Question-loop test. |
| `bench/scripts/tests/tc062_lifecycle_runner_limits_test.sh` | New cost, wall-time, generated-task, and named-stop test. |
| `bench/scripts/tests/tc063_review_finding_capture_test.sh` | New review-gate and raw finding lineage test. |
| `bench/scripts/tests/tc064_lifecycle_runner_offline_determinism_test.sh` | New no-provider and byte-identical rerun test. |
| `bench/scripts/testdata/lifecycle/` | New stub Shark/worker/Question command set and complete, forked, stopped, malformed, and finding-bearing fixtures. No evaluator-only material is placed in an agent-visible fixture. |
| `tests/contracts/e40_i07_lifecycle_run_contract_test.go` | New test-only Go contract validator, `package contracts`, `TC-061`; reads committed I-07 fixtures and `i07-schema.yaml` only. |
| `tests/contracts/testdata/e40_i07/{valid,invalid}/` | New static I-07 fixtures for schema, identity, stop, finding, fork, and malformed-record cases. |
| `bench/scripts/tests/run-all.sh` | Modified only to register TC-060 through TC-064. |
| `bench/README.md` | Modified with the I-07 lifecycle runner contract, dispatch command sequence, artifact layout, stop semantics, and test-tier instructions. |

No file under `internal/` or `cmd/` is modified. Existing `bench/scenarios/**`,
`bench/evidence/**`, and `bench/replay/**` remain read-only inputs owned by
F05–F07.

### Data model changes

No Shark schema, migration, or database change. I-07 is a file artifact under
`bench/runs/<run_id>/lifecycle.jsonl`; one JSON object represents one scenario
run and contains nested dispatch, fork, stage, Question, review-gate, limit,
and outcome records. I-05 stage snapshots and I-06 replay results remain their
own artifacts and are joined by run ID, entity key, dispatch ordinal, digest,
and path references.

The I-07 top-level record contains at least:

| Block | Required contents |
|---|---|
| `identity` | schema version, run ID, scenario/version, fixture and adapter identity, Shark binary/content identity references, and roots |
| `entity_graph` | root, `resolved_via`, fork candidates, selected entity keys, entity types, ordinals, and ineligibility reasons |
| `dispatches` | exact keyed response metadata, prompt digest/size, claim session, worker identity, outcome, transition, release, timing, and evidence references |
| `stages` | I-05 snapshot references, stage category, usage/cost, interval ledger reference, candidate reference, and artifact access events |
| `workflow_policy` | gate set/order, reviewer configuration, prompt and review-bundle digests, and fix policy |
| `review_gates` | one of `findings`, `zero_findings`, `collection_failure`, or `not_reached`, plus raw finding references |
| `questions` | Question key, X-13 handoff, authorized response reference, and terminal result |
| `limits` | policy ceilings, observed consumption, and the first ceiling that stopped the scenario, if any |
| `outcome` | named terminal outcome, partial-evidence status, publication eligibility, and reasons |

The validator distinguishes an empty collection from missing evidence. For
example, `findings: []` is a valid `zero_findings` gate record, while an absent
`review_gates` entry is invalid when the gate was reached.

### API and interface contracts

#### Shark keyed lifecycle sequence

For each scheduled concrete entity, the controller performs this sequence:

1. `shark next <key> --json [--prompt-out <path>]`; store the response and verify the prompt digest if a prompt exists.
2. `shark claim <response.entity_key> --by <runner-identity> --json`; store the returned `session_id`.
3. Invoke the host adapter with the unchanged prompt and entity metadata.
4. Heartbeat the returned entity/session during provider work.
5. Parse the bounded worker control envelope and persist the semantic outcome.
6. `shark status advance <entity_key> --outcome <outcome> --session <sid> --from-status <response.status> --agent <agent@provider/model>`.
7. `shark release <entity_key> --session <sid> --outcome <outcome>` in a finally-equivalent cleanup path.

`pause`, `archive`, `error`, fork, lease loss, missing outcome, and Question
responses are recorded as wire events, not converted into `completed`.

#### Host adapter contract

`lifecycle-worker-adapter.sh` receives a JSON request containing the exact
`NextResponse` fields named in `host-adapter-contract.md`, the claim session,
working roots, and policy identity. It returns one bounded JSON result with
`worker_id`, `session_id`, `kind`, `recommended_outcome` when `kind=final`, and
bounded `evidence`. It must run synchronously unless the provider reference
documents a terminal retirement operation; the current installed-provider
references require an awaited foreground invocation.

#### I-07 digest and identity rules

All stored references use SHA-256 over canonical UTF-8 JSON or file bytes as
declared by `i07-schema.yaml`. Candidate identity includes the I-05 base
commit, tree digest, binary diff digest, changed-path digest, dirty/untracked
manifest, and test-suite digest. A branch name, `HEAD`, or equal base commit
alone is not identity. Missing or disagreeing identity is retained as an
invalidity reason and cannot enter F09 aggregates.

### Key technical decisions

**ADR-F08-01 — Host-side keyed loop, not `shark run`.** Lifecycle v2 needs
replay, per-dispatch evidence, deterministic fork scheduling, and Questions.
The existing `shark run` controller remains the Phase 1 path; F08 uses public
keyed CLI contracts and records the outer loop. This follows E40 ADR-006 and
the F08 research decision.

**ADR-F08-02 — Serial deterministic scheduling.** F08 processes every fork
candidate in canonical key order. The Shark response may say parallel execution
is available, but F08 does not introduce concurrency into the benchmark record;
serial scheduling makes lease ownership, cost ceilings, and evidence ordinals
reproducible. F10 may compare operator modes later without changing I-07.

**ADR-F08-03 — Parent-only mutation.** The controller owns claim, heartbeat,
transition, and release. This follows `skills/shark-rider/verbs/run.md`,
`internal/services/claim_service.go`, and the X-11 contract. A worker result is
semantic input to the parent, never authorization to mutate Shark directly.

**ADR-F08-04 — File-backed I-07.** JSONL keeps benchmark state outside Shark's
database and follows ADR-002. A single schema owner and offline validator make
partial records, replay, and cross-feature consumption inspectable.

**ADR-F08-05 — Capture, do not adjudicate.** F08 preserves raw review findings
and exact candidate/policy links. F09 owns confirmation, normalization, and
aggregate eligibility, preventing the runner from treating reviewer prose or a
passing terminal status as truth.

**ADR-F08-06 — Fail closed on missing evidence.** Missing usage, prompt
provenance, candidate identity, isolation proof, stage evidence, or Question
authorization stops or invalidates the run with a named reason. The runner
never substitutes zero, terminal workflow status, or worker self-report.

### Integration with existing code

The runner is a bench consumer of these existing surfaces:

- `internal/cli/commands/next.go:143-205` — `NextResponse` and `parallel_candidates`; use the exact wire fields and `prompt_sha256`/`prompt_bytes`.
- `internal/cli/commands/claim.go` and `internal/services/claim_service.go` — session-scoped claim, heartbeat, release, TTL, and work-session journaling.
- `internal/cli/commands/status_group.go` and `internal/services/transition_types.go` — outcome-routed transitions with `--session` and `--from-status`.
- `skills/shark-rider/verbs/run.md` and `skills/shark-rider/context/host-adapter-contract.md` — parent-owned lifecycle, exact prompt transport, bounded result envelope, and awaited worker policy.
- `internal/cli/commands/notes_add_dispatch.go`, the generic entity `notes` read surface, and `internal/models/validation.go` — typed `review-finding` note metadata; F08 reads and preserves it without changing note storage.
- `internal/services/question_blocker.go`, `internal/services/question_workflow_service.go`, and the E39-F04 contract — X-13 Question blocking, response, and resolution surfaces.
- `bench/scripts/verify-evidence-roots.sh`, `bench/scripts/verify-stage-evidence.sh`, `bench/scripts/replay-stage-evidence.sh`, and `bench/scripts/verify-replay-result.sh` — existing I-05/I-06 guards and joins, invoked rather than copied.

The implementation must add contract tests with stubs around these boundaries,
then one retained UAT invocation against the real scratch project and Shark
binary. Any required change to a production surface is a separate owning-epic
item, not an F08 implementation shortcut.

## Cross-feature interactions

### Consumes

- **I-04 — Lifecycle scenario corpus and adapter contract**; producer E40-F05; F08 reads scenario identity/version, entity family, lifecycle stage matrix, fixture and adapter, replay/evaluator references, and resource policy. Shape source: `../architecture.md#lifecycle-scenario-package-contract`. Contract test pointer: `tests/contracts/e40_i04_scenario_contract_test.go#TC-030`.
- **I-05 — Stage evidence and evaluator isolation**; producer E40-F06; F08 writes observed stage snapshots, time intervals, candidate/artifact references, and access events using the producer's three-root and vocabulary rules. Shape source: `../architecture.md#stage-evidence-and-isolation-contract`. Contract test pointer: `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042`.
- **I-06 — Product-design replay result**; producer E40-F07; F08 consumes the authorized D01–D05 result and propagates its terminal mapping when replay blocks the lifecycle. Shape source: `../architecture.md#product-design-replay-contract`. Contract test pointer: `tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052`.

### Produces

- **I-07 — Lifecycle run record**; consumer features E40-F09 and E40-F10; F08 writes the entity graph, keyed dispatches, fork scheduling, leases, transitions, stage/evidence references, policy identity, Questions, raw review-gate findings, resource limits, stop outcome, and aggregate-eligibility flag. Shape source: `../architecture.md#lifecycle-run-record-contract`. Contract test pointer: `tests/contracts/e40_i07_lifecycle_run_contract_test.go#TC-061`.

The I-04, I-05, I-06, and I-07 rows use the map-assigned `contract-only` gate
mode until each named consumer proves live production-path use. F08 is the
activation owner for its I-04/I-05/I-06 slices and the producer review basis
for I-07; it does not alter counterpart status, closure keys, or review basis.

## Cross-epic integrations

### Consumes

- **X-11 — Canonical host-side keyed loop**; producer E38 (E38-F07 Rider Execution and Escalation Loop; E38-F09 Provider-Neutral Coordination and Live Resume); purpose: reuse dispatch response, claim, unchanged prompt, heartbeat, semantic outcome, transition, release, prompt provenance, and bounded resume. Contract / shape source: `E38-F07 and E38-F09 feature contracts; Shark Rider run procedure; E40 architecture "Lifecycle v2 controller boundary"`. UX / CX handoff: “The benchmark records and schedules the loop but does not create a second workflow engine, claim store, or prompt assembler.” Test coverage: E40 UAT-11 and UAT-12, plus `tc060_lifecycle_runner_contract_test.sh` and `tc061_lifecycle_runner_loop_test.sh`.
- **X-13 — Durable Question lifecycle**; producer E39-F04; purpose: use the durable Question lifecycle for unresolved decisions and replay-authorized responses during a scored lifecycle. Contract / shape source: `E39 architecture and E39-F04 consumer handoff; E40 architecture "Lifecycle run record contract"`. UX / CX handoff: “Missing authorized input remains a visible `unresolved_gate`; the benchmark never invents a decision or hides the blocking Question in transcript-only state.” Test coverage: E40 UAT-12, plus AC-007 in `tc061_lifecycle_runner_loop_test.sh`.

F08 neither produces nor validates X-07, X-08, X-09, X-10, or X-12. X-10's
product-design action is consumed by F07; F08 consumes the resulting I-06
artifact rather than crossing that boundary directly.

## Durable unresolved decisions

No material unresolved decision remains for this specification. The host-loop
boundary, serial fork policy, file-backed I-07 store, parent-owned mutation,
and fail-closed evidence policy are fixed by the parent architecture, the
interaction map, the X-11/X-13 contracts, and the validated F08 research
decisions. No Q### record is required; these are settled architecture choices,
not open scope, contract, sequencing, or risk-acceptance questions.

## Traceability summary

| Requirement group | Contract / proof |
|---|---|
| REQ-F-001–008 | I-04/I-05, X-11, AC-001–005, UAT-11 |
| REQ-F-009, REQ-F-014 | I-06, X-13, AC-007/012, UAT-12 |
| REQ-F-010–013 | I-05/I-07, AC-006/008/009, UAT-11/12 |
| REQ-F-015 and REQ-NF-001–008 | I-07 schema/validator, AC-005/009–011, E40 ADR-002/006/007/008/009 |

RECOMMENDED OUTCOME: pass
