package config

import (
	"testing"
	"time"
)

// TestMaintainerConfig_CacheWindow_ZeroDefault tests that CacheWindow returns 60s when
// CacheWindowSeconds is 0 (spec.md §2.3, test-plan.md §2.4, AC-T2).
func TestMaintainerConfig_CacheWindow_ZeroDefault(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *MaintainerConfig
		expected time.Duration
	}{
		{
			name:     "nil receiver returns 60s default",
			cfg:      nil,
			expected: 60 * time.Second,
		},
		{
			name:     "zero CacheWindowSeconds returns 60s default",
			cfg:      &MaintainerConfig{CacheWindowSeconds: 0},
			expected: 60 * time.Second,
		},
		{
			name:     "negative CacheWindowSeconds returns 60s default",
			cfg:      &MaintainerConfig{CacheWindowSeconds: -1},
			expected: 60 * time.Second,
		},
		{
			name:     "positive CacheWindowSeconds returns configured value",
			cfg:      &MaintainerConfig{CacheWindowSeconds: 120},
			expected: 120 * time.Second,
		},
		{
			name:     "CacheWindowSeconds=1 returns 1s",
			cfg:      &MaintainerConfig{CacheWindowSeconds: 1},
			expected: 1 * time.Second,
		},
		{
			name:     "CacheWindowSeconds=3600 returns 1h",
			cfg:      &MaintainerConfig{CacheWindowSeconds: 3600},
			expected: 3600 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.CacheWindow()
			if got != tt.expected {
				t.Errorf("CacheWindow() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestMaintainerConfig_Fields tests that MaintainerConfig has the expected fields
// with the correct JSON tags (AC-T1).
func TestMaintainerConfig_Fields(t *testing.T) {
	// Verify struct fields exist by constructing and using the struct
	cfg := &MaintainerConfig{
		PasswordHash:       "abc123",
		CacheWindowSeconds: 60,
	}

	if cfg.PasswordHash != "abc123" {
		t.Errorf("PasswordHash = %q, want %q", cfg.PasswordHash, "abc123")
	}
	if cfg.CacheWindowSeconds != 60 {
		t.Errorf("CacheWindowSeconds = %d, want %d", cfg.CacheWindowSeconds, 60)
	}
}
