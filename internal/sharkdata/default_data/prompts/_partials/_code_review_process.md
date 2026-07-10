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

{{template "_review_output_policy" .}}

PRODUCE code review report at {{.review_base}}code-review-<timestamp>-{{.id}}.md:
- If zero findings: compact PASS artifact only
  - Verdict: PASS
  - Scope reviewed: task spec, feature context, diff, and implementation surface
  - Checks run: review checklist and tests/commands referenced, summarized
  - AC count reviewed
  - Duration if known
  - `0 defects found`
- If any blocker or non-blocking observation exists: full detailed report
  - Verdict: PASS or FAIL
  - Findings with file paths, evidence, failed commands/tests if any, affected ACs, and concrete fix guidance
  - AC verification status

DECISION:
- ALL PASS → {{template "advance" .}}
- ANY FAIL → shark status set {{.id}} development --reason "<specific findings to fix>"
  (Check report at {{.review_base}} on resume)
{{end}}
