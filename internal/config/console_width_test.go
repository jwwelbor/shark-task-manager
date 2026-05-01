package config

import "testing"

// TestGetConsoleWidth_NilConfig verifies that calling GetConsoleWidth on a
// nil *Config falls through to the detected width / DefaultConsoleWidth.
func TestGetConsoleWidth_NilConfig(t *testing.T) {
	tests := []struct {
		name     string
		detected int
		want     int
	}{
		{"detection failed → default", 0, DefaultConsoleWidth},
		{"detection failed (negative) → default", -5, DefaultConsoleWidth},
		{"detected width passes through", 95, 95},
		{"detected wide width passes through", 200, 200},
	}
	var cfg *Config
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetConsoleWidth(tt.detected)
			if got != tt.want {
				t.Errorf("GetConsoleWidth(%d) = %d, want %d", tt.detected, got, tt.want)
			}
		})
	}
}

// TestGetConsoleWidth_UnsetField verifies that an unset (zero) ConsoleWidth
// behaves identically to a nil config — auto-detect with default fallback.
func TestGetConsoleWidth_UnsetField(t *testing.T) {
	cfg := &Config{}
	if got := cfg.GetConsoleWidth(0); got != DefaultConsoleWidth {
		t.Errorf("GetConsoleWidth(0) on empty config = %d, want %d", got, DefaultConsoleWidth)
	}
	if got := cfg.GetConsoleWidth(150); got != 150 {
		t.Errorf("GetConsoleWidth(150) on empty config = %d, want 150", got)
	}
}

// TestGetConsoleWidth_ExplicitValue verifies that a positive ConsoleWidth
// in config takes precedence over both the detected width and the default.
func TestGetConsoleWidth_ExplicitValue(t *testing.T) {
	tests := []struct {
		name        string
		configWidth int
		detected    int
		want        int
	}{
		{"explicit value wins over detected", 80, 200, 80},
		{"explicit value wins over default", 100, 0, 100},
		{"explicit equal to MinConsoleWidth", MinConsoleWidth, 0, MinConsoleWidth},
		{"explicit just above MinConsoleWidth", MinConsoleWidth + 1, 0, MinConsoleWidth + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ConsoleWidth: tt.configWidth}
			if got := cfg.GetConsoleWidth(tt.detected); got != tt.want {
				t.Errorf("GetConsoleWidth() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestGetConsoleWidth_ClampsBelowMinimum verifies that very small explicit
// widths are clamped up to MinConsoleWidth so list views remain renderable.
func TestGetConsoleWidth_ClampsBelowMinimum(t *testing.T) {
	tests := []struct {
		name        string
		configWidth int
	}{
		{"width=1 clamps to min", 1},
		{"width=10 clamps to min", 10},
		{"width=39 (just below min) clamps to min", MinConsoleWidth - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ConsoleWidth: tt.configWidth}
			got := cfg.GetConsoleWidth(0)
			if got != MinConsoleWidth {
				t.Errorf("GetConsoleWidth() = %d, want %d (clamped)", got, MinConsoleWidth)
			}
		})
	}
}

// TestGetConsoleWidth_NegativeFieldFallsThrough verifies that a negative
// ConsoleWidth (treated as "unset") falls through to detection/default,
// mirroring the zero-value behavior.
func TestGetConsoleWidth_NegativeFieldFallsThrough(t *testing.T) {
	cfg := &Config{ConsoleWidth: -1}
	if got := cfg.GetConsoleWidth(0); got != DefaultConsoleWidth {
		t.Errorf("GetConsoleWidth(0) with negative ConsoleWidth = %d, want %d", got, DefaultConsoleWidth)
	}
	if got := cfg.GetConsoleWidth(140); got != 140 {
		t.Errorf("GetConsoleWidth(140) with negative ConsoleWidth = %d, want 140", got)
	}
}
