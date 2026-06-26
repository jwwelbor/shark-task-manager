---
name: org-standards
description: Define and apply organization-level engineering standards for naming, file layout, documentation, code style, review expectations, and repository conventions. Use when creating, auditing, or harmonizing standards that should guide many features or teams.
when_to_use: when work needs reusable engineering conventions, naming rules, repository structure guidance, or standards drift review before implementation or review begins
version: 1.0.0
domain: engineering-standards
inputs:
  - existing_conventions: observed patterns from the repository or organization
  - target_scope: the team, repository, artifact type, or workflow the standard applies to
  - pain_points: inconsistencies or recurring mistakes the standard should reduce
  - constraints: platform, language, framework, compliance, or legacy constraints
outputs:
  - selected_workflow: one of {derive-standards, audit-standards-compliance}
  - standards_document: concise conventions with rationale and examples
  - compliance_report: findings that identify where artifacts diverge from the standard
  - open_questions: unresolved policy choices that need an owner decision
---

# Org Standards Skill

This skill turns observed practice and team preferences into standards that are clear enough to apply repeatedly. A good standard reduces decision noise without freezing legitimate local variation.

## Workflow Selection

### Derive Standards

Use `workflows/derive-standards.md` when creating or revising a standard. The workflow extracts common patterns, separates mandatory rules from preferences, and writes a concise standard with examples.

### Audit Standards Compliance

Use `workflows/audit-standards-compliance.md` when comparing artifacts against an existing standard. The workflow reports deviations, severity, and recommended remediation.

## Principles

1. **Prefer observed practice.** Start from what the codebase already does well.
2. **Make rules testable.** A standard should say what to check, not merely what to value.
3. **Separate rule from rationale.** Keep the requirement short; explain why immediately after.
4. **Avoid taste masquerading as policy.** If two options are equally maintainable, mark the choice as a preference or leave it open.
5. **Optimize for repeated use.** Standards should be easy for another agent to apply without needing the original author.

## Output Shape

For a standards document, include:

- Scope
- Mandatory rules
- Recommended defaults
- Naming and structure examples
- Exceptions and escalation criteria
- Open decisions

For a compliance report, include:

- Summary verdict
- Findings table
- Evidence
- Severity
- Remediation
