---
type: cross-epic-integration-map
---
# Cross-Epic Integration Map

This product-level map tracks stable `X-##` integration rows that cross epic
boundaries. Keep per-epic `I-##` cross-feature interaction maps separate.

## Integration Rows

| ID | Producer epic | Consumer epic(s) | Integration purpose | Contract / shape source | UX / CX handoff notes | Owning feature | Status | Test coverage pointer |
|----|---------------|------------------|---------------------|-------------------------|-----------------------|----------------|--------|-----------------------|
| X-01 | TBD | TBD | TBD | TBD | TBD | TBD | proposed | TBD |

## Status Values

- `proposed`: identified during epic design, but feature ownership is not yet assigned.
- `assigned`: producer and consumer feature ownership is known.
- `covered`: test coverage exists and is linked.
- `deferred`: intentionally postponed with a decision-log entry.

## Rules

- Assign each `X-##` once. Do not renumber existing rows.
- Use `X-##` only for cross-epic integrations. Use `I-##` for cross-feature
  interactions within one epic.
- Link the contract / shape source to the canonical architecture, API, schema,
  product journey, or UX artifact that defines the shared shape.
- Name producer and consumer epics during epic design. Add owning features
  during decomposition or feature specification when ownership becomes known.
- Keep UX / CX handoff notes focused on journey continuity, handoff copy,
  state continuity, and product workflow implications.
- Add or update the test coverage pointer before setting a row to `covered`.
