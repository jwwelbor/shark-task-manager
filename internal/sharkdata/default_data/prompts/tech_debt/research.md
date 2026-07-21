{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research tech debt {{.id}}: "{{.title}}" using the v2 universal recipe
catalog. Write only `{{.id}}.research-report.md` beside {{.file_path}} and
register it as a related document. Front matter must include
`research_schema: 2`, `entity_key`, `entity_type: tech_debt`,
`recipe: universal`, `rigor`, selected `categories`, and `related_work`.

Include Scope, Research checklist, Findings, Decisions, and Sources. The
checked checklist is the plan and completion record: every selected catalog
module needs a checked entry with concrete `Evidence:` paths. Always select
`scope_vocabulary` and `affected_implementation_or_contract`; standard adds
`pattern_contract` or `dependency_impact`; complex also adds
`cross_boundary_risks` and `alternatives`.

Record existing remediation and the affected capability, including whether the
work extends or replaces it. Add a Capability map only when related work is
discovered or changed, with REUSE, EXTEND, NEW, or CONTRADICTS decisions.
Return `pass` only after the one report is complete.
