package services_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	planhierarchyrepo "github.com/jwwelbor/shark-task-manager/internal/repository/planhierarchy"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// planEdgeWorkflowConfig is a self-contained workflow for the edge tests so
// they do not depend on fixtures owned by other test files.
const planEdgeWorkflowConfig = `{
  "task_workflow": {
    "statuses": ["todo", "in_progress", "shipped"],
    "status_flow": {"todo": ["in_progress"], "in_progress": ["shipped"], "shipped": []},
    "special_statuses": {"_start_": ["todo"], "_complete_": ["shipped"]},
    "status_metadata": {
      "todo": {"color": "gray", "phase": "planning"},
      "in_progress": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  },
  "feature_workflow": {
    "statuses": ["draft", "active", "shipped"],
    "status_flow": {"draft": ["active"], "active": ["shipped"], "shipped": []},
    "special_statuses": {"_start_": ["draft"], "_complete_": ["shipped"]},
    "status_metadata": {
      "draft": {"color": "gray", "phase": "planning"},
      "active": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  },
  "epic_workflow": {
    "statuses": ["draft", "active", "shipped"],
    "status_flow": {"draft": ["active"], "active": ["shipped"], "shipped": []},
    "special_statuses": {"_start_": ["draft"], "_complete_": ["shipped"]},
    "status_metadata": {
      "draft": {"color": "gray", "phase": "planning"},
      "active": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  }
}`

// planEdgeFixture is a real-database harness: real repositories, real
// entity_relationships rows, real registry wiring. Nothing about the edges
// under test is stubbed, so a tier that cannot actually resolve edges fails
// here instead of silently returning an empty map.
type planEdgeFixture struct {
	db      *sql.DB
	service *services.PlanHierarchyService
}

func newPlanEdgeFixture(t *testing.T) *planEdgeFixture {
	t.Helper()
	database := testutil.NewIsolatedTestDB(t)
	repoDB := repository.NewDB(database)

	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeEpic,
		services.NewEpicRepositoryAdapter(repository.NewEpicRepository(repoDB)))
	registry.Register(models.EntityTypeFeature,
		services.NewFeatureRepositoryAdapter(repository.NewFeatureRepository(repoDB)))
	taskRepo := repository.NewTaskRepository(repoDB)
	registry.Register(models.EntityTypeTask,
		services.NewTaskRepositoryAdapter(taskRepo))

	relationships := services.NewEntityRelationshipService(
		repository.NewEntityRelationshipRepository(repoDB),
		taskRepo,
	)

	tmp := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(tmp, ".sharkconfig.json"), []byte(planEdgeWorkflowConfig), 0o644,
	); err != nil {
		t.Fatalf("write workflow config: %v", err)
	}

	service := services.NewPlanHierarchyService(
		planhierarchyrepo.NewRepository(repoDB),
		workflow.NewService(tmp),
		mockPlanHierarchyClaimPolicy{},
		services.PlanHierarchyEdgeReaders{
			Relationships:    relationships,
			Registry:         registry,
			TaskDependencies: taskRepo,
		},
	)
	return &planEdgeFixture{db: database, service: service}
}

func (f *planEdgeFixture) insertEpic(t *testing.T, key string) int64 {
	t.Helper()
	result, err := f.db.Exec(
		`INSERT INTO epics (key, title, description, status, priority)
		 VALUES (?, ?, 'fixture', 'active', 'medium')`,
		key, "Epic "+key,
	)
	if err != nil {
		t.Fatalf("insert epic %s: %v", key, err)
	}
	return planEdgeLastID(t, result)
}

func (f *planEdgeFixture) insertFeature(
	t *testing.T, epicID int64, key, status string, executionOrder *int,
) int64 {
	t.Helper()
	result, err := f.db.Exec(
		`INSERT INTO features
		 (epic_id, key, title, description, status, progress_pct, execution_order)
		 VALUES (?, ?, ?, 'fixture', ?, 0, ?)`,
		epicID, key, "Feature "+key, status, executionOrder,
	)
	if err != nil {
		t.Fatalf("insert feature %s: %v", key, err)
	}
	return planEdgeLastID(t, result)
}

func (f *planEdgeFixture) insertTask(
	t *testing.T, featureID int64, key, status string, executionOrder *int, dependsOn *string,
) int64 {
	t.Helper()
	result, err := f.db.Exec(
		`INSERT INTO tasks
		 (feature_id, key, title, description, status, priority, execution_order, depends_on)
		 VALUES (?, ?, ?, 'fixture', ?, 5, ?, ?)`,
		featureID, key, "Task "+key, status, executionOrder, dependsOn,
	)
	if err != nil {
		t.Fatalf("insert task %s: %v", key, err)
	}
	return planEdgeLastID(t, result)
}

func (f *planEdgeFixture) link(
	t *testing.T,
	fromType models.EntityType, fromID int64,
	toType models.EntityType, toID int64,
	relType models.EntityRelationshipType,
) {
	t.Helper()
	if _, err := f.db.Exec(
		`INSERT INTO entity_relationships
		 (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
		 VALUES (?, ?, ?, ?, ?)`,
		fromType, fromID, toType, toID, relType,
	); err != nil {
		t.Fatalf("insert %s relationship: %v", relType, err)
	}
}

func planEdgeLastID(t *testing.T, result sql.Result) int64 {
	t.Helper()
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func planEdgeSummaries(edges []services.PlanHierarchyEdge) []string {
	summaries := make([]string, 0, len(edges))
	for _, edge := range edges {
		summaries = append(summaries, edge.Key+"|"+edge.Status+"|"+edge.Type)
	}
	return summaries
}

// TestDescribeChildEdgesTaskTierReturnsRealDependencyAndLinkEdges pins the
// task-tier contract against a real database: outgoing depends_on and incoming
// blocks land in DependsOn, incoming depends_on and outgoing blocks land in
// Blocks, everything else lands in Links, and the legacy tasks.depends_on JSON
// column is unioned in so the edge graph agrees with the dispatch filter that
// produced the candidates.
func TestDescribeChildEdgesTaskTierReturnsRealDependencyAndLinkEdges(t *testing.T) {
	fixture := newPlanEdgeFixture(t)
	epicID := fixture.insertEpic(t, "E07")
	featureID := fixture.insertFeature(t, epicID, "E07-F01", "active", nil)

	legacyJSON, err := json.Marshal([]string{"T-E07-F01-006"})
	if err != nil {
		t.Fatalf("marshal legacy dependencies: %v", err)
	}
	legacy := string(legacyJSON)

	order := 1
	candidateID := fixture.insertTask(t, featureID, "T-E07-F01-001", "todo", &order, &legacy)
	prerequisiteID := fixture.insertTask(t, featureID, "T-E07-F01-002", "shipped", &order, nil)
	blockerID := fixture.insertTask(t, featureID, "T-E07-F01-003", "in_progress", &order, nil)
	blockedID := fixture.insertTask(t, featureID, "T-E07-F01-004", "todo", &order, nil)
	dependentID := fixture.insertTask(t, featureID, "T-E07-F01-005", "todo", &order, nil)
	fixture.insertTask(t, featureID, "T-E07-F01-006", "shipped", &order, nil)

	fixture.link(t, models.EntityTypeTask, candidateID,
		models.EntityTypeTask, prerequisiteID, models.EntityRelDependsOn)
	fixture.link(t, models.EntityTypeTask, blockerID,
		models.EntityTypeTask, candidateID, models.EntityRelBlocks)
	fixture.link(t, models.EntityTypeTask, candidateID,
		models.EntityTypeTask, blockedID, models.EntityRelBlocks)
	fixture.link(t, models.EntityTypeTask, dependentID,
		models.EntityTypeTask, candidateID, models.EntityRelDependsOn)
	fixture.link(t, models.EntityTypeTask, candidateID,
		models.EntityTypeFeature, featureID, models.EntityRelRelatedTo)

	edges, err := fixture.service.DescribeChildEdges(
		context.Background(), string(models.EntityTypeTask), []string{"T-E07-F01-001"},
	)
	if err != nil {
		t.Fatalf("DescribeChildEdges() error = %v", err)
	}
	candidate, ok := edges["T-E07-F01-001"]
	if !ok {
		t.Fatalf("edges missing candidate, got keys %#v", edges)
	}

	// Statuses are reported raw — a shipped prerequisite must still appear, or
	// a consumer cannot tell a satisfied dependency from a missing one.
	if got := planEdgeSummaries(candidate.DependsOn); !reflect.DeepEqual(got, []string{
		"T-E07-F01-002|shipped|depends_on",
		"T-E07-F01-003|in_progress|blocks",
		"T-E07-F01-006|shipped|depends_on",
	}) {
		t.Fatalf("DependsOn = %#v", got)
	}
	if got := planEdgeSummaries(candidate.Blocks); !reflect.DeepEqual(got, []string{
		"T-E07-F01-004|todo|blocks",
		"T-E07-F01-005|todo|depends_on",
	}) {
		t.Fatalf("Blocks = %#v", got)
	}
	if got := planEdgeSummaries(candidate.Links); !reflect.DeepEqual(got, []string{
		"E07-F01|active|related_to",
	}) {
		t.Fatalf("Links = %#v", got)
	}
}

// TestDescribeChildEdgesFeatureTierReturnsRealFeatureEdges is the guard against
// the specific regression "task-tier edges work, feature-tier candidates come
// back edge-less". Feature-to-feature relationships are real rows in
// entity_relationships, so an epic-tier fork must report them.
func TestDescribeChildEdgesFeatureTierReturnsRealFeatureEdges(t *testing.T) {
	fixture := newPlanEdgeFixture(t)
	epicID := fixture.insertEpic(t, "E07")
	orderOne, orderTwo, orderThree := 1, 2, 3
	candidateID := fixture.insertFeature(t, epicID, "E07-F01", "active", &orderOne)
	prerequisiteID := fixture.insertFeature(t, epicID, "E07-F02", "shipped", &orderTwo)
	dependentID := fixture.insertFeature(t, epicID, "E07-F03", "draft", &orderThree)

	fixture.link(t, models.EntityTypeFeature, candidateID,
		models.EntityTypeFeature, prerequisiteID, models.EntityRelDependsOn)
	fixture.link(t, models.EntityTypeFeature, dependentID,
		models.EntityTypeFeature, candidateID, models.EntityRelDependsOn)
	fixture.link(t, models.EntityTypeFeature, candidateID,
		models.EntityTypeEpic, epicID, models.EntityRelRelatedTo)

	edges, err := fixture.service.DescribeChildEdges(
		context.Background(), string(models.EntityTypeFeature),
		[]string{"E07-F01", "E07-F02", "E07-F03"},
	)
	if err != nil {
		t.Fatalf("DescribeChildEdges() error = %v", err)
	}
	if len(edges) != 3 {
		t.Fatalf("edge entries = %d, want one per candidate: %#v", len(edges), edges)
	}
	candidate := edges["E07-F01"]
	if got := planEdgeSummaries(candidate.DependsOn); !reflect.DeepEqual(got, []string{
		"E07-F02|shipped|depends_on",
	}) {
		t.Fatalf("feature DependsOn = %#v, want real feature-tier dependency", got)
	}
	if got := planEdgeSummaries(candidate.Blocks); !reflect.DeepEqual(got, []string{
		"E07-F03|draft|depends_on",
	}) {
		t.Fatalf("feature Blocks = %#v", got)
	}
	if got := planEdgeSummaries(candidate.Links); !reflect.DeepEqual(got, []string{
		"E07|active|related_to",
	}) {
		t.Fatalf("feature Links = %#v", got)
	}
	// A candidate with no edges is still present, with empty buckets.
	if isolated := edges["E07-F03"]; len(isolated.DependsOn) != 1 || len(isolated.Blocks) != 0 {
		t.Fatalf("E07-F03 edges = %#v", isolated)
	}
}

// TestDescribeChildEdgesKeysResultsByCanonicalEntityKey guards the lookup
// contract between the two service methods: callers index edges with the key
// DescribeChildren reported. If DescribeChildEdges echoed the caller's input
// spelling instead, every lookup would miss and every candidate would appear
// edge-less.
func TestDescribeChildEdgesKeysResultsByCanonicalEntityKey(t *testing.T) {
	fixture := newPlanEdgeFixture(t)
	epicID := fixture.insertEpic(t, "E07")
	featureID := fixture.insertFeature(t, epicID, "E07-F01", "active", nil)
	order := 1
	candidateID := fixture.insertTask(t, featureID, "T-E07-F01-001", "todo", &order, nil)
	prerequisiteID := fixture.insertTask(t, featureID, "T-E07-F01-002", "shipped", &order, nil)
	fixture.link(t, models.EntityTypeTask, candidateID,
		models.EntityTypeTask, prerequisiteID, models.EntityRelDependsOn)

	ctx := context.Background()
	state, err := fixture.service.DescribeChildren(ctx, string(models.EntityTypeFeature), "E07-F01")
	if err != nil {
		t.Fatalf("DescribeChildren() error = %v", err)
	}
	if len(state.Children) == 0 {
		t.Fatal("DescribeChildren returned no claimable children")
	}
	child := state.Children[0]

	edges, err := fixture.service.DescribeChildEdges(
		ctx, string(models.EntityTypeTask), []string{child.Key},
	)
	if err != nil {
		t.Fatalf("DescribeChildEdges() error = %v", err)
	}
	found, ok := edges[child.Key]
	if !ok {
		t.Fatalf("edges not keyed by DescribeChildren key %q, got %#v", child.Key, edges)
	}
	if len(found.DependsOn) != 1 || found.DependsOn[0].Key != "T-E07-F01-002" {
		t.Fatalf("edges for %s = %#v", child.Key, found)
	}

	// A key the entity repository cannot resolve is an error, never a silently
	// edge-less entry: candidate keys must be the stored keys DescribeChildren
	// reports, not arbitrary user spellings (resolution happens upstream).
	if _, err := fixture.service.DescribeChildEdges(
		ctx, string(models.EntityTypeTask), []string{"t-e07-f01-001"},
	); err == nil {
		t.Fatal("DescribeChildEdges(unresolvable key) error = nil, want failure")
	}
}

// TestDescribeChildEdgesWithoutEdgeReadersFailsLoudly pins that a service
// constructed without edge readers errors instead of returning an empty map,
// which would be indistinguishable from "this candidate has no edges".
func TestDescribeChildEdgesWithoutEdgeReadersFailsLoudly(t *testing.T) {
	service := services.NewPlanHierarchyService(
		&mockPlanHierarchySnapshotReader{},
		newPlanEdgeWorkflow(t),
		mockPlanHierarchyClaimPolicy{},
		services.PlanHierarchyEdgeReaders{},
	)
	if _, err := service.DescribeChildEdges(
		context.Background(), string(models.EntityTypeTask), []string{"T-E07-F01-001"},
	); err == nil {
		t.Fatal("DescribeChildEdges() error = nil, want unwired-dependency error")
	}
	// No keys requested is not an error — there is nothing to under-report.
	edges, err := service.DescribeChildEdges(context.Background(), string(models.EntityTypeTask), nil)
	if err != nil {
		t.Fatalf("DescribeChildEdges(no keys) error = %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("DescribeChildEdges(no keys) = %#v, want empty", edges)
	}
}

func newPlanEdgeWorkflow(t *testing.T) *workflow.Service {
	t.Helper()
	tmp := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(tmp, ".sharkconfig.json"), []byte(planEdgeWorkflowConfig), 0o644,
	); err != nil {
		t.Fatalf("write workflow config: %v", err)
	}
	return workflow.NewService(tmp)
}
