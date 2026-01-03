# Architecture Decision: File Path Management

**Date:** 2026-01-02
**Status:** APPROVED
**Task:** T-E07-F18-001
**Participants:** Product Manager, UX Designer, Architect

---

## Context

Investigation of bug "feature update --path doesn't work" revealed a fundamental architectural flaw:
- Database has `custom_folder_path` column that is stored but **never used** in file path calculations
- The problem affects epics, features, and tasks
- Multiple path-related columns create confusion and inconsistency

## Decision

**Use single `file_path` column storing the full file path.**

### Schema Design

```sql
-- REMOVE these columns (they cause the bug)
ALTER TABLE epics DROP COLUMN custom_folder_path;
ALTER TABLE features DROP COLUMN custom_folder_path;

-- KEEP and use correctly
epics.file_path TEXT        -- Full path: "docs/plan/E01-epic-name/epic.md"
features.file_path TEXT     -- Full path: "docs/plan/E01/E01-F01-feature/feature.md"
tasks.file_path TEXT        -- Full path: "docs/plan/E01/E01-F01/tasks/T-E01-F01-001.md"
```

**No path hierarchy, no path segments, no inheritance. Just store the full file path.**

### CLI Design

**Support multiple flag aliases for flexibility:**

```bash
# Primary flag (shown in documentation)
shark epic create "Title" --file="docs/custom/location.md"

# Aliases (hidden in help, but functional)
shark epic create "Title" --filepath="docs/custom/location.md"
shark epic create "Title" --path="docs/custom/location.md"
```

All three flags do the same thing - specify the full file path.

### Typical Workflow (95% case)

```bash
# User doesn't specify path - shark uses default convention
shark epic create "Authentication System"
# → Computes default: docs/plan/E01-authentication-system/epic.md
# → Creates file from template at that location
# → Stores: epic.file_path = "docs/plan/E01-authentication-system/epic.md"

shark feature create --epic=E01 "User Login"
# → Computes default: docs/plan/E01-authentication-system/E01-F01-user-login/feature.md
# → Creates file from template
# → Stores: feature.file_path = "docs/plan/E01-.../E01-F01-.../feature.md"
```

### Custom Path Workflow (5% case)

```bash
# User specifies custom location
shark epic create "Legacy Spec" --file="docs/roadmap/2025/legacy.md"
# → Creates file at: docs/roadmap/2025/legacy.md
# → Stores: epic.file_path = "docs/roadmap/2025/legacy.md"

# Import existing file
shark epic import docs/existing/spec.md --key=E01
# → Stores: epic.file_path = "docs/existing/spec.md"
```

### Update Workflow (The Bug Fix)

```bash
# Update file location
shark feature update E01-F01 --file="docs/new/location/feature.md"
# → Moves file: old_path → docs/new/location/feature.md
# → Updates database: feature.file_path = "docs/new/location/feature.md"
```

**This is the fix.** Currently it stores in wrong column (`custom_folder_path`) and doesn't update `file_path`.

---

## Rationale

### Product Perspective

**Job to be done:**
- AI agents get task from shark
- Read spec file at provided location
- Update status frequently via CLI

**Critical path:** Status tracking (frequent), not file organization (rare)

**User needs:**
1. ✅ Specify file location when creating (5% case)
2. ✅ Fast status queries (shark task next, shark task list)
3. ✅ Import existing files
4. ❌ Reorganize folder hierarchies (not needed - "files don't move")

**Hierarchical path system solves #4, which users don't need.**

### UX Perspective

**Mental model complexity:**

| Approach | Concepts User Must Understand |
|----------|------------------------------|
| Hierarchical paths | docs_root, epic.path, feature.path, task.path, inheritance, NULL=default (6+ concepts) |
| Full paths | Full file path (1 concept) |

**Cognitive load:** Simple wins.

**Intuitive commands:**
```bash
# Simple (full path)
shark epic create "Title" --file="docs/roadmap/2025/epic.md"
# → Clear: file lives at docs/roadmap/2025/epic.md

# Complex (hierarchical)
shark epic create "Title" --path="roadmap/2025" --filename="epic.md"
# → Confusing: where does it actually live? docs/roadmap/2025/epic.md? docs/plan/roadmap/2025/epic.md?
```

### Technical Perspective

**Performance:**
- Status updates are frequent operations
- Hierarchical: Requires joins (task → feature → epic) to compute path
- Full path: Direct column lookup
- **Full path is faster for the common case**

**Code complexity:**
```go
// Hierarchical approach
func GetTaskPath(task) string {
    feature := db.GetFeature(task.FeatureID)  // Query 1
    epic := db.GetEpic(feature.EpicID)         // Query 2
    return PathResolver.Resolve(task, feature, epic)  // Complex logic
}

// Full path approach
func GetTaskPath(task) string {
    return task.FilePath  // Done!
}
```

**Import/Discovery:**
```go
// Hierarchical: Must parse structure
shark epic import docs/roadmap/2025/auth/epic.md
// → Must parse: epic.path="roadmap/2025/auth", epic.filename="epic.md"

// Full path: Store as-is
shark epic import docs/roadmap/2025/auth/epic.md
// → Store: epic.file_path="docs/roadmap/2025/auth/epic.md"
```

### Architecture Review

Architect reviewed hierarchical approach and noted:
- ✅ Hierarchical paths enable auto-cascading moves (epic moves → children follow)
- ❌ **But users don't need cascading** - files are static
- ❌ Adds complexity (joins, NULL handling, PathResolver)
- ❌ Solves problem users don't have

**Verdict:** Hierarchical is elegant engineering solving the wrong problem.

---

## Consequences

### Positive

1. **Simplicity**: One column, one concept, one value
2. **Performance**: No joins for path resolution
3. **Fast status queries**: Critical for AI agent workflow
4. **Easy import**: Just store the path as-is
5. **Clear semantics**: `file_path` = where file actually lives
6. **Trivial bug fix**: Just update the correct column

### Negative

1. **No auto-cascading**: If you wanted to move epic + all children, must update each manually
2. **Path duplication**: "docs/plan/E01/" stored in multiple rows
   - **Mitigation:** Not a problem - disk is cheap, files don't move

### Migration Strategy

**Phase 1: Fix the bug (use file_path correctly)**
```sql
-- Feature update should update file_path, not custom_folder_path
UPDATE features SET file_path = ? WHERE key = ?
```

**Phase 2: Clean up schema (remove unused column)**
```sql
-- After verifying fix works
ALTER TABLE epics DROP COLUMN custom_folder_path;
ALTER TABLE features DROP COLUMN custom_folder_path;
```

**Backward compatibility:** Existing `file_path` values work as-is. No migration needed.

---

## Alternatives Considered

### Alternative 1: Hierarchical Path Segments

**Approach:** Store path segments at each level, compute full path by inheritance
```sql
epic.path = "roadmap/2025"
epic.filename = "epic.md"
feature.path = "features"
feature.filename = "prd.md"
→ Full path: docs/{epic.path}/{feature.path}/{feature.filename}
```

**Rejected because:**
- Solves reorganization use case that users don't need
- Adds complexity (6+ concepts, joins, NULL handling)
- Slower queries (must join to compute path)
- Harder to import existing files

### Alternative 2: NULL for Defaults, Store for Custom

**Approach:** Store NULL when using default convention, only store custom paths
```sql
epic.file_path = NULL  -- use convention: docs/plan/{key}/epic.md
epic.file_path = "docs/custom.md"  -- explicit override
```

**Rejected because:**
- Must check NULL and compute default on every read
- Slower (computation on hot path)
- Files don't move, so path is permanent anyway
- No benefit to optimize for default vs custom

---

## Implementation Plan

### Step 1: Update Feature Update Command
**File:** `internal/cli/commands/feature.go`

```go
// In runFeatureUpdate function (around line 1585)
if fileFlag != "" {
    // Move file on disk
    oldPath := feature.FilePath
    newPath := fileFlag
    if err := os.Rename(oldPath, newPath); err != nil {
        return err
    }

    // Update database - THIS IS THE FIX
    feature.FilePath = newPath
    if err := featureRepo.Update(ctx, feature); err != nil {
        return err
    }
}
```

### Step 2: Add Flag Aliases
**File:** `internal/cli/commands/epic.go`, `feature.go`, `task.go`

```go
// Add to create/update commands
cmd.Flags().String("file", "", "Full path to file")
cmd.Flags().String("filepath", "", "Full path to file (alias for --file)")
cmd.Flags().String("path", "", "Full path to file (alias for --file)")

// Hide aliases from help
cmd.Flags().MarkHidden("filepath")
cmd.Flags().MarkHidden("path")
```

### Step 3: Schema Cleanup (Future)
```sql
-- After fix is verified
ALTER TABLE epics DROP COLUMN custom_folder_path;
ALTER TABLE features DROP COLUMN custom_folder_path;
```

### Step 4: Update Documentation
- Update CLI_REFERENCE.md with --file flag examples
- Update MIGRATION_CUSTOM_PATHS.md (mark as deprecated)
- Update CLAUDE.md to reflect simplified path handling

---

## Acceptance Criteria

- [x] Architecture decision documented
- [ ] Feature update --file works correctly (moves file + updates database)
- [ ] Epic/feature/task commands support --file, --filepath, --path flags
- [ ] Default path computation unchanged (backward compatible)
- [ ] Tests verify file_path is updated correctly
- [ ] Documentation updated

---

## References

- Original bug report: Idea I-2026-01-02-05
- Analysis: `dev-artifacts/2026-01-02-feature-update-path-bug/CURRENT_IMPLEMENTATION_ANALYSIS.md`
- Previous proposal: `dev-artifacts/2026-01-02-feature-update-path-bug/PATH_FILENAME_ARCHITECTURE.md` (hierarchical approach)
- Architect review: Agent afdb5bc

---

## Sign-off

- **Product Manager**: ✅ Approved - Solves actual user needs, not imaginary ones
- **UX Designer**: ✅ Approved - Simple mental model, low cognitive load
- **Architect**: ✅ Approved - Performance optimized for common case, maintainable
- **Client**: ✅ Approved - "Files don't move" + fast status queries = correct priorities
