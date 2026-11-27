{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Implement change {{.id}}: "{{.title}}".

REOPENED OR REWORK DISPATCH: before you change any code, read the latest transition note and the gate report that note cites.

RESOLVE THE QUALITY GATE (before you implement):
Determine the commands from the project's agent instructions (`CLAUDE.md`, `AGENTS.md`), `docs/architecture/tech-stack.md` (the **Quality Gate** section), `docs/architecture/coding-standards.md`, `docs/architecture/test-gate.md`, or the repository's documented build/task targets. Check the sources in that order. If every source is absent or contains only a placeholder (for example "To be defined" or an unfilled template field), recommend `blocked`, put a recommendation to run project bootstrap in `gate_result.no_kickback_reason`, and do not implement. Missing quality limits never block.

Check for existing implementation. If changes exist, read the sources in READ, then advance immediately only when the resolved quality gate passes, the self-review in the Craft Rubric below is recorded with no unresolved finding against that existing diff, and every EXIT GATE item is met. Otherwise, continue with the implementation below.

---

PRECEDENCE: for existing or resumed code, preserve the code and history, and prove it with tests that fail without it instead of deleting it.

READ:
(1) Change card description at {{.file_path}}
(2) Local standards, each one that exists:
    - `CLAUDE.md` and `AGENTS.md`
    - `docs/architecture/coding-standards.md` and the standards sub-files it links, including the Quality Limits and Testing Standards sections
    - `docs/architecture/patterns-catalog.md`
    - the Quality Gate section of `docs/architecture/tech-stack.md`
(3) Existing code in the affected areas

IMPLEMENT the change:

Step 1 — VERIFY AND IMPLEMENT:
{{template "_tdd_process" .}}
Follow existing patterns in the codebase and the local standards.
Resolve each limit as the Craft Rubric below directs, and write within the resolved limits on the first pass. When the project declares a measuring command, run it on the changed files before exit.

Step 2 — QUALITY GATE (MANDATORY):
Run the resolved quality gate commands before advancing. Fix ALL failures before advancing. No exceptions.
A failure that predates your change and sits outside the files you changed: fix it when the fix is straightforward and within scope; otherwise route it as a bug through the triage skill. Either way, while the gate fails it is unmet: recommend `fail`.

Step 3 — SELF-REVIEW (before exit):
Run the self-review checklist in the Craft Rubric below against your own diff. Fix every finding, or record a cited reason why the item is not applicable. Re-run the quality gate after any fix.

EXIT GATE:
- Change implemented per description
- Tests pass
- Quality gate passes
- Evidence walk: `gate_result.summary` names the test or check for each AC ID
- Every new or changed function and file is within the resolved limits, with each limit's source label stated
- Self-review is recorded with no unresolved finding

## Reference Methodology

{{include: skills/quality/context/craft-rubric.md}}

SUMMARY LINE: Begin `gate_result.summary` with the line `ACs verified: <ids>; red-first: <n>/<m>; commit: <sha>`. Write `none` when there are no AC IDs, `0/0` when no TC changes behavior, and `none` when there is no commit.

DECISION:
- Unclear requirements -> recommended_outcome: blocked; put the specific
  question in `gate_result.no_kickback_reason` rather than guessing
- Exit gate met -> recommended_outcome: pass
- Cannot implement (external blocker) -> recommended_outcome: blocked; state
  the blocker in `gate_result.no_kickback_reason`
- Otherwise unresolved -> recommended_outcome: fail. This outcome's role is
  `route_rework` — `gate_result.kickbacks` must stay empty; state the
  specific blockers in `gate_result.summary`.

{{template "_gate_result_directive" .}}
