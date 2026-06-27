---
inputs:
  - task_specs: list of {task_id, spec_path, ac_list, depends_on, assigned_agent, estimated_time}
  - feature_prd_path: absolute path to the parent feature PRD markdown (used for design-doc reference checks)
  - design_doc_paths: list of absolute paths to design docs that tasks may reference
  - interaction_map_path: absolute path to parent `<epic-id>-interaction-map.md` if present
  - tasks_index_path: absolute path to the tasks index README (e.g. tasks/created/README.md)
  - validation_report_path: absolute path where the validation report markdown should be written
outputs:
  - validation_report: structured markdown written to validation_report_path
  - blockers: list of {task_id, issue_type, description}
  - warnings: list of {task_id, issue_type, description}
  - dependency_graph: list of {task_id, depends_on, valid: bool, cycle_detected: bool}
  - integration_coverage: list of {contract_id, producer_task, consumer_tasks, shape_source, contract_test_pointer, status}
  - total_estimated_time_hours: number
  - verdict: READY | READY_WITH_WARNINGS | NOT_READY
---

# Workflow: Validate Task Readiness (craft)

## Purpose

Verify that all Tasks (Product Requirement Prompts) for a feature are complete, properly structured, and ready for implementation agents.

## What This Workflow Checks

### 1. Required Tasks Exist

Verify each task in `task_specs` has a corresponding spec file and that the tasks index (`tasks_index_path`) lists them all.

**Note**: Number and names of tasks vary based on feature complexity.

### 2. Task Structure Validation

For each task spec, verify YAML frontmatter and required sections. See `../context/task-validation-criteria.md` for detailed requirements.

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
- **Design Doc References**: Links to relevant design doc sections (must resolve against `design_doc_paths`)
- **Agent Assignment**: Appropriate specialized agent for the task type

### 4. Dependency Validation

Check that task dependencies form a valid execution sequence:
- No circular dependencies
- All referenced dependencies exist (every entry in `depends_on` must match a task in `task_specs`)
- Logical execution order (database → API → frontend → integration → deployment)

Populate `dependency_graph` with one entry per task showing whether its dependencies resolve and whether a cycle was detected.

### 5. Index Validation

Check that the tasks index at `tasks_index_path`:
- Contains an "Active Tasks" table
- Table matches actual task files in `task_specs`
- Contains a workflow / execution-order section
- Contains status definitions (created, todo, active, blocked, ready-for-review, completed, archived)
- Links to all design documents in `design_doc_paths`

### 6. Success Criteria Completeness

For each task's success criteria:
- At least 3-5 measurable checkboxes
- Specific, testable outcomes
- Includes validation gates passing
- Includes documentation updates

### 7. Integration Coverage

For STANDARD/COMPLEX features:

- Every internal CONTRACT-### appears in exactly one producer task and at least
  one consumer task.
- Every I-## the feature PRD declares under "Cross-feature interactions" appears
  in the relevant task spec(s) under `Integration Contracts > Cross-feature`.
- Producer and consumer cite the same shape source.
- Each contract has a single contract-test pointer that both sides reference
  verbatim.
- No orphan contracts. Missing producer, missing consumer, or mismatched pointer
  is a blocker.

## Execution Steps

### Step 1: Validate File Existence

For each task in `task_specs`, verify `spec_path` exists. Read all task files in parallel.

### Step 2: Parse and Validate Each Task

For each task:
1. Extract and validate YAML frontmatter
2. Check all required sections present
3. Verify content quality (no code, proper length)
4. Validate design doc references resolve against `design_doc_paths`
5. Check success criteria completeness
6. Check integration coverage for CONTRACT-### and I-## rows

Each finding is added to `blockers` (severity error) or `warnings` (severity warning).

### Step 3: Build Dependency Graph

Create a dependency graph from all tasks:
- Extract dependencies from frontmatter (`depends_on`)
- Verify dependency targets exist in `task_specs`
- Check for circular dependencies (DFS cycle detection)
- Ensure logical execution order (database before API before frontend, etc.)

Populate `dependency_graph` with the result. Cycles or missing targets become `blockers`.

### Step 4: Validate Index

Check that the tasks index correctly lists all tasks, dependencies, and links to design docs. Mismatches between the index and `task_specs` become `blockers` (if missing) or `warnings` (if out-of-date wording).

### Step 5: Generate Validation Report

Write a validation report to `validation_report_path` with:
- Summary
- Task inventory table
- Dependency graph visualization
- Integration coverage matrix for CONTRACT-### and I-## rows
- Detailed validation results for each task
- Issues summary (errors and warnings)
- Readiness assessment
- Total estimated time (sum of `estimated_time` over `task_specs`)
- Next steps

### Step 6: Decide Verdict

- **READY** when: all expected tasks exist, all have valid frontmatter, all required sections present, no code in tasks, all design doc references valid, dependency chain valid (no cycles), index matches tasks, success criteria are specific and measurable, all tasks assigned to appropriate agents, lengths are reasonable.
- **READY_WITH_WARNINGS** when: only `warnings` remain.
- **NOT_READY** when: any `blockers` (missing task, invalid frontmatter, missing sections, circular dependencies, code in task, missing required references).
  Missing or mismatched I-## mirrors are blockers.

## Success Criteria

Validation passes when:
1. All expected tasks exist
2. All tasks have valid YAML frontmatter
3. All required sections present
4. No code implementation found
5. All design doc references valid
6. Dependency chain valid (no circular dependencies)
7. Index matches actual tasks
8. Success criteria are specific and measurable
9. All tasks assigned to appropriate agents
10. Task lengths are reasonable (50-100 lines)
11. Every I-## from the feature PRD is mirrored in task specs with the same
    shape source and contract test pointer

## Common Issues

- **Circular Dependencies**: Restructure to flow one direction
- **Code Found in Task**: Replace with high-level requirements
- **Missing Design Doc References**: Add links to design docs
- **Vague Success Criteria**: Make specific, measurable, testable
- **Wrong Agent Assignment**: Use specialized agent for component type
- **Missing Success Criteria Items**: Include validation gates, documentation updates

## Output Format

The validation report should:
- Show dependency graph visually
- List all tasks with status
- Highlight issues with severity (error/warning)
- Provide actionable fixes
- Calculate total estimated time
- Determine activation readiness

See `../context/task-validation-criteria.md` for complete validation criteria.
