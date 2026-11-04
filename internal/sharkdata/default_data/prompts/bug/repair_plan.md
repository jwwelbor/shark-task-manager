{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Prepare a repair plan for {{.id}} from {{.file_path}}, its research report, related decisions, prior findings, and affected production callers. Register the plan as a related document; identify it by path and content hash. State the diagnosis, approach, preserved behavior, constraints, scope, assertions and review baseline. Keep the approved plan immutable after approval; place evolving evidence in a separate linked record so evidence updates do not change the plan hash. Maintain one coverage index: obligation, authoritative source, production caller or boundary, planned change, required assertions, and links to a separate mutable evidence record. {{template "_bug_repair_contract" .}}
Simple bugs may use brief evidence; recurrent, uncertain, stateful, accounting and cross-boundary bugs require deeper analysis. Return needs_research for a causal gap, blocked for unavailable external evidence, and pass only for a reviewable plan. The parent owns transitions.
Keep the project plan document in `docs/plan`, tracked and committed in its project checkout. Local review reports and development artifacts can be retained separately without committing them.

Use fail for an unresolved plan. The fail outcome's role is `route_rework`; leave kickbacks empty and retain findings.

{{template "_gate_result_directive" .}}
