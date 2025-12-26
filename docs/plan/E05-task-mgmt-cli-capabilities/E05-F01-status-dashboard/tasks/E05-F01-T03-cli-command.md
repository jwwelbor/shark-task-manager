# Task: E05-F01-T03 - Implement Status Dashboard CLI Command

**Feature**: E05-F01 Status Dashboard & Reporting
**Epic**: E05 Task Management CLI Capabilities
**Task Key**: E05-F01-T03

## Description

Implement the `shark status` command that serves as the user-facing interface to the status dashboard. This task bridges the CLI framework with the StatusService, handling flag parsing, error handling, context management, and output routing.

The command:
- Parses flags: `--epic`, `--recent`, `--include-archived`, `--json`, `--no-color` (inherited from root)
- Validates inputs and epic keys
- Creates and initializes the StatusService
- Calls GetDashboard with proper context and timeout management
- Routes to JSON or Rich table formatters
- Handles all error cases with user-friendly messages

**Why This Matters**: The CLI command is the user's entry point to the dashboard feature. A well-implemented command provides excellent error messages, respects all flags, and gracefully handles edge cases like database errors and timeouts.

## What You'll Build

Complete implementation of `shark status` command in `internal/cli/commands/status.go`:

```go
var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Display project dashboard",
    RunE:  runStatus,
}

func init() {
    // Register flags and command
}

func runStatus(cmd *cobra.Command, args []string) error {
    // 1. Database initialization
    // 2. Repository setup
    // 3. StatusService creation
    // 4. Request building
    // 5. GetDashboard call
    // 6. Output routing
}

func outputJSON(dashboard *StatusDashboard) error {
    // JSON marshaling and output
}
```

Plus error handling functions for each error case.

## Success Criteria

- [x] `shark status --help` shows all flags and usage examples
- [x] `shark status` displays full dashboard (routes to Rich formatter)
- [x] `shark status --json` outputs valid JSON
- [x] `shark status --epic=E01` filters to single epic
- [x] `shark status --recent=7d` filters recent completions
- [x] `shark status --no-color` removes ANSI codes
- [x] `shark status --include-archived` includes archived items
- [x] Combined flags work: `shark status --epic=E01 --json --recent=7d`
- [x] Invalid epic key shows helpful error: "Epic not found: E999"
- [x] Invalid timeframe shows helpful error: "Invalid timeframe: badformat"
- [x] Database connection error shows: "Failed to open database: [details]"
- [x] Timeout error shows: "Dashboard query timed out (>5s)"
- [x] Empty project shows: "No epics found. Create epics to get started."
- [x] Context timeout set to 5 seconds
- [x] Exit codes correct: 0 (success), 1 (user error), 2 (system error)
- [x] All tests pass: `go test ./internal/cli/commands -run Status -v`

## Implementation Notes

### Command Structure

```go
var (
    flagEpicKey       string
    flagRecentWindow  string
    flagIncludeArchived bool
)

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Display project dashboard",
    Long: `Display comprehensive project status dashboard with epic progress,
active tasks, blocked tasks, and recent completions.

The dashboard shows:
  • Project summary (total epics/features/tasks)
  • Epic breakdown with progress percentages
  • Active tasks grouped by agent type
  • Blocked tasks with blocking reasons
  • Recent completions from last 24 hours

Examples:
  shark status                 Show full project dashboard
  shark status --epic=E01      Show status for specific epic only
  shark status --json          Output as JSON for parsing
  shark status --no-color      Output without color codes
  shark status --recent=7d     Show completions from last 7 days
  shark status --epic=E01 --json  Combine multiple options`,

    RunE: runStatus,
}

func init() {
    statusCmd.Flags().StringVar(
        &flagEpicKey, "epic", "",
        "Filter dashboard to specific epic (e.g., 'E01')",
    )

    statusCmd.Flags().StringVar(
        &flagRecentWindow, "recent", "24h",
        "Time window for recent completions (e.g., '24h', '7d', '30d')",
    )

    statusCmd.Flags().BoolVar(
        &flagIncludeArchived, "include-archived", false,
        "Include archived epics and features (default: false)",
    )

    cli.RootCmd.AddCommand(statusCmd)
}
```

### Error Handling Pattern

Create specific error handler functions for each error category:

```go
func runStatus(cmd *cobra.Command, args []string) error {
    // 1. Database initialization
    database, err := initDatabase()
    if err != nil {
        return handleDatabaseError(err)
    }
    defer database.Close()

    // 2. Initialize repositories
    epicRepo := repository.NewEpicRepository(database)
    featureRepo := repository.NewFeatureRepository(database)
    taskRepo := repository.NewTaskRepository(database)
    historyRepo := repository.NewTaskHistoryRepository(database)

    // 3. Create service
    service := status.NewStatusService(database, epicRepo, featureRepo, taskRepo, historyRepo)

    // 4. Validate epic key if specified
    if flagEpicKey != "" {
        if _, err := epicRepo.GetByKey(context.Background(), flagEpicKey); err != nil {
            return handleEpicNotFound(flagEpicKey)
        }
    }

    // 5. Create request
    request := &status.StatusRequest{
        EpicKey:         flagEpicKey,
        RecentWindow:    flagRecentWindow,
        IncludeArchived: flagIncludeArchived,
    }

    // 6. Validate request
    if err := request.Validate(); err != nil {
        return handleValidationError(err)
    }

    // 7. Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // 8. Get dashboard
    dashboard, err := service.GetDashboard(ctx, request)
    if err != nil {
        return handleDashboardError(err)
    }

    // 9. Route to formatter
    if cli.GlobalConfig.JSON {
        return outputJSON(dashboard)
    } else {
        return outputRichTable(dashboard)
    }
}
```

### Error Handler Functions

```go
func handleDatabaseError(err error) error {
    if err == sql.ErrConnDone {
        return cli.Error("Database connection lost")
    }
    return cli.Error(fmt.Sprintf(
        "Failed to open database: %v\nCheck database path with --db flag",
        err))
}

func handleEpicNotFound(key string) error {
    return cli.Error(fmt.Sprintf(
        "Epic not found: %s\nUse 'shark epic list' to see available epics",
        key))
}

func handleValidationError(err error) error {
    return cli.Error(fmt.Sprintf(
        "Invalid input: %v\nRun 'shark status --help' for usage",
        err))
}

func handleDashboardError(err error) error {
    if errors.Is(err, context.DeadlineExceeded) {
        return cli.Error(
            "Dashboard query timed out (>5s)\n" +
            "Try filtering with --epic=<key> or check database performance")
    }
    if status.IsStatusError(err) {
        return cli.Error(err.Error())
    }
    return cli.Error(fmt.Sprintf(
        "Error retrieving status: %v\nCheck log output for details",
        err))
}

func outputJSON(dashboard *status.StatusDashboard) error {
    data, err := json.MarshalIndent(dashboard, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal JSON: %w", err)
    }
    fmt.Println(string(data))
    return nil
}
```

### Empty Data Handling

When dashboard has no epics:

```go
if len(dashboard.Epics) == 0 {
    fmt.Println("\nNo epics found. Create epics to get started.")
    return nil  // Exit code 0, not an error
}
```

This is intentional - empty projects should not exit with error code.

### Context Management

Always use context.WithTimeout to prevent hanging on slow databases:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

dashboard, err := service.GetDashboard(ctx, request)
```

The 5-second timeout balances responsiveness with allowing for reasonable query times on large projects.

### Exit Codes

Follow project conventions:
- 0: Success (including empty projects)
- 1: User error (invalid epic key, bad timeframe)
- 2: System error (database, timeout)

The `cli.Error()` helper function automatically sets exit code 2.

## Dependencies

- Cobra: CLI framework (already available)
- Go standard library: context, database/sql, encoding/json, fmt, time
- Internal packages:
  - `internal/cli` - cli.GlobalConfig, cli.RootCmd, cli.Error()
  - `internal/db` - *db.DB
  - `internal/repository` - All repository types
  - `internal/status` - StatusService, StatusRequest, StatusDashboard

## Related Tasks

- **E05-F01-T01**: Service Data Structures - Defines types used
- **E05-F01-T02**: Database Queries - Implements GetDashboard
- **E05-F01-T04**: Output Formatting - Implements outputRichTable

## Acceptance Criteria

**Functional**:
- [ ] Command registers in CLI: `shark help | grep status`
- [ ] All flags work individually and in combination
- [ ] Database initialization handles errors gracefully
- [ ] Repository creation succeeds without errors
- [ ] StatusService created with all repositories
- [ ] GetDashboard called with correct request object
- [ ] JSON output is valid and parseable
- [ ] Empty project shows helpful message, not error
- [ ] All error cases produce user-friendly messages

**Error Handling**:
- [ ] Database connection error: helpful message with --db hint
- [ ] Epic not found: suggests `shark epic list` command
- [ ] Invalid timeframe: lists valid options
- [ ] Timeout: suggests filtering with --epic flag
- [ ] Exit codes correct (0, 1, or 2)

**Performance**:
- [ ] Command completes in <600ms (including <500ms dashboard)
- [ ] Context timeout (5 seconds) prevents hangs

**Testing**:
- [ ] Unit test: command with --json flag
- [ ] Unit test: command with --epic flag
- [ ] Unit test: command with invalid epic
- [ ] Unit test: command with empty database
- [ ] Integration test: JSON output is valid
- [ ] All tests pass: `go test ./internal/cli/commands -run Status -v`

**Code Quality**:
- [ ] Error handling follows project patterns
- [ ] Flag parsing is robust
- [ ] Command registration correct
- [ ] Comments on public functions
- [ ] No linting errors: `golangci-lint run ./internal/cli/commands`

## Verification Steps

```bash
# Test command exists
./bin/shark status --help
# Should show help with flags

# Test with JSON
./bin/shark status --json | jq '.' | head -20
# Should output valid JSON

# Test with empty database
rm shark-tasks.db* 2>/dev/null
./bin/shark init --non-interactive
./bin/shark status
# Should show "No epics found" message

# Test error cases
./bin/shark status --epic=INVALID 2>&1
# Should show "Epic not found: INVALID"

./bin/shark status --recent=badformat 2>&1
# Should show "Invalid timeframe: badformat"

# Run all status command tests
go test ./internal/cli/commands -run Status -v
```

## Implementation Checklist

See Phase 3 in implementation-checklist.md:
- [ ] Task 3.1: Create command definition with flags
- [ ] Task 3.2: Implement command handler (runStatus)
- [ ] Task 3.3: JSON output formatter
- [ ] Task 3.4: Error handling in command
- [ ] Task 3.5: Integration with existing CLI
- [ ] Task 3.6: Command tests
