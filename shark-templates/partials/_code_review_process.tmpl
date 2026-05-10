{{define "_code_review_process"}}LOAD: quality skill workflow review-code.md.

READ:
(1) Task spec at {{.file_path}} for acceptance criteria
(2) Feature spec.md for architecture decisions and requirements
(3) Feature test-plan.md for expected test cases
(4) Git diff of changes: git log --oneline -5 && git diff HEAD~1
(5) CLAUDE.md for coding standards

REVIEW:
- [ ] Changes match task scope (no unrelated modifications)
- [ ] Code follows CLAUDE.md patterns (service layer, error handling, etc.)
- [ ] Architecture aligns with feature spec.md decisions
- [ ] No hardcoded values that should be configurable
- [ ] Error handling follows project conventions
- [ ] No security vulnerabilities (injection, XSS, etc.)
- [ ] Acceptance criteria from task spec verified against implementation
- [ ] TDD compliance — tests from feature test-plan.md are implemented and passing

REVIEW BASE PATH: Derive from {{.file_path}} — replace "docs/plan/" with "docs/review/", keep epic-folder/feature-folder segments (strip /tasks/filename if present).
Example: docs/plan/E19-sprint/E19-F04-analytics/tasks/T-E19-F04-001.md → docs/review/E19-sprint/E19-F04-analytics/

PRODUCE code review report at <review-base>/code_review/<timestamp>-{{.id}}-review.md:
- Verdict: PASS or FAIL
- Findings (if any)
- AC verification status

DECISION:
- ALL PASS → {{template "advance" .}}
- ANY FAIL → shark status set {{.id}} ready_for_development --reason "<specific findings to fix>"
  (Check report at <review-base>/code_review/ on resume)
{{end}}
