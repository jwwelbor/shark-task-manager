{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Fix bug {{.id}}: "{{.title}}".

Read the research report, accepted repair plan and approval, authoritative decisions, prior findings, affected production callers and coverage index before evaluating any existing fix. Existing code must satisfy the same obligations and exit gate as new work. Before advancing an in-flight bug, inspect existing evidence and return to the earliest step with missing or invalid evidence; preserve code and history. Verify the approved plan content hash matches the current plan. Maintain current evidence in the separately linked evidence record without editing the approved plan. The plan contains the single coverage index: obligation, authoritative source, production caller or boundary, planned change, required assertions, and links to a separate mutable evidence record. {{template "_bug_repair_contract" .}}
Apply `skills/quality/workflows/defect-class-sweep.md` (if the bundle is not extracted, read it with `shark skill get quality workflows/defect-class-sweep.md`) to rejection defect classes, preserving valid dispositions and existing retry policy.

---

BUG FIX WITH REGRESSION TEST

{{include: skills/debugging/SKILL.md}}

{{include: skills/test-driven-development/SKILL.md}}

READ:
(1) Bug description at {{.file_path}}
(2) CLAUDE.md for coding standards and test patterns
(3) Related code areas (grep for affected components)

IMPLEMENT:

Step 1 — REPRODUCE:
- Write a failing test that demonstrates the bug
- Confirm test fails: `go test -v ./path/to/package -run TestName`

Step 2 — FIX:
- Implement minimum fix to make the test pass
- Follow existing patterns in the codebase

Step 3 — QUALITY GATE (MANDATORY):
Run the project's quality gate before advancing. Determine the commands from `docs/architecture/tech-stack.md` (the **Quality Gate** section), or `docs/architecture/coding-standards.md` if present. If neither exists, infer from the repo: a `Makefile` → its documented format/lint/test targets; `go.mod` → `gofmt`/`go vet ./...`/`go test ./...`; `package.json` → its format/lint/test scripts; `pyproject.toml` → the configured formatter/linter and `pytest`. Fix ALL failures before advancing. No exceptions.

EXIT GATE:
- Regression test proves the bug is fixed
- Quality gate passes
- No unrelated changes

DECISION: pass only when the accepted plan and coverage obligations are verified. Use review_plan for a valid plan whose approval is missing or stale; use needs_research for changed diagnosis, replan for a material approach change, fail for implementation rework, and blocked for unavailable external evidence. For route_rework outcomes leave kickbacks empty and retain findings in the summary. The parent owns transitions.

The fail outcome's role is `route_rework`.

{{template "_gate_result_directive" .}}
