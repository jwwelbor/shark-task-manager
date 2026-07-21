{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research feature {{.id}}: "{{.title}}" using the v2 universal recipe catalog
at `shark-data/research/recipes.yaml`. Read the parent report, parent PRD,
siblings, {{.file_path}}, and only the category-specific sources selected by
the report checklist.

Write only `research-report.md` beside the feature file and register that
report as a related document on {{.id}}. Its front matter must include
`research_schema: 2`, `entity_key`, `entity_type: feature`,
`recipe: universal`, `rigor`, selected `categories`, and `related_work`.

Include Scope, Research checklist, Capability map, Findings, Decisions, and
Sources. The checked checklist is both the research plan and completion
record: every selected catalog module needs a concrete `Evidence:` path.
Always select `scope_vocabulary`, `affected_implementation_or_contract`, and
`related_work`; standard adds `pattern_contract` or `dependency_impact`; and
complex also adds `cross_boundary_risks` and `alternatives`.

The Capability map must decide REUSE, EXTEND, NEW, or CONTRADICTS for relevant
parent, sibling, and related capabilities. Cite upstream reports rather than
copying them, and record whether this feature extends or creates a capability.
Route from rigor: SIMPLE ends `RECOMMENDED OUTCOME: simple`; STANDARD ends
`RECOMMENDED OUTCOME: standard`; COMPLEX ends `RECOMMENDED OUTCOME: pass`.
