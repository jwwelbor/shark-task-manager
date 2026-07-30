---
epic_key: E39
title: Question and Decision Workflow Management
description: Introduce first-class Question entities for routed, serially answered, blocking questions with authoritative resolution records; discovered while reviewing E38-F09 live-question gaps.
---

# Question and Decision Workflow Management

**Epic Key**: E39

## Goal

Give teams a dependable way to raise, route, answer, and resolve questions that
otherwise disappear in worker output or at the end of Markdown files. A question
must stay visible until an accountable responder resolves it, and a blocking
question must prevent only the work it explicitly blocks from moving forward.

This epic creates a first-class **Question** entity. It reuses Shark's existing
workflow, keyed dispatch, claims, notes, context, relationship, history, and
search patterns instead of creating another council-only queue or runtime engine.

## Problem

Agents and people frequently discover questions that need another persona or an
owner to answer. Today, they can be written as notes or appended to a report,
but neither form reliably provides all of the following:

- an accountable recipient;
- a compact view of outstanding questions;
- a distinction between open and resolved questions;
- a way to request several perspectives without losing progress;
- a gate that exposes a blocking question before linked work advances; and
- a durable record of where the final answer belongs.

E38-F09 made this gap concrete. Its live-worker protocol can receive a bounded
question, but the UAT review showed that a question needs a parent-owned
lifecycle and a durable route to an answer. E39 generalizes that product need;
it is not a reimplementation of F09's provider-adapter repair.

## Approved direction from discovery

The following direction was agreed during discovery and is the starting point
for refinement. It is not an implementation plan or acceptance approval.

### Model questions as entities

A Question is a top-level Shark entity, analogous to a bug or change card. It
has its own key, workflow, claim lifecycle, history, notes, context, and links
to one or more affected epics, features, tasks, bugs, change cards, or other
Questions where the relationship is valid.

The entity is the coordination record. It is not automatically a work item.
When an answer reveals implementation work, the resolver creates or links the
appropriate task, bug, change card, tech-debt item, feature, or epic.

### Reuse the PR-review interaction shape

Use a Question in the same way a team uses a pull request:

| Question concept | Comparable review concept |
|---|---|
| Question entity | Pull request |
| Current claimant | Maintainer performing the current action |
| Requested responders | Requested reviewers |
| Bounded responses | Reviews or comments |
| Resolution | Merge or close decision |
| Blocking link | Required approval before merge |

The first version deliberately uses **serial** responses. A claimant has the
Question checked out while responding, so no other responder can update its
state concurrently. That trades parallel response collection for a simpler,
safer implementation that reuses existing claim behavior.

### Keep responder state in entity context

Use structured `question_state` context for responder routing and bounded
results. Do not overload free-form notes or unrelated context fields such as
`implementation_decisions`.

Illustrative shape:

```json
{
  "question_state": {
    "blocking": true,
    "linked_entities": ["E38-F09", "T-E38-F09-003"],
    "owner_role": "architect",
    "responders": [
      {"role": "architect", "state": "pending"},
      {"role": "developer", "state": "pending"},
      {"role": "qa", "state": "pending"}
    ],
    "responses": []
  }
}
```

Store only concise response summaries and evidence pointers in context. Store a
long response or a material record in a typed note or authoritative document,
then retain its pointer. Never store prompts, transcripts, credentials, or
unbounded worker output in Question context.

### Route one responder at a time

`shark question next <key>` should identify the first pending responder in
`question_state` and render that responder's scoped prompt. A successful
response records a bounded result, marks that responder complete, releases the
claim, and either routes the next responder or makes the Question ready for an
owner's resolution.

The intended lifecycle is:

```text
draft → open → awaiting input → ready for resolution → resolved
                         ↘ withdrawn or superseded
```

The exact workflow names and outcome map remain design work. The invariants do
not: one active claimant at a time, serial responder routing, explicit owner
resolution, and no silent disappearance of an unresolved blocking question.

## Expose outstanding questions without flooding context

Agents must not scan every note, report, or historical question before they
work. The open Question view is the source for outstanding work:

```text
shark question list --status open --recipient <role>
shark question list --blocks <entity-key>
shark question get <question-key>
```

These command names illustrate the required read model; the final CLI design is
not yet decided. Before a linked entity dispatches or advances, the parent
checks only its open blocking Questions. A prompt receives a compact summary
(identifier, recipient, blocking state, one-line question, and record pointer)
and reads the full record only when it is the assigned responder.

Question notes remain useful as backlinks and search anchors. They are not the
source of truth for whether a Question is open or resolved.

## Resolve questions into authoritative records

Answering a Question is not sufficient. Before resolution, the workflow must
classify the answer and verify its durable destination:

| Resolution kind | Required destination |
|---|---|
| Local clarification with no lasting contract change | Question context and a concise linked entity note |
| Feature behavior, requirements, or acceptance change | The authoritative feature specification or feature document |
| Product direction, priority, sequencing, or owner decision | `docs/product/progress.md` decision log |
| Shared technical policy or cross-feature architecture | An ADR and affected architecture/spec references |
| Newly discovered implementation work | A linked Shark task, bug, change card, tech-debt item, feature, or epic |
| No lasting consequence | A bounded context answer and resolved Question history |

The resolver must provide a `resolution_kind` and record pointer before the
Question can transition to `resolved`. If an answer changes what the project
builds, tests, routes, or accepts, the resolver updates the authoritative
record; a note alone is not enough.

## Scope boundaries

E39 includes the generic Question entity and the platform surfaces necessary to
use it safely: storage, key parsing, model/service/repository registration,
workflow configuration, claims, context, relationships, CLI/query surfaces,
prompt dispatch, and advancement gates.

E39 does not authorize these changes by itself:

- repairing the current F09 native-adapter provenance or live-continuation
  defects;
- turning a Question into an unrestricted chat or transcript store;
- allowing worker-owned workflow transitions or claims on linked work;
- blocking unrelated entities because an open Question exists elsewhere; or
- creating implementation tasks before the epic is refined and decomposed.

## Business value

**Rating**: High

The entity gives people and agents a reliable answer queue with ownership,
history, and precise advancement gates. It prevents decision loss, reduces
context flooding during resume, and makes the difference between a temporary
clarification and a durable product or architecture decision explicit.

## Open design work

- Define the Question key format and all entity registration touchpoints.
- Decide the exact configurable workflow states, outcomes, and role-routing
  mechanism for serial responders.
- Define context validation, bounded response size, and evidence-pointer rules.
- Define relationship semantics and the precise advancement-gate behavior.
- Define CLI, API/viewer, search, and reporting requirements for open Questions.
- Specify authorization for creating, claiming, responding to, and resolving a
  Question, including owner and persona behavior.
- Decompose the epic without re-scoping the blocked E38-F09 feature.

*Last updated*: 2026-07-30
