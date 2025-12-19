package templates

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TemplateData holds all variables available to task templates
type TemplateData struct {
	Key         string
	Title       string
	Description string
	Epic        string
	Feature     string
	AgentType   models.AgentType
	Priority    int
	DependsOn   []string
	CreatedAt   time.Time
}

// Renderer handles template rendering for task markdown files
type Renderer struct {
	loader *Loader
}

// NewRenderer creates a new template renderer
func NewRenderer(loader *Loader) *Renderer {
	return &Renderer{
		loader: loader,
	}
}

// Render renders a task template with the given data
func (r *Renderer) Render(agentType models.AgentType, data TemplateData) (string, error) {
	// Load template for agent type
	tmplContent, err := r.loader.LoadTemplate(agentType)
	if err != nil {
		return "", fmt.Errorf("failed to load template for %s: %w", agentType, err)
	}

	// Create template with custom functions
	tmpl, err := template.New("task").Funcs(templateFuncs()).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderWithSelection renders a task template with template selection priority:
// 1. Custom template (if provided)
// 2. Agent-specific template (if agentType provided and template exists)
// 3. General template (fallback)
func (r *Renderer) RenderWithSelection(agentType models.AgentType, customTemplatePath string, data TemplateData) (string, error) {
	var tmplContent string
	var err error

	// Priority 1: Custom template
	if customTemplatePath != "" {
		tmplContent, err = r.loader.LoadCustomTemplate(customTemplatePath)
		if err != nil {
			return "", fmt.Errorf("failed to load custom template: %w", err)
		}
	} else if agentType != "" {
		// Priority 2: Agent-specific template (with fallback to general built-in)
		tmplContent, err = r.loader.LoadTemplate(agentType)
		if err != nil {
			return "", fmt.Errorf("failed to load template: %w", err)
		}
	} else {
		// Priority 3: General template (should not reach here due to default in validator)
		tmplContent, err = r.loader.LoadTemplate(models.AgentTypeGeneral)
		if err != nil {
			return "", fmt.Errorf("failed to load general template: %w", err)
		}
	}

	// Create template with custom functions
	tmpl, err := template.New("task").Funcs(templateFuncs()).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"quote": func(items []string) []string {
			quoted := make([]string, len(items))
			for i, item := range items {
				quoted[i] = fmt.Sprintf("%q", item)
			}
			return quoted
		},
		"isEmpty": func(s string) bool {
			return strings.TrimSpace(s) == ""
		},
		"formatTime": func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
	}
}
