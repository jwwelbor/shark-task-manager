package services

// Compile-time interface satisfaction checks.
// These blank-identifier assignments verify that concrete repository types satisfy
// the narrow interfaces declared by the status package. If a method is missing or
// has the wrong signature the build will fail with a clear error.
//
// They live in this package so that the status package does not need to import
// the repository package (which would create a dependency cycle).

import (
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/status"
)

var (
	_ status.BugDashboardRepository        = (*repository.BugRepository)(nil)
	_ status.ChangeCardDashboardRepository = (*repository.ChangeCardRepository)(nil)
)
