---
research_schema: 2
entity_key: E34-F03
entity_type: feature
recipe: universal
rigor: standard
categories:
  - workflow_operations
  - documentation
related_work: true
---

# E34-F03 Research Report: Deliverable Feature Decomposition and Staged Integration Acceptance

## Scope

E34-F03 extends Shark's existing prompt-and-skill policy contracts so an epic is
decomposed into independently demonstrable feature state transitions, while a
narrow, predeclared `contract-only` interaction may be staged until its named
activation owner proves the live production path. It does not add a workflow
engine, database model, entity type, automatic verdict downgrade, or a
replacement for independent UAT.

Terms used consistently in this report are: **live** (the current feature owns
a real trigger-to-observable-result path), **contract-only** (a predeclared
shared-contract obligation with a later activation owner), **activation owner**
(the feature whose UAT proves live wiring), **closure key** (tracked work that
closes the obligation), **assessor verdict** (the independent UAT result), and
**owner decision** (a separate approval or override fact).

The parent has no `research-report.md`; its applicable PRD/context is
`epic.md`, `requirements.md`, and `scope.md`. Complexity triage assigns
STANDARD rigor (13/27), so this report selects the required core modules plus
`pattern_contract`.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md`; also read
  `/home/jwwel/projects/wwgm/.worktrees/e04-complete-book-intake/dev-artifacts/2026-07-21-1600-halfway-feature-uat-retrospective/remediation-plan.md`
  (“Adopt the target hierarchy” and “Define staged integration acceptance”).
- [x] `affected_implementation_or_contract` — Evidence: `internal/sharkdata/default_data/prompts/epic/decomposition.md`; also read
  `internal/sharkdata/default_data/prompts/feature/specification.md`,
  `internal/sharkdata/default_data/prompts/feature/specification.md`,
  `internal/sharkdata/default_data/prompts/feature/task_review.md`,
  `internal/sharkdata/default_data/skills/quality/workflows/qa-testing.md`, and
  `internal/sharkdata/default_data/skills/uat/references/redteam-rubric.md`.
- [x] `related_work` — Evidence: `epic.md`, `requirements.md`,
  `E34-F01-harness-aware-prompt-rendering/feature.md`,
  `E34-F02-evidence-based-demo-script-skill/feature.md`, and the external
  remediation plan listed above.
- [x] `pattern_contract` — Evidence: `internal/sharkdata/default_data/prompts/epic/design.md`; also read
  `internal/sharkdata/default_data/skills/specification-writing/context/interaction-map-template.md`,
  `internal/sharkdata/default_data/skills/specification-writing/context/interaction-map-template.md`,
  and `internal/sharkdata/default_data/skills/quality/workflows/validate-design.md`.

## Capability map

| Capability | Evidence | Decision | E34-F03 responsibility |
|---|---|---|---|
| Epic interaction-map lifecycle (`I-##`) | `prompts/epic/design.md`; `prompts/epic/decomposition.md`; interaction-map template | EXTEND | Add gate-mode, counterpart, evidence, activation-owner, closure, and review-basis semantics without replacing the stable ID or shared-shape rules. |
| Feature specification and task-review cross-feature closure | `prompts/feature/specification.md`; `prompts/feature/task_review.md`; `skills/quality/workflows/validate-design.md` | EXTEND | Require the deliverability matrix and predeclared staged-edge disposition to be mirrored and checked before UAT. |
| QA/UAT wiring and verdict policy | `skills/quality/workflows/qa-testing.md`; `skills/uat/references/redteam-rubric.md`; `agents/uat-agent.md` | EXTEND | Keep missing live wiring, security, integrity, and unmet current ACs blocking; distinguish a UAT verdict from an owner decision and require closure evidence. |
| E34-F02 evidence-based demo | `E34-F02-evidence-based-demo-script-skill/feature.md` | EXTEND (producer contract) | Produce the readiness fields F02 consumes; do not implement demo generation or let demos act as acceptance authority. |
| E34-F01 harness-aware rendering | `E34-F01-harness-aware-prompt-rendering/feature.md` | REUSE | Keep prompt-rendering changes compatible with harness-aware variants; F03 does not change the handshake or renderer design. |
| WWGM remediation policy | external remediation plan cited in front matter | EXTEND | Generalize its deliverability and staged-integration rules into Shark's reusable workflow content; do not rewrite historic WWGM verdicts. |
| New runtime/persistence or a weaker UAT gate | `feature.md` Out of scope; `scope.md` | CONTRADICTS | Exclude it: this is a policy/content contract, and a future key cannot waive a present security, integrity, or live-path requirement. |

## Findings

1. Shark already has a propagation pattern for cross-feature contracts:
   `epic/design.md` creates the interaction map, `epic/decomposition.md`
   checks producer/consumer coverage, and feature specification, task review,
   design validation, and QA preserve I-## shape and shared-test pointers.
   E34-F03 should extend this established documentation/workflow contract,
   rather than create a parallel acceptance ledger.

2. The current interaction-map schema records producer, consumer, shape,
   payload, and style, but has no explicit acceptance mode or closure fields.
   The remediation plan supplies the minimum staged-edge fields: mode,
   counterpart keys/status, current evidence, activation owner, closure key,
   and review basis. These should be added consistently wherever the map is
   authored, copied, reviewed, and tested.

3. Existing UAT policy correctly treats missing call sites and unregistered
   components as blockers, and makes CRITICAL/HIGH findings Reject. E34-F03
   must preserve that rule for `live` edges and for security/integrity/current
   acceptance gaps. `contract-only` can only change the treatment of a complete
   predeclared contract with explicit approval and tracked closure; it cannot
   rewrite the independent assessor verdict.

4. E34-F02 is already explicitly dependent on this feature's vocabulary and
   must remain a read-only consumer of it. Its `Demonstrated now` category
   therefore needs the activation closure and assessor/owner facts from this
   feature; it must not invent a competing readiness interpretation.

5. E34 now has three features, so its existing 3+-feature interaction-map
   threshold applies. Design must create an I-## row from E34-F03 to E34-F02
   whose shared shape carries assessor verdict, owner decision, gate mode,
   activation owner, closure key, counterpart status, review basis, and
   demonstrability disposition.

## Decisions

1. **Extend the I-## workflow contract, do not add persistence.** Use the
   existing interaction map as the authoritative design-time artifact and
   register it through the existing related-document convention.

2. **Make live the default.** An undeclared interaction remains live and a
   missing production caller is blocking. A feature boundary that cannot name
   a real trigger, observable result, production path, complete UAT scenario,
   and current prerequisites should be merged or redesigned before build.

3. **Constrain contract-only.** It must be declared by specification, confirmed
   at task review, have a named later counterpart and activation owner, shared
   contract evidence, closure key, counterpart status, and explicit review
   basis. Reverse build-order consumption is a decomposition warning.

4. **Preserve gate integrity.** Neither a later Shark key nor `override-accept`
   makes missing authentication, authorization, integrity protection, unsafe
   exposure, a missing live path, or an unmet current-feature AC non-blocking.
   Store/report assessor verdict and owner decision as separate facts.

5. **Close downstream wiring.** The activation owner's UAT must prove the real
   caller chain, production-path integration test, and counterfactual failure
   when wiring is bypassed or removed. Epic completion cannot leave an internal
   activation obligation unresolved.

## Sources

- `docs/plan/E34-prompt-and-skill-improvements/E34-F03-deliverable-feature-decomposition-and-staged-integ/feature.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F03-deliverable-feature-decomposition-and-staged-integ/complexity-triage.md`
- `docs/plan/E34-prompt-and-skill-improvements/epic.md`, `requirements.md`, and `scope.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F01-harness-aware-prompt-rendering/feature.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/feature.md`
- `internal/sharkdata/default_data/prompts/epic/design.md` and `decomposition.md`
- `internal/sharkdata/default_data/prompts/feature/specification.md` and `task_review.md`
- `internal/sharkdata/default_data/skills/specification-writing/context/interaction-map-template.md`
- `internal/sharkdata/default_data/skills/quality/workflows/validate-design.md` and `qa-testing.md`
- `internal/sharkdata/default_data/skills/uat/references/redteam-rubric.md`
- `/home/jwwel/projects/wwgm/.worktrees/e04-complete-book-intake/dev-artifacts/2026-07-21-1600-halfway-feature-uat-retrospective/remediation-plan.md`

RECOMMENDED OUTCOME: standard
