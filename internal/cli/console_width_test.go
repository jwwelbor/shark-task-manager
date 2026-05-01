package cli

import (
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestTitleColumnWidth_DerivesFromConsoleWidth verifies that TitleColumnWidth
// subtracts the reserved-column estimate from the resolved console width.
// CC-036.
func TestTitleColumnWidth_DerivesFromConsoleWidth(t *testing.T) {
	tests := []struct {
		name            string
		stubbedDetected int
		reservedColumns int
		wantTitleWidth  int
	}{
		{
			name:            "narrow terminal, modest reserved → minimum clamp",
			stubbedDetected: 60,
			reservedColumns: 50,
			wantTitleWidth:  20, // 60-50=10 → clamped to 20
		},
		{
			name:            "default-baseline (120) reproduces historical task list cap",
			stubbedDetected: 120,
			reservedColumns: 80,
			wantTitleWidth:  40, // matches old hardcoded 40
		},
		{
			name:            "default-baseline (120) reproduces historical feature get cap",
			stubbedDetected: 120,
			reservedColumns: 60,
			wantTitleWidth:  60, // matches old hardcoded 60
		},
		{
			name:            "wide terminal yields wider title",
			stubbedDetected: 200,
			reservedColumns: 80,
			wantTitleWidth:  120,
		},
		{
			name:            "extremely narrow terminal still renders min title",
			stubbedDetected: 30,
			reservedColumns: 80,
			wantTitleWidth:  20, // 30-80=-50 → clamped to 20
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStubbedConsoleWidth(t, tt.stubbedDetected, func() {
				got := TitleColumnWidth(tt.reservedColumns)
				if got != tt.wantTitleWidth {
					t.Errorf("TitleColumnWidth(%d) = %d, want %d (resolved width = %d)",
						tt.reservedColumns, got, tt.wantTitleWidth, tt.stubbedDetected)
				}
			})
		})
	}
}

// TestGetConsoleWidth_TerminalDetectionFallback verifies that when
// terminal-size detection fails (returns 0) and no config override is set,
// GetConsoleWidth falls through to config.DefaultConsoleWidth.
// CC-036.
func TestGetConsoleWidth_TerminalDetectionFallback(t *testing.T) {
	withStubbedConsoleWidth(t, 0, func() {
		got := GetConsoleWidth()
		if got != config.DefaultConsoleWidth {
			t.Errorf("GetConsoleWidth() with detection=0 = %d, want %d (default)",
				got, config.DefaultConsoleWidth)
		}
	})
}

// TestGetConsoleWidth_DetectionUsedWhenConfigUnset verifies that when no
// .sharkconfig.json override is set but detection succeeds, the detected
// width is returned verbatim. CC-036.
func TestGetConsoleWidth_DetectionUsedWhenConfigUnset(t *testing.T) {
	withStubbedConsoleWidth(t, 95, func() {
		got := GetConsoleWidth()
		if got != 95 {
			t.Errorf("GetConsoleWidth() with detection=95 = %d, want 95", got)
		}
	})
}

// withStubbedConsoleWidth runs fn with detectTerminalWidth temporarily
// returning `detected`, the console-width cache cleared before and after,
// and GlobalConfig.ConfigFile pointed at /dev/null so the config loader
// returns an empty Config (ConsoleWidth=0, the auto-detect sentinel).
//
// This isolates GetConsoleWidth from the host environment so tests run
// deterministically regardless of whether the test runner is attached to
// a TTY or has a real .sharkconfig.json on disk.
func withStubbedConsoleWidth(t *testing.T, detected int, fn func()) {
	t.Helper()

	prevDetect := detectTerminalWidth
	prevCfgFile := GlobalConfig.ConfigFile
	t.Cleanup(func() {
		detectTerminalWidth = prevDetect
		GlobalConfig.ConfigFile = prevCfgFile
		resetConsoleWidthCache()
	})

	detectTerminalWidth = func() int { return detected }
	// Point at a missing path inside a temp dir so config.Manager.Load
	// returns an empty Config with ConsoleWidth=0 (the unset sentinel).
	// Manager.Load specifically handles os.IsNotExist by returning an empty
	// config with no error.
	GlobalConfig.ConfigFile = filepath.Join(t.TempDir(), "missing-sharkconfig.json")
	resetConsoleWidthCache()

	fn()
}
