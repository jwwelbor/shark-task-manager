{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research tech debt {{.id}}: "{{.title}}" using the universal recipe catalog.
Write adjacent `{{.id}}.research-plan.md` and `{{.id}}.research-report.md`
sidecars beside {{.file_path}}. Both front-matter blocks must identify
`entity_key`, `entity_type: tech_debt`, `recipe: universal`, `rigor`,
`categories`, `source_set`, and `related_work`. The plan requires Scope,
Recipe, Source set, and Steps. The report requires Scope, Capability map,
Ubiquitous vocabulary, Findings, Decisions, and Sources.

Always establish vocabulary, inspect existing debt-remediation and related
capabilities, and record REUSE, EXTEND, NEW, or CONTRADICTS when related work
exists. Select only relevant category modules, register both sidecars as
related docs, then return `pass`.
