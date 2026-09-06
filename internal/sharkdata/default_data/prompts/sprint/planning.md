{{template "advance_preamble" .}}

Plan sprint {{.id}}: "{{.title}}".

{{include: skills/sprint-planning/SKILL.md}}

{{template "_product_critical_path_guard" .}}

Read the sprint plan and readiness data (`shark sprint plan {{.id}} --json`),
propose entity assignments, and confirm scope. Do NOT start the sprint — starting
is an explicit user action. When scope is agreed, release outcome `pass` to make
the sprint active.
