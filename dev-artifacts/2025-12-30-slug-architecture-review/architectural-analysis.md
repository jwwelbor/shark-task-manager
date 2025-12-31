# Slug Generation and Path Resolution Architecture Review

**Date**: 2025-12-30
**Scope**: Epic, Feature, and Task file path management
**Focus**: Slug generation, database schema, and path resolution consistency

---

## Executive Summary

### Critical Issues Identified

1. **Inconsistent Frontmatter/Metadata Storage**: Tasks use YAML frontmatter with `task_key`, while epics/features embed keys in markdown body using bold headers (`**Epic Key**: E05-...`)
2. **File-Based Metadata Retrieval**: Discovery system reads files to extract slugs/titles instead of using database as source of truth
3. **No Slug Column in Database**: Slugs are computed on-the-fly from titles, not stored or validated
4. **Key Format Confusion**: Users must know full slugged keys (`E05-task-mgmt-cli-capabilities`) instead of just numeric keys (`E05`)
5. **Path Resolution Complexity**: Multiple code paths for determining file locations, some reading files unnecessarily

### Impact

- **Data Integrity**: File content becomes source of truth for metadata, violating database-first architecture
- **Performance**: Unnecessary file I/O during path resolution and discovery
- **User Experience**: Complex key requirements (`E05-task-mgmt-cli-capabilities` vs `E05`)
- **Maintainability**: Inconsistent patterns across epics/features/tasks create technical debt
- **Sync Complexity**: Discovery system must parse markdown to extract metadata

---

## Current State Analysis

### 1. Database Schema

#### Epics Table
```sql
CREATE TABLE epics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,              -- E.g., "E05"
    title TEXT NOT NULL,                   -- E.g., "Task Management CLI - Extended Capabilities"
    file_path TEXT,                        -- E.g., "docs/plan/E05-task-mgmt-cli-capabilities/epic.md"
    custom_folder_path TEXT,               -- Optional custom base path
    -- ... other fields ...
);
```

**Missing**: No `slug` column. Slug is embedded in `file_path` but not stored separately.

#### Features Table
```sql
CREATE TABLE features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,              -- E.g., "E06-F04"
    title TEXT NOT NULL,                   -- E.g., "Incremental Sync Engine"
    file_path TEXT,                        -- E.g., "docs/plan/E06-intelligent-scanning/E06-F04-incremental-sync-engine/prd.md"
    custom_folder_path TEXT,               -- Optional custom base path
    -- ... other fields ...
);
```

**Missing**: No `slug` column. Slug is embedded in `file_path` but not stored separately.

#### Tasks Table
```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feature_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,              -- E.g., "T-E04-F01-001"
    title TEXT NOT NULL,                   -- E.g., "Some Task Description"
    file_path TEXT,                        -- E.g., "docs/plan/E04-epic/E04-F01-feature/tasks/T-E04-F01-001-some-task-description.md"
    -- ... other fields ...
);
```

**Missing**: No `slug` column. Recent commit (960a807) added slug generation for filenames, but slugs are NOT stored in database.

### 2. File Format Inconsistencies

#### Epic File Format (`epic.md`)
```markdown
# Epic: Task Management CLI - Extended Capabilities

**Epic Key**: E05-task-mgmt-cli-capabilities
**Created**: 2025-12-14
**Status**: Draft
...
```

**Issues**:
- Key embedded in markdown body, not frontmatter
- Key includes slug (`E05-task-mgmt-cli-capabilities`), not just numeric part (`E05`)
- Requires markdown parsing to extract metadata

#### Feature File Format (`prd.md` or `feature.md`)
```markdown
# Feature: Incremental Sync Engine

## Epic

- [Epic PRD](/path/to/epic.md)
- [Epic Requirements](/path/to/requirements.md)

## Goal
...
```

**Issues**:
- Feature key NOT in file at all (discovered from folder name)
- No frontmatter for structured metadata
- Completely different pattern from tasks

#### Task File Format (`T-*.md`)
```yaml
---
task_key: T-E06-F04-001
---

# Task Title

Content...
```

**Issues**:
- Uses YAML frontmatter (good!)
- Consistent structure
- But pattern not replicated for epics/features

### 3. Path Resolution Logic

#### Current Flow for Task Creation (`internal/taskcreation/creator.go`)

```go
// Lines 144-175: Task path resolution
if input.Filename != "" {
    // Custom filename - validate it
    absPath, relPath, err := ValidateCustomFilename(input.Filename, c.projectRoot)
    filePath = relPath
    fullFilePath = absPath
} else {
    // Default: derive task path from feature's actual location
    feature, err := c.featureRepo.GetByKey(ctx, validated.NormalizedFeatureKey)

    if feature.FilePath != nil && *feature.FilePath != "" {
        // Feature has a file path - derive task path from it
        featureDir := filepath.Dir(*feature.FilePath)
        relPath := filepath.Join(featureDir, "tasks", key+".md")
        fullFilePath = filepath.Join(c.projectRoot, relPath)
        filePath = relPath
    } else {
        // Fallback: feature has no file_path, use PathBuilder to reconstruct from keys
        epic, err := c.epicRepo.GetByID(ctx, feature.EpicID)

        pb := utils.NewPathBuilder(c.projectRoot)
        fullFilePath, err = pb.ResolveTaskPath(epic.Key, validated.NormalizedFeatureKey, key, input.Title, nil, feature.CustomFolderPath, epic.CustomFolderPath)
    }
}
```

**Analysis**:
- ✅ Good: Database-first approach when `feature.FilePath` exists
- ❌ Bad: Fallback to PathBuilder requires epic/feature lookup
- ❌ Bad: PathBuilder computes paths from keys/titles/custom_folder_path instead of just reading database

#### PathBuilder Logic (`internal/utils/path_builder.go`)

```go
// Lines 88-130: ResolveTaskPath
func (pb *PathBuilder) ResolveTaskPath(epicKey, featureKey, taskKey, taskTitle string, filename *string, featureCustomPath, epicCustomPath *string) (string, error) {
    // Precedence 1: Explicit filename override
    if filename != nil && *filename != "" {
        return *filename, nil
    }

    // Build the base directory
    var baseDir string

    // Precedence 2: Feature's custom path
    if featureCustomPath != nil && *featureCustomPath != "" {
        _, relPath, err := ValidateFolderPath(*featureCustomPath, pb.projectRoot)
        baseDir = filepath.Join(pb.projectRoot, relPath, featureKey, "tasks")
    } else if epicCustomPath != nil && *epicCustomPath != "" {
        // Precedence 3: Inherit from epic
        _, relPath, err := ValidateFolderPath(*epicCustomPath, pb.projectRoot)
        baseDir = filepath.Join(pb.projectRoot, relPath, epicKey, featureKey, "tasks")
    } else {
        // Precedence 4: Default path
        baseDir = filepath.Join(pb.projectRoot, "docs", "plan", epicKey, featureKey, "tasks")
    }

    // Generate filename with slug if title provided
    filename_str := slug.GenerateFilename(taskKey, taskTitle)  // ← COMPUTES SLUG HERE

    return filepath.Join(baseDir, filename_str), nil
}
```

**Analysis**:
- ❌ Bad: PathBuilder uses keys (not slugs) to construct directory paths
- ❌ Bad: Slug generation happens at path construction time, not during creation
- ❌ Bad: No validation that computed slug matches stored slug (because there is no stored slug!)

### 4. Discovery System (`internal/sync/discovery.go`)

```go
// Lines 285-298: convertFolderEpics
func convertFolderEpics(folderEpics []discovery.FolderEpic) []discovery.DiscoveredEpic {
    result := make([]discovery.DiscoveredEpic, len(folderEpics))
    for i, epic := range folderEpics {
        result[i] = discovery.DiscoveredEpic{
            Key:              epic.Key,
            Title:            epic.Slug,  // ← SLUG FROM FOLDER NAME BECOMES TITLE!
            FilePath:         epic.EpicMdPath,
            CustomFolderPath: epic.CustomFolderPath,
            Source:           discovery.SourceFolder,
        }
    }
    return result
}
```

**Analysis**:
- ❌ Critical: Folder slug becomes database title
- ❌ Critical: System reads filesystem to determine slugs, then updates database
- ❌ Critical: File system is treated as source of truth, not database

### 5. Slug Generation (`internal/slug/slug.go`)

```go
// Lines 45-80: Generate
func Generate(title string) string {
    // Step 1: Normalize unicode characters
    // Step 2: Convert to lowercase
    // Step 3: Replace spaces, underscores, periods with hyphens
    // Step 4: Remove non-alphanumeric characters except hyphens
    // Step 5: Collapse multiple hyphens
    // Step 6: Remove leading/trailing hyphens
    // Step 7: Truncate to maxSlugLength (100 chars)
    return slug
}

// Lines 82-101: GenerateFilename
func GenerateFilename(taskKey, title string) string {
    slug := Generate(title)
    if slug == "" {
        return taskKey + ".md"
    }
    return taskKey + "-" + slug + ".md"
}
```

**Analysis**:
- ✅ Good: Deterministic slug generation from title
- ✅ Good: Unicode handling, special character removal
- ❌ Bad: Slug NOT stored anywhere (computed on-the-fly)
- ❌ Bad: If title changes, slug changes → file path changes → potential file orphaning

---

## Root Cause Analysis

### Primary Problem: Database Is Not Single Source of Truth

The system violates the fundamental principle that **database should be the single source of truth for all metadata**.

#### Evidence:

1. **Discovery reads files to determine slugs**:
   - Folder names are parsed to extract slugs
   - Slugs from folders become database titles
   - File content is parsed to extract epic/feature keys

2. **PathBuilder reconstructs paths instead of reading from database**:
   - Paths are computed from keys + titles + custom_folder_path
   - No validation that computed path matches database `file_path`
   - Slug generation happens during path construction, not during entity creation

3. **Inconsistent metadata storage**:
   - Tasks use frontmatter (database-friendly)
   - Epics/features use markdown body (requires parsing)
   - Keys in files include slugs, but database only stores numeric keys

### Secondary Problem: No Slug Column in Database

Slugs are derived data (computed from titles) but not persisted. This causes:

1. **Non-deterministic file paths**: If slug generation logic changes, all file paths change
2. **Title change cascades**: Changing title changes slug → file path → potential orphaning
3. **Performance overhead**: Must recompute slug every time path is needed
4. **No slug validation**: Can't verify that file path matches expected slug

### Tertiary Problem: Key Format Inconsistency

Users must know full slugged keys to reference entities:
- CLI accepts: `E05-task-mgmt-cli-capabilities` (slugged)
- Database stores: `E05` (numeric only)
- Files contain: `E05-task-mgmt-cli-capabilities` (slugged)

This creates confusion:
- Is `E05` or `E05-task-mgmt-cli-capabilities` the "real" key?
- Must users remember slugs to reference epics?
- What happens if title changes (slug changes)?

---

## Desired End State

### Architectural Principles

1. **Database as Single Source of Truth**: All metadata (key, title, slug) stored in database
2. **Keys Are Sufficient Identifiers**: `E05` alone should work; slugs are for human readability
3. **Slugs Are Deterministic and Stored**: Generated once at creation, stored in database
4. **File Paths Are Derived from Database**: Never read files to determine paths
5. **Consistent Patterns Across All Entities**: Epics, features, tasks use same approach

### Proposed Database Schema Changes

#### Add `slug` Column to All Entity Tables

```sql
-- Migration for epics table
ALTER TABLE epics ADD COLUMN slug TEXT;
CREATE INDEX IF NOT EXISTS idx_epics_slug ON epics(slug);

-- Migration for features table
ALTER TABLE features ADD COLUMN slug TEXT;
CREATE INDEX IF NOT EXISTS idx_features_slug ON features(slug);

-- Migration for tasks table (optional - tasks already include slug in filename)
-- Tasks may not need slug column since title is always in filename
-- But for consistency, we could add it:
ALTER TABLE tasks ADD COLUMN slug TEXT;
CREATE INDEX IF NOT EXISTS idx_tasks_slug ON tasks(slug);
```

#### Populate Existing Slugs

```sql
-- Backfill epics: Extract slug from file_path
UPDATE epics
SET slug = (
    CASE
        WHEN file_path IS NOT NULL
        THEN substr(file_path,
                    instr(file_path, key || '-') + length(key) + 1,
                    instr(file_path, '/epic.md') - instr(file_path, key || '-') - length(key) - 1)
        ELSE NULL
    END
)
WHERE file_path IS NOT NULL AND slug IS NULL;

-- Backfill features: Extract slug from file_path
UPDATE features
SET slug = (
    CASE
        WHEN file_path IS NOT NULL
        THEN substr(file_path,
                    instr(file_path, key || '-') + length(key) + 1,
                    instr(file_path, '/prd.md') - instr(file_path, key || '-') - length(key) - 1)
        ELSE NULL
    END
)
WHERE file_path IS NOT NULL AND slug IS NULL;

-- For tasks: Extract from filename
UPDATE tasks
SET slug = (
    CASE
        WHEN file_path IS NOT NULL AND instr(file_path, key || '-') > 0
        THEN substr(file_path,
                    instr(file_path, key || '-') + length(key) + 1,
                    instr(file_path, '.md') - instr(file_path, key || '-') - length(key) - 1)
        ELSE NULL
    END
)
WHERE file_path IS NOT NULL AND slug IS NULL;
```

### Proposed File Format Standardization

#### Option A: YAML Frontmatter for All (Recommended)

**Epic File Format**:
```yaml
---
epic_key: E05
slug: task-mgmt-cli-capabilities
title: Task Management CLI - Extended Capabilities
status: active
priority: high
created_at: 2025-12-14
---

# Epic: Task Management CLI - Extended Capabilities

## Goal

### Problem
...
```

**Feature File Format**:
```yaml
---
feature_key: E06-F04
epic_key: E06
slug: incremental-sync-engine
title: Incremental Sync Engine
status: active
created_at: 2025-12-18
---

# Feature: Incremental Sync Engine

## Epic

- [E06 - Intelligent Scanning](/docs/plan/E06-intelligent-scanning/epic.md)

## Goal
...
```

**Task File Format** (no change):
```yaml
---
task_key: T-E06-F04-001
feature_key: E06-F04
epic_key: E06
slug: conflict-detection-resolution
title: Conflict Detection and Resolution System
status: completed
priority: 5
created_at: 2025-12-18
completed_at: 2025-12-18
---

# Task: Conflict Detection and Resolution System

## Overview
...
```

**Benefits**:
- Consistent parsing across all entity types
- Machine-readable metadata
- Easy to extract with YAML parser
- Supports sync without markdown parsing

#### Option B: Keep Current Format, Add Slug to Epic/Feature Body

**Epic File Format**:
```markdown
# Epic: Task Management CLI - Extended Capabilities

**Epic Key**: E05
**Slug**: task-mgmt-cli-capabilities
**Created**: 2025-12-14
**Status**: Draft
...
```

**Benefits**:
- Minimal change to existing files
- Human-readable
- Keeps markdown-first approach

**Drawbacks**:
- Still requires markdown parsing
- Harder to validate
- Inconsistent with tasks

### Proposed Path Resolution Strategy

#### Unified Path Resolution

```go
// PathResolver uses database as single source of truth
type PathResolver struct {
    db *repository.DB
    projectRoot string
}

// ResolveEpicPath resolves epic file path from database only
func (pr *PathResolver) ResolveEpicPath(ctx context.Context, epicKey string) (string, error) {
    epic, err := pr.db.EpicRepo.GetByKey(ctx, epicKey)
    if err != nil {
        return "", err
    }

    // Precedence 1: Explicit file_path in database
    if epic.FilePath != nil && *epic.FilePath != "" {
        return filepath.Join(pr.projectRoot, *epic.FilePath), nil
    }

    // Precedence 2: Compute from database fields
    if epic.Slug == nil || *epic.Slug == "" {
        return "", fmt.Errorf("epic %s has no slug or file_path", epicKey)
    }

    // Compute default path
    var baseDir string
    if epic.CustomFolderPath != nil && *epic.CustomFolderPath != "" {
        baseDir = filepath.Join(pr.projectRoot, *epic.CustomFolderPath)
    } else {
        baseDir = filepath.Join(pr.projectRoot, "docs", "plan")
    }

    folderName := epic.Key + "-" + *epic.Slug
    return filepath.Join(baseDir, folderName, "epic.md"), nil
}

// Similar for features and tasks
```

**Benefits**:
- Database-first (never read files to determine paths)
- Slug stored in database
- Path computation is deterministic
- Handles custom_folder_path correctly

### Proposed Creation Flow

#### Epic Creation

```go
func CreateEpic(ctx context.Context, input CreateEpicInput) (*Epic, error) {
    // 1. Generate key (E01, E02, etc.)
    key := generateNextEpicKey(ctx)

    // 2. Generate slug from title (ONE TIME ONLY)
    slug := slug.Generate(input.Title)

    // 3. Compute file path from key + slug + custom_folder_path
    pathResolver := NewPathResolver(db, projectRoot)
    filePath := pathResolver.ComputeEpicPath(key, slug, input.CustomFolderPath)

    // 4. Create database record (FIRST)
    epic := &Epic{
        Key:              key,
        Title:            input.Title,
        Slug:             &slug,  // ← STORE SLUG IN DATABASE
        FilePath:         &filePath,
        CustomFolderPath: input.CustomFolderPath,
        // ... other fields ...
    }
    err := epicRepo.Create(ctx, epic)

    // 5. Create file (SECOND, using database data)
    err = writeEpicFile(filePath, epic)

    return epic, nil
}
```

**Key Changes**:
- Slug generated ONCE at creation
- Slug stored in database
- File path computed from database fields
- Database record created before file

### Proposed Key Lookup Strategy

#### Support Both Numeric and Slugged Keys

```go
// GetByKey supports both "E05" and "E05-task-mgmt-cli-capabilities"
func (r *EpicRepository) GetByKey(ctx context.Context, keyOrSlug string) (*Epic, error) {
    // Try exact key match first
    epic, err := r.getByExactKey(ctx, keyOrSlug)
    if err == nil {
        return epic, nil
    }

    // If not found and input contains hyphen, try extracting numeric key
    if strings.Contains(keyOrSlug, "-") {
        numericKey := extractNumericKey(keyOrSlug) // "E05-some-slug" → "E05"
        epic, err := r.getByExactKey(ctx, numericKey)
        if err == nil {
            return epic, nil
        }
    }

    return nil, sql.ErrNoRows
}

func extractNumericKey(keyOrSlug string) string {
    // "E05-some-slug" → "E05"
    // "E04-F02-feature-name" → "E04-F02"
    // "T-E04-F02-001-task-name" → "T-E04-F02-001"
    parts := strings.Split(keyOrSlug, "-")

    // For epics: E## (first part)
    if len(parts) > 0 && strings.HasPrefix(parts[0], "E") {
        return parts[0]
    }

    // For features: E##-F## (first two parts)
    if len(parts) >= 2 && strings.HasPrefix(parts[1], "F") {
        return parts[0] + "-" + parts[1]
    }

    // For tasks: T-E##-F##-### (first four parts)
    if len(parts) >= 4 && parts[0] == "T" {
        return parts[0] + "-" + parts[1] + "-" + parts[2] + "-" + parts[3]
    }

    return keyOrSlug
}
```

**Benefits**:
- Backward compatible with existing commands
- Users can use `E05` or `E05-task-mgmt-cli-capabilities`
- Simple extraction logic

---

## Migration Strategy

### Phase 1: Add Slug Column (Backward Compatible)

1. **Database Migration**:
   ```sql
   ALTER TABLE epics ADD COLUMN slug TEXT;
   ALTER TABLE features ADD COLUMN slug TEXT;
   ALTER TABLE tasks ADD COLUMN slug TEXT;
   ```

2. **Backfill Existing Slugs**:
   - Extract from `file_path` for existing records
   - Leave NULL for records without `file_path`

3. **Update Creation Logic**:
   - Generate slug at creation time
   - Store in database immediately
   - Continue writing files with slugged names

4. **Test**:
   - Create new epics/features/tasks
   - Verify slug stored in database
   - Verify file path matches slug

### Phase 2: Standardize File Formats (Breaking Change)

1. **Decision**: Choose Option A (YAML frontmatter) or Option B (markdown body with slug)

2. **Migration Script**:
   - Read existing epic/feature files
   - Convert to new format
   - Preserve all metadata
   - Update `file_path` if format changes filenames

3. **Update Sync/Discovery**:
   - Parse frontmatter instead of markdown body
   - Extract slug from frontmatter
   - Validate against database slug

4. **Test**:
   - Run sync on migrated files
   - Verify metadata preserved
   - Verify no duplicate records created

### Phase 3: Refactor Path Resolution (Breaking Change)

1. **Deprecate PathBuilder**:
   - Replace with `PathResolver` that uses database
   - Remove file reads from path computation
   - Compute paths from database fields only

2. **Update All Commands**:
   - Epic/feature/task creation uses `PathResolver`
   - Get/list commands read from database
   - No file reads for path determination

3. **Update Discovery**:
   - Extract metadata from files
   - Validate against database
   - Report conflicts (file slug ≠ database slug)

4. **Test**:
   - Create entities with custom paths
   - Create entities with default paths
   - Verify paths match expectations
   - Verify no file reads during path resolution

### Phase 4: Support Flexible Key Lookup (Enhancement)

1. **Update Repositories**:
   - `GetByKey` accepts both `E05` and `E05-slug`
   - Extract numeric key from slugged key
   - Fall back to numeric lookup

2. **Update CLI Commands**:
   - All commands accept both formats
   - Help text explains both work

3. **Test**:
   - `shark epic get E05`
   - `shark epic get E05-task-mgmt-cli-capabilities`
   - Both return same result

---

## Code Architecture Recommendations

### Recommended Module Structure

```
internal/
├── slug/
│   ├── slug.go                    # Slug generation (no change)
│   └── slug_test.go
├── pathresolver/                  # NEW: Database-driven path resolution
│   ├── resolver.go                # PathResolver type
│   ├── epic.go                    # Epic path resolution
│   ├── feature.go                 # Feature path resolution
│   ├── task.go                    # Task path resolution
│   └── resolver_test.go
├── repository/
│   ├── epic_repository.go         # Update: Support key/slug lookup
│   ├── feature_repository.go      # Update: Support key/slug lookup
│   └── task_repository.go
├── cli/commands/
│   ├── epic.go                    # Update: Use PathResolver
│   ├── feature.go                 # Update: Use PathResolver
│   └── task.go
├── sync/
│   ├── discovery.go               # Update: Read from files, validate against DB
│   └── conflict.go                # Update: Compare file slug vs DB slug
└── utils/
    └── path_builder.go            # DEPRECATED: Mark for removal
```

### Key Interface Definitions

```go
// PathResolver provides database-driven path resolution for all entities
type PathResolver interface {
    // ResolveEpicPath resolves epic file path from database
    ResolveEpicPath(ctx context.Context, epicKey string) (string, error)

    // ResolveFeaturePath resolves feature file path from database
    ResolveFeaturePath(ctx context.Context, featureKey string) (string, error)

    // ResolveTaskPath resolves task file path from database
    ResolveTaskPath(ctx context.Context, taskKey string) (string, error)

    // ComputeEpicPath computes expected path from key, slug, and custom path
    ComputeEpicPath(key, slug string, customPath *string) string

    // ComputeFeaturePath computes expected path from key, slug, and custom path
    ComputeFeaturePath(epicKey, featureKey, slug string, featureCustomPath, epicCustomPath *string) string

    // ComputeTaskPath computes expected path from key, slug, and custom path
    ComputeTaskPath(epicKey, featureKey, taskKey, slug string, featureCustomPath, epicCustomPath *string) string
}

// SlugExtractor extracts numeric key from slugged key
type SlugExtractor interface {
    // ExtractNumericKey extracts "E05" from "E05-some-slug"
    ExtractNumericKey(keyOrSlug string) string

    // IsSluggedKey checks if key contains slug
    IsSluggedKey(key string) bool
}
```

---

## Validation and Testing Strategy

### Unit Tests

1. **Slug Generation**:
   - Test deterministic generation
   - Test unicode handling
   - Test special character removal
   - Test truncation

2. **Path Resolution**:
   - Test epic path resolution from database
   - Test feature path resolution with inheritance
   - Test task path resolution with slugs
   - Test custom_folder_path handling

3. **Key Extraction**:
   - Test numeric key extraction from slugged keys
   - Test edge cases (malformed keys)

### Integration Tests

1. **Epic Creation**:
   - Create epic with default path
   - Create epic with custom path
   - Verify slug stored in database
   - Verify file created at expected path

2. **Feature Creation**:
   - Create feature with epic custom path (inherit)
   - Create feature with own custom path (override)
   - Verify slug stored in database

3. **Task Creation**:
   - Create task with slugged filename
   - Verify slug in filename matches database

4. **Discovery**:
   - Discover epics from filesystem
   - Validate slugs against database
   - Report conflicts (file slug ≠ DB slug)

5. **Key Lookup**:
   - Get epic by numeric key (`E05`)
   - Get epic by slugged key (`E05-task-mgmt-cli-capabilities`)
   - Both return same record

### Migration Tests

1. **Schema Migration**:
   - Run migration on test database
   - Verify columns added
   - Verify indexes created

2. **Data Backfill**:
   - Backfill slugs from existing file_path
   - Verify correctness
   - Verify NULL for missing file_path

3. **File Format Migration**:
   - Convert epic files to new format
   - Convert feature files to new format
   - Verify metadata preserved

---

## Backward Compatibility Considerations

### Breaking Changes

1. **File Format Change** (if using YAML frontmatter):
   - Existing epic/feature files need conversion
   - Discovery will fail on old format

2. **PathBuilder Removal**:
   - Code using PathBuilder needs refactor
   - All commands updated to use PathResolver

### Non-Breaking Changes

1. **Add Slug Column**:
   - Existing queries continue to work
   - NULL slugs handled gracefully

2. **Flexible Key Lookup**:
   - Both `E05` and `E05-slug` work
   - Existing commands work unchanged

### Migration Path for Existing Projects

1. **Run Schema Migration**:
   ```bash
   shark migrate add-slug-column
   ```

2. **Backfill Slugs**:
   ```bash
   shark migrate backfill-slugs
   ```

3. **Convert Files** (if using YAML frontmatter):
   ```bash
   shark migrate convert-file-format --dry-run
   shark migrate convert-file-format
   ```

4. **Verify**:
   ```bash
   shark sync --dry-run
   shark epic list
   shark feature list
   ```

---

## Performance Impact Analysis

### Current Performance Issues

1. **Discovery Reads All Files**: Scans filesystem, parses markdown
2. **PathBuilder Recomputes**: Slug generated every time path needed
3. **No Slug Caching**: Slug computed from title on every operation

### Expected Performance Improvements

1. **Path Resolution**:
   - Before: Compute slug from title + build path = ~1ms
   - After: Read from database = ~0.1ms
   - **10x faster**

2. **Discovery**:
   - Before: Parse markdown for every file
   - After: Parse frontmatter only (YAML is faster than markdown)
   - **2-3x faster**

3. **Slug Lookup**:
   - Before: Generate slug, compare with folder name
   - After: Direct database lookup
   - **5x faster**

### Database Size Impact

- Adding `slug` column: ~50-100 bytes per record
- For 1000 epics/features: ~50-100 KB additional storage
- **Negligible impact**

---

## Recommended Implementation Order

1. **Phase 1: Database Schema** (2-3 hours)
   - Add slug columns
   - Create indexes
   - Write migration script
   - Test migration

2. **Phase 2: Slug Storage** (3-4 hours)
   - Update epic creation to store slug
   - Update feature creation to store slug
   - Update task creation to store slug
   - Test creation flow

3. **Phase 3: Backfill Existing Data** (1-2 hours)
   - Extract slugs from file_path
   - Update database records
   - Verify correctness

4. **Phase 4: PathResolver** (4-5 hours)
   - Implement PathResolver
   - Replace PathBuilder usage
   - Test path resolution

5. **Phase 5: Key Lookup Enhancement** (2-3 hours)
   - Update GetByKey methods
   - Support slugged keys
   - Test both formats

6. **Phase 6: File Format (Optional)** (4-6 hours)
   - Decide on format (YAML vs markdown)
   - Write conversion script
   - Update discovery
   - Test sync

**Total Estimated Effort**: 16-23 hours

---

## Conclusion

The current architecture has fundamental issues with metadata management:
- Database is not single source of truth
- Slugs are computed on-the-fly instead of stored
- Inconsistent patterns across entity types
- File system reads required for path resolution

The proposed architecture:
- Adds `slug` column to all entity tables
- Stores slugs at creation time (deterministic, one-time)
- Standardizes file formats (YAML frontmatter recommended)
- Implements database-first PathResolver
- Supports flexible key lookup (numeric or slugged)

**Recommendation**: Implement in phases, starting with database schema changes (non-breaking) and gradually refactoring to PathResolver. File format standardization can be delayed if needed.

**Priority**: HIGH - Current architecture creates technical debt and violates database-first principles.

---

## Appendix A: Example Migration SQL

```sql
-- Migration: Add slug columns
-- Date: 2025-12-30

BEGIN TRANSACTION;

-- Add slug column to epics
ALTER TABLE epics ADD COLUMN slug TEXT;
CREATE INDEX IF NOT EXISTS idx_epics_slug ON epics(slug);

-- Add slug column to features
ALTER TABLE features ADD COLUMN slug TEXT;
CREATE INDEX IF NOT EXISTS idx_features_slug ON features(slug);

-- Add slug column to tasks (optional)
ALTER TABLE tasks ADD COLUMN slug TEXT;
CREATE INDEX IF NOT EXISTS idx_tasks_slug ON tasks(slug);

-- Backfill epics (extract from file_path)
UPDATE epics
SET slug = (
    SELECT CASE
        WHEN file_path LIKE '%/' || key || '-%' THEN
            substr(
                file_path,
                instr(file_path, '/' || key || '-') + length('/' || key || '-'),
                instr(substr(file_path, instr(file_path, '/' || key || '-')), '/') - 1
            )
        ELSE NULL
    END
)
WHERE file_path IS NOT NULL AND slug IS NULL;

-- Backfill features (extract from file_path)
UPDATE features
SET slug = (
    SELECT CASE
        WHEN file_path LIKE '%/' || key || '-%' THEN
            substr(
                file_path,
                instr(file_path, '/' || key || '-') + length('/' || key || '-'),
                instr(substr(file_path, instr(file_path, '/' || key || '-')), '/') - 1
            )
        ELSE NULL
    END
)
WHERE file_path IS NOT NULL AND slug IS NULL;

-- Backfill tasks (extract from filename)
UPDATE tasks
SET slug = (
    SELECT CASE
        WHEN file_path LIKE '%/' || key || '-%' THEN
            REPLACE(
                substr(
                    file_path,
                    instr(file_path, '/' || key || '-') + length('/' || key || '-')
                ),
                '.md',
                ''
            )
        ELSE NULL
    END
)
WHERE file_path IS NOT NULL AND slug IS NULL;

COMMIT;
```

## Appendix B: Example YAML Frontmatter Templates

### Epic Template
```yaml
---
epic_key: {{ .Key }}
slug: {{ .Slug }}
title: {{ .Title }}
description: {{ .Description }}
status: {{ .Status }}
priority: {{ .Priority }}
business_value: {{ .BusinessValue }}
created_at: {{ .CreatedAt }}
updated_at: {{ .UpdatedAt }}
---

# Epic: {{ .Title }}

{{ if .Description }}
## Description

{{ .Description }}
{{ end }}

## Features

{{ range .Features }}
- [{{ .Key }} - {{ .Title }}]({{ .FilePath }})
{{ end }}
```

### Feature Template
```yaml
---
feature_key: {{ .Key }}
epic_key: {{ .EpicKey }}
slug: {{ .Slug }}
title: {{ .Title }}
description: {{ .Description }}
status: {{ .Status }}
execution_order: {{ .ExecutionOrder }}
created_at: {{ .CreatedAt }}
updated_at: {{ .UpdatedAt }}
---

# Feature: {{ .Title }}

## Epic

- [{{ .EpicKey }} - {{ .EpicTitle }}]({{ .EpicFilePath }})

{{ if .Description }}
## Description

{{ .Description }}
{{ end }}

## Tasks

{{ range .Tasks }}
- [{{ .Key }} - {{ .Title }}]({{ .FilePath }})
{{ end }}
```

### Task Template (no change)
```yaml
---
task_key: {{ .Key }}
feature_key: {{ .FeatureKey }}
epic_key: {{ .EpicKey }}
slug: {{ .Slug }}
title: {{ .Title }}
description: {{ .Description }}
status: {{ .Status }}
priority: {{ .Priority }}
agent_type: {{ .AgentType }}
depends_on: {{ .DependsOn }}
created_at: {{ .CreatedAt }}
started_at: {{ .StartedAt }}
completed_at: {{ .CompletedAt }}
---

# Task: {{ .Title }}

{{ if .Description }}
## Description

{{ .Description }}
{{ end }}
```
