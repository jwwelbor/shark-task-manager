package commands

import (
	"strings"
	"testing"
)

// TestEnhancedErrorMessages_Integration verifies that enhanced error messages
// work correctly in real-world command scenarios
func TestEnhancedErrorMessages_Integration(t *testing.T) {
	tests := []struct {
		name         string
		testFunc     func() error
		wantContains []string
	}{
		{
			name: "Invalid epic key shows helpful error",
			testFunc: func() error {
				return InvalidEpicKeyError("E1")
			},
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"Expected:",
				"two-digit",
				"Valid syntax:",
				"E07",
			},
		},
		{
			name: "Invalid feature key shows helpful error",
			testFunc: func() error {
				return InvalidFeatureKeyError("F1")
			},
			wantContains: []string{
				"Error:",
				"invalid feature key format",
				"Expected:",
				"two-digit",
				"Valid syntax:",
				"F01",
				"E07-F01",
			},
		},
		{
			name: "Invalid task key shows helpful error",
			testFunc: func() error {
				return InvalidTaskKeyError("T-E7-F1-1")
			},
			wantContains: []string{
				"Error:",
				"invalid task key format",
				"Expected:",
				"Valid syntax:",
				"T-E07-F01-001",
				"E07-F01-001",
			},
		},
		{
			name: "Missing arguments shows helpful error",
			testFunc: func() error {
				return MissingArgumentsError(2, 1, []string{
					"shark feature create E07 \"Title\"",
				})
			},
			wantContains: []string{
				"Error:",
				"missing required arguments",
				"Expected: 2",
				"Got: 1",
				"Valid syntax:",
			},
		},
		{
			name: "Too many arguments shows helpful error",
			testFunc: func() error {
				return TooManyArgumentsError(2, 4)
			},
			wantContains: []string{
				"Error:",
				"too many arguments",
				"Expected: 2",
				"Got: 4",
				"quote multi-word",
			},
		},
		{
			name: "Ambiguous key shows helpful error",
			testFunc: func() error {
				return AmbiguousKeyError("001", []string{
					"T-E07-F01-001",
					"T-E07-F02-001",
				})
			},
			wantContains: []string{
				"Error:",
				"ambiguous key",
				"multiple matches",
				"Did you mean:",
				"T-E07-F01-001",
				"T-E07-F02-001",
			},
		},
		{
			name: "Not found error shows helpful suggestions",
			testFunc: func() error {
				return NotFoundError("task", "T-E99-F99-999")
			},
			wantContains: []string{
				"Error:",
				"task not found",
				"Suggestions:",
				"shark task list",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()

			// Verify all required content is present
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing %q\nGot: %s", want, errMsg)
				}
			}

			// Verify error message is multiline (user-friendly format)
			if !strings.Contains(errMsg, "\n") {
				t.Errorf("error message should be multi-line for readability\nGot: %s", errMsg)
			}

			// Verify error starts with "Error:"
			if !strings.HasPrefix(errMsg, "Error:") {
				t.Errorf("error message should start with 'Error:'\nGot: %s", errMsg)
			}
		})
	}
}

// TestErrorMessageConsistency verifies all error messages follow consistent formatting
func TestErrorMessageConsistency(t *testing.T) {
	errorFactories := []struct {
		name string
		fn   func() error
	}{
		{
			name: "InvalidEpicKeyError",
			fn:   func() error { return InvalidEpicKeyError("E1") },
		},
		{
			name: "InvalidFeatureKeyError",
			fn:   func() error { return InvalidFeatureKeyError("F1") },
		},
		{
			name: "InvalidTaskKeyError",
			fn:   func() error { return InvalidTaskKeyError("T1") },
		},
		{
			name: "MissingArgumentsError",
			fn:   func() error { return MissingArgumentsError(2, 1, []string{"example"}) },
		},
		{
			name: "TooManyArgumentsError",
			fn:   func() error { return TooManyArgumentsError(2, 3) },
		},
		{
			name: "InvalidPositionalArgsError",
			fn:   func() error { return InvalidPositionalArgsError("cmd", "reason", []string{"ex"}) },
		},
		{
			name: "AmbiguousKeyError",
			fn:   func() error { return AmbiguousKeyError("key", []string{"match1"}) },
		},
		{
			name: "NotFoundError",
			fn:   func() error { return NotFoundError("task", "T-E07-F01-001") },
		},
	}

	for _, ef := range errorFactories {
		t.Run(ef.name, func(t *testing.T) {
			err := ef.fn()
			if err == nil {
				t.Fatalf("%s: expected error, got nil", ef.name)
			}

			errMsg := err.Error()

			// All errors should start with "Error:"
			if !strings.HasPrefix(errMsg, "Error:") {
				t.Errorf("%s: should start with 'Error:', got: %s", ef.name, errMsg)
			}

			// All errors should be multiline
			if !strings.Contains(errMsg, "\n") {
				t.Errorf("%s: should be multiline, got: %s", ef.name, errMsg)
			}

			// All errors should be at least 50 characters (substantive)
			if len(errMsg) < 50 {
				t.Errorf("%s: too short (%d chars), should be substantive, got: %s",
					ef.name, len(errMsg), errMsg)
			}

			// All errors should not exceed 1000 characters (concise)
			if len(errMsg) > 1000 {
				t.Errorf("%s: too long (%d chars), should be concise, got: %s",
					ef.name, len(errMsg), errMsg)
			}
		})
	}
}

// TestRealWorldErrorScenarios tests error messages in realistic command scenarios
func TestRealWorldErrorScenarios(t *testing.T) {
	t.Run("User types short epic key", func(t *testing.T) {
		_, err := ParseFeatureListArgs([]string{"E1"})
		if err == nil {
			t.Fatal("expected error for invalid epic key")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "two-digit") {
			t.Errorf("should explain epic keys must be two-digit, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "E07") || !strings.Contains(errMsg, "E04") {
			t.Errorf("should show valid examples, got: %s", errMsg)
		}
	})

	t.Run("User provides too many args to feature list", func(t *testing.T) {
		_, err := ParseFeatureListArgs([]string{"E07", "F01"})
		if err == nil {
			t.Fatal("expected error for too many arguments")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "too many arguments") {
			t.Errorf("should indicate too many arguments, got: %s", errMsg)
		}
	})

	t.Run("User forgets to quote feature title", func(t *testing.T) {
		// Simulates: shark feature create E07 My Feature Title
		// Instead of: shark feature create E07 "My Feature Title"
		_, _, err := ParseFeatureCreateArgs([]string{"E07", "My", "Feature", "Title"})
		if err == nil {
			t.Fatal("expected error for too many arguments")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "quote") {
			t.Errorf("should remind user to quote titles, got: %s", errMsg)
		}
	})

	t.Run("User provides invalid task key format", func(t *testing.T) {
		_, _, err := ParseTaskListArgs([]string{"E7-F1-001"})
		if err == nil {
			t.Fatal("expected error for invalid key format")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "Expected:") {
			t.Errorf("should explain expected format, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "Valid syntax:") {
			t.Errorf("should show valid examples, got: %s", errMsg)
		}
	})
}
