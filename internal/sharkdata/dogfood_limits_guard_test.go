package sharkdata

import (
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const (
	limitsConfigFlag = "--config=.golangci-limits.yml"
	wholeFilesFlag   = "--whole-files"
)

var (
	fullCommitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
	// guardedByLimits matches an exclusion that names a limits linter or a
	// test-file path.
	guardedByLimits = regexp.MustCompile(`funlen|gocyclo|_test|test\\?\.go`)
	// swallowedFailure matches a shell tail that discards a command's failure.
	swallowedFailure = regexp.MustCompile(`(\|\||;)\s*true\b`)
)

// violations collects every rule the limits setup breaks.
type violations []string

func (v *violations) require(ok bool, message string) {
	if !ok {
		*v = append(*v, message)
	}
}

type limitsConfig struct {
	Run struct {
		Tests *bool `yaml:"tests"`
	} `yaml:"run"`
	Linters struct {
		Default    string   `yaml:"default"`
		Enable     []string `yaml:"enable"`
		Disable    []string `yaml:"disable"`
		Exclusions any      `yaml:"exclusions"`
		Settings   struct {
			Funlen struct {
				Lines      int `yaml:"lines"`
				Statements int `yaml:"statements"`
			} `yaml:"funlen"`
			Gocyclo struct {
				MinComplexity int `yaml:"min-complexity"`
			} `yaml:"gocyclo"`
		} `yaml:"settings"`
	} `yaml:"linters"`
	Issues map[string]any `yaml:"issues"`
}

// limitsConfigViolations reports how the limits config is loosened away from
// the values the coding standards declare, or stops enforcing them.
func limitsConfigViolations(data []byte) violations {
	var config limitsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return violations{"limits config does not parse: " + err.Error()}
	}
	return slices.Concat(
		limitsValueViolations(config),
		limitsLinterSetViolations(config),
		limitsExclusionViolations(config),
	)
}

func limitsValueViolations(config limitsConfig) violations {
	var found violations
	settings := config.Linters.Settings
	found.require(settings.Funlen.Lines == 50, "funlen lines must be 50")
	found.require(settings.Funlen.Statements == -1, "funlen statements must be disabled")
	found.require(settings.Gocyclo.MinComplexity == 10, "gocyclo min-complexity must be 10")
	revision, _ := config.Issues["new-from-rev"].(string)
	found.require(fullCommitSHA.MatchString(revision), "new-from-rev must be a full SHA")
	return found
}

// limitsLinterSetViolations reports a config that runs other linters than the
// two limits linters, or does not run them on every file.
func limitsLinterSetViolations(config limitsConfig) violations {
	var found violations
	linters := config.Linters
	enabled := slices.Sorted(slices.Values(linters.Enable))
	found.require(linters.Default == "none", "linters default must be none")
	found.require(slices.Equal(enabled, []string{"funlen", "gocyclo"}),
		"linters must be exactly funlen and gocyclo")
	found.require(len(linters.Disable) == 0, "linters must not disable any linter")
	found.require(config.Run.Tests == nil || *config.Run.Tests, "the limits must run on test files")
	return found
}

// limitsExclusionViolations reports exclusions that hide limits findings.
func limitsExclusionViolations(config limitsConfig) violations {
	var found violations
	found.require(!namesGuardedTarget(config.Linters.Exclusions),
		"linters.exclusions must not exclude a limits linter or test files")
	for key, value := range config.Issues {
		if strings.HasPrefix(key, "exclude") {
			found.require(!namesGuardedTarget(value),
				"issues."+key+" must not exclude a limits linter or test files")
		}
	}
	return found
}

// namesGuardedTarget reports whether any string under node names a limits linter
// or a test-file path.
func namesGuardedTarget(node any) bool {
	switch value := node.(type) {
	case string:
		return guardedByLimits.MatchString(value)
	case []any:
		return slices.ContainsFunc(value, namesGuardedTarget)
	case map[string]any:
		return slices.ContainsFunc(slices.Collect(maps.Values(value)), namesGuardedTarget)
	default:
		return false
	}
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

// makefileLimitsViolations reports how `make lint` stops running the main
// config and then the limits config over whole changed files.
func makefileLimitsViolations(data []byte) violations {
	recipe := makeRecipe(string(data), "lint")
	limits := linesWithField(recipe, limitsConfigFlag)
	if len(limits) != 1 {
		return violations{"make lint must run the limits config once"}
	}
	var found violations
	line := recipe[limits[0]]
	found.require(slices.Contains(strings.Fields(line), wholeFilesFlag),
		"make lint must measure whole files")
	// A recipe line may open with any mix of make's @, - and + prefixes.
	command := strings.TrimLeft(line, " \t@-+")
	prefix := strings.TrimSuffix(line, command)
	found.require(!strings.HasPrefix(command, "#"), "the limits line must not be commented out")
	found.require(!strings.Contains(prefix, "-"), "the limits line must not ignore errors")
	found.require(!swallowedFailure.MatchString(line), "the limits line must not swallow its failure")

	// The main run keeps its own line, ahead of the limits run, so its failure
	// still stops the target.
	mainRun := slices.IndexFunc(recipe, func(line string) bool {
		return strings.HasSuffix(line, "&& golangci-lint run")
	})
	found.require(mainRun >= 0 && mainRun < limits[0],
		"make lint must run the main config before the limits config")
	return found
}

// ciLimitsSteps returns the lint job steps that run the limits config.
func ciLimitsSteps(data []byte) ([]map[string]any, error) {
	var workflow struct {
		Jobs map[string]struct {
			Steps []map[string]any `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, err
	}
	var limits []map[string]any
	for _, step := range workflow.Jobs["lint"].Steps {
		uses, _ := step["uses"].(string)
		isLintStep := strings.HasPrefix(uses, "golangci/golangci-lint-action@")
		if isLintStep && slices.Contains(stepArgs(step), limitsConfigFlag) {
			limits = append(limits, step)
		}
	}
	return limits, nil
}

func stepArgs(step map[string]any) []string {
	with, _ := step["with"].(map[string]any)
	args, _ := with["args"].(string)
	return strings.Fields(args)
}

// ciLimitsViolations reports how the CI lint job stops running the limits
// config over whole changed files as its own step that can fail the job.
func ciLimitsViolations(data []byte) violations {
	steps, err := ciLimitsSteps(data)
	if err != nil {
		return violations{"CI workflow does not parse: " + err.Error()}
	}
	if len(steps) != 1 {
		return violations{"the lint job must have one limits step"}
	}
	var found violations
	found.require(slices.Contains(stepArgs(steps[0]), wholeFilesFlag),
		"the CI limits step must measure whole files")
	_, continues := steps[0]["continue-on-error"]
	found.require(!continues, "the CI limits step must have no continue-on-error")
	_, conditional := steps[0]["if"]
	found.require(!conditional, "the CI limits step must have no if condition")
	return found
}

// TestE31F01_TC025_LimitsConfigKeepsDeclaredValues fails when the limits config
// is loosened away from the values the coding standards declare.
func TestE31F01_TC025_LimitsConfigKeepsDeclaredValues(t *testing.T) {
	assert.Empty(t, limitsConfigViolations(readRepoDoc(t, ".golangci-limits.yml")))
}

// TestE31F01_TC025_MakeLintMeasuresWholeEditedFiles requires `make lint` to run
// the main config and then the limits config over whole changed files.
func TestE31F01_TC025_MakeLintMeasuresWholeEditedFiles(t *testing.T) {
	assert.Empty(t, makefileLimitsViolations(readRepoDoc(t, "Makefile")))
}

// TestE31F01_TC025_CILintJobMeasuresWholeEditedFiles requires the same limits
// run as its own step of the CI lint job.
func TestE31F01_TC025_CILintJobMeasuresWholeEditedFiles(t *testing.T) {
	assert.Empty(t, ciLimitsViolations(readRepoDoc(t, ".github", "workflows", "ci.yml")))
}
