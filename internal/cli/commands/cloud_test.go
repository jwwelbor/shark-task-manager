package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCloudInit_ValidatesURL tests URL validation
func TestCloudInit_ValidatesURL(t *testing.T) {
	// RED PHASE: This test should FAIL

	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid turso URL",
			url:         "libsql://mydb.turso.io",
			expectError: false,
		},
		{
			name:        "valid turso URL with path",
			url:         "libsql://mydb.turso.io/dbname",
			expectError: false,
		},
		{
			name:        "valid https URL",
			url:         "https://mydb.turso.io",
			expectError: false,
		},
		{
			name:        "invalid scheme",
			url:         "http://mydb.turso.io",
			expectError: true,
			errorMsg:    "URL must use libsql:// scheme for Turso",
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
			errorMsg:    "database URL is required",
		},
		{
			name:        "missing scheme",
			url:         "mydb.turso.io",
			expectError: true,
			errorMsg:    "invalid URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTursoURL(tt.url)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCloudInit_HandlesAuthToken tests auth token handling
func TestCloudInit_HandlesAuthToken(t *testing.T) {
	// Use a realistic mock JWT token (>50 chars, starts with eyJ)
	mockToken := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.test_signature_here"

	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string // Returns auth token or file path
		expectError bool
	}{
		{
			name: "direct token",
			setupFunc: func(t *testing.T) string {
				return mockToken
			},
			expectError: false,
		},
		{
			name: "token from file",
			setupFunc: func(t *testing.T) string {
				tempFile := filepath.Join(t.TempDir(), "token.txt")
				err := os.WriteFile(tempFile, []byte(mockToken), 0600)
				require.NoError(t, err)
				return tempFile
			},
			expectError: false,
		},
		{
			name: "token from environment",
			setupFunc: func(t *testing.T) string {
				t.Setenv("TURSO_AUTH_TOKEN", mockToken)
				return "" // Empty means use env var
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authInput := tt.setupFunc(t)

			// This function should be implemented in cloud.go
			token, err := resolveAuthToken(authInput)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
				assert.Contains(t, token, "eyJ") // JWT prefix
			}
		})
	}
}

// TestMaskToken tests token masking for display
func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "long token",
			token:    "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.test.signature",
			expected: "eyJh...ture",
		},
		{
			name:     "short token",
			token:    "short",
			expected: "***",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "(none)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCloudStatus tests the cloud status command
func TestCloudStatus(t *testing.T) {
	// RED PHASE: Test should fail initially

	tests := []struct {
		name             string
		setupConfig      func(t *testing.T) string // Returns config file path
		expectBackend    string
		expectURL        string
		expectConfigured bool
	}{
		{
			name: "turso configured",
			setupConfig: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, ".sharkconfig.json")

				cfgData := map[string]interface{}{
					"database": map[string]interface{}{
						"backend":         "turso",
						"url":             "libsql://test.turso.io",
						"auth_token_file": "/path/to/token",
					},
				}

				data, err := json.MarshalIndent(cfgData, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(configPath, data, 0644)
				require.NoError(t, err)

				return configPath
			},
			expectBackend:    "turso",
			expectURL:        "libsql://test.turso.io",
			expectConfigured: true,
		},
		{
			name: "local sqlite configured",
			setupConfig: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, ".sharkconfig.json")

				cfgData := map[string]interface{}{
					"database": map[string]interface{}{
						"backend": "local",
						"url":     "./shark-tasks.db",
					},
				}

				data, err := json.MarshalIndent(cfgData, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(configPath, data, 0644)
				require.NoError(t, err)

				return configPath
			},
			expectBackend:    "local",
			expectURL:        "./shark-tasks.db",
			expectConfigured: false, // Not cloud
		},
		{
			name: "no config file",
			setupConfig: func(t *testing.T) string {
				tempDir := t.TempDir()
				return filepath.Join(tempDir, ".sharkconfig.json") // Doesn't exist
			},
			expectBackend:    "",
			expectURL:        "",
			expectConfigured: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupConfig(t)

			// Call getCloudStatus helper (to be implemented)
			status, err := getCloudStatus(configPath)

			require.NoError(t, err)
			assert.Equal(t, tt.expectBackend, status.Backend)
			assert.Equal(t, tt.expectURL, status.URL)
			assert.Equal(t, tt.expectConfigured, status.IsCloudConfigured)
		})
	}
}
