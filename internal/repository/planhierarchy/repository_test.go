package planhierarchy_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	planhierarchyrepo "github.com/jwwelbor/shark-task-manager/internal/repository/planhierarchy"
	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
)

func TestReadDirectChildrenEpicReturnsOrderedFeaturesAndClaimState(t *testing.T) {
	database := testutil.NewIsolatedTestDB(t)
	epicID := insertEpic(t, database, "E07")
	orderOne, orderTwo := 1, 2
	insertFeature(t, database, epicID, "E07-F02", "Second", &orderTwo)
	insertFeature(t, database, epicID, "E07-F01", "First", &orderOne)
	insertFeature(t, database, epicID, "E07-F03", "Unordered", nil)

	now := time.Now().UTC()
	insertClaim(t, database, models.EntityTypeFeature, "E07-F02", now)
	insertClaim(t, database, models.EntityTypeFeature, "E07-F03", now.Add(-30*time.Minute))

	repo := planhierarchyrepo.NewRepository(dbconn.NewDB(database))
	snapshot, err := repo.ReadDirectChildren(
		context.Background(),
		string(models.EntityTypeEpic),
		"E07",
		15*time.Minute,
		now,
	)
	if err != nil {
		t.Fatalf("ReadDirectChildren() error = %v", err)
	}
	if !snapshot.ParentFound {
		t.Fatal("ParentFound = false, want true")
	}
	if got := childKeys(snapshot.Children); !reflect.DeepEqual(got, []string{
		"E07-F01", "E07-F02", "E07-F03",
	}) {
		t.Fatalf("child keys = %#v, want execution order then unordered", got)
	}
	if got := childClaimFlags(snapshot.Children); !reflect.DeepEqual(got, []bool{
		false, true, false,
	}) {
		t.Fatalf("claim flags = %#v, want unclaimed/fresh/expired", got)
	}

	neverExpires, err := repo.ReadDirectChildren(
		context.Background(),
		string(models.EntityTypeEpic),
		"E07",
		0,
		now,
	)
	if err != nil {
		t.Fatalf("ReadDirectChildren(TTL=0) error = %v", err)
	}
	if got := childClaimFlags(neverExpires.Children); !reflect.DeepEqual(got, []bool{
		false, true, true,
	}) {
		t.Fatalf("claim flags with TTL=0 = %#v, want all persisted claims active", got)
	}
}

func TestReadDirectChildrenFeatureReturnsTasksAndAllDependencySources(t *testing.T) {
	database := testutil.NewIsolatedTestDB(t)
	epicID := insertEpic(t, database, "E07")
	featureID := insertFeature(t, database, epicID, "E07-F01", "Feature", nil)
	orderOne, orderTwo := 1, 2
	dependencyID := insertTask(t, database, featureID, "T-E07-F01-001", "Dependency", "completed", &orderOne, 1, nil)
	legacyJSON, err := json.Marshal([]string{"T-E07-F01-001"})
	if err != nil {
		t.Fatalf("marshal legacy dependencies: %v", err)
	}
	legacy := string(legacyJSON)
	insertTask(t, database, featureID, "T-E07-F01-002", "Legacy dependent", "todo", &orderTwo, 4, &legacy)
	relationshipTaskID := insertTask(
		t,
		database,
		featureID,
		"T-E07-F01-003",
		"Relationship dependent",
		"todo",
		&orderTwo,
		5,
		nil,
	)
	if _, err := database.Exec(
		`INSERT INTO entity_relationships
		 (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
		 VALUES (?, ?, ?, ?, ?)`,
		models.EntityTypeTask,
		relationshipTaskID,
		models.EntityTypeTask,
		dependencyID,
		models.EntityRelDependsOn,
	); err != nil {
		t.Fatalf("insert task dependency relationship: %v", err)
	}

	now := time.Now().UTC()
	insertClaim(t, database, models.EntityTypeTask, "T-E07-F01-002", now)

	repo := planhierarchyrepo.NewRepository(dbconn.NewDB(database))
	snapshot, err := repo.ReadDirectChildren(
		context.Background(),
		string(models.EntityTypeFeature),
		"E07-F01",
		15*time.Minute,
		now,
	)
	if err != nil {
		t.Fatalf("ReadDirectChildren() error = %v", err)
	}
	if got := childKeys(snapshot.Children); !reflect.DeepEqual(got, []string{
		"T-E07-F01-001", "T-E07-F01-002", "T-E07-F01-003",
	}) {
		t.Fatalf("child keys = %#v, want stable execution/priority order", got)
	}
	if got := childClaimFlags(snapshot.Children); !reflect.DeepEqual(got, []bool{
		false, true, false,
	}) {
		t.Fatalf("task claim flags = %#v, want only T-E07-F01-002 claimed", got)
	}
	for _, index := range []int{1, 2} {
		dependencies := snapshot.Children[index].Dependencies
		if len(dependencies) != 1 ||
			dependencies[0].Key != "T-E07-F01-001" ||
			dependencies[0].Status != "completed" {
			t.Fatalf("dependencies for %s = %#v", snapshot.Children[index].Key, dependencies)
		}
	}
}

func TestReadDirectChildrenMissingParent(t *testing.T) {
	database := testutil.NewIsolatedTestDB(t)
	repo := planhierarchyrepo.NewRepository(dbconn.NewDB(database))

	snapshot, err := repo.ReadDirectChildren(
		context.Background(),
		string(models.EntityTypeEpic),
		"E99",
		15*time.Minute,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("ReadDirectChildren() error = %v", err)
	}
	if snapshot.ParentFound || snapshot.Children == nil || len(snapshot.Children) != 0 {
		t.Fatalf("snapshot = %#v, want allocated missing-parent snapshot", snapshot)
	}
}

func TestReadDirectChildrenExistingLeafReportsParentFound(t *testing.T) {
	database := testutil.NewIsolatedTestDB(t)
	epicID := insertEpic(t, database, "E07")
	featureID := insertFeature(t, database, epicID, "E07-F01", "Feature", nil)
	insertTask(t, database, featureID, "T-E07-F01-001", "Leaf", "development", nil, 5, nil)
	repo := planhierarchyrepo.NewRepository(dbconn.NewDB(database))

	snapshot, err := repo.ReadDirectChildren(context.Background(), string(models.EntityTypeTask), "T-E07-F01-001", 15*time.Minute, time.Now().UTC())
	if err != nil {
		t.Fatalf("ReadDirectChildren() error = %v", err)
	}
	if !snapshot.ParentFound || len(snapshot.Children) != 0 {
		t.Fatalf("snapshot = %#v, want existing leaf with no children", snapshot)
	}
}

func insertEpic(t *testing.T, database *sql.DB, key string) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO epics (key, title, description, status, priority)
		 VALUES (?, ?, 'fixture', 'active', 'medium')`,
		key,
		"Epic "+key,
	)
	if err != nil {
		t.Fatalf("insert epic %s: %v", key, err)
	}
	return lastInsertID(t, result)
}

func insertFeature(
	t *testing.T,
	database *sql.DB,
	epicID int64,
	key, title string,
	executionOrder *int,
) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO features
		 (epic_id, key, title, description, status, progress_pct, execution_order)
		 VALUES (?, ?, ?, 'fixture', 'active', 0, ?)`,
		epicID,
		key,
		title,
		executionOrder,
	)
	if err != nil {
		t.Fatalf("insert feature %s: %v", key, err)
	}
	return lastInsertID(t, result)
}

func insertTask(
	t *testing.T,
	database *sql.DB,
	featureID int64,
	key, title, status string,
	executionOrder *int,
	priority int,
	dependsOn *string,
) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO tasks
		 (feature_id, key, title, description, status, priority, execution_order, depends_on)
		 VALUES (?, ?, ?, 'fixture', ?, ?, ?, ?)`,
		featureID,
		key,
		title,
		status,
		priority,
		executionOrder,
		dependsOn,
	)
	if err != nil {
		t.Fatalf("insert task %s: %v", key, err)
	}
	return lastInsertID(t, result)
}

func lastInsertID(t *testing.T, result sql.Result) int64 {
	t.Helper()
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert ID: %v", err)
	}
	return id
}

func insertClaim(
	t *testing.T,
	database *sql.DB,
	entityType models.EntityType,
	entityKey string,
	heartbeat time.Time,
) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT INTO entity_claims
		 (entity_type, entity_key, claimed_by, session_id, claimed_at, last_heartbeat)
		 VALUES (?, ?, 'worker', ?, ?, ?)`,
		entityType,
		entityKey,
		"session-"+entityKey,
		dbconn.FormatTime(heartbeat),
		dbconn.FormatTime(heartbeat),
	); err != nil {
		t.Fatalf("insert %s %s claim: %v", entityType, entityKey, err)
	}
}

func childKeys(children []planhierarchyrepo.Child) []string {
	keys := make([]string, 0, len(children))
	for _, child := range children {
		keys = append(keys, child.Key)
	}
	return keys
}

func childClaimFlags(children []planhierarchyrepo.Child) []bool {
	flags := make([]bool, 0, len(children))
	for _, child := range children {
		flags = append(flags, child.Claimed)
	}
	return flags
}
