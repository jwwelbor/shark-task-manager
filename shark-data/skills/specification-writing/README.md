# Specification Writing Skill

## Overview

The specification-writing skill is the authoritative source for all product specification document generation in Claude Code. It consolidates ~1,720 lines of template and procedural knowledge previously scattered across three agents into a single, reusable skill domain.

## What This Skill Provides

### Document Types

1. **Epic PRDs** - High-level product requirements for major initiatives
   - Multi-file architecture (6 interconnected files)
   - Business value analysis
   - User personas and journeys
   - Comprehensive requirements and metrics

2. **Feature PRDs** - Detailed requirements for individual features
   - Scaled to feature complexity
   - User stories and acceptance criteria
   - Integration with parent epic
   - Scoped to prevent feature creep

3. **Implementation Tasks** - Agent-executable work items
   - High-level directives, not code tutorials
   - Reference design documents
   - Clear dependencies and validation gates
   - Agent-appropriate assignments
   - Centralized in /docs/tasks/ directory structure

### Key Benefits

- **Single source of truth** - Templates updated once, all agents benefit
- **Reusability** - Any agent can generate specifications
- **Consistency** - Standardized document structure across all projects
- **Maintainability** - Clear ownership of specification patterns

## Skill Structure

```
specification-writing/
├── SKILL.md                    # Router with workflow selection
├── README.md                   # This file
├── workflows/
│   ├── write-epic.md          # Epic PRD creation workflow
│   ├── write-feature-prd.md   # Feature PRD creation workflow
│   └── write-task.md          # Task generation workflow
└── context/
    ├── epic-template.md       # Epic structure and sections
    ├── prd-template.md        # Feature PRD structure
    ├── task-template.md       # Task structure and frontmatter
    └── naming-conventions.md  # File naming standards
```

## Usage

### For Agents

Reference workflows in your agent instructions:

```markdown
## Your Process
1. Analyze requirements
2. Invoke specification-writing skill for document generation
3. Follow workflow from: ~/.claude/skills/specification-writing/workflows/write-{type}.md
4. Use templates from: ~/.claude/skills/specification-writing/context/
```

### For Users

Agents will automatically invoke this skill when you request:
- "Create an epic PRD for..."
- "Write a feature PRD for..."
- "Generate tasks for this feature"

## Migration Notes

This skill was extracted from three agents:

- **epic-prd-writer** (880 lines) → `workflows/write-epic.md` + `context/epic-template.md`
- **prd-writer** (175 lines) → `workflows/write-feature-prd.md` + `context/prd-template.md`
- **task-generator** (665 lines) → `workflows/write-task.md` + `context/task-template.md`

The original agents now serve as lightweight coordinators that invoke this skill.

## Consumers

Agents that use this skill:
- epic-prd-writer (immediate consumer)
- prd-writer (immediate consumer)
- task-generator (immediate consumer)
- feature-architect (may invoke for documentation)
- Any future documentation agents

## Version History

- **v1.0.0** (2025-12-09) - Initial extraction from epic-prd-writer, prd-writer, and task-generator agents
- **v1.1.0** (2025-12-14) - Updated from PRP terminology to Task-based workflow with centralized task management
