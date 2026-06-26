---
inputs:
  - epic_id: opaque epic identifier (string)
  - epic_spec_path: absolute path to the epic document (string)
  - ba_doc_paths: list of absolute paths — BA docs already produced (epic.md, requirements.md, scope.md, plus any optional)
  - research_report_path: absolute path to the research report (optional; from the research phase)
  - ba_feasibility_report_path: absolute path to BA feasibility review report (optional; from BA feasibility review)
  - prior_art_report_path: absolute path to `prior-art-report.md` produced by consult-related-work workflow (REQUIRED — host must produce this before calling craft)
  - complexity_tier: "SIMPLE" | "STANDARD" | "COMPLEX" (optional; if not provided, the craft assesses from BA docs)
outputs:
  - epic_signals: object with {complexity_tier, integration_footprint, risk_profile, has_schema_impact}
  - planned_docs: list of doc_type strings — required + INCLUDED optional tech docs
  - excluded_docs: list of {doc_type, exclusion_reason} for every optional doc that was EXCLUDED
  - decision_log: human-readable line summarizing the plan
  - capability_map_summary: short summary of REUSE/EXTEND/RE-IMPLEMENT capabilities pulled from prior_art_report
---

# Workflow: Epic Tech Plan (craft)

## Purpose

Apply a document-selection decision tree from a technical lens. Decide which technical
documents this epic needs (architecture, risks register, integration map) and produce a plan
with specific, entity-grounded rationale for every INCLUDE / EXCLUDE.

The rigor of this step is the foundation for implementable tasks. Tech-side planning leans on
two prerequisite reads: the BA docs (already validated) and a **prior-art report** that maps
which capabilities can be REUSED / EXTENDED from sibling epics versus those that need new
architecture in THIS epic.

---

## Step 1: Assess the Epic from a Technical Lens

Read the inputs:

1. All BA docs at `ba_doc_paths` (epic.md, requirements.md, scope.md, plus any optional).
2. `research_report_path` (if provided) — research report from the research phase.
3. `ba_feasibility_report_path` (if provided) — BA feasibility review report.
4. `prior_art_report_path` — **MANDATORY**. This report enumerates sibling epics' capabilities
   and produces REUSE / EXTEND / RE-IMPLEMENT decisions per capability. The Capability Map
   inside it drives this tech plan: capabilities marked REUSE/EXTEND **do not need new
   architecture sections** in THIS epic — `02-architecture.md` should reference the sibling's
   design rather than duplicating it. If `prior_art_report_path` was not provided, STOP and
   surface that as an upstream gate failure; the host must run the consult-related-work
   workflow first.

Extract these signals into `epic_signals`:

- **Complexity tier**: SIMPLE / STANDARD / COMPLEX (from triage metadata or BA assessment).
- **Integration footprint**: How many existing epics/systems does this touch?
- **Risk profile**: Are there external dependencies, scaling concerns, or novel technology
  choices?
- **Schema impact**: Does this epic introduce new persistent data or modify existing schemas?

Also produce `capability_map_summary` — a short prose summary of which capabilities you found
in the prior-art report and what the REUSE/EXTEND/RE-IMPLEMENT verdict was for each. This
becomes the trace evidence that REUSE/EXTEND decisions were honored when later docs are
written.

---

## Step 2: Apply the Decision Tree

### Always Required
- `tech-feasibility.md` — Always required at epic level.

### Optional: `technical-risks.md`

**INCLUDE if:**
- COMPLEX tier.
- High integration risk (touches 3+ existing systems or external services).
- Novel technology choices with uncertain team capability.
- Significant performance, security, or data migration concerns.

**EXCLUDE if:**
- SIMPLE or STANDARD tier with known, well-understood patterns.
- Incremental extension of existing well-tested architecture.
- No external dependencies beyond already-used services.

### Optional: `integration-map.md`

**INCLUDE if:**
- Epic touches 2+ existing epics or third-party systems in non-trivial ways.
- New data contracts or API boundaries need to be established.
- Existing systems need changes to accommodate this epic.

**EXCLUDE if:**
- Fully self-contained epic with no cross-system data flows.
- Only depends on already-documented integrations in existing epics.

For every optional doc, write the reason in the form `"INCLUDE — [specific reason]"` or
`"EXCLUDE — [specific reason]"` grounded in THIS epic. Generic reasons fail the downstream
check.

---

## Step 3: Emit the Plan

Produce the structured outputs:

- `planned_docs` — `["tech-feasibility.md", ...]` plus each optional doc whose decision was
  INCLUDE.
- `excluded_docs` — one entry per optional doc whose decision was EXCLUDE, each with the
  specific entity-grounded reason from Step 2.
- `decision_log` — single-line summary: `"TECH PLAN: Will create N tech docs: [list]. Excluded: [doc — reason]."`
- `capability_map_summary` — REUSE/EXTEND/RE-IMPLEMENT trace from Step 1.

The host stores these into `context_data` (under `tech_`-prefixed fields) and advances to
the ACT phase.

---

## Anti-Patterns to Avoid

- **Do NOT** plan tech docs without first consuming the prior-art report. Skipping it produces
  duplicate architecture work and contradicts capability decisions that other epics already made.
- **Do NOT** include `technical-risks.md` for SIMPLE/STANDARD epics that follow well-trodden
  patterns — risk registers without real risks become noise.
- **Do NOT** use generic exclusion reasons. The downstream tech check fails generic reasons.
