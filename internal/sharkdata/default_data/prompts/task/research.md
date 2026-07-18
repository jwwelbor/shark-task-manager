{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research task {{.id}}: "{{.title}}" using the universal recipe catalog.
Write adjacent sidecars `{{.id}}.research-plan.md` and
`{{.id}}.research-report.md` beside {{.file_path}}. Both front-matter blocks
must identify `entity_key`, `entity_type: task`, `recipe: universal`, `rigor`,
`categories`, `source_set`, and `related_work`. The plan must contain Scope,
Recipe, Source set, and Steps. The report must contain Scope, Capability map,
Ubiquitous vocabulary, Findings, Decisions, and Sources.

Always establish shared vocabulary. Inspect related work and existing
capabilities when they exist; then record REUSE, EXTEND, NEW, or CONTRADICTS
in the Capability map. Select only relevant frontend, backend, API, data,
workflow/operations, and documentation modules. Register both sidecars as
related docs and return `pass` only after the files are complete.
