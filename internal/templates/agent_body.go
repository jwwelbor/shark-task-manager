package templates

import (
	"regexp"
	"strings"
)

// agentBodyTokenRe matches `<token>` placeholders embedded in agent body
// files. Tokens use a lowercase, underscore- or dash-separated name. The
// pattern intentionally rejects whitespace, uppercase, and punctuation so
// HTML-like substrings (e.g. `<br>`, `<div class="x">`) never match by
// accident.
var agentBodyTokenRe = regexp.MustCompile(`<([a-z][a-z0-9_-]*)>`)

// RenderAgentBody substitutes `<token>` placeholders in body with values from
// vars. Tokens are matched against the placeholder map two ways:
//
//  1. The literal token name (e.g. `<task_id>` → vars["task_id"]).
//  2. The dash-to-underscore form (e.g. `<task-id>` → vars["task_id"]).
//
// The dual-form lookup lets agent files use kebab-case (`<task-id>`) while
// the engine-side placeholder generator uses snake_case (`task_id`), keeping
// both author ergonomics and the placeholder API consistent without
// duplication. Tokens with no matching key are left untouched — callers
// should follow up with FirstUnrenderedToken to fail loudly on misses.
func RenderAgentBody(body string, vars map[string]string) string {
	if body == "" || len(vars) == 0 {
		return body
	}
	return agentBodyTokenRe.ReplaceAllStringFunc(body, func(match string) string {
		token := match[1 : len(match)-1]
		if v, ok := vars[token]; ok {
			return v
		}
		alt := strings.ReplaceAll(token, "-", "_")
		if v, ok := vars[alt]; ok {
			return v
		}
		return match // leave unrendered so the post-render guard catches it
	})
}

// inlineCodeRe matches single-backtick inline code spans (markdown convention).
// We strip these before scanning for tokens so authors can write paths like
// `docs/plan/<epic>/<feature>/test.md` in prose without the guard tripping.
var inlineCodeRe = regexp.MustCompile("`[^`]*`")

// FirstUnrenderedToken returns the first `<token>` substring still present
// in s after all rendering passes have run, along with a bool indicating
// whether anything was found. Callers use this to fail loudly when a
// placeholder slipped through every rendering pass — silent pass-through
// is the failure mode the 2026-05-11 trial hit.
//
// Only the angle-bracket token shape is considered (lowercase, underscore-
// or dash-separated). Other `<…>` runs (HTML tags, comparisons, generics)
// are ignored.
//
// Tokens inside fenced code blocks (``` … ```) or inline code spans
// (` … `) are skipped — authors routinely write `<example>` inside markdown
// code as documentation, and treating those as failures would force a
// rewrite of every shipped agent file. Real misses appear in body prose
// outside code regions.
func FirstUnrenderedToken(s string) (string, bool) {
	scanned := stripFencedCodeBlocks(s)
	scanned = inlineCodeRe.ReplaceAllString(scanned, "")
	match := agentBodyTokenRe.FindString(scanned)
	return match, match != ""
}

// stripFencedCodeBlocks returns s with all triple-backtick fenced code
// blocks replaced by an equivalent number of newlines so line offsets are
// preserved for any downstream diagnostics. Fences are opened by a line
// starting with ``` (optionally followed by a language tag) and closed by
// a line starting with ```. Unclosed fences are dropped from the
// open-fence line to end-of-string.
func stripFencedCodeBlocks(s string) string {
	var out strings.Builder
	inFence := false
	for _, line := range splitKeepEOL(s) {
		trimmed := strings.TrimSpace(line)
		isFence := strings.HasPrefix(trimmed, "```")
		switch {
		case isFence && !inFence:
			inFence = true
			out.WriteString("\n")
		case isFence && inFence:
			inFence = false
			out.WriteString("\n")
		case inFence:
			// Replace code-block line with an empty line so line numbers
			// downstream still line up.
			out.WriteString("\n")
		default:
			out.WriteString(line)
		}
	}
	return out.String()
}

// splitKeepEOL splits s into lines while preserving the trailing newline
// on each line so reassembly is byte-exact.
func splitKeepEOL(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// AugmentPlaceholderAliases enriches vars with the alias keys the agent
// body templates expect. The placeholder generators emit canonical
// underscore-form keys (task_key, epic_key, feature_key, title, file_path,
// status, …); this helper adds the additional names that the agent files
// — and the bare-`<entity>` template tokens — use.
//
// New aliases are cheap; one source of truth for both the underscore and
// dash forms means we never re-introduce the trial's `<task-id>` ≠
// `task_id` mismatch.
//
// The function mutates the map in place and returns it for chaining.
func AugmentPlaceholderAliases(vars map[string]string) map[string]string {
	if vars == nil {
		return vars
	}

	// Canonical key aliases (entity-key shorthands). Placeholders may carry
	// either the canonical "<entity>_key" or the legacy "<entity>_id" form
	// (or both) depending on which generator built the map. We seed all
	// shorthand variants from whichever is present.
	if v := firstNonEmpty(vars, "task_key", "task_id"); v != "" {
		setIfMissing(vars, "task", v)
		setIfMissing(vars, "task_key", v)
		setIfMissing(vars, "task_id", v)
	}
	if v := firstNonEmpty(vars, "epic_key", "epic_id"); v != "" {
		setIfMissing(vars, "epic", v)
		setIfMissing(vars, "epic_key", v)
		setIfMissing(vars, "epic_id", v)
	}
	if v := firstNonEmpty(vars, "feature_key", "feature_id"); v != "" {
		setIfMissing(vars, "feature", v)
		setIfMissing(vars, "feature_key", v)
		setIfMissing(vars, "feature_id", v)
	}

	// When the entity itself is an epic/feature/task, the generic "key"
	// placeholder is the entity's own key — seed the corresponding entity
	// alias from it so `<feature>` etc. resolve even on standalone calls
	// (e.g., `shark next E01-F01` where the placeholder generator emits
	// "feature_id" + "key" but not "feature_key").
	if v := vars["key"]; v != "" {
		switch vars["entity_type"] {
		case "epic":
			setIfMissing(vars, "epic", v)
			setIfMissing(vars, "epic_key", v)
			setIfMissing(vars, "epic_id", v)
		case "feature":
			setIfMissing(vars, "feature", v)
			setIfMissing(vars, "feature_key", v)
			setIfMissing(vars, "feature_id", v)
		case "task":
			setIfMissing(vars, "task", v)
			setIfMissing(vars, "task_key", v)
			setIfMissing(vars, "task_id", v)
		}
		setIfMissing(vars, "id", v)
	}

	return vars
}

// firstNonEmpty returns the first non-empty value found in vars for the
// listed keys, or "" if none of the keys are populated.
func firstNonEmpty(vars map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := vars[k]; ok && v != "" {
			return v
		}
	}
	return ""
}

func setIfMissing(m map[string]string, k, v string) {
	if _, exists := m[k]; !exists {
		m[k] = v
	}
}
