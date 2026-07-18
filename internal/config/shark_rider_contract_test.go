package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharkRiderHostContract(t *testing.T) {
	root := findRepoRootForAudit(t)

	_, err := os.Stat(filepath.Join(root, "skills", "shark"))
	require.True(t, os.IsNotExist(err), "legacy host skill directory must not remain")

	skill := readContractFile(t, root, "skills", "shark-rider", "SKILL.md")
	assertContainsAll(t, skill, "host skill identity", []string{
		"name: shark-rider",
		"# Shark Rider",
		"/shark-rider project bootstrap",
		"shark <command>",
		"CLI owns, Rider drives",
	})

	project := readContractFile(t, root, "skills", "shark-rider", "verbs", "project.md")
	assertContainsAll(t, project, "bootstrap coordinator", []string{
		"shark admin init --non-interactive",
		"shark admin validate",
		"/shark-rider project product-design",
		"/shark-rider project brownfield-analysis",
		"docs/product/progress.md",
		"schema_version: 2",
		"[~]",
		"D07",
		"feasible as described",
	})
	if strings.Contains(project, "shark skill get product-design") {
		t.Error("bootstrap must load the product-design Rider procedure instead of owning its bundle retrieval")
	}

	product := readContractFile(t, root, "skills", "shark-rider", "verbs", "product-design.md")
	assertContainsAll(t, product, "product-design adapter", []string{
		"shark skill get product-design",
		"After each D01–D14 artifact",
		"same coordination run",
		"docs/product/progress.md",
	})

	template := readContractFile(t, root, "file_templates", "progress.md")
	assertContainsAll(t, template, "progress template", []string{
		"schema_version: 2",
		"estate:",
		"initiative_posture:",
		"product_design_scope:",
		"brownfield_depth:",
		"architecture_state:",
		"stack_summary:",
		"artifact_paths:",
		"last_refreshed:",
		"[~]",
		"## Cross-Epic Integration Map",
		"## Decision Log",
	})
}

func TestSharkRiderRunAliasRoutesToCoreLoop(t *testing.T) {
	root := findRepoRootForAudit(t)
	skill := readContractFile(t, root, "skills", "shark-rider", "SKILL.md")

	assert.Contains(t, skill, "`/run <key>` is an alias for `/shark-rider run <key>`")
	assert.Contains(t, skill, "**`/shark-rider run <key>`** (alias `/run <key>`) — the core loop")
	assert.Contains(t, skill, "`/shark-rider run` and `/run` both route to `verbs/run.md`")
	assert.NotContains(t, skill, "`/shark-rider run <key>` is an alias for `/shark-rider run <key>`")
}

func TestSharkRiderBootstrapChildHandoffContract(t *testing.T) {
	root := findRepoRootForAudit(t)
	project := readContractFile(t, root, "skills", "shark-rider", "verbs", "project.md")
	product := readContractFile(t, root, "skills", "shark-rider", "verbs", "product-design.md")
	brownfield := readContractFile(t, root, "skills", "shark-rider", "verbs", "brownfield-analysis.md")
	execution := readContractFile(t, root, "skills", "shark-rider", "context", "project-bootstrap-execution.md")

	assert.Contains(t, project, "skills/shark-rider/verbs/product-design.md")
	assert.Contains(t, project, "skills/shark-rider/verbs/brownfield-analysis.md")
	assert.Contains(t, project, "CHILD ACTION RESULT")
	assert.Contains(t, project, "same coordination run")
	assert.NotContains(t, project, "Dispatch `/shark-rider project product-design`")
	assert.NotContains(t, project, "Dispatch `/shark-rider project brownfield-analysis`")

	assert.Contains(t, product, "shark skill get product-design")
	assert.Contains(t, product, "Return a `CHILD ACTION RESULT`")
	assert.Contains(t, product, "stack_feedback")
	assert.NotContains(t, product, "dispatch `/shark-rider project bootstrap`")
	assert.Contains(t, brownfield, "CHILD ACTION RESULT")
	assert.NotContains(t, brownfield, "dispatch another host-level Rider action")

	for _, checkpoint := range []string{
		"created before product or analysis work starts",
		"changes after the durable artifact",
		"Interrupt after one artifact",
		"bootstrap consumes it",
		"feasible as described",
	} {
		assert.Contains(t, execution, checkpoint)
	}
}

func TestSharkRiderProjectInitCompatibilityRoute(t *testing.T) {
	root := findRepoRootForAudit(t)
	alias := readContractFile(t, root, "skills", "shark-rider", "verbs", "project-init.md")

	assert.Contains(t, alias, "deprecated")
	assert.Contains(t, alias, "/shark-rider project bootstrap")
	assert.Contains(t, alias, "skills/shark-rider/verbs/project.md")
	assert.NotContains(t, alias, "shark admin init")
}

func TestActiveBundleAndRiderInstructionsUseRiderSyntax(t *testing.T) {
	root := findRepoRootForAudit(t)
	paths := []string{
		filepath.Join(root, "skills", "shark-rider"),
		filepath.Join(root, "internal", "sharkdata", "default_data", "skills"),
	}

	for _, base := range paths {
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sanitized := strings.ReplaceAll(string(contents), "/shark-rider", "")
			if strings.Contains(sanitized, "/shark") {
				t.Errorf("active Rider or bundle instruction uses the legacy host syntax: %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk active instruction tree %s: %v", base, err)
		}
	}
}

func readContractFile(t *testing.T, root string, parts ...string) string {
	t.Helper()
	path := filepath.Join(root, filepath.Join(parts...))
	contents, err := os.ReadFile(path)
	if err != nil {
		require.NoErrorf(t, err, "read %s", path)
	}
	return string(contents)
}

func assertContainsAll(t *testing.T, contents, description string, needles []string) {
	t.Helper()
	for _, needle := range needles {
		assert.Containsf(t, contents, needle, "%s is missing %q", description, needle)
	}
}
