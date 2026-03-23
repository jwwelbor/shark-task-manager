package repository

// Key parsing utilities have been moved to internal/repository/repoutil/key_lookup.go.
// Use repoutil.ContainsHyphen, repoutil.IsNumeric, repoutil.SplitAtFirstHyphen,
// and repoutil.SplitAtNthHyphen directly.
//
// Package-private wrappers (containsHyphen, isNumeric, splitSluggedKey) for test
// compatibility are defined in epic_repository.go and feature_repository.go.
