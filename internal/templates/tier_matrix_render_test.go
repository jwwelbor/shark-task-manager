package templates

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-E34-F08-002 / TC-002 — proves the rendered SIMPLE/STANDARD/COMPLEX
// routes in task_generation.md (artifact surface) and code_review.md/qa.md
// (gate surface) require exactly tier-matrix.md's matrix-defined artifacts
// and gates for that tier, no more and no fewer (spec.md AC-2).
//
// task_generation.md and code_review.md/qa.md carry their tier branching as
// free-form prose (there is no Go-template `{{if}}` on tier — an LLM agent
// reads the prose and picks a branch), so this test does not attempt to
// re-derive requirements by parsing that prose generically. Instead,
// T-E34-F08-002 added one deliberately structured, machine-parseable line to
// each file ("Required planning artifacts — SIMPLE: `a`, `b`; ...", "Required
// gates — SIMPLE: `a`, `b`; ..."). This test parses that line with the same
// backtick-token technique table_parser.go's extractArtifactFilenames uses,
// and diffs it against tier-matrix.md's own parsed table — never a
// hand-transcribed expected value — so a matrix edit that isn't mirrored in
// the prompt line fails here.

// tierMatrixRenderVars are the placeholder values needed to render
// feature/task_generation.md, feature/code_review.md, and feature/qa.md
// through the production renderer. Kept local to this package: the
// equivalent goldenVars() fixture in internal/cli/commands is unexported.
func tierMatrixRenderVars() map[string]string {
	return map[string]string{
		"id":          "E07-F01",
		"title":       "Sample feature",
		"file_path":   "docs/plan/E07/E07-F01/E07-F01.md",
		"epic_id":     "E07",
		"review_base": "docs/review/E07/E07-F01/",
	}
}

var backtickTokenPattern = regexp.MustCompile("`([^`]+)`")

// extractTierList parses a "<label> — SIMPLE: `a`, `b`; STANDARD: ...;
// COMPLEX: ... (canonical source: ...)" line out of rendered and returns the
// backtick-quoted tokens for the requested tier. The trailing "(canonical
// source: ...)" parenthetical (present on the COMPLEX/last segment) is
// stripped before splitting on ";" so it is never mistaken for part of that
// tier's own list.
func extractTierList(rendered, label, tier string) ([]string, error) {
	marker := label + " — "
	idx := strings.Index(rendered, marker)
	if idx == -1 {
		return nil, fmt.Errorf("line labeled %q not found in rendered output", label)
	}
	rest := rendered[idx+len(marker):]
	if nl := strings.IndexByte(rest, '\n'); nl != -1 {
		rest = rest[:nl]
	}
	if paren := strings.Index(rest, " (canonical source"); paren != -1 {
		rest = rest[:paren]
	}

	tierPrefix := tier + ":"
	for _, segment := range strings.Split(rest, ";") {
		segment = strings.TrimSpace(segment)
		if !strings.HasPrefix(segment, tierPrefix) {
			continue
		}
		var tokens []string
		for _, m := range backtickTokenPattern.FindAllStringSubmatch(segment, -1) {
			tokens = append(tokens, m[1])
		}
		return tokens, nil
	}
	return nil, fmt.Errorf("tier %q segment not found in %q line", tier, label)
}

func TestTC002_TierRoutesMatchTierMatrixExactly(t *testing.T) {
	tierMatrixBytes, err := sharkdata.ReadEmbedded("skills/quality/context/tier-matrix.md")
	require.NoError(t, err, "tier-matrix.md must exist in the embedded bundle")

	tierRows, err := sharkdata.ParseTierMatrixTable(tierMatrixBytes)
	require.NoError(t, err)
	byTier := make(map[string]sharkdata.TierRow, len(tierRows))
	for _, row := range tierRows {
		byTier[row.Tier] = row
	}
	require.Contains(t, byTier, "SIMPLE")
	require.Contains(t, byTier, "STANDARD")
	require.Contains(t, byTier, "COMPLEX")

	renderer, err := NewOrchestratorRenderer(t.TempDir())
	require.NoError(t, err)

	vars := tierMatrixRenderVars()
	renderedTaskGen, err := renderer.Render("feature/task_generation.md", vars)
	require.NoError(t, err)
	renderedCodeReview, err := renderer.Render("feature/code_review.md", vars)
	require.NoError(t, err)
	renderedQA, err := renderer.Render("feature/qa.md", vars)
	require.NoError(t, err)

	for _, tier := range []string{"SIMPLE", "STANDARD", "COMPLEX"} {
		t.Run(tier, func(t *testing.T) {
			wantArtifacts := byTier[tier].RequiredArtifacts
			gotArtifacts, err := extractTierList(renderedTaskGen, "Required planning artifacts", tier)
			require.NoError(t, err)
			assert.ElementsMatch(t, wantArtifacts, gotArtifacts,
				"task_generation.md's declared planning artifacts for %s must match tier-matrix.md exactly", tier)

			wantGates := byTier[tier].RequiredGates
			gotGatesCodeReview, err := extractTierList(renderedCodeReview, "Required gates", tier)
			require.NoError(t, err)
			assert.ElementsMatch(t, wantGates, gotGatesCodeReview,
				"code_review.md's declared gates for %s must match tier-matrix.md exactly", tier)

			gotGatesQA, err := extractTierList(renderedQA, "Required gates", tier)
			require.NoError(t, err)
			assert.ElementsMatch(t, wantGates, gotGatesQA,
				"qa.md's declared gates for %s must match tier-matrix.md exactly", tier)
		})
	}

	// Edge case (TC-002): SIMPLE must not require the separate-QA gate;
	// COMPLEX must require it.
	assert.NotContains(t, byTier["SIMPLE"].RequiredGates, "qa")
	assert.Contains(t, byTier["COMPLEX"].RequiredGates, "qa")
}
