{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research sprint {{.id}}: "{{.title}}" before activation using the universal
recipe catalog. Write adjacent `{{.id}}.research-plan.md` and
`{{.id}}.research-report.md` sidecars beside {{.file_path}}. Both front-matter
blocks must identify `entity_key`, `entity_type: sprint`, `recipe: universal`,
`rigor`, `categories`, `source_set`, and `related_work`. The plan requires
Scope, Recipe, Source set, and Steps. The report requires Scope, Capability
map, Ubiquitous vocabulary, Findings, Decisions, and Sources.

Use workflow/operations as a category and add only the other applicable
modules. Inspect assigned and related work where it exists, recording REUSE,
EXTEND, NEW, or CONTRADICTS in a non-empty Capability map. Register both
sidecars as related docs, then return `pass` to activate the sprint.
