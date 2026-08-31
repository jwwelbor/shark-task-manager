---
research_schema: 2
entity_key: E34
entity_type: epic
recipe: universal
rigor: complex
categories:
  - backend
  - workflow_operations
  - documentation
related_work: true
---

# E34 Research Report: Prompt and Skill Improvements

## Scope

E34 is a complex, cross-boundary improvement to Shark's reusable workflow
policy. It consolidates the existing prompt and skill capabilities for
harness-aware rendering, evidence-based demos, staged interactions, and
material Questions; it then plans new contracts for parent-persisted gate
results, defect-class closure, decision propagation, tier-consistent quality
gates, final epic integration review, and override-drift visibility.

The epic does not add a project-specific command, test runner, model policy,
or automatic override merge. It also does not make E40's benchmark corpus an
implementation prerequisite. The parent loop retains claims, persistence,
transitions, and releases; a dispatched worker returns bounded evidence.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/{epic.md,requirements.md,scope.md}`; `E34-interaction-map.md`; and `architecture.md`. These define the stable terms: `I-##`, `GateResult`, `DefectClassSweep`, `ChangeImpactSet`, `CanonicalAdoptionManifest`, `integration_review`, and override classifications.
- [x] `affected_implementation_or_contract` — Evidence: `internal/sharkdata/default_data/{workflow/epic.yaml,workflow/feature.yaml,manifest.yaml}`; `internal/sharkdata/default_data/prompts/{epic/refinement.md,feature/code_review.md,feature/qa.md}`; `internal/runner/{controller.go,dispatcher.go}`; and `internal/cli/commands/{next.go,sharkdata_cmd.go}`. Brownfield evidence shows an embedded prompt/skill bundle, route-based workflows, runner dispatch, and admin/bundle surfaces that E34 must extend rather than replace.
- [x] `related_work` — Evidence: E34 feature packets `E34-F01-harness-aware-prompt-rendering/feature.md`, `E34-F02-evidence-based-demo-script-skill/research-report.md`, `E34-F03-deliverable-feature-decomposition-and-staged-integ/research-report.md`, `E34-F04-question-surface-adoption-for-design-and-decision/research-report.md`, and `E34-F05` through `E34-F09` `research-report.md`; `docs/plan/E40-shark-bench-workflow-benchmarking-harness/{epic.md,shark-bench-design.md}`; and `E34-review-quality-improvement-plan.md`.
- [x] `pattern_contract` — Evidence: `internal/sharkdata/default_data/skills/{README.md,question-management/SKILL.md,quality/workflows/review-code.md}`; `skills/shark-rider/verbs/run.md`; `internal/models/question.go`; and `internal/services/question_workflow_service.go`. Existing patterns establish embedded-skill registration, shared workflow guidance, bounded parent-owned Question handoff, and route-configured outcomes.
- [x] `dependency_impact` — Evidence: `E34-interaction-map.md`; `architecture.md`; `E34-F05-structured-gate-results-and-parent-owned-persisten/feature.md`; `E34-F08-tier-consistent-gates-and-final-integration-review/feature.md`; and `E34-F09-override-drift-visibility-and-wwgm-reconciliation/feature.md`. I-02 flows from F05 to F06–F08; I-03 and I-04 feed F08; I-05 feeds F09; E40 is only a later validation consumer.
- [x] `cross_boundary_risks` — Evidence: `E34-F05-structured-gate-results-and-parent-owned-persisten/research-report.md`; `E34-F08-tier-consistent-gates-and-final-integration-review/research-report.md`; `E34-F09-override-drift-visibility-and-wwgm-reconciliation/research-report.md`; and `scope.md`. The planned worker-to-parent result crosses untrusted model output and lifecycle persistence; final review crosses feature verdict authority; and override inspection crosses canonical content and project-owned files.
- [x] `alternatives` — Evidence: `scope.md#alternatives-considered`; `E34-review-quality-improvement-plan.md`; and `E34-F09-override-drift-visibility-and-wwgm-reconciliation/feature.md`. The packet explicitly rejects worker-authored notes, separate free-form parsers, retry-count escalation, one-hop impact analysis, and automatic override merges.

## Capability map

| Capability | Brownfield evidence | Decision | E34 responsibility |
|---|---|---|---|
| Harness-aware prompt rendering | F01 feature; `internal/cli/commands/next.go`; prompt-rendering tests | EXTEND | Preserve the dispatch/render handshake and add variants without rewriting every prompt. |
| Embedded skill delivery and evidence-based demos | `manifest.yaml`; `skills/README.md`; F02 research report | REUSE | Keep the shipped `demo-script` skill and host-local Mode-3 action; demos remain evidence consumers, not UAT authority. |
| Interaction map and staged handoff | `E34-interaction-map.md`; F03 research report; specification-writing interaction-map template | EXTEND | Retain I-## as the single shape registry and require explicit `contract-only` closure evidence. |
| Durable material Questions and councils | `question-management/SKILL.md`; F04 feature; Question model/service | REUSE | Route unresolved material decisions through E39/E38 instead of creating a parallel decision store. |
| Parent-owned gate-result persistence | Rider run contract and core runner parse/transition path; F05 research report | NEW | Define versioned `GateResult`, validation, idempotent persistence, and Rider/core parity before a parent transition. |
| Defect-class sweep and structural guard | Existing review/UAT workflows; F06 feature and research report | EXTEND | Turn existing sibling-sweep and re-verification guidance into one reusable I-03 contract; do not add a recurrence table. |
| State and decision-impact closure | Interaction map; F07 research report; existing specs and caller-path tests | NEW | Define I-04 as a bounded planning contract for affected artifacts, consumers, amendments, and follow-ups. |
| Complexity-tier gate routing | `workflow/feature.yaml`; feature review/QA prompts; F08 research report | EXTEND | Centralize the SIMPLE/STANDARD/COMPLEX artifact and evidence matrix while preserving configured routes. |
| Final epic integration review | `workflow/epic.yaml` currently completes from active; F08 feature | NEW | Insert an additive `integration_review` over the accumulated diff; it cannot supersede a failed required feature gate. |
| Canonical adoption and override drift | `internal/sharkdata/{embed.go,resolve_at_test.go}`; admin bundle commands; F09 feature | NEW | Add digest-only classification and explicit acknowledgement; never merge, expose, or rewrite project overrides. |
| E40 benchmark evaluation | E40 epic/design documents; E34 improvement plan | REUSE AS LATER VALIDATION | Record scenarios and metrics after harness readiness; do not make E40 a dependency. |

## Findings

1. E34 is an extension epic, not a prompt rewrite. The embedded bundle,
   manifest, route-based workflow YAML, renderer, Question platform, and
   related-document registry already supply the durable surfaces. The new work
   is limited to the missing cross-boundary contracts and their consumers.

2. Parent ownership is the governing safety boundary. Existing Question
   handling demonstrates bounded parsing and durable, replay-aware parent
   mutation. `GateResult` should generalize that pattern for configured gates;
   workers must not claim, persist, advance, or release the dispatched entity.

3. The interaction map supplies a complete dependency sequence: F05 produces
   I-02; F06 and F07 independently produce I-03 and I-04; F08 consumes them
   and produces I-05; F09 consumes I-05 for a deliberate cross-project
   adoption plan. This sequence is the implementation contract, not merely
   documentation.

4. Existing tier routes are useful but their policy is distributed. A single
   tier matrix and executable evidence fields are needed so SIMPLE and
   STANDARD are not held to nonexistent artifacts, and COMPLEX still receives
   its separate QA path.

5. The current epic workflow has no named whole-epic review. An additive
   `integration_review` is required to check accumulated changes and cross-
   feature closure, while feature-gate verdicts remain independently
   authoritative.

6. Replace-only overrides make local policy a compatibility boundary. Digest
   provenance and read-only classification are feasible on existing embedded
   bundle/admin surfaces; automatic three-way merge is unsafe because prompts
   and workflow YAML carry semantic policy.

## Decisions

1. Treat E34 as **complex**: all universal-recipe modules apply because it
   changes backend, workflow, documentation, and cross-project adoption
   contracts.
2. Reuse the existing embedded bundle, Questions/councils, I-## map, notes,
   and route configuration before adding any storage or lifecycle mechanism.
3. Implement F05 → F06/F07 → F08 → F09 in the declared dependency order;
   retain F01–F04 as existing capability evidence and consumers.
4. Use one bounded, versioned `GateResult` and parent persistence before any
   configured route executes. Reject malformed, conflicting, or partially
   persisted results from advancing work.
5. Make integration review additive, tier evidence executable, and override
   drift read-only. Keep E40 as a separately measured validation consumer.

## Sources

- `internal/sharkdata/default_data/research/recipes.yaml`
- `docs/plan/E34-prompt-and-skill-improvements/{epic.md,requirements.md,scope.md,architecture.md,E34-interaction-map.md,E34-review-quality-improvement-plan.md}`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F01-harness-aware-prompt-rendering/feature.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F03-deliverable-feature-decomposition-and-staged-integ/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F04-question-surface-adoption-for-design-and-decision/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F05-structured-gate-results-and-parent-owned-persisten/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F06-defect-class-completeness-and-recurrence-routing/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F07-state-space-planning-and-decision-propagation/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F08-tier-consistent-gates-and-final-integration-review/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F09-override-drift-visibility-and-wwgm-reconciliation/research-report.md`
- `internal/sharkdata/default_data/{manifest.yaml,workflow/epic.yaml,workflow/feature.yaml,skills/README.md}`
- `internal/{runner/controller.go,runner/dispatcher.go,models/question.go,services/question_workflow_service.go}`
- `skills/shark-rider/verbs/run.md`

RECOMMENDED OUTCOME: pass
