package entityhistory

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// helper: cleanup entity_history rows for given entity_ids
func cleanupEntityHistory(t *testing.T, ctx context.Context, db *dbconn.DB, entityIDs ...int64) {
	t.Helper()
	for _, id := range entityIDs {
		_, err := db.ExecContext(ctx, "DELETE FROM entity_history WHERE entity_id = ?", id)
		if err != nil {
			t.Fatalf("failed to cleanup entity_history for entity_id %d: %v", id, err)
		}
	}
}

// helper: create a valid EntityHistory for testing
func newTestEntityHistory(entityType models.EntityType, entityID int64, fromStatus *string, toStatus string, changedAt time.Time) *models.EntityHistory {
	return &models.EntityHistory{
		EntityType: entityType,
		EntityID:   entityID,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		ChangedAt:  changedAt,
	}
}

// helper: string pointer for entity history tests
func ehStrPtr(s string) *string {
	return &s
}

func TestEntityHistoryRepo_Create(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	const testEntityID int64 = 99901

	// Clean up before test
	cleanupEntityHistory(t, ctx, db, testEntityID)
	defer cleanupEntityHistory(t, ctx, db, testEntityID)

	entityTypes := []models.EntityType{
		models.EntityTypeEpic,
		models.EntityTypeFeature,
		models.EntityTypeTask,
		models.EntityTypeBug,
		models.EntityTypeChange,
	}

	now := time.Now().UTC().Truncate(time.Second)

	for i, et := range entityTypes {
		t.Run("create_"+string(et), func(t *testing.T) {
			h := &models.EntityHistory{
				EntityType:      et,
				EntityID:        testEntityID,
				FromStatus:      ehStrPtr("old_status"),
				ToStatus:        "new_status",
				ChangedBy:       ehStrPtr("agent-test"),
				Notes:           ehStrPtr("test note"),
				Forced:          i%2 == 0, // alternate true/false
				RejectionReason: ehStrPtr("test reason"),
				ChangedAt:       now.Add(time.Duration(i) * time.Second),
			}

			err := repo.Create(ctx, h)
			if err != nil {
				t.Fatalf("Create() returned error for entity type %s: %v", et, err)
			}

			if h.ID == 0 {
				t.Error("expected ID to be set after Create()")
			}

			// Verify record in DB
			var dbEntityType, dbToStatus string
			var dbID int64
			err = database.QueryRowContext(ctx,
				"SELECT id, entity_type, to_status FROM entity_history WHERE id = ?", h.ID,
			).Scan(&dbID, &dbEntityType, &dbToStatus)
			if err != nil {
				t.Fatalf("failed to query created record: %v", err)
			}
			if dbEntityType != string(et) {
				t.Errorf("expected entity_type %s, got %s", et, dbEntityType)
			}
			if dbToStatus != "new_status" {
				t.Errorf("expected to_status 'new_status', got %s", dbToStatus)
			}
		})
	}

	// Verify all 5 entity types are represented
	var count int
	err := database.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT entity_type) FROM entity_history WHERE entity_id = ?", testEntityID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count distinct entity types: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 distinct entity types, got %d", count)
	}
}

func TestEntityHistoryRepo_Create_Validation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	// Count before
	var countBefore int
	_ = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM entity_history").Scan(&countBefore)

	t.Run("empty_to_status", func(t *testing.T) {
		h := &models.EntityHistory{
			EntityType: models.EntityTypeTask,
			EntityID:   99902,
			ToStatus:   "", // invalid
			ChangedAt:  time.Now(),
		}

		err := repo.Create(ctx, h)
		if err == nil {
			t.Fatal("expected error for empty to_status, got nil")
		}
		if !strings.Contains(err.Error(), "validation failed") {
			t.Errorf("expected error to contain 'validation failed', got: %v", err)
		}
		if h.ID != 0 {
			t.Error("expected ID to remain 0 on validation failure")
		}
	})

	// Verify no rows inserted
	var countAfter int
	_ = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM entity_history").Scan(&countAfter)
	if countAfter != countBefore {
		t.Errorf("expected count unchanged (%d), got %d", countBefore, countAfter)
	}
}

func TestEntityHistoryRepo_Create_InvalidType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	h := &models.EntityHistory{
		EntityType: models.EntityType("invalid_type"),
		EntityID:   99903,
		ToStatus:   "some_status",
		ChangedAt:  time.Now(),
	}

	err := repo.Create(ctx, h)
	if err == nil {
		t.Fatal("expected error for invalid entity type, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected error to contain 'validation failed', got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid entity_type") {
		t.Errorf("expected error chain to contain 'invalid entity_type', got: %v", err)
	}
}

func TestEntityHistoryRepo_ListByEntity(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	const targetEntityID int64 = 99904
	const otherEntityID int64 = 99905
	const epicEntityID int64 = 99906

	cleanupEntityHistory(t, ctx, db, targetEntityID, otherEntityID, epicEntityID)
	defer cleanupEntityHistory(t, ctx, db, targetEntityID, otherEntityID, epicEntityID)

	baseTime := time.Now().UTC().Truncate(time.Second)

	// Insert 3 records for target (task, entity_id=99904)
	for i := 0; i < 3; i++ {
		h := newTestEntityHistory(models.EntityTypeTask, targetEntityID, ehStrPtr("status_a"), "status_b",
			baseTime.Add(time.Duration(i)*time.Second))
		if err := repo.Create(ctx, h); err != nil {
			t.Fatalf("failed to create test history: %v", err)
		}
	}

	// Insert 2 records for other (task, entity_id=99905)
	for i := 0; i < 2; i++ {
		h := newTestEntityHistory(models.EntityTypeTask, otherEntityID, ehStrPtr("status_x"), "status_y",
			baseTime.Add(time.Duration(i)*time.Second))
		if err := repo.Create(ctx, h); err != nil {
			t.Fatalf("failed to create test history: %v", err)
		}
	}

	// Insert 1 record for epic entity_id=99906
	h := newTestEntityHistory(models.EntityTypeEpic, epicEntityID, nil, "started",
		baseTime)
	if err := repo.Create(ctx, h); err != nil {
		t.Fatalf("failed to create test history: %v", err)
	}

	t.Run("returns_correct_records", func(t *testing.T) {
		results, err := repo.ListByEntity(ctx, models.EntityTypeTask, targetEntityID)
		if err != nil {
			t.Fatalf("ListByEntity() returned error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 records, got %d", len(results))
		}

		// Check all have correct entity_type and entity_id
		for _, r := range results {
			if r.EntityType != models.EntityTypeTask {
				t.Errorf("expected entity_type 'task', got '%s'", r.EntityType)
			}
			if r.EntityID != targetEntityID {
				t.Errorf("expected entity_id %d, got %d", targetEntityID, r.EntityID)
			}
		}

		// Check DESC ordering (most recent first)
		for i := 1; i < len(results); i++ {
			if results[i].ChangedAt.After(results[i-1].ChangedAt) {
				t.Errorf("expected DESC order: results[%d].ChangedAt (%v) should not be after results[%d].ChangedAt (%v)",
					i, results[i].ChangedAt, i-1, results[i-1].ChangedAt)
			}
		}

		// Check all fields populated
		r := results[0]
		if r.ID == 0 {
			t.Error("expected ID to be populated")
		}
		if r.FromStatus == nil {
			t.Error("expected FromStatus to be populated")
		}
		if r.ToStatus == "" {
			t.Error("expected ToStatus to be populated")
		}
	})

	t.Run("no_records_returns_empty_slice", func(t *testing.T) {
		results, err := repo.ListByEntity(ctx, models.EntityTypeFeature, 99999)
		if err != nil {
			t.Fatalf("ListByEntity() returned error: %v", err)
		}
		if results == nil {
			t.Error("expected non-nil empty slice, got nil")
		}
		if len(results) != 0 {
			t.Errorf("expected 0 records, got %d", len(results))
		}
	})
}

func TestEntityHistoryRepo_NullHandling(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	const testEntityID int64 = 99960

	cleanupEntityHistory(t, ctx, db, testEntityID)
	defer cleanupEntityHistory(t, ctx, db, testEntityID)

	// Create record with all nullable fields as nil
	h := &models.EntityHistory{
		EntityType:      models.EntityTypeTask,
		EntityID:        testEntityID,
		FromStatus:      nil,
		ToStatus:        "created",
		ChangedBy:       nil,
		Notes:           nil,
		Forced:          false,
		RejectionReason: nil,
		ChangedAt:       time.Now().UTC().Truncate(time.Second),
	}

	if err := repo.Create(ctx, h); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Retrieve and check NULL handling
	results, err := repo.ListByEntity(ctx, models.EntityTypeTask, testEntityID)
	if err != nil {
		t.Fatalf("ListByEntity() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}

	r := results[0]
	if r.FromStatus != nil {
		t.Errorf("expected FromStatus nil, got %v", r.FromStatus)
	}
	if r.ChangedBy != nil {
		t.Errorf("expected ChangedBy nil, got %v", r.ChangedBy)
	}
	if r.Notes != nil {
		t.Errorf("expected Notes nil, got %v", r.Notes)
	}
	if r.RejectionReason != nil {
		t.Errorf("expected RejectionReason nil, got %v", r.RejectionReason)
	}
	if r.Forced != false {
		t.Errorf("expected Forced false, got %v", r.Forced)
	}

	// Now create a record with Forced=true and verify
	h2 := &models.EntityHistory{
		EntityType: models.EntityTypeTask,
		EntityID:   testEntityID,
		ToStatus:   "forced_status",
		Forced:     true,
		ChangedAt:  time.Now().UTC().Truncate(time.Second).Add(time.Second),
	}

	if err := repo.Create(ctx, h2); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	results2, err := repo.ListByEntity(ctx, models.EntityTypeTask, testEntityID)
	if err != nil {
		t.Fatalf("ListByEntity() error: %v", err)
	}

	// Most recent first (forced=true)
	if !results2[0].Forced {
		t.Error("expected Forced=true for most recent record")
	}
	if results2[1].Forced {
		t.Error("expected Forced=false for older record")
	}
}
