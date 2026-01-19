# Architecture Design: Rejection Reason for Status Transitions

**Feature:** E07-F22
**Version:** 1.0
**Last Updated:** 2026-01-16

## Executive Summary

This document details the architecture for capturing and displaying rejection reasons when tasks move backward in workflow phases. The system leverages existing task_notes infrastructure with JSON metadata linking to task_history records, ensuring minimal schema changes while maintaining backward compatibility.

---

## System Overview

### Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Command Layer                         │
│  (shark task update --status=X --reason="...")              │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Workflow Validation Layer                       │
│  - Detect backward transitions (phase comparison)           │
│  - Require reason for backward transitions                  │
│  - Allow --force to bypass                                  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Repository Transaction Layer                    │
│  - UpdateStatus (existing method, enhanced)                 │
│  - CreateRejectionNote (new method)                         │
│  - GetRejectionHistory (new method)                         │
│  - LinkDocument (existing, enhanced)                        │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   tasks      │  │ task_notes   │  │task_history  │
│   table      │  │   table      │  │   table      │
│              │  │(+metadata)   │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
```

---

## Integration with Existing Systems

### 1. Task Notes System (E10-F01)

**Current Structure:**
```go
type TaskNote struct {
    ID        int64
    TaskID    int64
    NoteType  NoteType  // Enum: comment, decision, blocker, etc.
    Content   string
    CreatedBy *string
    CreatedAt time.Time
}
```

**Enhancement:**
- Add `metadata` column (TEXT, JSON string)
- Add new NoteType: `rejection`
- No breaking changes to existing notes

**Rejection Note Structure:**
```go
// TaskNote with metadata for rejection
{
    "id": 45,
    "task_id": 123,
    "note_type": "rejection",
    "content": "Missing error handling on line 67. Add null check.",
    "created_by": "reviewer-agent-001",
    "created_at": "2026-01-15T14:30:00Z",
    "metadata": {
        "history_id": 234,
        "from_status": "ready_for_code_review",
        "to_status": "in_development",
        "document_path": null
    }
}
```

### 2. Task History System (Existing)

**Current Behavior:**
- Automatically records all status transitions via trigger
- Provides audit trail with timestamps, agent, notes

**Integration:**
- Rejection notes link to history records via `metadata.history_id`
- No changes to task_history table structure
- Relationship is one-to-one: history record → rejection note

**Relationship Diagram:**
```
task_history.id (234) ──links to──> task_notes.metadata.history_id (234)
```

### 3. Task Documents Linking (E10-F03)

**Current Structure:**
```sql
CREATE TABLE task_documents (
    task_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at TIMESTAMP,
    PRIMARY KEY (task_id, document_id)
);
```

**Enhancement:**
- Use existing table, no schema changes
- Link documents via `--reason-doc` flag
- Document path stored in `task_notes.metadata.document_path`
- Bi-directional reference: metadata → document, task_documents → task

### 4. Workflow Configuration System

**Integration Point: Backward Transition Detection**

```go
// Use existing workflow config to detect backward transitions
func isBackwardTransition(currentStatus, newStatus string, workflow *config.WorkflowConfig) bool {
    phaseOrder := map[string]int{
        "planning": 1,
        "development": 2,
        "review": 3,
        "qa": 4,
        "approval": 5,
        "done": 6,
        "any": 0,  // Special statuses (blocked, on_hold) not considered backward
    }

    currentMeta := workflow.StatusMetadata[currentStatus]
    newMeta := workflow.StatusMetadata[newStatus]

    currentOrder := phaseOrder[currentMeta.Phase]
    newOrder := phaseOrder[newMeta.Phase]

    // Backward if: moving to lower phase AND not moving to "any" phase
    return newOrder < currentOrder && newOrder > 0
}
```

**Examples:**
- `ready_for_code_review` (review, 3) → `in_development` (development, 2): **Backward ✓**
- `in_qa` (qa, 4) → `in_development` (development, 2): **Backward ✓**
- `in_development` (development, 2) → `blocked` (any, 0): **NOT backward** (blocked is special)
- `in_development` (development, 2) → `ready_for_code_review` (review, 3): **Forward ✗**

---

## Component Interactions

### Scenario 1: Code Review Rejection (Happy Path)

```sequence
Reviewer Agent → CLI: shark task update E07-F01-003 --status=in_development --reason="..."
CLI → Workflow Validator: ValidateTransition(current, new, reason)
Workflow Validator → Workflow Config: isBackwardTransition?
Workflow Config → Workflow Validator: true (backward)
Workflow Validator → CLI: OK (reason provided)
CLI → TaskRepository: UpdateStatus(taskID, newStatus, agent, notes, reason)
TaskRepository → DB: BEGIN TRANSACTION
TaskRepository → DB: UPDATE tasks SET status=...
TaskRepository → DB: INSERT INTO task_history (...)
TaskRepository → DB: SELECT last_insert_id() [history_id=234]
TaskRepository → TaskNoteRepository: CreateRejectionNote(taskID, historyID=234, reason, fromStatus, toStatus)
TaskNoteRepository → DB: INSERT INTO task_notes (note_type='rejection', metadata=JSON)
TaskRepository → DB: COMMIT
TaskRepository → CLI: Success (updated task)
CLI → Reviewer Agent: Task E07-F01-003 rejected → in_development
```

### Scenario 2: Rejection Without Reason (Error Path)

```sequence
Reviewer Agent → CLI: shark task update E07-F01-003 --status=in_development
CLI → Workflow Validator: ValidateTransition(current, new, reason=nil)
Workflow Validator → Workflow Config: isBackwardTransition?
Workflow Config → Workflow Validator: true (backward)
Workflow Validator → CLI: ERROR: Reason required for backward transitions
CLI → Reviewer Agent: Error + helpful message with --reason example
```

### Scenario 3: Rejection with Document Link

```sequence
QA Agent → CLI: shark task update E07-F01-005 --status=in_development --reason="..." --reason-doc="docs/bugs/BUG-123.md"
CLI → File System: Validate file exists
File System → CLI: OK
CLI → TaskRepository: UpdateStatus(..., reason, reasonDoc)
TaskRepository → DB: BEGIN TRANSACTION
TaskRepository → DB: UPDATE tasks, INSERT history [historyID=241]
TaskRepository → TaskNoteRepository: CreateRejectionNote(..., documentPath="docs/bugs/BUG-123.md")
TaskNoteRepository → DB: INSERT task_notes (metadata includes document_path)
TaskRepository → DocumentRepository: LinkDocument(taskID, documentPath)
DocumentRepository → DB: INSERT INTO documents, task_documents
TaskRepository → DB: COMMIT
TaskRepository → CLI: Success
CLI → QA Agent: Task rejected with linked bug report
```

---

## Transaction Flow

### Critical Transaction Boundaries

**Status Update with Rejection (Atomic):**

```go
func (r *TaskRepository) UpdateStatus(
    ctx context.Context,
    taskID int64,
    newStatus string,
    agent *string,
    notes *string,
    reason *string,        // NEW: rejection reason
    reasonDoc *string,     // NEW: document path
    force bool,
) error {
    // Start transaction
    tx, err := r.db.BeginTxContext(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()  // Auto-rollback if not committed

    // 1. Get current task state
    var currentStatus string
    err = tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
    if err != nil {
        return fmt.Errorf("task not found or query failed: %w", err)
    }

    // 2. Validate transition (unless force=true)
    if !force {
        if isBackwardTransition(currentStatus, newStatus, r.workflow) && reason == nil {
            return fmt.Errorf("rejection reason required for backward transitions (use --force to bypass)")
        }
    }

    // 3. Update task status
    _, err = tx.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", newStatus, taskID)
    if err != nil {
        return fmt.Errorf("failed to update task status: %w", err)
    }

    // 4. Create history record
    result, err := tx.ExecContext(ctx,
        "INSERT INTO task_history (task_id, old_status, new_status, agent, notes) VALUES (?, ?, ?, ?, ?)",
        taskID, currentStatus, newStatus, agent, notes)
    if err != nil {
        return fmt.Errorf("failed to create history record: %w", err)
    }
    historyID, _ := result.LastInsertId()

    // 5. Create rejection note if reason provided AND backward transition
    if reason != nil && isBackwardTransition(currentStatus, newStatus, r.workflow) {
        noteRepo := NewTaskNoteRepository(r.db)
        _, err := noteRepo.CreateRejectionNoteInTx(ctx, tx, taskID, historyID, currentStatus, newStatus, *reason, agent, reasonDoc)
        if err != nil {
            return fmt.Errorf("failed to create rejection note: %w", err)
        }
    }

    // 6. Link document if provided
    if reasonDoc != nil {
        docRepo := NewDocumentRepository(r.db)
        if err := docRepo.LinkDocumentInTx(ctx, tx, taskID, *reasonDoc); err != nil {
            return fmt.Errorf("failed to link document: %w", err)
        }
    }

    // 7. Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

**Key Properties:**
- **Atomicity**: All operations succeed or all fail
- **Consistency**: Task status, history, rejection note, document link all consistent
- **Isolation**: No partial state visible to other transactions
- **Durability**: Once committed, changes persist

---

## Data Flow Diagrams

### Rejection Creation Flow

```
┌──────────────┐
│ CLI Command  │
│ (--reason)   │
└──────┬───────┘
       │
       ▼
┌──────────────────────┐
│ Workflow Validation  │
│ - Detect backward    │
│ - Require reason     │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ BEGIN TRANSACTION    │
└──────┬───────────────┘
       │
       ├─────► UPDATE tasks SET status
       │
       ├─────► INSERT task_history (get historyID)
       │
       ├─────► INSERT task_notes (metadata.history_id = historyID)
       │
       ├─────► INSERT documents + task_documents (if --reason-doc)
       │
       ▼
┌──────────────────────┐
│ COMMIT TRANSACTION   │
└──────────────────────┘
```

### Rejection Display Flow

```
┌──────────────┐
│ CLI Command  │
│ task get     │
└──────┬───────┘
       │
       ▼
┌──────────────────────┐
│ TaskRepository       │
│ GetByKey()           │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ TaskNoteRepository   │
│ GetRejectionHistory()│
└──────┬───────────────┘
       │
       ▼
┌────────────────────────────────────────┐
│ SQL Query (JOIN task_notes + history)  │
│ WHERE note_type='rejection'            │
│ ORDER BY created_at DESC               │
└──────┬─────────────────────────────────┘
       │
       ▼
┌──────────────────────┐
│ Format Display       │
│ - Terminal (colors)  │
│ - JSON (API)         │
└──────────────────────┘
```

---

## Workflow Configuration Integration

### Status Metadata Structure

```json
{
  "status_metadata": {
    "ready_for_code_review": {
      "phase": "review",
      "color": "magenta"
    },
    "in_development": {
      "phase": "development",
      "color": "yellow"
    },
    "blocked": {
      "phase": "any",
      "color": "red"
    }
  }
}
```

### Backward Detection Algorithm

```go
// Phase priority (lower = earlier in workflow)
var phasePriority = map[string]int{
    "planning": 1,
    "development": 2,
    "review": 3,
    "qa": 4,
    "approval": 5,
    "done": 6,
    "any": 0,  // Not part of linear flow
}

// Detect backward transition
func isBackwardTransition(fromStatus, toStatus string, wf *config.WorkflowConfig) bool {
    fromMeta := wf.StatusMetadata[fromStatus]
    toMeta := wf.StatusMetadata[toStatus]

    fromPhase := phasePriority[fromMeta.Phase]
    toPhase := phasePriority[toMeta.Phase]

    // Moving backward if:
    // 1. Target phase < source phase (numerically)
    // 2. Target phase is not "any" (special phase)
    return toPhase < fromPhase && toPhase > 0
}
```

**Edge Cases Handled:**
- Blocked/unblocked transitions: Not considered backward (phase="any")
- Same phase transitions: Not backward
- Forward then backward: Each checked independently

---

## Error Handling Strategy

### Validation Errors (User-Facing)

```go
// Clear, actionable error messages
var (
    ErrReasonRequired = errors.New("rejection reason required for backward transitions")
    ErrDocumentNotFound = errors.New("document file not found at specified path")
    ErrInvalidTransition = errors.New("invalid status transition")
)

// Enhanced error messages
func validateRejection(currentStatus, newStatus string, reason *string, workflow *config.WorkflowConfig) error {
    if isBackwardTransition(currentStatus, newStatus, workflow) && reason == nil {
        return fmt.Errorf(`%w

Reason is required when moving task backward in workflow.

Example:
  shark task update E07-F01-003 --status=%s --reason="Missing error handling on line 67"

Or use --force to bypass:
  shark task update E07-F01-003 --status=%s --force
`, ErrReasonRequired, newStatus, newStatus)
    }
    return nil
}
```

### Database Errors (Technical)

```go
// Wrap database errors with context
func (r *TaskRepository) UpdateStatus(...) error {
    tx, err := r.db.BeginTx(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin rejection transaction: %w", err)
    }
    defer tx.Rollback()

    // ... operations ...

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit rejection (status=%s, taskID=%d): %w", newStatus, taskID, err)
    }

    return nil
}
```

### Transaction Rollback Handling

```go
// Automatic rollback on error
defer func() {
    if err != nil {
        tx.Rollback()  // Rollback on any error
        log.Printf("Transaction rolled back: %v", err)
    }
}()
```

---

## Performance Considerations

### Query Performance

**Rejection History Query:**
```sql
-- Optimized query with JOIN and index usage
SELECT
    tn.id,
    tn.created_at,
    tn.content AS reason,
    tn.created_by AS rejected_by,
    json_extract(tn.metadata, '$.history_id') AS history_id,
    json_extract(tn.metadata, '$.from_status') AS from_status,
    json_extract(tn.metadata, '$.to_status') AS to_status,
    json_extract(tn.metadata, '$.document_path') AS document_path
FROM task_notes tn
WHERE tn.task_id = ?
  AND tn.note_type = 'rejection'
ORDER BY tn.created_at DESC
LIMIT 10;
```

**Indexes Used:**
- `idx_task_notes_task_type` (task_id, note_type) - Composite index
- `idx_task_notes_created_at` - For ORDER BY
- `idx_task_notes_type` - For note_type filter

**Expected Performance:**
- < 10ms for tasks with up to 10 rejections
- < 50ms for tasks with up to 100 rejections
- Index scan (not full table scan)

### Transaction Performance

**Critical Path:**
1. Begin transaction: ~1ms
2. SELECT current status: ~5ms (indexed)
3. UPDATE task status: ~5ms (primary key)
4. INSERT task_history: ~5ms
5. INSERT task_notes: ~5ms
6. Commit: ~10ms

**Total:** ~30-40ms per rejection

**Optimization:**
- Prepared statements reduce parsing overhead
- Single transaction reduces round trips
- Indexes minimize query time

---

## Backward Compatibility

### Schema Compatibility

**Adding metadata column:**
```sql
-- Migration: Add metadata column (nullable, default NULL)
ALTER TABLE task_notes ADD COLUMN metadata TEXT;

-- Index for JSON queries (optional, for future)
CREATE INDEX idx_task_notes_metadata ON task_notes(json_extract(metadata, '$.history_id'));
```

**Backward Compatibility:**
- Existing task_notes without metadata: NULL value (valid)
- Existing queries: Unaffected (metadata not required)
- New queries: Handle NULL gracefully

### API Compatibility

**Existing Commands:**
```bash
# Old command still works (no --reason flag)
shark task update E07-F01-003 --status=in_development

# Behavior:
# - Forward transitions: Work as before
# - Backward transitions: Error with helpful message about --reason
# - Use --force to skip reason (maintains old behavior)
```

**New Commands:**
```bash
# New flag added
shark task update E07-F01-003 --status=in_development --reason="..."

# New flag (optional)
shark task update E07-F01-003 --status=in_development --reason="..." --reason-doc="..."
```

### Data Migration

**No migration required:**
- Metadata column is nullable
- Existing notes are valid without metadata
- System works with mix of old and new notes

**Optional: Backfill script (for analytics):**
```go
// Backfill metadata.history_id for existing rejection notes
// This is OPTIONAL and only needed if you want to link old notes to history
func backfillRejectionMetadata(db *sql.DB) error {
    // Find rejection notes without metadata
    // Try to match to task_history based on timestamp proximity
    // Update metadata with matched history_id
}
```

---

## Security Considerations

### Input Sanitization

**Rejection Reason:**
```go
func sanitizeReason(reason string) (string, error) {
    // 1. Trim whitespace
    reason = strings.TrimSpace(reason)

    // 2. Validate length (prevent DoS)
    if len(reason) > 5000 {  // 5KB limit
        return "", errors.New("rejection reason too long (max 5000 characters)")
    }

    // 3. Check for null bytes (SQL injection prevention)
    if strings.Contains(reason, "\x00") {
        return "", errors.New("rejection reason contains invalid characters")
    }

    // 4. Escape special characters (stored as TEXT, no additional escaping needed)
    // SQLite parameterized queries handle this automatically

    return reason, nil
}
```

**Document Path:**
```go
func validateDocumentPath(path string, projectRoot string) error {
    // 1. Normalize path
    absPath := filepath.Join(projectRoot, path)
    cleanPath := filepath.Clean(absPath)

    // 2. Prevent directory traversal
    if !strings.HasPrefix(cleanPath, projectRoot) {
        return errors.New("document path must be within project root")
    }

    // 3. Validate file exists
    if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
        return fmt.Errorf("document not found: %s", path)
    }

    return nil
}
```

### SQL Injection Prevention

**Parameterized Queries:**
```go
// SAFE: Parameterized query
query := "INSERT INTO task_notes (task_id, content, metadata) VALUES (?, ?, ?)"
_, err := db.Exec(query, taskID, reason, metadataJSON)

// UNSAFE: String concatenation (DON'T DO THIS)
query := fmt.Sprintf("INSERT INTO task_notes (content) VALUES ('%s')", reason)  // ❌
```

**Metadata JSON:**
```go
// Encode metadata as JSON (safe)
metadata := map[string]interface{}{
    "history_id": historyID,
    "from_status": fromStatus,  // Already validated enum
    "to_status": toStatus,      // Already validated enum
}
metadataJSON, err := json.Marshal(metadata)
// Store as TEXT (no SQL injection risk)
```

---

## Monitoring & Observability

### Logging Strategy

```go
// Log rejection events for analytics
func (r *TaskRepository) UpdateStatus(...) error {
    // ... transaction logic ...

    if reason != nil && isBackwardTransition(...) {
        log.Printf("REJECTION: task=%d, from=%s, to=%s, agent=%v, reason_length=%d",
            taskID, currentStatus, newStatus, agent, len(*reason))
    }

    return nil
}
```

### Metrics to Track

**Operational Metrics:**
- Rejection rate: rejections / total status transitions
- Rejection reason length: avg, p50, p95, p99
- Document link rate: rejections with documents / total rejections
- Transaction time: p50, p95, p99

**Business Metrics:**
- Top rejection reasons (by keyword analysis)
- Most rejected tasks (by task ID)
- Rejection trends over time
- Agent rejection patterns

---

## Future Extensibility

### Planned Enhancements

**E07-F23: Rejection Notifications**
- Webhook integration: POST to external URL on rejection
- Agent notification: In-band message to developer agent
- Email notifications: For human reviewers

**E07-F24: Rejection Analytics Dashboard**
- Visualize rejection trends
- Identify quality bottlenecks
- Track improvement over time

**E07-F25: AI-Powered Reason Suggestions**
- ML model trained on historical rejection reasons
- Auto-suggest common issues based on task context
- Improve consistency of rejection feedback

**E07-F26: Rejection Reason Templates**
- Predefined templates for common issues
- Team-specific templates
- Template variables (file, line, method)

### Extension Points

**Custom Metadata Fields:**
```json
// Metadata structure is extensible
{
  "history_id": 234,
  "from_status": "ready_for_code_review",
  "to_status": "in_development",
  "document_path": "docs/bugs/BUG-123.md",

  // Future fields (backward compatible):
  "severity": "critical",
  "category": "error_handling",
  "estimated_fix_time": 60,  // minutes
  "related_rejections": [45, 52]
}
```

---

## Summary

This architecture achieves rejection reason tracking through:

1. **Minimal Schema Changes**: Single `metadata` column added to existing table
2. **Strong Integration**: Links rejection notes to task_history for complete audit trail
3. **Workflow-Aware**: Automatically detects backward transitions using phase comparison
4. **Transaction Safety**: Atomic operations ensure data consistency
5. **Backward Compatible**: Existing commands work unchanged, new flags are optional
6. **Extensible**: Metadata JSON supports future enhancements

**Key Design Decisions:**
- ✅ Use task_notes (not task_history.notes) for queryability
- ✅ JSON metadata for flexibility and extensibility
- ✅ Phase-based backward detection for workflow independence
- ✅ Require reason by default, allow --force bypass
- ✅ Link documents via existing task_documents table

**Non-Goals (Out of Scope):**
- ❌ Rejection notifications (E07-F23)
- ❌ Rejection analytics (E07-F24)
- ❌ AI-powered suggestions (E07-F25)
- ❌ Reason templates (E07-F26)
