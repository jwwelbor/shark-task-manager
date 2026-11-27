package sharkdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestE31F01_TC018_CodingStandardsLimitLanguageScope keeps the Go-only scope in
// a sentence under the table, the form the craft rubric honors.
func TestE31F01_TC018_CodingStandardsLimitLanguageScope(t *testing.T) {
	doc := readRepoDoc(t, "docs", "architecture", "coding-standards.md")
	section, err := extractHeadingSection(doc, "Quality Limits")
	require.NoError(t, err)

	lines := strings.Split(string(section), "\n")
	lastRow := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "|") {
			lastRow = i
		}
	}
	require.Positive(t, lastRow, "the section must hold the limits table")
	below := normalizeWhitespace(strings.Join(lines[lastRow+1:], "\n"))
	assert.Contains(t, below, "Go source only")
	assert.Contains(t, below, "review_workflow.js")
}

// TestE31F01_TC018_LimitsConfigHeaderStatesWholeFileScope keeps the config's
// own description in step with what `make lint` measures.
func TestE31F01_TC018_LimitsConfigHeaderStatesWholeFileScope(t *testing.T) {
	header, _, found := strings.Cut(string(readRepoDoc(t, ".golangci-limits.yml")), "\nversion:")
	require.True(t, found, "the config must declare a version after its header")
	text := normalizeWhitespace(strings.ReplaceAll(header, "#", " "))

	for _, literal := range []string{"every function", "--whole-files", "new-from-rev", "Quality Limits"} {
		t.Run(literal, func(t *testing.T) {
			assert.Contains(t, text, literal)
		})
	}
	assert.NotContains(t, strings.ToLower(text), "new code only")
}

// TestE31F01_TC018_TestingStandardsRequireIsolation guards the B082 rule against
// shared command state, and keeps the existing assertion-style rule.
func TestE31F01_TC018_TestingStandardsRequireIsolation(t *testing.T) {
	doc := string(readRepoDoc(t, "docs", "architecture", "coding-standards", "testing.md"))

	assert.Contains(t, doc, "Do not share package-level command or flag state between tests.")
	assert.Contains(t, doc, "Use `testify/require` for setup and fatal assertions.")
}
