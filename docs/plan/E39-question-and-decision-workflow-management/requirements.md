# Requirements

**Epic**: [Question and Decision Workflow Management](./epic.md)

This catalog specifies the business requirements for E39. It derives its
context and boundaries from the epic PRD and must be read with
[Scope](./scope.md), [Personas](./personas.md), and
[User Journeys](./user-journeys.md).

## Functional Requirements

### Question lifecycle

**REQ-F-001 — First-class Question record**

- The system must create, retrieve, list, search, link, and retain history for
  a top-level Question entity.
- A Question must expose its identifier, one-line question, current lifecycle
  state, blocking state, owner, requested responders, linked entities, and
  resolution state without requiring a consumer to parse unstructured notes.

**REQ-F-002 — Structured responder state**

- The system must retain responder routing and response summaries in a
  structured `question_state` context value.
- Each responder must have an identity/role and a state that distinguishes at
  least pending and completed; the exact schema and validation limits are a
  design decision.

**REQ-F-003 — Serial routed responses**

- Question dispatch must select the first pending responder only.
- A successful response must record a bounded result and evidence pointer,
  mark that responder complete, and make the next pending responder eligible
  only after the current claim is released.

**REQ-F-004 — One active claimant**

- Existing claim semantics must prevent concurrent response handling for the
  same Question.
- A failed, expired, or released claim must not silently mark a responder
  complete or resolve the Question.

### Blocking and relationships

**REQ-F-005 — Explicit blocking relationship**

- A Question may link to affected eligible Shark entities.
- Only an open Question explicitly marked blocking and linked to the entity
  under evaluation may stop that entity's dispatch or advancement.

**REQ-F-006 — Focused outstanding-question views**

- Consumers must be able to find open Questions by requested responder and by
  entity blocked.
- A blocked-work prompt must receive a compact Question summary and load the
  full record only when the actor is its assigned responder or resolution owner.

### Resolution and provenance

**REQ-F-007 — Explicit resolution classification**

- Before a Question resolves, the resolver must supply a `resolution_kind`.
- A resolution that changes feature behavior, requirements, acceptance,
  product direction, architecture, or implementation work must include a
  pointer to the required authoritative record or linked Shark item.

**REQ-F-008 — Authoritative destinations**

| Resolution kind | Required durable destination |
|---|---|
| Local clarification without contract change | Question context and concise linked-entity note |
| Feature behavior, requirements, or acceptance change | Authoritative feature specification/document |
| Product direction, priority, sequencing, or owner decision | `docs/product/progress.md` decision log |
| Shared technical policy or cross-feature architecture | ADR and affected architecture/spec references |
| Newly discovered implementation work | Linked Shark task, bug, change card, tech-debt item, feature, or epic |
| No lasting consequence | Bounded context answer and resolved Question history |

**REQ-F-009 — No automatic mutation of linked work**

- Answering, responding to, or resolving a Question must not claim, advance,
  resolve, or otherwise mutate a linked entity.
- Follow-up work must be explicitly created or linked by an authorized resolver.

## Non-Functional Requirements

**REQ-NF-001 — Bounded and safe context**

- `question_state` must reject prompts, transcripts, credentials, and content
  beyond the documented response bound.
- Long-form material must be stored as a typed note or an authoritative
  document and referenced from context.

**REQ-NF-002 — Durable auditability**

- Create, response, resolution, withdrawal, and supersession actions must be
  visible in the Question's history or linked durable record.

**REQ-NF-003 — Compatibility**

- Existing entity workflows, claims, keyed dispatch, note/context behavior,
  links, search, and unrelated advancement must retain their established
  behavior.

**REQ-NF-004 — Decomposition handoff**

- The design phase must create `E39-interaction-map.md` before feature
  implementation if decomposition produces three or more features or a
  producer/consumer contract.
- Its I-## rows must use only shapes that resolve to `architecture.md` sections;
  feature and task specs must then reuse the stable identifiers verbatim.

## Requirement Traceability

| Requirement group | Journeys | UAT scenarios |
|---|---|---|
| REQ-F-001 to REQ-F-004 | J-01, J-02 | UAT-01, UAT-02 |
| REQ-F-005 to REQ-F-006 | J-03 | UAT-01, UAT-03 |
| REQ-F-007 to REQ-F-009 | J-02, J-04 | UAT-04, UAT-06 |
| REQ-NF-001 to REQ-NF-004 | J-01 to J-04 | UAT-02, UAT-05 |

