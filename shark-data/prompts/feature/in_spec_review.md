{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

RESUME spec review for feature {{.id}}: "{{.title}}".

Check for existing spec review report at docs/review/{epic-folder}/{feature-folder}/spec_review/{{.id}}-spec-review.md.
(Derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders.)
If report exists with PASS verdict, advance immediately. If FAIL, send back to ready_for_specification.

Otherwise, continue the spec review following the ready_for_spec_review instructions.
