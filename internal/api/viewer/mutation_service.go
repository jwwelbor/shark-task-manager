package viewer

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// EpicMutationServicer is the minimal epic write interface needed by the viewer mutation façade.
type EpicMutationServicer interface {
	UpdateEpic(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error)
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
}

// FeatureMutationServicer is the minimal feature write interface needed by the viewer mutation façade.
type FeatureMutationServicer interface {
	UpdateFeature(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error)
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
}

// TaskMutationServicer is the minimal task write interface needed by the viewer mutation façade.
type TaskMutationServicer interface {
	UpdateTask(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error)
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
}

// NoteMutationServicer is the note write contract needed by the viewer mutation façade.
type NoteMutationServicer interface {
	AddNote(ctx context.Context, entityType models.EntityType, entityKey, noteType, content, createdBy string) (*models.EntityNote, error)
}

// RelationshipMutationServicer is the relationship write contract needed by the viewer mutation façade.
type RelationshipMutationServicer interface {
	CreateRelationship(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) (*models.EntityRelationship, error)
	UnlinkEntities(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error
}

// EntityResolver resolves a viewer mutation key to a concrete entity.
type EntityResolver interface {
	Resolve(ctx context.Context, key string) (models.Entity, error)
}

// MutationServicer is the viewer-facing contract used by the mutation handler.
type MutationServicer interface {
	UpdateEpic(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error)
	UpdateFeature(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error)
	UpdateTask(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error)
	TransitionEpic(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	TransitionFeature(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	TransitionTask(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	AddNote(ctx context.Context, entityKey, noteType, content, createdBy string) (*models.EntityNote, error)
	CreateRelationship(ctx context.Context, fromKey, toKey, relType string) (*models.EntityRelationship, error)
	DeleteRelationship(ctx context.Context, fromKey, relType, toKey string) error
}

// MutationService is a thin viewer-facing façade over the existing epic, feature, task,
// note, and relationship services.
type MutationService struct {
	epicSvc         EpicMutationServicer
	featureSvc      FeatureMutationServicer
	taskSvc         TaskMutationServicer
	noteSvc         NoteMutationServicer
	relationshipSvc RelationshipMutationServicer
	resolver        EntityResolver
}

// NewMutationService constructs a viewer mutation façade.
func NewMutationService(
	epicSvc EpicMutationServicer,
	featureSvc FeatureMutationServicer,
	taskSvc TaskMutationServicer,
	noteSvc NoteMutationServicer,
	relationshipSvc RelationshipMutationServicer,
	resolver EntityResolver,
) *MutationService {
	if epicSvc == nil {
		panic("MutationService: epicSvc is required")
	}
	if featureSvc == nil {
		panic("MutationService: featureSvc is required")
	}
	if taskSvc == nil {
		panic("MutationService: taskSvc is required")
	}
	if noteSvc == nil {
		panic("MutationService: noteSvc is required")
	}
	if relationshipSvc == nil {
		panic("MutationService: relationshipSvc is required")
	}
	if resolver == nil {
		panic("MutationService: resolver is required")
	}
	return &MutationService{
		epicSvc:         epicSvc,
		featureSvc:      featureSvc,
		taskSvc:         taskSvc,
		noteSvc:         noteSvc,
		relationshipSvc: relationshipSvc,
		resolver:        resolver,
	}
}

// UpdateEpic delegates to the existing epic service.
func (s *MutationService) UpdateEpic(ctx context.Context, key string, updates services.EpicUpdates) (*models.Epic, error) {
	return s.epicSvc.UpdateEpic(ctx, key, updates)
}

// UpdateFeature delegates to the existing feature service.
func (s *MutationService) UpdateFeature(ctx context.Context, key string, updates services.FeatureUpdates) (*models.Feature, error) {
	return s.featureSvc.UpdateFeature(ctx, key, updates)
}

// UpdateTask delegates to the existing task service.
func (s *MutationService) UpdateTask(ctx context.Context, key string, updates services.TaskUpdates) (*models.Task, error) {
	return s.taskSvc.UpdateTask(ctx, key, updates)
}

// TransitionEpic delegates to the existing epic transition service.
//
// Unlike the CLI's `shark status advance`, this and the two Transition*
// methods below do not check the E39-F03 Question-blocking gate
// (guardQuestionBlockedStatusAdvance) -- that gate's v1 scope is CLI-only
// per the E39-F03 spec (REQ-F-005 names only "shark status advance"). If the
// gate is ever pushed here, prefer moving the check into the shared
// TransitionStatus path both CLI commands and this viewer already use,
// rather than a third duplicated call site.
func (s *MutationService) TransitionEpic(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return s.epicSvc.TransitionStatus(ctx, key, targetStatus, opts)
}

// TransitionFeature delegates to the existing feature transition service.
func (s *MutationService) TransitionFeature(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return s.featureSvc.TransitionStatus(ctx, key, targetStatus, opts)
}

// TransitionTask delegates to the existing task transition service.
func (s *MutationService) TransitionTask(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return s.taskSvc.TransitionStatus(ctx, key, targetStatus, opts)
}

// AddNote resolves the source entity and delegates to the existing note service.
func (s *MutationService) AddNote(ctx context.Context, entityKey, noteType, content, createdBy string) (*models.EntityNote, error) {
	entity, err := s.resolveEntity(ctx, entityKey)
	if err != nil {
		return nil, err
	}
	return s.noteSvc.AddNote(ctx, entity.GetEntityType(), entity.GetKey(), noteType, content, createdBy)
}

// CreateRelationship resolves source and target entities and delegates to the
// normalized relationship service.
func (s *MutationService) CreateRelationship(ctx context.Context, fromKey, toKey, relType string) (*models.EntityRelationship, error) {
	fromEntity, err := s.resolveEntity(ctx, fromKey)
	if err != nil {
		return nil, err
	}
	toEntity, err := s.resolveEntity(ctx, toKey)
	if err != nil {
		return nil, err
	}
	return s.relationshipSvc.CreateRelationship(
		ctx,
		fromEntity.GetEntityType(),
		fromEntity.GetID(),
		toEntity.GetEntityType(),
		toEntity.GetID(),
		models.EntityRelationshipType(relType),
	)
}

// DeleteRelationship resolves source and target entities and delegates to the
// normalized relationship service.
func (s *MutationService) DeleteRelationship(ctx context.Context, fromKey, relType, toKey string) error {
	fromEntity, err := s.resolveEntity(ctx, fromKey)
	if err != nil {
		return err
	}
	toEntity, err := s.resolveEntity(ctx, toKey)
	if err != nil {
		return err
	}
	return s.relationshipSvc.UnlinkEntities(
		ctx,
		fromEntity.GetEntityType(),
		fromEntity.GetID(),
		toEntity.GetEntityType(),
		toEntity.GetID(),
		models.EntityRelationshipType(relType),
	)
}

func (s *MutationService) resolveEntity(ctx context.Context, key string) (models.Entity, error) {
	entity, err := s.resolver.Resolve(ctx, key)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("entity not found: %s", key)
	}
	return entity, nil
}

type registryEntityResolver struct {
	registry *services.EntityRegistry
}

func NewRegistryEntityResolver(registry *services.EntityRegistry) EntityResolver {
	if registry == nil {
		panic("MutationService: registry is required")
	}
	return &registryEntityResolver{registry: registry}
}

func (r *registryEntityResolver) Resolve(ctx context.Context, key string) (models.Entity, error) {
	entityType, normalizedKey, err := resolveMutationKey(key)
	if err != nil {
		return nil, err
	}
	repo, err := r.registry.GetRepository(entityType)
	if err != nil {
		return nil, err
	}
	return repo.GetByKey(ctx, normalizedKey)
}

func resolveMutationKey(rawKey string) (models.EntityType, string, error) {
	upper := strings.ToUpper(strings.TrimSpace(rawKey))
	if upper == "" {
		return "", "", fmt.Errorf("invalid entity key: %s", rawKey)
	}

	switch {
	case keys.IsEpicKey(upper):
		return models.EntityTypeEpic, upper, nil
	case keys.IsFeatureKey(upper):
		return models.EntityTypeFeature, upper, nil
	case keys.IsShortTaskKey(upper), keys.IsTaskKey(upper):
		key, err := keys.NormalizeTaskKey(upper)
		if err != nil {
			return "", "", err
		}
		return models.EntityTypeTask, key, nil
	case keys.IsBugKey(upper):
		return models.EntityTypeBug, upper, nil
	case keys.IsChangeKey(upper):
		return models.EntityTypeChange, upper, nil
	case keys.IsTechDebtKey(upper):
		return models.EntityTypeTechDebt, upper, nil
	case keys.NewKeyService().DetectEntityType(upper) == keys.EntityTypeQuestion:
		return models.EntityTypeQuestion, upper, nil
	default:
		return "", "", fmt.Errorf("invalid entity key: %s", rawKey)
	}
}

var _ MutationServicer = (*MutationService)(nil)
var _ EpicMutationServicer = (*services.EpicService)(nil)
var _ FeatureMutationServicer = (*services.FeatureService)(nil)
var _ TaskMutationServicer = (*services.TaskService)(nil)
var _ NoteMutationServicer = (*services.NoteService)(nil)
var _ RelationshipMutationServicer = (*services.EntityRelationshipService)(nil)
