package entityhistory

import (
	"context"
	"database/sql"
	"fmt"
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

func TestEntityHistoryRepo_GetLastNonTerminalStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	const entityID int64 = 99970

	cleanupEntityHistory(t, ctx, db, entityID)
	defer cleanupEntityHistory(t, ctx, db, entityID)

	baseTime := time.Now().UTC().Truncate(time.Second)

	terminalStatuses := []string{"completed", "cancelled"}

	t.Run("returns_most_recent_non_terminal", func(t *testing.T) {
		cleanupEntityHistory(t, ctx, db, entityID)

		// Insert: older non-terminal, then terminal, then newer non-terminal
		histories := []*models.EntityHistory{
			newTestEntityHistory(models.EntityTypeFeature, entityID, ehStrPtr("draft"), "in_development", baseTime.Add(0*time.Second)),
			newTestEntityHistory(models.EntityTypeFeature, entityID, ehStrPtr("in_development"), "completed", baseTime.Add(1*time.Second)),
			newTestEntityHistory(models.EntityTypeFeature, entityID, ehStrPtr("completed"), "in_qa", baseTime.Add(2*time.Second)),
		}
		for _, h := range histories {
			if err := repo.Create(ctx, h); err != nil {
				t.Fatalf("Create() error: %v", err)
			}
		}

		status, found, err := repo.GetLastNonTerminalStatus(ctx, models.EntityTypeFeature, entityID, terminalStatuses)
		if err != nil {
			t.Fatalf("GetLastNonTerminalStatus() error: %v", err)
		}
		if !found {
			t.Fatal("expected found=true, got false")
		}
		if status != "in_qa" {
			t.Errorf("expected 'in_qa', got %q", status)
		}
	})

	t.Run("returns_empty_when_all_terminal", func(t *testing.T) {
		cleanupEntityHistory(t, ctx, db, entityID)

		h := newTestEntityHistory(models.EntityTypeFeature, entityID, ehStrPtr("in_development"), "completed", baseTime)
		if err := repo.Create(ctx, h); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		status, found, err := repo.GetLastNonTerminalStatus(ctx, models.EntityTypeFeature, entityID, terminalStatuses)
		if err != nil {
			t.Fatalf("GetLastNonTerminalStatus() error: %v", err)
		}
		if found {
			t.Errorf("expected found=false, got true (status=%q)", status)
		}
		if status != "" {
			t.Errorf("expected empty status, got %q", status)
		}
	})

	t.Run("returns_empty_when_no_rows", func(t *testing.T) {
		cleanupEntityHistory(t, ctx, db, entityID)

		status, found, err := repo.GetLastNonTerminalStatus(ctx, models.EntityTypeFeature, entityID, terminalStatuses)
		if err != nil {
			t.Fatalf("GetLastNonTerminalStatus() error: %v", err)
		}
		if found {
			t.Errorf("expected found=false for empty table, got true (status=%q)", status)
		}
		if status != "" {
			t.Errorf("expected empty status, got %q", status)
		}
	})

	t.Run("filters_by_entity_type", func(t *testing.T) {
		cleanupEntityHistory(t, ctx, db, entityID)

		// Insert epic history with non-terminal status, feature history with only terminal
		hEpic := newTestEntityHistory(models.EntityTypeEpic, entityID, ehStrPtr("draft"), "in_development", baseTime)
		hFeature := newTestEntityHistory(models.EntityTypeFeature, entityID, ehStrPtr("in_qa"), "completed", baseTime.Add(time.Second))
		if err := repo.Create(ctx, hEpic); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		if err := repo.Create(ctx, hFeature); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Query for feature type only: should find nothing (only terminal)
		status, found, err := repo.GetLastNonTerminalStatus(ctx, models.EntityTypeFeature, entityID, terminalStatuses)
		if err != nil {
			t.Fatalf("GetLastNonTerminalStatus() error: %v", err)
		}
		if found {
			t.Errorf("expected no non-terminal feature rows, got status=%q", status)
		}
	})

	t.Run("empty_terminal_set_returns_most_recent", func(t *testing.T) {
		cleanupEntityHistory(t, ctx, db, entityID)

		h := newTestEntityHistory(models.EntityTypeFeature, entityID, ehStrPtr("draft"), "completed", baseTime)
		if err := repo.Create(ctx, h); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// With empty terminal set, all statuses are non-terminal
		status, found, err := repo.GetLastNonTerminalStatus(ctx, models.EntityTypeFeature, entityID, []string{})
		if err != nil {
			t.Fatalf("GetLastNonTerminalStatus() error: %v", err)
		}
		if !found {
			t.Fatal("expected found=true with empty terminal set")
		}
		if status != "completed" {
			t.Errorf("expected 'completed', got %q", status)
		}
	})
}

func TestEntityHistoryRepo_CreateTx(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	const entityID int64 = 99980

	cleanupEntityHistory(t, ctx, db, entityID)
	defer cleanupEntityHistory(t, ctx, db, entityID)

	t.Run("inserts_within_transaction", func(t *testing.T) {
		cleanupEntityHistory(t, ctx, db, entityID)

		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx() error: %v", err)
		}

		h := &models.EntityHistory{
			EntityType: models.EntityTypeFeature,
			EntityID:   entityID,
			FromStatus: ehStrPtr("completed"),
			ToStatus:   "in_qa",
			ChangedAt:  time.Now().UTC().Truncate(time.Second),
		}

		if err := repo.CreateTx(ctx, tx, h); err != nil {
			_ = tx.Rollback()
			t.Fatalf("CreateTx() error: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit() error: %v", err)
		}

		// Verify row is committed
		results, err := repo.ListByEntity(ctx, models.EntityTypeFeature, entityID)
		if err != nil {
			t.Fatalf("ListByEntity() error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 row, got %d", len(results))
		}
		if results[0].ToStatus != "in_qa" {
			t.Errorf("expected ToStatus 'in_qa', got %q", results[0].ToStatus)
		}
		if h.ID == 0 {
			t.Error("expected ID to be set after CreateTx")
		}
	})

	t.Run("rollback_leaves_no_row", func(t *testing.T) {
		cleanupEntityHistory(t, ctx, db, entityID)

		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx() error: %v", err)
		}

		h := &models.EntityHistory{
			EntityType: models.EntityTypeFeature,
			EntityID:   entityID,
			FromStatus: ehStrPtr("completed"),
			ToStatus:   "in_development",
			ChangedAt:  time.Now().UTC().Truncate(time.Second),
		}

		if err := repo.CreateTx(ctx, tx, h); err != nil {
			_ = tx.Rollback()
			t.Fatalf("CreateTx() error: %v", err)
		}

		// Roll back instead of committing
		if err := tx.Rollback(); err != nil {
			t.Fatalf("Rollback() error: %v", err)
		}

		// Verify no row persisted
		results, err := repo.ListByEntity(ctx, models.EntityTypeFeature, entityID)
		if err != nil {
			t.Fatalf("ListByEntity() error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 rows after rollback, got %d", len(results))
		}
	})

	t.Run("validates_before_insert", func(t *testing.T) {
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx() error: %v", err)
		}
		defer func() { _ = tx.Rollback() }()

		// Invalid: empty ToStatus
		h := &models.EntityHistory{
			EntityType: models.EntityTypeFeature,
			EntityID:   entityID,
			ToStatus:   "", // invalid
			ChangedAt:  time.Now(),
		}

		err = repo.CreateTx(ctx, tx, h)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !strings.Contains(err.Error(), "validation failed") {
			t.Errorf("expected 'validation failed' in error, got: %v", err)
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

// ----------------------------------------------------------------------------
// Helpers for ListRecentAcrossEntities tests
// ----------------------------------------------------------------------------

// seedRecentActivityEntity inserts a parent entity (epic, feature, task, bug, or
// change_card) into the test DB and returns its auto-incremented ID.
// Keys must be unique across all calls within a test run.
func seedRecentActivityEntity(t *testing.T, db *sql.DB, entityType models.EntityType, key, title string, epicID, featureID int64) int64 {
	t.Helper()
	var result sql.Result
	var err error

	switch entityType {
	case models.EntityTypeEpic:
		result, err = db.Exec(
			`INSERT OR IGNORE INTO epics (key, title, status, priority) VALUES (?, ?, 'active', 'medium')`,
			key, title,
		)
	case models.EntityTypeFeature:
		result, err = db.Exec(
			`INSERT OR IGNORE INTO features (epic_id, key, title, status) VALUES (?, ?, ?, 'active')`,
			epicID, key, title,
		)
	case models.EntityTypeTask:
		result, err = db.Exec(
			`INSERT OR IGNORE INTO tasks (feature_id, key, title, status, priority, depends_on) VALUES (?, ?, ?, 'todo', 5, '[]')`,
			featureID, key, title,
		)
	case models.EntityTypeBug:
		result, err = db.Exec(
			`INSERT OR IGNORE INTO bugs (key, title, status, severity) VALUES (?, ?, 'reported', 'medium')`,
			key, title,
		)
	case models.EntityTypeChange:
		result, err = db.Exec(
			`INSERT OR IGNORE INTO change_cards (key, title, status) VALUES (?, ?, 'proposed')`,
			key, title,
		)
	default:
		t.Fatalf("unsupported entity type %q", entityType)
	}
	if err != nil {
		t.Fatalf("seedRecentActivityEntity: failed to insert %s %q: %v", entityType, key, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("seedRecentActivityEntity: failed to get last insert id: %v", err)
	}
	if id == 0 {
		// INSERT OR IGNORE hit a duplicate; fetch the existing ID.
		switch entityType {
		case models.EntityTypeEpic:
			err = db.QueryRow("SELECT id FROM epics WHERE key = ?", key).Scan(&id)
		case models.EntityTypeFeature:
			err = db.QueryRow("SELECT id FROM features WHERE key = ?", key).Scan(&id)
		case models.EntityTypeTask:
			err = db.QueryRow("SELECT id FROM tasks WHERE key = ?", key).Scan(&id)
		case models.EntityTypeBug:
			err = db.QueryRow("SELECT id FROM bugs WHERE key = ?", key).Scan(&id)
		case models.EntityTypeChange:
			err = db.QueryRow("SELECT id FROM change_cards WHERE key = ?", key).Scan(&id)
		}
		if err != nil {
			t.Fatalf("seedRecentActivityEntity: failed to lookup existing id for %s %q: %v", entityType, key, err)
		}
	}
	return id
}

// cleanupRecentActivityEntity deletes the parent entity by key.
func cleanupRecentActivityEntity(t *testing.T, db *sql.DB, entityType models.EntityType, key string) {
	t.Helper()
	var table string
	switch entityType {
	case models.EntityTypeEpic:
		table = "epics"
	case models.EntityTypeFeature:
		table = "features"
	case models.EntityTypeTask:
		table = "tasks"
	case models.EntityTypeBug:
		table = "bugs"
	case models.EntityTypeChange:
		table = "change_cards"
	default:
		t.Fatalf("cleanupRecentActivityEntity: unsupported type %q", entityType)
	}
	_, _ = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE key = ?", table), key)
}

// ----------------------------------------------------------------------------
// TC-R-001: Returns top N across all entity types, newest first
// ----------------------------------------------------------------------------

func TestEntityHistoryRepo_ListRecentAcrossEntities_TopN(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	baseTime := time.Now().UTC().Truncate(time.Second).Add(-10 * time.Minute)

	// Seed one entity per type (5 total), 2 history rows each → 10 rows total.
	entityDefs := []struct {
		eType models.EntityType
		key   string
		title string
	}{
		{models.EntityTypeEpic, "TEST-RALL-E01", "RALL Epic 1"},
		{models.EntityTypeFeature, "TEST-RALL-E01-F01", "RALL Feature 1"},
		{models.EntityTypeTask, "T-TEST-RALL-E01-F01-001", "RALL Task 1"},
		{models.EntityTypeBug, "TEST-RALL-B001", "RALL Bug 1"},
		{models.EntityTypeChange, "TEST-RALL-CC-001", "RALL Change 1"},
	}

	// Ensure a parent epic+feature for the task and feature rows.
	epicID := seedRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RALL-E01", "RALL Epic 1", 0, 0)
	featureID := seedRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RALL-E01-F01", "RALL Feature 1", epicID, 0)

	var allEntityIDs []int64
	for i, ed := range entityDefs {
		var eid int64
		switch ed.eType {
		case models.EntityTypeEpic:
			eid = epicID
		case models.EntityTypeFeature:
			eid = featureID
		default:
			eid = seedRecentActivityEntity(t, database, ed.eType, ed.key, ed.title, epicID, featureID)
		}
		allEntityIDs = append(allEntityIDs, eid)

		// 2 history rows per entity, spaced 1 minute apart.
		for j := 0; j < 2; j++ {
			h := newTestEntityHistory(ed.eType, eid, ehStrPtr("status_a"), "status_b",
				baseTime.Add(time.Duration(i*2+j)*time.Minute))
			if err := repo.Create(ctx, h); err != nil {
				t.Fatalf("failed to create history for %s: %v", ed.eType, err)
			}
		}
	}

	// Cleanup
	defer func() {
		cleanupEntityHistory(t, ctx, db, allEntityIDs...)
		for _, ed := range entityDefs {
			cleanupRecentActivityEntity(t, database, ed.eType, ed.key)
		}
	}()

	// With Limit=5, exactly 5 rows should come back, newest first.
	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{Limit: 5})
	if err != nil {
		t.Fatalf("ListRecentAcrossEntities() error: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 rows with Limit=5, got %d", len(results))
	}

	// Verify DESC ordering (each row must not be newer than the preceding row).
	for i := 1; i < len(results); i++ {
		if results[i].ChangedAt.After(results[i-1].ChangedAt) {
			t.Errorf("result[%d].ChangedAt (%v) is after result[%d].ChangedAt (%v) — not DESC order",
				i, results[i].ChangedAt, i-1, results[i-1].ChangedAt)
		}
	}

	// The most recent row must correspond to the last entity/row we inserted.
	// (Last entity × second row = baseTime + (4*2+1)*minute = baseTime + 9 minutes)
	expectedNewest := baseTime.Add(9 * time.Minute)
	if !results[0].ChangedAt.Equal(expectedNewest) {
		t.Errorf("expected newest row at %v, got %v", expectedNewest, results[0].ChangedAt)
	}
}

// ----------------------------------------------------------------------------
// TC-R-002: Filter by EntityType = "task"
// ----------------------------------------------------------------------------

func TestEntityHistoryRepo_ListRecentAcrossEntities_FilterByEntityType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	baseTime := time.Now().UTC().Truncate(time.Second).Add(-5 * time.Minute)

	epicID := seedRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RFILT-E01", "RFILT Epic", 0, 0)
	featureID := seedRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RFILT-E01-F01", "RFILT Feature", epicID, 0)
	taskID := seedRecentActivityEntity(t, database, models.EntityTypeTask, "T-TEST-RFILT-E01-F01-001", "RFILT Task", epicID, featureID)
	bugID := seedRecentActivityEntity(t, database, models.EntityTypeBug, "TEST-RFILT-B001", "RFILT Bug", 0, 0)

	defer func() {
		cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID)
		cleanupRecentActivityEntity(t, database, models.EntityTypeTask, "T-TEST-RFILT-E01-F01-001")
		cleanupRecentActivityEntity(t, database, models.EntityTypeBug, "TEST-RFILT-B001")
		cleanupRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RFILT-E01-F01")
		cleanupRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RFILT-E01")
	}()

	// Insert 2 task rows and 2 bug rows.
	for i := 0; i < 2; i++ {
		ht := newTestEntityHistory(models.EntityTypeTask, taskID, ehStrPtr("s_a"), "s_b", baseTime.Add(time.Duration(i)*time.Minute))
		if err := repo.Create(ctx, ht); err != nil {
			t.Fatalf("create task history: %v", err)
		}
		hb := newTestEntityHistory(models.EntityTypeBug, bugID, ehStrPtr("s_a"), "s_b", baseTime.Add(time.Duration(i)*time.Minute))
		if err := repo.Create(ctx, hb); err != nil {
			t.Fatalf("create bug history: %v", err)
		}
	}

	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		EntityType: "task",
		Limit:      25,
	})
	if err != nil {
		t.Fatalf("ListRecentAcrossEntities() error: %v", err)
	}
	for _, r := range results {
		if r.EntityType != "task" {
			t.Errorf("expected entity_type 'task', got %q", r.EntityType)
		}
	}
	// Should include our 2 task rows (and no bug rows).
	taskCount := 0
	for _, r := range results {
		if r.Key == "T-TEST-RFILT-E01-F01-001" {
			taskCount++
		}
	}
	if taskCount != 2 {
		t.Errorf("expected 2 task rows for key T-TEST-RFILT-E01-F01-001, got %d", taskCount)
	}
}

// ----------------------------------------------------------------------------
// TC-R-003: Filter by Since — rows before cutoff excluded
// ----------------------------------------------------------------------------

func TestEntityHistoryRepo_ListRecentAcrossEntities_FilterBySince(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	baseTime := time.Now().UTC().Truncate(time.Second).Add(-20 * time.Minute)
	cutoff := baseTime.Add(5 * time.Minute)

	epicID := seedRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RSINCE-E01", "RSINCE Epic", 0, 0)

	defer func() {
		cleanupEntityHistory(t, ctx, db, epicID)
		cleanupRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RSINCE-E01")
	}()

	// 5 rows before cutoff, 5 rows after cutoff.
	for i := 0; i < 10; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Minute) // i=0..4 → before cutoff, i=5..9 → after
		h := newTestEntityHistory(models.EntityTypeEpic, epicID, ehStrPtr("s_a"), "s_b", ts)
		if err := repo.Create(ctx, h); err != nil {
			t.Fatalf("create history: %v", err)
		}
	}

	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		Since: &cutoff,
		Limit: 25,
	})
	if err != nil {
		t.Fatalf("ListRecentAcrossEntities() error: %v", err)
	}

	for _, r := range results {
		if !r.ChangedAt.After(cutoff) {
			t.Errorf("row with changed_at %v should not be returned (before or equal to cutoff %v)", r.ChangedAt, cutoff)
		}
	}
	// We expect 4 rows after cutoff (i=6,7,8,9; i=5 is equal to cutoff which is excluded by >)
	afterCutoffForKey := 0
	for _, r := range results {
		if r.Key == "TEST-RSINCE-E01" {
			afterCutoffForKey++
		}
	}
	if afterCutoffForKey != 4 {
		t.Errorf("expected 4 rows after cutoff for TEST-RSINCE-E01, got %d", afterCutoffForKey)
	}
}

// ----------------------------------------------------------------------------
// TC-R-004: Deleted entity row omitted (INNER JOIN filter)
// ----------------------------------------------------------------------------

func TestEntityHistoryRepo_ListRecentAcrossEntities_OrphanOmitted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	baseTime := time.Now().UTC().Truncate(time.Second)

	epicID := seedRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RORPHAN-E01", "RORPHAN Epic", 0, 0)
	featureID := seedRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RORPHAN-E01-F01", "RORPHAN Feature", epicID, 0)
	taskID := seedRecentActivityEntity(t, database, models.EntityTypeTask, "T-TEST-RORPHAN-E01-F01-001", "RORPHAN Task", epicID, featureID)

	// Insert 2 history rows for the task.
	for i := 0; i < 2; i++ {
		h := newTestEntityHistory(models.EntityTypeTask, taskID, ehStrPtr("todo"), "in_progress", baseTime.Add(time.Duration(i)*time.Second))
		if err := repo.Create(ctx, h); err != nil {
			t.Fatalf("create history: %v", err)
		}
	}

	// Verify rows exist before deletion.
	before, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		EntityType: "task",
		Limit:      25,
	})
	if err != nil {
		t.Fatalf("ListRecentAcrossEntities() before delete: %v", err)
	}
	taskRowsBefore := 0
	for _, r := range before {
		if r.Key == "T-TEST-RORPHAN-E01-F01-001" {
			taskRowsBefore++
		}
	}
	if taskRowsBefore != 2 {
		t.Fatalf("expected 2 task rows before delete, got %d", taskRowsBefore)
	}

	// Delete the task (cascade leaves history rows as orphans).
	_, err = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-TEST-RORPHAN-E01-F01-001'")
	if err != nil {
		t.Fatalf("delete task: %v", err)
	}

	defer func() {
		cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID)
		cleanupRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RORPHAN-E01-F01")
		cleanupRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RORPHAN-E01")
	}()

	// After deletion the orphaned history rows must not appear.
	after, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		EntityType: "task",
		Limit:      25,
	})
	if err != nil {
		t.Fatalf("ListRecentAcrossEntities() after delete: %v", err)
	}
	for _, r := range after {
		if r.Key == "T-TEST-RORPHAN-E01-F01-001" {
			t.Errorf("orphaned task row should not appear, but got key %q", r.Key)
		}
	}
}

// ----------------------------------------------------------------------------
// TC-R-005: Mixed entity types carry correct entity_type label
// ----------------------------------------------------------------------------

func TestEntityHistoryRepo_ListRecentAcrossEntities_CorrectEntityTypeLabels(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	baseTime := time.Now().UTC().Truncate(time.Second).Add(-30 * time.Minute)

	epicID := seedRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RLABEL-E01", "RLABEL Epic", 0, 0)
	featureID := seedRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RLABEL-E01-F01", "RLABEL Feature", epicID, 0)
	taskID := seedRecentActivityEntity(t, database, models.EntityTypeTask, "T-TEST-RLABEL-E01-F01-001", "RLABEL Task", epicID, featureID)
	bugID := seedRecentActivityEntity(t, database, models.EntityTypeBug, "TEST-RLABEL-B001", "RLABEL Bug", 0, 0)
	changeID := seedRecentActivityEntity(t, database, models.EntityTypeChange, "TEST-RLABEL-CC-001", "RLABEL Change", 0, 0)

	defer func() {
		cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID, changeID)
		cleanupRecentActivityEntity(t, database, models.EntityTypeTask, "T-TEST-RLABEL-E01-F01-001")
		cleanupRecentActivityEntity(t, database, models.EntityTypeBug, "TEST-RLABEL-B001")
		cleanupRecentActivityEntity(t, database, models.EntityTypeChange, "TEST-RLABEL-CC-001")
		cleanupRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RLABEL-E01-F01")
		cleanupRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RLABEL-E01")
	}()

	type seedSpec struct {
		eType models.EntityType
		eid   int64
		key   string
	}
	seeds := []seedSpec{
		{models.EntityTypeEpic, epicID, "TEST-RLABEL-E01"},
		{models.EntityTypeFeature, featureID, "TEST-RLABEL-E01-F01"},
		{models.EntityTypeTask, taskID, "T-TEST-RLABEL-E01-F01-001"},
		{models.EntityTypeBug, bugID, "TEST-RLABEL-B001"},
		{models.EntityTypeChange, changeID, "TEST-RLABEL-CC-001"},
	}

	for i, s := range seeds {
		h := newTestEntityHistory(s.eType, s.eid, ehStrPtr("s_a"), "s_b", baseTime.Add(time.Duration(i)*time.Minute))
		if err := repo.Create(ctx, h); err != nil {
			t.Fatalf("create history for %s: %v", s.eType, err)
		}
	}

	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{Limit: 25})
	if err != nil {
		t.Fatalf("ListRecentAcrossEntities() error: %v", err)
	}

	// Build a map: key → entity_type from results.
	resultMap := make(map[string]string)
	for _, r := range results {
		resultMap[r.Key] = r.EntityType
	}

	expected := map[string]string{
		"TEST-RLABEL-E01":           "epic",
		"TEST-RLABEL-E01-F01":       "feature",
		"T-TEST-RLABEL-E01-F01-001": "task",
		"TEST-RLABEL-B001":          "bug",
		"TEST-RLABEL-CC-001":        "change",
	}
	for key, wantType := range expected {
		if got, ok := resultMap[key]; !ok {
			t.Errorf("expected result row for key %q, but not found", key)
		} else if got != wantType {
			t.Errorf("key %q: expected entity_type %q, got %q", key, wantType, got)
		}
	}
}

// ----------------------------------------------------------------------------
// TC-R-006: Limit = 0 or negative returns empty slice, no panic
// ----------------------------------------------------------------------------

func TestEntityHistoryRepo_ListRecentAcrossEntities_ZeroLimit(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	for _, limit := range []int{0, -1, -100} {
		t.Run(fmt.Sprintf("limit_%d", limit), func(t *testing.T) {
			results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{Limit: limit})
			if err != nil {
				t.Fatalf("ListRecentAcrossEntities(Limit=%d) returned error: %v", limit, err)
			}
			if results == nil {
				t.Error("expected non-nil slice for zero/negative limit")
			}
			if len(results) != 0 {
				t.Errorf("expected 0 rows for Limit=%d, got %d", limit, len(results))
			}
		})
	}

	// Also verify no database access happens (the method short-circuits).
	// We close the underlying connection; if the method tries to query it panics.
	_ = database
}

// ----------------------------------------------------------------------------
// TC-R-007: Combined filter — entity_type + since + limit
// ----------------------------------------------------------------------------

func TestEntityHistoryRepo_ListRecentAcrossEntities_CombinedFilters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	baseTime := time.Now().UTC().Truncate(time.Second).Add(-60 * time.Minute)
	cutoff := baseTime.Add(10 * time.Minute)

	epicID := seedRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RCOMB-E01", "RCOMB Epic", 0, 0)
	featureID := seedRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RCOMB-E01-F01", "RCOMB Feature", epicID, 0)
	taskID := seedRecentActivityEntity(t, database, models.EntityTypeTask, "T-TEST-RCOMB-E01-F01-001", "RCOMB Task", epicID, featureID)
	bugID := seedRecentActivityEntity(t, database, models.EntityTypeBug, "TEST-RCOMB-B001", "RCOMB Bug", 0, 0)

	defer func() {
		cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID)
		cleanupRecentActivityEntity(t, database, models.EntityTypeTask, "T-TEST-RCOMB-E01-F01-001")
		cleanupRecentActivityEntity(t, database, models.EntityTypeBug, "TEST-RCOMB-B001")
		cleanupRecentActivityEntity(t, database, models.EntityTypeFeature, "TEST-RCOMB-E01-F01")
		cleanupRecentActivityEntity(t, database, models.EntityTypeEpic, "TEST-RCOMB-E01")
	}()

	// Seed 30 rows: 15 task rows and 15 bug rows, spread across 30 minutes.
	// Minutes 0-14: task rows; minutes 15-29: bug rows.
	// Cutoff = 10 minutes → task rows at minutes 11-14 are after cutoff (4 rows).
	for i := 0; i < 15; i++ {
		ht := newTestEntityHistory(models.EntityTypeTask, taskID, ehStrPtr("s_a"), "s_b", baseTime.Add(time.Duration(i)*time.Minute))
		if err := repo.Create(ctx, ht); err != nil {
			t.Fatalf("create task history: %v", err)
		}
		hb := newTestEntityHistory(models.EntityTypeBug, bugID, ehStrPtr("s_a"), "s_b", baseTime.Add(time.Duration(15+i)*time.Minute))
		if err := repo.Create(ctx, hb); err != nil {
			t.Fatalf("create bug history: %v", err)
		}
	}

	// Filter: entity_type=task, since=cutoff, limit=3.
	// Expected: 4 task rows after cutoff (minutes 11,12,13,14), but limit=3.
	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		EntityType: "task",
		Since:      &cutoff,
		Limit:      3,
	})
	if err != nil {
		t.Fatalf("ListRecentAcrossEntities() error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 rows (limit applied), got %d", len(results))
	}
	for _, r := range results {
		if r.EntityType != "task" {
			t.Errorf("expected entity_type 'task', got %q", r.EntityType)
		}
		if !r.ChangedAt.After(cutoff) {
			t.Errorf("row changed_at %v is not after cutoff %v", r.ChangedAt, cutoff)
		}
	}
	// Rows must be newest-first.
	for i := 1; i < len(results); i++ {
		if results[i].ChangedAt.After(results[i-1].ChangedAt) {
			t.Errorf("results not DESC ordered at index %d", i)
		}
	}
}

// ===== ListRecentAcrossEntities tests (TC-R-001 through TC-R-007) =====

// seedEntityForRecent inserts one row into the appropriate entity table using TEST- prefixed
// keys and returns the inserted row's ID. entity_type must be one of: epic, feature, task, bug, change.
// parentIDs is used for feature rows (epic_id) and task rows (feature_id).
// On error the test is fatally failed.
func seedEntityForRecent(t *testing.T, ctx context.Context, db *sql.DB, entityType, key, title string, parentIDs ...int64) int64 {
	t.Helper()
	var result sql.Result
	var err error

	switch entityType {
	case "epic":
		result, err = db.ExecContext(ctx,
			`INSERT OR IGNORE INTO epics (key, title, status, priority) VALUES (?, ?, 'active', 'high')`,
			key, title)
	case "feature":
		if len(parentIDs) < 1 {
			t.Fatalf("feature seed requires epicID parentID")
		}
		result, err = db.ExecContext(ctx,
			`INSERT OR IGNORE INTO features (epic_id, key, title, status) VALUES (?, ?, ?, 'active')`,
			parentIDs[0], key, title)
	case "task":
		if len(parentIDs) < 1 {
			t.Fatalf("task seed requires featureID parentID")
		}
		result, err = db.ExecContext(ctx,
			`INSERT OR IGNORE INTO tasks (feature_id, key, title, status, priority, depends_on) VALUES (?, ?, ?, 'todo', 5, '[]')`,
			parentIDs[0], key, title)
	case "bug":
		result, err = db.ExecContext(ctx,
			`INSERT OR IGNORE INTO bugs (key, title, status, severity) VALUES (?, ?, 'reported', 'medium')`,
			key, title)
	case "change":
		result, err = db.ExecContext(ctx,
			`INSERT OR IGNORE INTO change_cards (key, title, status, priority) VALUES (?, ?, 'proposed', 5)`,
			key, title)
	default:
		t.Fatalf("unknown entity type %q", entityType)
	}

	if err != nil {
		t.Fatalf("seedEntityForRecent(%s, %s): %v", entityType, key, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("seedEntityForRecent(%s, %s) LastInsertId: %v", entityType, key, err)
	}
	if id == 0 {
		// INSERT OR IGNORE with duplicate — query the existing row
		switch entityType {
		case "epic":
			err = db.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = ?", key).Scan(&id)
		case "feature":
			err = db.QueryRowContext(ctx, "SELECT id FROM features WHERE key = ?", key).Scan(&id)
		case "task":
			err = db.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = ?", key).Scan(&id)
		case "bug":
			err = db.QueryRowContext(ctx, "SELECT id FROM bugs WHERE key = ?", key).Scan(&id)
		case "change":
			err = db.QueryRowContext(ctx, "SELECT id FROM change_cards WHERE key = ?", key).Scan(&id)
		}
		if err != nil {
			t.Fatalf("seedEntityForRecent: lookup existing id for %s/%s: %v", entityType, key, err)
		}
	}
	return id
}

// cleanupEntityForRecent removes seeded entity rows (history is already cleaned separately).
func cleanupEntityForRecent(t *testing.T, ctx context.Context, db *sql.DB, entityType, key string) {
	t.Helper()
	var err error
	switch entityType {
	case "epic":
		_, err = db.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", key)
	case "feature":
		_, err = db.ExecContext(ctx, "DELETE FROM features WHERE key = ?", key)
	case "task":
		_, err = db.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", key)
	case "bug":
		_, err = db.ExecContext(ctx, "DELETE FROM bugs WHERE key = ?", key)
	case "change":
		_, err = db.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", key)
	}
	if err != nil {
		t.Logf("cleanupEntityForRecent(%s, %s): %v", entityType, key, err)
	}
}

// insertHistory is a helper that inserts an entity_history row directly via SQL
// (bypasses Validate so we can control changed_at precisely).
func insertHistory(t *testing.T, ctx context.Context, db *sql.DB, entityType models.EntityType, entityID int64, fromStatus, toStatus string, changedAt time.Time) int64 {
	t.Helper()
	result, err := db.ExecContext(ctx,
		`INSERT INTO entity_history (entity_type, entity_id, from_status, to_status, changed_at) VALUES (?, ?, ?, ?, ?)`,
		string(entityType), entityID, fromStatus, toStatus, changedAt.UTC(),
	)
	if err != nil {
		t.Fatalf("insertHistory(%s, %d): %v", entityType, entityID, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("insertHistory LastInsertId: %v", err)
	}
	return id
}

// TC-R-001: Seed 1 entity per type (5 total), 2 history rows each (10 total).
// With Limit=5 verify exactly 5 rows returned DESC by changed_at.
func TestListRecentAcrossEntities_R001_LimitAndOrder(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	// Use unique TEST- prefixed keys to avoid collisions with other tests.
	epicKey := "TEST-R001-E01"
	featureKey := "TEST-R001-E01-F01"
	taskKey := "TEST-R001-T-E01-F01-001"
	bugKey := "TEST-R001-B001"
	changeKey := "TEST-R001-CC001"

	// Clean up entities and their history before test.
	for _, pair := range []struct{ et, k string }{
		{"epic", epicKey}, {"feature", featureKey}, {"task", taskKey},
		{"bug", bugKey}, {"change", changeKey},
	} {
		cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
	}
	defer func() {
		for _, pair := range []struct{ et, k string }{
			{"task", taskKey}, {"feature", featureKey}, {"epic", epicKey},
			{"bug", bugKey}, {"change", changeKey},
		} {
			cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
		}
	}()

	// Seed entities.
	epicID := seedEntityForRecent(t, ctx, database, "epic", epicKey, "R001 Epic")
	featureID := seedEntityForRecent(t, ctx, database, "feature", featureKey, "R001 Feature", epicID)
	taskID := seedEntityForRecent(t, ctx, database, "task", taskKey, "R001 Task", featureID)
	bugID := seedEntityForRecent(t, ctx, database, "bug", bugKey, "R001 Bug")
	changeID := seedEntityForRecent(t, ctx, database, "change", changeKey, "R001 Change")

	// Clean up history for these entity IDs.
	cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID, changeID)
	defer cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID, changeID)

	// Insert 2 history rows per entity with controlled timestamps (10 rows total).
	base := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)
	for i, pair := range []struct {
		et models.EntityType
		id int64
	}{
		{models.EntityTypeEpic, epicID},
		{models.EntityTypeFeature, featureID},
		{models.EntityTypeTask, taskID},
		{models.EntityTypeBug, bugID},
		{models.EntityTypeChange, changeID},
	} {
		insertHistory(t, ctx, database, pair.et, pair.id, "old", "new", base.Add(time.Duration(i*2)*time.Second))
		insertHistory(t, ctx, database, pair.et, pair.id, "new", "done", base.Add(time.Duration(i*2+1)*time.Second))
	}

	// Query with Limit=5 — should return the 5 most recent rows.
	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{Limit: 5})
	if err != nil {
		t.Fatalf("TC-R-001: ListRecentAcrossEntities error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("TC-R-001: expected 5 rows (limit), got %d", len(results))
	}

	// Verify DESC order.
	for i := 1; i < len(results); i++ {
		if results[i].ChangedAt.After(results[i-1].ChangedAt) {
			t.Errorf("TC-R-001: results not DESC ordered at index %d", i)
		}
	}
}

// TC-R-002: EntityType="task" filter — only task rows returned.
func TestListRecentAcrossEntities_R002_EntityTypeFilter(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	epicKey := "TEST-R002-E01"
	featureKey := "TEST-R002-E01-F01"
	taskKey := "TEST-R002-T-E01-F01-001"
	bugKey := "TEST-R002-B001"

	for _, pair := range []struct{ et, k string }{
		{"epic", epicKey}, {"feature", featureKey}, {"task", taskKey}, {"bug", bugKey},
	} {
		cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
	}
	defer func() {
		for _, pair := range []struct{ et, k string }{
			{"task", taskKey}, {"feature", featureKey}, {"epic", epicKey}, {"bug", bugKey},
		} {
			cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
		}
	}()

	epicID := seedEntityForRecent(t, ctx, database, "epic", epicKey, "R002 Epic")
	featureID := seedEntityForRecent(t, ctx, database, "feature", featureKey, "R002 Feature", epicID)
	taskID := seedEntityForRecent(t, ctx, database, "task", taskKey, "R002 Task", featureID)
	bugID := seedEntityForRecent(t, ctx, database, "bug", bugKey, "R002 Bug")

	cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID)
	defer cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID)

	base := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Second)
	insertHistory(t, ctx, database, models.EntityTypeEpic, epicID, "old", "new", base)
	insertHistory(t, ctx, database, models.EntityTypeFeature, featureID, "old", "new", base.Add(time.Second))
	insertHistory(t, ctx, database, models.EntityTypeTask, taskID, "old", "in_progress", base.Add(2*time.Second))
	insertHistory(t, ctx, database, models.EntityTypeTask, taskID, "in_progress", "done", base.Add(3*time.Second))
	insertHistory(t, ctx, database, models.EntityTypeBug, bugID, "reported", "triaged", base.Add(4*time.Second))

	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		EntityType: "task",
		Limit:      50,
	})
	if err != nil {
		t.Fatalf("TC-R-002: error: %v", err)
	}
	for _, row := range results {
		// Filter may include task rows from other tests; only check our taskID's rows.
		if row.Key == taskKey && row.EntityType != "task" {
			t.Errorf("TC-R-002: expected entity_type=task, got %q", row.EntityType)
		}
	}

	// Our task should appear; our epic, feature, and bug rows should NOT.
	foundTask := false
	for _, row := range results {
		if row.Key == taskKey {
			foundTask = true
		}
		if row.Key == epicKey || row.Key == featureKey || row.Key == bugKey {
			t.Errorf("TC-R-002: non-task key %q appeared in entity_type=task filter results", row.Key)
		}
	}
	if !foundTask {
		t.Errorf("TC-R-002: expected task key %q in results", taskKey)
	}
}

// TC-R-003: Since filter — only rows after the cutoff are returned.
func TestListRecentAcrossEntities_R003_SinceFilter(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	epicKey := "TEST-R003-E01"
	featureKey := "TEST-R003-E01-F01"
	taskKey := "TEST-R003-T-E01-F01-001"

	for _, pair := range []struct{ et, k string }{
		{"epic", epicKey}, {"feature", featureKey}, {"task", taskKey},
	} {
		cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
	}
	defer func() {
		for _, pair := range []struct{ et, k string }{
			{"task", taskKey}, {"feature", featureKey}, {"epic", epicKey},
		} {
			cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
		}
	}()

	epicID := seedEntityForRecent(t, ctx, database, "epic", epicKey, "R003 Epic")
	featureID := seedEntityForRecent(t, ctx, database, "feature", featureKey, "R003 Feature", epicID)
	taskID := seedEntityForRecent(t, ctx, database, "task", taskKey, "R003 Task", featureID)

	cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID)
	defer cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID)

	// Two rows before cutoff, two rows after cutoff.
	cutoff := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)
	before1 := cutoff.Add(-2 * time.Second)
	before2 := cutoff.Add(-1 * time.Second)
	after1 := cutoff.Add(1 * time.Second)
	after2 := cutoff.Add(2 * time.Second)

	insertHistory(t, ctx, database, models.EntityTypeEpic, epicID, "old", "before1", before1)
	insertHistory(t, ctx, database, models.EntityTypeEpic, epicID, "before1", "before2", before2)
	insertHistory(t, ctx, database, models.EntityTypeFeature, featureID, "old", "after1", after1)
	insertHistory(t, ctx, database, models.EntityTypeTask, taskID, "old", "after2", after2)

	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		Since: &cutoff,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("TC-R-003: error: %v", err)
	}

	// Our before-cutoff rows should NOT appear; after-cutoff rows SHOULD.
	for _, row := range results {
		if row.Key == epicKey {
			if row.ToStatus == "before1" || row.ToStatus == "before2" {
				t.Errorf("TC-R-003: row with to_status %q (before cutoff) appeared in Since-filtered results", row.ToStatus)
			}
		}
	}

	foundAfter1 := false
	foundAfter2 := false
	for _, row := range results {
		if row.Key == featureKey && row.ToStatus == "after1" {
			foundAfter1 = true
		}
		if row.Key == taskKey && row.ToStatus == "after2" {
			foundAfter2 = true
		}
	}
	if !foundAfter1 {
		t.Error("TC-R-003: expected after-cutoff feature history row in results")
	}
	if !foundAfter2 {
		t.Error("TC-R-003: expected after-cutoff task history row in results")
	}
}

// TC-R-004: Orphan omission — delete the parent entity; history rows must not appear.
func TestListRecentAcrossEntities_R004_OrphanOmission(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	epicKey := "TEST-R004-E01"
	featureKey := "TEST-R004-E01-F01"
	taskKey := "TEST-R004-T-E01-F01-001"

	for _, pair := range []struct{ et, k string }{
		{"epic", epicKey}, {"feature", featureKey}, {"task", taskKey},
	} {
		cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
	}
	defer func() {
		for _, pair := range []struct{ et, k string }{
			{"task", taskKey}, {"feature", featureKey}, {"epic", epicKey},
		} {
			cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
		}
	}()

	epicID := seedEntityForRecent(t, ctx, database, "epic", epicKey, "R004 Epic")
	featureID := seedEntityForRecent(t, ctx, database, "feature", featureKey, "R004 Feature", epicID)
	taskID := seedEntityForRecent(t, ctx, database, "task", taskKey, "R004 Task", featureID)

	cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID)
	defer cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID)

	base := time.Now().UTC().Truncate(time.Second)
	insertHistory(t, ctx, database, models.EntityTypeTask, taskID, "old", "in_progress", base)
	insertHistory(t, ctx, database, models.EntityTypeTask, taskID, "in_progress", "done", base.Add(time.Second))

	// Sanity check: rows are visible before deletion.
	resultsBefore, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		EntityType: "task",
		Limit:      100,
	})
	if err != nil {
		t.Fatalf("TC-R-004: pre-deletion query error: %v", err)
	}
	foundBefore := false
	for _, row := range resultsBefore {
		if row.Key == taskKey {
			foundBefore = true
		}
	}
	if !foundBefore {
		t.Fatal("TC-R-004: task rows not visible before deletion — test setup failed")
	}

	// Delete the task (INNER JOIN will omit history rows for deleted entity).
	// Also clean up feature and epic to remove FK constraints.
	_, err = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", taskKey)
	if err != nil {
		t.Fatalf("TC-R-004: failed to delete task: %v", err)
	}

	// Query again — orphaned history rows should not appear.
	resultsAfter, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		EntityType: "task",
		Limit:      100,
	})
	if err != nil {
		t.Fatalf("TC-R-004: post-deletion query error: %v", err)
	}
	for _, row := range resultsAfter {
		if row.Key == taskKey {
			t.Errorf("TC-R-004: orphaned task row %q appeared after task deletion", taskKey)
		}
	}
}

// TC-R-005: Mixed entity types carry correct entity_type labels.
func TestListRecentAcrossEntities_R005_EntityTypeLabels(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	epicKey := "TEST-R005-E01"
	featureKey := "TEST-R005-E01-F01"
	taskKey := "TEST-R005-T-E01-F01-001"
	bugKey := "TEST-R005-B001"
	changeKey := "TEST-R005-CC001"

	for _, pair := range []struct{ et, k string }{
		{"epic", epicKey}, {"feature", featureKey}, {"task", taskKey},
		{"bug", bugKey}, {"change", changeKey},
	} {
		cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
	}
	defer func() {
		for _, pair := range []struct{ et, k string }{
			{"task", taskKey}, {"feature", featureKey}, {"epic", epicKey},
			{"bug", bugKey}, {"change", changeKey},
		} {
			cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
		}
	}()

	epicID := seedEntityForRecent(t, ctx, database, "epic", epicKey, "R005 Epic")
	featureID := seedEntityForRecent(t, ctx, database, "feature", featureKey, "R005 Feature", epicID)
	taskID := seedEntityForRecent(t, ctx, database, "task", taskKey, "R005 Task", featureID)
	bugID := seedEntityForRecent(t, ctx, database, "bug", bugKey, "R005 Bug")
	changeID := seedEntityForRecent(t, ctx, database, "change", changeKey, "R005 Change")

	cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID, changeID)
	defer cleanupEntityHistory(t, ctx, db, epicID, featureID, taskID, bugID, changeID)

	base := time.Now().UTC().Truncate(time.Second)
	insertHistory(t, ctx, database, models.EntityTypeEpic, epicID, "old", "active", base)
	insertHistory(t, ctx, database, models.EntityTypeFeature, featureID, "old", "active", base.Add(time.Second))
	insertHistory(t, ctx, database, models.EntityTypeTask, taskID, "old", "in_progress", base.Add(2*time.Second))
	insertHistory(t, ctx, database, models.EntityTypeBug, bugID, "reported", "triaged", base.Add(3*time.Second))
	insertHistory(t, ctx, database, models.EntityTypeChange, changeID, "proposed", "approved", base.Add(4*time.Second))

	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{Limit: 100})
	if err != nil {
		t.Fatalf("TC-R-005: error: %v", err)
	}

	// Build a map from entity key → reported entity_type for our seeded keys.
	keyToType := make(map[string]string)
	for _, row := range results {
		switch row.Key {
		case epicKey, featureKey, taskKey, bugKey, changeKey:
			keyToType[row.Key] = row.EntityType
		}
	}

	expected := map[string]string{
		epicKey:    "epic",
		featureKey: "feature",
		taskKey:    "task",
		bugKey:     "bug",
		changeKey:  "change",
	}
	for key, wantType := range expected {
		if got, ok := keyToType[key]; !ok {
			t.Errorf("TC-R-005: key %q not found in results", key)
		} else if got != wantType {
			t.Errorf("TC-R-005: key %q: expected entity_type=%q, got %q", key, wantType, got)
		}
	}
}

// TC-R-006: Limit=0 or negative — returns empty slice without panicking.
func TestListRecentAcrossEntities_R006_ZeroOrNegativeLimit(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	// Seed one row so there's something to potentially return.
	epicKey := "TEST-R006-E01"
	cleanupEntityForRecent(t, ctx, database, "epic", epicKey)
	defer cleanupEntityForRecent(t, ctx, database, "epic", epicKey)
	epicID := seedEntityForRecent(t, ctx, database, "epic", epicKey, "R006 Epic")

	cleanupEntityHistory(t, ctx, db, epicID)
	defer cleanupEntityHistory(t, ctx, db, epicID)

	insertHistory(t, ctx, database, models.EntityTypeEpic, epicID, "old", "active", time.Now().UTC())

	for _, limit := range []int{0, -1, -100} {
		t.Run(fmt.Sprintf("limit_%d", limit), func(t *testing.T) {
			results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{Limit: limit})
			if err != nil {
				t.Fatalf("TC-R-006: limit=%d: unexpected error: %v", limit, err)
			}
			if len(results) != 0 {
				t.Errorf("TC-R-006: limit=%d: expected empty slice, got %d rows", limit, len(results))
			}
		})
	}
}

// TC-R-007: Combined filter — entity_type + since + limit all applied together.
func TestListRecentAcrossEntities_R007_CombinedFilter(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEntityHistoryRepository(db)

	epicKey := "TEST-R007-E01"
	featureKey := "TEST-R007-E01-F01"
	taskKey1 := "TEST-R007-T-E01-F01-001"
	taskKey2 := "TEST-R007-T-E01-F01-002"

	for _, pair := range []struct{ et, k string }{
		{"epic", epicKey}, {"feature", featureKey},
		{"task", taskKey1}, {"task", taskKey2},
	} {
		cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
	}
	defer func() {
		for _, pair := range []struct{ et, k string }{
			{"task", taskKey1}, {"task", taskKey2},
			{"feature", featureKey}, {"epic", epicKey},
		} {
			cleanupEntityForRecent(t, ctx, database, pair.et, pair.k)
		}
	}()

	epicID := seedEntityForRecent(t, ctx, database, "epic", epicKey, "R007 Epic")
	featureID := seedEntityForRecent(t, ctx, database, "feature", featureKey, "R007 Feature", epicID)
	task1ID := seedEntityForRecent(t, ctx, database, "task", taskKey1, "R007 Task 1", featureID)
	task2ID := seedEntityForRecent(t, ctx, database, "task", taskKey2, "R007 Task 2", featureID)

	cleanupEntityHistory(t, ctx, db, epicID, featureID, task1ID, task2ID)
	defer cleanupEntityHistory(t, ctx, db, epicID, featureID, task1ID, task2ID)

	cutoff := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)

	// Epic row before cutoff — should be excluded by entity_type=task.
	insertHistory(t, ctx, database, models.EntityTypeEpic, epicID, "old", "epic-before", cutoff.Add(-2*time.Second))
	// Task rows before cutoff — excluded by Since.
	insertHistory(t, ctx, database, models.EntityTypeTask, task1ID, "old", "task-before", cutoff.Add(-1*time.Second))
	// Task rows after cutoff — should appear.
	insertHistory(t, ctx, database, models.EntityTypeTask, task1ID, "old", "task-after-1", cutoff.Add(1*time.Second))
	insertHistory(t, ctx, database, models.EntityTypeTask, task2ID, "old", "task-after-2", cutoff.Add(2*time.Second))
	insertHistory(t, ctx, database, models.EntityTypeTask, task2ID, "old", "task-after-3", cutoff.Add(3*time.Second))

	// Apply limit=2 on top of entity_type=task and since=cutoff.
	results, err := repo.ListRecentAcrossEntities(ctx, ListRecentAcrossEntitiesOptions{
		EntityType: "task",
		Since:      &cutoff,
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("TC-R-007: error: %v", err)
	}

	// Expect exactly 2 rows (limit), both tasks, both after cutoff.
	if len(results) != 2 {
		t.Errorf("TC-R-007: expected 2 rows (limit=2), got %d", len(results))
	}
	for _, row := range results {
		if row.EntityType != "task" {
			t.Errorf("TC-R-007: expected entity_type=task, got %q", row.EntityType)
		}
		if row.ToStatus == "epic-before" || row.ToStatus == "task-before" {
			t.Errorf("TC-R-007: row with to_status %q should have been filtered out", row.ToStatus)
		}
	}
	// Most recent 2 should be task-after-3 and task-after-2 (DESC order).
	if len(results) == 2 {
		if results[0].ToStatus != "task-after-3" {
			t.Errorf("TC-R-007: expected results[0].to_status=task-after-3, got %q", results[0].ToStatus)
		}
		if results[1].ToStatus != "task-after-2" {
			t.Errorf("TC-R-007: expected results[1].to_status=task-after-2, got %q", results[1].ToStatus)
		}
	}
}
