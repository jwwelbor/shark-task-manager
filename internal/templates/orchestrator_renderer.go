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

// defaultTemplateDir is the default template directory name (legacy layout).
// Shark 2.0 prefers shark-data/prompts/ — see findTemplateDir for resolution order.
const defaultTemplateDir = "shark-templates"

// sharkDataPromptsSubdir is the prompts subdirectory inside the shark-data layout.
// Resolution prefers <project>/shark-data/prompts/ over <project>/shark-templates/.
const sharkDataPromptsSubdir = "shark-data/prompts"

// promptFileExtensions are the file extensions the engine recognizes as prompt
// files. Shark 2.0 introduces .md alongside the legacy .tmpl. The engine reads
// either; .md files may carry optional YAML frontmatter that is stripped before
// the body is parsed as a Go template.
var promptFileExtensions = []string{".tmpl", ".md"}

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

// configuredTemplateDir is an optional override set via SetConfiguredTemplateDir.
// When non-empty, findTemplateDir uses this name instead of the default "shark-templates".
var configuredTemplateDir string

// SetConfiguredTemplateDir sets the template directory name from config.
// This should be called early in CLI initialization with the value from
// Config.GetTemplateDirectory(). Pass empty string to use the default.
func SetConfiguredTemplateDir(dir string) {
	configuredTemplateDir = dir
}

// GetTemplateDirName returns the configured template directory name, falling
// back to "shark-templates" if not configured. This is safe to call after
// CLI initialization has run (PersistentPreRunE sets it via SetConfiguredTemplateDir).
// Paths starting with "~/" are expanded to the user's home directory.
func GetTemplateDirName() string {
	dir := configuredTemplateDir
	if dir == "" {
		return defaultTemplateDir
	}
	if strings.HasPrefix(dir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, dir[2:])
		}
	}
	return dir
}

// findTemplateDir locates the template directory by walking up from the
// working directory. Returns the first directory containing a matching
// subdirectory with prompt files (.tmpl or .md), or falls back to the
// configured directory name.
//
// Resolution order (Shark 2.0):
//  1. If GetTemplateDirName() returns an absolute path, use it directly.
//  2. Walk up looking for a `shark-data/prompts/` subdirectory containing .md
//     prompt files — this is the canonical Shark 2.0 layout.
//  3. Walk up looking for the configured (or default) directory containing
//     .tmpl prompt files — legacy `shark-templates/` fallback for one release.
//  4. Fall back to the configured directory name as a relative path.
func findTemplateDir() string {
	dirName := GetTemplateDirName()

	// Absolute paths are used directly — no walk-up needed.
	if filepath.IsAbs(dirName) {
		return dirName
	}

	wd, err := os.Getwd()
	if err != nil {
		return dirName
	}

	// Pass 1: prefer shark-data/prompts/ (Shark 2.0).
	currentDir := wd
	for {
		candidate := filepath.Join(currentDir, sharkDataPromptsSubdir)
		if hasPromptFiles(candidate) {
			return candidate
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	// Pass 2: fall back to legacy shark-templates/.
	currentDir = wd
	for {
		candidate := filepath.Join(currentDir, dirName)
		if hasPromptFiles(candidate) {
			return candidate
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return dirName
}

// hasPromptFiles reports whether dir/<entity>/<status>.<ext> matches any prompt
// file recognized by the engine (.tmpl or .md).
func hasPromptFiles(dir string) bool {
	for _, ext := range promptFileExtensions {
		matches, _ := filepath.Glob(filepath.Join(dir, "*", "*"+ext))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// stripFrontmatter removes a leading YAML frontmatter block from prompt content
// if present. The frontmatter must be delimited by `---` on its own line at the
// very top of the file and a closing `---` on its own line. The returned content
// is the body without the frontmatter; if no frontmatter is present, content is
// returned unchanged.
//
// This is applied to .md prompt files in Shark 2.0; .tmpl files are returned
// unchanged so legacy templates render exactly as before.
func stripFrontmatter(content string) string {
	// Frontmatter must start with --- on the first line.
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return content
	}

	// Find the closing --- on its own line.
	rest := content[strings.Index(content, "\n")+1:]
	for i, line := range splitLines(rest) {
		if line == "---" {
			// Skip past the closing delimiter line.
			lines := splitLines(rest)
			body := strings.Join(lines[i+1:], "\n")
			return body
		}
	}

	// No closing delimiter — treat as if there were no frontmatter.
	return content
}

// splitLines splits s into lines, preserving line ordering and trimming the
// trailing newline of each line. Handles both LF and CRLF endings.
func splitLines(s string) []string {
	// Normalize CRLF -> LF then split on LF.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

// NewOrchestratorRenderer creates a new orchestrator template renderer.
// It precompiles all prompt files (.tmpl or .md) in the templateDir and its
// subdirectories. .md files may carry a YAML frontmatter block which is
// stripped before parsing the body as a Go template.
func NewOrchestratorRenderer(templateDir string) (*OrchestratorRenderer, error) {
	// Create a new template with custom functions
	tmpl := template.New("orchestrator").Funcs(orchestratorFuncs())

	// Glob all recognized prompt extensions across the entity subdirectories.
	var matches []string
	for _, ext := range promptFileExtensions {
		pattern := filepath.Join(templateDir, "*", "*"+ext)
		extMatches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob templates (%s): %w", ext, err)
		}
		matches = append(matches, extMatches...)
	}

	// If no prompt files found, return empty renderer (valid for empty directory).
	if len(matches) == 0 {
		return &OrchestratorRenderer{
			templates:   tmpl,
			templateDir: templateDir,
		}, nil
	}

	// Parse all templates manually to preserve subdirectory paths in template names.
	// This allows us to distinguish between epic/ready_for_research.tmpl and
	// feature/ready_for_research.tmpl.
	for _, filePath := range matches {
		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read template file %s: %w", filePath, err)
		}

		// Strip optional YAML frontmatter from .md prompts before parsing.
		// .tmpl files pass through unchanged for backward compatibility.
		body := string(content)
		if filepath.Ext(filePath) == ".md" {
			body = stripFrontmatter(body)
		}

		// Calculate the relative path from templateDir for the template name
		relPath, err := filepath.Rel(templateDir, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate relative path for %s: %w", filePath, err)
		}

		// Parse the template with the relative path as its name (e.g.,
		// "epic/ready_for_research.tmpl" or "task/in_qa.md")
		_, err = tmpl.New(relPath).Parse(body)
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
		// Use test directory if set, otherwise find shark-templates by walking up
		var templateDir string
		if testTemplateDir != "" {
			templateDir = testTemplateDir
		} else {
			templateDir = findTemplateDir()
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
