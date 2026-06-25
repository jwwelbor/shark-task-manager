package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/jwwelbor/shark-task-manager/internal/pathutil"
)

// defaultTemplateDir is the default template directory name (legacy layout).
// Shark 2.0 prefers shark-data/prompts/ — see findTemplateDir for resolution order.
const defaultTemplateDir = "shark-templates"

// defaultSharkDataDirName is the default content-bundle root directory name.
// It mirrors config.DefaultSharkDataPath; redeclared here to avoid a
// templates -> config import edge. findTemplateDir derives the prompts
// subdirectory from the configured shark_data_path (see
// SetConfiguredSharkDataPath), defaulting to this value.
const defaultSharkDataDirName = "shark-data"

// sharkDataPromptsLeaf is the prompts subdirectory leaf inside the shark-data
// bundle layout. Joined onto the configured bundle root to form the resolved
// prompts directory (e.g. "shark-data/prompts" by default).
const sharkDataPromptsLeaf = "prompts"

// promptFileExtensions are the file extensions the engine recognizes as prompt
// files. Shark 2.0 introduces .md alongside the legacy .tmpl. The engine reads
// either; .md files may carry optional YAML frontmatter that is stripped before
// the body is parsed as a Go template.
var promptFileExtensions = []string{".tmpl", ".md"}

// OrchestratorRenderer handles template rendering for orchestrator instructions
type OrchestratorRenderer struct {
	templates   *template.Template // Precompiled template set
	templateDir string             // Base directory for templates
	includeRoot string             // Data root for {{include:}} resolution (empty in legacy mode)
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

// configuredSharkDataPath is an optional override set via
// SetConfiguredSharkDataPath. When non-empty, findTemplateDir derives the
// Shark 2.0 prompts directory from <configuredSharkDataPath>/prompts instead
// of the default "shark-data/prompts". This is the bundle root selected by
// `shark_data_path` in .sharkconfig.json.
var configuredSharkDataPath string

// SetConfiguredTemplateDir sets the template directory name from config.
// This should be called early in CLI initialization with the value from
// Config.GetTemplateDirectory(). Pass empty string to use the default.
func SetConfiguredTemplateDir(dir string) {
	configuredTemplateDir = dir
}

// SetConfiguredSharkDataPath sets the content-bundle root from config. Call
// early in CLI initialization with the value from Config.GetSharkDataPath().
// Pass empty string to use the default ("shark-data"). findTemplateDir derives
// the prompts subdir (<root>/prompts) from this value.
func SetConfiguredSharkDataPath(dir string) {
	configuredSharkDataPath = dir
}

// sharkDataPromptsSubdir returns the prompts subdirectory derived from the
// configured shark_data_path bundle root, defaulting to "shark-data/prompts".
// A "~/"-prefixed root is expanded to the user's home directory so absolute
// shared-bundle roots resolve. The returned path is used as a walk-up leaf
// (relative) unless the configured root is absolute, in which case it is
// returned as an absolute path.
func sharkDataPromptsSubdir() string {
	root := configuredSharkDataPath
	if root == "" {
		root = defaultSharkDataDirName
	}
	root = pathutil.ExpandHome(root)
	return filepath.Join(root, sharkDataPromptsLeaf)
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
	dir = pathutil.ExpandHome(dir)
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

	promptsSubdir := sharkDataPromptsSubdir()

	// Pass 0: an absolute shark_data_path bundle root resolves its prompts
	// directory directly — no walk-up needed (shared-bundle parity).
	if filepath.IsAbs(promptsSubdir) {
		if hasPromptFiles(promptsSubdir) {
			return promptsSubdir
		}
	} else {
		// Pass 1: prefer <shark_data_path>/prompts/ (Shark 2.0), walking up.
		currentDir := wd
		for {
			candidate := filepath.Join(currentDir, promptsSubdir)
			if hasPromptFiles(candidate) {
				return candidate
			}
			parentDir := filepath.Dir(currentDir)
			if parentDir == currentDir {
				break
			}
			currentDir = parentDir
		}
	}

	// Pass 2: fall back to legacy shark-templates/.
	currentDir := wd
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
//
// If templateDir is the Shark 2.0 layout (under shark-data/prompts/),
// {{include:}} directives in any prompt are resolved at parse time against
// the parent shark-data/ directory, with shark-data/overrides/<path> taking
// precedence over shark-data/<path>. In the legacy shark-templates/ layout,
// no data root is detected and {{include:}} directives pass through verbatim.
func NewOrchestratorRenderer(templateDir string) (*OrchestratorRenderer, error) {
	includeRoot := detectIncludeRoot(templateDir)
	resolver := NewIncludeResolver(includeRoot)

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
			includeRoot: includeRoot,
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

		// Resolve {{include:}} directives before Go-template parsing. In legacy
		// mode (no data root), the resolver is a no-op.
		body, err = resolver.Resolve(body)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve includes in %s: %w", filePath, err)
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
		includeRoot: includeRoot,
	}, nil
}

// IncludeRoot returns the Shark 2.0 data root this renderer resolves
// {{include:}} directives against, or an empty string when the renderer is
// operating in legacy shark-templates/ mode. Callers outside this package
// use this to drive auxiliary include-style resolution (e.g., `shark next`
// prepending the agent body to the rendered prompt).
func (r *OrchestratorRenderer) IncludeRoot() string {
	return r.includeRoot
}

// detectIncludeRoot returns the Shark 2.0 data root for include resolution
// when templateDir is the prompts/ subdirectory of a shark-data/ tree, or an
// empty string when the legacy shark-templates/ layout is in use.
//
// templateDir is considered Shark 2.0-shaped when its base directory name is
// "prompts" and its parent directory exists.
func detectIncludeRoot(templateDir string) string {
	abs, err := filepath.Abs(templateDir)
	if err != nil {
		return ""
	}
	if filepath.Base(abs) != "prompts" {
		return ""
	}
	parent := filepath.Dir(abs)
	if info, err := os.Stat(parent); err == nil && info.IsDir() {
		return parent
	}
	return ""
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

// orchestratorFuncs returns custom template functions for orchestrator templates.
//
// The function set is deliberately small: comparison + string predicates +
// complexity-tier shortcuts + the Sprig-style data-structure helpers `dict`
// and `list` (used by Shark 2.0 partials for keyword-argument-style template
// invocation), plus `default` (Sprig parity for nil/empty fallbacks).
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

		// Sprig-parity helpers for partial composition.
		//
		// `dict` builds a map[string]interface{} from alternating key/value
		// pairs. Used as: {{template "_advance" (dict "note_type" "review" "summary" "QA PASS")}}.
		//
		// `list` builds a []interface{}. Used as: {{template "_resolve_spec_paths" (dict "domains" (list "QA_REPORTS"))}}.
		//
		// `default` returns the first non-empty value: {{.x | default "fallback"}}.
		"dict": func(pairs ...interface{}) (map[string]interface{}, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments, got %d", len(pairs))
			}
			out := make(map[string]interface{}, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				key, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict key at position %d must be a string, got %T", i, pairs[i])
				}
				out[key] = pairs[i+1]
			}
			return out, nil
		},
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"default": func(fallback interface{}, value interface{}) interface{} {
			// Treat nil and empty string as "use fallback".
			if value == nil {
				return fallback
			}
			if s, ok := value.(string); ok && s == "" {
				return fallback
			}
			return value
		},
	}
}
