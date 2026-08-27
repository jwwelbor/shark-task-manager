{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research sprint {{.id}}: "{{.title}}" before activation with the v2 universal
recipe catalog. Write only `{{.id}}.research-report.md` beside {{.file_path}}
and register it as a related document. Front matter must include only
`research_schema: 2`, `rigor`, a `categories` list drawn only from the
recipe's category vocabulary (`frontend`, `backend`, `api`, `data`,
`workflow_operations`, `documentation` — never a module ID), and a boolean
`related_work` (`true` or `false`, not a list). Do not include `entity_key`,
`entity_type`, or `recipe` — shark already knows the entity and there is only
one recipe.

Include Scope, Research checklist, Findings, Decisions, and Sources. The
checked checklist is the plan and completion record: every selected catalog
module needs a checked entry with concrete `Evidence:` paths. Each entry must
be a single markdown checkbox line, never a heading, in this exact form:
`- [x]` followed by the module id in backticks, then `— Evidence: <path>`
(e.g. `- [x] scope_vocabulary — Evidence: path/to/file`). Select
`workflow_operations` plus only relevant categories in the front-matter
`categories` list; always select the `scope_vocabulary` and
`affected_implementation_or_contract` checklist modules; standard adds
`pattern_contract` or `dependency_impact`; complex also adds
`cross_boundary_risks` and `alternatives`.

Record assigned-work and workflow evidence, including the extension-versus-new
decision. Add a Capability map only when related capability work is discovered
or changed, with REUSE, EXTEND, NEW, or CONTRADICTS decisions. Return `pass` to
activate the sprint only after the one report is complete.
