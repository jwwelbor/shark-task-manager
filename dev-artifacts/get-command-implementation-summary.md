# Unified `get` Command Implementation Summary

## Overview
Implemented unified `shark get` command with smart positional argument routing using Test-Driven Development (TDD).

## Features Implemented

### Unified `shark get` Command
Smart dispatcher that routes to appropriate subcommand based on arguments:

**Epic retrieval:**
- `shark get E10` → Get epic E10 details

**Feature retrieval:**
- `shark get E10 F01` → Get feature E10-F01 details
- `shark get E10-F01` → Get feature E10-F01 details (combined format)

**Task retrieval (multiple formats):**
- `shark get E10 F01 001` → Get task T-E10-F01-001 details (full task number)
- `shark get E10 F01 1` → Get task T-E10-F01-001 details (short task number, auto-padded)
- `shark get T-E10-F01-001` → Get task T-E10-F01-001 details (full task key)

## Implementation Details

### Parser Function: `ParseGetArgs()`
Located in `internal/cli/commands/helpers.go`

**Function Signature:**
```go
func ParseGetArgs(args []string) (command string, key string, err error)
```

**Returns:**
- `command`: "epic", "feature", or "task"
- `key`: The full key to pass to the get command
- `err`: Error if parsing fails

**Supported Argument Patterns:**
1. 1 arg (E##) → epic get
2. 1 arg (E##-F##) → feature get
3. 1 arg (T-E##-F##-###) → task get
4. 2 args (E## F##) → feature get
5. 3 args (E## F## ###) → task get (auto-pads task number to 3 digits)

### Helper Functions

**`isTaskKey(s string) bool`**
- Validates task key format: T-E##-F##-###
- Checks exact length (13 characters)
- Validates each component

**`parseTaskNumber(s string) (int, error)`**
- Parses task number from string
- Validates range: 1-999
- Returns error for invalid numbers

### Command Dispatcher
File: `internal/cli/commands/get.go`

Routes to appropriate subcommands:
- `runEpicGet()` for epic keys
- `runFeatureGet()` for feature keys
- `runTaskGet()` for task keys

## Test-Driven Development (TDD) Process

### Phase 1: Red (Write Failing Tests)
Created comprehensive unit tests:
- File: `internal/cli/commands/get_test.go`
- 14 test cases covering:
  - All valid argument combinations
  - Task number padding (001 vs 1)
  - Full task key parsing
  - Invalid formats
  - Error handling

### Phase 2: Green (Implement to Pass)
Implemented parser and helper functions:
- `ParseGetArgs()` - Main parsing logic
- `isTaskKey()` - Task key validation
- `parseTaskNumber()` - Task number parsing with range validation

### Phase 3: Refactor (End-to-End Testing)
Created comprehensive integration test script:
- File: `test-get-command.sh`
- Tests all command variations with real data
- Verified both human-readable and JSON output
- Tested error handling

## Test Results
✅ All unit tests passing (14/14 test cases)
✅ All existing command tests still passing
✅ End-to-end integration tests verified

## Files Modified/Created
1. `internal/cli/commands/get.go` (NEW - command dispatcher)
2. `internal/cli/commands/get_test.go` (NEW - unit tests)
3. `internal/cli/commands/helpers.go` (UPDATED - added ParseGetArgs, isTaskKey, parseTaskNumber)
4. `test-get-command.sh` (NEW - integration tests)

## Usage Examples

```bash
# Get epic details
shark get E10

# Get feature details (two formats)
shark get E10 F01
shark get E10-F01

# Get task details (multiple formats)
shark get E10 F01 001           # Full task number
shark get E10 F01 1             # Short task number (auto-padded to 001)
shark get T-E10-F01-001         # Full task key

# JSON output
shark get E10 --json
shark get T-E10-F01-001 --json
```

## Key Features

### Smart Task Number Handling
- Accepts short form: `1` → auto-pads to `001`
- Accepts full form: `001` → used as-is
- Validates range: 1-999
- Constructs full task key: `T-E10-F01-001`

### Flexible Input Formats
- Accepts combined keys: `E10-F01`
- Accepts separate args: `E10 F01`
- Accepts full task keys: `T-E10-F01-001`
- Consistent behavior across all formats

### Error Handling
- Clear error messages for invalid formats
- Helpful syntax examples in error output
- Validates all components before routing

## Backward Compatibility
- No breaking changes to existing commands
- All existing `epic get`, `feature get`, and `task get` commands work as before
- New unified `get` command provides convenient alternative

## Completion
✅ TDD approach followed throughout
✅ All tests passing
✅ Backward compatibility maintained
✅ End-to-end testing completed
✅ Code formatted and ready for commit
