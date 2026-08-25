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
  - estate: brownfield | greenfield (optional; from the bootstrap marker — greenfield-scaffold only runs on the greenfield estate, but the field keeps the contract explicit)
  - mode: provisional | reconcile (optional, default provisional — provisional = first pass before product context exists; reconcile = re-run after D04 to revise the stack against the vision + feasibility)
  - vision_path: absolute path to D01-vision-statement.md, if it exists (optional — enables the evidence-aware path in Phase 0/1b)
  - feasibility_path: absolute path to D04-feasibility-report.md, if it exists (optional — required for reconcile mode)
  - user_needs_path: absolute path to D07-user-needs.md, if it exists (optional)
outputs:
  - idea_readiness: 1 | 2 | 3 (just-stack / needs-refinement / solid-idea — captured for the caller's "next step" suggestion)
  - chosen_stack: structured summary of the language, framework(s), and primary dependencies selected
  - tech_stack: prescriptive markdown written to tech_stack_path
  - architecture_overview: prescriptive markdown written to architecture_overview_path
  - file_system: prescriptive markdown written to file_system_path
  - patterns_catalog: prescriptive markdown written to patterns_catalog_path
  - integration_map: prescriptive markdown written to integration_map_path
  - stack_revisions: (reconcile mode only) list of {doc, was, now, driver} entries recording each divergence applied and why (null in provisional mode)
---

# Workflow: Greenfield Scaffold

**Purpose**: Interactive stack selection and prescriptive foundation document generation for new projects
**Use for**: Greenfield projects when no existing codebase is detected
**Output**: `tech-stack.md`, `architecture-overview.md`, `file-system.md`, `patterns-catalog.md`, `integration-map.md` (5 docs — coding-standards handled by orchestrator)
**Output location**: caller-supplied (typically `docs/architecture/` in project root)

## Overview

This workflow is a **routing + foundation generation** step, NOT an idea exploration tool. It produces prescriptive architecture documents based on the user's chosen stack and official best practices.

**Does NOT do**: idea exploration, Socratic questioning, design presentation — those are `/brainstorming`'s and `/vision`'s jobs.

**Two modes**, set by the caller (`mode`):

- **Provisional** (default, first pass) — pick and scaffold a stack. When product-design has already produced a vision/feasibility, read it first (Phase 0) and *propose* rather than interrogate; otherwise ask cold.
- **Reconcile** (re-run after D04) — the provisional stack already exists on disk. Diff it against the vision + feasibility that product-design produced, revise the affected docs, and record what changed (Phase 3.5). This closes the loop the proposed bootstrap-to-design flow needs: a greenfield stack chosen before the problem is understood gets reconciled once D04 has tested it.

## Required Tools

- **AskUserQuestion** — Stack determination and idea readiness
- **WebSearch** — Official docs, reference architectures, community patterns
- **Write** — Creating output documents
- **Read** — Product-design artifacts (vision, feasibility, user needs) when the caller supplies their paths

---

## Phase 0: Read Product Context (if available)

Before asking anything, check whether product-design has already produced evidence. The caller passes `vision_path`, `feasibility_path`, and `user_needs_path` when those artifacts exist.

For each path that is set and present on disk, read it and extract the stack-relevant signals:

| Artifact | Read for |
|----------|----------|
| D01 vision (`vision_path`) | problem domain, target user, scope, and any stated technology / regulatory / team **constraints** |
| D04 feasibility (`feasibility_path`) | technical risks, integration constraints, scalability needs, and the verdict (feasible / with changes / not feasible) |
| D07 user needs (`user_needs_path`) | non-functional needs that bear on the stack (offline, real-time, scale, accessibility) |

Summarize what you found in 3–5 lines: domain, scale, hard constraints, and any stack signal the evidence implies. This summary drives the evidence-aware path in Phase 1b — and, in reconcile mode, the diff in Phase 3.5.

If none of these paths is set or present, skip this phase and run the cold path in Phase 1/1b exactly as before.

> **Reconcile mode:** when `mode` is `reconcile`, `feasibility_path` (D04) must be present — that is the whole point of the re-run. Read the product context here, then jump to **Phase 3.5** (the provisional docs already exist; you are revising, not generating from scratch).

---

## Phase 1: Idea Readiness Assessment

Ask the user:

> **How well defined is your idea?**
>
> 1. **No idea yet — just setting up a tech stack** (I know the stack, just generate the foundation)
> 2. **I need help thinking it through** (rough idea, needs refinement)
> 3. **I have a solid idea — ready to implement** (clear vision, just need the project set up)

Record the answer — it determines the "next steps" suggestion at the end.

| Answer | After Init Suggestion (full text in Phase 4) |
|--------|-----------------------|
| 1 — Just tech stack | Stack is the deliverable — use the product-design workflow when an idea is ready |
| 2 — Needs refinement | **Lightweight path** — capture stack constraint, write `tech-stack.md` placeholder only, then use the product-design workflow. Re-run project bootstrap in reconcile mode after D04 to expand the placeholder. |
| 3 — Solid idea | Provisional stack — use the product-design workflow through D04, then re-run project bootstrap to reconcile |

> **Readiness 2** routes to **Phase 1c** (Lightweight Placeholder Output) immediately after Phase 1b.
> Skip Phases 2 and 3. No web research, no architecture docs — just a `tech-stack.md` placeholder.

---

## Phase 1b: Stack Determination (1-2 turns max)

For all three answers, determine the stack. Ask:

> **What tech stack?** (e.g., "Next.js + PostgreSQL", "Python FastAPI", "Go with SQLite")
>
> Or describe what you're building and I'll recommend one.

### Evidence-aware path (when Phase 0 found product context)

If Phase 0 produced a vision / feasibility / needs summary, **do not ask cold**. Derive a candidate stack from the domain, scale, and constraints you extracted, then confirm in a single turn:

> Based on {one-line problem + scale + the binding constraints}, I'd recommend **{stack}**. Use this, or adjust?

Treat a stated hard constraint from D01 (e.g. "must run on-prem", "team only knows Python") or a D04 technical risk as **binding** — it overrides the generic default. Note which evidence element drove each choice so a later reconcile (Phase 3.5) can trace it. One confirmation turn is enough; fall through to the cold routing below only if the user wants a different direction.

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

## Phase 1c: Lightweight Placeholder Output (Readiness 2 only)

> **Readiness 2 only.** Skip Phases 2 and 3 entirely. Write a single placeholder doc and proceed to Phase 4.

The idea is still rough. Running web research and generating five prescriptive architecture docs now —
before any product vision exists — creates documents that product-design will likely revise wholesale.
Instead: capture what's known, write a minimal placeholder, and route immediately to product-design.

### Step 1c.1: Stack constraint intake

If Phase 1b already captured a stack answer, use it — do not ask again.

If Phase 1b did not produce an answer (the user skipped or deferred), ask one brief question:

> **Any known tech-stack constraints?** (e.g. "must use Python", "team knows React", "not sure yet")

Accept any answer including "not sure" or "TBD". One turn only.

### Step 1c.2: Write `tech-stack.md` as a provisional placeholder

Write to `tech_stack_path` (typically `docs/architecture/tech-stack.md`):

```markdown
> **Greenfield — Provisional placeholder** · generated {generation_date}
> Stack captured at intake, before product-design. Expand this after D04 feasibility by
> re-running project bootstrap in reconcile mode.

# Tech Stack (Provisional)

**Status:** Placeholder — not yet validated against product requirements.

## Early Stack Signal

{user's stated constraint or preference}
{or "None stated — TBD after product-design." if they said "not sure" or equivalent}

## Quality Gate

To be defined after tech-stack is finalized and D04 has confirmed the approach.

---

*This document is intentionally minimal. Re-run project bootstrap in reconcile mode
after completing product-design (D01–D04) to expand it into a complete tech-stack document.*
```

### Step 1c.3: Skip remaining phases

- Do **not** run Phase 2 (web research).
- Do **not** run Phase 3 (architecture docs, patterns, integration map).
- Coding standards are skipped — there is nothing to base them on yet. They generate on the reconcile pass.
- Set `chosen_stack` to the stated constraint, or `null` if TBD.
- Set `generated_files` to `[{path: tech_stack_path, status: created}]`.
- Set `idea_readiness = 2`.

Proceed to Phase 4 for the next-step suggestion.

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

**Provisional labeling (idea readiness 2 or 3).** When an idea exists but vision/feasibility have not run yet, the stack was chosen before the problem was fully understood. Add a second header line to every doc so the reader knows it is pending reconciliation:

```markdown
> **Provisional**: chosen before feasibility (D04). Re-run project bootstrap in reconcile mode after product-design to confirm or revise.
```

When idea readiness is **1** (the stack *is* the deliverable), the standard greenfield header is sufficient — there is nothing to reconcile against.

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

## Quality Gate

The commands an agent must run before advancing work. Populate with the recommended commands for the chosen stack.

| Step | Command | When |
|------|---------|------|
| Format | {e.g. `make fmt` / `gofmt -w .` / `npm run format`} | before commit |
| Lint | {e.g. `make lint` / `go vet ./...` / `npm run lint`} | before commit |
| Unit tests | {e.g. `make test` / `go test ./...` / `npm test` / `uv run pytest tests/unit`} | before advancing |
| Integration tests | {if applicable} | when crossing a seam |
| Full suite | {the full gate, e.g. `make fmt && make lint && make test`} | before finishing a feature |
| Frontend visual check | {if applicable} | when UI changes |

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

## Phase 3.5: Reconcile Mode (re-run after D04)

Run this phase **only when `mode` is `reconcile`** — i.e. bootstrap is re-invoked after product-design has produced D01/D04 (and possibly D07). The provisional stack already exists on disk; this phase revises it against the product evidence instead of regenerating from scratch. (Phase 0 has already read the product context by the time you arrive here.)

1. **Read the existing stack.** Load the current `tech-stack.md` (and `architecture-overview.md`, `integration-map.md` if they are affected).
2. **Diff against the evidence.** Using the Phase 0 summary, list each place where the vision, a D04 risk/verdict, or a D07 need argues for a different choice than the provisional stack records. A D04 verdict of **"feasible with changes"** or **"not feasible"** must be resolved here — name the change it forces.
3. **Rewrite the affected docs.** Update only the documents that change. Keep the greenfield header, drop the **Provisional** line once reconciled, and bump the `Generated` date.
4. **Append a Stack revision section** to `tech-stack.md` recording every divergence and its driver:

```markdown
## Stack revision — YYYY-MM-DD

Reconciled against: D01 vision{, D04 feasibility}{, D07 user needs}.

| Was | Now | Driver |
|-----|-----|--------|
| {prior choice} | {revised choice} | {vision element / D04 risk / D07 need that forced it} |

{If nothing changed: "No divergences — provisional stack holds against the vision and feasibility evidence."}
```

5. **Return `stack_revisions`** so the caller can surface what changed.

After reconcile, **skip Phase 4** — the next step is to resume product-design (D05+) on a stack that now matches the evidence, not the cold "develop an idea" hint.

---

## Phase 4: Context-Dependent Next Step

Skip this phase entirely when `mode` is `reconcile` (Phase 3.5 owns the next step). Otherwise, base the suggestion on the Phase 1 answer:

**Answer 1 (just tech stack)** — the stack *is* the deliverable; there is no vision to reconcile against:
> Foundation documents generated. When you're ready to develop an idea, use the product-design workflow (or `/brainstorming` to refine a rough one first).

**Answer 2 (needs refinement)** — lightweight placeholder only; idea refinement comes first:
> Placeholder `tech-stack.md` created. Use the product-design workflow (D01–D04) to define vision, user needs, and feasibility. Then re-run project bootstrap in **reconcile mode** to expand the placeholder into a complete tech-stack document. `/brainstorming` can sharpen the idea first.

**Answer 3 (solid idea)** — also **provisional** until feasibility confirms it:
> Provisional foundation generated. Use the product-design workflow to capture the vision and feasibility (D01–D04), then re-run project bootstrap to **reconcile** `tech-stack.md` against the D04 verdict before building.

---

## Success Criteria

**Readiness 1 and 3 (full path):**
- [ ] All 5 documents written to `docs/architecture/`
- [ ] All documents include greenfield header with date and stack summary
- [ ] Tech stack includes actual current version numbers (from web research)
- [ ] Architecture references official best practices (not generic patterns)
- [ ] File system structure matches official framework recommendations
- [ ] Idea-readiness 3 docs carry the **Provisional** header line

**Readiness 2 (lightweight placeholder path):**
- [ ] Only `tech-stack.md` written (placeholder with stack signal or TBD)
- [ ] Placeholder carries the **Provisional placeholder** header
- [ ] No web research was performed, no other architecture docs were generated
- [ ] Next step routes to product-design then reconcile

**All paths:**
- [ ] User received context-appropriate next step suggestion
- [ ] Stack intake completed in ≤2 turns
- [ ] When product context (vision/feasibility/needs) was supplied, the stack was proposed from it — not asked cold
- [ ] Reconcile runs (`mode: reconcile`) appended a **Stack revision** section and returned `stack_revisions`

## Related Files

- `../context/stack-research-guide.md` — Authoritative sources per stack
- `bootstrap.md` — Orchestrator that invokes this workflow (supplies `estate`, `mode`, and product-artifact paths)
- `brownfield-analysis.md` — Alternative estate for existing codebases
- `../../product-design/workflows/d04-feasibility.md` — Produces the verdict that triggers reconcile mode
