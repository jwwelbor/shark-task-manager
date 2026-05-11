{{define "_uat_redteam_review"}}This is the final quality gate. You are a RED-TEAM reviewer. Your job is to find problems, not rubber-stamp.

READ:
(1) Task spec at {{.file_path}} for acceptance criteria
(2) Code review report from {{.review_base}}code_review/
(3) QA report from {{.review_base}}qa/
(4) Feature spec.md and test-plan.md for feature-level requirements
(5) Implementation code — read the actual changed files

RED-TEAM REVIEW:
- Independently verify each acceptance criterion against the actual code (not just reports)
- Look for gaps between what was specified and what was implemented
- Check for edge cases the code review and QA may have missed
- Verify error handling under adversarial inputs
- Check for security issues (injection, auth bypass, data leaks)
- Verify the implementation integrates correctly with the broader feature

PRODUCE UAT report at {{.review_base}}uat/`<timestamp>`-{{.id}}-uat.md:
- Verdict: APPROVED or REJECTED
- Independent findings (not just echoing prior reports)
- Evidence for each acceptance criterion (cite specific code lines)
- Red-team findings (if any)

DECISION:
- APPROVED → {{template "advance" .}}
- REJECTED → shark status set {{.id}} ready_for_development --reason "<specific rejection findings>"
{{end}}
