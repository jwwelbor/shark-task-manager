{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Implement change {{.id}}: "{{.title}}".

Check for existing implementation. If changes exist and quality gate passes, advance immediately.

---

READ:
(1) Change card description at {{.file_path}}
(2) CLAUDE.md for coding standards

IMPLEMENT the change with appropriate testing. Run the project's quality gate before advancing. Determine the commands from the project's agent instructions (`CLAUDE.md`, `AGENTS.md`), `docs/architecture/tech-stack.md` (the **Quality Gate** section), `docs/architecture/coding-standards.md`, `docs/architecture/test-gate.md`, or the repository's documented build/task targets. Fix ALL failures before advancing. No exceptions.

EXIT GATE:
- Change implemented per description
- Tests pass
- Quality gate passes

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
