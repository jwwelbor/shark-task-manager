package repository

import (
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
)

// DB is the canonical database connection type for all repositories.
// It is a type alias for dbconn.DB, making dbconn.DB the single source
// of truth while preserving full backward compatibility: all existing code
// that uses *repository.DB continues to work without modification.
//
// Type aliases (type T = U) are resolved at compile time with zero runtime
// overhead. Assignment and type assertion compatibility is preserved.
type DB = dbconn.DB

// NewDB creates a new DB instance wrapping the provided *sql.DB.
// All callers of repository.NewDB continue to work unchanged via this alias.
var NewDB = dbconn.NewDB

// ErrNotFound is returned by repositories when a requested entity does not
// exist. Callers should use errors.Is(err, ErrNotFound) to detect it.
var ErrNotFound = repoerr.ErrNotFound
