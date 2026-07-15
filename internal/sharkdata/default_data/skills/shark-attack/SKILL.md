---
name: shark-attack
description: Chair-led, evidence-based team protocol for Shark work without adding a runtime, workflow engine, or claim store.
---

# Shark Attack

## Purpose

Shark Attack is a durable collaboration protocol for work that benefits from a
chair-led council. It coordinates judgment and evidence; Shark workflow metadata,
the owning dispatch loop, and the existing claim service remain authoritative.
It does not create an AI runtime, provider configuration, credential store,
workflow engine, or second lease store.

## Setup and roster

Use `context/roster-schema.yaml` as the canonical roster template. Its `team`
value is `shark-attack`; its chair is a roster member who facilitates a
decision, not a new workflow authority. Stable built-in IDs resolve to existing
personas: tech-director, product-manager, architect, business-analyst,
scrum-master, developer, and qa. A project specialist may omit `persona`.

`model_tier` is an optional preference only. It cannot select work, override
workflow metadata, modify status, or affect a claim. Role responsibilities must
be scoped to analysis, implementation, review, evidence, or facilitation.

Keep council memory below `docs/council/`:

- `decisions/` for durable direction and rationale.
- `handoffs/` for bounded scope, evidence, owner, and next action.
- `escalations/` for unresolved material questions and their resolution.
- `inbox/<member-id>/` for short-lived messages.

Private council material may be ignored locally. Do not put credentials,
access tokens, rendered prompts, or unrestricted worker output in council
artifacts.

## Communication and ownership

Every inbox message identifies sender role, recipient role, root key, optional
child key, subject, requested action or question, urgency, evidence links, and
creation time. After the recipient acts, acknowledge or remove the message and
preserve the resulting decision, handoff, unresolved question, or resolution in
the durable directories. Store bounded paths and metadata, never transcripts.

Workers may read the scoped state, work on authorized children, emit evidence,
and return a semantic outcome. They never change the dispatched root lease or
root workflow state. Role-aware self-pull follows resolved workflow role and
existing priority/dependency ordering; roster membership and model tier grant no
claim or status authority.

## Escalation and resume

Escalate missing evidence, material direction changes, specialist disagreement,
or unresolved process/quality blockers. If the project has no escalation policy,
record an unresolved escalation, route it to `council-review`, and recommend
pause/review. Never invent a fixed human destination.

A refreshed worker reads durable decisions, handoffs, unresolved escalations,
and its inbox before acting. Preserve unresolved context with bounded pointers
so the next worker can continue without relying on prior chat.

## Distribution

This skill ships in the embedded Shark-data bundle. Project customizations use
the replace-only overrides skill subtree for shark-attack; an
override replaces the matching shark-attack file and does not shadow unrelated
embedded skills. Use the existing Shark Rider, sprint, notes, context, and
claim procedures rather than copying their implementation here.
