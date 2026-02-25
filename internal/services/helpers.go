package services

import "fmt"

// requireNonNil panics if value is nil. Used in service constructors to validate
// required dependencies at construction time rather than at first use.
func requireNonNil(value interface{}, name string) {
	if value == nil {
		panic(fmt.Sprintf("%s must not be nil", name))
	}
}
