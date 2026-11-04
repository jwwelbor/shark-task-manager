{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Prepare a repair plan for {{.id}} from {{.file_path}}, its research report, related decisions, prior findings, and affected production callers. Register the plan as a related document; identify it by path and content hash. State the diagnosis, approach, preserved behavior, constraints, scope, assertions and review baseline. Maintain one coverage index: obligation, authoritative source, production caller or boundary, planned change, required assertions, and current evidence. Reference existing decisions; escalate conflicting authority to its owner. Keep all findings and route to the earliest invalidated stage. Diagnosis changes invalidate the plan and downstream evidence; material plan changes invalidate approval; candidate changes invalidate review and QA. Reuse unchanged valid evidence. Unavailable external evidence is blocked, not acceptance.
Simple bugs may use brief evidence; recurrent, uncertain, stateful, accounting and cross-boundary bugs require deeper analysis. Return needs_research for a causal gap, blocked for unavailable external evidence, and pass only for a reviewable plan. The parent owns transitions.

End with `RECOMMENDED OUTCOME: pass`, `RECOMMENDED OUTCOME: needs_research`, `RECOMMENDED OUTCOME: fail`, or `RECOMMENDED OUTCOME: blocked` as appropriate. Use fail for an unresolved plan.
