# Error Messages

Shark CLI provides user-friendly error messages with context and examples.

## Enhanced Error Format

When an error occurs, you'll see:
1. **Clear description** of what went wrong
2. **Context** about why it happened
3. **Example** showing the correct syntax
4. **Suggestions** for resolution

## Common Errors and Solutions

### Invalid Epic Key Format

**Error:**
```
Error: invalid epic key format: "invalid"

Epic keys must follow format: E{number} or E{number}-{slug}

Valid examples:
  - E07
  - e07 (case insensitive)
  - E07-user-management
  - e07-user-management (case insensitive)
```

**Solution:** Use the correct epic key format with `E` prefix followed by a number.

---

### Invalid Feature Key Format

**Error:**
```
Error: invalid feature key format: "invalid"

Feature keys must follow one of these formats:
  - E{epic}-F{feature} (full format)
  - F{feature} (short format)
  - With optional slug suffix

Valid examples:
  - E07-F01, e07-f01 (case insensitive)
  - F01, f01 (case insensitive)
  - E07-F01-authentication
  - F01-authentication
```

**Solution:** Use the correct feature key format.

---

### Invalid Task Key Format

**Error:**
```
Error: invalid task key format: "invalid"

Task keys must follow one of these formats:
  - E{epic}-F{feature}-{number} (short format, recommended)
  - T-E{epic}-F{feature}-{number} (traditional format)
  - With optional slug suffix

Valid examples:
  - E07-F20-001, e07-f20-001 (case insensitive)
  - T-E07-F20-001, t-e07-f20-001
  - E07-F20-001-implement-jwt
  - T-E07-F20-001-implement-jwt
```

**Solution:** Use the correct task key format. The `T-` prefix is optional.

---

### Task Not Found

**Error:**
```
Error: task not found: "E07-F20-999"

The task key was not found in the database.

Possible solutions:
  - Check the task key spelling
  - List tasks: shark task list E07 F20
  - Verify epic and feature exist
```

**Solution:** Verify the task exists using `shark task list` or check for typos.

---

### Invalid Status Transition

**Error:**
```
Error: cannot transition from 'completed' to 'in_progress'

Valid transitions from 'completed':
  - No valid transitions (task is completed)

Task lifecycle:
  todo → in_progress → ready_for_review → completed
           ↓              ↓
        blocked ←────────┘
```

**Solution:** Follow the valid task lifecycle transitions. Use `shark task reopen` to return a task from review to in-progress.

---

### Missing Required Arguments

**Error:**
```
Error: missing required arguments

Usage: shark task create <epic-key> <feature-key> "<title>" [flags]
   OR: shark task create <epic-feature-key> "<title>" [flags]
   OR: shark task create --epic=<key> --feature=<key> --title="<title>" [flags]

Examples:
  shark task create E07 F20 "Task Title"
  shark task create E07-F20 "Task Title"
  shark task create --epic=E07 --feature=F20 --title="Task Title"
```

**Solution:** Provide all required arguments in one of the supported syntaxes.

---

## Interpreting Error Messages

All error messages follow this structure:

```
Error: <brief description>

<detailed explanation>

<valid examples or solutions>
```

**Tips:**
- Read the entire error message for context
- Check the examples provided
- Verify your syntax matches one of the valid formats
- Use case insensitive keys (e07 works same as E07)
- Try the short format (E07-F20-001 instead of T-E07-F20-001)

## Related Documentation

- [Key Formats](key-formats.md) - Valid key formats
- [Best Practices](best-practices.md) - Error handling in scripts
- [Task Commands](task-commands.md) - Task lifecycle
