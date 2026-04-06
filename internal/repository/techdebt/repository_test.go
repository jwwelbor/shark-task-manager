package techdebt

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

func techDebtTestSetup(t *testing.T) (*TechDebtRepository, func()) {
	t.Helper()
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	// Clean up existing test tech-debt items before test
	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key LIKE 'TD-9%'")

	cleanup := func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key LIKE 'TD-9%'")
	}

	return repo, cleanup
}

func newTestTechDebt(key, title, status string, category models.TechDebtCategory, severity models.TechDebtSeverity) *models.TechDebt {
	return &models.TechDebt{
		BaseEntity: models.BaseEntity{Key: key, Title: title},
		Status:     models.TechDebtStatus(status),
		Category:   category,
		Severity:   severity,
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	td := newTestTechDebt("TD-901", "Test tech debt for create", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityHigh)

	err := repo.Create(ctx, td)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if td.ID == 0 {
		t.Error("expected td.ID to be set after Create()")
	}

	// Verify slug was auto-generated
	if td.Slug == nil || *td.Slug == "" {
		t.Error("expected slug to be auto-generated")
	}
}

func TestCreate_DuplicateKey(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	td1 := newTestTechDebt("TD-902", "First tech debt", "identified", models.TechDebtCategoryArchitecture, models.TechDebtSeverityMedium)
	if err := repo.Create(ctx, td1); err != nil {
		t.Fatalf("Create() first tech-debt error = %v", err)
	}

	td2 := newTestTechDebt("TD-902", "Duplicate key tech debt", "identified", models.TechDebtCategoryTesting, models.TechDebtSeverityLow)
	err := repo.Create(ctx, td2)
	if err == nil {
		t.Fatal("Create() should have returned error for duplicate key")
	}
	if !strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
		t.Errorf("expected UNIQUE constraint error, got: %v", err)
	}
}

func TestGetByKey(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	original := newTestTechDebt("TD-910", "Tech debt for GetByKey test", "identified", models.TechDebtCategoryPerformance, models.TechDebtSeverityCritical)
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("exact key match", func(t *testing.T) {
		td, err := repo.GetByKey(ctx, "TD-910")
		if err != nil {
			t.Fatalf("GetByKey() error = %v", err)
		}
		if td.Key != "TD-910" {
			t.Errorf("GetByKey() key = %q, want %q", td.Key, "TD-910")
		}
		if td.Title != "Tech debt for GetByKey test" {
			t.Errorf("GetByKey() title = %q, want %q", td.Title, "Tech debt for GetByKey test")
		}
		if td.Category != models.TechDebtCategoryPerformance {
			t.Errorf("GetByKey() category = %q, want %q", td.Category, models.TechDebtCategoryPerformance)
		}
		if td.Severity != models.TechDebtSeverityCritical {
			t.Errorf("GetByKey() severity = %q, want %q", td.Severity, models.TechDebtSeverityCritical)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByKey(ctx, "TD-999")
		if err == nil {
			t.Fatal("GetByKey() should have returned error for non-existent key")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("GetByKey() error = %q, expected 'not found'", err.Error())
		}
	})
}

func TestGetByKey_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	original := newTestTechDebt("TD-911", "Case insensitive test", "identified", models.TechDebtCategoryDependency, models.TechDebtSeverityMedium)
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	td, err := repo.GetByKey(ctx, "td-911")
	if err != nil {
		t.Fatalf("GetByKey(lowercase) error = %v", err)
	}
	if td.Key != "TD-911" {
		t.Errorf("GetByKey(lowercase) key = %q, want %q", td.Key, "TD-911")
	}
}

func TestGetByKey_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	_, err := repo.GetByKey(ctx, "TD-998")
	if err == nil {
		t.Fatal("GetByKey() should have returned error for non-existent key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("GetByKey() error = %q, expected 'not found'", err.Error())
	}
}

func TestGetByID(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	original := newTestTechDebt("TD-912", "Tech debt for GetByID test", "identified", models.TechDebtCategoryDocumentation, models.TechDebtSeverityLow)
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("found by ID", func(t *testing.T) {
		td, err := repo.GetByID(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if td.ID != original.ID {
			t.Errorf("GetByID() ID = %d, want %d", td.ID, original.ID)
		}
		if td.Key != "TD-912" {
			t.Errorf("GetByID() key = %q, want %q", td.Key, "TD-912")
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

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	effort := "medium"
	td := newTestTechDebt("TD-920", "Original title", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityLow)
	td.EffortEstimate = &effort
	if err := repo.Create(ctx, td); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	td.Title = "Updated title"
	td.Category = models.TechDebtCategoryArchitecture
	td.Severity = models.TechDebtSeverityCritical
	newEffort := "large"
	td.EffortEstimate = &newEffort

	if err := repo.Update(ctx, td); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update persisted
	updated, err := repo.GetByKey(ctx, "TD-920")
	if err != nil {
		t.Fatalf("GetByKey() after Update() error = %v", err)
	}
	if updated.Title != "Updated title" {
		t.Errorf("Update() title = %q, want %q", updated.Title, "Updated title")
	}
	if updated.Category != models.TechDebtCategoryArchitecture {
		t.Errorf("Update() category = %q, want %q", updated.Category, models.TechDebtCategoryArchitecture)
	}
	if updated.Severity != models.TechDebtSeverityCritical {
		t.Errorf("Update() severity = %q, want %q", updated.Severity, models.TechDebtSeverityCritical)
	}
	if updated.EffortEstimate == nil || *updated.EffortEstimate != "large" {
		t.Errorf("Update() effort_estimate = %v, want %q", updated.EffortEstimate, "large")
	}
}

func TestUpdateStatus(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	td := newTestTechDebt("TD-930", "Tech debt to update status", "identified", models.TechDebtCategoryTesting, models.TechDebtSeverityHigh)
	if err := repo.Create(ctx, td); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	newStatus := models.TechDebtStatus("triaged")
	if err := repo.UpdateStatus(ctx, td.ID, newStatus); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	// Verify status changed
	updated, err := repo.GetByKey(ctx, "TD-930")
	if err != nil {
		t.Fatalf("GetByKey() after UpdateStatus() error = %v", err)
	}
	if updated.Status != newStatus {
		t.Errorf("UpdateStatus() status = %q, want %q", updated.Status, newStatus)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	repo, cleanup := techDebtTestSetup(t)
	defer cleanup()

	td := newTestTechDebt("TD-940", "Tech debt to delete", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityMedium)
	if err := repo.Create(ctx, td); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, td.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err := repo.GetByKey(ctx, "TD-940")
	if err == nil {
		t.Fatal("GetByKey() should have returned error for deleted tech-debt")
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	// Clean up test tech-debts
	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key IN ('TD-950','TD-951','TD-952')")

	items := []*models.TechDebt{
		newTestTechDebt("TD-950", "First item", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityCritical),
		newTestTechDebt("TD-951", "Second item", "triaged", models.TechDebtCategoryArchitecture, models.TechDebtSeverityHigh),
		newTestTechDebt("TD-952", "Third item", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityLow),
	}

	for _, td := range items {
		if err := repo.Create(ctx, td); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, td := range items {
			_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE id = ?", td.ID)
		}
	}()

	results, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(results) < 3 {
		t.Errorf("List() returned %d items, want at least 3", len(results))
	}

	// Verify ordering by key ascending
	for i := 1; i < len(results); i++ {
		if results[i].Key < results[i-1].Key {
			t.Errorf("List() not ordered by key ascending: %q < %q", results[i].Key, results[i-1].Key)
			break
		}
	}
}

func TestListWithFilters_Category(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key IN ('TD-960','TD-961','TD-962')")

	items := []*models.TechDebt{
		newTestTechDebt("TD-960", "Code quality item", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityMedium),
		newTestTechDebt("TD-961", "Architecture item", "identified", models.TechDebtCategoryArchitecture, models.TechDebtSeverityHigh),
		newTestTechDebt("TD-962", "Another code quality", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityLow),
	}

	for _, td := range items {
		if err := repo.Create(ctx, td); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, td := range items {
			_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE id = ?", td.ID)
		}
	}()

	cat := models.TechDebtCategoryCodeQuality
	results, err := repo.ListWithFilters(ctx, TechDebtFilters{Category: &cat})
	if err != nil {
		t.Fatalf("ListWithFilters(category) error = %v", err)
	}

	for _, td := range results {
		if td.Category != models.TechDebtCategoryCodeQuality {
			t.Errorf("ListWithFilters(category) returned item with category %q", td.Category)
		}
	}
	if len(results) < 2 {
		t.Errorf("ListWithFilters(category) returned %d items, want at least 2", len(results))
	}
}

func TestListWithFilters_Severity(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key IN ('TD-963','TD-964','TD-965')")

	items := []*models.TechDebt{
		newTestTechDebt("TD-963", "Critical item", "identified", models.TechDebtCategoryTesting, models.TechDebtSeverityCritical),
		newTestTechDebt("TD-964", "High item", "identified", models.TechDebtCategoryTesting, models.TechDebtSeverityHigh),
		newTestTechDebt("TD-965", "Another critical", "identified", models.TechDebtCategoryTesting, models.TechDebtSeverityCritical),
	}

	for _, td := range items {
		if err := repo.Create(ctx, td); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, td := range items {
			_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE id = ?", td.ID)
		}
	}()

	sev := models.TechDebtSeverityCritical
	results, err := repo.ListWithFilters(ctx, TechDebtFilters{Severity: &sev})
	if err != nil {
		t.Fatalf("ListWithFilters(severity) error = %v", err)
	}

	for _, td := range results {
		if td.Severity != models.TechDebtSeverityCritical {
			t.Errorf("ListWithFilters(severity) returned item with severity %q", td.Severity)
		}
	}
	if len(results) < 2 {
		t.Errorf("ListWithFilters(severity) returned %d items, want at least 2", len(results))
	}
}

func TestListWithFilters_Status(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key IN ('TD-966','TD-967','TD-968')")

	items := []*models.TechDebt{
		newTestTechDebt("TD-966", "Identified item", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityMedium),
		newTestTechDebt("TD-967", "Triaged item", "triaged", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityMedium),
		newTestTechDebt("TD-968", "Another identified", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityMedium),
	}

	for _, td := range items {
		if err := repo.Create(ctx, td); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, td := range items {
			_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE id = ?", td.ID)
		}
	}()

	status := "identified"
	results, err := repo.ListWithFilters(ctx, TechDebtFilters{Status: &status})
	if err != nil {
		t.Fatalf("ListWithFilters(status) error = %v", err)
	}

	for _, td := range results {
		if string(td.Status) != "identified" {
			t.Errorf("ListWithFilters(status) returned item with status %q", td.Status)
		}
	}
	if len(results) < 2 {
		t.Errorf("ListWithFilters(status) returned %d items, want at least 2", len(results))
	}
}

func TestListWithFilters_Combined(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key IN ('TD-970','TD-971','TD-972')")

	items := []*models.TechDebt{
		newTestTechDebt("TD-970", "Match all filters", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityCritical),
		newTestTechDebt("TD-971", "Wrong category", "identified", models.TechDebtCategoryArchitecture, models.TechDebtSeverityCritical),
		newTestTechDebt("TD-972", "Wrong severity", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityLow),
	}

	for _, td := range items {
		if err := repo.Create(ctx, td); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, td := range items {
			_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE id = ?", td.ID)
		}
	}()

	status := "identified"
	cat := models.TechDebtCategoryCodeQuality
	sev := models.TechDebtSeverityCritical
	results, err := repo.ListWithFilters(ctx, TechDebtFilters{
		Status:   &status,
		Category: &cat,
		Severity: &sev,
	})
	if err != nil {
		t.Fatalf("ListWithFilters(combined) error = %v", err)
	}

	// Should match TD-970 only (among our test data)
	found := false
	for _, td := range results {
		if td.Key == "TD-970" {
			found = true
		}
		if td.Category != models.TechDebtCategoryCodeQuality {
			t.Errorf("ListWithFilters(combined) returned item with wrong category %q", td.Category)
		}
		if td.Severity != models.TechDebtSeverityCritical {
			t.Errorf("ListWithFilters(combined) returned item with wrong severity %q", td.Severity)
		}
	}
	if !found {
		t.Error("ListWithFilters(combined) did not return TD-970")
	}
}

func TestGenerateNextKey(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	// Clean all test tech-debts to have controlled state
	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key LIKE 'TD-9%'")

	// Get next key
	key1, err := repo.GenerateNextKey(ctx)
	if err != nil {
		t.Fatalf("GenerateNextKey() error = %v", err)
	}
	if len(key1) != 6 || !strings.HasPrefix(key1, "TD-") {
		t.Errorf("GenerateNextKey() = %q, expected format TD-###", key1)
	}

	// Create a tech-debt item and verify next key increments
	td := newTestTechDebt(key1, "Tech debt for GenerateNextKey test", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityLow)
	if err := repo.Create(ctx, td); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE id = ?", td.ID)
	}()

	key2, err := repo.GenerateNextKey(ctx)
	if err != nil {
		t.Fatalf("GenerateNextKey() second call error = %v", err)
	}
	if key2 == key1 {
		t.Errorf("GenerateNextKey() should return different key after creating tech-debt, got same key %q", key2)
	}
}

func TestCountByStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key IN ('TD-980','TD-981','TD-982')")

	items := []*models.TechDebt{
		newTestTechDebt("TD-980", "Status count 1", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityHigh),
		newTestTechDebt("TD-981", "Status count 2", "identified", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityMedium),
		newTestTechDebt("TD-982", "Status count 3", "triaged", models.TechDebtCategoryCodeQuality, models.TechDebtSeverityLow),
	}
	for _, td := range items {
		if err := repo.Create(ctx, td); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, td := range items {
			_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE id = ?", td.ID)
		}
	}()

	counts, err := repo.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("CountByStatus() error = %v", err)
	}

	if counts["identified"] < 2 {
		t.Errorf("CountByStatus()['identified'] = %d, want at least 2", counts["identified"])
	}
	if counts["triaged"] < 1 {
		t.Errorf("CountByStatus()['triaged'] = %d, want at least 1", counts["triaged"])
	}
}

func TestCountByCategory(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTechDebtRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key IN ('TD-985','TD-986','TD-987')")

	items := []*models.TechDebt{
		newTestTechDebt("TD-985", "Category count 1", "identified", models.TechDebtCategoryArchitecture, models.TechDebtSeverityHigh),
		newTestTechDebt("TD-986", "Category count 2", "identified", models.TechDebtCategoryArchitecture, models.TechDebtSeverityMedium),
		newTestTechDebt("TD-987", "Category count 3", "identified", models.TechDebtCategoryTesting, models.TechDebtSeverityLow),
	}
	for _, td := range items {
		if err := repo.Create(ctx, td); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		for _, td := range items {
			_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE id = ?", td.ID)
		}
	}()

	counts, err := repo.CountByCategory(ctx)
	if err != nil {
		t.Fatalf("CountByCategory() error = %v", err)
	}

	if counts["architecture"] < 2 {
		t.Errorf("CountByCategory()['architecture'] = %d, want at least 2", counts["architecture"])
	}
	if counts["testing"] < 1 {
		t.Errorf("CountByCategory()['testing'] = %d, want at least 1", counts["testing"])
	}
}

func TestDatabaseConstraints_CategoryCheck(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)

	// Insert directly to bypass model validation and test DB CHECK constraint
	query := `INSERT INTO tech_debts (key, title, status, category, severity) VALUES (?, ?, ?, ?, ?)`
	_, err := database.ExecContext(ctx, query, "TD-990", "Invalid category test", "identified", "invalid-category", "medium")
	if err == nil {
		// Clean up if it somehow succeeded
		_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key = 'TD-990'")
		t.Fatal("expected CHECK constraint error for invalid category")
	}

	_ = db // use db to avoid unused variable
}

func TestDatabaseConstraints_SeverityCheck(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)

	// Insert directly to bypass model validation and test DB CHECK constraint
	query := `INSERT INTO tech_debts (key, title, status, category, severity) VALUES (?, ?, ?, ?, ?)`
	_, err := database.ExecContext(ctx, query, "TD-991", "Invalid severity test", "identified", "code-quality", "invalid-severity")
	if err == nil {
		// Clean up if it somehow succeeded
		_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key = 'TD-991'")
		t.Fatal("expected CHECK constraint error for invalid severity")
	}

	_ = db // use db to avoid unused variable
}
