{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Resolve tech debt {{.id}}: "{{.title}}".

Category: {{.category}} | Severity: {{.severity}}

Check for existing resolution: review recent git changes. If resolution exists and the project quality gate passes (see `docs/architecture/tech-stack.md` **Quality Gate** section, or infer from the repo), advance immediately.

Otherwise, continue with resolution per the plan below.

---

TECH DEBT RESOLUTION

READ:
(1) Tech debt description at {{.file_path}}
(2) CLAUDE.md for coding standards and patterns
(3) Related code areas (grep for affected components)

IMPLEMENT:

Step 1 — ANALYZE:
- Understand the scope and impact of the tech debt
- Identify affected files and components

Step 2 — RESOLVE:
- Implement the resolution following existing patterns
- Ensure no regressions are introduced

Step 3 — QUALITY GATE (MANDATORY):
Run the project's quality gate before advancing. Determine the commands from `docs/architecture/tech-stack.md` (the **Quality Gate** section), or `docs/architecture/coding-standards.md` if present. If neither exists, infer from the repo: a `Makefile` → its documented format/lint/test targets; `go.mod` → `gofmt`/`go vet ./...`/`go test ./...`; `package.json` → its format/lint/test scripts; `pyproject.toml` → the configured formatter/linter and `pytest`. Fix ALL failures before advancing. No exceptions.

EXIT GATE:
- Tech debt is resolved
- Quality gate passes
- No unrelated changes

DECISION:
- Exit gate met -> recommended_outcome: pass
- Cannot resolve (external blocker) -> recommended_outcome: blocked; state
  the blocker in `gate_result.no_kickback_reason` (this is a single-entity
  item — there is nothing else to kick back)
- Determined not worth fixing -> recommended_outcome: wont_fix; state why in
  `gate_result.no_kickback_reason`
- Otherwise unresolved -> recommended_outcome: fail. This outcome's role is
  `route_rework` — `gate_result.kickbacks` must stay empty; state the
  specific blockers in `gate_result.summary`.

{{template "_gate_result_directive" .}}
