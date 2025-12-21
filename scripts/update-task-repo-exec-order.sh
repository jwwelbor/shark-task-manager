#!/bin/bash

# Script to update task_repository.go to include execution_order field

FILE="internal/repository/task_repository.go"

# Backup the file
cp "$FILE" "$FILE.bak"

# Add execution_order to SELECT clauses (add after blocked_reason)
sed -i 's/blocked_reason,$/blocked_reason, execution_order,/g' "$FILE"

# Add execution_order to Scan calls (add after &task.BlockedReason)
sed -i 's/&task\.BlockedReason,$/&task.BlockedReason,\n\t\t&task.ExecutionOrder,/g' "$FILE"

# Update ORDER BY clauses to include execution_order NULLS LAST
sed -i 's/ORDER BY priority ASC, created_at ASC/ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC/g' "$FILE"
sed -i 's/ORDER BY t\.priority ASC, t\.created_at ASC/ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC/g' "$FILE"

echo "Updated $FILE with execution_order field"
echo "Backup saved to $FILE.bak"
