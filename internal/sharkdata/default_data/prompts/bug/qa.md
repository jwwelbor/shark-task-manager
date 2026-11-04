{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Verify bug {{.id}} against its accepted plan, coverage index and reviewed candidate. Execute production-caller scenarios and boundary and preservation assertions; source review alone is insufficient. Confirm candidate and plan evidence are current. Before advancing an in-flight bug, inspect existing evidence and return to the earliest step with missing or invalid evidence; preserve code and history. Verify the approved plan content hash matches the current plan. Review the plan's single coverage index, which maps each obligation to its authoritative source, production caller or boundary, planned change, required assertions, and links to a separate mutable evidence record. Do not change the plan while reviewing its revision. {{template "_bug_repair_contract" .}}
Use review_plan for a valid plan whose approval is missing or stale; use needs_research for causal contradictions, replan for plan defects, fail for implementation defects, and blocked for unavailable external evidence. Retain every finding and route to the earliest invalidated stage. For route_rework outcomes leave kickbacks empty.

{{template "_qa_process_body" .}}

The fail outcome's role is `route_rework`. Return pass only when all checks and complete current evidence pass; otherwise choose the earliest invalidated stage as above. The parent applies the outcome.

{{template "_gate_result_directive" .}}
