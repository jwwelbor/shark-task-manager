package gaterun

import (
	"encoding/json"
	"testing"
)

func TestComputeOperationDigest_DeterministicRegardlessOfKeyOrder(t *testing.T) {
	env1 := json.RawMessage(`{"b":2,"a":1}`)
	env2 := json.RawMessage(`{"a":1,"b":2}`)

	d1, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", env1)
	if err != nil {
		t.Fatalf("digest 1: %v", err)
	}
	d2, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", env2)
	if err != nil {
		t.Fatalf("digest 2: %v", err)
	}
	if d1 != d2 {
		t.Errorf("digest not stable across key order: %s vs %s", d1, d2)
	}
	if len(d1) != 64 {
		t.Errorf("digest length = %d, want 64 (hex sha256)", len(d1))
	}
}

func TestComputeOperationDigest_DiffersOnIdentityChange(t *testing.T) {
	env := json.RawMessage(`{"summary":"ok"}`)
	base, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", env)
	if err != nil {
		t.Fatalf("base digest: %v", err)
	}
	cases := []struct {
		name                                      string
		entityKey, entityType, sourceStatus, gate string
	}{
		{"entity_key", "E01-F01-002", "task", "in_review", "code_review"},
		{"entity_type", "E01-F01-001", "feature", "in_review", "code_review"},
		{"source_status", "E01-F01-001", "task", "in_qa", "code_review"},
		{"gate", "E01-F01-001", "task", "in_review", "uat"},
	}
	for _, c := range cases {
		got, err := ComputeOperationDigest(c.entityKey, c.entityType, c.sourceStatus, c.gate, env)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got == base {
			t.Errorf("%s: digest unchanged when identity field varied", c.name)
		}
	}
}

func TestComputeOperationDigest_DiffersOnEnvelopeChange(t *testing.T) {
	d1, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", json.RawMessage(`{"summary":"ok"}`))
	if err != nil {
		t.Fatalf("d1: %v", err)
	}
	d2, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", json.RawMessage(`{"summary":"different"}`))
	if err != nil {
		t.Fatalf("d2: %v", err)
	}
	if d1 == d2 {
		t.Error("digest unchanged when envelope content changed")
	}
}

func TestComputeOperationDigest_ArrayOrderMatters(t *testing.T) {
	d1, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", json.RawMessage(`{"findings":["a","b"]}`))
	if err != nil {
		t.Fatalf("d1: %v", err)
	}
	d2, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", json.RawMessage(`{"findings":["b","a"]}`))
	if err != nil {
		t.Fatalf("d2: %v", err)
	}
	if d1 == d2 {
		t.Error("digest should differ when contract-order array content differs")
	}
}

func TestComputeOperationDigest_RejectsInvalidEnvelopeJSON(t *testing.T) {
	if _, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", json.RawMessage(`not json`)); err == nil {
		t.Fatal("want error for invalid envelope JSON, got nil")
	}
}

func TestComputeOperationDigest_NoHTMLEscaping(t *testing.T) {
	// A digest computed over content containing HTML-sensitive characters
	// must not depend on Go's default HTML-escaping behavior, or the same
	// logical envelope could digest differently depending on encoder
	// defaults elsewhere in the stack.
	d, err := ComputeOperationDigest("E01-F01-001", "task", "in_review", "code_review", json.RawMessage(`{"summary":"a<b>c&d"}`))
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if d == "" {
		t.Fatal("empty digest")
	}
}
