{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research epic {{.id}}: "{{.title}}" using the v2 universal recipe catalog at
`shark-data/research/recipes.yaml`.

Write only `research-report.md` beside {{.file_path}} and register that report
as a related document on {{.id}}. Its front matter must include only
`research_schema: 2`, `rigor`, a `categories` list drawn only from the
recipe's category vocabulary (e.g. `frontend`, `backend`, `api`, `data`,
`workflow_operations`, `documentation` — never a module ID), and a boolean
`related_work` front matter field (`true` or `false`, not a list — distinct
from the `related_work` checklist module below). Do not include `entity_key`,
`entity_type`, or `recipe` — shark already knows the entity and there is only
one recipe.

The report must contain Scope, Research checklist, Capability map, Findings,
Decisions, and Sources. The checked Research checklist is the plan and the
record of completion: select the applicable atomic catalog modules, check each
one only after completion, and add concrete `Evidence:` paths to every entry.
Each entry must be a single markdown checkbox line, never a heading, in this
exact form: `- [x]` followed by the module id in backticks, then
`— Evidence: <path>` (e.g. `- [x] scope_vocabulary — Evidence: path/to/file`).
Always select `scope_vocabulary`, `affected_implementation_or_contract`, and
`related_work`; simple work stops there, standard work also selects
`pattern_contract` or `dependency_impact`, and complex work also selects both
`cross_boundary_risks` and `alternatives`.

Inspect parent/sibling/related capability evidence and make a Capability map
decision for each relevant capability: REUSE, EXTEND, NEW, or CONTRADICTS.
Record the brownfield evidence and whether this extends an existing capability
or creates a new one. Return `pass` only after the one report is complete.
