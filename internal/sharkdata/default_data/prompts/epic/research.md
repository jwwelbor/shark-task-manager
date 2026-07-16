{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research epic {{.id}}: "{{.title}}" using the universal recipe catalog at
`shark-data/research/recipes.yaml`.

First write `research-plan.md` beside {{.file_path}}. Select `recipe:
universal`, a rigor tier (`simple`, `standard`, or `complex`), and only the
applicable categories: `frontend`, `backend`, `api`, `data`,
`workflow_operations`, `documentation`. The plan must have front matter with
`entity_key`, `entity_type: epic`, `recipe`, `rigor`, `categories`,
`source_set`, and `related_work`, then these sections: Scope, Recipe, Source
set, Steps.

Execute only the selected recipe steps. Always establish ubiquitous vocabulary.
When sibling epics, existing features, related work, or relevant capabilities
exist, inspect them and include a non-empty Capability map with REUSE, EXTEND,
NEW, or CONTRADICTS decisions. Cite paths and link to upstream material instead
of reproducing it.

Then write `research-report.md` with matching front matter and these sections:
Scope, Capability map, Ubiquitous vocabulary, Findings, Decisions, Sources.
Register both documents as related docs on {{.id}}. Return `pass` only after
both documents meet this structural contract.
