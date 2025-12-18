# E06-F04: Incremental Sync Engine

## Overview

This feature adds intelligent change detection to the shark sync system, reducing sync times from 5-10 seconds (full scan) to <2 seconds (incremental) for typical development sessions where only 3-7 files have changed.

## Documentation

### POC Architecture Documentation

This feature has **minimal, essential architecture docs** appropriate for POC-level development:

1. **[02-architecture.md](./02-architecture.md)** - High-level component design and data flow
   - System overview and component architecture
   - Incremental sync algorithm
   - Performance optimization strategies
   - Integration with existing E04-F07 sync engine

2. **[04-backend-design.md](./04-backend-design.md)** - Detailed implementation specifications
   - Data structures and function signatures
   - Core algorithms with code examples
   - CLI integration details
   - Implementation checklist

### Why Only Two Docs?

For a POC, we focus on **practical implementation guidance** rather than comprehensive documentation:

- ❌ **NOT NEEDED**: Frontend design (CLI-only feature)
- ❌ **NOT NEEDED**: Database design (uses existing schema with `updated_at` timestamps)
- ❌ **NOT NEEDED**: Security design (extends existing security model)
- ❌ **NOT NEEDED**: Performance design (covered in architecture doc)
- ✅ **ESSENTIAL**: Architecture overview (understand the design)
- ✅ **ESSENTIAL**: Backend implementation specs (build the feature)

## Key Design Decisions

### 1. Extend, Don't Rewrite

This feature **extends** the existing E04-F07 sync engine rather than creating a new one. This means:

- Reuse proven transaction management
- Reuse file scanning and parsing
- Reuse conflict resolution strategies
- Add incremental filtering layer on top

### 2. Simple Timestamp Comparison

The POC uses **filesystem mtime vs. last_sync_time** comparison:

- ✅ Fast (O(n) with ~1ns per comparison)
- ✅ No Git dependency
- ✅ Works on all filesystems
- ⚠️ Limitation: Misses content changes with old mtime (rare edge case)

Future enhancement: Git-based change detection (E06-F04 Could-Have story)

### 3. Config-Based State

Store `last_sync_time` in `.sharkconfig.json`:

```json
{
  "default_epic": null,
  "default_agent": null,
  "color_enabled": true,
  "json_output": false,
  "last_sync_time": "2025-12-17T14:30:45-08:00"
}
```

- ✅ No separate state file
- ✅ Human-readable
- ✅ Atomic updates (temp file + rename)
- ✅ Backward compatible (null = first sync)

### 4. Fail-Safe Defaults

If `last_sync_time` is missing or invalid:

- Automatically perform full scan
- Log warning (not error)
- Update config after successful sync
- Next sync is incremental

### 5. Transaction Safety

Update order is critical:

1. Commit database transaction
2. **THEN** update `last_sync_time` in config

If config write fails:
- Database is still committed (data safe)
- Log warning (next sync may reprocess files)
- Don't fail the entire sync operation

## Implementation Strategy

### Phase 1: Config Management (Lowest Risk)

Extend configuration system to support `last_sync_time`:

- Add field to `ConfigDefaults` struct
- Implement load/save with validation
- Unit tests for config operations

**Risk**: Low (isolated change)

### Phase 2: Filtering Logic (Core Feature)

Add file filtering based on modification time:

- Implement `filterChangedFiles()` algorithm
- Add clock skew detection
- Unit tests for filtering

**Risk**: Low (pure algorithm, no side effects)

### Phase 3: Sync Engine Integration (Medium Risk)

Integrate filtering into existing sync flow:

- Conditionally apply filtering (if --incremental)
- Update `last_sync_time` after commit
- Integration tests

**Risk**: Medium (modifies critical sync path)

### Phase 4: Conflict Detection Enhancement (Low Risk)

Add timestamp-based conflict detection:

- Only report conflicts if both file AND database modified
- Extend `Conflict` struct with timestamps
- Unit tests

**Risk**: Low (improves existing logic)

### Phase 5: CLI Integration (Low Risk)

Add CLI flags and help text:

- `--incremental` flag
- `--force-full-scan` flag
- Update sync report display

**Risk**: Low (user interface only)

## Performance Impact

### Best Case (5 Files Changed)

- **Before**: 8 seconds (full scan of 287 files)
- **After**: 1.8 seconds (incremental scan, 5 files processed)
- **Improvement**: 4.4x faster

### Worst Case (All Files Changed)

- **Before**: 8 seconds
- **After**: 8 seconds (same as full scan)
- **Regression**: None

### Overhead (Filtering)

- **Mtime comparison**: <100ms for 500 files
- **Impact**: Negligible

## Backward Compatibility

### Existing Projects

- Work without modification
- First `shark sync --incremental` performs full scan
- Subsequent syncs are incremental

### Existing Workflows

- `shark sync` (without --incremental) → full scan (E04-F07 behavior)
- Can mix full and incremental syncs
- Gradual adoption

## Testing Strategy

### Unit Tests

- Config load/save with timestamps
- File filtering algorithm
- Timestamp conflict detection
- Clock skew handling

### Integration Tests

- First sync (no last_sync_time)
- Incremental sync (some files changed)
- Conflict resolution with timestamps
- Force full scan

### Performance Tests

- 500 files, 5 changed → <2 seconds
- 500 files, all changed → <30 seconds

## Next Steps

1. **Review** this architecture documentation
2. **Generate tasks** using `/task` command
3. **Implement** following the phase order above
4. **Test** against acceptance criteria in PRD
5. **Deploy** to POC environment

## Related Documentation

- **PRD**: [prd.md](./prd.md) - Product requirements
- **E04-F07 Sync**: [../E04-task-mgmt-cli-core/E04-F07-initialization-sync/](../../E04-task-mgmt-cli-core/E04-F07-initialization-sync/) - Base sync system
- **System Design**: [../../architecture/SYSTEM_DESIGN.md](../../architecture/SYSTEM_DESIGN.md) - Overall system architecture
