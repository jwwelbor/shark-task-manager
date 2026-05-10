---
inputs:
  - entity_type: "epic" | "feature" — which level is being checked
  - entity_id: opaque identifier for the entity being checked (string)
  - planned_docs: list of {doc_type, path} — tech documents the plan said would be produced
  - excluded_docs: list of {doc_type, exclusion_reason} — optional tech docs the plan said would NOT be produced, with rationale
  - registered_docs: list of {doc_type, path} — tech documents already registered in the host system's related-docs index
outputs:
  - findings: list of {doc_type, verdict: PASS|FAIL|SKIP, reason}
  - exclusion_audit: list of {doc_type, verdict: PASS|FAIL, reason}
  - verdict: PASS | FAIL — overall validation result
  - failure_summary: list of {doc_type, reason} (empty on PASS)
---

# Workflow: Check Tech Docs (craft)

## Purpose

Shared validation logic used by the tech check templates at both epic and feature level.
This is the authoritative reference for what constitutes a passing technical document.

The activity is: given a tech plan that declared which technical documents would be produced
and which would be excluded, decide whether the produced artifacts meet minimum content depth
(contracts, schemas, decisions) and whether exclusion rationales are specific enough to defend
the omissions. The contract-first principle of the planning gate means an "incomplete" tech doc
is one that leaves implementation contracts unspecified.

---

## Validation Protocol

### Phase 1: Verify a Tech Plan Exists

A tech check is meaningful only if the entity went through the tech PLAN gate and produced an
explicit list of tech documents to create plus rationales for any exclusions. If `planned_docs`
is empty AND `excluded_docs` is empty, the entity skipped tech planning — produce an immediate
overall FAIL with reason "No tech plan found. Re-run tech refinement with PLAN gate." and return.

Otherwise proceed to document validation.

### Phase 2: Validate Each Planned Tech Document

For every entry in `planned_docs`, evaluate three checks:

**Check 1 — Existence:** File at `path` must exist and be readable.
**Check 2 — Content Depth:** Meets minimum depth (see below).
**Check 3 — Registration:** Document appears in `registered_docs`.

### Phase 3: Validate Exclusion Reasoning

For every entry in `excluded_docs`:
- FAIL if `exclusion_reason` is missing.
- FAIL if it uses a generic reason ("not needed", "n/a", "not relevant") without specific
  rationale tying the decision to characteristics of THIS entity.

---

## Minimum Depth Requirements

### `02-architecture.md`
- **Required**: Component overview or diagram, integration points list, decision log with rationale.
- **FAIL indicators**: No integration points, no decisions documented, stub with only heading.

### `03-data-design.md` (when included)
- **Required**: All new/modified tables with columns (name, type, constraint), index definitions,
  migration strategy.
- **FAIL indicators**: Table names without column definitions, no migration notes, no constraints.

### `04-backend-design.md` (when included)
- **Required**: All new/modified endpoints with HTTP method, path, full request schema (field +
  type), full response schema (field + type + status code), error responses.
- **FAIL indicators**: Endpoints listed without schemas, schemas without types, no error responses.

### `05-frontend-design.md` (when included)
- **Required**: Component list, state management approach, API calls per component (which endpoint,
  when triggered, what params), loading and error states.
- **FAIL indicators**: Components listed without API integration, no error/loading states.

### `06-security-design.md` (when included)
- **Required**: Auth model with specific roles/permissions, data exposure matrix (what data, which
  roles), input validation rules for new data surfaces.
- **FAIL indicators**: Vague "standard security practices", no role-based analysis.

### `07-performance-design.md` (when included)
- **Required**: Specific numeric targets (latency ms, throughput req/s, cache TTL), measurement
  approach, scaling strategy.
- **FAIL indicators**: No numeric targets, "should be fast" language.

### `08-implementation-phases.md` (when included)
- **Required**: Phase breakdown with clear deliverables per phase, feature flag specification if
  applicable, migration sequencing (pre/post deploy).
- **FAIL indicators**: "Phase 1: build everything", no sequencing, no flag specifications.

### Tech Feasibility-Specific (epic level)

### `tech-feasibility.md` (always required at epic level)
- **Required**: Assessment for each requirement area, explicit APPROVED or CONCERNS_FOUND verdict,
  next-step actions if CONCERNS_FOUND.
- **FAIL indicators**: No verdict, hedged language ("might work"), no per-area assessment.

### `technical-risks.md` (when included at epic level)
- **Required**: Risk registry with name, severity (HIGH/MED/LOW), probability, mitigation action.
- **FAIL indicators**: Risks without mitigations, no severity rating, vague risk descriptions.

---

## Output Format

```
FINDINGS:
  PASS | 02-architecture.md | Component diagram present, 4 decisions logged, integration points mapped
  FAIL | 04-backend-design.md | 3 endpoints listed without response schemas; no error responses
  PASS | 03-data-design.md   | 2 new tables, full column definitions, migration scripts documented
  SKIP | 05-frontend-design.md | Excluded per plan. Reason: backend-only feature, no UI changes
  SKIP | 06-security-design.md | Excluded per plan. Reason: no new auth surfaces

EXCLUSION AUDIT:
  PASS | 05-frontend-design.md | Reason documented: "backend-only feature"
  PASS | 06-security-design.md | Reason documented: "no new auth surfaces"
```

---

## Decision Rules

- **ALL planned docs PASS** AND **all excluded docs have specific, documented reasons** →
  overall verdict **PASS**. `failure_summary` is empty.

- **ANY FAIL** in either FINDINGS or EXCLUSION AUDIT → overall verdict **FAIL**.
  `failure_summary` contains one entry per failing document with its reason. The host MUST NOT
  advance on FAIL.
