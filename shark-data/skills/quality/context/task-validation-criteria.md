# Task Validation Criteria

This document defines the complete validation criteria for Product Requirement Prompts (Tasks).

## Required Frontmatter

Each Task file (except README.md) must have valid YAML frontmatter:

```yaml
---
status: created | todo | active | blocked | ready-for-review | completed | archived
feature: /docs/plan/{epic}/{feature}  # Absolute path
created: YYYY-MM-DD
assigned_agent: api-developer | frontend-developer | devops-engineer | general-purpose
dependencies: [list-of-task-files.md] | []  # Other task files in /docs/tasks/
estimated_time: X hours  # Typically 2-12 hours
---
```

### Validation Checks:
- [ ] All fields present
- [ ] Status is "created" initially
- [ ] Feature path is valid absolute path
- [ ] Assigned agent is valid agent name
- [ ] Dependencies list is valid (files exist in /docs/tasks/ or empty list)
- [ ] Estimated time is reasonable (2-12 hours typical)

## Required Sections

Each Task must have these sections:

### 1. Title
- Format: `# Task: {Component Name}`
- Clear, descriptive component name
- Example: `# Task: Contract Validation`

### 2. Goal
- 1-2 sentences
- States WHAT will be built and WHY it matters
- High-level objective

### 3. Success Criteria
- Minimum 3-5 measurable checkboxes
- Specific, testable outcomes
- Must include:
  - [ ] Component-specific validation
  - [ ] All validation gates pass
  - [ ] Documentation updates (IMPLEMENTATION.md, TODO.md)
  - [ ] Artifacts from design docs created/updated

### 4. Implementation Guidance
Must include:
- **Overview** (2-3 paragraphs)
- **Key Requirements** (5-8 items with design doc references)
- **Files to Create/Modify** (list of file paths, no code)
- **Integration Points** (how component integrates)

Optional subsections:
- Codebase Analysis Results (for API/Frontend tasks)
- Contract Specifications (for API/Frontend tasks)
- Data Flow (for API/Frontend tasks)

### 5. Validation Gates
- What needs to be validated (not how)
- Linting and type checking requirements
- Unit test requirements
- Integration test requirements
- Performance requirements
- Security requirements

### 6. Context & Resources
- Direct links to design doc sections in /docs/plan/{epic}/{feature}/
- Format: `[Section Name](../../plan/{epic}/{feature}/0X-doc-name.md#section)`

### 7. Notes for Agent
- 3-5 bullet points
- Patterns to follow
- Edge cases
- Performance considerations
- Security requirements

## Content Quality Requirements

### High-Level, Not Code (CRITICAL)
Tasks are directives, NOT tutorials:

**IMPORTANT**: Tasks are high-level work specifications that tell agents WHAT to build, not HOW to code it.

**NEVER include**:
- ❌ SQL statements, DDL, migrations
- ❌ Python, TypeScript, JavaScript code
- ❌ Bash scripts, shell commands
- ❌ Configuration files (YAML, JSON)
- ❌ Step-by-step implementation instructions
- ❌ Line-by-line coding tutorials

**ONLY include**:
- ✅ Clear goal and success criteria
- ✅ WHAT needs to be built
- ✅ WHY it's needed
- ✅ References to design docs
- ✅ List of files that will change
- ✅ Integration points
- ✅ Validation requirements

### Proper Length
- Target: 50-100 lines (excluding frontmatter)
- Too brief (< 40 lines): Missing context
- Too verbose (> 120 lines): Duplicating design docs or including code

### Design Doc References
- Must link to relevant design doc sections
- Format: `[Section](../0X-doc.md#section)`
- No duplicated content from design docs
- Reference instead of repeat

### Agent Assignment
Appropriate specialized agent:
- Database Tasks → `general-purpose` or `api-developer`
- API Tasks → `api-developer`
- Frontend Tasks → `frontend-developer`
- Deployment Tasks → `devops-engineer`
- Integration Tasks → `general-purpose`
- Contract validation → `api-developer, frontend-developer` (parallel)

## Dependency Validation

### Valid Dependency Chain
Dependencies must form logical execution sequence:

1. Contract Validation (P00) - No dependencies
2. Database Setup (P01) - Depends on P00
3. API Implementation (P02) - Depends on P00, P01
4. Frontend Development (P03) - Depends on P00, P02
5. Integration Testing (P04) - Depends on P03
6. Deployment (P05) - Depends on P04

### Dependency Rules
- [ ] No circular dependencies
- [ ] All referenced dependency files exist
- [ ] Database Tasks have minimal dependencies
- [ ] API Tasks depend on Database Tasks
- [ ] Frontend Tasks depend on API Tasks
- [ ] Integration Tasks depend on Frontend Tasks
- [ ] Deployment Tasks depend on Integration Tasks

### Validation Checks
- Build dependency graph
- Detect circular references
- Verify logical execution order
- Calculate critical path

## README.md Index Requirements

The `tasks/README.md` must include:

- [ ] **Active Tasks** table with columns:
  - Task name (linked)
  - Status
  - Assigned Agent
  - Dependencies
  - Estimated Time
- [ ] Table matches actual Task files
- [ ] **Workflow** section with execution order
- [ ] **Status Definitions** section
- [ ] Links to all design documents
- [ ] No longer says "Tasks not yet generated"

## Success Criteria Quality

Good success criteria are:
- ✅ Specific and measurable
- ✅ Testable by validation gates
- ✅ Complete (cover all aspects)
- ✅ Actionable (clear what to check)

**Good Example**:
```markdown
## Success Criteria
- [ ] Database tables created with all fields from design spec
- [ ] Row-level security policies enforced per security design
- [ ] Indexes created for common query patterns per DB design
- [ ] All migrations run successfully without errors
- [ ] All validation gates pass (lint, unit tests, integration tests)
- [ ] IMPLEMENTATION.md created with implementation summary
- [ ] TODO.md created with outstanding items
```

**Bad Example**:
```markdown
## Success Criteria
- [ ] Database done
- [ ] Tests pass
- [ ] Looks good
```

## Pass/Fail Thresholds

### READY ✅
- All Tasks exist with README
- Valid frontmatter in all Tasks
- All required sections present
- No code implementation
- Proper length (50-100 lines)
- Design doc references valid
- Valid dependency chain
- No circular dependencies
- README index complete
- Success criteria specific

### READY WITH WARNINGS ⚠️
- Minor issues that don't block activation:
  - Missing optional design doc references
  - Success criteria could be more specific
  - Length slightly outside range (40-120 lines)
  - Minor style inconsistencies

### NOT READY ❌
- Blocking issues that prevent activation:
  - Missing required Tasks
  - Invalid frontmatter
  - Missing required sections
  - Code implementation found
  - Vague or incomplete success criteria
  - Circular dependencies
  - Broken design doc references
  - Severe length violations (< 30 or > 150 lines)

## Common Issues and Fixes

| Issue | Severity | Fix |
|-------|----------|-----|
| Circular dependencies | Blocker | Restructure to flow one direction |
| Code found in Task | Blocker | Replace with high-level requirements |
| Missing design doc references | Major | Add links to relevant sections |
| Vague success criteria | Major | Make specific and measurable |
| Wrong agent assignment | Major | Assign to specialized agent |
| Missing success criteria items | Major | Include validation gates, docs updates |
| Length too short/long | Minor | Adjust detail level |
| Style inconsistencies | Minor | Standardize formatting |
