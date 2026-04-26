// Package tag_test provides integration tests for EntityTagRepository.
// Per .claude/rules/testing/repository-tests.md, repository tests use the real test database.
// Each test uses test.NewIsolatedTestDB(t) for a fresh, independent database — no
// DELETE-before boilerplate needed; isolation is guaranteed by the helper.
package tag_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/epic"
	"github.com/jwwelbor/shark-task-manager/internal/repository/tag"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// setupEntityTagRepo creates both TagRepository and EntityTagRepository backed
// by the same isolated test database. Returns both repos plus the DB wrapper
// for raw SQL if needed.
func setupEntityTagRepo(t *testing.T) (*tag.TagRepository, *tag.EntityTagRepository, *dbconn.DB) {
	t.Helper()
	database := test.NewIsolatedTestDB(t)
	db := dbconn.NewDB(database)
	tagRepo := tag.NewTagRepository(db)
	entityTagRepo := tag.NewEntityTagRepository(db)
	return tagRepo, entityTagRepo, db
}

// insertTestEpic inserts a minimal epic row via raw SQL and returns its id.
// Used when the test only needs an entity row to attach tags against —
// not when testing cascade-delete behaviour (use the real EpicRepository for that).
func insertTestEpic(t *testing.T, db *dbconn.DB, key string) int64 {
	t.Helper()
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES (?, 'Test Epic', 'test', 'active', 'high')
	`, key)
	if err != nil {
		t.Fatalf("insertTestEpic: failed to insert epic %q: %v", key, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("insertTestEpic: LastInsertId: %v", err)
	}
	return id
}

// ─── EntityTagRepository: Attach ─────────────────────────────────────────────

// AC-T3: Attach links tag to entity.
func TestEntityTagRepository_Attach(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epicID := insertTestEpic(t, db, "E76")
	t1, err := tagRepo.Create(ctx, "attach-test")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("Attach() unexpected error: %v", err)
	}

	// Verify the link exists.
	links, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() after Attach: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("expected 1 link after Attach, got %d", len(links))
	}
}

// AC-T3: Duplicate Attach is idempotent — the same (entity_type, entity_id, tag_id)
// combination inserted twice must not cause an error and must not create a second row.
func TestEntityTagRepository_Attach_Idempotent(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epicID := insertTestEpic(t, db, "E75")
	t1, err := tagRepo.Create(ctx, "idempotent-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("first Attach() error: %v", err)
	}
	// Second attach — must be a no-op (INSERT OR IGNORE semantics).
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("second Attach() should be idempotent, got error: %v", err)
	}

	links, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() after idempotent Attach: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("after idempotent Attach, expected 1 link, got %d", len(links))
	}
}

// ─── EntityTagRepository: Detach ─────────────────────────────────────────────

// AC-T3: Detach removes one link.
func TestEntityTagRepository_Detach(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epicID := insertTestEpic(t, db, "E74")
	t1, err := tagRepo.Create(ctx, "detach-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("Attach() error: %v", err)
	}

	if err := entityTagRepo.Detach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("Detach() unexpected error: %v", err)
	}

	links, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() after Detach: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("after Detach, expected 0 links, got %d", len(links))
	}
}

// Detaching a non-existent link should be a no-op (not an error).
func TestEntityTagRepository_Detach_NotFound_NoError(t *testing.T) {
	_, entityTagRepo, _ := setupEntityTagRepo(t)
	ctx := context.Background()

	if err := entityTagRepo.Detach(ctx, models.EntityTypeEpic, 99999, 99999); err != nil {
		t.Fatalf("Detach() on non-existent link should not error, got: %v", err)
	}
}

// ─── EntityTagRepository: ListByEntity ───────────────────────────────────────

// AC-T3: ListByEntity returns all tags for an entity.
func TestEntityTagRepository_ListByEntity(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epicID := insertTestEpic(t, db, "E73")

	t1, err := tagRepo.Create(ctx, "tag-a")
	if err != nil {
		t.Fatalf("Create(tag-a) error: %v", err)
	}
	t2, err := tagRepo.Create(ctx, "tag-b")
	if err != nil {
		t.Fatalf("Create(tag-b) error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("Attach(tag-a) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t2.ID); err != nil {
		t.Fatalf("Attach(tag-b) error: %v", err)
	}

	links, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() error: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("ListByEntity() returned %d links, want 2", len(links))
	}
}

func TestEntityTagRepository_ListByEntity_Empty(t *testing.T) {
	_, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epicID := insertTestEpic(t, db, "E72")

	links, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() error: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("ListByEntity() for entity with no tags: got %d, want 0", len(links))
	}
}

// ─── EntityTagRepository: ListByTag ──────────────────────────────────────────

func TestEntityTagRepository_ListByTag(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epic1ID := insertTestEpic(t, db, "E71")
	epic2ID := insertTestEpic(t, db, "E70")

	sharedTag, err := tagRepo.Create(ctx, "shared-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epic1ID, sharedTag.ID); err != nil {
		t.Fatalf("Attach(epic1) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epic2ID, sharedTag.ID); err != nil {
		t.Fatalf("Attach(epic2) error: %v", err)
	}

	links, err := entityTagRepo.ListByTag(ctx, sharedTag.ID)
	if err != nil {
		t.Fatalf("ListByTag() error: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("ListByTag() returned %d links, want 2", len(links))
	}
}

// ─── EntityTagRepository: ListByEntityType ───────────────────────────────────

func TestEntityTagRepository_ListByEntityType(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epic1ID := insertTestEpic(t, db, "E69")
	epic2ID := insertTestEpic(t, db, "E68")

	sharedTag, err := tagRepo.Create(ctx, "type-filter-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epic1ID, sharedTag.ID); err != nil {
		t.Fatalf("Attach(epic1) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epic2ID, sharedTag.ID); err != nil {
		t.Fatalf("Attach(epic2) error: %v", err)
	}

	links, err := entityTagRepo.ListByEntityType(ctx, models.EntityTypeEpic, sharedTag.ID)
	if err != nil {
		t.Fatalf("ListByEntityType() error: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("ListByEntityType() returned %d links, want 2", len(links))
	}
}

// ─── EntityTagRepository: CountByTag ─────────────────────────────────────────

// AC-T3: CountByTag returns accurate count.
func TestEntityTagRepository_CountByTag(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epic1ID := insertTestEpic(t, db, "E67")
	epic2ID := insertTestEpic(t, db, "E66")

	counted, err := tagRepo.Create(ctx, "count-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	other, err := tagRepo.Create(ctx, "other-tag")
	if err != nil {
		t.Fatalf("Create(other) error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epic1ID, counted.ID); err != nil {
		t.Fatalf("Attach(epic1, counted) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epic2ID, counted.ID); err != nil {
		t.Fatalf("Attach(epic2, counted) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epic1ID, other.ID); err != nil {
		t.Fatalf("Attach(epic1, other) error: %v", err)
	}

	count, err := entityTagRepo.CountByTag(ctx, counted.ID)
	if err != nil {
		t.Fatalf("CountByTag() error: %v", err)
	}
	if count != 2 {
		t.Errorf("CountByTag() = %d, want 2", count)
	}

	// CountByTag for a non-existent tag must return 0, not an error.
	zeroCount, err := entityTagRepo.CountByTag(ctx, 99999)
	if err != nil {
		t.Fatalf("CountByTag(nonexistent) error: %v", err)
	}
	if zeroCount != 0 {
		t.Errorf("CountByTag(nonexistent) = %d, want 0", zeroCount)
	}
}

// ─── EntityTagRepository: Polymorphic entity types ───────────────────────────

// Verify the entity_type CHECK constraint allows valid types defined in ADR-10.
func TestEntityTagRepository_PolymorphicEntityTypes(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	epicID := insertTestEpic(t, db, "E65")

	t1, err := tagRepo.Create(ctx, "poly-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("Attach(epic) error: %v", err)
	}

	links, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() error: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("expected 1 link, got %d", len(links))
	}
	if links[0].EntityType != models.EntityTypeEpic {
		t.Errorf("link.EntityType = %q, want %q", links[0].EntityType, models.EntityTypeEpic)
	}
	if links[0].TagID != t1.ID {
		t.Errorf("link.TagID = %d, want %d", links[0].TagID, t1.ID)
	}
}

// ─── AC-T4: Cascade-delete trigger via real EpicRepository ───────────────────

// AC-T4: Deleting a parent entity via the real EpicRepository.Delete() causes
// entity_tags rows for that entity to be removed by the cascade-delete trigger
// defined in the migration (architecture.md §3.2).
//
// This test exercises the DB trigger directly — something a mock cannot do.
// It also exercises the end-to-end integration between EpicRepository and the
// tag layer via a real DELETE on the epics table.
func TestEntityTagRepository_CascadeDeleteOnEpicDelete(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	// Create a real epic via EpicRepository so the DELETE path matches production.
	epicRepo := epic.NewEpicRepository(db)
	e := &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:   "E64",
			Title: "Cascade Test Epic",
		},
		Status:   models.EpicStatusActive,
		Priority: models.PriorityHigh,
	}
	if err := epicRepo.Create(ctx, e); err != nil {
		t.Fatalf("EpicRepository.Create() error: %v", err)
	}
	if e.ID == 0 {
		t.Fatal("EpicRepository.Create() did not populate ID")
	}

	// Create a tag and attach it to the new epic.
	tg, err := tagRepo.Create(ctx, "cascade-test-tag")
	if err != nil {
		t.Fatalf("TagRepository.Create() error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, e.ID, tg.ID); err != nil {
		t.Fatalf("EntityTagRepository.Attach() error: %v", err)
	}

	// Confirm the entity_tag row exists before the delete.
	linksBefore, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, e.ID)
	if err != nil {
		t.Fatalf("ListByEntity() before delete: %v", err)
	}
	if len(linksBefore) != 1 {
		t.Fatalf("expected 1 entity_tag row before delete, got %d", len(linksBefore))
	}

	// Delete the epic via EpicRepository — this should fire the cascade trigger.
	if err := epicRepo.Delete(ctx, e.ID); err != nil {
		t.Fatalf("EpicRepository.Delete() error: %v", err)
	}

	// Assert the entity_tag row is gone — fired by the DB trigger, not application code.
	linksAfter, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, e.ID)
	if err != nil {
		t.Fatalf("ListByEntity() after delete: %v", err)
	}
	if len(linksAfter) != 0 {
		t.Errorf("cascade-delete trigger failed: expected 0 entity_tag rows after epic delete, got %d",
			len(linksAfter))
	}
}

// ─── Helpers for FilterEntityIDs / ListTagNamesByEntities tests ──────────────

// insertTestFeature inserts a minimal feature row and returns its id.
func insertTestFeature(t *testing.T, db *dbconn.DB, epicID int64, key string) int64 {
	t.Helper()
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO features (epic_id, key, title, status)
		VALUES (?, ?, 'Test Feature', 'active')
	`, epicID, key)
	if err != nil {
		t.Fatalf("insertTestFeature: failed to insert feature %q: %v", key, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("insertTestFeature: LastInsertId: %v", err)
	}
	return id
}

// insertTestTask inserts a minimal task row (requires a feature) and returns its id.
func insertTestTask(t *testing.T, db *dbconn.DB, featureID int64, key string) int64 {
	t.Helper()
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO tasks (feature_id, key, title, status, priority, depends_on)
		VALUES (?, ?, 'Test Task', 'todo', 5, '[]')
	`, featureID, key)
	if err != nil {
		t.Fatalf("insertTestTask: failed to insert task %q: %v", key, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("insertTestTask: LastInsertId: %v", err)
	}
	return id
}

// setupTaskSetup creates an epic + feature + N tasks; returns featureID and taskIDs.
// epicKey must be unique per test; tasks are keyed epicKey-F01-001, -002, etc.
func setupTaskSetup(t *testing.T, db *dbconn.DB, epicKey string, count int) (int64, []int64) {
	t.Helper()
	epicID := insertTestEpic(t, db, epicKey)
	featureKey := epicKey + "-F01"
	featureID := insertTestFeature(t, db, epicID, featureKey)
	ids := make([]int64, count)
	for i := 0; i < count; i++ {
		taskKey := featureKey + "-" + fmt.Sprintf("%03d", i+1)
		ids[i] = insertTestTask(t, db, featureID, taskKey)
	}
	return featureID, ids
}

// ─── FilterEntityIDs tests ───────────────────────────────────────────────────

// AC-7a: Single tag, two matching entities.
func TestFilterEntityIDs_SingleTagSingleMatch(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	_, taskIDs := setupTaskSetup(t, db, "E55", 3)
	t1, err := tagRepo.Create(ctx, "filter-single")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Attach t1 to task IDs [0] and [1], not [2].
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[0], t1.ID); err != nil {
		t.Fatalf("Attach(0) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[1], t1.ID); err != nil {
		t.Fatalf("Attach(1) error: %v", err)
	}

	got, err := entityTagRepo.FilterEntityIDs(ctx, models.EntityTypeTask, []int64{t1.ID})
	if err != nil {
		t.Fatalf("FilterEntityIDs() error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("FilterEntityIDs() returned %d IDs, want 2: %v", len(got), got)
	}
	if got[0] != taskIDs[0] || got[1] != taskIDs[1] {
		t.Errorf("FilterEntityIDs() = %v, want %v", got, []int64{taskIDs[0], taskIDs[1]})
	}
}

// AC-7b: Two tags, AND intersection — only entity with both tags is returned.
func TestFilterEntityIDs_TwoTagsBothRequired(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	_, taskIDs := setupTaskSetup(t, db, "E54", 4)
	t1, err := tagRepo.Create(ctx, "filter-t1")
	if err != nil {
		t.Fatalf("Create(t1) error: %v", err)
	}
	t2, err := tagRepo.Create(ctx, "filter-t2")
	if err != nil {
		t.Fatalf("Create(t2) error: %v", err)
	}

	// task[0] has both t1 and t2.
	// task[1], task[2] have only t1.
	// task[3] has only t2.
	for _, id := range []int64{taskIDs[0], taskIDs[1], taskIDs[2]} {
		if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, id, t1.ID); err != nil {
			t.Fatalf("Attach(t1) error: %v", err)
		}
	}
	for _, id := range []int64{taskIDs[0], taskIDs[3]} {
		if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, id, t2.ID); err != nil {
			t.Fatalf("Attach(t2) error: %v", err)
		}
	}

	got, err := entityTagRepo.FilterEntityIDs(ctx, models.EntityTypeTask, []int64{t1.ID, t2.ID})
	if err != nil {
		t.Fatalf("FilterEntityIDs() error: %v", err)
	}

	if len(got) != 1 || got[0] != taskIDs[0] {
		t.Errorf("FilterEntityIDs() = %v, want [%d]", got, taskIDs[0])
	}
}

// AC-7c: No entity has both tags — returns non-nil empty slice, no error.
func TestFilterEntityIDs_NoMatchReturnsEmpty(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	_, taskIDs := setupTaskSetup(t, db, "E53", 2)
	t1, err := tagRepo.Create(ctx, "nomatch-t1")
	if err != nil {
		t.Fatalf("Create(t1) error: %v", err)
	}
	t2, err := tagRepo.Create(ctx, "nomatch-t2")
	if err != nil {
		t.Fatalf("Create(t2) error: %v", err)
	}

	// task[0] has only t1; task[1] has only t2.
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[0], t1.ID); err != nil {
		t.Fatalf("Attach(t1) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[1], t2.ID); err != nil {
		t.Fatalf("Attach(t2) error: %v", err)
	}

	got, err := entityTagRepo.FilterEntityIDs(ctx, models.EntityTypeTask, []int64{t1.ID, t2.ID})
	if err != nil {
		t.Fatalf("FilterEntityIDs() error: %v", err)
	}
	if got == nil {
		t.Error("FilterEntityIDs() returned nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("FilterEntityIDs() = %v, want []", got)
	}
}

// AC-7d: EntityType scoping — the same tag attached to both a task and an epic
// must not allow cross-type contamination. FilterEntityIDs(EntityTypeTask) must
// return only the task entity_id, and FilterEntityIDs(EntityTypeEpic) must
// return only the epic entity_id. We verify both directions to confirm the
// WHERE entity_type = ? filter is applied correctly.
func TestFilterEntityIDs_EntityTypeScopedCorrectly(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	// Create an epic and a task. In a fresh isolated DB their IDs may coincide
	// (both could be 1); we handle this below by checking each direction
	// independently rather than comparing IDs across entity types.
	epicID := insertTestEpic(t, db, "E52")
	featureID := insertTestFeature(t, db, epicID, "E52-F01")
	taskID := insertTestTask(t, db, featureID, "E52-F01-001")

	sharedTag, err := tagRepo.Create(ctx, "scope-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Attach the same tag to both entity types.
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskID, sharedTag.ID); err != nil {
		t.Fatalf("Attach(task) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, sharedTag.ID); err != nil {
		t.Fatalf("Attach(epic) error: %v", err)
	}

	// EntityTypeTask filter must return exactly taskID.
	gotTask, err := entityTagRepo.FilterEntityIDs(ctx, models.EntityTypeTask, []int64{sharedTag.ID})
	if err != nil {
		t.Fatalf("FilterEntityIDs(EntityTypeTask) error: %v", err)
	}
	if len(gotTask) != 1 || gotTask[0] != taskID {
		t.Errorf("FilterEntityIDs(EntityTypeTask) = %v, want [%d]", gotTask, taskID)
	}

	// EntityTypeEpic filter must return exactly epicID.
	gotEpic, err := entityTagRepo.FilterEntityIDs(ctx, models.EntityTypeEpic, []int64{sharedTag.ID})
	if err != nil {
		t.Fatalf("FilterEntityIDs(EntityTypeEpic) error: %v", err)
	}
	if len(gotEpic) != 1 || gotEpic[0] != epicID {
		t.Errorf("FilterEntityIDs(EntityTypeEpic) = %v, want [%d]", gotEpic, epicID)
	}
}

// AC-8: Empty tagIDs returns ErrEmptyTagIDs.
func TestFilterEntityIDs_EmptySliceReturnsErrEmptyTagIDs(t *testing.T) {
	_, entityTagRepo, _ := setupEntityTagRepo(t)
	ctx := context.Background()

	_, err := entityTagRepo.FilterEntityIDs(ctx, models.EntityTypeTask, []int64{})
	if err == nil {
		t.Fatal("FilterEntityIDs(empty) expected error, got nil")
	}
	if !errors.Is(err, tag.ErrEmptyTagIDs) {
		t.Errorf("FilterEntityIDs(empty) error = %v, want ErrEmptyTagIDs", err)
	}
}

// AC-8b: Three tags AND — only entity with all three is returned.
func TestFilterEntityIDs_ThreeTagsAnd(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	_, taskIDs := setupTaskSetup(t, db, "E51", 5)

	t1, _ := tagRepo.Create(ctx, "3tag-t1")
	t2, _ := tagRepo.Create(ctx, "3tag-t2")
	t3, _ := tagRepo.Create(ctx, "3tag-t3")

	// task[0] gets all three tags.
	for _, tagID := range []int64{t1.ID, t2.ID, t3.ID} {
		if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[0], tagID); err != nil {
			t.Fatalf("Attach(task[0]) error: %v", err)
		}
	}
	// task[1]: t1 and t2 only.
	for _, tagID := range []int64{t1.ID, t2.ID} {
		if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[1], tagID); err != nil {
			t.Fatalf("Attach(task[1]) error: %v", err)
		}
	}
	// task[2]: t2 and t3 only.
	for _, tagID := range []int64{t2.ID, t3.ID} {
		if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[2], tagID); err != nil {
			t.Fatalf("Attach(task[2]) error: %v", err)
		}
	}

	got, err := entityTagRepo.FilterEntityIDs(ctx, models.EntityTypeTask, []int64{t1.ID, t2.ID, t3.ID})
	if err != nil {
		t.Fatalf("FilterEntityIDs(3 tags) error: %v", err)
	}

	if len(got) != 1 || got[0] != taskIDs[0] {
		t.Errorf("FilterEntityIDs(3 tags) = %v, want [%d]", got, taskIDs[0])
	}
}

// ─── ListTagNamesByEntities tests ────────────────────────────────────────────

// TestListTagNamesByEntities_SingleEntity: one task with two tags.
func TestListTagNamesByEntities_SingleEntity(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	_, taskIDs := setupTaskSetup(t, db, "E50", 1)
	tAuth, err := tagRepo.Create(ctx, "auth")
	if err != nil {
		t.Fatalf("Create(auth) error: %v", err)
	}
	tVoice, err := tagRepo.Create(ctx, "voice")
	if err != nil {
		t.Fatalf("Create(voice) error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[0], tAuth.ID); err != nil {
		t.Fatalf("Attach(auth) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[0], tVoice.ID); err != nil {
		t.Fatalf("Attach(voice) error: %v", err)
	}

	rows, err := entityTagRepo.ListTagNamesByEntities(ctx, models.EntityTypeTask, []int64{taskIDs[0]})
	if err != nil {
		t.Fatalf("ListTagNamesByEntities() error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("ListTagNamesByEntities() returned %d rows, want 2", len(rows))
	}
	// Rows are ordered by entity_id ASC, name ASC.
	if rows[0].EntityID != taskIDs[0] || rows[0].TagName != "auth" {
		t.Errorf("rows[0] = {%d, %q}, want {%d, %q}", rows[0].EntityID, rows[0].TagName, taskIDs[0], "auth")
	}
	if rows[1].EntityID != taskIDs[0] || rows[1].TagName != "voice" {
		t.Errorf("rows[1] = {%d, %q}, want {%d, %q}", rows[1].EntityID, rows[1].TagName, taskIDs[0], "voice")
	}
}

// TestListTagNamesByEntities_MultipleEntities: two tasks with different tag sets.
func TestListTagNamesByEntities_MultipleEntities(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	_, taskIDs := setupTaskSetup(t, db, "E49", 2)
	tAuth, err := tagRepo.Create(ctx, "multi-auth")
	if err != nil {
		t.Fatalf("Create(auth) error: %v", err)
	}
	tVoice, err := tagRepo.Create(ctx, "multi-voice")
	if err != nil {
		t.Fatalf("Create(voice) error: %v", err)
	}
	tBeta, err := tagRepo.Create(ctx, "multi-beta")
	if err != nil {
		t.Fatalf("Create(beta) error: %v", err)
	}

	// task[0]: auth, voice
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[0], tAuth.ID); err != nil {
		t.Fatalf("Attach(task0, auth) error: %v", err)
	}
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[0], tVoice.ID); err != nil {
		t.Fatalf("Attach(task0, voice) error: %v", err)
	}
	// task[1]: beta
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[1], tBeta.ID); err != nil {
		t.Fatalf("Attach(task1, beta) error: %v", err)
	}

	rows, err := entityTagRepo.ListTagNamesByEntities(ctx, models.EntityTypeTask, []int64{taskIDs[0], taskIDs[1]})
	if err != nil {
		t.Fatalf("ListTagNamesByEntities() error: %v", err)
	}

	// Expected: 3 rows total, ordered entity_id ASC, name ASC.
	if len(rows) != 3 {
		t.Fatalf("ListTagNamesByEntities() returned %d rows, want 3: %v", len(rows), rows)
	}
}

// TestListTagNamesByEntities_EmptyInputReturnsEmpty: empty entityIDs input.
func TestListTagNamesByEntities_EmptyInputReturnsEmpty(t *testing.T) {
	_, entityTagRepo, _ := setupEntityTagRepo(t)
	ctx := context.Background()

	rows, err := entityTagRepo.ListTagNamesByEntities(ctx, models.EntityTypeTask, []int64{})
	if err != nil {
		t.Fatalf("ListTagNamesByEntities(empty) error: %v", err)
	}
	if rows == nil {
		t.Error("ListTagNamesByEntities(empty) returned nil, want non-nil empty slice")
	}
	if len(rows) != 0 {
		t.Errorf("ListTagNamesByEntities(empty) returned %d rows, want 0", len(rows))
	}
}

// TestListTagNamesByEntities_EntityWithNoTagsOmitted: entity with no attachments
// produces no rows (method does NOT guarantee an entry per input ID).
func TestListTagNamesByEntities_EntityWithNoTagsOmitted(t *testing.T) {
	tagRepo, entityTagRepo, db := setupEntityTagRepo(t)
	ctx := context.Background()

	_, taskIDs := setupTaskSetup(t, db, "E48", 2)
	tAuth, err := tagRepo.Create(ctx, "omit-auth")
	if err != nil {
		t.Fatalf("Create(auth) error: %v", err)
	}

	// Only task[0] gets a tag; task[1] has none.
	if err := entityTagRepo.Attach(ctx, models.EntityTypeTask, taskIDs[0], tAuth.ID); err != nil {
		t.Fatalf("Attach(task0) error: %v", err)
	}

	rows, err := entityTagRepo.ListTagNamesByEntities(ctx, models.EntityTypeTask, []int64{taskIDs[0], taskIDs[1]})
	if err != nil {
		t.Fatalf("ListTagNamesByEntities() error: %v", err)
	}

	// Only task[0] has tags; expect 1 row.
	if len(rows) != 1 {
		t.Fatalf("ListTagNamesByEntities() returned %d rows, want 1: %v", len(rows), rows)
	}
	if rows[0].EntityID != taskIDs[0] || rows[0].TagName != "omit-auth" {
		t.Errorf("rows[0] = {%d, %q}, want {%d, %q}", rows[0].EntityID, rows[0].TagName, taskIDs[0], "omit-auth")
	}
}
