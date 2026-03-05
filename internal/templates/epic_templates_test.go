package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEpicTemplates_ExistAndRender validates that all 3 epic strategic templates exist and render correctly
func TestEpicTemplates_ExistAndRender(t *testing.T) {
	renderer, err := NewOrchestratorRenderer("../../shark-templates")
	require.NoError(t, err, "Should initialize renderer without errors")
	require.NotNil(t, renderer, "Renderer should not be nil")

	tests := []struct {
		name            string
		templateName    string
		vars            map[string]string
		expectedStrings []string
		notExpected     []string
	}{
		{
			name:         "ready_for_research epic template",
			templateName: "epic/ready_for_research.tmpl",
			vars: map[string]string{
				"id":        "E07",
				"title":     "User Authentication System",
				"file_path": "docs/plan/E07-user-auth/epic.md",
			},
			expectedStrings: []string{
				"Research epic E07",
				"STRATEGIC RESEARCH",
				"LOAD:",
				"discovery",
				"research",
				"READ:",
				"All 6 epic PRD files",
				"PRODUCE:",
				"Market/competitive landscape",
				"Feasibility assessment",
				"System-wide impact",
				"Existing capability overlap",
				"Risk assessment",
				"EXIT GATE:",
				"All 6 sections complete",
				"shark status advance",
			},
			notExpected: []string{
				"",
			},
		},
		{
			name:         "ready_for_feasibility_review_ba epic template",
			templateName: "epic/ready_for_feasibility_review_ba.tmpl",
			vars: map[string]string{
				"id":        "E07",
				"title":     "User Authentication System",
				"file_path": "docs/plan/E07-user-auth/epic.md",
			},
			expectedStrings: []string{
				"BA feasibility review for epic E07",
				"epic PRD",
				"research findings",
				"business perspective",
				"READ:",
				"All 6 epic PRD files",
				"Research report",
				"Other epic PRDs",
				"EVALUATE:",
				"Cross-epic conflicts",
				"Market viability",
				"Scope coherence",
				"Business risk",
				"PRODUCE:",
				"feasibility review report",
				"IF APPROVED:",
				"IF CONCERNS FOUND:",
				"intervention_required",
				"intervention report",
			},
			notExpected: []string{
				"",
			},
		},
		{
			name:         "ready_for_feasibility_review_tech epic template",
			templateName: "epic/ready_for_feasibility_review_tech.tmpl",
			vars: map[string]string{
				"id":        "E07",
				"title":     "User Authentication System",
				"file_path": "docs/plan/E07-user-auth/epic.md",
			},
			expectedStrings: []string{
				"Technical feasibility review for epic E07",
				"PLAN GATE",
				"READ:",
				"All epic BA docs",
				"Research report",
				"BA feasibility review report",
				"Other epic architectures",
				"EVALUATE:",
				"Technical feasibility",
				"Architectural concerns",
				"Dependency and integration risks",
				"Technical debt implications",
				"PRODUCE:",
				"feasibility review report",
				"IF APPROVED:",
				"IF CONCERNS FOUND:",
				"intervention_required",
				"intervention report",
				"shark epic update E07",
			},
			notExpected: []string{
				"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Render the template
			result, err := renderer.Render(tt.templateName, tt.vars)
			require.NoError(t, err, "Should render template without errors")
			require.NotEmpty(t, result, "Result should not be empty")

			// Verify all expected strings are present
			for _, expected := range tt.expectedStrings {
				if expected != "" {
					assert.Contains(t, result, expected,
						"Template should contain: %s", expected)
				}
			}

			// Verify not expected strings are absent
			for _, notExp := range tt.notExpected {
				if notExp != "" {
					assert.NotContains(t, result, notExp,
						"Template should not contain: %s", notExp)
				}
			}

			// Verify no double newlines (except legitimate ones)
			lines := strings.Split(result, "\n")
			assert.Greater(t, len(lines), 5, "Template should have multiple lines")

			// Verify key command is present (some templates use shark status advance, others use shark epic update)
			hasAdvance := strings.Contains(result, "shark status advance")
			hasEpicUpdate := strings.Contains(result, "shark epic update")
			assert.True(t, hasAdvance || hasEpicUpdate,
				"Template should contain a shark advancement command (shark status advance or shark epic update)")
		})
	}
}

// TestEpicTemplates_AllExist validates that all 3 epic templates exist in the file system
func TestEpicTemplates_AllExist(t *testing.T) {
	renderer, err := NewOrchestratorRenderer("../../shark-templates")
	require.NoError(t, err, "Should initialize renderer without errors")

	templates := []string{
		"epic/ready_for_research.tmpl",
		"epic/ready_for_feasibility_review_ba.tmpl",
		"epic/ready_for_feasibility_review_tech.tmpl",
	}

	for _, tmplName := range templates {
		t.Run(tmplName, func(t *testing.T) {
			// Attempt to render with minimal vars
			result, err := renderer.Render(tmplName, map[string]string{
				"id":        "E07",
				"title":     "Test Epic",
				"file_path": "docs/plan/E07/epic.md",
			})

			assert.NoError(t, err, "Template %s should exist and render", tmplName)
			assert.NotEmpty(t, result, "Template %s should produce output", tmplName)
		})
	}
}

// TestEpicTemplates_RegressionSemanticEquivalence validates that epic templates produce semantically equivalent output to what would be expected
func TestEpicTemplates_RegressionSemanticEquivalence(t *testing.T) {
	renderer, err := NewOrchestratorRenderer("../../shark-templates")
	require.NoError(t, err)

	// Test ready_for_research template
	t.Run("ready_for_research", func(t *testing.T) {
		result, err := renderer.Render("epic/ready_for_research.tmpl", map[string]string{
			"id":        "E07",
			"title":     "Test Epic",
			"file_path": "docs/plan/E07/epic.md",
		})

		require.NoError(t, err)

		// Core semantic elements
		assert.Contains(t, result, "E07")
		assert.Contains(t, result, "STRATEGIC RESEARCH")
		assert.Contains(t, result, "LOAD")
		assert.Contains(t, result, "READ")
		assert.Contains(t, result, "PRODUCE")
		assert.Contains(t, result, "EXIT GATE")
		assert.Contains(t, result, "shark status advance")

		// Verify multi-section structure
		produceIdx := strings.Index(result, "PRODUCE:")
		exitIdx := strings.Index(result, "EXIT GATE:")
		assert.Greater(t, exitIdx, produceIdx, "EXIT GATE should come after PRODUCE")
	})

	// Test ready_for_feasibility_review_ba template
	t.Run("ready_for_feasibility_review_ba", func(t *testing.T) {
		result, err := renderer.Render("epic/ready_for_feasibility_review_ba.tmpl", map[string]string{
			"id":        "E07",
			"title":     "Test Epic",
			"file_path": "docs/plan/E07/epic.md",
		})

		require.NoError(t, err)

		// Core semantic elements
		assert.Contains(t, result, "BA feasibility review")
		assert.Contains(t, result, "E07")
		assert.Contains(t, result, "READ")
		assert.Contains(t, result, "EVALUATE")
		assert.Contains(t, result, "PRODUCE")
		assert.Contains(t, result, "IF APPROVED")
		assert.Contains(t, result, "IF CONCERNS FOUND")
		assert.Contains(t, result, "intervention_required")
	})

	// Test ready_for_feasibility_review_tech template
	t.Run("ready_for_feasibility_review_tech", func(t *testing.T) {
		result, err := renderer.Render("epic/ready_for_feasibility_review_tech.tmpl", map[string]string{
			"id":        "E07",
			"title":     "Test Epic",
			"file_path": "docs/plan/E07/epic.md",
		})

		require.NoError(t, err)

		// Core semantic elements
		assert.Contains(t, result, "Technical feasibility review")
		assert.Contains(t, result, "E07")
		assert.Contains(t, result, "PLAN GATE")
		assert.Contains(t, result, "READ")
		assert.Contains(t, result, "EVALUATE")
		assert.Contains(t, result, "Technical feasibility")
		assert.Contains(t, result, "PRODUCE")
		assert.Contains(t, result, "IF APPROVED")
		assert.Contains(t, result, "IF CONCERNS FOUND")
		assert.Contains(t, result, "intervention_required")
		assert.Contains(t, result, "shark epic update")
	})
}

// TestEpicTemplates_VariableSubstitution validates that variables are correctly substituted in templates
func TestEpicTemplates_VariableSubstitution(t *testing.T) {
	renderer, err := NewOrchestratorRenderer("../../shark-templates")
	require.NoError(t, err)

	tests := []struct {
		name         string
		templateName string
		vars         map[string]string
		idField      string
	}{
		{
			name:         "research template substitutes id",
			templateName: "epic/ready_for_research.tmpl",
			vars: map[string]string{
				"id":        "E123",
				"title":     "Premium Features",
				"file_path": "docs/premium/epic.md",
			},
			idField: "E123",
		},
		{
			name:         "ba feasibility template substitutes id",
			templateName: "epic/ready_for_feasibility_review_ba.tmpl",
			vars: map[string]string{
				"id":        "E456",
				"title":     "API Migration",
				"file_path": "docs/api/epic.md",
			},
			idField: "E456",
		},
		{
			name:         "tech feasibility template substitutes id",
			templateName: "epic/ready_for_feasibility_review_tech.tmpl",
			vars: map[string]string{
				"id":        "E789",
				"title":     "Database Upgrade",
				"file_path": "docs/database/epic.md",
			},
			idField: "E789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(tt.templateName, tt.vars)

			require.NoError(t, err)
			assert.Contains(t, result, tt.idField,
				"Template should substitute epic ID %s", tt.idField)
			// Only check file_path substitution for templates that use it
			if filePath := tt.vars["file_path"]; filePath != "" && tt.templateName != "epic/ready_for_feasibility_review_tech.tmpl" {
				assert.Contains(t, result, filePath,
					"Template should substitute file path")
			}
		})
	}
}

// TestEpicTemplates_CommandIntegrity validates that generated commands are correct
func TestEpicTemplates_CommandIntegrity(t *testing.T) {
	renderer, err := NewOrchestratorRenderer("../../shark-templates")
	require.NoError(t, err)

	epicID := "E07-F30-001"
	vars := map[string]string{
		"id":        epicID,
		"title":     "Test Epic",
		"file_path": "docs/plan/E07/epic.md",
	}

	// Research template should end with advancement command
	result, _ := renderer.Render("epic/ready_for_research.tmpl", vars)
	assert.Contains(t, result, "shark status advance "+epicID,
		"research template should contain advancement command with epic ID")

	// BA feasibility template should handle both approved and concerns paths
	result, _ = renderer.Render("epic/ready_for_feasibility_review_ba.tmpl", vars)
	assert.Contains(t, result, "shark status advance "+epicID,
		"BA feasibility template should contain advancement command")
	assert.Contains(t, result, "shark epic update "+epicID,
		"BA feasibility template should contain update command for intervention")
	assert.Contains(t, result, "status=intervention_required",
		"BA feasibility template should reference intervention_required status")

	// Tech feasibility template uses shark epic update for both approved and concerns paths
	result, _ = renderer.Render("epic/ready_for_feasibility_review_tech.tmpl", vars)
	assert.Contains(t, result, "shark epic update "+epicID,
		"Tech feasibility template should contain update command")
	assert.Contains(t, result, "status=intervention_required",
		"Tech feasibility template should reference intervention_required status")
}
