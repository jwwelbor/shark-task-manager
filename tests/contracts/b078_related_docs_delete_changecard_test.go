package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestB078RelatedDocsDeletePrefersCanonicalChangeCardAlias(t *testing.T) {
	cli, sqlDB, repoDB := newB078RelatedDocsFixture(t)
	canonical, canonicalDoc := seedB078ChangeCardLink(t, sqlDB, repoDB, "CC-001", "fixture", "shared title", "docs/b078/canonical.md")
	compact, compactDoc := seedB078ChangeCardLink(t, sqlDB, repoDB, "C001", "fixture", "shared title", "docs/b078/compact.md")

	var response map[string]string
	require.NoError(t, json.Unmarshal(cli.success(t, "related-docs", "delete", "shared title", "--change=C001"), &response))
	require.Equal(t, "change-card", response["parent"])
	assertB078Link(t, sqlDB, models.EntityTypeChange, canonical.ID, canonicalDoc.ID, false)
	assertB078Link(t, sqlDB, models.EntityTypeChange, compact.ID, compactDoc.ID, true)
}

func TestB078RelatedDocsDeleteFallsBackToCompactChangeCard(t *testing.T) {
	cli, sqlDB, repoDB := newB078RelatedDocsFixture(t)
	compact, doc := seedB078ChangeCardLink(t, sqlDB, repoDB, "C002", "compact-only", "compact-only title", "docs/b078/compact-only.md")

	var response map[string]string
	require.NoError(t, json.Unmarshal(cli.success(t, "related-docs", "delete", "compact-only title", "--change=C002"), &response))
	assertB078Link(t, sqlDB, models.EntityTypeChange, compact.ID, doc.ID, false)
}

func TestB078RelatedDocsDeleteDoesNotFallbackAfterCanonicalChangeCardScanFailure(t *testing.T) {
	cli, sqlDB, repoDB := newB078RelatedDocsFixture(t)
	canonical, canonicalDoc := seedB078ChangeCardLink(t, sqlDB, repoDB, "CC-003", "fixture", "unreadable title", "docs/b078/unreadable-canonical.md")
	compact, compactDoc := seedB078ChangeCardLink(t, sqlDB, repoDB, "C003", "fixture", "unreadable title", "docs/b078/unreadable-compact.md")
	_, err := sqlDB.ExecContext(context.Background(), "UPDATE change_cards SET priority = NULL WHERE key = 'CC-003'")
	require.NoError(t, err)

	for _, key := range []string{"C003", "CC-003-fixture"} {
		for _, jsonMode := range []bool{true, false} {
			assertB078UnreadableChangeCardDelete(t, cli, sqlDB, key, jsonMode, canonical, canonicalDoc, compact, compactDoc)
		}
	}
}

func TestB078RelatedDocsDeleteRejectsWrongChangeCardSlug(t *testing.T) {
	cli, sqlDB, repoDB := newB078RelatedDocsFixture(t)
	card, doc := seedB078ChangeCardLink(t, sqlDB, repoDB, "CC-004", "correct", "wrong slug title", "docs/b078/wrong-slug.md")

	message, exitCode := cli.rejectionWithExit(t, "related-docs", "delete", "wrong slug title", "--change=CC-004-wrong")
	require.Equal(t, 1, exitCode)
	require.Contains(t, message, "CC-004-wrong")
	assertB078Link(t, sqlDB, models.EntityTypeChange, card.ID, doc.ID, true)
}

func seedB078ChangeCardLink(t *testing.T, sqlDB *sql.DB, repoDB *repository.DB, key, slug, title, path string) (*models.ChangeCard, *models.Document) {
	t.Helper()
	card := createB078ChangeCard(t, sqlDB, key, slug)
	doc, err := repository.NewDocumentRepository(repoDB).CreateOrGet(context.Background(), title, path)
	require.NoError(t, err)
	require.NoError(t, repository.NewEntityDocumentRepository(repoDB).Link(context.Background(), models.EntityTypeChange, card.ID, doc.ID, "general"))
	return card, doc
}

func assertB078UnreadableChangeCardDelete(t *testing.T, cli b078CLI, sqlDB *sql.DB, key string, jsonMode bool, canonical *models.ChangeCard, canonicalDoc *models.Document, compact *models.ChangeCard, compactDoc *models.Document) {
	t.Helper()
	stdout, stderr, exitCode := cli.runWithDatabase(cli.dbPath, jsonMode, "related-docs", "delete", "unreadable title", "--change="+key)
	require.Equal(t, 2, exitCode)
	message := string(stderr)
	if jsonMode {
		var failure struct {
			Error   bool   `json:"error"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal(stdout, &failure))
		require.True(t, failure.Error)
		require.Equal(t, "COMMAND_ERROR", failure.Code)
		message = failure.Message
	} else {
		require.Empty(t, stdout)
	}
	require.Contains(t, message, "failed to unlink document from change-card "+key)
	require.Contains(t, strings.ToLower(message), "null")
	require.NotContains(t, message, "not found")
	require.NotContains(t, message, "unlinked")
	assertB078Link(t, sqlDB, models.EntityTypeChange, canonical.ID, canonicalDoc.ID, true)
	assertB078Link(t, sqlDB, models.EntityTypeChange, compact.ID, compactDoc.ID, true)
}
