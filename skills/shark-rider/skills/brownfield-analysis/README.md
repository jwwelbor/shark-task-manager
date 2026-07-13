# Brownfield Analysis Skill

A methodology skill for deep analysis and documentation of existing codebases.

## Philosophy

Good brownfield analysis produces documentation that should have existed all along. Rather
than describing an idealized version of the system, it documents what is actually there —
with specific evidence, real version numbers, actual file paths, and honest assessments of
risk. The output should read like a senior architect's project handoff: complete enough
that someone with no prior context can navigate, understand, and make decisions about the
system.

Two principles underpin everything:

**Evidence over assertion.** Every claim about the codebase references a specific file,
class, or configuration. "Uses the Repository pattern" is not a claim — "Uses the
Repository pattern (`src/main/java/com/example/repo/UserRepository.java`)" is.

**Adapt to what you find.** The analysis areas cover everything a complex system might
have. Apply what is relevant and skip what is not. A simple library does not need
deployment diagrams or stored-procedure documentation.

## Getting started

- `SKILL.md` — the full methodology: analytical principles, the ten analysis areas, and the
  quality standard
- `context/output-structure.md` — the canonical `docs/brownfield-docs/` directory layout
- `context/output-conventions.md` — document formatting: headers, Mermaid, evidence,
  cross-references, tables
- `context/quality-bar.md` — what "done well" looks like, with a concrete worked example

## Analysis area procedures

Each `workflows/` file is a self-contained domain procedure for one analysis area:

| File | Analysis area |
|---|---|
| `workflows/analyze-discovery.md` | Discovery & inventory |
| `workflows/analyze-architecture.md` | Architecture |
| `workflows/analyze-code-reference.md` | Code reference |
| `workflows/analyze-behavior.md` | Behavior & business logic |
| `workflows/produce-diagrams.md` | Visual documentation |
| `workflows/assess-technical-debt.md` | Technical debt |
| `workflows/analyze-code-quality.md` | Code quality |
| `workflows/assess-migration-readiness.md` | Migration readiness |
| `workflows/document-specialized-areas.md` | Specialized areas |
| `workflows/finalize-documentation.md` | Finalization |
