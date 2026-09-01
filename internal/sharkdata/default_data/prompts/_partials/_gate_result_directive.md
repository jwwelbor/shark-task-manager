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

The nested `gate_result` shape is validated and typed in `internal/gateresult`
(see its doc comment for the full field list — `findings[]`,
`kickbacks[]`, `remediation_sweeps[]`, `change_impacts[]`); this note does
not restate it. Two rules this step's outcomes commonly need:

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

Write your full narrative report to the file this step already directs
(unchanged); the envelope's `summary` is a compact verdict statement, not a
duplicate of that report.{{end}}
