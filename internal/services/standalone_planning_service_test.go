package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

type standaloneNextBugStub struct {
	items         []*models.Bug
	err           error
	inspectFilter func(BugFilters)
}

func (s standaloneNextBugStub) ListBugs(_ context.Context, filters BugFilters) ([]*models.Bug, error) {
	if s.inspectFilter != nil {
		s.inspectFilter(filters)
	}
	return s.items, s.err
}

type standaloneNextChangeStub struct {
	items         []*models.ChangeCard
	err           error
	inspectFilter func(ChangeCardFilters)
}

func (s standaloneNextChangeStub) ListChangeCards(
	_ context.Context,
	filters ChangeCardFilters,
) ([]*models.ChangeCard, error) {
	if s.inspectFilter != nil {
		s.inspectFilter(filters)
	}
	return s.items, s.err
}

type standaloneNextTechDebtStub struct {
	items         []*models.TechDebt
	err           error
	inspectFilter func(TechDebtFilters)
}

func (s standaloneNextTechDebtStub) ListTechDebts(
	_ context.Context,
	filters TechDebtFilters,
) ([]*models.TechDebt, error) {
	if s.inspectFilter != nil {
		s.inspectFilter(filters)
	}
	return s.items, s.err
}

type standaloneNextClaimStub struct {
	err error
}

func (s standaloneNextClaimStub) IsClaimable(context.Context, string, string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return true, nil
}

type keyedStandalonePlanClaimStub struct {
	claimed map[string]bool
}

func (s keyedStandalonePlanClaimStub) IsClaimable(_ context.Context, _, key string) (bool, error) {
	return !s.claimed[key], nil
}

type standaloneNextDependencyStub struct {
	blocked map[string]bool
	err     error
}

func (s standaloneNextDependencyStub) HasUnresolvedHardDependency(
	_ context.Context,
	_ models.EntityType,
	entityID int64,
) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.blocked[fmt.Sprint(entityID)], nil
}

func TestStandalonePlanningServiceGroupsBugsBySeverityAndClaims(t *testing.T) {
	start := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	service := NewStandalonePlanningService(
		standaloneNextBugStub{items: []*models.Bug{
			{BaseEntity: models.BaseEntity{Key: "B003", CreatedAt: start.Add(2 * time.Hour)}, Severity: models.BugSeverityHigh},
			{BaseEntity: models.BaseEntity{Key: "B001", CreatedAt: start}, Severity: models.BugSeverityCritical},
			{BaseEntity: models.BaseEntity{Key: "B002", CreatedAt: start.Add(time.Hour)}, Severity: models.BugSeverityCritical},
		}},
		standaloneNextChangeStub{},
		standaloneNextTechDebtStub{},
		keyedStandalonePlanClaimStub{claimed: map[string]bool{"B002": true}},
		standaloneNextDependencyStub{},
	)

	plan, err := service.Plan(context.Background(), StandalonePlanBugs)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	want := [][]StandalonePlanCandidate{
		{{Key: "B001", EntityType: models.EntityTypeBug}},
		{{Key: "B003", EntityType: models.EntityTypeBug}},
	}
	if !reflect.DeepEqual(plan.Layers, want) {
		t.Fatalf("Layers = %#v, want %#v", plan.Layers, want)
	}
}

func TestStandalonePlanningServiceGroupsChangeCardsByAscendingPriority(t *testing.T) {
	start := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	service := NewStandalonePlanningService(
		standaloneNextBugStub{},
		standaloneNextChangeStub{items: []*models.ChangeCard{
			{BaseEntity: models.BaseEntity{Key: "CC-003", CreatedAt: start.Add(2 * time.Hour)}, Priority: 5},
			{BaseEntity: models.BaseEntity{Key: "CC-002", CreatedAt: start.Add(time.Hour)}, Priority: 2},
			{BaseEntity: models.BaseEntity{Key: "CC-001", CreatedAt: start}, Priority: 2},
		}},
		standaloneNextTechDebtStub{},
		standaloneNextClaimStub{},
		standaloneNextDependencyStub{},
	)

	plan, err := service.Plan(context.Background(), StandalonePlanChangeCards)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	want := [][]StandalonePlanCandidate{
		{
			{Key: "CC-001", EntityType: models.EntityTypeChange},
			{Key: "CC-002", EntityType: models.EntityTypeChange},
		},
		{{Key: "CC-003", EntityType: models.EntityTypeChange}},
	}
	if !reflect.DeepEqual(plan.Layers, want) {
		t.Fatalf("Layers = %#v, want %#v", plan.Layers, want)
	}
}

func TestStandalonePlanningServiceGroupsTechDebtBySeverity(t *testing.T) {
	service := NewStandalonePlanningService(
		standaloneNextBugStub{},
		standaloneNextChangeStub{},
		standaloneNextTechDebtStub{items: []*models.TechDebt{
			{BaseEntity: models.BaseEntity{Key: "TD-002"}, Severity: models.TechDebtSeverityLow},
			{BaseEntity: models.BaseEntity{Key: "TD-001"}, Severity: models.TechDebtSeverityHigh},
		}},
		standaloneNextClaimStub{},
		standaloneNextDependencyStub{},
	)

	plan, err := service.Plan(context.Background(), StandalonePlanTechDebt)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	want := [][]StandalonePlanCandidate{
		{{Key: "TD-001", EntityType: models.EntityTypeTechDebt}},
		{{Key: "TD-002", EntityType: models.EntityTypeTechDebt}},
	}
	if !reflect.DeepEqual(plan.Layers, want) {
		t.Fatalf("Layers = %#v, want %#v", plan.Layers, want)
	}
}

func TestStandalonePlanningServiceRanksEverySeverityAndUnknownForBugsAndTechDebt(t *testing.T) {
	wantKeys := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "UNKNOWN"}
	tests := []struct {
		name       string
		collection StandalonePlanCollection
		bugs       []*models.Bug
		techDebt   []*models.TechDebt
	}{
		{
			name:       "bugs",
			collection: StandalonePlanBugs,
			bugs: []*models.Bug{
				{BaseEntity: models.BaseEntity{Key: "LOW"}, Severity: models.BugSeverityLow},
				{BaseEntity: models.BaseEntity{Key: "UNKNOWN"}, Severity: models.BugSeverity("unknown")},
				{BaseEntity: models.BaseEntity{Key: "HIGH"}, Severity: models.BugSeverityHigh},
				{BaseEntity: models.BaseEntity{Key: "CRITICAL"}, Severity: models.BugSeverityCritical},
				{BaseEntity: models.BaseEntity{Key: "MEDIUM"}, Severity: models.BugSeverityMedium},
			},
		},
		{
			name:       "tech debt",
			collection: StandalonePlanTechDebt,
			techDebt: []*models.TechDebt{
				{BaseEntity: models.BaseEntity{Key: "LOW"}, Severity: models.TechDebtSeverityLow},
				{BaseEntity: models.BaseEntity{Key: "UNKNOWN"}, Severity: models.TechDebtSeverity("unknown")},
				{BaseEntity: models.BaseEntity{Key: "HIGH"}, Severity: models.TechDebtSeverityHigh},
				{BaseEntity: models.BaseEntity{Key: "CRITICAL"}, Severity: models.TechDebtSeverityCritical},
				{BaseEntity: models.BaseEntity{Key: "MEDIUM"}, Severity: models.TechDebtSeverityMedium},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewStandalonePlanningService(
				standaloneNextBugStub{items: tt.bugs},
				standaloneNextChangeStub{},
				standaloneNextTechDebtStub{items: tt.techDebt},
				standaloneNextClaimStub{},
				standaloneNextDependencyStub{},
			)

			plan, err := service.Plan(context.Background(), tt.collection)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			if len(plan.Layers) != len(wantKeys) {
				t.Fatalf("len(Layers) = %d, want %d", len(plan.Layers), len(wantKeys))
			}
			for i, wantKey := range wantKeys {
				if got := plan.Layers[i][0].Key; got != wantKey {
					t.Errorf("Layers[%d][0].Key = %q, want %q", i, got, wantKey)
				}
			}
		})
	}
}

func TestStandalonePlanningServiceRequestsNonTerminalProviderLists(t *testing.T) {
	t.Run("bugs", func(t *testing.T) {
		called := false
		service := NewStandalonePlanningService(
			standaloneNextBugStub{inspectFilter: func(filters BugFilters) {
				called = true
				if filters.ShowAll {
					t.Fatal("bug ShowAll = true, want terminal entities excluded by provider")
				}
			}},
			standaloneNextChangeStub{},
			standaloneNextTechDebtStub{},
			standaloneNextClaimStub{},
			standaloneNextDependencyStub{},
		)

		if _, err := service.Plan(context.Background(), StandalonePlanBugs); err != nil {
			t.Fatalf("Plan(bugs) error = %v", err)
		}
		if !called {
			t.Fatal("bug provider was not called")
		}
	})

	t.Run("change cards", func(t *testing.T) {
		called := false
		service := NewStandalonePlanningService(
			standaloneNextBugStub{},
			standaloneNextChangeStub{inspectFilter: func(filters ChangeCardFilters) {
				called = true
				if filters.ShowAll {
					t.Fatal("change-card ShowAll = true, want terminal entities excluded by provider")
				}
			}},
			standaloneNextTechDebtStub{},
			standaloneNextClaimStub{},
			standaloneNextDependencyStub{},
		)

		if _, err := service.Plan(context.Background(), StandalonePlanChangeCards); err != nil {
			t.Fatalf("Plan(change-cards) error = %v", err)
		}
		if !called {
			t.Fatal("change-card provider was not called")
		}
	})

	t.Run("tech debt", func(t *testing.T) {
		called := false
		service := NewStandalonePlanningService(
			standaloneNextBugStub{},
			standaloneNextChangeStub{},
			standaloneNextTechDebtStub{inspectFilter: func(filters TechDebtFilters) {
				called = true
				if filters.ShowAll {
					t.Fatal("tech-debt ShowAll = true, want terminal entities excluded by provider")
				}
			}},
			standaloneNextClaimStub{},
			standaloneNextDependencyStub{},
		)

		if _, err := service.Plan(context.Background(), StandalonePlanTechDebt); err != nil {
			t.Fatalf("Plan(tech-debt) error = %v", err)
		}
		if !called {
			t.Fatal("tech-debt provider was not called")
		}
	})
}

func TestStandalonePlanningServiceFailsWhenClaimStateIsUnavailable(t *testing.T) {
	sentinel := errors.New("claim store unavailable")
	service := NewStandalonePlanningService(
		standaloneNextBugStub{items: []*models.Bug{{
			BaseEntity: models.BaseEntity{Key: "B001"},
			Severity:   models.BugSeverityCritical,
		}}},
		standaloneNextChangeStub{},
		standaloneNextTechDebtStub{},
		standaloneNextClaimStub{err: sentinel},
		standaloneNextDependencyStub{},
	)

	_, err := service.Plan(context.Background(), StandalonePlanBugs)
	if !errors.Is(err, sentinel) {
		t.Fatalf("Plan() error = %v, want wrapped claim error", err)
	}
}

func TestStandalonePlanningServiceExcludesUnresolvedHardDependencies(t *testing.T) {
	service := NewStandalonePlanningService(
		standaloneNextBugStub{items: []*models.Bug{
			{BaseEntity: models.BaseEntity{ID: 1, Key: "B001"}, Severity: models.BugSeverityCritical},
			{BaseEntity: models.BaseEntity{ID: 2, Key: "B002"}, Severity: models.BugSeverityCritical},
		}},
		standaloneNextChangeStub{},
		standaloneNextTechDebtStub{},
		standaloneNextClaimStub{},
		standaloneNextDependencyStub{blocked: map[string]bool{"1": true}},
	)

	plan, err := service.Plan(context.Background(), StandalonePlanBugs)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	want := [][]StandalonePlanCandidate{{{Key: "B002", EntityType: models.EntityTypeBug}}}
	if !reflect.DeepEqual(plan.Layers, want) {
		t.Fatalf("Layers = %#v, want unresolved dependency excluded", plan.Layers)
	}
}

type standaloneDependencyEntityRepo struct {
	entity models.Entity
}

func (r standaloneDependencyEntityRepo) GetByKey(context.Context, string) (models.Entity, error) {
	return r.entity, nil
}
func (r standaloneDependencyEntityRepo) GetByID(context.Context, int64) (models.Entity, error) {
	return r.entity, nil
}
func (r standaloneDependencyEntityRepo) UpdateStatus(context.Context, int64, string) error {
	return nil
}
func (r standaloneDependencyEntityRepo) UpdateStatusIfCurrent(context.Context, int64, string, string) (bool, error) {
	return true, nil
}
func (r standaloneDependencyEntityRepo) Update(context.Context, models.Entity) error {
	return nil
}
func (r standaloneDependencyEntityRepo) GetContextData(context.Context, int64) (*string, error) {
	return nil, nil
}
func (r standaloneDependencyEntityRepo) UpdateContextData(context.Context, int64, *string) error {
	return nil
}

func TestStandaloneHardDependencyServiceEvaluatesDependencyStatus(t *testing.T) {
	tests := []struct {
		name        string
		depStatus   models.TaskStatus
		wantBlocked bool
	}{
		{name: "non-terminal dependency blocks", depStatus: "development", wantBlocked: true},
		{name: "terminal dependency is satisfied", depStatus: "completed", wantBlocked: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relationships := &MockEntityRelationshipRepository{
				GetOutgoingFunc: func(context.Context, models.EntityType, int64, []models.EntityRelationshipType) ([]*models.EntityRelationship, error) {
					return []*models.EntityRelationship{{
						FromEntityType:   models.EntityTypeBug,
						FromEntityID:     1,
						ToEntityType:     models.EntityTypeTask,
						ToEntityID:       2,
						RelationshipType: models.EntityRelDependsOn,
					}}, nil
				},
			}
			registry := NewEntityRegistry()
			registry.Register(models.EntityTypeTask, standaloneDependencyEntityRepo{entity: &models.Task{
				BaseEntity: models.BaseEntity{ID: 2, Key: "T-E01-F01-001"},
				Status:     tt.depStatus,
			}})
			service := NewStandaloneHardDependencyService(relationships, registry, workflow.NewService("."))

			blocked, err := service.HasUnresolvedHardDependency(
				context.Background(), models.EntityTypeBug, 1,
			)
			if err != nil {
				t.Fatalf("HasUnresolvedHardDependency() error = %v", err)
			}
			if blocked != tt.wantBlocked {
				t.Fatalf("blocked = %t, want %t", blocked, tt.wantBlocked)
			}
		})
	}
}
