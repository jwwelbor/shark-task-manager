# E15-F11: Service Layer Completion and CLI Integration - Research Report

**Feature**: E15-F11-service-layer-completion-and-cli-integration
**Date**: 2026-02-19
**Status at research time**: `in_research`

---

## Executive Summary

The feature is well-specified and its scope is accurate. All 13 fat-controller CLI command files remain unmodified - none have been partially migrated. Critically, more service infrastructure exists than the PRD acknowledged: `CriteriaService` already exists in `internal/services/criteria_service.go` with all major methods, and `GetCriteriaService()` is registered in `services_global.go`. The remaining gaps are: `TaskService.GetTaskHistory()`, analytics service methods (work session analytics by epic/feature), a `NoteService.GetNoteTimeline()` (for interleaved notes+history), `ResumeService.GetTaskResume()` (task-specific resume, the service only has epic/feature resume), and `NoteService.SearchNotesWithTimePeriod()`. The placeholder task breakdown (T-E15-F11-006 through T-E15-F11-025) is reasonable but could benefit from explicit ordering clarifications. All 30 test packages pass currently.

---

## Research Questions

1. For each of the 13 fat-controller CLI command files, what is the CURRENT state?
2. For the service methods listed as "New" in Dependencies section, do they exist yet?
3. What do the existing placeholder task files contain? Are they reasonable?
4. What is the current test state?
5. Do the 13 files still have fat controller patterns?

---

## Methodology

- Grepped `internal/cli/commands/` for `repository.New*`, `cli.GetDB`, `repoDb` patterns
- Read `internal/services/*.go` for existing service methods
- Read `internal/cli/services_global.go` for registered accessors
- Ran `make test` to confirm baseline
- Read each task file frontmatter for placeholder assessment

---

## Findings

### Finding 1: All 13 Files Are Fully Unmodified (Untouched Fat Controllers)

Every one of the 13 identified fat-controller CLI command files still contains direct repository access. None have been partially migrated.

**Evidence - fat-controller call counts per file:**

| File | Lines | Fat-Controller Calls | Status |
|------|-------|---------------------|--------|
| `task_criteria.go` | 460 | 16 | Untouched |
| `task_note.go` | 442 | 12 | Untouched |
| `task_resume.go` | 401 | 5 | Untouched |
| `task_context.go` | 397 | 6 | Untouched |
| `task_next_status.go` | 361 | 7 | Untouched |
| `history.go` | 277 | 4 | Untouched |
| `task_history.go` | 247 | 4 | Untouched |
| `analytics.go` | 238 | 5 | Untouched |
| `notes_search.go` | 230 | 6 | Untouched |
| `feature_criteria.go` | 195 | 5 | Untouched |
| `task_link.go` | 198 | 4 | Untouched |
| `view.go` | 104 | 4 | Untouched |
| `validate.go` | 98 | 4 | Untouched |

**Total fat-controller pattern calls to eliminate**: 66 across 13 files.

**Specific patterns found:**
- `cli.GetDB(cmd.Context())` - 13 files
- `repository.NewTaskRepository(...)` - 9 files
- `repository.NewEntityNoteRepository(...)` - 4 files
- `repository.NewTaskCriteriaRepository(...)` - 3 files
- `repository.NewTaskHistoryRepository(...)` - 3 files
- `repository.NewWorkSessionRepository(...)` - 2 files
- `repository.NewEpicRepository(...)` / `repository.NewFeatureRepository(...)` - 4 files each

**Verification command for AC-001:**
```bash
grep -rn "repository\.New\|cli\.GetDB\|repoDb" internal/cli/commands/ --include="*.go" \
  | grep -v "_test.go" | grep -v "mock_" \
  | grep -v "cloud.go\|init.go\|migrate_"
```
Currently returns 66 matches across 13 files. Target is 0 matches.

---

### Finding 2: Service Method Coverage - What Exists vs What Is Missing

#### ALREADY EXISTS (more than PRD acknowledged):

**CriteriaService** - `internal/services/criteria_service.go` (fully implemented):
- `CriteriaService.ImportCriteria()` - imports criteria from text
- `CriteriaService.ListCriteria()` - lists by task key (returns criteria with summary)
- `CriteriaService.CheckCriterion()` - marks criterion as complete
- `CriteriaService.FailCriterion()` - marks criterion as failed
- `CriteriaService.GetFeatureCriteria()` - aggregates criteria across a feature

**`GetCriteriaService(ctx)` registered in `services_global.go`** - Wire is complete.

**TaskService** (existing methods relevant to migrations):
- `TransitionStatus()` - handles task_next_status.go transition needs
- `GetNextStatus()` - handles preview mode for task_next_status.go
- `GetWorkSessions()` - partial analytics support
- `LinkDocument()` / `UnlinkRelationships()` - covers task_link.go document operations
- `AddDependency()` - covers dependency creation in task_link.go

**NoteService** (existing):
- `AddNote()` - covers task_note.go note add
- `ListNotes()` - covers task_note.go note list
- `SearchNotes()` - covers notes_search.go basic search

**ContextService** (existing):
- `GetContext()`, `SetContextField()`, `ClearContext()` - all needed by task_context.go
- Already handles `models.EntityTypeTask` - task context is supported

**ResumeService** (existing but task-specific variant missing):
- `GetEpicResume()` - exists
- `GetFeatureResume()` - exists
- **`GetTaskResume()` - MISSING** (task_resume.go needs task-level resume)

#### MISSING (must be added before migrations):

1. **`TaskService.GetTaskHistory()`** (critical - blocks `history.go` AND `task_history.go`):
   - `history.go` needs: `ListWithFilters(ctx, HistoryFilters{Agent, Since, Epic, Feature, OldStatus, NewStatus, Limit, Offset})`
   - `task_history.go` needs: `GetHistoryByTaskKey(ctx, taskKey)` with CSV/JSON format
   - Neither exists in TaskService; both go directly to `repository.NewTaskHistoryRepository()`

2. **Analytics service method** (blocks `analytics.go`):
   - Needs: `GetSessionAnalyticsByFeature(ctx, featureKey, agentType)` and `GetSessionAnalyticsByEpic(ctx, epicKey, agentType)`
   - `analytics.go` currently calls `sessionRepo.GetSessionAnalyticsByFeature()` and `GetSessionAnalyticsByEpic()` directly
   - `TaskService.GetWorkSessions()` handles single task; no epic/feature analytics method exists in any service

3. **`NoteService.GetNoteTimeline()` / `NoteService.SearchNotesWithTimePeriod()`** (blocks `task_note.go` timeline and `notes_search.go` with time filters):
   - `task_note.go` timeline interleaves task history + notes chronologically - needs a composite service method
   - `notes_search.go` calls `noteRepo.SearchWithTimePeriod(ctx, query, noteTypes, epicKey, featureKey, since, until)` - `NoteService.SearchNotes()` doesn't support time-period filtering

4. **`ResumeService.GetTaskResume()`** (blocks `task_resume.go`):
   - `task_resume.go` aggregates: task, context data, notes, work sessions, session stats, active session, dependencies, completion metadata
   - `ResumeService` only has `GetEpicResume()` and `GetFeatureResume()`

5. **`TaskService.CreateRelationship()` or equivalent** (partially blocks `task_link.go`):
   - `task_link.go` calls `relRepo.DetectCycle()` and `relRepo.Create()` directly
   - `TaskService.AddDependency()` only handles `depends_on` type
   - `task_link.go` needs all 7 relationship types: `depends_on`, `blocks`, `related_to`, `follows`, `spawned_from`, `duplicates`, `references`
   - A `CreateRelationship(ctx, taskKey, relType, targetKey)` method is needed in TaskService

**Summary of New Service Methods Needed:**

| Method | Priority | Blocks |
|--------|----------|--------|
| `TaskService.GetTaskHistory(ctx, filters)` | High | `history.go`, `task_history.go` |
| `ResumeService.GetTaskResume(ctx, taskKey)` | High | `task_resume.go` |
| `TaskService.GetAnalyticsByFeature/Epic()` | High | `analytics.go` |
| `NoteService.GetNoteTimeline(ctx, taskKey)` | High | `task_note.go` (timeline subcommand) |
| `NoteService.SearchNotesWithTimePeriod()` | Medium | `notes_search.go` (--since/--until flags) |
| `TaskService.CreateRelationship(ctx, taskKey, relType, targetKey)` | High | `task_link.go` |

---

### Finding 3: Assessment of Placeholder Tasks (T-E15-F11-006 through T-E15-F11-025)

**20 placeholder tasks were created. Assessment:**

| Task | Title | Assessment |
|------|-------|------------|
| T-E15-F11-006 | Implement CriteriaService with feature criteria CRUD operations | **ALREADY DONE** - CriteriaService exists in `criteria_service.go`. This task should be reassessed or repurposed to cover missing GetByID/Delete operations for task_criteria.go subcommands. |
| T-E15-F11-007 | Add TaskService.GetTaskHistory and ListTaskHistory service methods | **Correct** - These are missing and needed. Should clarify the `HistoryFilters` DTO structure needed. |
| T-E15-F11-008 | Add TaskService.GetTaskAnalytics service method | **Correct** - Analytics by epic/feature is missing. Should clarify it needs both `GetAnalyticsByFeature()` and `GetAnalyticsByEpic()`. |
| T-E15-F11-009 | Add NoteService.GetNoteTimeline and ResumeService.GetTaskResume service methods | **Correct** - Both missing. Good bundling since both are smaller additions. Also needs `NoteService.SearchNotesWithTimePeriod()`. |
| T-E15-F11-010 | Add global service accessor functions for CriteriaService in services_global.go | **ALREADY DONE** - `GetCriteriaService(ctx)` already exists in services_global.go. This task should be repurposed or closed. |
| T-E15-F11-011 | Migrate validate.go to thin wrapper | **Correct** - Still fat controller. But PRD notes this may be acceptable since `validation.RepositoryAdapter` already abstracts repositories. |
| T-E15-F11-012 | Migrate view.go to thin wrapper | **Correct** - Still fat controller. Uses `view.NewService()` internally, so the business logic is encapsulated - mostly a wiring change. |
| T-E15-F11-013 | Migrate task_link.go to thin wrapper | **Correct** - Needs `TaskService.CreateRelationship()` first. |
| T-E15-F11-014 | Migrate feature_criteria.go to thin wrapper | **Correct** - CriteriaService.GetFeatureCriteria() exists. Should be straightforward. |
| T-E15-F11-015 | Migrate history.go to thin wrapper | **Correct** - Needs TaskService.GetTaskHistory() first. |
| T-E15-F11-016 | Migrate task_history.go to thin wrapper | **Correct** - Needs TaskService.GetTaskHistory() first. |
| T-E15-F11-017 | Migrate analytics.go to thin wrapper | **Correct** - Needs GetAnalyticsByFeature/Epic() first. |
| T-E15-F11-018 | Migrate notes_search.go to thin wrapper | **Correct** - NoteService.SearchNotes() exists but needs time-period variant. |
| T-E15-F11-019 | Migrate task_next_status.go to thin wrapper | **Correct** - TransitionStatus() and GetNextStatus() exist in TaskService. Cascade is the tricky part. |
| T-E15-F11-020 | Migrate task_context.go to thin wrapper | **Correct** - ContextService has all needed methods (GetContext, SetContextField, ClearContext). Should be straightforward. |
| T-E15-F11-021 | Migrate task_resume.go to thin wrapper | **Correct** - But needs ResumeService.GetTaskResume() first. |
| T-E15-F11-022 | Migrate task_note.go to thin wrapper | **Correct** - NoteService.AddNote and ListNotes exist; timeline needs GetNoteTimeline(). |
| T-E15-F11-023 | Migrate task_criteria.go to thin wrapper | **Correct** - CriteriaService has most needed methods. May need GetByID wrapper for criterion lookup. |
| T-E15-F11-024 | Write service layer tests for CriteriaService, TaskService history and analytics methods | **Correct** - Needed. Should scope to the NEW methods added in 007-009. |
| T-E15-F11-025 | Verify all migrated CLI files pass lint, tests, and fat-controller compliance check | **Correct** - Final gate task. |

**Issues with placeholder breakdown:**
- T-E15-F11-006 and T-E15-F11-010 are already done - these should be marked complete or repurposed
- Missing task for `TaskService.CreateRelationship()` method (needed for task_link.go)
- T-E15-F11-009 should explicitly include `NoteService.SearchNotesWithTimePeriod()`

---

### Finding 4: Current Test State

**Result: 30/30 packages PASS, 0 failures.**

```
ok  github.com/jwwelbor/shark-task-manager/internal/cli              (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/cli/commands      (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/cli/scope         (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/config            (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/db                (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/dependency        (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/discovery         (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/fileops           (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/filepath          (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/formatters        (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/init              (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/keygen            (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/keys              (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/models            (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/parser            (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/pathresolver      (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/patterns          (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/reporting         (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/repository        0.824s
ok  github.com/jwwelbor/shark-task-manager/internal/services          (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/slug              (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/status            0.145s
ok  github.com/jwwelbor/shark-task-manager/internal/taskcreation      1.661s
ok  github.com/jwwelbor/shark-task-manager/internal/taskfile          (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/template          (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/templates         (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/utils             (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/validation        (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/view              (cached)
ok  github.com/jwwelbor/shark-task-manager/internal/workflow          (cached)
```

**CLI test database compliance baseline:**
- 17 CLI test files currently use `test.GetTestDB()`
- 11 of the 13 fat-controller files have NO test file at all (only `task_criteria_test.go` and `task_history_test.go` exist, both using in-memory DB not GetTestDB)
- The feature target of reducing from 17 to ≤3 integration test files is achievable

---

### Finding 5: Special Cases and Edge Cases Verified

**task_next_status.go** - Contains `triggerStatusCascade(ctx, repoDb, task.FeatureID)` private function that directly uses repoDb. This must move to `TaskService.TransitionStatus()` cascade triggering. `TransitionStatus()` exists in TaskService but must be verified to include cascade.

**view.go** - Uses `view.NewService(epicRepo, featureRepo, taskRepo)` internally already (the `internal/view` package). The 4 fat-controller calls are: getting repoDb, creating 3 repos to pass to view.NewService. This is the simplest migration - just need to expose view.NewService wiring via a helper or add `GetViewService()` accessor.

**validate.go** - Uses `validation.NewRepositoryAdapter(epicRepo, featureRepo, taskRepo)` which already abstracts the repos into an adapter. The PRD correctly notes this as an acceptable infrastructure pattern. The 4 fat-controller calls create repos to pass to this adapter - minimal business logic exposure.

**task_link.go** - Creates all 7 relationship types using `relRepo.Create()` directly. `TaskService.AddDependency()` only handles one type. A new `TaskService.CreateRelationship(ctx, sourceKey, relType, targetKey)` method is needed.

**task_context.go** - `ContextService` already handles `models.EntityTypeTask` context operations. The migration here should be straightforward: replace `cli.GetDB()` + `repository.NewTaskRepository()` with `cli.GetContextService(ctx)` and call the appropriate methods.

**notes_search.go** - Uses `noteRepo.SearchWithTimePeriod()` for `--since`/`--until` flags. `NoteService.SearchNotes()` doesn't have time-period support. This needs `NoteService.SearchNotesWithTimePeriod()` to be added.

---

## Constraints Identified

**Technical:**
- `task_note.go` timeline subcommand interleaves `task_history` + notes - requires `NoteService.GetNoteTimeline()` to return a composite struct aggregating both, OR the command can make two service calls and merge in presentation layer
- `task_criteria.go` calls `criteriaRepo.GetByID()` for criterion lookup - CriteriaService doesn't expose `GetByID()` publicly; needs either a `GetCriterionByID()` service method or criteria operations should accept criterion IDs directly (which they do - `CheckCriterion(criterionID)` exists)
- `analytics.go` needs `WorkSessionRepository.GetSessionAnalyticsByFeature/Epic()` - the TaskService currently only wraps `GetByTaskID()` operations. A new analytics method needs to work at epic/feature scope

**Architecture:**
- `TaskService.GetWorkSessions()` requires the service to be created with `NewTaskServiceWithRelationships()` (which wires the `sessionRepo`). The analytics migrations need `GetTaskServiceWithDeps()` or a new dedicated analytics accessor.

---

## Recommendations

### 1. Fix Stale Placeholder Tasks Before Starting

**Mark T-E15-F11-006 as completed** (CriteriaService already exists):
- `criteria_service.go` has: `ImportCriteria()`, `ListCriteria()`, `CheckCriterion()`, `FailCriterion()`, `GetFeatureCriteria()`
- Only gap is no `GetCriterionByID()` - but `CheckCriterion(criterionID)` and `FailCriterion(criterionID)` accept IDs directly

**Mark T-E15-F11-010 as completed** (GetCriteriaService already registered):
- `services_global.go` has `GetCriteriaService(ctx context.Context)` already wired

**Add a new task** for `TaskService.CreateRelationship()` for task_link.go multi-type relationship support.

**Update T-E15-F11-009** to include `NoteService.SearchNotesWithTimePeriod()`.

### 2. Recommended Implementation Order

**Phase 1 - Service Method Additions (prerequisite for all migrations)**:
1. (T-E15-F11-007) `TaskService.GetTaskHistory(ctx, filters HistoryFilters)` - blocks 2 migrations
2. (T-E15-F11-008) `TaskService.GetAnalyticsByFeature/Epic()` - blocks analytics.go
3. (NEW) `TaskService.CreateRelationship(ctx, sourceKey, relType, targetKey)` - blocks task_link.go
4. (T-E15-F11-009) `ResumeService.GetTaskResume()` AND `NoteService.GetNoteTimeline()` AND `NoteService.SearchNotesWithTimePeriod()` - blocks 2 migrations

**Phase 2 - Easy wins (no new service methods needed)**:
5. (T-E15-F11-020) Migrate `task_context.go` - ContextService is fully ready
6. (T-E15-F11-012) Migrate `view.go` - view.NewService() already encapsulates repos; just need accessor
7. (T-E15-F11-014) Migrate `feature_criteria.go` - CriteriaService.GetFeatureCriteria() ready
8. (T-E15-F11-011) Migrate `validate.go` - Document as accepted infrastructure pattern

**Phase 3 - Command migrations that depend on Phase 1 methods**:
9. (T-E15-F11-015) Migrate `history.go`
10. (T-E15-F11-016) Migrate `task_history.go`
11. (T-E15-F11-017) Migrate `analytics.go`
12. (T-E15-F11-018) Migrate `notes_search.go`
13. (T-E15-F11-013) Migrate `task_link.go`
14. (T-E15-F11-021) Migrate `task_resume.go`
15. (T-E15-F11-022) Migrate `task_note.go`
16. (T-E15-F11-023) Migrate `task_criteria.go`
17. (T-E15-F11-019) Migrate `task_next_status.go` - most complex (interactive prompts + cascade)

**Phase 4 - Tests and verification**:
18. (T-E15-F11-024) Write service layer tests for new service methods
19. (T-E15-F11-025) Final compliance verification

### 3. Decision Points for Developer

**validate.go**: The PRD suggests documenting as accepted infrastructure pattern (the `validation.RepositoryAdapter` abstraction). Recommend keeping as is and updating AC-001 to explicitly exclude it. The migration effort provides minimal architectural value.

**view.go**: Create a `GetViewService()` global accessor that wires `view.NewService(epicRepo, featureRepo, taskRepo)` from the `internal/view` package, similar to how other services are wired.

**task_link.go**: The relationship creation needs a new service method. Alternatively, accept task_link.go as analogous to infrastructure (relationship management is low-frequency, complex, and the service benefit is lower). Either approach is defensible.

---

## Risks and Unknowns

- **task_next_status.go cascade**: `triggerStatusCascade()` inside this file directly uses `repoDb`. Need to verify `TaskService.TransitionStatus()` invokes cascade before migrating.
- **task_note.go timeline**: Interleaving notes and task history chronologically requires the `GetNoteTimeline()` service method to aggregate from two repositories atomically. This is the most complex new service method to design.
- **Analytics scope**: `analytics.go` may need a new `AnalyticsService` or `GetTaskServiceWithAnalytics()` accessor since analytics at epic/feature level require the WorkSessionRepository to be wired into the service.
- **NoteService.SearchNotesWithTimePeriod()**: The repository method `noteRepo.SearchWithTimePeriod()` exists; the service method just needs to expose this with the same parameters.

---

## References

- `internal/cli/commands/` - All 13 fat-controller files
- `internal/services/criteria_service.go` - Already-implemented CriteriaService
- `internal/services/task_service.go` - 40+ methods, missing history/analytics
- `internal/services/note_service.go` - Missing timeline and time-period search
- `internal/services/resume_service.go` - Missing GetTaskResume()
- `internal/services/context_service.go` - Complete for task_context.go migration
- `internal/cli/services_global.go` - All accessors including GetCriteriaService()
- `internal/repository/task_history_repository.go` - HistoryFilters struct exists at repo layer
