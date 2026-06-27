# Interaction Map Template

Use this template when an epic is expected to decompose into 3+ features or when
the epic has explicit cross-feature handoffs. Skip it for single-feature epics
and small epics with no cross-feature contract.

## File Name

`<epic-id>-interaction-map.md`

Example: `E34-interaction-map.md`

## Table

| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |
|----|------------------|---------------------|-------|---------|-------|
| I-01 | F01 producer capability | F02 consumer capability | architecture.md#contract-section | Contract payload name and required fields | API/event/artifact/etc. |

## Rules

- I-## IDs are stable. Assign each ID once, then reference it verbatim in epic
  decomposition, feature specs, test plans, producing/consuming task specs,
  feature review, task review, and QA.
- Shape must be a link or anchor that resolves to a section in `architecture.md`
  or a linked architecture section.
- Producer and consumer features must reference the same shape source verbatim.
- Payload names the concrete data, event, file, state, API, CLI output, or
  artifact crossing the feature boundary.
- Style names the interaction mode, such as API, event, shared data model, file
  artifact, CLI contract, UI handoff, or another concrete mechanism.
- Contract tests are shared by producer and consumer features. Do not create
  twin tests for the same I-##.

## Registration

After writing the map, register it with Shark:

```bash
shark related-docs add "Interaction Map" "<epic-id>-interaction-map.md" --epic=<epic-id>
```

## Exit Check

Before advancing to decomposition:

- Every I-## has a producer feature or intended producer capability.
- Every I-## has at least one consumer feature or intended consumer capability.
- Every shape source resolves to `architecture.md`.
- No orphan wires remain.
