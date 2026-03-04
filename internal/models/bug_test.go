package models

import (
	"errors"
	"strings"
	"testing"
)

func TestBug_Validate(t *testing.T) {
	tests := []struct {
		name    string
		bug     Bug
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid bug - high severity",
			bug: Bug{
				Key:      "B001",
				Title:    "Login button not working",
				Status:   BugStatus("reported"),
				Severity: BugSeverityHigh,
			},
			wantErr: false,
		},
		{
			name: "valid bug - critical severity",
			bug: Bug{
				Key:      "B042",
				Title:    "Data corruption on save",
				Status:   BugStatus("triaged"),
				Severity: BugSeverityCritical,
			},
			wantErr: false,
		},
		{
			name: "valid bug - medium severity",
			bug: Bug{
				Key:      "B100",
				Title:    "Tooltip misaligned",
				Status:   BugStatus("in_progress"),
				Severity: BugSeverityMedium,
			},
			wantErr: false,
		},
		{
			name: "valid bug - low severity",
			bug: Bug{
				Key:      "B999",
				Title:    "Minor UI inconsistency",
				Status:   BugStatus("completed"),
				Severity: BugSeverityLow,
			},
			wantErr: false,
		},
		{
			name: "empty key",
			bug: Bug{
				Key:      "",
				Title:    "Some bug",
				Status:   BugStatus("reported"),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
			errMsg:  ErrEmptyKey.Error(),
		},
		{
			name: "invalid key - wrong prefix",
			bug: Bug{
				Key:      "A001",
				Title:    "Some bug",
				Status:   BugStatus("reported"),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid key - too few digits",
			bug: Bug{
				Key:      "B01",
				Title:    "Some bug",
				Status:   BugStatus("reported"),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid key - too many digits",
			bug: Bug{
				Key:      "B0001",
				Title:    "Some bug",
				Status:   BugStatus("reported"),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid key - lowercase",
			bug: Bug{
				Key:      "b001",
				Title:    "Some bug",
				Status:   BugStatus("reported"),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "empty title",
			bug: Bug{
				Key:      "B001",
				Title:    "",
				Status:   BugStatus("reported"),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
			errMsg:  ErrEmptyTitle.Error(),
		},
		{
			name: "whitespace title",
			bug: Bug{
				Key:      "B001",
				Title:    "   ",
				Status:   BugStatus("reported"),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
			errMsg:  ErrEmptyTitle.Error(),
		},
		{
			name: "empty status",
			bug: Bug{
				Key:      "B001",
				Title:    "Some bug",
				Status:   BugStatus(""),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
			errMsg:  "bug status cannot be empty",
		},
		{
			name: "whitespace status",
			bug: Bug{
				Key:      "B001",
				Title:    "Some bug",
				Status:   BugStatus("   "),
				Severity: BugSeverityHigh,
			},
			wantErr: true,
			errMsg:  "bug status cannot be empty",
		},
		{
			name: "invalid severity",
			bug: Bug{
				Key:      "B001",
				Title:    "Some bug",
				Status:   BugStatus("reported"),
				Severity: BugSeverity("extreme"),
			},
			wantErr: true,
			errMsg:  `invalid severity "extreme": must be one of critical, high, medium, low`,
		},
		{
			name: "empty severity",
			bug: Bug{
				Key:      "B001",
				Title:    "Some bug",
				Status:   BugStatus("reported"),
				Severity: BugSeverity(""),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bug.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestValidateBugKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		errType error
	}{
		{
			name:    "valid B001",
			key:     "B001",
			wantErr: false,
		},
		{
			name:    "valid B042",
			key:     "B042",
			wantErr: false,
		},
		{
			name:    "valid B999",
			key:     "B999",
			wantErr: false,
		},
		{
			name:    "valid B100",
			key:     "B100",
			wantErr: false,
		},
		{
			name:    "empty key returns ErrEmptyKey",
			key:     "",
			wantErr: true,
			errType: ErrEmptyKey,
		},
		{
			name:    "lowercase b001",
			key:     "b001",
			wantErr: true,
			errType: ErrInvalidBugKey,
		},
		{
			name:    "wrong prefix A001",
			key:     "A001",
			wantErr: true,
			errType: ErrInvalidBugKey,
		},
		{
			name:    "two digits B01",
			key:     "B01",
			wantErr: true,
			errType: ErrInvalidBugKey,
		},
		{
			name:    "four digits B0001",
			key:     "B0001",
			wantErr: true,
			errType: ErrInvalidBugKey,
		},
		{
			name:    "letters instead of digits Babc",
			key:     "Babc",
			wantErr: true,
			errType: ErrInvalidBugKey,
		},
		{
			name:    "just B with no digits",
			key:     "B",
			wantErr: true,
			errType: ErrInvalidBugKey,
		},
		{
			name:    "task key format not allowed",
			key:     "T-E07-F01-001",
			wantErr: true,
			errType: ErrInvalidBugKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBugKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBugKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("ValidateBugKey(%q) error type = %v, want errors.Is(%v)", tt.key, err, tt.errType)
				}
			}
		})
	}
}

func TestBugSeverityConstants(t *testing.T) {
	expected := map[BugSeverity]bool{
		BugSeverityCritical: true,
		BugSeverityHigh:     true,
		BugSeverityMedium:   true,
		BugSeverityLow:      true,
	}

	for severity := range expected {
		if !ValidBugSeverities[severity] {
			t.Errorf("ValidBugSeverities[%q] = false, want true", severity)
		}
	}

	if len(ValidBugSeverities) != 4 {
		t.Errorf("ValidBugSeverities has %d entries, want 4", len(ValidBugSeverities))
	}
}

func TestBugSeverityValues(t *testing.T) {
	if string(BugSeverityCritical) != "critical" {
		t.Errorf("BugSeverityCritical = %q, want %q", BugSeverityCritical, "critical")
	}
	if string(BugSeverityHigh) != "high" {
		t.Errorf("BugSeverityHigh = %q, want %q", BugSeverityHigh, "high")
	}
	if string(BugSeverityMedium) != "medium" {
		t.Errorf("BugSeverityMedium = %q, want %q", BugSeverityMedium, "medium")
	}
	if string(BugSeverityLow) != "low" {
		t.Errorf("BugSeverityLow = %q, want %q", BugSeverityLow, "low")
	}
}

func TestErrInvalidBugKey_Message(t *testing.T) {
	msg := ErrInvalidBugKey.Error()
	if !strings.Contains(msg, "B###") {
		t.Errorf("ErrInvalidBugKey message does not mention B### format: %q", msg)
	}
}
