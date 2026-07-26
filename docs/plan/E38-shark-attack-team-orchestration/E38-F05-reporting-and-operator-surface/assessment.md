# Superseded assessment: lightweight handoff and operator guidance

> Superseded on 2026-07-26. This assessment considered only the original F05
> feature file and omitted the approved Shark Attack v2 triage decision. The
> v2 implementation plan assigns Tranche B deterministic council tooling to
> F05. Use the implementation plan and the F05 future-scope note as the
> current scope.

## Goal

Decide whether E38-F05 needs separate feature delivery.

## Evidence

The feature asks for concise operator reporting of pulled, claimed,
dispatched, completed, blocked, and escalated work. Completed E38-F04 already
defines the durable council handoff, decision, and escalation artifacts. Its
embedded `message-schema.md` supplies the bounded scope, evidence, role,
status, and next-action fields. Completed E38-F07 already defines the Rider
parent loop and its operator-facing execution, escalation, stop, and resume
guidance in `workflows/execute.md`.

The remaining F05 wording does not identify an independent user outcome,
runtime surface, or contract that those features do not provide. Adding a
separate status or reporting subsystem would violate its out-of-scope boundary.

## Superseded scope decision

This conclusion is invalid because it omitted the approved v2 re-scope. F05
owns the deterministic council-artifact contract: typed artifacts, validated
generation, immutable revisions, entity-or-collection scope, evidence
confinement, effective-roster role checks, and thin `shark admin council`
commands. Those capabilities are distinct from the prose-only F04/F07
protocol.

## Superseded complexity triage

**Score:** 5/27
**Tier:** SIMPLE

### Technical complexity

1. File impact: 1/3 — one existing Rider or Shark Attack guide would change.
2. Pattern novelty: 0/3 — reuse the existing handoff and escalation pattern.
3. Data model: 0/3 — no persistence change.
4. API surface: 0/3 — no command or service change.
5. Cross-feature dependencies: 1/3 — reuse F04 and F07 artifacts.
6. UI complexity: 0/3 — Markdown guidance only.

### Execution complexity

7. Task estimation: 1/3 — one documentation task.
8. Regression risk: 1/3 — documentation could misstate Rider ownership.
9. Execution effort: 1/3 — a bounded review and contract check are required.

**Technical total:** 2/18  
**Execution total:** 3/9  
**Overall total:** 5/27

## Correct next action

Continue F05 from assessment using the linked Shark Attack v2 implementation
plan. Do not treat the plan's open schema, namespace, closeout-role, or
migration decisions as approved.
