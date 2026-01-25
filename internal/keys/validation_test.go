package keys

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase epic", "e01", "E01"},
		{"uppercase epic", "E01", "E01"},
		{"lowercase task", "t-e04-f02-001", "T-E04-F02-001"},
		{"mixed case feature", "E01-feature-name", "E01-FEATURE-NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsEpicKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid uppercase", "E01", true},
		{"valid lowercase", "e01", true},
		{"valid max", "E99", true},
		{"invalid single digit", "E1", false},
		{"invalid three digits", "E001", false},
		{"invalid no number", "E", false},
		{"invalid wrong prefix", "F01", false},
		{"empty", "", false},
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

func TestIsFeatureKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid uppercase", "E04-F01", true},
		{"valid lowercase", "e04-f01", true},
		{"valid mixed", "e04-F01", true},
		{"invalid no dash", "E04F01", false},
		{"invalid wrong format", "E4-F01", false},
		{"invalid only epic", "E04", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFeatureKey(tt.input)
			if got != tt.want {
				t.Errorf("IsFeatureKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsFeatureKeySuffix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid uppercase", "F01", true},
		{"valid lowercase", "f01", true},
		{"valid max", "F99", true},
		{"invalid single digit", "F1", false},
		{"invalid wrong prefix", "E01", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFeatureKeySuffix(tt.input)
			if got != tt.want {
				t.Errorf("IsFeatureKeySuffix(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFeatureKey(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantEpic    string
		wantFeature string
		wantErr     bool
	}{
		{"valid uppercase", "E04-F01", "E04", "F01", false},
		{"valid lowercase", "e04-f01", "E04", "F01", false},
		{"invalid format", "E04F01", "", "", true},
		{"invalid epic", "E4-F01", "", "", true},
		{"empty", "", "", "", true},
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

func TestIsTaskKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid traditional", "T-E04-F01-001", true},
		{"valid with slug", "T-E04-F01-001-IMPLEMENT-AUTH", true},
		{"invalid no T prefix", "E04-F01-001", false},
		{"invalid wrong format", "T-E4-F01-001", false},
		{"invalid too short", "T-E04-F01", false},
		{"empty", "", false},
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

func TestIsShortTaskKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid short", "E04-F01-001", true},
		{"valid uppercase", "E04-F01-001", true},
		{"invalid with T prefix", "T-E04-F01-001", false},
		{"invalid wrong format", "E4-F01-001", false},
		{"invalid too short", "E04-F01", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsShortTaskKey(tt.input)
			if got != tt.want {
				t.Errorf("IsShortTaskKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeTaskKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"traditional format", "T-E01-F02-001", "T-E01-F02-001", false},
		{"short format", "E01-F02-001", "T-E01-F02-001", false},
		{"lowercase short", "e01-f02-001", "T-E01-F02-001", false},
		{"slugged short", "E01-F02-001-task-name", "T-E01-F02-001-TASK-NAME", false},
		{"invalid format", "INVALID", "", true},
		{"empty", "", "", true},
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

func TestParseTaskNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"valid min", "1", 1, false},
		{"valid mid", "123", 123, false},
		{"valid max", "999", 999, false},
		{"invalid zero", "0", 0, true},
		{"invalid too large", "1000", 0, true},
		{"invalid non-numeric", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseTaskNumber(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
