package services

import (
	"context"
	"fmt"

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

// NoteEpicRepository defines the epic repository interface needed by NoteService.
type NoteEpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetByID(ctx context.Context, id int64) (*models.Epic, error)
}

// NoteFeatureRepository defines the feature repository interface needed by NoteService.
type NoteFeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
}

// NoteTaskRepository defines the task repository interface needed by NoteService.
type NoteTaskRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
}

// NoteChangeCardRepository defines the change-card repository interface needed by NoteService.
type NoteChangeCardRepository interface {
	GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
	GetByID(ctx context.Context, id int64) (*models.ChangeCard, error)
}

// NoteBugRepository defines the bug repository interface needed by NoteService.
type NoteBugRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Bug, error)
	GetByID(ctx context.Context, id int64) (*models.Bug, error)
}

// NoteEntityDetails contains the key and name of an entity referenced by a note.
type NoteEntityDetails struct {
	Key  string
	Name string
}

// GetEntityDetails returns the key and name of the entity referenced by a note.
// Returns nil if the entity cannot be found (caller should skip or handle gracefully).
func (s *NoteService) GetEntityDetails(ctx context.Context, entityType models.EntityType, entityID int64) *NoteEntityDetails {
	switch entityType {
	case models.EntityTypeTask:
		task, err := s.taskRepo.GetByID(ctx, entityID)
		if err != nil {
			return nil
		}
		return &NoteEntityDetails{Key: task.Key, Name: task.Title}
	case models.EntityTypeEpic:
		epic, err := s.epicRepo.GetByID(ctx, entityID)
		if err != nil {
			return nil
		}
		return &NoteEntityDetails{Key: epic.Key, Name: epic.Title}
	case models.EntityTypeFeature:
		feature, err := s.featureRepo.GetByID(ctx, entityID)
		if err != nil {
			return nil
		}
		return &NoteEntityDetails{Key: feature.Key, Name: feature.Title}
	case models.EntityTypeChange:
		if s.changeCardRepo == nil {
			return nil
		}
		card, err := s.changeCardRepo.GetByID(ctx, entityID)
		if err != nil {
			return nil
		}
		return &NoteEntityDetails{Key: card.Key, Name: card.Title}
	case models.EntityTypeBug:
		if s.bugRepo == nil {
			return nil
		}
		bug, err := s.bugRepo.GetByID(ctx, entityID)
		if err != nil {
			return nil
		}
		return &NoteEntityDetails{Key: bug.Key, Name: bug.Title}
	default:
		return nil
	}
}

// NoteService provides business logic for note operations across all entity types.
type NoteService struct {
	noteRepo       NoteEntityNoteRepository
	epicRepo       NoteEpicRepository
	featureRepo    NoteFeatureRepository
	taskRepo       NoteTaskRepository
	changeCardRepo NoteChangeCardRepository
	bugRepo        NoteBugRepository
}

// SetChangeCardRepo sets the optional change-card repository for change entity support.
func (s *NoteService) SetChangeCardRepo(repo NoteChangeCardRepository) {
	s.changeCardRepo = repo
}

// SetBugRepo sets the optional bug repository for bug entity note support.
func (s *NoteService) SetBugRepo(repo NoteBugRepository) {
	s.bugRepo = repo
}

// NewNoteService creates a new NoteService with injected dependencies.
func NewNoteService(noteRepo NoteEntityNoteRepository, epicRepo NoteEpicRepository, featureRepo NoteFeatureRepository, taskRepo NoteTaskRepository) *NoteService {
	return &NoteService{
		noteRepo:    noteRepo,
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
	}
}

// AddNote resolves the entity key to an ID and creates a note on the entity.
func (s *NoteService) AddNote(ctx context.Context, entityType models.EntityType, entityKey string, noteType string, content string, createdBy string) (*models.EntityNote, error) {
	if err := models.ValidateNoteType(noteType); err != nil {
		return nil, fmt.Errorf("invalid note type: %w", err)
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
	}

	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
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

// resolveEntityID resolves an entity key to its database ID using the appropriate repository.
func (s *NoteService) resolveEntityID(ctx context.Context, entityType models.EntityType, key string) (int64, error) {
	switch entityType {
	case models.EntityTypeEpic:
		epic, err := s.epicRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("epic not found: %s: %w", key, err)
		}
		return epic.ID, nil
	case models.EntityTypeFeature:
		feature, err := s.featureRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("feature not found: %s: %w", key, err)
		}
		return feature.ID, nil
	case models.EntityTypeTask:
		task, err := s.taskRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("task not found: %s: %w", key, err)
		}
		return task.ID, nil
	case models.EntityTypeChange:
		if s.changeCardRepo == nil {
			return 0, fmt.Errorf("change-card support not configured")
		}
		card, err := s.changeCardRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("change-card not found: %s: %w", key, err)
		}
		return card.ID, nil
	case models.EntityTypeBug:
		if s.bugRepo == nil {
			return 0, fmt.Errorf("bug repository not configured for note operations")
		}
		bug, err := s.bugRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("bug not found: %s: %w", key, err)
		}
		return bug.ID, nil
	default:
		return 0, fmt.Errorf("unsupported entity type: %s", entityType)
	}
}
