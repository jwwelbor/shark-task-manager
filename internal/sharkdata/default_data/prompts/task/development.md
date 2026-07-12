{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Develop task {{.id}}: "{{.title}}".

Check for existing implementation: review git diff and test files. If implementation exists and passes quality gate, advance immediately.

REWORK? If the task spec contains a rejection section (e.g. "## UAT Rejection") or the kickback reason names a defect class: BEFORE fixing the cited instance, enumerate every code site in the touched module(s) matching that defect class, fix ALL of them, and list the swept sites in your completion note. A point fix that leaves sibling instances of the same class costs another full review round.

---

TDD IMPLEMENTATION

{{include: skills/implementation/SKILL.md}}

{{include: skills/test-driven-development/SKILL.md}}

READ (in this order):
(1) Task spec at {{.file_path}} for goal, scope, and file list
(2) Feature spec.md for architecture and requirements (referenced in task spec)
(3) Feature test-plan.md for test cases (referenced in task spec)
(4) CLAUDE.md for coding standards and patterns
(5) Existing code in files listed in task scope

IMPLEMENT using TDD:

Step 1 — WRITE FAILING TESTS FIRST:
- Write test cases from feature test-plan.md that apply to this task
- Follow project test patterns: repo tests use real DB, service tests use mocks, CLI tests use mocks
- Run tests to confirm they fail: `go test -v ./path/to/package -run TestName`

Step 2 — IMPLEMENT MINIMUM CODE:
- Write the minimum code to make tests pass
- Follow patterns from existing code (identified in feature spec.md architecture section)
- Do NOT add features beyond what the task spec requires

Step 3 — REFACTOR:
- Clean up while keeping tests green
- Ensure code follows CLAUDE.md patterns

Step 4 — QUALITY GATE (MANDATORY):
Run the project's quality gate before advancing. Determine the commands from `docs/architecture/tech-stack.md` (the **Quality Gate** section), or `docs/architecture/coding-standards.md` if present. If neither exists, infer from the repo: a `Makefile` → its documented format/lint/test targets; `go.mod` → `gofmt`/`go vet ./...`/`go test ./...`; `package.json` → its format/lint/test scripts; `pyproject.toml` → the configured formatter/linter and `pytest`. Fix ALL failures before advancing. No exceptions.

EXIT GATE:
- All test cases from feature test-plan.md (for this task) pass
- Each test names its TC-ID and calls its Caller-Path Contract entrypoint from test-plan.md, mocking no higher than the declared seam
- Quality gate passes (commands from `docs/architecture/tech-stack.md` or inferred from the repo)
- Implementation follows patterns from feature spec.md
- No unrelated changes included

When done: stop and summarize what changed, what tests ran, and whether the task is ready for the parent loop to advance to completed. Do NOT run Shark status commands yourself — code review, QA, and UAT still run at feature level once all tasks are done.
