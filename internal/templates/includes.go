package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// Resolve preprocesses content by inlining all {{include: <path>}} and
// {{augment: <path>}} directives. Resolution is recursive (included files may
// contain their own includes); cycle detection is enforced by a depth cap of
// IncludeDepthCap.
//
// Override semantics: for each include, <dataRoot>/overrides/<path> is checked
// first; if it exists, its content replaces the default at <dataRoot>/<path>.
// Override files are never merged with defaults — they win or they're absent.
//
// If dataRoot is empty (legacy mode), include directives are left in place
// verbatim. This is intentional: the legacy `shark-templates/` mode has no
// data root, so include is a no-op there. Callers using {{template ...}}
// for in-tree partials are unaffected.
func (r *IncludeResolver) Resolve(content string) (string, error) {
	if r.dataRoot == "" {
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

		resolvedPath, err := r.resolvePath(path)
		if err != nil {
			firstErr = err
			return match
		}

		if visited[resolvedPath] {
			firstErr = fmt.Errorf("include cycle detected: %s already on the include stack", resolvedPath)
			return match
		}

		fileContent, err := os.ReadFile(resolvedPath)
		if err != nil {
			firstErr = fmt.Errorf("failed to read include %s: %w", resolvedPath, err)
			return match
		}

		if len(fileContent) > IncludeSizeWarnBytes && r.warnFn != nil {
			r.warnFn(resolvedPath, len(fileContent))
		}

		// Strip frontmatter from .md includes — same rule as top-level prompts.
		body := string(fileContent)
		if filepath.Ext(resolvedPath) == ".md" {
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

// resolvePath looks up an include path under the data root, preferring an
// override file at <dataRoot>/overrides/<path>. The path is resolved as POSIX
// (forward slashes) and converted to OS-native separators; absolute paths and
// paths escaping the data root via .. are rejected.
func (r *IncludeResolver) resolvePath(includePath string) (string, error) {
	// Normalize separators and reject absolute or upward paths.
	cleanedPath := strings.TrimSpace(includePath)
	if cleanedPath == "" {
		return "", fmt.Errorf("include path is empty")
	}
	if filepath.IsAbs(cleanedPath) {
		return "", fmt.Errorf("include path must be relative to data root: %q", cleanedPath)
	}
	// Convert any forward slashes to OS-native separators.
	osPath := filepath.FromSlash(cleanedPath)
	// Reject upward traversal, but only on a real leading ".." segment — a
	// substring check over-matches legitimate names like "notes..draft.md".
	// filepath.Clean collapses any interior ".." so an escaping path surfaces
	// as a leading "..".
	if cleaned := filepath.Clean(osPath); cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("include path must not escape the data root: %q", cleanedPath)
	}

	// Override wins.
	overridePath := filepath.Join(r.dataRoot, "overrides", osPath)
	if _, err := os.Stat(overridePath); err == nil {
		return overridePath, nil
	}

	// Default location.
	defaultPath := filepath.Join(r.dataRoot, osPath)
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	}

	return "", fmt.Errorf("include not found: %s (looked under %s and %s/overrides/)", cleanedPath, r.dataRoot, r.dataRoot)
}
