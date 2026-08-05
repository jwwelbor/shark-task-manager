---
research_schema: 2
entity_key: E34-F06
entity_type: feature
recipe: universal
rigor: complex
categories:
  - workflow_operations
  - documentation
related_work: true
---

# E34-F06 Research Report: Defect-Class Completeness and Recurrence Routing

## Scope

E34-F06 standardizes how Shark quality and development workflows define,
enumerate, repair, guard, re-verify, and route a defect class. It promotes
general lessons from WWGM without embedding WWGM rules. It relies on E34-F05
for durable structured results and on existing Question/council capabilities
for material conflicts.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `E34-F06-defect-class-completeness-and-recurrence-routing/feature.md`; `internal/sharkdata/default_data/skills/quality/workflows/review-code.md`; `internal/sharkdata/default_data/skills/uat/references/redteam-rubric.md`. These sources define finding, defect class, enumeration, re-verification, disposition, and blocking severity.
- [x] `affected_implementation_or_contract` — Evidence: `internal/sharkdata/default_data/prompts/task/development.md`; prompts `feature/{code_review.md,qa.md,approval.md}`; quality and UAT skills. Current behavior is duplicated and lacks a completed-sweep/guard contract.
- [x] `related_work` — Evidence: E34-F04 Question adoption; E38 Shark Attack council content; E39 Question workflow; WWGM E04 proposal, inventory, approval override, and red-team-rubric override.
- [x] `pattern_contract` — Evidence: `internal/sharkdata/default_data/skills/shark-attack/workflows/{route-question.md,council.md}`; `skills/question-management/SKILL.md`; host `defect-class-sweep` skill. The host procedure is useful methodology, while the embedded bundle needs its own registered canonical workflow.
- [x] `dependency_impact` — Evidence: review workers produce findings, development consumes kickbacks, later gates consume prior findings and decisions, and E34-F08 consumes closure evidence.
- [x] `cross_boundary_risks` — Evidence: class identity must remain stable across prompts, notes, rework sessions, Questions, and final review. Loose prose or copied policies can misclassify new evidence as recurrence or re-litigate accepted risk.
- [x] `alternatives` — Evidence: proposal P2 suggested round-count escalation. E04 rounds correlate with failure but cannot distinguish a new class from a failed completed sweep; durable class evidence is the safer general trigger.

## Capability map

| Capability | Evidence | Decision | E34-F06 responsibility |
|---|---|---|---|
| Enumerate-not-iterate review stance | UAT red-team rubric | REUSE | Keep full enumeration and make its output structured and countable. |
| Touched-module sibling sweep | `prompts/task/development.md` | EXTEND | Add prior-record lookup, declared scope, guard closure, and result evidence. |
| Re-verification three-part pass | approval prompt and UAT rubric | REUSE | Preserve named-fix, class-sweep, and full-rubric checks in one shared workflow. |
| Already-dispositioned finding handling | WWGM approval/rubric overrides | PROMOTE | Generalize upstream with durable-decision and new-evidence safeguards. |
| Question and council routing | E39 and E38 embedded skills | REUSE | Route unresolved severity/materiality conflicts; add no competing process. |
| DefectClassSweep v1 | No canonical bundle contract exists | NEW | Define stable identity, scope, counts, instances, guard, and verification evidence. |

## Findings

1. Existing canonical content contains most of the right review stance but no
   single closure contract. Duplication makes it possible for development to
   interpret “sweep” more narrowly than UAT.

2. WWGM's already-dispositioned logic is broadly reusable, but its whole-file
   override now hides newer upstream changes. The policy should be promoted as
   a shared fragment/workflow, then the override removed or reduced.

3. A retry number is not defect evidence. The reliable signal is a fingerprint
   or same-class instance inside a previously asserted complete sweep. This
   also makes failures actionable: repair the scope or guard that proved false.

4. Structural guards turn a review lesson into an executable prevention
   mechanism. When none is feasible, a durable follow-up is more truthful than
   a “closed” class that reviewers must remember manually.

5. Severity conflicts are decision conflicts, not ordinary task rework. E39
   and Shark Attack already supply the needed durable routing and should remain
   the sole escalation mechanisms.

## Decisions

1. Add one embedded defect-class workflow referenced by all producers and
   consumers.
2. Persist I-03 through GateResult and typed notes; add no new data store.
3. Classify recurrence from completed-sweep evidence, never round count alone.
4. Promote reusable WWGM disposition semantics without project-specific names.
5. Route unresolved material conflicts through the existing Question/council
   boundary.

## Sources

- `internal/sharkdata/default_data/prompts/feature/{code_review.md,qa.md,approval.md}`
- `internal/sharkdata/default_data/prompts/task/development.md`
- `internal/sharkdata/default_data/skills/quality/workflows/review-code.md`
- `internal/sharkdata/default_data/skills/uat/references/redteam-rubric.md`
- `internal/sharkdata/default_data/skills/shark-attack/workflows/{route-question.md,council.md}`
- `internal/sharkdata/default_data/skills/question-management/SKILL.md`
- WWGM `shark-data/overrides/prompts/feature/approval.md`
- WWGM `shark-data/overrides/skills/uat/references/redteam-rubric.md`
- WWGM `dev-artifacts/2026-08-04-1530-e04-review-gap-analysis/{PROPOSAL.md,INVENTORY.md}`
