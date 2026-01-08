# CLI UX Specification: Standardized Command Patterns

**Feature**: E10-F20 - Standardize CLI Command Options
**Created**: 2026-01-03
**Status**: Draft - Pending Approval
**Owner**: UX Designer + Product Manager

---

## Executive Summary

This specification defines standardized, AI-agent friendly command patterns for the shark CLI tool. The goal is to create consistent, predictable interfaces across all commands with case-insensitive key handling, flexible positional arguments, and clear error messages.

---

## Design Principles

### 1. **AI Agent First**
- Clear, predictable patterns that are easy to generate programmatically
- Minimal ambiguity in syntax
- Machine-readable error messages with clear resolution paths
- JSON output support for all commands

### 2. **Progressive Disclosure**
- Simple cases should be simple: `shark list E01`
- Complex cases should be possible: `shark list E01 F02 --status=todo --agent=backend`
- Flags provide additional control without cluttering simple use cases

### 3. **Cognitive Consistency**
- Same patterns work across all command types (epic, feature, task)
- Positional arguments follow natural hierarchy: Epic → Feature → Task
- Case insensitivity reduces cognitive load and typo errors

### 4. **Graceful Degradation**
- Multiple valid syntaxes for the same operation (positional + flags)
- Backward compatibility with existing flag-based commands
- Clear migration path for deprecated patterns

---

## Key Format Standards

### Case Insensitivity

**Current Behavior** (❌ Inconsistent):
```bash
shark epic get E01     # ✓ Works
shark epic get e01     # ✗ Fails - "invalid epic key format"
shark epic get E01-feature-name  # ✓ Works (slugged)
```

**Proposed Behavior** (✅ Consistent):
```bash
shark epic get E01     # ✓ Works
shark epic get e01     # ✓ Works (normalized to E01)
shark epic get E-01    # ✗ Fails with helpful error
shark epic get E001    # ✗ Fails with helpful error
```

**Implementation Notes**:
- Normalize keys to uppercase before validation
- Regex patterns: `(?i)^e\d{2}$` → `^E\d{2}$` (normalize first, then match)
- Error messages show normalized format: "Invalid key 'e-01', expected format: E##"

### Supported Key Formats

#### Epic Keys
```
Numeric:  E01, E04, E99
Slugged:  E01-epic-name, E04-enhancements
Invalid:  E1, e01 (before normalization), E-01, E001
```

#### Feature Keys
```
Full:     E01-F02, E04-F01
Partial:  F02, F01
Slugged:  E01-F02-feature-name, F02-feature-name
Invalid:  E01F02, E1-F2, F1
```

#### Task Keys
```
Full:     T-E01-F02-001
Short:    E01-F02-001 (drop T- prefix) ✨ NEW
Slugged:  T-E01-F02-001-task-name, E01-F02-001-task-name
Invalid:  T-E1-F2-1, TE01F02001, E1-F2-1
```

**Case Normalization Rules**:
1. Convert input to uppercase
2. Validate against regex pattern
3. Use normalized key for lookups
4. Return normalized key in responses

**Task Key Prefix Rules** (✨ NEW):
1. If key starts with `T-` → use as-is
2. If key matches `E\d{2}-F\d{2}-\d{3}` → prepend `T-`
3. Validate final key against task pattern
4. Return canonical format with `T-` prefix

---

## Command Pattern Taxonomy

### Pattern A: List Commands (Query Multiple)

**Signature**:
```
shark <entity> list [EPIC] [FEATURE] [--filters...] [--json]
```

**Examples**:
```bash
# Epic list
shark epic list                    # List all epics
shark epic list --json             # JSON output

# Feature list
shark feature list                 # List all features
shark feature list E01             # Features in epic E01
shark feature list e01             # Features in epic E01 (normalized)
shark feature list --epic=E01      # Flag syntax (backward compatible)

# Task list
shark task list                    # List all tasks
shark task list E01                # Tasks in epic E01
shark task list E01 F02            # Tasks in E01-F02
shark task list E01-F02            # Alternative syntax
shark task list e01 f02            # Case insensitive
shark task list --epic=E01 --feature=F02  # Flag syntax
shark task list E01 --status=todo  # Mixed: positional + filter
```

**Argument Precedence**:
1. Positional arguments parsed first
2. Flags override positional if both provided
3. Warning if both positional and flag syntax used (except for filters)

### Pattern B: Get Commands (Query Single)

**Signature**:
```
shark <entity> get <KEY> [--json]
```

**Examples**:
```bash
# Epic get
shark epic get E01                 # Numeric key
shark epic get e01                 # Case insensitive
shark epic get E01-epic-name       # Slugged key
shark epic get e01-epic-name       # Case insensitive slug

# Feature get
shark feature get E01-F02          # Full key
shark feature get F02              # Partial key
shark feature get f02              # Case insensitive
shark feature get E01-F02-auth     # Slugged key

# Task get
shark task get T-E01-F02-001       # Numeric key (full)
shark task get t-e01-f02-001       # Case insensitive
shark task get e01-f02-001         # Short format (drop T-) ✨ NEW
shark task get E01-F02-001         # Short format, any case ✨ NEW
shark task get T-E01-F02-001-name  # Slugged key
shark task get e01-f02-001-name    # Short slugged key ✨ NEW
```

**Case Handling**:
- Accept any case in input
- Normalize to uppercase before lookup
- Return canonical uppercase format in output

### Pattern C: Create Commands (Mutate - Add)

**Current Signature** (❌ Flag-based):
```bash
shark feature create --epic=E01 "Feature Title"
shark task create --epic=E01 --feature=F02 "Task Title"
```

**Proposed Signature** (✅ Positional + Flags):
```bash
# Option 1: Positional context (RECOMMENDED)
shark feature create E01 "Feature Title" [--flags...]
shark task create E01 F02 "Task Title" [--flags...]

# Option 2: Flag-based (backward compatible)
shark feature create --epic=E01 "Feature Title"
shark task create --epic=E01 --feature=F02 "Task Title"

# Case insensitive
shark feature create e01 "Feature Title"
shark task create e01 f02 "Task Title"
```

**Examples**:
```bash
# Epic create (no parent context)
shark epic create "User Management System"
shark epic create "User Management" --priority=high

# Feature create
shark feature create E01 "Authentication"
shark feature create e01 "Authentication"  # Case insensitive
shark feature create E01 "Auth" --execution-order=1
shark feature create --epic=E01 "Auth"     # Flag syntax

# Task create
shark task create E01 F02 "Implement JWT validation"
shark task create e01 f02 "Implement JWT" --agent=backend
shark task create --epic=E01 --feature=F02 "JWT" --priority=3
```

**Argument Parsing Rules**:
1. If first arg matches epic pattern → positional context
2. If `--epic` flag present → flag context
3. If both → flags take precedence with warning
4. Last positional arg is title (quoted or unquoted)

### Pattern D: Action Commands (Mutate - State Change)

**Signature**:
```
shark <entity> <action> <KEY> [--flags...] [--json]
```

**Examples**:
```bash
# Task lifecycle
shark task start T-E01-F02-001
shark task start t-e01-f02-001         # Case insensitive
shark task start e01-f02-001           # Short format (drop T-) ✨ NEW
shark task complete T-E01-F02-001 --notes="Done"
shark task complete e01-f02-001 --notes="Done"  # Short format ✨ NEW
shark task approve T-E01-F02-001
shark task block T-E01-F02-001 --reason="Blocked"

# Feature actions
shark feature complete E01-F02
shark feature complete f02             # Case insensitive
```

**Case Handling**: Same as Get commands

---

## Error Message Standards

### Invalid Format Errors

**Current** (❌):
```
Error: invalid epic key format: "e01" (expected E##, e.g., E04)
```

**Proposed** (✅):
```
Error: Invalid epic key "e-01"
  Expected format: E## (e.g., E01, E04, E99)
  Tip: Use two-digit numbers (E01, not E1)
  Case insensitive: e01, E01, and e01 are all valid
```

### Ambiguous Arguments

```
Error: Ambiguous arguments detected
  Positional: epic=E01 (from argument)
  Flag: epic=E02 (from --epic flag)
  Resolution: Remove positional argument or remove --epic flag
```

### Case Normalization Notice

Only show when `--verbose` is enabled:
```
[DEBUG] Normalized key: e01 → E01
[DEBUG] Normalized key: t-e04-f02-001 → T-E04-F02-001
```

---

## Implementation Checklist

### Phase 1: Case Insensitivity (Non-Breaking)
- [ ] Update regex patterns in `helpers.go`
- [ ] Add normalization function: `NormalizeKey(input string) string`
- [ ] Update `IsEpicKey`, `IsFeatureKey`, `IsTaskKey` to normalize first
- [ ] Update all repository lookups to use normalized keys
- [ ] Add tests for case-insensitive matching
- [ ] Update error messages to show expected format

### Phase 1.5: Short Task Key Format (Non-Breaking) ✨ NEW
- [ ] Add task key prefix detection in `helpers.go`
- [ ] Implement `NormalizeTaskKey(input string) string` to add T- if missing
- [ ] Update `IsTaskKey` to accept both formats (T-E##-F##-### and E##-F##-###)
- [ ] Add tests for short key format parsing
- [ ] Update error messages to mention short format option
- [ ] Document short format in examples

### Phase 2: Positional Arguments for Create (Non-Breaking)
- [ ] Update `featureCreateCmd` to accept positional epic arg
- [ ] Update `taskCreateCmd` to accept positional epic + feature args
- [ ] Add argument parsing logic with flag precedence
- [ ] Add tests for positional + flag combinations
- [ ] Add warning for conflicting positional + flag syntax
- [ ] Update documentation and examples

### Phase 3: Standardize Error Messages (Non-Breaking)
- [ ] Create error message templates
- [ ] Update all validation errors to use templates
- [ ] Add contextual tips to error messages
- [ ] Add case insensitivity hints
- [ ] Test error messages with AI agents

### Phase 4: Documentation Updates (Non-Breaking)
- [ ] Update CLI_REFERENCE.md with new patterns
- [ ] Add examples for case-insensitive usage
- [ ] Add examples for positional create commands
- [ ] Update CLAUDE.md with new patterns
- [ ] Add migration guide for deprecated patterns (if any)

---

## Backward Compatibility

### Guarantees

1. **All existing flag-based commands continue to work**
   - `--epic=E01` syntax remains supported
   - `--feature=F02` syntax remains supported
   - No breaking changes to existing automation

2. **Case sensitivity is additive**
   - `E01` continues to work exactly as before
   - `e01` now also works (previously failed)
   - No behavior change for uppercase keys

3. **Positional arguments are additive**
   - New optional syntax, not required
   - Flag syntax remains primary in documentation
   - Positional syntax shown as "alternative" or "shorthand"

### Deprecation Policy

**Nothing is deprecated in this change.**

All changes are additive:
- New case handling is strictly more permissive
- New positional syntax is optional alternative
- Existing patterns continue to work unchanged

---

## Testing Strategy

### Unit Tests

```go
// Case normalization
func TestNormalizeKey(t *testing.T) {
    tests := []struct{
        input string
        want  string
    }{
        {"E01", "E01"},
        {"e01", "E01"},
        {"E-01", "E-01"},  // Invalid, but normalized
        {"t-e04-f02-001", "T-E04-F02-001"},
    }
}

// Positional argument parsing
func TestParseFeatureCreateArgs(t *testing.T) {
    tests := []struct{
        args     []string
        wantEpic string
        wantTitle string
    }{
        {[]string{"E01", "Feature Title"}, "E01", "Feature Title"},
        {[]string{"e01", "Feature Title"}, "E01", "Feature Title"},
    }
}
```

### Integration Tests

```bash
# Case insensitivity
shark epic create "Test Epic"  # Create E01
shark epic get e01             # Should work
shark feature create e01 "Test Feature"  # Should work
shark task list e01            # Should work

# Positional syntax
shark feature create E02 "New Feature"
shark task create E02 F01 "New Task"

# Mixed syntax
shark task list E01 --status=todo  # Positional epic + flag filter
```

### AI Agent Tests

```python
# Simulated AI agent usage
def test_ai_agent_workflow():
    # Agent shouldn't need to worry about case
    run("shark task create e01 f01 'implement feature'")
    run("shark task start t-e01-f01-001")
    run("shark task complete T-E01-F01-001")  # Different case

    # Agent can use natural positional syntax
    run("shark list e01")
    run("shark list e01 f01")
```

---

## Examples: Before & After

### Example 1: Creating a Task

**Before** (current):
```bash
shark task create \
  --epic=E01 \
  --feature=F02 \
  --title="Implement JWT validation" \
  --agent=backend \
  --priority=3
```

**After** (proposed, both work):
```bash
# Positional syntax (preferred for AI agents)
shark task create E01 F02 "Implement JWT validation" \
  --agent=backend \
  --priority=3

# Flag syntax (still works)
shark task create \
  --epic=E01 \
  --feature=F02 \
  "Implement JWT validation" \
  --agent=backend \
  --priority=3
```

### Example 2: Case Sensitivity

**Before** (current):
```bash
shark epic get E01         # ✓ Works
shark epic get e01         # ✗ Error: invalid epic key format
```

**After** (proposed):
```bash
shark epic get E01         # ✓ Works
shark epic get e01         # ✓ Works (normalized to E01)
shark epic get E-01        # ✗ Error: Invalid format (with helpful message)
```

### Example 3: Listing Tasks

**Before** (current):
```bash
# Both syntaxes already work
shark task list E01 F02
shark task list --epic=E01 --feature=F02
```

**After** (proposed - enhanced):
```bash
# All of these now work
shark task list E01 F02
shark task list e01 f02        # Case insensitive
shark task list e01-f02        # Combined format
shark task list --epic=E01 --feature=F02
shark task list E01 --status=todo    # Mixed
```

---

## AI Agent Integration Guide

### Code Generation Patterns

AI agents can generate commands using simple templates:

```python
# Template 1: List tasks in feature
def list_tasks(epic, feature):
    return f"shark task list {epic.lower()} {feature.lower()} --json"

# Template 2: Create task
def create_task(epic, feature, title, agent="general", priority=5):
    return f"shark task create {epic} {feature} '{title}' --agent={agent} --priority={priority} --json"

# Template 3: Start task
def start_task(task_key):
    # Case doesn't matter
    return f"shark task start {task_key.lower()} --json"
```

### Error Handling

```python
import subprocess
import json

def run_shark_command(cmd):
    result = subprocess.run(
        cmd.split(),
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        # Parse error message
        error = result.stderr

        # Common case: invalid key format
        if "invalid" in error.lower() and "format" in error.lower():
            # Extract suggested format from error message
            # New error messages include "Expected format: E##"
            pass

        raise Exception(f"Command failed: {error}")

    return json.loads(result.stdout) if "--json" in cmd else result.stdout
```

---

## Decision Log

### Decision 1: Case Insensitivity Everywhere

**Options Considered**:
1. Keep case sensitivity (status quo)
2. Add case insensitivity for all keys
3. Add case insensitivity only for prefixes (E, F, T)

**Decision**: Option 2 - Case insensitivity for all keys

**Rationale**:
- AI agents shouldn't need case-specific logic
- Human users make typos
- Uppercase is just a convention, not semantic
- No downside - uppercase remains canonical

### Decision 2: Positional Arguments for Create Commands

**Options Considered**:
1. Keep flag-based syntax only
2. Add positional syntax, deprecate flags
3. Support both flag and positional syntax

**Decision**: Option 3 - Support both

**Rationale**:
- Positional syntax is cleaner for simple cases
- Flags provide clarity for complex cases
- Backward compatibility is critical
- AI agents can use either pattern

### Decision 3: Argument Precedence (Positional vs Flags)

**Options Considered**:
1. Positional takes precedence
2. Flags take precedence
3. Error if both provided

**Decision**: Option 2 - Flags take precedence with warning

**Rationale**:
- Flags are more explicit
- Allows gradual migration
- Warning prevents silent conflicts
- Matches common CLI conventions (e.g., git)

---

## Approval

**Status**: Draft - Pending Review

**Reviewers**:
- [ ] Product Manager - Scope and priority approval
- [ ] Tech Lead - Implementation feasibility
- [ ] AI Agent Team - Usability validation
- [ ] QA - Test coverage review

**Next Steps**:
1. Review with stakeholders
2. Gather feedback on examples
3. Update based on feedback
4. Approve for implementation
5. Create implementation tasks in shark

---

## Related Documents

- `/home/jwwelbor/projects/shark-task-manager/docs/CLI_REFERENCE.md` - Current CLI documentation
- `/home/jwwelbor/projects/shark-task-manager/CLAUDE.md` - Development guidelines
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers.go` - Current parsing logic
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/shared_flags.go` - Flag definitions

---

## Appendix A: Complete Command Reference

### Epic Commands

```bash
# List
shark epic list                    # All epics
shark epic list --json             # JSON output

# Get
shark epic get E01                 # Numeric
shark epic get e01                 # Case insensitive
shark epic get E01-epic-name       # Slugged

# Create
shark epic create "Epic Title"
shark epic create "Epic Title" --priority=high

# Update (not shown, follows get pattern)
shark epic update E01 --title="New Title"
```

### Feature Commands

```bash
# List
shark feature list                 # All features
shark feature list E01             # In epic E01
shark feature list e01             # Case insensitive
shark feature list --epic=E01      # Flag syntax

# Get
shark feature get E01-F02          # Full key
shark feature get f02              # Partial, case insensitive
shark feature get E01-F02-name     # Slugged

# Create (NEW)
shark feature create E01 "Feature Title"
shark feature create e01 "Feature" --execution-order=1
shark feature create --epic=E01 "Feature"  # Flag syntax
```

### Task Commands

```bash
# List
shark task list                    # All tasks
shark task list E01                # In epic
shark task list E01 F02            # In feature
shark task list e01-f02            # Combined format
shark task list E01 --status=todo  # Mixed

# Get
shark task get T-E01-F02-001       # Full format
shark task get t-e01-f02-001       # Case insensitive
shark task get e01-f02-001         # Short format ✨ NEW
shark task get T-E01-F02-001-name  # Slugged

# Create (NEW)
shark task create E01 F02 "Task Title"
shark task create e01 f02 "Task" --agent=backend
shark task create --epic=E01 --feature=F02 "Task"  # Flag syntax

# Actions
shark task start t-e01-f02-001             # Case insensitive
shark task start e01-f02-001               # Short format ✨ NEW
shark task complete T-E01-F02-001 --notes="Done"
shark task complete e01-f02-001 --notes="Done"  # Short format ✨ NEW
shark task approve T-E01-F02-001
shark task approve e01-f02-001             # Short format ✨ NEW
shark task block T-E01-F02-001 --reason="Blocked"
```

---

## Appendix B: Regex Pattern Reference

### Current Patterns (Case Sensitive)

```go
epicKeyPattern       = regexp.MustCompile(`^E\d{2}$`)
featureKeyPattern    = regexp.MustCompile(`^E\d{2}-F\d{2}$`)
featureSuffixPattern = regexp.MustCompile(`^F\d{2}$`)
taskKeyPattern       = regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`)
```

### Proposed Patterns (Case Insensitive - After Normalization)

```go
// Same patterns, but input is normalized to uppercase first
func NormalizeKey(input string) string {
    return strings.ToUpper(input)
}

// Then validate with existing patterns
func IsEpicKey(s string) bool {
    normalized := NormalizeKey(s)
    return epicKeyPattern.MatchString(normalized)
}
```

### Task Key Normalization (✨ NEW - Short Format Support)

```go
// Pattern for short task key (without T- prefix)
shortTaskKeyPattern = regexp.MustCompile(`^E\d{2}-F\d{2}-\d{3}$`)

// Normalize task key - add T- prefix if missing
func NormalizeTaskKey(input string) (string, error) {
    normalized := strings.ToUpper(input)

    // Already has T- prefix
    if strings.HasPrefix(normalized, "T-") {
        return normalized, nil
    }

    // Check if it matches short format (E##-F##-###)
    if shortTaskKeyPattern.MatchString(normalized) {
        return "T-" + normalized, nil
    }

    // Check if it matches slug format (E##-F##-###-slug or just has T-)
    if strings.Contains(normalized, "-") {
        parts := strings.SplitN(normalized, "-", 4)
        if len(parts) >= 3 && shortTaskKeyPattern.MatchString(strings.Join(parts[:3], "-")) {
            return "T-" + normalized, nil
        }
    }

    return normalized, fmt.Errorf("invalid task key format")
}
```

**Alternative**: Use case-insensitive regex
```go
epicKeyPattern = regexp.MustCompile(`(?i)^e\d{2}$`)
```

**Recommendation**: Normalize first (cleaner, easier to debug)
