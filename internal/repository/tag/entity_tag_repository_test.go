// Package tag_test provides integration tests for EntityTagRepository.
// Per .claude/rules/testing/repository-tests.md, repository tests use the real test database.
// Each test uses test.NewIsolatedTestDB(t) for a fresh, independent database — no
// DELETE-before boilerplate needed; isolation is guaranteed by the helper.
package tag_test

import (
	"context"
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
