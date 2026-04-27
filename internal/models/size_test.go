package models

import (
	"errors"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// REQ-F-002 — ValidateSize: canonical numeric values
// ─────────────────────────────────────────────────────────────────────────────

// TC-F002-A: All valid canonical values accepted (table-driven)
func TestValidateSize_CanonicalValuesAccepted(t *testing.T) {
	canonical := []int{1, 2, 3, 5, 8, 13}
	for _, n := range canonical {
		n := n
		t.Run("valid", func(t *testing.T) {
			if err := ValidateSize(n); err != nil {
				t.Errorf("ValidateSize(%d) returned unexpected error: %v", n, err)
			}
		})
	}
}

// TC-F002-B: Invalid numeric values rejected (table-driven)
func TestValidateSize_InvalidValuesRejected(t *testing.T) {
	invalid := []int{0, 4, 6, 7, 9, 10, 11, 12, 14, -1, -8, 100, 21}
	for _, n := range invalid {
		n := n
		t.Run("invalid", func(t *testing.T) {
			err := ValidateSize(n)
			if err == nil {
				t.Errorf("ValidateSize(%d) expected error but got nil", n)
				return
			}
			if !errors.Is(err, ErrInvalidSize) {
				t.Errorf("ValidateSize(%d) returned wrong error type: got %v, want ErrInvalidSize", n, err)
			}
		})
	}
}

// TC-F002-C: Error wraps ErrInvalidSize sentinel
func TestValidateSize_ErrorWrapsErrInvalidSize(t *testing.T) {
	err := ValidateSize(4)
	if err == nil {
		t.Fatal("ValidateSize(4) expected error but got nil")
	}
	if !errors.Is(err, ErrInvalidSize) {
		t.Errorf("errors.Is(err, ErrInvalidSize) = false; err = %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// REQ-F-003 — ParseSize: bidirectional label mapping
// ─────────────────────────────────────────────────────────────────────────────

// TC-F003-A: ParseSize — all valid label forms (table-driven, case-insensitive, whitespace-trimmed)
func TestParseSize_ValidLabelForms(t *testing.T) {
	tests := []struct {
		input   string
		wantVal int
	}{
		{"XS", 1},
		{"xs", 1},
		{"Xs", 1},
		{" XS ", 1},
		{"S", 2},
		{"M", 3},
		{"L", 5},
		{"XL", 8},
		{"xl", 8},
		{"XXL", 13},
		{"xxl", 13},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("label_"+tt.input, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if err != nil {
				t.Errorf("ParseSize(%q) returned unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.wantVal {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.wantVal)
			}
		})
	}
}

// TC-F003-B: ParseSize — all valid numeric strings (table-driven)
func TestParseSize_ValidNumericStrings(t *testing.T) {
	tests := []struct {
		input   string
		wantVal int
	}{
		{"1", 1},
		{"2", 2},
		{"3", 3},
		{"5", 5},
		{"8", 8},
		{"13", 13},
		{" 5 ", 5}, // leading/trailing space
	}

	for _, tt := range tests {
		tt := tt
		t.Run("numeric_"+tt.input, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if err != nil {
				t.Errorf("ParseSize(%q) returned unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.wantVal {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.wantVal)
			}
		})
	}
}

// TC-F003-C: ParseSize — invalid inputs rejected (table-driven)
func TestParseSize_InvalidInputsRejected(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unknown label XXXL", "XXXL"},
		{"non-canonical numeric 4", "4"},
		{"word medium", "medium"},
		{"empty string", ""},
		{"zero", "0"},
		{"invalid 14", "14"},
		{"negative -1", "-1"},
		{"non-numeric abc", "abc"},
		{"whitespace only", " "},
		{"float 1.5", "1.5"},
		{"SQL injection", "'; DROP TABLE epics; --"},
		{"oversized 21 chars", strings.Repeat("A", 21)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSize(tt.input)
			if err == nil {
				t.Errorf("ParseSize(%q) expected error but got nil", tt.input)
				return
			}
			if !errors.Is(err, ErrInvalidSize) {
				t.Errorf("ParseSize(%q) returned wrong error type: got %v, want ErrInvalidSize", tt.input, err)
			}
		})
	}
}

// TC-F003-D: SizeLabel — all valid numerics map to labels (table-driven)
func TestSizeLabel_AllValidNumerics(t *testing.T) {
	tests := []struct {
		input     int
		wantLabel string
	}{
		{1, "XS"},
		{2, "S"},
		{3, "M"},
		{5, "L"},
		{8, "XL"},
		{13, "XXL"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("size_label", func(t *testing.T) {
			got, err := SizeLabel(tt.input)
			if err != nil {
				t.Errorf("SizeLabel(%d) returned unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.wantLabel {
				t.Errorf("SizeLabel(%d) = %q, want %q", tt.input, got, tt.wantLabel)
			}
		})
	}
}

// TC-F003-E: SizeLabel — invalid numeric returns error
func TestSizeLabel_InvalidNumericReturnsError(t *testing.T) {
	invalid := []int{0, 4, 6, 14, -1}
	for _, n := range invalid {
		n := n
		t.Run("invalid", func(t *testing.T) {
			label, err := SizeLabel(n)
			if err == nil {
				t.Errorf("SizeLabel(%d) expected error but got nil (label=%q)", n, label)
				return
			}
			if label != "" {
				t.Errorf("SizeLabel(%d) returned non-empty label on error: %q", n, label)
			}
			if !errors.Is(err, ErrInvalidSize) {
				t.Errorf("SizeLabel(%d) returned wrong error type: got %v, want ErrInvalidSize", n, err)
			}
		})
	}
}

// TC-F003-F: ParseSize/SizeLabel round-trip
// For each valid label, ParseSize(label) → SizeLabel(result) should recover the
// canonical (upper-cased) label.
func TestParseSize_SizeLabel_RoundTrip(t *testing.T) {
	labels := []string{"XS", "S", "M", "L", "XL", "XXL"}
	for _, label := range labels {
		label := label
		t.Run("roundtrip_"+label, func(t *testing.T) {
			n, err := ParseSize(label)
			if err != nil {
				t.Fatalf("ParseSize(%q) returned unexpected error: %v", label, err)
			}
			recovered, err := SizeLabel(n)
			if err != nil {
				t.Fatalf("SizeLabel(%d) returned unexpected error: %v", n, err)
			}
			if recovered != label {
				t.Errorf("round-trip for %q: ParseSize→SizeLabel = %q, want %q", label, recovered, label)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// REQ-NF-003 — Input sanitization
// ─────────────────────────────────────────────────────────────────────────────

// TC-SAN-A: ParseSize strips leading/trailing whitespace
func TestParseSize_StripsWhitespace(t *testing.T) {
	tests := []struct {
		input   string
		wantVal int
	}{
		{" XS ", 1},
		{" 5 ", 5},
		{"\t13\t", 13},
	}
	for _, tt := range tests {
		tt := tt
		t.Run("whitespace_stripped", func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if err != nil {
				t.Errorf("ParseSize(%q) returned unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.wantVal {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.wantVal)
			}
		})
	}
}

// TC-SAN-B: ParseSize is case-insensitive for labels
func TestParseSize_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"xs"},
		{"XS"},
		{"xS"},
		{"Xs"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run("case_"+tt.input, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if err != nil {
				t.Errorf("ParseSize(%q) returned unexpected error: %v", tt.input, err)
				return
			}
			if got != 1 {
				t.Errorf("ParseSize(%q) = %d, want 1", tt.input, got)
			}
		})
	}
}

// TC-SAN-C: ParseSize rejects oversized input (> 20 characters)
func TestParseSize_RejectsOversizedInput(t *testing.T) {
	oversized := strings.Repeat("A", 21)
	_, err := ParseSize(oversized)
	if err == nil {
		t.Errorf("ParseSize(21-char string) expected error but got nil")
		return
	}
	if !errors.Is(err, ErrInvalidSize) {
		t.Errorf("ParseSize(21-char string) returned wrong error type: got %v, want ErrInvalidSize", err)
	}
}

// TC-SAN-D: ParseSize rejects control characters and injection payloads.
// Note: inputs like "1\n" are NOT included here because strings.TrimSpace strips
// trailing newlines (whitespace), making "1\n" equivalent to "1" — a valid size.
// TC-SAN-A already confirms that whitespace-wrapped valid values are accepted.
// The payloads tested here are those that survive TrimSpace and must still be rejected.
func TestParseSize_RejectsInjectionPayloads(t *testing.T) {
	payloads := []struct {
		name  string
		input string
	}{
		{"SQL injection", "'; DROP TABLE epics; --"},
		{"null byte", "1\x00"},
		{"OR injection", "1 OR 1=1"},
	}
	for _, tt := range payloads {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSize(tt.input)
			if err == nil {
				t.Errorf("ParseSize(%q) expected error but got nil", tt.input)
			}
		})
	}
}

// TC-SAN-E: Error messages quote user input with %q
func TestParseSize_ErrorMessageQuotesInput(t *testing.T) {
	// ParseSize("4") should produce an error containing the input quoted via %q.
	_, err := ParseSize("4")
	if err == nil {
		t.Fatal("ParseSize(\"4\") expected error but got nil")
	}
	errMsg := err.Error()
	// The %q format produces `"4"` in the message.
	if !strings.Contains(errMsg, `"4"`) {
		t.Errorf("ParseSize(\"4\") error message does not contain quoted input %q: got %q", `"4"`, errMsg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Task-specific AC criteria (from T-E07-F42-001)
// ─────────────────────────────────────────────────────────────────────────────

// AC-T2: ErrInvalidSize sentinel is exported and wrappable via errors.Is
func TestErrInvalidSize_IsWrappable(t *testing.T) {
	// Ensure the sentinel is accessible and wrappable.
	wrapped := errors.New("outer: " + ErrInvalidSize.Error())
	if wrapped == nil {
		t.Fatal("expected wrapped error, got nil")
	}
	// Direct sentinel check
	err := ValidateSize(4)
	if !errors.Is(err, ErrInvalidSize) {
		t.Errorf("errors.Is(ValidateSize(4), ErrInvalidSize) = false; err = %v", err)
	}
}

// AC-T3: ParseSize("clear") returns a non-nil error (not a valid size value)
func TestParseSize_Clear_IsNotAValidSize(t *testing.T) {
	_, err := ParseSize("clear")
	if err == nil {
		t.Error(`ParseSize("clear") expected error but got nil; "clear" must not be a valid size value`)
	}
}
