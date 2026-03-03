# Feature PRD: Bug CLI Commands

**Feature Key**: E18-F04
**Epic**: [E18 - Bug and Change-Card Management System](../epic.md)
**Complexity Tier**: STANDARD
**Status**: In Refinement (BA)

---

## Goal

Expose all BugService (F02) operations through the Shark CLI so that developers, QA engineers, and AI agents can create, triage, query, and manage bugs using terminal commands. Every subcommand follows the thin-wrapper pattern established in E15/E17: parse arguments, call a service method, format output.

### Business Context

The BugService provides all bug business logic (create, triage, status transitions, link validation, severity management). Without CLI commands, that logic is unreachable. CLI is the primary -- and for many users the only -- entry point for Shark operations. This feature bridges the service layer to the user.

### Value Delivered

- Developers report bugs in under 30 seconds without leaving the terminal (REQ-F-001 acceptance)
- QA engineers triage bugs with a single compound command (REQ-F-005)
- AI agents and CI/CD pipelines consume bug data via `--json` output (REQ-NF-005)
- All commands are consistent with existing `shark task`, `shark epic`, `shark feature`, `shark idea` patterns (REQ-NF-005)

---

## Personas (Reference Only)

This feature serves the same three personas defined at the epic level. No feature-specific persona refinements are needed.

- **Developer**: Creates bugs, views assigned bugs, advances bug status during fix
- **QA Engineer**: Triages bugs (severity + assignment), verifies fixes, queries bugs by severity/status
- **CI/CD Pipeline Agent**: Creates bugs programmatically with `--json` output, no interactive prompts

See [E18 Personas](../personas.md) for full profiles.

---

## Command Inventory

The `shark bug` command group contains 10 subcommands organized into three categories.

### CRUD Commands

| Command | Args/Flags | Service Method | Epic Req |
|---------|-----------|---------------|----------|
| `shark bug create "<title>" [--severity=S] [--link=KEY]` | title (positional), severity (optional, default: medium), link (optional) | `BugService.CreateBug()` | REQ-F-001, REQ-F-002, REQ-F-003, REQ-F-006 |
| `shark bug get <key> [--json] [--field NAME]` | key (positional), json (flag), field (flag) | `BugService.GetBug()` | REQ-F-006 |
| `shark bug list [--status=S] [--severity=S] [--link=KEY] [--json]` | status, severity, link (all optional filters) | `BugService.ListBugs()` | REQ-F-006, REQ-F-018 |
| `shark bug update <key> [--title="..."] [--severity=S]` | key (positional), title, severity (optional) | `BugService.UpdateBug()` | REQ-F-006 |
| `shark bug delete <key> [--force]` | key (positional), force (flag, skips confirmation) | `BugService.DeleteBug()` | REQ-F-006 |

### Lifecycle Commands

| Command | Args/Flags | Service Method | Epic Req |
|---------|-----------|---------------|----------|
| `shark bug triage <key> --severity=S [--assign=AGENT]` | key (positional), severity (required), assign (optional) | `BugService.TriageBug()` | REQ-F-005 |

### Note and Context Commands

| Command | Args/Flags | Service Method | Epic Req |
|---------|-----------|---------------|----------|
| `shark bug note add <key> --type=TYPE "<content>"` | key (positional), type (required), content (positional) | `NoteService.AddNote()` with entity_type="bug" | REQ-F-015 |
| `shark bug notes <key>` | key (positional) | `NoteService.ListNotes()` with entity_type="bug" | REQ-F-015 |
| `shark bug context set <key> --field F --value V` | key, field, value | `ContextService.SetContext()` with entity_type="bug" | REQ-F-015 |
| `shark bug context get <key>` | key (positional) | `ContextService.GetContext()` with entity_type="bug" | REQ-F-015 |

### Integration with Existing Commands (Not in F04 Scope)

These commands are extended in F06, not F04:
- `shark get B001` -- auto-detection dispatch (F06)
- `shark status advance B001` -- unified status command (F06)
- `shark view B001` -- markdown viewer extension (F06)
- `shark search "login" --type=bug` -- search integration (F06)

---

## Architecture

### File Layout

| File | Purpose |
|------|---------|
| `internal/cli/commands/bug.go` | All `shark bug` subcommands. Parent command + 10 subcommands. |
| `internal/cli/services_global.go` | Add `GetBugService()` accessor function |

### Command Structure

```
shark bug (parent)
  |-- create
  |-- get
  |-- list
  |-- update
  |-- delete
  |-- triage
  |-- note
  |   |-- add
  |-- notes
  |-- context
      |-- set
      |-- get
      |-- clear
```

### Pattern Compliance

Every handler follows the three-step pattern:

```go
func runBugCreate(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    input := parseBugCreateInput(cmd, args)

    // Step 2: Call service
    svc := cli.GetBugService()
    bug, err := svc.CreateBug(cmd.Context(), input)
    if err != nil {
        return err
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bug)
    }
    cli.Success(fmt.Sprintf("Created bug %s", bug.Key))
    return nil
}
```

Each subcommand is 15-25 lines. No business logic in command handlers.

### Service Accessor

```go
// internal/cli/services_global.go
func GetBugService() *services.BugService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    bugRepo := repository.NewBugRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewBugService(bugRepo, workflowSvc, nil)
}
```

---

## Dependencies

| Dependency | Type | Feature |
|-----------|------|---------|
| BugService | Hard | F02 (Bug Entity Core) |
| BugRepository | Hard | F02 |
| Bug model | Hard | F02 |
| NoteService (existing) | Soft | Existing codebase -- needs `entity_type="bug"` support via F01 |
| ContextService (existing) | Soft | Existing codebase -- needs `entity_type="bug"` support via F01 |
| Workflow engine with `LevelBug` | Hard | F01 (Database Schema and Workflow Engine Extension) |

F04 cannot start until F02 delivers BugService with all CRUD + triage methods, and F01 delivers the workflow engine extension with `LevelBug`.

---

## Output Formats

### JSON Output (--json)

All commands support `--json` for machine-readable output. JSON structure mirrors the Bug model:

```json
{
  "key": "B001",
  "title": "Login fails on Safari when 2FA is enabled",
  "status": "reported",
  "severity": "high",
  "slug": "login-fails-on-safari-when-2fa-is-enabled",
  "linked_entity_type": "feature",
  "linked_entity_key": "E07-F01",
  "file_path": "docs/bugs/B001.md",
  "created_at": "2026-03-03T10:00:00Z",
  "updated_at": "2026-03-03T10:00:00Z"
}
```

### Table Output (Default)

List command renders a table:

```
KEY    TITLE                                    STATUS     SEVERITY  LINKED TO
B001   Login fails on Safari when 2FA enabled   reported   high      E07-F01
B002   Dashboard chart tooltip renders behind    triaged    medium    E07-F03
B003   API returns 500 on empty payload          in_fix     critical  E07-F02-003
```

### Field Extraction (--field)

`shark bug get B001 --field severity` returns just: `high`

---

## Testing Strategy

### Approach

CLI tests use mocked BugService. No real database in CLI tests.

### Test Scope

| Test Category | Count | What is Tested |
|--------------|-------|----------------|
| Create command | 4 | Success, with severity, with link, JSON output |
| Get command | 3 | Success, not found error, field extraction |
| List command | 4 | No filters, status filter, severity filter, link filter |
| Update command | 3 | Title update, severity update, not found |
| Delete command | 3 | Success, force flag, not found |
| Triage command | 4 | Success, already triaged error, missing severity flag, with assign |
| Note commands | 3 | Add note, list notes, entity type validation |
| Context commands | 3 | Set context, get context, clear context |
| Argument parsing | 5 | Key format validation, empty title, invalid severity value, invalid link format |

**Total: ~32 test cases**

### Mock Pattern

```go
type MockBugService struct {
    CreateBugFunc func(ctx context.Context, input services.CreateBugInput) (*models.Bug, error)
    GetBugFunc    func(ctx context.Context, key string) (*models.Bug, error)
    ListBugsFunc  func(ctx context.Context, filters services.BugFilters) ([]*models.Bug, error)
    // ...
}
```

---

## Error Handling

Commands translate BugService errors to user-facing messages using the standard exit code mapping:

| Error Type | Exit Code | Example Message |
|-----------|-----------|-----------------|
| Bug not found | 1 | `Bug not found: B999` |
| Database error | 2 | `Failed to create bug: database connection failed` |
| Invalid state | 3 | `Cannot triage bug B001: already in status 'in_fix'` |
| Validation error | 3 | `Invalid severity 'extreme': must be critical, high, medium, or low` |

---

## Incremental Over Epic

This PRD does not repeat epic-level content. It specifies:
- **Command signatures**: Exact flags, argument positions, and defaults that are not specified at epic level
- **Output format details**: Table columns, JSON structure, field extraction behavior
- **Test strategy**: CLI-specific mock patterns and test case coverage
- **Architecture decisions**: File placement, service accessor pattern, command tree structure
- **Error mapping**: Exit codes for each error category

Epic-level content referenced but not repeated:
- Bug entity model and schema (see [epic requirements REQ-F-001 through REQ-F-003](../requirements.md))
- Bug workflow definition (see [epic requirements REQ-F-004](../requirements.md))
- Persona profiles (see [epic personas](../personas.md))
- User journey flows (see [epic user journeys Journey 1](../user-journeys.md))

---

## Requirements Traceability

| Epic Requirement | F04 Coverage | Notes |
|-----------------|-------------|-------|
| REQ-F-001 | `shark bug create` command | CLI surface for bug creation |
| REQ-F-002 | `--severity` flag on create, list, triage | CLI surface for severity tracking |
| REQ-F-003 | `--link` flag on create | CLI surface for entity linking |
| REQ-F-005 | `shark bug triage` command | CLI surface for triage operation |
| REQ-F-006 | create, get, list, update, delete commands | Full CRUD CLI coverage |
| REQ-F-015 | note and context subcommands | CLI surface for notes/context |
| REQ-F-016 | `shark view B001` integration | Delegated to F06 |
| REQ-F-018 | `--link` flag on list command | CLI surface for linked-entity filtering |
| REQ-NF-001 | N/A | Performance is service/DB layer concern |
| REQ-NF-005 | All commands follow established CLI patterns | Enforced via pattern compliance |

---

*Last Updated*: 2026-03-03
