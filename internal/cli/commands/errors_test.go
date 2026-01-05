package commands

import (
	"strings"
	"testing"
)

// TestInvalidEpicKeyError tests the epic key error formatting
func TestInvalidEpicKeyError(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "lowercase epic key",
			key:  "e1",
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"e1",
				"Expected:",
				"two-digit",
				"Valid syntax:",
				"E07",
				"E04",
			},
		},
		{
			name: "single digit epic key",
			key:  "E1",
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"E1",
				"Expected:",
				"two-digit",
				"Valid syntax:",
			},
		},
		{
			name: "non-numeric epic key",
			key:  "EAB",
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"EAB",
				"Expected:",
				"Valid syntax:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InvalidEpicKeyError(tt.key)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()

			// Check for required content
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing %q\nGot: %s", want, errMsg)
				}
			}

			// Check for excluded content
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(errMsg, notWant) {
					t.Errorf("error message should not contain %q\nGot: %s", notWant, errMsg)
				}
			}
		})
	}
}

// TestInvalidFeatureKeyError tests the feature key error formatting
func TestInvalidFeatureKeyError(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		wantContains []string
	}{
		{
			name: "lowercase feature key",
			key:  "f1",
			wantContains: []string{
				"Error:",
				"invalid feature key format",
				"f1",
				"Expected:",
				"Valid syntax:",
				"F01",
				"E07-F01",
			},
		},
		{
			name: "single digit feature key",
			key:  "F1",
			wantContains: []string{
				"Error:",
				"invalid feature key format",
				"F1",
				"Expected:",
				"two-digit",
			},
		},
		{
			name: "invalid epic-feature combo",
			key:  "E1-F01",
			wantContains: []string{
				"Error:",
				"invalid feature key format",
				"E1-F01",
				"Expected:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InvalidFeatureKeyError(tt.key)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing %q\nGot: %s", want, errMsg)
				}
			}
		})
	}
}

// TestInvalidTaskKeyError tests the task key error formatting
func TestInvalidTaskKeyError(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		wantContains []string
	}{
		{
			name: "lowercase task key",
			key:  "t-e07-f01-001",
			wantContains: []string{
				"Error:",
				"invalid task key format",
				"t-e07-f01-001",
				"Expected:",
				"Valid syntax:",
				"T-E07-F01-001",
				"E07-F01-001",
			},
		},
		{
			name: "missing dashes",
			key:  "TE07F01001",
			wantContains: []string{
				"Error:",
				"invalid task key format",
				"TE07F01001",
				"Expected:",
			},
		},
		{
			name: "wrong number format",
			key:  "T-E7-F1-1",
			wantContains: []string{
				"Error:",
				"invalid task key format",
				"T-E7-F1-1",
				"Expected:",
				"two-digit",
				"three-digit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InvalidTaskKeyError(tt.key)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing %q\nGot: %s", want, errMsg)
				}
			}
		})
	}
}

// TestMissingArgumentsError tests the missing arguments error formatting
func TestMissingArgumentsError(t *testing.T) {
	tests := []struct {
		name         string
		expected     int
		got          int
		examples     []string
		wantContains []string
	}{
		{
			name:     "missing one argument",
			expected: 2,
			got:      1,
			examples: []string{
				"shark feature create E07 \"Feature Title\"",
				"shark feature create E04 \"User Management\"",
			},
			wantContains: []string{
				"Error:",
				"missing required arguments",
				"Expected: 2",
				"Got: 1",
				"Valid syntax:",
				"shark feature create E07",
				"shark feature create E04",
			},
		},
		{
			name:     "missing multiple arguments",
			expected: 3,
			got:      0,
			examples: []string{
				"shark task create E07 F01 \"Task Title\"",
			},
			wantContains: []string{
				"Error:",
				"missing required arguments",
				"Expected: 3",
				"Got: 0",
				"Valid syntax:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MissingArgumentsError(tt.expected, tt.got, tt.examples)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing %q\nGot: %s", want, errMsg)
				}
			}
		})
	}
}

// TestTooManyArgumentsError tests the too many arguments error formatting
func TestTooManyArgumentsError(t *testing.T) {
	tests := []struct {
		name         string
		expected     int
		got          int
		wantContains []string
	}{
		{
			name:     "one extra argument",
			expected: 2,
			got:      3,
			wantContains: []string{
				"Error:",
				"too many arguments",
				"Expected: 2",
				"Got: 3",
			},
		},
		{
			name:     "many extra arguments",
			expected: 1,
			got:      5,
			wantContains: []string{
				"Error:",
				"too many arguments",
				"Expected: 1",
				"Got: 5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TooManyArgumentsError(tt.expected, tt.got)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing %q\nGot: %s", want, errMsg)
				}
			}
		})
	}
}

// TestInvalidPositionalArgsError tests the invalid positional args error formatting
func TestInvalidPositionalArgsError(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		reason       string
		examples     []string
		wantContains []string
	}{
		{
			name:    "invalid epic and feature combination",
			command: "task list",
			reason:  "cannot specify both epic and feature as positional arguments",
			examples: []string{
				"shark task list E07",
				"shark task list E07-F01",
				"shark task list --epic=E07 --feature=F01",
			},
			wantContains: []string{
				"Error:",
				"invalid arguments for task list",
				"cannot specify both epic and feature",
				"Valid syntax:",
				"shark task list E07",
				"shark task list E07-F01",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InvalidPositionalArgsError(tt.command, tt.reason, tt.examples)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing %q\nGot: %s", want, errMsg)
				}
			}
		})
	}
}

// TestAmbiguousKeyError tests the ambiguous key error formatting
func TestAmbiguousKeyError(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		suggestions  []string
		wantContains []string
	}{
		{
			name: "ambiguous short task key",
			key:  "001",
			suggestions: []string{
				"T-E07-F01-001",
				"T-E07-F02-001",
				"T-E04-F01-001",
			},
			wantContains: []string{
				"Error:",
				"ambiguous key",
				"001",
				"multiple matches",
				"Did you mean:",
				"T-E07-F01-001",
				"T-E07-F02-001",
				"T-E04-F01-001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AmbiguousKeyError(tt.key, tt.suggestions)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing %q\nGot: %s", want, errMsg)
				}
			}
		})
	}
}

// TestErrorMessageQuality tests that all error messages meet quality standards
func TestErrorMessageQuality(t *testing.T) {
	errorFuncs := []struct {
		name string
		fn   func() error
	}{
		{
			name: "InvalidEpicKeyError",
			fn:   func() error { return InvalidEpicKeyError("e1") },
		},
		{
			name: "InvalidFeatureKeyError",
			fn:   func() error { return InvalidFeatureKeyError("f1") },
		},
		{
			name: "InvalidTaskKeyError",
			fn:   func() error { return InvalidTaskKeyError("t-e7-f1-1") },
		},
		{
			name: "MissingArgumentsError",
			fn: func() error {
				return MissingArgumentsError(2, 1, []string{"example1", "example2"})
			},
		},
		{
			name: "TooManyArgumentsError",
			fn:   func() error { return TooManyArgumentsError(2, 3) },
		},
	}

	for _, tf := range errorFuncs {
		t.Run(tf.name, func(t *testing.T) {
			err := tf.fn()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()

			// All error messages must start with "Error:"
			if !strings.HasPrefix(errMsg, "Error:") {
				t.Errorf("error message should start with 'Error:'\nGot: %s", errMsg)
			}

			// All error messages should contain "Expected:" (except TooManyArgumentsError)
			if tf.name != "TooManyArgumentsError" {
				if !strings.Contains(errMsg, "Expected:") {
					t.Errorf("error message should contain 'Expected:'\nGot: %s", errMsg)
				}
			}

			// Error messages should not be empty
			if len(errMsg) < 20 {
				t.Errorf("error message too short (< 20 chars): %s", errMsg)
			}

			// Error messages should be multi-line (contain newlines)
			if !strings.Contains(errMsg, "\n") {
				t.Errorf("error message should be multi-line\nGot: %s", errMsg)
			}
		})
	}
}
