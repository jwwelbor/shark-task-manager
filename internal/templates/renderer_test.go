package templates

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderer_Render_ConsolidatedTaskTemplate(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E01-F02-001",
		Title:       "Build Login Component",
		Description: "Create a reusable login component",
		Epic:        "E01",
		Feature:     "E01-F02",
		AgentType:   "frontend",
		Priority:    5,
		DependsOn:   []string{"T-E01-F01-001"},
		CreatedAt:   time.Date(2025, 12, 14, 10, 30, 0, 0, time.UTC),
	}

	result, err := renderer.Render("frontend", data)

	require.NoError(t, err)
	assert.Contains(t, result, "key: T-E01-F02-001")
	assert.Contains(t, result, "title: Build Login Component")
	assert.Contains(t, result, "epic: E01")
	assert.Contains(t, result, "feature: E01-F02")
	assert.Contains(t, result, "agent: frontend")
	assert.Contains(t, result, "priority: 5")
	assert.Contains(t, result, `depends_on: ["T-E01-F01-001"]`)
	assert.Contains(t, result, "2025-12-14T10:30:00Z")
	assert.Contains(t, result, "# Task: Build Login Component")
	assert.Contains(t, result, "Create a reusable login component")
	assert.Contains(t, result, "## Requirements")
	assert.Contains(t, result, "## Implementation Plan")
	assert.Contains(t, result, "## Deliverables")
	assert.Contains(t, result, "## Acceptance Criteria")
}

func TestRenderer_Render_ConsolidatedTaskTemplateForAnyAgent(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	for _, agentType := range []string{"backend", "api", "testing", "devops", "general"} {
		t.Run(agentType, func(t *testing.T) {
			data := TemplateData{
				Key:       "T-E01-F03-002",
				Title:     "Implement User Service",
				Epic:      "E01",
				Feature:   "E01-F03",
				AgentType: agentType,
				Priority:  7,
				CreatedAt: time.Date(2025, 12, 14, 11, 0, 0, 0, time.UTC),
			}

			result, err := renderer.Render(agentType, data)

			require.NoError(t, err)
			assert.Contains(t, result, "agent: "+agentType)
			assert.Contains(t, result, "## Requirements")
			assert.Contains(t, result, "## Implementation Plan")
			assert.Contains(t, result, "## Deliverables")
			assert.NotContains(t, result, "depends_on:")
		})
	}
}

func TestRenderer_Render_EmptyDescription(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E01-F02-010",
		Title:       "Task Without Description",
		Description: "",
		Epic:        "E01",
		Feature:     "E01-F02",
		AgentType:   "general",
		Priority:    5,
		DependsOn:   []string{},
		CreatedAt:   time.Now().UTC(),
	}

	result, err := renderer.Render("general", data)

	require.NoError(t, err)
	assert.Contains(t, result, "[Describe what needs to be accomplished]")
	assert.NotContains(t, result, "Description: \n")
}

func TestRenderer_Render_MultipleDependencies(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:       "T-E01-F02-020",
		Title:     "Integration Task",
		Epic:      "E01",
		Feature:   "E01-F02",
		AgentType: "backend",
		Priority:  5,
		DependsOn: []string{"T-E01-F02-001", "T-E01-F02-002", "T-E01-F02-003"},
		CreatedAt: time.Now().UTC(),
	}

	result, err := renderer.Render("backend", data)

	require.NoError(t, err)
	assert.Contains(t, result, `depends_on: ["T-E01-F02-001", "T-E01-F02-002", "T-E01-F02-003"]`)
}

func TestRenderer_Render_FrontmatterValidYAML(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E01-F01-001",
		Title:       "Test Task",
		Description: "Test Description",
		Epic:        "E01",
		Feature:     "E01-F01",
		AgentType:   "general",
		Priority:    5,
		DependsOn:   []string{"T-E01-F01-000"},
		CreatedAt:   time.Date(2025, 12, 14, 10, 30, 0, 0, time.UTC),
	}

	result, err := renderer.Render("general", data)
	require.NoError(t, err)

	// Extract frontmatter (between --- markers)
	parts := strings.Split(result, "---")
	require.GreaterOrEqual(t, len(parts), 3, "Should have frontmatter delimited by ---")

	frontmatter := parts[1]

	// Check required fields are present
	assert.Contains(t, frontmatter, "key: T-E01-F01-001")
	assert.Contains(t, frontmatter, "title: Test Task")
	assert.Contains(t, frontmatter, "epic: E01")
	assert.Contains(t, frontmatter, "feature: E01-F01")
	assert.Contains(t, frontmatter, "agent: general")
	assert.Contains(t, frontmatter, "priority: 5")
	assert.Contains(t, frontmatter, `depends_on: ["T-E01-F01-000"]`)
	assert.Contains(t, frontmatter, "created_at: 2025-12-14T10:30:00Z")
}

func TestRenderer_Render_UnknownAgentTypeUsesConsolidatedTemplate(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:       "T-E01-F01-001",
		Title:     "Test Task",
		Epic:      "E01",
		Feature:   "E01-F01",
		AgentType: "code-reviewer",
		Priority:  5,
		CreatedAt: time.Now().UTC(),
	}

	result, err := renderer.Render("code-reviewer", data)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "agent: code-reviewer")
	assert.Contains(t, result, "## Requirements")
	assert.Contains(t, result, "## Implementation Plan")
	assert.Contains(t, result, "## Deliverables")
}

func TestRenderer_Render_CustomAgentTypeWithHyphens(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:       "T-E01-F01-002",
		Title:     "Custom Agent Task",
		Epic:      "E01",
		Feature:   "E01-F01",
		AgentType: "ui-designer",
		Priority:  5,
		CreatedAt: time.Now().UTC(),
	}

	result, err := renderer.Render("ui-designer", data)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "agent: ui-designer")
}

func TestRenderer_Render_CustomAgentTypeWithUnderscores(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:       "T-E01-F01-003",
		Title:     "Database Architect Task",
		Epic:      "E01",
		Feature:   "E01-F01",
		AgentType: "database_architect",
		Priority:  5,
		CreatedAt: time.Now().UTC(),
	}

	result, err := renderer.Render("database_architect", data)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "agent: database_architect")
}

func TestRenderer_Render_EmptyAgentTypeUsesConsolidatedTemplate(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:       "T-E01-F01-004",
		Title:     "Empty Agent Type Task",
		Epic:      "E01",
		Feature:   "E01-F01",
		AgentType: "",
		Priority:  5,
		CreatedAt: time.Now().UTC(),
	}

	result, err := renderer.Render("", data)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "## Requirements")
}

func TestRenderer_Render_CustomAgentTypeUsesConsolidatedTemplate(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:       "T-E01-F01-005",
		Title:     "Custom Architect Task",
		Epic:      "E01",
		Feature:   "E01-F01",
		AgentType: "architect",
		Priority:  5,
		CreatedAt: time.Now().UTC(),
	}

	result, err := renderer.Render("architect", data)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "agent: architect")
	assert.Contains(t, result, "## Requirements")
}

func TestTemplateFuncs_Join(t *testing.T) {
	funcs := templateFuncs()
	joinFunc := funcs["join"].(func([]string, string) string)

	result := joinFunc([]string{"a", "b", "c"}, ", ")
	assert.Equal(t, "a, b, c", result)
}

func TestTemplateFuncs_Quote(t *testing.T) {
	funcs := templateFuncs()
	quoteFunc := funcs["quote"].(func([]string) []string)

	result := quoteFunc([]string{"task1", "task2"})
	assert.Equal(t, []string{`"task1"`, `"task2"`}, result)
}

func TestTemplateFuncs_IsEmpty(t *testing.T) {
	funcs := templateFuncs()
	isEmptyFunc := funcs["isEmpty"].(func(string) bool)

	assert.True(t, isEmptyFunc(""))
	assert.True(t, isEmptyFunc("   "))
	assert.False(t, isEmptyFunc("text"))
	assert.False(t, isEmptyFunc("  text  "))
}

func TestTemplateFuncs_FormatTime(t *testing.T) {
	funcs := templateFuncs()
	formatTimeFunc := funcs["formatTime"].(func(time.Time) string)

	testTime := time.Date(2025, 12, 14, 10, 30, 0, 0, time.UTC)
	result := formatTimeFunc(testTime)
	assert.Equal(t, "2025-12-14T10:30:00Z", result)
}

func TestTemplateFuncs_FormatDate(t *testing.T) {
	funcs := templateFuncs()
	formatDateFunc := funcs["formatDate"].(func(time.Time) string)

	testTime := time.Date(2025, 12, 14, 10, 30, 0, 0, time.UTC)
	result := formatDateFunc(testTime)
	assert.Equal(t, "2025-12-14", result)
}
