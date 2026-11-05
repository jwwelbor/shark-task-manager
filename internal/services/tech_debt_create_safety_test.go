package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755), "create fixture directory")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644), "write fixture")
}

func assertTechDebtDocument(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	require.NoError(t, err, "read document")
	assert.Equal(t, want, string(got), "document at %q", path)
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
			require.Error(t, err)
			assert.Contains(t, err.Error(), "collision")
			assert.Nil(t, created)
			assert.False(t, linked)
			assert.Zero(t, createCalls)
			assert.Zero(t, deleteCalls)
			assertTechDebtDocument(t, path, tc.original)
		})
	}
}

func TestTechDebtService_Create_ForceRejectsNonRegularPaths(t *testing.T) {
	for _, pathKind := range []string{"directory", "symlink"} {
		t.Run(pathKind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
			targetPath := filepath.Join(root, "preserved.md")
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755), "create target parent")
			switch pathKind {
			case "directory":
				require.NoError(t, os.MkdirAll(path, 0755), "create existing directory")
			case "symlink":
				writeTechDebtFixture(t, targetPath, "PRESERVED-TARGET\n")
				if err := os.Symlink(targetPath, path); err != nil {
					t.Skipf("symlink creation is unavailable: %v", err)
				}
			}

			createCalls, deleteCalls := 0, 0
			svc := newTechDebtService(&mockTechDebtRepo{
				createFn: func(context.Context, *models.TechDebt) error { createCalls++; return nil },
				deleteFn: func(context.Context, int64) error { deleteCalls++; return nil },
			})
			svc.projectRoot = root
			created, linked, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
				Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
				Severity: models.TechDebtSeverityHigh, Force: true,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "existing path is not a regular file")
			assert.Nil(t, created)
			assert.False(t, linked)
			assert.Zero(t, createCalls)
			assert.Zero(t, deleteCalls)

			info, err := os.Lstat(path)
			require.NoError(t, err, "existing path must remain")
			if pathKind == "directory" {
				assert.True(t, info.IsDir(), "existing directory must be preserved")
				return
			}
			assert.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink)
			gotTarget, err := os.Readlink(path)
			require.NoError(t, err, "read preserved symlink")
			assert.Equal(t, targetPath, gotTarget)
			assertTechDebtDocument(t, targetPath, "PRESERVED-TARGET\n")
		})
	}
}

func TestTechDebtService_Create_ForceReusesExactDocumentWithoutWriting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
	writeTechDebtFixture(t, path, b112Placeholder)
	oldTime := time.Unix(100, 0)
	require.NoError(t, os.Chtimes(path, oldTime, oldTime), "set fixture timestamp")
	before, err := os.Stat(path)
	require.NoError(t, err, "stat fixture")
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
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, int64(17), created.ID)
	assert.True(t, linked)
	assert.Equal(t, 1, createCalls)
	after, err := os.Stat(path)
	require.NoError(t, err, "stat reused document")
	assert.True(t, os.SameFile(before, after), "exact-match reuse replaced the existing document")
	assert.True(t, after.ModTime().Equal(oldTime), "exact-match reuse rewrote the existing document")
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
					require.NotNil(t, td.FilePath)
					assert.Equal(t, path, *td.FilePath)
					assertTechDebtDocument(t, fullPath, b112Placeholder)
					return nil
				},
			})
			svc.projectRoot = root
			created, linked, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
				Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
				Severity: models.TechDebtSeverityHigh, FilePath: inputPath,
			})
			require.NoError(t, err)
			require.NotNil(t, created)
			assert.False(t, linked)
			assert.Equal(t, 1, createCalls)
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
	require.Error(t, err, "invalid model must be rejected")
	assert.Zero(t, createCalls, "invalid model must not reach repository Create")
	path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
	_, statErr := os.Stat(path)
	assert.ErrorIs(t, statErr, os.ErrNotExist, "invalid input must not create a document at %q", path)
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
				require.NoError(t, err)
				assert.True(t, linked)
				assert.Equal(t, 1, createCalls)
			} else {
				require.Error(t, err)
				assert.False(t, linked)
				assert.Zero(t, createCalls)
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
	assert.Zero(t, createCalls)
	assert.Zero(t, deleteCalls)
	assertTechDebtDocument(t, blocker, "PRESERVE-PARENT-FILE")
}

func TestTechDebtService_Create_CommittedInsertErrorRequiresPersistenceCheckBeforeRetry(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
	insertErr := errors.New("last insert ID unavailable after commit")
	nextKey := "TD-001"
	createCalls, deleteCalls := 0, 0
	svc := newTechDebtService(&mockTechDebtRepo{
		generateNextKeyFn: func(context.Context) (string, error) { return nextKey, nil },
		createFn: func(_ context.Context, td *models.TechDebt) error {
			createCalls++
			assert.Equal(t, "TD-001", td.Key)
			// The insert committed even though the repository could not report success.
			// Re-entering create would now allocate a different key.
			nextKey = "TD-002"
			return insertErr
		},
		deleteFn: func(context.Context, int64) error { deleteCalls++; return nil },
	})
	svc.projectRoot = root
	created, linked, err := svc.CreateTechDebt(context.Background(), CreateTechDebtInput{
		Title: "Retry debt", Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityHigh,
	})
	require.ErrorIs(t, err, insertErr)
	assert.Nil(t, created)
	assert.False(t, linked)
	assert.Contains(t, err.Error(), "TD-001")
	assert.Contains(t, err.Error(), "docs/plan/tech-debt/TD-001.md")
	assert.Contains(t, err.Error(), "check whether TD-001 was persisted before retrying")
	assert.Contains(t, err.Error(), "a blind retry can create a second item")
	assert.NotContains(t, err.Error(), "retry the unchanged request with --force")
	assert.Equal(t, "TD-002", nextKey)
	assert.Equal(t, 1, createCalls)
	assert.Zero(t, deleteCalls)
	assertTechDebtDocument(t, path, b112Placeholder)
}

func TestTechDebtService_Create_UncommittedInsertRetainsDocumentForVerifiedExactRetry(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	createCalls, deleteCalls, persistedRows := 0, 0, 0
	svc := newTechDebtService(&mockTechDebtRepo{
		generateNextKeyFn: func(context.Context) (string, error) {
			if persistedRows == 0 {
				return "TD-001", nil
			}
			return "TD-002", nil
		},
		createFn: func(ctx context.Context, td *models.TechDebt) error {
			createCalls++
			assertTechDebtDocument(t, path, b112Placeholder)
			if createCalls == 1 {
				cancel()
				return ctx.Err()
			}
			td.ID = 23
			persistedRows++
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
	require.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "docs/plan/tech-debt/TD-001.md")
	assert.Contains(t, err.Error(), "check whether TD-001 was persisted before retrying")
	// This mock confirms that no row persisted and the next key remains TD-001.
	// Only after that verification is exact-content reuse a safe retry.
	require.Zero(t, persistedRows)
	assertTechDebtDocument(t, path, b112Placeholder)
	input.Force = true
	created, linked, err := svc.CreateTechDebt(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, int64(23), created.ID)
	assert.True(t, linked)
	assert.Equal(t, 2, createCalls)
	assert.Zero(t, deleteCalls)
	assert.Equal(t, 1, persistedRows)
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
			assert.NoError(t, err, "read complete document before insert")
			assert.Equal(t, expected[td.Title], string(got), "insert for %q must follow its complete document", td.Title)
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
	require.Equal(t, 1, successes)
	require.Len(t, created, 1)
	assert.Empty(t, deleted)
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
	require.NoError(t, err)
	assertTechDebtDocument(t, filepath.Join(root, "docs", "plan", "tech-debt", "TD-001.md"),
		"---\ntech_debt_key: TD-001\ntitle: Body debt\ncategory: architecture\nseverity: medium\nslug: body-debt\n---\nCUSTOM-CONTENT-BODY\n")
}
