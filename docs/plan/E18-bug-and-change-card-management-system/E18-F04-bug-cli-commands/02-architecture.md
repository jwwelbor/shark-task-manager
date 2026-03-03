# Technical Architecture: Bug CLI Commands

**Feature Key**: E18-F04
**Complexity Tier**: STANDARD
**Status**: Technical Refinement

---

## Architecture Overview

This feature adds the `shark bug` command group to the CLI. It follows the established thin-wrapper pattern: each command handler parses arguments, calls a BugService method, and formats output. No business logic in command handlers.

### Key Architecture Decision: Single File

**Decision**: All bug commands live in a single file `internal/cli/commands/bug.go`.

**Rationale**: The `shark idea` command group follows this same pattern -- one file for the parent command and all subcommands. Bug has 10 subcommands, comparable to idea's 8. Splitting into multiple files adds navigation overhead without reducing complexity, since each handler is 15-25 lines.

**Alternative Considered**: Separate files per subcommand group (bug_crud.go, bug_lifecycle.go, bug_notes.go). Rejected because the total command code is under 400 lines and each handler is trivially small.

---

## File Changes

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/cli/commands/bug.go` | **NEW** | Parent command + 10 subcommands (~350-400 lines) |
| `internal/cli/service_accessors.go` | **MODIFY** | Add `GetBugService()` accessor function (~15 lines) |

No other files are modified. The BugService, BugRepository, and Bug model are delivered by F02. The NoteService and ContextService already exist and support arbitrary entity types.

---

## Command Tree

```
shark bug (parent command, GroupID: "advanced")
  |-- create   <title> [--severity=S] [--link=KEY] [--json]
  |-- get      <key> [--json] [--field NAME]
  |-- list     [--status=S] [--severity=S] [--link=KEY] [--json]
  |-- update   <key> [--title=...] [--severity=S] [--json]
  |-- delete   <key> [--force] [--json]
  |-- triage   <key> --severity=S [--assign=AGENT] [--json]
  |-- note
  |   |-- add  <key> --type=TYPE "<content>"
  |-- notes    <key> [--json]
  |-- context
      |-- set  <key> --field F --value V
      |-- get  <key> [--json]
      |-- clear <key> --field F
```

---

## Service Wiring

### GetBugService Accessor

Added to `internal/cli/service_accessors.go`, following the `GetIdeaService()` pattern:

```go
// GetBugService returns a BugService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
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

**Dependency Graph**:
```
GetBugService()
  |-- repository.NewBugRepository(db)
  |     |-- *repository.DB (shared global connection)
  |-- GetWorkflowService()
        |-- workflow.Service (shared global instance)
```

### Note and Context Service Reuse

Bug note and context commands reuse the existing `GetNoteService()` and `GetContextService()` accessors. These services accept entity_type as a parameter, so no new accessor functions are needed.

**Note commands** call:
- `NoteService.AddNote(ctx, entityType, entityKey, noteType, content)` with entityType = "bug"
- `NoteService.ListNotes(ctx, entityType, entityKey)` with entityType = "bug"

**Context commands** call:
- `ContextService.SetField(ctx, entityType, entityKey, field, value)` with entityType = "bug"
- `ContextService.GetContext(ctx, entityType, entityKey)` with entityType = "bug"
- `ContextService.ClearField(ctx, entityType, entityKey, field)` with entityType = "bug"

**Precondition**: F01 must extend NoteService and ContextService to recognize entity_type="bug". If the existing services already accept arbitrary entity types (they do -- they use a generic entity_notes table with entity_type column), then no F01 changes are needed for this particular integration.

---

## Command Handler Pattern

Every handler follows the three-step pattern. No exceptions.

### Step 1: Parse Arguments

Each command has a corresponding parse helper function (or inline parsing for simple cases):

```go
// For complex inputs (create):
func parseBugCreateInput(cmd *cobra.Command, args []string) services.CreateBugInput {
    title := args[0]
    severity, _ := cmd.Flags().GetString("severity")
    link, _ := cmd.Flags().GetString("link")
    return services.CreateBugInput{
        Title:    title,
        Severity: severity,
        LinkKey:  link,
    }
}

// For simple inputs (get, delete): inline in handler
```

### Step 2: Call Service

```go
svc := cli.GetBugService()
bug, err := svc.CreateBug(cmd.Context(), input)
if err != nil {
    return err
}
```

### Step 3: Format Output

```go
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(bug)
}
cli.Success(fmt.Sprintf("Created bug %s", bug.Key))
return nil
```

---

## Flag Registration

All flags registered in the `init()` function, following the idea.go pattern:

```go
var (
    bugSeverity string
    bugLink     string
    bugTitle    string
    bugAssign   string
    bugStatus   string
    bugForce    bool
    bugField    string
    bugValue    string
    bugNoteType string
)

func init() {
    cli.RootCmd.AddCommand(bugCmd)
    bugCmd.AddCommand(bugCreateCmd, bugGetCmd, bugListCmd, bugUpdateCmd,
                      bugDeleteCmd, bugTriageCmd, bugNoteCmd, bugNotesCmd,
                      bugContextCmd)
    bugNoteCmd.AddCommand(bugNoteAddCmd)
    bugContextCmd.AddCommand(bugContextSetCmd, bugContextGetCmd, bugContextClearCmd)

    // Create flags
    bugCreateCmd.Flags().StringVar(&bugSeverity, "severity", "", "Bug severity (critical, high, medium, low)")
    bugCreateCmd.Flags().StringVar(&bugLink, "link", "", "Link to epic/feature/task key")

    // List flags
    bugListCmd.Flags().StringVar(&bugStatus, "status", "", "Filter by status")
    bugListCmd.Flags().StringVar(&bugSeverity, "severity", "", "Filter by severity")
    bugListCmd.Flags().StringVar(&bugLink, "link", "", "Filter by linked entity")

    // Update flags
    bugUpdateCmd.Flags().StringVar(&bugTitle, "title", "", "New title")
    bugUpdateCmd.Flags().StringVar(&bugSeverity, "severity", "", "New severity")

    // Delete flags
    bugDeleteCmd.Flags().BoolVar(&bugForce, "force", false, "Skip confirmation")

    // Triage flags
    bugTriageCmd.Flags().StringVar(&bugSeverity, "severity", "", "Bug severity (required)")
    _ = bugTriageCmd.MarkFlagRequired("severity")
    bugTriageCmd.Flags().StringVar(&bugAssign, "assign", "", "Agent to assign")

    // Note flags
    bugNoteAddCmd.Flags().StringVar(&bugNoteType, "type", "", "Note type")
    _ = bugNoteAddCmd.MarkFlagRequired("type")

    // Context flags
    bugContextSetCmd.Flags().StringVar(&bugField, "field", "", "Context field name")
    bugContextSetCmd.Flags().StringVar(&bugValue, "value", "", "Context field value")
    _ = bugContextSetCmd.MarkFlagRequired("field")
    _ = bugContextSetCmd.MarkFlagRequired("value")
    bugContextClearCmd.Flags().StringVar(&bugField, "field", "", "Context field to clear")
    _ = bugContextClearCmd.MarkFlagRequired("field")
}
```

---

## Table Output Format

The list command renders a table with these columns:

```
KEY    TITLE                                    STATUS     SEVERITY  LINKED TO
B001   Login fails on Safari when 2FA enabled   reported   high      E07-F01
B002   Dashboard chart tooltip renders behind    triaged    medium    E07-F03
```

The get command renders a detail view (similar to `shark idea get`):

```
Bug B001
  Title:     Login fails on Safari when 2FA is enabled
  Status:    reported
  Severity:  high
  Linked To: E07-F01 (feature)
  Created:   2026-03-03T10:00:00Z
  Updated:   2026-03-03T10:00:00Z
```

---

## Error Handling

Commands return errors from service calls. The root command error handler translates to exit codes.

| Error Scenario | Service Error | Exit Code |
|---------------|---------------|-----------|
| Bug not found | NotFoundError | 1 |
| DB failure | wrapped DB error | 2 |
| Invalid state (triage already-triaged bug) | WorkflowTransitionError | 3 |
| Invalid severity value | ValidationError | 3 |
| Invalid field name (--field) | FieldNotFoundError | 4 |
| Linked entity not found | NotFoundError (from link validation) | 1 |

Commands do not catch and translate these errors individually. They return `err` from the service call and Cobra's error handler (or a shared error-to-exit-code mapper if one exists) handles the rest.

---

## Testing Strategy

### Test Architecture

CLI tests use mocked BugService. No real database.

### Mock Definition

```go
// internal/cli/commands/mock_bug_service_test.go
type MockBugService struct {
    CreateBugFunc func(ctx context.Context, input services.CreateBugInput) (*models.Bug, error)
    GetBugFunc    func(ctx context.Context, key string) (*models.Bug, error)
    ListBugsFunc  func(ctx context.Context, filters services.BugFilters) ([]*models.Bug, error)
    UpdateBugFunc func(ctx context.Context, key string, input services.UpdateBugInput) (*models.Bug, error)
    DeleteBugFunc func(ctx context.Context, key string) error
    TriageBugFunc func(ctx context.Context, key string, input services.TriageBugInput) (*models.Bug, error)
}
```

### Test File

`internal/cli/commands/bug_test.go` -- tests each subcommand with mocked service responses.

### Test Count Estimate

~32 test cases as specified in the PRD test strategy section.

---

## Cross-Feature Dependencies

| Dependency | Feature | Required Before F04 |
|-----------|---------|---------------------|
| `models.Bug` struct | F02 | Yes |
| `repository.BugRepository` | F02 | Yes |
| `services.BugService` with CreateBug, GetBug, ListBugs, UpdateBug, DeleteBug, TriageBug | F02 | Yes |
| `services.CreateBugInput`, `BugFilters`, `UpdateBugInput`, `TriageBugInput` DTOs | F02 | Yes |
| `workflow.LevelBug` in workflow engine | F01 | Yes |
| NoteService recognizing entity_type="bug" | F01 (if needed) | Soft -- may already work |
| ContextService recognizing entity_type="bug" | F01 (if needed) | Soft -- may already work |

F04 cannot begin implementation until F02 delivers the BugService interface. F04 can be developed with mock BugService interfaces if F02 is not complete, but integration testing requires F02.

---

## Implementation Phases

### Phase 1: Scaffolding (~30 min)
- Create `internal/cli/commands/bug.go` with parent command and empty subcommands
- Add `GetBugService()` to `internal/cli/service_accessors.go`
- Register parent command in init()
- Verify `shark bug --help` works

### Phase 2: CRUD Commands (~2 hours)
- Implement create, get, list, update, delete handlers
- Each handler: parse -> call service -> format output
- Add parse helper functions for complex inputs
- Test each with mock service

### Phase 3: Triage Command (~30 min)
- Implement triage handler
- Uses `--severity` (required) and `--assign` (optional)
- Calls BugService.TriageBug()

### Phase 4: Note and Context Commands (~1 hour)
- Implement note add, notes list handlers
- Implement context set, get, clear handlers
- Reuse existing NoteService and ContextService with entity_type="bug"

### Phase 5: Tests (~2 hours)
- Create `bug_test.go` with MockBugService
- Cover all ~32 test cases
- Verify JSON and table output formats

---

*Last Updated*: 2026-03-03
