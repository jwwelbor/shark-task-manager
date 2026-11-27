package commands

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	lintSourcePath         = regexp.MustCompile(`(?:^|[^\w./{}-])((?:internal|cmd)/[a-z][\w/.-]*)`)
	lintNonLocalBundlePath = regexp.MustCompile(`(?:^|[^\w./{}-])((?:shark-data|file_templates|prompts|agents)/[A-Za-z0-9_{}/.-]+|(?:epic|feature|task|bug|change|tech_debt|sprint|question)/[a-z_]+\.md\b)`)
	lintSkillSubdirPath    = regexp.MustCompile(`(?:^|[^\w./{}-])((?:[A-Za-z0-9_-]+/)?[a-z][a-z0-9-]*/(?:workflows|context|references|assets|scripts)/[A-Za-z0-9_{}/.-]+)`)
	lintTestCommand        = regexp.MustCompile(`\b(?:go (?:test|vet)|npm (?:run )?test|pytest|cargo test|yarn test|pnpm test)\b`)
	lintUnnamedFile        = "the file this step already directs"
	lintForbiddenPhrases   = []string{
		"repo tests use real DB",
		"ask your human partner",
		"human partner's permission",
		"via the Skill tool",
	}
	lintEvidenceKind = map[string][]string{
		"repair_plan": {"bug_repair_plan", "bug_review_baseline"},
		"development": {"bug_repair_plan", "bug_review_baseline"},
		"plan_review": {"bug_repair_plan", "bug_review_baseline", "plan_review_report"},
		"code_review": {"bug_repair_plan", "bug_review_baseline", "bug_review_scope", "bug_review_report", "bug_review_decisions", "bug_review_angles"},
		"qa":          {"bug_repair_plan", "bug_review_baseline", "bug_review_scope"},
	}
)

// nonLocalBundlePaths returns bundle-internal or prompt-template paths that are
// not rooted under skills/<name>/ (which the worker ownership preamble maps to
// `shark skill get <name> <path>`).
func nonLocalBundlePaths(source string) []string {
	var bad []string
	for _, m := range lintNonLocalBundlePath.FindAllStringSubmatch(source, -1) {
		bad = append(bad, m[1])
	}
	for _, m := range lintSkillSubdirPath.FindAllStringSubmatch(source, -1) {
		if !strings.HasPrefix(m[1], "skills/") {
			bad = append(bad, m[1])
		}
	}
	return bad
}

// missingEvidenceKinds returns the required kinds absent from the prompt's last
// outer evidence example, the one that supersedes the illustrative empty array.
func missingEvidenceKinds(prompt string, kinds []string) []string {
	example := ""
	if i := strings.LastIndex(prompt, `"evidence": [`); i >= 0 {
		example = prompt[i:]
	}
	var missing []string
	for _, kind := range kinds {
		if !strings.Contains(example, `"kind": "`+kind+`"`) {
			missing = append(missing, kind)
		}
	}
	return missing
}

// languageCommandLines returns lines naming a language-specific command.
func languageCommandLines(source string) []string {
	var bad []string
	for _, line := range strings.Split(source, "\n") {
		if lintTestCommand.MatchString(line) {
			bad = append(bad, line)
		}
	}
	return bad
}

// forbiddenPhraseMatches returns any forbidden project-specific or conflicting
// phrases present in source.
func forbiddenPhraseMatches(source string) []string {
	lower := strings.ToLower(source)
	var found []string
	for _, phrase := range lintForbiddenPhrases {
		if strings.Contains(lower, strings.ToLower(phrase)) {
			found = append(found, phrase)
		}
	}
	return found
}

func lintFiles(t *testing.T, root, suffix string, visit func(path, body string)) {
	t.Helper()
	seen := 0
	require.NoError(t, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, suffix) {
			return err
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		seen++
		visit(path, string(body))
		return nil
	}))
	require.NotZero(t, seen)
}

func TestRenderedPromptLint(t *testing.T) {
	lintFiles(t, filepath.Join("testdata", "rendered-prompts"), ".golden", func(path, body string) {
		assert.Empty(t, lintSourcePath.FindAllString(body, -1), "%s names a Shark source path", path)
		assert.Empty(t, nonLocalBundlePaths(body), "%s names a non-local bundle or prompt path", path)
		assert.NotContains(t, body, lintUnnamedFile, path)
		assert.Empty(t, languageCommandLines(body), "%s hard-codes a language-specific command", path)
		assert.Empty(t, forbiddenPhraseMatches(body), "%s contains forbidden project-specific or conflicting phrase", path)
		gate := strings.TrimSuffix(filepath.Base(path), ".golden")
		if kinds, ok := lintEvidenceKind[gate]; ok && filepath.Base(filepath.Dir(path)) == "bug" {
			assert.Empty(t, missingEvidenceKinds(body, kinds), "%s omits required evidence kinds", path)
		}
	})
	lintFiles(t, findRepoPromptsDir(t), ".md", func(path, body string) {
		assert.Empty(t, lintSourcePath.FindAllString(body, -1), "%s names a Shark source path", path)
		assert.Empty(t, nonLocalBundlePaths(body), "%s names a non-local bundle or prompt path", path)
		assert.Empty(t, languageCommandLines(body), "%s hard-codes a language-specific command", path)
		assert.Empty(t, forbiddenPhraseMatches(body), "%s contains forbidden project-specific or conflicting phrase", path)
	})
}

// Counterfactuals: each rule must be able to fail.
func TestRenderedPromptLint_RejectsBadInput(t *testing.T) {
	assert.NotEmpty(t, lintSourcePath.FindAllString("typed in `internal/gateresult`", -1))
	assert.NotEmpty(t, lintSourcePath.FindAllString("entrypoint in `cmd/shark/main.go`", -1))
	assert.NotEmpty(t, nonLocalBundlePaths("Load skills from `shark-data/skills/architecture/`"))
	assert.NotEmpty(t, nonLocalBundlePaths("Create it from file_templates/cross-epic-integration-map.md"))
	assert.NotEmpty(t, nonLocalBundlePaths("same signals as `epic/assessment.md`"))
	assert.NotEmpty(t, nonLocalBundlePaths("Invoke quality skill: quality/workflows/validate-design.md"))
	assert.NotEmpty(t, nonLocalBundlePaths("Invokesprint skill: sprint-planning/workflows/plan-sprint.md"))
	assert.Empty(t, nonLocalBundlePaths("Read `skills/quality/workflows/validate-design.md`"))
	assert.NotEmpty(t, missingEvidenceKinds(`"evidence": []`, lintEvidenceKind["development"]))
	assert.NotEmpty(t, missingEvidenceKinds(`no evidence block here`, lintEvidenceKind["development"]))
	assert.NotEmpty(t, languageCommandLines("Confirm test fails: `go test -v ./pkg`"))
	assert.NotEmpty(t, languageCommandLines("otherwise infer from the repo (`go.mod` -> `go vet` / `go test`; `pyproject.toml` -> `pytest`)"))
	assert.Contains(t, "report to "+lintUnnamedFile, lintUnnamedFile)
	assert.Equal(t, []string{"repo tests use real DB"}, forbiddenPhraseMatches("Follow project test patterns: repo tests use real DB, service tests use mocks"))
	assert.Equal(t, []string{"ask your human partner"}, forbiddenPhraseMatches("If unclear, ask your human partner before proceeding"))
	assert.Equal(t, []string{"human partner's permission"}, forbiddenPhraseMatches("No exceptions without your human partner's permission"))
	assert.Equal(t, []string{"via the Skill tool"}, forbiddenPhraseMatches("Invoke this skill FIRST before any production code via the Skill tool"))
}
