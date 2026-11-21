package contracts

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestB078RelatedDocsDeleteBindsQualifiedFeatureAndRejectsAmbiguousShorthand(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	repoDB := repository.NewDB(fixture.sqlDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E11", Title: "B078 second epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	require.NoError(t, repository.NewEpicRepository(repoDB).Create(context.Background(), epic))
	first, err := repository.NewFeatureRepository(repoDB).GetByKey(context.Background(), "E10-F02")
	require.NoError(t, err)
	second := &models.Feature{BaseEntity: models.BaseEntity{Key: "E11-F02", Title: first.Title}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	require.NoError(t, repository.NewFeatureRepository(repoDB).Create(context.Background(), second))
	require.Equal(t, *first.Slug, *second.Slug)
	doc, err := fixture.documents.CreateOrGet(context.Background(), "B078 shared feature doc", "docs/b078/feature/shared.md")
	require.NoError(t, err)
	links := repository.NewEntityDocumentRepository(repoDB)
	require.NoError(t, links.Link(context.Background(), models.EntityTypeFeature, first.ID, doc.ID, "first"))
	require.NoError(t, links.Link(context.Background(), models.EntityTypeFeature, second.ID, doc.ID, "second"))

	// Qualified numeric keys must preserve the epic identity on the registered
	// get, add, and list routes even when the feature slug is shared.
	qualifiedGet, err := fixture.cli.run("feature", "get", "E10-F02")
	require.NoError(t, err)
	require.Contains(t, string(qualifiedGet), "E10-F02")
	_, err = fixture.cli.run("related-docs", "add", "B078 qualified feature doc", "docs/b078/feature/qualified.md", "--feature=E10-F02")
	require.NoError(t, err)
	qualifiedIDs := fixture.cli.relatedDocIDs(t, "--feature=E10-F02")
	siblingIDs := fixture.cli.relatedDocIDs(t, "--feature=E11-F02")
	require.NotEmpty(t, qualifiedIDs)
	require.NotEqual(t, qualifiedIDs, siblingIDs)
	var qualifiedDocs []models.Document
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "list", "--feature=E10-F02"), &qualifiedDocs))
	var siblingDocs []models.Document
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "list", "--feature=E11-F02"), &siblingDocs))
	require.Contains(t, documentPaths(qualifiedDocs), "docs/b078/feature/qualified.md")
	require.NotContains(t, documentPaths(siblingDocs), "docs/b078/feature/qualified.md")
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, first.ID, doc.ID, true)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, second.ID, doc.ID, true)

	_, exitCode := fixture.cli.rejectionWithExit(t, "related-docs", "delete", doc.Title, "--feature=E10-F02-wrong-slug")
	require.Equal(t, 1, exitCode)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, first.ID, doc.ID, true)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, second.ID, doc.ID, true)

	var output map[string]string
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", doc.Title, "--feature=E10-F02-"+*first.Slug), &output))
	require.Equal(t, "unlinked", output["status"])
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, first.ID, doc.ID, false)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, second.ID, doc.ID, true)

	for _, key := range []string{"F02", "F02-" + *first.Slug} {
		message, code := fixture.cli.rejectionWithExit(t, "related-docs", "delete", doc.Title, "--feature="+key)
		require.Equal(t, 3, code)
		require.Contains(t, message, key)
		require.Contains(t, message, "fully qualified feature key")
		assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, second.ID, doc.ID, true)
	}
	_, err = fixture.sqlDB.ExecContext(context.Background(), "UPDATE features SET slug = 'second-feature-slug' WHERE id = ?", second.ID)
	require.NoError(t, err)
	require.NoError(t, links.Link(context.Background(), models.EntityTypeFeature, first.ID, doc.ID, "first-retry"))
	_, err = fixture.cli.run("feature", "get", "F02-"+*first.Slug)
	require.NoError(t, err)
	_, err = fixture.cli.run("related-docs", "list", "--feature=F02-"+*first.Slug)
	require.NoError(t, err)
	fixture.cli.success(t, "related-docs", "delete", doc.Title, "--feature=F02-"+*first.Slug)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, first.ID, doc.ID, false)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, second.ID, doc.ID, true)
	// Non-delete feature callers retain their historical shorthand lookup.
	_, err = fixture.cli.run("feature", "get", "F02")
	require.NoError(t, err)
	_, err = fixture.cli.run("related-docs", "list", "--feature=F02")
	require.NoError(t, err)
	_, err = fixture.cli.run("related-docs", "add", "B078 shorthand compatibility", "docs/b078/feature/compatibility.md", "--feature=F02")
	require.NoError(t, err)
	selected, err := repository.NewFeatureRepository(repoDB).GetByKey(context.Background(), "F02")
	require.NoError(t, err)
	selectedIDs := fixture.cli.relatedDocIDs(t, "--feature="+selected.Key)
	otherKey := first.Key
	if selected.Key == first.Key {
		otherKey = second.Key
	}
	otherIDs := fixture.cli.relatedDocIDs(t, "--feature="+otherKey)
	var selectedDocs []models.Document
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "list", "--feature="+selected.Key), &selectedDocs))
	var otherDocs []models.Document
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "list", "--feature="+otherKey), &otherDocs))
	selectedHasAdded, otherHasAdded := false, false
	for _, item := range selectedDocs {
		selectedHasAdded = selectedHasAdded || strings.Contains(item.FilePath, "compatibility.md")
	}
	for _, item := range otherDocs {
		otherHasAdded = otherHasAdded || strings.Contains(item.FilePath, "compatibility.md")
	}
	require.True(t, selectedHasAdded)
	require.False(t, otherHasAdded)

	// Slugged shorthand add keeps the historical compatibility lookup target.
	slugSelected, err := repository.NewFeatureRepository(repoDB).GetByKey(context.Background(), "F02-"+*first.Slug)
	require.NoError(t, err)
	_, err = fixture.cli.run("related-docs", "add", "B078 slugged feature compatibility", "docs/b078/feature/slug-compatibility.md", "--feature=F02-"+*first.Slug)
	require.NoError(t, err)
	slugSelectedDocs := fixture.cli.relatedDocIDs(t, "--feature="+slugSelected.Key)
	slugOtherKey := first.Key
	if slugSelected.Key == first.Key {
		slugOtherKey = second.Key
	}
	slugOtherDocs := fixture.cli.relatedDocIDs(t, "--feature="+slugOtherKey)
	var slugSelectedList []models.Document
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "list", "--feature="+slugSelected.Key), &slugSelectedList))
	var slugOtherList []models.Document
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "list", "--feature="+slugOtherKey), &slugOtherList))
	require.Contains(t, documentPaths(slugSelectedList), "docs/b078/feature/slug-compatibility.md")
	require.NotContains(t, documentPaths(slugOtherList), "docs/b078/feature/slug-compatibility.md")
	require.NotEmpty(t, slugSelectedDocs)
	require.NotEmpty(t, slugOtherDocs)
	require.NotEmpty(t, selectedIDs)
	require.NotEmpty(t, otherIDs)
}

func documentPaths(documents []models.Document) []string {
	paths := make([]string, 0, len(documents))
	for _, document := range documents {
		paths = append(paths, document.FilePath)
	}
	return paths
}

func TestB078RelatedDocsDeleteValidatesNumericQualifiedSlug(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	repoDB := repository.NewDB(fixture.sqlDB)
	feature, err := repository.NewFeatureRepository(repoDB).GetByKey(context.Background(), "E10-F02")
	require.NoError(t, err)
	doc, err := fixture.documents.CreateOrGet(context.Background(), "B078 numeric slug doc", "docs/b078/feature/numeric-slug.md")
	require.NoError(t, err)
	links := repository.NewEntityDocumentRepository(repoDB)
	require.NoError(t, links.Link(context.Background(), models.EntityTypeFeature, feature.ID, doc.ID, "numeric-slug"))
	_, err = fixture.sqlDB.ExecContext(context.Background(), "UPDATE features SET slug = '123' WHERE id = ?", feature.ID)
	require.NoError(t, err)

	_, exitCode := fixture.cli.rejectionWithExit(t, "related-docs", "delete", doc.Title, "--feature=E10-F02-456")
	require.Equal(t, 1, exitCode)
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, feature.ID, doc.ID, true)

	var output map[string]string
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", doc.Title, "--feature=E10-F02-123"), &output))
	require.Equal(t, "unlinked", output["status"])
	assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, feature.ID, doc.ID, false)
}

func TestB078RelatedDocsDeleteRejectsWildcardFeatureSelectors(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	repoDB := repository.NewDB(fixture.sqlDB)
	feature, err := repository.NewFeatureRepository(repoDB).GetByKey(context.Background(), "E10-F02")
	require.NoError(t, err)
	doc, err := fixture.documents.CreateOrGet(context.Background(), "B078 wildcard selector doc", "docs/b078/feature/wildcard.md")
	require.NoError(t, err)
	links := repository.NewEntityDocumentRepository(repoDB)
	require.NoError(t, links.Link(context.Background(), models.EntityTypeFeature, feature.ID, doc.ID, "wildcard"))

	for _, selector := range []string{"F0_2", "F%2"} {
		_, exitCode := fixture.cli.rejectionWithExit(t, "related-docs", "delete", doc.Title, "--feature="+selector)
		require.Equal(t, 1, exitCode)
		assertB078Link(t, fixture.sqlDB, models.EntityTypeFeature, feature.ID, doc.ID, true)
	}
}

func TestB078RelatedDocsDeleteFeatureResolverRejectsNilContext(t *testing.T) {
	fixture := newB078IdentityFixture(t)
	repoDB := repository.NewDB(fixture.sqlDB)
	var nilContext context.Context
	resolved, err := repository.NewFeatureRepository(repoDB).ResolveDocumentUnlinkKey(nilContext, "E10-F02")
	require.Nil(t, resolved)
	require.ErrorContains(t, err, "context is required")
}
