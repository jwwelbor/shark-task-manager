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

func TestAgentGetJSONIncludesResolutionMetadata(t *testing.T) {
	setupContentCommandProject(t, `{}`)
	cli.GlobalConfig.JSON = true

	var runErr error
	out := captureOutput(t, func() {
		runErr = runBundleContentGet(testContentGetCommand(false), services.BundleContentKindAgent, []string{"developer"})
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &payload))
	assert.Equal(t, "agent", payload["kind"])
	assert.Equal(t, "developer", payload["name"])
	assert.Equal(t, "developer.md", payload["path"])
	assert.Equal(t, "embedded", payload["source"])
	assert.Contains(t, payload["content"], "# Developer Agent")
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

	for _, entry := range entries {
		assert.Contains(t, entry, "name")
		assert.Contains(t, entry, "description")
		assert.NotContains(t, entry, "source")
	}

	var seenTriage int
	for _, entry := range entries {
		if strings.EqualFold(entry["name"], "triage") {
			seenTriage++
		}
	}
	assert.Equal(t, 1, seenTriage, "list should include one logical entry per name")
}

func TestBundleContentListHumanOutputIsCompactByDefault(t *testing.T) {
	root := setupContentCommandProject(t, `{"shark_data_path":"bundle"}`)
	writeContentCommandBundleFile(t, root, "bundle/skills/zebra/SKILL.md", "---\nname: zebra\ndescription: A zebra skill\n---\n")
	writeContentCommandBundleFile(t, root, "bundle/skills/alpha/SKILL.md", "---\nname: alpha\ndescription: An alpha skill\n---\n")

	origNoColor := cli.GlobalConfig.NoColor
	t.Cleanup(func() { cli.GlobalConfig.NoColor = origNoColor })
	cli.GlobalConfig.NoColor = false

	var runErr error
	out := captureOutput(t, func() {
		runErr = runBundleContentList(&cobra.Command{}, services.BundleContentKindSkill)
	})
	require.NoError(t, runErr)

	text := string(out)
	assert.Contains(t, text, cli.ColorCyan+"alpha"+cli.ColorReset)
	assert.Contains(t, text, cli.ColorCyan+"zebra"+cli.ColorReset)
	assert.NotContains(t, text, "An alpha skill")
	assert.NotContains(t, text, "A zebra skill")
	assert.Contains(t, text, "Use --all to show descriptions.")
	assert.Less(t, strings.Index(text, "alpha"), strings.Index(text, "zebra"))
}

func TestBundleContentListHumanOutputAllShowsDescriptionsAndRespectsNoColor(t *testing.T) {
	root := setupContentCommandProject(t, `{"shark_data_path":"bundle"}`)
	writeContentCommandBundleFile(t, root, "bundle/skills/alpha/SKILL.md", "---\nname: alpha\ndescription: A skill description\n---\n")
	writeContentCommandBundleFile(t, root, "bundle/agents/alpha.md", "---\nname: alpha\ndescription: An alpha agent\n---\n")
	writeContentCommandBundleFile(t, root, "bundle/agents/no-desc.md", "---\nname: no-desc\n---\n")

	origNoColor := cli.GlobalConfig.NoColor
	t.Cleanup(func() { cli.GlobalConfig.NoColor = origNoColor })
	cli.GlobalConfig.NoColor = true

	for _, test := range []struct {
		name        string
		cmd         *cobra.Command
		kind        services.BundleContentKind
		description string
	}{
		{name: "skills", cmd: skillListCmd, kind: services.BundleContentKindSkill, description: "A skill description"},
		{name: "agents", cmd: agentListCmd, kind: services.BundleContentKindAgent, description: "An alpha agent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			setBundleContentAllFlag(t, test.cmd)

			var runErr error
			out := captureOutput(t, func() {
				runErr = runBundleContentList(test.cmd, test.kind)
			})
			require.NoError(t, runErr)

			text := string(out)
			assert.Contains(t, text, "alpha\n"+test.description+"\n")
			assert.NotContains(t, text, "\033[")
			assert.NotContains(t, text, "Use --all to show descriptions.")
		})
	}
}

func TestBundleContentListJSONAllIncludesSourcesForAgentsAndSkills(t *testing.T) {
	root := setupContentCommandProject(t, `{"shark_data_path":"bundle"}`)
	writeContentCommandBundleFile(t, root, "bundle/skills/no-desc/SKILL.md", "---\nname: no-desc\n---\n")
	writeContentCommandBundleFile(t, root, "bundle/agents/no-desc.md", "---\nname: no-desc\n---\n")
	cli.GlobalConfig.JSON = true

	for _, test := range []struct {
		name string
		cmd  *cobra.Command
		kind services.BundleContentKind
	}{
		{name: "skills", cmd: skillListCmd, kind: services.BundleContentKindSkill},
		{name: "agents", cmd: agentListCmd, kind: services.BundleContentKindAgent},
	} {
		t.Run(test.name, func(t *testing.T) {
			setBundleContentAllFlag(t, test.cmd)
			var runErr error
			out := captureOutput(t, func() {
				runErr = runBundleContentList(test.cmd, test.kind)
			})
			require.NoError(t, runErr)

			var entries []map[string]string
			require.NoError(t, json.Unmarshal(out, &entries))
			entry := findBundleContentListEntry(t, entries, "no-desc")
			assert.Equal(t, "disk", entry["source"])
		})
	}
}

func TestBundleContentListDefaultJSONIncludesDescriptionsWithoutSourcesForAgentsAndSkills(t *testing.T) {
	root := setupContentCommandProject(t, `{"shark_data_path":"bundle"}`)
	writeContentCommandBundleFile(t, root, "bundle/skills/no-desc/SKILL.md", "---\nname: no-desc\n---\n")
	writeContentCommandBundleFile(t, root, "bundle/agents/no-desc.md", "---\nname: no-desc\n---\n")
	cli.GlobalConfig.JSON = true

	for _, kind := range []services.BundleContentKind{services.BundleContentKindSkill, services.BundleContentKindAgent} {
		t.Run(string(kind), func(t *testing.T) {
			var runErr error
			out := captureOutput(t, func() {
				runErr = runBundleContentList(&cobra.Command{}, kind)
			})
			require.NoError(t, runErr)

			var entries []map[string]string
			require.NoError(t, json.Unmarshal(out, &entries))
			for _, entry := range entries {
				assert.Contains(t, entry, "name")
				assert.Contains(t, entry, "description")
				assert.NotContains(t, entry, "source")
			}
		})
	}
}

func findRegisteredCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func setBundleContentAllFlag(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	flag := cmd.Flags().Lookup("all")
	require.NotNil(t, flag, "registered list command must define --all")
	previousValue := flag.Value.String()
	require.NoError(t, cmd.Flags().Set("all", "true"))
	t.Cleanup(func() {
		require.NoError(t, cmd.Flags().Set("all", previousValue))
	})
}

func findBundleContentListEntry(t *testing.T, entries []map[string]string, name string) map[string]string {
	t.Helper()
	for _, entry := range entries {
		if entry["name"] == name {
			return entry
		}
	}
	t.Fatalf("bundle content entry %q not found", name)
	return nil
}
