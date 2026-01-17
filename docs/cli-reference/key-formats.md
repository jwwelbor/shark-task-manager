# Key Format Improvements

Shark CLI supports flexible key formats for improved usability.

## Case Insensitive Keys

All entity keys are case insensitive. You can use any combination of uppercase and lowercase:

### Epics

```bash
shark epic get E07       # Standard
shark epic get e07       # Lowercase
shark epic get E07-user-management-system
shark epic get e07-user-management-system
```

### Features

```bash
shark feature get E07-F01        # Standard
shark feature get e07-f01        # Lowercase
shark feature get E07-f01        # Mixed case
shark feature get F01            # Short format
shark feature get f01            # Short format (lowercase)
```

### Tasks

```bash
shark task start E07-F20-001     # Short format
shark task start e07-f20-001     # Lowercase
shark task start T-E07-F20-001   # Traditional format
shark task start t-e07-f20-001   # Traditional lowercase
```

## Short Task Key Format

Task keys can now be referenced without the `T-` prefix:

### Traditional Format

```bash
shark task get T-E07-F20-001
shark task start T-E07-F20-001
shark task complete T-E07-F20-001
```

### Short Format (Recommended)

```bash
shark task get E07-F20-001
shark task start E07-F20-001
shark task complete E07-F20-001
```

Both formats work identically. The CLI automatically normalizes keys internally.

## Positional Arguments

Feature and task creation commands support cleaner positional argument syntax:

### Feature Creation

```bash
# New positional syntax (recommended)
shark feature create E07 "Feature Title"
shark feature create e07 "Feature Title"  # Case insensitive

# Traditional flag syntax (still supported)
shark feature create --epic=E07 --title="Feature Title"
```

### Task Creation

```bash
# New positional syntax - 3 arguments (epic, feature, title)
shark task create E07 F20 "Task Title"
shark task create e07 f20 "Task Title"  # Case insensitive

# New positional syntax - 2 arguments (combined epic-feature, title)
shark task create E07-F20 "Task Title"
shark task create e07-f20 "Task Title"  # Case insensitive

# Traditional flag syntax (still supported)
shark task create --epic=E07 --feature=F20 --title="Task Title"
```

## Syntax Compatibility

**All legacy syntax remains fully supported.** The new formats are additive improvements:

- ✅ Old commands continue to work unchanged
- ✅ Scripts don't need updates
- ✅ Mix and match syntaxes as preferred
- ✅ Case insensitivity works with all formats
- ✅ Backward compatibility guaranteed

## Dual Key Format Support

All `get`, `start`, `complete`, `approve`, `reopen`, `block`, and `unblock` commands support both numeric and slugged keys:

### Numeric Keys

- Epic: `E07`
- Feature: `E07-F01` or `F01`
- Task: `T-E07-F01-001` or `E07-F01-001`

### Slugged Keys

- Epic: `E07-user-management-system`
- Feature: `E07-F01-authentication` or `F01-authentication`
- Task: `T-E07-F01-001-implement-jwt-validation` or `E07-F01-001-implement-jwt-validation`

### Examples

```bash
# Using numeric keys
shark epic get E07
shark feature get E07-F01
shark task start E07-F01-001

# Using slugged keys (same entities)
shark epic get E07-user-management-system
shark feature get E07-F01-authentication
shark task start E07-F01-001-implement-jwt-validation
```

## Related Documentation

- [Epic Commands](epic-commands.md)
- [Feature Commands](feature-commands.md)
- [Task Commands](task-commands.md)
- [Error Messages](error-messages.md) - Invalid key format errors
