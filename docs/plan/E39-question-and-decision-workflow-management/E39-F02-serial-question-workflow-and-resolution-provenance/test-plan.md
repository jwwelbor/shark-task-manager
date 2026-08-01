# Test Plan: E39-F02 — Serial Question Workflow and Resolution Provenance

**Created:** 2026-07-30  
**Feature PRD:** `spec.md`  
**Epic UAT:** `../uat-plan.md`  
**Status:** APPROVED

## Scope, drift, and integration

F02 consumes I-01 and produces I-02: validated bounded Question state, serial
responder routing, response/resolution/terminal provenance, and the finite
CLI/HTTP surface. It does not own F03's gate, F04's focused reads, or X-06.
No unresolved drift was found between the specification, E39 UAT plan, and
interaction maps. The I-02 consumers E39-F03 and E39-F04 are currently `draft`
(live Shark read, 2026-07-30). X-06 belongs to E39-F04/E38-F09, so F02 has no
X-## coverage to create.

| Requirement | AC | Cases | Coverage note |
|---|---|---|---|
| REQ-F-001, NF-001 | AC-001, AC-002 | TC-101, TC-102 | Migration and bounded state partitions. |
| REQ-F-002, NF-002 | AC-003 | TC-103, TC-104 | Serial keyed dispatch and lease lifecycle. |
| REQ-F-003, NF-001/002 | AC-004 | TC-105 | Atomic response and idempotency. |
| REQ-F-004 | AC-005 | TC-106 | Classified destination closure. |
| REQ-F-005 | AC-006 | TC-107 | Terminal provenance. |
| REQ-F-006, NF-003 | AC-007 | TC-108, TC-109 | Finite CLI/API contract. |
| REQ-F-007 | AC-008 | TC-001, TC-002, TC-110 | I-01/I-02 and forbidden manifest. |

| Interaction | Test location | Boundary / UAT evidence |
|---|---|---|
| I-01, F01 → F02 | `tests/contracts/e39_interactions_test.go#TC-001` | Existing registered Q, registry/context/history/claim adapters; UAT-01; live gate unchanged. |
| I-02, F02 → F03/F04 | `tests/contracts/e39_interactions_test.go#TC-002` | Validated state exposes only first pending responder and bounded provenance; UAT-02/UAT-04; live gate; F02 code-review owner records TC-002 and named UAT evidence. |
| X-06 | N/A | Explicitly owned by F04/E38-F09; F02 must not add an E38 adapter. |

## AC test matrix

| AC | Technique | Input/setup and expected outcome including edges |
|---|---|---|
| AC-001 | State transition + equivalence partitioning | F01 `draft` Questions with context, associations, history, claim; fresh install fixture. Migration changes only `draft` to `open`, synthesizes no responders, and preserves all predecessor bytes; repeat is idempotent. |
| AC-002 | EP + BVA + attack-class enumeration | Owner/responder at 0/1/256/257 UTF-8 bytes; 0/1/10/11 responders; duplicate, raw JSON, completed-without-response, credential/prompt/transcript. Only ordered valid configuration persists derived first responder; every rejected partition and a second configure write nothing. |
| AC-003 | State transition + decision table | `alice,bob,carol`; successful response, release, heartbeat, expiry, failed claim. Only first pending dispatches; only committed success plus release enables next; every lifecycle-only path leaves current pending. |
| AC-004 | Race decision table + idempotency EP | Competing sessions and exact/conflicting retries. One matching active session completes current responder; exact retry is no-op success; stale/wrong/conflicting retry changes no state/note/history/status. |
| AC-005 | Decision table + EP | Six resolution kinds with valid/invalid/missing destinations and incomplete responders. Only ready owner with valid destination resolves and writes provenance; invalid paths write nothing. |
| AC-006 | State transition + BVA | Eligible/terminal statuses, owner/nonowner, reason 0/1/1000/1001 bytes, self/missing/valid superseder. Eligible operation commits terminal provenance atomically; invalid paths write nothing. |
| AC-007 | Contract-surface enumeration | Each CLI/HTTP operation valid plus missing input, malformed/unknown JSON, wrong actor/session, invalid/terminal state. Valid returns metadata projection; rejects use existing error shape without mutation. |
| AC-008 | Contract/source enumeration | Existing I-01, new I-02, and `f02-forbidden-v1`. I-01 remains unchanged; TC-002 proves I-02; excluded gate/read/X-06 surfaces absent or reject without mutation. |

## Test infrastructure

| Layer | Existing pattern | Planned use |
|---|---|---|
| Unit | `internal/models/question_test.go` | Pure state validation, byte bounds, forbidden content, derived responder. |
| Service | `internal/services/question_service_test.go` | Mock repository/claim/note/history seams; direct production service requests. |
| Repository | `internal/repository/question/repository_test.go` (`test.NewIsolatedTestDB`) | Real SQLite migration and transactional conditional-update/rollback. |
| CLI/API | `internal/cli/commands/question_test.go`, `internal/api/question_handler_test.go` | Mock service, strict flags/JSON, metadata projection/errors. |
| Contract | `tests/contracts/e39_interactions_test.go` | Real temporary DB, CLI/API/keyed-next wiring, TC-002 and forbidden scan. |

No new helper is needed initially. A repeated response fixture may be extracted
beside its test package only; it must not hide the listed production entrypoint.

## Caller-path contracts

| TC | Production entrypoint / argument shape | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-101 | `db.InitDB(path)`, `QuestionRepository.GetByKey(ctx,key)` | None; real SQLite | Migration/repository | Migration overwrites context or associations. |
| TC-102 | `QuestionService.ConfigureWorkflow(ctx, ConfigureWorkflowInput{Key,ResolutionOwner,Responders})` | Repo transaction/note/history adapters | State validator/service | Generic raw context persists invalid state. |
| TC-103 | Cobra keyed `shark next Q001 --json` via `runner.EntityTransitioner` | Prompt renderer only | `GetNextStatus`, claim lookup, transitioner | Helper selects a responder unlike production next. |
| TC-104 | `ClaimService.{Claim,Heartbeat,Release,ReclaimExpired}` then keyed next | None for integration | Claim service/keyed-next | Lease lifecycle completes responder. |
| TC-105 | `QuestionService.RecordResponse(ctx, RecordQuestionResponseInput{Key,SessionID,Responder,Summary,EvidencePointer})` | Repo transaction, ClaimService.Get | Validator/note-history transaction | Wrong or conflicting retry mutates provenance. |
| TC-106 | `QuestionService.Resolve(ctx, ResolveQuestionInput{Key,Owner,Kind,Pointer})` | Destination lookup/repo transaction | Destination validator/status transition | Bad destination resolves or mutates linked work. |
| TC-107 | `QuestionService.{Withdraw,Supersede}(ctx,input)` | Repo transaction/target lookup | Owner/state/status checks | Invalid terminal provenance is recorded. |
| TC-108 | Full Cobra `shark question` operation tree | Question service | CLI parsing/service | Command accepts unsupported/missing input or leaks content. |
| TC-109 | `POST /api/v1/questions/{key}/{workflow,response,resolve,withdraw,supersede}` | `QuestionServicer` | Strict decoder/route registration | Bad JSON reaches service write. |
| TC-001 | Existing real I-01 registry/context/claim/history contract | None | F01 adapters | F02 changes I-01 behavior. |
| TC-002 | Real `shark next Q001 --json`, Question/claim service, workflow bundle | Prompt renderer only | State/claim service and next envelope | Consumer gets unvalidated/second responder. |
| TC-110 | Public CLI/API plus F02 path source/config scan | None for public rejection | Router/registration in black-box cases | F03/F04/X-06/queue/disclosure appears in F02. |

## ISO 25010 and observability

No performance threshold is specified; N/A below means bounded local Go/SQLite
behavior, not untested performance. Portability is N/A. CLI/API errors provide
the only applicable usability evidence. NF-001 prohibits full response,
credential, prompt, or full-pointer telemetry; concise typed note/history is
the required durable runtime evidence.

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅101 | N/A bounded migration | ✅101 | N/A | ✅101 | ✅101 | ✅101 | N/A |
| AC-002 | ✅102 | N/A bounded 10 | ✅102 | ✅108 | ✅102 | ✅102 | ✅102 | N/A |
| AC-003 | ✅103/104 | N/A | ✅103 | N/A | ✅104 | ✅104 | ✅103 | N/A |
| AC-004 | ✅105 | N/A | N/A | N/A | ✅105 | ✅105 | ✅105 | N/A |
| AC-005 | ✅106 | N/A | ✅106 | ✅108 | ✅106 | ✅106 | ✅106 | N/A |
| AC-006 | ✅107 | N/A | N/A | ✅108 | ✅107 | ✅107 | ✅107 | N/A |
| AC-007 | ✅108/109 | N/A | ✅108/109 | ✅108/109 | ✅109 | ✅109 | ✅108 | N/A |
| AC-008 | ✅001/002/110 | N/A | ✅001 | N/A | ✅002 | ✅110 | ✅110 | N/A |

| Behavior | Metric/log/trace | Required evidence / assertion |
|---|---|---|
| Configuration, response, resolution, withdraw, supersede | No new response telemetry; full content forbidden | Typed concise note + history; TC-102/105/106/107 prove atomic presence and rollback. |
| Keyed serial dispatch | No new metric specified | TC-103/104 assert normal `NextResponse` exposes only current metadata. |
| Rejected input | Existing actionable error shape, no secrets | TC-102/105/106/108/109 assert no durable mutation. |

## Acceptance test cases

### TC-101: Additive draft-to-open migration preserves I-01 record

REQ-F-001/002, NF-003; AC-001. Run fresh-install and upgrade against a `draft`
Question with generic context, association, history, and claim plus a non-draft
control. Expect only `draft → open`, zero synthesized responders, unchanged
base/association/context bytes, and idempotent rerun. Negative: non-draft is
unchanged.

### TC-102: Configure only one valid bounded QuestionState

REQ-F-001, NF-001; AC-002. Configure `Q001` with `release-owner` and ordered
`alice,bob,carol`; expect derived `alice`, all pending, concise note/history,
metadata projection. Reject every matrix partition and compare state/note/history
snapshots exactly; negative: a second configure never overwrites state.

### TC-103: Keyed next routes exactly the first pending responder (I-02)

REQ-F-002; AC-003; I-02. Drive `shark next Q001 --json` for configured
`alice,bob,carol`. Expect normal complete envelope for alice and no mutation;
after successful persisted response and release, only bob; after carol, pause.
Negative: neither bob nor carol is exposed early or with response material.

### TC-104: Lease lifecycle alone never advances responder state

REQ-F-002, NF-002; AC-003. Cover active/failed claim, wrong heartbeat, expiry,
and release without response. Current alice remains pending, no audit record is
created, and no next responder appears while claimed or after lifecycle-only
events. Negative: release/expiry never constitutes a response.

### TC-105: Response write is session-bound, atomic, exact-idempotent

REQ-F-003, NF-001/002; AC-004. With alice/session-a, submit `approved` and
the feature spec path as evidence. Expect one completed response with state,
note, history in one transaction. Concurrent session-b, stale session, wrong
responder, 1001-byte summary, bad evidence, and changed same-session retry
leave identical snapshots; exact repeat has no duplicate audit rows.

### TC-106: Resolution validates every kind/destination before closure

REQ-F-004; AC-005. For ready Q001 owned by `release-owner`, cover all six
kind/pointer pairs (existing note, feature spec, progress anchor, ADR+refs,
canonical linked key, empty only for no consequence). Expect resolved plus one
note/history. Negative: incomplete responder, wrong owner, malformed/missing
destination, or nonempty no-consequence pointer makes no write.

### TC-107: Withdraw and supersede retain terminal provenance safely

REQ-F-005; AC-006. For each eligible state test authorized 1/1000-byte reason
and existing distinct Q002 superseder. Expect atomic terminal status/provenance.
Negative: unauthorized, 0/1001-byte reason, self/missing target, terminal state
preserve all state and completion snapshots.

### TC-108: CLI operations enforce finite command contract

REQ-F-006; AC-007. Drive full Cobra configure/respond/resolve/withdraw/supersede
commands. Valid calls return metadata only. Omit each required flag and supply
unsupported input; existing CLI error shape returns with no write. Negative:
generic update/context cannot carry F02-specific state or provenance.

### TC-109: HTTP routes strictly decode and project metadata

REQ-F-006; AC-007. Exercise every F02 POST route validly, then send malformed,
unknown/trailing JSON, missing input, wrong actor/session, invalid and terminal
state. Valid returns 200 metadata projection; every reject uses existing error
shape and preserves durable snapshot.

### TC-110: F02 forbidden manifest stays absent or rejects without mutation

REQ-F-007, NF-001/003; AC-008. Scan F02-owned source/config/bundle paths and
invoke public CLI/API for `question_blocks`, focused reads, gate hooks, E38
adapter, second queue/claim, response search/viewer/telemetry, and linked-work
mutation. Expect absence or unknown/404 and unchanged Question/linked records.
The scan is limited to F02-owned paths so it does not prohibit future F03/F04.

## Codex Test-Plan Red-Team

**Verdict:** PASS  
**Issues raised:** 0  
**Issues addressed before dev:** 0  
**Issues deferred:** 0

Each AC was checked for open-ended robustness language, technique fit,
partition completeness, ISO coverage, observability, negative cases, and
caller-path contract. The dispatch prompt supplied no separate `codex_command`;
this review was performed on the rendered plan. Not adding response telemetry
is intentional: NF-001 prohibits it and requires typed note/history evidence.

## Recommendations

- [x] Ready for development
- [ ] Needs BA refinement
- [ ] Needs tech refinement
