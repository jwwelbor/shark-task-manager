---
research_schema: 2
entity_key: E34-F08
entity_type: feature
recipe: universal
rigor: complex
categories:
  - workflow_operations
  - documentation
related_work: true
---

# E34-F08 Research Report: Tier-Consistent Gates and Final Integration Review

## Scope

E34-F08 aligns planning artifacts and quality gates for all three feature
complexity tiers, defines executable gate evidence, and adds a final epic
integration review over the accumulated change. It does not add project-local
validation scripts or wait for E40 benchmark completion.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `E34-F08-tier-consistent-gates-and-final-integration-review/feature.md`; `internal/sharkdata/default_data/workflow/{feature.yaml,epic.yaml}`; prompts `feature/{assessment.md,task_generation.md,code_review.md,qa.md,approval.md}`.
- [x] `affected_implementation_or_contract` — Evidence: canonical feature/epic workflows; review prompts; quality review workflow; prompt renderer and workflow validation tests.
- [x] `related_work` — Evidence: E34-F03 staged-integration packet; E34-F05 through F07 plans; E40 epic/design documents; WWGM E04 proposal and local overrides; CC-007 and CC-008 references in that proposal.
- [x] `pattern_contract` — Evidence: route-based YAML outcomes; canonical prompt include/augment resolution; exact production prompt rendering; merge-base diff instructions in current gates.
- [x] `dependency_impact` — Evidence: tier selection changes planning artifacts and gate routes; final epic review consumes every feature gate, interaction, decision, finding, and accumulated changed path.
- [x] `cross_boundary_risks` — Evidence: duplicated tier text can make gates require nonexistent artifacts; model-written test totals can diverge from runner output; a final PASS can be misused to mask an unresolved feature verdict if authority is not explicit.
- [x] `alternatives` — Evidence: the E04 proposal suggested a whole-diff step after repeated rounds. Always running one epic integration review is deterministic, avoids retry-count policy, and covers even one-round defects introduced by earlier features.

## Capability map

| Capability | Evidence | Decision | E34-F08 responsibility |
|---|---|---|---|
| Complexity-tier routing | canonical `workflow/feature.yaml` and feature prompts | EXTEND | Centralize the artifact/gate matrix and test every route. |
| Combined review/QA for SIMPLE and STANDARD | `feature/code_review.md` and workflow YAML | REUSE | Preserve the route and make its evidence standard explicit. |
| Separate QA for COMPLEX | `feature/qa.md` and workflow YAML | REUSE | Preserve deeper gate division without treating it as the only rigorous tier. |
| SIMPLE-lite prompt support | WWGM overrides | PROMOTE | Upstream the reusable portions through a shared tier reference. |
| Exact command execution | Current prompts require commands but accept summaries | EXTEND | Require runner-native exit/count/skip evidence and bounded pointers. |
| Epic final integration review | No canonical step exists | NEW | Add a route-based step with accumulated-diff scope and non-supersession authority. |
| Staged edge and disposition handling | E34-F03 plus WWGM amendments | PROMOTE | Consume the canonical declarations and E34-F06 disposition rule. |
| Benchmark evaluation | E40 design is underway | DEFER VALIDATION | Record scenarios without making E40 a delivery prerequisite. |

## Findings

1. The tier route already exists, but its policy is spread across prompts. The
   WWGM fix proves the value of tier-aware instructions and the danger of
   carrying them as full replacement overrides.

2. “Run tests” is not enough evidence. A reliable generic contract captures
   the command and the test runner's own result fields while letting each
   project define its commands and environment.

3. Per-feature merge-base review still cannot guarantee whole-epic closure
   when features land or are repaired at different times. A named final epic
   step is the right stable authority boundary.

4. The final step must be additive. E04's shipped-while-rejected history shows
   that an undocumented supersession rule damages auditability even if a
   whole-diff review is technically sound.

5. E40 is useful for measuring later behavior, but its current active planning
   and implementation state makes it an inappropriate prerequisite.

## Decisions

1. Centralize the exact SIMPLE/STANDARD/COMPLEX matrix.
2. Require executable evidence without embedding project commands.
3. Always run one epic integration review before completion.
4. Make final review additive and forbid silent feature-verdict supersession.
5. Produce I-05 so override adopters can reconcile deliberately.
6. Use E40 for later benchmark scenarios, not the acceptance gate.

## Sources

- `internal/sharkdata/default_data/workflow/{feature.yaml,epic.yaml}`
- `internal/sharkdata/default_data/prompts/feature/{assessment.md,task_generation.md,code_review.md,qa.md,approval.md}`
- `internal/sharkdata/default_data/skills/quality/workflows/review-code.md`
- `internal/sharkdata/{embed.go,embed_test.go}`
- `internal/cli/commands/next_golden_test.go`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F03-deliverable-feature-decomposition-and-staged-integ/`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/{epic.md,shark-bench-design.md}`
- WWGM `shark-data/overrides/`
- WWGM `dev-artifacts/2026-08-04-1530-e04-review-gap-analysis/PROPOSAL.md`
