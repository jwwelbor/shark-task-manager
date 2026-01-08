package db

import (
	"os"
	"testing"
)

// TestLoadAuthToken tests loading authentication tokens
func TestLoadAuthToken(t *testing.T) {
	// Create temporary token file
	tmpFile := t.TempDir() + "/token.txt"
	err := os.WriteFile(tmpFile, []byte("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test-token"), 0600)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		envVar    string
		wantToken string
		wantErr   bool
	}{
		{
			name:      "Load from file",
			input:     tmpFile,
			wantToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test-token",
			wantErr:   false,
		},
		{
			name:      "Direct token (JWT format)",
			input:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.direct-token",
			wantToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.direct-token",
			wantErr:   false,
		},
		{
			name:      "Empty input (no token)",
			input:     "",
			wantToken: "",
			wantErr:   false,
		},
		{
			name:      "Nonexistent file",
			input:     "/nonexistent/token.txt",
			wantToken: "",
			wantErr:   true,
		},
	}

	// Save and restore environment
	oldEnv := os.Getenv("TURSO_AUTH_TOKEN")
	defer os.Setenv("TURSO_AUTH_TOKEN", oldEnv)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if specified
			if tt.envVar != "" {
				os.Setenv("TURSO_AUTH_TOKEN", tt.envVar)
			} else {
				os.Unsetenv("TURSO_AUTH_TOKEN")
			}

			token, err := LoadAuthToken(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if token != tt.wantToken {
					t.Errorf("LoadAuthToken() = %q, want %q", token, tt.wantToken)
				}
			}
		})
	}
}

// TestLoadAuthToken_FromEnvironment tests loading from environment variable
func TestLoadAuthToken_FromEnvironment(t *testing.T) {
	// Save and restore environment
	oldEnv := os.Getenv("TURSO_AUTH_TOKEN")
	defer os.Setenv("TURSO_AUTH_TOKEN", oldEnv)

	// Set environment variable
	os.Setenv("TURSO_AUTH_TOKEN", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.env-token")

	// Load with empty input (should use env var)
	token, err := LoadAuthToken("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.env-token"
	if token != expected {
		t.Errorf("LoadAuthToken() = %q, want %q", token, expected)
	}
}

// TestLoadAuthToken_EmptyFile tests handling of empty token file
func TestLoadAuthToken_EmptyFile(t *testing.T) {
	// Create empty file
	tmpFile := t.TempDir() + "/empty.txt"
	err := os.WriteFile(tmpFile, []byte(""), 0600)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	_, err = LoadAuthToken(tmpFile)
	if err == nil {
		t.Error("Expected error for empty file, got nil")
	}
}

// TestLoadAuthToken_FileWithWhitespace tests trimming of whitespace
func TestLoadAuthToken_FileWithWhitespace(t *testing.T) {
	tmpFile := t.TempDir() + "/whitespace.txt"
	err := os.WriteFile(tmpFile, []byte("  \neyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.token\n  "), 0600)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	token, err := LoadAuthToken(tmpFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.token"
	if token != expected {
		t.Errorf("LoadAuthToken() = %q, want %q", token, expected)
	}
}

// TestValidateAuthToken tests token validation
func TestValidateAuthToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "Valid JWT token",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ",
			wantErr: false,
		},
		{
			name:    "Empty token (valid for local dev)",
			token:   "",
			wantErr: false,
		},
		{
			name:    "Too short",
			token:   "eyJhbGciOiJ",
			wantErr: true,
		},
		{
			name:    "Invalid prefix",
			token:   "not-a-jwt-token-should-start-with-eyJ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
