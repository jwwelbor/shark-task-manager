---
research_schema: 2
entity_key: E39-F02
entity_type: feature
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

# E39-F02 Research Report: Serial Question Workflow and Resolution Provenance

## Scope

E39-F02 consumes I-01, the registered `question`/`Q###` record, and produces
I-02: validated `question_state`, first-pending responder routing, bounded
response recording, and explicit resolution provenance. It extends the
existing route-based keyed-dispatch and one-active-claim contracts; it does not
implement `question_blocks` or dispatch/advance gating (F03), focused query or
disclosure policy (F04), a parallel queue, or any mutation of linked work.

Terms used here follow the parent [research report](../research-report.md):
**Question**, **question_state**, **current responder**, **resolution owner**,
**evidence pointer**, **resolution kind**, and **authoritative destination**.
The parent classifies E39 as complex and its architecture assigns F02 the
serial workflow/provenance boundary, so this report uses COMPLEX rigor.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md`, parent `epic.md`, and `requirements.md` REQ-F-002 through REQ-F-004 and REQ-F-007 through REQ-F-009 define I-02, serial response, bounded state, provenance, and the no-linked-work-mutation boundary.
- [x] `affected_implementation_or_contract` — Evidence: `internal/models/context_data.go`, `internal/services/context_service.go`, `internal/models/entity_claim.go`, `internal/services/claim_service.go`, `internal/cli/commands/next.go`, and `internal/sharkdata/default_data/workflow/{task,epic}.yaml` establish the current context, lease, keyed-dispatch, and route-workflow contracts that I-02 extends.
- [x] `related_work` — Evidence: parent `research-report.md`, `architecture.md`, `E39-interaction-map.md`, F01–F04 `feature.md` files, and `E39-F01-question-entity-and-platform-registration/research-report.md` define I-01 as F02's prerequisite and I-02 as the F03/F04 consumer contract.
- [x] `pattern_contract` — Evidence: `internal/services/context_service.go` validates and serializes shared context through the entity registry; `internal/services/claim_service.go` and `internal/models/entity_claim.go` keep a type/key lease independent of workflow phase; `internal/cli/commands/next.go` is the keyed dispatch boundary; route YAML files express per-level state/outcome routing.
- [x] `dependency_impact` — Evidence: `internal/models/entity_relationship.go` and `internal/repository/entityrel/repository.go` provide directed polymorphic links but no resolution semantics; `internal/services/viewer_service.go` consumes typed entities for normal reads; F03 and F04 consume I-02 without owning its state schema.
- [x] `cross_boundary_risks` — Evidence: the claim model's unique `(entity_type, entity_key)` lease and the parent architecture's `shark next Q### --json` contract show that responder completion must be a validated Question-domain write, never a claim expiry/release side effect or an alteration of the host dispatch envelope.
- [x] `alternatives` — Evidence: `internal/models/context_data.go` exposes only untyped `open_questions` strings and metadata, while the parent `scope.md`, `requirements.md`, and `architecture.md` reject transcript storage, generic metadata, parallel responders, global blocking, and linked-work worker transitions.

## Capability map

| Capability | Evidence | Decision | F02 responsibility |
| --- | --- | --- | --- |
| I-01 registered Question record and generic adapters | F01 `feature.md`; F01 research report; E39 interaction map | REUSE | Require the registered record and normal keyed dispatch support; do not repeat F01 registration or persistence inventory. |
| Shared JSON context | `models.ContextData`; `ContextService` | EXTEND | Add a Question-specific validated `question_state`; do not encode responder routing in `open_questions` or generic metadata. |
| Route-based workflow and keyed `next` | `next.go`; task and epic workflow YAML; parent architecture | EXTEND | Define Question states/outcomes that expose only the first pending responder while preserving the normal `NextResponse` envelope. |
| One-active claim lease | `EntityClaim`; `ClaimService`; parent research report | REUSE | Use the existing type/key session lease. Claim, expiry, and release never complete a responder or resolve a Question. |
| Typed notes/history and authoritative records | Parent requirements REQ-F-007 to REQ-F-009; parent architecture | EXTEND | Validate resolution kind and destination before consequential closure; write bounded history/note provenance, leaving linked entities unchanged. |
| Generic relationships | `EntityRelationship`; entity relationship repository | REUSE | Accept links/pointers as evidence where required, but neither define `question_blocks` nor infer a blocking predicate; those belong to F03. |
| Focused safe reads and external handoff | F04 `feature.md`; E39 interaction map | EXTEND (producer contract) | Produce stable, bounded I-02 state and provenance for F04 to project; do not implement responder/owner disclosure queries. |
| F03 scoped gate | F03 `feature.md`; E39 interaction map | CONTRADICTS | Do not stop or advance linked work from any response or resolution path. |
| Parallel collection, transcripts, or a host queue | Parent scope and architecture | CONTRADICTS | Exclude them from the first release; they break bounded state, serial safety, or existing Rider authority. |

## Findings

1. **The F02 state must be domain-validated, not a generic-context field.**
   `ContextData` currently accepts `OpenQuestions []string` and arbitrary
   metadata, and `ContextService` merges then serializes those values through a
   generic registry. Neither surface encodes ordered responder identity,
   completion, bounded answers, evidence pointers, or closure readiness. I-02
   therefore needs a Question-owned schema/validator and bounded write path.

2. **Seriality is already supported by the lease boundary, but completion is
   not automatic.** `EntityClaim` permits one lease per entity type/key and
   `ClaimService` treats status as a separate phase with expiry/reclaim and
   release operations. F02 must validate and persist a successful response
   transactionally before marking that responder complete; a failed, expired,
   or released lease leaves the responder pending. The next responder becomes
   eligible only after the successful holder releases the existing lease.

3. **Use a Question workflow, not a second queue or altered host loop.** The
   existing workflow YAML route model and `shark next <key> --json` provide the
   extension seam. The Question workflow should represent the architecture's
   `open`, `answering`, `ready_for_resolution`, terminal resolution/withdrawal/
   supersession boundaries and render one current-responder prompt. It must not
   change the parent-owned claim/transition responsibility or dispatch another
   entity as a side effect.

4. **Resolution requires classified provenance, not merely a response.** The
   parent contract requires a resolution kind and, for consequential outcomes,
   a durable destination: feature specification, product decision log, ADR and
   references, or linked Shark work. F02 must make that validation precede the
   resolution transition and preserve the relevant concise history/note
   evidence. A linked work item is a pointer, never authorization for F02 to
   claim, update, or advance it.

5. **I-02 must be a narrow producer contract.** F01 supplies the durable
   record; F03 consumes state while owning the direct-only blocking predicate;
   F04 consumes state while owning safe, focused query/disclosure behavior.
   The parent architecture proposes concrete limits (1–10 distinct responders,
   1,000-byte summaries, and 2,048-byte evidence pointers), but implementation
   should keep them as the explicit schema contract and cover rejection of
   prompts, credentials, transcripts, and oversized content.

## Decisions

1. **Proceed with a dedicated Question state service/validator over registered
   I-01 storage.** Model ordered responders, completed responses, a derived
   current responder, resolution owner, resolution kind, and destination
   pointer; validate these before persistence or transition.

2. **Reuse the existing claim as the only lease.** Couple successful response
   persistence to Question-domain transition/history in one service-owned
   transaction where possible, but never infer completion from claim lifecycle
   events.

3. **Keep keyed dispatch stable and serial.** `shark next Q### --json` remains
   the canonical handoff and returns only the first pending responder. No
   parallel responder queue, new host runtime, or linked-work workflow action
   is introduced.

4. **Gate closure on provenance.** Require a supported `resolution_kind` and
   validate the authoritative destination appropriate to that kind before
   resolving; record withdrawal and supersession explicitly too.

5. **Publish I-02 as bounded producer state.** F03/F04 may consume it but must
   not duplicate its validation or change its ownership. Any unresolved choice
   over exact role authorization and API/viewer shape remains for the design
   and downstream-surface decisions, not an implicit F02 behavior change.

## Sources

- `docs/plan/E39-question-and-decision-workflow-management/E39-F02-serial-question-workflow-and-resolution-provenance/feature.md`
- `docs/plan/E39-question-and-decision-workflow-management/epic.md`, `requirements.md`, `scope.md`, `research-report.md`, `architecture.md`, and `E39-interaction-map.md`
- `docs/plan/E39-question-and-decision-workflow-management/E39-F01-question-entity-and-platform-registration/{feature.md,research-report.md}`
- `docs/plan/E39-question-and-decision-workflow-management/E39-F03-scoped-question-blocking-gate/feature.md` and `E39-F04-focused-question-read-surfaces-and-consumer-handof/feature.md`
- `internal/sharkdata/default_data/research/recipes.yaml`
- `internal/models/{context_data,entity_claim,entity_note,entity_relationship}.go`
- `internal/services/{context_service,claim_service,viewer_service}.go`
- `internal/repository/entityrel/repository.go`
- `internal/cli/commands/next.go`, `internal/cli/services_global.go`, `internal/keys/service.go`, and `internal/db/db.go`
- `internal/sharkdata/default_data/workflow/{task,epic}.yaml`

RECOMMENDED OUTCOME: pass
