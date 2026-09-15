package keys

import "fmt"

// DuplicateKeyError formats the consistent custom-key collision error used by
// entity creation flows. A suggestion is optional because it is best-effort.
func DuplicateKeyError(entityKind, key, suggestion string) error {
	if suggestion != "" {
		return fmt.Errorf("%s with key %q already exists (next available: %s)", entityKind, key, suggestion)
	}
	return fmt.Errorf("%s with key %q already exists", entityKind, key)
}
