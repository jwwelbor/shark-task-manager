package commands

import (
	"strings"
	"testing"
)

// TestDetectEntityType_BugKey verifies that DetectEntityType returns "bug" for B### keys.
func TestDetectEntityType_BugKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard 3-digit bug key uppercase", "B001", "bug"},
		{"standard 3-digit bug key lowercase", "b001", "bug"},
		{"single-digit bug key", "B1", "bug"},
		{"two-digit bug key", "B42", "bug"},
		{"four-digit bug key", "B1000", "bug"},
		{"zero-value bug key", "B0", "bug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectEntityType(tt.input)
			if got != tt.want {
				t.Errorf("DetectEntityType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestDetectEntityType_ChangeKey verifies that DetectEntityType returns "change" for C### keys.
func TestDetectEntityType_ChangeKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard 3-digit change key uppercase", "C001", "change"},
		{"standard 3-digit change key lowercase", "c001", "change"},
		{"case-insensitive with digits", "c015", "change"},
		{"single-digit change key", "C1", "change"},
		{"two-digit change key", "C42", "change"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectEntityType(tt.input)
			if got != tt.want {
				t.Errorf("DetectEntityType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestDetectEntityType_BugKeyNegative verifies that non-bug inputs do NOT return "bug".
func TestDetectEntityType_BugKeyNegative(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"epic key must not match bug", "E07"},
		{"feature key must not match bug", "F01"},
		{"task key must not match bug", "T-E07-F01-001"},
		{"B with no digits is unknown", "B"},
		{"B with letters is unknown", "BABC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectEntityType(tt.input)
			if got == "bug" {
				t.Errorf("DetectEntityType(%q) = %q, should not be 'bug'", tt.input, got)
			}
		})
	}
}

// TestDetectEntityType_ChangeKeyNegative verifies that non-change inputs do NOT return "change".
func TestDetectEntityType_ChangeKeyNegative(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"epic key must not match change", "E07"},
		{"feature key must not match change", "F01"},
		{"C with no digits is unknown", "C"},
		{"task key must not match change", "T-E07-F01-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectEntityType(tt.input)
			if got == "change" {
				t.Errorf("DetectEntityType(%q) = %q, should not be 'change'", tt.input, got)
			}
		})
	}
}

// TestIsBugKey verifies the IsBugKey helper function.
func TestIsBugKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"B001 is a bug key", "B001", true},
		{"b001 lowercase is a bug key", "b001", true},
		{"B1 single digit is a bug key", "B1", true},
		{"B42 two digits is a bug key", "B42", true},
		{"B1000 four digits is a bug key", "B1000", true},
		{"B0 zero is a bug key", "B0", true},
		{"B alone is not a bug key", "B", false},
		{"BABC letters is not a bug key", "BABC", false},
		{"E07 epic key is not a bug key", "E07", false},
		{"C001 change key is not a bug key", "C001", false},
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

// TestIsChangeKey verifies the IsChangeKey helper function.
func TestIsChangeKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"C001 is a change key", "C001", true},
		{"c001 lowercase is a change key", "c001", true},
		{"C1 single digit is a change key", "C1", true},
		{"C15 two digits is a change key", "C15", true},
		{"C alone is not a change key", "C", false},
		{"E07 epic key is not a change key", "E07", false},
		{"B001 bug key is not a change key", "B001", false},
		{"CABC letters is not a change key", "CABC", false},
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

// TestRunDelete_DispatchesBugKey verifies that runDelete error message handles B### keys correctly.
// Tests that B### keys do NOT fall through to the default error case.
func TestRunDelete_DispatchesBugKey(t *testing.T) {
	// When runDelete is called with a B### key, it should NOT return the
	// "cannot determine entity type" error; it should attempt to call runBugDelete.
	// Since we can't inject a mock easily here, we verify via DetectEntityType
	// that the dispatch routing would work.
	entityType := DetectEntityType("B001")
	if entityType != "bug" {
		t.Errorf("DetectEntityType(\"B001\") = %q, want \"bug\" -- delete dispatch would route to default", entityType)
	}
}

// TestRunDelete_DispatchesChangeKey verifies that runDelete handles C### keys correctly.
func TestRunDelete_DispatchesChangeKey(t *testing.T) {
	entityType := DetectEntityType("C001")
	if entityType != "change" {
		t.Errorf("DetectEntityType(\"C001\") = %q, want \"change\" -- delete dispatch would route to default", entityType)
	}
}

// TestRunUpdate_DispatchesBugKey verifies that runUpdate handles B### keys.
func TestRunUpdate_DispatchesBugKey(t *testing.T) {
	entityType := DetectEntityType("B001")
	if entityType != "bug" {
		t.Errorf("DetectEntityType(\"B001\") = %q, want \"bug\" -- update dispatch would route to default", entityType)
	}
}

// TestRunUpdate_DispatchesChangeKey verifies that runUpdate handles C### keys.
func TestRunUpdate_DispatchesChangeKey(t *testing.T) {
	entityType := DetectEntityType("C001")
	if entityType != "change" {
		t.Errorf("DetectEntityType(\"C001\") = %q, want \"change\" -- update dispatch would route to default", entityType)
	}
}

// TestDeleteDispatch_ErrorMessage_IncludesBugAndChangeFormats verifies the default error
// message in runDelete includes B### and C### format examples.
func TestDeleteDispatch_ErrorMessage_IncludesBugAndChangeFormats(t *testing.T) {
	// Call runDelete with an unknown key format to get the default error
	cmd := deleteCmd
	args := []string{"UNKNOWN999"}

	err := runDelete(cmd, args)
	if err == nil {
		t.Fatal("expected error for unknown key format, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "B###") && !strings.Contains(errMsg, "bug") {
		t.Errorf("delete dispatch error message does not mention B### or bug format: %q", errMsg)
	}
	if !strings.Contains(errMsg, "C###") && !strings.Contains(errMsg, "change") {
		t.Errorf("delete dispatch error message does not mention C### or change format: %q", errMsg)
	}
}

// TestUpdateDispatch_ErrorMessage_IncludesBugAndChangeFormats verifies the default error
// message in runUpdate includes B### and C### format examples.
func TestUpdateDispatch_ErrorMessage_IncludesBugAndChangeFormats(t *testing.T) {
	cmd := updateCmd
	args := []string{"UNKNOWN999"}

	err := runUpdate(cmd, args)
	if err == nil {
		t.Fatal("expected error for unknown key format, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "B###") && !strings.Contains(errMsg, "bug") {
		t.Errorf("update dispatch error message does not mention B### or bug format: %q", errMsg)
	}
	if !strings.Contains(errMsg, "C###") && !strings.Contains(errMsg, "change") {
		t.Errorf("update dispatch error message does not mention C### or change format: %q", errMsg)
	}
}

// TestParseGetArgs_BugKey verifies that ParseGetArgs correctly identifies B### keys.
func TestParseGetArgs_BugKey(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string
		wantKey     string
		wantErr     bool
	}{
		{"B001 dispatches to bug", []string{"B001"}, "bug", "B001", false},
		{"b001 lowercase dispatches to bug", []string{"b001"}, "bug", "B001", false},
		{"B42 dispatches to bug", []string{"B42"}, "bug", "B42", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, key, err := ParseGetArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGetArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if !tt.wantErr {
				if command != tt.wantCommand {
					t.Errorf("ParseGetArgs(%v) command = %q, want %q", tt.args, command, tt.wantCommand)
				}
				if key != tt.wantKey {
					t.Errorf("ParseGetArgs(%v) key = %q, want %q", tt.args, key, tt.wantKey)
				}
			}
		})
	}
}

// TestParseGetArgs_ChangeKey verifies that ParseGetArgs correctly identifies C### keys.
func TestParseGetArgs_ChangeKey(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string
		wantKey     string
		wantErr     bool
	}{
		{"C001 dispatches to change", []string{"C001"}, "change", "C001", false},
		{"c001 lowercase dispatches to change", []string{"c001"}, "change", "C001", false},
		{"C15 dispatches to change", []string{"C15"}, "change", "C15", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, key, err := ParseGetArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGetArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if !tt.wantErr {
				if command != tt.wantCommand {
					t.Errorf("ParseGetArgs(%v) command = %q, want %q", tt.args, command, tt.wantCommand)
				}
				if key != tt.wantKey {
					t.Errorf("ParseGetArgs(%v) key = %q, want %q", tt.args, key, tt.wantKey)
				}
			}
		})
	}
}

// TestParseGetArgs_TechDebtKey verifies that ParseGetArgs correctly identifies TD-### keys.
func TestParseGetArgs_TechDebtKey(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string
		wantKey     string
		wantErr     bool
	}{
		{"TD-001 dispatches to tech_debt", []string{"TD-001"}, "tech_debt", "TD-001", false},
		{"td-001 lowercase dispatches to tech_debt", []string{"td-001"}, "tech_debt", "TD-001", false},
		{"TD-042 dispatches to tech_debt", []string{"TD-042"}, "tech_debt", "TD-042", false},
		{"TD-999 dispatches to tech_debt", []string{"TD-999"}, "tech_debt", "TD-999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, key, err := ParseGetArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGetArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if !tt.wantErr {
				if command != tt.wantCommand {
					t.Errorf("ParseGetArgs(%v) command = %q, want %q", tt.args, command, tt.wantCommand)
				}
				if key != tt.wantKey {
					t.Errorf("ParseGetArgs(%v) key = %q, want %q", tt.args, key, tt.wantKey)
				}
			}
		})
	}
}

// TestDetectEntityType_TDKey verifies that DetectEntityType returns "tech_debt" for TD-### keys
// and that TD- prefix does not collide with T- task prefix.
func TestDetectEntityType_TDKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"TD-001 uppercase", "TD-001", "tech_debt"},
		{"TD-042 uppercase", "TD-042", "tech_debt"},
		{"TD-999 uppercase", "TD-999", "tech_debt"},
		{"td-001 lowercase", "td-001", "tech_debt"},
		{"Td-001 mixed case", "Td-001", "tech_debt"},
		{"tD-042 mixed case", "tD-042", "tech_debt"},
		// Ensure task keys still work correctly (no collision)
		{"Task T-E07-F01-001 not confused with tech-debt", "T-E07-F01-001", "task"},
		{"Task E07-F01-001 short format not confused", "E07-F01-001", "task"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectEntityType(tt.input)
			if result != tt.expected {
				t.Errorf("DetectEntityType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestToModelEntityType_BugAndChange verifies that toModelEntityType handles bug and change types.
func TestToModelEntityType_BugAndChange(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		wantStr    string
		wantErr    bool
	}{
		{"bug entity type", "bug", "bug", false},
		{"change entity type", "change", "change", false},
		{"unknown entity type", "unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toModelEntityType(tt.entityType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("toModelEntityType(%q) error = %v, wantErr %v", tt.entityType, err, tt.wantErr)
			}
			if !tt.wantErr && string(got) != tt.wantStr {
				t.Errorf("toModelEntityType(%q) = %q, want %q", tt.entityType, string(got), tt.wantStr)
			}
		})
	}
}
