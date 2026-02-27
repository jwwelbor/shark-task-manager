---
timestamp: 2026-02-27T18:45:00-06:00
branch: E17
last_commit: e89a8f0
status: all implementation complete, uncommitted changes ready to commit
---

# Continue: Commit E17 Work (CLI Restructure + Epic Display View)

## Current State

Branch `E17` has **uncommitted changes** from two bodies of work, both fully implemented and passing all quality gates (`make fmt && make lint && make test`).

### 1. Epic Display View Optimization (most recent work)

Reduces `shark get <epic>` from ~8 DB round-trips to 1 via a SQL view with JSON aggregation (critical for Turso cloud latency).

| File | Change |
|------|--------|
| `internal/db/db.go` | Added `migrateEpicDisplayDataView()` — SQL view with json_group_array |
| `internal/repository/epic_repository.go` | Added `EpicDisplayDataRaw` struct + `GetEpicDisplayDataRaw()` |
| `internal/services/epic_service.go` | Replaced multi-query `GetEpicDisplayData` with single-view query + JSON unmarshal |
| `internal/services/epic_service_test.go` | Added mock for `GetEpicDisplayDataRaw` |
| `internal/cli/commands/epic_helpers.go` | Simplified `buildEpicGetData` from 5 goroutines to 1 call |

Plan: `.claude/plans/functional-puzzling-micali.md`

### 2. E17 CLI Restructuring (prior work, same branch)

4-group CLI reorganization, flag removal from update commands, admin nesting, update dispatcher. See `git diff --stat HEAD` for full file list.

## What To Do

1. **Commit** — quality gates pass. Run `git diff --stat` and `git log --oneline -3`, then create one or two commits covering the changes.
2. **Check remaining E17 tasks**: `shark list E17` — see if any CLI restructuring tasks remain.
3. **Optional investigation**: `shark feature list E17` returns empty while `shark get E17 --json` correctly shows 12 features. Pre-existing bug, not introduced by view changes.

## Execution Optimization

- **Commit directly in main thread** — no need to read files, just `git status` + `git diff --stat` + commit.
- **If investigating the feature-list bug**: spawn an Explore agent to trace the `feature list` code path while main thread handles other work.
- **Avoid reading large files** — `epic_service.go` and `epic_helpers.go` are 1000+ lines. Use Explore agents for any investigation.
