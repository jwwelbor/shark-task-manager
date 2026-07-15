# Test Plan: E38-F01 - Team Plan and Durable Ledger

**Created:** 2026-07-13
**Feature PRD:** docs/plan/E38-shark-attack-team-orchestration/E38-F01-team-plan-and-durable-ledger/feature.md
**Task/Feature Spec:** docs/plan/E38-shark-attack-team-orchestration/E38-F01-team-plan-and-durable-ledger/spec.md
**Parent UAT Plan:** docs/plan/E38-shark-attack-team-orchestration/uat-plan.md
**Status:** READY_FOR_DEVELOPMENT_WITH_CODEX_REVIEW_DEFERRED

## Scope and test policy

This is a feature-level shift-left plan. No child task specifications exist in
the feature directory, so the implementation paths and test pointers in spec.md
are treated as the feature's task decomposition for this planning gate.

The scope is the read-only planner, normalized durable ledger, shared
dispatch-step extraction, and additive schema migration. Scheduler claims and
dispatch, aggregate routing and resume orchestration, operator formatting, and
the shark-attack skill remain owned by later E38 features.

Testing follows CLAUDE.md and .claude/rules/testing/architecture.md:

- Planner, adapter, ledger-service, and model tests use injected mocks and no
  real database.
- internal/repository/teamrun and internal/db tests use the real repository test
  database, clean up rows, and verify actual SQLite constraints and migrations.
- CLI tests use mocked services/repositories but drive the real Cobra command.
- Each test case below has a Caller-Path Contract. Lower-level white-box tests
  are supplemental only.

## Spec drift analysis

### Drift findings

| Finding | Severity | Result |
|---|---|---|
| feature.md describes a read-only dependency-aware plan and durable run/item ledger; spec.md preserves that scope and assigns execution to later features. | None | No drift. |
| feature.md produces I-01 for F02, F03, and F05; spec.md declares the same consumers and exact shared pointer. | None | No drift. |
| feature.md assigns no X-## ownership; the parent map assigns X-01 through X-05 to later features. | None | No drift and no X test is invented. |
| AC-F01-005 fallback/error selection | Resolved | Sequential mode is selected when one ordinary worker can run and all work can be serialized; typed capability error is returned before mutation when no required adapter can run. |
| AC-F01-007 idempotent-success/conflict behavior | Resolved | Same root/hash returns existing run; different hash returns `ErrPlanDrift`; identical terminal result returns existing result; conflicting terminal result returns `ErrConflictingTerminalResult`. |

### Traceability matrix

| Feature requirement | Acceptance criterion | Covered? | Notes |
|---|---|---|---|
| REQ-F-001 complete direct-child snapshot | AC-F01-001 | Yes | TC-001 covers all child partitions and no mutation. |
| REQ-F-002 graph, waves, external prerequisites | AC-F01-002, AC-F01-003 | Yes | TC-002 and TC-003. |
| REQ-F-003 canonical metadata, no prompt | AC-F01-001, AC-F01-010 | Yes | TC-001 and TC-010. |
| REQ-F-004 actual mode and positive limit | AC-F01-004, AC-F01-005 | Partial | TC-004/005; fallback/error rule remains ambiguous. |
| REQ-F-005 canonical hash and drift | AC-F01-001, AC-F01-008 | Yes | TC-001 and TC-008. |
| REQ-F-006 read-only preview and atomic persistence | AC-F01-001, AC-F01-006 | Yes | TC-001 and TC-006. |
| REQ-F-007 membership, lifecycle, idempotency, retry | AC-F01-006, AC-F01-007 | Partial | TC-006/007; duplicate outcome needs clarification. |
| REQ-F-008 shared Team-run shape | AC-F01-001 and I-01 | Yes | TC-001 is the single shared contract test. |
| REQ-NFR-001 through REQ-NFR-007 | Supporting TC-011 through TC-015 | Yes | Supporting non-functional tests below. |

### Feature-level design coverage

No sibling task specs exist. Every design element in spec.md maps to an
acceptance criterion and implementation/test path.

| Design element | Spec location | Coverage | Paths |
|---|---|---|---|
| Team models and bounded evidence | Architecture lines 92-98 | AC-001, 007, 009; TC-001, 007, 009, 014 | internal/team/models.go and models_test.go |
| Planner and dependency normalization | Architecture lines 94-98 | AC-001 through 005, 008; TC-001 through 005, 008 | internal/team/plan.go and dependency_adapter_test.go |
| Ledger service/repository | Architecture lines 97-100 | AC-006 through 009; TC-006 through 009 | internal/team/ledger_service.go and repository/teamrun |
| Migration, constraints, indexes | Architecture lines 101-102 and Data model | AC-006/007; TC-006, 011, 012 | internal/db/db.go and team_run_migration_test.go |
| Shared dispatch-step seam | Architecture lines 103-106 | AC-001, 003, 010; TC-001, 003, 010 | internal/dispatch/step.go, next.go, run.go |
| Shared I-01 contract | Architecture line 107 and Cross-feature interactions | AC-001; TC-001 | tests/contracts/e38_interactions_test.go#TC-001 |

## Acceptance criteria review

### Ambiguity findings

1. AC-F01-005 is resolved by the capability decision table in spec.md:
   ordinary single-worker execution plus serializable work selects sequential
   mode; absence of any required adapter returns a pre-mutation capability
   error.
2. AC-F01-007 is resolved by the explicit `ErrPlanDrift` and
   `ErrConflictingTerminalResult` contracts in spec.md.

### Missing coverage

None. The two findings are interpretation gaps, not untested behaviors.

## ISTQB technique application (per AC)

| AC | Technique(s) | Test cases | Rationale |
|---|---|---|---|
| AC-F01-001 | Equivalence Partitioning, Contract Surface Enumeration, State Transition | TC-001 | Child eligibility/exclusion partitions, shared shape, read-only state. |
| AC-F01-002 | Attack-class Enumeration, State Transition | TC-002 | Cycle, missing reference, malformed JSON, unresolved workflow classes. |
| AC-F01-003 | Equivalence Partitioning, Contract Surface Enumeration | TC-003 | Legacy, relationship, duplicate dual-source, malformed inputs. |
| AC-F01-004 | Boundary Value Analysis, Decision Table | TC-004 | Concurrency 0, 1, requested, host cap, and over-cap cases. |
| AC-F01-005 | Decision Table, Equivalence Partitioning | TC-005 | Safe, unsafe, unavailable, unknown, and resource-conflict capability classes. |
| AC-F01-006 | State Transition, Attack-class Enumeration | TC-006 | Transaction, failure, rollback, retry, and uniqueness states. |
| AC-F01-007 | State Transition, Contract Surface Enumeration | TC-007 | Terminal idempotency, conflict, and explicit retry transitions. |
| AC-F01-008 | Equivalence Partitioning, Contract Surface Enumeration | TC-008 | Material versus volatile hash inputs. |
| AC-F01-009 | Boundary Value Analysis, Attack-class Enumeration | TC-009 | Evidence limits, secrets, prompts, output, and path attacks. |
| AC-F01-010 | Contract Surface Enumeration, Equivalence Partitioning | TC-010 | Real next/run callers and metadata/pause/terminal partitions. |

## ISO 25010 coverage matrix

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-F01-001 | TC-001 | TC-013 | TC-001 (I-01) | TC-001 exclusions | TC-001 repeat | TC-001 no mutation | TC-001 seam | N/A: local domain |
| AC-F01-002 | TC-002 | N/A: no AC SLO | N/A: no version boundary | TC-002 diagnostics | TC-002 no partial plan | TC-002 input rejection | TC-002 typed error | N/A: parser contract |
| AC-F01-003 | TC-003 | N/A: no adapter SLO | TC-003 legacy/new formats | N/A: internal adapter | TC-003 loud failure | TC-003 malformed JSON | TC-003 normalized seam | N/A: repository-defined |
| AC-F01-004 | TC-004 | TC-004 bounded limit | N/A: injected host facts | TC-004 actual mode | TC-004 no worker | TC-004 resource cap | TC-004 deterministic waves | TC-004 capability facts |
| AC-F01-005 | TC-005 | TC-005 fallback | N/A: injected host facts | TC-005 actionable reason | TC-005 no unsafe parallel | TC-005 ownership safety | TC-005 pending decision table | TC-005 capability branch |
| AC-F01-006 | TC-006 | TC-012 | TC-006 SQLite/Turso | N/A: persistence | TC-006 rollback/retry | TC-006 constraints | TC-006 layer boundary | TC-011 migration |
| AC-F01-007 | TC-007 | N/A: no throughput SLO | TC-001 shared shape | TC-007 terminal state | TC-007 idempotency | TC-007 membership authority | TC-007 pending API choice | N/A: SQLite contract |
| AC-F01-008 | TC-008 | TC-008 serializer | TC-008 row-order independence | TC-008 refresh result | TC-008 no silent merge | TC-008 no secret hash | TC-008 canonical serializer | N/A: fixed SHA-256 |
| AC-F01-009 | TC-009 | TC-009 bounded evidence | N/A: no external boundary | TC-009 validation message | TC-009 pre-write reject | TC-009 secret/path defense | TC-009 central validator | N/A: project policy |
| AC-F01-010 | TC-010 | N/A: no new SLO | TC-010 ordinary callers | TC-010 existing output | TC-010 ownership | TC-010 no CLI recursion | TC-010 shared seam | N/A: CLI contract |

### Coverage gaps

- AC-F01-005 × Maintainability: add the capability decision table, then rerun
  TC-005.
- AC-F01-007 × Maintainability: choose existing-run success versus typed conflict,
  then rerun TC-007.

## Observability design

These are implementation requirements. Evidence includes root/run/child
identifiers and counts, but never prompts, credentials, tokens, unrestricted
stdout/stderr, or transcripts.

| Behavior | Metric | Log | Trace span | Alert | Test assertion |
|---|---|---|---|---|---|
| Plan success/failure | team_plan_build_total{root_type,result} | team_plan_built | team.plan | N/A: no SLO | TC-001/002 sink assertion |
| Graph/workflow validation failure | team_plan_validation_failures_total{cause} | team_plan_validation_failed | team.plan.validate | Ops-owned spike | TC-002 safe cause only |
| Atomic ledger persistence | team_ledger_persist_total and rollback_total | team_ledger_persisted/rollback | team.ledger.persist | Rollback baseline | TC-006 commit/rollback |
| Duplicate/conflicting result | team_ledger_idempotency_total{result} | team_ledger_idempotency | team.ledger.record_result | Conflict baseline | TC-007 result |
| Material plan drift | team_plan_drift_total{change_class} | team_plan_drift_detected | team.plan.drift | N/A: refresh control | TC-008 |
| Sensitive evidence rejection | team_ledger_rejected_input_total{cause} | team_ledger_input_rejected | team.ledger.validate_result | Rejection baseline | TC-009 no submitted values |
| Dispatch metadata compatibility | team_dispatch_step_resolution_total{caller,result} | team_dispatch_step_resolved | team.dispatch_step.resolve | N/A: regression | TC-010 |

## Caller-Path Contracts

Every test case has the following contract fields. The listed entrypoint must
be driven with the production argument shape; the lowest seam may be mocked only
where stated.

| TC | Entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | Planner.Plan(ctx, PlanInput{RootType:"feature", RootKey:"E38-F01-fixture", RequestedConcurrency:2, Capabilities:facts}) | Child/dependency/workflow/claim/capability readers | Do not mock Planner, ledger, CLI, or mutation | Dispatchable-only listing or claim mutation changes child count/state. |
| TC-002 | Planner.Plan(ctx, PlanInput{RootType:"feature", RootKey:"E38-F01-fixture", ...}) | Dependency and dispatch readers | Do not call only a graph helper or mock validation result | Swallowed malformed/missing edge returns a partial plan. |
| TC-003 | Planner.Plan(ctx, PlanInput{RootType:"feature", RootKey:"E38-F01-fixture", ...}) via planner dependency adapter | Legacy and relationship repositories | Do not bypass the adapter or mock normalized edges | Dual-source input yields duplicate edges or malformed JSON is ignored. |
| TC-004 | Planner.Plan(ctx, PlanInput{..., RequestedConcurrency:2, Capabilities:capable}) | Child/dependency/workflow/capability readers | Do not mock mode choice or start dispatcher | Host cap is ignored or planning starts work. |
| TC-005 | Planner.Plan(ctx, PlanInput{..., RequestedConcurrency:4, Capabilities:unsafe}) | Capability/resource provider | Do not mock selected mode or suppress reason | Safe single-worker host selects sequential mode with an exact degraded reason; no-adapter host returns a typed capability error before mutation. |
| TC-006 | Ledger.PersistConfirmedPlan(ctx, plan, "root-session-001") | Real teamrun repository and SQLite failure fixture | Do not mock transaction, SQL constraints, or rows | Mid-insert failure leaves run or partial items. |
| TC-007 | Ledger.PersistConfirmedPlan then RecordItemResult(ctx, ItemResultUpdate{RunID:1,ItemID:1,Attempt:0,...}) twice | Service repository seam; real SQLite repository coverage | Do not use only an in-memory map or bypass state validation | Same root/hash returns the existing run idempotently; hash drift returns ErrPlanDrift; identical terminal result is idempotent; conflicting terminal result returns ErrConflictingTerminalResult. |
| TC-008 | Planner.Plan twice, then Ledger.GetRun(ctx, 1) | Injected readers and ledger lookup repository | Do not mock hash or compare only object identity | Material drift or row-order change has the wrong hash/result. |
| TC-009 | Ledger.RecordItemResult(ctx, ItemResultUpdate{Evidence:evidence,ArtifactRefs:refs}) | Validator/repository boundary | Do not sanitize after persistence or pre-sanitize input | Prompt, secret, raw output, or traversal path reaches storage. |
| TC-010 | Cobra root invocations shark next <key> and shark run <key> | Mocked lower service/repository and run dispatcher | Do not only call resolveNext/runRun helpers or mock shared resolver | Production wiring, prompt, ownership, or status behavior changes. |
| TC-011 | db.ApplySchemaIfNeeded(ctx) on fresh/current/partial/v28 DBs | Real SQLite test database | Do not mock migration SQL or delete source DB | Rerun fails or existing data is rewritten. |
| TC-012 | Ledger.PersistConfirmedPlan(ctx, plan, "root-session-001") with busy fixture | DB/repository busy seam | Do not sleep-and-hope or mock successful commit | Retry duplicates rows or holds write transaction during reads. |
| TC-013 | Planner.Plan(ctx, PlanInput{...}) with shuffled reader results | Injected permutation readers | Do not mock canonical serializer | Equivalent database order changes plan_hash. |
| TC-014 | TeamRunResult conversion from persisted run and complete item list | Domain conversion boundary; repository read integration | Do not use a hand-built DTO bypassing validation | I-01 shape omits a required field or leaks prompt/secret. |
| TC-015 | Ledger.RecordItemResult(ctx, ItemResultUpdate{...}) for all field boundaries | Model validation before repository | Do not validate only SQL or private helpers | Invalid enums, numbers, keys, or text reach persistence. |

## Acceptance test cases

### TC-001: Read-only complete snapshot and I-01 contract

**Requirements:** REQ-F-001, REQ-F-003, REQ-F-005, REQ-F-006, REQ-F-008;
architecture §4.1-4.2; UAT-01. **I-01 tag:** this is the exact shared test at
tests/contracts/e38_interactions_test.go#TC-001, referenced by F02, F03, and F05.

**AC:** AC-F01-001. **Technique:** Equivalence Partitioning, Contract Surface
Enumeration, State Transition. **ISO:** Functional Suitability, Compatibility,
Usability, Reliability, Security, Maintainability.

**Setup/input:** A feature has four direct children: two independent children,
one child depending on both, and one approval boundary. The complete snapshot
fixture also includes terminal, claimed, capability-excluded, and
dependency-ineligible partitions. Resolved metadata is role developer, provider
anthropic, model claude-sonnet, effort medium, plus one unresolved-placeholder
diagnostic. Capture entity, claim, note, file, and ledger state before two
identical Planner.Plan calls.

**Expected:** Every direct child appears once with canonical identity, status,
wave, metadata, and explicit exclusion. The dependent child follows both
predecessors; the approval gate is visible; hashes match; no claim, status,
note, file, or ledger state changes; and no rendered prompt is in plan/ledger.
The shared result assertion covers root identity, run fields, complete item
results, wave, planned/actual metadata, claim/session links, attempt, outcome,
skip reason, evidence, and timestamps.

**Edges/negative:** zero or one child; reordered equivalent metadata; live claim
is represented but not stolen; duplicate direct-child rows fail; no mutation
method or dispatcher may be called. Assert team_plan_build_total, log, and
team.plan evidence without prompts or secrets.

**Caller-Path Contract:** Entrypoint Planner.Plan with the concrete PlanInput
above. Lowest seam is injected readers. Forbidden mocks are Planner, ledger,
CLI, and mutators. Counter-factual: dispatchable-only listing or a claim write
changes the snapshot or returns fewer than four children.

### TC-002: Reject invalid dependency and workflow graphs

**Requirements:** REQ-F-002; UAT-03/UAT-04 safety. **AC:** AC-F01-002.
**Technique:** Attack-class Enumeration and State Transition. **ISO:** Functional
Suitability, Usability, Reliability, Security, Maintainability.

**Setup/input:** Call Planner.Plan for cycle A→B→A, missing child
T-MISSING-999, malformed legacy JSON {"depends_on":, and an unresolved workflow
step.

**Expected:** A typed error identifies root, offending child/reference, and one
cause: cycle, missing_reference, malformed_dependency, or unresolved_workflow.
No partial dispatchable plan or ledger row exists. Assert validation metric/log/
span includes cause but not sensitive input.

**Edges/negative:** self-cycle, valid successful external prerequisite, non-success
external prerequisite, empty graph, unresolved placeholder. Never drop an edge,
hardcode success, or swallow parse failure.

**Caller-Path Contract:** Entrypoint Planner.Plan(ctx, PlanInput{RootType:"feature",
RootKey:"E38-F01-fixture", ...}). Lowest seam is injected dependency/workflow
readers. Forbidden mocks are direct graph-only tests and mocked validation.
Counter-factual: a swallowed missing edge returns a dispatchable plan.

### TC-003: Normalize legacy and relationship dependency sources

**Requirements:** REQ-F-002 and dependency compatibility decision.
**AC:** AC-F01-003. **Technique:** Equivalence Partitioning and Contract Surface
Enumeration. **ISO:** Functional Suitability, Compatibility, Reliability, Security,
Maintainability.

**Setup/input:** For T-E38-F01-003 provide legacy-only dependency to
T-E38-F01-001, relationship-only dependency to T-E38-F01-002, both sources
naming T-E38-F01-001, and malformed legacy JSON. Add new dependency through the
relationship service.

**Expected:** Each valid source produces one canonical sorted edge; dual source
deduplicates by canonical key; new writes use entity_relationships; malformed
JSON fails loudly and leaves no partial graph.

**Edges/negative:** reverse relationship order, case/slug equivalent keys,
distinct typed entities with same text key, external prerequisite. No duplicate
wave edge or silent source preference.

**Caller-Path Contract:** Entrypoint is Planner.Plan through the
planner-facing dependency adapter. Lowest seam is legacy/relationship
repositories. Forbidden mocks are normalized edges and parser-only tests.
Counter-factual: a source preference produces duplicate or missing dependency.

### TC-004: Select bounded parallel mode and deterministic waves

**Requirements:** REQ-F-004; UAT-03. **AC:** AC-F01-004. **Technique:** Boundary
Value Analysis and Decision Table. **ISO:** Functional Suitability, Performance
Efficiency, Usability, Reliability, Security, Maintainability, Portability.

**Setup/input:** Three independent children, capable host limit 2; requested
concurrency 1, 2, 3, 4, 0, and -1.

**Expected:** Requested 2 reports parallel, limit 2, deterministic waves, and no
worker. Requests 3/4 cap at 2; request 1 gives limit 1; zero/negative are typed
validation errors before mutation.

**Edges/negative:** one child, exact host limit, over-cap, tied order/priority.
No parallel result has non-positive/over-cap limit and no dispatcher starts.

**Caller-Path Contract:** Entrypoint Planner.Plan with RequestedConcurrency and
CapabilityFacts as above. Lowest seam is capability/child readers. Forbidden
mocks are mode selection and dispatcher. Counter-factual: ignored host cap
reports limit 4.

### TC-005: Use explicit sequential fallback for unsafe capability facts

**Requirements:** REQ-F-004; UAT-07. **AC:** AC-F01-005. **Technique:** Decision
Table and Equivalence Partitioning. **ISO:** Functional Suitability, Performance
Efficiency, Usability, Reliability, Security, Maintainability, Portability.

**Setup/input:** Test safe team/isolation, no safe team, unavailable worktree,
unknown overlapping ownership, and resource ownership conflict.

**Expected:** For each capability class, the final specification's decision table
must select sequential with actionable reason or a typed capability error. Never
parallel on an unsafe host and never mutate state.

**Edges/negative:** one child, all excluded, one safe plus one unknown child,
omitted capabilities; reason must be non-empty and safe.

**Caller-Path Contract:** Entrypoint Planner.Plan(ctx, PlanInput{..., RequestedConcurrency:4,
Capabilities:unsafe}). Lowest seam is capability/resource provider. Forbidden
mocks are selected mode and reason. Counter-factual: unsafe host reports parallel.
This test is blocked from unconditional approval until the decision table exists.

### TC-006: Persist a confirmed plan atomically

**Requirements:** REQ-F-006, NFR-002, NFR-007; UAT-01/UAT-06. **AC:** AC-F01-006.
**Technique:** State Transition and Attack-class Enumeration. **ISO:** Functional
Suitability, Performance, Compatibility, Reliability, Security, Maintainability,
Portability.

**Setup/input:** Valid four-item plan, root-session-001, item insert failure at
item four; inspect rows before/after failure/retry.

**Expected:** Failed transaction leaves neither run nor items. Retry creates one
complete run and four unique membership rows. Preview writes zero rows.
Foreign-key cascade and unique membership are enforced.

**Edges/negative:** empty/one item, duplicate input, commit error, FK error,
busy then success. No partial run/item or pre-first-claim write.

**Caller-Path Contract:** Entrypoint Ledger.PersistConfirmedPlan(ctx, plan,
"root-session-001"). Lowest seam is real teamrun repository and SQLite failure
fixture. Forbidden mocks are transaction, constraints, and row state.
Counter-factual: mid-insert failure leaves a run or partial items.

### TC-007: Preserve membership and idempotent terminal results

**Requirements:** REQ-F-007; UAT-02/UAT-06. **AC:** AC-F01-007. **Technique:**
State Transition and Contract Surface Enumeration. **ISO:** Functional Suitability,
Compatibility, Usability, Reliability, Security, Maintainability.

**Setup/input:** Persist same plan twice. Record item 1 attempt 0 passed with
bounded evidence twice; submit conflicting failed result; explicitly retry at
attempt 1.

**Expected:** One run and one row per child; identical terminal result is
idempotent; conflicting result is rejected without overwrite; explicit retry is
attempt 1; completed item is not implicitly dispatched. Repeated-plan outcome
must follow the clarified existing-run-success or typed-conflict contract.

**Edges/negative:** non-terminal repeat, unknown item, attempt -1, failed retry,
reordered artifact refs. No duplicate membership, silent overwrite, or implicit
attempt increment. Assert idempotency metric/log/span.

**Caller-Path Contract:** Entrypoint PersistConfirmedPlan then
RecordItemResult(ctx, ItemResultUpdate{RunID:1, ItemID:1, Attempt:0, ...}).
Lowest seam is service repository seam with real SQLite coverage. Forbidden mocks
are in-memory-only maps and bypassed state validation. Counter-factual: duplicate
write creates membership or changes terminal outcome.

### TC-008: Detect material plan drift and require refresh

**Requirements:** REQ-F-005/007; UAT-06. **AC:** AC-F01-008. **Technique:**
Equivalence Partitioning and Contract Surface Enumeration. **ISO:** Functional
Suitability, Performance, Compatibility, Usability, Reliability, Security,
Maintainability.

**Setup/input:** Persist plan, then change exactly one of child status,
dependency edge, workflow metadata, mode, or concurrency. Separately reorder
rows and alter timestamps, claim IDs, prompts, and worker output.

**Expected:** Each material change produces a different lowercase 64-character
SHA-256 and refresh-required result. Row order and volatile fields preserve hash.
No silent merge. Assert drift metric/log/span.

**Edges/negative:** add/remove excluded child, reorder dependency only, absent versus
empty optional metadata, unchanged second plan. Prompt/credential never hashes.

**Caller-Path Contract:** Entrypoint two Planner.Plan calls followed by
Ledger.GetRun(ctx, 1). Lowest seam is injected readers and lookup repository.
Forbidden mock is hash. Counter-factual: material drift has same hash or is merged.

### TC-009: Reject sensitive and unbounded evidence

**Requirements:** NFR-004 and REQ-F-007. **AC:** AC-F01-009. **Technique:** Boundary
Value Analysis and Attack-class Enumeration. **ISO:** Functional Suitability,
Performance, Usability, Reliability, Security, Maintainability.

**Setup/input:** Evidence at max-1/max/max+1 bytes; rendered prompt; Bearer
sk-test-123; AWS-like key; private-key marker; raw stdout/stderr; artifact refs
artifacts/run-1/result.md, ../secret, and /tmp/secret.

**Expected:** Valid bounded summary and project-relative ref persist. All other
inputs reject before persistence; rows/JSON contain no forbidden content.

**Edges/negative:** empty/null, Unicode byte boundary, safe word containing
prompt, nested project path, duplicate ref. Never truncate then store a secret
or accept path traversal. Assert rejection evidence contains cause only.

**Caller-Path Contract:** Entrypoint Ledger.RecordItemResult(ctx,
ItemResultUpdate{Evidence:evidence, ArtifactRefs:refs}). Lowest seam is validator
then repository. Forbidden mock is pre-sanitized input or post-persistence
redaction. Counter-factual: prompt/secret reaches storage before rejection.

### TC-010: Preserve ordinary next and run behavior

**Requirements:** REQ-F-003/NFR-006 and existing integration seams; UAT gate.
**AC:** AC-F01-010. **Technique:** Contract Surface Enumeration and Equivalence
Partitioning. **ISO:** Functional Suitability, Compatibility, Usability,
Reliability, Security, Maintainability.

**Setup/input:** Same task/feature fixture before and after extraction. Invoke
real Cobra paths shark next T-E38-F01-001 and shark run T-E38-F01-001 with
lower-level services and dispatcher mocked.

**Expected:** NextResponse, action, status, metadata, placeholders, prompt
assembly, and ownership/status behavior are unchanged. Team planning uses shared
structured metadata without Cobra recursion or prompt persistence.

**Edges/negative:** task/feature/approval, unresolved placeholder, terminal child,
dry run, provider failure. No team ledger/claim/root transition from ordinary
next/run. Assert resolver metric/log/span identifies caller, not prompt.

**Caller-Path Contract:** Entrypoint is the actual Cobra root invocation with
production argv. Lowest seam is service/repository and dispatcher. Forbidden
mocks are resolveNext/runRun as the only path and shared resolver. Counter-
factual: production wiring changes response/prompt/ownership.

## Supporting non-functional and infrastructure cases

### TC-011: Additive migration across database states

**Requirement:** NFR-007. **Technique:** State Transition and Equivalence
Partitioning. **ISO:** Functional, Compatibility, Reliability, Security,
Maintainability, Portability.

Apply schema to fresh, v27 with existing entities/claims/sessions, partial
team tables, and v28 databases; rerun twice. Expect version 28, tables,
constraints, indexes, cascade, idempotency, unchanged existing rows, and
rejection of invalid enums/negative values.

**Caller-Path Contract:** Entrypoint db.ApplySchemaIfNeeded(ctx). Lowest seam is
real SQLite test DB. Forbidden mock is migration SQL or source DB deletion.
Counter-factual: rerun fails or rewrites existing data.

### TC-012: Short transactions and busy retry

**Requirement:** NFR-001/002. **Technique:** Boundary Value Analysis, State
Transition, Attack-class Enumeration. **ISO:** Functional, Performance,
Reliability, Maintainability.

Use roots with 0, 1, 10, and 100 children; instrument transaction timing and
inject one transient busy/locked result. Expect work bounded by child count, no
write transaction during metadata/wave/capability reads, bounded retry, no
duplicates, and cancellation cleanup.

**Caller-Path Contract:** Entrypoint Ledger.PersistConfirmedPlan(ctx, plan,
"root-session-001"). Lowest seam is DB/repository busy fixture. Forbidden mock is
successful commit or sleep-only retry. Counter-factual: retry duplicates or
holds a write transaction during reads.

### TC-013: Canonical hash serialization permutations

**Requirement:** NFR-003. **Technique:** Equivalence Partitioning, Boundary
Value Analysis, Contract Surface Enumeration. **ISO:** Functional, Performance,
Compatibility, Security, Maintainability, Portability.

Shuffle children, dependency keys, metadata lists, optional absent/empty values,
and volatile fields. Expect sort by wave/order/priority/key, sorted lists,
absent distinct from empty, lowercase UTF-8 JSON SHA-256, stable equivalents,
different material inputs, and no prompt/credential hashing.

**Caller-Path Contract:** Entrypoint Planner.Plan(ctx, PlanInput{...}) with
permuted reader results. Lowest seam is permutation readers. Forbidden mock is
canonical serializer. Counter-factual: equivalent row order changes plan_hash.

### TC-014: TeamRunResult shape and serialization

**Requirement:** REQ-F-008, architecture §4.2, I-01. **Technique:** Contract
Surface Enumeration. **ISO:** Functional, Compatibility, Usability, Security,
Maintainability.

Read a run with planned/completed/failed/blocked/skipped/paused items, links,
attempts, outcomes, skip reasons, evidence, timestamps, nullable and populated
aggregate/next-action. Expect every root/run/item field and JSON round-trip,
with no prompt/secret.

**Caller-Path Contract:** Entrypoint TeamRunResult conversion from persisted run
and complete item list. Lowest seam is domain conversion plus repository read.
Forbidden mock is hand-built DTO bypassing validation. Counter-factual: I-01 shape
omits a required field or leaks prompt/secret.

### TC-015: Model and persistence boundaries

**Requirement:** NFR-005/006. **Technique:** Boundary Value Analysis,
Equivalence Partitioning, Attack-class Enumeration. **ISO:** Functional,
Reliability, Security, Maintainability, Portability.

Exercise null/empty/min/max/max+1 for keys, enums, mode, limit, wave, attempt,
JSON lists, bounded text, and paths; call every public service method context
first with cancellation and repository errors. Expect valid values persist,
invalid values fail before SQL with wrapped operation/entity context, and roster
membership grants no workflow/claim authority.

**Caller-Path Contract:** Entrypoint Ledger.RecordItemResult(ctx,
ItemResultUpdate{...}) and each context-first public service method. Lowest seam
is model validator before repository. Forbidden mock is SQL-only validation or
private-helper-only tests. Counter-factual: invalid enum/number/key/text reaches
persistence.

## Integration scenarios and parent UAT coverage

| Scenario | Boundary | Verification | UAT |
|---|---|---|---|
| Plan-to-run continuity | planner → ledger → F02/F03/F05 | Persist exact waves, metadata, exclusions, hash, and membership | UAT-01, plan half of UAT-03/UAT-06 |
| Dependency safety | child repos → adapter → planner | Normalize edges once; reject cycles/missing targets; show external reason | UAT-03, UAT-04 |
| Workflow boundary | workflow/action config → resolver → planner/next/run | Canonical metadata; transient prompt discarded; CLI parity | UAT-01, UAT-03, UAT-05 |
| Capability boundary | host facts → planner | Actual mode/limit/reason; unsafe parallel impossible | UAT-03, UAT-07 |
| Durable ledger | service → SQLite repository/migration | Atomic membership, constraints, retry, idempotency, drift | UAT-02, UAT-06 |
| Claim/session links | diagnostics → plan/ledger | F01 reads and stores links only; never claims or releases | UAT-01, UAT-02, UAT-06 |

## Test infrastructure

### Existing patterns

- Real repository DB fixture and cleanup: internal/repository/*_test.go and
  internal/test/testdb.go.
- Mocked service/repository tests: internal/services/*_test.go and injected F01
  interfaces.
- Additive migrations: internal/db/db.go and internal/db/*migration*_test.go.
- CLI mocks and command execution: internal/cli/commands/*_test.go.
- Claim/session patterns: internal/repository/claim/claim_repository.go and
  internal/repository/worksession/repository.go.
- Canonical dispatch resolution: internal/cli/commands/next.go, run.go,
  internal/config/workflow/schema.go.

### New helpers required

- Fixture builders for typed roots, child partitions, graphs, dispatch metadata,
  capabilities, and complete TeamPlan values.
- SQLite failure/busy fixtures exposing row counts and transaction phases.
- Mutation snapshot helper for entity rows, claims, notes, files, and ledger rows.
- E23-compatible observability sink that rejects prompt/token/unrestricted output.
- Exact shared contract fixture tests/contracts/e38_interactions_test.go#TC-001;
  F02/F03/F05 reference it rather than adding twin tests.

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumers | Shape source | Contract pointer | TC |
|---|---|---|---|---|---|
| I-01 | E38-F01 | E38-F02, E38-F03, E38-F05 | E38 architecture §4.2 Team-run domain contract | tests/contracts/e38_interactions_test.go#TC-001 | TC-001 |

TC-001 asserts root identity, run status, mode/limit, plan hash, complete item
results, wave, planned/actual metadata, claim/session links, attempt, outcome,
skip reason, evidence, and timestamps. It proves shape, not merely a call.

## Cross-epic integration tests (X-##)

None are declared for F01. E38-cross-epic-map.md and the global product map
assign X-01 through X-05 to later features. No deferral entry is needed in
docs/product/progress.md.

## Codex Test-Plan Red-Team

**Verdict:** FAIL (Codex unavailable; non-blocking execution failure)
**Issues raised:** 1 availability failure; substantive findings unavailable
**Issues addressed before development:** 2 specification ambiguities resolved
in spec.md and reflected in TC-005/TC-007
**Issues deferred:** 1 — rerun the red-team after Codex authentication is
restored; this is an environment verification dependency, not an unresolved
feature contract

The required red-team must inspect every AC for open-endedness, ISTQB fit,
enumeration completeness, ISO coverage, runtime evidence, negative cases, and
Caller-Path Contracts. It could not complete because the local Codex CLI could
not authenticate. Per the workflow, this is documented but does not gate the
artifact by itself.

### Codex output (verbatim)

Attempt 1:

    WARNING: proceeding, even though we could not create PATH aliases: Read-only file system (os error 30)
    Reading additional input from stdin...
    Error: failed to initialize in-process app-server client: Read-only file system (os error 30)

Attempt 2 (after retry with an ephemeral CODEX_HOME under /tmp):

    WARNING: proceeding, even though we could not create PATH aliases: Refusing to create helper binaries under temporary dir "/tmp" (codex_home: AbsolutePathBuf("/tmp/codex-e38"))
    Reading additional input from stdin...
    OpenAI Codex v0.144.1
    --------
    workdir: /home/jwwel/projects/shark-task-manager
    model: gpt-5.6-sol
    provider: openai
    approval: never
    sandbox: read-only
    reasoning effort: none
    reasoning summaries: none
    session id: 019f5a0e-c680-77f2-af32-82d12e6a0d10
    --------
    2026-07-13T05:59:05.866344Z ERROR codex_api::endpoint::responses_websocket: failed to connect to websocket: HTTP error: 401 Unauthorized, url: wss://api.openai.com/v1/responses
    2026-07-13T05:59:06.011598Z ERROR codex_api::endpoint::responses_websocket: failed to connect to websocket: HTTP error 401 Unauthorized, url: wss://api.openai.com/v1/responses
    ERROR: Reconnecting... 2/5
    ERROR: Reconnecting... 3/5
    ERROR: Reconnecting... 4/5
    ERROR: Reconnecting... 5/5
    warning: Falling back from WebSockets to HTTPS transport. unexpected status 401 Unauthorized: Missing bearer or basic authentication in header
    ERROR: unexpected status 401 Unauthorized: Missing bearer or basic authentication in header

**Codex test-plan review: FAILED — local Codex CLI could not authenticate to the OpenAI Responses endpoint (401 Unauthorized).**

## Recommendations and exit verdict

- [x] Ready for development after the two ambiguity decisions were made and
  encoded in spec.md and the test cases.
- [ ] Codex red-team review complete; local authentication remains unavailable
  with HTTP 401 and must be rerun when the verification environment is restored.
- [x] Observability hooks remain explicit implementation acceptance criteria for
  F01, not a later reporting-only concern.

**Current verdict:** READY_FOR_DEVELOPMENT_WITH_CODEX_REVIEW_DEFERRED. Every AC
has a concrete test, named technique, ISO coverage, caller-path contract, edge
and negative cases; I-01 uses the declared shared pointer; F01 has no X-## row;
and the two previously ambiguous outcomes are deterministic. The independent
Codex review remains explicitly deferred because the local CLI cannot
authenticate, and must be rerun before a final acceptance gate.
