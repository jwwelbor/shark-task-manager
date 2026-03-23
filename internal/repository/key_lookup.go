package repository

// key_lookup.go provides backward-compatible package-private wrappers around
// repoutil key-parsing utilities (ContainsHyphen, IsNumeric, SplitAtFirstHyphen).

import "github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"

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
