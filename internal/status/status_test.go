package status

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// Unit Tests - Models and Validation

// TestStatusDashboardJSONMarshaling verifies that StatusDashboard marshals correctly to JSON
func TestStatusDashboardJSONMarshaling(t *testing.T) {
	dashboard := &StatusDashboard{
		Summary: &ProjectSummary{
			Epics:    &CountBreakdown{Total: 5, Active: 3},
			Features: &CountBreakdown{Total: 15, Active: 8},
			Tasks: &StatusBreakdown{
				Total:          50,
				Todo:           10,
				InProgress:     15,
				ReadyForReview: 5,
				Completed:      18,
				Blocked:        2,
			},
			OverallProgress: 36.0,
			BlockedCount:    2,
		},
		Epics:        []*EpicSummary{},
		ActiveTasks:  make(map[string][]*TaskInfo),
		BlockedTasks: []*BlockedTaskInfo{},
	}

	data, err := json.Marshal(dashboard)
	if err != nil {
		t.Fatalf("Failed to marshal StatusDashboard: %v", err)
	}

	// Verify we got valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Marshaled data is not valid JSON: %v", err)
	}

	// Verify key fields exist
	if _, ok := result["summary"]; !ok {
		t.Error("JSON missing 'summary' field")
	}
	if _, ok := result["epics"]; !ok {
		t.Error("JSON missing 'epics' field")
	}
	if _, ok := result["active_tasks"]; !ok {
		t.Error("JSON missing 'active_tasks' field")
	}
	if _, ok := result["blocked_tasks"]; !ok {
		t.Error("JSON missing 'blocked_tasks' field")
	}
}

// TestEpicSummaryJSONMarshaling verifies EpicSummary marshaling with all fields
func TestEpicSummaryJSONMarshaling(t *testing.T) {
	epic := &EpicSummary{
		Key:             "E05",
		Title:           "Task Management CLI",
		ProgressPercent: 45.5,
		Health:          "healthy",
		TasksTotal:      20,
		TasksCompleted:  9,
		TasksBlocked:    1,
		FeaturesTotal:   3,
		FeaturesActive:  2,
	}

	data, err := json.Marshal(epic)
	if err != nil {
		t.Fatalf("Failed to marshal EpicSummary: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Marshaled data is not valid JSON: %v", err)
	}

	// Verify snake_case field names
	if _, ok := result["progress_percent"]; !ok {
		t.Error("JSON missing 'progress_percent' field (should be snake_case)")
	}
	if _, ok := result["tasks_total"]; !ok {
		t.Error("JSON missing 'tasks_total' field")
	}
	if _, ok := result["tasks_completed"]; !ok {
		t.Error("JSON missing 'tasks_completed' field")
	}
}

// TestCompletionInfoWithNilFields verifies marshaling with optional nil fields
func TestCompletionInfoWithNilFields(t *testing.T) {
	completion := &CompletionInfo{
		Key:         "T-E05-F01-001",
		Title:       "Test Task",
		Feature:     "E05-F01",
		Epic:        "E05",
		CompletedAt: time.Now(),
		// CompletedAgo is nil
		// AgentType is nil
	}

	data, err := json.Marshal(completion)
	if err != nil {
		t.Fatalf("Failed to marshal CompletionInfo: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Marshaled data is not valid JSON: %v", err)
	}

	// Verify required fields exist
	if _, ok := result["key"]; !ok {
		t.Error("JSON missing 'key' field")
	}
	if _, ok := result["completed_at"]; !ok {
		t.Error("JSON missing 'completed_at' field")
	}

	// Optional fields should be omitted when nil (due to omitempty)
	// completed_ago and agent_type should not be present if nil
}

// TestStatusRequestValidate_ValidInputs verifies validation accepts valid inputs
func TestStatusRequestValidate_ValidInputs(t *testing.T) {
	testCases := []struct {
		name string
		req  *StatusRequest
	}{
		{
			name: "Empty request",
			req:  &StatusRequest{},
		},
		{
			name: "Valid epic key",
			req: &StatusRequest{
				EpicKey: "E05",
			},
		},
		{
			name: "Valid epic key with multiple digits",
			req: &StatusRequest{
				EpicKey: "E123",
			},
		},
		{
			name: "Valid timeframe 24h",
			req: &StatusRequest{
				RecentWindow: "24h",
			},
		},
		{
			name: "Valid timeframe 7d",
			req: &StatusRequest{
				RecentWindow: "7d",
			},
		},
		{
			name: "Valid timeframe 30d",
			req: &StatusRequest{
				RecentWindow: "30d",
			},
		},
		{
			name: "Valid epic and timeframe",
			req: &StatusRequest{
				EpicKey:      "E05",
				RecentWindow: "7d",
			},
		},
		{
			name: "Include archived",
			req: &StatusRequest{
				IncludeArchived: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if err != nil {
				t.Errorf("Expected validation to pass, but got error: %v", err)
			}
		})
	}
}

// TestStatusRequestValidate_InvalidInputs verifies validation rejects invalid inputs
func TestStatusRequestValidate_InvalidInputs(t *testing.T) {
	testCases := []struct {
		name        string
		req         *StatusRequest
		expectedErr string
	}{
		{
			name: "Invalid epic key format - lowercase",
			req: &StatusRequest{
				EpicKey: "e05",
			},
			expectedErr: "invalid epic key format",
		},
		{
			name: "Invalid epic key format - no number",
			req: &StatusRequest{
				EpicKey: "E",
			},
			expectedErr: "invalid epic key format",
		},
		{
			name: "Invalid epic key format - letters after number",
			req: &StatusRequest{
				EpicKey: "E05ABC",
			},
			expectedErr: "invalid epic key format",
		},
		{
			name: "Invalid timeframe",
			req: &StatusRequest{
				RecentWindow: "2h",
			},
			expectedErr: "invalid timeframe",
		},
		{
			name: "Invalid timeframe - wrong format",
			req: &StatusRequest{
				RecentWindow: "1week",
			},
			expectedErr: "invalid timeframe",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if err == nil {
				t.Error("Expected validation to fail, but it passed")
				return
			}
			// Check that error message contains expected substring
			if tc.expectedErr != "" {
				errMsg := err.Error()
				if len(errMsg) == 0 || !contains(errMsg, tc.expectedErr) {
					t.Errorf("Expected error to contain '%s', got: %v", tc.expectedErr, err)
				}
			}
		})
	}
}

// TestStatusErrorImplementsError verifies StatusError implements error interface
func TestStatusErrorImplementsError(t *testing.T) {
	err := NewStatusError("test error")

	// Verify it implements error interface
	var _ error = err

	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got: %s", err.Error())
	}

	// Verify it's a StatusError type
	statusErr, ok := err.(*StatusError)
	if !ok {
		t.Error("Expected error to be of type *StatusError")
	}

	if statusErr.Code != 1 {
		t.Errorf("Expected default error code 1, got: %d", statusErr.Code)
	}
}

// TestStatusErrorWithCode verifies custom error codes
func TestStatusErrorWithCode(t *testing.T) {
	err := NewStatusErrorWithCode("custom error", 2)

	statusErr, ok := err.(*StatusError)
	if !ok {
		t.Fatal("Expected error to be of type *StatusError")
	}

	if statusErr.Message != "custom error" {
		t.Errorf("Expected message 'custom error', got: %s", statusErr.Message)
	}

	if statusErr.Code != 2 {
		t.Errorf("Expected error code 2, got: %d", statusErr.Code)
	}
}

// TestAgentTypesOrderConstant verifies the AgentTypesOrder is defined
func TestAgentTypesOrderConstant(t *testing.T) {
	if len(AgentTypesOrder) == 0 {
		t.Error("AgentTypesOrder should not be empty")
	}

	expectedTypes := map[string]bool{
		"frontend":   true,
		"backend":    true,
		"api":        true,
		"testing":    true,
		"devops":     true,
		"general":    true,
		"unassigned": true,
	}

	for _, agentType := range AgentTypesOrder {
		if !expectedTypes[agentType] {
			t.Errorf("Unexpected agent type in AgentTypesOrder: %s", agentType)
		}
	}
}

// TestValidTimeframesConstant verifies ValidTimeframes is defined
func TestValidTimeframesConstant(t *testing.T) {
	if len(ValidTimeframes) == 0 {
		t.Error("ValidTimeframes should not be empty")
	}

	expectedTimeframes := []string{"24h", "1d", "48h", "7d", "30d", "90d"}
	for _, timeframe := range expectedTimeframes {
		if !ValidTimeframes[timeframe] {
			t.Errorf("Expected timeframe '%s' to be in ValidTimeframes", timeframe)
		}
	}
}

// TestDetermineEpicHealth_Healthy verifies healthy status conditions
func TestDetermineEpicHealth_Healthy(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	testCases := []struct {
		name         string
		progress     float64
		blockedCount int
		expected     string
	}{
		{"75% progress, no blocked", 75.0, 0, "healthy"},
		{"100% progress, no blocked", 100.0, 0, "healthy"},
		{"90% progress, no blocked", 90.0, 0, "healthy"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if ctx.Err() != nil {
				t.Fatal("Context cancelled")
			}
			health := service.determineEpicHealth(tc.progress, tc.blockedCount)
			if health != tc.expected {
				t.Errorf("Expected health '%s', got '%s'", tc.expected, health)
			}
		})
	}
}

// TestDetermineEpicHealth_Warning verifies warning status conditions
func TestDetermineEpicHealth_Warning(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	testCases := []struct {
		name         string
		progress     float64
		blockedCount int
		expected     string
	}{
		{"50% progress, no blocked", 50.0, 0, "warning"},
		{"74% progress, no blocked", 74.0, 0, "warning"},
		{"75% progress, 1 blocked", 75.0, 1, "warning"},
		{"80% progress, 2 blocked", 80.0, 2, "warning"},
		{"90% progress, 3 blocked", 90.0, 3, "warning"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if ctx.Err() != nil {
				t.Fatal("Context cancelled")
			}
			health := service.determineEpicHealth(tc.progress, tc.blockedCount)
			if health != tc.expected {
				t.Errorf("Expected health '%s', got '%s'", tc.expected, health)
			}
		})
	}
}

// TestDetermineEpicHealth_Critical verifies critical status conditions
func TestDetermineEpicHealth_Critical(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	testCases := []struct {
		name         string
		progress     float64
		blockedCount int
		expected     string
	}{
		{"0% progress, no blocked", 0.0, 0, "critical"},
		{"10% progress, no blocked", 10.0, 0, "critical"},
		{"24% progress, no blocked", 24.0, 0, "critical"},
		{"50% progress, 4 blocked", 50.0, 4, "critical"},
		{"90% progress, 5 blocked", 90.0, 5, "critical"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if ctx.Err() != nil {
				t.Fatal("Context cancelled")
			}
			health := service.determineEpicHealth(tc.progress, tc.blockedCount)
			if health != tc.expected {
				t.Errorf("Expected health '%s', got '%s'", tc.expected, health)
			}
		})
	}
}

// TestIsValidEpicKey_ValidKeys verifies valid epic key patterns
func TestIsValidEpicKey_ValidKeys(t *testing.T) {
	validKeys := []string{
		"E1",
		"E01",
		"E05",
		"E10",
		"E99",
		"E123",
		"E9999",
	}

	for _, key := range validKeys {
		t.Run(key, func(t *testing.T) {
			if !isValidEpicKey(key) {
				t.Errorf("Expected '%s' to be valid", key)
			}
		})
	}
}

// TestIsValidEpicKey_InvalidKeys verifies invalid epic key patterns are rejected
func TestIsValidEpicKey_InvalidKeys(t *testing.T) {
	invalidKeys := []string{
		"e05",     // lowercase
		"E",       // no number
		"E05F01",  // has feature part
		"E05-F01", // has dash
		"Epic05",  // wrong prefix
		"5",       // just number
		"E05ABC",  // letters after number
		"",        // empty
		"E 05",    // space
		"E-05",    // dash
	}

	for _, key := range invalidKeys {
		t.Run(key, func(t *testing.T) {
			if isValidEpicKey(key) {
				t.Errorf("Expected '%s' to be invalid", key)
			}
		})
	}
}

// TestGetDashboard_CancelledContext verifies context cancellation is handled
func TestGetDashboard_CancelledContext(t *testing.T) {
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &StatusRequest{}
	dashboard, err := service.GetDashboard(ctx, req)

	// Should return context error
	if err == nil {
		t.Error("Expected error from cancelled context, got nil")
	}
	if dashboard != nil {
		t.Error("Expected nil dashboard from cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestGetDashboard_InvalidRequest verifies validation is enforced
func TestGetDashboard_InvalidRequest(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	testCases := []struct {
		name string
		req  *StatusRequest
	}{
		{
			name: "Invalid epic key",
			req:  &StatusRequest{EpicKey: "invalid"},
		},
		{
			name: "Invalid timeframe",
			req:  &StatusRequest{RecentWindow: "5h"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dashboard, err := service.GetDashboard(ctx, tc.req)
			if err == nil {
				t.Error("Expected validation error, got nil")
			}
			if dashboard != nil {
				t.Error("Expected nil dashboard from invalid request")
			}
		})
	}
}

// Integration Tests for StatusService.GetDashboard
// Note: Remaining integration tests use isolated data to avoid parallel test conflicts

// TestGetDashboard_WithData tests dashboard generation with real project data
func TestGetDashboard_WithData(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Create unique isolated test data
	epicNum := 10 + int((time.Now().UnixNano()%80))
	epicKey := fmt.Sprintf("E%02d", epicNum)
	featureKey := fmt.Sprintf("%s-F01", epicKey)

	// Clean up any existing data for this epic
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?))", epicKey)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?)", epicKey)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create test epic
	result, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES (?, 'Test Epic', 'Test epic description', 'active', 'high')
	`, epicKey)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}
	epicID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get epic ID: %v", err)
	}

	// Create test feature
	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, ?, 'Test Feature', 'Test feature description', 'active')
	`, epicID, featureKey)
	featureID, _ := result.LastInsertId()

	// Create test tasks with various statuses
	_, _ = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES
			(?, ?, 'Completed Task', 'completed', 'backend', 5, '[]'),
			(?, ?, 'In Progress Task', 'in_progress', 'backend', 5, '[]'),
			(?, ?, 'Todo Task', 'todo', 'frontend', 5, '[]'),
			(?, ?, 'Blocked Task', 'blocked', 'backend', 5, '[]')
	`, featureID, fmt.Sprintf("T-%s-001", featureKey), featureID, fmt.Sprintf("T-%s-002", featureKey), featureID, fmt.Sprintf("T-%s-003", featureKey), featureID, fmt.Sprintf("T-%s-004", featureKey))

	// Update blocked task with reason
	_, _ = database.ExecContext(ctx, `
		UPDATE tasks SET blocked_reason = 'Waiting on dependency'
		WHERE key = ?
	`, fmt.Sprintf("T-%s-004", featureKey))

	// Request dashboard filtered by epic to see only our test data
	req := &StatusRequest{EpicKey: epicKey}
	dashboard, err := service.GetDashboard(ctx, req)

	// Verify no error
	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}

	// Verify epic counts
	if dashboard.Summary.Epics.Total != 1 {
		t.Errorf("Expected 1 total epic, got %d", dashboard.Summary.Epics.Total)
	}
	if dashboard.Summary.Epics.Active != 1 {
		t.Errorf("Expected 1 active epic, got %d", dashboard.Summary.Epics.Active)
	}

	// Verify feature counts
	if dashboard.Summary.Features.Total != 1 {
		t.Errorf("Expected 1 total feature, got %d", dashboard.Summary.Features.Total)
	}

	// Verify task counts
	if dashboard.Summary.Tasks.Total != 4 {
		t.Errorf("Expected 4 total tasks, got %d", dashboard.Summary.Tasks.Total)
	}
	if dashboard.Summary.Tasks.Completed != 1 {
		t.Errorf("Expected 1 completed task, got %d", dashboard.Summary.Tasks.Completed)
	}
	if dashboard.Summary.Tasks.InProgress != 1 {
		t.Errorf("Expected 1 in_progress task, got %d", dashboard.Summary.Tasks.InProgress)
	}
	if dashboard.Summary.Tasks.Todo != 1 {
		t.Errorf("Expected 1 todo task, got %d", dashboard.Summary.Tasks.Todo)
	}
	if dashboard.Summary.Tasks.Blocked != 1 {
		t.Errorf("Expected 1 blocked task, got %d", dashboard.Summary.Tasks.Blocked)
	}

	// Verify blocked count
	if dashboard.Summary.BlockedCount != 1 {
		t.Errorf("Expected 1 blocked task in summary, got %d", dashboard.Summary.BlockedCount)
	}

	// Verify overall progress (1 completed out of 4 = 25%)
	expectedProgress := 25.0
	if dashboard.Summary.OverallProgress != expectedProgress {
		t.Errorf("Expected %f%% overall progress, got %f%%", expectedProgress, dashboard.Summary.OverallProgress)
	}

	// Verify epic list
	if len(dashboard.Epics) != 1 {
		t.Fatalf("Expected 1 epic in list, got %d", len(dashboard.Epics))
	}
	epic := dashboard.Epics[0]
	if epic.Key != epicKey {
		t.Errorf("Expected epic key %s, got %s", epicKey, epic.Key)
	}
	if epic.TasksTotal != 4 {
		t.Errorf("Expected 4 total tasks in epic, got %d", epic.TasksTotal)
	}
	if epic.TasksCompleted != 1 {
		t.Errorf("Expected 1 completed task in epic, got %d", epic.TasksCompleted)
	}

	// Verify active tasks
	if len(dashboard.ActiveTasks) == 0 {
		t.Error("Expected active tasks, got none")
	}
	backendTasks, exists := dashboard.ActiveTasks["backend"]
	if !exists || len(backendTasks) != 1 {
		t.Errorf("Expected 1 backend active task, got %d", len(backendTasks))
	}

	// Verify blocked tasks
	if len(dashboard.BlockedTasks) != 1 {
		t.Fatalf("Expected 1 blocked task, got %d", len(dashboard.BlockedTasks))
	}
	blockedTask := dashboard.BlockedTasks[0]
	expectedBlockedKey := fmt.Sprintf("T-%s-004", featureKey)
	if blockedTask.Key != expectedBlockedKey {
		t.Errorf("Expected blocked task %s, got %s", expectedBlockedKey, blockedTask.Key)
	}
	if blockedTask.BlockedReason == nil || *blockedTask.BlockedReason != "Waiting on dependency" {
		t.Error("Expected blocked reason to be set")
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE feature_id = ?", featureID)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)
}

// TestGetDashboard_FilterByEpic tests filtering dashboard by epic key
func TestGetDashboard_FilterByEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear and seed test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create two epics
	result1, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Epic 1', 'First epic', 'active', 'high')
	`)
	epicID1, _ := result1.LastInsertId()

	result2, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E02', 'Epic 2', 'Second epic', 'active', 'medium')
	`)
	epicID2, _ := result2.LastInsertId()

	// Create features and tasks for each epic
	result, _ := database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Feature 1', 'First feature', 'active')
	`, epicID1)
	featureID1, _ := result.LastInsertId()

	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E02-F01', 'Feature 2', 'Second feature', 'active')
	`, epicID2)
	featureID2, _ := result.LastInsertId()

	_, _ = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES
			(?, 'T-E01-F01-001', 'Task in Epic 1', 'todo', 'backend', 5, '[]'),
			(?, 'T-E02-F01-001', 'Task in Epic 2', 'todo', 'backend', 5, '[]')
	`, featureID1, featureID2)

	// Request dashboard filtered by E01
	req := &StatusRequest{EpicKey: "E01"}
	dashboard, err := service.GetDashboard(ctx, req)

	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}

	// Should only see Epic 1 data
	if dashboard.Summary.Epics.Total != 1 {
		t.Errorf("Expected 1 epic when filtering by E01, got %d", dashboard.Summary.Epics.Total)
	}
	if dashboard.Summary.Tasks.Total != 1 {
		t.Errorf("Expected 1 task when filtering by E01, got %d", dashboard.Summary.Tasks.Total)
	}
	if len(dashboard.Epics) != 1 {
		t.Errorf("Expected 1 epic in list, got %d", len(dashboard.Epics))
	}
	if len(dashboard.Epics) > 0 && dashboard.Epics[0].Key != "E01" {
		t.Errorf("Expected epic E01, got %s", dashboard.Epics[0].Key)
	}

	// Verify filter is set in response
	if dashboard.Filter == nil {
		t.Error("Expected filter to be set")
	}
	if dashboard.Filter.EpicKey == nil || *dashboard.Filter.EpicKey != "E01" {
		t.Error("Expected filter epic key to be E01")
	}
}

// TestGetDashboard_MultipleAgentTypes tests grouping of active tasks by agent type
func TestGetDashboard_MultipleAgentTypes(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear and seed test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create test epic and feature
	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()

	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Test Feature', 'Test feature', 'active')
	`, epicID)
	featureID, _ := result.LastInsertId()

	// Create tasks with different agent types
	_, _ = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES
			(?, 'T-E01-F01-001', 'Backend Task 1', 'in_progress', 'backend', 5, '[]'),
			(?, 'T-E01-F01-002', 'Backend Task 2', 'in_progress', 'backend', 7, '[]'),
			(?, 'T-E01-F01-003', 'Frontend Task', 'in_progress', 'frontend', 5, '[]'),
			(?, 'T-E01-F01-004', 'DevOps Task', 'in_progress', 'devops', 3, '[]'),
			(?, 'T-E01-F01-005', 'Unassigned Task', 'in_progress', NULL, 5, '[]')
	`, featureID, featureID, featureID, featureID, featureID)

	// Request dashboard
	req := &StatusRequest{}
	dashboard, err := service.GetDashboard(ctx, req)

	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}

	// Verify task grouping
	if len(dashboard.ActiveTasks) != 4 {
		t.Errorf("Expected 4 agent type groups, got %d", len(dashboard.ActiveTasks))
	}

	// Verify backend tasks
	backendTasks, exists := dashboard.ActiveTasks["backend"]
	if !exists {
		t.Error("Expected backend group to exist")
	}
	if len(backendTasks) != 2 {
		t.Errorf("Expected 2 backend tasks, got %d", len(backendTasks))
	}

	// Verify frontend tasks
	frontendTasks, exists := dashboard.ActiveTasks["frontend"]
	if !exists {
		t.Error("Expected frontend group to exist")
	}
	if len(frontendTasks) != 1 {
		t.Errorf("Expected 1 frontend task, got %d", len(frontendTasks))
	}

	// Verify unassigned tasks
	unassignedTasks, exists := dashboard.ActiveTasks["unassigned"]
	if !exists {
		t.Error("Expected unassigned group to exist")
	}
	if len(unassignedTasks) != 1 {
		t.Errorf("Expected 1 unassigned task, got %d", len(unassignedTasks))
	}
}

// TestGetDashboard_NoActiveTasks tests dashboard when no tasks are in progress
func TestGetDashboard_NoActiveTasks(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear and seed test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create test epic and feature
	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()

	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Test Feature', 'Test feature', 'active')
	`, epicID)
	featureID, _ := result.LastInsertId()

	// Create only completed and todo tasks (no in_progress)
	_, _ = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES
			(?, 'T-E01-F01-001', 'Completed Task', 'completed', 'backend', 5, '[]'),
			(?, 'T-E01-F01-002', 'Todo Task', 'todo', 'backend', 5, '[]')
	`, featureID, featureID)

	// Request dashboard
	req := &StatusRequest{}
	dashboard, err := service.GetDashboard(ctx, req)

	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}

	// Verify no active tasks
	if len(dashboard.ActiveTasks) != 0 {
		t.Errorf("Expected 0 active task groups, got %d", len(dashboard.ActiveTasks))
	}
}

// TestGetProjectSummary_ZeroDivision tests that zero division is handled correctly
func TestGetProjectSummary_ZeroDivision(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear all data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create epic and feature but no tasks
	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()

	_, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Test Feature', 'Test feature', 'active')
	`, epicID)

	summary, err := service.getProjectSummary(ctx, "")
	if err != nil {
		t.Fatalf("getProjectSummary failed: %v", err)
	}

	// With 0 tasks, progress should be 0 (not NaN or panic)
	if summary.OverallProgress != 0.0 {
		t.Errorf("Expected 0.0 progress with no tasks, got %f", summary.OverallProgress)
	}
}

// TestGetRecentCompletions_NotImplemented verifies recent completions stub behavior
func TestGetRecentCompletions_NotImplemented(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Call with various parameters
	completions, err := service.getRecentCompletions(ctx, "", "24h")
	if err != nil {
		t.Errorf("getRecentCompletions should not error, got: %v", err)
	}

	// Currently returns empty list
	if len(completions) != 0 {
		t.Errorf("Expected empty list (not yet implemented), got %d items", len(completions))
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestNewStatusService_NilDatabase verifies service creation with nil database
func TestNewStatusService_NilDatabase(t *testing.T) {
	// This should not panic
	service := NewStatusService(nil)
	if service == nil {
		t.Error("Expected service instance, got nil")
	}
	if service.db != nil {
		t.Error("Expected nil db in service")
	}
}

// TestGetBlockedTasks_OrderedByPriority tests that blocked tasks are ordered correctly
func TestGetBlockedTasks_OrderedByPriority(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear and seed test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create test epic and feature
	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()

	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Test Feature', 'Test feature', 'active')
	`, epicID)
	featureID, _ := result.LastInsertId()

	// Create blocked tasks with different priorities
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)

	_, _ = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, priority, depends_on, blocked_at, blocked_reason)
		VALUES
			(?, 'T-E01-F01-001', 'Low Priority', 'blocked', 3, '[]', ?, 'Reason 1'),
			(?, 'T-E01-F01-002', 'High Priority', 'blocked', 9, '[]', ?, 'Reason 2'),
			(?, 'T-E01-F01-003', 'Medium Priority', 'blocked', 5, '[]', ?, 'Reason 3')
	`, featureID, earlier.Format(time.RFC3339), featureID, now.Format(time.RFC3339), featureID, earlier.Format(time.RFC3339))

	blockedTasks, err := service.getBlockedTasks(ctx, "")
	if err != nil {
		t.Fatalf("getBlockedTasks failed: %v", err)
	}

	// Should be ordered by priority DESC, then blocked_at DESC
	if len(blockedTasks) != 3 {
		t.Fatalf("Expected 3 blocked tasks, got %d", len(blockedTasks))
	}

	// First should be highest priority (9)
	if blockedTasks[0].Key != "T-E01-F01-002" {
		t.Errorf("Expected first task to be high priority, got %s", blockedTasks[0].Key)
	}
}

// TestGetActiveTasks_WithNullAgentType tests handling of NULL agent_type
func TestGetActiveTasks_WithNullAgentType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear and seed test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create test epic and feature
	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()

	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Test Feature', 'Test feature', 'active')
	`, epicID)
	featureID, _ := result.LastInsertId()

	// Create task with NULL agent_type
	_, err := database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES (?, 'T-E01-F01-001', 'Unassigned Task', 'in_progress', NULL, 5, '[]')
	`, featureID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	activeTasks, err := service.getActiveTasks(ctx, "")
	if err != nil {
		t.Fatalf("getActiveTasks failed: %v", err)
	}

	// Should be grouped under "unassigned"
	unassignedTasks, exists := activeTasks["unassigned"]
	if !exists {
		t.Fatal("Expected unassigned group to exist")
	}
	if len(unassignedTasks) != 1 {
		t.Fatalf("Expected 1 unassigned task, got %d", len(unassignedTasks))
	}

	task := unassignedTasks[0]
	if task.AgentType != nil {
		t.Errorf("Expected nil agent type for unassigned task, got %v", *task.AgentType)
	}
}

// TestGetBlockedTasks_WithNullBlockedReason tests handling of NULL blocked_reason
func TestGetBlockedTasks_WithNullBlockedReason(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear and seed test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create test epic and feature
	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()

	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Test Feature', 'Test feature', 'active')
	`, epicID)
	featureID, _ := result.LastInsertId()

	// Create blocked task with NULL reason
	_, _ = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, priority, depends_on, blocked_reason)
		VALUES (?, 'T-E01-F01-001', 'Blocked Task', 'blocked', 5, '[]', NULL)
	`, featureID)

	blockedTasks, err := service.getBlockedTasks(ctx, "")
	if err != nil {
		t.Fatalf("getBlockedTasks failed: %v", err)
	}

	if len(blockedTasks) != 1 {
		t.Fatalf("Expected 1 blocked task, got %d", len(blockedTasks))
	}

	task := blockedTasks[0]
	if task.BlockedReason != nil {
		t.Errorf("Expected nil blocked reason, got %v", *task.BlockedReason)
	}
}

// TestGetDashboard_SQLInjectionProtection tests that parameters are properly escaped
func TestGetDashboard_SQLInjectionProtection(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Attempt SQL injection via epic key (should fail validation first)
	req := &StatusRequest{
		EpicKey: "E01' OR '1'='1",
	}

	_, err := service.GetDashboard(ctx, req)
	// Should fail validation before reaching database
	if err == nil {
		t.Error("Expected validation error for SQL injection attempt")
	}
}

// TestGetDashboard_ConcurrentAccess tests concurrent dashboard requests
func TestGetDashboard_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear and seed minimal data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Test Epic', 'Test epic', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()

	_, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Test Feature', 'Test feature', 'active')
	`, epicID)

	// Run multiple concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := &StatusRequest{}
			_, err := service.GetDashboard(ctx, req)
			if err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestGetEpics_NullSafeAggregation tests that NULL values in aggregations don't cause issues
func TestGetEpics_NullSafeAggregation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Use unique epic key to avoid parallel test conflicts
	epicKey := fmt.Sprintf("E-TEST-%d", time.Now().UnixNano()%1000000)

	// Clean up only this test's data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic with no features or tasks
	_, _ = database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES (?, 'Empty Epic', 'Epic with no features', 'active', 'high')
	`, epicKey)

	epics, err := service.getEpics(ctx, epicKey)
	if err != nil {
		t.Fatalf("getEpics failed: %v", err)
	}

	if len(epics) != 1 {
		t.Fatalf("Expected 1 epic, got %d", len(epics))
	}

	epic := epics[0]
	// Should have zero counts, not fail
	if epic.FeaturesTotal != 0 {
		t.Errorf("Expected 0 features, got %d", epic.FeaturesTotal)
	}
	if epic.TasksTotal != 0 {
		t.Errorf("Expected 0 tasks, got %d", epic.TasksTotal)
	}
	if epic.ProgressPercent != 0.0 {
		t.Errorf("Expected 0%% progress, got %f%%", epic.ProgressPercent)
	}
}

// TestGetDashboard_RowScanErrors tests handling of database scan errors
func TestGetDashboard_InvalidDatabase(t *testing.T) {
	t.Skip("Skipping test that closes database - affects other tests")
	// NOTE: Closing the database would affect other tests in the suite
	// This test demonstrates the concept but should not actually close the shared test DB
}

// TestStatusRequest_EmptyEpicKey tests validation with empty string (should pass)
func TestStatusRequest_EmptyEpicKey(t *testing.T) {
	req := &StatusRequest{
		EpicKey: "",
	}

	err := req.Validate()
	if err != nil {
		t.Errorf("Expected empty epic key to be valid, got error: %v", err)
	}
}

// TestGetActiveTasks_EmptyAgentTypeString tests tasks with empty string agent_type
func TestGetActiveTasks_EmptyAgentTypeString(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Use unique keys to avoid parallel test conflicts
	timestamp := time.Now().UnixNano() % 1000000
	epicKey := fmt.Sprintf("E-TEST-%d", timestamp)
	featureKey := fmt.Sprintf("%s-F01", epicKey)
	taskKey := fmt.Sprintf("T-%s-001", featureKey)

	// Clean up only this test's data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", taskKey)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES (?, 'Test Epic', 'Test epic', 'active', 'high')
	`, epicKey)
	epicID, _ := result.LastInsertId()

	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, ?, 'Test Feature', 'Test feature', 'active')
	`, epicID, featureKey)
	featureID, _ := result.LastInsertId()

	// Create task with empty string agent_type
	_, _ = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES (?, ?, 'Empty Agent Task', 'in_progress', '', 5, '[]')
	`, featureID, taskKey)

	activeTasks, err := service.getActiveTasks(ctx, "")
	if err != nil {
		t.Fatalf("getActiveTasks failed: %v", err)
	}

	// Empty string should be treated as unassigned
	unassignedTasks, exists := activeTasks["unassigned"]
	if !exists {
		t.Fatal("Expected unassigned group for empty agent_type")
	}
	if len(unassignedTasks) != 1 {
		t.Errorf("Expected 1 unassigned task, got %d", len(unassignedTasks))
	}
}
