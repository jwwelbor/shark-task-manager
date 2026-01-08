# Path and Filename Architecture Design

## Overview

The current implementation conflates paths and filenames, leading to confusion and broken functionality. This document defines the proper architecture for how paths and filenames should work throughout the shark system.

## Core Principles

### 1. Separation of Concerns

- **Filename**: Only the file name (e.g., `epic.md`, `feature.md`, `my-task.md`)
- **Path**: The directory structure leading to the file
- **Full Path**: Computed as `{path}/{filename}`

### 2. Hierarchical Path Composition

Paths should compose hierarchically with inheritance:

```
Epic:    {docs_root}/{epic_custom_path OR epic_default_path}/{epic_filename}
Feature: {docs_root}/{epic_path}/{feature_custom_path OR feature_default_path}/{feature_filename}
Task:    {docs_root}/{epic_path}/{feature_path}/{task_custom_path OR task_default_path}/{task_filename}
```

### 3. Custom Overrides at Each Level

Each entity can override its own path segment:
- Epic can set `custom_path` to override default `docs/plan/E01-epic-slug/`
- Feature can set `custom_path` to override default `{epic_path}/F01-feature-slug/`
- Task can set `custom_path` to override default `{feature_path}/tasks/`

### 4. Filename Behavior

- Custom filename can be provided via `--filename` flag
- Default filename:
  - Epic: `epic.md`
  - Feature: `feature.md`
  - Task: `{task-key}.md` (e.g., `T-E07-F01-001.md`)

## Database Schema

### Current Schema (needs update)

```sql
-- Epics
file_path TEXT              -- Currently stores full path (WRONG)
custom_folder_path TEXT     -- Currently unused (WRONG)

-- Features
file_path TEXT              -- Currently stores full path (WRONG)
custom_folder_path TEXT     -- Currently unused (WRONG)

-- Tasks
file_path TEXT              -- Currently stores full path (WRONG)
custom_folder_path TEXT     -- Doesn't exist yet
```

### Proposed Schema

```sql
-- Epics
custom_path TEXT            -- Custom path segment (e.g., "docs/roadmap/2025-q1")
filename TEXT               -- Just the filename (e.g., "epic.md")
-- Computed: full_path = {custom_path OR default}/{filename}

-- Features
custom_path TEXT            -- Custom path segment (e.g., "authentication")
filename TEXT               -- Just the filename (e.g., "feature.md")
-- Computed: full_path = {epic_path}/{custom_path OR default}/{filename}

-- Tasks
custom_path TEXT            -- Custom path segment (e.g., "backend-tasks")
filename TEXT               -- Just the filename (e.g., "T-E07-F01-001.md")
-- Computed: full_path = {feature_path}/{custom_path OR default}/{filename}
```

## Path Resolution Logic

### Function: `ResolvePath(entity Entity) string`

```go
func ResolvePath(entity Entity) string {
    docsRoot := config.GetDocsRoot() // e.g., "docs"

    switch entity.Type {
    case Epic:
        path := entity.CustomPath
        if path == "" {
            path = fmt.Sprintf("plan/%s", entity.Key) // Default: plan/E01
        }
        return filepath.Join(docsRoot, path, entity.Filename)

    case Feature:
        epicPath := getEpicPathSegment(entity.EpicID) // Recursive: get epic's path
        featurePath := entity.CustomPath
        if featurePath == "" {
            featurePath = entity.Key // Default: E01-F01-feature-slug
        }
        return filepath.Join(docsRoot, epicPath, featurePath, entity.Filename)

    case Task:
        featurePath := getFeaturePathSegment(entity.FeatureID) // Recursive: get feature's full path
        taskPath := entity.CustomPath
        if taskPath == "" {
            taskPath = "tasks" // Default: tasks/
        }
        return filepath.Join(docsRoot, featurePath, taskPath, entity.Filename)
    }
}
```

### Path Inheritance

- **Feature** inherits from Epic: Epic's custom_path becomes part of Feature's full path
- **Task** inherits from Feature: Feature's full path becomes part of Task's full path
- Each level can override its own segment but inherits parent segments

## Create/Update Behavior

### During Create

1. **Parse flags:**
   - `--path` → `custom_path` (just the path segment for this level)
   - `--filename` → `filename` (just the filename)

2. **Calculate full path:**
   - Use `ResolvePath()` to compute full file path
   - Check if file exists

3. **If file exists:**
   - Just create database entry referencing existing file
   - Output: "Referenced existing file at {full_path}"

4. **If file doesn't exist:**
   - Create placeholder file from template
   - Create database entry
   - Output: "Created placeholder at {full_path}. Update with details."

### During Update

1. **Parse flags:**
   - `--path` → update `custom_path` in database
   - `--filename` → update `filename` in database

2. **Recalculate full path:**
   - Use `ResolvePath()` to compute new full file path
   - Compare with old file path

3. **If path changed:**
   - **Option A (safer):** Ask user if they want to move the file
   - **Option B (automatic):** Move file automatically and update references
   - **Option C (reference only):** Just update database, don't move file (warn user)

4. **Update database:**
   - Set new `custom_path` and/or `filename`
   - Trigger recalculation of all child entities' paths (cascading update)

## Default Path Structures

### Epic Defaults
- Path: `docs/plan/{epic-key}`
- Filename: `epic.md`
- Full: `docs/plan/E01-epic-slug/epic.md`

### Feature Defaults
- Path: `{epic-path}/{feature-key}`
- Filename: `feature.md`
- Full: `docs/plan/E01-epic-slug/E01-F01-feature-slug/feature.md`

### Task Defaults
- Path: `{feature-path}/tasks`
- Filename: `{task-key}.md`
- Full: `docs/plan/E01-epic-slug/E01-F01-feature-slug/tasks/T-E01-F01-001.md`

## Example Scenarios

### Scenario 1: Default Structure

```bash
# Create epic (default)
shark epic create "User Management"
# Path: docs/plan/E01-user-management/epic.md

# Create feature (default, inherits epic path)
shark feature create --epic=E01 "Authentication"
# Path: docs/plan/E01-user-management/E01-F01-authentication/feature.md

# Create task (default, inherits feature path)
shark task create --epic=E01 --feature=F01 "JWT Token Service"
# Path: docs/plan/E01-user-management/E01-F01-authentication/tasks/T-E01-F01-001.md
```

### Scenario 2: Custom Roadmap Structure

```bash
# Create epic with custom path
shark epic create "Q1 2025 Goals" --path="roadmap/2025-q1"
# Path: docs/roadmap/2025-q1/epic.md

# Create feature (inherits epic's custom path)
shark feature create --epic=E02 "User Growth"
# Path: docs/roadmap/2025-q1/E02-F01-user-growth/feature.md

# Create feature with its own custom path
shark feature create --epic=E02 "Retention" --path="metrics/retention"
# Path: docs/roadmap/2025-q1/metrics/retention/feature.md
```

### Scenario 3: Custom Filenames

```bash
# Create epic with custom filename
shark epic create "API v2" --filename="api-v2-specification.md"
# Path: docs/plan/E03-api-v2/api-v2-specification.md

# Create feature with custom path AND filename
shark feature create --epic=E03 "GraphQL" --path="graphql" --filename="graphql-schema.md"
# Path: docs/plan/E03-api-v2/graphql/graphql-schema.md
```

### Scenario 4: Referencing Existing Files

```bash
# File already exists at docs/design/auth-spec.md
shark feature create --epic=E01 "Auth Spec" --path="../design" --filename="auth-spec.md"
# Output: "Referenced existing file at docs/design/auth-spec.md"
# (No new file created, just database entry)
```

### Scenario 5: Updating Paths

```bash
# Move feature to different organization
shark feature update E01-F01 --path="archived/old-features"
# Triggers:
#   1. Confirm: "This will move E01-F01-authentication from docs/plan/E01/E01-F01-authentication to docs/plan/E01/archived/old-features. Continue? [y/N]"
#   2. Move file on disk
#   3. Update database
#   4. Update all child tasks' paths (cascading)
```

## Configuration

### .sharkconfig.json

```json
{
  "docs_root": "docs",
  "default_epic_path": "plan",
  "default_feature_path": "",  // Empty = use feature key as path segment
  "default_task_path": "tasks",
  "default_epic_filename": "epic.md",
  "default_feature_filename": "feature.md",
  "default_task_filename_pattern": "{task-key}.md"
}
```

## Migration Strategy

### Phase 1: Add New Columns (Non-Breaking)

1. Add `custom_path` and `filename` columns to all tables
2. Keep `file_path` for backward compatibility
3. Populate new columns from existing `file_path` (split into path + filename)

### Phase 2: Update Path Resolution (Non-Breaking)

1. Implement `ResolvePath()` function
2. Update all create/update commands to use new columns
3. Maintain `file_path` as computed/cached value for performance

### Phase 3: Deprecate Old Column (Breaking)

1. Remove `file_path` column (after several releases)
2. Always compute paths on-the-fly or cache in application layer

## Implementation Checklist

- [ ] Update database schema (add custom_path, filename columns)
- [ ] Implement path resolution function with inheritance
- [ ] Update epic create command
- [ ] Update epic update command
- [ ] Update feature create command
- [ ] Update feature update command
- [ ] Update task create command
- [ ] Update task update command
- [ ] Add configuration options for defaults
- [ ] Handle file existence checking
- [ ] Handle file moving on path updates
- [ ] Add tests for path resolution logic
- [ ] Add tests for inheritance behavior
- [ ] Update CLI documentation
- [ ] Create migration guide for existing projects

## Open Questions

1. **File moving behavior**: Should `update --path` automatically move files, ask for confirmation, or just update database?
2. **Cascading updates**: When epic path changes, should it automatically update all child features and tasks?
3. **Backward compatibility**: How long to maintain `file_path` column before deprecating?
4. **Validation**: Should we validate that parent paths exist when creating children?
5. **Symbolic links**: Should we support symlinks for cases where files need to exist in multiple locations?
