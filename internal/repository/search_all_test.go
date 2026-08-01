package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSearchAllTestDB creates an in-memory DB for SearchAll tests.
func setupSearchAllTestDB(t *testing.T) *DB {
	t.Helper()
	testDB, err := db.InitDB(":memory:")
	require.NoError(t, err)
	return &DB{DB: testDB}
}

// seedSearchAllTestData inserts minimal searchable records for each supported entity type.
func seedSearchAllTestData(t *testing.T, repoDb *DB) {
	t.Helper()
	ctx := context.Background()

	// Epic
	epicRepo := NewEpicRepository(repoDb)
	epicDesc := "Enhancements roadmap for cross entity search"
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E01", Title: "Login Enhancements System", Description: &epicDesc},
		Status:     "active",
		Priority:   "high",
	}
	require.NoError(t, epicRepo.Create(ctx, epic))

	// Feature
	featureRepo := NewFeatureRepository(repoDb)
	featureDesc := "Unified index synchronizer appears only in the feature body"
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "Authentication", Description: &featureDesc},
		EpicID:     epic.ID,
		Status:     "active",
	}
	require.NoError(t, featureRepo.Create(ctx, feature))

	// Task
	taskRepo := NewTaskRepository(repoDb)
	taskDesc := "Implement login endpoint"
	agent := "backend"
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001",
		Title:       "Implement login flow",
		Description: &taskDesc}, FeatureID: feature.ID,

		Status:    "todo",
		Priority:  5,
		AgentType: &agent,
	}
	require.NoError(t, taskRepo.Create(ctx, task))

	// Bug
	bugRepo := NewBugRepository(repoDb)
	bugDesc := "Login button fails on mobile"
	bug := &models.Bug{BaseEntity: models.BaseEntity{Key: "B001",
		Title:       "Login button broken",
		Description: &bugDesc}, Status: models.BugStatus("reported"),
		Severity: models.BugSeverityHigh,
	}
	require.NoError(t, bugRepo.Create(ctx, bug))

	// Change-Card
	ccRepo := NewChangeCardRepository(repoDb)
	ccDesc := "Add dark mode support to login page"
	cc := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001",
		Title:       "Dark mode login",
		Description: &ccDesc}, Status: models.ChangeCardStatus("proposed"),
		Priority: 5,
	}
	require.NoError(t, ccRepo.Create(ctx, cc))

	_, err := repoDb.ExecContext(ctx, `
		INSERT INTO tech_debts (key, title, description, status, category, severity)
		VALUES ('TD-001', 'Refactor login search debt', 'Search repository needs unified FTS coverage', 'identified', 'code-quality', 'medium')
	`)
	require.NoError(t, err)

	_, err = repoDb.ExecContext(ctx, `
		INSERT INTO ideas (key, title, description, created_date, status)
		VALUES ('I-2026-07-05-01', 'Search everything idea', 'Users should find notes and ideas with full text search', '2026-07-05', 'new')
	`)
	require.NoError(t, err)

	// Question metadata is searchable, while ContextData must remain outside
	// the FTS projection. The explicit sentinel is asserted below.
	_, err = repoDb.ExecContext(ctx, `
		INSERT INTO questions (key, title, summary, requester, blocking, status, context_data)
		VALUES ('Q001', 'Login decision question', 'Choose the login provider', 'security-owner', 1, 'draft', 'search-context-sentinel-must-not-project')
	`)
	require.NoError(t, err)
}

// --- SearchAll tests ---

func rebuildSearchAllIndex(t *testing.T, repo *SearchRepository) {
	t.Helper()
	require.NoError(t, repo.RebuildIndex(context.Background()))
}

func requireSearchResult(t *testing.T, results []*EntitySearchResult, entityType, key string) *EntitySearchResult {
	t.Helper()
	for _, result := range results {
		if result.EntityType == entityType && result.Key == key {
			assert.NotZero(t, result.ID, "search result ID should be populated for %s %s", entityType, key)
			return result
		}
	}
	t.Fatalf("expected %s %s in search results, got %#v", entityType, key, results)
	return nil
}

func TestSearchAll_ReturnsAllEntityTypes(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	// "login" matches the task title, bug title, and change-card description
	results, err := repo.SearchAll(ctx, "login", nil)
	require.NoError(t, err)

	typesSeen := map[string]bool{}
	for _, r := range results {
		typesSeen[r.EntityType] = true
	}

	assert.True(t, typesSeen["task"], "expected task in results")
	assert.True(t, typesSeen["bug"], "expected bug in results")
	assert.True(t, typesSeen["change"], "expected change-card in results")
}

func TestSearchAll_QuestionMetadataIsDiscoverableWithoutContextData(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()
	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)

	entityType := "question"
	results, err := repo.SearchAll(context.Background(), "login", &entityType)
	require.NoError(t, err)
	result := requireSearchResult(t, results, "question", "Q001")
	assert.Equal(t, "draft", result.Status)
	assert.NotContains(t, strings.ToLower(result.Snippet), "search-context-sentinel-must-not-project")

	var indexedBody, indexedNotes, indexedMetadata string
	require.NoError(t, repoDb.QueryRow(`
		SELECT body, note_text, metadata_text FROM entity_search_fts
		WHERE entity_type = 'question' AND key = 'Q001'
	`).Scan(&indexedBody, &indexedNotes, &indexedMetadata))
	assert.Contains(t, indexedBody, "Choose the login provider")
	assert.Empty(t, indexedNotes)
	assert.NotContains(t, indexedBody+indexedNotes+indexedMetadata, "search-context-sentinel-must-not-project")
}

func TestSearchAll_EmptyQueryReturnsEmpty(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	results, err := repo.SearchAll(ctx, "", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchAll_TypeFilterBug(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	entityType := "bug"
	results, err := repo.SearchAll(ctx, "login", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.Equal(t, "bug", r.EntityType)
	}
}

func TestSearchAll_TypeFilterChange(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	entityType := "change"
	results, err := repo.SearchAll(ctx, "dark mode", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.Equal(t, "change", r.EntityType)
	}
}

func TestSearchAll_TypeFilterTask(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	entityType := "task"
	results, err := repo.SearchAll(ctx, "login", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.Equal(t, "task", r.EntityType)
	}
}

func TestSearchAll_BugResultIncludesSeverity(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	entityType := "bug"
	results, err := repo.SearchAll(ctx, "login", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		if r.EntityType == "bug" {
			assert.NotEmpty(t, r.Severity, "bug result should have severity")
		}
	}
}

func TestSearchAll_ChangeResultHasNoSeverity(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	entityType := "change"
	results, err := repo.SearchAll(ctx, "dark mode", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		if r.EntityType == "change" {
			assert.Empty(t, r.Severity, "change-card result should not have severity")
		}
	}
}

func TestSearchAll_ResultsHaveRequiredFields(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	results, err := repo.SearchAll(ctx, "login", nil)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.NotEmpty(t, r.EntityType, "EntityType must be set")
		assert.NotEmpty(t, r.Key, "Key must be set")
		assert.NotEmpty(t, r.Title, "Title must be set")
		assert.NotEmpty(t, r.Status, "Status must be set")
	}
}

func TestSearchAll_NoMatchReturnsEmpty(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)
	ctx := context.Background()

	results, err := repo.SearchAll(ctx, "xyznonexistent12345", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchAll_ReturnsRankAndSnippetForEpicTitle(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)

	results, err := repo.SearchAll(context.Background(), "enhancements", nil)
	require.NoError(t, err)

	result := requireSearchResult(t, results, "epic", "E01")
	assert.NotZero(t, result.Rank, "rank should come from FTS backend")
	assert.Contains(t, strings.ToLower(result.Snippet), "<mark>enhancements</mark>")
}

func TestSearchAll_SearchesFeatureDescriptionBody(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)

	entityType := "feature"
	results, err := repo.SearchAll(context.Background(), "synchronizer", &entityType)
	require.NoError(t, err)

	result := requireSearchResult(t, results, "feature", "E01-F01")
	assert.NotZero(t, result.Rank)
	assert.Contains(t, strings.ToLower(result.Snippet), "<mark>synchronizer</mark>")
}

func TestSearchAll_IdeaTypeFilterReturnsIndexedIdea(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)

	entityType := "idea"
	results, err := repo.SearchAll(context.Background(), "everything", &entityType)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "idea", results[0].EntityType)
	assert.Equal(t, "I-2026-07-05-01", results[0].Key)
	assert.NotZero(t, results[0].Rank)
	assert.NotEmpty(t, results[0].Snippet)
}

func TestSearchAll_TechDebtTypeFilterReturnsRankedSnippet(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)

	entityType := "tech_debt"
	results, err := repo.SearchAll(context.Background(), "refactor", &entityType)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "TD-001", results[0].Key)
	assert.NotZero(t, results[0].Rank)
	assert.Contains(t, strings.ToLower(results[0].Snippet), "<mark>refactor</mark>")
}

func TestSearchAll_SearchesEntityKeys(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)

	entityType := "bug"
	results, err := repo.SearchAll(context.Background(), "B001", &entityType)
	require.NoError(t, err)

	result := requireSearchResult(t, results, "bug", "B001")
	assert.NotZero(t, result.Rank)
	assert.Contains(t, strings.ToLower(result.Snippet), "<mark>b001</mark>")
}

func TestSearchAll_UsesPorterTokenizerStemming(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)

	entityType := "task"
	results, err := repo.SearchAll(context.Background(), "implementing", &entityType)
	require.NoError(t, err)

	requireSearchResult(t, results, "task", "T-E01-F01-001")
}

func TestSearchAll_EscapesFTSQuerySyntax(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	rebuildSearchAllIndex(t, repo)

	syntaxHeavyQueries := []string{
		`login "`,
		`login OR title:task`,
		`login -broken`,
		`E01-F01-001`,
	}
	for _, query := range syntaxHeavyQueries {
		t.Run(query, func(t *testing.T) {
			_, err := repo.SearchAll(context.Background(), query, nil)
			require.NoError(t, err)
		})
	}
}
