package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/stretchr/testify/require"
)

func TestB078RelatedDocsDeleteRequiresTechDebtKeyAndSlugToMatch(t *testing.T) {
	fixture := newB078TechDebtFixture(t)
	for _, selector := range []string{"TD-001-stale", "TD-999-fixture"} {
		assertB078TechDebtSelectorMiss(t, fixture, selector)
	}
}

func TestB078RelatedDocsDeleteUsesTechDebtParentIdentity(t *testing.T) {
	fixture := newB078TechDebtFixture(t)
	first, err := fixture.documents.CreateOrGet(context.Background(), "TD ambiguous", "docs/b078/td-ambiguous-1.md")
	require.NoError(t, err)
	second, err := fixture.documents.CreateOrGet(context.Background(), "TD ambiguous", "docs/b078/td-ambiguous-2.md")
	require.NoError(t, err)
	fixture.link(t, fixture.target.ID, first.ID, "first")
	fixture.link(t, fixture.target.ID, second.ID, "second")
	message, exitCode := fixture.cli.rejectionWithExit(t, "related-docs", "delete", "TD ambiguous", "--tech-debt=TD-001-fixture")
	require.Equal(t, 3, exitCode)
	require.Contains(t, message, "ambiguous")
	require.ElementsMatch(t, []int64{fixture.targetDoc.ID, first.ID, second.ID}, fixture.cli.relatedDocIDs(t, "--tech-debt=TD-001"))

	fixture.cli.success(t, "related-docs", "delete", fixture.targetDoc.Title, "--tech-debt=TD-002-sibling")
	require.Equal(t, []int64{fixture.targetDoc.ID, first.ID, second.ID}, fixture.cli.relatedDocIDs(t, "--tech-debt=TD-001"))
	require.Empty(t, fixture.cli.relatedDocIDs(t, "--tech-debt=TD-002"))
	deleteB078TechDebtTargetLink(t, fixture, []int64{first.ID, second.ID})
}

func TestB078RelatedDocsDeleteNormalizesTechDebtSlugAndSupportsNumericRetry(t *testing.T) {
	fixture := newB078TechDebtFixture(t)
	padded := createB078TechDebt(t, fixture.repo, "TD-003", "padded")
	doc, err := fixture.documents.CreateOrGet(context.Background(), "Padded selector", "docs/b078/td-padded.md")
	require.NoError(t, err)
	require.NoError(t, fixture.links.Link(context.Background(), models.EntityTypeTechDebt, padded.ID, doc.ID, "general"))
	stdout, stderr, exitCode := fixture.cli.runWithDatabase(fixture.cli.dbPath, false, "related-docs", "delete", doc.Title, "--tech-debt= TD-003-padded ")
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)
	require.Equal(t, "Document unlinked from tech-debt TD-003-padded\n", string(stdout))
	require.Empty(t, fixture.cli.relatedDocIDs(t, "TD-003"))
	var retry map[string]string
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", doc.Title, "--tech-debt=TD-003"), &retry))
	require.Equal(t, "unlinked", retry["status"])
}

func TestB078RelatedDocsDeletePreservesTechDebtLinkOnSlugScanFailure(t *testing.T) {
	fixture := newB078TechDebtFixture(t)
	scan := createB078TechDebt(t, fixture.repo, "TD-004", "scan")
	doc, err := fixture.documents.CreateOrGet(context.Background(), "Unreadable tech-debt link", "docs/b078/td-scan.md")
	require.NoError(t, err)
	require.NoError(t, fixture.links.Link(context.Background(), models.EntityTypeTechDebt, scan.ID, doc.ID, "general"))
	_, err = fixture.sqlDB.ExecContext(context.Background(), "UPDATE tech_debts SET created_at = ? WHERE key = ?", "not-a-timestamp", scan.Key)
	require.NoError(t, err)

	message, exitCode := fixture.cli.rejectionWithExit(t, "related-docs", "delete", doc.Title, "--tech-debt=TD-004-scan")
	require.Equal(t, 2, exitCode)
	require.Contains(t, message, "failed to unlink document from tech-debt TD-004-scan")
	require.NotContains(t, message, "not found")
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTechDebt, scan.ID, doc.ID, true)
}

type b078TechDebtFixture struct {
	cli        b078CLI
	sqlDB      *sql.DB
	repo       *techdebt.TechDebtRepository
	documents  *repository.DocumentRepository
	links      *repository.EntityDocumentRepository
	target     *models.TechDebt
	sibling    *models.TechDebt
	targetDoc  *models.Document
	siblingDoc *models.Document
}

func newB078TechDebtFixture(t *testing.T) b078TechDebtFixture {
	t.Helper()
	cli, sqlDB, repoDB := newB078RelatedDocsFixture(t)
	repo := techdebt.NewTechDebtRepository(repoDB)
	target := createB078TechDebt(t, repo, "TD-001", "fixture")
	sibling := createB078TechDebt(t, repo, "TD-002", "sibling")
	documents := repository.NewDocumentRepository(repoDB)
	targetDoc, err := documents.CreateOrGet(context.Background(), "TD shared title", "docs/b078/td-target.md")
	require.NoError(t, err)
	siblingDoc, err := documents.CreateOrGet(context.Background(), "TD shared title", "docs/b078/td-sibling.md")
	require.NoError(t, err)
	links := repository.NewEntityDocumentRepository(repoDB)
	require.NoError(t, links.Link(context.Background(), models.EntityTypeTechDebt, target.ID, targetDoc.ID, "general"))
	require.NoError(t, links.Link(context.Background(), models.EntityTypeTechDebt, sibling.ID, siblingDoc.ID, "reference"))
	return b078TechDebtFixture{cli: cli, sqlDB: sqlDB, repo: repo, documents: documents, links: links,
		target: target, sibling: sibling, targetDoc: targetDoc, siblingDoc: siblingDoc}
}

func (fixture b078TechDebtFixture) link(t *testing.T, entityID, documentID int64, linkType string) {
	t.Helper()
	require.NoError(t, fixture.links.Link(context.Background(), models.EntityTypeTechDebt, entityID, documentID, linkType))
}

func assertB078TechDebtSelectorMiss(t *testing.T, fixture b078TechDebtFixture, selector string) {
	t.Helper()
	message, exitCode := fixture.cli.rejectionWithExit(t, "related-docs", "delete", "TD shared title", "--tech-debt="+selector)
	require.Equal(t, 1, exitCode)
	require.Contains(t, message, selector)
	require.Contains(t, message, "not found")
	require.Equal(t, []int64{fixture.targetDoc.ID}, fixture.cli.relatedDocIDs(t, "--tech-debt=TD-001"))
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTechDebt, fixture.target.ID, fixture.targetDoc.ID, true)
}

func deleteB078TechDebtTargetLink(t *testing.T, fixture b078TechDebtFixture, remainingIDs []int64) {
	t.Helper()
	var response map[string]string
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", fixture.targetDoc.Title, "--tech-debt=TD-001-fixture"), &response))
	require.Equal(t, "unlinked", response["status"])
	require.Equal(t, "tech-debt", response["parent"])
	require.ElementsMatch(t, remainingIDs, fixture.cli.relatedDocIDs(t, "--tech-debt=TD-001"))
	assertB078Link(t, fixture.sqlDB, models.EntityTypeTechDebt, fixture.target.ID, fixture.targetDoc.ID, false)
	var retainedDocument int
	require.NoError(t, fixture.sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM documents WHERE id = ?", fixture.siblingDoc.ID).Scan(&retainedDocument))
	require.Equal(t, 1, retainedDocument)
}
