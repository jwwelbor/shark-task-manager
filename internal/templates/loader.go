package templates

import (
	"os"
	"path/filepath"
)

// Loader handles loading consolidated task file templates from disk or embedded shark-data.
type Loader struct {
	templateDir string
	useEmbedded bool
}

// NewLoader creates a new template loader
// If templateDir is empty, it resolves task.md through shark-data/file_templates
// and embedded defaults.
func NewLoader(templateDir string) *Loader {
	useEmbedded := templateDir == ""
	if templateDir == "" {
		templateDir = GetFileTemplatesDirName()
	}
	return &Loader{
		templateDir: templateDir,
		useEmbedded: useEmbedded,
	}
}

// LoadTemplate loads the consolidated task file template. The agent type is
// rendered as data inside the template; it does not select a different skeleton.
func (l *Loader) LoadTemplate(agentType string) (string, error) {
	return l.loadTemplateFile("task.md")
}

func (l *Loader) loadTemplateFile(filename string) (string, error) {
	if l.useEmbedded {
		content, err := ReadFileTemplate(filename)
		if err == nil {
			return string(content), nil
		}
		return "", err
	}

	path := filepath.Join(l.templateDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetAvailableAgentTypes returns common agent labels for callers that present choices.
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
