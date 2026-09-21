package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestStatusDocumentationUsesNamespaceContract prevents B070 from regressing:
// status is a workflow namespace, while progress and get own inspection views.
func TestStatusDocumentationUsesNamespaceContract(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))

	documents := []struct {
		path     string
		contains []string
		forbids  []string
	}{
		{
			path: filepath.Join(".claude", "rules", "quickref.md"),
			contains: []string{
				"shark progress",
				"shark get <key>",
				"shark status transitions",
			},
			forbids: []string{
				"shark status                               # Project dashboard",
				"shark status E07                           # Epic status",
				"shark status E07-F01                       # Feature status",
				"shark status options E07-F01-001",
			},
		},
		{
			path: filepath.Join("docs", "CLI_REFERENCE.md"),
			contains: []string{
				"shark progress",
				"shark get E07-F01-001 --field status",
				"shark status transitions",
			},
			forbids: []string{
				"shark status                               # Project dashboard",
			},
		},
		{
			path: filepath.Join("docs", "cli-reference", "README.md"),
			contains: []string{
				"`shark progress [EPIC] [FEATURE]`",
				"`shark get <key>`",
				"`shark status set <key> <status>`",
				"`shark status transitions <key>`",
			},
			forbids: []string{
				"`shark status [key]` | Project dashboard or entity status",
				"`shark status options <key>`",
			},
		},
		{
			path: filepath.Join("docs", "cli-reference", "status-commands.md"),
			contains: []string{
				"`shark status` is a subcommand namespace",
				"shark progress [EPIC] [FEATURE]",
				"shark get <key> --field status",
				"## shark status transitions",
			},
			forbids: []string{
				"- `shark status` - Display status dashboard",
				"shark status [EPIC] [FEATURE] [flags]",
				"shark status E05",
				"shark status options",
			},
		},
		{
			path: filepath.Join(".claude", "rules", "cli", "commands.md"),
			contains: []string{
				"`shark status` — Status operation namespace",
				"`shark status transitions <key>`",
			},
			forbids: []string{
				"`shark status [KEY]` — Project dashboard or entity status",
				"`shark status options <key>`",
			},
		},
	}

	for _, document := range documents {
		document := document
		t.Run(document.path, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(projectRoot, document.path))
			if err != nil {
				t.Fatalf("read %s: %v", document.path, err)
			}
			text := string(content)
			for _, expected := range document.contains {
				if !strings.Contains(text, expected) {
					t.Errorf("%s does not document %q", document.path, expected)
				}
			}
			for _, forbidden := range document.forbids {
				if strings.Contains(text, forbidden) {
					t.Errorf("%s still contains stale status contract %q", document.path, forbidden)
				}
			}
		})
	}
}
