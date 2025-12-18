package keygen

import (
	"path/filepath"
	"testing"
)

func TestPathParser_ParsePath(t *testing.T) {
	parser := NewPathParser("/home/user/project")

	tests := []struct {
		name           string
		filePath       string
		wantEpicKey    string
		wantFeatureKey string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "standard path with tasks folder",
			filePath:       "/home/user/project/docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/tasks/auth.prp.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F02",
			wantErr:        false,
		},
		{
			name:           "standard path with prps folder",
			filePath:       "/home/user/project/docs/plan/E04-task-mgmt/E04-F07-initialization-sync/prps/implement-caching.prp.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F07",
			wantErr:        false,
		},
		{
			name:           "path with project number",
			filePath:       "/home/user/project/docs/plan/E09-game-engine/E09-P02-F01-character-mgmt/tasks/create-character.prp.md",
			wantEpicKey:    "E09",
			wantFeatureKey: "E09-P02-F01",
			wantErr:        false,
		},
		{
			name:           "path without tasks/prps subfolder",
			filePath:       "/home/user/project/docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/auth.prp.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F02",
			wantErr:        false,
		},
		{
			name:        "invalid path - missing feature folder",
			filePath:    "/home/user/project/docs/plan/E04-task-mgmt/tasks/random.md",
			wantErr:     true,
			errContains: "cannot infer epic/feature from path",
		},
		{
			name:        "invalid path - random folder structure",
			filePath:    "/home/user/project/docs/random-folder/file.md",
			wantErr:     true,
			errContains: "cannot infer epic/feature from path",
		},
		{
			name:        "invalid path - no epic in hierarchy",
			filePath:    "/tmp/random/E04-F02-feature/tasks/file.md",
			wantErr:     true,
			errContains: "epic folder 'E04' not found in path hierarchy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParsePath(tt.filePath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePath() expected error containing '%s', got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ParsePath() error = '%v', want error containing '%s'", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParsePath() unexpected error = %v", err)
				return
			}

			if result.EpicKey != tt.wantEpicKey {
				t.Errorf("ParsePath() EpicKey = %v, want %v", result.EpicKey, tt.wantEpicKey)
			}

			if result.FeatureKey != tt.wantFeatureKey {
				t.Errorf("ParsePath() FeatureKey = %v, want %v", result.FeatureKey, tt.wantFeatureKey)
			}

			// Verify absolute path is returned
			if !filepath.IsAbs(result.FilePath) {
				t.Errorf("ParsePath() FilePath should be absolute, got %v", result.FilePath)
			}
		})
	}
}

func TestPathParser_ValidatePath(t *testing.T) {
	parser := NewPathParser("/home/user/project")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path",
			path:    "/home/user/project/docs/plan/E04-task-mgmt/E04-F02-cli/tasks/auth.prp.md",
			wantErr: false,
		},
		{
			name:    "invalid path",
			path:    "/home/user/project/docs/random/file.md",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
