package templates

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
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

// limitExitGateItem names the literals of REQ-F-017(b), the edited-code gate.
var limitExitGateItem = []string{
	"Every code file you edited is within the resolved limits",
	"measured in full",
	"overages that predate the change",
	"source label",
	"limit exception",
}

// envelopeExitGate returns a prompt's own exit gate items plus REQ-F-017
// (a), (b) and (d). Item (c), the task Scope list, applies to tasks only.
func envelopeExitGate(own ...string) []string {
	own = append(own, limitExitGateItem...)
	return append(own,
		"for each AC ID",
		"Self-review is recorded with no unresolved finding",
	)
}

var developmentPrompts = []developmentPrompt{
	{
		name:        "task/development.md",
		summaryLead: taskSummaryLead,
		exitGateItems: append([]string{
			// Pre-existing items that must remain.
			"All test cases from feature test-plan.md",
			"Each test names its TC-ID and calls its Caller-Path Contract entrypoint",
			"Quality gate passes",
			// REQ-F-017 (a), (c) and (d); (b) is limitExitGateItem.
			"for each AC ID",
			taskScopeExitGateItem,
			"files a limit refactor creates",
			"Self-review is recorded with no unresolved finding",
		}, limitExitGateItem...),
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
	taskScopeExitGateItem = "`## Scope` \"Files to modify\""
	// retiredLimitWording is the round-1 gate that the edited-code rule replaced.
	retiredLimitWording = "Every new or changed function and file"
)

// promptContractLiterals are the spec's prompt contract literals (REQ-F-033(b)).
var promptContractLiterals = []string{
	"If every source is absent or contains only a placeholder",
	"project bootstrap",
	"Missing quality limits never block.",
	"Run the self-review checklist in the Craft Rubric below against your own diff",
	"before you change any code, read the latest transition note and the gate report that note cites",
	"Every code file you edited is within the resolved limits",
	"measured in full",
	"consult the architect",
}

// readListStandards are the REQ-F-011 files each READ list names exactly once.
var readListStandards = []string{
	"CLAUDE.md",
	"AGENTS.md",
	"docs/architecture/coding-standards.md",
	"patterns-catalog.md",
	"tech-stack.md",
}

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

func readEmbeddedPrompt(t *testing.T, name string) string {
	t.Helper()
	body, err := sharkdata.ReadEmbedded("prompts/" + name)
	require.NoError(t, err, "embedded prompt %s must exist", name)
	return string(body)
}

// forEachDevelopmentRender runs check as one subtest per family prompt, on the
// production render of that prompt.
func forEachDevelopmentRender(t *testing.T, check func(t *testing.T, prompt developmentPrompt, out string)) {
	t.Helper()
	renderer := newCanonicalOrchestratorRenderer(t)
	for _, prompt := range developmentPrompts {
		t.Run(prompt.name, func(t *testing.T) {
			check(t, prompt, renderDevelopmentPrompt(t, renderer, prompt.name))
		})
	}
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
	forEachDevelopmentRender(t, func(t *testing.T, _ developmentPrompt, out string) {
		assertMarkersOnce(t, out)
		assert.Contains(t, out, genericDefaultLabel)
		assertNoUnresolvedDirectives(t, out)
		assert.NotContains(t, out, "09-test-plan.md")
	})
}

// The rubric cites a skill-relative path, so a worker needs the note on how to
// read it before the rubric text starts.
func TestE31F01_TC007_QualitySkillNotePrecedesRubric(t *testing.T) {
	forEachDevelopmentRender(t, func(t *testing.T, _ developmentPrompt, out string) {
		note := strings.Index(out, "Bundled `quality` skill")
		require.NotEqual(t, -1, note, "the bundled quality skill note must render")
		assert.Less(t, note, strings.Index(out, craftRubricMarker))
	})
}

// Asserted on the source template: the included craft rubric uses some of the
// same words, so a render could satisfy them without the prompt's own rule.
func TestE31F01_TC008_SourceContractLiterals(t *testing.T) {
	for _, prompt := range developmentPrompts {
		t.Run(prompt.name, func(t *testing.T) {
			source := readEmbeddedPrompt(t, prompt.name)
			for _, literal := range promptContractLiterals {
				t.Run(literal, func(t *testing.T) {
					assert.Contains(t, source, literal)
				})
			}
		})
	}
}

func TestE31F01_TC008_FamilyPromptRules(t *testing.T) {
	forEachDevelopmentRender(t, func(t *testing.T, prompt developmentPrompt, out string) {
		read := promptSection(t, out, "READ")
		for _, file := range readListStandards {
			assert.Equal(t, 1, strings.Count(read, file), "READ must name %s exactly once", file)
		}
		assertContainsAll(t, read, []string{"Quality Limits", "Limit exceptions", "Testing Standards"})
		assert.NotContains(t, out, "CLAUDE.md for coding standards")
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

func TestE31F01_TC009_ExitGate(t *testing.T) {
	forEachDevelopmentRender(t, func(t *testing.T, prompt developmentPrompt, out string) {
		gate := promptSection(t, out, "EXIT GATE:")
		assertContainsAll(t, gate, prompt.exitGateItems)
		if prompt.name != "task/development.md" {
			assert.NotContains(t, gate, taskScopeExitGateItem, "the Scope list item applies to tasks only")
		}
		assert.NotContains(t, out, retiredLimitWording)
		assert.NotContains(t, readEmbeddedPrompt(t, prompt.name), retiredLimitWording)
	})
}

func TestE31F01_TC021_LimitExceptionAndConsultation(t *testing.T) {
	forEachDevelopmentRender(t, func(t *testing.T, prompt developmentPrompt, out string) {
		assert.Contains(t, readEmbeddedPrompt(t, prompt.name), "consult the architect")
		assert.Contains(t, out, "## Limit exceptions\n", "the exception rule must be inlined")
	})
}

// Change and tech-debt results have a 1000-byte summary cap, so their evidence
// detail belongs in outer evidence entries (D12). Task and bug are unchanged.
func TestE31F01_TC022_EvidenceRouting(t *testing.T) {
	routed := map[string]bool{
		"change/development.md":    true,
		"tech_debt/triaged.md":     true,
		"tech_debt/in_progress.md": true,
	}
	forEachDevelopmentRender(t, func(t *testing.T, prompt developmentPrompt, out string) {
		gate := promptSection(t, out, "EXIT GATE:")
		if !routed[prompt.name] {
			assert.NotContains(t, gate, "`evidence[]`")
			return
		}
		assert.Contains(t, gate, "`evidence[]`")
		assert.Contains(t, promptSection(t, out, "SUMMARY LINE:"), "`evidence[]`")
		assert.NotContains(t, gate, "compactly")
		assert.NotContains(t, gate, "stays within its size limit")
	})
}

func TestE31F01_TC023_TDDPartialAndPrecedence(t *testing.T) {
	assert.NotContains(t, readEmbeddedPrompt(t, "_partials/_tdd_process.md"), "the included skills")
	forEachDevelopmentRender(t, func(t *testing.T, _ developmentPrompt, out string) {
		assert.Contains(t, strings.ToLower(out), "one test per tc assigned to this work item")
		assert.Contains(t, out, "any included skill")
		assertPrecedenceLimitsFailingTests(t, out)
	})
}

func assertPrecedenceLimitsFailingTests(t *testing.T, rendered string) {
	t.Helper()
	found := 0
	for _, line := range strings.Split(rendered, "\n") {
		if !strings.HasPrefix(line, "PRECEDENCE:") {
			continue
		}
		found++
		assertContainsAll(t, line, []string{"behavior-changing", "preserve the code and history"})
	}
	assert.NotZero(t, found, "rendered prompt must carry a PRECEDENCE line")
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
