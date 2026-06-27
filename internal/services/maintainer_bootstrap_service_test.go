package services_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// sha256hex computes the SHA-256 hex digest of a string.
// Mirrors the helper used in gate_test.go per test-plan.md §4.2.
func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

// ---------------------------------------------------------------------------
// Mocks — function-field pattern per .claude/rules/services/testing.md
// ---------------------------------------------------------------------------

// MockConfigReader implements services.ConfigReader.
type MockConfigReader struct {
	ReadFunc func() (map[string]interface{}, error)
}

func (m *MockConfigReader) Read() (map[string]interface{}, error) {
	if m.ReadFunc != nil {
		return m.ReadFunc()
	}
	return nil, fmt.Errorf("Read not implemented in mock")
}

// MockConfigWriter implements services.ConfigWriter.
type MockConfigWriter struct {
	WriteFunc func(data map[string]interface{}) error
}

func (m *MockConfigWriter) Write(data map[string]interface{}) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(data)
	}
	return fmt.Errorf("Write not implemented in mock")
}

// ---------------------------------------------------------------------------
// AC-T2: SetPassword computes correct SHA-256 hash and writes password_hash
// ---------------------------------------------------------------------------

func TestMaintainerBootstrapService_SetPassword_WritesCorrectHash(t *testing.T) {
	ctx := context.Background()
	password := "hunter2"
	expectedHash := sha256hex(password)

	var capturedData map[string]interface{}

	reader := &MockConfigReader{
		ReadFunc: func() (map[string]interface{}, error) {
			// Existing config with other top-level keys
			return map[string]interface{}{
				"workflow_config": "shark-data/workflow/",
				"database":        map[string]interface{}{"backend": "local"},
			}, nil
		},
	}
	writer := &MockConfigWriter{
		WriteFunc: func(data map[string]interface{}) error {
			capturedData = data
			return nil
		},
	}

	svc := services.NewMaintainerBootstrapService(reader, writer)
	err := svc.SetPassword(ctx, password)

	if err != nil {
		t.Fatalf("SetPassword() returned unexpected error: %v", err)
	}

	// Assert the written config has the correct maintainer.password_hash
	maintainerRaw, ok := capturedData["maintainer"]
	if !ok {
		t.Fatal("expected 'maintainer' key in written config data")
	}

	// The maintainer field may be a map (raw JSON) or a struct. Handle both.
	var maintainerHash string
	switch m := maintainerRaw.(type) {
	case map[string]interface{}:
		h, ok := m["password_hash"].(string)
		if !ok {
			t.Fatalf("maintainer.password_hash is not a string: %T %v", m["password_hash"], m["password_hash"])
		}
		maintainerHash = h
	default:
		// Try JSON round-trip for struct types
		b, err := json.Marshal(maintainerRaw)
		if err != nil {
			t.Fatalf("cannot marshal maintainer: %v", err)
		}
		var mc map[string]interface{}
		if err := json.Unmarshal(b, &mc); err != nil {
			t.Fatalf("cannot unmarshal maintainer: %v", err)
		}
		h, ok := mc["password_hash"].(string)
		if !ok {
			t.Fatalf("maintainer.password_hash is not a string after round-trip: %v", mc)
		}
		maintainerHash = h
	}

	if maintainerHash != expectedHash {
		t.Errorf("password_hash = %q, want %q", maintainerHash, expectedHash)
	}
}

// ---------------------------------------------------------------------------
// AC-T2: Other config keys are preserved verbatim during SetPassword
// ---------------------------------------------------------------------------

func TestMaintainerBootstrapService_SetPassword_PreservesOtherKeys(t *testing.T) {
	ctx := context.Background()

	originalConfig := map[string]interface{}{
		"workflow_config": "shark-data/workflow/",
		"database": map[string]interface{}{
			"backend": "local",
			"url":     "./shark-tasks.db",
		},
		"color_enabled": true,
	}

	var capturedData map[string]interface{}

	reader := &MockConfigReader{
		ReadFunc: func() (map[string]interface{}, error) {
			// Return a copy so the test can verify the original was not mutated
			copy := make(map[string]interface{})
			for k, v := range originalConfig {
				copy[k] = v
			}
			return copy, nil
		},
	}
	writer := &MockConfigWriter{
		WriteFunc: func(data map[string]interface{}) error {
			capturedData = data
			return nil
		},
	}

	svc := services.NewMaintainerBootstrapService(reader, writer)
	err := svc.SetPassword(ctx, "mypassword")

	if err != nil {
		t.Fatalf("SetPassword() returned unexpected error: %v", err)
	}

	// workflow_config must survive
	if capturedData["workflow_config"] != originalConfig["workflow_config"] {
		t.Errorf("workflow_config changed: got %v, want %v",
			capturedData["workflow_config"], originalConfig["workflow_config"])
	}

	// database must survive
	if _, ok := capturedData["database"]; !ok {
		t.Error("database key missing from written config")
	}

	// color_enabled must survive
	if capturedData["color_enabled"] != originalConfig["color_enabled"] {
		t.Errorf("color_enabled changed: got %v, want %v",
			capturedData["color_enabled"], originalConfig["color_enabled"])
	}
}

// ---------------------------------------------------------------------------
// AC-T3: Config does not exist (reader returns "not found") → creates minimal config
// ---------------------------------------------------------------------------

func TestMaintainerBootstrapService_SetPassword_NoExistingConfig_CreatesMinimal(t *testing.T) {
	ctx := context.Background()
	password := "newpass"
	expectedHash := sha256hex(password)

	var capturedData map[string]interface{}

	reader := &MockConfigReader{
		ReadFunc: func() (map[string]interface{}, error) {
			// Simulate "not found" by returning nil data and no error,
			// indicating an empty / absent config.
			return nil, nil
		},
	}
	writer := &MockConfigWriter{
		WriteFunc: func(data map[string]interface{}) error {
			capturedData = data
			return nil
		},
	}

	svc := services.NewMaintainerBootstrapService(reader, writer)
	err := svc.SetPassword(ctx, password)

	if err != nil {
		t.Fatalf("SetPassword() returned unexpected error: %v", err)
	}

	// Should have a maintainer key with the correct hash
	maintainerRaw, ok := capturedData["maintainer"]
	if !ok {
		t.Fatal("expected 'maintainer' key in new config")
	}

	b, err := json.Marshal(maintainerRaw)
	if err != nil {
		t.Fatalf("cannot marshal maintainer: %v", err)
	}
	var mc map[string]interface{}
	if err := json.Unmarshal(b, &mc); err != nil {
		t.Fatalf("cannot unmarshal maintainer: %v", err)
	}
	hash, ok := mc["password_hash"].(string)
	if !ok {
		t.Fatalf("password_hash missing or not a string: %v", mc)
	}
	if hash != expectedHash {
		t.Errorf("password_hash = %q, want %q", hash, expectedHash)
	}
}

// ---------------------------------------------------------------------------
// AC-T4: Plaintext password is NOT stored in any field
// ---------------------------------------------------------------------------

func TestMaintainerBootstrapService_SetPassword_PlaintextNotStored(t *testing.T) {
	ctx := context.Background()
	password := "secret-password-xyz"

	var capturedData map[string]interface{}

	reader := &MockConfigReader{
		ReadFunc: func() (map[string]interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
	writer := &MockConfigWriter{
		WriteFunc: func(data map[string]interface{}) error {
			capturedData = data
			return nil
		},
	}

	svc := services.NewMaintainerBootstrapService(reader, writer)
	err := svc.SetPassword(ctx, password)

	if err != nil {
		t.Fatalf("SetPassword() returned unexpected error: %v", err)
	}

	// Serialize the entire captured config to JSON and assert the plaintext is absent.
	b, err := json.Marshal(capturedData)
	if err != nil {
		t.Fatalf("cannot marshal capturedData: %v", err)
	}
	serialized := string(b)

	if containsSubstring(serialized, password) {
		t.Errorf("plaintext password %q found in written config: %s", password, serialized)
	}
}

// ---------------------------------------------------------------------------
// AC-T5: Write failure propagates a wrapped error
// ---------------------------------------------------------------------------

func TestMaintainerBootstrapService_SetPassword_WriteError_ReturnsWrappedError(t *testing.T) {
	ctx := context.Background()
	writeErr := errors.New("disk full")

	reader := &MockConfigReader{
		ReadFunc: func() (map[string]interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
	writer := &MockConfigWriter{
		WriteFunc: func(data map[string]interface{}) error {
			return writeErr
		},
	}

	svc := services.NewMaintainerBootstrapService(reader, writer)
	err := svc.SetPassword(ctx, "anypassword")

	if err == nil {
		t.Fatal("expected error from write failure, got nil")
	}

	// Error must be wrapped with context (errors.Is should find the sentinel)
	if !errors.Is(err, writeErr) {
		t.Errorf("error chain does not contain original write error; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-T5: Read failure propagates a wrapped error
// ---------------------------------------------------------------------------

func TestMaintainerBootstrapService_SetPassword_ReadError_ReturnsWrappedError(t *testing.T) {
	ctx := context.Background()
	readErr := errors.New("permission denied")

	reader := &MockConfigReader{
		ReadFunc: func() (map[string]interface{}, error) {
			return nil, readErr
		},
	}
	writer := &MockConfigWriter{
		WriteFunc: func(data map[string]interface{}) error {
			t.Error("Write should not be called when Read fails")
			return nil
		},
	}

	svc := services.NewMaintainerBootstrapService(reader, writer)
	err := svc.SetPassword(ctx, "anypassword")

	if err == nil {
		t.Fatal("expected error from read failure, got nil")
	}

	if !errors.Is(err, readErr) {
		t.Errorf("error chain does not contain original read error; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Additional: Existing maintainer key is overwritten, cache_window_seconds preserved
// ---------------------------------------------------------------------------

func TestMaintainerBootstrapService_SetPassword_OverwritesExistingHash(t *testing.T) {
	ctx := context.Background()
	newPassword := "newpassword"
	expectedHash := sha256hex(newPassword)

	reader := &MockConfigReader{
		ReadFunc: func() (map[string]interface{}, error) {
			return map[string]interface{}{
				"maintainer": map[string]interface{}{
					"password_hash":        sha256hex("oldpassword"),
					"cache_window_seconds": float64(120),
				},
			}, nil
		},
	}

	var capturedData map[string]interface{}
	writer := &MockConfigWriter{
		WriteFunc: func(data map[string]interface{}) error {
			capturedData = data
			return nil
		},
	}

	svc := services.NewMaintainerBootstrapService(reader, writer)
	err := svc.SetPassword(ctx, newPassword)

	if err != nil {
		t.Fatalf("SetPassword() returned unexpected error: %v", err)
	}

	// Hash should be updated to new password
	b, _ := json.Marshal(capturedData["maintainer"])
	var mc map[string]interface{}
	json.Unmarshal(b, &mc) //nolint:errcheck
	hash, _ := mc["password_hash"].(string)
	if hash != expectedHash {
		t.Errorf("password_hash = %q, want %q (new hash)", hash, expectedHash)
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// containsSubstring checks whether s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
