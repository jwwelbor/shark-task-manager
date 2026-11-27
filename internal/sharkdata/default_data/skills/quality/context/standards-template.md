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

### Quality Limits and Testing Conventions

Development and review read `## Quality Limits` and `## Testing Standards (Code-Level)` by name, so keep both headings. The standards file may hold either section itself or in a linked standards sub-file.

| Quality Limits column | Content |
|---|---|
| Value | The project's own value, marked as derived from a named config file or as proposed. Limit resolution and generic defaults live only in `craft-rubric.md` |
| Applies to | production, test or both |
| Enforcement | `tool-enforced` only when the project's tooling fails on a violation; otherwise `review-enforced` |
| Measuring command | The command that measures the limit, or none for review |

Keep one row per limit. The coverage row is optional; omit it when the project sets no target. The stacks shown side by side illustrate the form: keep only what the project uses.

The two Testing Standards entries are required. The project decides their content.

### Required Headings (use exact format):

```markdown
# Executive Summary (Code-Only)

## Universal Coding Standards

## Language-Specific Standards

## Framework-Specific Standards

## Quality Limits

| Limit | Value | Applies to | Enforcement | Measuring command |
|---|---|---|---|---|
| Cyclomatic complexity | {value} | {production, test or both} | {tool-enforced or review-enforced} | TypeScript: `npx eslint .` / Python: `ruff check .` / Go: `golangci-lint run` |
| Function length (lines) | {value} | {production, test or both} | {tool-enforced or review-enforced} | TypeScript: `npx eslint .` / Python: `pylint src` / Go: `golangci-lint run` |
| File length (lines) | {value} | {production, test or both} | {tool-enforced or review-enforced} | TypeScript: `npx eslint .` / Python: `pylint src` / Go: none (review) |
| Nesting depth | {value} | {production, test or both} | {tool-enforced or review-enforced} | TypeScript: `npx eslint .` / Python: `pylint src` / Go: none (review) |
| Coverage (optional) | {value} | {production, test or both} | {tool-enforced or review-enforced} | TypeScript: `npx vitest run --coverage` / Python: `pytest --cov` / Go: `go test -cover ./...` |

## Testing Standards (Code-Level)

### Test Isolation
{how this project keeps tests independent: shared state, fixtures, test data, execution order}

### Assertion Style
{the assertion library or style this project uses, and when a failed assertion stops the test}

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
