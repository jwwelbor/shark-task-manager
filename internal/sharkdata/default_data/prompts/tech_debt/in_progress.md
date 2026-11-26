{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

RESUME tech debt resolution for {{.id}}: "{{.title}}".

Category: {{.category}} | Severity: {{.severity}}

Check for existing resolution: review recent git changes. If resolution exists and the project quality gate passes (commands from `CLAUDE.md`, `AGENTS.md`, `docs/architecture/tech-stack.md` **Quality Gate** section, `docs/architecture/coding-standards.md`, `docs/architecture/test-gate.md`, or the repository's documented build/task targets), advance immediately.

Otherwise, continue with resolution per triaged instructions.

DECISION:
- Exit gate met -> recommended_outcome: pass
- Cannot resolve (external blocker) -> recommended_outcome: blocked; state
  the blocker in `gate_result.no_kickback_reason`
- Determined not worth fixing -> recommended_outcome: wont_fix; state why in
  `gate_result.no_kickback_reason`
- Otherwise unresolved -> recommended_outcome: fail. This outcome's role is
  `route_rework` — `gate_result.kickbacks` must stay empty; state the
  specific blockers in `gate_result.summary`.

{{template "_gate_result_directive" .}}
