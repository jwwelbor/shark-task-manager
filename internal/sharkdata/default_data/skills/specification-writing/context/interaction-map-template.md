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
- `live` is the default gate mode. Declare `contract-only` no later than feature
  specification, and only when the row records counterpart identities,
  shared-contract evidence, activation owner, closure key, and review basis.
  Read current counterpart status live from Shark at review/UAT time; do not
  copy a lifecycle-state snapshot into the map. Reject partial or late
  declarations; report reverse build-order consumption as a decomposition warning.

## Readiness evidence shape

Every `contract-only` row has this exact nine-field readiness shape. These are
documented policy facts, not runtime fields; counterpart identities and shared
contract evidence remain staged-edge declaration metadata in the map row.

| Field | Required value/meaning |
|---|---|
| `assessor_verdict` | Independent UAT assessment, recorded without owner-decision rewrite |
| `owner_decision` | Separate approval or `override-accept` decision with conditions |
| `open_conditions` | Open activation and any recorded conditions remain visible |
| `gate_mode` | `live` by default; `contract-only` only when declared by specification |
| `activation_owner` | Later feature responsible for the real caller chain |
| `closure_key` | Tracked key that closes the activation obligation |
| `counterpart_status` | Read live from Shark at review/UAT time; never a copied static map status |
| `review_basis` | Isolated feature, accumulated branch, or another explicit review scope |
| `demonstrability_disposition` | `demonstrated-now`, `pending-integration`, or `accepted-risk-and-override` |

## Registration

After writing the map, register it with Shark:

```bash
DOC_PATH="docs/plan/<epic-id>/<epic-id>-interaction-map.md"
shark related-docs add "Interaction Map" "${DOC_PATH}" --epic=<epic-id>
```

## Exit Check

Before advancing to decomposition:

- Every I-## has a producer feature or intended producer capability.
- Every I-## has at least one consumer feature or intended consumer capability.
- Every shape source resolves to `architecture.md`.
- No orphan wires remain.
- Every `contract-only` row has all mandatory staged-edge metadata and the exact
  nine-field readiness shape.
