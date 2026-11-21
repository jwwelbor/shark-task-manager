package repoerr

import (
	"database/sql"
	"errors"
	"strings"
)

// ErrNotFound is returned by repositories when a requested entity does not
// exist. Callers should use errors.Is(err, ErrNotFound) to detect it.
var ErrNotFound = errors.New("repository not found")

// ErrDatabaseFailure marks a database operation error that callers may map to
// the CLI's database-error exit code while preserving the underlying cause.
var ErrDatabaseFailure = errors.New("repository database failure")

// ErrConditionalWriteConflict marks an optimistic conditional write that
// matched no row because a concurrent mutation changed its expected state.
var ErrConditionalWriteConflict = errors.New("repository conditional write conflict")

// IsNotFound identifies the repository's stable missing-record sentinel and
// the database/sql no-row result still exposed by a small number of legacy
// repository paths.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, sql.ErrNoRows)
}

// IsDatabaseFailure identifies database failures explicitly marked by a
// repository. Other application and validation errors are left unclassified.
func IsDatabaseFailure(err error) bool {
	return errors.Is(err, ErrDatabaseFailure)
}

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
