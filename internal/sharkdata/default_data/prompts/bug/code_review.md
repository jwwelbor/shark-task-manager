{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Review bug {{.id}} against the accepted repair plan, research, coverage index and all unaccepted predecessor changes. Bind review evidence to the complete candidate and plan revision. Before advancing an in-flight bug, inspect existing evidence and return to the earliest step with missing or invalid evidence; preserve code and history. Verify the approved plan content hash matches the current plan. Review the plan's single coverage index, which maps each obligation to its authoritative source, production caller or boundary, planned change, required assertions, and links to a separate mutable evidence record. Do not change the plan while reviewing its revision. {{template "_bug_repair_contract" .}}

For a structured pass, retain the complete scope JSON produced by the deep-review capture in a project-relative ignored location such as `.shark/runs/`, and include exactly one outer evidence entry with `kind: bug_review_scope`, its project-relative `pointer`, and `summary: sha256:<hex digest>` of the retained scope bytes. The parent checks the scope and current source candidate before advancing. Keep the same scope artifact for QA.

Also retain the generated deep-review report and include a digest-bound `bug_review_report` evidence reference in the same format. For a native or dispatched six-angle review, retain the actual six angle-results JSON and include a digest-bound `bug_review_angles` reference. A comparable alternate-model CLI review uses its report without angle results. An incomplete review cannot pass.
Use review_plan for a valid plan whose approval is missing or stale; use needs_research for a causal contradiction, replan for a design omission, fail for a coding defect, and blocked for unavailable external evidence. Retain every finding; route to the earliest invalidated stage. For route_rework outcomes leave kickbacks empty.

{{template "_code_review_process_body" .}}

For re-review, run the full-class re-verification procedure in
`skills/quality/workflows/defect-class-sweep.md` (if the bundle is not extracted, read it with `shark skill get quality workflows/defect-class-sweep.md`). Its evidence-based recurrence
classification and the project's retry limits govern escalation; do not treat
a raw review round count alone as recurrence. Preserve accepted dispositions
unless new evidence invalidates them.


The fail outcome's role is `route_rework`. Return pass only when all checks and complete current evidence pass; otherwise choose the earliest invalidated stage as above. The parent applies the outcome.

{{template "_gate_result_directive" .}}
