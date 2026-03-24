package repository

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	epicpkg "github.com/jwwelbor/shark-task-manager/internal/repository/epic"
	featurepkg "github.com/jwwelbor/shark-task-manager/internal/repository/feature"
	"github.com/jwwelbor/shark-task-manager/internal/repository/note"
	taskpkg "github.com/jwwelbor/shark-task-manager/internal/repository/task"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// setupTracingTest configures a recording TracerProvider and returns the exporter.
// The exporter captures all spans created during the test for assertion.
// Cleanup restores the original tracer and provider.
//
// Goroutine safety: tracerMu is acquired before any package-level tracer variable is
// mutated and released only after the cleanup closure has fully restored all originals.
// This prevents data races when tests are run with t.Parallel().
func setupTracingTest(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	// Acquire the tracer lock before touching any global state.
	// The lock is released in the t.Cleanup closure below, after all originals
	// are restored, so only one test at a time can swap the global tracers.
	tracerMu.Lock()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	prevProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)

	// Re-initialize the root-package tracer so it picks up the new provider.
	prevTracer := repoTracer
	repoTracer = tp.Tracer("internal/repository")

	// Re-initialize the note sub-package tracer. EntityNoteRepository was moved
	// to note/ in Phase 3 and uses its own noteTracer; this ensures spans from
	// note.EntityNoteRepository are captured by the in-memory exporter.
	restoreNoteTracer := note.SetTracerForTesting(tp.Tracer("internal/repository/note"))

	// Re-initialize the Phase 4 sub-package tracers (epic, feature, task).
	// These repositories were moved to dedicated sub-packages in Phase 4 and each
	// maintain their own package-level tracer. Replacing them here ensures spans
	// emitted by EpicRepository, FeatureRepository, and TaskRepository are
	// captured by the in-memory exporter during tracing tests.
	restoreTaskTracer := taskpkg.SetTracerForTesting(tp.Tracer("internal/repository/task"))
	restoreFeatureTracer := featurepkg.SetTracerForTesting(tp.Tracer("internal/repository/feature"))
	restoreEpicTracer := epicpkg.SetTracerForTesting(tp.Tracer("internal/repository/epic"))

	t.Cleanup(func() {
		// Restore all tracers to their original values before releasing the lock
		// so that no concurrent test can observe a partially-restored state.
		restoreEpicTracer()
		restoreFeatureTracer()
		restoreTaskTracer()
		restoreNoteTracer()
		repoTracer = prevTracer
		otel.SetTracerProvider(prevProvider)
		_ = tp.Shutdown(context.Background())
		tracerMu.Unlock()
	})

	return exporter
}

// stubAttrMap extracts span stub attributes into a string map for easy assertion.
func stubAttrMap(stub tracetest.SpanStub) map[string]string {
	result := make(map[string]string)
	for _, attr := range stub.Attributes {
		result[string(attr.Key)] = attr.Value.Emit()
	}
	return result
}

// findStubByName returns the first span stub with the given name, or nil.
func findStubByName(stubs tracetest.SpanStubs, name string) *tracetest.SpanStub {
	for i := range stubs {
		if stubs[i].Name == name {
			return &stubs[i]
		}
	}
	return nil
}

// testMu protects concurrent test access to the shared test database seeding.
var testMu sync.Mutex

// tracerMu serialises the global-provider swap and all sub-package tracer swaps
// performed by setupTracingTest. The lock is acquired before any tracer variable is
// mutated and released only after the cleanup closure has restored all originals.
// This prevents data races if tests are ever run with t.Parallel().
var tracerMu sync.Mutex

func TestTaskRepository_GetByKey_SpanCreated(t *testing.T) {
	exporter := setupTracingTest(t)

	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)

	testMu.Lock()
	_, featureID := test.SeedTestData()
	testMu.Unlock()

	// Create a test task
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:   "T-E99-F01-901",
			Title: "Tracing Test Task",
		},
		FeatureID: featureID,
		Status:    models.TaskStatus("todo"),
		Priority:  5,
	}

	ctx := context.Background()
	_ = repo.Create(ctx, task)
	defer func() {
		_ = repo.Delete(ctx, task.ID)
	}()

	exporter.Reset()

	// Act
	_, _ = repo.GetByKey(ctx, "T-E99-F01-901")

	// Assert
	stubs := exporter.GetSpans()
	stub := findStubByName(stubs, "TaskRepository.GetByKey")
	if stub == nil {
		t.Fatal("expected span TaskRepository.GetByKey, got none")
	}

	attrs := stubAttrMap(*stub)
	if attrs["db.operation"] != "SELECT" {
		t.Errorf("expected db.operation=SELECT, got %s", attrs["db.operation"])
	}
	if attrs["db.table"] != "tasks" {
		t.Errorf("expected db.table=tasks, got %s", attrs["db.table"])
	}
	if attrs["db.system"] != "sqlite" {
		t.Errorf("expected db.system=sqlite, got %s", attrs["db.system"])
	}
}

func TestTaskRepository_GetByKey_ErrorRecorded(t *testing.T) {
	exporter := setupTracingTest(t)

	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)

	ctx := context.Background()

	// Act - look up a key that doesn't exist
	_, err := repo.GetByKey(ctx, "NONEXISTENT-KEY-999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Assert
	stubs := exporter.GetSpans()
	stub := findStubByName(stubs, "TaskRepository.GetByKey")
	if stub == nil {
		t.Fatal("expected span TaskRepository.GetByKey, got none")
	}

	if stub.Status.Code != codes.Error {
		t.Errorf("expected span status Error, got %v", stub.Status.Code)
	}

	// RecordError adds an event
	if len(stub.Events) == 0 {
		t.Error("expected span events from RecordError, got none")
	}
}

func TestFeatureRepository_GetByKey_SpanCreated(t *testing.T) {
	exporter := setupTracingTest(t)

	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewFeatureRepository(db)

	testMu.Lock()
	test.SeedTestData()
	testMu.Unlock()

	exporter.Reset()

	ctx := context.Background()

	// Act - use the seeded feature key
	_, _ = repo.GetByKey(ctx, "E99-F01")

	// Assert
	stubs := exporter.GetSpans()
	stub := findStubByName(stubs, "FeatureRepository.GetByKey")
	if stub == nil {
		t.Fatal("expected span FeatureRepository.GetByKey, got none")
	}

	attrs := stubAttrMap(*stub)
	if attrs["db.operation"] != "SELECT" {
		t.Errorf("expected db.operation=SELECT, got %s", attrs["db.operation"])
	}
	if attrs["db.table"] != "features" {
		t.Errorf("expected db.table=features, got %s", attrs["db.table"])
	}
	if attrs["db.system"] != "sqlite" {
		t.Errorf("expected db.system=sqlite, got %s", attrs["db.system"])
	}
}

func TestEpicRepository_GetByKey_SpanCreated(t *testing.T) {
	exporter := setupTracingTest(t)

	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	testMu.Lock()
	test.SeedTestData()
	testMu.Unlock()

	exporter.Reset()

	ctx := context.Background()

	// Act
	_, _ = repo.GetByKey(ctx, "E99")

	// Assert
	stubs := exporter.GetSpans()
	stub := findStubByName(stubs, "EpicRepository.GetByKey")
	if stub == nil {
		t.Fatal("expected span EpicRepository.GetByKey, got none")
	}

	attrs := stubAttrMap(*stub)
	if attrs["db.operation"] != "SELECT" {
		t.Errorf("expected db.operation=SELECT, got %s", attrs["db.operation"])
	}
	if attrs["db.table"] != "epics" {
		t.Errorf("expected db.table=epics, got %s", attrs["db.table"])
	}
	if attrs["db.system"] != "sqlite" {
		t.Errorf("expected db.system=sqlite, got %s", attrs["db.system"])
	}
}

func TestEntityNoteRepository_Create_SpanCreated(t *testing.T) {
	exporter := setupTracingTest(t)

	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEntityNoteRepository(db)

	testMu.Lock()
	epicID, _ := test.SeedTestData()
	testMu.Unlock()

	exporter.Reset()

	ctx := context.Background()

	note := &models.EntityNote{
		EntityType: models.EntityTypeEpic,
		EntityID:   epicID,
		NoteType:   models.NoteTypeComment,
		Content:    "tracing test note",
	}

	// Act
	err := repo.Create(ctx, note)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_ = repo.Delete(ctx, note.ID)
	}()

	// Assert
	stubs := exporter.GetSpans()
	stub := findStubByName(stubs, "EntityNoteRepository.Create")
	if stub == nil {
		t.Fatal("expected span EntityNoteRepository.Create, got none")
	}

	attrs := stubAttrMap(*stub)
	if attrs["db.operation"] != "INSERT" {
		t.Errorf("expected db.operation=INSERT, got %s", attrs["db.operation"])
	}
	if attrs["db.table"] != "entity_notes" {
		t.Errorf("expected db.table=entity_notes, got %s", attrs["db.table"])
	}
	if attrs["db.system"] != "sqlite" {
		t.Errorf("expected db.system=sqlite, got %s", attrs["db.system"])
	}
}

func TestTaskRepository_Create_SpanCreated(t *testing.T) {
	exporter := setupTracingTest(t)

	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)

	testMu.Lock()
	_, featureID := test.SeedTestData()
	testMu.Unlock()

	exporter.Reset()

	ctx := context.Background()

	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:   "T-E99-F01-902",
			Title: "Tracing Create Test",
		},
		FeatureID: featureID,
		Status:    models.TaskStatus("todo"),
		Priority:  5,
	}

	// Act
	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_ = repo.Delete(ctx, task.ID)
	}()

	// Assert
	stubs := exporter.GetSpans()
	stub := findStubByName(stubs, "TaskRepository.Create")
	if stub == nil {
		t.Fatal("expected span TaskRepository.Create, got none")
	}

	attrs := stubAttrMap(*stub)
	if attrs["db.operation"] != "INSERT" {
		t.Errorf("expected db.operation=INSERT, got %s", attrs["db.operation"])
	}
	if attrs["db.table"] != "tasks" {
		t.Errorf("expected db.table=tasks, got %s", attrs["db.table"])
	}
}

func TestTaskRepository_UpdateStatus_SpanCreated(t *testing.T) {
	exporter := setupTracingTest(t)

	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)

	testMu.Lock()
	_, featureID := test.SeedTestData()
	testMu.Unlock()

	ctx := context.Background()

	// Create a test task first
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:   "T-E99-F01-903",
			Title: "Tracing Status Test",
		},
		FeatureID: featureID,
		Status:    models.TaskStatus("todo"),
		Priority:  5,
	}
	_ = repo.Create(ctx, task)
	defer func() {
		_ = repo.Delete(ctx, task.ID)
	}()

	exporter.Reset()

	// Act
	_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, nil)

	// Assert
	stubs := exporter.GetSpans()
	stub := findStubByName(stubs, "TaskRepository.UpdateStatus")
	if stub == nil {
		t.Fatal("expected span TaskRepository.UpdateStatus, got none")
	}

	attrs := stubAttrMap(*stub)
	if attrs["db.operation"] != "UPDATE" {
		t.Errorf("expected db.operation=UPDATE, got %s", attrs["db.operation"])
	}
	if attrs["db.table"] != "tasks" {
		t.Errorf("expected db.table=tasks, got %s", attrs["db.table"])
	}
}

func TestTaskRepository_Delete_SpanCreated(t *testing.T) {
	exporter := setupTracingTest(t)

	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)

	testMu.Lock()
	_, featureID := test.SeedTestData()
	testMu.Unlock()

	ctx := context.Background()

	// Create a test task to delete
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:   "T-E99-F01-904",
			Title: "Tracing Delete Test",
		},
		FeatureID: featureID,
		Status:    models.TaskStatus("todo"),
		Priority:  5,
	}
	_ = repo.Create(ctx, task)

	exporter.Reset()

	// Act
	err := repo.Delete(ctx, task.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Assert
	stubs := exporter.GetSpans()
	stub := findStubByName(stubs, "TaskRepository.Delete")
	if stub == nil {
		t.Fatal("expected span TaskRepository.Delete, got none")
	}

	attrs := stubAttrMap(*stub)
	if attrs["db.operation"] != "DELETE" {
		t.Errorf("expected db.operation=DELETE, got %s", attrs["db.operation"])
	}
	if attrs["db.table"] != "tasks" {
		t.Errorf("expected db.table=tasks, got %s", attrs["db.table"])
	}
}
