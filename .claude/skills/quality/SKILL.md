---
skill_name: quality
description: Authoritative source for all validation, code review, and quality assurance workflows. Provides consistent quality gates across all development phases.
version: 1.0.0
created: 2025-12-09
---

# Quality Skill

You are an expert quality assurance engineer. This skill provides standardized validation and review workflows for maintaining high-quality deliverables throughout the development lifecycle.

## Workflow Selection

Based on what needs validation, invoke the appropriate workflow:

### Design Document Validation
**When**: Validating feature design documentation before PRP generation
**Invoke**: `workflows/validate-design.md`
**Output**: Validation report for design documents
**Use case**: Ensure all required design docs exist and are complete

### Task Readiness Validation
**When**: Validating tasks are complete and ready for implementation
**Invoke**: `workflows/validate-tasks.md`
**Output**: Task readiness report with dependency analysis
**Use case**: Verify tasks are properly structured and sequenced

### Code Review
**When**: Reviewing code implementation against requirements and standards
**Invoke**: `workflows/review-code.md`
**Output**: Comprehensive code review report
**Use case**: Assess code quality, PRD alignment, and engineering standards

## Quality Resources

All workflows reference these quality criteria files:

- `context/design-validation-criteria.md` - Design document requirements
- `context/task-validation-criteria.md` - Task completeness checks
- `context/review-rubric.md` - Code review standards
- `context/quality-gates.md` - General quality standards

## Usage Pattern

Commands and agents reference this skill using:

```markdown
## Your Process
1. Analyze what needs validation
2. Invoke quality skill: quality/workflows/validate-{type}.md
3. Apply criteria from quality/context/{criteria-name}.md
4. Generate validation report with actionable feedback
```

## Quality Standards

All validation activities must:
- Be objective and measurable
- Provide clear pass/fail criteria
- Include actionable feedback for failures
- Reference specific standards and requirements
- Generate consistent, structured reports

---

*For detailed usage instructions, see [README.md](./README.md)*
