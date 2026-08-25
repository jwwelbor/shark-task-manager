---
feature_key: E32-F06
epic_key: E32
title: Cleanup — remove fallback paths, retire shark-templates/
description: Drop one-release back-compat code — remove shark-templates/ fallback from the engine, remove deprecated slash commands, remove the legacy .sharkworkflow.json reader, update docs.
size: S
---

# Cleanup — remove fallback paths, retire shark-templates/

**Feature Key**: E32-F06

---

## Epic

- **Epic PRD**: [Epic](../epic.md)

---

## Goal

### Problem

F2–F5 shipped deliberate compatibility paths. The renderer cutover is now already canonical-only: a legacy `shark-templates/` tree must be ignored rather than become an error source. The remaining legacy concern is implicit `.sharkworkflow.json` loading; deprecated slash commands required a release-window audit before retirement.

After one release window, the back-compat becomes dead weight that complicates the engine and slows iteration.

### Solution

Cleanup pass:

- Preserve canonical prompt resolution and ensure a `shark-templates/` tree cannot affect it.
- Remove deprecated slash commands.
- Remove the legacy `.sharkworkflow.json` reader.
- Update current-facing docs; preserve historical plans and review evidence.

### Impact

- Engine code is simpler — single resolution path.
- Slash commands list is leaner — only canonical entry points remain.
- `.sharkworkflow.json` is no longer recognized; deprecation error if encountered.

---

## Scope

### Engine fallback removal

- Retain the already-shipped canonical-only renderer behavior; a legacy tree is ignored, not loaded or rejected.
- Remove the legacy `.sharkworkflow.json` reader.
- Refuse a root or explicit legacy `.sharkworkflow.json` with clear migration guidance. A legacy prompt tree remains non-operative.

### Slash command removal

Delete from `~/.claude/commands/`:
- `/run`
- `/feature`
- `/epic`
- `/task`
- `/prd`
- `/dispatch`
- `/develop`
- `/release`

(All carried deprecation headers in F5; F6 deletes them.)

### Documentation cleanup

- Remove retired-path claims from current operator guidance; preserve historical references in plans, changelogs, and review evidence.
- Update any onboarding instructions to reference `shark init` and `shark-data/` only.
- Search `~/.claude/hooks/` and `~/projects/shark-task-manager/scripts/` for hardcoded `shark-templates/` paths and update to `shark-data/`.

---

## Acceptance Criteria / Exit gate

1. `grep -r "shark-templates" ~/projects/shark-task-manager/cmd/ ~/projects/shark-task-manager/internal/` returns nothing.
2. Current operator documentation contains no retired path as supported behavior; historical records retain accurate migration history.
3. Old `.sharkworkflow.json` files refused with deprecation error message.
4. Deprecated slash commands no longer present in `~/.claude/commands/`.
5. CLAUDE.md and README contain no `shark-templates/` references.

---

## Out of Scope (for F6)

- Anything new — F6 is strictly deletion + doc cleanup.
- Override drift / golden-output validation / `shark dev` mode (follow-up epic).

---

## Dependencies

- **Blocks**: none (final feature).
- **Blocked by**: F5 + one release window (so the back-compat actually served its purpose).

---

## Risks

- **Premature cleanup.** Don't merge F6 until at least one full release of F5 has been in daily use. The back-compat is the safety net.
- **Hidden hardcoded paths.** Bash hooks, scripts, third-party tools may reference `shark-templates/`. The audit happens in F4 but should be re-verified in F6 before deletion.

---

*Last Updated*: 2026-05-10
