// Package tag_test provides integration tests for TagRepository.
// Per .claude/rules/testing/repository-tests.md, repository tests use the real test database.
// Each test uses test.NewIsolatedTestDB(t) for a fresh, independent database — no
// DELETE-before boilerplate needed; isolation is guaranteed by the helper.
package tag_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/tag"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// setupTagRepo creates a TagRepository backed by an isolated test database.
// Each call returns a fresh *tag.TagRepository and its *dbconn.DB.
func setupTagRepo(t *testing.T) (*tag.TagRepository, *dbconn.DB) {
	t.Helper()
	database := test.NewIsolatedTestDB(t)
	db := dbconn.NewDB(database)
	repo := tag.NewTagRepository(db)
	return repo, db
}

// setupTagAndEntityTagRepos creates both TagRepository and EntityTagRepository
// backed by the same isolated test database. Returns both repos and the raw DB
// wrapper so tests can perform low-level insertions or assertions.
func setupTagAndEntityTagRepos(t *testing.T) (*tag.TagRepository, *tag.EntityTagRepository, *dbconn.DB) {
	t.Helper()
	database := test.NewIsolatedTestDB(t)
	db := dbconn.NewDB(database)
	tagRepo := tag.NewTagRepository(db)
	entityTagRepo := tag.NewEntityTagRepository(db)
	return tagRepo, entityTagRepo, db
}

// insertRawEpic inserts a minimal epic row via raw SQL and returns its id.
// Used by TagRepository tests that need an entity to link tags against.
func insertRawEpic(t *testing.T, db *dbconn.DB, key string) int64 {
	t.Helper()
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES (?, 'Test Epic', 'test', 'active', 'high')
	`, key)
	if err != nil {
		t.Fatalf("insertRawEpic: failed to insert epic %q: %v", key, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("insertRawEpic: LastInsertId: %v", err)
	}
	return id
}

// ─── TagRepository: Create ────────────────────────────────────────────────────

func TestTagRepository_Create(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	tag1, err := repo.Create(ctx, "voice")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if tag1 == nil {
		t.Fatal("Create() returned nil tag")
	}
	if tag1.ID == 0 {
		t.Error("Create() ID should be non-zero")
	}
	if tag1.Name != "voice" {
		t.Errorf("Create() Name = %q, want %q", tag1.Name, "voice")
	}
	if tag1.CreatedAt.IsZero() {
		t.Error("Create() CreatedAt should be set")
	}
}

// AC-T2: duplicate name returns a UNIQUE constraint error.
func TestTagRepository_Create_DuplicateName_ReturnsError(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, "voice")
	if err != nil {
		t.Fatalf("first Create() unexpected error: %v", err)
	}

	_, err = repo.Create(ctx, "voice")
	if err == nil {
		t.Fatal("second Create() with duplicate name should return error, got nil")
	}
}

// The DB has UNIQUE COLLATE NOCASE so "Voice" conflicts with "voice".
func TestTagRepository_Create_CaseNormalized(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, "voice")
	if err != nil {
		t.Fatalf("Create(voice) unexpected error: %v", err)
	}

	_, err = repo.Create(ctx, "Voice")
	if err == nil {
		t.Fatal("Create(Voice) should conflict with existing 'voice' due to NOCASE collation")
	}
}

// ─── TagRepository: GetByName ─────────────────────────────────────────────────

func TestTagRepository_GetByName(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, "auth")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	got, err := repo.GetByName(ctx, "auth")
	if err != nil {
		t.Fatalf("GetByName() unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetByName() ID = %d, want %d", got.ID, created.ID)
	}
	if got.Name != "auth" {
		t.Errorf("GetByName() Name = %q, want %q", got.Name, "auth")
	}
}

// AC-T2: GetByName is case-insensitive (NOCASE collation).
func TestTagRepository_GetByName_CaseInsensitive(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, "auth")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	got, err := repo.GetByName(ctx, "AUTH")
	if err != nil {
		t.Fatalf("GetByName(AUTH) unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("GetByName(AUTH) returned nil, want tag")
	}
}

func TestTagRepository_GetByName_NotFound(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	_, err := repo.GetByName(ctx, "nonexistent")
	if err == nil {
		t.Fatal("GetByName() for missing tag should return error, got nil")
	}
}

// ─── TagRepository: GetByID ───────────────────────────────────────────────────

func TestTagRepository_GetByID(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, "backend")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetByID() ID = %d, want %d", got.ID, created.ID)
	}
	if got.Name != "backend" {
		t.Errorf("GetByID() Name = %q, want %q", got.Name, "backend")
	}
}

func TestTagRepository_GetByID_NotFound(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Fatal("GetByID() for missing id should return error, got nil")
	}
}

// ─── TagRepository: List ──────────────────────────────────────────────────────

func TestTagRepository_List_Empty(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	tags, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("List() on empty db = %d tags, want 0", len(tags))
	}
}

func TestTagRepository_List_ReturnsSortedByName(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	for _, name := range []string{"zebra", "alpha", "middle"} {
		if _, err := repo.Create(ctx, name); err != nil {
			t.Fatalf("Create(%q) unexpected error: %v", name, err)
		}
	}

	tags, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("List() returned %d tags, want 3", len(tags))
	}
	if tags[0].Name != "alpha" || tags[1].Name != "middle" || tags[2].Name != "zebra" {
		t.Errorf("List() not sorted by name: got %v", tagNames(tags))
	}
}

// ─── TagRepository: Rename ───────────────────────────────────────────────────

// AC-T2: Rename succeeds and the tag's own ID is preserved.
// Additionally verifies that entity_tag row IDs are NOT changed by a rename
// (the tag ID column in entity_tags is immutable through a rename — ADR-8).
func TestTagRepository_Rename_PreservesIDsAndEntityTagLinks(t *testing.T) {
	tagRepo, entityTagRepo, db := setupTagAndEntityTagRepos(t)
	ctx := context.Background()

	// Create a tag and attach it to an epic.
	original, err := tagRepo.Create(ctx, "voice")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	epicID := insertRawEpic(t, db, "E80")
	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, original.ID); err != nil {
		t.Fatalf("Attach() error: %v", err)
	}

	// Capture the entity_tag row ID before rename.
	linksBefore, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() before rename: %v", err)
	}
	if len(linksBefore) != 1 {
		t.Fatalf("expected 1 entity_tag link before rename, got %d", len(linksBefore))
	}
	entityTagIDBeforeRename := linksBefore[0].ID

	// Rename the tag.
	renamed, err := tagRepo.Rename(ctx, original.ID, "audio")
	if err != nil {
		t.Fatalf("Rename() unexpected error: %v", err)
	}
	if renamed.Name != "audio" {
		t.Errorf("Rename() Name = %q, want %q", renamed.Name, "audio")
	}
	if renamed.ID != original.ID {
		t.Errorf("Rename() should preserve tag ID: got %d, want %d", renamed.ID, original.ID)
	}

	// Verify entity_tag row ID is unchanged (ADR-8: entity_tag rows are immutable through rename).
	linksAfter, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() after rename: %v", err)
	}
	if len(linksAfter) != 1 {
		t.Fatalf("expected 1 entity_tag link after rename, got %d", len(linksAfter))
	}
	if linksAfter[0].ID != entityTagIDBeforeRename {
		t.Errorf("entity_tag row ID changed after rename: before=%d, after=%d",
			entityTagIDBeforeRename, linksAfter[0].ID)
	}
	if linksAfter[0].TagID != original.ID {
		t.Errorf("entity_tag.tag_id changed after rename: got %d, want %d",
			linksAfter[0].TagID, original.ID)
	}
}

// AC-T2: Rename to an existing name returns ErrTagConflict (ADR-8).
func TestTagRepository_Rename_ConflictReturnsErrTagConflict(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	if _, err := repo.Create(ctx, "voice"); err != nil {
		t.Fatalf("Create(voice) error: %v", err)
	}
	existing, err := repo.Create(ctx, "audio")
	if err != nil {
		t.Fatalf("Create(audio) error: %v", err)
	}

	_, err = repo.Rename(ctx, existing.ID, "voice")
	if err == nil {
		t.Fatal("Rename() to existing name should return error, got nil")
	}
	if !errors.Is(err, tag.ErrTagConflict) {
		t.Errorf("Rename() conflict error = %v, want ErrTagConflict", err)
	}
}

func TestTagRepository_Rename_NotFound(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	_, err := repo.Rename(ctx, 99999, "newname")
	if err == nil {
		t.Fatal("Rename() on missing ID should return error")
	}
}

// ─── TagRepository: Delete ───────────────────────────────────────────────────

func TestTagRepository_Delete_NoUsage(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	tag1, err := repo.Create(ctx, "removeme")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, tag1.ID, false); err != nil {
		t.Fatalf("Delete(force=false) on unused tag error: %v", err)
	}

	// Verify the tag row is gone.
	_, err = repo.GetByID(ctx, tag1.ID)
	if err == nil {
		t.Error("GetByID() after Delete should return error, got nil")
	}
}

// AC-T2: Delete without force and with active links returns ErrTagInUse (ADR-9).
func TestTagRepository_Delete_InUse_ForceFalse_ReturnsErrTagInUse(t *testing.T) {
	tagRepo, entityTagRepo, db := setupTagAndEntityTagRepos(t)
	ctx := context.Background()

	epicID := insertRawEpic(t, db, "E79")

	t1, err := tagRepo.Create(ctx, "used-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("Attach() error: %v", err)
	}

	err = tagRepo.Delete(ctx, t1.ID, false)
	if err == nil {
		t.Fatal("Delete(force=false) on in-use tag should return error, got nil")
	}
	if !errors.Is(err, tag.ErrTagInUse) {
		t.Errorf("Delete(force=false) error = %v, want ErrTagInUse", err)
	}
}

// AC-T2: Delete with force removes links and tag atomically (ADR-9).
func TestTagRepository_Delete_InUse_ForceTrue_DeletesAll(t *testing.T) {
	tagRepo, entityTagRepo, db := setupTagAndEntityTagRepos(t)
	ctx := context.Background()

	epicID := insertRawEpic(t, db, "E78")

	t1, err := tagRepo.Create(ctx, "force-delete-tag")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := entityTagRepo.Attach(ctx, models.EntityTypeEpic, epicID, t1.ID); err != nil {
		t.Fatalf("Attach() error: %v", err)
	}

	if err := tagRepo.Delete(ctx, t1.ID, true); err != nil {
		t.Fatalf("Delete(force=true) error: %v", err)
	}

	// Verify tag row is gone.
	_, err = tagRepo.GetByID(ctx, t1.ID)
	if err == nil {
		t.Error("tag should be deleted after force delete")
	}

	// Verify entity_tags rows are also gone (transactional delete).
	links, err := entityTagRepo.ListByEntity(ctx, models.EntityTypeEpic, epicID)
	if err != nil {
		t.Fatalf("ListByEntity() error: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("entity_tags rows should be gone after force delete, got %d", len(links))
	}
}

func TestTagRepository_Delete_NotFound(t *testing.T) {
	repo, _ := setupTagRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999, false)
	if err == nil {
		t.Fatal("Delete() on missing id should return error")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// tagNames extracts tag names for use in assertion error messages.
func tagNames(tags []*models.Tag) []string {
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names
}
