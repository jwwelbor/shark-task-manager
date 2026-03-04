package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

func bugTestSetup(t *testing.T) (*BugRepository, func()) {
	t.Helper()
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Clean up existing test bugs before test
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B9%'")

	cleanup := func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B9%'")
	}

	return repo, cleanup
}

func newTestBug(key, title, status string, severity models.BugSeverity) *models.Bug {
	return &models.Bug{
		Key:      key,
		Title:    title,
		Status:   models.BugStatus(status),
		Severity: severity,
	}
}

func TestBugRepository_Create(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	bug := newTestBug("B901", "Test bug for create", "reported", models.BugSeverityHigh)

	err := repo.Create(ctx, bug)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if bug.ID == 0 {
		t.Error("expected bug.ID to be set after Create()")
	}
}

func TestBugRepository_Create_ValidationFailure(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	// Bug with empty title should fail validation
	bug := &models.Bug{
		Key:      "B902",
		Title:    "",
		Status:   models.BugStatus("reported"),
		Severity: models.BugSeverityHigh,
	}

	err := repo.Create(ctx, bug)
	if err == nil {
		t.Fatal("Create() should have returned error for empty title")
	}
}

func TestBugRepository_Create_DuplicateKey(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	bug1 := newTestBug("B903", "First bug", "reported", models.BugSeverityHigh)
	if err := repo.Create(ctx, bug1); err != nil {
		t.Fatalf("Create() first bug error = %v", err)
	}

	bug2 := newTestBug("B903", "Duplicate key bug", "reported", models.BugSeverityMedium)
	err := repo.Create(ctx, bug2)
	if err == nil {
		t.Fatal("Create() should have returned error for duplicate key")
	}
	if !strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
		t.Errorf("expected UNIQUE constraint error, got: %v", err)
	}
}

func TestBugRepository_GetByKey(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	// Create a bug first
	original := newTestBug("B910", "Bug for GetByKey test", "reported", models.BugSeverityCritical)
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("exact key match", func(t *testing.T) {
		bug, err := repo.GetByKey(ctx, "B910")
		if err != nil {
			t.Fatalf("GetByKey() error = %v", err)
		}
		if bug.Key != "B910" {
			t.Errorf("GetByKey() key = %q, want %q", bug.Key, "B910")
		}
		if bug.Title != "Bug for GetByKey test" {
			t.Errorf("GetByKey() title = %q, want %q", bug.Title, "Bug for GetByKey test")
		}
		if bug.Severity != models.BugSeverityCritical {
			t.Errorf("GetByKey() severity = %q, want %q", bug.Severity, models.BugSeverityCritical)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		bug, err := repo.GetByKey(ctx, "b910")
		if err != nil {
			t.Fatalf("GetByKey(lowercase) error = %v", err)
		}
		if bug.Key != "B910" {
			t.Errorf("GetByKey(lowercase) key = %q, want %q", bug.Key, "B910")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByKey(ctx, "B999")
		if err == nil {
			t.Fatal("GetByKey() should have returned error for non-existent key")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("GetByKey() error = %q, expected 'not found'", err.Error())
		}
	})
}

func TestBugRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	original := newTestBug("B911", "Bug for GetByID test", "reported", models.BugSeverityMedium)
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("found by ID", func(t *testing.T) {
		bug, err := repo.GetByID(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if bug.ID != original.ID {
			t.Errorf("GetByID() ID = %d, want %d", bug.ID, original.ID)
		}
		if bug.Key != "B911" {
			t.Errorf("GetByID() key = %q, want %q", bug.Key, "B911")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 999999)
		if err == nil {
			t.Fatal("GetByID() should have returned error for non-existent ID")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("GetByID() error = %q, expected 'not found'", err.Error())
		}
	})
}

func TestBugRepository_Update(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	bug := newTestBug("B920", "Original title", "reported", models.BugSeverityLow)
	if err := repo.Create(ctx, bug); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	bug.Title = "Updated title"
	bug.Severity = models.BugSeverityCritical

	if err := repo.Update(ctx, bug); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update persisted
	updated, err := repo.GetByKey(ctx, "B920")
	if err != nil {
		t.Fatalf("GetByKey() after Update() error = %v", err)
	}
	if updated.Title != "Updated title" {
		t.Errorf("Update() title = %q, want %q", updated.Title, "Updated title")
	}
	if updated.Severity != models.BugSeverityCritical {
		t.Errorf("Update() severity = %q, want %q", updated.Severity, models.BugSeverityCritical)
	}
}

func TestBugRepository_Update_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	bug := newTestBug("B921", "Non-existent bug", "reported", models.BugSeverityHigh)
	bug.ID = 999999 // Non-existent ID

	err := repo.Update(ctx, bug)
	if err == nil {
		t.Fatal("Update() should have returned error for non-existent ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Update() error = %q, expected 'not found'", err.Error())
	}
}

func TestBugRepository_Delete(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	bug := newTestBug("B930", "Bug to delete", "reported", models.BugSeverityMedium)
	if err := repo.Create(ctx, bug); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, bug.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err := repo.GetByKey(ctx, "B930")
	if err == nil {
		t.Fatal("GetByKey() should have returned error for deleted bug")
	}
}

func TestBugRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	err := repo.Delete(ctx, 999999)
	if err == nil {
		t.Fatal("Delete() should have returned error for non-existent ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Delete() error = %q, expected 'not found'", err.Error())
	}
}

func TestBugRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	bug := newTestBug("B940", "Bug to update status", "reported", models.BugSeverityHigh)
	if err := repo.Create(ctx, bug); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	newStatus := models.BugStatus("triaged")
	if err := repo.UpdateStatus(ctx, bug.ID, newStatus); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	// Verify status changed
	updated, err := repo.GetByKey(ctx, "B940")
	if err != nil {
		t.Fatalf("GetByKey() after UpdateStatus() error = %v", err)
	}
	if updated.Status != newStatus {
		t.Errorf("UpdateStatus() status = %q, want %q", updated.Status, newStatus)
	}
}

func TestBugRepository_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	err := repo.UpdateStatus(ctx, 999999, models.BugStatus("triaged"))
	if err == nil {
		t.Fatal("UpdateStatus() should have returned error for non-existent ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("UpdateStatus() error = %q, expected 'not found'", err.Error())
	}
}

func TestBugRepository_GetNextKey(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Clean all test bugs to have a controlled state
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B9%'")

	// Get next key before any bugs with B9xx exist
	key1, err := repo.GetNextKey(ctx)
	if err != nil {
		t.Fatalf("GetNextKey() error = %v", err)
	}
	if len(key1) != 4 || key1[0] != 'B' {
		t.Errorf("GetNextKey() = %q, expected format B###", key1)
	}

	// Create a bug and verify next key increments
	bug := newTestBug(key1, "Bug for GetNextKey test", "reported", models.BugSeverityLow)
	if err := repo.Create(ctx, bug); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", bug.ID)

	key2, err := repo.GetNextKey(ctx)
	if err != nil {
		t.Fatalf("GetNextKey() second call error = %v", err)
	}
	if key2 == key1 {
		t.Errorf("GetNextKey() should return different key after creating bug, got same key %q", key2)
	}
}

func TestBugRepository_List(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Clean up test bugs
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key IN ('B950','B951','B952')")

	// Create test bugs with different statuses/severities
	bugs := []*models.Bug{
		newTestBug("B950", "Critical reported bug", "reported", models.BugSeverityCritical),
		newTestBug("B951", "High triaged bug", "triaged", models.BugSeverityHigh),
		newTestBug("B952", "Low reported bug", "reported", models.BugSeverityLow),
	}

	for _, b := range bugs {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, b := range bugs {
			database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
		}
	}()

	t.Run("list all (no filter)", func(t *testing.T) {
		results, err := repo.List(ctx, nil)
		if err != nil {
			t.Fatalf("List(nil) error = %v", err)
		}
		// Should have at least our 3 bugs
		if len(results) < 3 {
			t.Errorf("List() returned %d bugs, want at least 3", len(results))
		}
	})

	t.Run("filter by status reported", func(t *testing.T) {
		status := models.BugStatus("reported")
		filters := &BugListFilters{Status: &status}
		results, err := repo.List(ctx, filters)
		if err != nil {
			t.Fatalf("List(status=reported) error = %v", err)
		}
		for _, b := range results {
			if b.Status != "reported" {
				t.Errorf("List(status=reported) returned bug with status %q", b.Status)
			}
		}
		// Should include both B950 and B952
		if len(results) < 2 {
			t.Errorf("List(status=reported) returned %d bugs, want at least 2", len(results))
		}
	})

	t.Run("filter by severity critical", func(t *testing.T) {
		severity := models.BugSeverityCritical
		filters := &BugListFilters{Severity: &severity}
		results, err := repo.List(ctx, filters)
		if err != nil {
			t.Fatalf("List(severity=critical) error = %v", err)
		}
		for _, b := range results {
			if b.Severity != models.BugSeverityCritical {
				t.Errorf("List(severity=critical) returned bug with severity %q", b.Severity)
			}
		}
		// Should include B950
		if len(results) < 1 {
			t.Errorf("List(severity=critical) returned 0 bugs, want at least 1")
		}
	})

	t.Run("filter by status and severity", func(t *testing.T) {
		status := models.BugStatus("reported")
		severity := models.BugSeverityCritical
		filters := &BugListFilters{Status: &status, Severity: &severity}
		results, err := repo.List(ctx, filters)
		if err != nil {
			t.Fatalf("List(status=reported,severity=critical) error = %v", err)
		}
		for _, b := range results {
			if b.Status != "reported" || b.Severity != models.BugSeverityCritical {
				t.Errorf("List() returned bug with unexpected status=%q severity=%q", b.Status, b.Severity)
			}
		}
	})

	t.Run("empty filter object", func(t *testing.T) {
		results, err := repo.List(ctx, &BugListFilters{})
		if err != nil {
			t.Fatalf("List(&BugListFilters{}) error = %v", err)
		}
		if len(results) < 3 {
			t.Errorf("List(&BugListFilters{}) returned %d bugs, want at least 3", len(results))
		}
	})
}

func TestBugRepository_CountByStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Clean up and create known set
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key IN ('B960','B961','B962')")

	bugs := []*models.Bug{
		newTestBug("B960", "Status count bug 1", "reported", models.BugSeverityHigh),
		newTestBug("B961", "Status count bug 2", "reported", models.BugSeverityMedium),
		newTestBug("B962", "Status count bug 3", "triaged", models.BugSeverityLow),
	}
	for _, b := range bugs {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, b := range bugs {
			database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
		}
	}()

	counts, err := repo.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("CountByStatus() error = %v", err)
	}

	if counts["reported"] < 2 {
		t.Errorf("CountByStatus()['reported'] = %d, want at least 2", counts["reported"])
	}
	if counts["triaged"] < 1 {
		t.Errorf("CountByStatus()['triaged'] = %d, want at least 1", counts["triaged"])
	}
}

func TestBugRepository_CountBySeverity(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewBugRepository(db)

	// Clean up and create known set
	_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key IN ('B970','B971','B972')")

	bugs := []*models.Bug{
		newTestBug("B970", "Severity count bug 1", "reported", models.BugSeverityCritical),
		newTestBug("B971", "Severity count bug 2", "reported", models.BugSeverityCritical),
		newTestBug("B972", "Severity count bug 3", "reported", models.BugSeverityLow),
	}
	for _, b := range bugs {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, b := range bugs {
			database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", b.ID)
		}
	}()

	counts, err := repo.CountBySeverity(ctx)
	if err != nil {
		t.Fatalf("CountBySeverity() error = %v", err)
	}

	if counts["critical"] < 2 {
		t.Errorf("CountBySeverity()['critical'] = %d, want at least 2", counts["critical"])
	}
	if counts["low"] < 1 {
		t.Errorf("CountBySeverity()['low'] = %d, want at least 1", counts["low"])
	}
}

func TestBugRepository_NullableFields(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := bugTestSetup(t)
	defer cleanup()

	description := "Detailed description of the bug"
	linkedType := "feature"
	linkedKey := "E07-F01"

	bug := &models.Bug{
		Key:              "B980",
		Title:            "Bug with nullable fields",
		Status:           models.BugStatus("reported"),
		Severity:         models.BugSeverityHigh,
		Description:      &description,
		LinkedEntityType: &linkedType,
		LinkedEntityKey:  &linkedKey,
	}

	if err := repo.Create(ctx, bug); err != nil {
		t.Fatalf("Create() with nullable fields error = %v", err)
	}

	retrieved, err := repo.GetByKey(ctx, "B980")
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}

	if retrieved.Description == nil || *retrieved.Description != description {
		t.Errorf("Description = %v, want %q", retrieved.Description, description)
	}
	if retrieved.LinkedEntityType == nil || *retrieved.LinkedEntityType != linkedType {
		t.Errorf("LinkedEntityType = %v, want %q", retrieved.LinkedEntityType, linkedType)
	}
	if retrieved.LinkedEntityKey == nil || *retrieved.LinkedEntityKey != linkedKey {
		t.Errorf("LinkedEntityKey = %v, want %q", retrieved.LinkedEntityKey, linkedKey)
	}
}
