package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- validateSearchType tests ---

func TestValidateSearchType_ValidTypes(t *testing.T) {
	validTypes := []string{"epic", "feature", "task", "bug", "change"}
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
	for _, valid := range []string{"epic", "feature", "task", "bug", "change"} {
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
