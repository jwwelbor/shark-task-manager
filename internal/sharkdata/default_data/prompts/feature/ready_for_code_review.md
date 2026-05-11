All tasks for feature {{.id}} ("{{.title}}") have completed development.

Launch tech-lead with quality skill to perform a single feature-level code review.

Load skill: `shark-data/skills/quality/workflows/review-code.md`

READ:
(1) Feature spec at {{.file_path}} for architecture decisions and acceptance criteria
(2) Feature test-plan.md for all expected test cases and coverage requirements
(3) CLAUDE.md for codebase standards and patterns
(4) All task specs: `{{template "list_json" .}}` → read each task's file_path in sequence
(5) Full feature diff: `git log --oneline -20` then `git diff $(git merge-base HEAD main)..HEAD`
(6) Implementation code for all changed files

REVIEW (treat this as a single PR review, not per-task):
- Code quality, security, and adherence to CLAUDE.md across the full changeset
- Architecture aligns with feature spec decisions (naming, layers, patterns)
- All feature acceptance criteria are met in aggregate — cite the AC and the code that satisfies it
- TDD compliance — tests from the feature test-plan are implemented and passing
- Cross-task integration — do the task implementations fit together correctly?
- No regressions or unintended side effects in the combined changeset
- Flag any drift between feature intent (spec/test-plan) and actual implementation

ON PASS (all ACs met, no blockers across full feature):
- Write code review report to docs/review/{epic-folder}/{feature-folder}/code_review/<timestamp>-{{.id}}-review.md
  (derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders)
  - Verdict: PASS
  - AC verification: list each AC with the code/test that satisfies it
  - Notes on any non-blocking observations
- Add note: {{template "create_note" .}} --type=review "Feature code review approved — see report"
- Advance: {{template "advance" .}}

ON FAIL (blockers, spec drift, or missing ACs):
- Write code review report with findings grouped by task
  - Verdict: FAIL
  - Per-task findings: list task ID, specific issues, required changes
- For each failing task, kick back: `shark status set <task-id> ready_for_development --reason "<specific findings>"`
- Set feature back to active: `{{template "status_set" .}} active --reason "Code review failed — see report, tasks kicked back"`
  (report lives at docs/review/{epic-folder}/{feature-folder}/code_review/)
- Do NOT advance the feature.
