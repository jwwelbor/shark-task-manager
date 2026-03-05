# API Contracts: Bug CLI Commands

**Feature Key**: E18-F04
**Complexity Tier**: STANDARD
**Status**: Technical Refinement

---

## Overview

This document defines the contract between CLI command handlers and the BugService. Each command is specified with its exact Cobra definition, the service method it calls, and the expected input/output types.

F04 does not define the BugService implementation -- that is F02's responsibility. F04 defines how CLI commands consume the service interface.

---

## Service Method Contracts (Consumed by F04)

These are the BugService methods that F04 depends on. F02 must deliver these with the specified signatures.

### BugService Interface (F02 Delivers)

```go
type BugService interface {
    CreateBug(ctx context.Context, input CreateBugInput) (*models.Bug, error)
    GetBug(ctx context.Context, key string) (*models.Bug, error)
    ListBugs(ctx context.Context, filters BugFilters) ([]*models.Bug, error)
    UpdateBug(ctx context.Context, key string, input UpdateBugInput) (*models.Bug, error)
    DeleteBug(ctx context.Context, key string) error
    TriageBug(ctx context.Context, key string, input TriageBugInput) (*models.Bug, error)
}
```

### DTOs (F02 Delivers)

```go
// CreateBugInput is the input for creating a new bug.
type CreateBugInput struct {
    Title    string // Required. Cannot be empty.
    Severity string // Optional. Default: "medium". Valid: critical, high, medium, low.
    LinkKey  string // Optional. Entity key to link (E07, E07-F01, E07-F01-001).
}

// BugFilters is the input for filtering bug lists.
type BugFilters struct {
    Status   string // Optional. Filter by bug status.
    Severity string // Optional. Filter by severity.
    LinkKey  string // Optional. Filter by linked entity key.
}

// UpdateBugInput is the input for updating bug fields.
type UpdateBugInput struct {
    Title    *string // Optional. Non-nil means update title.
    Severity *string // Optional. Non-nil means update severity.
}

// TriageBugInput is the input for triaging a bug.
type TriageBugInput struct {
    Severity string  // Required. Must be valid severity value.
    Assign   *string // Optional. Agent to assign.
}
```

### Bug Model (F02 Delivers)

```go
type Bug struct {
    ID               int64     `json:"id"`
    Key              string    `json:"key"`               // B001, B002, ...
    Title            string    `json:"title"`
    Status           string    `json:"status"`            // reported, triaged, in_fix, ...
    Severity         string    `json:"severity"`          // critical, high, medium, low
    Slug             string    `json:"slug"`
    LinkedEntityType string    `json:"linked_entity_type,omitempty"` // epic, feature, task
    LinkedEntityKey  string    `json:"linked_entity_key,omitempty"`
    FilePath         string    `json:"file_path,omitempty"`
    ContextData      string    `json:"context_data,omitempty"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

---

## Command Contracts

### 1. shark bug create

**Cobra Definition**:
```go
var bugCreateCmd = &cobra.Command{
    Use:   "create <title>",
    Short: "Create a new bug report",
    Long:  `Create a new bug report with auto-generated key (B### format).

Examples:
  shark bug create "Login page crashes on submit"
  shark bug create "API timeout" --severity=critical
  shark bug create "Button misaligned" --link=E07-F01
  shark bug create "New bug" --json`,
    Args: cobra.ExactArgs(1),
    RunE: runBugCreate,
}
```

**Flags**:
| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--severity` | string | "" (service defaults to "medium") | No | Bug severity |
| `--link` | string | "" | No | Entity key to link |

**Service Call**: `BugService.CreateBug(ctx, CreateBugInput{Title, Severity, LinkKey})`

**Handler Logic**:
```go
func runBugCreate(cmd *cobra.Command, args []string) error {
    input := services.CreateBugInput{
        Title:    args[0],
        Severity: bugSeverity,
        LinkKey:  bugLink,
    }

    svc := cli.GetBugService()
    bug, err := svc.CreateBug(cmd.Context(), input)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bug)
    }
    cli.Success(fmt.Sprintf("Created bug %s", bug.Key))
    return nil
}
```

**Output (success)**: `Created bug B001`
**Output (--json)**: Full Bug model as JSON
**Errors**: Empty title (validation), invalid severity (validation), linked entity not found (not found)

---

### 2. shark bug get

**Cobra Definition**:
```go
var bugGetCmd = &cobra.Command{
    Use:   "get <key>",
    Short: "Get bug details",
    Long:  `Display detailed information about a specific bug.

Examples:
  shark bug get B001
  shark bug get B001 --json
  shark bug get B001 --field severity`,
    Args: cobra.ExactArgs(1),
    RunE: runBugGet,
}
```

**Flags**: Inherits global `--json` and `--field` flags.

**Service Call**: `BugService.GetBug(ctx, key)`

**Handler Logic**:
```go
func runBugGet(cmd *cobra.Command, args []string) error {
    key := args[0]

    svc := cli.GetBugService()
    bug, err := svc.GetBug(cmd.Context(), key)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bug)
    }
    return printBugDetail(bug)
}
```

**Output (default)**: Formatted detail view (key, title, status, severity, linked entity, dates)
**Output (--json)**: Full Bug model as JSON
**Output (--field severity)**: Just the severity value string
**Errors**: Bug not found (exit 1)

---

### 3. shark bug list

**Cobra Definition**:
```go
var bugListCmd = &cobra.Command{
    Use:   "list",
    Short: "List bugs",
    Long:  `List bugs with optional filtering by status, severity, and linked entity.

Examples:
  shark bug list
  shark bug list --status=reported
  shark bug list --severity=critical
  shark bug list --link=E07-F01
  shark bug list --status=reported --severity=high --json`,
    RunE: runBugList,
}
```

**Flags**:
| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--status` | string | "" | No | Filter by bug status |
| `--severity` | string | "" | No | Filter by severity |
| `--link` | string | "" | No | Filter by linked entity key |

**Service Call**: `BugService.ListBugs(ctx, BugFilters{Status, Severity, LinkKey})`

**Handler Logic**:
```go
func runBugList(cmd *cobra.Command, args []string) error {
    filters := services.BugFilters{
        Status:   bugStatus,
        Severity: bugSeverity,
        LinkKey:  bugLink,
    }

    svc := cli.GetBugService()
    bugs, err := svc.ListBugs(cmd.Context(), filters)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bugs)
    }

    if len(bugs) == 0 {
        cli.Info("No bugs found")
        return nil
    }
    return printBugTable(bugs)
}
```

**Table Format**:
```
KEY    TITLE                                    STATUS     SEVERITY  LINKED TO
B001   Login fails on Safari when 2FA enabled   reported   high      E07-F01
B002   Dashboard chart tooltip renders behind    triaged    medium    E07-F03
```

**Output (--json)**: JSON array of Bug objects
**Output (empty)**: "No bugs found" message, exit code 0

---

### 4. shark bug update

**Cobra Definition**:
```go
var bugUpdateCmd = &cobra.Command{
    Use:   "update <key>",
    Short: "Update a bug",
    Long:  `Update bug fields (title, severity).

At least one update flag must be provided.

Examples:
  shark bug update B001 --title="Updated title"
  shark bug update B001 --severity=critical
  shark bug update B001 --title="New" --severity=low --json`,
    Args: cobra.ExactArgs(1),
    RunE: runBugUpdate,
}
```

**Flags**:
| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | "" | No (but one update flag required) | New title |
| `--severity` | string | "" | No (but one update flag required) | New severity |

**Service Call**: `BugService.UpdateBug(ctx, key, UpdateBugInput{Title, Severity})`

**Handler Logic**:
```go
func runBugUpdate(cmd *cobra.Command, args []string) error {
    key := args[0]

    input := services.UpdateBugInput{}
    if cmd.Flags().Changed("title") {
        input.Title = &bugTitle
    }
    if cmd.Flags().Changed("severity") {
        input.Severity = &bugSeverity
    }

    if input.Title == nil && input.Severity == nil {
        return fmt.Errorf("at least one update flag is required (--title or --severity)")
    }

    svc := cli.GetBugService()
    bug, err := svc.UpdateBug(cmd.Context(), key, input)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bug)
    }
    cli.Success(fmt.Sprintf("Updated bug %s", bug.Key))
    return nil
}
```

**Note**: The "at least one update flag" check is the one case where the command handler performs a trivial validation. This is acceptable because it is a CLI UX concern (detecting that the user provided no actionable input), not a business rule.

**Errors**: Bug not found (exit 1), invalid severity (exit 3), no flags provided (exit 3)

---

### 5. shark bug delete

**Cobra Definition**:
```go
var bugDeleteCmd = &cobra.Command{
    Use:   "delete <key>",
    Short: "Delete a bug",
    Long:  `Delete a bug and its associated markdown file.

Confirmation is required unless --force is provided.

Examples:
  shark bug delete B001
  shark bug delete B001 --force
  shark bug delete B001 --force --json`,
    Args: cobra.ExactArgs(1),
    RunE: runBugDelete,
}
```

**Flags**:
| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--force` | bool | false | No | Skip confirmation prompt |

**Service Call**: `BugService.DeleteBug(ctx, key)`

**Handler Logic**:
```go
func runBugDelete(cmd *cobra.Command, args []string) error {
    key := args[0]

    if !bugForce {
        // Get bug first for confirmation display
        svc := cli.GetBugService()
        bug, err := svc.GetBug(cmd.Context(), key)
        if err != nil {
            return err
        }
        if !confirmBugDelete(bug) {
            cli.Info("Delete cancelled")
            return nil
        }
    }

    svc := cli.GetBugService()
    if err := svc.DeleteBug(cmd.Context(), key); err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]string{"deleted": key})
    }
    cli.Success(fmt.Sprintf("Deleted bug %s", key))
    return nil
}
```

**Errors**: Bug not found (exit 1)

---

### 6. shark bug triage

**Cobra Definition**:
```go
var bugTriageCmd = &cobra.Command{
    Use:   "triage <key> --severity=S [--assign=AGENT]",
    Short: "Triage a bug",
    Long:  `Triage a bug by setting severity and optionally assigning an agent.
Advances bug status from 'reported' to 'triaged'.

Examples:
  shark bug triage B001 --severity=high
  shark bug triage B001 --severity=medium --assign=developer
  shark bug triage B001 --severity=critical --json`,
    Args: cobra.ExactArgs(1),
    RunE: runBugTriage,
}
```

**Flags**:
| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--severity` | string | "" | **Yes** (MarkFlagRequired) | Bug severity |
| `--assign` | string | "" | No | Agent to assign |

**Service Call**: `BugService.TriageBug(ctx, key, TriageBugInput{Severity, Assign})`

**Handler Logic**:
```go
func runBugTriage(cmd *cobra.Command, args []string) error {
    key := args[0]

    input := services.TriageBugInput{
        Severity: bugSeverity,
    }
    if cmd.Flags().Changed("assign") {
        input.Assign = &bugAssign
    }

    svc := cli.GetBugService()
    bug, err := svc.TriageBug(cmd.Context(), key, input)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bug)
    }
    cli.Success(fmt.Sprintf("Triaged bug %s (severity: %s, status: %s)", bug.Key, bug.Severity, bug.Status))
    return nil
}
```

**Errors**: Bug not found (exit 1), already triaged / invalid transition (exit 3), invalid severity (exit 3)

---

### 7. shark bug note add

**Cobra Definition**:
```go
var bugNoteAddCmd = &cobra.Command{
    Use:   "add <key> --type=TYPE <content>",
    Short: "Add a note to a bug",
    Long:  `Add a note to a bug with a specified type.

Note types: comment, decision, progress, blocker, reference, implementation, future

Examples:
  shark bug note add B001 --type=comment "Reproduced on Safari 17.2"
  shark bug note add B001 --type=decision "Root cause is race condition"`,
    Args: cobra.ExactArgs(2),
    RunE: runBugNoteAdd,
}
```

**Flags**:
| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--type` | string | "" | **Yes** | Note type |

**Service Call**: `NoteService.AddNote(ctx, "bug", key, noteType, content)`

**Handler Logic**:
```go
func runBugNoteAdd(cmd *cobra.Command, args []string) error {
    key := args[0]
    content := args[1]

    noteSvc, err := cli.GetNoteService(cmd.Context())
    if err != nil {
        return err
    }

    if err := noteSvc.AddNote(cmd.Context(), "bug", key, bugNoteType, content); err != nil {
        return err
    }

    cli.Success(fmt.Sprintf("Note added to bug %s", key))
    return nil
}
```

**Note**: This reuses the existing NoteService, not BugService. The entity_type="bug" routes the note to the correct entity.

---

### 8. shark bug notes

**Cobra Definition**:
```go
var bugNotesCmd = &cobra.Command{
    Use:   "notes <key>",
    Short: "List notes for a bug",
    Long:  `Display all notes for a specific bug.

Examples:
  shark bug notes B001
  shark bug notes B001 --json`,
    Args: cobra.ExactArgs(1),
    RunE: runBugNotes,
}
```

**Service Call**: `NoteService.ListNotes(ctx, "bug", key)`

**Handler Logic**:
```go
func runBugNotes(cmd *cobra.Command, args []string) error {
    key := args[0]

    noteSvc, err := cli.GetNoteService(cmd.Context())
    if err != nil {
        return err
    }

    notes, err := noteSvc.ListNotes(cmd.Context(), "bug", key)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(notes)
    }
    return printNotes(notes)
}
```

---

### 9. shark bug context set

**Cobra Definition**:
```go
var bugContextSetCmd = &cobra.Command{
    Use:   "set <key> --field F --value V",
    Short: "Set a context field on a bug",
    Args:  cobra.ExactArgs(1),
    RunE:  runBugContextSet,
}
```

**Flags**:
| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--field` | string | **Yes** | Context field name |
| `--value` | string | **Yes** | Context field value |

**Service Call**: `ContextService.SetField(ctx, "bug", key, field, value)`

---

### 10. shark bug context get

**Cobra Definition**:
```go
var bugContextGetCmd = &cobra.Command{
    Use:   "get <key>",
    Short: "Get context for a bug",
    Args:  cobra.ExactArgs(1),
    RunE:  runBugContextGet,
}
```

**Service Call**: `ContextService.GetContext(ctx, "bug", key)`

---

### 11. shark bug context clear

**Cobra Definition**:
```go
var bugContextClearCmd = &cobra.Command{
    Use:   "clear <key> --field F",
    Short: "Clear a context field on a bug",
    Args:  cobra.ExactArgs(1),
    RunE:  runBugContextClear,
}
```

**Flags**:
| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--field` | string | **Yes** | Context field to clear |

**Service Call**: `ContextService.ClearField(ctx, "bug", key, field)`

---

## Helper Functions

### printBugDetail

Renders formatted bug detail view for `shark bug get` (non-JSON output).

```go
func printBugDetail(bug *models.Bug) error {
    fmt.Printf("Bug %s\n", bug.Key)
    fmt.Printf("  Title:     %s\n", bug.Title)
    fmt.Printf("  Status:    %s\n", bug.Status)
    fmt.Printf("  Severity:  %s\n", bug.Severity)
    if bug.LinkedEntityKey != "" {
        fmt.Printf("  Linked To: %s (%s)\n", bug.LinkedEntityKey, bug.LinkedEntityType)
    }
    fmt.Printf("  Created:   %s\n", bug.CreatedAt.Format(time.RFC3339))
    fmt.Printf("  Updated:   %s\n", bug.UpdatedAt.Format(time.RFC3339))
    return nil
}
```

### printBugTable

Renders table for `shark bug list` (non-JSON output).

```go
func printBugTable(bugs []*models.Bug) error {
    headers := []string{"KEY", "TITLE", "STATUS", "SEVERITY", "LINKED TO"}
    var rows [][]string
    for _, b := range bugs {
        linkedTo := ""
        if b.LinkedEntityKey != "" {
            linkedTo = b.LinkedEntityKey
        }
        rows = append(rows, []string{
            b.Key,
            truncateString(b.Title, 45),
            b.Status,
            b.Severity,
            linkedTo,
        })
    }
    return cli.OutputTable(headers, rows)
}
```

### confirmBugDelete

Prompts for confirmation before delete (when --force is not set).

```go
func confirmBugDelete(bug *models.Bug) bool {
    fmt.Printf("Delete bug %s: %s? [y/N] ", bug.Key, bug.Title)
    var response string
    fmt.Scanln(&response)
    return strings.ToLower(response) == "y"
}
```

---

## JSON Output Schema

All commands that support `--json` output serialize the Bug model directly using `cli.OutputJSON()`. The JSON structure is:

```json
{
  "id": 1,
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

For list output (`shark bug list --json`), the response is a JSON array of the above objects.

For delete output (`shark bug delete --force --json`), the response is:
```json
{
  "deleted": "B001"
}
```

---

*Last Updated*: 2026-03-03
