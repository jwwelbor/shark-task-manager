---
description: Validate that all required design documents exist and are complete before generating tasks
---

# Validate Feature Design Documentation

Verify that all required design documents for a feature are present and complete before proceeding to task generation.

## Usage

```bash
/validate-feature-design {epic-name} {feature-name}
```

**Example:**
```bash
/validate-feature-design E01-claude-reorg E01-F01-skill-extraction
```

## Implementation

This command invokes the quality skill's design validation workflow.

**Skill**: `~/.claude/skills/quality/workflows/validate-design.md`
**Context**: `~/.claude/skills/quality/context/design-validation-criteria.md`

**Arguments**:
- Epic name: {epic-name}
- Feature name: {feature-name}
