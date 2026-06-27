---
inputs:
  - epic_id: opaque epic identifier (string)
  - epic_title: epic title (string)
  - epic_doc_paths: object with paths — `{index, personas, user_journeys, requirements, success_metrics, scope}` (all required)
  - epic_research_report_path: absolute path to epic research report
  - ba_feasibility_report_path: absolute path to BA feasibility review report
  - tech_feasibility_report_path: absolute path to technical feasibility review report
  - existing_features: list of {feature_id, title, scope_summary} for any features already created under this epic (may be empty)
  - sibling_epic_summaries: list of {epic_id, title, scope_summary} for cross-epic awareness (optional)
  - interaction_map_path: absolute path to `<epic-id>-interaction-map.md` if present (optional; required for 3+ feature epics after design)
  - decomposition_summary_path: absolute path where the decomposition summary document should be written
  - feature_directory_resolver: function/contract for translating a feature slug to a directory path (host supplies resolved directory in `created_features` after creation)
outputs:
  - feature_candidates: list of {feature_slug, title, scope_summary, requirements_mapping, personas_served, user_journeys_covered, size_estimate, risk_level}
  - dependency_graph: list of {from_feature, to_feature, kind: "data" | "api" | "ux" | "infra"}
  - execution_order: list of {feature_slug, order} — features at the same order can run in parallel
  - traceability_matrix: list of {requirement_id, feature_slugs, coverage: "Full" | "Partial" | "Cross-cutting"}
  - decomposition_summary: markdown body for the summary document
  - decisions_log: list of {decision, rationale, alternative_considered}
  - open_questions: list of unresolved items needing stakeholder input
  - completeness_gaps: list of {requirement_id, reason} — empty if all epic requirements traced
---

# Workflow: Decompose Epic into Features (craft)

## Purpose

Break a fully-refined epic into discrete, implementable features with clear scope boundaries, dependency ordering, and full traceability to epic requirements. Each feature must be independently refinable and sized for 1–3 sprints of work.

## Core Responsibility

Read the complete epic PRD (all required files), research report, and feasibility review reports (BA and technical), then decompose the epic into features that collectively deliver all epic requirements.

## Input Requirements

Before starting, you must have access to:

1. **Epic PRD files** (all required) — paths supplied via `epic_doc_paths`:
   - **Index** (epic.md or equivalent) — main index with goal, business value, quick reference
   - **Personas** — user persona profiles
   - **User journeys** — user workflow narratives
   - **Requirements** — functional and non-functional requirements catalog
   - **Success metrics** — KPIs and measurement framework
   - **Scope** — boundaries and exclusions

2. **Epic research report** (`epic_research_report_path`) — strategic research findings

3. **Feasibility review reports** — BA feasibility (business viability, cross-epic conflicts, scope coherence) and technical feasibility (architectural concerns, dependency risks)

4. **Existing features** (`existing_features`) — to avoid duplication

5. **Interaction map** (`interaction_map_path`) — required when the epic has 3+
   features or explicit cross-feature handoffs. Every I-## must have a producer
   feature and at least one consumer feature in the feature list you create.

6. **Sibling epic context** (`sibling_epic_summaries`) — cross-epic awareness

## Your Process

### Step 0: Verify Inputs Are Substantive

1. **Read each epic PRD file** and confirm content is substantive (not placeholder). If any file is missing or thin, STOP and report which files need completion.

2. **Read the research and feasibility reports**. Note feasibility constraints, system interactions, and risk items from all three documents.

3. **Inspect `existing_features`**. Understand what's already been created and avoid duplication.

4. **Read the interaction map if present**. Extract every I-## row, producer
   capability, consumer capability, shape source, payload, and style. Treat
   missing producer/consumer assignments as blockers to resolve in the feature
   set, not as optional notes.

### Step 1: Analyze Epic Through Four Decomposition Lenses

Apply each lens independently, then synthesize into a unified feature set.

#### Lens 1: User Journey Decomposition

- Map each user journey to potential features.
- Identify natural workflow boundaries where a user completes a meaningful action.
- Group related journey steps that must ship together for coherent UX.

#### Lens 2: Functional Domain Decomposition

- Group requirements by functional domain.
- Identify cohesive sets of functionality (e.g., authentication, data management, reporting).
- Ensure each domain group delivers independent value.

#### Lens 3: Technical Layer Decomposition

- Identify infrastructure or platform capabilities needed before user-facing features.
- Separate data model setup from business logic from UI concerns where appropriate.
- Flag shared services or libraries that multiple features will depend on.

#### Lens 4: Risk-Based Decomposition

- Reference the research report's risk assessment.
- Identify high-risk areas that should be isolated into their own features.
- Consider which features de-risk others (e.g., proof-of-concept features first).

### Step 2: Synthesize Feature Candidates

Merge insights from all four lenses into a candidate feature list:

1. **Identify natural feature boundaries** where multiple lenses agree.
2. **Resolve conflicts** where lenses suggest different groupings:
   - Prefer user-journey coherence over technical layering.
   - Prefer risk isolation over domain grouping.
3. **For each feature candidate, define**:
   - **Title**: clear, descriptive name (3–8 words).
   - **Scope summary**: 2–3 sentences describing what this feature delivers.
   - **Requirements mapping**: which epic requirements (REQ-F-xxx, REQ-NF-xxx) this feature addresses.
   - **Personas served**: which personas benefit.
   - **User journeys covered**: which journeys or journey segments.
   - **Size estimate**: Small (1 sprint), Medium (2 sprints), Large (3 sprints).
   - **Risk level**: Low, Medium, High (from research report).

Capture each candidate in `feature_candidates`.

### Step 3: Validate Feature Boundaries

Before finalizing, verify the decomposition is sound.

#### Completeness Check

- [ ] Every functional requirement (REQ-F-xxx) maps to at least one feature
- [ ] Every non-functional requirement (REQ-NF-xxx) is addressed (either by a specific feature or as a cross-cutting concern)
- [ ] Every success metric traces to at least one feature
- [ ] Every user journey is fully covered across features
- [ ] No epic scope items are left unaddressed
- [ ] Every I-## in the interaction map maps to one producer feature and at
      least one consumer feature; no orphan wires

Capture any gaps in `completeness_gaps`. The decomposition cannot be considered complete until this list is empty.

#### Overlap Check

- [ ] No two features deliver the same requirement (unless explicitly shared)
- [ ] Feature boundaries are crisp — a developer can tell which feature owns what
- [ ] Shared concerns (auth, logging, error handling) are explicitly assigned or noted as cross-cutting

#### Size Check

- [ ] No feature exceeds 3 sprints of estimated effort
- [ ] No feature is so small it's really a task (< 3 tasks expected)
- [ ] Features are roughly comparable in size (no 10x variance)

#### Scope Boundary Check

- [ ] Features collectively do not exceed epic scope
- [ ] Out-of-scope items from `scope.md` are not included in any feature
- [ ] Each feature's scope can be explained in 2–3 sentences

### Step 4: Define Dependency Ordering

1. **Identify dependencies between features**:
   - **Data**: Feature B needs tables/models created by Feature A.
   - **API**: Feature B consumes APIs built by Feature A.
   - **UX**: Feature B's UI extends or builds on Feature A's UI.
   - **Infrastructure**: Feature B needs services deployed by Feature A.

   Capture as `dependency_graph` entries.

2. **Build the dependency graph** and verify it is a **DAG** (no circular dependencies). If circular, refactor feature boundaries to break cycles.

3. **Assign execution order**:
   - Features with no dependencies → `order: 1`.
   - Features depending only on order-1 features → `order: 2`.
   - Continue until all features are ordered.
   - Features at the same order level can run in parallel.

   Capture as `execution_order`.

4. **Common ordering patterns** (use whichever fits the epic):
   - **Infrastructure-first**: shared services, database setup, auth → then domain features.
   - **Vertical slice**: one complete user journey first → then expand breadth.
   - **Risk-first**: highest-risk features early to fail fast.

### Step 5: Build the Traceability Matrix

For each requirement in the epic, list the feature(s) that address it and classify coverage:

| Epic Requirement | Feature(s) | Coverage |
|------------------|-----------|----------|
| REQ-F-001 | F01 | Full |
| REQ-F-002 | F01 | Full |
| REQ-NF-001 | F01, F03 | Cross-cutting |

Capture as `traceability_matrix`. If any requirement has no entry, that's a `completeness_gaps` failure.

If an interaction map exists, add an interaction trace:

| Interaction | Producer feature | Consumer feature(s) | Shape source | Closure |
|-------------|------------------|---------------------|--------------|---------|
| I-01 | F01 | F02, F03 | architecture.md#section | Closed |

Closure is `Closed` only when producer and all consumers exist in
`feature_candidates`; otherwise it is a decomposition blocker.

### Step 6: Write the Decomposition Summary

Produce a decomposition summary document at `decomposition_summary_path` containing:

#### Feature Tree

```
Epic: {epic-id} — "{Epic Title}"
├── F01: {Feature 1 Title} [order: 1, size: Medium, risk: Low]
│   └── Requirements: REQ-F-001, REQ-F-002, REQ-NF-001
├── F02: {Feature 2 Title} [order: 1, size: Small, risk: Medium]
│   └── Requirements: REQ-F-003, REQ-F-004
├── F03: {Feature 3 Title} [order: 2, size: Large, risk: High]
│   ├── Depends on: F01, F02
│   └── Requirements: REQ-F-005, REQ-F-006, REQ-F-007
└── F04: {Feature 4 Title} [order: 3, size: Medium, risk: Low]
    ├── Depends on: F03
    └── Requirements: REQ-F-008, REQ-NF-002
```

#### Traceability Matrix

(rendered from `traceability_matrix`)

#### Dependency Graph

Visual representation of feature dependencies and execution order.

#### Metrics Traceability

Map each success metric to the feature(s) that enable its measurement.

Place the rendered markdown in `decomposition_summary` (output) AND write it to `decomposition_summary_path`.

### Step 7: Completeness Verification

Before returning, run through:

- [ ] `feature_candidates` covers every epic requirement (`completeness_gaps` is empty)
- [ ] `dependency_graph` is acyclic
- [ ] Each feature sized for 1–3 sprints
- [ ] `execution_order` defined for all features
- [ ] `traceability_matrix` complete
- [ ] Out-of-scope items from epic scope file are not included in any feature
- [ ] Every I-## from the interaction map is named in exactly the feature
      candidates that produce or consume it

## MANDATORY: Interactive Review with User

After generating the decomposition, present it to the user for review **before returning**. Do NOT silently proceed.

### Review Process

1. **Present the decomposition summary**:
   - Feature tree with sizes and risk levels.
   - Dependency graph.
   - Traceability matrix.
   - Any trade-offs or decisions you made (`decisions_log`).

2. **Highlight areas needing input**:
   - Features that could be split or merged differently.
   - Ordering decisions that involve trade-offs.
   - Requirements that were ambiguous about feature assignment.
   - Risk assessments that might affect prioritization.

3. **Walk through feedback**:
   - Accept modifications to feature boundaries.
   - Adjust dependencies and ordering as directed.
   - Re-validate completeness after changes.

4. **Only return** when the user explicitly approves the decomposition.

## Common Patterns

### Infrastructure-First Pattern

Best for: Epics that introduce new technical capabilities.

```
Order 1: Infrastructure/platform feature (database, services, auth)
Order 2: Core domain features (business logic, APIs)
Order 3: User-facing features (UI, integrations)
Order 4: Enhancement features (optimization, analytics)
```

### Vertical Slice Pattern

Best for: Epics that deliver a new user workflow.

```
Order 1: Minimum viable slice (one complete user journey, end-to-end)
Order 2: Expand breadth (additional journeys, personas)
Order 3: Enhance depth (edge cases, advanced features)
Order 4: Polish (performance, accessibility, monitoring)
```

### Risk-First Pattern

Best for: Epics with significant technical uncertainty.

```
Order 1: Proof-of-concept feature (validate highest-risk assumption)
Order 2: Core features (build on validated approach)
Order 3: Extension features (expand functionality)
Order 4: Hardening features (security, performance, resilience)
```

## Anti-Patterns to Avoid

- **Feature-per-requirement**: don't create a feature for each requirement; group related requirements into cohesive features.
- **Technical-layer features**: avoid "Database Feature" + "API Feature" + "UI Feature"; instead, create features around user-facing capabilities that cut across layers.
- **Mega-features**: if a feature has more than 8–10 expected tasks, it's too large — split it.
- **Micro-features**: if a feature has fewer than 3 expected tasks, it's too small — merge it with a related feature.
- **Hidden dependencies**: don't assume features are independent; explicitly map all dependencies.
- **Scope creep**: don't add features for things outside the epic scope; flag them as potential future epics.
