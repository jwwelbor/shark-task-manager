package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupContentCommandProject(t *testing.T, configJSON string) string {
	t.Helper()

	root := t.TempDir()
	if configJSON == "" {
		configJSON = `{}`
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".sharkconfig.json"), []byte(configJSON), 0644))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(root))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	origJSON := cli.GlobalConfig.JSON
	origField := cli.GlobalConfig.Field
	t.Cleanup(func() {
		cli.GlobalConfig.JSON = origJSON
		cli.GlobalConfig.Field = origField
	})
	cli.GlobalConfig.JSON = false
	cli.GlobalConfig.Field = ""

	return root
}

func writeContentCommandBundleFile(t *testing.T, root, relPath, content string) {
	t.Helper()

	fullPath := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
}

func testContentGetCommand(raw bool) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("raw", raw, "")
	return cmd
}

func TestSkillAndAgentCommandsRegistered(t *testing.T) {
	require.NotNil(t, findRegisteredCommand(cli.RootCmd, "skill"), "top-level skill command should be registered")
	require.NotNil(t, findRegisteredCommand(cli.RootCmd, "agent"), "top-level agent command should be registered")
}

func TestSkillGetImplementationHumanOutputFromEmbedded(t *testing.T) {
	setupContentCommandProject(t, `{}`)

	var runErr error
	out := captureOutput(t, func() {
		runErr = runBundleContentGet(testContentGetCommand(false), services.BundleContentKindSkill, []string{"implementation"})
	})
	require.NoError(t, runErr)

	text := string(out)
	assert.Contains(t, text, "# Implementation Skill")
	assert.NotRegexp(t, `(?m)\A---\nname:`, text, "human get output should print content only without frontmatter")
}

func TestAgentGetDeveloperHumanOutputFromEmbedded(t *testing.T) {
	setupContentCommandProject(t, `{}`)

	var runErr error
	out := captureOutput(t, func() {
		runErr = runBundleContentGet(testContentGetCommand(false), services.BundleContentKindAgent, []string{"developer"})
	})
	require.NoError(t, runErr)

	text := string(out)
	assert.Contains(t, text, "# Developer Agent")
	assert.NotRegexp(t, `(?m)\A---\nname:`, text)
}

func TestBundleContentGetRawPreservesFrontmatter(t *testing.T) {
	setupContentCommandProject(t, `{}`)

	var runErr error
	out := captureOutput(t, func() {
		runErr = runBundleContentGet(testContentGetCommand(true), services.BundleContentKindSkill, []string{"implementation"})
	})
	require.NoError(t, runErr)

	assert.Regexp(t, `(?m)\A---\nname: implementation`, string(out))
}

func TestSkillGetJSONIncludesResolutionMetadata(t *testing.T) {
	setupContentCommandProject(t, `{}`)
	cli.GlobalConfig.JSON = true

	var runErr error
	out := captureOutput(t, func() {
		runErr = runBundleContentGet(testContentGetCommand(false), services.BundleContentKindSkill, []string{"implementation"})
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &payload))
	assert.Equal(t, "skill", payload["kind"])
	assert.Equal(t, "implementation", payload["name"])
	assert.Equal(t, "SKILL.md", payload["path"])
	assert.Equal(t, "embedded", payload["source"])
	assert.Equal(t, true, payload["resolved"])
	assert.Equal(t, false, payload["raw"])
	assert.Contains(t, payload["content"], "# Implementation Skill")
}

func TestSkillListJSONIncludesEmbeddedAndDedupesOverrides(t *testing.T) {
	root := setupContentCommandProject(t, `{"shark_data_path":"bundle"}`)
	writeContentCommandBundleFile(t, root, "bundle/skills/triage/SKILL.md", "DISK")
	writeContentCommandBundleFile(t, root, "bundle/overrides/skills/triage/SKILL.md", "OVERRIDE")
	cli.GlobalConfig.JSON = true

	var runErr error
	out := captureOutput(t, func() {
		runErr = runBundleContentList(&cobra.Command{}, services.BundleContentKindSkill)
	})
	require.NoError(t, runErr)

	var entries []map[string]string
	require.NoError(t, json.Unmarshal(out, &entries))

	sourcesByName := map[string]string{}
	for _, entry := range entries {
		sourcesByName[entry["name"]] = entry["source"]
	}

	assert.Equal(t, "override", sourcesByName["triage"])
	assert.Equal(t, "embedded", sourcesByName["feature-design"])
	assert.Equal(t, "embedded", sourcesByName["implementation"])

	var seenTriage int
	for _, entry := range entries {
		if strings.EqualFold(entry["name"], "triage") {
			seenTriage++
		}
	}
	assert.Equal(t, 1, seenTriage, "list should include one logical entry per name")
}

func findRegisteredCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}
