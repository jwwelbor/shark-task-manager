---
inputs:
  - project_root: absolute path to the project being initialized
  - output_dir: absolute path to the architecture docs directory (where the 7 output files are written)
  - file_system_path: absolute path for file-system.md
  - coding_standards_path: absolute path for coding-standards.md
  - tech_stack_path: absolute path for tech-stack.md
  - architecture_overview_path: absolute path for architecture-overview.md
  - patterns_catalog_path: absolute path for patterns-catalog.md
  - integration_map_path: absolute path for integration-map.md
  - marker_path: absolute path for the project-init marker file
  - existing_marker: parsed marker contents from a prior run (null if first run)
  - rerun_choice: fill-gaps | regenerate-all | cancel (null if no prior marker)
  - host_state_status: free-form string the wrapper wants recorded in the marker (e.g. "shark: initialized", "none") — workflow does not interpret this
  - generation_date: ISO date for marker and document headers
outputs:
  - track: brownfield | greenfield
  - confidence: HIGH | MEDIUM | LOW
  - detection_signals: list of signals that drove the track decision
  - stack_summary: short string summarizing the resolved stack
  - generated_files: list of {path, status: created | updated | skipped | failed}
  - inferred_decisions: list of architectural decisions inferred during analysis (for ADR seeding)
  - idea_readiness: 1 | 2 | 3 | null (only set for greenfield)
  - next_step_hint: short string the host can surface to the user (e.g. "ready for vision", "ready for brainstorming")
---

# Workflow: Project Init Orchestrator

**Purpose**: Detect brownfield vs greenfield, route to the correct track, and produce a stable set of architecture foundation files
**Use for**: Project bootstrapping at the start of a new repository or before formal planning begins
**Output**: 7 files in the caller-supplied architecture directory (see Output Contract below)

## Output Contract

| File | Purpose |
|------|---------|
| `file-system.md` | Project structure map |
| `coding-standards.md` | Enforceable code rules with examples |
| `tech-stack.md` | Languages, frameworks, versions, rationale |
| `architecture-overview.md` | Components, boundaries, data flow |
| `patterns-catalog.md` | Design patterns with file:line refs |
| `integration-map.md` | APIs, data stores, external services |
| `project-init.md` | Marker file: track, date, stack summary, file status |

Output paths come from the caller's inputs (`file_system_path`, `coding_standards_path`, etc.). The standard convention is `docs/architecture/<file>.md`, but the workflow does not assume that.

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

The caller supplies `project_root`. If it points to a config directory (e.g., `~/.claude/`), ask the user for the actual target project path before continuing.

### Step 0.2: Check for Existing Marker

If `existing_marker` is provided (a previous run wrote to `marker_path`), use `rerun_choice` to decide:

- **fill-gaps** — Check which of the 7 files exist on disk; only generate the missing ones.
- **regenerate-all** — Proceed as if fresh run, overwrite all.
- **cancel** — Exit with "No changes made".

If `rerun_choice` is null but `existing_marker` is present, ask the user:

> This project was previously initialized ({track} track, {date}).
>
> 1. **Fill gaps only** — Regenerate only missing files (Recommended)
> 2. **Regenerate all** — Overwrite all foundation documents
> 3. **Cancel** — Keep everything as-is

### Step 0.3: Ensure Output Directory

```bash
mkdir -p {output_dir}
```

---

## Phase 1: Track Detection

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
- `track`: brownfield | greenfield
- `confidence`: HIGH | MEDIUM | LOW
- `detection_signals`: list of what was detected

Announce to user: "Detected **{track}** project ({confidence} confidence based on {primary signal})."

---

## Phase 2: Route to Track Workflow

### Brownfield Track

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

### Greenfield Track

Invoke `greenfield-scaffold.md` workflow:

1. Idea readiness assessment (Phase 1 of greenfield workflow) — capture as `idea_readiness`
2. Stack determination (Phase 1b — 1-2 turns max)
3. Web research for stack (Phase 2)
4. Generate 5 prescriptive docs (Phase 3) at the supplied output paths:
   - `tech-stack.md`
   - `architecture-overview.md`
   - `file-system.md` (prescriptive, not scanned)
   - `patterns-catalog.md`
   - `integration-map.md`

Record the idea readiness answer for `next_step_hint` in Phase 5.

Proceed to Phase 3.

---

## Phase 3: Coding Standards Generation (Group D)

This runs **after** Phase 2 completes, because it reads the generated documents.

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

**Track**: {brownfield | greenfield}
**Date**: {generation_date}
**Stack**: {stack_summary — e.g., "TypeScript / Next.js 14 / PostgreSQL / Prisma"}
**Host State**: {host_state_status — supplied by the wrapper, e.g., "shark: initialized" or "none"}

## Generated Files

| File | Status | Generated |
|------|--------|-----------|
| file-system.md | {created | updated | skipped} | {date} |
| coding-standards.md | {created | updated | skipped} | {date} |
| tech-stack.md | {created | updated | skipped} | {date} |
| architecture-overview.md | {created | updated | skipped} | {date} |
| patterns-catalog.md | {created | updated | skipped} | {date} |
| integration-map.md | {created | updated | skipped} | {date} |
| project-init.md | created | {date} |

## Detection Signals

{List the brownfield/greenfield detection signals and their confidence}

## Notes

{Any notable findings, warnings, or recommendations from the analysis}
```

---

## Phase 5: Output Summary + Next Steps

Display to user:

```markdown
## Project Init Complete

**Track**: {brownfield | greenfield}
**Stack**: {stack_summary}
**Files generated**: {count}/7

| File | Status |
|------|--------|
| {file_system_path} | {created/updated/existed} |
| {coding_standards_path} | {created/updated/existed} |
| {tech_stack_path} | {created/updated/existed} |
| {architecture_overview_path} | {created/updated/existed} |
| {patterns_catalog_path} | {created/updated/existed} |
| {integration_map_path} | {created/updated/existed} |
| {marker_path} | {created/updated} |
```

Set `next_step_hint` based on track and (for greenfield) `idea_readiness`. The wrapper is responsible for converting the hint into a host-specific suggestion (slash-command names, scheduled steps, etc.).

| Track / Readiness | next_step_hint |
|---|---|
| Brownfield | "review-architecture-then-vision" |
| Greenfield, answer 1 | "ready-for-brainstorming" |
| Greenfield, answer 2 | "ready-for-brainstorming-then-vision" |
| Greenfield, answer 3 | "ready-for-vision" |

---

## Error Handling

| Error | Recovery |
|-------|----------|
| Not in a project directory | Ask user for project path |
| Permission denied on output_dir | `mkdir -p` and retry |
| Web search fails | Proceed without web research, note in marker |
| Subagent times out | Report partial results, mark failed files in marker |
| Already initialized (fill-gaps) | Only generate files that don't exist |

## Success Criteria

- [ ] All 7 files written to the supplied output paths
- [ ] Marker file accurately reflects what was generated
- [ ] Brownfield docs contain real file:line references
- [ ] Greenfield docs contain current version numbers from web research
- [ ] `next_step_hint` is set so the wrapper can route the user appropriately

## Related Files

- `brownfield-analysis.md` — Brownfield track (Groups B+C)
- `greenfield-scaffold.md` — Greenfield track
- `map-filesystem.md` — File system mapping (Group A)
- `../context/brownfield-detection.md` — Detection algorithm
- `../context/stack-research-guide.md` — Stack research patterns
- `skills/quality/workflows/generate-standards.md` — Coding standards generation
