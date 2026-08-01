package cli

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// TestGetQuestionServiceResolvesDocumentPointerAgainstProjectRootFromSubdirectory
// proves GetQuestionService wires QuestionService with the project root
// FindProjectRoot() discovers, not the process's current directory -- a
// relative feature_change/architecture_decision resolution pointer must
// still resolve when a shark command is invoked from a nested subdirectory
// (the common case for AI agents working in docs/plan/<epic>/<feature>/).
// Before this wiring existed, resolution defaulted to ".", so this same
// pointer would fail with "document destination ... does not exist" whenever
// invoked from anywhere but the project root.
func TestGetQuestionServiceResolvesDocumentPointerAgainstProjectRootFromSubdirectory(t *testing.T) {
	cleanup := setupAccessorTestDB(t)
	defer cleanup()

	projectRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	docDir := filepath.Join(projectRoot, "docs", "architecture")
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(docs/architecture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(docDir, "coding-standards.md"), []byte("# Standards\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(coding-standards.md) error = %v", err)
	}

	subdir := filepath.Join(projectRoot, "work", "session")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll(subdir) error = %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("Chdir(subdir) error = %v", err)
	}
	defer func() {
		if err := os.Chdir(projectRoot); err != nil {
			t.Fatalf("restore Chdir(projectRoot) error = %v", err)
		}
	}()

	svc := GetQuestionService()
	ctx := context.Background()
	question, err := svc.CreateQuestion(ctx, services.CreateQuestionInput{
		Title: "Root-aware resolution", Summary: "Prove subdirectory invocation resolves docs against project root", Requester: "release-owner",
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	if _, err := svc.ConfigureWorkflow(ctx, services.ConfigureWorkflowInput{Key: question.Key, ResolutionOwner: "release-owner", Responders: []string{"alice"}}); err != nil {
		t.Fatalf("ConfigureWorkflow() error = %v", err)
	}

	// Force the Question directly to ready_for_resolution with a completed
	// responder. The full responder dispatch/claim lifecycle is exercised
	// elsewhere (TC102/TC103/TC105); this test isolates the root-aware
	// document-pointer seam that Resolve's feature_change/architecture_decision
	// kinds exercise.
	sqlDB, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("GetDB() error = %v", err)
	}
	questionRepo := repository.NewQuestionRepository(sqlDB)
	persisted, err := questionRepo.GetByKey(ctx, question.Key)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	state, err := models.DecodeQuestionState(persisted.ContextData)
	if err != nil || state == nil {
		t.Fatalf("DecodeQuestionState() = %v, %v", state, err)
	}
	state.Responders[0].Status = models.QuestionResponderCompleted
	state.Responses = append(state.Responses, models.QuestionResponse{
		SessionID: "session-alice", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC(),
	})
	encoded, err := models.EncodeQuestionState(persisted.ContextData, *state)
	if err != nil {
		t.Fatalf("EncodeQuestionState() error = %v", err)
	}
	persisted.ContextData = encoded
	if err := questionRepo.Update(ctx, persisted); err != nil {
		t.Fatalf("Update(context data) error = %v", err)
	}
	// Update() deliberately excludes status -- it's set only through the
	// typed status paths -- so advance it separately.
	if err := questionRepo.UpdateStatus(ctx, persisted.ID, models.QuestionStatusReadyForResolution); err != nil {
		t.Fatalf("UpdateStatus(ready_for_resolution) error = %v", err)
	}

	if _, err := svc.Resolve(ctx, services.ResolveQuestionInput{
		Key: question.Key, Owner: "release-owner", Kind: "feature_change", Pointer: "docs/architecture/coding-standards.md",
	}); err != nil {
		t.Fatalf("Resolve() from subdirectory error = %v, want root-aware resolution to succeed", err)
	}
}

func TestGetPortfolioAdviceServiceWiresProductionReaders(t *testing.T) {
	cleanup := setupAccessorTestDB(t)
	defer cleanup()
	ResetServices()
	ResetWorkflowService()
	defer func() {
		ResetServices()
		ResetWorkflowService()
	}()

	first := GetPortfolioAdviceService()
	second := GetPortfolioAdviceService()
	if first == nil || second == nil {
		t.Fatal("GetPortfolioAdviceService() returned nil")
	}
	if first == second {
		t.Error("GetPortfolioAdviceService() should create a lightweight service per call")
	}

	advice, err := first.Advise(context.Background())
	if err != nil {
		t.Fatalf("production-wired Advise() error = %v", err)
	}
	if advice == nil || advice.Epics == nil || advice.Relationships == nil || advice.Warnings == nil {
		t.Fatalf("production-wired advice has nil contract fields: %#v", advice)
	}
}

// TestResetServices_ReplacesContainer verifies that ResetServices() replaces
// the container with a fresh one, so subsequent Get* calls re-initialize.
func TestResetServices_ReplacesContainer(t *testing.T) {
	// Capture the original container pointer.
	before := loadContainer()
	if before == nil {
		t.Fatal("expected non-nil container before reset")
	}

	ResetServices()
	defer ResetServices() // ensure cleanup

	after := loadContainer()
	if after == nil {
		t.Fatal("expected non-nil container after reset")
	}

	if before == after {
		t.Error("ResetServices() should have replaced the container with a new one, but pointer is unchanged")
	}
}

// TestResetServices_FreshContainerIsEmpty verifies that after ResetServices()
// the container holds no initialized services (all fields are zero/nil).
func TestResetServices_FreshContainerIsEmpty(t *testing.T) {
	ResetServices()
	defer ResetServices()

	c := loadContainer()

	if c.registry != nil {
		t.Error("expected registry to be nil in fresh container")
	}
	if c.entityService != nil {
		t.Error("expected entityService to be nil in fresh container")
	}
	if c.actionService != nil {
		t.Error("expected actionService to be nil in fresh container")
	}
	if c.actionServiceErr != nil {
		t.Error("expected actionServiceErr to be nil in fresh container")
	}
	if c.noteService != nil {
		t.Error("expected noteService to be nil in fresh container")
	}
	if c.noteServiceErr != nil {
		t.Error("expected noteServiceErr to be nil in fresh container")
	}
	if c.contextService != nil {
		t.Error("expected contextService to be nil in fresh container")
	}
	if c.resumeService != nil {
		t.Error("expected resumeService to be nil in fresh container")
	}
	if c.resumeServiceErr != nil {
		t.Error("expected resumeServiceErr to be nil in fresh container")
	}
}

// TestServiceContainer_ConcurrentSwap verifies that concurrent swaps of the
// service container via storeContainer / loadContainer do not cause data races.
// This is the core safety property of the ServiceContainer pattern: resets are
// atomic pointer swaps rather than non-atomic reassignment of individual
// sync.Once values.
//
// Run with -race to detect any remaining issues: go test -race ./internal/cli/
func TestServiceContainer_ConcurrentSwap(t *testing.T) {
	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Directly exercise the atomic swap helpers, not ResetServices()
			// (which calls ResetObservability that has its own pre-existing state).
			storeContainer(new(serviceContainer))
			_ = loadContainer()
		}()
	}

	wg.Wait()

	// After concurrent swaps, container must still be non-nil and valid.
	c := loadContainer()
	if c == nil {
		t.Fatal("container must not be nil after concurrent swaps")
	}
}

// TestLoadStoreContainer_RoundTrip verifies the atomic load/store helpers
// maintain pointer identity correctly.
func TestLoadStoreContainer_RoundTrip(t *testing.T) {
	original := loadContainer()

	fresh := new(serviceContainer)
	storeContainer(fresh)
	defer storeContainer(original) // restore

	got := loadContainer()
	if got != fresh {
		t.Errorf("loadContainer() = %p, want %p", got, fresh)
	}
}

// TestServiceContainer_SyncOnceIsolation verifies that each serviceContainer
// has independent sync.Once instances, so resetting the container truly
// allows re-initialization rather than being a no-op.
func TestServiceContainer_SyncOnceIsolation(t *testing.T) {
	c1 := new(serviceContainer)
	c2 := new(serviceContainer)

	// Track how many times the Do function runs in each container.
	count1 := 0
	count2 := 0

	c1.registryOnce.Do(func() { count1++ })
	c1.registryOnce.Do(func() { count1++ }) // should not run again

	c2.registryOnce.Do(func() { count2++ }) // should run independently
	c2.registryOnce.Do(func() { count2++ }) // should not run again

	if count1 != 1 {
		t.Errorf("c1 sync.Once ran %d times, want 1", count1)
	}
	if count2 != 1 {
		t.Errorf("c2 sync.Once ran %d times, want 1", count2)
	}
}
