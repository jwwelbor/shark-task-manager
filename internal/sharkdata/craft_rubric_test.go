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
	craftRubricLimit = 150
	reviewCodePath   = "skills/quality/workflows/review-code.md"
	bundleDir        = "default_data"
	deepReviewDir    = "../../skills/shark-rider/skills/deep-review"
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

func TestE31F01_TC002_ThresholdsLiveOnlyInCraftRubric(t *testing.T) {
	var hits []string
	for _, root := range []string{"prompts", "skills"} {
		for _, p := range filesUnder(t, bundleDir+"/"+root) {
			rel := strings.TrimPrefix(p, bundleDir+"/")
			if rel != craftRubricPath {
				hits = append(hits, thresholdHits(rel, embeddedText(t, rel))...)
			}
		}
	}
	for _, p := range filesUnder(t, deepReviewDir+"/references") {
		rel := strings.TrimPrefix(p, deepReviewDir+"/")
		hits = append(hits, thresholdHits(rel, deepReviewText(t, rel))...)
	}
	assert.Empty(t, hits, "hardcoded thresholds belong only in %s:\n%s", craftRubricPath, strings.Join(hits, "\n"))
}

func TestE31F01_TC002_CraftRubricStatesTheGenericDefaults(t *testing.T) {
	rubric := embeddedText(t, craftRubricPath)
	for _, row := range []string{
		"| Cyclomatic complexity | 10 |",
		"| Function length (lines) | 50 |",
		"| File length (lines) | 500 |",
		"| Nesting depth | 4 |",
	} {
		assert.Contains(t, rubric, row)
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
		assert.Contains(t, matchingThresholds(literal), id, "%s must match %q", id, literal)
	}
	for _, literal := range []string{
		"- Coverage: 94%",
		"Skip deep nesting beyond 3-4 levels",
		"50 lines MAX per task file",
		"≥80% of new hires",
	} {
		assert.Empty(t, matchingThresholds(literal), "%q is not a quality threshold", literal)
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
		"skills/quality/context/review-rubric.md",
	} {
		assert.Contains(t, rubric, literal)
	}
	// The rubric is inlined into dispatch prompts, so it must carry no
	// directives and none of the commands those prompts are forbidden to name.
	for _, forbidden := range []string{
		"{{",
		"shark status advance", "shark status set",
		"shark task next-status", "shark task set-status",
		"shark feature next-status", "shark epic next-status",
		"shark claim", "shark heartbeat", "shark release",
	} {
		assert.NotContains(t, rubric, forbidden)
	}
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
		assert.Contains(t, deepReviewText(t, rel), "craft-rubric.md", rel)
	}
	assert.Empty(t, thresholdHits("SKILL.md", deepReviewText(t, "SKILL.md")))
}
