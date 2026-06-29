package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupBundleContentProject(t *testing.T, configJSON string) string {
	t.Helper()

	root := t.TempDir()
	if configJSON == "" {
		configJSON = `{}`
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".sharkconfig.json"), []byte(configJSON), 0644))
	return root
}

func writeBundleFile(t *testing.T, root, relPath, content string) {
	t.Helper()

	fullPath := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
}

func TestBundleContentServiceGetEmbeddedImplementationSkillWithoutDiskBundle(t *testing.T) {
	root := setupBundleContentProject(t, `{}`)
	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	result, err := svc.Get(context.Background(), BundleContentKindSkill, "implementation", "", BundleContentGetOptions{})
	require.NoError(t, err)

	assert.Equal(t, "skill", result.Kind)
	assert.Equal(t, "implementation", result.Name)
	assert.Equal(t, "SKILL.md", result.Path)
	assert.Equal(t, "embedded", result.Source)
	assert.True(t, result.Resolved)
	assert.False(t, result.Raw)
	assert.Contains(t, result.Content, "# Implementation Skill")
	assert.NotRegexp(t, `(?m)\A---\nname:`, result.Content, "default output should strip top-level frontmatter")
}

func TestBundleContentServiceGetEmbeddedFeatureDesignSkillWithoutDiskBundle(t *testing.T) {
	root := setupBundleContentProject(t, `{}`)
	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	result, err := svc.Get(context.Background(), BundleContentKindSkill, "feature-design", "", BundleContentGetOptions{})
	require.NoError(t, err)

	assert.Equal(t, "skill", result.Kind)
	assert.Equal(t, "feature-design", result.Name)
	assert.Equal(t, "SKILL.md", result.Path)
	assert.Equal(t, "embedded", result.Source)
	assert.Contains(t, result.Content, "# Feature Design Skill")
}

func TestBundleContentServiceGetEmbeddedFeatureDesignWireframesWorkflow(t *testing.T) {
	root := setupBundleContentProject(t, `{}`)
	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	result, err := svc.Get(context.Background(), BundleContentKindSkill, "feature-design", "workflows/wireframes.md", BundleContentGetOptions{})
	require.NoError(t, err)

	assert.Equal(t, "skill", result.Kind)
	assert.Equal(t, "feature-design", result.Name)
	assert.Equal(t, "workflows/wireframes.md", result.Path)
	assert.Equal(t, "embedded", result.Source)
	assert.Contains(t, result.Content, "# Wireframes (Feature-Level)")
}

func TestBundleContentServiceGetEmbeddedAgentWithoutDiskBundle(t *testing.T) {
	root := setupBundleContentProject(t, `{}`)
	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	result, err := svc.Get(context.Background(), BundleContentKindAgent, "developer", "", BundleContentGetOptions{})
	require.NoError(t, err)

	assert.Equal(t, "agent", result.Kind)
	assert.Equal(t, "developer", result.Name)
	assert.Equal(t, "developer.md", result.Path)
	assert.Equal(t, "embedded", result.Source)
	assert.Contains(t, result.Content, "# Developer Agent")
	assert.NotRegexp(t, `(?m)\A---\nname:`, result.Content)
}

func TestBundleContentServiceDiskDefaultWinsOverEmbedded(t *testing.T) {
	root := setupBundleContentProject(t, `{"shark_data_path":"bundle"}`)
	writeBundleFile(t, root, "bundle/skills/triage/SKILL.md", "---\nname: triage\n---\nDISK TRIAGE")
	writeBundleFile(t, root, "bundle/agents/developer.md", "---\nname: developer\n---\nDISK DEVELOPER")

	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	skill, err := svc.Get(context.Background(), BundleContentKindSkill, "triage", "", BundleContentGetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "disk", skill.Source)
	assert.Equal(t, "DISK TRIAGE", strings.TrimSpace(skill.Content))

	agent, err := svc.Get(context.Background(), BundleContentKindAgent, "developer", "", BundleContentGetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "disk", agent.Source)
	assert.Equal(t, "DISK DEVELOPER", strings.TrimSpace(agent.Content))
}

func TestBundleContentServiceOverrideWinsOverDiskAndEmbedded(t *testing.T) {
	root := setupBundleContentProject(t, `{"shark_data_path":"bundle"}`)
	writeBundleFile(t, root, "bundle/skills/triage/SKILL.md", "DISK TRIAGE")
	writeBundleFile(t, root, "bundle/overrides/skills/triage/SKILL.md", "OVERRIDE TRIAGE")
	writeBundleFile(t, root, "bundle/agents/developer.md", "DISK DEVELOPER")
	writeBundleFile(t, root, "bundle/overrides/agents/developer.md", "OVERRIDE DEVELOPER")

	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	skill, err := svc.Get(context.Background(), BundleContentKindSkill, "triage", "", BundleContentGetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "override", skill.Source)
	assert.Equal(t, "OVERRIDE TRIAGE", strings.TrimSpace(skill.Content))

	agent, err := svc.Get(context.Background(), BundleContentKindAgent, "developer", "", BundleContentGetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "override", agent.Source)
	assert.Equal(t, "OVERRIDE DEVELOPER", strings.TrimSpace(agent.Content))
}

func TestBundleContentServiceRawPreservesStoredBytes(t *testing.T) {
	root := setupBundleContentProject(t, `{"shark_data_path":"bundle"}`)
	writeBundleFile(t, root, "bundle/skills/example/SKILL.md", "---\nname: example\n---\nHello {{include: skills/example/context.md}} {{.task_id}} <task-id>")
	writeBundleFile(t, root, "bundle/skills/example/context.md", "CTX")

	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	resolved, err := svc.Get(context.Background(), BundleContentKindSkill, "example", "", BundleContentGetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "Hello CTX {{.task_id}} <task-id>", strings.TrimSpace(resolved.Content))
	assert.True(t, resolved.Resolved)

	raw, err := svc.Get(context.Background(), BundleContentKindSkill, "example", "", BundleContentGetOptions{Raw: true})
	require.NoError(t, err)
	assert.Contains(t, raw.Content, "---\nname: example\n---")
	assert.Contains(t, raw.Content, "{{include: skills/example/context.md}}")
	assert.False(t, raw.Resolved)
	assert.True(t, raw.Raw)
}

func TestBundleContentServiceGetRejectsUnsafePaths(t *testing.T) {
	root := setupBundleContentProject(t, `{}`)
	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	tests := []struct {
		name    string
		item    string
		relPath string
	}{
		{name: "absolute path", item: "triage", relPath: "/etc/passwd"},
		{name: "path traversal", item: "triage", relPath: "../README.md"},
		{name: "name traversal", item: "../triage", relPath: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Get(context.Background(), BundleContentKindSkill, tt.item, tt.relPath, BundleContentGetOptions{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "must be relative")
		})
	}
}

func TestBundleContentServiceMissingNameReturnsNotFound(t *testing.T) {
	root := setupBundleContentProject(t, `{}`)
	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	_, err = svc.Get(context.Background(), BundleContentKindSkill, "does-not-exist", "", BundleContentGetOptions{})
	require.Error(t, err)

	var notFound *BundleContentNotFoundError
	assert.True(t, errors.As(err, &notFound), "expected BundleContentNotFoundError, got %T", err)
	assert.Contains(t, err.Error(), `skill not found: does-not-exist`)
}

func TestBundleContentServiceListDedupesByPrecedence(t *testing.T) {
	root := setupBundleContentProject(t, `{"shark_data_path":"bundle"}`)
	writeBundleFile(t, root, "bundle/skills/triage/SKILL.md", "DISK TRIAGE")
	writeBundleFile(t, root, "bundle/overrides/skills/triage/SKILL.md", "OVERRIDE TRIAGE")
	writeBundleFile(t, root, "bundle/skills/local-only/SKILL.md", "LOCAL")

	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	entries, err := svc.List(context.Background(), BundleContentKindSkill)
	require.NoError(t, err)

	names := map[string]string{}
	for _, entry := range entries {
		names[entry.Name] = entry.Source
	}

	assert.Equal(t, "override", names["triage"])
	assert.Equal(t, "disk", names["local-only"])
	assert.Equal(t, "embedded", names["implementation"])

	var last string
	for _, entry := range entries {
		assert.GreaterOrEqual(t, entry.Name, last, "entries should be sorted by name")
		last = entry.Name
	}
}

func TestBundleContentServiceListSkipsHiddenAndPrivateDiskNames(t *testing.T) {
	root := setupBundleContentProject(t, `{"shark_data_path":"bundle"}`)
	writeBundleFile(t, root, "bundle/skills/visible-skill/SKILL.md", "VISIBLE")
	writeBundleFile(t, root, "bundle/skills/.hidden-skill/SKILL.md", "HIDDEN")
	writeBundleFile(t, root, "bundle/skills/_extracted/SKILL.md", "PRIVATE")
	writeBundleFile(t, root, "bundle/agents/visible-agent.md", "VISIBLE")
	writeBundleFile(t, root, "bundle/agents/.hidden-agent.md", "HIDDEN")
	writeBundleFile(t, root, "bundle/agents/_private-agent.md", "PRIVATE")
	writeBundleFile(t, root, "bundle/agents/.hidden-agent-dir/.hidden-agent-dir.md", "HIDDEN")
	writeBundleFile(t, root, "bundle/agents/_private-agent-dir/_private-agent-dir.md", "PRIVATE")

	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	skills, err := svc.List(context.Background(), BundleContentKindSkill)
	require.NoError(t, err)
	agents, err := svc.List(context.Background(), BundleContentKindAgent)
	require.NoError(t, err)

	skillNames := bundleListNames(skills)
	agentNames := bundleListNames(agents)
	assert.Contains(t, skillNames, "visible-skill")
	assert.NotContains(t, skillNames, ".hidden-skill")
	assert.NotContains(t, skillNames, "_extracted")
	assert.Contains(t, agentNames, "visible-agent")
	assert.NotContains(t, agentNames, ".hidden-agent")
	assert.NotContains(t, agentNames, "_private-agent")
	assert.NotContains(t, agentNames, ".hidden-agent-dir")
	assert.NotContains(t, agentNames, "_private-agent-dir")
}

func TestEmbeddedNameHelpersSkipHiddenAndPrivateNames(t *testing.T) {
	fsys := fstest.MapFS{
		"skills/visible-skill/SKILL.md":                   {},
		"skills/.hidden-skill/SKILL.md":                   {},
		"skills/_extracted/SKILL.md":                      {},
		"agents/visible-agent.md":                         {},
		"agents/.hidden-agent.md":                         {},
		"agents/_private-agent.md":                        {},
		"agents/.hidden-agent-dir/.hidden-agent-dir.md":   {},
		"agents/_private-agent-dir/_private-agent-dir.md": {},
	}

	assert.Equal(t, []string{"visible-skill"}, embeddedDirectoryNames(fsys, "skills"))
	assert.Equal(t, []string{"visible-agent"}, embeddedMarkdownNames(fsys, "agents"))
}

func bundleListNames(entries []BundleContentEntry) map[string]bool {
	names := make(map[string]bool, len(entries))
	for _, entry := range entries {
		names[entry.Name] = true
	}
	return names
}

func TestBundleContentServiceListPopulatesDescription(t *testing.T) {
	root := setupBundleContentProject(t, `{"shark_data_path":"bundle"}`)
	writeBundleFile(t, root, "bundle/skills/my-skill/SKILL.md", "---\nname: my-skill\ndescription: A useful skill\n---\n# My Skill\n")
	writeBundleFile(t, root, "bundle/skills/no-desc-skill/SKILL.md", "---\nname: no-desc\n---\n# No Desc\n")
	writeBundleFile(t, root, "bundle/agents/my-agent.md", "---\nname: my-agent\ndescription: A useful agent\n---\n# My Agent\n")

	svc, err := NewBundleContentService(root)
	require.NoError(t, err)

	skills, err := svc.List(context.Background(), BundleContentKindSkill)
	require.NoError(t, err)

	descMap := map[string]string{}
	for _, entry := range skills {
		descMap[entry.Name] = entry.Description
	}
	assert.Equal(t, "A useful skill", descMap["my-skill"])
	assert.Equal(t, "", descMap["no-desc-skill"])

	agents, err := svc.List(context.Background(), BundleContentKindAgent)
	require.NoError(t, err)

	for _, entry := range agents {
		if entry.Name == "my-agent" {
			assert.Equal(t, "A useful agent", entry.Description)
			return
		}
	}
	t.Error("my-agent not found in agent list")
}
