RESUME CONTEXT: QA testing is in progress for task {{.id}}: "{{.title}}".

{{if eq .entity_type "Bug"}}Check for existing QA report at docs/review/bugs/{{.id}}/qa/*-{{.id}}-qa.md{{else}}Check for existing QA report at docs/review/changes/{{.id}}/qa/*-{{.id}}-qa.md{{end}}
- If QA report exists with PASS verdict → {{template "advance" .}}
- If QA report exists with FAIL verdict → shark status set {{.id}} ready_for_development --reason "<failures from report>"
- If NO QA report exists → you MUST perform the full QA validation below. Do NOT advance without a written report.

---

QA testing for task {{.id}}: "{{.title}}".

{{template "_qa_process" .}}