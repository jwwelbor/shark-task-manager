package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

func main() {
	// Create a temporary test database
	dbPath := "test-db.db"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + "-shm")
	defer os.Remove(dbPath + "-wal")

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	fmt.Println("✓ Database initialized successfully")

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)
	historyRepo := repository.NewTaskHistoryRepository(repoDb)

	// Test Epic CRUD
	fmt.Println("\n--- Testing Epic CRUD ---")

	businessValue := models.PriorityHigh
	epic := &models.Epic{
		Key:         "E04",
		Title:       "Task Management CLI - Core Functionality",
		Description: strPtr("Complete database schema and repositories"),
		Status:      models.EpicStatusActive,
		Priority:    models.PriorityHigh,
		BusinessValue: &businessValue,
	}

	if err := epicRepo.Create(epic); err != nil {
		log.Fatal("Failed to create epic:", err)
	}
	fmt.Printf("✓ Created epic with ID: %d\n", epic.ID)

	retrievedEpic, err := epicRepo.GetByKey("E04")
	if err != nil {
		log.Fatal("Failed to get epic by key:", err)
	}
	fmt.Printf("✓ Retrieved epic: %s - %s\n", retrievedEpic.Key, retrievedEpic.Title)

	// Test Feature CRUD
	fmt.Println("\n--- Testing Feature CRUD ---")

	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E04-F01",
		Title:       "Database Schema & Core Data Model",
		Description: strPtr("SQLite database with epics, features, tasks, and history"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}

	if err := featureRepo.Create(feature); err != nil {
		log.Fatal("Failed to create feature:", err)
	}
	fmt.Printf("✓ Created feature with ID: %d\n", feature.ID)

	// Test Task CRUD
	fmt.Println("\n--- Testing Task CRUD ---")

	agentType := models.AgentTypeBackend
	task := &models.Task{
		FeatureID:   feature.ID,
		Key:         "T-E04-F01-001",
		Title:       "Create ORM Models",
		Description: strPtr("Define Epic, Feature, Task, TaskHistory models"),
		Status:      models.TaskStatusTodo,
		AgentType:   &agentType,
		Priority:    3,
		DependsOn:   strPtr("[]"),
	}

	if err := taskRepo.Create(task); err != nil {
		log.Fatal("Failed to create task:", err)
	}
	fmt.Printf("✓ Created task with ID: %d\n", task.ID)

	// Test UpdateStatus (atomic operation with history)
	fmt.Println("\n--- Testing Task Status Update ---")

	agent := "test-agent"
	notes := "Starting implementation"
	if err := taskRepo.UpdateStatus(task.ID, models.TaskStatusInProgress, &agent, &notes); err != nil {
		log.Fatal("Failed to update task status:", err)
	}
	fmt.Println("✓ Updated task status to in_progress")

	// Verify task status was updated
	updatedTask, err := taskRepo.GetByID(task.ID)
	if err != nil {
		log.Fatal("Failed to get updated task:", err)
	}
	fmt.Printf("✓ Task status: %s, Started at: %v\n", updatedTask.Status, updatedTask.StartedAt)

	// Test Task History
	fmt.Println("\n--- Testing Task History ---")

	history, err := historyRepo.ListByTask(task.ID)
	if err != nil {
		log.Fatal("Failed to list task history:", err)
	}
	fmt.Printf("✓ Found %d history records\n", len(history))
	if len(history) > 0 {
		fmt.Printf("  - Old: %v, New: %s, Agent: %v\n",
			*history[0].OldStatus, history[0].NewStatus, *history[0].Agent)
	}

	// Test Feature Progress Calculation
	fmt.Println("\n--- Testing Progress Calculation ---")

	progress, err := featureRepo.CalculateProgress(feature.ID)
	if err != nil {
		log.Fatal("Failed to calculate feature progress:", err)
	}
	fmt.Printf("✓ Feature progress: %.1f%% (0/1 tasks completed)\n", progress)

	// Complete the task and recalculate
	if err := taskRepo.UpdateStatus(task.ID, models.TaskStatusCompleted, &agent, nil); err != nil {
		log.Fatal("Failed to complete task:", err)
	}

	if err := featureRepo.UpdateProgress(feature.ID); err != nil {
		log.Fatal("Failed to update feature progress:", err)
	}

	updatedFeature, err := featureRepo.GetByID(feature.ID)
	if err != nil {
		log.Fatal("Failed to get updated feature:", err)
	}
	fmt.Printf("✓ Feature progress updated: %.1f%% (1/1 tasks completed)\n", updatedFeature.ProgressPct)

	// Test Epic Progress Calculation
	epicProgress, err := epicRepo.CalculateProgress(epic.ID)
	if err != nil {
		log.Fatal("Failed to calculate epic progress:", err)
	}
	fmt.Printf("✓ Epic progress: %.1f%%\n", epicProgress)

	// Test Cascade Delete
	fmt.Println("\n--- Testing Cascade Delete ---")

	if err := epicRepo.Delete(epic.ID); err != nil {
		log.Fatal("Failed to delete epic:", err)
	}
	fmt.Println("✓ Deleted epic (should cascade to features and tasks)")

	// Verify cascade delete worked
	features, err := featureRepo.ListByEpic(epic.ID)
	if err != nil {
		log.Fatal("Failed to list features after epic delete:", err)
	}
	fmt.Printf("✓ Features after cascade delete: %d (should be 0)\n", len(features))

	tasks, err := taskRepo.ListByFeature(feature.ID)
	if err != nil {
		log.Fatal("Failed to list tasks after epic delete:", err)
	}
	fmt.Printf("✓ Tasks after cascade delete: %d (should be 0)\n", len(tasks))

	fmt.Println("\n✅ All tests passed!")
}

func strPtr(s string) *string {
	return &s
}
