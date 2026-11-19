{{template "advance_preamble" .}}

Drive active sprint {{.id}}: "{{.title}}" to completion.

{{include: skills/sprint-execution/SKILL.md}}

{{template "_product_critical_path_guard" .}}

Select entities within this sprint via `shark plan {{.id}} --json` and dispatch
each per the execution skill. If no dispatchable entity is selected, report that
sprint drain was not verified. When the sprint is ready to wrap up, release
outcome `pass` to move it to closing.
