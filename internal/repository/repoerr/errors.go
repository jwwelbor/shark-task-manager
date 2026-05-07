package repoerr

import "errors"

// ErrNotFound is returned by repositories when a requested entity does not
// exist. Callers should use errors.Is(err, ErrNotFound) to detect it.
var ErrNotFound = errors.New("repository not found")
