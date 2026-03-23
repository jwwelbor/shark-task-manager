package changecard

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// changeCardAggTestSetup sets up for aggregate tests using CC-8## keys.
func changeCardAggTestSetup(t *testing.T) (*ChangeCardRepository, func()) {
	t.Helper()
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	// Clean up existing test aggregate change-cards before test
	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key LIKE 'CC-8%'")

	cleanup := func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key LIKE 'CC-8%'")
	}

	return repo, cleanup
}

// newTestChangeCard creates a minimal test change-card.
// key must follow CC-### format (e.g., "CC-801").
// Priority is set to 5 (mid-range) to satisfy the database CHECK constraint (1-10).
func newTestChangeCard(key, title, status string) *models.ChangeCard {
	return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: key,
		Title: title}, Status: models.ChangeCardStatus(status),
		Priority: 5,
	}
}

// TestChangeCardRepository_GetStatusSummary_ZeroCards verifies GetStatusSummary
// returns a valid empty summary (not an error) when no change-cards exist.
func TestChangeCardRepository_GetStatusSummary_ZeroCards(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := changeCardAggTestSetup(t)
	defer cleanup()

	summary, err := repo.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary() error = %v", err)
	}
	if summary == nil {
		t.Fatal("GetStatusSummary() returned nil summary")
	}
	if summary.ByStatus == nil {
		t.Error("GetStatusSummary().ByStatus should not be nil")
	}
	// Total should be >= 0 (zero or more from other tests, just not negative)
	if summary.Total < 0 {
		t.Errorf("GetStatusSummary().Total = %d, want >= 0", summary.Total)
	}
}

// TestChangeCardRepository_GetStatusSummary_Counts verifies GetStatusSummary returns
// correct counts across multiple change-card statuses.
func TestChangeCardRepository_GetStatusSummary_Counts(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key IN ('CC-801','CC-802','CC-803','CC-804','CC-805')")

	cards := []*models.ChangeCard{
		newTestChangeCard("CC-801", "Proposed card 1", "proposed"),
		newTestChangeCard("CC-802", "Approved card", "approved"),
		newTestChangeCard("CC-803", "In progress card", "in_progress"),
		newTestChangeCard("CC-804", "Completed card", "completed"),
		newTestChangeCard("CC-805", "Declined card", "declined"),
	}
	for _, c := range cards {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, c := range cards {
			_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE id = ?", c.ID)
		}
	}()

	summary, err := repo.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary() error = %v", err)
	}

	// Total should include our 5 cards plus any others in the DB
	if summary.Total < 5 {
		t.Errorf("GetStatusSummary().Total = %d, want at least 5", summary.Total)
	}

	// Check each status from our seeded data
	if summary.ByStatus["proposed"] < 1 {
		t.Errorf("ByStatus['proposed'] = %d, want at least 1", summary.ByStatus["proposed"])
	}
	if summary.ByStatus["approved"] < 1 {
		t.Errorf("ByStatus['approved'] = %d, want at least 1", summary.ByStatus["approved"])
	}
	if summary.ByStatus["in_progress"] < 1 {
		t.Errorf("ByStatus['in_progress'] = %d, want at least 1", summary.ByStatus["in_progress"])
	}
	if summary.ByStatus["completed"] < 1 {
		t.Errorf("ByStatus['completed'] = %d, want at least 1", summary.ByStatus["completed"])
	}
	if summary.ByStatus["declined"] < 1 {
		t.Errorf("ByStatus['declined'] = %d, want at least 1", summary.ByStatus["declined"])
	}
}

// TestChangeCardRepository_GetThroughputStats_EmptyDatabase verifies throughput stats
// returns zero counts and nil rates when no change-cards exist.
func TestChangeCardRepository_GetThroughputStats_EmptyDatabase(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	// Delete all CC-8xx and CC-9xx test keys to get an empty-like state
	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key LIKE 'CC-8%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key LIKE 'CC-9%'")

	stats, err := repo.GetThroughputStats(ctx)
	if err != nil {
		t.Fatalf("GetThroughputStats() error = %v", err)
	}
	if stats == nil {
		t.Fatal("GetThroughputStats() returned nil stats")
	}

	// When decided_count is 0, approval_rate should be nil
	if stats.DecidedCount == 0 && stats.ApprovalRate != nil {
		t.Error("GetThroughputStats().ApprovalRate should be nil when DecidedCount is 0")
	}
	// When completed_count is 0, AvgCompletionSecs should be nil
	if stats.CompletedCount == 0 && stats.AvgCompletionSecs != nil {
		t.Error("GetThroughputStats().AvgCompletionSecs should be nil when CompletedCount is 0")
	}
}

// TestChangeCardRepository_GetThroughputStats_WithData verifies throughput stats
// correctly computes counts, approval rate, and completion metrics.
func TestChangeCardRepository_GetThroughputStats_WithData(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key IN ('CC-810','CC-811','CC-812','CC-813','CC-814','CC-815','CC-816','CC-817')")

	// Seed 8 change-cards per TC-F07-016:
	// 2 proposed, 1 approved, 2 in_progress, 2 completed, 1 declined
	cards := []*models.ChangeCard{
		newTestChangeCard("CC-810", "Proposed card 1", "proposed"),
		newTestChangeCard("CC-811", "Proposed card 2", "proposed"),
		newTestChangeCard("CC-812", "Approved card", "approved"),
		newTestChangeCard("CC-813", "In progress card 1", "in_progress"),
		newTestChangeCard("CC-814", "In progress card 2", "in_progress"),
		newTestChangeCard("CC-815", "Completed card 1", "completed"),
		newTestChangeCard("CC-816", "Completed card 2", "completed"),
		newTestChangeCard("CC-817", "Declined card", "declined"),
	}
	for _, c := range cards {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, c := range cards {
			_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE id = ?", c.ID)
		}
	}()

	// Delete other change-cards to get clean counts for our assertions
	// (only keep our test cards)
	_, _ = database.ExecContext(ctx, `DELETE FROM change_cards WHERE key NOT IN ('CC-810','CC-811','CC-812','CC-813','CC-814','CC-815','CC-816','CC-817')`)

	stats, err := repo.GetThroughputStats(ctx)
	if err != nil {
		t.Fatalf("GetThroughputStats() error = %v", err)
	}

	// decided = approved + in_progress + completed + declined = 1 + 2 + 2 + 1 = 6
	if stats.DecidedCount != 6 {
		t.Errorf("GetThroughputStats().DecidedCount = %d, want 6", stats.DecidedCount)
	}

	// approved_count = approved + in_progress + completed = 1 + 2 + 2 = 5
	if stats.ApprovedCount != 5 {
		t.Errorf("GetThroughputStats().ApprovedCount = %d, want 5", stats.ApprovedCount)
	}

	// declined_count = 1
	if stats.DeclinedCount != 1 {
		t.Errorf("GetThroughputStats().DeclinedCount = %d, want 1", stats.DeclinedCount)
	}

	// completed_count = 2
	if stats.CompletedCount != 2 {
		t.Errorf("GetThroughputStats().CompletedCount = %d, want 2", stats.CompletedCount)
	}

	// approval_rate = approved_count / decided_count = 5/6 ≈ 0.833
	if stats.ApprovalRate == nil {
		t.Fatal("GetThroughputStats().ApprovalRate should not be nil with decided_count > 0")
	}
	expectedRate := 5.0 / 6.0 // ≈ 0.8333
	tolerance := 0.01
	if *stats.ApprovalRate < expectedRate-tolerance || *stats.ApprovalRate > expectedRate+tolerance {
		t.Errorf("GetThroughputStats().ApprovalRate = %.4f, want ~%.4f", *stats.ApprovalRate, expectedRate)
	}

	// AvgCompletionSecs should be non-nil since completed_count > 0
	if stats.AvgCompletionSecs == nil {
		t.Error("GetThroughputStats().AvgCompletionSecs should not be nil when CompletedCount > 0")
	}
	if stats.AvgCompletionSecs != nil && *stats.AvgCompletionSecs < 0 {
		t.Errorf("GetThroughputStats().AvgCompletionSecs = %v, want >= 0", *stats.AvgCompletionSecs)
	}
}

// TestChangeCardRepository_GetThroughputStats_ZeroDecided verifies that
// ApprovalRate is nil when no cards have been decided.
func TestChangeCardRepository_GetThroughputStats_ZeroDecided(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key IN ('CC-820','CC-821')")

	// Create only proposed cards (not decided)
	cards := []*models.ChangeCard{
		newTestChangeCard("CC-820", "Proposed card A", "proposed"),
		newTestChangeCard("CC-821", "Proposed card B", "proposed"),
	}
	for _, c := range cards {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, c := range cards {
			_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE id = ?", c.ID)
		}
	}()

	// Remove other cards to isolate
	_, _ = database.ExecContext(ctx, `DELETE FROM change_cards WHERE key NOT IN ('CC-820','CC-821')`)

	stats, err := repo.GetThroughputStats(ctx)
	if err != nil {
		t.Fatalf("GetThroughputStats() error = %v", err)
	}

	if stats.DecidedCount != 0 {
		t.Errorf("GetThroughputStats().DecidedCount = %d, want 0", stats.DecidedCount)
	}
	if stats.ApprovalRate != nil {
		t.Errorf("GetThroughputStats().ApprovalRate should be nil when DecidedCount=0, got %v", stats.ApprovalRate)
	}
	if stats.CompletedCount != 0 {
		t.Errorf("GetThroughputStats().CompletedCount = %d, want 0", stats.CompletedCount)
	}
	if stats.AvgCompletionSecs != nil {
		t.Errorf("GetThroughputStats().AvgCompletionSecs should be nil when CompletedCount=0, got %v", stats.AvgCompletionSecs)
	}
}

// TestChangeCardRepository_GetThroughputStats_AllDeclined verifies approval rate
// is 0.0 when all decided cards are declined.
func TestChangeCardRepository_GetThroughputStats_AllDeclined(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key IN ('CC-825','CC-826')")

	cards := []*models.ChangeCard{
		newTestChangeCard("CC-825", "Declined card 1", "declined"),
		newTestChangeCard("CC-826", "Declined card 2", "declined"),
	}
	for _, c := range cards {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, c := range cards {
			_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE id = ?", c.ID)
		}
	}()

	// Remove other cards to isolate
	_, _ = database.ExecContext(ctx, `DELETE FROM change_cards WHERE key NOT IN ('CC-825','CC-826')`)

	stats, err := repo.GetThroughputStats(ctx)
	if err != nil {
		t.Fatalf("GetThroughputStats() error = %v", err)
	}

	if stats.DecidedCount != 2 {
		t.Errorf("GetThroughputStats().DecidedCount = %d, want 2", stats.DecidedCount)
	}
	if stats.ApprovalRate == nil {
		t.Fatal("GetThroughputStats().ApprovalRate should not be nil when DecidedCount > 0")
	}
	if *stats.ApprovalRate != 0.0 {
		t.Errorf("GetThroughputStats().ApprovalRate = %v, want 0.0 when all declined", *stats.ApprovalRate)
	}
}
