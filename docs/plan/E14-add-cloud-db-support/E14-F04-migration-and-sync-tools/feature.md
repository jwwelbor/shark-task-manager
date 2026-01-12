# Feature: Migration and Sync Tools

**Feature Key:** E14-F04
**Epic:** E14 - Cloud Database Support
**Status:** Draft
**Execution Order:** 4

## Overview

Provide robust data migration and synchronization tools for moving task data between local SQLite and Turso cloud databases, with conflict detection and resolution capabilities.

## Goal

###Problem

Users need to safely migrate data between local and cloud databases:
- **Initial migration:** Export existing local data to newly created cloud database
- **Multi-machine setup:** Import cloud data to new workstation
- **Bidirectional sync:** Merge changes when working offline on multiple machines
- **Conflict resolution:** Handle simultaneous edits to same task/epic/feature
- **Backup/restore:** Export cloud data to local for disaster recovery

Without proper tools, users risk:
- Data loss during migration
- Duplicate records (same task created twice)
- Silent data overwrites (no conflict detection)
- Inability to recover from cloud failures

### Solution

Implement comprehensive migration and sync utilities:
- **Export:** `shark cloud export` - Push local database to cloud
- **Import:** `shark cloud import` - Pull cloud database to local
- **Sync:** `shark cloud sync` - Bidirectional merge with conflict detection
- **Conflict resolution:** Last-write-wins with warnings for user intervention
- **Backup:** `shark backup export/import` - SQL dump for disaster recovery

### Impact

**For Users:**
- Safe migration (zero data loss)
- Multi-machine workflows work reliably
- Visibility into conflicts (not silent overwrites)
- Disaster recovery capability

**For Support:**
- Fewer "lost my tasks" support requests
- Clear sync status and logs for debugging
- Rollback capability if sync fails

## User Stories

### Must-Have Stories

**Story 1:** As a user, I want to export my local database to cloud so that I can start using cloud sync without losing existing tasks.

**Acceptance Criteria:**
- [ ] All tasks, epics, features exported to cloud
- [ ] Task history preserved
- [ ] Related documents links preserved
- [ ] Progress indicator for large databases
- [ ] Rollback on failure (atomic operation)
- [ ] Verification step shows what was exported

**Story 2:** As a user, I want to import cloud database to a new workstation so that I can see my existing tasks on that machine.

**Acceptance Criteria:**
- [ ] Downloads all cloud data to local
- [ ] Option to merge with existing local data or overwrite
- [ ] Conflict warnings if local has uncommitted changes
- [ ] Progress indicator
- [ ] Verification summary

**Story 3:** As a multi-machine user, I want bidirectional sync with conflict detection so that changes from all machines are merged safely.

**Acceptance Criteria:**
- [ ] Detects conflicts (same task edited on 2+ machines while offline)
- [ ] Applies last-write-wins with timestamp comparison
- [ ] Warns user about conflicts (doesn't silently overwrite)
- [ ] Syncs in both directions (push + pull)
- [ ] Works incrementally (only changed records)

**Story 4:** As a user, I want to export cloud database to local SQLite so that I can have a backup or switch back to local-only mode.

**Acceptance Criteria:**
- [ ] Exports complete cloud database to local .db file
- [ ] Preserves all relationships (foreign keys)
- [ ] Includes metadata (created_at, updated_at)
- [ ] Can be used as standalone local database

**Story 5:** As a user, I want clear resolution strategies for conflicts so that I understand what happens when two machines edit the same task.

**Acceptance Criteria:**
- [ ] Default: Last-write-wins (newer timestamp wins)
- [ ] User notified of conflicts with details (what changed, when, where)
- [ ] Option to manually resolve conflicts
- [ ] Conflict log for audit trail

## Requirements

### Functional Requirements

**REQ-F-001: Export Local to Cloud**
- **Description:** Push all local database records to cloud
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Exports epics, features, tasks, task_history
  - [ ] Atomic operation (all-or-nothing)
  - [ ] Detects duplicates (by key) and skips or updates
  - [ ] Progress indicator with ETA
  - [ ] Summary: X records exported, Y skipped (duplicates)

**REQ-F-002: Import Cloud to Local**
- **Description:** Pull all cloud database records to local
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Downloads epics, features, tasks, task_history
  - [ ] Option: `--merge` or `--overwrite`
  - [ ] Merge mode: detects conflicts, applies resolution strategy
  - [ ] Overwrite mode: warning + confirmation prompt
  - [ ] Summary: X records imported, Y conflicts resolved

**REQ-F-003: Bidirectional Sync**
- **Description:** Merge changes from both local and cloud
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Compares timestamps (updated_at) to detect changes
  - [ ] Pushes local changes to cloud
  - [ ] Pulls cloud changes to local
  - [ ] Detects conflicts (both modified since last sync)
  - [ ] Applies conflict resolution strategy
  - [ ] Incremental (only changed records, not full export/import)

**REQ-F-004: Conflict Detection**
- **Description:** Identify when same record modified on multiple machines
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Conflict = record modified both locally and in cloud since last sync
  - [ ] Tracks last_sync_timestamp per machine
  - [ ] Detects conflicts on: title, status, description, priority
  - [ ] Reports conflicts to user with details

**REQ-F-005: Conflict Resolution Strategies**
- **Description:** Automated strategies for resolving conflicts
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Last-write-wins (default): Use record with newer updated_at
  - [ ] Cloud-wins: Always prefer cloud version
  - [ ] Local-wins: Always prefer local version
  - [ ] Manual: Prompt user to choose (future enhancement)
  - [ ] Strategy configurable in .sharkconfig.json

**REQ-F-006: Backup and Restore**
- **Description:** SQL dump format for disaster recovery
- **Priority:** Should-Have
- **Acceptance Criteria:**
  - [ ] `shark backup export --output=backup.sql` - SQLite dump
  - [ ] `shark backup import --input=backup.sql` - Restore from dump
  - [ ] Works with both local and cloud backends
  - [ ] Includes schema + data

### Non-Functional Requirements

**Data Integrity:**
- **REQ-NF-001:** Zero data loss during migration (100% fidelity)
- **Measurement:** Automated tests with 10,000+ records
- **Target:** 0 lost records, 0 corrupted relationships
- **Justification:** Users must trust migration tools

**Performance:**
- **REQ-NF-002:** Export/import 10,000 tasks in < 30 seconds
- **Measurement:** Benchmark on cloud connection (50ms latency)
- **Target:** ~300 tasks/second throughput
- **Justification:** Large databases should sync quickly

**Reliability:**
- **REQ-NF-003:** Failed syncs rollback completely (atomic)
- **Measurement:** Test failure scenarios (network errors, invalid data)
- **Target:** 100% rollback success, no partial states
- **Justification:** Better to fail cleanly than corrupt data

## Technical Design

### Sync Algorithm (Bidirectional)

```
1. Fetch last_sync_timestamp from local metadata
2. Query local: SELECT * FROM tasks WHERE updated_at > last_sync_timestamp
3. Query cloud: SELECT * FROM tasks WHERE updated_at > last_sync_timestamp
4. Merge results:
   - Local-only changes → Push to cloud
   - Cloud-only changes → Pull to local
   - Both changed (conflict) → Apply resolution strategy
5. Update last_sync_timestamp to current time
```

### Conflict Detection Logic

```go
type ConflictRecord struct {
    Key          string
    LocalValue   interface{}
    CloudValue   interface{}
    LocalTime    time.Time
    CloudTime    time.Time
    ConflictType string  // "title", "status", "description", etc.
}

func DetectConflicts(localRecord, cloudRecord *Task, lastSync time.Time) []ConflictRecord {
    conflicts := []ConflictRecord{}

    // Both modified since last sync?
    if localRecord.UpdatedAt.After(lastSync) && cloudRecord.UpdatedAt.After(lastSync) {
        // Compare fields
        if localRecord.Title != cloudRecord.Title {
            conflicts = append(conflicts, ConflictRecord{
                Key:        localRecord.Key,
                LocalValue: localRecord.Title,
                CloudValue: cloudRecord.Title,
                LocalTime:  localRecord.UpdatedAt,
                CloudTime:  cloudRecord.UpdatedAt,
                ConflictType: "title",
            })
        }
        // Repeat for status, description, priority, etc.
    }

    return conflicts
}
```

### Conflict Resolution: Last-Write-Wins

```go
func ResolveConflict(conflict ConflictRecord, strategy string) interface{} {
    switch strategy {
    case "last-write-wins":
        if conflict.LocalTime.After(conflict.CloudTime) {
            return conflict.LocalValue
        }
        return conflict.CloudValue

    case "cloud-wins":
        return conflict.CloudValue

    case "local-wins":
        return conflict.LocalValue

    default:
        // Default to last-write-wins
        return ResolveConflict(conflict, "last-write-wins")
    }
}
```

### Sync Metadata Storage

Track last sync time per machine in local metadata:

```sql
CREATE TABLE sync_metadata (
    machine_id TEXT PRIMARY KEY,  -- UUID generated per machine
    last_sync_timestamp DATETIME,
    last_sync_direction TEXT,     -- "push", "pull", "bidirectional"
    conflicts_count INTEGER,
    created_at DATETIME,
    updated_at DATETIME
);
```

### Export/Import Flow

**Export (Local → Cloud):**
```
1. Begin transaction on cloud database
2. For each table (epics, features, tasks, task_history):
   a. SELECT all records from local
   b. INSERT INTO cloud (or UPDATE if exists)
   c. Track progress (X of Y)
3. Commit transaction
4. Update last_sync_timestamp
5. Show summary
```

**Import (Cloud → Local):**
```
1. Begin transaction on local database
2. For each table:
   a. SELECT all records from cloud
   b. INSERT INTO local (or UPDATE if exists based on --merge flag)
   c. Track progress
3. Commit transaction
4. Update last_sync_timestamp
5. Show summary
```

## Tasks

- **T-E14-F04-003:** Implement bidirectional sync with conflict detection (Priority: 9)
- **T-E14-F04-001:** Implement export local database to cloud functionality (Priority: 8)
- **T-E14-F04-002:** Implement import cloud database to local functionality (Priority: 8)
- **T-E14-F04-004:** Implement conflict resolution strategies (Priority: 7)
- **T-E14-F04-005:** Add backup and restore utilities (Priority: 6)

## Dependencies

- **F01:** Database abstraction layer (for multi-backend support)
- **F02:** Turso integration (for cloud connection)
- **F03:** Cloud CLI commands (wraps these tools in user-friendly commands)

## Success Metrics

**Data Integrity:**
- [ ] 0 data loss incidents in beta testing (100 users, 30 days)
- [ ] 0 corrupted foreign key relationships
- [ ] 100% of conflicts logged and reported

**Performance:**
- [ ] 10,000 tasks export in < 30 seconds
- [ ] 10,000 tasks import in < 30 seconds
- [ ] Bidirectional sync < 10 seconds for 100 changed tasks

**Usability:**
- [ ] 90% of users successfully migrate on first try
- [ ] Conflict warnings understood (user survey)
- [ ] Zero "lost my tasks" support requests

## Out of Scope

### Explicitly Excluded

1. **Manual Conflict Resolution UI**
   - **Why:** Last-write-wins sufficient for task management
   - **Future:** Add if users report frequent conflicts
   - **Workaround:** User manually updates after sync

2. **Partial Sync (Selective Tables)**
   - **Why:** All tables should sync together (referential integrity)
   - **Future:** Could add if users request it
   - **Workaround:** Use full sync

3. **Real-Time Sync (Push on Every Change)**
   - **Why:** Batched sync every 30 seconds is sufficient
   - **Future:** Could add WebSocket-based real-time sync
   - **Workaround:** Use `shark cloud sync` for immediate sync

4. **Multi-Way Merge (3+ Machines Editing Same Task)**
   - **Why:** Rare edge case, complex to implement
   - **Future:** Track edit history for better merge
   - **Workaround:** Last-write-wins handles this, may lose edits

## Security Considerations

**Data in Transit:**
- All sync operations use HTTPS/TLS 1.3
- Auth tokens required for cloud access
- No data transmitted in plain text

**Data Validation:**
- Verify foreign key integrity before import
- Validate schema version compatibility
- Reject imports with mismatched schema

**Rollback Safety:**
- All operations wrapped in transactions
- Rollback on any error (atomic)
- Backup created before destructive operations

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Data loss during sync | Atomic transactions; rollback on failure; automated backups |
| Conflicts silently overwrite | Detect conflicts; log warnings; notify user |
| Network failure mid-sync | Transaction rollback; resume from checkpoint |
| Schema version mismatch | Validate schema before import; migration scripts |
| Large database performance | Incremental sync; progress indicators; batch operations |

---

*Last Updated:* 2026-01-04
*Dependencies:* F01, F02, F03
