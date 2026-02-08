---
paths: "internal/cli/commands/**/*"
---

# CLI Commands Reference

This rule is loaded when working with CLI command implementations.

## Command Categories

### Smart Dispatchers (Recommended Primary Interface)

Smart dispatchers automatically detect the entity type based on key format. These are the recommended commands for most operations:

- `shark list [EPIC] [FEATURE] [--json]` - Smart list dispatcher
  - No args: List all epics
  - With epic key: `shark list E07` - List features in epic
  - With epic and feature: `shark list E07 F01` - List tasks in feature
  - Auto-detection: Key format determines what to list

- `shark get <KEY> [--json]` - Smart get dispatcher (auto-detects entity type)
  - Epic format: `shark get E07` - Get epic details
  - Feature format: `shark get E07-F01` - Get feature details (also works: `shark get F01`)
  - Task format: `shark get E07-F01-001` - Get task details (also works: `shark get T-E07-F01-001`)

- `shark status <KEY> [--json]` - Get entity status and progress
  - `shark status E07` - Epic status with feature rollups
  - `shark status E07-F01` - Feature status with task breakdown
  - `shark status E07-F01-001` - Task status and history

- `shark history <KEY> [--json]` - Get entity change history
  - `shark history E07-F01-001` - Task change history
  - `shark history E07-F01` - Feature change history

All smart dispatchers support:
- Case insensitive keys: `e07`, `E07`, `E07-user-management` all work
- `--json` flag for machine-readable output
- Both numeric and slugged key formats

### Initialization
- `shark init --non-interactive`: Setup project infrastructure (folders, database, config)

### Epic Management
- `shark epic create --title="..." [--file=<path>] [--force] [--priority=...] [--business-value=...] [--json]`
  - `--file`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another epic or feature
- `shark epic list [--json]`
- `shark epic get <epic-key> [--json]`
  - Case insensitive: `shark epic get E07`, `shark epic get e07`

### Feature Management
- **Positional syntax (recommended):** `shark feature create <epic-key> "<title>" [--file=<path>] [--force] [--execution-order=...] [--json]`
- **Flag syntax (legacy):** `shark feature create --epic=<epic-key> --title="..." [--file=<path>] [--force] [--execution-order=...] [--json]`
  - `--file`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another feature or epic
  - Case insensitive: `shark feature create E07 "Title"`, `shark feature create e07 "Title"`
- `shark feature list [EPIC] [--json]` - List features, optionally filter by epic key
  - Examples: `shark feature list`, `shark feature list E04`, `shark feature list e04`, `shark feature list E04 --json`
  - Flag syntax still works: `shark feature list --epic=E04`
- `shark feature get <feature-key> [--json]`
  - Case insensitive: `shark feature get E07-F01`, `shark feature get e07-f01`, `shark feature get F01`, `shark feature get f01`

### File Path Organization

Epics and features support custom file paths for flexible project organization:

```bash
# Create epic with custom file path
shark epic create "Q1 2025 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# Create feature with custom file path
shark feature create --epic=E01 "User Growth" --file="docs/roadmap/2025-q1/features/user-growth.md"

# Default behavior (no --file flag)
shark epic create "User Management"  # Creates docs/plan/E07-user-management/epic.md
shark feature create E07 "Authentication"  # Positional syntax (recommended)
shark feature create --epic=E07 --title="Authentication"  # Flag syntax (legacy)
# Creates: docs/plan/E07-user-management/E07-F01-authentication/feature.md
```

### Task Management (Primary AI Interface)
- `shark task next [--agent=<type>] [--epic=<epic>] [--json]`: Get next available task
- `shark task list [EPIC] [FEATURE] [--status=<status>] [--agent=<type>] [--json]` - List tasks with flexible positional filtering
  - Examples: `shark task list`, `shark task list E04`, `shark task list e04`, `shark task list E04 F01`, `shark task list E04-F01`
  - Flag syntax still works: `shark task list --epic=E04 --feature=F01`
- `shark task get <task-key> [--json]`
  - Short format (recommended): `shark task get E07-F20-001`, `shark task get e07-f20-001`
  - Traditional format: `shark task get T-E07-F20-001`, `shark task get t-e07-f20-001`
- **Positional syntax (recommended):** `shark task create <epic> <feature> "<title>" [--agent=<type>] [--priority=<1-10>] [--depends-on=...] [--file=<path>] [--force]`
  - 3-arg format: `shark task create E07 F20 "Task Title"`
  - 2-arg format: `shark task create E07-F20 "Task Title"`
  - Case insensitive: `shark task create e07 f20 "Task Title"`
- **Flag syntax (legacy):** `shark task create --epic=E04 --feature=F06 --title="..." [--agent=<type>] [--priority=<1-10>] [--depends-on=...] [--file=<path>] [--force]`
  - `--file`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another task
- `shark task start <task-key> [--agent=<agent-id>] [--json]`
  - Short format: `shark task start E07-F20-001`, `shark task start e07-f20-001`
- `shark task complete <task-key> [--notes="..."] [--json]` (ready for review)
  - Short format: `shark task complete E07-F20-001`, `shark task complete e07-f20-001`
- `shark task approve <task-key> [--notes="..."] [--json]` (mark completed)
  - Short format: `shark task approve E07-F20-001`, `shark task approve e07-f20-001`
- `shark task reopen <task-key> [--notes="..."] [--json]` (back to in_progress)
- `shark task block <task-key> --reason="..." [--json]`
- `shark task unblock <task-key> [--json]`

### Synchronization
- `shark sync [--dry-run] [--strategy=<strategy>] [--create-missing] [--cleanup] [--pattern=<type>] [--json]`

### Configuration
- `shark config set <key> <value>`
- `shark config get <key>`

## Command Implementation Pattern

### Standard Command Structure (Target Pattern)

CLI commands must be **thin wrappers**: parse arguments, call a service, format output. **No business logic in commands.**

```go
var myCmd = &cobra.Command{
    Use:   "mycommand [args]",
    Short: "Brief description",
    Long:  "Detailed description",
    Args:  cobra.ExactArgs(1), // or MinimumNArgs, etc.
    RunE:  runMyCommand,
}

func init() {
    // Add flags
    myCmd.Flags().StringVar(&myFlag, "flag", "", "Flag description")

    // Register command
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
