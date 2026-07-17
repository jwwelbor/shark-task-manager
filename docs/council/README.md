# Use the council memory layout

## Goal

Use `docs/council/` to retain the bounded decisions and handoffs that a
refreshed Shark Attack worker needs. This directory stores project memory; it
does not replace Shark workflow, claim, or status authority.

## Layout

```text
docs/council/
├── decisions/       Durable direction and rationale
├── handoffs/        Scoped next-step records
├── escalations/     Unresolved material questions and resolutions
└── inbox/<member-id>/  Short-lived actionable messages
```

The committed `.gitkeep` files preserve the directory layout. Create member
inboxes from the stable member IDs in the Shark Attack roster.

## Create and acknowledge records

1. Write a decision, handoff, escalation, or resolution before you
   acknowledge its inbox message.
2. Include the record ID, type, status, roles, root key, optional child key,
   evidence links, timestamps, and next action.
3. Keep messages in `inbox/<member-id>/` short-lived. Acknowledge or remove a
   message only after its durable result exists.
4. On a refreshed worker start, read decisions, handoffs, unresolved
   escalations, and that member's inbox. Use the bounded pointers and metadata
   to rebuild context.

## Retain private material safely

Keep shared decisions, handoffs, and resolution pointers available to the
project. If a council needs private local material, add only that material to
the project `.gitignore`. Do not commit credentials, access tokens, rendered
prompts, unrestricted worker output, absolute paths, or paths outside this
directory.

## Escalate unresolved questions

Use `docs/product/escalation_triggers.md` when the project provides it. If the
file is absent and a material question remains unresolved, record an
escalation with route `council-review` and recommendation `pause/review`. Do
not choose a fixed human destination.
