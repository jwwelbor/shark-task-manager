# Exploratory Findings: T-E07-F22-028

**QA Agent:** qa-agent
**Date:** 2026-01-17 01:07:32
**Task:** T-E07-F22-028 - Add rejection reason to task history display

## Findings Summary

| Type | Severity | Count |
|------|----------|-------|
| Bug | High | 1 |
| Observation | Info | 2 |

---

## 🐛 Bug: Task Get Queries Wrong Table for Rejection History

**Severity:** High
**Impact:** Core feature non-functional

**Description:**
The `task get` command implementation queries `task_notes` table for rejection history, but rejection_reason is stored in `task_history` table. This causes the "Last Rejection" section to never display.

**Steps to Reproduce:**
1. Create task: `shark task create E07 F22 "Test task"`
2. Move through workflow to ready_for_code_review
3. Reject: `shark task update <task> --status in_development --reason "Test rejection"`
4. View task: `shark task get <task>`
5. Observe: No rejection history displayed

**Expected:** Rejection history section displayed
**Actual:** Nothing displayed

**Code Location:** `internal/cli/commands/task.go` lines 683-691

**Root Cause:** Mismatch between data storage (task_history) and query location (task_notes)

**Suggested Fix:**
```go
// Replace noteRepo.GetRejectionHistory with task_history query
historyRepo := repository.NewTaskHistoryRepository(repoDb)
allHistory, err := historyRepo.GetHistoryByTaskID(ctx, task.ID)
if err == nil {
    // Filter for entries with rejection_reason
    rejectionHistory := filterRejections(allHistory)
}
```

---

## 💡 Observation: Two Separate Rejection Storage Mechanisms

**Type:** Architectural

**Description:**
The codebase has two different mechanisms for storing rejection information:

1. **task_history.rejection_reason** (TEXT column) - Stores inline rejection text
2. **task_notes** with note_type='rejection' - Stores rich rejection notes with metadata

**Files Involved:**
- `internal/repository/task_history_repository.go` - Handles task_history table
- `internal/repository/task_note_repository.go` - Handles task_notes table with GetRejectionHistory method

**Impact:**
Creates confusion about which storage mechanism is authoritative. Current implementation in E07-F22 uses task_history.rejection_reason, but task_note_repository has a GetRejectionHistory method that queries task_notes.

**Recommendation:**
- Document when to use each mechanism
- Consider unifying rejection storage in future refactoring
- Update GetRejectionHistory to query task_history if that's the primary source

---

## 💡 Observation: Excellent Color Coding in Task History

**Type:** Positive

**Description:**
The task history command uses effective color coding:
- Cyan for timeline markers and timestamps
- Red for rejection reasons
- Yellow for agent names
- Gray for relative time

**User Experience Impact:** Positive - Makes rejection reasons immediately visible

**Example Output:**
```
├─ 2026-01-17 07:06:12 ready_for_code_review → in_development (just now)
│  Rejection: Missing error handling for edge cases. Added TODO comments in code.
                ^^^ Red color makes this stand out
```

**Recommendation:** Apply same color coding to task get rejection display when bug is fixed

---

## Test Data

**Test Task Created:** T-E07-F22-032
**Test Scenario:** Full workflow with rejection

**Workflow Steps:**
1. draft → ready_for_refinement
2. ready_for_refinement → in_refinement
3. in_refinement → ready_for_development
4. ready_for_development → in_development
5. in_development → ready_for_code_review
6. **ready_for_code_review → in_development** (with rejection reason)

**Database Verification:**
```sql
-- Verified rejection_reason stored correctly
SELECT rejection_reason FROM task_history
WHERE task_id = (SELECT id FROM tasks WHERE key = 'T-E07-F22-032')
AND rejection_reason IS NOT NULL;

Result: "Missing error handling for edge cases. Added TODO comments in code."
```

---

## Recommendations for Future Testing

1. **Integration Tests Needed:**
   - Test task get with rejection history
   - Test task get without rejection history
   - Test JSON output includes rejection_history

2. **Edge Cases to Test:**
   - Multiple rejections (should show most recent)
   - Very long rejection reasons (text wrapping)
   - Rejection reason with special characters
   - Rejection reason with markdown formatting

3. **Performance Considerations:**
   - Task history query performance with many entries
   - Filtering rejection entries from full history

---

## Positive Observations

✅ Task history command works flawlessly
✅ JSON output structure is clean and complete
✅ Color coding enhances readability
✅ Chronological ordering is correct
✅ No errors when task has no rejections

---

## Action Items

1. **Fix bug in task get** - Update to query task_history instead of task_notes
2. **Add integration test** - Ensure task get displays rejection history
3. **Document storage mechanism** - Clarify when to use task_history vs task_notes
4. **Add edge case tests** - Multiple rejections, long text, special characters
