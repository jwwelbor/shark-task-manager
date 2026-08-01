package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSearchService struct {
	SearchAllFunc func(ctx context.Context, query string, entityType string, tags []string) ([]*repository.EntitySearchResult, error)
}

func (m *mockSearchService) SearchAll(ctx context.Context, query string, entityType string, tags []string) ([]*repository.EntitySearchResult, error) {
	if m.SearchAllFunc != nil {
		return m.SearchAllFunc(ctx, query, entityType, tags)
	}
	return []*repository.EntitySearchResult{}, nil
}

func withSearchSvcOverride(t *testing.T, svc searchServicer) {
	t.Helper()
	orig := searchSvcOverride
	searchSvcOverride = svc
	t.Cleanup(func() { searchSvcOverride = orig })
}

func buildSearchQueryCmdForTest() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "search [query]",
		RunE:         runSearchQuery,
		SilenceUsage: true,
	}
	cmd.Flags().String("type", "", "type")
	cmd.Flags().StringSlice("tag", nil, "tag")
	return cmd
}

// --- validateSearchType tests ---

func TestValidateSearchType_ValidTypes(t *testing.T) {
	validTypes := []string{"epic", "feature", "task", "bug", "change", "changes", "change-card", "change_card", "idea", "ideas", "tech_debt", "tech-debt", "td", "question"}
	for _, typ := range validTypes {
		t.Run(typ, func(t *testing.T) {
			err := validateSearchType(typ)
			assert.NoError(t, err)
		})
	}
}

func TestValidateSearchType_EmptyStringIsValid(t *testing.T) {
	// Empty string means "all types" — no filter
	err := validateSearchType("")
	assert.NoError(t, err)
}

func TestValidateSearchType_InvalidTypeReturnsError(t *testing.T) {
	err := validateSearchType("foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "foo")
	assert.Contains(t, err.Error(), "bug")
	assert.Contains(t, err.Error(), "change")
}

func TestValidateSearchType_InvalidType_ListsAllValidTypes(t *testing.T) {
	err := validateSearchType("invalid")
	require.Error(t, err)
	msg := err.Error()
	for _, valid := range []string{"epic", "feature", "task", "bug", "change", "idea", "tech_debt", "question"} {
		assert.Contains(t, msg, valid, "error should list valid type %q", valid)
	}
}

// --- EntitySearchResult field tests ---

func TestEntitySearchResult_BugHasSeverity(t *testing.T) {
	r := &repository.EntitySearchResult{
		EntityType: "bug",
		Key:        "B001",
		Title:      "Login broken",
		Status:     "reported",
		Severity:   "high",
	}
	assert.NotEmpty(t, r.Severity, "bug result should have severity")
}

func TestEntitySearchResult_ChangeCardNoSeverity(t *testing.T) {
	r := &repository.EntitySearchResult{
		EntityType: "change",
		Key:        "CC-001",
		Title:      "Dark mode",
		Status:     "proposed",
		Severity:   "",
	}
	assert.Empty(t, r.Severity, "change-card result should not have severity")
}

// --- parseSearchQuery tests ---

func TestParseSearchQuery_PositionalArgUsedAsQuery(t *testing.T) {
	args := []string{"login"}
	query, err := parseSearchQuery(args)
	require.NoError(t, err)
	assert.Equal(t, "login", query)
}

func TestParseSearchQuery_EmptyArgsReturnsError(t *testing.T) {
	args := []string{}
	_, err := parseSearchQuery(args)
	require.Error(t, err)
}

func TestParseSearchQuery_MultipleWordsJoined(t *testing.T) {
	// If user passes "dark mode" as a single arg (quoted in shell)
	args := []string{"dark mode"}
	query, err := parseSearchQuery(args)
	require.NoError(t, err)
	assert.Equal(t, "dark mode", query)
}

func TestRunSearchQuery_ForwardsTypeAndTagsToService(t *testing.T) {
	var capturedQuery string
	var capturedType string
	var capturedTags []string
	mock := &mockSearchService{
		SearchAllFunc: func(ctx context.Context, query string, entityType string, tags []string) ([]*repository.EntitySearchResult, error) {
			capturedQuery = query
			capturedType = entityType
			capturedTags = tags
			return []*repository.EntitySearchResult{}, nil
		},
	}
	withSearchSvcOverride(t, mock)

	cmd := buildSearchQueryCmdForTest()
	cmd.SetArgs([]string{"unified", "index", "--type=idea", "--tag=voice", "--tag=auth"})
	require.NoError(t, cmd.Execute())

	assert.Equal(t, "unified index", capturedQuery)
	assert.Equal(t, "idea", capturedType)
	assert.Equal(t, []string{"voice", "auth"}, capturedTags)
}

func TestRunSearchQuery_NormalizesChangeTypeAlias(t *testing.T) {
	var capturedType string
	mock := &mockSearchService{
		SearchAllFunc: func(ctx context.Context, query string, entityType string, tags []string) ([]*repository.EntitySearchResult, error) {
			capturedType = entityType
			return []*repository.EntitySearchResult{}, nil
		},
	}
	withSearchSvcOverride(t, mock)

	cmd := buildSearchQueryCmdForTest()
	cmd.SetArgs([]string{"dark", "mode", "--type=change_card"})
	require.NoError(t, cmd.Execute())

	assert.Equal(t, "change", capturedType)
}

func TestRunSearchQuery_JSONIncludesRankAndSnippet(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	t.Cleanup(func() { cli.GlobalConfig.JSON = origJSON })

	mock := &mockSearchService{
		SearchAllFunc: func(ctx context.Context, query string, entityType string, tags []string) ([]*repository.EntitySearchResult, error) {
			return []*repository.EntitySearchResult{
				{
					EntityType: "idea",
					ID:         99,
					Key:        "I-2026-07-05-01",
					Title:      "Search everything idea",
					Status:     "new",
					Rank:       -1.25,
					Snippet:    "Search <mark>everything</mark> idea",
				},
			}, nil
		},
	}
	withSearchSvcOverride(t, mock)

	cmd := buildSearchQueryCmdForTest()
	cmd.SetArgs([]string{"everything", "--type=idea"})
	output := captureOutput(t, func() {
		require.NoError(t, cmd.Execute())
	})

	var results []map[string]interface{}
	require.NoError(t, json.Unmarshal(output, &results))
	require.Len(t, results, 1)
	assert.Equal(t, float64(99), results[0]["id"])
	assert.Equal(t, -1.25, results[0]["rank"])
	assert.Equal(t, "Search <mark>everything</mark> idea", results[0]["snippet"])
}

func TestPrintEntitySearchResults_IncludesSnippet(t *testing.T) {
	results := []*repository.EntitySearchResult{
		{
			EntityType: "task",
			Key:        "T-E01-F01-001",
			Title:      "Search task",
			Status:     "todo",
			Rank:       -0.75,
			Snippet:    "Notes mention <mark>needle</mark> here",
		},
	}

	output := captureOutput(t, func() {
		require.NoError(t, printEntitySearchResults(results, "needle"))
	})

	assert.Contains(t, string(output), "[task] T-E01-F01-001: Search task (todo)")
	assert.Contains(t, string(output), "Notes mention <mark>needle</mark> here")
}
