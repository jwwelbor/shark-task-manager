package init

import (
	"context"
	"path/filepath"
)

// Initializer orchestrates Shark CLI initialization
type Initializer struct {
	// No persistent state
}

// NewInitializer creates a new Initializer instance
func NewInitializer() *Initializer {
	return &Initializer{}
}

// Initialize performs complete Shark CLI initialization
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
	folders, err := i.createFolders(opts.TemplateDir)
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

	// Step 4: Copy templates (always runs — keeps shark-templates/ in sync with
	// the embedded versions). Files modified by the user are left alone unless
	// opts.Force is set; their paths are returned in TemplatesDiffered so the
	// caller can warn.
	tmpl, err := i.copyTemplates(opts.Force, opts.TemplateDir)
	if err != nil {
		return nil, &InitError{Step: "templates", Message: "Failed to copy templates", Err: err}
	}
	result.TemplatesCopied = tmpl.Copied
	result.TemplatesRefreshed = tmpl.Refreshed
	result.TemplatesDiffered = tmpl.Differed

	return result, nil
}
