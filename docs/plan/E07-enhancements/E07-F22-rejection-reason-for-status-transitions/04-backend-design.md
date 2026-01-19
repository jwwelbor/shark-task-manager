# Backend API/Repository Design: Rejection Reason for Status Transitions

**Feature:** E07-F22
**Version:** 1.0
**Last Updated:** 2026-01-16

## Executive Summary

This document specifies the backend repository methods, API contracts, and error handling patterns for rejection reason functionality. All methods follow Shark's established patterns: context-first parameters, error wrapping, transaction safety, and workflow-aware validation.

---

## Repository Architecture

### Component Overview

```
┌──────────────────────────────────────────────────────────┐
│           Repository Layer Components                     │
├──────────────────────────────────────────────────────────┤
│  TaskRepository (Enhanced)                               │
│    - UpdateStatus() [modified to accept reason params]  │
│    - isBackwardTransition() [new helper]                │
│                                                           │
│  TaskNoteRepository (New Methods)                        │
│    - CreateRejectionNote()                               │
│    - CreateRejectionNoteInTx()                           │
│    - GetRejectionHistory()                               │
│    - GetLatestRejection()                                │
│                                                           │
│  DocumentRepository (Existing, Reused)                   │
│    - LinkDocument()                                      │
│    - LinkDocumentInTx()                                  │
└──────────────────────────────────────────────────────────┘
```

---

## TaskRepository Enhancements

### 1. UpdateStatus (Modified Signature)

**File:** `internal/repository/task_repository.go`

**Current Signature:**
```go
func (r *TaskRepository) UpdateStatus(
    ctx context.Context,
    taskID int64,
    newStatus models.TaskStatus,
    agent *string,
    notes *string,
) error
```

**New Signature:**
```go
func (r *TaskRepository) UpdateStatus(
    ctx context.Context,
    taskID int64,
    newStatus models.TaskStatus,
    agent *string,
    notes *string,
    reason *string,        // NEW: rejection reason
    reasonDoc *string,     // NEW: document path
) error
```

**Implementation:**
```go
// UpdateStatus atomically updates task status with optional rejection reason
func (r *TaskRepository) UpdateStatus(
    ctx context.Context,
    taskID int64,
    newStatus models.TaskStatus,
    agent *string,
    notes *string,
    reason *string,
    reasonDoc *string,
) error {
    // Start transaction
    tx, err := r.db.BeginTxContext(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Get current task state
    var currentStatus string
    var startedAt, completedAt, blockedAt sql.NullTime
    err = tx.QueryRowContext(ctx,
        "SELECT status, started_at, completed_at, blocked_at FROM tasks WHERE id = ?",
        taskID).Scan(&currentStatus, &startedAt, &completedAt, &blockedAt)
    if err == sql.ErrNoRows {
        return fmt.Errorf("task not found with id %d", taskID)
    }
    if err != nil {
        return fmt.Errorf("failed to get current task status: %w", err)
    }

    // Validate status enum
    if !r.isValidStatusEnum(newStatus) {
        return fmt.Errorf("invalid status: %s", newStatus)
    }

    // Check backward transition
    isBackward := r.isBackwardTransition(models.TaskStatus(currentStatus), newStatus)

    // Validate reason requirement
    if isBackward && reason == nil {
        return r.buildReasonRequiredError(currentStatus, string(newStatus))
    }

    // Validate document exists if provided
    if reasonDoc != nil {
        if err := validateDocumentPath(*reasonDoc); err != nil {
            return fmt.Errorf("invalid document path: %w", err)
        }
    }

    // Update task status and timestamps
    now := time.Now()
    query := "UPDATE tasks SET status = ?"
    args := []interface{}{newStatus}

    // Set appropriate timestamp
    if newStatus == models.TaskStatusInProgress && !startedAt.Valid {
        query += ", started_at = ?"
        args = append(args, now)
    } else if newStatus == models.TaskStatusCompleted && !completedAt.Valid {
        query += ", completed_at = ?"
        args = append(args, now)
    } else if newStatus == models.TaskStatusBlocked && !blockedAt.Valid {
        query += ", blocked_at = ?"
        args = append(args, now)
    }

    query += " WHERE id = ?"
    args = append(args, taskID)

    _, err = tx.ExecContext(ctx, query, args...)
    if err != nil {
        return fmt.Errorf("failed to update task status: %w", err)
    }

    // Create history record
    historyQuery := `
        INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced)
        VALUES (?, ?, ?, ?, ?, false)
    `
    result, err := tx.ExecContext(ctx, historyQuery, taskID, currentStatus, newStatus, agent, notes)
    if err != nil {
        return fmt.Errorf("failed to create history record: %w", err)
    }

    // Get history ID
    historyID, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get history ID: %w", err)
    }

    // Create rejection note if reason provided AND backward transition
    if reason != nil && isBackward {
        noteRepo := NewTaskNoteRepository(r.db)
        _, err := noteRepo.CreateRejectionNoteInTx(
            ctx, tx, taskID, historyID, currentStatus, string(newStatus), *reason, agent, reasonDoc,
        )
        if err != nil {
            return fmt.Errorf("failed to create rejection note: %w", err)
        }

        // Link document if provided
        if reasonDoc != nil {
            docRepo := NewDocumentRepository(r.db)
            if err := docRepo.LinkDocumentInTx(ctx, tx, taskID, *reasonDoc); err != nil {
                return fmt.Errorf("failed to link document: %w", err)
            }
        }
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

**Error Handling:**
```go
func (r *TaskRepository) buildReasonRequiredError(fromStatus, toStatus string) error {
    return fmt.Errorf(`rejection reason required for backward transitions

Current status: %s
New status: %s

Example:
  shark task update <task-key> --status=%s --reason="Missing error handling on line 67"

Or use --force to bypass:
  shark task update <task-key> --status=%s --force

Valid transitions from %s:
%s
`, fromStatus, toStatus, toStatus, toStatus, fromStatus, r.getValidTransitionsHelp(fromStatus))
}

func (r *TaskRepository) getValidTransitionsHelp(fromStatus string) string {
    if r.workflow == nil {
        return "  (workflow not configured)"
    }
    validStatuses, ok := r.workflow.StatusFlow[fromStatus]
    if !ok {
        return "  (no valid transitions)"
    }
    var lines []string
    for _, status := range validStatuses {
        lines = append(lines, fmt.Sprintf("  - %s", status))
    }
    return strings.Join(lines, "\n")
}
```

---

### 2. isBackwardTransition (New Helper Method)

**Signature:**
```go
func (r *TaskRepository) isBackwardTransition(
    currentStatus models.TaskStatus,
    newStatus models.TaskStatus,
) bool
```

**Implementation:**
```go
// isBackwardTransition checks if a status transition moves backward in workflow phases
func (r *TaskRepository) isBackwardTransition(
    currentStatus models.TaskStatus,
    newStatus models.TaskStatus,
) bool {
    // Phase priority (lower = earlier in workflow)
    phasePriority := map[string]int{
        "planning":    1,
        "development": 2,
        "review":      3,
        "qa":          4,
        "approval":    5,
        "done":        6,
        "any":         0, // Special phase (blocked, on_hold)
    }

    // Get phase for each status
    currentMeta, foundCurrent := r.workflow.StatusMetadata[string(currentStatus)]
    newMeta, foundNew := r.workflow.StatusMetadata[string(newStatus)]

    // If status not in workflow, assume not backward (fallback)
    if !foundCurrent || !foundNew {
        return false
    }

    // Get priority for each phase
    currentPhase := phasePriority[currentMeta.Phase]
    newPhase := phasePriority[newMeta.Phase]

    // Backward if:
    // 1. New phase < current phase (numerically)
    // 2. New phase is not "any" (special statuses like blocked)
    return newPhase < currentPhase && newPhase > 0
}
```

**Test Cases:**
```go
func TestIsBackwardTransition(t *testing.T) {
    tests := []struct {
        name           string
        currentStatus  models.TaskStatus
        newStatus      models.TaskStatus
        wantBackward   bool
    }{
        {
            name:          "code review to development (backward)",
            currentStatus: "ready_for_code_review",
            newStatus:     "in_development",
            wantBackward:  true,
        },
        {
            name:          "qa to development (backward)",
            currentStatus: "in_qa",
            newStatus:     "in_development",
            wantBackward:  true,
        },
        {
            name:          "development to review (forward)",
            currentStatus: "in_development",
            newStatus:     "ready_for_code_review",
            wantBackward:  false,
        },
        {
            name:          "development to blocked (not backward, special phase)",
            currentStatus: "in_development",
            newStatus:     "blocked",
            wantBackward:  false,
        },
        {
            name:          "blocked to development (not backward, from special phase)",
            currentStatus: "blocked",
            newStatus:     "in_development",
            wantBackward:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := setupTestRepository(t)
            got := repo.isBackwardTransition(tt.currentStatus, tt.newStatus)
            if got != tt.wantBackward {
                t.Errorf("isBackwardTransition() = %v, want %v", got, tt.wantBackward)
            }
        })
    }
}
```

---

## TaskNoteRepository New Methods

**File:** `internal/repository/task_note_repository.go` (create if doesn't exist)

### Repository Structure

```go
package repository

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskNoteRepository handles CRUD operations for task notes
type TaskNoteRepository struct {
    db *DB
}

// NewTaskNoteRepository creates a new TaskNoteRepository
func NewTaskNoteRepository(db *DB) *TaskNoteRepository {
    return &TaskNoteRepository{db: db}
}
```

### 1. CreateRejectionNote

**Signature:**
```go
func (r *TaskNoteRepository) CreateRejectionNote(
    ctx context.Context,
    taskID int64,
    historyID int64,
    fromStatus string,
    toStatus string,
    reason string,
    rejectedBy *string,
    documentPath *string,
) (*models.TaskNote, error)
```

**Implementation:**
```go
// CreateRejectionNote creates a rejection note with metadata
func (r *TaskNoteRepository) CreateRejectionNote(
    ctx context.Context,
    taskID int64,
    historyID int64,
    fromStatus string,
    toStatus string,
    reason string,
    rejectedBy *string,
    documentPath *string,
) (*models.TaskNote, error) {
    // Start new transaction
    tx, err := r.db.BeginTxContext(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Create rejection note within transaction
    note, err := r.CreateRejectionNoteInTx(ctx, tx, taskID, historyID, fromStatus, toStatus, reason, rejectedBy, documentPath)
    if err != nil {
        return nil, err
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }

    return note, nil
}
```

### 2. CreateRejectionNoteInTx (Transaction-Safe)

**Signature:**
```go
func (r *TaskNoteRepository) CreateRejectionNoteInTx(
    ctx context.Context,
    tx *sql.Tx,
    taskID int64,
    historyID int64,
    fromStatus string,
    toStatus string,
    reason string,
    rejectedBy *string,
    documentPath *string,
) (*models.TaskNote, error)
```

**Implementation:**
```go
// CreateRejectionNoteInTx creates a rejection note within an existing transaction
func (r *TaskNoteRepository) CreateRejectionNoteInTx(
    ctx context.Context,
    tx *sql.Tx,
    taskID int64,
    historyID int64,
    fromStatus string,
    toStatus string,
    reason string,
    rejectedBy *string,
    documentPath *string,
) (*models.TaskNote, error) {
    // Validate inputs
    if taskID == 0 {
        return nil, fmt.Errorf("task_id is required")
    }
    if historyID == 0 {
        return nil, fmt.Errorf("history_id is required")
    }
    if fromStatus == "" {
        return nil, fmt.Errorf("from_status is required")
    }
    if toStatus == "" {
        return nil, fmt.Errorf("to_status is required")
    }
    if reason == "" {
        return nil, fmt.Errorf("reason cannot be empty")
    }

    // Sanitize reason
    reason, err := sanitizeReason(reason)
    if err != nil {
        return nil, fmt.Errorf("invalid reason: %w", err)
    }

    // Build metadata
    metadata := map[string]interface{}{
        "history_id":    historyID,
        "from_status":   fromStatus,
        "to_status":     toStatus,
        "document_path": documentPath,
    }
    metadataJSON, err := json.Marshal(metadata)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal metadata: %w", err)
    }

    // Insert rejection note
    query := `
        INSERT INTO task_notes (task_id, note_type, content, created_by, metadata)
        VALUES (?, 'rejection', ?, ?, ?)
    `
    result, err := tx.ExecContext(ctx, query, taskID, reason, rejectedBy, string(metadataJSON))
    if err != nil {
        return nil, fmt.Errorf("failed to insert rejection note: %w", err)
    }

    // Get inserted ID
    noteID, err := result.LastInsertId()
    if err != nil {
        return nil, fmt.Errorf("failed to get note ID: %w", err)
    }

    // Build TaskNote object
    note := &models.TaskNote{
        ID:        noteID,
        TaskID:    taskID,
        NoteType:  models.NoteType("rejection"),
        Content:   reason,
        CreatedBy: rejectedBy,
        CreatedAt: time.Now(),
        // Metadata field added to models.TaskNote:
        // Metadata: string(metadataJSON),
    }

    return note, nil
}
```

**Helper Functions:**
```go
// sanitizeReason validates and sanitizes rejection reason
func sanitizeReason(reason string) (string, error) {
    // Trim whitespace
    reason = strings.TrimSpace(reason)

    // Validate length
    if len(reason) == 0 {
        return "", errors.New("reason cannot be empty")
    }
    if len(reason) > 5000 {
        return "", errors.New("reason too long (max 5000 characters)")
    }

    // Check for null bytes (prevent SQL injection)
    if strings.Contains(reason, "\x00") {
        return "", errors.New("reason contains invalid characters")
    }

    return reason, nil
}
```

---

### 3. GetRejectionHistory

**Signature:**
```go
func (r *TaskNoteRepository) GetRejectionHistory(
    ctx context.Context,
    taskID int64,
) ([]*RejectionHistoryEntry, error)
```

**Return Type:**
```go
// RejectionHistoryEntry represents a single rejection in task history
type RejectionHistoryEntry struct {
    ID           int64     `json:"id"`
    Timestamp    time.Time `json:"timestamp"`
    Reason       string    `json:"reason"`
    RejectedBy   *string   `json:"rejected_by,omitempty"`
    HistoryID    int64     `json:"history_id"`
    FromStatus   string    `json:"from_status"`
    ToStatus     string    `json:"to_status"`
    DocumentPath *string   `json:"document_path,omitempty"`
}
```

**Implementation:**
```go
// GetRejectionHistory retrieves all rejection notes for a task
func (r *TaskNoteRepository) GetRejectionHistory(
    ctx context.Context,
    taskID int64,
) ([]*RejectionHistoryEntry, error) {
    query := `
        SELECT
            tn.id,
            tn.created_at,
            tn.content,
            tn.created_by,
            json_extract(tn.metadata, '$.history_id'),
            json_extract(tn.metadata, '$.from_status'),
            json_extract(tn.metadata, '$.to_status'),
            json_extract(tn.metadata, '$.document_path')
        FROM task_notes tn
        WHERE tn.task_id = ?
          AND tn.note_type = 'rejection'
        ORDER BY tn.created_at DESC
    `

    rows, err := r.db.QueryContext(ctx, query, taskID)
    if err != nil {
        return nil, fmt.Errorf("failed to query rejection history: %w", err)
    }
    defer rows.Close()

    var history []*RejectionHistoryEntry
    for rows.Next() {
        entry := &RejectionHistoryEntry{}
        var rejectedBy sql.NullString
        var historyID sql.NullInt64
        var documentPath sql.NullString

        err := rows.Scan(
            &entry.ID,
            &entry.Timestamp,
            &entry.Reason,
            &rejectedBy,
            &historyID,
            &entry.FromStatus,
            &entry.ToStatus,
            &documentPath,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan rejection entry: %w", err)
        }

        // Handle nullable fields
        if rejectedBy.Valid {
            entry.RejectedBy = &rejectedBy.String
        }
        if historyID.Valid {
            entry.HistoryID = historyID.Int64
        }
        if documentPath.Valid {
            entry.DocumentPath = &documentPath.String
        }

        history = append(history, entry)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating rejection history: %w", err)
    }

    return history, nil
}
```

---

### 4. GetLatestRejection

**Signature:**
```go
func (r *TaskNoteRepository) GetLatestRejection(
    ctx context.Context,
    taskID int64,
) (*RejectionHistoryEntry, error)
```

**Implementation:**
```go
// GetLatestRejection retrieves the most recent rejection for a task
func (r *TaskNoteRepository) GetLatestRejection(
    ctx context.Context,
    taskID int64,
) (*RejectionHistoryEntry, error) {
    query := `
        SELECT
            tn.id,
            tn.created_at,
            tn.content,
            tn.created_by,
            json_extract(tn.metadata, '$.history_id'),
            json_extract(tn.metadata, '$.from_status'),
            json_extract(tn.metadata, '$.to_status'),
            json_extract(tn.metadata, '$.document_path')
        FROM task_notes tn
        WHERE tn.task_id = ?
          AND tn.note_type = 'rejection'
        ORDER BY tn.created_at DESC
        LIMIT 1
    `

    entry := &RejectionHistoryEntry{}
    var rejectedBy sql.NullString
    var historyID sql.NullInt64
    var documentPath sql.NullString

    err := r.db.QueryRowContext(ctx, query, taskID).Scan(
        &entry.ID,
        &entry.Timestamp,
        &entry.Reason,
        &rejectedBy,
        &historyID,
        &entry.FromStatus,
        &entry.ToStatus,
        &documentPath,
    )

    if err == sql.ErrNoRows {
        return nil, nil // No rejections found (not an error)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get latest rejection: %w", err)
    }

    // Handle nullable fields
    if rejectedBy.Valid {
        entry.RejectedBy = &rejectedBy.String
    }
    if historyID.Valid {
        entry.HistoryID = historyID.Int64
    }
    if documentPath.Valid {
        entry.DocumentPath = &documentPath.String
    }

    return entry, nil
}
```

---

## DocumentRepository Reuse

**File:** `internal/repository/document_repository.go` (existing)

### Methods to Use

**1. LinkDocument:**
```go
func (r *DocumentRepository) LinkDocument(
    ctx context.Context,
    taskID int64,
    documentPath string,
) error
```

**2. LinkDocumentInTx (Transaction-Safe):**
```go
func (r *DocumentRepository) LinkDocumentInTx(
    ctx context.Context,
    tx *sql.Tx,
    taskID int64,
    documentPath string,
) error
```

**Usage in rejection flow:**
```go
// Inside UpdateStatus transaction
if reasonDoc != nil {
    docRepo := NewDocumentRepository(r.db)
    if err := docRepo.LinkDocumentInTx(ctx, tx, taskID, *reasonDoc); err != nil {
        return fmt.Errorf("failed to link rejection document: %w", err)
    }
}
```

---

## Error Handling Patterns

### Custom Error Types

**File:** `internal/repository/errors.go`

```go
package repository

import "fmt"

// RejectionError represents errors related to rejection operations
type RejectionError struct {
    Operation string
    Reason    string
    Err       error
}

func (e *RejectionError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("rejection %s failed: %s (%v)", e.Operation, e.Reason, e.Err)
    }
    return fmt.Sprintf("rejection %s failed: %s", e.Operation, e.Reason)
}

func (e *RejectionError) Unwrap() error {
    return e.Err
}

// NewRejectionError creates a new RejectionError
func NewRejectionError(operation, reason string, err error) *RejectionError {
    return &RejectionError{
        Operation: operation,
        Reason:    reason,
        Err:       err,
    }
}

// Common rejection errors
var (
    ErrReasonRequired = &RejectionError{
        Operation: "status_update",
        Reason:    "rejection reason required for backward transitions",
    }

    ErrInvalidReason = &RejectionError{
        Operation: "create_note",
        Reason:    "rejection reason is invalid",
    }

    ErrDocumentNotFound = &RejectionError{
        Operation: "link_document",
        Reason:    "rejection document not found",
    }
)
```

### Error Wrapping

```go
// Wrap database errors with context
func (r *TaskNoteRepository) CreateRejectionNote(...) (*models.TaskNote, error) {
    // ... query ...
    if err != nil {
        return nil, NewRejectionError("create_note", "database insert failed", err)
    }
}

// Wrap validation errors
func sanitizeReason(reason string) (string, error) {
    if len(reason) > 5000 {
        return "", NewRejectionError("validate_reason", "reason too long", nil)
    }
}
```

### Error Handling in CLI

```go
func runUpdateCommand(cmd *cobra.Command, args []string) error {
    // ... call UpdateStatus ...

    if err != nil {
        // Check for specific rejection errors
        var rejectionErr *repository.RejectionError
        if errors.As(err, &rejectionErr) {
            if rejectionErr == repository.ErrReasonRequired {
                // Show helpful message
                cli.Error("Rejection reason required for backward transitions")
                cli.Info("\nExample:")
                cli.Info("  shark task update E07-F01-003 --status=in_development --reason=\"Missing error handling\"")
                return nil
            }
        }

        // Generic error
        return fmt.Errorf("failed to update task: %w", err)
    }

    return nil
}
```

---

## Transaction Safety Guidelines

### Pattern 1: Repository Method Creates Transaction

```go
func (r *TaskNoteRepository) CreateRejectionNote(...) (*models.TaskNote, error) {
    // Create transaction in method
    tx, err := r.db.BeginTxContext(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()  // Always rollback (no-op after commit)

    // ... operations ...

    // Commit at end
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit: %w", err)
    }

    return result, nil
}
```

### Pattern 2: InTx Method Uses Existing Transaction

```go
func (r *TaskNoteRepository) CreateRejectionNoteInTx(
    ctx context.Context,
    tx *sql.Tx,  // Transaction passed in
    ...
) (*models.TaskNote, error) {
    // Use provided transaction
    // DO NOT commit or rollback (caller's responsibility)

    result, err := tx.ExecContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to insert: %w", err)
    }

    return note, nil
}
```

### Pattern 3: Caller Manages Transaction

```go
func (r *TaskRepository) UpdateStatus(...) error {
    tx, err := r.db.BeginTxContext(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Call InTx methods
    _, err = tx.ExecContext(ctx, query1, args1...)
    if err != nil {
        return fmt.Errorf("step 1 failed: %w", err)
    }

    noteRepo := NewTaskNoteRepository(r.db)
    _, err = noteRepo.CreateRejectionNoteInTx(ctx, tx, ...)
    if err != nil {
        return fmt.Errorf("step 2 failed: %w", err)
    }

    // Commit all operations atomically
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit failed: %w", err)
    }

    return nil
}
```

---

## Models Enhancement

**File:** `internal/models/task_note.go`

**Add Metadata Field:**
```go
// TaskNote represents a typed note attached to a task
type TaskNote struct {
    ID        int64     `json:"id" db:"id"`
    TaskID    int64     `json:"task_id" db:"task_id"`
    NoteType  NoteType  `json:"note_type" db:"note_type"`
    Content   string    `json:"content" db:"content"`
    CreatedBy *string   `json:"created_by,omitempty" db:"created_by"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    Metadata  *string   `json:"metadata,omitempty" db:"metadata"` // NEW
}
```

**Add Rejection NoteType:**
```go
const (
    NoteTypeComment        NoteType = "comment"
    NoteTypeDecision       NoteType = "decision"
    NoteTypeBlocker        NoteType = "blocker"
    NoteTypeSolution       NoteType = "solution"
    NoteTypeReference      NoteType = "reference"
    NoteTypeImplementation NoteType = "implementation"
    NoteTypeTesting        NoteType = "testing"
    NoteTypeFuture         NoteType = "future"
    NoteTypeQuestion       NoteType = "question"
    NoteTypeRejection      NoteType = "rejection"  // NEW
)
```

**Add Validation:**
```go
func ValidateNoteType(noteType string) error {
    validTypes := []string{
        "comment", "decision", "blocker", "solution",
        "reference", "implementation", "testing", "future",
        "question", "rejection",  // NEW
    }
    for _, valid := range validTypes {
        if noteType == valid {
            return nil
        }
    }
    return fmt.Errorf("invalid note_type: %s", noteType)
}
```

---

## Testing Strategy

### Unit Tests

**File:** `internal/repository/task_note_repository_test.go`

```go
func TestCreateRejectionNote(t *testing.T) {
    ctx := context.Background()
    db := test.GetTestDB()
    repo := NewTaskNoteRepository(NewDB(db))

    // Setup test data
    taskID := createTestTask(t, db)
    historyID := createTestHistory(t, db, taskID)

    // Create rejection note
    note, err := repo.CreateRejectionNote(
        ctx,
        taskID,
        historyID,
        "ready_for_code_review",
        "in_development",
        "Missing error handling on line 67",
        strPtr("reviewer-agent-001"),
        nil,
    )

    if err != nil {
        t.Fatalf("CreateRejectionNote() error = %v", err)
    }

    if note.ID == 0 {
        t.Error("expected note ID to be set")
    }

    if note.NoteType != models.NoteTypeRejection {
        t.Errorf("expected note_type 'rejection', got %s", note.NoteType)
    }

    // Verify metadata
    if note.Metadata == nil {
        t.Fatal("expected metadata to be set")
    }

    var metadata map[string]interface{}
    json.Unmarshal([]byte(*note.Metadata), &metadata)

    if metadata["history_id"].(float64) != float64(historyID) {
        t.Errorf("expected history_id %d, got %v", historyID, metadata["history_id"])
    }
}

func TestGetRejectionHistory(t *testing.T) {
    ctx := context.Background()
    db := test.GetTestDB()
    repo := NewTaskNoteRepository(NewDB(db))

    // Setup: Create task with 3 rejections
    taskID := createTestTask(t, db)
    for i := 0; i < 3; i++ {
        historyID := createTestHistory(t, db, taskID)
        _, err := repo.CreateRejectionNote(
            ctx, taskID, historyID, "ready_for_code_review", "in_development",
            fmt.Sprintf("Rejection %d", i+1), strPtr("reviewer"), nil,
        )
        if err != nil {
            t.Fatalf("failed to create test rejection: %v", err)
        }
    }

    // Get rejection history
    history, err := repo.GetRejectionHistory(ctx, taskID)
    if err != nil {
        t.Fatalf("GetRejectionHistory() error = %v", err)
    }

    if len(history) != 3 {
        t.Errorf("expected 3 rejections, got %d", len(history))
    }

    // Verify order (most recent first)
    if !strings.Contains(history[0].Reason, "Rejection 3") {
        t.Error("expected most recent rejection first")
    }
}

func TestIsBackwardTransition(t *testing.T) {
    // See test cases documented in TaskRepository section above
}
```

---

## Summary

### New Repository Methods

**TaskRepository:**
- ✅ UpdateStatus() - Enhanced with reason/reasonDoc params
- ✅ isBackwardTransition() - Helper for phase comparison

**TaskNoteRepository (New):**
- ✅ CreateRejectionNote() - Create rejection with metadata
- ✅ CreateRejectionNoteInTx() - Transaction-safe creation
- ✅ GetRejectionHistory() - Retrieve all rejections for task
- ✅ GetLatestRejection() - Get most recent rejection

**DocumentRepository (Existing, Reused):**
- ✅ LinkDocument() - Link document to task
- ✅ LinkDocumentInTx() - Transaction-safe document linking

### Error Handling
- ✅ Custom RejectionError type
- ✅ Clear, actionable error messages
- ✅ Context-aware error wrapping
- ✅ Transaction safety patterns

### Transaction Patterns
- ✅ Pattern 1: Method creates transaction
- ✅ Pattern 2: InTx method uses existing transaction
- ✅ Pattern 3: Caller manages transaction
- ✅ Always use defer tx.Rollback()

### Validation
- ✅ Reason sanitization (length, null bytes)
- ✅ Document path validation (exists, within project)
- ✅ Backward transition detection (phase-based)
- ✅ Metadata JSON schema validation
