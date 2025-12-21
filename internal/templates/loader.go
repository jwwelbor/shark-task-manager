package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/models"
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
func (l *Loader) LoadTemplate(agentType models.AgentType) (string, error) {
	filename := fmt.Sprintf("task-%s.md", agentType)

	// Try embedded templates first if configured
	if l.useEmbedded {
		content, err := embeddedTemplates.ReadFile(filepath.Join("task_templates", filename))
		if err == nil {
			return string(content), nil
		}

		// If agent-specific template not found and it's not "general", try general template
		if agentType != models.AgentTypeGeneral {
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
		if agentType != models.AgentTypeGeneral {
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

// LoadCustomTemplate loads a custom template from the specified path
// with security validation to prevent directory traversal attacks
func (l *Loader) LoadCustomTemplate(templatePath string) (string, error) {
	// Security: Prevent directory traversal
	cleanPath := filepath.Clean(templatePath)
	if containsDirectoryTraversal(cleanPath) {
		return "", fmt.Errorf("invalid template path: directory traversal not allowed")
	}

	// Check file exists
	if _, err := os.Stat(cleanPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("template file not found: %s", cleanPath)
		}
		return "", fmt.Errorf("failed to access template file: %w", err)
	}

	// Load and return template
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	return string(content), nil
}

// containsDirectoryTraversal checks if a path contains directory traversal attempts
func containsDirectoryTraversal(path string) bool {
	// Check for .. in the path (after cleaning)
	parts := filepath.SplitList(path)
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	// Also check the full path string
	return !filepath.IsAbs(path) && (len(path) >= 2 && (path[0:2] == ".." || path[0:3] == "../"))
}

// GetAvailableAgentTypes returns all available agent types
func (l *Loader) GetAvailableAgentTypes() []models.AgentType {
	return []models.AgentType{
		models.AgentTypeFrontend,
		models.AgentTypeBackend,
		models.AgentTypeAPI,
		models.AgentTypeTesting,
		models.AgentTypeDevOps,
		models.AgentTypeGeneral,
	}
}
