# Bug Tracker - Implementation Roadmap

This document provides a phased implementation plan for the bug tracker feature, including task breakdown, testing strategy, and validation checkpoints.

---

## Overview

**Epic**: E10 - Bug Tracking System (Proposed)
**Estimated Complexity**: L (Large - 5-8 story points per phase)
**Total Phases**: 5
**Recommended Approach**: Incremental delivery with working software at each phase

---

## Phase 1: Core Infrastructure (Week 1)

### Goals
- Database schema in place
- Bug model with validation
- Basic repository with CRUD operations
- Repository tests passing

### Tasks

#### T1.1: Database Schema
**File**: `internal/db/db.go`

```go
// Add to createSchema()
CREATE TABLE IF NOT EXISTS bugs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    severity TEXT NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')) DEFAULT 'medium',
    priority INTEGER CHECK (priority >= 1 AND priority <= 10) DEFAULT 5,
    category TEXT,
    steps_to_reproduce TEXT,
    expected_behavior TEXT,
    actual_behavior TEXT,
    error_message TEXT,
    environment TEXT,
    os_info TEXT,
    version TEXT,
    reporter_type TEXT CHECK (reporter_type IN ('human', 'ai_agent')) DEFAULT 'human',
    reporter_id TEXT,
    detected_at TIMESTAMP NOT NULL,
    attachment_file TEXT,
    related_docs TEXT,
    related_to_epic TEXT,
    related_to_feature TEXT,
    related_to_task TEXT,
    dependencies TEXT,
    status TEXT NOT NULL CHECK (status IN ('new', 'confirmed', 'in_progress', 'resolved', 'closed', 'wont_fix', 'duplicate')) DEFAULT 'new',
    resolution TEXT,
    converted_to_type TEXT CHECK (converted_to_type IN ('task', 'epic', 'feature')),
    converted_to_key TEXT,
    converted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_bugs_key ON bugs(key);
CREATE INDEX IF NOT EXISTS idx_bugs_status ON bugs(status);
CREATE INDEX IF NOT EXISTS idx_bugs_severity ON bugs(severity);
CREATE INDEX IF NOT EXISTS idx_bugs_priority ON bugs(priority);
CREATE INDEX IF NOT EXISTS idx_bugs_detected_at ON bugs(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_bugs_category ON bugs(category);
CREATE INDEX IF NOT EXISTS idx_bugs_environment ON bugs(environment);
CREATE INDEX IF NOT EXISTS idx_bugs_related_epic ON bugs(related_to_epic);
CREATE INDEX IF NOT EXISTS idx_bugs_related_feature ON bugs(related_to_feature);
CREATE INDEX IF NOT EXISTS idx_bugs_related_task ON bugs(related_to_task);

-- Triggers
CREATE TRIGGER IF NOT EXISTS bugs_updated_at
AFTER UPDATE ON bugs
FOR EACH ROW
BEGIN
    UPDATE bugs SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS bugs_resolved_at
AFTER UPDATE ON bugs
FOR EACH ROW
WHEN NEW.status = 'resolved' AND OLD.status != 'resolved'
BEGIN
    UPDATE bugs SET resolved_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS bugs_closed_at
AFTER UPDATE ON bugs
FOR EACH ROW
WHEN NEW.status = 'closed' AND OLD.status != 'closed'
BEGIN
    UPDATE bugs SET closed_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Validation**:
```bash
# Test schema creation
./bin/shark init --non-interactive
sqlite3 shark-tasks.db ".schema bugs"
```

---

#### T1.2: Bug Model
**File**: `internal/models/bug.go`

```go
package models

import (
    "fmt"
    "regexp"
    "time"
)

type BugSeverity string
const (
    BugSeverityCritical BugSeverity = "critical"
    BugSeverityHigh     BugSeverity = "high"
    BugSeverityMedium   BugSeverity = "medium"
    BugSeverityLow      BugSeverity = "low"
)

type BugStatus string
const (
    BugStatusNew        BugStatus = "new"
    BugStatusConfirmed  BugStatus = "confirmed"
    BugStatusInProgress BugStatus = "in_progress"
    BugStatusResolved   BugStatus = "resolved"
    BugStatusClosed     BugStatus = "closed"
    BugStatusWontFix    BugStatus = "wont_fix"
    BugStatusDuplicate  BugStatus = "duplicate"
)

type ReporterType string
const (
    ReporterTypeHuman   ReporterType = "human"
    ReporterTypeAIAgent ReporterType = "ai_agent"
)

type Bug struct {
    // [Full model from comprehensive design doc]
}

func (b *Bug) Validate() error {
    // [Validation logic]
}

func ValidateBugKey(key string) error {
    pattern := `^B-\d{4}-\d{2}-\d{2}-\d{2}$`
    // [Validation logic]
}

func ValidateBugSeverity(severity string) error { /* ... */ }
func ValidateBugStatus(status string) error { /* ... */ }
func ValidateReporterType(reporterType string) error { /* ... */ }
```

**Validation**:
```go
// Add to internal/models/bug_test.go
func TestBugValidation(t *testing.T) {
    // Test valid bug
    // Test invalid key format
    // Test invalid severity
    // Test invalid status
    // Test invalid priority range
    // Test invalid JSON arrays
}
```

---

#### T1.3: Bug Repository
**File**: `internal/repository/bug_repository.go`

```go
package repository

import (
    "context"
    "database/sql"
    "github.com/jwwelbor/shark-task-manager/internal/models"
)

type BugRepository struct {
    db *DB
}

type BugFilter struct {
    Status           *models.BugStatus
    Severity         *models.BugSeverity
    Category         *string
    Environment      *string
    RelatedToEpic    *string
    RelatedToFeature *string
    RelatedToTask    *string
}

func NewBugRepository(db *DB) *BugRepository {
    return &BugRepository{db: db}
}

func (r *BugRepository) Create(ctx context.Context, bug *models.Bug) error {
    // [Implementation]
}

func (r *BugRepository) GetByID(ctx context.Context, id int64) (*models.Bug, error) {
    // [Implementation]
}

func (r *BugRepository) GetByKey(ctx context.Context, key string) (*models.Bug, error) {
    // [Implementation]
}

func (r *BugRepository) List(ctx context.Context, filter *BugFilter) ([]*models.Bug, error) {
    // [Implementation]
}

func (r *BugRepository) Update(ctx context.Context, bug *models.Bug) error {
    // [Implementation]
}

func (r *BugRepository) Delete(ctx context.Context, id int64) error {
    // [Implementation]
}

func (r *BugRepository) GetNextSequenceForDate(ctx context.Context, dateStr string) (int, error) {
    // [Implementation - similar to idea tracker]
}
```

**Validation**:
```go
// Add to internal/repository/bug_repository_test.go
func TestBugRepository_Create(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewBugRepository(db)

    // Clean up
    _, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B-TEST-%'")

    // Create bug
    bug := &models.Bug{
        Key:          "B-TEST-2026-01-04-01",
        Title:        "Test bug",
        Severity:     models.BugSeverityMedium,
        Status:       models.BugStatusNew,
        DetectedAt:   time.Now(),
        ReporterType: models.ReporterTypeHuman,
    }

    err := repo.Create(ctx, bug)
    assert.NoError(t, err)
    assert.NotZero(t, bug.ID)

    // Cleanup
    defer database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", bug.ID)
}
```

---

### Phase 1 Deliverables

- [ ] Database schema created
- [ ] Bug model implemented with validation
- [ ] Bug repository with CRUD operations
- [ ] Repository tests passing (100% coverage)
- [ ] Migration script for existing databases

**Validation Command**:
```bash
make test-db  # Should pass all bug repository tests
```

---

## Phase 2: Basic CLI Commands (Week 2)

### Goals
- `bug create` with essential flags
- `bug list` with basic filtering
- `bug get` for details
- `bug update` for modifications
- CLI tests with mocks

### Tasks

#### T2.1: Bug Create Command
**File**: `internal/cli/commands/bug.go`

```go
var bugCreateCmd = &cobra.Command{
    Use:   "create <title>",
    Short: "Create a new bug",
    Long: `Create a new bug with auto-generated key (B-YYYY-MM-DD-xx format).

Examples:
  shark bug create "Login fails"
  shark bug create "DB timeout" --severity=critical --category=database
  shark bug create "UI glitch" --steps="1. Click button\n2. Observe error"`,
    Args: cobra.ExactArgs(1),
    RunE: runBugCreate,
}

func runBugCreate(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    title := args[0]

    // Initialize database
    database, err := db.InitDB(cli.GlobalConfig.DBPath)
    if err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }
    defer database.Close()

    dbWrapper := repository.NewDB(database)
    repo := repository.NewBugRepository(dbWrapper)

    // Generate bug key
    bugKey, err := generateBugKey(ctx, repo)
    if err != nil {
        return fmt.Errorf("failed to generate bug key: %w", err)
    }

    // Build bug
    bug := &models.Bug{
        Key:          bugKey,
        Title:        title,
        DetectedAt:   time.Now(),
        Severity:     models.BugSeverity(bugSeverity),
        Status:       models.BugStatusNew,
        ReporterType: models.ReporterType(bugReporterType),
    }

    // Set optional fields
    if bugDescription != "" {
        bug.Description = &bugDescription
    }
    // [... set other fields ...]

    // Create bug
    if err := repo.Create(ctx, bug); err != nil {
        return fmt.Errorf("failed to create bug: %w", err)
    }

    // Output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bug)
    }

    cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
    return nil
}

func generateBugKey(ctx context.Context, repo BugRepository) (string, error) {
    now := time.Now()
    dateStr := now.Format("2006-01-02")
    baseKey := fmt.Sprintf("B-%s", dateStr)

    nextSeq, err := repo.GetNextSequenceForDate(ctx, dateStr)
    if err != nil {
        return "", err
    }

    return fmt.Sprintf("%s-%02d", baseKey, nextSeq), nil
}
```

**Flags**:
```go
var (
    bugDescription     string
    bugSeverity        string
    bugPriority        int
    bugCategory        string
    bugSteps           string
    bugExpected        string
    bugActual          string
    bugError           string
    bugEnvironment     string
    bugOS              string
    bugVersion         string
    bugReporter        string
    bugReporterType    string
    bugAttachmentFile  string
    bugRelatedDocs     []string
    bugRelatedEpic     string
    bugRelatedFeature  string
    bugRelatedTask     string
    bugDependsOn       []string
)

func init() {
    bugCreateCmd.Flags().StringVar(&bugDescription, "description", "", "Bug description")
    bugCreateCmd.Flags().StringVar(&bugSeverity, "severity", "medium", "Severity (critical, high, medium, low)")
    bugCreateCmd.Flags().IntVar(&bugPriority, "priority", 5, "Priority (1-10)")
    bugCreateCmd.Flags().StringVar(&bugCategory, "category", "", "Category (backend, frontend, database, etc.)")
    bugCreateCmd.Flags().StringVar(&bugSteps, "steps", "", "Steps to reproduce")
    bugCreateCmd.Flags().StringVar(&bugExpected, "expected", "", "Expected behavior")
    bugCreateCmd.Flags().StringVar(&bugActual, "actual", "", "Actual behavior")
    bugCreateCmd.Flags().StringVar(&bugError, "error", "", "Error message or stack trace")
    bugCreateCmd.Flags().StringVar(&bugEnvironment, "environment", "", "Environment (production, staging, development, test)")
    bugCreateCmd.Flags().StringVar(&bugOS, "os", "", "Operating system info")
    bugCreateCmd.Flags().StringVar(&bugVersion, "version", "", "Software version")
    bugCreateCmd.Flags().StringVar(&bugReporter, "reporter", "", "Reporter ID")
    bugCreateCmd.Flags().StringVar(&bugReporterType, "reporter-type", "human", "Reporter type (human, ai_agent)")
    bugCreateCmd.Flags().StringVar(&bugAttachmentFile, "file", "", "Attachment file path")
    bugCreateCmd.Flags().StringSliceVar(&bugRelatedDocs, "related-docs", []string{}, "Related document paths")
    bugCreateCmd.Flags().StringVar(&bugRelatedEpic, "epic", "", "Related epic key")
    bugCreateCmd.Flags().StringVar(&bugRelatedFeature, "feature", "", "Related feature key")
    bugCreateCmd.Flags().StringVar(&bugRelatedTask, "task", "", "Related task key")
    bugCreateCmd.Flags().StringSliceVar(&bugDependsOn, "depends-on", []string{}, "Dependencies (bug keys)")
}
```

**Validation**:
```bash
# Manual test
./bin/shark bug create "Test bug" --severity=high --category=backend

# Expected output:
# Created bug B-2026-01-04-01: Test bug
```

---

#### T2.2: Bug List Command
**File**: `internal/cli/commands/bug.go`

```go
var bugListCmd = &cobra.Command{
    Use:   "list",
    Short: "List bugs",
    Long: `List bugs with optional filtering.

Examples:
  shark bug list
  shark bug list --severity=critical
  shark bug list --status=new --category=backend`,
    RunE: runBugList,
}

func runBugList(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Initialize database
    database, err := db.InitDB(cli.GlobalConfig.DBPath)
    if err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }
    defer database.Close()

    dbWrapper := repository.NewDB(database)
    repo := repository.NewBugRepository(dbWrapper)

    // Build filter
    filter := &repository.BugFilter{}
    if bugStatus != "" {
        status := models.BugStatus(bugStatus)
        filter.Status = &status
    }
    if bugSeverity != "" {
        severity := models.BugSeverity(bugSeverity)
        filter.Severity = &severity
    }
    if bugCategory != "" {
        filter.Category = &bugCategory
    }
    // [... other filters ...]

    // Get bugs
    bugs, err := repo.List(ctx, filter)
    if err != nil {
        return fmt.Errorf("failed to list bugs: %w", err)
    }

    // Output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bugs)
    }

    // Table output
    if len(bugs) == 0 {
        fmt.Println("No bugs found")
        return nil
    }

    headers := []string{"Key", "Title", "Severity", "Status", "Category", "Detected"}
    rows := make([][]string, len(bugs))
    for i, bug := range bugs {
        category := "-"
        if bug.Category != nil {
            category = *bug.Category
        }
        rows[i] = []string{
            bug.Key,
            bug.Title,
            string(bug.Severity),
            string(bug.Status),
            category,
            bug.DetectedAt.Format("2006-01-02"),
        }
    }

    cli.OutputTable(headers, rows)
    return nil
}
```

---

#### T2.3: Bug Get Command
**File**: `internal/cli/commands/bug.go`

```go
var bugGetCmd = &cobra.Command{
    Use:   "get <bug-key>",
    Short: "Get bug details",
    Args:  cobra.ExactArgs(1),
    RunE:  runBugGet,
}

func runBugGet(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    bugKey := args[0]

    // Initialize database
    database, err := db.InitDB(cli.GlobalConfig.DBPath)
    if err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }
    defer database.Close()

    dbWrapper := repository.NewDB(database)
    repo := repository.NewBugRepository(dbWrapper)

    // Get bug
    bug, err := repo.GetByKey(ctx, bugKey)
    if err != nil {
        return fmt.Errorf("failed to get bug: %w", err)
    }

    // Output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bug)
    }

    // Detailed text output
    fmt.Printf("Bug: %s\n", bug.Key)
    fmt.Printf("Title: %s\n", bug.Title)
    fmt.Printf("Status: %s\n", bug.Status)
    fmt.Printf("Severity: %s\n", bug.Severity)
    // [... print all fields ...]

    return nil
}
```

---

### Phase 2 Deliverables

- [ ] `bug create` command working
- [ ] `bug list` command with filters
- [ ] `bug get` command
- [ ] `bug update` command
- [ ] CLI tests with mocks passing
- [ ] Help text for all commands

**Validation Commands**:
```bash
# Create bugs
./bin/shark bug create "Test 1" --severity=critical
./bin/shark bug create "Test 2" --severity=high --category=backend

# List bugs
./bin/shark bug list
./bin/shark bug list --severity=critical
./bin/shark bug list --json

# Get bug
./bin/shark bug get B-2026-01-04-01

# Update bug
./bin/shark bug update B-2026-01-04-01 --status=confirmed
```

---

## Phase 3: Status Management (Week 3)

### Goals
- `bug confirm` command
- `bug resolve` command
- `bug close` command
- `bug reopen` command
- `bug delete` command
- Status transition validation

### Tasks

#### T3.1: Status Transition Commands
**File**: `internal/cli/commands/bug.go`

```go
var bugConfirmCmd = &cobra.Command{
    Use:   "confirm <bug-key>",
    Short: "Confirm a bug",
    Args:  cobra.ExactArgs(1),
    RunE:  runBugConfirm,
}

func runBugConfirm(cmd *cobra.Command, args []string) error {
    return updateBugStatus(args[0], models.BugStatusConfirmed, bugNotes)
}

var bugResolveCmd = &cobra.Command{
    Use:   "resolve <bug-key>",
    Short: "Mark bug as resolved",
    Args:  cobra.ExactArgs(1),
    RunE:  runBugResolve,
}

func runBugResolve(cmd *cobra.Command, args []string) error {
    return updateBugStatusWithResolution(args[0], models.BugStatusResolved, bugResolution)
}

// Helper functions
func updateBugStatus(bugKey string, status models.BugStatus, notes *string) error {
    ctx := context.Background()

    database, err := db.InitDB(cli.GlobalConfig.DBPath)
    if err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }
    defer database.Close()

    dbWrapper := repository.NewDB(database)
    repo := repository.NewBugRepository(dbWrapper)

    // Get bug
    bug, err := repo.GetByKey(ctx, bugKey)
    if err != nil {
        return fmt.Errorf("failed to get bug: %w", err)
    }

    // Update status
    bug.Status = status
    if notes != nil && *notes != "" {
        // Append notes to resolution field
        if bug.Resolution == nil {
            bug.Resolution = notes
        } else {
            updated := fmt.Sprintf("%s\n\n%s", *bug.Resolution, *notes)
            bug.Resolution = &updated
        }
    }

    if err := repo.Update(ctx, bug); err != nil {
        return fmt.Errorf("failed to update bug: %w", err)
    }

    // Output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(bug)
    }

    cli.Success(fmt.Sprintf("Bug %s marked as %s", bug.Key, status))
    return nil
}
```

---

### Phase 3 Deliverables

- [ ] `bug confirm` command
- [ ] `bug resolve` command
- [ ] `bug close` command
- [ ] `bug reopen` command
- [ ] `bug delete` command (soft/hard)
- [ ] Status transition tests
- [ ] Timestamp trigger validation

**Validation Commands**:
```bash
# Create bug
./bin/shark bug create "Status test"

# Confirm
./bin/shark bug confirm B-2026-01-04-01 --notes="Reproduced"

# Resolve
./bin/shark bug resolve B-2026-01-04-01 --resolution="Fixed in commit abc123"

# Close
./bin/shark bug close B-2026-01-04-01 --notes="Verified"

# Reopen
./bin/shark bug reopen B-2026-01-04-01 --notes="Bug still occurs"

# Delete
./bin/shark bug delete B-2026-01-04-01 --force
```

---

## Phase 4: Conversion & Integration (Week 4)

### Goals
- `bug convert task` command
- `bug convert feature` command
- `bug convert epic` command
- Conversion tracking
- Integration with task/feature/epic repositories

### Tasks

#### T4.1: Conversion Commands
**File**: `internal/cli/commands/bug.go`

```go
var bugConvertTaskCmd = &cobra.Command{
    Use:   "task <bug-key> --epic=<epic> --feature=<feature>",
    Short: "Convert bug to task",
    Args:  cobra.ExactArgs(1),
    RunE:  runBugConvertTask,
}

func runBugConvertTask(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    bugKey := args[0]

    // Initialize database
    database, err := db.InitDB(cli.GlobalConfig.DBPath)
    if err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }
    defer database.Close()

    dbWrapper := repository.NewDB(database)
    bugRepo := repository.NewBugRepository(dbWrapper)
    epicRepo := repository.NewEpicRepository(dbWrapper)
    featureRepo := repository.NewFeatureRepository(dbWrapper)
    taskRepo := repository.NewTaskRepository(dbWrapper)

    // Get bug
    bug, err := bugRepo.GetByKey(ctx, bugKey)
    if err != nil {
        return fmt.Errorf("failed to get bug: %w", err)
    }

    // Check if already converted
    if bug.Status == models.BugStatusConverted {
        return fmt.Errorf("bug %s already converted to %s %s",
            bug.Key, *bug.ConvertedToType, *bug.ConvertedToKey)
    }

    // Get epic and feature
    epic, err := epicRepo.GetByKey(ctx, bugConvertEpic)
    if err != nil {
        return fmt.Errorf("failed to get epic: %w", err)
    }

    feature, err := featureRepo.GetByKey(ctx, bugConvertFeature)
    if err != nil {
        return fmt.Errorf("failed to get feature: %w", err)
    }

    // Verify feature belongs to epic
    if feature.EpicID != epic.ID {
        return fmt.Errorf("feature %s does not belong to epic %s", feature.Key, epic.Key)
    }

    // Generate task key
    kg := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
    taskKey, err := kg.GenerateTaskKey(ctx, epic.Key, feature.Key)
    if err != nil {
        return fmt.Errorf("failed to generate task key: %w", err)
    }

    // Create task from bug
    task := &models.Task{
        FeatureID:   feature.ID,
        Key:         taskKey,
        Title:       bug.Title,
        Description: bug.Description,
        Status:      "todo",
        Priority:    5,
    }

    // Copy priority if set
    if bug.Priority != nil {
        task.Priority = *bug.Priority
    }

    // Build comprehensive description
    var descParts []string
    if bug.Description != nil {
        descParts = append(descParts, *bug.Description)
    }
    descParts = append(descParts, fmt.Sprintf("\n## Original Bug: %s", bug.Key))
    if bug.StepsToReproduce != nil {
        descParts = append(descParts, fmt.Sprintf("\n### Steps to Reproduce\n%s", *bug.StepsToReproduce))
    }
    if bug.ExpectedBehavior != nil {
        descParts = append(descParts, fmt.Sprintf("\n### Expected Behavior\n%s", *bug.ExpectedBehavior))
    }
    if bug.ActualBehavior != nil {
        descParts = append(descParts, fmt.Sprintf("\n### Actual Behavior\n%s", *bug.ActualBehavior))
    }
    if bug.ErrorMessage != nil {
        descParts = append(descParts, fmt.Sprintf("\n### Error Message\n```\n%s\n```", *bug.ErrorMessage))
    }
    fullDesc := strings.Join(descParts, "\n")
    task.Description = &fullDesc

    // Create task
    if err := taskRepo.Create(ctx, task); err != nil {
        return fmt.Errorf("failed to create task: %w", err)
    }

    // Mark bug as converted
    if err := bugRepo.MarkAsConverted(ctx, bug.ID, "task", task.Key); err != nil {
        return fmt.Errorf("failed to mark bug as converted: %w", err)
    }

    // Output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "bug_key":      bugKey,
            "converted_to": task.Key,
            "type":         "task",
        })
    }

    cli.Success(fmt.Sprintf("Bug %s converted to task %s", bugKey, task.Key))
    return nil
}
```

---

### Phase 4 Deliverables

- [ ] `bug convert task` command
- [ ] `bug convert feature` command
- [ ] `bug convert epic` command
- [ ] Conversion tracking in database
- [ ] Task description includes bug context
- [ ] Conversion tests

**Validation Commands**:
```bash
# Create bug
./bin/shark bug create "Conversion test" --severity=high

# Convert to task
./bin/shark bug convert task B-2026-01-04-01 --epic=E07 --feature=E07-F20

# Verify conversion
./bin/shark bug get B-2026-01-04-01
# Should show: Converted to: task T-E07-F20-025

# Verify task created
./bin/shark task get T-E07-F20-025
# Description should include bug details
```

---

## Phase 5: Documentation & Polish (Week 5)

### Goals
- CLI help text refinement
- Usage examples in documentation
- Error message improvements
- Performance optimization
- Integration testing

### Tasks

#### T5.1: Documentation
- Update `docs/CLI_REFERENCE.md` with bug commands
- Create `docs/BUG_TRACKER_GUIDE.md` with user guide
- Add examples to command help text
- Update `README.md` with bug tracker feature

#### T5.2: Error Message Refinement
- Improve validation error messages
- Add helpful suggestions in error output
- Test error handling for edge cases

#### T5.3: Performance Optimization
- Add database indexes based on query patterns
- Optimize list filtering queries
- Profile repository operations

#### T5.4: Integration Testing
- End-to-end workflow tests
- AI agent integration examples
- CI/CD integration scripts

---

### Phase 5 Deliverables

- [ ] Complete documentation
- [ ] Refined error messages
- [ ] Performance benchmarks
- [ ] Integration examples
- [ ] User guide

**Validation**:
```bash
# Help text
./bin/shark bug --help
./bin/shark bug create --help
./bin/shark bug convert --help

# Integration test
./scripts/test-bug-tracker-workflow.sh
```

---

## Testing Strategy

### Repository Tests (Real Database)

**Pattern**:
```go
func TestBugRepository_XXX(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewBugRepository(db)

    // CRITICAL: Clean up before test
    _, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B-TEST-%'")

    // Test logic
    // ...

    // Cleanup after test
    defer database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", bug.ID)
}
```

**Coverage Goals**:
- Create: 100%
- Read (GetByID, GetByKey): 100%
- Update: 100%
- Delete: 100%
- List with filters: 100%
- Conversion tracking: 100%

---

### CLI Tests (Mocks Only)

**Pattern**:
```go
type MockBugRepository struct {
    CreateFunc       func(ctx context.Context, bug *models.Bug) error
    GetByKeyFunc     func(ctx context.Context, key string) (*models.Bug, error)
    ListFunc         func(ctx context.Context, filter *BugFilter) ([]*models.Bug, error)
    UpdateFunc       func(ctx context.Context, bug *models.Bug) error
    DeleteFunc       func(ctx context.Context, id int64) error
    MarkAsConvertedFunc func(ctx context.Context, bugID int64, convertedToType, convertedToKey string) error
}

func TestBugCreateCommand(t *testing.T) {
    mockRepo := &MockBugRepository{
        CreateFunc: func(ctx context.Context, bug *models.Bug) error {
            bug.ID = 123
            return nil
        },
    }

    // Test command with mock
    // Verify correct repo calls
    // Verify output format
}
```

**Coverage Goals**:
- All commands: 100%
- JSON output: 100%
- Error handling: 100%
- Flag parsing: 100%

---

### Integration Tests

**End-to-End Workflows**:
```bash
#!/bin/bash
# test-bug-tracker-workflow.sh

set -e

echo "=== Bug Tracker Integration Test ==="

# Phase 1: AI Agent Reports Bug
echo "Phase 1: Creating bug as AI agent..."
BUG_KEY=$(./bin/shark bug create "Test integration bug" \
  --severity=critical \
  --category=test \
  --reporter=test-agent \
  --reporter-type=ai_agent \
  --json | jq -r '.key')

echo "Created bug: $BUG_KEY"

# Phase 2: Human Confirms Bug
echo "Phase 2: Confirming bug..."
./bin/shark bug confirm "$BUG_KEY" --notes="Reproduced in test environment"

# Phase 3: Convert to Task
echo "Phase 3: Converting to task..."
TASK_KEY=$(./bin/shark bug convert task "$BUG_KEY" --epic=E07 --feature=E07-F20 --json | jq -r '.converted_to')

echo "Converted to task: $TASK_KEY"

# Phase 4: Verify Conversion
echo "Phase 4: Verifying conversion..."
CONVERTED_TYPE=$(./bin/shark bug get "$BUG_KEY" --json | jq -r '.converted_to_type')

if [ "$CONVERTED_TYPE" != "task" ]; then
  echo "ERROR: Conversion failed"
  exit 1
fi

echo "✓ Integration test passed"
```

---

## Success Criteria

### Phase 1
- [ ] Schema created in database
- [ ] All repository tests passing
- [ ] No database constraint violations

### Phase 2
- [ ] Can create bugs via CLI
- [ ] Can list bugs with filters
- [ ] Can get bug details
- [ ] Can update bugs
- [ ] All CLI tests passing

### Phase 3
- [ ] Can transition bug statuses
- [ ] Timestamps set correctly (resolved_at, closed_at)
- [ ] Can delete bugs (soft/hard)
- [ ] Status workflow enforced

### Phase 4
- [ ] Can convert bugs to tasks
- [ ] Can convert bugs to features
- [ ] Can convert bugs to epics
- [ ] Task description includes bug context
- [ ] Conversion tracking accurate

### Phase 5
- [ ] Documentation complete
- [ ] Help text clear and helpful
- [ ] Error messages actionable
- [ ] Performance acceptable (<100ms for list, <10ms for get)
- [ ] Integration examples working

---

## Risk Mitigation

### Risk: Schema Migration Conflicts
**Mitigation**: Test migration on copy of production database first

### Risk: Performance Degradation with Large Bug Count
**Mitigation**: Add indexes early, benchmark with 10k+ bugs

### Risk: CLI Flag Complexity
**Mitigation**: Use sensible defaults, provide examples in help text

### Risk: Conversion Logic Bugs
**Mitigation**: Extensive testing of conversion workflows, rollback mechanism

---

## Rollout Plan

### Week 1: Internal Alpha
- Deploy to dev environment
- Internal team testing
- Gather feedback

### Week 2: Beta
- Deploy to staging
- Select users test bug reporting
- Monitor for issues

### Week 3: General Availability
- Deploy to production
- Announce feature
- Monitor usage metrics

---

## Metrics to Track

Post-launch metrics:
- Bugs created per day
- Average time to resolution (detected_at → resolved_at)
- Conversion rate (bugs converted to tasks)
- AI agent vs human reporter ratio
- Severity distribution
- Category distribution

---

## Future Enhancements (Post-Launch)

1. **Bug Analytics Dashboard**
   ```bash
   shark bug stats --epic=E07
   shark bug metrics --metric=resolution-time
   ```

2. **Bug Templates**
   ```bash
   shark bug create --template=crash "App crashes"
   ```

3. **Binary Attachments**
   ```bash
   shark bug attach B-2026-01-04-01 screenshot.png
   ```

4. **Email Notifications**
   ```bash
   shark bug notify B-2026-01-04-01 --to=team@example.com
   ```

5. **Bug Board View**
   ```bash
   shark bug board --epic=E07
   ```

---

**End of Roadmap**
