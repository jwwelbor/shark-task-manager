# Database Validation Guide

## Overview

The `shark validate` command checks database integrity by verifying file paths and relationships between entities. This guide explains how to use validation, interpret results, and fix common issues.

## Quick Start

```bash
# Run validation
shark validate

# Output as JSON
shark validate --json

# Verbose output
shark validate --verbose
```

## What Validation Checks

### 1. File Path Integrity

Verifies that task file paths in the database point to files that exist on the filesystem.

**Why it matters**: Tasks with broken file paths can't be opened or edited.

**Common causes**:
- Files moved or renamed
- Files deleted
- Repository cloned to different location

### 2. Relationship Integrity

Verifies that parent-child relationships are valid:
- Features reference existing epics
- Tasks reference existing features

**Why it matters**: Orphaned records break navigation and reporting.

**Common causes**:
- Manual database edits
- Interrupted sync operations
- Data import issues

## Running Validation

### Basic Usage

```bash
shark validate
```

**Example Output (Success)**:

```
Shark Validation Report
=======================

Summary
-------
Total entities validated: 247
  - Issues found: 0
  - Broken file paths: 0
  - Orphaned records: 0
Duration: 127ms

Validation Result
-----------------
✓ All validations passed!
```

### JSON Output

For scripting and automation:

```bash
shark validate --json
```

**Example Output**:

```json
{
  "broken_file_paths": [],
  "orphaned_records": [],
  "summary": {
    "total_checked": 247,
    "total_issues": 0,
    "broken_file_paths": 0,
    "orphaned_records": 0
  },
  "duration_ms": 127
}
```

### Verbose Output

For detailed information:

```bash
shark validate --verbose
```

Shows additional details about the validation process.

## Understanding Validation Results

### Exit Codes

- **0**: All validations passed
- **1**: Validation failed (issues found)

Use in scripts:

```bash
if shark validate; then
    echo "Database is valid"
else
    echo "Database has issues - see report above"
    exit 1
fi
```

### Success Example

```
Shark Validation Report
=======================

Summary
-------
Total entities validated: 247
  - Issues found: 0
  - Broken file paths: 0
  - Orphaned records: 0
Duration: 127ms

Validation Result
-----------------
✓ All validations passed!
```

### Failure Example

```
Shark Validation Report
=======================

Summary
-------
Total entities validated: 247
  - Issues found: 5
  - Broken file paths: 3
  - Orphaned records: 2
Duration: 145ms

Broken File Paths
-----------------
  ✗ T-E04-F07-003 [task]
    Path: /home/user/project/docs/plan/E04/E04-F07/tasks/T-E04-F07-003.md
    Issue: File does not exist (may have been moved or deleted)
    Suggestion: Re-scan to update file paths: 'shark sync --incremental' or update path manually in database

  ✗ T-E04-F08-002 [task]
    Path: /home/user/project/docs/plan/E04/E04-F08/tasks/T-E04-F08-002.md
    Issue: File does not exist (may have been moved or deleted)
    Suggestion: Re-scan to update file paths: 'shark sync --incremental' or update path manually in database

Orphaned Records
----------------
  ✗ E05-F02 [feature]
    Missing parent: epic (ID: 999)
    Issue: Orphaned feature: parent epic with ID 999 does not exist
    Suggestion: Create missing parent epic or delete orphaned feature: 'shark feature delete E05-F02'

  ✗ T-E05-F02-001 [task]
    Missing parent: feature (ID: 123)
    Issue: Orphaned task: parent feature with ID 123 does not exist
    Suggestion: Create missing parent feature or delete orphaned task: 'shark task delete T-E05-F02-001'

Validation Result
-----------------
✗ VALIDATION FAILED: Found 5 issue(s)

Next Steps:
  - Run 'shark sync --incremental' to update file paths
  - Review orphaned records and create missing parents or delete orphans
```

## Fixing Validation Issues

### Fixing Broken File Paths

**Problem**: Tasks reference files that don't exist.

**Solution 1: Re-sync** (recommended):

```bash
# Re-scan to update file paths
shark sync
```

This will:
- Find tasks at their new locations
- Update database with correct paths

**Solution 2: Manual Fix**:

If you know the file was deleted:

```bash
shark task delete T-E04-F07-003
```

### Fixing Orphaned Records

**Problem**: Records reference non-existent parents.

**Solution 1: Create Missing Parent**:

```bash
# Create missing epic
shark epic create E05

# Create missing feature
shark feature create E05-F02
```

**Solution 2: Delete Orphaned Record**:

```bash
# Delete orphaned feature
shark feature delete E05-F02

# Delete orphaned task
shark task delete T-E05-F02-001
```

### Complete Workflow

```bash
# 1. Run validation
shark validate

# 2. If validation fails, fix issues
shark sync  # Fix broken file paths

# 3. Manually fix orphaned records
# (use suggestions from validation report)

# 4. Re-run validation
shark validate

# 5. Verify success
echo $?  # Should be 0
```

## Common Scenarios

### Scenario 1: After Git Pull

After pulling changes from git:

```bash
# Files may have moved or been deleted
shark sync    # Update paths
shark validate # Verify integrity
```

### Scenario 2: After Manual Edits

After manually editing the database:

```bash
shark validate  # Check for orphaned records
```

### Scenario 3: CI/CD Pipeline

In automated checks:

```bash
#!/bin/bash
set -e

# Sync and validate
shark sync
shark validate

# If we get here, all validations passed
echo "✓ Database is valid"
```

### Scenario 4: Cleanup Old Data

When cleaning up:

```bash
# Find issues
shark validate

# Review orphaned records
# Delete if no longer needed
shark task delete T-E99-F99-001
shark feature delete E99-F99

# Verify clean
shark validate
```

## Performance

### Expected Performance

- **Small databases** (<100 entities): <50ms
- **Medium databases** (100-1000 entities): 50-500ms
- **Large databases** (>1000 entities): <1s

Target: 1000 entities validated in <1 second

### Performance Tips

Validation is fast and can be run frequently:

```bash
# Add to pre-commit hook
shark validate --quiet

# Add to CI pipeline
shark validate --json > validation-report.json
```

## JSON Output Schema

### Success Response

```json
{
  "broken_file_paths": [],
  "orphaned_records": [],
  "summary": {
    "total_checked": 247,
    "total_issues": 0,
    "broken_file_paths": 0,
    "orphaned_records": 0
  },
  "duration_ms": 127
}
```

### Failure Response

```json
{
  "broken_file_paths": [
    {
      "entity_type": "task",
      "entity_key": "T-E04-F07-003",
      "file_path": "/path/to/file.md",
      "issue": "File does not exist (may have been moved or deleted)",
      "suggested_fix": "Re-scan to update file paths: 'shark sync --incremental' or update path manually in database"
    }
  ],
  "orphaned_records": [
    {
      "entity_type": "feature",
      "entity_key": "E05-F02",
      "missing_parent_type": "epic",
      "missing_parent_id": 999,
      "issue": "Orphaned feature: parent epic with ID 999 does not exist",
      "suggested_fix": "Create missing parent epic or delete orphaned feature: 'shark feature delete E05-F02'"
    }
  ],
  "summary": {
    "total_checked": 247,
    "total_issues": 5,
    "broken_file_paths": 3,
    "orphaned_records": 2
  },
  "duration_ms": 145
}
```

## Integration with Sync

Validation is complementary to sync:

```bash
# 1. Sync updates database from files
shark sync

# 2. Validate checks database integrity
shark validate
```

**Best Practice**: Run validation after sync:

```bash
shark sync && shark validate
```

## Troubleshooting

### Problem: Many broken file paths

**Likely cause**: Files moved in bulk

**Solution**:
```bash
shark sync  # Re-scan entire tree
shark validate
```

### Problem: Orphaned records after import

**Likely cause**: Import script didn't create full hierarchy

**Solution**:
```bash
# Create missing parents
shark epic create E05
shark feature create E05-F02

# Or delete orphans
shark task delete T-E05-F02-001

# Verify
shark validate
```

### Problem: Validation slow

**Check database size**:
```bash
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM tasks"
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM features"
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM epics"
```

**Normal**: <1s for 1000 entities
**Slow**: >1s may indicate database issues

## Advanced Usage

### Scripting with JSON

```bash
# Check if validation passed
if shark validate --json | jq -e '.summary.total_issues == 0' > /dev/null; then
    echo "✓ Valid"
else
    echo "✗ Invalid"
    shark validate --json | jq '.summary'
fi
```

### Filtering Issues

```bash
# Show only broken file paths
shark validate --json | jq '.broken_file_paths[]'

# Show only orphaned records
shark validate --json | jq '.orphaned_records[]'

# Count issues by type
shark validate --json | jq '.summary.broken_file_paths'
```

### Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Run validation before commit
if ! shark validate --quiet; then
    echo "✗ Database validation failed"
    echo "Run 'shark validate' for details"
    exit 1
fi
```

## See Also

- [Sync Reporting Guide](sync-reporting.md) - Understanding sync reports
- [JSON Schema Reference](../api/json-schema.md) - Complete schema
- [CLI Reference](../CLI.md) - All CLI commands
