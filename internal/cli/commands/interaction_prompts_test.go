package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/require"
)

func TestCrossFeatureInteractionLifecyclePrompts(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)
	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err, "shipped prompts must parse with includes resolved")

	vars := goldenVars()
	cases := []struct {
		name string
		tmpl string
		want []string
	}{
		{
			name: "epic design creates canonical interaction map",
			tmpl: "epic/design.md",
			want: []string{
				"interaction-map.md",
				"| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |",
				"I-## IDs are stable",
				"shape source resolves to a section in architecture.md",
			},
		},
		{
			name: "epic decomposition preserves every interaction wire",
			tmpl: "epic/decomposition.md",
			want: []string{
				"interaction-map.md",
				"every I-## must have a producer feature and at least one consumer feature",
				"no orphan wires",
			},
		},
		{
			name: "feature specification mirrors touched interactions",
			tmpl: "feature/specification.md",
			want: []string{
				"## Cross-feature interactions",
				"Produces",
				"Consumes",
				"Contract tests",
				"Use the I-## IDs verbatim",
			},
		},
		{
			name: "feature test planning designs shared contract tests",
			tmpl: "feature/test_planning.md",
			want: []string{
				"### Cross-feature contract tests (I-##)",
				"The SAME TC is referenced by both producer and consumer features",
				"Tag the TC with the I-## ID",
			},
		},
		{
			name: "feature task generation propagates cross-feature contracts",
			tmpl: "feature/task_generation.md",
			want: []string{
				"## Step 0 - Identify contracts before decomposing",
				"Integration Contracts > Cross-feature",
				"Do NOT invent new CONTRACT-### IDs for cross-feature wires",
			},
		},
		{
			name: "epic feature review validates interaction closure",
			tmpl: "epic/feature_review.md",
			want: []string{
				"### Interaction-map closure (multi-feature epics only)",
				"Producer and consumer cite the SAME shape source",
				"Print interaction-map closure table",
			},
		},
		{
			name: "feature task review validates task mirror",
			tmpl: "feature/task_review.md",
			want: []string{
				"### Integration coverage (STANDARD/COMPLEX only)",
				"Every I-## the feature spec declares under \"Cross-feature interactions\"",
				"contract-test pointer",
			},
		},
		{
			name: "feature qa enforces wiring coverage",
			tmpl: "feature/qa.md",
			want: []string{
				"Wiring coverage matrix",
				"CONTRACT-### and I-## rows",
				"missing or broken I-## contract test",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered, err := renderer.Render(tc.tmpl, vars)
			require.NoError(t, err, "render %s", tc.tmpl)
			for _, want := range tc.want {
				require.Contains(t, rendered, want)
			}
		})
	}
}

func TestInteractionMapTemplateIsShippedWithSpecificationWritingSkill(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)
	dataRoot := filepath.Dir(promptsDir)
	templatePath := filepath.Join(dataRoot, "skills", "specification-writing", "context", "interaction-map-template.md")

	body, err := os.ReadFile(templatePath)
	require.NoError(t, err, "interaction map template should be shipped with default shark-data")

	content := string(body)
	for _, want := range []string{
		"| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |",
		"I-##",
		"architecture.md",
		"shark related-docs add",
	} {
		require.Truef(t, strings.Contains(content, want), "template missing %q", want)
	}
}
