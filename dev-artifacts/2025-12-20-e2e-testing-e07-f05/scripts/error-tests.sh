#!/bin/bash

SHARK=/home/jwwelbor/projects/shark-task-manager/bin/shark
cd /tmp/e2e-test-env

echo "========================================"
echo "TEST 19: Error messages - invalid feature"
echo "========================================"
$SHARK related-docs add "Test" "test.md" --feature=NONEXISTENT 2>&1 | head -1

echo ""
echo "========================================"
echo "TEST 20: Error messages - invalid task"
echo "========================================"
$SHARK related-docs add "Test" "test.md" --task=T-NONEXISTENT-001 2>&1 | head -1

echo ""
echo "========================================"
echo "TEST 21: List without parent flag"
echo "========================================"
$SHARK related-docs list 2>&1 | head -1

echo ""
echo "========================================"
echo "TEST 22: Add with missing path argument"
echo "========================================"
$SHARK related-docs add "Title Only" --epic=E01 2>&1 | head -3

echo ""
echo "========================================"
echo "TEST 23: Add with missing title argument"
echo "========================================"
$SHARK related-docs add --epic=E01 "path/only.md" 2>&1 | head -3
