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

// recentBugRepo is the consumer-side interface for fetching recent bugs.
type recentBugRepo interface {
	GetRecent(ctx context.Context, limit int) ([]*models.Bug, error)
}

// recentChangeCardRepo is the consumer-side interface for fetching recent change-cards.
type recentChangeCardRepo interface {
	GetRecent(ctx context.Context, limit int) ([]*models.ChangeCard, error)
}

// recentIdeaRepo is the consumer-side interface for fetching recent ideas.
type recentIdeaRepo interface {
	GetRecent(ctx context.Context, limit int) ([]*models.Idea, error)
}

// recentTechDebtRepo is the consumer-side interface for fetching recent tech-debt items.
type recentTechDebtRepo interface {
	GetRecent(ctx context.Context, limit int) ([]*models.TechDebt, error)
}

// RecentService fans out to the enabled repository types, merges results in memory,
// sorts by created_at DESC (with deterministic tie-breaking), and applies the final limit.
// This is the only layer that contains merge-and-sort business logic.
//
// Optional repository dependencies (all entity types except task/feature/epic) may be
// nil; the service will simply skip them when fanning out.
type RecentService struct {
	taskRepo     recentTaskRepo
	featureRepo  recentFeatureRepo
	epicRepo     recentEpicRepo
	bugRepo      recentBugRepo
	changeRepo   recentChangeCardRepo
	ideaRepo     recentIdeaRepo
	techDebtRepo recentTechDebtRepo
}

// NewRecentService constructs a RecentService with injected repository dependencies.
// task/feature/epic repos are required; bug/change/idea/tech_debt repos are optional
// and may be nil for callers that don't yet wire them.
func NewRecentService(
	t recentTaskRepo,
	f recentFeatureRepo,
	e recentEpicRepo,
	b recentBugRepo,
	c recentChangeCardRepo,
	i recentIdeaRepo,
	td recentTechDebtRepo,
) *RecentService {
	return &RecentService{
		taskRepo:     t,
		featureRepo:  f,
		epicRepo:     e,
		bugRepo:      b,
		changeRepo:   c,
		ideaRepo:     i,
		techDebtRepo: td,
	}
}

// typeOrder assigns tie-breaking priority when CreatedAt values are equal (lower = earlier).
// Order reflects logical hierarchy: epic → feature → task, then standalone entities.
var typeOrder = map[string]int{
	"epic":      0,
	"feature":   1,
	"task":      2,
	"bug":       3,
	"change":    4,
	"idea":      5,
	"tech_debt": 6,
}

// ListRecent returns the most recently created entities, merged and sorted by created_at DESC.
// When no Include* filter is set, all entity types are included (subject to repo availability).
func (s *RecentService) ListRecent(ctx context.Context, filters RecentFilters) ([]RecentItem, error) {
	// If no type filter specified, include all types.
	includeAll := !filters.IncludeTasks && !filters.IncludeFeatures && !filters.IncludeEpics &&
		!filters.IncludeBugs && !filters.IncludeChanges && !filters.IncludeIdeas && !filters.IncludeTechDebt

	merged := make([]RecentItem, 0, 7*filters.Limit)

	if includeAll || filters.IncludeTasks {
		tasks, err := s.taskRepo.GetRecent(ctx, filters.Limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list recent tasks: %w", err)
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
			return nil, fmt.Errorf("failed to list recent features: %w", err)
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
			return nil, fmt.Errorf("failed to list recent epics: %w", err)
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

	if (includeAll || filters.IncludeBugs) && s.bugRepo != nil {
		bugs, err := s.bugRepo.GetRecent(ctx, filters.Limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list recent bugs: %w", err)
		}
		for _, b := range bugs {
			merged = append(merged, RecentItem{
				Type:      "bug",
				Key:       b.Key,
				Title:     b.Title,
				CreatedAt: b.CreatedAt,
				Status:    string(b.Status),
			})
		}
	}

	if (includeAll || filters.IncludeChanges) && s.changeRepo != nil {
		cards, err := s.changeRepo.GetRecent(ctx, filters.Limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list recent change-cards: %w", err)
		}
		for _, c := range cards {
			merged = append(merged, RecentItem{
				Type:      "change",
				Key:       c.Key,
				Title:     c.Title,
				CreatedAt: c.CreatedAt,
				Status:    string(c.Status),
			})
		}
	}

	if (includeAll || filters.IncludeIdeas) && s.ideaRepo != nil {
		ideas, err := s.ideaRepo.GetRecent(ctx, filters.Limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list recent ideas: %w", err)
		}
		for _, i := range ideas {
			merged = append(merged, RecentItem{
				Type:      "idea",
				Key:       i.Key,
				Title:     i.Title,
				CreatedAt: i.CreatedAt,
				Status:    string(i.Status),
			})
		}
	}

	if (includeAll || filters.IncludeTechDebt) && s.techDebtRepo != nil {
		items, err := s.techDebtRepo.GetRecent(ctx, filters.Limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list recent tech-debts: %w", err)
		}
		for _, td := range items {
			merged = append(merged, RecentItem{
				Type:      "tech_debt",
				Key:       td.Key,
				Title:     td.Title,
				CreatedAt: td.CreatedAt,
				Status:    string(td.Status),
			})
		}
	}

	// created_at DESC; tie-break by type order, then key ASC for test determinism.
	sort.SliceStable(merged, func(i, j int) bool {
		ti := merged[i].CreatedAt
		tj := merged[j].CreatedAt

		if !ti.Equal(tj) {
			return ti.After(tj) // DESC
		}

		// Tie-break 1: type order (epic first, then feature, task, bug, change, idea, tech_debt)
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
