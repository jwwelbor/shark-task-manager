{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

RESUME test plan review for feature {{.id}}: "{{.title}}".

Check for existing test plan review report at docs/review/{epic-folder}/{feature-folder}/test_plan_review/{{.id}}-test-plan-review.md.
(Derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders.)
If report exists with PASS verdict, advance immediately. If FAIL, send back to ready_for_test_planning.

Otherwise, continue the test plan review following the ready_for_test_plan_review instructions.
