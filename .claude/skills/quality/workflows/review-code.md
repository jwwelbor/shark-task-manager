# Workflow: Code Review

## Purpose

Perform comprehensive code review against product requirements and engineering standards. Assess code changes for PRD alignment, SOLID principles, DRY violations, and idiomatic patterns.

## Usage

This workflow is invoked when reviewing code:
- By code-review-orchestrator agent
- After implementation completion
- Before pull request merge
- For post-implementation validation

## Review Process

### 1. Context Gathering

Extract from user input:
- PRD link or feature path
- Code diff/patch or changed files
- Repository root path
- Epic and feature names
- Links to architecture/standards docs

### 2. PRD Traceability

Map each code change to PRD requirements:
- Verify all PRD requirements are implemented
- Identify out-of-scope work (YAGNI violations)
- Flag missing implementations

### 3. Reuse Pass (DRY Enforcement)

Before flagging duplication:
- Search codebase for existing implementations
- Identify reusable utilities, patterns, modules
- Document reuse opportunities with file paths
- Propose specific refactors

### 4. Standards Validation

Cross-reference code against:
- `/docs/architecture/system-design-*.md`
- `/docs/architecture/coding-standards.md`
- SOLID principles
- Language-specific best practices

Note deviations with exact section citations.

### 5. Idiomatic Analysis

Review language-specific patterns:
- Pythonic patterns for Python
- TypeScript strictness
- Go idioms
- Replace non-idiomatic code with community standards

### 6. Quality Scoring

Apply 0-5 rubric to each modified file. See `../context/review-rubric.md` for scoring criteria:
- (a) Readability
- (b) Maintainability
- (c) Performance
- (d) Testability
- (e) Standards Compliance

Justify scores with evidence.

### 7. Risk Assessment

Identify hotspots and potential issues:
- Complexity concentrations
- Tight coupling
- Performance bottlenecks
- Security vulnerabilities
- Concurrency issues
- I/O boundary risks

### 8. Action Plan Creation

For every issue, provide:
- Minimal-change fixes
- Before/after code snippets
- Specific file paths and line numbers
- Avoid vague suggestions

## Required Output Structure

### A. Executive Summary (≤10 lines)
- What the PR accomplishes
- Fitness to PRD
- Overall risk level
- Ship/no-ship recommendation
- Summary TODO List

### B. Findings Table
For each issue:
- id (sequential)
- severity (blocker/major/minor)
- file:line
- rule (DRY/SRP/YAGNI/ARCH/CODING-STANDARD/IDIOM)
- diagnosis (what's wrong)
- evidence (code quote or line references)
- correction (exact steps to fix)
- snippet (before/after code when helpful)

### C. Reuse Opportunities
- Existing symbols/modules that can replace new code
- Include file paths and diff-ready refactor steps

### D. Standards Crosswalk
- Cite exact sections from architecture and coding standards docs
- Map each issue to violated standard

### E. Size/Style Check
- Note file/function size outliers
- Justify if acceptable or recommend splitting

### F. Idiomatic Improvements
- Language-specific pattern suggestions
- Replace non-idiomatic code with standards

### G. Quality Scores
- Score each file 0-5 on rubric dimensions
- Justify scores with evidence

### H. Risk Hotspots
- Identify complex areas requiring extra attention
- Note potential issues

### I. Action Plan
- Ordered list of fixes (blockers first)
- Specific, actionable steps
- Before/after examples

## Review Principles

**Non-Negotiable**:
- **DRY**: No duplicated logic - search and reuse existing code
- **SOLID & SRP**: Each unit has one clear responsibility
- **YAGNI**: Only implement what PRD requires
- **Principle of Least Surprise**: Code should be obvious and consistent
- **Language Best Practices**: Follow idiomatic patterns
- **Standards Compliance**: Adhere to project architecture and coding standards

**Size & Style Guidelines** (guides, not hard rules):
- Files ideally < 500 lines
- Functions ideally < 50 lines
- Comments only where non-obvious
- Avoid docstrings longer than code
- Avoid trivial 1-2 line functions unless justified
- Never re-implement existing functionality

## Success Criteria

Review is complete when:
- All sections of output structure provided
- Every issue has actionable correction
- PRD alignment verified
- DRY violations identified with reuse paths
- Standards deviations cited with section references
- Quality scores justified with evidence
- Risk assessment provided
- Ship/no-ship recommendation given

## Common Issues

- **Duplicated Logic**: Search codebase first, propose reuse
- **YAGNI Violation**: Flag speculative code not in PRD
- **SRP Violation**: Recommend splitting into focused units
- **Non-Idiomatic**: Suggest language-specific patterns
- **Missing Tests**: Flag testability issues
- **Performance**: Note bottlenecks and optimization opportunities

See `../context/review-rubric.md` for detailed review criteria and scoring rubric.
