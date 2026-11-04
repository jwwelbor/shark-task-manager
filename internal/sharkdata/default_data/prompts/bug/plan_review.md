{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Independently review the exact revision of the repair plan for {{.id}} against the research, code, production callers, and accepted decisions. Record an approval report as a related document containing the plan path and content hash, reviewer identity and findings, including preservation of cost, model and other existing behavior where applicable. Review the plan's single coverage index, which maps each obligation to its authoritative source, production caller or boundary, planned change, required assertions, and links to a separate mutable evidence record. Do not change the plan while reviewing its revision. {{template "_bug_repair_contract" .}}
Approve only when the coverage index and meaningful assertions cover the obligations. Use needs_research for causal gaps, replan for approach or coverage gaps, blocked for unavailable external evidence, and pass for approval. For route_rework outcomes leave kickbacks empty and retain all findings in the summary. The parent owns transitions.

The fail outcome's role is `route_rework`.

{{template "_gate_result_directive" .}}
