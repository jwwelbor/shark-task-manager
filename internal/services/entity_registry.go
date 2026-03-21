package services

import (
	"fmt"
	"sort"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityRegistry maps EntityType to EntityRepository, providing a single
// lookup point for cross-cutting services to access any entity's repository.
//
// Thread-safe for concurrent reads and writes via sync.RWMutex.
// Registration is expected at startup; lookups happen at request time.
type EntityRegistry struct {
	mu    sync.RWMutex
	repos map[models.EntityType]EntityRepository
}

// NewEntityRegistry creates an empty EntityRegistry.
func NewEntityRegistry() *EntityRegistry {
	return &EntityRegistry{
		repos: make(map[models.EntityType]EntityRepository),
	}
}

// Register adds an EntityRepository for the given entity type.
// Panics if a repository is already registered for that type --
// duplicate registration indicates a wiring bug that should be caught at startup.
func (r *EntityRegistry) Register(entityType models.EntityType, repo EntityRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.repos[entityType]; exists {
		panic(fmt.Sprintf("EntityRegistry: duplicate registration for entity type %q", entityType))
	}
	r.repos[entityType] = repo
}

// GetRepository returns the EntityRepository for the given entity type,
// or an error if no repository has been registered for that type.
func (r *EntityRegistry) GetRepository(entityType models.EntityType) (EntityRepository, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	repo, ok := r.repos[entityType]
	if !ok {
		return nil, fmt.Errorf("EntityRegistry: no repository registered for entity type %q", entityType)
	}
	return repo, nil
}

// MustGetRepository returns the EntityRepository for the given entity type.
// Panics if no repository has been registered -- intended for CLI entry points
// where missing registration is a fatal startup error.
func (r *EntityRegistry) MustGetRepository(entityType models.EntityType) EntityRepository {
	repo, err := r.GetRepository(entityType)
	if err != nil {
		panic(err.Error())
	}
	return repo
}

// RegisteredTypes returns all entity types that have registered repositories,
// sorted alphabetically for deterministic output.
func (r *EntityRegistry) RegisteredTypes() []models.EntityType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]models.EntityType, 0, len(r.repos))
	for t := range r.repos {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return string(types[i]) < string(types[j])
	})
	return types
}
