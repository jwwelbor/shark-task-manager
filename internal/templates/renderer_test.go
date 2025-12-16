package templates

import (
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderer_Render_Frontend(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E01-F02-001",
		Title:       "Build Login Component",
		Description: "Create a reusable login component",
		Epic:        "E01",
		Feature:     "E01-F02",
		AgentType:   models.AgentTypeFrontend,
		Priority:    5,
		DependsOn:   []string{"T-E01-F01-001"},
		CreatedAt:   time.Date(2025, 12, 14, 10, 30, 0, 0, time.UTC),
	}

	result, err := renderer.Render(models.AgentTypeFrontend, data)

	require.NoError(t, err)
	assert.Contains(t, result, "key: T-E01-F02-001")
	assert.Contains(t, result, "title: Build Login Component")
	assert.Contains(t, result, "epic: E01")
	assert.Contains(t, result, "feature: E01-F02")
	assert.Contains(t, result, "agent: frontend")
	assert.Contains(t, result, "status: todo")
	assert.Contains(t, result, "priority: 5")
	assert.Contains(t, result, `depends_on: ["T-E01-F01-001"]`)
	assert.Contains(t, result, "2025-12-14T10:30:00Z")
	assert.Contains(t, result, "# Task: Build Login Component")
	assert.Contains(t, result, "Create a reusable login component")
	assert.Contains(t, result, "## Component Specifications")
	assert.Contains(t, result, "## Acceptance Criteria")
}

func TestRenderer_Render_Backend(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E01-F03-002",
		Title:       "Implement User Service",
		Description: "Create user service with CRUD operations",
		Epic:        "E01",
		Feature:     "E01-F03",
		AgentType:   models.AgentTypeBackend,
		Priority:    7,
		DependsOn:   []string{},
		CreatedAt:   time.Date(2025, 12, 14, 11, 0, 0, 0, time.UTC),
	}

	result, err := renderer.Render(models.AgentTypeBackend, data)

	require.NoError(t, err)
	assert.Contains(t, result, "agent: backend")
	assert.Contains(t, result, "## API Endpoints")
	assert.Contains(t, result, "## Data Models")
	assert.Contains(t, result, "## Business Logic")
	assert.NotContains(t, result, "depends_on:")
}

func TestRenderer_Render_API(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E02-F01-001",
		Title:       "Create User API Endpoint",
		Description: "Implement POST /api/v1/users endpoint",
		Epic:        "E02",
		Feature:     "E02-F01",
		AgentType:   models.AgentTypeAPI,
		Priority:    8,
		DependsOn:   []string{"T-E02-F01-000"},
		CreatedAt:   time.Now().UTC(),
	}

	result, err := renderer.Render(models.AgentTypeAPI, data)

	require.NoError(t, err)
	assert.Contains(t, result, "agent: api")
	assert.Contains(t, result, "## API Specification")
	assert.Contains(t, result, "### Request Schema")
	assert.Contains(t, result, "### Response Schema")
	assert.Contains(t, result, "## Authentication & Authorization")
}

func TestRenderer_Render_Testing(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E03-F02-005",
		Title:       "Test Authentication Flow",
		Description: "Write comprehensive tests for auth",
		Epic:        "E03",
		Feature:     "E03-F02",
		AgentType:   models.AgentTypeTesting,
		Priority:    6,
		DependsOn:   []string{"T-E03-F02-001", "T-E03-F02-002"},
		CreatedAt:   time.Now().UTC(),
	}

	result, err := renderer.Render(models.AgentTypeTesting, data)

	require.NoError(t, err)
	assert.Contains(t, result, "agent: testing")
	assert.Contains(t, result, "## Test Scenarios")
	assert.Contains(t, result, "## Coverage Requirements")
	assert.Contains(t, result, "## Performance Benchmarks")
}

func TestRenderer_Render_DevOps(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E04-F01-003",
		Title:       "Setup CI/CD Pipeline",
		Description: "Configure automated deployment",
		Epic:        "E04",
		Feature:     "E04-F01",
		AgentType:   models.AgentTypeDevOps,
		Priority:    9,
		DependsOn:   []string{},
		CreatedAt:   time.Now().UTC(),
	}

	result, err := renderer.Render(models.AgentTypeDevOps, data)

	require.NoError(t, err)
	assert.Contains(t, result, "agent: devops")
	assert.Contains(t, result, "## Infrastructure Requirements")
	assert.Contains(t, result, "## Deployment Configuration")
	assert.Contains(t, result, "## Monitoring & Observability")
}

func TestRenderer_Render_General(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:         "T-E05-F01-001",
		Title:       "Research Technical Options",
		Description: "Evaluate different approaches",
		Epic:        "E05",
		Feature:     "E05-F01",
		AgentType:   models.AgentTypeGeneral,
		Priority:    4,
		DependsOn:   []string{},
		CreatedAt:   time.Now().UTC(),
	}

	result, err := renderer.Render(models.AgentTypeGeneral, data)

	require.NoError(t, err)
	assert.Contains(t, result, "agent: general")
	assert.Contains(t, result, "## Requirements")
	assert.Contains(t, result, "## Implementation Plan")
	assert.Contains(t, result, "## Deliverables")
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
		AgentType:   models.AgentTypeGeneral,
		Priority:    5,
		DependsOn:   []string{},
		CreatedAt:   time.Now().UTC(),
	}

	result, err := renderer.Render(models.AgentTypeGeneral, data)

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
		AgentType: models.AgentTypeBackend,
		Priority:  5,
		DependsOn: []string{"T-E01-F02-001", "T-E01-F02-002", "T-E01-F02-003"},
		CreatedAt: time.Now().UTC(),
	}

	result, err := renderer.Render(models.AgentTypeBackend, data)

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
		AgentType:   models.AgentTypeGeneral,
		Priority:    5,
		DependsOn:   []string{"T-E01-F01-000"},
		CreatedAt:   time.Date(2025, 12, 14, 10, 30, 0, 0, time.UTC),
	}

	result, err := renderer.Render(models.AgentTypeGeneral, data)
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
	assert.Contains(t, frontmatter, "status: todo")
	assert.Contains(t, frontmatter, "priority: 5")
	assert.Contains(t, frontmatter, `depends_on: ["T-E01-F01-000"]`)
	assert.Contains(t, frontmatter, "created_at: 2025-12-14T10:30:00Z")
}

func TestRenderer_Render_InvalidAgentType(t *testing.T) {
	loader := NewLoader("")
	renderer := NewRenderer(loader)

	data := TemplateData{
		Key:       "T-E01-F01-001",
		Title:     "Test Task",
		Epic:      "E01",
		Feature:   "E01-F01",
		AgentType: "invalid",
		Priority:  5,
		CreatedAt: time.Now().UTC(),
	}

	result, err := renderer.Render("invalid", data)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load template")
	assert.Empty(t, result)
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
