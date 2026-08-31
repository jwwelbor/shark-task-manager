# Test plan: E19-F10 — Roadmap-gated sprint admission and goal acceptance

**Status:** APPROVED

## Inputs and drift analysis

Reviewed the feature brief, `spec.md`, E19 epic PRD, and the canonical
`uat-acceptance-plan.md`. The workflow prompt calls the latter `uat-plan.md`;
that path is absent, so this plan records and uses the committed canonical file.
The specification is traceable to the brief's ancestor gate, portfolio gate,
reasoned override, readiness failure, planning output, and goal-review close
rule. Its evaluator and persistent records make those requirements testable;
they do not add product scope. No unresolved drift or PRD-completeness gap was
found.

## AC test matrix

| AC | Techniques | Cases | Expected evidence |
| --- | --- | --- | --- |
| AC-1 / REQ-F10-001..002 | Decision table; contract-surface enumeration | TC-001, TC-002 | Same decision/reason at every consumer; errors fail closed. |
| AC-2 / REQ-F10-003 | BVA; state transition | TC-003, TC-004 | Exactly one auditable override; bulk obeys same gate. |
| AC-3 / REQ-F10-004 | Decision table | TC-005 | Blocked item forces factor zero. |
| AC-4 / REQ-F10-005 | Differential contract; state transition | TC-006 | Plan and next return allowed rank 2 and write nothing. |
| AC-5 / REQ-F10-005, NF10-002 | Contract-surface enumeration | TC-007 | JSON gate/reason-count output and no write. |
| AC-6 / REQ-F10-006, NF10-003 | State transition; equivalence partitioning | TC-008, TC-009 | Absent/rejected review cannot complete; accepted review commits atomically. |

## Acceptance cases and caller-path contracts

### TC-001: Shared blocked decision reaches every consumer

**Requirements:** REQ-F10-001, REQ-F10-002, REQ-NF10-001; AC-1.
**Technique:** decision table over unmet/met prerequisite, inside/outside
portfolio, and configured terminal-success/nonterminal workflow states;
contract-surface enumeration over add, bulk add, plan, selection, and next.
**Setup and expected:** A task under an epic with an unmet ancestor dependency
is rejected before individual assignment, skipped/reported by bulk add, and
omitted by plan, selector, and next with the exact same reason code. An
otherwise identical allowed candidate remains eligible. A custom terminal
success status proves metadata, not literal status names, decides eligibility.
**Negative:** missing dependency/portfolio data is an error, never an allow.

**Caller-Path Contract:**
- Entrypoints: `SprintService.AddEntityToSprint(ctx, AddEntityInput{SprintKey, EntityKey})`, `BulkAddToSprint`, `PlanSprint(ctx, sprintKey)`, `SelectSprint(ctx, request)`, and `GetNextTask(ctx, agentType)`.
- Lowest allowed mock seam: assignment/epic/dependency repositories and workflow metadata; use isolated real SQLite for persisted assertions.
- Forbidden mocks: admission evaluator, prefiltered candidate list, or consumer final decision.
- Counter-factual: independent command predicates admit or omit the same blocked candidate differently.

### TC-002: Evaluator fails closed and uses bounded reads

**Requirements:** REQ-F10-002, REQ-NF10-001, REQ-NF10-004; AC-1.
**Technique:** failure-class and contract-surface enumeration.
**Expected:** repository error propagates with no assignment/override; a call spy
observes batched ancestor lookup, not one dependency lookup per candidate.

**Caller-Path Contract:**
- Entrypoint: production evaluator through `SprintService.AddEntityToSprint`.
- Lowest allowed mock seam: dependency and epic repositories.
- Forbidden mocks: evaluator, terminal-status predicate, or a fabricated decision.
- Counter-factual: a fail-open/N+1 implementation assigns work or loops lookup calls.

### TC-003: Valid reason authorizes exactly one override

**Requirements:** REQ-F10-003, REQ-NF10-003; AC-2.
**Technique:** BVA for trimmed lengths 19/20/500/501 and whitespace; blocked →
overridden state transition.
**Expected:** only 20–500 non-whitespace characters permit assignment, persist
one immutable override with original reason/requester, and make plan/readiness
say `overridden`. Invalid length/blank creates neither assignment nor override;
retry creates no second active record.

**Caller-Path Contract:**
- Entrypoint: `runSprintAdd(cmd, [sprintKey, entityKey])` with `--override-reason`, then `AddEntityToSprint`.
- Lowest allowed mock seam: CLI service constructor; real SQLite for uniqueness/transaction assertions.
- Forbidden mocks: flag parsing, override validator/repository, or success decision.
- Counter-factual: Boolean bypass accepts short input, loses evidence, or duplicates records.

### TC-004: Bulk admission applies the same decision per candidate

**Requirements:** REQ-F10-001, REQ-F10-003, REQ-NF10-003; AC-1/2.
**Technique:** decision table for allowed, blocked-no-override, and
blocked-valid-override candidates.
**Expected:** bulk assigns only allowed/validly overridden work, reports each
exclusion reason, and leaves blocked candidates without assignment/override.

**Caller-Path Contract:**
- Entrypoint: `runSprintAdd` with production `--bulk` arguments.
- Lowest allowed mock seam: entity-discovery/repository interfaces; real DB final rows.
- Forbidden mocks: `BulkAddToSprint` or precomputed eligibility.
- Counter-factual: bulk bypasses individual-add policy.

### TC-005: Blocked assigned work makes roadmap readiness zero

**Requirements:** REQ-F10-004; AC-3.
**Technique:** decision table over zero/one blocked items × two/three healthy items.
**Expected:** four assigned items (one blocked, three otherwise healthy) produce
admission factor zero, overall result naming blocked key/reason, with the
three-item factor unable to offset it.

**Caller-Path Contract:**
- Entrypoint: `SprintService.GetSprintReadiness(ctx, sprintKey)` and `runSprintReadiness` JSON.
- Lowest allowed mock seam: assignment/admission repositories.
- Forbidden mocks: readiness aggregation or precomputed readiness result.
- Counter-factual: capacity/count factors mask a roadmap block.

### TC-006: Plan and next skip blocked rank 1 without side effects

**Requirements:** REQ-F10-001, REQ-F10-005, REQ-NF10-002; AC-4.
**Technique:** differential contract and state transition.
**Setup and expected:** active sprint ranks blocked A first and allowed B second;
`shark plan sprint --json` and `shark sprint next --json` return B. Snapshot
assignment, claim, session, status, review, and override tables before/after:
they are identical. Retain existing claim, Question, terminal, role, and order
filters after admission filtering.

**Caller-Path Contract:**
- Entrypoints: `runPlan(cmd, ["sprint"])`, `runSprintNext(cmd, args)`, and production `SelectSprint`.
- Lowest allowed mock seam: service dependencies; real SQLite for table snapshots.
- Forbidden mocks: handlers, selector/next, and prefiltered claims/questions.
- Counter-factual: one path returns A or creates a lease/status/session.

### TC-007: Planning JSON publishes audit context and remains read-only

**Requirements:** REQ-F10-005, REQ-NF10-002; AC-5.
**Technique:** contract-surface enumeration over `sprint plan`, `plan sprint`,
and `sprint next` JSON envelopes.
**Expected:** response retains F09 selection fields and names selected portfolio
epic plus reason-code exclusion counts; snapshot proves no assignments, overrides,
reviews, claims, sessions, or status writes.

**Caller-Path Contract:**
- Entrypoints: `runSprintPlan`, `runPlan(cmd, ["sprint"])`, `runSprintNext` with `--json`.
- Lowest allowed mock seam: service dependencies; real DB for snapshot.
- Forbidden mocks: response renderer or hand-built JSON payload.
- Counter-factual: audit fields disappear or a read creates an override.

### TC-008: Absent/rejected goal review returns the close attempt to active

**Requirements:** REQ-F10-006, REQ-NF10-003; AC-6.
**Technique:** active → closing → active state transition and partitions
{absent, rejected}.
**Expected:** all-complete sprint with absent or rejected latest review is active
after close and has no completion/carryover write; rejected review remains durable.

**Caller-Path Contract:**
- Entrypoint: `runSprintClose(cmd, [sprintKey])` to `CloseSprintWithCarryover(ctx, input)`.
- Lowest allowed mock seam: isolated real SQLite transaction and repositories.
- Forbidden mocks: close transaction, latest-review query, completion writer, direct status mutation.
- Counter-factual: completed backlog alone completes the sprint.

### TC-009: Accepted complete review authorizes normal close atomically

**Requirements:** REQ-F10-006, REQ-NF10-003; AC-6.
**Technique:** BVA/equivalence partitions for goal, before/after, reviewer, and
outcome; transaction-failure injection.
**Expected:** only accepted review with every nonblank field permits the existing
carryover/completion transaction and preserves evidence. Persistence failure
rolls back completion authorization and completion row.

**Caller-Path Contract:**
- Entrypoints: production goal-review submission followed by `runSprintClose`.
- Lowest allowed mock seam: SQLite transaction boundary (failure injection below it).
- Forbidden mocks: review validation/latest-review lookup/completion insert.
- Counter-factual: malformed/rejected evidence completes or a partial transaction leaks completion.

## Integration coverage

| Scenario | Boundary | Evidence |
| --- | --- | --- |
| Planning admission | add/bulk → evaluator → override persistence | TC-001..004; extends UAT-J2-02/J2-04. |
| Read-only dispatch | F09 plan/next → evaluator → E38 role pull | TC-006..007 and unchanged-write snapshots. |
| X-03 | E19 ordered role-aware pull → E38-F04 | Extend `tests/contracts/e38_f04_interactions_test.go#TC-003` with an F10-tagged blocked-candidate case; shape source E38 §4.1/§4.6. |
| Completion | close transaction → review/completion/carryover | TC-008..009; extends sprint-close UAT. |

No I-## is declared. X-03 has no deferred product-map row, so the shared test is
required rather than duplicated.

## Test infrastructure

- Follow `internal/services/sprint_service_test.go` for service caller paths.
- Follow `internal/cli/commands/sprint_test.go` and
  `internal/cli/commands/plan_sprint_test.go` for CLI/JSON overrides.
- Follow `internal/repository/sprint/repository_test.go` for migration/persistence.
- Extend, rather than duplicate, `tests/contracts/e38_f04_interactions_test.go`.
  Use real SQLite for persistence/rollback; unit tests mock only declared seams.

## ISO 25010 coverage

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| AC-1 | ✅ TC-001 | ✅ TC-002 bounded | ✅ TC-001/X-03 | N/A: JSON API | ✅ TC-002 | ✅ fail-closed TC-001 | ✅ shared evaluator | N/A: existing Go/SQLite |
| AC-2 | ✅ TC-003/4 | N/A: one record | N/A | ✅ flag errors TC-003 | ✅ idempotency TC-003 | ✅ validation TC-003 | ✅ one validator | N/A |
| AC-3 | ✅ TC-005 | N/A: existing scale | N/A | ✅ key/reason TC-005 | ✅ deterministic factor | N/A: local policy | ✅ factor seam | N/A |
| AC-4 | ✅ TC-006 | N/A: bounded by TC-002 | ✅ X-03 | ✅ consistent JSON | ✅ snapshot TC-006 | ✅ no bypass TC-006 | ✅ shared selector | N/A |
| AC-5 | ✅ TC-007 | N/A | ✅ F09 envelope | ✅ audit fields | ✅ no-write snapshot | ✅ no read mutation | ✅ renderer contract | N/A |
| AC-6 | ✅ TC-008/9 | N/A: one transaction | ✅ carryover TC-009 | ✅ validation errors | ✅ rollback TC-009 | ✅ reviewer validation | ✅ transaction boundary | N/A |

## Observability requirements

This local CLI has no metrics pipeline. Production evidence is the persisted
override/review record and structured JSON decision fields. Implementation must
publish decision state/reason, portfolio key, and exclusion counts without
writes; TC-003, TC-005, TC-007, and TC-008/9 verify those durable signals.
Bounded reads are internal evidence verified by TC-002 instrumentation.

## Codex test-plan red-team

**Verdict:** PASS  
**Issues raised:** 2  
**Issues addressed before dev:** 2  
**Issues deferred:** 0

The missing UAT pointer is corrected above. The broad single-evaluator claim is
made falsifiable by cross-consumer production caller paths and forbidden mocks.

## Recommendation

Ready for task generation.

RECOMMENDED OUTCOME: pass

