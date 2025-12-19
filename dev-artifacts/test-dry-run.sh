#!/bin/bash
#
# Test script for T-E06-F05-003: Enhanced Sync Reporting and Dry-Run Mode
#

set -e

echo "=================================="
echo "Dry-Run Mode Validation Tests"
echo "=================================="
echo ""

# Create test directory
TEST_DIR=$(mktemp -d)
echo "Test directory: $TEST_DIR"

# Create test task file
mkdir -p "$TEST_DIR/docs/plan/E99-test/E99-F99-test-feature/tasks"
cat > "$TEST_DIR/docs/plan/E99-test/E99-F99-test-feature/tasks/T-E99-F99-001.md" <<EOF
---
task_key: T-E99-F99-001
title: Test Task for Dry Run
status: todo
---

# Task: Test Task for Dry Run

This is a test task to validate dry-run mode.
EOF

# Get initial task count
INITIAL_COUNT=$(./bin/shark task list --json 2>/dev/null | grep -o '"key"' | wc -l || echo "0")
echo "Initial task count in database: $INITIAL_COUNT"
echo ""

# Test 1: Run sync with --dry-run
echo "Test 1: Running sync with --dry-run flag..."
./bin/shark sync --folder="$TEST_DIR/docs/plan" --dry-run --create-missing
echo ""

# Verify no tasks were added
AFTER_DRY_RUN=$(./bin/shark task list --json 2>/dev/null | grep -o '"key"' | wc -l || echo "0")
echo "Task count after dry-run: $AFTER_DRY_RUN"

if [ "$INITIAL_COUNT" -eq "$AFTER_DRY_RUN" ]; then
    echo "✓ PASS: Dry-run did not add tasks to database"
else
    echo "✗ FAIL: Dry-run added tasks (expected $INITIAL_COUNT, got $AFTER_DRY_RUN)"
    exit 1
fi
echo ""

# Test 2: Run sync with --dry-run --json
echo "Test 2: Running sync with --dry-run --json..."
RESULT=$(./bin/shark sync --folder="$TEST_DIR/docs/plan" --dry-run --create-missing --json 2>&1)
DRY_RUN_FIELD=$(echo "$RESULT" | grep -o '"dry_run": true' || echo "missing")

if [ "$DRY_RUN_FIELD" = '"dry_run": true' ]; then
    echo "✓ PASS: JSON output includes dry_run: true"
else
    echo "✗ FAIL: JSON output missing or incorrect dry_run field"
    echo "Result: $RESULT"
    exit 1
fi
echo ""

# Test 3: Run real sync (without --dry-run)
echo "Test 3: Running real sync (without --dry-run)..."
./bin/shark sync --folder="$TEST_DIR/docs/plan" --create-missing
echo ""

# Verify task was added
AFTER_REAL_SYNC=$(./bin/shark task list --json 2>/dev/null | grep -o '"key"' | wc -l || echo "0")
echo "Task count after real sync: $AFTER_REAL_SYNC"

if [ "$AFTER_REAL_SYNC" -gt "$INITIAL_COUNT" ]; then
    echo "✓ PASS: Real sync added tasks to database"
else
    echo "✗ FAIL: Real sync did not add tasks"
    exit 1
fi
echo ""

# Cleanup
rm -rf "$TEST_DIR"

echo "=================================="
echo "All validation tests passed!"
echo "=================================="
