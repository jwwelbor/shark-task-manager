package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
)

// mapRepositoryError adds the CLI exit code for repository errors with a
// stable classification. Unknown errors remain unchanged so validation and
// other service errors keep their existing behavior.
func mapRepositoryError(entity, key string, err error) error {
	if repoerr.IsNotFound(err) {
		return fmt.Errorf("exit code 1: %s %s not found: %w", entity, key, err)
	}
	if repoerr.IsDatabaseFailure(err) {
		// Keep the repository/service context in the wrapped cause. This mapper
		// is shared by reads and writes, so the message must not imply a lookup.
		return fmt.Errorf("exit code 2: %w", err)
	}
	return err
}
