package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

// IncludeDepthCap is the maximum nesting depth for {{include:}} directives.
// A depth >= IncludeDepthCap during resolution is treated as a cycle / runaway
// recursion and aborts with an error.
const IncludeDepthCap = 5

// IncludeSizeWarnBytes is the threshold above which an inlined file logs a
// warning. The render still succeeds; the warning is informational so authors
// can spot prompts that have grown beyond a comfortable single-prompt budget.
const IncludeSizeWarnBytes = 50 * 1024 // 50KB

// includeDirectivePattern matches `{{include: <path>}}` or
// `{{augment: <path>}}` directives. The path may contain forward slashes,
// underscores, dots, and word characters. Whitespace around the colon and the
// path is tolerated.
//
// Examples that match:
//
//	{{include: skills/quality/workflows/qa-testing.md}}
//	{{include:  prompts/_partials/_advance.md  }}
//	{{augment: skills/architecture/SKILL.md}}
var includeDirectivePattern = regexp.MustCompile(`\{\{\s*(include|augment)\s*:\s*([^}\s]+(?:\s+[^}\s]+)*?)\s*\}\}`)

// IncludeResolver resolves {{include:}} (and {{augment:}}) directives against
// a Shark 2.0 data root. Override resolution: a file at
// <dataRoot>/overrides/<path> fully replaces <dataRoot>/<path> — never merges.
//
// IncludeResolver is independent of Go's text/template; callers preprocess
// source text with Resolve before handing it to template.Parse.
type IncludeResolver struct {
	// dataRoot is the directory under which include paths resolve. For Shark
	// 2.0 this is the `shark-data/` directory (the parent of `prompts/`,
	// `skills/`, `agents/`, `overrides/`).
	dataRoot string

	// useEmbed enables hybrid resolution: when a file is not found on disk,
	// the resolver falls back to the embedded canonical tree (sharkdata.ReadEmbedded).
	// This allows zero-config operation without requiring `shark install-shark-data`.
	useEmbed bool

	// warnFn is called when an inlined file's size exceeds
	// IncludeSizeWarnBytes. Defaults to writing to os.Stderr; tests inject
	// their own to capture warnings.
	warnFn func(path string, size int)
}

// NewIncludeResolver constructs an IncludeResolver against the given data
// root. dataRoot may be empty — in that case Resolve becomes a no-op for any
// include directive (returns the directive's literal text). This lets the
// engine continue working in legacy `shark-templates/` mode where there is
// no data root.
func NewIncludeResolver(dataRoot string) *IncludeResolver {
	return &IncludeResolver{
		dataRoot: dataRoot,
		warnFn: func(path string, size int) {
			fmt.Fprintf(os.Stderr,
				"[shark] warning: included file %s is %d bytes (>%d threshold) — consider splitting\n",
				path, size, IncludeSizeWarnBytes,
			)
		},
	}
}

// NewIncludeResolverWithEmbed constructs an IncludeResolver with hybrid
// embed/disk resolution. When a file is not found on disk (under dataRoot),
// it falls back to the embedded canonical shark-data/ tree. dataRoot may be
// empty — disk lookup is skipped but embed fallback still applies.
//
// Precedence order:
//  1. <dataRoot>/overrides/<path>   (disk override, wins always)
//  2. <dataRoot>/<path>             (disk default)
//  3. embedded canonical tree       (zero-config backstop)
func NewIncludeResolverWithEmbed(dataRoot string) *IncludeResolver {
	r := NewIncludeResolver(dataRoot)
	r.useEmbed = true
	return r
}

// Resolve preprocesses content by inlining all {{include: <path>}} and
// {{augment: <path>}} directives. Resolution is recursive (included files may
// contain their own includes); cycle detection is enforced by a depth cap of
// IncludeDepthCap.
//
// Override semantics: for each include, <dataRoot>/overrides/<path> is checked
// first; if it exists, its content replaces the default at <dataRoot>/<path>.
// Override files are never merged with defaults — they win or they're absent.
//
// If dataRoot is empty and embed fallback is disabled (legacy mode), include
// directives are left in place verbatim. This is intentional: the legacy
// `shark-templates/` mode has no data root, so include is a no-op there.
// Embed-aware resolvers still resolve includes from the embedded bundle.
func (r *IncludeResolver) Resolve(content string) (string, error) {
	if r.dataRoot == "" && !r.useEmbed {
		return content, nil
	}
	return r.resolveDepth(content, 0, map[string]bool{})
}

func (r *IncludeResolver) resolveDepth(content string, depth int, visited map[string]bool) (string, error) {
	if depth >= IncludeDepthCap {
		return "", fmt.Errorf("include resolution exceeded depth cap of %d (likely cycle or runaway recursion)", IncludeDepthCap)
	}

	// Walk every directive in `content`. Use a non-trivial loop so we can
	// surface the first error rather than swallow it inside ReplaceAllString.
	var firstErr error
	result := includeDirectivePattern.ReplaceAllStringFunc(content, func(match string) string {
		if firstErr != nil {
			return match
		}
		submatches := includeDirectivePattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			// Pattern didn't capture as expected — leave the directive
			// in place rather than error.
			return match
		}
		// directive := submatches[1]   // "include" or "augment" — same handling for now
		path := submatches[2]

		resolvedPath, fileContent, err := r.resolveContent(path)
		if err != nil {
			firstErr = err
			return match
		}

		// Use a stable key for cycle detection. For embedded content, the
		// resolved path is a virtual "embed:<relPath>" string; for disk it's
		// the absolute OS path.
		if visited[resolvedPath] {
			firstErr = fmt.Errorf("include cycle detected: %s already on the include stack", resolvedPath)
			return match
		}

		if len(fileContent) > IncludeSizeWarnBytes && r.warnFn != nil {
			r.warnFn(resolvedPath, len(fileContent))
		}

		// Strip frontmatter from .md includes — same rule as top-level prompts.
		body := string(fileContent)
		if filepath.Ext(resolvedPath) == ".md" || strings.HasSuffix(resolvedPath, ".md") {
			body = stripFrontmatter(body)
		}

		// Recurse: included content may itself contain {{include:}} directives.
		// The visited set is path-scoped: we add the current path before recursion
		// and remove it after, so siblings can include the same partial.
		visited[resolvedPath] = true
		nested, err := r.resolveDepth(body, depth+1, visited)
		delete(visited, resolvedPath)
		if err != nil {
			firstErr = err
			return match
		}
		return nested
	})

	if firstErr != nil {
		return "", firstErr
	}
	return result, nil
}

// resolveContent resolves an include path to its content using the hybrid
// embed/disk precedence:
//  1. <dataRoot>/overrides/<path>   (disk override)
//  2. <dataRoot>/<path>             (disk default)
//  3. embedded canonical tree       (zero-config backstop, when useEmbed=true)
//
// Returns (resolvedKey, content, error). resolvedKey is the disk path for disk
// files, or "embed:<relPath>" for embedded files (used for cycle detection).
func (r *IncludeResolver) resolveContent(includePath string) (resolvedKey string, content []byte, err error) {
	cleanedPath := strings.TrimSpace(includePath)
	if cleanedPath == "" {
		return "", nil, fmt.Errorf("include path is empty")
	}
	if filepath.IsAbs(cleanedPath) {
		return "", nil, fmt.Errorf("include path must be relative to data root: %q", cleanedPath)
	}
	osPath := filepath.FromSlash(cleanedPath)
	if cleaned := filepath.Clean(osPath); cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", nil, fmt.Errorf("include path must not escape the data root: %q", cleanedPath)
	}

	if r.dataRoot != "" {
		// Override wins.
		overridePath := filepath.Join(r.dataRoot, "overrides", osPath)
		if _, statErr := os.Stat(overridePath); statErr == nil {
			data, readErr := os.ReadFile(overridePath)
			if readErr != nil {
				return "", nil, fmt.Errorf("failed to read include %s: %w", overridePath, readErr)
			}
			return overridePath, data, nil
		}

		// Default disk location.
		defaultPath := filepath.Join(r.dataRoot, osPath)
		if _, statErr := os.Stat(defaultPath); statErr == nil {
			data, readErr := os.ReadFile(defaultPath)
			if readErr != nil {
				return "", nil, fmt.Errorf("failed to read include %s: %w", defaultPath, readErr)
			}
			return defaultPath, data, nil
		}
	}

	// Embed backstop.
	if r.useEmbed {
		data, readErr := sharkdata.ReadEmbedded(cleanedPath)
		if readErr == nil {
			return "embed:" + cleanedPath, data, nil
		}
	}

	if r.dataRoot != "" {
		return "", nil, fmt.Errorf("include not found: %s (looked under %s and %s/overrides/)", cleanedPath, r.dataRoot, r.dataRoot)
	}
	return "", nil, fmt.Errorf("include not found: %s (no disk data root and embed fallback %v)", cleanedPath, r.useEmbed)
}
