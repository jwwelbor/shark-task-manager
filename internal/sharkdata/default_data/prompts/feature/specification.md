{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Write specification for feature {{.id}}: "{{.title}}".

Check for existing spec.md in feature directory. If spec exists meeting exit gate criteria, advance immediately.

---

COMBINED REQUIREMENTS + ARCHITECTURE SPECIFICATION

This is a SINGLE document (spec.md) that covers both what to build and how to build it. In brownfield development, the architect who understands the codebase is best positioned to write both.

{{include: skills/architecture/SKILL.md}}

{{include: skills/specification-writing/SKILL.md}}

READ:
(1) Parent epic PRD for business context and scope (DO NOT restate — reference by section)
(2) Parent epic architecture doc for system-level decisions
(3) Feature research report (if exists, for COMPLEX features)
(4) Feature description at {{.file_path}}
(5) CLAUDE.md for coding standards and patterns
(6) Existing code in affected areas (grep for related services, models, repos)
(7) Parent epic {{.epic_id}}-interaction-map.md if present. Use only I-## IDs
    from that map; do not invent new interaction IDs.

PRODUCE spec.md with these sections:

**Requirements** (INCREMENTAL over epic — only what this feature adds):
(1) Functional requirements with IDs (REQ-F-001, etc.)
(2) Non-functional requirements (performance, security)
(3) Acceptance criteria (testable, specific)
(4) Out of scope for this feature

**Architecture** (detailed design for this feature):
(1) Component changes: what files to modify/create
(2) Data model changes (schemas, migrations)
(3) API/interface contracts (if applicable)
(4) Key technical decisions with rationale
(5) Integration with existing code (specific file paths, function signatures)

## Cross-feature interactions

Required for STANDARD/COMPLEX features under a multi-feature epic. If the parent
epic has an interaction map, add this section to spec.md for every I-## this
feature touches:

- **Produces**: I-## - short name, consumer feature(s), shape source, contract
  test pointer
- **Consumes**: I-## - short name, producer feature, shape source, contract test
  pointer
- **Shape source**: architecture.md#section-name from the parent epic
- **Contract tests**: tests/contracts/<epic-or-feature>_interactions_test.<ext>#TC-###

Notes:
- Use the I-## IDs verbatim from {{.epic_id}}-interaction-map.md. Do not invent
  new IDs here.
- Producer and consumer features reference the same shape source verbatim.
- Producer and consumer features reference the same contract test pointer
  verbatim. The same test proves the shared contract; do not create twin tests.

CRITICAL RULES:
- DO NOT restate epic-level business context. Say "See epic PRD Section X" instead.
- DO NOT restate existing architecture. Say "Follows pattern in internal/services/task_service.go" instead.
- Requirements MUST trace to epic PRD requirements.
- Architecture MUST align with CLAUDE.md patterns.
- Include file paths for ALL code that will be modified.
- Cross-feature interactions MUST mirror the parent interaction map exactly.
  Internal contracts within this feature may use local names, but interfaces
  crossing outside this feature use I-##.

EXIT GATE:
- Every requirement is testable
- Every architecture decision references existing patterns or explains deviation
- File paths listed for all changes
- No TBDs in critical sections
- Multi-feature epic: every I-## this feature produces or consumes is declared
  in the "Cross-feature interactions" section with shape source and contract
  test pointer
