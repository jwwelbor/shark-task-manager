---
research_schema: 2
entity_key: E39
entity_type: epic
recipe: universal
rigor: complex
categories:
  - backend
  - api
  - data
  - workflow_operations
  - documentation
related_work: true
---

# E39 research report: Question and Decision Workflow Management

## Scope

E39 introduces a first-class, top-level Question entity for accountable,
serially answered questions. It must preserve the existing ownership boundary:
the Question has its own workflow and claim, and responding to it must not
claim, advance, or resolve linked work. A Question may stop only an explicitly
linked entity when it remains open and explicitly blocking.

This is COMPLEX research. The change adds a persisted entity and crosses entity
registration, key parsing, workflow dispatch, claim handling, context,
relationships, history, search, CLI/API, viewer, and advancement behavior.

Terms used in this report are: **Question** (the new durable coordination
entity), **requester**, **requested responder**, **resolution owner**,
**question_state** (bounded structured routing and response state), **blocking
Question** (an open Question with an explicit blocking designation and link),
and **authoritative destination** (the durable record required for a
consequential resolution).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E39-question-and-decision-workflow-management/epic.md`, `requirements.md`, `scope.md`, `personas.md`, and `user-journeys.md` define the Question lifecycle, serial responder, scoped gate, and provenance vocabulary.
- [x] `affected_implementation_or_contract` — Evidence: `internal/models/entity_note.go`, `internal/cli/services_global.go`, `internal/keys/service.go`, `internal/services/context_service.go`, `internal/cli/commands/next.go`, and `internal/sharkdata/default_data/workflow/epic.yaml` show a closed entity-type contract across models, registry, keys, context, dispatch, and workflow configuration.
- [x] `related_work` — Evidence: `docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/feature.md`, `docs/plan/E38-shark-attack-team-orchestration/E38-interaction-map.md`, and E39's parent documents establish E38-F09 as a blocked future consumer and distinguish its host-adapter scope from E39's platform lifecycle.
- [x] `pattern_contract` — Evidence: `internal/services/claim_service.go`, `internal/repository/entityrel/repository.go`, `internal/models/entity_relationship.go`, `internal/models/context_data.go`, and `internal/sharkdata/default_data/workflow/task.yaml` establish existing claim, relationship, context, and route-based workflow patterns that E39 must extend.
- [x] `dependency_impact` — Evidence: `internal/cli/commands/next.go`, `internal/cli/services_global.go`, `internal/services/viewer_service.go`, `internal/keys/service.go`, and `internal/db/db.go` identify dispatch, registry, viewer, key routing, and persistence as direct consumers of the entity-type set.
- [x] `cross_boundary_risks` — Evidence: `internal/services/claim_service.go` separates the lease from workflow phase, while `internal/cli/commands/next.go` emits the keyed-dispatch contract; a Question gate that mutates linked work or changes that contract would violate both boundaries.
- [x] `alternatives` — Evidence: `docs/plan/E39-question-and-decision-workflow-management/scope.md` excludes notes/transcripts, global blocking, worker-owned linked-work mutation, and parallel responders; `internal/models/context_data.go` shows that existing `open_questions` is only an untyped string list, not a lifecycle record.

## Capability map

| Capability | Brownfield evidence | Decision | E39 responsibility |
|---|---|---|---|
| Polymorphic entity model, repository registry, and shared services | `internal/models/entity_note.go`; `internal/services/entity_registry.go`; `internal/cli/services_global.go` | EXTEND | Register `question` everywhere a supported entity type is enumerated, with a model, repository adapter, service wiring, and deletion/migration support. This is a new entity capability built on existing polymorphism. |
| Key routing and CLI/API entity resolution | `internal/keys/service.go`; `internal/cli/commands/next.go` | EXTEND | Design a Question key prefix and route it through key parsing and all generic command/API dispatch points. Do not overload an existing epic, task, bug, or change-card key. |
| Route-based workflow and keyed dispatch | `internal/sharkdata/default_data/workflow/epic.yaml`; `internal/sharkdata/default_data/workflow/task.yaml`; `internal/cli/commands/next.go` | EXTEND | Add a Question workflow and prompt set that selects only the first pending responder. Preserve `shark next <key> --json` as the sole keyed-dispatch contract. |
| One-active-claim lease lifecycle | `internal/services/claim_service.go`; `internal/models/entity_claim.go` | REUSE | Use the existing claim key/type lease for one Question claimant. Keep responder completion as a Question-domain transition after valid bounded output, never as a side effect of claim expiry or release. |
| Structured context | `internal/models/context_data.go`; `internal/services/context_service.go` | EXTEND | Add validated `question_state` support with explicit size and content limits. Existing `open_questions` and free-form metadata do not satisfy serial routing, response provenance, or secret/transcript rejection. |
| Cross-entity relationships | `internal/models/entity_relationship.go`; `internal/repository/entityrel/repository.go`; `internal/db/db.go` | EXTEND | Reuse directed polymorphic links, but decide the exact blocking relationship direction and the open/blocking predicate. Avoid treating every existing `blocks` relation as an E39 Question gate. |
| Notes, history, and authoritative resolution record | `internal/models/entity_note.go`; `internal/db/db.go`; E39 `requirements.md` REQ-F-007 to REQ-F-009 | EXTEND | Reuse typed notes and history for auditability; validate `resolution_kind` and a durable destination before consequential resolution. |
| Search, viewer, query, and reporting surfaces | `internal/services/viewer_service.go`; `docs/plan/E39-question-and-decision-workflow-management/requirements.md` REQ-F-001 and REQ-F-006 | EXTEND | Add focused open-by-responder and blocked-work views plus normal retrieve/list/search behavior. The exact CLI/API/viewer form remains a design decision. |
| E38-F09 provider-neutral live-question handling | `docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/feature.md`; E39 epic note dated 2026-07-30 | REUSE (consumer) | E39 supplies the generic Question lifecycle. Do not repair F09 adapters or continuation behavior, and do not add a separate council queue or host runtime. |
| Notes-only coordination, global blocking, parallel collection, or linked-work worker transitions | E39 `scope.md`; E39 `requirements.md` REQ-F-003, REQ-F-004, REQ-F-005, and REQ-F-009 | CONTRADICTS | Exclude these alternatives from the first release because they break the stated durable lifecycle, scoped gate, or existing authority/claim boundary. |

## Findings

1. Shark has strong reusable generic foundations, but `question` is not a
   supported entity type. `ValidEntityTypes`, the startup `EntityRegistry`, key
   parsing, workflow files, and viewer/service switches enumerate known types.
   E39 is therefore a new entity capability, not a context-only extension.

2. Existing claim behavior already provides the required one-active-holder
   safety. A lease uses `entity_type` and `entity_key`, auto-reclaims expired
   claims, and does not itself change workflow status. The Question service
   must keep responder completion and resolution explicit so release, failure,
   or expiry cannot silently advance a responder.

3. Existing context supports `open_questions` as `[]string` and arbitrary
   metadata, but it does not validate a Question schema or content bounds.
   E39 needs a dedicated validated `question_state`; putting prompts,
   transcripts, credentials, or long responses there would violate the epic's
   safety requirement. Typed notes and authoritative document pointers remain
   the durable destinations for long-form evidence.

4. The relationship store is polymorphic and has an existing `blocks` type,
   but generic relationships alone cannot implement E39's gate. The gate must
   query a Question-specific open-and-blocking predicate plus an explicit link
   to the candidate being dispatched or advanced. It must return a compact
   summary and leave unrelated work eligible.

5. `shark next <key> --json` is the keyed dispatch boundary. It returns the
   rendered prompt and entity status to the host loop; it does not grant a
   worker authority over other entities. E39 must integrate its responder
   routing inside this contract and preserve parent-owned claims and
   transitions.

6. E38-F09 is a direct consumer dependency but not implementation scope.
   Its feature record describes provider-neutral, live question routing and is
   hard-blocked on E39. E39 should expose a generic, durable platform
   capability that F09 may consume later; it should not assume F09's unresolved
   adapter design or copy its host-specific behavior.

7. The requested API/viewer/search/reporting surface, role authorization,
   exact key prefix, workflow labels, relationship direction, context limits,
   and gate timing remain open design decisions. Research establishes their
   extension points and non-negotiable invariants; it does not settle them.

## Decisions

1. **Proceed as a new top-level Question entity that extends shared platform
   contracts.** Design must inventory every closed entity-type switch,
   persistence table, migration/trigger, registry registration, generic
   command, search index, viewer projection, and workflow route before feature
   decomposition.

2. **Reuse the current claim lifecycle for serial safety.** The Question
   workflow must make exactly one pending responder dispatchable. Successful,
   validated response recording may make the next responder eligible only after
   the current claim is released; failed, expired, or released claims do not
   mark a responder complete.

3. **Create a validated `question_state` contract rather than reuse
   `open_questions` or unbounded metadata.** Design must specify its fields,
   maximum response/evidence sizes, rejected content classes, migration shape,
   and tests. Long evidence belongs in typed notes or the authoritative
   destination and is referenced by pointer.

4. **Implement a narrow Question gate at dispatch/advancement boundaries.** It
   must evaluate only an open, explicitly blocking Question directly linked to
   the candidate. It must not claim, transition, or otherwise mutate linked
   work, and it must not impose a global block.

5. **Require resolution provenance before consequential closure.** Design must
   define `resolution_kind`, which kinds require an authoritative destination,
   permitted destination forms, and the history/note evidence written when a
   Question resolves, withdraws, or is superseded.

6. **Decompose by platform boundary, then create the required interaction
   map.** At minimum, separate entity/persistence registration, Question
   workflow and claim/context behavior, scoped gate/relationship behavior, and
   focused read surfaces. Because this is a multi-feature producer/consumer
   change, design must create `E39-interaction-map.md` before implementation
   and assign I-## identifiers only to resolved architecture sections.

## Sources

- `docs/plan/E39-question-and-decision-workflow-management/epic.md`
- `docs/plan/E39-question-and-decision-workflow-management/requirements.md`
- `docs/plan/E39-question-and-decision-workflow-management/scope.md`
- `docs/plan/E39-question-and-decision-workflow-management/personas.md`
- `docs/plan/E39-question-and-decision-workflow-management/user-journeys.md`
- `docs/plan/E39-question-and-decision-workflow-management/success-metrics.md`
- `internal/sharkdata/default_data/research/recipes.yaml`
- `internal/models/entity_note.go`, `entity_relationship.go`, `entity_claim.go`, and `context_data.go`
- `internal/services/entity_registry.go`, `context_service.go`, `claim_service.go`, and `viewer_service.go`
- `internal/repository/entityrel/repository.go`
- `internal/cli/services_global.go` and `internal/cli/commands/next.go`
- `internal/keys/service.go` and `internal/db/db.go`
- `internal/sharkdata/default_data/workflow/epic.yaml` and `task.yaml`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/feature.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-interaction-map.md`

RECOMMENDED OUTCOME: pass
