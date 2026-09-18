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
			},
		},
		{
			name: "feature qa compact pass contract",
			tmpl: "feature/qa.md",
			want: []string{
				"If zero findings: compact PASS artifact only",
				"`0 defects found`",
				"If any failed command/test, missing coverage, regression, pre-existing failure in scope, or non-blocking observation exists: full detailed report",
			},
		},
		{
			name: "feature approval compact pass contract",
			tmpl: "feature/approval.md",
			want: []string{
				"If zero findings: compact APPROVED artifact only",
				"`0 defects found`",
				"If any finding, rejection, failed verification step, or non-blocking observation exists: full detailed report",
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
				if !strings.Contains(rendered, want) {
					t.Errorf("%s must contain compact-pass/detailed-findings phrase %q", tc.tmpl, want)
				}
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
				if !strings.Contains(content, want) {
					t.Errorf("%s must contain compact-pass/detailed-findings phrase %q", tc.path, want)
				}
			}
		})
	}
}

// TestConsolidatorRoutesNonBlockersThroughTriage is a B066 regression gate.
// The deep-review consolidator and the deep-review skill itself must not map
// a non-blocking finding's severity directly to tech-debt ("host triages as
// tech-debt" / "host files as tech-debt"), and must not gate on findings
// being "fixed or explicitly triaged" without naming the triage skill's
// decision tree — the vaguer phrasing that let a finding skip the fix-now
// gate, duplicate search, and type classification (found in
// deep-review/SKILL.md's consolidation contract, 2026-09-18, after the
// original B066 fix scoped only consolidator.md). Every finding must route
// through the triage skill's decision tree (fix-now gate, duplicate search,
// type classification), of which tech-debt is only one possible outcome.
// This mirrors TestEmbedded_ReviewFindingsRouteThroughTriage in
// internal/sharkdata/embed_test.go, which covers the embedded-bundle
// producers (review-code.md, tech-lead.md); the files below live in the
// repo-tracked skills/shark-rider/ tree instead of the embedded bundle, so
// they need their own file-read assertion.
func TestConsolidatorRoutesNonBlockersThroughTriage(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	scopedFiles := []string{
		filepath.Join(repoRoot, "skills", "shark-rider", "skills", "deep-review", "references", "consolidator.md"),
		filepath.Join(repoRoot, "skills", "shark-rider", "skills", "deep-review", "SKILL.md"),
	}

	const triageReference = "triage skill's decision tree"
	directMapping := []string{
		"host triages as tech-debt",
		"host files as tech-debt",
		"host files these as tech-debt",
		"File as tech-debt only if",
	}

	for _, path := range scopedFiles {
		body, err := os.ReadFile(path)
		require.NoError(t, err, "%s should exist", path)
		content := string(body)

		if !strings.Contains(content, triageReference) {
			t.Errorf("%s must route non-blockers through the triage skill's decision tree; missing phrase %q", path, triageReference)
		}

		for _, phrase := range directMapping {
			if strings.Contains(content, phrase) {
				t.Errorf("%s must not map finding severity directly to tech-debt; found forbidden phrase %q", path, phrase)
			}
		}
	}
}

// TestConsolidatorNonBlockersCarryStableFingerprint is a review-triage-and-
// agent-attention-plan regression gate (plan.md Workstream 1: "Each item
// should include a stable fingerprint so reruns update or link the same
// record instead of multiplying it"). Mirrors
// TestEmbedded_NonBlockersCarryStableFingerprint in
// internal/sharkdata/embed_test.go for the repo-tracked consolidator.md.
func TestConsolidatorNonBlockersCarryStableFingerprint(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	path := filepath.Join(repoRoot, "skills", "shark-rider", "skills", "deep-review", "references", "consolidator.md")

	body, err := os.ReadFile(path)
	require.NoError(t, err, "%s should exist", path)
	content := string(body)

	if !strings.Contains(content, "fingerprint") {
		t.Errorf("%s: the Triage Summary's non-blockers-to-triage line must declare a `fingerprint` field for stable cross-run deduplication", path)
	}
}
