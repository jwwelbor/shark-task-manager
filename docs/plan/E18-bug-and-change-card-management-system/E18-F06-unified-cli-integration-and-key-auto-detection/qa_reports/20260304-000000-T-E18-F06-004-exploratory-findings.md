# Exploratory Findings: T-E18-F06-004 - Search Extension for Bugs and Change-Cards

**Date:** 2026-03-04
**Task:** T-E18-F06-004
**Feature:** E18-F06 - Unified CLI Integration and Key Auto-Detection
**Session Duration:** Full QA session
**Overall Finding:** No blocking issues. Two low-priority observations noted.

---

## Exploratory Charter

"Explore the search extension implementation to discover integration issues, edge case gaps, and consistency concerns between the search repository, CLI handler, and test fixtures."

---

## Findings

### FINDING-001: Test Fixture Key Format Inconsistency (Severity: Low)

**Observed:** `search_all_test.go` seeds change-card test data with key `"CC-001"`, while the E18-F06 feature spec defines the change-card key format as `C###` (e.g., `C001`, `C015`).

**Location:** `/home/jwwel/projects/shark-task-manager/internal/repository/search_all_test.go` line 67

**Impact:** None on functionality. Search matches on content (title/description), not key format. The key validation and generation are handled by the keys service layer, not the search repository. Tests pass correctly regardless of key format used in seed data.

**Recommendation:** In a future cleanup pass, align the test fixture key from `"CC-001"` to `"C001"` to be consistent with the feature spec's defined key format. This is a cosmetic consistency improvement, not a defect.

**Classification:** Low (cosmetic / test data consistency)

---

### FINDING-002: FTS5 Not Available in Test Environment (Severity: Informational)

**Observed:** During repository tests, the following warning is emitted:
```
Warning: FTS5 not available, skipping full-text search table: no such module: fts5
```

**Location:** `/home/jwwel/projects/shark-task-manager/internal/repository/search_all_test.go` - during DB setup

**Impact:** None. The FTS5 warning does not cause any test failures. The `SearchAll()` implementation uses LIKE-based pattern matching which does not depend on FTS5. The task spec explicitly allows this: "LIKE-based search is acceptable for initial implementation."

**Recommendation:** If FTS5 support is added in a future task, ensure the test environment's SQLite build includes the FTS5 extension. No action required for this task.

**Classification:** Informational only

---

### FINDING-003: Direct Repository Access in Search Command (Severity: Informational)

**Observed:** `search.go` calls `repository.SearchRepository` directly, bypassing the service layer pattern. A code comment acknowledges this as an acceptable deviation for the search command (no business logic beyond type validation).

**Location:** `/home/jwwel/projects/shark-task-manager/internal/cli/commands/search.go`

**Impact:** None on this task. Consistent with established patterns for legacy commands.

**Recommendation:** If the search command grows in complexity (e.g., adding result ranking, access control, or cross-entity joining logic), consider migrating to a `SearchService` in the service layer. Out of scope for T-E18-F06-004.

**Classification:** Informational (architecture observation)

---

## Edge Cases Verified

| Edge Case | Result |
|-----------|--------|
| Empty search query (no args) | Error returned: "search query is required" |
| No matches for valid query | Empty results returned gracefully (no panic, no error) |
| Valid type filter with no matches | Empty results, no error |
| Invalid type filter | Error with full list of valid types |
| Bug with empty severity field | Severity omitted from output (omitempty) |
| Mixed entity types in results | Each result correctly labeled by EntityType |

---

## Positive Quality Observations

1. Table-driven tests are used effectively in both `search_all_test.go` and `search_query_test.go`.
2. The `validSearchTypes` slice is a single source of truth used in both validation and the error message - no duplication.
3. The `omitempty` JSON tag on `Severity` correctly handles the bug-vs-non-bug case without explicit nil checks in the output formatter.
4. Repository tests use proper in-memory SQLite setup with full schema initialization and cleanup - no test pollution.
5. The UNION SQL correctly uses `'' AS severity` for non-bug entities, ensuring clean separation.

---

## Conclusion

No blocking issues found. The implementation is complete, correct, and well-tested. The two observations (test fixture key format and FTS5 note) are informational only and do not require remediation before advancing the task.
