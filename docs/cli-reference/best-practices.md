# Best Practices

Guidelines for using Shark CLI effectively, for both AI agents and human developers.

## Quick Reference by Persona

### AI Agent (DevAgent)

```bash
# Always use --json for machine-readable output
shark status advance E07-F01-001 --json                # Advance to next status
shark task next-status E07-F01-001 --json              # Same as above (standard form)

# Use --field for specific values
shark get E07-F01-001 --field status                   # Just the status
shark get E07-F01-001 --field key                      # Just the key

# Use status commands for workflow
shark status advance E07-F01-001                       # Advance workflow status
shark status set E07-F01-001 in_development            # Set status directly
```

### Human Developer

```bash
# Use readable output (default)
shark status                               # Project dashboard
shark progress E07                         # Epic progress
shark list E07                             # Features in epic

# Use status commands for workflow
shark status advance E07-F01-001           # Advance to next status
shark status set E07-F01-001 in_progress   # Set a specific status
shark task list --status=todo              # Find tasks to work on
```

---

## Command Style

Shark supports two command styles that work identically:

| Quick Style | Standard Style | Description |
|-------------|----------------|-------------|
| `shark status advance <key>` | `shark task next-status <key>` | Advance to next status |
| `shark status set <key> <status>` | `shark task set-status <key> <status>` | Set a specific status |

**Recommendation**: Use status commands for daily workflow, standard entity commands for scripts and documentation.

---

## Exit Codes

| Code | Meaning | Example |
|------|---------|---------|
| `0` | Success | Task started |
| `1` | Not found | Entity doesn't exist |
| `2` | Database error | Connection failed |
| `3` | Invalid state | Bad status transition |
| `4` | Field not found | `--field` requested a non-existent field |

### Script Usage

```bash
#!/bin/bash
shark status advance E07-F01-001 --json
if [ $? -eq 0 ]; then
  echo "Task advanced"
elif [ $? -eq 1 ]; then
  echo "Task not found"
elif [ $? -eq 3 ]; then
  echo "Invalid status transition"
fi
```

### Python Usage

```python
import subprocess, json

result = subprocess.run(
    ["shark", "status", "advance", "E07-F01-001", "--json"],
    capture_output=True, text=True
)

if result.returncode == 0:
    task = json.loads(result.stdout)
    print(f"Advanced: {task['key']}")
else:
    print(f"Error (code {result.returncode}): {result.stderr}")
```

---

## AI Agent Best Practices

### 1. Always Use `--json`

```bash
shark get E07-F01-001 --json
shark status advance E07-F01-001 --json
shark task list --json
```

### 2. Use `--field` for Quick Extraction

```bash
# Get just the status
shark get E07-F01-001 --field status

# Get just the key
shark get E07-F01-001 --field key
```

### 3. Use Task List for Finding Work

```bash
# Find tasks by status and agent type
shark task list --status=todo --agent=backend --json
```

### 4. Track Agent Identity

```bash
shark task next-status E07-F01-001 --json
```

### 5. Use Completion Notes

```bash
shark task next-status E07-F01-001 --json
shark task note add E07-F01-001 --type implementation "Added JWT token validation"
```

### 6. Check Status Transitions

```bash
# See what transitions are available
shark status options E07-F01-001 --json

# Use --force only as admin override
shark status advance E07-F01-001 --force  # Bypasses validation
```

---

## Workflow Best Practices

### Handle Blocked Tasks

```bash
# Block with clear reason
shark status set E07-F01-002 blocked --reason="Waiting for API design approval"

# Unblock when ready
shark status set E07-F01-002 todo
```

### Use Task Context

```bash
# Save work-in-progress context
shark context set E07-F01-001 --field current_step --value "Implementing API"

# Resume later with full context
shark task resume E07-F01-001 --json
```

### Add Notes

```bash
# Add progress notes
shark task note add E07-F01-001 --content="Halfway through implementation"

# View timeline
shark task timeline E07-F01-001
```

---

## Agent Type Selection

### Standard Types

| Agent Type | Use For |
|-----------|---------|
| `frontend` | UI/UX implementation |
| `backend` | Server-side development |
| `architect` | System design |
| `qa` | Testing and QA |
| `tech-lead` | Code review |
| `developer` | General development |

### Multi-Agent Workflows

```bash
# Create tasks for different agents
shark task create E07 F01 "Design API" --agent=architect --order=1
shark task create E07 F01 "Implement API" --agent=backend --order=2
shark task create E07 F01 "Build UI" --agent=frontend --order=3

# Each agent finds their work
shark task list --agent=architect --status=todo --json
shark task list --agent=backend --status=todo --json
shark task list --agent=frontend --status=todo --json
```

---

## Performance Best Practices

### Filter Early

```bash
# Filter in command (faster)
shark task list --status=todo --agent=backend --json

# Don't fetch all then filter (slower)
shark task list --json | jq '.[] | select(.status == "todo")'
```

### Batch Operations

```bash
# Get all tasks once
tasks=$(shark task list E07 --json)
echo "$tasks" | jq -r '.[] | .key'
```

---

## Entity Updates

### Quick Updates with `shark update`

Use `shark update <KEY>` to update any entity. The command auto-detects entity type from the key format:

```bash
# Update epic
shark update E07 --title="New Epic Title"

# Update feature
shark update E07-F01 --title="New Feature Title"

# Update task
shark update E07-F01-001 --title="New Task Title" --priority=8
```

**Note:** For status changes, use `shark status set <KEY> <status>` instead of update flags. Status transitions have workflow validation that the update command does not enforce.

```bash
# Correct: use status commands for status changes
shark status set E07-F01-001 in_progress
shark status advance E07-F01-001

# Incorrect: don't use update for status changes
# shark update E07-F01-001 --status=in_progress  # Use status commands instead
```

### Related Documents

Use `shark docs` (alias for `shark related-docs`) to manage related documentation:

```bash
shark docs list --feature=E07-F01
shark docs add --feature=E07-F01 --path="docs/design.md"
```

---

## Administration Commands

Setup and maintenance commands are grouped under `shark admin`:

```bash
shark admin init                           # Initialize project
shark admin init update --workflow=advanced  # Apply advanced workflow
shark admin validate                       # Validate project structure
shark admin config show                    # Show full config
shark admin config validate                # Validate config file
shark admin cloud init --url="..." --auth-token="..."  # Cloud database setup
shark admin cloud status                   # Cloud database status
shark admin migrate slugs                  # Run slug migration
```

---

## Security Best Practices

1. **Never commit auth tokens** — Use `auth_token_file` or environment variables
2. **Use `--config` for multi-environment** — Don't edit shared config
3. **Validate inputs** — Check task keys exist before operations

```bash
# Validate before operating
if ! shark get "$key" --json > /dev/null 2>&1; then
  echo "Invalid key: $key"
  exit 1
fi
```

## Related Documentation

- [Global Flags](global-flags.md) - Flags available to all commands
- [Error Messages](error-messages.md) - Error handling
- [JSON Output](json-output.md) - JSON response format
- [Status Commands](status-commands.md) - Status workflow commands
