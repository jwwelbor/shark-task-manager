---
description: Create a detailed Feature-level Product Requirements Document
---

# Create Feature PRD

Generate a comprehensive Feature PRD that serves as the single source of truth for engineering teams.

## Usage

```bash
/prd {epic-key} {feature-description}
```

**Arguments**:
- `epic-key`: The parent epic directory name (e.g., E01-claude-reorg)
- `feature-description`: Brief description of the feature to create

**Example:**
```bash
/prd E01-claude-reorg "skill extraction and reorganization"
```

You can also use it interactively without arguments:
```bash
/prd
```

## Implementation

This command invokes the specification-writing skill's feature PRD creation workflow.

**Skill**: `~/.claude/skills/specification-writing/workflows/write-feature-prd.md`
**Context**: `~/.claude/skills/specification-writing/context/prd-template.md`

## What This Does

The skill will:
1. Gather information about the feature through clarifying questions
2. Assess feature complexity (snow-shoveling, ikea furniture, heart-surgery)
3. Create `/docs/plan/{epic-key}/{feature-key}/prd.md`
4. Generate comprehensive sections:
   - Feature Name & Goal
   - User Personas & User Stories
   - Functional & Non-Functional Requirements
   - Acceptance Criteria
   - Out of Scope boundaries
