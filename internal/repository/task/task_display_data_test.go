package task

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entityrel"
	"github.com/jwwelbor/shark-task-manager/internal/repository/epic"
	"github.com/jwwelbor/shark-task-manager/internal/repository/feature"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/require"
)

// relationshipEntry mirrors the JSON shape emitted by the task_display_data
// view's blocked_by_json/blocks_json/relationships_json columns. Declared
// locally (rather than importing internal/services.RelationshipWithTask) to
// avoid an import cycle: internal/services imports internal/repository (the
// aggregator), which imports this package.
type relationshipEntry struct {
	RelationshipType string `json:"relationship_type"`
	Direction        string `json:"direction"`
	TaskKey          string `json:"task_key"`
	TaskTitle        string `json:"task_title"`
	TaskStatus       string `json:"task_status"`
	EntityType       string `json:"entity_type"`
}

// TestGetTaskDisplayDataRaw_CrossEntityBlocks is the B049 regression test.
// Before the fix, task_display_data's blocks_json hardcoded
// to_entity_type = 'task', so a task blocking a feature (the exact real-world
// shape reported in B049: T-E04-F03-017 blocks E04-F04) was silently dropped
// even though it exists in entity_relationships and `shark links` shows it
// correctly. This test creates that same shape (task --blocks--> feature) and
// asserts it survives through GetTaskDisplayDataRaw's blocks_json column.
func TestGetTaskDisplayDataRaw_CrossEntityBlocks(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)

	taskRepo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)
	relRepo := entityrel.NewEntityRelationshipRepository(db)

	// Clean up before test (dedicated E94 epic/features so this test never
	// collides with fixtures used by other tests in this package).
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE (from_entity_type = 'task' AND from_entity_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E94-%'))")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E94-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E94-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E94'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E94", Title: "B049 Regression Epic"},
		Status: models.EpicStatusActive, Priority: models.PriorityHigh}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	sourceFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E94-F01", Title: "Source Feature"},
		EpicID: testEpic.ID, Status: models.FeatureStatusDraft}
	require.NoError(t, featureRepo.Create(ctx, sourceFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", sourceFeature.ID) }()

	// The feature this task will block -- the cross-entity-type edge B049 reports missing.
	targetFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E94-F02", Title: "Target Feature"},
		EpicID: testEpic.ID, Status: models.FeatureStatusDraft}
	require.NoError(t, featureRepo.Create(ctx, targetFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", targetFeature.ID) }()

	// A second task, used for a task-to-task "related_to" relationship -- a
	// non-blocking relationship type that must surface in relationships_json
	// but must NOT appear in blocked_by_json/blocks_json.
	otherTask := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E94-F01-002", Title: "Other Task"},
		FeatureID: sourceFeature.ID, Status: "todo", Priority: 5}
	require.NoError(t, taskRepo.Create(ctx, otherTask))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", otherTask.ID) }()

	mainTask := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E94-F01-001", Title: "Main Task"},
		FeatureID: sourceFeature.ID, Status: "todo", Priority: 5}
	require.NoError(t, taskRepo.Create(ctx, mainTask))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", mainTask.ID) }()

	// mainTask --blocks--> targetFeature (cross-entity, matches B049's repro).
	blocksRel := &models.EntityRelationship{
		FromEntityType: models.EntityTypeTask, FromEntityID: mainTask.ID,
		ToEntityType: models.EntityTypeFeature, ToEntityID: targetFeature.ID,
		RelationshipType: models.EntityRelBlocks,
	}
	require.NoError(t, relRepo.Create(ctx, blocksRel))
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE id = ?", blocksRel.ID)
	}()

	// mainTask --related_to--> otherTask (same entity type, non-blocking relationship type).
	relatedRel := &models.EntityRelationship{
		FromEntityType: models.EntityTypeTask, FromEntityID: mainTask.ID,
		ToEntityType: models.EntityTypeTask, ToEntityID: otherTask.ID,
		RelationshipType: models.EntityRelRelatedTo,
	}
	require.NoError(t, relRepo.Create(ctx, relatedRel))
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE id = ?", relatedRel.ID)
	}()

	// mainTask --depends_on--> testEpic (cross-entity, exercises blocked_by_json's
	// generalized entity-type join with a *third* entity type, not just feature).
	dependsOnRel := &models.EntityRelationship{
		FromEntityType: models.EntityTypeTask, FromEntityID: mainTask.ID,
		ToEntityType: models.EntityTypeEpic, ToEntityID: testEpic.ID,
		RelationshipType: models.EntityRelDependsOn,
	}
	require.NoError(t, relRepo.Create(ctx, dependsOnRel))
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE id = ?", dependsOnRel.ID)
	}()

	raw, err := taskRepo.GetTaskDisplayDataRaw(ctx, mainTask.ID)
	require.NoError(t, err)

	var blockedBy []relationshipEntry
	require.NoError(t, json.Unmarshal([]byte(raw.BlockedByJSON), &blockedBy))
	require.Len(t, blockedBy, 1, "expected the cross-entity task->epic depends_on relationship to survive blocked_by_json")
	require.Equal(t, "E94", blockedBy[0].TaskKey)
	require.Equal(t, "B049 Regression Epic", blockedBy[0].TaskTitle)
	require.Equal(t, "epic", blockedBy[0].EntityType, "blocked_by_json must expose the related entity's type for cross-entity entries")
	require.Equal(t, "depends_on", blockedBy[0].RelationshipType)
	require.Equal(t, "outgoing", blockedBy[0].Direction)

	var blocks []relationshipEntry
	require.NoError(t, json.Unmarshal([]byte(raw.BlocksJSON), &blocks))
	require.Len(t, blocks, 1, "expected the cross-entity task->feature blocks relationship to survive blocks_json")
	require.Equal(t, "E94-F02", blocks[0].TaskKey)
	require.Equal(t, "Target Feature", blocks[0].TaskTitle)
	require.Equal(t, "feature", blocks[0].EntityType, "blocks_json must expose the related entity's type for cross-entity entries")
	require.Equal(t, "blocks", blocks[0].RelationshipType)
	require.Equal(t, "outgoing", blocks[0].Direction)

	// The related_to relationship must not leak into blocks_json/blocked_by_json --
	// blocked_by/blocks keep their existing depends_on/blocks-only semantics
	// so a merely-related entity never masquerades as a blocker.
	for _, b := range blocks {
		require.NotEqual(t, "related_to", b.RelationshipType)
	}

	var relationships []relationshipEntry
	require.NoError(t, json.Unmarshal([]byte(raw.RelationshipsJSON), &relationships))
	require.Len(t, relationships, 3, "relationships_json must surface every relationship type/entity type, not just depends_on/blocks")

	foundBlocks, foundRelated, foundDependsOn := false, false, false
	for _, r := range relationships {
		switch r.RelationshipType {
		case "blocks":
			foundBlocks = true
			require.Equal(t, "feature", r.EntityType)
			require.Equal(t, "E94-F02", r.TaskKey)
		case "related_to":
			foundRelated = true
			require.Equal(t, "task", r.EntityType)
			require.Equal(t, "T-E94-F01-002", r.TaskKey)
		case "depends_on":
			foundDependsOn = true
			require.Equal(t, "epic", r.EntityType)
			require.Equal(t, "E94", r.TaskKey)
		}
	}
	require.True(t, foundBlocks, "relationships_json missing the cross-entity blocks relationship")
	require.True(t, foundRelated, "relationships_json missing the non-blocking related_to relationship")
	require.True(t, foundDependsOn, "relationships_json missing the cross-entity depends_on relationship")
}
