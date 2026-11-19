{{template "advance_preamble" .}}

Drive active sprint {{.id}}: "{{.title}}" to completion.

{{include: skills/sprint-execution/SKILL.md}}

{{template "_product_critical_path_guard" .}}

Select entities within this sprint via `shark plan {{.id}} --json` and dispatch
each per the execution skill. If no dispatchable entity is selected, read the
full unfiltered assignment set with `shark sprint backlog {{.id}}
--view=ordered --all --json` and resolve each entity type's configured terminal
statuses. Report the sprint drained and release outcome `pass` only after that
complete read confirms every assigned entity is terminal. If the read is
incomplete, a status or terminal marker cannot be resolved, or assignments
remain, pause and report that sprint drain was not verified. An empty
`--agent` slice does not prove the sprint is drained. Keep the Sprint Goal
Review and explicit close confirmation as separate gates.
