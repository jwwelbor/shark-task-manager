---
feature_key: E07-F09-custom-folder-base-paths
epic_key: E07
title: Custom Folder Base Paths
description: Custom base folder paths that cascade to child items
---

# Feature PRD: Custom Folder Base Paths

**Epic**: E07-Enhancements
**Feature**: E07-F09-Custom Folder Base Paths
**Version**: 1.0
**Status**: Draft
**Last Updated**: 2025-12-19

---

## 1. Overview

### Problem Statement

Currently, Shark Task Manager enforces a rigid default hierarchy for organizing documentation:

```
docs/plan/<epic-key>/epic.md
docs/plan/<epic-key>/<feature-key>/feature.md
docs/plan/<epic-key>/<feature-key>/tasks/<task-key>.md
```

While E07-F08 enables custom filenames for individual epics and features, it requires specifying the exact path for each item. This becomes repetitive when organizing an entire epic or feature hierarchy under a custom base folder.

For example, to organize an epic with custom naming in a different location, users must:
1. Use `--filename` for the epic
2. Use `--filename` for every feature within that epic
3. Manually construct each path to maintain the hierarchy

This is tedious and error-prone for large epics with many features.

### Solution

Introduce a `--path` parameter for epics and features that sets a **custom base folder** for all child items. This folder path cascades down the hierarchy:

- **Epic with custom path**: All features and tasks use this custom base instead of `docs/plan/<epic-key>`
- **Feature with custom path**: All tasks use this custom base instead of the default feature folder

#### Example Use Cases

**Default behavior** (current):
```
docs/plan/E01-Default-epic-naming/epic.md
docs/plan/E01-Default-epic-naming/E01-F01-default-feature-name/feature.md
docs/plan/E01-Default-epic-naming/E01-F01-default-feature-name/tasks/T-E01-F01-001-default-task.md
```

**With custom epic path**:
```bash
shark epic create --title="Special Epic" --path="docs/plan/special-epic"
```
Result:
```
docs/plan/special-epic/E01/epic.md
docs/plan/special-epic/E01/E01-F01-default-feature-name/feature.md
docs/plan/special-epic/E01/E01-F01-default-feature-name/tasks/T-E01-F01-001-default-task.md
```

**With custom feature path**:
```bash
shark feature create --epic=E01 --title="Custom Feature" --path="docs/plan/E01-default-epic/phase-1"
```
Result:
```
docs/plan/E01-default-epic/epic.md
docs/plan/E01-default-epic/phase-1/E01-F01-custom-feature-name/feature.md
docs/plan/E01-default-epic/phase-1/E01-F01-custom-feature-name/tasks/T-E01-F01-001-default-task.md
```

### Why Now

- **Builds on E07-F08**: Custom filename infrastructure is proven and tested
- **Reduces repetition**: Single `--path` parameter replaces dozens of `--filename` calls
- **Organizational flexibility**: Users can organize work by phases, modules, or custom structures
- **User demand**: Teams need to organize epics by quarters, product areas, or strategic themes

---

## 2. User Stories

### User Story 1: Epic with Custom Base Path

**ID**: US-E07-F09-001

As a **product manager**, I want to create an epic with a custom base folder path, so that all features and tasks within that epic are automatically organized under my custom hierarchy without specifying paths for each child item.

#### Acceptance Criteria

- **AC-1.1**: `shark epic create --title="Q1 2025 Roadmap" --path="docs/roadmap/2025-q1"` creates the epic and sets the custom base path
- **AC-1.2**: The epic file is created at `docs/roadmap/2025-q1/<epic-key>/epic.md`
- **AC-1.3**: All features created under this epic inherit the base path: `docs/roadmap/2025-q1/<epic-key>/<feature-key>/feature.md`
- **AC-1.4**: All tasks created under those features inherit the path: `docs/roadmap/2025-q1/<epic-key>/<feature-key>/tasks/<task-key>.md`
- **AC-1.5**: The custom base path is stored in the database for the epic
- **AC-1.6**: Without `--path`, the epic uses the default behavior: `docs/plan/<epic-key>/epic.md`

---

### User Story 2: Feature with Custom Base Path

**ID**: US-E07-F09-002

As a **tech lead**, I want to organize features into phases or modules using custom folder paths, so that I can group related features together without changing the epic structure.

#### Acceptance Criteria

- **AC-2.1**: `shark feature create --epic=E04 --title="Authentication Core" --path="docs/plan/E04-auth-system/core"` creates the feature with a custom base path
- **AC-2.2**: The feature file is created at `docs/plan/E04-auth-system/core/<feature-key>/feature.md`
- **AC-2.3**: All tasks created under this feature inherit the base path: `docs/plan/E04-auth-system/core/<feature-key>/tasks/<task-key>.md`
- **AC-2.4**: The custom base path is stored in the database for the feature
- **AC-2.5**: Other features in the same epic without `--path` use the default epic base path
- **AC-2.6**: Tasks created under this feature automatically use the custom base path without additional parameters

---

### User Story 3: Path Inheritance and Override

**ID**: US-E07-F09-003

As a **developer**, I want clear rules for how custom paths cascade from parent to child, so that I can predict where files will be created and optionally override the inherited path when needed.

#### Acceptance Criteria

- **AC-3.1**: When creating a feature under an epic with a custom path, the feature inherits the epic's base path by default
- **AC-3.2**: When creating a task under a feature with a custom path, the task inherits the feature's base path by default
- **AC-3.3**: A feature can override the epic's custom path by specifying its own `--path`
- **AC-3.4**: A task can override inherited paths by using `--filename` (from E07-F05)
- **AC-3.5**: The precedence order is clear: task `--filename` > feature `--path` > epic `--path` > default structure
- **AC-3.6**: Documentation clearly explains the inheritance rules with examples

---

### User Story 4: Validation and Safety

**ID**: US-E07-F09-004

As a **system administrator**, I want custom base paths to be validated for security and correctness, so that users cannot create files outside the project boundaries or use invalid paths.

#### Acceptance Criteria

- **AC-4.1**: Paths must be relative to the project root (no absolute paths)
- **AC-4.2**: Paths cannot contain `..` (no path traversal)
- **AC-4.3**: Paths must resolve within the project directory
- **AC-4.4**: Invalid paths produce clear error messages explaining the violation
- **AC-4.5**: The same validation logic used for `--filename` (E07-F08) applies to `--path`
- **AC-4.6**: Empty or whitespace-only paths are rejected

---

## 3. Feature Specification

### 3.1 CLI Interface

#### Epic Creation with Custom Path

```bash
shark epic create --title="<title>" [--path="<custom-base-path>"] [other-flags]
```

**New Flag**:
- `--path` (string, optional): Custom base folder path for this epic and all child features/tasks
  - Must be relative to project root
  - Becomes the base for all child items
  - Stored in database as `custom_folder_path`

**Examples**:
```bash
# Organize by roadmap quarters
shark epic create --title="Q1 2025 Goals" --path="docs/roadmap/2025-q1"

# Organize by product area
shark epic create --title="Platform Services" --path="docs/products/platform"

# Use default (no custom path)
shark epic create --title="Core Features"  # Creates in docs/plan/E09/
```

#### Feature Creation with Custom Path

```bash
shark feature create --epic=<epic-key> --title="<title>" [--path="<custom-base-path>"] [other-flags]
```

**New Flag**:
- `--path` (string, optional): Custom base folder path for this feature and all child tasks
  - Must be relative to project root
  - Overrides epic's custom path if specified
  - Becomes the base for all child tasks
  - Stored in database as `custom_folder_path`

**Examples**:
```bash
# Group features by phase
shark feature create --epic=E04 --title="Phase 1 Core" --path="docs/plan/E04-auth/phase-1"

# Group by module
shark feature create --epic=E04 --title="OAuth Module" --path="docs/plan/E04-auth/modules/oauth"

# Inherit from epic (no custom path)
shark feature create --epic=E04 --title="User Management"  # Uses epic's custom_folder_path if set
```

---

### 3.2 Path Resolution Logic

#### Resolution Algorithm

When creating a file (epic, feature, or task), the system determines the base path using this precedence:

**For Epics**:
1. If `--filename` is provided → use exact file path (E07-F08 behavior)
2. If `--path` is provided → use `<custom_folder_path>/<epic-key>/epic.md`
3. Otherwise → use default `docs/plan/<epic-key>/epic.md`

**For Features**:
1. If `--filename` is provided → use exact file path (E07-F08 behavior)
2. If `--path` is provided → use `<custom_folder_path>/<feature-key>/feature.md`
3. If parent epic has `custom_folder_path` → use `<epic-custom-path>/<epic-key>/<feature-key>/feature.md`
4. Otherwise → use default `docs/plan/<epic-key>/<feature-key>/feature.md`

**For Tasks**:
1. If `--filename` is provided → use exact file path (E07-F05 behavior)
2. If parent feature has `custom_folder_path` → use `<feature-custom-path>/<feature-key>/tasks/<task-key>.md`
3. If parent epic has `custom_folder_path` → use `<epic-custom-path>/<epic-key>/<feature-key>/tasks/<task-key>.md`
4. Otherwise → use default `docs/plan/<epic-key>/<feature-key>/tasks/<task-key>.md`

#### Path Resolution Examples

**Scenario 1: Epic with custom path**
```bash
shark epic create --title="Special" --path="docs/special"
# Creates: docs/special/E09/epic.md
# Stores: custom_folder_path = "docs/special"

shark feature create --epic=E09 --title="Feature A"
# Creates: docs/special/E09/E09-F01-feature-a/feature.md (inherits epic's custom path)

shark task create --epic=E09 --feature=F01 --title="Task 1"
# Creates: docs/special/E09/E09-F01-feature-a/tasks/T-E09-F01-001-task-1.md
```

**Scenario 2: Feature overrides epic's custom path**
```bash
# Assume E09 has custom_folder_path = "docs/special"

shark feature create --epic=E09 --title="Feature B" --path="docs/special/phase-2"
# Creates: docs/special/phase-2/E09-F02-feature-b/feature.md
# Stores: custom_folder_path = "docs/special/phase-2" (overrides epic's path)

shark task create --epic=E09 --feature=F02 --title="Task 1"
# Creates: docs/special/phase-2/E09-F02-feature-b/tasks/T-E09-F02-001-task-1.md
```

**Scenario 3: Task overrides with filename**
```bash
# Assume E09-F01 has custom_folder_path = "docs/special/phase-2"

shark task create --epic=E09 --feature=F01 --title="Task 2" --filename="docs/custom/task.md"
# Creates: docs/custom/task.md (exact override)
```

---

### 3.3 File Frontmatter Format

Epic and feature files will include `custom_folder_path` in their YAML frontmatter when a custom path is set.

#### Epic File Frontmatter

**Example**: `docs/roadmap/2025-q1/E09/epic.md`

```yaml
---
epic_key: E09
title: Q1 2025 Roadmap
description: Strategic goals for Q1 2025
status: active
priority: high
custom_folder_path: docs/roadmap/2025-q1
---

# Q1 2025 Roadmap
...
```

**Fields**:
- `custom_folder_path` (optional): Base folder path for this epic and all child features/tasks
- If not present or null, use default path structure

#### Feature File Frontmatter

**Example**: `docs/plan/E04-auth/modules/oauth/E04-F01-oauth-module/feature.md`

```yaml
---
feature_key: E04-F01-oauth-module
epic_key: E04
title: OAuth Module
description: OAuth 2.0 authentication module
status: active
custom_folder_path: docs/plan/E04-auth/modules/oauth
---

# OAuth Module
...
```

**Fields**:
- `custom_folder_path` (optional): Base folder path for this feature and all child tasks
- Overrides parent epic's custom_folder_path if both are set
- If not present or null, inherit from parent epic or use default

#### Task File Frontmatter

Tasks don't have `custom_folder_path` (they use `file_path` from E07-F05 for explicit overrides). Task locations are determined by:
1. Explicit `file_path` (if set via `--filename`)
2. Parent feature's `custom_folder_path`
3. Parent epic's `custom_folder_path`
4. Default structure

---

### 3.4 Database Schema Changes

#### Add `custom_folder_path` Column

Both `epics` and `features` tables need a new column to store the custom base path:

```sql
-- Add to epics table
ALTER TABLE epics ADD COLUMN custom_folder_path TEXT;

-- Add to features table
ALTER TABLE features ADD COLUMN custom_folder_path TEXT;

-- Add indexes for lookups
CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);
CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);
```

**Column Details**:
- **Type**: `TEXT`
- **Nullable**: `YES` (NULL means use default behavior)
- **Purpose**: Stores the custom base folder path for this entity and its children
- **Different from `file_path`**:
  - `file_path` (E07-F08): Exact file location for this entity
  - `custom_folder_path` (this feature): Base folder for this entity and all children

#### Model Updates

**File**: `internal/models/epic.go`
```go
type Epic struct {
    // ... existing fields ...
    FilePath           *string    `db:"file_path"`            // E07-F08: exact file location
    CustomFolderPath   *string    `db:"custom_folder_path"`   // E07-F09: base folder for children
}
```

**File**: `internal/models/feature.go`
```go
type Feature struct {
    // ... existing fields ...
    FilePath           *string    `db:"file_path"`            // E07-F08: exact file location
    CustomFolderPath   *string    `db:"custom_folder_path"`   // E07-F09: base folder for children
}
```

---

### 3.5 Validation Rules

Reuse the existing `ValidateCustomFilename` function with a wrapper for folder path validation:

**Function**: `ValidateFolderPath(path, projectRoot string) (absPath, relPath string, err error)`

**Validation Rules**:

| Rule | Requirement | Error Message |
|------|-------------|---------------|
| **Relative Path** | Must be relative to project root | `path must be relative to project root, got absolute path: /absolute/path` |
| **No Path Traversal** | Cannot contain `..` | `invalid path: contains '..' (path traversal not allowed)` |
| **Within Project** | Must resolve inside project root | `path validation failed: resolves outside project root` |
| **Non-Empty** | After normalization, must be valid | `invalid path: resolved to empty or invalid path` |
| **No Trailing Slash** | Normalized to remove trailing slash | (Auto-corrected, no error) |

**Implementation Location**: `internal/patterns/validator.go` or `internal/utils/path_validation.go`

---

### 3.6 Repository Methods

#### Epic Repository (`internal/repository/epic_repository.go`)

**New Method**:
```go
// GetCustomFolderPath retrieves the custom folder path for an epic
// Returns nil if no custom path is set
func (r *EpicRepository) GetCustomFolderPath(ctx context.Context, epicKey string) (*string, error)
```

**Modified Method**:
```go
// Create now accepts CustomFolderPath in the Epic struct
func (r *EpicRepository) Create(ctx context.Context, epic *models.Epic) error
```

#### Feature Repository (`internal/repository/feature_repository.go`)

**New Method**:
```go
// GetCustomFolderPath retrieves the custom folder path for a feature
// Returns nil if no custom path is set
func (r *FeatureRepository) GetCustomFolderPath(ctx context.Context, featureKey string) (*string, error)
```

**Modified Method**:
```go
// Create now accepts CustomFolderPath in the Feature struct
func (r *FeatureRepository) Create(ctx context.Context, feature *models.Feature) error
```

---

### 3.7 Path Builder Service

Create a new service to centralize path resolution logic:

**File**: `internal/utils/path_builder.go` (or similar)

```go
package utils

type PathBuilder struct {
    projectRoot string
}

// ResolveEpicPath determines the file path for an epic
// Priority: filename > custom_folder_path > default
func (pb *PathBuilder) ResolveEpicPath(
    epicKey string,
    filename *string,           // from --filename flag
    customFolderPath *string,   // from --path flag
) (string, error)

// ResolveFeaturePath determines the file path for a feature
// Priority: filename > feature custom_folder_path > epic custom_folder_path > default
func (pb *PathBuilder) ResolveFeaturePath(
    epicKey string,
    featureKey string,
    filename *string,              // from --filename flag
    featureCustomPath *string,     // from feature's --path flag
    epicCustomPath *string,        // from parent epic's custom_folder_path
) (string, error)

// ResolveTaskPath determines the file path for a task
// Priority: filename > feature custom_folder_path > epic custom_folder_path > default
func (pb *PathBuilder) ResolveTaskPath(
    epicKey string,
    featureKey string,
    taskKey string,
    filename *string,              // from --filename flag
    featureCustomPath *string,     // from parent feature's custom_folder_path
    epicCustomPath *string,        // from parent epic's custom_folder_path
) (string, error)
```

This service encapsulates all path resolution logic, making it testable and reusable across epic, feature, and task creation.

---

### 3.8 Integration Points

#### Epic Creation Command (`internal/cli/commands/epic.go`)

**Changes**:
1. Add `--path` flag to `epic create` command
2. Parse the flag value
3. Validate using `ValidateFolderPath`
4. Pass to epic creator with `CustomFolderPath` field set
5. Use `PathBuilder.ResolveEpicPath` to determine final file location

#### Feature Creation Command (`internal/cli/commands/feature.go`)

**Changes**:
1. Add `--path` flag to `feature create` command
2. Parse the flag value
3. Look up parent epic's `custom_folder_path` from database
4. Validate using `ValidateFolderPath`
5. Pass to feature creator with `CustomFolderPath` field set
6. Use `PathBuilder.ResolveFeaturePath` to determine final file location

#### Task Creation Command (`internal/cli/commands/task.go`)

**Changes**:
1. Look up parent feature's `custom_folder_path` from database
2. Look up parent epic's `custom_folder_path` from database (if feature has none)
3. Use `PathBuilder.ResolveTaskPath` to determine final file location (if no `--filename` provided)

#### Sync Command (`internal/cli/commands/sync.go`)

**Changes**:
1. Update discovery to scan beyond `docs/plan/` for custom path locations
2. Parse `custom_folder_path` from epic/feature frontmatter during discovery
3. Look for child items in custom path locations using PathBuilder
4. Handle conflict resolution when file and database custom paths differ
5. Preserve `custom_folder_path` in frontmatter during sync updates
6. Support multiple scan locations for tasks based on parent custom paths

**Discovery Algorithm**:
```
For each epic file discovered:
  - Parse custom_folder_path from frontmatter
  - Store in database
  - Look for features in: <epic-custom-path>/<epic-key>/* AND default location

For each feature file discovered:
  - Parse custom_folder_path from frontmatter
  - Resolve parent epic's custom_folder_path
  - Look for tasks in: <feature-custom-path>/<feature-key>/tasks/*
    OR <epic-custom-path>/<epic-key>/<feature-key>/tasks/*
    OR default location
```

---

## 4. Implementation Plan

### Phase 1: Foundation (Database & Models)

**Tasks**:
1. Add `custom_folder_path` column to `epics` and `features` tables (with migration)
2. Update `Epic` and `Feature` models with `CustomFolderPath` field
3. Add repository methods: `GetCustomFolderPath` for epics and features
4. Update `Create` methods to accept and store `custom_folder_path`

**Validation**: Database schema verified, models compile, repository tests pass

---

### Phase 2: Path Resolution Logic

**Tasks**:
1. Create `ValidateFolderPath` function (wrapper around existing validation)
2. Create `PathBuilder` service with `ResolveEpicPath`, `ResolveFeaturePath`, `ResolveTaskPath`
3. Write comprehensive unit tests for path resolution edge cases

**Validation**: Path resolution tests cover all scenarios (default, custom, override, inheritance)

---

### Phase 3: CLI Integration (Create Commands)

**Tasks**:
1. Add `--path` flag to `epic create` command
2. Add `--path` flag to `feature create` command
3. Update epic creation flow to use `PathBuilder.ResolveEpicPath`
4. Update feature creation flow to use `PathBuilder.ResolveFeaturePath`
5. Update task creation flow to use `PathBuilder.ResolveTaskPath`

**Validation**: CLI commands accept `--path` flag, files created at correct locations

---

### Phase 4: Sync Integration (Discovery & Synchronization)

**Critical**: This phase is foundational to the system. Without sync integration, custom paths only work for new CLI-created items and won't discover existing files or handle manual reorganization.

**Tasks**:

1. **Update Discovery Logic** (`internal/sync/discovery.go` or `internal/discovery/`):
   - Scan beyond `docs/plan/` - look for epic/feature files in custom locations
   - Parse YAML frontmatter to extract `custom_folder_path` values
   - Build a map of epic/feature custom paths during discovery phase

2. **Update Epic Discovery**:
   - When discovering an epic file, extract `custom_folder_path` from frontmatter
   - If `custom_folder_path` exists in database but not in file, use database value
   - If both exist and differ, apply conflict resolution strategy
   - Store discovered `custom_folder_path` in database during sync

3. **Update Feature Discovery**:
   - Look for features in both default locations AND parent epic's custom path
   - Extract `custom_folder_path` from feature frontmatter
   - Resolve feature path precedence: feature custom path > epic custom path > default
   - Handle features that moved between custom paths

4. **Update Task Discovery**:
   - Look for tasks in both default locations AND parent feature/epic custom paths
   - Use PathBuilder to resolve expected task location based on parent custom paths
   - Scan multiple possible locations:
     - `<feature-custom-path>/<feature-key>/tasks/` (if feature has custom path)
     - `<epic-custom-path>/<epic-key>/<feature-key>/tasks/` (if epic has custom path)
     - `docs/plan/<epic-key>/<feature-key>/tasks/` (default)
   - Handle tasks with explicit `file_path` (E07-F05)

5. **Update Frontmatter Parsing**:
   - Add `custom_folder_path` field to epic/feature frontmatter parsing
   - Write `custom_folder_path` to frontmatter when creating new files
   - Preserve `custom_folder_path` during sync operations

6. **Conflict Resolution**:
   - If file `custom_folder_path` ≠ database `custom_folder_path`, apply strategy:
     - `--strategy=file-wins`: Update database from file
     - `--strategy=db-wins`: Update file from database
     - Default: Warn user and skip update
   - Handle moved files (file exists at new location but DB has old path)

7. **Update Sync Command Options**:
   - Existing flags remain unchanged
   - Sync automatically detects custom paths from frontmatter
   - `--cleanup` should respect custom paths when looking for orphaned files

8. **Integration with PathBuilder**:
   - Sync uses same `PathBuilder` service for consistency
   - Verify discovered file locations match expected paths from PathBuilder

**Validation**:
- Sync discovers epics/features/tasks in custom path locations
- Sync correctly updates `custom_folder_path` in database from file frontmatter
- Sync handles path inheritance (epic → feature → task)
- Sync respects conflict resolution strategies
- Manual file reorganization is reflected after sync
- Existing default-path items continue to sync correctly

---

### Phase 5: Testing & Documentation

**Tasks**:
1. Integration tests for epic creation with `--path`
2. Integration tests for feature creation with `--path` (with and without epic custom path)
3. Integration tests for task creation (with feature and epic custom paths)
4. **Sync integration tests**:
   - `TestSync_DiscoverEpicWithCustomPath`: Discover epic with custom_folder_path in frontmatter
   - `TestSync_DiscoverFeatureInheritingEpicPath`: Feature inherits epic's custom path
   - `TestSync_DiscoverFeatureWithOwnCustomPath`: Feature overrides epic's custom path
   - `TestSync_DiscoverTasksInCustomPaths`: Tasks discovered in custom locations
   - `TestSync_ConflictResolution_CustomPath`: Handle file vs DB custom_folder_path conflicts
   - `TestSync_MovedFiles`: Handle files moved to custom locations
5. Update `docs/CLI_REFERENCE.md` with `--path` flag documentation
6. Update `CLAUDE.md` with new command syntax and examples
7. Add examples to command help text
8. Document sync behavior with custom paths

**Validation**: All tests pass, documentation complete and accurate

---

## 5. Test Strategy

### 5.1 Unit Tests

#### Path Builder Tests (`internal/utils/path_builder_test.go`)

**Test Cases**:
- `TestResolveEpicPath_Default`: No flags → default path
- `TestResolveEpicPath_WithFilename`: `--filename` takes precedence
- `TestResolveEpicPath_WithCustomPath`: `--path` creates custom base
- `TestResolveEpicPath_BothFlags`: `--filename` overrides `--path`
- `TestResolveFeaturePath_Default`: No custom paths → default
- `TestResolveFeaturePath_InheritsEpicPath`: Uses epic's custom path
- `TestResolveFeaturePath_OverridesEpicPath`: Feature `--path` overrides epic
- `TestResolveFeaturePath_WithFilename`: `--filename` takes precedence over all
- `TestResolveTaskPath_Default`: No custom paths → default
- `TestResolveTaskPath_InheritsFeaturePath`: Uses feature's custom path
- `TestResolveTaskPath_InheritsEpicPath`: Uses epic's custom path (no feature path)
- `TestResolveTaskPath_WithFilename`: `--filename` takes precedence

#### Validation Tests (`internal/utils/path_validation_test.go`)

**Test Cases**:
- `TestValidateFolderPath_ValidRelative`: Accept `docs/custom/path`
- `TestValidateFolderPath_AbsolutePath`: Reject `/absolute/path`
- `TestValidateFolderPath_PathTraversal`: Reject `docs/../../outside`
- `TestValidateFolderPath_OutsideProject`: Reject paths resolving outside project
- `TestValidateFolderPath_EmptyPath`: Reject empty string
- `TestValidateFolderPath_TrailingSlash`: Auto-normalize `docs/path/` → `docs/path`

---

### 5.2 Integration Tests

#### Epic Creation Tests (`internal/cli/commands/epic_test.go`)

**Test Cases**:
- `TestEpicCreate_WithCustomPath`: `--path="docs/custom"` creates files correctly
- `TestEpicCreate_WithInvalidPath`: Reject invalid paths with clear errors
- `TestEpicCreate_WithFilenameAndPath`: `--filename` takes precedence
- `TestEpicCreate_Default`: No `--path` uses default location
- `TestEpicCreate_CustomPath_StoresInDB`: Verify `custom_folder_path` saved

#### Feature Creation Tests (`internal/cli/commands/feature_test.go`)

**Test Cases**:
- `TestFeatureCreate_InheritsEpicCustomPath`: Feature uses epic's custom path
- `TestFeatureCreate_OverridesEpicCustomPath`: Feature `--path` overrides epic
- `TestFeatureCreate_WithCustomPath_NoEpicPath`: Feature custom path only
- `TestFeatureCreate_WithFilename`: `--filename` overrides all paths
- `TestFeatureCreate_Default`: No custom paths → default location
- `TestFeatureCreate_CustomPath_StoresInDB`: Verify `custom_folder_path` saved

#### Task Creation Tests (`internal/cli/commands/task_test.go`)

**Test Cases**:
- `TestTaskCreate_InheritsFeatureCustomPath`: Task uses feature's custom path
- `TestTaskCreate_InheritsEpicCustomPath`: Task uses epic's custom path (no feature path)
- `TestTaskCreate_WithFilename`: `--filename` overrides inherited paths
- `TestTaskCreate_Default`: No custom paths → default location
- `TestTaskCreate_FeaturePathOverridesEpicPath`: Correct precedence

---

### 5.3 End-to-End Scenarios

#### Scenario 1: Epic with Custom Path Hierarchy

```bash
# Create epic with custom base path
shark epic create --title="Q1 Roadmap" --path="docs/roadmap/2025-q1"
# Expected: docs/roadmap/2025-q1/E09/epic.md

# Create feature (should inherit epic's custom path)
shark feature create --epic=E09 --title="Authentication"
# Expected: docs/roadmap/2025-q1/E09/E09-F01-authentication/feature.md

# Create task (should inherit from feature/epic hierarchy)
shark task create --epic=E09 --feature=F01 --title="Login Flow"
# Expected: docs/roadmap/2025-q1/E09/E09-F01-authentication/tasks/T-E09-F01-001-login-flow.md

# Verify all files exist at expected paths
```

#### Scenario 2: Feature Overrides Epic Path

```bash
# Assume E09 has custom_folder_path = "docs/roadmap/2025-q1"

# Create feature with override path
shark feature create --epic=E09 --title="OAuth Module" --path="docs/roadmap/2025-q1/modules/oauth"
# Expected: docs/roadmap/2025-q1/modules/oauth/E09-F02-oauth-module/feature.md

# Create task (should use feature's custom path, not epic's)
shark task create --epic=E09 --feature=F02 --title="OAuth Provider"
# Expected: docs/roadmap/2025-q1/modules/oauth/E09-F02-oauth-module/tasks/T-E09-F02-001-oauth-provider.md
```

#### Scenario 3: Task Filename Overrides All

```bash
# Assume E09 has custom_folder_path = "docs/roadmap/2025-q1"
# Assume E09-F01 has custom_folder_path = "docs/roadmap/2025-q1/phase-1"

# Create task with explicit filename (overrides all inheritance)
shark task create --epic=E09 --feature=F01 --title="Special Task" --filename="docs/special/task.md"
# Expected: docs/special/task.md (exact override)
```

#### Scenario 4: Sync with Custom Paths

```bash
# Manually create epic file with custom_folder_path in frontmatter
mkdir -p docs/strategic/2025/E10
cat > docs/strategic/2025/E10/epic.md <<EOF
---
epic_key: E10
title: Strategic Initiatives
custom_folder_path: docs/strategic/2025
---
# Strategic Initiatives
EOF

# Manually create feature file inheriting epic's custom path
mkdir -p docs/strategic/2025/E10/E10-F01-innovation
cat > docs/strategic/2025/E10/E10-F01-innovation/feature.md <<EOF
---
feature_key: E10-F01-innovation
epic_key: E10
title: Innovation Projects
---
# Innovation Projects
EOF

# Manually create task in custom location
mkdir -p docs/strategic/2025/E10/E10-F01-innovation/tasks
cat > docs/strategic/2025/E10/E10-F01-innovation/tasks/T-E10-F01-001-ai-research.md <<EOF
---
task_key: T-E10-F01-001
feature_key: E10-F01
epic_key: E10
title: AI Research Initiative
---
# AI Research Initiative
EOF

# Run sync to discover and import all custom path items
shark sync

# Verify database has custom_folder_path set for E10
shark epic get E10 --json | jq '.custom_folder_path'
# Expected: "docs/strategic/2025"

# Verify feature and task were discovered
shark feature get E10-F01 --json
shark task get T-E10-F01-001 --json
```

---

## 6. Success Criteria

### Functional Requirements

**Creation Commands**:
- [ ] `shark epic create --path=<folder>` creates epic and stores `custom_folder_path` in database
- [ ] Features created under an epic with custom path inherit that path by default
- [ ] Features can override epic's custom path with their own `--path` parameter
- [ ] Tasks inherit custom paths from features and epics (feature path takes precedence)
- [ ] Tasks with `--filename` override all inherited paths
- [ ] Path validation prevents absolute paths, path traversal, and out-of-bounds paths
- [ ] Default behavior (no `--path`) remains unchanged (backward compatible)

**Sync Integration**:
- [ ] `shark sync` discovers epics with `custom_folder_path` in frontmatter
- [ ] Sync discovers features in custom path locations (inheriting from epic)
- [ ] Sync discovers tasks in custom path locations (inheriting from feature/epic)
- [ ] Sync updates database `custom_folder_path` from file frontmatter
- [ ] Sync writes `custom_folder_path` to file frontmatter when creating/updating files
- [ ] Sync handles conflict resolution when file and database custom paths differ
- [ ] Sync respects `--strategy=file-wins` and `--strategy=db-wins` for custom path conflicts
- [ ] Manually reorganized files are discovered after sync

### Code Quality

- [ ] `PathBuilder` service centralizes all path resolution logic
- [ ] Validation logic reuses existing `ValidateCustomFilename` patterns
- [ ] All repository methods handle `custom_folder_path` correctly
- [ ] Database schema includes `custom_folder_path` column with proper indexes
- [ ] Unit tests cover all path resolution scenarios
- [ ] Integration tests verify inheritance and override behavior

### Documentation

- [ ] `docs/CLI_REFERENCE.md` documents `--path` flag for epic and feature commands
- [ ] `CLAUDE.md` updated with examples of custom path usage
- [ ] Command help text includes `--path` flag with clear descriptions
- [ ] README mentions organizational flexibility (if applicable)

### Backward Compatibility

- [ ] Existing commands without `--path` continue to work unchanged
- [ ] Default file paths remain identical to current behavior
- [ ] No breaking changes to CLI syntax or database schema
- [ ] NULL values in `custom_folder_path` indicate default behavior

---

## 7. Edge Cases & Error Handling

| Edge Case | Input | Expected Behavior |
|-----------|-------|------------------|
| Empty path | `--path=""` | Use default behavior (treat as if flag not provided) |
| Absolute path | `--path="/absolute"` | Reject: "path must be relative to project root" |
| Path traversal | `--path="../../outside"` | Reject: "contains '..' (path traversal not allowed)" |
| Path outside project | `--path="../../../outside"` | Reject: "resolves outside project root" |
| Both `--path` and `--filename` | Both flags provided | Use `--filename` (takes precedence), ignore `--path` |
| Trailing slash | `--path="docs/custom/"` | Auto-normalize to `docs/custom` |
| Nested custom paths | Epic has custom path, feature has custom path | Feature path fully replaces epic path (no concatenation) |
| Task with feature custom path | Feature has custom path, create task | Task inherits feature's custom path |
| Task with epic custom path only | Epic has custom path, feature has none | Task inherits epic's custom path |
| Database stores NULL | `custom_folder_path` is NULL | Use default path resolution |

---

## 8. Dependencies & Sequencing

### External Dependencies

- **E07-F08 (Custom Filenames for Epics & Features)**: REQUIRED
  - Provides `file_path` column and `--filename` flag infrastructure
  - Provides validation patterns for custom paths
  - Must be complete before this feature begins

- **E07-F05 (Custom Filenames for Tasks)**: REQUIRED
  - Provides task `--filename` functionality
  - Must be complete for task override behavior to work

### Prerequisite Work

1. ✓ E07-F08 implemented (epic/feature `--filename` support)
2. ✓ E07-F05 implemented (task `--filename` support)
3. ✓ Database migration capability exists
4. Create `custom_folder_path` columns in database
5. Create `PathBuilder` service
6. Implement CLI flag parsing

### Implementation Sequence

**Phase 1** → **Phase 2** → **Phase 3** → **Phase 4** → **Phase 5** (sequential, no parallelization)

**Critical Note**: Phase 4 (Sync Integration) is foundational and must be completed before the feature is considered production-ready. Without it, custom paths only work for CLI-created items and won't discover existing files.

---

## 9. Risks & Mitigations

### Risk 1: Confusion Between `--path` and `--filename`

**Risk**: Users may not understand the difference between `--path` (base folder for children) and `--filename` (exact file location).

**Mitigation**:
- Clear documentation with examples
- Error message when both flags used: "Warning: --filename takes precedence over --path"
- Help text explicitly explains the difference
- Name the flag carefully: `--path` implies "base path" while `--filename` implies "exact file"

### Risk 2: Migration Complexity

**Risk**: Adding `custom_folder_path` column requires database migration for existing projects.

**Mitigation**:
- Column is nullable (NULL = default behavior)
- Migration script provided in documentation
- `shark init` automatically creates schema with new columns for new projects
- Graceful degradation: code checks for NULL and falls back to defaults

### Risk 3: Path Resolution Bugs

**Risk**: Complex inheritance rules (epic → feature → task) may have edge case bugs.

**Mitigation**:
- Centralize all logic in `PathBuilder` service (single source of truth)
- Comprehensive unit tests covering all precedence scenarios
- Integration tests verify end-to-end workflows
- Clear precedence rules documented in code and user docs

---

## 10. Future Enhancements

### Potential Follow-Ups

These enhancements can be added after the core feature is complete:

1. **Epic Index Generation**: Update epic index generation (`/generate-epic-index`) to follow custom paths when discovering epics
2. **Move Command**: `shark epic move --path=<new-path>` to relocate entire epic hierarchies and update all child paths
3. **Copy Command**: `shark epic copy --new-key=E10 --path=<new-path>` to duplicate epics with new base paths
4. **Path Templates**: `--path-template="docs/{{year}}/{{quarter}}/{{epic-key}}"` with variable substitution for dynamic paths
5. **Validation Command**: `shark validate paths` to verify all files exist at expected locations based on custom paths
6. **UI/Dashboard**: Visual representation of custom folder hierarchies and path inheritance
7. **Bulk Update**: `shark epic update-path E09 --new-path="docs/new-location"` to move existing epics and update all references

---

## 11. References

- **Original Requirement**: `docs/plan/override-path.md`
- **E07-F08 PRD**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md`
- **E07-F05 Implementation**: Task custom filename feature
- **CLI Reference**: `docs/CLI_REFERENCE.md`
- **Database Schema**: `internal/db/db.go`

---

## Appendix A: Command Examples

### Example 1: Organize Epics by Quarter

```bash
# Q1 2025 Epic
shark epic create --title="Q1 2025 OKRs" --path="docs/roadmap/2025/q1"
# Creates: docs/roadmap/2025/q1/E09/epic.md

# Features automatically inherit the custom base
shark feature create --epic=E09 --title="User Growth"
# Creates: docs/roadmap/2025/q1/E09/E09-F01-user-growth/feature.md

shark feature create --epic=E09 --title="Performance"
# Creates: docs/roadmap/2025/q1/E09/E09-F02-performance/feature.md

# Q2 2025 Epic (different structure)
shark epic create --title="Q2 2025 OKRs" --path="docs/roadmap/2025/q2"
# Creates: docs/roadmap/2025/q2/E10/epic.md
```

### Example 2: Organize Features by Module

```bash
# Authentication Epic (default path)
shark epic create --title="Authentication System"
# Creates: docs/plan/E11-authentication-system/epic.md

# Organize features by module
shark feature create --epic=E11 --title="OAuth Module" --path="docs/plan/E11-authentication-system/modules/oauth"
# Creates: docs/plan/E11-authentication-system/modules/oauth/E11-F01-oauth-module/feature.md

shark feature create --epic=E11 --title="SAML Module" --path="docs/plan/E11-authentication-system/modules/saml"
# Creates: docs/plan/E11-authentication-system/modules/saml/E11-F02-saml-module/feature.md

# Tasks inherit module structure
shark task create --epic=E11 --feature=F01 --title="Google OAuth"
# Creates: docs/plan/E11-authentication-system/modules/oauth/E11-F01-oauth-module/tasks/T-E11-F01-001-google-oauth.md
```

### Example 3: Override with Task Filename

```bash
# Epic and feature with custom paths
shark epic create --title="Platform" --path="docs/platform"
shark feature create --epic=E12 --title="Core Services" --path="docs/platform/services"

# Task with exact filename override
shark task create --epic=E12 --feature=F01 --title="Special Task" --filename="docs/special-tasks/investigation.md"
# Creates: docs/special-tasks/investigation.md (ignores inheritance)
```

---

## Appendix B: Database Schema

### Updated Schema

```sql
-- Epics table with custom_folder_path
CREATE TABLE epics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'completed', 'archived')),
    priority TEXT NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    business_value TEXT CHECK (business_value IN ('high', 'medium', 'low')),
    file_path TEXT,                    -- E07-F08: exact file location
    custom_folder_path TEXT,           -- E07-F09: base folder for children
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Features table with custom_folder_path
CREATE TABLE features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'completed', 'archived')),
    progress_pct REAL NOT NULL DEFAULT 0.0 CHECK (progress_pct >= 0 AND progress_pct <= 100),
    execution_order INTEGER NULL,
    file_path TEXT,                    -- E07-F08: exact file location
    custom_folder_path TEXT,           -- E07-F09: base folder for children
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);
CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);
CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);
CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);
```

---

*Last Updated*: 2025-12-19
