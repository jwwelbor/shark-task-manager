# Best Practices

Guidelines for using Shark CLI effectively, especially for AI agents and automation.

## AI Agent Best Practices

1. **Always use `--json` flag** for machine-readable output
2. **Check dependencies** before starting tasks via `shark task next --json`
3. **Use atomic operations** - each command is a single transaction
4. **Handle blocked tasks** - use `block` command with reasons
5. **Sync after Git operations** - run `shark sync` after pulls/checkouts
6. **Track work with agent identifier** - use `--agent` flag for audit trail
7. **Use priority effectively** - 1=highest, 10=lowest for task ordering
8. **Check exit codes** - Non-zero indicates errors

## Exit Codes

Shark CLI uses standard exit codes:

- `0`: Success
- `1`: Not found (entity does not exist)
- `2`: Database error
- `3`: Invalid state (e.g., invalid status transition)

### Example Usage in Scripts

```bash
#!/bin/bash

shark task start T-E07-F01-001
if [ $? -eq 0 ]; then
  echo "Task started successfully"
else
  echo "Failed to start task"
  exit 1
fi
```

### Python Example

```python
import subprocess
import sys

result = subprocess.run(
    ["shark", "task", "start", "E07-F01-001", "--json"],
    capture_output=True,
    text=True
)

if result.returncode == 0:
    print("Task started successfully")
    task = json.loads(result.stdout)
else:
    print(f"Failed to start task: {result.stderr}")
    sys.exit(result.returncode)
```

## JSON Output Best Practices

### Always Parse JSON

```python
import json
import subprocess

# Run command with --json flag
result = subprocess.run(
    ["shark", "task", "list", "--status=todo", "--json"],
    capture_output=True,
    text=True
)

# Parse JSON response
tasks = json.loads(result.stdout)

# Process tasks
for task in tasks:
    print(f"Task {task['key']}: {task['title']}")
```

### Error Handling

```python
import json
import subprocess

def get_task(task_key):
    result = subprocess.run(
        ["shark", "task", "get", task_key, "--json"],
        capture_output=True,
        text=True
    )
    
    if result.returncode != 0:
        if result.returncode == 1:
            raise ValueError(f"Task not found: {task_key}")
        elif result.returncode == 2:
            raise RuntimeError(f"Database error: {result.stderr}")
        else:
            raise RuntimeError(f"Unknown error: {result.stderr}")
    
    return json.loads(result.stdout)
```

## Workflow Best Practices

### Check Dependencies First

```bash
# Get next available task (dependencies resolved)
shark task next --agent=backend --json

# This ensures all dependencies are completed
```

### Track Agent Work

```bash
# Start task with agent identifier
shark task start E07-F01-001 --agent="ai-agent-backend-001" --json

# Complete task with notes
shark task complete E07-F01-001 \
  --notes="Implementation complete, 15 tests passing" \
  --json
```

### Handle Blocked Tasks

```bash
# Block task with clear reason
shark task block E07-F01-002 \
  --reason="Waiting for API design approval from architect" \
  --json

# Unblock when ready
shark task unblock E07-F01-002 --json
```

## Database Sync Best Practices

### After Git Operations

```bash
# After pulling changes
git pull
shark sync --strategy=file-wins

# After switching branches
git checkout feature-branch
shark sync --strategy=file-wins

# Preview changes before sync
shark sync --dry-run --json
```

### Conflict Resolution

```bash
# File system is source of truth (recommended for most cases)
shark sync --strategy=file-wins

# Database is source of truth (for recovering from file corruption)
shark sync --strategy=database-wins

# Most recent modification wins
shark sync --strategy=newer-wins
```

## Performance Best Practices

### Batch Operations

```bash
# List all tasks in epic once, then process
tasks=$(shark task list E07 --json)

# Process tasks in script
echo "$tasks" | jq -r '.[] | .key'
```

### Filter Early

```bash
# Filter in shark command (faster)
shark task list --status=todo --agent=backend --json

# Don't filter after fetching all tasks (slower)
shark task list --json | jq '.[] | select(.status == "todo")'
```

## Security Best Practices

### Protect Sensitive Data

```bash
# Don't commit auth tokens
echo ".sharkconfig.json" >> .gitignore

# Use token files instead
shark cloud init \
  --url="libsql://..." \
  --auth-file="~/.turso/token" \
  --non-interactive
```

### Validate Input

```bash
# Validate task keys before operations
if ! shark task get "$task_key" --json > /dev/null 2>&1; then
  echo "Invalid task key: $task_key"
  exit 1
fi

# Proceed with operation
shark task start "$task_key" --json
```

## Related Documentation

- [Global Flags](global-flags.md) - Common flags for all commands
- [Error Messages](error-messages.md) - Error handling and troubleshooting
- [JSON Output](json-output.md) - JSON response format reference
- [Orchestrator Actions](orchestrator-actions.md) - AI orchestrator integration
