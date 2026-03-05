# E18-F05: Change-Card CLI Commands -- Technical Architecture

**Feature Key**: E18-F05
**Complexity Tier**: STANDARD
**Date**: 2026-03-03

---

## 1. Architecture Overview

This feature adds the `shark change` Cobra command group to `internal/cli/commands/change.go`. Every subcommand follows the established **thin-wrapper pattern**: parse arguments, call `ChangeCardService` (F03), format output. Zero business logic resides in the command layer.

The architecture is intentionally parallel to the `shark bug` command group (F04) so that both new entity CLI surfaces are consistent in structure, naming conventions, and testing approach.

### System Context

```
User / AI Agent
       |
       v
  CLI Layer (change.go)          <-- THIS FEATURE
       |
       v
  ChangeCardService (F03)        <-- Exists (or will exist from F03)
       |
       v
  ChangeCardRepository (F03)
       |
       v
  SQLite / Turso (change_cards table, F01)
```

---

## 2. Key Architecture Decisions

### ADR-F05-001: Single File for All Change-Card Commands

**Decision**: All `shark change` subcommands live in a single file `internal/cli/commands/change.go`.

**Rationale**: The idea command group (`idea.go`, ~810 lines) demonstrates that a single file works well for entity-specific command groups with up to 10-12 subcommands. Splitting into multiple files (e.g., `change_create.go`, `change_list.go`) adds indirection without reducing complexity for thin wrappers of 15-25 lines each.

**Consequence**: File will be approximately 400-500 lines total (10 subcommands + helpers + init registration).

### ADR-F05-002: Service Accessor in service_accessors.go

**Decision**: Add `GetChangeCardService()` to `internal/cli/service_accessors.go`, following the `GetIdeaService()` pattern (simplest accessor -- no complex dependency wiring).

**Rationale**: The ChangeCardService constructor requires: `ChangeCardRepository`, `workflow.Service`, `EpicRepository` (link validation), `FeatureRepository` (link validation). This matches the moderate-complexity pattern used by `GetFeatureService()`. New instance created per call (stateless, lightweight).

**Consequence**: Each CLI command calls `cli.GetChangeCardService()` to get a fresh service instance.

### ADR-F05-003: Delegate Notes and Context to Existing Services

**Decision**: The `shark change note` and `shark change context` subcommands delegate directly to the existing `NoteService` and `ContextService` with `entity_type="change"`. No new service methods are needed.

**Rationale**: The NoteService and ContextService are entity-type-agnostic by design. They accept entity_type as a parameter and route internally. Adding "change" support requires only that the F01 schema migration allows `entity_type="change"` in the CHECK constraint, which is F01's responsibility.

**Consequence**: Note and context commands follow the same implementation pattern used by `shark task note` and `shark task context`.

### ADR-F05-004: Command Group Placement in "advanced" GroupID

**Decision**: Register `changeCmd` with `GroupID: "advanced"` to match the `ideaCmd` grouping pattern.

**Rationale**: Entity-specific command groups (idea, task, feature, epic) are all in the "advanced" group. Bug (F04) and change (F05) follow the same pattern. The unified commands (`shark get`, `shark status`) in F06 will provide the "workflow" group shortcuts.

**Consequence**: `shark --help` groups `change` under "Advanced" alongside `task`, `feature`, `epic`, `idea`, and `bug`.

### ADR-F05-005: View Command Extension for C### Keys

**Decision**: Extend the existing `shark view` command in `internal/cli/commands/view.go` (or its underlying `view.Service`) to recognize `C###` key format and resolve the file path from the `change_cards` table.

**Rationale**: The view command already handles E##, E##-F##, and E##-F##-### patterns. Adding C### is a small key-detection switch case. This is included in F05 scope rather than F06 because it operates on the change-card entity directly, not through unified dispatch.

**Consequence**: Requires adding a `ChangeCardRepository.GetByKey` call path in `view.Service` or adding a `ChangeCardFilePathResolver` to the view service's dependency set.

### ADR-F05-006: Confirmation Prompt Pattern for Delete

**Decision**: The `shark change delete` command follows the `shark idea delete` pattern: prompt for confirmation unless `--force` is provided. Use `fmt.Scanln` for interactive confirmation.

**Rationale**: Destructive operations should require confirmation. The idea command already implements this pattern (`confirmIdeaDelete`). Change-card delete follows the same UX.

**Consequence**: The delete command is not purely a thin wrapper -- it has a small confirmation conditional before the service call. This is accepted as a presentation concern, not business logic.

---

## 3. Command Structure

### Parent Command

```go
var changeCmd = &cobra.Command{
    Use:     "change",
    Short:   "Manage change-cards",
    GroupID: "advanced",
    Long:    `Change-card management operations for lightweight enhancement tracking.
...`,
}
```

### Subcommand Inventory

| Subcommand | Use Signature | Args | Flags | Service Method |
|------------|---------------|------|-------|----------------|
| `create` | `create "<title>" [--link=KEY]` | ExactArgs(1) | `--link` | `ChangeCardService.CreateChangeCard` |
| `get` | `get <key>` | ExactArgs(1) | `--json`, `--field` (inherited) | `ChangeCardService.GetChangeCard` |
| `list` | `list [--status=S] [--link=KEY]` | NoArgs | `--status`, `--link`, `--json` (inherited) | `ChangeCardService.ListChangeCards` |
| `update` | `update <key> [--title="..."]` | ExactArgs(1) | `--title` | `ChangeCardService.UpdateChangeCard` |
| `delete` | `delete <key> [--force]` | ExactArgs(1) | `--force` | `ChangeCardService.DeleteChangeCard` |
| `approve` | `approve <key>` | ExactArgs(1) | (none) | `ChangeCardService.ApproveChangeCard` |
| `note add` | `note add <key> --type=TYPE "content"` | MinArgs(1) | `--type` | `NoteService.AddNote` (entity_type="change") |
| `notes` | `notes <key>` | ExactArgs(1) | `--json` (inherited) | `NoteService.ListNotes` (entity_type="change") |
| `context set` | `context set <key> --field F --value V` | ExactArgs(1) | `--field`, `--value` | `ContextService.SetField` (entity_type="change") |
| `context get` | `context get <key>` | ExactArgs(1) | `--json` (inherited) | `ContextService.GetContext` (entity_type="change") |
| `context clear` | `context clear <key> [--field F]` | ExactArgs(1) | `--field` | `ContextService.ClearField` (entity_type="change") |

### Command Registration (init function)

```go
func init() {
    cli.RootCmd.AddCommand(changeCmd)

    // CRUD + lifecycle
    changeCmd.AddCommand(changeCreateCmd)
    changeCmd.AddCommand(changeGetCmd)
    changeCmd.AddCommand(changeListCmd)
    changeCmd.AddCommand(changeUpdateCmd)
    changeCmd.AddCommand(changeDeleteCmd)
    changeCmd.AddCommand(changeApproveCmd)

    // Notes
    changeCmd.AddCommand(changeNoteCmd)
    changeNoteCmd.AddCommand(changeNoteAddCmd)
    changeCmd.AddCommand(changeNotesCmd)

    // Context
    changeCmd.AddCommand(changeContextCmd)
    changeContextCmd.AddCommand(changeContextSetCmd)
    changeContextCmd.AddCommand(changeContextGetCmd)
    changeContextCmd.AddCommand(changeContextClearCmd)

    // Flags
    changeCreateCmd.Flags().StringVar(&changeLinkKey, "link", "", "Link to epic or feature (E## or E##-F##)")
    changeListCmd.Flags().StringVar(&changeStatusFilter, "status", "", "Filter by status (proposed, approved, in_progress, completed, declined)")
    changeListCmd.Flags().StringVar(&changeLinkFilter, "link", "", "Filter by linked entity key")
    changeUpdateCmd.Flags().StringVar(&changeTitle, "title", "", "New title")
    changeDeleteCmd.Flags().BoolVar(&changeForce, "force", false, "Skip confirmation prompt")
    changeNoteAddCmd.Flags().StringVar(&changeNoteType, "type", "comment", "Note type")
    changeContextSetCmd.Flags().StringVar(&changeCtxField, "field", "", "Context field name")
    changeContextSetCmd.Flags().StringVar(&changeCtxValue, "value", "", "Context field value")
    changeContextClearCmd.Flags().StringVar(&changeCtxField, "field", "", "Context field to clear")
}
```

---

## 4. Service Accessor Wiring

### GetChangeCardService

Add to `internal/cli/service_accessors.go`:

```go
// GetChangeCardService returns a ChangeCardService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetChangeCardService()
//	card, err := svc.CreateChangeCard(ctx, services.CreateChangeCardInput{Title: "..."})
func GetChangeCardService() *services.ChangeCardService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    changeRepo := repository.NewChangeCardRepository(db)
    workflowSvc := GetWorkflowService()
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    return services.NewChangeCardService(changeRepo, workflowSvc, epicRepo, featureRepo)
}
```

### Dependency Graph

```
GetChangeCardService()
├── repository.NewChangeCardRepository(db)
│   └── *repository.DB (global singleton)
├── GetWorkflowService()
│   └── .sharkconfig.json
├── repository.NewEpicRepository(db)      // for link validation passthrough
│   └── *repository.DB
└── repository.NewFeatureRepository(db)   // for link validation passthrough
    └── *repository.DB
```

---

## 5. Command Handler Patterns

### Create (Canonical Example)

```go
func runChangeCreate(cmd *cobra.Command, args []string) error {
    // Step 1: Parse
    title := args[0]
    input := services.CreateChangeCardInput{
        Title:           title,
        LinkedEntityKey: changeLinkKey,
    }

    // Step 2: Call service
    svc := cli.GetChangeCardService()
    card, err := svc.CreateChangeCard(cmd.Context(), input)
    if err != nil {
        return err
    }

    // Step 3: Format
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(card)
    }
    cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
    return nil
}
```

### List (Table Output Pattern)

```go
func runChangeList(cmd *cobra.Command, args []string) error {
    filters := services.ChangeCardFilters{
        Status:          changeStatusFilter,
        LinkedEntityKey: changeLinkFilter,
    }

    svc := cli.GetChangeCardService()
    cards, err := svc.ListChangeCards(cmd.Context(), filters)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(cards)
    }
    return printChangeCardList(cards)
}
```

### Approve (Domain-Specific Command)

```go
func runChangeApprove(cmd *cobra.Command, args []string) error {
    key := args[0]

    svc := cli.GetChangeCardService()
    card, err := svc.ApproveChangeCard(cmd.Context(), key)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(card)
    }
    cli.Success(fmt.Sprintf("Approved change-card %s", card.Key))
    return nil
}
```

### Delete (Confirmation Pattern)

```go
func runChangeDelete(cmd *cobra.Command, args []string) error {
    key := args[0]

    svc := cli.GetChangeCardService()
    card, err := svc.GetChangeCard(cmd.Context(), key)
    if err != nil {
        return err
    }

    if !changeForce {
        if !confirmChangeDelete(card) {
            return nil
        }
    }

    if err := svc.DeleteChangeCard(cmd.Context(), key); err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]string{"status": "deleted", "key": key})
    }
    cli.Success(fmt.Sprintf("Deleted change-card %s", key))
    return nil
}
```

### Note Add (NoteService Delegation)

```go
func runChangeNoteAdd(cmd *cobra.Command, args []string) error {
    key := args[0]
    content := strings.Join(args[1:], " ")

    // Resolve change-card ID
    svc := cli.GetChangeCardService()
    card, err := svc.GetChangeCard(cmd.Context(), key)
    if err != nil {
        return err
    }

    noteSvc, err := cli.GetNoteService(cmd.Context())
    if err != nil {
        return err
    }
    err = noteSvc.AddNote(cmd.Context(), "change", card.ID, changeNoteType, content)
    if err != nil {
        return err
    }

    cli.Success(fmt.Sprintf("Note added to change-card %s", key))
    return nil
}
```

---

## 6. Output Formatting

### Table Layout for List

```
Key    Title                        Status     Linked Entity  Created
C001   Add dark mode toggle         proposed   E07            2026-03-01
C002   Improve search performance   approved   E07-F03        2026-03-02
C003   Keyboard shortcuts           proposed   (none)         2026-03-03
```

Columns: `Key`, `Title`, `Status`, `Linked Entity`, `Created`

The `Linked Entity` column displays `(none)` when no link exists, or the linked entity key (e.g., `E07`, `E07-F01`).

### Detail Layout for Get

```
Change-Card: C001
Title: Add dark mode toggle
Status: proposed
Linked Entity: E07 (epic)
File: docs/changes/C001.md
Created: 2026-03-01 10:00:00
Updated: 2026-03-01 10:00:00
```

### JSON Output

All commands support `--json` via `cli.OutputJSON()`. JSON output returns the full `models.ChangeCard` struct serialized with `json.Marshal`. The `--field` flag (inherited from root) extracts a single field.

---

## 7. View Command Extension

### Change Required

In `internal/view/service.go` (or wherever the view service resolves file paths), add a case for the `C###` key format:

```go
// Key detection addition
case isChangeCardKey(key):
    card, err := s.changeRepo.GetByKey(ctx, key)
    if err != nil {
        return "", err
    }
    return card.FilePath, nil
```

### Key Detection Function

```go
// isChangeCardKey returns true for keys matching C### or C###-slug pattern
func isChangeCardKey(key string) bool {
    return regexp.MustCompile(`^(?i)C\d{3}(-[a-z0-9-]+)?$`).MatchString(key)
}
```

### View Service Dependency Addition

The `view.Service` constructor (and its wiring in `GetViewService()` in `services_global.go`) must be extended to accept a `ChangeCardRepository` (or a minimal `ChangeCardFilePathResolver` interface) so it can resolve C### key file paths.

```go
// In internal/cli/services_global.go, update GetViewService:
func GetViewService() *view.Service {
    // ... existing repos ...
    changeRepo := repository.NewChangeCardRepository(db)
    return view.NewService(epicRepo, featureRepo, taskRepo, changeRepo)
}
```

---

## 8. File Layout

### New Files

| File | Purpose | Lines (est.) |
|------|---------|-------------|
| `internal/cli/commands/change.go` | All `shark change` commands | ~450 |

### Modified Files

| File | Change | Lines Added (est.) |
|------|--------|--------------------|
| `internal/cli/service_accessors.go` | Add `GetChangeCardService()` | ~20 |
| `internal/cli/services_global.go` or `service_accessors.go` | Update `GetViewService()` to wire change repo | ~3 |
| `internal/view/service.go` | Add C### key detection and file path resolution | ~15 |

### Test Files

| File | Purpose |
|------|---------|
| `internal/cli/commands/change_test.go` | CLI tests with mocked ChangeCardService |

---

## 9. Testing Strategy

### Test Approach: Mocked Services Only

All CLI tests use mocked services. No real database in CLI tests (per testing architecture rules).

### Mock Interface

```go
type MockChangeCardService struct {
    CreateFunc  func(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error)
    GetFunc     func(ctx context.Context, key string) (*models.ChangeCard, error)
    ListFunc    func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error)
    UpdateFunc  func(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error)
    DeleteFunc  func(ctx context.Context, key string) error
    ApproveFunc func(ctx context.Context, key string) (*models.ChangeCard, error)
}
```

### Test Cases per Command

| Command | Test Cases |
|---------|-----------|
| `create` | Happy path, with `--link`, JSON output, service error |
| `get` | Happy path, not found (exit 1), `--json`, `--field` |
| `list` | No filters, `--status` filter, `--link` filter, empty results, JSON |
| `update` | Happy path, not found, `--json` |
| `delete` | With `--force`, not found |
| `approve` | Happy path, wrong status (exit 3), not found (exit 1), JSON |
| `note add` | Happy path, invalid type |
| `notes` | Happy path, JSON |
| `context set/get/clear` | Happy path each |

### Exit Code Coverage

| Exit Code | Trigger | Commands |
|-----------|---------|----------|
| 0 | Success | All |
| 1 | Not found | get, update, delete, approve |
| 2 | Service/DB error | All (generic) |
| 3 | Invalid state | approve (wrong status) |

---

## 10. Error Handling

Commands return errors to Cobra. The service layer provides typed errors that the command layer can inspect:

- `NotFoundError` -> CLI prints "change-card not found: C999", exit 1
- `WorkflowTransitionError` -> CLI prints status constraint message, exit 3
- Generic errors -> CLI prints message, exit 2

The error-to-exit-code mapping follows the existing pattern from task/feature/epic commands. For the initial implementation, returning the error to Cobra (which prints it and exits 1) is acceptable. Explicit exit code mapping can be added if the orchestrator requires specific codes.

---

## 11. Consistency with F04 (Bug CLI)

Both F04 and F05 must be structurally parallel:

| Aspect | F04 (Bug CLI) | F05 (Change-Card CLI) |
|--------|---------------|----------------------|
| File | `internal/cli/commands/bug.go` | `internal/cli/commands/change.go` |
| Parent command | `shark bug` | `shark change` |
| GroupID | `"advanced"` | `"advanced"` |
| Service accessor | `GetBugService()` | `GetChangeCardService()` |
| CRUD subcommands | create, get, list, update, delete | create, get, list, update, delete |
| Domain command | `triage` (severity + assign) | `approve` (proposed -> approved) |
| Note/context | Delegate to NoteService/ContextService | Delegate to NoteService/ContextService |
| View extension | B### key detection | C### key detection |
| Test file | `bug_test.go` | `change_test.go` |
| Test approach | Mocked BugService | Mocked ChangeCardService |

---

## 12. Dependencies and Ordering

### Must Be Complete Before F05 Development

1. **F01 (Database Schema)**: `change_cards` table, `entity_type="change"` CHECK constraint for notes
2. **F03 (ChangeCardService)**: All service methods this feature calls

### Can Be Developed In Parallel

- **F04 (Bug CLI)**: Independent command file, no shared code. F04 and F05 can be developed simultaneously.

### F05 Does NOT Depend On

- F06 (Unified CLI Integration) -- that comes after
- F07 (Dashboard and Analytics) -- that comes after

---

## 13. Security Considerations

- **Input sanitization**: Title input is passed to the service layer which performs validation. The CLI layer does not need separate sanitization.
- **No credentials handling**: Change-card commands do not handle auth tokens or secrets.
- **File path safety**: Markdown file creation is handled by the service layer using `fileops.EntityFileWriter` which prevents path traversal.

---

## 14. Performance Considerations

- **CLI overhead target**: < 50ms above service call time (per F05-NFR-002). Thin wrappers add negligible overhead (argument parsing + JSON marshaling).
- **No caching needed**: Each command is a single invocation. No in-memory caching required.
- **Service accessor creates new instance per call**: This is intentional and lightweight. No connection pooling concerns at the CLI layer.

---

*Last Updated*: 2026-03-03
