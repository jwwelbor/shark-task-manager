package services

import (
	"testing"
)

func TestRequireNonNil_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		expected := "TestDep must not be nil"
		if msg != expected {
			t.Errorf("expected panic message %q, got %q", expected, msg)
		}
	}()

	requireNonNil(nil, "TestDep")
}

func TestRequireNonNil_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	// Should not panic for non-nil values
	requireNonNil("hello", "StringDep")
	requireNonNil(42, "IntDep")
	requireNonNil(&struct{}{}, "StructDep")
}

func TestRequireNonNil_InterfaceWithNilValue(t *testing.T) {
	// An interface holding a typed nil pointer is NOT nil itself in Go,
	// so requireNonNil should NOT panic. This is consistent with
	// the original nil-check pattern (if repo == nil).
	//
	// We use reflect to construct the value to avoid staticcheck SA4023.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	// (*testing.T)(nil) wrapped in interface{} is non-nil because type info is set.
	// requireNonNil checks interface{} == nil, which is false here.
	requireNonNil((*testing.T)(nil), "NilPointerInterface")
}
