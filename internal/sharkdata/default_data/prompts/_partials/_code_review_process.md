{{define "_code_review_process_body"}}{{include: skills/quality/workflows/review-code.md}}

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

PROMPT-ONLY SCOPE:
- For embedded prompt, skill, template, or documentation-only changes, verify
  rendering, include resolution, golden updates, and newly documented file
  references.
- Review policy wording against the specification. Do not require wording
  mutation tests, decision tables, or caller-path contracts unless the change
  modifies deterministic runtime behavior.

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
{{end}}
{{define "_code_review_process"}}{{template "_code_review_process_body" .}}
DECISION:
- ALL PASS → end with `RECOMMENDED OUTCOME: pass`
- ANY FAIL → end with `RECOMMENDED OUTCOME: fail` and include the specific findings to fix in your final summary
  (Check report at {{.review_base}} on resume)
- Do NOT run Shark status commands yourself; the parent loop will apply the outcome.
{{end}}
{{define "_code_review_process_gate_result"}}{{template "_code_review_process_body" .}}
DECISION:
- ALL PASS -> recommended_outcome: pass
- ANY FAIL -> recommended_outcome: fail. This outcome's role is `route_rework`
  — `gate_result.kickbacks` must stay empty; state the specific findings to
  fix in `gate_result.summary`. (Check report at {{.review_base}} on resume)
- Do NOT run Shark status commands yourself; the parent loop will apply the outcome.

{{template "_gate_result_directive" .}}{{end}}
