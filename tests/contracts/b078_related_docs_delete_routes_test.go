package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/bug"
	"github.com/jwwelbor/shark-task-manager/internal/repository/changecard"
	"github.com/jwwelbor/shark-task-manager/internal/repository/question"
	"github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestB078RelatedDocsDeleteRejectedWriteHasOneDiagnosticOnAllRoutes(t *testing.T) {
	fixture := newB078RouteFixture(t)
	for index := range fixture.routes {
		route := &fixture.routes[index]
		doc, err := repository.NewDocumentRepository(fixture.repoDB).CreateOrGet(context.Background(), route.title, "docs/b078/rejected/"+route.flag+".md")
		require.NoError(t, err)
		require.NoError(t, fixture.links.Link(context.Background(), route.entityType, route.entityID, doc.ID, "specification"))
		route.documentID = doc.ID
		registerB078RouteLinkCleanup(t, fixture.sqlDB, *route)
	}
	installB078UnlinkFailureTrigger(t, fixture.sqlDB)
	for _, route := range fixture.routes {
		t.Run(route.label, func(t *testing.T) {
			for _, jsonMode := range []bool{true, false} {
				mode := "human"
				if jsonMode {
					mode = "json"
				}
				t.Run(mode, func(t *testing.T) { assertB078RejectedRouteDiagnostic(t, fixture.cli, fixture.sqlDB, route, jsonMode) })
			}
		})
	}
}

func TestB078RelatedDocsDeleteDispatchesAllSevenRoutesAndLabels(t *testing.T) {
	fixture := newB078RouteFixture(t)
	for _, route := range fixture.routes {
		t.Run(route.label, func(t *testing.T) { assertB078RouteDelete(t, fixture, route) })
	}
}

func TestB078RelatedDocsAddListDispatchesTechDebtRoute(t *testing.T) {
	fixture := newB078RouteFixture(t)
	route := routeByFlag(t, fixture.routes, "tech-debt")
	title, path := "B078 tech-debt add-list", "docs/b078/tech-debt-add-list.md"
	var response struct {
		Title     string `json:"title"`
		Path      string `json:"path"`
		LinkedTo  string `json:"linked_to"`
		ParentKey string `json:"parent_key"`
	}
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "add", title, path, "--tech-debt="+route.key), &response))
	require.Equal(t, title, response.Title)
	require.Equal(t, path, response.Path)
	require.Equal(t, "tech-debt", response.LinkedTo)
	require.Equal(t, route.key, response.ParentKey)
	require.Equal(t, fixture.cli.relatedDocIDs(t, "--tech-debt="+route.key), fixture.cli.relatedDocIDsPositional(t, route.key))
	require.NotEmpty(t, fixture.cli.relatedDocIDs(t, "--tech-debt="+route.key))
	var deleted struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", title, "--tech-debt="+route.key), &deleted))
	require.Equal(t, "unlinked", deleted.Status)
	require.Empty(t, fixture.cli.relatedDocIDs(t, "--tech-debt="+route.key))
}

type b078RelatedDocsRoute struct {
	flag, key, label, title string
	entityType              models.EntityType
	entityID                int64
	documentID              int64
}

type b078RouteFixture struct {
	cli    b078CLI
	sqlDB  *sql.DB
	repoDB *repository.DB
	links  *repository.EntityDocumentRepository
	routes []b078RelatedDocsRoute
}

func newB078RouteFixture(t *testing.T) b078RouteFixture {
	t.Helper()
	cli, sqlDB, targetTask, _ := newB078RelatedDocsDeleteFixture(t)
	repoDB := repository.NewDB(sqlDB)
	ctx := context.Background()
	var epicID, featureID int64
	require.NoError(t, sqlDB.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = 'E10'").Scan(&epicID))
	require.NoError(t, sqlDB.QueryRowContext(ctx, "SELECT id FROM features WHERE key = 'E10-F02'").Scan(&featureID))
	bugRecord := &models.Bug{BaseEntity: models.BaseEntity{Key: "B901", Title: "B078 route bug"}, Status: models.BugStatus("identified"), Severity: models.BugSeverityMedium}
	require.NoError(t, bug.NewBugRepository(repoDB).Create(ctx, bugRecord))
	changeRecord := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-901", Title: "B078 route change"}, Status: models.ChangeCardStatus("proposed"), Priority: 1}
	require.NoError(t, changecard.NewChangeCardRepository(repoDB).Create(ctx, changeRecord))
	questionRecord := &models.Question{BaseEntity: models.BaseEntity{Key: "Q901", Title: "B078 route question"}, Status: models.QuestionStatusDraft, Summary: "Verify related document command dispatch", Requester: "B078 contract"}
	require.NoError(t, question.NewQuestionRepository(repoDB).Create(ctx, questionRecord))
	techDebtRecord := createB078TechDebt(t, techdebt.NewTechDebtRepository(repoDB), "TD-901", "route")
	routes := []b078RelatedDocsRoute{
		{flag: "epic", key: "E10", label: "epic", title: "B078 route epic", entityType: models.EntityTypeEpic, entityID: epicID},
		{flag: "feature", key: "E10-F02", label: "feature", title: "B078 route feature", entityType: models.EntityTypeFeature, entityID: featureID},
		{flag: "task", key: targetTask.Key, label: "task", title: "B078 route task", entityType: models.EntityTypeTask, entityID: targetTask.ID},
		{flag: "bug", key: bugRecord.Key, label: "bug", title: bugRecord.Title, entityType: models.EntityTypeBug, entityID: bugRecord.ID},
		{flag: "change", key: changeRecord.Key, label: "change-card", title: changeRecord.Title, entityType: models.EntityTypeChange, entityID: changeRecord.ID},
		{flag: "question", key: questionRecord.Key, label: "question", title: questionRecord.Title, entityType: models.EntityTypeQuestion, entityID: questionRecord.ID},
		{flag: "tech-debt", key: techDebtRecord.Key, label: "tech-debt", title: "B078 route tech debt", entityType: models.EntityTypeTechDebt, entityID: techDebtRecord.ID},
	}
	return b078RouteFixture{cli: cli, sqlDB: sqlDB, repoDB: repoDB, links: repository.NewEntityDocumentRepository(repoDB), routes: routes}
}

func registerB078RouteLinkCleanup(t *testing.T, sqlDB *sql.DB, route b078RelatedDocsRoute) {
	t.Helper()
	t.Cleanup(func() {
		_, err := sqlDB.ExecContext(context.Background(), "DELETE FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?", route.entityType, route.entityID, route.documentID)
		assert.NoError(t, err, "remove fixture association")
	})
}

func assertB078RejectedRouteDiagnostic(t *testing.T, cli b078CLI, sqlDB *sql.DB, route b078RelatedDocsRoute, jsonMode bool) {
	t.Helper()
	stdout, stderr, exitCode := cli.runWithDatabase(cli.dbPath, jsonMode, "related-docs", "delete", route.title, "--"+route.flag+"="+route.key)
	require.Equal(t, 2, exitCode)
	message := string(stderr)
	if jsonMode {
		var response struct {
			Error   bool   `json:"error"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal(stdout, &response))
		require.True(t, response.Error)
		require.Equal(t, "COMMAND_ERROR", response.Code)
		message = response.Message
	} else {
		require.Empty(t, stdout)
	}
	publicPhrase := "failed to unlink document from " + route.label + " " + route.key
	require.Equal(t, 1, strings.Count(message, publicPhrase), message)
	require.Equal(t, 1, strings.Count(message, "failed to unlink document"), message)
	require.Equal(t, 1, strings.Count(message, route.key), message)
	require.Contains(t, message, "fixture unlink failure")
	require.NotContains(t, message, "unlinked")
	assertB078Link(t, sqlDB, route.entityType, route.entityID, route.documentID, true)
}

func assertB078RouteDelete(t *testing.T, fixture b078RouteFixture, route b078RelatedDocsRoute) {
	t.Helper()
	path := "docs/b078/routes/" + route.flag + ".md"
	doc, err := repository.NewDocumentRepository(fixture.repoDB).CreateOrGet(context.Background(), route.title, path)
	require.NoError(t, err)
	require.NoError(t, fixture.links.Link(context.Background(), route.entityType, route.entityID, doc.ID, "specification"))
	registerB078RouteLinkCleanup(t, fixture.sqlDB, b078RelatedDocsRoute{entityType: route.entityType, entityID: route.entityID, documentID: doc.ID})

	var response struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Parent string `json:"parent"`
	}
	require.NoError(t, json.Unmarshal(fixture.cli.success(t, "related-docs", "delete", route.title, "--"+route.flag+"="+route.key), &response))
	require.Equal(t, "unlinked", response.Status)
	require.Equal(t, route.title, response.Title)
	require.Equal(t, route.label, response.Parent)
	assertB078Link(t, fixture.sqlDB, route.entityType, route.entityID, doc.ID, false)
	require.Empty(t, fixture.cli.relatedDocIDs(t, "--"+route.flag+"="+route.key))
	var documentCount int
	require.NoError(t, fixture.sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM documents WHERE id = ?", doc.ID).Scan(&documentCount))
	require.Equal(t, 1, documentCount)
	stdout, stderr, exitCode := fixture.cli.runWithDatabase(fixture.cli.dbPath, false, "related-docs", "delete", route.title, "--"+route.flag+"="+route.key)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)
	require.Equal(t, "Document unlinked from "+route.label+" "+route.key+"\n", string(stdout))
}

func routeByFlag(t *testing.T, routes []b078RelatedDocsRoute, flag string) b078RelatedDocsRoute {
	t.Helper()
	for _, route := range routes {
		if route.flag == flag {
			return route
		}
	}
	require.FailNowf(t, "missing B078 route %q", flag)
	return b078RelatedDocsRoute{}
}
