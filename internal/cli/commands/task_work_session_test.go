package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskContextSetAndGet tests setting and retrieving context data
func TestTaskContextSetAndGet(t *testing.T) {
	// Create test database
	dbWrapper := setupTestDB(t)
	defer dbWrapper.Close()

	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

	// Create test epic
	priority := models.PriorityHigh
	epic := &models.Epic{
		Key:              "E99",
		Title:            "Test Epic",
		Status:           models.EpicStatusActive,
		Priority:         priority,
		BusinessValue:    &priority,
		CustomFolderPath: nil,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	execOrder := 1
	feature := &models.Feature{
		EpicID:           epic.ID,
		Key:              "E99-F01",
		Title:            "Test Feature",
		Status:           models.FeatureStatusActive,
		CustomFolderPath: nil,
		ExecutionOrder:   &execOrder,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Create test task
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-001",
		Title:     "Test Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)

	// Test: Set current step
	contextData := &models.ContextData{}
	currentStep := "Implementing test functionality"
	if contextData.Progress == nil {
		contextData.Progress = &models.ProgressContext{}
	}
	contextData.Progress.CurrentStep = &currentStep

	jsonStr, err := contextData.ToJSON()
	require.NoError(t, err)

	task.ContextData = &jsonStr
	err = taskRepo.Update(ctx, task)
	require.NoError(t, err)

	// Retrieve and verify
	retrievedTask, err := taskRepo.GetByKey(ctx, "T-E99-F01-001")
	require.NoError(t, err)
	require.NotNil(t, retrievedTask.ContextData)

	parsedContext, err := models.FromJSON(*retrievedTask.ContextData)
	require.NoError(t, err)
	require.NotNil(t, parsedContext.Progress)
	require.NotNil(t, parsedContext.Progress.CurrentStep)
	assert.Equal(t, "Implementing test functionality", *parsedContext.Progress.CurrentStep)
}

// TestTaskContextCompletedSteps tests setting and retrieving completed steps
func TestTaskContextCompletedSteps(t *testing.T) {
	// Create test database
	dbWrapper := setupTestDB(t)
	defer dbWrapper.Close()

	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

	// Create test data
	priority := models.PriorityMedium
	epic := &models.Epic{
		Key:           "E99",
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      priority,
		BusinessValue: &priority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	execOrder := 1
	feature := &models.Feature{
		EpicID:         epic.ID,
		Key:            "E99-F01",
		Title:          "Test Feature",
		Status:         models.FeatureStatusActive,
		ExecutionOrder: &execOrder,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-002",
		Title:     "Test Task 2",
		Status:    models.TaskStatusInProgress,
		Priority:  7,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)

	// Set completed steps
	contextData := &models.ContextData{
		Progress: &models.ProgressContext{
			CompletedSteps: []string{"Step 1", "Step 2", "Step 3"},
			RemainingSteps: []string{"Step 4", "Step 5"},
		},
		OpenQuestions: []string{"Question 1?", "Question 2?"},
	}

	jsonStr, err := contextData.ToJSON()
	require.NoError(t, err)

	task.ContextData = &jsonStr
	err = taskRepo.Update(ctx, task)
	require.NoError(t, err)

	// Retrieve and verify
	retrievedTask, err := taskRepo.GetByKey(ctx, "T-E99-F01-002")
	require.NoError(t, err)

	parsedContext, err := models.FromJSON(*retrievedTask.ContextData)
	require.NoError(t, err)
	require.NotNil(t, parsedContext.Progress)
	assert.Len(t, parsedContext.Progress.CompletedSteps, 3)
	assert.Equal(t, "Step 1", parsedContext.Progress.CompletedSteps[0])
	assert.Len(t, parsedContext.Progress.RemainingSteps, 2)
	assert.Len(t, parsedContext.OpenQuestions, 2)
}

// TestWorkSessionCreationAndRetrieval tests work session CRUD operations
func TestWorkSessionCreationAndRetrieval(t *testing.T) {
	// Create test database
	dbWrapper := setupTestDB(t)
	defer dbWrapper.Close()

	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	sessionRepo := repository.NewWorkSessionRepository(dbWrapper)

	// Create test data
	priority := models.PriorityHigh
	epic := &models.Epic{
		Key:           "E99",
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      priority,
		BusinessValue: &priority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	execOrder := 1
	feature := &models.Feature{
		EpicID:         epic.ID,
		Key:            "E99-F01",
		Title:          "Test Feature",
		Status:         models.FeatureStatusActive,
		ExecutionOrder: &execOrder,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-003",
		Title:     "Test Task 3",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)

	// Create work session
	agentID := "test-agent"
	session := &models.WorkSession{
		TaskID:    task.ID,
		AgentID:   &agentID,
		StartedAt: time.Now(),
	}
	err = sessionRepo.Create(ctx, session)
	require.NoError(t, err)
	assert.Greater(t, session.ID, int64(0))

	// Retrieve session
	sessions, err := sessionRepo.GetByTaskID(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, task.ID, sessions[0].TaskID)
	assert.Equal(t, "test-agent", *sessions[0].AgentID)
	assert.True(t, sessions[0].IsActive())

	// End session
	outcome := models.SessionOutcomeCompleted
	notes := "Test session completed"
	err = sessionRepo.EndSession(ctx, session.ID, outcome, &notes)
	require.NoError(t, err)

	// Verify session ended
	sessions, err = sessionRepo.GetByTaskID(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.False(t, sessions[0].IsActive())
	assert.NotNil(t, sessions[0].Outcome)
	assert.Equal(t, models.SessionOutcomeCompleted, *sessions[0].Outcome)
	assert.NotNil(t, sessions[0].SessionNotes)
	assert.Equal(t, "Test session completed", *sessions[0].SessionNotes)
}

// TestWorkSessionStats tests session statistics calculation
func TestWorkSessionStats(t *testing.T) {
	// Create test database
	dbWrapper := setupTestDB(t)
	defer dbWrapper.Close()

	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	sessionRepo := repository.NewWorkSessionRepository(dbWrapper)

	// Create test data
	priority := models.PriorityMedium
	epic := &models.Epic{
		Key:           "E99",
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      priority,
		BusinessValue: &priority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	execOrder := 1
	feature := &models.Feature{
		EpicID:         epic.ID,
		Key:            "E99-F01",
		Title:          "Test Feature",
		Status:         models.FeatureStatusActive,
		ExecutionOrder: &execOrder,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-004",
		Title:     "Test Task 4",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)

	// Create multiple sessions
	agentID := "test-agent"

	// Session 1: 1 hour
	session1 := &models.WorkSession{
		TaskID:    task.ID,
		AgentID:   &agentID,
		StartedAt: time.Now().Add(-2 * time.Hour),
	}
	err = sessionRepo.Create(ctx, session1)
	require.NoError(t, err)

	outcome1 := models.SessionOutcomePaused
	err = sessionRepo.EndSession(ctx, session1.ID, outcome1, nil)
	require.NoError(t, err)

	// Update ended_at to be exactly 1 hour duration
	session1Retrieved, err := sessionRepo.GetByID(ctx, session1.ID)
	require.NoError(t, err)
	oneHourLater := session1Retrieved.StartedAt.Add(1 * time.Hour)
	session1Retrieved.EndedAt.Time = oneHourLater
	session1Retrieved.EndedAt.Valid = true
	err = sessionRepo.Update(ctx, session1Retrieved)
	require.NoError(t, err)

	// Session 2: 30 minutes
	session2 := &models.WorkSession{
		TaskID:    task.ID,
		AgentID:   &agentID,
		StartedAt: time.Now().Add(-1 * time.Hour),
	}
	err = sessionRepo.Create(ctx, session2)
	require.NoError(t, err)

	outcome2 := models.SessionOutcomeCompleted
	err = sessionRepo.EndSession(ctx, session2.ID, outcome2, nil)
	require.NoError(t, err)

	// Update ended_at to be exactly 30 minutes duration
	session2Retrieved, err := sessionRepo.GetByID(ctx, session2.ID)
	require.NoError(t, err)
	thirtyMinutesLater := session2Retrieved.StartedAt.Add(30 * time.Minute)
	session2Retrieved.EndedAt.Time = thirtyMinutesLater
	session2Retrieved.EndedAt.Valid = true
	err = sessionRepo.Update(ctx, session2Retrieved)
	require.NoError(t, err)

	// Get session stats
	stats, err := sessionRepo.GetSessionStatsByTaskID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalSessions)
	assert.Equal(t, 90*time.Minute, stats.TotalDuration)
	assert.Equal(t, 45*time.Minute, stats.AverageDuration)
	assert.False(t, stats.ActiveSession)
}

// TestSessionAnalyticsByEpic tests epic-level session analytics
func TestSessionAnalyticsByEpic(t *testing.T) {
	// Create test database
	dbWrapper := setupTestDB(t)
	defer dbWrapper.Close()

	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	sessionRepo := repository.NewWorkSessionRepository(dbWrapper)

	// Create test data
	priority := models.PriorityHigh
	epic := &models.Epic{
		Key:           "E99",
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      priority,
		BusinessValue: &priority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	execOrder := 1
	feature := &models.Feature{
		EpicID:         epic.ID,
		Key:            "E99-F01",
		Title:          "Test Feature",
		Status:         models.FeatureStatusActive,
		ExecutionOrder: &execOrder,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Create multiple tasks with sessions
	for i := 1; i <= 3; i++ {
		taskKey := fmt.Sprintf("T-E99-F01-%03d", i)
		task := &models.Task{
			FeatureID: feature.ID,
			Key:       taskKey,
			Title:     "Test Task",
			Status:    models.TaskStatusTodo,
			Priority:  5,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Create 2 sessions per task
		agentID := "test-agent"
		for j := 0; j < 2; j++ {
			session := &models.WorkSession{
				TaskID:    task.ID,
				AgentID:   &agentID,
				StartedAt: time.Now().Add(-time.Duration(j+1) * time.Hour),
			}
			err = sessionRepo.Create(ctx, session)
			require.NoError(t, err)

			if j == 0 {
				// First session paused
				outcome := models.SessionOutcomePaused
				err = sessionRepo.EndSession(ctx, session.ID, outcome, nil)
				require.NoError(t, err)
			} else {
				// Second session completed
				outcome := models.SessionOutcomeCompleted
				err = sessionRepo.EndSession(ctx, session.ID, outcome, nil)
				require.NoError(t, err)
			}
		}
	}

	// Get analytics
	analytics, err := sessionRepo.GetSessionAnalyticsByEpic(ctx, epic.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 6, analytics.TotalSessions)
	assert.Equal(t, 3, analytics.TasksWithSessions)
	assert.Equal(t, 3, analytics.TasksWithPauses)
	assert.Equal(t, 2.0, analytics.AverageSessionsPerTask)
	assert.Equal(t, 50.0, analytics.PauseRate)
}

// TestContextDataMerge tests merging context data
func TestContextDataMerge(t *testing.T) {
	// Initial context
	currentStep := "Step 1"
	ctx1 := &models.ContextData{
		Progress: &models.ProgressContext{
			CurrentStep:    &currentStep,
			CompletedSteps: []string{"Setup", "Config"},
		},
		OpenQuestions: []string{"Question 1?"},
	}

	// New context to merge
	newStep := "Step 2"
	ctx2 := &models.ContextData{
		Progress: &models.ProgressContext{
			CurrentStep:    &newStep,
			RemainingSteps: []string{"Step 3", "Step 4"},
		},
		ImplementationDecisions: map[string]string{
			"framework": "cobra",
		},
	}

	// Merge
	ctx1.Merge(ctx2)

	// Verify merged result
	assert.Equal(t, "Step 2", *ctx1.Progress.CurrentStep)
	assert.Len(t, ctx1.Progress.CompletedSteps, 2)
	assert.Len(t, ctx1.Progress.RemainingSteps, 2)
	assert.Len(t, ctx1.OpenQuestions, 1)
	assert.Len(t, ctx1.ImplementationDecisions, 1)
	assert.Equal(t, "cobra", ctx1.ImplementationDecisions["framework"])
}

// TestContextDataValidation tests context data validation
func TestContextDataValidation(t *testing.T) {
	// Test valid context data
	validCtx := &models.ContextData{
		Blockers: []models.BlockerContext{
			{
				Description:  "Waiting for API",
				BlockerType:  "external_dependency",
				BlockedSince: time.Now(),
			},
		},
		AcceptanceCriteriaStatus: []models.AcceptanceCriterionContext{
			{
				Criterion: "Test passes",
				Status:    "complete",
			},
		},
	}
	err := validCtx.Validate()
	assert.NoError(t, err)

	// Test invalid blocker (missing description)
	invalidBlockerCtx := &models.ContextData{
		Blockers: []models.BlockerContext{
			{
				BlockerType:  "external_dependency",
				BlockedSince: time.Now(),
			},
		},
	}
	err = invalidBlockerCtx.Validate()
	assert.Error(t, err)

	// Test invalid AC status
	invalidACCtx := &models.ContextData{
		AcceptanceCriteriaStatus: []models.AcceptanceCriterionContext{
			{
				Criterion: "Test",
				Status:    "invalid_status",
			},
		},
	}
	err = invalidACCtx.Validate()
	assert.Error(t, err)
}

// TestContextDataJSONRoundTrip tests JSON serialization/deserialization
func TestContextDataJSONRoundTrip(t *testing.T) {
	currentStep := "Implementation"
	original := &models.ContextData{
		Progress: &models.ProgressContext{
			CurrentStep:    &currentStep,
			CompletedSteps: []string{"Design", "Planning"},
			RemainingSteps: []string{"Testing", "Deployment"},
		},
		ImplementationDecisions: map[string]string{
			"database":  "sqlite",
			"framework": "cobra",
		},
		OpenQuestions: []string{"Performance requirements?"},
		RelatedTasks:  []string{"T-E99-F01-001"},
	}

	// Serialize
	jsonStr, err := original.ToJSON()
	require.NoError(t, err)

	// Deserialize
	parsed, err := models.FromJSON(jsonStr)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "Implementation", *parsed.Progress.CurrentStep)
	assert.Len(t, parsed.Progress.CompletedSteps, 2)
	assert.Len(t, parsed.Progress.RemainingSteps, 2)
	assert.Len(t, parsed.ImplementationDecisions, 2)
	assert.Equal(t, "sqlite", parsed.ImplementationDecisions["database"])
	assert.Len(t, parsed.OpenQuestions, 1)
	assert.Len(t, parsed.RelatedTasks, 1)
}

// TestResumeContextStructure tests the ResumeContext structure used by resume command
func TestResumeContextStructure(t *testing.T) {
	// Create sample data
	currentStep := "Testing"
	task := &models.Task{
		ID:       1,
		Key:      "T-E99-F01-001",
		Title:    "Test Task",
		Status:   models.TaskStatusInProgress,
		Priority: 5,
	}

	contextData := &models.ContextData{
		Progress: &models.ProgressContext{
			CurrentStep: &currentStep,
		},
	}

	session := &models.WorkSession{
		ID:        1,
		TaskID:    1,
		StartedAt: time.Now(),
	}

	stats := &repository.SessionStats{
		TotalSessions: 1,
		ActiveSession: true,
	}

	// Create resume context
	resumeCtx := &ResumeContext{
		Task:          task,
		ContextData:   contextData,
		WorkSessions:  []*models.WorkSession{session},
		SessionStats:  stats,
		ActiveSession: session,
	}

	// Verify structure
	assert.Equal(t, "T-E99-F01-001", resumeCtx.Task.Key)
	assert.NotNil(t, resumeCtx.ContextData.Progress)
	assert.Equal(t, "Testing", *resumeCtx.ContextData.Progress.CurrentStep)
	assert.Len(t, resumeCtx.WorkSessions, 1)
	assert.Equal(t, 1, resumeCtx.SessionStats.TotalSessions)
	assert.True(t, resumeCtx.SessionStats.ActiveSession)

	// Test JSON serialization
	jsonBytes, err := json.Marshal(resumeCtx)
	require.NoError(t, err)

	var deserialized ResumeContext
	err = json.Unmarshal(jsonBytes, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, "T-E99-F01-001", deserialized.Task.Key)
}
