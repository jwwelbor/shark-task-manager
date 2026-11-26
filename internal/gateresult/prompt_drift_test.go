package gateresult

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Workers cannot read this package, so prompt and skill text is their only
// schema. Adding a field here without documenting it there breaks dispatches.
func TestBundleDocumentsGateResultFields(t *testing.T) {
	bundle := filepath.Join("..", "sharkdata", "default_data")
	cases := []struct {
		doc   string
		types []any
	}{
		{"prompts/_partials/_gate_result_directive.md", []any{GateResult{}, Finding{}, Kickback{}}},
		{"skills/quality/workflows/defect-class-sweep.md", []any{DefectClassSweep{}, SweepInstance{}, SweepGuard{}}},
		{"skills/quality/workflows/state-space-coverage.md", []any{ChangeImpactSet{}, AffectedArtifact{}, AffectedConsumer{}, SharedName{}}},
	}
	for _, c := range cases {
		body, err := os.ReadFile(filepath.Join(bundle, c.doc))
		require.NoError(t, err)
		visible := stripYAMLFrontmatter(string(body))
		for _, value := range c.types {
			assert.Empty(t, undocumentedFields(visible, value), "%s omits %T fields", c.doc, value)
		}
	}
	for _, doc := range []string{
		"skills/quality/workflows/defect-class-sweep.md",
		"skills/quality/SKILL.md",
	} {
		body, err := os.ReadFile(filepath.Join(bundle, doc))
		require.NoError(t, err)
		assert.NotContains(t, string(body), "disposition, evidence}", "%s uses stale SweepInstance evidence field", doc)
		assert.NotContains(t, string(body), "`disposition`, `evidence`", "%s uses stale SweepInstance evidence field", doc)
	}
}

func stripYAMLFrontmatter(s string) string {
	if !strings.HasPrefix(s, "---\n") {
		return s
	}
	if idx := strings.Index(s[4:], "\n---\n"); idx >= 0 {
		return s[4+idx+5:]
	}
	return s
}

func undocumentedFields(doc string, value any) []string {
	var missing []string
	typ := reflect.TypeOf(value)
	for i := 0; i < typ.NumField(); i++ {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		if !strings.Contains(doc, "`"+name+"`") && !strings.Contains(doc, "`"+name+"[]`") {
			missing = append(missing, name)
		}
	}
	return missing
}

func TestUndocumentedFields_ReportsMissingTag(t *testing.T) {
	assert.Equal(t, []string{"target_status", "reason"}, undocumentedFields("only `entity_key` and reason,", Kickback{}))
}
