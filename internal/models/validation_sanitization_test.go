package models

import (
	"errors"
	"strings"
	"testing"
)

// TestValidateEpicKey_AllowlistPattern verifies that epic key validation uses a strict
// regex allowlist that rejects characters outside the expected pattern.
//
// Documents the pattern described in .claude/rules/go/input-sanitization.md.
func TestValidateEpicKey_AllowlistPattern(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		// Valid keys
		{"valid E07", "E07", false},
		{"valid E00", "E00", false},
		{"valid E99", "E99", false},

		// Invalid: empty or whitespace
		{"empty string", "", true},
		{"whitespace only", "   ", true},

		// Invalid: wrong prefix
		{"lowercase e", "e07", true},
		{"wrong prefix T", "T07", true},

		// Invalid: wrong format
		{"no digits", "E", true},
		{"one digit", "E7", true},
		{"three digits", "E007", true},
		{"extra suffix", "E07-extra", true},

		// Invalid: injection attempts — these must be rejected by the allowlist
		{"SQL injection attempt", "E07 OR 1=1", true},
		{"single quote injection", "E07'", true},
		{"semicolon injection", "E07;DROP TABLE tasks", true},
		{"newline injection", "E07\n", true},
		{"null byte injection", "E07\x00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEpicKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEpicKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !errors.Is(err, ErrInvalidEpicKey) {
					t.Errorf("ValidateEpicKey(%q) returned wrong error type: got %v, want ErrInvalidEpicKey", tt.key, err)
				}
			}
		})
	}
}

// TestValidateFeatureKey_AllowlistPattern verifies the feature key regex allowlist.
func TestValidateFeatureKey_AllowlistPattern(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		// Valid keys
		{"valid E07-F01", "E07-F01", false},
		{"valid E00-F00", "E00-F00", false},
		{"valid E99-F99", "E99-F99", false},

		// Invalid: wrong format
		{"missing feature part", "E07", true},
		{"lowercase", "e07-f01", true},
		{"extra digits in epic", "E007-F01", true},
		{"extra digits in feature", "E07-F001", true},
		{"missing dash", "E07F01", true},

		// Invalid: injection attempts
		{"SQL injection", "E07-F01 OR 1=1", true},
		{"single quote", "E07-F01'", true},
		{"null byte", "E07-F01\x00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeatureKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFeatureKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !errors.Is(err, ErrInvalidFeatureKey) {
					t.Errorf("ValidateFeatureKey(%q) returned wrong error type: got %v, want ErrInvalidFeatureKey", tt.key, err)
				}
			}
		})
	}
}

// TestValidateTaskKey_AllowlistPattern verifies the task key regex allowlist.
func TestValidateTaskKey_AllowlistPattern(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		// Valid keys
		{"valid T-E07-F01-001", "T-E07-F01-001", false},
		{"valid T-E00-F00-000", "T-E00-F00-000", false},
		{"valid T-E99-F99-999", "T-E99-F99-999", false},

		// Invalid: missing T- prefix
		{"no T prefix", "E07-F01-001", true},
		{"lowercase t prefix", "t-E07-F01-001", true},

		// Invalid: wrong digit counts
		{"extra task digits", "T-E07-F01-0001", true},
		{"too few task digits", "T-E07-F01-01", true},

		// Invalid: injection attempts
		{"SQL injection", "T-E07-F01-001 OR 1=1", true},
		{"single quote", "T-E07-F01-001'", true},
		{"backslash", "T-E07-F01-001\\", true},
		{"null byte", "T-E07-F01-001\x00", true},
		{"space injection", "T-E07-F01-001 ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !errors.Is(err, ErrInvalidTaskKey) {
					t.Errorf("ValidateTaskKey(%q) returned wrong error type: got %v, want ErrInvalidTaskKey", tt.key, err)
				}
			}
		})
	}
}

// TestValidateNoteType_EnumAllowlist verifies that only defined enum values are accepted.
//
// Documents the enum allowlist pattern from input-sanitization.md.
func TestValidateNoteType_EnumAllowlist(t *testing.T) {
	validTypes := []string{
		"comment", "decision", "blocker", "solution",
		"reference", "implementation", "testing", "future",
		"question", "rejection", "requirement",
		// B027: "review" must be accepted so the canonical code-review
		// workflow (which emits --type=review at PASS verdicts) works
		// across all entity types — including bugs.
		"review",
	}

	for _, validType := range validTypes {
		t.Run("valid_"+validType, func(t *testing.T) {
			err := ValidateNoteType(validType)
			if err != nil {
				t.Errorf("ValidateNoteType(%q) returned unexpected error: %v", validType, err)
			}
		})
	}

	invalidInputs := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"unknown type", "unknown"},
		{"uppercase valid", "COMMENT"},
		{"mixed case", "Comment"},
		{"SQL injection", "comment OR 1=1"},
		{"single quote", "comment'"},
		{"semicolon", "comment;DROP TABLE notes"},
	}

	for _, tt := range invalidInputs {
		t.Run("invalid_"+tt.name, func(t *testing.T) {
			err := ValidateNoteType(tt.input)
			if err == nil {
				t.Errorf("ValidateNoteType(%q) expected error but got nil", tt.input)
			}
			if !errors.Is(err, ErrInvalidNoteType) {
				t.Errorf("ValidateNoteType(%q) returned wrong error type: got %v, want ErrInvalidNoteType", tt.input, err)
			}
		})
	}
}

// TestValidateNoteType_ErrorMessageLocksAllowlist guards against drift
// between the validNoteTypes allowlist and the user-facing error
// message rendered by ValidateNoteType. Both must be sourced from the
// same slice so that adding a new note type automatically updates the
// error hint. TD-015 was filed precisely because they had drifted
// ("review" was accepted by the validator but missing from the
// sentinel message); this test makes that drift impossible to
// reintroduce.
func TestValidateNoteType_ErrorMessageLocksAllowlist(t *testing.T) {
	err := ValidateNoteType("definitely-not-a-real-note-type")
	if err == nil {
		t.Fatal("ValidateNoteType returned nil for invalid input")
	}
	msg := err.Error()

	for _, nt := range ValidNoteTypes() {
		if !strings.Contains(msg, nt) {
			t.Errorf("error message missing note type %q (drift from allowlist); full message: %s", nt, msg)
		}
	}

	// Sanity: ValidNoteTypes() must match the test's own expectations
	// for the canonical set. If a new type is added, update the list
	// here too — this is an intentional cross-check that catches both
	// silent removals from validNoteTypes and accidental additions.
	want := map[string]bool{
		"comment": true, "decision": true, "blocker": true, "solution": true,
		"reference": true, "implementation": true, "testing": true,
		"future": true, "question": true, "rejection": true,
		"requirement": true, "review": true, "review-finding": true,
	}
	got := ValidNoteTypes()
	if len(got) != len(want) {
		t.Errorf("ValidNoteTypes() length = %d, want %d (%v)", len(got), len(want), got)
	}
	for _, nt := range got {
		if !want[nt] {
			t.Errorf("ValidNoteTypes() returned unexpected type %q — update the test if this is intentional", nt)
		}
	}
}

// TestValidateRelationshipType_EnumAllowlist verifies that only defined relationship types are accepted.
func TestValidateRelationshipType_EnumAllowlist(t *testing.T) {
	validTypes := []string{
		"depends_on", "blocks", "related_to", "follows",
		"spawned_from", "duplicates", "references", "linked_to", "question_blocks",
	}

	for _, validType := range validTypes {
		t.Run("valid_"+validType, func(t *testing.T) {
			err := ValidateRelationshipType(validType)
			if err != nil {
				t.Errorf("ValidateRelationshipType(%q) returned unexpected error: %v", validType, err)
			}
		})
	}

	invalidInputs := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"unknown type", "links_to"},
		{"uppercase", "DEPENDS_ON"},
		{"SQL injection", "depends_on OR 1=1"},
		{"single quote", "depends_on'"},
	}

	for _, tt := range invalidInputs {
		t.Run("invalid_"+tt.name, func(t *testing.T) {
			err := ValidateRelationshipType(tt.input)
			if err == nil {
				t.Errorf("ValidateRelationshipType(%q) expected error but got nil", tt.input)
			}
			if !errors.Is(err, ErrInvalidRelationshipType) {
				t.Errorf("ValidateRelationshipType(%q) returned wrong error type: got %v, want ErrInvalidRelationshipType", tt.input, err)
			}
		})
	}
}

// TestValidateDependsOn_JSONArrayParsing verifies that the depends_on field is parsed
// and validated structurally before reaching the database.
//
// Documents the JSON array validation pattern from input-sanitization.md.
func TestValidateDependsOn_JSONArrayParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid inputs
		{"empty string", "", false},
		{"null string", "null", false},
		{"empty array", "[]", false},
		{"single valid task", `["T-E07-F01-001"]`, false},
		{"multiple valid tasks", `["T-E07-F01-001","T-E07-F01-002"]`, false},

		// Invalid: malformed JSON
		{"not JSON", "not-json", true},
		{"incomplete array", `["T-E07-F01-001"`, true},
		{"object instead of array", `{"key":"value"}`, true},
		{"JSON injection attempt", `["T-E07-F01-001","' OR '1'='1"]`, true},

		// Invalid: valid JSON but invalid task keys inside
		{"invalid task key format", `["E07-F01-001"]`, true},        // missing T- prefix
		{"lowercase task key", `["t-e07-f01-001"]`, true},           // lowercase
		{"injection in task key", `["T-E07-F01-001 OR 1=1"]`, true}, // SQL injection in key
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDependsOn(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDependsOn(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestValidateAgentType_WhitespaceSanitization specifically tests the whitespace
// sanitization behavior documented in input-sanitization.md.
func TestValidateAgentType_WhitespaceSanitization(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Whitespace-only inputs must be rejected (not treated as valid)
		{"only spaces", "     ", true},
		{"only tabs", "\t\t\t", true},
		{"only newlines", "\n\n", true},
		{"mixed whitespace", " \t\n ", true},

		// Non-empty after trimming must be accepted
		{"leading spaces", "  backend", false},
		{"trailing spaces", "backend  ", false},
		{"surrounding spaces", "  backend  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestValidateErrorMessages_QuotedInput verifies that error messages quote user input
// using %q format to prevent log injection and make injected characters visible.
//
// Documents the error message pattern from input-sanitization.md.
func TestValidateErrorMessages_QuotedInput(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"contains newline injection", "E07\n", true},
		{"contains carriage return", "E07\r", true},
		{"contains special chars", "E07<script>", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEpicKey(tt.key)
			if err == nil {
				t.Errorf("ValidateEpicKey(%q) expected error but got nil", tt.key)
				return
			}
			// Error message should use %q quoting (the invalid value appears quoted in the error)
			errMsg := err.Error()
			if !strings.Contains(errMsg, "got ") {
				t.Errorf("ValidateEpicKey error message does not include 'got': %q", errMsg)
			}
		})
	}
}

// TestValidateJSONArray_StructuralValidation verifies that ValidateJSONArray
// rejects non-array JSON structures and malformed JSON.
func TestValidateJSONArray_StructuralValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid
		{"empty string", "", false},
		{"null string", "null", false},
		{"empty array", "[]", false},
		{"string array", `["a","b","c"]`, false},

		// Invalid: malformed JSON
		{"plain string", "hello", true},
		{"json object", `{"key":"val"}`, true},
		{"incomplete", `["a"`, true},
		{"unquoted values", `[a, b]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSONArray(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSONArray(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
