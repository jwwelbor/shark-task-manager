---
paths: "internal/cli/commands/**/*"
---

# CLI Commands Reference

This rule is loaded when working with CLI command implementations.

## Command Architecture (E17)

The CLI is organized into categories, not by entity type. Entity type is auto-detected from key format:
- `E##` → Epic
- `E##-F##` or `F##` → Feature
- `E##-F##-###` or `T-E##-F##-###` → Task

## Command Categories

### Core Commands (Entity Auto-Detection)

- `shark get <KEY> [--json] [--field <name>]` — Get entity details (auto-detects type)
- `shark list [EPIC] [FEATURE] [--json]` — List entities with smart filtering
- `shark create <type> [args]` — Create epic, feature, or task
- `shark delete <key> [--force]` — Delete an entity
- `shark view <key>` — View entity markdown file

### Status & Analytics

- `shark status [KEY]` — Project dashboard or entity status
- `shark status set <key> <status> [--reason] [--force]` — Set status directly
- `shark status advance <key>` — Advance to next workflow status
- `shark status options <key>` — Show valid next statuses
- `shark status history <key>` — View status change history
- `shark progress <key>` — Detailed progress breakdown
- `shark analytics [key]` — Project or entity analytics

### Entity Management

#### Task Commands (18 subcommands)

**CRUD:**
- `shark task create <epic> <feature> "<title>" [--order=N] [--agent=TYPE] [--priority=N]`
- `shark task get <key> [--json] [--field <name>]`
- `shark task list [EPIC] [FEATURE] [--status=S] [--agent=TYPE] [--blocked] [--json]`
- `shark task update <key> [--title=...] [--priority=N] [--execution-order=N]`
- `shark task delete <key>`

**Lifecycle:**
- `shark task approve <key> [--notes="..."]`
- `shark task reopen <key> [--notes="..."]`
- `shark task next-status <key> [--force]`
- `shark task set-status <key> <status> [--reason="..."] [--force]`

**Dependencies:**
- `shark task deps <key> [--depth=N]` — Dependency tree
- `shark task link <key1> <key2> --type=TYPE` — Link entities
- `shark task unlink <key1> <key2>` — Remove link
- `shark task blocked-by <key>` — Show what blocks this task
- `shark task blocks <key>` — Show what this task blocks

**Context & Notes:**
- `shark task context set/get/clear <key>` — Manage context fields
- `shark task note add <key> --content="..." [--type=TYPE]`
- `shark task notes <key>` — View notes
- `shark task criteria <key> list/import/check/fail` — Acceptance criteria
- `shark task resume <key>` — Resume with full context

**History:**
- `shark task history <key>` — Status change history
- `shark task sessions <key>` — Session history
- `shark task timeline <key>` — Full timeline

#### Feature Commands (13 subcommands)

- `shark feature create <epic> "<title>" [--execution-order=N]`
- `shark feature get <key> [--json]`
- `shark feature list [EPIC] [--json]`
- `shark feature update <key> [--title=...]`
- `shark feature delete <key>`
- `shark feature complete <key>`
- `shark feature next-status <key>`
- `shark feature set-status <key> <status>`
- `shark feature context set/get/clear <key>`
- `shark feature note add <key> --content="..."`
- `shark feature notes <key>`
- `shark feature criteria <key>`
- `shark feature resume <key>`

#### Epic Commands (14 subcommands)

- `shark epic create "<title>" [--priority=N] [--business-value=N]`
- `shark epic get <key> [--json]`
- `shark epic list [--json]`
- `shark epic delete <key>`
- `shark epic update <key> [--title=...]`
- `shark epic complete <key>`
- `shark epic next-status <key>`
- `shark epic set-status <key> <status>`
- `shark epic status <key>` — Epic status with rollups
- `shark epic context set/get/clear <key>`
- `shark epic note add <key> --content="..."`
- `shark epic notes <key>`
- `shark epic resume <key>`

#### Idea Commands (6 subcommands)

- `shark idea create "<title>" --description="..."`
- `shark idea get <id>`
- `shark idea list [--status=S]`
- `shark idea update <id> [--status=S]`
- `shark idea delete <id>`
- `shark idea promote <id> [--epic=KEY]`

### Discovery Commands

- `shark search <query> [--type=TYPE]` — Search across entities
- `shark notes <key>` — View entity notes
- `shark related-docs list/add/delete` — Manage related documents

### Setup & Configuration

- `shark admin init [--non-interactive] [--force]` — Initialize project config, database, and planning folders
- `shark admin validate` — Validate project structure
- `shark migrate slugs` — Backfill slugs
- `shark cloud init/status` — Cloud database
- `shark config show/validate/get-format/get-status-action` — Config management

### Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Machine-readable JSON output |
| `--field <name>` | Extract single field (implies `--json`) |
| `--no-color` | Disable colored output |
| `--verbose` / `-v` | Debug logging |
| `--db <path>` | Override database path |
| `--config <path>` | Override config path |

---

## Command Implementation Pattern

### Standard Command Structure (Target Pattern)

CLI commands must be **thin wrappers**: parse arguments, call a service, format output. **No business logic in commands.**

```go
var myCmd = &cobra.Command{
    Use:   "mycommand [args]",
    Short: "Brief description",
    Long:  "Detailed description",
    Args:  cobra.ExactArgs(1),
    RunE:  runMyCommand,
}

func init() {
    myCmd.Flags().StringVar(&myFlag, "flag", "", "Flag description")
    cli.RootCmd.AddCommand(myCmd)
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    // 1. Parse arguments
    taskKey := args[0]

    // 2. Call service (all business logic lives in services)
    svc := cli.GetTaskService()
    result, err := svc.CompleteTask(cmd.Context(), taskKey, notes)
    if err != nil {
        return err
    }

    // 3. Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }

    cli.Success(fmt.Sprintf("Task %s completed", result.Key))
    return nil
}
```

### Anti-Pattern: Direct Repository Access (Legacy - Do Not Add More)

> **DEPRECATED**: Many existing commands call repositories directly. This is being
> refactored in Epic E15. Do NOT add new commands that follow this pattern.

```go
// BAD - do not do this in new code
func runMyCommand(cmd *cobra.Command, args []string) error {
    repoDb, err := cli.GetDB(cmd.Context())
    repo := repository.NewTaskRepository(repoDb)
    // ... business logic that should be in a service ...
}
```

### What Belongs Where

| Layer | Responsibilities | Does NOT Belong |
|-------|-----------------|-----------------|
| **CLI Command** | Parse args/flags, call service, format JSON/table output, exit codes | Business rules, repo calls, transactions, filtering logic |
| **Service** | Business rules, validation, orchestration, transactions, status transitions | Argument parsing, output formatting, Cobra dependencies |
| **Repository** | CRUD queries, data access, prepared statements | Progress calculation, status derivation, workflow logic |

### Testing Commands

**CRITICAL**: Write tests using MOCKED services (or mocked repositories for legacy commands). Never use real database in CLI tests.

```go
// See .claude/rules/testing/cli-tests.md for details
```
