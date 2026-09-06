package gaterun

import "testing"

func TestDeriveSuboperationID_DeterministicAndUnique(t *testing.T) {
	digest := "deadbeef"
	kinds := []struct {
		kind, item string
	}{
		{"finding", "fingerprint-abc123"},
		{"sweep", "class_key-xyz"},
		{"impact", "adr:ADR-001"},
		{"kickback", "E01-F02-003"},
		{"gate_summary", "gate_summary"},
	}

	seen := map[string]string{}
	for _, k := range kinds {
		id1 := DeriveSuboperationID(digest, k.kind, k.item)
		id2 := DeriveSuboperationID(digest, k.kind, k.item)
		if id1 != id2 {
			t.Errorf("%s/%s: not deterministic: %s vs %s", k.kind, k.item, id1, id2)
		}
		if len(id1) != 64 {
			t.Errorf("%s/%s: id length = %d, want 64", k.kind, k.item, len(id1))
		}
		if prev, ok := seen[id1]; ok {
			t.Errorf("%s/%s collided with %s", k.kind, k.item, prev)
		}
		seen[id1] = k.kind + "/" + k.item
	}
}

func TestDeriveSuboperationID_DiffersByDigest(t *testing.T) {
	a := DeriveSuboperationID("digest-a", "finding", "fp-1")
	b := DeriveSuboperationID("digest-b", "finding", "fp-1")
	if a == b {
		t.Error("suboperation id unchanged when operation digest differs")
	}
}

func TestDeriveSuboperationID_NoDelimiterCollision(t *testing.T) {
	// "kind|item" boundaries must not be confusable: ("a|b", "c") vs
	// ("a", "b|c") must not collide even though naive concatenation would
	// produce the same joined string.
	id1 := DeriveSuboperationID("d", "a|b", "c")
	id2 := DeriveSuboperationID("d", "a", "b|c")
	if id1 == id2 {
		t.Error("suboperation id collides across a kind/item delimiter boundary")
	}
}
