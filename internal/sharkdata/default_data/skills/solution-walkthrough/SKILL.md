---
name: solution-walkthrough
description: Walk a Shark entity or an authoritative project document through its solution decisions one at a time. Use for architecture or design walkthroughs, document decision review, resolving open choices, or ratifying an important documented direction; ground recommendations in Shark state when available, linked documents, docs/product, docs/architecture, and code, then persist each approved outcome in the appropriate durable record.
---

# Solution Walkthrough

Use this skill for a collaborative, recommendation-first decision walk. It is
not a lifecycle transition or a one-shot review report. The Rider supplies the
resolved entity, Shark state, and linked documents; treat those and the
project's documentation as the evidence base.

## Read and queue

1. Read the selected entity or authoritative document in full. For an entity,
   also read its Shark file, related documents, and parent/child context when
   material. For a document, identify explicit entity keys, referenced records,
   and its owning documentation area; do not invent a Shark association.
   Review outstanding Question entities: `open`, `answering`, and
   `ready_for_resolution`. Page each status with `shark question list` using
   `--limit=100` and increasing `--offset` until a page is short. For an
   non-Question entity target, also read
   `shark question blocking-for <entity-key>`. Read potentially relevant
   Questions and their linked documents. Prioritize
   Questions that directly block the target or explicitly name the selected
   entity or document. Report reviewed-but-out-of-scope Questions separately;
   do not infer a relationship from topical similarity.
   Then read the relevant project documents:
   `docs/product/progress.md`, `docs/product/cross-epic-integration-map.md`,
   `docs/architecture/`, and existing ADRs or decision records.
2. Build a queue from material outstanding Question entities, explicit open
   questions in the selected records, and consequential directions already
   stated in the documents. Do not reopen a settled decision unless the
   operator placed it in scope.
3. Show the queue and recommend dependency order. Confirm the order before the
   first decision. Work one decision at a time.

## Per-decision loop

1. **Discover.** Inspect the minimum relevant code, configuration, tests, and
   documents before forming a view. Cite repository facts with file paths and
   line numbers; label assumptions and planned work separately.
2. **Frame from purpose.** State the user or system need the decision answers,
   then trace the affected flow and constraints. If the existing options answer
   the wrong question, reframe instead of forcing a choice.
3. **Recommend.** Lead with the simplest recommendation that fits the evidence,
   followed by alternatives, trade-offs, risks, and downstream effects. Ask the
   operator to approve, amend, or redirect it.
4. **Record only after a response.** Approval records a resolution; an amendment
   updates the source section and leaves the item open; a reopened direction
   returns to the queue. Never invent approval, silently lock a decision, or
   turn a discussion into a status change. When the operator approves an answer
   to an in-scope Question, first write or update the durable decision record
   that supplies its evidence pointer. Then retrieve `shark next <question-key>
   --json`. Continue only when its `current_responder` matches the authenticated
   walkthrough operator; then claim it with `shark claim
   <question-key> --by=<current-responder> --json`, capture the returned
   `session_id`, and
   record the approved answer with
   `shark question respond <question-key> --session=<session-id>
   --responder=<current-responder> --summary="<approved answer>"
   --evidence-pointer=<durable-record-path>`. Release the claim afterward. If
   the operator cannot verify that it is the responder, hand off the durable
   decision and Question; never infer or impersonate a responder. Do not resolve, withdraw, supersede, or
   otherwise close a Question as part of the walk.

## Ratifying documented decisions

A ratification reviews an important direction that the source document already
states. Re-check its purpose, evidence, alternatives, and downstream effects;
then ask the operator to confirm it. On confirmation, annotate the source
record as **Reviewed and confirmed** with the date, rationale sharpened by the
walk, and a pointer to any ADR or decision record. Do not duplicate the
decision merely to say it was ratified. If the review exposes a conflict or an
unacceptable consequence, reopen it instead.

## Durable-record routing

Put each resolved decision in the narrowest durable source of truth, update
every affected source document, and preserve existing local conventions.

| Decision scope | Durable record |
|---|---|
| Product direction, cross-epic sequencing, or an accepted product deferral | Append the decision to `docs/product/progress.md`'s Decision Log; update the cross-epic map when its contract or ownership changes. |
| Shared or system architecture, a reusable technical policy, or a decision affecting multiple entities | Create or update `docs/architecture/adr/ADR-<next-number>-<slug>.md` using the architecture ADR template. |
| Epic architecture | Update the epic `architecture.md` (including its local ADR section) and its interaction/cross-epic maps when affected. |
| Feature, task, bug, or change-card implementation decision | Update the authoritative spec/design first. When the reasoning needs a standalone record, use the existing entity-local `decisions.md`; otherwise create the smallest sibling decision record beside the entity's Shark file. |
| Idea or tech-debt decision | Record it in the entity's authoritative plan file or the narrowest applicable product/architecture record. |

Each record must state the question from purpose, decision, rationale and
alternatives, consequences/risks, evidence anchors, affected documents, and
the date. Use [the decision-record template](context/decision-record-template.md)
for an entity-local record. Prefer updating an existing record over duplicating
one. Validate changed Mermaid before saving it.

## Linking and handoff

After a durable document exists, link it to the selected or explicitly
identified epic, feature, task, bug, or change-card with Shark related
documents. For entity types without a related-document command, add a
`reference` note naming the canonical path. A document-only walk has no Shark
link unless that relationship is evidenced. Do not use notes as a replacement
where a related-document link is supported.

Update the next open decision and settled-decision summary in the authoritative
record. At a natural stop, create or update a sibling `continue-prompt.md` only
when unresolved decisions remain. It must name the entity and scope, confirmed
order, completed decisions with record pointers, the next decision and its
evidence, and settled decisions that should not be relitigated.

## Final check

Before reporting a result, confirm that every factual claim has evidence, every
approved decision has a durable location and required Shark reference, source
documents cross-reference the outcome where necessary, and no lifecycle state
or backlog item changed implicitly. Route newly discovered work through normal
triage; do not create it automatically.
