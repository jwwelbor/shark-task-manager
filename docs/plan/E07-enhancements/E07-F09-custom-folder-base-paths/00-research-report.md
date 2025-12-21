# Research Report: E07-F09 Custom Folder Base Paths

**Feature**: E07-F09-Custom Folder Base Paths
**Epic**: E07-Enhancements
**Date**: 2025-12-19
**Researcher**: Architecture Workflow

---

## Executive Summary

This research report analyzes the Shark Task Manager codebase to inform the design of custom folder base paths (E07-F09). This feature enables epics and features to specify custom base folder paths that cascade to child items, building upon the custom filename infrastructure from E07-F08.

**Key Findings**:
- Database uses SQLite with foreign keys, WAL mode, and triggers for timestamp management
- E07-F08 adds `file_path` column to epics/features for exact file locations
- E07-F09 needs parallel `custom_folder_path` column for base folder inheritance
- Existing pattern validation infrastructure can be reused for path security
- Repository layer follows constructor injection pattern with context-based methods
- CLI uses Cobra framework with consistent flag naming and JSON output support

---

## 1. Database Architecture

### 1.1 Current Schema

The database schema is defined in `internal/db/db.go` using SQLite with the following configuration:

**SQLite Configuration** (db.go:36-62):
- **Foreign Keys**: Enabled via `PRAGMA foreign_keys = ON`
- **Journal Mode**: WAL (Write-Ahead Logging) for concurrency
- **Busy Timeout**: 5 seconds
- **Cache Size**: 64MB in-memory cache
- **Memory Mapping**: 30GB mmap_size for large databases

**Current Epic Schema** (db.go:71-94):
```
epics table:
- id (INTEGER PRIMARY KEY AUTOINCREMENT)
- key (TEXT NOT NULL UNIQUE)
- title (TEXT NOT NULL)
- description (TEXT)
- status (TEXT NOT NULL CHECK: draft|active|completed|archived)
- priority (TEXT NOT NULL CHECK: high|medium|low)
- business_value (TEXT CHECK: high|medium|low)
- created_at (TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)
- updated_at (TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)

Indexes:
- idx_epics_key (UNIQUE on key)
- idx_epics_status (on status)

Triggers:
- epics_updated_at (auto-update updated_at on UPDATE)
```

**Current Feature Schema** (db.go:98-124):
```
features table:
- id (INTEGER PRIMARY KEY AUTOINCREMENT)
- epic_id (INTEGER NOT NULL, FK to epics)
- key (TEXT NOT NULL UNIQUE)
- title (TEXT NOT NULL)
- description (TEXT)
- status (TEXT NOT NULL CHECK: draft|active|completed|archived)
- progress_pct (REAL NOT NULL DEFAULT 0.0 CHECK: 0-100)
- execution_order (INTEGER NULL)
- created_at (TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)
- updated_at (TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)

Indexes:
- idx_features_key (UNIQUE on key)
- idx_features_epic_id (on epic_id)
- idx_features_status (on status)

Triggers:
- features_updated_at (auto-update updated_at on UPDATE)
```

**Task Schema** (db.go:129-167):
```
tasks table includes:
- file_path (TEXT) - custom file location for tasks
- Index: idx_tasks_file_path (on file_path)
```

### 1.2 Schema Extension Requirements

Per E07-F08 PRD (prd.md:249-270), the following schema changes are needed:

**E07-F08 (Custom Filenames)**:
- Add `file_path TEXT` column to `epics` table
- Add `file_path TEXT` column to `features` table
- Add indexes: `idx_epics_file_path`, `idx_features_file_path`

**E07-F09 (Custom Folder Base Paths - this feature)**:
- Add `custom_folder_path TEXT` column to `epics` table
- Add `custom_folder_path TEXT` column to `features` table
- Add indexes: `idx_epics_custom_folder_path`, `idx_features_custom_folder_path`

**Column Semantics**:
- `file_path`: Exact file location for this entity (E07-F08)
- `custom_folder_path`: Base folder for this entity and all children (E07-F09)
- Both nullable (NULL = default behavior)

### 1.3 Migration Strategy

**Pattern Observed**:
- Schema is created via `createSchema()` function in db.go
- Uses `CREATE TABLE IF NOT EXISTS` for idempotent schema creation
- No explicit migration files; schema evolves in-place
- `shark init` creates fresh databases with current schema

**Recommendation**:
- Update `createSchema()` to include new columns
- Provide `ALTER TABLE` statements in documentation for existing databases
- Maintain backward compatibility via NULL defaults

---

## 2. Data Models

### 2.1 Current Model Structure

**Epic Model** (internal/models/epic.go:25-35):
```go
type Epic struct {
    ID            int64      `json:"id" db:"id"`
    Key           string     `json:"key" db:"key"`
    Title         string     `json:"title" db:"title"`
    Description   *string    `json:"description,omitempty" db:"description"`
    Status        EpicStatus `json:"status" db:"status"`
    Priority      Priority   `json:"priority" db:"priority"`
    BusinessValue *Priority  `json:"business_value,omitempty" db:"business_value"`
    CreatedAt     time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}
```

**Feature Model** (internal/models/feature.go:16-27):
```go
type Feature struct {
    ID             int64         `json:"id" db:"id"`
    EpicID         int64         `json:"epic_id" db:"epic_id"`
    Key            string        `json:"key" db:"key"`
    Title          string        `json:"title" db:"title"`
    Description    *string       `json:"description,omitempty" db:"description"`
    Status         FeatureStatus `json:"status" db:"status"`
    ProgressPct    float64       `json:"progress_pct" db:"progress_pct"`
    ExecutionOrder *int          `json:"execution_order,omitempty" db:"execution_order"`
    CreatedAt      time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
}
```

**Task Model** (internal/models/task.go:44):
```go
type Task struct {
    // ... other fields ...
    FilePath       *string         `json:"file_path,omitempty" db:"file_path"`
}
```

### 2.2 Model Extension Requirements

**Add to Epic struct**:
- `FilePath *string` (E07-F08)
- `CustomFolderPath *string` (E07-F09)

**Add to Feature struct**:
- `FilePath *string` (E07-F08)
- `CustomFolderPath *string` (E07-F09)

**Naming Convention**:
- Pointer types for nullable fields (observed pattern)
- JSON tags use snake_case with omitempty for optional fields
- DB tags match column names exactly

### 2.3 Validation Patterns

**Epic Validation** (epic.go:38-57):
- Uses dedicated `Validate()` method
- Calls helper functions: `ValidateEpicKey()`, `ValidateEpicStatus()`, `ValidatePriority()`
- Returns descriptive errors

**Feature Validation** (feature.go:30-44):
- Similar pattern with `Validate()` method
- Validates key, title, status, progress_pct range

**Recommendation**:
- Add path validation to models via new helper: `ValidateCustomPath(path, projectRoot string) error`

---

## 3. Repository Layer

### 3.1 Repository Pattern

**Epic Repository** (internal/repository/epic_repository.go:11-19):
```go
type EpicRepository struct {
    db *DB
}

func NewEpicRepository(db *DB) *EpicRepository {
    return &EpicRepository{db: db}
}
```

**Method Signature Pattern** (epic_repository.go:22-51):
- Accept `ctx context.Context` as first parameter
- Use explicit SQL queries (no ORM)
- Return descriptive errors with `fmt.Errorf("context: %w", err)`
- Use `QueryRowContext` and `ExecContext` for context cancellation support

**Example - Create Method** (epic_repository.go:22-51):
```go
func (r *EpicRepository) Create(ctx context.Context, epic *models.Epic) error {
    if err := epic.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    query := `INSERT INTO epics (key, title, description, ...) VALUES (?, ?, ?, ...)`

    result, err := r.db.ExecContext(ctx, query, epic.Key, epic.Title, ...)
    if err != nil {
        return fmt.Errorf("failed to create epic: %w", err)
    }

    id, _ := result.LastInsertId()
    epic.ID = id
    return nil
}
```

### 3.2 Repository Extension Requirements

**New Methods Needed for Epic Repository**:
- `GetCustomFolderPath(ctx, epicKey) (*string, error)` - Retrieve custom folder path
- Update `Create()` to accept `FilePath` and `CustomFolderPath` fields
- Update `GetByKey()` to SELECT new columns

**New Methods Needed for Feature Repository**:
- `GetCustomFolderPath(ctx, featureKey) (*string, error)` - Retrieve custom folder path
- Update `Create()` to accept `FilePath` and `CustomFolderPath` fields
- Update `GetByKey()` to SELECT new columns

---

## 4. CLI Architecture

### 4.1 CLI Framework

**Framework**: Cobra (github.com/spf13/cobra)

**Root Command Pattern** (internal/cli/commands/epic.go:34-43):
```go
var epicCmd = &cobra.Command{
    Use:   "epic",
    Short: "Manage epics",
    Long: `Query and manage epics with automatic progress calculation.`,
}

func init() {
    cli.RootCmd.AddCommand(epicCmd)
    epicCmd.AddCommand(epicListCmd)
    epicCmd.AddCommand(epicGetCmd)
    epicCmd.AddCommand(epicCreateCmd)
}
```

### 4.2 Flag Management

**Flag Declaration Pattern** (epic.go:113-136):
```go
var (
    epicCreateDescription string
)

func init() {
    // Add flags for create command
    epicCreateCmd.Flags().StringVar(&epicCreateDescription, "description", "", "Epic description (optional)")
    epicDeleteCmd.Flags().Bool("force", false, "Force deletion even if epic has features")
}
```

**Flag Retrieval Pattern** (epic.go:145-147):
```go
sortBy, _ := cmd.Flags().GetString("sort-by")
statusFilter, _ := cmd.Flags().GetString("status")
```

### 4.3 Output Management

**JSON Output Support**:
- Global flag: `--json` (handled by cli.GlobalConfig.JSON)
- Commands check this flag to format output appropriately
- Helper functions: `cli.OutputJSON(data)`, `cli.OutputTable(headers, rows)`

**Error Handling Pattern** (epic.go:160-163):
```go
if !valid {
    cli.Error(fmt.Sprintf("Error: Invalid status '%s'. ...", statusFilter))
    os.Exit(1)
}
```

### 4.4 CLI Extension Requirements

**Epic Create Command**:
- Add `--path` flag (type: string, optional)
- Validate path using `ValidateCustomPath()`
- Pass to epic creator via `Epic.CustomFolderPath`

**Feature Create Command**:
- Add `--path` flag (type: string, optional)
- Fetch parent epic's `custom_folder_path` from database
- Pass to feature creator via `Feature.CustomFolderPath`

---

## 5. Path Validation Infrastructure

### 5.1 Existing Validation Patterns

**Pattern Validator** (internal/patterns/validator.go:1-340):
- Validates regex patterns for catastrophic backtracking
- Extracts and validates capture groups
- Provides timeout-based validation to prevent DoS
- Returns `ValidationError` with detailed context

**Validation Functions**:
- `ValidatePattern(pattern, entityType string) error`
- `ValidatePatternSyntaxOnly(pattern string) error`
- `ValidatePatternConfig(config *PatternConfig) error`
- `ValidateWithTimeout(pattern, entityType string, timeout time.Duration) error`

### 5.2 Path Security Requirements

Per PRD (prd.md:380-394), path validation must:
1. Reject absolute paths
2. Prevent path traversal (`..`)
3. Ensure paths resolve within project root
4. Reject empty/whitespace paths
5. Normalize trailing slashes

**Recommended Implementation**:
Create `internal/utils/path_validation.go` with:
- `ValidateFolderPath(path, projectRoot string) (absPath, relPath string, err error)`
- Reuse security patterns from task filename validation (E07-F05)

---

## 6. Sync and Discovery

### 6.1 Current Sync Architecture

**Sync Command** (internal/cli/commands/sync.go):
- Scans `docs/plan/` directory for epic/feature/task files
- Parses YAML frontmatter
- Synchronizes with database
- Handles conflicts via `--strategy` flag (file-wins | db-wins)

**Discovery Patterns**:
- Assumes fixed directory structure: `docs/plan/<epic-key>/<feature-key>/tasks/`
- Uses pattern matching from `internal/patterns/` package

### 6.2 Sync Extension Requirements

**Critical Changes Needed**:
1. **Scan Beyond Default Locations**: Look for epic/feature files in custom paths
2. **Parse Custom Paths from Frontmatter**: Extract `custom_folder_path` from YAML
3. **Multi-Location Discovery**: Check both default and custom path locations
4. **Conflict Resolution**: Handle file vs. database custom_folder_path differences
5. **Frontmatter Preservation**: Write `custom_folder_path` to frontmatter when creating/updating

**Path Resolution During Sync**:
- Epic discovered at custom path → store `custom_folder_path` in DB
- Feature discovered → check parent epic's custom path for resolution
- Task discovered → check feature and epic custom paths for resolution

---

## 7. Path Resolution Logic

### 7.1 Resolution Precedence (from PRD)

**For Epics**:
1. `--filename` (exact path) → use as-is
2. `--path` (custom folder) → `<custom_folder_path>/<epic-key>/epic.md`
3. Default → `docs/plan/<epic-key>/epic.md`

**For Features**:
1. `--filename` (exact path) → use as-is
2. Feature `--path` → `<feature-custom-path>/<feature-key>/feature.md`
3. Parent epic's `custom_folder_path` → `<epic-custom-path>/<epic-key>/<feature-key>/feature.md`
4. Default → `docs/plan/<epic-key>/<feature-key>/feature.md`

**For Tasks**:
1. `--filename` (exact path) → use as-is
2. Feature `custom_folder_path` → `<feature-custom-path>/<feature-key>/tasks/<task-key>.md`
3. Epic `custom_folder_path` → `<epic-custom-path>/<epic-key>/<feature-key>/tasks/<task-key>.md`
4. Default → `docs/plan/<epic-key>/<feature-key>/tasks/<task-key>.md`

### 7.2 Path Builder Service Design

**Proposed Location**: `internal/utils/path_builder.go`

**Service Structure**:
```go
type PathBuilder struct {
    projectRoot string
}

// Methods:
- ResolveEpicPath(epicKey, filename, customFolderPath) (string, error)
- ResolveFeaturePath(epicKey, featureKey, filename, featureCustomPath, epicCustomPath) (string, error)
- ResolveTaskPath(epicKey, featureKey, taskKey, filename, featureCustomPath, epicCustomPath) (string, error)
```

**Design Rationale**:
- Centralizes all path resolution logic (single source of truth)
- Reusable across epic, feature, task creation and sync
- Testable in isolation with comprehensive unit tests
- Encapsulates complex precedence rules

---

## 8. Technology Stack Summary

| Layer | Technology | Notes |
|-------|------------|-------|
| **Language** | Go 1.23.4+ | Statically typed, compiled |
| **Database** | SQLite | WAL mode, foreign keys enabled |
| **CLI Framework** | Cobra | Command hierarchy, flag parsing |
| **Config Management** | Viper | Config file handling (not directly used in commands) |
| **Testing** | Go testing | stdlib with testify for assertions |
| **Validation** | Regex + custom | Pattern matching, security checks |

---

## 9. File Structure Conventions

### 9.1 Current Conventions

**Default File Paths**:
- Epic: `docs/plan/<epic-key>/epic.md`
- Feature: `docs/plan/<epic-key>/<feature-key>/feature.md`
- Task: `docs/plan/<epic-key>/<feature-key>/tasks/<task-key>.md`

**Frontmatter Format** (YAML):
```yaml
---
epic_key: E04
title: Task Management CLI Core
description: Core CLI functionality
status: active
priority: high
business_value: high
---

# Epic Content
```

### 9.2 Extended Frontmatter

**Epic with Custom Folder Path**:
```yaml
---
epic_key: E09
title: Q1 2025 Roadmap
custom_folder_path: docs/roadmap/2025-q1
file_path: docs/roadmap/2025-q1/E09/epic.md
---
```

**Feature with Custom Folder Path**:
```yaml
---
feature_key: E04-F01
epic_key: E04
title: OAuth Module
custom_folder_path: docs/plan/E04-auth/modules/oauth
file_path: docs/plan/E04-auth/modules/oauth/E04-F01-oauth-module/feature.md
---
```

---

## 10. Dependencies and Integration Points

### 10.1 Prerequisite Features

**E07-F08 (Custom Filenames for Epics & Features)**: REQUIRED
- Adds `file_path` column to epics/features tables
- Provides `--filename` flag infrastructure
- Establishes path validation patterns
- Must be complete before E07-F09

**E07-F05 (Custom Filenames for Tasks)**: REQUIRED
- Provides task `--filename` functionality
- Establishes precedence for exact path overrides
- Must be complete before E07-F09

### 10.2 Internal Package Dependencies

**Will Create**:
- `internal/utils/path_builder.go` - Path resolution service
- `internal/utils/path_validation.go` - Path security validation

**Will Modify**:
- `internal/db/db.go` - Add custom_folder_path columns
- `internal/models/epic.go` - Add CustomFolderPath field
- `internal/models/feature.go` - Add CustomFolderPath field
- `internal/repository/epic_repository.go` - Add GetCustomFolderPath method
- `internal/repository/feature_repository.go` - Add GetCustomFolderPath method
- `internal/cli/commands/epic.go` - Add --path flag
- `internal/cli/commands/feature.go` - Add --path flag
- `internal/cli/commands/task.go` - Use PathBuilder for resolution
- `internal/sync/` - Update discovery to scan custom paths

---

## 11. Naming Conventions

### 11.1 Observed Patterns

**Database**:
- Tables: plural nouns (epics, features, tasks)
- Columns: snake_case (epic_id, custom_folder_path)
- Indexes: `idx_<table>_<column>`
- Triggers: `<table>_updated_at`

**Go Code**:
- Types: PascalCase (Epic, Feature, Task)
- Fields: PascalCase (CustomFolderPath, FilePath)
- Methods: PascalCase (GetCustomFolderPath, Create)
- Variables: camelCase (epicKey, customPath)
- Packages: lowercase (repository, models, utils)

**CLI**:
- Commands: lowercase with hyphens (epic create, feature list)
- Flags: lowercase with hyphens (--custom-path, --file-path)
- Flag variables: camelCase (epicCreateDescription)

**JSON**:
- Keys: snake_case with omitempty for optionals
- Example: `{"custom_folder_path": "docs/custom", "file_path": "docs/custom/E09/epic.md"}`

### 11.2 Recommended Naming for E07-F09

**Flag Name**: `--path` (short, clear, distinct from `--filename`)
**Database Column**: `custom_folder_path` (explicit, matches pattern)
**Model Field**: `CustomFolderPath *string`
**JSON Key**: `custom_folder_path`
**Service**: `PathBuilder` struct with `Resolve<Entity>Path` methods

---

## 12. Error Handling Patterns

### 12.1 Error Conventions

**Repository Layer**:
```go
return fmt.Errorf("failed to create epic: %w", err)
return fmt.Errorf("epic not found with id %d", id)
```

**CLI Layer**:
```go
cli.Error(fmt.Sprintf("Error: Invalid status '%s'. Must be one of: ...", statusFilter))
os.Exit(1)
```

**Validation Layer**:
```go
return &ValidationError{
    Pattern: pattern,
    EntityType: entityType,
    Message: fmt.Sprintf("invalid regex syntax: %v", err),
}
```

### 12.2 Error Messages for E07-F09

**Path Validation Errors** (from PRD prd.md:380-393):
- Absolute path: `"path must be relative to project root, got absolute path: /..."`
- Path traversal: `"invalid path: contains '..' (path traversal not allowed)"`
- Outside project: `"path validation failed: resolves outside project root"`
- Empty path: `"invalid path: resolved to empty or invalid path"`

---

## 13. Testing Strategy

### 13.1 Testing Infrastructure

**Test Patterns Observed**:
- Test files alongside implementation: `foo.go` + `foo_test.go`
- Test database: `internal/repository/test-shark-tasks.db`
- Table-driven tests for validation
- Integration tests with real SQLite database

### 13.2 Testing Requirements for E07-F09

**Unit Tests**:
- `internal/utils/path_builder_test.go` - Path resolution logic
- `internal/utils/path_validation_test.go` - Security validation
- Model validation tests in `internal/models/`

**Integration Tests**:
- Epic creation with `--path` flag
- Feature creation inheriting epic custom path
- Feature creation overriding epic custom path
- Task creation with custom path inheritance
- Sync discovering items in custom paths
- Conflict resolution for custom_folder_path

**Edge Cases**:
- Empty path string
- Absolute paths (rejected)
- Path traversal attempts (rejected)
- Paths outside project root (rejected)
- Both `--filename` and `--path` provided (filename wins)
- NULL vs. empty string in database

---

## 14. Performance Considerations

### 14.1 Database Performance

**Current Optimizations**:
- Indexes on frequently queried columns (key, status, file_path)
- WAL mode for concurrent reads
- 64MB cache size
- Foreign key cascades for deletion efficiency

**Required for E07-F09**:
- Add indexes: `idx_epics_custom_folder_path`, `idx_features_custom_folder_path`
- Lookup performance: O(1) for exact custom_folder_path queries
- Inheritance queries: requires JOIN to parent epic/feature (acceptable for infrequent operation)

### 14.2 Sync Performance

**Current Behavior**:
- Single-pass directory scan
- Pattern matching via regex
- Frontmatter parsing for all files

**Impact of E07-F09**:
- May require multiple scan locations (default + custom paths)
- PathBuilder adds negligible overhead (simple string operations)
- Frontmatter parsing unchanged (just add one field)

**Mitigation**:
- Build path map during discovery phase (cache parent custom paths)
- Avoid redundant filesystem scans by tracking visited directories

---

## 15. Security Considerations

### 15.1 Path Traversal Prevention

**Requirements**:
- Reject absolute paths
- Reject paths containing `..`
- Validate resolved paths stay within project root
- Normalize paths before validation

**Implementation Location**: `internal/utils/path_validation.go`

**Reference**: Similar validation exists for task filenames (E07-F05)

### 15.2 Database Injection Prevention

**Current Protection**:
- Parameterized queries with `?` placeholders
- No string concatenation in SQL
- Example: `query := "SELECT ... WHERE key = ?"` + `r.db.QueryRowContext(ctx, query, key)`

**No Changes Needed**: Continue existing pattern for custom_folder_path queries

---

## 16. Backward Compatibility

### 16.1 Schema Compatibility

**Strategy**:
- New columns are nullable (NULL = default behavior)
- Existing records get NULL values (default paths)
- No data migration required
- Existing queries unaffected (don't SELECT new columns unless needed)

### 16.2 CLI Compatibility

**Strategy**:
- `--path` flag is optional
- Without flag, behavior is identical to current
- Existing scripts and workflows continue unchanged
- JSON output includes new fields with `omitempty` (absent when NULL)

### 16.3 File Format Compatibility

**Strategy**:
- Frontmatter fields are optional
- Sync reads `custom_folder_path` if present, ignores if absent
- Files without custom_folder_path use default locations
- Old files remain valid and functional

---

## 17. Recommendations

### 17.1 Implementation Priorities

**Phase 1 - Foundation** (MUST):
1. Add `custom_folder_path` column to epics/features tables
2. Update Epic and Feature models
3. Update repository Create/Get methods
4. Add path validation utility

**Phase 2 - Path Resolution** (MUST):
1. Create PathBuilder service
2. Implement path resolution algorithms
3. Write comprehensive unit tests

**Phase 3 - CLI Integration** (MUST):
1. Add `--path` flag to epic/feature create commands
2. Integrate PathBuilder in creation flow
3. Update command help text

**Phase 4 - Sync Integration** (CRITICAL):
1. Update discovery to scan custom path locations
2. Parse custom_folder_path from frontmatter
3. Handle multi-location task discovery
4. Implement conflict resolution

**Phase 5 - Testing & Docs** (MUST):
1. Integration tests for all scenarios
2. Update CLI_REFERENCE.md
3. Update CLAUDE.md with examples

### 17.2 Design Decisions

**Use `--path` instead of `--base-path` or `--folder`**:
- Shorter, clearer, mirrors `--filename`
- Distinct from `--filename` (exact vs. base)

**Use `custom_folder_path` instead of `base_path`**:
- Explicit and descriptive
- Matches existing `file_path` naming pattern
- Clear that it's a folder, not a file

**Use PathBuilder service instead of scattered logic**:
- Single source of truth for path resolution
- Testable in isolation
- Easier to maintain and debug
- Encapsulates complexity

**Separate path validation from PathBuilder**:
- Security validation is distinct concern
- Reusable across different path contexts
- Allows independent testing

### 17.3 Risk Mitigations

**Risk**: Confusion between `--path` and `--filename`
- **Mitigation**: Clear documentation, help text, error messages

**Risk**: Complex inheritance rules cause bugs
- **Mitigation**: Centralize logic in PathBuilder, comprehensive tests

**Risk**: Sync performance degradation with many custom paths
- **Mitigation**: Build path map during discovery, avoid redundant scans

**Risk**: Database migration issues for existing projects
- **Mitigation**: Provide clear migration guide, use ALTER TABLE in docs

---

## 18. References

**Codebase Files Analyzed**:
- `internal/db/db.go` - Database schema and configuration
- `internal/models/epic.go` - Epic data model
- `internal/models/feature.go` - Feature data model
- `internal/models/task.go` - Task data model (file_path reference)
- `internal/repository/epic_repository.go` - Epic CRUD operations
- `internal/cli/commands/epic.go` - Epic CLI commands
- `internal/patterns/validator.go` - Pattern validation infrastructure

**Related Features**:
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` - Custom filenames (prerequisite)
- `docs/plan/E07-enhancements/E07-F09-custom-folder-base-paths/prd.md` - This feature's requirements

**Documentation**:
- `README.md` - Project overview and usage
- `CLAUDE.md` - Development guidelines and conventions
- `docs/CLI_REFERENCE.md` - CLI command documentation

---

## Appendix A: Code Examples

### A.1 Database Column Addition

```sql
-- Epic table extension
ALTER TABLE epics ADD COLUMN custom_folder_path TEXT;
CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);

-- Feature table extension
ALTER TABLE features ADD COLUMN custom_folder_path TEXT;
CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);
```

### A.2 Model Extension

```go
// Epic model with new field
type Epic struct {
    // ... existing fields ...
    FilePath           *string    `json:"file_path,omitempty" db:"file_path"`
    CustomFolderPath   *string    `json:"custom_folder_path,omitempty" db:"custom_folder_path"`
}
```

### A.3 Repository Method Signature

```go
// GetCustomFolderPath retrieves the custom folder path for an epic
// Returns nil if no custom path is set
func (r *EpicRepository) GetCustomFolderPath(ctx context.Context, epicKey string) (*string, error) {
    query := `SELECT custom_folder_path FROM epics WHERE key = ?`
    var path *string
    err := r.db.QueryRowContext(ctx, query, epicKey).Scan(&path)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("epic not found: %s", epicKey)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get custom folder path: %w", err)
    }
    return path, nil
}
```

### A.4 PathBuilder Service Signature

```go
package utils

type PathBuilder struct {
    projectRoot string
}

func NewPathBuilder(projectRoot string) *PathBuilder {
    return &PathBuilder{projectRoot: projectRoot}
}

// ResolveEpicPath determines the file path for an epic
// Priority: filename > custom_folder_path > default
func (pb *PathBuilder) ResolveEpicPath(
    epicKey string,
    filename *string,           // from --filename flag
    customFolderPath *string,   // from --path flag
) (string, error) {
    // Implementation...
}
```

---

**End of Research Report**
