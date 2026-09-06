{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Write specification for feature {{.id}}: "{{.title}}".

Check for existing spec.md in feature directory. If spec exists meeting exit gate criteria, advance immediately.

---

{{template "_product_critical_path_guard" .}}

COMBINED REQUIREMENTS + ARCHITECTURE SPECIFICATION

This is a SINGLE document (spec.md) that covers both what to build and how to build it. In brownfield development, the architect who understands the codebase is best positioned to write both.

{{include: skills/architecture/SKILL.md}}

{{include: skills/specification-writing/SKILL.md}}

READ:
(1) Parent epic PRD for business context and scope (DO NOT restate — reference by section)
(2) Parent epic architecture doc for system-level decisions
(3) Validated feature research report and its Capability map (required for every tier that reaches specification)
(4) Feature description at {{.file_path}}
(5) CLAUDE.md for coding standards and patterns
(6) Existing code in affected areas — use
    `skills/quality/workflows/state-space-coverage.md#dependency-discovery-by-interaction-and-caller-path`
    to find every dependency before specifying a change
(7) Parent epic {{.epic_id}}-interaction-map.md if present. Use only I-## IDs
    from that map; do not invent new interaction IDs.
(8) Parent epic {{.epic_id}}-cross-epic-map.md and
    docs/product/cross-epic-integration-map.md if present. Use only X-## IDs
    from those maps; do not invent new cross-epic IDs.

PRODUCE spec.md with these sections:

**Requirements** (INCREMENTAL over epic — only what this feature adds):
(1) Functional requirements with IDs (REQ-F-001, etc.). For a behavior-bearing
    lifecycle field, declare a closed table per
    `skills/quality/workflows/state-space-coverage.md`'s "Closed lifecycle
    tables" section instead of a prose progression.
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
- The map-assigned gate mode is `live` by default. A `contract-only` I-## is
  valid only when declared here and mirrors counterpart identities, a current
  status read live from Shark, shared-contract evidence, activation owner, closure key, and review basis.
  Do not invent or alter these map-assigned values in a feature spec.

## Cross-epic integrations

Required when this feature produces, consumes, or validates any X-## row from
{{.epic_id}}-cross-epic-map.md or docs/product/cross-epic-integration-map.md.
Add this section to spec.md for every X-## this feature touches:

- **Produces**: X-## - integration purpose, consumer epic(s)/feature(s),
  contract / shape source, UX / CX handoff notes, test coverage pointer or
  explicit deferral
- **Consumes**: X-## - integration purpose, producer epic/feature, contract /
  shape source, UX / CX handoff notes, test coverage pointer or explicit
  deferral
- **Validates**: X-## - shared contract or journey handoff this feature proves,
  contract / shape source, test coverage pointer
- **Contract / shape source**: the exact source from the global product map
- **Test coverage**: test-plan.md TC, test file pointer, or explicit deferral
  linked to docs/product/progress.md

Notes:
- Use X-## IDs verbatim from the cross-epic maps. Do not invent new IDs here.
- Producer and consumer features reference the same contract / shape source
  verbatim.
- X-## rows are product-level cross-epic integrations. Do not replace I-##
  cross-feature interactions with X-## rows.

CRITICAL RULES:
- DO NOT restate epic-level business context. Say "See epic PRD Section X" instead.
- DO NOT restate existing architecture. Say "Follows pattern in internal/services/task_service.go" instead.
- Use the research report's Capability map to name reused or extended capabilities and what this feature will not re-implement.
- Requirements MUST trace to epic PRD requirements.
- Architecture MUST align with CLAUDE.md patterns.
- Include file paths for ALL code that will be modified.
- Cross-feature interactions MUST mirror the parent interaction map exactly.
  Internal contracts within this feature may use local names, but interfaces
  crossing outside this feature use I-##.
- Cross-epic integrations MUST mirror {{.epic_id}}-cross-epic-map.md and
  docs/product/cross-epic-integration-map.md exactly. Interfaces crossing epic
  boundaries use X-##, not I-## or CONTRACT-###.

## Durable unresolved decisions

For each material unresolved requirement or architecture decision, use
`skills/question-management/SKILL.md` to create or reuse a linked Q###. Record
a non-material rationale in `spec.md`; do not treat the absence of `TBD` text
as decision closure.

EXIT GATE:
- Every requirement is testable
- Every architecture decision references existing patterns or explains deviation
- File paths listed for all changes
- No TBDs in critical sections
- Multi-feature epic: every I-## this feature produces or consumes is declared
  in the "Cross-feature interactions" section with shape source and contract
  test pointer
- Every X-## this feature produces, consumes, or validates is declared in the
  "Cross-epic integrations" section with producer/consumer feature ownership,
  matching contract / shape source, UX / CX handoff notes, and test coverage
  pointer or explicit progress-log deferral

DECISION:
- Exit gate met -> recommended_outcome: pass
- Exit gate not met -> recommended_outcome: fail. This outcome's role is
  `route_rework` — `gate_result.kickbacks` must stay empty; state the
  specific gaps in `gate_result.summary`.

{{template "_gate_result_directive" .}}
