package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

func main() {
	ctx := context.Background()

	// Use the main database (not a test database)
	dbPath := "shark-tasks.db"

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	fmt.Println("📊 Shark Task Manager - Database Demo")
	fmt.Println("=====================================")

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create sample epic
	fmt.Println("1️⃣  Creating Epic...")
	businessValue := models.PriorityHigh
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:         "E04",
			Title:       "Task Management CLI - Core Functionality",
			Description: strPtr("Complete database schema and repository implementation"),
		},
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &businessValue,
	}

	// Check if epic already exists
	existingEpic, _ := epicRepo.GetByKey(ctx, epic.Key)
	if existingEpic != nil {
		fmt.Printf("   Epic %s already exists (ID: %d)\n", epic.Key, existingEpic.ID)
		epic = existingEpic
	} else {
		if err := epicRepo.Create(ctx, epic); err != nil {
			slog.Error("Failed to create epic", "error", err)
			os.Exit(1)
		}
		fmt.Printf("   ✓ Created Epic: %s - %s\n", epic.Key, epic.Title)
	}

	// Create sample feature
	fmt.Println("\n2️⃣  Creating Feature...")
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E04-F01",
		Title:       "Database Schema & Core Data Model",
		Description: strPtr("SQLite database with full schema implementation")}, EpicID: epic.ID,

		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}

	existingFeature, _ := featureRepo.GetByKey(ctx, feature.Key)
	if existingFeature != nil {
		fmt.Printf("   Feature %s already exists (ID: %d)\n", feature.Key, existingFeature.ID)
		feature = existingFeature
	} else {
		if err := featureRepo.Create(ctx, feature); err != nil {
			slog.Error("Failed to create feature", "error", err)
			os.Exit(1)
		}
		fmt.Printf("   ✓ Created Feature: %s - %s\n", feature.Key, feature.Title)
	}

	// Create sample tasks
	fmt.Println("\n3️⃣  Creating Tasks...")

	tasks := []struct {
		key         string
		title       string
		description string
		agentType   string
		priority    int
	}{
		{"T-E04-F01-001", "Create ORM Models", "Define Epic, Feature, Task, TaskHistory models", "backend", 1},
		{"T-E04-F01-002", "Implement Validation", "Add validation for keys, enums, and ranges", "backend", 2},
		{"T-E04-F01-003", "Create Database Schema", "Define all tables, indexes, and triggers", "backend", 3},
		{"T-E04-F01-004", "Build Repository Layer", "Implement CRUD operations for all models", "backend", 4},
		{"T-E04-F01-005", "Add Unit Tests", "Create comprehensive test coverage", "testing", 5},
	}

	createdTasks := []*models.Task{}
	for _, t := range tasks {
		existingTask, _ := taskRepo.GetByKey(ctx, t.key)
		if existingTask != nil {
			fmt.Printf("   Task %s already exists\n", t.key)
			createdTasks = append(createdTasks, existingTask)
			continue
		}

		task := &models.Task{BaseEntity: models.BaseEntity{Key: t.key,
			Title:       t.title,
			Description: strPtr(t.description)}, FeatureID: feature.ID,

			Status:    models.TaskStatus("todo"),
			AgentType: &t.agentType,
			Priority:  t.priority,
			DependsOn: strPtr("[]"),
		}

		if err := taskRepo.Create(ctx, task); err != nil {
			slog.Error("Failed to create task", "error", err)
			os.Exit(1)
		}
		fmt.Printf("   ✓ Created: %s - %s\n", task.Key, task.Title)
		createdTasks = append(createdTasks, task)
	}

	// Update task statuses to simulate work
	fmt.Println("\n4️⃣  Simulating Task Progress...")

	agent := "demo-agent"

	// Mark first task as in progress
	if len(createdTasks) > 0 && createdTasks[0].Status == models.TaskStatus("todo") {
		if err := taskRepo.UpdateStatus(ctx, createdTasks[0].ID, models.TaskStatus("in_progress"), &agent, strPtr("Starting implementation")); err != nil {
			slog.Error("Failed to update task status", "error", err)
			os.Exit(1)
		}
		fmt.Printf("   ✓ %s → in_progress\n", createdTasks[0].Key)
	}

	// Mark first three tasks as completed
	for i := 0; i < 3 && i < len(createdTasks); i++ {
		if createdTasks[i].Status != models.TaskStatus("completed") {
			if err := taskRepo.UpdateStatus(ctx, createdTasks[i].ID, models.TaskStatus("completed"), &agent, strPtr("Implementation complete")); err != nil {
				slog.Error("Failed to update task status", "error", err)
				os.Exit(1)
			}
			fmt.Printf("   ✓ %s → completed\n", createdTasks[i].Key)
		}
	}

	// Update feature progress via service layer
	workflowSvc := workflow.NewService(".")
	entitySvc := services.NewEntityService(workflowSvc)
	featureSvc := services.NewFeatureService(featureRepo, entitySvc, services.NewNoopEntityRepository(), taskRepo, epicRepo)
	if err := featureSvc.RecalculateAndSetProgress(ctx, feature.ID); err != nil {
		slog.Error("Failed to update feature progress", "error", err)
		os.Exit(1)
	}

	// Display current state
	fmt.Println("\n5️⃣  Current State:")
	fmt.Println("   ─────────────────────────────────────────")

	// Get updated feature
	updatedFeature, _ := featureRepo.GetByID(ctx, feature.ID)
	fmt.Printf("   Epic: %s - %s\n", epic.Key, epic.Title)
	fmt.Printf("   Feature: %s - %s (%.1f%% complete)\n",
		updatedFeature.Key, updatedFeature.Title, updatedFeature.ProgressPct)

	fmt.Println("\n   Tasks:")
	allTasks, _ := taskRepo.ListByFeature(ctx, feature.ID)
	statusCounts := make(map[models.TaskStatus]int)
	for _, task := range allTasks {
		statusCounts[task.Status]++
		statusIcon := getStatusIcon(task.Status)
		fmt.Printf("     %s %s - %s [%s]\n",
			statusIcon, task.Key, task.Title, task.Status)
	}

	fmt.Println("\n   Summary:")
	fmt.Printf("     Total Tasks: %d\n", len(allTasks))
	for status, count := range statusCounts {
		fmt.Printf("     %s: %d\n", status, count)
	}

	// Show epic progress via service layer
	epicSvc := services.NewEpicService(epicRepo, entitySvc, services.NewNoopEntityRepository(), featureRepo, taskRepo)
	epicProgressInfo, err := epicSvc.GetProgress(ctx, epic.Key)
	if err != nil {
		slog.Error("Failed to get epic progress", "error", err)
		os.Exit(1)
	}
	fmt.Printf("\n   Epic Progress: %.1f%%\n", epicProgressInfo.ProgressPct)

	fmt.Println("\n6️⃣  Testing Queries:")
	fmt.Println("   ─────────────────────────────────────────")

	// Filter by status
	todoTasks, _ := taskRepo.FilterByStatus(ctx, models.TaskStatus("todo"))
	fmt.Printf("   Tasks with status 'todo': %d\n", len(todoTasks))

	// Filter by agent type
	backendTasks, _ := taskRepo.FilterByAgentType(ctx, "backend")
	fmt.Printf("   Tasks for backend agent: %d\n", len(backendTasks))

	// Combined filter
	todoStatus := models.TaskStatus("todo")
	maxPriority := 3
	filteredTasks, _ := taskRepo.FilterCombined(ctx, &todoStatus, nil, nil, &maxPriority)
	fmt.Printf("   High-priority todo tasks (priority ≤ 3): %d\n", len(filteredTasks))

	fmt.Println("\n✅ Demo completed! Database: shark-tasks.db")
	fmt.Println("\nTo inspect the database manually, run:")
	fmt.Println("  make clean  # to reset")
	fmt.Println("  make run    # to start the server")
}

func strPtr(s string) *string {
	return &s
}

func getStatusIcon(status models.TaskStatus) string {
	switch status {
	case models.TaskStatus("todo"):
		return "⭕"
	case models.TaskStatus("in_progress"):
		return "🔄"
	case models.TaskStatus("completed"):
		return "✅"
	case models.TaskStatus("blocked"):
		return "🚫"
	case models.TaskStatus("ready_for_review"):
		return "👀"
	case models.TaskStatus("archived"):
		return "📦"
	default:
		return "❓"
	}
}
