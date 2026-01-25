# Shark View Command - Technical Design

## Overview

This document describes the design for adding a `shark view` command that opens specification files in an external viewer. The solution follows DRY and SOLID principles and includes a reusable scope interpreter component.

## Requirements

1. Add optional "viewer" setting in config to specify external tool (glow, nano, or default to cat)
2. Implement `shark view` command that launches the external viewer with filepath of specification files
3. Dynamic behavior based on amount of information provided:
   - If only epic specified: show epic spec
   - If epic + feature: show feature spec
   - If epic + feature + task: show task spec
4. Support multiple CLI formats:
   - `shark view E01`
   - `shark view E01 F01`
   - `shark view E01-F01`
   - `shark view --epic=E01 --feature=F01`
   - `shark view E01 F01 001`
   - `shark view T-E01-F01-001`
5. Solution should be DRY and SOLID
6. Extract a reusable scope interpreter that determines if we are operating on an epic, feature, or task

## Current State Analysis

### Existing Patterns

**Argument Parsing:**
- The `shark get` command already implements similar dynamic behavior
- Uses `ParseGetArgs()` in `/internal/cli/commands/helpers.go`
- Returns `(command, key, error)` where command is "epic", "feature", or "task"

**File Path Resolution:**
- Epic, Feature, and Task models have `file_path` field stored in database
- Repository queries can retrieve file paths via `SELECT file_path FROM {table} WHERE key = ?`

**Config Management:**
- Config loaded from `.sharkconfig.json` via `/internal/config/config.go`
- Config struct has extensible `RawData map[string]interface{}` for unknown fields
- Can add new `Viewer *string` field to Config struct

### Reusable Components

The existing `ParseGetArgs()` function already implements scope interpretation logic but is not extracted as a reusable component. We can extract this into a dedicated scope interpreter.

## Architecture Design

### Component Overview

```
┌─────────────────────────────────────────────────────┐
│                 shark view E01 F01                  │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│          CLI Command Layer (view.go)                │
│  • Parse flags                                      │
│  • Call ScopeInterpreter                            │
│  • Call ViewService                                 │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│    ScopeInterpreter (internal/cli/scope/)           │
│  • ParseScope(args) -> (ScopeType, key)             │
│  • Reusable across commands                         │
│  • SINGLE RESPONSIBILITY: Determine scope from CLI  │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│       ViewService (internal/view/service.go)        │
│  • GetFilePath(scope, key) -> filepath              │
│  • LaunchViewer(filepath, viewerCmd)                │
│  • SINGLE RESPONSIBILITY: View operations           │
└────────────────────┬────────────────────────────────┘
                     │
         ┌───────────┴───────────┐
         ▼                       ▼
┌─────────────────┐    ┌─────────────────────┐
│   Repository    │    │  Config             │
│   (file_path    │    │  (viewer setting)   │
│    queries)     │    │                     │
└─────────────────┘    └─────────────────────┘
```

## Component Design

### 1. ScopeInterpreter (NEW - Reusable)

**Location:** `/internal/cli/scope/interpreter.go`

**Purpose:** Determine the scope (epic, feature, or task) from CLI arguments. This is extracted as a standalone package to be reusable across multiple commands (get, view, list, etc.)

**Interface:**

```go
package scope

// ScopeType represents the type of entity being referenced
type ScopeType string

const (
    ScopeEpic    ScopeType = "epic"
    ScopeFeature ScopeType = "feature"
    ScopeTask    ScopeType = "task"
)

// Scope represents a parsed scope from CLI arguments
type Scope struct {
    Type ScopeType
    Key  string  // Normalized key (E01, E01-F01, or T-E01-F01-001)
}

// Interpreter parses CLI arguments to determine scope
type Interpreter struct{}

// NewInterpreter creates a new scope interpreter
func NewInterpreter() *Interpreter {
    return &Interpreter{}
}

// ParseScope parses CLI arguments and returns the scope
// Examples:
//   ParseScope(["E01"]) -> (ScopeEpic, "E01", nil)
//   ParseScope(["E01", "F01"]) -> (ScopeFeature, "E01-F01", nil)
//   ParseScope(["E01-F01"]) -> (ScopeFeature, "E01-F01", nil)
//   ParseScope(["T-E01-F01-001"]) -> (ScopeTask, "T-E01-F01-001", nil)
//   ParseScope(["E01", "F01", "001"]) -> (ScopeTask, "T-E01-F01-001", nil)
func (i *Interpreter) ParseScope(args []string) (*Scope, error) {
    // Implementation leverages existing ParseGetArgs logic
    // but returns a Scope struct instead of (command, key, error)
}
```

**Implementation Strategy:**

1. Extract logic from `ParseGetArgs()` in `/internal/cli/commands/helpers.go`
2. Reuse existing helper functions: `IsEpicKey()`, `IsFeatureKey()`, `IsFeatureKeySuffix()`, `NormalizeKey()`, etc.
3. Keep backward compatibility by making `ParseGetArgs()` call `ParseScope()` internally

**Benefits:**
- SINGLE RESPONSIBILITY: Only concerned with scope determination
- OPEN/CLOSED: Can extend with new scope types without modifying existing code
- DEPENDENCY INVERSION: Commands depend on abstraction (Interpreter interface), not concrete implementation
- REUSABLE: Can be used by `get`, `view`, `list`, and future commands

### 2. ViewService (NEW)

**Location:** `/internal/view/service.go`

**Purpose:** Handle viewing operations (file path resolution and viewer launching)

**Interface:**

```go
package view

import (
    "context"
    "github.com/jwwelbor/shark-task-manager/internal/cli/scope"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

// Service handles viewing specification files
type Service struct {
    epicRepo    *repository.EpicRepository
    featureRepo *repository.FeatureRepository
    taskRepo    *repository.TaskRepository
}

// NewService creates a new ViewService
func NewService(
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
    taskRepo *repository.TaskRepository,
) *Service {
    return &Service{
        epicRepo:    epicRepo,
        featureRepo: featureRepo,
        taskRepo:    taskRepo,
    }
}

// GetFilePath retrieves the file path for a given scope
func (s *Service) GetFilePath(ctx context.Context, scope *scope.Scope) (string, error) {
    switch scope.Type {
    case scope.ScopeEpic:
        epic, err := s.epicRepo.GetByKey(ctx, scope.Key)
        if err != nil {
            return "", fmt.Errorf("epic not found: %w", err)
        }
        return epic.FilePath, nil

    case scope.ScopeFeature:
        feature, err := s.featureRepo.GetByKey(ctx, scope.Key)
        if err != nil {
            return "", fmt.Errorf("feature not found: %w", err)
        }
        return feature.FilePath, nil

    case scope.ScopeTask:
        task, err := s.taskRepo.GetByKey(ctx, scope.Key)
        if err != nil {
            return "", fmt.Errorf("task not found: %w", err)
        }
        return task.FilePath, nil

    default:
        return "", fmt.Errorf("unknown scope type: %s", scope.Type)
    }
}

// LaunchViewer opens the file in the specified viewer
func (s *Service) LaunchViewer(ctx context.Context, filePath string, viewerCmd string) error {
    // Validate file exists
    // Construct command (viewerCmd + filePath)
    // Execute using os/exec
    // Return error if viewer fails
}
```

**Benefits:**
- SINGLE RESPONSIBILITY: Only concerned with viewing operations
- DEPENDENCY INJECTION: Repositories injected via constructor
- TESTABLE: Can mock repositories and test file path resolution separately from viewer launching

### 3. Config Extension

**Location:** `/internal/config/config.go`

**Changes:**

```go
type Config struct {
    // ... existing fields ...

    // Viewer specifies the external tool to use for viewing files
    // Examples: "glow", "nano", "cat", "bat", "less"
    // Default: "cat" if not specified
    Viewer *string `json:"viewer,omitempty"`
}

// GetViewer returns the configured viewer or default "cat"
func (c *Config) GetViewer() string {
    if c.Viewer != nil && *c.Viewer != "" {
        return *c.Viewer
    }
    return "cat"
}
```

**Example `.sharkconfig.json`:**

```json
{
  "viewer": "glow",
  "color_enabled": true,
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}
```

### 4. View Command

**Location:** `/internal/cli/commands/view.go`

**Implementation:**

```go
package commands

import (
    "context"
    "fmt"
    "github.com/jwwelbor/shark-task-manager/internal/cli"
    "github.com/jwwelbor/shark-task-manager/internal/cli/scope"
    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
    "github.com/jwwelbor/shark-task-manager/internal/view"
    "github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
    Use:     "view <KEY>",
    Short:   "View epic, feature, or task specification in external viewer",
    GroupID: "essentials",
    Long: `View specification files in an external viewer (glow, nano, cat, etc.)

The viewer can be configured in .sharkconfig.json:
{
  "viewer": "glow"
}

Defaults to "cat" if not configured.

Positional Arguments:
  EPIC                  View epic spec (e.g., E04)
  EPIC FEATURE          View feature spec (e.g., E04 F01 or E04-F01)
  EPIC FEATURE TASKNUM  View task spec (e.g., E04 F01 001)
  FULL_TASK_KEY         View task spec (e.g., T-E04-F01-001)

Examples:
  shark view E10                    View epic E10 spec
  shark view E10 F01                View feature E10-F01 spec
  shark view E10-F01                View feature E10-F01 spec (combined)
  shark view E10 F01 001            View task T-E10-F01-001 spec
  shark view T-E10-F01-001          View task spec (full key)
`,
    RunE: runView,
}

func init() {
    cli.RootCmd.AddCommand(viewCmd)
}

func runView(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Get database
    repoDb, err := cli.GetDB(ctx)
    if err != nil {
        return fmt.Errorf("failed to get database: %w", err)
    }

    // Load config for viewer setting
    cfg, err := config.LoadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Parse scope from arguments
    interpreter := scope.NewInterpreter()
    parsedScope, err := interpreter.ParseScope(args)
    if err != nil {
        return err
    }

    // Create repositories
    epicRepo := repository.NewEpicRepository(repoDb)
    featureRepo := repository.NewFeatureRepository(repoDb)
    taskRepo := repository.NewTaskRepository(repoDb)

    // Create view service
    viewService := view.NewService(epicRepo, featureRepo, taskRepo)

    // Get file path
    filePath, err := viewService.GetFilePath(ctx, parsedScope)
    if err != nil {
        return fmt.Errorf("failed to get file path: %w", err)
    }

    // Launch viewer
    viewerCmd := cfg.GetViewer()
    if err := viewService.LaunchViewer(ctx, filePath, viewerCmd); err != nil {
        return fmt.Errorf("failed to launch viewer: %w", err)
    }

    return nil
}
```

## File Structure

```
shark-task-manager/
├── internal/
│   ├── cli/
│   │   ├── commands/
│   │   │   ├── view.go                    # NEW: View command
│   │   │   ├── view_test.go               # NEW: View command tests
│   │   │   ├── get.go                     # MODIFIED: Use scope.Interpreter
│   │   │   └── helpers.go                 # MODIFIED: Refactor to use scope package
│   │   └── scope/                         # NEW: Reusable scope interpreter
│   │       ├── interpreter.go             # NEW: Scope parsing logic
│   │       ├── interpreter_test.go        # NEW: Scope interpreter tests
│   │       └── types.go                   # NEW: ScopeType, Scope struct
│   ├── view/                              # NEW: View service package
│   │   ├── service.go                     # NEW: View service implementation
│   │   ├── service_test.go                # NEW: View service tests
│   │   └── launcher.go                    # NEW: Viewer launcher logic
│   └── config/
│       └── config.go                      # MODIFIED: Add Viewer field
└── docs/
    └── design/
        └── view-command-design.md         # This document
```

## Integration Points

### 1. Refactor `get` Command

**Current State:** `ParseGetArgs()` in `/internal/cli/commands/helpers.go`

**Refactored:**

```go
// ParseGetArgs parses arguments for get command
// Now delegates to scope.Interpreter for DRY
func ParseGetArgs(args []string) (command string, key string, err error) {
    interpreter := scope.NewInterpreter()
    parsedScope, err := interpreter.ParseScope(args)
    if err != nil {
        return "", "", err
    }

    return string(parsedScope.Type), parsedScope.Key, nil
}
```

### 2. Future Commands Can Reuse ScopeInterpreter

Any command that needs to determine epic/feature/task scope can use:

```go
interpreter := scope.NewInterpreter()
parsedScope, err := interpreter.ParseScope(args)
```

Examples:
- `shark edit E01 F01` - Open spec in editor
- `shark validate E01` - Validate all specs in epic
- `shark export E01-F01` - Export feature to different format

## Testing Strategy

### Unit Tests

**1. ScopeInterpreter Tests (`internal/cli/scope/interpreter_test.go`)**

```go
func TestInterpreter_ParseScope(t *testing.T) {
    tests := []struct {
        name      string
        args      []string
        wantType  ScopeType
        wantKey   string
        wantError bool
    }{
        {
            name:     "epic scope",
            args:     []string{"E01"},
            wantType: ScopeEpic,
            wantKey:  "E01",
        },
        {
            name:     "feature scope - combined",
            args:     []string{"E01-F01"},
            wantType: ScopeFeature,
            wantKey:  "E01-F01",
        },
        {
            name:     "feature scope - separate args",
            args:     []string{"E01", "F01"},
            wantType: ScopeFeature,
            wantKey:  "E01-F01",
        },
        {
            name:     "task scope - full key",
            args:     []string{"T-E01-F01-001"},
            wantType: ScopeTask,
            wantKey:  "T-E01-F01-001",
        },
        {
            name:     "task scope - three args",
            args:     []string{"E01", "F01", "001"},
            wantType: ScopeTask,
            wantKey:  "T-E01-F01-001",
        },
        {
            name:      "invalid - no args",
            args:      []string{},
            wantError: true,
        },
        {
            name:      "invalid - bad epic key",
            args:      []string{"E1"},
            wantError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            interpreter := NewInterpreter()
            scope, err := interpreter.ParseScope(tt.args)

            if tt.wantError {
                assert.Error(t, err)
                return
            }

            assert.NoError(t, err)
            assert.Equal(t, tt.wantType, scope.Type)
            assert.Equal(t, tt.wantKey, scope.Key)
        })
    }
}
```

**2. ViewService Tests (`internal/view/service_test.go`)**

```go
func TestService_GetFilePath(t *testing.T) {
    // Use mock repositories (NOT real database - CLI test pattern)
    mockEpicRepo := &MockEpicRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
            return &models.Epic{
                Key:      "E01",
                FilePath: "docs/plan/E01-epic-name/epic.md",
            }, nil
        },
    }

    mockFeatureRepo := &MockFeatureRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
            return &models.Feature{
                Key:      "E01-F01",
                FilePath: "docs/plan/E01-epic-name/E01-F01-feature-name/feature.md",
            }, nil
        },
    }

    mockTaskRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{
                Key:      "T-E01-F01-001",
                FilePath: "docs/plan/E01-epic-name/E01-F01-feature-name/T-E01-F01-001.md",
            }, nil
        },
    }

    service := NewService(mockEpicRepo, mockFeatureRepo, mockTaskRepo)

    tests := []struct {
        name         string
        scope        *scope.Scope
        wantFilePath string
    }{
        {
            name: "epic file path",
            scope: &scope.Scope{
                Type: scope.ScopeEpic,
                Key:  "E01",
            },
            wantFilePath: "docs/plan/E01-epic-name/epic.md",
        },
        {
            name: "feature file path",
            scope: &scope.Scope{
                Type: scope.ScopeFeature,
                Key:  "E01-F01",
            },
            wantFilePath: "docs/plan/E01-epic-name/E01-F01-feature-name/feature.md",
        },
        {
            name: "task file path",
            scope: &scope.Scope{
                Type: scope.ScopeTask,
                Key:  "T-E01-F01-001",
            },
            wantFilePath: "docs/plan/E01-epic-name/E01-F01-feature-name/T-E01-F01-001.md",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            filePath, err := service.GetFilePath(context.Background(), tt.scope)
            assert.NoError(t, err)
            assert.Equal(t, tt.wantFilePath, filePath)
        })
    }
}

func TestService_LaunchViewer(t *testing.T) {
    // Test viewer launching logic
    // Mock os/exec to verify correct command construction
}
```

**3. View Command Tests (`internal/cli/commands/view_test.go`)**

```go
func TestViewCommand(t *testing.T) {
    // Use MOCKED repositories (never real database in CLI tests)
    // Test command argument parsing
    // Test integration with ScopeInterpreter
    // Test config loading for viewer setting
}
```

### Integration Tests

Create end-to-end tests that:
1. Create test epic/feature/task in database
2. Run `shark view E01` command
3. Verify correct viewer command is executed
4. Cleanup test data

## Error Handling

### Error Scenarios

1. **Invalid scope arguments**
   - Error: "invalid key format"
   - Example: `shark view E1` (should be E01)
   - Handled by: ScopeInterpreter

2. **Entity not found**
   - Error: "epic/feature/task not found: E99"
   - Example: `shark view E99` (non-existent epic)
   - Handled by: ViewService (repository query)

3. **File path not set**
   - Error: "file path not set for epic E01"
   - Example: Epic exists in DB but FilePath is NULL
   - Handled by: ViewService

4. **File does not exist**
   - Error: "file not found: /path/to/spec.md"
   - Example: FilePath in DB but file deleted from filesystem
   - Handled by: ViewService.LaunchViewer

5. **Viewer command fails**
   - Error: "failed to launch viewer: command not found: glow"
   - Example: Viewer configured but not installed
   - Handled by: ViewService.LaunchViewer

6. **Config load failure**
   - Error: "failed to load config"
   - Example: Malformed .sharkconfig.json
   - Handled by: View command

### Error Messages

Use consistent error format:
```
Error: <context>: <specific error>

Examples:
  shark view E99
  Error: epic not found: E99

  shark view E01
  Error: file not found: docs/plan/E01-epic-name/epic.md

  shark view E01-F01
  Error: failed to launch viewer: command not found: glow
```

## SOLID Principles Application

### Single Responsibility

- **ScopeInterpreter**: Only parses CLI arguments to determine scope
- **ViewService**: Only handles viewing operations (file path + launcher)
- **ViewCommand**: Only handles CLI interaction (parse flags, call services)

### Open/Closed

- **ScopeInterpreter**: Can add new scope types (e.g., ScopeProject) without modifying existing code
- **ViewService**: Can add new viewing methods (e.g., ViewInBrowser) without changing GetFilePath

### Liskov Substitution

- **Repository interfaces**: Can swap mock repositories in tests without breaking ViewService
- **Interpreter interface**: Can create alternative scope interpreters (e.g., JSONScopeInterpreter)

### Interface Segregation

- **Repositories**: ViewService only depends on GetByKey() method, not entire repository interface
- **ScopeInterpreter**: Simple interface with single ParseScope() method

### Dependency Inversion

- **ViewCommand**: Depends on ViewService abstraction, not concrete implementation
- **ViewService**: Depends on repository interfaces, not concrete repository implementations

## DRY Improvements

### Before (Duplicated Logic)

- `get` command has scope parsing logic in `ParseGetArgs()`
- `list` command has scope parsing logic in `ParseListArgs()`
- Future commands would duplicate this logic again

### After (Reusable Component)

- All commands use `scope.Interpreter.ParseScope()`
- Single source of truth for scope determination
- Consistent error messages across commands

## Migration Path

### Phase 1: Extract ScopeInterpreter

1. Create `/internal/cli/scope/` package
2. Extract logic from `ParseGetArgs()` to `Interpreter.ParseScope()`
3. Refactor `get` command to use ScopeInterpreter
4. Run existing tests to ensure no regression

### Phase 2: Add ViewService

1. Create `/internal/view/` package
2. Implement `Service.GetFilePath()`
3. Implement `Service.LaunchViewer()`
4. Write unit tests

### Phase 3: Add View Command

1. Create `/internal/cli/commands/view.go`
2. Implement `runView()` using ScopeInterpreter and ViewService
3. Write CLI tests with mocks

### Phase 4: Config Extension

1. Add `Viewer` field to `Config` struct
2. Add `GetViewer()` helper method
3. Update config tests

## Example Usage

### Basic Usage

```bash
# View epic specification
shark view E01

# View feature specification
shark view E01 F01
shark view E01-F01

# View task specification
shark view E01 F01 001
shark view T-E01-F01-001
```

### With Custom Viewer

```bash
# Configure viewer in .sharkconfig.json
{
  "viewer": "glow"
}

# View with glow (renders Markdown with syntax highlighting)
shark view E01-F01

# Or use different viewers:
{
  "viewer": "bat"      # Syntax highlighting
}
{
  "viewer": "nano"     # Interactive editor
}
{
  "viewer": "less"     # Paginated viewer
}
```

### Flag-Based Usage

```bash
# Flag syntax (for backward compatibility)
shark view --epic=E01
shark view --epic=E01 --feature=F01
```

## Future Enhancements

1. **Interactive mode**: If multiple specs exist, let user select which one to view
2. **Diff mode**: `shark view --diff E01-F01` shows changes since last sync
3. **Browser mode**: `shark view --browser E01` opens in web browser with rendered Markdown
4. **Editor integration**: `shark view --edit E01-F01` opens in configured editor
5. **JSON output**: `shark view E01 --json` outputs file path as JSON (for scripting)

## Conclusion

This design:
- ✅ Implements all requirements
- ✅ Follows DRY principle (reusable ScopeInterpreter)
- ✅ Follows SOLID principles (SRP, OCP, LSP, ISP, DIP)
- ✅ Reuses existing patterns (argument parsing, repository queries, config)
- ✅ Testable with mocks (no database dependency in CLI tests)
- ✅ Extensible for future commands and features
- ✅ Backward compatible with existing CLI patterns

The ScopeInterpreter is the key innovation - it extracts common CLI argument parsing logic into a reusable component that can be used by multiple commands (get, view, list, edit, validate, export, etc.).
