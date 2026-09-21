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
				"`shark progress [EPIC]`",
				"`shark get <key>`",
				"`shark status set <key> <status>`",
				"`shark status transitions <key>`",
			},
			forbids: []string{
				"`shark status [key]` | Project dashboard or entity status",
				"`shark status options <key>`",
				"| `shark progress <key>` | Detailed progress breakdown |",
			},
		},
		{
			path: filepath.Join("docs", "cli-reference", "status-commands.md"),
			contains: []string{
				"`shark status` is a subcommand namespace",
				"shark progress [EPIC]",
				"shark get <key> --field status",
				"## shark status transitions",
			},
			forbids: []string{
				"Use `shark progress` for project,\nepic, or feature dashboards",
				"- `shark status` - Display status dashboard",
				"shark status [EPIC] [FEATURE] [flags]",
				"shark status E05",
				"shark status options",
				"shark progress [EPIC] [FEATURE]",
				"shark progress E05 F02",
			},
		},
		{
			path: filepath.Join("docs", "cli-reference", "progress-analytics.md"),
			contains: []string{
				"shark progress [EPIC] [flags]",
				"shark get E05-F02 --field status",
			},
			forbids: []string{
				"shark progress [EPIC] [FEATURE]",
				"shark progress E05 F02",
				"shark progress E05-F02",
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
				"`shark progress [EPIC] [FEATURE]`",
			},
		},
		{
			path: filepath.Join("skills", "shark-rider", "verbs", "query.md"),
			contains: []string{
				"shark progress",
				"shark get E01-F02",
			},
			forbids: []string{
				"shark status E01",
				"shark status E01-F02",
			},
		},
		{
			path: filepath.Join("skills", "shark-rider", "context", "workflow-and-status.md"),
			contains: []string{
				"shark progress",
				"shark get E01-F02",
			},
			forbids: []string{
				"shark status            # project-wide dashboard",
				"shark status E01        # epic status",
				"shark status E01-F02    # feature status",
			},
		},
		{
			path:     filepath.Join("skills", "shark-rider", "SKILL.md"),
			contains: []string{"shark progress [epic]"},
			forbids:  []string{"shark status [key]"},
		},
		{
			path:     filepath.Join("skills", "shark-rider", "verbs", "help.md"),
			contains: []string{"Read:             shark progress [epic]"},
			forbids:  []string{"Read:             shark status [key]"},
		},
		{
			path:     filepath.Join("skills", "shark-rider", "verbs", "viewer.md"),
			contains: []string{"shark progress"},
			forbids:  []string{"suggest `shark status`"},
		},
		{
			path:     filepath.Join("skills", "shark-rider", "skills", "triage", "SKILL.md"),
			contains: []string{"`shark progress` to understand current shape"},
			forbids:  []string{"`shark status` to understand current shape"},
		},
		{
			path:     filepath.Join("docs", "architectural-overview.md"),
			contains: []string{"shark progress E07"},
			forbids:  []string{"shark status E07\n"},
		},
		{
			path:     filepath.Join("docs", "guides", "observability.md"),
			contains: []string{"shark progress"},
			forbids: []string{
				"shark status E07-F01",
				"./bin/shark status",
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
