package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newB078RelatedDocsFixture(t *testing.T) (b078CLI, *sql.DB, *repository.DB) {
	t.Helper()
	cli, sqlDB, _, _ := newB078RelatedDocsDeleteFixture(t)
	return cli, sqlDB, repository.NewDB(sqlDB)
}

func stringPointer(value string) *string { return &value }

func createB078TechDebt(t *testing.T, repo *techdebt.TechDebtRepository, key, slug string) *models.TechDebt {
	t.Helper()
	td := &models.TechDebt{
		BaseEntity: models.BaseEntity{Key: key, Title: "B078 " + key, Slug: stringPointer(slug)},
		Status:     models.TechDebtStatus("identified"),
		Category:   models.TechDebtCategoryCodeQuality,
		Severity:   models.TechDebtSeverityMedium,
	}
	require.NoError(t, repo.Create(context.Background(), td))
	return td
}

func createB078ChangeCard(t *testing.T, database *sql.DB, key, slug string) *models.ChangeCard {
	t.Helper()
	_, err := database.Exec(`INSERT INTO change_cards (key, title, status, priority, slug) VALUES (?, ?, ?, ?, ?)`,
		key, "B078 "+key, "proposed", 1, slug)
	require.NoError(t, err)
	card := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key}}
	require.NoError(t, database.QueryRow(`SELECT id FROM change_cards WHERE key = ?`, key).Scan(&card.ID))
	return card
}

func assertB078Link(t *testing.T, database *sql.DB, entityType models.EntityType, entityID, documentID int64, want bool) {
	t.Helper()
	var count int
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?`,
		entityType, entityID, documentID).Scan(&count))
	if want {
		require.Equal(t, 1, count)
		return
	}
	require.Zero(t, count)
}

func assertB078LinkMetadata(t *testing.T, database *sql.DB, entityType models.EntityType, entityID, documentID int64, wantLinkType string) {
	t.Helper()
	var linkType string
	require.NoError(t, database.QueryRow(`SELECT link_type FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?`, entityType, entityID, documentID).Scan(&linkType))
	require.Equal(t, wantLinkType, linkType)
}

func readB078RelatedDocuments(t *testing.T, cli b078CLI, selector string) []*models.Document {
	t.Helper()
	var docs []*models.Document
	require.NoError(t, json.Unmarshal(cli.success(t, "related-docs", "list", selector), &docs))
	return docs
}

func assertB078DocumentPresent(t *testing.T, docs []*models.Document, documentID int64, path string, wantPresent bool) {
	t.Helper()
	for _, doc := range docs {
		if doc.ID == documentID {
			require.True(t, wantPresent, "document %d unexpectedly listed", documentID)
			require.Equal(t, path, doc.FilePath)
			return
		}
	}
	require.False(t, wantPresent, "document %d at %s was not listed", documentID, path)
}

func assertB078DocumentRowAndFile(t *testing.T, database *sql.DB, projectDir string, expected *models.Document, expectedBytes []byte) {
	t.Helper()
	var title, path string
	require.NoError(t, database.QueryRow("SELECT title, file_path FROM documents WHERE id = ?", expected.ID).Scan(&title, &path))
	require.Equal(t, expected.Title, title)
	require.Equal(t, expected.FilePath, path)
	actualBytes, err := os.ReadFile(filepath.Join(projectDir, path))
	require.NoError(t, err)
	require.Equal(t, expectedBytes, actualBytes)
}

func newB078RelatedDocsDeleteFixture(t *testing.T) (b078CLI, *sql.DB, *models.Task, *models.Task) {
	t.Helper()
	projectDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".sharkconfig.json"), []byte("{}"), 0600))
	dbPath := filepath.Join(projectDir, "shark-tasks.db")
	sqlDB, err := db.InitDB(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	repoDB := repository.NewDB(sqlDB)
	ctx := context.Background()
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E10", Title: "B078 related docs test"},
		Status:     models.EpicStatusActive,
		Priority:   models.PriorityMedium,
	}
	require.NoError(t, repository.NewEpicRepository(repoDB).Create(ctx, epic))
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{Key: "E10-F02", Title: "B078 related docs test feature"},
		EpicID:     epic.ID,
		Status:     models.FeatureStatusDraft,
	}
	require.NoError(t, repository.NewFeatureRepository(repoDB).Create(ctx, feature))
	targetTask := createB078RelatedDocsTask(t, repoDB, feature.ID, "T-E10-F02-013", "Task with continuation link")
	otherTask := createB078RelatedDocsTask(t, repoDB, feature.ID, "T-E10-F02-014", "Task with another continuation link")
	return b078CLI{binary: buildSharkTC008(t), dir: projectDir, dbPath: dbPath}, sqlDB, targetTask, otherTask
}

func createB078RelatedDocsTask(t *testing.T, repoDB *repository.DB, featureID int64, key, title string) *models.Task {
	t.Helper()
	task := &models.Task{
		BaseEntity: models.BaseEntity{Key: key, Title: title},
		FeatureID:  featureID,
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	require.NoError(t, repository.NewTaskRepository(repoDB).Create(context.Background(), task))
	return task
}

func seedB078DuplicateTitleTaskLinks(t *testing.T, sqlDB *sql.DB, targetTask, otherTask *models.Task, title string) {
	t.Helper()
	repoDB := repository.NewDB(sqlDB)
	documentRepo := repository.NewDocumentRepository(repoDB)
	// Insert duplicate titles, then link whichever title-only row the legacy
	// lookup actually selects only to the unrelated task. Query order is not
	// defined, so the fixture must use the observed match rather than insertion
	// order to preserve the counterexample deterministically.
	otherDoc, err := documentRepo.CreateOrGet(context.Background(), title, "docs/continuation/other.md")
	require.NoError(t, err)
	targetDoc, err := documentRepo.CreateOrGet(context.Background(), title, "docs/continuation/target.md")
	require.NoError(t, err)
	legacyMatch, err := documentRepo.GetByTitle(context.Background(), title)
	require.NoError(t, err)
	require.Contains(t, []int64{otherDoc.ID, targetDoc.ID}, legacyMatch.ID)
	targetDocID := otherDoc.ID
	if legacyMatch.ID == otherDoc.ID {
		targetDocID = targetDoc.ID
	}
	documentLinks := repository.NewEntityDocumentRepository(repoDB)
	ctx := context.Background()
	require.NoError(t, documentLinks.Link(ctx, models.EntityTypeTask, targetTask.ID, targetDocID, "general"))
	require.NoError(t, documentLinks.Link(ctx, models.EntityTypeTask, otherTask.ID, legacyMatch.ID, "general"))
}

func installB078UnlinkFailureTrigger(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	_, err := sqlDB.Exec(`
		CREATE TRIGGER reject_related_doc_unlink
		BEFORE DELETE ON entity_documents
		BEGIN SELECT RAISE(ABORT, 'fixture unlink failure'); END`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := sqlDB.Exec("DROP TRIGGER IF EXISTS reject_related_doc_unlink")
		assert.NoError(t, cleanupErr, "drop fixture unlink trigger")
	})
}

func dropB078Trigger(t *testing.T, sqlDB *sql.DB, name string) {
	t.Helper()
	_, err := sqlDB.Exec("DROP TRIGGER IF EXISTS " + name)
	require.NoError(t, err)
}

func requireB078UnlinkFailure(t *testing.T, cli b078CLI, title string, targetTask, otherTask *models.Task) {
	t.Helper()
	message := cli.rejection(t, "related-docs", "delete", title, "--task="+targetTask.Key)
	require.Contains(t, message, "failed to unlink document from task "+targetTask.Key)
	require.Equal(t, 1, strings.Count(message, "failed to unlink document from task "+targetTask.Key))
	require.Contains(t, message, "fixture unlink failure")
	require.Equal(t, []string{"docs/continuation/target.md"}, cli.reportPaths(t, targetTask.Key))
	require.Equal(t, []string{"docs/continuation/other.md"}, cli.reportPaths(t, otherTask.Key))
}

func dropB078UnlinkFailureTrigger(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	_, err := sqlDB.Exec("DROP TRIGGER reject_related_doc_unlink")
	require.NoError(t, err)
}

func requireB078SuccessfulUnlink(t *testing.T, cli b078CLI, title string, targetTask, otherTask *models.Task) {
	t.Helper()
	var response struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Parent string `json:"parent"`
	}
	require.NoError(t, json.Unmarshal(cli.success(t, "related-docs", "delete", title, "--task="+targetTask.Key), &response))
	require.Equal(t, "unlinked", response.Status)
	require.Equal(t, title, response.Title)
	require.Equal(t, "task", response.Parent)
	require.Empty(t, cli.reportPaths(t, targetTask.Key))
	require.Equal(t, []string{"docs/continuation/other.md"}, cli.reportPaths(t, otherTask.Key))
}

type b078CLI struct {
	binary string
	dir    string
	dbPath string
}

func (c b078CLI) run(args ...string) ([]byte, error) {
	command := exec.Command(c.binary, append([]string{"--db", c.dbPath, "--json"}, args...)...)
	command.Dir = c.dir
	return command.Output()
}

func (c b078CLI) runWithDatabase(databasePath string, jsonMode bool, args ...string) ([]byte, []byte, int) {
	commandArgs := []string{"--db", databasePath}
	if jsonMode {
		commandArgs = append(commandArgs, "--json")
	}
	commandArgs = append(commandArgs, args...)
	command := exec.Command(c.binary, commandArgs...)
	command.Dir = c.dir
	stdout, err := command.Output()
	if err == nil {
		return stdout, nil, 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return stdout, []byte(err.Error()), -1
	}
	return stdout, exitErr.Stderr, exitErr.ExitCode()
}

func (c b078CLI) success(t *testing.T, args ...string) []byte {
	t.Helper()
	output, err := c.run(args...)
	require.NoError(t, err, "%v: %s", args, output)
	return output
}

func (c b078CLI) rejection(t *testing.T, args ...string) string {
	t.Helper()
	message, _ := c.rejectionWithExit(t, args...)
	return message
}

func (c b078CLI) rejectionWithExit(t *testing.T, args ...string) (string, int) {
	t.Helper()
	output, err := c.run(args...)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "%v returned success: %s", args, output)
	if len(output) == 0 {
		require.NotEmpty(t, exitErr.Stderr)
		return string(exitErr.Stderr), exitErr.ExitCode()
	}
	var failure struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(output, &failure), "%s\n%s", output, exitErr.Stderr)
	require.True(t, failure.Error, "%s", output)
	return failure.Message, exitErr.ExitCode()
}

func (c b078CLI) reportPaths(t *testing.T, key string) []string {
	t.Helper()
	var documents []models.Document
	require.NoError(t, json.Unmarshal(c.success(t, "related-docs", "list", "--task="+key), &documents))
	paths := make([]string, 0, len(documents))
	for _, document := range documents {
		paths = append(paths, document.FilePath)
	}
	return paths
}

func (c b078CLI) relatedDocIDs(t *testing.T, selector string) []int64 {
	t.Helper()
	return relatedB078DocumentIDs(t, c.success(t, "related-docs", "list", selector))
}

func relatedB078DocumentIDs(t *testing.T, output []byte) []int64 {
	t.Helper()
	var documents []models.Document
	require.NoError(t, json.Unmarshal(output, &documents))
	ids := make([]int64, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document.ID)
	}
	return ids
}

func (c b078CLI) relatedDocIDsPositional(t *testing.T, key string) []int64 {
	t.Helper()
	return relatedB078DocumentIDs(t, c.success(t, "related-docs", "list", key))
}
