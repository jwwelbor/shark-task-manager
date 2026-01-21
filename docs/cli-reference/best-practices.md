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

## Agent Type Selection

### Choose Appropriate Agent Types

Agent types should match your workflow and team structure:

**Standard Agent Types** (use when working with traditional development roles):
- `frontend` - Frontend development and UI implementation
- `backend` - Backend development and API implementation
- `api` - API design and integration
- `testing` - Test development and quality assurance
- `devops` - DevOps and infrastructure
- `general` - General purpose tasks

**Custom Agent Types** (use for specialized workflow phases and roles):
- `architect` - System design and architecture decisions
- `business-analyst` - Requirements elaboration and user stories
- `qa` - Test planning and quality assurance
- `tech-lead` - Technical coordination and code review
- `product-manager` - Feature planning and prioritization
- `ux-designer` - UI/UX design and prototyping

### Maintain Consistency

```bash
# Good: Consistent naming
shark task create E07 F01 "Design system" --agent=architect
shark task list --agent=architect

# Avoid: Inconsistent naming (always use same string)
shark task create E07 F01 "Design database" --agent=architect
shark task create E07 F01 "Design API" --agent=Architect  # Different case
shark task create E07 F01 "Design schema" --agent=archit  # Typo

# Use standard names for filtering
shark task next --agent=architect --json  # Matches above tasks
```

### Multi-Agent Workflows

When coordinating multiple AI agents or team members:

```bash
# Assign specific agent types to each role
shark task create E07 F01 "Build API" --agent=backend
shark task create E07 F01 "Build UI" --agent=frontend
shark task create E07 F01 "Design architecture" --agent=architect

# Each agent retrieves their work
shark task next --agent=backend --json      # Backend agent gets their task
shark task next --agent=frontend --json     # Frontend agent gets their task
shark task next --agent=architect --json    # Architect gets their task

# Filter by agent for role-based task lists
shark task list --agent=architect --status=todo --json
```

### Template Awareness

- **Standard agent types** may have role-specific templates for task creation
- **Custom agent types** automatically use the `general` template
- All templates can be customized in `internal/init/shark-templates/`
- Template choice doesn't affect filtering or task assignment

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
