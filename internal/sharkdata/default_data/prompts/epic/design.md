{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Design architecture for epic {{.id}}: "{{.title}}".

Check for existing architecture docs and UAT plan in epic directory. If both exist and meet exit gate criteria, advance immediately.

---

Load skills from `shark-data/skills/architecture/`:
  - `workflows/design-system.md`
  - `workflows/design-backend.md`
  - `workflows/design-database as applicable.md`

READ:
(1) Epic PRD for scope and requirements
(2) Research report for existing implementations and extension points
(3) CLAUDE.md for architecture patterns and coding standards
(4) Existing architecture docs in codebase for consistency

PRODUCE two documents:

**architecture.md** — System-level design:
(1) Component overview: what changes, what stays
(2) Key technical decisions with rationale (ADRs)
(3) Data model changes (if any)
(4) Integration approach (how new code connects to existing)
(5) Migration strategy (if applicable)

**uat-plan.md** — Epic acceptance plan:
(1) UAT scenarios derived from epic PRD success criteria
(2) Acceptance test descriptions (what to verify, not how)
(3) Cross-feature integration scenarios
(4) Performance/security considerations

CRITICAL: Architecture must align with existing patterns from research report. Do not propose patterns that contradict CLAUDE.md conventions.

EXIT GATE:
- Architecture follows existing codebase patterns
- Every technical decision has rationale
- UAT scenarios map to epic success criteria
- No orphaned requirements
