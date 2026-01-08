package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/spf13/cobra"
)

// PathValidationResult holds validated path information
type PathValidationResult struct {
	RelativePath string
	AbsolutePath string
}

// ValidateCustomPath validates and processes the --path flag
// Returns nil if flag not provided, error if validation fails
func ValidateCustomPath(cmd *cobra.Command, flagName string) (*PathValidationResult, error) {
	customPath, _ := cmd.Flags().GetString(flagName)
	if customPath == "" {
		return nil, nil // Not provided, not an error
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	absPath, relPath, err := utils.ValidateFolderPath(customPath, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", customPath, err)
	}

	return &PathValidationResult{
		RelativePath: relPath,
		AbsolutePath: absPath,
	}, nil
}

// ValidateCustomFilename validates and processes the --filename flag
// Returns nil if flag not provided, error if validation fails
func ValidateCustomFilename(cmd *cobra.Command, flagName string, projectRoot string) (*PathValidationResult, error) {
	filename, _ := cmd.Flags().GetString(flagName)
	if filename == "" {
		return nil, nil // Not provided, not an error
	}

	// Check that filename ends with .md
	if !strings.HasSuffix(filename, ".md") {
		return nil, fmt.Errorf("filename must end with .md extension, got: %s", filename)
	}

	// Use the same path validation logic as custom path
	absPath, relPath, err := utils.ValidateFolderPath(filepath.Dir(filename), projectRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid filename path %q: %w", filename, err)
	}

	// Construct full paths with the filename
	fullRelPath := filepath.Join(relPath, filepath.Base(filename))
	fullAbsPath := filepath.Join(absPath, filepath.Base(filename))

	return &PathValidationResult{
		RelativePath: fullRelPath,
		AbsolutePath: fullAbsPath,
	}, nil
}

// ValidateNoSpaces ensures a key doesn't contain spaces
func ValidateNoSpaces(key string, entityType string) error {
	if strings.Contains(key, " ") {
		return fmt.Errorf("%s key cannot contain spaces: %q", entityType, key)
	}
	return nil
}

// ValidateStatus ensures status is one of: draft, active, completed, archived
func ValidateStatus(status string, entityType string) error {
	if status == "" {
		return fmt.Errorf("%s status cannot be empty", entityType)
	}

	// Use the existing model validation functions
	if entityType == "epic" {
		return models.ValidateEpicStatus(status)
	}
	if entityType == "feature" {
		return models.ValidateFeatureStatus(status)
	}

	// Generic validation for other entity types
	validStatuses := map[string]bool{
		"draft":     true,
		"active":    true,
		"completed": true,
		"archived":  true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid %s status %q: must be one of draft, active, completed, archived", entityType, status)
	}
	return nil
}

// ValidatePriority ensures priority is one of: low, medium, high
func ValidatePriority(priority string, entityType string) error {
	if priority == "" {
		return fmt.Errorf("%s priority cannot be empty", entityType)
	}

	// Use the existing model validation function
	return models.ValidatePriority(priority)
}
