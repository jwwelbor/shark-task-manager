# Resume council work after a worker refresh

## Goal

Restore the bounded context a council member needs after a fresh worker starts,
without depending on prior chat or storing sensitive transcripts.

## Prerequisites

- Know the project root and the member ID.
- Use the council layout under `docs/council/`.

## Steps

1. Read relevant durable records in this order:

   - `docs/council/decisions/`
   - `docs/council/handoffs/`
   - `docs/council/escalations/`, including unresolved items
   - `docs/council/inbox/<member-id>/`

2. Keep only records that match the active root key and, when present, child
   key. Report malformed, stale, secret-bearing, or out-of-root references as
   actionable errors; do not load them into worker context.

3. Build the resume context from bounded paths and metadata: IDs, types,
   roles, root and child scope, statuses, evidence, timestamps, and next
   actions. Do not load rendered prompts, credentials, access tokens, or
   unrestricted worker output.

4. Act on each actionable inbox message. Before you acknowledge it, preserve
   the resulting decision, handoff, unresolved escalation, or resolution in
   the applicable durable directory.

5. Acknowledge or remove the message after its durable result exists.
   Repeating the acknowledgement is a no-op and does not remove the durable
   record.

## Result

A refreshed worker has the same durable decision, handoff, unresolved
escalation, and inbox pointers needed to continue safely. Use the existing
workflow and claim procedures for any work selection or lease operation.
