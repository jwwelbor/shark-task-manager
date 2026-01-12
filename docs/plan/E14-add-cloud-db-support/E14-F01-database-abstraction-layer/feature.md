# Feature: Database Abstraction Layer

**Feature Key:** E14-F01
**Epic:** E14 - Cloud Database Support
**Status:** Draft
**Execution Order:** 1

## Overview

Create a database abstraction layer that allows Shark to support multiple database backends (SQLite local, Turso cloud) without changing business logic or command implementations.

## Goal

### Problem

Currently, Shark is tightly coupled to SQLite with direct `database/sql` calls throughout the codebase. Adding cloud database support (Turso) requires:
- Swapping database drivers
- Supporting different connection strings (file paths vs URLs)
- Runtime backend selection based on configuration
- Maintaining backward compatibility with existing local SQLite code

Without an abstraction layer, we'd need to modify hundreds of repository method calls and duplicate logic.

### Solution

Implement a database interface that abstracts CRUD operations, allowing:
- Single codebase supporting multiple backends
- Configuration-driven backend selection (local vs cloud)
- Driver registry for easy addition of new backends
- Zero changes to command/business logic layers

### Impact

- **Developer velocity:** New backends added in days, not weeks
- **Backward compatibility:** Existing SQLite code works unchanged (100% coverage)
- **Testability:** Mock database interface for unit tests
- **Future-proof:** Easy to add PostgreSQL, MySQL, etc. if needed

## User Stories

### Must-Have Stories

**Story 1:** As a developer, I want Shark to automatically detect my database backend from configuration so that I don't need to change any commands when switching between local and cloud.

**Acceptance Criteria:**
- [ ] `SHARK_DB_URL=libsql://...` detected as Turso backend
- [ ] `SHARK_DB_URL=./shark-tasks.db` detected as SQLite backend
- [ ] All commands work identically on both backends
- [ ] No code changes required in command layer

**Story 2:** As a Shark contributor, I want to add new database backends without modifying existing code so that the system remains maintainable.

**Acceptance Criteria:**
- [ ] New backend requires only implementing interface
- [ ] Driver registry allows registration of new backends
- [ ] Existing tests pass without modification
- [ ] Documentation guides backend implementation

**Story 3:** As a user, I want configuration to control my database backend so that I can switch between local and cloud without reinstalling.

**Acceptance Criteria:**
- [ ] `.sharkconfig.json` specifies backend
- [ ] Environment variables override config file
- [ ] `--db` flag overrides environment variables
- [ ] Invalid backend shows helpful error message

## Requirements

### Functional Requirements

**REQ-F-001: Database Interface Definition**
- **Description:** Define Go interface with all CRUD operations needed by repositories
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Interface includes: Connect, Close, Query, Exec, Begin, Commit, Rollback
  - [ ] Interface supports transactions
  - [ ] Interface supports prepared statements
  - [ ] Interface compatible with existing repository signatures

**REQ-F-002: SQLite Driver Implementation**
- **Description:** Refactor existing SQLite code to implement database interface
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] All existing SQLite operations work through interface
  - [ ] No behavior changes (existing tests pass)
  - [ ] Connection pooling preserved
  - [ ] PRAGMA settings preserved (WAL mode, foreign keys, etc.)

**REQ-F-003: Driver Registry**
- **Description:** Registry pattern for backend selection at runtime
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Backends register themselves (SQLite, Turso)
  - [ ] Configuration specifies active backend
  - [ ] Invalid backend returns clear error
  - [ ] Default backend is SQLite (backward compat)

**REQ-F-004: Configuration Integration**
- **Description:** Read backend selection from config, env vars, and CLI flags
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Priority: CLI flag > env var > config file > default
  - [ ] URL format auto-detects backend (libsql:// = Turso, path = SQLite)
  - [ ] Config validation on startup
  - [ ] Clear error messages for misconfiguration

### Non-Functional Requirements

**Performance:**
- **REQ-NF-001:** Abstraction layer adds < 5ms latency per query
- **Measurement:** Benchmark existing queries vs abstracted queries
- **Target:** 95th percentile < 5ms overhead
- **Justification:** Must not degrade user experience

**Maintainability:**
- **REQ-NF-002:** Interface has < 15 methods
- **Measurement:** Code review
- **Justification:** Large interfaces are hard to implement and test

**Testability:**
- **REQ-NF-003:** All repositories testable with mock database
- **Measurement:** Unit test coverage of commands without real DB
- **Target:** 80%+ command test coverage with mocks

## Technical Design

### Database Interface

```go
package db

// Database interface abstracts database operations
type Database interface {
    // Connection management
    Connect(ctx context.Context, dsn string) error
    Close() error
    Ping(ctx context.Context) error

    // Query operations
    Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) Row
    Exec(ctx context.Context, query string, args ...interface{}) (Result, error)

    // Transaction support
    Begin(ctx context.Context) (Tx, error)

    // Metadata
    DriverName() string
}

// Tx represents a database transaction
type Tx interface {
    Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) Row
    Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
    Commit() error
    Rollback() error
}
```

### Driver Registry

```go
package db

var drivers = make(map[string]DriverFactory)

type DriverFactory func() Database

// RegisterDriver allows backends to register themselves
func RegisterDriver(name string, factory DriverFactory) {
    drivers[name] = factory
}

// NewDatabase creates database instance based on config
func NewDatabase(config Config) (Database, error) {
    backend := DetectBackend(config.URL)
    factory, exists := drivers[backend]
    if !exists {
        return nil, fmt.Errorf("unknown backend: %s", backend)
    }
    return factory(), nil
}

// DetectBackend auto-detects from URL
func DetectBackend(url string) string {
    if strings.HasPrefix(url, "libsql://") {
        return "turso"
    }
    return "sqlite"
}
```

### Usage Example

```go
// Repository layer (no changes needed!)
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*Task, error) {
    row := r.db.QueryRow(ctx, "SELECT * FROM tasks WHERE id = ?", id)
    // ... same as before
}

// Initialization (command layer)
func initDB(config Config) (db.Database, error) {
    db, err := db.NewDatabase(config)
    if err != nil {
        return nil, err
    }
    if err := db.Connect(ctx, config.URL); err != nil {
        return nil, err
    }
    return db, nil
}
```

## Tasks

- **T-E14-F01-001:** Define database interface with CRUD operations (Priority: 9)
- **T-E14-F01-002:** Refactor SQLite repository to implement database interface (Priority: 8)
- **T-E14-F01-003:** Implement database driver registry for backend selection (Priority: 7)
- **T-E14-F01-004:** Write unit tests for database abstraction layer (Priority: 6)
- **T-E14-F01-005:** Add configuration fields for database backend selection (Priority: 9)

## Dependencies

- None (this is the foundation for all other features)

## Success Metrics

- [ ] All existing tests pass with SQLite driver
- [ ] Command layer has zero direct database imports
- [ ] Mock database interface used in 80%+ command tests
- [ ] Benchmark shows < 5ms abstraction overhead

## Out of Scope

- **ORM adoption:** Staying with raw SQL for performance and simplicity
- **Query builder:** Current SQL strings work fine
- **Migration to different SQL dialects:** Focusing on SQLite-compatible backends only

---

*Last Updated:* 2026-01-04
