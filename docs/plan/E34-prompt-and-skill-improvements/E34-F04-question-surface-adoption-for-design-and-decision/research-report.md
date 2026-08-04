---
research_schema: 2
entity_key: E34-F04
entity_type: feature
recipe: universal
rigor: standard
categories:
  - workflow_operations
  - documentation
related_work: true
---

# E34-F04 Research Report: Question Surface Adoption for Design and Decision Prompts

## Scope

E34-F04 adopts the completed E39 Question lifecycle in the design- and
decision-producing bundle content. It adds one registered reusable
`question-management` skill and makes selected architecture, specification,
product-design, frontend-design, and workflow prompts invoke it at their
material-open-decision boundary. A Question remains the lifecycle record; the
narrowest product, ADR, epic, or local design/specification document remains
the authoritative decision record.

This is a STANDARD feature (12/27). It is content-only: no Question schema,
CLI, API, workflow YAML, database, or migration work is in scope. It must not
weaken the existing Shark Attack council threshold or make solution walkthrough
create or resolve Questions automatically.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `E34-F04-question-surface-adoption-for-design-and-decision/feature.md`; `internal/models/question.go`; `internal/services/question_workflow_service.go`.
  These sources establish Q###,
  configured responders, resolution owner, authoritative resolution pointer,
  and supported resolution kinds.
- [x] `affected_implementation_or_contract` — Evidence: `internal/sharkdata/default_data/skills/question-management/SKILL.md`; `internal/sharkdata/default_data/manifest.yaml`; `internal/sharkdata/default_data/skills/README.md`; `internal/cli/commands/question.go`.
  The proposed local content is not yet in the
  manifest or index, while the E39 CLI already provides create, configure,
  respond, resolve, and read surfaces.
- [x] `related_work` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/{epic.md,requirements.md,scope.md}`; `E34-F01-harness-aware-prompt-rendering/feature.md`; `E34-F02-evidence-based-demo-script-skill/{feature.md,research-report.md}`; `E34-F03-deliverable-feature-decomposition-and-staged-integ/{feature.md,research-report.md}`.
- [x] `pattern_contract` — Evidence: `internal/sharkdata/default_data/skills/shark-attack/workflows/{council.md,route-question.md}`; `internal/sharkdata/default_data/skills/solution-walkthrough/SKILL.md`; `internal/sharkdata/default_data/prompts/{epic/refinement.md,epic/design.md,feature/specification.md}`.
  `internal/sharkdata/default_data/skills/architecture/workflows/design-*.md`;
  `internal/sharkdata/default_data/skills/specification-writing/workflows/{write-epic.md,write-feature-prd.md,refine-task-requirements.md,decompose-epic.md}`;
  `internal/sharkdata/default_data/skills/product-design/workflows/{d01-vision.md,d04-feasibility.md,d06-user-insights.md,d07-user-needs.md,d08-user-personas.md,d09-journey-maps.md,d12-test-results.md,d14-validated-designs.md}`;
  and `internal/sharkdata/default_data/skills/frontend-design/workflows/commit-to-aesthetic-direction.md`.

## Capability map

| Capability | Evidence | Decision | E34-F04 responsibility |
|---|---|---|---|
| E39 Question lifecycle and responder-owned resolution | `internal/models/question.go`; `internal/services/question_workflow_service.go`; `internal/cli/commands/question.go` | REUSE | Refer decision producers to the existing Q### lifecycle; do not recreate runtime behavior. |
| Direct, configured Question gate | `internal/services/question_blocker.go`; `internal/cli/commands/link.go` | REUSE | Instruct authors to add `question_blocks` only after configuration and only where progress is unsafe. |
| Shark Attack routine-versus-council threshold | `skills/shark-attack/workflows/{council.md,route-question.md}` and embedded counterparts | REUSE | Reference the canonical materiality threshold; no competing escalation artifact or threshold. |
| Solution-walkthrough decision consumer | `internal/sharkdata/default_data/skills/solution-walkthrough/SKILL.md` | REUSE | Preserve its operator-approved consumption boundary; it must not auto-create or auto-resolve Questions. |
| Existing open-question/TBD prompt review | architecture `design-*.md`, specification-writing `write-*.md`, product-design D01/D04/D06/D07/D08/D12/D14, frontend aesthetic-direction workflow, and epic/feature prompts named in the checklist | EXTEND | Replace material interactive-only closure with the shared Question procedure while retaining documented non-material rationale. |
| Bundle skill registration and rendered-prompt regression guard | `internal/sharkdata/default_data/{manifest.yaml,README.md,skills/README.md}`; `internal/cli/commands/next_golden_test.go` | EXTEND | Register the new skill, update the contributor index, render changed prompts, and commit only affected golden fixtures plus focused content checks. |
| `question-management` reusable procedure | proposed `internal/sharkdata/default_data/skills/question-management/SKILL.md` | NEW | Land one concise, copyable bundle skill that ties existing lifecycle commands and authority boundaries together; it is not a new platform capability. |
| Harness-aware render variants | `E34-F01-harness-aware-prompt-rendering/feature.md` | REUSE | Do not alter provider/harness routing; changed prompts remain compatible with Shark-owned assembly. |
| Evidence-based demo and staged acceptance vocabulary | `E34-F02.../research-report.md`; `E34-F03.../research-report.md`; `E34-interaction-map.md` | REUSE | Keep Questions distinct from demos, UAT verdicts, and I-01 readiness evidence. |

No contradictory capability was found.

## Findings

1. E39 already provides the needed durable lifecycle. Workflow configuration
   requires a real resolution owner and responders; response recording is
   claim-bound, and resolution requires all responders plus a valid typed
   evidence destination. E34-F04 should document that lifecycle rather than
   encode policy in another data shape.

2. A configured, open or answering blocking Question is the only qualifying
   source for the direct `question_blocks` gate. The proposed skill correctly
   orders configuration before linking, but the feature must verify this wording
   against the E39 command surface and retain the parent-owned mutation rule.

3. Current producer workflows repeatedly require interactive handling of
   `TBD`, `TODO`, deferred, and open-decision text. The affected set crosses
   architecture, specification, product, frontend, and prompt layers, so a
   shared skill reference plus focused per-workflow candidate cues is the
   consistent extension; copied command sequences in each workflow would drift.

4. Shark Attack already distinguishes routine scope-bounded Questions from
   council cases involving specialist disagreement, inconsistent cross-entity
   contracts, high blast radius, irreversibility, or no safe evidence-based
   path. E34-F04 must defer to that rule and preserve its existing council
   routing instead of treating every material Question as a council artifact.

5. The embedded bundle requires new skills to be declared by normalized name
   in `manifest.yaml` and discoverable through `skills/README.md`. Prompt
   changes are covered by the production renderer's golden corpus; focused
   content tests should assert the shared-skill reference, materiality,
   authority, and no-auto-resolution boundaries rather than simulate a policy
   engine.

## Decisions

1. **Create one content-level Question procedure.** Register the proposed
   `question-management` skill and make it the sole reusable lifecycle
   procedure for the selected decision producers.

2. **Extend producers, not the E39 platform.** Each selected producer should
   create/reuse a linked Q### for a material unresolved item or state why it is
   non-material; routine facts and low-impact authoring preferences remain in
   their local document.

3. **Preserve authority boundaries.** Workers that cannot mutate Shark return
   a bounded Question proposal; the parent owns creation, claims, responses,
   resolution, and gates. Only explicitly linked work may be blocked.

4. **Keep council and walkthrough boundaries intact.** Use Shark Attack's
   existing threshold for escalation; retain solution walkthrough as an
   operator-approved consumer of durable Questions.

5. **Validate as content.** Use bundle registration/index checks, production
   prompt rendering with affected goldens, referenced-file existence, focused
   content assertions, `git diff --check`, and the standard Go quality gate.

## Sources

- `internal/sharkdata/default_data/research/recipes.yaml`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F04-question-surface-adoption-for-design-and-decision/feature.md`
- `docs/plan/E34-prompt-and-skill-improvements/{epic.md,requirements.md,scope.md,E34-interaction-map.md}`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F0{1,2,3}-*/{feature.md,research-report.md}`
- `internal/models/question.go`; `internal/services/{question_workflow_service.go,question_blocker.go}`; `internal/cli/commands/{question.go,link.go}`
- `internal/sharkdata/default_data/{manifest.yaml,README.md,skills/README.md}`
- `internal/sharkdata/default_data/skills/{question-management,shark-attack,solution-walkthrough,architecture,specification-writing,product-design,frontend-design}/`
- `internal/sharkdata/default_data/prompts/{epic/refinement.md,epic/design.md,feature/specification.md}`
- `internal/cli/commands/next_golden_test.go`
