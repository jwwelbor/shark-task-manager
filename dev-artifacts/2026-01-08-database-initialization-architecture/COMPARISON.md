# Database Initialization Pattern Comparison

## Current Pattern (74 Commands)

### Code Structure

```
┌─────────────────────────────────────────────────────────┐
│ task.go (taskListCmd.RunE)                              │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  func runTaskList(cmd, args) error {                    │
│                                                          │
│    ┌──────────────────────────────────────────────┐    │
│    │ Database Initialization (9 lines)            │    │
│    ├──────────────────────────────────────────────┤    │
│    │ dbPath, err := cli.GetDBPath()               │    │
│    │ if err != nil {                              │    │
│    │   return fmt.Errorf("...")                   │    │
│    │ }                                            │    │
│    │                                              │    │
│    │ database, err := db.InitDB(dbPath)          │    │
│    │ if err != nil {                              │    │
│    │   return fmt.Errorf("...")                   │    │
│    │ }                                            │    │
│    │                                              │    │
│    │ repoDb := repository.NewDB(database)        │    │
│    └──────────────────────────────────────────────┘    │
│                                                          │
│    // Actual business logic (5-50 lines)                │
│    taskRepo := repository.NewTaskRepository(repoDb)     │
│    tasks, err := taskRepo.List(...)                     │
│    // ... format and output                             │
│  }                                                       │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ feature.go (featureListCmd.RunE)                        │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  func runFeatureList(cmd, args) error {                 │
│                                                          │
│    ┌──────────────────────────────────────────────┐    │
│    │ Database Initialization (9 lines)            │    │
│    │ EXACT SAME CODE AS ABOVE                     │    │
│    └──────────────────────────────────────────────┘    │
│                                                          │
│    // Actual business logic                             │
│  }                                                       │
└─────────────────────────────────────────────────────────┘

... REPEATED 72 MORE TIMES ...
```

### Problems

```
┌─────────────────────────────────────────────────────────┐
│                     DUPLICATION                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  9 lines × 74 commands = 666 lines total                │
│                                                          │
│  - 370 lines of actual initialization code              │
│  - 296 lines of error handling                          │
│                                                          │
│  All doing EXACTLY the same thing                       │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                MISSING FUNCTIONALITY                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ❌ Ignores .sharkconfig.json cloud settings            │
│  ❌ Only supports local SQLite                          │
│  ❌ No Turso cloud support                              │
│  ❌ Each command must be updated individually           │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│            MAINTENANCE NIGHTMARE                         │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Want to add connection pooling?                        │
│    → Update 74 files                                    │
│                                                          │
│  Want to add metrics/tracing?                           │
│    → Update 74 files                                    │
│                                                          │
│  Want to add retry logic?                               │
│    → Update 74 files                                    │
│                                                          │
│  Bug in initialization?                                 │
│    → Fix 74 files                                       │
└─────────────────────────────────────────────────────────┘
```

---

## Proposed Pattern (Zero-Touch Migration)

### Code Structure

```
┌──────────────────────────────────────────────────────────────────┐
│ db_global.go (Package-Level Singleton)                           │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  var globalDB *repository.DB                                     │
│  var dbInitOnce sync.Once                                        │
│  var dbInitErr error                                             │
│                                                                   │
│  func GetDB(ctx) (*repository.DB, error) {                       │
│    dbInitOnce.Do(func() {                                        │
│      globalDB, dbInitErr = initDatabase(ctx)  // Cloud-aware    │
│    })                                                            │
│    return globalDB, dbInitErr                                    │
│  }                                                               │
│                                                                   │
│  func CloseDB() error { /* cleanup */ }                          │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│ root.go (Lifecycle Management)                                   │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  RootCmd.PersistentPostRunE = func(cmd, args) error {           │
│    return commands.CloseDB()  // Automatic cleanup              │
│  }                                                               │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│ task.go (taskListCmd.RunE)                                       │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  func runTaskList(cmd, args) error {                             │
│                                                                   │
│    ┌────────────────────────────────────────────────┐           │
│    │ Database Access (4 lines)                      │           │
│    ├────────────────────────────────────────────────┤           │
│    │ repoDb, err := GetDB(cmd.Context())           │           │
│    │ if err != nil {                                │           │
│    │   return fmt.Errorf("failed to get db: %w")   │           │
│    │ }                                              │           │
│    └────────────────────────────────────────────────┘           │
│                                                                   │
│    // Actual business logic (UNCHANGED)                          │
│    taskRepo := repository.NewTaskRepository(repoDb)              │
│    tasks, err := taskRepo.List(...)                              │
│    // ... format and output                                      │
│  }                                                                │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│ feature.go (featureListCmd.RunE)                                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  func runFeatureList(cmd, args) error {                          │
│                                                                   │
│    ┌────────────────────────────────────────────────┐           │
│    │ Database Access (4 lines)                      │           │
│    │ SAME PATTERN, NO DUPLICATION                   │           │
│    └────────────────────────────────────────────────┘           │
│                                                                   │
│    // Actual business logic                                      │
│  }                                                                │
└──────────────────────────────────────────────────────────────────┘

... 72 MORE COMMANDS, ALL USE SAME PATTERN ...
```

### Benefits

```
┌──────────────────────────────────────────────────────────────────┐
│                     DRY PRINCIPLE                                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Database initialization: 1 function (db_global.go)              │
│  Commands just call: GetDB(ctx)                                  │
│                                                                   │
│  4 lines × 74 commands = 296 lines                               │
│                                                                   │
│  Reduction: 666 lines → 296 lines                                │
│  Saved: 370 lines of duplicate code (55% reduction)              │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│              AUTOMATIC CLOUD SUPPORT                              │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ✅ Reads .sharkconfig.json automatically                        │
│  ✅ Detects backend (SQLite or Turso)                            │
│  ✅ Loads auth tokens from file or environment                   │
│  ✅ All 74 commands get cloud support instantly                  │
│  ✅ Zero code changes in existing commands                       │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│              MAINTENANCE PARADISE                                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Want to add connection pooling?                                 │
│    → Update GetDB() function (1 place)                           │
│                                                                   │
│  Want to add metrics/tracing?                                    │
│    → Update initDatabase() (1 place)                             │
│                                                                   │
│  Want to add retry logic?                                        │
│    → Update GetDB() function (1 place)                           │
│                                                                   │
│  Bug in initialization?                                          │
│    → Fix initDatabase() (1 place)                                │
│                                                                   │
│  All 74 commands benefit automatically ✨                        │
└──────────────────────────────────────────────────────────────────┘
```

---

## Migration Comparison

### Current State (Manual Migration)

```
Step 1: Update command 1
  ├─ Read documentation
  ├─ Replace 9 lines with new pattern
  ├─ Test command
  └─ Commit

Step 2: Update command 2
  ├─ Read documentation (again)
  ├─ Replace 9 lines with new pattern
  ├─ Test command
  └─ Commit

... Repeat 72 more times ...

Step 74: Update command 74
  ├─ Still reading documentation
  ├─ Replace 9 lines (getting tired)
  ├─ Test command (hopefully)
  └─ Commit (finally!)

Time: ~5-10 minutes per command = 6-12 hours total
Risk: Copy-paste errors, inconsistencies, forgotten commands
Testing: Each command needs individual testing
```

### Proposed Approach (Automated Migration)

```
Step 1: Create db_global.go
  ├─ Implement GetDB() function
  ├─ Implement CloseDB() function
  ├─ Write unit tests
  └─ Commit

Step 2: Add lifecycle hook
  ├─ Update root.go PersistentPostRunE
  ├─ Test with existing commands
  └─ Commit

Step 3: Run migration script
  ├─ Execute: ./scripts/migrate-to-global-db.sh
  ├─ Script updates all 74 commands automatically
  ├─ Review changes: git diff
  └─ Run test suite: make test

Step 4: Fix any test failures
  ├─ Add ResetDB() calls to test setup
  └─ Commit

Time: 1-2 hours total (automated)
Risk: Low (script ensures consistency)
Testing: Full test suite validates all commands at once
```

---

## Architecture Comparison

### Current: Scattered Initialization

```
                     ┌──────────────┐
                     │   Commands   │
                     └──────┬───────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
   ┌─────────┐         ┌─────────┐       ┌─────────┐
   │ task.go │         │feature.go│  ...  │ epic.go │
   └────┬────┘         └────┬─────┘       └────┬────┘
        │                   │                   │
        ├─ GetDBPath()     ├─ GetDBPath()     ├─ GetDBPath()
        ├─ InitDB()        ├─ InitDB()        ├─ InitDB()
        └─ NewDB()         └─ NewDB()         └─ NewDB()

   ❌ Each command duplicates initialization
   ❌ 74 places to maintain
   ❌ No central control
```

### Proposed: Centralized Initialization

```
                     ┌──────────────┐
                     │  RootCmd     │
                     │ (Lifecycle)  │
                     └──────┬───────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  db_global.go │
                    │   GetDB()     │ ◄─── Single Source of Truth
                    │   CloseDB()   │
                    └───────┬───────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
   ┌─────────┐         ┌─────────┐       ┌─────────┐
   │ task.go │         │feature.go│  ...  │ epic.go │
   └────┬────┘         └────┬─────┘       └────┬────┘
        │                   │                   │
        └─ GetDB(ctx)      └─ GetDB(ctx)      └─ GetDB(ctx)

   ✅ All commands use same function
   ✅ 1 place to maintain
   ✅ Central control and lifecycle management
```

---

## Testing Comparison

### Current: Manual Test Updates

```go
// Every command test must be updated individually

// feature_test.go
func TestFeatureList(t *testing.T) {
    // Setup: Create temp database
    tmpDB := createTestDB(t)
    defer tmpDB.Close()

    // Old pattern - hardcoded to local SQLite
    // ❌ Can't test Turso backend
    // ❌ Doesn't use config file

    // Run command
    // ...
}

// task_test.go - SAME BOILERPLATE
func TestTaskList(t *testing.T) {
    tmpDB := createTestDB(t)
    defer tmpDB.Close()
    // ... duplicate code
}

// 74 test files with similar setup
```

### Proposed: Consistent Test Pattern

```go
// All command tests use same pattern

// feature_test.go
func TestFeatureList(t *testing.T) {
    defer ResetDB()  // ✅ Clean state

    // Setup: Create config with desired backend
    setupTestConfig(t, "sqlite")  // or "turso"

    // ✅ Can test both backends
    // ✅ Uses same config mechanism as production

    // Run command
    // ...
}

// task_test.go - SAME PATTERN
func TestTaskList(t *testing.T) {
    defer ResetDB()  // ✅ Clean state

    setupTestConfig(t, "sqlite")

    // ... same pattern as above
}

// Consistent across all 74 test files
```

---

## Execution Flow Comparison

### Current Flow (Local Only)

```
User: shark task list

┌──────────────────────────────────────────────┐
│ 1. RootCmd.PersistentPreRunE                 │
│    - Load config                             │
│    - Set flags                               │
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│ 2. taskListCmd.RunE                          │
│    - Call cli.GetDBPath()                    │
│    - Call db.InitDB(dbPath)                  │ ◄─── Always SQLite
│    - Create repository.NewDB()               │      Ignores config
│    - Execute business logic                  │
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│ 3. Command exits                             │
│    - Database connection leaked?             │ ◄─── No cleanup hook
│    - No explicit cleanup                     │
└──────────────────────────────────────────────┘
```

### Proposed Flow (Cloud-Aware)

```
User: shark task list

┌──────────────────────────────────────────────┐
│ 1. RootCmd.PersistentPreRunE                 │
│    - Load config                             │
│    - Set flags                               │
│    - (DB NOT initialized yet)                │
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│ 2. taskListCmd.RunE                          │
│    - Call GetDB(ctx)                         │ ◄─── Lazy init
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│ 3. GetDB() (first call)                      │
│    - Check sync.Once                         │
│    - Call initDatabase(ctx)                  │
│      ├─ Read .sharkconfig.json               │ ◄─── Cloud-aware
│      ├─ Detect backend (sqlite/turso)        │
│      ├─ Load auth tokens                     │
│      └─ Connect to database                  │
│    - Cache instance                          │
│    - Return to command                       │
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│ 4. Command executes business logic           │
│    - Uses database instance                  │
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│ 5. RootCmd.PersistentPostRunE                │
│    - Call CloseDB()                          │ ◄─── Automatic cleanup
│    - Close database connection               │      No leaks
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│ 6. Command exits cleanly                     │
└──────────────────────────────────────────────┘
```

---

## Error Handling Comparison

### Current: Inconsistent Errors

```go
// Command A
dbPath, err := cli.GetDBPath()
if err != nil {
    return fmt.Errorf("failed to get database path: %w", err)
}

// Command B
dbPath, err := cli.GetDBPath()
if err != nil {
    return fmt.Errorf("database path error: %w", err)
}

// Command C
dbPath, err := cli.GetDBPath()
if err != nil {
    return err  // No context
}

// Command D
dbPath, err := cli.GetDBPath()
if err != nil {
    return fmt.Errorf("DB initialization failed: %w", err)
}

// ❌ Inconsistent error messages
// ❌ Poor user experience
// ❌ Harder to debug
```

### Proposed: Consistent Errors

```go
// All commands use same error handling

func runTaskList(cmd, args) error {
    repoDb, err := GetDB(cmd.Context())
    if err != nil {
        return err  // Already wrapped by GetDB
    }
    // ...
}

func runFeatureList(cmd, args) error {
    repoDb, err := GetDB(cmd.Context())
    if err != nil {
        return err  // Same error format
    }
    // ...
}

// ✅ Consistent error messages
// ✅ Better user experience
// ✅ Easier to add context in one place
// ✅ Can add suggestions (e.g., "Run 'shark cloud init'")
```

**Example error output**:
```
Error: failed to get database: failed to connect to Turso: authentication failed: token expired

Suggestions:
  - Run 'shark cloud init' to reconfigure authentication
  - Check token file: ~/.turso/shark-token
  - Verify token with: turso db tokens validate <token>
```

---

## Metrics Summary

| Metric | Current | Proposed | Improvement |
|--------|---------|----------|-------------|
| Lines of init code | 666 | 296 | -55% |
| Duplicate code blocks | 74 | 1 | -99% |
| Cloud support | ❌ None | ✅ Full | ∞ |
| Maintenance points | 74 files | 1 file | -99% |
| Migration time | 6-12 hours | 1-2 hours | -83% |
| Error consistency | Low | High | ✅ |
| Test complexity | High | Low | ✅ |
| Connection cleanup | Manual | Automatic | ✅ |

---

## Conclusion

The proposed architecture provides:

- **55% reduction in code** (370 lines eliminated)
- **99% reduction in duplication** (74 init blocks → 1 function)
- **Automatic cloud support** for all 74 commands
- **Consistent error handling** across all commands
- **Easier testing** with ResetDB() pattern
- **Future-proof design** for pooling, metrics, multi-DB

**The migration is low-risk, highly automated, and delivers immediate benefits.**
