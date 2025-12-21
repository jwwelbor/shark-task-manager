# Delete Command Bug Fix - START HERE

**Date:** 2025-12-20
**QA Agent:** Claude (Haiku 4.5)
**Status:** Phase 1 Complete - AWAITING DEVELOPER FIX

---

## What's The Problem?

The `shark related-docs delete` command doesn't actually delete document links from the database.

### Example
```bash
# User adds a document to epic E01
shark related-docs add "TestDoc1" docs/test1.md --epic=E01
# Result: Document linked successfully ✓

# User lists documents (should show TestDoc1)
shark related-docs list --epic=E01
# Result: TestDoc1 appears ✓

# User deletes the document
shark related-docs delete "TestDoc1" --epic=E01
# Result: Success message ✓ BUT...

# User lists documents again
shark related-docs list --epic=E01
# Result: TestDoc1 STILL appears! ✗ BUG!
```

---

## What Causes It?

**File:** `internal/cli/commands/related_docs.go`
**Function:** `runRelatedDocsDelete` (lines 218-340)
**Issue:** Stub implementation that validates the epic/feature/task exists but NEVER calls the delete/unlink methods

---

## What I've Prepared (Complete Analysis & Testing Materials)

### 8 Documents + 1 Test Script = 76 KB of Materials

**For Everyone:**
1. **INDEX.md** (9.7 KB) - Complete file map and navigation
2. **README.md** (7.1 KB) - Project overview and quick reference
3. **EXECUTION_SUMMARY.md** (8.9 KB) - Current status and next steps

**For Developers:**
4. **DEVELOPER_FIX_GUIDE.md** (12 KB) - Step-by-step implementation guide with code examples

**For QA Testing:**
5. **test-plan.md** (6.9 KB) - 11 detailed test cases (8 core + 3 regression)
6. **quick-test-script.sh** (4.1 KB) - Automated test execution (~2 minutes)

**For Reference:**
7. **issue-analysis.md** (2.2 KB) - Bug analysis and root cause
8. **status.md** (4.1 KB) - Timeline and developer handoff
9. **00-START-HERE.md** (this file) - Quick orientation

---

## What Happens Next?

### Phase 1: Analysis ✓ COMPLETE
- Identified bug location
- Analyzed root cause
- Prepared fix guide
- Designed test plan

**Status:** DONE

### Phase 2: Developer Implementation ⏳ IN PROGRESS
- Developer reads DEVELOPER_FIX_GUIDE.md
- Implements fix in `runRelatedDocsDelete`
- Runs local tests
- Signals "Fix is ready"

**Expected time:** 30-60 minutes

### Phase 3: QA Verification (PENDING)
- I run automated test script: `./quick-test-script.sh`
- Tests confirm delete actually removes documents
- Verify no regressions
- Sign off: APPROVED or REJECTED

**Expected time:** ~2 minutes to 2 hours

---

## For Developers: Next Steps

1. **Read:** `DEVELOPER_FIX_GUIDE.md`
   - Details on what needs fixing
   - Step-by-step implementation instructions
   - Code examples for all 3 handlers
   - Testing checklist before signaling ready

2. **Implement:**
   - Fix epic handler (add unlink call)
   - Fix feature handler (add unlink call)
   - Fix task handler (add unlink call)

3. **Test Locally:**
   - Run `make build` (should succeed)
   - Run `make test` (all should pass)
   - Manual test: add/delete/list
   - Verify document is gone after delete

4. **Signal Ready:**
   - Commit changes
   - Say "Fix is ready for verification"

---

## For QA: Ready to Execute

When developer signals "ready":

**Option A: Automated (Fast)**
```bash
cd /home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-20-delete-command-verification
./quick-test-script.sh
```
**Takes:** ~2 minutes
**Output:** "FINAL RESULT: ALL TESTS PASSED" (if fix works)

**Option B: Manual (Detailed)**
- Follow test-plan.md
- Execute 8 core tests
- Run 3 regression tests
- Document results

**Takes:** ~30-60 minutes

---

## Key Test: Delete from Epic

This test validates the core bug fix:

```bash
# Add document to epic
shark related-docs add "TestDoc1" docs/test1.md --epic=E01

# Verify it appears
shark related-docs list --epic=E01
# Output includes TestDoc1 ✓

# DELETE IT
shark related-docs delete "TestDoc1" --epic=E01

# Verify it's GONE (CRITICAL TEST)
shark related-docs list --epic=E01
# Output should NOT include TestDoc1
```

**This is THE test** that proves the fix works.

---

## Quick Navigation

**You are here:** `00-START-HERE.md` (orientation)

**Next:**
- **Developers:** Go to `DEVELOPER_FIX_GUIDE.md`
- **QA:** Go to `test-plan.md` or `quick-test-script.sh`
- **Everyone:** Read `README.md` or `EXECUTION_SUMMARY.md`

**Complete Map:** See `INDEX.md` for all files and navigation

---

## File Locations

**Workspace:**
```
/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-20-delete-command-verification/
```

**Bug Location:**
```
/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/related_docs.go
Lines: 218-340 (function: runRelatedDocsDelete)
```

**Repository Methods Needed:**
```
/home/jwwelbor/projects/shark-task-manager/internal/repository/document_repository.go
Methods: UnlinkFromEpic, UnlinkFromFeature, UnlinkFromTask (lines 165-199)
```

---

## The Fix in 30 Seconds

**What's Missing:**
The `runRelatedDocsDelete` function doesn't call the unlink methods.

**What's Needed:**
For each parent type (epic, feature, task):
1. Get the parent entity ✓ (already done)
2. Find the document by title (needs implementation)
3. Call `docRepo.UnlinkFromEpic/Feature/Task()` (MISSING)
4. Return success

**Example:**
```go
// Instead of just returning success, actually unlink:
if err := docRepo.UnlinkFromEpic(ctx, e.ID, doc.ID); err != nil {
    return fmt.Errorf("failed to unlink: %w", err)
}
```

**Full guide:** See DEVELOPER_FIX_GUIDE.md

---

## Quality Gate Decision

When testing complete:

**APPROVED When:**
- ✓ Delete removes documents from list (TC-02 passes)
- ✓ All other tests pass
- ✓ No regressions
- ✓ Build succeeds

**REJECTED When:**
- ✗ Documents still appear after delete
- ✗ Any test fails
- ✗ Build breaks

---

## Status at a Glance

| Item | Status | Details |
|------|--------|---------|
| Bug Analysis | ✓ COMPLETE | Root cause identified |
| Test Plan | ✓ COMPLETE | 11 tests designed |
| Test Script | ✓ COMPLETE | Automated execution ready |
| Developer Guide | ✓ COMPLETE | Implementation instructions ready |
| Developer Implementation | ⏳ WAITING | Awaiting developer |
| QA Testing | PENDING | Ready to execute once fix ready |
| Quality Gate | PENDING | Pending test results |

---

## Three Simple Steps to Completion

1. **Developer:** Implement fix in `runRelatedDocsDelete` (30 min)
2. **QA:** Run `./quick-test-script.sh` (2 min)
3. **Everyone:** Done! (Approve or reject based on results)

---

## Questions?

**What's the bug?**
- Delete command returns success but doesn't remove document links from database

**Why?**
- Stub implementation missing calls to `docRepo.UnlinkFromEpic/Feature/Task()`

**How to fix?**
- Read DEVELOPER_FIX_GUIDE.md for step-by-step instructions

**How to verify?**
- Run test-plan.md tests or use quick-test-script.sh

**How long?**
- Developer: 30-60 min
- QA: 2 min (automated) or 30 min (manual)

---

## Next Step

**Choose your role:**

- **I'm the Developer:** Read `DEVELOPER_FIX_GUIDE.md` and implement the fix
- **I'm QA:** Read `test-plan.md` and prepare to execute tests
- **I want overview:** Read `README.md` or `EXECUTION_SUMMARY.md`
- **I want full map:** Read `INDEX.md`

---

## Bottom Line

✓ Complete bug analysis
✓ Comprehensive test plan ready
✓ Automated test script ready
✓ Developer implementation guide ready

⏳ **Awaiting developer to implement fix**

Once fix complete: QA will test and sign off (APPROVED or REJECTED)

---

**Created:** 2025-12-20
**By:** Claude QA Agent (Haiku 4.5)
**Status:** Ready for Developer Implementation & QA Verification
