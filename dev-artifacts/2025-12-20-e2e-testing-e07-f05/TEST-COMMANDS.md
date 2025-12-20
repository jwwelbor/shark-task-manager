# Manual E2E Test Commands Reference
## Document Repository CLI (shark related-docs)

This document contains all test commands executed during E2E testing, organized by category for easy reference and future re-execution.

---

## Test Environment Setup

### Initialize Clean Environment
```bash
mkdir -p /tmp/e2e-test-env
cd /tmp/e2e-test-env
rm -f shark-tasks.db shark-tasks.db-shm shark-tasks.db-wal

# Initialize shark project
/path/to/bin/shark init --non-interactive
```

### Create Test Structures
```bash
# Create epic
/path/to/bin/shark epic create "Test Epic"
# Result: E01

# Create feature
/path/to/bin/shark feature create --epic=E01 "Test Feature"
# Result: E01-F01-test-feature

# Create task
/path/to/bin/shark task create --epic=E01 --feature=E01-F01 "Test Task" --agent=backend
# Result: T-E01-F01-001
```

---

## ADD COMMAND TESTS

### Test 1: Add document to epic
```bash
/path/to/bin/shark related-docs add "OAuth Specification" "docs/oauth.md" --epic=E01
```
**Expected**: Document linked to epic E01
**Exit Code**: 0

### Test 2: Add document to feature
```bash
/path/to/bin/shark related-docs add "API Design Doc" "docs/api-design.md" --feature=E01-F01
```
**Expected**: Document linked to feature E01-F01
**Exit Code**: 0

### Test 3: Add document to task
```bash
/path/to/bin/shark related-docs add "Implementation Notes" "docs/impl-notes.md" --task=T-E01-F01-001
```
**Expected**: Document linked to task T-E01-F01-001
**Exit Code**: 0

### Test 4: Add with JSON output
```bash
/path/to/bin/shark related-docs add "JSON Test Doc" "docs/json-test.md" --epic=E01 --json
```
**Expected**: Valid JSON with fields: document_id, title, path, linked_to, parent_key
**Exit Code**: 0

### Test 5: Add with invalid epic (error)
```bash
/path/to/bin/shark related-docs add "Bad Epic Doc" "docs/bad.md" --epic=INVALID_EPIC
```
**Expected**: Error message starting with "Error: epic not found"
**Exit Code**: 1

### Test 6: Add with no parent flag
```bash
/path/to/bin/shark related-docs add "No Parent Doc" "docs/no-parent.md"
```
**Expected**: Usage message shown, no document created
**Exit Code**: 1 or 0 (depending on implementation)

### Test 7: Add with multiple parent flags
```bash
/path/to/bin/shark related-docs add "Multi Parent Doc" "docs/multi.md" --epic=E01 --feature=E01-F01
```
**Expected**: Usage message shown, no document created
**Exit Code**: 1 or 0 (depending on implementation)

### Test 8: Add duplicate document (idempotency)
```bash
# First add
/path/to/bin/shark related-docs add "DuplicateTest" "docs/dup-test.md" --epic=E01 --json
# Record document_id returned

# Second add (same title and path)
/path/to/bin/shark related-docs add "DuplicateTest" "docs/dup-test.md" --epic=E01 --json
```
**Expected**: Same document_id in both calls
**Exit Code**: 0 for both

### Test 9: Add with invalid feature
```bash
/path/to/bin/shark related-docs add "Test" "test.md" --feature=NONEXISTENT
```
**Expected**: Error message about feature not found
**Exit Code**: 1

### Test 10: Add with invalid task
```bash
/path/to/bin/shark related-docs add "Test" "test.md" --task=T-NONEXISTENT-001
```
**Expected**: Error message about task not found
**Exit Code**: 1

---

## LIST COMMAND TESTS

### Test 11: List documents for epic
```bash
/path/to/bin/shark related-docs list --epic=E01
```
**Expected**: Table format with documents linked to epic
```
Related Documents:
  - OAuth Specification (docs/oauth.md)
  - JSON Test Doc (docs/json-test.md)
```
**Exit Code**: 0

### Test 12: List documents for feature
```bash
/path/to/bin/shark related-docs list --feature=E01-F01
```
**Expected**: Table format with documents linked to feature
**Exit Code**: 0

### Test 13: List documents for task
```bash
/path/to/bin/shark related-docs list --task=T-E01-F01-001
```
**Expected**: Table format with documents linked to task
**Exit Code**: 0

### Test 14: List with JSON output
```bash
/path/to/bin/shark related-docs list --epic=E01 --json
```
**Expected**: JSON array with document objects containing: id, title, file_path, created_at
**Exit Code**: 0

### Test 15: List without parent flag (error)
```bash
/path/to/bin/shark related-docs list
```
**Expected**: Error message: "Error: one of --epic, --feature, or --task must be specified"
**Exit Code**: 1

### Test 16: List empty results
```bash
# For a parent with no documents
/path/to/bin/shark related-docs list --epic=E02
```
**Expected**: "No documents found" message or empty array
**Exit Code**: 0

---

## DELETE COMMAND TESTS

### Test 17: Delete document from epic
```bash
/path/to/bin/shark related-docs delete "OAuth Specification" --epic=E01
```
**Expected**: Document unlinked from epic (currently BROKEN - see BUG-001)
**Exit Code**: 0

### Test 18: Delete document from feature
```bash
/path/to/bin/shark related-docs delete "API Design Doc" --feature=E01-F01
```
**Expected**: Document unlinked from feature (currently BROKEN - see BUG-001)
**Exit Code**: 0

### Test 19: Delete document from task
```bash
/path/to/bin/shark related-docs delete "Implementation Notes" --task=T-E01-F01-001
```
**Expected**: Document unlinked from task (currently BROKEN - see BUG-001)
**Exit Code**: 0

### Test 20: Delete with JSON output
```bash
/path/to/bin/shark related-docs delete "Some Doc" --epic=E01 --json
```
**Expected**: Valid JSON with fields: status, title, parent
**Exit Code**: 0

### Test 21: Delete non-existent document (idempotent)
```bash
/path/to/bin/shark related-docs delete "NonExistentDoc123" --epic=E01
```
**Expected**: Returns success (delete is idempotent)
**Exit Code**: 0

### Test 22: Delete without parent flag (error)
```bash
/path/to/bin/shark related-docs delete "Some Doc"
```
**Expected**: Error message about parent flag required
**Exit Code**: 1

---

## HELP COMMAND TESTS

### Test 23: Help for add command
```bash
/path/to/bin/shark related-docs add --help
```
**Expected**: Usage and examples for add command

### Test 24: Help for delete command
```bash
/path/to/bin/shark related-docs delete --help
```
**Expected**: Usage and examples for delete command

### Test 25: Help for list command
```bash
/path/to/bin/shark related-docs list --help
```
**Expected**: Usage and examples for list command

### Test 26: Help for related-docs group
```bash
/path/to/bin/shark related-docs --help
```
**Expected**: Shows all subcommands (add, delete, list)

---

## DATABASE VERIFICATION TESTS

### Test 27: Query documents table
```bash
sqlite3 /tmp/e2e-test-env/shark-tasks.db "SELECT id, title, file_path FROM documents ORDER BY id;"
```
**Expected**: All created documents listed

### Test 28: Query epic_documents links
```bash
sqlite3 /tmp/e2e-test-env/shark-tasks.db "SELECT epic_id, document_id FROM epic_documents ORDER BY epic_id, document_id;"
```
**Expected**: Links between epics and documents

### Test 29: Query feature_documents links
```bash
sqlite3 /tmp/e2e-test-env/shark-tasks.db "SELECT feature_id, document_id FROM feature_documents ORDER BY feature_id, document_id;"
```
**Expected**: Links between features and documents

### Test 30: Query task_documents links
```bash
sqlite3 /tmp/e2e-test-env/shark-tasks.db "SELECT task_id, document_id FROM task_documents ORDER BY task_id, document_id;"
```
**Expected**: Links between tasks and documents

---

## ERROR MESSAGE VALIDATION TESTS

### Test 31: Invalid epic error message
```bash
/path/to/bin/shark related-docs add "Test" "test.md" --epic=NONEXISTENT 2>&1 | head -1
```
**Expected Output**: Error: epic not found: ...

### Test 32: Invalid feature error message
```bash
/path/to/bin/shark related-docs add "Test" "test.md" --feature=NONEXISTENT 2>&1 | head -1
```
**Expected Output**: Error: feature not found: ...

### Test 33: Invalid task error message
```bash
/path/to/bin/shark related-docs add "Test" "test.md" --task=T-NONEXISTENT 2>&1 | head -1
```
**Expected Output**: Error: task not found: ...

### Test 34: Missing title argument
```bash
/path/to/bin/shark related-docs add --epic=E01 "docs/test.md" 2>&1 | head -2
```
**Expected Output**: Error: accepts 2 arg(s), received 1

### Test 35: Missing path argument
```bash
/path/to/bin/shark related-docs add "Title Only" --epic=E01 2>&1 | head -2
```
**Expected Output**: Error: accepts 2 arg(s), received 1

---

## JSON OUTPUT VALIDATION TESTS

### Test 36: Validate add JSON structure
```bash
/path/to/bin/shark related-docs add "JSON Valid" "docs/json.md" --epic=E01 --json | python3 -m json.tool > /dev/null && echo "VALID"
```
**Expected Output**: VALID

### Test 37: Validate list JSON structure
```bash
/path/to/bin/shark related-docs list --epic=E01 --json | python3 -m json.tool > /dev/null && echo "VALID"
```
**Expected Output**: VALID

### Test 38: Validate delete JSON structure
```bash
/path/to/bin/shark related-docs delete "Any Doc" --epic=E01 --json | python3 -m json.tool > /dev/null && echo "VALID"
```
**Expected Output**: VALID

---

## REGRESSION TESTS (After Fix)

After fixing BUG-001, run these tests to verify no regressions:

### Test 39: Add still works after delete fix
```bash
/path/to/bin/shark related-docs add "Regression Test" "docs/regression.md" --epic=E01 --json
```
**Expected**: Document created successfully

### Test 40: Delete now actually works
```bash
# Add
/path/to/bin/shark related-docs add "Delete Test" "docs/delete-test.md" --epic=E01

# List before delete (should show it)
/path/to/bin/shark related-docs list --epic=E01 | grep "Delete Test"

# Delete
/path/to/bin/shark related-docs delete "Delete Test" --epic=E01

# List after delete (should NOT show it)
/path/to/bin/shark related-docs list --epic=E01 | grep "Delete Test"
```
**Expected**: First grep matches, second grep doesn't match

### Test 41: List still works after delete fix
```bash
/path/to/bin/shark related-docs list --epic=E01 --json
```
**Expected**: Valid JSON with remaining documents

---

## PERFORMANCE BASELINE TESTS

All commands should complete in < 100ms:

### Test 42: Add performance
```bash
time /path/to/bin/shark related-docs add "Perf Test" "docs/perf.md" --epic=E01
```
**Expected**: < 100ms

### Test 43: List performance
```bash
time /path/to/bin/shark related-docs list --epic=E01
```
**Expected**: < 50ms

### Test 44: Delete performance
```bash
time /path/to/bin/shark related-docs delete "Any Doc" --epic=E01
```
**Expected**: < 100ms

---

## Quick Test Suite

To run all tests quickly, use this bash script:

```bash
#!/bin/bash
SHARK=/path/to/bin/shark
cd /tmp/e2e-test-env

echo "1. Add to epic..."
$SHARK related-docs add "Doc1" "docs/1.md" --epic=E01

echo "2. Add to feature..."
$SHARK related-docs add "Doc2" "docs/2.md" --feature=E01-F01

echo "3. Add to task..."
$SHARK related-docs add "Doc3" "docs/3.md" --task=T-E01-F01-001

echo "4. List epic..."
$SHARK related-docs list --epic=E01

echo "5. List feature..."
$SHARK related-docs list --feature=E01-F01

echo "6. List task..."
$SHARK related-docs list --task=T-E01-F01-001

echo "7. JSON output..."
$SHARK related-docs list --epic=E01 --json | head -5

echo "8. Error handling..."
$SHARK related-docs add "Test" "test.md" --epic=INVALID

echo "All tests completed!"
```

---

## Known Issues

### BUG-001: Delete Command Non-Functional
The delete command returns success but doesn't actually remove document links from the database.

**Workaround**: None available via CLI. Database deletion must be done manually via SQL.

**Status**: Pending fix

---

## Documentation References

- Feature PRD: `/docs/plan/E07-enhancements/E07-F05-add-related-documents/prd.md`
- Implementation Plan: `/docs/plan/E07/E07-F05/MASTER-IMPLEMENTATION-PLAN.md`
- Full Test Report: `E2E-TEST-REPORT.md`
- Testing Summary: `TESTING-SUMMARY.md`

---

**Last Updated**: 2025-12-20
**Test Version**: 1.0
