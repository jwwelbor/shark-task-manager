# Feature E07-F19: File Path Flag Standardization

**Created:** 2026-01-02
**Epic:** E07 - Enhancements
**Status:** Draft (tasks created, ready for implementation)

---

## Overview

Standardize file path handling across all commands (epic, feature, task) using a single `--file` flag and full path storage in the database.

**Key Change:** Drop `custom_folder_path` column, use only `file_path` for all entities.

## Task Breakdown (7 tasks, ~12 hours)

### Phase 1: Foundation (Must run first)

**T-E07-F19-001: Database schema migration** (~1-2 hours)
- Priority: 9
- Agent: backend
- Dependencies: None
- Scope:
  - Add migration to drop `custom_folder_path` from epics and features tables
  - Verify existing `file_path` values preserved
  - Integration tests for migration safety
  - Handle column not existing (idempotent migration)

### Phase 2: Command Updates (Can run in parallel)

**T-E07-F19-002: Feature command flag standardization** (~2 hours)
- Priority: 8
- Agent: backend
- Dependencies: T-E07-F19-001
- Scope:
  - Add `--file`, `--filepath`, `--path` flag aliases to feature create/update
  - Fix update to use `file_path` (not `custom_folder_path`)
  - Move file on disk when path changes
  - Rollback file move if database update fails
  - Unit tests with mocked repository

**T-E07-F19-003: Epic command flag standardization** (~2 hours)
- Priority: 8
- Agent: backend
- Dependencies: T-E07-F19-001
- Scope:
  - Add same flag aliases to epic create/update
  - Update to use `file_path` instead of `custom_folder_path`
  - File moving logic with rollback
  - Unit tests with mocked repository

**T-E07-F19-004: Task command flag standardization** (~2 hours)
- Priority: 8
- Agent: backend
- Dependencies: T-E07-F19-001
- Scope:
  - Add flag support to task create (first time!)
  - Support custom file paths for tasks
  - Default path computation unchanged
  - Unit tests with mocked repository

### Phase 3: Model Cleanup

**T-E07-F19-005: Update data models** (~1 hour)
- Priority: 7
- Agent: backend
- Dependencies: T-E07-F19-002, T-E07-F19-003, T-E07-F19-004
- Scope:
  - Remove `CustomFolderPath` field from Epic model
  - Remove `CustomFolderPath` field from Feature model
  - Update JSON serialization
  - Update any remaining references in codebase

### Phase 4: Validation & Documentation

**T-E07-F19-006: Documentation updates** (~1 hour)
- Priority: 5
- Agent: backend
- Dependencies: None (can start anytime)
- Scope:
  - Update `docs/CLI_REFERENCE.md` with `--file` flag examples
  - Update `CLAUDE.md` to reflect simplified path handling
  - Mark `docs/MIGRATION_CUSTOM_PATHS.md` as deprecated
  - Add migration guide for users

**T-E07-F19-007: Integration and E2E testing** (~2 hours)
- Priority: 6
- Agent: testing
- Dependencies: T-E07-F19-005
- Scope:
  - E2E test: create epic/feature/task with `--file` flag
  - E2E test: update with `--file` (verify file moves)
  - E2E test: default paths still work (backward compat)
  - E2E test: all flag aliases work (`--file`, `--filepath`, `--path`)
  - Integration test: migration is idempotent

---

## Execution Strategy

### Recommended: Parallel Agent Execution

**Phase 1 (Sequential):**
```bash
# Must complete first
shark task start T-E07-F19-001
# Developer agent handles schema migration
```

**Phase 2 (Parallel):**
```bash
# After Phase 1 complete, run these 3 in parallel
shark task start T-E07-F19-002  # Developer Agent 1: Feature commands
shark task start T-E07-F19-003  # Developer Agent 2: Epic commands
shark task start T-E07-F19-004  # Developer Agent 3: Task commands
```

**Phase 3 (Sequential):**
```bash
# After all Phase 2 complete
shark task start T-E07-F19-005  # Developer Agent: Model cleanup
```

**Phase 4 (Parallel):**
```bash
# Documentation can start anytime
shark task start T-E07-F19-006  # Developer Agent: Docs

# Testing after Phase 3
shark task start T-E07-F19-007  # Testing Agent: E2E tests
```

### Alternative: Single Agent Sequential

```bash
# One developer agent works through tasks 001 → 007 in order
# Simpler coordination, but slower (no parallelization)
# Estimated: Same 12 hours, but wall-clock time is longer
```

---

## Dependency Graph

```
T-E07-F19-001 (Schema Migration)
    ├─→ T-E07-F19-002 (Feature Commands) ──┐
    ├─→ T-E07-F19-003 (Epic Commands)    ──┼─→ T-E07-F19-005 (Models) ─→ T-E07-F19-007 (Testing)
    └─→ T-E07-F19-004 (Task Commands)    ──┘

T-E07-F19-006 (Docs) - No dependencies, can start anytime
```

---

## Files to Modify

**Commands:**
- `internal/cli/commands/epic.go`
- `internal/cli/commands/feature.go`
- `internal/cli/commands/task.go`

**Models:**
- `internal/models/epic.go`
- `internal/models/feature.go`

**Database:**
- `internal/db/db.go` (add migration)

**Tests:**
- `internal/cli/commands/epic_test.go`
- `internal/cli/commands/feature_test.go`
- `internal/cli/commands/task_test.go`
- `internal/repository/epic_repository_test.go`
- `internal/repository/feature_repository_test.go`

**Documentation:**
- `docs/CLI_REFERENCE.md`
- `CLAUDE.md`
- `docs/MIGRATION_CUSTOM_PATHS.md` (deprecate)

**Total: ~14 files**

---

## Success Criteria

### Functionality
- ✅ `shark epic create "Title" --file="docs/custom/epic.md"` works
- ✅ `shark feature update E01-F01 --file="docs/new/location.md"` moves file + updates DB
- ✅ `shark task create "Title" --epic=E01 --feature=F01 --file="docs/custom/task.md"` works
- ✅ All three flag aliases work: `--file`, `--filepath`, `--path`
- ✅ Default paths work when flag omitted (95% case - backward compatible)
- ✅ File moves are atomic (rollback on DB error)

### Schema
- ✅ `custom_folder_path` column dropped from epics table
- ✅ `custom_folder_path` column dropped from features table
- ✅ Migration is idempotent (safe to run multiple times)
- ✅ Existing `file_path` values preserved

### Code Quality
- ✅ All unit tests pass (mocked repositories)
- ✅ All integration tests pass (real database with cleanup)
- ✅ All E2E tests pass
- ✅ No regressions in existing functionality

### Documentation
- ✅ CLI reference updated with examples
- ✅ CLAUDE.md reflects simplified architecture
- ✅ Migration guide for users upgrading

---

## Architecture Decision Reference

See `dev-artifacts/2026-01-02-feature-update-path-bug/ARCHITECTURE_DECISION.md` for:
- Product/UX/Architect perspectives
- Why hierarchical paths were rejected
- Full rationale for simplified approach
- Alternatives considered

**Key Insight:** Files don't move in typical workflow. Simple full path storage is faster, easier to understand, and matches actual usage patterns.

---

## Notes

### Test Data in Production DB

**Issue noted:** Test features (like F19) appearing in main database.

**Root cause (hypothesis):** Tests may be using main `shark-tasks.db` instead of test database.

**Investigation needed:**
- Audit test files to ensure `test.GetTestDB()` is used
- Check for hardcoded `shark-tasks.db` paths in tests
- Verify test cleanup is working

**Tracking:** Create separate task/idea for test database isolation audit.

---

## Supersedes

**T-E07-F18-001** - Original single task that grew into this feature after investigation revealed cross-cutting scope.
