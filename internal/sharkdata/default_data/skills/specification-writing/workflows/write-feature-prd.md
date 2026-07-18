---
inputs:
  - feature_concept: brief or description of what the feature does (string)
  - parent_epic_id: opaque epic identifier (string)
  - parent_epic_paths: list of {label, path} for epic PRD, requirements, scope, personas, journeys, success metrics
  - sibling_features: list of {feature_id, title, scope_summary, doc_paths} — siblings under the same epic (may be empty for the first feature)
  - research_report_path: absolute path to validated unified research report and Capability map (REQUIRED)
  - interaction_map_path: absolute path to parent `<epic-id>-interaction-map.md` if present (optional; required for multi-feature epics with I-## wires)
  - existing_personas: list of {name, file_path} already documented (optional)
  - feature_prd_path: absolute path where the feature PRD should be written
  - feature_directory: absolute path of the feature's directory (where additional BA-tier docs may go)
  - plan_decision: object describing which BA-tier documents to include for this feature
  - plan_remaining_steps: list of document slugs still to produce (RESUME mode); empty if starting fresh
  - complexity_tier: "snow-shoveling" | "ikea-furniture" | "heart-surgery" — determines PRD detail depth
outputs:
  - created_documents: list of {slug, path, title}
  - decisions_log: list of {decision, rationale, file_referenced}
  - open_questions: list of unresolved items needing stakeholder input
  - sibling_capability_reuse: list of {capability, sibling_feature_id, mode: "reuse" | "extend" | "delegate"} — capabilities the new feature consumes from siblings instead of re-implementing
  - sibling_capability_excluded: list of {capability, sibling_feature_id, reason} — capabilities explicitly NOT re-implemented
  - cross_feature_interactions: list of {interaction_id, mode: "produces" | "consumes", counterpart_features, shape_source, contract_test_pointer}
  - prd_complexity_tier: echoed back so host can record
---

# Workflow: Write Feature PRD (craft)

## Purpose

Transform a high-level feature description (or epic enabler) into a detailed, comprehensive PRD that serves as the single source of truth for engineering teams.

## Core Principles

1. **Clarity Over Brevity**: Be thorough and specific. Ambiguity leads to implementation errors and rework.
2. **User-Centric Thinking**: Always ground requirements in user needs and business outcomes.
3. **Completeness**: Anticipate edge cases, error states, and non-functional requirements.
4. **Actionability**: Every requirement should be testable and verifiable.
5. **Scope Discipline**: Clearly define what is in scope and what is not, to prevent scope creep.

## Your Process

### Step 1: Gather Information

Before writing the PRD, ensure you have:

- The parent Epic documentation (PRD and architecture if available) — provided via `parent_epic_paths`
- A clear understanding of the feature request, **verified by the user**
- Target user personas (`existing_personas`)
- Business context and success metrics (from epic)
- **Research report (MANDATORY):** The host must provide `research_report_path`.
  Its Capability map records sibling and related capability decisions. The PRD
  must reference it and explicitly call out what the feature reuses, extends,
  or will not re-implement.

If any critical information is missing, ask specific, targeted questions. Examples:

- "What specific user problem does this feature solve?"
- "Are there any performance requirements or constraints?"
- "What are the expected success metrics?"
- "Are there any compliance or security considerations?"
- "What should explicitly NOT be included in this feature?"

### Step 2: Assess Complexity

`complexity_tier` is supplied by the host but you re-validate during writing:

- The PRD detail depth must match the complexity tier.
- **snow-shoveling** (e.g., adding a button to a UI) — short PRD, minimal sections.
- **ikea-furniture** — medium PRD, all standard sections, modest depth.
- **heart-surgery** (significant architectural impact) — full PRD, deep on edge cases, NFRs, and integration concerns.

If the supplied tier feels miscategorized for the actual feature, flag it in `decisions_log` and proceed at the level you judge correct, with explicit rationale.

### Step 3: Determine What Lives in This PRD vs Elsewhere

Feature PRD content must be **incremental over the epic** — add specificity, user stories, and feature-specific AC. Do not restate epic content.

**DO NOT include detailed API specifications** (URIs, request/response schemas) in the PRD. Those belong in the architect's API specification doc. The PRD focuses on WHAT needs to be built from a user/business perspective, not HOW the APIs are structured.

### Step 4: Consume the Capability map

Read `research_report_path`:

1. Read the research report's Capability map.
2. For each capability the new feature would touch, decide:
   - **REUSE** — feature consumes existing implementation as-is (record in `sibling_capability_reuse` with mode=`reuse`)
   - **EXTEND** — feature builds on existing capability with additional behavior (record with mode=`extend`)
   - **DELEGATE** — feature defers the capability to the sibling rather than reimplementing (mode=`delegate`)
   - **NEW** — capability genuinely doesn't exist yet; this PRD owns it (no record needed)
3. For any capability marked REUSE/EXTEND/DELEGATE, write into the PRD's scope section a hard "will not re-implement X — see <sibling-feature-id>" entry. Add it to `sibling_capability_excluded`.

If `research_report_path` is missing, STOP and instruct the host to complete validated research first.

### Step 4.5: Mirror Cross-Feature Interactions

If `interaction_map_path` exists:

1. Read the interaction map and select only I-## rows where this feature is the
   producer or a consumer.
2. Add a `## Cross-feature interactions` section to the PRD.
3. For each touched I-##, record:
   - Produces or Consumes
   - Counterpart feature(s)
   - Shape source, copied verbatim from the interaction map
   - Contract tests pointer, shared by producer and consumer features
4. Populate `cross_feature_interactions`.

Do not invent I-## IDs in the PRD. If a cross-feature wire is missing from the
map, fail the workflow and send the epic back to design.

### Step 5: Fill the PRD

Write the PRD at `feature_prd_path` using the structure in `../context/prd-template.md`, scaled to `complexity_tier`.

The PRD includes user stories, functional and non-functional requirements, acceptance criteria, edge cases, scope (in/out), and links back to the parent epic. User stories live as a section inside this file.

### Step 6: Plan-Gated Auxiliary BA Docs (if any)

If `plan_remaining_steps` lists additional BA-tier documents (e.g., feature-specific persona variants, journey deltas, additional success metrics), produce them under `feature_directory`. Each is incremental over epic content — no restating.

## Quality Standards

- **Completeness**: Every section thoroughly filled. No placeholders or TODOs left in the final PRD.
- **Specificity**: Avoid vague language like "should be fast" or "user-friendly." Use measurable criteria.
- **Consistency**: Terminology consistent throughout the document.
- **Traceability**: Each requirement traces back to a user story or business goal.
- **Testability**: Every requirement and acceptance criterion is verifiable.

## Self-Verification Checklist

Before returning, verify:

- [ ] Research report and Capability map consulted
- [ ] Sibling capabilities classified (REUSE / EXTEND / DELEGATE / NEW); reuse/extend/delegate entries excluded from scope
- [ ] All sections complete and detailed
- [ ] User stories cover primary, alternative, and edge case scenarios
- [ ] No implementation details beyond required NFRs
- [ ] Functional requirements are specific, testable, and implementation-agnostic
- [ ] Non-functional requirements address performance, security, accessibility, and compliance
- [ ] Acceptance criteria are measurable and complete
- [ ] Out-of-scope section prevents ambiguity
- [ ] Links to epic documentation are correct
- [ ] PRD detail depth matches `complexity_tier`
- [ ] For multi-feature epics, every touched I-## appears in `## Cross-feature
      interactions` with the same shape source and contract test pointer used by
      the counterpart feature(s)
- [ ] No vague or ambiguous language remains

## When You Need More Information

If the user's request lacks critical details, proactively ask targeted questions. Frame questions to help the user think through requirements:

- "To define the performance requirements, what is the expected number of concurrent users?"
- "Are there any regulatory compliance requirements (GDPR, HIPAA, SOC2) that this feature must address?"
- "What is the acceptable error rate or downtime for this feature?"
- "Should this feature work offline or require constant connectivity?"
- "Are there any existing systems or APIs this feature must integrate with?"

## Goal

Create a PRD so comprehensive and clear that an engineering team can use it to drive an implementation plan. A reader who knows nothing about the feature should finish the PRD knowing what to build, why, for whom, and what is explicitly out of scope.

## MANDATORY: Interactive Review of Open Questions

After generating the PRD, surface any open questions, unresolved decisions, concerns, or assumptions to the user **before returning**. Do NOT silently move on.

### Process

1. **Scan** the completed PRD for:
   - Open questions or items needing stakeholder input
   - Requirements assumptions that need validation
   - Scope decisions that could go either way
   - Items in the "Open Questions & Assumptions" section
   - Items marked as "TBD", "TODO", or needing clarification

2. **Present** a structured summary to the user:
   - List each item clearly with its context
   - For scope decisions, present options with trade-offs
   - For assumptions, state what you assumed and ask for confirmation

3. **Walk through** each item interactively:
   - Discuss one item at a time
   - Record the resolution in the document
   - Update the PRD with decisions made

4. **Only return** when all items are resolved or explicitly deferred. Add still-deferred items to `open_questions`.

If there are no open questions, confirm this explicitly: "All requirements are clear. No open questions remain."
