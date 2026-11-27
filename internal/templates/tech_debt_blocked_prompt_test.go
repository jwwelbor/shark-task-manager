package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockedPromptCases are the two reason variants of a pause prompt.
var blockedPromptCases = []struct {
	name   string
	reason string
}{
	{name: "with blocked reason", reason: "waiting for an architect decision"},
	{name: "without blocked reason"},
}

// blockedPromptEntities are the entities whose pause prompts mirror each other,
// keyed by prompt directory, with the noun each one opens with.
var blockedPromptEntities = []struct{ dir, noun, id string }{
	{dir: "bug", noun: "Bug", id: "B001"},
	{dir: "change", noun: "Change", id: "CC-001"},
	{dir: "tech_debt", noun: "Tech debt", id: "TD-001"},
}

func renderBlockedPrompt(t *testing.T, renderer *OrchestratorRenderer, dir, id, reason string) string {
	t.Helper()
	out, err := renderer.Render(dir+"/blocked.md", map[string]string{"id": id, "blocked_reason": reason})
	require.NoError(t, err)
	return out
}

// The bug and tech-debt pause prompts are the change-card pause prompt with the
// entity noun swapped, with and without a blocked reason.
func TestE31F01_TC020f_TechDebtBlockedPromptMirrorsChange(t *testing.T) {
	renderer := newCanonicalOrchestratorRenderer(t)

	for _, tc := range blockedPromptCases {
		t.Run(tc.name, func(t *testing.T) {
			change := renderBlockedPrompt(t, renderer, "change", "TD-001", tc.reason)
			for _, entity := range blockedPromptEntities {
				t.Run(entity.dir, func(t *testing.T) {
					got := renderBlockedPrompt(t, renderer, entity.dir, "TD-001", tc.reason)

					assert.Equal(t, strings.Replace(change, "Change TD-001", entity.noun+" TD-001", 1), got)
					assert.Contains(t, got, entity.noun+" TD-001 is blocked.")
					assert.Equal(t, tc.reason != "", strings.Contains(got, "Blocked reason: "+tc.reason))
				})
			}
		})
	}
}

// A plain advance from a parking step does not return to the parked step, so
// the prompt has to send the reader to the history and a direct status set.
func TestE31F01_TC027_BlockedPromptsResumeTheParkedStep(t *testing.T) {
	renderer := newCanonicalOrchestratorRenderer(t)

	for _, entity := range blockedPromptEntities {
		t.Run(entity.dir, func(t *testing.T) {
			for _, tc := range blockedPromptCases {
				t.Run(tc.name, func(t *testing.T) {
					out := renderBlockedPrompt(t, renderer, entity.dir, entity.id, tc.reason)
					assertResumeInstructions(t, out, entity.id)
				})
			}
		})
	}
}

func assertResumeInstructions(t *testing.T, rendered, id string) {
	t.Helper()
	text := strings.Join(strings.Fields(rendered), " ")
	assert.Contains(t, text, "shark status history "+id)
	assert.Contains(t, text, "shark status set "+id+" <parked step> --reason")
	for _, banned := range []string{"shark status advance", "--force", "<no value>"} {
		assert.NotContains(t, rendered, banned)
	}
}
