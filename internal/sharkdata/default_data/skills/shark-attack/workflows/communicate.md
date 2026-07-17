# Communicate through the council inbox

## Goal

Send a bounded, actionable council message and preserve its result after the
recipient acknowledges the message.

## Prerequisites

- Use a valid root Shark key and, when applicable, a valid child key.
- Use a recipient role that exists in the council roster.
- Store messages below `docs/council/inbox/<member-id>/`.

## Create a message

1. Create one structured message with these fields:

   - `message_id`
   - `sender_role` and `recipient_role`
   - `root_key` and optional `child_key`
   - `subject`
   - `requested_action` or `question`
   - `urgency`
   - `evidence` paths or artifact links
   - `created_at` in RFC 3339 format

2. Keep the body to the context needed to act. Do not include rendered
   prompts, credentials, access tokens, unrestricted worker output, absolute
   paths, or paths that escape `docs/council/`.

3. After acting, write the result as a bounded artifact in
   `docs/council/decisions/`, `handoffs/`, or `escalations/`. Include the
   artifact ID, type, status, roles, root and child scope, evidence, timestamps,
   and `next_action`.

4. Acknowledge or remove the inbox message only after the durable artifact is
   present. Repeating an acknowledgement for an already acknowledged message
   is a no-op.

5. Reuse an artifact ID only for byte-equivalent content. Report a conflicting
   ID and preserve the first artifact; never overwrite it.

## Result

The inbox stays short-lived while decisions, handoffs, unresolved questions,
and resolutions remain available through bounded paths and structured metadata.

## Ownership boundary

Writing a communication artifact does not claim work, release a root lease, or
advance a root workflow state. Use the existing Shark Rider, sprint, notes,
context, and claim procedures for those operations.
