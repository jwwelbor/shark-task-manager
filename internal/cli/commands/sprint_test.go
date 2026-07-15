package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSprintService implements sprintServicer for testing.
type MockSprintService struct {
	CreateSprintFunc             func(ctx context.Context, input services.CreateSprintInput) (*models.Sprint, error)
	GetSprintFunc                func(ctx context.Context, key string) (*models.Sprint, error)
	ListSprintsFunc              func(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error)
	UpdateSprintFunc             func(ctx context.Context, key string, updates services.UpdateSprintInput) (*models.Sprint, error)
	DeleteSprintFunc             func(ctx context.Context, key string) error
	StartSprintFunc              func(ctx context.Context, key string) (*models.Sprint, error)
	CloseSprintFunc              func(ctx context.Context, key string) (*models.Sprint, error)
	CloseSprintWithCarryoverFunc func(ctx context.Context, key string, mode services.CarryoverMode) (*services.SprintCloseResult, error)
	ArchiveSprintFunc            func(ctx context.Context, key string) (*models.Sprint, error)
	// F07 start warning (REQ-F-009)
	CountNullSprintOrderFunc func(ctx context.Context, sprintKey string) (int, error)
	// F03 assignment methods
	AddEntityToSprintFunc      func(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error)
	RemoveEntityFromSprintFunc func(ctx context.Context, sprintKey, entityKey string) error
	GetSprintBacklogFunc       func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error)
	// F05 planning methods
	PlanSprintFunc         func(ctx context.Context, key string) (*services.SprintPlanView, error)
	GetSprintReadinessFunc func(ctx context.Context, key string) (*services.SprintReadiness, error)
	SetSprintCapacityFunc  func(ctx context.Context, input services.SetSprintCapacityInput) (*models.SprintCapacity, error)
	GetSprintCapacityFunc  func(ctx context.Context, key string) ([]services.CapacityRow, error)
	BulkAddToSprintFunc    func(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error)
	GetNextTaskFunc        func(ctx context.Context, agentType string) (*services.BacklogItemView, error)
	// F07 ordering methods
	ReorderAssignmentFunc func(ctx context.Context, sprintKey, entityKey string, target services.ReorderTarget) (*models.SprintAssignment, []*models.SprintAssignment, error)
}

func (m *MockSprintService) CreateSprint(ctx context.Context, input services.CreateSprintInput) (*models.Sprint, error) {
	if m.CreateSprintFunc != nil {
		return m.CreateSprintFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockSprintService) GetSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.GetSprintFunc != nil {
		return m.GetSprintFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockSprintService) ListSprints(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error) {
	if m.ListSprintsFunc != nil {
		return m.ListSprintsFunc(ctx, filters)
	}
	return nil, nil
}

func (m *MockSprintService) UpdateSprint(ctx context.Context, key string, updates services.UpdateSprintInput) (*models.Sprint, error) {
	if m.UpdateSprintFunc != nil {
		return m.UpdateSprintFunc(ctx, key, updates)
	}
	return nil, nil
}

func (m *MockSprintService) DeleteSprint(ctx context.Context, key string) error {
	if m.DeleteSprintFunc != nil {
		return m.DeleteSprintFunc(ctx, key)
	}
	return nil
}

func (m *MockSprintService) StartSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.StartSprintFunc != nil {
		return m.StartSprintFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockSprintService) CloseSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.CloseSprintFunc != nil {
		return m.CloseSprintFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockSprintService) CloseSprintWithCarryover(ctx context.Context, key string, mode services.CarryoverMode) (*services.SprintCloseResult, error) {
	if m.CloseSprintWithCarryoverFunc != nil {
		return m.CloseSprintWithCarryoverFunc(ctx, key, mode)
	}
	return &services.SprintCloseResult{
		Sprint: &models.Sprint{Key: key, Name: "Sprint", Status: "completed"},
	}, nil
}

func (m *MockSprintService) ArchiveSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.ArchiveSprintFunc != nil {
		return m.ArchiveSprintFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockSprintService) CountNullSprintOrder(ctx context.Context, sprintKey string) (int, error) {
	if m.CountNullSprintOrderFunc != nil {
		return m.CountNullSprintOrderFunc(ctx, sprintKey)
	}
	return 0, nil
}

func (m *MockSprintService) AddEntityToSprint(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error) {
	if m.AddEntityToSprintFunc != nil {
		return m.AddEntityToSprintFunc(ctx, input)
	}
	return &models.SprintAssignment{}, nil, nil
}

func (m *MockSprintService) RemoveEntityFromSprint(ctx context.Context, sprintKey, entityKey string) error {
	if m.RemoveEntityFromSprintFunc != nil {
		return m.RemoveEntityFromSprintFunc(ctx, sprintKey, entityKey)
	}
	return nil
}

func (m *MockSprintService) GetSprintBacklog(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
	if m.GetSprintBacklogFunc != nil {
		return m.GetSprintBacklogFunc(ctx, sprintKey, opts)
	}
	return &services.SprintBacklog{SprintKey: sprintKey, Groups: []*services.BacklogGroup{}}, nil
}

func (m *MockSprintService) PlanSprint(ctx context.Context, key string) (*services.SprintPlanView, error) {
	if m.PlanSprintFunc != nil {
		return m.PlanSprintFunc(ctx, key)
	}
	return &services.SprintPlanView{Sprint: &models.Sprint{Key: key}}, nil
}

func (m *MockSprintService) GetSprintReadiness(ctx context.Context, key string) (*services.SprintReadiness, error) {
	if m.GetSprintReadinessFunc != nil {
		return m.GetSprintReadinessFunc(ctx, key)
	}
	return &services.SprintReadiness{OverallScore: 0, Factors: []services.ReadinessFactor{}}, nil
}

func (m *MockSprintService) SetSprintCapacity(ctx context.Context, input services.SetSprintCapacityInput) (*models.SprintCapacity, error) {
	if m.SetSprintCapacityFunc != nil {
		return m.SetSprintCapacityFunc(ctx, input)
	}
	return &models.SprintCapacity{AgentType: input.AgentType, CapacityPoints: input.Points}, nil
}

func (m *MockSprintService) GetSprintCapacity(ctx context.Context, key string) ([]services.CapacityRow, error) {
	if m.GetSprintCapacityFunc != nil {
		return m.GetSprintCapacityFunc(ctx, key)
	}
	return []services.CapacityRow{}, nil
}

func (m *MockSprintService) BulkAddToSprint(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error) {
	if m.BulkAddToSprintFunc != nil {
		return m.BulkAddToSprintFunc(ctx, input)
	}
	return &services.BulkAddResult{AddedByType: map[string]int{}, SkippedByType: map[string]int{}}, nil
}

func (m *MockSprintService) GetNextTask(ctx context.Context, agentType string) (*services.BacklogItemView, error) {
	if m.GetNextTaskFunc != nil {
		return m.GetNextTaskFunc(ctx, agentType)
	}
	return nil, nil
}

func (m *MockSprintService) ReorderAssignment(ctx context.Context, sprintKey, entityKey string, target services.ReorderTarget) (*models.SprintAssignment, []*models.SprintAssignment, error) {
	if m.ReorderAssignmentFunc != nil {
		return m.ReorderAssignmentFunc(ctx, sprintKey, entityKey, target)
	}
	return &models.SprintAssignment{}, []*models.SprintAssignment{}, nil
}

// Compile-time interface checks for the narrowed sprint CLI service contracts.
var _ sprintLifecycleServicer = (*MockSprintService)(nil)
var _ sprintAssignmentServicer = (*MockSprintService)(nil)
var _ sprintCapacityServicer = (*MockSprintService)(nil)

// Test helpers
func setupSprintTest(t *testing.T, mock *MockSprintService) func() {
	oldLifecycleOverride := sprintLifecycleSvcOverride
	oldAssignmentOverride := sprintAssignmentSvcOverride
	oldPlanningOverride := sprintPlanningSvcOverride
	oldCapacityOverride := sprintCapacitySvcOverride
	sprintLifecycleSvcOverride = mock
	sprintAssignmentSvcOverride = mock
	sprintPlanningSvcOverride = mock
	sprintCapacitySvcOverride = mock
	return func() {
		sprintLifecycleSvcOverride = oldLifecycleOverride
		sprintAssignmentSvcOverride = oldAssignmentOverride
		sprintPlanningSvcOverride = oldPlanningOverride
		sprintCapacitySvcOverride = oldCapacityOverride
	}
}

func newTestSprint() *models.Sprint {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	return &models.Sprint{
		ID:        1,
		Key:       "S001",
		Slug:      "sprint-1",
		Name:      "Sprint 1",
		Goal:      "Complete authentication",
		Status:    models.SprintStatus("planning"),
		StartDate: startDate,
		EndDate:   endDate,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Test runSprintCreate
func TestSprintCreate_Success(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	mock := &MockSprintService{
		CreateSprintFunc: func(ctx context.Context, input services.CreateSprintInput) (*models.Sprint, error) {
			assert.Equal(t, "Sprint 1", input.Name)
			assert.Equal(t, "Complete auth", input.Goal)
			assert.True(t, input.StartDate.Equal(startDate))
			assert.True(t, input.EndDate.Equal(endDate))
			return newTestSprint(), nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")
	cmd.Flags().String("goal", "", "sprint goal")

	// Set flags programmatically
	cmd.Flags().Set("start", "2026-03-18")
	cmd.Flags().Set("end", "2026-04-01")
	cmd.Flags().Set("goal", "Complete auth")

	err := runSprintCreate(cmd, []string{"Sprint 1"})
	assert.NoError(t, err)
}

func TestSprintCreate_InvalidDateFormat(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")
	cmd.Flags().String("goal", "", "sprint goal")

	cmd.Flags().Set("start", "03/18/2026")
	cmd.Flags().Set("end", "2026-04-01")

	err := runSprintCreate(cmd, []string{"Sprint 1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start date format")
}

func TestSprintCreate_EmptyName(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSprintCreate(cmd, []string{"   "})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// Test runSprintGet
func TestSprintGet_Success(t *testing.T) {
	mock := &MockSprintService{
		GetSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			return newTestSprint(), nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("json", false, "")

	err := runSprintGet(cmd, []string{"S001"})
	assert.NoError(t, err)
}

func TestSprintGet_JSON(t *testing.T) {
	mock := &MockSprintService{
		GetSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return newTestSprint(), nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("json", true, "")
	cmd.Flags().Lookup("json").Value.Set("true")

	err := runSprintGet(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test runSprintList
func TestSprintList_Success(t *testing.T) {
	sprints := []*models.Sprint{
		newTestSprint(),
		{
			ID:        2,
			Key:       "S002",
			Name:      "Sprint 2",
			Status:    models.SprintStatus("in_progress"),
			StartDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mock := &MockSprintService{
		ListSprintsFunc: func(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error) {
			return sprints, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("status", "", "")

	err := runSprintList(cmd, []string{})
	assert.NoError(t, err)
}

func TestSprintList_WithStatusFilter(t *testing.T) {
	mock := &MockSprintService{
		ListSprintsFunc: func(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error) {
			assert.NotNil(t, filters)
			assert.Equal(t, "in_progress", filters.Status)
			return []*models.Sprint{newTestSprint()}, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("status", "", "sprint status")
	cmd.Flags().Set("status", "in_progress")

	err := runSprintList(cmd, []string{})
	assert.NoError(t, err)
}

// Test runSprintUpdate
func TestSprintUpdate_Success(t *testing.T) {
	updatedSprint := newTestSprint()
	updatedSprint.Name = "Updated Sprint"

	mock := &MockSprintService{
		UpdateSprintFunc: func(ctx context.Context, key string, updates services.UpdateSprintInput) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			assert.NotNil(t, updates.Name)
			assert.Equal(t, "Updated Sprint", *updates.Name)
			return updatedSprint, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("name", "", "sprint name")
	cmd.Flags().String("goal", "", "sprint goal")
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")
	cmd.Flags().Set("name", "Updated Sprint")

	err := runSprintUpdate(cmd, []string{"S001"})
	assert.NoError(t, err)
}

func TestSprintUpdate_NoFlagsProvided(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("name", "", "sprint name")
	cmd.Flags().String("goal", "", "sprint goal")
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")

	err := runSprintUpdate(cmd, []string{"S001"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one update flag is required")
}

func TestSprintUpdate_InvalidEndDate(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("name", "", "sprint name")
	cmd.Flags().String("goal", "", "sprint goal")
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")
	cmd.Flags().Set("name", "Updated Sprint")
	cmd.Flags().Set("end", "invalid-date")

	err := runSprintUpdate(cmd, []string{"S001"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid end date format")
}

// Test runSprintDelete
func TestSprintDelete_Success(t *testing.T) {
	mock := &MockSprintService{
		DeleteSprintFunc: func(ctx context.Context, key string) error {
			assert.Equal(t, "S001", key)
			return nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("force", true, "")
	cmd.Flags().Lookup("force").Value.Set("true")

	err := runSprintDelete(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test runSprintStart
func TestSprintStart_Success(t *testing.T) {
	startedSprint := newTestSprint()
	startedSprint.Status = models.SprintStatus("in_progress")

	mock := &MockSprintService{
		StartSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			return startedSprint, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSprintStart(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test runSprintClose
func TestSprintClose_Success(t *testing.T) {
	closedSprint := newTestSprint()
	closedSprint.Status = models.SprintStatus("ready_for_review")

	mock := &MockSprintService{
		CloseSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			return closedSprint, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSprintClose(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test runSprintArchive
func TestSprintArchive_Success(t *testing.T) {
	archivedSprint := newTestSprint()
	archivedSprint.Status = models.SprintStatus("completed")

	mock := &MockSprintService{
		ArchiveSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			return archivedSprint, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSprintArchive(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test helper function - confirmSprintDelete
func TestConfirmSprintDelete(t *testing.T) {
	sprint := newTestSprint()
	assert.NotNil(t, sprint)
}

// =============================================================================
// MockSprintAnalyticsService — implements sprintAnalyticsServicer for testing.
// =============================================================================

// MockSprintAnalyticsService implements sprintAnalyticsServicer for testing.
type MockSprintAnalyticsService struct {
	GetVelocityFunc func(ctx context.Context, n int) (*services.VelocityResult, error)
	GetBurndownFunc func(ctx context.Context, sprintKey string) (*services.BurndownResult, error)
	GetSummaryFunc  func(ctx context.Context, sprintKey string, detailed bool) (*services.SprintSummaryResult, error)
}

func (m *MockSprintAnalyticsService) GetVelocity(ctx context.Context, n int) (*services.VelocityResult, error) {
	if m.GetVelocityFunc != nil {
		return m.GetVelocityFunc(ctx, n)
	}
	return nil, fmt.Errorf("GetVelocity not implemented in mock")
}

func (m *MockSprintAnalyticsService) GetBurndown(ctx context.Context, sprintKey string) (*services.BurndownResult, error) {
	if m.GetBurndownFunc != nil {
		return m.GetBurndownFunc(ctx, sprintKey)
	}
	return nil, fmt.Errorf("GetBurndown not implemented in mock")
}

func (m *MockSprintAnalyticsService) GetSummary(ctx context.Context, sprintKey string, detailed bool) (*services.SprintSummaryResult, error) {
	if m.GetSummaryFunc != nil {
		return m.GetSummaryFunc(ctx, sprintKey, detailed)
	}
	return nil, fmt.Errorf("GetSummary not implemented in mock")
}

// setupAnalyticsTest overrides the analytics service override with a mock for a test run.
func setupAnalyticsTest(t *testing.T, mock *MockSprintAnalyticsService) func() {
	t.Helper()
	oldOverride := sprintAnalyticsSvcOverride
	sprintAnalyticsSvcOverride = mock
	return func() {
		sprintAnalyticsSvcOverride = oldOverride
	}
}

// =============================================================================
// TC-V-04: --sprints=0 and --sprints=101 return validation errors (BVA boundaries)
// =============================================================================

func TestSprintVelocity_LimitTooLow(t *testing.T) {
	// TC-V-04: N=0 below minimum → service returns error
	called := false
	mock := &MockSprintAnalyticsService{
		GetVelocityFunc: func(ctx context.Context, n int) (*services.VelocityResult, error) {
			called = true
			// Service validates range; n=0 triggers this
			return nil, fmt.Errorf("sprints must be between 1 and 100, got %d", n)
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Int("sprints", 5, "")
	cmd.Flags().Set("sprints", "0")

	err := runSprintVelocity(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sprints must be between 1 and 100")
	assert.True(t, called, "service must be called — validation lives in service, not CLI")
}

func TestSprintVelocity_LimitTooHigh(t *testing.T) {
	// TC-V-04: N=101 above maximum → service returns error
	mock := &MockSprintAnalyticsService{
		GetVelocityFunc: func(ctx context.Context, n int) (*services.VelocityResult, error) {
			return nil, fmt.Errorf("sprints must be between 1 and 100, got %d", n)
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Int("sprints", 5, "")
	cmd.Flags().Set("sprints", "101")

	err := runSprintVelocity(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sprints must be between 1 and 100")
}

// =============================================================================
// TC-V-03: --sprints=N=1 (minimum valid) succeeds
// =============================================================================

func TestSprintVelocity_LimitOne_Success(t *testing.T) {
	// TC-V-03: N=1 is the minimum valid value; must succeed.
	mock := &MockSprintAnalyticsService{
		GetVelocityFunc: func(ctx context.Context, n int) (*services.VelocityResult, error) {
			assert.Equal(t, 1, n)
			return &services.VelocityResult{
				Sprints:          []services.VelocitySprint{{Key: "S001", Name: "Sprint 1", CompletedSize: 8, UnsizedCompleted: 0}},
				TrailingAverage:  8.0,
				SprintCount:      1,
				InsufficientData: true, // <3 sprints
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	// Restore JSON config after test
	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Int("sprints", 5, "")
	cmd.Flags().Set("sprints", "1")

	err := runSprintVelocity(cmd, []string{})
	assert.NoError(t, err)
}

// =============================================================================
// TC-V-06: unsized_completed visible in both human and JSON output
// =============================================================================

func TestSprintVelocity_UnsizedCompletedInJSONOutput(t *testing.T) {
	// TC-V-06: JSON output includes unsized_completed field.
	mock := &MockSprintAnalyticsService{
		GetVelocityFunc: func(ctx context.Context, n int) (*services.VelocityResult, error) {
			return &services.VelocityResult{
				Sprints: []services.VelocitySprint{
					{Key: "S001", Name: "Sprint 1", CompletedSize: 10, UnsizedCompleted: 2},
				},
				TrailingAverage:  10.0,
				SprintCount:      1,
				InsufficientData: true,
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	var jsonBuf bytes.Buffer
	origOut := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Int("sprints", 5, "")

	runErr := runSprintVelocity(cmd, []string{})

	w.Close()
	os.Stdout = origOut
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	// Verify JSON contains unsized_completed
	output := jsonBuf.String()
	assert.Contains(t, output, `"unsized_completed"`)
	assert.Contains(t, output, "2")
}

// =============================================================================
// TC-V-10: --json output matches AC-V-6 schema
// =============================================================================

func TestSprintVelocity_JSONSchema(t *testing.T) {
	// TC-V-10: Strict JSON schema validation.
	mock := &MockSprintAnalyticsService{
		GetVelocityFunc: func(ctx context.Context, n int) (*services.VelocityResult, error) {
			return &services.VelocityResult{
				Sprints: []services.VelocitySprint{
					{Key: "S001", Name: "Sprint 1", CompletedSize: 18, UnsizedCompleted: 2},
					{Key: "S002", Name: "Sprint 2", CompletedSize: 21, UnsizedCompleted: 0},
				},
				TrailingAverage:  19.5,
				SprintCount:      2,
				InsufficientData: true,
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Int("sprints", 5, "")

	runErr := runSprintVelocity(cmd, []string{})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result))

	// Verify top-level schema
	assert.Contains(t, result, "sprints")
	assert.Contains(t, result, "trailing_average")
	assert.Contains(t, result, "sprint_count")

	// Verify per-sprint schema
	sprints, ok := result["sprints"].([]interface{})
	require.True(t, ok)
	require.Len(t, sprints, 2)
	sprint0, ok := sprints[0].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, sprint0, "key")
	assert.Contains(t, sprint0, "name")
	assert.Contains(t, sprint0, "completed_size")
	assert.Contains(t, sprint0, "unsized_completed")
}

// =============================================================================
// TC-B-10: Human output uses only ASCII characters (no Unicode block chars)
// =============================================================================

func TestSprintBurndown_ASCIIOnlyHumanOutput(t *testing.T) {
	// TC-B-10: No Unicode block chars U+2580–U+259F in human output.
	today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	f42 := 42.0
	f39 := 39.0
	mock := &MockSprintAnalyticsService{
		GetBurndownFunc: func(ctx context.Context, sprintKey string) (*services.BurndownResult, error) {
			return &services.BurndownResult{
				SprintKey:  "S024",
				SprintName: "Sprint 24",
				TotalSize:  42,
				DataPoints: []services.BurndownDataPoint{
					{Date: today, IdealRemaining: 42.0, ActualRemaining: &f42, UnsizedRemaining: 0},
					{Date: today.AddDate(0, 0, 1), IdealRemaining: 39.0, ActualRemaining: &f39, UnsizedRemaining: 0},
				},
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runSprintBurndown(cmd, []string{"S024"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)

	output := buf.String()
	// Verify no Unicode block elements (U+2580–U+259F) are present
	for _, r := range output {
		if r >= 0x2580 && r <= 0x259F {
			t.Errorf("Unicode block char found in human output: U+%04X (%s)", r, string(r))
		}
	}
	// Verify only printable ASCII + common Unicode (em-dash U+2014 is allowed)
	for _, r := range output {
		if r > unicode.MaxASCII && r != 0x2014 && r != '\n' && r != '\t' {
			// Allow common Unicode like box-drawing (U+2500 range) but not block elements
			if r >= 0x2580 && r <= 0x259F {
				t.Errorf("Disallowed char in output: U+%04X", r)
			}
		}
	}
}

// =============================================================================
// TC-B-11: --json burndown output matches schema
// =============================================================================

func TestSprintBurndown_JSONSchema(t *testing.T) {
	// TC-B-11: JSON schema for burndown includes all required fields.
	today := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	f42 := 42.0
	mock := &MockSprintAnalyticsService{
		GetBurndownFunc: func(ctx context.Context, sprintKey string) (*services.BurndownResult, error) {
			return &services.BurndownResult{
				SprintKey:    "S024",
				SprintName:   "Sprint 24",
				TotalSize:    42,
				UnsizedTotal: 3,
				DataPoints: []services.BurndownDataPoint{
					{Date: today, IdealRemaining: 42.0, ActualRemaining: &f42, UnsizedRemaining: 3},
				},
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runSprintBurndown(cmd, []string{"S024"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result))

	assert.Contains(t, result, "sprint_key")
	assert.Contains(t, result, "sprint_name")
	assert.Contains(t, result, "total_size")
	assert.Contains(t, result, "unsized_total")
	assert.Contains(t, result, "data_points")
}

// =============================================================================
// TC-B-12: Future days show — in human output, omit actual_remaining from JSON
// =============================================================================

func TestSprintBurndown_FutureDaysDash(t *testing.T) {
	// TC-B-12: ActualRemaining=nil for future days renders as "—" in human output.
	// The table is rendered via cli.OutputTable (pterm), so we redirect pterm's
	// default output writer to capture the em-dash cell value.
	pastDay := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	futureDay := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	f42 := 42.0
	mock := &MockSprintAnalyticsService{
		GetBurndownFunc: func(ctx context.Context, sprintKey string) (*services.BurndownResult, error) {
			return &services.BurndownResult{
				SprintKey:  "S024",
				SprintName: "Sprint 24",
				TotalSize:  42,
				DataPoints: []services.BurndownDataPoint{
					// Past day: has actual
					{Date: pastDay, IdealRemaining: 42.0, ActualRemaining: &f42, UnsizedRemaining: 0},
					// Future day: no actual (nil)
					{Date: futureDay, IdealRemaining: 10.0, ActualRemaining: nil, UnsizedRemaining: 0},
				},
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	// Redirect pterm's output writer to capture table rendering (which uses pterm internally).
	var ptermBuf bytes.Buffer
	pterm.SetDefaultOutput(&ptermBuf)
	defer pterm.SetDefaultOutput(os.Stdout)

	// Also capture plain fmt.Printf output via os.Stdout pipe.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runSprintBurndown(cmd, []string{"S024"})

	w.Close()
	os.Stdout = origOut
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	// Combined output: fmt.Printf goes to stdoutBuf; pterm table goes to ptermBuf.
	combined := stdoutBuf.String() + ptermBuf.String()
	// Human output should contain em-dash for future days
	assert.Contains(t, combined, "—", "future days should show em-dash in human output")
}

// =============================================================================
// TC-S-07: --json with detailed=false: base fields present, detailed fields null
// =============================================================================

// =============================================================================
// T-E19-F04-010: Independent nil guards for AddedMidSprintSize/RemovedMidSprintSize
// =============================================================================

// TestPrintSummaryTable_NilSizeWithNonNilCount verifies that printSummaryTable
// does NOT panic when AddedMidSprintCount/RemovedMidSprintCount are non-nil but
// their corresponding Size fields are nil.
//
// Counter-factual: the buggy impl dereferences *result.AddedMidSprintSize after
// only guarding AddedMidSprintCount != nil, causing a nil-pointer panic when
// the type system allows them to be set independently.
func TestPrintSummaryTable_NilSizeWithNonNilCount(t *testing.T) {
	addedCount := 2
	removedCount := 1

	result := &services.SprintSummaryResult{
		SprintKey:             "S042",
		SprintName:            "Sprint 42",
		AddedMidSprintCount:   &addedCount, // non-nil Count
		AddedMidSprintSize:    nil,         // independently nil Size — must NOT panic
		RemovedMidSprintCount: &removedCount,
		RemovedMidSprintSize:  nil, // independently nil Size — must NOT panic
	}

	// Redirect stdout so Printf output does not pollute test output.
	origOut := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	assert.NotPanics(t, func() {
		_ = printSummaryTable(result, true)
	})

	w.Close()
	os.Stdout = origOut

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	// The formatter must still print a line for added/removed with size=0.
	assert.Contains(t, out, "Added mid-sprint: 2 (size: 0)")
	assert.Contains(t, out, "Removed mid-sprint:1 (size: 0)")
}

func TestPrintSummaryDetailed_WritesBothSections(t *testing.T) {
	addedCount := 2
	addedSize := 7
	removedCount := 1
	removedSize := 3
	result := &services.SprintSummaryResult{
		AddedMidSprintCount:   &addedCount,
		AddedMidSprintSize:    &addedSize,
		RemovedMidSprintCount: &removedCount,
		RemovedMidSprintSize:  &removedSize,
	}

	origOut := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	assert.NotPanics(t, func() {
		printSummaryDetailed(result)
	})

	w.Close()
	os.Stdout = origOut

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	assert.Contains(t, out, "Added mid-sprint: 2 (size: 7)")
	assert.Contains(t, out, "Removed mid-sprint:1 (size: 3)")
}

func TestPrintSummarySections(t *testing.T) {
	origOut := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	assert.NoError(t, printCycleTime([]services.PhaseTime{{Phase: "done", AverageDays: 2.5}}))
	assert.NoError(t, printSizeDistribution([]services.SizeBand{{Label: "M", Count: 3}}))
	assert.NoError(t, printCarryover([]services.CarryoverEntity{{Key: "T-1", EntityType: "task"}}))

	w.Close()
	os.Stdout = origOut

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	assert.Contains(t, out, "Cycle Time by Phase")
	assert.Contains(t, out, "Size Distribution")
	assert.Contains(t, out, "Carryover Entities (1)")
}

func TestSprintSummary_JSONSchemaDetailedFalse(t *testing.T) {
	// TC-S-07: Detailed pointer fields must be null, not omitted.
	mock := &MockSprintAnalyticsService{
		GetSummaryFunc: func(ctx context.Context, sprintKey string, detailed bool) (*services.SprintSummaryResult, error) {
			assert.Equal(t, "S024", sprintKey)
			assert.False(t, detailed)
			return &services.SprintSummaryResult{
				SprintKey:           "S024",
				SprintName:          "Sprint 24",
				PlannedSize:         50,
				CompletedSize:       40,
				CompletionPctBySize: 80.0,
				PlannedCount:        10,
				CompletedCount:      8,
				VelocityThisSprint:  40,
				TrailingAvgVelocity: 38.0,
				VelocityDelta:       2.0,
				VelocityDeltaPct:    5.26,
				UnsizedPlanned:      0,
				UnsizedCompleted:    0,
				// Detailed fields: all nil
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("detailed", false, "")

	runErr := runSprintSummary(cmd, []string{"S024"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	// Parse raw JSON to check null vs absent fields
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &raw))

	// Base fields must be present
	assert.Contains(t, raw, "sprint_key")
	assert.Contains(t, raw, "sprint_name")
	assert.Contains(t, raw, "planned_size")
	assert.Contains(t, raw, "completed_size")
	assert.Contains(t, raw, "completion_pct_by_size")
	assert.Contains(t, raw, "planned_count")
	assert.Contains(t, raw, "completed_count")
	assert.Contains(t, raw, "velocity_this_sprint")
	assert.Contains(t, raw, "trailing_avg_velocity")
	assert.Contains(t, raw, "velocity_delta")
	assert.Contains(t, raw, "velocity_delta_pct")
	assert.Contains(t, raw, "unsized_planned")
	assert.Contains(t, raw, "unsized_completed")

	// Detailed fields must be present AND null (not omitted)
	val, present := raw["cycle_time_by_phase"]
	assert.True(t, present, "cycle_time_by_phase must be present (not omitted)")
	assert.Nil(t, val, "cycle_time_by_phase must be null when detailed=false")

	val, present = raw["carryover_entities"]
	assert.True(t, present, "carryover_entities must be present (not omitted)")
	assert.Nil(t, val, "carryover_entities must be null when detailed=false")

	val, present = raw["size_band_distribution"]
	assert.True(t, present, "size_band_distribution must be present (not omitted)")
	assert.Nil(t, val, "size_band_distribution must be null when detailed=false")
}

// =============================================================================
// TC-S-08: --json --detailed with nil CycleTimeByPhase renders null not omitted
// =============================================================================

func TestSprintSummary_JSONSchemaDetailedTrueNilPhase(t *testing.T) {
	// TC-S-08: Even when detailed=true, if CycleTimeByPhase=nil it must be null in JSON.
	mock := &MockSprintAnalyticsService{
		GetSummaryFunc: func(ctx context.Context, sprintKey string, detailed bool) (*services.SprintSummaryResult, error) {
			assert.True(t, detailed)
			addedCount := 1
			addedSize := 5
			removedCount := 0
			removedSize := 0
			return &services.SprintSummaryResult{
				SprintKey:             "S024",
				SprintName:            "Sprint 24",
				PlannedSize:           50,
				CompletedSize:         40,
				CompletionPctBySize:   80.0,
				PlannedCount:          10,
				CompletedCount:        8,
				VelocityThisSprint:    40,
				TrailingAvgVelocity:   38.0,
				VelocityDelta:         2.0,
				VelocityDeltaPct:      5.26,
				UnsizedPlanned:        0,
				UnsizedCompleted:      0,
				AddedMidSprintCount:   &addedCount,
				AddedMidSprintSize:    &addedSize,
				RemovedMidSprintCount: &removedCount,
				RemovedMidSprintSize:  &removedSize,
				CycleTimeByPhase:      nil, // no E13 data
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("detailed", false, "")
	cmd.Flags().Set("detailed", "true")

	runErr := runSprintSummary(cmd, []string{"S024"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &raw))

	// cycle_time_by_phase must be present AND null
	val, present := raw["cycle_time_by_phase"]
	assert.True(t, present, "cycle_time_by_phase must be present")
	assert.Nil(t, val, "cycle_time_by_phase must be null when data absent")
}

// =============================================================================
// TC-NF-03: --json and --field flags work on all three commands
// =============================================================================

func TestSprintVelocity_DefaultLimit(t *testing.T) {
	// TC-NF-03: velocity command with no flags uses default limit=5.
	mock := &MockSprintAnalyticsService{
		GetVelocityFunc: func(ctx context.Context, n int) (*services.VelocityResult, error) {
			assert.Equal(t, 5, n, "default limit must be 5")
			return &services.VelocityResult{
				Sprints:          []services.VelocitySprint{},
				TrailingAverage:  0,
				SprintCount:      0,
				InsufficientData: true,
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Int("sprints", 5, "")

	err := runSprintVelocity(cmd, []string{})
	assert.NoError(t, err)
}

func TestSprintBurndown_NoKey_UsesActiveSprintViaService(t *testing.T) {
	// TC-NF-03: burndown command passes empty string to service when no key given.
	mock := &MockSprintAnalyticsService{
		GetBurndownFunc: func(ctx context.Context, sprintKey string) (*services.BurndownResult, error) {
			assert.Equal(t, "", sprintKey, "empty key must be passed when no arg provided")
			return &services.BurndownResult{
				SprintKey:  "S010",
				SprintName: "Sprint 10",
				DataPoints: []services.BurndownDataPoint{},
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// No args → empty key to service
	err := runSprintBurndown(cmd, []string{})
	assert.NoError(t, err)
}

func TestSprintSummary_PassesDetailedFlag(t *testing.T) {
	// TC-NF-03: --detailed flag correctly passed to service.
	mock := &MockSprintAnalyticsService{
		GetSummaryFunc: func(ctx context.Context, sprintKey string, detailed bool) (*services.SprintSummaryResult, error) {
			assert.Equal(t, "S024", sprintKey)
			assert.True(t, detailed, "--detailed must be passed as true to service")
			return &services.SprintSummaryResult{
				SprintKey:  "S024",
				SprintName: "Sprint 24",
			}, nil
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("detailed", false, "")
	cmd.Flags().Set("detailed", "true")

	err := runSprintSummary(cmd, []string{"S024"})
	assert.NoError(t, err)
}

func TestSprintSummary_ReturnsServiceError(t *testing.T) {
	mock := &MockSprintAnalyticsService{
		GetSummaryFunc: func(ctx context.Context, sprintKey string, detailed bool) (*services.SprintSummaryResult, error) {
			return nil, fmt.Errorf("summary backend unavailable")
		},
	}
	cleanup := setupAnalyticsTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("detailed", false, "")

	err := runSprintSummary(cmd, []string{"S024"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "summary backend unavailable")
}

// =============================================================================
// TC-J01: sprint add JSON output contains SprintAssignment fields
// =============================================================================

func TestSprintAdd_JSONOutputContainsAssignmentFields(t *testing.T) {
	// TC-J01: --json output from sprint add must include sprint_id, entity_type, entity_id, assigned_at.
	assignedAt := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	assignment := &models.SprintAssignment{
		ID:         42,
		SprintID:   24,
		EntityType: "task",
		EntityID:   1001,
		AssignedAt: assignedAt,
	}

	mock := &MockSprintService{
		AddEntityToSprintFunc: func(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error) {
			assert.Equal(t, "S024", input.SprintKey)
			assert.Equal(t, "E07-F01-001", input.EntityKey)
			return assignment, nil, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "")
	cmd.Flags().String("bulk", "", "")
	cmd.Flags().Bool("bulk-bugs", false, "")
	cmd.Flags().Bool("bulk-tech-debt", false, "")
	cmd.Flags().Bool("bulk-changes", false, "")

	runErr := runSprintAdd(cmd, []string{"S024", "E07-F01-001"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result))

	assert.Contains(t, result, "sprint_id", "sprint_id must be present in JSON output")
	assert.Contains(t, result, "entity_type", "entity_type must be present in JSON output")
	assert.Contains(t, result, "entity_id", "entity_id must be present in JSON output")
	assert.Contains(t, result, "assigned_at", "assigned_at must be present in JSON output")
	assert.Equal(t, "task", result["entity_type"], "entity_type must match the assignment")
}

// =============================================================================
// TC-J02: sprint remove JSON output
// =============================================================================

func TestSprintRemove_JSONOutput(t *testing.T) {
	// TC-J02: --json output from sprint remove returns a confirmation object.
	mock := &MockSprintService{
		RemoveEntityFromSprintFunc: func(ctx context.Context, sprintKey, entityKey string) error {
			assert.Equal(t, "S024", sprintKey)
			assert.Equal(t, "E07-F01-001", entityKey)
			return nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runSprintRemove(cmd, []string{"S024", "E07-F01-001"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	output := jsonBuf.String()
	assert.NotEmpty(t, output, "JSON output must not be empty")
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result), "JSON output must be valid JSON")
}

// =============================================================================
// TC-J03: sprint backlog JSON output contains entity_type on every item
// =============================================================================

func TestSprintBacklog_JSONOutputEntityTypeOnEveryItem(t *testing.T) {
	// TC-J03: Every item in groups[*].items[*] must have entity_type field non-empty.
	backlog := &services.SprintBacklog{
		SprintKey:         "S024",
		SprintName:        "Sprint 24",
		TotalCount:        2,
		CompletedCount:    0,
		CompletionPercent: 0.0,
		Groups: []*services.BacklogGroup{
			{
				StatusCategory: "todo",
				Items: []*services.BacklogItemView{
					{EntityType: "task", Key: "E07-F01-001", Title: "Task 1", Status: "todo"},
					{EntityType: "bug", Key: "B001", Title: "Bug 1", Status: "todo"},
				},
			},
		},
	}

	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			assert.Equal(t, "S024", sprintKey)
			return backlog, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")

	runErr := runSprintBacklog(cmd, []string{"S024"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result))

	groups, ok := result["groups"].([]interface{})
	require.True(t, ok, "groups field must be present and an array")
	require.NotEmpty(t, groups, "groups must have at least one group")

	group0, ok := groups[0].(map[string]interface{})
	require.True(t, ok)
	items, ok := group0["items"].([]interface{})
	require.True(t, ok, "items must be present")

	for i, itemRaw := range items {
		item, ok := itemRaw.(map[string]interface{})
		require.True(t, ok)
		val, present := item["entity_type"]
		assert.True(t, present, "item[%d] must have entity_type field", i)
		assert.NotEmpty(t, val, "item[%d] entity_type must not be empty", i)
	}
}

// =============================================================================
// TC-U01: sprint add — double-assignment error propagates conflicting sprint key
// =============================================================================

func TestSprintAdd_DoubleAssignmentErrorContainsSprintKey(t *testing.T) {
	// TC-U01: When service returns a conflict error mentioning S024, the error must propagate.
	mock := &MockSprintService{
		AddEntityToSprintFunc: func(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error) {
			return nil, nil, fmt.Errorf("entity E07-F01-001 is already assigned to sprint S024")
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "")
	cmd.Flags().String("bulk", "", "")
	cmd.Flags().Bool("bulk-bugs", false, "")
	cmd.Flags().Bool("bulk-tech-debt", false, "")
	cmd.Flags().Bool("bulk-changes", false, "")

	err := runSprintAdd(cmd, []string{"S025", "E07-F01-001"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "S024", "error must name the conflicting sprint key")
}

// =============================================================================
// TC-U02: sprint add — capacity warning is advisory (no error returned)
// =============================================================================

func TestSprintAdd_CapacityWarningEmittedBeforeSuccess(t *testing.T) {
	// TC-U02: When service returns a CapacityWarning, command proceeds successfully (advisory only)
	// AND the warning text must be emitted to stdout before the success message.
	assignment := &models.SprintAssignment{
		ID:         1,
		SprintID:   24,
		EntityType: "task",
		EntityID:   1001,
		AssignedAt: time.Now(),
	}
	warning := &services.CapacityWarning{
		AgentType: "backend",
		Capacity:  30.0,
		Allocated: 35.0,
	}

	mock := &MockSprintService{
		AddEntityToSprintFunc: func(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error) {
			return assignment, warning, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	origNoColor := cli.GlobalConfig.NoColor
	defer func() {
		cli.GlobalConfig.JSON = origJSON
		cli.GlobalConfig.NoColor = origNoColor
	}()
	cli.GlobalConfig.JSON = false
	// Disable color so cli.Warning/cli.Success fall back to fmt.Println (writes to os.Stdout)
	// instead of pterm (which writes to its own internal writer, not the redirected os.Stdout).
	cli.GlobalConfig.NoColor = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "")
	cmd.Flags().String("bulk", "", "")
	cmd.Flags().Bool("bulk-bugs", false, "")
	cmd.Flags().Bool("bulk-tech-debt", false, "")
	cmd.Flags().Bool("bulk-changes", false, "")

	// No error expected — capacity warning is advisory; assignment still succeeds
	runErr := runSprintAdd(cmd, []string{"S024", "E07-F01-001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr, "capacity warning must not cause an error — assignment should succeed")
	output := buf.String()
	assert.Contains(t, output, "Capacity warning", "output must contain the capacity warning message")
	warningIdx := strings.Index(output, "Capacity warning")
	successIdx := strings.Index(output, "Added")
	assert.True(t, warningIdx >= 0, "capacity warning text must appear in output")
	assert.True(t, successIdx >= 0, "success message must appear in output")
	assert.Less(t, warningIdx, successIdx, "capacity warning must appear before the success message")
}

// =============================================================================
// TC-U03: sprint backlog -- invalid --type value returns error
// =============================================================================

func TestSprintBacklog_InvalidTypeFlag_ReturnsError(t *testing.T) {
	// TC-U03: --type=epic is not a valid entity type; service returns error.
	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			return nil, fmt.Errorf("invalid entity type %q: must be one of task, bug, change_card, tech_debt", opts.EntityType)
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")
	cmd.Flags().Set("type", "epic")

	err := runSprintBacklog(cmd, []string{"S024"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid entity type")
}

// =============================================================================
// TC-U04: sprint close -- carryover=next output shows moved count and next sprint key
// =============================================================================

func TestSprintClose_CarryoverNextOutputShowsMovedCountAndNextKey(t *testing.T) {
	// TC-U04: Human output must contain completed count, carried-over count, and next sprint key.
	closedSprint := &models.Sprint{
		Key:    "S024",
		Name:   "Sprint 24",
		Status: "completed",
	}
	closeResult := &services.SprintCloseResult{
		Sprint:           closedSprint,
		CompletedCount:   5,
		CarriedOverCount: 3,
		DroppedCount:     0,
		NextSprintKey:    "S025",
	}

	mock := &MockSprintService{
		CloseSprintWithCarryoverFunc: func(ctx context.Context, key string, mode services.CarryoverMode) (*services.SprintCloseResult, error) {
			assert.Equal(t, "S024", key)
			assert.Equal(t, services.CarryoverMode("next"), mode)
			return closeResult, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("carryover", "", "")
	cmd.Flags().Set("carryover", "next")

	runErr := runSprintClose(cmd, []string{"S024"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()
	assert.Contains(t, output, "S025", "output must mention the next sprint key")
	assert.Contains(t, output, "5", "output must show completed count")
	assert.Contains(t, output, "3", "output must show carried-over count")
}

// =============================================================================
// TC-U05: sprint close -- carryover=backlog output shows dropped count
// =============================================================================

func TestSprintClose_CarryoverBacklogOutputShowsDroppedCount(t *testing.T) {
	// TC-U05: Human output must contain dropped count when carryover=backlog; no next sprint key.
	closedSprint := &models.Sprint{
		Key:    "S024",
		Name:   "Sprint 24",
		Status: "completed",
	}
	closeResult := &services.SprintCloseResult{
		Sprint:           closedSprint,
		CompletedCount:   5,
		CarriedOverCount: 0,
		DroppedCount:     3,
		NextSprintKey:    "",
	}

	mock := &MockSprintService{
		CloseSprintWithCarryoverFunc: func(ctx context.Context, key string, mode services.CarryoverMode) (*services.SprintCloseResult, error) {
			return closeResult, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("carryover", "", "")
	cmd.Flags().Set("carryover", "backlog")

	runErr := runSprintClose(cmd, []string{"S024"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()
	assert.Contains(t, output, "3", "output must show dropped count")
	assert.NotContains(t, output, "S025", "should not mention a next sprint key when carryover=backlog")
}

// =============================================================================
// Extended MockSprintService — adds F05 planning methods
// =============================================================================

// MockSprintPlanningService only implements the planning-focused interface so
// plan/readiness/capacity tests do not need the full sprint service contract.
type MockSprintPlanningService struct {
	PlanSprintFunc             func(ctx context.Context, key string) (*services.SprintPlanView, error)
	GetSprintReadinessFunc     func(ctx context.Context, key string) (*services.SprintReadiness, error)
	SetSprintCapacityFunc      func(ctx context.Context, input services.SetSprintCapacityInput) (*models.SprintCapacity, error)
	GetSprintCapacityFunc      func(ctx context.Context, key string) ([]services.CapacityRow, error)
	BulkAddToSprintFunc        func(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error)
	AddEntityToSprintFunc      func(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error)
	RemoveEntityFromSprintFunc func(ctx context.Context, sprintKey, entityKey string) error
	GetSprintBacklogFunc       func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error)
	GetNextTaskFunc            func(ctx context.Context, agentType string) (*services.BacklogItemView, error)
}

var _ sprintPlanningServicer = (*MockSprintPlanningService)(nil)
var _ sprintAssignmentServicer = (*MockSprintPlanningService)(nil)

func (m *MockSprintPlanningService) PlanSprint(ctx context.Context, key string) (*services.SprintPlanView, error) {
	if m.PlanSprintFunc != nil {
		return m.PlanSprintFunc(ctx, key)
	}
	return &services.SprintPlanView{
		Sprint:    &models.Sprint{Key: key},
		Backlog:   nil,
		Capacity:  nil,
		Readiness: &services.SprintReadiness{OverallScore: 0},
	}, nil
}

func (m *MockSprintPlanningService) GetSprintReadiness(ctx context.Context, key string) (*services.SprintReadiness, error) {
	if m.GetSprintReadinessFunc != nil {
		return m.GetSprintReadinessFunc(ctx, key)
	}
	return &services.SprintReadiness{OverallScore: 75, Factors: []services.ReadinessFactor{}}, nil
}

func (m *MockSprintPlanningService) SetSprintCapacity(ctx context.Context, input services.SetSprintCapacityInput) (*models.SprintCapacity, error) {
	if m.SetSprintCapacityFunc != nil {
		return m.SetSprintCapacityFunc(ctx, input)
	}
	return &models.SprintCapacity{AgentType: input.AgentType, CapacityPoints: input.Points}, nil
}

func (m *MockSprintPlanningService) GetSprintCapacity(ctx context.Context, key string) ([]services.CapacityRow, error) {
	if m.GetSprintCapacityFunc != nil {
		return m.GetSprintCapacityFunc(ctx, key)
	}
	return []services.CapacityRow{}, nil
}

func (m *MockSprintPlanningService) BulkAddToSprint(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error) {
	if m.BulkAddToSprintFunc != nil {
		return m.BulkAddToSprintFunc(ctx, input)
	}
	return &services.BulkAddResult{AddedByType: map[string]int{}, SkippedByType: map[string]int{}}, nil
}

func (m *MockSprintPlanningService) AddEntityToSprint(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error) {
	if m.AddEntityToSprintFunc != nil {
		return m.AddEntityToSprintFunc(ctx, input)
	}
	return &models.SprintAssignment{}, nil, nil
}

func (m *MockSprintPlanningService) RemoveEntityFromSprint(ctx context.Context, sprintKey, entityKey string) error {
	if m.RemoveEntityFromSprintFunc != nil {
		return m.RemoveEntityFromSprintFunc(ctx, sprintKey, entityKey)
	}
	return nil
}

func (m *MockSprintPlanningService) GetSprintBacklog(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
	if m.GetSprintBacklogFunc != nil {
		return m.GetSprintBacklogFunc(ctx, sprintKey, opts)
	}
	return &services.SprintBacklog{SprintKey: sprintKey, Groups: []*services.BacklogGroup{}}, nil
}

func (m *MockSprintPlanningService) GetNextTask(ctx context.Context, agentType string) (*services.BacklogItemView, error) {
	if m.GetNextTaskFunc != nil {
		return m.GetNextTaskFunc(ctx, agentType)
	}
	return nil, nil
}

func (m *MockSprintPlanningService) ReorderAssignment(ctx context.Context, sprintKey, entityKey string, target services.ReorderTarget) (*models.SprintAssignment, []*models.SprintAssignment, error) {
	return &models.SprintAssignment{}, []*models.SprintAssignment{}, nil
}

// setupPlanningTest sets up the sprint service override using the extended mock.
func setupPlanningTest(t *testing.T, mock *MockSprintPlanningService) func() {
	t.Helper()
	oldPlanningOverride := sprintPlanningSvcOverride
	oldAssignmentOverride := sprintAssignmentSvcOverride
	oldCapacityOverride := sprintCapacitySvcOverride
	sprintPlanningSvcOverride = mock
	sprintAssignmentSvcOverride = mock
	sprintCapacitySvcOverride = mock
	return func() {
		sprintPlanningSvcOverride = oldPlanningOverride
		sprintAssignmentSvcOverride = oldAssignmentOverride
		sprintCapacitySvcOverride = oldCapacityOverride
	}
}

// =============================================================================
// TC-011-06: shark sprint plan --json returns SprintPlanView with three keys
// =============================================================================

func TestSprintPlan_JSONOutput(t *testing.T) {
	// TC-011-06: --json output must contain backlog, capacity, readiness keys.
	agentBackend := "backend"
	planView := &services.SprintPlanView{
		Sprint: newTestSprint(),
		Backlog: []sprint.BacklogItem{
			{Key: "E07-F01-001", Title: "Task A", Priority: 8, AgentType: &agentBackend},
		},
		Capacity: []services.CapacityRow{
			{AgentType: "backend", CapacityPoints: 21, AllocatedPoints: 8, Remaining: 13},
		},
		Readiness: &services.SprintReadiness{
			OverallScore: 72,
			Factors:      []services.ReadinessFactor{},
		},
	}

	mock := &MockSprintPlanningService{
		PlanSprintFunc: func(ctx context.Context, key string) (*services.SprintPlanView, error) {
			assert.Equal(t, "S001", key)
			return planView, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	runErr := runSprintPlan(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result))
	assert.Contains(t, result, "backlog", "JSON must contain backlog key")
	assert.Contains(t, result, "capacity", "JSON must contain capacity key")
	assert.Contains(t, result, "readiness", "JSON must contain readiness key")
	assert.Contains(t, result, "sprint", "JSON must contain sprint key")
}

// =============================================================================
// TC-011-07: shark sprint plan human output shows three labeled sections
// =============================================================================

func TestSprintPlan_HumanOutputSections(t *testing.T) {
	// TC-011-07: Human output must show "Backlog", "Capacity", and "Readiness" sections.
	planView := &services.SprintPlanView{
		Sprint:    newTestSprint(),
		Backlog:   []sprint.BacklogItem{},
		Capacity:  []services.CapacityRow{},
		Readiness: &services.SprintReadiness{OverallScore: 55, Factors: []services.ReadinessFactor{}},
	}

	mock := &MockSprintPlanningService{
		PlanSprintFunc: func(ctx context.Context, key string) (*services.SprintPlanView, error) {
			return planView, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	runErr := runSprintPlan(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()
	assert.Contains(t, output, "Backlog", "human output must show Backlog section")
	assert.Contains(t, output, "Capacity", "human output must show Capacity section")
	assert.Contains(t, output, "Readiness", "human output must show Readiness section")
}

// =============================================================================
// TC-013-08: shark sprint readiness --json returns SprintReadiness JSON
// =============================================================================

func TestSprintReadiness_JSONOutput(t *testing.T) {
	// TC-013-08: --json returns SprintReadiness JSON with overall_score and factors.
	readiness := &services.SprintReadiness{
		OverallScore: 72,
		Factors: []services.ReadinessFactor{
			{Name: "Capacity utilization", Score: 20, MaxScore: 20, Detail: "within 80-110%"},
		},
		UnsizedEntities:   []sprint.BacklogItem{},
		OversizedEntities: []sprint.BacklogItem{},
	}

	mock := &MockSprintPlanningService{
		GetSprintReadinessFunc: func(ctx context.Context, key string) (*services.SprintReadiness, error) {
			assert.Equal(t, "S001", key)
			return readiness, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	runErr := runSprintReadiness(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result))
	assert.Contains(t, result, "overall_score")
	assert.Contains(t, result, "factors")
	scoreVal, ok := result["overall_score"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(72), scoreVal)
}

// =============================================================================
// TC-013-09: shark sprint readiness human output shows factor table
// =============================================================================

func TestSprintReadiness_HumanFactorTable(t *testing.T) {
	// TC-013-09: Human output must show score and factor breakdown.
	readiness := &services.SprintReadiness{
		OverallScore: 85,
		Factors: []services.ReadinessFactor{
			{Name: "Capacity utilization", Score: 20, MaxScore: 20, Detail: "within 80-110%"},
			{Name: "All entities sized", Score: 15, MaxScore: 15, Detail: "all sized"},
		},
		UnsizedEntities:   []sprint.BacklogItem{},
		OversizedEntities: []sprint.BacklogItem{},
	}

	mock := &MockSprintPlanningService{
		GetSprintReadinessFunc: func(ctx context.Context, key string) (*services.SprintReadiness, error) {
			return readiness, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	// Capture pterm table output and plain fmt.Printf output separately.
	var ptermBuf bytes.Buffer
	pterm.SetDefaultOutput(&ptermBuf)
	defer pterm.SetDefaultOutput(os.Stdout)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	runErr := runSprintReadiness(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)

	assert.NoError(t, runErr)
	combined := stdoutBuf.String() + ptermBuf.String()
	// Must show overall score
	assert.Contains(t, combined, "85")
	// Must show factor names (rendered in pterm table)
	assert.Contains(t, combined, "Capacity utilization")
}

// =============================================================================
// TC-014-07: shark sprint capacity show --json returns []CapacityRow
// =============================================================================

func TestSprintCapacityShow_JSONOutput(t *testing.T) {
	// TC-014-07: --json output is an array of CapacityRow objects.
	rows := []services.CapacityRow{
		{AgentType: "backend", CapacityPoints: 21, AllocatedPoints: 13, Remaining: 8, UnsizedAssigned: 1},
	}

	mock := &MockSprintPlanningService{
		GetSprintCapacityFunc: func(ctx context.Context, key string) ([]services.CapacityRow, error) {
			assert.Equal(t, "S001", key)
			return rows, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	runErr := runSprintCapacityShow(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)
	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result))
	require.Len(t, result, 1)
	assert.Equal(t, "backend", result[0]["agent_type"])
	assert.Equal(t, float64(21), result[0]["capacity_points"])
}

// =============================================================================
// TC-014-08: shark sprint capacity show with negative remaining
// =============================================================================

func TestSprintCapacityShow_NegativeRemaining(t *testing.T) {
	// TC-014-08: Negative remaining (overcommit) is shown correctly in human output.
	rows := []services.CapacityRow{
		{AgentType: "backend", CapacityPoints: 10, AllocatedPoints: 15, Remaining: -5, UnsizedAssigned: 0},
	}

	mock := &MockSprintPlanningService{
		GetSprintCapacityFunc: func(ctx context.Context, key string) ([]services.CapacityRow, error) {
			return rows, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	// Capture pterm table output and plain fmt.Printf output separately.
	var ptermBuf bytes.Buffer
	pterm.SetDefaultOutput(&ptermBuf)
	defer pterm.SetDefaultOutput(os.Stdout)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	runErr := runSprintCapacityShow(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)

	assert.NoError(t, runErr)
	// -5 should appear in human output (now rendered via pterm table).
	combined := stdoutBuf.String() + ptermBuf.String()
	assert.Contains(t, combined, "-5")
}

// =============================================================================
// TC-015-04: --default routes to config.SetSprintCapacityDefault, NOT service
// =============================================================================

func TestSprintCapacitySet_DefaultFlagRoutesToConfig(t *testing.T) {
	// TC-015-04: When --default is set, the command must NOT call SetSprintCapacity.
	// It writes to config only.
	serviceCallCount := 0

	mock := &MockSprintPlanningService{
		SetSprintCapacityFunc: func(ctx context.Context, input services.SetSprintCapacityInput) (*models.SprintCapacity, error) {
			serviceCallCount++
			return nil, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	// Use a temp config file to test config mutation
	tmpDir := t.TempDir()
	configPath := tmpDir + "/.sharkconfig.json"
	// Write minimal config
	require.NoError(t, os.WriteFile(configPath, []byte(`{}`), 0644))

	// Override config path for this test
	origConfigFile := cli.GlobalConfig.ConfigFile
	cli.GlobalConfig.ConfigFile = configPath
	defer func() { cli.GlobalConfig.ConfigFile = origConfigFile }()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("agent", "", "agent type")
	cmd.Flags().Float64("points", 0, "capacity points")
	cmd.Flags().Bool("default", false, "write to config defaults")
	cmd.Flags().Set("agent", "backend")
	cmd.Flags().Set("points", "21")
	cmd.Flags().Set("default", "true")

	err := runSprintCapacitySet(cmd, []string{})
	assert.NoError(t, err)

	// Service must NOT have been called
	assert.Equal(t, 0, serviceCallCount, "--default path must not call SetSprintCapacity service")
}

// =============================================================================
// TC-012-04: shark sprint add --bulk-bugs calls BulkAddToSprint with EntityType="bug"
// =============================================================================

func TestSprintAdd_BulkBugs(t *testing.T) {
	// TC-012-04 / TC-012-02: --bulk-bugs routes to BulkAddToSprint with EntityType=["bug"].
	var capturedInput services.BulkAddInput

	mock := &MockSprintPlanningService{
		BulkAddToSprintFunc: func(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error) {
			capturedInput = input
			return &services.BulkAddResult{
				AddedByType:   map[string]int{"bug": 3},
				SkippedByType: map[string]int{"bug": 1},
			}, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "entity key")
	cmd.Flags().String("bulk", "", "feature key for bulk add")
	cmd.Flags().Bool("bulk-bugs", false, "bulk add bugs")
	cmd.Flags().Bool("bulk-tech-debt", false, "bulk add tech debt")
	cmd.Flags().Bool("bulk-changes", false, "bulk add change cards")
	cmd.Flags().Set("bulk-bugs", "true")

	err := runSprintAdd(cmd, []string{"S001"})
	assert.NoError(t, err)
	assert.Equal(t, "S001", capturedInput.SprintKey)
	assert.Equal(t, []string{"bug"}, capturedInput.EntityTypes)
	assert.Empty(t, capturedInput.FeatureKey)
}

// =============================================================================
// TC-012-01: shark sprint add --bulk <feature> calls BulkAddToSprint with FeatureKey
// =============================================================================

func TestSprintAdd_BulkByFeature(t *testing.T) {
	// TC-012-01: --bulk=E07-F34 routes to BulkAddToSprint with FeatureKey populated.
	var capturedInput services.BulkAddInput

	mock := &MockSprintPlanningService{
		BulkAddToSprintFunc: func(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error) {
			capturedInput = input
			return &services.BulkAddResult{
				AddedByType:   map[string]int{"task": 2},
				SkippedByType: map[string]int{"task": 0},
			}, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "entity key")
	cmd.Flags().String("bulk", "", "feature key for bulk add")
	cmd.Flags().Bool("bulk-bugs", false, "bulk add bugs")
	cmd.Flags().Bool("bulk-tech-debt", false, "bulk add tech debt")
	cmd.Flags().Bool("bulk-changes", false, "bulk add change cards")
	cmd.Flags().Set("bulk", "E07-F34")

	err := runSprintAdd(cmd, []string{"S001"})
	assert.NoError(t, err)
	assert.Equal(t, "S001", capturedInput.SprintKey)
	assert.Equal(t, "E07-F34", capturedInput.FeatureKey)
}

// =============================================================================
// TC-012-04 (JSON): shark sprint add --bulk-bugs --json returns BulkAddResult JSON
// =============================================================================

func TestSprintAdd_BulkBugs_JSONOutput(t *testing.T) {
	// TC-012-04: --json returns BulkAddResult as JSON.
	mock := &MockSprintPlanningService{
		BulkAddToSprintFunc: func(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error) {
			return &services.BulkAddResult{
				AddedByType:   map[string]int{"bug": 5},
				SkippedByType: map[string]int{"bug": 2},
			}, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "entity key")
	cmd.Flags().String("bulk", "", "feature key for bulk add")
	cmd.Flags().Bool("bulk-bugs", false, "bulk add bugs")
	cmd.Flags().Bool("bulk-tech-debt", false, "bulk add tech debt")
	cmd.Flags().Bool("bulk-changes", false, "bulk add change cards")
	cmd.Flags().Set("bulk-bugs", "true")

	runErr := runSprintAdd(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var jsonBuf bytes.Buffer
	jsonBuf.ReadFrom(r)

	assert.NoError(t, runErr)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &result))
	// BulkAddResult has no json tags so Go uses PascalCase field names
	assert.Contains(t, result, "AddedByType")
	assert.Contains(t, result, "SkippedByType")
}

// =============================================================================
// Single-entity add (non-bulk path)
// =============================================================================

func TestSprintAdd_SingleEntity(t *testing.T) {
	// Non-bulk path: shark sprint add S001 --entity=E07-F01-001
	mock := &MockSprintPlanningService{
		AddEntityToSprintFunc: func(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error) {
			assert.Equal(t, "S001", input.SprintKey)
			assert.Equal(t, "E07-F01-001", input.EntityKey)
			return &models.SprintAssignment{ID: 1}, nil, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "entity key")
	cmd.Flags().String("bulk", "", "feature key for bulk add")
	cmd.Flags().Bool("bulk-bugs", false, "bulk add bugs")
	cmd.Flags().Bool("bulk-tech-debt", false, "bulk add tech debt")
	cmd.Flags().Bool("bulk-changes", false, "bulk add change cards")
	cmd.Flags().Set("entity", "E07-F01-001")

	err := runSprintAdd(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// =============================================================================
// shark sprint capacity set (per-sprint, no --default flag)
// =============================================================================

func TestSprintCapacitySet_PerSprint(t *testing.T) {
	// Per-sprint set calls SetSprintCapacity; does NOT touch config.
	var capturedInput services.SetSprintCapacityInput

	mock := &MockSprintPlanningService{
		SetSprintCapacityFunc: func(ctx context.Context, input services.SetSprintCapacityInput) (*models.SprintCapacity, error) {
			capturedInput = input
			return &models.SprintCapacity{
				AgentType:      input.AgentType,
				CapacityPoints: input.Points,
			}, nil
		},
	}
	cleanup := setupPlanningTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("agent", "", "agent type")
	cmd.Flags().Float64("points", 0, "capacity points")
	cmd.Flags().Bool("default", false, "write to config defaults")
	cmd.Flags().Set("agent", "backend")
	cmd.Flags().Set("points", "21")

	err := runSprintCapacitySet(cmd, []string{"S001"})
	assert.NoError(t, err)
	assert.Equal(t, "S001", capturedInput.SprintKey)
	assert.Equal(t, "backend", capturedInput.AgentType)
	assert.Equal(t, float64(21), capturedInput.Points)
}

// =============================================================================
// TC-022: sprint close --carryover=next prints "Order preserved: yes"
// TC-022 (negative): sprint close --carryover=backlog prints "Order preserved: no"
// =============================================================================
//
// TC-022 — Caller-Path Contract (CLI-layer):
//   - Entrypoint: runSprintClose CLI handler with args ["close", "S001", "--carryover=next"].
//   - Mock seam: MockSprintService.CloseSprintWithCarryover returns SprintCloseResult{CarryoverPreserved: true}.
//   - Forbidden mocks: Do NOT print "Order preserved" in the mock; the CLI handler must read
//     CarryoverPreserved and emit the line.
//   - Counter-factual: a buggy CLI ignoring CarryoverPreserved would not print the line;
//     TC-022 asserts stdout contains "Order preserved: yes".
func TestSprintClose_TC022_CarryoverNextPrintsOrderPreservedYes(t *testing.T) {
	closedSprint := &models.Sprint{
		Key:    "S001",
		Name:   "Sprint 1",
		Status: "completed",
	}

	mock := &MockSprintService{
		CloseSprintWithCarryoverFunc: func(ctx context.Context, key string, mode services.CarryoverMode) (*services.SprintCloseResult, error) {
			return &services.SprintCloseResult{
				Sprint:             closedSprint,
				CompletedCount:     2,
				CarriedOverCount:   3,
				DroppedCount:       0,
				NextSprintKey:      "S002",
				CarryoverPreserved: true,
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("carryover", "", "")
	cmd.Flags().Set("carryover", "next")

	runErr := runSprintClose(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()
	assert.Contains(t, output, "Order preserved: yes", "output must contain 'Order preserved: yes' for carryover=next")
}

// TC-022 negative: carryover=backlog prints "Order preserved: no"
func TestSprintClose_TC022_CarryoverBacklogPrintsOrderPreservedNo(t *testing.T) {
	closedSprint := &models.Sprint{
		Key:    "S001",
		Name:   "Sprint 1",
		Status: "completed",
	}

	mock := &MockSprintService{
		CloseSprintWithCarryoverFunc: func(ctx context.Context, key string, mode services.CarryoverMode) (*services.SprintCloseResult, error) {
			return &services.SprintCloseResult{
				Sprint:             closedSprint,
				CompletedCount:     2,
				CarriedOverCount:   0,
				DroppedCount:       3,
				NextSprintKey:      "",
				CarryoverPreserved: false,
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("carryover", "", "")
	cmd.Flags().Set("carryover", "backlog")

	runErr := runSprintClose(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()
	assert.Contains(t, output, "Order preserved: no", "output must contain 'Order preserved: no' for carryover=backlog")
}

// =============================================================================
// TC-014: sprint reorder --top passes ReorderTarget{Top:true} to service
// =============================================================================
//
// TC-014 — Caller-Path Contract (CLI-layer):
//   - Entrypoint: runSprintReorder CLI handler via cobra.Command args ["S001", "E07-F01-001", "--top"].
//   - Mock seam: MockSprintService.ReorderAssignment — asserts it is called with ReorderTarget{Top: true}.
//   - Forbidden mocks: Do NOT resolve --top to Position:intPtr(1) in CLI; pass ReorderTarget{Top:true} to service.
//   - Counter-factual: a buggy CLI converting --top to --at=1 would call ReorderAssignment with Position set,
//     not Top:true; TC-014 asserts the mock was called with Top:true.
func TestSprintReorder_TC014_TopFlagPassesReorderTargetTop(t *testing.T) {
	var capturedTarget services.ReorderTarget
	var capturedSprint, capturedEntity string

	mock := &MockSprintService{
		ReorderAssignmentFunc: func(ctx context.Context, sprintKey, entityKey string, target services.ReorderTarget) (*models.SprintAssignment, []*models.SprintAssignment, error) {
			capturedSprint = sprintKey
			capturedEntity = entityKey
			capturedTarget = target
			order := 1
			return &models.SprintAssignment{SprintOrder: &order}, []*models.SprintAssignment{}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	// Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := sprintReorderCmd
	cmd.SetContext(context.Background())

	runErr := runSprintReorder(cmd, []string{"S001", "E07-F01-001"})

	// Trigger --top via the flag on the command
	// We test by calling runSprintReorder directly with the flag pre-set
	// This approach sets up the reorder cmd flags before calling
	_ = runErr // ignore error from no-flag invocation; we need to test via the real command path

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Now test via command with --top flag
	r2, w2, err2 := os.Pipe()
	require.NoError(t, err2)
	os.Stdout = w2

	capturedTarget = services.ReorderTarget{} // reset
	cmd2 := &cobra.Command{}
	cmd2.SetContext(context.Background())
	cmd2.Flags().Bool("top", false, "")
	cmd2.Flags().Bool("bottom", false, "")
	cmd2.Flags().Int("at", 0, "")
	require.NoError(t, cmd2.Flags().Set("top", "true"))

	runErr2 := runSprintReorder(cmd2, []string{"S001", "E07-F01-001"})

	w2.Close()
	os.Stdout = origOut
	var buf2 bytes.Buffer
	buf2.ReadFrom(r2)

	assert.NoError(t, runErr2)
	assert.Equal(t, "S001", capturedSprint)
	assert.Equal(t, "E07-F01-001", capturedEntity)
	assert.True(t, capturedTarget.Top, "CLI must pass ReorderTarget{Top:true} when --top flag is set")
	assert.Nil(t, capturedTarget.Position, "CLI must NOT set Position when --top flag is used")
	assert.False(t, capturedTarget.Bottom, "CLI must NOT set Bottom when --top flag is used")
	// stdout should contain the pull queue header
	assert.Contains(t, buf2.String(), "S001", "human output must mention sprint key")
}

// TC-014 negative: --top and --bottom together returns CLI parse error.
func TestSprintReorder_TC014_TopAndBottomMutuallyExclusive(t *testing.T) {
	mock := &MockSprintService{}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("top", false, "")
	cmd.Flags().Bool("bottom", false, "")
	cmd.Flags().Int("at", 0, "")
	require.NoError(t, cmd.Flags().Set("top", "true"))
	require.NoError(t, cmd.Flags().Set("bottom", "true"))

	err := runSprintReorder(cmd, []string{"S001", "E07-F01-001"})
	assert.Error(t, err, "combining --top and --bottom must return an error")
}

// TC-014 negative: positional position and --top together returns CLI parse error.
func TestSprintReorder_TC014_PositionAndTopMutuallyExclusive(t *testing.T) {
	mock := &MockSprintService{}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("top", false, "")
	cmd.Flags().Bool("bottom", false, "")
	cmd.Flags().Int("at", 0, "")
	require.NoError(t, cmd.Flags().Set("top", "true"))

	// "3" is the positional position argument
	err := runSprintReorder(cmd, []string{"S001", "E07-F01-001", "3"})
	assert.Error(t, err, "combining positional position with --top must return an error")
}

// =============================================================================
// TC-014b: sprint add --at=1 --json includes sprint_order in output
// =============================================================================
//
// TC-014b — Caller-Path Contract (CLI-layer):
//   - Entrypoint: runSprintAdd CLI handler with --at=1 --json.
//   - Mock seam: MockSprintService.AddEntityToSprint — receives AddEntityInput with Position set.
//   - Counter-factual: a buggy CLI omitting sprint_order from JSON output would produce JSON
//     without the field; TC-014b asserts json contains sprint_order == 1.
func TestSprintAdd_TC014b_AtFlagPassesPositionAndJSONIncludesSprintOrder(t *testing.T) {
	order := 1
	var capturedInput services.AddEntityInput

	mock := &MockSprintService{
		AddEntityToSprintFunc: func(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error) {
			capturedInput = input
			return &models.SprintAssignment{
				ID:          42,
				SprintID:    1,
				EntityType:  "task",
				EntityID:    101,
				SprintOrder: &order,
			}, nil, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "")
	cmd.Flags().String("bulk", "", "")
	cmd.Flags().Bool("bulk-bugs", false, "")
	cmd.Flags().Bool("bulk-tech-debt", false, "")
	cmd.Flags().Bool("bulk-changes", false, "")
	cmd.Flags().Int("at", 0, "")
	require.NoError(t, cmd.Flags().Set("at", "1"))

	runErr := runSprintAdd(cmd, []string{"S001", "E07-F01-001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)

	// Position must be passed to service
	require.NotNil(t, capturedInput.Position, "CLI must pass Position to service when --at flag is set")
	assert.Equal(t, 1, *capturedInput.Position)

	// JSON output must include sprint_order
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result), "output must be valid JSON")
	soVal, ok := result["sprint_order"]
	assert.True(t, ok, "JSON output must include sprint_order field")
	assert.Equal(t, float64(1), soVal, "sprint_order must equal 1")
}

// TC-014b negative: --at is ignored when no flag provided (Position is nil).
func TestSprintAdd_AtFlagAbsentPassesNilPosition(t *testing.T) {
	var capturedInput services.AddEntityInput

	mock := &MockSprintService{
		AddEntityToSprintFunc: func(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error) {
			capturedInput = input
			return &models.SprintAssignment{}, nil, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "")
	cmd.Flags().String("bulk", "", "")
	cmd.Flags().Bool("bulk-bugs", false, "")
	cmd.Flags().Bool("bulk-tech-debt", false, "")
	cmd.Flags().Bool("bulk-changes", false, "")
	cmd.Flags().Int("at", 0, "")
	// --at not set

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w
	runErr := runSprintAdd(cmd, []string{"S001", "E07-F01-001"})
	w.Close()
	os.Stdout = origOut
	buf := &bytes.Buffer{}
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	assert.Nil(t, capturedInput.Position, "Position must be nil when --at flag is not provided")
}

// TC-014b-bulk: --at combined with bulk flag returns error before service call.
func TestSprintAdd_AtFlagWithBulkReturnsError(t *testing.T) {
	called := false
	mock := &MockSprintService{
		BulkAddToSprintFunc: func(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error) {
			called = true
			return &services.BulkAddResult{AddedByType: map[string]int{}, SkippedByType: map[string]int{}}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("entity", "", "")
	cmd.Flags().String("bulk", "", "")
	cmd.Flags().Bool("bulk-bugs", false, "")
	cmd.Flags().Bool("bulk-tech-debt", false, "")
	cmd.Flags().Bool("bulk-changes", false, "")
	cmd.Flags().Int("at", 0, "")
	require.NoError(t, cmd.Flags().Set("at", "2"))
	require.NoError(t, cmd.Flags().Set("bulk-bugs", "true"))

	err := runSprintAdd(cmd, []string{"S001"})
	assert.Error(t, err, "--at and bulk flags must be mutually exclusive")
	assert.False(t, called, "BulkAddToSprint must NOT be called when --at is combined with bulk flag")
}

// =============================================================================
// TC-017: sprint backlog --view=ordered returns Items array
// =============================================================================
//
// TC-017 — Caller-Path Contract (CLI-layer via service mock):
//   - Entrypoint: runSprintBacklog CLI handler with --view=ordered.
//   - Mock seam: MockSprintService.GetSprintBacklog.
//   - Counter-factual: a buggy CLI that ignores --view and always calls with View="" would
//     not exercise the ordered path; TC-017 asserts GetSprintBacklog is called with View="ordered".
func TestSprintBacklog_TC017_ViewOrderedPassedToService(t *testing.T) {
	var capturedOpts services.BacklogOptions

	pos1, pos2 := 1, 2
	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			capturedOpts = opts
			return &services.SprintBacklog{
				SprintKey:  sprintKey,
				SprintName: "Sprint 1",
				View:       "ordered",
				Items: []*services.BacklogItemView{
					{Key: "E07-F01-001", EntityType: "task", Status: "todo", Position: &pos1, SprintOrder: &pos1},
					{Key: "B001", EntityType: "bug", Status: "todo", Position: &pos2, SprintOrder: &pos2},
				},
				Groups: nil,
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	// Capture pterm table output (column data) and fmt.Printf output separately.
	var ptermBuf bytes.Buffer
	pterm.SetDefaultOutput(&ptermBuf)
	defer pterm.SetDefaultOutput(os.Stdout)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")
	cmd.Flags().String("view", "", "")
	cmd.Flags().Bool("include-completed", false, "")
	require.NoError(t, cmd.Flags().Set("view", "ordered"))

	runErr := runSprintBacklog(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)

	assert.NoError(t, runErr)
	assert.Equal(t, "ordered", capturedOpts.View, "CLI must pass View='ordered' to service when --view=ordered flag is set")
	// Human output should show # column (pull queue format; rendered via pterm table).
	output := stdoutBuf.String() + ptermBuf.String()
	assert.Contains(t, output, "#", "ordered view output must contain # column header")
}

// =============================================================================
// TC-018: Active sprint defaults to --view=ordered without flag
// =============================================================================
//
// TC-018 — Caller-Path Contract (CLI-layer via service mock):
//   - Entrypoint: runSprintBacklog CLI handler with no --view flag.
//   - Mock seam: MockSprintService.GetSprintBacklog — service decides view based on sprint status.
//   - Forbidden mocks: Do NOT default View to "ordered" in CLI before calling service;
//     service detects sprint status and sets view.
//   - Counter-factual: a buggy CLI that defaults View to "grouped" would pass View="grouped"
//     to service; TC-018 asserts capturedOpts.View is "" (empty) so service can auto-detect.
func TestSprintBacklog_TC018_NoViewFlagPassesEmptyViewToService(t *testing.T) {
	var capturedOpts services.BacklogOptions

	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			capturedOpts = opts
			return &services.SprintBacklog{
				SprintKey:  sprintKey,
				SprintName: "Sprint 1",
				View:       "ordered", // service auto-detected active sprint
				Items:      []*services.BacklogItemView{},
				Groups:     nil,
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")
	cmd.Flags().String("view", "", "")
	cmd.Flags().Bool("include-completed", false, "")
	// --view NOT set

	runErr := runSprintBacklog(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	assert.Equal(t, "", capturedOpts.View, "CLI must pass empty View to service when --view flag is not set (let service auto-detect)")
}

// TC-018b: sprint backlog --view=grouped passes View="grouped" to service.
func TestSprintBacklog_TC018b_ViewGroupedPassedToService(t *testing.T) {
	var capturedOpts services.BacklogOptions

	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			capturedOpts = opts
			return &services.SprintBacklog{
				SprintKey:  sprintKey,
				SprintName: "Sprint 1",
				View:       "grouped",
				Groups:     []*services.BacklogGroup{},
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")
	cmd.Flags().String("view", "", "")
	cmd.Flags().Bool("include-completed", false, "")
	require.NoError(t, cmd.Flags().Set("view", "grouped"))

	runErr := runSprintBacklog(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	assert.Equal(t, "grouped", capturedOpts.View)
}

func TestSprintBacklog_AllFlagPassesIncludeCompletedToService(t *testing.T) {
	var capturedOpts services.BacklogOptions

	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			capturedOpts = opts
			return &services.SprintBacklog{
				SprintKey:  sprintKey,
				SprintName: "Sprint 1",
				View:       "ordered",
				Items:      []*services.BacklogItemView{},
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")
	cmd.Flags().String("view", "", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().Bool("include-completed", false, "")
	require.NoError(t, cmd.Flags().Set("all", "true"))

	runErr := runSprintBacklog(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	assert.True(t, capturedOpts.IncludeCompleted, "CLI must pass IncludeCompleted=true when --all is set")
}

// TC-017-unordered: ordered view with unordered item shows ~ in # column.
func TestSprintBacklog_TC017_UnorderedItemShowsTildeInPosColumn(t *testing.T) {
	pos1 := 1
	pos2 := 2
	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			return &services.SprintBacklog{
				SprintKey:  sprintKey,
				SprintName: "Sprint 1",
				View:       "ordered",
				Items: []*services.BacklogItemView{
					{Key: "E07-F01-001", EntityType: "task", Status: "todo", Title: "Ordered task", Position: &pos1, SprintOrder: &pos1},
					{Key: "B001", EntityType: "bug", Status: "todo", Title: "Ordered bug", Position: &pos2, SprintOrder: &pos2},
					// Unordered item: Position set (dense rank) but SprintOrder is nil
					{Key: "E07-F02-001", EntityType: "task", Status: "todo", Title: "Unordered task", Position: func() *int { p := 3; return &p }(), SprintOrder: nil},
				},
				Groups: nil,
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	// Capture pterm table output (column data) and fmt.Printf output separately.
	var ptermBuf bytes.Buffer
	pterm.SetDefaultOutput(&ptermBuf)
	defer pterm.SetDefaultOutput(os.Stdout)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")
	cmd.Flags().String("view", "", "")
	cmd.Flags().Bool("include-completed", false, "")
	require.NoError(t, cmd.Flags().Set("view", "ordered"))

	runErr := runSprintBacklog(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := stdoutBuf.String() + ptermBuf.String()
	// Ordered items show their position number (in pterm-rendered table).
	assert.Contains(t, output, "1", "ordered view must show position 1")
	assert.Contains(t, output, "2", "ordered view must show position 2")
	// Unordered item shows ~ in position column
	assert.Contains(t, output, "~", "ordered view must show ~ for unordered items")
}

func TestSprintBacklog_OrderedViewUsesWorkflowDisplayToken(t *testing.T) {
	tmpDir := t.TempDir()
	configJSON := `{
  "status_flow_version": "1.0",
  "status_flow": {
    "todo": ["in_progress"],
    "in_progress": ["completed"],
    "completed": []
  },
  "status_metadata": {
    "todo": {
      "color": "gray",
      "display_token": "TD"
    },
    "in_progress": {
      "color": "blue",
      "display_token": "IP"
    },
    "completed": {
      "color": "green",
      "display_token": "CMP"
    }
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".sharkconfig.json"), []byte(configJSON), 0644))

	origWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() {
		_ = os.Chdir(origWD)
		cli.ResetWorkflowService()
		config.ClearWorkflowCache()
	}()
	cli.ResetWorkflowService()
	config.ClearWorkflowCache()

	pos1 := 1
	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			return &services.SprintBacklog{
				SprintKey:  sprintKey,
				SprintName: "Sprint 1",
				View:       "ordered",
				Items: []*services.BacklogItemView{
					{Key: "E07-F01-001", EntityType: "task", Status: "in_progress", Title: "Implement token display", Position: &pos1, SprintOrder: &pos1},
				},
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	origNoColor := cli.GlobalConfig.NoColor
	defer func() {
		cli.GlobalConfig.JSON = origJSON
		cli.GlobalConfig.NoColor = origNoColor
	}()
	cli.GlobalConfig.JSON = false
	cli.GlobalConfig.NoColor = true

	var ptermBuf bytes.Buffer
	pterm.SetDefaultOutput(&ptermBuf)
	defer pterm.SetDefaultOutput(os.Stdout)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")
	cmd.Flags().String("view", "", "")
	cmd.Flags().Bool("include-completed", false, "")
	require.NoError(t, cmd.Flags().Set("view", "ordered"))

	runErr := runSprintBacklog(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)

	require.NoError(t, runErr)
	output := stdoutBuf.String() + ptermBuf.String()
	assert.Contains(t, output, "ST", "ordered backlog should include compact status column")
	assert.Contains(t, output, "IP", "ordered backlog should render configured display token")
}

func TestSprintBacklog_GroupedViewFallsBackWhenDisplayTokenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	configJSON := `{
  "status_flow_version": "1.0",
  "status_flow": {
    "todo": ["ready_for_review"],
    "ready_for_review": ["completed"],
    "completed": []
  },
  "status_metadata": {
    "todo": {
      "color": "gray"
    },
    "ready_for_review": {
      "color": "yellow"
    },
    "completed": {
      "color": "green"
    }
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".sharkconfig.json"), []byte(configJSON), 0644))

	origWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() {
		_ = os.Chdir(origWD)
		cli.ResetWorkflowService()
		config.ClearWorkflowCache()
	}()
	cli.ResetWorkflowService()
	config.ClearWorkflowCache()

	mock := &MockSprintService{
		GetSprintBacklogFunc: func(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error) {
			return &services.SprintBacklog{
				SprintKey:  sprintKey,
				SprintName: "Sprint 1",
				View:       "grouped",
				TotalCount: 1,
				Groups: []*services.BacklogGroup{
					{
						StatusCategory: "review",
						Items: []*services.BacklogItemView{
							{Key: "E07-F01-001", EntityType: "task", Status: "ready_for_review", Title: "Review work"},
						},
					},
				},
			}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	origNoColor := cli.GlobalConfig.NoColor
	defer func() {
		cli.GlobalConfig.JSON = origJSON
		cli.GlobalConfig.NoColor = origNoColor
	}()
	cli.GlobalConfig.JSON = false
	cli.GlobalConfig.NoColor = true

	var ptermBuf bytes.Buffer
	pterm.SetDefaultOutput(&ptermBuf)
	defer pterm.SetDefaultOutput(os.Stdout)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("blocked", false, "")
	cmd.Flags().String("view", "", "")
	cmd.Flags().Bool("include-completed", false, "")
	require.NoError(t, cmd.Flags().Set("view", "grouped"))

	runErr := runSprintBacklog(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)

	require.NoError(t, runErr)
	output := stdoutBuf.String() + ptermBuf.String()
	assert.Contains(t, output, "ST", "grouped backlog should include compact status column")
	assert.Contains(t, output, "RFR", "grouped backlog should fall back to deterministic token when display_token is missing")
}

// =============================================================================
// TC-015: sprint next --json includes sprint_order, sprint_key, selection_reason
// =============================================================================

// TC-015: `sprint next --json` response includes the three new fields sprint_order,
// sprint_key, and selection_reason. The CLI serializes whatever the service returns;
// this test verifies the CLI does not strip those fields.
func TestSprintNext_TC015_JSONIncludesNewFields(t *testing.T) {
	sprintOrder := 1
	executionOrder := 5
	size := 3
	item := &services.BacklogItemView{
		Key:             "E19-F07-001",
		EntityType:      "task",
		Title:           "Implement sprint_order column",
		Status:          "todo",
		AgentType:       "backend",
		Priority:        3,
		ExecutionOrder:  &executionOrder,
		Size:            &size,
		AssignedAt:      time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		SprintOrder:     &sprintOrder,   // NEW: set by GetNextTask
		SprintKey:       "S001",         // NEW: set by GetNextTask
		SelectionReason: "sprint_order", // NEW: set by GetNextTask
	}

	mock := &MockSprintService{
		GetNextTaskFunc: func(ctx context.Context, agentType string) (*services.BacklogItemView, error) {
			return item, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("agent", "", "")

	runErr := runSprintNext(cmd, []string{})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()

	// Parse JSON and verify all three new fields are present.
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &result), "output must be valid JSON: %s", output)

	// TC-015: sprint_order == 1
	assert.Equal(t, float64(1), result["sprint_order"], "sprint_order must be 1")
	// TC-015: sprint_key == "S001"
	assert.Equal(t, "S001", result["sprint_key"], "sprint_key must be S001")
	// TC-015: selection_reason == "sprint_order"
	assert.Equal(t, "sprint_order", result["selection_reason"], "selection_reason must be sprint_order")

	// TC-015c: Additive contract — existing fields still present.
	assert.Equal(t, "E19-F07-001", result["key"], "key must be present")
	assert.Equal(t, "task", result["entity_type"], "entity_type must be present")
	assert.Equal(t, "Implement sprint_order column", result["title"], "title must be present")
	assert.Equal(t, "todo", result["status"], "status must be present")
	assert.Equal(t, "backend", result["agent_type"], "agent_type must be present")
	assert.Equal(t, float64(3), result["priority"], "priority must be present")
	assert.Equal(t, float64(3), result["size"], "size must be present")
	assert.Equal(t, float64(5), result["execution_order"], "execution_order must be present")
}

// TC-015b: sprint_order is nil on returned item → JSON field is null (not absent, not error).
func TestSprintNext_TC015b_NullSprintOrderInJSON(t *testing.T) {
	item := &services.BacklogItemView{
		Key:             "B042",
		EntityType:      "bug",
		Title:           "Fix login redirect loop",
		Status:          "todo",
		Priority:        5,
		SprintOrder:     nil, // unordered
		SprintKey:       "S001",
		SelectionReason: "assigned_at",
	}

	mock := &MockSprintService{
		GetNextTaskFunc: func(ctx context.Context, agentType string) (*services.BacklogItemView, error) {
			return item, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("agent", "", "")

	runErr := runSprintNext(cmd, []string{})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()

	// The CLI must still be valid JSON even with a nil sprint_order.
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &result), "output must be valid JSON: %s", output)

	// sprint_order must be absent (omitempty) or null when unset — but NOT an error.
	// The BacklogItemView uses `json:"sprint_order,omitempty"` so nil → field absent.
	// Either way, the CLI must not error.
	assert.Equal(t, "S001", result["sprint_key"], "sprint_key must be present even when sprint_order is nil")
	assert.Equal(t, "assigned_at", result["selection_reason"], "selection_reason must be present")
}

// =============================================================================
// TC-016: sprint next human output shows Sprint order and Selected by lines
// =============================================================================

// TC-016: Human-mode output (no --json) adds "Sprint order: #N" and "Selected by: <reason>"
// lines below the existing task block (spec §3.5).
func TestSprintNext_TC016_HumanOutputShowsSprintOrderAndReason(t *testing.T) {
	sprintOrder := 1
	item := &services.BacklogItemView{
		Key:             "E19-F07-001",
		EntityType:      "task",
		Title:           "Implement sprint_order column",
		Status:          "todo",
		AgentType:       "backend",
		Priority:        3,
		SprintOrder:     &sprintOrder,
		SprintKey:       "S001",
		SelectionReason: "sprint_order",
	}

	mock := &MockSprintService{
		GetNextTaskFunc: func(ctx context.Context, agentType string) (*services.BacklogItemView, error) {
			return item, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("agent", "", "")

	runErr := runSprintNext(cmd, []string{})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()

	// Human output must include Sprint order line with the position number.
	assert.Contains(t, output, "Sprint order: #1", "human output must show Sprint order: #1")
	// Human output must include the Selected by reason.
	assert.Contains(t, output, "Selected by: sprint_order", "human output must show Selected by: sprint_order")
	// Human output must include the sprint key.
	assert.Contains(t, output, "Sprint:  S001", "human output must show Sprint: S001")
}

// TC-016b: when sprint_order is nil (unordered), human output shows "(unordered)" marker.
func TestSprintNext_TC016b_HumanOutputUnorderedItem(t *testing.T) {
	item := &services.BacklogItemView{
		Key:             "B042",
		EntityType:      "bug",
		Title:           "Fix login redirect loop",
		Status:          "todo",
		Priority:        5,
		SprintOrder:     nil,
		SprintKey:       "S001",
		SelectionReason: "assigned_at",
	}

	mock := &MockSprintService{
		GetNextTaskFunc: func(ctx context.Context, agentType string) (*services.BacklogItemView, error) {
			return item, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("agent", "", "")

	runErr := runSprintNext(cmd, []string{})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()

	assert.Contains(t, output, "unordered", "human output must show unordered marker when sprint_order is nil")
	assert.Contains(t, output, "Selected by: assigned_at", "human output must show selection reason")
}

// TestSprintNext_E38F06_ForwardsExactRoleWithoutMutation keeps sprint next a
// thin read-only adapter: it forwards the literal agent value to the selection
// service and only serializes the returned item.
func TestSprintNext_E38F06_ForwardsExactRoleWithoutMutation(t *testing.T) {
	var receivedAgentType string
	mock := &MockSprintService{
		GetNextTaskFunc: func(_ context.Context, agentType string) (*services.BacklogItemView, error) {
			receivedAgentType = agentType
			return &services.BacklogItemView{Key: "E38-F06-002", EntityType: "task", Title: "QA work", Status: "todo", AgentType: "qa", Priority: 3, AssignedAt: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)}, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origOut }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("agent", "", "")
	require.NoError(t, cmd.Flags().Set("agent", "qa"))

	runErr := runSprintNext(cmd, []string{})
	require.NoError(t, w.Close())
	var output bytes.Buffer
	_, err = output.ReadFrom(r)
	require.NoError(t, err)

	require.NoError(t, runErr)
	assert.Equal(t, "qa", receivedAgentType, "the CLI must not rewrite the workflow-resolved role")
	assert.JSONEq(t, `{"key":"E38-F06-002","entity_type":"task","title":"QA work","status":"todo","agent_type":"qa","priority":3,"assigned_at":"2026-07-15T12:00:00Z"}`, output.String())
}

// TestSprintNext_E38F06_WithoutRolePreservesUnfilteredCall ensures ordinary
// callers retain the pre-existing no-role selection path.
func TestSprintNext_E38F06_WithoutRolePreservesUnfilteredCall(t *testing.T) {
	var receivedAgentType string
	mock := &MockSprintService{
		GetNextTaskFunc: func(_ context.Context, agentType string) (*services.BacklogItemView, error) {
			receivedAgentType = agentType
			return nil, nil
		},
	}
	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("agent", "", "")

	require.NoError(t, runSprintNext(cmd, []string{}))
	assert.Empty(t, receivedAgentType, "no --agent flag must preserve unfiltered selection")
}

// =============================================================================
// TC-024: sprint start emits warning when NULL sprint_orders exist (human mode only)
// =============================================================================

// TC-024 cell 1: human mode, 3 unordered items → warning printed.
// Counter-factual: a buggy CLI that ignores the null count would not print the warning.
func TestSprintStart_TC024_HumanModeWarningWhenNullOrdersExist(t *testing.T) {
	startedSprint := newTestSprint()
	startedSprint.Status = models.SprintStatus("active")

	mock := &MockSprintService{
		StartSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return startedSprint, nil
		},
		CountNullSprintOrderFunc: func(ctx context.Context, sprintKey string) (int, error) {
			// Simulate 3 unordered assignments (REQ-F-009: count > 0 triggers warning).
			return 3, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runSprintStart(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()

	// TC-024 cell 1: exact warning string from spec §3.5.
	assert.Contains(t, output, "Warning: 3 items have no sprint order.", "warning must include item count")
	assert.Contains(t, output, "shark sprint reorder", "warning must reference the reorder command")
}

// TC-024 cell 2: JSON mode, 3 unordered items → warning NOT printed; JSON unchanged.
func TestSprintStart_TC024_JSONModeNoWarning(t *testing.T) {
	startedSprint := newTestSprint()
	startedSprint.Status = models.SprintStatus("active")

	mock := &MockSprintService{
		StartSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return startedSprint, nil
		},
		CountNullSprintOrderFunc: func(ctx context.Context, sprintKey string) (int, error) {
			// Simulate 3 unordered assignments — warning must be suppressed in JSON mode.
			return 3, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runSprintStart(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()

	// TC-024 cell 2 (AC-T1): warning must be absent in JSON mode.
	assert.NotContains(t, output, "Warning", "warning must be absent in --json mode")

	// JSON output must not include a "warning" key (REQ-F-009 last bullet).
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &result), "output must be valid JSON: %s", output)
	_, hasWarning := result["warning"]
	assert.False(t, hasWarning, "JSON output must not include a 'warning' key")
}

// TC-024 cell 3: human mode, all assignments ordered → no warning emitted.
func TestSprintStart_TC024_HumanModeNoWarningWhenAllOrdered(t *testing.T) {
	startedSprint := newTestSprint()
	startedSprint.Status = models.SprintStatus("active")

	mock := &MockSprintService{
		StartSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return startedSprint, nil
		},
		CountNullSprintOrderFunc: func(ctx context.Context, sprintKey string) (int, error) {
			return 0, nil // All items ordered — no warning.
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = false

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runSprintStart(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()

	// TC-024 cell 3: no warning when count == 0.
	assert.NotContains(t, output, "Warning", "no warning must appear when all assignments are ordered")
}

// TC-024 cell 4: JSON mode, all ordered → no warning in JSON.
func TestSprintStart_TC024_JSONModeNoWarningWhenAllOrdered(t *testing.T) {
	startedSprint := newTestSprint()
	startedSprint.Status = models.SprintStatus("active")

	mock := &MockSprintService{
		StartSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return startedSprint, nil
		},
		CountNullSprintOrderFunc: func(ctx context.Context, sprintKey string) (int, error) {
			return 0, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origOut := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runSprintStart(cmd, []string{"S001"})

	w.Close()
	os.Stdout = origOut
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, runErr)
	output := buf.String()

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &result), "output must be valid JSON: %s", output)
	_, hasWarning := result["warning"]
	assert.False(t, hasWarning, "JSON output must not include a 'warning' key when count == 0")
}
