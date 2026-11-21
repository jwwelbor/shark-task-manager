package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/bug"
	"github.com/stretchr/testify/require"
)

func TestB078RelatedDocsDeletePreservesWWGMConsumerAssociationsAndFiles(t *testing.T) {
	fixture := seedB078ConsumerScenario(t)
	before := readB078RelatedDocuments(t, fixture.cli, "--task="+fixture.targetTask.Key)
	assertB078DocumentPresent(t, before, fixture.documents[1].ID, fixture.malformedPath, true)

	var response struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Parent string `json:"parent"`
	}
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", fixture.currentTitle, "--task="+fixture.targetTask.Key), &response))
	require.Equal(t, "unlinked", response.Status)
	require.Equal(t, fixture.currentTitle, response.Title)
	require.Equal(t, "task", response.Parent)

	assertB078ConsumerAssociations(t, fixture)
	assertB078ConsumerDocuments(t, fixture)
}

type b078ConsumerScenario struct {
	cli           b078CLI
	sqlDB         *sql.DB
	targetTask    *models.Task
	otherTask     *models.Task
	bug           *models.Bug
	currentTitle  string
	malformedPath string
	documents     []*models.Document
	rootBytes     []byte
}

func seedB078ConsumerScenario(t *testing.T) b078ConsumerScenario {
	t.Helper()
	cli, sqlDB, targetTask, _ := newB078RelatedDocsDeleteFixture(t)
	ctx := context.Background()
	repoDB := repository.NewDB(sqlDB)
	var featureID int64
	require.NoError(t, sqlDB.QueryRowContext(ctx, "SELECT id FROM features WHERE key = 'E10-F02'").Scan(&featureID))
	otherTask := createB078RelatedDocsTask(t, repoDB, featureID, "T-E10-F02-011", "T011 workaround holder")
	setB078TaskIDs(t, sqlDB, targetTask, otherTask)
	bugRecord := seedB078ConsumerBug(t, sqlDB, repoDB)

	const malformedPath = "docs/plan/E10-runtime-combat-turn/E10-F02-persistent-package-backed-text-session-and-restart/tasks/continue_prompt.md"
	rootBytes := []byte("WWGM shared root continuation fixture\n")
	require.NoError(t, os.WriteFile(filepath.Join(cli.dir, "continue_prompt.md"), rootBytes, 0600))
	documents := seedB078ConsumerDocuments(t, sqlDB, cli, malformedPath)
	seedB078ConsumerLinks(t, sqlDB, repoDB, targetTask, otherTask, bugRecord, documents)
	return b078ConsumerScenario{cli: cli, sqlDB: sqlDB, targetTask: targetTask, otherTask: otherTask, bug: bugRecord,
		currentTitle: "S010 current continuation", malformedPath: malformedPath, documents: documents, rootBytes: rootBytes}
}

func setB078TaskIDs(t *testing.T, sqlDB *sql.DB, targetTask, otherTask *models.Task) {
	t.Helper()
	for _, entity := range []struct {
		key string
		id  int64
	}{{key: targetTask.Key, id: 452}, {key: otherTask.Key, id: 450}} {
		_, err := sqlDB.ExecContext(context.Background(), "UPDATE tasks SET id = ? WHERE key = ?", entity.id, entity.key)
		require.NoError(t, err)
	}
	targetTask.ID = 452
	otherTask.ID = 450
}

func seedB078ConsumerBug(t *testing.T, sqlDB *sql.DB, repoDB *repository.DB) *models.Bug {
	t.Helper()
	bugRecord := &models.Bug{
		BaseEntity: models.BaseEntity{Key: "B107", Title: "B107 workaround holder"},
		Status:     models.BugStatus("identified"), Severity: models.BugSeverityMedium,
	}
	require.NoError(t, bug.NewBugRepository(repoDB).Create(context.Background(), bugRecord))
	_, err := sqlDB.ExecContext(context.Background(), "UPDATE bugs SET id = 107 WHERE key = ?", bugRecord.Key)
	require.NoError(t, err)
	bugRecord.ID = 107
	return bugRecord
}

func seedB078ConsumerDocuments(t *testing.T, sqlDB *sql.DB, cli b078CLI, malformedPath string) []*models.Document {
	t.Helper()
	documents := []*models.Document{
		{ID: 542, Title: "S010 current continuation", FilePath: "continue_prompt.md"},
		{ID: 563, Title: "S010 current continuation", FilePath: malformedPath},
		{ID: 564, Title: "S010 root continuation", FilePath: "continue_prompt.md"},
	}
	for _, doc := range documents {
		_, err := sqlDB.ExecContext(context.Background(), "INSERT INTO documents (id, title, file_path) VALUES (?, ?, ?)", doc.ID, doc.Title, doc.FilePath)
		require.NoError(t, err)
	}
	require.NoFileExists(t, filepath.Join(cli.dir, malformedPath))
	return documents
}

func seedB078ConsumerLinks(t *testing.T, sqlDB *sql.DB, repoDB *repository.DB, targetTask, otherTask *models.Task, bugRecord *models.Bug, docs []*models.Document) {
	t.Helper()
	links := repository.NewEntityDocumentRepository(repoDB)
	for _, association := range []struct {
		entityType models.EntityType
		entityID   int64
		documentID int64
		linkType   string
	}{
		{models.EntityTypeBug, bugRecord.ID, docs[0].ID, "specification"},
		{models.EntityTypeTask, targetTask.ID, docs[1].ID, "general"},
		{models.EntityTypeTask, targetTask.ID, docs[2].ID, "reference"},
		{models.EntityTypeTask, otherTask.ID, docs[2].ID, "reference"},
	} {
		require.NoError(t, links.Link(context.Background(), association.entityType, association.entityID, association.documentID, association.linkType))
	}
}

func assertB078ConsumerAssociations(t *testing.T, fixture b078ConsumerScenario) {
	t.Helper()
	bugDoc, malformedDoc, rootDoc := fixture.documents[0], fixture.documents[1], fixture.documents[2]
	assertB078Link(t, fixture.sqlDB, models.EntityTypeBug, fixture.bug.ID, bugDoc.ID, true)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.targetTask.ID, malformedDoc.ID, false)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.targetTask.ID, rootDoc.ID, true)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.otherTask.ID, rootDoc.ID, true)
	assertB078LinkMetadata(t, fixture.sqlDB, models.EntityTypeBug, fixture.bug.ID, bugDoc.ID, "specification")
	assertB078LinkMetadata(t, fixture.sqlDB, models.EntityTypeTask, fixture.targetTask.ID, rootDoc.ID, "reference")
	assertB078LinkMetadata(t, fixture.sqlDB, models.EntityTypeTask, fixture.otherTask.ID, rootDoc.ID, "reference")
}

func assertB078ConsumerDocuments(t *testing.T, fixture b078ConsumerScenario) {
	t.Helper()
	bugDoc, malformedDoc, rootDoc := fixture.documents[0], fixture.documents[1], fixture.documents[2]
	remaining := readB078RelatedDocuments(t, fixture.cli, "--task="+fixture.targetTask.Key)
	assertB078DocumentPresent(t, remaining, malformedDoc.ID, fixture.malformedPath, false)
	assertB078DocumentPresent(t, remaining, rootDoc.ID, "continue_prompt.md", true)
	assertB078DocumentPresent(t, readB078RelatedDocuments(t, fixture.cli, "--bug=B107"), bugDoc.ID, "continue_prompt.md", true)
	assertB078DocumentPresent(t, readB078RelatedDocuments(t, fixture.cli, "--task="+fixture.otherTask.Key), rootDoc.ID, "continue_prompt.md", true)
	assertB078DocumentRowAndFile(t, fixture.sqlDB, fixture.cli.dir, bugDoc, fixture.rootBytes)
	assertB078DocumentRowAndFile(t, fixture.sqlDB, fixture.cli.dir, rootDoc, fixture.rootBytes)
	var retainedPath string
	require.NoError(t, fixture.sqlDB.QueryRowContext(context.Background(), "SELECT file_path FROM documents WHERE id = ?", malformedDoc.ID).Scan(&retainedPath))
	require.Equal(t, fixture.malformedPath, retainedPath)
	require.NoFileExists(t, filepath.Join(fixture.cli.dir, retainedPath))
}
