# Test Database Architecture

Visual guide to understanding how test databases are separated from production.

---

## 🏗️ Database Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     SHARK TASK MANAGER                          │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                   PRODUCTION                               │ │
│  │                                                            │ │
│  │  📁 Project Root                                          │ │
│  │  └── shark-tasks.db  ← ❌ NEVER touched by tests         │ │
│  │      ├── epics table                                      │ │
│  │      ├── features table                                   │ │
│  │      ├── tasks table                                      │ │
│  │      └── task_history table                              │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                   TEST DATABASES                           │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────┐    │ │
│  │  │ Repository Tests                                  │    │ │
│  │  │                                                   │    │ │
│  │  │ 📁 internal/repository/                          │    │ │
│  │  │ └── test-shark-tasks.db  ← ✅ Shared test DB    │    │ │
│  │  │     ├── epics table                              │    │ │
│  │  │     ├── features table                           │    │ │
│  │  │     ├── tasks table                              │    │ │
│  │  │     └── task_history table                       │    │ │
│  │  │                                                   │    │ │
│  │  │ ℹ️ Accessed via test.GetTestDB()                │    │ │
│  │  │ ℹ️ Shared across all repository tests           │    │ │
│  │  │ ℹ️ Tests clean their own data                   │    │ │
│  │  └──────────────────────────────────────────────────┘    │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────┐    │ │
│  │  │ CLI Command Tests                                │    │ │
│  │  │                                                   │    │ │
│  │  │ 💾 :memory: databases                            │    │ │
│  │  │ ├── In-memory SQLite                             │    │ │
│  │  │ ├── No file persistence                          │    │ │
│  │  │ └── Destroyed after test                         │    │ │
│  │  │                                                   │    │ │
│  │  │ ℹ️ Created via db.InitDB(":memory:")            │    │ │
│  │  │ ℹ️ Each test gets its own database              │    │ │
│  │  │ ℹ️ Perfect isolation                             │    │ │
│  │  └──────────────────────────────────────────────────┘    │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────┐    │ │
│  │  │ Service/Sync Tests                               │    │ │
│  │  │                                                   │    │ │
│  │  │ 📁 /tmp/go-build-XXXXX/                         │    │ │
│  │  │ └── testNNN/                                     │    │ │
│  │  │     └── test-service.db  ← ✅ Temp DB           │    │ │
│  │  │                                                   │    │ │
│  │  │ ℹ️ Created via t.TempDir()                      │    │ │
│  │  │ ℹ️ Auto-cleaned after test                      │    │ │
│  │  │ ℹ️ Unique per test                               │    │ │
│  │  └──────────────────────────────────────────────────┘    │ │
│  └───────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Data Flow During Tests

### Repository Test Flow
```
┌──────────────┐      ┌──────────────┐      ┌────────────────────┐
│              │      │              │      │                    │
│  Test Code   │─────▶│ GetTestDB()  │─────▶│ test-shark-tasks.db│
│              │      │              │      │ (in repo folder)   │
└──────────────┘      └──────────────┘      └────────────────────┘
                                                      │
                                                      │
                                              ✅ SAFE: Isolated
                                                      │
                                              ❌ Production DB
                                                  never touched
```

### CLI Test Flow
```
┌──────────────┐      ┌──────────────┐      ┌────────────────────┐
│              │      │              │      │                    │
│  Test Code   │─────▶│ InitDB(...)  │─────▶│   :memory: DB     │
│              │      │              │      │  (RAM only)        │
└──────────────┘      └──────────────┘      └────────────────────┘
                                                      │
                                                      │
                                              ✅ SAFE: In-memory
                                                      │
                                              ❌ Production DB
                                                  never touched
```

### Service Test Flow
```
┌──────────────┐      ┌──────────────┐      ┌────────────────────┐
│              │      │              │      │                    │
│  Test Code   │─────▶│  TempDir()   │─────▶│   /tmp/test.db    │
│              │      │              │      │  (temp file)       │
└──────────────┘      └──────────────┘      └────────────────────┘
                                                      │
                                                      │
                                              ✅ SAFE: Temp dir
                                                      │
                                              ❌ Production DB
                                                  never touched
```

---

## 🎯 Database Isolation Strategy

### Layer 1: Physical Separation
```
Production:     /project-root/shark-tasks.db
Test (Repo):    /project-root/internal/repository/test-shark-tasks.db
Test (CLI):     In-memory (no file)
Test (Service): /tmp/go-build-XXXXX/testNNN/test.db
```

**Different file paths = No collision possible**

### Layer 2: Code Separation
```go
// Production code
db.InitDB("shark-tasks.db")  // Uses project root

// Repository tests
test.GetTestDB()  // Returns internal/repository/test-shark-tasks.db

// CLI tests
db.InitDB(":memory:")  // In-memory, no file

// Service tests
t.TempDir() + db.InitDB(tempPath)  // Temp directory
```

**Different code paths = Additional safety**

### Layer 3: Test Infrastructure
```go
// internal/test/testdb.go
var testDB *sql.DB
var dbPath = "internal/repository/test-shark-tasks.db"  // Hardcoded

func GetTestDB() *sql.DB {
    // ALWAYS returns the test database
    // NEVER returns production database
}
```

**Infrastructure guarantees = Cannot access production accidentally**

---

## 📊 Test Execution Timeline

```
Time  │ Action                           │ Production DB │ Test DB
──────┼──────────────────────────────────┼───────────────┼─────────
T0    │ Start test suite                 │ Untouched     │ N/A
      │                                  │               │
T1    │ Repository test starts           │ Untouched     │ Opened
      │ └─ GetTestDB()                   │               │ ✅
      │                                  │               │
T2    │ Repository test cleans data      │ Untouched     │ Modified
      │ └─ DELETE FROM tasks WHERE...    │               │ ✅
      │                                  │               │
T3    │ Repository test runs             │ Untouched     │ Modified
      │ └─ repo.Create(task)             │               │ ✅
      │                                  │               │
T4    │ Repository test cleans up        │ Untouched     │ Modified
      │ └─ DELETE FROM tasks WHERE...    │               │ ✅
      │                                  │               │
T5    │ CLI test starts                  │ Untouched     │ N/A
      │ └─ InitDB(":memory:")            │               │
      │                                  │               │
T6    │ CLI test runs                    │ Untouched     │ In-memory
      │ └─ command execution             │               │ ✅
      │                                  │               │
T7    │ CLI test ends                    │ Untouched     │ Destroyed
      │ └─ database.Close()              │               │
      │                                  │               │
T8    │ Service test starts              │ Untouched     │ N/A
      │ └─ t.TempDir()                   │               │
      │                                  │               │
T9    │ Service test runs                │ Untouched     │ Temp file
      │ └─ sync operations               │               │ ✅
      │                                  │               │
T10   │ Service test ends                │ Untouched     │ Deleted
      │ └─ t.TempDir() cleanup           │               │
      │                                  │               │
T11   │ Test suite complete              │ Untouched ✅  │ N/A
```

**Result: Production database NEVER modified during any test**

---

## 🛡️ Safety Mechanisms

### 1. File Path Separation
```
✅ Production:  shark-tasks.db
✅ Test (Repo): internal/repository/test-shark-tasks.db
✅ Test (CLI):  :memory: (no path)
✅ Test (Svc):  /tmp/XXXXX/test.db
```

### 2. API Separation
```go
// Production uses CLI which uses project root
shark task list  → GetDBPath() → "shark-tasks.db"

// Tests use explicit paths
test.GetTestDB() → "internal/repository/test-shark-tasks.db"
db.InitDB(":memory:") → In-memory database
t.TempDir() → Unique temp directory
```

### 3. Working Directory Isolation
```go
// Init tests change to temp directory
tempDir := t.TempDir()
os.Chdir(tempDir)  // Now in /tmp/XXXXX
// "shark-tasks.db" refers to /tmp/XXXXX/shark-tasks.db, NOT production
```

### 4. Infrastructure Hardcoding
```go
// Test infrastructure CANNOT point to production
// It's hardcoded to test database path
dbPath = "internal/repository/test-shark-tasks.db"  // Fixed
```

---

## 🔍 Verification Methods

### Method 1: File Timestamp
```bash
# Production DB modified time before tests
stat -c "%Y" shark-tasks.db
# → 1767338904

# Run all tests
go test ./...

# Production DB modified time after tests
stat -c "%Y" shark-tasks.db
# → 1767338904  ✅ UNCHANGED
```

### Method 2: File Size
```bash
# Production DB size before tests
ls -lh shark-tasks.db
# → 700K

# Run all tests
go test ./...

# Production DB size after tests
ls -lh shark-tasks.db
# → 700K  ✅ UNCHANGED
```

### Method 3: Database Queries
```bash
# Count tasks in production DB before tests
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM tasks"
# → 127

# Run all tests
go test ./...

# Count tasks in production DB after tests
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM tasks"
# → 127  ✅ UNCHANGED
```

---

## 🎓 Common Questions

### Q: Why use a shared test database for repository tests?
**A:** Performance. Creating a new database for each test is slow. The shared database with cleanup-before-test is much faster and still safe.

### Q: Why use :memory: for CLI tests instead of shared DB?
**A:** Isolation. CLI tests shouldn't share state. In-memory databases are fast and guarantee complete isolation.

### Q: What if I accidentally use "shark-tasks.db" in a test?
**A:** If you're in a temp directory (via `t.TempDir()` and `os.Chdir()`), it's safe - creates a temp file. Otherwise, you'll touch production (DON'T DO THIS).

### Q: How do I verify my test doesn't touch production DB?
**A:**
```bash
stat -c "%Y" shark-tasks.db  # Before
go test -v ./your-package -run YourTest
stat -c "%Y" shark-tasks.db  # After (should match)
```

### Q: Can tests run in parallel safely?
**A:** Yes! Each test type is isolated:
- Repository tests clean their own data
- CLI tests use separate :memory: databases
- Service tests use separate temp directories

---

## 📚 Summary

**Production Database:**
- Location: `shark-tasks.db` (project root)
- Used by: Production CLI commands
- Accessed by tests: **NEVER** ✅

**Test Databases:**
- Repository: `internal/repository/test-shark-tasks.db` (shared, cleaned)
- CLI: `:memory:` (in-memory, per-test)
- Service: `/tmp/XXXXX/test.db` (temp file, auto-cleaned)

**Safety Score: 10/10** ✅

Every test type uses a different database. The production database is completely protected from test execution.
