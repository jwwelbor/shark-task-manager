# Test plan: E39-F04 — Focused Question read surfaces and consumer handoff

**Created:** 2026-07-31  
**Feature specification:** spec.md  
**Epic UAT:** ../uat-plan.md  
**Status:** APPROVED

## Goal and scope

Verify the two bounded focused reads and the distinct authorized-full read
without weakening the existing compact projection. Verify common CLI/HTTP
service wiring, read-only behavior, and X-06's producer-only public contract.

F04 consumes live I-01, I-02, and I-03. A live Shark read during planning
found E39-F01 through E39-F03 completed and F04 in test planning. It produces
X-06 producer v1; E38-F09 remains its blocked activation owner. F04 adds no
migration, index, queue, cache, telemetry, viewer redesign, or E38 adapter.

## Spec drift and traceability

The specification, research report, E39 UAT plan, interaction map, and global
cross-epic map agree on named read surfaces, safe/full separation, and
producer-only X-06 ownership. X-06 is assigned, not deferred: E38-F09 must add
consumer coverage when it resumes. No unresolved drift was found.

| Requirement | AC | Test cases | Coverage |
|---|---|---|---|
| REQ-F-001, NF-001/002 | AC-001/004/005 | TC-401, TC-402 | Derived responder partition, finite validation, no writes. |
| REQ-F-002, NF-001/002 | AC-002/004/005 | TC-403, TC-404 | I-03 predicate reuse, deterministic edge order, no writes. |
| REQ-F-003, NF-001 | AC-003/004/005 | TC-405, TC-406 | Full policy and compact compatibility. |
| REQ-F-004, NF-003 | AC-006 | TC-004, TC-407 | Producer handoff and forbidden manifest. |
| All F04 reads | AC-005 | TC-408 | Temporary SQLite durable snapshots. |

### AC quality review

The scope words “only”, “every”, and “unchanged” are closed by the explicit
state, edge, identity, transport, and forbidden-field partitions below. Actor
authorization is an exact policy-seam comparison, not an authentication or
durable-identity design.

## AC test matrix

| AC | Technique | Input/setup | Expected outcome and edges |
|---|---|---|---|
| AC-001 | Equivalence partitioning + BVA | Open, answering, ready, terminal, malformed, unconfigured states; limits 1/50/100, offsets 0/1 | Exact current responder gets only open/answering records in canonical key order; unmatched page is empty array. |
| AC-002 | Decision table + deterministic ordering | Direct/generic/indirect/false-blocking/closed/malformed edges and multiple direct edges | Only F03-qualified direct sources, ordered by edge created_at then ID, with I-03 fields. |
| AC-003 | Attack-class enumeration + state transition | Base reads, focused reads, full reads as requester/unrelated/responder/owner | Ordinary and focused output stays compact; only responder or owner receives bounded full fields. |
| AC-004 | Contract-surface enumeration + BVA | Blank actor/identity, bad target, Question target, unknown keys, limit 0/101, offset -1 | Established actionable rejection before service mutation; base reads unchanged. |
| AC-005 | Atomicity enumeration | Temporary SQLite with Questions, edges, claims, histories, notes, work sessions | Every successful/failed read preserves serialized before/after durable snapshots. |
| AC-006 | Cross-feature contract enumeration | Real I-01 identity, I-02 state, I-03 relation, public surfaces | TC-001–003 remain compatible; TC-004 proves X-06 without E38 implementation. |

## Test infrastructure

| Layer | Existing pattern | Planned use |
|---|---|---|
| Models/service | internal/models/question_test.go; internal/services/question_service_test.go | Mock dependencies below service; test validation, projection, policy, ordering, no write calls. |
| Repository/blocker | internal/repository/question/repository.go; internal/services/question_blocker.go | Isolated SQLite repository tests and F03 direct predicate reuse. |
| CLI | internal/cli/commands/question.go and question_test.go | Cobra production commands with mocked service seam. |
| HTTP/viewer | internal/api/question_handler.go; handler tests; internal/viewer/assets_test.go | Real QuestionHandler.RegisterRoutes mux; no browser/npm harness. |
| Contracts | tests/contracts/e39_interactions_test.go | Isolated temporary SQLite, public composition, I-01–03/X-06 evidence. |

Repository/contract tests use isolated real SQLite. CLI, handler, and service
tests mock below their production entrypoint and never use the repository test
database.

## Caller-path contracts

| TC | Production entrypoint and argument shape | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-401 | Cobra question open-by-responder alice --limit 50 --offset 0 and matching GET responder route | ListOpenQuestionsByResponder dependencies | Command/handler allowlists, service decode, projection | Transport returns completed/malformed state or accepts bad identity. |
| TC-402 | QuestionService.ListOpenQuestionsByResponder(ctx, responder, limit, offset) | Bounded Question candidate reader | CurrentResponder, status filter, ordering/projection | Shortcut infers responder from claim/response data or leaks state. |
| TC-403 | Cobra question blocking-for F001 --limit 50 --offset 0 and matching GET route | ListQuestionsBlocking dependencies | Target validation, F03 qualifier, I-03 projection | Route accepts Question target or returns generic/indirect edge. |
| TC-404 | QuestionService.ListQuestionsBlocking(ctx, targetType, targetKey, limit, offset) | Incoming relation/Question reader | Directness/status/blocking predicate, sorter | F04 disagrees with QuestionBlocker.Check or leaks relationship IDs. |
| TC-405 | Cobra question full Q001 --actor alice and GET /api/v1/questions/Q001/full?actor=alice | ReadQuestionFull dependencies | Actor policy, state decode, full projection | Raw context leaks or unrelated actor is authorized. |
| TC-406 | Existing get/list/search/viewer and QuestionHandler.GetQuestion/ListQuestions | Existing service seam | Compact projector and generic route | Full read changes baseline output or shadows generic route. |
| TC-408 | Public CLI/API sequences over temporary SQLite | Prompt/worker only when irrelevant | Read service, registry, state/edge predicate, snapshot reader | Read writes state, relationship, claim, history, note, work session, or search data. |
| TC-004 | tests/contracts/e39_interactions_test.go#TC-004 real public composition | No mock above composition | I-01 registry, I-02 lifecycle, I-03 predicate, projections | Producer supplies queue, mutable copy, hidden fields, or linked-work authority. |
| TC-407 | Source/configuration scan and public probes | None for absence checks | Router/command registrations | F04 adds base full switch, generic filters, mutation, queue, adapter, migration, or raw transport. |

## ISO 25010 coverage and observability

N/A means not applicable to this bounded Go/SQLite read extension. Existing
result/error contracts and durable snapshots are runtime evidence. New
telemetry is forbidden; each behavior is intentionally internal — no new
observability.

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-401/402 | ✅ TC-402 bounded page | ✅ TC-401 | ✅ TC-401 errors | ✅ TC-402 | ✅ TC-402 compact | ✅ TC-402 | N/A local Go/SQLite |
| AC-002 | ✅ TC-403/404 | ✅ TC-404 direct bounded read | ✅ TC-404 I-03 | ✅ TC-403 | ✅ TC-404 order | ✅ TC-404 compact | ✅ TC-404 | N/A |
| AC-003 | ✅ TC-405/406 | N/A bounded fields | ✅ TC-406 bytes | ✅ TC-405 denial | ✅ TC-405 | ✅ TC-405/406 | ✅ TC-405 | N/A |
| AC-004 | ✅ TC-401/403/405 | N/A pre-service rejection | ✅ TC-406 | ✅ TC-401/403 | ✅ TC-408 snapshots | ✅ TC-408 | ✅ TC-407 | N/A |
| AC-005 | ✅ TC-408 | N/A | ✅ TC-408 | N/A internal | ✅ TC-408 | ✅ TC-408 | ✅ TC-408 | N/A |
| AC-006 | ✅ TC-004/407 | N/A | ✅ TC-004 | ✅ TC-004 | ✅ TC-004 | ✅ TC-004/407 | ✅ TC-407 | N/A |

| Behavior | Metric/log/trace | Test assertion |
|---|---|---|
| Focused reads | Internal — no new telemetry; existing result/error only | TC-401–404 assert projection/error and bounded seams. |
| Full read | Internal — do not log disclosed material | TC-405 asserts authorization and raw-data absence. |
| Read-only surface | Internal durable snapshots | TC-408 compares all relevant rows before/after. |
| X-06 producer handoff | Contract result only; no host queue telemetry | TC-004 and TC-407 prove compact handoff and absent queue/copy. |

## Integration scenarios

| Interaction | Test and boundary | Gate and closure evidence |
|---|---|---|
| I-01 | tests/contracts/e39_interactions_test.go#TC-001; TC-004 registered Q001 | Live; focused reads consume canonical record without altering base projection. |
| I-02 | TC-002; TC-401/405 validated responder and owner | Live; malformed state is actionable error, not guessed identity. |
| I-03 | TC-003; TC-403/404 exact blocker predicate | Live; four-field direct compact handoff remains identical. |
| UAT-01/03/05 | Temporary SQLite: two responders, owner, direct target, unlinked control | Capture compact pages, blocker handoff, allowed/denied full, redacted snapshots. |
| X-06 | TC-004 | Assigned live producer coverage. E38-F09 remains blocked and owns later consumer coverage. |

## Acceptance test cases

### TC-401: Return only the exact current responder's compact open page

**Feature requirement:** REQ-F-001, REQ-NF-001/002.  
**Acceptance criterion:** AC-001, AC-004.  
**Technique:** Equivalence partitioning and BVA.  
**ISO 25010:** Functional suitability, compatibility, usability, security.

Create Q001/Q002 open or answering with alice current, Q003 with bob, and
ready, terminal, malformed, and unconfigured controls. Drive CLI and HTTP as
alice with limits 1/50/100 and offsets 0/1. Expect Q001/Q002 in canonical key
order, compact fields only, and empty array for no match. Reject blank identity,
unknown keys, limit 0/101, and offset -1 before service invocation. Negative:
completed responders, owner, claim holder, response text, or invalid state
never select a result.

### TC-402: Derive responder from validated serial state without leakage

**Feature requirement:** REQ-F-001, REQ-NF-001/002.  
**Acceptance criterion:** AC-001, AC-004.  
**Technique:** Equivalence partitioning and contract-surface enumeration.  
**ISO 25010:** Functional suitability, performance, reliability, security.

Drive the service with bounded fixtures for first-pending, completed-responder,
invalid JSON, and unconfigured state. Assert only open/answering validated
first-pending matches, bounded candidate read, no writes, and actionable
malformed-state error. Negative: compact JSON has no context, response,
evidence, claim, session, prompt, or credential.

### TC-403: Return only direct F03-qualified blockers in deterministic order

**Feature requirement:** REQ-F-002, REQ-NF-001/002.  
**Acceptance criterion:** AC-002, AC-004.  
**Technique:** Decision table and deterministic-order BVA.  
**ISO 25010:** Functional suitability, compatibility, usability, reliability, security.

For F001 create qualifying Q002/Q001 direct edges with equal timestamps and IDs
8/7, later Q003, plus generic, indirect, false-blocking, closed, malformed,
and unlinked controls. Both public surfaces return Q001/Q002/Q003 in created-at
then ID order, with exactly I-03's four fields. Reject Question target, bad
pagination, and unknown keys before service work. Negative: no relationship ID
or excluded edge appears.

### TC-404: Reuse exact F03 qualification at the focused-read seam

**Feature requirement:** REQ-F-002, REQ-NF-001/002.  
**Acceptance criterion:** AC-002, AC-005.  
**Technique:** Decision table and contract-surface enumeration.  
**ISO 25010:** Functional suitability, performance, reliability, security.

Compare ListQuestionsBlocking with QuestionBlocker.Check across direct state,
blocking flag, and relationship-kind partitions. Both use the same incoming
typed edge and validated-state rules; F04 returns an ordered qualifying page
where F03 selects first handoff. Assert bounded reads, no graph traversal or
writes, and unchanged snapshots. Negative: generic blocks and indirect
Q001-to-T001-to-F001 do not qualify.

### TC-405: Restrict full projection to assigned identities

**Feature requirement:** REQ-F-003, REQ-NF-001.  
**Acceptance criterion:** AC-003, AC-004/005.  
**Technique:** Decision table, attack-class enumeration, state transition.  
**ISO 25010:** Functional suitability, reliability, security, maintainability.

For Q001 with responders alice/bob and owner owner, request full as current
alice, owner, prior bob, requester, unrelated mallory, blank actor, and against
invalid/unconfigured state. Only alice and owner receive compact fields plus
ordered responders, bounded responses, owner, and optional resolution
kind/pointer. Others get typed access/validation errors with identical snapshots.
Negative: no raw ContextData, claims, relationship IDs, prompts, credentials,
or unbounded response set is serialized.

### TC-406: Preserve existing compact reads byte-for-byte

**Feature requirement:** REQ-F-003, REQ-NF-001.  
**Acceptance criterion:** AC-003, AC-004.  
**Technique:** Contract-surface enumeration.  
**ISO 25010:** Functional suitability, compatibility, security.

Capture base JSON for question get/list, generic search/viewer, and base HTTP
get/list. Register F04 routes and repeat. Expect identical compact output and
errors. Probe /Q001/full and /Q001 to prove route precedence without generic
shadowing. Negative: base get/list reject full switch; generic responder/target
list filters remain absent or rejected.

### TC-408: Prove every F04 read is durable-state read-only

**Feature requirement:** REQ-F-001 through REQ-F-003, REQ-NF-001/002.  
**Acceptance criterion:** AC-005.  
**Technique:** Atomicity enumeration.  
**ISO 25010:** Functional suitability, reliability, security.

On temporary SQLite snapshot every Question/state, relationship, claim, history,
note, work-session, and search row. Run successful and rejected TC-401/403/405
cases. Compare ordered serialized snapshots before/after, not counts alone.
Assert no queue, mutable copy, migration, index, cache, background job, or
telemetry payload. Negative: denial creates no last-access, claim, audit/history,
or linked-work write.

### TC-004: Prove X-06 producer v1 through public Question surfaces

**Feature requirement:** REQ-F-004, REQ-NF-003.  
**Acceptance criterion:** AC-006; **Interactions:** I-01, I-02, I-03, X-06.  
**Technique:** Cross-feature contract-surface enumeration.  
**ISO 25010:** Functional suitability, compatibility, reliability, security.

In tests/contracts/e39_interactions_test.go#TC-004 compose registered Q001,
serial shark next Q001 --json, I-03 handoff, both focused reads, and allowed
full read using temporary SQLite. Assert consumer receives only public
keys/safe projections and can use public APIs to inspect/resume Question work.
Negative: no queue, adapter, mutable copy, transcript, prompt, or authority to
claim/advance/resolve linked work; do not start or modify E38-F09.

### TC-407: Enforce f04-forbidden-v1

**Feature requirement:** REQ-F-004, REQ-NF-001 through REQ-NF-003.  
**Acceptance criterion:** AC-006.  
**Technique:** Source/configuration enumeration and black-box probes.  
**ISO 25010:** Functional suitability, security, maintainability.

Scan F04-owned transports/projections/registration and probe CLI/API for raw
Question JSON, compact forbidden fields, base full switch, generic filters,
focused-read mutation, queue/adapter, recursive traversal, migration, index,
cache, and telemetry. Expect absence or actionable rejection and unchanged
snapshots.

## Codex test-plan red-team

**Verdict:** PASS  
**Issues raised:** 2  
**Issues addressed before development:** 2  
**Issues deferred:** 0

The parent dispatch contract forbids external AI CLI delegation and supplied no
codex command; this worker performed the required class-based review locally.
It found and resolved: count-only no-write evidence could miss in-place
mutation, so TC-408 requires serialized snapshots; generic full-read testing
could miss route shadowing/base compatibility, so TC-406 tests explicit route
precedence and byte-for-byte base output. Every AC has a closed input/attack
model, technique, ISO coverage, caller-path contract, negative case, and
runtime evidence plan.

## Recommendations

- [x] Ready for development after red-team PASS
- [ ] Needs BA refinement
- [ ] Needs technical refinement
