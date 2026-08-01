---
feature_key: E39-F02
epic_key: E39
title: Serial Question workflow and resolution provenance specification
type: combined-spec
interaction_ids:
  - I-01
  - I-02
contract_version: i-02-v1
---

# Serial Question workflow and resolution provenance specification

**Feature**: [E39-F02](feature.md)  
**Parent contracts**: [Requirements](../requirements.md), [architecture](../architecture.md), [research report](research-report.md), [interaction map](../E39-interaction-map.md), and [UAT plan](../uat-plan.md)

This specification is incremental to the E39 PRD. It consumes the registered
Question record from I-01 and produces the I-02 serial workflow and provenance
contract. See the parent requirements for business context and the parent
architecture for system-wide decisions.

## Requirements

### Functional requirements

#### REQ-F-001 — Store validated, bounded responder state

**Trace**: E39 REQ-F-002 and REQ-NF-001; research capability map, “Shared JSON
context” and “I-02 narrow producer contract”.

Add `QuestionState` as a Question-owned value serialized in the existing
`questions.context_data` column. Do not add it to generic `models.ContextData`
or expose it through `ContextService.SetContextField`. The schema is:

| Field | Type and validation |
|---|---|
| `resolution_owner` | Required trimmed identity, 1–256 UTF-8 bytes. |
| `responders` | Ordered list of 1–10 distinct trimmed identities, each 1–256 UTF-8 bytes and state `pending` or `completed`. |
| `responses` | At most one response per responder. A response stores responder identity, summary, evidence pointer, and recorded timestamp. |
| `current_responder` | Derived first responder whose state is `pending`; it is not independently accepted or persisted as a trusted input. |
| `resolution_kind` | Empty before resolution, otherwise one supported value in REQ-F-004. |
| `resolution_pointer` | Empty only where REQ-F-004 permits it; otherwise a trimmed pointer of at most 2,048 UTF-8 bytes. |

Each response summary contains 1–1,000 UTF-8 bytes. Each evidence pointer is
1–2,048 UTF-8 bytes. Validation rejects duplicate responders, unknown response
authors, a response for a pending/other responder mismatch, a completed
responder without exactly one response, and state with no pending responder
before all responders complete. It rejects credential-shaped values, rendered
prompts, and transcript-shaped material in every state text field. Validation
errors name the field, limit or forbidden content class, and the durable
alternative: a typed note or authoritative record pointer.

Provide a one-time workflow-configuration operation for an `open` Question.
It accepts `resolution_owner` and the ordered responder identities, validates
the complete initial state, persists it, and records a concise typed note and
Question history entry. It rejects a second configuration attempt and never
accepts raw JSON. This operation is the only F02 path that creates pending
responders; F01 creation remains a base-record operation.

#### REQ-F-002 — Route only the first pending responder

**Trace**: E39 REQ-F-003 and REQ-F-004; [E39 architecture workflow and direct
gate](../architecture.md#workflow-and-direct-gate).

Replace F01's Question-only `draft`/`archived` fixture with the route-based
Question workflow states `open`, `answering`, `ready_for_resolution`,
`resolved`, `withdrawn`, and `superseded`. New Questions enter `open` after
F02 initialization. The existing `draft` value is not a valid F02 runtime
state; the additive migration converts persisted F01 `draft` Questions to
`open` without changing their base fields or generic associations.

For `shark next Q### --json`, `QuestionService.GetNextStatus` and the existing
keyed-next path derive the first pending responder from validated state. The
result retains the normal `NextResponse` envelope and renders one Question
responder prompt for that identity. It does not select a second responder,
create a new queue, claim the Question, or mutate a linked entity. When a
claim exists, the response remains non-dispatchable; no competing responder is
returned. After a successful response is committed and the claimant releases
the existing lease, the next keyed dispatch returns only the next pending
responder. Claim failure, expiry, heartbeat, or release without a successful
response leaves the current responder pending.

#### REQ-F-003 — Record a bounded successful response

**Trace**: E39 REQ-F-003, REQ-F-004, REQ-F-009, and REQ-NF-002.

Provide a Question response operation that accepts canonical Question key,
claim session ID, responder identity, summary, and evidence pointer. The
service validates all input, loads the active claim through the existing
`ClaimService` contract, and requires that its session and owner match the
derived current responder. In one Question-owned transaction, it writes the
validated state, adds a concise typed Question note, and records a Question
history entry before making the responder `completed`. The operation does not
release the claim; the existing parent loop releases it after its normal
successful transition.

A retry with the same completed response is idempotent only when the session,
responder, summary, and evidence pointer exactly match the durable response.
Every other retry fails without mutation. A response operation never claims,
releases, transitions, resolves, or edits a linked entity.

#### REQ-F-004 — Require classified resolution provenance

**Trace**: E39 REQ-F-007 through REQ-F-009 and REQ-NF-002.

Provide a resolution operation that accepts the canonical Question key,
resolution-owner identity, `resolution_kind`, and `resolution_pointer`. It is
available only in `ready_for_resolution`, after every requested responder has
one completed bounded response. The service requires the actor to equal
`resolution_owner`, validates the kind/destination pair below, records a
concise typed note and history entry, stores the resolved state, and then
performs the Question-only transition to `resolved`.

| `resolution_kind` | Required `resolution_pointer` |
|---|---|
| `local_clarification` | A concise linked-entity note reference. |
| `feature_change` | An authoritative feature specification or document path. |
| `product_decision` | A `docs/product/progress.md` decision-log anchor. |
| `architecture_decision` | An ADR path plus affected architecture/specification references. |
| `follow_up_work` | A canonical linked Shark task, bug, change card, tech-debt item, feature, or epic key. |
| `no_lasting_consequence` | No pointer; the bounded responses and Question history are the durable record. |

For every non-empty pointer, the service validates its required form and the
referenced local Shark key or document exists before any write. F02 only reads
the destination to validate it. It does not create, link, claim, update, or
advance follow-up work.

#### REQ-F-005 — Preserve withdrawal and supersession provenance

**Trace**: E39 REQ-F-007, REQ-F-009, and REQ-NF-002.

Provide Question-only `withdraw` and `supersede` operations. Both require the
configured `resolution_owner`, a concise reason of 1–1,000 UTF-8 bytes, and a
typed note plus history record in the same transaction as the terminal status.
`supersede` additionally requires an existing canonical Question key that is
not the target Question; the stored provenance points to that Question.
Neither operation is available after a terminal status, changes responder
completion, or mutates the superseding or linked record.

#### REQ-F-006 — Keep the transport surface finite

**Trace**: E39 REQ-F-003, REQ-F-007, and REQ-NF-003.

Extend the existing Question command and HTTP handler with only these F02
operations:

| Surface | Operation | Required result |
|---|---|---|
| CLI | `shark question configure-workflow <key>` | Requires `--resolution-owner` and ordered `--responder` values; returns metadata-only Question projection. |
| CLI | `shark question respond <key>` | Requires `--session`, `--responder`, `--summary`, and `--evidence-pointer`; returns metadata-only Question projection. |
| CLI | `shark question resolve <key>` | Requires `--owner` and `--resolution-kind`; requires `--resolution-pointer` except for `no_lasting_consequence`; returns metadata-only projection. |
| CLI | `shark question withdraw <key>` and `supersede <key>` | Requires `--owner` and `--reason`; supersede also requires `--superseded-by`; returns metadata-only projection. |
| HTTP | `POST /api/v1/questions/{key}/workflow` | Accepts the workflow-configuration fields and returns `200` with metadata-only projection. |
| HTTP | `POST /api/v1/questions/{key}/response` | Accepts the response operation fields and returns `200` with metadata-only projection. |
| HTTP | `POST /api/v1/questions/{key}/resolve`, `/withdraw`, and `/supersede` | Accept the corresponding operation fields and return `200` with metadata-only projection. |
| Keyed dispatch | `shark next Q### --json` | Returns the existing complete `NextResponse` envelope for the one derived responder or `pause` for terminal/no-action states. |

Malformed JSON, unsupported fields, missing required input, invalid state,
wrong owner/responder, missing or stale claim session, invalid destination, and
terminal-state operations use the project's existing error shape and write
nothing. Generic `shark update`, Question create/update, normal list/search,
viewer, and F01 HTTP routes do not accept response or provenance fields.

#### REQ-F-007 — Publish I-02 v1 without owning downstream behavior

**Trace**: E39 interaction map I-02; research capability map, “Focused safe
reads” and “F03 scoped gate”.

I-02 v1 contains the validated `QuestionState` schema and bounds, the derived
first pending responder, the Question workflow state, bounded response
provenance, and validated resolution terminal state described by REQ-F-001
through REQ-F-005. F03 may read whether a Question remains open and F04 may
project the bounded state, but neither consumer may revalidate, alter, or
duplicate I-02 ownership.

I-02 does not contain `question_blocks`, a blocking predicate, focused
responder/owner read routes, response disclosure rules, X-06 integration
behavior, a parallel responder queue, or a host-side workflow runtime.

### Non-functional requirements

#### REQ-NF-001 — Keep Question state safe and bounded

Use byte-counted UTF-8 limits from REQ-F-001 before serialization and before
note/history writes. Do not index, render in viewer/search output, emit in
telemetry, or include complete response material in a blocked-work handoff.
Use parameterized repository methods and validate all service input before
state, note, history, or status mutation.

#### REQ-NF-002 — Preserve serial and transactional behavior

Use the existing unique `(entity_type, entity_key)` claim as the sole lease.
The response, Question-state update, typed note, and history record commit or
roll back together. A status conditional update protects a stale response or
resolution operation. Existing claim expiry/release behavior and all
non-Question workflow dispatch behavior remain unchanged.

#### REQ-NF-003 — Preserve compatibility and observability boundaries

The migration is additive, idempotent, and forward-corrective. It preserves
every F01 Question identity, base field, context value, generic association,
history row, and claim. Existing entity workflows, generic context operations,
relationships, search, and viewer projections keep their established behavior.
Errors remain actionable without including response content, prompts,
credentials, or full evidence pointers.

### Acceptance criteria

| ID | Scenario | Expected result |
|---|---|---|
| AC-001 | Create F01 Questions and initialize F02. | Every `draft` Question becomes `open`; base fields and generic associations remain unchanged. |
| AC-002 | Configure a Question and submit valid and invalid `QuestionState` partitions. | Only one configuration with ordered 1–10 distinct responders, derived current responder, bounded responses, and permitted provenance persists; every rejected partition writes nothing. |
| AC-003 | Dispatch a three-responder Question across claim, successful response, release, retry, expiry, and failed-response paths. | Keyed next exposes only the first pending responder; only a valid successful response plus release enables the next responder; all other paths leave it pending. |
| AC-004 | Attempt concurrent and stale response writes. | One matching active session can complete the current responder; stale, wrong-session, and conflicting retries do not mutate state, notes, history, or status. |
| AC-005 | Resolve each `resolution_kind` with valid and invalid destinations. | Only a ready Question with a valid owner/kind/destination reaches `resolved`; each accepted closure has note and history provenance. |
| AC-006 | Withdraw and supersede Questions in every eligible and terminal-state partition. | Valid operations write their terminal status and provenance atomically; invalid/self/missing supersession targets write nothing. |
| AC-007 | Exercise all F02 CLI and HTTP routes and their rejected partitions. | Each valid route returns the normal metadata projection; malformed, unsupported, unauthorized, or invalid-state input returns the existing error shape without mutation. |
| AC-008 | Run I-01 and I-02 contract tests plus the F02 forbidden manifest. | I-01 remains unchanged; TC-002 proves I-02; every F02-excluded gate/read/X-06 surface remains absent or rejects without mutation. |

### Out of scope and forbidden manifest

F02 excludes F01 registration and base-record persistence, F03's
`question_blocks` relationship and dispatch/advance predicate, F04's
open-by-responder and blocking-for query surfaces and disclosure policy, X-06
handoff/activation, any E38 adapter, parallel responders, transcripts,
unbounded context, and all linked-work mutation.

The `f02-forbidden-v1` manifest includes `QuestionBlocker`, `question_blocks`,
`/open-by-responder`, `/blocking-for`, advance/dispatch gate hooks, E38 Question
adapters, a second claim or queue table/service, response material in search,
viewer, telemetry, or blocked-work summaries, and any command/API that updates
a linked entity as a response side effect. Black-box checks and source/config
scans must prove each remains absent or rejects without mutation.

## Architecture

### Component changes

F02 extends the F01 Question model and service pattern. Cobra and HTTP handlers
parse and format only; the Question-domain service owns validation and the
repository owns parameterized storage. The complete planned file inventory is:

| Path group | Change |
|---|---|
| `internal/models/question.go` and `internal/models/context_data.go` | Add Question-owned state, responder, response, provenance types, byte-limit validation, and a serializer that preserves generic context fields without extending generic field mutation. |
| `internal/repository/question/repository.go` and `internal/repository/question/repository_test.go` | Add transactional Question-state/status read and conditional-update methods; retain parameterized SQL and base-record methods. |
| `internal/db/db.go` and `internal/db/db_test.go` | Add an idempotent F02 migration that converts Question `draft` to `open` and preserves predecessor data. |
| `internal/services/question_service.go` and `internal/services/question_service_test.go` | Add workflow configuration/state initialization, first-pending dispatch resolution, response, resolve, withdraw, supersede, provenance validation, and transaction orchestration. |
| `internal/services/question_repo_adapter.go`, `internal/services/claim_service.go`, and focused tests | Add only the interfaces needed to validate an existing lease/session; do not change generic claim lifecycle semantics. |
| `internal/cli/services_global.go`, `internal/cli/commands/question.go`, `internal/cli/commands/question_test.go`, `internal/cli/commands/run.go`, `internal/cli/commands/next.go`, `internal/cli/commands/status_group.go`, and their focused tests | Wire the Question state service into the existing Question command, keyed-next transitioner, placeholder generation, and status routing. |
| `internal/api/question_handler.go`, `internal/api/question_handler_test.go`, and `cmd/server/services.go` | Add the five finite F02 operation routes and service wiring. |
| `internal/sharkdata/default_data/workflow/question.yaml` and `internal/sharkdata/default_data/prompts/question/` | Replace the F01 fixture with the six-state Question workflow and the first-pending responder prompt. |
| `tests/contracts/e39_interactions_test.go` | Add TC-002 for I-02 and retain TC-001 unchanged. |

Implementation review must search the full Question integration surface for
additional closed type switches or direct status writes. A supported Question
caller discovered by that search belongs to this feature when it is required
to preserve I-02 behavior.

### Data model and migration

Continue to store the serialized state in `questions.context_data`; no new
table is required. Preserve non-Question generic context keys on decode and
encode, but reject Question-specific state through generic context setters.
The service owns decoding, validation, and serialization. The migration updates
only Questions with `status = 'draft'` to `open`; it must neither synthesize
responders nor overwrite `context_data`. A Question remains non-dispatchable
until a valid F02 state is established through the service-owned initialization
path.

### Interface contracts

`QuestionService` adds service-owned request types for workflow configuration,
response, resolution, withdrawal, and supersession. Each returns a
metadata-only `*models.Question`. Workflow configuration takes `key`,
`resolutionOwner`, and ordered responders. The response operation takes `key`,
`sessionID`, `responder`, `summary`, and `evidencePointer`; its resolution
operation takes `key`, `owner`, `kind`, and `pointer`. Withdrawal takes `key`,
`owner`, and `reason`; supersession adds `supersededBy`. Repository calls stay
typed and accept `context.Context`; they do not accept raw JSON or transport
request types.

The existing `runner.EntityTransitioner` remains the keyed-dispatch seam. The
Question placeholder generator reads only validated state and exposes the
current responder identity and Question metadata required by the bundled
prompt. It does not expose stored response summaries or evidence pointers.

### Key technical decisions

| Decision | Rationale |
|---|---|
| Store I-02 in a Question-owned validated value in `context_data`. | It reuses I-01 persistence while preventing generic metadata or `open_questions` from acting as an unbounded workflow schema. |
| Derive the current responder from ordered pending responders. | It removes contradictory independently supplied routing state and makes keyed dispatch deterministic. |
| Reuse the existing claim as the only lease. | `EntityClaim` already guarantees one active lease per type/key; a second queue or lease would duplicate lifecycle authority. |
| Commit response/provenance with Question state and audit records. | A partial response or closure would break serial routing and accountable resolution history. |
| Validate destinations before terminal resolution. | A response alone cannot establish an authoritative decision; validation prevents a resolved Question without the required durable record. |
| Keep F02 out of gate and focused-read paths. | I-02 supplies state to F03/F04 without changing the direct-only blocking predicate or disclosure boundary. |

### Integration with existing code

Follow `internal/services/question_service.go` for Question validation and
orchestration, `internal/repository/question/repository.go` for storage,
`internal/services/claim_service.go` for lease semantics, and
`internal/cli/commands/next.go` plus `internal/cli/commands/run.go` for the
normal keyed-dispatch envelope. Reuse `EntityHistory` and typed `EntityNote`
records for audit evidence. Do not add Question-specific behavior to
`ContextService`; it remains the F01 generic context seam.

## Cross-feature interactions

### Consumes: I-01 — Entity and platform registration (v1)

| Property | Contract |
|---|---|
| Producer | E39-F01 |
| Shape source | [E39-F01 I-01 v1 contract](../E39-F01-question-entity-and-platform-registration/spec.md#req-f-005--produce-the-narrow-i-01-v1-contract) |
| Consumed shape | `Q###`, registered type, persisted base record, `models.ContextData`, and generic adapters only |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-001` |
| Gate mode | live, as assigned by [the interaction map](../E39-interaction-map.md) |

### Produces: I-02 — Serial workflow and response provenance (v1)

| Property | Contract |
|---|---|
| Consumers | E39-F03 and E39-F04 |
| Shape source | [E39 architecture workflow and direct gate](../architecture.md#workflow-and-direct-gate) |
| Producer shape | Validated Question state, first pending responder, bounded evidence, and resolution kind/pointer from REQ-F-001 through REQ-F-005 |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-002` |
| Gate mode | live, as assigned by [the interaction map](../E39-interaction-map.md) |
| Closure owner | E39-F02 code-review owner |
| Required UAT evidence | UAT-02 serial responder dispatch and UAT-04 provenance closure, including I-02 contract evidence. |
| Reviewer and transition | Code reviewer verifies TC-002 and the named UAT evidence, records the I-02 live-gate review in F02 code-review evidence, then approves F02's normal workflow transition. |

## Cross-epic integrations

F02 produces, consumes, and validates no X-## row. X-06 is owned by E39-F04
and E38-F09; this feature supplies no E38 adapter, consumer handoff, or
cross-epic coverage.

## Verification plan

Run AC-001 through AC-008, I-01 TC-001, I-02 TC-002, the response/session
race tests, F02 migration fixtures, and `f02-forbidden-v1`. After Go changes,
run:

```text
make fmt
make lint
make test
```

RECOMMENDED OUTCOME: pass
