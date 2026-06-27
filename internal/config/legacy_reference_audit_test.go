package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperationalScopesDoNotReferenceDeprecatedTemplateTree(t *testing.T) {
	repoRoot := findRepoRootForAudit(t)
	deprecatedName := "shark" + "-templates"

	scopes := []struct {
		path   string
		goOnly bool
	}{
		{path: "internal", goOnly: true},
		{path: "README.md"},
		{path: ".claude"},
		{path: filepath.Join("docs", "cli-reference")},
		{path: filepath.Join("docs", "architecture")},
		{path: "examples"},
		{path: "test-fixtures"},
		{path: filepath.Join("internal", "sharkdata", "default_data")},
	}

	var hits []string
	for _, scope := range scopes {
		root := filepath.Join(repoRoot, scope.path)
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("stat %s: %v", root, err)
		}
		if !info.IsDir() {
			if fileContainsDeprecatedName(t, root, deprecatedName) {
				hits = append(hits, scope.path)
			}
			continue
		}
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if scope.goOnly && filepath.Ext(path) != ".go" {
				return nil
			}
			if fileContainsDeprecatedName(t, path, deprecatedName) {
				rel, err := filepath.Rel(repoRoot, path)
				if err != nil {
					return err
				}
				hits = append(hits, rel)
			}
			return nil
		}); err != nil {
			t.Fatalf("scan %s: %v", scope.path, err)
		}
	}

	if len(hits) > 0 {
		t.Fatalf("deprecated template tree references found in operational scopes:\n%s", strings.Join(hits, "\n"))
	}
}

func findRepoRootForAudit(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root from test working directory")
		}
	}
}

func fileContainsDeprecatedName(t *testing.T, path, deprecatedName string) bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Contains(string(data), deprecatedName)
}
