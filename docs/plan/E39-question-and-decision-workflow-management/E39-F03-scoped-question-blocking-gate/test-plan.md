# Test plan: E39-F03 — Scoped Question blocking gate

**Created:** 2026-07-31  
**Feature specification:** `spec.md`  
**Epic UAT:** `../uat-plan.md`  
**Status:** APPROVED

## Goal and scope

Verify that an open, blocking Question stops only a directly linked eligible
entity through `question_blocks`. Verify that the check is read-only, returns
only the I-03 compact handoff, and is identical at keyed `next`, `run`, cascade,
and supported `status advance` boundaries.

The approved F03 scope includes one additive SQLite migration that widens the
existing relationship vocabulary for `question_blocks`. Verify upgrade from a
pre-migration database, preserving relationship rows, indexes, dependent views,
and the Question cleanup trigger.

F03 consumes live I-01 and I-02. A live Shark read on 2026-07-31 found
E39-F01 and E39-F02 completed and E39-F04 draft. F03 produces I-03 for F04;
it does not implement X-06, F04 read routes, or an E38 adapter.

## Spec drift and traceability

The owner approved the relationship-vocabulary migration on 2026-07-31; this
plan and `spec.md` record its bounded upgrade coverage. No other unresolved
drift was found among the E39 UAT plan, the interaction map, the cross-epic
map, and the research report. Task specifications now decompose the approved
feature-level contract; this plan remains their acceptance source. The
specification declares no X-## interaction for F03. X-06 remains assigned to
E39-F04 and E38-F09 in `docs/product/cross-epic-integration-map.md`.

| Feature requirement | AC | Test cases | Coverage note |
|---|---|---|---|
| REQ-F-001, NF-001/003 | AC-001 | TC-301, TC-302 | Directed finite vocabulary, upgrade preservation, and atomic rejected writes. |
| REQ-F-002, NF-001/002/003 | AC-002, AC-003 | TC-303, TC-304 | Predicate partitions, deterministic winner, bounded reads, no writes. |
| REQ-F-003, NF-001/002/003 | AC-004 | TC-305, TC-306 | Root/cascade keyed-next ordering and compatible no-match envelope. |
| REQ-F-004, NF-002/003 | AC-005 | TC-307, TC-308 | Direct/cascade normal and dry-run run parity before lease/worker work. |
| REQ-F-005, NF-001/002/003 | AC-006 | TC-309, TC-310 | Supported advance rejection, exact JSON/human error boundary, closure retry. |
| REQ-F-006, NF-001/003 | AC-007, AC-008 | TC-311, TC-003, TC-312 | Excluded behavior remains unchanged; I-03 and forbidden manifest. |

### AC quality review

All ACs identify their candidate partition, observable result, and non-mutation
expectation. The bounded-read requirement is testable through repository call
counts and source scans, not a time threshold. “Every” in AC-001, AC-004,
AC-005, AC-006, and AC-007 means the explicitly enumerated types and paths in
the cases below; it does not create an open-ended robustness obligation.

## AC test matrix

| AC | Technique | Input/setup | Expected outcome and edges |
|---|---|---|---|
| AC-001 | Equivalence partitioning + decision table | `Q001 -> F001` and `Q001 -> T001` plus each eligible `epic`, `feature`, `task`, `bug`, `change`, `tech_debt`; reverse edge; Question/Sprint/Idea target; duplicate; unlink | Only Question-to-eligible rows persist. Invalid and duplicate creates change no rows. Unlink removes only the selected valid edge. Existing relationship types retain their current behavior. |
| AC-002 | Decision table + contract-surface enumeration | Direct `question_blocks` sources partitioned by `blocking` true/false and state `draft`, `open`, `answering`, `ready_for_resolution`, `resolved`, `withdrawn`, `superseded`; generic `blocks`, indirect chain, unlinked, claimed, completed-responder controls | Only direct `open`/`answering` plus `blocking=true` qualifies. All controls return no handoff and preserve snapshots. |
| AC-003 | Boundary value analysis + deterministic ordering | Two/three qualifying Question edges with earlier/later `created_at`; equal timestamps and relationship IDs 7/8; absent or invalid persisted F02 state | Earliest timestamp then lowest ID wins. Handoff contains exactly `question_key`, `summary`, `resolution_owner`, `current_responder`. An absent/invalid state returns an actionable error, not a qualifying handoff. |
| AC-004 | State transition + decision table | Root and cascade parent with a qualifying direct child, blocked child plus unlinked eligible sibling, and no-match controls | `next` returns pause before placeholders/action/prompt/lease. A blocked child falls through to the sibling; the parent does not transition. No match retains the prior JSON envelope without `question_block`. |
| AC-005 | State transition + decision table | Qualifying direct root and qualifying cascade child under `run`, each in normal and `--dry-run`; an unblocked Question responder control | Both modes return normal paused result, same compact handoff, and zero worker/lease/heartbeat/release/transition writes. No-match uses F02 action-before-lease behavior. |
| AC-006 | State transition + atomicity enumeration | One qualifying linked candidate of each supported non-Question type; resolve source Question and retry; Question self-transition control | First advance returns typed compact blocked error and preserves all tracked rows. After resolution, normal advance succeeds. Question transitions remain available. |
| AC-007 | Contract-surface enumeration | Generic `blocks`, unrelated candidate, direct service call, Question lifecycle operations, and an installed gate edge | Only the directly linked supported CLI boundary changes. Generic planning/relationship semantics, F01/F02 lifecycle, and unrelated work remain unchanged. |
| AC-008 | Source/configuration enumeration + contract test | I-01/I-02 fixtures, I-03 handoff, exact JSON, and `f03-forbidden-v1` paths | TC-001 and TC-002 remain live-compatible; TC-003 proves I-03. Forbidden reads/disclosure/adapter/queue/mutation surfaces are absent or reject without writes. |

## Test infrastructure

| Layer | Existing pattern | Planned use |
|---|---|---|
| Model and relationship service | `internal/models/question_test.go`; `internal/services/entity_relationship_service_test.go` | Finite type inventory and direction/target matrix with mocked service dependencies. |
| Question blocker service | `internal/services/question_service_test.go`; F01 `QuestionRepositoryAdapter` | Mock incoming-edge, registry, and Question seams; assert one candidate lookup, one filtered incoming query, bounded source reads, and no write seam. |
| Repository and contract | `tests/contracts/e39_interactions_test.go`; `repository.NewQuestionRepository`; `db.InitDB` | Real temporary SQLite for I-03, edge ordering, durable snapshots, and public composition. |
| CLI `next` and `run` | `internal/cli/commands/next_normalize_test.go`; `internal/cli/commands/run_test.go` | Cobra/command production entrypoints with mocks at the prompt/action/runner seams only. |
| CLI status and viewer mutation | `internal/cli/commands/question_test.go`; `internal/api/question_handler_test.go`; existing relationship command tests | Supported transition routing, typed error formatting, and generic relationship mutation parity. |

Repository and contract tests use a real isolated SQLite database. CLI and
service tests use mocks below their declared entrypoint; they must not use the
repository test database.

## Caller-path contracts

| TC | Production entrypoint and argument shape | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-301 | `EntityRelationshipService.Create(ctx, Question, Q001, targetType, targetKey, question_blocks)` and `Unlink` | Entity registry/repository lookup | Direction validator, persistence write | A generic relationship write accepts a reverse or unsupported target. |
| TC-302 | `link`/viewer relationship mutation with `from=Q001`, `to=<candidate>`, `type=question_blocks` | Relationship service | Command/controller validation | A transport bypasses the same direction policy as the service. |
| TC-303 | `QuestionBlocker.Check(ctx, candidateType, candidateKey)` | Incoming-edge repository, Question read, registry read | Blocker predicate and F02 state decode | A direct source, generic block, or nonqualifying state is confused. |
| TC-304 | `QuestionBlocker.Check(ctx, feature, F001)` with ordered edge fixtures | Read seams only | Sorter, compact handoff projection | Tied edges select nondeterministically or disclose extra data. |
| TC-305 | Cobra `shark next <candidate> --json` and cascade resolution | Prompt renderer/action service after blocker | `resolveNext`, placeholder generation, cascade walker | A pause is formed after a prompt/action/lease side effect. |
| TC-306 | Cobra keyed `shark next <cascade-parent> --json` | Child candidate status/relationship seams | Cascade selection and parent transitioner | A blocked child stops unrelated eligible siblings or advances the parent. |
| TC-307 | `RunController.Run(ctx, RunOptions{EntityKey:<candidate>, DryRun:false})` | Worker launcher, action/prompt service | Run controller, blocker, lease acquire | Normal run claims or starts a worker before recognizing a blocker. |
| TC-308 | `RunController.Run(ctx, RunOptions{EntityKey:<candidate>, DryRun:true})` and cascade child invocation | Worker launcher, action/prompt service | Preflight lease and F02 responder action ordering | Dry-run or cascade omits the blocker check or a no-match claims too early. |
| TC-309 | Cobra `shark status advance <key> --json` for each supported entity type | Transition service after guarded command check | `dispatchTransition`, error renderer, transition service | A blocked advance writes status/history/claim before returning an error. |
| TC-310 | Same status command after `QuestionService.Resolve(ctx, ResolveQuestionInput{...})` | Transition service after predicate | Question closure, blocker | A closed Question still blocks or an unrelated Question transition is blocked. |
| TC-311 | Public generic planning/Question lifecycle callers with a gate edge | Services below public caller | Generic blocks reader, Question lifecycle service | Gate logic reinterprets legacy blocks or changes the Question itself. |
| TC-003 | Real I-01/I-02 services plus `shark next`, `shark run`, and status advance using temporary SQLite | Prompt renderer and worker launcher only | Registry, Question service, blocker, dispatch/run/status entrypoints | One boundary emits a different compact shape, leaks detail, or writes linked work. |
| TC-312 | F03-owned source/configuration scan and public unknown-route/command probes | None for absence checks | Router/registration for black-box probes | F03 adds F04 reads, E38 adapter, queue, traversal, or write-capable checker. |

## ISO 25010 coverage and observability

`N/A` means the characteristic does not apply to this bounded local CLI/service
change. No performance threshold or telemetry payload is specified. The runtime
evidence is the existing CLI result/error and durable-row snapshots; adding
telemetry would violate the compact handoff boundary.

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-301/302 | N/A bounded finite validation | ✅ TC-301 | ✅ TC-302 actionable rejection | ✅ TC-301 | ✅ TC-301 | ✅ TC-301 | N/A local Go/SQLite |
| AC-002 | ✅ TC-303 | ✅ TC-303 bounded calls | ✅ TC-303 | N/A internal predicate | ✅ TC-303 | ✅ TC-303 | ✅ TC-303 | N/A |
| AC-003 | ✅ TC-304 | ✅ TC-304 bounded inputs | N/A new contract | N/A internal predicate | ✅ TC-304 | ✅ TC-304 | ✅ TC-304 | N/A |
| AC-004 | ✅ TC-305/306 | N/A no threshold | ✅ TC-305 | ✅ TC-305 pause result | ✅ TC-306 | ✅ TC-305 | ✅ TC-305 | N/A |
| AC-005 | ✅ TC-307/308 | N/A no threshold | ✅ TC-308 F02 parity | ✅ TC-307 pause result | ✅ TC-307 | ✅ TC-307 | ✅ TC-308 | N/A |
| AC-006 | ✅ TC-309/310 | N/A no threshold | ✅ TC-310 unblocked retry | ✅ TC-309 typed error | ✅ TC-309 atomicity | ✅ TC-309 compact JSON | ✅ TC-309 | N/A |
| AC-007 | ✅ TC-311 | N/A | ✅ TC-311 legacy behavior | N/A | ✅ TC-311 | ✅ TC-311 | ✅ TC-312 | N/A |
| AC-008 | ✅ TC-003/312 | N/A | ✅ TC-003 I-01/I-02 | ✅ TC-003 compact handoff | ✅ TC-003 | ✅ TC-003/312 | ✅ TC-312 | N/A |

| Behavior | Metric | Log | Trace | Test assertion |
|---|---|---|---|---|
| Read-only qualification | Internal — no new observability; compact handoff is the allowed evidence | None | None | TC-303/304 count reads and compare all write-side snapshots. |
| Blocked `next`/`run` | Existing paused result only; no metric or telemetry | Existing command error/result conventions only | None | TC-305 through TC-308 assert `pause`, exact handoff, and zero side effects. |
| Blocked advance | Existing typed CLI error only; no metric or telemetry | Existing error convention only | None | TC-309 asserts human and JSON rendering plus unchanged rows. |

## Integration scenarios

| Interaction | Test and boundary | Gate and closure evidence |
|---|---|---|
| I-01 — F01 to F03 | `tests/contracts/e39_interactions_test.go#TC-001` stays live; TC-003 reuses Q identity/registry/base `blocking` and summary | Live. F03 code-review owner confirms the existing contract remains compatible. |
| I-02 — F02 to F03 | `tests/contracts/e39_interactions_test.go#TC-002` stays live; TC-003 reads validated state and first pending responder | Live. F03 code-review owner confirms open/answering qualification and closure behavior. |
| I-03 — F03 to F04 | `tests/contracts/e39_interactions_test.go#TC-003` | Live. TC-003 asserts four fields only at next/run/advance; F03 code-review owner records closure evidence. |
| UAT-03 | Temporary SQLite: Q001 open/blocking, directly linked F001, unlinked F002 | Capture paused `next`/`run`, rejected advance, byte-for-byte durable snapshots, then resolve Q001 and show F001 resumes normally. |
| X-06 | No F03 test | The map assigns producer coverage to F04 and consumer activation to E38-F09. This plan must not claim X-06 coverage. |

## Acceptance test cases

### TC-301: Validate the finite `question_blocks` direction matrix

**Feature requirement:** REQ-F-001, REQ-NF-001, REQ-NF-003.  
**Acceptance criterion:** AC-001.  
**Technique applied:** Equivalence partitioning and decision table.  
**ISO 25010:** Functional suitability, reliability, security, compatibility.

Open a pre-migration SQLite fixture containing existing relationship rows,
indexes, dependent views, and the Question cleanup trigger. Upgrade it, assert
the schema version and preserved durable structures, then create `Q001 ->` one
persisted fixture of each eligible target type. Create the
reverse partition, Question/Sprint/Idea targets, duplicate, malformed type, and
unlink partition. Expect only eligible forward edges, existing duplicate/unlink
semantics, and identical relationship counts after every rejected call.

Negative cases: `F001 -> Q001`, `Q001 -> Q002`, `Q001 -> S001`, and
`Q001 -> I001` write nothing. Existing `blocks`, `depends_on`, `linked_to`, and
`references` continue to validate as before.

### TC-302: Preserve transport parity for relationship mutations

**Feature requirement:** REQ-F-001, REQ-NF-001.  
**Acceptance criterion:** AC-001.  
**Technique applied:** Contract-surface enumeration.  
**ISO 25010:** Functional suitability, usability, security.

Drive the CLI link/unlink and existing viewer relationship mutation paths for
one accepted `Q001 -> F001` edge and every rejected matrix representative.
Expect the transport to return the service’s actionable error and no row write;
expect accepted/unlink payloads to follow the generic relationship contract.

### TC-303: Qualify only the direct open blocking predicate

**Feature requirement:** REQ-F-002, REQ-NF-001 through REQ-NF-003.  
**Acceptance criterion:** AC-002.  
**Technique applied:** Decision table and contract-surface enumeration.  
**ISO 25010:** Functional suitability, performance, reliability, security.

Call `QuestionBlocker.Check` for F001 across the complete state × blocking ×
edge-kind table. A direct `question_blocks` source in `open` or `answering`
with `blocking=true` returns I-03. Direct false-blocking, every other state,
generic `blocks`, indirect `Q001 -> T001 -> F001`, unlinked Q001, claim state,
and responder-completion controls return no match. Assert one candidate lookup,
one filtered incoming-edge lookup, bounded Question reads, no traversal, and
unchanged Question/candidate/relationship/claim/history/note/work-session
snapshots.

### TC-304: Return a deterministic, compact I-03 handoff

**Feature requirement:** REQ-F-002, REQ-NF-001/002.  
**Acceptance criteria:** AC-002, AC-003.  
**Technique applied:** Boundary value analysis and deterministic ordering.  
**ISO 25010:** Functional suitability, performance, reliability, security.

Give F001 Q003/Q002/Q001 qualifying edges at timestamps `10:00:00`,
`10:00:01`, and equal `10:00:00` IDs 8 and 7. Expect Q001/ID 7 to win. Marshal
the handoff and assert the exact four keys and canonical values. Use no-state
and invalid-state fixtures to require an actionable error rather than empty
owner/responder handoff. Negative: response summary, evidence, context,
relationship ID, claim, requester, prompt, and credentials are absent.

### TC-305: Pause direct keyed next before dispatch work

**Feature requirement:** REQ-F-003, REQ-NF-001 through REQ-NF-003.  
**Acceptance criterion:** AC-004.  
**Technique applied:** State transition and decision table.  
**ISO 25010:** Functional suitability, compatibility, usability, reliability, security.

Run `shark next F001 --json` with a qualifying Q001 edge. Expect the normal
identity/current status, `action="pause"`, blank agent/provider/model/effort/
prompt, and the exact four-field `question_block`. Assert that placeholder,
action, prompt, auto-advance, and lease seams have zero calls. Repeat with no
match and compare the existing response byte-for-byte except that
`question_block` is omitted.

### TC-306: Fall through a blocked cascade child

**Feature requirement:** REQ-F-003, REQ-NF-002/003.  
**Acceptance criterion:** AC-004.  
**Technique applied:** State transition and decision table.  
**ISO 25010:** Functional suitability, compatibility, reliability.

Resolve a cascade parent whose first selected child F001 is blocked and second
eligible child F002 is unlinked. Expect F002’s ordinary live dispatch result;
the parent and F001 retain status and no parent auto-advance occurs. With all
children blocked, return the normal paused root/cascade result without prompt
or lease work. Negative: a blocked child never prevents selection of unrelated
eligible F002.

### TC-307: Pause direct run in normal mode without side effects

**Feature requirement:** REQ-F-004, REQ-NF-002/003.  
**Acceptance criterion:** AC-005.  
**Technique applied:** State transition and decision table.  
**ISO 25010:** Functional suitability, usability, reliability, security.

Run `shark run F001` for a qualifying edge. Expect the runner’s paused result
with the exact I-03 handoff. Assert no worker launcher, lease, claim,
heartbeat, release, prompt, transition, or work-session action. Compare all
durable rows before and after. Negative: the F02 Question responder control
still uses its existing action-before-lease ordering when it has no blocker.

### TC-308: Preserve dry-run and cascade run parity

**Feature requirement:** REQ-F-004, REQ-NF-002/003.  
**Acceptance criterion:** AC-005.  
**Technique applied:** State transition and decision table.  
**ISO 25010:** Functional suitability, compatibility, reliability, maintainability.

Repeat TC-307 with `--dry-run` and with a cascade-selected blocked child. Both
return the same paused compact handoff with zero worker/lease side effects.
No-match normal and dry-run controls preserve F02’s action-before-lease
behavior. Negative: no responder lookup occurs for an already paused or
archived action, and a cascade child cannot bypass the preflight.

### TC-309: Reject each supported advance atomically

**Feature requirement:** REQ-F-005, REQ-NF-001 through REQ-NF-003.  
**Acceptance criterion:** AC-006.  
**Technique applied:** State transition and atomicity enumeration.  
**ISO 25010:** Functional suitability, usability, reliability, security.

For `epic`, `feature`, `task`, `bug`, `change`, and `tech_debt`, run
`shark status advance <key> --json` against a qualifying source. Expect a typed
Question-blocked error with only I-03. Verify global JSON contains the same four
fields and human output names the candidate/key/summary only. Compare candidate
status/history, Question/state, relationship, claim, note, and work-session
snapshots byte-for-byte before and after each rejection. Counts alone are not
evidence because an in-place mutation must also fail this test.

### TC-310: Clear the predicate on Question closure only

**Feature requirement:** REQ-F-005, REQ-NF-003.  
**Acceptance criteria:** AC-006, AC-007.  
**Technique applied:** State transition.  
**ISO 25010:** Functional suitability, compatibility, reliability.

Resolve Q001 through the F02 service after TC-309’s rejection, then rerun the
same advance. Expect the normal target transition and no special Question-owned
linked-work write. Drive a Question self-transition while its gate edge exists;
expect its normal lifecycle result. Negative: an unrelated Q002 and a generic
`blocks` edge neither block nor clear F001.

### TC-311: Preserve excluded and legacy behavior

**Feature requirement:** REQ-F-001, REQ-F-005, REQ-F-006, REQ-NF-003.  
**Acceptance criterion:** AC-007.  
**Technique applied:** Contract-surface enumeration.  
**ISO 25010:** Functional suitability, compatibility, reliability, security.

Exercise generic dependency planning, direct service transitions outside the
supported CLI boundary, all F01/F02 Question lifecycle commands, generic
`blocks`, and unrelated candidates while Q001 has a valid gate edge. Expect
unchanged legacy behavior and no Question auto-resolution, claim, status,
history, note, relationship, or linked-work mutation from a check.

### TC-003: Prove the shared I-03 v1 contract without disclosure

**Feature requirement:** REQ-F-002 through REQ-F-006, REQ-NF-001 through REQ-NF-003.  
**Acceptance criterion:** AC-008; **Interaction:** I-03.  
**Technique applied:** Contract-surface enumeration and state transition.  
**ISO 25010:** Functional suitability, compatibility, reliability, security.

In `tests/contracts/e39_interactions_test.go#TC-003`, use real temporary
SQLite, I-01 registry identity, and I-02 configured state. Assert the same
I-03 handoff shape and values at keyed next, run (normal/dry), cascade, and
blocked status advance. Assert direct-only qualification, closure recovery,
unlinked peer eligibility, no durable writes, and no hidden Question field.
This same test remains the F04 consumer pointer; do not create a twin test.

### TC-312: Enforce the F03 forbidden manifest

**Feature requirement:** REQ-F-006, REQ-NF-001/003.  
**Acceptance criterion:** AC-008.  
**Technique applied:** Source/configuration enumeration.  
**ISO 25010:** Functional suitability, security, maintainability.

Scan F03-owned paths and execute public unknown-command/route probes for
`open-by-responder`, `blocking-for`, E38 adapter/queue, recursive traversal,
response/evidence/context disclosure, and checker writes. Exercise rejected
reverse/Question/Sprint/Idea edges. Expect absence or actionable rejection and
no write. Scope the scan to F03-owned code so it does not prohibit future F04
or E38 work.

## Codex test-plan red-team

**Verdict:** PASS  
**Issues raised:** 1  
**Issues addressed before development:** 1  
**Issues deferred:** 0

The dispatch prompt names no `codex_command`, so the local read-only Codex CLI
reviewed the rendered plan and specification on 2026-07-31. It returned:

> CONCERNS — AC-006 / TC-309 used row counts, which could miss an in-place
> candidate or Question mutation. Require byte-for-byte durable snapshots.

TC-309 now compares byte-for-byte candidate status/history, Question/state,
relationship, claim, note, and work-session snapshots before and after every
rejected advance. A second read-only review after that edit returned `PASS`.
No concerns were deferred.

## Recommendations

- [x] Ready for development after Codex red-team PASS or resolved concerns
- [ ] Needs BA refinement
- [ ] Needs technical refinement
