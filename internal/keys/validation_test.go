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

func TestIsBugKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid bug keys - variable digit count
		{"valid 3 digits uppercase", "B001", true},
		{"valid 3 digits lowercase", "b001", true},
		{"valid 1 digit", "B1", true},
		{"valid 2 digits", "B42", true},
		{"valid 4 digits", "B1000", true},
		{"valid zero", "B0", true},
		// Invalid bug keys
		{"invalid no digits", "B", false},
		{"invalid alpha", "BABC", false},
		{"invalid wrong prefix", "E01", false},
		{"invalid change key", "C001", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBugKey(tt.input)
			if got != tt.want {
				t.Errorf("IsBugKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsChangeKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid change-card key aliases
		{"valid canonical uppercase", "CC-001", true},
		{"valid canonical lowercase", "cc-001", true},
		{"valid compact CC alias", "CC001", true},
		{"valid compact C alias", "C001", true},
		{"valid hyphen C alias", "C-001", true},
		{"valid 1 digit alias", "C1", true},
		{"valid 4 digits alias", "C1000", true},
		{"valid zero", "C0", true},
		// Invalid change keys
		{"invalid no digits", "C", false},
		{"invalid CC no digits", "CC", false},
		{"invalid alpha", "CABC", false},
		{"invalid wrong prefix", "E01", false},
		{"invalid bug key", "B001", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsChangeKey(tt.input)
			if got != tt.want {
				t.Errorf("IsChangeKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeChangeKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"canonical", "CC-001", "CC-001", false},
		{"compact CC alias", "CC001", "CC-001", false},
		{"compact C alias", "C001", "CC-001", false},
		{"hyphen C alias", "C-001", "CC-001", false},
		{"lowercase", "cc001", "CC-001", false},
		{"pad one digit", "C1", "CC-001", false},
		{"pad two digits", "C42", "CC-042", false},
		{"preserve four digits", "C1000", "CC-1000", false},
		{"bug key invalid", "B001", "", true},
		{"tech debt key invalid", "TD-001", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeChangeKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeChangeKey(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("NormalizeChangeKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsTechDebtKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid tech-debt keys
		{"valid uppercase TD-001", "TD-001", true},
		{"valid lowercase td-001", "td-001", true},
		{"valid TD-042", "TD-042", true},
		{"valid TD-999", "TD-999", true},
		{"valid mixed case Td-001", "Td-001", true},
		// Invalid tech-debt keys
		{"invalid no digits TD-", "TD-", false},
		{"invalid two digits TD-01", "TD-01", false},
		{"invalid four digits TD-0001", "TD-0001", false},
		{"invalid no hyphen TD001", "TD001", false},
		{"invalid letters TD-abc", "TD-abc", false},
		{"invalid wrong prefix B001", "B001", false},
		{"invalid task key", "T-E01-F01-001", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTechDebtKey(tt.input)
			if got != tt.want {
				t.Errorf("IsTechDebtKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsQuestionKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid lower boundary Q001", "Q001", true},
		{"valid lowercase q100", "q100", true},
		{"valid upper boundary Q999", "Q999", true},
		{"invalid reserved Q000", "Q000", false},
		{"invalid two digits Q01", "Q01", false},
		{"invalid four digits Q0001", "Q0001", false},
		{"invalid no digits Q", "Q", false},
		{"invalid wrong prefix B001", "B001", false},
		{"invalid task key", "T-E01-F01-001", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsQuestionKey(tt.input)
			if got != tt.want {
				t.Errorf("IsQuestionKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsSprintKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid sprint keys (strict 3-digit)
		{"valid uppercase S001", "S001", true},
		{"valid lowercase s001", "s001", true},
		{"valid S024", "S024", true},
		{"valid S999", "S999", true},
		{"valid mixed case lowercase same as uppercase", "s999", true},
		// Invalid sprint keys
		{"invalid 1 digit S1", "S1", false},
		{"invalid 2 digits S42", "S42", false},
		{"invalid 4 digits S0001", "S0001", false},
		{"invalid 4 digits high S1000", "S1000", false},
		{"invalid no digits S", "S", false},
		{"invalid letters S-abc", "Sabc", false},
		{"invalid SPRINT lookalike", "SPRINT-1", false},
		{"invalid wrong prefix B001", "B001", false},
		{"invalid task key", "T-E01-F01-001", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSprintKey(tt.input)
			if got != tt.want {
				t.Errorf("IsSprintKey(%q) = %v, want %v", tt.input, got, tt.want)
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
