RESUME CONTEXT: Code review is in progress for task {{.id}}: "{{.title}}".

{{if eq .entity_type "Bug"}}Check for existing code review report at docs/review/bugs/{{.id}}/code_review/*-{{.id}}-review.md{{else}}Check for existing code review report at docs/review/changes/{{.id}}/code_review/*-{{.id}}-review.md{{end}}
- If report exists with PASS verdict AND no outstanding issues → {{template "advance" .}}
- If report exists with FAIL verdict → shark status set {{.id}} ready_for_development --reason "<findings from report>"
- If NO report exists → you MUST perform the full code review below. Do NOT advance without a written report.

---

CODE REVIEW

{{template "_code_review_process" .}}