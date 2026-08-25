---
name: quality
description: Authoritative source for all validation, code review, and quality assurance workflows. Provides consistent quality gates across all development phases.
version: 1.0.0
created: 2025-12-09
inputs:
  - entity_type: the type of entity being validated (task|feature|epic|design|code)
  - entity_key: the entity key being validated
  - spec_path: absolute path to the specification document
  - implementation_paths: list of paths to implemented code or documentation
  - test_paths: list of paths to test files (optional)
  - design_refs: list of paths to design documents or wireframes (optional)
  - acceptance_criteria: list of acceptance criteria text to validate against
outputs:
  - selected_workflow: one of {validate-design, validate-tasks, review-code, test-planning, qa-testing, generate-standards}
  - outcome: pass | fail | blocked
  - validation_report: structured validation results (PASS|FAIL)
  - issues_found: list of {severity, description, location, remediation}
  - coverage_analysis: validation coverage and gaps
  - recommendations: actionable recommendations for improvement
---

# Quality Skill

This skill provides quality assurance capabilities with standardized validation and review workflows for maintaining high-quality deliverables throughout the development lifecycle.

## Outcome contract

Every quality response includes one semantic `outcome`: `pass`, `fail`, or
`blocked`. `pass` and `fail` describe a completed validation decision; `blocked`
means required evidence, environment access, or authority is unavailable. The
host, not this craft skill, maps the outcome to workflow state.

## Workflow Selection

Based on what needs validation, invoke the appropriate workflow:

### Design Document Validation
**When**: Validating feature design documentation before task generation
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

### Test Planning (Shift-Left QA)
**When**: Reviewing task specifications BEFORE development begins
**Invoke**: `workflows/test-planning.md`
**Output**: Spec drift analysis + concrete acceptance test cases
**Use case**: Compare task spec against feature PRD for drift, write test plan that becomes the developer's source of truth. This is the quality gate between planning and development.

### QA Testing (Post-Development)
**When**: Validating implementation after code review
**Invoke**: `workflows/qa-testing.md`
**Output**: QA test results and exploratory findings
**Use case**: Validate implementation against pre-written test plan and acceptance criteria

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
