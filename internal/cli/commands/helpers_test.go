package commands

import (
	"testing"
)

// TestNormalizeKey tests the NormalizeKey function for case insensitivity
func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Epic keys
		{"Uppercase epic E04", "E04", "E04"},
		{"Lowercase epic e04", "e04", "E04"},

		// Feature keys
		{"Uppercase feature E04-F01", "E04-F01", "E04-F01"},
		{"Lowercase feature e04-f01", "e04-f01", "E04-F01"},
		{"Mixed case feature E04-f01", "E04-f01", "E04-F01"},
		{"Mixed case feature e04-F01", "e04-F01", "E04-F01"},

		// Task keys
		{"Uppercase task T-E04-F01-001", "T-E04-F01-001", "T-E04-F01-001"},
		{"Lowercase task t-e04-f01-001", "t-e04-f01-001", "T-E04-F01-001"},
		{"Mixed case task T-e04-F01-001", "T-e04-F01-001", "T-E04-F01-001"},

		// Feature suffix
		{"Uppercase F01", "F01", "F01"},
		{"Lowercase f01", "f01", "F01"},

		// Empty and special
		{"Empty string", "", ""},
		{"Numbers only", "123", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeKey(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsEpicKey tests the IsEpicKey pattern matching function
func TestIsEpicKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid epic keys
		{"Valid E01", "E01", true},
		{"Valid E04", "E04", true},
		{"Valid E99", "E99", true},
		{"Valid E00", "E00", true},
		{"Valid E50", "E50", true},

		// Invalid epic keys - wrong format
		{"Invalid E1", "E1", false},
		{"Invalid E001", "E001", false},
		{"Invalid E", "E", false},
		{"Invalid E0", "E0", false},
		{"Invalid E0001", "E0001", false},

		// Valid epic keys - case insensitive (NEW: should now be valid)
		{"Valid lowercase e04", "e04", true},
		{"Valid lowercase e01", "e01", true},

		// Invalid epic keys - extra characters
		{"Invalid E04-", "E04-", false},
		{"Invalid E04X", "E04X", false},
		{"Invalid E04 ", "E04 ", false},

		// Invalid epic keys - wrong prefix
		{"Invalid F04", "F04", false},
		{"Invalid T04", "T04", false},
		{"Invalid 04", "04", false},

		// Empty and special cases
		{"Empty string", "", false},
		{"Only digits", "0404", false},
		{"Special chars", "E#$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEpicKey(tt.input)
			if result != tt.expected {
				t.Errorf("IsEpicKey(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseFeatureCreateArgs tests parsing positional arguments for feature create command
func TestParseFeatureCreateArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantEpic    *string
		wantTitle   *string
		wantErr     bool
		errContains string
	}{
		// Valid cases with 2 arguments (EPIC TITLE)
		{
			name:      "Valid: E04 'Feature Title'",
			args:      []string{"E04", "Feature Title"},
			wantEpic:  strPtr("E04"),
			wantTitle: strPtr("Feature Title"),
			wantErr:   false,
		},
		{
			name:      "Valid: e07 'lowercase epic'",
			args:      []string{"e07", "Test Feature"},
			wantEpic:  strPtr("E07"),
			wantTitle: strPtr("Test Feature"),
			wantErr:   false,
		},
		{
			name:      "Valid: E01 'Single Word'",
			args:      []string{"E01", "Authentication"},
			wantEpic:  strPtr("E01"),
			wantTitle: strPtr("Authentication"),
			wantErr:   false,
		},

		// Invalid cases - wrong number of arguments
		{
			name:        "Invalid: no arguments",
			args:        []string{},
			wantEpic:    nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "missing required arguments",
		},
		{
			name:        "Invalid: only 1 argument",
			args:        []string{"E04"},
			wantEpic:    nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "missing required arguments",
		},
		{
			name:        "Invalid: too many arguments",
			args:        []string{"E04", "Title", "Extra"},
			wantEpic:    nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "too many arguments",
		},

		// Invalid epic key format
		{
			name:        "Invalid: bad epic format",
			args:        []string{"E1", "Title"},
			wantEpic:    nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "invalid epic key format",
		},
		{
			name:        "Invalid: not an epic key",
			args:        []string{"F04", "Title"},
			wantEpic:    nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "invalid epic key format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epic, title, err := ParseFeatureCreateArgs(tt.args)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFeatureCreateArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check error message contains expected text
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ParseFeatureCreateArgs() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			// Check epic
			if !equalStringPtrs(epic, tt.wantEpic) {
				t.Errorf("ParseFeatureCreateArgs() epic = %v, want %v", ptrToString(epic), ptrToString(tt.wantEpic))
			}

			// Check title
			if !equalStringPtrs(title, tt.wantTitle) {
				t.Errorf("ParseFeatureCreateArgs() title = %v, want %v", ptrToString(title), ptrToString(tt.wantTitle))
			}
		})
	}
}

// TestIsFeatureKey tests the IsFeatureKey pattern matching function
func TestIsFeatureKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid feature keys
		{"Valid E04-F01", "E04-F01", true},
		{"Valid E01-F01", "E01-F01", true},
		{"Valid E99-F99", "E99-F99", true},
		{"Valid E00-F00", "E00-F00", true},
		{"Valid E50-F50", "E50-F50", true},

		// Invalid - missing hyphen
		{"Invalid E04F01", "E04F01", false},
		{"Invalid E04 F01", "E04 F01", false},

		// Valid - case insensitive (NEW: should now be valid)
		{"Valid lowercase e04-f01", "e04-f01", true},
		{"Valid mixed E04-f01", "E04-f01", true},
		{"Valid mixed e04-F01", "e04-F01", true},

		// Invalid - wrong epic format
		{"Invalid E4-F01", "E4-F01", false},
		{"Invalid E001-F01", "E001-F01", false},
		{"Invalid E-F01", "E-F01", false},

		// Invalid - wrong feature format
		{"Invalid E04-F1", "E04-F1", false},
		{"Invalid E04-F001", "E04-F001", false},
		{"Invalid E04-F", "E04-F", false},

		// Invalid - only epic part
		{"Invalid E04", "E04", false},

		// Invalid - only feature part
		{"Invalid F01", "F01", false},

		// Invalid - extra characters
		{"Invalid E04-F01-", "E04-F01-", false},
		{"Invalid E04-F01X", "E04-F01X", false},
		{"Invalid xE04-F01", "xE04-F01", false},

		// Empty and special cases
		{"Empty string", "", false},
		{"Just hyphen", "-", false},
		{"Special chars", "E##-F##", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFeatureKey(tt.input)
			if result != tt.expected {
				t.Errorf("IsFeatureKey(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsFeatureKeySuffix tests the IsFeatureKeySuffix pattern matching function
func TestIsFeatureKeySuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid feature key suffixes
		{"Valid F01", "F01", true},
		{"Valid F99", "F99", true},
		{"Valid F00", "F00", true},
		{"Valid F50", "F50", true},

		// Invalid - wrong format
		{"Invalid F1", "F1", false},
		{"Invalid F001", "F001", false},
		{"Invalid F", "F", false},
		{"Invalid F0", "F0", false},

		// Valid - case insensitive (NEW: should now be valid)
		{"Valid lowercase f01", "f01", true},

		// Invalid - extra characters
		{"Invalid F01-", "F01-", false},
		{"Invalid F01X", "F01X", false},
		{"Invalid F01 ", "F01 ", false},

		// Invalid - wrong prefix
		{"Invalid E01", "E01", false},
		{"Invalid T01", "T01", false},
		{"Invalid 01", "01", false},

		// Empty and special cases
		{"Empty string", "", false},
		{"Only digits", "0101", false},
		{"Special chars", "F#$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFeatureKeySuffix(tt.input)
			if result != tt.expected {
				t.Errorf("IsFeatureKeySuffix(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseFeatureKey tests the ParseFeatureKey parsing function
func TestParseFeatureKey(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantEpic    string
		wantFeature string
		wantErr     bool
	}{
		// Valid feature keys
		{"Valid E04-F01", "E04-F01", "E04", "F01", false},
		{"Valid E01-F99", "E01-F99", "E01", "F99", false},
		{"Valid E99-F00", "E99-F00", "E99", "F00", false},

		// Valid - case insensitive (NEW: normalized to uppercase)
		{"Valid lowercase e04-f01", "e04-f01", "E04", "F01", false},
		{"Valid mixed e04-F01", "e04-F01", "E04", "F01", false},
		{"Valid mixed E04-f01", "E04-f01", "E04", "F01", false},

		// Invalid - missing hyphen
		{"Invalid E04F01", "E04F01", "", "", true},

		// Invalid - wrong epic format
		{"Invalid E4-F01", "E4-F01", "", "", true},

		// Invalid - wrong feature format
		{"Invalid E04-F1", "E04-F1", "", "", true},

		// Invalid - only epic part
		{"Invalid E04", "E04", "", "", true},

		// Invalid - only feature part
		{"Invalid F01", "F01", "", "", true},

		// Empty and special cases
		{"Empty string", "", "", "", true},
		{"Special chars", "E##-F##", "", "", true},
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

// TestIsValidEpicKey tests backward compatibility with deprecated function
func TestIsValidEpicKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid E01", "E01", true},
		{"Valid E04", "E04", true},
		{"Invalid E1", "E1", false},
		{"Valid e04 (case insensitive)", "e04", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEpicKey(tt.input)
			if result != tt.expected {
				t.Errorf("isValidEpicKey(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// BenchmarkIsEpicKey benchmarks the IsEpicKey function
func BenchmarkIsEpicKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsEpicKey("E04")
	}
}

// BenchmarkIsFeatureKey benchmarks the IsFeatureKey function
func BenchmarkIsFeatureKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsFeatureKey("E04-F01")
	}
}

// BenchmarkIsFeatureKeySuffix benchmarks the IsFeatureKeySuffix function
func BenchmarkIsFeatureKeySuffix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsFeatureKeySuffix("F01")
	}
}

// BenchmarkParseFeatureKey benchmarks the ParseFeatureKey function
func BenchmarkParseFeatureKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseFeatureKey("E04-F01")
	}
}

// TestParseFeatureListArgs tests the ParseFeatureListArgs parsing function
func TestParseFeatureListArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantKey *string
		wantErr bool
	}{
		// No args - list all
		{"No args", []string{}, nil, false},

		// Valid single epic arg
		{"Valid epic E04", []string{"E04"}, strPtr("E04"), false},
		{"Valid epic E01", []string{"E01"}, strPtr("E01"), false},
		{"Valid epic E99", []string{"E99"}, strPtr("E99"), false},

		// Valid - case insensitive (NEW: normalized to uppercase)
		{"Valid lowercase e04", []string{"e04"}, strPtr("E04"), false},
		{"Valid lowercase e01", []string{"e01"}, strPtr("E01"), false},

		// Invalid formats
		{"Invalid E1", []string{"E1"}, nil, true},
		{"Invalid feature key", []string{"E04-F01"}, nil, true},
		{"Invalid F01", []string{"F01"}, nil, true},

		// Too many args
		{"Two args", []string{"E04", "F01"}, nil, true},
		{"Three args", []string{"E04", "F01", "extra"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, err := ParseFeatureListArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFeatureListArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if (gotKey == nil) != (tt.wantKey == nil) {
				t.Errorf("ParseFeatureListArgs(%v) = %v, want %v", tt.args, gotKey, tt.wantKey)
				return
			}
			if gotKey != nil && tt.wantKey != nil && *gotKey != *tt.wantKey {
				t.Errorf("ParseFeatureListArgs(%v) = %q, want %q", tt.args, *gotKey, *tt.wantKey)
			}
		})
	}
}

// TestParseTaskListArgs tests the ParseTaskListArgs parsing function
func TestParseTaskListArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantEpic    *string
		wantFeature *string
		wantErr     bool
	}{
		// No args - list all
		{"No args", []string{}, nil, nil, false},

		// Single epic arg
		{"Single epic E04", []string{"E04"}, strPtr("E04"), nil, false},
		{"Single epic E01", []string{"E01"}, strPtr("E01"), nil, false},

		// Single epic arg - case insensitive (NEW: normalized to uppercase)
		{"Single epic lowercase e04", []string{"e04"}, strPtr("E04"), nil, false},
		{"Single epic lowercase e01", []string{"e01"}, strPtr("E01"), nil, false},

		// Single combined feature key
		{"Combined E04-F01", []string{"E04-F01"}, strPtr("E04"), strPtr("F01"), false},
		{"Combined E01-F99", []string{"E01-F99"}, strPtr("E01"), strPtr("F99"), false},

		// Single combined feature key - case insensitive (NEW: normalized)
		{"Combined lowercase e04-f01", []string{"e04-f01"}, strPtr("E04"), strPtr("F01"), false},
		{"Combined mixed e04-F01", []string{"e04-F01"}, strPtr("E04"), strPtr("F01"), false},

		// Two args - epic and feature suffix
		{"Two args E04 F01", []string{"E04", "F01"}, strPtr("E04"), strPtr("F01"), false},
		{"Two args E01 F99", []string{"E01", "F99"}, strPtr("E01"), strPtr("F99"), false},

		// Two args - case insensitive (NEW: normalized)
		{"Two args lowercase e04 f01", []string{"e04", "f01"}, strPtr("E04"), strPtr("F01"), false},
		{"Two args mixed e04 F01", []string{"e04", "F01"}, strPtr("E04"), strPtr("F01"), false},

		// Two args - epic and full feature key
		{"Two args E04 E04-F01", []string{"E04", "E04-F01"}, strPtr("E04"), strPtr("F01"), false},
		{"Two args E01 E01-F99", []string{"E01", "E01-F99"}, strPtr("E01"), strPtr("F99"), false},

		// Two args - epic and full feature key - case insensitive (NEW)
		{"Two args lowercase e04 e04-f01", []string{"e04", "e04-f01"}, strPtr("E04"), strPtr("F01"), false},

		// Invalid formats
		{"Invalid E1", []string{"E1"}, nil, nil, true},
		{"Invalid F01 only", []string{"F01"}, nil, nil, true},
		{"Invalid two E04 F", []string{"E04", "F"}, nil, nil, true},
		{"Invalid two E E04-F01", []string{"E", "E04-F01"}, nil, nil, true},

		// Too many args
		{"Three args", []string{"E04", "F01", "extra"}, nil, nil, true},
		{"Four args", []string{"E04", "F01", "extra", "more"}, nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEpic, gotFeature, err := ParseTaskListArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskListArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if (gotEpic == nil) != (tt.wantEpic == nil) {
				t.Errorf("ParseTaskListArgs(%v) epic = %v, want %v", tt.args, gotEpic, tt.wantEpic)
				return
			}
			if gotEpic != nil && tt.wantEpic != nil && *gotEpic != *tt.wantEpic {
				t.Errorf("ParseTaskListArgs(%v) epic = %q, want %q", tt.args, *gotEpic, *tt.wantEpic)
			}
			if (gotFeature == nil) != (tt.wantFeature == nil) {
				t.Errorf("ParseTaskListArgs(%v) feature = %v, want %v", tt.args, gotFeature, tt.wantFeature)
				return
			}
			if gotFeature != nil && tt.wantFeature != nil && *gotFeature != *tt.wantFeature {
				t.Errorf("ParseTaskListArgs(%v) feature = %q, want %q", tt.args, *gotFeature, *tt.wantFeature)
			}
		})
	}
}

// Helper function to create string pointers for test comparisons
func strPtr(s string) *string {
	return &s
}

// TestNormalizeTaskKey tests the NormalizeTaskKey function for short task key format support
// This is part of T-E07-F20-006: Add short task key pattern and normalization function
func TestNormalizeTaskKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Full format uppercase - no change needed
		{"Full format uppercase", "T-E01-F02-001", "T-E01-F02-001", false},
		{"Full format with different epic", "T-E04-F01-005", "T-E04-F01-005", false},

		// Full format lowercase - should uppercase
		{"Full format lowercase", "t-e01-f02-001", "T-E01-F02-001", false},
		{"Full format mixed case", "T-e01-F02-001", "T-E01-F02-001", false},

		// Short format uppercase - should add T- prefix
		{"Short format uppercase", "E01-F02-001", "T-E01-F02-001", false},
		{"Short format different epic", "E04-F01-005", "T-E04-F01-005", false},

		// Short format lowercase - should add T- prefix and uppercase
		{"Short format lowercase", "e01-f02-001", "T-E01-F02-001", false},
		{"Short format mixed case", "E01-f02-001", "T-E01-F02-001", false},

		// Slugged full format - should preserve slug and uppercase
		{"Slugged full format", "T-E01-F02-001-task-name", "T-E01-F02-001-TASK-NAME", false},
		{"Slugged full lowercase", "t-e01-f02-001-task-name", "T-E01-F02-001-TASK-NAME", false},

		// Slugged short format - should add T- prefix, preserve slug, and uppercase
		{"Slugged short format", "e01-f02-001-task-name", "T-E01-F02-001-TASK-NAME", false},
		{"Slugged short uppercase", "E01-F02-001-IMPLEMENT-AUTH", "T-E01-F02-001-IMPLEMENT-AUTH", false},
		{"Slugged short mixed", "E01-f02-001-Task-Name", "T-E01-F02-001-TASK-NAME", false},

		// Edge cases - should return errors
		{"Invalid format", "INVALID", "", true},
		{"Empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeTaskKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeTaskKey(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("NormalizeTaskKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNormalizeTaskKeyBackwardCompatibility ensures full task keys still work
// This is part of T-E07-F20-008: Integration tests for short task key format
func TestNormalizeTaskKeyBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Existing full format keys should still work
		{"Full key T-E07-F20-001", "T-E07-F20-001", "T-E07-F20-001"},
		{"Full key lowercase", "t-e07-f20-001", "T-E07-F20-001"},
		{"Full key with slug", "T-E07-F20-001-task-name", "T-E07-F20-001-TASK-NAME"},

		// Short format keys are now supported
		{"Short key E07-F20-001", "E07-F20-001", "T-E07-F20-001"},
		{"Short key lowercase", "e07-f20-001", "T-E07-F20-001"},
		{"Short key with slug", "E07-F20-001-task-name", "T-E07-F20-001-TASK-NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeTaskKey(tt.input)
			if err != nil {
				t.Errorf("NormalizeTaskKey(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("NormalizeTaskKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsShortTaskKey tests if a string matches the short task key pattern (E##-F##-###)
// This is part of T-E07-F20-006: Add short task key pattern and normalization function
func TestIsShortTaskKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid short task keys
		{"Valid E01-F02-001", "E01-F02-001", true},
		{"Valid E04-F01-005", "E04-F01-005", true},
		{"Valid E99-F99-999", "E99-F99-999", true},
		{"Valid E00-F00-000", "E00-F00-000", true},

		// Invalid - has T- prefix (this is full format, not short)
		{"Invalid with T- prefix", "T-E01-F02-001", false},
		{"Invalid with t- prefix", "t-e01-f02-001", false},

		// Invalid - wrong number of digits
		{"Invalid E1-F02-001", "E1-F02-001", false},
		{"Invalid E01-F2-001", "E01-F2-001", false},
		{"Invalid E01-F02-01", "E01-F02-01", false},
		{"Invalid E01-F02-1", "E01-F02-1", false},

		// Invalid - missing parts
		{"Invalid E01-F02", "E01-F02", false},
		{"Invalid F02-001", "F02-001", false},
		{"Invalid 001", "001", false},

		// Invalid - extra characters
		{"Invalid with suffix", "E01-F02-001-", false},
		{"Invalid with slug", "E01-F02-001-task-name", false},

		// Empty and special cases
		{"Empty string", "", false},
		{"Invalid special chars", "E##-F##-###", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isShortTaskKey(tt.input)
			if result != tt.expected {
				t.Errorf("isShortTaskKey(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseTaskCreateArgs tests parsing positional arguments for task create command
func TestParseTaskCreateArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantEpic    *string
		wantFeature *string
		wantTitle   *string
		wantErr     bool
		errContains string
	}{
		// Valid cases with 3 arguments (EPIC FEATURE TITLE)
		{
			name:        "Valid: E07 F20 'Task Title'",
			args:        []string{"E07", "F20", "Task Title"},
			wantEpic:    strPtr("E07"),
			wantFeature: strPtr("F20"),
			wantTitle:   strPtr("Task Title"),
			wantErr:     false,
		},
		{
			name:        "Valid: E07 E07-F20 'Task Title' (redundant epic in feature)",
			args:        []string{"E07", "E07-F20", "Task Title"},
			wantEpic:    strPtr("E07"),
			wantFeature: strPtr("F20"),
			wantTitle:   strPtr("Task Title"),
			wantErr:     false,
		},
		{
			name:        "Valid: e04 f01 'lowercase keys'",
			args:        []string{"e04", "f01", "Test Task"},
			wantEpic:    strPtr("E04"),
			wantFeature: strPtr("F01"),
			wantTitle:   strPtr("Test Task"),
			wantErr:     false,
		},

		// Valid cases with 2 arguments (EPIC-FEATURE TITLE)
		{
			name:        "Valid: E07-F20 'Task Title'",
			args:        []string{"E07-F20", "Task Title"},
			wantEpic:    strPtr("E07"),
			wantFeature: strPtr("F20"),
			wantTitle:   strPtr("Task Title"),
			wantErr:     false,
		},
		{
			name:        "Valid: e07-f20 'lowercase combined'",
			args:        []string{"e07-f20", "Test Task"},
			wantEpic:    strPtr("E07"),
			wantFeature: strPtr("F20"),
			wantTitle:   strPtr("Test Task"),
			wantErr:     false,
		},

		// Invalid cases - wrong number of arguments
		{
			name:        "Invalid: no arguments",
			args:        []string{},
			wantEpic:    nil,
			wantFeature: nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "missing required arguments",
		},
		{
			name:        "Invalid: only 1 argument",
			args:        []string{"E07"},
			wantEpic:    nil,
			wantFeature: nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "missing required arguments",
		},
		{
			name:        "Invalid: too many arguments",
			args:        []string{"E07", "F20", "Title", "Extra"},
			wantEpic:    nil,
			wantFeature: nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "too many arguments",
		},

		// Invalid epic key format
		{
			name:        "Invalid: bad epic format (3 args)",
			args:        []string{"E1", "F20", "Title"},
			wantEpic:    nil,
			wantFeature: nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "invalid epic key format",
		},
		{
			name:        "Invalid: bad epic in combined key",
			args:        []string{"E1-F20", "Title"},
			wantEpic:    nil,
			wantFeature: nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "invalid",
		},

		// Invalid feature key format
		{
			name:        "Invalid: bad feature format",
			args:        []string{"E07", "F2", "Title"},
			wantEpic:    nil,
			wantFeature: nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "invalid feature key format",
		},
		{
			name:        "Invalid: not a feature key",
			args:        []string{"E07", "E08", "Title"},
			wantEpic:    nil,
			wantFeature: nil,
			wantTitle:   nil,
			wantErr:     true,
			errContains: "invalid feature key format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epic, feature, title, err := ParseTaskCreateArgs(tt.args)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskCreateArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check error message contains expected text
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ParseTaskCreateArgs() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			// Check epic
			if !equalStringPtrs(epic, tt.wantEpic) {
				t.Errorf("ParseTaskCreateArgs() epic = %v, want %v", ptrToString(epic), ptrToString(tt.wantEpic))
			}

			// Check feature
			if !equalStringPtrs(feature, tt.wantFeature) {
				t.Errorf("ParseTaskCreateArgs() feature = %v, want %v", ptrToString(feature), ptrToString(tt.wantFeature))
			}

			// Check title
			if !equalStringPtrs(title, tt.wantTitle) {
				t.Errorf("ParseTaskCreateArgs() title = %v, want %v", ptrToString(title), ptrToString(tt.wantTitle))
			}
		})
	}
}

// Helper functions for tests (ParseFeatureCreateArgs-specific)
func ptrToString(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func equalStringPtrs(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
