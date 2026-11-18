package contracts

import (
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

type researchCLI struct {
	binary string
	dir    string
	dbPath string
}

func (c researchCLI) run(args ...string) ([]byte, error) {
	cmd := exec.Command(c.binary, append([]string{"--db", c.dbPath, "--json"}, args...)...)
	cmd.Dir = c.dir
	return cmd.Output()
}

func (c researchCLI) success(t *testing.T, args ...string) []byte {
	t.Helper()
	out, err := c.run(args...)
	require.NoError(t, err, "%v: %s", args, out)
	return out
}

func (c researchCLI) rejection(t *testing.T, args ...string) string {
	t.Helper()
	out, err := c.run(args...)
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, "%v returned success: %s", args, out)
	// The list command has a local --json flag and prints failures to stderr.
	if len(out) == 0 {
		require.NotEmpty(t, exit.Stderr)
		return string(exit.Stderr)
	}
	var failure struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(out, &failure), "%s\n%s", out, exit.Stderr)
	require.True(t, failure.Error, "%s", out)
	return failure.Message
}

func (c researchCLI) reportPaths(t *testing.T, key string) []string {
	t.Helper()
	var docs []models.Document
	require.NoError(t, json.Unmarshal(c.success(t, "related-docs", "list", key), &docs))
	var paths []string
	for _, doc := range docs {
		paths = append(paths, doc.FilePath)
	}
	return paths
}

func TestB079ResearchCLI(t *testing.T) {
	binary := buildSharkTC008(t)
	project := func(t *testing.T) researchCLI {
		t.Helper()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte("{}"), 0600))
		return researchCLI{binary: binary, dir: dir, dbPath: filepath.Join(dir, "fixture.db")}
	}

	for _, mutation := range [][]string{{"add", "Report", "docs/report.md"}, {"delete", "Report"}} {
		for _, selectors := range []struct {
			name string
			args []string
		}{
			{"missing", nil},
			{"conflicting", []string{"--bug=B001", "--tech-debt=TD-001"}},
			{"empty_with_valid", []string{"--bug=", "--tech-debt=TD-001"}},
		} {
			t.Run(mutation[0]+"_"+selectors.name+"_selector", func(t *testing.T) {
				c := project(t)
				args := append([]string{"related-docs"}, mutation...)
				message := c.rejection(t, append(args, selectors.args...)...)
				require.Contains(t, message, "one of")
				require.NoFileExists(t, c.dbPath, "invalid selectors must be rejected before opening a service")
			})
		}
	}

	const title = "Research Report"
	t.Run("unlink resolves the title on the selected item", func(t *testing.T) {
		c := project(t)
		c.success(t, "td", "create", "First item")
		c.success(t, "td", "create", "Second item")
		c.success(t, "related-docs", "add", title, "docs/first.md", "--tech-debt=TD-001")
		c.success(t, "related-docs", "add", title, "docs/second.md", "--tech-debt=TD-002")
		require.Equal(t, []string{"docs/second.md"}, c.reportPaths(t, "TD-002"))
		for i := 0; i < 2; i++ {
			c.success(t, "related-docs", "delete", title, "--tech-debt=TD-002")
			require.Empty(t, c.reportPaths(t, "TD-002"))
			require.Equal(t, []string{"docs/first.md"}, c.reportPaths(t, "TD-001"))
		}
	})

	t.Run("ambiguous title preserves both links", func(t *testing.T) {
		c := project(t)
		c.success(t, "td", "create", "Two reports")
		paths := []string{"docs/a.md", "docs/b.md"}
		for _, path := range paths {
			c.success(t, "related-docs", "add", title, path, "--tech-debt=TD-001")
		}
		require.Contains(t, c.rejection(t, "related-docs", "delete", title, "--tech-debt=TD-001"), "ambiguous")
		require.ElementsMatch(t, paths, c.reportPaths(t, "TD-001"))
	})

	t.Run("unlink and list surface storage failures", func(t *testing.T) {
		c := project(t)
		c.success(t, "td", "create", "Storage failures")
		c.success(t, "related-docs", "add", title, "docs/report.md", "--tech-debt=TD-001")
		db, err := sql.Open("sqlite", c.dbPath)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, db.Close()) })
		_, err = db.Exec("CREATE TRIGGER refuse_unlink BEFORE DELETE ON entity_documents BEGIN SELECT RAISE(ABORT, 'fixture unlink failure'); END")
		require.NoError(t, err)
		t.Run("write failure", func(t *testing.T) {
			msg := c.rejection(t, "related-docs", "delete", title, "--tech-debt=TD-001")
			require.Contains(t, msg, "fixture unlink failure")
			require.Contains(t, msg, "TD-001")
			require.Equal(t, []string{"docs/report.md"}, c.reportPaths(t, "TD-001"))
		})
		t.Run("missing item", func(t *testing.T) {
			require.Contains(t, c.rejection(t, "related-docs", "delete", title, "--tech-debt=TD-999"), "TD-999")
		})
		_, err = db.Exec("ALTER TABLE documents RENAME COLUMN title TO unavailable_title")
		require.NoError(t, err)
		t.Run("lookup failure", func(t *testing.T) {
			msg := c.rejection(t, "related-docs", "delete", title, "--tech-debt=TD-001")
			require.Contains(t, msg, "TD-001")
			require.Contains(t, msg, "column")
		})
		t.Run("list failure", func(t *testing.T) {
			msg := c.rejection(t, "related-docs", "list", "TD-001")
			require.Contains(t, msg, "list documents for tech-debt TD-001")
			require.Contains(t, msg, "column")
		})
	})

	t.Run("catalog follows the selected config", func(t *testing.T) {
		c := project(t)
		for _, fixture := range []struct {
			config, bundle, location, content, source string
		}{
			{".sharkconfig.json", "default", "research", "version: default\n", "disk"},
			{"alternate.json", "alternate", "overrides/research", "version: alternate\n", "override"},
		} {
			require.NoError(t, os.WriteFile(filepath.Join(c.dir, fixture.config), []byte(`{"shark_data_path":"`+fixture.bundle+`"}`), 0600))
			location := filepath.Join(c.dir, fixture.bundle, fixture.location)
			require.NoError(t, os.MkdirAll(location, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(location, "recipes.yaml"), []byte(fixture.content), 0600))
			out := c.success(t, "data", "get", "research/recipes.yaml", "--config", fixture.config)
			var got map[string]string
			require.NoError(t, json.Unmarshal(out, &got))
			require.Equal(t, fixture.content, got["content"])
			require.Equal(t, fixture.source, got["source"])
		}
	})
}
