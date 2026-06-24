---
inputs:
  - epic_concept: user-provided idea/brief for the epic (string)
  - existing_personas_index: list of {name, file_path, role} for personas already documented (optional)
  - upstream_artifacts: list of {label, path} for any discovery/vision/journey docs informing this epic (optional)
  - sibling_epic_summaries: list of {epic_id, title, scope_summary} for cross-epic awareness (optional)
  - epic_directory: absolute path to the directory where the epic's files should be created
  - epic_main_file_path: absolute path of the main epic index file (epic.md or shark-generated equivalent)
  - plan_decision: object describing which optional documents to include — `{personas: bool, user_journeys: bool, success_metrics: bool, requirements: bool, scope: bool}` (host runs the PLAN gate; craft accepts the decision)
  - plan_remaining_steps: list of document slugs still to be produced (RESUME mode); empty if starting fresh
  - plan_completed_steps: list of document slugs already completed in a prior run
outputs:
  - created_documents: list of {slug, path, title} for every file written
  - decisions_log: list of {decision, rationale, file_referenced} for in-line scope/assumption resolutions
  - open_questions: list of unresolved items needing stakeholder input (description, file_referenced, options_with_tradeoffs)
  - personas_referenced: list of persona names actually used in the epic
  - cross_references_verified: bool — true once link integrity check passed
---

# Workflow: Write Epic PRD (craft)

## Purpose

Create a comprehensive Epic-level Product Requirements Document using a modular, multi-file architecture that improves navigability and maintainability.

## Your Approach

### 1. Information Gathering

When a user presents an epic idea, first assess whether you have sufficient information. If critical details are missing, ask targeted clarifying questions about:

- Target users and their pain points
- Expected business outcomes and success metrics
- Key workflows and user journeys
- Technical or business constraints
- Integration requirements with existing systems
- Timeline or priority considerations

### 2. Structured Thinking

Before writing, mentally organize the epic into modular components:

- **Core summary** (epic name, goal, business value) → stays in main index
- **User context** (personas, journeys) → separate files for detail
- **Requirements** (functional + non-functional) → consolidated requirements file
- **Measurement** (success metrics, KPIs) → dedicated metrics file
- **Boundaries** (out of scope) → explicit scope file

### 3. Quality Standards

Your PRDs must:

- Be specific and actionable, avoiding vague language
- Include concrete examples where helpful
- Anticipate edge cases and error scenarios
- Define clear success metrics that are measurable
- Establish explicit boundaries to prevent scope creep
- Reference existing personas when available
- Maintain consistent cross-references between files

## Multi-File Architecture

The set of files you create is determined by `plan_decision` and `plan_remaining_steps`. Always required: a main index (`epic.md`), a requirements catalog, and a scope/exclusions file. Other files are conditional.

### Always-Required Files

1. **Main index** — concise overview, epic goal, business value, navigation to detail files
2. **Requirements catalog** — comprehensive functional and non-functional requirements
3. **Scope** — explicit boundaries and exclusions

### Conditionally Required Files (per `plan_decision`)

- **Personas** — only if user-facing epic with distinct user types
- **User journeys** — only if distinct multi-step user workflows exist
- **Success metrics** — only if externally measurable business outcomes are claimed

## File Templates

For detailed structure of each file, see:

- **Epic structure**: `../context/epic-template.md`
- **Naming conventions**: `../context/naming-conventions.md`

## Implementation Protocol

### Step 1 — Confirm Information Sufficiency

Engage with the user, ask clarifying questions, and confirm you have enough to produce a substantive PRD. Do not proceed if the epic concept is too vague to produce concrete requirements.

### Step 2 — Resolve the Document Set

If `plan_remaining_steps` is non-empty, treat this as RESUME mode and proceed only with those documents. If empty, the host has already produced `plan_decision` covering every optional document — every INCLUDE/EXCLUDE call must be justified with a specific reason for THIS epic, and exclusions get logged in `decisions_log`.

Default to INCLUDE when uncertain.

### Step 3 — Fill the Main Index First

Write the main epic index at `epic_main_file_path`. Keep it concise — an executive should be able to read it in 3 minutes. The index lists business value, summary, and links out to detail files.

### Step 4 — Produce Each Planned Detail File

For each slug in `plan_remaining_steps`, create the corresponding file under `epic_directory`. After each file is complete, append it to `created_documents`.

Each detail file should:

- Provide depth without redundancy — do not repeat content from the index
- Use consistent terminology with the index and other detail files
- Include cross-references where concepts span files

### Step 5 — Cross-Reference Verification

Verify:

1. All relative links work correctly.
2. Personas are referenced consistently across files.
3. Requirements trace back to user journeys (where journeys exist).
4. Metrics align with requirements (where metrics exist).
5. File names match exactly in links (case-sensitive).

Set `cross_references_verified = true` once these checks pass.

### Step 6 — Quality Check

Run through the self-verification checklist before returning.

## Best Practices

### Navigation & Linking

- **Always use relative links** between files (e.g., `./personas.md`) for portability.
- **Link bidirectionally**: if the index links to requirements, requirements should link back.
- **Use descriptive link text**: instead of "click here", use "see [User Journeys](./user-journeys.md)".

### Content Distribution

- **Main index**: keep it concise.
- **Detail files**: provide depth without redundancy — don't repeat index content.
- **Cross-references**: when concepts span files, use links rather than duplicating content.

### Consistency

- **Terminology**: use identical terms across all files (if it's "user persona" in one file, don't call it "user profile" in another).
- **Requirement IDs**: use consistent prefixes (REQ-F for functional, REQ-NF for non-functional).
- **Persona names**: ensure persona names/roles match exactly across all files.

### Maintainability

- **Date stamps**: include "Last Updated" in the main index.
- **Independent evolution**: each file can evolve independently — keep them synchronized.
- **Orphan prevention**: if you remove content from one file, update references in other files.

### Examples & Specificity

- **Replace vague language**: "Users can manage settings" → "Users can view, edit, and delete their notification preferences from the Settings page".
- **Provide concrete examples**: when requirements might be ambiguous, add an example scenario.
- **Quantify where possible**: "Fast response time" → "Page load < 2 seconds on 3G connection".

## Self-Verification Checklist

Before returning, verify:

### Content Completeness

- [ ] Only planned documents created (no extras, no omissions vs `plan_remaining_steps`)
- [ ] Exclusion decisions captured in `decisions_log` for every optional doc not produced
- [ ] Epic name is clear and descriptive (3–6 words, title case)
- [ ] Problem statement is specific and compelling
- [ ] User personas are well-defined or properly referenced
- [ ] User journeys cover both happy and unhappy paths
- [ ] Functional requirements are comprehensive, testable, and prioritized
- [ ] Non-functional requirements address security, performance, accessibility, and scalability
- [ ] Success metrics are measurable, time-bound, and have clear targets
- [ ] Out-of-scope items prevent ambiguity and set clear boundaries
- [ ] Business value justification is data-informed

### Structure & Navigation

- [ ] Main index serves as effective hub with navigation to all sections
- [ ] All relative links between files are correct and functional
- [ ] Each detail file links back to the index
- [ ] Cross-references between files are accurate (e.g., requirements reference specific journeys)
- [ ] File names match exactly in links (case-sensitive)

### Quality Standards

- [ ] All content is specific and actionable, avoiding vague language
- [ ] Concrete examples provided where helpful
- [ ] Edge cases and error scenarios anticipated
- [ ] Terminology is consistent across all files
- [ ] No content duplication between files
- [ ] Each file stands alone while supporting the broader epic narrative

### Persona Integration

- [ ] Checked `existing_personas_index` for existing personas before authoring new ones
- [ ] Personas referenced consistently by name across all files

## MANDATORY: Interactive Review of Open Questions

After generating all epic files, surface any open questions, unresolved decisions, concerns, or assumptions to the user **before returning**. Do NOT silently move on.

### Process

1. **Scan** all completed documents for:
   - Open questions or items needing stakeholder input
   - Requirements assumptions that need validation
   - Scope decisions that could go either way
   - Items in the "Open Questions & Assumptions" section of the index
   - Items marked as "TBD", "TODO", or needing clarification across all files

2. **Present** a structured summary to the user:
   - List each item clearly with its context and which file it appears in
   - For scope decisions, present options with trade-offs
   - For assumptions, state what you assumed and ask for confirmation

3. **Walk through** each item interactively:
   - Discuss one item at a time
   - Record the resolution in the appropriate document
   - Update the files with decisions made

4. **Only return** when all items are resolved or explicitly deferred by the user. Add still-deferred items to `open_questions`.

If there are no open questions, confirm this explicitly: "All epic-level decisions are resolved. No open questions remain."
