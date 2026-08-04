# E34-F04 Specification — Question Surface Adoption for Design and Decision Prompts

## Requirements

This specification incrementally implements the E34 epic PRD's reusable,
composable skill-content goal. See the epic PRD **Goal**, **Business Value**,
and **Scope Boundaries** sections for business context and epic exclusions.

### Functional requirements

#### REQ-F-001 — Registered reusable Question procedure

Create `internal/sharkdata/default_data/skills/question-management/SKILL.md`
and register the normalized `question-management` canonical skill in both
`internal/sharkdata/default_data/manifest.yaml` and
`internal/sharkdata/default_data/skills/README.md`. The procedure must define:

- the materiality test and explicit non-material outcome;
- Question deduplication before creation;
- creation, responder and resolution-owner configuration, decision-document
  linkage, conditional `question_blocks` linkage, response, and resolution;
- E39's supported resolution kinds and authoritative resolution-pointer rule;
- the distinction between a routine E39 Question and Shark Attack council
  escalation; and
- the parent-owned mutation boundary for a dispatched worker.

#### REQ-F-002 — Decision-producer adoption

Update the following existing decision-producing content to invoke the shared
procedure when a material unresolved decision is found, while retaining
workflow-specific candidate cues and an explicit non-material rationale:

- `internal/sharkdata/default_data/skills/architecture/context/templates/architecture-doc.md`;
- `internal/sharkdata/default_data/skills/architecture/workflows/design-system.md`,
  `design-backend.md`, `design-database.md`, `design-frontend.md`, and
  `design-security.md`;
- `internal/sharkdata/default_data/skills/specification-writing/workflows/write-epic.md`,
  `write-feature-prd.md`, `refine-task-requirements.md`, and
  `decompose-epic.md`;
- `internal/sharkdata/default_data/skills/product-design/workflows/d01-vision.md`,
  `d04-feasibility.md`, `d06-user-insights.md`, `d07-user-needs.md`,
  `d08-user-personas.md`, `d09-journey-maps.md`, `d12-test-results.md`, and
  `d14-validated-designs.md`;
- `internal/sharkdata/default_data/skills/frontend-design/workflows/commit-to-aesthetic-direction.md`; and
- `internal/sharkdata/default_data/prompts/epic/refinement.md`,
  `internal/sharkdata/default_data/prompts/epic/design.md`, and
  `internal/sharkdata/default_data/prompts/feature/specification.md`.

The adoption language must require a linked Q### only for material unresolved
items. It must not equate absence of `TBD` text with decision closure.

#### REQ-F-003 — Existing authority and consumer boundaries

The shared procedure and every adopted producer must preserve these contracts:

- Use E39's existing Question commands and lifecycle; do not introduce a new
  CLI command, API route, entity, persistence field, workflow status, or
  migration.
- Add a `question_blocks` relation only after workflow configuration and only
  to named work that cannot safely proceed without the answer.
- Refer specialist disagreement, inconsistent cross-entity contracts, high
  blast radius, irreversibility, or no safe evidence-based path to
  `skills/shark-attack/workflows/council.md`; a bounded single-role Question
  stays in the ordinary E39 lifecycle with no council artifact.
- Preserve `solution-walkthrough` as an operator-approved consumer. It neither
  creates nor resolves Questions automatically.
- In a parent-run dispatch, a worker returns a bounded Question proposal;
  the parent owns Question creation, claims, responses, resolution, and gates.

### Non-functional requirements

#### REQ-NF-001 — Content-only and bundle integrity

The delivery changes only bundle markdown, manifest/index metadata, rendered
prompt goldens, and focused content tests. The production renderer must parse
every changed prompt and every new skill identity must be consistent across the
directory, front matter, manifest, and skills index.

#### REQ-NF-002 — Safe and durable decision evidence

The procedure must direct resolution evidence to the narrowest authoritative
document or valid E39 resolution destination. It must not put credentials,
rendered prompts, full transcripts, or unbounded chat history into Question
fields. It must retain the existing claim-bound responder and parent-owned
mutation boundaries.

### Acceptance criteria

1. `TestE34F04QuestionManagementBundle_TC001_TC005` in
   `internal/sharkdata/embed_test.go` proves the manifest, skills index, and
   `question-management/SKILL.md` agree on the skill identity and contain the
   required lifecycle, authority, council, and non-material vocabulary.
2. `TestE34F04QuestionManagementPromptReferences` in
   `internal/cli/commands/interaction_prompts_test.go` renders
   `epic/refinement.md`, `epic/design.md`, and `feature/specification.md` with
   the production renderer and asserts their shared-procedure and durable-Q###
   policy references.
3. The same focused content test reads every path enumerated in REQ-F-002 and
   verifies its materiality cue plus shared `question-management` reference;
   it does not emulate lifecycle policy.
4. `TestRenderedPromptsGolden` in
   `internal/cli/commands/next_golden_test.go` passes after intentional updates
   to `internal/cli/commands/testdata/rendered-prompts/epic/refinement.golden`,
   `internal/cli/commands/testdata/rendered-prompts/epic/design.golden`,
   `internal/cli/commands/testdata/rendered-prompts/epic/decomposition.golden`,
   and `internal/cli/commands/testdata/rendered-prompts/feature/specification.golden`.
5. `make fmt`, `make lint`, `make test`, and `git diff --check` pass without
   introducing Go runtime, schema, API, workflow-YAML, or migration changes.

### Out of scope

- E39 Question platform behavior, including `internal/models/question.go`,
  `internal/services/question_workflow_service.go`, repositories, schema,
  commands, HTTP handlers, and Question workflow YAML.
- A new Shark Attack policy, council artifact format, or materiality threshold.
- Automatically creating, responding to, resolving, or blocking Questions from
  solution walkthrough or any workflow that lacks parent mutation authority.
- Questions for routine fact lookup, settled choices, low-impact prose
  preferences, or every speculative assumption.

## Architecture

### Component changes

| Surface | Change |
|---|---|
| `internal/sharkdata/default_data/skills/question-management/SKILL.md` | New reusable, bundle-local content procedure; no runtime implementation. |
| `internal/sharkdata/default_data/manifest.yaml` | Add canonical `question-management` identity. |
| `internal/sharkdata/default_data/skills/README.md` | Add the contributor-facing skill inventory row. |
| Architecture, specification-writing, product-design, frontend-design, and prompt files enumerated in REQ-F-002 | Replace material interactive-only open-question handling with a reference to the shared procedure and preserve local candidate cues. |
| `internal/sharkdata/embed_test.go` | Add `TestE34F04QuestionManagementBundle_TC001_TC005`. |
| `internal/cli/commands/interaction_prompts_test.go` | Add `TestE34F04QuestionManagementPromptReferences`. |
| `internal/cli/commands/testdata/rendered-prompts/epic/refinement.golden`, `internal/cli/commands/testdata/rendered-prompts/epic/design.golden`, `internal/cli/commands/testdata/rendered-prompts/epic/decomposition.golden`, `internal/cli/commands/testdata/rendered-prompts/feature/specification.golden` | Regenerated snapshots for direct prompt and included-workflow edits. |

### Data model changes

None. E34-F04 consumes E39's existing `Question`, `QuestionState`, configured
responder, resolution-owner, and resolution-provenance model. It adds no table,
column, index, migration, or persisted policy field.

### API and interface contracts

No public runtime interface changes. The content procedure uses the existing
Question interface: `shark question create`, `shark question configure-workflow`,
`shark question respond`, `shark question resolve`, `shark related-docs add`,
and `shark link ... --type=question_blocks`. Its gate order is fixed:
deduplicate, create or reuse, configure owner/responders, link decision source,
then add a blocking edge only if progress is unsafe. Resolution first updates
the authoritative decision record, then records E39 provenance.

### Key technical decisions

1. **One shared skill, many thin references.** This follows the embedded bundle
   organization in `internal/sharkdata/default_data/skills/README.md` and
   avoids copying lifecycle commands into every producer.
2. **Reuse E39 rather than duplicate its protocol.** The existing lifecycle in
   `internal/services/question_workflow_service.go` validates configured
   responders, claim-bound responses, and typed resolution pointers. Content
   may explain those constraints but cannot implement alternatives.
3. **Keep blocking explicit and ordered.** `internal/services/question_blocker.go`
   only qualifies configured, open or answering blocking Questions. The skill
   therefore configures before linking and treats a block as a safety gate,
   never as a priority marker.
4. **Delegate material escalation.** The threshold remains in
   `internal/sharkdata/default_data/skills/shark-attack/workflows/council.md`;
   the new skill routes to it instead of defining a second council process.
5. **Test rendered content, not a simulated policy engine.** This follows
   `internal/cli/commands/next_golden_test.go` and the focused content-test
   approach in `internal/cli/commands/interaction_prompts_test.go`.

### Integration with existing code

The feature consumes the Question lifecycle exposed by
`internal/cli/commands/question.go`, the directed relationship validation in
`internal/cli/commands/link.go`, and the typed blocking qualifier in
`internal/services/question_blocker.go`. It consumes, without modifying, the
Shark Attack authority and routing definitions at
`internal/sharkdata/default_data/skills/shark-attack/workflows/council.md` and
`route-question.md`, and the operator boundary at
`internal/sharkdata/default_data/skills/solution-walkthrough/SKILL.md`.

There is no parent epic architecture document in the E34 plan folder. The
research report's Capability map is the applicable brownfield design input;
this specification introduces no system-level deviation from it.

## Cross-feature interactions

E34-F04 does not produce or consume the sole mapped interaction, **I-01**.
`E34-interaction-map.md#i-01-readiness-evidence-shape` remains exclusively
E34-F03's producer contract and E34-F02's consumer contract with shared
pointer `E34-F03-deliverable-feature-decomposition-and-staged-integ/test-plan.md#TC-002`.
E34-F04 must not add, rename, or mirror an I-## interaction for its local
Question-content adoption.

## Cross-epic integrations

No X-## row applies. E34-F04 reuses E39's local Question interface but is not
the producer or consumer named by the product map's X-06 row, whose ownership
is E39-F04 and E38-F09. It must not claim, alter, or add coverage for X-06.
