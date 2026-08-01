---
feature_key: E39-F01
epic_key: E39
title: Question entity and platform registration specification
type: combined-spec
interaction_ids:
  - I-01
contract_version: i-01-v1
---

# Question entity and platform registration specification

**Feature**: [E39-F01](feature.md)  
**Parent contracts**: [Requirements](../requirements.md), [architecture](../architecture.md), [research report](research-report.md), [interaction map](../E39-interaction-map.md), and [UAT plan](../uat-plan.md)

This specification is incremental to the E39 PRD. It defines the registered
platform record that produces I-01. See the parent requirements for the
business need and the parent architecture for system-wide decisions.

## Requirements

### Functional requirements

#### REQ-F-001 — Create a bounded Question record

**Trace**: E39 REQ-F-001; research capability map, “Top-level entity model”.

Add `question` as a top-level entity type. Its canonical key grammar is
`Q[0-9]{3}` with numeric range `001` through `999`; `q001` normalizes to
`Q001`. The parser does not trim input. Therefore ` Q001`, `Q001 `, Unicode
digits, `Q000`, `Q1`, `Q0001`, `Q001-extra`, and `q00a` are unknown keys.
Question creation allocates `max(persisted key) + 1` and does not reuse an
existing key; it fails before mutation when `Q999` already exists. A unique
database constraint is the concurrency authority:
two concurrent creations either receive two distinct keys or one receives the
existing constraint error wrapped as an allocation failure; neither may return
an unpersisted identity.

Creation trims `title`, `summary`, and `requester` once, then rejects an empty
result. It stores the trimmed values. `description` is optional and is stored
unchanged when present. `blocking` defaults to `false`. The only accepted
creation status is an omitted status or `draft` after trim and lower-case
normalization; both persist the base Question workflow's `draft` state. Every
other status is rejected before key allocation. The resulting record includes
shared entity fields plus `summary`, `blocking`, and `requester`.

#### REQ-F-002 — Add finite, recoverable persistence

**Trace**: E39 REQ-F-001 and REQ-NF-003; research capability map, “Additive
persistence and cleanup”.

Add `questions` by an idempotent, additive migration. It contains the standard
standalone entity columns used by the chosen local model pattern and the
Question columns `summary`, `blocking`, and `requester`; it has a unique `key`
index, a key lookup index, a `(status, requester, blocking, key)` list index,
an `updated_at` trigger, and the same concrete-entity cleanup triggers used by
the selected standalone pattern. `context_data` uses the existing structured
context representation described in I-01 v1.

Migration verification uses exactly these predecessor fixtures, each made by
the current pre-F01 `db.InitDB` and repository APIs: P0 is an empty initialized
database; P1 contains one task; P2 contains P1 plus one row each in
`entity_notes`, `entity_history`, `entity_documents`, `entity_relationships`,
`entity_tags`, `entity_claims`, and `work_sessions` that references the task.
For P0-P2, initialization preserves the original row counts and selected task
values, adds the stated Question schema objects, and permits Q001 creation.
The test forces one DDL error, captures the operation and database error, then
reruns initialization; the retry either completes a clean additive migration or
returns the same actionable error without deleting or rewriting predecessor
rows. A deployed correction is forward-only; table deletion is not a rollback.

#### REQ-F-003 — Register only existing generic semantics

**Trace**: E39 REQ-F-001 and REQ-F-004; research capability map, “Registry and
typed repository adapters” and “Context, history, documents, and
relationships”.

Register `QuestionRepositoryAdapter` in `EntityRegistry`. The exact generic
context contract is the existing `ContextService` API:
`GetContext(ctx, EntityTypeQuestion, key)`,
`SetContextField(ctx, EntityTypeQuestion, key, field, value)`, and
`ClearContext(ctx, EntityTypeQuestion, key)`. It stores and returns
`models.ContextData` JSON. F01 supports the existing fields
`current_step`, `completed_steps`, `remaining_steps`,
`implementation_decisions`, `open_questions`, and `blockers` with their
existing value encodings. It does not add a raw-context method, an arbitrary
JSON setter, a Question-only field, or a new validation schema.

The finite generic surface matrix is part of this requirement:

| Surface | Production call or command | Valid F01 proof | Invalid proof | Stored result |
|---|---|---|---|---|
| Registry | `registry.GetRepository(EntityTypeQuestion)` | returns the Question adapter | unknown entity type errors | none |
| Notes | `NoteService.AddNote(ctx, EntityTypeQuestion, "Q001", type, content, actor)` and `ListNotes` | one typed note round-trips | empty required note input follows existing validation | `entity_notes` row |
| Context | the three `ContextService` calls above | set/get/clear `current_step` | `question_state` and `metadata` fields reject | `questions.context_data` |
| History | `EntityHistoryService.GetHistory(ctx, EntityTypeQuestion, "Q001")` | creation history reads | unknown Q key errors | `entity_history` rows created by normal operations |
| Documents | existing `EntityDocumentService` add/list/remove calls with Question type/key | one document association round-trips | missing Question errors | `entity_documents` row |
| Tags | `TagService.AttachMany` and `ListTagsForEntity` with the resolved Question ID | one tag round-trips | invalid tag follows current validation | `entity_tags` row |
| Relationships | `entityrel.Repository.Create(ctx, fromType, fromKey, toType, toKey, relationType)` | one existing `linked_to` relationship round-trips | missing endpoint errors | `entity_relationships` row |
| Claims | `ClaimService.Claim`, `Heartbeat`, `Release` with `"question", "Q001"` | one lease lifecycle succeeds | wrong session follows existing error | claim and work-session rows only |

Every unit test may mock only the named backing repository below the service.
The I-01 integration test uses the real service, registry, adapter, and SQLite
database. Claims, heartbeats, expiry, and release must not change Question
status or `context_data`.

#### REQ-F-004 — Define the finite transport and dispatch contract

**Trace**: E39 REQ-F-001 and REQ-NF-003; research capability map, “Unified
command and search/viewer projections”.

Commands and handlers remain thin and call `QuestionService`. F01 registers
only these Question-specific transport operations:

| Surface | Supported request | Valid result | Rejected input or operation |
|---|---|---|---|
| CLI | `shark question create`, `get`, `list`, `update`, `delete`, and `status`; generic `shark create/get/update/delete/list`, `link`, `note`, `context`, `history`, `tag`, `related-docs`, `search`, and `view` | uses the service/registered generic path | unknown key/type and every operation in the forbidden manifest |
| HTTP | `GET, POST /api/v1/questions`; `GET, PATCH, DELETE /api/v1/questions/{key}`; existing status read/transition routes | POST returns `201`, Q key, and `Location: /api/v1/questions/Q###` | malformed JSON, unknown key, and unsupported method use current error shape |
| List | `status` exact match, `requester` exact match, `blocking` boolean, `limit` 1–100 default 50, `offset` >=0 default 0, key ascending only | returns the bounded requested page | unsupported filter/sort, limit outside range, non-boolean blocking, or negative offset |
| Update | trimmed `title`, `summary`, `requester`; optional `description`; `blocking` | persists only supplied supported fields | key, status, `context_data`, or any response/provenance field |
| Search/viewer | normal identity, title, summary, status, blocking, requester, and timestamps | metadata projection only | context and other forbidden material never project |
| Keyed next | `shark next Q001` (JSON is implicit) | the fixture below returns the complete unchanged envelope | invalid key, missing record, unsupported status, and terminal state |

The shipped Question workflow fixture contains `draft` and terminal `archived`.
For persisted `Q001` in `draft`, `shark next Q001` returns the normal JSON
`NextResponse` with `entity_key=Q001`, `entity_type=question`, `status=draft`,
`action=pause`, empty agent/provider/model/prompt fields, no `error`, and no
claim, history, context, or status mutation. `archived` returns the same
identity/status fields with `action=archive`; an unsupported stored status is a
non-zero standard error and makes no mutation. F01 does not select an actor or
produce a worker prompt.

SQLite is the required F01 runtime database. `internal/db.Database` and
parameterized repository methods must remain driver-neutral, but a live Turso
credentialed migration test is explicitly deferred to the repository-wide
Turso integration suite; it is not a F01 acceptance claim.

#### REQ-F-005 — Produce the narrow I-01 v1 contract

**Trace**: E39 interaction map I-01; research capability map, “F02 serial
state/provenance, F03 gate, F04 safe reads”.

I-01 v1 contains only: (1) a canonical `Q001`-style key in the closed numeric
domain; (2) `models.EntityTypeQuestion`; (3) the persisted base record with
shared entity identity, `title`, `summary`, `requester`, `blocking`, and base
`status`; and (4) the registered adapter plus the existing structured
`models.ContextData` get/set-field/clear calls named in REQ-F-003. It contains
no other domain contract.

The public consumer seams are deliberately limited to
`keys.NewKeyService().Parse("Q001")`,
`registry.GetRepository(EntityTypeQuestion).GetByKey(ctx, "Q001")`, and the
three existing `ContextService` calls in REQ-F-003. F02, F03, and F04 must use
those seams before they add their separately owned behavior. This feature does
not add future workflow fields, response data, specialized relations, gates,
or focused query/disclosure routes.

### Non-functional requirements

#### REQ-NF-001 — Preserve data and interface safety

Use parameterized queries and local standalone-service transaction boundaries.
Validate all service input before persistence or association mutation. Do not
reinterpret existing `E`, `F`, `T`, `B`, `CC`, `S`, `I`, or `TD` keys. Metadata
projections must omit `context_data`. User-visible validation errors name the
input, violated constraint, and corrective action.

#### REQ-NF-002 — Preserve a versioned baseline

The baseline inventory is the currently registered types: `epic`, `feature`,
`task`, `bug`, `change`, `tech_debt`, `sprint`, and `idea`. Version it in
`tests/contracts/e39_interactions_test.go` as `registration-baseline-v1` and
execute every type × surface row: key parse/normalize, registry resolution,
note add/list, context get/set/clear, history read, claim/heartbeat/release,
generic command detection, and keyed-next baseline. A row records its expected
pre-F01 result; Question is an added row, not a replacement expectation.

#### REQ-NF-003 — Keep results bounded and recoverable

Use the list bounds in REQ-F-004 and the listed indexes. Initialization is
idempotent and migration failures remain actionable. Search, viewer, command,
and API errors use the existing error convention.

### Acceptance criteria

| ID | Scenario | Expected result |
|---|---|---|
| AC-001 | Create through CLI and HTTP using each valid/invalid required-field partition, omitted/draft status, allocation failure, and two concurrent creates. | Valid requests persist trimmed fields and distinct canonical keys; every invalid request writes nothing. |
| AC-002 | Parse the complete Q-key partition. | Q001, Q100, and Q999 normalize; every malformed, whitespace, Unicode-digit, and colliding candidate stays unknown. |
| AC-003 | Initialize P0-P2, inject migration failure, and retry. | Finite oracle preserves every predecessor row, adds the named objects, and reports/retries errors safely. |
| AC-004 | Execute every REQ-F-003 surface-matrix row. | Each valid operation reaches its named storage; invalid inputs do not mutate; leases preserve base state. |
| AC-005 | Execute every REQ-F-004 transport and query-matrix row, including delete rollback. | Supported paths preserve the finite contract; invalid queries/updates fail; failed delete leaves the record and associations unchanged. |
| AC-006 | Run the exact `shark next Q001` draft, archived, and unsupported-status fixtures. | Each complete response/error oracle matches and causes no F01 mutation. |
| AC-007 | Execute `registration-baseline-v1`. | Every existing type × surface baseline row retains its expected result. |
| AC-008 | Run the forbidden manifest's source/configuration scan and black-box operation attempts. | Every named forbidden artifact and public operation remains absent or rejects without mutation. |

### Out of scope and forbidden manifest

F01 excludes the separately owned serial workflow, provenance, scoped gate,
focused reads, disclosure policy, X-06 handoff, queues, transcript storage,
auth-policy redesign, legacy data backfill, and linked-work mutation.

The versioned `f01-forbidden-v1` manifest comprises these symbols/files/routes:
`question_state`, `QuestionResponse`, `QuestionResolution`, `question_blocks`,
`QuestionBlocker`, responder/response/resolve/withdraw/supersede handlers,
`/api/v1/questions/{key}/response`, `/resolve`, `/withdraw`, `/supersede`,
`/open-by-responder`, `/blocking-for`, an E38 Question adapter, and a Question
prompt that selects an actor. The black-box checks attempt each named public
route/command operation and require the current unknown-command/route error
without database mutation. The source/configuration scan covers the listed
symbols, workflow files, API routes, and CLI registrations.

## Architecture

### Component changes

Use the standalone `TechDebt` model/repository/adapter pattern and thin Cobra
and HTTP boundaries. The complete F01 inventory is:

| Path group | Change |
|---|---|
| `internal/models/question.go`, `entity.go`, and `entity_note.go` | Add base Question model, `EntityTypeQuestion`, validation, and entity conformance. |
| `internal/keys/service.go` and tests | Add the strict ASCII Q range parser, normalization, and collision coverage. |
| `internal/repository/question/`, `aliases.go`, `dbconn/db.go` | Add typed persistence and recognized concrete table wiring. |
| `internal/db/db.go` and migration tests | Add the idempotent table, indexes, trigger, cleanup, P0-P2, and failure/retry tests. |
| `internal/services/question_service.go`, `question_repo_adapter.go`, `entity_registry.go`, `services_global.go`, and tests | Add service, adapter, registry, and caller paths; retain generic service signatures. |
| `internal/cli/commands/question.go`, generic dispatch commands, and tests | Register the finite CLI matrix. |
| `internal/api/question_handler.go`, its tests, and `cmd/server/services.go` | Register the finite HTTP matrix. |
| `internal/searchindex/sql.go`, search tests, viewer service/handler/tests | Add metadata-only normal projections. |
| `internal/config/workflow/`, `internal/config/action/`, `internal/sharkdata/default_data/workflow/question.yaml`, and tests | Add only the draft/archived base fixture required by AC-006. |
| `tests/contracts/e39_interactions_test.go` | Add I-01 v1 and registration-baseline-v1 live tests. |

Review must search for further closed type switches. A discovered supported-type
switch belongs to F01 and is not deferred.

### Data and interface decisions

| Decision | Rationale |
|---|---|
| Use a concrete `questions` table. | A distinct durable key and generic lookup cannot be supplied by notes or metadata. |
| Use the strict Q001-Q999 range without trimming keys. | It is finite, unambiguous, and does not silently change key input conventions. |
| Reuse `models.ContextData` and `ContextService` field operations. | Those are the actual production callers; a raw JSON setter would be new scope. |
| Use SQLite acceptance evidence and driver-neutral APIs. | It is the configured local runtime; live Turso credentials are unavailable and separately covered. |
| Make the base `draft` workflow pause. | It proves keyed dispatch registration without claiming later actor-routing behavior. |
| Keep search/viewer metadata-only. | It enables discovery while withholding generic context from projections. |

## Cross-feature interactions

### Produces: I-01 — Entity and platform registration (v1)

| Property | Contract |
|---|---|
| Consumers | E39-F02, E39-F03, and E39-F04 |
| Shape source | This specification, [REQ-F-005](#req-f-005--produce-the-narrow-i-01-v1-contract) |
| Producer shape | The four I-01 v1 elements enumerated in REQ-F-005 only |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-001` |
| Consumer calls | Key parser, registered adapter `GetByKey`, and existing typed `ContextService` calls |
| Gate mode | live, as assigned by [the interaction map](../E39-interaction-map.md) |
| Closure owner | E39-F01 code-review owner |
| Required UAT evidence | UAT-01 F01 foundation: create Q001; link it; retrieve/list/search it; show the persisted row and history; verify metadata-only projection. F02 separately owns the responder-dispatch portion of UAT-01. |
| Reviewer and transition | Code reviewer verifies TC-001 plus the named UAT evidence, records the I-01 live-gate review in F01 code-review evidence, then approves F01's normal workflow transition. |

F01 consumes no I-## row. I-02 and I-03 remain later producer contracts.

## Cross-epic integrations

F01 produces, consumes, and validates no X-## row. X-06 remains owned by
E39-F04 and E38-F09; F01 adds no E38 adapter or test.

## Runtime evidence procedure

Run a read-only evidence capture against a temporary initialized database after
implementation. Use the CLI to create Q001, use HTTP to read it, add one
`linked_to` association, run `shark next Q001`, then delete it. Capture the
redacted command/API output, the `questions`, `entity_relationships`, and
`entity_history` rows before deletion, and the dependent-row counts after
deletion. Search the captured application log, trace export when enabled,
search output, and viewer output for the unique sentinel context value; the
sentinel must be absent from every projection. Do not include credentials or
unredacted context in evidence. This procedure supplies F01's UAT-01 foundation
and I-01 closure evidence; it does not demonstrate later feature behavior.

## Verification plan

Run the AC matrix, I-01 `TC-001`, `registration-baseline-v1`, and
`f01-forbidden-v1`. After Go changes, run:

```text
make fmt
make lint
make test
```

RECOMMENDED OUTCOME: pass
