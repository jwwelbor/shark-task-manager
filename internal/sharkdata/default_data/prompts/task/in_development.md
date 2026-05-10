{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

RESUME development for task {{.task_id}}: "{{.title}}".

Check for existing implementation code and test files. Run existing tests to assess current state. If all tests pass and implementation matches task spec, advance immediately. If tests exist but some fail, continue from the failing tests. If partial implementation exists, continue building on it.

---

Launch developer with test-driven-development skill for task {{.task_id}}: "{{.title}}".

LOAD: test-driven-development + implementation skills.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for test cases, acceptance criteria tests, and API contract tests relevant to this task
(3) Feature architecture docs for contracts and patterns
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}5{{else}}4{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

{{template "_tdd_process" .}}

EXIT GATE:
- All mapped test cases from feature test plan pass
- Implementation matches task spec
- Code follows codebase conventions from research
