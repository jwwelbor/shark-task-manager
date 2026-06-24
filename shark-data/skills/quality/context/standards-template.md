# Coding Standards Template

## Context to Gather

Team/Product: {{team}}

Stack: {{languages_frameworks}} (e.g., TypeScript+React, Node/NestJS, Python/FastAPI, Java/Spring, Go)
- Pull from `{project root}/docs/architecture/system-design-*.md`
- Or examine codebase files

Codebase Inputs: {{repo_links_or_sample_files}}

Local Tools: {{linters_formatters_test_runners}}

Priorities (ranked): {{maintainability|correctness|security|performance|accessibility}}

Constraints: {{legacy_areas|deadline|compliance_code_only}}

## Scope Rules

- Only code concerns: style, naming, API/DTO boundaries, typing, errors, validation, performance idioms, secure-by-construction rules, accessibility (for UI), and code-level testing guidance.
- Be opinionated; avoid vague language.
- Each rule must include: Rule • Why • How to apply/enforce locally • Example (✅/❌, ≤50 lines).
- Assume gaps where needed; state assumptions explicitly.

## Required Output A Structure

**File**: `{project root}/docs/architecture/coding-standards.md`

### Required Headings (use exact format):

```markdown
# Executive Summary (Code-Only)

## Universal Coding Standards

## Language-Specific Standards

## Framework-Specific Standards

## Testing Standards (Code-Level)

## Secure Coding Standards

## Review Rubric (YAML)
```yaml
rules:
  - id: "TS-NAMING-001"
    title: "TypeScript filenames use kebab-case"
    appliesTo:
      - "ts"
      - "tsx"
    severity: "major"
    why: "Consistency and predictable imports"
    how_to_check: "Static filename pattern"
    example_good: "user-service.ts"
    example_bad: "UserService.ts"
```

## Reference Configs (Local Dev)
.editorconfig, formatter/linter configs (ESLint+Prettier, ruff+Black, golangci-lint, etc.), optional local pre-commit

## Adoption Guide & Checklists
PR checklist, new-module checklist, API endpoint checklist

## One-Page Quickstart
files to add, commands to run, adoption order: format → lint → refactor → tests
```

## Required Output B Structure

**File**: `{project root}/docs/plan/tech-debt/coding-standards-gaps.md`

### Required Headings (use exact format):

```markdown
## Current State Snapshot (Code Practices)

## Gap Analysis (Current → Recommended)
```

## Reference Stubs to Include

Adapt to stack:

### TypeScript
- Strict tsconfig.json (strict, noUncheckedIndexedAccess, exactOptionalPropertyTypes)
- ESLint + Prettier baseline with key rules (no any, explicit returns, import order, no console except warn/error)
- React conventions: component/file naming, hooks usage, state patterns, a11y queries

### Python
- pyproject.toml for Black+ruff with curated rule sets
- Error taxonomy template and logging level table

### Go
- golangci-lint.yaml enabling errcheck, staticcheck, etc.

### General
- Local pre-commit (optional) for format/lint only
- Error taxonomy template and logging level table

## Deliverable Style

- Tables for conventions/checklists
- Compact code blocks
- Unambiguous wording
- No infrastructure/CI/CD/cloud instructions

## Success Criteria

- Clear, enforceable rules with copy-paste configs (where appropriate)
- Rubric is directly usable by the review-agent as the source of truth
- All required sections present
- Examples are concrete and actionable
