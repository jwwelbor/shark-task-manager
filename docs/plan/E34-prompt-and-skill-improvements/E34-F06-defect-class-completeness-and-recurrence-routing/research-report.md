---
research_schema: 2
rigor: standard
categories:
  - workflow_operations
  - documentation
related_work: true
---

# E34-F06 Research Report: Defect-Class Completeness and Recurrence Routing

## Scope

E34-F06 adds one reusable defect-class-sweep workflow to the embedded quality
bundle, replaces duplicated sweep language across review/QA/UAT/development
prompts with references to it, and adds evidence-based recurrence and
severity-conflict routing through the existing E39 Question and E38 council
mechanisms. It consumes E34-F05's **I-02 GateResult v1** persistence and
produces the **I-03 DefectClassSweep v1** shape already specified in
`architecture.md`. It introduces no new persistence layer, retry counter, or
lifecycle engine (REQ-NF-001) — content and prompt-wiring only.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/E34-F06-defect-class-completeness-and-recurrence-routing/feature.md`; `docs/plan/E34-prompt-and-skill-improvements/architecture.md#i-03-defectclasssweep-v1`. These define the defect-class vocabulary (`class_key`, `class_statement`, `search_scope`, counts, `guard`, `status`), the completeness invariant (`matching_count = fixed_count + dispositioned_count`, `open_count = 0`, guard `verified`), and the recurrence rule (fingerprint repeat, or matching `class_key` **and**
in-scope site — both conjuncts required).
- [x] `affected_implementation_or_contract` — Evidence: `internal/sharkdata/default_data/prompts/feature/code_review.md` (line 81, kickback reason template already names "defect-class statement" and "sweep the touched module(s)"); `internal/sharkdata/default_data/prompts/feature/approval.md` (line 54, identical duplicated template); `internal/sharkdata/default_data/skills/uat/references/redteam-rubric.md` (lines 32, 48, "Defect-class sweep" step in the red-team rubric). These three files currently hold independent, duplicated prose describing the same sweep concept this feature must consolidate into one bundle workflow (REQ-F-001/REQ-F-002), confirming the "affected" surface named in the feature's own Implementation plan step 2.
- [x] `pattern_contract` — Evidence: `internal/sharkdata/default_data/skills/quality/` workflow directory (`review-code.md`, `qa-testing.md`, `validate-tasks.md`, `validate-design.md`, `test-planning.md`, `generate-standards.md`) shows the established pattern for a bundle-local quality workflow file plus a `context/` reference doc (e.g. `quality-gates.md`, `review-rubric.md`, `caller-path-contracts.md`). The new defect-class-sweep workflow should follow this same `workflows/<name>.md` + optional `context/<reference>.md` shape rather than inventing a new bundle layout. No file at `skills/quality/workflows/` or `skills/quality/context/` currently names "defect" or "sweep" — confirmed via targeted search — so this is a NEW workflow file, not an edit to an existing one.
- [x] `related_work` — Evidence: E34-F05 (`docs/plan/E34-prompt-and-skill-improvements/E34-F05-structured-gate-results-and-parent-owned-persistence/`, completed) defines and implements **I-02 GateResult v1**, the persistence contract this feature's I-03 sweeps nest inside (`remediation_sweeps: array of I-03` per `architecture.md` line 164). E34-F08 ("Tier-Consistent Gates and Final Integration Review") is the declared downstream consumer of I-03. E38 (Shark Attack council) and E39 (Question lifecycle) are the existing escalation mechanisms REQ-F-006 must route through rather than duplicate — confirmed present via `skills/uat/references/redteam-rubric.md` and epic-level cross-references; this feature adds no new escalation primitive.

## Capability map

| Capability | Evidence | Decision | E34-F06 responsibility |
|---|---|---|---|
| GateResult / I-02 persistence | E34-F05 (completed) | REUSE | Nest I-03 sweeps inside the existing `remediation_sweeps` field; no new persistence |
| Defect-class-sweep workflow content | No `skills/quality/workflows/*defect*` or `*sweep*` file exists | NEW | Add one `skills/quality/workflows/defect-class-sweep.md` (or equivalent) per the established quality-workflow file pattern |
| Sibling-sweep prose in `code_review.md` / `approval.md` | Lines 81/54, near-identical duplicated kickback-reason template | EXTEND (consolidate) | Replace with a reference to the new workflow plus gate-specific inputs, per REQ-F-001 |
| Red-team "Defect-class sweep" step (`redteam-rubric.md`) | Lines 32/48 | EXTEND (consolidate) | Same consolidation, UAT-side |
| Recurrence / severity-conflict routing | E38 council, E39 Question (both pre-existing, out of this feature's scope to build) | REUSE | Route conflicts into these mechanisms per REQ-F-006; do not build new escalation logic |
| I-03 DefectClassSweep v1 shape | `architecture.md#i-03-defectclasssweep-v1` (fully specified, no Go implementation — this is bundle-content-only work) | REUSE (contract), NEW (workflow that produces it) | Implement the workflow that emits this shape; the shape itself is not renegotiated |

No CONTRADICTS findings — the feature's own plan already matches the discovered bundle structure and existing dependencies.

## Findings

- The "sibling sweep" and "defect-class sweep" concepts already exist as **duplicated prose** in at least three locations (`code_review.md`, `approval.md`, `redteam-rubric.md`), exactly matching the problem statement in `feature.md`. This confirms the consolidation work is real and scoped correctly, not speculative.
- No Go code implements `DefectClassSweep`, `GateResult`, or `class_key` anywhere under `internal/` — this is consistent with E34-F05 also being bundle/content-only work (structured result *schema and prompt wiring*, not a new Go persistence layer), and confirms E34-F06 is the same kind of work: prose, workflow files, and bundle manifest/index registration.
- The quality skill bundle has an established two-tier pattern (`workflows/*.md` for procedures, `context/*.md` for reference material) that the new defect-class-sweep content should follow for consistency with `review-code.md`, `qa-testing.md`, etc.
- The prior stale `research-report.md` (superseded by this one) had tagged `rigor: complex`; the fresh complexity triage (see feature notes, 2026-09-04) scored this STANDARD (14/27) — largely because the I-03 data-model design work was already completed by E34-F05, leaving F06 as workflow-content-and-wiring rather than novel-schema design.

## Decisions

- Follow the existing `skills/quality/workflows/` + `context/` file pattern for the new defect-class-sweep content (no new bundle directory structure).
- Consolidate all three duplicated sweep-prose sites into references to the single new workflow, per REQ-F-001/REQ-F-002 — do not leave any of the three as freestanding duplicated text.
- No new Go code, persistence type, or escalation primitive — REQ-NF-001 is satisfied by routing recurrence/severity-conflict decisions through the existing E38/E39 mechanisms.

## Sources

- `docs/plan/E34-prompt-and-skill-improvements/E34-F06-defect-class-completeness-and-recurrence-routing/feature.md`
- `docs/plan/E34-prompt-and-skill-improvements/architecture.md` (§I-02 GateResult v1, §I-03 DefectClassSweep v1)
- `internal/sharkdata/default_data/prompts/feature/code_review.md`
- `internal/sharkdata/default_data/prompts/feature/approval.md`
- `internal/sharkdata/default_data/skills/uat/references/redteam-rubric.md`
- `internal/sharkdata/default_data/skills/quality/workflows/` (directory listing, pattern precedent)
- E34-F06 feature notes (complexity triage decision, 2026-09-04)

RECOMMENDED OUTCOME: standard
