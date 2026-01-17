# Exploratory Testing Findings: T-E07-F22-027

**Date:** 2026-01-17 00:56:00
**Task:** T-E07-F22-027 - Add CLI flag --rejection-reason
**QA Agent:** QA
**Testing Duration:** Unable to perform exploratory testing due to compilation errors

---

## Testing Scope

**Charter:** Explore task update command with rejection reason flag to discover usability issues and edge cases

**Target Areas:**
- CLI flag naming and usability
- Error message clarity
- Backward vs forward transition handling
- Integration with existing workflow
- Code quality and test coverage

---

## Findings

### Finding #1: Repository Checks Wrong Parameter - FEATURE BROKEN (CRITICAL)

**Type:** Logic Bug
**Severity:** CRITICAL - Blocks Feature
**Status:** Open

**Description:**
The core validation logic in `UpdateStatusForced()` checks the wrong parameter. It validates `notes` instead of `rejectionReason`, making the entire feature non-functional.

**Code Location:**
File: `internal/repository/task_repository.go`
Line: 881

**Broken Code:**
```go
if notes == nil || strings.TrimSpace(*notes) == "" {
    return fmt.Errorf("rejection reason required for backward transition from %s to %s: use --reason flag or use --force to bypass", currentStatus, newStatus)
}
```

**Impact:**
- **100% Feature Failure** - Rejection reason NEVER works
- User provides `--reason "..."` but it's completely ignored
- Validation always fails even with valid rejection reason
- Only workaround is `--force` which bypasses workflow entirely

**Reproduction:**
```bash
# Task in in_qa status
$ ./bin/shark task get T-E07-F22-027
Status: in_qa

# Attempt backward transition WITH rejection reason:
$ ./bin/shark task update T-E07-F22-027 --status in_development \
    --reason "Test compilation errors - UpdateStatusForced() calls missing rejectionReason parameter"

# Result: ERROR (rejection reason was provided but ignored!)
ERROR Error: Failed to update task status: rejection reason required for backward
transition from in_qa to in_development: use --reason flag or use --force to bypass
```

**Root Cause Analysis:**
Function signature has 3 optional string pointers:
- `agent *string` - Who performed the transition
- `notes *string` - Optional notes/comments
- `rejectionReason *string` - Required for backward transitions

Developer confused these parameters and checked `notes` instead of `rejectionReason`.

**Correct Fix:**
```go
if rejectionReason == nil || strings.TrimSpace(*rejectionReason) == "" {
    return fmt.Errorf("rejection reason required for backward transition from %s to %s: use --reason flag or use --force to bypass", currentStatus, newStatus)
}
```

**Testing Evidence:**
QA attempted to reject this task using the implemented feature and discovered it doesn't work. This is a "dogfooding" bug - found by using the feature to test itself.

**Severity Justification:**
- Blocks entire feature functionality
- Cannot use rejection reasons at all
- Only discoverable through actual usage (not caught by unit tests)
- No workaround except bypassing workflow with `--force`

---

### Finding #2: Test Compilation Errors Block All Testing (CRITICAL)

**Type:** Blocker
**Severity:** Critical
**Status:** Open

**Description:**
Two test files have compilation errors that prevent the test suite from running:
- `internal/cli/commands/task_update_status_test.go:82`
- `internal/cli/commands/task_update_status_test.go:180`

**Details:**
Both lines call `UpdateStatusForced()` with 6 arguments instead of required 7. Missing the `rejectionReason *string` parameter.

**Impact:**
- Cannot compile or run tests
- Cannot verify functionality
- Blocks QA approval
- Violates "tests must pass" acceptance criterion

**Reproduction:**
```bash
make test
# Compilation error prevents execution
```

**Root Cause:**
Developer added `rejectionReason` parameter to `UpdateStatusForced()` signature but forgot to update test calls.

**Recommendation:**
Add `nil` as 6th parameter to both function calls.

---

### Finding #2: Flag Name Differs from Specification (MINOR)

**Type:** Discrepancy
**Severity:** Low
**Status:** Open (Discussion Needed)

**Description:**
Task specification requests `--rejection-reason` flag, but implementation uses `--reason`.

**Analysis:**

**Pros of `--reason`:**
- Shorter, easier to type
- Consistent with existing `task block --reason` command
- More general-purpose (works for any "reason" context)
- Better UX (less typing)

**Pros of `--rejection-reason`:**
- Matches specification exactly
- More explicit about purpose
- Self-documenting
- Matches other flags like `--rejection-reason` in `task approve` and `task reopen`

**Evidence:**
```bash
# Current implementation:
shark task update E07-F22-001 --status in_development --reason "..."

# Specification expected:
shark task update E07-F22-001 --status in_development --rejection-reason "..."
```

**Other Commands Using --rejection-reason:**
- `task approve --rejection-reason` (line 2158)
- `task reopen --rejection-reason` (line 2170)

**Recommendation:**
**CHANGE to `--rejection-reason`** for consistency with existing commands (`task approve`, `task reopen`).

**Alternative:**
Add `--rejection-reason` as primary flag with `--reason` as hidden alias for backward compatibility.

---

### Finding #3: Good Error Message UX (POSITIVE)

**Type:** Positive Finding
**Severity:** N/A
**Status:** Verified in Code Review

**Description:**
Error messages are well-crafted and user-friendly:

```go
cli.Error(fmt.Sprintf("Error: %s", err.Error()))
cli.Info(fmt.Sprintf("Use --reason to provide a reason, or use --force to bypass this requirement"))
cli.Info(fmt.Sprintf("Example: shark task update %s --status %s --reason \"Reason for transition\"", taskKey, status))
```

**Positive Aspects:**
- Clear error message
- Provides actionable guidance ("Use --reason...")
- Includes concrete example with actual task key and status
- Mentions `--force` escape hatch for admin use

**Recommendation:**
No changes needed. Excellent UX pattern.

---

### Finding #4: Missing Test Coverage for Rejection Reason Storage (MEDIUM)

**Type:** Test Gap
**Severity:** Medium
**Status:** Open

**Description:**
While tests verify that `UpdateStatusForced()` is called, there's no test that verifies:
1. Rejection reason is actually stored in the database
2. Rejection reason can be retrieved later
3. Rejection history is correctly populated

**Impact:**
- Cannot verify data persistence
- Risk that rejection reason is lost
- No validation that feature works end-to-end

**Recommendation:**
Add integration test:
1. Create task in `ready_for_code_review`
2. Update to `in_development` with `--reason "Test rejection"`
3. Query task to verify rejection reason was stored
4. Verify rejection appears in `task get` output

---

### Finding #5: Workflow Validation Dependency (OBSERVATION)

**Type:** Observation
**Severity:** N/A
**Status:** Informational

**Description:**
Command relies on `validation.ValidateReasonForStatusTransition()` which loads workflow config:

```go
workflow, err := config.LoadWorkflowConfig(configPath)
if err != nil {
    cli.Error(fmt.Sprintf("Error: Failed to load workflow config: %v", err))
    os.Exit(1)
}
```

**Observations:**
- Tight coupling to workflow config file
- Error if config file missing or malformed
- No fallback behavior

**Questions:**
1. What happens if `.sharkconfig.json` doesn't exist?
2. What if workflow config is incomplete?
3. Should there be default workflow rules?

**Recommendation:**
Document expected behavior when config is missing. Consider adding helpful error message pointing to workflow config setup.

---

### Finding #6: Force Flag Bypass (SECURITY CONSIDERATION)

**Type:** Security/Governance
**Severity:** Low
**Status:** Informational

**Description:**
The `--force` flag allows users to bypass rejection reason requirement:

```go
force, _ := cmd.Flags().GetBool("force")
if err := validation.ValidateReasonForStatusTransition(status, string(task.Status), reason, force, workflow); err != nil {
    // Error only if validation fails and force=false
}
```

**Considerations:**
- Useful for admin overrides
- Could be abused to skip documentation
- No audit of force flag usage
- No warning when force is used (actually, there IS a warning at line 2400-2402)

**Positive Finding:**
Code DOES display warning when force is used:
```go
if force && !cli.GlobalConfig.JSON {
    cli.Warning(fmt.Sprintf("⚠️  Forced transition from %s to %s (bypassed workflow validation)", task.Status, newStatus))
}
```

**Recommendation:**
Current implementation is good. Warning is sufficient.

---

## Edge Cases to Test (When Tests Compile)

### Edge Case #1: Empty Rejection Reason
```bash
shark task update E07-F22-001 --status in_development --reason ""
```
**Expected:** Should reject (empty string is not valid reason)
**Status:** Untested

### Edge Case #2: Very Long Rejection Reason
```bash
shark task update E07-F22-001 --status in_development --reason "$(head -c 10000 /dev/zero | tr '\0' 'a')"
```
**Expected:** Should accept (or reject with clear error if length limit)
**Status:** Untested

### Edge Case #3: Rejection Reason with Special Characters
```bash
shark task update E07-F22-001 --status in_development --reason "Missing \"error handling\" & validation"
```
**Expected:** Should properly escape and store special characters
**Status:** Untested

### Edge Case #4: Multiple Status Updates in Sequence
```bash
# Transition 1: ready_for_code_review -> in_development
shark task update E07-F22-001 --status in_development --reason "First rejection"

# Transition 2: in_development -> ready_for_code_review
shark task update E07-F22-001 --status ready_for_code_review

# Transition 3: ready_for_code_review -> in_development
shark task update E07-F22-001 --status in_development --reason "Second rejection"

# View history
shark task get E07-F22-001
```
**Expected:** Should show rejection history with both reasons
**Status:** Untested

---

## Usability Observations

### Positive Patterns

1. ✅ **Consistent CLI syntax** - Follows existing `shark task` patterns
2. ✅ **Good error messages** - Clear, actionable, with examples
3. ✅ **Force flag escape hatch** - Admin override available with warning
4. ✅ **JSON output support** - Machine-readable output available

### Potential UX Issues

1. ⚠️ **Flag naming inconsistency** - Should match `task approve --rejection-reason`
2. ⚠️ **No validation of reason content** - Could users provide empty or meaningless reasons?
3. ⚠️ **Workflow dependency unclear** - What if config is missing?

---

## Code Quality Assessment

### Strengths

- Clean, readable code structure
- Good error handling with context
- Proper use of validation layer
- Timeout context for database operations
- Graceful handling of optional parameters (nil pointers)

### Weaknesses

- Tests out of sync with implementation (compilation errors)
- Missing integration test for rejection reason storage
- No test for rejection history retrieval
- Flag naming doesn't match specification or existing commands

---

## Recommendations Summary

### Critical (Must Fix)

1. ✅ Fix repository parameter check - line 881 change `notes` to `rejectionReason` (**BREAKS ENTIRE FEATURE**)
2. ✅ Fix test compilation errors (2 lines, add `nil` parameter)
3. ✅ Verify tests pass after fix

### High Priority (Should Fix)

1. ⚠️ Change flag name to `--rejection-reason` for consistency
2. ⚠️ Add integration test for rejection reason storage
3. ⚠️ Test rejection history display in `task get` output

### Low Priority (Nice to Have)

1. 💡 Add validation for empty rejection reasons
2. 💡 Document workflow config dependency
3. 💡 Add test for rejection reason with special characters

---

## Testing Blocked

**Cannot proceed with exploratory testing until compilation errors are fixed.**

**Next Steps:**
1. Developer fixes test compilation errors
2. QA re-runs test suite
3. QA performs manual exploratory testing
4. QA validates all edge cases
5. QA approves or rejects with detailed findings

---

## QA Notes

**Testing Limitations:**
- Could not run automated tests (compilation errors)
- Could not perform manual CLI testing (tests must pass first)
- Code review only - no functional validation

**Confidence Level:**
- Code review: High confidence (implementation looks correct)
- Functional testing: Zero confidence (not tested)
- Overall: **Cannot approve without functional validation**

**Estimated Re-Test Time:** 30 minutes (once compilation errors fixed)

---

## Sign-Off

**QA Agent:** QA
**Date:** 2026-01-17 00:56:00
**Status:** Exploratory testing blocked by compilation errors
**Recommendation:** Fix critical issues, then re-test
