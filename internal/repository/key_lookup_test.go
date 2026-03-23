package repository

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
)

func TestSharedContainsHyphen(t *testing.T) {
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
			got := repoutil.ContainsHyphen(tt.input)
			if got != tt.want {
				t.Errorf("repoutil.ContainsHyphen(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSharedIsNumeric(t *testing.T) {
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
			got := repoutil.IsNumeric(tt.input)
			if got != tt.want {
				t.Errorf("repoutil.IsNumeric(%q) = %v, want %v", tt.input, got, tt.want)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, suffix, ok := repoutil.SplitAtFirstHyphen(tt.input)
			if prefix != tt.wantPrefix || suffix != tt.wantSuffix || ok != tt.wantOk {
				t.Errorf("repoutil.SplitAtFirstHyphen(%q) = (%q, %q, %v), want (%q, %q, %v)",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, suffix, ok := repoutil.SplitAtNthHyphen(tt.input, tt.n)
			if prefix != tt.wantPrefix || suffix != tt.wantSuffix || ok != tt.wantOk {
				t.Errorf("repoutil.SplitAtNthHyphen(%q, %d) = (%q, %q, %v), want (%q, %q, %v)",
					tt.input, tt.n, prefix, suffix, ok, tt.wantPrefix, tt.wantSuffix, tt.wantOk)
			}
		})
	}
}

// TestWrapperConsistency verifies that the backward-compatible package-private wrappers
// produce the same results as the repoutil exported functions.
func TestWrapperConsistency(t *testing.T) {
	t.Run("containsHyphen matches repoutil.ContainsHyphen", func(t *testing.T) {
		cases := []string{"", "E04", "E04-name", "-", "a-b-c"}
		for _, s := range cases {
			if containsHyphen(s) != repoutil.ContainsHyphen(s) {
				t.Errorf("containsHyphen(%q) != repoutil.ContainsHyphen(%q)", s, s)
			}
		}
	})

	t.Run("isNumeric matches repoutil.IsNumeric", func(t *testing.T) {
		cases := []string{"", "123", "abc", "12a", "007"}
		for _, s := range cases {
			if isNumeric(s) != repoutil.IsNumeric(s) {
				t.Errorf("isNumeric(%q) != repoutil.IsNumeric(%q)", s, s)
			}
		}
	})

	t.Run("splitSluggedKey matches repoutil.SplitAtFirstHyphen", func(t *testing.T) {
		cases := []string{"E04", "E04-epic-name", "", "E04-epic-name-test"}
		for _, s := range cases {
			parts := splitSluggedKey(s)
			prefix, suffix, ok := repoutil.SplitAtFirstHyphen(s)

			if !ok {
				if len(parts) != 1 || parts[0] != prefix {
					t.Errorf("splitSluggedKey(%q) = %v, but repoutil.SplitAtFirstHyphen returned (%q, %q, false)", s, parts, prefix, suffix)
				}
			} else {
				if len(parts) != 2 || parts[0] != prefix || parts[1] != suffix {
					t.Errorf("splitSluggedKey(%q) = %v, but repoutil.SplitAtFirstHyphen returned (%q, %q, true)", s, parts, prefix, suffix)
				}
			}
		}
	})
}
