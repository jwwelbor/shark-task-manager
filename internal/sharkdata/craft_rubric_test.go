// External test package: the include resolver lives in internal/templates,
// which imports sharkdata.
package sharkdata_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	craftRubricPath  = "skills/quality/context/craft-rubric.md"
	craftRubricLimit = 170
	reviewCodePath   = "skills/quality/workflows/review-code.md"
	bundleDir        = "default_data"
	deepReviewDir    = "../../skills/shark-rider/skills/deep-review"

	standardsTemplatePath = "skills/quality/context/standards-template.md"
	generateStandardsPath = "skills/quality/workflows/generate-standards.md"
	bootstrapPath         = "skills/research/workflows/bootstrap.md"
	defectClassSweepPath  = "skills/quality/workflows/defect-class-sweep.md"
)

// thresholdPatterns is the closed list from the feature test plan (TC-002).
var thresholdPatterns = compileThresholdPatterns(map[string]string{
	"T01": `cyclomatic[^\n]{0,30}?\b10\b`,
	"T02": `complexity(?:_score)?\b["\'\]\[:, ]{0,8}(?:"error", ?)?(?:≤|<=|<|>|≥|>=)? ?\b(?:10|5)\b`,
	"T03": `-over[ =]10\b`,
	"T04": `(?:function[^\n]{0,25}?\b50\b|\b50 lines per function)`,
	"T05": `\bfiles?\b[^\n]{0,15}?\b500\b`,
	"T06": `\b10 ?/ ?500 ?/ ?50\b`,
	"T07": `nesting(?: depth)?[ :]{0,3}(?:≥|>=|>)? ?4\b`,
	"T08": `(?:coverage[^\n]{0,70}?(?:>=|≥|>) ?\d{2,3} ?%|(?:>=|≥|>) ?\d{2,3} ?% ?(?:code |test )?coverage)`,
	"T09": `(?:\d{2,3}%\+ ?(?:code |test )?coverage|coverage[^\n]{0,30}?\d{2,3}%\+)`,
	"T10": `(?:--cov-fail-under[= ]\d+|coverage\.threshold\.\w+[= ]\d+)`,
	"T11": `^\s*(?:- |\| )(?:Service [Ll]ayer|API [Ee]ndpoints|Repositories|Utilities|Components(?: \(Frontend\))?|Overall Project)\s*(?::|\|)\s*(?:>=|≥)? ?\d{2,3}%`,
})

// thresholdAllowlist maps an exact file to exact phrases that are exempt. Only
// document-length phrases may be added; none match the patterns today.
var thresholdAllowlist = map[string][]string{}

// retiredSeverityWording is the pre-edited-code-rule severity wording. Any of it
// left behind lets review pass an overage that the development exit gate blocks.
var retiredSeverityWording = regexp.MustCompile(`(?i)substantially\s+expand`)

// craftCategories are the review rule categories whose defect classes the sweep
// records in the project's local standards.
var craftCategories = []string{
	"ALTITUDE", "CODING-STANDARD", "COMPLEXITY", "DIP", "DRY", "ERROR-HANDLING",
	"IDIOM", "ISP", "LOGGING", "LSP", "NAMING", "OCP", "SIMPLIFICATION", "SRP",
	"TESTS", "TYPES", "YAGNI",
}

func compileThresholdPatterns(sources map[string]string) map[string]*regexp.Regexp {
	compiled := make(map[string]*regexp.Regexp, len(sources))
	for id, source := range sources {
		compiled[id] = regexp.MustCompile(`(?i)` + source)
	}
	return compiled
}

func matchingThresholds(line string) []string {
	var ids []string
	for id, pattern := range thresholdPatterns {
		if pattern.MatchString(line) {
			ids = append(ids, id)
		}
	}
	return ids
}

// thresholdHits reports every non-allowlisted threshold line in body.
func thresholdHits(file, body string) []string {
	var hits []string
	for _, line := range strings.Split(body, "\n") {
		for _, phrase := range thresholdAllowlist[file] {
			line = strings.ReplaceAll(line, phrase, "")
		}
		if ids := matchingThresholds(line); len(ids) > 0 {
			hits = append(hits, file+" "+strings.Join(ids, ",")+": "+strings.TrimSpace(line))
		}
	}
	return hits
}

func filesUnder(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	require.NoError(t, filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, filepath.ToSlash(p))
		}
		return err
	}))
	require.NotEmpty(t, files, "scan root %s yielded no files", root)
	return files
}

func embeddedText(t *testing.T, rel string) string {
	t.Helper()
	body, err := sharkdata.ReadEmbedded(rel)
	require.NoError(t, err, rel)
	return string(body)
}

func deepReviewText(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(deepReviewDir, rel))
	require.NoError(t, err, rel)
	return string(body)
}

// scanThresholds reports threshold lines in every file under scanRoot. Paths are
// made relative to base before read loads them.
func scanThresholds(t *testing.T, base, scanRoot string, read func(*testing.T, string) string) []string {
	t.Helper()
	var hits []string
	for _, p := range filesUnder(t, base+"/"+scanRoot) {
		rel := strings.TrimPrefix(p, base+"/")
		if rel != craftRubricPath {
			hits = append(hits, thresholdHits(rel, read(t, rel))...)
		}
	}
	return hits
}

func TestE31F01_TC002_ThresholdsLiveOnlyInCraftRubric(t *testing.T) {
	cases := []struct {
		base, root string
		read       func(*testing.T, string) string
	}{
		{bundleDir, "prompts", embeddedText},
		{bundleDir, "skills", embeddedText},
		{deepReviewDir, "references", deepReviewText},
	}
	for _, tc := range cases {
		t.Run(tc.root, func(t *testing.T) {
			hits := scanThresholds(t, tc.base, tc.root, tc.read)
			assert.Empty(t, hits, "hardcoded thresholds belong only in %s:\n%s", craftRubricPath, strings.Join(hits, "\n"))
		})
	}
}

func TestE31F01_TC002_CraftRubricStatesTheGenericDefaults(t *testing.T) {
	rubric := embeddedText(t, craftRubricPath)
	for _, row := range []string{
		"| Cyclomatic complexity | 10 |",
		"| Function length (lines) | 50 |",
		"| File length (lines) | 500 |",
		"| Nesting depth | 4 |",
	} {
		t.Run(row, func(t *testing.T) {
			assert.Contains(t, rubric, row)
		})
	}
}

// Counterfactual: every pattern must catch the wording it was written to ban,
// and must leave unrelated numbers alone.
func TestE31F01_TC002_ThresholdPatternCounterfactuals(t *testing.T) {
	banned := map[string]string{
		"T01": "defaults: cyclomatic 10, deep nesting",
		"T02": `npx eslint <files> --rule 'complexity: ["error", 10]'`,
		"T03": "gocyclo -over 10 <files>",
		"T04": "**Target:** < 50 lines per function",
		"T05": "file 500 lines",
		"T06": "(optional — defaults: 10 / 500 / 50)",
		"T07": "deep nesting ≥4):",
		"T08": "ensure the coverage threshold (typical: >= 80% for new code) is met.",
		"T09": "- [ ] Unit tests passing (80%+ coverage)",
		"T10": "uv run pytest --cov=app --cov-fail-under=80",
		"T11": "| Service Layer | 90% | Core business logic, must be proven |",
	}
	require.Len(t, banned, len(thresholdPatterns))
	for id, literal := range banned {
		t.Run(id, func(t *testing.T) {
			assert.Contains(t, matchingThresholds(literal), id, "%s must match %q", id, literal)
		})
	}
	for _, literal := range []string{
		"- Coverage: 94%",
		"Skip deep nesting beyond 3-4 levels",
		"50 lines MAX per task file",
		"≥80% of new hires",
	} {
		t.Run(literal, func(t *testing.T) {
			assert.Empty(t, matchingThresholds(literal), "%q is not a quality threshold", literal)
		})
	}
}

func lineCount(body string) int {
	return len(strings.Split(strings.TrimSuffix(body, "\n"), "\n"))
}

func TestE31F01_TC005_CraftRubricSizeBoundary(t *testing.T) {
	assert.Equal(t, craftRubricLimit, lineCount(strings.Repeat("x\n", craftRubricLimit)))
	assert.Greater(t, lineCount(strings.Repeat("x\n", craftRubricLimit+1)), craftRubricLimit)
	assert.LessOrEqual(t, lineCount(embeddedText(t, craftRubricPath)), craftRubricLimit)
}

func TestE31F01_TC005_CraftRubricContractLiterals(t *testing.T) {
	rubric := embeddedText(t, craftRubricPath)
	for _, literal := range []string{
		"generic default — not declared in local standards",
		"declared in local standards (§Quality Limits)",
		"proposed in local standards, unconfirmed",
		// Relative, so inlining the rubric adds the bundled-skill read note.
		"../context/review-rubric.md",
		"\n## Edited-code rule\n",
		"\n## Severity table\n",
		"\n## Limit exceptions\n",
		"\n## Self-review checklist\n",
		"\n### Reuse\n",
		"\n### Standards citation\n",
		"\n### SOLID and architecture\n",
		"\n### Idiom\n",
		"\n### Complexity and size\n",
		"\n### Test design\n",
	} {
		t.Run(strings.TrimSpace(literal), func(t *testing.T) {
			assert.Contains(t, rubric, literal)
		})
	}
}

// The rubric is inlined into dispatch prompts, so it must carry no directives
// and none of the commands those prompts are forbidden to name.
func TestE31F01_TC005_CraftRubricForbiddenText(t *testing.T) {
	rubric := embeddedText(t, craftRubricPath)
	for _, forbidden := range []string{
		"{{",
		"skills/quality/context/review-rubric.md",
		"shark status advance", "shark status set",
		"shark task next-status", "shark task set-status",
		"shark feature next-status", "shark epic next-status",
		"shark claim", "shark heartbeat", "shark release",
	} {
		t.Run(forbidden, func(t *testing.T) {
			assert.NotContains(t, rubric, forbidden)
		})
	}
	t.Run("retired severity wording", func(t *testing.T) {
		assert.NotRegexp(t, retiredSeverityWording, rubric)
	})
}

func TestE31F01_TC021_CraftRubricStatesExceptionAndConsultationRules(t *testing.T) {
	rubric := embeddedText(t, craftRubricPath)
	exceptions := section(t, rubric, "\n## Limit exceptions\n", "\n## ")
	assert.Contains(t, exceptions, "| Unit | Limit | Recorded value | Reason | Decision |")
	consultation := section(t, rubric, "\n## Architect consultation\n", "\n## ")
	assert.Contains(t, consultation, "consult the architect")
	assert.Contains(t, consultation, "`blocked`")
	assert.Contains(t, consultation, "`gate_result.no_kickback_reason`")
}

func TestE31F01_TC005_CraftRubricResolvesAsInclude(t *testing.T) {
	resolver := templates.NewIncludeResolverWithEmbed("")
	resolved, err := resolver.Resolve("{{include: " + craftRubricPath + "}}")
	require.NoError(t, err)
	assert.Contains(t, resolved, "generic default — not declared in local standards")
	assert.NotContains(t, resolved, "{{include:")
}

func TestE31F01_TC006_ReviewCodeDefersToCraftRubric(t *testing.T) {
	body := embeddedText(t, reviewCodePath)
	assert.Contains(t, body, "../context/craft-rubric.md")
	assert.Contains(t, body, "triage skill's decision tree")
	assert.Contains(t, body, "unattended review disposition")
	assert.Empty(t, thresholdHits(reviewCodePath, body))
	assert.NotRegexp(t, retiredSeverityWording, body)
}

func TestE31F01_TC013_DeepReviewWorkersLoadCraftRubric(t *testing.T) {
	skill := deepReviewText(t, "SKILL.md")
	_, runner, found := strings.Cut(skill, "## Runner 2")
	require.True(t, found)
	runner, _, found = strings.Cut(runner, "## Runner 3")
	require.True(t, found)
	// Both load forms: the skill read and the extracted file.
	assert.Contains(t, runner, "shark skill get quality context/craft-rubric.md")
	assert.Contains(t, runner, "extracted file")
	// The native Workflow runner builds its own worker load instruction.
	assert.Contains(t, deepReviewText(t, "scripts/review_workflow.js"),
		"shark skill get quality context/craft-rubric.md")
}

func TestE31F01_TC013_DeepReviewDefersToCraftRubric(t *testing.T) {
	for _, rel := range []string{"references/consolidator.md", "references/angle-d-cleanup.md"} {
		t.Run(rel, func(t *testing.T) {
			body := deepReviewText(t, rel)
			assert.Contains(t, body, "craft-rubric.md")
			assert.Contains(t, body, "Severity table")
		})
	}
	assert.Empty(t, thresholdHits("SKILL.md", deepReviewText(t, "SKILL.md")))
}

func TestE31F01_TC013_DeepReviewDropsRetiredSeverityWording(t *testing.T) {
	for _, rel := range []string{"SKILL.md", "references/consolidator.md", "references/angle-d-cleanup.md"} {
		t.Run(rel, func(t *testing.T) {
			assert.NotRegexp(t, retiredSeverityWording, deepReviewText(t, rel))
		})
	}
}

// The rubric precondition must be stated before any runner, so no runner can
// dispatch a reviewer ahead of it.
func TestE31F01_TC013_DeepReviewChecksRubricBeforeAnyRunner(t *testing.T) {
	skill := deepReviewText(t, "SKILL.md")
	precondition := section(t, skill, "\n## Craft rubric precondition\n", "\n## Runner 1")
	assert.Contains(t, precondition, "shark skill get quality context/craft-rubric.md")
	assert.Contains(t, precondition, "extracted")
	assert.Contains(t, precondition, "runner_mode=incomplete")
}

// section returns the text of body between the start and end markers.
func section(t *testing.T, body, start, end string) string {
	t.Helper()
	_, rest, found := strings.Cut(body, start)
	require.True(t, found, "missing %q", start)
	text, _, found := strings.Cut(rest, end)
	require.True(t, found, "missing %q after %q", end, start)
	return text
}

func TestE31F01_TC001_StandardsTemplateDeclaresQualityLimits(t *testing.T) {
	template := embeddedText(t, standardsTemplatePath)
	outputA := section(t, template, "## Required Output A Structure", "## Required Output B Structure")
	assert.Contains(t, outputA, "\n## Quality Limits\n")

	limits := section(t, outputA, "\n## Quality Limits\n", "\n## ")
	assert.Contains(t, limits, "| Limit | Value | Applies to | Enforcement | Measuring command |")
	for _, row := range []string{
		"| Cyclomatic complexity |",
		"| Function length (lines) |",
		"| File length (lines) |",
		"| Nesting depth |",
		"| Coverage (optional) |",
	} {
		t.Run(row, func(t *testing.T) {
			assert.Contains(t, limits, row)
		})
	}
	assert.Contains(t, limits, "tool-enforced")
	assert.Contains(t, limits, "review-enforced")
	assert.NotRegexp(t, `(?m)^\|.*\d`, limits, "example rows must state no numeric value")
}

func TestE31F01_TC001_StandardsTemplateDefinesLimitExceptions(t *testing.T) {
	template := embeddedText(t, standardsTemplatePath)
	outputA := section(t, template, "## Required Output A Structure", "## Required Output B Structure")
	limits := section(t, outputA, "\n## Quality Limits\n", "\n## ")
	assert.Contains(t, limits, "\n### Limit exceptions\n")
	assert.Contains(t, limits, "| Unit | Limit | Recorded value | Reason | Decision |")
}

func TestE31F01_TC001_StandardsTemplateRequiresTestingConventions(t *testing.T) {
	template := embeddedText(t, standardsTemplatePath)
	testingStandards := section(t, template, "\n## Testing Standards (Code-Level)\n", "\n## ")
	assert.Contains(t, testingStandards, "### Test Isolation")
	assert.Contains(t, testingStandards, "### Assertion Style")
}

func TestE31F01_TC003_GeneratorDerivesProposesAndValidatesLimits(t *testing.T) {
	workflow := embeddedText(t, generateStandardsPath)
	outputA := section(t, workflow, "### Output A: Coding Standards Document", "### Output B")
	assert.Contains(t, outputA, "- Quality Limits")

	derive := section(t, workflow, "### Step 1:", "### Step 2:")
	assert.Contains(t, derive, "Quality Limits")

	write := section(t, workflow, "### Step 2:", "### Step 3:")
	assert.Contains(t, write, "## Quality Limits")
	assert.Contains(t, write, "../context/craft-rubric.md")
	assert.Contains(t, write, "proposed — confirm")

	validate := section(t, workflow, "### Step 4:", "## Success Criteria")
	assert.Contains(t, validate, "## Quality Limits")
}

func TestE31F01_TC004_BootstrapMarkerRecordsQualityLimits(t *testing.T) {
	workflow := embeddedText(t, bootstrapPath)
	marker := section(t, workflow, "## Phase 4:", "## Phase 5:")
	assert.Contains(t, marker, "\n**Quality Limits**: declared, not declared, or n/a (readiness 2)\n")
}

func TestE31F01_TC004_BootstrapFillGapsAddsMissingLimitsSection(t *testing.T) {
	workflow := embeddedText(t, bootstrapPath)
	preflight := section(t, workflow, "### Step 0.2:", "### Step 0.3:")
	assert.Contains(t, preflight, "## Quality Limits")
	assert.Contains(t, preflight, "existing_standards_path")
	assert.Contains(t, preflight, "skills/quality/workflows/generate-standards.md")
}

func TestE31F01_TC004_BootstrapGreenfieldStandardsFollowTemplate(t *testing.T) {
	workflow := embeddedText(t, bootstrapPath)
	step := section(t, workflow, "### Step 3.3:", "## Phase 4:")
	// The brownfield branch of the same step already names the generator, so
	// only the text after the greenfield marker counts.
	_, greenfield, found := strings.Cut(step, "**Greenfield**:")
	require.True(t, found, "missing greenfield standards branch")
	for _, literal := range []string{
		standardsTemplatePath,
		"## Quality Limits",
		"Test Isolation",
		"Assertion Style",
		craftRubricPath,
		"proposed — confirm",
	} {
		t.Run(literal, func(t *testing.T) {
			assert.Contains(t, greenfield, literal)
		})
	}
}

func TestE31F01_TC012_SweepRecordsCraftClassesInLocalStandards(t *testing.T) {
	workflow := embeddedText(t, defectClassSweepPath)
	guardSelection := section(t, workflow, "## Guard selection", "## Structural guard closure")
	assert.Contains(t, guardSelection, "coding-standards.md")
	assert.Contains(t, guardSelection, "proposed — confirm")

	// Scoped to the craft list itself, so a category named only among the ones
	// that never edit standards does not count. Backticks match whole tokens.
	craftList := section(t, guardSelection, "craft categories:", ".")
	for _, category := range craftCategories {
		t.Run(category, func(t *testing.T) {
			assert.Contains(t, craftList, "`"+category+"`")
		})
	}
}
