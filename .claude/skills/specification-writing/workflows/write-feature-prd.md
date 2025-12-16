# Workflow: Write Feature PRD

## Purpose

Transform high-level feature descriptions or enablers from an Epic into detailed, comprehensive PRDs that serve as the single source of truth for engineering teams.

## Core Principles

1. **Clarity Over Brevity**: Be thorough and specific. Ambiguity leads to implementation errors and rework.
2. **User-Centric Thinking**: Always ground requirements in user needs and business outcomes.
3. **Completeness**: Anticipate edge cases, error states, and non-functional requirements.
4. **Actionability**: Every requirement should be testable and verifiable.
5. **Scope Discipline**: Clearly define what is in scope and what is not to prevent scope creep.

## Your Process

### Step 1: Gather Information

Before writing the PRD, ensure you have:
- The parent Epic documentation (PRD and Architecture if available)
- A clear understanding of the feature request _verified_ by the user
- Target user personas (`/docs/personas/`)
- Business context and success metrics

If any critical information is missing, ask specific, targeted questions. For example:
- "What specific user problem does this feature solve?"
- "Are there any performance requirements or constraints?"
- "What are the expected success metrics?"
- "Are there any compliance or security considerations?"
- "What should explicitly NOT be included in this feature?"

### Step 2: Assess Complexity

Before preparing the PRD, determine:
- "How complex is this to implement? [snow-shoveling, ikea furniture, heart-surgery]"
- The PRD should be tailored to that level of detail
- The length of the PRD should correspond to the difficulty of the task
- "Adding a button to a UI" doesn't require the same level of PRD documentation as adding a "new feature to the application"
- Adjust accordingly

### Step 3: Structure the PRD

Create a PRD following the structure defined in `../context/prd-template.md`.

Key sections:
1. Feature Name
2. Epic (with links to parent)
3. Goal (Problem, Solution, Impact)
4. User Personas
5. User Stories
6. Requirements (Functional & Non-Functional)
7. Acceptance Criteria
8. Out of Scope

### Step 4: Save the Document

Save the completed PRD to: `/docs/plan/{epic-key}/{feature-key}/prd.md`

Use the numbering/slug format for directories and file names:
- Epic key: `E##-{epic-slug}`
- Feature key: `E##-F##-{feature-slug}`
- Example: `/docs/plan/E09-identity-platform/E09-F01-oauth-integration/prd.md`

**Note on phased implementation**: If there is a phased implementation to the epic, the phases should be broken out into separate epics to keep the scope reasonable. It could be something like:
- `E09a-F01-oauth-integration`
- `E09b-F01-login-screen`

## Quality Standards

- **Completeness**: Every section must be thoroughly filled out. No placeholders or TODOs.
- **Specificity**: Avoid vague language like "should be fast" or "user-friendly." Use measurable criteria.
- **Consistency**: Ensure terminology is consistent throughout the document.
- **Traceability**: Each requirement should trace back to a user story or business goal.
- **Testability**: Every requirement and acceptance criterion must be verifiable.

## Self-Verification Checklist

Before finalizing the PRD, verify:
- [ ] All sections are complete and detailed
- [ ] User stories cover primary, alternative, and edge case scenarios
- [ ] No implementation details beyond required NFR and as necessary for related feature requirements
- [ ] Functional requirements are specific, testable, and implementation-agnostic
- [ ] Non-functional requirements address performance, security, accessibility, and compliance
- [ ] Acceptance criteria are measurable and complete
- [ ] Out of scope section prevents ambiguity
- [ ] Links to Epic documentation are correct
- [ ] Document is saved to the correct location
- [ ] No vague or ambiguous language remains

## When You Need More Information

If the user's request lacks critical details, proactively ask targeted questions. Frame questions to help the user think through requirements:

- "To define the performance requirements, what is the expected number of concurrent users?"
- "Are there any regulatory compliance requirements (GDPR, HIPAA, SOC2) that this feature must address?"
- "What is the acceptable error rate or downtime for this feature?"
- "Should this feature work offline or require constant connectivity?"
- "Are there any existing systems or APIs this feature must integrate with?"

## Goal

Your goal is to create a PRD so comprehensive and clear that an engineering team can use it to create an implementation plan for execution based on the document.
