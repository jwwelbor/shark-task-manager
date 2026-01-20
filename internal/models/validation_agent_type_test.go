package models

import (
	"strings"
	"testing"
)

// TestValidateAgentType_StandardTypes tests that all standard agent types still pass validation
func TestValidateAgentType_StandardTypes(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"standard frontend", "frontend", false},
		{"standard backend", "backend", false},
		{"standard api", "api", false},
		{"standard testing", "testing", false},
		{"standard devops", "devops", false},
		{"standard general", "general", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentType(tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentType(%q) error = %v, wantErr %v", tt.agentType, err, tt.wantErr)
			}
		})
	}
}

// TestValidateAgentType_CustomTypes tests that custom agent types are accepted
func TestValidateAgentType_CustomTypes(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"custom architect", "architect", false},
		{"custom business-analyst", "business-analyst", false},
		{"custom qa", "qa", false},
		{"custom tech-lead", "tech-lead", false},
		{"custom product-manager", "product-manager", false},
		{"custom ux-designer", "ux-designer", false},
		{"custom data-engineer", "data-engineer", false},
		{"custom ml-specialist", "ml-specialist", false},
		{"custom security-auditor", "security-auditor", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentType(tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentType(%q) error = %v, wantErr %v", tt.agentType, err, tt.wantErr)
			}
		})
	}
}

// TestValidateAgentType_EdgeCases tests edge cases for agent type validation
func TestValidateAgentType_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		// Hyphenated agent types
		{"hyphenated data-engineer", "data-engineer", false},
		{"hyphenated ml-specialist", "ml-specialist", false},
		{"hyphenated security-auditor", "security-auditor", false},

		// Underscore-separated agent types
		{"underscore qa_engineer", "qa_engineer", false},
		{"underscore tech_lead", "tech_lead", false},
		{"underscore product_manager", "product_manager", false},

		// Mixed case agent types
		{"mixed case DataEngineer", "DataEngineer", false},
		{"mixed case BusinessAnalyst", "BusinessAnalyst", false},
		{"mixed case QA-Lead", "QA-Lead", false},

		// Single-character agent types
		{"single char x", "x", false},
		{"single char a", "a", false},

		// Very long agent type names
		{"long name", "very-long-custom-agent-type-name-for-edge-case-testing", false},

		// Special characters in agent types
		{"with dot agent.dev", "agent.dev", false},
		{"with underscore and dash qa_engineer-lead", "qa_engineer-lead", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentType(tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentType(%q) error = %v, wantErr %v", tt.agentType, err, tt.wantErr)
			}
		})
	}
}

// TestValidateAgentType_InvalidCases tests invalid cases that should fail validation
func TestValidateAgentType_InvalidCases(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"tab whitespace", "\t\t", true},
		{"newline whitespace", "\n\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentType(tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentType(%q) error = %v, wantErr %v", tt.agentType, err, tt.wantErr)
			}
		})
	}
}

// TestValidateAgentType_ErrorMessages tests that error messages are clear and actionable
func TestValidateAgentType_ErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		agentType      string
		wantErr        bool
		errorContains  string
	}{
		{"empty string error", "", true, "empty"},
		{"whitespace only error", "   ", true, "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentType(tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentType(%q) error = %v, wantErr %v", tt.agentType, err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorContains != "" {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("ValidateAgentType(%q) error message %q does not contain %q", tt.agentType, err.Error(), tt.errorContains)
				}
			}
		})
	}
}

// TestValidateAgentType_BackwardCompatibility ensures existing code still works
func TestValidateAgentType_BackwardCompatibility(t *testing.T) {
	// This test ensures that the validation changes are fully backward compatible
	// All existing tests that use standard agent types should continue to work

	standardTypes := []string{"frontend", "backend", "api", "testing", "devops", "general"}

	for _, agentType := range standardTypes {
		err := ValidateAgentType(agentType)
		if err != nil {
			t.Errorf("ValidateAgentType(%q) error = %v, expected no error (backward compatibility)", agentType, err)
		}
	}
}

// BenchmarkValidateAgentType_Standard benchmarks validation of standard agent types
func BenchmarkValidateAgentType_Standard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ValidateAgentType("backend")
	}
}

// BenchmarkValidateAgentType_Custom benchmarks validation of custom agent types
func BenchmarkValidateAgentType_Custom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ValidateAgentType("architect")
	}
}

// BenchmarkValidateAgentType_LongName benchmarks validation of very long agent type names
func BenchmarkValidateAgentType_LongName(b *testing.B) {
	longName := "very-long-custom-agent-type-name-for-edge-case-testing"
	for i := 0; i < b.N; i++ {
		_ = ValidateAgentType(longName)
	}
}
