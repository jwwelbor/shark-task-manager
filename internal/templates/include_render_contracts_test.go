package templates

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIncludeResolverWithEmbed_StateSpaceCoverageRenders verifies that the
// checked-in state-space-coverage.md workflow file (E34-F07) renders cleanly
// through the production renderer and carries every one of the six required
// sections (spec.md REQ-F-001-004/006/007), asserting each section's required
// substantive clauses per test-plan.md TC-001 — not just heading presence.
func TestIncludeResolverWithEmbed_StateSpaceCoverageRenders(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")

	out, err := r.Resolve("{{include: skills/quality/workflows/state-space-coverage.md}}")
	require.NoError(t, err, "state-space-coverage.md must render through the production renderer with no errors")

	// All six required section headings (REQ-F-001-004/006/007).
	for _, section := range []string{
		"## Closed lifecycle tables",
		"## Technique selection from state shape",
		"## Dependency discovery by interaction and caller path",
		"## Shipped consumer re-verification",
		"## I-04 propagation",
		"## Design divergence",
	} {
		assert.Contains(t, out, section, "state-space-coverage.md must contain section %q", section)
	}

	// Closed lifecycle tables: the detection heuristic and all seven required
	// table columns (TC-001 clause 1).
	assert.Contains(t, out, "behavior-bearing", "must state the lifecycle-field detection heuristic")
	for _, column := range []string{
		"value", "meaning", "entry transition", "exit transition",
		"terminal", "invalid-transition", "failure/recovery",
	} {
		assert.Contains(t, out, column, "closed-table section must name column %q", column)
	}

	// Technique selection: trigger condition and technique name (TC-001 clause 2).
	assert.Contains(t, out, "state-transition", "must name the state-transition/decision-table technique")
	assert.Contains(t, out, "decision-table", "must name the decision-table technique")

	// Dependency discovery: priority-ordered sources and per-axis rationale
	// recording (TC-001 clause 3).
	assert.Contains(t, out, "interaction-map", "must name interaction-map rows as a discovery source")
	assert.Contains(t, out, "Caller-Path Contract", "must name production caller paths via the Caller-Path Contract concept")
	assert.Contains(t, out, "persistence reader", "must name persistence readers as a discovery source")
	assert.Contains(t, out, "inclusion/exclusion rationale", "must require per-axis inclusion/exclusion rationale")

	// Shipped consumer re-verification: all four required fields (TC-001 clause 4).
	for _, field := range []string{
		"caller path", "owning feature key", "affected AC ID", "regression-test pointer",
	} {
		assert.Contains(t, out, field, "shipped-consumer section must name field %q", field)
	}

	// I-04 propagation: shape reference and the no-silent-omission language
	// (TC-001 clause 5).
	assert.Contains(t, out, "ChangeImpactSet", "must reference the ChangeImpactSet shape")
	assert.Contains(t, out, "never a completion record that omits an affected artifact without a stated disposition",
		"must carry the exact no-silent-omission language")

	// Design divergence: references (does not restate) defect-class-sweep.md's
	// Backward-looking rework section (TC-001 clause 6 / REQ-F-007).
	assert.Contains(t, out, "defect-class-sweep.md", "design-divergence section must reference defect-class-sweep.md")
	assert.Contains(t, out, "Backward-looking rework", "must reference defect-class-sweep.md's Backward-looking rework section by name")
}

// TestNewIncludeResolverWithEmbed_EmptyDataRoot verifies that the embed
// backstop works even with an empty data root (zero-config consumer mode).
func TestNewIncludeResolverWithEmbed_EmptyDataRoot(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")

	out, err := r.Resolve("{{include: prompts/_partials/_commands.md}}")
	require.NoError(t, err, "embed-only mode (empty dataRoot) must resolve from embed")
	assert.NotEmpty(t, strings.TrimSpace(out))
}

// TestOrchestratorRenderer_LoadsFromEmbed verifies the zero-config path:
// when no disk prompts exist, the renderer loads templates from the embedded FS.
func TestOrchestratorRenderer_LoadsFromEmbed(t *testing.T) {
	// Point at a directory that exists but contains no prompt files.
	emptyPromptsDir := t.TempDir()

	renderer, err := NewOrchestratorRenderer(emptyPromptsDir)
	require.NoError(t, err, "renderer must initialize even with no disk prompts")
	require.NotNil(t, renderer)

	// Embedded prompts include task/development.md and others.
	// Verify at least one template is loaded from the embed.
	out, err := renderer.Render("task/development.md", map[string]string{
		"task_title":      "Test Task",
		"task_key":        "E01-F01-001",
		"feature_key":     "E01-F01",
		"epic_key":        "E01",
		"task_id":         "E01-F01-001",
		"task_status":     "development",
		"complexity_tier": "SIMPLE",
	})
	require.NoError(t, err, "embedded task/development.md must render successfully")
	assert.NotEmpty(t, strings.TrimSpace(out), "rendered output must be non-empty")
}

// ============================================================================
// T-E34-F08-002 / TC-001 — tier-matrix.md single-source-of-truth
// ============================================================================

// tierMatrixCanonicalPath is the exact bundle path the five consuming
// prompts must reference. TC-001's edge case ("a prompt that references the
// file by a stale/renamed path fails ... not just the grep") is why this
// constant, not a loose substring like "tier-matrix", is what the grep test
// below matches against.
const tierMatrixCanonicalPath = "skills/quality/context/tier-matrix.md"

// TestTierMatrixRendersThroughProductionRenderer renders tier-matrix.md
// through the production include resolver (embed-aware, matching how the
// five consuming prompts would pull it in) and asserts the SIMPLE/STANDARD/
// COMPLEX tier contract and the "missing artifacts are failures only when
// the selected tier requires them" paragraph (REQ-F-001) are present.
func TestTierMatrixRendersThroughProductionRenderer(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")

	out, err := r.Resolve("{{include: " + tierMatrixCanonicalPath + "}}")
	require.NoError(t, err, "tier-matrix.md must render through the production renderer with no error")

	assert.Contains(t, out, "## Tier contract")
	assert.Contains(t, out, "## Executable gate evidence")
	for _, tier := range []string{"SIMPLE", "STANDARD", "COMPLEX"} {
		assert.Contains(t, out, tier, "tier contract table must name tier %q", tier)
	}
	assert.Contains(t, out, "Missing artifacts are failures only when the selected tier requires them.")
}

// TestTierMatrixIncludeResolvesFromStalePathFails is TC-001's explicit edge
// case: a stale or renamed path must fail include resolution, not just a
// grep for a loose substring.
func TestTierMatrixIncludeResolvesFromStalePathFails(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")
	_, err := r.Resolve("{{include: skills/quality/context/tier-matrix-renamed.md}}")
	require.Error(t, err, "a stale/renamed tier-matrix.md path must fail include resolution")
}

// tierMatrixConsumingPrompts is the exact five-prompt set T-E34-F08-002's
// task scope names (REQ-F-001, AC-1/AC-T2).
var tierMatrixConsumingPrompts = []string{
	"feature/assessment.md",
	"feature/task_generation.md",
	"feature/code_review.md",
	"feature/qa.md",
	"feature/approval.md",
}

// tierMatrixTableHeaderCells is tier-matrix.md's own table header. A
// restated (not referenced) copy of the tier table in a consuming prompt
// would reproduce this same header row — this is the structural signal
// TC-001 requires ("a table with the same three tier names and a 'gate'
// column, not an exact-string blacklist").
var tierMatrixTableHeaderCells = []string{
	"Planning source", "Test source", "Same-model gate", "Separate QA", "Final UAT",
}

// TestFivePromptsReferenceTierMatrixWithoutRestatingIt is TC-001's
// structural grep/regex check: each of the five consuming prompts contains a
// textual pointer to tier-matrix.md's exact canonical path, and none of them
// restate the tier table itself.
func TestFivePromptsReferenceTierMatrixWithoutRestatingIt(t *testing.T) {
	for _, prompt := range tierMatrixConsumingPrompts {
		t.Run(prompt, func(t *testing.T) {
			body, err := sharkdata.ReadEmbedded("prompts/" + prompt)
			require.NoError(t, err, "embedded prompt %s must exist", prompt)
			content := string(body)

			assert.Contains(t, content, tierMatrixCanonicalPath,
				"%s must reference tier-matrix.md by its exact canonical path", prompt)

			restated := true
			for _, cell := range tierMatrixTableHeaderCells {
				if !strings.Contains(content, cell) {
					restated = false
					break
				}
			}
			assert.False(t, restated,
				"%s must not restate the tier table's own header cells (%v)", prompt, tierMatrixTableHeaderCells)
		})
	}
}
