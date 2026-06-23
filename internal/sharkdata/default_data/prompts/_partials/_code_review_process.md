{{define "_code_review_process"}}{{include: skills/quality/workflows/review-code.md}}

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

PRODUCE code review report at {{.review_base}}code_review/`<timestamp>`-{{.id}}-review.md:
- Verdict: PASS or FAIL
- Findings (if any)
- AC verification status

DECISION:
- ALL PASS → {{template "advance" .}}
- ANY FAIL → shark status set {{.id}} ready_for_development --reason "<specific findings to fix>"
  (Check report at {{.review_base}}code_review/ on resume)
{{end}}
