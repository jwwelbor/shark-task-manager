package commands

import (
	"strings"
	"testing"
)

// TestParseFeatureKeyErrorMessages tests that ParseFeatureKey uses enhanced error templates
func TestParseFeatureKeyErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantContains []string
	}{
		{
			name:  "invalid feature key shows enhanced error",
			input: "F1",
			wantContains: []string{
				"Error:",
				"invalid feature key format",
				"Expected:",
				"Valid syntax:",
			},
		},
		{
			name:  "lowercase invalid feature key",
			input: "f1",
			wantContains: []string{
				"Error:",
				"invalid feature key format",
				"Expected:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFeatureKey(tt.input)
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

// TestParseFeatureListArgsErrorMessages tests enhanced error messages
func TestParseFeatureListArgsErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantContains []string
	}{
		{
			name: "too many arguments shows enhanced error",
			args: []string{"E07", "F01"},
			wantContains: []string{
				"Error:",
				"too many arguments",
			},
		},
		{
			name: "invalid epic key shows enhanced error",
			args: []string{"E1"},
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"Expected:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFeatureListArgs(tt.args)
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

// TestParseTaskListArgsErrorMessages tests enhanced error messages
func TestParseTaskListArgsErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantContains []string
	}{
		{
			name: "too many arguments",
			args: []string{"E07", "F01", "extra"},
			wantContains: []string{
				"Error:",
				"too many arguments",
			},
		},
		{
			name: "invalid epic key",
			args: []string{"E1"},
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"Expected:",
			},
		},
		{
			name: "invalid feature key in second position",
			args: []string{"E07", "F1"},
			wantContains: []string{
				"Error:",
				"invalid feature key format",
				"Expected:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseTaskListArgs(tt.args)
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

// TestParseListArgsErrorMessages tests enhanced error messages for generic list
func TestParseListArgsErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantContains []string
	}{
		{
			name: "too many arguments",
			args: []string{"E07", "F01", "extra"},
			wantContains: []string{
				"Error:",
				"too many arguments",
			},
		},
		{
			name: "invalid epic key",
			args: []string{"E1"},
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"Expected:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseListArgs(tt.args)
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

// TestParseGetArgsErrorMessages tests enhanced error messages for get commands
func TestParseGetArgsErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantContains []string
	}{
		{
			name: "missing arguments",
			args: []string{},
			wantContains: []string{
				"Error:",
				"missing",
			},
		},
		{
			name: "too many arguments",
			args: []string{"E07", "F01", "T001", "extra"},
			wantContains: []string{
				"Error:",
				"too many arguments",
			},
		},
		{
			name: "invalid epic key",
			args: []string{"E1"},
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"Expected:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseGetArgs(tt.args)
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

// TestParseFeatureCreateArgsErrorMessages tests enhanced error messages
func TestParseFeatureCreateArgsErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantContains []string
	}{
		{
			name: "too many arguments",
			args: []string{"E07", "Title", "extra"},
			wantContains: []string{
				"Error:",
				"too many arguments",
			},
		},
		{
			name: "invalid epic key",
			args: []string{"E1", "Title"},
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"Expected:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFeatureCreateArgs(tt.args)
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

// TestParseTaskCreateArgsErrorMessages tests enhanced error messages
func TestParseTaskCreateArgsErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantContains []string
	}{
		{
			name: "too many arguments",
			args: []string{"E07", "F01", "Title", "extra"},
			wantContains: []string{
				"Error:",
				"too many arguments",
			},
		},
		{
			name: "invalid epic key in 3-arg format",
			args: []string{"E1", "F01", "Title"},
			wantContains: []string{
				"Error:",
				"invalid epic key format",
				"Expected:",
			},
		},
		{
			name: "invalid feature key in 3-arg format",
			args: []string{"E07", "F1", "Title"},
			wantContains: []string{
				"Error:",
				"invalid feature key format",
				"Expected:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseTaskCreateArgs(tt.args)
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
