package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
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

func newB079ResearchCLI(t *testing.T) researchCLI {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte("{}"), 0600))
	return researchCLI{
		binary: buildSharkTC008(t),
		dir:    dir,
		dbPath: filepath.Join(dir, "fixture.db"),
	}
}

func TestB079ResearchCLIRejectsInvalidMutationSelectors(t *testing.T) {
	c := newB079ResearchCLI(t)
	for _, mutation := range [][]string{{"add", "Report", "docs/report.md"}, {"delete", "Report"}} {
		for _, selectors := range []struct {
			name string
			args []string
		}{
			{"missing", nil},
			{"conflicting", []string{"--bug=B001", "--tech-debt=TD-001"}},
			{"empty", []string{"--bug="}},
			{"empty_with_valid", []string{"--bug=", "--tech-debt=TD-001"}},
		} {
			t.Run(mutation[0]+"_"+selectors.name, func(t *testing.T) {
				args := append([]string{"related-docs"}, mutation...)
				message := c.rejection(t, append(args, selectors.args...)...)
				require.Contains(t, message, "one of")
				require.NoFileExists(t, c.dbPath, "invalid selectors must be rejected before opening a service")
			})
		}
	}
}

func TestB079ResearchCLIUnlinksTitleWithinSelectedEntity(t *testing.T) {
	c := newB079ResearchCLI(t)
	const title = "Research Report"
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
}

func TestB079ResearchCLIRejectsAmbiguousTitleWithoutRemovingLinks(t *testing.T) {
	c := newB079ResearchCLI(t)
	const title = "Research Report"
	c.success(t, "td", "create", "Two reports")
	paths := []string{"docs/a.md", "docs/b.md"}
	for _, path := range paths {
		c.success(t, "related-docs", "add", title, path, "--tech-debt=TD-001")
	}
	require.Contains(t, c.rejection(t, "related-docs", "delete", title, "--tech-debt=TD-001"), "ambiguous")
	require.ElementsMatch(t, paths, c.reportPaths(t, "TD-001"))
}

func TestB079ResearchCLIPropagatesStorageFailures(t *testing.T) {
	c := newB079ResearchCLI(t)
	const title = "Research Report"
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
}

func TestB079ResearchCLIUsesCatalogSelectedByConfig(t *testing.T) {
	c := newB079ResearchCLI(t)
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
}

func TestB079ResearchCLIResolvesNestedConfigBundleFromProjectRoot(t *testing.T) {
	c := newB079ResearchCLI(t)
	configPath := filepath.Join(c.dir, "configs", "profiles", "alternate.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0700))
	require.NoError(t, os.WriteFile(configPath, []byte(`{"shark_data_path":"nested-bundle"}`), 0600))
	location := filepath.Join(c.dir, "nested-bundle", "research")
	require.NoError(t, os.MkdirAll(location, 0700))
	const content = "version: project-root\n"
	require.NoError(t, os.WriteFile(filepath.Join(location, "recipes.yaml"), []byte(content), 0600))

	out := c.success(t, "data", "get", "research/recipes.yaml", "--config", filepath.Join("configs", "profiles", "alternate.json"))
	var got map[string]string
	require.NoError(t, json.Unmarshal(out, &got))
	require.Equal(t, content, got["content"])
	require.Equal(t, "disk", got["source"])
}

func TestB079ResearchCLIRejectsMalformedTechDebtSelectorBeforeService(t *testing.T) {
	c := newB079ResearchCLI(t)
	for _, selector := range []string{"TD-1", "TD-001-"} {
		t.Run(selector, func(t *testing.T) {
			message := c.rejection(t, "related-docs", "list", "--tech-debt="+selector)
			require.Contains(t, message, "invalid tech-debt key")
			require.NoFileExists(t, c.dbPath, "invalid key should be rejected before service lookup")
		})
	}
}

func TestB079ResearchCLIResolvesSluggedTechDebtSelector(t *testing.T) {
	c := newB079ResearchCLI(t)
	c.success(t, "td", "create", "Slug Lookup")
	const sluggedKey = "TD-001-slug-lookup"
	c.success(t, "related-docs", "add", "Slugged report", "docs/slugged.md", "--tech-debt="+sluggedKey)
	var documents []models.Document
	require.NoError(t, json.Unmarshal(c.success(t, "related-docs", "list", "--tech-debt="+sluggedKey), &documents))
	require.Len(t, documents, 1)
	require.Equal(t, "docs/slugged.md", documents[0].FilePath)
	c.success(t, "related-docs", "delete", "Slugged report", "--tech-debt="+sluggedKey)
	require.Empty(t, c.reportPaths(t, "TD-001"))
}

type b079RelatedDocsRoute struct {
	name, selector, key, linkedTo, entityType string
	entityID                                  int64
}

func TestB079RelatedDocsRoutesEverySupportedEntity(t *testing.T) {
	c := newB079ResearchCLI(t)
	routes := seedB079RelatedDocsEntities(t, c)
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			title := "B079 route " + route.name
			path := "docs/b079-route-" + route.name + ".md"
			documentID := assertB079RelatedDocsAddRoute(t, c, route, title, path)
			assertB079RelatedDocsListRoute(t, c, route, title, path)
			assertB079RelatedDocsDeleteRoute(t, c, route, title, documentID)
		})
	}
}

func TestB079RelatedDocsMissingEntityReturnsServiceError(t *testing.T) {
	c := newB079ResearchCLI(t)
	message := c.rejection(t, "related-docs", "add", "Missing bug document", "docs/missing-bug.md", "--bug=B999")
	require.Contains(t, message, "link document to bug B999")
}

func seedB079RelatedDocsEntities(t *testing.T, c researchCLI) []b079RelatedDocsRoute {
	t.Helper()
	rawDB, err := db.InitDB(c.dbPath)
	require.NoError(t, err)
	defer func() {
		if err := rawDB.Close(); err != nil {
			t.Errorf("close B079 related-docs fixture DB: %v", err)
		}
	}()
	repoDB := repository.NewDB(rawDB)
	routes := seedB079EpicFeatureTaskRoutes(t, repoDB)
	return append(routes, seedB079BugChangeQuestionRoutes(t, repoDB)...)
}

func seedB079EpicFeatureTaskRoutes(t *testing.T, repoDB *repository.DB) []b079RelatedDocsRoute {
	t.Helper()
	ctx := context.Background()
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E90", Title: "Related docs test epic"},
		Status:     models.EpicStatusActive,
		Priority:   models.PriorityMedium,
	}
	require.NoError(t, repository.NewEpicRepository(repoDB).Create(ctx, epic))
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{Key: "E90-F01", Title: "Related docs test feature"},
		EpicID:     epic.ID,
		Status:     models.FeatureStatusDraft,
	}
	require.NoError(t, repository.NewFeatureRepository(repoDB).Create(ctx, feature))
	task := &models.Task{
		BaseEntity: models.BaseEntity{Key: "T-E90-F01-001", Title: "Related docs test task"},
		FeatureID:  feature.ID,
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	require.NoError(t, repository.NewTaskRepository(repoDB).Create(ctx, task))
	return []b079RelatedDocsRoute{
		{name: "epic", selector: "epic", key: epic.Key, linkedTo: "epic", entityType: string(models.EntityTypeEpic), entityID: epic.ID},
		{name: "feature", selector: "feature", key: feature.Key, linkedTo: "feature", entityType: string(models.EntityTypeFeature), entityID: feature.ID},
		{name: "task", selector: "task", key: task.Key, linkedTo: "task", entityType: string(models.EntityTypeTask), entityID: task.ID},
	}
}

func seedB079BugChangeQuestionRoutes(t *testing.T, repoDB *repository.DB) []b079RelatedDocsRoute {
	t.Helper()
	ctx := context.Background()
	bug := &models.Bug{
		BaseEntity: models.BaseEntity{Key: "B901", Title: "Related docs test bug"},
		Status:     models.BugStatus("identified"),
		Severity:   models.BugSeverityMedium,
	}
	require.NoError(t, repository.NewBugRepository(repoDB).Create(ctx, bug))
	change := &models.ChangeCard{
		BaseEntity: models.BaseEntity{Key: "CC-901", Title: "Related docs test change"},
		Status:     models.ChangeCardStatus("proposed"),
		Priority:   1,
	}
	require.NoError(t, repository.NewChangeCardRepository(repoDB).Create(ctx, change))
	question := &models.Question{
		BaseEntity: models.BaseEntity{Key: "Q901", Title: "Related docs test question"},
		Status:     models.QuestionStatusDraft,
		Summary:    "Question to verify related-document routing",
		Requester:  "related-docs-test",
	}
	require.NoError(t, repository.NewQuestionRepository(repoDB).Create(ctx, question))
	return []b079RelatedDocsRoute{
		{name: "bug", selector: "bug", key: bug.Key, linkedTo: "bug", entityType: string(models.EntityTypeBug), entityID: bug.ID},
		{name: "change-card", selector: "change", key: change.Key, linkedTo: "change-card", entityType: string(models.EntityTypeChange), entityID: change.ID},
		{name: "question", selector: "question", key: question.Key, linkedTo: "question", entityType: string(models.EntityTypeQuestion), entityID: question.ID},
	}
}

func assertB079RelatedDocsAddRoute(t *testing.T, c researchCLI, route b079RelatedDocsRoute, title, path string) int64 {
	t.Helper()
	var response struct {
		Title     string `json:"title"`
		Path      string `json:"path"`
		LinkedTo  string `json:"linked_to"`
		ParentKey string `json:"parent_key"`
	}
	require.NoError(t, json.Unmarshal(c.success(t, "related-docs", "add", title, path, "--"+route.selector+"="+route.key), &response))
	require.Equal(t, title, response.Title)
	require.Equal(t, path, response.Path)
	require.Equal(t, route.linkedTo, response.LinkedTo)
	require.Equal(t, route.key, response.ParentKey)

	database, err := sql.Open("sqlite", c.dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	var entityType string
	var entityID, documentID int64
	require.NoError(t, database.QueryRow(`
		SELECT ed.entity_type, ed.entity_id, ed.document_id
		FROM entity_documents ed
		JOIN documents d ON d.id = ed.document_id
		WHERE d.title = ? AND d.file_path = ?`, title, path).
		Scan(&entityType, &entityID, &documentID))
	require.Equal(t, route.entityType, entityType)
	require.Equal(t, route.entityID, entityID)
	return documentID
}

func assertB079RelatedDocsListRoute(t *testing.T, c researchCLI, route b079RelatedDocsRoute, title, path string) {
	t.Helper()
	var documents []models.Document
	require.NoError(t, json.Unmarshal(c.success(t, "related-docs", "list", "--"+route.selector+"="+route.key), &documents))
	for _, document := range documents {
		if document.Title == title && document.FilePath == path {
			return
		}
	}
	t.Fatalf("related-docs list for %s %s omitted %q at %q: %#v", route.linkedTo, route.key, title, path, documents)
}

func assertB079RelatedDocsDeleteRoute(t *testing.T, c researchCLI, route b079RelatedDocsRoute, title string, documentID int64) {
	t.Helper()
	var response map[string]string
	require.NoError(t, json.Unmarshal(c.success(t, "related-docs", "delete", title, "--"+route.selector+"="+route.key), &response))
	require.Equal(t, "unlinked", response["status"])
	require.Equal(t, title, response["title"])
	require.Equal(t, route.linkedTo, response["parent"])

	database, err := sql.Open("sqlite", c.dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	var remaining int
	require.NoError(t, database.QueryRow(
		"SELECT COUNT(*) FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?",
		route.entityType, route.entityID, documentID).Scan(&remaining))
	require.Zero(t, remaining)
}
