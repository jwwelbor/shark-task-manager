# Sync Reporting Guide

## Overview

The `shark sync` command provides comprehensive reporting of synchronization operations between task files and the database. This guide explains how to use the reporting features, interpret the output, and troubleshoot common issues.

## Output Formats

### Text Output (Default)

By default, `shark sync` outputs a human-readable report with color-coded information:

```bash
shark sync
```

**Example Output:**

```
Shark Scan Report
=================

Scan completed at 2025-12-18 15:30:45
Duration: 2.3 seconds
Validation level: basic
Documentation root: docs/plan

Summary
-------
Total files scanned: 150
  ✓ Matched: 145
  ✗ Skipped: 5

Breakdown by Type
-----------------
Epics:
  ✓ Matched: 12

Features:
  ✓ Matched: 48

Tasks:
  ✓ Matched: 85
  ✗ Skipped: 5

Errors and Warnings
-------------------

Parse Errors (3):
  ERROR: docs/plan/E04/E04-F07/tasks/T-E04-F07-005.md
    Missing required field: task_key
    Suggestion: Add task_key field to frontmatter

  ERROR: docs/plan/E04/E04-F08/tasks/invalid.md:12
    Invalid YAML frontmatter: unclosed quote
    Suggestion: Fix YAML syntax and re-run sync

Validation Warnings (2):
  WARNING: docs/plan/E04/E04-F07/tasks/T-E04-F07-002.md
    Task title extracted from filename (no title in frontmatter)
    Suggestion: Add explicit title field for better clarity

Scan Complete
-------------
Successfully imported 145 items:
  - 12 epics
  - 48 features
  - 85 tasks

Run 'shark validate' to verify database integrity.
```

### JSON Output

For scripting and automation, use `--output=json`:

```bash
shark sync --output=json
```

**Example Output:**

```json
{
  "schema_version": "1.0",
  "status": "success",
  "dry_run": false,
  "metadata": {
    "timestamp": "2025-12-18T15:30:45Z",
    "duration_seconds": 2.3,
    "validation_level": "basic",
    "documentation_root": "docs/plan",
    "patterns": {
      "task": "enabled"
    }
  },
  "counts": {
    "scanned": 150,
    "matched": 145,
    "skipped": 5
  },
  "entities": {
    "epics": {
      "matched": 12,
      "skipped": 0
    },
    "features": {
      "matched": 48,
      "skipped": 0
    },
    "tasks": {
      "matched": 85,
      "skipped": 5
    },
    "related_docs": {
      "matched": 0,
      "skipped": 0
    }
  },
  "errors": [
    {
      "file_path": "docs/plan/E04/E04-F07/tasks/T-E04-F07-005.md",
      "reason": "Missing required field: task_key",
      "suggested_fix": "Add task_key field to frontmatter",
      "error_type": "parse_error"
    }
  ],
  "warnings": [
    {
      "file_path": "docs/plan/E04/E04-F07/tasks/T-E04-F07-002.md",
      "reason": "Task title extracted from filename (no title in frontmatter)",
      "suggested_fix": "Add explicit title field for better clarity",
      "error_type": "validation_warning"
    }
  ]
}
```

### Quiet Mode

For use in scripts where you only want to see errors:

```bash
shark sync --quiet
```

In quiet mode:
- No output is printed for successful operations
- Errors are printed to stderr
- Exit code is 0 for success, non-zero for errors

## Common Scenarios

### Scenario 1: Initial Import

When importing tasks for the first time:

```bash
# Preview what will be imported
shark sync --dry-run

# Review the report, then do actual import
shark sync

# Verify database integrity
shark validate
```

### Scenario 2: Regular Sync After Changes

```bash
# Sync changed files only (incremental)
shark sync

# Review any warnings or errors in the report
```

### Scenario 3: Troubleshooting Errors

If the sync report shows errors:

1. **Review the error details** - Each error includes:
   - File path and line number (if applicable)
   - Reason for the error
   - Suggested fix

2. **Fix the issues** - Common fixes:
   ```yaml
   # Missing task_key
   ---
   task_key: T-E04-F07-001  # Add this
   status: todo
   ---
   ```

3. **Re-run sync** - After fixing:
   ```bash
   shark sync
   ```

### Scenario 4: Bulk Operations

When importing many files:

```bash
# Use JSON output for processing results
shark sync --output=json > sync-results.json

# Parse results with jq
cat sync-results.json | jq '.counts.matched'
cat sync-results.json | jq '.errors[] | .file_path'
```

### Scenario 5: CI/CD Integration

In automated pipelines:

```bash
#!/bin/bash
# Run sync in quiet mode
if shark sync --quiet; then
    echo "Sync successful"
    exit 0
else
    echo "Sync failed - see errors above"
    exit 1
fi
```

## Understanding the Report

### Counts Section

- **Scanned**: Total files found and examined
- **Matched**: Files successfully parsed and imported/updated
- **Skipped**: Files that couldn't be processed (see errors)

### Entity Breakdown

Shows statistics per entity type:
- **Epics**: Top-level project containers
- **Features**: Feature sets within epics
- **Tasks**: Individual work items
- **Related Docs**: PRPs and other documentation

### Errors vs Warnings

**Errors** (red ✗):
- Prevent file from being imported
- Must be fixed for sync to complete successfully
- Examples: missing required fields, invalid YAML

**Warnings** (yellow ⚠):
- File is imported but may need attention
- Non-critical issues
- Examples: missing optional fields, inferred values

### Error Types

1. **parse_error**: File couldn't be parsed
   - Invalid YAML frontmatter
   - Missing required fields
   - File access issues

2. **validation_failure**: File parsed but validation failed
   - Invalid field values
   - Reference to non-existent parent

3. **validation_warning**: Minor validation issue
   - Missing optional fields
   - Inferred values used

4. **pattern_mismatch**: File doesn't match expected pattern
   - Wrong filename format
   - Wrong directory structure

## Troubleshooting

### Problem: "Missing required field: task_key"

**Solution**: Add task_key to frontmatter:

```yaml
---
task_key: T-E04-F07-001
status: todo
---
```

### Problem: "Invalid YAML frontmatter"

**Solution**: Check YAML syntax:

```yaml
---
# Good
task_key: T-E04-F07-001
description: "This is a description"

# Bad - unclosed quote
description: "This is a description
---
```

### Problem: "Feature E04-F07 not found"

**Solution**: Either:
1. Create the feature first: `shark feature create E04-F07`
2. Use `--create-missing` flag: `shark sync --create-missing`

### Problem: Too many skipped files

**Check**:
1. File naming follows pattern: `T-E##-F##-###.md`
2. Files are in correct directory: `docs/plan/E##/E##-F##/tasks/`
3. YAML frontmatter is valid
4. Required fields are present

## Performance Tips

### For Large Repositories

1. **Use incremental sync** (automatic):
   ```bash
   shark sync  # Only syncs changed files
   ```

2. **Sync specific folders**:
   ```bash
   shark sync --folder=docs/plan/E04/E04-F07
   ```

3. **Use quiet mode in scripts**:
   ```bash
   shark sync --quiet
   ```

### Expected Performance

- **Small projects** (<50 files): <1 second
- **Medium projects** (50-500 files): 1-5 seconds
- **Large projects** (>500 files): 5-30 seconds

Reporting adds <5% overhead to sync operations.

## Advanced Usage

### Dry-Run Mode

Preview changes without modifying the database:

```bash
shark sync --dry-run
```

The report shows:
- What would be imported/updated
- Conflicts that would be resolved
- Warnings and errors

### Pattern Selection

Sync specific file types:

```bash
# Sync only task files (default)
shark sync --pattern=task

# Sync only PRP files
shark sync --pattern=prp

# Sync both
shark sync --pattern=task --pattern=prp
```

### Conflict Resolution

When file and database differ:

```bash
# File values win (default)
shark sync --strategy=file-wins

# Database values win
shark sync --strategy=database-wins

# Newest modification wins
shark sync --strategy=newer-wins
```

The report shows resolved conflicts:

```
Conflicts Resolved: 3
  T-E04-F07-001:
    Field: title
    Database: "Old Title"
    File: "New Title"
    Resolution: file-wins → "New Title"
```

## Integration with Validate

After sync, verify database integrity:

```bash
# Full workflow
shark sync
shark validate
```

The sync report will remind you:

```
Run 'shark validate' to verify database integrity.
```

## JSON Schema

For detailed JSON schema documentation, see [JSON Schema Reference](../api/json-schema.md).

## See Also

- [Validation Guide](validation.md) - Database validation
- [JSON Schema Reference](../api/json-schema.md) - Complete schema
- [CLI Reference](../CLI.md) - All CLI commands
