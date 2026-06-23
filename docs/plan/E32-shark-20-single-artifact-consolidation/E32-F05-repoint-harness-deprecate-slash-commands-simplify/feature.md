---
feature_key: E02-F05-repoint-harness-deprecate-slash-commands-simplify
epic_key: E02
title: Repoint harness, deprecate slash commands, simplify shark/SKILL.md
description: Harness becomes minimal — shark/SKILL.md is the only shark-specific thing in ~/.claude/skills/. Delete in-scope skills and agents from the harness, delete the orchestration skill, rewrite shark/SKILL.md to include the tiny dispatch loop, add deprecation headers to slash commands.
size: M
---

# Repoint harness, deprecate slash commands, simplify shark/SKILL.md

**Feature Key**: E02-F05

---

## Epic

- **Epic PRD**: [Epic](../epic.md)

---

## Goal

### Problem

After F4, `shark-data/` ships canonical defaults and is the source of truth at dispatch time. But the harness still has stale duplicates: in-scope skills under `~/.claude/skills/`, in-scope agents under `~/.claude/agents/`, the now-redundant `orchestration` skill, and slash commands wired to the old dispatch path.

Until those are cleared, two skills and two agents claim ownership of the same name, and `~/.claude/` carries baggage shark 2.0 was supposed to eliminate.

### Solution

Repoint the harness:

- **Delete in-scope skills** from `~/.claude/skills/`: specification-writing, quality, architecture, research, implementation, test-driven-development, assessment, uat, debugging. They live in `shark-data/skills/` and are inlined at render time.
- **Delete the `orchestration` skill** — replaced by the `shark next` loop.
- **Delete in-scope agents** from `~/.claude/agents/`: architect, business-analyst, developer, qa, tech-lead, product-manager, tech-director, researcher, uat-agent. They live in `shark-data/agents/`.
- **Rewrite `~/.claude/skills/shark/SKILL.md`** to include the tiny dispatch loop instruction (`while true: shark next; spawn agent; shark advance`).
- **Add deprecation header** to slash commands: `/run`, `/feature`, `/epic`, `/task`, `/prd`, `/dispatch`, `/develop`, `/release`. Point to new entry points. Leave functional for one release; remove in F6.

### Impact

- A fresh project + a fresh harness install can run shark workflows entirely from `shark-data/` plus only the `shark` skill in the harness.
- `~/.claude/skills/` shrinks dramatically; out-of-scope skills (brainstorming, frontend-design, etc.) untouched.
- `~/.claude/agents/` keeps only harness-level helpers (Explore, Plan, general-purpose).

---

## Scope

### Skills to delete from `~/.claude/skills/`

- `specification-writing/`
- `quality/`
- `architecture/`
- `research/`
- `implementation/`
- `test-driven-development/`
- `assessment/`
- `uat/`
- `debugging/`
- `orchestration/`

### Agents to delete from `~/.claude/agents/`

- `architect.md`
- `business-analyst.md`
- `developer.md`
- `qa.md`
- `tech-lead.md`
- `product-manager.md`
- `tech-director.md`
- `researcher.md`
- `uat-agent.md`

### `~/.claude/skills/shark/SKILL.md` rewrite

Tiny dispatch trigger. Approximately:

```markdown
# shark — dispatch loop trigger

When the user invokes shark dispatch (`/run <key>`, etc.):

while true:
  response = shark next <key> --json
  case response.action:
    spawn_agent: spawn(response.agent_type, prompt=response.prompt)
                 → on completion: shark advance <key>
    pause:      stop, report status to user
    archive:    stop, report archive to user
```

**Reality check**: pre-flight checks, agent-spawn adapters per harness, retry/error UX, "pause" rendering, telemetry — some of this *must* stay harness-side because each harness spawns agents differently. The 5-line dispatch loop is the happy-path skeleton; in practice the file will grow with real-world edges. **Don't budget F5 as if `shark/SKILL.md` is trivial.**

### Slash commands — deprecation headers

Add a header to each that points to the new entry point. Leave functional for one release.

- `/run`
- `/feature`
- `/epic`
- `/task`
- `/prd`
- `/dispatch`
- `/develop`
- `/release`

---

## Acceptance Criteria / Exit gate

1. A fresh project + a fresh harness install can run shark workflows entirely from `shark-data/` plus only the `shark` skill in the harness.
2. `move ~/.claude/skills/quality/ aside; /run` workflow that uses quality still works (because everything resolves from `shark-data/skills/`).
3. `/run` slash command works with deprecation header shown but functional.
4. `~/.claude/agents/` contains only harness-level helpers (Explore, Plan, general-purpose) and out-of-scope agents.
5. `~/.claude/skills/` contains only out-of-scope skills + `shark/SKILL.md`.

---

## Out of Scope (for F5)

- Removing the `shark-templates/` fallback (F6).
- Removing deprecated slash commands (F6).
- Removing the legacy `.sharkworkflow.json` reader (F6).

---

## Dependencies

- **Blocks**: F6.
- **Blocked by**: F4.

---

## Risks

- **Global hard cutover for the shared harness.** Skills under `~/.claude/skills/` are shared across every project on the machine. F5 deletes them. Any project that hasn't run `shark init` yet (and pulled `shark-data/`) breaks at that moment. **Mitigation**: F5 should ship harness skills with deprecation warnings for one release before deletion — not a single commit.
- **The "tiny `shark/SKILL.md`" claim won't survive harness diversity.** Plan accordingly: the file will grow with real-world edges.
- **Status vocabulary leaks into consumers shark can't see.** Hooks under `~/.claude/hooks/`, scripts under `shark-task-manager/scripts/`, dashboards, CLAUDE.md content, slash commands, even muscle memory all speak shark's status vocabulary. F5 doesn't decouple any of these. **Audit needed during F4** before declaring `shark-templates/` retired.

---

---

## Implementation Spec

All changes are harness-side file edits only. No Go binary changes. No `shark-data/` changes.

### Step 1 — Delete 10 skills from `~/.claude/skills/`

Remove these directories entirely (each has subdirs; `rm -rf` each):

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

Verification after deletion:
```bash
ls ~/.claude/skills/
# Must NOT contain any of the above; shark/ must remain
```

The following 30+ skills remain untouched (representative list): `brainstorming`, `brownfield-analysis`, `code-review`, `collaboration`, `ddd-lite-*`, `devops`, `feature-design`, `frontend-design`, `graphify`, `mermaid`, `product-design`, `refactoring`, `resolve-pr-comments`, `shark`, `skill-creator`, `triage`, `vue-development`, etc.

### Step 2 — Delete 9 agents from `~/.claude/agents/`

Remove these files:

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

Verification after deletion:
```bash
ls ~/.claude/agents/
# Must contain ONLY these 6 files:
#   client.md  code-simplifier.md  cx-designer.md
#   devops.md  human-checkpoint.md  ux-designer.md
```

Note: `Explore`, `Plan`, and `general-purpose` are built-in Claude Code agent types — they do not appear as files in `~/.claude/agents/`.

### Step 3 — Rewrite `~/.claude/skills/shark/SKILL.md`

**Current state**: 246-line full shark CLI reference (original, pre-F05 content). The assessment incorrectly stated this file was already rewritten — it was not. The rewrite did not survive the harness restore.

**Source material for the new content**: `~/.claude/skills/orchestration/workflows/run.md` contains the full dispatch loop (entity detection, branch check, rejection escalation guard, action handlers, task dispatch loop, activity log events). This file is deleted in Step 1; its content must be migrated into `shark/SKILL.md` before deletion.

**New file structure**:

```markdown
---
name: shark
description: Query and manage project state - epics, features, tasks, notes, analytics. Use for ALL project state queries, task lifecycle management, status transitions (advance/set), blocking/unblocking, adding notes/context, viewing status dashboards, analytics, workflow management, creating/updating/deleting entities, and any command starting with "shark".
---

# Shark — Dispatch Loop + CLI Reference

## Dispatch Loop (invoked via /run <key> or shark run <key>)

[FULL content migrated from ~/.claude/skills/orchestration/workflows/run.md]
[Includes: entity detection, branch check, Step 0/1/2, spawn_agent handler,
 rejection escalation guard (2-strikes), check_or_resume, wait_for_triage,
 pause, archive, cascade, task dispatch loop, activity log event table]

## Common Mistakes to Avoid

[Keep the 6-item list from the current SKILL.md — these are used by spawned agents]

## Key Format (Auto-Detected)

[Keep the entity key table from the current SKILL.md]

## Detailed References

- `context/task-execution-pattern.md` - Mandatory agent workflow
- `context/entity-crud.md` - Create, update, delete patterns
- `context/workflow-and-status.md` - Status transitions, workflow profiles
- `context/notes-context-docs.md` - Notes, context, related docs
- `HOOKS.md` - Optional automation hooks
```

The existing `shark/context/` files (4 files) and `shark/HOOKS.md` are left unchanged.

The shark CLI reference sections (Quick Reference, Workflow, Manage, Notes, Context, Related Documents, JSON Output, Agent Workflow Pattern, Global Flags) are replaced by the dispatch loop. The dispatch loop is the primary content. The minimal shark CLI commands agents need are already covered in `context/entity-crud.md` and `context/workflow-and-status.md`.

### Step 4 — Add deprecation headers to 8 slash commands

**Files**: `~/.claude/commands/{run,feature,epic,task,prd,dispatch,develop,release}.md`

**Placement rule**: Insert the deprecation block immediately after the closing `---` of the YAML front matter, before the first heading. Leave all existing body content intact and functional.

**Header for `/run`** (specific — the orchestration skill reference is being removed):

```markdown
> **DEPRECATED (F05)** — The `/run` dispatch loop has moved into `~/.claude/skills/shark/SKILL.md`.
> The `orchestration` skill is no longer the primary dispatch mechanism. This command remains
> functional for one release; it will be removed in F06.
```

**Header for all other 7 commands** (`/feature`, `/epic`, `/task`, `/prd`, `/dispatch`, `/develop`, `/release`):

```markdown
> **DEPRECATED (F05)** — This slash command is a legacy entry point. Use the `shark` skill
> dispatch loop instead: `/run E01-F02` becomes `shark run E01-F02` in the new harness, or
> simply invoke the `shark` skill and follow the orchestration loop. This command remains
> functional for one release (F05); it will be removed in F06.
```

**Example result for `run.md`**:

```markdown
---
description: Drive an entity through its shark workflow - spawns agents, advances statuses, cascades into children
---

> **DEPRECATED (F05)** — The `/run` dispatch loop has moved into `~/.claude/skills/shark/SKILL.md`.
> The `orchestration` skill is no longer the primary dispatch mechanism. This command remains
> functional for one release; it will be removed in F06.

# /run — Dispatch Loop
...rest of existing content unchanged...
```

Verification:
```bash
head -20 ~/.claude/commands/run.md
head -20 ~/.claude/commands/feature.md
# etc. — deprecation block must appear before the first heading
```

### Step 5 — Verify AC-2

AC-2 tests that the engine resolves skills from `shark-data/`, not from `~/.claude/skills/`. This is statically verifiable without a live `/run` execution.

```bash
# 1. Verify quality skill exists in canonical sharkdata
ls internal/sharkdata/default_data/skills/quality/SKILL.md

# 2. Verify prompts reference quality via shark-data-relative paths
grep -r "include: skills/quality" internal/sharkdata/default_data/prompts/
# Expected: matches in ready_for_qa.md, in_qa.md, in_code_review.md,
#           ready_for_code_review.md, ready_for_test_planning.md,
#           _partials/_code_review_process.md, _partials/_qa_process.md

# 3. Confirm engine resolves from shark-data/, not from ~/.claude/skills/
grep -n "dataRoot\|detectIncludeRoot\|IncludeRoot" internal/templates/orchestrator_renderer.go

# 4. Move quality aside from harness
mv ~/.claude/skills/quality /tmp/quality-aside

# 5. Run shark validate (resolves all {{include:}} directives against shark-data/)
./bin/shark admin validate
# Must exit 0 with no errors about quality includes

# 6. Restore
mv /tmp/quality-aside ~/.claude/skills/quality
```

AC-2 is structurally guaranteed to pass because `IncludeResolver` (in `internal/templates/includes.go`) resolves against `<project>/shark-data/` only, never against `~/.claude/skills/`. The test confirms the architecture is intact.

---

## Updated Acceptance Criteria

| # | Criterion | How to verify |
|---|-----------|---------------|
| AC-1 | A fresh project with a fresh harness install can run shark workflows from `shark-data/` + only the `shark` skill in the harness | `ls ~/.claude/skills/` shows no deleted skills; `shark admin init` on a new project populates `shark-data/` |
| AC-2 | `shark admin validate` passes with `~/.claude/skills/quality/` moved aside | Steps in Step 5 above; engine resolves from `shark-data/skills/`, exit 0 |
| AC-3 | `/run` has deprecation header at top of file but remains fully functional | `head -10 ~/.claude/commands/run.md` shows the deprecation block; body unchanged |
| AC-4 | `~/.claude/agents/` contains exactly 6 files: `client.md`, `code-simplifier.md`, `cx-designer.md`, `devops.md`, `human-checkpoint.md`, `ux-designer.md` | `ls ~/.claude/agents/` after Step 2 |
| AC-5 | `~/.claude/skills/` contains only out-of-scope skills + `shark/` (with dispatch loop SKILL.md) | `ls ~/.claude/skills/` after Step 1; `~/.claude/skills/shark/SKILL.md` contains the dispatch loop from `orchestration/workflows/run.md` |

---

## Out-of-Scope Clarification

**The hard-cutover decision**: A previous F05 implementation attempt deleted 7 of the 8 in-scope slash commands outright (only `/run` was left present). This caused UAT rejection. The current machine state has all 8 commands restored with full functional bodies and no deprecation headers.

For this implementation, all 8 commands are present and will receive deprecation headers (not deletion). Deletion happens in F06. This matches the original feature spec intent: "Leave functional for one release; remove in F6."

**The assessment error on `shark/SKILL.md`**: The assessment (2026-06-22) stated `shark/SKILL.md` was "already correct as-is — a dispatch-loop rewrite from the previous F05 attempt." This is false. The current file is the original 246-line CLI reference. The rewrite did not survive the harness restore. Step 3 above is required work.

**AC-2 scope correction**: The original AC says "move quality aside; `/run` workflow that uses quality still works." This required a live `/run` execution. The corrected verification is `shark admin validate` — it tests the same architectural property (include resolution from `shark-data/`) without requiring a full workflow run. This is more reliable and deterministic.

---

*Last Updated*: 2026-06-22
