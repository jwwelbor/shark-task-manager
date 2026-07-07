package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// NoteEntityNoteRepository defines the repository interface for entity notes.
type NoteEntityNoteRepository interface {
	Create(ctx context.Context, note *models.EntityNote) error
	GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
	GetByEntityAndType(ctx context.Context, entityType models.EntityType, entityID int64, noteTypes []string) ([]*models.EntityNote, error)
	Search(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error)
	SearchWithTimePeriod(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error)
}

// NoteEntityDetails contains the key and name of an entity referenced by a note.
type NoteEntityDetails struct {
	Key  string
	Name string
}

// NoteService provides business logic for note operations across all entity types.
type NoteService struct {
	noteRepo      NoteEntityNoteRepository
	registry      *EntityRegistry
	searchIndexer SearchIndexer
}

// NewNoteService creates a new NoteService with injected dependencies.
func NewNoteService(noteRepo NoteEntityNoteRepository, registry *EntityRegistry) (*NoteService, error) {
	if registry == nil {
		return nil, fmt.Errorf("NoteService: EntityRegistry must not be nil")
	}
	return &NoteService{
		noteRepo: noteRepo,
		registry: registry,
	}, nil
}

// SetSearchIndexer wires the optional search indexer used to refresh the
// parent entity after note writes.
func (s *NoteService) SetSearchIndexer(indexer SearchIndexer) {
	s.searchIndexer = indexer
}

// GetEntityDetails returns the key and name of the entity referenced by a note.
// Returns nil if the entity cannot be found (caller should skip or handle gracefully).
func (s *NoteService) GetEntityDetails(ctx context.Context, entityType models.EntityType, entityID int64) *NoteEntityDetails {
	repo, err := s.registry.GetRepository(entityType)
	if err != nil {
		return nil
	}
	entity, err := repo.GetByID(ctx, entityID)
	if err != nil {
		return nil
	}
	return &NoteEntityDetails{
		Key:  entity.GetKey(),
		Name: entity.GetTitle(),
	}
}

// AddNote resolves the entity key to an ID and creates a note on the entity.
func (s *NoteService) AddNote(ctx context.Context, entityType models.EntityType, entityKey string, noteType string, content string, createdBy string) (*models.EntityNote, error) {
	return s.AddNoteWithMetadata(ctx, entityType, entityKey, noteType, content, createdBy, "")
}

// AddNoteWithMetadata is AddNote with an optional structured-metadata JSON
// payload. Metadata makes note fields queryable (e.g. review-finding notes
// carry gate/round/severity/defect_class/fingerprint) instead of burying
// them in free text. Empty metadata stores NULL; non-empty metadata must be
// a valid JSON object.
func (s *NoteService) AddNoteWithMetadata(ctx context.Context, entityType models.EntityType, entityKey string, noteType string, content string, createdBy string, metadata string) (*models.EntityNote, error) {
	if err := models.ValidateNoteType(noteType); err != nil {
		return nil, fmt.Errorf("invalid note type: %w", err)
	}

	metadata = strings.TrimSpace(metadata)
	var metadataPtr *string
	if metadata != "" {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(metadata), &obj); err != nil {
			return nil, fmt.Errorf("invalid note metadata: must be a JSON object: %w", err)
		}
		metadataPtr = &metadata
	}

	entityID, err := s.resolveEntityID(ctx, entityType, entityKey)
	if err != nil {
		return nil, err
	}

	var createdByPtr *string
	if createdBy != "" {
		createdByPtr = &createdBy
	}

	note := &models.EntityNote{
		EntityType: entityType,
		EntityID:   entityID,
		NoteType:   models.NoteType(noteType),
		Content:    content,
		CreatedBy:  createdByPtr,
		Metadata:   metadataPtr,
	}

	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, entityType, entityID); err != nil {
		return nil, err
	}

	return note, nil
}

// ListNotes returns notes for the specified entity, optionally filtered by note types.
func (s *NoteService) ListNotes(ctx context.Context, entityType models.EntityType, entityKey string, noteTypes []string) ([]*models.EntityNote, error) {
	entityID, err := s.resolveEntityID(ctx, entityType, entityKey)
	if err != nil {
		return nil, err
	}

	if len(noteTypes) > 0 {
		return s.noteRepo.GetByEntityAndType(ctx, entityType, entityID, noteTypes)
	}
	return s.noteRepo.GetByEntity(ctx, entityType, entityID)
}

// SearchNotes searches across notes with optional filters.
func (s *NoteService) SearchNotes(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error) {
	return s.noteRepo.Search(ctx, query, noteTypes, entityType, epicKey, featureKey)
}

// SearchNotesWithTimePeriod searches notes filtered by time period (since/until date strings).
// The since and until parameters are date strings in YYYY-MM-DD format; empty means no bound.
func (s *NoteService) SearchNotesWithTimePeriod(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error) {
	return s.noteRepo.SearchWithTimePeriod(ctx, query, noteTypes, epicKey, featureKey, since, until)
}

// resolveEntityID resolves an entity key to its database ID using the registry.
func (s *NoteService) resolveEntityID(ctx context.Context, entityType models.EntityType, key string) (int64, error) {
	repo, err := s.registry.GetRepository(entityType)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve entity for note operation: %w", err)
	}
	entity, err := repo.GetByKey(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("%s not found: %s: %w", entityType, key, err)
	}
	return entity.GetID(), nil
}
