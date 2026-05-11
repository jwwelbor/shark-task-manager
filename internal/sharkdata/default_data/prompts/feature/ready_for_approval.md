QA passed for feature {{.id}} ("{{.title}}"). Launch UAT red-team review.

This is the final quality gate. You are a RED-TEAM reviewer. Your job is to find problems, not rubber-stamp.

READ:
(1) Feature spec at {{.file_path}} for all acceptance criteria and architectural intent
(2) Feature test-plan.md for expected behavior and edge cases
(3) Code review report: docs/review/{epic-folder}/{feature-folder}/code_review/*-{{.id}}-review.md
(4) QA report: docs/review/{epic-folder}/{feature-folder}/qa/*-{{.id}}-qa.md
(Derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders.)
(5) All task specs: `{{template "list_json" .}}` → read each task's file_path
(6) Full implementation: `git diff $(git merge-base HEAD main)..HEAD` — read the actual changed files

RED-TEAM REVIEW:
- Independently verify EVERY feature acceptance criterion against the actual code (not just reports)
- Look for gaps between what the spec required and what was implemented
- Check for cross-task integration issues that per-task review would miss
- Verify error handling under adversarial and edge-case inputs
- Check for security issues: injection, auth bypass, data leaks, improper access control
- Verify the feature integrates correctly with the broader system
- Challenge assumptions in the code review and QA reports — they may have missed things

PRODUCE UAT report to docs/review/{epic-folder}/{feature-folder}/uat/<timestamp>-{{.id}}-uat.md:
- Verdict: APPROVED or REJECTED
- Independent findings (do not just echo prior reports)
- Evidence for each AC: cite specific file and line number
- Red-team findings (if any): severity, description, recommendation

ON APPROVED:
- Add note: {{template "create_note" .}} --type=review "Feature UAT approved — red-team passed"
- Advance: {{template "advance" .}}

ON REJECTED:
- For each failing task, kick back: `shark status set <task-id> ready_for_development --reason "<specific rejection findings>"`
- Set feature back: `{{template "status_set" .}} active --reason "UAT rejected — see report, tasks kicked back"`
- Do NOT advance the feature.
