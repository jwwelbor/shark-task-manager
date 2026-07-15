# Test Plan: E38-F03 - Aggregate Routing and Resume

**Created:** 2026-07-13
**Feature PRD:** `docs/plan/E38-shark-attack-team-orchestration/E38-F03-aggregate-routing-and-resume/feature.md`
**Feature Spec:** `docs/plan/E38-shark-attack-team-orchestration/E38-F03-aggregate-routing-and-resume/spec.md`
**Parent UAT Plan:** `docs/plan/E38-shark-attack-team-orchestration/uat-plan.md`
**Status:** APPROVED

## Scope, drift, and test policy

This plan treats `spec.md` as the task specification and `feature.md` as the
intent source. No implementation has begun and no sibling task specifications
exist in this feature directory; the feature spec's implementation paths and
test pointers are therefore the complete decomposition for this gate.

Testing follows `CLAUDE.md` and `.claude/rules/testing/architecture.md`:
`internal/team` coordinator tests use injected mocks; repository tests under
`internal/repository/teamrun` use the real cleaned test database; service and
CLI tests do not use a real database. Tests must invoke `Aggregate`, `Resume`,
or `Finalize` with production argument shapes, not private aggregation helpers.

### Drift findings

| Finding | Severity | Disposition |
|---|---|---|
| `feature.md` describes outcome aggregation, configured root routing, and safe resume; `spec.md` covers all three and keeps F01/F02/F04/F05 boundaries. | None | No drift. |
| `spec.md` adds explicit secret-safe evidence validation and ordinary-run compatibility. | None | These are traceable to the feature's diagnosability and integration intent; retain as bounded scope. |
| F03 consumes I-01, I-02, and I-04, produces I-03, and owns X-02 exactly as the feature record states. | None | Shared pointers below; no twin contract tests. |

### Traceability matrix

| Feature requirement | Acceptance criteria | Covered by | Notes |
|---|---|---|---|
| REQ-F-001 complete F01 snapshot and missing membership rejection | AC-F03-005, AC-F03-006 | TC-005, TC-006 | Ledger authority and drift are tested before mutation. |
| REQ-F-002/003 complete vocabulary and precedence | AC-F03-001, AC-F03-002 | TC-001, TC-002 | Decision-table cases cover every listed outcome. |
| REQ-F-004/005 configured routing and boundaries | AC-F03-003, AC-F03-004 | TC-003, TC-004 | Uses `workflow.Service.Release`; no positional target. |
| REQ-F-006/007 run selection and fingerprint | AC-F03-005, AC-F03-006 | TC-005, TC-006 | Ambiguity and all material drift classes included. |
| REQ-F-008/009 terminal items and claim/session reconciliation | AC-F03-007 | TC-007 | No force-steal or unscoped release. |
| REQ-F-010 dependency gating | AC-F03-008 | TC-008 | All prerequisite non-success classes included. |
| REQ-F-011 bounded F04 council context | AC-F03-009 | TC-009 | Shared I-04 lifecycle pointers are consumed. |
| REQ-F-012 complete result shape | AC-F03-001, AC-F03-002, AC-F03-009 | TC-001, TC-002, TC-003, TC-009, TC-012 | I-03 shared contract is canonical TC-003; TC-012 adds handoff regression coverage. |
| REQ-F-013 idempotent finalization | AC-F03-010 | TC-010 | Replays and interruption are both covered. |
| REQ-NFR-001..006 | AC-F03-002, 006, 007, 011, 012 | TC-002, TC-006, TC-007, TC-011, TC-013 | Supporting reliability, security, and bounded-read cases. |

No PRD completeness blocker was found: every architecture design element in
`spec.md` has an AC and an implementing path. There are no sibling tasks to
leave uncovered.

## Acceptance-criteria review and techniques

All ACs are concrete and testable. None is an open-ended robustness assertion:
the evidence security model is explicitly enumerated as prompts, credentials,
tokens, unrestricted output, and oversized values. The selected techniques
are:

| AC | Technique(s) | Test cases | Rationale |
|---|---|---|---|
| AC-F03-001 | Equivalence Partitioning, BVA | TC-001 | Required, eligible, and allowed-excluded item classes. |
| AC-F03-002 | Decision Table, Equivalence Partitioning | TC-002 | Each outcome class and precedence combination. |
| AC-F03-003 | Contract Surface Enumeration | TC-003 | Configured outcome route, custom extension, and pause boundary. |
| AC-F03-004 | State Transition, Equivalence Partitioning | TC-004 | Non-terminal, terminal, parking, missing, and absent-route states. |
| AC-F03-005 | Decision Table, Attack-class Enumeration | TC-005 | One active run, ambiguous runs, and missing ledger membership. |
| AC-F03-006 | Contract Surface Enumeration, Attack-class Enumeration | TC-006 | Child, edge, workflow, mode, and limit drift without mutation. |
| AC-F03-007 | State Transition, Decision Table | TC-007 | Terminal, stale, missing, live-conflict, and reissued claims. |
| AC-F03-008 | Decision Table, State Transition | TC-008 | Every dependency-success/non-success class and unrelated ready work. |
| AC-F03-009 | Contract Surface Enumeration, Attack-class Enumeration | TC-009 | Bounded council fields and sensitive-content exclusions. |
| AC-F03-010 | State Transition, Attack-class Enumeration | TC-010 | Replay after success, interruption, and conflicting terminal result. |
| AC-F03-011 | Attack-class Enumeration, BVA | TC-011 | Prompt/credential/token/output/path classes and size boundaries. |
| AC-F03-012 | Equivalence Partitioning, Contract Surface Enumeration | TC-013 | Team-run present versus absent through ordinary callers. |

## ISO 25010 coverage matrix

`N/A` cells are deliberate: this feature has no browser UX, cross-OS promise,
or independent throughput SLO beyond bounded reads and deterministic decisions.

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-F03-001 | ✅ TC-001 | N/A: no SLO | ✅ TC-012 | ✅ TC-001 | ✅ TC-010 | ✅ TC-011 | ✅ TC-014 | N/A: local service |
| AC-F03-002 | ✅ TC-002 | N/A: no SLO | ✅ TC-012 | ✅ TC-002 | ✅ TC-002 | N/A: no boundary | ✅ TC-014 | N/A |
| AC-F03-003 | ✅ TC-003 | N/A | ✅ TC-003 | ✅ TC-003 | ✅ TC-003 | N/A | ✅ TC-014 | N/A |
| AC-F03-004 | ✅ TC-004 | N/A | ✅ TC-004 | ✅ TC-004 | ✅ TC-004 | N/A | ✅ TC-014 | N/A |
| AC-F03-005 | ✅ TC-005 | ✅ TC-014 | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | ✅ TC-014 | N/A |
| AC-F03-006 | ✅ TC-006 | ✅ TC-014 | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 | ✅ TC-014 | N/A |
| AC-F03-007 | ✅ TC-007 | N/A | ✅ TC-007 | ✅ TC-007 | ✅ TC-007, TC-010 | ✅ TC-007 | ✅ TC-014 | N/A |
| AC-F03-008 | ✅ TC-008 | N/A | ✅ TC-008 | ✅ TC-008 | ✅ TC-008 | N/A | ✅ TC-014 | N/A |
| AC-F03-009 | ✅ TC-009 | N/A | ✅ TC-009 | ✅ TC-009 | ✅ TC-009 | ✅ TC-011 | ✅ TC-014 | N/A |
| AC-F03-010 | ✅ TC-010 | N/A | ✅ TC-010 | ✅ TC-010 | ✅ TC-010 | ✅ TC-010 | ✅ TC-014 | N/A |
| AC-F03-011 | ✅ TC-011 | ✅ TC-011 bounded evidence | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-014 | N/A |
| AC-F03-012 | ✅ TC-013 | N/A | ✅ TC-013 | ✅ TC-013 | ✅ TC-013 | N/A | ✅ TC-014 | N/A |

## Observability design

These are implementation requirements; fields must contain IDs, counts, safe
causes, and bounded pointers only.

| Behavior | Metric | Log | Trace span | Alert | Test assertion |
|---|---|---|---|---|---|
| Aggregate outcome and precedence | `team_aggregate_total{outcome}` | `team.aggregate.completed` with run/outcome/counts | `team.aggregate` | N/A | TC-001/002 |
| Root route resolution/boundary | `team_route_resolution_total{outcome,result}` | `team.route.resolved` or `.unavailable` | `team.route.resolve` | route-unavailable baseline | TC-003/004 |
| Resume drift/integrity rejection | `team_resume_rejected_total{cause}` | `team.resume.rejected` | `team.resume.verify` | spike review | TC-005/006 |
| Claim/session reconciliation | `team_resume_claim_conflicts_total{kind}` | `team.resume.claim_reconciled` | `team.resume.reconcile_claim` | live-conflict baseline | TC-007 |
| Dependency suppression | `team_resume_dependency_gates_total{reason}` | `team.resume.dependency_blocked` | `team.resume.gate_dependencies` | N/A | TC-008 |
| Council pause | `team_resume_paused_total{reason}` | `team.resume.paused` with bounded artifact pointer | `team.resume.council` | unresolved escalation baseline | TC-009 |
| Idempotent finalization | `team_finalize_total{result}` | `team.finalize.idempotent` | `team.finalize` | conflicting replay baseline | TC-010 |
| Sensitive evidence rejection | `team_evidence_rejected_total{class}` | `team.evidence.rejected` | `team.evidence.validate` | rejection baseline | TC-011 |
| Ordinary-run compatibility | `team_compatibility_total{team_run_present}` | `team.compatibility.checked` | `team.compatibility` | N/A | TC-013 |

## Integration scenarios

| Scenario | Boundaries | UAT contribution |
|---|---|---|
| Complete and mixed aggregation | F02 persisted item outcomes → F03 precedence → root workflow adapter | UAT-03, UAT-04 |
| Pause and unavailable route | F03 → configured E16/E35 workflow route or explicit next action | UAT-05, UAT-10 |
| Interrupted resume | F01 ledger + F02 claims/sessions + dependency snapshot | UAT-02, UAT-06 |
| Council escalation | F04 bounded artifact pointers → F03 paused result | UAT-05, UAT-11 |
| Operator result handoff | F03 `TeamRunResult` → F05 reporting shape | UAT-04, UAT-06, UAT-07 |

## Test infrastructure

Existing patterns to follow:

- `docs/plan/E38-shark-attack-team-orchestration/E38-F01-team-plan-and-durable-ledger/test-plan.md`: mock coordinator seams, real `teamrun` repository DB tests, bounded evidence fixtures, and I-01 contract shape.
- `docs/plan/E38-shark-attack-team-orchestration/E38-F02-scheduler-and-claims/test-plan.md`: persisted F02 outcome/session fixtures and no-force-steal claims.
- `docs/plan/E38-shark-attack-team-orchestration/E38-F04-shark-attack-skill-and-role-protocol/test-plan.md`: I-04 shared message/artifact fixtures.
- `internal/test/testdb.go` and existing `internal/repository/*_test.go`: cleaned real SQLite fixtures only for repository behavior.

New helpers needed: deterministic in-memory F01 run/item snapshot, workflow
route table with custom and missing outcomes, claim/session clock fixture,
bounded council-context fixture, and an observability sink. Keep these behind
the injected interfaces; do not add a second claim store or workflow engine.

## Cross-feature contract tests (I-##)

| I-## | Direction | Shape source | Canonical pointer | F03 test use |
|---|---|---|---|---|
| I-01 | F01 → F03 | E38 architecture §4.2 Team-run domain contract | `tests/contracts/e38_interactions_test.go#TC-001` | TC-001/005/006 consume complete immutable snapshot. |
| I-02 | F02 → F03 | E38 architecture §4.4 Aggregate outcome contract | `tests/contracts/e38_interactions_test.go#TC-002` | TC-002/008 consume persisted child outcomes and dependency state. |
| I-03 | F03 → F05 | E38 architecture §4.4 Aggregate outcome contract | `tests/contracts/e38_interactions_test.go#TC-003` | TC-003 asserts the route and the complete shared aggregate result shape. |
| I-04 | F04 → F03 | E38 architecture §4.5 Council communication contract | `tests/contracts/e38_f04_interactions_test.go#TC-001` | TC-009 consumes the canonical shared contract; TC-002 supplies supplementary durable artifact fixtures for bounded escalation, handoff, decision, resolution, and inbox pointers. Same tests; no twins. |

## Cross-epic integration tests (X-##)

| X-## | Boundary | Shape source | Coverage pointer | Test |
|---|---|---|---|---|
| X-02 | E16/E35 configured roles, semantic outcomes, pause/terminal boundaries, root routing | E38 architecture §4.1/§4.4; `docs/guides/route-based-workflow.md` | E38 `uat-plan.md` UAT-04, UAT-05, UAT-10 | TC-003/004. |

X-02 is not deferred in `docs/product/progress.md`; it is covered by the
configured-route and boundary cases above. X-01, X-03, X-04, and X-05 are
owned by other E38 features and are not duplicated here.

## Caller-Path Contracts

| TC | Production entrypoint and argument shape | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `Coordinator.Aggregate(ctx, runID=41)` | F01 ledger snapshot and F02 outcome readers | Do not call a precedence helper or infer from child counts | Excluded/terminal items disappear and the result incorrectly completes. |
| TC-002 | `Coordinator.Aggregate(ctx, runID=41)` | Persisted item-outcome reader | Do not hand-build aggregate input or mock precedence | Provider failure/partial outcome is collapsed into success. |
| TC-003 | `Coordinator.Finalize(ctx, runID=41, result)` | `workflow.Service` route resolver | Do not mock `Release`, select route by index, or mock root adapter | Custom route is ignored and a hard-coded `completed` target is used. |
| TC-004 | `Coordinator.Finalize(ctx, runID=41, result)` | Typed root transition adapter | Do not mock terminal/parking classification or force transition | Missing route causes a guessed transition instead of durable `routing_unavailable`. |
| TC-005 | `Coordinator.Resume(ctx, rootKey="E38-F03-fixture")` | F01 active-run/ledger repository | Do not bypass active-run selection or mock integrity result | Ambiguous runs or missing membership mutate claims or dispatch. |
| TC-006 | `Coordinator.Resume(ctx, rootKey="E38-F03-fixture")` | Canonical plan reader and ledger CAS seam | Do not merge changed snapshots or mock fingerprint comparison | Plan drift is silently accepted and changed work dispatches. |
| TC-007 | `Coordinator.Resume(ctx, rootKey="E38-F03-fixture")` | ClaimService `Get`/`IsClaimable` and session-scoped repository seam | Do not call force-steal, unscoped release, or a private claim classifier | A live claim is stolen or a completed item is redispatched. |
| TC-008 | `Coordinator.Resume(ctx, rootKey="E38-F03-fixture")` | Persisted dependency/item reader | Do not test only graph helpers or mock dependency success | A dependent dispatches after a failed prerequisite. |
| TC-009 | `Coordinator.Aggregate(ctx, runID=41)` | Bounded `CouncilContextReader` | Do not provide unrestricted prompt/output fixtures or choose a human destination | Escalation is ignored or sensitive council content is persisted. |
| TC-010 | `Coordinator.Finalize(ctx, runID=41, result)` repeated with same result | Ledger CAS/history seam and typed root adapter | Do not reset the ledger or assert only returned DTO equality | Replay creates a second transition/history/pointer. |
| TC-011 | `Coordinator.Finalize(ctx, runID=41, result)` with concrete evidence values | Central evidence validator immediately before persistence | Do not pre-sanitize fixtures or test only a private validator | Prompt, credential, transcript, or oversized output reaches storage/logs. |
| TC-012 | `Coordinator.Aggregate(ctx, runID=41)` returning `TeamRunResult` | Shared contract assertion at `tests/contracts/e38_interactions_test.go#TC-003` | Do not hand-build a F05 DTO or test only JSON formatting | A required item/session/count/next-action field is omitted. |
| TC-013 | Production ordinary callers `run`/`next` with no team run | Existing service/dispatcher seam | Do not call only a compatibility helper or inject a fake team result | Ordinary execution gains team reconciliation or root side effects. |
| TC-014 | `Coordinator.Resume(ctx, rootKey="E38-F03-fixture")` with item count N | Ledger query/repository boundary | Do not benchmark a private loop or rescan unrelated roots | Resume scans unrelated roots or exceeds persisted item-bounded work. |

## Acceptance test cases

Each case below is a specification, not implementation code. Every case has a
Caller-Path Contract above and must include at least one explicit negative
assertion.

### TC-001: All required items pass and allowed exclusions complete

**AC:** AC-F03-001; REQ-F-001/002. **Technique:** Equivalence Partitioning +
BVA. **ISO:** Functional Suitability, Reliability, Compatibility.

**Setup/input:** Run 41 contains required eligible children A and B with
persisted `completed` outcomes, one plan-allowed excluded child X, complete F01
membership, and configured `pass -> qa` route.

**Expected:** `Aggregate` returns `completed`, reports A/B/X exactly once,
resolves `pass -> qa`, invokes one root transition with run ID and outcome in a
bounded reason, and includes counts, sessions, evidence refs, and next action.

**Edges/negative:** zero eligible items is `partial` or the documented plan
result, not success; a disallowed skipped item is not success; no second root
transition or inferred count-only completion.

### TC-002: Mixed outcomes obey deterministic precedence

**AC:** AC-F03-002; REQ-F-002/003. **Technique:** Decision Table +
Equivalence Partitioning. **ISO:** Functional Suitability, Reliability.

**Setup/input:** Table-drive runs containing combinations of `partial`,
`failed`, `blocked`, `paused`, `cancelled`, `skipped`, `provider_unavailable`,
and `refresh_required`, with required/optional flags and dependency reasons.

**Expected:** For each row, return the specified highest-precedence semantic
outcome and preserve every item diagnostic; only all required eligible
configured-success items return `completed`.

**Edges/negative:** provider-unavailable never becomes success; allowed skipped
work may complete, disallowed skipped work is partial; counts alone cannot
override a required failure or refresh condition.

### TC-003: Configured custom outcome and pause route are honored

**AC:** AC-F03-003; REQ-F-004; X-02. **Technique:** Contract Surface
Enumeration. **ISO:** Functional Suitability, Compatibility, Usability.

**Setup/input:** Root step `team_review` has custom `quality_hold -> qa_review`
and `pass -> approved`; inject a paused aggregate and a completed aggregate.

**Expected:** `workflow.Service.Release(fromStatus, outcome)` resolves the exact
configured route and the typed adapter receives that target once; paused
boundary is returned when configured. The shared I-03 contract at
`tests/contracts/e38_interactions_test.go#TC-003` also asserts root identity,
mode/limit, aggregate outcome, target/boundary, transition result, complete
item diagnostics, counts, sessions, evidence references, and next action.

**Negative:** positional outcome order, hard-coded `completed`, direct entity
table update, or unconfigured target must fail the assertions.

### TC-004: Missing or terminal route stops safely

**AC:** AC-F03-004/005; REQ-F-005; X-02. **Technique:** State Transition +
Equivalence Partitioning. **ISO:** Functional Suitability, Reliability,
Usability.

**Setup/input:** Exercise terminal, parking, missing-root-step, and absent-
selected-outcome route states.

**Expected:** Aggregate persists durably, performs zero root transitions, and
returns `routing_unavailable` or configured `paused` with exact missing-route or
review next action.

**Negative:** never guess a neighboring route or force a terminal transition.

### TC-005: Ambiguous active run and incomplete ledger are rejected

**AC:** AC-F03-005; REQ-F-001/006. **Technique:** Decision Table + Attack-class
Enumeration. **ISO:** Functional Suitability, Reliability, Security.

**Setup/input:** Root `E38-F03-fixture` has two active run IDs; separately,
run 41 omits planned child B from membership while current entity B exists.

**Expected:** Return typed ambiguity/integrity errors containing the root/run
context; perform no claim, dispatch, release, transition, or snapshot mutation.

**Negative:** current entity status/count must not repair the ledger implicitly.

### TC-006: Resume rejects every material plan drift class

**AC:** AC-F03-006/007; REQ-F-007. **Technique:** Contract Surface
Enumeration + Attack-class Enumeration. **ISO:** Functional Suitability,
Reliability, Security, Performance Efficiency.

**Setup/input:** Persist hash A; independently change child membership,
dependency edge, workflow metadata, execution mode, and limit before resume.

**Expected:** Each case returns `refresh_required`, preserves hash A and the old
snapshot byte-for-byte, and performs no dispatch or merge.

**Negative:** volatile ordering changes that canonicalize identically must not
produce false drift; no unrelated-root rescan.

### TC-007: Resume reconciles terminal items and claim sessions non-destructively

**AC:** AC-F03-007; REQ-F-008/009. **Technique:** State Transition + Decision
Table. **ISO:** Functional Suitability, Reliability, Security.

**Setup/input:** Run 41 contains terminal A, expired B, missing-lease C, live
claim session `other-session` on D, and explicit reissued attempt E.

**Expected:** A is not redispatched and retains evidence/history; B/C get
visible stale-attempt diagnostics; D gets `claim_conflict` without stealing or
unscoped release; E is a new attempt and cannot overwrite its prior terminal
attempt.

**Negative:** no force-steal, no release without exact session, no terminal
result overwrite.

### TC-008: Dependencies gate only affected unfinished work

**AC:** AC-F03-008; REQ-F-010. **Technique:** Decision Table + State Transition.
**ISO:** Functional Suitability, Usability, Reliability.

**Setup/input:** A depends on B; vary B through failed, blocked, paused,
cancelled, unsatisfied, and configured-success outcomes; add unrelated ready C.

**Expected:** A gets durable exact blocked/skipped reason and is not dispatched
for every non-success B class; C remains selectable; returned aggregate includes
both diagnostics.

**Negative:** A must not dispatch after only an unrelated prerequisite succeeds.

### TC-009: Council escalation pauses with bounded I-04 context

**AC:** AC-F03-009/011; REQ-F-011; I-04. **Technique:** Contract Surface
Enumeration + Attack-class Enumeration. **ISO:** Functional Suitability,
Security, Usability.

**Setup/input:** Supply the shared I-04 message/artifact fixtures at
`tests/contracts/e38_f04_interactions_test.go#TC-001`, using the canonical shared
contract and its supplementary TC-002 lifecycle fixtures, with unresolved
escalation, review status, safe artifact path, and next action.

**Expected:** Result is `paused`, preserves only bounded pointer/status/roles/
next-action metadata, and names council/review action without choosing a human.

**Negative/edges:** resolved escalation does not pause; prompt, credential,
token, transcript, and traversal path are rejected or reduced.

### TC-010: Finalization is idempotent across retry and interruption

**AC:** AC-F03-010; REQ-F-013. **Technique:** State Transition + Attack-class
Enumeration. **ISO:** Functional Suitability, Reliability, Security.

**Setup/input:** Finalize run 41, replay the same terminal result after success,
and simulate interruption after persistence but before transition completion.

**Expected:** Every replay returns the same terminal result; root transition,
history, item result, and council pointer counts remain one; conflicting result
returns a typed conflict and cannot change terminal outcome.

**Negative:** no duplicate side effect and no terminal-to-different-outcome
transition.

### TC-011: Sensitive and oversized evidence is rejected or bounded

**AC:** AC-F03-011; REQ-NFR-004. **Technique:** Attack-class Enumeration + BVA.
**ISO:** Functional Suitability, Security, Reliability.

**Setup/input:** Submit rendered prompt, `Bearer` credential, token, unrestricted
stdout/stderr, path traversal, empty, max-boundary, and max+1 evidence values.

**Expected:** Unsafe values are rejected or replaced by bounded safe summaries;
ledger, result, logs, and telemetry contain no sensitive content.

**Negative:** no test fixture may pre-sanitize before the production entrypoint.

### TC-012: Aggregate result remains complete at the F05 handoff

**AC:** AC-F03-001/002/009; REQ-F-012; I-03. **Technique:** Contract Surface
Enumeration. **ISO:** Functional Suitability, Compatibility, Maintainability.

**Setup/input:** Use the same aggregate fixture as the shared
`tests/contracts/e38_interactions_test.go#TC-003` contract, containing root
identity, mode, limit, mixed items, sessions, counts, target/boundary,
transition result, evidence refs, and next action.

**Expected:** The shared contract asserts every documented §4.4 field, stable
item ordering, bounded values, and no F05 formatting dependency.

**Negative:** missing any required field or leaking prompt/secret fails.

### TC-013: Ordinary execution remains unchanged without a team run

**AC:** AC-F03-012; REQ-F-012. **Technique:** Equivalence Partitioning +
Contract Surface Enumeration. **ISO:** Functional Suitability, Compatibility,
Reliability.

**Setup/input:** Invoke existing production `shark next` and `shark run` paths
for a root with no team run, using their normal service/dispatcher wiring.

**Expected:** Existing result/status/claim behavior is unchanged; F03 performs
no team reconciliation, aggregate persistence, or root transition.

**Negative:** no recursive CLI invocation, team-only claim mutation, or altered
ordinary `/run` semantics.

### TC-014: Resume work is bounded and deterministic

**AC:** REQ-NFR-001/006. **Technique:** BVA + Contract Surface Enumeration.
**ISO:** Performance Efficiency, Maintainability, Functional Suitability.

**Setup/input:** Resume runs with 0, 1, and N persisted items; shuffle storage
order and include unrelated roots.

**Expected:** Work is bounded by the selected run's item count, diagnostics are
ordered by wave, execution order, priority, canonical key, and equivalent
snapshots produce equivalent results.

**Negative:** no unrelated-root scan or unstable map iteration output.

## Codex test-plan red-team

**Verdict:** CONCERNS (tool invocation unavailable in this worker context)
**Issues raised:** 1
**Issues addressed before dev:** 1
**Issues deferred:** 0

The workflow supplied a `codex_command` input contract but no concrete command
or prompt invocation. The available `codex` binary was detected, but invoking
an unspecified external command would not be reproducible. As a bounded
fallback, this plan performed the required red-team checks manually: every AC
has a named technique, enumerated partitions/boundaries, negative coverage,
ISO row, observability evidence, and caller-path contract; I-## and X-02
pointers are exact. No open-ended robustness AC remains.

## Verdict and recommendations

- [x] Ready for development: all 12 ACs have coverage, techniques, ISO entries,
  observability, negative/edge cases, and caller-path contracts.
- [x] I-01, I-02, I-03, and I-04 shared contract pointers are covered without
  twin tests.
- [x] X-02 is covered by TC-003/TC-004 and UAT-04/UAT-05/UAT-10.
- [x] No unresolved spec drift or PRD completeness gap.

**Recommended outcome:** APPROVED; parent loop may advance the feature's test-
planning gate.
