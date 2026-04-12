package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/dbinit"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// initDatabase initializes a cloud-aware database connection.
// It delegates to the shared internal/dbinit package so the same logic is
// available to cmd/server without importing internal/cli.
func initDatabase(ctx context.Context) (*repository.DB, error) {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	repoDb, err := dbinit.Init(ctx, dbinit.Options{
		ProjectRoot: projectRoot,
		ConfigPath:  filepath.Join(projectRoot, ".sharkconfig.json"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return repoDb, nil
}
