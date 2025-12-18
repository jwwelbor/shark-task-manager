# Incremental Sync

## Overview

Incremental sync is an automatic performance optimization that makes `shark sync` dramatically faster by processing only files that have changed since the last sync. This feature is enabled automatically and requires no configuration.

## How It Works

### Automatic Behavior

When you run `shark sync`:

1. **First Sync**: Performs a full scan of all task files and records the completion time in `.sharkconfig.json`
2. **Subsequent Syncs**: Automatically filters to process only files modified since the last sync
3. **Result**: Sync operations complete in seconds instead of minutes for large codebases

### Performance Benefits

| File Changes | Full Scan | Incremental Sync | Improvement |
|-------------|-----------|------------------|-------------|
| 0 changes | ~10s | <1s | 10x faster |
| 1-10 files | ~10s | <2s | 5x faster |
| 100 files | ~30s | <5s | 6x faster |
| 500 files | ~120s | <30s | 4x faster |

## Usage

### Basic Sync (Automatic Incremental)

```bash
# First sync - performs full scan
shark sync

# Subsequent syncs - automatically incremental
shark sync
```

No flags or configuration needed! Incremental sync is enabled automatically based on `.sharkconfig.json`.

### Force Full Scan

To force a complete rescan of all files (ignoring incremental filtering):

```bash
shark sync --force-full-scan
```

Use this when:
- You suspect the incremental state is incorrect
- Files were modified outside of normal workflows
- You want to verify all files are in sync

### Dry Run with Incremental

Preview what would be synced without making changes:

```bash
shark sync --dry-run
```

The dry-run respects incremental filtering and shows which files would be processed.

## Configuration

### Last Sync Time

The last sync timestamp is automatically stored in `.sharkconfig.json`:

```json
{
  "last_sync_time": "2025-12-18T14:30:45-08:00"
}
```

This file is automatically created and updated. No manual configuration needed!

### Resetting Sync State

To reset and force a full scan next time:

```bash
# Method 1: Use force-full-scan flag
shark sync --force-full-scan

# Method 2: Delete last_sync_time from .sharkconfig.json
# (Edit the file and remove the last_sync_time field)
```

## How Files Are Filtered

### Modified Files

A file is considered "modified" and processed if:

1. File modification time (`mtime`) is after `last_sync_time`
2. File is new (not in database)
3. Clock skew tolerance applied (±60 seconds)

### Skipped Files

Files are skipped (not processed) if:

1. Modification time is before `last_sync_time`
2. File already exists in database with same content
3. File unchanged since last sync

## Conflict Detection

Incremental sync includes smart conflict detection:

### When Both File and Database Modified

If a file AND its database record were both modified since the last sync:

1. Conflicts are detected and reported
2. Resolution strategy is applied (`--strategy` flag)
3. Changes are synchronized based on strategy

Example output:
```
Sync Summary:
  Files scanned:      250
  Files filtered:     12 (changed)
  Files skipped:      238 (unchanged)
  Tasks updated:      12
  Conflicts resolved: 3

Conflicts:
  T-E04-F07-001:
    Field:    title
    Database: "Old Title"
    File:     "New Title"
```

### Resolution Strategies

```bash
# File wins (default)
shark sync --strategy=file-wins

# Database wins
shark sync --strategy=database-wins

# Newer wins (based on timestamps)
shark sync --strategy=newer-wins

# Manual resolution (interactive)
shark sync --strategy=manual
```

## Sync Report

The sync report shows incremental statistics:

```
Sync Summary:
  Files scanned:      500      # Total files found
  Files filtered:     25       # Files processed (changed)
  Files skipped:      475      # Files skipped (unchanged)
  Tasks imported:     2        # New tasks created
  Tasks updated:      23       # Existing tasks updated
  Conflicts resolved: 5        # Conflicts resolved
```

### Understanding the Report

- **Files scanned**: Total task files discovered
- **Files filtered**: Files processed (modified since last sync)
- **Files skipped**: Files skipped (unchanged since last sync)
- High skip count means incremental sync is working efficiently!

## Troubleshooting

### "All files being processed even though nothing changed"

**Cause**: `.sharkconfig.json` missing or `last_sync_time` not set.

**Solution**:
```bash
# Check if config exists
cat .sharkconfig.json

# Run sync again - it will set last_sync_time
shark sync
```

### "File changes not being detected"

**Cause**: File modification times not updating correctly.

**Solution**:
```bash
# Force full scan to resync
shark sync --force-full-scan
```

### "Clock skew warnings in output"

**Cause**: File modification time is in the future (clock drift).

**Impact**: File will still be processed (treated as modified).

**Solution**: Sync system clocks or ignore (warning is informational only).

## Best Practices

### 1. Commit .sharkconfig.json

**DO commit** `.sharkconfig.json` to version control:
```bash
git add .sharkconfig.json
git commit -m "Update shark sync state"
```

This allows team members to benefit from incremental sync after pulling changes.

### 2. Run Sync Regularly

```bash
# After modifying tasks
shark sync

# Before starting work
shark sync

# In CI/CD pipelines
shark sync --quiet
```

### 3. Use --force-full-scan Sparingly

Only use `--force-full-scan` when:
- Debugging sync issues
- After bulk file operations
- Verifying correctness

Don't use it for routine syncs - it defeats the performance benefits!

### 4. Monitor Sync Performance

Watch the sync report to ensure incremental sync is working:

```bash
shark sync

# Look for:
# - Files skipped > 0 (incremental working)
# - Files filtered < Files scanned (not full scan)
```

## FAQ

### Is incremental sync enabled by default?

Yes! No configuration needed. It activates automatically after the first sync.

### What happens on the first sync?

The first sync performs a full scan and creates `.sharkconfig.json` with `last_sync_time`.

### Can I disable incremental sync?

Use `--force-full-scan` for a one-time full scan. There's no permanent disable option (it's a performance optimization).

### What if I delete .sharkconfig.json?

Next sync will be a full scan and will recreate `.sharkconfig.json`.

### Does dry-run update last_sync_time?

No. Dry-run preview only, doesn't update any state.

### How does it handle concurrent edits?

Incremental sync detects when both file and database were modified since last sync and applies conflict resolution strategies.

## Technical Details

### Clock Skew Tolerance

Incremental sync applies a ±60 second tolerance window to handle:
- Clock drift between systems
- File system timestamp rounding
- Network time sync delays

Files modified within 60 seconds of `last_sync_time` are treated as "possibly modified during sync" and are processed.

### File Modification Detection

File changes detected using:
1. File system `mtime` (modification time)
2. Database `file_path` tracking
3. New file detection (not in database)

### Transaction Safety

Incremental sync maintains full transaction safety:
- All database updates in single transaction
- Rollback on error (state unchanged)
- `last_sync_time` only updated on successful sync

This ensures incremental state is always consistent.

## Related Commands

- `shark sync --help` - Full sync command reference
- `shark validate` - Validate task files without syncing
- `shark task list` - List all tasks in database
