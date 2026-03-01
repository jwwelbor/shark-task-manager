# Error Messages

Shark CLI provides clear error messages with context and suggestions for resolution.

---

## Error Format

### Human-Readable (Default)

```
Error: <brief description>

<detailed explanation>

<valid examples or solutions>
```

### JSON Error Format (`--json`)

When using `--json`, errors are returned as structured JSON:

```json
{
  "error": true,
  "code": 1,
  "message": "task not found: E07-F20-999",
  "details": "The task key was not found in the database.",
  "suggestions": [
    "Check the task key spelling",
    "List tasks: shark task list E07 F20",
    "Verify epic and feature exist"
  ]
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `error` | bool | Always `true` for errors |
| `code` | int | Exit code (1=not found, 2=DB error, 3=invalid state) |
| `message` | string | Brief error description |
| `details` | string | Extended context (optional) |
| `suggestions` | string[] | Possible fixes (optional) |

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
case $? in
  0) echo "Task advanced" ;;
  1) echo "Task not found" ;;
  2) echo "Database error" ;;
  3) echo "Invalid status transition" ;;
esac
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
    error = json.loads(result.stdout)
    print(f"Error ({error['code']}): {error['message']}")
```

---

## Common Errors

### Invalid Key Format

**Error:**
```
Error: invalid epic key format: "invalid"

Epic keys must follow format: E{number} or E{number}-{slug}

Valid examples:
  - E07
  - e07 (case insensitive)
  - E07-user-management
```

**JSON:**
```json
{
  "error": true,
  "code": 1,
  "message": "invalid epic key format: \"invalid\"",
  "suggestions": ["Use format: E{number}", "Keys are case insensitive"]
}
```

**Solution:** Use the correct key format. See [Key Formats](key-formats.md).

| Entity | Valid Formats |
|--------|-------------|
| Epic | `E07`, `e07`, `E07-user-management` |
| Feature | `E07-F01`, `F01`, `E07-F01-authentication` |
| Task | `E07-F01-001`, `T-E07-F01-001`, `E07-F01-001-implement-jwt` |

---

### Entity Not Found

**Error:**
```
Error: task not found: "E07-F20-999"

The task key was not found in the database.

Possible solutions:
  - Check the task key spelling
  - List tasks: shark task list E07 F20
  - Verify epic and feature exist
```

**Solution:** Verify the entity exists with `shark list` or `shark get`.

---

### Invalid Status Transition

**Error:**
```
Error: cannot transition from 'completed' to 'in_progress'

Valid transitions from 'completed':
  - No valid transitions (task is completed)

Use --force to bypass transition validation.
```

**Solution:** Follow valid lifecycle transitions. Use `shark status options <key>` to see valid next statuses. Use `--force` as admin override if needed.

---

### Missing Required Arguments

**Error:**
```
Error: missing required arguments

Usage: shark task create <epic-key> <feature-key> "<title>" [flags]
   OR: shark task create <epic-feature-key> "<title>" [flags]

Examples:
  shark task create E07 F20 "Task Title"
  shark task create E07-F20 "Task Title"
```

**Solution:** Provide all required arguments. Run `shark <command> --help` for usage.

---

### Database Connection Error

**Error:**
```
Error: failed to open database: unable to open database file

Possible solutions:
  - Check database path in .sharkconfig.json
  - Verify file permissions
  - Run 'shark admin init' to create database
```

**JSON:**
```json
{
  "error": true,
  "code": 2,
  "message": "failed to open database",
  "details": "unable to open database file: shark-tasks.db",
  "suggestions": ["Check .sharkconfig.json database.url", "Run 'shark admin init'"]
}
```

**Solution:** Ensure the database exists and is accessible. Run `shark admin init --non-interactive` to create a new database.

---

### Blocked Task

**Error:**
```
Error: cannot advance task E07-F01-002: task is blocked

Reason: Waiting for API design approval

Unblock with: shark status set E07-F01-002 todo
```

**Solution:** Resolve the blocking issue, then use `shark status set <key> <status>` to move the task back to an active status.

---

### Dependency Not Met

**Error:**
```
Error: cannot start task E07-F01-003: dependencies not met

Unresolved dependencies:
  - E07-F01-001 (status: in_progress)
  - E07-F01-002 (status: todo)

All dependencies must be completed before starting this task.
```

**Solution:** Complete dependency tasks first, or use `--force` to bypass dependency checks.

---

## Tips

- Read the full error message including examples and suggestions
- All keys are **case insensitive** (`E07` = `e07`)
- Use `--json` for machine-parseable error details
- Use `shark <command> --help` for command syntax
- Use `shark status options <key>` to see valid status transitions
- Use `shark update <key>` to quickly update any entity (auto-detects type)
- Use `shark docs` as a shorthand for `shark related-docs`
- Use `--force` to bypass validation (admin override)
- Use `--verbose` to see debug information for troubleshooting
- Administration commands (init, validate, config, cloud, migrate) are under `shark admin`

## Related Documentation

- [Key Formats](key-formats.md) - Valid key formats
- [Global Flags](global-flags.md) - `--json`, `--verbose`, `--force`
- [Best Practices](best-practices.md) - Error handling in scripts
- [Status Commands](status-commands.md) - Status transitions
