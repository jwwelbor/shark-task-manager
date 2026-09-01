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

// TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY is the shared structural
// guard for the I-01 producer/consumer contract. It prevents a source-anchor or
// contract-test migration from updating only one documentation surface.
func TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	e34Root := filepath.Join(repoRoot, "docs", "plan", "E34-prompt-and-skill-improvements")

	read := func(path string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(repoRoot, path))
		require.NoError(t, err, "%s must exist", path)
		return string(body)
	}

	architecture := read("docs/plan/E34-prompt-and-skill-improvements/architecture.md")
	wantFields := []string{
		"assessor_verdict",
		"owner_decision",
		"open_conditions",
		"gate_mode",
		"activation_owner",
		"closure_key",
		"counterpart_status",
		"review_basis",
		"demonstrability_disposition",
	}
	sectionPattern := regexp.MustCompile(`(?ms)^## I-01 ReadinessEvidence v1\s*$\n(.*?)(?:^## )`)
	fieldPattern := regexp.MustCompile("(?m)^\\| `([^`]+)` \\|")
	parseFields := func(document string) []string {
		sections := sectionPattern.FindAllStringSubmatch(document, -1)
		require.Len(t, sections, 1, "canonical I-01 section must occur exactly once")
		fieldRows := fieldPattern.FindAllStringSubmatch(sections[0][1], -1)
		fields := make([]string, 0, len(fieldRows))
		for _, row := range fieldRows {
			fields = append(fields, row[1])
		}
		return fields
	}
	gotFields := parseFields(architecture)
	require.Equal(t, wantFields, gotFields, "canonical I-01 field set and order must not drift")

	t.Run("counterfactual field mutations are detected", func(t *testing.T) {
		mutations := map[string]string{
			"removed": strings.Replace(architecture, "| `owner_decision` |", "| owner decision removed |", 1),
			"renamed": strings.Replace(architecture, "`gate_mode`", "`readiness_mode`", 1),
			"added":   strings.Replace(architecture, "| `open_conditions` |", "| `unexpected_field` | Added field |\n| `open_conditions` |", 1),
		}
		for name, mutation := range mutations {
			t.Run(name, func(t *testing.T) {
				assert.NotEqual(t, wantFields, parseFields(mutation))
			})
		}
	})

	interactionMap, err := os.ReadFile(filepath.Join(e34Root, "E34-interaction-map.md"))
	require.NoError(t, err)
	require.Contains(t, string(interactionMap), "### I-01 readiness evidence shape")
	require.Contains(t, string(interactionMap), "architecture.md#i-01-readinessevidence-v1")

	anchorConsumers := []string{
		"docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/feature.md",
		"docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/spec.md",
		"docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/test-plan.md",
		"docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/tasks/T-E34-F02-002.md",
		"docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/tasks/T-E34-F02-003.md",
		"docs/plan/E34-prompt-and-skill-improvements/E34-F03-deliverable-feature-decomposition-and-staged-integ/spec.md",
		"internal/sharkdata/default_data/skills/demo-script/SKILL.md",
		"skills/shark-rider/verbs/demo.md",
	}
	for _, path := range anchorConsumers {
		content := read(path)
		require.Contains(t, content, "E34-interaction-map.md#i-01-readiness-evidence-shape", path)
	}

	extractTableFields := func(document, marker string) []string {
		t.Helper()
		markerAt := strings.Index(document, marker)
		require.NotEqual(t, -1, markerAt, "missing table marker %q", marker)
		tableAt := strings.Index(document[markerAt:], "| Field |")
		require.NotEqual(t, -1, tableAt, "missing field table after %q", marker)
		table := document[markerAt+tableAt:]
		if end := strings.Index(table, "\n\n"); end >= 0 {
			table = table[:end]
		}
		rows := fieldPattern.FindAllStringSubmatch(table, -1)
		fields := make([]string, 0, len(rows))
		for _, row := range rows {
			fields = append(fields, row[1])
		}
		return fields
	}
	extractInlineFields := func(document, marker, terminator string) []string {
		t.Helper()
		normalized := strings.Join(strings.Fields(document), " ")
		markerAt := strings.Index(normalized, marker)
		require.NotEqual(t, -1, markerAt, "missing inline marker %q", marker)
		declaration := normalized[markerAt+len(marker):]
		end := strings.Index(declaration, terminator)
		require.NotEqual(t, -1, end, "missing inline terminator %q", terminator)
		matches := regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(declaration[:end], -1)
		fields := make([]string, 0, len(matches))
		for _, match := range matches {
			fields = append(fields, match[1])
		}
		return fields
	}

	mirrors := []struct {
		path       string
		table      bool
		marker     string
		terminator string
	}{
		{
			path:   "docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/feature.md",
			table:  true,
			marker: "E34-F02 consumes this exact nine-field readiness shape read-only:",
		},
		{
			path:       "docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/spec.md",
			marker:     "The procedure reads these nine values without transforming their meaning:",
			terminator: ". It reads current counterpart status",
		},
		{
			path:   "internal/sharkdata/default_data/skills/demo-script/SKILL.md",
			table:  true,
			marker: "Record these nine fields without changing their meaning:",
		},
		{
			path:       "skills/shark-rider/verbs/demo.md",
			marker:     "read these nine fields without changing their meanings:",
			terminator: ". Read counterpart status",
		},
	}
	for _, mirror := range mirrors {
		content := read(mirror.path)
		var fields []string
		if mirror.table {
			fields = extractTableFields(content, mirror.marker)
		} else {
			fields = extractInlineFields(content, mirror.marker, mirror.terminator)
		}
		require.Equal(t, wantFields, fields, "%s must mirror the exact I-01 field set and order", mirror.path)
	}

	t.Run("counterfactual consumer mutations are detected", func(t *testing.T) {
		fixture := "fields: `assessor_verdict`, `owner_decision`, `open_conditions`. End"
		want := []string{"assessor_verdict", "owner_decision", "open_conditions"}
		require.Equal(t, want, extractInlineFields(fixture, "fields:", ". End"))
		for name, mutation := range map[string]string{
			"removed": strings.Replace(fixture, ", `owner_decision`", "", 1),
			"renamed": strings.Replace(fixture, "`owner_decision`", "`approval_decision`", 1),
			"added":   strings.Replace(fixture, ". End", ", `consumer_only`. End", 1),
		} {
			t.Run(name, func(t *testing.T) {
				assert.NotEqual(t, want, extractInlineFields(mutation, "fields:", ". End"))
			})
		}
	})

	pointerConsumers := append([]string{
		"docs/plan/E34-prompt-and-skill-improvements/E34-interaction-map.md",
		"docs/plan/E34-prompt-and-skill-improvements/E34-F08-tier-consistent-gates-and-final-integration-review/feature.md",
	}, anchorConsumers...)
	for _, path := range pointerConsumers {
		content := read(path)
		require.Contains(t, content, "TC-I-01-READINESS-SYMMETRY", path)
		assert.NotContains(t, content, "shared contract-test pointer is **TC-002**", path)
	}

	legacyPointers := []string{
		"E34-F03-deliverable-feature-decomposition-and-staged-integ/test-plan.md#TC-002",
		"exact I-01 shape source and TC-002",
		"shared contract-test pointer is **TC-002**",
	}
	require.NoError(t, filepath.WalkDir(e34Root, func(path string, entry os.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		body, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, stale := range legacyPointers {
			assert.NotContains(t, string(body), stale, "%s contains a stale I-01 test pointer", path)
		}
		return nil
	}))
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

// TestE34F02DemoRiderProcedure guards the documented
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
			"/shark-rider demo <epic-key|feature-key|sprint-key> [--draft]",
			"`demo`",
			"`verbs/demo.md`",
		},
		"static help": {
			"demo <epic-key|feature-key|sprint-key> [--draft]",
			"`demo`",
		},
		"demo procedure": {
			"Only epic, feature, and sprint targets are valid",
			"Use the canonical entity key returned by `shark get`",
			"remains below `docs/demos/`",
			"Accept exactly one target and the optional `--draft` flag.",
			"Reject unknown flags,",
			"additional positional arguments, and a missing target.",
			"shark skill get demo-script",
			"shark related-docs list --epic=<epic-key> --json",
			"shark related-docs list --feature=<feature-key> --json",
			"shark sprint get <sprint-key> --json",
			"shark sprint backlog <sprint-key> --all --json",
			"A completed status selects work for review",
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
			"sprint reference-note command",
			"shark create note <key>",
			"--type=reference",
			"script is successfully created",
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

// TestE34F02DemoTargetSetIsClosed is a structural guard from the round-2
// code-review rework: the shipped demo target set (epic, feature, sprint)
// must stay identical everywhere it is documented. It extracts the exact
// target token list from every usage line rather than doing a substring
// Contains check, so a future undocumented 4th target fails this test
// instead of silently drifting past it, the way the sprint target once
// drifted past the feature/spec docs.
func TestE34F02DemoTargetSetIsClosed(t *testing.T) {
	repoRoot := findRepoRootForInteractionTest(t)
	paths := map[string]string{
		"demo procedure": filepath.Join(repoRoot, "skills", "shark-rider", "verbs", "demo.md"),
		"rider router":   filepath.Join(repoRoot, "skills", "shark-rider", "SKILL.md"),
		"static help":    filepath.Join(repoRoot, "skills", "shark-rider", "verbs", "help.md"),
	}

	wantTargets := []string{"epic-key", "feature-key", "sprint-key"}
	usageRe := regexp.MustCompile(`demo <([a-zA-Z0-9|-]+)> \[--draft\]`)

	for name, path := range paths {
		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile(path)
			require.NoError(t, err, "%s should be shipped", name)
			matches := usageRe.FindAllStringSubmatch(string(body), -1)
			require.NotEmpty(t, matches, "%s must document the demo usage line", name)
			for _, m := range matches {
				targets := strings.Split(m[1], "|")
				assert.Equal(t, wantTargets, targets,
					"%s documents a demo target set that no longer matches the closed epic/feature/sprint set", name)
			}
		})
	}

	// The per-verb static help entry and the demo-script skill's purpose
	// line also name the target set in prose; guard those independently so
	// a prose-only drift (no usage-line change) is caught too.
	helpBody, err := os.ReadFile(paths["static help"])
	require.NoError(t, err)
	require.Contains(t, string(helpBody), "Prepare an evidence-based demo for an epic, feature, or sprint.")

	skillBody, err := os.ReadFile(filepath.Join(repoRoot, "internal", "sharkdata", "default_data", "skills", "demo-script", "SKILL.md"))
	require.NoError(t, err, "demo-script skill should be shipped")
	normalizedSkill := regexp.MustCompile(`\s+`).ReplaceAllString(string(skillBody), " ")
	require.Contains(t, normalizedSkill, "for an epic, feature, or sprint.")
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
			"confirm, create, and verify approved epics in the same interaction",
			"leave feature decomposition to each epic's Shark workflow",
		},
		"static help": {
			"breakdown <docs-path>",
			"`breakdown`",
			"smallest coherent portfolio of charter-ready epics",
			"existing epics as an optional cross-check",
			"creates and verifies the epics in the same interaction",
			"stops before feature decomposition",
		},
		"procedure": {
			"/shark-rider breakdown <docs-path> [--output=<docs-path>]",
			"approved epics in the same interaction",
			"shark list --all --json",
			"Do not inspect sprint capacity or velocity",
			"Establish intrinsic scale, then compare when useful",
			"procedure must work when the project has no existing",
			"needs several demonstrable increments",
			"optional secondary drift check",
			"feature count, completed and remaining feature count, and known task count",
			"solely because precedents are absent",
			"Classify each candidate at the smallest plausible Shark level",
			"This classification is routing, not decomposition",
			"below epic",
			"ADR, oracle, benchmark, migration, test harness, research gate, or storage",
			"Do not assume that each source outcome, gate, phase, or heading is an epic",
			"A stable contract alone does",
			"Run the merge challenge",
			"A later delivery wave is not, by itself, an epic boundary",
			"Prefer fewer coherent epics when the evidence is ambiguous",
			"Check portfolio inflation and decomposition readiness",
			"Treat cross-epic contract overhead as a",
			"measurable success criteria",
			"high-level UAT scenarios with observable results",
			"Define the decomposition handoff",
			"stops at charter-ready epics",
			"Do not propose feature titles, feature counts, feature sizes",
			"use `/shark-rider run <epic-key>`",
			"Sprint planning follows task generation",
			"Shark `depends_on` represents a hard completion barrier",
			"Present the proposal and ask for approval",
			"Ask the user whether to create and apply that exact proposal",
			"confirmation before changing Shark state",
			"Do not require a second invocation or a special apply flag",
			"like Rider triage",
			"Create the approved epics",
			"If the refresh changes the approved delta",
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

	for _, forbidden := range []string{
		"--create",
		"proposal mode",
		"create mode",
		"shark sprint velocity --json",
		"### Likely feature slices",
		"Target feature sizes",
		"First sprint-ready target",
	} {
		require.NotContains(t, contents["procedure"], forbidden)
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
		"project-management workflow",
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
