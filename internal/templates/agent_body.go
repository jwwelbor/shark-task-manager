package templates

import (
	"fmt"
	"regexp"
	"strings"
)

// UnrenderedTokenError signals that an agent body rendering pass left a
// `<token>` placeholder unfilled — i.e., RenderAgentBody had no value for it
// in the vars map. Callers (notably `shark next`) treat this as a hard error
// because the agent receiving the prompt would otherwise see an unresolved
// placeholder as instructional text.
//
// AgentType is the agent file the body came from (e.g., "developer"), to
// make diagnostics actionable. Token is the literal surviving substring
// (e.g., "<task_id>").
type UnrenderedTokenError struct {
	AgentType string
	Token     string
}

func (e *UnrenderedTokenError) Error() string {
	if e.AgentType == "" {
		return fmt.Sprintf("unrendered placeholder %s left in agent body", e.Token)
	}
	return fmt.Sprintf("unrendered placeholder %s left in %q agent body", e.Token, e.AgentType)
}

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

// RenderAndLintAgentBody renders an agent body with RenderAgentBody, then
// fails fast on any surviving `<token>` placeholder. This is the canonical
// entry point for agent-body rendering in `shark next` and `shark run`: it
// pairs the silent-on-miss substitution with the post-render guard that
// originally lived in next.go, but scoped to the agent body itself so that
// action prompts (which legitimately contain `<...>` as instructional prose)
// are not falsely flagged.
//
// agentType is included in returned errors for diagnostics — it identifies
// which shipped agent file is missing a placeholder mapping.
func RenderAndLintAgentBody(body, agentType string, vars map[string]string) (string, error) {
	rendered := RenderAgentBody(body, vars)
	if tok, found := FirstUnrenderedToken(rendered); found {
		return rendered, &UnrenderedTokenError{AgentType: agentType, Token: tok}
	}
	return rendered, nil
}

// inlineCodeRe matches single-backtick inline code spans (markdown convention).
// We strip these before scanning for tokens so authors can write paths like
// `docs/plan/<epic>/<feature>/test.md` in prose without the guard tripping.
var inlineCodeRe = regexp.MustCompile("`[^`]*`")

// FirstUnrenderedToken returns the first `<token>` substring still present
// in s after all rendering passes have run, along with a bool indicating
// whether anything was found. Callers use this to fail loudly when a
// placeholder slipped through every rendering pass.
//
// Only the angle-bracket token shape is considered (lowercase, underscore-
// or dash-separated). Other `<…>` runs (HTML tags, comparisons, generics)
// are ignored.
//
// Tokens inside fenced code blocks (``` … ```) or inline code spans
// (` … `) are skipped — authors routinely write `<example>` inside markdown
// code as documentation, and treating those as failures would force a
// rewrite of every shipped agent file.
func FirstUnrenderedToken(s string) (string, bool) {
	// Fast path: no '<' anywhere means no token shape possible.
	if !strings.ContainsRune(s, '<') {
		return "", false
	}
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
// The function mutates the map in place and returns it for chaining.
func AugmentPlaceholderAliases(vars map[string]string) map[string]string {
	if vars == nil {
		return vars
	}

	// Canonical key aliases (entity-key shorthands). Placeholders may carry
	// either "<entity>_key" or "<entity>_id" depending on which generator
	// built the map; seed all shorthand variants from whichever is present.
	entityAliases := []struct {
		entity, keyVar, idVar string
	}{
		{"task", "task_key", "task_id"},
		{"epic", "epic_key", "epic_id"},
		{"feature", "feature_key", "feature_id"},
	}
	for _, e := range entityAliases {
		if v := firstNonEmpty(vars, e.keyVar, e.idVar); v != "" {
			setIfMissing(vars, e.entity, v)
			setIfMissing(vars, e.keyVar, v)
			setIfMissing(vars, e.idVar, v)
		}
	}

	// When the entity itself is an epic/feature/task, the generic "key"
	// placeholder is the entity's own key — seed the corresponding entity
	// alias from it so `<feature>` etc. resolve even on standalone calls
	// (e.g., `shark next E01-F01` where the placeholder generator emits
	// "feature_id" + "key" but not "feature_key").
	if v := vars["key"]; v != "" {
		for _, e := range entityAliases {
			if vars["entity_type"] == e.entity {
				setIfMissing(vars, e.entity, v)
				setIfMissing(vars, e.keyVar, v)
				setIfMissing(vars, e.idVar, v)
				break
			}
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
