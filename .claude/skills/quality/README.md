# Quality Skill

## Overview

The quality skill is the authoritative source for all validation, code review, and quality assurance workflows in Claude Code. It consolidates ~885+ lines of validation logic from commands and code review processes into structured workflows that any agent or command can invoke.

## What This Skill Provides

### Validation Types

1. **Design Document Validation** - Verify all required design documents exist and are complete
   - File existence checks
   - Section completeness verification
   - Cross-reference validation
   - Anti-pattern detection

2. **Task Readiness Validation** - Ensure tasks are properly structured and ready for implementation
   - Task structure validation
   - Dependency graph analysis
   - Agent assignment verification
   - Success criteria completeness

3. **Code Review** - Comprehensive code quality assessment
   - PRD alignment verification
   - SOLID and DRY enforcement
   - Standards compliance checking
   - Quality rubric scoring

### Key Benefits

- **Consistent Quality Gates** - Standardized validation across all phases
- **Objective Criteria** - Clear, measurable pass/fail thresholds
- **Actionable Feedback** - Specific fixes for validation failures
- **Self-Validation** - Agents can validate their own output
- **User-Friendly** - Commands provide easy validation shortcuts

## Skill Structure

```
quality/
├── SKILL.md                           # Router for workflow selection
├── README.md                          # This file
├── workflows/
│   ├── validate-design.md            # Design document validation
│   ├── validate-tasks.md             # Task readiness validation
│   └── review-code.md                # Code review orchestration
└── context/
    ├── design-validation-criteria.md # Design doc requirements
    ├── task-validation-criteria.md   # Task completeness checks
    ├── review-rubric.md              # Code review standards
    └── quality-gates.md              # General quality standards
```

## Usage

### For Commands

Commands invoke quality workflows:

```markdown
---
description: Validate feature design documentation
---
Invoke quality skill validation workflow:

Follow process in: ~/.claude/skills/quality/workflows/validate-design.md
Apply criteria from: ~/.claude/skills/quality/context/design-validation-criteria.md
```

### For Agents

Agents can validate their own output:

```markdown
## Your Process
1. Complete your work (design docs, PRPs, code, etc.)
2. Validate using quality skill workflows
3. Fix any issues found
4. Proceed only after validation passes
```

### For Users

Users invoke via commands:
- `/validate-feature-design {epic} {feature}` - Validates design documents
- `/validate-task-readiness` - Validates tasks
- Code review via code-review-orchestrator agent

## Migration Notes

This skill was extracted from:

- **validate-feature-design.md** (370 lines) → `workflows/validate-design.md` + criteria
- **validate-task-readiness.md** (515 lines) → `workflows/validate-tasks.md` + criteria
- **code-review-orchestrator** (process-heavy) → `workflows/review-code.md` + rubric

The original commands and agents now serve as lightweight wrappers that invoke this skill.

## Consumers

Commands/agents that use this skill:
- `/validate-feature-design` command (immediate consumer)
- `/validate-task-readiness` command (immediate consumer)
- code-review-orchestrator agent (immediate consumer)
- feature-architect agent (validates own output)
- task-generator agent (validates own output)
- Any agent needing quality validation

## Version History

- **v1.0.0** (2025-12-09) - Initial extraction from validation commands and code-review-orchestrator
- **v1.1.0** (2025-12-14) - Updated from PRP terminology to Task-based workflow with centralized task management
