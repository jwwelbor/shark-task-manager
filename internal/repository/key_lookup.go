package repository

// key_lookup.go provides backward-compatible package-private wrappers around
// repoutil key-parsing utilities (ContainsHyphen, IsNumeric, SplitAtFirstHyphen).
//
// It also defines orderedItem as a package-private alias for repoutil.OrderedItem,
// preserving backward compatibility for root-package callers after order_resequence.go
// was merged into this file.

import "github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"

// orderedItem represents any entity with an ID and execution order.
// This is a type alias for repoutil.OrderedItem, allowing repository files
// to use the shorter name while delegating logic to repoutil.ResequenceOrders.
type orderedItem = repoutil.OrderedItem

// containsHyphen reports whether s contains a hyphen character.
// Delegates to repoutil.ContainsHyphen.
func containsHyphen(s string) bool {
	return repoutil.ContainsHyphen(s)
}

// isNumeric reports whether s consists entirely of ASCII digits.
// Delegates to repoutil.IsNumeric.
func isNumeric(s string) bool {
	return repoutil.IsNumeric(s)
}

// splitSluggedKey splits a slugged key at the first hyphen.
// Returns a two-element slice [prefix, suffix] when a hyphen is present,
// or a one-element slice [key] when there is no hyphen.
// Delegates to repoutil.SplitAtFirstHyphen.
func splitSluggedKey(key string) []string {
	prefix, suffix, ok := repoutil.SplitAtFirstHyphen(key)
	if !ok {
		return []string{key}
	}
	return []string{prefix, suffix}
}
