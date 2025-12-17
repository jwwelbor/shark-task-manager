package init

import (
	"context"
	"path/filepath"
)

// Initializer orchestrates PM CLI initialization
type Initializer struct {
	// No persistent state
}

// NewInitializer creates a new Initializer instance
func NewInitializer() *Initializer {
	return &Initializer{}
}

// Initialize performs complete PM CLI initialization
func (i *Initializer) Initialize(ctx context.Context, opts InitOptions) (*InitResult, error) {
	result := &InitResult{
		FoldersCreated: []string{}, // Initialize to empty slice, not nil
	}

	// Step 1: Create database
	dbCreated, err := i.createDatabase(ctx, opts.DBPath)
	if err != nil {
		return nil, &InitError{Step: "database", Message: "Failed to create database", Err: err}
	}
	result.DatabaseCreated = dbCreated
	result.DatabasePath, _ = filepath.Abs(opts.DBPath)

	// Step 2: Create folders
	folders, err := i.createFolders()
	if err != nil {
		return nil, &InitError{Step: "folders", Message: "Failed to create folders", Err: err}
	}
	result.FoldersCreated = folders

	// Step 3: Create config
	configCreated, err := i.createConfig(opts)
	if err != nil {
		return nil, &InitError{Step: "config", Message: "Failed to create config", Err: err}
	}
	result.ConfigCreated = configCreated
	result.ConfigPath, _ = filepath.Abs(opts.ConfigPath)

	// Step 4: Copy templates
	count, err := i.copyTemplates(opts.Force)
	if err != nil {
		return nil, &InitError{Step: "templates", Message: "Failed to copy templates", Err: err}
	}
	result.TemplatesCopied = count

	return result, nil
}
