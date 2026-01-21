package templates

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// TemplateData holds all variables available to task templates
type TemplateData struct {
	Key         string
	Title       string
	Description string
	Epic        string
	Feature     string
	AgentType   string
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
// Accepts any non-empty agent type string and falls back to general template if needed
func (r *Renderer) Render(agentType string, data TemplateData) (string, error) {
	// Load template for agent type (will fallback to general if agent-specific not found)
	tmplContent, err := r.loader.LoadTemplate(agentType)
	if err != nil {
		return "", fmt.Errorf("failed to load template: %w", err)
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
