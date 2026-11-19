{{template "advance_preamble" .}}

Drive active sprint {{.id}}: "{{.title}}" to completion.

{{include: skills/sprint-execution/SKILL.md}}

{{template "_product_critical_path_guard" .}}

Pull entities via `shark sprint next` and dispatch each per the execution skill.
An empty selection means no candidate is currently eligible; inspect the full
unfiltered all-status sprint backlog and each entity's configured terminal
status before calling the sprint drained. If any assignment remains
nonterminal or terminality cannot be verified, report the sprint as paused and
do not release outcome `pass` as if it were complete. An empty `--agent` slice
may leave unfinished assignments for other roles. Only after every assigned
entity is verified terminal should the owner follow the existing Sprint Goal
Review and close gates and release outcome `pass` to move the sprint to closing.
