# Cross-Epic Integration Map Template

Use this template for `docs/product/cross-epic-integration-map.md`. It is the
product-level source of truth for stable `X-##` integrations that cross epic
boundaries. Keep it separate from per-epic `I-##` cross-feature interaction
maps.

## File Name

`docs/product/cross-epic-integration-map.md`

## Table

| ID | Producer epic | Consumer epic(s) | Integration purpose | Contract / shape source | UX / CX handoff notes | Owning feature | Status | Test coverage pointer |
|----|---------------|------------------|---------------------|-------------------------|-----------------------|----------------|--------|-----------------------|
| X-01 | E01 producer epic | E02 consumer epic | Shared capability or handoff purpose | docs/architecture/example.md#contract-section | Journey continuity and handoff notes | TBD until decomposition | proposed | TBD |

## Rules

- `X-##` IDs are stable. Assign each ID once, then reference it verbatim in
  epic design, epic decomposition, feature specs, test plans, producing or
  consuming task specs, feature review, and QA.
- Use `X-##` only for cross-epic integrations. Use `I-##` for cross-feature
  interactions inside one epic.
- Producer epic and consumer epic(s) must be known before a row is accepted.
- Contract / shape source must resolve to the canonical architecture, API,
  schema, product journey, or UX artifact that defines the shared shape.
- UX / CX handoff notes capture journey continuity, state continuity, handoff
  copy, and product workflow implications.
- Owning feature may be `TBD` during epic design, but decomposition must assign
  producer and consumer features before exit.
- Test coverage pointer may be `TBD` while proposed or assigned. A row cannot
  move to `covered` until a test plan, test case, or test file pointer exists.

## Registration

After creating or updating the map, update `docs/product/progress.md`:

- Mark the global map as present.
- Set `Last updated` to the current date.
- Set `Updated by` to the epic and design decision that changed the map.
- Append a decision-log entry that names the added, changed, assigned, covered,
  or deferred `X-##` rows.

## Exit Check

Before advancing epic design:

- Every relevant cross-epic dependency has an `X-##` row or an explicit
  "none found" note in the epic design output.
- Every `X-##` row names producer and consumer epic(s).
- Every `X-##` row has a contract / shape source or an explicit deferral
  decision in `docs/product/progress.md`.
- CX handoff notes are reviewed or marked "not applicable" with a reason.
