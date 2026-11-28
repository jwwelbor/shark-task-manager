package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The text of the bug DECISION paragraph around its `blocked` clause, as it
// stood at base d9427496. Only that clause may change.
const (
	bugDecisionBeforeBlocked = "DECISION: pass only when the accepted plan, coverage obligations, and exit gate are satisfied. " +
		"Use review_plan for a valid plan whose approval is missing or stale; use needs_research for changed diagnosis, " +
		"replan for a material plan update, fail for implementation rework, and blocked "
	bugDecisionAfterBlocked = ". For route_rework outcomes leave kickbacks empty, keep the summary concise, " +
		"and retain findings in the structured findings. The parent owns transitions."
)

// Without the wider clause a worker that must park would pick `fail`, which
// routes straight back to development.
func TestE31F01_TC011_BugDecisionWidensOnlyTheBlockedClause(t *testing.T) {
	out := renderDevelopmentPrompt(t, newCanonicalOrchestratorRenderer(t), "bug/development.md")
	decision := "DECISION:" + promptSection(t, out, "DECISION:")

	require.True(t, strings.HasPrefix(decision, bugDecisionBeforeBlocked), "outcomes before blocked changed: %s", decision)
	require.True(t, strings.HasSuffix(decision, bugDecisionAfterBlocked), "text after blocked changed: %s", decision)

	clause := strings.TrimSuffix(strings.TrimPrefix(decision, bugDecisionBeforeBlocked), bugDecisionAfterBlocked)
	for _, reason := range []string{
		"unavailable external evidence",
		"missing quality gate",
		"architect consultation",
		"unclear requirements",
	} {
		t.Run(reason, func(t *testing.T) {
			assert.Contains(t, clause, reason)
		})
	}
}

// The prompt linter reads the last outer-evidence example as the required-kinds
// list, so the added routing text must not introduce another one.
func TestE31F01_TC011_BugPromptKeepsOneRequiredEvidenceExample(t *testing.T) {
	source := readEmbeddedPrompt(t, "bug/development.md")
	assert.Equal(t, 1, strings.Count(source, `"evidence": [`))
	assert.Equal(t, 2, strings.Count(source, `"kind": "`))
}

// The added entries use their own kinds, which must not collide with the kinds
// the runner admits exactly once.
func TestE31F01_TC022_BugEvidenceKinds(t *testing.T) {
	out := renderDevelopmentPrompt(t, newCanonicalOrchestratorRenderer(t), "bug/development.md")
	summary := strings.Join(strings.Fields(promptSection(t, out, "SUMMARY LINE:")), " ")
	for _, literal := range []string{"ac_evidence", "limit_measurement", "self_review", "alongside"} {
		t.Run(literal, func(t *testing.T) {
			assert.Contains(t, summary, literal)
		})
	}

	source := readEmbeddedPrompt(t, "bug/development.md")
	routing := strings.Index(source, "`ac_evidence`")
	require.NotEqual(t, -1, routing, "the routing text must name its kinds")
	assert.Less(t, routing, strings.Index(source, "REQUIRED OUTER EVIDENCE"))
}
