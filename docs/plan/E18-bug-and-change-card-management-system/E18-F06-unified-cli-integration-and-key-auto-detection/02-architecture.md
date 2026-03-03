# E18-F06 Technical Architecture: Unified CLI Integration and Key Auto-Detection

**Feature**: E18-F06
**Complexity Tier**: STANDARD
**Date**: 2026-03-03
**Status**: Draft

---

## 1. Architecture Overview

F06 extends the existing unified CLI command infrastructure (established by E17) to recognize `B###` (bug) and `C###` (change-card) key formats. The feature touches three architectural layers:

1. **Key Detection Layer** -- Pattern recognition and normalization (`internal/keys/`, `internal/cli/commands/helpers.go`, `internal/cli/scope/`)
2. **Command Dispatch Layer** -- Entity-type routing in unified commands (`internal/cli/commands/`)
3. **Service Dispatch Layer** -- Entity-type routing in cross-entity services (`internal/services/`)

The design follows the principle of **minimal, surgical extension** -- adding `case "bug"` and `case "change"` branches to existing dispatch points rather than introducing new abstractions. The PRD explicitly scopes out an entity-type registry pattern as out-of-scope for E18.

### Design Principles Applied

- **Appropriate**: Manual dispatch extension matches the existing codebase pattern and avoids over-engineering
- **Proven**: Follows the exact same switch/case pattern used for epic/feature/task dispatch throughout the codebase
- **Simple**: No new abstractions, no new packages, no new interfaces -- just new cases in existing switches

---

## 2. Key Detection Extension

### 2.1. `internal/keys/service.go` Changes

**New Constants:**

```go
const (
    EntityTypeBug    EntityType = "bug"
    EntityTypeChange EntityType = "change"
)
```

**New Regex Patterns:**

```go
// bugKeyPattern matches B followed by one or more digits: B001, B1, B42, B1000
bugKeyPattern = regexp.MustCompile(`^B(\d+)$`)

// changeKeyPattern matches C followed by one or more digits: C001, C1, C15
changeKeyPattern = regexp.MustCompile(`^C(\d+)$`)
```

**`Parse()` Method Extension:**

Add bug and change-card pattern matching **after** the epic pattern check (last, since they are the simplest patterns and cannot be confused with existing key formats). Bug and change-card keys do not support slugs in the initial implementation (per PRD scope).

```go
// Bug key: B followed by digits
if m := bugKeyPattern.FindStringSubmatch(upper); m != nil {
    result.EntityType = EntityTypeBug
    result.Normalized = fmt.Sprintf("B%s", m[1])
    return result
}

// Change-card key: C followed by digits
if m := changeKeyPattern.FindStringSubmatch(upper); m != nil {
    result.EntityType = EntityTypeChange
    result.Normalized = fmt.Sprintf("C%s", m[1])
    return result
}
```

The `ParsedKey` struct does not need new fields. Bug/change-card keys have no epic/feature/task number components. The `Normalized` field holds the canonical form (e.g., `B001`, `C015`).

**`Format()` Method Extension:**

```go
case EntityTypeBug:
    return parsed.Normalized  // "B001"

case EntityTypeChange:
    return parsed.Normalized  // "C001"
```

**No changes needed to:**
- `Normalize()` -- already delegates to `Parse()` then returns `Normalized`
- `IsValid()` -- already checks `EntityType != EntityTypeUnknown`
- `NormalizeTaskKey()` -- task-specific, no change needed

### 2.2. `internal/keys/validation.go` Changes

Add convenience functions following the existing pattern:

```go
func IsBugKey(s string) bool {
    return bugKeyPattern.MatchString(strings.ToUpper(s))
}

func IsChangeKey(s string) bool {
    return changeKeyPattern.MatchString(strings.ToUpper(s))
}
```

### 2.3. `internal/cli/commands/helpers.go` -- `DetectEntityType()` Extension

The `DetectEntityType()` function in helpers.go uses a chain of checks. Add bug and change-card detection **before** the slugged-key fallback section (since `B001` and `C001` are simple patterns that should match early):

```go
// After epic pattern checks and before the slugged-key fallback:

// Bug key: B followed by digits
if keys.IsBugKey(normalized) {
    return "bug"
}

// Change-card key: C followed by digits
if keys.IsChangeKey(normalized) {
    return "change"
}
```

### 2.4. `internal/cli/scope/interpreter.go` -- `parseGetArgsLogic()` Extension

The scope interpreter handles single-argument key parsing for `view`, `get`, and other commands. Add bug/change-card detection in the single-argument branch **after** epic checks:

```go
// In the single-argument branch, after IsEpicKey check:

// Bug key: B followed by digits
if keys.IsBugKey(normalized) {
    return "bug", normalized, nil
}

// Change-card key: C followed by digits
if keys.IsChangeKey(normalized) {
    return "change", normalized, nil
}
```

Also extend the `ScopeType` constants:

```go
const (
    ScopeBug    ScopeType = "bug"
    ScopeChange ScopeType = "change"
)
```

And the `ParseScope()` switch:

```go
case "bug":
    scopeType = ScopeBug
case "change":
    scopeType = ScopeChange
```

### 2.5. Key Detection Ordering

The ordering of pattern checks matters. The existing order (most specific first) is:

1. Task (T-E##-F##-### or E##-F##-###) -- most specific
2. Feature (E##-F## or F##)
3. Epic (E##)
4. Slugged variants

Bug and change-card keys (`B###`, `C###`) cannot conflict with any existing pattern because:
- They start with `B` or `C`, not `E`, `F`, or `T`
- There is no ambiguity with feature suffix keys (`F##`) because bugs start with `B` and change-cards with `C`

Therefore, bug/change detection can be placed **anywhere** in the chain. For clarity and consistency, place them after epic checks and before the slugged-key fallback.

---

## 3. Dispatch Point Inventory

This section enumerates every entity-type dispatch point that must be extended. This inventory serves as the implementation checklist required by F06-REQ-019.

### 3.1. Command Layer Dispatch Points

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 1 | `internal/cli/commands/get.go:48` | `runGet()` switch | epic, feature, task | Add case "bug" -> `runBugGet(cmd, []string{key})`, case "change" -> `runChangeGet(cmd, []string{key})` |
| 2 | `internal/cli/commands/delete_dispatch.go:37` | `runDelete()` switch | epic, feature, task | Add case "bug" -> `runBugDelete(cmd, args)`, case "change" -> `runChangeDelete(cmd, args)` |
| 3 | `internal/cli/commands/update_dispatch.go:85` | `runUpdate()` switch | epic, feature, task | Add case "bug" -> `runBugUpdate(cmd, args)`, case "change" -> `runChangeUpdate(cmd, args)` |
| 4 | `internal/cli/commands/status_group.go:175` | `dispatchTransition()` switch | epic, feature, task | Add case "bug" -> `cli.GetBugService().TransitionStatus(...)`, case "change" -> `cli.GetChangeCardService().TransitionStatus(...)` |
| 5 | `internal/cli/commands/status_group.go:189` | `dispatchNextStatus()` switch | epic, feature, task | Add case "bug" -> `cli.GetBugService().GetNextStatus(...)`, case "change" -> `cli.GetChangeCardService().GetNextStatus(...)` |
| 6 | `internal/cli/commands/context.go:235` | `toModelEntityType()` switch | epic, feature, task | Add case "bug" -> `models.EntityTypeBug`, case "change" -> `models.EntityTypeChange` |
| 7 | `internal/cli/commands/render_common.go:16` | `EntityDisplayOptions.EntityType` | "epic", "feature", "task" (string) | Document that "bug" and "change" are valid values. Add display header logic for "Bug: B001" and "Change Card: C001". |
| 8 | `internal/cli/commands/errors.go:129` | Error formatting switch | epic, feature, task | Add case "bug" and case "change" for entity-type-specific error messages |
| 9 | `internal/cli/commands/validators.go:93` | Validation switch | epic, feature, task | Add case "bug" and case "change" if applicable |

### 3.2. Service Layer Dispatch Points

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 10 | `internal/services/note_service.go:46` | `GetEntityDetails()` switch | EntityTypeTask, EntityTypeEpic, EntityTypeFeature | Add case EntityTypeBug and EntityTypeChange |
| 11 | `internal/services/note_service.go:89` | `resolveEntityID()` (called by AddNote, ListNotes) | epic, feature, task | Add case for bug and change |
| 12 | `internal/services/context_service.go` | `getContextJSON()` / `setContextJSON()` dispatch | epic, feature, task | Add case for bug and change (calls BugRepository/ChangeCardRepository) |
| 13 | `internal/services/resume_service.go` | `Resume()` dispatch | epic, feature, task | Add case for bug and change (should-have, Story 8) |

### 3.3. Model Layer Constants

| # | File | Location | Current Values | Action Required |
|---|------|----------|----------------|-----------------|
| 14 | `internal/models/entity_note.go:12-16` | EntityType constants | EntityTypeEpic, EntityTypeFeature, EntityTypeTask | Add EntityTypeBug, EntityTypeChange |
| 15 | `internal/models/entity_note.go:19-23` | ValidEntityTypes map | epic, feature, task | Add bug: true, change: true |

### 3.4. Key Detection Layer

| # | File | Location | Action Required |
|---|------|----------|-----------------|
| 16 | `internal/keys/service.go` | EntityType constants, Parse(), Format() | Add EntityTypeBug, EntityTypeChange; extend Parse() and Format() |
| 17 | `internal/keys/validation.go` | Validation helpers | Add IsBugKey(), IsChangeKey() |
| 18 | `internal/cli/commands/helpers.go:568` | DetectEntityType() | Add bug and change detection |
| 19 | `internal/cli/scope/interpreter.go:57` | ParseScope() switch | Add case "bug" and case "change" |
| 20 | `internal/cli/scope/interpreter.go:76` | parseGetArgsLogic() | Add bug and change key detection |

### 3.5. Search Extension

| # | File | Location | Action Required |
|---|------|----------|-----------------|
| 21 | `internal/cli/commands/search.go` | search command | Extend or create a new search pathway that includes bugs and change-cards. The current search.go only supports `--file` search on tasks. The PRD requires full-text `shark search "query" --type=bug` support. |
| 22 | `internal/repository/search_repository.go` (new or extended) | Full-text search | Add bug and change-card tables to FTS5 index or UNION query |

### 3.6. Service Accessor Functions

| # | File | Location | Action Required |
|---|------|----------|-----------------|
| 23 | `internal/cli/services_global.go` | Global accessors | Add `GetBugService()` and `GetChangeCardService()` if not already provided by F04/F05 |

### 3.7. Display Rendering

| # | File | Location | Action Required |
|---|------|----------|-----------------|
| 24 | `internal/cli/commands/render_common.go` | renderHeader() or equivalent | Add "Bug" and "Change Card" as entity type display names. Map "bug" -> "Bug", "change" -> "Change Card" for headers. |

### 3.8. View Command

| # | File | Location | Action Required |
|---|------|----------|-----------------|
| 25 | `internal/services/view_service.go` | `GetFilePath()` | Extend to resolve file paths for bug (`docs/bugs/B001.md`) and change-card (`docs/changes/C001.md`) entities |

### 3.9. CLI Output Helpers

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 26 | `internal/cli/output.go:11` | `GetRequiredSectionsForEntityType()` switch | epic, feature, task | Add case "bug" with bug-specific required sections (e.g., "Reproduction Steps", "Environment", "Fix Plan") and case "change" with change-card sections (e.g., "Change Description", "Impact Analysis", "Rollback Plan") |
| 27 | `internal/cli/output.go:77` | `FormatEntityCreationJSON()` next commands switch | epic, feature, task | Add case "bug" with next commands like `shark bug get B001`, `shark status advance B001`; case "change" with `shark change get C001`, `shark status advance C001` |

### 3.10. Configuration Commands (Pattern System)

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 28 | `internal/cli/commands/config.go:314` | `runGetFormat()` pattern format switch | epic, feature, task | N/A -- The pattern system handles filesystem discovery patterns for epics/features/tasks. Bugs and change-cards do not use filesystem discovery (they are stored in database with simple key-based lookup, not discovered from folder/file patterns). No extension needed. |
| 29 | `internal/cli/commands/config.go:949` | `findMatchingPatterns()` switch | epic, feature, task | N/A -- Returns matching filesystem discovery patterns for a test string. Bugs and change-cards are not discovered via filesystem patterns. No extension needed. |
| 30 | `internal/cli/commands/config.go:985` | `getPlaceholdersForType()` switch | epic, feature, task | N/A -- Returns generation format placeholders for filesystem path templates. Bugs and change-cards do not use filesystem generation formats. No extension needed. |

### 3.11. Unified List Command

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 31 | `internal/cli/commands/list.go:58` | `runList()` entity dispatch | epic, feature, task | N/A -- The unified `list` command uses positional arguments to traverse the epic->feature->task hierarchy (e.g., `shark list E07 F01`). Bugs and change-cards are flat entities outside this hierarchy. They are listed via dedicated `shark bug list` and `shark change list` commands (defined in F04/F05), not through the unified `shark list` dispatcher. No extension needed. |

### 3.12. Note Search Entity Type Reference

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 32 | `internal/cli/commands/notes_search.go:98` | `ValidEntityTypes` check and line 138 `EntityTypeTask` reference | epic, feature, task (via ValidEntityTypes map) | N/A -- The note search command validates entity types against `models.ValidEntityTypes` (dispatch point #15). Once that map is extended with "bug" and "change", `--entity-type bug` and `--entity-type change` will automatically work. The line 138 reference is a legacy backward-compatibility check for `EntityTypeTask` specifically; it does not need a bug/change case. The error message on line 99 should be updated from "must be one of: epic, feature, task" to include bug and change -- this is a string literal fix, not a dispatch point. |

### 3.13. Workflow Multi-Level Resolution

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 33 | `internal/config/workflow_multilevel.go:21` | `GetWorkflowForLevel()` switch | epic, feature, task | Add case "bug" returning bug-specific workflow config (with bug statuses like `triaged`, `in_fix`, `verified`), and case "change" returning change-card workflow config (with statuses like `proposed`, `approved`, `declined`). Requires `MultiLevelWorkflow` struct to gain `Bug *WorkflowConfig` and `Change *WorkflowConfig` fields. This is an F01 (schema/workflow) responsibility but must be verified during F06 integration. |

### 3.14. Pattern Matcher and Validator (Filesystem Discovery)

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 34 | `internal/patterns/matcher.go:223` | `GetAttemptedPatterns()` switch | epic, feature, task | N/A -- Pattern matcher handles filesystem discovery of entities from folder/file naming conventions. Bugs and change-cards are not discovered from filesystem patterns; they live in the database only. No extension needed. |
| 35 | `internal/patterns/validator.go:206` | `validateRequiredCaptureGroups()` switch | epic, feature, task | N/A -- Validates that regex patterns contain required capture groups for filesystem entity discovery. Not applicable to bugs/change-cards since they have no filesystem discovery patterns. No extension needed. |

### 3.15. Scan Report (Filesystem Scanning)

| # | File | Function/Location | Current Cases | Action Required |
|---|------|-------------------|---------------|-----------------|
| 36 | `internal/reporting/scan_report.go:93` | `AddMatched()` switch | epic, feature, task, related | N/A -- Tracks matched files during filesystem scans. Bugs and change-cards are not discovered via filesystem scanning. No extension needed. |
| 37 | `internal/reporting/scan_report.go:111` | `AddSkipped()` switch | epic, feature, task, related | N/A -- Tracks skipped files during filesystem scans. Same rationale as above. No extension needed. |

**Total dispatch points: 37** (25 original + 12 additional from tech check; scan_report.go contains 2 switches counted as dispatch points 36 and 37)

---

## 4. Unified Command Dispatch Pattern

Each unified command follows the same three-step extension pattern:

### 4.1. Get Command (`get.go`)

```go
func runGet(cmd *cobra.Command, args []string) error {
    command, key, err := ParseGetArgs(args)
    if err != nil {
        return err
    }

    switch command {
    case "epic":
        return runEpicGet(cmd, []string{key})
    case "feature":
        return runFeatureGet(cmd, []string{key})
    case "task":
        return runTaskGet(cmd, []string{key})
    case "bug":                                    // NEW
        return runBugGet(cmd, []string{key})       // NEW - delegates to F04 handler
    case "change":                                 // NEW
        return runChangeGet(cmd, []string{key})    // NEW - delegates to F05 handler
    default:
        return fmt.Errorf("cannot determine entity type from key: %s\n"+
            "Expected format: E## (epic), E##-F## (feature), E##-F##-### (task), B### (bug), C### (change card)", key)
    }
}
```

The `runBugGet` and `runChangeGet` functions are expected to be defined in F04 and F05 respectively (in `bug.go` and `change.go` command files). They follow the same thin-wrapper pattern: parse key, call `BugService.GetBug()` / `ChangeCardService.GetChangeCard()`, format output.

### 4.2. Status Commands (`status_group.go`)

The status commands use `ParseGetArgs(args[:1])` for entity type detection. With the key detection extension (Section 2), `ParseGetArgs("B001")` will return `("bug", "B001", nil)`. The dispatch functions need two new cases each:

```go
func dispatchTransition(ctx context.Context, entityType, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
    switch entityType {
    case "epic":    return cli.GetEpicService().TransitionStatus(ctx, key, targetStatus, opts)
    case "feature": return cli.GetFeatureService().TransitionStatus(ctx, key, targetStatus, opts)
    case "task":    return cli.GetTaskService().TransitionStatus(ctx, key, targetStatus, opts)
    case "bug":     return cli.GetBugService().TransitionStatus(ctx, key, targetStatus, opts)     // NEW
    case "change":  return cli.GetChangeCardService().TransitionStatus(ctx, key, targetStatus, opts) // NEW
    default:        return nil, fmt.Errorf("unsupported entity type: %s", entityType)
    }
}
```

**Prerequisite**: `BugService` and `ChangeCardService` must implement `TransitionStatus()` and `GetNextStatus()` methods with the same signature as the existing services. This is an F02/F03 responsibility and must be verified during integration.

### 4.3. Delete and Update Commands

Both follow the same pattern as get -- detect entity type via `DetectEntityType()`, switch on it, delegate to entity-specific handler from F04/F05.

### 4.4. Context Commands

The `toModelEntityType()` helper (context.go:234) maps string entity types to `models.EntityType`. Add:

```go
case "bug":
    return models.EntityTypeBug, nil
case "change":
    return models.EntityTypeChange, nil
```

The `ContextService` must be extended (dispatch point #12) to handle `EntityTypeBug` and `EntityTypeChange` by calling `BugRepository.GetByKey()` / `ChangeCardRepository.GetByKey()` for context get/set/clear operations.

### 4.5. View Command

The view command uses `scope.Interpreter.ParseScope()` which already delegates to `parseGetArgsLogic()`. With the key detection extension, `ParseScope(["B001"])` will return `ScopeBug`. The `ViewService.GetFilePath()` must handle the new scope types:

```go
case scope.ScopeBug:
    bug, err := s.bugRepo.GetByKey(ctx, parsedScope.Key)
    if err != nil {
        return "", err
    }
    return bug.FilePath, nil

case scope.ScopeChange:
    card, err := s.changeRepo.GetByKey(ctx, parsedScope.Key)
    if err != nil {
        return "", err
    }
    return card.FilePath, nil
```

### 4.6. Search Extension

The current `search.go` implements file-based search (`--file` flag) for tasks only. The PRD requires full-text search across entity types via `shark search "query" --type=bug|change`.

This requires a new or extended search pathway:

1. **Search Repository**: Add a `SearchAll(ctx, query, entityType)` method that queries across bugs and change-cards tables using SQLite LIKE or FTS5 (matching the existing search approach for tasks)
2. **Search Command**: Extend the search command to support a `--type` flag with valid values including `bug` and `change`. When `--type` is not specified, search all entity types.
3. **Search Results**: Return a unified result format with entity type, key, title, status, and severity (for bugs).

**Decision**: Since the current `search.go` is task-specific with `--file` search, the full-text search feature should be implemented as a new subcommand or mode. The architecture recommendation is to extend the existing `searchCmd` with a positional query argument (matching the PRD syntax `shark search "login" --type=bug`).

---

## 5. Service Layer Extension

### 5.1. NoteService Extension

`NoteService.GetEntityDetails()` (note_service.go:46) currently handles task, epic, and feature. Add:

```go
case models.EntityTypeBug:
    if s.bugRepo != nil {
        bug, err := s.bugRepo.GetByKey(ctx, entityKey)
        if err == nil {
            return &NoteEntityDetails{Key: bug.Key, Title: bug.Title}
        }
    }
case models.EntityTypeChange:
    if s.changeRepo != nil {
        card, err := s.changeRepo.GetByKey(ctx, entityKey)
        if err == nil {
            return &NoteEntityDetails{Key: card.Key, Title: card.Title}
        }
    }
```

**Dependency injection**: `NoteService` constructor gains optional `BugRepository` and `ChangeCardRepository` parameters. Following the existing pattern, these are passed as `nil` when not needed (graceful degradation).

### 5.2. ContextService Extension

Same pattern as NoteService -- add bug and change repository dependencies, add dispatch cases.

### 5.3. ResumeService Extension (Should-Have)

`ResumeService.Resume()` gains cases for bug and change-card entities. For bugs, the resume output includes severity and linked entity. For change-cards, includes linked entity.

### 5.4. Service Accessor Functions

If F04/F05 do not already provide `GetBugService()` and `GetChangeCardService()` in `services_global.go`, F06 must add them following the established pattern:

```go
func GetBugService() *services.BugService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    bugRepo := repository.NewBugRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewBugService(bugRepo, workflowSvc)
}
```

---

## 6. Model Layer Extension

### 6.1. EntityType Constants

In `internal/models/entity_note.go`:

```go
const (
    EntityTypeEpic    EntityType = "epic"
    EntityTypeFeature EntityType = "feature"
    EntityTypeTask    EntityType = "task"
    EntityTypeBug     EntityType = "bug"       // NEW
    EntityTypeChange  EntityType = "change"    // NEW
)

var ValidEntityTypes = map[EntityType]bool{
    EntityTypeEpic:    true,
    EntityTypeFeature: true,
    EntityTypeTask:    true,
    EntityTypeBug:     true,   // NEW
    EntityTypeChange:  true,   // NEW
}
```

---

## 7. Display Rendering

### 7.1. Entity Type Display Names

The `render_common.go` infrastructure uses `EntityType` strings for headers. Add a mapping for human-readable display:

```go
func displayEntityTypeName(entityType string) string {
    switch entityType {
    case "bug":
        return "Bug"
    case "change":
        return "Change Card"
    default:
        return strings.ToUpper(entityType[:1]) + entityType[1:]
    }
}
```

This produces headers like "Bug: B001" and "Change Card: C001", matching the existing "Epic: E07" pattern.

### 7.2. Bug Display Fields

When `shark get B001` renders a bug, the display includes:
- Title, Status, Severity, Linked Entity, Created/Updated timestamps

### 7.3. Change-Card Display Fields

When `shark get C001` renders a change-card, the display includes:
- Title, Status, Linked Entity, Created/Updated timestamps

---

## 8. Error Handling

### 8.1. Entity-Type-Aware Error Messages

The `handleServiceError()` function in helpers.go uses the `entityType` parameter for messages. No structural changes needed -- callers pass `"bug"` or `"change card"` as the entity type string.

The `errors.go` formatting switch gains:

```go
case "bug":
    return "Bug"
case "change":
    return "Change Card"  // Note: space, not hyphen
```

### 8.2. Not-Found Error Pattern

When `shark get B999` fails, the error chain is:
- Repository: `fmt.Errorf("bug not found: %s", key)` (or NotFoundError)
- Service: propagates
- Command: `handleServiceError(err, "bug", "B999")` produces "Bug not found: B999"

### 8.3. Invalid Key Format Error

When `shark get B` (no digits) is entered, the key detection returns "unknown". The existing error path in `ParseGetArgs` produces an error listing valid formats. Update the error message to include B### and C### examples:

```
invalid key format "B" - expected E## (epic), E##-F## (feature), E##-F##-### (task), B### (bug), or C### (change card)
```

---

## 9. Search Architecture

### 9.1. Current State

The `search.go` command currently implements `shark search --file="pattern"` for task file-based search only. There is no full-text query search.

### 9.2. Required Extension

The PRD requires `shark search "query" [--type=bug|change|epic|feature|task]`. This is a different search mode from the existing file search.

**Architecture Decision**: Extend the search command to support both modes:
- `shark search --file="pattern"` -- existing file search (task-only)
- `shark search "query" [--type=TYPE]` -- new full-text search (all entity types)

When a positional argument is provided (the query string), use full-text search mode. When `--file` is provided, use file search mode.

### 9.3. Full-Text Search Implementation

**Repository Layer**: Create a `SearchRepository` (or extend existing) with:

```go
type SearchResult struct {
    EntityType string `json:"entity_type"`
    Key        string `json:"key"`
    Title      string `json:"title"`
    Status     string `json:"status"`
    Severity   string `json:"severity,omitempty"` // bugs only
    Score      float64 `json:"score,omitempty"`
}

func (r *SearchRepository) Search(ctx context.Context, query string, entityType *string) ([]*SearchResult, error) {
    // UNION query across epics, features, tasks, bugs, change_cards
    // Filter by entity type if provided
    // Use LIKE '%query%' on title and description columns
    // Order by relevance (exact match first, then partial)
}
```

**Service Layer**: `TaskService.SearchByFile()` remains for file search. A new `SearchService` or method handles full-text search.

### 9.4. `--type` Flag Validation

Valid values: `epic`, `feature`, `task`, `bug`, `change`

Invalid values produce: `invalid type "foo": valid types are epic, feature, task, bug, change`

---

## 10. Dependency Summary

### 10.1. F06 Depends On (Must Be Complete Before F06)

| Dependency | What F06 Needs |
|------------|---------------|
| **F01** (Database Schema) | `bugs` and `change_cards` tables exist; workflow engine supports bug and change levels |
| **F02** (Bug Entity Core) | `BugService`, `BugRepository` interfaces and implementations exist; `Bug` model defined |
| **F03** (Change-Card Entity Core) | `ChangeCardService`, `ChangeCardRepository` interfaces and implementations exist; `ChangeCard` model defined |
| **F04** (Bug CLI Commands) | `runBugGet()`, `runBugDelete()`, `runBugUpdate()` handlers exist in command files |
| **F05** (Change-Card CLI Commands) | `runChangeGet()`, `runChangeDelete()`, `runChangeUpdate()` handlers exist in command files |
| **E17** (CLI Simplification) | Unified command infrastructure (`get`, `status`, `delete`, `update`, `context`, `notes`, `view`) exists |

### 10.2. Interface Contracts Expected from F02/F03

F06 expects `BugService` and `ChangeCardService` to implement:

```go
// Required for status commands
TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)

// Required for context commands
GetByKey(ctx context.Context, key string) (*models.Bug, error)  // or *models.ChangeCard

// Required for note service integration
// BugRepository and ChangeCardRepository must implement GetByKey
```

### 10.3. F06 Does NOT Depend On

| Feature | Why Not |
|---------|---------|
| F07 (Dashboard/Analytics) | F06 handles command dispatch; F07 handles reporting views |

---

## 11. Testing Strategy

### 11.1. Key Detection Tests

Add to `internal/keys/service_test.go`:
- `TestParse_BugKey` -- B001, b001, B1, B42, B1000
- `TestParse_ChangeKey` -- C001, c001, C1, C15
- `TestDetectEntityType_Bug` -- B001 returns EntityTypeBug
- `TestDetectEntityType_Change` -- C001 returns EntityTypeChange
- `BenchmarkDetectEntityType` -- extend existing benchmark with B### and C### cases

Add to `internal/cli/commands/helpers_test.go`:
- Extend `TestDetectEntityType` table with bug and change-card cases
- Extend `TestDetectEntityTypeEdgeCases` with edge cases like "B", "C", "B0", "C0"

### 11.2. Dispatch Point Tests

For each dispatch point in the inventory (Section 3), write a test that verifies:
1. `"bug"` case is handled (not falling through to default)
2. `"change"` case is handled
3. Correct service/handler is called
4. Error message contains correct entity type name

### 11.3. Integration Tests (End-to-End)

Test each unified command with B### and C### keys:
- `shark get B001` -- returns bug details
- `shark get C001` -- returns change-card details
- `shark status advance B001` -- advances bug
- `shark status set C001 declined` -- sets change-card status
- `shark delete B001 --force` -- deletes bug
- `shark update B001 --title="new"` -- updates bug
- `shark context set B001 --field env --value "Safari"` -- sets context
- `shark notes B001` -- lists notes
- `shark view B001` -- opens bug file
- `shark search "login" --type=bug` -- searches bugs

### 11.4. Error Path Tests

- `shark get B999` -- "bug not found: B999" (exit 1)
- `shark get C999` -- "change card not found: C999" (exit 1)
- `shark get B` -- invalid key format error with examples
- `shark search "query" --type=invalid` -- lists valid types including bug and change

---

## 12. Implementation Phases

### Phase 1: Key Detection (Foundation)
1. Add EntityTypeBug and EntityTypeChange to `keys/service.go`
2. Add IsBugKey/IsChangeKey to `keys/validation.go`
3. Extend DetectEntityType in `helpers.go`
4. Extend scope interpreter
5. Add EntityTypeBug/EntityTypeChange to `models/entity_note.go`
6. Write key detection tests

### Phase 2: Command Dispatch (Core)
7. Extend `get.go` dispatch
8. Extend `delete_dispatch.go` dispatch
9. Extend `update_dispatch.go` dispatch
10. Extend `status_group.go` dispatch (both functions)
11. Extend `context.go` toModelEntityType()
12. Extend error formatting
13. Update error messages to include B### and C### examples
14. Add service accessor functions if needed

### Phase 3: Service Layer (Integration)
15. Extend NoteService for bug/change entities
16. Extend ContextService for bug/change entities
17. Extend ViewService for bug/change file paths
18. Extend ResumeService for bug/change (should-have)

### Phase 4: Search Extension
19. Extend search command with full-text query mode
20. Add --type flag with bug/change values
21. Implement cross-entity search repository method

### Phase 5: Display and Polish
22. Add bug/change display rendering
23. Run dispatch inventory verification (grep + diff)
24. Write integration tests
25. Performance benchmark (key detection)

---

## 13. Risk Mitigation

### Risk 1: Missed Dispatch Points (MITIGATED)

**Mitigation**: The dispatch inventory in Section 3 is the single source of truth. A tech check on 2026-03-03 identified 12 additional dispatch points beyond the original 25 (total: 37). All 12 were analyzed and added to the inventory -- 4 require action (dispatch points #26, #27, #33, and note search error message update) and 8 are N/A (filesystem discovery/pattern/scan infrastructure not applicable to bug/change entities). Before marking F06 complete, re-run the dispatch inventory grep:

```bash
grep -rn 'case "epic"\|case "feature"\|case "task"\|EntityTypeEpic\|EntityTypeFeature\|EntityTypeTask' internal/
```

Diff against the inventory to catch any new dispatch points added by concurrent work.

### Risk 2: F02/F03 Interface Mismatch

**Mitigation**: Section 10.2 documents the exact interface contracts. Verify before implementation that BugService and ChangeCardService match.

### Risk 3: Search Scope Creep

**Mitigation**: The search extension (Phase 4) is the most complex part. If timeline pressure exists, implement a minimal LIKE-based search first and defer FTS5 optimization.

---

## 14. Architecture Decision Records

### ADR-F06-001: Manual Dispatch Over Entity Registry

**Decision**: Extend each dispatch point manually rather than implementing an entity-type registry.

**Rationale**: The PRD explicitly excludes the registry pattern as out-of-scope for E18. Manual dispatch is consistent with the existing codebase pattern. The 25 dispatch points are manageable with the inventory checklist approach.

**Consequences**: Adding future entity types (beyond bug/change) will require the same manual process. If a third new entity type is needed, the registry pattern should be reconsidered.

### ADR-F06-002: No Slug Support for Bug/Change Keys

**Decision**: Bug (B###) and change-card (C###) keys do not support slug suffixes in the initial implementation.

**Rationale**: The PRD explicitly scopes this out. Slug support adds complexity to key parsing and dual-key lookup without immediate value.

**Consequences**: Keys are numeric-only (B001, C015). Slug support can be added as a follow-on task.

### ADR-F06-003: Entity Type String "change" (Not "change-card")

**Decision**: Use `"change"` as the entity type string throughout the codebase.

**Rationale**: Matches the short, lowercase naming convention used for other types ("epic", "feature", "task", "bug"). Hyphens in entity type strings would require special handling in switch statements and JSON keys. The display name "Change Card" (with space) is used only in user-facing output via `displayEntityTypeName()`.

**Consequences**: `--type=change` in CLI, `"entity_type": "change"` in JSON output, `models.EntityTypeChange` in code.
