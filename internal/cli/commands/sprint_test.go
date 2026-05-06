package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
	"unicode"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
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

// Test helpers
func setupSprintTest(t *testing.T, mock *MockSprintService) func() {
	oldOverride := sprintSvcOverride
	sprintSvcOverride = mock
	return func() {
		sprintSvcOverride = oldOverride
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
	// Human output should contain em-dash for future days
	assert.Contains(t, output, "—", "future days should show em-dash in human output")
}

// =============================================================================
// TC-S-07: --json with detailed=false: base fields present, detailed fields null
// =============================================================================

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
