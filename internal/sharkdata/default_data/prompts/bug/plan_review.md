{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

{{template "_durable_sources" .}}

Independently review the exact revision of the repair plan for {{.id}} against the research, code, production callers, and accepted decisions. Record an approval report as a related document containing the plan path and content hash, reviewer identity and findings, including preservation of cost, model and other existing behavior where applicable. Review the plan's single coverage index, which maps each obligation to its authoritative source, production caller or boundary, planned change, required assertions, and links to a separate mutable evidence record. Do not change the plan while reviewing its revision. Return `replan` when the plan cites host agent memory instead of stating the instruction or citing a committed file. {{template "_bug_repair_contract" .}}

{{template "_bug_decision_sources" .}}

Independently reconcile the decision sources with the coverage index. For each row without an authorized disposition, check that its assertion would fail if the protected behavior were lost at the named caller while adjacent checks still pass. For preservation constraints, compare the exact expected values or observable behavior at every affected caller. Return `replan` for an omitted applicable constraint or an assertion that would pass despite a violation.

For approval, include exactly one `plan_review_report` evidence entry with the project-relative path of the linked report as `pointer` and `sha256:<hex digest>` of its bytes as `summary`. In the report include exactly one plain `plan_path: <project-relative plan path>` line and one `plan_sha256: sha256:<hex digest>` line identifying the reviewed revision. The parent checks the report is linked to this bug and retains the approved plan snapshot for downstream workers.
Approve only when the coverage index and meaningful assertions cover the obligations. Use needs_research for causal gaps, replan for approach or coverage gaps, blocked for unavailable external evidence, and pass for approval. For route_rework outcomes leave kickbacks empty, keep the summary concise, and retain findings in the structured findings and review report. The parent owns transitions.

The fail outcome's role is `route_rework`.

{{template "_gate_result_directive" .}}

REQUIRED OUTER EVIDENCE for a success outcome of this step (this replaces the empty array shown above; replace every `<...>` placeholder, and copy parent-supplied approved entries exactly):

```
  "evidence": [
    {"kind": "bug_repair_plan", "pointer": "<project-relative plan file>", "summary": "sha256:<hex digest of the plan>"},
    {"kind": "bug_review_baseline", "pointer": "<full commit hash of the plan's review baseline>"},
    {"kind": "plan_review_report", "pointer": "<project-relative linked report>", "summary": "sha256:<hex digest of its bytes>"}
  ]
```
