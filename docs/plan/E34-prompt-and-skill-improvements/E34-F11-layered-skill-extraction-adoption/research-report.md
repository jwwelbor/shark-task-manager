---
research_schema: 2
rigor: simple
categories:
  - documentation
related_work: true
---

# E34-F11 Research Report: Layered Skill Extraction Adoption

## Scope

E34-F11 exists solely to give
`dev-artifacts/planning/skill-workflow-extraction-prompt.md` (dated
2026-06-22) a tracked Shark owner and repository-path record, closing item 2
of decision `D-E34-LEGACY-PROMPTS-001`. It covers verifying the artifact is
current, confirming a discoverability pointer exists outside `dev-artifacts/`,
and confirming the task/feature record the tracked path. It does not cover
authoring new prompt content, revising the artifact itself, or any product
code change.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/E34-F11-layered-skill-extraction-adoption/feature.md`; `docs/plan/E34-prompt-and-skill-improvements/E34-F11-layered-skill-extraction-adoption/tasks/T-E34-F11-001.md`; `dev-artifacts/planning/skill-workflow-extraction-prompt.md`. These sources define the tracked artifact, the "layered extraction" (workflow/prompt/methodology/reference) vocabulary, and the D-E34-LEGACY-PROMPTS-001/REQ-F-006 scope this feature closes.
- [x] `affected_implementation_or_contract` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/epic.md` (decision-note prose referencing `D-E34-LEGACY-PROMPTS-001` and `E34-F11`/`T-E34-F11-001`); `docs/plan/E34-prompt-and-skill-improvements/requirements.md` REQ-F-006. Both already state the artifact is "TRACKED" and owned by this feature/task — the "affected contract" here is documentation record-keeping, not code.
- [x] `related_work` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/epic.md` D-E34-LEGACY-PROMPTS-001 note; `docs/plan/E34-prompt-and-skill-improvements/requirements.md` REQ-F-006; `docs/plan/E34-prompt-and-skill-improvements/E34-F11-layered-skill-extraction-adoption/tasks/T-E34-F11-001.md`.

## Capability map

| Capability | Evidence | Decision | E34-F11 responsibility |
|---|---|---|---|
| skill-workflow-extraction prompt content | `dev-artifacts/planning/skill-workflow-extraction-prompt.md` | REUSE | Adopt as-is; no new prompt content authored (matches feature's explicit "Out of Scope" statement). |
| D-E34-LEGACY-PROMPTS-001 resolution record | `epic.md` decision-note prose | REUSE | Cite the existing resolution rather than re-deciding it; the epic already records item 2 as TRACKED and owned by E34-F11/T-E34-F11-001. |
| REQ-F-006 tracked-reference requirement | `requirements.md` REQ-F-006 | REUSE | The requirement's acceptance language ("owned by E34-F11", "no new prompt content authored") is already satisfied by the existing epic and task records. |
| Discoverability pointer outside `dev-artifacts/` | `epic.md` feature-portfolio/decision text; `requirements.md` REQ-F-006 text | EXTEND (verification only) | Confirm the pointer exists in both places; the feature's own AC2 checkbox is unchecked even though the pointer is present — see Findings. |

## Findings

1. `dev-artifacts/planning/skill-workflow-extraction-prompt.md` exists,
   is dated 2026-06-22, and its content (workflow / prompt / methodology /
   reference separation) matches REQ-F-006's layered-extraction concept
   verbatim — confirmed by reading the file directly.
2. The artifact already has a tracked Shark owner: `T-E34-F11-001` names the
   exact repository path in both its task file and its DB description, and
   every requirement/deliverable/acceptance-criteria checkbox on that task is
   marked complete.
3. A discoverability pointer outside `dev-artifacts/` already exists in two
   places: `docs/plan/E34-prompt-and-skill-improvements/epic.md` (decision
   note referencing `D-E34-LEGACY-PROMPTS-001`) and
   `docs/plan/E34-prompt-and-skill-improvements/requirements.md` (REQ-F-006),
   both of which name `E34-F11`/`T-E34-F11-001` as the owner and state "no new
   prompt content was authored."
4. `D-E34-LEGACY-PROMPTS-001` is already marked "RESOLVED 2026-08-31" in
   `epic.md`, with item 2 (this artifact) explicitly TRACKED and item 1 (the
   unrelated "earlier ignored dev-artifact review prompt") explicitly
   CANCELLED after an exhaustive repo search.
5. Discrepancy: `feature.md`'s own Story 1 AC2 ("A short discoverability
   pointer to the artifact exists outside `dev-artifacts/`") and REQ-F-001's
   acceptance criteria are still shown unchecked in the feature file, even
   though the pointer they describe already exists in `epic.md` and
   `requirements.md`. This is a stale-checkbox gap in `feature.md` itself,
   not missing work — closing it is a documentation-sync edit, not new scope.
6. No code, schema, or CLI contract is implicated anywhere in this feature;
   every artifact touched is Markdown planning/tracking content.

## Decisions

1. Treat E34-F11 as a verification/close-out feature: the tracked path,
   decision-note record, and discoverability pointer already exist and were
   independently confirmed, not newly created by this research pass.
2. Do not author or revise any prompt content — the existing artifact is
   final per the feature's own "Out of Scope" section.
3. Flag the stale unchecked boxes in `feature.md` (Story 1 AC2, REQ-F-001
   acceptance criteria) as a small documentation-sync item for the
   specification/task-generation phase to close out, rather than as new
   research scope.
4. No capability is EXTENDed or created by code; the only capability change
   is administrative (ownership/discoverability of an existing artifact),
   which is already REUSE-complete.

## Sources

- `docs/plan/E34-prompt-and-skill-improvements/E34-F11-layered-skill-extraction-adoption/feature.md`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F11-layered-skill-extraction-adoption/tasks/T-E34-F11-001.md`
- `dev-artifacts/planning/skill-workflow-extraction-prompt.md`
- `docs/plan/E34-prompt-and-skill-improvements/epic.md` (D-E34-LEGACY-PROMPTS-001 note)
- `docs/plan/E34-prompt-and-skill-improvements/requirements.md` (REQ-F-006)

RECOMMENDED OUTCOME: simple
