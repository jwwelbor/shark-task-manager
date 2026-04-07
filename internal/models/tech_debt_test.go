package models

import (
	"errors"
	"strings"
	"testing"
)

func TestTechDebt_Validate(t *testing.T) {
	tests := []struct {
		name    string
		td      TechDebt
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tech-debt - code-quality critical",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: "Refactor auth module"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityCritical,
			},
			wantErr: false,
		},
		{
			name: "valid tech-debt - architecture high",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-042", Title: "Decouple service layer"},
				Status:     TechDebtStatus("triaged"),
				Category:   TechDebtCategoryArchitecture,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: false,
		},
		{
			name: "valid tech-debt - dependency medium",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-100", Title: "Update deprecated library"},
				Status:     TechDebtStatus("in_progress"),
				Category:   TechDebtCategoryDependency,
				Severity:   TechDebtSeverityMedium,
			},
			wantErr: false,
		},
		{
			name: "valid tech-debt - testing low",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-999", Title: "Add missing unit tests"},
				Status:     TechDebtStatus("resolved"),
				Category:   TechDebtCategoryTesting,
				Severity:   TechDebtSeverityLow,
			},
			wantErr: false,
		},
		{
			name: "valid tech-debt - performance",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-050", Title: "Optimize N+1 queries"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryPerformance,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: false,
		},
		{
			name: "valid tech-debt - documentation",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-051", Title: "Document API endpoints"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryDocumentation,
				Severity:   TechDebtSeverityLow,
			},
			wantErr: false,
		},
		{
			name: "valid tech-debt - with effort estimate",
			td: TechDebt{
				BaseEntity:     BaseEntity{Key: "TD-002", Title: "Fix code smells"},
				Status:         TechDebtStatus("triaged"),
				Category:       TechDebtCategoryCodeQuality,
				Severity:       TechDebtSeverityMedium,
				EffortEstimate: strPtr("2 days"),
			},
			wantErr: false,
		},
		{
			name: "empty key",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
			errMsg:  ErrEmptyKey.Error(),
		},
		{
			name: "invalid key - wrong prefix",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "B001", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid key - too few digits",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-01", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid key - too many digits",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-0001", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid key - lowercase",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "td-001", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid key - missing hyphen",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD001", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "empty title",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: ""},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
			errMsg:  ErrEmptyTitle.Error(),
		},
		{
			name: "whitespace title",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: "   "},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
			errMsg:  ErrEmptyTitle.Error(),
		},
		{
			name: "empty status",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: "Some debt"},
				Status:     TechDebtStatus(""),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
			errMsg:  "tech-debt status cannot be empty",
		},
		{
			name: "whitespace status",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: "Some debt"},
				Status:     TechDebtStatus("   "),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
			errMsg:  "tech-debt status cannot be empty",
		},
		{
			name: "invalid category",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategory("invalid"),
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
			errMsg:  `invalid category "invalid": must be one of code-quality, architecture, dependency, testing, performance, documentation`,
		},
		{
			name: "empty category",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategory(""),
				Severity:   TechDebtSeverityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid severity",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverity("extreme"),
			},
			wantErr: true,
			errMsg:  `invalid severity "extreme": must be one of critical, high, medium, low`,
		},
		{
			name: "empty severity",
			td: TechDebt{
				BaseEntity: BaseEntity{Key: "TD-001", Title: "Some debt"},
				Status:     TechDebtStatus("identified"),
				Category:   TechDebtCategoryCodeQuality,
				Severity:   TechDebtSeverity(""),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.td.Validate()
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

func TestValidateTechDebtKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		errType error
	}{
		{
			name:    "valid TD-001",
			key:     "TD-001",
			wantErr: false,
		},
		{
			name:    "valid TD-042",
			key:     "TD-042",
			wantErr: false,
		},
		{
			name:    "valid TD-999",
			key:     "TD-999",
			wantErr: false,
		},
		{
			name:    "valid TD-100",
			key:     "TD-100",
			wantErr: false,
		},
		{
			name:    "empty key returns ErrEmptyKey",
			key:     "",
			wantErr: true,
			errType: ErrEmptyKey,
		},
		{
			name:    "lowercase td-001",
			key:     "td-001",
			wantErr: true,
			errType: ErrInvalidTechDebtKey,
		},
		{
			name:    "wrong prefix B001",
			key:     "B001",
			wantErr: true,
			errType: ErrInvalidTechDebtKey,
		},
		{
			name:    "two digits TD-01",
			key:     "TD-01",
			wantErr: true,
			errType: ErrInvalidTechDebtKey,
		},
		{
			name:    "four digits TD-0001",
			key:     "TD-0001",
			wantErr: true,
			errType: ErrInvalidTechDebtKey,
		},
		{
			name:    "letters instead of digits TD-abc",
			key:     "TD-abc",
			wantErr: true,
			errType: ErrInvalidTechDebtKey,
		},
		{
			name:    "missing hyphen TD001",
			key:     "TD001",
			wantErr: true,
			errType: ErrInvalidTechDebtKey,
		},
		{
			name:    "just TD with no digits",
			key:     "TD-",
			wantErr: true,
			errType: ErrInvalidTechDebtKey,
		},
		{
			name:    "task key format not allowed",
			key:     "T-E07-F01-001",
			wantErr: true,
			errType: ErrInvalidTechDebtKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTechDebtKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTechDebtKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("ValidateTechDebtKey(%q) error type = %v, want errors.Is(%v)", tt.key, err, tt.errType)
				}
			}
		})
	}
}

func TestTechDebtCategoryConstants(t *testing.T) {
	expected := map[TechDebtCategory]bool{
		TechDebtCategoryCodeQuality:   true,
		TechDebtCategoryArchitecture:  true,
		TechDebtCategoryDependency:    true,
		TechDebtCategoryTesting:       true,
		TechDebtCategoryPerformance:   true,
		TechDebtCategoryDocumentation: true,
	}

	for category := range expected {
		if !ValidTechDebtCategories[category] {
			t.Errorf("ValidTechDebtCategories[%q] = false, want true", category)
		}
	}

	if len(ValidTechDebtCategories) != 6 {
		t.Errorf("ValidTechDebtCategories has %d entries, want 6", len(ValidTechDebtCategories))
	}
}

func TestTechDebtCategoryValues(t *testing.T) {
	if string(TechDebtCategoryCodeQuality) != "code-quality" {
		t.Errorf("TechDebtCategoryCodeQuality = %q, want %q", TechDebtCategoryCodeQuality, "code-quality")
	}
	if string(TechDebtCategoryArchitecture) != "architecture" {
		t.Errorf("TechDebtCategoryArchitecture = %q, want %q", TechDebtCategoryArchitecture, "architecture")
	}
	if string(TechDebtCategoryDependency) != "dependency" {
		t.Errorf("TechDebtCategoryDependency = %q, want %q", TechDebtCategoryDependency, "dependency")
	}
	if string(TechDebtCategoryTesting) != "testing" {
		t.Errorf("TechDebtCategoryTesting = %q, want %q", TechDebtCategoryTesting, "testing")
	}
	if string(TechDebtCategoryPerformance) != "performance" {
		t.Errorf("TechDebtCategoryPerformance = %q, want %q", TechDebtCategoryPerformance, "performance")
	}
	if string(TechDebtCategoryDocumentation) != "documentation" {
		t.Errorf("TechDebtCategoryDocumentation = %q, want %q", TechDebtCategoryDocumentation, "documentation")
	}
}

func TestTechDebtSeverityConstants(t *testing.T) {
	expected := map[TechDebtSeverity]bool{
		TechDebtSeverityCritical: true,
		TechDebtSeverityHigh:     true,
		TechDebtSeverityMedium:   true,
		TechDebtSeverityLow:      true,
	}

	for severity := range expected {
		if !ValidTechDebtSeverities[severity] {
			t.Errorf("ValidTechDebtSeverities[%q] = false, want true", severity)
		}
	}

	if len(ValidTechDebtSeverities) != 4 {
		t.Errorf("ValidTechDebtSeverities has %d entries, want 4", len(ValidTechDebtSeverities))
	}
}

func TestTechDebtSeverityValues(t *testing.T) {
	if string(TechDebtSeverityCritical) != "critical" {
		t.Errorf("TechDebtSeverityCritical = %q, want %q", TechDebtSeverityCritical, "critical")
	}
	if string(TechDebtSeverityHigh) != "high" {
		t.Errorf("TechDebtSeverityHigh = %q, want %q", TechDebtSeverityHigh, "high")
	}
	if string(TechDebtSeverityMedium) != "medium" {
		t.Errorf("TechDebtSeverityMedium = %q, want %q", TechDebtSeverityMedium, "medium")
	}
	if string(TechDebtSeverityLow) != "low" {
		t.Errorf("TechDebtSeverityLow = %q, want %q", TechDebtSeverityLow, "low")
	}
}

func TestErrInvalidTechDebtKey_Message(t *testing.T) {
	msg := ErrInvalidTechDebtKey.Error()
	if !strings.Contains(msg, "TD-###") {
		t.Errorf("ErrInvalidTechDebtKey message does not mention TD-### format: %q", msg)
	}
}

func TestTechDebt_EntityInterface(t *testing.T) {
	td := &TechDebt{
		BaseEntity: BaseEntity{Key: "TD-001", Title: "Test debt"},
		Status:     TechDebtStatus("identified"),
		Category:   TechDebtCategoryCodeQuality,
		Severity:   TechDebtSeverityMedium,
	}

	t.Run("GetEntityType", func(t *testing.T) {
		if got := td.GetEntityType(); got != EntityTypeTechDebt {
			t.Errorf("GetEntityType() = %q, want %q", got, EntityTypeTechDebt)
		}
	})

	t.Run("GetStatus", func(t *testing.T) {
		if got := td.GetStatus(); got != "identified" {
			t.Errorf("GetStatus() = %q, want %q", got, "identified")
		}
	})

	t.Run("SetStatus", func(t *testing.T) {
		td.SetStatus("triaged")
		if got := td.GetStatus(); got != "triaged" {
			t.Errorf("GetStatus() after SetStatus = %q, want %q", got, "triaged")
		}
	})
}

func TestTechDebt_EntityTypeTechDebt_Registered(t *testing.T) {
	if !ValidEntityTypes[EntityTypeTechDebt] {
		t.Error("EntityTypeTechDebt not registered in ValidEntityTypes")
	}
}

func TestTechDebt_EntityTypeTechDebt_Value(t *testing.T) {
	if string(EntityTypeTechDebt) != "tech_debt" {
		t.Errorf("EntityTypeTechDebt = %q, want %q", EntityTypeTechDebt, "tech_debt")
	}
}

// strPtr is a helper to get a pointer to a string value.
func strPtr(s string) *string {
	return &s
}
