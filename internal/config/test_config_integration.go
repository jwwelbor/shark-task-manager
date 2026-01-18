package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestManager_LoadRequireRejectionReason(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"require_rejection_reason": true,
		"color_enabled":            true,
	}

	data, _ := json.MarshalIndent(configData, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager := NewManager(configPath)
	cfg, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.RequireRejectionReason {
		t.Error("RequireRejectionReason should be true")
	}

	if !cfg.IsRequireRejectionReasonEnabled() {
		t.Error("IsRequireRejectionReasonEnabled() should return true")
	}
}
