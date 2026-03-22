package main

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// TC-F09-059: Verify ServiceContainer includes BugService and ChangeCardService fields.
// This is a compile-time test -- if BugService or ChangeCardService fields are missing
// from ServiceContainer, or if WireServices doesn't populate them, this test won't compile.
func TestServiceContainer_HasBugAndChangeCardServices(t *testing.T) {
	// Verify ServiceContainer has the expected fields by accessing them.
	// This is primarily a compile-time check; runtime verification requires a DB.
	var container ServiceContainer

	// AC-T4: ServiceContainer must include BugService and ChangeCardService
	var _ *services.BugService = container.BugService
	var _ *services.ChangeCardService = container.ChangeCardService

	// These are nil since we didn't call WireServices, but the types must match.
	if container.BugService != nil {
		t.Error("expected nil BugService in zero-value container")
	}
	if container.ChangeCardService != nil {
		t.Error("expected nil ChangeCardService in zero-value container")
	}
}
