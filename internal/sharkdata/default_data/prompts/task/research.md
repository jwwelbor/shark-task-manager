{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research task {{.id}}: "{{.title}}" with the v2 universal recipe catalog.
Write only `{{.id}}.research-report.md` beside {{.file_path}} and register it
as a related document. Its front matter must include `research_schema: 2`,
`entity_key`, `entity_type: task`, `recipe: universal`, `rigor`, selected
`categories`, and `related_work`.

Include Scope, Research checklist, Findings, Decisions, and Sources. The
checked checklist is the plan and completion record: each selected catalog
module must be checked and include concrete `Evidence:` paths. Always select
`scope_vocabulary` and `affected_implementation_or_contract`; standard also
selects `pattern_contract` or `dependency_impact`; complex also selects both
`cross_boundary_risks` and `alternatives`.

For simple work, cite the parent feature Capability map in Findings or Sources
instead of copying it. Add a Capability map only if this task discovers or
changes related capability work, then make REUSE, EXTEND, NEW, or CONTRADICTS
decisions. Record the concrete affected implementation or contract and whether
the task extends it or introduces new work. Return `pass` only after the one
report is complete.
