---
inputs:
  - entity_kind: feature | epic — the kind of entity whose related work is being consulted
  - entity_key: opaque identifier of the current entity (e.g., feature key, epic key)
  - entity_purpose: one-sentence description of the capability this entity provides
  - parent_key: opaque identifier of the parent entity (epic key when entity_kind=feature; null when entity_kind=epic)
  - domain_terms: list of domain-specific search terms (e.g., for a PDF feature: "pdf", "docling", "import", "source-block")
  - sibling_entities: list of {key, title, related_doc_paths} — siblings supplied by the wrapper (for a feature, all sibling features in the parent epic; for an epic, related epics the wrapper has identified)
  - related_epics: list of {key, title, related_doc_paths} — for cross-epic awareness (may be empty)
  - search_results: list of {term, hits: [{entity_key, snippet, source}]} — pre-run domain-term search results from the wrapper's search backend
  - filesystem_adr_globs: list of glob patterns to scan for ADRs not registered in the wrapper's index (e.g., `docs/**/adr/**/*.md`)
  - report_path: path to the entity's unified research report
outputs:
  - inspection_set: list of {key, doc_paths_read} — every entity inspected
  - capability_map: list of {capability, established_in, doc_path, decision: REUSE | EXTEND | RE-IMPLEMENT | CONTRADICTS, justification}
  - reconciliations_needed: list of {capability, contradiction, surfaced_to_user}
  - reuse_links: list of {sibling_key, link_kind: depends_on | related_to} — relationships the wrapper should create
  - open_questions: list of unresolved questions surfaced to the user/BA/architect
  - will_not_reimplement: list of capability names this entity will explicitly NOT re-implement
  - research_report_update: capability-map content written to report_path
---

# Workflow: Consult Related Work

**Purpose**: Mechanically discover and read prior art (sibling features, related epics, ADRs, prior decisions) before starting any new feature, epic refinement, or task spec — so we don't rebuild what's already been built or contradict prior decisions.

**Use for**: Mandatory first step before refining or designing a new feature/epic. Also called as the input-gathering Step 1 by epic-tech-plan and feature-tech-plan workflows.

**Estimated time**: 10–20 minutes (scales with epic size).

**Output**: A Capability map section in the entity's unified `research-report.md`, with explicit reuse, extension, or new-work decisions per capability.

## Why This Workflow Exists

Without a mechanical procedure, an architect or BA starting a new feature in an existing epic will not look at sibling features, even if a sibling established the canonical pipeline the new feature should reuse. "Be aware of related work" is not enforceable — a procedure that lists siblings and a required report section that names each one is enforceable. This workflow is the latter.

**Failure mode this prevents**: Sibling feature established a canonical approach (e.g., "docling is the PDF import pipeline; produces SourceBlock/SourcePage"). New feature in the same epic re-implements the same capability from scratch because nobody read the sibling's architecture doc. By the time it's caught at code review, weeks are wasted.

## Required Tools

- **Read** — Read each architecture/ADR/decision-record found
- **Grep** — Search filesystem for ADRs that may not be in the wrapper's index

## Inputs

The wrapper supplies:

1. The current entity's identifier (`entity_key`) and kind (`entity_kind`).
2. A one-sentence purpose summary (`entity_purpose`).
3. Domain terms relevant to the work (`domain_terms`) — e.g., for a PDF feature: `pdf`, `docling`, `pdfminer`, `import`, `source-block`. If the wrapper does not have these yet, it derives them from the parent epic's PRD/architecture before invoking this workflow — those terms drive the search step.
4. The list of `sibling_entities` (with each sibling's already-discovered related-doc paths).
5. Pre-run `search_results` for the domain terms from the wrapper's search backend.

This workflow is the related-work module of the unified research recipe. The mechanics of enumerating siblings, indexing entity metadata, and registering the entity's report live with the host's project-state machine.

## Procedure

### Step 1: Enumerate the inspection set

The wrapper has already provided `sibling_entities` and `related_epics`. Combine them with any ADRs / decision records found in Step 3 to produce the **inspection set** — every artifact you will read.

Record the inspection set so it can be cited in the report.

### Step 2: Identify priority docs per sibling

For each entity in your inspection set, prioritize which of its registered docs to read, in this order:

1. The architecture document (typically `02-architecture.md` or named equivalent).
2. The API / backend design (typically `04-api-specification.md` / `04-backend-design.md`).
3. The data design (typically `03-data-design.md` / `03-database-design.md`).
4. Anything titled "ADR" or stored under `adr/` or `decisions/`.
5. The sibling's PRD (`feature.md` / `prd.md`).

Build a flat list of every doc path you intend to read. Skip anything the sibling does not have — siblings without architecture docs are read at PRD-level only.

### Step 3: Search for ADRs not in the host index

For each domain term, the wrapper has supplied `search_results` from its primary search backend. Augment that with a filesystem ADR scan (some ADRs are committed but not registered):

```bash
# For each glob in filesystem_adr_globs:
find docs -type d -iname "adr" -o -iname "decisions" -o -iname "architecture-decisions" 2>/dev/null
grep -rln "^# ADR" docs/ 2>/dev/null
grep -rln "## Decision" docs/plan/ 2>/dev/null
```

Add anything found to the inspection set.

### Step 4: Read every artifact in the inspection set

This is the part that cannot be skipped. For each doc on your list:

1. Read it.
2. For each capability it establishes (data model, pipeline, API contract, ADR), record one entry in your capability map:
   - Capability name
   - Sibling/source key
   - Doc path
   - Decision: **REUSE** / **EXTEND** / **RE-IMPLEMENT (with reason)** / **CONTRADICTS — needs reconciliation**

If you mark anything **RE-IMPLEMENT**, you must justify why reuse is not viable. "Didn't see it" is not a reason — at this point you have read it.

If you mark anything **CONTRADICTS**, stop and surface to the user before continuing. Two pieces of architecture in the same epic disagreeing is a refinement bug that won't fix itself.

### Step 5: Add the related-work findings to the unified research report

Write the report to `report_path`. Use this structure (keep it under ~120 lines):

```markdown
# Research Report: <entity-key> — <one-sentence purpose>

**Date**: YYYY-MM-DD
**Author**: <agent or workflow that produced this>
**Inspection set size**: N siblings, M related epics, K ADRs

## Inspection Set

### Sibling features in <parent>
- `<sibling-key>` <title> — read: 02-architecture.md, 04-backend-design.md
- `<sibling-key>` <title> — read: 02-architecture.md
- `<sibling-key>` <title> — no architecture doc; read PRD only

### Related epics
- `<epic-key>` <title> — read: 02-architecture.md

### ADRs / decision records
- `docs/.../adr-001-pdf-pipeline.md` — established docling as the canonical PDF parser

### Domain-term search hits (terms: <list>)
- "docling" → 3 hits: <key> (PRD), <key> (note: implementation), <key> (related_to)

## Capability Map

| Capability | Established in | Doc | Decision |
|---|---|---|---|
| PDF parsing pipeline | <key> | <key>/02-architecture.md | **REUSE** — call docling adapter |
| SourceBlock data model | <key> | <key>/03-data-design.md | **REUSE** — import the model |
| Per-page metadata extraction | (none — gap) | — | NEW — owned by this feature |
| OCR fallback | <key> | <key>/04-backend-design.md | **EXTEND** — add image-only-page detection |

## Reconciliations Needed

- (List anything marked CONTRADICTS in Step 4. If empty: "None — no contradictions found.")

## Open Questions Surfaced

- (Anything you couldn't resolve from prior art alone. Forwarded to the BA/architect/user.)

## What This Feature Will NOT Re-Implement

- (Explicit list. This is the section that prevents duplication of capabilities established in siblings. If empty, you probably haven't looked hard enough.)
```

### Step 6: Surface contradictions

For each entry where decision = **CONTRADICTS**, surface to the user/BA/architect immediately — do not continue planning until reconciliation is decided. The wrapper records reconciliation outcomes; this workflow only produces the list.

### Step 7: Return the structured outputs

Return:

- `inspection_set` — every entity and the doc paths read.
- `capability_map` — the table above as structured data.
- `reconciliations_needed` — the CONTRADICTS list.
- `reuse_links` — for every capability marked **REUSE** or **EXTEND**, emit `{sibling_key, link_kind}`. The wrapper translates these into host link primitives.
- `open_questions` — anything unresolved.
- `will_not_reimplement` — the explicit list from the report.

## Success Criteria

This workflow is complete when:
- [ ] Every sibling and related epic has been listed and its priority docs read (or PRD if no architecture).
- [ ] Every ADR / decision record on the inspection list has been read (not skimmed).
- [ ] Every domain term has been considered against the supplied `search_results`.
- [ ] The unified research report has a non-empty Capability Map.
- [ ] The "What This Feature Will NOT Re-Implement" section is filled in.
- [ ] Any CONTRADICTS entries have been surfaced to the user.
- [ ] `reuse_links` lists every sibling whose work will be reused or extended.

## Anti-Patterns

❌ **Skipping siblings without architecture docs.** Read the PRD instead — it still tells you what the sibling does.

❌ **Marking everything REUSE without reading.** The point is to make a real decision per capability. If you didn't read the sibling's architecture, you cannot reuse from it confidently.

❌ **Treating this as informational.** The output is a structured report and reuse links — not a mental note. The next agent will not re-read the parent context from scratch; they'll read your report.

❌ **Running this only for COMPLEX features.** Duplication risk is independent of complexity tier — STANDARD features fall into the same trap.

## Time Budget

- Small parent (≤5 siblings, no ADRs): 10 minutes.
- Medium parent (6–15 siblings, some ADRs): 15–25 minutes.
- Large parent (15+ siblings or cross-parent ADRs): 30–45 minutes — but if you're spending more than 45 minutes, the parent epic is probably mis-scoped and should be split.

## Related

- `skills/research/workflows/understand-feature.md` — deeper dive into a single sibling once Step 4 flags it as critical
- `skills/research/workflows/tracing-knowledge-lineages.md` — for "why was this decided" archaeology beyond what ADRs capture
- `skills/specification-writing/workflows/write-feature-prd.md` — calls this workflow as Step 1
- `skills/specification-writing/workflows/plan/feature-tech-plan.md` — calls this workflow as Step 1
- `skills/specification-writing/workflows/plan/epic-tech-plan.md` — calls this workflow as Step 1
- `skills/specification-writing/workflows/write-task.md` — sources Brownfield Context from the report this workflow produces
