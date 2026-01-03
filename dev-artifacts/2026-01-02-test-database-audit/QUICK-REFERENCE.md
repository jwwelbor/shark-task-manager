# Test Database Patterns - Quick Reference

---

## Repository Tests

```go
func TestMyRepository(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()  // ✅
    db := NewDB(database)
    repo := NewMyRepository(db)

    // Clean before
    _, _ = database.ExecContext(ctx, "DELETE FROM my_table WHERE key = 'TEST-001'")

    // Test
    entity := &MyEntity{Key: "TEST-001"}
    err := repo.Create(ctx, entity)

    // Clean after
    defer database.ExecContext(ctx, "DELETE FROM my_table WHERE id = ?", entity.ID)
}
```

**Database:** `internal/repository/test-shark-tasks.db` (shared)

---

## CLI Tests

```go
func TestMyCommand(t *testing.T) {
    database, err := db.InitDB(":memory:")  // ✅
    if err != nil {
        t.Fatalf("Failed: %v", err)
    }
    defer database.Close()

    // Test command
}
```

**Database:** In-memory (no persistence)

---

## Service Tests

```go
func TestMyService(t *testing.T) {
    tmpDir := t.TempDir()  // ✅
    dbPath := filepath.Join(tmpDir, "test.db")

    database, err := db.InitDB(dbPath)
    if err != nil {
        t.Fatalf("Failed: %v", err)
    }
    defer database.Close()

    // Test service
}
```

**Database:** Temp file (auto-cleaned)

---

## Init Tests

```go
func TestInit(t *testing.T) {
    tempDir := t.TempDir()  // ✅
    originalDir, _ := os.Getwd()
    defer os.Chdir(originalDir)
    os.Chdir(tempDir)

    // Now safe to use "shark-tasks.db"
}
```

**Database:** Temp directory (isolated)

---

## DON'Ts

```go
// ❌ NEVER
database := db.InitDB("shark-tasks.db")
database := db.InitDB("./shark-tasks.db")
database := db.InitDB(filepath.Join(os.Getwd(), "shark-tasks.db"))

// ✅ ALWAYS
database := test.GetTestDB()              // Repository tests
database := db.InitDB(":memory:")         // CLI tests
database := db.InitDB(filepath.Join(t.TempDir(), "test.db"))  // Service tests
```

---

## Verify Safety

```bash
# Before test
stat -c "%Y" shark-tasks.db

# Run test
go test -v ./your-package -run YourTest

# After test (should be SAME)
stat -c "%Y" shark-tasks.db
```

---

## Test Database Location

```
Project Root
├── shark-tasks.db                           ← ❌ Production (DON'T touch in tests)
└── internal/repository/
    └── test-shark-tasks.db                  ← ✅ Test DB (repository tests only)
```

---

## Decision Tree

```
Repository CRUD? → test.GetTestDB()
CLI command?     → db.InitDB(":memory:")
Service/Sync?    → t.TempDir() + temp DB
Init/Setup?      → t.TempDir() + os.Chdir()
Pure logic?      → No database
```
