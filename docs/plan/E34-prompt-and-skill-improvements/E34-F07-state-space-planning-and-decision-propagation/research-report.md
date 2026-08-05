---
research_schema: 2
entity_key: E34-F07
entity_type: feature
recipe: universal
rigor: complex
categories:
  - workflow_operations
  - documentation
related_work: true
---

# E34-F07 Research Report: State-Space Planning and Decision Propagation

## Scope

E34-F07 strengthens planning and decision-resolution content. It covers closed
lifecycle tables, state-aware test technique selection, interaction/caller-path
consumer discovery, shared naming, shipped-consumer re-verification, and
affected-artifact propagation. It does not implement application state
machines or add a Shark dependency store.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `E34-F07-state-space-planning-and-decision-propagation/feature.md`; `internal/sharkdata/default_data/prompts/feature/{specification.md,test_planning.md,task_review.md}`; `docs/plan/E34-prompt-and-skill-improvements/E34-interaction-map.md`.
- [x] `affected_implementation_or_contract` — Evidence: the listed feature prompts; `internal/sharkdata/default_data/skills/quality/context/test-design-techniques.md`; `internal/sharkdata/default_data/prompts/tech_debt/resolved.md`.
- [x] `related_work` — Evidence: E34-F03 staged integration feature/research/test plan; E34-F04 Question adoption; E39 Question resolution provenance; WWGM B023 retrospective and E04 proposal gaps D and F.
- [x] `pattern_contract` — Evidence: E34 I-## interaction map, caller-path contract language in feature test planning, and Question resolution pointers to affected references.
- [x] `dependency_impact` — Evidence: state producers affect persistence readers, deduplication and short-circuit consumers, task mirrors, interaction maps, ACs, and regression plans across feature boundaries.
- [x] `cross_boundary_risks` — Evidence: prose lifecycles and direct-dependency-only searches can omit failure states or non-FK consumers; decision resolution can make durable documents contradict running code.
- [x] `alternatives` — Evidence: the E04 proposal suggested a one-FK-hop rule. Interaction maps and production caller paths cover APIs, events, files, services, and cross-epic contracts that a database-only heuristic misses.

## Capability map

| Capability | Evidence | Decision | E34-F07 responsibility |
|---|---|---|---|
| State-transition and decision-table techniques | quality test-design guidance | EXTEND | Make technique selection mandatory when the behavior shape requires it. |
| I-## interaction lifecycle | E34-F03 and E34 interaction map | REUSE | Use declared producer/consumer contracts as dependency-discovery input. |
| Caller-path contract review | feature test-planning content | REUSE | Trace real production consumers beyond stored entity links. |
| Task contract-name validation | `feature/task_review.md` | EXTEND | Compare shared names verbatim and report unexplained drift. |
| Typed decision provenance | E39 Question and `tech_debt/resolved.md` | EXTEND | Require affected-artifact and consumer-AC propagation after resolution. |
| ChangeImpactSet v1 | No canonical shape exists | NEW | Provide one bounded amendment/follow-up accounting payload. |

## Findings

1. The needed techniques already exist as optional methodology. The gap is the
   trigger and exit gate: planning can choose a simpler technique even when a
   behavior-bearing state field makes transition coverage necessary.

2. A one-FK-hop heuristic is too specific to WWGM and too narrow for Shark's
   generic workflows. Interaction contracts and caller paths express the real
   consumer boundary across storage, APIs, events, commands, and files.

3. State additions have backward impact. Planning must reopen the acceptance
   surface of already implemented consumers instead of assuming their old tests
   still cover the enlarged value set.

4. Shared-name drift is a low-cost signal of an unreviewed contract fork and
   belongs in task review before implementation.

5. Durable decision resolution is incomplete when authoritative planning and
   test documents remain stale. I-04 should require amendment or explicit
   follow-up accounting without pretending an automated rewrite is safe.

## Decisions

1. Require closed tables for behavior-bearing lifecycle/disposition fields.
2. Trigger state-transition testing from the contract shape.
3. Discover consumers through interactions and caller paths, not FK distance.
4. Reverify shipped consumer ACs when their state input expands.
5. Require I-04 for material decision resolution and justified design
   divergence.

## Sources

- `internal/sharkdata/default_data/prompts/feature/{specification.md,test_planning.md,task_review.md}`
- `internal/sharkdata/default_data/prompts/epic/feature_review.md`
- `internal/sharkdata/default_data/prompts/tech_debt/resolved.md`
- `internal/sharkdata/default_data/skills/quality/context/test-design-techniques.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-interaction-map.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F03-deliverable-feature-decomposition-and-staged-integ/`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F04-question-surface-adoption-for-design-and-decision/`
- WWGM `docs/plan/E04-upload-source-ledger/DB-2026-07-25-b023-spec-process-retrospective.md`
- WWGM `dev-artifacts/2026-08-04-1530-e04-review-gap-analysis/PROPOSAL.md`
