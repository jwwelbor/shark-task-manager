Verification passed for feature {{.id}} ("{{.title}}"). Launch UAT red-team review.

This is the final quality gate. You are a RED-TEAM reviewer. Your job is to find problems, not rubber-stamp.

READ:
(1) Feature spec at {{.file_path}} for all acceptance criteria and architectural intent
(2) Feature test-plan.md for expected behavior and edge cases
(3) Code review report: {{.review_base}}code-review-*-{{.id}}.md
(4) QA report: {{.review_base}}qa-*-{{.id}}.md
(5) All task specs: `{{template "list_json" .}}` → read each task's file_path
(6) Full implementation: `git diff $(git merge-base HEAD main)..HEAD` — read the actual changed files

RED-TEAM REVIEW:
- Independently verify EVERY feature acceptance criterion against the actual code (not just reports)
- ENUMERATE, don't iterate: for each AC, list ALL violations within each defect class in this pass — finding one issue per round produces a rejection spiral
- Look for gaps between what the spec required and what was implemented
- Check for cross-task integration issues that per-task review would miss
- Verify error handling under adversarial and edge-case inputs
- Check for security issues: injection, auth bypass, data leaks, improper access control
- Verify the feature integrates correctly with the broader system
- Challenge assumptions in the code review and QA reports — they may have missed things

RE-VERIFICATION ROUND (a prior UAT report matching {{.review_base}}uat-*-{{.id}}.md exists)? Then the review is NEVER limited to confirming prior findings are fixed. Always do all three: (a) verify the named fixes, (b) re-audit the touched functions/modules for every remaining instance of each prior finding's defect class, (c) full red-team pass (all checks above) over the feature surface. Narrow asks get narrow answers.

{{template "_review_output_policy" .}}

PRODUCE UAT report to {{.review_base}}uat-<timestamp>-{{.id}}.md:
- If zero findings: compact APPROVED artifact only
  - Verdict: APPROVED
  - Scope reviewed: spec, test-plan, prior reports, tasks, diff, and red-team surface
  - AC count reviewed and any re-verification scope covered
  - Duration if known
  - `0 defects found`
  - End the report with a final delimited line: `VERDICT: APPROVED`
- If any finding, rejection, failed verification step, or non-blocking observation exists: full detailed report
  - Verdict: APPROVED or REJECTED — any CRITICAL or HIGH severity finding means REJECTED (never "approved with concerns")
  - Independent findings (do not just echo prior reports)
  - Evidence for each AC: cite specific file and line number
  - Red-team findings (if any): severity, description, recommendation, and a one-line defect-class statement (the general class, not the point instance)
  - Concrete fix guidance for each finding
  - End the report with a final delimited line: `VERDICT: APPROVED` or `VERDICT: REJECTED`

REVIEW-FINDING LOG (structured, queryable — only when findings exist, on APPROVED or REJECTED):
- One note per finding: {{template "create_note" .}} "<one-line finding summary>" --type=review-finding --created-by="codex" --metadata='{"gate":"uat","round":<N>,"severity":"<critical|high|medium|low>","defect_class":"<one-line class statement>","fingerprint":"<file>:<symbol>:<class-slug>","tc_id":"<TC-ID or omit>","disposition":"open"}'
- round = count of prior UAT reports matching {{.review_base}}uat-*-{{.id}}.md + 1. The fingerprint lets the same finding resurfacing across rounds group mechanically — a recurring fingerprint is the defect-class-protocol failure signal.
- Zero-finding APPROVED writes no `review-finding` notes.

ON APPROVED:
- Include `PARENT NOTE: Feature UAT approved — red-team passed` in your final response
- End with `RECOMMENDED OUTCOME: pass`

ON REJECTED:
- In your final response, list the exact task kickbacks the parent loop should apply, using the reason format:
  `<task-id> -> development --reason "<defect-class statement> — <specific findings>. Before fixing the cited instance, sweep the touched module(s) for every other instance of this defect class; fix all; list swept sites in the completion note."`
- Include `PARENT NOTE: UAT rejected — see report, tasks kicked back`
- End with `RECOMMENDED OUTCOME: fail`
- Do NOT run Shark status commands yourself; the parent loop will reopen tasks and reset the feature.
