---
research_schema: 2
rigor: complex
categories:
  - workflow_operations
  - documentation
related_work: true
---

# E34-F10 Research Report: Product Critical-Path Guard for Delivery Workflows

## Scope

E34-F10 adds a reusable pre-dispatch guard that makes sprint, epic, feature, and
task workflow prompts consult a durable product critical-path artifact (sourced
from `docs/product/D01-vision-statement.md`, `docs/product/D02-success-criteria.md`,
and `docs/plan/product-delivery-roadmap.md`) and report the current gate,
contribution, evidence, prerequisites, and side-quest disposition before
selecting or dispatching work. It covers the shared guard content and its
touchpoints across the twelve named workflow stages. It does not implement
runtime DB-level admission enforcement (E19-F10's boundary) and does not
author the D01/D02 vision/success-criteria content itself (E36-F02's product-design
boundary) — it only consumes those artifacts and defines the critical-path
artifact's own shape.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `E34-F10-product-critical-path-guard-for-delivery-workflows/feature.md`; `docs/plan/E34-prompt-and-skill-improvements/requirements.md` (Area 10, REQ-F-032/033/034); `docs/plan/E34-prompt-and-skill-improvements/epic.md` feature-portfolio table. These sources define "product-roadmap gate," "last passing production step," "executable advancement evidence," and the disallowed evidence classes.
- [x] `affected_implementation_or_contract` — Evidence: `internal/sharkdata/default_data/prompts/sprint/{planning.md,active.md,closing.md}`; `internal/sharkdata/default_data/prompts/epic/{assessment.md,decomposition.md,active.md}`; `internal/sharkdata/default_data/prompts/feature/{specification.md,test_planning.md,task_generation.md,task_review.md,approval.md}`; `internal/sharkdata/default_data/prompts/task/development.md` (completion reporting). A grep for `roadmap|critical.path|product gate|side.quest` across all twelve files returned zero matches — none currently reference a product-roadmap gate, a critical-path artifact, or side-quest disposition, so the guard is wholly new content in each.
- [x] `related_work` — Evidence: `docs/plan/E19-sprint-management-planning-system/E19-F10-roadmap-gated-sprint-admission-and-goal-acceptance/feature.md`; `docs/plan/E36-project-layer-and-consult-bridge/E36-F02-project-namespace-and-progress-record/feature.md`; `docs/product/cross-epic-integration-map.md` row X-10; `docs/plan/E34-prompt-and-skill-improvements/E34-interaction-map.md` (line 89); `docs/plan/E34-prompt-and-skill-improvements/architecture.md` (line 12); sibling reports `E34-F07-state-space-planning-and-decision-propagation/research-report.md` and `E34-F09-override-drift-visibility-and-wwgm-reconciliation/research-report.md`.
- [x] `pattern_contract` — Evidence: `internal/sharkdata/default_data/prompts/_partials/_qa_process.md` and `_partials/_commands.md` (e.g. the `advance_preamble` define) show the established "one shared `{{define}}` partial, invoked via `{{template "_name" .}}` from every consuming status prompt" pattern already used for cross-cutting policy (QA process, resume preamble).
- [x] `dependency_impact` — Evidence: the guard's producers are `docs/product/D01-vision-statement.md`, `docs/product/D02-success-criteria.md` (owned by E36-F02's product-design action, not yet run in this repository — see Findings), and `docs/plan/product-delivery-roadmap.md` / `docs/plan/product-critical-path.md` (no producer exists anywhere in the repository today). Its consumers are the twelve prompt touchpoints listed above, plus a forward dependency from E19-F10, whose feature.md explicitly states "E34-F10 owns the reusable prompt guard."
- [x] `cross_boundary_risks` — Evidence: `E34-interaction-map.md` line 89 ("F10 is an independent pre-dispatch product-alignment guard... does not produce or consume an I-## payload") and `E19-F10/feature.md` ("E34-F10 owns the reusable prompt guard") jointly define the boundary between F10's advisory prompt-layer guard and E19-F10's DB-level sprint-admission enforcement; a guard that duplicates roadmap-layer checks in Go risks two divergent enforcement paths. Separately, `docs/product/` currently contains only `progress.md` and `cross-epic-integration-map.md` — none of the four REQ-F-032 source files exist in this repository, so a guard worded as an unconditional block risks stalling every dispatch cycle in a project (such as this one) that has not run product-design or authored a roadmap.
- [x] `alternatives` — Evidence: E19-F10's feature.md and the E34 architecture's "reusable bundle content" principle (`architecture.md` §Design principles #1, "Parent owns state" / prompts-only policy) support keeping F10 as prompt/skill content rather than a new Go command or DB-backed entity; `internal/sharkdata/default_data/skills/product-design/workflows/d01-vision.md` through `d14-validated-designs.md` show the D01–D14 arc stops at validated designs and never produces a roadmap or critical-path artifact, so an alternative of "extend product-design to D15/D16" was considered and rejected in favor of F10 defining its own lightweight, human-authored artifact shape consistent with D01/D02's plain-markdown convention.

## Capability map

| Capability | Evidence | Decision | E34-F10 responsibility |
|---|---|---|---|
| Shared `{{define}}` partial + `{{template}}` invocation pattern | `_partials/_qa_process.md`, `_partials/_commands.md` | REUSE | Author the guard as one `_partials/_product_critical_path_guard.md`-style define, invoked from all twelve touchpoints, so wording lives in one place per the architecture's "one producer contract, many consumers" principle. |
| D01/D02 vision & success-criteria artifacts | `skills/product-design/workflows/d01-vision.md`, `d02-success-criteria.md` | REUSE | Consume D01/D02 as sources only; do not author or extend them — that stays E36-F02's boundary (confirmed by X-10). |
| `docs/plan/product-delivery-roadmap.md` | No producer exists anywhere in the repository | NEW | Define the artifact's minimal shape (current gate, last passing production step, ordered upcoming gates) since REQ-F-032 requires it and nothing else creates it. |
| `docs/plan/product-critical-path.md` | No producer exists anywhere in the repository | NEW | Same as above — F10 must define this artifact's shape and who maintains it (human-authored, mirroring D01/D02's convention), not just consume it. |
| Sprint/epic/feature/task prompt touchpoints | the twelve files listed under `affected_implementation_or_contract` (verified via grep for `roadmap`/`critical.path`/`product gate`/`side.quest` across all twelve — zero matches) | EXTEND | Insert the shared guard invocation into each named status prompt without altering their existing status-specific instructions. |
| Runtime roadmap-gated admission enforcement | `E19-F10/feature.md` | REUSE | Depend on E19-F10's DB-level ancestor-dependency/readiness enforcement rather than duplicating it; F10 stays advisory prompt content only, retained local to F10's own scope. |
| Evidence-class exclusion language (fixture/capture/actor/contract-only/component) | `E34-F02-evidence-based-demo-script-skill/feature.md` and its skill content (evidence-vs-verdict distinctions) | REUSE | Phrase REQ-F-034's exclusion list consistently with F02's existing evidence-authenticity vocabulary rather than inventing new terms for the same concept. |

## Findings

1. **Zero of the four REQ-F-032 source artifacts exist in this repository today.** `docs/product/` holds only `progress.md` and `cross-epic-integration-map.md`; there is no `D01-vision-statement.md`, `D02-success-criteria.md`, `product-delivery-roadmap.md`, or `product-critical-path.md`. The guard's design must define a graceful "prerequisite artifact missing" disposition (report it as an unresolved prerequisite, per REQ-F-033) rather than assume the artifacts are always present — otherwise F10 would block dispatch in any Shark-managed project (including this one) that has not run the product-design arc.

2. **No existing skill or workflow authors `product-delivery-roadmap.md` or `product-critical-path.md`.** The product-design D01–D14 arc (`skills/product-design/workflows/`) stops at validated designs and never produces a roadmap or critical-path file. REQ-F-032 requires these two artifacts to exist, so F10's own scope must include defining their minimal schema and an authoring/update convention (most consistent with the existing D01/D02 pattern: human-elicited, plain markdown, "last updated" stamped) — this is in-scope work, not a pre-existing capability to merely reuse.

3. **The guard belongs as one shared partial, not twelve independent edits.** `_partials/_qa_process.md` and `_partials/_commands.md` establish the "define once, `{{template}}` everywhere" convention already used for cross-cutting policy (QA process, resume preamble, advance preamble). Twelve separate hand-copies of the same gate/contribution/evidence/prerequisite/side-quest reporting instructions would immediately violate the architecture's own "one producer contract, many consumers" principle and drift on the next unrelated edit to any one prompt.

4. **The boundary with E19-F10 is already explicit and non-overlapping.** Both `E19-F10/feature.md` ("E34-F10 owns the reusable prompt guard") and `E34-interaction-map.md` (line 89, "F10 is an independent pre-dispatch product-alignment guard... does not produce or consume an I-## payload") agree: E19-F10 enforces ancestor-dependency/roadmap-layer admission at the DB/service level for sprints; E34-F10 is advisory prompt content consulted before dispatch across all four entity levels. F10 must not add Go-level enforcement or a new DB table — doing so would duplicate E19-F10's responsibility and create two sources of roadmap-gate truth.

5. **REQ-F-034's evidence-exclusion list overlaps conceptually with E34-F02's evidence-authenticity work.** F02 already distinguishes genuine executable evidence from completion-status or override claims for demo scripts. F10 should reuse that vocabulary (rather than inventing parallel terms) when it says fixture, capture, hand-authored-actor, contract-only, and component-suite evidence cannot satisfy a production gate, keeping the epic's evidence language consistent across F02 and F10.

## Decisions

- **D-F10-01**: Implement the guard as shared prompt/skill markdown content (a `_partials/` define plus, if needed, a short skill reference), not as a new Go command, CLI flag, or database table. Rationale: matches E34's "reusable bundle content" architecture principle and F07/F09's precedent; keeps the guard host-agnostic across Shark Rider and the core runner.
- **D-F10-02**: F10 owns defining the shape of `docs/plan/product-delivery-roadmap.md` and `docs/plan/product-critical-path.md` (both currently absent everywhere), consistent with D01/D02's plain-markdown, human-authored convention. F10 does not extend the product-design D01–D14 arc to produce them.
- **D-F10-03**: The guard must degrade to reporting "prerequisite artifact missing / unresolved" rather than hard-blocking dispatch when D01, D02, the roadmap, or the critical-path file do not yet exist in a given project — consistent with REQ-F-033's required "unresolved prerequisites" reporting field, and necessary because none of the four sources exist in this repository today.
- **D-F10-04**: Preserve the existing boundary with E19-F10 verbatim: F10 stays advisory/prompt-layer; runtime admission/readiness enforcement remains E19-F10's DB-level responsibility. No shared Go surface is introduced between the two features.

## Sources

- `docs/plan/E34-prompt-and-skill-improvements/E34-F10-product-critical-path-guard-for-delivery-workflows/feature.md`
- `docs/plan/E34-prompt-and-skill-improvements/requirements.md` (Area 10)
- `docs/plan/E34-prompt-and-skill-improvements/epic.md`
- `docs/plan/E34-prompt-and-skill-improvements/architecture.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-interaction-map.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F07-state-space-planning-and-decision-propagation/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F09-override-drift-visibility-and-wwgm-reconciliation/research-report.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/feature.md`
- `docs/plan/E19-sprint-management-planning-system/E19-F10-roadmap-gated-sprint-admission-and-goal-acceptance/feature.md`
- `docs/plan/E36-project-layer-and-consult-bridge/E36-F02-project-namespace-and-progress-record/feature.md`
- `docs/product/progress.md`
- `docs/product/cross-epic-integration-map.md`
- `internal/sharkdata/default_data/prompts/sprint/{planning.md,active.md,closing.md}`
- `internal/sharkdata/default_data/prompts/epic/{assessment.md,decomposition.md,active.md}`
- `internal/sharkdata/default_data/prompts/feature/{specification.md,test_planning.md,task_generation.md,task_review.md,approval.md}`
- `internal/sharkdata/default_data/prompts/task/development.md`
- `internal/sharkdata/default_data/prompts/_partials/{_qa_process.md,_commands.md}`
- `internal/sharkdata/default_data/skills/product-design/workflows/{d01-vision.md,d02-success-criteria.md,d14-validated-designs.md}`
- `internal/sharkdata/default_data/research/recipes.yaml` (the recipe path this prompt cites as `shark-data/research/recipes.yaml` does not exist as a standalone path in this repository; it resolves to this embedded canonical copy)

RECOMMENDED OUTCOME: pass
