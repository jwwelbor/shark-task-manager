#!/bin/bash
# Test script to verify CLI commands accept both numeric and slugged keys

set -e

SHARK="./bin/shark"

echo "=== Testing CLI Dual Key Support ==="
echo

# Test epic get with both key formats
echo "1. Testing epic get with numeric key..."
$SHARK epic get E07 --json > /dev/null 2>&1 && echo "✓ Epic get with numeric key (E07) works" || echo "✗ Epic get with numeric key failed"

echo "2. Testing epic get with slugged key..."
$SHARK epic get E07-enhancements --json > /dev/null 2>&1 && echo "✓ Epic get with slugged key (E07-enhancements) works" || echo "✗ Epic get with slugged key failed"

# Test feature get with both key formats
echo "3. Testing feature get with numeric key..."
$SHARK feature get E07-F11 --json > /dev/null 2>&1 && echo "✓ Feature get with numeric key (E07-F11) works" || echo "✗ Feature get with numeric key failed"

echo "4. Testing feature get with slugged key..."
$SHARK feature get E07-F11-slug-architecture-improvement --json > /dev/null 2>&1 && echo "✓ Feature get with slugged key works" || echo "✗ Feature get with slugged key failed"

# Test task get with both key formats
echo "5. Testing task get with numeric key..."
$SHARK task get T-E07-F11-014 --json > /dev/null 2>&1 && echo "✓ Task get with numeric key (T-E07-F11-014) works" || echo "✗ Task get with numeric key failed"

echo "6. Testing task get with slugged key..."
# Find a task with a slug from database
TASK_SLUG=$(sqlite3 shark-tasks.db "SELECT slug FROM tasks WHERE key = 'T-E07-F11-014' LIMIT 1")
if [ -n "$TASK_SLUG" ]; then
    $SHARK task get "T-E07-F11-014-$TASK_SLUG" --json > /dev/null 2>&1 && echo "✓ Task get with slugged key works" || echo "✗ Task get with slugged key failed"
else
    echo "⚠ No slug found for task T-E07-F11-014, skipping slugged key test"
fi

echo
echo "=== Test Complete ==="
