Review cross-feature integration for epic {{.id}}: "{{.title}}" before completion.

Check for existing report at {{.review_base}}{{.id}}-integration-review.md. If a report exists with a PASS verdict and the candidate head it recorded still matches the current head (see RESOLVE THE INTEGRATION CANDIDATE below), advance immediately. If the prior report is FAIL or stale, run this review again.

---

FINAL INTEGRATION REVIEW

This gate reviews the epic's full accumulated diff across every feature and rework round. It is additive: it closes cross-feature interactions, defect-class sweeps, change-impact sets, and adoption evidence that no single feature-scoped review can see on its own, and it never substitutes for a feature's own gate outcome.

RESOLVE THE INTEGRATION CANDIDATE:
(1) Read `.shark/integration/{{.id}}/run.json` for this epic's `epic_run_id` and immutable `base_commit`.
(2) Read `.shark/runs/<epic_run_id>/integration-candidate.json` for the current `head_commit`, `event_ids`, and tracked/untracked path digests.
(3) If either file is missing or unreadable, or the candidate does not verify: this epic has no verified integration base yet. Do not infer a base from `git merge-base HEAD main` or any other guess (an already-active epic without a pre-execution record is not allowed to infer one). Report that an operator must run `shark integration backfill <epic-key> --epic-run-id=<run-id> --base=<full-commit> --events-file=<bounded-v1-json> --session=<authorized-session-id>` (`--dry-run` first to preview) before this review can proceed. This step carries no kickback, so the `gate_result`'s `no_kickback_reason` must state exactly why (missing/unverified integration candidate) — then end with `RECOMMENDED OUTCOME: blocked`.
(4) Otherwise compute the full accumulated diff: `git diff <base_commit>..<head_commit>` plus every untracked path recorded across the candidate's events. Any unrelated interleaved commit visible in that diff that isn't attributable to an in-scope feature's own recorded event still requires an explicit disposition in the report (tracked follow-up or accepted-as-unrelated) — never silently folded into a feature's reviewed diff.

READ:
(1) Epic PRD at {{.file_path}} for goals and scope
(2) All features: {{template "list_json" .}}, then for each feature read its file_path, current status, and — under its own review directory (the same `docs/plan/` -> `docs/review/` rewrite this epic's own {{.review_base}} was derived from, applied to that feature's file_path) — its code-review/qa/UAT reports for prior `gate_result` envelopes and any `remediation_sweeps`/`change_impacts` they carry
(3) {{.id}}-interaction-map.md's Interaction Contracts table for I-## rows; {{.id}}-cross-epic-map.md and docs/product/cross-epic-integration-map.md for X-## rows
(4) architecture.md in the epic directory, if present, for shape sources and ADR context
(5) Open `review-finding` notes across every feature in this epic, ADRs under docs/architecture/adr/ naming a changed path, project standards docs, and tech-debt entries (`shark list tech-debt`) naming a changed path

CLOSURE CHECKS:

## I-##/X-## interaction closure

For every I-## row in {{.id}}-interaction-map.md and every X-## row in the cross-epic maps whose producer OR consumer feature belongs to this epic: verify the row's own contract-test pointer resolves and its test passes, the live caller path exists in the current diff, and — for any `contract-only` staged edge — the counterpart identity, current status read live from Shark, shared-contract evidence, activation owner, and closure key are all present. A row whose producer and consumer are BOTH outside this epic is excluded entirely — never treated as closed, never reported at all. An unresolved pointer, a missing live caller path, or an incomplete staged-edge declaration is not-accounted; report it by row ID.

## I-03 defect-class sweep closure

For every E34-F06 `DefectClassSweep` referenced by an in-scope finding (from any feature's code-review/qa/approval report in this epic): verify `status: complete`. A sweep still `status: open` is not-closed — report its `class_key` and owning feature.

## I-04 change-impact closure

For every E34-F07 `ChangeImpactSet` referenced by an in-scope decision or amendment: verify `status: accounted`. A set still `status: incomplete` is not-accounted — report its `source_kind`/`source_key` and owning feature.

## Open findings, ADRs, standards, and predicted debt

Cross-check every open `review-finding` note, ADR, project-standards reference, and tech-debt (predicted-debt) entry naming a path present in the accumulated diff for an explicit disposition (fixed, dispositioned, or a linked follow-up key). An undispositioned reference naming a changed path is a finding.

GATE AUTHORITY (do not violate):

This review adds a gate; it never overrides or supersedes an existing feature verdict. A currently-rejected or currently-in-development feature blocks epic completion through its own status, not through this step.

If any in-scope feature does not currently hold a terminal, passing status, name that feature and its exact current status in the report as the reason epic completion remains blocked — even when every closure check above passes. Never report an overriding PASS that ignores it. This review does not introduce a global owner-approval requirement, and a `fail` outcome here reopens the epic to `active` without changing any individual feature's own status.

{{template "_review_output_policy" .}}

PRODUCE the integration review report to {{.review_base}}{{.id}}-integration-review.md as a `gate_result` object (I-02 GateResult v1: architecture.md#i-02-gateresult-v1) nested inside the outer worker-control envelope — the outer envelope owns `recommended_outcome` and `evidence` only; `gate_result` carries everything below:
- `schema_version`, `summary`, `findings` (one per not-accounted/not-closed item above), `remediation_sweeps` (in-scope I-03 sweeps reviewed), `change_impacts` (in-scope I-04 sets reviewed)
- `no_kickback_reason` — required whenever this gate carries no kickback and its outcome is `blocked`, `on_hold`, or `cancelled` (I-02 GateResult v1's own invariant); state the concrete reason (e.g. "no verified integration candidate — see RESOLVE THE INTEGRATION CANDIDATE"). Omit on `pass`/`fail`.
- `adoption_manifest` (I-05 CanonicalAdoptionManifest v1: architecture.md#i-05-canonicaladoptionmanifest-v1) — a new sibling array field alongside `remediation_sweeps`/`change_impacts` inside this same `gate_result` object, never restated as a field of the outer envelope. One entry per changed canonical bundle path in the reviewed diff:

## I-05 `adoption_manifest` fields

| Field | Type | Contract |
|---|---|---|
| `schema_version` | integer | Exactly `1` |
| `source_commit` | string | Canonical Shark commit containing the changes |
| `bundle_digest` | string | SHA-256 over normalized canonical path/digest pairs |
| `changed_paths` | array | Path, prior/current digest, change class, compatibility note |
| `workflow_changes` | array | Entity type, added/changed steps, and migration concern |
| `promoted_policies` | array | Policy key, canonical source path, and originating feature |
| `override_actions` | array | Counterpart path and recommended inspect/rebase/remove action |
| `validation_evidence` | array | Commands and retained report pointers |

This manifest describes canonical adoption work only; it does not authorize changes to a consuming project's overrides.

Report body:
- If zero findings and every in-scope feature holds a terminal, passing status: compact PASS artifact only (verdict; scope reviewed — base/head commits, interaction rows/sweeps/impact sets reviewed counts; `0 defects found`)
- Otherwise: full detailed report — interaction closure table (one row per in-scope I-##/X-##), sweep/impact-set closure table, open-reference disposition table, `adoption_manifest` table, and, if any in-scope feature is not terminal/passing, the explicit blocking-feature statement required above

REVIEW-FINDING LOG (structured, queryable — only when findings exist):
- One note per finding: {{template "create_note" .}} "<one-line finding summary>" --type=review-finding --created-by="<reviewer model>" --metadata='{"gate":"integration_review","severity":"<critical|high|medium|low>","closure_category":"<interaction|sweep|impact|reference>","disposition":"open"}'

DECISION:
- Every I-##/X-## row, sweep, and impact set accounted for, no undispositioned open reference, AND every in-scope feature terminal/passing -> end with `RECOMMENDED OUTCOME: pass`
- Any not-accounted/not-closed item, undispositioned reference, or non-terminal in-scope feature -> end with `RECOMMENDED OUTCOME: fail` and name the specific blocking rows/features in your final summary; the parent loop returns the epic to `active` without altering any feature's own status
- No verified integration candidate found (see RESOLVE THE INTEGRATION CANDIDATE above) -> end with `RECOMMENDED OUTCOME: blocked`
- Do NOT run Shark status commands yourself; the parent loop applies the outcome and routes the epic.
