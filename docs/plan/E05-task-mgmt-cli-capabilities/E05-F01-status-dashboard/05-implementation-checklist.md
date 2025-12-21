# Implementation Checklist: Status Dashboard (E05-F01)

**Document Version**: 1.0
**Status**: Implementation Guide
**Last Updated**: 2025-12-19

This checklist provides step-by-step guidance for implementing the Status Dashboard feature. Each section corresponds to an implementation phase with specific, actionable tasks.

---

## Table of Contents

1. [Pre-Implementation Setup](#pre-implementation-setup)
2. [Phase 1: Core Service (Data Structures)](#phase-1-core-service-data-structures)
3. [Phase 2: Database Queries](#phase-2-database-queries)
4. [Phase 3: CLI Command](#phase-3-cli-command)
5. [Phase 4: Output Formatting](#phase-4-output-formatting)
6. [Phase 5: Testing & Optimization](#phase-5-testing--optimization)
7. [Integration & Verification](#integration--verification)

---

## Pre-Implementation Setup

### Task 1: Create Service Package Structure

- [ ] Create directory: `internal/status/`
- [ ] Create file: `internal/status/status.go`
- [ ] Create file: `internal/status/models.go`
- [ ] Create file: `internal/status/errors.go`
- [ ] Create file: `internal/status/status_test.go`

**Verification**:
```bash
ls -la internal/status/
# Should show: errors.go, models.go, status.go, status_test.go
```

### Task 2: Update Command Package

- [ ] Verify `internal/cli/commands/status.go` does not exist
- [ ] Verify `internal/cli/commands/commands.go` exists
- [ ] Plan command registration in `commands.go` init function

**Verification**:
```bash
grep -l "statusCmd" internal/cli/commands/*.go
# Should return nothing or show status.go when ready
```

### Task 3: Review Existing Repository Patterns

- [ ] Read `internal/repository/task_repository.go` (sample implementation)
- [ ] Note error handling patterns used
- [ ] Note context.Context parameter usage
- [ ] Note how methods are documented

**Verification**:
```bash
grep "func (r \*TaskRepository)" internal/repository/task_repository.go | head -5
```

### Task 4: Understand Model Structure

- [ ] Read `internal/models/task.go`
- [ ] Read `internal/models/epic.go`
- [ ] Note JSON tags and database field mapping
- [ ] Note how `sql.NullTime` is used for optional timestamps

**Verification**:
```bash
grep "json:" internal/models/task.go | head -5
```

---

## Phase 1: Core Service (Data Structures)

**Estimated Duration**: 3-4 hours
**Deliverable**: Service interface, data models, basic implementation

### Task 1.1: Define Output Data Models

**File**: `internal/status/models.go`

Create these structs (in order):

- [ ] `StatusDashboard` struct
  - Fields: Summary, Epics, ActiveTasks, BlockedTasks, RecentCompletions, Filter
  - Add JSON tags with proper formatting

- [ ] `ProjectSummary` struct
  - Fields: Epics (CountBreakdown), Features (CountBreakdown), Tasks (StatusBreakdown)
  - Add fields: OverallProgress, BlockedCount

- [ ] `CountBreakdown` struct (for Epics/Features counts)
  - Fields: Total, Active, Completed

- [ ] `StatusBreakdown` struct (for Tasks by status)
  - Fields: Total, Todo, InProgress, ReadyForReview, Completed, Blocked

- [ ] `EpicSummary` struct
  - Fields: Key, Title, Progress, Health, TasksTotal, TasksCompleted, BlockedCount, Priority, Status
  - Add JSON tags

- [ ] `TaskInfo` struct (used in ActiveTasks grouping)
  - Fields: Key, Title, EpicKey, FeatureKey, AgentType (pointer)
  - Add JSON tags

- [ ] `BlockedTaskInfo` struct
  - Fields: Key, Title, EpicKey, FeatureKey, BlockedReason (pointer), Priority, BlockedAt
  - Add JSON tags

- [ ] `CompletionInfo` struct
  - Fields: Key, Title, EpicKey, FeatureKey, CompletedAt, CompletedAgo
  - Add JSON tags

- [ ] `DashboardFilter` struct
  - Fields: EpicKey (pointer), RecentWindow (pointer)
  - Add JSON tags

**Verification**:
```bash
# Verify no compilation errors
go build ./internal/status

# Verify JSON marshaling works
go test ./internal/status -run TestModels
```

### Task 1.2: Define Request/Response Types

**File**: `internal/status/models.go` (same file)

- [ ] `StatusRequest` struct
  - Fields: EpicKey, RecentWindow (default "24h"), IncludeArchived
  - Add validation method: `Validate() error`

- [ ] `StatusError` struct (for validation errors)
  - Fields: Message, Code
  - Implement `Error()` interface

- [ ] Constants for valid timeframes
  ```go
  var ValidTimeframes = map[string]bool{
      "24h": true, "1d": true,
      "7d": true, "48h": true,
      "30d": true, "90d": true,
      "90d": true,
  }
  ```

- [ ] Constants for agent type ordering
  ```go
  var AgentTypesOrder = []string{
      "frontend", "backend", "api", "testing", "devops", "general", "unassigned",
  }
  ```

**Verification**:
```bash
go test ./internal/status -run TestRequest
```

### Task 1.3: Define Service Interface

**File**: `internal/status/status.go`

- [ ] Define `StatusService` interface with methods:
  - `GetDashboard(ctx context.Context, request *StatusRequest) (*StatusDashboard, error)`
  - `GetEpicStats(ctx context.Context, epicKey string) (*EpicStats, error)`

- [ ] Document interface with comments explaining purpose

**Verification**:
```bash
grep "type StatusService interface" internal/status/status.go
```

### Task 1.4: Implement Service Constructor

**File**: `internal/status/status.go`

- [ ] Define `statusService` struct with fields:
  - `db *db.DB`
  - `epicRepo *repository.EpicRepository`
  - `featureRepo *repository.FeatureRepository`
  - `taskRepo *repository.TaskRepository`
  - `taskHistoryRepo *repository.TaskHistoryRepository`

- [ ] Create `NewStatusService` constructor function
  - Accept all repository parameters
  - Return `StatusService` interface (not concrete type)

- [ ] Add method `GetDashboard` stub (empty implementation)

**Verification**:
```bash
grep "func NewStatusService" internal/status/status.go
```

### Task 1.5: Implement Basic GetDashboard Flow

**File**: `internal/status/status.go`

- [ ] Implement `GetDashboard` method with basic flow:
  1. Validate request
  2. Initialize empty dashboard struct
  3. Call helper methods (stubs) for each section
  4. Return dashboard

- [ ] Add helper method stubs (will implement in Phase 2):
  ```go
  func (s *statusService) getProjectSummary(ctx context.Context, epicKey string) (*ProjectSummary, error)
  func (s *statusService) getEpics(ctx context.Context, epicKey string) ([]*EpicSummary, error)
  func (s *statusService) getActiveTasks(ctx context.Context, epicKey string) (map[string][]*TaskInfo, error)
  func (s *statusService) getBlockedTasks(ctx context.Context, epicKey string) ([]*BlockedTaskInfo, error)
  func (s *statusService) getRecentCompletions(ctx context.Context, epicKey string, window string) ([]*CompletionInfo, error)
  ```

**Verification**:
```bash
go build ./internal/status
```

### Task 1.6: Error Handling

**File**: `internal/status/errors.go`

- [ ] Define error types:
  ```go
  type StatusError struct {
      Message string
      Code    int
  }
  ```

- [ ] Implement `Error()` interface for StatusError

- [ ] Create helper functions:
  ```go
  func NewStatusError(message string) error
  func NewValidationError(message string) error
  func IsStatusError(err error) bool
  ```

**Verification**:
```bash
grep "func (e \*StatusError) Error" internal/status/errors.go
```

### Task 1.7: Basic Unit Tests

**File**: `internal/status/status_test.go`

- [ ] Test `NewStatusService` creates service
- [ ] Test `StatusRequest.Validate()` with valid inputs
- [ ] Test `StatusRequest.Validate()` with invalid inputs
- [ ] Test JSON marshaling of each data structure

**Verification**:
```bash
go test ./internal/status -v
```

---

## Phase 2: Database Queries

**Estimated Duration**: 2-3 hours
**Deliverable**: Optimized queries for each dashboard section

### Task 2.1: Project Summary Query

**File**: `internal/status/status.go`

In `getProjectSummary` method:

- [ ] Implement query to count epics by status
- [ ] Implement query to count features by status
- [ ] Implement query to count tasks by status
- [ ] Implement query to calculate overall progress percentage

**Specific tasks**:

- [ ] Query 1: Epic counts (total, active, completed)
  ```
  SELECT COUNT(*), status FROM epics WHERE ... GROUP BY status
  ```

- [ ] Query 2: Feature counts (total, active, completed)
  ```
  SELECT COUNT(*), status FROM features WHERE ... GROUP BY status
  ```

- [ ] Query 3: Task counts by status
  ```
  SELECT COUNT(*), status FROM tasks WHERE ... GROUP BY status
  ```

- [ ] Query 4: Overall progress percentage
  ```
  SELECT SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END) / COUNT(*)
  ```

- [ ] Build `ProjectSummary` struct from results

**Verification**:
```bash
go test ./internal/status -run TestGetProjectSummary
```

### Task 2.2: Epic Breakdown Query

**File**: `internal/status/status.go`

In `getEpics` method:

- [ ] Implement single JOIN query to get all epic metrics
  ```sql
  SELECT e.id, e.key, e.title, e.priority,
         COUNT(DISTINCT t.id) as total_tasks,
         SUM(CASE WHEN t.status='completed' THEN 1 ELSE 0 END) as completed_tasks,
         SUM(CASE WHEN t.status='blocked' THEN 1 ELSE 0 END) as blocked_tasks
  FROM epics e
  LEFT JOIN features f ON e.id = f.epic_id
  LEFT JOIN tasks t ON f.id = t.feature_id
  WHERE ...
  GROUP BY e.id, e.key, e.title, e.priority
  ORDER BY e.priority DESC, e.key ASC
  ```

- [ ] Iterate over result rows
- [ ] Calculate progress percentage for each epic
- [ ] Determine health status (healthy/warning/critical)
- [ ] Build `[]*EpicSummary` array

- [ ] Add private method:
  ```go
  func (s *statusService) determineEpicHealth(progress float64, blockedCount int) string
  ```

**Verification**:
```bash
go test ./internal/status -run TestGetEpics
```

### Task 2.3: Active Tasks Query

**File**: `internal/status/status.go`

In `getActiveTasks` method:

- [ ] Implement query for in_progress tasks with JOINs to get epic/feature keys
  ```sql
  SELECT t.key, t.title, t.agent_type, e.key, f.key
  FROM tasks t
  JOIN features f ON t.feature_id = f.id
  JOIN epics e ON f.epic_id = e.id
  WHERE t.status = 'in_progress' AND ...
  ORDER BY t.agent_type NULLS LAST, t.key ASC
  ```

- [ ] Iterate over results
- [ ] Build `[]*TaskInfo` structs
- [ ] Group by `agent_type` in map[string][]*TaskInfo
- [ ] Handle NULL agent_type as "unassigned" group

- [ ] Add private method:
  ```go
  func (s *statusService) groupTasksByAgent(tasks []*TaskInfo) map[string][]*TaskInfo
  ```

**Verification**:
```bash
go test ./internal/status -run TestGetActiveTasks
```

### Task 2.4: Blocked Tasks Query

**File**: `internal/status/status.go`

In `getBlockedTasks` method:

- [ ] Implement query for blocked tasks with reasons
  ```sql
  SELECT t.key, t.title, t.blocked_reason, t.priority, t.blocked_at, e.key, f.key
  FROM tasks t
  JOIN features f ON t.feature_id = f.id
  JOIN epics e ON f.epic_id = e.id
  WHERE t.status = 'blocked' AND ...
  ORDER BY t.priority DESC, t.blocked_at DESC NULLS LAST
  ```

- [ ] Iterate over results
- [ ] Build `[]*BlockedTaskInfo` structs
- [ ] Return sorted by priority (high first)

**Verification**:
```bash
go test ./internal/status -run TestGetBlockedTasks
```

### Task 2.5: Recent Completions Query

**File**: `internal/status/status.go`

In `getRecentCompletions` method:

- [ ] Add helper method to parse timeframe into SQLite modifier
  ```go
  func (s *statusService) timeframeToSQLiteModifier(timeframe string) string
  // "24h" -> "-24 hours"
  // "7d" -> "-7 days"
  ```

- [ ] Implement query for completed tasks since timeframe
  ```sql
  SELECT t.key, t.title, t.completed_at, e.key, f.key
  FROM tasks t
  JOIN features f ON t.feature_id = f.id
  JOIN epics e ON f.epic_id = e.id
  WHERE t.status = 'completed' AND t.completed_at >= datetime('now', ?)
  ORDER BY t.completed_at DESC
  LIMIT 100
  ```

- [ ] Iterate over results
- [ ] Build `[]*CompletionInfo` structs
- [ ] Calculate relative time ("2 hours ago") for CompletedAgo field

- [ ] Add helper method:
  ```go
  func (s *statusService) calculateRelativeTime(t time.Time) string
  // Returns strings like "2 hours ago", "5 minutes ago", "1 day ago"
  ```

**Verification**:
```bash
go test ./internal/status -run TestGetRecentCompletions
```

### Task 2.6: Epic Filtering

**File**: `internal/status/status.go`

Update all query methods to filter by epic when `epicKey != ""`:

- [ ] In `getProjectSummary`: Filter to epic's features/tasks only
- [ ] In `getEpics`: Return single epic if epicKey specified
- [ ] In `getActiveTasks`: Filter to epic's tasks
- [ ] In `getBlockedTasks`: Filter to epic's tasks
- [ ] In `getRecentCompletions`: Filter to epic's tasks

- [ ] Add validation method:
  ```go
  func (s *statusService) validateEpicKey(ctx context.Context, epicKey string) (*models.Epic, error)
  // Returns error if epic not found
  ```

**Verification**:
```bash
go test ./internal/status -run TestEpicFiltering
```

### Task 2.7: Query Performance Tests

**File**: `internal/status/status_test.go`

- [ ] Add `BenchmarkGetDashboard_SmallProject` benchmark
  - Test with 3 epics, 12 features, 127 tasks
  - Target: <50ms

- [ ] Add `BenchmarkGetDashboard_LargeProject` benchmark
  - Test with 100 epics, 500 features, 2000 tasks
  - Target: <500ms

- [ ] Add test to verify no N+1 queries
  - Use prepared query counter or similar

**Verification**:
```bash
go test ./internal/status -bench=GetDashboard -benchmem
# Verify results meet <500ms target for large project
```

---

## Phase 3: CLI Command

**Estimated Duration**: 2-3 hours
**Deliverable**: Functional `shark status` command with --json output

### Task 3.1: Create Command Definition

**File**: `internal/cli/commands/status.go`

- [ ] Create `statusCmd` Cobra command
  - Use: `status`
  - Short: `Display project dashboard`
  - Long: Full description (from architecture doc)
  - RunE: `runStatus` function

- [ ] Add command flags:
  - [ ] `--epic string` - Filter to single epic
  - [ ] `--recent string` - Time window (default "24h")
  - [ ] `--include-archived bool` - Include archived (default false)

- [ ] Register command in `init()` function:
  ```go
  func init() {
      // Define flags
      statusCmd.Flags().StringVar(&flagEpic, "epic", "", "...")
      statusCmd.Flags().StringVar(&flagRecent, "recent", "24h", "...")
      statusCmd.Flags().BoolVar(&flagArchived, "include-archived", false, "...")

      // Register command
      cli.RootCmd.AddCommand(statusCmd)
  }
  ```

**Verification**:
```bash
grep "statusCmd" internal/cli/commands/status.go
grep "func runStatus" internal/cli/commands/status.go
```

### Task 3.2: Implement Command Handler

**File**: `internal/cli/commands/status.go`

In `runStatus` function:

- [ ] Load database with error handling
  ```go
  database, err := initDatabase()
  if err != nil {
      return cli.Error(fmt.Sprintf("Failed to open database: %v", err))
  }
  defer database.Close()
  ```

- [ ] Initialize repositories
  ```go
  epicRepo := repository.NewEpicRepository(database)
  featureRepo := repository.NewFeatureRepository(database)
  taskRepo := repository.NewTaskRepository(database)
  historyRepo := repository.NewTaskHistoryRepository(database)
  ```

- [ ] Create StatusService
  ```go
  service := status.NewStatusService(database, epicRepo, featureRepo, taskRepo, historyRepo)
  ```

- [ ] Build StatusRequest from flags
  ```go
  request := &status.StatusRequest{
      EpicKey:         flagEpic,
      RecentWindow:    flagRecent,
      IncludeArchived: flagArchived,
  }
  ```

- [ ] Create context with timeout
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  ```

- [ ] Call service and handle errors
  ```go
  dashboard, err := service.GetDashboard(ctx, request)
  if err != nil {
      return cli.Error(fmt.Sprintf("Error retrieving status: %v", err))
  }
  ```

- [ ] Route to output formatter
  ```go
  if cli.GlobalConfig.JSON {
      return outputJSON(dashboard)
  } else {
      return outputRichTable(dashboard)
  }
  ```

**Verification**:
```bash
make build
./bin/shark status --json 2>&1 | jq '.' | head -20
```

### Task 3.3: JSON Output Formatter

**File**: `internal/cli/commands/status.go`

Create `outputJSON` function:

- [ ] Marshal dashboard to JSON with indentation
  ```go
  data, err := json.MarshalIndent(dashboard, "", "  ")
  if err != nil {
      return fmt.Errorf("failed to marshal JSON: %w", err)
  }
  ```

- [ ] Print to stdout
- [ ] Handle errors gracefully

**Verification**:
```bash
./bin/shark status --json
# Should output valid JSON to stdout
```

### Task 3.4: Error Handling in Command

**File**: `internal/cli/commands/status.go`

- [ ] Handle database connection errors
  ```go
  if err == sql.ErrConnDone {
      return cli.Error("Database connection lost")
  }
  ```

- [ ] Handle context timeouts
  ```go
  if errors.Is(err, context.DeadlineExceeded) {
      return cli.Error("Dashboard query timed out (>5s)")
  }
  ```

- [ ] Handle validation errors
  ```go
  if status.IsStatusError(err) {
      return cli.Error(err.Error())
  }
  ```

- [ ] Handle empty projects gracefully
  ```go
  if len(dashboard.Epics) == 0 {
      fmt.Println("No epics found. Create epics to get started.")
      return nil
  }
  ```

**Verification**:
```bash
./bin/shark status --epic=INVALID
# Should show friendly error message
```

### Task 3.5: Integration with Existing CLI

**File**: `internal/cli/commands/commands.go`

- [ ] Verify import of status command package
- [ ] Ensure `init()` functions are called (automatic with imports)

**File**: Check command registration

- [ ] Run `./bin/shark help` and verify `status` appears
- [ ] Run `./bin/shark status --help` and verify flags show

**Verification**:
```bash
./bin/shark help | grep status
./bin/shark status --help
```

### Task 3.6: Command Tests

**File**: `internal/cli/commands/status_test.go` (create new file)

- [ ] Test command with --json flag
- [ ] Test command with --epic flag
- [ ] Test command with invalid epic
- [ ] Test command with empty project
- [ ] Test JSON output validity

**Verification**:
```bash
go test ./internal/cli/commands -run Status -v
```

---

## Phase 4: Output Formatting

**Estimated Duration**: 2-3 hours
**Deliverable**: Rich formatted tables with colors and progress bars

### Task 4.1: Rich Table Formatter Setup

**File**: `internal/cli/commands/status.go`

Create main formatter function:

- [ ] Create `outputRichTable` function
  - Accepts `*status.StatusDashboard`
  - Routes to section formatters

- [ ] Import pterm library
  ```go
  "github.com/pterm/pterm"
  ```

**Verification**:
```bash
grep "import.*pterm" internal/cli/commands/status.go
```

### Task 4.2: Project Summary Section

**File**: `internal/cli/commands/status.go`

Create `outputProjectSummary` function:

- [ ] Print header: "PROJECT SUMMARY"
- [ ] Print epic counts: "Epics: 5 (3 active, 2 completed)"
- [ ] Print feature counts: "Features: 23 (15 active, 8 completed)"
- [ ] Print task counts: "Tasks: 127 (45 todo, 12 in_progress, ...)"
- [ ] Print overall progress: "Overall Progress: 47.3%"
- [ ] Print blocked count: "Blocked: 5 tasks"

**Format example**:
```
PROJECT SUMMARY
===============
Epics: 5 (3 active, 2 completed)
Features: 23 (15 active, 8 completed)
Tasks: 127 (45 todo, 12 in_progress, 5 ready_for_review, 60 completed, 5 blocked)
Overall Progress: 47.3%
Blocked: 5 tasks
```

- [ ] Use pterm.DefaultBox for framing (optional, for visual appeal)
- [ ] Handle empty projects gracefully

**Verification**:
```bash
./bin/shark status | head -20
```

### Task 4.3: Epic Breakdown Table

**File**: `internal/cli/commands/status.go`

Create `outputEpicTable` function:

- [ ] Create table with columns: Epic | Title | Progress | Tasks | Status
- [ ] For each epic:
  - [ ] Key in first column
  - [ ] Title in second column
  - [ ] Progress bar + percentage in third column
  - [ ] Task counts in fourth column (e.g., "30/50")
  - [ ] Status in fifth column

- [ ] Create `renderProgressBar` helper function
  ```go
  func renderProgressBar(progress float64, health string) string
  // Returns "[########--------] 60.0%" with colors
  ```

- [ ] Color code by health status
  - Green: health = "healthy"
  - Yellow: health = "warning"
  - Red: health = "critical"

**Progress bar format**:
- 20 characters wide
- "#" for completed, "-" for remaining
- Example: `[##########----------] 60.0%`

**Verification**:
```bash
./bin/shark status 2>&1 | grep -A 20 "EPIC BREAKDOWN"
```

### Task 4.4: Active Tasks Section

**File**: `internal/cli/commands/status.go`

Create `outputActiveTasks` function:

- [ ] Print header: "ACTIVE TASKS (12)" with count
- [ ] For each agent type in canonical order:
  - [ ] Print agent header: "Frontend (3):"
  - [ ] For each task, print: "• KEY: Title"

- [ ] Handle empty case: "No tasks currently in progress"
- [ ] Handle NULL agent_type: Group as "Unassigned"

**Format example**:
```
ACTIVE TASKS (12)
=================
Frontend (3):
  • T-E01-F02-005: Build user profile component
  • T-E01-F02-007: Implement responsive navigation
  • T-E02-F01-003: Create task list UI

Backend (5):
  • T-E01-F01-002: Implement JWT validation
  ...
```

**Verification**:
```bash
./bin/shark status 2>&1 | grep -A 30 "ACTIVE TASKS"
```

### Task 4.5: Blocked Tasks Section

**File**: `internal/cli/commands/status.go`

Create `outputBlockedTasks` function:

- [ ] Print header: "BLOCKED TASKS (5)" with count
- [ ] For each blocked task:
  - [ ] Print: "• KEY: Title" (in red)
  - [ ] Print: "Reason: [blocked_reason]"

- [ ] Color entire section red
- [ ] Handle empty case: "No blocked tasks" (in green)

**Format example**:
```
BLOCKED TASKS (5)
=================
• T-E01-F02-003: User authentication flow
  Reason: Waiting for API specification from backend team

• T-E02-F01-007: Task dependency validation
  Reason: Missing dependency graph algorithm implementation
```

**Verification**:
```bash
./bin/shark status 2>&1 | grep -A 20 "BLOCKED TASKS"
```

### Task 4.6: Recent Completions Section

**File**: `internal/cli/commands/status.go`

Create `outputRecentCompletions` function:

- [ ] Print header: "RECENT COMPLETIONS (Last 24 hours)"
- [ ] For each completion (up to 10):
  - [ ] Print: "• KEY: Title - TIME_AGO" (in green)

- [ ] Color section green
- [ ] Handle empty case: "No tasks completed in last 24 hours"
- [ ] Use CompletionInfo.CompletedAgo for time display

**Format example**:
```
RECENT COMPLETIONS (Last 24 hours)
===================================
• T-E01-F01-003: JWT token generation - 2 hours ago
• T-E01-F02-001: Login form component - 5 hours ago
• T-E02-F01-001: Database connection setup - 18 hours ago
```

**Verification**:
```bash
./bin/shark status 2>&1 | tail -20
```

### Task 4.7: Color Coding Implementation

**File**: `internal/cli/commands/status.go`

Create helper functions:

- [ ] `getHealthColor(health string) string`
  - Returns pterm color code based on health

- [ ] `colorize(text string, color string) string`
  - Wraps text in color codes

- [ ] Respect `--no-color` flag
  - Check `cli.GlobalConfig.NoColor`
  - Strip all color codes if true

**Verification**:
```bash
./bin/shark status --no-color 2>&1 | grep -v "\\[" | head -30
# Should have no ANSI color codes
```

### Task 4.8: Terminal Width Handling

**File**: `internal/cli/commands/status.go`

- [ ] Get terminal width
  ```go
  width, _, _ := terminal.GetSize(int(os.Stdout.Fd()))
  ```

- [ ] Create `truncateTitle(title string, maxWidth int) string`
  - Truncate to maxWidth-3 and add "..."

- [ ] Apply truncation to epic titles and task titles

**Verification**:
```bash
# Test in narrow terminal
./bin/shark status
# Titles should be truncated if terminal is narrow
```

---

## Phase 5: Testing & Optimization

**Estimated Duration**: 3-4 hours
**Deliverable**: Tested, optimized code meeting <500ms goal

### Task 5.1: Unit Tests

**File**: `internal/status/status_test.go`

Create comprehensive unit tests:

- [ ] Test `GetDashboard` with empty project
  - Verify zero counts, empty lists

- [ ] Test `GetDashboard` with full project data
  - Verify correct counts and aggregations

- [ ] Test epic health calculation
  - Test healthy case (>75%, no blockers)
  - Test warning case (25-75% or 1-3 blockers)
  - Test critical case (<25% or >3 blockers)

- [ ] Test task grouping by agent type
  - Verify NULL agent_type handling
  - Verify sort order

- [ ] Test recent completions filtering
  - Verify 24h window
  - Verify 7d window
  - Verify tasks outside window excluded

- [ ] Test epic filtering
  - Verify with valid epic
  - Verify error with invalid epic

**Verification**:
```bash
go test ./internal/status -v
# All tests should pass
```

### Task 5.2: Integration Tests

**File**: `internal/cli/commands/status_test.go`

Create integration tests:

- [ ] Test command execution with real database
  - Create test database with known data
  - Run command, verify output

- [ ] Test `--json` flag
  - Parse output as JSON
  - Verify structure

- [ ] Test `--epic` flag
  - Verify filtered output

- [ ] Test `--recent` flag with different timeframes

- [ ] Test `--no-color` flag
  - Verify no ANSI codes in output

- [ ] Test error cases
  - Invalid epic key
  - Database connection error
  - Timeout

**Verification**:
```bash
go test ./internal/cli/commands -run Status -v
# All tests should pass
```

### Task 5.3: Performance Testing

**File**: `internal/status/status_test.go`

- [ ] Create benchmark functions

- [ ] Benchmark with small project
  ```bash
  go test -bench=SmallProject -benchmem ./internal/status
  # Target: <50ms
  ```

- [ ] Benchmark with large project
  ```bash
  go test -bench=LargeProject -benchmem ./internal/status
  # Target: <500ms
  ```

- [ ] Profile if needed
  ```bash
  go test -cpuprofile=cpu.prof ./internal/status
  go tool pprof cpu.prof
  ```

**Verification**:
```bash
go test ./internal/status -bench=GetDashboard -benchmem
# Verify <500ms for large project
```

### Task 5.4: Optimize Query Performance

If benchmarks show queries >200ms:

- [ ] Enable query logging
- [ ] Run `EXPLAIN QUERY PLAN` on slow queries
- [ ] Verify indexes are being used
- [ ] Add missing indexes if needed

**Index verification**:
```bash
sqlite3 shark-tasks.db ".indexes"
# Verify idx_tasks_status, idx_tasks_completed_at, etc. exist
```

### Task 5.5: Memory Profiling

If memory usage is high:

- [ ] Run memory benchmark
  ```bash
  go test -memprofile=mem.prof ./internal/status
  go tool pprof mem.prof
  ```

- [ ] Look for large allocations
- [ ] Consider reusing slices if applicable

### Task 5.6: Code Coverage

- [ ] Run tests with coverage
  ```bash
  go test ./internal/status -cover
  ```

- [ ] Target >80% coverage
  ```bash
  go test ./internal/status -coverprofile=coverage.out
  go tool cover -html=coverage.out
  ```

**Verification**:
```bash
go test ./internal/status -cover
# Should show >80% coverage
```

### Task 5.7: Documentation

- [ ] Update code comments
- [ ] Document any performance trade-offs
- [ ] Document query optimization decisions
- [ ] Add examples in doc strings

**Verification**:
```bash
# Run go doc to verify comments
go doc ./internal/status | head -30
```

---

## Integration & Verification

### Task 6.1: Build & Test

- [ ] Build project
  ```bash
  make build
  ```

- [ ] Run all tests
  ```bash
  make test
  ```

- [ ] Check for race conditions
  ```bash
  go test -race ./internal/status
  go test -race ./internal/cli/commands
  ```

**Verification**:
```bash
make build && make test
# Should complete without errors
```

### Task 6.2: Manual Testing

- [ ] Test with empty database
  ```bash
  rm shark-tasks.db* 2>/dev/null
  ./bin/shark init --non-interactive
  ./bin/shark status
  ```

- [ ] Test with sample data
  ```bash
  ./bin/shark epic create "Test Epic" --priority=high
  ./bin/shark feature create --epic=E01 "Test Feature"
  ./bin/shark task create "Test Task" --epic=E01 --feature=F01
  ./bin/shark status
  ```

- [ ] Test all flags
  ```bash
  ./bin/shark status --json | jq '.' | head -30
  ./bin/shark status --epic=E01
  ./bin/shark status --no-color | head -30
  ./bin/shark status --recent=7d
  ```

- [ ] Test error cases
  ```bash
  ./bin/shark status --epic=INVALID
  ./bin/shark status --recent=invalid_window
  ```

**Verification**:
All commands should execute without crashing

### Task 6.3: Performance Verification

- [ ] Create test project with 100 epics
- [ ] Measure command execution time
  ```bash
  time ./bin/shark status > /dev/null
  # Should complete in <1 second
  ```

**Verification**:
```bash
# Command should complete quickly
time ./bin/shark status --json > /dev/null
# Should show <500ms real time
```

### Task 6.4: Cross-Check with Architecture Spec

Compare implementation against `/docs/plan/E05-task-mgmt-cli-capabilities/E05-F01-status-dashboard/04-backend-design.md`:

- [ ] Service layer design matches spec
- [ ] Data models match spec
- [ ] JSON output schema matches spec
- [ ] Query patterns use recommended optimization
- [ ] Error handling follows spec
- [ ] Performance targets met

### Task 6.5: Code Review Preparation

- [ ] Ensure code follows project conventions
  - Error handling with `fmt.Errorf("context: %w", err)`
  - Comments on public functions
  - Consistent naming

- [ ] Run linter
  ```bash
  make lint
  ```

- [ ] Format code
  ```bash
  make fmt
  ```

**Verification**:
```bash
make lint && make fmt && make build
# Should pass without warnings/errors
```

### Task 6.6: Documentation Review

- [ ] Verify all public functions documented
- [ ] Update README.md with new command
- [ ] Add command examples to docs
- [ ] Document JSON schema in docs

**Verification**:
```bash
./bin/shark status --help
# Should show all flags and examples
```

### Task 6.7: Sign-off Checklist

- [ ] All tests pass
- [ ] Code coverage >80%
- [ ] Performance <500ms for 100 epics
- [ ] Manual testing completed
- [ ] No linting errors
- [ ] No race conditions
- [ ] Documentation complete
- [ ] Ready for code review

---

## Quick Reference

### Command Line Examples

```bash
# Basic usage
./bin/shark status

# JSON output
./bin/shark status --json

# Filtered by epic
./bin/shark status --epic=E01

# Custom timeframe
./bin/shark status --recent=7d

# No colors
./bin/shark status --no-color

# Combined options
./bin/shark status --epic=E01 --json --recent=7d
```

### File Locations

- Service implementation: `internal/status/status.go`
- Data models: `internal/status/models.go`
- CLI command: `internal/cli/commands/status.go`
- Tests: `internal/status/status_test.go`, `internal/cli/commands/status_test.go`

### Testing Commands

```bash
# Unit tests
go test ./internal/status -v

# Integration tests
go test ./internal/cli/commands -run Status -v

# Benchmarks
go test -bench=GetDashboard -benchmem ./internal/status

# Coverage
go test ./internal/status -cover
go test -coverprofile=coverage.out ./internal/status
go tool cover -html=coverage.out

# Race detection
go test -race ./internal/status
go test -race ./internal/cli/commands
```

### Performance Targets

| Metric | Target |
|--------|--------|
| Query execution | <200ms |
| Output formatting | <100ms |
| Total for 100 epics | <500ms |

### Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| Slow queries | Check EXPLAIN QUERY PLAN, add indexes |
| High memory | Profile with -memprofile, look for leaks |
| Color not showing | Check pterm import, verify terminal supports colors |
| Output too wide | Implement title truncation for narrow terminals |
| Timeouts on large DB | Increase context timeout or optimize queries |
