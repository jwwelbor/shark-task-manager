# Workflow: Validate Task Readiness

## Purpose

Verify that all Product Requirement Prompts (Tasks) for a feature are complete, properly structured, and ready for implementation agents.

## Usage

This workflow is invoked when validating task readiness:
- By `/validate-task-readiness` command
- By task-generator agent to validate its own output
- Before moving tasks from `/docs/tasks/created/` to `/docs/tasks/todo/`

## What This Workflow Checks

### 1. Required Tasks Exist

Verify tasks are present in `/docs/tasks/created/`:
- `README.md` - Task index and workflow
- Component tasks (database, API, frontend, integration, deployment)

**Note**: Number and names vary based on feature complexity.

### 2. Task Structure Validation

For each task file, verify frontmatter and sections. See `../context/task-validation-criteria.md` for detailed requirements.

**Frontmatter must include**:
- status (created initially)
- feature (absolute path)
- created (YYYY-MM-DD)
- assigned_agent (valid agent name)
- dependencies (list or empty)
- estimated_time (hours)

**Required sections**:
- Goal (1-2 sentences)
- Success Criteria (checkboxes)
- Implementation Guidance
- Validation Gates
- Context & Resources
- Notes for Agent

### 3. Content Quality Checks

For each task, verify:
- **High-Level, Not Code**: No SQL, Python, TypeScript, or implementation tutorials
- **Proper Length**: 50-100 lines (excluding frontmatter)
- **Design Doc References**: Links to relevant design doc sections
- **Agent Assignment**: Appropriate specialized agent

### 4. Dependency Validation

Check that Task dependencies form a valid execution sequence:
- No circular dependencies
- All referenced dependencies exist
- Logical execution order (database → API → frontend → integration → deployment)

### 5. README Index Validation

Check `/docs/tasks/created/README.md`:
- Contains "Active Tasks" table
- Table matches actual task files
- Contains workflow/execution order section
- Contains status definitions (created, todo, active, blocked, ready-for-review, completed, archived)
- Links to all design documents

### 6. Success Criteria Completeness

For each Task's success criteria:
- At least 3-5 measurable checkboxes
- Specific, testable outcomes
- Includes validation gates passing
- Includes documentation updates

## Execution Steps

### Step 1: Validate File Existence

Use Glob to find all Tasks, then read all in parallel.

### Step 2: Parse and Validate Each Task

For each Task:
1. Extract and validate YAML frontmatter
2. Check all required sections present
3. Verify content quality (no code, proper length)
4. Validate design doc references
5. Check success criteria completeness

### Step 3: Build Dependency Graph

Create a dependency graph from all Tasks:
- Extract dependencies from frontmatter
- Verify dependency files exist
- Check for circular dependencies
- Ensure logical execution order

### Step 4: Validate README Index

Check that README.md:
- Lists all Task files correctly
- Shows correct dependencies
- Has proper workflow section
- Links to design documents

### Step 5: Generate Validation Report

Create `/docs/tasks/created/validation-report.md` with:
- Summary
- Task inventory table
- Dependency graph visualization
- Detailed validation results for each task
- Issues summary (errors and warnings)
- Readiness assessment (ready to move to /docs/tasks/todo/)
- Next steps

### Step 6: Return Summary

Output concise summary with:
- Status (READY/READY WITH WARNINGS/NOT READY)
- Number of tasks validated
- Issues found
- Total estimated time
- Link to full report
- Next steps (move to /docs/tasks/todo/ if ready)

## Success Criteria

Validation passes when:
1. All expected Tasks exist
2. All Tasks have valid YAML frontmatter
3. All required sections present
4. No code implementation found
5. All design doc references valid
6. Dependency chain valid (no circular dependencies)
7. README index matches actual Tasks
8. Success criteria are specific and measurable
9. All Tasks assigned to appropriate agents
10. Task lengths are reasonable (50-100 lines)

## Common Issues

- **Circular Dependencies**: Restructure to flow one direction
- **Code Found in Task**: Replace with high-level requirements
- **Missing Design Doc References**: Add links to design docs in /docs/plan/{epic}/{feature}/
- **Vague Success Criteria**: Make specific, measurable, testable
- **Wrong Agent Assignment**: Use specialized agent for component type
- **Missing Success Criteria Items**: Include validation gates, documentation updates

## Output Format

The validation report should:
- Show dependency graph visually
- List all Tasks with status
- Highlight issues with severity (error/warning)
- Provide actionable fixes
- Calculate total estimated time
- Determine activation readiness

See `../context/task-validation-criteria.md` for complete validation criteria.
