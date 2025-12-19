# Backend Design: Custom Folder Base Paths

**Epic**: E07-Enhancements
**Feature**: E07-F09-Custom Folder Base Paths
**Date**: 2025-12-19
**Author**: backend-architect

---

## Interface Overview

This feature extends the Shark Task Manager CLI to support custom base folder paths for epics and features. It provides new command-line flags, a path resolution service, and repository methods to manage hierarchical path inheritance.

**Interface Type**: CLI + Library (internal service layer)

---

## Codebase Analysis

### Existing Related Interfaces

- **Epic Create Command** at `internal/cli/commands/epic.go:82-95` - Creates epics with auto-assigned keys
- **Feature Create Command** at `internal/cli/commands/feature.go` - Creates features under epics
- **Task Create Command** at `internal/cli/commands/task.go` - Creates tasks with --filename support (E07-F05)
- **Epic Repository** at `internal/repository/epic_repository.go:11-19` - CRUD operations for epics
- **Feature Repository** at `internal/repository/feature_repository.go` - CRUD operations for features
- **Pattern Validator** at `internal/patterns/validator.go` - Path and pattern validation infrastructure

### Naming Patterns Found

- **CLI Flags**: lowercase with hyphens (`--description`, `--status`, `--force`)
- **Flag Variables**: camelCase (`epicCreateDescription`, `forceDelete`)
- **Repository Methods**: PascalCase (`Create`, `GetByKey`, `GetCustomFolderPath`)
- **Service Functions**: PascalCase public, camelCase private
- **Error Messages**: Descriptive with context, prefixed with "Error:"

### Extension vs. New Code Decision

**Extend Existing**:
- Add `--path` flag to existing `epic create` and `feature create` commands
- Add methods to existing `EpicRepository` and `FeatureRepository`
- Update existing model structs (`Epic`, `Feature`)

**Create New**:
- `PathBuilder` service in `internal/utils/path_builder.go` (new file)
- Path validation functions in `internal/utils/path_validation.go` (new file)

**Rationale**: Minimal new code, maximum reuse of existing patterns

---

## Data Structures

### Epic (Extended)

**Purpose**: Represents a top-level project organization unit with optional custom folder path

**New Field**:
| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| CustomFolderPath | *string | No | ValidateFolderPath | Custom base directory for this epic and children. NULL = default behavior. |

**Existing Fields** (for context):
| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| ID | int64 | Yes (auto) | N/A | Unique numeric identifier |
| Key | string | Yes | ValidateEpicKey | Epic key (E##) |
| Title | string | Yes | Non-empty | Epic title |
| FilePath | *string | No | ValidateCustomFilename | Exact file path (E07-F08) |

---

### Feature (Extended)

**Purpose**: Represents a mid-level feature within an epic with optional custom folder path

**New Field**:
| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| CustomFolderPath | *string | No | ValidateFolderPath | Custom base directory for this feature and child tasks. NULL = inherit from epic or use default. |

**Existing Fields** (for context):
| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| ID | int64 | Yes (auto) | N/A | Unique numeric identifier |
| EpicID | int64 | Yes | FK constraint | Parent epic reference |
| Key | string | Yes | ValidateFeatureKey | Feature key (E##-F##) |
| Title | string | Yes | Non-empty | Feature title |
| FilePath | *string | No | ValidateCustomFilename | Exact file path (E07-F08) |

---

### PathResolutionInput

**Purpose**: Input structure for PathBuilder service methods

**Fields**:
| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| EpicKey | string | Yes (epics/features/tasks) | ValidateEpicKey | Epic key for context |
| FeatureKey | string | Conditional (features/tasks) | ValidateFeatureKey | Feature key for context |
| TaskKey | string | Conditional (tasks) | ValidateTaskKey | Task key for context |
| Filename | *string | No | ValidateCustomFilename | Explicit file path (highest precedence) |
| CustomFolderPath | *string | No | ValidateFolderPath | Custom base folder path |
| ParentEpicCustomPath | *string | No | N/A | Parent epic's custom folder path (for features) |
| ParentFeatureCustomPath | *string | No | N/A | Parent feature's custom folder path (for tasks) |

---

## CLI Commands

### epic create

**Purpose**: Create a new epic with optional custom base folder path

**Usage**: `shark epic create <title> [--path <custom-folder-path>] [--description <desc>]`

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| title | Yes | Epic title (positional argument) |

**Options**:
| Option | Short | Type | Default | Description |
|--------|-------|------|---------|-------------|
| --path | N/A | string | NULL | Custom base folder path for this epic and children. Relative to project root. |
| --description | N/A | string | NULL | Epic description (existing flag) |

**Output**:
- **Human-readable** (default): Success message with epic key and file path
- **JSON** (`--json` flag): Complete Epic object with all fields

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success - Epic created |
| 1 | Invalid arguments or validation failure |
| 2 | Database error |

**Processing Steps**:
1. Parse `<title>` positional argument
2. Parse `--path` flag (optional)
3. Validate path using `ValidateFolderPath(path, projectRoot)` if provided
4. Auto-assign next epic key (E##)
5. Resolve file path using `PathBuilder.ResolveEpicPath(epicKey, nil, customFolderPath)`
6. Create `Epic` struct with `CustomFolderPath` field set
7. Call `EpicRepository.Create(ctx, epic)`
8. Create epic.md file at resolved path with frontmatter
9. Output success message or JSON

**Examples**:
```bash
# Default behavior (no custom path)
shark epic create "User Authentication"
# Output: Created epic E09 at docs/plan/E09/epic.md

# With custom base folder path
shark epic create "Q1 2025 Roadmap" --path="docs/roadmap/2025-q1"
# Output: Created epic E09 at docs/roadmap/2025-q1/E09/epic.md

# JSON output
shark epic create "Platform Services" --path="docs/products/platform" --json
# Output: {"key": "E09", "title": "Platform Services", "custom_folder_path": "docs/products/platform", ...}
```

**Errors**:
| Condition | Error Message |
|-----------|---------------|
| Path is absolute | "Error: path must be relative to project root, got absolute path: /..." |
| Path contains `..` | "Error: invalid path: contains '..' (path traversal not allowed)" |
| Path resolves outside project | "Error: path validation failed: resolves outside project root" |
| Empty path string | Treat as NULL (use default behavior) |
| Database insert fails | "Error: Failed to create epic. Run with --verbose for details." |

---

### feature create

**Purpose**: Create a new feature with optional custom base folder path, inheriting from parent epic if not specified

**Usage**: `shark feature create --epic=<epic-key> <title> [--path <custom-folder-path>] [--description <desc>]`

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| title | Yes | Feature title (positional argument) |

**Options**:
| Option | Short | Type | Default | Description |
|--------|-------|------|---------|-------------|
| --epic | N/A | string | (required) | Parent epic key (E##) |
| --path | N/A | string | NULL | Custom base folder path for this feature and child tasks. Overrides epic's custom path if both are set. |
| --description | N/A | string | NULL | Feature description (existing flag) |

**Output**:
- **Human-readable**: Success message with feature key and file path
- **JSON** (`--json` flag): Complete Feature object with all fields

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success - Feature created |
| 1 | Invalid arguments, epic not found, or validation failure |
| 2 | Database error |

**Processing Steps**:
1. Parse `--epic` flag and validate epic exists
2. Parse `<title>` positional argument
3. Parse `--path` flag (optional)
4. Fetch parent epic's `custom_folder_path` from database
5. Validate path using `ValidateFolderPath(path, projectRoot)` if provided
6. Auto-assign next feature key (E##-F##)
7. Resolve file path using `PathBuilder.ResolveFeaturePath(epicKey, featureKey, nil, customFolderPath, epicCustomPath)`
8. Create `Feature` struct with `CustomFolderPath` field set
9. Call `FeatureRepository.Create(ctx, feature)`
10. Create feature.md file at resolved path with frontmatter
11. Output success message or JSON

**Examples**:
```bash
# Default behavior (inherit from epic's custom path or use default)
shark feature create --epic=E09 "User Management"
# If E09 has custom_folder_path="docs/roadmap/2025-q1":
# Output: Created feature E09-F01 at docs/roadmap/2025-q1/E09/E09-F01-user-management/feature.md

# Override epic's custom path with feature-specific path
shark feature create --epic=E09 "OAuth Module" --path="docs/roadmap/2025-q1/modules/oauth"
# Output: Created feature E09-F02 at docs/roadmap/2025-q1/modules/oauth/E09-F02-oauth-module/feature.md

# Epic has no custom path, feature sets one
shark feature create --epic=E04 "Phase 1 Core" --path="docs/plan/E04-auth/phase-1"
# Output: Created feature E04-F03 at docs/plan/E04-auth/phase-1/E04-F03-phase-1-core/feature.md
```

**Errors**:
| Condition | Error Message |
|-----------|---------------|
| Epic not found | "Error: Epic E99 not found" |
| Path validation failures | (same as epic create) |
| Database insert fails | "Error: Failed to create feature. Run with --verbose for details." |

---

### task create (Modified)

**Purpose**: Create a new task with path inherited from feature or epic custom paths (no new flag, just updated path resolution)

**Usage**: `shark task create --epic=<epic-key> --feature=<feature-key> <title> [--filename <exact-path>] [options]`

**Changes**:
- No new flags added
- Path resolution logic updated to use `PathBuilder.ResolveTaskPath()`
- Fetches parent feature's `custom_folder_path` and parent epic's `custom_folder_path` from database
- Passes to PathBuilder for resolution

**Processing Steps** (updated):
1-4. (existing steps: parse flags, validate)
5. **NEW**: Fetch parent feature's `custom_folder_path` from database
6. **NEW**: Fetch parent epic's `custom_folder_path` from database (if feature has no custom path)
7. **MODIFIED**: Resolve file path using `PathBuilder.ResolveTaskPath(epicKey, featureKey, taskKey, filename, featureCustomPath, epicCustomPath)`
8-10. (existing steps: create Task, insert, create file)

**Examples**:
```bash
# Task inherits from feature's custom path
# (Feature E09-F02 has custom_folder_path="docs/roadmap/2025-q1/modules/oauth")
shark task create --epic=E09 --feature=F02 "Google OAuth Integration"
# Output: Created task T-E09-F02-001 at docs/roadmap/2025-q1/modules/oauth/E09-F02-oauth-module/tasks/T-E09-F02-001-google-oauth-integration.md

# Task inherits from epic's custom path (feature has no custom path)
# (Epic E09 has custom_folder_path="docs/roadmap/2025-q1", Feature E09-F01 has custom_folder_path=NULL)
shark task create --epic=E09 --feature=F01 "Setup Authentication DB"
# Output: Created task T-E09-F01-001 at docs/roadmap/2025-q1/E09/E09-F01-user-management/tasks/T-E09-F01-001-setup-authentication-db.md

# Task overrides with explicit filename
shark task create --epic=E09 --feature=F01 "Special Investigation" --filename="docs/investigations/auth-bug.md"
# Output: Created task T-E09-F01-002 at docs/investigations/auth-bug.md
```

---

## Library Interface

### PathBuilder Service

**Location**: `internal/utils/path_builder.go`

**Purpose**: Centralized service for resolving file paths based on custom folder path inheritance and precedence rules

#### NewPathBuilder

**Purpose**: Create a new PathBuilder instance

**Signature**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| projectRoot | string | Yes | N/A | Absolute path to project root directory |

**Returns**: `*PathBuilder` - Configured path builder instance

**Behavior**:
- Validates that projectRoot is an absolute path
- Stores projectRoot for use in path resolution
- Does not validate that projectRoot exists (assumed valid)

---

#### ResolveEpicPath

**Purpose**: Determine the file path for an epic.md file

**Signature**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| epicKey | string | Yes | N/A | Epic key (E##) |
| filename | *string | No | nil | Explicit file path from --filename flag (E07-F08) |
| customFolderPath | *string | No | nil | Custom base folder from --path flag |

**Returns**: `string` - Resolved absolute file path

**Raises**:
| Error | Condition |
|-------|-----------|
| ErrInvalidPath | Path validation fails (security check) |
| ErrPathOutsideProject | Resolved path escapes project root |

**Behavior**:
- **Precedence 1**: If `filename` is non-nil, return it as-is (exact override)
- **Precedence 2**: If `customFolderPath` is non-nil, return `<customFolderPath>/<epicKey>/epic.md`
- **Precedence 3**: Otherwise, return default `docs/plan/<epicKey>/epic.md`
- All paths are normalized and validated for security
- Returns absolute path (joined with projectRoot)

---

#### ResolveFeaturePath

**Purpose**: Determine the file path for a feature.md file

**Signature**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| epicKey | string | Yes | N/A | Parent epic key (E##) |
| featureKey | string | Yes | N/A | Feature key (E##-F##) |
| filename | *string | No | nil | Explicit file path from --filename flag |
| featureCustomPath | *string | No | nil | Feature's custom folder path from --path flag |
| epicCustomPath | *string | No | nil | Parent epic's custom folder path from database |

**Returns**: `string` - Resolved absolute file path

**Raises**: (same as ResolveEpicPath)

**Behavior**:
- **Precedence 1**: If `filename` is non-nil, return it (exact override)
- **Precedence 2**: If `featureCustomPath` is non-nil, return `<featureCustomPath>/<featureKey>/feature.md`
- **Precedence 3**: If `epicCustomPath` is non-nil, return `<epicCustomPath>/<epicKey>/<featureKey>/feature.md`
- **Precedence 4**: Otherwise, return default `docs/plan/<epicKey>/<featureKey>/feature.md`
- All paths validated for security

---

#### ResolveTaskPath

**Purpose**: Determine the file path for a task.md file

**Signature**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| epicKey | string | Yes | N/A | Parent epic key (E##) |
| featureKey | string | Yes | N/A | Parent feature key (F##) |
| taskKey | string | Yes | N/A | Task key (T-E##-F##-###) |
| filename | *string | No | nil | Explicit file path from --filename flag |
| featureCustomPath | *string | No | nil | Parent feature's custom folder path from database |
| epicCustomPath | *string | No | nil | Parent epic's custom folder path from database |

**Returns**: `string` - Resolved absolute file path

**Raises**: (same as ResolveEpicPath)

**Behavior**:
- **Precedence 1**: If `filename` is non-nil, return it (exact override)
- **Precedence 2**: If `featureCustomPath` is non-nil, return `<featureCustomPath>/<featureKey>/tasks/<taskKey>.md`
- **Precedence 3**: If `epicCustomPath` is non-nil, return `<epicCustomPath>/<epicKey>/<featureKey>/tasks/<taskKey>.md`
- **Precedence 4**: Otherwise, return default `docs/plan/<epicKey>/<featureKey>/tasks/<taskKey>.md`
- All paths validated for security

---

### Path Validation Functions

**Location**: `internal/utils/path_validation.go`

#### ValidateFolderPath

**Purpose**: Validate a custom folder path for security and correctness

**Signature**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| path | string | Yes | N/A | Relative folder path to validate |
| projectRoot | string | Yes | N/A | Absolute project root for boundary checks |

**Returns**: `(absPath string, relPath string, err error)`
- `absPath`: Absolute path (joined with projectRoot)
- `relPath`: Normalized relative path
- `err`: Validation error if path is invalid

**Raises**:
| Error | Condition |
|-------|-----------|
| ErrAbsolutePath | Path starts with `/` (absolute paths not allowed) |
| ErrPathTraversal | Path contains `..` (traversal not allowed) |
| ErrPathOutsideProject | Resolved path escapes project root |
| ErrEmptyPath | Path is empty or whitespace only |

**Behavior**:
- Normalize path (remove trailing slashes, clean `./` sequences)
- Check for absolute path prefix
- Check for `..` sequences
- Join with projectRoot to get absolute path
- Verify absolute path is within projectRoot
- Return both absolute and relative paths for caller convenience

---

## Repository Methods

### EpicRepository Extensions

#### GetCustomFolderPath

**Purpose**: Retrieve the custom folder path for an epic

**Signature**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| ctx | context.Context | Yes | N/A | Context for cancellation |
| epicKey | string | Yes | N/A | Epic key (E##) |

**Returns**: `(*string, error)` - Custom folder path (nil if not set) and error

**Raises**:
| Error | Condition |
|-------|-----------|
| ErrEpicNotFound | No epic with given key exists |
| database error | Query execution failure |

**Behavior**:
- Query: `SELECT custom_folder_path FROM epics WHERE key = ?`
- Return nil if custom_folder_path is NULL in database
- Return descriptive error if epic not found
- Wrap database errors with context

---

### FeatureRepository Extensions

#### GetCustomFolderPath

**Purpose**: Retrieve the custom folder path for a feature

**Signature**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| ctx | context.Context | Yes | N/A | Context for cancellation |
| featureKey | string | Yes | N/A | Feature key (E##-F##) |

**Returns**: `(*string, error)` - Custom folder path (nil if not set) and error

**Raises**:
| Error | Condition |
|-------|-----------|
| ErrFeatureNotFound | No feature with given key exists |
| database error | Query execution failure |

**Behavior**:
- Query: `SELECT custom_folder_path FROM features WHERE key = ?`
- Return nil if custom_folder_path is NULL in database
- Return descriptive error if feature not found
- Wrap database errors with context

---

## Error Handling

### Error Codes

| Code | Meaning | Response |
|------|---------|----------|
| ErrInvalidPath | Path validation failed (security or format) | CLI exits with code 1, displays specific validation error |
| ErrAbsolutePath | Path is absolute (not relative) | "Error: path must be relative to project root, got absolute path: ..." |
| ErrPathTraversal | Path contains `..` | "Error: invalid path: contains '..' (path traversal not allowed)" |
| ErrPathOutsideProject | Path resolves outside project root | "Error: path validation failed: resolves outside project root" |
| ErrEmptyPath | Path is empty or whitespace | "Error: invalid path: resolved to empty or invalid path" |
| ErrEpicNotFound | Epic does not exist | "Error: Epic E## not found" |
| ErrFeatureNotFound | Feature does not exist | "Error: Feature E##-F## not found" |
| ErrDatabaseError | Database operation failed | "Error: Database error. Run with --verbose for details." |

### Error Response Format

**CLI Output** (human-readable):
```
Error: path must be relative to project root, got absolute path: /absolute/path
```

**CLI Output** (--json):
```json
{
  "error": "path validation failed",
  "message": "path must be relative to project root, got absolute path: /absolute/path",
  "code": "invalid_path"
}
```

**Verbose Mode** (`--verbose` flag):
- Includes full stack trace
- Includes database query details
- Includes file system paths for debugging

---

## Integration Points

### Database Layer
- Repository methods query `custom_folder_path` column
- NULL values are handled explicitly (pointer semantics)
- JOINs between features and epics for path inheritance

### File System Layer
- PathBuilder generates absolute paths for file creation
- File creation logic validates directory existence
- Parent directories created automatically if missing

### Sync Layer
- Discovery scans both default and custom path locations
- Frontmatter parser reads `custom_folder_path` field
- Conflict resolution handles file vs. database custom path differences

---

## Summary

This backend design introduces minimal CLI changes (two new flags) while leveraging a centralized PathBuilder service for complex path resolution logic. The design prioritizes:

1. **Simplicity**: Single service encapsulates all path resolution
2. **Security**: All paths validated before use
3. **Testability**: PathBuilder is a pure function (easily tested)
4. **Backward Compatibility**: All new fields and flags are optional

The implementation extends existing commands and repositories with minimal disruption, following established patterns from the codebase research.

---

**End of Backend Design Document**
