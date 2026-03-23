package repoutil

import (
	"testing"
)

func TestContainsHyphen(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", false},
		{"no hyphen", "E04", false},
		{"single hyphen", "E04-name", true},
		{"multiple hyphens", "E04-epic-name", true},
		{"hyphen only", "-", true},
		{"leading hyphen", "-name", true},
		{"trailing hyphen", "name-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsHyphen(tt.input)
			if got != tt.want {
				t.Errorf("ContainsHyphen(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", false},
		{"single digit", "5", true},
		{"multiple digits", "12345", true},
		{"leading zero", "007", true},
		{"with letter", "12a3", false},
		{"all letters", "abc", false},
		{"with hyphen", "12-3", false},
		{"with space", "12 3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNumeric(tt.input)
			if got != tt.want {
				t.Errorf("IsNumeric(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitAtFirstHyphen(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPrefix string
		wantSuffix string
		wantOk     bool
	}{
		{"no hyphen", "E04", "E04", "", false},
		{"single hyphen", "E04-name", "E04", "name", true},
		{"multiple hyphens", "E04-epic-name", "E04", "epic-name", true},
		{"empty prefix", "-name", "", "name", true},
		{"empty suffix", "name-", "name", "", true},
		{"only hyphen", "-", "", "", true},
		{"empty string", "", "", "", false},
		// TC-F001-1 edge cases
		{"feature key with hyphens", "E07-F01", "E07", "F01", true},
		{"task key", "T-E07-F01-001", "T", "E07-F01-001", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, suffix, ok := SplitAtFirstHyphen(tt.input)
			if prefix != tt.wantPrefix || suffix != tt.wantSuffix || ok != tt.wantOk {
				t.Errorf("SplitAtFirstHyphen(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.input, prefix, suffix, ok, tt.wantPrefix, tt.wantSuffix, tt.wantOk)
			}
		})
	}
}

func TestSplitAtNthHyphen(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		n          int
		wantPrefix string
		wantSuffix string
		wantOk     bool
	}{
		// TC-F001-1 documented cases
		{"4th hyphen in task key with slug", "T-E07-F01-001-slug-text", 4, "T-E07-F01-001", "slug-text", true},
		{"4th hyphen in task key without slug", "T-E07-F01-001", 4, "T-E07-F01-001", "", false},
		{"1st hyphen", "A-B-C", 1, "A", "B-C", true},
		{"2nd hyphen", "A-B-C", 2, "A-B", "C", true},
		{"3rd hyphen not found", "A-B-C", 3, "A-B-C", "", false},
		{"no hyphens", "ABC", 1, "ABC", "", false},
		{"empty string", "", 1, "", "", false},
		{"n=0 never matches", "A-B", 0, "A-B", "", false},
		{"hyphen at end with empty suffix", "T-E07-F01-001-", 4, "T-E07-F01-001", "", true},
		{"multi-word slug", "T-E07-F01-001-implement-jwt-token", 4, "T-E07-F01-001", "implement-jwt-token", true},
		// TC-F001-1: documented test case
		{"n larger than available hyphens", "T-E07-F01", 5, "T-E07-F01", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, suffix, ok := SplitAtNthHyphen(tt.input, tt.n)
			if prefix != tt.wantPrefix || suffix != tt.wantSuffix || ok != tt.wantOk {
				t.Errorf("SplitAtNthHyphen(%q, %d) = (%q, %q, %v), want (%q, %q, %v)",
					tt.input, tt.n, prefix, suffix, ok, tt.wantPrefix, tt.wantSuffix, tt.wantOk)
			}
		})
	}
}
