{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research feature {{.id}}: "{{.title}}" using the universal recipe in
`shark-data/research/recipes.yaml`. Read the parent report, parent PRD,
siblings, {{.file_path}}, and only the category-specific sources selected by
the plan.

First write `research-plan.md` beside the feature file. Its front matter must
identify `entity_key`, `entity_type: feature`, `recipe: universal`, selected
`rigor`, `categories`, `source_set`, and `related_work`; add Scope, Recipe,
Source set, and Steps sections. Always define ubiquitous vocabulary. Where
siblings or related work exist, inspect them and record a non-empty Capability
map that decides REUSE, EXTEND, NEW, or CONTRADICTS. Reference upstream and
sibling reports instead of copying them.

Write `research-report.md` with matching front matter and Scope, Capability
map, Ubiquitous vocabulary, Findings, Decisions, and Sources sections. Register
both files as related docs on {{.id}}.

Route from the selected rigor: SIMPLE ends `RECOMMENDED OUTCOME: simple`;
STANDARD ends `RECOMMENDED OUTCOME: standard`; COMPLEX ends `RECOMMENDED
OUTCOME: pass`. Do not skip research for SIMPLE work.
