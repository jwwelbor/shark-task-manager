package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// recentTaskRepo is the consumer-side interface for fetching recent tasks.
// Defined here (consumer side) per .claude/rules/go/patterns.md "Interface Design".
type recentTaskRepo interface {
	GetRecent(ctx context.Context, limit int) ([]*models.Task, error)
}

// recentFeatureRepo is the consumer-side interface for fetching recent features.
type recentFeatureRepo interface {
	GetRecent(ctx context.Context, limit int) ([]*models.Feature, error)
}

// recentEpicRepo is the consumer-side interface for fetching recent epics.
type recentEpicRepo interface {
	GetRecent(ctx context.Context, limit int) ([]*models.Epic, error)
}

// RecentService fans out to the enabled repository types, merges results in memory,
// sorts by created_at DESC (with deterministic tie-breaking), and applies the final limit.
// This is the only layer that contains merge-and-sort business logic.
type RecentService struct {
	taskRepo    recentTaskRepo
	featureRepo recentFeatureRepo
	epicRepo    recentEpicRepo
}

// NewRecentService constructs a RecentService with injected repository dependencies.
func NewRecentService(t recentTaskRepo, f recentFeatureRepo, e recentEpicRepo) *RecentService {
	return &RecentService{
		taskRepo:    t,
		featureRepo: f,
		epicRepo:    e,
	}
}

// typeOrder assigns tie-breaking priority when CreatedAt values are equal (lower = earlier).
var typeOrder = map[string]int{
	"epic":    0,
	"feature": 1,
	"task":    2,
}

// ListRecent returns the most recently created entities, merged and sorted by created_at DESC.
// When no Include* filter is set, all three entity types are included.
func (s *RecentService) ListRecent(ctx context.Context, filters RecentFilters) ([]RecentItem, error) {
	// If no type filter specified, include all three types.
	includeAll := !filters.IncludeTasks && !filters.IncludeFeatures && !filters.IncludeEpics

	merged := make([]RecentItem, 0, 3*filters.Limit)

	if includeAll || filters.IncludeTasks {
		tasks, err := s.taskRepo.GetRecent(ctx, filters.Limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list recent task: %w", err)
		}
		for _, t := range tasks {
			merged = append(merged, RecentItem{
				Type:      "task",
				Key:       t.Key,
				Title:     t.Title,
				CreatedAt: t.CreatedAt,
				Status:    string(t.Status),
			})
		}
	}

	if includeAll || filters.IncludeFeatures {
		features, err := s.featureRepo.GetRecent(ctx, filters.Limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list recent feature: %w", err)
		}
		for _, f := range features {
			merged = append(merged, RecentItem{
				Type:      "feature",
				Key:       f.Key,
				Title:     f.Title,
				CreatedAt: f.CreatedAt,
				Status:    string(f.Status),
			})
		}
	}

	if includeAll || filters.IncludeEpics {
		epics, err := s.epicRepo.GetRecent(ctx, filters.Limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list recent epic: %w", err)
		}
		for _, e := range epics {
			merged = append(merged, RecentItem{
				Type:      "epic",
				Key:       e.Key,
				Title:     e.Title,
				CreatedAt: e.CreatedAt,
				Status:    string(e.Status),
			})
		}
	}

	// created_at DESC; tie-break by type (epic→feature→task), then key ASC for test determinism.
	sort.SliceStable(merged, func(i, j int) bool {
		ti := merged[i].CreatedAt
		tj := merged[j].CreatedAt

		if !ti.Equal(tj) {
			return ti.After(tj) // DESC
		}

		// Tie-break 1: type order (epic first)
		oi := typeOrder[merged[i].Type]
		oj := typeOrder[merged[j].Type]
		if oi != oj {
			return oi < oj
		}

		// Tie-break 2: key ascending
		return merged[i].Key < merged[j].Key
	})

	if filters.Limit > 0 && len(merged) > filters.Limit {
		merged = merged[:filters.Limit]
	}

	return merged, nil
}
