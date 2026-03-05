# Exploratory Findings: T-E18-F06-007 - Fix 4 Dispatch Gaps for C### Keys

**Date:** 2026-03-04 12:49:35
**Task ID:** T-E18-F06-007
**QA Run:** 20260304-124935
**Charter:** Explore dispatch routing for C### keys to discover any remaining edge cases or consistency issues

---

## Summary

No blocking issues found. One design pattern was initially flagged as suspicious but confirmed intentional. Two minor observations documented for awareness.

---

## Findings

### Finding F-001: `getChangeCardService()` Pattern (RESOLVED - Intentional Design)

**Severity:** None (confirmed intentional)

**Observation:** In `status_group.go`, the `case "change":` branches use `getChangeCardService()` (package-local, lowercase) while the `case "change_card":` branches use `cli.GetChangeCardService()` (global accessor, uppercase). This initial discrepancy raised a question about consistency.

**Investigation Result:** `getChangeCardService()` is defined in `internal/cli/commands/change.go` as a testable wrapper function that enables mock injection during tests. In production it delegates to `cli.GetChangeCardService()`. This is the same testability pattern used by `getBugService()` for bug entity dispatch. The difference between "change" and "change_card" cases using different accessor forms is intentional: "change" case tests exercise mock injection while "change_card" cases call the global accessor directly. Both resolve to the same service in production.

**Conclusion:** No issue. This is established codebase pattern.

---

### Finding F-002: Error Message Improvement in delete_dispatch.go (LOW - Positive Change)

**Severity:** Low (improvement observed)

**Observation:** The default error message in `delete_dispatch.go` was updated as part of this fix to include `C### (change card)` in the format hint:

```
Expected format: E## (epic), E##-F## (feature), E##-F##-### (task), B### (bug), C### (change card), or CC-### (change-card)
```

**Assessment:** This is a positive side effect. Users who accidentally pass an unrecognized key format (e.g., a typo) now receive a more informative error message that includes change card key formats in the hint.

---

### Finding F-003: dispatch inventory N/A entries (INFORMATIONAL)

**Severity:** None (informational)

**Observation:** `TestDispatchInventory_FullStatus` reports 5 N/A entries in addition to 28 HANDLED and 0 GAPs. The N/A entries represent dispatch points that are intentionally not applicable (e.g., operations not supported for certain entity types, or dispatch functions that use different routing mechanisms).

**Assessment:** The 5 N/A entries are expected and correct. The task specification does not require all 33 inventory points to be HANDLED - only the 4 previously-GAP entries needed to change to HANDLED.

---

## Exploratory Testing Areas Covered

### CRUD Dispatch Coverage
- Delete dispatch: C### correctly routes, no fall-through to error
- Status set dispatch: C### correctly routes to SetChangeCardStatus
- Status options dispatch: C### correctly routes to GetChangeCard for transitions
- Status advance dispatch: C### correctly routes to AdvanceChangeCardStatus

### Edge Cases Tested
- `dispatchAdvance()` default `return nil, nil` preserved (EC-005)
- `dispatchTransition()` with "change" vs "change_card" keys both handled
- EntityType string in results consistently "change" for C### keys (BR-002)

### Regression Areas Checked
- Epic dispatch paths: All existing cases unchanged
- Feature dispatch paths: All existing cases unchanged
- Task dispatch paths: All existing cases unchanged
- Bug dispatch paths: All B### cases unchanged
- Change_card CC-### paths: All existing cases unchanged

---

## Conclusion

The implementation is clean and correct. The 4 dispatch gaps are resolved with no side effects on existing functionality. The testability pattern is consistent with established codebase conventions. No remediation required.

---

## QA Agent: claude-sonnet-4-6
