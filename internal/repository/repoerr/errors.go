package repoerr

import (
	"errors"
	"strings"
)

// ErrNotFound is returned by repositories when a requested entity does not
// exist. Callers should use errors.Is(err, ErrNotFound) to detect it.
var ErrNotFound = errors.New("repository not found")

// ErrConditionalWriteConflict marks an optimistic conditional write that
// matched no row because a concurrent mutation changed its expected state.
var ErrConditionalWriteConflict = errors.New("repository conditional write conflict")

// IsSQLiteUniqueViolation identifies the portable representations of a SQLite
// UNIQUE constraint failure. modernc/sqlite and libsql/Turso expose different,
// unexported driver error types, so errors.As cannot provide one shared
// contract. Keep this compatibility boundary narrow and limited to the
// observed driver messages rather than making each repository inspect errors.
func IsSQLiteUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "UNIQUE constraint failed") ||
		strings.Contains(message, "constraint failed: UNIQUE")
}
