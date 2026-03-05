package services

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli/scope"
)

// TestViewService_GetFilePath tests file path resolution for all scope types.
func TestViewService_GetFilePath(t *testing.T) {
	tests := []struct {
		name        string
		scopeType   scope.ScopeType
		key         string
		wantPath    string
		wantErr     bool
		errContains string
	}{
		{
			name:      "bug scope resolves to docs/plan/bugs/",
			scopeType: scope.ScopeBug,
			key:       "B001",
			wantPath:  "docs/plan/bugs/B001.md",
			wantErr:   false,
		},
		{
			name:      "bug scope with different key",
			scopeType: scope.ScopeBug,
			key:       "B042",
			wantPath:  "docs/plan/bugs/B042.md",
			wantErr:   false,
		},
		{
			name:      "change scope resolves to docs/plan/changes/",
			scopeType: scope.ScopeChange,
			key:       "C001",
			wantPath:  "docs/plan/changes/C001.md",
			wantErr:   false,
		},
		{
			name:      "change scope with different key",
			scopeType: scope.ScopeChange,
			key:       "C099",
			wantPath:  "docs/plan/changes/C099.md",
			wantErr:   false,
		},
		{
			name:        "unknown scope type returns error",
			scopeType:   scope.ScopeType("unknown"),
			key:         "X001",
			wantPath:    "",
			wantErr:     true,
			errContains: "unsupported scope type",
		},
	}

	svc := NewViewService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetFilePath(tt.scopeType, tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetFilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !containsString(err.Error(), tt.errContains) {
					t.Errorf("GetFilePath() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if got != tt.wantPath {
				t.Errorf("GetFilePath() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

// TestViewService_GetFilePath_TaskNotSupported verifies task scope returns an error
// (tasks use a different path resolution that depends on epic/feature keys).
func TestViewService_GetFilePath_TaskNotSupported(t *testing.T) {
	svc := NewViewService()

	_, err := svc.GetFilePath(scope.ScopeTask, "T-E01-F01-001")
	if err == nil {
		t.Error("GetFilePath() with ScopeTask should return error, got nil")
	}
}

// containsString is a helper for checking error message content without importing strings.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
