---
feature_key: E39-F01
epic_key: E39
title: Test plan - Question entity and platform registration
status: approved
created: 2026-07-30
---

# Test plan: E39-F01 - Question entity and platform registration

**Feature specification:** [spec.md](spec.md)
**Parent UAT:** [E39 UAT plan](../uat-plan.md)
**Status:** APPROVED pending implementation. This is a test-design verdict, not
an implementation, review, UAT, or workflow approval.

## Scope and drift disposition

This plan proves one additive, first-class `question` entity and I-01 v1. It
does not implement F02 serial response/provenance, F03 blocking gates, F04
focused reads, or X-06. The revised specification closes every finding from the
previous review; no behavior below is inferred from a broader architecture
document.

| Previous finding | Closed specification source | Plan coverage |
|---|---|---|
| Field normalization, accepted status, allocation failure, concurrency | REQ-F-001 | TC-001, TC-002 |
| Q-key range and malformed partitions | REQ-F-001, AC-002 | TC-003 |
| Predecessor data, preservation oracle, failure/retry | REQ-F-002 | TC-004, TC-005 |
| Typed ContextService rather than raw JSON | REQ-F-003 | TC-006, TC-007 |
| Per-surface caller paths | REQ-F-003 matrix | TC-007 |
| Finite transports, filters, update/delete, SQLite/Turso | REQ-F-004 | TC-008, TC-009 |
| Actual keyed-next fixture and complete oracle | REQ-F-004 | TC-010 |
| Versioned type x surface baseline | REQ-NF-002 | TC-012 |
| Forbidden manifest plus black-box rejection | forbidden manifest, AC-008 | TC-011 |
| Narrow I-01 pointer, calls, live closure, UAT split | REQ-F-005 and I-01 table | TC-006, evidence procedure |
| Read-only runtime evidence/redaction procedure | spec Runtime evidence procedure | TC-013 |

## Traceability and test techniques

| Requirement / AC | Test cases | ISTQB technique and negative proof |
|---|---|---|
| REQ-F-001; AC-001 | TC-001, TC-002 | Equivalence partitions for all fields/statuses; state transition for allocation; concurrency race. Invalid input has zero writes. |
| REQ-F-001; AC-002 | TC-003 | Equivalence partitions and BVA across `001..999`; malformed/whitespace/Unicode/collision enumeration. |
| REQ-F-002; AC-003 | TC-004, TC-005 | State-transition migration fixtures and injected-DDL-error retry. Preservation count/value oracle is finite P0-P2. |
| REQ-F-003; AC-004 | TC-006, TC-007 | Contract-surface enumeration and lease state-transition testing. Every invalid surface input makes no mutation. |
| REQ-F-004; AC-005 | TC-008, TC-009 | Decision table for commands, HTTP, filters, updates, deletion, projections; boundary values for pagination. |
| REQ-F-004; AC-006 | TC-010 | State-transition and exact response-envelope comparison for draft, archived, and unsupported state. |
| REQ-NF-002; AC-007 | TC-012 | Complete versioned Cartesian `type x surface` baseline, not all-pairs. |
| Forbidden manifest; AC-008 | TC-011 | Closed artifact/public-operation enumeration; source/config scan and black-box rejection. |
| Runtime evidence and I-01 closure | TC-013 | Production-path demonstration plus redaction scan; no secret/context value may project. |

## ISO 25010 coverage matrix

`N/A` is justified in the Notes column; no unmarked characteristic is implied.

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability | Notes |
|---|---|---|---|---|---|---|---|---|---|
| AC-001 | TC-001, TC-002 | N/A | TC-001 | TC-001 | TC-001 | TC-001 | N/A | N/A | Bounded input/transport behavior; no throughput claim. |
| AC-002 | TC-003 | N/A | TC-003 | TC-003 | TC-003 | TC-003 | N/A | N/A | Strict input boundary and existing-key compatibility. |
| AC-003 | TC-004, TC-005 | TC-004 query-plan check | TC-004 | N/A | TC-005 | TC-005 | TC-004 | TC-004 | SQLite is required; live Turso migration is explicitly deferred. |
| AC-004 | TC-006, TC-007 | N/A | TC-006 | N/A | TC-007 | TC-007 | TC-007 | N/A | Generic-service semantics and lease isolation. |
| AC-005 | TC-008, TC-009 | TC-008 bounds/index check | TC-008 | TC-008 errors | TC-009 rollback | TC-009 projection | TC-008 | TC-008 driver-neutral API | Bounded list and metadata-only result. |
| AC-006 | TC-010 | N/A | TC-010 | TC-010 pause/archive response | TC-010 no mutation | TC-010 no prompt/claim | N/A | N/A | Exact existing response shape. |
| AC-007 | TC-012 | N/A | TC-012 | N/A | TC-012 | N/A | TC-012 | N/A | Baseline version is explicit. |
| AC-008 | TC-011 | N/A | TC-011 | N/A | TC-011 | TC-011 | TC-011 | N/A | Closed forbidden manifest. |

## Test architecture and caller-path contracts

Repository/migration and contract integration tests use a temporary real SQLite
database with cleanup. CLI, HTTP, and service unit tests mock only their named
service/repository seam; they never use a real DB. Pure key/model tests use no
I/O. `tests/contracts/e39_interactions_test.go` is a real service/registry/
adapter/SQLite integration test.

| TC | Production entrypoint and argument shape | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `shark question create TITLE --summary S --requester R --blocking`; `QuestionService.CreateQuestion(ctx, CreateQuestionInput{Title, Summary, Requester, Description, Blocking, Status})` | CLI: Question-service interface; service: typed Question repo | Cobra parser in CLI tests; service in service tests; repo in repository tests | A transport drops `requester`/`blocking`, or a service returns a key never persisted. |
| TC-002 | `POST /api/v1/questions` JSON then `GET` returned `Location` | Question-service interface at HTTP handler | Route registration, decoder, response writer | Handler returns 201 but has an incomplete location/body or accepts invalid status. |
| TC-003 | `keys.NewKeyService().Parse(raw)` and `Normalize(raw)` | None | Copied regex or pre-normalized raw string | Parser accepts whitespace, Unicode digits, suffix, or out-of-range Q key. |
| TC-004 | `db.InitDB(tempPath)` then Question repo `Create/GetByKey/List` | None below `InitDB`; real SQLite | Migration internals, SQLite metadata, query plan | Fresh schema is missing an index, trigger, or column. |
| TC-005 | `db.InitDB` over P0/P1/P2 and a forced DDL error, then retry | None below `InitDB`; real SQLite | Fixture DDL/rows, migration error | Upgrade deletes/re-writes predecessor rows or hides an unrecoverable error. |
| TC-006 | `QuestionService.CreateQuestion`; `registry.GetRepository(EntityTypeQuestion).GetByKey`; `ContextService` get/set/clear | None above real service/registry/adapter/SQLite | Question service, registry, adapter, consumer seam | The record persists but F02/F03/F04 cannot consume the I-01 seam. |
| TC-007 | Exact service calls in the surface matrix below | Only named backing repo in service-unit coverage | Registry, ClaimService state reads, replacement Question lease | A generic operation rejects Question or a lease changes base state/context. |
| TC-008 | Concrete Question commands/handlers and generic registered routes | Command/handler service interface; real registration in dispatch integration | Generic dispatch, key detection, registry | Specific CRUD works while generic routing/filter/update behavior is incomplete. |
| TC-009 | Search/viewer production service and `QuestionService.DeleteQuestion(ctx, "Q001")` | Search/viewer repositories; real SQLite for cleanup | Projection builder or cleanup trigger | A projection leaks context or a failed deletion leaves/re-writes associations. |
| TC-010 | `shark next Q001` through key detection, workflow adapter, response renderer | Workflow/action provider only after real detection/registry/adapter | Detection, adapter construction, renderer, claim state | A helper passes while actual command rejects Q001 or mutates state. |
| TC-011 | Manifest scan and public forbidden command/route attempts | None for scan; real SQLite for black-box no-mutation check | Mocked absence assertion | Later-feature behavior becomes reachable through an existing path. |
| TC-012 | Existing production key/registry/service/claim/command/next callers for each baseline type | Existing transport service interfaces only | Detection, registry, skipped types | Adding Question changes a closed switch for an existing type. |
| TC-013 | CLI create/link/next/delete plus HTTP get against temp initialized DB | None above real CLI/API/service/SQLite | Manually composed outputs or redaction-only mocks | Automated tests pass but the integrated path leaks sentinel context or fails cleanup. |

### REQ-F-003 finite surface matrix (TC-007)

| Surface | Exact caller | Valid / invalid partition | Durable assertion |
|---|---|---|---|
| Registry | `registry.GetRepository(EntityTypeQuestion)` | adapter / unknown type | adapter returned / error, no write |
| Notes | `NoteService.AddNote(ctx, EntityTypeQuestion, "Q001", type, content, actor)` and `ListNotes` | one typed note / empty required input | one `entity_notes` row / no row |
| Context | `GetContext`, `SetContextField(..., "current_step", value)`, `ClearContext` | listed existing fields / `question_state`, `metadata` | `questions.context_data` round-trip then clear / unchanged |
| History | `EntityHistoryService.GetHistory(ctx, EntityTypeQuestion, "Q001")` | creation history / unknown key | normal history rows / error, no mutation |
| Documents | existing `EntityDocumentService` add/list/remove | association / missing Question | `entity_documents` row / no row |
| Tags | `TagService.AttachMany` then `ListTagsForEntity(resolvedID)` | valid tag / current invalid-tag partition | `entity_tags` row / no row |
| Relationship | `entityrel.Repository.Create(ctx, fromType, fromKey, toType, toKey, "linked_to")` | existing endpoints / missing endpoint | `entity_relationships` row / no row |
| Claims | `ClaimService.Claim/Heartbeat/Release(ctx, "question", "Q001", session)` | one session / wrong session | claim/work-session lifecycle only; status and context byte-equal |

## Acceptance test cases

### TC-001 — CLI and service creation partitions (AC-001)

Use values `"  Release gate  "`, `"  Confirm gate  "`, and
`"  release-manager  "`; assert persisted values are each trimmed once.
Create accepts omitted status and whitespace/case-normalized `draft` only, with
optional description unchanged and default `blocking=false`. For title,
summary, requester, and status test one non-empty valid value plus empty and
whitespace-only invalid values. Test every rejected status partition, including
non-draft after normalization, before allocation. Force `Q999` to prove an
actionable allocation failure and no new row. Two concurrent valid creates must
produce two distinct persisted canonical keys or one wrapped unique-allocation
error; neither may return an unpersisted identity. Every invalid/allocation
failure leaves rows, history, associations, and search unchanged.

### TC-002 — HTTP create/read contract (AC-001)

POST valid JSON with all fields and with optional description omitted. Assert
201, `Q001`, and `Location: /api/v1/questions/Q001`, then GET the returned
location and compare persisted normalized fields. Malformed JSON, missing/
whitespace required values, rejected status, and unknown key each use the
existing error shape and write nothing.

### TC-003 — strict Q-key grammar and collision partitions (AC-002)

Accept/normalize `q001`, `Q001`, `Q100`, and `Q999`. Reject exactly the
specified invalid representatives: leading/trailing whitespace, Unicode digits,
`Q000`, `Q1`, `Q0001`, `Q001-extra`, `q00a`, and mixed-case malformed suffixes.
Existing E/F/T/B/CC/S/I/TD representatives retain their types. A duplicate
Q001 follows the allocation-error contract and never resolves a malformed key.

### TC-004 — fresh schema and bounded query artifacts (AC-003)

Initialize P0. Assert the Question columns, unique key index, key lookup index,
`(status, requester, blocking, key)` list index, updated-at trigger, and the
same selected standalone cleanup triggers. Create/update Q001 and prove the
timestamp trigger. `EXPLAIN QUERY PLAN` for exact key and bounded requester/
blocking listing must select a Question index; failed duplicate insertion is
the negative case.

### TC-005 — P0-P2 migration preservation, forced error, retry (AC-003)

Construct predecessor fixtures exclusively through the pre-F01 `db.InitDB` and
repository APIs: P0 empty; P1 one task; P2 P1 plus one row in each of
`entity_notes`, `entity_history`, `entity_documents`, `entity_relationships`,
`entity_tags`, `entity_claims`, and `work_sessions` referencing that task.
For each fixture, snapshot row counts and selected task values; initialization
must preserve them, add Question schema, and allow Q001. Force one DDL error,
assert operation and database error are retained, retry init, and require either
a clean additive completion or the same actionable error with every predecessor
row intact. No test accepts table deletion as rollback.

### TC-006 — I-01 v1 shared live contract (AC-004, I-01)

At exactly `tests/contracts/e39_interactions_test.go#TC-001`, create Q001 via
the real service, resolve it through the registered adapter, and invoke only
`ContextService.GetContext`, `SetContextField`, and `ClearContext` with the
existing supported fields/encodings. Assert the four and only four I-01 v1
elements: canonical key, `EntityTypeQuestion`, persisted base record, and
typed `models.ContextData` operations. Consumer seams are
`keys.NewKeyService().Parse("Q001")`, adapter `GetByKey`, and those three
ContextService calls; F02/F03/F04 add no future behavior in this test. Invalid
`question_state` and `metadata` fields reject with context unchanged.

### TC-007 — generic associations and lease isolation (AC-004)

Execute every row of the finite surface matrix. Each valid operation must reach
its named storage. Each invalid partition must return the existing validation/
not-found error before association mutation. Snapshot Question status and
`context_data` before claim, heartbeat, expiry, and release; compare byte-for-
byte afterward. No responder state, Question-owned lease, or status transition
is allowed.

### TC-008 — finite transport/query matrix (AC-005)

Exercise Question-specific `create/get/list/update/delete/status` and generic
`create/get/update/delete/list/link/note/context/history/tag/related-docs/
search/view`. Exercise HTTP `GET,POST /api/v1/questions` and `GET,PATCH,DELETE
/api/v1/questions/{key}`. List partitions: exact `status`, exact `requester`,
boolean `blocking`, limits 1/50/100, offsets 0/last valid, key ascending only;
reject sort/filter omissions, non-boolean blocking, limit 0/101, and negative
offset. Update only trimmed title/summary/requester, optional description, and
blocking; reject key/status/context/provenance inputs before mutation. Assert
parameterized driver-neutral repository calls and SQLite acceptance; no live
Turso credentialed migration claim is made.

### TC-009 — projection and delete atomicity (AC-005)

With a unique context sentinel and generic associations, assert search/viewer
expose only identity, title, summary, status, blocking, requester, and
timestamps—not `context_data`, sentinel, or future response/provenance. Test
empty and first/last bounded pages. Inject one cleanup failure during delete:
the Question and all associations remain unchanged. Successful deletion cleans
notes, relationships, tags, documents, history, claims, and work sessions.

### TC-010 — exact keyed-next fixtures (AC-006)

Run `shark next Q001` (JSON is implicit) using the shipped Question workflow.
For `draft`, assert full `NextResponse`: Q001, `question`, `draft`, `pause`,
empty agent/provider/model/prompt, no error, and no claim/history/context/status
mutation. For `archived`, assert same identity/status with `archive`. An
unsupported stored status exits non-zero with the standard error and makes no
mutation. Invalid/missing Q keys are also non-mutating errors.

### TC-011 — `f01-forbidden-v1` structural and black-box guard (AC-008)

Scan the exact manifest symbols/files/routes from the specification, including
`question_state`, response/resolution types and handlers, `question_blocks`,
focused routes, E38 adapter, and actor-selecting Question prompt. For every
public named operation, attempt it against Q001 and require the current
unknown-command/route error plus identical pre/post database snapshot. A scan
pass alone is insufficient.

### TC-012 — `registration-baseline-v1` complete matrix (AC-007)

For `epic`, `feature`, `task`, `bug`, `change`, `tech_debt`, `sprint`, and
`idea`, execute every required baseline surface: parse/normalize, registry
resolution, note add/list, context get/set/clear, history read, claim/
heartbeat/release, generic command detection, and keyed next. Store the
pre-F01 expected result per row in `tests/contracts/e39_interactions_test.go`.
Question is an added row—not a changed existing expectation—and a skipped row
fails the test.

### TC-013 — read-only runtime evidence and I-01 closure procedure

Against a temporary initialized database, use CLI to create Q001, HTTP to read
it, add a `linked_to` association, run `shark next Q001`, and delete it. Capture
redacted CLI/API output; `questions`, relationship, and history rows before
delete; and dependent-row counts afterward. Search captured app logs, optional
trace export, search output, and viewer output for the unique sentinel context
value; it must be absent. This is UAT-01 F01 foundation only—not F02 responder
dispatch. F01 code reviewer verifies TC-006 and this evidence, records the I-01
live-gate review, and approves the normal F01 code-review transition.

## Integration and delivery gate

| Boundary | Required proof | UAT contribution |
|---|---|---|
| Migration/repository | TC-004/005 schema, preservation, retry | UAT-01 foundation |
| Registry/generic services | TC-006/007 real I-01 seam and lease isolation | I-01 |
| CLI/API/service | TC-001/002/008 finite transports | UAT-01 foundation |
| Key/registry/workflow | TC-003/010 exact no-side-effect dispatch | I-01; later UAT-02 prerequisite |
| Search/viewer/delete | TC-009/013 metadata/read-redaction/cleanup | UAT-01; UAT-05 prerequisite |

After Go changes run `make fmt`, `make lint`, and `make test`. Development may
advance only when all AC cases, I-01 shared pointer, `registration-baseline-v1`,
`f01-forbidden-v1`, and the evidence procedure pass.

## Codex test-plan red-team

**Verdict:** FAILED — no terminal verdict emitted (non-blocking review-tool
execution gap)
**Issues raised:** none returned
**Issues addressed before development:** 16 prior findings addressed by the
revised specification and this plan.
**Issues deferred:** none.

Two independent long-budget, read-only Codex executions were run against this
exact revised plan and its cited spec, UAT, interaction-map, and source seams.
Both completed repository inspection but emitted no final `PASS`, `CONCERNS`,
or `FAIL` message; the captured diagnostic transcript ends during source
inspection. This is recorded as `Codex test-plan review: FAILED — no terminal
verdict emitted after retry`. Per the test-planning procedure, unavailable
Codex review is non-blocking after the retry; it is not evidence of a PASS.
The prior review's 16 findings remain traceable in the drift table above and
are concretely closed by the requirement/test mappings there.

## Recommendation

- [x] Ready for development test planning; the required red-team execution gap
  is documented as non-blocking after retry, and later implementation gates
  still apply.
- [ ] Needs BA or technical refinement.

RECOMMENDED OUTCOME: pass
