package sharkdata

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findRepoRootForTableParserTest walks up from the working directory to the
// nearest ancestor containing go.mod, mirroring the repo-root discovery
// pattern used by internal/cli/commands/interaction_prompts_test.go.
func findRepoRootForTableParserTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root (go.mod) walking up from %s", wd)
		}
		dir = parent
	}
}

// ---------------------------------------------------------------------------
// AC-T1 / TC-014 support: ParseInteractionMapTable
// ---------------------------------------------------------------------------

func TestParseInteractionMapTable_RealFile(t *testing.T) {
	repoRoot := findRepoRootForTableParserTest(t)
	path := filepath.Join(repoRoot, "docs", "plan", "E34-prompt-and-skill-improvements", "E34-interaction-map.md")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "real E34-interaction-map.md must exist")

	rows, err := ParseInteractionMapTable(data)
	require.NoError(t, err)
	require.Len(t, rows, 5, "expected one row per I-01..I-05")

	// Row order mirrors the table's own order (I-01..I-05).
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	assert.Equal(t, []string{"I-01", "I-02", "I-03", "I-04", "I-05"}, ids)

	// I-02 is the multi-consumer row: hand-checked against the real table.
	i02 := rows[1]
	assert.Equal(t, "I-02", i02.ID)
	assert.Equal(t, "E34-F05 Structured Gate Results and Parent-Owned Persistence", i02.Producer)
	assert.Equal(t, []string{"E34-F06", "E34-F07", "E34-F08"}, i02.Consumers)
	assert.Equal(t, "[I-02 GateResult v1](./architecture.md#i-02-gateresult-v1)", i02.ShapeSource)
	assert.Contains(t, i02.Payload, "Outer final-envelope outcome/evidence")
	assert.Equal(t, "JSON worker-to-parent contract", i02.Style)

	// I-05 is a single-consumer row.
	i05 := rows[4]
	assert.Equal(t, "I-05", i05.ID)
	assert.Equal(t, []string{"E34-F09"}, i05.Consumers)
}

func TestParseInteractionMapTable_BlankFieldStillReturnsRow(t *testing.T) {
	fixture := []byte(`# fixture

| ID | Producer feature | Consumer feature(s) | Shape source | Payload | Style |
|---|---|---|---|---|---|
| I-01 | E34-F03 Some Feature |  | [link](./architecture.md#i-01) | some payload | some style |
`)

	rows, err := ParseInteractionMapTable(fixture)
	require.NoError(t, err, "a blank field is a completeness concern (T-E34-F08-011), not a parse error")
	require.Len(t, rows, 1)
	assert.Equal(t, "I-01", rows[0].ID)
	assert.Nil(t, rows[0].Consumers, "blank Consumer feature(s) cell yields a nil, not a panic")
}

func TestParseInteractionMapTable_TableNotFound(t *testing.T) {
	_, err := ParseInteractionMapTable([]byte("# no table here\n\nJust prose.\n"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTableNotFound))
}

func TestParseInteractionMapTable_MalformedRowColumnCount(t *testing.T) {
	fixture := []byte(`| ID | Producer feature | Consumer feature(s) | Shape source | Payload | Style |
|---|---|---|---|---|---|
| I-01 | E34-F03 | E34-F02 | link | payload |
`)
	_, err := ParseInteractionMapTable(fixture)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTableRowMalformed))
}

func TestParseInteractionMapTable_ZeroDataRows(t *testing.T) {
	fixture := []byte(`| ID | Producer feature | Consumer feature(s) | Shape source | Payload | Style |
|---|---|---|---|---|---|
`)
	_, err := ParseInteractionMapTable(fixture)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTableEmpty))
}

func TestParseInteractionMapTable_NilInput(t *testing.T) {
	_, err := ParseInteractionMapTable(nil)
	require.Error(t, err, "nil input must return a typed error, not panic")
	assert.True(t, errors.Is(err, ErrTableNotFound))
}

// ---------------------------------------------------------------------------
// AC-T2 / TC-I-05-ADOPTION-MANIFEST support: ParseI05FieldList
// ---------------------------------------------------------------------------

func TestParseI05FieldList_RealFile(t *testing.T) {
	repoRoot := findRepoRootForTableParserTest(t)
	path := filepath.Join(repoRoot, "docs", "plan", "E34-prompt-and-skill-improvements", "architecture.md")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "real architecture.md must exist")

	fields, err := ParseI05FieldList(data)
	require.NoError(t, err)

	// Passing the whole file (not a pre-sliced section) is the point of this
	// test: I-01..I-04 each carry their own "| Field | Type | Contract |"
	// table, so this also proves ParseI05FieldList scopes to the I-05
	// heading rather than matching the first such table in the document.
	require.Len(t, fields, 8, "architecture.md's I-05 section defines exactly 8 fields")
	assert.Equal(t, []string{
		"schema_version",
		"source_commit",
		"bundle_digest",
		"changed_paths",
		"workflow_changes",
		"promoted_policies",
		"override_actions",
		"validation_evidence",
	}, fields)
}

func TestParseI05FieldList_SectionNotFound(t *testing.T) {
	_, err := ParseI05FieldList([]byte("## I-01 ReadinessEvidence v1\n\n| Field | Type | Contract |\n|---|---|---|\n| `x` | string | y |\n"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSectionNotFound))
}

func TestParseI05FieldList_TableNotFoundWithinSection(t *testing.T) {
	fixture := []byte(`## I-05 CanonicalAdoptionManifest v1

No table here, just prose.

## Next section

| Field | Type | Contract |
|---|---|---|
| ` + "`unrelated`" + ` | string | not the I-05 table |
`)
	_, err := ParseI05FieldList(fixture)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTableNotFound), "must not fall through to the next section's table")
}

func TestParseI05FieldList_ZeroDataRows(t *testing.T) {
	fixture := []byte(`## I-05 CanonicalAdoptionManifest v1

| Field | Type | Contract |
|---|---|---|
`)
	_, err := ParseI05FieldList(fixture)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTableEmpty))
}

func TestParseI05FieldList_NilInput(t *testing.T) {
	_, err := ParseI05FieldList(nil)
	require.Error(t, err, "nil input must return a typed error, not panic")
	assert.True(t, errors.Is(err, ErrSectionNotFound))
}

// ---------------------------------------------------------------------------
// AC-T4 / TC-002 support: ParseTierMatrixTable
// ---------------------------------------------------------------------------

// TestParseTierMatrixTable_RealTierContractTable reads feature.md's own
// "## Tier contract" table. tier-matrix.md doesn't exist yet — it is
// T-E34-F08-002's own deliverable — but REQ-F-001 requires it to carry this
// exact table verbatim, so this is genuinely real, already-committed table
// content of the shape ParseTierMatrixTable must handle, not an invented
// fixture. If T-002 ever changes this table's shape, T-002's own set-equality
// test (TC-002) is where that surfaces, not here.
func TestParseTierMatrixTable_RealTierContractTable(t *testing.T) {
	repoRoot := findRepoRootForTableParserTest(t)
	path := filepath.Join(repoRoot, "docs", "plan", "E34-prompt-and-skill-improvements",
		"E34-F08-tier-consistent-gates-and-final-integration-review", "feature.md")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "real feature.md must exist")

	rows, err := ParseTierMatrixTable(data)
	require.NoError(t, err)
	require.Len(t, rows, 3, "expected one TierRow per SIMPLE/STANDARD/COMPLEX")

	byTier := make(map[string]TierRow, len(rows))
	for _, r := range rows {
		byTier[r.Tier] = r
	}
	require.Contains(t, byTier, "SIMPLE")
	require.Contains(t, byTier, "STANDARD")
	require.Contains(t, byTier, "COMPLEX")

	// Full, exact derived sets for all three tiers — not just the COMPLEX/
	// SIMPLE qa-gate contrast — so a later change to the derivation rule (or
	// to the source table) that T-002 doesn't expect fails here first.
	assert.ElementsMatch(t, []string{"feature.md", "research-report.md"}, byTier["SIMPLE"].RequiredArtifacts)
	assert.Equal(t, []string{"code_review", "approval"}, byTier["SIMPLE"].RequiredGates)

	assert.ElementsMatch(t, []string{"spec.md", "test-plan.md"}, byTier["STANDARD"].RequiredArtifacts)
	assert.Equal(t, []string{"code_review", "approval"}, byTier["STANDARD"].RequiredGates)

	assert.ElementsMatch(t, []string{"spec.md", "test-plan.md"}, byTier["COMPLEX"].RequiredArtifacts)
	assert.Equal(t, []string{"code_review", "qa", "approval"}, byTier["COMPLEX"].RequiredGates)
}

func TestParseTierMatrixTable_BlankGateColumnIsTypedError(t *testing.T) {
	fixture := []byte(`| Tier | Planning source | Test source | Same-model gate | Separate QA | Final UAT |
|---|---|---|---|---|---|
| SIMPLE | ` + "`feature.md`" + ` | Inline ACs | Combined code review and QA |  | Yes |
`)
	_, err := ParseTierMatrixTable(fixture)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTierGateBlank), "a blank gate column must be a typed error, not an empty RequiredGates")
}

func TestParseTierMatrixTable_UnrecognizedGateValueIsTypedError(t *testing.T) {
	fixture := []byte(`| Tier | Planning source | Test source | Same-model gate | Separate QA | Final UAT |
|---|---|---|---|---|---|
| SIMPLE | ` + "`feature.md`" + ` | Inline ACs | Combined code review and QA | Maybe | Yes |
`)
	_, err := ParseTierMatrixTable(fixture)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTableRowMalformed))
}

func TestParseTierMatrixTable_TableNotFound(t *testing.T) {
	_, err := ParseTierMatrixTable([]byte("# no tier table here\n"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTableNotFound))
}

func TestParseTierMatrixTable_ZeroDataRows(t *testing.T) {
	fixture := []byte(`| Tier | Planning source | Test source | Same-model gate | Separate QA | Final UAT |
|---|---|---|---|---|---|
`)
	_, err := ParseTierMatrixTable(fixture)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTableEmpty))
}

func TestParseTierMatrixTable_NilInput(t *testing.T) {
	_, err := ParseTierMatrixTable(nil)
	require.Error(t, err, "nil input must return a typed error, not panic")
	assert.True(t, errors.Is(err, ErrTableNotFound))
}
