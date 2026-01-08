package db

import (
	"fmt"
	"os"
	"strings"
)

// LoadAuthToken loads authentication token from file or environment variable
// Priority: 1) File path, 2) Environment variable, 3) Direct token value
func LoadAuthToken(authTokenFile string) (string, error) {
	// If empty, try environment variable
	if authTokenFile == "" {
		token := os.Getenv("TURSO_AUTH_TOKEN")
		if token != "" {
			return token, nil
		}
		return "", nil // No token configured (may be valid for local dev)
	}

	// If it looks like a direct token (starts with eyJ for JWT), return it
	if strings.HasPrefix(authTokenFile, "eyJ") {
		return authTokenFile, nil
	}

	// Otherwise, treat as file path
	data, err := os.ReadFile(authTokenFile)
	if err != nil {
		return "", fmt.Errorf("failed to read auth token from %s: %w", authTokenFile, err)
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("auth token file %s is empty", authTokenFile)
	}

	return token, nil
}

// ValidateAuthToken performs basic validation on auth token format
func ValidateAuthToken(token string) error {
	if token == "" {
		return nil // Empty token may be valid (local development)
	}

	// Basic JWT format check (tokens typically start with eyJ)
	if !strings.HasPrefix(token, "eyJ") {
		return fmt.Errorf("auth token does not appear to be a valid JWT (should start with 'eyJ')")
	}

	// Check for reasonable length (JWTs are typically 100-500 characters)
	if len(token) < 50 {
		return fmt.Errorf("auth token is too short (expected at least 50 characters, got %d)", len(token))
	}

	return nil
}
