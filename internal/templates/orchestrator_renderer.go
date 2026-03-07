package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

// OrchestratorRenderer handles template rendering for orchestrator instructions
type OrchestratorRenderer struct {
	templates   *template.Template // Precompiled template set
	templateDir string             // Base directory for templates
}

// Singleton pattern for global template engine
var (
	engineOnce      sync.Once
	engineInstance  *OrchestratorRenderer
	engineError     error
	testTemplateDir string // For testing only
)

// NewOrchestratorRenderer creates a new orchestrator template renderer
// It precompiles all .tmpl files in the templateDir and its subdirectories
func NewOrchestratorRenderer(templateDir string) (*OrchestratorRenderer, error) {
	// Parse all .tmpl files in templateDir/**/*.tmpl
	pattern := filepath.Join(templateDir, "*", "*.tmpl")

	// Create a new template with custom functions
	tmpl := template.New("orchestrator").Funcs(orchestratorFuncs())

	// Try to parse templates - empty dir is ok
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob templates: %w", err)
	}

	// If no templates found, return empty renderer (valid for empty directory)
	if len(matches) == 0 {
		return &OrchestratorRenderer{
			templates:   tmpl,
			templateDir: templateDir,
		}, nil
	}

	// Parse all templates manually to preserve subdirectory paths in template names
	// This allows us to distinguish between epic/ready_for_research.tmpl and feature/ready_for_research.tmpl
	for _, filePath := range matches {
		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read template file %s: %w", filePath, err)
		}

		// Calculate the relative path from templateDir for the template name
		relPath, err := filepath.Rel(templateDir, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate relative path for %s: %w", filePath, err)
		}

		// Parse the template with the relative path as its name (e.g., "epic/ready_for_research.tmpl")
		_, err = tmpl.New(relPath).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", relPath, err)
		}
	}

	return &OrchestratorRenderer{
		templates:   tmpl,
		templateDir: templateDir,
	}, nil
}

// GetOrchestratorEngine returns the singleton orchestrator template engine
// It initializes the engine on first call using the default or test template directory
func GetOrchestratorEngine() *OrchestratorRenderer {
	engineOnce.Do(func() {
		// Use test directory if set, otherwise default to "templates"
		templateDir := "shark-templates"
		if testTemplateDir != "" {
			templateDir = testTemplateDir
		}

		engineInstance, engineError = NewOrchestratorRenderer(templateDir)
		if engineError != nil {
			// In production, this would log.Fatalf
			// For tests, we let the error propagate
			panic(fmt.Sprintf("Failed to initialize template engine: %v", engineError))
		}
	})

	return engineInstance
}

// Render executes a template with the given variables
// Returns the rendered string or an error if the template is not found or execution fails
// Template lookup strategy:
// 1. First try exact match (for "epic/ready_for_research.tmpl" style references)
// 2. Fall back to basename only (for backward compatibility)
func (r *OrchestratorRenderer) Render(templateName string, vars map[string]string) (string, error) {
	// First try to find template by full path (handles "epic/ready_for_research.tmpl")
	tmpl := r.templates.Lookup(templateName)

	// If not found, try base name only (backward compatibility)
	if tmpl == nil {
		baseName := filepath.Base(templateName)
		tmpl = r.templates.Lookup(baseName)
	}

	if tmpl == nil {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	// Execute template with variables
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// orchestratorFuncs returns custom template functions for orchestrator templates
func orchestratorFuncs() template.FuncMap {
	return template.FuncMap{
		// Comparison functions
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},

		// String helper functions
		"isEmpty": func(s string) bool {
			return strings.TrimSpace(s) == ""
		},

		// Complexity tier helper functions (convenience wrappers)
		"isSimple": func(tier string) bool {
			return tier == "SIMPLE"
		},
		"isStandard": func(tier string) bool {
			return tier == "STANDARD"
		},
		"isComplex": func(tier string) bool {
			return tier == "COMPLEX"
		},
	}
}
