# Epic E07 Enhancements - Comprehensive Architecture Review

**Review Date**: 2025-12-18
**Reviewer**: TechDirector Agent
**Epic**: E07 - Enhancements
**Scope**: All 7 features architectural assessment

---

## Executive Summary

This document provides architectural review and design recommendations for all 7 enhancement features in Epic E07. The features range from simple CLI improvements to complex database schema changes. All features are technically feasible with proper design.

**Overall Risk Assessment**: LOW to MEDIUM
**Implementation Order Recommendation**: F02 → F01 → F03 → F06 → F04 → F05 → F07

---

## Feature E07-F01: Remove Agent Requirement

### Feature Summary
Make the `--agent` flag optional when creating tasks. Allow any string value for agent type (not just predefined set). Support custom templates via `--template` flag.

### Feasibility Rating: **HIGH**

**Justification**:
- Simple flag modification in Cobra command
- Database already supports NULL and arbitrary strings for agent_type field
- Template logic is straightforward conditional branching

### Architectural Concerns

**None identified**. This is a low-risk change that reduces constraints rather than adding complexity.

### Recommended Design Approach

**1. Command Flag Changes** (`internal/cli/commands/task.go`):
```go
// Remove MarkFlagRequired("agent")
// Agent flag remains optional
taskCreateCmd.Flags().StringP("agent", "a", "", "Agent type (optional)")
taskCreateCmd.Flags().StringP("template", "t", "", "Custom template path (optional)")
```

**2. Template Selection Logic** (`internal/taskcreation/creator.go`):
```go
func (c *Creator) selectTemplate(agentType, customTemplate string) (string, error) {
    // Priority: custom template > agent-specific > general
    if customTemplate != "" {
        return c.validateAndLoadCustomTemplate(customTemplate)
    }

    if agentType != "" {
        if template := c.getAgentTemplate(agentType); template != "" {
            return template, nil
        }
    }

    // Fallback to general template
    return c.getGeneralTemplate(), nil
}
```

**3. Validation Changes**:
- Remove enum validation for agent type
- Add file existence check for custom template paths
- Ensure NULL handling in database queries

### Key Implementation Considerations

1. **Backward Compatibility**: Existing tasks with agent types must work unchanged
2. **User Feedback**: Clear messaging when falling back to general template
3. **Template Path Security**: Validate template paths to prevent directory traversal
4. **Database Schema**: Verify agent_type column allows NULL (should already)

### Dependencies and Integration Points

- `internal/cli/commands/task.go` - Command definition
- `internal/taskcreation/creator.go` - Template selection logic
- Database schema validation (no changes needed, verify TEXT type)

### Complexity Estimate: **SIMPLE**

Estimated effort: 4-6 hours including tests

### Risk Assessment: **LOW**

- No database migration required
- No breaking changes to existing functionality
- Straightforward logic changes

---

## Feature E07-F02: Make Task Title Like Feature

### Feature Summary
Change task creation to accept title as positional argument instead of `--title` flag, matching epic/feature command patterns.

### Feasibility Rating: **HIGH**

**Justification**:
- Simple argument parsing change
- Cobra supports positional args natively
- Minimal code changes required

### Architectural Concerns

**Breaking Change**: This changes existing CLI syntax. Users must update scripts/workflows.

**Mitigation**:
- Clear deprecation notice in release notes
- Helpful error messages for old syntax
- Version this as breaking change (v2.0.0 if following semver)

### Recommended Design Approach

**Command Definition Change** (`internal/cli/commands/task.go`):
```go
var taskCreateCmd = &cobra.Command{
    Use:   "create <title> [flags]",
    Short: "Create a new task",
    Long:  "Create a new task with the given title...",
    Args:  cobra.ExactArgs(1), // Require exactly one positional arg
    Example: `  shark task create "Build login form" --epic=E01 --feature=F02 --agent=frontend
  shark task create "Fix auth bug" --epic=E01 --feature=F03`,
    RunE: runTaskCreate,
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
    title := args[0] // Get title from positional arg

    // Get other flags
    epicKey, _ := cmd.Flags().GetString("epic")
    featureKey, _ := cmd.Flags().GetString("feature")
    agentType, _ := cmd.Flags().GetString("agent")

    // Proceed with task creation...
}
```

**Remove Title Flag**:
```go
// DELETE: taskCreateCmd.Flags().StringP("title", "t", "", "Task title")
// DELETE: taskCreateCmd.MarkFlagRequired("title")
```

### Key Implementation Considerations

1. **Consistency Check**: Verify epic and feature commands use same pattern
2. **Error Messages**: Provide helpful usage examples when args missing
3. **Test Updates**: Update all tests using old --title syntax
4. **Documentation**: Update README, help text, examples

### Dependencies and Integration Points

- `internal/cli/commands/task.go` - Sole point of change
- No dependency on other E07 features
- **Can be implemented independently**

### Complexity Estimate: **SIMPLE**

Estimated effort: 2-3 hours including tests and docs

### Risk Assessment: **LOW**

- Breaking change but simple to communicate and fix
- No database changes
- Limited scope of impact

---

## Feature E07-F03: Move Task Files to Feature Folder

### Feature Summary
Create task files under feature folder in `tasks/` subdirectory: `docs/plan/{epic-slug}/{feature-slug}/tasks/T-{epic-key}-{feature-key}-{number}.md`

### Feasibility Rating: **HIGH**

**Justification**:
- File system operation only
- Database stores file paths, just need to update path generation
- No schema changes required

### Architectural Concerns

**Existing Tasks**: Old task files remain in their current locations. No migration.

**Path Resolution**: Need to resolve epic/feature slugs to build correct folder path.

### Recommended Design Approach

**1. Path Resolution** (`internal/taskcreation/creator.go`):
```go
func (c *Creator) resolveTaskFilePath(epicKey, featureKey, taskSlug string, taskNumber int) (string, error) {
    // Get epic and feature from database to get their slugs
    epic, err := c.epicRepo.GetByKey(ctx, epicKey)
    if err != nil {
        return "", fmt.Errorf("epic not found: %w", err)
    }

    feature, err := c.featureRepo.GetByKey(ctx, featureKey)
    if err != nil {
        return "", fmt.Errorf("feature not found: %w", err)
    }

    // Build path: docs/plan/{epic-slug}/{feature-slug}/tasks/
    basePath := filepath.Join("docs", "plan", epic.Slug, feature.Slug, "tasks")

    // Ensure directory exists
    if err := os.MkdirAll(basePath, 0755); err != nil {
        return "", fmt.Errorf("failed to create tasks directory: %w", err)
    }

    // Generate filename
    filename := fmt.Sprintf("T-%s-%s-%03d-%s.md", epicKey, featureKey, taskNumber, taskSlug)

    return filepath.Join(basePath, filename), nil
}
```

**2. Slug Storage**: Need epic.Slug and feature.Slug in database

**Option A**: Query epic.md/prd.md frontmatter for slug
**Option B**: Derive slug from directory name
**Option C**: Store slug in database (recommended)

### Key Implementation Considerations

1. **Slug Availability**: Ensure epic/feature records have slug information
2. **Directory Creation**: Auto-create `tasks/` subdirectory
3. **Error Handling**: Clear errors if epic/feature folder doesn't exist
4. **Backward Compatibility**: Existing task file paths remain valid

### Dependencies and Integration Points

- `internal/taskcreation/creator.go` - Path generation logic
- `internal/repository/epic_repository.go` - Need to query epic for slug
- `internal/repository/feature_repository.go` - Need to query feature for slug
- **Dependency on F03**: May need slug field in database (or parse from filesystem)

### Complexity Estimate: **MODERATE**

Estimated effort: 8-10 hours including slug resolution and tests

### Risk Assessment: **MEDIUM**

- Slug availability uncertainty (may need discovery package integration)
- Potential for missing folder errors if epic/feature not synced
- Need robust error handling

**Recommendation**: Implement after E07-F07 (discovery integration) to leverage slug information from filesystem.

---

## Feature E07-F04: Implementation Order

### Feature Summary
Add `execution_order` field at feature and task levels to specify recommended implementation sequence.

### Feasibility Rating: **HIGH**

**Justification**:
- Simple database schema addition
- Nullable integer field
- Query modifications straightforward

### Architectural Concerns

**Schema Migration**: Requires database migration to add new column.

**Sorting Logic**: Need to handle NULL values in ORDER BY clauses.

### Recommended Design Approach

**1. Database Migration**:
```sql
-- Migration: Add execution_order to features table
ALTER TABLE features ADD COLUMN execution_order INTEGER NULL;

-- Migration: Add execution_order to tasks table
ALTER TABLE tasks ADD COLUMN execution_order INTEGER NULL;

-- Indexes for performance (optional)
CREATE INDEX idx_features_execution_order ON features(execution_order);
CREATE INDEX idx_tasks_execution_order ON tasks(execution_order);
```

**2. Model Updates** (`internal/models/`):
```go
type Feature struct {
    // ... existing fields ...
    ExecutionOrder *int `json:"execution_order,omitempty" db:"execution_order"`
}

type Task struct {
    // ... existing fields ...
    ExecutionOrder *int `json:"execution_order,omitempty" db:"execution_order"`
}
```

**3. Repository Query Updates**:
```go
func (r *TaskRepository) ListByFeature(ctx context.Context, featureKey string, sortBy string) ([]*models.Task, error) {
    query := `SELECT * FROM tasks WHERE feature_key = ?`

    switch sortBy {
    case "order":
        query += ` ORDER BY execution_order NULLS LAST, created_at ASC`
    case "status":
        query += ` ORDER BY status, created_at ASC`
    default:
        query += ` ORDER BY created_at ASC`
    }

    // Execute query...
}
```

**4. CLI Flag** (`internal/cli/commands/task.go`):
```go
taskListCmd.Flags().String("sort-by", "created", "Sort by: created, order, status, priority")
```

### Key Implementation Considerations

1. **Migration Safety**: Ensure migration is reversible
2. **NULL Handling**: SQLite NULLS LAST syntax support (verify version)
3. **Validation**: Execution order should be positive integers
4. **Update Commands**: Need commands to set/update execution order

### Dependencies and Integration Points

- Database migration script
- `internal/models/` - Model updates
- `internal/repository/` - Query modifications
- `internal/cli/commands/` - Sort flag additions
- **No dependencies on other E07 features**

### Complexity Estimate: **MODERATE**

Estimated effort: 6-8 hours including migration and tests

### Risk Assessment: **LOW**

- Well-understood pattern (adding nullable column)
- No breaking changes
- Backward compatible (NULL means no order specified)

---

## Feature E07-F05: Add Related Documents

### Feature Summary
Create `documents` table and junction tables (`epic_documents`, `feature_documents`, `task_documents`) to link supporting documents to work items. Provide CLI commands for management.

### Feasibility Rating: **HIGH**

**Justification**:
- Standard relational database pattern
- Many-to-many relationships via junction tables
- No complex logic required

### Architectural Concerns

**Schema Complexity**: Adds 4 new tables and foreign key relationships.

**Document Lifecycle**: Documents persist even if all links removed (by design).

**Path Validation**: Should paths be validated at link time?

### Recommended Design Approach

**1. Database Schema**:
```sql
-- Documents table
CREATE TABLE documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, file_path)
);

-- Junction tables
CREATE TABLE epic_documents (
    epic_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (epic_id, document_id),
    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE TABLE feature_documents (
    feature_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (feature_id, document_id),
    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE TABLE task_documents (
    task_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (task_id, document_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);
```

**2. Repository Layer** (`internal/repository/document_repository.go`):
```go
type DocumentRepository struct {
    db *sql.DB
}

func (r *DocumentRepository) CreateOrGet(ctx context.Context, title, path string) (*models.Document, error)
func (r *DocumentRepository) LinkToEpic(ctx context.Context, docID, epicID int64) error
func (r *DocumentRepository) LinkToFeature(ctx context.Context, docID, featureID int64) error
func (r *DocumentRepository) LinkToTask(ctx context.Context, docID, taskID int64) error
func (r *DocumentRepository) UnlinkFromEpic(ctx context.Context, docID, epicID int64) error
func (r *DocumentRepository) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error)
// ... similar for Feature and Task
```

**3. CLI Commands** (`internal/cli/commands/related_docs.go`):
```go
// New command group: shark related-docs
var relatedDocsCmd = &cobra.Command{
    Use:   "related-docs",
    Short: "Manage related documents",
}

var addRelatedDocCmd = &cobra.Command{
    Use:   "add <title> <path>",
    Args:  cobra.ExactArgs(2),
    RunE:  runAddRelatedDoc,
}

var listRelatedDocsCmd = &cobra.Command{
    Use:   "list",
    RunE:  runListRelatedDocs,
}

var deleteRelatedDocCmd = &cobra.Command{
    Use:   "delete <title>",
    Args:  cobra.ExactArgs(1),
    RunE:  runDeleteRelatedDoc,
}
```

### Key Implementation Considerations

1. **Upsert Logic**: CreateOrGet should return existing document if title+path match
2. **Cascade Deletes**: ON DELETE CASCADE ensures orphan cleanup
3. **Transaction Management**: Link operations should be atomic
4. **Path Validation**: Optional --no-validate flag to skip file existence check
5. **JSON Output**: Support --json for all list commands

### Dependencies and Integration Points

- New file: `internal/repository/document_repository.go`
- New file: `internal/models/document.go`
- New file: `internal/cli/commands/related_docs.go`
- Database migration script
- **No dependencies on other E07 features**

### Complexity Estimate: **COMPLEX**

Estimated effort: 12-16 hours including schema, repositories, CLI, and tests

### Risk Assessment: **MEDIUM**

- Most complex schema change in E07
- Foreign key constraints require careful migration
- Multiple integration points (epics, features, tasks)

**Recommendation**: Implement after simpler features to leverage patterns established.

---

## Feature E07-F06: Allow Force Status on Task

### Feature Summary
Add `--force` flag to task and feature status update commands to bypass status flow validation. For features, force updates all child tasks.

### Feasibility Rating: **HIGH**

**Justification**:
- Flag-based conditional logic
- Existing status update mechanisms can be modified
- No schema changes required (optional: add forced indicator to history)

### Architectural Concerns

**Audit Trail**: Forced status changes should be tracked for accountability.

**Cascading Updates**: Feature force update must handle batch task updates atomically.

### Recommended Design Approach

**1. CLI Flag Addition**:
```go
// Add to all status commands
taskCompleteCmd.Flags().Bool("force", false, "Force status change without validation")
taskStartCmd.Flags().Bool("force", false, "Force status change without validation")

featureCompleteCmd.Flags().Bool("force", false, "Force status change for feature and all tasks")
```

**2. Repository Method Updates** (`internal/repository/task_repository.go`):
```go
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, newStatus string, force bool) error {
    if !force {
        // Validate status transition
        current, err := r.GetByID(ctx, taskID)
        if err != nil {
            return err
        }

        if err := validateStatusTransition(current.Status, newStatus); err != nil {
            return err
        }
    }

    // Update status
    tx, err := r.db.BeginTxContext(ctx, nil)
    defer tx.Rollback()

    _, err = tx.ExecContext(ctx, `UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, newStatus, taskID)

    // Record in history with forced flag
    _, err = tx.ExecContext(ctx, `INSERT INTO task_history (task_id, status, forced, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, taskID, newStatus, force)

    return tx.Commit()
}
```

**3. Feature Cascade Logic**:
```go
func (r *FeatureRepository) UpdateStatusWithTasks(ctx context.Context, featureID int64, newStatus string, force bool) error {
    tx, err := r.db.BeginTxContext(ctx, nil)
    defer tx.Rollback()

    // Update feature status
    _, err = tx.ExecContext(ctx, `UPDATE features SET status = ? WHERE id = ?`, newStatus, featureID)

    // Update all child tasks
    _, err = tx.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE feature_id = ?`, newStatus, featureID)

    // Record history for each task (optional: batch insert)
    rows, err := tx.QueryContext(ctx, `SELECT id FROM tasks WHERE feature_id = ?`, featureID)
    // ... insert history records with forced=true

    return tx.Commit()
}
```

**4. History Schema** (if not exists):
```sql
ALTER TABLE task_history ADD COLUMN forced BOOLEAN DEFAULT FALSE;
```

### Key Implementation Considerations

1. **Transaction Atomicity**: Feature + tasks update must be atomic (all or nothing)
2. **History Tracking**: Record forced=true for audit trail
3. **Error Handling**: What if some tasks fail to update? (Transaction handles this)
4. **User Feedback**: Report count of tasks updated when forcing feature status
5. **Permissions**: Document that force is administrative operation

### Dependencies and Integration Points

- `internal/repository/task_repository.go` - Status update logic
- `internal/repository/feature_repository.go` - Cascade update logic
- `internal/cli/commands/task.go` - Add force flag to status commands
- `internal/cli/commands/feature.go` - Add force flag and create status commands if missing
- Database: May need to add task_history.forced column

### Complexity Estimate: **MODERATE**

Estimated effort: 8-10 hours including cascade logic and tests

### Risk Assessment: **MEDIUM**

- Cascading updates introduce transaction complexity
- Need careful testing of rollback scenarios
- Audit trail is critical for accountability

**Recommendation**: Implement after F04 (execution order) to avoid migration conflicts.

---

## Feature E07-F07: Epic Index Discovery Integration

### Feature Summary
Integrate the existing `internal/discovery` package with `shark sync` command to enable epic-index.md parsing, conflict resolution, and database synchronization.

### Feasibility Rating: **HIGH**

**Justification**:
- Discovery package already implemented and tested
- Integration is primarily "wiring" existing components
- Clear interfaces and patterns established

### Architectural Concerns

**Complexity**: Most complex feature in E07 due to multiple integration points.

**Performance**: Parsing large epic-index.md files and scanning folders may be slow.

**Conflict Resolution**: Strategy selection and conflict reporting requires careful UX design.

### Recommended Design Approach

**1. Sync Command Extensions** (`internal/cli/commands/sync.go`):
```go
var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Synchronize database with documentation structure",
    RunE:  runSync,
}

func init() {
    syncCmd.Flags().String("index", "", "Path to epic-index.md file for discovery")
    syncCmd.Flags().String("discovery-strategy", "merge", "Conflict resolution strategy: index-precedence, folder-precedence, merge")
    syncCmd.Flags().String("validation-level", "balanced", "Validation level: strict, balanced, permissive")
    syncCmd.Flags().Bool("create-missing", false, "Auto-create missing epics/features")
}
```

**2. Sync Engine Integration** (`internal/sync/engine.go`):
```go
func (e *Engine) Sync(ctx context.Context, opts SyncOptions) (*SyncReport, error) {
    report := &SyncReport{}

    // Existing: Scan task files
    tasks, err := e.scanTasks(ctx, opts.RootPath)
    report.TasksScanned = len(tasks)

    // NEW: Discovery workflow if --index provided
    if opts.IndexPath != "" {
        discoveryReport, err := e.runDiscovery(ctx, opts)
        if err != nil {
            return nil, fmt.Errorf("discovery failed: %w", err)
        }
        report.Discovery = discoveryReport
    }

    return report, nil
}

func (e *Engine) runDiscovery(ctx context.Context, opts SyncOptions) (*DiscoveryReport, error) {
    // 1. Parse index
    indexParser := discovery.NewIndexParser()
    indexResults, err := indexParser.Parse(opts.IndexPath)

    // 2. Scan folders
    folderScanner := discovery.NewFolderScanner(e.patternRegistry)
    folderResults, err := folderScanner.Scan(opts.RootPath)

    // 3. Detect conflicts
    conflictDetector := discovery.NewConflictDetector()
    conflicts := conflictDetector.Detect(indexResults, folderResults)

    // 4. Resolve conflicts
    resolver := discovery.NewConflictResolver(opts.DiscoveryStrategy)
    resolved, err := resolver.Resolve(conflicts)

    // 5. Write to database (if not dry-run)
    if !opts.DryRun {
        err = e.importDiscoveredEntities(ctx, resolved)
    }

    return &DiscoveryReport{
        EpicsFromIndex:   len(indexResults.Epics),
        EpicsFromFolders: len(folderResults.Epics),
        FeaturesFromIndex: len(indexResults.Features),
        FeaturesFromFolders: len(folderResults.Features),
        Conflicts: conflicts,
        Resolved: resolved,
    }, nil
}
```

**3. Report Extension** (`internal/sync/types.go`):
```go
type SyncReport struct {
    // Existing fields
    TasksScanned int
    TasksImported int
    Errors []error

    // NEW: Discovery fields
    Discovery *DiscoveryReport `json:"discovery,omitempty"`
}

type DiscoveryReport struct {
    EpicsFromIndex      int
    EpicsFromFolders    int
    FeaturesFromIndex   int
    FeaturesFromFolders int
    Conflicts           []Conflict
    Resolved            *ResolvedEntities
}
```

**4. Database Import Logic**:
```go
func (e *Engine) importDiscoveredEntities(ctx context.Context, resolved *ResolvedEntities) error {
    tx, err := e.db.BeginTxContext(ctx, nil)
    defer tx.Rollback()

    // Import epics
    for _, epic := range resolved.Epics {
        _, err = e.epicRepo.CreateOrUpdate(ctx, epic)
    }

    // Import features
    for _, feature := range resolved.Features {
        _, err = e.featureRepo.CreateOrUpdate(ctx, feature)
    }

    return tx.Commit()
}
```

### Key Implementation Considerations

1. **Package Integration**: Import and wire `internal/discovery` components
2. **Configuration**: Read .sharkconfig.json patterns via existing patterns package
3. **Error Handling**: Strategy failures (e.g., index-precedence with missing folders) must fail gracefully
4. **Performance**: Run IndexParser and FolderScanner in parallel (goroutines)
5. **Reporting**: Clear conflict output with actionable suggestions
6. **Dry-Run**: Ensure --dry-run works with discovery
7. **Backward Compatibility**: Sync works identically when --index not provided

### Dependencies and Integration Points

- **Existing**: `internal/discovery/*` - All components already implemented
- **Existing**: `internal/patterns` - Pattern registry for validation
- **Modify**: `internal/sync/engine.go` - Add discovery workflow
- **Modify**: `internal/sync/types.go` - Extend SyncReport
- **Modify**: `internal/cli/commands/sync.go` - Add flags
- **Modify**: `internal/reporting/` - Add discovery output formatting

**Dependency on E07-F03**: F07 makes slug resolution easier for F03 (task file paths).

### Complexity Estimate: **COMPLEX**

Estimated effort: 16-20 hours including integration, testing, and documentation

### Risk Assessment: **MEDIUM**

- Highest complexity in E07 but discovery package already tested
- Integration risks (wiring errors, missing error handling)
- Performance testing required for large projects
- UX for conflict reporting needs iteration

**Recommendation**: Implement early (after simple features) to unblock F03.

---

## Cross-Feature Analysis

### Dependency Graph

```
F02 (Title Positional)
  ↓ (no dependencies)

F01 (Remove Agent Requirement)
  ↓ (no dependencies)

F07 (Discovery Integration) ← Priority: Unblocks F03
  ↓

F03 (Move Task Files) ← Needs slug resolution from F07
  ↓ (no dependencies)

F06 (Force Status) ⇅ F04 (Execution Order) ← Can be parallel

F05 (Related Documents) ← Most complex, implement last
```

### Recommended Implementation Order

**Phase 1: Simple CLI Improvements** (Week 1)
1. **E07-F02**: Make task title positional argument (2-3 hours)
2. **E07-F01**: Remove agent requirement (4-6 hours)

**Phase 2: Discovery Foundation** (Week 2)
3. **E07-F07**: Epic Index Discovery Integration (16-20 hours)

**Phase 3: File Organization** (Week 2-3)
4. **E07-F03**: Move task files to feature folder (8-10 hours)

**Phase 4: Database Enhancements** (Week 3-4)
5. **E07-F06**: Allow force status on task (8-10 hours)
6. **E07-F04**: Implementation order (6-8 hours)

**Phase 5: Advanced Features** (Week 4-5)
7. **E07-F05**: Add related documents (12-16 hours)

**Total Estimated Effort**: 56-73 hours (7-9 developer days)

### Schema Migration Coordination

**Database Changes Required**:
- F04: Add execution_order to features and tasks tables
- F05: Add documents, epic_documents, feature_documents, task_documents tables
- F06: Add forced column to task_history table (optional)

**Recommendation**: Create unified migration script for all schema changes to avoid multiple migrations.

```sql
-- Migration: E07-enhancements.sql
-- Add execution_order (F04)
ALTER TABLE features ADD COLUMN execution_order INTEGER NULL;
ALTER TABLE tasks ADD COLUMN execution_order INTEGER NULL;

-- Add documents tables (F05)
CREATE TABLE documents (...);
CREATE TABLE epic_documents (...);
CREATE TABLE feature_documents (...);
CREATE TABLE task_documents (...);

-- Add forced tracking (F06)
ALTER TABLE task_history ADD COLUMN forced BOOLEAN DEFAULT FALSE;
```

### Integration Testing Strategy

1. **Unit Tests**: Each feature has isolated unit tests
2. **Integration Tests**: Test feature combinations
   - F01 + F02: Create task with positional title and no agent
   - F07 + F03: Sync with index, then create task in discovered feature
   - F04 + F06: Force status change respects execution order in next suggestion
3. **End-to-End**: Full workflow test with all features enabled

---

## Risk Matrix

| Feature | Complexity | Risk Level | Mitigation |
|---------|-----------|-----------|------------|
| E07-F01 | Simple | LOW | Thorough testing of template fallback |
| E07-F02 | Simple | LOW | Clear migration guide for users |
| E07-F03 | Moderate | MEDIUM | Implement after F07 for slug resolution |
| E07-F04 | Moderate | LOW | Standard NULL handling patterns |
| E07-F05 | Complex | MEDIUM | Careful foreign key and transaction testing |
| E07-F06 | Moderate | MEDIUM | Robust transaction rollback testing |
| E07-F07 | Complex | MEDIUM | Performance testing on large projects |

---

## Architecture Decision Records

### ADR-001: Optional Agent Field Implementation

**Decision**: Make agent field optional by removing MarkFlagRequired, not by adding "TBD" value.

**Rationale**: Optional fields are cleaner than magic values. Database already supports NULL.

**Consequences**: Simpler validation logic, more flexible workflow.

---

### ADR-002: Title as Positional Argument

**Decision**: Accept breaking change to make title positional for consistency.

**Rationale**: Consistency across epic/feature/task commands improves UX. One-time migration cost is acceptable.

**Consequences**: Breaking change requires version bump and migration guide.

---

### ADR-003: Task File Location Strategy

**Decision**: Place tasks under feature folder in tasks/ subdirectory.

**Rationale**: Hierarchical organization improves navigation. Existing tasks remain in place (no migration).

**Consequences**: Requires epic/feature slug resolution. Depends on discovery integration.

---

### ADR-004: Execution Order as Nullable Integer

**Decision**: Use nullable integer for execution_order, not JSON blob.

**Rationale**: Simple integer is sufficient for initial use case. Can add complexity later if needed.

**Consequences**: Easy sorting, standard database patterns. May need to evolve if complex ordering needed.

---

### ADR-005: Related Documents Many-to-Many Design

**Decision**: Use dedicated documents table with junction tables for relationships.

**Rationale**: Proper normalization allows document reuse across entities. Better than JSON blobs.

**Consequences**: More tables but cleaner queries and referential integrity.

---

### ADR-006: Force Flag for Status Changes

**Decision**: Add --force flag instead of separate admin commands.

**Rationale**: Simpler UX, clear intent. Flag-based override is standard CLI pattern.

**Consequences**: Need audit trail (forced flag in history). Document as admin feature.

---

### ADR-007: Discovery Integration Opt-In

**Decision**: Discovery via --index flag (opt-in), not automatic detection.

**Rationale**: Explicit behavior is predictable. Some projects may have epic-index.md but not want it to drive database.

**Consequences**: Users must explicitly enable discovery. Clear opt-in UX.

---

## Conclusion

All 7 features in Epic E07 are architecturally sound and feasible. The recommended implementation order prioritizes:

1. **Quick wins** (F02, F01) to deliver immediate value
2. **Foundation** (F07) to enable dependent features
3. **Dependencies** (F03 after F07)
4. **Parallel tracks** (F04, F06 independently)
5. **Complex features last** (F05)

**Estimated Timeline**: 7-9 developer days for full epic completion.

**Key Success Factors**:
- Coordinated database migrations
- Comprehensive integration testing
- Clear user migration guides for breaking changes (F02)
- Performance validation for discovery (F07)

**Next Steps**:
1. Approve architecture recommendations
2. Create implementation tasks for each feature
3. Execute in recommended order
4. Integration testing between features

---

**Review Completed**: 2025-12-18
**Status**: Ready for task generation and implementation
