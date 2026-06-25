# Optional Hooks for Task Management

## Overview

This document describes optional hooks you can enable to automate or enhance task management workflows.

## Test Completion Reminder Hook

Automatically reminds you to complete a task when all tests pass.

### How It Works

When you run tests via Bash (pytest, go test, npm test, make test) and they all pass, you'll see a reminder:

```
All tests passed! Consider completing the task with: /task-complete <task-id>
```

### Installation

Add this to your `.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "jq -r 'if (.tool_input.command | contains(\"pytest\") or contains(\"go test\") or contains(\"npm test\") or contains(\"make test\")) and (.exit_code == 0) then \"\\nAll tests passed! Consider completing the task with: /task-complete <task-id>\\n\" else \"\" end'",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

## Auto-Start Hook (Advanced)

**Warning:** This hook automatically starts tasks, which may not be desired in all workflows.

Automatically starts a task when you begin implementation by reading a file from a task directory.

### Installation

Add this to your `.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Read",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/auto-start-task.sh",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

Create the script at `.claude/hooks/auto-start-task.sh`:

```bash
#!/bin/bash
input=$(cat)
file_path=$(echo "$input" | jq -r '.tool_input.file_path // empty')

if [[ "$file_path" =~ /tasks/(T-[A-Z0-9]+-[A-Z0-9]+-[0-9]+)\.md$ ]]; then
    task_id="${BASH_REMATCH[1]}"
    status=$(shark get "$task_id" --field status 2>/dev/null)
    if [[ "$status" == "todo" || "$status" == "ready_for_development" ]]; then
        echo "Auto-starting task: $task_id"
        shark status advance "$task_id"
    fi
fi
```

## Task Validation Hook

Validates that tasks are started before making file edits in implementation directories.

### Installation

Create the script at `.claude/hooks/validate-active-task.sh`:

```bash
#!/bin/bash
input=$(cat)
file_path=$(echo "$input" | jq -r '.tool_input.file_path // empty')

if [[ "$file_path" =~ ^(src/|internal/|cmd/|pkg/) ]]; then
    active_tasks=$(shark task list --status=in_progress --json 2>/dev/null | jq -r '.[] | .key')
    if [[ -z "$active_tasks" ]]; then
        echo "No active task found. Start a task with: shark status advance <task-key>"
        exit 0
    fi
fi
```

Add to `.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/validate-active-task.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```
