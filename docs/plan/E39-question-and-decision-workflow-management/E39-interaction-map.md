---
type: interaction-map
epic: E39
last_updated: 2026-07-30
---

# E39 Cross-Feature Interaction Map

Four intended features each have a real trigger, production path, observable UAT, current prerequisites, and downstream output in [architecture](architecture.md#delivery-boundaries-and-traceability). No criterion depends on later unbuilt behavior.

| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |
|---|---|---|---|---|---|
| I-01 | E39-F01 Entity and platform registration | E39-F02 Serial workflow/provenance; E39-F03 Scoped gate; E39-F04 Focused safe reads | [E39-F01 I-01 v1 contract](E39-F01-question-entity-and-platform-registration/spec.md#produces-i-01--entity-and-platform-registration-v1) | `Q###`, registered type, persisted base record, `models.ContextData`, generic adapters | Shared data model and registry |
| I-02 | E39-F02 Serial workflow and response provenance | E39-F03 Scoped gate; E39-F04 Focused safe reads | [Architecture workflow](architecture.md#workflow-and-direct-gate) | Validated state, first pending responder, bounded evidence, resolution kind/pointer | Service and keyed-dispatch contract |
| I-03 | E39-F03 Scoped relationship gate | E39-F04 Focused safe reads | [Architecture direct gate](architecture.md#workflow-and-direct-gate) | Qualifying `question_blocks` predicate and compact Q summary | Service and CLI/API handoff |

All rows use live gate mode; no `contract-only` row is declared. Decomposition must create matching boundaries or update this map before advancing, and feature/task specs, review, QA, and UAT reuse these stable IDs.
