---
inputs:
  - codebase_path: absolute path to the project root (used to discover sample files, lint configs, test-runner configs)
  - sample_paths: list of representative source files to analyze for current style/conventions (optional — derived from codebase_path if not supplied)
  - languages: list of language identifiers (e.g. ["typescript", "python", "go"])
  - frameworks: list of framework identifiers (e.g. ["react", "fastapi", "gin"])
  - existing_standards_path: absolute path to a prior coding-standards doc (optional, used as a baseline to update rather than rewrite)
  - priorities: list of priority keywords from {maintainability, correctness, security, performance, accessibility}
  - constraints: list of {area, description} for legacy code areas, deadlines, compliance requirements
  - team_name: string — team or product name for document header
  - standards_doc_path: absolute path where the coding-standards markdown should be written
  - gaps_doc_path: absolute path where the gaps-analysis markdown should be written
outputs:
  - coding_standards_doc: structured markdown written to standards_doc_path
  - gaps_doc: structured markdown written to gaps_doc_path
  - rubric: YAML review rubric embedded in coding_standards_doc, also returned as a structured object
  - reference_configs: list of {tool, config_path, content} for linters/formatters/test-runners
---

# Workflow: Generate Coding Standards (craft)

## Purpose

Produce a single, authoritative, code-only Coding Standards document for the provided stack. The output is deterministic, example-driven, and directly enforceable by linters/formatters/static analysis.

This workflow generates two artifacts:

- **Coding Standards document** at `standards_doc_path` — the authoritative rules
- **Gaps Analysis document** at `gaps_doc_path` — current state vs. recommended

## What This Workflow Generates

### Output A: Coding Standards Document

Contains:
- Executive Summary (Code-Only)
- Universal Coding Standards
- Language-Specific Standards (one section per entry in `languages`)
- Framework-Specific Standards (one section per entry in `frameworks`)
- Testing Standards (Code-Level)
- Secure Coding Standards
- Review Rubric (YAML format)
- Reference Configs (Local Dev)
- Adoption Guide & Checklists
- One-Page Quickstart

### Output B: Gaps Analysis Document

Contains:
- Current State Snapshot (Code Practices) — derived from `sample_paths` analysis
- Gap Analysis (Current → Recommended)

## Execution Steps

### Step 1: Analyze Codebase Context

Gather information about:
- Stack — confirmed by `languages` and `frameworks` inputs; cross-check by inspecting `codebase_path` (file extensions, manifest files like `package.json`, `pyproject.toml`, `go.mod`)
- Codebase structure and sample files — read `sample_paths` (or sample from `codebase_path`)
- Local tools — discover linters/formatters/test runners by inspecting config files in `codebase_path` (e.g., `.eslintrc`, `pyproject.toml`, `.prettierrc`, `Makefile`)
- Priorities — from `priorities` input
- Constraints — from `constraints` input

If `existing_standards_path` is provided, read it as a baseline. The output should evolve the existing standards rather than replacing them.

### Step 2: Generate Standards Document

Write `standards_doc_path` covering **code-only concerns only**:
- Style, naming, API/DTO boundaries, typing, errors, validation
- Performance idioms
- Secure-by-construction rules
- Accessibility (for UI frameworks)
- Code-level testing guidance

**Each rule must include**:
- **Rule** — the imperative statement
- **Why** — the reasoning
- **How to apply/enforce locally** — the linter/formatter/static-analysis config that enforces it
- **Example** — ✅ correct / ❌ incorrect, ≤50 lines

**Opinionated, not vague.** Avoid "should generally" / "consider" / "where possible" — pick a position.

**Excluded concerns** (these belong elsewhere):
- Infrastructure / CI/CD pipeline configuration
- Cloud provider setup
- Deployment automation

See `../context/standards-template.md` for detailed structure.

### Step 3: Generate Gaps Analysis

Write `gaps_doc_path` analyzing:

**Current State Snapshot (Code Practices)** — derived from analyzing `sample_paths`:
- Observed style/naming conventions
- Observed error-handling patterns
- Observed testing patterns
- Observed security patterns

**Gap Analysis (Current → Recommended)** — for each rule in the standards doc:
- Current practice (citing file:line or "not present")
- Recommended practice (the rule)
- Migration effort estimate (low / medium / high)
- Risk if deferred

### Step 4: Validate Output

Ensure:
- Standards are clear and enforceable (every rule has a how-to-enforce entry)
- Rubric is usable as source of truth for code review (YAML structured, machine-readable)
- Copy-paste configs provided for every tool the codebase already uses
- No infrastructure/CI/CD/cloud instructions included
- All required sections present per template

## Success Criteria

- Clear, enforceable rules with copy-paste configs
- Rubric directly usable by review-agent
- Both output files written
- Standards appropriate for the languages and frameworks supplied
- Examples are concrete and actionable (no abstract advice)
- Gap analysis grounds every recommendation in observed current-state evidence

## Output Format

Documents should use:
- Tables for conventions/checklists
- Compact code blocks
- Unambiguous wording (no "should generally")
- YAML format for review rubric
- Markdown formatting

See `../context/standards-template.md` for complete template structure and reference configurations.
