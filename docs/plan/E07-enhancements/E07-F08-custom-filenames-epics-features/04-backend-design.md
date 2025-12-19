# Backend Design: Custom Filenames for Epics & Features

**Epic**: E07
**Feature**: E07-F08
**Date**: 2025-12-19
**Author**: backend-architect

## Interface Overview

This feature extends the `shark epic create` and `shark feature create` CLI commands with two new flags: `--filename` (custom file path) and `--force` (force reassignment). The interface design follows the established pattern from `shark task create` to ensure consistent CLI behavior across all entity types.

**Interface Type**: CLI (Command-Line Interface)

## Codebase Analysis

### Existing Related Interfaces

- `shark task create` at `internal/cli/commands/task.go` - Already supports `--filename` and `--force` flags (E07-F05)
- `shark epic create` at `internal/cli/commands/epic.go` - Currently does NOT support custom filenames (to be extended)
- `shark feature create` at `internal/cli/commands/feature.go` - Currently does NOT support custom filenames (to be extended)
- `ValidateCustomFilename` at `internal/taskcreation/creator.go` - Reusable filename validation function

### Naming Patterns Found

- **Command Structure**: `shark {entity} {action} [flags] [arguments]`
- **Flag Naming**: Kebab-case with double dashes (e.g., `--filename`, `--epic`, `--description`)
- **Variable Naming**: camelCase for Go variables (e.g., `epicCreateFilename`, `featureCreateForce`)
- **Function Naming**: PascalCase for exported functions, camelCase for private (e.g., `ValidateCustomFilename`, `runEpicCreate`)

### Extension vs. New Code Decision

**Decision**: Extend existing commands

**Rationale**:
- `shark epic create` and `shark feature create` commands already exist with comprehensive functionality
- Adding two flags to existing commands maintains backward compatibility (flags are optional)
- No new commands are created; existing command handlers are enhanced
- Validation logic is reused from `internal/taskcreation/creator.go` without modification

---

## CLI Commands

### shark epic create

**Purpose**: Create a new epic with optional custom file location

**Usage**: `shark epic create [flags] <title>`

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| title | Yes | Epic title (double-quoted if contains spaces) |

**Options**:
| Option | Short | Type | Default | Description |
|--------|-------|------|---------|-------------|
| --filename | | string | (empty) | Custom file path relative to project root. Must end in `.md`. Example: `docs/roadmap/2025.md` |
| --force | | bool | false | Force reassignment if file path already claimed by another epic or feature |
| --description | | string | (empty) | Optional epic description |
| --priority | | string | medium | Priority: high, medium, or low |
| --business-value | | string | (empty) | Business value: high, medium, or low |

**Output**:

*Without --json flag* (human-readable):
```
Created epic E09 'Platform Roadmap'
File: docs/roadmap/2025.md
Key: E09
```

*With --json flag* (machine-readable):
```json
{
  "epic": {
    "id": 9,
    "key": "E09",
    "title": "Platform Roadmap",
    "description": null,
    "status": "draft",
    "priority": "medium",
    "business_value": null,
    "file_path": "docs/roadmap/2025.md",
    "created_at": "2025-12-19T10:30:00Z",
    "updated_at": "2025-12-19T10:30:00Z"
  }
}
```

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success - epic created |
| 1 | Validation error (invalid filename, collision without --force, missing required arguments) |
| 2 | Database error (transaction failed, constraint violation) |

**Examples**:
```bash
# Default location (backward compatible)
shark epic create "User Authentication System"
# Output: Created epic E04 at docs/plan/E04/epic.md

# Custom location
shark epic create --filename="docs/roadmap/2025-platform.md" "Platform Roadmap"
# Output: Created epic E09 at docs/roadmap/2025-platform.md

# Custom location with collision (error)
shark epic create --filename="docs/shared/auth.md" "SSO"
# Error: file 'docs/shared/auth.md' is already claimed by epic E04 ('Authentication'). Use --force to reassign.

# Force reassignment
shark epic create --filename="docs/shared/auth.md" --force "SSO Integration"
# Output: Created epic E10 at docs/shared/auth.md (reassigned from epic E04)

# With multiple flags
shark epic create --filename="docs/roadmap/q1.md" --priority=high --business-value=high "Q1 Objectives"
```

---

### shark feature create

**Purpose**: Create a new feature within an epic with optional custom file location

**Usage**: `shark feature create --epic=<epic-key> [flags] <title>`

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| title | Yes | Feature title (double-quoted if contains spaces) |

**Options**:
| Option | Short | Type | Default | Description |
|--------|-------|------|---------|-------------|
| --epic | | string | (required) | Parent epic key (e.g., E04) |
| --filename | | string | (empty) | Custom file path relative to project root. Must end in `.md`. Example: `docs/specs/auth-service.md` |
| --force | | bool | false | Force reassignment if file path already claimed by another feature or epic |
| --description | | string | (empty) | Optional feature description |
| --execution-order | | int | (auto) | Optional execution order within the epic |

**Output**:

*Without --json flag* (human-readable):
```
Created feature E04-F15 'OAuth Integration'
File: docs/specs/auth-oauth.md
Key: E04-F15
```

*With --json flag* (machine-readable):
```json
{
  "feature": {
    "id": 45,
    "epic_id": 4,
    "key": "E04-F15",
    "title": "OAuth Integration",
    "description": null,
    "status": "draft",
    "progress_pct": 0.0,
    "execution_order": null,
    "file_path": "docs/specs/auth-oauth.md",
    "created_at": "2025-12-19T10:35:00Z",
    "updated_at": "2025-12-19T10:35:00Z"
  }
}
```

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success - feature created |
| 1 | Validation error (invalid filename, collision without --force, missing --epic, epic not found) |
| 2 | Database error (transaction failed, constraint violation) |

**Examples**:
```bash
# Default location (backward compatible)
shark feature create --epic=E04 "OAuth Login Integration"
# Output: Created feature E04-F15 at docs/plan/E04/E04-F15/feature.md

# Custom location
shark feature create --epic=E04 --filename="docs/specs/auth-oauth.md" "OAuth Integration"
# Output: Created feature E04-F15 at docs/specs/auth-oauth.md

# Custom location with collision (error)
shark feature create --epic=E05 --filename="docs/shared/auth.md" "SSO"
# Error: file 'docs/shared/auth.md' is already claimed by epic E04 ('Authentication'). Use --force to reassign.

# Force reassignment from feature to feature
shark feature create --epic=E05 --filename="docs/specs/auth.md" --force "SSO Provider"
# Output: Created feature E05-F03 at docs/specs/auth.md (reassigned from feature E04-F12)

# With multiple flags
shark feature create --epic=E07 --filename="docs/enhancements/caching.md" --execution-order=2 "Response Caching"
```

---

## Validation Logic

### ValidateCustomFilename Function

**Purpose**: Validate custom file paths against security and project boundary constraints (reused from task implementation)

**Location**: `internal/taskcreation/creator.go` (existing, exported for reuse)

**Signature**:
```
ValidateCustomFilename(filename string, projectRoot string) (absPath string, relPath string, error)
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| filename | string | User-provided relative file path (from `--filename` flag) |
| projectRoot | string | Absolute path to project root directory |

**Returns**:
| Return Value | Type | Description |
|--------------|------|-------------|
| absPath | string | Absolute file path (for file operations) |
| relPath | string | Relative file path from project root (for database storage) |
| error | error | Validation error if path is invalid; nil on success |

**Validation Rules**:

| Rule | Check | Error Message |
|------|-------|---------------|
| No absolute paths | `filepath.IsAbs(filename)` | `filename must be relative to project root, got absolute path: {filename}` |
| Extension required | `filepath.Ext(filename) != ".md"` | `invalid file extension: {ext} (must be .md)` |
| No path traversal | `strings.Contains(filename, "..")` | `invalid path: contains '..' (path traversal not allowed)` |
| Within project | Resolved path outside project root | `path validation failed: path resolves outside project root` |
| Non-empty filename | After normalization, path is empty | `invalid filename: resolved to empty or invalid path` |

**Behavior**:
- Normalizes path using `filepath.Clean`
- Converts to absolute path for file operations
- Computes relative path for database storage
- Ensures path stays within project boundaries using `filepath.Rel` check
- Does NOT check if file exists (file may or may not exist; both are valid)

---

## Collision Detection

### Collision Check Logic

**When**: Before inserting epic or feature into database with a custom `file_path`

**Query Pattern**:

**For Epic Creation**:
```
Query: SELECT * FROM epics WHERE file_path = ?
Parameters: [validated_relative_path]
```

**For Feature Creation**:
```
Query: SELECT * FROM features WHERE file_path = ?
Parameters: [validated_relative_path]
```

**Behavior**:

| Scenario | --force Flag | Action |
|----------|--------------|--------|
| No collision (file_path unclaimed) | N/A | Proceed with insert; set `file_path` to validated path |
| Collision detected | false (default) | Reject creation; return error with details of conflicting entity |
| Collision detected | true | Clear `file_path` on conflicting entity (set to NULL); proceed with insert |

**Error Message Format** (collision without --force):
```
Error: file '{file_path}' is already claimed by {entity_type} {entity_key} ('{entity_title}'). Use --force to reassign.
```

**Examples**:
- `Error: file 'docs/roadmap/2025.md' is already claimed by epic E04 ('Platform Roadmap'). Use --force to reassign.`
- `Error: file 'docs/specs/auth.md' is already claimed by feature E04-F12 ('OAuth Service'). Use --force to reassign.`

---

## Force Reassignment

### Reassignment Flow

**Trigger**: `--force` flag is set AND collision is detected

**Steps**:
1. Detect collision via query (`GetByFilePath`)
2. Identify conflicting entity (epic or feature)
3. Update conflicting entity: `UPDATE {table} SET file_path = NULL WHERE key = ?`
4. Proceed with new entity creation with desired `file_path`
5. Log reassignment in success message (if verbose mode enabled)

**Success Message Format** (with reassignment):
```
Created {entity_type} {new_key} '{new_title}' at {file_path} (reassigned from {old_entity_type} {old_key})
```

**Examples**:
- `Created epic E10 'SSO Integration' at docs/shared/auth.md (reassigned from epic E04)`
- `Created feature E05-F03 'Caching' at docs/specs/cache.md (reassigned from feature E04-F08)`

**Transaction Safety**:
- Both UPDATE and INSERT occur within a single database transaction
- If INSERT fails, UPDATE is rolled back
- Ensures atomicity: either both succeed or both fail

---

## Repository Methods

### Epic Repository Extensions

**File**: `internal/repository/epic_repository.go`

#### GetByFilePath

**Purpose**: Retrieve an epic by its file path for collision detection

**Signature**:
```
GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error)
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| ctx | context.Context | Request context for cancellation |
| filePath | string | Relative file path to search |

**Returns**:
| Return Value | Type | Description |
|--------------|------|-------------|
| epic | *models.Epic | Epic claiming the file path, or nil if not found |
| error | error | Database error; nil on success (including "not found" case) |

**Behavior**:
- Query: `SELECT * FROM epics WHERE file_path = ?`
- Returns `nil, nil` if no epic found (not an error condition)
- Returns `epic, nil` if epic found
- Returns `nil, error` on database failures

---

#### UpdateFilePath

**Purpose**: Update or clear the file path for an epic

**Signature**:
```
UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| ctx | context.Context | Request context |
| epicKey | string | Epic key (e.g., E04) |
| newFilePath | *string | New file path (nil to clear/set to NULL) |

**Returns**:
| Return Value | Type | Description |
|--------------|------|-------------|
| error | error | Database error; nil on success |

**Behavior**:
- Query: `UPDATE epics SET file_path = ? WHERE key = ?`
- Pass `nil` to set `file_path` to NULL (clear custom path)
- Returns error if epic not found (no rows updated)

---

### Feature Repository Extensions

**File**: `internal/repository/feature_repository.go`

#### GetByFilePath

**Purpose**: Retrieve a feature by its file path for collision detection

**Signature**:
```
GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error)
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| ctx | context.Context | Request context |
| filePath | string | Relative file path to search |

**Returns**:
| Return Value | Type | Description |
|--------------|------|-------------|
| feature | *models.Feature | Feature claiming the file path, or nil if not found |
| error | error | Database error; nil on success |

**Behavior**:
- Query: `SELECT * FROM features WHERE file_path = ?`
- Returns `nil, nil` if no feature found
- Returns `feature, nil` if feature found
- Returns `nil, error` on database failures

---

#### UpdateFilePath

**Purpose**: Update or clear the file path for a feature

**Signature**:
```
UpdateFilePath(ctx context.Context, featureKey string, newFilePath *string) error
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| ctx | context.Context | Request context |
| featureKey | string | Feature key (e.g., E04-F12) |
| newFilePath | *string | New file path (nil to clear/set to NULL) |

**Returns**:
| Return Value | Type | Description |
|--------------|------|-------------|
| error | error | Database error; nil on success |

**Behavior**:
- Query: `UPDATE features SET file_path = ? WHERE key = ?`
- Pass `nil` to set `file_path` to NULL
- Returns error if feature not found

---

## Error Handling

### Error Codes

| Code | Meaning | Response |
|------|---------|----------|
| ERR_INVALID_FILENAME | Filename validation failed | Display validation error message; suggest correct format |
| ERR_FILE_COLLISION | File already claimed by another entity | Show which entity owns the file; suggest `--force` |
| ERR_EPIC_NOT_FOUND | Parent epic not found (for feature creation) | Display "Epic {key} not found" error |
| ERR_DATABASE_FAILURE | Database transaction or query failed | Display generic database error; suggest checking database file |

### Error Response Format

All CLI errors follow this format:

**Human-readable** (default):
```
Error: {error message}
```

**JSON** (with `--json` flag):
```json
{
  "error": {
    "code": "ERR_FILE_COLLISION",
    "message": "file 'docs/roadmap/2025.md' is already claimed by epic E04 ('Platform Roadmap')",
    "suggestion": "Use --force to reassign"
  }
}
```

## Backward Compatibility

### Default Behavior (No --filename)

When `--filename` is NOT provided:

**Epic Creation**:
- Database: `file_path` set to NULL
- File created at: `docs/plan/{epic-key}/epic.md`
- Example: Epic E09 → `docs/plan/E09/epic.md`

**Feature Creation**:
- Database: `file_path` set to NULL
- File created at: `docs/plan/{epic-key}/{feature-key}/feature.md`
- Example: Feature E04-F15 → `docs/plan/E04/E04-F15/feature.md`

**Guarantee**: Existing scripts and workflows that call `shark epic create` or `shark feature create` without `--filename` continue to work identically to before this feature.

## Versioning

- **Strategy**: CLI versioning follows semantic versioning for the `shark` binary
- **Current version**: v1.0.0 (approximate)
- **Deprecation policy**: New flags are additive; no deprecation of existing flags or behavior in this feature
