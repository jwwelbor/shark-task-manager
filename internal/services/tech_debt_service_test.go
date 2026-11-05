package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockTechDebtRepo implements TechDebtRepository for testing.
type mockTechDebtRepo struct {
	createFn          func(ctx context.Context, td *models.TechDebt) error
	getByKeyFn        func(ctx context.Context, key string) (*models.TechDebt, error)
	getByIDFn         func(ctx context.Context, id int64) (*models.TechDebt, error)
	updateFn          func(ctx context.Context, td *models.TechDebt) error
	deleteFn          func(ctx context.Context, id int64) error
	updateStatusFn    func(ctx context.Context, id int64, status models.TechDebtStatus) error
	generateNextKeyFn func(ctx context.Context) (string, error)
	listFn            func(ctx context.Context) ([]*models.TechDebt, error)
	listWithFiltersFn func(ctx context.Context, filters techdebt.TechDebtFilters) ([]*models.TechDebt, error)
	countByStatusFn   func(ctx context.Context) (map[string]int, error)
	countByCategoryFn func(ctx context.Context) (map[string]int, error)
}

func (m *mockTechDebtRepo) Create(ctx context.Context, td *models.TechDebt) error {
	if m.createFn != nil {
		return m.createFn(ctx, td)
	}
	return nil
}

func (m *mockTechDebtRepo) GetByKey(ctx context.Context, key string) (*models.TechDebt, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found: %s", key)
}

func (m *mockTechDebtRepo) GetByID(ctx context.Context, id int64) (*models.TechDebt, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, fmt.Errorf("not found: %d", id)
}

func (m *mockTechDebtRepo) Update(ctx context.Context, td *models.TechDebt) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, td)
	}
	return nil
}

func (m *mockTechDebtRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTechDebtRepo) UpdateStatus(ctx context.Context, id int64, status models.TechDebtStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *mockTechDebtRepo) GenerateNextKey(ctx context.Context) (string, error) {
	if m.generateNextKeyFn != nil {
		return m.generateNextKeyFn(ctx)
	}
	return "TD-001", nil
}

func (m *mockTechDebtRepo) List(ctx context.Context) ([]*models.TechDebt, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []*models.TechDebt{}, nil
}

func (m *mockTechDebtRepo) ListWithFilters(ctx context.Context, filters techdebt.TechDebtFilters) ([]*models.TechDebt, error) {
	if m.listWithFiltersFn != nil {
		return m.listWithFiltersFn(ctx, filters)
	}
	return []*models.TechDebt{}, nil
}

func (m *mockTechDebtRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx)
	}
	return map[string]int{}, nil
}

func (m *mockTechDebtRepo) CountByCategory(ctx context.Context) (map[string]int, error) {
	if m.countByCategoryFn != nil {
		return m.countByCategoryFn(ctx)
	}
	return map[string]int{}, nil
}

// mockTDEntityRepo is a minimal EntityRepository adapter for tech-debt tests.
type mockTDEntityRepo struct {
	tdRepo *mockTechDebtRepo
}

func (m *mockTDEntityRepo) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return m.tdRepo.GetByKey(ctx, key)
}

func (m *mockTDEntityRepo) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return m.tdRepo.GetByID(ctx, id)
}

func (m *mockTDEntityRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return m.tdRepo.UpdateStatus(ctx, id, models.TechDebtStatus(status))
}

func (m *mockTDEntityRepo) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error) {
	td, err := m.tdRepo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	if td == nil || !strings.EqualFold(string(td.Status), expectedCurrentStatus) {
		return false, nil
	}
	return true, m.tdRepo.UpdateStatus(ctx, id, models.TechDebtStatus(newStatus))
}

func (m *mockTDEntityRepo) Update(ctx context.Context, entity models.Entity) error {
	td, ok := entity.(*models.TechDebt)
	if !ok {
		return fmt.Errorf("expected *models.TechDebt, got %T", entity)
	}
	return m.tdRepo.Update(ctx, td)
}

func (m *mockTDEntityRepo) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}

func (m *mockTDEntityRepo) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return nil
}

func newTDWorkflowSvc() *workflow.Service {
	return workflow.NewService("")
}

func newTechDebtService(repo *mockTechDebtRepo) *TechDebtService {
	wfSvc := newTDWorkflowSvc()
	entitySvc := NewEntityService(wfSvc)
	entityRepo := &mockTDEntityRepo{tdRepo: repo}
	return NewTechDebtService(repo, entitySvc, entityRepo, "", nil)
}

// --- CreateTechDebt tests ---

func TestTechDebtService_Create_Success(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		generateNextKeyFn: func(_ context.Context) (string, error) {
			return "TD-001", nil
		},
		createFn: func(_ context.Context, td *models.TechDebt) error {
			if td.Key != "TD-001" {
				t.Errorf("expected key TD-001, got %s", td.Key)
			}
			if td.Title != "Refactor database layer" {
				t.Errorf("expected title 'Refactor database layer', got %s", td.Title)
			}
			if td.Category != models.TechDebtCategoryArchitecture {
				t.Errorf("expected category architecture, got %s", td.Category)
			}
			if td.Severity != models.TechDebtSeverityHigh {
				t.Errorf("expected severity high, got %s", td.Severity)
			}
			td.ID = 1
			return nil
		},
	}

	svc := newTechDebtService(repo)
	svc.projectRoot = t.TempDir()

	td, _, err := svc.CreateTechDebt(ctx, CreateTechDebtInput{
		Title:    "Refactor database layer",
		Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityHigh,
	})
	if err != nil {
		t.Fatalf("CreateTechDebt() error = %v", err)
	}
	if td == nil {
		t.Fatal("expected tech-debt, got nil")
	}
	if td.Key != "TD-001" {
		t.Errorf("expected key TD-001, got %s", td.Key)
	}
	if td.Slug == nil || *td.Slug == "" {
		t.Error("expected non-empty slug")
	}
}

func TestTechDebtService_Create_ValidationError(t *testing.T) {
	ctx := context.Background()
	svc := newTechDebtService(&mockTechDebtRepo{})

	_, _, err := svc.CreateTechDebt(ctx, CreateTechDebtInput{
		Title:    "",
		Category: models.TechDebtCategoryCodeQuality,
		Severity: models.TechDebtSeverityMedium,
	})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("expected error about title, got: %v", err)
	}
}

func TestTechDebtService_Create_InvalidCategory(t *testing.T) {
	ctx := context.Background()
	svc := newTechDebtService(&mockTechDebtRepo{})

	_, _, err := svc.CreateTechDebt(ctx, CreateTechDebtInput{
		Title:    "Some debt",
		Category: "invalid-category",
		Severity: models.TechDebtSeverityMedium,
	})
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
	if !strings.Contains(err.Error(), "invalid category") {
		t.Errorf("expected error about invalid category, got: %v", err)
	}
}

// Independent document fixtures keep collision and persistence assertions from
// accepting the same rendering mistake made by the production generator.
const b112Placeholder = `---
tech_debt_key: TD-001
title: Retry debt
category: architecture
severity: high
slug: retry-debt
---

# Retry debt

## Description

[Describe the technical debt and its impact]

## Impact

[What is the impact of this tech debt?]

## Resolution Plan

[How should this be resolved?]

## Resolution Notes

[Notes on the resolution once complete]
`

func writeTechDebtFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func assertTechDebtDocument(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read document: %v", err)
	}
	if string(got) != want {
		t.Errorf("document at %q = %q, want %q", path, got, want)
	}
}

func TestTechDebtService_Create_CollisionsPreserveDocumentsWithoutPersisting(t *testing.T) {
	for _, tc := range []struct {
		name, original, title string
		force                 bool
	}{
		{"substantive", "# Existing debt\n\nMaintainer notes.\n", "Retry debt", false},
		{"forced substantive", "# Existing debt\n\nMaintainer notes.\n", "Retry debt", true},
		{"matching placeholder requires force", b112Placeholder, "Retry debt", false},
		{"forced different placeholder", b112Placeholder, "Replacement debt", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
			writeTechDebtFixture(t, path, tc.original)
			createCalls, deleteCalls := 0, 0
			svc := newTechDebtService(&mockTechDebtRepo{
				createFn: func(context.Context, *models.TechDebt) error { createCalls++; return nil },
				deleteFn: func(context.Context, int64) error { deleteCalls++; return nil },
			})
			svc.projectRoot = root
			created, linked, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
				Title: tc.title, Category: models.TechDebtCategoryArchitecture,
				Severity: models.TechDebtSeverityHigh, Force: tc.force,
			})
			if err == nil || !strings.Contains(err.Error(), "collision") {
				t.Fatalf("CreateTechDebt() error = %v, want collision", err)
			}
			if created != nil || linked || createCalls != 0 || deleteCalls != 0 {
				t.Errorf("collision returned created=%v linked=%v, create/delete=%d/%d", created, linked, createCalls, deleteCalls)
			}
			assertTechDebtDocument(t, path, tc.original)
		})
	}
}

func TestTechDebtService_Create_ForceReusesExactDocumentWithoutWriting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
	writeTechDebtFixture(t, path, b112Placeholder)
	oldTime := time.Unix(100, 0)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatalf("set fixture timestamp: %v", err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat fixture: %v", err)
	}
	createCalls := 0
	svc := newTechDebtService(&mockTechDebtRepo{
		createFn: func(_ context.Context, td *models.TechDebt) error {
			createCalls++
			td.ID = 17
			return nil
		},
	})
	svc.projectRoot = root
	created, linked, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
		Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityHigh, Force: true,
	})
	if err != nil {
		t.Fatalf("CreateTechDebt() error = %v", err)
	}
	if created == nil || created.ID != 17 || !linked || createCalls != 1 {
		t.Fatalf("exact reuse returned created=%v linked=%v createCalls=%d", created, linked, createCalls)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat reused document: %v", err)
	}
	if !os.SameFile(before, after) || !after.ModTime().Equal(oldTime) {
		t.Error("exact-match reuse rewrote or replaced the existing document")
	}
	assertTechDebtDocument(t, path, b112Placeholder)
}

func TestTechDebtService_Create_WritesCompleteDocumentBeforePersistingPath(t *testing.T) {
	for _, pathKind := range []string{"default", "relative", "absolute"} {
		t.Run(pathKind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join("docs", "plan", "tech-debt", "TD-001.md")
			var inputPath *string
			if pathKind != "default" {
				path = filepath.Join("custom", "debt.md")
				if pathKind == "absolute" {
					path = filepath.Join(root, path)
				}
				inputPath = &path
			}
			fullPath := path
			if !filepath.IsAbs(fullPath) {
				fullPath = filepath.Join(root, fullPath)
			}
			createCalls := 0
			svc := newTechDebtService(&mockTechDebtRepo{
				createFn: func(_ context.Context, td *models.TechDebt) error {
					createCalls++
					if td.FilePath == nil || *td.FilePath != path {
						t.Errorf("persisted path = %v, want %q", td.FilePath, path)
					}
					assertTechDebtDocument(t, fullPath, b112Placeholder)
					return nil
				},
			})
			svc.projectRoot = root
			created, linked, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
				Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
				Severity: models.TechDebtSeverityHigh, FilePath: inputPath,
			})
			if err != nil || created == nil || linked || createCalls != 1 {
				t.Fatalf("create returned created=%v linked=%v calls=%d error=%v", created, linked, createCalls, err)
			}
			assertTechDebtDocument(t, fullPath, b112Placeholder)
		})
	}
}

func TestTechDebtService_Create_InvalidModelDoesNotCreateDocument(t *testing.T) {
	root := t.TempDir()
	createCalls := 0
	svc := newTechDebtService(&mockTechDebtRepo{
		createFn: func(_ context.Context, td *models.TechDebt) error {
			createCalls++
			return td.Validate()
		},
	})
	svc.projectRoot = root
	invalidSize := 4
	_, _, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
		Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityHigh, Size: &invalidSize,
	})
	if err == nil {
		t.Fatal("invalid model was accepted")
	}
	if createCalls != 0 {
		t.Errorf("invalid model reached repository Create %d times", createCalls)
	}
	path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("invalid input created a document at %q: stat error=%v", path, statErr)
	}
}

func TestTechDebtService_Create_ForceRequiresMatchingCompleteBody(t *testing.T) {
	const original = "---\ntech_debt_key: TD-001\ntitle: Retry debt\ncategory: architecture\nseverity: high\nslug: retry-debt\n---\nSAVED-CONTENT\n"
	for _, body := range []string{"SAVED-CONTENT", "DIFFERENT-CONTENT"} {
		t.Run(body, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
			writeTechDebtFixture(t, path, original)
			createCalls := 0
			svc := newTechDebtService(&mockTechDebtRepo{
				createFn: func(context.Context, *models.TechDebt) error { createCalls++; return nil },
			})
			svc.projectRoot = root
			_, linked, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
				Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
				Severity: models.TechDebtSeverityHigh, Body: body, Force: true,
			})
			if body == "SAVED-CONTENT" {
				if err != nil || !linked || createCalls != 1 {
					t.Errorf("matching body returned linked=%v createCalls=%d error=%v", linked, createCalls, err)
				}
			} else if err == nil || linked || createCalls != 0 {
				t.Errorf("different body returned linked=%v createCalls=%d error=%v", linked, createCalls, err)
			}
			assertTechDebtDocument(t, path, original)
		})
	}
}

func TestTechDebtService_Create_WriteFailureDoesNotCreateOrDeleteRecord(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "blocked")
	writeTechDebtFixture(t, blocker, "PRESERVE-PARENT-FILE")
	path := filepath.Join("blocked", "TD-001.md")
	createCalls, deleteCalls := 0, 0
	svc := newTechDebtService(&mockTechDebtRepo{
		createFn: func(context.Context, *models.TechDebt) error { createCalls++; return nil },
		deleteFn: func(context.Context, int64) error { deleteCalls++; return nil },
	})
	svc.projectRoot = root
	_, _, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
		Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityHigh, FilePath: &path,
	})
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("write error = %v, want failed path %q", err, path)
	}
	if createCalls != 0 || deleteCalls != 0 {
		t.Errorf("write failure called create/delete %d/%d times", createCalls, deleteCalls)
	}
	assertTechDebtDocument(t, blocker, "PRESERVE-PARENT-FILE")
}

func TestTechDebtService_Create_CanceledInsertRetainsDocumentForExactRetry(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	createCalls, deleteCalls := 0, 0
	svc := newTechDebtService(&mockTechDebtRepo{
		createFn: func(ctx context.Context, td *models.TechDebt) error {
			createCalls++
			assertTechDebtDocument(t, path, b112Placeholder)
			if createCalls == 1 {
				cancel()
				return ctx.Err()
			}
			td.ID = 23
			return nil
		},
		deleteFn: func(context.Context, int64) error { deleteCalls++; return fmt.Errorf("deletion unavailable") },
	})
	svc.projectRoot = root
	input := CreateTechDebtInput{
		Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityHigh,
	}
	_, _, err := svc.CreateTechDebt(ctx, input)
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "docs/plan/tech-debt/TD-001.md") || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("insert error = %v, want cancellation and retained-path retry guidance", err)
	}
	assertTechDebtDocument(t, path, b112Placeholder)
	input.Force = true
	created, linked, err := svc.CreateTechDebt(context.Background(), input)
	if err != nil || created == nil || created.ID != 23 || !linked {
		t.Fatalf("retry returned created=%v linked=%v error=%v", created, linked, err)
	}
	if createCalls != 2 || deleteCalls != 0 {
		t.Errorf("recovery called create/delete %d/%d times, want 2/0", createCalls, deleteCalls)
	}
	assertTechDebtDocument(t, path, b112Placeholder)
}

func TestTechDebtService_Create_ConcurrentCollisionPersistsOnlyDocumentOwner(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
	expected := map[string]string{
		"First debt":  "---\ntech_debt_key: TD-001\ntitle: First debt\ncategory: architecture\nseverity: high\nslug: first-debt\n---\nFIRST-BODY\n",
		"Second debt": "---\ntech_debt_key: TD-001\ntitle: Second debt\ncategory: architecture\nseverity: high\nslug: second-debt\n---\nSECOND-BODY\n",
	}
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	created := make(chan string, 2)
	deleted := make(chan int64, 2)
	svc := newTechDebtService(&mockTechDebtRepo{
		generateNextKeyFn: func(context.Context) (string, error) {
			ready.Done()
			<-start
			return "TD-001", nil
		},
		createFn: func(_ context.Context, td *models.TechDebt) error {
			got, err := os.ReadFile(path)
			if err != nil || string(got) != expected[td.Title] {
				t.Errorf("insert for %q preceded its complete document: bytes=%q error=%v", td.Title, got, err)
			}
			created <- td.Title
			return nil
		},
		deleteFn: func(_ context.Context, id int64) error { deleted <- id; return nil },
	})
	svc.projectRoot = root
	results := make(chan error, 2)
	for _, input := range []CreateTechDebtInput{
		{Title: "First debt", Category: models.TechDebtCategoryArchitecture, Severity: models.TechDebtSeverityHigh, Body: "FIRST-BODY", Force: true},
		{Title: "Second debt", Category: models.TechDebtCategoryArchitecture, Severity: models.TechDebtSeverityHigh, Body: "SECOND-BODY", Force: true},
	} {
		go func(input CreateTechDebtInput) {
			_, _, err := svc.CreateTechDebt(context.Background(), input)
			results <- err
		}(input)
	}
	ready.Wait()
	close(start)
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 || len(created) != 1 || len(deleted) != 0 {
		t.Fatalf("race successes/create/delete = %d/%d/%d, want 1/1/0", successes, len(created), len(deleted))
	}
	assertTechDebtDocument(t, path, expected[<-created])
}

func TestTechDebtService_Create_ContentKeepsSingleFrontmatterBlock(t *testing.T) {
	root := t.TempDir()
	svc := newTechDebtService(&mockTechDebtRepo{})
	svc.projectRoot = root
	_, _, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
		Title: "Body debt", Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityMedium, Body: "CUSTOM-CONTENT-BODY\n\n",
	})
	if err != nil {
		t.Fatalf("CreateTechDebt() error = %v", err)
	}
	assertTechDebtDocument(t, filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md"),
		"---\ntech_debt_key: TD-001\ntitle: Body debt\ncategory: architecture\nseverity: medium\nslug: body-debt\n---\nCUSTOM-CONTENT-BODY\n")
}

// --- GetTechDebt tests ---

func TestTechDebtService_Get_Success(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test Debt"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	td, err := svc.GetTechDebt(ctx, "TD-001")
	if err != nil {
		t.Fatalf("GetTechDebt() error = %v", err)
	}
	if td == nil {
		t.Fatal("expected tech-debt, got nil")
	}
	if td.Key != "TD-001" {
		t.Errorf("expected key TD-001, got %s", td.Key)
	}
}

func TestTechDebtService_Get_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return nil, fmt.Errorf("tech-debt not found with key %q", key)
		},
	}

	svc := newTechDebtService(repo)

	_, err := svc.GetTechDebt(ctx, "TD-999")
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !strings.Contains(err.Error(), "TD-999") {
		t.Errorf("expected error containing key, got: %v", err)
	}
}

// --- UpdateTechDebt tests ---

func TestTechDebtService_Update_Success(t *testing.T) {
	ctx := context.Background()

	var updatedTD *models.TechDebt

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Old Title"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}

	svc := newTechDebtService(repo)

	newTitle := "New Title"
	newSeverity := models.TechDebtSeverityHigh
	td, err := svc.UpdateTechDebt(ctx, "TD-001", TechDebtUpdates{
		Title:    &newTitle,
		Severity: &newSeverity,
	})
	if err != nil {
		t.Fatalf("UpdateTechDebt() error = %v", err)
	}
	if td.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %s", td.Title)
	}
	if updatedTD == nil {
		t.Fatal("expected update to be called")
	}
	if updatedTD.Severity != models.TechDebtSeverityHigh {
		t.Errorf("expected severity high, got %s", updatedTD.Severity)
	}
	// Category should remain unchanged
	if updatedTD.Category != models.TechDebtCategoryCodeQuality {
		t.Errorf("expected category unchanged (code-quality), got %s", updatedTD.Category)
	}
}

// --- Size on Create / Update ---

// TestTechDebtService_Create_WithSize verifies that a Size value passed in the
// CreateTechDebtInput is propagated to the persisted TechDebt model.
func TestTechDebtService_Create_WithSize(t *testing.T) {
	ctx := context.Background()

	var createdTD *models.TechDebt
	repo := &mockTechDebtRepo{
		generateNextKeyFn: func(_ context.Context) (string, error) { return "TD-001", nil },
		createFn: func(_ context.Context, td *models.TechDebt) error {
			createdTD = td
			td.ID = 1
			return nil
		},
	}

	svc := newTechDebtService(repo)
	svc.projectRoot = t.TempDir()
	five := 5
	td, _, err := svc.CreateTechDebt(ctx, CreateTechDebtInput{
		Title:    "Refactor auth",
		Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityMedium,
		Size:     &five,
	})
	if err != nil {
		t.Fatalf("CreateTechDebt() error = %v", err)
	}
	if td.Size == nil || *td.Size != 5 {
		t.Errorf("expected returned td.Size=5, got %v", td.Size)
	}
	if createdTD == nil || createdTD.Size == nil || *createdTD.Size != 5 {
		t.Errorf("expected persisted td.Size=5, got %v", createdTD)
	}
}

// TestTechDebtService_Update_SetSize verifies the three-branch dispatch:
// Size != nil → set; ClearSize=true → clear; both empty → no-op.
func TestTechDebtService_Update_SetSize(t *testing.T) {
	ctx := context.Background()
	existingSize := 3
	var updatedTD *models.TechDebt

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Old", Size: &existingSize},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}
	svc := newTechDebtService(repo)

	eight := 8
	td, err := svc.UpdateTechDebt(ctx, "TD-001", TechDebtUpdates{Size: &eight})
	if err != nil {
		t.Fatalf("UpdateTechDebt() error = %v", err)
	}
	if td.Size == nil || *td.Size != 8 {
		t.Errorf("expected td.Size=8 after update, got %v", td.Size)
	}
	if updatedTD == nil || updatedTD.Size == nil || *updatedTD.Size != 8 {
		t.Errorf("expected persisted td.Size=8, got %v", updatedTD)
	}
}

func TestTechDebtService_Update_ClearSize(t *testing.T) {
	ctx := context.Background()
	existingSize := 5
	var updatedTD *models.TechDebt

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Old", Size: &existingSize},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}
	svc := newTechDebtService(repo)

	td, err := svc.UpdateTechDebt(ctx, "TD-001", TechDebtUpdates{ClearSize: true})
	if err != nil {
		t.Fatalf("UpdateTechDebt() error = %v", err)
	}
	if td.Size != nil {
		t.Errorf("expected td.Size=nil after ClearSize, got %d", *td.Size)
	}
	if updatedTD == nil {
		t.Fatal("expected update to be called")
	}
	if updatedTD.Size != nil {
		t.Errorf("expected persisted td.Size=nil, got %d", *updatedTD.Size)
	}
}

func TestTechDebtService_Update_SizeNoOpPreservesExisting(t *testing.T) {
	ctx := context.Background()
	existingSize := 5
	var updatedTD *models.TechDebt

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Old", Size: &existingSize},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}
	svc := newTechDebtService(repo)

	newTitle := "New Title"
	// Touch only Title; Size and ClearSize both unset → existing size preserved.
	_, err := svc.UpdateTechDebt(ctx, "TD-001", TechDebtUpdates{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateTechDebt() error = %v", err)
	}
	if updatedTD == nil || updatedTD.Size == nil || *updatedTD.Size != 5 {
		t.Errorf("expected td.Size to be preserved as 5, got %v", updatedTD)
	}
}

// --- DeleteTechDebt tests ---

func TestTechDebtService_Delete_Success(t *testing.T) {
	ctx := context.Background()

	var deletedID int64
	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, _ string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 42, Key: "TD-001", Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		deleteFn: func(_ context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}

	svc := newTechDebtService(repo)

	err := svc.DeleteTechDebt(ctx, "TD-001")
	if err != nil {
		t.Fatalf("DeleteTechDebt() error = %v", err)
	}
	if deletedID != 42 {
		t.Errorf("expected delete with ID 42, got %d", deletedID)
	}
}

// --- AdvanceStatus tests ---

func TestTechDebtService_AdvanceStatus_Success(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ int64, status models.TechDebtStatus) error {
			if status != "triaged" {
				t.Errorf("expected status triaged, got %s", status)
			}
			return nil
		},
	}

	svc := newTechDebtService(repo)

	info, err := svc.GetNextStatus(ctx, "TD-001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if len(info.AvailableTransitions) == 0 {
		t.Fatal("expected at least one transition from identified")
	}
}

func TestTechDebtService_AdvanceStatus_TerminalState(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "resolved",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	info, err := svc.GetNextStatus(ctx, "TD-001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if len(info.AvailableTransitions) != 0 {
		t.Errorf("expected no transitions from resolved, got %d", len(info.AvailableTransitions))
	}
}

// --- SetStatus tests ---

func TestTechDebtService_SetStatus_ValidTransition(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ int64, status models.TechDebtStatus) error {
			return nil
		},
	}

	svc := newTechDebtService(repo)

	result, err := svc.TransitionStatus(ctx, "TD-001", "research", TransitionOptions{})
	if err != nil {
		t.Fatalf("TransitionStatus() error = %v", err)
	}
	if result.ToStatus != "research" {
		t.Errorf("expected to_status research, got %s", result.ToStatus)
	}
}

func TestTechDebtService_SetStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	// "resolved" is not a valid transition from "identified" (must go through triaged first)
	_, err := svc.TransitionStatus(ctx, "TD-001", "resolved", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestTechDebtService_SetStatus_Force(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ int64, _ models.TechDebtStatus) error {
			return nil
		},
	}

	svc := newTechDebtService(repo)

	// Force bypasses transition validation (requires reason)
	result, err := svc.TransitionStatus(ctx, "TD-001", "resolved", TransitionOptions{Force: true, Reason: "testing force"})
	if err != nil {
		t.Fatalf("TransitionStatus() with force error = %v", err)
	}
	if result.ToStatus != "resolved" {
		t.Errorf("expected to_status resolved, got %s", result.ToStatus)
	}
}

// --- Triage tests ---

func TestTechDebtService_Triage_FromIdentified(t *testing.T) {
	ctx := context.Background()

	var updatedTD *models.TechDebt
	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}

	svc := newTechDebtService(repo)

	td, err := svc.TriageTechDebt(ctx, "TD-001", TriageTechDebtInput{
		Severity: "critical",
		Category: "architecture",
	})
	if err != nil {
		t.Fatalf("TriageTechDebt() error = %v", err)
	}
	if td.Status != "research" {
		t.Errorf("expected status research, got %s", td.Status)
	}
	if updatedTD == nil {
		t.Fatal("expected update to be called")
	}
	if updatedTD.Severity != models.TechDebtSeverityCritical {
		t.Errorf("expected severity critical, got %s", updatedTD.Severity)
	}
	if updatedTD.Category != models.TechDebtCategoryArchitecture {
		t.Errorf("expected category architecture, got %s", updatedTD.Category)
	}
}

func TestTechDebtService_Triage_AlreadyTriaged(t *testing.T) {
	ctx := context.Background()

	var updatedTD *models.TechDebt
	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "in_progress",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}

	svc := newTechDebtService(repo)

	td, err := svc.TriageTechDebt(ctx, "TD-001", TriageTechDebtInput{
		Severity: "high",
	})
	if err != nil {
		t.Fatalf("TriageTechDebt() error = %v", err)
	}
	// Status should NOT change from in_progress (already past identified)
	if td.Status != "in_progress" {
		t.Errorf("expected status in_progress (unchanged), got %s", td.Status)
	}
	if updatedTD.Severity != models.TechDebtSeverityHigh {
		t.Errorf("expected severity high, got %s", updatedTD.Severity)
	}
}

// --- ListTechDebts tests ---

func TestTechDebtService_List_WithFilters(t *testing.T) {
	ctx := context.Background()

	var capturedFilters techdebt.TechDebtFilters
	repo := &mockTechDebtRepo{
		listWithFiltersFn: func(_ context.Context, filters techdebt.TechDebtFilters) ([]*models.TechDebt, error) {
			capturedFilters = filters
			return []*models.TechDebt{
				{
					BaseEntity: models.BaseEntity{ID: 1, Key: "TD-001", Title: "Test"},
					Status:     "identified",
					Category:   models.TechDebtCategoryCodeQuality,
					Severity:   models.TechDebtSeverityMedium,
				},
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	cat := models.TechDebtCategoryCodeQuality
	status := "identified"
	items, err := svc.ListTechDebts(ctx, TechDebtFilters{
		Status:   &status,
		Category: &cat,
	})
	if err != nil {
		t.Fatalf("ListTechDebts() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if capturedFilters.Status == nil || *capturedFilters.Status != "identified" {
		t.Error("expected status filter to be passed to repository")
	}
	if capturedFilters.Category == nil || *capturedFilters.Category != models.TechDebtCategoryCodeQuality {
		t.Error("expected category filter to be passed to repository")
	}
}

// --- GetStatusOptions tests ---

func TestTechDebtService_GetStatusOptions(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	options, err := svc.GetStatusOptions(ctx, "TD-001")
	if err != nil {
		t.Fatalf("GetStatusOptions() error = %v", err)
	}
	if len(options) == 0 {
		t.Error("expected at least one status option from identified")
	}
	// Verify "research" is a valid option from "identified"
	found := false
	for _, opt := range options {
		if opt == "research" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'research' in options, got %v", options)
	}
}
