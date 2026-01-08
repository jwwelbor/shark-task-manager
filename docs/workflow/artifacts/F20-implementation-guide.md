# Implementation Guide: CLI UX Standardization

**Feature**: E10-F20 - Standardize CLI Command Options
**Created**: 2026-01-03
**Audience**: Development Team

---

## Overview

This guide provides concrete implementation steps for the CLI UX improvements. All changes are **non-breaking** and can be implemented incrementally.

---

## Phase 1: Case Insensitivity (Priority 1)

### 1.1 Add Key Normalization Function

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers.go`

**Changes**:

```go
// Add after the regex pattern declarations

// NormalizeKey converts a key to canonical uppercase format.
// This enables case-insensitive key handling throughout the CLI.
//
// Examples:
//   e01 -> E01
//   t-e04-f02-001 -> T-E04-F02-001
//   E01-FEATURE-NAME -> E01-FEATURE-NAME
func NormalizeKey(key string) string {
	return strings.ToUpper(key)
}
```

### 1.2 Update Validation Functions

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers.go`

**Before**:
```go
// IsEpicKey validates if a string is a valid epic key format (E##)
func IsEpicKey(s string) bool {
	return epicKeyPattern.MatchString(s)
}
```

**After**:
```go
// IsEpicKey validates if a string is a valid epic key format (E##)
// Case insensitive: e01, E01, and E-01 → normalized to E01 before validation
func IsEpicKey(s string) bool {
	normalized := NormalizeKey(s)
	return epicKeyPattern.MatchString(normalized)
}
```

**Repeat for**:
- `IsFeatureKey()`
- `IsFeatureKeySuffix()`

### 1.3 Update Parsing Functions

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers.go`

**Update `ParseFeatureKey`**:

```go
func ParseFeatureKey(s string) (epic, feature string, err error) {
	// Normalize to uppercase first
	normalized := NormalizeKey(s)

	if !featureKeyPattern.MatchString(normalized) {
		return "", "", fmt.Errorf("invalid feature key format: %q (expected E##-F##)", s)
	}

	// Split on hyphen - we know it's valid format at this point
	epic = normalized[:3]   // E##
	feature = normalized[4:7] // F##

	return epic, feature, nil
}
```

**Update `ParseFeatureListArgs`**:

```go
func ParseFeatureListArgs(args []string) (*string, error) {
	if len(args) == 0 {
		return nil, nil
	}

	if len(args) > 1 {
		return nil, fmt.Errorf("too many positional arguments: feature list accepts at most 1 positional argument (got %d)", len(args))
	}

	// Normalize the epic key
	epicKey := NormalizeKey(args[0])

	// Validate format
	if !IsEpicKey(epicKey) {
		return nil, fmt.Errorf("invalid epic key format: %q (expected E##, e.g., E04). Case insensitive.", args[0])
	}

	return &epicKey, nil
}
```

**Update `ParseTaskListArgs`**:

```go
func ParseTaskListArgs(args []string) (*string, *string, error) {
	if len(args) == 0 {
		return nil, nil, nil
	}

	if len(args) > 2 {
		return nil, nil, fmt.Errorf("too many positional arguments: task list accepts at most 2 positional arguments (got %d)", len(args))
	}

	// Single argument case
	if len(args) == 1 {
		normalized := NormalizeKey(args[0])

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(normalized) {
			epic, feature, err := ParseFeatureKey(normalized)
			if err != nil {
				return nil, nil, err
			}
			return &epic, &feature, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(normalized) {
			return &normalized, nil, nil
		}

		return nil, nil, fmt.Errorf("invalid key format: %q (expected E## or E##-F##). Case insensitive.", args[0])
	}

	// Two argument case
	epicNormalized := NormalizeKey(args[0])
	featureNormalized := NormalizeKey(args[1])

	if !IsEpicKey(epicNormalized) {
		return nil, nil, fmt.Errorf("invalid epic key format: %q (expected E##). Case insensitive.", args[0])
	}

	// Feature can be F## or E##-F##
	if IsFeatureKeySuffix(featureNormalized) {
		return &epicNormalized, &featureNormalized, nil
	}

	if IsFeatureKey(featureNormalized) {
		_, featureSuffix, err := ParseFeatureKey(featureNormalized)
		if err != nil {
			return nil, nil, err
		}
		return &epicNormalized, &featureSuffix, nil
	}

	return nil, nil, fmt.Errorf("invalid feature key format: %q (expected F## or E##-F##). Case insensitive.", args[1])
}
```

### 1.4 Add Tests

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers_test.go`

```go
package commands

import (
	"testing"
)

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"uppercase epic", "E01", "E01"},
		{"lowercase epic", "e01", "E01"},
		{"mixed case epic", "E01", "E01"},
		{"uppercase feature", "E04-F02", "E04-F02"},
		{"lowercase feature", "e04-f02", "E04-F02"},
		{"mixed case feature", "e04-F02", "E04-F02"},
		{"lowercase task", "t-e04-f02-001", "T-E04-F02-001"},
		{"mixed case task", "T-e04-F02-001", "T-E04-F02-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeKey(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsEpicKey_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"uppercase valid", "E01", true},
		{"lowercase valid", "e01", true},
		{"mixed case valid", "E01", true},
		{"invalid format uppercase", "E1", false},
		{"invalid format lowercase", "e1", false},
		{"invalid separator", "E-01", false},
		{"too many digits", "E001", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEpicKey(tt.input)
			if got != tt.want {
				t.Errorf("IsEpicKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFeatureKey_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantEpic    string
		wantFeature string
		wantErr     bool
	}{
		{"uppercase", "E04-F02", "E04", "F02", false},
		{"lowercase", "e04-f02", "E04", "F02", false},
		{"mixed case", "E04-f02", "E04", "F02", false},
		{"invalid format", "E04F02", "", "", true},
		{"missing epic", "F02", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEpic, gotFeature, err := ParseFeatureKey(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFeatureKey(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if gotEpic != tt.wantEpic {
				t.Errorf("ParseFeatureKey(%q) epic = %q, want %q", tt.input, gotEpic, tt.wantEpic)
			}

			if gotFeature != tt.wantFeature {
				t.Errorf("ParseFeatureKey(%q) feature = %q, want %q", tt.input, gotFeature, tt.wantFeature)
			}
		})
	}
}

func TestParseTaskListArgs_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantEpic    *string
		wantFeature *string
		wantErr     bool
	}{
		{
			name:        "lowercase epic",
			args:        []string{"e04"},
			wantEpic:    strPtr("E04"),
			wantFeature: nil,
			wantErr:     false,
		},
		{
			name:        "lowercase epic and feature",
			args:        []string{"e04", "f02"},
			wantEpic:    strPtr("E04"),
			wantFeature: strPtr("F02"),
			wantErr:     false,
		},
		{
			name:        "mixed case combined",
			args:        []string{"E04-f02"},
			wantEpic:    strPtr("E04"),
			wantFeature: strPtr("F02"),
			wantErr:     false,
		},
		{
			name:        "invalid epic format",
			args:        []string{"e1"},
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEpic, gotFeature, err := ParseTaskListArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskListArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
				return
			}

			if !ptrEqual(gotEpic, tt.wantEpic) {
				t.Errorf("ParseTaskListArgs(%v) epic = %v, want %v", tt.args, ptrStr(gotEpic), ptrStr(tt.wantEpic))
			}

			if !ptrEqual(gotFeature, tt.wantFeature) {
				t.Errorf("ParseTaskListArgs(%v) feature = %v, want %v", tt.args, ptrStr(gotFeature), ptrStr(tt.wantFeature))
			}
		})
	}
}

// Helper functions for tests
func strPtr(s string) *string {
	return &s
}

func ptrStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func ptrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
```

### 1.5 Integration Tests

**File**: Create `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/case_insensitive_integration_test.go`

```go
package commands

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/test"
)

func TestCaseInsensitiveKeys_Integration(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create test data with uppercase keys
	_, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description)
		VALUES ('E99', 'Test Epic', 'Test epic for case insensitivity')
	`)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO features (key, epic_id, title, description)
		VALUES ('E99-F99', (SELECT id FROM epics WHERE key = 'E99'), 'Test Feature', 'Test feature')
	`)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	// Test case-insensitive parsing
	tests := []struct {
		name  string
		input []string
		want  struct {
			epic    *string
			feature *string
		}
	}{
		{
			name:  "lowercase epic",
			input: []string{"e99"},
			want: struct {
				epic    *string
				feature *string
			}{
				epic:    strPtr("E99"),
				feature: nil,
			},
		},
		{
			name:  "lowercase feature",
			input: []string{"e99", "f99"},
			want: struct {
				epic    *string
				feature *string
			}{
				epic:    strPtr("E99"),
				feature: strPtr("F99"),
			},
		},
		{
			name:  "mixed case",
			input: []string{"E99", "f99"},
			want: struct {
				epic    *string
				feature *string
			}{
				epic:    strPtr("E99"),
				feature: strPtr("F99"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epic, feature, err := ParseTaskListArgs(tt.input)

			if err != nil {
				t.Errorf("ParseTaskListArgs(%v) unexpected error: %v", tt.input, err)
				return
			}

			if !ptrEqual(epic, tt.want.epic) {
				t.Errorf("epic = %v, want %v", ptrStr(epic), ptrStr(tt.want.epic))
			}

			if !ptrEqual(feature, tt.want.feature) {
				t.Errorf("feature = %v, want %v", ptrStr(feature), ptrStr(tt.want.feature))
			}
		})
	}

	// Cleanup
	defer database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F99'")
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")
}
```

---

## Phase 1.5: Short Task Key Format (Priority 1.5) ✨ NEW

This enhancement allows users to drop the `T-` prefix from task keys, making them shorter and cleaner:
- `T-E01-F02-001` → `e01-f02-001`
- Minimal implementation cost: ~3 hours
- Fits naturally in Week 2 alongside case normalization

### 1.5.1 Add Short Task Key Pattern

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers.go`

**Changes**:

```go
// Add after existing pattern declarations

// shortTaskKeyPattern matches task keys without the T- prefix (E##-F##-###)
var shortTaskKeyPattern = regexp.MustCompile(`^E\d{2}-F\d{2}-\d{3}$`)
```

### 1.5.2 Add Task Key Normalization Function

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers.go`

**Changes**:

```go
// NormalizeTaskKey converts a task key to canonical format with T- prefix.
// Accepts both full format (T-E##-F##-###) and short format (E##-F##-###).
//
// Examples:
//   T-E01-F02-001 → T-E01-F02-001 (no change)
//   e01-f02-001 → T-E01-F02-001 (add prefix, uppercase)
//   E01-F02-001 → T-E01-F02-001 (add prefix)
//   e01-f02-001-task-name → T-E01-F02-001-TASK-NAME (slugged, add prefix)
func NormalizeTaskKey(input string) (string, error) {
	// First normalize case
	normalized := strings.ToUpper(input)

	// Already has T- prefix
	if strings.HasPrefix(normalized, "T-") {
		return normalized, nil
	}

	// Check if it matches short format (E##-F##-###)
	if shortTaskKeyPattern.MatchString(normalized) {
		return "T-" + normalized, nil
	}

	// Check for slugged short format (E##-F##-###-slug)
	// Split on hyphen and check first 3 parts
	parts := strings.SplitN(normalized, "-", 4)
	if len(parts) >= 3 {
		keyPart := strings.Join(parts[:3], "-")
		if shortTaskKeyPattern.MatchString(keyPart) {
			return "T-" + normalized, nil
		}
	}

	// If none of the above, return as-is and let validation catch it
	return normalized, nil
}
```

### 1.5.3 Update Task Key Validation

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers.go`

**Before**:
```go
func IsTaskKey(s string) bool {
	normalized := NormalizeKey(s)
	return taskKeyPattern.MatchString(normalized)
}
```

**After**:
```go
func IsTaskKey(s string) bool {
	normalized, err := NormalizeTaskKey(s)
	if err != nil {
		return false
	}

	// Check both numeric pattern (T-E##-F##-###) and slugged pattern
	if taskKeyPattern.MatchString(normalized) {
		return true
	}

	// Check slugged pattern (T-E##-F##-###-slug)
	if strings.HasPrefix(normalized, "T-") {
		parts := strings.SplitN(normalized[2:], "-", 4)
		if len(parts) >= 3 {
			keyPart := strings.Join(parts[:3], "-")
			if shortTaskKeyPattern.MatchString(keyPart) {
				return true
			}
		}
	}

	return false
}
```

### 1.5.4 Update Task Commands to Use Normalization

**Files to Update**:
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go`
- All task action commands: `start`, `complete`, `approve`, `block`, `reopen`, `get`

**Pattern to Apply**:

```go
// In runTaskStart, runTaskComplete, runTaskGet, etc.
func runTaskStart(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Normalize task key (adds T- if missing)
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Validate normalized key
	if !IsTaskKey(taskKey) {
		return fmt.Errorf("invalid task key format: %q (expected T-E##-F##-### or E##-F##-###)", args[0])
	}

	// Use normalized key for repository lookup
	// ... rest of function uses taskKey
}
```

### 1.5.5 Add Tests for Short Format

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers_test.go`

```go
func TestNormalizeTaskKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"full format uppercase", "T-E01-F02-001", "T-E01-F02-001", false},
		{"full format lowercase", "t-e01-f02-001", "T-E01-F02-001", false},
		{"short format uppercase", "E01-F02-001", "T-E01-F02-001", false},
		{"short format lowercase", "e01-f02-001", "T-E01-F02-001", false},
		{"short format mixed case", "E01-f02-001", "T-E01-F02-001", false},
		{"slugged full format", "T-E01-F02-001-task-name", "T-E01-F02-001-TASK-NAME", false},
		{"slugged short format", "e01-f02-001-task-name", "T-E01-F02-001-TASK-NAME", false},
		{"invalid format", "E1-F2-1", "E1-F2-1", false}, // Returns normalized but invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeTaskKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeTaskKey(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeTaskKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsTaskKey_ShortFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"full format", "T-E01-F02-001", true},
		{"full format lowercase", "t-e01-f02-001", true},
		{"short format", "E01-F02-001", true},
		{"short format lowercase", "e01-f02-001", true},
		{"short format mixed", "E01-f02-001", true},
		{"slugged full", "T-E01-F02-001-name", true},
		{"slugged short", "e01-f02-001-name", true},
		{"invalid - wrong digits", "E1-F2-1", false},
		{"invalid - missing parts", "E01-F02", false},
		{"invalid - feature key", "E01-F02", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTaskKey(tt.input)
			if got != tt.want {
				t.Errorf("IsTaskKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
```

### 1.5.6 Update Error Messages

**Pattern for Error Messages**:

```go
return fmt.Errorf(`invalid task key format: %q

Expected formats:
  Full:  T-E##-F##-### (e.g., T-E01-F02-001)
  Short: E##-F##-### (e.g., E01-F02-001) - T- prefix optional

All formats are case insensitive.`, input)
```

### 1.5.7 Estimated Effort

**Total Time**: ~3 hours

**Breakdown**:
- Add patterns and normalization function: 30 min
- Update validation functions: 30 min
- Update task commands (6 commands × 10 min): 1 hour
- Write tests: 45 min
- Test manually and fix bugs: 15 min

**Integration Points**:
- Fits naturally in Week 2 (case normalization)
- Uses same `NormalizeKey` infrastructure
- No changes to database schema
- No changes to repository layer
- Purely a CLI parsing enhancement

---

## Phase 2: Positional Arguments for Create Commands (Priority 2)

### 2.1 Update Feature Create Command

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/feature.go`

**Before**:
```go
var featureCreateCmd = &cobra.Command{
	Use:   "create --epic=<key> <title>",
	Short: "Create a new feature",
	Args: cobra.ExactArgs(1), // Only title is positional
	RunE: runFeatureCreate,
}
```

**After**:
```go
var featureCreateCmd = &cobra.Command{
	Use:   "create [EPIC] <title> [flags]",
	Short: "Create a new feature",
	Long: `Create a new feature with auto-assigned key.

Positional Syntax (NEW):
  shark feature create E01 "Feature Title"
  shark feature create e01 "Feature Title"  # Case insensitive

Flag Syntax (backward compatible):
  shark feature create --epic=E01 "Feature Title"

Examples:
  shark feature create E01 "Authentication"
  shark feature create e01 "Auth" --execution-order=1
  shark feature create --epic=E01 "OAuth Login"`,
	Args: cobra.RangeArgs(1, 2), // 1 arg (title) or 2 args (epic + title)
	RunE: runFeatureCreate,
}
```

**Update `runFeatureCreate`**:

```go
func runFeatureCreate(cmd *cobra.Command, args []string) error {
	var epicKey string
	var title string

	// Parse arguments based on count
	if len(args) == 2 {
		// Positional syntax: epic + title
		epicKey = NormalizeKey(args[0])
		title = args[1]

		// Warn if --epic flag also provided
		epicFlag, _ := cmd.Flags().GetString("epic")
		if epicFlag != "" {
			cli.Warning("Both positional epic and --epic flag provided. Using --epic flag value.")
			epicKey = NormalizeKey(epicFlag)
		}
	} else {
		// Single arg - must be title, epic from flag
		title = args[0]
		epicFlag, _ := cmd.Flags().GetString("epic")
		if epicFlag == "" {
			return fmt.Errorf("epic is required: provide as positional argument or --epic flag")
		}
		epicKey = NormalizeKey(epicFlag)
	}

	// Validate epic key format
	if !IsEpicKey(epicKey) {
		return fmt.Errorf("invalid epic key format: %q (expected E##, case insensitive)", epicKey)
	}

	// Rest of existing logic...
	// (use epicKey and title)
}
```

### 2.2 Update Task Create Command

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go`

**Before**:
```go
var taskCreateCmd = &cobra.Command{
	Use:   "create <title> [flags]",
	Short: "Create a new task",
	Args: cobra.ExactArgs(1),
	RunE: runTaskCreate,
}
```

**After**:
```go
var taskCreateCmd = &cobra.Command{
	Use:   "create [EPIC] [FEATURE] <title> [flags]",
	Short: "Create a new task",
	Long: `Create a new task with automatic key generation.

Positional Syntax (NEW):
  shark task create E01 F02 "Task Title"
  shark task create e01 f02 "Task Title"  # Case insensitive

Flag Syntax (backward compatible):
  shark task create --epic=E01 --feature=F02 "Task Title"

Examples:
  shark task create E01 F02 "Implement JWT validation"
  shark task create e01 f02 "JWT" --agent=backend --priority=3
  shark task create --epic=E01 --feature=F02 "JWT validation"`,
	Args: cobra.RangeArgs(1, 3), // 1 (title), 2 (epic+title), or 3 (epic+feature+title)
	RunE: runTaskCreate,
}
```

**Update `runTaskCreate`**:

```go
func runTaskCreate(cmd *cobra.Command, args []string) error {
	var epicKey string
	var featureKey string
	var title string

	// Parse arguments based on count
	switch len(args) {
	case 3:
		// Positional syntax: epic + feature + title
		epicKey = NormalizeKey(args[0])
		featureKey = NormalizeKey(args[1])
		title = args[2]

		// Warn if flags also provided
		epicFlag, _ := cmd.Flags().GetString("epic")
		featureFlag, _ := cmd.Flags().GetString("feature")
		if epicFlag != "" || featureFlag != "" {
			cli.Warning("Both positional and flag syntax provided. Using flag values.")
			if epicFlag != "" {
				epicKey = NormalizeKey(epicFlag)
			}
			if featureFlag != "" {
				featureKey = NormalizeKey(featureFlag)
			}
		}

	case 2:
		// Ambiguous: could be epic+title or feature+title
		// Require flags in this case
		return fmt.Errorf("ambiguous arguments: use 3 args (epic feature title) or 1 arg (title) with flags")

	case 1:
		// Single arg - must be title, epic/feature from flags
		title = args[0]
		epicFlag, _ := cmd.Flags().GetString("epic")
		featureFlag, _ := cmd.Flags().GetString("feature")

		if epicFlag == "" || featureFlag == "" {
			return fmt.Errorf("epic and feature are required: provide as positional arguments or flags")
		}

		epicKey = NormalizeKey(epicFlag)
		featureKey = NormalizeKey(featureFlag)

	default:
		return fmt.Errorf("invalid number of arguments")
	}

	// Validate key formats
	if !IsEpicKey(epicKey) {
		return fmt.Errorf("invalid epic key format: %q (expected E##, case insensitive)", epicKey)
	}

	if !IsFeatureKeySuffix(featureKey) {
		return fmt.Errorf("invalid feature key format: %q (expected F##, case insensitive)", featureKey)
	}

	// Rest of existing logic...
	// (use epicKey, featureKey, and title)
}
```

### 2.3 Add Tests

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task_create_positional_test.go`

```go
package commands

import (
	"testing"
)

func TestParseTaskCreateArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		wantEpic    string
		wantFeature string
		wantTitle   string
		wantErr     bool
	}{
		{
			name:        "positional all args",
			args:        []string{"E01", "F02", "Task Title"},
			flags:       map[string]string{},
			wantEpic:    "E01",
			wantFeature: "F02",
			wantTitle:   "Task Title",
			wantErr:     false,
		},
		{
			name:        "positional lowercase",
			args:        []string{"e01", "f02", "Task Title"},
			flags:       map[string]string{},
			wantEpic:    "E01",
			wantFeature: "F02",
			wantTitle:   "Task Title",
			wantErr:     false,
		},
		{
			name:        "flag syntax only",
			args:        []string{"Task Title"},
			flags:       map[string]string{"epic": "E01", "feature": "F02"},
			wantEpic:    "E01",
			wantFeature: "F02",
			wantTitle:   "Task Title",
			wantErr:     false,
		},
		{
			name:        "mixed positional and flags - flags win",
			args:        []string{"E01", "F02", "Task Title"},
			flags:       map[string]string{"epic": "E99", "feature": "F99"},
			wantEpic:    "E99", // Flags take precedence
			wantFeature: "F99",
			wantTitle:   "Task Title",
			wantErr:     false,
		},
		{
			name:        "two args - ambiguous",
			args:        []string{"E01", "Task Title"},
			flags:       map[string]string{},
			wantErr:     true,
		},
	}

	// Implement parsing logic and test
	// This is a sketch - actual implementation will be in task.go
}
```

---

## Phase 3: Enhanced Error Messages (Priority 3)

### 3.1 Create Error Template System

**File**: Create `/home/jwwelbor/projects/shark-task-manager/internal/cli/errors.go`

```go
package cli

import (
	"fmt"
	"strings"
)

// KeyFormatError creates a user-friendly error for invalid key formats
func KeyFormatError(key string, expectedFormat string, examples ...string) error {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("Invalid key format: %q\n", key))
	msg.WriteString(fmt.Sprintf("  Expected: %s\n", expectedFormat))

	if len(examples) > 0 {
		msg.WriteString("  Examples: ")
		msg.WriteString(strings.Join(examples, ", "))
		msg.WriteString("\n")
	}

	msg.WriteString("  Note: Case insensitive (e01, E01, and E01 are equivalent)")

	return fmt.Errorf(msg.String())
}

// EpicKeyError creates an error for invalid epic keys
func EpicKeyError(key string) error {
	return KeyFormatError(key, "E## (two-digit epic number)", "E01", "E04", "E99")
}

// FeatureKeyError creates an error for invalid feature keys
func FeatureKeyError(key string) error {
	return KeyFormatError(key, "E##-F## or F##", "E01-F02", "F02")
}

// TaskKeyError creates an error for invalid task keys
func TaskKeyError(key string) error {
	return KeyFormatError(key, "T-E##-F##-###", "T-E01-F02-001")
}
```

### 3.2 Update Error Messages in Helpers

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/helpers.go`

**Before**:
```go
return nil, fmt.Errorf("invalid epic key format: %q (expected E##, e.g., E04)", epicKey)
```

**After**:
```go
return nil, cli.EpicKeyError(epicKey)
```

---

## Phase 4: Documentation Updates (Priority 4)

### 4.1 Update CLI Reference

**File**: `/home/jwwelbor/projects/shark-task-manager/docs/CLI_REFERENCE.md`

Add sections showing new syntax patterns with case-insensitive examples.

### 4.2 Update CLAUDE.md

**File**: `/home/jwwelbor/projects/shark-task-manager/CLAUDE.md`

Update command examples to show both positional and flag syntax.

---

## Testing Checklist

### Unit Tests
- [ ] `TestNormalizeKey` - Case normalization
- [ ] `TestIsEpicKey_CaseInsensitive` - Epic validation
- [ ] `TestIsFeatureKey_CaseInsensitive` - Feature validation
- [ ] `TestParseFeatureKey_CaseInsensitive` - Feature parsing
- [ ] `TestParseTaskListArgs_CaseInsensitive` - Task list parsing
- [ ] `TestParseFeatureCreateArgs` - Feature create parsing
- [ ] `TestParseTaskCreateArgs` - Task create parsing

### Integration Tests
- [ ] Create epic with lowercase key
- [ ] Create feature with positional syntax
- [ ] Create task with positional syntax
- [ ] List tasks with lowercase epic key
- [ ] Get task with mixed case key
- [ ] Flag precedence when both positional and flags provided

### Manual Testing
```bash
# Case insensitivity
shark epic get e01
shark feature list e01
shark task list e01 f02

# Positional create
shark feature create e01 "New Feature"
shark task create e01 f02 "New Task"

# Mixed syntax
shark task list e01 --status=todo
```

---

## Rollout Plan

### Week 1: Case Insensitivity
- Implement `NormalizeKey()` function
- Update validation functions
- Update parsing functions
- Write and run all tests
- Merge to main

### Week 2: Positional Arguments
- Update `feature create` command
- Update `task create` command
- Add parsing logic with flag precedence
- Write and run tests
- Merge to main

### Week 3: Enhanced Errors
- Create error template system
- Update all error messages
- Test error messages
- Merge to main

### Week 4: Documentation
- Update CLI_REFERENCE.md
- Update CLAUDE.md
- Update README.md
- Create migration guide (if needed)

---

## Backward Compatibility Verification

Before each merge, verify:

1. **Existing flag syntax still works**:
   ```bash
   shark feature create --epic=E01 "Feature"
   shark task create --epic=E01 --feature=F02 "Task"
   ```

2. **Uppercase keys still work**:
   ```bash
   shark epic get E01
   shark feature list E01
   shark task list E01 F02
   ```

3. **JSON output unchanged**:
   ```bash
   shark epic get E01 --json | jq '.key'  # Should be "E01"
   shark task list --json | jq '.[0].key'  # Should be uppercase
   ```

4. **Exit codes unchanged**:
   ```bash
   shark epic get INVALID
   echo $?  # Should be 1 (not found)
   ```

---

## Success Criteria

- [ ] All existing commands continue to work without modification
- [ ] Case-insensitive keys work for all commands
- [ ] Positional syntax works for create commands
- [ ] Error messages are helpful and include examples
- [ ] All tests pass (unit + integration)
- [ ] Documentation is updated
- [ ] No breaking changes to API or output format
- [ ] AI agents can use simpler command templates
- [ ] New users succeed on first try (validated with test users)

---

## Code Review Checklist

- [ ] All functions have clear documentation
- [ ] Error messages are user-friendly
- [ ] Tests cover edge cases (empty input, invalid format, etc.)
- [ ] No performance regression (normalization is O(1))
- [ ] Backward compatibility maintained
- [ ] JSON output format unchanged
- [ ] Verbose mode shows normalization (for debugging)
