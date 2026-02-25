# Exploratory Findings: T-E15-F11-025 - Holistic Verification

**Date:** 2026-02-19
**Task:** T-E15-F11-025
**QA Agent:** claude-sonnet-4-6
**Session Duration:** ~15 minutes
**Charter:** Explore all E15-F11 CLI migrations to discover architecture compliance issues

---

## Findings

### Finding 1 (Critical): GetTaskServiceWithHistory() is Unused

**Description:** `GetTaskServiceWithHistory()` was created in `services_global.go` expressly to support the history command migration, but `history.go` never calls it. The accessor is fully wired (includes historyRepo, epicRepo, featureRepo, keygen, validator, renderer, creatorSvc), yet the command still creates its own repositories.

**File:** `internal/cli/commands/history.go`
**Evidence:** The accessor at line 149 of `services_global.go` exists and is ready to use.

### Finding 2 (Positive): Incomplete migration is still compilable

The project compiles cleanly despite the violation. This is because Go does not enforce architectural patterns at compile time. The fat controller pattern is a code quality / architecture concern, not a language constraint.

### Finding 3 (Positive): All 8 other migrated files are clean

Commands task_history.go, analytics.go, notes_search.go, task_next_status.go, task_context.go, task_resume.go, task_note.go, and task_criteria.go all correctly use service layer accessors and do not import repositories directly.

### Finding 4 (Positive): Service accessor coverage is complete

The `services_global.go` file provides all the accessors needed for a complete migration:
- `GetTaskService()` - for standard task operations
- `GetTaskServiceWithHistory()` - for history queries (wired but unused)
- `GetTaskServiceWithDeps()` - for dependency/relationship operations
- `GetCriteriaService()` - for criteria management
- `GetNoteService()` - for note operations
- `GetContextService()` - for context retrieval
- `GetResumeService()` - for resume context

### Finding 5 (Observation): history.go has complex display logic that should move to service

The `runHistory` function (lines 174-208) builds display records by looping over history entries and calling `taskRepo.GetByID()` for each one. This business logic - enriching history records with task information - belongs in the service layer, not the command. A `GetProjectHistory` service method should return enriched results ready for display.

---

## Summary

The migration is 8/9 complete. One file (`history.go`) was either missed or its migration was incomplete. Given that `GetTaskServiceWithHistory()` already exists and is wired correctly, the fix is straightforward - it requires:

1. Adding a `GetProjectHistory(ctx, filters)` method to `TaskService` that queries via `historyRepo.ListWithFilters()` and enriches with task keys
2. Updating `history.go` to call `cli.GetTaskServiceWithHistory()` and delegate to the new service method
3. Removing direct repository imports and instantiation from `history.go`
