---
inputs:
  - entity_type: "epic" | "feature" — which level is being checked
  - entity_id: opaque identifier for the entity being checked (string)
  - planned_docs: list of {doc_type, path} — documents the plan said would be produced
  - excluded_docs: list of {doc_type, exclusion_reason} — optional docs the plan said would NOT be produced, with rationale
  - registered_docs: list of {doc_type, path} — documents already registered in the host system's related-docs index
  - tier: "SIMPLE" | "STANDARD" | "COMPLEX" (optional; only relevant for feature.md/PRD depth rules)
outputs:
  - findings: list of {doc_type, verdict: PASS|FAIL|SKIP, reason}
  - exclusion_audit: list of {doc_type, verdict: PASS|FAIL, reason}
  - verdict: PASS | FAIL — overall validation result
  - failure_summary: list of {doc_type, reason} (empty on PASS) — used by the host to construct a blocker note
---

# Workflow: Check BA Docs (craft)

## Purpose

Shared validation logic used by the BA check templates at both epic and feature level.
This is the authoritative reference for what constitutes a passing BA document.

The activity is: given a plan that declared which BA documents would be created and which
would be excluded, decide whether the produced artifacts meet minimum content depth and
whether exclusion rationales are specific enough to defend the omissions.

---

## Validation Protocol

### Phase 1: Verify a Plan Exists

A BA check is meaningful only if the entity went through the PLAN gate and produced an
explicit list of documents to create plus rationales for any exclusions. If `planned_docs`
is empty AND `excluded_docs` is empty, the entity skipped planning — produce an immediate
overall FAIL with reason "No plan found. Entity skipped PLAN gate. Re-run refinement with
PLAN gate active." and return.

Otherwise proceed to document validation.

### Phase 2: Validate Each Planned Document

For every entry in `planned_docs`, evaluate three checks:

**Check 1 — Existence**
- The file at `path` must exist and be readable.
- FAIL with reason `"File missing: {path}"` if the file does not exist.

**Check 2 — Content Depth**
- Apply the depth rules per document type (see Minimum Depth Requirements below).
- A document fails depth if required sections are absent OR contain only placeholder text
  (TBD, TODO, placeholder, [fill in], lorem ipsum, etc.).

**Check 3 — Registration**
- The document must appear in `registered_docs` (matched by `doc_type` or `path`).
- FAIL with reason `"Not registered in related-docs index"` if it does not.

### Phase 3: Validate Exclusion Reasoning

For every entry in `excluded_docs`:
- The exclusion must carry a specific, entity-grounded `exclusion_reason`.
- FAIL with `"No exclusion reason documented"` if the reason is missing or empty.
- FAIL with `"Generic exclusion reason"` if the reason is generic ("not relevant",
  "not needed", "n/a") without specific rationale tying the decision to characteristics
  of THIS entity.

---

## Minimum Depth Requirements

### `epic.md`
- **Required sections**: Goal (with problem statement), Business Value, Quick Reference table,
  Features/Scope summary, Open Questions section (may be empty but must exist).
- **FAIL indicators**: Goals section is 1-2 lines, no business value, no quick reference.

### `requirements.md`
- **Required**: Functional requirements with `REQ-F` prefix, Non-functional with `REQ-NF` prefix.
- **FAIL indicators**: Fewer than 3 functional requirements, no non-functional requirements,
  requirements are vague ("system should be fast", "users can manage settings").
- **Testability check**: Each functional requirement must have a verifiable criterion.

### `scope.md`
- **Required**: In-scope items list, Out-of-scope items list with rationale for each exclusion.
- **FAIL indicators**: No out-of-scope section, out-of-scope items without rationale.

### `personas.md` (when included)
- **Required**: At least 2 persona profiles, each with: Role, Goals (2+), Pain Points (2+),
  Success Criteria.
- **FAIL indicators**: Single persona, personas without goals or pain points.

### `user-journeys.md` (when included)
- **Required**: At least 1 journey per persona, each with happy path AND at least 1 unhappy path.
- **FAIL indicators**: Happy paths only, no error/edge scenarios.

### `success-metrics.md` (when included)
- **Required**: KPIs with numeric targets and time bounds; measurement method specified.
- **FAIL indicators**: Vague metrics ("improve satisfaction"), no targets, no time bounds.

### Feature-Level Additions

### `feature.md` / PRD (SIMPLE tier)
- **Required**: Problem statement, Solution summary, Acceptance Criteria (3+ testable items).
- **FAIL indicators**: Missing AC, non-testable AC ("should look good"), under 30 meaningful lines.

### `feature.md` / PRD (STANDARD tier)
- **Required**: Goal (problem, solution, impact), MoSCoW user stories, Functional requirements,
  Acceptance criteria, Out of scope section.
- **FAIL indicators**: Missing MoSCoW prioritization, AC not testable/measurable.

---

## Output Format

Produce findings as a structured list before the overall verdict.

```
FINDINGS:
  PASS | epic.md          | Goal, business value, quick reference present. 5 sections.
  PASS | requirements.md  | 8 functional (REQ-F001-008), 4 non-functional (REQ-NF001-004). All testable.
  FAIL | scope.md         | Out-of-scope items listed without rationale for any exclusion.
  PASS | personas.md      | 2 personas (Admin, Developer). Goals and pain points defined.
  SKIP | user-journeys.md | Excluded per plan. Reason: backend-only epic, no user workflows.
  SKIP | success-metrics.md | Excluded per plan. Reason: internal refactor, no external KPIs.

EXCLUSION AUDIT:
  PASS | user-journeys.md | Reason documented: "backend-only epic"
  PASS | success-metrics.md | Reason documented: "internal refactor"
```

---

## Decision Rules

- **ALL planned docs PASS** AND **all excluded docs have specific, documented reasons** →
  overall verdict **PASS**. `failure_summary` is empty.

- **ANY FAIL** in either FINDINGS or EXCLUSION AUDIT → overall verdict **FAIL**.
  `failure_summary` contains one entry per failing document with its reason. The writing
  agent must fix the issues and re-enter the check gate; the host MUST NOT advance on FAIL.
