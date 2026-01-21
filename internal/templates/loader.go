package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed task_templates/*.md
var embeddedTemplates embed.FS

// Loader handles loading task templates from filesystem or embedded files
type Loader struct {
	templateDir string
	useEmbedded bool
}

// NewLoader creates a new template loader
// If templateDir is empty, uses embedded templates
func NewLoader(templateDir string) *Loader {
	useEmbedded := templateDir == ""
	if templateDir == "" {
		templateDir = "templates"
	}
	return &Loader{
		templateDir: templateDir,
		useEmbedded: useEmbedded,
	}
}

// LoadTemplate loads a template for the given agent type
// Falls back to general template if agent-specific template not found
func (l *Loader) LoadTemplate(agentType string) (string, error) {
	filename := fmt.Sprintf("task-%s.md", agentType)

	// Try embedded templates first if configured
	if l.useEmbedded {
		content, err := embeddedTemplates.ReadFile(filepath.Join("task_templates", filename))
		if err == nil {
			return string(content), nil
		}

		// If agent-specific template not found and it's not "general", try general template
		if agentType != "general" {
			generalFilename := "task-general.md"
			content, err := embeddedTemplates.ReadFile(filepath.Join("task_templates", generalFilename))
			if err == nil {
				return string(content), nil
			}
		}
	}

	// Try filesystem
	path := filepath.Join(l.templateDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		// If agent-specific template not found and it's not "general", try general template
		if agentType != "general" {
			generalPath := filepath.Join(l.templateDir, "task-general.md")
			content, err := os.ReadFile(generalPath)
			if err == nil {
				return string(content), nil
			}
		}
		return "", fmt.Errorf("template not found: %s (and fallback to general template failed)", filename)
	}

	return string(content), nil
}

// GetAvailableAgentTypes returns all available agent types
func (l *Loader) GetAvailableAgentTypes() []string {
	return []string{
		"frontend",
		"backend",
		"api",
		"testing",
		"devops",
		"general",
	}
}
