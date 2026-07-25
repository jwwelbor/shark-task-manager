package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetMaxParallelItems(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want int
	}{
		{name: "nil config uses default", cfg: nil, want: DefaultMaxParallelItems},
		{name: "missing value uses default", cfg: &Config{}, want: DefaultMaxParallelItems},
		{name: "negative value uses default", cfg: &Config{MaxParallelItems: -1}, want: DefaultMaxParallelItems},
		{name: "positive value is honored", cfg: &Config{MaxParallelItems: 3}, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GetMaxParallelItems(); got != tt.want {
				t.Fatalf("GetMaxParallelItems() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestManagerLoadMaxParallelItems(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{"max_parallel_items": 2}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := NewManager(configPath).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.MaxParallelItems != 2 {
		t.Fatalf("MaxParallelItems = %d, want 2", cfg.MaxParallelItems)
	}
	if got := cfg.GetMaxParallelItems(); got != 2 {
		t.Fatalf("GetMaxParallelItems() = %d, want 2", got)
	}
}
