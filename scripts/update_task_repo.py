#!/usr/bin/env python3
"""
Script to add execution_order field to all SQL queries in task_repository.go
"""

import re

def update_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Pattern 1: Add execution_order to SELECT statements
    # Replace "blocked_reason," with "blocked_reason, execution_order,"
    # But only if execution_order is not already there
    content = re.sub(
        r'blocked_reason,(\s+)(created_at|t\.created_at)',
        r'blocked_reason, execution_order,\1\2',
        content
    )

    # Pattern 2: Add &task.ExecutionOrder to Scan statements
    # Replace "&task.BlockedReason," with "&task.BlockedReason,\n\t\t&task.ExecutionOrder,"
    content = re.sub(
        r'(&task\.BlockedReason,)(\s+)(&task\.CreatedAt)',
        r'\1\n\t\t&task.ExecutionOrder,\2\3',
        content
    )

    # Pattern 3: Update ORDER BY clauses to include execution_order NULLS LAST
    # For simple ORDER BY
    content = re.sub(
        r'ORDER BY priority ASC, created_at ASC',
        r'ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC',
        content
    )

    # For table-prefixed ORDER BY
    content = re.sub(
        r'ORDER BY t\.priority ASC, t\.created_at ASC',
        r'ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC',
        content
    )

    with open(filepath, 'w') as f:
        f.write(content)

    print(f"Updated {filepath}")

if __name__ == '__main__':
    update_file('internal/repository/task_repository.go')
