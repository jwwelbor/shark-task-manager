package entitydoc

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"
	"path/filepath"
	"sync"
	"testing"
)

var (
	b078CancelUnlinkContextMu sync.Mutex //nolint:gochecknoglobals // Test-only SQLite callback state for deterministic unlink cancellation.
	b078CancelUnlinkContext   context.CancelFunc
	b078RegisterCancelFunc    sync.Once
	b078RegisterCancelErr     error
)

func TestEntityDocumentRepoUnlinkRejectsIgnoredDeleteAndPreservesSiblings(t *testing.T) {
	assertB078RetainedUnlinkFailsAndRetryPreservesSiblings(t, "b078_retain_ignored_delete", 99991,
		"CREATE TRIGGER %s BEFORE DELETE ON entity_documents WHEN OLD.entity_type = 'task' AND OLD.entity_id = %d AND OLD.document_id = %d BEGIN SELECT RAISE(IGNORE); END")
}

func TestEntityDocumentRepoUnlinkRejectsRestoredDeleteAndPreservesSiblings(t *testing.T) {
	assertB078RetainedUnlinkFailsAndRetryPreservesSiblings(t, "b078_retain_restored_delete", 99981,
		"CREATE TRIGGER %s AFTER DELETE ON entity_documents WHEN OLD.entity_type = 'task' AND OLD.entity_id = %d AND OLD.document_id = %d BEGIN INSERT INTO entity_documents (entity_type, entity_id, document_id, link_type, created_at) VALUES (OLD.entity_type, OLD.entity_id, OLD.document_id, OLD.link_type, OLD.created_at); END")
}

func assertB078RetainedUnlinkFailsAndRetryPreservesSiblings(t *testing.T, triggerName string, entityID int64, triggerSQL string) {
	t.Helper()
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)
	otherEntityID := entityID - 10
	var documentIDs []int64
	t.Cleanup(func() {
		requireB078ExecNoError(t, database, "DROP TRIGGER IF EXISTS "+triggerName)
		requireB078ExecNoError(t, database, "DELETE FROM entity_documents WHERE entity_id IN (?, ?)", entityID, otherEntityID)
		for _, documentID := range documentIDs {
			requireB078ExecNoError(t, database, "DELETE FROM documents WHERE id = ?", documentID)
		}
	})
	cleanupEntityDocuments(t, ctx, db, entityID)
	cleanupEntityDocuments(t, ctx, db, otherEntityID)
	targetDocID := createTestDocument(t, ctx, db, "B078-retained", "docs/b078-retained.md")
	documentIDs = append(documentIDs, targetDocID)
	siblingDocID := createTestDocument(t, ctx, db, "B078-sibling", "docs/b078-sibling.md")
	documentIDs = append(documentIDs, siblingDocID)
	require.NoError(t, repo.Link(ctx, models.EntityTypeTask, entityID, targetDocID, "target"))
	require.NoError(t, repo.Link(ctx, models.EntityTypeBug, entityID, targetDocID, "other-type"))
	require.NoError(t, repo.Link(ctx, models.EntityTypeTask, otherEntityID, targetDocID, "other-parent"))
	require.NoError(t, repo.Link(ctx, models.EntityTypeTask, entityID, siblingDocID, "other-document"))

	before := snapshotEntityDocumentAssociations(t, database, targetDocID, siblingDocID)
	_, err := database.ExecContext(ctx, fmt.Sprintf(triggerSQL, triggerName, entityID, targetDocID))
	require.NoError(t, err)
	require.ErrorContains(t, repo.Unlink(ctx, models.EntityTypeTask, entityID, targetDocID), "association remains after delete")
	requireB078ExecNoError(t, database, "DROP TRIGGER "+triggerName)
	require.Equal(t, before, snapshotEntityDocumentAssociations(t, database, targetDocID, siblingDocID))

	require.NoError(t, repo.Unlink(ctx, models.EntityTypeTask, entityID, targetDocID))
	after := snapshotEntityDocumentAssociations(t, database, targetDocID, siblingDocID)
	require.Len(t, after, 3)
	require.Equal(t, b078AssociationsWithoutTarget(before, entityID, targetDocID), after)
	// A direct repository retry still verifies absence and remains idempotent.
	require.NoError(t, repo.Unlink(ctx, models.EntityTypeTask, entityID, targetDocID))
}

func b078AssociationsWithoutTarget(associations []associationSnapshot, entityID, documentID int64) []associationSnapshot {
	var survivors []associationSnapshot
	for _, association := range associations {
		if association.entityType == string(models.EntityTypeTask) && association.entityID == entityID && association.documentID == documentID {
			continue
		}
		survivors = append(survivors, association)
	}
	return survivors
}

func TestEntityDocumentRepoUnlinkValidatesBeforeOpeningTransaction(t *testing.T) {
	database, err := db.InitDB(filepath.Join(t.TempDir(), "b078-validation.db"))
	require.NoError(t, err, "initialize isolated database")
	require.NoError(t, database.Close(), "close isolated database")
	repo := NewEntityDocumentRepository(dbconn.NewDB(database))

	for _, testCase := range []struct {
		name       string
		entityType models.EntityType
		entityID   int64
		wantError  string
	}{
		{name: "type before identifier", entityType: "project", entityID: 0, wantError: `invalid entity type: "project"`},
		{name: "zero identifier", entityType: models.EntityTypeTask, entityID: 0, wantError: "entity_id must be positive, got 0"},
		{name: "negative identifier", entityType: models.EntityTypeTask, entityID: -1, wantError: "entity_id must be positive, got -1"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := repo.Unlink(context.Background(), testCase.entityType, testCase.entityID, 563)
			require.EqualError(t, err, testCase.wantError)
		})
	}
	require.ErrorContains(t, repo.Unlink(context.Background(), models.EntityTypeTask, 452, 563), "association removal transaction")
}

func TestEntityDocumentRepoUnlinkHonorsCancelledCallerContext(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	db := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(db)
	const entityID int64 = 99980
	cleanupEntityDocuments(t, context.Background(), db, entityID)
	var docID int64
	t.Cleanup(func() {
		requireB078ExecNoError(t, database, "DELETE FROM entity_documents WHERE entity_id = ?", entityID)
		if docID != 0 {
			requireB078ExecNoError(t, database, "DELETE FROM documents WHERE id = ?", docID)
		}
	})
	docID = createTestDocument(t, context.Background(), db, "B078 cancelled unlink", "docs/b078-cancelled-unlink.md")
	require.NoError(t, repo.Link(context.Background(), models.EntityTypeTask, entityID, docID, "specification"))
	before := snapshotEntityDocumentAssociations(t, database, docID, docID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := repo.Unlink(ctx, models.EntityTypeTask, entityID, docID)
	require.ErrorIs(t, err, context.Canceled)
	got := snapshotEntityDocumentAssociations(t, database, docID, docID)
	require.Equal(t, before, got)
	require.NoError(t, repo.Unlink(context.Background(), models.EntityTypeTask, entityID, docID), "live retry")
	require.Empty(t, snapshotEntityDocumentAssociations(t, database, docID, docID))
}

func TestEntityDocumentRepoUnlinkCancellationAfterDeleteRollsBackBeforeVerification(t *testing.T) {
	registerB078CancelUnlinkFunction(t)
	database, err := db.InitDB(filepath.Join(t.TempDir(), "b078-cancel-after-delete.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	ctx, cancel := context.WithCancel(context.Background())
	b078CancelUnlinkContextMu.Lock()
	b078CancelUnlinkContext = cancel
	b078CancelUnlinkContextMu.Unlock()
	t.Cleanup(func() {
		b078CancelUnlinkContextMu.Lock()
		b078CancelUnlinkContext = nil
		b078CancelUnlinkContextMu.Unlock()
		cancel()
	})

	databaseConn := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(databaseConn)
	const entityID int64 = 4582
	const triggerName = "b078_cancel_after_delete"
	docID := createTestDocument(t, context.Background(), databaseConn, "B078 cancel after delete", "docs/b078-cancel-after-delete.md")
	t.Cleanup(func() {
		requireB078ExecNoError(t, database, "DROP TRIGGER IF EXISTS "+triggerName)
		requireB078ExecNoError(t, database, "DELETE FROM entity_documents WHERE entity_type = 'task' AND entity_id = ? AND document_id = ?", entityID, docID)
		requireB078ExecNoError(t, database, "DELETE FROM documents WHERE id = ?", docID)
	})
	require.NoError(t, repo.Link(context.Background(), models.EntityTypeTask, entityID, docID, "specification"))
	before := snapshotEntityDocumentAssociations(t, database, docID, docID)
	_, err = database.ExecContext(context.Background(), `CREATE TRIGGER b078_cancel_after_delete
		AFTER DELETE ON entity_documents
		WHEN OLD.entity_type = 'task' AND OLD.entity_id = 4582
		BEGIN SELECT b078_cancel_unlink_context(); END`)
	require.NoError(t, err)

	err = repo.Unlink(ctx, models.EntityTypeTask, entityID, docID)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "verify entity association removal")
	require.ErrorIs(t, ctx.Err(), context.Canceled, "fixture must cancel the context after DELETE")
	require.Equal(t, before, snapshotEntityDocumentAssociations(t, database, docID, docID))

	_, err = database.ExecContext(context.Background(), "DROP TRIGGER "+triggerName)
	require.NoError(t, err)
	require.NoError(t, repo.Unlink(context.Background(), models.EntityTypeTask, entityID, docID), "live retry after cancellation")
	require.Empty(t, snapshotEntityDocumentAssociations(t, database, docID, docID))
}

func TestEntityDocumentRepoUnlinkCommitFailureRollsBackAssociation(t *testing.T) {
	database, err := db.InitDB(filepath.Join(t.TempDir(), "b078-commit-failure.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	ctx := context.Background()
	databaseConn := dbconn.NewDB(database)
	repo := NewEntityDocumentRepository(databaseConn)
	const entityID int64 = 4583
	const triggerName = "b078_fail_unlink_commit"
	docID := createTestDocument(t, ctx, databaseConn, "B078 commit failure", "docs/b078-commit-failure.md")
	t.Cleanup(func() {
		requireB078ExecNoError(t, database, "DROP TRIGGER IF EXISTS "+triggerName)
		requireB078ExecNoError(t, database, "DELETE FROM entity_documents WHERE entity_type = 'task' AND entity_id = ? AND document_id = ?", entityID, docID)
		requireB078ExecNoError(t, database, "DELETE FROM documents WHERE id = ?", docID)
		requireB078ExecNoError(t, database, "DROP TABLE IF EXISTS b078_commit_violation")
		requireB078ExecNoError(t, database, "DROP TABLE IF EXISTS b078_commit_parent")
	})
	require.NoError(t, repo.Link(ctx, models.EntityTypeTask, entityID, docID, "specification"))
	before := snapshotEntityDocumentAssociations(t, database, docID, docID)
	_, err = database.ExecContext(ctx, "CREATE TABLE b078_commit_parent (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `CREATE TABLE b078_commit_violation (
		parent_id INTEGER NOT NULL REFERENCES b078_commit_parent(id) DEFERRABLE INITIALLY DEFERRED
	)`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `CREATE TRIGGER b078_fail_unlink_commit
		AFTER DELETE ON entity_documents
		WHEN OLD.entity_type = 'task' AND OLD.entity_id = 4583
		BEGIN INSERT INTO b078_commit_violation(parent_id) VALUES (999); END`)
	require.NoError(t, err)

	err = repo.Unlink(ctx, models.EntityTypeTask, entityID, docID)
	require.ErrorContains(t, err, "commit association removal transaction")
	require.ErrorContains(t, err, "FOREIGN KEY constraint failed")
	require.Equal(t, before, snapshotEntityDocumentAssociations(t, database, docID, docID))
	var violations int
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM b078_commit_violation").Scan(&violations))
	require.Zero(t, violations)

	_, err = database.ExecContext(ctx, "DROP TRIGGER "+triggerName)
	require.NoError(t, err)
	require.NoError(t, repo.Unlink(ctx, models.EntityTypeTask, entityID, docID))
	require.Empty(t, snapshotEntityDocumentAssociations(t, database, docID, docID))
}

func registerB078CancelUnlinkFunction(t *testing.T) {
	t.Helper()
	b078RegisterCancelFunc.Do(func() {
		b078RegisterCancelErr = sqlite.RegisterScalarFunction("b078_cancel_unlink_context", 0,
			func(_ *sqlite.FunctionContext, _ []driver.Value) (driver.Value, error) {
				b078CancelUnlinkContextMu.Lock()
				cancel := b078CancelUnlinkContext
				b078CancelUnlinkContextMu.Unlock()
				if cancel != nil {
					cancel()
				}
				return int64(0), nil
			})
	})
	require.NoError(t, b078RegisterCancelErr)
}

type associationSnapshot struct {
	entityType string
	entityID   int64
	documentID int64
	linkType   string
	createdAt  string
}

func snapshotEntityDocumentAssociations(t *testing.T, database *sql.DB, documentIDs ...int64) []associationSnapshot {
	t.Helper()
	rows, err := database.QueryContext(context.Background(), `
		SELECT entity_type, entity_id, document_id, link_type, created_at
		FROM entity_documents WHERE document_id IN (?, ?) ORDER BY entity_type, entity_id, document_id`, documentIDs[0], documentIDs[1])
	require.NoError(t, err)
	defer rows.Close()
	var got []associationSnapshot
	for rows.Next() {
		var item associationSnapshot
		require.NoError(t, rows.Scan(&item.entityType, &item.entityID, &item.documentID, &item.linkType, &item.createdAt))
		got = append(got, item)
	}
	require.NoError(t, rows.Err())
	return got
}

func requireB078ExecNoError(t *testing.T, database *sql.DB, statement string, args ...any) {
	t.Helper()
	_, err := database.ExecContext(context.Background(), statement, args...)
	require.NoError(t, err)
}
