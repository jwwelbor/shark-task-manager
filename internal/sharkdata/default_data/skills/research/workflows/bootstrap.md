---
inputs:
  - project_root: absolute path to the project being initialized
  - output_dir: absolute path to the architecture docs directory (where the output files are written)
  - file_system_path: absolute path for file-system.md
  - coding_standards_path: absolute path for coding-standards.md
  - tech_stack_path: absolute path for tech-stack.md
  - architecture_overview_path: absolute path for architecture-overview.md
  - patterns_catalog_path: absolute path for patterns-catalog.md
  - integration_map_path: absolute path for integration-map.md
  - marker_path: absolute path for the bootstrap marker file
  - existing_marker: parsed marker contents from a prior run (null if first run)
  - rerun_choice: fill-gaps | regenerate-all | reconcile-stack | cancel (null if no prior marker)
  - initiative_posture: stack-only | new-capability | extend | modernize | replace (confirmed by the host)
  - host_state_status: free-form string the wrapper wants recorded in the marker (e.g. "shark: initialized", "none") — workflow does not interpret this
  - generation_date: ISO date for marker and document headers
outputs:
  - estate: brownfield | greenfield
  - initiative_posture: confirmed posture or null when setup is paused before confirmation
  - confidence: HIGH | MEDIUM | LOW
  - detection_signals: list of signals that drove the estate decision
  - stack_summary: short string summarizing the resolved stack
  - generated_files: list of {path, status: created | updated | skipped | failed}
  - inferred_decisions: list of architectural decisions inferred during analysis (for ADR seeding)
  - idea_readiness: 1 | 2 | 3 | null (only set for greenfield)
  - stack_revisions: list of {doc, was, now, driver} entries when a greenfield reconcile ran (null otherwise)
  - next_step_hint: short string the host can surface to the user (e.g. "ready for vision", "ready for brainstorming")
---

# Workflow: Bootstrap Orchestrator

**Purpose**: Detect brownfield vs greenfield, route to the correct estate, and produce a stable set of architecture foundation files
**Use for**: Project bootstrapping at the start of a new repository or before formal planning begins
**Output**: Architecture docs in the caller-supplied directory (see Output Contract below)

## Output Contract

| File | Purpose | When created |
|------|---------|--------------|
| `file-system.md` | Project structure map | Brownfield, greenfield readiness 1 or 3 |
| `coding-standards.md` | Enforceable code rules with examples | Brownfield, greenfield readiness 1 or 3 |
| `tech-stack.md` | Languages, frameworks, versions, rationale | All greenfield paths (placeholder for readiness 2) |
| `architecture-overview.md` | Components, boundaries, data flow | Brownfield, greenfield readiness 1 or 3 |
| `patterns-catalog.md` | Design patterns with file:line refs | Brownfield, greenfield readiness 1 or 3 |
| `integration-map.md` | APIs, data stores, external services | Brownfield, greenfield readiness 1 or 3 |
| `bootstrap.md` | Marker file: estate, initiative posture, date, stack summary, file status | All paths |

> **Readiness 2 (idea needs refinement)** creates only `tech-stack.md` as a placeholder and the marker.
> The remaining docs are generated on the reconcile-stack pass after product-design completes.

Output paths come from the caller's inputs. The standard convention is `docs/architecture/<file>.md`, but the workflow does not assume that.

## Required Tools

- **Read** — Files and configs
- **Grep** — Pattern search
- **Glob** — File discovery
- **Bash** — Directory listing, git commands
- **WebSearch** — Stack research
- **AskUserQuestion** — Re-run policy, greenfield stack selection
- **Write** — Output documents
- **Agent** — Parallel subagent dispatch for brownfield analysis

---

## Phase 0: Pre-Flight

### Step 0.1: Confirm Project Root

The caller supplies `project_root`. If it points to a tool/config directory (e.g., a `.claude/`, `.codex/`, or similar dotfile directory) rather than a real project, ask the user for the actual target project path before continuing.

### Step 0.2: Check for Existing Marker

If `existing_marker` is provided (a previous run wrote to `marker_path`), use `rerun_choice` to decide:

- **fill-gaps** — Check which files exist on disk; only generate the missing ones.
- **regenerate-all** — Proceed as if fresh run, overwrite all.
- **reconcile-stack** — *(greenfield only, and only when `docs/product/D04-feasibility-report.md` now exists)* re-run `greenfield-scaffold.md` in **reconcile mode** to revise `tech-stack.md` against the vision + feasibility product-design has produced since the last bootstrap. This is the reverse-feed step: the provisional stack (or placeholder) chosen at first bootstrap is now tested against D04. See Phase 2 → Greenfield estate.
- **cancel** — Exit with "No changes made".

If `rerun_choice` is null but `existing_marker` is present, ask the user (offer **Reconcile stack** only when the estate is greenfield and `docs/product/D04-feasibility-report.md` exists):

> This project was previously bootstrapped ({estate} estate, {date}).
>
> 1. **Fill gaps only** — Regenerate only missing files (Recommended)
> 2. **Regenerate all** — Overwrite all foundation documents
> 3. **Reconcile stack** — Revise tech-stack.md against the vision + feasibility produced since bootstrap *(greenfield, only if D04 exists)*
> 4. **Cancel** — Keep everything as-is

### Step 0.3: Ensure Output Directory

```bash
mkdir -p {output_dir}
```

---

## Phase 1: estate Detection

Follow the algorithm in `../context/brownfield-detection.md`:

### Step 1.1: Build Manifest Check

```
Glob: {project_root}/**/package.json (depth ≤2, exclude node_modules)
Glob: {project_root}/**/go.mod (depth ≤2)
Glob: {project_root}/**/pyproject.toml (depth ≤2)
Glob: {project_root}/**/Cargo.toml (depth ≤2)
Glob: {project_root}/**/pom.xml (depth ≤2)
Glob: {project_root}/**/Gemfile (depth ≤2)
```

If manifest found with real dependencies → **brownfield**, skip to Phase 2.

### Step 1.2: Source File Count

```
Glob: {project_root}/**/*.{ts,tsx,js,jsx,py,go,rs,java,rb,php} (exclude vendor dirs)
```

- \>5 files → **brownfield**
- 1-5 files → ask user
- 0 files → continue

### Step 1.3: Git History

```bash
git -C {project_root} rev-list --count HEAD 2>/dev/null
```

- \>3 commits → **brownfield**
- 1-3 → check for template clone (per detection rules)
- 0 or not git → **greenfield**

### Step 1.4: Record Detection Result

Set:
- `estate`: brownfield | greenfield
- `confidence`: HIGH | MEDIUM | LOW
- `detection_signals`: list of what was detected

Announce to user: "Detected **{estate}** project ({confidence} confidence based on {primary signal})."

---

## Phase 2: Route to estate Workflow

### Brownfield estate

Execute in parallel groups:

**Group A** (parallel with B, C):
- Invoke `map-filesystem.md` workflow → produces `file-system.md` at `file_system_path`

**Group B+C** (parallel with A):
- Invoke `brownfield-analysis.md` workflow → produces:
  - `tech-stack.md` (Part 1) at `tech_stack_path`
  - `patterns-catalog.md` (Part 2) at `patterns_catalog_path`
  - `integration-map.md` (Part 3) at `integration_map_path`
  - `architecture-overview.md` (Part 4) at `architecture_overview_path`

**Implementation**: Use Agent tool to dispatch Group A and Group B+C as parallel subagents. Each subagent reads the relevant workflow file and executes it against `project_root` with the supplied output paths.

Wait for all groups to complete before Phase 3.

### Greenfield estate

Invoke `greenfield-scaffold.md`, passing the **estate** (`greenfield`), a **mode**, and any product-design artifacts that already exist so the scaffold can read product context:

- `estate`: `greenfield`
- `mode`: `reconcile` when `rerun_choice` is `reconcile-stack` (or the host explicitly requests a stack reconcile and `docs/product/D04-feasibility-report.md` exists); otherwise `provisional`.
- `vision_path`, `feasibility_path`, `user_needs_path`: pass each of `docs/product/D01-vision-statement.md`, `docs/product/D04-feasibility-report.md`, `docs/product/D07-user-needs.md` that exists; null otherwise.

**Provisional mode** (first pass — no feasibility yet):

1. Idea readiness assessment (greenfield Phase 1) — capture as `idea_readiness`.
2. Stack intake (Phase 1b). When product context exists, the scaffold proposes from evidence and confirms in one turn instead of asking cold.
3. **Readiness 2 (needs refinement) → Phase 1c** — write a `tech-stack.md` placeholder only. Skip web research and architecture docs. Route directly to product-design. No further phases run.
4. **Readiness 1 or 3 → Phases 2 and 3** — web research for stack, then generate the 5 prescriptive docs:
   - `tech-stack.md`
   - `architecture-overview.md`
   - `file-system.md` (prescriptive, not scanned)
   - `patterns-catalog.md`
   - `integration-map.md`

   Readiness 3 docs are marked **provisional** — they are reconciled after D04.

**Reconcile mode** (re-run after product-design): the scaffold reads the product context (its Phase 0) then runs **Phase 3.5** — it diffs the provisional stack (or expands the placeholder) against D01/D04/D07, rewrites the affected docs, and appends a *Stack revision* section. Capture the scaffold's `stack_revisions` for the Phase 5 summary. Coding standards (Phase 3) run only if `tech-stack.md` changed.

Record the idea readiness answer for `next_step_hint` in Phase 5.

Proceed to Phase 3 (brownfield or greenfield readiness 1/3 only).

---

## Phase 3: Coding Standards Generation (Group D)

> **Skip for greenfield readiness 2** — no complete tech-stack exists yet to base standards on.
> Coding standards are generated on the reconcile-stack pass after product-design.

This runs **after** Phase 2 completes (brownfield or greenfield readiness 1/3), because it reads the generated documents.

### Step 3.1: Read Generated Context

Read the just-generated documents:
- `tech_stack_path` — to know the stack
- `patterns_catalog_path` — to know discovered/recommended patterns

### Step 3.2: Web Research for Standards

Using `../context/stack-research-guide.md`, research authoritative coding standards for the detected/chosen stack:
- Official style guide for the primary language
- Framework-specific conventions
- Community-established best practices

### Step 3.3: Generate Standards

**Brownfield**: Invoke the existing coding-standards workflow (`skills/quality/workflows/generate-standards.md`) with additional context from:
- Discovered patterns (from `patterns-catalog.md`)
- Official standards (from web research)
- Reconciliation using the augmentation pattern in `stack-research-guide.md`

Output:
- `coding-standards.md` at `coding_standards_path` — Enforceable standards
- A divergences/gaps document (e.g., `docs/plan/tech-debt/coding-standards-gaps.md` — caller may override) capturing differences from official guidance

**Greenfield**: Generate prescriptive coding standards based purely on:
- Official style guide for chosen language
- Framework conventions from official docs
- Web research results

Output:
- `coding-standards.md` at `coding_standards_path` — Prescriptive standards with greenfield header

---

## Phase 4: Write Marker File

Write the marker to `marker_path`:

```markdown
# Project Init

**estate**: {brownfield | greenfield}
**Initiative Posture**: {stack-only | new-capability | extend | modernize | replace | unconfirmed}
**Date**: {generation_date}
**Stack**: {stack_summary — e.g., "TypeScript / Next.js 14 / PostgreSQL / Prisma", or "Provisional: Go (stated constraint)" for readiness 2}
**Host State**: {host_state_status — supplied by the wrapper, e.g., "shark: initialized" or "none"}

## Generated Files

| File | Status | Generated |
|------|--------|-----------|
| file-system.md | {created | updated | skipped | n/a (readiness 2)} | {date or —} |
| coding-standards.md | {created | updated | skipped | n/a (readiness 2)} | {date or —} |
| tech-stack.md | {created | updated | skipped} | {date} |
| architecture-overview.md | {created | updated | skipped | n/a (readiness 2)} | {date or —} |
| patterns-catalog.md | {created | updated | skipped | n/a (readiness 2)} | {date or —} |
| integration-map.md | {created | updated | skipped | n/a (readiness 2)} | {date or —} |
| bootstrap.md | created | {date} |

## Detection Signals

{List the brownfield/greenfield detection signals and their confidence}

## Notes

{Any notable findings, warnings, or recommendations from the analysis}
```

---

## Phase 5: Output Summary + Next Steps

Display to user:

```markdown
## Bootstrap Complete

**estate**: {brownfield | greenfield}
**Initiative Posture**: {stack-only | new-capability | extend | modernize | replace | unconfirmed}
**Stack**: {stack_summary}
**Files generated**: {count}

| File | Status |
|------|--------|
| {file_system_path} | {created/updated/existed/skipped} |
| {coding_standards_path} | {created/updated/existed/skipped} |
| {tech_stack_path} | {created/updated/existed} |
| {architecture_overview_path} | {created/updated/existed/skipped} |
| {patterns_catalog_path} | {created/updated/existed/skipped} |
| {integration_map_path} | {created/updated/existed/skipped} |
| {marker_path} | {created/updated} |
```

When a greenfield reconcile ran, also show the **Stack revision** summary from the scaffold's `stack_revisions`:

```markdown
**Stack revisions** (reconciled against vision + feasibility):
| Was | Now | Driver |
|-----|-----|--------|
| {prior} | {revised} | {vision element / D04 risk / D07 need} |
```

Set `next_step_hint` based on estate, mode, and (for greenfield) `idea_readiness`. The wrapper is responsible for converting the hint into a host-specific suggestion.

| estate / Readiness / Mode | next_step_hint |
|---|---|
| Brownfield | "review-architecture-then-vision" |
| Greenfield, answer 1 | "ready-for-brainstorming" |
| Greenfield, answer 2 (placeholder) | "vision-then-feasibility-then-reconcile-stack" |
| Greenfield, answer 3 (provisional) | "vision-then-feasibility-then-reconcile-stack" |
| Greenfield, reconcile mode | "stack-reconciled-resume-product-design" |

**Carry the estate and initiative posture forward.** Product-design must know
whether the repository is brownfield or greenfield, plus which current-state
elements are hard, soft, or unresolved, so D04 frames feasibility correctly.
The marker records the observed **estate** and available posture evidence;
product-design reads it from `docs/architecture/bootstrap.md` (plus
`tech-stack.md` when it exists). This workflow only returns the hint — the
owning Rider action acts on it. Greenfield provisional hints encode the loop
the host runs: vision + feasibility (D01–D04), then the host re-invokes
bootstrap in reconcile mode (option 3 in Step 0.2) to fill out or revise the
stack against the D04 verdict.

---

## Error Handling

| Error | Recovery |
|-------|----------|
| Not in a project directory | Ask user for project path |
| Permission denied on output_dir | `mkdir -p` and retry |
| Web search fails | Proceed without web research, note in marker |
| Subagent times out | Report partial results, mark failed files in marker |
| Already initialized (fill-gaps) | Only generate files that don't exist |
| Reconcile requested but no D04 | Tell user feasibility hasn't run yet; offer fill-gaps / regenerate-all instead |
| Reconcile requested on brownfield | Reconcile the target against observed evidence and the selected initiative posture; route only deferred gaps via D04's tech-debt / constraint-note path |

## Success Criteria

- [ ] Marker file accurately reflects what was generated
- [ ] Brownfield docs contain real file:line references
- [ ] Greenfield readiness 1/3 docs contain current version numbers from web research
- [ ] Greenfield readiness 2: only `tech-stack.md` placeholder written (no web research, no other docs)
- [ ] Greenfield passes `estate` + `mode` + available product-artifact paths to `greenfield-scaffold.md`
- [ ] A greenfield reconcile run surfaces `stack_revisions` in the summary
- [ ] `next_step_hint` is set so the wrapper can route the user appropriately

## Related Files

- `brownfield-analysis.md` — Brownfield estate (Groups B+C)
- `greenfield-scaffold.md` — Greenfield estate (provisional + reconcile modes)
- `map-filesystem.md` — File system mapping (Group A)
- `../context/brownfield-detection.md` — Detection algorithm
- `../context/stack-research-guide.md` — Stack research patterns
- `skills/quality/workflows/generate-standards.md` — Coding standards generation
- `../../product-design/workflows/d04-feasibility.md` — Consumes the marker + stack; its verdict drives reconcile
