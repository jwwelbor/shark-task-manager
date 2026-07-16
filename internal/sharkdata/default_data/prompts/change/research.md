{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research change card {{.id}}: "{{.title}}" using the universal recipe
catalog. Write adjacent `{{.id}}.research-plan.md` and
`{{.id}}.research-report.md` sidecars beside {{.file_path}}. Both front-matter
blocks must identify `entity_key`, `entity_type: change`, `recipe: universal`,
`rigor`, `categories`, `source_set`, and `related_work`. The plan requires
Scope, Recipe, Source set, and Steps. The report requires Scope, Capability
map, Ubiquitous vocabulary, Findings, Decisions, and Sources.

Always establish vocabulary and examine related work when it exists. The
Capability map must make REUSE, EXTEND, NEW, or CONTRADICTS decisions without
copying upstream material. Select only relevant category modules, register both
sidecars as related docs, then return `pass`.
