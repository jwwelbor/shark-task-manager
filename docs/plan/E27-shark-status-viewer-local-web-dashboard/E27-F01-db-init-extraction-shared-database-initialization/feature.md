---
feature_key: E27-F01-db-init-extraction-shared-database-initialization
epic_key: E27
title: DB Init Extraction - Shared Database Initialization Package
description: Extract cloud-aware database initialization from internal/cli/db_init.go into a shared internal/dbinit package so that cmd/server can read .sharkconfig.json and support Turso, exactly as the CLI does today.
---

# DB Init Extraction - Shared Database Initialization Package

**Feature Key**: E27-F01-db-init-extraction-shared-database-initialization
**Execution Order**: 1 (prerequisite — all other E27 features depend on this)

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Description

This feature extracts the cloud-aware database initialization logic currently embedded in `internal/cli/db_init.go` into a new shared package `internal/dbinit/init.go`, exposing `Init(ctx, Options)` and `MustInit(ctx, Options)` functions callable from both `cmd/shark` and `cmd/server`. The existing `internal/cli/db_init.go` is then reduced to a thin delegate so all existing CLI callers (`GetDB`) continue working with no call-site changes. `cmd/server/main.go` is updated to call `dbinit.Init` instead of hardcoding `db.InitDB("shark-tasks.db")`, giving the server automatic `.sharkconfig.json` discovery and full Turso cloud support. This is a pure refactor with no new user-facing behavior — it is the prerequisite that unlocks Turso support for the viewer server and all subsequent E27 features.

---

## Dependencies

- No dependencies on other E27 features (this is the foundation)

## Integration Points

- `internal/cli/db_init.go` — collapses to delegate; all CLI callers unchanged
- `cmd/server/main.go` — switches from hardcoded SQLite path to `dbinit.Init`
- `cmd/server/services.go` — no changes needed; receives DB from updated main.go

---

*Last Updated*: 2026-04-11
