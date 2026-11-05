{{define "_code_review_process_body"}}Run the host-installed `shark-rider/skills/deep-review/SKILL.md` multi-agent procedure for this code-review gate, including its capability ladder and consolidation. The parent retains the entity lease and transitions; reviewers supply evidence only.

{{include: skills/quality/workflows/review-code.md}}

READ:
(1) Entity specification at {{.file_path}} for acceptance criteria
(2) Applicable feature spec or accepted repair plan for architecture decisions and requirements
(3) Applicable test plan or coverage index for expected assertions
(4) Freeze the complete review scope against the accepted baseline, including all unaccepted predecessor changes, staged and unstaged changes and relevant untracked files. Record base, candidate identity, complete file manifest and diff; never assume the last commit is the whole scope.
(5) CLAUDE.md for coding standards

REVIEW:
- [ ] Changes match entity scope (no unrelated modifications)
- [ ] Code follows CLAUDE.md patterns (service layer, error handling, etc.)
- [ ] Architecture aligns with accepted decisions
- [ ] No hardcoded values that should be configurable
- [ ] Error handling follows project conventions
- [ ] No security vulnerabilities (injection, XSS, etc.)
- [ ] Acceptance criteria from the entity specification verified against implementation
- [ ] TDD compliance — tests from the applicable test plan or coverage index are implemented and passing

PROMPT-ONLY SCOPE:
- For embedded prompt, skill, template, or documentation-only changes, verify
  rendering, include resolution, golden updates, and newly documented file
  references.
- Review policy wording against the specification. Do not require wording
  mutation tests, decision tables, or caller-path contracts unless the change
  modifies deterministic runtime behavior.

{{template "_review_output_policy" .}}

PRODUCE code review report at `{{.review_base}}code-review-<timestamp>-{{.id}}.md`.
Include the actual deep-review report, retained scope and retained diff paths so
later gates can follow the evidence; keep earlier reports.
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
