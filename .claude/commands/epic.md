---
description: Create a comprehensive Epic-level Product Requirements Document
---

# Create Epic PRD

Generate a complete Epic PRD using a modular, multi-file architecture for improved navigability and maintainability.

## Usage

```bash
/epic
```

This command is interactive - it will guide you through the epic creation process by asking clarifying questions.

## Implementation

This command invokes the specification-writing skill's epic creation workflow.

**Skill**: `~/.claude/skills/specification-writing/workflows/write-epic.md`
**Context**: `~/.claude/skills/specification-writing/context/epic-template.md`

## What This Does

The skill will:
1. Gather information about your epic through clarifying questions
2. Create a `/docs/plan/{epic-key}/` directory
3. Generate 6 interconnected files:
   - `epic.md` - Main index/reference file
   - `personas.md` - User personas detail
   - `user-journeys.md` - User workflow narratives
   - `requirements.md` - Comprehensive requirements catalog
   - `success-metrics.md` - KPIs and measurement framework
   - `scope.md` - Boundaries and exclusions

## Example

```bash
/epic
```

Then respond to the skill's questions about your epic concept.
