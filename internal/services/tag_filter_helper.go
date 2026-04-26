package services

// filterByTagIDs applies an in-memory set-membership filter over ents by
// extracting each entity's ID via getID. Returns the filtered slice in-place
// (ents is reused — the caller must not retain the original backing array
// after this call). When taggedIDSet is nil, returns ents unchanged.
//
// When taggedIDSet is non-nil but empty, all entities are filtered out and
// a non-nil empty slice is returned (the ents[:0] expression preserves the
// non-nil property of the backing array).
//
// E28-F05 spec §2.5.2 "Block 2 — post-filter" pattern. Go generics are
// supported in Go 1.23.4+ per project minimum (spec §2.5.2 note).
func filterByTagIDs[E any](ents []E, taggedIDSet map[int64]struct{}, getID func(E) int64) []E {
	if taggedIDSet == nil {
		return ents
	}
	kept := ents[:0]
	for _, e := range ents {
		if _, ok := taggedIDSet[getID(e)]; ok {
			kept = append(kept, e)
		}
	}
	return kept
}
