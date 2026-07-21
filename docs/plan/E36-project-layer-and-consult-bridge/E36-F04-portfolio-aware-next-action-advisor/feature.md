---
feature_key: E36-F04-portfolio-aware-next-action-advisor
epic_key: E36
title: Portfolio-Aware Next-Action Advisor
description: Keyed shark next resolves work inside a caller-selected root, but Shark has no project-wide way to assemble the known epic graph and product context needed to choose that root. Add a read-only bare shark next mode that returns portfolio state, dependency and ordering evidence, and a prompt that directs an agent to inspect docs/product and recommend what should come next. Keep keyed shark next behavior unchanged. The existing cross-entity implementation-plan idea is related prior art, not a duplicate.
---

# Portfolio-Aware Next-Action Advisor

**Feature Key**: E36-F04-portfolio-aware-next-action-advisor

---

## Epic

- **Parent epic**: [E36 — Project Layer and Consult Bridge](../epic.md)

---

## Goal

### Problem

`shark next <key>` can resolve the next workflow step within a supplied root,
but it does not decide which epic should be supplied. Shark Rider therefore
depends on the operator already knowing which initiative deserves attention.
That becomes ambiguous when several epics are active, no sprint supplies the
next item, or stored priority and progress values do not express product order.

### Solution

Add a read-only portfolio-advice mode to the existing `shark next` command.
Bare `shark next` assembles the known Shark dependency graph, stored ordering,
live epic state, and a prompt that directs an agent to inspect `docs/product/`
and recommend what should come next. Keyed `shark next <key>` keeps its current
dispatch behavior.

### Command contract

`shark next` supports two modes:

| Command | Purpose | Mutation behavior |
| --- | --- | --- |
| `shark next` | Return portfolio evidence and an advisory prompt. | Read-only. It must not change entities, workflow status, relationships, claims, or history. |
| `shark next <key>` | Resolve the next dispatch inside the supplied root. | Existing behavior. It may normalize cascade-complete parents or agentless `advance_status` steps. |

Bare `shark next` returns one portfolio-advice envelope with:

- non-terminal epics and their status, priority, business value, progress,
  blockers, claims, and current work in progress;
- `depends_on`, `blocks`, and `follows` relationships;
- deterministic dependency and ordering layers derived from Shark data;
- warnings for cycles, contradictory relationships, and missing ordering data;
- a rendered prompt for choosing one epic root.

The response has this conceptual shape; the specification defines exact field
types and validation rules:

```json
{
  "mode": "portfolio_advice",
  "epics": [],
  "relationships": [],
  "ordering": {
    "layers": [],
    "warnings": []
  },
  "prompt": "Inspect docs/product and this Shark state, then recommend one epic root."
}
```

### Advisory prompt contract

The prompt directs the receiving agent to:

1. Inspect the relevant artifacts under `docs/product/` for product intent,
   roadmap context, and cross-epic constraints.
2. Treat the returned Shark data as authoritative for entity state,
   dependencies, relationships, blockers, and active work.
3. Treat product documents as decision context, not a second workflow or
   status store.
4. Recommend one epic root, explain why it should come next, and compare it
   with the strongest alternative.
5. Report missing evidence or contradictory ordering instead of guessing.

Go code performs deterministic graph extraction, ordering, and warning
generation. The prompt delegates the product judgment that combines those
facts with `docs/product/`; Go code does not hard-code the final recommendation.

### Success conditions

- `shark next` accepts no entity argument and returns the portfolio-advice
  envelope.
- The no-argument mode performs no database writes, including lease cleanup.
- `shark next <key>` retains its current dispatch response and normalization
  behavior.
- The graph includes every relevant stored epic relationship and produces
  stable ordering layers for the same database state.
- The prompt states the authority boundary between Shark state and product
  documentation and requests one evidence-backed root recommendation.
- The command reports cycles, contradictory edges, and incomplete evidence
  without inventing a total order.

### Out of scope

- Reintroducing `shark next --preview`.
- Selecting work across multiple Shark projects.
- Adding a second roadmap or workflow state store under `docs/product/`.
- Automatically claiming or dispatching the recommended root.

### Impact

An operator or agent can move from project state to a justified epic root
without guessing. The same command now separates portfolio advice from keyed
workflow dispatch by argument shape.

## Triage Context

- This is feature-sized work under E36's project-level coordination boundary;
  it is not a new portfolio-management epic.
- Bare `shark next` is the primary command surface. State-aware
  `/shark-rider help` may consume its response; this feature does not add a
  separate Rider advisor action.
- Idea `I-2026-01-02-06` ("add implementation plan") is related prior art for
  cross-entity ordering, but it is broader than selecting a root epic for Rider.
- Full roadmap modeling and cross-project implementation planning remain out of
  scope. The normal feature workflow defines the implementation tasks.

---

*Last Updated*: 2026-07-20
