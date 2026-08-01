---
epic_key: E39
title: Question and Decision Workflow Management
description: A first-class, serial Question workflow that preserves accountable answers and blocks only the work explicitly waiting on them.
open_questions:
  - "Which roles may create, claim, respond to, resolve, withdraw, or supersede a Question, and how are those roles authenticated or authorized?"
  - "What key prefix, workflow state/outcome names, exact context schema limits, relationship types, and gate timing best fit existing Shark conventions?"
  - "Which CLI/API/viewer/search/reporting surfaces are required for the first release, and which are deferred?"
  - "What telemetry source, baseline, target, release cohort, and accountable owner should govern post-release adoption and time-to-resolution metrics?"
---

# Question and Decision Workflow Management

**Epic Key**: E39

This document is E39's single source of business context. Feature specifications
must reference it for the problem, goals, boundaries, constraints, stakeholders,
and epic-level UAT rather than restating them. Detailed, traceable requirements
are in [Requirements](./requirements.md); explicit exclusions are in
[Scope](./scope.md); role context, journeys, and measurement are in
[Personas](./personas.md), [User Journeys](./user-journeys.md), and
[Success Metrics](./success-metrics.md).

## 1. Problem Statement and Business Justification

People and AI workers can discover a question that needs a specific responder,
but Shark currently has no accountable, queryable record that keeps the question
visible through answer and resolution. A note or report can preserve prose, but
cannot reliably state who must answer, whether the issue is still open, which
work is blocked, or where a consequential decision was recorded. The result is
lost decisions, unnecessary context recovery, and either premature advancement
or overly broad blocking.

E39 provides a first-class Question entity with a serial responder workflow and
an explicit owner resolution. It reuses Shark's existing entity, workflow,
claim, note, context, relationship, history, search, and keyed-dispatch
capabilities. The business value is high: the capability makes a bounded answer
queue visible to humans and agents, preserves decision provenance, and limits a
blocking question to the linked work that actually depends on it.

## 2. Goals and Success Criteria

### Goals

1. Give a requester an accountable, durable Question record instead of relying
   on transient worker output or unstructured notes.
2. Enable multiple requested perspectives without concurrent writes by routing
   exactly one pending responder at a time under the existing claim lifecycle.
3. Prevent a linked entity from advancing only when it has an open, explicitly
   blocking Question; do not block unrelated work.
4. Require a resolver to classify a consequential answer and point to its
   authoritative destination before resolution.

### Release success criteria

The release is successful only when all of the following are demonstrated by
automated coverage and the UAT scenarios in section 6:

- A Question can be created, opened, listed, retrieved, linked, claimed,
  serially answered, and resolved with its history preserved.
- At no time can two responders hold an active claim for the same Question, and
  the next dispatch selects only the first pending responder.
- An unresolved blocking Question stops advancement of each explicitly linked
  entity; an unresolved Question has no effect on an entity without that link.
- Every resolved Question has a `resolution_kind` and a valid pointer to the
  required durable record when its answer changes requirements, product
  direction, architecture, or implementation work.
- Question context rejects prompt text, transcripts, credentials, and response
  content beyond the documented bound; long material is retained by typed note
  or authoritative document pointer instead.

Post-release adoption and elapsed-time targets are intentionally not set: no
baseline, telemetry source, release cohort, or accountable metric owner has
been supplied. Those are deferred open questions, not assumed product facts.

## 3. Scope

### In scope

- A generic top-level Question entity with a key, workflow, claims, history,
  notes, structured context, relationships, search, CLI/query surfaces, and
  keyed dispatch.
- Serial responder routing stored in structured `question_state`, including
  responder identity/state, bounded response summaries, evidence pointers, and
  owner resolution readiness.
- Focused read views for open Questions by recipient and by entity blocked.
- A precise advancement gate that consults only open, blocking Questions linked
  to the entity being dispatched or advanced.
- Resolution classification and durable-record validation for consequential
  answers, with links to created follow-up Shark work where applicable.

### Out of scope

The exclusions and their rationale are authoritative in [Scope](./scope.md).
They include repairing E38-F09's current adapter/continuation defects,
unbounded conversational storage, worker-owned transitions on linked work,
global blocking, and implementation work created before E39 is refined and
decomposed.

## 4. Constraints and Assumptions

- E39 must reuse existing workflow, keyed dispatch, claim, relationship,
  context, note, history, and search patterns; it must not introduce a parallel
  council queue, runtime, workflow engine, or claim store.
- One Question has at most one active claimant. Responses are intentionally
  serial in the first release; parallel collection is not implied.
- `question_state` is structured, bounded coordination state. Prompts,
  transcripts, credentials, and unbounded worker output are forbidden there.
- A Question is a coordination record, not automatic implementation work. A
  resolver creates or links a task, bug, change card, tech-debt item, feature,
  or epic only when the answer reveals such work.
- The exact key format, workflow labels/outcomes, role-routing policy, context
  schema limits, authorization model, API/viewer form, search/reporting shape,
  relationship semantics, and gate timing remain design decisions. They must
  not be treated as settled by this PRD.
- E39 is likely to decompose into three or more features because it spans entity
  registration, workflow/claims, advancement gates, and read/query surfaces.
  The design phase must create `E39-interaction-map.md` from the interaction-map
  template, assigning I-## identifiers only after each cross-feature shape
  resolves to an `architecture.md` section. No I-## identifiers are assigned in
  this PRD.
- Document-set decision: personas, journeys, and success metrics are included
  because human owners, assigned responders, and agent workers have distinct
  workflows and the release claims verifiable outcomes. No optional PRD detail
  document is excluded.

## 5. Stakeholder Impact

| Stakeholder | Impact | Required outcome |
|---|---|---|
| Requester (human or agent) | Can create a durable question and see its owner, state, links, and final record. | The question does not disappear into a report or chat transcript. |
| Requested responder | Receives one scoped Question at a time and can provide a bounded answer with evidence. | The responder knows what requires action without scanning unrelated history. |
| Resolution owner | Classifies the answer and records consequential decisions in the correct authoritative location. | Closure has accountable provenance rather than a free-form “done” note. |
| Linked-work owner | Sees only Questions that explicitly block that work. | Relevant work pauses safely; unrelated work continues. |
| Shark maintainer and API/CLI consumer | Gains a new supported entity and query surface that must follow established platform conventions. | Existing entity and workflow behavior remains intact. |

## 6. High-Level Acceptance Criteria (UAT Scenarios)

| ID | Scenario | Demonstrable outcome |
|---|---|---|
| UAT-01 | A requester opens a blocking Question linked to one feature and one task. | The Question is discoverable by recipient and by each linked entity, with its blocking flag and links visible. |
| UAT-02 | Three responders are requested for one Question. | Only the first pending responder receives dispatch while claimed; after a successful bounded response and release, exactly the next pending responder becomes eligible. |
| UAT-03 | A linked entity is ready to advance while its open blocking Question remains unresolved. | That entity is stopped with a compact Question summary; a different, unlinked entity remains eligible to advance. |
| UAT-04 | A responder supplies an answer that changes feature behavior. | The resolver selects the applicable resolution kind, updates the authoritative feature specification, records its pointer, and only then resolves the Question. |
| UAT-05 | A responder attempts to save a transcript, prompt, credential, or over-limit content in Question context. | Validation rejects the content and directs durable long-form material to a typed note or authoritative record pointer. |
| UAT-06 | An answer identifies implementation work rather than a durable decision. | The resolver links a newly created or existing Shark work item and preserves the Question resolution history without automatically mutating linked work. |

*Last Updated*: 2026-07-30
