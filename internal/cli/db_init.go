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
	configPath := GlobalConfig.ConfigFile
	projectRoot := ""

	if configPath != "" {
		if !filepath.IsAbs(configPath) {
			absPath, err := filepath.Abs(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve config path %q: %w", configPath, err)
			}
			configPath = absPath
		}
		projectRoot = filepath.Dir(configPath)
	} else {
		var err error
		projectRoot, err = FindProjectRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to find project root: %w", err)
		}
		configPath = filepath.Join(projectRoot, ".sharkconfig.json")
	}

	repoDb, err := dbinit.Init(ctx, dbinit.Options{
		ProjectRoot: projectRoot,
		ConfigPath:  configPath,
		DBPath:      resolvedDBOverride(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return repoDb, nil
}

func resolvedDBOverride() string {
	dbFlag := RootCmd.PersistentFlags().Lookup("db")
	if dbFlag == nil || !dbFlag.Changed || GlobalConfig.DBPath == "" {
		return ""
	}

	if filepath.IsAbs(GlobalConfig.DBPath) {
		return GlobalConfig.DBPath
	}

	if GlobalConfig.ConfigFile != "" {
		return filepath.Join(filepath.Dir(GlobalConfig.ConfigFile), GlobalConfig.DBPath)
	}

	projectRoot, err := FindProjectRoot()
	if err != nil {
		return GlobalConfig.DBPath
	}
	return filepath.Join(projectRoot, GlobalConfig.DBPath)
}
