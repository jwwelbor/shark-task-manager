package contracts

import (
	"context"
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

func TestB078RelatedDocsDeleteReturnsUnlinkFailureForExistingTaskLink(t *testing.T) {
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
	targetTask := &models.Task{
		BaseEntity: models.BaseEntity{Key: "T-E10-F02-013", Title: "Task with continuation link"},
		FeatureID:  feature.ID,
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	require.NoError(t, repository.NewTaskRepository(repoDB).Create(ctx, targetTask))
	otherTask := &models.Task{
		BaseEntity: models.BaseEntity{Key: "T-E10-F02-014", Title: "Task with another continuation link"},
		FeatureID:  feature.ID,
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	require.NoError(t, repository.NewTaskRepository(repoDB).Create(ctx, otherTask))

	const title = "S010 current continuation"
	documentRepo := repository.NewDocumentRepository(repoDB)
	targetDoc, err := documentRepo.CreateOrGet(ctx, title, "docs/continuation/target.md")
	require.NoError(t, err)
	otherDoc, err := documentRepo.CreateOrGet(ctx, title, "docs/continuation/other.md")
	require.NoError(t, err)
	documentLinks := repository.NewEntityDocumentRepository(repoDB)
	require.NoError(t, documentLinks.Link(ctx, models.EntityTypeTask, targetTask.ID, targetDoc.ID, "general"))
	require.NoError(t, documentLinks.Link(ctx, models.EntityTypeTask, otherTask.ID, otherDoc.ID, "general"))
	_, err = sqlDB.Exec(`
		CREATE TRIGGER reject_related_doc_unlink
		BEFORE DELETE ON entity_documents
		BEGIN SELECT RAISE(ABORT, 'fixture unlink failure'); END`)
	require.NoError(t, err)

	cli := b078CLI{
		binary: buildSharkTC008(t),
		dir:    projectDir,
		dbPath: dbPath,
	}
	message := cli.rejection(t, "related-docs", "delete", title, "--task="+targetTask.Key)
	require.Contains(t, message, "failed to unlink document from task "+targetTask.Key)
	require.Contains(t, message, "fixture unlink failure")
	require.Equal(t, []string{"docs/continuation/target.md"}, cli.reportPaths(t, targetTask.Key))
	require.Equal(t, []string{"docs/continuation/other.md"}, cli.reportPaths(t, otherTask.Key))

	_, err = sqlDB.Exec("DROP TRIGGER reject_related_doc_unlink")
	require.NoError(t, err)
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
