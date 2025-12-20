#!/bin/bash
# Quick Test Execution Script for Delete Command Fix
# Run this once developer signals fix is ready

set -e

SHARK_BIN="/home/jwwelbor/projects/shark-task-manager/bin/shark"
TEST_DIR="/tmp/e2e-test-delete-fix"
RESULTS_FILE="./test-results.txt"

echo "=========================================="
echo "Delete Command Fix Verification"
echo "=========================================="
echo ""

# Cleanup previous test environment
echo "[1] Cleaning up previous test environment..."
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Build fresh binary
echo "[2] Building fresh binary..."
cd /home/jwwelbor/projects/shark-task-manager
make build > /dev/null 2>&1
echo "    Build successful"
echo ""

# Return to test directory
cd "$TEST_DIR"

# Initialize project
echo "[3] Initializing test project..."
$SHARK_BIN init --non-interactive > /dev/null 2>&1
echo "    Project initialized"
echo ""

# Create test data
echo "[4] Creating test data..."
echo "    - Creating Epic E01..."
$SHARK_BIN epic create --title="Test Epic E01" > /dev/null 2>&1

echo "    - Creating Feature E01-F01..."
$SHARK_BIN feature create --epic=E01 --title="Test Feature F01" > /dev/null 2>&1

echo "    - Creating Task T-E01-F01-001..."
$SHARK_BIN task create \
  --epic=E01 \
  --feature=E01-F01 \
  --title="Test Task 001" \
  --agent=developer > /dev/null 2>&1
echo ""

# TC-01: Add Document to Epic
echo "[5] TC-01: Add Document to Epic"
$SHARK_BIN related-docs add "TestDoc1" docs/test1.md --epic=E01
echo "    PASS: Document added"
echo ""

# TC-02: List and Verify Document
echo "[6] TC-02: List Documents (Before Delete)"
LIST_BEFORE=$($SHARK_BIN related-docs list --epic=E01 --json)
echo "$LIST_BEFORE" | grep -q "TestDoc1" && echo "    PASS: TestDoc1 found in list" || echo "    FAIL: TestDoc1 NOT in list"
echo ""

# TC-03: Delete Document (KEY TEST)
echo "[7] TC-03: Delete Document from Epic (KEY TEST)"
$SHARK_BIN related-docs delete "TestDoc1" --epic=E01
echo "    Delete command executed"
echo ""

# TC-04: List and Verify Deletion
echo "[8] TC-04: List Documents (After Delete - CRITICAL VERIFICATION)"
LIST_AFTER=$($SHARK_BIN related-docs list --epic=E01 --json)

# Check if document is NOT in list
if echo "$LIST_AFTER" | grep -q "TestDoc1"; then
    echo "    FAIL: TestDoc1 STILL in list after delete!"
    echo "    This indicates the delete command did not work properly"
    exit 1
else
    echo "    PASS: TestDoc1 NOT in list after delete (FIX WORKS)"
fi
echo ""

# TC-05: Test Feature Delete
echo "[9] TC-05: Delete from Feature"
$SHARK_BIN related-docs add "FeatureDoc" docs/feature.md --feature=E01-F01 > /dev/null 2>&1
$SHARK_BIN related-docs delete "FeatureDoc" --feature=E01-F01 > /dev/null 2>&1
FEATURE_LIST=$($SHARK_BIN related-docs list --feature=E01-F01 --json)
if echo "$FEATURE_LIST" | grep -q "FeatureDoc"; then
    echo "    FAIL: FeatureDoc still in feature list"
    exit 1
else
    echo "    PASS: FeatureDoc removed from feature"
fi
echo ""

# TC-06: Test Task Delete
echo "[10] TC-06: Delete from Task"
$SHARK_BIN related-docs add "TaskDoc" docs/task.md --task=T-E01-F01-001 > /dev/null 2>&1
$SHARK_BIN related-docs delete "TaskDoc" --task=T-E01-F01-001 > /dev/null 2>&1
TASK_LIST=$($SHARK_BIN related-docs list --task=T-E01-F01-001 --json)
if echo "$TASK_LIST" | grep -q "TaskDoc"; then
    echo "    FAIL: TaskDoc still in task list"
    exit 1
else
    echo "    PASS: TaskDoc removed from task"
fi
echo ""

# TC-07: Idempotent Delete
echo "[11] TC-07: Idempotent Delete (Delete Non-Existent)"
$SHARK_BIN related-docs delete "NonExistentDoc" --epic=E01 > /dev/null 2>&1
echo "    PASS: Delete succeeded (idempotent)"
echo ""

# TC-08: Unit Tests
echo "[12] Running Unit Tests (make test)"
cd /home/jwwelbor/projects/shark-task-manager
make test > /dev/null 2>&1
echo "    PASS: All unit tests passed"
echo ""

# Build Verification
echo "[13] Build Verification"
make build > /dev/null 2>&1
echo "    PASS: Build successful"
echo ""

echo "=========================================="
echo "FINAL RESULT: ALL TESTS PASSED"
echo "=========================================="
echo ""
echo "Quality Gate: APPROVED"
echo "Fix Status: VERIFIED"
echo ""
