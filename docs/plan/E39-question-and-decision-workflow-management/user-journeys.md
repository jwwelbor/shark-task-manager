# User Journeys

**Epic**: [Question and Decision Workflow Management](./epic.md)

## J-01: Raise and Route a Blocking Question

**Actor**: Requester

1. The requester creates a Question, marks it blocking, and links only the
   affected work.
2. The requester names the owner and requested responders in `question_state`.
3. The Question appears in the focused outstanding-question views.
4. The first pending responder becomes the only responder eligible for dispatch.

**Unhappy path**: If the Question is not linked or is not marked blocking, it
does not prevent any work from advancing.

## J-02: Collect Serial Responses and Resolve a Feature Change

**Actors**: Requested responder, resolution owner

1. The first responder claims the Question and records a bounded answer with an
   evidence pointer.
2. The claim is released; the next pending responder becomes eligible.
3. After all required responses, the resolution owner classifies an answer that
   changes feature behavior.
4. The owner updates the authoritative feature specification, records its
   pointer, then resolves the Question.

**Unhappy path**: A response that lacks required evidence or exceeds the
context bound is rejected and does not mark the responder complete.

## J-03: Gate Only Affected Work

**Actor**: Linked-work owner

1. The owner attempts to dispatch or advance linked work.
2. The system checks only open, blocking Questions linked to that entity.
3. If one exists, the owner receives a compact summary and the entity pauses.
4. Unlinked work remains available through normal workflow behavior.

## J-04: Turn an Answer into Follow-Up Work

**Actor**: Resolution owner

1. The owner determines that an answer reveals implementation work rather than
   a durable policy or requirement change.
2. The owner creates or selects the appropriate Shark work item and links it.
3. The owner records the resolution kind and link, then resolves the Question.

*See also*: [Requirements](./requirements.md) and [Scope](./scope.md).

