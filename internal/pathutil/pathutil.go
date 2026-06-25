// Package pathutil holds small filesystem path helpers shared across packages
// that must not import one another (notably config and templates, which avoid
// a direct import edge). Keeping these in a leaf package lets every layer apply
// identical path semantics (e.g. "~/" expansion) without duplicating the logic.
package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome expands a leading "~/" in path to the user's home directory.
// If path does not start with "~/", or the home directory cannot be
// determined, path is returned unchanged.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
