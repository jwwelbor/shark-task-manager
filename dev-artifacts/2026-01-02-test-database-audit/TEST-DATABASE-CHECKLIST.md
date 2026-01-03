# Test Database Isolation Checklist

Use this checklist when writing new tests to ensure production database safety.

---

## Repository Tests (`internal/repository/*_test.go`)

### ✅ DO:
```go
func TestMyRepository(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()  // ✅ Use shared test database
    db := NewDB(database)
    repo := NewMyRepository(db)

    // ✅ Clean up test data BEFORE running test
    _, _ = database.ExecContext(ctx, "DELETE FROM my_table WHERE key = 'TEST-KEY'")

    // Run test
    // ...

    // ✅ Clean up after test (optional but recommended)
    defer database.ExecContext(ctx, "DELETE FROM my_table WHERE id = ?", myEntity.ID)
}
```

### ❌ DON'T:
```go
// ❌ NEVER use production database path
database, err := db.InitDB("shark-tasks.db")

// ❌ NEVER use project root database
database, err := db.InitDB("./shark-tasks.db")

// ❌ NEVER skip cleanup (causes test pollution)
repo.Create(ctx, myEntity)
// Missing: cleanup before and after
```

---

## CLI Command Tests (`internal/cli/commands/*_test.go`)

### ✅ DO:
```go
func TestMyCommand(t *testing.T) {
    // ✅ Option 1: Use in-memory database
    database, err := db.InitDB(":memory:")
    if err != nil {
        t.Fatalf("Failed to initialize database: %v", err)
    }
    defer database.Close()

    // ✅ Option 2: Use mocks (preferred for CLI tests)
    mockRepo := &MockRepository{
        CreateFunc: func(ctx context.Context, entity *Entity) error {
            entity.ID = 123
            return nil
        },
    }

    // Test command logic
    // ...
}
```

### ❌ DON'T:
```go
// ❌ NEVER use production database
database, err := db.InitDB("shark-tasks.db")

// ❌ NEVER use shared test database (use :memory: instead)
database := test.GetTestDB()  // Wrong for CLI tests
```

---

## Service/Sync Tests (`internal/sync/*_test.go`, etc.)

### ✅ DO:
```go
func TestMyService(t *testing.T) {
    // ✅ Use temporary directory
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test-service.db")

    database, err := db.InitDB(dbPath)
    if err != nil {
        t.Fatalf("Failed to initialize database: %v", err)
    }
    defer database.Close()

    // Test service logic
    // ...

    // No cleanup needed - t.TempDir() auto-cleans
}
```

### ❌ DON'T:
```go
// ❌ NEVER use production database
database, err := db.InitDB("shark-tasks.db")

// ❌ NEVER create database in project root
dbPath := filepath.Join(os.Getwd(), "test.db")  // Wrong - use t.TempDir()
```

---

## Init/Setup Tests (`internal/init/*_test.go`)

### ✅ DO:
```go
func TestInit(t *testing.T) {
    // ✅ Create temp directory
    tempDir := t.TempDir()

    // ✅ Change to temp directory
    originalDir, _ := os.Getwd()
    defer os.Chdir(originalDir)
    os.Chdir(tempDir)

    // ✅ Now safe to create "shark-tasks.db" (in temp dir)
    opts := InitOptions{
        DBPath:     "shark-tasks.db",  // Safe - in temp directory
        ConfigPath: ".sharkconfig.json",
    }

    // Test init logic
    // ...
}
```

### ❌ DON'T:
```go
// ❌ NEVER init in project root without temp directory
opts := InitOptions{
    DBPath: "shark-tasks.db",  // Wrong - will use project root
}
initializer.Initialize(ctx, opts)  // Dangerous!

// ✅ FIX: Use temp directory first
tempDir := t.TempDir()
os.Chdir(tempDir)
// Now safe to use "shark-tasks.db"
```

---

## Unit Tests (Pure Logic)

### ✅ DO:
```go
func TestSlugGeneration(t *testing.T) {
    // ✅ No database needed for pure logic
    slug := GenerateSlug("My Title")

    if slug != "my-title" {
        t.Errorf("Expected 'my-title', got '%s'", slug)
    }
}
```

### ❌ DON'T:
```go
// ❌ NEVER use database for pure logic tests
database := test.GetTestDB()  // Unnecessary
```

---

## Quick Decision Tree

```
┌─ Writing a new test?
│
├─ Testing repository CRUD operations?
│  └─ ✅ Use: test.GetTestDB()
│     └─ Clean data before test
│
├─ Testing CLI command?
│  ├─ Need real DB operations?
│  │  └─ ✅ Use: db.InitDB(":memory:")
│  └─ Can mock?
│     └─ ✅ Use: Mocks (preferred)
│
├─ Testing service/sync logic?
│  └─ ✅ Use: t.TempDir() + temp database
│
├─ Testing init/setup?
│  └─ ✅ Use: t.TempDir() + os.Chdir()
│
└─ Testing pure logic (utils, parsers)?
   └─ ✅ Use: No database
```

---

## Verification Commands

### Check Production Database Safety
```bash
# Get timestamp before test
stat -c "%Y" shark-tasks.db

# Run tests
go test ./internal/repository -v

# Get timestamp after test
stat -c "%Y" shark-tasks.db

# Timestamps should match! ✅
```

### Find Hardcoded Database References
```bash
# Search for production DB references in tests
grep -r "shark-tasks\.db" --include="*_test.go" internal/

# All results should be:
# 1. Inside t.TempDir()
# 2. In comments
# 3. Testing init command
```

### Verify Test Database Location
```bash
# Check test database exists and is separate
ls -la internal/repository/test-shark-tasks.db

# Should NOT be in project root
ls -la shark-tasks.db  # This is production
```

---

## Common Mistakes

### Mistake 1: Forgetting to Clean Up
```go
// ❌ BAD
func TestCreate(t *testing.T) {
    database := test.GetTestDB()
    repo := NewTaskRepository(db)
    repo.Create(ctx, task)  // Leaves data in DB
}

// ✅ GOOD
func TestCreate(t *testing.T) {
    database := test.GetTestDB()
    repo := NewTaskRepository(db)

    // Clean first
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", task.Key)

    repo.Create(ctx, task)

    // Clean after
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
}
```

### Mistake 2: Using GetTestDB in CLI Tests
```go
// ❌ BAD - CLI tests should NOT use shared test database
func TestCommand(t *testing.T) {
    database := test.GetTestDB()  // Wrong pattern for CLI
}

// ✅ GOOD - Use in-memory or mocks
func TestCommand(t *testing.T) {
    database, _ := db.InitDB(":memory:")
    defer database.Close()
}
```

### Mistake 3: Creating Database in Project Root
```go
// ❌ BAD
func TestSync(t *testing.T) {
    dbPath := "test-sync.db"  // Creates in project root!
    database, _ := db.InitDB(dbPath)
}

// ✅ GOOD
func TestSync(t *testing.T) {
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test-sync.db")
    database, _ := db.InitDB(dbPath)
}
```

---

## Pre-Commit Checklist

Before committing new test files:

- [ ] No hardcoded "shark-tasks.db" paths (except in temp directories)
- [ ] Repository tests use `test.GetTestDB()`
- [ ] Repository tests clean up data before running
- [ ] CLI tests use `:memory:` or mocks
- [ ] Service tests use `t.TempDir()`
- [ ] No database created in project root
- [ ] Run `go test ./...` and verify production DB timestamp unchanged

---

## Getting Help

**If unsure which pattern to use:**
1. Check existing tests in the same category
2. Refer to CLAUDE.md "Testing Architecture" section
3. Run this verification after writing test:
   ```bash
   stat -c "%Y" shark-tasks.db  # Before
   go test -v ./your-package -run YourTest
   stat -c "%Y" shark-tasks.db  # After (should be same)
   ```

**If production database was touched:**
1. Stop immediately
2. Don't commit changes
3. Review test database pattern
4. Restore production database from backup if needed
5. Rewrite test using correct pattern

---

## References

- **CLAUDE.md** - Testing Architecture section (lines 550-700)
- **internal/test/testdb.go** - Test database infrastructure
- **Full audit report** - `dev-artifacts/2026-01-02-test-database-audit/audit-report.md`
