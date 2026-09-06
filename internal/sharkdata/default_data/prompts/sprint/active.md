{{template "advance_preamble" .}}

Drive active sprint {{.id}}: "{{.title}}" to completion.

{{include: skills/sprint-execution/SKILL.md}}

{{template "_product_critical_path_guard" .}}

Pull entities via `shark sprint next` and dispatch each per the execution skill
until the sprint backlog is drained or no further progress is possible. When the
sprint is ready to wrap up, release outcome `pass` to move it to closing.
