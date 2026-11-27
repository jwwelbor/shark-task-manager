package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// developmentPrompt is one row of the development prompt family (E31-F01).
// Each row names the literals that are contracts for that prompt.
type developmentPrompt struct {
	name string
	// summaryLead is the instruction that places the completion summary line:
	// envelope prompts lead gate_result.summary with it.
	summaryLead   string
	exitGateItems []string
}

const (
	taskSummaryLead     = "Begin the summary with the line " + completionSummaryLine
	envelopeSummaryLead = "Begin `gate_result.summary` with the line " + completionSummaryLine
)

// envelopeExitGate returns a prompt's own exit gate items plus REQ-F-017
// (a), (b) and (d). Item (c), the task Scope list, applies to tasks only.
func envelopeExitGate(own ...string) []string {
	return append(own,
		"for each AC ID",
		"source label",
		"Self-review is recorded with no unresolved finding",
	)
}

var developmentPrompts = []developmentPrompt{
	{
		name:        "task/development.md",
		summaryLead: taskSummaryLead,
		exitGateItems: []string{
			// Pre-existing items that must remain.
			"All test cases from feature test-plan.md",
			"Each test names its TC-ID and calls its Caller-Path Contract entrypoint",
			"Quality gate passes",
			// REQ-F-017 (a) to (d).
			"for each AC ID",
			"source label",
			"`## Scope` \"Files to modify\"",
			"Self-review is recorded with no unresolved finding",
		},
	},
	{
		name:        "bug/development.md",
		summaryLead: envelopeSummaryLead,
		exitGateItems: envelopeExitGate(
			"Regression test proves the bug is fixed",
			"Quality gate passes",
			"Every applicable decision constraint has passing verification evidence",
			"No unrelated changes",
		),
	},
	{
		name:        "change/development.md",
		summaryLead: envelopeSummaryLead,
		exitGateItems: envelopeExitGate(
			"Change implemented per description",
			"Tests pass",
			"Quality gate passes",
		),
	},
	{
		name:        "tech_debt/triaged.md",
		summaryLead: envelopeSummaryLead,
		exitGateItems: envelopeExitGate(
			"Tech debt is resolved",
			"Quality gate passes",
			"No unrelated changes",
		),
	},
	{
		name:        "tech_debt/in_progress.md",
		summaryLead: envelopeSummaryLead,
		exitGateItems: envelopeExitGate(
			"Tech debt is resolved",
			"Quality gate passes",
			"No unrelated changes",
		),
	},
}

const (
	craftRubricMarker     = "# Craft Rubric\n"
	tddProcessMarker      = "TDD PROCESS:"
	genericDefaultLabel   = "generic default — not declared in local standards"
	completionSummaryLine = "`ACs verified: <ids>; red-first: <n>/<m>; commit: <sha>`"
)

func renderDevelopmentPrompt(t *testing.T, renderer *OrchestratorRenderer, name string) string {
	t.Helper()
	out, err := renderer.Render(name, map[string]string{
		"id":        "E01-F01-001",
		"title":     "Test Task",
		"file_path": "docs/plan/task.md",
		"category":  "code-quality",
		"severity":  "medium",
	})
	require.NoError(t, err, "render %s", name)
	return out
}

// promptSection returns the paragraph that starts at heading, up to the next
// blank line.
func promptSection(t *testing.T, rendered, heading string) string {
	t.Helper()
	_, after, found := strings.Cut(rendered, "\n"+heading)
	require.True(t, found, "rendered prompt must contain a %q section", heading)
	section, _, _ := strings.Cut(after, "\n\n")
	return section
}

func assertContainsAll(t *testing.T, text string, wants []string) {
	t.Helper()
	for _, want := range wants {
		assert.Contains(t, text, want)
	}
}

func assertMarkersOnce(t *testing.T, rendered string) {
	t.Helper()
	assert.Equal(t, 1, strings.Count(rendered, craftRubricMarker), "craft rubric must render exactly once")
	assert.Equal(t, 1, strings.Count(rendered, tddProcessMarker), "TDD process partial must render exactly once")
}

func assertNoUnresolvedDirectives(t *testing.T, rendered string) {
	t.Helper()
	for _, bad := range []string{"{{include:", "{{template", "<no value>"} {
		assert.NotContains(t, rendered, bad)
	}
}

func TestE31F01_TC007_RenderedDevelopmentPrompt(t *testing.T) {
	renderer := newCanonicalOrchestratorRenderer(t)
	for _, prompt := range developmentPrompts {
		t.Run(prompt.name, func(t *testing.T) {
			out := renderDevelopmentPrompt(t, renderer, prompt.name)
			assertMarkersOnce(t, out)
			assert.Contains(t, out, genericDefaultLabel)
			assertNoUnresolvedDirectives(t, out)
			assert.NotContains(t, out, "09-test-plan.md")
		})
	}
}

func TestE31F01_TC008_FamilyPromptRules(t *testing.T) {
	renderer := newCanonicalOrchestratorRenderer(t)
	for _, prompt := range developmentPrompts {
		t.Run(prompt.name, func(t *testing.T) {
			out := renderDevelopmentPrompt(t, renderer, prompt.name)
			assertContainsAll(t, promptSection(t, out, "READ"), []string{
				"CLAUDE.md",
				"AGENTS.md",
				"docs/architecture/coding-standards.md",
				"patterns-catalog.md",
				"tech-stack.md",
				"Quality Limits",
				"Testing Standards",
			})
			assertMarkersOnce(t, out)
			assertContainsAll(t, out, []string{
				"RESOLVE THE QUALITY GATE",
				"REOPENED OR REWORK DISPATCH:",
				"SELF-REVIEW",
				prompt.summaryLead,
			})
			assertNoUnresolvedDirectives(t, out)
		})
	}
}

func TestE31F01_TC009_ExitGate(t *testing.T) {
	renderer := newCanonicalOrchestratorRenderer(t)
	for _, prompt := range developmentPrompts {
		t.Run(prompt.name, func(t *testing.T) {
			out := renderDevelopmentPrompt(t, renderer, prompt.name)
			assertContainsAll(t, promptSection(t, out, "EXIT GATE:"), prompt.exitGateItems)
		})
	}
}

// TC-019 renders from the checked-in bundle and from an empty prompts
// directory, which falls back to the embedded bundle: no override and no
// project documents in either case.
func TestE31F01_TC019_ProjectWithoutStandardsRendersDefaults(t *testing.T) {
	for _, dir := range []string{canonicalPromptsDir, t.TempDir()} {
		renderer, err := NewOrchestratorRenderer(dir)
		require.NoError(t, err)
		for _, prompt := range developmentPrompts {
			t.Run(prompt.name, func(t *testing.T) {
				out := renderDevelopmentPrompt(t, renderer, prompt.name)
				assert.Contains(t, out, genericDefaultLabel)
				assertNoUnresolvedDirectives(t, out)
			})
		}
	}
}
