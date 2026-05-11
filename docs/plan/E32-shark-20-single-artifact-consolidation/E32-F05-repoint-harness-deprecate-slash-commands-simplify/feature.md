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

*Last Updated*: 2026-05-10
