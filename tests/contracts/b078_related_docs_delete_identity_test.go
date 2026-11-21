package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestB078RelatedDocsDeleteUsesParentScopedDocumentIdentity(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	winnerPaths := make([]string, 0, 2)
	for _, reverse := range []int{0, 1} {
		t.Run(fmt.Sprintf("reverse unordered select %d", reverse), func(t *testing.T) {
			winnerPaths = append(winnerPaths, assertB078LegacyTitleUnlink(t, fixture, reverse))
		})
	}
	require.NotEqual(t, winnerPaths[0], winnerPaths[1], "legacy lookup must select both path assignments")
}

func TestB078RelatedDocsDeleteUsesExactTitleMatching(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	caseVariant, err := fixture.documents.CreateOrGet(context.Background(), "B078 Case Variant", "docs/b078/identity/case.md")
	require.NoError(t, err)
	differentTitle, err := fixture.documents.CreateOrGet(context.Background(), "B078 different title", "docs/b078/identity/different.md")
	require.NoError(t, err)
	fixture.link(t, fixture.target.ID, caseVariant.ID, "case")
	fixture.link(t, fixture.target.ID, differentTitle.ID, "different")

	var response map[string]string
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", "b078 case variant", "--task="+fixture.target.Key), &response))
	require.Equal(t, "unlinked", response["status"])
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.target.ID, caseVariant.ID, true)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.target.ID, differentTitle.ID, true)
}

func TestB078RelatedDocsDeleteIgnoresTitleLinkedOnlyElsewhere(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	doc, err := fixture.documents.CreateOrGet(context.Background(), "B078 title linked elsewhere only", "docs/b078/identity/elsewhere-only.md")
	require.NoError(t, err)
	fixture.link(t, fixture.other.ID, doc.ID, "elsewhere")

	fixture.cli.success(t, "related-docs", "delete", doc.Title, "--task="+fixture.target.Key)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.other.ID, doc.ID, true)
}

func TestB078RelatedDocsDeletePreservesSharedDocumentForOtherParent(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	doc := createB078IdentityDocument(t, fixture, "B078 shared document", "docs/b078/identity/shared.md", "shared document bytes\n")
	fixture.link(t, fixture.target.ID, doc.ID, "target-share")
	fixture.link(t, fixture.other.ID, doc.ID, "other-share")

	fixture.cli.success(t, "related-docs", "delete", doc.Title, "--task="+fixture.target.Key)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.target.ID, doc.ID, false)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.other.ID, doc.ID, true)
	assertB078LinkMetadata(t, fixture.sqlDB, models.EntityTypeTask, fixture.other.ID, doc.ID, "other-share")
	assertB078DocumentRowAndFile(t, fixture.sqlDB, fixture.cli.dir, doc, []byte("shared document bytes\n"))
}

func TestB078RelatedDocsDeleteRejectsAmbiguousParentMatches(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	first, err := fixture.documents.CreateOrGet(context.Background(), "B078 same-parent ambiguity", "docs/b078/identity/ambiguous-one.md")
	require.NoError(t, err)
	second, err := fixture.documents.CreateOrGet(context.Background(), "B078 same-parent ambiguity", "docs/b078/identity/ambiguous-two.md")
	require.NoError(t, err)
	fixture.link(t, fixture.target.ID, first.ID, "first")
	fixture.link(t, fixture.target.ID, second.ID, "second")

	message, exitCode := fixture.cli.rejectionWithExit(t, "related-docs", "delete", first.Title, "--task="+fixture.target.Key)
	require.Equal(t, 3, exitCode)
	require.Contains(t, message, first.Title)
	require.Contains(t, message, fixture.target.Key)
	require.Equal(t, 1, strings.Count(message, fixture.target.Key), "public entity context should appear only once")
	require.Contains(t, message, "(2 linked documents)")
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.target.ID, first.ID, true)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.target.ID, second.ID, true)
}

type b078IdentityFixture struct {
	cli       b078CLI
	sqlDB     *sql.DB
	target    *models.Task
	other     *models.Task
	documents *repository.DocumentRepository
	links     *repository.EntityDocumentRepository
}

func newB078IdentityFixture(t *testing.T) b078IdentityFixture {
	t.Helper()
	cli, sqlDB, target, other := newB078RelatedDocsDeleteFixture(t)
	sqlDB.SetMaxIdleConns(0)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	repoDB := repository.NewDB(sqlDB)
	return b078IdentityFixture{cli: cli, sqlDB: sqlDB, target: target, other: other,
		documents: repository.NewDocumentRepository(repoDB), links: repository.NewEntityDocumentRepository(repoDB)}
}

func (fixture b078IdentityFixture) link(t *testing.T, taskID, documentID int64, linkType string) {
	t.Helper()
	require.NoError(t, fixture.links.Link(context.Background(), models.EntityTypeTask, taskID, documentID, linkType))
}

func assertB078LegacyTitleUnlink(t *testing.T, fixture b078IdentityFixture, reverse int) string {
	t.Helper()
	_, err := fixture.sqlDB.ExecContext(context.Background(), fmt.Sprintf("PRAGMA reverse_unordered_selects = %d", reverse))
	require.NoError(t, err)
	title := fmt.Sprintf("B078 duplicate legacy title %d", reverse)
	first := createB078IdentityDocument(t, fixture, title, fmt.Sprintf("docs/b078/identity/%d-first.md", reverse), "first bytes\n")
	second := createB078IdentityDocument(t, fixture, title, fmt.Sprintf("docs/b078/identity/%d-second.md", reverse), "second bytes\n")
	winner, err := fixture.documents.GetByTitle(context.Background(), title)
	require.NoError(t, err)
	require.Contains(t, []int64{first.ID, second.ID}, winner.ID)
	targetDoc, otherDoc := first, second
	if winner.ID == first.ID {
		targetDoc, otherDoc = second, first
	}
	fixture.link(t, fixture.target.ID, targetDoc.ID, "specification")
	fixture.link(t, fixture.other.ID, otherDoc.ID, "legacy")

	var response map[string]string
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", title, "--task="+fixture.target.Key), &response))
	require.Equal(t, "unlinked", response["status"])
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.target.ID, targetDoc.ID, false)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTask, fixture.other.ID, otherDoc.ID, true)
	assertB078LinkMetadata(t, fixture.sqlDB, models.EntityTypeTask, fixture.other.ID, otherDoc.ID, "legacy")
	assertB078DocumentRowAndFile(t, fixture.sqlDB, fixture.cli.dir, targetDoc, identityDocumentBytes("first bytes\n", "second bytes\n", targetDoc, first, second))
	assertB078DocumentRowAndFile(t, fixture.sqlDB, fixture.cli.dir, otherDoc, identityDocumentBytes("first bytes\n", "second bytes\n", otherDoc, first, second))
	return winner.FilePath
}

func createB078IdentityDocument(t *testing.T, fixture b078IdentityFixture, title, path, contents string) *models.Document {
	t.Helper()
	doc, err := fixture.documents.CreateOrGet(context.Background(), title, path)
	require.NoError(t, err)
	fullPath := filepath.Join(fixture.cli.dir, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0700))
	require.NoError(t, os.WriteFile(fullPath, []byte(contents), 0600))
	return doc
}

func identityDocumentBytes(firstBytes, secondBytes string, doc, first, second *models.Document) []byte {
	if doc.ID == first.ID {
		return []byte(firstBytes)
	}
	return []byte(secondBytes)
}
