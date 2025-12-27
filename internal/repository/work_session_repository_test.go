package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestCreateWorkSession tests creating a work session
func TestCreateWorkSession(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task to add session to
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing active sessions for this task
	_, _ = database.Exec("UPDATE work_sessions SET ended_at = ?, outcome = 'completed' WHERE task_id = ? AND ended_at IS NULL", time.Now(), taskID)

	// Create a work session
	agentID := "test-agent-001"
	session := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}

	err = sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create work session: %v", err)
	}

	if session.ID == 0 {
		t.Error("Expected session ID to be set after creation")
	}

	// Verify session was created in database
	var count int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM work_sessions WHERE id = ?", session.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query work sessions: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session in database, got %d", count)
	}
}

// TestCreateWorkSessionDuplicateActive tests that creating a second active session fails
func TestCreateWorkSessionDuplicateActive(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing active sessions for this task
	_, _ = database.Exec("UPDATE work_sessions SET ended_at = ?, outcome = 'completed' WHERE task_id = ? AND ended_at IS NULL", time.Now(), taskID)

	// Create first active session
	agentID := "test-agent-001"
	session1 := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}

	err = sessionRepo.Create(ctx, session1)
	if err != nil {
		t.Fatalf("Failed to create first work session: %v", err)
	}

	// Try to create second active session (should fail)
	session2 := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}

	err = sessionRepo.Create(ctx, session2)
	if err == nil {
		t.Error("Expected error when creating duplicate active session, got nil")
	}
}

// TestCreateWorkSessionValidation tests validation during session creation
func TestCreateWorkSessionValidation(t *testing.T) {
	outcome := models.SessionOutcomeCompleted
	invalidOutcome := models.SessionOutcome("invalid")

	tests := []struct {
		name        string
		session     *models.WorkSession
		expectError bool
	}{
		{
			name: "invalid task ID",
			session: &models.WorkSession{
				TaskID:    0,
				StartedAt: time.Now(),
			},
			expectError: true,
		},
		{
			name: "zero timestamp",
			session: &models.WorkSession{
				TaskID:    1,
				StartedAt: time.Time{},
			},
			expectError: true,
		},
		{
			name: "invalid outcome",
			session: &models.WorkSession{
				TaskID:    1,
				StartedAt: time.Now(),
				Outcome:   &invalidOutcome,
			},
			expectError: true,
		},
		{
			name: "valid session with outcome",
			session: &models.WorkSession{
				TaskID:    1,
				StartedAt: time.Now(),
				Outcome:   &outcome,
			},
			expectError: false, // Validation should pass (DB constraint might fail if task doesn't exist)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got %v", err)
			}
		})
	}
}

// TestGetWorkSessionByID tests retrieving a work session by ID
func TestGetWorkSessionByID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing active sessions for this task
	_, _ = database.Exec("UPDATE work_sessions SET ended_at = ?, outcome = 'completed' WHERE task_id = ? AND ended_at IS NULL", time.Now(), taskID)

	// Create a session
	agentID := "test-agent-001"
	session := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}
	err = sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Retrieve the session
	retrieved, err := sessionRepo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get session by ID: %v", err)
	}

	// Verify
	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %d, got %d", session.ID, retrieved.ID)
	}
	if retrieved.TaskID != taskID {
		t.Errorf("Expected TaskID %d, got %d", taskID, retrieved.TaskID)
	}
	if retrieved.AgentID == nil || *retrieved.AgentID != agentID {
		t.Errorf("Expected AgentID %s, got %v", agentID, retrieved.AgentID)
	}
	if !retrieved.IsActive() {
		t.Error("Expected session to be active")
	}
}

// TestGetWorkSessionByTaskID tests retrieving all sessions for a task
func TestGetWorkSessionByTaskID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing active sessions for this task
	_, _ = database.Exec("DELETE FROM work_sessions WHERE task_id = ?", taskID)

	// Create multiple sessions
	agentID := "test-agent-001"
	outcome := models.SessionOutcomePaused

	// First session (ended)
	session1 := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now().Add(-2 * time.Hour),
		EndedAt:   sql.NullTime{Time: time.Now().Add(-1 * time.Hour), Valid: true},
		Outcome:   &outcome,
	}
	_, err = database.Exec("INSERT INTO work_sessions (task_id, agent_id, started_at, ended_at, outcome) VALUES (?, ?, ?, ?, ?)",
		session1.TaskID, session1.AgentID, session1.StartedAt, session1.EndedAt.Time, session1.Outcome)
	if err != nil {
		t.Fatalf("Failed to insert first session: %v", err)
	}

	// Second session (active)
	session2 := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}
	err = sessionRepo.Create(ctx, session2)
	if err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}

	// Retrieve all sessions
	sessions, err := sessionRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		t.Fatalf("Failed to get sessions by task ID: %v", err)
	}

	if len(sessions) < 2 {
		t.Errorf("Expected at least 2 sessions, got %d", len(sessions))
	}

	// Verify sessions are ordered by started_at DESC (most recent first)
	if len(sessions) >= 2 {
		if sessions[0].StartedAt.Before(sessions[1].StartedAt) {
			t.Error("Expected sessions to be ordered by started_at DESC")
		}
	}
}

// TestGetActiveSessionByTaskID tests retrieving the active session for a task
func TestGetActiveSessionByTaskID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing sessions for this task
	_, _ = database.ExecContext(ctx, "DELETE FROM work_sessions WHERE task_id = ?", taskID)

	// Create an active session
	agentID := "test-agent-001"
	session := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}
	err = sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Retrieve active session
	active, err := sessionRepo.GetActiveSessionByTaskID(ctx, taskID)
	if err != nil {
		t.Fatalf("Failed to get active session: %v", err)
	}

	if active.ID != session.ID {
		t.Errorf("Expected session ID %d, got %d", session.ID, active.ID)
	}
	if !active.IsActive() {
		t.Error("Expected session to be active")
	}
}

// TestGetActiveSessionByTaskIDNoActive tests when there's no active session
func TestGetActiveSessionByTaskIDNoActive(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-002'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing sessions for this task
	_, _ = database.Exec("DELETE FROM work_sessions WHERE task_id = ?", taskID)

	// Try to get active session (should return sql.ErrNoRows)
	_, err = sessionRepo.GetActiveSessionByTaskID(ctx, taskID)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

// TestEndSession tests ending an active work session
func TestEndSession(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-002'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing active sessions for this task
	_, _ = database.Exec("UPDATE work_sessions SET ended_at = ?, outcome = 'completed' WHERE task_id = ? AND ended_at IS NULL", time.Now(), taskID)

	// Create an active session
	agentID := "test-agent-001"
	session := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now().Add(-1 * time.Hour),
	}
	err = sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// End the session
	notes := "Completed implementation"
	err = sessionRepo.EndSession(ctx, session.ID, models.SessionOutcomeCompleted, &notes)
	if err != nil {
		t.Fatalf("Failed to end session: %v", err)
	}

	// Retrieve and verify
	updated, err := sessionRepo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	if updated.IsActive() {
		t.Error("Expected session to be ended")
	}
	if updated.Outcome == nil || *updated.Outcome != models.SessionOutcomeCompleted {
		t.Errorf("Expected outcome to be completed, got %v", updated.Outcome)
	}
	if updated.SessionNotes == nil || *updated.SessionNotes != notes {
		t.Errorf("Expected session notes %q, got %v", notes, updated.SessionNotes)
	}

	// Verify duration is positive
	duration := updated.Duration()
	if duration <= 0 {
		t.Errorf("Expected positive duration, got %v", duration)
	}
}

// TestEndSessionAlreadyEnded tests ending an already-ended session
func TestEndSessionAlreadyEnded(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	// Clean up any existing test data first
	_, _ = database.Exec("DELETE FROM epics WHERE key = 'E97'")

	// Create our own test data to avoid test isolation issues
	epic := &models.Epic{
		Key:      "E97",
		Title:    "Test Epic for End Session",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E97-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E97-F01-001",
		Title:     "Test Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	taskID := task.ID

	// Clean up any existing sessions for this task
	_, _ = database.Exec("DELETE FROM work_sessions WHERE task_id = ?", taskID)

	// Create and end a session
	agentID := "test-agent-001"
	session := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now().Add(-1 * time.Hour),
	}
	err = sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err = sessionRepo.EndSession(ctx, session.ID, models.SessionOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("Failed to end session first time: %v", err)
	}

	// Try to end it again (should fail)
	err = sessionRepo.EndSession(ctx, session.ID, models.SessionOutcomePaused, nil)
	if err == nil {
		t.Error("Expected error when ending already-ended session, got nil")
	}
}

// TestUpdateWorkSession tests updating a work session
func TestUpdateWorkSession(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing active sessions for this task
	_, _ = database.Exec("UPDATE work_sessions SET ended_at = ?, outcome = 'completed' WHERE task_id = ? AND ended_at IS NULL", time.Now(), taskID)

	// Create a session
	agentID := "test-agent-001"
	session := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}
	err = sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Update the session
	newAgentID := "test-agent-002"
	session.AgentID = &newAgentID
	outcome := models.SessionOutcomePaused
	session.Outcome = &outcome

	err = sessionRepo.Update(ctx, session)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Retrieve and verify
	updated, err := sessionRepo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	if updated.AgentID == nil || *updated.AgentID != newAgentID {
		t.Errorf("Expected agent ID %s, got %v", newAgentID, updated.AgentID)
	}
	if updated.Outcome == nil || *updated.Outcome != outcome {
		t.Errorf("Expected outcome %s, got %v", outcome, updated.Outcome)
	}
}

// TestDeleteWorkSession tests deleting a work session
func TestDeleteWorkSession(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing active sessions for this task
	_, _ = database.Exec("UPDATE work_sessions SET ended_at = ?, outcome = 'completed' WHERE task_id = ? AND ended_at IS NULL", time.Now(), taskID)

	// Create a session
	agentID := "test-agent-001"
	session := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}
	err = sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Delete the session
	err = sessionRepo.Delete(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify it's gone
	_, err = sessionRepo.GetByID(ctx, session.ID)
	if err == nil {
		t.Error("Expected error when getting deleted session, got nil")
	}
}

// TestGetSessionStatsByTaskID tests session statistics calculation
func TestGetSessionStatsByTaskID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	agentID := "test-agent-001"
	outcome := models.SessionOutcomeCompleted

	// Create multiple ended sessions
	session1 := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now().Add(-3 * time.Hour),
		EndedAt:   sql.NullTime{Time: time.Now().Add(-2 * time.Hour), Valid: true},
		Outcome:   &outcome,
	}
	_, _ = database.Exec("INSERT INTO work_sessions (task_id, agent_id, started_at, ended_at, outcome) VALUES (?, ?, ?, ?, ?)",
		session1.TaskID, session1.AgentID, session1.StartedAt, session1.EndedAt.Time, session1.Outcome)

	session2 := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now().Add(-2 * time.Hour),
		EndedAt:   sql.NullTime{Time: time.Now().Add(-1 * time.Hour), Valid: true},
		Outcome:   &outcome,
	}
	_, _ = database.Exec("INSERT INTO work_sessions (task_id, agent_id, started_at, ended_at, outcome) VALUES (?, ?, ?, ?, ?)",
		session2.TaskID, session2.AgentID, session2.StartedAt, session2.EndedAt.Time, session2.Outcome)

	// Create an active session
	session3 := &models.WorkSession{
		TaskID:    taskID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}
	err = sessionRepo.Create(ctx, session3)
	if err != nil {
		t.Fatalf("Failed to create active session: %v", err)
	}

	// Get stats
	stats, err := sessionRepo.GetSessionStatsByTaskID(ctx, taskID)
	if err != nil {
		t.Fatalf("Failed to get session stats: %v", err)
	}

	if stats.TotalSessions < 3 {
		t.Errorf("Expected at least 3 sessions, got %d", stats.TotalSessions)
	}
	if !stats.ActiveSession {
		t.Error("Expected active session flag to be true")
	}
	if stats.TotalDuration <= 0 {
		t.Error("Expected positive total duration")
	}
	if stats.AverageDuration <= 0 {
		t.Error("Expected positive average duration")
	}
}

// TestGetSessionAnalyticsByEpic tests epic-level session analytics
func TestGetSessionAnalyticsByEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	sessionRepo := NewWorkSessionRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	// Clean up any existing test data first to avoid UNIQUE constraint errors
	_, _ = database.Exec("DELETE FROM epics WHERE key = 'E98'")

	// Create our own test data to avoid test isolation issues
	epic := &models.Epic{
		Key:      "E98",
		Title:    "Test Epic for Analytics",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E98-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E98-F01-001",
		Title:     "Test Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	taskID := task.ID
	epicID := epic.ID

	// Clean up any existing sessions for this task
	_, _ = database.Exec("DELETE FROM work_sessions WHERE task_id = ?", taskID)

	// Create sessions
	agentID := "test-agent-001"
	outcome := models.SessionOutcomePaused

	result, err := database.Exec("INSERT INTO work_sessions (task_id, agent_id, started_at, ended_at, outcome) VALUES (?, ?, ?, ?, ?)",
		taskID, agentID, time.Now().Add(-1*time.Hour), time.Now(), outcome)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected != 1 {
		t.Fatalf("Expected 1 row affected, got %d", rowsAffected)
	}

	// Verify the session was actually inserted
	var count int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM work_sessions WHERE task_id = ?", taskID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}
	if count == 0 {
		t.Fatal("Session was not inserted")
	}

	// Get analytics
	analytics, err := sessionRepo.GetSessionAnalyticsByEpic(ctx, epicID, nil)
	if err != nil {
		t.Fatalf("Failed to get epic analytics: %v", err)
	}

	if analytics.TotalSessions < 1 {
		t.Errorf("Expected at least 1 session, got %d", analytics.TotalSessions)
	}
	if analytics.TasksWithSessions < 1 {
		t.Errorf("Expected at least 1 task with sessions, got %d", analytics.TasksWithSessions)
	}
	if analytics.TasksWithPauses < 1 {
		t.Errorf("Expected at least 1 task with pauses, got %d", analytics.TasksWithPauses)
	}
	if analytics.PauseRate <= 0 {
		t.Error("Expected positive pause rate")
	}
}
