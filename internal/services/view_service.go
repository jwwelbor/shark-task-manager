package services

import (
	"fmt"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/cli/scope"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
)

// ViewService provides file path resolution for entity view operations.
type ViewService struct{}

// NewViewService creates a new ViewService.
func NewViewService() *ViewService {
	return &ViewService{}
}

// GetFilePath returns the relative file path for the given scope type and key.
//
// Supported scope types:
//   - ScopeBug: resolves to docs/plan/bugs/<key>.md
//   - ScopeChange: resolves to docs/plan/changes/<key>.md
//
// Returns an error for unsupported scope types (epic, feature, task use
// different path resolution that depends on project structure).
func (s *ViewService) GetFilePath(scopeType scope.ScopeType, key string) (string, error) {
	switch scopeType {
	case scope.ScopeBug:
		return filepath.Join("docs", "plan", "bugs", key+".md"), nil
	case scope.ScopeChange, scope.ScopeChangeCard:
		if canonicalKey, err := keys.NormalizeChangeKey(key); err == nil {
			key = canonicalKey
		}
		return filepath.Join("docs", "plan", "changes", key+".md"), nil
	default:
		return "", fmt.Errorf("unsupported scope type: %q", scopeType)
	}
}
