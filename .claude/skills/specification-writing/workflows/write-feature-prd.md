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

### Step 3: Create Feature in Shark Database

Before writing the PRD, create the feature in the database:

1. Run `shark feature create --epic=<epic-key> --key=<feature-key> --title="<Feature Name>" --status=draft`
   - Epic key: `E##-{epic-slug}` (e.g., `E09-identity-platform`)
   - Feature key: `E##-F##-{feature-slug}` (e.g., `E09-F01-oauth-integration`)
   - Shark will create `/docs/plan/{epic-key}/{feature-key}/` and initialize `prd.md` with basic structure
   - **Note**: If shark feature create is not yet implemented, manually create the directory and `prd.md` file

2. Verify the feature was created: `shark feature get <feature-key>`

### Step 4: Fill in the PRD with Detailed Content

Expand the generated `prd.md` following the structure defined in `../context/prd-template.md`.

Fill in all sections with comprehensive detail:
1. Feature Name (already set)
2. Epic (add links to parent epic documentation)
3. Goal (Problem, Solution, Impact with specific metrics)
4. User Personas (detailed profiles or references to existing personas)
5. User Stories (categorized as Must/Should/Could Have)
6. Requirements (Functional & Non-Functional with specific acceptance criteria)
7. Acceptance Criteria (testable, measurable criteria)
8. Out of Scope (explicit exclusions and future considerations)

Location: `/docs/plan/{epic-key}/{feature-key}/prd.md`

**Note on phased implementation**: If there is a phased implementation to the epic, the phases should be broken out into separate epics to keep the scope reasonable. It could be something like:
- `E09a-F01-oauth-integration`
- `E09b-F01-login-screen`

## Quality Standards

- **Completeness**: Every section must be thoroughly filled out. No placeholders or TODOs.
- **Specificity**: Avoid vague language like "should be fast" or "user-friendly." Use measurable criteria.
- **Consistency**: Ensure terminology is consistent throughout the document.
- **Traceability**: Each requirement should trace back to a user story or business goal.
- **Testability**: Every requirement and acceptance criterion must be verifiable.

### Step 5: Verify Feature in Database

After completing the PRD:
1. Confirm feature exists: `shark feature get <feature-key>`
2. Verify feature shows in epic's feature list: `shark feature list --epic=<epic-key>`
3. Check PRD file location matches database file_path

## Self-Verification Checklist

Before finalizing the PRD, verify:
- [ ] Feature created in shark database: `shark feature get <feature-key>` returns the feature
- [ ] All sections are complete and detailed
- [ ] User stories cover primary, alternative, and edge case scenarios
- [ ] No implementation details beyond required NFR and as necessary for related feature requirements
- [ ] Functional requirements are specific, testable, and implementation-agnostic
- [ ] Non-functional requirements address performance, security, accessibility, and compliance
- [ ] Acceptance criteria are measurable and complete
- [ ] Out of scope section prevents ambiguity
- [ ] Links to Epic documentation are correct
- [ ] Document is saved to the correct location: `/docs/plan/{epic-key}/{feature-key}/prd.md`
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

## Output Confirmation

After completing the PRD, provide the user with:

1. **Confirmation message**: "Feature PRD created successfully:"
   - Feature created in database: `shark feature get {feature-key}`
   - PRD file: `/docs/plan/{epic-key}/{feature-key}/prd.md`
2. **Database verification**: Show that feature is tracked in shark
3. **Next steps**: Suggest workflow options:
   - "Create architecture design docs for this feature (if needed)"
   - "Generate implementation tasks: `/task {epic-key}/{feature-key}` once design docs are ready"
   - "Create tasks directly: `shark task create --epic={epic-key} --feature={feature-key} --title='Task Name' --agent=backend`"
