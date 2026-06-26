---
inputs:
  - feature_id: opaque feature identifier (string)
  - feature_spec_path: absolute path to the feature document (string)
  - feature_ba_doc_paths: list of absolute paths — feature-level BA docs (just validated)
  - parent_epic_id: opaque epic identifier (string)
  - parent_epic_tech_doc_paths: list of absolute paths — parent epic's tech docs (architecture, feasibility, risks, integration map)
  - research_report_path: absolute path to the feature's research report (optional; from in_research)
  - prior_art_report_path: absolute path to feature-level `prior-art-report.md` from consult-related-work workflow (REQUIRED — host must produce this before calling craft)
  - complexity_tier: "SIMPLE" | "STANDARD" | "COMPLEX" (optional; if not provided, craft assesses from feature)
outputs:
  - feature_signals: object with {complexity_tier, has_user_facing_changes, has_api_changes, has_schema_changes, has_ui_changes, has_cross_epic_dependencies, has_explicit_nfrs, has_rollout_complexity}
  - planned_docs: list of doc_type strings — required + INCLUDED optional tech docs
  - excluded_docs: list of {doc_type, exclusion_reason} for every optional doc that was EXCLUDED
  - decision_log: human-readable line summarizing the plan
  - capability_map_summary: short summary of REUSE/EXTEND/RE-IMPLEMENT capabilities pulled from prior-art-report
---

# Workflow: Feature Tech Plan (craft)

## Purpose

Apply a document-selection decision tree from a technical lens to a feature. Decide which
technical documents this feature needs (architecture, data design, backend, frontend, security,
performance, implementation phases) and produce a plan with specific entity-grounded rationale
for every INCLUDE / EXCLUDE.

**Contract-first principle:** The architect's job is to define contracts before any
implementation begins. Every API endpoint, data model, and integration interface must be
specified before tasks are generated. Implementation agents read these; they do not define
them.

The rigor of this step is the foundation for implementable tasks. As at the epic level,
sibling-feature architecture must be consulted via a prior-art report — capabilities marked
REUSE / EXTEND should reference sibling designs rather than duplicate them in new docs.

---

## Step 1: Read All Context

Read the inputs:

1. Feature BA docs at `feature_ba_doc_paths` (just validated by BA check).
2. Parent epic tech docs at `parent_epic_tech_doc_paths` (architecture, feasibility, risks).
3. `research_report_path` (if provided) — critical for brownfield analysis.
4. `prior_art_report_path` — **MANDATORY**. Produced by the consult-related-work workflow
   during the feature's research phase. If this report is missing, STOP and surface that as an
   upstream gate failure — the host must run consult-related-work first.

**Sibling-feature architecture reading is mandatory.** The prior-art report enumerates sibling
features under the same epic, pulls their `02-architecture.md`, `03-data-design.md`,
`04-backend-design.md`, and ADRs, and produces a Capability Map with REUSE / EXTEND /
RE-IMPLEMENT verdicts per capability. Reuse signals from this map drive your decision tree
below: a capability marked REUSE does NOT need a new architecture section in THIS feature —
`02-architecture.md` should reference the sibling's design instead of duplicating it.

Extract these signals into `feature_signals`:

- **Complexity tier**: from feature metadata or assessed from BA docs.
- **Scope signals**: What user-facing changes exist? API changes? Schema changes? UI changes?
- **Cross-epic dependencies**: What existing systems does this touch?
- **Performance/security signals**: Are there explicit NFRs?
- **Rollout complexity**: Feature flags, migrations, phased rollout needed?
- **Reuse signals from prior-art-report**: Which capabilities are REUSE/EXTEND from siblings?

Also produce `capability_map_summary` — a short prose summary of the prior-art Capability Map
that becomes the trace evidence that REUSE/EXTEND decisions are honored downstream.

---

## Step 2: Apply the Decision Tree

### Always Required
- `02-architecture.md` — Always required for all tiers.

### Optional Documents (apply for STANDARD and COMPLEX)

**`03-data-design.md`**
- INCLUDE: If the feature introduces new database tables, modifies existing schemas, adds new
  columns, changes relationships, or requires data migrations.
- EXCLUDE: Feature has no schema changes; existing data model is unchanged.

**`04-backend-design.md`**
- INCLUDE: If the feature introduces new API endpoints, modifies existing endpoint behavior,
  adds new business logic services, or changes data access patterns.
- EXCLUDE: Pure frontend changes with no new or modified backend behavior.

**`05-frontend-design.md`**
- INCLUDE: If the feature introduces or modifies any UI components, screens, or user
  interactions.
- EXCLUDE: Backend/API-only feature with no UI changes.

**`06-security-design.md`**
- INCLUDE: If the feature introduces new authentication surfaces, authorization rules, new
  data exposure (new data visible to users), or handles sensitive data.
- EXCLUDE: Internal refactor with no new data exposure or auth changes.

**`07-performance-design.md`**
- INCLUDE: If explicit performance requirements are stated (load targets, latency SLAs,
  throughput requirements, or volume targets).
- EXCLUDE: No explicit performance requirements beyond standard service defaults.

**`08-implementation-phases.md`**
- INCLUDE: If the feature requires a multi-phase rollout, feature flags for gradual release,
  database migrations that must run before/after code deployment, or backward compatibility
  shims.
- EXCLUDE: Single-phase deployment with no migration complexity.

For every optional doc, write the reason in the form `"INCLUDE — [specific reason]"` or
`"EXCLUDE — [specific reason]"` grounded in THIS feature.

---

## Step 3: Depth Requirements Per Document

Before writing, understand what each document must contain at minimum. These are the same
depth bars the tech-check applies — planning to "include" a doc is a commitment to meet these.

**`02-architecture.md`** (all tiers):
- Component diagram or architecture table showing how this feature fits the system.
- Integration points with existing components.
- Decision log: key architectural choices and rationale.
- For SIMPLE: existing patterns used, 1–2 pages max.
- Where the prior-art Capability Map says REUSE/EXTEND, reference the sibling design rather
  than re-derive it.

**`03-data-design.md`** (when included):
- Complete schema: new/modified tables with all columns, types, constraints, indexes.
- Relationship diagram or ERD notation.
- Migration strategy (up/down).
- Data validation rules.

**`04-backend-design.md`** (when included):
- All new/modified API endpoints with full request/response schemas (not just field names —
  types too).
- Service layer design: which services handle which business logic.
- Error responses and status codes.
- Data transformation logic between persistence and API layers.

**`05-frontend-design.md`** (when included):
- Component hierarchy.
- State management approach.
- API integration points (which endpoints called, when, with what params).
- Loading/error states for each async operation.

**`06-security-design.md`** (when included):
- Auth model: who can do what (permission matrix or role-based table).
- Data exposure analysis: what data is newly visible to which user types.
- Input validation and sanitization requirements.
- Security test requirements.

---

## Step 4: Emit the Plan

Produce the structured outputs:

- `planned_docs` — `["02-architecture.md", ...]` plus each optional doc whose decision was
  INCLUDE.
- `excluded_docs` — one entry per optional doc whose decision was EXCLUDE, each with the
  specific entity-grounded reason from Step 2.
- `decision_log` — single-line summary: `"TECH PLAN: Will create N docs: [list]. Excluded: [doc — reason]."`
- `capability_map_summary` — REUSE/EXTEND/RE-IMPLEMENT trace from Step 1.

The host stores these into `context_data` (under `tech_`-prefixed fields) and advances
to the ACT phase.

---

## Anti-Patterns to Avoid

- **Do NOT** plan tech docs without first consuming the prior-art report. Skipping it produces
  duplicate architecture work and contradicts capability decisions sibling features made.
- **Do NOT** plan to write a placeholder doc to satisfy the list — depth requirements are
  enforced at tech check.
- **Do NOT** include `06-security-design.md` for refactors that don't change auth or data
  exposure. Generic "always include" produces noise.
- **Do NOT** use generic exclusion reasons — the tech check fails them.
