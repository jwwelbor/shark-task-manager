package services

// TagQueryOp is the multi-tag filter operator for EntityIDsByTags.
// Only "and" is supported in v1 (see Epic Architecture ADR-5).
type TagQueryOp string

const (
	// TagQueryOpAnd is the AND intersection operator — all supplied tag names
	// must be attached to the entity for it to be included in the result.
	// This is the only supported operator in v1.
	TagQueryOpAnd TagQueryOp = "and"
)

// normalize returns the canonical op. An empty string defaults to AND
// so callers (CLI, HTTP) can pass "" without special-casing.
func (o TagQueryOp) normalize() TagQueryOp {
	if o == "" {
		return TagQueryOpAnd
	}
	return o
}
