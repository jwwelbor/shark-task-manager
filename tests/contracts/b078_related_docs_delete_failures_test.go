package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
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

func TestB078RelatedDocsDeleteVerifiesRetainedRowsAtCompiledCLI(t *testing.T) {
	for _, testCase := range b078RetainedRowTriggers() {
		t.Run(testCase.name, func(t *testing.T) {
			assertB078RetainedRowsAtCLI(t, testCase.trigger)
		})
	}
}

func b078RetainedRowTriggers() []struct {
	name    string
	trigger func(name string, entityID, documentID int64) string
} {
	return []struct {
		name    string
		trigger func(name string, entityID, documentID int64) string
	}{
		{name: "ignored delete", trigger: func(name string, entityID, documentID int64) string {
			return fmt.Sprintf("CREATE TRIGGER %s BEFORE DELETE ON entity_documents WHEN OLD.entity_type = 'task' AND OLD.entity_id = %d AND OLD.document_id = %d BEGIN SELECT RAISE(IGNORE); END", name, entityID, documentID)
		}},
		{name: "same-statement restoration", trigger: func(name string, entityID, documentID int64) string {
			return fmt.Sprintf("CREATE TRIGGER %s AFTER DELETE ON entity_documents WHEN OLD.entity_type = 'task' AND OLD.entity_id = %d AND OLD.document_id = %d BEGIN INSERT INTO entity_documents (entity_type, entity_id, document_id, link_type, created_at) VALUES (OLD.entity_type, OLD.entity_id, OLD.document_id, OLD.link_type, OLD.created_at); END", name, entityID, documentID)
		}},
	}
}

func assertB078RetainedRowsAtCLI(t *testing.T, trigger func(name string, entityID, documentID int64) string) {
	t.Helper()
	cli, sqlDB, targetTask, _ := newB078RelatedDocsDeleteFixture(t)
	ctx := context.Background()
	doc, err := repository.NewDocumentRepository(repository.NewDB(sqlDB)).CreateOrGet(ctx, "B078 retained row", "docs/b078/retained-row.md")
	require.NoError(t, err)
	require.NoError(t, repository.NewEntityDocumentRepository(repository.NewDB(sqlDB)).Link(ctx, models.EntityTypeTask, targetTask.ID, doc.ID, "specification"))
	const triggerName = "b078_compiled_retain"
	_, err = sqlDB.ExecContext(ctx, trigger(triggerName, targetTask.ID, doc.ID))
	require.NoError(t, err)
	t.Cleanup(func() { cleanupB078RetainedRowFixture(t, sqlDB, targetTask.ID, doc.ID, triggerName) })

	for _, jsonMode := range []bool{true, false} {
		mode := "human"
		if jsonMode {
			mode = "json"
		}
		t.Run(mode, func(t *testing.T) { assertB078RetainedDeleteFailure(t, cli, sqlDB, targetTask, doc, jsonMode) })
	}
	dropB078Trigger(t, sqlDB, triggerName)
	assertB078RetainedDeleteRetry(t, cli, sqlDB, targetTask, doc)
}

func cleanupB078RetainedRowFixture(t *testing.T, sqlDB *sql.DB, taskID, documentID int64, triggerName string) {
	t.Helper()
	_, err := sqlDB.ExecContext(context.Background(), "DROP TRIGGER IF EXISTS "+triggerName)
	assert.NoError(t, err, "drop retained-row trigger")
	_, err = sqlDB.ExecContext(context.Background(), "DELETE FROM entity_documents WHERE entity_type = 'task' AND entity_id = ? AND document_id = ?", taskID, documentID)
	assert.NoError(t, err, "remove retained-row association")
}

func assertB078RetainedDeleteFailure(t *testing.T, cli b078CLI, sqlDB *sql.DB, task *models.Task, doc *models.Document, jsonMode bool) {
	t.Helper()
	stdout, stderr, exitCode := cli.runWithDatabase(cli.dbPath, jsonMode, "related-docs", "delete", doc.Title, "--task="+task.Key)
	require.Equal(t, 2, exitCode)
	message := string(stderr)
	if jsonMode {
		var response struct {
			Error   bool   `json:"error"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal(stdout, &response))
		require.True(t, response.Error)
		require.Equal(t, "COMMAND_ERROR", response.Code)
		message = response.Message
	} else {
		require.Empty(t, stdout)
	}
	require.Contains(t, message, "failed to unlink document from task "+task.Key)
	require.Contains(t, message, "entity association remains after delete")
	require.NotContains(t, message, "unlinked")
	assertB078Link(t, sqlDB, models.EntityTypeTask, task.ID, doc.ID, true)
}

func assertB078RetainedDeleteRetry(t *testing.T, cli b078CLI, sqlDB *sql.DB, task *models.Task, doc *models.Document) {
	t.Helper()
	var response struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(cli.success(t, "related-docs", "delete", doc.Title, "--task="+task.Key), &response))
	require.Equal(t, "unlinked", response.Status)
	assertB078Link(t, sqlDB, models.EntityTypeTask, task.ID, doc.ID, false)
	var documentCount int
	require.NoError(t, sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM documents WHERE id = ?", doc.ID).Scan(&documentCount))
	require.Equal(t, 1, documentCount)
}

func TestB078RelatedDocsDeleteListScanFailurePreservesLink(t *testing.T) {
	cli, sqlDB, targetTask, _ := newB078RelatedDocsDeleteFixture(t)
	ctx := context.Background()
	repoDB := repository.NewDB(sqlDB)
	doc, err := repository.NewDocumentRepository(repoDB).CreateOrGet(ctx, "B078 unreadable task association", "docs/b078/unreadable-association.md")
	require.NoError(t, err)
	require.NoError(t, repository.NewEntityDocumentRepository(repoDB).Link(ctx, models.EntityTypeTask, targetTask.ID, doc.ID, "specification"))
	_, err = sqlDB.ExecContext(ctx, "UPDATE entity_documents SET created_at = 'not-a-timestamp' WHERE entity_type = 'task' AND entity_id = ? AND document_id = ?", targetTask.ID, doc.ID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := sqlDB.ExecContext(context.Background(), "DELETE FROM entity_documents WHERE entity_type = 'task' AND entity_id = ? AND document_id = ?", targetTask.ID, doc.ID)
		assert.NoError(t, cleanupErr, "remove unreadable association")
	})

	for _, jsonMode := range []bool{true, false} {
		mode := "human"
		if jsonMode {
			mode = "json"
		}
		t.Run(mode, func(t *testing.T) {
			stdout, stderr, exitCode := cli.runWithDatabase(cli.dbPath, jsonMode, "related-docs", "delete", doc.Title, "--task="+targetTask.Key)
			require.Equal(t, 2, exitCode)
			message := string(stderr)
			if jsonMode {
				var failure struct {
					Error   bool   `json:"error"`
					Code    string `json:"code"`
					Message string `json:"message"`
				}
				require.NoError(t, json.Unmarshal(stdout, &failure))
				require.True(t, failure.Error)
				require.Equal(t, "COMMAND_ERROR", failure.Code)
				message = failure.Message
			} else {
				require.Empty(t, stdout)
			}
			require.Contains(t, message, "failed to unlink document from task "+targetTask.Key)
			require.Contains(t, message, "scan")
			require.NotContains(t, message, "not found")
			require.NotContains(t, message, "unlinked")
			assertB078Link(t, sqlDB, models.EntityTypeTask, targetTask.ID, doc.ID, true)
		})
	}
}

func TestB078RelatedDocsDeleteMapsTaskLookupNoRowsToExitOne(t *testing.T) {
	cli, _, _, _ := newB078RelatedDocsDeleteFixture(t)
	message, exitCode := cli.rejectionWithExit(t, "related-docs", "delete", "Missing task document", "--task=T-E10-F02-999")
	require.Equal(t, 1, exitCode)
	require.Contains(t, message, "failed to unlink document from task T-E10-F02-999")
	require.Contains(t, message, "no rows")
	require.NotContains(t, message, "unlinked")
}

func TestB078RelatedDocsDeleteKeepsQuestionKeyInUnlinkErrors(t *testing.T) {
	cli, _, _, _ := newB078RelatedDocsDeleteFixture(t)
	message := cli.rejection(t, "related-docs", "delete", "Question spec", "--question=Q404")
	require.Contains(t, message, "unlink document from question Q404:")
}

func TestB078RelatedDocsDeleteMapsMissingQuestionToExitOne(t *testing.T) {
	cli, _, _, _ := newB078RelatedDocsDeleteFixture(t)
	message, exitCode := cli.rejectionWithExit(t, "related-docs", "delete", "Question spec", "--question=Q404")
	require.Equal(t, 1, exitCode)
	require.Contains(t, message, "unlink document from question Q404")
	require.Contains(t, strings.ToLower(message), "not found")
	require.NotContains(t, message, "unlinked")
}
