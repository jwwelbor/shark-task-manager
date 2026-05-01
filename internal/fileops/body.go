package fileops

import "strings"

// ReplaceBodyAfterFrontmatter swaps the body of a markdown file (everything
// after the closing "---" frontmatter line) with newBody. If newBody is empty
// the rendered string is returned unchanged. If the input has no frontmatter
// delimiters the function returns newBody alone (with a single trailing
// newline).
//
// The function is used by entity create flows to honour `--content`/stdin
// overrides without disturbing frontmatter that the service or template
// renderer already wrote.
func ReplaceBodyAfterFrontmatter(rendered, newBody string) string {
	if newBody == "" {
		return rendered
	}

	body := ensureSingleTrailingNewline(newBody)

	// Locate the first frontmatter delimiter ("---" on its own line at the start)
	// followed by the closing delimiter.
	const delim = "---\n"
	if !strings.HasPrefix(rendered, delim) {
		return body
	}
	// Find the closing "---" line after the opening one.
	closeIdx := strings.Index(rendered[len(delim):], "\n"+delim)
	if closeIdx < 0 {
		// Try to match a closing delimiter that is the last line without a trailing newline.
		if strings.HasSuffix(rendered, "\n---") {
			return rendered + "\n" + body
		}
		return body
	}
	end := len(delim) + closeIdx + 1 + len(delim) // start of post-frontmatter region
	prefix := rendered[:end]
	if !strings.HasSuffix(prefix, "\n") {
		prefix += "\n"
	}
	return prefix + body
}

func ensureSingleTrailingNewline(s string) string {
	s = strings.TrimRight(s, "\n")
	return s + "\n"
}
