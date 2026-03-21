// Package runner provides the AgentDispatcher interface and related types for
// invoking external AI agents (Claude, Codex, etc.) as part of the E22 run loop.
// This file provides output formatting helpers for non-verbose CLI display.
package runner

import "strings"

// TruncateOutput shortens s for single-line display. It replaces newlines with
// spaces and, if the result exceeds maxLen runes, truncates it and appends "…".
//
// Behaviour:
//   - Empty input returns "".
//   - Input with no newlines and len <= maxLen is returned unchanged (no allocation).
//   - Newlines are replaced with a single space each.
//   - If the processed string exceeds maxLen, it is truncated at maxLen-1 runes
//     and "…" is appended, giving a final length of exactly maxLen runes.
//
// maxLen must be >= 1; values < 1 are treated as 1.
func TruncateOutput(s string, maxLen int) string {
	if s == "" {
		return ""
	}
	if maxLen < 1 {
		maxLen = 1
	}

	// Flatten newlines to spaces.
	flat := strings.ReplaceAll(s, "\n", " ")
	flat = strings.ReplaceAll(flat, "\r", "")

	runes := []rune(flat)
	if len(runes) <= maxLen {
		return flat
	}

	// Truncate and append ellipsis (one rune, not three dots).
	return string(runes[:maxLen-1]) + "…"
}
