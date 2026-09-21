package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestB071NotesDocumentationMatchesSupportedCommandSurface prevents the
// generic notes documentation from promising the unsupported `notes <key>`
// listing form. Entity-scoped listing remains available through commands such
// as `shark task notes <key>`.
func TestB071NotesDocumentationMatchesSupportedCommandSurface(t *testing.T) {
	commandNames := make(map[string]bool)
	for _, command := range notesCmd.Commands() {
		commandNames[command.Name()] = true
	}
	for _, want := range []string{"add", "search"} {
		if !commandNames[want] {
			t.Errorf("generic notes command is missing supported %q subcommand", want)
		}
	}
	if commandNames["list"] {
		t.Fatal("generic notes command must not advertise an unsupported list subcommand")
	}

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
				`shark notes add E07-F01-001 --type decision "Chose JWT over sessions"`,
				`shark notes search "authentication"`,
				"shark task notes E07-F01-001",
			},
			forbids: []string{
				"shark notes E07-F01-001",
				"shark notes list",
				"shark note list",
				"shark notes {ENTITY_KEY}",
			},
		},
		{
			path: filepath.Join(".claude", "rules", "cli", "commands.md"),
			contains: []string{
				"`shark notes search <query>`",
				"`shark task notes <task-key>`",
			},
			forbids: []string{
				"`shark notes <key>` — View entity notes",
				"shark notes list",
				"shark note list",
				"shark notes {ENTITY_KEY}",
			},
		},
		{
			path: filepath.Join("docs", "cli-reference", "README.md"),
			contains: []string{
				"`shark notes add <key> --type <type> <content>`",
				"`shark notes search <query>`",
			},
			forbids: []string{
				"`shark notes <key>` | View entity notes",
				"shark notes list",
				"shark note list",
				"shark notes {ENTITY_KEY}",
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
					t.Errorf("%s still documents unsupported notes form %q", document.path, forbidden)
				}
			}
		})
	}
}
