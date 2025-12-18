# Workflow: Write Epic PRD

## Purpose

Create comprehensive Epic-level Product Requirements Documents using a modular, multi-file architecture that improves navigability and maintainability.

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
- Reference existing personas when available in `/docs/personas/`
- Maintain consistent cross-references between files

## Multi-File Architecture

You will create **6 interconnected files** in `/docs/plan/{epic-key}/`, where `{epic-key}` is `E##-{epic-slug}` (two-digit epic number + hyphenated epic name, e.g., `E09-identity-platform`).

### File Creation Sequence

Follow this order to ensure proper dependencies and cross-references:

1. **epic.md** - Main index/reference file (create first)
2. **personas.md** - User personas detail
3. **user-journeys.md** - User workflow narratives
4. **requirements.md** - Comprehensive requirements catalog
5. **success-metrics.md** - KPIs and measurement framework
6. **scope.md** - Boundaries and exclusions

## File Templates

For detailed structure of each file, see:
- **Epic structure**: `../context/epic-template.md`
- **Naming conventions**: `../context/naming-conventions.md`

## Implementation Protocol

### Step-by-Step File Creation Process

**Phase 1: Information Gathering**
1. Engage with user to understand epic concept
2. Ask clarifying questions as needed
3. Confirm you have sufficient information to proceed

**Phase 2: Create Epic in Shark Database**
1. Run `shark epic create --key=<epic-key> --title="<Epic Name>" --status=draft`
   - This creates the database record and generates the epic directory structure
   - Epic key format: `E##-{epic-slug}` (e.g., `E09-identity-platform`)
   - Shark will create `/docs/plan/{epic-key}/` and initialize `epic.md` with basic structure
   - **Note**: If shark epic create is not yet implemented, manually create the directory and files

2. Verify the epic was created: `shark epic get <epic-key>`

**Phase 3: Fill in Epic Files with Detailed Content**
1. **Fill `epic.md`** - Expand the generated file with comprehensive goal, business value, quick reference
2. **Create and fill `personas.md`** - Add detailed user persona profiles
3. **Create and fill `user-journeys.md`** - Map detailed user workflows
4. **Create and fill `requirements.md`** - Add comprehensive functional and non-functional requirements
5. **Create and fill `success-metrics.md`** - Define detailed KPIs and measurement framework
6. **Create and fill `scope.md`** - Establish explicit boundaries and exclusions

**Phase 4: Verify Epic in Database**
1. Confirm epic exists: `shark epic get <epic-key>`
2. Verify epic shows in list: `shark epic list`

**Phase 5: Cross-Reference Verification**
1. Verify all relative links work correctly
2. Ensure personas are referenced consistently across files
3. Confirm requirements trace back to user journeys
4. Validate metrics align with requirements

**Phase 6: Quality Check**
Run through the self-verification checklist before finalizing.

## Best Practices

### Navigation & Linking
- **Always use relative links** between files (e.g., `./personas.md`) for portability
- **Link bidirectionally**: If epic.md links to requirements.md, requirements.md should link back
- **Use descriptive link text**: Instead of "click here", use "see [User Journeys](./user-journeys.md)"

### Content Distribution
- **Main epic.md**: Keep it concise—executives should be able to read it in 3 minutes
- **Detail files**: Provide depth without redundancy—don't repeat content from epic.md
- **Cross-references**: When concepts span files, use links rather than duplicating content

### Consistency
- **Terminology**: Use identical terms across all files (if it's "user persona" in one file, don't call it "user profile" in another)
- **Requirement IDs**: Use consistent prefixes (REQ-F for functional, REQ-NF for non-functional)
- **Persona names**: Ensure persona names/roles match exactly across all files

### Maintainability
- **Date stamps**: Include "Last Updated" in epic.md
- **Version control**: Each file can evolve independently but keep them synchronized
- **Orphan prevention**: If you remove content from one file, update references in other files

### Examples & Specificity
- **Replace vague language**: "Users can manage settings" → "Users can view, edit, and delete their notification preferences from the Settings page"
- **Provide concrete examples**: When requirements might be ambiguous, add an example scenario
- **Quantify where possible**: "Fast response time" → "Page load < 2 seconds on 3G connection"

## Self-Verification Checklist

Before delivering the PRD, verify:

### Content Completeness
- [ ] All 6 files created in `/docs/plan/{epic-key}/` directory
- [ ] Epic name is clear and descriptive (3-6 words, title case)
- [ ] Problem statement is specific and compelling
- [ ] User personas are well-defined or properly referenced
- [ ] User journeys cover both happy and unhappy paths
- [ ] Functional requirements are comprehensive, testable, and prioritized
- [ ] Non-functional requirements address security, performance, accessibility, and scalability
- [ ] Success metrics are measurable, time-bound, and have clear targets
- [ ] Out of scope items prevent ambiguity and set clear boundaries
- [ ] Business value justification is data-informed

### Structure & Navigation
- [ ] `epic.md` serves as effective index with clear navigation to all sections
- [ ] All relative links between files are correct and functional
- [ ] Each detail file links back to `epic.md`
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
- [ ] Checked for existing personas in `/docs/personas/`
- [ ] If creating new personas, saved to `/docs/personas/{persona-name-desc}.md`
- [ ] Personas referenced consistently by name across all files

## Output Confirmation

After creating all files, provide the user with:

1. **Confirmation message**: "Epic PRD created successfully:"
   - Epic created in database: `shark epic get {epic-key}`
   - Files created in `/docs/plan/{epic-key}/`
2. **File list**: List all 6 files created
3. **Database verification**: Show output of `shark epic get {epic-key}` to confirm epic is tracked
4. **Navigation instruction**: "Start with `epic.md` for overview, then navigate to detail files as needed"
5. **Next steps**: Suggest workflow options:
   - "Create features for this epic: `shark feature create --epic={epic-key} --key={feature-key} --title='Feature Name'`"
   - "Generate tasks with `/task` command once feature design docs are ready"
   - "This PRD is ready to be used as input for technical architecture specifications"
