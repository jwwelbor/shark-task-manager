{{define "_gate_result_directive"}}FINAL RESPONSE — STRUCTURED GATE RESULT (T-E34-F05, `result_contract: gate_result_v1`):
Do NOT end your response with a `RECOMMENDED OUTCOME: <key>` line, a `PARENT NOTE: <text>` line, or a bare `<entity-id> -> <status> --reason "<why>"` kickback line — those are the legacy free-form directive grammar and this step does not use it. Instead, the ENTIRE trimmed final response must be exactly one worker-control envelope (`kind: final`), with a nested `gate_result` payload:

```
{
  "kind": "final",
  "recommended_outcome": "<the outcome key you are recommending, e.g. pass or fail>",
  "evidence": [],
  "gate_result": {
    "schema_version": 1,
    "summary": "<one paragraph: verdict and why>",
    "findings": [ ... ],
    "kickbacks": [ ... ],
    "no_kickback_reason": "<only when required — see below>"
  }
}
```

The empty arrays above are illustrative. If this prompt requires outer
`evidence` entries, include them; each is an object with `kind`, `pointer` and
`summary`.

`gate_result` fields (unknown fields are rejected):

| Object | Field | Rule |
|---|---|---|
| result | `schema_version` | `1` |
| result | `summary` | required, 1000 B max |
| result | `no_kickback_reason` | 1000 B max |
| `findings[]` | `severity` | required, 256 B max |
| `findings[]` | `class_key` | required, 256 B max |
| `findings[]` | `class_statement` | required, 1000 B max |
| `findings[]` | `fingerprint` | required, 256 B max |
| `findings[]` | `disposition` | required: `open`, `fixed`, `already_dispositioned`, `severity_conflict` or `not_reproducible` |
| `findings[]` | `disposition_pointer` | 2048 B max; required for `already_dispositioned` and `severity_conflict` |
| `findings[]` | `affected_ids` | optional strings, 256 B max each |
| `findings[]` | `evidence_pointers` | optional strings, 2048 B max each |
| `kickbacks[]` | `entity_key` | required, 256 B max |
| `kickbacks[]` | `target_status` | required, 256 B max |
| `kickbacks[]` | `reason` | required, 1000 B max |

Every array holds at most 100 items and the whole `gate_result` at most
256 KiB. Omit `remediation_sweeps` and `change_impacts` unless this prompt or
a skill it names defines their contents for this step.

Rules this step's outcomes commonly need:

For each `kickbacks[]` entry, `target_status` is required by GateResult v1
but is only a compatibility assertion. Read the target entity's current
status and its workflow step, then set `target_status` to that step's
`rework_outcomes[recommended_outcome]`. Do not choose or infer a destination
independently. If the route is missing or cannot be read, do not guess or
submit that kickback; report the missing route and return a blocked result.

- Whether `kickbacks` must be empty or non-empty for a given outcome depends
  on that outcome's configured semantic role (`outcome_roles`, visible via
  `shark get {{.id}} --json` or `shark next {{.id}} --json` ->
  `result_contract`/`outcome_roles`), not on this prompt's own text:
  - role `success` (typically `pass`): zero `kickbacks`, and no `findings`
    left with `disposition: open` or `severity_conflict`.
  - role `kickback_rework` (a per-item rework outcome): at least one
    `kickbacks` entry (`entity_key`, `target_status`, `reason`) — one per
    item that must reopen, replacing the legacy
    `<entity-id> -> <status> --reason "<why>"` line format.
  - role `route_rework` (this step's own main entity re-routes as a whole):
    zero `kickbacks` — the configured route itself is the rework path; do
    not also emit per-item kickbacks for this role.
  - roles `blocked`/`hold`/`cancelled`: either at least one kickback or a
    non-empty `no_kickback_reason` explaining why none applies.
- A blocking finding you are reporting (not fixing) must carry
  `disposition: open` (or `severity_conflict` when reviewers disagree);
  every other GateResult field constraint is enforced the same way
  regardless of which role this outcome carries.

If this step names or requires a report file, write the full narrative there
and keep `summary` a compact verdict statement. Otherwise the structured
findings and `summary` are the record; do not invent a report location.{{end}}
