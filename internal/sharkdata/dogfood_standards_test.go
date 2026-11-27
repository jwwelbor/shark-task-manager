package sharkdata

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	limitsConfigFlag = "--config=.golangci-limits.yml"
	wholeFilesFlag   = "--whole-files"
)

// readRepoDoc reads a file of this repository, addressed from the repo root.
func readRepoDoc(t *testing.T, parts ...string) []byte {
	t.Helper()
	root := findRepoRootForTableParserTest(t)
	data, err := os.ReadFile(filepath.Join(append([]string{root}, parts...)...))
	require.NoError(t, err)
	return data
}

// TestE31F01_TC018_TechStackDeclaresQualityGateTable keeps the gate commands in
// the section and table shape that development prompts resolve them from.
func TestE31F01_TC018_TechStackDeclaresQualityGateTable(t *testing.T) {
	doc := readRepoDoc(t, "docs", "architecture", "tech-stack.md")

	assert.NotContains(t, string(doc), "### Quality Gate (Mandatory)")
	assert.Contains(t, strings.Split(string(doc), "\n"), "## Quality Gate")

	section, err := extractHeadingSection(doc, "Quality Gate")
	require.NoError(t, err)
	rows, err := findMarkdownTable(section, []string{"Step", "Command", "When"})
	require.NoError(t, err)

	commands := make([]string, 0, len(rows))
	for _, row := range rows {
		commands = append(commands, row[1])
	}
	assert.Subset(t, commands, []string{"`make fmt`", "`make lint`", "`make test`"})
}

// TestE31F01_TC018_CodingStandardsDeclareQualityLimits pins the limit values and
// who enforces each one, which must agree with .golangci-limits.yml.
func TestE31F01_TC018_CodingStandardsDeclareQualityLimits(t *testing.T) {
	doc := readRepoDoc(t, "docs", "architecture", "coding-standards.md")

	assert.Contains(t, strings.Split(string(doc), "\n"), "## Quality Limits")
	section, err := extractHeadingSection(doc, "Quality Limits")
	require.NoError(t, err)
	rows, err := findMarkdownTable(section,
		[]string{"Limit", "Value", "Applies to", "Enforcement", "Measuring command"})
	require.NoError(t, err)

	byLimit := make(map[string][]string, len(rows))
	for _, row := range rows {
		byLimit[row[0]] = row[1:]
	}
	const scope = "production and test"
	const unmeasured = "none (review)"
	assert.Equal(t, map[string][]string{
		"Cyclomatic complexity":   {"10", scope, "tool-enforced (`gocyclo`)", "`make lint`"},
		"Function length (lines)": {"50", scope, "tool-enforced (`funlen`, line limit only)", "`make lint`"},
		"File length (lines)":     {"500", scope, "review-enforced", unmeasured},
		"Nesting depth":           {"4", scope, "review-enforced", unmeasured},
	}, byLimit)

	assert.Contains(t, string(section), "The measuring command is `make lint`.")
	assert.Contains(t, string(section), "Blank lines count; comment-only lines do not.")
}

// TestE31F01_TC018_CodingStandardsStateEditedFileScope keeps the documented
// scope in step with `make lint`: whole changed files are measured, and the
// limits no linter measures are checked by people on every edited file.
func TestE31F01_TC018_CodingStandardsStateEditedFileScope(t *testing.T) {
	doc := readRepoDoc(t, "docs", "architecture", "coding-standards.md")
	section, err := extractHeadingSection(doc, "Quality Limits")
	require.NoError(t, err)
	text := string(section)

	assert.Contains(t, text, "`make lint` measures every function in each Go file changed since the pinned revision")
	assert.Contains(t, text, "Self-review and code review check file length and nesting depth on every edited file.")
	assert.NotContains(t, text, "declaration line changed")
	assert.NotContains(t, text, "Growth inside an existing function is review-enforced")
}

// TestE31F01_TC018_TestingStandardsRequireIsolation guards the B082 rule against
// shared command state, and keeps the existing assertion-style rule.
func TestE31F01_TC018_TestingStandardsRequireIsolation(t *testing.T) {
	doc := string(readRepoDoc(t, "docs", "architecture", "coding-standards", "testing.md"))

	assert.Contains(t, doc, "Do not share package-level command or flag state between tests.")
	assert.Contains(t, doc, "Use `testify/require` for setup and fatal assertions.")
}

// TestE31F01_TC025_LimitsConfigKeepsDeclaredValues fails when the limits config
// is loosened away from the values the coding standards declare.
func TestE31F01_TC025_LimitsConfigKeepsDeclaredValues(t *testing.T) {
	var config struct {
		Linters struct {
			Settings struct {
				Funlen struct {
					Lines      int `yaml:"lines"`
					Statements int `yaml:"statements"`
				} `yaml:"funlen"`
				Gocyclo struct {
					MinComplexity int `yaml:"min-complexity"`
				} `yaml:"gocyclo"`
			} `yaml:"settings"`
		} `yaml:"linters"`
		Issues struct {
			NewFromRev string `yaml:"new-from-rev"`
		} `yaml:"issues"`
	}
	require.NoError(t, yaml.Unmarshal(readRepoDoc(t, ".golangci-limits.yml"), &config))

	settings := config.Linters.Settings
	assert.Equal(t, 50, settings.Funlen.Lines)
	assert.Equal(t, -1, settings.Funlen.Statements)
	assert.Equal(t, 10, settings.Gocyclo.MinComplexity)
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{40}$`), config.Issues.NewFromRev)
}

// makeRecipe returns the command lines of one Makefile target.
func makeRecipe(makefile, target string) []string {
	var recipe []string
	inTarget := false
	for _, line := range strings.Split(makefile, "\n") {
		switch {
		case strings.HasPrefix(line, target+":"):
			inTarget = true
		case inTarget && strings.HasPrefix(line, "\t"):
			recipe = append(recipe, line)
		default:
			inTarget = false
		}
	}
	return recipe
}

// linesWithField returns the lines that carry field as a whole word.
func linesWithField(lines []string, field string) []int {
	var found []int
	for i, line := range lines {
		if slices.Contains(strings.Fields(line), field) {
			found = append(found, i)
		}
	}
	return found
}

// TestE31F01_TC025_MakeLintMeasuresWholeEditedFiles requires `make lint` to run
// the main config and then the limits config over whole changed files.
func TestE31F01_TC025_MakeLintMeasuresWholeEditedFiles(t *testing.T) {
	recipe := makeRecipe(string(readRepoDoc(t, "Makefile")), "lint")
	require.NotEmpty(t, recipe)

	limits := linesWithField(recipe, limitsConfigFlag)
	require.Len(t, limits, 1, "make lint must run the limits config once")
	assert.Contains(t, strings.Fields(recipe[limits[0]]), wholeFilesFlag)

	// The main run keeps its own line, ahead of the limits run, so its failure
	// still stops the target.
	mainRun := slices.IndexFunc(recipe, func(line string) bool {
		return strings.HasSuffix(line, "&& golangci-lint run")
	})
	require.GreaterOrEqual(t, mainRun, 0, "make lint must run the main config")
	assert.Less(t, mainRun, limits[0])
}

// TestE31F01_TC025_CILintJobMeasuresWholeEditedFiles requires the same limits
// run as its own step of the CI lint job.
func TestE31F01_TC025_CILintJobMeasuresWholeEditedFiles(t *testing.T) {
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Uses string            `yaml:"uses"`
				With map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	data := readRepoDoc(t, ".github", "workflows", "ci.yml")
	require.NoError(t, yaml.Unmarshal(data, &workflow))

	var limitsArgs []string
	for _, step := range workflow.Jobs["lint"].Steps {
		isLintStep := strings.HasPrefix(step.Uses, "golangci/golangci-lint-action@")
		if isLintStep && slices.Contains(strings.Fields(step.With["args"]), limitsConfigFlag) {
			limitsArgs = append(limitsArgs, step.With["args"])
		}
	}
	require.Len(t, limitsArgs, 1, "the lint job must have one limits step")
	assert.Contains(t, strings.Fields(limitsArgs[0]), wholeFilesFlag)
}
