package commands

import (
	"testing"
)

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

		// Invalid epic keys - wrong case
		{"Invalid lowercase e04", "e04", false},
		{"Invalid mixed Ee04", "Ee04", false},

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

		// Invalid - wrong epic format
		{"Invalid e04-F01", "e04-F01", false},
		{"Invalid E4-F01", "E4-F01", false},
		{"Invalid E001-F01", "E001-F01", false},
		{"Invalid E-F01", "E-F01", false},

		// Invalid - wrong feature format
		{"Invalid E04-f01", "E04-f01", false},
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

		// Invalid - wrong case
		{"Invalid lowercase f01", "f01", false},
		{"Invalid mixed Ff01", "Ff01", false},

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

		// Invalid - missing hyphen
		{"Invalid E04F01", "E04F01", "", "", true},

		// Invalid - wrong epic format
		{"Invalid e04-F01", "e04-F01", "", "", true},
		{"Invalid E4-F01", "E4-F01", "", "", true},

		// Invalid - wrong feature format
		{"Invalid E04-f01", "E04-f01", "", "", true},
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
		{"Invalid e04", "e04", false},
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

		// Invalid formats
		{"Invalid E1", []string{"E1"}, nil, true},
		{"Invalid e04", []string{"e04"}, nil, true},
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

		// Single combined feature key
		{"Combined E04-F01", []string{"E04-F01"}, strPtr("E04"), strPtr("F01"), false},
		{"Combined E01-F99", []string{"E01-F99"}, strPtr("E01"), strPtr("F99"), false},

		// Two args - epic and feature suffix
		{"Two args E04 F01", []string{"E04", "F01"}, strPtr("E04"), strPtr("F01"), false},
		{"Two args E01 F99", []string{"E01", "F99"}, strPtr("E01"), strPtr("F99"), false},

		// Two args - epic and full feature key
		{"Two args E04 E04-F01", []string{"E04", "E04-F01"}, strPtr("E04"), strPtr("F01"), false},
		{"Two args E01 E01-F99", []string{"E01", "E01-F99"}, strPtr("E01"), strPtr("F99"), false},

		// Invalid formats
		{"Invalid E1", []string{"E1"}, nil, nil, true},
		{"Invalid e04", []string{"e04"}, nil, nil, true},
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
