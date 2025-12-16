# Workflow: Generate Coding Standards

## Purpose

Produce a single, authoritative, code-only Coding Standards document for the provided stack. This workflow generates comprehensive coding standards that are deterministic, example-driven, and directly enforceable by linters/formatters/static analysis.

## Usage

This workflow is invoked when generating coding standards:
- By `/coding-standards` command
- By architecture or quality agents
- When establishing project standards

## What This Workflow Generates

### Output A: Coding Standards Document
**Path**: `{project root}/docs/architecture/coding-standards.md`

Contains:
- Executive Summary (Code-Only)
- Universal Coding Standards
- Language-Specific Standards
- Framework-Specific Standards
- Testing Standards (Code-Level)
- Secure Coding Standards
- Review Rubric (YAML format)
- Reference Configs (Local Dev)
- Adoption Guide & Checklists
- One-Page Quickstart

### Output B: Gaps Analysis Document
**Path**: `{project root}/docs/plan/tech-debt/coding-standards-gaps.md`

Contains:
- Current State Snapshot (Code Practices)
- Gap Analysis (Current → Recommended)

## Execution Steps

### Step 1: Analyze Codebase Context

Gather information about:
- Team/Product name
- Stack (languages and frameworks) - check `docs/architecture/system-design-*.md` or examine files
- Codebase structure and sample files
- Local tools (linters, formatters, test runners)
- Priorities (maintainability, correctness, security, performance, accessibility)
- Constraints (legacy areas, deadlines, compliance requirements)

### Step 2: Generate Standards Document

Create `docs/architecture/coding-standards.md` with:
- Code-only concerns: style, naming, API/DTO boundaries, typing, errors, validation, performance idioms, secure-by-construction rules, accessibility (for UI), code-level testing guidance
- Opinionated rules (avoid vague language)
- Each rule includes: Rule • Why • How to apply/enforce locally • Example (✅/❌, ≤50 lines)
- See `../context/standards-template.md` for detailed structure

### Step 3: Generate Gaps Analysis

Create `docs/plan/tech-debt/coding-standards-gaps.md` analyzing:
- Current State Snapshot (Code Practices)
- Gap Analysis (Current → Recommended)

### Step 4: Validate Output

Ensure:
- Standards are clear and enforceable
- Rubric is usable as source of truth for code review
- Copy-paste configs provided where appropriate
- No infrastructure/CI/CD/cloud instructions included
- All required sections present

## Success Criteria

- Clear, enforceable rules with copy-paste configs
- Rubric directly usable by review-agent
- Both output files created at expected paths
- Standards appropriate for tech stack
- Examples are concrete and actionable

## Output Format

Documents should use:
- Tables for conventions/checklists
- Compact code blocks
- Unambiguous wording
- YAML format for review rubric
- Markdown formatting

See `../context/standards-template.md` for complete template structure and reference configurations.
