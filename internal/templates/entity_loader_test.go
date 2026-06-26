package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEntityTemplate_UsesEmbeddedDefaultWhenNoDiskTree(t *testing.T) {
	restoreTemplateGlobals(t)
	setWorkingDir(t, t.TempDir())

	SetConfiguredSharkDataPath("shark-data")

	content, err := LoadEntityTemplate("epic.md")
	if err != nil {
		t.Fatalf("LoadEntityTemplate() error = %v", err)
	}
	if !strings.Contains(string(content), "epic_key: {{.EpicSlug}}") {
		t.Fatalf("embedded epic template missing expected frontmatter: %s", string(content))
	}
}

func TestLoadEntityTemplate_UsesDiskOverrideBeforeDiskDefault(t *testing.T) {
	restoreTemplateGlobals(t)
	root := t.TempDir()
	setWorkingDir(t, root)

	defaultPath := filepath.Join(root, "shark-data", "templates", "epic.md")
	overridePath := filepath.Join(root, "shark-data", "overrides", "templates", "epic.md")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0755); err != nil {
		t.Fatalf("mkdir default template dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(overridePath), 0755); err != nil {
		t.Fatalf("mkdir override template dir: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte("default template"), 0644); err != nil {
		t.Fatalf("write default template: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte("override template"), 0644); err != nil {
		t.Fatalf("write override template: %v", err)
	}

	SetConfiguredSharkDataPath("shark-data")

	content, err := LoadEntityTemplate("epic.md")
	if err != nil {
		t.Fatalf("LoadEntityTemplate() error = %v", err)
	}
	if string(content) != "override template" {
		t.Fatalf("LoadEntityTemplate() = %q, want override template", string(content))
	}
}

func TestLoadEntityTemplate_RejectsNestedPath(t *testing.T) {
	restoreTemplateGlobals(t)

	_, err := LoadEntityTemplate("../epic.md")
	if err == nil {
		t.Fatal("LoadEntityTemplate() expected error for nested path, got nil")
	}
	if !strings.Contains(err.Error(), "must be a file name") {
		t.Fatalf("LoadEntityTemplate() error = %v, want file-name validation", err)
	}
}

func restoreTemplateGlobals(t *testing.T) {
	t.Helper()
	prevTemplateDir := configuredTemplateDir
	prevSharkDataPath := configuredSharkDataPath
	prevGetwd := getwd
	t.Cleanup(func() {
		configuredTemplateDir = prevTemplateDir
		configuredSharkDataPath = prevSharkDataPath
		getwd = prevGetwd
	})
}

func setWorkingDir(t *testing.T, dir string) {
	t.Helper()
	getwd = func() (string, error) {
		return dir, nil
	}
}
