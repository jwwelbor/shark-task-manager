{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Fix bug {{.id}}: "{{.title}}".

Read the research report, accepted repair plan and approval, authoritative decisions, prior findings, affected production callers and coverage index before evaluating any existing fix. Existing code must satisfy the same obligations and exit gate as new work. Before advancing an in-flight bug, inspect existing evidence and return to the earliest step with missing or invalid evidence; preserve code and history. Verify the parent-supplied approved snapshot and digest; a clean older checkout may retain an earlier plan, while a new material plan change requires review. Maintain current evidence in the separately linked evidence record without editing the approved plan. The plan contains the single coverage index: obligation, authoritative source, production caller or boundary, planned change, required assertions, and links to a separate mutable evidence record. {{template "_bug_repair_contract" .}}

{{template "_bug_decision_sources" .}}

Check each applicable decision constraint against the approved coverage index before evaluating new, existing, and resumed fixes. Record the assertion and observed result for each applicable constraint in the linked evidence record. Return `replan` when an omitted or changed constraint requires a material plan update, even if the regression tests and project quality gate pass.

Apply `skills/quality/workflows/defect-class-sweep.md` (if the bundle is not extracted, read it with `shark skill get quality workflows/defect-class-sweep.md`) to rejection defect classes, preserving valid dispositions and existing retry policy.

---

UNCLEAR REQUIREMENTS: do not guess and do not wait for a reply. Return `blocked` with the specific question in `gate_result.no_kickback_reason`.

BUG FIX WITH REGRESSION TEST

PRECEDENCE: for existing or resumed code, this step's instructions override the included skills: preserve the code and history, and prove it with tests that fail without it instead of deleting it.

READ:
(1) Bug description at {{.file_path}}
(2) CLAUDE.md for coding standards and test patterns
(3) Related code areas (grep for affected components)

IMPLEMENT:

Step 1 — REPRODUCE:
- Write a failing test that demonstrates the bug
- Confirm test fails by running the focused test with the project's test command (see QUALITY GATE for how to determine it)

Step 2 — FIX:
- Implement minimum fix to make the test pass
- Follow existing patterns in the codebase

Step 3 — QUALITY GATE (MANDATORY):
Run the project's quality gate before advancing. Determine the commands from the project's agent instructions (`CLAUDE.md`, `AGENTS.md`), `docs/architecture/tech-stack.md` (the **Quality Gate** section), `docs/architecture/coding-standards.md`, `docs/architecture/test-gate.md`, or the repository's documented build/task targets. Fix ALL failures before advancing. No exceptions.

EXIT GATE:
- Regression test proves the bug is fixed
- Quality gate passes
- Every applicable decision constraint has passing verification evidence or an authorized disposition
- No unrelated changes

## Reference Methodology

{{include: skills/debugging/SKILL.md}}

{{include: skills/test-driven-development/SKILL.md}}

DECISION: pass only when the accepted plan, coverage obligations, and exit gate are satisfied. Use review_plan for a valid plan whose approval is missing or stale; use needs_research for changed diagnosis, replan for a material plan update, fail for implementation rework, and blocked for unavailable external evidence. For route_rework outcomes leave kickbacks empty, keep the summary concise, and retain findings in the structured findings. The parent owns transitions.

The fail outcome's role is `route_rework`.

{{template "_gate_result_directive" .}}

REQUIRED OUTER EVIDENCE for a success outcome of this step (this replaces the empty array shown above; replace every `<...>` placeholder, and copy parent-supplied approved entries exactly):

```
  "evidence": [
    {"kind": "bug_repair_plan", "pointer": "<project-relative plan file>", "summary": "sha256:<hex digest of the plan>"},
    {"kind": "bug_review_baseline", "pointer": "<full commit hash of the plan's review baseline>"}
  ]
```
