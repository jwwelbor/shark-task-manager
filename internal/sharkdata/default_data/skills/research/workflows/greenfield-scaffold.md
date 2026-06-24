---
inputs:
  - project_root: absolute path to the (mostly empty) project directory
  - output_dir: absolute path where the five greenfield docs should be written
  - tech_stack_path: absolute path for tech-stack.md output
  - architecture_overview_path: absolute path for architecture-overview.md output
  - file_system_path: absolute path for file-system.md output (prescriptive)
  - patterns_catalog_path: absolute path for patterns-catalog.md output
  - integration_map_path: absolute path for integration-map.md output
  - generation_date: ISO date for the greenfield header on each doc
outputs:
  - idea_readiness: 1 | 2 | 3 (just-stack / needs-refinement / solid-idea — captured for the caller's "next step" suggestion)
  - chosen_stack: structured summary of the language, framework(s), and primary dependencies selected
  - tech_stack: prescriptive markdown written to tech_stack_path
  - architecture_overview: prescriptive markdown written to architecture_overview_path
  - file_system: prescriptive markdown written to file_system_path
  - patterns_catalog: prescriptive markdown written to patterns_catalog_path
  - integration_map: prescriptive markdown written to integration_map_path
---

# Workflow: Greenfield Scaffold

**Purpose**: Interactive stack selection and prescriptive foundation document generation for new projects
**Use for**: Greenfield projects when no existing codebase is detected
**Output**: `tech-stack.md`, `architecture-overview.md`, `file-system.md`, `patterns-catalog.md`, `integration-map.md` (5 docs — coding-standards handled by orchestrator)
**Output location**: caller-supplied (typically `docs/architecture/` in project root)

## Overview

This workflow is a **routing + foundation generation** step, NOT an idea exploration tool. It produces prescriptive architecture documents based on the user's chosen stack and official best practices.

**Does NOT do**: idea exploration, Socratic questioning, design presentation — those are `/brainstorming`'s and `/vision`'s jobs.

## Required Tools

- **AskUserQuestion** — Stack determination and idea readiness
- **WebSearch** — Official docs, reference architectures, community patterns
- **Write** — Creating output documents

---

## Phase 1: Idea Readiness Assessment

Ask the user:

> **How well defined is your idea?**
>
> 1. **No idea yet — just setting up a tech stack** (I know the stack, just generate the foundation)
> 2. **I need help thinking it through** (rough idea, needs refinement)
> 3. **I have a solid idea — ready to implement** (clear vision, just need the project set up)

Record the answer — it determines the "next steps" suggestion at the end.

| Answer | After Init Suggestion |
|--------|-----------------------|
| 1 — Just tech stack | "When you're ready to develop an idea, try `/brainstorming`" |
| 2 — Needs refinement | "Run `/brainstorming` to refine your idea into a design" |
| 3 — Solid idea | "Run `/vision` to capture your vision and kick off the PDLC" |

---

## Phase 1b: Stack Determination (1-2 turns max)

For all three answers, determine the stack. Ask:

> **What tech stack?** (e.g., "Next.js + PostgreSQL", "Python FastAPI", "Go with SQLite")
>
> Or describe what you're building and I'll recommend one.

### Routing by response:

**User gives explicit stack** (e.g., "Next.js + Prisma + PostgreSQL"):
- Accept as-is → proceed to Phase 2

**User describes what they're building** (e.g., "a SaaS for scheduling appointments"):
- Extract domain signals (web app, API, data pipeline, etc.)
- Extract scale signals (MVP, production, enterprise)
- Consult `../context/stack-research-guide.md` for domain → candidate stacks
- Recommend a stack with brief rationale
- Confirm with user before proceeding

**User says "recommend for me" with no context**:
- Ask two follow-up questions (one turn):
  1. What domain? (web app, API, CLI, data pipeline, mobile)
  2. Team experience? (what languages/frameworks are you comfortable with?)
- Use answers to select from `../context/stack-research-guide.md`
- Recommend with rationale, confirm

### Stack determination constraints:
- Maximum 2 turns of interaction
- If user is indecisive after 2 turns, default to Next.js + PostgreSQL (safest general-purpose choice) and note this in the docs

---

## Phase 2: Web Research for Stack

Using `../context/stack-research-guide.md` research queries for the confirmed stack:

### Research targets:
1. **Official style guide** for the primary language
2. **Recommended project structure** from framework docs
3. **Community-established patterns** (architecture, testing, error handling)
4. **Current stable versions** of all stack components
5. **Reference architecture** for the stack at the stated scale

### Research approach:
- Use WebSearch with year-specific queries from the stack research guide
- Focus on authoritative sources (official docs, framework creators, major contributors)
- Capture version numbers, URLs, and key recommendations

---

## Phase 3: Generate 5 Prescriptive Foundation Documents

All greenfield documents include this header:

```markdown
> **Greenfield**: Recommended patterns, not discovered. Update as project grows.
> Generated: YYYY-MM-DD | Stack: {stack summary}
```

### Document 1: `tech-stack.md`

Based on confirmed stack + web research:

```markdown
> **Greenfield**: Recommended patterns, not discovered. Update as project grows.
> Generated: YYYY-MM-DD | Stack: {summary}

# Tech Stack

## Languages

| Language | Version | Role | Official Docs |
|----------|---------|------|---------------|
| {lang} | {version} | Primary | {url} |

## Frameworks

| Framework | Version | Purpose | Official Docs |
|-----------|---------|---------|---------------|
| {framework} | {version} | {role} | {url} |

## Recommended Dependencies

| Package | Version | Purpose | Why |
|---------|---------|---------|-----|
| {pkg} | {ver} | {purpose} | {rationale from official docs} |

## Dev Tooling (Recommended)

| Tool | Purpose | Config |
|------|---------|--------|
| {tool} | {purpose} | {config file} |

## Rationale

{Why this stack was chosen — domain fit, team experience, ecosystem maturity}
```

### Document 2: `architecture-overview.md`

Reference architecture for the chosen stack at stated scale:

```markdown
> **Greenfield**: Recommended patterns, not discovered. Update as project grows.
> Generated: YYYY-MM-DD | Stack: {summary}

# Architecture Overview

## System Summary

{What this system will be — based on stack choice and any domain hints}

## Architecture Style

**Recommended**: {e.g., Modular Monolith for MVP, Microservices for scale}
**Rationale**: {from official docs / community consensus}

## Component Diagram

{Standard reference architecture for this stack}

## Recommended Module Structure

| Module | Responsibility | Pattern |
|--------|---------------|---------|
| {module} | {purpose} | {e.g., Service + Repository} |

## Cross-Cutting Concerns

| Concern | Recommended Approach | Library |
|---------|---------------------|---------|
| Logging | {approach} | {lib} |
| Error Handling | {approach} | {lib} |
| Configuration | {approach} | {lib} |
| Authentication | {approach} | {lib} |

## Scaling Considerations

{Based on scale modifier — MVP: keep simple, Growth: plan for X, Enterprise: ensure Y}
```

### Document 3: `file-system.md`

Prescriptive recommended structure (NOT scanned — this is what to build):

```markdown
> **Greenfield**: Recommended patterns, not discovered. Update as project grows.
> Generated: YYYY-MM-DD | Stack: {summary}

# File System (Recommended Structure)

## Directory Tree

{Official/recommended project structure for this stack}

## Directory Responsibilities

{What each directory should contain}

## File Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| {type} | {convention} | {example} |

## How to Extend

### Add a New Feature
{Step-by-step based on the recommended structure}

### Add a New API Endpoint
{Step-by-step}
```

### Document 4: `patterns-catalog.md`

Standard patterns for this stack from official docs:

```markdown
> **Greenfield**: Recommended patterns, not discovered. Update as project grows.
> Generated: YYYY-MM-DD | Stack: {summary}

# Patterns Catalog (Recommended)

## Architectural Patterns

### {Pattern 1: e.g., Repository Pattern}
- **When to use**: {guidance}
- **Implementation**: {brief code sketch or structure}
- **Source**: {official docs reference}

## Naming Conventions

{From official style guide for this language/framework}

## Testing Patterns

| Type | Framework | Location | Naming |
|------|-----------|----------|--------|
| Unit | {framework} | {dir} | {pattern} |
| Integration | {framework} | {dir} | {pattern} |
| E2E | {framework} | {dir} | {pattern} |

## Error Handling Pattern

{Official recommended approach for this stack}

## State Management (if frontend)

{Recommended approach from framework docs}
```

### Document 5: `integration-map.md`

Anticipated integrations — initially sparse:

```markdown
> **Greenfield**: Recommended patterns, not discovered. Update as project grows.
> Generated: YYYY-MM-DD | Stack: {summary}

# Integration Map

> This document is intentionally sparse for a new project. Update as integrations are added.

## Inbound API Surface

| Protocol | Framework | Status |
|----------|-----------|--------|
| {e.g., REST} | {e.g., Next.js API Routes} | Planned |

## Data Storage

| Store | Technology | Purpose | Status |
|-------|-----------|---------|--------|
| {e.g., Primary DB} | {e.g., PostgreSQL} | Application data | Planned |

## External Services

{To be added as integrations are built}

## Environment Variables (Anticipated)

| Variable | Service | Required |
|----------|---------|----------|
| DATABASE_URL | {DB} | Yes |
```

---

## Phase 4: Context-Dependent Next Step

Based on Phase 1 answer, display the appropriate suggestion:

**Answer 1 (just tech stack)**:
> Foundation documents generated. When you're ready to develop an idea, try `/brainstorming` to refine it into a design.

**Answer 2 (needs refinement)**:
> Foundation documents generated. Run `/brainstorming` to refine your idea into a fully-formed design, then `/vision` to kick off the PDLC.

**Answer 3 (solid idea)**:
> Foundation documents generated. Run `/vision` to capture your vision and create an epic — this will kick off the full development workflow.

---

## Success Criteria

- [ ] All 5 documents written to `docs/architecture/`
- [ ] All documents include greenfield header with date and stack summary
- [ ] Tech stack includes actual current version numbers (from web research)
- [ ] Architecture references official best practices (not generic patterns)
- [ ] File system structure matches official framework recommendations
- [ ] User received context-appropriate next step suggestion
- [ ] Stack determination completed in ≤2 turns

## Related Files

- `../context/stack-research-guide.md` — Authoritative sources per stack
- `project-init.md` — Orchestrator that invokes this workflow
- `brownfield-analysis.md` — Alternative track for existing codebases
