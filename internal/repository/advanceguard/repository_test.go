package advanceguard

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

const testEntityType = "epic"

func cleanupConsumptions(t *testing.T, ctx context.Context, db *dbconn.DB, entityID int64) {
	t.Helper()
	if _, err := db.ExecContext(ctx, "DELETE FROM advance_guard_consumptions WHERE entity_id = ?", entityID); err != nil {
		t.Fatalf("failed to cleanup advance_guard_consumptions for entity_id %d: %v", entityID, err)
	}
}

func TestAdvanceGuardRepository_RecordConsumed_ThenWasConsumed(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewRepository(db)

	const entityID int64 = 88801
	cleanupConsumptions(t, ctx, db, entityID)
	t.Cleanup(func() { cleanupConsumptions(t, ctx, db, entityID) })

	consumed, err := repo.WasConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass")
	if err != nil {
		t.Fatalf("WasConsumed() error = %v", err)
	}
	if consumed {
		t.Fatal("expected not consumed before RecordConsumed")
	}

	if err := repo.RecordConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass"); err != nil {
		t.Fatalf("RecordConsumed() error = %v", err)
	}

	consumed, err = repo.WasConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass")
	if err != nil {
		t.Fatalf("WasConsumed() error = %v", err)
	}
	if !consumed {
		t.Fatal("expected consumed after RecordConsumed")
	}
}

func TestAdvanceGuardRepository_RecordConsumed_DuplicateRejected(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewRepository(db)

	const entityID int64 = 88802
	cleanupConsumptions(t, ctx, db, entityID)
	t.Cleanup(func() { cleanupConsumptions(t, ctx, db, entityID) })

	if err := repo.RecordConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass"); err != nil {
		t.Fatalf("first RecordConsumed() error = %v", err)
	}

	err := repo.RecordConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass")
	if !errors.Is(err, ErrAlreadyConsumed) {
		t.Fatalf("expected ErrAlreadyConsumed on duplicate tuple, got %v", err)
	}
}

func TestAdvanceGuardRepository_RecordConsumedWithTx_DuplicateRejected(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewRepository(db)

	const entityID int64 = 88805
	cleanupConsumptions(t, ctx, db, entityID)
	t.Cleanup(func() { cleanupConsumptions(t, ctx, db, entityID) })

	firstTx, err := db.BeginTx()
	if err != nil {
		t.Fatalf("begin first transaction: %v", err)
	}
	if err := repo.RecordConsumedWithTx(ctx, firstTx, testEntityType, entityID, "sess-1", "draft", "pass"); err != nil {
		t.Fatalf("first RecordConsumedWithTx() error = %v", err)
	}
	if err := firstTx.Commit(); err != nil {
		t.Fatalf("commit first transaction: %v", err)
	}

	secondTx, err := db.BeginTx()
	if err != nil {
		t.Fatalf("begin duplicate transaction: %v", err)
	}
	defer secondTx.Rollback()
	err = repo.RecordConsumedWithTx(ctx, secondTx, testEntityType, entityID, "sess-1", "draft", "pass")
	if !errors.Is(err, ErrAlreadyConsumed) {
		t.Fatalf("expected ErrAlreadyConsumed on transactional duplicate tuple, got %v", err)
	}
}

// TestAdvanceGuardRepository_DeleteConsumed_CompensatesRecord proves the
// compensating-delete path used by EntityService.compensateAdvanceGuard:
// after a CAS status update fails following a successful RecordConsumed, the
// ledger row must be removable so a later legitimate replay of the same
// tuple isn't falsely blocked.
func TestAdvanceGuardRepository_DeleteConsumed_CompensatesRecord(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewRepository(db)

	const entityID int64 = 88803
	cleanupConsumptions(t, ctx, db, entityID)
	t.Cleanup(func() { cleanupConsumptions(t, ctx, db, entityID) })

	if err := repo.RecordConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass"); err != nil {
		t.Fatalf("RecordConsumed() error = %v", err)
	}

	if err := repo.DeleteConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass"); err != nil {
		t.Fatalf("DeleteConsumed() error = %v", err)
	}

	consumed, err := repo.WasConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass")
	if err != nil {
		t.Fatalf("WasConsumed() error = %v", err)
	}
	if consumed {
		t.Fatal("expected consumption to be gone after DeleteConsumed")
	}

	// A retry of the exact same tuple must now succeed rather than being
	// rejected as a duplicate.
	if err := repo.RecordConsumed(ctx, testEntityType, entityID, "sess-1", "draft", "pass"); err != nil {
		t.Fatalf("expected retry after DeleteConsumed to succeed, got error: %v", err)
	}
}

func TestAdvanceGuardRepository_DeleteConsumed_NoMatchIsNoop(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewRepository(db)

	const entityID int64 = 88804
	cleanupConsumptions(t, ctx, db, entityID)

	if err := repo.DeleteConsumed(ctx, testEntityType, entityID, "sess-never-recorded", "draft", "pass"); err != nil {
		t.Fatalf("DeleteConsumed() on a non-existent tuple should not error, got: %v", err)
	}
}
