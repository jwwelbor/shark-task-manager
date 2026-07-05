package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderrOutput redirects stderr for the duration of fn and returns
// what was written. Mirrors capturingOutput (analytics_test.go) but for
// stderr.
func captureStderrOutput(fn func()) string {
	old := os.Stderr
	defer func() { os.Stderr = old }()
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// TestAnnotateUnresolvedPlaceholders_PopulatesField verifies BUG-3/4:
// unresolved placeholders are surfaced via the stable "unresolved_placeholders"
// JSON field, in addition to (not instead of) the stderr "[shark-stats] WARN"
// line required by E32-F07 REQ-F-003 — stdout stays valid JSON either way
// since stderr is a separate stream.
func TestAnnotateUnresolvedPlaceholders_PopulatesField(t *testing.T) {
	resp := NextResponse{
		EntityKey:  "E07-F01-001",
		EntityType: "task",
		Status:     "in_progress",
		Action:     "spawn_agent",
		Prompt:     "Implement <task_id> per the spec.",
	}

	annotated := annotateUnresolvedPlaceholders(resp)
	assert.Equal(t, []string{"<task_id>"}, annotated.UnresolvedPlaceholders)

	var stdout string
	stderr := captureStderrOutput(func() {
		warnUnresolvedPlaceholdersToStderr(annotated)
		stdout = capturingOutput(func() {
			require.NoError(t, outputNextJSON(annotated))
		})
	})

	assert.Contains(t, stderr, "[shark-stats] WARN: E07-F01-001 has 1 unresolved placeholders",
		"E32-F07 REQ-F-003 requires the stderr WARN line unconditionally, not just the JSON field")

	var parsed NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed), "shark next --json stdout must parse as valid JSON")
	assert.Equal(t, []string{"<task_id>"}, parsed.UnresolvedPlaceholders)
}

// TestAnnotateUnresolvedPlaceholders_OmittedWhenFullyRendered verifies the
// field is omitted from JSON (not an empty array) when the prompt has no
// surviving placeholders, matching the existing resolved_via omitempty
// convention, and that no stderr WARN is emitted.
func TestAnnotateUnresolvedPlaceholders_OmittedWhenFullyRendered(t *testing.T) {
	resp := NextResponse{
		EntityKey:  "E07-F01-001",
		EntityType: "task",
		Status:     "in_progress",
		Action:     "spawn_agent",
		Prompt:     "Implement the fully-rendered task.",
	}

	annotated := annotateUnresolvedPlaceholders(resp)
	assert.Empty(t, annotated.UnresolvedPlaceholders)

	stderr := captureStderrOutput(func() {
		warnUnresolvedPlaceholdersToStderr(annotated)
	})
	assert.Empty(t, stderr)

	stdout := capturingOutput(func() {
		require.NoError(t, outputNextJSON(annotated))
	})
	assert.NotContains(t, stdout, "unresolved_placeholders")
}

// TestUnrenderedTokens_DedupesRepeatedPlaceholder verifies a placeholder
// referenced more than once in one prompt is reported once in
// unresolved_placeholders, not once per occurrence.
func TestUnrenderedTokens_DedupesRepeatedPlaceholder(t *testing.T) {
	resp := NextResponse{
		EntityKey: "E07-F01-001",
		Prompt:    "Task <task_id>: implement <task_id> per the spec.",
	}

	annotated := annotateUnresolvedPlaceholders(resp)
	assert.Equal(t, []string{"<task_id>"}, annotated.UnresolvedPlaceholders)
}
