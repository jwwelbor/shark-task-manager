#!/bin/bash
set -e

echo "=== Testing Task History Display with Rejection Reasons ==="
echo ""

# Create a test task (starts in draft)
echo "1. Creating test task..."
TASK_OUTPUT=$(./bin/shark task create E07 F22 "Test task for history display" --agent=backend --json)
TASK_KEY=$(echo "$TASK_OUTPUT" | jq -r '.task.key')
echo "Created task: $TASK_KEY (draft)"
echo ""

# Move through workflow to in_code_review
echo "2. Moving task through workflow..."
./bin/shark task update "$TASK_KEY" --status=ready_for_refinement --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_refinement --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_development --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_development --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_code_review --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_code_review --json > /dev/null
echo "Task in code review"
echo ""

# Reject from code review using task update with --reason (backward transition)
echo "3. Rejecting from code review with reason..."
./bin/shark task update "$TASK_KEY" --status=in_development \
  --reason="Missing error handling on line 67. Add null check." \
  --json > /dev/null
echo "Task rejected with reason"
echo ""

# Move back to QA
echo "4. Moving to QA..."
./bin/shark task update "$TASK_KEY" --status=ready_for_code_review --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_code_review --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_qa --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_qa --json > /dev/null
echo "Task in QA"
echo ""

# Reject from QA with reason (backward transition)
# Note: --reason-doc is not supported by task update, only by task reopen/approve
echo "5. Rejecting from QA with reason..."
./bin/shark task update "$TASK_KEY" --status=in_development \
  --reason="Found 3 issues during QA testing." \
  --json > /dev/null
echo "Task rejected with reason"
echo ""

# Get task history - Terminal format
echo "=== Terminal Format Output ==="
echo ""
./bin/shark task get "$TASK_KEY"
echo ""

# Get task history - JSON format
echo "=== JSON Format Output ==="
echo ""
TASK_JSON=$(./bin/shark task get "$TASK_KEY" --json)
echo "$TASK_JSON" | jq '.rejection_history'
echo ""

# Validate JSON structure
echo "=== Validation Results ==="
echo ""

# Check rejection_history field exists
if echo "$TASK_JSON" | jq -e '.rejection_history' > /dev/null; then
  echo "✅ rejection_history field present in JSON"
else
  echo "❌ rejection_history field missing in JSON"
  exit 1
fi

# Check rejection_history is an array
if echo "$TASK_JSON" | jq -e '.rejection_history | type == "array"' > /dev/null; then
  echo "✅ rejection_history is an array"
else
  echo "❌ rejection_history is not an array"
  exit 1
fi

# Check we have 2 rejections
REJECTION_COUNT=$(echo "$TASK_JSON" | jq '.rejection_history | length')
if [ "$REJECTION_COUNT" -eq 2 ]; then
  echo "✅ Found 2 rejection records"
else
  echo "❌ Expected 2 rejections, found $REJECTION_COUNT"
  exit 1
fi

# Check first rejection has required fields
echo ""
echo "Checking first rejection..."
FIRST_REJECTION=$(echo "$TASK_JSON" | jq '.rejection_history[0]')
for field in "id" "timestamp" "from_status" "to_status" "reason"; do
  if echo "$FIRST_REJECTION" | jq -e ".$field" > /dev/null; then
    VALUE=$(echo "$FIRST_REJECTION" | jq -r ".$field")
    echo "✅ First rejection has field: $field = $VALUE"
  else
    echo "❌ First rejection missing field: $field"
    exit 1
  fi
done

# Check rejection reason content
FIRST_REASON=$(echo "$TASK_JSON" | jq -r '.rejection_history[0].reason')
if echo "$FIRST_REASON" | grep -q "Missing error handling"; then
  echo "✅ Rejection reason contains expected text"
else
  echo "❌ Rejection reason missing expected content: $FIRST_REASON"
  exit 1
fi

# Check second rejection has required fields
echo ""
echo "Checking second rejection..."
SECOND_REJECTION=$(echo "$TASK_JSON" | jq '.rejection_history[1]')
for field in "id" "timestamp" "from_status" "to_status" "reason"; do
  if echo "$SECOND_REJECTION" | jq -e ".$field" > /dev/null; then
    VALUE=$(echo "$SECOND_REJECTION" | jq -r ".$field")
    echo "✅ Second rejection has field: $field = $VALUE"
  else
    echo "❌ Second rejection missing field: $field"
    exit 1
  fi
done

SECOND_REASON=$(echo "$TASK_JSON" | jq -r '.rejection_history[1].reason')
if echo "$SECOND_REASON" | grep -q "Found 3 issues"; then
  echo "✅ Second rejection reason contains expected text"
else
  echo "❌ Second rejection reason missing expected content: $SECOND_REASON"
  exit 1
fi

# Cleanup
echo ""
echo "6. Cleaning up test task..."
./bin/shark db exec "DELETE FROM tasks WHERE key = '$TASK_KEY'" > /dev/null
echo "Test task deleted"
echo ""

echo "=== All Tests Passed ==="
