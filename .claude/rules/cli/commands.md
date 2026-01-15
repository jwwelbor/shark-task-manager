---
paths: "internal/cli/commands/**/*"
---

# CLI Commands Reference

This rule is loaded when working with CLI command implementations.

## Command Categories

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

### Standard Command Structure

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
    // Get database
    repoDb, err := cli.GetDB(cmd.Context())
    if err != nil {
        return fmt.Errorf("failed to get database: %w", err)
    }

    // Create repository
    repo := repository.NewTaskRepository(repoDb)

    // Business logic
    // ...

    // Output result
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }

    cli.Success("Operation completed")
    return nil
}
```

### Testing Commands

**CRITICAL**: Write tests using MOCKED repositories (never use real database in CLI tests)

```go
// See .claude/rules/testing/cli-tests.md for details
```
