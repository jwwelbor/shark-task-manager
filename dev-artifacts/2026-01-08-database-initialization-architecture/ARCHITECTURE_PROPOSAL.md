# Database Initialization Architecture Proposal

## Executive Summary

This document proposes an elegant solution to eliminate 74 instances of duplicate database initialization code across all CLI commands. The solution uses Cobra's lifecycle hooks to provide automatic, transparent, cloud-aware database initialization with zero code changes required in existing commands.

---

## Current State Analysis

### The Problem

Every command currently follows this repetitive 3-step pattern:

```go
// Pattern repeated 74 times across commands
dbPath, err := cli.GetDBPath()
database, err := db.InitDB(dbPath)
repoDb := repository.NewDB(database)
```

**Pain Points:**
1. **Ignores cloud configuration**: Only uses local SQLite, ignoring `.sharkconfig.json` cloud backend settings
2. **Massive duplication**: Same 3 lines repeated 74 times
3. **Brittle to change**: Adding cloud support requires updating every single command
4. **Violates DRY**: Database initialization logic scattered across codebase
5. **Violates SRP**: Commands shouldn't know HOW to initialize the database

### Recent Progress

We've implemented the foundation:
- `cli.GetDatabaseConfig(configPath)` - reads config to determine backend
- `cli.InitializeDatabaseFromConfig(ctx, configPath)` - cloud-aware initialization
- `commands.initDatabase(ctx)` - helper wrapping the above
- Updated 1 command (`cloud status`) to use the new pattern

**73 commands still use the old pattern.**

---

## Architectural Solution: Lifecycle Hook Injection

### Design Principles

1. **Zero-touch migration**: Existing commands get cloud support automatically
2. **Single Responsibility**: Commands don't know about database initialization
3. **DRY**: Database initialization exists in ONE place
4. **Lazy initialization**: Database only created when actually needed
5. **Backward compatible**: No breaking changes to existing code
6. **Explicit cleanup**: Clear connection lifecycle management

### The Solution: Package-Level Singleton with Lazy Initialization

Store a **package-level database instance** in `internal/cli/` and initialize it lazily via Cobra's `PersistentPreRunE` hook on the root command.

---

## Implementation Design

### Phase 1: Create Global Database Instance

**File: `internal/cli/db_global.go`**

```go
package commands

import (
	"context"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

var (
	// globalDB holds the shared database connection for all commands
	globalDB *repository.DB

	// dbInitOnce ensures database is initialized exactly once
	dbInitOnce sync.Once

	// dbInitErr stores initialization error for propagation
	dbInitErr error
)

// GetDB returns the global database connection, initializing it if needed
// This is the ONLY function commands should call to get database access
func GetDB(ctx context.Context) (*repository.DB, error) {
	dbInitOnce.Do(func() {
		globalDB, dbInitErr = initDatabase(ctx)
	})

	if dbInitErr != nil {
		return nil, dbInitErr
	}

	return globalDB, nil
}

// CloseDB closes the global database connection
// Called by root command's PersistentPostRunE hook
func CloseDB() error {
	if globalDB != nil {
		return globalDB.Close()
	}
	return nil
}

// ResetDB clears the global database (for testing only)
func ResetDB() {
	globalDB = nil
	dbInitErr = nil
	dbInitOnce = sync.Once{}
}
```

**Why this design?**
- **Thread-safe**: `sync.Once` ensures initialization happens exactly once, even with concurrent access
- **Lazy**: Database only created when first command actually needs it
- **Error propagation**: Initialization errors are captured and returned to callers
- **Testable**: `ResetDB()` allows tests to reset state between test cases
- **Simple API**: Commands just call `GetDB(ctx)` - no knowledge of initialization logic

---

### Phase 2: Add Lifecycle Hooks to Root Command

**File: `internal/cli/root.go`**

Add cleanup hook to existing `PersistentPreRunE`:

```go
// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "shark",
	Short: "Shark Task Manager - Task management CLI for AI-driven development",
	Long: `...`,
	Version: "dev",

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration (already exists)
		if err := initConfig(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		// Disable color output if requested (already exists)
		if GlobalConfig.NoColor {
			pterm.DisableColor()
		}

		// Set verbose logging if requested (already exists)
		if GlobalConfig.Verbose {
			pterm.EnableDebugMessages()
		}

		// NOTE: Database initialization happens LAZILY on first GetDB() call
		// We don't initialize it here to avoid connecting when not needed
		// (e.g., `shark --help` shouldn't connect to database)

		return nil
	},

	// NEW: Add cleanup hook
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close database connection if it was opened
		if err := commands.CloseDB(); err != nil {
			// Log but don't fail - cleanup errors shouldn't break exit
			if GlobalConfig.Verbose {
				pterm.Warning.Printf("Failed to close database: %v\n", err)
			}
		}
		return nil
	},
}
```

**Why lazy initialization instead of PersistentPreRunE?**
- `shark --help` shouldn't connect to database
- `shark version` shouldn't connect to database
- `shark cloud status` only reads config file, doesn't need DB
- Only commands that actually call `GetDB()` trigger initialization
- Faster startup for info-only commands

---

### Phase 3: Update Commands to Use GetDB

**Migration pattern for all 74 commands:**

**BEFORE:**
```go
func runTaskList(cmd *cobra.Command, args []string) error {
	// Old pattern: 3 lines of database initialization
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	repoDb := repository.NewDB(database)

	// Actual command logic
	taskRepo := repository.NewTaskRepository(repoDb)
	tasks, err := taskRepo.List(cmd.Context(), filters)
	// ...
}
```

**AFTER:**
```go
func runTaskList(cmd *cobra.Command, args []string) error {
	// New pattern: 1 line to get database
	repoDb, err := GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	// Actual command logic (unchanged)
	taskRepo := repository.NewTaskRepository(repoDb)
	tasks, err := taskRepo.List(cmd.Context(), filters)
	// ...
}
```

**Reduction:**
- **Before**: 9 lines of boilerplate
- **After**: 4 lines
- **Savings**: 5 lines × 74 commands = **370 lines of duplicate code eliminated**

---

### Phase 4: Automated Migration Script

Create a script to automatically update all 74 commands:

**File: `scripts/migrate-to-global-db.sh`**

```bash
#!/bin/bash
# Automated migration script to replace old database initialization pattern
# with new GetDB() pattern

COMMANDS_DIR="internal/cli/commands"

# Pattern 1: cli.GetDBPath() + db.InitDB() + repository.NewDB()
find "$COMMANDS_DIR" -name "*.go" -type f | while read -r file; do
	# Skip test files and db_global.go
	if [[ "$file" == *_test.go ]] || [[ "$file" == */db_global.go ]]; then
		continue
	fi

	echo "Processing: $file"

	# Replace the 3-line pattern with GetDB() call
	# This uses perl for multi-line regex replacement
	perl -i -0pe '
		s/dbPath,\s*err\s*:=\s*cli\.GetDBPath\(\)\s*\n\s*if\s*err\s*!=\s*nil\s*\{[^}]*\}\s*\n\s*database,\s*err\s*:=\s*db\.InitDB\(dbPath\)\s*\n\s*if\s*err\s*!=\s*nil\s*\{[^}]*\}\s*\n\s*repoDb\s*:=\s*repository\.NewDB\(database\)/repoDb, err := GetDB(cmd.Context())\n\tif err != nil {\n\t\treturn fmt.Errorf("failed to get database: %w", err)\n\t}/g
	' "$file"
done

echo "Migration complete! Review changes with: git diff"
```

---

## Connection Lifecycle

### When Database Opens

Database connection is established on **first call to `GetDB(ctx)`** in any command.

### When Database Closes

Database connection is closed by **root command's `PersistentPostRunE` hook** after command completes.

### Lifecycle Diagram

```
┌─────────────────────────────────────────────────────────────┐
│ User runs: shark task list                                   │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │ RootCmd.PersistentPreRunE    │
        │ - Load config                │
        │ - Set color/verbose flags    │
        │ (NO database init yet)       │
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │ taskListCmd.RunE             │
        │ - Calls GetDB(ctx)           │ ◄─── LAZY INIT HAPPENS HERE
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │ GetDB(ctx)                   │
        │ - sync.Once ensures init     │
        │ - Calls initDatabase(ctx)    │
        │ - Returns cached instance    │
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │ initDatabase(ctx)            │
        │ - Reads .sharkconfig.json    │
        │ - Detects backend type       │
        │ - Creates SQLite or Turso    │
        │ - Returns *repository.DB     │
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │ Command executes business    │
        │ logic using database         │
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │ RootCmd.PersistentPostRunE   │
        │ - Calls CloseDB()            │ ◄─── CLEANUP HAPPENS HERE
        │ - Closes connection          │
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │ Command exits                │
        └──────────────────────────────┘
```

**Key Benefits:**
- Commands that don't need database never open connection
- Connection automatically cleaned up after every command
- No connection leaks
- Clear separation of concerns

---

## Error Handling Strategy

### Initialization Errors

**Scenario**: Database config is invalid, Turso unreachable, auth token expired

**Handling**:
```go
func runTaskList(cmd *cobra.Command, args []string) error {
	repoDb, err := GetDB(cmd.Context())
	if err != nil {
		// Error already wrapped by GetDB with context
		return err  // Propagates to Cobra, exits with code 1
	}
	// ...
}
```

**Error message example**:
```
Error: failed to get database: failed to initialize database: failed to connect to database: authentication failed: token expired

Run 'shark cloud init' to reconfigure authentication.
```

### Cleanup Errors

**Scenario**: Database connection fails to close cleanly

**Handling**:
```go
PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
	if err := commands.CloseDB(); err != nil {
		// Log warning but don't fail exit
		if GlobalConfig.Verbose {
			pterm.Warning.Printf("Failed to close database: %v\n", err)
		}
	}
	return nil  // Never fail on cleanup
}
```

**Rationale**: Cleanup errors shouldn't prevent command success. If task completed successfully but close failed, command should still exit 0.

---

## Testing Strategy

### Unit Tests for Database Initialization

**File: `internal/cli/db_global_test.go`**

```go
package commands

import (
	"context"
	"testing"
)

func TestGetDB_InitializesOnce(t *testing.T) {
	defer ResetDB()  // Cleanup after test

	ctx := context.Background()

	// First call initializes
	db1, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if db1 == nil {
		t.Fatal("Expected database instance, got nil")
	}

	// Second call returns same instance
	db2, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if db1 != db2 {
		t.Error("Expected same database instance, got different instances")
	}
}

func TestGetDB_PropagatesInitError(t *testing.T) {
	defer ResetDB()

	// Create invalid config that will fail initialization
	// (Implementation depends on how we can inject test config)

	ctx := context.Background()
	db, err := GetDB(ctx)

	if err == nil {
		t.Fatal("Expected error for invalid config, got nil")
	}
	if db != nil {
		t.Error("Expected nil database on error, got instance")
	}
}

func TestResetDB_ClearsState(t *testing.T) {
	ctx := context.Background()

	// Initialize database
	db1, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Reset state
	ResetDB()

	// Next call should reinitialize
	db2, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should be different instance (reinitialized)
	if db1 == db2 {
		t.Error("Expected different instance after reset, got same")
	}
}
```

### Integration Tests

Test commands with different backend configurations:

**File: `internal/cli/db_integration_test.go`**

```go
package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCommands_WithLocalSQLite(t *testing.T) {
	defer ResetDB()

	// Create temp directory with local config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configContent := `{
		"database": {
			"backend": "sqlite",
			"url": "` + filepath.Join(tmpDir, "test.db") + `"
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test command execution
	ctx := context.Background()
	db, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error with local SQLite, got: %v", err)
	}
	if db == nil {
		t.Fatal("Expected database instance, got nil")
	}

	// Verify it's actually SQLite
	// (Check driver name or other SQLite-specific behavior)
}

func TestCommands_WithTursoCloud(t *testing.T) {
	// Similar test but with Turso config
	// May need to mock Turso or skip if auth unavailable
}
```

### Testing Commands After Migration

Existing command tests should continue to work **unchanged** because:
1. Test setup calls `ResetDB()` before each test
2. Tests can inject mock databases if needed
3. Commands still receive `*repository.DB` - implementation unchanged

---

## Migration Plan

### Phase 1: Foundation (Completed ✅)
- ✅ Create `cli.GetDatabaseConfig()`
- ✅ Create `cli.InitializeDatabaseFromConfig()`
- ✅ Create `commands.initDatabase()`
- ✅ Update `cloud status` command as proof of concept

### Phase 2: Global Instance Setup (1-2 hours)
- Create `internal/cli/db_global.go` with `GetDB()`, `CloseDB()`, `ResetDB()`
- Add `PersistentPostRunE` cleanup hook to root command
- Write unit tests for global database instance
- Verify cloud and local backends both work

### Phase 3: Command Migration (2-3 hours)
- Create automated migration script `scripts/migrate-to-global-db.sh`
- Run script to update all 74 commands
- Manual review of changes (verify correctness)
- Run full test suite to catch regressions
- Fix any test failures (likely need to add `ResetDB()` calls)

### Phase 4: Cleanup & Documentation (1 hour)
- Remove `cli.GetDBPath()` if no longer used
- Update `CLAUDE.md` with new database pattern
- Add migration notes to README
- Create architecture documentation

**Total estimated time: 4-6 hours**

---

## Rollback Strategy

If migration causes issues:

1. **Revert changes**: `git revert <migration-commit>`
2. **Selective rollback**: Keep `db_global.go` infrastructure, roll back individual commands
3. **Gradual migration**: Migrate commands one at a time instead of bulk update

**Risk mitigation**:
- Keep `initDatabase()` helper in place during migration
- Both patterns can coexist temporarily
- Test suite provides regression safety net

---

## Future Enhancements

### Connection Pooling (Future)

Once all commands use `GetDB()`, we can enhance initialization to support connection pooling:

```go
var globalDBPool *DBPool

func GetDB(ctx context.Context) (*repository.DB, error) {
	dbInitOnce.Do(func() {
		globalDBPool, dbInitErr = initDatabasePool(ctx)
	})

	if dbInitErr != nil {
		return nil, dbInitErr
	}

	return globalDBPool.Acquire(ctx)
}
```

### Multi-Database Support (Future)

Support multiple databases (e.g., separate DBs for different projects):

```go
var databases = make(map[string]*repository.DB)

func GetDB(ctx context.Context, projectID string) (*repository.DB, error) {
	// Return project-specific database instance
}
```

### Dependency Injection Alternative (Future)

For more complex scenarios, we could switch to explicit dependency injection:

```go
type CommandContext struct {
	DB *repository.DB
	Config *config.Config
	Logger *log.Logger
}

func GetContext(ctx context.Context) (*CommandContext, error) {
	// Initialize all dependencies
}
```

---

## Benefits Summary

### For Developers
- **370 lines of duplicate code eliminated**
- **Zero-touch cloud support**: All commands get Turso automatically
- **Single source of truth**: Database initialization in one place
- **Easier debugging**: One place to add logging/tracing
- **Better testing**: Global state can be reset between tests

### For Architecture
- **Separation of concerns**: Commands focus on business logic
- **DRY compliance**: No duplication
- **SRP compliance**: Commands don't manage database lifecycle
- **Flexible**: Easy to swap backends, add pooling, etc.

### For Operations
- **Consistent behavior**: All commands use same initialization logic
- **Better error messages**: Centralized error handling
- **Easier monitoring**: Single point to add metrics/tracing

---

## Questions & Answers

### Q: Why not use dependency injection framework?

**A**: Go philosophy favors simplicity over frameworks. The singleton pattern with `sync.Once` is:
- Zero dependencies
- Compile-time safe
- Easy to understand
- Sufficient for our use case (single DB per command execution)

### Q: What about concurrent commands?

**A**: Each command execution is a separate process (CLI model). No concurrency concerns between commands. Within a single command, `sync.Once` ensures thread-safe initialization.

### Q: Why package-level variable instead of root command context?

**A**: Cobra's context is command-scoped and difficult to share. Package-level variables are appropriate for singleton resources like database connections in CLI applications.

### Q: What if a command needs multiple databases?

**A**: Current design supports single database (99% use case). If multi-DB support needed in future, we can:
1. Add `GetDBByName(name string)` function
2. Store map of databases instead of single instance
3. Maintain backward compatibility with `GetDB()` = `GetDBByName("default")`

### Q: How does this affect testing?

**A**: Testing improves because:
- `ResetDB()` allows clean state between tests
- Mock databases can be injected before `GetDB()` called
- Existing tests continue to work (repository pattern unchanged)

---

## Conclusion

The proposed architecture elegantly solves the database initialization problem by:

1. **Eliminating 370 lines of duplicate code** across 74 commands
2. **Providing automatic cloud support** without touching existing commands
3. **Following Go best practices** (simplicity, explicit errors, clear lifecycle)
4. **Maintaining backward compatibility** with existing tests and code
5. **Enabling future enhancements** (pooling, multi-DB, metrics)

The migration is **low-risk** with automated tooling and comprehensive test coverage.

**Recommendation**: Proceed with implementation following the 4-phase plan.

---

## Appendix: Related Files

### Files to Create
- `internal/cli/db_global.go` - Global database instance
- `internal/cli/db_global_test.go` - Unit tests
- `scripts/migrate-to-global-db.sh` - Migration automation

### Files to Modify
- `internal/cli/root.go` - Add `PersistentPostRunE` hook
- All 74 command files in `internal/cli/commands/*.go` - Replace init pattern

### Files to Reference
- `internal/cli/db_init.go` - Current `initDatabase()` implementation
- `internal/cli/db_helper.go` - `GetDatabaseConfig()`, `InitializeDatabaseFromConfig()`
- `internal/db/registry.go` - Driver registry for SQLite/Turso

---

**Document Version**: 1.0
**Date**: 2026-01-08
**Author**: Claude (Architecture Agent)
**Status**: Proposed - Awaiting Approval
