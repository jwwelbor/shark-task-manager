package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// bugAggTestSetup sets up for aggregate tests using B8xx keys.
func bugAggTestSetup(t *testing.T) (*BugRepository, func()) {
	t.Helper()
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Clean up existing test aggregate bugs before test
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B8%'")

	cleanup := func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B8%'")
	}

	return repo, cleanup
}

// TestBugRepository_GetStatusSummary_ZeroBugs verifies that GetStatusSummary returns
// empty counts (not an error) when no bugs exist matching the test prefix.
func TestBugRepository_GetStatusSummary_ZeroBugs(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugAggTestSetup(t)
	defer cleanup()

	// Don't create any bugs -- all B8xx bugs are cleaned up in setup

	// We need to isolate this test from other bugs in the database.
	// GetStatusSummary counts all bugs, so we need to verify the return type
	// and that it doesn't error, not exact counts.
	summary, err := repo.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary() with no test bugs error = %v", err)
	}
	if summary == nil {
		t.Fatal("GetStatusSummary() returned nil summary")
	}
	if summary.ByStatus == nil {
		t.Error("GetStatusSummary() ByStatus map should not be nil")
	}
	if summary.BySeverity == nil {
		t.Error("GetStatusSummary() BySeverity map should not be nil")
	}
	if summary.OpenBySeverity == nil {
		t.Error("GetStatusSummary() OpenBySeverity map should not be nil")
	}
}

// TestBugRepository_GetStatusSummary_Counts verifies that GetStatusSummary returns
// correct bug counts across multiple statuses and that open/terminal distinction works.
func TestBugRepository_GetStatusSummary_Counts(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Clean and create a controlled set of bugs
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key IN ('B801','B802','B803','B804','B805')")

	bugs := []*models.Bug{
		newTestBug("B801", "Open critical bug", "reported", models.BugSeverityCritical),
		newTestBug("B802", "Open high bug", "in_fix", models.BugSeverityHigh),
		newTestBug("B803", "Open high bug 2", "triaged", models.BugSeverityHigh),
		newTestBug("B804", "Resolved medium bug", "resolved", models.BugSeverityMedium),
		newTestBug("B805", "Wont fix low bug", "wont_fix", models.BugSeverityLow),
	}
	for _, b := range bugs {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, b := range bugs {
			_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
		}
	}()

	summary, err := repo.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary() error = %v", err)
	}

	// Total should include all 5 bugs (plus whatever else is in the DB from other tests)
	// We check our known bugs are accounted for
	if summary.Total < 5 {
		t.Errorf("GetStatusSummary().Total = %d, want at least 5", summary.Total)
	}

	// Status counts for our seeded bugs
	if summary.ByStatus["reported"] < 1 {
		t.Errorf("ByStatus['reported'] = %d, want at least 1", summary.ByStatus["reported"])
	}
	if summary.ByStatus["in_fix"] < 1 {
		t.Errorf("ByStatus['in_fix'] = %d, want at least 1", summary.ByStatus["in_fix"])
	}
	if summary.ByStatus["triaged"] < 1 {
		t.Errorf("ByStatus['triaged'] = %d, want at least 1", summary.ByStatus["triaged"])
	}
	if summary.ByStatus["resolved"] < 1 {
		t.Errorf("ByStatus['resolved'] = %d, want at least 1", summary.ByStatus["resolved"])
	}
	if summary.ByStatus["wont_fix"] < 1 {
		t.Errorf("ByStatus['wont_fix'] = %d, want at least 1", summary.ByStatus["wont_fix"])
	}

	// BySeverity counts (all bugs, including terminal)
	if summary.BySeverity["critical"] < 1 {
		t.Errorf("BySeverity['critical'] = %d, want at least 1", summary.BySeverity["critical"])
	}
	if summary.BySeverity["high"] < 2 {
		t.Errorf("BySeverity['high'] = %d, want at least 2", summary.BySeverity["high"])
	}

	// OpenBySeverity should EXCLUDE terminal bugs (resolved, wont_fix, duplicate)
	// B801 (critical/reported) and B802 (high/in_fix) and B803 (high/triaged) are open
	// B804 (medium/resolved) and B805 (low/wont_fix) are terminal -- should NOT be in OpenBySeverity
	if summary.OpenBySeverity["critical"] < 1 {
		t.Errorf("OpenBySeverity['critical'] = %d, want at least 1", summary.OpenBySeverity["critical"])
	}
	if summary.OpenBySeverity["high"] < 2 {
		t.Errorf("OpenBySeverity['high'] = %d, want at least 2", summary.OpenBySeverity["high"])
	}
	// medium bug (B804) is resolved -- should have 0 in OpenBySeverity from our test bugs
	// (Other tests may have created medium open bugs, so we can only check it's not negative)
}

// TestBugRepository_GetStatusSummary_OpenSeverityExcludesTerminal verifies that
// OpenBySeverity specifically excludes bugs in terminal statuses.
func TestBugRepository_GetStatusSummary_OpenSeverityExcludesTerminal(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Use unique keys for isolation
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key IN ('B811','B812','B813')")

	// Create only terminal bugs with a unique severity (critical) not used in other agg tests
	// We'll verify that with only terminal bugs, OpenBySeverity count doesn't grow
	terminalBugs := []*models.Bug{
		newTestBug("B811", "Resolved critical bug", "resolved", models.BugSeverityCritical),
		newTestBug("B812", "Wont fix critical bug", "wont_fix", models.BugSeverityCritical),
		newTestBug("B813", "Duplicate critical bug", "duplicate", models.BugSeverityCritical),
	}
	for _, b := range terminalBugs {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("Create() terminal bug error = %v", err)
		}
	}
	defer func() {
		for _, b := range terminalBugs {
			_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
		}
	}()

	// Get summary before removing -- these terminal bugs should not increase OpenBySeverity
	summaryBefore, err := repo.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary() error = %v", err)
	}
	openCriticalBefore := summaryBefore.OpenBySeverity["critical"]

	// Delete the terminal bugs and re-query
	for _, b := range terminalBugs {
		_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
	}

	summaryAfter, err := repo.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary() after delete error = %v", err)
	}
	openCriticalAfter := summaryAfter.OpenBySeverity["critical"]

	// OpenBySeverity should be the same (terminal bugs don't contribute)
	if openCriticalBefore != openCriticalAfter {
		t.Errorf("OpenBySeverity['critical'] changed when adding/removing terminal bugs: before=%d after=%d",
			openCriticalBefore, openCriticalAfter)
	}
}

// TestBugRepository_GetResolutionStats_NoTerminalBugs verifies resolution stats
// returns zero count and nil average when no bugs are in terminal status.
func TestBugRepository_GetResolutionStats_NoTerminalBugs(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Clean all bugs first to create an isolated environment
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B8%'")

	// Create only open (non-terminal) bugs
	openBugs := []*models.Bug{
		newTestBug("B820", "Open reported bug", "reported", models.BugSeverityHigh),
		newTestBug("B821", "Open triaged bug", "triaged", models.BugSeverityMedium),
	}
	for _, b := range openBugs {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, b := range openBugs {
			_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
		}
	}()

	// Also remove any terminal B9xx bugs that might interfere
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B9%' AND status IN ('resolved','wont_fix','duplicate')")

	stats, err := repo.GetResolutionStats(ctx)
	if err != nil {
		t.Fatalf("GetResolutionStats() error = %v", err)
	}
	if stats == nil {
		t.Fatal("GetResolutionStats() returned nil stats")
	}
	// Since we cleared B8xx terminal bugs and cleaned B9xx terminal bugs,
	// resolved_count may be 0 if no other terminal bugs exist
	// The key assertion: AvgResolutionSecs should be nil if resolved_count is 0
	if stats.ResolvedCount == 0 && stats.AvgResolutionSecs != nil {
		t.Error("GetResolutionStats() should have nil AvgResolutionSecs when ResolvedCount is 0")
	}
}

// TestBugRepository_GetResolutionStats_WithTerminalBugs verifies resolution stats
// counts bugs in terminal statuses correctly.
func TestBugRepository_GetResolutionStats_WithTerminalBugs(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key IN ('B830','B831','B832','B833')")

	// Create a mix of terminal and open bugs
	bugs := []*models.Bug{
		newTestBug("B830", "Resolved bug 1", "resolved", models.BugSeverityHigh),
		newTestBug("B831", "Wont fix bug", "wont_fix", models.BugSeverityMedium),
		newTestBug("B832", "Duplicate bug", "duplicate", models.BugSeverityLow),
		newTestBug("B833", "Open bug", "reported", models.BugSeverityCritical),
	}
	for _, b := range bugs {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, b := range bugs {
			_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
		}
	}()

	stats, err := repo.GetResolutionStats(ctx)
	if err != nil {
		t.Fatalf("GetResolutionStats() error = %v", err)
	}

	// Should count bugs in resolved + wont_fix + duplicate (3 of our 4)
	if stats.ResolvedCount < 3 {
		t.Errorf("GetResolutionStats().ResolvedCount = %d, want at least 3", stats.ResolvedCount)
	}
	// AvgResolutionSecs should be non-nil since we have resolved bugs
	if stats.AvgResolutionSecs == nil {
		t.Error("GetResolutionStats().AvgResolutionSecs should not be nil when bugs exist in terminal status")
	}
	if stats.AvgResolutionSecs != nil && *stats.AvgResolutionSecs < 0 {
		t.Errorf("GetResolutionStats().AvgResolutionSecs = %v, want >= 0", *stats.AvgResolutionSecs)
	}
}

// TestBugRepository_GetFeatureBugSummary_NoLinkedBugs verifies feature bug summary
// returns zero counts when no bugs are linked to the given feature.
func TestBugRepository_GetFeatureBugSummary_NoLinkedBugs(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugAggTestSetup(t)
	defer cleanup()

	summary, err := repo.GetFeatureBugSummary(ctx, "E99-F99-nonexistent")
	if err != nil {
		t.Fatalf("GetFeatureBugSummary() error = %v", err)
	}
	if summary == nil {
		t.Fatal("GetFeatureBugSummary() returned nil summary")
	}
	if summary.TotalLinked != 0 {
		t.Errorf("GetFeatureBugSummary().TotalLinked = %d, want 0", summary.TotalLinked)
	}
	if summary.OpenCount != 0 {
		t.Errorf("GetFeatureBugSummary().OpenCount = %d, want 0", summary.OpenCount)
	}
	if summary.OpenBySeverity == nil {
		t.Error("GetFeatureBugSummary().OpenBySeverity should not be nil")
	}
}

// TestBugRepository_GetFeatureBugSummary_WithLinkedBugs verifies feature bug summary
// correctly counts linked bugs and separates open from terminal.
func TestBugRepository_GetFeatureBugSummary_WithLinkedBugs(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key IN ('B840','B841','B842','B843')")

	featureKey := "E07-F01"
	otherFeatureKey := "E07-F02"

	linkedType := "feature"
	featureKeyStr := featureKey
	otherFeatureKeyStr := otherFeatureKey

	bugs := []*models.Bug{
		// Linked to E07-F01 (open)
		{
			Key:              "B840",
			Title:            "Open high bug linked to feature",
			Status:           models.BugStatus("reported"),
			Severity:         models.BugSeverityHigh,
			LinkedEntityType: &linkedType,
			LinkedEntityKey:  &featureKeyStr,
		},
		// Linked to E07-F01 (open medium)
		{
			Key:              "B841",
			Title:            "Open medium bug linked to feature",
			Status:           models.BugStatus("in_fix"),
			Severity:         models.BugSeverityMedium,
			LinkedEntityType: &linkedType,
			LinkedEntityKey:  &featureKeyStr,
		},
		// Linked to E07-F01 (terminal -- should count in total but not open)
		{
			Key:              "B842",
			Title:            "Resolved bug linked to feature",
			Status:           models.BugStatus("resolved"),
			Severity:         models.BugSeverityLow,
			LinkedEntityType: &linkedType,
			LinkedEntityKey:  &featureKeyStr,
		},
		// Linked to different feature -- should NOT be counted
		{
			Key:              "B843",
			Title:            "Bug linked to different feature",
			Status:           models.BugStatus("reported"),
			Severity:         models.BugSeverityCritical,
			LinkedEntityType: &linkedType,
			LinkedEntityKey:  &otherFeatureKeyStr,
		},
	}

	for _, b := range bugs {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, b := range bugs {
			_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
		}
	}()

	summary, err := repo.GetFeatureBugSummary(ctx, featureKey)
	if err != nil {
		t.Fatalf("GetFeatureBugSummary(%q) error = %v", featureKey, err)
	}

	// Should count all 3 bugs linked to E07-F01 (B840, B841, B842)
	if summary.TotalLinked != 3 {
		t.Errorf("GetFeatureBugSummary().TotalLinked = %d, want 3", summary.TotalLinked)
	}

	// Only 2 are open (B840 and B841), B842 is resolved
	if summary.OpenCount != 2 {
		t.Errorf("GetFeatureBugSummary().OpenCount = %d, want 2", summary.OpenCount)
	}

	// OpenBySeverity should reflect open bugs only
	if summary.OpenBySeverity["high"] != 1 {
		t.Errorf("GetFeatureBugSummary().OpenBySeverity['high'] = %d, want 1", summary.OpenBySeverity["high"])
	}
	if summary.OpenBySeverity["medium"] != 1 {
		t.Errorf("GetFeatureBugSummary().OpenBySeverity['medium'] = %d, want 1", summary.OpenBySeverity["medium"])
	}
	// low severity bug (B842) is resolved, should not appear in open severity
	if summary.OpenBySeverity["low"] != 0 {
		t.Errorf("GetFeatureBugSummary().OpenBySeverity['low'] = %d, want 0 (terminal bug excluded)", summary.OpenBySeverity["low"])
	}
	// critical bug (B843) is for a different feature, should not appear
	if summary.OpenBySeverity["critical"] != 0 {
		t.Errorf("GetFeatureBugSummary().OpenBySeverity['critical'] = %d, want 0 (different feature)", summary.OpenBySeverity["critical"])
	}
}
