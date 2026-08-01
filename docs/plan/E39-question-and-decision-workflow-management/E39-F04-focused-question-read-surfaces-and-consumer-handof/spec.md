---
feature_key: E39-F04
epic_key: E39
title: Focused Question read surfaces and consumer handoff specification
type: combined-spec
interaction_ids:
  - I-01
  - I-02
  - I-03
cross_epic_ids:
  - X-06
contract_version: x-06-producer-v1
---

# Focused Question read surfaces and consumer handoff specification

**Feature**: [E39-F04](feature.md)  
**Parent contracts**: [Requirements](../requirements.md), [architecture](../architecture.md), [research report](research-report.md), [interaction map](../E39-interaction-map.md), [cross-epic map](../E39-cross-epic-map.md), and [UAT plan](../uat-plan.md)

This specification is incremental to the E39 PRD. It delivers the focused
read and producer-handoff portion of E39 REQ-F-006; see the parent PRD for the
business context and the parent architecture for system-wide decisions.

## Requirements

### Functional requirements

#### REQ-F-001 — Expose open Questions by current responder

**Trace**: E39 REQ-F-006; I-01 and I-02; UAT-01.

Add the named focused read `open-by-responder`. It accepts one non-empty,
trimmed responder identity plus `limit` and `offset`, and returns only
Questions whose persisted status is `open` or `answering` and whose validated
`QuestionState.CurrentResponder()` exactly equals that identity. It must not
select a completed responder, resolution owner, claim holder, response text,
or an unconfigured/invalid Question state.

The CLI command is `shark question open-by-responder <identity> [--limit N]
[--offset N]`. The HTTP route is `GET /api/v1/questions/open-by-responder`
with required `responder`, optional `limit`, and optional `offset` query
parameters. The CLI and HTTP routes return the same compact page: ascending
canonical Question key order, default limit 50, limit 1–100, and offset >= 0.
An empty page is a successful empty array. Unknown parameters, missing or
blank responder, malformed pagination, and limit/offset outside those bounds
are rejected before the service call and write nothing.

#### REQ-F-002 — Expose direct Questions blocking an entity

**Trace**: E39 REQ-F-005, REQ-F-006, and REQ-F-009; I-03; UAT-01 and UAT-03.

Add the named focused read `blocking-for`. It accepts exactly one canonical
non-Question target key plus `limit` and `offset`; the key resolves through
the existing entity registry. It returns every direct incoming
`question_blocks` source that satisfies the existing `QuestionBlocker`
qualification: source type `question`, `blocking=true`, status `open` or
`answering`, and valid I-02 state. It preserves the direct-only relationship
meaning from F03: no graph traversal, generic `blocks`, unrelated links, or
candidate/Question mutation.

The CLI command is `shark question blocking-for <entity-key> [--limit N]
[--offset N]`. The HTTP route is `GET /api/v1/questions/blocking-for` with
required `entity_key`, optional `limit`, and optional `offset`. Results use
the I-03 compact shape, ordered by relationship `created_at` then relationship
ID (the same deterministic order used by F03), and never disclose a
relationship ID. A missing/unsupported target, Question target, malformed
pagination, or unknown query/flag fails before data is returned or changed.

#### REQ-F-003 — Keep compact and authorized-full reads separate

**Trace**: E39 REQ-F-006 and REQ-NF-001; parent architecture “Interfaces,
migration, security, and ADRs”; UAT-05.

Keep `shark question get <key>`, `GET /api/v1/questions/{key}`, generic
Question list/search/viewer projections, both focused reads, and I-03 results
compact. Compact Question data is `models.QuestionProjection` plus, only for
the blocking-focused result, the I-03 fields `resolution_owner` and
`current_responder`. It excludes `context_data`, response summaries, evidence
pointers, resolution pointer/kind, claim/session data, relationship IDs,
prompts, and credentials.

Add a distinct full-read service operation and explicit transport endpoints:
`shark question full <key> --actor <identity>` and
`GET /api/v1/questions/{key}/full?actor=<identity>`. The full operation first
loads and validates `QuestionState`; it returns the compact projection plus
the ordered responders, bounded responses, resolution owner, resolution kind,
and resolution pointer only when `actor` exactly equals the current responder
or `resolution_owner`. Any other actor, a missing/blank actor, a Question
without valid state, or an unknown parameter is a typed access/validation
error; it does not fall back to a full persisted model and writes nothing.

`actor` is a request identity supplied to the service-owned policy seam, not
an authentication redesign or durable authorization record. CLI/API transport
only passes it through. A later authenticated host may derive this identity at
the seam without changing the compact or full projection contracts.

#### REQ-F-004 — Publish the X-06 producer handoff

**Trace**: X-06; E39 REQ-F-006 and REQ-NF-003; UAT X-06.

Publish X-06 producer v1 as the provider-neutral combination of: (1) the
existing serial `shark next Q### --json` lifecycle from I-02; (2) the direct
I-03 compact blocked-work handoff; (3) the two F04 focused compact reads; and
(4) the explicitly authorized full read. A consumer receives Question keys
and compact safe projections, and uses the public Question APIs/CLI to resume
or inspect work. It must not receive a host queue, copied mutable
`question_state`, chat/council transcript, provider adapter, or authority to
claim/advance/resolve linked work.

E39-F04 is the producer documentation and contract-test owner. E38-F09 is
the blocked activation consumer and must add its own live consumer coverage
when E39 completes; this feature neither edits nor resumes E38-F09.

### Non-functional requirements

#### REQ-NF-001 — Enforce bounded, safe, compatible reads

Use parameterized repository queries and the existing finite validation
patterns. Each query has the stated 1–100 page bound and stable order. Decode
and validate I-02 only at the service boundary; malformed state is an
actionable read error, never a guessed responder or partially disclosed
result. No F04 operation writes Questions, relationships, claims, history,
notes, work sessions, search data, or workflow state. Existing base
get/list/search/viewer/CLI/API outputs remain byte-for-byte compatible except
for the newly registered explicit routes and commands.

#### REQ-NF-002 — Preserve focused-query performance and operations

The responder query uses a bounded Question candidate query and in-process
validated-state filter. The blocking query uses the existing indexed incoming
`question_blocks` relationship lookup and bounded Question source reads. It
does not recursively traverse a graph or run on unrelated dispatch paths.
Both report ordinary repository/service validation errors without raw
`context_data` or credential-like material. No schema migration, index,
cache, telemetry payload, or background queue is added.

### Acceptance criteria

| ID | Scenario | Expected result |
| --- | --- | --- |
| AC-001 | Configure ordered responders across open, answering, ready, terminal, malformed, and unconfigured Questions; query each responder through CLI and HTTP. | Only `open`/`answering` Questions with that exact derived pending responder appear in canonical-key order; no response/provenance/context content appears. |
| AC-002 | Link qualifying and nonqualifying Questions to targets using `question_blocks`, generic links, indirect paths, false blocking, terminal states, and multiple direct blockers. | `blocking-for` returns only direct F03-qualified sources in edge-created-at/ID order with compact I-03 data; every excluded partition is absent. |
| AC-003 | Invoke base get/list/search/viewer, both focused reads, and explicit full read as a requester, unrelated caller, responder, and resolution owner. | All ordinary/focused outputs remain compact; only the assigned responder or resolution owner receives full bounded fields; every denial writes nothing. |
| AC-004 | Exercise missing/blank identities, unknown flags/query parameters, malformed target keys, Question targets, non-boolean/invalid pagination, and repository/state errors. | Each boundary reports the existing actionable error shape before service mutation; existing get/list behavior remains unchanged. |
| AC-005 | Execute the public CLI/API/runner contract against a temporary SQLite database and inspect rows before/after. | All F04 reads are read-only; no Question/relationship/claim/history/note/work-session row changes, no host queue, and no mutable state copy occur. |
| AC-006 | Execute the shared I-01/I-02/I-03 tests plus X-06 producer contract tests. | Existing live contracts remain compatible and `TC-004` proves the documented producer handoff without E38 implementation. |

### Out of scope and forbidden manifest

F04 excludes Question workflow/routing changes, response recording and
resolution changes, claims, mutation of linked work, relationship vocabulary
or predicate changes, generic `blocks` behavior, recursive/global blockers,
schema migrations, queues/schedulers, caches, telemetry changes, viewer UI
redesign, raw-context access, auth-provider implementation, and every E38-F09
provider adapter or continuation behavior.

The versioned `f04-forbidden-v1` manifest rejects a raw `Question` JSON
transport; `context_data`, `responses`, `evidence_pointer`,
`resolution_pointer`, `resolution_kind`, claims, relationship IDs, prompts,
or credentials in compact/focused/I-03 JSON; any `--full` switch on existing
`get`/`list`; generic-list responder/blocking-target filters; and routes or
commands that mutate state during a focused read. A source/configuration scan
and black-box requests prove each item remains absent or rejects without
mutation.

## Architecture

### Component changes

| Path | Change |
| --- | --- |
| `internal/models/question.go` and tests | Add explicit compact blocking-focused and authorized-full projection types/functions. Retain `ProjectQuestion` as the existing compact baseline and never marshal the persisted model. |
| `internal/repository/question/repository.go` and tests | Add parameterized bounded candidate reads for open Questions and direct blocking relationships, preserving existing list ordering and F03 predicate inputs. No schema change. |
| `internal/services/question_service.go` and tests | Add focused input/result types, `ListOpenQuestionsByResponder`, `ListQuestionsBlocking`, and `ReadQuestionFull`; own state validation, direct predicate reuse, projection, and actor policy. |
| `internal/services/question_blocker.go` and tests | Export/reuse the narrow direct qualification/read seam needed by `blocking-for`; do not duplicate or change `Check`'s dispatch behavior. |
| `internal/cli/services_global.go` and `internal/viewer/server/wire.go` | Wire the existing Question service, relationship repository, and blocker/read dependencies consistently for CLI and viewer-hosted API operation. |
| `internal/cli/commands/question.go` and tests | Register `open-by-responder`, `blocking-for`, and `full`; enforce finite flags and render compact/full projections. Existing `get` and `list` remain unchanged. |
| `internal/api/question_handler.go`, `internal/viewer/server/server.go`, and tests | Register/validate the two focused GET routes and full GET route; use the same service seam and JSON projection contracts. |
| `tests/contracts/e39_interactions_test.go` | Add `TC-004` covering I-01/I-02/I-03 composition and X-06 producer v1. |
| `docs/plan/E39-question-and-decision-workflow-management/E39-F04-focused-question-read-surfaces-and-consumer-handof/{spec.md,test-plan.md}` | Keep the versioned X-06 producer contract and explicit E38-F09 activation breadcrumb durable. |

### Data model and interface contracts

No table, migration, index, or persistent field changes. F04 reads the F01
`questions` record and decodes the existing I-02 `question_state` only after
repository retrieval. It reads F03 `question_blocks` through the existing
relationship repository and qualification logic. The read inputs are finite:

| Service operation | Input | Output | Error/no-match |
| --- | --- | --- | --- |
| `ListOpenQuestionsByResponder` | responder, limit, offset | compact `[]QuestionProjection` | invalid identity/state is error; zero matches is empty page |
| `ListQuestionsBlocking` | target type/key, limit, offset | `[]QuestionBlock` I-03 compact records | invalid target/state is error; zero matches is empty page |
| `ReadQuestionFull` | key, actor | `QuestionFullProjection` | not authorized/invalid state is typed error; no compact fallback |

`QuestionFullProjection` contains the compact Question fields plus
`resolution_owner`, ordered `responders`, ordered bounded `responses`, and
optional `resolution_kind`/`resolution_pointer`. It contains no raw
`ContextData`, claim/session data beyond response `session_id` already held in
the bounded provenance model, relationship IDs, prompts, or credentials. The
compact blocking result is exactly F03's `QuestionBlock` four fields; it does
not acquire an edge ID or create a second model.

### API and CLI contracts

| Surface | Request | Success | Rejection |
| --- | --- | --- | --- |
| CLI responder read | `question open-by-responder <identity> --limit --offset` | compact JSON/page or three-column human rows | unsupported flag, blank identity, invalid pagination |
| CLI blocking read | `question blocking-for <entity-key> --limit --offset` | I-03 compact JSON/page | bad/Question target, unsupported flag, invalid pagination |
| CLI full read | `question full <key> --actor <identity>` | full JSON/explicit human detail | missing actor, non-authorized actor, invalid state |
| HTTP responder read | `GET /api/v1/questions/open-by-responder?responder=&limit=&offset=` | `200` compact array | `400` finite validation error |
| HTTP blocking read | `GET /api/v1/questions/blocking-for?entity_key=&limit=&offset=` | `200` I-03 compact array | `400` finite validation error |
| HTTP full read | `GET /api/v1/questions/{key}/full?actor=` | `200` full projection | `400` malformed request, `403` policy denial, established not-found status |

HTTP route registration must precede the generic `GET /api/v1/questions/{key}`
route when required by the standard library pattern matcher. JSON field names
are the projection struct tags above. Human output never includes a field not
available in its JSON projection.

### Key technical decisions

| Decision | Rationale |
| --- | --- |
| Add named focused service operations instead of expanding base list filters. | The current CLI/API intentionally reject unknown filters; named queries make the semantic/state cost auditable. |
| Derive responder only with `QuestionState.CurrentResponder()`. | It preserves I-02 serial routing and prevents claim/response inference. |
| Reuse F03's direct predicate and I-03 shape. | A second relationship interpretation would let read and dispatch disagree. |
| Require an explicit full endpoint and actor. | Existing get/list consumers stay safe; a full read has a deliberate authorization boundary. |
| Use supplied identity as a policy seam, not authentication. | The application has no authenticated principal contract to reuse; F04 must not invent one. |
| Add no migration. | F01/F03 already persist the required Question and relationship data; F04 is a read/projection extension. |

### Integration with existing code

`QuestionRepository.List` and `QuestionService.ListQuestions` remain the
baseline finite-list path. New repository methods follow the same
`context.Context`, parameterized SQL, `QuestionListFilter` bounds, and
`scanQuestion` conventions in `internal/repository/question/repository.go`.
The service mirrors `GetQuestion`, `GetQuestionByID`, and `ListQuestions`,
uses `models.DecodeQuestionState`, and exposes only typed projections to
commands/handlers. `QuestionBlocker.Check` remains the authoritative F03
preflight; a shared lower-level qualifying-read helper may be extracted only
if both callers preserve exactly its source/target/status/directness rules.

`internal/cli/commands/question.go` and
`internal/api/question_handler.go` retain their current thin interfaces,
finite local flags/query-key allowlists, error mapping, and `models.ProjectQuestion`
baseline. `internal/viewer/server/wire.go` and `server.go` use the same
Question service instance as existing routes. No focused route belongs in the
generic viewer mutation service.

## Cross-feature interactions

### Consumes: I-01 — Entity and platform registration

| Property | Contract |
| --- | --- |
| Producer | E39-F01 |
| Shape source | [E39-F01 I-01 v1 contract](../E39-F01-question-entity-and-platform-registration/spec.md#produces-i-01--entity-and-platform-registration-v1) |
| Consumer use | Canonical Q key, registered Question record, and metadata-only base projection. |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-001` |
| Gate mode | live |

### Consumes: I-02 — Serial workflow and response provenance

| Property | Contract |
| --- | --- |
| Producer | E39-F02 |
| Shape source | [Architecture workflow](../architecture.md#workflow-and-direct-gate) |
| Consumer use | Validated status, resolution owner, and derived first pending responder for read authorization/filtering. |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-002` |
| Gate mode | live |

### Consumes: I-03 — Scoped relationship gate

| Property | Contract |
| --- | --- |
| Producer | E39-F03 |
| Shape source | [Architecture direct gate](../architecture.md#workflow-and-direct-gate) |
| Consumer use | Direct `question_blocks` qualification and four-field compact blocking result. |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-003` |
| Gate mode | live |

## Cross-epic integrations

### Produces: X-06 — Provider-neutral live Question handling

| Property | Contract |
| --- | --- |
| Consumer | E38-F09 Provider-Neutral Coordination and Live Resume (activation owner) |
| Contract / shape source | E39 architecture §2–§4; E38-F09 feature.md |
| Producer handoff | Public serial Question lifecycle/query/dispatch, F03 compact blocked-work handoff, F04 focused compact reads, and explicit authorized full read; no queue or copied mutable state. |
| UX / CX handoff | One scoped responder prompt and compact blocked-work handoff; a responder/resolution owner can deliberately request full bounded detail. |
| Test coverage | `tests/contracts/e39_interactions_test.go#TC-004`; E39 UAT-01–06 and X-06. |
| Activation / closure | E39-F04 proves producer v1. E38-F09 remains blocked and must add consumer coverage when it resumes; this does not declare E38 live activation. |

## Verification plan

Run AC-001 through AC-006; `TC-001` through `TC-004`; the established F01,
F02, and F03 regression suites; and `f04-forbidden-v1`. Capture a temporary
SQLite demonstration with two pending responders, one resolution owner, one
directly blocked target, and an unlinked control. Record compact CLI/API
results, an allowed full read, denied full reads, empty pages, and before/after
row counts. Redact response/evidence values in evidence capture. Finally run
`make fmt && make lint && make test` after implementation.
