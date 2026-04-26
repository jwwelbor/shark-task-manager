package tag

import "errors"

// ErrEmptyTagIDs is returned by FilterEntityIDs when the caller provides an
// empty tagIDs slice. Passing an empty slice is a programming error; the
// service layer must never call FilterEntityIDs with zero tag IDs.
var ErrEmptyTagIDs = errors.New("entity_tag repository: FilterEntityIDs requires at least one tagID")
