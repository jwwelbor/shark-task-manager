package cli

import (
	"context"
	"fmt"

	tagrepo "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// GetTagService returns a *TagService wired to the current project's DB and
// maintainer gate. Creates a new service instance on every call; shares the
// global DB connection. Panics on DB failure.
func GetTagService() *services.TagService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for tag service: %v", err))
	}

	tagRepo := tagrepo.NewTagRepository(db)
	entityTagRepo := tagrepo.NewEntityTagRepository(db)
	gate := GetMaintainerGate()

	// Degrade gracefully when config cannot be loaded — empty stub disables enforcement.
	var enforcement services.TagEnforcementConfig = services.EmptyTagEnforcementConfig{}
	if cfg, cfgErr := GetConfig(); cfgErr == nil && cfg != nil {
		enforcement = cfg
	}

	return services.NewTagService(tagRepo, entityTagRepo, gate, enforcement)
}
