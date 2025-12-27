package status

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

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

// Integration Tests for StatusService.GetDashboard

// TestGetDashboard_EmptyDatabase tests dashboard generation with no data
func TestGetDashboard_EmptyDatabase(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear all data to ensure empty state
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Request dashboard
	req := &StatusRequest{}
	dashboard, err := service.GetDashboard(ctx, req)

	// Verify no error
	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}

	// Verify all counts are zero
	if dashboard.Summary.Epics.Total != 0 {
		t.Errorf("Expected 0 total epics, got %d", dashboard.Summary.Epics.Total)
	}
	if dashboard.Summary.Epics.Active != 0 {
		t.Errorf("Expected 0 active epics, got %d", dashboard.Summary.Epics.Active)
	}
	if dashboard.Summary.Features.Total != 0 {
		t.Errorf("Expected 0 total features, got %d", dashboard.Summary.Features.Total)
	}
	if dashboard.Summary.Tasks.Total != 0 {
		t.Errorf("Expected 0 total tasks, got %d", dashboard.Summary.Tasks.Total)
	}
	if dashboard.Summary.OverallProgress != 0.0 {
		t.Errorf("Expected 0.0 overall progress, got %f", dashboard.Summary.OverallProgress)
	}
	if dashboard.Summary.BlockedCount != 0 {
		t.Errorf("Expected 0 blocked count, got %d", dashboard.Summary.BlockedCount)
	}

	// Verify empty lists
	if len(dashboard.Epics) != 0 {
		t.Errorf("Expected 0 epics in list, got %d", len(dashboard.Epics))
	}
	if len(dashboard.ActiveTasks) != 0 {
		t.Errorf("Expected 0 active tasks groups, got %d", len(dashboard.ActiveTasks))
	}
	if len(dashboard.BlockedTasks) != 0 {
		t.Errorf("Expected 0 blocked tasks, got %d", len(dashboard.BlockedTasks))
	}
}

// TestGetDashboard_WithData tests dashboard generation with real project data
func TestGetDashboard_WithData(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	service := NewStatusService(db)

	// Clear and seed test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks")
	_, _ = database.ExecContext(ctx, "DELETE FROM features")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics")

	// Create test epic
	result, _ := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E01', 'Test Epic', 'Test epic description', 'active', 'high')
	`)
	epicID, _ := result.LastInsertId()

	// Create test feature
	result, _ = database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E01-F01', 'Test Feature', 'Test feature description', 'active')
	`, epicID)
	featureID, _ := result.LastInsertId()

	// Create test tasks with various statuses
	_, _ = database.ExecContext(ctx, `
		INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
		VALUES
			(?, 'T-E01-F01-001', 'Completed Task', 'completed', 'backend', 5, '[]'),
			(?, 'T-E01-F01-002', 'In Progress Task', 'in_progress', 'backend', 5, '[]'),
			(?, 'T-E01-F01-003', 'Todo Task', 'todo', 'frontend', 5, '[]'),
			(?, 'T-E01-F01-004', 'Blocked Task', 'blocked', 'backend', 5, '[]')
	`, featureID, featureID, featureID, featureID)

	// Update blocked task with reason
	_, _ = database.ExecContext(ctx, `
		UPDATE tasks SET blocked_reason = 'Waiting on dependency'
		WHERE key = 'T-E01-F01-004'
	`)

	// Request dashboard
	req := &StatusRequest{}
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
	if epic.Key != "E01" {
		t.Errorf("Expected epic key E01, got %s", epic.Key)
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
	if blockedTask.Key != "T-E01-F01-004" {
		t.Errorf("Expected blocked task T-E01-F01-004, got %s", blockedTask.Key)
	}
	if blockedTask.BlockedReason == nil || *blockedTask.BlockedReason != "Waiting on dependency" {
		t.Error("Expected blocked reason to be set")
	}
}
