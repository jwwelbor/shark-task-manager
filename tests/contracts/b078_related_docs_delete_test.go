package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestB078RelatedDocsDeleteReturnsUnlinkFailureForExistingTaskLink(t *testing.T) {
	cli, sqlDB, targetTask, otherTask := newB078RelatedDocsDeleteFixture(t)
	const title = "S010 current continuation"
	seedB078DuplicateTitleTaskLinks(t, sqlDB, targetTask, otherTask, title)
	installB078UnlinkFailureTrigger(t, sqlDB)
	requireB078UnlinkFailure(t, cli, title, targetTask, otherTask)
	dropB078UnlinkFailureTrigger(t, sqlDB)
	requireB078SuccessfulUnlink(t, cli, title, targetTask, otherTask)
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
	// Insert the unrelated same-title document first: legacy title-only lookup
	// must select this ID, which is not linked to the task being unlinked.
	otherDoc, err := documentRepo.CreateOrGet(context.Background(), title, "docs/continuation/other.md")
	require.NoError(t, err)
	targetDoc, err := documentRepo.CreateOrGet(context.Background(), title, "docs/continuation/target.md")
	require.NoError(t, err)
	legacyMatch, err := documentRepo.GetByTitle(context.Background(), title)
	require.NoError(t, err)
	require.Equal(t, otherDoc.ID, legacyMatch.ID)
	documentLinks := repository.NewEntityDocumentRepository(repoDB)
	ctx := context.Background()
	require.NoError(t, documentLinks.Link(ctx, models.EntityTypeTask, targetTask.ID, targetDoc.ID, "general"))
	require.NoError(t, documentLinks.Link(ctx, models.EntityTypeTask, otherTask.ID, otherDoc.ID, "general"))
}

func installB078UnlinkFailureTrigger(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	_, err := sqlDB.Exec(`
		CREATE TRIGGER reject_related_doc_unlink
		BEFORE DELETE ON entity_documents
		BEGIN SELECT RAISE(ABORT, 'fixture unlink failure'); END`)
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

func (c b078CLI) success(t *testing.T, args ...string) []byte {
	t.Helper()
	output, err := c.run(args...)
	require.NoError(t, err, "%v: %s", args, output)
	return output
}

func (c b078CLI) rejection(t *testing.T, args ...string) string {
	t.Helper()
	output, err := c.run(args...)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "%v returned success: %s", args, output)
	if len(output) == 0 {
		require.NotEmpty(t, exitErr.Stderr)
		return string(exitErr.Stderr)
	}
	var failure struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(output, &failure), "%s\n%s", output, exitErr.Stderr)
	require.True(t, failure.Error, "%s", output)
	return failure.Message
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
