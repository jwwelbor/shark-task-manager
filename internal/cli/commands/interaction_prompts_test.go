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

// TestE34F03PromptBundleAndReferences keeps this prompt-only feature focused on
// mechanical bundle integrity. It verifies that the altered templates render
// through the shipped renderer and that the feature's documented handoff files
// exist; policy wording remains a human-review concern.
func TestE34F03PromptBundleAndReferences(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)
	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err, "shipped prompts must parse with includes resolved")

	for _, tmpl := range []string{
		"epic/design.md",
		"epic/decomposition.md",
		"epic/feature_review.md",
		"feature/specification.md",
		"feature/task_generation.md",
		"feature/task_review.md",
		"feature/code_review.md",
		"feature/qa.md",
		"feature/test_planning.md",
	} {
		t.Run("render "+tmpl, func(t *testing.T) {
			_, err := renderer.Render(tmpl, goldenVars())
			require.NoError(t, err)
		})
	}

	repoRoot := findRepoRootForInteractionTest(t)
	for _, path := range []string{
		filepath.Join(repoRoot, "docs", "plan", "E34-prompt-and-skill-improvements", "E34-interaction-map.md"),
		filepath.Join(repoRoot, "docs", "plan", "E34-prompt-and-skill-improvements", "E34-F03-deliverable-feature-decomposition-and-staged-integ", "feature.md"),
		filepath.Join(repoRoot, "docs", "plan", "E34-prompt-and-skill-improvements", "E34-F02-evidence-based-demo-script-skill", "feature.md"),
	} {
		t.Run("reference exists "+filepath.Base(path), func(t *testing.T) {
			info, err := os.Stat(path)
			require.NoError(t, err)
			require.False(t, info.IsDir())
		})
	}
}

func TestCrossEpicIntegrationLifecyclePrompts(t *testing.T) {
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
			name: "epic design creates and updates cross-epic maps",
			tmpl: "epic/design.md",
			want: []string{
				"docs/product/cross-epic-integration-map.md",
				"cross-epic-map.md",
				"X-## IDs are stable product-level IDs",
				"Ask the CX designer to review user journey handoffs",
				"X-## is only for cross-epic integrations",
			},
		},
		{
			name: "epic decomposition assigns X rows to features",
			tmpl: "epic/decomposition.md",
			want: []string{
				"Every relevant X-## must be assigned to producer and consumer",
				"Produces: X-##",
				"Consumes: X-##",
				"Keep X-## separate from I-##",
			},
		},
		{
			name: "epic feature review validates X closure",
			tmpl: "epic/feature_review.md",
			want: []string{
				"### Cross-epic integration closure",
				"Every relevant X-## row has producer epic and consumer epic(s) named",
				"Test coverage pointer exists or is explicitly deferred",
			},
		},
		{
			name: "feature specification mirrors X rows",
			tmpl: "feature/specification.md",
			want: []string{
				"## Cross-epic integrations",
				"Use X-## IDs verbatim",
				"Interfaces crossing epic",
				"not I-## or CONTRACT-###",
			},
		},
		{
			name: "feature test planning requires X coverage or deferral",
			tmpl: "feature/test_planning.md",
			want: []string{
				"### Cross-epic integration tests (X-##)",
				"Tag the TC with the X-## ID",
				"explicit deferral recorded in docs/product/progress.md",
			},
		},
		{
			name: "feature task generation keeps X tasks distinct",
			tmpl: "feature/task_generation.md",
			want: []string{
				"Integration Contracts > Cross-epic",
				"Do NOT invent I-## IDs for cross-epic integrations",
				"X-## work in the distinct",
			},
		},
		{
			name: "feature task review validates X task mirrors",
			tmpl: "feature/task_review.md",
			want: []string{
				"Every X-## the feature spec declares under \"Cross-epic integrations\"",
				"Integration Contracts > Cross-epic",
				"missing coverage disposition is FAIL",
			},
		},
		{
			name: "feature code review checks X implementation ownership",
			tmpl: "feature/code_review.md",
			want: []string{
				"one row per CONTRACT-### and I-##",
				"one row per X-## with contract/shape source",
				"coverage pointer or explicit deferral",
			},
		},
		{
			name: "feature qa enforces X wiring coverage",
			tmpl: "feature/qa.md",
			want: []string{
				"Cross-epic X-## integration",
				"include X-## rows with",
				"missing or broken X-## contract test",
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

func TestCrossEpicIntegrationMapTemplates(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	rootTemplate := filepath.Join(repoRoot, "file_templates", "cross-epic-integration-map.md")
	progressTemplate := filepath.Join(repoRoot, "file_templates", "progress.md")

	for _, tc := range []struct {
		name string
		path string
		want []string
	}{
		{
			name: "root cross-epic template",
			path: rootTemplate,
			want: []string{
				"| ID | Producer epic | Consumer epic(s) | Integration purpose | Contract / shape source | UX / CX handoff notes | Owning feature | Status | Test coverage pointer |",
				"X-##",
				"Use `X-##` only for cross-epic integrations",
				"Use `I-##` for cross-feature",
			},
		},
		{
			name: "progress template tracks cross-epic map",
			path: progressTemplate,
			want: []string{
				"## Cross-Epic Integration Map",
				"docs/product/cross-epic-integration-map.md",
				"Last updated:",
				"Updated by:",
				"Latest design decision:",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := os.ReadFile(tc.path)
			require.NoError(t, err, "%s should exist", tc.path)
			content := string(body)
			for _, want := range tc.want {
				require.Contains(t, content, want)
			}
		})
	}

	promptsDir := findRepoPromptsDir(t)
	dataRoot := filepath.Dir(promptsDir)
	skillTemplatePath := filepath.Join(dataRoot, "skills", "specification-writing", "context", "cross-epic-integration-map-template.md")
	body, err := os.ReadFile(skillTemplatePath)
	require.NoError(t, err, "cross-epic integration map template should be shipped with default shark-data")
	content := string(body)
	for _, want := range []string{
		"docs/product/cross-epic-integration-map.md",
		"`X-##` IDs are stable",
		"Use `X-##` only for cross-epic integrations",
		"Use `I-##` for cross-feature",
		"docs/product/progress.md",
	} {
		require.Contains(t, content, want)
	}
}

func findRepoRootForInteractionTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir := wd
	for {
		if isDirExist(filepath.Join(dir, "internal", "sharkdata", "default_data", "prompts")) {
			if _, err := os.Stat(filepath.Join(dir, "file_templates", "progress.md")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root walking up from %s", wd)
		}
		dir = parent
	}
}
