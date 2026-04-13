package services

import (
	"fmt"
	"os"
	"strings"
)

// requireNonNil panics if value is nil. Used in service constructors to validate
// required dependencies at construction time rather than at first use.
func requireNonNil(value interface{}, name string) {
	if value == nil {
		panic(fmt.Sprintf("%s must not be nil", name))
	}
}

// isContained reports whether targetCanon is equal to rootCanon or is a direct
// descendant of it. Both arguments must be canonicalized (EvalSymlinks-resolved)
// absolute paths. Used by path-security checks in viewer and edit services.
func isContained(rootCanon, targetCanon string) bool {
	return targetCanon == rootCanon || strings.HasPrefix(targetCanon, rootCanon+string(os.PathSeparator))
}
