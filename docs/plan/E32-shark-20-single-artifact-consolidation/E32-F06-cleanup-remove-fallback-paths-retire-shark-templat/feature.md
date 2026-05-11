---
feature_key: E02-F06-cleanup-remove-fallback-paths-retire-shark-templat
epic_key: E02
title: Cleanup — remove fallback paths, retire shark-templates/
description: Drop one-release back-compat code — remove shark-templates/ fallback from the engine, remove deprecated slash commands, remove the legacy .sharkworkflow.json reader, update docs.
size: S
---

# Cleanup — remove fallback paths, retire shark-templates/

**Feature Key**: E02-F06

---

## Epic

- **Epic PRD**: [Epic](../epic.md)

---

## Goal

### Problem

F2–F5 ship with deliberate back-compat: the engine still falls back to `shark-templates/` if `shark-data/` is missing; deprecated slash commands still function with a header pointing to the new entry point; the legacy `.sharkworkflow.json` reader still loads. This back-compat exists so daily work isn't disrupted during the migration.

After one release window, the back-compat becomes dead weight that complicates the engine and slows iteration.

### Solution

Cleanup pass:

- Remove the `shark-templates/` resolution fallback from the engine.
- Remove deprecated slash commands.
- Remove the legacy `.sharkworkflow.json` reader.
- Update all docs to remove `shark-templates/` references.

### Impact

- Engine code is simpler — single resolution path.
- Slash commands list is leaner — only canonical entry points remain.
- `.sharkworkflow.json` is no longer recognized; deprecation error if encountered.

---

## Scope

### Engine fallback removal

- Remove `shark-templates/` resolution path from `internal/templates/` resolver (the F2/E5 fallback).
- Remove the legacy `.sharkworkflow.json` reader.
- Refuse to load with a clear deprecation error if either is encountered:
  > "shark-templates/ is no longer supported. Run `shark init` to lay down shark-data/, then migrate any local edits to shark-data/overrides/."

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

- Remove `shark-templates/` references from CLAUDE.md, README, internal docs.
- Update any onboarding instructions to reference `shark init` and `shark-data/` only.
- Search `~/.claude/hooks/` and `~/projects/shark-task-manager/scripts/` for hardcoded `shark-templates/` paths and update to `shark-data/`.

---

## Acceptance Criteria / Exit gate

1. `grep -r "shark-templates" ~/projects/shark-task-manager/cmd/ ~/projects/shark-task-manager/internal/` returns nothing.
2. `grep -r "shark-templates" ~/projects/shark-task-manager/` returns only historical CHANGELOG entries.
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
