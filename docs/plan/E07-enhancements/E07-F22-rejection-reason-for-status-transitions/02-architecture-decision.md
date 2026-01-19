# Architecture Decision: Rejection Reason Storage

**Feature:** E07-F22 - Rejection Reason for Status Transitions
**Version:** 1.0
**Date:** 2026-01-16
**Status:** Decision Made - Implementation Required

---

## Executive Summary

**Problem:** Rejection reasons are not being stored when users provide `--rejection-reason` flags during backward status transitions (e.g., code review rejections).

**Root Cause:** The `UpdateStatusForced()` method in `task_repository.go` performs DIRECT INSERT into `task_history` table, bypassing the repository layer and missing the `rejection_reason` column.

**Decision:** Use the existing `task_history.rejection_reason` column (already implemented) and fix the data flow to populate it correctly.

**Impact:** Minimal - requires updating one repository method and one CLI call site.

---

## Current State Analysis

### Database Schema (✅ CORRECT)

The `task_history` table **already has** the `rejection_reason` column:

```sql
PRAGMA table_info(task_history);
-- Output:
-- ...
-- 8|rejection_reason|TEXT|0||0
```

**Migration:** Already completed in T-E07-F22-024
**Index:** `idx_task_history_rejection` exists
**Status:** ✅ Schema is correct

### Data Model (✅ CORRECT)

The `TaskHistory` model includes the field:

```go
type TaskHistory struct {
    ID              int64
    TaskID          int64
    OldStatus       string
    NewStatus       string
    RejectionReason string  // ✅ Field exists
    Notes           string
    Agent           string
    Timestamp       time.Time
}
```

**Status:** ✅ Model is correct

### Repository Layer (⚠️ PARTIAL)

**TaskHistoryRepository.Create():** ✅ CORRECT
```go
// File: internal/repository/task_history_repository.go:41-51
query := `
    INSERT INTO task_history (task_id, old_status, new_status, agent, notes, rejection_reason)
    VALUES (?, ?, ?, ?, ?, ?)
`
result, err := r.db.ExecContext(ctx, query,
    history.TaskID,
    history.OldStatus,
    history.NewStatus,
    history.Agent,
    history.Notes,
    history.RejectionReason,  // ✅ Correctly included
)
```

**TaskRepository.UpdateStatusForced():** ❌ INCORRECT
```go
// File: internal/repository/task_repository.go:914-918
historyQuery := `
    INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced)
    VALUES (?, ?, ?, ?, ?, ?)
`
_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, newStatus, agent, notes, force)
// ❌ Missing rejection_reason parameter
// ❌ Direct INSERT bypassing TaskHistoryRepository
```

**Problem:** `UpdateStatusForced()` does DIRECT INSERT with hardcoded column list, missing `rejection_reason`.

**Impact:** All status transitions using `UpdateStatusForced()` fail to record rejection reasons.

### CLI Layer (⚠️ PARTIAL)

**Flag Definition:** ✅ CORRECT
```go
// File: internal/cli/commands/task.go
taskReopenCmd.Flags().String("rejection-reason", "", "Reason for rejection or sending task back")
```

**Flag Usage:** ❌ INCORRECT
```go
// File: internal/cli/commands/task.go:2002-2012
notesFlag, _ := cmd.Flags().GetString("notes")  // ✅ Gets notes flag
var notes *string
if notesFlag != "" {
    notes = &notesFlag
}

// ❌ Missing: rejection-reason flag retrieval
// ❌ Missing: passing rejection reason to repository

// Calls repository with ONLY notes parameter
if err := repo.ReopenTaskForced(ctx, task.ID, &agent, notes, force); err != nil {
```

**Problem:** CLI reads `--rejection-reason` flag but never retrieves or passes it to the repository.

---

## Design Documents Review

### Original Design (03-database-design.md)

**Approach:** Use `task_notes` table with JSON `metadata` column

```json
{
  "history_id": 234,
  "from_status": "ready_for_code_review",
  "to_status": "in_development",
  "document_path": null
}
```

**Rationale:**
- Leverages existing `task_notes` table
- Flexible metadata storage (JSON)
- Allows linking to external documents

**Status:** NOT IMPLEMENTED

### Current Schema (Implemented)

**Approach:** Direct column `task_history.rejection_reason`

**Rationale:**
- Simpler query patterns (no JSON extraction)
- Better performance (indexed TEXT column vs JSON path queries)
- Direct SQL aggregation and filtering
- Backward compatible (NULL = no reason)

**Status:** ✅ IMPLEMENTED (T-E07-F22-024)

### Task Requirements (T-E07-F22-026)

**Requirement:** Validation in `UpdateStatus()` to enforce rejection reasons for backward transitions

**Status:** ⚠️ PARTIALLY IMPLEMENTED
- Validation logic NOT in `UpdateStatus()`
- CLI flag exists but not used
- Repository method bypasses validation

---

## Architectural Decision

### Decision: Use `task_history.rejection_reason` Column

**Chosen Approach:** Direct column storage (already implemented)

**Rationale:**
1. ✅ Schema already exists and tested
2. ✅ Model already includes field
3. ✅ Repository layer (TaskHistoryRepository) already supports it
4. ✅ Simpler queries (no JSON extraction)
5. ✅ Better performance (indexed TEXT vs JSON path)
6. ✅ Easier for reporting and analytics
7. ✅ Backward compatible (NULL rejection_reason for old records)

**Rejected Alternative:** `task_notes.metadata` JSON approach
- ❌ Requires new migration
- ❌ More complex queries (JSON extraction)
- ❌ Slower query performance
- ❌ Additional table join for retrieval
- ❌ Duplicate data (reason in both history and notes)

**Verdict:** The current schema design is **correct** - we just need to fix the implementation to use it.

---

## Root Cause Analysis

### The Bug: UpdateStatusForced() Bypasses Repository Layer

**File:** `internal/repository/task_repository.go:914-918`

**Problem:**
```go
// ❌ Direct INSERT bypassing TaskHistoryRepository.Create()
historyQuery := `
    INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced)
    VALUES (?, ?, ?, ?, ?, ?)
`
_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, newStatus, agent, notes, force)
```

**Why This is Wrong:**
1. Hardcoded column list excludes `rejection_reason`
2. Bypasses `TaskHistoryRepository.Create()` which DOES include rejection_reason
3. Violates repository pattern (data access should go through repository)
4. Creates maintenance burden (two places to update for history changes)

**Who Calls This Method:**
```bash
$ grep -r "UpdateStatusForced" internal/ | grep -v test | grep -v backup
internal/repository/task_repository.go (definition)
internal/cli/commands/task.go (multiple callers)
```

**Call Sites:**
- `ReopenTaskForced()` → calls `UpdateStatusForced()`
- `BlockTaskForced()` → calls `UpdateStatusForced()`
- `UnblockTaskForced()` → calls `UpdateStatusForced()`
- CLI commands → call `*TaskForced()` methods

**Data Flow (Current - Broken):**
```
CLI:
  --rejection-reason="..." flag defined ✅
  ↓
  flag value NOT retrieved ❌
  ↓
  notes parameter passed (NOT rejection_reason) ❌
  ↓
Repository:
  ReopenTaskForced(notes) receives only notes ❌
  ↓
  UpdateStatusForced(notes) direct INSERT ❌
  ↓
Database:
  rejection_reason column = NULL ❌
```

**Data Flow (Correct - To Be Implemented):**
```
CLI:
  --rejection-reason="..." flag defined ✅
  ↓
  rejectionReason, _ := cmd.Flags().GetString("rejection-reason") ✅
  ↓
  ReopenTaskForced(notes, rejectionReason) ✅
  ↓
Repository:
  UpdateStatusForced receives rejectionReason ✅
  ↓
  Create TaskHistory struct with RejectionReason ✅
  ↓
  TaskHistoryRepository.Create(history) ✅
  ↓
Database:
  rejection_reason column populated ✅
```

---

## Recommended Solution

### Strategy: Refactor UpdateStatusForced to Use Repository

**Approach:** Replace direct INSERT with proper repository call

**Benefits:**
1. ✅ Single source of truth (TaskHistoryRepository)
2. ✅ Automatic support for future history fields
3. ✅ Consistent with repository pattern
4. ✅ Easier to test and maintain
5. ✅ Type-safe (uses models.TaskHistory struct)

### Implementation Steps

#### Step 1: Update UpdateStatusForced Signature

**File:** `internal/repository/task_repository.go`

**Current:**
```go
func (r *TaskRepository) UpdateStatusForced(
    ctx context.Context,
    taskID int64,
    newStatus models.TaskStatus,
    agent *string,
    notes *string,
    force bool,
) error
```

**Updated:**
```go
func (r *TaskRepository) UpdateStatusForced(
    ctx context.Context,
    taskID int64,
    newStatus models.TaskStatus,
    agent *string,
    notes *string,
    rejectionReason *string,  // NEW PARAMETER
    force bool,
) error
```

#### Step 2: Replace Direct INSERT with Repository Call

**Current (lines 914-918):**
```go
// ❌ Direct INSERT
historyQuery := `
    INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced)
    VALUES (?, ?, ?, ?, ?, ?)
`
_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, newStatus, agent, notes, force)
```

**Updated:**
```go
// ✅ Use TaskHistoryRepository
historyRepo := NewTaskHistoryRepository(&DB{DB: tx}) // Use transaction

history := &models.TaskHistory{
    TaskID:          taskID,
    OldStatus:       string(currentStatus),
    NewStatus:       string(newStatus),
    Agent:           derefString(agent),
    Notes:           derefString(notes),
    RejectionReason: derefString(rejectionReason),  // NEW
    Timestamp:       time.Now(),
}

if err := historyRepo.Create(ctx, history); err != nil {
    return fmt.Errorf("failed to create history record: %w", err)
}
```

**Helper Function (if not exists):**
```go
func derefString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}
```

#### Step 3: Update Wrapper Methods

**File:** `internal/repository/task_repository.go`

**Methods to Update:**
- `ReopenTaskForced()`
- `BlockTaskForced()`
- `UnblockTaskForced()`

**Example (ReopenTaskForced):**

**Current:**
```go
func (r *TaskRepository) ReopenTaskForced(
    ctx context.Context,
    taskID int64,
    agent *string,
    notes *string,
    force bool,
) error {
    return r.UpdateStatusForced(ctx, taskID, models.TaskStatusInProgress, agent, notes, force)
}
```

**Updated:**
```go
func (r *TaskRepository) ReopenTaskForced(
    ctx context.Context,
    taskID int64,
    agent *string,
    notes *string,
    rejectionReason *string,  // NEW PARAMETER
    force bool,
) error {
    return r.UpdateStatusForced(ctx, taskID, models.TaskStatusInProgress, agent, notes, rejectionReason, force)
}
```

#### Step 4: Update CLI Commands

**File:** `internal/cli/commands/task.go`

**Current (runTaskReopen):**
```go
// Lines 2002-2012
notesFlag, _ := cmd.Flags().GetString("notes")
var notes *string
if notesFlag != "" {
    notes = &notesFlag
}

// ❌ Missing rejection-reason flag retrieval

if err := repo.ReopenTaskForced(ctx, task.ID, &agent, notes, force); err != nil {
```

**Updated:**
```go
// Get notes flag
notesFlag, _ := cmd.Flags().GetString("notes")
var notes *string
if notesFlag != "" {
    notes = &notesFlag
}

// ✅ Get rejection-reason flag
rejectionReasonFlag, _ := cmd.Flags().GetString("rejection-reason")
var rejectionReason *string
if rejectionReasonFlag != "" {
    rejectionReason = &rejectionReasonFlag
}

// ✅ Pass rejection reason to repository
if err := repo.ReopenTaskForced(ctx, task.ID, &agent, notes, rejectionReason, force); err != nil {
```

**Apply same pattern to:**
- `runTaskApprove()` (if it supports rejection feedback)
- Any other command using `*TaskForced()` methods

---

## Data Flow Diagram (Text)

### Current Flow (Broken)

```
┌─────────────────────────────────────────────────────────────┐
│ CLI Layer (internal/cli/commands/task.go)                   │
│                                                              │
│  1. Flag defined: --rejection-reason ✅                      │
│  2. Flag value NOT retrieved ❌                              │
│  3. Only --notes passed to repository ❌                     │
└───────────────────────────┬──────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Repository Layer (internal/repository/task_repository.go)   │
│                                                              │
│  ReopenTaskForced(notes) ❌                                  │
│    │                                                         │
│    └──> UpdateStatusForced(notes) ❌                         │
│          │                                                   │
│          └──> DIRECT INSERT to task_history ❌               │
│                (missing rejection_reason column)            │
└───────────────────────────┬──────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Database (task_history table)                               │
│                                                              │
│  rejection_reason = NULL ❌                                  │
└─────────────────────────────────────────────────────────────┘
```

### Fixed Flow (Proposed)

```
┌─────────────────────────────────────────────────────────────┐
│ CLI Layer (internal/cli/commands/task.go)                   │
│                                                              │
│  1. Flag defined: --rejection-reason ✅                      │
│  2. rejectionReason = cmd.Flags().GetString("rejection-...")│
│  3. Pass rejectionReason to repository ✅                    │
└───────────────────────────┬──────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Repository Layer (internal/repository/task_repository.go)   │
│                                                              │
│  ReopenTaskForced(notes, rejectionReason) ✅                 │
│    │                                                         │
│    └──> UpdateStatusForced(notes, rejectionReason) ✅        │
│          │                                                   │
│          └──> Create TaskHistory struct ✅                   │
│                │                                             │
│                └──> TaskHistoryRepository.Create() ✅        │
└───────────────────────────┬──────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Database (task_history table)                               │
│                                                              │
│  rejection_reason = "Missing error handling..." ✅           │
└─────────────────────────────────────────────────────────────┘
```

---

## Files Requiring Changes

### 1. Repository Layer

**File:** `internal/repository/task_repository.go`

**Changes:**
- Update `UpdateStatusForced()` signature (add `rejectionReason` parameter)
- Replace direct INSERT with `TaskHistoryRepository.Create()` call
- Update wrapper methods: `ReopenTaskForced()`, `BlockTaskForced()`, `UnblockTaskForced()`

**Lines:** ~914-918 (direct INSERT replacement), wrapper methods

**Estimated LOC:** 30 lines changed

### 2. CLI Commands

**File:** `internal/cli/commands/task.go`

**Changes:**
- Add rejection reason flag retrieval in `runTaskReopen()`
- Add rejection reason flag retrieval in `runTaskApprove()` (if applicable)
- Pass rejection reason to repository method calls

**Lines:** ~2002-2012 (runTaskReopen), ~1740-1760 (runTaskApprove if applicable)

**Estimated LOC:** 20 lines changed

### 3. Tests (Repository)

**File:** `internal/repository/task_repository_test.go`

**Changes:**
- Update test calls to `*TaskForced()` methods (add rejectionReason parameter)
- Add test case: rejection reason stored correctly
- Add test case: NULL rejection reason handled correctly

**Estimated LOC:** 50 lines (new tests + updates)

### 4. Tests (CLI)

**File:** `internal/cli/commands/task_test.go` or `task_update_test.go`

**Changes:**
- Verify rejection-reason flag exists (already exists)
- Verify rejection reason passed to repository (new test)

**Estimated LOC:** 20 lines

---

## Testing Strategy

### Unit Tests (Repository Layer)

```go
func TestUpdateStatusForced_WithRejectionReason(t *testing.T) {
    // Setup: Create task in ready_for_code_review
    // Action: Call UpdateStatusForced with rejection reason
    // Assert: task_history has rejection_reason populated
}

func TestUpdateStatusForced_WithoutRejectionReason(t *testing.T) {
    // Setup: Create task
    // Action: Call UpdateStatusForced WITHOUT rejection reason
    // Assert: task_history.rejection_reason = "" (empty, not NULL)
}

func TestReopenTaskForced_WithRejectionReason(t *testing.T) {
    // Setup: Create task in ready_for_review
    // Action: Call ReopenTaskForced with rejection reason
    // Assert: task status updated AND rejection reason stored
}
```

### Integration Tests (CLI)

```bash
# Test 1: CLI flag retrieval
$ shark task reopen T-E07-F22-001 --rejection-reason="Missing tests"
# Verify: rejection_reason stored in database

# Test 2: No rejection reason
$ shark task reopen T-E07-F22-002
# Verify: rejection_reason = "" (not NULL)

# Test 3: Query rejection history
$ shark task get T-E07-F22-001 --json | jq '.rejection_history'
# Verify: rejection reason appears in output
```

### Database Verification

```sql
-- Verify rejection reason stored
SELECT task_id, old_status, new_status, rejection_reason
FROM task_history
WHERE rejection_reason IS NOT NULL;

-- Expected: Rows with rejection reasons appear
```

---

## Success Metrics

### Functional Metrics
- ✅ Rejection reasons stored in database when `--rejection-reason` flag provided
- ✅ NULL rejection reasons handled gracefully (empty string in Go)
- ✅ `shark task get` displays rejection history with reasons
- ✅ All existing tests pass
- ✅ New tests cover rejection reason scenarios

### Code Quality Metrics
- ✅ Repository pattern maintained (no direct INSERT bypass)
- ✅ Single source of truth (TaskHistoryRepository)
- ✅ Type-safe (uses models.TaskHistory struct)
- ✅ Backward compatible (existing code continues to work)

---

## Risks and Mitigations

### Risk 1: Breaking Existing Callers

**Risk:** Changing `UpdateStatusForced()` signature breaks existing callers

**Mitigation:**
- Grep for all callers: `grep -r "UpdateStatusForced" internal/`
- Update all call sites in same commit
- Run full test suite before commit
- Create comprehensive integration tests

**Likelihood:** Medium
**Impact:** High (breaks builds)
**Mitigation Cost:** Low (search and replace)

### Risk 2: Transaction Handling

**Risk:** Creating `TaskHistoryRepository` with transaction context fails

**Mitigation:**
- Use `&DB{DB: tx}` wrapper to pass transaction to repository
- Test transaction rollback scenarios
- Verify ACID properties maintained

**Likelihood:** Low
**Impact:** Medium (data consistency)
**Mitigation Cost:** Low (proper testing)

### Risk 3: Backward Compatibility

**Risk:** Existing task_history records have NULL rejection_reason

**Mitigation:**
- Go empty string ("") for NULL rejection_reason (already works)
- JSON serialization uses `omitempty` (already implemented)
- Queries use `rejection_reason IS NOT NULL` filter (already implemented)

**Likelihood:** Low
**Impact:** Low (cosmetic)
**Mitigation Cost:** None (already handled)

---

## Implementation Timeline

### Phase 1: Repository Layer (1-2 hours)
1. Update `UpdateStatusForced()` signature
2. Replace direct INSERT with `TaskHistoryRepository.Create()`
3. Update wrapper methods (`ReopenTaskForced`, etc.)
4. Write repository unit tests

### Phase 2: CLI Layer (1 hour)
1. Add rejection reason flag retrieval in `runTaskReopen()`
2. Add rejection reason flag retrieval in `runTaskApprove()` (if applicable)
3. Update method calls to pass rejection reason
4. Write CLI integration tests

### Phase 3: Testing & Validation (1 hour)
1. Run full test suite
2. Manual testing with `shark task reopen --rejection-reason="..."`
3. Verify database contains rejection reasons
4. Test rejection history retrieval

**Total Estimated Time:** 3-4 hours

---

## Conclusion

### Summary

**Problem:** Rejection reasons not stored due to `UpdateStatusForced()` bypassing repository layer

**Root Cause:** Direct INSERT with hardcoded column list missing `rejection_reason`

**Solution:** Refactor to use `TaskHistoryRepository.Create()` (proper repository pattern)

**Impact:** Minimal code changes, significant improvement to data flow and maintainability

**Recommendation:** Proceed with refactor approach (Step 1-4 above)

### Next Steps

1. **Architect Review:** Approve this decision document
2. **Implementation:** Developer implements Steps 1-4
3. **QA Testing:** Verify rejection reasons stored and retrieved correctly
4. **Documentation:** Update CLI_REFERENCE.md with rejection reason examples

---

**Document Version:** 1.0
**Last Updated:** 2026-01-16
**Status:** Ready for Implementation
**Reviewed By:** Architect Agent
