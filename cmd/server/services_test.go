package main

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// Compile-time check that ServiceContainer carries BugService and
// ChangeCardService slots. If either field is dropped this stops compiling.
func TestServiceContainer_HasBugAndChangeCardServices(t *testing.T) {
	var container ServiceContainer

	var _ *services.BugService = container.BugService
	var _ *services.ChangeCardService = container.ChangeCardService

	if container.BugService != nil {
		t.Error("expected nil BugService in zero-value container")
	}
	if container.ChangeCardService != nil {
		t.Error("expected nil ChangeCardService in zero-value container")
	}
}

// Compile-time check that ServiceContainer carries a TagService slot.
// The runtime check that every entity service receives the same injected
// *TagService lives in internal/viewer/server/wire_test.go (same-package
// access to unexported fields).
func TestServiceContainer_HasTagService(t *testing.T) {
	var container ServiceContainer
	var _ *services.TagService = container.TagService

	if container.TagService != nil {
		t.Error("expected nil TagService in zero-value container")
	}
}

// Runtime smoke test: WireServices constructs a non-nil TagService and
// every entity-service slot is populated.
func TestWireServices_InjectsTagService(t *testing.T) {
	sqlDB, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory test DB: %v", err)
	}
	repoDB := repository.NewDB(sqlDB)

	container := WireServices(repoDB, t.TempDir())
	if container == nil {
		t.Fatal("WireServices returned nil container")
	}

	if container.TagService == nil {
		t.Fatal("WireServices did not construct a TagService")
	}

	if container.TaskService == nil {
		t.Error("TaskService is nil")
	}
	if container.FeatureService == nil {
		t.Error("FeatureService is nil")
	}
	if container.EpicService == nil {
		t.Error("EpicService is nil")
	}
	if container.BugService == nil {
		t.Error("BugService is nil")
	}
	if container.ChangeCardService == nil {
		t.Error("ChangeCardService is nil")
	}
}
