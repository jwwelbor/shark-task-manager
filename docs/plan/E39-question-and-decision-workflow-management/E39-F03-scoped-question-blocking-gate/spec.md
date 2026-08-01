---
feature_key: E39-F03
epic_key: E39
title: Scoped Question blocking gate specification
type: combined-spec
interaction_ids:
  - I-01
  - I-02
  - I-03
contract_version: i-03-v1
---

# Scoped Question blocking gate specification

**Feature**: [E39-F03](feature.md)  
**Parent contracts**: [Requirements](../requirements.md), [architecture](../architecture.md), [research report](research-report.md), [interaction map](../E39-interaction-map.md), and [UAT plan](../uat-plan.md)

This specification is incremental to the E39 PRD. It implements the direct
Question-only gate in parent architecture §“Workflow and direct gate”; it does
not restate the epic business context, serial-response lifecycle, or F04
focused read/disclosure surfaces.

## Requirements

### Functional requirements

#### REQ-F-001 — Define a distinct direct relationship

**Trace**: E39 REQ-F-005; architecture ADR-003; research capability map,
“Directed polymorphic relationships”.

Add `question_blocks` to the finite polymorphic relationship vocabulary. A
valid edge has `from_entity_type=question` and an eligible non-Question target:
`epic`, `feature`, `task`, `bug`, `change`, or `tech_debt`. Reverse edges,
Question targets, `sprint`, and `idea` targets are rejected before an
`entity_relationships` write. Existing structural validation, endpoint lookup,
duplicate detection, and unlink behavior apply unchanged.

`question_blocks` is directed and non-transitive. It is not a cyclic generic
dependency relationship and is not exposed as `depends_on` or `blocks`.
Existing `blocks`, `depends_on`, `linked_to`, `references`, and every other
relationship retain their current validation and planning behavior.

#### REQ-F-002 — Qualify one compact, read-only blocker

**Trace**: E39 REQ-F-005 and REQ-F-009; architecture §“Workflow and direct
gate”; research capability map, “Compact Question handoff”.

Provide `QuestionBlocker.Check(ctx, candidateType, candidateKey)` as the sole
gate predicate. It resolves the candidate through the registered entity
repository, reads only incoming `question_blocks` edges, then reads each source
Question through the F01 adapter/service seam. A source qualifies only when it
is a Question with `blocking=true` and an I-02 open workflow state (`open` or
`answering`). `draft`, `ready_for_resolution`, `resolved`, `withdrawn`, and
`superseded` do not qualify. Claim presence, responder completion, a generic
relationship, and a merely related Question do not independently qualify.

When more than one source qualifies, choose the earliest relationship
`created_at`, breaking an equal timestamp by ascending relationship ID. The
result is a stable I-03 v1 compact handoff containing exactly:

| Field | Source | Constraint |
| --- | --- | --- |
| `question_key` | F01 Question identity | Canonical `Q###` |
| `summary` | F01 Question summary | Existing bounded summary only |
| `resolution_owner` | I-02 QuestionState | Empty only when the persisted F02 state is absent or invalid, which is an actionable error rather than a qualifying handoff |
| `current_responder` | I-02 derived first pending responder | Empty is allowed at an otherwise open Question; it is not inferred from claims |

The predicate does not create a claim, status/history/note row, relationship,
work session, or search update. It does not resolve, withdraw, supersede,
respond to, or change the linked candidate or source Question.

#### REQ-F-003 — Stop keyed dispatch before dispatch work

**Trace**: E39 REQ-F-005; J-03; architecture §“Workflow and direct gate”.

Apply `QuestionBlocker.Check` to every resolved candidate in `shark next`,
including the root and each recursively selected cascade child. The check occurs
after identity/status read sufficient to name the candidate, but before
placeholder generation, action population, prompt assembly, auto-advance,
or any lease acquisition. A qualifying result returns the normal `NextResponse`
identity and current status with `action="pause"`, an empty agent/provider/model/
effort/prompt set, and a `question_block` I-03 v1 field. It must not leak
response summaries, evidence pointers, Question context, or relationship IDs.

No result preserves existing keyed-next behavior byte-for-byte except for the
absence of the optional `question_block` field. A paused direct candidate is
not dispatched; a paused cascade child is not a live candidate and existing
cascade fall-through continues to an unrelated eligible sibling. The parent
entity itself is not changed merely because a child is blocked.

#### REQ-F-004 — Keep direct `run` and cascade parity

**Trace**: E39 REQ-F-005 and REQ-F-009; research finding 6.

Apply the same predicate before top-level `shark run` and every in-process
cascade child preflight. A qualifying result returns the runner’s normal
paused result with the same I-03 compact handoff and creates no lease, worker
process, prompt, heartbeat, release, or transition. This rule applies to both
dry-run and normal run. No result preserves the existing action-before-lease
ordering and Question responder handling introduced by F02.

#### REQ-F-005 — Reject supported linked-work advancement atomically

**Trace**: E39 REQ-F-005 and REQ-F-009; UAT-03; architecture §“Workflow and
direct gate”.

Before `shark status advance` commits a supported non-Question entity
transition (`epic`, `feature`, `task`, `bug`, `change`, or `tech_debt`), call
the same blocker. A match returns a typed, actionable Question-blocked error
that contains only the I-03 handoff and performs no candidate status/history,
claim, Question, relationship, or work-session write. Global JSON error output
serializes the same compact handoff; human output names the candidate and
Question key/summary without printing hidden Question material.

Question transitions are excluded so a blocking Question can continue through
its own configured response/resolution lifecycle. Generic `blocks` behavior,
direct service calls outside the supported command boundary, and automatic
Question lifecycle operations are unchanged.

#### REQ-F-006 — Publish I-03 v1 without F04 behavior

**Trace**: E39 interaction map I-03; parent architecture delivery boundary.

I-03 v1 is the direct-only predicate plus the compact handoff described in
REQ-F-002. It is produced by F03 for F04. F03 neither adds `open-by-responder`
or `blocking-for` read routes, nor authorizes full Question disclosure, nor
creates an E38 host adapter. A handoff is an ephemeral dispatch/advance result,
not copied state in `docs/council/`, a queue, context data, or telemetry.

### Non-functional requirements

#### REQ-NF-001 — Preserve security and disclosure boundaries

Use parameterized repository queries and validate all relationship direction
and target-type rules before mutation. The gate exposes only the four I-03
fields; it never serializes `question_state`, response content, evidence
pointers, requester details, claims, relationship IDs, prompts, or credentials.
All no-match and rejection paths retain existing error conventions.

#### REQ-NF-002 — Bound lookup cost and preserve concurrency behavior

Each check performs one candidate resolution, one incoming edge query filtered
to `question_blocks`, and bounded source Question reads. It does not walk
outgoing edges or recurse through the graph. The result remains advisory until
the immediately following guarded transition; the transition path checks before
its commit and does not claim lock ownership. Existing claim and advance-guard
rules remain authoritative.

#### REQ-NF-003 — Preserve compatibility and recoverability

This feature adds no database table. It adds one additive SQLite migration that
widens the existing `entity_relationships` relationship-type CHECK to accept
`question_blocks`. The migration preserves existing relationship rows, indexes,
dependent views, and the Question cleanup trigger before recording its schema
version. Generic dependency behavior, F01 records, F02 state/provenance,
non-Question dispatch, and unlinked transition behavior retain their current
contracts.
Failures in relationship or Question reads are actionable errors and write
nothing; they never degrade into an unsafe dispatch or transition.

### Acceptance criteria

| ID | Scenario | Expected result |
| --- | --- | --- |
| AC-001 | Create/unlink every direction and target-type partition for `question_blocks`. | Only `Question -> eligible non-Question` edges persist; invalid direction/target writes nothing; existing types remain valid. |
| AC-002 | Check candidates against all predicate partitions. | Only a direct, open/answering, `blocking=true` Question returns I-03; generic blocks, indirect edges, false blocking, unlinked, draft, ready, and terminal sources do not. |
| AC-003 | Give one candidate multiple qualifying Questions with equal and unequal timestamps. | The deterministic oldest edge/lowest-ID winner is returned with exactly the four compact fields. |
| AC-004 | Run direct keyed `next` and all cascade child paths for qualifying and control candidates. | A match pauses before placeholders/prompts/leases; a blocked child falls through to an unrelated live sibling; no match retains ordinary output. |
| AC-005 | Run direct and cascade `run` in normal and dry-run modes. | A match returns paused without worker/lease/heartbeat/release/transition side effects; no match retains F02 action-before-lease parity. |
| AC-006 | Advance every supported non-Question entity with a qualifying blocker, then repeat after Question closure. | The first attempt rejects atomically with compact I-03; closure clears the predicate and the normal advance succeeds. |
| AC-007 | Attempt Question self-transition, generic `blocks`, unrelated candidates, and every F01/F02 lifecycle operation while a gate edge exists. | Only the directly linked non-Question dispatch/advance boundary changes; all excluded behavior is unchanged. |
| AC-008 | Execute I-01, I-02, I-03, relationship, dispatch/run/cascade, status, and no-write regression suites. | I-01/I-02 remain live-compatible and TC-003 proves I-03 without disclosure or legacy-block drift. |

### Out of scope and forbidden manifest

F03 excludes F04 focused lookup routes, authorization policy changes, full
Question details in blocked results, response/provenance changes, global or
transitive graph blocking, generic `blocks` reinterpretation, Question
auto-resolution, linked-work mutation, new queues, a scheduler, E38-F09
activation, telemetry payload changes, and schema migrations other than the
approved additive relationship-vocabulary migration.

The `f03-forbidden-v1` manifest requires source/configuration and black-box
coverage for: `QuestionBlocker` writes; `question_blocks` reverse, Question,
Sprint, and Idea targets; recursive edge traversal; response/evidence/context
fields in `NextResponse.question_block`; `open-by-responder`; `blocking-for`;
an E38 adapter; and any automatic source/candidate status, claim, history,
note, work-session, or relationship mutation from a gate check.

## Architecture

### Component changes

| Path | Change |
| --- | --- |
| `internal/models/entity_relationship.go` and tests | Add the `EntityRelQuestionBlocks` constant, valid-type inventory, and non-cyclic classification. |
| `internal/db/db.go` and migration tests | Add the approved relationship-vocabulary migration; preserve existing rows, indexes, dependent views, and the Question cleanup trigger across upgrade. |
| `internal/services/entity_relationship_service.go` and tests | Enforce the Question-source and eligible-target policy in create/unlink validation without changing generic relationship rules. |
| `internal/services/question_blocker.go` and tests | Add the read-only checker, I-03 handoff model, deterministic selection, typed blocked error, and narrow repository/Question/registry interfaces. |
| `internal/cli/services_global.go` | Wire `QuestionBlocker` from existing entity-relationship repository, F01 Question service, entity registry, and workflow-state reader. |
| `internal/cli/commands/next.go` and tests | Add optional `NextResponse.question_block`; check root and cascade candidates before placeholder/action/prompt work and return compact pause. |
| `internal/cli/commands/run.go` and tests | Inject the checker into top-level and cascade preflight before lease/runner creation; preserve no-match action ordering. |
| `internal/cli/commands/status_group.go` and tests | Gate supported non-Question `status advance` immediately before dispatching its transition; render the compact typed error. |
| `internal/cli/commands/link.go` and tests | Advertise and validate `question_blocks` through the existing generic link/unlink surface. |
| `internal/api/viewer/mutation_service.go`, `mutation_handler.go`, and tests | Preserve the same service direction validation for existing generic relationship mutations; add accepted/rejected payload coverage. |
| `internal/viewer/server/wire.go` and tests | Reuse the normal relationship service wiring; no new HTTP read route is added. |
| `tests/contracts/e39_interactions_test.go` | Add `TC-003` for I-03 and regress I-01/I-02. |

Implementation must also update adjacent table-driven tests in the listed
packages whenever their relationship type inventories or `NextResponse` exact
JSON fixtures enumerate the closed vocabulary.

### Data model and interface contracts

`entity_relationships` remains the sole persistence table. `question_blocks`
uses the existing five relationship columns and unique edge constraint. The
approved migration widens only its relationship-type CHECK; it adds no new
column, index, cache, or behavior beyond preserving the existing indexes,
dependent views, and Question cleanup trigger during rebuild.

`QuestionBlocker` receives read-only interfaces for incoming relationships,
registered candidate resolution, and Question retrieval. It returns either no
match, an I-03 handoff, or an error. It has no claim, transition, note,
history, search-index, or write-repository dependency. The error type wraps a
handoff rather than a mutable Question model so status and CLI callers cannot
accidentally disclose or mutate source state.

The `NextResponse.question_block` JSON member is omitted on no match. On a
match, `action` is `pause`; `question_block` has exactly `question_key`,
`summary`, `resolution_owner`, and `current_responder`. It is the shared
producer/consumer shape, not an additional action verb. The runner’s paused
result and status-advance JSON error use the same names and values.

### Key technical decisions

| Decision | Rationale |
| --- | --- |
| Add a distinct relationship type rather than reuse `blocks`. | Parent ADR-003 requires preservation of generic dependency semantics and direct Question qualification. |
| Centralize qualification in `QuestionBlocker`. | F02 state semantics must not be copied into next, run, cascade, and status callers. |
| Gate before placeholders, action rendering, lease acquisition, and transition commit. | It prevents responder derivation, prompt leakage, claims, or writes for work that must merely stop. |
| Return an optional compact field, not a full Question model. | I-03 needs safe ownership handoff; F04 alone owns focused reads and disclosure. |
| Select deterministically when several direct blockers qualify. | The runner/CLI need reproducible output without converting the gate into a global queue. |
| Add one vocabulary-only migration; do not add a cache. | SQLite enforces the finite relationship vocabulary in a CHECK, so `question_blocks` needs an additive upgrade that preserves existing durable structure. |

### Integration with existing code

The service follows the existing dependency-injected service pattern in
`internal/services/entity_relationship_service.go` and
`internal/services/question_service.go`; commands remain thin wrappers.
`next.go`’s `resolveEntity` is the canonical keyed-dispatch seam, including
`tryCascadeCandidates`; `run.go`’s `acquireRunLeaseForRunnableAction` is the
matching pre-lease seam; `status_group.go`’s `dispatchTransition` remains the
single CLI transition routing point. F03 extends those seams and must not
duplicate their workflow, claim, or prompt assembly logic.

## Cross-feature interactions

### Consumes: I-01 — Entity and platform registration

| Property | Contract |
| --- | --- |
| Producer | E39-F01 |
| Shape source | [E39-F01 I-01 v1 contract](../E39-F01-question-entity-and-platform-registration/spec.md#produces-i-01--entity-and-platform-registration-v1) |
| Consumer use | Resolve canonical `Q###`, registered Question adapter, base `blocking` and `summary` fields. |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-001` |
| Gate mode | live |

### Consumes: I-02 — Serial workflow and response provenance

| Property | Contract |
| --- | --- |
| Producer | E39-F02 |
| Shape source | [Architecture workflow](../architecture.md#workflow-and-direct-gate) |
| Consumer use | Determine open state, resolution owner, and first pending responder without changing state or provenance. |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-002` |
| Gate mode | live |

### Produces: I-03 — Scoped relationship gate

| Property | Contract |
| --- | --- |
| Consumers | E39-F04 |
| Shape source | [Architecture direct gate](../architecture.md#workflow-and-direct-gate) |
| Producer shape | Direct qualifying `question_blocks` predicate and the four-field I-03 v1 compact handoff. |
| Shared contract test | `tests/contracts/e39_interactions_test.go#TC-003` |
| Gate mode | live |
| Closure owner | E39-F03 code-review owner |
| Required UAT evidence | UAT-03: a linked owner pauses with compact safe handoff while an unlinked owner remains eligible; resolving the Question clears the gate without linked-work mutation. |

## Cross-epic integrations

F03 produces, consumes, and validates no X-## row. X-06 remains owned by
E39-F04 and E38-F09. F03 supplies I-03 to F04 only and adds no E38 adapter,
consumer activation, or cross-epic test claim.

## Verification plan

Run AC-001 through AC-008, `TC-001`, `TC-002`, `TC-003`, and
`f03-forbidden-v1`. Capture UAT-03 using a temporary SQLite database with one
open blocking Question linked directly to one candidate and an unlinked control
candidate. Record the compact pause and rejected advance, then resolve the
Question and show the candidate’s normal path resumes. Compare Question,
candidate, relationship, claim, history, note, and work-session row counts
before and after every blocked operation. Redact no Question material because
the evidence must contain only the compact handoff.

After Go changes, run:

```text
make fmt
make lint
make test
```

RECOMMENDED OUTCOME: pass
