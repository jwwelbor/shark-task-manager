---
inputs:
  - feature_id: opaque feature identifier (string)
  - feature_spec_path: absolute path to the draft feature document (string)
  - feature_description: feature description / brief (string)
  - parent_epic_id: opaque epic identifier (string)
  - parent_epic_doc_paths: list of absolute paths — BA docs the parent epic produced (epic.md, requirements.md, scope.md, plus any optional)
  - parent_epic_implementation_decisions: object — the parent epic's `implementation_decisions` JSON (which BA docs the epic INCLUDED/EXCLUDED and why)
  - complexity_tier: "SIMPLE" | "STANDARD" | "COMPLEX" (optional; if not provided, the craft assesses from feature description)
outputs:
  - tier: "SIMPLE" | "STANDARD" | "COMPLEX" — the tier the craft used to drive the decision tree
  - planned_docs: list of doc_type strings — required + INCLUDED optional documents
  - excluded_docs: list of {doc_type, exclusion_reason} for every optional doc that was EXCLUDED
  - decision_log: human-readable line summarizing the plan
---

# Workflow: Feature BA Plan (craft)

## Purpose

Apply a tier-aware document-selection decision tree to a draft feature. Determine which BA-side
documents this feature needs, with specific entity-grounded rationale for each INCLUDE / EXCLUDE.

**Key principle: feature docs are incremental over the epic docs.** Read the parent epic PRD
first — do not repeat what's already there. Feature docs add specificity, user stories,
acceptance criteria, and feature-specific scope. If the parent epic's `personas.md` already
covers all relevant personas, the feature does NOT need its own personas doc.

---

## Step 1: Read the Parent Epic Context First

Before deciding what to create, read:

- The parent epic's BA docs at `parent_epic_doc_paths`.
- `parent_epic_implementation_decisions` — what was already covered at the epic level and what
  was deliberately excluded.

Your feature docs must be incremental. Do not rewrite the epic's personas or journeys unless
the feature meaningfully extends them.

---

## Step 2: Determine Complexity Tier

Use `complexity_tier` if provided. Otherwise assess from the feature description:

- **SIMPLE**: Single focused change, 1 agent, no new systems, no new user types.
- **STANDARD**: Multiple requirements, possibly 2–3 agents, minor new user workflows.
- **COMPLEX**: Multi-agent, new user surface, schema changes, or cross-epic dependencies.

The chosen tier is emitted as the `tier` output and drives the decision tree below.

---

## Step 3: Apply the Decision Tree by Tier

### SIMPLE tier

Documents: `feature.md` only.

No optional docs. `feature.md` must contain:
- Problem being solved (1–2 sentences).
- Solution approach (1–2 sentences).
- Acceptance criteria (3+ testable items).

`planned_docs = ["feature.md"]`. `excluded_docs` lists `personas.md`, `user-journeys.md`, and
`success-metrics.md` each with reason `"EXCLUDE — SIMPLE tier; epic-level docs sufficient"`
(or a more specific reason if applicable).

### STANDARD tier

Documents: `feature.md` plus optional `personas.md`.

**`personas.md`** — INCLUDE only if this feature introduces a NEW persona type not covered in
the epic's `personas.md`. If the epic already has all relevant personas, EXCLUDE.

`planned_docs` includes `feature.md` plus `personas.md` if the decision was INCLUDE.

### COMPLEX tier

Documents: `feature.md` plus assessed supporting files.

For each supporting file, apply the decision:

**`personas.md`**:
- INCLUDE only if new persona types not in epic.
- If the epic covered them: EXCLUDE.
- If a persona needs feature-specific elaboration (not repetition): INCLUDE with delta
  content only.

**`user-journeys.md`**:
- INCLUDE if this feature introduces new user journeys not covered at epic level, or if the
  feature significantly changes an existing journey.
- EXCLUDE if journeys are adequately covered at epic level and this feature is a technical
  sub-step.

**`success-metrics.md`**:
- INCLUDE if the feature has its own measurable KPIs separate from the epic's metrics
  (e.g., a specific conversion rate for a new onboarding flow).
- EXCLUDE if success is measured at epic level only.

For every optional doc, write the reason in the form `"INCLUDE — [specific reason]"` or
`"EXCLUDE — [specific reason]"`. Example EXCLUDE reasons that pass the downstream check: "Epic
already defines all relevant personas", "Journey is fully covered by parent epic's
user-journeys.md (Section: Onboarding)", "Success measured at epic level via the conversion
KPI; no feature-specific KPI needed".

---

## Step 4: Emit the Plan

Produce the structured outputs:

- `tier` — the tier used for decisions.
- `planned_docs` — `["feature.md", ...]` plus each optional doc whose decision was INCLUDE.
- `excluded_docs` — one entry per optional doc whose decision was EXCLUDE.
- `decision_log` — single-line summary: `"PLAN: {tier} tier. Will create N docs: [list]. Excluded: [doc — reason]."`

The host stores these into `context_data` and advances to the ACT phase.

---

## Anti-Patterns to Avoid

- **Do NOT** restate epic personas or journeys in feature docs. Reference and extend, don't
  duplicate.
- **Do NOT** include optional docs at SIMPLE tier — the simplicity is the point.
- **Do NOT** use generic exclusion reasons. Cite the specific epic-level coverage that makes
  the feature-level doc unnecessary.
