---
description: Generate agent-executable implementation tasks from design documents
---

# Generate Tasks

Create agent-executable implementation tasks from existing technical design documentation.

## Usage

```bash
/task {epic-key} {feature-key}
```

**Arguments**:
- `epic-key`: Epic directory name (e.g., E01-claude-reorg)
- `feature-key`: Feature directory name (e.g., E01-F01-skill-extraction)

**Example:**
```bash
/task E01-claude-reorg E01-F01-skill-extraction
```

## Implementation

This command invokes the specification-writing skill's task generation workflow.

**Skill**: `~/.claude/skills/specification-writing/workflows/write-task.md`
**Context**: `~/.claude/skills/specification-writing/context/task-template.md`

## What This Does

The skill will:
1. Read all technical design documents in the feature directory
2. Validate contract consistency across frontend/backend/database
3. Break down implementation into logical phases
4. Generate focused tasks that:
   - Define WHAT to build and WHY (not detailed HOW)
   - Reference design documents for implementation details
   - Specify success criteria and validation gates
   - Document dependencies and integration points
5. Create tasks in `/docs/tasks/created/`

## Prerequisites

The feature directory must contain design documents:
- `02-architecture.md`
- `03-database-design.md`
- `04-api-specification.md`
- `05-frontend-design.md`
- `06-security-performance.md`
- `07-implementation-phases.md`

## Output

Tasks are created in `/docs/tasks/created/` with:
- Naming format: `E##-F##-T##-{task-slug}.md`
- Initial status: `created`
- Task lifecycle: created → todo → active → blocked (if needed) → ready-for-review → completed → archived
