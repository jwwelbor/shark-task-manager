# Delete Command Test Plan

**Status:** Ready to Execute Once Fix is Applied
**Date:** 2025-12-20

## Test Environment Setup

### Prerequisites
- Fresh build: `make build`
- Test directory: `/tmp/e2e-test-env`
- Fresh database (will be initialized via `shark init`)

### Test Data Setup Script

```bash
#!/bin/bash
# Setup test environment
cd /tmp
rm -rf e2e-test-env
mkdir -p e2e-test-env
cd e2e-test-env

# Initialize shark project
/home/jwwelbor/projects/shark-task-manager/bin/shark init --non-interactive

# Create test epic E01
/home/jwwelbor/projects/shark-task-manager/bin/shark epic create --title="Test Epic E01" --json

# Create test feature E01-F01
/home/jwwelbor/projects/shark-task-manager/bin/shark feature create --epic=E01 --title="Test Feature F01" --json

# Create test task T-E01-F01-001
/home/jwwelbor/projects/shark-task-manager/bin/shark task create \
  --epic=E01 \
  --feature=E01-F01 \
  --title="Test Task 001" \
  --agent=developer \
  --json
```

## Core Test Cases

### TC-01: Add Document to Epic

**Steps:**
1. Run: `shark related-docs add "TestDoc1" docs/test1.md --epic=E01`
2. Expected: Document added successfully
3. Verify: `shark related-docs list --epic=E01`
4. Expected: TestDoc1 appears in list

**Validation:**
- Command returns success (exit code 0)
- Document title and path correct in list output
- Database shows epic_documents link created

---

### TC-02: Delete Document from Epic (KEY TEST)

**Steps:**
1. Prerequisite: TC-01 completed (TestDoc1 linked to E01)
2. Run: `shark related-docs delete "TestDoc1" --epic=E01`
3. Expected: Delete succeeds (exit code 0)
4. Run: `shark related-docs list --epic=E01`
5. **CRITICAL:** Expected: TestDoc1 NOT in list

**Validation:**
- Delete command returns success
- List shows empty or without TestDoc1
- Database query confirms epic_documents link removed:
  ```sql
  SELECT COUNT(*) FROM epic_documents WHERE epic_id=1 AND document_id=1;
  -- Should return 0
  ```

**Pass Criteria:** List after delete does NOT show TestDoc1

---

### TC-03: Delete Document from Feature

**Steps:**
1. Run: `shark related-docs add "FeatureDoc" docs/feature.md --feature=E01-F01`
2. Verify: `shark related-docs list --feature=E01-F01` shows FeatureDoc
3. Run: `shark related-docs delete "FeatureDoc" --feature=E01-F01`
4. Verify: `shark related-docs list --feature=E01-F01` does NOT show FeatureDoc

**Validation:**
- Document successfully unlinked from feature
- Database confirms feature_documents link removed

---

### TC-04: Delete Document from Task

**Steps:**
1. Run: `shark related-docs add "TaskDoc" docs/task.md --task=T-E01-F01-001`
2. Verify: `shark related-docs list --task=T-E01-F01-001` shows TaskDoc
3. Run: `shark related-docs delete "TaskDoc" --task=T-E01-F01-001`
4. Verify: `shark related-docs list --task=T-E01-F01-001` does NOT show TaskDoc

**Validation:**
- Document successfully unlinked from task
- Database confirms task_documents link removed

---

### TC-05: Idempotent Delete (Non-existent Document)

**Steps:**
1. Run: `shark related-docs delete "NonExistent" --epic=E01`
2. Expected: Command succeeds (exit code 0)
3. No error thrown

**Validation:**
- Delete is idempotent as per design
- No database corruption
- No error messages

---

### TC-06: Idempotent Delete (Already Deleted)

**Steps:**
1. Prerequisites: TC-02 completed (TestDoc1 deleted from E01)
2. Run: `shark related-docs delete "TestDoc1" --epic=E01`
3. Expected: Command succeeds (exit code 0)

**Validation:**
- Second delete succeeds
- No error messages
- No database side effects

---

### TC-07: JSON Output Validation

**Steps:**
1. Run: `shark related-docs delete "TestDoc1" --epic=E01 --json`
2. Expected: Valid JSON output

**Validation:**
- Output is valid JSON
- Contains expected fields: status, title, parent
- Example output:
  ```json
  {
    "status": "unlinked",
    "title": "TestDoc1",
    "parent": "epic"
  }
  ```

---

### TC-08: Multiple Documents on Same Parent

**Steps:**
1. Add "Doc1" to E01
2. Add "Doc2" to E01
3. List E01: should show Doc1 and Doc2
4. Delete Doc1 from E01
5. List E01: should show only Doc2
6. Delete Doc2 from E01
7. List E01: should be empty

**Validation:**
- Selective deletion works correctly
- Only target document is deleted
- Other documents remain linked

---

## Regression Test Suite

### RT-01: Unit Tests

**Steps:**
1. Run: `make test`

**Expected:**
- All 12 unit tests pass
- No new test failures
- No regressions in other commands

**Validation:**
- Test output shows all PASS
- Exit code 0

---

### RT-02: Other Commands Still Work

**Steps:**
1. Epic operations: `shark epic list --json`
2. Feature operations: `shark feature list --epic=E01 --json`
3. Task operations: `shark task list --epic=E01 --json`
4. Document add: `shark related-docs add "Doc" path --epic=E01`
5. Document list: `shark related-docs list --epic=E01`

**Validation:**
- No commands broken
- All JSON outputs valid
- No database corruption

---

### RT-03: Build Verification

**Steps:**
1. Run: `make clean`
2. Run: `make build`

**Expected:**
- Build succeeds
- Binary created at `bin/shark`
- No compilation errors

**Validation:**
- Exit code 0
- Binary is executable

---

## Database Validation

### Query to Verify Delete State

```bash
# Check epic documents links
sqlite3 /tmp/e2e-test-env/shark-tasks.db "SELECT * FROM epic_documents;"

# Check feature documents links
sqlite3 /tmp/e2e-test-env/shark-tasks.db "SELECT * FROM feature_documents;"

# Check task documents links
sqlite3 /tmp/e2e-test-env/shark-tasks.db "SELECT * FROM task_documents;"

# Check documents still exist (not deleted from documents table)
sqlite3 /tmp/e2e-test-env/shark-tasks.db "SELECT * FROM documents;"
```

---

## Test Results Template

```markdown
# Test Results - 2025-12-20

## Core Tests
- [ ] TC-01: Add Document to Epic - PASS/FAIL
- [ ] TC-02: Delete Document from Epic (KEY) - PASS/FAIL
- [ ] TC-03: Delete Document from Feature - PASS/FAIL
- [ ] TC-04: Delete Document from Task - PASS/FAIL
- [ ] TC-05: Idempotent Delete (Non-existent) - PASS/FAIL
- [ ] TC-06: Idempotent Delete (Already Deleted) - PASS/FAIL
- [ ] TC-07: JSON Output Validation - PASS/FAIL
- [ ] TC-08: Multiple Documents Selective Delete - PASS/FAIL

## Regression Tests
- [ ] RT-01: Unit Tests (make test) - PASS/FAIL
- [ ] RT-02: Other Commands - PASS/FAIL
- [ ] RT-03: Build Verification - PASS/FAIL

## Database Validation
- [ ] epic_documents links verified
- [ ] feature_documents links verified
- [ ] task_documents links verified
- [ ] documents table intact

## Overall Quality Gate: PASS/FAIL
```

---

## Execution Instructions

When developer signals fix is ready:

1. Build fresh binary: `make build`
2. Execute test setup script above
3. Run test cases TC-01 through TC-08
4. Verify database state with SQL queries
5. Run regression tests RT-01 through RT-03
6. Document all results
7. Sign off on quality gate

**Expected:** All tests PASS, quality gate approved
