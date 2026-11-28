package sharkdata

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// limitsTarget is one of the three files that make the limits gate work.
type limitsTarget struct {
	path  []string
	check func([]byte) violations
}

var (
	limitsConfigTarget = limitsTarget{path: []string{".golangci-limits.yml"}, check: limitsConfigViolations}
	makefileTarget     = limitsTarget{path: []string{"Makefile"}, check: makefileLimitsViolations}
	ciTarget           = limitsTarget{path: []string{".github", "workflows", "ci.yml"}, check: ciLimitsViolations}
)

// limitsMutation weakens the limits gate by replacing old with replacement in
// the real file; want is part of the violation the guard must report for it.
type limitsMutation struct {
	name        string
	target      limitsTarget
	old         string
	replacement string
	want        string
}

const (
	makeLimitsCommand = "@export PATH=$$PATH:$$HOME/go/bin && golangci-lint run " +
		limitsConfigFlag + " " + wholeFilesFlag
	makeLimitsLine = "\t" + makeLimitsCommand
	ciLimitsName   = "      - name: golangci-lint (quality limits)\n"
	ciLimitsArgs   = "args: " + limitsConfigFlag + " " + wholeFilesFlag
	ciLimitsStep   = ciLimitsName + "        uses: golangci/golangci-lint-action@v7\n" +
		"        with:\n          version: latest\n          " + ciLimitsArgs + "\n" +
		"          skip-cache: true # the step above already restored the cache\n"
	wantLinterSet = "linters must be exactly funlen and gocyclo"
	wantExclusion = "must not exclude"
	wantSwallowed = "must not swallow"
	wantIgnored   = "must not ignore errors"
)

// configInsert adds YAML under a top-level or nested key of the limits config.
func configInsert(name, after, yamlText, want string) limitsMutation {
	return limitsMutation{
		name: name, target: limitsConfigTarget,
		old: after, replacement: after + yamlText, want: want,
	}
}

func configReplace(name, old, replacement, want string) limitsMutation {
	return limitsMutation{name: name, target: limitsConfigTarget, old: old, replacement: replacement, want: want}
}

func makeReplace(name, replacement, want string) limitsMutation {
	return limitsMutation{name: name, target: makefileTarget, old: makeLimitsLine, replacement: replacement, want: want}
}

var limitsValueMutations = []limitsMutation{
	configReplace("funlen lines raised", "lines: 50", "lines: 500", "funlen lines"),
	configReplace("gocyclo raised", "min-complexity: 10", "min-complexity: 100", "gocyclo min-complexity"),
	configReplace("funlen statements enabled", "statements: -1", "statements: 0", "funlen statements"),
	configReplace("short new-from-rev", "new-from-rev: ", "new-from-rev: d9427496 # ", "full SHA"),
}

var limitsFlagMutations = []limitsMutation{
	makeReplace("Makefile without whole files",
		strings.TrimSuffix(makeLimitsLine, " "+wholeFilesFlag), "measure whole files"),
	{name: "Makefile limits line deleted", target: makefileTarget,
		old: "\n" + makeLimitsLine, want: "run the limits config once"},
	{name: "CI without whole files", target: ciTarget, old: ciLimitsArgs,
		replacement: "args: " + limitsConfigFlag, want: "measure whole files"},
	{name: "CI limits step deleted", target: ciTarget, old: ciLimitsStep, want: "one limits step"},
}

var limitsLinterSetMutations = []limitsMutation{
	configReplace("funlen not enabled", "    - funlen\n", "", wantLinterSet),
	configReplace("gocyclo not enabled", "    - gocyclo\n", "", wantLinterSet),
	configInsert("third linter", "    - gocyclo\n", "    - errcheck\n", wantLinterSet),
	configReplace("standard default", "default: none", "default: standard", "linters default"),
	configInsert("funlen disabled", "  default: none\n", "  disable:\n    - funlen\n", "must not disable"),
	configInsert("gocyclo disabled", "  default: none\n", "  disable:\n    - gocyclo\n", "must not disable"),
	configInsert("tests not run", "  timeout: 5m\n", "  tests: false\n", "run on test files"),
}

var limitsExclusionMutations = []limitsMutation{
	configInsert("rule excluding funlen", "  default: none\n",
		"  exclusions:\n    rules:\n      - path: internal/\n        linters:\n          - funlen\n", wantExclusion),
	configInsert("rule excluding gocyclo", "  default: none\n",
		"  exclusions:\n    rules:\n      - path: internal/\n        linters:\n          - gocyclo\n", wantExclusion),
	configInsert("rule excluding test files", "  default: none\n",
		"  exclusions:\n    rules:\n      - path: _test\\.go\n        linters:\n          - errcheck\n", wantExclusion),
	configInsert("test file path excluded", "  default: none\n",
		"  exclusions:\n    paths:\n      - _test\\.go\n", wantExclusion),
	configInsert("issues rule excluding funlen", "issues:\n",
		"  exclude-rules:\n    - linters:\n        - funlen\n", wantExclusion),
}

var limitsMakefileMutations = []limitsMutation{
	makeReplace("limits line commented", "\t#"+makeLimitsCommand, "must not be commented out"),
	makeReplace("dash prefix", "\t-"+makeLimitsCommand, wantIgnored),
	makeReplace("at dash prefix", "\t@-"+strings.TrimPrefix(makeLimitsCommand, "@"), wantIgnored),
	makeReplace("or true appended", makeLimitsLine+" || true", wantSwallowed),
	makeReplace("semicolon true appended", makeLimitsLine+"; true", wantSwallowed),
}

var limitsCIMutations = []limitsMutation{
	{name: "CI continue on error", target: ciTarget, old: ciLimitsName,
		replacement: ciLimitsName + "        continue-on-error: true\n", want: "continue-on-error"},
	{name: "CI step condition", target: ciTarget, old: ciLimitsName,
		replacement: ciLimitsName + "        if: false\n", want: "must have no if"},
}

// TestE31F01_TC025_LimitsGuardRejectsWeakening derives each way of weakening
// the limits gate from the real files and requires the guard to report it.
func TestE31F01_TC025_LimitsGuardRejectsWeakening(t *testing.T) {
	groups := [][]limitsMutation{
		limitsValueMutations, limitsFlagMutations, limitsLinterSetMutations,
		limitsExclusionMutations, limitsMakefileMutations, limitsCIMutations,
	}
	for _, group := range groups {
		for _, mutation := range group {
			t.Run(mutation.name, func(t *testing.T) {
				assertGuardRejects(t, mutation)
			})
		}
	}
}

func assertGuardRejects(t *testing.T, mutation limitsMutation) {
	t.Helper()
	original := string(readRepoDoc(t, mutation.target.path...))
	require.Contains(t, original, mutation.old, "the mutation no longer applies to the real file")
	mutated := strings.Replace(original, mutation.old, mutation.replacement, 1)
	require.NotEqual(t, original, mutated, "the mutation must change the file")

	found := mutation.target.check([]byte(mutated))
	assert.Contains(t, strings.Join(found, "\n"), mutation.want)
}
