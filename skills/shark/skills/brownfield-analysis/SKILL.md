---
name: brownfield-analysis
description: >
  Methodology for deep analysis and documentation of existing (brownfield) codebases.
  Covers how to analyze architecture, code structure, business logic, technical debt,
  security, code quality, and migration readiness, and how to produce enterprise-grade
  documentation from what you find. Use whenever the work involves analyzing a codebase,
  reverse-engineering a project, documenting an existing system, assessing technical debt,
  understanding a legacy codebase, or judging whether a system is ready to migrate — even
  when the word "brownfield" is not used.
when_to_use: when analyzing, reverse-engineering, or documenting any existing codebase
version: 1.0.0
---

# Brownfield Codebase Analysis

## Overview

This skill is the methodology for analyzing an existing codebase and producing
documentation that should have existed all along: a complete picture of what the system
is, how it works, where its risks are, and how to move it forward.

It is technology-agnostic. A Python FastAPI microservice gets the same analytical rigor
as a Java EJB monolith — the methodology stays the same; the specifics adapt to whatever
you find.

## What this skill owns

This skill owns the **analytical methodology** — how to do each kind of analysis well and
what good output looks like. It does not own ordering, scheduling, resumability, or any
record of what has been done. Those concerns live outside the skill.

The methodology is organized into ten analysis areas. Each area has a self-contained
procedure under `workflows/`. The areas are independent units of methodology; this file
does not prescribe an order between them.

| Analysis area | Procedure | Typical output location |
|---|---|---|
| Discovery & inventory | `workflows/analyze-discovery.md` | `project-overview.md` |
| Architecture | `workflows/analyze-architecture.md` | `architecture/*.md` |
| Code reference | `workflows/analyze-code-reference.md` | `reference/*.md` |
| Behavior | `workflows/analyze-behavior.md` | `behavior/*.md` |
| Visual documentation | `workflows/produce-diagrams.md` | `diagrams/**/*.md` |
| Technical debt | `workflows/assess-technical-debt.md` | `technical-debt/*.md` |
| Code quality | `workflows/analyze-code-quality.md` | `analysis/*.md` |
| Migration readiness | `workflows/assess-migration-readiness.md` | `migration/*.md` |
| Specialized areas | `workflows/document-specialized-areas.md` | `specialized/**/*.md` |
| Finalization | `workflows/finalize-documentation.md` | `README.md`, `technical-debt-report.md` |

For where each document lives and what the output tree looks like, see
`context/output-structure.md`. For document formatting conventions (headers,
cross-references, Mermaid, evidence references, tables), see `context/output-conventions.md`.
For the depth and specificity to aim for, with a worked example, see `context/quality-bar.md`.

## Cross-cutting analytical principles

These principles apply to every analysis area.

### Scan, then document

For each area, first do a broad scan to understand what you are dealing with, then document
thoroughly. Do not document files you have not read. When you document something, go deep:
record its purpose, key classes/functions, dependencies, patterns used, and any concerns.

### Evidence-based claims

Every assertion references a specific file path. When you say "uses the Factory pattern,"
point to the file and the relevant code. When you list a dependency, include its version
from the actual build file. When you identify a vulnerability, cite the specific library
and version. Documentation without evidence is speculation.

### Diagram with Mermaid

Use Mermaid syntax for all diagrams — architecture overviews, sequence diagrams, class
hierarchies, data flows, deployment topologies. Mermaid renders natively in most
documentation tools, which makes the output immediately useful.

### Ask when uncertain

When you encounter ambiguous business logic, unclear architectural decisions, or code whose
purpose is not evident from context, ask rather than guess. It is better to pause and get it
right than to fill documentation with speculation.

### Adapt to what you find

The analysis areas are a menu, not a mandate. If a project has no database, there is no
database documentation to write. If it is a simple library with no deployment
infrastructure, there are no deployment or environment diagrams. Focus energy where
there is substance to document, and skip what does not apply.

## Methodology for large codebases

For codebases with hundreds or thousands of files:

- **Discovery is your map.** Invest in understanding the module/package structure early.
  A complete structural picture lets you work systematically instead of randomly, and it
  pays for itself many times over.
- **Work package by package.** Within any analysis area, do not try to cover the whole
  codebase at once. Go package by package. This keeps each unit of work bounded and
  reviewable.
- **Prioritize high-value areas.** If the goal is migration, weight technical debt and
  dependencies. If the goal is onboarding, weight architecture and business logic. When the
  priority is not obvious from the request, ask what matters most.

## Output quality standard

The documentation produced should be at the level of a senior architect's project handoff.
Someone reading it with no prior context should be able to:

1. Understand what the system does and why it exists.
2. Navigate the codebase confidently.
3. Identify the riskiest areas — technical debt, security, complexity.
4. Plan a modernization or migration effort.
5. Onboard new team members effectively.

See `context/quality-bar.md` for a concrete worked example of this standard and the depth
to aim for.

## Related

The bundle's `research/workflows/brownfield-analysis.md` is the **lightweight bootstrap** — it
produces the four `docs/architecture/` foundation documents as part of the `/shark project bootstrap`
flow. This sub-skill is the comprehensive standalone methodology for full enterprise analysis.
Both exist for different use cases; they are not duplicates.
