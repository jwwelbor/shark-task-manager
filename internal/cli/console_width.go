package cli

import (
	"os"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"golang.org/x/term"
)

// CC-036: Configurable console width for CLI list views.
//
// This file is the single source of truth for the resolved console width
// used by list/table renderers. Width-sensitive renderers should call
// GetConsoleWidth() instead of hardcoding column widths.
//
// Resolution order (highest priority first):
//  1. Explicit config value (`console_width` in .sharkconfig.json) when > 0,
//     clamped to >= config.MinConsoleWidth.
//  2. Auto-detected terminal width via golang.org/x/term.GetSize on stdout.
//  3. config.DefaultConsoleWidth fallback (used when not running under a TTY,
//     e.g., piped output, CI).

// detectTerminalWidth is overridable in tests so we can force a known width
// without depending on the test runner's controlling terminal.
var detectTerminalWidth = defaultDetectTerminalWidth

// defaultDetectTerminalWidth returns the controlling terminal's width in
// columns, or 0 if detection fails (e.g., stdout is not a TTY).
func defaultDetectTerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 0
	}
	return w
}

// consoleWidthCache memoizes the resolved console width within a single CLI
// invocation. Detection involves an ioctl, and renderers may consult the
// width once per row for many rows; caching avoids redundant syscalls and
// keeps the value stable across a single command's output.
var (
	consoleWidthCacheOnce sync.Once
	consoleWidthCached    int
)

// GetConsoleWidth returns the resolved console width for use by CLI list
// views. The value is cached per process — callers should treat this as a
// single, stable number for the lifetime of the command.
//
// Lookup order:
//  1. config.Config.ConsoleWidth (if positive, clamped to MinConsoleWidth)
//  2. golang.org/x/term auto-detect on stdout
//  3. config.DefaultConsoleWidth fallback
//
// Safe to call before any config has been loaded: a config load failure or
// missing field falls through to terminal detection.
func GetConsoleWidth() int {
	consoleWidthCacheOnce.Do(func() {
		var cfg *config.Config
		if loaded, err := GetConfig(); err == nil {
			cfg = loaded
		}
		consoleWidthCached = cfg.GetConsoleWidth(detectTerminalWidth())
	})
	return consoleWidthCached
}

// resetConsoleWidthCache clears the memoized console width. It exists so
// tests can force re-resolution after swapping detectTerminalWidth or
// adjusting GlobalConfig. Not exported.
func resetConsoleWidthCache() {
	consoleWidthCacheOnce = sync.Once{}
	consoleWidthCached = 0
}

// SetConsoleWidthForTesting forces GetConsoleWidth and TitleColumnWidth to
// return `width` for the duration of the returned restore function's
// lifetime, bypassing both .sharkconfig.json and terminal-size detection.
// Callers MUST defer (or call) the returned restore function to revert
// the override.
//
// This is a test seam for downstream packages (formatters, command tests)
// that want to verify width-sensitive rendering at fixed widths without
// duplicating cache-poking logic.
//
// Not for production use — wraps internal cache state.
func SetConsoleWidthForTesting(width int) func() {
	// Force the once-do to mark itself "done" so subsequent GetConsoleWidth
	// calls return the stubbed value without re-resolving from config or
	// terminal detection. We can't copy a sync.Once, so we re-allocate it
	// fresh and prime its cached value.
	resetConsoleWidthCache()
	consoleWidthCacheOnce.Do(func() {
		consoleWidthCached = width
	})

	return func() {
		resetConsoleWidthCache()
	}
}

// TitleColumnWidth returns the recommended max width for a "Title" /
// description column in list-view tables, derived from the resolved
// console width.
//
// The formula is intentionally simple and matches the existing layout
// intent: the historical 45-char title cap was chosen for an 85-90 column
// terminal where the surrounding columns (key, status, progress, health,
// size) consume ~40-45 columns. We preserve that ratio:
//
//	titleWidth = consoleWidth - reservedForOtherColumns
//
// `reservedForOtherColumns` is the caller-supplied estimate of fixed
// chrome (other columns + separators) for a given list view. We clamp the
// result to a minimum so list views remain legible even when the terminal
// is unrealistically narrow.
func TitleColumnWidth(reservedForOtherColumns int) int {
	const minTitleWidth = 20

	width := GetConsoleWidth() - reservedForOtherColumns
	if width < minTitleWidth {
		return minTitleWidth
	}
	return width
}
