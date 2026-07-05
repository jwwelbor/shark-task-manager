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
// stderr, so tests can assert the BUG-3/4 "[shark-stats] WARN" line is gone.
func captureStderrOutput(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// TestAnnotateUnresolvedPlaceholders_PopulatesField verifies BUG-3/4:
// unresolved placeholders are surfaced via the stable "unresolved_placeholders"
// JSON field instead of a stderr-only "[shark-stats] WARN" line, which used
// to corrupt --json consumers piping stdout.
func TestAnnotateUnresolvedPlaceholders_PopulatesField(t *testing.T) {
	resp := NextResponse{
		EntityKey:  "E07-F01-001",
		EntityType: "task",
		Status:     "in_progress",
		Action:     "spawn_agent",
		Prompt:     "Implement <task_id> per the spec.",
	}

	var stdout string
	stderr := captureStderrOutput(func() {
		annotated, tokens := annotateUnresolvedPlaceholders(resp)
		assert.Equal(t, []string{"<task_id>"}, tokens)
		assert.Equal(t, []string{"<task_id>"}, annotated.UnresolvedPlaceholders)

		stdout = capturingOutput(func() {
			require.NoError(t, outputNextJSON(annotated))
		})
	})

	assert.NotContains(t, stderr, "[shark-stats]",
		"unresolved placeholders must not be reported via a stderr WARN line anymore")

	var parsed NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed), "shark next --json stdout must parse as valid JSON")
	assert.Equal(t, []string{"<task_id>"}, parsed.UnresolvedPlaceholders)
}

// TestAnnotateUnresolvedPlaceholders_OmittedWhenFullyRendered verifies the
// field is omitted from JSON (not an empty array) when the prompt has no
// surviving placeholders, matching the existing resolved_via omitempty
// convention.
func TestAnnotateUnresolvedPlaceholders_OmittedWhenFullyRendered(t *testing.T) {
	resp := NextResponse{
		EntityKey:  "E07-F01-001",
		EntityType: "task",
		Status:     "in_progress",
		Action:     "spawn_agent",
		Prompt:     "Implement the fully-rendered task.",
	}

	annotated, tokens := annotateUnresolvedPlaceholders(resp)
	assert.Empty(t, tokens)

	stdout := capturingOutput(func() {
		require.NoError(t, outputNextJSON(annotated))
	})
	assert.NotContains(t, stdout, "unresolved_placeholders")
}
