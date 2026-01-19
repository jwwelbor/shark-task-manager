package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTaskUpdateWithReasonDoc tests that shark task update stores document path in rejection notes
func TestTaskUpdateWithReasonDoc(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	defer cli.ResetDB()

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E99%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	// Create test epic and feature
	epicRepo := repository.NewEpicRepository(db)
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic for Rejection Doc",
		Status:   "active",
		Priority: "medium",
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
	}()

	featureRepo := repository.NewFeatureRepository(db)
	feature := &models.Feature{
		Key:    "E99-F99",
		Status: "active",
		Title:  "Test Feature",
		EpicID: epic.ID,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}()

	// Create test task
	taskRepo := repository.NewTaskRepository(db)
	task := &models.Task{
		Key:       "T-E99-F99-001",
		Title:     "Test Task for Rejection Doc",
		Status:    models.TaskStatus("in_progress"),
		FeatureID: feature.ID,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}()

	// Move task to ready_for_review
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, nil)
	if err != nil {
		t.Fatalf("Failed to update task to ready_for_review: %v", err)
	}

	// Create document in project root for testing
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	testDocPath := "test-rejection-doc.md"
	fullDocPath := filepath.Join(projectRoot, testDocPath)
	err = os.WriteFile(fullDocPath, []byte("# Test Rejection Document\n\nThis is a test."), 0644)
	if err != nil {
		t.Fatalf("Failed to create test document in project root: %v", err)
	}
	defer os.Remove(fullDocPath)

	// Update task back to in_progress with rejection reason and document
	reason := "Code review failed - see test document"
	documentPath := &testDocPath
	err = taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("in_progress"), nil, nil, &reason, documentPath, true)
	if err != nil {
		t.Fatalf("Failed to update task with rejection reason and document: %v", err)
	}

	// Verify rejection note was created
	noteRepo := repository.NewTaskNoteRepository(db)
	rejectionHistory, err := noteRepo.GetRejectionHistory(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get rejection history: %v", err)
	}

	if len(rejectionHistory) == 0 {
		t.Fatal("Expected at least one rejection history entry")
	}

	latestRejection := rejectionHistory[0]

	// Verify reason is stored
	if latestRejection.Reason != reason {
		t.Errorf("Expected reason %q, got %q", reason, latestRejection.Reason)
	}

	// Verify document path is stored
	if latestRejection.ReasonDocument == nil {
		t.Fatal("Expected ReasonDocument to be set, got nil")
	}

	if *latestRejection.ReasonDocument != testDocPath {
		t.Errorf("Expected document path %q, got %q", testDocPath, *latestRejection.ReasonDocument)
	}

	// Verify from/to status
	if latestRejection.FromStatus != "ready_for_review" {
		t.Errorf("Expected from_status %q, got %q", "ready_for_review", latestRejection.FromStatus)
	}

	if latestRejection.ToStatus != "in_progress" {
		t.Errorf("Expected to_status %q, got %q", "in_progress", latestRejection.ToStatus)
	}
}

// TestTaskNextStatusWithReasonDoc tests that shark task next-status stores document path
func TestTaskNextStatusWithReasonDoc(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	defer cli.ResetDB()

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E98%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create test epic and feature
	epicRepo := repository.NewEpicRepository(db)
	epic := &models.Epic{
		Key:      "E98",
		Title:    "Test Epic for Next Status Doc",
		Status:   "active",
		Priority: "medium",
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
	}()

	featureRepo := repository.NewFeatureRepository(db)
	feature := &models.Feature{
		Key:    "E98-F98",
		Status: "active",
		Title:  "Test Feature",
		EpicID: epic.ID,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}()

	// Create test task
	taskRepo := repository.NewTaskRepository(db)
	task := &models.Task{
		Key:       "T-E98-F98-001",
		Title:     "Test Task for Next Status Doc",
		Status:    models.TaskStatus("in_progress"),
		FeatureID: feature.ID,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}()

	// Move task to ready_for_review
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, nil)
	if err != nil {
		t.Fatalf("Failed to update task to ready_for_review: %v", err)
	}

	// Create document in project root for testing
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	testDocPath := "test-next-status-doc.md"
	fullDocPath := filepath.Join(projectRoot, testDocPath)
	err = os.WriteFile(fullDocPath, []byte("# Test Next Status Document\n\nCode review findings."), 0644)
	if err != nil {
		t.Fatalf("Failed to create test document in project root: %v", err)
	}
	defer os.Remove(fullDocPath)

	// Simulate shark task next-status with --reason and --reason-doc
	reason := "Code review rejected - see document"
	documentPath := &testDocPath

	// Use UpdateStatusForced to simulate next-status with force flag
	err = taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("in_progress"), nil, nil, &reason, documentPath, true)
	if err != nil {
		t.Fatalf("Failed to transition with reason and document: %v", err)
	}

	// Verify rejection note was created with document path
	noteRepo := repository.NewTaskNoteRepository(db)
	rejectionHistory, err := noteRepo.GetRejectionHistory(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get rejection history: %v", err)
	}

	if len(rejectionHistory) == 0 {
		t.Fatal("Expected at least one rejection history entry")
	}

	latestRejection := rejectionHistory[0]

	// Verify document path is stored
	if latestRejection.ReasonDocument == nil {
		t.Fatal("Expected ReasonDocument to be set, got nil")
	}

	if *latestRejection.ReasonDocument != testDocPath {
		t.Errorf("Expected document path %q, got %q", testDocPath, *latestRejection.ReasonDocument)
	}

	// Verify reason
	if latestRejection.Reason != reason {
		t.Errorf("Expected reason %q, got %q", reason, latestRejection.Reason)
	}
}

// TestReasonDocValidation tests document path validation
func TestReasonDocValidation(t *testing.T) {
	tests := []struct {
		name    string
		docPath string
		wantErr bool
	}{
		{
			name:    "Valid markdown path",
			docPath: "docs/plan/epic/feature/review.md",
			wantErr: false,
		},
		{
			name:    "Valid nested path",
			docPath: "docs/code_review/2026-01-17/review.md",
			wantErr: false,
		},
		{
			name:    "Valid text file",
			docPath: "docs/review.txt",
			wantErr: false,
		},
		{
			name:    "Invalid - absolute path",
			docPath: "/home/user/docs/review.md",
			wantErr: true,
		},
		{
			name:    "Invalid - parent directory traversal",
			docPath: "../../../etc/passwd.md",
			wantErr: true,
		},
		{
			name:    "Invalid - empty",
			docPath: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRejectionReasonDocPath(tt.docPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRejectionReasonDocPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
