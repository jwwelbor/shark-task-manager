---
description: Generate authoritative coding standards document for the codebase
---

# Generate Coding Standards

Produce a single, authoritative, code-only Coding Standards document for the provided stack.

## Usage

```bash
/coding-standards
```

## Implementation

This command invokes the quality skill's coding standards generation workflow.

**Skill**: `~/.claude/skills/quality/workflows/generate-standards.md`
**Context**: `~/.claude/skills/quality/context/standards-template.md`

The skill will analyze the codebase and generate standards documents at:
- `{project root}/docs/architecture/coding-standards.md`
- `{project root}/docs/plan/tech-debt/coding-standards-gaps.md`
