package maintainer

import "time"

// clock is the interface for time operations, enabling clock injection in tests
// without exporting a public interface that adds unnecessary surface area.
//
// Spec reference: spec.md §2.5 F02-D1 (clock injection).
type clock interface {
	Now() time.Time
}

// realClock is the production clock implementation that delegates to time.Now.
type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}
