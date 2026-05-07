package services

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// SizeEnforcementConfig is the narrow contract entity services need from
// config to decide whether --size is required at create time. Implemented
// by *config.Config.SizeRequiredFor().
type SizeEnforcementConfig interface {
	SizeRequiredFor() []string
}

// EmptySizeEnforcementConfig is a safe no-op fallback used when the real
// config cannot be loaded. Disables size enforcement for all entity types.
type EmptySizeEnforcementConfig struct{}

// SizeRequiredFor returns nil — no entity types require size.
func (EmptySizeEnforcementConfig) SizeRequiredFor() []string { return nil }

// SizeRequiredError is returned when the given entity type is listed in
// Config.SizeRequiredFor but the size pointer is nil at create time.
//
// Error() format: "size is required for <EntityType>"
type SizeRequiredError struct {
	// EntityType is the string form of the entity type (e.g. "task",
	// "feature", "epic", "bug", "change", "idea", "tech-debt").
	EntityType string
}

// Error implements the error interface.
func (e *SizeRequiredError) Error() string {
	return fmt.Sprintf("size is required for %s (set --size or remove %q from size_required_for in .sharkconfig.json)", e.EntityType, e.EntityType)
}

// enforceSizeRequired returns *SizeRequiredError when cfg lists entityType in
// SizeRequiredFor and size is nil. Otherwise returns nil. A nil cfg disables
// enforcement (graceful degradation).
//
// Mirrors enforceTagsRequired in helpers.go. Mis-cased entries in the
// configured slice silently disable enforcement for that type — config
// values must match models.EntityType.String() output exactly.
func enforceSizeRequired(cfg SizeEnforcementConfig, entityType models.EntityType, size *int) error {
	if size != nil {
		return nil
	}
	if cfg == nil {
		return nil
	}
	required := cfg.SizeRequiredFor()
	if len(required) == 0 {
		return nil
	}
	et := string(entityType)
	for _, r := range required {
		if r == et {
			return &SizeRequiredError{EntityType: et}
		}
	}
	return nil
}
