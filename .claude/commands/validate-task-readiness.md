---
description: Validate that all tasks are complete, properly sequenced, and ready for implementation
---

# Validate Task Readiness

Verify that all implementation tasks are complete, properly structured, and ready for implementation agents.

## Usage

```bash
/validate-task-readiness
```

**Example:**
```bash
/validate-task-readiness
```

## Implementation

This command invokes the quality skill's task validation workflow.

**Skill**: `~/.claude/skills/quality/workflows/validate-tasks.md`
**Context**: `~/.claude/skills/quality/context/task-validation-criteria.md`

## What This Does

The skill will:
1. Find all tasks in `/docs/tasks/created/`
2. Validate each task's structure (frontmatter, sections, content quality)
3. Check dependency graph for circular dependencies
4. Verify agent assignments are appropriate
5. Ensure success criteria are specific and measurable
6. Generate validation report

## Validation Checks

- **File Existence**: All expected tasks present in `/docs/tasks/created/`
- **Structure**: Valid YAML frontmatter and required sections
- **Content Quality**: High-level directives (no code), proper length
- **Dependencies**: Valid execution sequence, no circular dependencies
- **Agent Assignment**: Appropriate specialized agents assigned
- **Success Criteria**: At least 3-5 measurable checkboxes

## Output

Generates `/docs/tasks/created/validation-report.md` with:
- Summary and status (READY/READY WITH WARNINGS/NOT READY)
- Task inventory table
- Dependency graph visualization
- Detailed validation results for each task
- Issues summary with actionable fixes
- Next steps (move to /docs/tasks/todo/ if ready)
