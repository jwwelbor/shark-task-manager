package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// requireNonNil panics if value is nil. Used in service constructors to validate
// required dependencies at construction time rather than at first use.
func requireNonNil(value interface{}, name string) {
	if value == nil {
		panic(fmt.Sprintf("%s must not be nil", name))
	}
}

// isContained reports whether targetCanon is equal to rootCanon or is a direct
// descendant of it. Both arguments must be canonicalized (EvalSymlinks-resolved)
// absolute paths. Used by path-security checks in viewer and edit services.
func isContained(rootCanon, targetCanon string) bool {
	return targetCanon == rootCanon || strings.HasPrefix(targetCanon, rootCanon+string(os.PathSeparator))
}

// enforceTagsRequired calls tagSvc.EnforceRequired when tagSvc is non-nil.
// A nil tagSvc is a silent no-op so entity services work without tag
// integration wired in.
func enforceTagsRequired(ctx context.Context, tagSvc TagAttacher, entityType models.EntityType, names []string) error {
	if tagSvc == nil {
		return nil
	}
	return tagSvc.EnforceRequired(ctx, entityType, names)
}

// attachTagsIfAny calls tagSvc.AttachMany when tagSvc is non-nil and names
// is non-empty. Call after the entity has been persisted so entityID is valid.
func attachTagsIfAny(ctx context.Context, tagSvc TagAttacher, entityType models.EntityType, entityID int64, names []string) error {
	if tagSvc == nil || len(names) == 0 {
		return nil
	}
	return tagSvc.AttachMany(ctx, entityType, entityID, names)
}
