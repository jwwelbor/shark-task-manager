# CLI Quick Reference: New Syntax Patterns

**Feature**: E10-F20 - Standardize CLI Command Options
**Status**: Approved for Implementation
**Quick Start Guide**

---

## TL;DR - What's New?

✅ **Case doesn't matter**: `e01` = `E01` = `e-01` (all work!)
✅ **Shorter syntax**: `shark task create e01 f02 "Title"` (no flags!)
✅ **Short task keys**: `e01-f02-001` instead of `T-E01-F02-001` ✨ NEW
✅ **Better errors**: Clear messages with examples

**All existing commands still work!** This is purely additive.

---

## Quick Comparison

### Old Way (Still Works)
```bash
shark feature create --epic=E01 "Feature Title"
shark task create --epic=E01 --feature=F02 "Task Title"
shark task list --epic=E01 --feature=F02
```

### New Way (Recommended)
```bash
shark feature create e01 "Feature Title"
shark task create e01 f02 "Task Title"
shark task list e01 f02
```

**Both syntaxes work!** Use whichever you prefer.

---

## Case Insensitivity

### Before
```bash
$ shark epic get e01
❌ Error: invalid epic key format: "e01"

$ shark epic get E01
✅ Works
```

### After
```bash
$ shark epic get e01
✅ Works (normalized to E01)

$ shark epic get E01
✅ Works

$ shark epic get E-01
❌ Error (but helpful message with examples)
```

**Rule**: Case doesn't matter, but format does.

---

## Positional Arguments

### Epic Commands
```bash
# List (no change)
shark epic list
shark epic list --json

# Get (case insensitive)
shark epic get e01
shark epic get E01
shark epic get E01-epic-name

# Create (no change - no parent context)
shark epic create "Epic Title"
```

### Feature Commands
```bash
# List
shark feature list              # All features
shark feature list e01          # Features in E01 (NEW!)
shark feature list --epic=E01   # Old syntax (still works)

# Get (case insensitive)
shark feature get f02
shark feature get e01-f02
shark feature get F02-feature-name

# Create (NEW SYNTAX!)
shark feature create e01 "Feature Title"           # NEW
shark feature create --epic=E01 "Feature Title"    # OLD (still works)
```

### Task Commands
```bash
# List
shark task list                 # All tasks
shark task list e01             # Tasks in E01 (NEW!)
shark task list e01 f02         # Tasks in E01-F02 (NEW!)
shark task list e01-f02         # Alternative format (NEW!)
shark task list --epic=E01 --feature=F02  # OLD (still works)

# Get (case insensitive + short format)
shark task get t-e01-f02-001           # Full format
shark task get T-E01-F02-001           # Case doesn't matter
shark task get e01-f02-001             # Short format (drop T-) ✨ NEW
shark task get T-E01-F02-001-task-name # Slugged

# Create (NEW SYNTAX!)
shark task create e01 f02 "Task Title"                      # NEW
shark task create --epic=E01 --feature=F02 "Task Title"    # OLD (still works)

# With additional flags
shark task create e01 f02 "Task Title" --agent=backend --priority=3

# Actions (case insensitive + short format)
shark task start t-e01-f02-001         # Full format
shark task start e01-f02-001           # Short format ✨ NEW
shark task complete T-E01-F02-001      # Full, uppercase
shark task complete e01-f02-001        # Short, lowercase ✨ NEW
shark task approve t-e01-f02-001
```

---

## Syntax Cheat Sheet

### List Commands
```
Pattern: shark <entity> list [EPIC] [FEATURE] [--filters...] [--json]

Examples:
  shark epic list
  shark feature list e01
  shark task list e01 f02
  shark task list e01 --status=todo
```

### Get Commands
```
Pattern: shark <entity> get <KEY> [--json]

Examples:
  shark epic get e01
  shark feature get e01-f02
  shark task get t-e01-f02-001      # Full format
  shark task get e01-f02-001        # Short format ✨ NEW
```

### Create Commands
```
Pattern (Epic):   shark epic create "Title" [--flags...]
Pattern (Feature): shark feature create [EPIC] "Title" [--flags...]
Pattern (Task):   shark task create [EPIC] [FEATURE] "Title" [--flags...]

Examples:
  shark epic create "User Management"
  shark feature create e01 "Authentication"
  shark task create e01 f02 "Implement JWT" --agent=backend
```

### Action Commands
```
Pattern: shark task <action> <KEY> [--flags...] [--json]

Examples:
  shark task start t-e01-f02-001                     # Full format
  shark task start e01-f02-001                       # Short format ✨ NEW
  shark task complete t-e01-f02-001 --notes="Done"   # Full format
  shark task complete e01-f02-001 --notes="Done"     # Short format ✨ NEW
  shark task approve t-e01-f02-001
  shark task block e01-f02-001 --reason="Waiting on API"
```

---

## Common Workflows

### Workflow 1: Create Epic → Feature → Task

```bash
# Create epic
shark epic create "User Management System"
# Output: Epic E01 created

# Create feature in that epic
shark feature create e01 "Authentication"
# Output: Feature E01-F01 created

# Create task in that feature
shark task create e01 f01 "Implement JWT validation" --agent=backend
# Output: Task T-E01-F01-001 created
```

### Workflow 2: Work on Tasks

```bash
# Find next task
shark task next --agent=backend --json

# Start the task (use either format)
shark task start t-e01-f01-001                        # Full format
shark task start e01-f01-001                          # Short format ✨ NEW

# Complete the task
shark task complete e01-f01-001 --notes="Implementation done"  # Short format ✨ NEW

# Approve the task
shark task approve e01-f01-001                        # Short format ✨ NEW
```

### Workflow 3: Browse Project

```bash
# List all epics
shark epic list

# List features in epic E01
shark feature list e01

# List tasks in feature E01-F01
shark task list e01 f01

# Get detailed task info
shark task get t-e01-f01-001 --json
```

---

## Error Messages

### Old Error Messages
```bash
$ shark epic get e-01
❌ Error: invalid epic key format: "e-01" (expected E##)
```

### New Error Messages
```bash
$ shark epic get e-01
❌ Error: Invalid key format: "e-01"
     Expected: E## (two-digit epic number)
     Examples: E01, E04, E99
     Note: Case insensitive (e01, E01, and E-01 are equivalent)
     Tip: Use two-digit numbers (E01, not E1)
```

**Much more helpful!**

---

## Key Formats

### Valid Formats

#### Epic
✅ `E01`, `e01`, `E99`
✅ `E01-epic-name` (slugged)
❌ `E1`, `E-01`, `E001`

#### Feature
✅ `E01-F02`, `e01-f02`, `F02`, `f02`
✅ `E01-F02-feature-name` (slugged)
❌ `E01F02`, `E1-F2`, `F1`

#### Task
✅ `T-E01-F02-001`, `t-e01-f02-001` (full format)
✅ `E01-F02-001`, `e01-f02-001` (short format - drop T-) ✨ NEW
✅ `T-E01-F02-001-task-name` (slugged, full)
✅ `e01-f02-001-task-name` (slugged, short) ✨ NEW
❌ `TE01F02001`, `T-E1-F2-1`, `E1-F2-1`

**Remember**: Case doesn't matter, format does!

---

## Flag Reference

### Global Flags (All Commands)
```bash
--json           # Machine-readable JSON output
--no-color       # Disable colored output
--verbose / -v   # Debug logging
--db <path>      # Override database path
--config <path>  # Override config file path
```

### Create Flags
```bash
# Epic create
--priority=<1-10>         # Priority (1 = highest)
--business-value=<1-10>   # Business value score

# Feature create
--epic=<key>              # Parent epic (or use positional)
--execution-order=<num>   # Order within epic

# Task create
--epic=<key>              # Parent epic (or use positional)
--feature=<key>           # Parent feature (or use positional)
--agent=<type>            # Agent type (frontend, backend, etc.)
--priority=<1-10>         # Priority (default: 5)
--depends-on=<keys>       # Comma-separated dependencies
```

### List Flags
```bash
--status=<status>   # Filter by status (todo, in_progress, etc.)
--agent=<type>      # Filter by agent type
--show-all          # Include completed items
```

---

## AI Agent Templates

### Python Example

```python
class SharkCLI:
    def create_task(self, epic, feature, title, agent="general", priority=5):
        """Create task - case and format handled by shark"""
        cmd = [
            "shark", "task", "create",
            epic.lower(),    # Case doesn't matter
            feature.lower(), # Case doesn't matter
            title,
            f"--agent={agent}",
            f"--priority={priority}",
            "--json"
        ]
        return self.run(cmd)

    def list_tasks(self, epic=None, feature=None, status=None):
        """List tasks with flexible filtering"""
        cmd = ["shark", "task", "list"]

        if epic:
            cmd.append(epic.lower())
        if feature:
            cmd.append(feature.lower())
        if status:
            cmd.append(f"--status={status}")

        cmd.append("--json")
        return self.run(cmd)
```

**No normalization code needed!** Shark handles it.

---

## Migration Guide

### For Existing Scripts

**Good News**: No changes needed! All existing commands still work.

**Optional Upgrades**:

Before:
```bash
#!/bin/bash
EPIC="E01"
FEATURE="F02"
shark task create --epic=$EPIC --feature=$FEATURE "Task Title"
```

After (optional):
```bash
#!/bin/bash
epic="e01"  # Lowercase works now
feature="f02"
shark task create $epic $feature "Task Title"
```

### For AI Agents

Before:
```python
# Required: Case normalization
def normalize(key):
    return key.upper()

epic = normalize(context.epic)
cmd = f"shark task create --epic={epic} --feature={feature} '{title}'"
```

After:
```python
# No normalization needed
epic = context.epic  # Any case works
cmd = f"shark task create {epic} {feature} '{title}'"
```

**56% less code!**

---

## Testing Your Upgrade

### Test Case Insensitivity
```bash
# All these should work
shark epic get E01
shark epic get e01
shark epic get E01-epic-name

# These should fail with helpful errors
shark epic get E1
shark epic get E-01
```

### Test Positional Syntax
```bash
# All these should work
shark feature create E01 "Feature Title"
shark feature create e01 "Feature Title"
shark task create E01 F02 "Task Title"
shark task create e01 f02 "Task Title"
```

### Test Flag Syntax (Backward Compatibility)
```bash
# All old commands should still work
shark feature create --epic=E01 "Feature Title"
shark task create --epic=E01 --feature=F02 "Task Title"
shark task list --epic=E01 --feature=F02
```

---

## Troubleshooting

### Issue: "Invalid epic key format"
**Old Error**:
```
Error: invalid epic key format: "e1"
```

**New Error**:
```
Error: Invalid key format: "e1"
  Expected: E## (two-digit epic number)
  Examples: E01, E04, E99
  Tip: Use two-digit numbers (E01, not E1)
```

**Fix**: Use two digits: `E01` instead of `E1`

### Issue: "Ambiguous arguments"
```bash
$ shark task create e01 "Task Title"
Error: ambiguous arguments: use 3 args (epic feature title) or 1 arg (title) with flags
```

**Fix**: Provide all 3 positional args or use flags:
```bash
# Good: 3 args
shark task create e01 f02 "Task Title"

# Good: Flags
shark task create --epic=e01 --feature=f02 "Task Title"

# Bad: 2 args (ambiguous)
shark task create e01 "Task Title"
```

### Issue: "Both positional and flags provided"
```bash
$ shark feature create e01 "Feature" --epic=E02
Warning: Both positional epic and --epic flag provided. Using --epic flag value.
```

**Fix**: Choose one syntax:
```bash
# Good: Positional only
shark feature create e01 "Feature"

# Good: Flags only
shark feature create --epic=e01 "Feature"
```

---

## Best Practices

### 1. Use Positional Syntax for Simple Cases
```bash
# Good
shark task create e01 f02 "Task Title"

# Less good (but still works)
shark task create --epic=E01 --feature=F02 "Task Title"
```

### 2. Use Flags for Complex Cases
```bash
# Good: Complex options benefit from explicit flags
shark task create e01 f02 "Task Title" \
  --agent=backend \
  --priority=3 \
  --depends-on="T-E01-F02-001,T-E01-F02-002"
```

### 3. Don't Worry About Case
```bash
# All equivalent
shark task list E01 F02
shark task list e01 f02
shark task list E01 f02
```

### 4. Use JSON for Automation
```bash
# Always use --json for scripts/AI agents
shark task next --agent=backend --json | jq '.key'
```

### 5. Check Help When Unsure
```bash
shark task create --help
shark task list --help
```

---

## Summary

### What Changed
✅ Case insensitivity for all keys
✅ Positional arguments for create commands
✅ Enhanced error messages

### What Didn't Change
✅ All existing commands still work
✅ Flag syntax still supported
✅ JSON output format unchanged
✅ Exit codes unchanged

### Key Takeaways
1. **Case doesn't matter** - `e01` and `E01` are equivalent
2. **Positional is simpler** - `shark task create e01 f02 "Title"`
3. **Both syntaxes work** - Use whichever you prefer
4. **No migration needed** - Existing scripts continue to work

---

## Quick Links

- **Full Specification**: `F20-cli-ux-specification.md`
- **User Journey Comparison**: `F20-user-journey-comparison.md`
- **Implementation Guide**: `F20-implementation-guide.md`
- **CLI Reference**: `/home/jwwelbor/projects/shark-task-manager/docs/CLI_REFERENCE.md`
- **Development Guide**: `/home/jwwelbor/projects/shark-task-manager/CLAUDE.md`

---

## Feedback

Found an issue? Have a suggestion?
- File an issue: `shark task create <epic> <feature> "Issue description"`
- Contact: [Product Manager / UX Designer]
