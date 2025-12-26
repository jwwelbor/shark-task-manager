# E07-F06: Streamline CLI Syntax with Positional Arguments

**Feature Title**: Streamlined CLI Syntax - Positional Arguments for Filters

**Epic**: E07 - CLI Enhancements
**Created**: 2025-12-21
**Status**: Draft
**Priority**: High
**Complexity**: Medium (5-8 story points)

---

## Feature Overview

Enhance the Shark CLI user experience by supporting positional arguments for common filtering operations, reducing verbosity while maintaining backward compatibility with existing flag-based syntax.

### Current State
```bash
shark feature list --epic=E04
shark task list --epic=E04 --feature=F01
```

### Desired State
```bash
shark feature list E04
shark task list E04 F01
shark task list E04-F01  # Alternative combined syntax
```

---

## Business Value

1. **Improved Usability**: Cleaner, more intuitive syntax reduces cognitive load
2. **Faster Workflow**: Fewer keystrokes for common operations (especially in AI agent contexts)
3. **CLI Convention Alignment**: Matches standard Unix CLI patterns (positional args for primary filters)
4. **Zero Disruption**: Backward compatible - existing scripts continue to work unchanged

---

## Scope & Boundaries

### In Scope
- Feature list command: positional epic key
- Task list command: positional epic key and/or feature key
- Pattern recognition for parsing arguments
- Full backward compatibility with flag-based syntax
- Comprehensive test coverage

### Out of Scope
- Other commands (epic list, feature create, task create, etc.)
- Other filtering parameters (status, agent type, priority, etc.) - remain flag-based for now
- New filtering capabilities - only syntax enhancement
- CLI completion/autocomplete suggestions

---

## Requirements

### Functional Requirements

#### FR1: Feature List Positional Arguments
- **ID**: FR1.1
- **Description**: Feature list command accepts 0 or 1 positional argument
- **Details**:
  - 0 args: List all features (no filter)
  - 1 arg: Treat as epic key (E##), filter features to that epic
  - 2+ args: Display clear error message
  - Must validate epic key format (E##)

#### FR1.2: Task List Positional Arguments
- **ID**: FR1.2
- **Description**: Task list command accepts 0, 1, or 2 positional arguments
- **Details**:
  - 0 args: List all tasks
  - 1 arg: Parse as either:
    - Epic key (E##): filter by epic
    - Feature key (E##-F##): parse and filter by epic + feature
  - 2 args:
    - First must be epic (E##)
    - Second can be feature suffix (F##) or full key (E##-F##)
    - Filter by epic + feature
  - 3+ args: Display clear error message

#### FR2: Backward Compatibility
- **ID**: FR2.1
- **Description**: All existing flag-based filtering continues to work unchanged
- **Details**:
  - `--epic=E04` continues to work
  - `--feature=F01` continues to work
  - Mixed flags + positional args: positional takes priority, flags ignored
  - Zero breaking changes to API or existing scripts

#### FR3: Pattern Recognition
- **ID**: FR3.1
- **Description**: Reliable pattern matching for epic and feature keys
- **Details**:
  - Epic pattern: Exactly `E` followed by 2 digits (E01-E99)
  - Feature pattern: `E##-F##` (e.g., E04-F01)
  - Feature suffix: `F` followed by 2 digits (F01-F99)
  - Case-sensitive (uppercase only)
  - Reject variations: E1, E001, e04, etc.

#### FR4: Error Handling
- **ID**: FR4.1
- **Description**: Clear, actionable error messages
- **Details**:
  - Invalid epic key format: "Error: Invalid epic key format. Expected E## (e.g., E04)"
  - Invalid feature key format: "Error: Invalid feature key format. Expected E##-F## or F## (e.g., E04-F01 or F01)"
  - Too many arguments: "Error: Too many positional arguments. Feature list accepts at most 1 positional argument"
  - Nonexistent epic/feature: Existing error handling preserved
  - Include suggestion to use `--help` for syntax examples

### Non-Functional Requirements

#### NFR1: Performance
- Argument parsing must be < 1ms
- No impact on command execution time
- No additional database queries

#### NFR2: Backward Compatibility
- 100% of existing workflows must continue to function
- No changes to JSON output format
- No changes to database schema

#### NFR3: Code Quality
- Test coverage > 85% for modified/new code
- No lint errors
- Follows project coding standards (CLAUDE.md)

---

## User Stories

### Story 1: Developer filters tasks by epic
**As a** developer
**I want to** list all tasks in an epic without typing `--epic=`
**So that** I can faster browse task options

**Acceptance Criteria**:
- `shark task list E04` returns all tasks in epic E04
- Results are identical to `shark task list --epic=E04`
- Works with all other flags (--status, --agent, etc.)

**Example**:
```bash
$ shark task list E04
Key         Title                      Status         Priority
T-E04-F01-001  Build login UI       in_progress    5
T-E04-F02-001  Setup database       todo           8
```

### Story 2: Developer filters tasks by epic and feature
**As a** developer
**I want to** list tasks in a specific feature without typing `--epic` and `--feature`
**So that** I get results faster with less command complexity

**Acceptance Criteria**:
- `shark task list E04 F01` returns tasks in E04/F01
- `shark task list E04-F01` produces identical results
- Both syntaxes are equivalent
- More intuitive than flag-based approach

**Example**:
```bash
$ shark task list E04 F01
Key         Title              Status         Priority
T-E04-F01-001  Build login UI    in_progress    5
T-E04-F01-002  Add 2FA support   todo           7
```

### Story 3: Product manager lists features for an epic
**As a** product manager
**I want to** see all features in an epic with just the epic key
**So that** I can quickly review epic scope without verbose syntax

**Acceptance Criteria**:
- `shark feature list E04` shows all features in E04
- Identical results to `shark feature list --epic=E04`
- Clear progress and task counts displayed

**Example**:
```bash
$ shark feature list E04
Key         Title                Status      Progress   Tasks
E04-F01     User Authentication draft       25.0%      2
E04-F02     API Authorization  active       50.0%      4
```

### Story 4: Existing scripts continue to work without modification
**As a** developer who uses automation scripts
**I want** all my existing `shark` commands to continue working
**So that** I don't have to update scripts

**Acceptance Criteria**:
- Old-style commands execute without modification
- Zero warnings or deprecation messages
- Identical output and behavior

**Example**:
```bash
# These all continue working exactly as before
shark feature list --epic=E04
shark task list --epic=E04 --feature=F01
shark task list --status=todo --epic=E04
```

---

## Design Details

### Argument Parsing Logic

#### Feature List
```
Input: shark feature list [arg1]

If arg1 is empty:
  → Apply no filter (list all)

If arg1 matches "E\d{2}":
  → Filter to epic (arg1)

If arg1 doesn't match expected patterns:
  → Error: "Invalid epic key format"
```

#### Task List
```
Input: shark task list [arg1] [arg2]

If no args:
  → Apply no filter (list all)

If 1 arg:
  If matches "E\d{2}":
    → Filter by epic (arg1)
  Else if matches "E\d{2}-F\d{2}":
    → Parse epic + feature, filter both
  Else:
    → Error: Invalid key format

If 2 args:
  If arg1 matches "E\d{2}":
    If arg2 matches "F\d{2}" or "E\d{2}-F\d{2}":
      → Parse both, filter by epic + feature
    Else:
      → Error: Invalid feature key format
  Else:
    → Error: Invalid epic key format

If 3+ args:
  → Error: Too many positional arguments
```

### Implementation Architecture

**New/Modified Files**:
1. `/internal/cli/commands/helpers.go` - Pattern matching functions
2. `/internal/cli/commands/feature.go` - Feature list argument parsing
3. `/internal/cli/commands/task.go` - Task list argument parsing
4. `/internal/cli/commands/*_test.go` - Test coverage

**Pattern Matching Functions** (new):
```go
// IsEpicKey returns true if s matches E## pattern
func IsEpicKey(s string) bool

// IsFeatureKey returns true if s matches E##-F## pattern
func IsFeatureKey(s string) bool

// IsFeatureKeySuffix returns true if s matches F## pattern
func IsFeatureKeySuffix(s string) bool

// ParseFeatureKey parses E##-F## and returns epic and feature keys
func ParseFeatureKey(s string) (epic, feature string, err error)
```

**Argument Parsing Functions** (new):
```go
// ParseFeatureListArgs parses positional arguments for feature list
func ParseFeatureListArgs(args []string) (epicKey *string, err error)

// ParseTaskListArgs parses positional arguments for task list
func ParseTaskListArgs(args []string) (epicKey *string, featureKey *string, err error)
```

---

## Testing Strategy

### Unit Tests (Pattern Matching)
- Valid epic keys (E01, E04, E07, E99)
- Invalid epic keys (E1, E001, e04, E, EE04, 04)
- Valid feature keys (E04-F01, E99-F99)
- Invalid feature keys (E04-F1, E4-F01, E04F01)
- Valid feature suffixes (F01, F99)
- Invalid feature suffixes (F1, F001, 01, F)

### Integration Tests (Feature List)
1. No args: List all features
2. Valid epic: Filter to features in that epic
3. Nonexistent epic: Error handling
4. Invalid format: Clear error message
5. Flag syntax: `--epic=E04` still works
6. Mixed: Positional takes priority over flag

### Integration Tests (Task List)
1. No args: List all tasks
2. Single epic: Filter by epic
3. Epic + feature (2 args): Filter both
4. Epic + feature (combined): `E04-F01` syntax works
5. Invalid formats: Clear errors
6. Nonexistent epic/feature: Proper error handling
7. Flag syntax: All flag combinations still work
8. Mixed flags + positional: Positional takes priority

### Test Coverage
- Target: > 85% for new/modified code
- Use database fixtures for integration tests
- Verify backward compatibility explicitly

---

## Help Text & Documentation

### Feature List Help
```
Usage:
  shark feature list [EPIC] [flags]

Examples:
  shark feature list              List all features
  shark feature list E04          List features in epic E04
  shark feature list --epic=E04   Same as above (flag syntax)

Positional Arguments:
  EPIC    Optional epic key (E##) to filter features

Flags:
  --epic string      Filter by epic key (alternative to positional arg)
  --status string    Filter by status: draft, active, completed, archived
  --sort-by string   Sort by: key, progress, status (default: key)
```

### Task List Help
```
Usage:
  shark task list [EPIC] [FEATURE] [flags]

Examples:
  shark task list                 List all tasks
  shark task list E04             List all tasks in epic E04
  shark task list E04 F01         List tasks in epic E04, feature F01
  shark task list E04-F01         Same as above (combined format)
  shark task list --epic=E04      Flag syntax (still supported)

Positional Arguments:
  EPIC      Optional epic key (E##) to filter by epic
  FEATURE   Optional feature key (F## or E##-F##) to filter by feature

Flags:
  --epic string       Filter by epic key (alternative to positional)
  --feature string    Filter by feature key (alternative to positional)
  --status string     Filter by status: todo, in_progress, completed, blocked
  --agent string      Filter by assigned agent type
  --priority-min int  Minimum priority (1-10)
  --priority-max int  Maximum priority (1-10)
```

---

## Acceptance Criteria Checklist

- [ ] Pattern matching helpers implemented and tested
- [ ] Feature list supports 1 positional argument
- [ ] Task list supports 1-2 positional arguments
- [ ] Task list supports combined format (E##-F##)
- [ ] All existing flag-based commands continue to work
- [ ] Help text updated with positional argument examples
- [ ] Integration tests pass (> 20 test cases)
- [ ] Unit tests pass (> 15 test cases)
- [ ] Code coverage > 85%
- [ ] No lint errors
- [ ] No regressions in other commands
- [ ] Manual testing successful with real CLI
- [ ] Documentation updated (README examples)
- [ ] PR ready with comprehensive description

---

## Implementation Tasks

This feature breaks down into 8 concrete development tasks:

1. **T-E07-F06-001**: Implement pattern matching helpers
2. **T-E07-F06-002**: Implement feature list argument parsing
3. **T-E07-F06-003**: Implement task list argument parsing
4. **T-E07-F06-004**: Write unit tests for pattern matching
5. **T-E07-F06-005**: Write integration tests for feature list
6. **T-E07-F06-006**: Write integration tests for task list
7. **T-E07-F06-007**: Update documentation and help text
8. **T-E07-F06-008**: Code review, testing, and integration

See `docs/E07-POSITIONAL-ARGS-COORDINATION.md` for detailed task specifications.

---

## Success Metrics

1. **User Experience**: Positional syntax reduces average command length by 25-40%
2. **Quality**: 100% backward compatibility verified
3. **Coverage**: > 85% test coverage for modified files
4. **Performance**: Zero performance impact
5. **Adoption**: New syntax available immediately after merge

---

## Dependencies & Prerequisites

- Go 1.23.4+
- Cobra CLI framework (existing)
- SQLite database (existing)
- All tests passing before implementation

---

## Risk Assessment

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|-----------|
| Break flag-based commands | HIGH | LOW | Comprehensive backward compat tests |
| Ambiguous parsing | MEDIUM | LOW | Clear pattern rules + validation |
| User confusion | MEDIUM | MEDIUM | Clear help text + examples |
| Edge case interactions | MEDIUM | LOW | Integration tests |

---

## Timeline

**Total Effort**: 7-8 hours
**Complexity**: Medium (5-8 story points)

Can be completed in a single development session by a single developer.

---

## Related Documentation

- `/docs/CLI_REFERENCE.md` - CLI usage guide
- `/docs/E07-POSITIONAL-ARGS-COORDINATION.md` - Detailed coordination document
- `/CLAUDE.md` - Project standards and guidelines
- `/README.md` - Usage examples

---

## Revision History

| Date | Author | Change |
|------|--------|--------|
| 2025-12-21 | ProductManager | Created feature specification |

---
