{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research bug {{.id}}: "{{.title}}" using the universal recipe catalog.
Write adjacent `{{.id}}.research-plan.md` and `{{.id}}.research-report.md`
sidecars beside {{.file_path}}. Both front-matter blocks must identify
`entity_key`, `entity_type: bug`, `recipe: universal`, `rigor`, `categories`,
`source_set`, and `related_work`. The plan requires Scope, Recipe, Source set,
and Steps. The report requires Scope, Capability map, Ubiquitous vocabulary,
Findings, Decisions, and Sources.

Establish vocabulary, reproduce and locate the affected capability, and inspect
related fixes or patterns. When related work exists, make the Capability map
non-empty with REUSE, EXTEND, NEW, or CONTRADICTS decisions. Select only the
relevant category modules, register both sidecars as related docs, then return
`pass`.
