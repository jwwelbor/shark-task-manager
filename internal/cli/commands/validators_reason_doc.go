package commands

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateRejectionReasonDocPath validates a document path for rejection reason linking
func ValidateRejectionReasonDocPath(docPath string) error {
	// Empty path check
	if docPath == "" {
		return fmt.Errorf("document path cannot be empty")
	}

	// Absolute path check
	if filepath.IsAbs(docPath) {
		return fmt.Errorf("document path must be relative")
	}

	// Path traversal check
	if strings.Contains(docPath, "..") {
		return fmt.Errorf("document path traversal not allowed")
	}

	return nil
}
