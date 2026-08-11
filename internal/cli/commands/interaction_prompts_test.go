package commands

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/assert"
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

// TestE34F04QuestionManagementPromptReferences reads shipped decision producers
// and rendered prompts. TC-002 and TC-003 are finite content contracts, so
// they must not emulate Question lifecycle policy.
func TestE34F04QuestionManagementPromptReferences(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	cases := []struct {
		name string
		path string
		cue  string
	}{
		{
			name: "architecture template",
			path: "internal/sharkdata/default_data/skills/architecture/context/templates/architecture-doc.md",
			cue:  "material unresolved design decision",
		},
		{
			name: "system design",
			path: "internal/sharkdata/default_data/skills/architecture/workflows/design-system.md",
			cue:  "cross-domain architecture decision",
		},
		{
			name: "backend design",
			path: "internal/sharkdata/default_data/skills/architecture/workflows/design-backend.md",
			cue:  "backend interface or service-boundary decision",
		},
		{
			name: "database design",
			path: "internal/sharkdata/default_data/skills/architecture/workflows/design-database.md",
			cue:  "schema, retention, or migration decision",
		},
		{
			name: "frontend architecture",
			path: "internal/sharkdata/default_data/skills/architecture/workflows/design-frontend.md",
			cue:  "frontend interaction or UX architecture decision",
		},
		{
			name: "security design",
			path: "internal/sharkdata/default_data/skills/architecture/workflows/design-security.md",
			cue:  "security control or risk-acceptance decision",
		},
		{
			name: "aesthetic direction",
			path: "internal/sharkdata/default_data/skills/frontend-design/workflows/commit-to-aesthetic-direction.md",
			cue:  "missing or contested aesthetic direction",
		},
		{
			name: "product vision",
			path: "internal/sharkdata/default_data/skills/product-design/workflows/d01-vision.md",
			cue:  "vision, scope, or constraint decision",
		},
		{
			name: "product feasibility",
			path: "internal/sharkdata/default_data/skills/product-design/workflows/d04-feasibility.md",
			cue:  "feasibility conclusion or proposed route",
		},
		{
			name: "user insights",
			path: "internal/sharkdata/default_data/skills/product-design/workflows/d06-user-insights.md",
			cue:  "research gap that changes a product decision",
		},
		{
			name: "user needs",
			path: "internal/sharkdata/default_data/skills/product-design/workflows/d07-user-needs.md",
			cue:  "prioritized user-need decision",
		},
		{
			name: "user personas",
			path: "internal/sharkdata/default_data/skills/product-design/workflows/d08-user-personas.md",
			cue:  "persona-priority or trade-off decision",
		},
		{
			name: "journey maps",
			path: "internal/sharkdata/default_data/skills/product-design/workflows/d09-journey-maps.md",
			cue:  "journey scope or critical-stage decision",
		},
		{
			name: "test results",
			path: "internal/sharkdata/default_data/skills/product-design/workflows/d12-test-results.md",
			cue:  "test-protocol or evidence-routing decision",
		},
		{
			name: "validated designs",
			path: "internal/sharkdata/default_data/skills/product-design/workflows/d14-validated-designs.md",
			cue:  "validated-design verdict or follow-up decision",
		},
		{
			name: "write epic",
			path: "internal/sharkdata/default_data/skills/specification-writing/workflows/write-epic.md",
			cue:  "epic scope or requirement decision",
		},
		{
			name: "write feature PRD",
			path: "internal/sharkdata/default_data/skills/specification-writing/workflows/write-feature-prd.md",
			cue:  "feature scope or requirement decision",
		},
		{
			name: "refine task requirements",
			path: "internal/sharkdata/default_data/skills/specification-writing/workflows/refine-task-requirements.md",
			cue:  "task requirement or architecture decision",
		},
		{
			name: "decompose epic",
			path: "internal/sharkdata/default_data/skills/specification-writing/workflows/decompose-epic.md",
			cue:  "feature-boundary or dependency-order decision",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(repoRoot, tc.path))
			require.NoError(t, err)
			content := string(body)
			require.Contains(t, content, "skills/question-management/SKILL.md")
			require.Contains(t, content, tc.cue)
			require.Contains(t, content, "non-material rationale")
			assert.NotContains(t, content, "shark question resolve")
			assert.NotContains(t, content, "--type=question_blocks")
		})
	}

	promptsDir := findRepoPromptsDir(t)
	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err)
	for _, tmpl := range []string{
		"epic/refinement.md",
		"epic/design.md",
		"feature/specification.md",
	} {
		t.Run("render "+tmpl, func(t *testing.T) {
			rendered, err := renderer.Render(tmpl, goldenVars())
			require.NoError(t, err)
			require.Contains(t, rendered, "skills/question-management/SKILL.md")
			require.Contains(t, rendered, "linked Q###")
		})
	}
}

// TestE34F02DemoRiderProcedure_TC001_TC005_TC007_TC008 guards the documented
// content contract for the explicit, host-local demo action. It intentionally
// checks shipped procedure text rather than inventing a runtime policy engine.
func TestE34F02DemoRiderProcedure_TC001_TC005_TC007_TC008(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	paths := map[string]string{
		"rider router":   filepath.Join(repoRoot, "skills", "shark-rider", "SKILL.md"),
		"static help":    filepath.Join(repoRoot, "skills", "shark-rider", "verbs", "help.md"),
		"demo procedure": filepath.Join(repoRoot, "skills", "shark-rider", "verbs", "demo.md"),
	}

	contents := make(map[string]string, len(paths))
	for name, path := range paths {
		body, err := os.ReadFile(path)
		require.NoError(t, err, "%s should be shipped", name)
		contents[name] = string(body)
	}

	for name, want := range map[string][]string{
		"rider router": {
			"/shark-rider demo <epic-key|feature-key> [--draft]",
			"`demo`",
			"`verbs/demo.md`",
		},
		"static help": {
			"demo <epic-key|feature-key> [--draft]",
			"`demo`",
		},
		"demo procedure": {
			"Only epic and feature targets are valid",
			"Use the canonical entity key returned by `shark get`",
			"remains below `docs/demos/`",
			"Accept exactly one target and the optional `--draft` flag.",
			"Reject unknown flags,",
			"additional positional arguments, and a missing target.",
			"shark skill get demo-script",
			"shark related-docs list --epic=<epic-key> --json",
			"shark related-docs list --feature=<feature-key> --json",
			"Demonstrated now",
			"Not demonstrated / pending integration",
			"Accepted risks and overrides",
			"documented existing environment/date-scoped evidence",
			"--draft",
			"Do not invent commands, credentials, deployments, endpoints, or proof",
			"docs/demos/<entity-key>/demo-script.md",
			"docs/demos/<entity-key>/evidence/",
			"shark related-docs add",
			"--epic=<epic-key>",
			"--feature=<feature-key>",
			"shark create note <key>",
			"--type=reference",
			"only after the script is successfully created",
			"normal deduplication and user confirmation",
			"does not call claim, status-transition, approval, provisioning, or automatic triage commands",
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, expected := range want {
				require.Contains(t, contents[name], expected)
			}
		})
	}

	// The procedure may mention prohibited operations in prose, so inspect only
	// executable examples. This guards the Mode-3 boundary without turning the
	// content test into a runtime policy engine.
	demoProcedure := contents["demo procedure"]
	commandBlocks := regexp.MustCompile("(?s)```bash\\n(.*?)```").FindAllStringSubmatch(demoProcedure, -1)
	require.NotEmpty(t, commandBlocks)
	commands := make([]string, 0)
	for _, block := range commandBlocks {
		for _, line := range strings.Split(block[1], "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "shark ") {
				commands = append(commands, line)
			}
		}
	}

	for _, command := range commands {
		for _, forbidden := range []string{
			"shark demo", "shark claim", "shark status", "shark approve",
			"shark provision", "shark create task", "shark create bug",
		} {
			require.NotContains(t, command, forbidden, "demo procedure command %q must preserve its Mode-3 boundary", command)
		}
	}
}

func TestSolutionWalkthroughRiderProcedure(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	paths := map[string]string{
		"rider router": filepath.Join(repoRoot, "skills", "shark-rider", "SKILL.md"),
		"static help":  filepath.Join(repoRoot, "skills", "shark-rider", "verbs", "help.md"),
		"procedure":    filepath.Join(repoRoot, "skills", "shark-rider", "verbs", "walkthrough.md"),
	}

	contents := make(map[string]string, len(paths))
	for name, path := range paths {
		body, err := os.ReadFile(path)
		require.NoError(t, err, "%s should be shipped", name)
		contents[name] = string(body)
	}

	for name, want := range map[string][]string{
		"rider router": {"/shark-rider walkthrough <target> [scope]", "entity key or `docs/` path", "`walkthrough`", "`verbs/walkthrough.md`"},
		"static help":  {"walkthrough <entity-key|docs-path> [scope]", "`walkthrough`"},
		"procedure": {
			"shark get <key> --json",
			"shark skill get solution-walkthrough",
			"shark related-docs list --epic=<epic-key> --json",
			"shark question list --status=open --limit=100 --offset=0 --json",
			"shark question list --status=answering --limit=100 --offset=0 --json",
			"shark question list --status=ready_for_resolution --limit=100 --offset=0 --json",
			"increasing `--offset` until a page is short",
			"shark question blocking-for <entity-key> --limit=100 --offset=0 --json",
			"shark related-docs list --question=<key> --json",
			"shark next <question-key>",
			"current_responder",
			"shark claim <question-key> --by=<current-responder> --json",
			"session_id",
			"shark question respond <question-key> --session=<session-id> --responder=<current-responder> --summary=\"<approved answer>\" --evidence-pointer=<durable-record-path>",
			"shark release <question-key> --session=<session-id>",
			"hand the Question to the",
			"infer or impersonate a responder. A response is",
			"document explicitly names",
			"Reviewed and confirmed",
			"docs/product/progress.md",
			"docs/architecture/adr/",
			"shark related-docs add \"Decision Record\" <path> --feature=<feature-key>",
			"shark create note <key> \"Decision record: <path>\" --type=reference",
			"Do not call status-transition, approval, or automatic triage commands.",
			"Do not create a decision record before the operator has resolved that decision.",
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, expected := range want {
				require.Contains(t, contents[name], expected)
			}
		})
	}
	for _, forbidden := range []string{
		"shark question resolve", "shark question withdraw", "shark question supersede",
	} {
		assert.NotContains(t, contents["procedure"], forbidden)
	}
}

func TestPortfolioBreakdownRiderProcedure(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	paths := map[string]string{
		"rider router": filepath.Join(repoRoot, "skills", "shark-rider", "SKILL.md"),
		"static help":  filepath.Join(repoRoot, "skills", "shark-rider", "verbs", "help.md"),
		"procedure":    filepath.Join(repoRoot, "skills", "shark-rider", "verbs", "breakdown.md"),
	}

	contents := make(map[string]string, len(paths))
	for name, path := range paths {
		body, err := os.ReadFile(path)
		require.NoError(t, err, "%s should be shipped", name)
		contents[name] = string(body)
	}

	for name, want := range map[string][]string{
		"rider router": {
			"/shark-rider breakdown <docs-path> [--output=<docs-path>]",
			"`breakdown`",
			"`verbs/breakdown.md`",
		},
		"static help": {
			"breakdown <docs-path>",
			"`breakdown`",
			"proposal mode does not create entities",
		},
		"procedure": {
			"/shark-rider breakdown <docs-path> [--output=<docs-path>]",
			"/shark-rider breakdown <approved-breakdown-path> --create",
			"The default mode writes a proposal only",
			"beneath the project `docs/`",
			"shark list --all --json",
			"shark sprint velocity --json",
			"Scrum does not supply a standard epic size",
			"tasks, bugs, change-cards, and tech-debt",
			"Epics and features are",
			"not sprint assignments",
			"Target feature sizes `1`, `2`, `3`, or `5`",
			"estimated at `5` or larger before sprint planning",
			"Delivery waves describe why-now sequence and safe parallelism",
			"Shark `depends_on` represents a hard completion barrier",
			"`docs/product/cross-epic-integration-map.md` records product-level X-##",
			"use candidate IDs such as `C-01`",
			"Current-state reconciliation",
			"Sprint-fit assessment",
			"Cross-epic map delta",
			"proposal mode made no Shark entity",
			"require explicit owner confirmation before the first write",
			"shark create epic \"<title>\" --description=\"<one-sentence outcome>\" --json",
			"shark admin validate",
			"Do not run the new epics",
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, expected := range want {
				require.Contains(t, contents[name], expected)
			}
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
