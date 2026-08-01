---
research_schema: 2
entity_key: E39-F01
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
source_set:
  - docs/plan/E39-question-and-decision-workflow-management/E39-F01-question-entity-and-platform-registration/feature.md
  - docs/plan/E39-question-and-decision-workflow-management/research-report.md
  - docs/plan/E39-question-and-decision-workflow-management/requirements.md
  - docs/plan/E39-question-and-decision-workflow-management/scope.md
  - docs/plan/E39-question-and-decision-workflow-management/architecture.md
  - docs/plan/E39-question-and-decision-workflow-management/E39-interaction-map.md
  - docs/plan/E39-question-and-decision-workflow-management/E39-F02-serial-question-workflow-and-resolution-provenance/feature.md
  - docs/plan/E39-question-and-decision-workflow-management/E39-F03-scoped-question-blocking-gate/feature.md
  - docs/plan/E39-question-and-decision-workflow-management/E39-F04-focused-question-read-surfaces-and-consumer-handof/feature.md
  - docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/feature.md
  - internal/models/entity_note.go
  - internal/models/context_data.go
  - internal/models/tech_debt.go
  - internal/services/entity_registry.go
  - internal/services/tech_debt_repo_adapter.go
  - internal/cli/services_global.go
  - internal/keys/service.go
  - internal/db/db.go
  - internal/services/context_service.go
  - internal/services/claim_service.go
  - internal/services/entity_history_service.go
  - internal/models/entity_relationship.go
  - internal/repository/entityrel/repository.go
  - internal/cli/commands/next.go
  - internal/cli/commands/delete_dispatch.go
  - internal/cli/commands/update_dispatch.go
  - internal/services/viewer_service.go
  - internal/searchindex/sql.go
  - internal/sharkdata/default_data/research/recipes.yaml
---

# Research report — Question entity and platform registration

## Scope

E39-F01 creates I-01: a durable, top-level Question record identified by the
proposed `Q###` key and recognized by Shark's generic platform paths. It covers
additive persistence, model and key recognition, repository registration,
generic services, claims, history, context, relationships, search, viewer,
CLI, API, and workflow registration required to create, retrieve, list, link,
and retain a Question.

This feature does not define `question_state`, responder sequencing, response
or resolution provenance, `question_blocks`, gate behavior, role disclosure,
or focused Question queries. Those remain with F02, F03, and F04 respectively.
The terms **Question**, **Question record**, **Q###**, **registered type**, and
**generic platform path** have the meanings established in the parent
[research report](../research-report.md) and
[architecture](../architecture.md#scope-and-component-design).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md`, E39 `requirements.md` REQ-F-001, and `architecture.md` define the bounded I-01 record, `Q###`, and exclusions.
- [x] `affected_implementation_or_contract` — Evidence: `internal/models/entity_note.go`, `internal/keys/service.go`, `internal/cli/services_global.go`, `internal/db/db.go`, and `internal/cli/commands/next.go` enumerate the current supported entity types, keys, registration, persistence, and dispatch adapters.
- [x] `related_work` — Evidence: the parent `research-report.md`, `architecture.md`, `E39-interaction-map.md`, and F02–F04 `feature.md` files define I-01 as the upstream contract and allocate serial state, gates, and focused reads to later features.
- [x] `pattern_contract` — Evidence: `internal/models/tech_debt.go`, `internal/services/tech_debt_repo_adapter.go`, `internal/services/entity_registry.go`, and `internal/services/context_service.go` show the model, typed adapter, registry, and generic-context pattern for a standalone entity.
- [x] `dependency_impact` — Evidence: `internal/cli/commands/{delete_dispatch,update_dispatch}.go`, `internal/services/{entity_history_service,viewer_service}.go`, `internal/searchindex/sql.go`, and `internal/cli/services_global.go` contain closed type routing or projections that must recognize Question.
- [x] `cross_boundary_risks` — Evidence: `internal/services/claim_service.go` keys leases by arbitrary type/key while `internal/cli/commands/next.go` builds typed transitioner and placeholder adapters; incomplete registration can make a persisted Question visible to one generic surface but unusable in dispatch, claims, or history.
- [x] `alternatives` — Evidence: `internal/models/context_data.go`, `internal/services/context_service.go`, E39 `scope.md`, and `architecture.md` show that string-based `open_questions`, metadata, notes, or reuse of `bug`/`change` cannot supply a separately addressable lifecycle record without breaking the stated boundaries.

## Capability map

| Capability | Established evidence | Decision | F01 application |
| --- | --- | --- | --- |
| Top-level entity model with shared identity/status access | `models.BaseEntity`; `models.TechDebt`; `models.Entity` | **EXTEND** | Add a Question model and its own typed repository; reuse shared entity accessors rather than model Question as a task, bug, or note. |
| Type validation and polymorphic notes/relationships | `models.EntityType`, `ValidEntityTypes`, `EntityNote.Validate`, `EntityRelationship.Validate` | **EXTEND** | Add `question` to the application allowlist so existing generic note, relationship, tag, document, and history services can resolve it. |
| Key recognition and normalization | `internal/keys/service.go`; CLI `DetectEntityType` tests | **EXTEND** | Add strict, case-insensitive `Q###` parsing, formatting, normalization, and collision tests before generic command paths use the key. |
| Registry and typed repository adapters | `EntityRegistry`; `TechDebtRepositoryAdapter`; CLI global registration | **EXTEND** | Register a Question adapter at startup and provide each generic adapter method, including context reads/writes and conditional status update. |
| Additive persistence and cleanup | `internal/db/db.go`; standalone entity tables; polymorphic cleanup triggers | **EXTEND** | Add a fresh-install table and idempotent forward migration, indexes, and cleanup coverage. Do not backfill or parse existing `open_questions`. |
| Claims and work sessions | `ClaimService`; `entity_claims`; `work_sessions` | **REUSE** | A Question can use the existing type/key lease. F01 must not introduce a second claim mechanism or responder-state side effect. |
| Context, history, documents, and relationships | `ContextService`; `EntityHistoryService`; `entityrel.Repository` | **EXTEND** | Make Question reachable by these generic services only; F02 owns the validated `question_state` schema and F03 owns the new blocking relation. |
| Keyed dispatch and route workflows | `next.go`; installed workflow files | **EXTEND** | Register a Question workflow and all adapter builders necessary for `shark next Q### --json` to preserve the standard response shape. F02 owns its responder routing semantics. |
| Unified command and search/viewer projections | `delete_dispatch.go`; `update_dispatch.go`; `viewer_service.go`; `searchindex/sql.go` | **EXTEND** | Include Question in generic get/update/delete/list/search/viewer and API registration without making response material searchable or adding F04’s focused query contract. |
| F02 serial state/provenance, F03 gate, F04 safe reads | F02–F04 feature briefs and I-01 row | **CONTRADICTS** | Do not pull later consumers into the platform-registration slice. They consume a registered record; they do not redefine it. |
| Notes, `open_questions`, generic metadata, or a renamed existing entity | Parent scope and `ContextService` | **CONTRADICTS** | These alternatives lack a distinct key, lifecycle, generic lookup, and history contract; they also mix F01 with future workflow behavior. |

## Findings

1. **Question is a closed-contract extension, not one model/table change.**
   `ValidEntityTypes`, the key service, global registry setup, command
   dispatch switches, viewer projections, workflow adapters, and tests each
   enumerate supported types. A Question record without complete registration
   will fail on otherwise generic operations such as notes, context, history,
   tags, links, and deletion.

2. **The platform has a proven adapter seam for standalone entities.**
   Tech debt supplies the closest local shape: a typed model embeds the shared
   base entity, a repository adapter implements the generic interface, and
   `GetEntityRegistry` wires it. The new Question should follow that seam so
   cross-cutting services continue to resolve one entity type through one
   registry.

3. **Generic persistence is permissive, but physical cleanup is still
   entity-specific.** The polymorphic association tables no longer constrain
   entity type in DDL, whereas `db.go` contains explicit tables, migrations,
   projections, and cleanup trigger maintenance for concrete entities. F01
   therefore needs fresh-database and upgrade paths plus an inventory of
   concrete SQL branches; the generic association tables alone do not make
   Question deletion or viewer/search behavior complete.

4. **Claims can already lease a Question safely once its key/type resolves.**
   The claim service persists an arbitrary `entity_type` and `entity_key` and
   treats workflow phase separately. F01 should reuse it unchanged. It must
   not infer response completion, request ownership, or resolution from claim
   acquisition, expiry, or release.

5. **The keyed-dispatch boundary needs complete typed adapter support.**
   `shark next <key> --json` parses a key and builds a per-entity transitioner,
   placeholder generator, and narrowed action service. F01 must leave that
   response envelope unchanged while making `Q###` a supported input. F02,
   not F01, decides which responder receives the rendered prompt.

6. **API and viewer work divide into registration now and policy later.**
   Existing handlers and viewer service use explicit repositories and flat
   collections; they are not automatically registry-driven. F01 must make a
   Question record available through normal generic transport/read paths.
   F04 retains open-by-responder, blocking-for, and compact-versus-full
   disclosure behavior.

7. **The main compatibility risk is a partial type inventory.** Prior sprint
   and idea additions required both key parsing and registry adapters before
   polymorphic note operations worked. A structural all-supported-types test
   matrix is the appropriate guard: adding Question must exercise every
   intended generic path and show that existing types keep their behavior.

## Decisions

1. **Create a first-class `question` entity with a strict `Q###` key.** Keep
   the key distinct from existing entity formats and make detection and
   normalization case-insensitive, following the current key-service contract.

2. **Treat F01 as an additive platform-registration slice.** Add the model,
   typed repository, adapter, registry wiring, persistence, base workflow
   registration, generic CLI/API/search/viewer support, and regression tests
   as one coherent I-01 contract.

3. **Use the existing generic services without expanding their semantics.**
   Claims, notes, context, history, documents, links, tags, and work sessions
   should resolve Question through the standard registry. Do not add a
   Question-owned lease, queue, or linked-work transition authority.

4. **Keep future domain behavior out of F01.** F02 owns validated
   `question_state`, serial routing, and provenance. F03 owns
   `question_blocks` and gates. F04 owns focused reads and disclosure. This
   preserves the I-01 producer/consumer boundary in the interaction map.

5. **Require an exhaustive registration inventory and compatibility tests.**
   Test fresh and upgraded databases; `Q###` parsing; registry resolution;
   generic note/context/history/document/tag/link operations; claims; normal
   CRUD/list/search/viewer/API paths; and unchanged behavior for existing
   entity types. Treat a missed closed switch as a feature defect, not a
   follow-up convenience.

## Sources

- E39 contracts: `feature.md`, parent `research-report.md`, `requirements.md`,
  `scope.md`, `architecture.md`, and `E39-interaction-map.md`.
- E39 siblings and consumers: F02, F03, and F04 `feature.md` files.
- Entity/type registration: `internal/models/entity_note.go`,
  `internal/models/tech_debt.go`, `internal/services/entity_registry.go`,
  `internal/services/tech_debt_repo_adapter.go`,
  `internal/cli/services_global.go`, and `internal/keys/service.go`.
- Generic operations and persistence: `internal/services/{context_service,
  claim_service,entity_history_service}.go`,
  `internal/models/entity_relationship.go`,
  `internal/repository/entityrel/repository.go`, and `internal/db/db.go`.
- Transport and projections: `internal/cli/commands/{next,delete_dispatch,
  update_dispatch}.go`, `internal/services/viewer_service.go`,
  `internal/searchindex/sql.go`, and existing entity-type regression tests.
- Recipe contract: `internal/sharkdata/default_data/research/recipes.yaml`.

RECOMMENDED OUTCOME: pass
