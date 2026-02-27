---
timestamp: 2026-02-26T21:17:25-06:00
epic: E17
branch: E17
status: implementation complete, needs commit + doc updates + verification
---

# Continue: E17 CLI Help Reorganization — Commit & Finalize

## What Was Done

The CLI help reorganization (4-group structure) is **fully implemented and passing all quality gates** (`make fmt && make lint && make test` all green). All code changes are staged but **uncommitted**.

### Changes Summary

**New files created:**
- `internal/cli/commands/admin.go` — `shark admin` parent command (GroupID "advanced"), nests init/config/cloud/migrate/validate/workflow
- `internal/cli/commands/update_dispatch.go` — `shark update <KEY>` auto-detect dispatcher (GroupID "manage"), routes to task/feature/epic update

**Group restructure** (root.go): 6 groups → 4 groups
- `"workflow"` — next, start, done, block, unblock, status
- `"inspect"` — get, list, view, progress, search
- `"manage"` — create, update, delete, idea, context, notes, related-docs, analytics, history
- `"advanced"` — task, feature, epic, admin

**Flag removal** (status flags removed from update commands — use `shark status set` instead):
- task update: removed `--status`, `--force`, `--reason`, `--reason-doc`
- feature update: removed `--status`, `--force`
- epic update: removed `--status`, `--force`
- Removed unused `applyFeatureStatusUpdate()` from feature_helpers.go

**Files modified:** 38 files total (see `git status` for full list)

## What Needs To Be Done

### 1. Commit all changes
Run `git status` to review, then commit with a descriptive message covering the reorganization.

### 2. Update CLI reference docs
Three doc files show as modified from a prior commit but need review for alignment with the new structure:
- `docs/cli-reference/README.md` — update command categories to match new 4-group structure
- `docs/cli-reference/best-practices.md` — update any references to old group names or `shark init` (now `shark admin init`)
- `docs/cli-reference/error-messages.md` — check for stale command references

### 3. Verify `shark admin init` works end-to-end
The `init` command moved under `admin`. Run `shark admin init --help` and verify it matches expectations. Also verify `shark admin --help` lists all 6 subcommands.

### 4. Verify `shark update` works end-to-end
Test: `shark update E17 --title="test"` (epic), `shark update E17-F01 --title="test"` (feature). Confirm auto-detection routes correctly.

### 5. Verify `shark docs` alias
Test: `shark docs --help` should show related-docs help.

## Execution Optimization

**Parallel agents (Task tool):**

1. **Agent 1 (developer, background):** Commit the changes — run `git status`, `git diff`, `git log` to understand state, then create a commit with all 38 modified + 2 new files.

2. **Agent 2 (developer, background):** Update `docs/cli-reference/README.md` to reflect the new 4-group structure (Workflow/Inspect/Manage/Advanced), update references to `shark admin init` instead of `shark init`, and add `shark update` and `shark docs` to the command tables.

3. **Agent 3 (developer, background):** Update `docs/cli-reference/best-practices.md` and `docs/cli-reference/error-messages.md` — replace any `shark init` references with `shark admin init`, update command group names, add `shark update` examples where relevant.

4. **Main thread:** After agents complete, run final `make fmt && make lint && make test` quality gate, then commit doc updates separately.

## Key Files for Context

| Purpose | File |
|---------|------|
| Group definitions | `internal/cli/root.go` (lines 85-110) |
| Admin parent cmd | `internal/cli/commands/admin.go` |
| Update dispatcher | `internal/cli/commands/update_dispatch.go` |
| Flag removal (task) | `internal/cli/commands/task_helpers.go` (registerUpdateFlags) |
| Flag removal (feature) | `internal/cli/commands/feature.go` + `feature_helpers.go` |
| Flag removal (epic) | `internal/cli/commands/epic.go` + `epic_helpers.go` |
| Test fixes | `aliases_test.go`, `create_test.go`, `delete_dispatch_test.go`, `init_test.go`, `epic_update_test.go`, `feature_update_test.go`, `task_update_test.go` |
| CLI reference docs | `docs/cli-reference/README.md`, `best-practices.md`, `error-messages.md` |
