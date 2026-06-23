# E32-F05 Brownfield Analysis — Repoint Harness, Deprecate Slash Commands, Simplify shark/SKILL.md

**Date**: 2026-06-22
**Status**: In research (supporting task decomposition)
**Resolution path**: Restore + Deprecate (Path A per assessment)

---

## Executive Summary

The machine is at a clean slate: all 8 in-scope slash commands exist without deprecation headers; all 10 in-scope skills and all 9 in-scope agents exist in `~/.claude/`. The `shark/SKILL.md` is still the old full CLI reference (246 lines), NOT the dispatch-loop rewrite the assessment incorrectly reported as complete. All canonical replacements are confirmed present in `internal/sharkdata/default_data/`. Three concrete tasks close F05: (1) add deprecation headers to 8 slash commands, (2) delete 10 skills and 9 agents from the harness, (3) rewrite `shark/SKILL.md`. AC-2 is verifiable without a live `/run` execution by inspecting the engine's include-resolution path.

**Assessment correction**: The assessment (2026-06-22) stated shark/SKILL.md was "already correct as-is — a dispatch-loop rewrite from the previous F05 attempt." This is false. The file is 246 lines of full shark CLI documentation identical to what it was before the previous F05 attempt. The previous rewrite did not survive the harness restore.

---

## 1. Slash Command Inventory

All 30 commands currently in `~/.claude/commands/`. The 8 in-scope commands are shown with their state and required change.

### In-Scope Commands (deprecation header required)

| File | Size | Current state | Required change | New entry point |
|------|------|---------------|-----------------|-----------------|
| `run.md` | 3.3k | Full functional dispatch loop referencing `orchestration` skill | Add deprecation header | `/shark` skill dispatch loop or `shark run` |
| `feature.md` | 7.1k | Feature refinement workflow start | Add deprecation header | `shark run <feature-key>` via the `shark` skill |
| `epic.md` | 1.2k | Epic PRD creation | Add deprecation header | `shark run <epic-key>` via the `shark` skill |
| `task.md` | 2.4k | Generate implementation tasks | Add deprecation header | `shark run <task-key>` via the `shark` skill |
| `prd.md` | 1.4k | Feature PRD creation | Add deprecation header | `shark run <feature-key>` at `ready_for_specification` |
| `dispatch.md` | 2.6k | Tech-director parallel dispatch | Add deprecation header | `shark run <epic/feature>` via the `shark` skill |
| `develop.md` | 8.0k | Development workflow | Add deprecation header | `shark run <feature-key>` at `ready_for_development` |
| `release.md` | 8.0k | Release cycle workflow | Add deprecation header | `shark run <epic-key>` at release phase |

### Out-of-Scope Commands (leave untouched)

These 22 commands are NOT in F05 scope. Do not touch them.

`check-balance.md`, `codex-exec.md`, `coding-standards.md`, `continue.md`, `council.md`, `gemini-exec.md`, `generate-epic-index.md`, `map-file-system.md`, `map.md`, `pause.md`, `plan-sprint.md`, `project-init.md`, `retro-sprint.md`, `review-remedy.md`, `run-agent-team.md`, `run-sprint-team.md`, `run-sprint.md`, `shark-sync.md`, `shark.md`, `validate-feature-design.md`, `validate-task-readiness.md`, `vision.md`

---

## 2. Deprecation Header Template

The header should be placed immediately after the YAML front matter, before the existing content body. It must be two logical parts: a YAML front-matter addition (if the command has front matter) and a callout block.

### Template

For commands that already have YAML front matter (all 8 in-scope commands do), insert after the closing `---` of the front matter:

```markdown
> **DEPRECATED (F05)** — This slash command is a legacy entry point. Use the `shark` skill
> dispatch loop instead: `/run E01-F02` becomes `shark run E01-F02` in the new harness, or
> simply invoke the `shark` skill and follow the orchestration loop. This command remains
> functional for one release (F05); it will be removed in F06.
```

For `/run` specifically, add the current entry point reference:

```markdown
> **DEPRECATED (F05)** — The `/run` dispatch loop has moved into `~/.claude/skills/shark/SKILL.md`.
> The `orchestration` skill is no longer the primary dispatch mechanism. This command remains
> functional for one release; it will be removed in F06.
```

### Placement rule

Insert the deprecation block as the first content line after the YAML front matter `---`, before any heading. This ensures it appears at the top of the rendered command body.

Example for `run.md`:

```markdown
---
description: Drive an entity through its shark workflow - spawns agents, advances statuses, cascades into children
---

> **DEPRECATED (F05)** — The `/run` dispatch loop has moved into `~/.claude/skills/shark/SKILL.md`.
> The `orchestration` skill is no longer the primary dispatch mechanism. This command remains
> functional for one release; it will be removed in F06.

# /run — Drive Entity Through Shark Workflow
...rest of existing content unchanged...
```

---

## 3. Skills to Delete from `~/.claude/skills/`

All 10 in-scope skills confirmed present. Verification method: `ls ~/.claude/skills/<name>/`.

| Skill directory | Exists | Lines in SKILL.md | Subdirs |
|-----------------|--------|-------------------|---------|
| `specification-writing/` | YES | 155 | `context/`, `workflows/`, `README.md` |
| `quality/` | YES | 92 | `context/`, `workflows/`, `README.md` |
| `architecture/` | YES | 120 | `context/`, `examples/`, `workflows/`, `README.md` |
| `research/` | YES | 182 | `context/`, `workflows/`, `README.md` |
| `implementation/` | YES | 229 | `context/`, `workflows/`, `README.md` |
| `test-driven-development/` | YES | 497 | `references/` |
| `assessment/` | YES | 599 | `assets/`, `references/` |
| `uat/` | YES | 411 | `references/` |
| `debugging/` | YES | 159 | `context/`, `workflows/`, `README.md` |
| `orchestration/` | YES | 78 | `workflows/` |

**Canonical replacements verified**: All 9 non-orchestration skills exist in `internal/sharkdata/default_data/skills/`. The `orchestration` skill has NO canonical replacement in sharkdata (intentional: it is replaced by the dispatch loop inlined in `shark/SKILL.md`).

### Canonical skill existence in sharkdata

| Skill | In `internal/sharkdata/default_data/skills/` |
|-------|----------------------------------------------|
| `specification-writing` | YES |
| `quality` | YES |
| `architecture` | YES |
| `research` | YES |
| `implementation` | YES |
| `test-driven-development` | YES |
| `assessment` | YES |
| `uat` | YES |
| `debugging` | YES |
| `orchestration` | NO — replaced by shark/SKILL.md dispatch loop |

### Skills to leave untouched (out of scope)

All 30 remaining skill directories stay: `brainstorming`, `brownfield-analysis`, `claude-md-improver`, `code-review`, `collaboration`, `command`, `ddd-lite-bounded-context`, `ddd-lite-entity`, `ddd-lite-principles`, `ddd-lite-repository`, `ddd-lite-ubiquitous-language`, `ddd-lite-value-object`, `devops`, `efficient-fable`, `feature-design`, `finish-feature`, `frontend-design`, `github-pr-review`, `graphify`, `mermaid`, `product-design`, `refactoring`, `resolve-pr-comments`, `shark`, `skill-creator`, `socratic-method`, `sprint-analytics`, `sprint-execution`, `sprint-planning`, `technical-writing`, `triage`, `vue-development`

---

## 4. Agents to Delete from `~/.claude/agents/`

All 9 in-scope agents confirmed present.

| Agent file | Exists | Size | Canonical in sharkdata |
|------------|--------|------|------------------------|
| `architect.md` | YES | 4,902 bytes | YES (4,902 bytes — identical) |
| `business-analyst.md` | YES | 13,542 bytes | YES (13,480 bytes — minor diff) |
| `developer.md` | YES | 28,046 bytes | YES (27,924 bytes — minor diff) |
| `qa.md` | YES | 20,796 bytes | YES (20,629 bytes — minor diff) |
| `tech-lead.md` | YES | 21,042 bytes | YES (20,778 bytes — minor diff) |
| `product-manager.md` | YES | 14,926 bytes | YES (14,926 bytes — identical) |
| `tech-director.md` | YES | 9,208 bytes | YES (9,208 bytes — identical) |
| `researcher.md` | YES | 11,598 bytes | YES (11,585 bytes — minor diff) |
| `uat-agent.md` | YES | 16,623 bytes | YES (16,529 bytes — minor diff) |

Minor byte differences are expected (sharkdata may be a slightly newer version).

### Agents to keep (out of scope)

These 6 agents stay in `~/.claude/agents/` — they have no sharkdata canonical and are harness-level helpers:

| Agent | Keep reason |
|-------|-------------|
| `client.md` | Harness-level product client persona, not in F05 scope |
| `code-simplifier.md` | Harness-level code quality helper |
| `cx-designer.md` | Customer experience, not in F05 scope |
| `devops.md` | Infrastructure helper, not in F05 scope |
| `human-checkpoint.md` | Harness-level approval gate |
| `ux-designer.md` | UX design helper, not in F05 scope |

Note: AC-4 says agents/ should contain "only harness-level helpers (Explore, Plan, general-purpose) and out-of-scope agents." The 6 above are the "out-of-scope agents" that remain. `Explore`, `Plan`, and `general-purpose` are built-in Claude Code agent types, not `~/.claude/agents/*.md` files.

---

## 5. shark/SKILL.md Rewrite

### Current state (incorrect — needs rewrite)

`~/.claude/skills/shark/SKILL.md` is currently 246 lines of full shark CLI documentation: entity key formats, every `shark` subcommand, JSON output patterns, orchestrator actions, and context commands. This is the pre-F05 content. The assessment's claim that "this content was produced by the previous F05 implementation and survived the harness restore" is false — what is present is the original, unmodified file.

The `shark/SKILL.md` at `~/.claude/skills/shark/` has no `workflows/` subdirectory. The current orchestration dispatch loop lives at `~/.claude/skills/orchestration/workflows/run.md` (which is being deleted in this feature).

### What the rewrite must contain

Per the feature spec, the rewrite should embed the dispatch loop. The feature's suggested skeleton:

```markdown
# shark — dispatch loop trigger

When the user invokes shark dispatch (/run <key>, etc.):

while true:
  response = shark next <key> --json
  case response.action:
    spawn_agent: spawn(response.agent_type, prompt=response.prompt)
                 → on completion: shark advance <key>
    pause:      stop, report status to user
    archive:    stop, report archive to user
```

The feature spec explicitly warns: "Don't budget F05 as if shark/SKILL.md is trivial. Pre-flight checks, agent-spawn adapters per harness, retry/error UX, 'pause' rendering, telemetry — some of this must stay harness-side."

The existing full dispatch loop is at `~/.claude/skills/orchestration/workflows/run.md` (179+ lines). The new `shark/SKILL.md` should incorporate the essential content from that file: entity detection, branch check, loop, rejection escalation guard, action handlers, and task dispatch loop for feature `active`. The shark CLI reference content (which accounts for most of the current 246 lines) can be trimmed to the essential commands that the dispatch loop needs inline, with a pointer to the `shark` skill's `context/` documents for the full reference.

The `shark/` skill directory already has `context/` files:
- `context/task-execution-pattern.md`
- `context/entity-crud.md`
- `context/workflow-and-status.md`
- `context/notes-context-docs.md`
- `HOOKS.md`

The rewrite should reference these context files for full shark CLI details rather than embedding all of them inline.

### Suggested structure for the rewrite

```markdown
---
name: shark
description: Query and manage project state - epics, features, tasks, notes, analytics. Use for ALL project state queries, task lifecycle management, status transitions (advance/set), blocking/unblocking, adding notes/context, viewing status dashboards, analytics, workflow management, creating/updating/deleting entities, and any command starting with "shark".
---

# Shark — Dispatch Loop + CLI Reference

## Dispatch Loop (primary mode: /run <key>)

[embed the run.md loop content here — entity detection, branch check, Step 1/2/3, action handlers, task dispatch loop, rejection escalation guard, activity log events]

## Shark CLI Quick Reference

[keep the Common Mistakes section + the minimal commands needed by the dispatch loop]

## Detailed References

- `context/task-execution-pattern.md` - Mandatory agent workflow
- `context/entity-crud.md` - Create, update, delete patterns
- `context/workflow-and-status.md` - Status transitions
- `context/notes-context-docs.md` - Notes, context, related docs
- `HOOKS.md` - Optional automation hooks
```

---

## 6. AC-2 Verification Plan

### What AC-2 tests

AC-2: "Move `~/.claude/skills/quality/` aside; `/run` workflow that uses quality still works (resolves via `shark-data/`)."

### Two skill resolution paths (critical distinction)

There are two independent resolution paths for "skills":

**Path A — Claude harness Skill tool**: When Claude's harness calls the `Skill` tool for a named skill, it looks at `~/.claude/skills/<name>/SKILL.md`. This path is what F05 deletes from.

**Path B — shark engine `{{include:}}`**: When shark's engine renders an instruction template (e.g., `feature/ready_for_qa.md`), it resolves `{{include: skills/quality/SKILL.md}}` against `<project>/shark-data/` as the data root. This is what AC-2 tests.

The engine code: `internal/templates/includes.go` → `IncludeResolver.Resolve()` → looks in `<dataRoot>/skills/<path>` where `dataRoot` is the parent of `shark-data/prompts/` (i.e., `<project>/shark-data/`). Source: `internal/templates/orchestrator_renderer.go:detectIncludeRoot()`.

The `{{include:}}` directives in prompts already reference `skills/quality/SKILL.md` (and other skills) directly from within `internal/sharkdata/default_data/prompts/`:

- `prompts/feature/ready_for_qa.md` → `{{include: skills/quality/SKILL.md}}`
- `prompts/feature/in_qa.md` → `{{include: skills/quality/SKILL.md}}`
- `prompts/feature/ready_for_test_planning.md` → `{{include: skills/quality/workflows/test-planning.md}}`
- `prompts/feature/in_code_review.md` → `{{include: skills/quality/workflows/review-code.md}}`
- `prompts/feature/ready_for_code_review.md` → `{{include: skills/quality/workflows/review-code.md}}`
- `prompts/_partials/_code_review_process.md` → `{{include: skills/quality/workflows/review-code.md}}`
- `prompts/_partials/_qa_process.md` → `{{include: skills/quality/SKILL.md}}`

Removing `~/.claude/skills/quality/` has NO effect on Path B. AC-2 is testing that Path B works — and it already does, since `shark-data/skills/quality/` is the source used at render time.

### AC-2 test steps (without live /run)

Since we cannot run a full live `/run` workflow without a full test project, AC-2 can be validated at two levels:

**Level 1 (static verification, runnable now)**:

```bash
# 1. Confirm quality skill is in shark-data canonical
ls /path/to/project/shark-data/skills/quality/SKILL.md

# 2. Confirm the prompts that reference quality use shark-data-relative paths
grep -r "include: skills/quality" internal/sharkdata/default_data/prompts/

# 3. Confirm IncludeResolver uses shark-data/ parent, NOT ~/.claude/skills/
grep -n "dataRoot\|detectIncludeRoot\|IncludeRoot" internal/templates/orchestrator_renderer.go

# 4. Move quality aside
mv ~/.claude/skills/quality /tmp/quality-aside

# 5. Run shark validate on the project (validates include resolution)
./bin/shark admin validate

# 6. Restore
mv /tmp/quality-aside ~/.claude/skills/quality
```

**Level 2 (functional, requires project with shark-data)**:

```bash
# After shark init has been run on a test project
cd /tmp/test-project
shark init

# Move quality aside from harness
mv ~/.claude/skills/quality /tmp/quality-aside

# Verify include resolution works
shark admin validate    # Should report no errors about quality includes

# Restore
mv /tmp/quality-aside ~/.claude/skills/quality
```

**What shark admin validate checks** (from `internal/sharkdata/embed.go:Validate()`): validates every `{{include:}}` directive in `shark-data/prompts/` resolves against `shark-data/` (not `~/.claude/skills/`). A clean validation pass confirms AC-2.

### Why AC-2 is structurally guaranteed to pass

The engine (`IncludeResolver`) never looks at `~/.claude/skills/`. It only looks at `<project>/shark-data/`. After `shark init` populates `shark-data/skills/quality/` from the embedded canonical, quality-skill resolution is independent of `~/.claude/skills/quality/`. This is architectural: it was the whole point of F4's consolidation.

---

## 7. Implementation Task Breakdown

These are the concrete tasks needed to close F05:

### Task 1: Add deprecation headers to 8 slash commands

**Files**: `~/.claude/commands/{run,feature,epic,task,prd,dispatch,develop,release}.md`

**Action**: Insert the deprecation block (2-3 lines) immediately after the YAML front matter `---` in each file, before the first heading. Leave all existing body content intact and functional.

**Verification**: Open each file and confirm the deprecation block appears at the top.

### Task 2: Delete 10 skills from `~/.claude/skills/`

**Directories to delete**:
```
~/.claude/skills/specification-writing/
~/.claude/skills/quality/
~/.claude/skills/architecture/
~/.claude/skills/research/
~/.claude/skills/implementation/
~/.claude/skills/test-driven-development/
~/.claude/skills/assessment/
~/.claude/skills/uat/
~/.claude/skills/debugging/
~/.claude/skills/orchestration/
```

**Verification**: `ls ~/.claude/skills/` should show none of the above; out-of-scope skills should still be present.

### Task 3: Delete 9 agents from `~/.claude/agents/`

**Files to delete**:
```
~/.claude/agents/architect.md
~/.claude/agents/business-analyst.md
~/.claude/agents/developer.md
~/.claude/agents/qa.md
~/.claude/agents/tech-lead.md
~/.claude/agents/product-manager.md
~/.claude/agents/tech-director.md
~/.claude/agents/researcher.md
~/.claude/agents/uat-agent.md
```

**Verification**: `ls ~/.claude/agents/` should show only `client.md`, `code-simplifier.md`, `cx-designer.md`, `devops.md`, `human-checkpoint.md`, `ux-designer.md`.

### Task 4: Rewrite `~/.claude/skills/shark/SKILL.md`

**File**: `~/.claude/skills/shark/SKILL.md`

**Action**: Replace the current 246-line full CLI reference with a new file that:
1. Keeps the YAML front matter (name, description)
2. Embeds the dispatch loop from `orchestration/workflows/run.md` (the content being deleted)
3. Retains only the "Common Mistakes to Avoid" section and essential CLI reference for the dispatch loop's use
4. Points to `context/` subdirectory files for full CLI reference

**Source material for the new content**: `~/.claude/skills/orchestration/workflows/run.md` (the full dispatch loop, branch check, rejection escalation guard, action handlers).

**Verification**: The new SKILL.md should contain the dispatch loop logic and reference the `orchestration/workflows/run.md` patterns (now inlined), not delegate to the `orchestration` skill.

### Task 5: Verify AC-2

Run `shark admin validate` after `shark init` on a test project, with `~/.claude/skills/quality/` moved aside. Should report no errors.

---

## 8. Key Files and Paths

| Item | Path |
|------|------|
| Slash commands | `~/.claude/commands/*.md` |
| Harness skills | `~/.claude/skills/` |
| Harness agents | `~/.claude/agents/` |
| shark skill SKILL.md (to rewrite) | `~/.claude/skills/shark/SKILL.md` |
| shark skill context files | `~/.claude/skills/shark/context/` |
| Canonical skills (source of truth) | `internal/sharkdata/default_data/skills/` |
| Canonical agents (source of truth) | `internal/sharkdata/default_data/agents/` |
| Engine include resolver | `internal/templates/includes.go` |
| Engine data root detection | `internal/templates/orchestrator_renderer.go:detectIncludeRoot()` |
| Prompt includes referencing quality | `internal/sharkdata/default_data/prompts/feature/ready_for_qa.md`, `in_qa.md`, `in_code_review.md`, `ready_for_code_review.md`, `ready_for_test_planning.md`, `_partials/_code_review_process.md`, `_partials/_qa_process.md` |
| Orchestration dispatch loop (to migrate into shark/SKILL.md) | `~/.claude/skills/orchestration/workflows/run.md` |

---

## 9. What Does NOT Need to Change

- `~/.claude/skills/shark/context/` files — leave all 4 context files unchanged
- `~/.claude/skills/shark/HOOKS.md` — leave unchanged
- Out-of-scope slash commands — 22 commands, leave all untouched
- Out-of-scope skills — 30+ skill directories, leave all untouched
- Out-of-scope agents — 6 agent files, leave all untouched
- `shark-data/` in the project root — F05 is harness-side changes only
- `internal/sharkdata/default_data/` — not touched by F05
- Any Go code — F05 is harness-only (no binary changes)
