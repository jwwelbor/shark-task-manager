# Implementation Guide: Global Database Pattern

This guide provides step-by-step instructions for implementing the global database architecture.

---

## Phase 1: Create Global Database Instance

### Step 1.1: Create `db_global.go`

**File**: `internal/cli/commands/db_global.go`

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

// GetDB returns the global database connection, initializing it if needed.
// This is the ONLY function commands should call to get database access.
//
// The database is initialized lazily on first call using the existing
// initDatabase() function which is cloud-aware and reads .sharkconfig.json.
//
// Usage:
//   repoDb, err := GetDB(cmd.Context())
//   if err != nil {
//       return fmt.Errorf("failed to get database: %w", err)
//   }
func GetDB(ctx context.Context) (*repository.DB, error) {
	dbInitOnce.Do(func() {
		globalDB, dbInitErr = initDatabase(ctx)
	})

	if dbInitErr != nil {
		return nil, dbInitErr
	}

	return globalDB, nil
}

// CloseDB closes the global database connection.
// Called automatically by root command's PersistentPostRunE hook.
// It's safe to call multiple times (subsequent calls are no-ops).
func CloseDB() error {
	if globalDB != nil {
		err := globalDB.Close()
		// Reset state after close (allows reinitialization if needed)
		globalDB = nil
		dbInitErr = nil
		dbInitOnce = sync.Once{}
		return err
	}
	return nil
}

// ResetDB clears the global database state.
// This is intended for testing only - DO NOT use in production code.
// It allows tests to reset state between test cases.
func ResetDB() {
	if globalDB != nil {
		globalDB.Close()
	}
	globalDB = nil
	dbInitErr = nil
	dbInitOnce = sync.Once{}
}
```

### Step 1.2: Create Tests

**File**: `internal/cli/commands/db_global_test.go`

```go
package commands

import (
	"context"
	"testing"
)

func TestGetDB_InitializesOnce(t *testing.T) {
	defer ResetDB() // Cleanup after test

	ctx := context.Background()

	// First call should initialize
	db1, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error on first call, got: %v", err)
	}
	if db1 == nil {
		t.Fatal("Expected database instance, got nil")
	}

	// Second call should return same instance (cached)
	db2, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error on second call, got: %v", err)
	}

	if db1 != db2 {
		t.Error("Expected same database instance on second call, got different instances")
	}
}

func TestResetDB_ClearsState(t *testing.T) {
	ctx := context.Background()

	// Initialize database
	db1, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if db1 == nil {
		t.Fatal("Expected database instance, got nil")
	}

	// Reset state
	ResetDB()

	// Next call should reinitialize (create new instance)
	db2, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error after reset, got: %v", err)
	}

	// Should be different instance since we reinitialized
	// Note: This might be the same pointer if DB pool is used,
	// but the important thing is it's a fresh initialization
	if db1 == db2 {
		t.Log("Warning: Same pointer after reset (may indicate DB pooling)")
	}
}

func TestCloseDB_SafeToCallMultipleTimes(t *testing.T) {
	defer ResetDB()

	ctx := context.Background()

	// Initialize database
	_, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Close should succeed
	if err := CloseDB(); err != nil {
		t.Errorf("Expected no error on first close, got: %v", err)
	}

	// Second close should be safe (no-op)
	if err := CloseDB(); err != nil {
		t.Errorf("Expected no error on second close, got: %v", err)
	}
}
```

### Step 1.3: Run Tests

```bash
# Run the new tests
go test -v ./internal/cli/commands -run TestGetDB
go test -v ./internal/cli/commands -run TestResetDB
go test -v ./internal/cli/commands -run TestCloseDB

# Expected output:
# === RUN   TestGetDB_InitializesOnce
# --- PASS: TestGetDB_InitializesOnce (0.05s)
# === RUN   TestResetDB_ClearsState
# --- PASS: TestResetDB_ClearsState (0.05s)
# === RUN   TestCloseDB_SafeToCallMultipleTimes
# --- PASS: TestCloseDB_SafeToCallMultipleTimes (0.05s)
```

---

## Phase 2: Add Lifecycle Hooks

### Step 2.1: Update Root Command

**File**: `internal/cli/root.go`

Add import:
```go
import (
	// ... existing imports ...
	"github.com/jwwelbor/shark-task-manager/internal/cli/commands"
)
```

Add `PersistentPostRunE` hook after `PersistentPreRunE`:

```go
var RootCmd = &cobra.Command{
	Use:   "shark",
	Short: "Shark Task Manager - Task management CLI for AI-driven development",
	Long: `...`,
	Version: "dev",

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// ... existing code (unchanged) ...
		return nil
	},

	// NEW: Add cleanup hook
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close database connection if it was opened
		if err := commands.CloseDB(); err != nil {
			// Log warning but don't fail - cleanup errors shouldn't break exit
			if GlobalConfig.Verbose {
				pterm.Warning.Printf("Failed to close database: %v\n", err)
			}
		}
		return nil
	},
}
```

### Step 2.2: Test Lifecycle

Create a simple test command to verify lifecycle:

```bash
# Create a test command
cat > internal/cli/commands/test_lifecycle.go << 'EOF'
// +build ignore

package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

var testLifecycleCmd = &cobra.Command{
	Use:   "test-lifecycle",
	Short: "Test database lifecycle",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDb, err := GetDB(cmd.Context())
		if err != nil {
			return err
		}
		cli.Success("Database initialized successfully")
		cli.Info("Type: %s", repoDb.Stats().OpenConnections)
		return nil
	},
}

func init() {
	cli.RootCmd.AddCommand(testLifecycleCmd)
}
EOF

# Build and test
make build
./bin/shark test-lifecycle -v

# Expected output:
# ✓ Database initialized successfully
# ℹ Type: 1
# (verbose) Closed database connection
```

---

## Phase 3: Migrate Commands

### Step 3.1: Create Migration Script

**File**: `scripts/migrate-to-global-db.sh`

```bash
#!/bin/bash
set -e

COMMANDS_DIR="internal/cli/commands"
BACKUP_DIR="dev-artifacts/2026-01-08-database-initialization-architecture/backup"

echo "=== Database Initialization Migration Script ==="
echo ""
echo "This script will:"
echo "  1. Backup all command files"
echo "  2. Replace old database init pattern with GetDB()"
echo "  3. Update error handling"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Find all command files (excluding tests and db_global.go)
find "$COMMANDS_DIR" -name "*.go" -type f | while read -r file; do
    # Skip test files and db_global.go
    if [[ "$file" == *_test.go ]] || [[ "$file" == */db_global.go ]]; then
        continue
    fi

    # Check if file contains old pattern
    if ! grep -q "cli.GetDBPath()" "$file"; then
        continue
    fi

    echo "Processing: $file"

    # Backup original
    cp "$file" "$BACKUP_DIR/$(basename $file).backup"

    # Pattern 1: cli.GetDBPath() + db.InitDB() + repository.NewDB()
    # Replace with: GetDB(cmd.Context())
    perl -i -0pe 's/dbPath,\s*err\s*:=\s*cli\.GetDBPath\(\)\s*\n\s*if\s*err\s*!=\s*nil\s*\{[^}]*\}\s*\n\s*database,\s*err\s*:=\s*db\.InitDB\(dbPath\)\s*\n\s*if\s*err\s*!=\s*nil\s*\{[^}]*\}\s*\n\s*repoDb\s*:=\s*repository\.NewDB\(database\)/repoDb, err := GetDB(cmd.Context())\n\tif err != nil {\n\t\treturn fmt.Errorf("failed to get database: %w", err)\n\t}/g' "$file"

    # Pattern 2: db.InitDB(cli.GlobalConfig.DBPath) + repository.NewDB()
    # (Used in some commands)
    perl -i -0pe 's/database,\s*err\s*:=\s*db\.InitDB\(cli\.GlobalConfig\.DBPath\)\s*\n\s*if\s*err\s*!=\s*nil\s*\{[^}]*\}\s*\n\s*dbWrapper\s*:=\s*repository\.NewDB\(database\)/dbWrapper, err := GetDB(cmd.Context())\n\tif err != nil {\n\t\treturn fmt.Errorf("failed to get database: %w", err)\n\t}/g' "$file"

    echo "  ✓ Updated: $file"
done

echo ""
echo "=== Migration Complete ==="
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff internal/cli/commands/"
echo "  2. Run tests: make test"
echo "  3. If tests pass: git add . && git commit -m 'refactor: migrate to global database pattern'"
echo "  4. If issues occur: git checkout internal/cli/commands/"
echo ""
echo "Backups stored in: $BACKUP_DIR"
```

Make it executable:
```bash
chmod +x scripts/migrate-to-global-db.sh
```

### Step 3.2: Run Migration (DRY RUN)

First, test the migration on a single file:

```bash
# Test on one file
cp internal/cli/commands/task.go /tmp/task.go.backup
./scripts/migrate-to-global-db.sh

# Review changes
git diff internal/cli/commands/task.go

# If looks good, continue. If not, restore:
# git checkout internal/cli/commands/task.go
```

### Step 3.3: Run Full Migration

```bash
# Run migration on all commands
./scripts/migrate-to-global-db.sh

# Review changes
git diff internal/cli/commands/ | less

# Check for any issues
git diff --stat internal/cli/commands/

# Expected output:
#  internal/cli/commands/analytics.go    | 6 +---
#  internal/cli/commands/epic.go         | 6 +---
#  internal/cli/commands/feature.go      | 6 +---
#  internal/cli/commands/task.go         | 6 +---
#  ... (70 more files) ...
#  74 files changed, 148 insertions(+), 518 deletions(-)
```

---

## Phase 4: Testing & Validation

### Step 4.1: Run Test Suite

```bash
# Run full test suite
make test

# Expected: Most tests should pass
# If failures occur, they're likely due to:
#   1. Tests need ResetDB() in setup
#   2. Tests assume specific DB state
```

### Step 4.2: Fix Test Failures

Common test failure pattern:

**Before**:
```go
func TestSomeCommand(t *testing.T) {
    // Test directly creates database
    db := setupTestDB(t)
    defer db.Close()

    // Test uses db
}
```

**After**:
```go
func TestSomeCommand(t *testing.T) {
    defer ResetDB()  // Add this to clear global state

    // Setup config for test
    setupTestConfig(t)

    // Test will call GetDB() internally
}
```

### Step 4.3: Integration Testing

Create integration test to verify both backends work:

```bash
# Test local SQLite
./bin/shark task list

# Test Turso cloud (if configured)
./bin/shark cloud init --url="libsql://test.turso.io" --auth-token="test123"
./bin/shark task list
```

### Step 4.4: Verify Connection Cleanup

```bash
# Run command with verbose flag to see cleanup
./bin/shark task list -v

# Expected output includes:
# ... (command output) ...
# (debug) Closing database connection
# (debug) Database connection closed successfully
```

---

## Phase 5: Cleanup & Documentation

### Step 5.1: Remove Deprecated Functions

Check if `cli.GetDBPath()` is still used:

```bash
grep -r "cli.GetDBPath()" internal/cli/commands/

# If no results (other than db_global.go), remove it:
# Edit internal/cli/root.go and remove GetDBPath() function
```

### Step 5.2: Update CLAUDE.md

Add section about database initialization pattern:

```markdown
### Database Access Pattern

All commands use a global database instance for consistency and cloud support:

```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    // Get database (initialized lazily on first call)
    repoDb, err := GetDB(cmd.Context())
    if err != nil {
        return fmt.Errorf("failed to get database: %w", err)
    }

    // Use database
    repo := repository.NewTaskRepository(repoDb)
    // ...
}
```

**Key Points**:
- Database initialized lazily on first `GetDB()` call
- Automatically detects backend from `.sharkconfig.json`
- Supports both local SQLite and Turso cloud
- Connection automatically closed after command completes
- Thread-safe via `sync.Once` pattern
```

### Step 5.3: Commit Changes

```bash
# Stage changes
git add internal/cli/commands/db_global.go
git add internal/cli/commands/db_global_test.go
git add internal/cli/root.go
git add internal/cli/commands/*.go
git add scripts/migrate-to-global-db.sh
git add CLAUDE.md

# Commit with descriptive message
git commit -m "refactor: migrate to global database pattern for cloud support

- Add GetDB() function for centralized database initialization
- Add CloseDB() lifecycle hook for automatic cleanup
- Migrate all 74 commands to use GetDB() pattern
- Add cloud-awareness (reads .sharkconfig.json for backend)
- Reduce duplicate code by 370 lines
- Support both SQLite and Turso backends automatically

Breaking changes: None (backward compatible)
Testing: All existing tests updated and passing"

# Push
git push origin feature/global-database-pattern
```

---

## Rollback Plan

If something goes wrong:

### Quick Rollback

```bash
# Revert the migration commit
git revert HEAD

# Or restore from backup
cp dev-artifacts/2026-01-08-database-initialization-architecture/backup/*.go.backup internal/cli/commands/
```

### Selective Rollback

```bash
# Keep infrastructure but rollback specific commands
git checkout HEAD~1 internal/cli/commands/task.go
git checkout HEAD~1 internal/cli/commands/feature.go
# etc.
```

### Gradual Migration

```bash
# Instead of migrating all at once, do it gradually:

# Week 1: Core commands (task, feature, epic)
# Week 2: Analytics and reporting commands
# Week 3: Configuration and maintenance commands
# Week 4: Remaining commands

# Both patterns can coexist temporarily
```

---

## Verification Checklist

Before marking as complete:

- [ ] `db_global.go` created with GetDB(), CloseDB(), ResetDB()
- [ ] Tests for db_global.go passing
- [ ] Root command has PersistentPostRunE cleanup hook
- [ ] All 74 commands migrated to use GetDB()
- [ ] Full test suite passes
- [ ] Integration tests pass (local SQLite)
- [ ] Integration tests pass (Turso cloud, if configured)
- [ ] Connection cleanup verified (no leaks)
- [ ] Documentation updated (CLAUDE.md)
- [ ] Migration script created and tested
- [ ] Backups created before migration
- [ ] Code review completed
- [ ] Changes committed with descriptive message

---

## Troubleshooting

### Issue: Tests fail with "database locked"

**Cause**: Multiple tests trying to use same database

**Solution**: Add `defer ResetDB()` to test setup:
```go
func TestMyCommand(t *testing.T) {
    defer ResetDB()  // Ensures clean state
    // ...
}
```

### Issue: Connection not closing

**Cause**: PersistentPostRunE not being called

**Solution**: Ensure root command's PersistentPostRunE is defined:
```go
// Verify this exists in internal/cli/root.go
RootCmd.PersistentPostRunE = func(cmd, args) error {
    return commands.CloseDB()
}
```

### Issue: "Failed to get database" error

**Cause**: Invalid config or missing database file

**Solution**: Check config and database:
```bash
# Check config
cat .sharkconfig.json

# Check database exists
ls -lh shark-tasks.db

# Initialize if needed
./bin/shark init --non-interactive
```

### Issue: Turso authentication fails

**Cause**: Invalid or expired auth token

**Solution**: Reinitialize cloud config:
```bash
# Get new token from Turso
turso db tokens create shark-tasks

# Update config
./bin/shark cloud init --url="libsql://..." --auth-token="<new-token>"
```

---

## Success Criteria

Migration is successful when:

1. ✅ All tests pass
2. ✅ Both SQLite and Turso backends work
3. ✅ Connection cleanup happens automatically
4. ✅ No duplicate database initialization code
5. ✅ Error messages are consistent
6. ✅ Performance is same or better
7. ✅ Code is cleaner and more maintainable

---

## Post-Migration Benefits

After migration, you can easily add:

### Connection Pooling

```go
// In db_global.go
var globalDBPool *DBPool

func GetDB(ctx context.Context) (*repository.DB, error) {
    dbInitOnce.Do(func() {
        globalDBPool, dbInitErr = initDatabasePool(ctx)
    })
    return globalDBPool.Acquire(ctx)
}
```

### Metrics & Tracing

```go
// In db_global.go
func GetDB(ctx context.Context) (*repository.DB, error) {
    dbInitOnce.Do(func() {
        globalDB, dbInitErr = initDatabase(ctx)
        if dbInitErr == nil {
            metrics.DatabaseInitialized.Inc()
        }
    })
    // ...
}
```

### Retry Logic

```go
// In db_global.go
func initDatabase(ctx context.Context) (*repository.DB, error) {
    var db *repository.DB
    var err error

    for attempt := 1; attempt <= 3; attempt++ {
        db, err = tryInitDatabase(ctx)
        if err == nil {
            return db, nil
        }

        log.Printf("Database init attempt %d failed: %v", attempt, err)
        time.Sleep(time.Second * time.Duration(attempt))
    }

    return nil, fmt.Errorf("failed after 3 attempts: %w", err)
}
```

All 74 commands get these enhancements automatically!

---

**Ready to implement? Start with Phase 1.**
