{{template "uat_spawn_hint" .}}

Launch the UAT red-team reviewer for task {{.id}}: "{{.title}}".

READ before spawning:
(1) Task spec at {{.file_path}}
(2) Implementation code and test results
(3) Code review report from docs/review/{epic-folder}/{feature-folder}/code_review/*-{{.id}}-review.md
(4) QA report from docs/review/{epic-folder}/{feature-folder}/qa/*-{{.id}}-qa.md
(Derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders, strip /tasks/filename.)

The UAT agent will:
- Independently verify each acceptance criterion against actual code
- Produce a report at docs/review/{epic-folder}/{feature-folder}/uat/<timestamp>-{{.id}}-uat.md with APPROVED or REJECTED verdict
- On APPROVED → {{template "advance" .}}
- On REJECTED → route back to ready_for_development with rejection findings
