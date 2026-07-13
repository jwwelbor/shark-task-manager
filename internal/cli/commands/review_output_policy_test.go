package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/require"
)

func TestReviewPromptsUseCompactSuccessAndDetailedFindingsPolicy(t *testing.T) {
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
			name: "epic feature review compact pass contract",
			tmpl: "epic/feature_review.md",
			want: []string{
				"If zero findings: compact PASS artifact only",
				"`0 defects found`",
				"If any gap, overlap, ordering issue, closure issue, or other finding exists: full detailed report",
			},
		},
		{
			name: "feature task review compact pass contract",
			tmpl: "feature/task_review.md",
			want: []string{
				"If zero findings: compact PASS artifact only",
				"`0 defects found`",
				"If any requirement gap, ordering issue, task quality issue, integration mismatch, or other finding exists: full detailed report",
			},
		},
		{
			name: "feature code review compact pass contract",
			tmpl: "feature/code_review.md",
			want: []string{
				"If zero findings: compact PASS artifact only",
				"`0 defects found`",
				"If any blocker, failed command/test, missing AC or wiring proof, spec drift, or non-blocking observation exists: full detailed report",
				"Zero-finding PASS writes no `review-finding` notes.",
			},
		},
		{
			name: "feature qa compact pass contract",
			tmpl: "feature/qa.md",
			want: []string{
				"If zero findings: compact PASS artifact only",
				"`0 defects found`",
				"If any failed command/test, missing coverage, regression, pre-existing failure in scope, or non-blocking observation exists: full detailed report",
				"Zero-finding PASS writes no `review-finding` notes.",
			},
		},
		{
			name: "feature approval compact pass contract",
			tmpl: "feature/approval.md",
			want: []string{
				"If zero findings: compact APPROVED artifact only",
				"`0 defects found`",
				"If any finding, rejection, failed verification step, or non-blocking observation exists: full detailed report",
				"Zero-finding APPROVED writes no `review-finding` notes.",
			},
		},
		{
			name: "shared code review compact pass contract",
			tmpl: "_shared/code_review.md",
			want: []string{
				"If zero findings: compact PASS artifact only",
				"`0 defects found`",
				"If any blocker or non-blocking observation exists: full detailed report",
			},
		},
		{
			name: "shared qa compact pass contract",
			tmpl: "_shared/qa.md",
			want: []string{
				"If zero findings: compact PASS artifact only",
				"`0 defects found`",
				"If any failed command/test, missing AC proof, regression, pre-existing failure in scope, or non-blocking observation exists: full detailed report",
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

func TestReviewPromptsDoNotDuplicateProduceHeaders(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)
	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err, "shipped prompts must parse with includes resolved")

	vars := goldenVars()
	cases := []struct {
		name   string
		tmpl   string
		header string
	}{
		{
			name:   "epic feature review",
			tmpl:   "epic/feature_review.md",
			header: "PRODUCE feature review report at docs/review/E07/E07-F01/E07-F01-feature-review.md:",
		},
		{
			name:   "feature task review",
			tmpl:   "feature/task_review.md",
			header: "PRODUCE task review report at docs/review/E07/E07-F01/E07-F01-task-review.md:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered, err := renderer.Render(tc.tmpl, vars)
			require.NoError(t, err, "render %s", tc.tmpl)
			require.Equal(t, 1, strings.Count(rendered, tc.header), "duplicate produce header in %s", tc.tmpl)
		})
	}
}

func TestDeepReviewUsesCompactPassAndDetailedFindingsPolicy(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)

	cases := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "consolidator compact pass contract",
			path: filepath.Join(repoRoot, "skills", "shark-rider", "skills", "deep-review", "references", "consolidator.md"),
			want: []string{
				"If the verdict is **PASS** and the finding counts are exactly 0 blockers / 0 non-blockers / 0 nits, write a compact saved report only.",
				"`0 defects found`",
				"For **PASS-with-triage** or **FAIL**, produce the full detailed report below.",
			},
		},
		{
			name: "skill persistence guidance stays terse on pass",
			path: filepath.Join(repoRoot, "skills", "shark-rider", "skills", "deep-review", "SKILL.md"),
			want: []string{
				"compact on a clean PASS",
				"Tell the user only a short verdict summary plus `review_output_path`.",
				"Save it first; do not dump the full report inline on a clean PASS.",
			},
		},
		{
			name: "workflow prompt asks for compact-or-detailed report",
			path: filepath.Join(repoRoot, "skills", "shark-rider", "skills", "deep-review", "scripts", "review_workflow.js"),
			want: []string{
				"compact PASS report or a detailed PASS-with-triage/FAIL report",
				"produce the compact-or-detailed markdown report it specifies",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := os.ReadFile(tc.path)
			require.NoError(t, err, "%s should exist", tc.path)
			content := string(body)
			for _, want := range tc.want {
				require.Contains(t, content, want)
			}
		})
	}
}
