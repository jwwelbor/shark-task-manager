// Package commands provides shared logic for CLI command handlers.
// This file contains parsing and validation functions for status and priority values
// used across epic, feature, and task creation/update commands.
//
// All parsing functions provide:
// - Case-insensitive input normalization
// - User-friendly error messages
// - Validation against allowed values
package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ParseEpicStatus parses and validates an epic status value.
// Input is case-insensitive and normalized to lowercase.
// Returns the normalized status value or an error if invalid.
func ParseEpicStatus(status string) (string, error) {
	// Trim whitespace and normalize to lowercase
	normalized := strings.TrimSpace(strings.ToLower(status))

	// Check for empty string
	if normalized == "" {
		return "", fmt.Errorf("epic status cannot be empty")
	}

	// Validate using the model validation function
	if err := models.ValidateEpicStatus(normalized); err != nil {
		return "", err
	}

	return normalized, nil
}

// ParseFeatureStatus parses and validates a feature status value.
// Input is case-insensitive and normalized to lowercase.
// Returns the normalized status value or an error if invalid.
func ParseFeatureStatus(status string) (string, error) {
	// Trim whitespace and normalize to lowercase
	normalized := strings.TrimSpace(strings.ToLower(status))

	// Check for empty string
	if normalized == "" {
		return "", fmt.Errorf("feature status cannot be empty")
	}

	// Validate using the model validation function
	if err := models.ValidateFeatureStatus(normalized); err != nil {
		return "", err
	}

	return normalized, nil
}

// ParseTaskStatus parses and validates a task status value.
// Input is case-insensitive and normalized to lowercase.
// Supports both old workflow (todo, in_progress, etc.) and new workflow (draft, in_development, etc.).
// Returns the normalized status value or an error if invalid.
func ParseTaskStatus(status string) (string, error) {
	// Trim whitespace and normalize to lowercase
	normalized := strings.TrimSpace(strings.ToLower(status))

	// Check for empty string
	if normalized == "" {
		return "", fmt.Errorf("task status cannot be empty")
	}

	// Validate using the model validation function (supports both workflows)
	if err := models.ValidateTaskStatus(normalized); err != nil {
		return "", err
	}

	return normalized, nil
}

// ParseEpicPriority parses and validates an epic priority value.
// Input is case-insensitive and normalized to lowercase.
// Valid values: low, medium, high.
// Returns the normalized priority value or an error if invalid.
func ParseEpicPriority(priority string) (string, error) {
	// Trim whitespace and normalize to lowercase
	normalized := strings.TrimSpace(strings.ToLower(priority))

	// Check for empty string
	if normalized == "" {
		return "", fmt.Errorf("epic priority cannot be empty")
	}

	// Validate using the model validation function
	if err := models.ValidatePriority(normalized); err != nil {
		return "", err
	}

	return normalized, nil
}

// ParseTaskPriority parses and validates a task priority value.
// Input is a string representing an integer from 1 to 10.
// Returns the parsed integer value or an error if invalid.
func ParseTaskPriority(priority string) (int, error) {
	// Trim whitespace
	trimmed := strings.TrimSpace(priority)

	// Check for empty string
	if trimmed == "" {
		return 0, fmt.Errorf("task priority cannot be empty")
	}

	// Parse as integer
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid task priority %q: must be a number between 1-10", priority)
	}

	// Validate range
	if value < 1 || value > 10 {
		return 0, fmt.Errorf("invalid task priority %d: must be between 1-10", value)
	}

	return value, nil
}
