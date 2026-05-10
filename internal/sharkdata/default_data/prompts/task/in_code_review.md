{{template "_resume_preamble" .}}RESUME code review for task {{.task_id}}: "{{.title}}".

Check for existing code review notes or comments. If review is complete with approval and no outstanding issues, advance immediately. If review found issues that have since been addressed, re-validate and advance.

---

Launch tech-lead with quality skill to review task {{.task_id}}: "{{.title}}".

Load skill: `shark-data/skills/quality/workflows/review-code.md`

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for expected test coverage and acceptance criteria
(3) Feature PRD for feature-level intent
(4) Implementation code changes
{{- if .related_docs}}
(5) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}6{{else}}5{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

REVIEW:
- Check code quality, security, and adherence to codebase standards
- Compare implementation behavior against the task spec, feature test plan, and feature PRD
- Verify acceptance criteria are met as specified in feature requirements
- Verify TDD compliance -- tests from feature test plan are implemented and passing
- Flag any deviation between feature intent, spec, and actual behavior

ON PASS (all acceptance criteria met, no blockers):
- Write code review report to docs/review/{epic}/{feature}/code_review/<timestamp>-{{.task_id}}-review.md
  (derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders, strip /tasks/filename)
- Add note: {{template "create_note_task" .}} --type=review "Code review approved - see report"
- Advance: {{template "advance" .}}

ON FAIL (blockers found, spec drift, or missing acceptance criteria):
- Write code review report with detailed findings and required changes
- Add note: {{template "create_note_task" .}} --type=blocker "Code review failed - see report for required changes"
- Send back to development: {{template "status_set_task" .}} ready_for_development
- Do NOT use changes_requested, in_development, or any invented status. The valid rejection status from code review is ready_for_development.
