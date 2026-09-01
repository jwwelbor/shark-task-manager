//go:build windows

package gaterun

import (
	"fmt"
	"os"
)

// openRunDirNoFollow is the Windows fallback counterpart of the Unix
// openat(O_NOFOLLOW|O_DIRECTORY) ancestor-chain implementation
// (dirhandle_unix.go). The standard library exposes no portable
// O_NOFOLLOW-equivalent open flag on this platform, so this opens runDir
// directly by path, leaving the same ancestor-directory TOCTOU window this
// package's Unix build closes unaddressed on Windows — a residual,
// documented gap (TD-181), consistent with this package's existing leaf-file
// Windows fallback (fsio_nofollow_windows.go) and its test suite, which
// skips the symlink-swap-race regression tests on windows for the same
// reason.
func openRunDirNoFollow(runDir string) (*os.File, error) {
	f, err := os.Open(runDir) // #nosec G304 -- runDir is joined from a validated project root; no portable no-follow primitive exists on windows.
	if err != nil {
		return nil, fmt.Errorf("gaterun: open run dir %s: %w", runDir, err)
	}
	return f, nil
}
