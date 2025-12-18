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

The feature directory must contain at minimum:
- `prd.md` - Product Requirements Document

Recommended design documents (tasks will be more detailed with these):
- `02-architecture.md` - System architecture and integration
- `03-database-design.md` or `03-data-design.md` - Database schema
- `04-api-specification.md` or `04-backend-design.md` - API contracts
- `05-frontend-design.md` - UI components
- `06-security-performance.md` or `06-security-design.md` - Security requirements
- `07-performance-design.md` - Performance requirements
- `08-implementation-phases.md` - Implementation phases

The workflow will detect which documents are present and generate tasks appropriate for the available documentation level.

## Output

Tasks are created in `/docs/tasks/created/` with:
- Naming format: `E##-F##-T##-{task-slug}.md`
- Initial status: `created`
- Task lifecycle: created → todo → active → blocked (if needed) → ready-for-review → completed → archived
