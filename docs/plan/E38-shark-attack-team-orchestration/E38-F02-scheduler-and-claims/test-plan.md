# Test Plan: E38-F02 - Scheduler and Claims

**Created:** 2026-07-13  
**Feature PRD:** `docs/plan/E38-shark-attack-team-orchestration/E38-F02-scheduler-and-claims/feature.md`  
**Feature Spec:** `docs/plan/E38-shark-attack-team-orchestration/E38-F02-scheduler-and-claims/spec.md`  
**Parent UAT source:** `docs/plan/E38-shark-attack-team-orchestration/epic.md` §6 (no separate E38 `uat-plan.md` exists)  
**Status:** READY

## Spec Drift Analysis

### Drift findings

1. **RESOLVED — cross-epic ownership:** F02 reuses E22 through the canonical dispatch seam and declares no X-## integration.
2. **RESOLVED — I-04 pointer:** `TC-004` is reserved for the shared F04 role/communication contract; F02's claim race is `TC-004A`.
3. **WARNING — acceptance wording:** AC-F02-005 combines four dispatcher outcomes and cancellation; the plan partitions these explicitly. AC-F02-007 combines enabled and zero TTL; the plan uses a decision table. No behavior is assumed beyond the spec.

### Traceability matrix

| Feature requirement | Acceptance criterion / shared contract | Covered? | Notes |
|---|---|---:|---|
| REQ-F-001 immutable confirmed snapshot and drift refusal | AC-F02-001, TC-001 | Yes | TC-001 asserts no re-plan, membership mutation, or dispatch after hash drift. |
| REQ-F-002 dependency-ready waves and bounded dependency reasons | AC-F02-002/003, TC-002/003 | Yes | Includes failed, blocked, paused, cancelled, skipped, and unsatisfied external prerequisites. |
| REQ-F-003 mode, concurrency, resource safety | AC-F02-001/011, TC-001/011 | Yes | Parallel limit, sequential limit, unknown/overlap fallback. |
| REQ-F-004 claim immediately before dispatch | AC-F02-004, TC-004A | Yes | Force=false, single winner, loser is non-destructive. |
| REQ-F-005 canonical resolver and non-dispatch outcomes | AC-F02-008, TC-008 | Yes | No recursive CLI or YAML prompt reconstruction. |
| REQ-F-006 root/child heartbeats and TTL | AC-F02-007, TC-007 | Yes | Includes explicit zero TTL and failed heartbeat. |
| REQ-F-007 exact-session cleanup | AC-F02-006, TC-006 | Yes | Dispatch, cancellation, panic, and persistence failure paths. |
| REQ-F-008 durable per-item mapping and failure containment | AC-F02-005, TC-005 | Yes | Process/semantic result, bounded evidence, unrelated work continues. |
| REQ-F-009 CAS, idempotency, resume/retry | AC-F02-009, TC-009 | Yes | Completed items are not redispatched; explicit attempt required. |
| REQ-F-010 complete operator result, no root transition | AC-F02-012, TC-012 | Yes | Shared I-02/I-05 shape is asserted by the existing contract pointer. |
| REQ-F-011 structured safe execution events | AC-F02-010, TC-010 | Yes | Sensitive-output rejection and event-field allow-list. |
| REQ-NFR-001..006 | TC-001, 005, 007, 009, 010, 011, 012 | Yes | Determinism, transaction boundaries, DI, claim safety, evidence bounds, resource safety. |

## Acceptance Criteria Review

### Ambiguity findings

- AC-F02-004's claim conflict is covered by TC-004A; TC-004 remains reserved for I-04.
- AC-F02-005 needs the explicit partition used in TC-005: success, non-zero exit, provider-not-found, and cancellation. The expected state for cancellation is recorded as `cancelled` or `paused` according to the configured contract; BA/architecture should choose one canonical value before implementation.
- AC-F02-011 permits either sequential fallback or item exclusion. TC-011 accepts both only when the chosen result contains an explicit durable reason and never claims parallel execution.
- AC-F02-012 says “complete” without a field-level schema in the AC; TC-012 uses the architecture §4.6/operator contract and the existing shared result test as the required field list.

### Missing coverage

No AC is uncovered. No X-## integration is declared for F02; E22 reuse is covered by the canonical dispatch seam.

## ISTQB Technique Application (per AC)

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-F02-001 | Boundary Value Analysis, Decision Table, State Transition | TC-001 | Limits 1, 2, and 3 independent children and slot release/order. |
| AC-F02-002 | State Transition, Decision Table | TC-002 | Readiness changes only after both prerequisite success outcomes. |
| AC-F02-003 | Equivalence Partitioning, Decision Table | TC-003 | Failed/blocked/paused/cancelled/skipped/unsatisfied versus unrelated ready items. |
| AC-F02-004 | Contract Surface Enumeration, State Transition | TC-004A | Claim race, force flag, session ownership, and loser cleanup. |
| AC-F02-005 | Equivalence Partitioning, State Transition | TC-005 | Four dispatcher result/error classes and independent-worker continuation. |
| AC-F02-006 | State Transition, Attack-class Enumeration | TC-006 | Every exit path and replacement-claim safety. |
| AC-F02-007 | Boundary Value Analysis, Decision Table | TC-007 | TTL positive, just-expired, live, and explicit zero/no-expiry branches. |
| AC-F02-008 | Equivalence Partitioning, Contract Surface Enumeration | TC-008 | Pause, terminal, unresolved workflow, and unresolved placeholder results. |
| AC-F02-009 | State Transition, Contract Surface Enumeration | TC-009 | Resume, completed terminal, stale running, and explicit retry attempts. |
| AC-F02-010 | Attack-class Enumeration, Contract Surface Enumeration | TC-010 | Prompt/credential/transcript/stdout leakage classes and event allow-list. |
| AC-F02-011 | Decision Table, Equivalence Partitioning | TC-011 | Safe, unknown, overlapping, and unavailable capability classes. |
| AC-F02-012 | Contract Surface Enumeration, Equivalence Partitioning | TC-012 | Mixed outcome partitions and every required result field. |

## ISO 25010 Coverage Matrix

`N/A` is deliberate: the characteristic has no additional evidence beyond the functional contract for this AC. All other cells cite a test.

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-F02-001 | ✅ TC-001 | ✅ TC-001 | N/A: internal scheduler contract | ✅ TC-001 | ✅ TC-001 | N/A: no secret boundary | ✅ TC-001 | ✅ TC-011 |
| AC-F02-002 | ✅ TC-002 | N/A: no latency SLO | ✅ TC-002 | ✅ TC-002 | ✅ TC-002 | N/A: graph data only | ✅ TC-002 | N/A: injected graph |
| AC-F02-003 | ✅ TC-003 | N/A: no throughput SLO | ✅ TC-003 | ✅ TC-003 | ✅ TC-003 | N/A: dependency policy | ✅ TC-003 | N/A: injected graph |
| AC-F02-004 | ✅ TC-004A | N/A: race correctness only | ✅ TC-004A | ✅ TC-004A | ✅ TC-004A | ✅ TC-004A | ✅ TC-004A | N/A: claim repository contract |
| AC-F02-005 | ✅ TC-005 | N/A: limit covered by TC-001 | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | N/A: evidence covered by TC-010 | ✅ TC-005 | N/A: injected dispatcher |
| AC-F02-006 | ✅ TC-006 | N/A: cleanup latency unspecified | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 | N/A: claim contract |
| AC-F02-007 | ✅ TC-007 | ✅ TC-007 | ✅ TC-007 | ✅ TC-007 | ✅ TC-007 | N/A: lease safety only | ✅ TC-007 | ✅ TC-007 |
| AC-F02-008 | ✅ TC-008 | N/A: no latency SLO | ✅ TC-008 | ✅ TC-008 | ✅ TC-008 | N/A: no sensitive input | ✅ TC-008 | N/A: workflow seam |
| AC-F02-009 | ✅ TC-009 | N/A: no resume SLO | ✅ TC-009 | ✅ TC-009 | ✅ TC-009 | N/A: no auth boundary | ✅ TC-009 | N/A: ledger contract |
| AC-F02-010 | ✅ TC-010 | N/A: no event SLO | ✅ TC-010 | ✅ TC-010 | ✅ TC-010 | ✅ TC-010 | ✅ TC-010 | N/A: slog/trace seam |
| AC-F02-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 |
| AC-F02-012 | ✅ TC-012 | N/A: result shape only | ✅ TC-001 | ✅ TC-012 | ✅ TC-012 | ✅ TC-010 | ✅ TC-012 | N/A: JSON contract |

### Coverage gaps

- AC-F02-004 × Compatibility is covered by the non-destructive claim seam in TC-004A.
- AC-F02-011 × Maintainability is covered by the decision-table requirement in TC-011, but the implementation must keep the policy injected rather than hard-coded.

## Observability Design (per behavior)

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Run/wave starts and completes | `team_scheduler_runs_total{status,mode}` | `team_scheduler_run_completed` with run/root/status/counts | `team.scheduler.run` | N/A | TC-001/012 |
| Dependency gate blocks/skips | `team_scheduler_dependency_blocks_total{reason}` | `team_scheduler_item_blocked` with child/reason | `team.scheduler.dependency_gate` | Baseline/spike review | TC-002/003 |
| Claim conflict or release failure | `team_scheduler_claim_conflicts_total`, `team_scheduler_release_failures_total` | `team_scheduler_claim_conflict` / `team_scheduler_release_failed` with session IDs | `team.scheduler.claim` | Any release failure requires operator review | TC-004A/006 |
| Dispatch outcome | `team_scheduler_dispatch_total{provider,outcome}` | `team_scheduler_item_completed` with exit/semantic outcome | `team.scheduler.dispatch` | Provider failure baseline | TC-005/008 |
| Heartbeat/reclaim behavior | `team_scheduler_heartbeat_failures_total` | `team_scheduler_heartbeat_failed` with entity/session | `team.scheduler.heartbeat` | Repeated failures | TC-007 |
| Resume/idempotency | `team_scheduler_resume_total{result}` | `team_scheduler_resume_reconciled` | `team.scheduler.resume` | Duplicate-dispatch alert | TC-009 |
| Sensitive evidence rejection | `team_scheduler_sensitive_evidence_rejected_total{cause}` | `team_scheduler_evidence_rejected` without submitted secret/prompt | `team.scheduler.evidence.validate` | Any credential leak is critical | TC-010 |
| Capability fallback/exclusion | `team_scheduler_capability_fallback_total{reason}` | `team_scheduler_degraded_mode` with mode/reason | `team.scheduler.capability` | Spike review | TC-011 |

**Implementation hook:** instrumentation above is part of the implementation contract. Logs/events may include run ID, root/child key, wave, provider, duration, claim/session correlation, and outcome, but never rendered prompts, credentials, unrestricted output, or transcripts.

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-01 | E38-F01 | E38-F02, E38-F03, E38-F05 | `architecture.md` §4.2 Team-run domain contract | `tests/contracts/e38_interactions_test.go#TC-001` / `TestTC001_I01TeamRunResultContract` | TC-001 |
| I-02 | E38-F02 | E38-F03, E38-F05 | `architecture.md` §4.4 Aggregate outcome contract | `tests/contracts/e38_interactions_test.go#TC-001` (shared result serialization) | TC-012 references the same shared assertion; no twin test |
| I-04 | E38-F04 | E38-F02, E38-F03, E38-F05 | `architecture.md` §4.5 Council communication contract | `tests/contracts/e38_interactions_test.go#TC-004` | TC-004 (shared role/communication contract) |
| I-05 | E38-F02 | E38-F05 | `architecture.md` §4.6 Operator contract | `tests/contracts/e38_interactions_test.go#TC-001` | TC-012 references the same shared JSON shape; no twin test |

### Cross-epic integrations (X-##)

F02 declares no X-## integration. E22 interoperability is through the existing canonical dispatch, claim, process-result, worktree, and prompt seams; no new cross-epic contract row is required.

## Caller-Path Contracts (per test case)

| TC | Production entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `Scheduler.Start(ctx, runID=41, rootSessionID="root-session-001")` with persisted confirmed `TeamRun`/items | F01 `TeamLedger` read/CAS seam and injected clock/event sink | Do not call a private wave helper; do not mock scheduler, claim policy, or result conversion | A scheduler that replans or changes membership dispatches a different set/hash. |
| TC-002 | `Scheduler.Start(ctx, 41, "root-session-001")` | `TeamLedger.ListItems` fixture | Do not mock readiness result or bypass persisted dependency keys | C starts before both A and B succeed. |
| TC-003 | `Scheduler.Start(ctx, 41, "root-session-001")` | Ledger read and injected dependency-state fixture | Do not mock dependency policy or mark all prerequisites successful | A failed/paused prerequisite falsely completes C. |
| TC-004 | `tests/contracts/e38_interactions_test.go#TC-004` shared F04 role/communication contract entrypoint as defined by F04 | F04 roster/inbox artifact boundary | Do not mock away the serialized role, recipient, scope, or handoff fields | Scheduler treats roster YAML as workflow authority or drops scoped communication metadata. |
| TC-004A | `Scheduler.Start(ctx, 41, "root-session-001")` | `TeamClaims` interface backed by claim-service-shaped fake | Do not mock `Scheduler`, `ClaimService.Claim`, force flag, or ledger CAS | Both coordinators dispatch, or loser force-steals the live claim. |
| TC-005 | `Scheduler.Start(ctx, 41, "root-session-001")` | Injected `runner.AgentDispatcher` at `Dispatch(ctx, runner.DispatchInput)` | Do not call a convenience result mapper or mock the scheduler loop | One failed worker prevents unrelated ready work or loses process outcome. |
| TC-006 | `Scheduler.Start(ctx, 41, "root-session-001")` | Claim/repository failure seam after real scheduler cleanup | Do not mock cleanup/defer logic or use force release | Cleanup releases a replacement claim or leaves the original session held. |
| TC-007 | `Scheduler.Start(ctx, 41, "root-session-001")` with injected clock and TTL config | `TeamClaims.Heartbeat`/clock seam | Do not mock heartbeat scheduling or TTL interpretation | Heartbeat uses root ID for child, or zero TTL is reclaimed. |
| TC-008 | `Scheduler.Start(ctx, 41, "root-session-001")` | `DispatchStepResolver.Resolve(ctx, entityType, key)` | Do not call CLI `next`, reconstruct YAML prompts, or mock resolver result at scheduler boundary | Pause/terminal work is claimed or an unresolved prompt is dispatched. |
| TC-009 | `Scheduler.Resume(ctx, 41, "root-session-001")` | Ledger CAS/repository with persisted rows | Do not reset in-memory state or mock completed-item lookup | Resume redispatches a completed item or overwrites another attempt. |
| TC-010 | `Scheduler.Start(ctx, 41, "root-session-001")` with `DispatchResult` containing concrete sensitive markers | Event sink and evidence validator at their lowest seams | Do not sanitize only in the test or assert only a private sanitizer | Prompt/credential/transcript reaches ledger, JSON, log, or event. |
| TC-011 | `Scheduler.Start(ctx, 41, "root-session-001")` with injected `ResourcePolicy` capability facts | `ResourcePolicy`/`WorktreeCreator` capability seam | Do not mock selected mode or suppress fallback diagnostics | Unknown overlap runs in unsafe parallel mode or claims parallel execution. |
| TC-012 | `Scheduler.Start(ctx, 41, "root-session-001")` returning `TeamRunResult` | `TeamLedger` read/result seam and existing `TeamRunResult` conversion | Do not hand-build a DTO or test only JSON formatting | Result omits planned item, session, count, timing, evidence, reason, or next action. |

**Implementation hook:** red-phase tests must invoke the listed production scheduler boundary with the stated argument shape. Tests that only exercise private helpers, convenience signatures, recursive CLI calls, or pre-sanitized fixtures are rejected.

## Acceptance Test Cases

### TC-001: Bounded parallelism and confirmed snapshot

**AC:** AC-F02-001; also shared I-01 result-shape evidence. **Technique:** BVA + Decision Table + State Transition. **ISO:** Functional Suitability, Performance Efficiency, Reliability, Usability, Maintainability.

**Setup/input:** Persist run 41 with three independent task items `T-E38-F02-001..003`, execution mode `parallel`, limit `2`, safe resource facts, and plan hash `hash-a`. Use dispatch gates so the first two remain active until released.

**Expected:** At most two dispatches are active; item 3 starts only after one slot releases; all three have durable claim/running/result transitions; returned mode is `parallel`, limit `2`, and plan hash remains `hash-a`.

**Edge cases:** limit 1 uses one active worker; limit 3 permits all three; shuffled persisted row order still follows wave/order/priority/key. A changed hash returns refresh-required before any child claim.

**Negative:** Never dispatch a fourth item, exceed two active workers, re-plan membership, or mutate the root workflow.

**Observability:** Assert run/wave metric, completion log, and `team.scheduler.run` span with IDs/counts and no prompt.

### TC-002: Dependency wave gating

**AC:** AC-F02-002. **Technique:** State Transition + Decision Table. **ISO:** Functional Suitability, Compatibility, Usability, Reliability, Maintainability.

**Setup/input:** Persist A and B in wave 0 and C in wave 1 with `dependency_keys=[A,B]`. Make A/B return configured success; record dispatch start times.

**Expected:** A/B may dispatch subject to limit; C is neither claimed nor dispatched until both durable item outcomes are acceptable success. C then receives its own resolved dispatch input.

**Edge cases:** A completes before B; limit 1; dependency order reversed in storage; a prerequisite has a terminal success recorded before scheduler start.

**Negative:** C must not start after only A succeeds, and must not be treated as successful because B was unrelated or absent.

### TC-003: Dependency-failure containment

**AC:** AC-F02-003. **Technique:** Equivalence Partitioning + Decision Table. **ISO:** Functional Suitability, Usability, Reliability, Maintainability.

**Setup/input:** A fails, B succeeds, C depends on A, and D is unrelated ready work. Repeat with A blocked, paused, cancelled, skipped, and with external prerequisite `Satisfied=false`.

**Expected:** C is `blocked` or `skipped` with the exact bounded dependency reason; C has no claim/dispatch; D completes independently; A/B/D evidence remains durable.

**Edge cases:** multiple failed prerequisites; empty dependency list; external prerequisite satisfied; unknown prerequisite is a bounded invalid-plan diagnostic.

**Negative:** C must never be marked successful, and one worker failure must not cancel unrelated D.

### TC-004: I-04 role and communication contract

**Interaction:** I-04; required pointer `tests/contracts/e38_interactions_test.go#TC-004`. **Technique:** Contract Surface Enumeration. **ISO:** Functional Suitability, Compatibility, Usability, Maintainability.

**Setup/input:** Use the F04-defined shared contract fixture containing role `developer`, recipient `chair`, root `E38-F02`, child `T-E38-F02-001`, urgency, handoff/decision fields, and inbox artifact reference.

**Expected:** F02 consumes role/communication metadata as execution context; serialized fields preserve scope and recipient; workflow routing and mutation authority remain sourced from workflow/claims, not roster YAML.

**Edge cases:** missing optional handoff; acknowledged inbox; role with no provider preference; unknown role produces bounded diagnostic.

**Negative:** Do not silently drop root/child scope, route by roster YAML, or grant worker claim/transition authority.

### TC-004A: Non-destructive claim conflict

**AC:** AC-F02-004. **Technique:** Contract Surface Enumeration + State Transition. **ISO:** Functional Suitability, Reliability, Security, Maintainability.

**Setup/input:** Two scheduler instances race on child `T-E38-F02-001`; claim repository allows exactly one `Force=false` winner and returns session `claim-001`.

**Expected:** One claim succeeds and dispatches once. The loser records claim-conflict diagnostic with child key/current availability, does not dispatch, does not force-steal, and does not overwrite `claim-001`.

**Edge cases:** conflict after stale reclaim; loser cancellation; unrelated child remains runnable.

**Negative:** No `Force=true` claim, duplicate dispatch, false completion, or replacement-session release.

### TC-005: Dispatcher outcome mapping and failure containment

**AC:** AC-F02-005. **Technique:** Equivalence Partitioning + State Transition. **ISO:** Functional Suitability, Usability, Reliability, Maintainability.

**Setup/input:** Three independent items return respectively success/exit 0, non-zero exit 7 with bounded stderr, and provider-not-found; a fourth run uses context cancellation.

**Expected:** Each item records semantic/process outcome, exit code/error class, bounded evidence, timing, worker session, and exact claim session release. Unrelated items continue. Cancellation records `cancelled` state and next action.

**Edge cases:** empty output; very long stdout/stderr; dispatcher returns result plus error; context cancelled between claim and dispatch.

**Negative:** A failed provider must not be reported as success, and raw unrestricted output must not be persisted.

### TC-006: Release on every exit path

**AC:** AC-F02-006. **Technique:** State Transition + Attack-class Enumeration. **ISO:** Functional Suitability, Reliability, Security, Maintainability.

**Setup/input:** Inject failures before dispatch, during dispatch, during cancellation, during panic recovery, and during result persistence. Claim child with `claim-001`; create a replacement claim `claim-002` after simulated staleness.

**Expected:** Cleanup calls session-scoped release with `claim-001` exactly once; release/persistence failures are durable diagnostics; `claim-002` is never released or overwritten.

**Edge cases:** release returns false; release returns error; panic recovery itself records outcome; zero-length evidence.

**Negative:** Never call unscoped release without force, force-release a replacement, or hide a cleanup failure.

### TC-007: Root and child heartbeat/TTL behavior

**AC:** AC-F02-007. **Technique:** BVA + Decision Table. **ISO:** Functional Suitability, Performance Efficiency, Compatibility, Usability, Reliability, Maintainability, Portability.

**Setup/input:** Root session `root-001`, child session `claim-001`, heartbeat interval 10s. Exercise TTL 5m at t=4m59s, t=5m, t=5m01s and TTL 0; inject one heartbeat error.

**Expected:** Root heartbeat uses `root-001`; child heartbeat uses `claim-001`; TTL 0 does not reclaim; failed heartbeat creates diagnostic/metric and cannot imply completion.

**Edge cases:** heartbeat at exact boundary; child finishes during heartbeat; root cancellation stops child heartbeat loop.

**Negative:** Never swap session IDs, reclaim zero-TTL claims, or let a worker perform workflow/claim ownership operations.

### TC-008: Non-dispatchable canonical steps

**AC:** AC-F02-008. **Technique:** Equivalence Partitioning + Contract Surface Enumeration. **ISO:** Functional Suitability, Compatibility, Usability, Reliability, Maintainability.

**Setup/input:** Resolver returns pause/human gate, terminal step, unresolved workflow, and unresolved placeholder for four planned items.

**Expected:** No item is claimed or dispatched. Each item receives its exact bounded pause/skip reason; result includes next action and preserves completed siblings.

**Edge cases:** resolver returns nil step/error; terminal item already terminal; placeholder appears only after dispatch-time re-resolution.

**Negative:** Never recursively call CLI, reconstruct prompt from YAML, auto-approve, or dispatch unresolved text.

### TC-009: Resume idempotency and explicit retry

**AC:** AC-F02-009. **Technique:** State Transition + Contract Surface Enumeration. **ISO:** Functional Suitability, Compatibility, Reliability, Maintainability.

**Setup/input:** Run 41 has A/B completed attempt 1, C running with stale claim, D planned. Interrupt after persistence, then call `Resume(ctx,41,"root-001")`; separately request explicit retry for C.

**Expected:** A/B are not dispatched; C is reconciled before retry; D runs once; prior evidence remains; explicit retry increments attempt and cannot overwrite terminal result from attempt 1.

**Edge cases:** resume twice; no stale claims; completed item has release failure diagnostic; conflicting terminal result.

**Negative:** No duplicate completion, note, claim, or dispatch for A/B, and no implicit retry of C.

### TC-010: Sensitive evidence and event boundary

**AC:** AC-F02-010. **Technique:** Attack-class Enumeration + Contract Surface Enumeration. **ISO:** Functional Suitability, Reliability, Security, Maintainability.

**Setup/input:** Dispatcher output contains `SYSTEM PROMPT`, `Authorization: Bearer secret-123`, credential-like JSON, 10MB stdout, transcript text, and safe artifact reference `docs/result.md`.

**Expected:** Unsafe evidence is rejected or reduced to bounded safe summary; result/events/logs contain only allowed IDs, wave, provider, duration, sessions, outcome, and safe artifact reference. Assert no sensitive marker in serialized result or captured events.

**Edge cases:** one marker at evidence boundary; Unicode; empty output; safe short summary.

**Negative:** Never persist rendered/system prompt, credential, unrestricted stdout/stderr, or transcript.

### TC-011: Conservative resource fallback

**AC:** AC-F02-011. **Technique:** Decision Table + Equivalence Partitioning. **ISO:** Functional Suitability, Performance Efficiency, Compatibility, Usability, Reliability, Security, Maintainability, Portability.

**Setup/input:** Resource policy reports safe distinct ownership, unknown ownership, overlapping ownership, and unavailable worktree capability for three items with requested parallel limit 2.

**Expected:** Safe facts allow bounded parallelism. Unknown/overlap selects documented sequential fallback or excludes unsafe item with durable reason. Unavailable capability stops before mutation with actionable error. Actual result never claims parallel when sequential occurred.

**Edge cases:** one item only; explicit override; worktree exists but ownership remains unknown.

**Negative:** Distinct entity keys alone must not prove resource safety.

### TC-012: Complete `TeamRunResult` and I-02/I-05 shape

**AC:** AC-F02-012; shared pointers `I-02` and `I-05`. **Technique:** Contract Surface Enumeration + Equivalence Partitioning. **ISO:** Functional Suitability, Compatibility, Usability, Reliability, Maintainability.

**Setup/input:** Finish a run with items in completed, failed, blocked, skipped, paused, and cancelled states; mode sequential, limit 1; include claim/worker sessions, timing, bounded evidence, artifact refs, counts, and next action.

**Expected:** Result contains root identity, run status, actual mode/limit, plan hash, every planned item exactly once, per-item status/outcome/reason, claim/worker sessions, timing, evidence refs, counts, and next action. F02 does not advance the root workflow. The existing shared assertion at `tests/contracts/e38_interactions_test.go#TC-001` remains the single JSON shape test.

**Edge cases:** zero planned items; all paused; nil optional aggregate outcome; partial result after cancellation.

**Negative:** No omitted planned item, invented success, root transition, prompt, secret, or transcript.

## Test Infrastructure

### Existing patterns to follow

- Scheduler tests: new `internal/team/scheduler_test.go`, mock-only coordinator tests with constructor-injected interfaces, following `internal/team/plan_test.go` and `internal/team/ledger_service_test.go`.
- Real database/repository coverage: `internal/repository/teamrun/repository_test.go`, `internal/services/claim_service_test.go`, and migration tests use real SQLite fixtures; do not mock SQL or transactions.
- Dispatcher contract: `internal/runner/dispatcher.go`, `internal/runner/controller_test.go`, and `internal/runner/integration_test/env.go` provide production-shaped `runner.DispatchInput`/`DispatchResult` fakes.
- Canonical resolution: `internal/dispatch/step.go` and `internal/cli/commands/next.go`; tests must drive `Resolve(ctx, entityType, key)` and must not call a convenient prompt helper.
- Shared contract: `tests/contracts/e38_interactions_test.go#TestTC001_I01TeamRunResultContract` is the existing shared result-shape test.
- Claims: `internal/services/claim_service_test.go` covers TTL, force semantics, session-scoped release, heartbeat, reclaim, and work-session behavior.

### New helpers needed

- A deterministic scheduler fixture builder for confirmed `TeamRun`/`TeamRunItem` snapshots with waves, dependencies, attempts, sessions, and bounded evidence.
- A blocking dispatcher fake that records active count, input metadata, release order, and cancellation.
- A claim fake that enforces unique `(entity_type, entity_key)`, records `Force`, sessions, heartbeat IDs, and release IDs.
- An event/log/trace sink fake with assertions for allow-listed fields and forbidden sensitive markers.
- A resource-policy fake for safe, unknown, overlap, and unavailable capability partitions.
- Real SQLite fixtures only for F01 ledger CAS and claim repository integration; worker execution remains outside transactions.

## Codex Test-Plan Red-Team

**Verdict:** FAILED — Codex test-plan review unavailable
**Issues raised:** unavailable
**Issues addressed before dev:** 0
**Issues deferred:** all independent red-team findings; manual review and the explicit drift blockers above remain authoritative

The prescribed command was not supplied as an input parameter. Two local attempts were made with a long-running `codex exec` review prompt; both failed before producing a review because the local client could not authenticate to the OpenAI Responses endpoint.

```text
ERROR: unexpected status 401 Unauthorized: Missing bearer or basic authentication in header, url: https://api.openai.com/v1/responses
```

The first attempt also reported `failed to initialize in-process app-server client: Read-only file system (os error 30)`. The second attempt used an ephemeral temporary home and reached the endpoint, then failed with the 401 above. No Codex verdict is claimed.

## Recommendations

- [ ] Ready for development
- [x] Resolved — removed stale X-01 claim; E22 reuse remains through the canonical dispatch seam.
- [x] Resolved — reserved TC-004 for I-04, TC-004A for claim conflict, and selected `cancelled` for cancellation.
