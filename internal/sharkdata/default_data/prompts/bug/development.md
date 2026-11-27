{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Fix bug {{.id}}: "{{.title}}".

Read the research report, accepted repair plan and approval, authoritative decisions, prior findings, affected production callers and coverage index before evaluating any existing fix. Existing code must satisfy the same obligations and exit gate as new work. Before advancing an in-flight bug, inspect existing evidence and return to the earliest step with missing or invalid evidence; preserve code and history. Verify the parent-supplied approved snapshot and digest; a clean older checkout may retain an earlier plan, while a new material plan change requires review. Maintain current evidence in the separately linked evidence record without editing the approved plan. The plan contains the single coverage index: obligation, authoritative source, production caller or boundary, planned change, required assertions, and links to a separate mutable evidence record. {{template "_bug_repair_contract" .}}

{{template "_bug_decision_sources" .}}

Check each applicable decision constraint against the approved coverage index before evaluating new, existing, and resumed fixes. Record the assertion and observed result for each applicable constraint in the linked evidence record. Return `replan` when an omitted or changed constraint requires a material plan update, even if the regression tests and project quality gate pass.

Apply `skills/quality/workflows/defect-class-sweep.md` (if the bundle is not extracted, read it with `shark skill get quality workflows/defect-class-sweep.md`) to rejection defect classes, preserving valid dispositions and existing retry policy.

REOPENED OR REWORK DISPATCH: before you change any code, read the latest transition note and the gate report that note cites.

---

UNCLEAR REQUIREMENTS: do not guess and do not wait for a reply. Return `blocked` with the specific question in `gate_result.no_kickback_reason`.

BUG FIX WITH REGRESSION TEST

PRECEDENCE: for existing or resumed code, this step's instructions override the included skills: preserve the code and history, and prove behavior-changing code with tests that fail without it instead of deleting it. Refactors and prompt, documentation or configuration changes follow the verification table in the TDD process below.

READ:
(1) Bug description at {{.file_path}}
(2) Related code areas (grep for affected components)
(3) Local standards, each one that exists:
    - `CLAUDE.md` and `AGENTS.md`
    - `docs/architecture/coding-standards.md` and the standards sub-files it links, including the Quality Limits (with Limit exceptions) and Testing Standards sections
    - `docs/architecture/patterns-catalog.md`
    - the Quality Gate section of `docs/architecture/tech-stack.md`

RESOLVE THE QUALITY GATE (before you implement):
Determine the commands from the project's agent instructions (`CLAUDE.md`, `AGENTS.md`), `docs/architecture/tech-stack.md` (the **Quality Gate** section), `docs/architecture/coding-standards.md`, `docs/architecture/test-gate.md`, or the repository's documented build/task targets. Check the sources in that order. If every source is absent or contains only a placeholder (for example "To be defined" or an unfilled template field), recommend `blocked`, put a recommendation to run project bootstrap in `gate_result.no_kickback_reason`, and do not implement. Missing quality limits never block.

IMPLEMENT:

Step 1 — REPRODUCE AND VERIFY:
{{template "_tdd_process" .}}
- For behavior-changing code, the failing test demonstrates the bug

Step 2 — FIX:
- Implement minimum fix to make the test pass
- Follow existing patterns in the codebase and the local standards
Resolve each limit as the Craft Rubric below directs, and write within the resolved limits on the first pass. Its edited-code rule covers every code file you edit: each ends within every resolved limit, including overages that predate your change, and each is measured in full, not only the changed lines. When the project declares a measuring command, run it so it covers every edited code file in full; when that command measures only new code, measure the rest of each edited file by tool or by hand.
If you doubt that a refactor this rule requires is safe, consult the architect: do not leave the overage in place, and do not record a limit exception yourself. Recommend `blocked` and name the file, the unit, the limit and its source label, the measured value, the refactor you considered and the safety concern in `gate_result.no_kickback_reason`.

Step 3 — QUALITY GATE (MANDATORY):
Run the resolved quality gate commands before advancing. Fix ALL failures before advancing. No exceptions.
A failure that predates your change and sits outside the files you changed: fix it when the fix is straightforward and within scope; otherwise route it as a bug through the triage skill. Either way, while the gate fails it is unmet: recommend `fail`. A limit overage inside a file you edited is not this case: the edited-code rule governs it.

Step 4 — SELF-REVIEW (before exit):
Run the self-review checklist in the Craft Rubric below against your own diff. Fix every finding, or record a cited reason why the item is not applicable. Re-run the quality gate after any fix.

EXIT GATE:
- Regression test proves the bug is fixed
- Quality gate passes
- Every applicable decision constraint has passing verification evidence or an authorized disposition
- No unrelated changes
- Evidence walk: name the test or check for each AC ID, compactly (AC-ID: test name) so `gate_result.summary` stays within its size limit
- Every code file you edited is within the resolved limits (file length and every function in it, including overages that predate the change), measured in full, with each limit's source label stated; any unit still over a limit names the limit exception that covers it
- Self-review is recorded with no unresolved finding

## Reference Methodology

{{include: skills/quality/context/craft-rubric.md}}

{{include: skills/debugging/SKILL.md}}

{{include: skills/test-driven-development/SKILL.md}}

SUMMARY LINE: Begin `gate_result.summary` with the line `ACs verified: <ids>; red-first: <n>/<m>; commit: <sha>`. Write `none` when there are no AC IDs, `0/0` when no TC changes behavior, and `none` when there is no commit.

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
