# Migration Guide: E07-F20 CLI Improvements

This guide covers the CLI enhancements introduced in E07-F20: CLI Command Options Standardization.

## Overview

E07-F20 introduces three major improvements to the Shark CLI:

1. **Case Insensitive Keys** - All entity keys now work with any case combination
2. **Short Task Key Format** - Task keys can omit the `T-` prefix
3. **Positional Argument Syntax** - Cleaner command syntax for feature and task creation

**Important: All changes are backward compatible.** Existing commands, scripts, and workflows will continue to work without modification.

---

## Why These Changes Were Made

### Problem Statement
The original CLI syntax was verbose and case-sensitive, leading to:
- Longer commands that were harder to type
- Errors when users typed keys in different cases
- Inconsistent UX compared to modern CLIs
- Friction for AI agents and human users alike

### Solution
We modernized the CLI while maintaining 100% backward compatibility:
- Case insensitivity reduces typing errors
- Short task keys reduce visual noise
- Positional arguments provide cleaner, more intuitive commands
- Enhanced error messages help users recover from mistakes quickly

---

## Key Format Changes

### Case Insensitivity

**Before (case-sensitive):**
```bash
shark epic get E07        # ✓ Works
shark epic get e07        # ✗ Error: epic not found
shark epic get E07-F01    # ✗ Error: invalid format
```

**After (case-insensitive):**
```bash
shark epic get E07        # ✓ Works
shark epic get e07        # ✓ Works
shark epic get E07-F01    # ✓ Works (feature, not epic)
shark epic get e07-f01    # ✓ Works (feature, not epic)
```

### Case Insensitivity Reference Table

| Entity Type | Old Format | New Formats (all equivalent) |
|-------------|------------|------------------------------|
| **Epic** | `E07` | `E07`, `e07`, `E07-user-management`, `e07-user-management` |
| **Feature** | `E07-F01` | `E07-F01`, `e07-f01`, `E07-f01`, `F01`, `f01` |
| **Task** | `T-E07-F20-001` | `T-E07-F20-001`, `t-e07-f20-001`, `E07-F20-001`, `e07-f20-001` |

---

### Short Task Key Format

**Before (traditional format):**
```bash
shark task get T-E07-F20-001
shark task start T-E07-F20-001
shark task complete T-E07-F20-001
```

**After (short format, recommended):**
```bash
shark task get E07-F20-001
shark task start E07-F20-001
shark task complete E07-F20-001
```

**Both formats work identically.** The `T-` prefix is optional and automatically normalized internally.

### Short Task Key Reference Table

| Command | Old Format | New Format (Recommended) |
|---------|------------|--------------------------|
| **Get Task** | `shark task get T-E07-F20-001` | `shark task get E07-F20-001` |
| **Start Task** | `shark task start T-E07-F20-001` | `shark task start E07-F20-001` |
| **Complete Task** | `shark task complete T-E07-F20-001` | `shark task complete E07-F20-001` |
| **Approve Task** | `shark task approve T-E07-F20-001` | `shark task approve E07-F20-001` |
| **Block Task** | `shark task block T-E07-F20-001 --reason="..."` | `shark task block E07-F20-001 --reason="..."` |
| **Unblock Task** | `shark task unblock T-E07-F20-001` | `shark task unblock E07-F20-001` |

---

## Command Syntax Changes

### Feature Create

**Before (flag-based syntax):**
```bash
shark feature create --epic=E07 --title="User Authentication"
shark feature create --epic=E07 --title="User Authentication" --execution-order=1
```

**After (positional syntax, recommended):**
```bash
shark feature create E07 "User Authentication"
shark feature create E07 "User Authentication" --execution-order=1
shark feature create e07 "User Authentication"  # Case insensitive
```

**Side-by-side comparison:**
```bash
# Old syntax (still works)
shark feature create --epic=E07 --title="Authentication" --execution-order=1

# New syntax (recommended)
shark feature create E07 "Authentication" --execution-order=1
```

---

### Task Create

**Before (flag-based syntax):**
```bash
shark task create --epic=E07 --feature=F20 --title="Update documentation"
shark task create --epic=E07 --feature=F20 --title="Update documentation" --agent=backend --priority=5
```

**After (positional syntax, recommended):**
```bash
# 3-argument format: epic, feature, title
shark task create E07 F20 "Update documentation"
shark task create E07 F20 "Update documentation" --agent=backend --priority=5

# 2-argument format: combined epic-feature, title
shark task create E07-F20 "Update documentation"
shark task create E07-F20 "Update documentation" --agent=backend --priority=5

# Case insensitive
shark task create e07 f20 "Update documentation"
shark task create e07-f20 "Update documentation"
```

**Side-by-side comparison:**
```bash
# Old syntax (still works)
shark task create --epic=E07 --feature=F20 --title="Update docs" --agent=backend --priority=5

# New syntax - 3 args (recommended)
shark task create E07 F20 "Update docs" --agent=backend --priority=5

# New syntax - 2 args (alternative)
shark task create E07-F20 "Update docs" --agent=backend --priority=5
```

---

## Migration Checklist

### For Users

- [ ] **No action required** - Old commands continue to work
- [ ] Optional: Update personal scripts to use shorter syntax
- [ ] Optional: Adopt case-insensitive keys for faster typing
- [ ] Review enhanced error messages if you encounter issues

### For Scripts

- [ ] **No action required** - Backward compatibility guaranteed
- [ ] Optional: Refactor to use positional arguments for cleaner code
- [ ] Optional: Use short task keys (`E07-F20-001` vs `T-E07-F20-001`)
- [ ] Test scripts continue to work (both old and new syntax)

### For AI Agents

- [ ] **No action required** - Both syntaxes work
- [ ] Recommended: Adopt short task key format in new code
- [ ] Recommended: Use positional syntax for cleaner prompts
- [ ] Case insensitivity reduces prompt engineering complexity

---

## Error Message Improvements

### Example: Old Error
**Before:**
```
Error: invalid key format
```

### Example: New Error
**After:**
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

Possible solutions:
  - Check the task key spelling
  - List tasks: shark task list E07 F20
  - Verify epic and feature exist
```

**Improvement:** Clear description + valid formats + examples + solutions

---

## FAQ

### Will my old commands still work?

**Yes.** All legacy syntax remains fully supported:

```bash
# These all work (old syntax)
shark feature create --epic=E07 --title="Feature Title"
shark task create --epic=E07 --feature=F20 --title="Task Title"
shark task start T-E07-F20-001

# These also work (new syntax)
shark feature create E07 "Feature Title"
shark task create E07 F20 "Task Title"
shark task start E07-F20-001
```

### Do I need to update my scripts?

**No.** Existing scripts will continue to work without modification. The new syntax is optional and additive.

However, you may want to update scripts for:
- Cleaner, more readable code
- Faster execution (fewer characters to parse)
- Better alignment with modern CLI conventions

### What if I prefer the old syntax?

That's perfectly fine! The old syntax is still fully supported and will remain so. Use whichever syntax you prefer:

- **Old syntax:** Explicit with flags (`--epic=E07 --title="Title"`)
- **New syntax:** Concise with positional args (`E07 "Title"`)

Both are valid and will continue to work indefinitely.

### Can I mix old and new syntax?

**Yes.** You can mix syntaxes as needed:

```bash
# Mix positional args with flags
shark task create E07 F20 "Task Title" --agent=backend --priority=5

# Mix case styles
shark feature create e07 "Feature Title"
shark task get E07-F20-001  # Different case, same command chain
```

### What about slugged keys?

Slugged keys continue to work exactly as before, now with case insensitivity:

```bash
# Old
shark epic get E07-user-management

# New (case insensitive)
shark epic get e07-user-management
shark feature get E07-F01-authentication
shark feature get e07-f01-authentication
shark task get E07-F20-001-update-docs
shark task get e07-f20-001-update-docs
```

---

## Examples Section

### Complete Before/After Examples

#### Creating a Feature

**Before:**
```bash
shark feature create --epic=E07 --title="User Authentication" --execution-order=1 --json
```

**After:**
```bash
# Recommended new syntax
shark feature create E07 "User Authentication" --execution-order=1 --json

# Case insensitive variant
shark feature create e07 "User Authentication" --execution-order=1 --json
```

#### Creating a Task

**Before:**
```bash
shark task create \
  --epic=E07 \
  --feature=F20 \
  --title="Update CLI documentation" \
  --agent=backend \
  --priority=4 \
  --depends-on="T-E07-F20-001,T-E07-F20-002"
```

**After:**
```bash
# Recommended new syntax (3-arg)
shark task create E07 F20 "Update CLI documentation" \
  --agent=backend \
  --priority=4 \
  --depends-on="E07-F20-001,E07-F20-002"

# Alternative new syntax (2-arg)
shark task create E07-F20 "Update CLI documentation" \
  --agent=backend \
  --priority=4 \
  --depends-on="E07-F20-001,E07-F20-002"

# Case insensitive
shark task create e07 f20 "Update CLI documentation" --agent=backend
```

#### Task Lifecycle

**Before:**
```bash
shark task start T-E07-F20-001
shark task complete T-E07-F20-001
shark task approve T-E07-F20-001
```

**After:**
```bash
# Short format (recommended)
shark task start E07-F20-001
shark task complete E07-F20-001
shark task approve E07-F20-001

# Case insensitive
shark task start e07-f20-001
shark task complete e07-f20-001
shark task approve e07-f20-001
```

---

### Common Use Cases

#### Use Case 1: Quick Task Creation

**Before (45 characters):**
```bash
shark task create --epic=E07 --feature=F20 --title="Fix bug"
```

**After (32 characters, 29% shorter):**
```bash
shark task create E07 F20 "Fix bug"
```

#### Use Case 2: Task Workflow

**Before:**
```bash
export TASK=T-E07-F20-001
shark task start $TASK
# ... do work ...
shark task complete $TASK
shark task approve $TASK
```

**After:**
```bash
export TASK=e07-f20-001  # Shorter, lowercase
shark task start $TASK
# ... do work ...
shark task complete $TASK
shark task approve $TASK
```

#### Use Case 3: Listing Tasks in Epic/Feature

**Before:**
```bash
shark task list --epic=E07 --feature=F20 --status=todo --json
```

**After:**
```bash
# Positional + flags (shorter)
shark task list E07 F20 --status=todo --json

# Alternative combined format
shark task list E07-F20 --status=todo --json

# Case insensitive
shark task list e07 f20 --status=todo --json
```

---

### AI Agent Integration Examples

#### Example 1: Python Script (Old Syntax)

**Before:**
```python
import subprocess
import json

# Create task
result = subprocess.run([
    "shark", "task", "create",
    "--epic=E07",
    "--feature=F20",
    "--title=Implement feature X",
    "--agent=backend",
    "--priority=5",
    "--json"
], capture_output=True, text=True)

task = json.loads(result.stdout)
task_key = task["key"]  # "T-E07-F20-003"

# Start task
subprocess.run(["shark", "task", "start", task_key, "--json"])
```

#### Example 2: Python Script (New Syntax)

**After:**
```python
import subprocess
import json

# Create task (shorter command)
result = subprocess.run([
    "shark", "task", "create",
    "E07", "F20", "Implement feature X",
    "--agent=backend",
    "--priority=5",
    "--json"
], capture_output=True, text=True)

task = json.loads(result.stdout)
task_key = task["key"]  # "T-E07-F20-003"

# Start task (short format, case insensitive)
subprocess.run(["shark", "task", "start", task_key.lower().replace("t-", ""), "--json"])
# Or just use the key as-is
subprocess.run(["shark", "task", "start", task_key, "--json"])
```

#### Example 3: Shell Script

**Before:**
```bash
#!/bin/bash
EPIC=E07
FEATURE=F20

# Create tasks
shark task create --epic=$EPIC --feature=$FEATURE --title="Task 1" --agent=backend
shark task create --epic=$EPIC --feature=$FEATURE --title="Task 2" --agent=backend
shark task create --epic=$EPIC --feature=$FEATURE --title="Task 3" --agent=backend

# List tasks
shark task list --epic=$EPIC --feature=$FEATURE --json
```

**After:**
```bash
#!/bin/bash
EPIC=e07  # Case insensitive
FEATURE=f20

# Create tasks (cleaner)
shark task create $EPIC $FEATURE "Task 1" --agent=backend
shark task create $EPIC $FEATURE "Task 2" --agent=backend
shark task create $EPIC $FEATURE "Task 3" --agent=backend

# List tasks (positional)
shark task list $EPIC $FEATURE --json
```

---

## Summary

### What Changed

✅ **Case insensitivity** - Use any case for all entity keys
✅ **Short task keys** - Omit `T-` prefix (`E07-F20-001` instead of `T-E07-F20-001`)
✅ **Positional arguments** - Cleaner syntax for feature/task create commands
✅ **Enhanced errors** - Better error messages with examples and solutions

### What Stayed the Same

✅ **All legacy syntax works** - 100% backward compatible
✅ **Flag-based syntax supported** - Use `--epic=` and `--title=` if preferred
✅ **Traditional task keys work** - `T-E07-F20-001` still valid
✅ **Slugged keys work** - With added case insensitivity

### Recommendations

- **New projects:** Use positional syntax and short task keys
- **Existing projects:** No changes required, migrate at your own pace
- **Scripts:** Update gradually or keep as-is (both work)
- **AI agents:** Adopt new syntax for cleaner prompts and outputs

---

## Related Documentation

- [CLI_REFERENCE.md](CLI_REFERENCE.md) - Complete command reference with new syntax
- [CLAUDE.md](../CLAUDE.md) - Development guidelines with updated examples
- [README.md](../README.md) - Project introduction with quick start examples

---

## Support

If you encounter issues or have questions about the migration:

1. Check error messages (now include examples and solutions)
2. Review [CLI_REFERENCE.md](CLI_REFERENCE.md) for syntax details
3. Test with `--json` flag for machine-readable output
4. Both old and new syntax work - use whichever you prefer

**Remember: All changes are backward compatible. Your existing commands will continue to work.**
