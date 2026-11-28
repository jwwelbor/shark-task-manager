package sharkdata_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collapsed joins every run of whitespace to one space, so line wrapping cannot
// hide a contract literal.
func collapsed(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func assertSliceContains(t *testing.T, slice string, literals ...string) {
	t.Helper()
	text := collapsed(slice)
	for _, literal := range literals {
		t.Run(literal, func(t *testing.T) {
			assert.Contains(t, text, literal)
		})
	}
}

// A project that limits only some languages must not have its other files held
// to the generic defaults under the edited-code rule.
func TestE31F01_TC005_CraftRubricHonorsALanguageScope(t *testing.T) {
	rubric := embeddedText(t, craftRubricPath)
	scope := section(t, rubric, "\n## Scope of the limits\n", "\n## ")
	assertSliceContains(t, scope, "only to named languages", "every code file is")
}

// The consultation decision is a decision note, so the re-dispatched author has
// to be told to read the notes.
func TestE31F01_TC005_CraftRubricRedispatchReadsDecisionNotes(t *testing.T) {
	rubric := embeddedText(t, craftRubricPath)
	consultation := section(t, rubric, "\n## Architect consultation\n", "\n## ")
	_, bullet, found := strings.Cut(consultation, "\n- The re-dispatched author")
	require.True(t, found, "missing the re-dispatch bullet")
	bullet, _, _ = strings.Cut(bullet, "\n- ")
	assertSliceContains(t, bullet, "and the work item's decision notes")
}

// A fallback reviewer sees only its prompt contents, so they must carry the
// rubric and both ways to read it.
func TestE31F01_TC013_DeepReviewFallbackPromptCarriesCraftRubric(t *testing.T) {
	skill := deepReviewText(t, "SKILL.md")
	runner := section(t, skill, "\n## Runner 3", "\n## Runner 4")
	assertSliceContains(t, runner,
		"craft-rubric.md",
		"shark skill get quality context/craft-rubric.md",
		craftRubricPath,
	)
}
