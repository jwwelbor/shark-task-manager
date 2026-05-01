package server

import (
	"reflect"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// newTestRepoDB creates an in-memory SQLite database with the full schema
// applied. It's the same helper shape used by server_test.go but lives in the
// package_internal test file so it can reach unexported fields.
func newTestRepoDB(t *testing.T) *repository.DB {
	t.Helper()
	sqlDB, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory test DB: %v", err)
	}
	return repository.NewDB(sqlDB)
}

// TestWireServices_ConstructsTagService verifies that WireServices returns a
// bundle which includes a non-nil *TagService and which has injected the
// shared TagService into every entity service the container exposes
// (TaskService, FeatureService, EpicService, BugService, ChangeCardService).
//
// This is the HTTP smoke test sketched in test-plan.md Section 2.5 bullet 3
// and required by T-E28-F04-011 AC-T1 and AC-T2.
//
// The check for tagSvc injection reads the unexported `tagSvc` field on each
// entity service via reflection. We accept a reflection-based field probe for
// this wiring test because:
//   - The field is deliberately unexported (see spec §2.6, REQ-F-018).
//   - No public accessor exists (the services keep the dep optional and
//     internal).
//   - The alternative — exercising a create-with-tag round-trip — requires
//     the full in-memory DB schema plus test fixtures for every entity type,
//     which is out of scope for a wiring smoke test. Unit tests in each
//     service package already cover the behavioural half.
func TestWireServices_ConstructsTagService(t *testing.T) {
	repoDB := newTestRepoDB(t)

	container := WireServices(repoDB, t.TempDir())

	if container == nil {
		t.Fatal("WireServices returned nil container")
	}

	// AC-T1: TagService must be non-nil in the returned bundle.
	if container.TagService == nil {
		t.Fatal("AC-T1: container.TagService is nil; WireServices must construct a TagService")
	}

	// Verify type.
	var _ *services.TagService = container.TagService

	// AC-T1 (injection): every entity service has its tagSvc field populated.
	cases := []struct {
		name string
		svc  interface{}
	}{
		{"TaskService", container.TaskService},
		{"FeatureService", container.FeatureService},
		{"EpicService", container.EpicService},
		{"BugService", container.BugService},
		{"ChangeCardService", container.ChangeCardService},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.svc == nil {
				t.Fatalf("%s is nil in the container", tc.name)
			}
			fv := tagSvcField(t, tc.svc)
			if !fv.IsValid() {
				t.Fatalf("%s has no tagSvc field", tc.name)
			}
			if fv.IsNil() {
				t.Errorf("%s has a nil tagSvc field; WireServices must inject the shared TagService", tc.name)
			}
		})
	}
}

// tagSvcField locates the unexported `tagSvc` field on the given service
// pointer and returns its reflect.Value. It supports both constructor-
// injected (BugService's NewBugService 8th arg) and setter-injected
// (SetTagService) fields because they both land in the same named field.
func tagSvcField(t *testing.T, svc interface{}) reflect.Value {
	t.Helper()
	v := reflect.ValueOf(svc)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		t.Fatalf("service value is not a non-nil pointer: %T", svc)
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		t.Fatalf("service does not point at a struct: %T", svc)
	}
	return elem.FieldByName("tagSvc")
}
