#!/bin/bash
set -e

echo "=== Testing Rejection History Display ==="
echo ""

# Use an existing feature
EPIC="E07"
FEATURE="F05"

# Create a test task
echo "1. Creating test task..."
TASK_OUTPUT=$(./bin/shark task create "$EPIC" "$FEATURE" "Test rejection history display" --agent=backend --json)
TASK_KEY=$(echo "$TASK_OUTPUT" | jq -r '.task.key')
echo "Created task: $TASK_KEY"
echo ""

# Move to in_approval
echo "2. Moving task to in_approval..."
./bin/shark task update "$TASK_KEY" --status=ready_for_refinement --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_refinement --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_development --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_development --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_code_review --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_code_review --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_qa --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_qa --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_approval --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_approval --json > /dev/null
echo "Task at in_approval"
echo ""

# Reject with reason (backward transition: in_approval → ready_for_development)
echo "3. Rejecting with reason (in_approval → ready_for_development)..."
./bin/shark task update "$TASK_KEY" --status=ready_for_development \
  --reason="Missing error handling on line 67" \
  --json > /dev/null
echo "Task rejected"
echo ""

# Move back to approval
echo "4. Moving back to in_approval..."
./bin/shark task update "$TASK_KEY" --status=in_development --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_code_review --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_code_review --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_qa --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_qa --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=ready_for_approval --json > /dev/null
./bin/shark task update "$TASK_KEY" --status=in_approval --json > /dev/null
echo "Task back at in_approval"
echo ""

# Reject again with different reason
echo "5. Rejecting again (in_approval → ready_for_qa)..."
./bin/shark task update "$TASK_KEY" --status=ready_for_qa \
  --reason="Found 3 critical bugs during final approval review" \
  --json > /dev/null
echo "Task rejected to QA"
echo ""

# Display task history in terminal format
echo "=== Terminal Format Output ===" 
echo ""
./bin/shark task get "$TASK_KEY"
echo ""

# Get JSON format
echo "=== JSON Format Output ==="
echo ""
TASK_JSON=$(./bin/shark task get "$TASK_KEY" --json)
echo "$TASK_JSON" | jq '.rejection_history'
echo ""

# Validate
echo "=== Validation Results ==="
echo ""

# Check field exists
if echo "$TASK_JSON" | jq -e '.rejection_history' > /dev/null; then
  echo "✅ rejection_history field exists"
else
  echo "❌ rejection_history field missing"
  exit 1
fi

# Check is array
if echo "$TASK_JSON" | jq -e '.rejection_history | type == "array"' > /dev/null; then
  echo "✅ rejection_history is an array"
else
  echo "❌ rejection_history is not an array"
  exit 1
fi

# Check count
COUNT=$(echo "$TASK_JSON" | jq '.rejection_history | length')
if [ "$COUNT" -eq 2 ]; then
  echo "✅ Found 2 rejection records"
else
  echo "❌ Expected 2 rejections, found $COUNT"
  exit 1
fi

# Check first rejection
echo ""
echo "First rejection:"
FIRST=$(echo "$TASK_JSON" | jq '.rejection_history[0]')
for field in "id" "timestamp" "from_status" "to_status" "reason"; do
  if echo "$FIRST" | jq -e ".$field" > /dev/null; then
    VALUE=$(echo "$FIRST" | jq -r ".$field")
    echo "✅ Has $field: $VALUE"
  else
    echo "❌ Missing $field"
    exit 1
  fi
done

# Check second rejection
echo ""
echo "Second rejection:"
SECOND=$(echo "$TASK_JSON" | jq '.rejection_history[1]')
for field in "id" "timestamp" "from_status" "to_status" "reason"; do
  if echo "$SECOND" | jq -e ".$field" > /dev/null; then
    VALUE=$(echo "$SECOND" | jq -r ".$field")
    echo "✅ Has $field: $VALUE"
  else
    echo "❌ Missing $field"
    exit 1
  fi
done

# Cleanup
echo ""
echo "6. Cleaning up test task..."
sqlite3 shark-tasks.db "DELETE FROM tasks WHERE key = '$TASK_KEY'"
echo "Test task deleted"
echo ""

echo "=== All Tests Passed ==="
