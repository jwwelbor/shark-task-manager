# Path/Filename Refactoring Plan

## Overview

This plan breaks down the path/filename architecture refactoring into manageable phases. Each phase can be tested and verified independently.

## Phase 0: Preparation & Design Approval

**Goal:** Validate architecture design and get stakeholder approval

### Tasks
- [x] Document architecture design (PATH_FILENAME_ARCHITECTURE.md)
- [x] Analyze current implementation (CURRENT_IMPLEMENTATION_ANALYSIS.md)
- [ ] Review design with stakeholders
- [ ] Get approval to proceed
- [ ] Create feature branch: `feature/path-filename-refactoring`

**Exit Criteria:** Design approved, branch created

---

## Phase 1: Database Schema Migration

**Goal:** Add new columns without breaking existing functionality

### Tasks

#### 1.1: Add custom_path column to tasks table
```sql
ALTER TABLE tasks ADD COLUMN custom_path TEXT;
CREATE INDEX IF NOT EXISTS idx_tasks_custom_path ON tasks(custom_path);
```

#### 1.2: Add filename columns to all tables
```sql
ALTER TABLE epics ADD COLUMN filename TEXT DEFAULT 'epic.md';
ALTER TABLE features ADD COLUMN filename TEXT DEFAULT 'feature.md';
ALTER TABLE tasks ADD COLUMN filename TEXT;  -- Will compute default: {task-key}.md
```

#### 1.3: Rename custom_folder_path to custom_path
```sql
-- For clarity and consistency
ALTER TABLE epics RENAME COLUMN custom_folder_path TO custom_path;
ALTER TABLE features RENAME COLUMN custom_folder_path TO custom_path;
```

#### 1.4: Update migration function in db.go
- Add migration logic to `internal/db/db.go`
- Make migrations idempotent (check if columns exist first)
- Add migration tests

#### 1.5: Backfill filename column
- Parse existing file_path values
- Extract filename component
- Update filename column for all existing records

**Test Plan:**
```bash
# Run migration
./bin/shark epic list  # Triggers auto-migration

# Verify schema
sqlite3 shark-tasks.db ".schema epics" | grep -E "custom_path|filename"
sqlite3 shark-tasks.db ".schema features" | grep -E "custom_path|filename"
sqlite3 shark-tasks.db ".schema tasks" | grep -E "custom_path|filename"

# Verify data backfill
sqlite3 shark-tasks.db "SELECT key, file_path, filename FROM epics LIMIT 5;"
```

**Exit Criteria:**
- Schema updated with new columns
- Existing data backfilled
- All tests pass
- No breaking changes to existing commands

---

## Phase 2: Path Resolution Logic

**Goal:** Implement centralized path resolution with inheritance

### Tasks

#### 2.1: Create path resolver utility
**File:** `internal/utils/path_resolver.go`

```go
package utils

// PathResolver handles path computation with inheritance
type PathResolver struct {
    docsRoot string
}

// ResolveEpicPath computes full path for an epic
func (pr *PathResolver) ResolveEpicPath(epic *models.Epic) string

// ResolveFeaturePath computes full path for a feature (inherits from epic)
func (pr *PathResolver) ResolveFeaturePath(feature *models.Feature, epic *models.Epic) string

// ResolveTaskPath computes full path for a task (inherits from feature and epic)
func (pr *PathResolver) ResolveTaskPath(task *models.Task, feature *models.Feature, epic *models.Epic) string

// GetPathSegment returns the path segment for this entity
func GetEpicPathSegment(epic *models.Epic) string
func GetFeaturePathSegment(feature *models.Feature) string
func GetTaskPathSegment(task *models.Task) string

// GetFilename returns the filename for this entity
func GetEpicFilename(epic *models.Epic) string
func GetFeatureFilename(feature *models.Feature) string
func GetTaskFilename(task *models.Task) string
```

#### 2.2: Add configuration support
**File:** `internal/config/config.go`

Add to .sharkconfig.json:
```json
{
  "docs_root": "docs",
  "default_epic_path": "plan",
  "default_task_path": "tasks",
  "default_epic_filename": "epic.md",
  "default_feature_filename": "feature.md",
  "default_task_filename_pattern": "{task-key}.md"
}
```

#### 2.3: Write comprehensive tests
**File:** `internal/utils/path_resolver_test.go`

Test scenarios:
- Default paths (no custom values)
- Custom epic path
- Custom feature path (inherits epic)
- Custom task path (inherits feature + epic)
- Custom filenames
- Edge cases (empty paths, special characters)

**Test Plan:**
```bash
go test -v ./internal/utils -run TestPathResolver
```

**Exit Criteria:**
- Path resolver implemented
- All tests pass (>90% coverage)
- Configuration support added

---

## Phase 3: Repository Layer Updates

**Goal:** Update repositories to support new columns and path resolution

### Tasks

#### 3.1: Update Epic Repository
**File:** `internal/repository/epic_repository.go`

- Add custom_path and filename to Create()
- Add custom_path and filename to Update()
- Add GetByFilePath() if not exists
- Update tests

#### 3.2: Update Feature Repository
**File:** `internal/repository/feature_repository.go`

- Add custom_path and filename to Create()
- Add custom_path and filename to Update()
- Add methods to get with epic data (for path resolution)
- Update tests

#### 3.3: Update Task Repository
**File:** `internal/repository/task_repository.go`

- Add custom_path and filename to Create()
- Add custom_path and filename to Update()
- Add methods to get with feature+epic data (for path resolution)
- Update tests

#### 3.4: Add path computation methods
Add to all repositories:
```go
// ComputeFullPath resolves the full file path including inheritance
func (r *EpicRepository) ComputeFullPath(ctx context.Context, epic *models.Epic) (string, error)
func (r *FeatureRepository) ComputeFullPath(ctx context.Context, feature *models.Feature) (string, error)
func (r *TaskRepository) ComputeFullPath(ctx context.Context, task *models.Task) (string, error)
```

**Test Plan:**
```bash
go test -v ./internal/repository -run TestEpicRepository
go test -v ./internal/repository -run TestFeatureRepository
go test -v ./internal/repository -run TestTaskRepository
```

**Exit Criteria:**
- All repository methods support new columns
- Path computation methods implemented
- All tests pass (with mocked repositories for unit tests)

---

## Phase 4: Update Epic Commands

**Goal:** Fix epic create/update to use new path resolution

### Tasks

#### 4.1: Update epic create command
**File:** `internal/cli/commands/epic.go`

Changes:
- Parse --path as path segment (not full path)
- Parse --filename as just filename
- Use PathResolver to compute full path
- Check if file exists (reference vs create)
- Create placeholder if file doesn't exist
- Store custom_path and filename separately

#### 4.2: Update epic update command
**File:** `internal/cli/commands/epic.go`

Changes:
- Support --path to update custom_path
- Support --filename to update filename
- Optionally move file if path changes (with confirmation)
- Cascade update to child features

#### 4.3: Update epic get command
**File:** `internal/cli/commands/epic.go`

Changes:
- Display custom_path and filename separately
- Show computed full path

#### 4.4: Write integration tests
**File:** `internal/cli/commands/epic_test.go`

Test scenarios:
- Create with default path
- Create with custom path
- Create with custom filename
- Create referencing existing file
- Update path (with file move)
- Update filename (with file rename)

**Test Plan:**
```bash
# Manual testing
./bin/shark epic create "Test Epic" --json
./bin/shark epic create "Test Epic 2" --path="roadmap/2025" --json
./bin/shark epic create "Test Epic 3" --filename="custom-epic.md" --json
./bin/shark epic update E01 --path="archived" --json

# Unit tests
go test -v ./internal/cli/commands -run TestEpicCreate
go test -v ./internal/cli/commands -run TestEpicUpdate
```

**Exit Criteria:**
- Epic commands fully support new architecture
- All tests pass
- No regressions in existing functionality

---

## Phase 5: Update Feature Commands

**Goal:** Fix feature create/update to use new path resolution with inheritance

### Tasks

#### 5.1: Update feature create command
**File:** `internal/cli/commands/feature.go`

Changes:
- Parse --path as path segment (relative to epic)
- Parse --filename as just filename
- Use PathResolver with epic inheritance
- Check if file exists (reference vs create)
- Create placeholder if file doesn't exist
- Store custom_path and filename separately

#### 5.2: Update feature update command
**File:** `internal/cli/commands/feature.go`

Changes:
- Support --path to update custom_path
- Support --filename to update filename
- Optionally move file if path changes (with confirmation)
- Cascade update to child tasks

#### 5.3: Update feature get command
**File:** `internal/cli/commands/feature.go`

Changes:
- Display custom_path and filename separately
- Show computed full path
- Show inherited path from epic

#### 5.4: Write integration tests
**File:** `internal/cli/commands/feature_test.go`

Test scenarios:
- Create with default path (inherits from epic)
- Create with custom path
- Create with custom filename
- Create referencing existing file
- Update path (with file move)
- Update filename (with file rename)
- Verify inheritance from epic

**Test Plan:**
```bash
# Manual testing
./bin/shark feature create --epic=E01 "Test Feature" --json
./bin/shark feature create --epic=E01 "Test Feature 2" --path="auth" --json
./bin/shark feature create --epic=E01 "Test Feature 3" --filename="custom.md" --json
./bin/shark feature update E01-F01 --path="security" --json

# Unit tests
go test -v ./internal/cli/commands -run TestFeatureCreate
go test -v ./internal/cli/commands -run TestFeatureUpdate
```

**Exit Criteria:**
- Feature commands fully support new architecture
- Inheritance from epic works correctly
- All tests pass
- Original bug (feature update --path) is fixed

---

## Phase 6: Update Task Commands

**Goal:** Add path support to tasks and fix task create/update

### Tasks

#### 6.1: Update task create command
**File:** `internal/cli/commands/task.go`

Changes:
- Add --path flag to set custom_path
- Parse --filename as just filename (default: {task-key}.md)
- Use PathResolver with feature+epic inheritance
- Check if file exists (reference vs create)
- Create placeholder if file doesn't exist
- Store custom_path and filename separately

#### 6.2: Update task update command
**File:** `internal/cli/commands/task.go`

Changes:
- Add --path flag to update custom_path
- Support --filename to update filename
- Optionally move file if path changes (with confirmation)

#### 6.3: Update task get command
**File:** `internal/cli/commands/task.go`

Changes:
- Display custom_path and filename separately
- Show computed full path
- Show inherited path from feature and epic

#### 6.4: Write integration tests
**File:** `internal/cli/commands/task_test.go`

Test scenarios:
- Create with default path (inherits from feature+epic)
- Create with custom path
- Create with custom filename
- Create referencing existing file
- Update path (with file move)
- Update filename (with file rename)
- Verify inheritance from feature and epic

**Test Plan:**
```bash
# Manual testing
./bin/shark task create --epic=E01 --feature=F01 "Test Task" --agent=backend --json
./bin/shark task create --epic=E01 --feature=F01 "Test Task 2" --path="backend" --agent=backend --json
./bin/shark task create --epic=E01 --feature=F01 "Test Task 3" --filename="custom-task.md" --agent=backend --json
./bin/shark task update T-E01-F01-001 --path="frontend" --json

# Unit tests
go test -v ./internal/cli/commands -run TestTaskCreate
go test -v ./internal/cli/commands -run TestTaskUpdate
```

**Exit Criteria:**
- Task commands fully support new architecture
- Inheritance from feature and epic works correctly
- All tests pass

---

## Phase 7: Cascading Updates & File Operations

**Goal:** Implement file moving and cascading path updates

### Tasks

#### 7.1: Implement file move utility
**File:** `internal/utils/file_ops.go`

```go
// MoveFile moves a file from old path to new path
func MoveFile(oldPath, newPath string) error

// BackupFile creates a backup before moving
func BackupFile(path string) error

// ConfirmFileMove asks user for confirmation before moving
func ConfirmFileMove(oldPath, newPath string) (bool, error)
```

#### 7.2: Implement cascading update logic
**File:** `internal/repository/cascading_updates.go`

```go
// CascadeEpicPathUpdate updates all child features when epic path changes
func CascadeEpicPathUpdate(ctx context.Context, epicID int64, db *DB) error

// CascadeFeaturePathUpdate updates all child tasks when feature path changes
func CascadeFeaturePathUpdate(ctx context.Context, featureID int64, db *DB) error
```

#### 7.3: Update epic/feature update commands
- Add confirmation prompts before moving files
- Call cascading update functions
- Show summary of what will change

#### 7.4: Write integration tests
Test scenarios:
- Update epic path → all features recalculate paths
- Update feature path → all tasks recalculate paths
- File move with backup creation
- Rollback on error

**Test Plan:**
```bash
# Create test hierarchy
./bin/shark epic create "Test Epic"
./bin/shark feature create --epic=E01 "Feature 1"
./bin/shark feature create --epic=E01 "Feature 2"
./bin/shark task create --epic=E01 --feature=F01 "Task 1" --agent=backend
./bin/shark task create --epic=E01 --feature=F01 "Task 2" --agent=backend

# Update epic path (should cascade)
./bin/shark epic update E01 --path="roadmap/q1"

# Verify all paths updated
./bin/shark feature get E01-F01 --json | jq '.computed_path'
./bin/shark task get T-E01-F01-001 --json | jq '.computed_path'
```

**Exit Criteria:**
- File moves work correctly
- Cascading updates propagate to children
- Backups created before moves
- All tests pass

---

## Phase 8: Documentation & Migration

**Goal:** Update all documentation and provide migration guide

### Tasks

#### 8.1: Update CLI documentation
**File:** `docs/CLI_REFERENCE.md`

- Document --path and --filename flags for all commands
- Add examples of custom paths
- Add examples of inheritance
- Document cascading updates

#### 8.2: Update CLAUDE.md
**File:** `CLAUDE.md`

- Update architecture section
- Update database schema documentation
- Update path resolution explanation
- Add migration notes

#### 8.3: Create migration guide
**File:** `docs/MIGRATION_PATH_FILENAME_REFACTOR.md`

- Explain changes
- Provide migration steps for existing projects
- Document backward compatibility
- Show before/after examples

#### 8.4: Update README
**File:** `README.md`

- Add section on path customization
- Link to detailed documentation
- Add examples

**Exit Criteria:**
- All documentation updated
- Migration guide complete
- Examples tested and working

---

## Phase 9: Testing & Validation

**Goal:** Comprehensive testing of all scenarios

### Tasks

#### 9.1: End-to-end test suite
Create comprehensive test script:
```bash
#!/bin/bash
# test-path-refactoring.sh

# Test default behavior
# Test custom paths
# Test custom filenames
# Test inheritance
# Test cascading updates
# Test file moves
# Test error handling
# Test backward compatibility
```

#### 9.2: Performance testing
- Test with large datasets (1000+ tasks)
- Verify path resolution performance
- Check database query performance

#### 9.3: Backward compatibility testing
- Test with existing database
- Verify old commands still work
- Verify migration doesn't break anything

#### 9.4: User acceptance testing
- Create demo scenarios
- Get feedback from stakeholders
- Fix any issues discovered

**Test Plan:**
```bash
# Run full test suite
make test

# Run end-to-end tests
./test-path-refactoring.sh

# Test migration
cp shark-tasks.db shark-tasks.backup
./bin/shark epic list  # Triggers migration
# Verify no data loss
```

**Exit Criteria:**
- All tests pass
- Performance acceptable
- No regressions
- Stakeholders approve

---

## Phase 10: Cleanup & Deprecation

**Goal:** Clean up old code and deprecate obsolete fields

### Tasks

#### 10.1: Mark file_path as deprecated
- Add deprecation notice in code comments
- Keep for backward compatibility in this version
- Plan removal for next major version

#### 10.2: Update sync logic
**File:** `internal/sync/sync.go`

- Update to use custom_path + filename
- Maintain compatibility with old file_path

#### 10.3: Code cleanup
- Remove unused code
- Consolidate duplicated path logic
- Improve error messages

#### 10.4: Final review
- Code review
- Security review
- Performance review

**Exit Criteria:**
- Code clean and maintainable
- All reviews passed
- Ready for merge

---

## Rollout Strategy

### Step 1: Merge to main
- Squash commits or use feature branch merge
- Tag release: `v1.x.0` (minor version bump)

### Step 2: Release notes
- Highlight new path customization features
- Document breaking changes (if any)
- Provide migration guide link

### Step 3: Monitor
- Watch for bug reports
- Gather user feedback
- Prepare hotfixes if needed

---

## Risk Mitigation

### Risk: Data loss during migration
**Mitigation:**
- Auto-backup before migration
- Migration is idempotent (can run multiple times safely)
- Test on copy of production data first

### Risk: Breaking existing workflows
**Mitigation:**
- Maintain backward compatibility with file_path
- Gradual deprecation (keep old behavior for one major version)
- Clear migration guide and examples

### Risk: Performance degradation
**Mitigation:**
- Cache computed paths in memory or database
- Add indexes on new columns
- Performance test with large datasets

### Risk: Complex inheritance logic
**Mitigation:**
- Comprehensive unit tests
- Clear documentation
- Logging for debugging path resolution

---

## Success Criteria

✅ All tests pass (unit, integration, E2E)
✅ Original bug fixed: `feature update --path` works correctly
✅ Path inheritance works: features inherit from epic, tasks from feature
✅ File operations work: create, move, rename files correctly
✅ Backward compatible: existing projects migrate seamlessly
✅ Well documented: clear examples and migration guide
✅ Performance acceptable: no noticeable slowdown
✅ Stakeholder approval: design and implementation approved

---

## Timeline Estimate

| Phase | Effort | Duration |
|-------|--------|----------|
| Phase 0: Preparation | S | 0.5 day |
| Phase 1: Schema Migration | M | 1 day |
| Phase 2: Path Resolution | L | 1.5 days |
| Phase 3: Repository Updates | M | 1 day |
| Phase 4: Epic Commands | M | 1 day |
| Phase 5: Feature Commands | M | 1 day |
| Phase 6: Task Commands | M | 1 day |
| Phase 7: Cascading Updates | L | 1.5 days |
| Phase 8: Documentation | M | 1 day |
| Phase 9: Testing | L | 1.5 days |
| Phase 10: Cleanup | S | 0.5 day |
| **Total** | | **~11.5 days** |

Note: This is a significant refactoring that touches many parts of the system. Consider breaking into multiple PRs by phase for easier review.

---

## Open Questions for Stakeholder

1. **File moving behavior:** Should `update --path` automatically move files, ask for confirmation, or just update the database reference?

2. **Cascading updates:** When an epic path changes, should all child features and tasks automatically recalculate their paths? Or should it be opt-in?

3. **Backward compatibility:** How long should we maintain the old `file_path` column before deprecating it completely?

4. **Default paths:** Are the proposed default paths acceptable?
   - Epic: `docs/plan/{epic-key}/epic.md`
   - Feature: `{epic-path}/{feature-key}/feature.md`
   - Task: `{feature-path}/tasks/{task-key}.md`

5. **File existence behavior:** When creating with a path where file already exists, should we:
   - A) Reference existing file (no overwrite)
   - B) Error and require --force flag
   - C) Create with .1, .2 suffix

6. **docs_root configuration:** Should this be configurable per-project, or global default?

Please review and provide feedback on the design and plan before proceeding with implementation.
