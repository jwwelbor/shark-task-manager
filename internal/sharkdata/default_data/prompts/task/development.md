{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Develop task {{.id}}: "{{.title}}".

Existing implementations must satisfy all context, rework, caller coverage and exit-gate requirements below before advancement.

UNCLEAR REQUIREMENTS: do not guess and do not wait for a reply. Return `blocked` with the specific question in your final response.

REWORK? If the task spec contains a rejection section (e.g. "## UAT Rejection") or the kickback reason names a defect class: BEFORE fixing the cited instance, apply the defect-class sweep procedure (`skills/quality/workflows/defect-class-sweep.md`) — enumerate every site in the touched module(s) matching that defect class per its "Enumeration procedure" section, disposition each per that same section (do not force-fix an existing accepted-risk disposition absent new evidence), and list the swept sites in your completion note. A point fix that leaves sibling instances of the same class costs another full review round.
If the bundle is not extracted, read the procedure with `shark skill get quality workflows/defect-class-sweep.md`.

---

TDD IMPLEMENTATION

PRECEDENCE: for existing or resumed code, this step's instructions override the included skills: preserve the code and history, and prove it with tests that fail without it instead of deleting it.

READ (in this order):
(1) Task spec at {{.file_path}} for goal, scope, and file list
(2) Feature spec.md for architecture and requirements (referenced in task spec)
(3) Feature test-plan.md for test cases (referenced in task spec)
(4) CLAUDE.md for coding standards and patterns{{if .feature_dir}}
    (feature documents live in `{{.feature_dir}}/`: `feature.md` or `spec.md`, `tasks/`, `test-plan.md` or `test_plans/`){{end}}
(5) Existing code in files listed in task scope

IMPLEMENT using TDD:

Step 1 — WRITE FAILING TESTS FIRST:
- Write test cases from feature test-plan.md that apply to this task
- Follow the project's documented test architecture and mocking boundaries (from CLAUDE.md, AGENTS.md, or docs/architecture/coding-standards.md)
- Run tests to confirm they fail by running the focused test with the project's test command (see QUALITY GATE for how to determine it)

Step 2 — IMPLEMENT MINIMUM CODE:
- Write the minimum code to make tests pass
- Follow patterns from existing code (identified in feature spec.md architecture section)
- Do NOT add features beyond what the task spec requires

Step 3 — REFACTOR:
- Clean up while keeping tests green
- Ensure code follows CLAUDE.md patterns

Step 4 — QUALITY GATE (MANDATORY):
Run the project's quality gate before advancing. Determine the commands from the project's agent instructions (`CLAUDE.md`, `AGENTS.md`), `docs/architecture/tech-stack.md` (the **Quality Gate** section), `docs/architecture/coding-standards.md`, `docs/architecture/test-gate.md`, or the repository's documented build/task targets. Fix ALL failures before advancing. No exceptions.

EXIT GATE:
- All test cases from feature test-plan.md (for this task) pass
- Each test names its TC-ID and calls its Caller-Path Contract entrypoint from test-plan.md, mocking no higher than the declared seam
- Quality gate passes (commands determined as in Step 4)
- Implementation follows patterns from feature spec.md
- No unrelated changes included

## Reference Methodology

{{include: skills/implementation/SKILL.md}}

{{include: skills/test-driven-development/SKILL.md}}

When done: stop and summarize what changed, what tests ran, and whether the task is ready for the parent loop to advance to completed. Do NOT run Shark status commands yourself — code review, QA, and UAT still run at feature level once all tasks are done.

{{template "_product_critical_path_guard" .}}
