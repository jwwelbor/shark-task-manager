# JSON Schema Reference

## Overview

This document describes the JSON output schemas for `shark sync` and `shark validate` commands. Use these schemas when integrating Shark with other tools or building automation scripts.

## Sync Report Schema

### Schema Version: 1.0

Used by: `shark sync --output=json`

### Top-Level Structure

```json
{
  "schema_version": "string",
  "status": "success | failure",
  "dry_run": boolean,
  "metadata": { ScanMetadata },
  "counts": { ScanCounts },
  "entities": { EntityBreakdown },
  "skipped_files": [ SkippedFileEntry ],
  "errors": [ SkippedFileEntry ],
  "warnings": [ SkippedFileEntry ]
}
```

### ScanMetadata

```json
{
  "timestamp": "ISO8601 datetime",
  "duration_seconds": number,
  "validation_level": "basic | strict",
  "documentation_root": "string (path)",
  "patterns": {
    "pattern_name": "enabled | disabled"
  }
}
```

**Fields**:
- `timestamp`: When the scan started (ISO 8601 format)
- `duration_seconds`: Total scan duration in seconds (float)
- `validation_level`: Validation strictness level
- `documentation_root`: Root directory that was scanned
- `patterns`: Map of patterns and their enabled/disabled status

**Example**:
```json
{
  "timestamp": "2025-12-18T15:30:45Z",
  "duration_seconds": 2.347,
  "validation_level": "basic",
  "documentation_root": "docs/plan",
  "patterns": {
    "task": "enabled",
    "prp": "disabled"
  }
}
```

### ScanCounts

```json
{
  "scanned": integer,
  "matched": integer,
  "skipped": integer
}
```

**Fields**:
- `scanned`: Total files examined
- `matched`: Files successfully parsed and imported
- `skipped`: Files that couldn't be processed

**Invariant**: `scanned = matched + skipped`

**Example**:
```json
{
  "scanned": 150,
  "matched": 145,
  "skipped": 5
}
```

### EntityBreakdown

```json
{
  "epics": { EntityCounts },
  "features": { EntityCounts },
  "tasks": { EntityCounts },
  "related_docs": { EntityCounts }
}
```

**Fields**:
- `epics`: Counts for epic-level files
- `features`: Counts for feature-level files
- `tasks`: Counts for task files
- `related_docs`: Counts for PRP and other documentation

### EntityCounts

```json
{
  "matched": integer,
  "skipped": integer
}
```

**Fields**:
- `matched`: Successfully processed entities of this type
- `skipped`: Failed entities of this type

**Example**:
```json
{
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
}
```

### SkippedFileEntry

```json
{
  "file_path": "string",
  "reason": "string",
  "suggested_fix": "string",
  "error_type": "string",
  "line_number": integer | null
}
```

**Fields**:
- `file_path`: Absolute or relative path to the file
- `reason`: Human-readable explanation of why the file was skipped
- `suggested_fix`: Actionable suggestion to fix the issue
- `error_type`: Machine-readable error category
- `line_number`: Line number where error occurred (optional)

**Error Types**:
- `parse_error`: File couldn't be parsed (YAML errors, etc.)
- `validation_failure`: File parsed but validation failed
- `validation_warning`: Minor validation issue
- `pattern_mismatch`: File doesn't match expected pattern
- `file_access_error`: File couldn't be read

**Example**:
```json
{
  "file_path": "docs/plan/E04/E04-F07/tasks/T-E04-F07-005.md",
  "reason": "Missing required field: task_key",
  "suggested_fix": "Add task_key field to frontmatter",
  "error_type": "parse_error",
  "line_number": null
}
```

### Complete Example

```json
{
  "schema_version": "1.0",
  "status": "success",
  "dry_run": false,
  "metadata": {
    "timestamp": "2025-12-18T15:30:45Z",
    "duration_seconds": 2.347,
    "validation_level": "basic",
    "documentation_root": "docs/plan",
    "patterns": {
      "task": "enabled",
      "prp": "disabled"
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
  "skipped_files": [],
  "errors": [
    {
      "file_path": "docs/plan/E04/E04-F07/tasks/T-E04-F07-005.md",
      "reason": "Missing required field: task_key",
      "suggested_fix": "Add task_key field to frontmatter",
      "error_type": "parse_error",
      "line_number": null
    },
    {
      "file_path": "docs/plan/E04/E04-F08/tasks/invalid.md",
      "reason": "Invalid YAML frontmatter: unclosed quote",
      "suggested_fix": "Fix YAML syntax and re-run sync",
      "error_type": "parse_error",
      "line_number": 12
    }
  ],
  "warnings": [
    {
      "file_path": "docs/plan/E04/E04-F07/tasks/T-E04-F07-002.md",
      "reason": "Task title extracted from filename (no title in frontmatter)",
      "suggested_fix": "Add explicit title field for better clarity",
      "error_type": "validation_warning",
      "line_number": null
    }
  ]
}
```

## Validation Report Schema

### Schema Version: 1.0

Used by: `shark validate --json`

### Top-Level Structure

```json
{
  "broken_file_paths": [ ValidationFailure ],
  "orphaned_records": [ ValidationFailure ],
  "summary": { ValidationSummary },
  "duration_ms": integer
}
```

### ValidationFailure

```json
{
  "entity_type": "epic | feature | task",
  "entity_key": "string",
  "file_path": "string",
  "missing_parent_type": "epic | feature",
  "missing_parent_id": integer,
  "issue": "string",
  "suggested_fix": "string"
}
```

**Fields**:
- `entity_type`: Type of entity with the issue
- `entity_key`: Unique key of the entity
- `file_path`: File path (for broken file path issues)
- `missing_parent_type`: Type of missing parent (for orphaned records)
- `missing_parent_id`: ID of missing parent (for orphaned records)
- `issue`: Human-readable description of the issue
- `suggested_fix`: Actionable suggestion to fix the issue

**Broken File Path Example**:
```json
{
  "entity_type": "task",
  "entity_key": "T-E04-F07-003",
  "file_path": "/path/to/missing/file.md",
  "issue": "File does not exist (may have been moved or deleted)",
  "suggested_fix": "Re-scan to update file paths: 'shark sync --incremental' or update path manually in database"
}
```

**Orphaned Record Example**:
```json
{
  "entity_type": "feature",
  "entity_key": "E05-F02",
  "missing_parent_type": "epic",
  "missing_parent_id": 999,
  "issue": "Orphaned feature: parent epic with ID 999 does not exist",
  "suggested_fix": "Create missing parent epic or delete orphaned feature: 'shark feature delete E05-F02'"
}
```

### ValidationSummary

```json
{
  "total_checked": integer,
  "total_issues": integer,
  "broken_file_paths": integer,
  "orphaned_records": integer
}
```

**Fields**:
- `total_checked`: Total entities validated
- `total_issues`: Total issues found
- `broken_file_paths`: Count of broken file path issues
- `orphaned_records`: Count of orphaned record issues

**Invariant**: `total_issues = broken_file_paths + orphaned_records`

**Example**:
```json
{
  "total_checked": 247,
  "total_issues": 5,
  "broken_file_paths": 3,
  "orphaned_records": 2
}
```

### Complete Example (Success)

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

### Complete Example (Failure)

```json
{
  "broken_file_paths": [
    {
      "entity_type": "task",
      "entity_key": "T-E04-F07-003",
      "file_path": "/home/user/project/docs/plan/E04/E04-F07/tasks/T-E04-F07-003.md",
      "issue": "File does not exist (may have been moved or deleted)",
      "suggested_fix": "Re-scan to update file paths: 'shark sync --incremental' or update path manually in database"
    },
    {
      "entity_type": "task",
      "entity_key": "T-E04-F08-002",
      "file_path": "/home/user/project/docs/plan/E04/E04-F08/tasks/T-E04-F08-002.md",
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
    },
    {
      "entity_type": "task",
      "entity_key": "T-E05-F02-001",
      "missing_parent_type": "feature",
      "missing_parent_id": 123,
      "issue": "Orphaned task: parent feature with ID 123 does not exist",
      "suggested_fix": "Create missing parent feature or delete orphaned task: 'shark task delete T-E05-F02-001'"
    }
  ],
  "summary": {
    "total_checked": 247,
    "total_issues": 4,
    "broken_file_paths": 2,
    "orphaned_records": 2
  },
  "duration_ms": 145
}
```

## Usage Examples

### Parsing Sync Results

```bash
# Check if sync succeeded
STATUS=$(shark sync --output=json | jq -r '.status')
if [ "$STATUS" = "success" ]; then
    echo "Sync successful"
fi

# Get error count
ERROR_COUNT=$(shark sync --output=json | jq '.errors | length')
echo "Errors: $ERROR_COUNT"

# List all error file paths
shark sync --output=json | jq -r '.errors[].file_path'

# Get matched task count
shark sync --output=json | jq '.entities.tasks.matched'
```

### Parsing Validation Results

```bash
# Check if validation passed
ISSUES=$(shark validate --json | jq '.summary.total_issues')
if [ "$ISSUES" -eq 0 ]; then
    echo "Validation passed"
else
    echo "Validation failed: $ISSUES issues"
fi

# List broken file paths
shark validate --json | jq -r '.broken_file_paths[].file_path'

# List orphaned records
shark validate --json | jq -r '.orphaned_records[].entity_key'

# Get specific issue details
shark validate --json | jq '.broken_file_paths[] | {key: .entity_key, path: .file_path}'
```

### CI/CD Integration

```bash
#!/bin/bash
set -e

# Run sync
SYNC_RESULT=$(shark sync --output=json)
echo "$SYNC_RESULT" > sync-report.json

# Check for errors
ERROR_COUNT=$(echo "$SYNC_RESULT" | jq '.errors | length')
if [ "$ERROR_COUNT" -gt 0 ]; then
    echo "Sync had $ERROR_COUNT errors"
    echo "$SYNC_RESULT" | jq '.errors'
    exit 1
fi

# Run validation
VALIDATE_RESULT=$(shark validate --json)
echo "$VALIDATE_RESULT" > validation-report.json

# Check for issues
ISSUE_COUNT=$(echo "$VALIDATE_RESULT" | jq '.summary.total_issues')
if [ "$ISSUE_COUNT" -gt 0 ]; then
    echo "Validation found $ISSUE_COUNT issues"
    echo "$VALIDATE_RESULT" | jq '.summary'
    exit 1
fi

echo "✓ All checks passed"
```

### Python Integration

```python
import json
import subprocess

# Run sync and parse results
result = subprocess.run(
    ['shark', 'sync', '--output=json'],
    capture_output=True,
    text=True
)
sync_data = json.loads(result.stdout)

# Process results
if sync_data['status'] == 'success':
    matched = sync_data['counts']['matched']
    print(f"✓ Imported {matched} tasks")
else:
    errors = sync_data['errors']
    print(f"✗ {len(errors)} errors")
    for error in errors:
        print(f"  - {error['file_path']}: {error['reason']}")
```

### TypeScript Integration

```typescript
interface SyncReport {
  schema_version: string;
  status: 'success' | 'failure';
  dry_run: boolean;
  metadata: ScanMetadata;
  counts: ScanCounts;
  entities: EntityBreakdown;
  errors: SkippedFileEntry[];
  warnings: SkippedFileEntry[];
}

interface ValidationResult {
  broken_file_paths: ValidationFailure[];
  orphaned_records: ValidationFailure[];
  summary: ValidationSummary;
  duration_ms: number;
}

// Run sync
const syncOutput = execSync('shark sync --output=json').toString();
const syncReport: SyncReport = JSON.parse(syncOutput);

if (syncReport.status === 'success') {
  console.log(`✓ Imported ${syncReport.counts.matched} tasks`);
} else {
  console.error(`✗ ${syncReport.errors.length} errors`);
}
```

## Schema Versioning

The `schema_version` field indicates the JSON schema version. This document describes version `1.0`.

Future versions will increment the version number and maintain backwards compatibility where possible. Breaking changes will result in a major version bump.

## See Also

- [Sync Reporting Guide](../user-guide/sync-reporting.md) - Understanding sync reports
- [Validation Guide](../user-guide/validation.md) - Database validation
- [CLI Reference](../CLI.md) - All CLI commands
