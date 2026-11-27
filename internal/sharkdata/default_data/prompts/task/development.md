{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Develop task {{.id}}: "{{.title}}".

Existing implementations must satisfy all context, rework, caller coverage and exit-gate requirements below before advancement.

UNCLEAR REQUIREMENTS: do not guess and do not wait for a reply. Return `blocked` with the specific question in your final response.

REWORK? If the task spec contains a rejection section (e.g. "## UAT Rejection") or the kickback reason names a defect class: BEFORE fixing the cited instance, apply the defect-class sweep procedure (`skills/quality/workflows/defect-class-sweep.md`) — enumerate every site in the touched module(s) matching that defect class per its "Enumeration procedure" section, disposition each per that same section (do not force-fix an existing accepted-risk disposition absent new evidence), and list the swept sites in your completion note. A point fix that leaves sibling instances of the same class costs another full review round.
If the bundle is not extracted, read the procedure with `shark skill get quality workflows/defect-class-sweep.md`.

REOPENED OR REWORK DISPATCH: before you change any code, read the latest transition note and the gate report that note cites.

---

TDD IMPLEMENTATION

PRECEDENCE: for existing or resumed code, this step's instructions override the included skills: preserve the code and history, and prove it with tests that fail without it instead of deleting it.

READ (in this order):
(1) Task spec at {{.file_path}} for goal, scope, and file list
(2) Feature spec.md for architecture and requirements (referenced in task spec)
(3) Feature test-plan.md for test cases (referenced in task spec){{if .feature_dir}}
    (feature documents live in `{{.feature_dir}}/`: `feature.md` or `spec.md`, `tasks/`, `test-plan.md` or `test_plans/`){{end}}
(4) Local standards, each one that exists:
    - `CLAUDE.md` and `AGENTS.md`
    - `docs/architecture/coding-standards.md` and the standards sub-files it links, including the Quality Limits and Testing Standards sections
    - `docs/architecture/patterns-catalog.md`
    - the Quality Gate section of `docs/architecture/tech-stack.md`
(5) Existing code in files listed in task scope

RESOLVE THE QUALITY GATE (before you implement):
Determine the commands from the project's agent instructions (`CLAUDE.md`, `AGENTS.md`), `docs/architecture/tech-stack.md` (the **Quality Gate** section), `docs/architecture/coding-standards.md`, `docs/architecture/test-gate.md`, or the repository's documented build/task targets. Check the sources in that order. If every source is absent or contains only a placeholder (for example "To be defined" or an unfilled template field), return `blocked`, recommend project bootstrap, and do not implement. Missing quality limits never block.

IMPLEMENT:

Step 1 — VERIFY AND IMPLEMENT:
{{template "_tdd_process" .}}
Follow patterns from existing code (identified in feature spec.md architecture section) and the local standards.
Resolve each limit as the Craft Rubric below directs, and write within the resolved limits on the first pass. When the project declares a measuring command, run it on the changed files before exit.

Step 2 — QUALITY GATE (MANDATORY):
Run the resolved quality gate commands before advancing. Fix ALL failures before advancing. No exceptions.
A failure that predates your change and sits outside the files you changed: fix it when the fix is straightforward and within scope; otherwise route it as a bug through the triage skill. Either way, while the gate fails it is unmet: report the task as "not ready".

Step 3 — SELF-REVIEW (before exit):
Run the self-review checklist in the Craft Rubric below against your own diff. Fix every finding, or record a cited reason why the item is not applicable. Re-run the quality gate after any fix.

EXIT GATE:
- All test cases from feature test-plan.md (for this task) pass
- Each test names its TC-ID and calls its Caller-Path Contract entrypoint from test-plan.md, mocking no higher than the declared seam
- Quality gate passes (commands resolved as in RESOLVE THE QUALITY GATE)
- Implementation follows patterns from feature spec.md
- No unrelated changes included
- Evidence walk: the summary names the test or check for each AC ID
- Every new or changed function and file is within the resolved limits, with each limit's source label stated
- The diff stays within the task's `## Scope` "Files to modify" list, or the summary justifies each extra file
- Self-review is recorded with no unresolved finding

## Reference Methodology

{{include: skills/quality/context/craft-rubric.md}}

{{include: skills/implementation/SKILL.md}}

{{include: skills/test-driven-development/SKILL.md}}

When done: stop and summarize what changed, what tests ran, and whether the task is ready for the parent loop to advance to completed. Do NOT run Shark status commands yourself — code review, QA, and UAT still run at feature level once all tasks are done.
Begin the summary with the line `ACs verified: <ids>; red-first: <n>/<m>; commit: <sha>`. Write `none` when there are no AC IDs, `0/0` when no TC changes behavior, and `none` when there is no commit.

{{template "_product_critical_path_guard" .}}
