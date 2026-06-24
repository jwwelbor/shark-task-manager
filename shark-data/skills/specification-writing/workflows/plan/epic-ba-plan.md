---
inputs:
  - epic_id: opaque epic identifier (string)
  - epic_spec_path: absolute path to the draft epic document (string)
  - epic_title: short epic title (string)
  - epic_description: epic description / brief / vision notes (string)
  - existing_context: prior context_data on the epic (object; may be empty)
  - complexity_tier: "SIMPLE" | "STANDARD" | "COMPLEX" (optional; if not provided, the craft assesses from scope)
outputs:
  - epic_signals: object with {epic_type, complexity_tier, has_user_surface, has_external_outcomes}
  - planned_docs: list of doc_type strings — required + INCLUDED optional documents
  - excluded_docs: list of {doc_type, exclusion_reason} for every optional doc that was EXCLUDED
  - decision_log: human-readable line summarizing the plan ("Will create N documents: [list]. Excluded: [doc - reason, ...].")
---

# Workflow: Epic BA Plan (craft)

## Purpose

Apply a document-selection decision tree to a draft epic. Determine exactly which BA-side
documents this epic needs, with a specific, entity-grounded rationale for every INCLUDE and
EXCLUDE decision. The output of this activity is a plan: a list of documents to write, and a
list of documents that will NOT be written along with why each one is inapplicable to THIS
epic.

The decision tree must produce non-generic reasoning. "Not relevant" is never sufficient — the
plan must cite a characteristic of THIS epic (its scope, its users, its measurability) that
makes a given document applicable or not.

---

## Step 1: Assess the Epic

Before applying the decision tree, read the inputs (`epic_spec_path`, `epic_title`,
`epic_description`, `existing_context`).

Extract these signals:

- **Epic type**: infrastructure / internal tooling, backend/API-only, user-facing product, or
  mixed.
- **Complexity tier**: SIMPLE / STANDARD / COMPLEX. Use `complexity_tier` if provided; otherwise
  assess from scope (small focused change → SIMPLE; multi-area work with new surfaces → COMPLEX).
- **User surface**: Does this epic introduce or significantly change any user-facing workflows?
- **External outcomes**: Are there externally measurable business outcomes tied to this epic?

These signals populate `epic_signals` in the output and drive the decision tree below.

---

## Step 2: Apply the Decision Tree

For each optional document, decide INCLUDE or EXCLUDE with a specific reason grounded in THIS
epic's characteristics. Do NOT use generic reasons. **If you are uncertain: default to INCLUDE.**

### Always Required (no decision needed)
- `epic.md` — Always created on epic creation; you fill it with content.
- `requirements.md` — Always required.
- `scope.md` — Always required.

### Optional: `personas.md`

**INCLUDE if:**
- The epic introduces or modifies user-facing workflows.
- There are distinct user types with different goals or access patterns.
- The epic involves a new product surface, UI, or user authentication flow.

**EXCLUDE if:**
- Infrastructure epic (no user interaction, e.g., observability, deployment pipeline, DB
  migration).
- Internal tooling used only by developers or automated systems.
- Backend/API-only work with no distinct user workflows.

**Write your reason** in the form:
- "INCLUDE — [specific characteristic of this epic that triggers inclusion]", or
- "EXCLUDE — [specific characteristic that makes this inapplicable, e.g., 'Infrastructure epic:
  refactors internal queue system, no user-facing workflows']".

### Optional: `user-journeys.md`

**INCLUDE if:**
- Multiple distinct user workflows exist that need to be mapped.
- Complex multi-step user interactions are part of scope.
- Different user types have meaningfully different workflow paths.

**EXCLUDE if:**
- Backend/API-only with no distinct user interactions.
- All user interaction is captured adequately in `requirements.md` (single simple flow).
- `personas.md` was excluded (user journeys without personas are not useful).

### Optional: `success-metrics.md`

**INCLUDE if:**
- There are externally measurable business outcomes (conversion, adoption, latency SLA).
- The epic has explicit KPI targets or executive-level success criteria.
- Product or business stakeholders need to track this epic's impact.

**EXCLUDE if:**
- Internal refactor with no externally observable impact.
- Technical debt reduction where "success" = code is cleaner (document in `scope.md` instead).
- Success is binary: either it works or it doesn't (e.g., migration to new infrastructure).

---

## Step 3: Emit the Plan

Produce the structured outputs:

- `planned_docs` — `["epic.md", "requirements.md", "scope.md", ...]` plus each optional doc whose
  decision was INCLUDE.
- `excluded_docs` — one entry per optional doc whose decision was EXCLUDE, each with the
  specific entity-grounded reason from Step 2.
- `decision_log` — single-line summary: `"PLAN: Will create N documents: [list]. Excluded: [doc — reason, doc — reason]."`

The host stores these into shark `context_data` and advances to the ACT phase.

---

## Anti-Patterns to Avoid

- **Do NOT** include all 6 documents by default "just to be safe". Plan deliberately.
- **Do NOT** plan to write a placeholder or stub document to satisfy a list — the BA check
  applies depth requirements and a stub will fail.
- **Do NOT** exclude documents without a specific, entity-grounded reason.
- **Do NOT** include vague reasons like "not relevant" — state WHY it's not relevant to THIS
  epic.
- **Do NOT** default EXCLUDE under uncertainty. If the signals are ambiguous, INCLUDE the doc.
