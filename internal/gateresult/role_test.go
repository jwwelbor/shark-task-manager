package gateresult

import "testing"

const mainEntity = "T-E34-F05-001"

func mustResult(t *testing.T) *GateResult {
	t.Helper()
	result, err := Decode(encode(t, validPayload()))
	if err != nil {
		t.Fatalf("fixture must decode: %v", err)
	}
	// The base fixture carries one open finding and one kickback targeting a
	// different entity; individual tests reshape it as needed.
	return result
}

func TestValidateRole_MainEntityKickbackRejectedRegardlessOfRole(t *testing.T) {
	for _, role := range []OutcomeRole{RoleSuccess, RoleRouteRework, RoleKickbackRework, RoleBlocked, RoleHold, RoleCancelled} {
		t.Run(string(role), func(t *testing.T) {
			payload := validPayload()
			payload["findings"] = []interface{}{}
			payload["kickbacks"] = []interface{}{
				map[string]interface{}{
					"entity_key":    mainEntity,
					"target_status": "todo",
					"reason":        "self-kickback must be rejected",
				},
			}
			payload["no_kickback_reason"] = "n/a"
			result, err := Decode(encode(t, payload))
			if err != nil {
				t.Fatalf("fixture must decode: %v", err)
			}
			if err := ValidateRole(role, result, mainEntity); err == nil {
				t.Fatalf("expected kickback targeting the bound main entity to be rejected under role %s", role)
			}
		})
	}
}

// TestValidateRole_SluggedAliasOfMainEntityRejected locks the
// authorization-bypass-via-key-aliasing fix (code-review round 11): a
// kickback whose entity_key is a slugged alias of the bound main entity
// (same canonical entity, different textual key) must be rejected exactly
// like an exact-string self-kickback. Before the fix, ValidateRole's `==`
// comparison let this through because boundedText performs no key-shape
// validation and a slugged alias never textually equals the bare main
// entity key.
func TestValidateRole_SluggedAliasOfMainEntityRejected(t *testing.T) {
	aliases := []string{
		mainEntity + "-implement-jwt-token-validation", // T-E34-F05-001-<slug>
		"E34-F05-001-implement-jwt-token-validation",   // short-form task key + slug
		"e34-f05-001",   // short-form task key, different case
		"t-e34-f05-001", // lowercase full form
	}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			payload := validPayload()
			payload["findings"] = []interface{}{}
			payload["kickbacks"] = []interface{}{
				map[string]interface{}{
					"entity_key":    alias,
					"target_status": "todo",
					"reason":        "aliased self-kickback must be rejected",
				},
			}
			payload["no_kickback_reason"] = "n/a"
			result, err := Decode(encode(t, payload))
			if err != nil {
				t.Fatalf("fixture must decode: %v", err)
			}
			if err := ValidateRole(RoleKickbackRework, result, mainEntity); err == nil {
				t.Fatalf("expected kickback entity_key %q (a canonical alias of main entity %q) to be rejected", alias, mainEntity)
			}
		})
	}
}

func TestValidateRole_Success(t *testing.T) {
	t.Run("no kickback no open finding accepted", func(t *testing.T) {
		payload := validPayload()
		payload["kickbacks"] = []interface{}{}
		finding := payload["findings"].([]interface{})[0].(map[string]interface{})
		finding["disposition"] = "fixed"
		result, err := Decode(encode(t, payload))
		if err != nil {
			t.Fatalf("fixture must decode: %v", err)
		}
		if err := ValidateRole(RoleSuccess, result, mainEntity); err != nil {
			t.Fatalf("expected clean success to be accepted, got %v", err)
		}
	})

	t.Run("open blocking finding rejected", func(t *testing.T) {
		payload := validPayload()
		payload["kickbacks"] = []interface{}{}
		result, err := Decode(encode(t, payload))
		if err != nil {
			t.Fatalf("fixture must decode: %v", err)
		}
		if err := ValidateRole(RoleSuccess, result, mainEntity); err == nil {
			t.Fatalf("expected open finding to reject success role")
		}
	})

	t.Run("severity_conflict blocking finding rejected", func(t *testing.T) {
		payload := validPayload()
		payload["kickbacks"] = []interface{}{}
		finding := payload["findings"].([]interface{})[0].(map[string]interface{})
		finding["disposition"] = "severity_conflict"
		finding["disposition_pointer"] = "docs/decisions/DEC-002.md"
		result, err := Decode(encode(t, payload))
		if err != nil {
			t.Fatalf("fixture must decode: %v", err)
		}
		if err := ValidateRole(RoleSuccess, result, mainEntity); err == nil {
			t.Fatalf("expected severity_conflict finding to reject success role")
		}
	})

	t.Run("non-blocking disposition tolerated", func(t *testing.T) {
		payload := validPayload()
		payload["kickbacks"] = []interface{}{}
		finding := payload["findings"].([]interface{})[0].(map[string]interface{})
		finding["disposition"] = "not_reproducible"
		result, err := Decode(encode(t, payload))
		if err != nil {
			t.Fatalf("fixture must decode: %v", err)
		}
		if err := ValidateRole(RoleSuccess, result, mainEntity); err != nil {
			t.Fatalf("expected not_reproducible finding to be tolerated under success, got %v", err)
		}
	})

	t.Run("any kickback rejected", func(t *testing.T) {
		result := mustResult(t)
		if err := ValidateRole(RoleSuccess, result, mainEntity); err == nil {
			t.Fatalf("expected any kickback to reject success role")
		}
	})

	t.Run("opaque deep_verify outcome key is unaffected by role validation", func(t *testing.T) {
		// deep_verify is a workflow outcome KEY, not a role; the parent maps it
		// to role success via outcome_roles (REQ-F-006, out of this task's
		// scope). This test only proves ValidateRole itself is indifferent to
		// the outcome key string and validates purely on the passed-in role.
		payload := validPayload()
		payload["kickbacks"] = []interface{}{}
		finding := payload["findings"].([]interface{})[0].(map[string]interface{})
		finding["disposition"] = "fixed"
		result, err := Decode(encode(t, payload))
		if err != nil {
			t.Fatalf("fixture must decode: %v", err)
		}
		if err := ValidateRole(RoleSuccess, result, mainEntity); err != nil {
			t.Fatalf("expected role success to accept regardless of which outcome key selected it, got %v", err)
		}
	})
}

func TestValidateRole_RouteRework(t *testing.T) {
	t.Run("no kickback accepted", func(t *testing.T) {
		payload := validPayload()
		payload["kickbacks"] = []interface{}{}
		result, err := Decode(encode(t, payload))
		if err != nil {
			t.Fatalf("fixture must decode: %v", err)
		}
		if err := ValidateRole(RoleRouteRework, result, mainEntity); err != nil {
			t.Fatalf("expected route_rework with no kickback to be accepted, got %v", err)
		}
	})

	t.Run("any kickback rejected", func(t *testing.T) {
		result := mustResult(t)
		if err := ValidateRole(RoleRouteRework, result, mainEntity); err == nil {
			t.Fatalf("expected route_rework with a kickback to be rejected")
		}
	})
}

func TestValidateRole_KickbackRework(t *testing.T) {
	t.Run("with child kickback accepted", func(t *testing.T) {
		result := mustResult(t)
		if err := ValidateRole(RoleKickbackRework, result, mainEntity); err != nil {
			t.Fatalf("expected kickback_rework with a kickback to be accepted, got %v", err)
		}
	})

	t.Run("zero kickbacks rejected", func(t *testing.T) {
		payload := validPayload()
		payload["kickbacks"] = []interface{}{}
		result, err := Decode(encode(t, payload))
		if err != nil {
			t.Fatalf("fixture must decode: %v", err)
		}
		if err := ValidateRole(RoleKickbackRework, result, mainEntity); err == nil {
			t.Fatalf("expected kickback_rework with zero kickbacks to be rejected")
		}
	})
}

func TestValidateRole_BlockedHoldCancelled(t *testing.T) {
	for _, role := range []OutcomeRole{RoleBlocked, RoleHold, RoleCancelled} {
		t.Run(string(role)+"_no_kickback_no_reason_rejected", func(t *testing.T) {
			payload := validPayload()
			payload["kickbacks"] = []interface{}{}
			result, err := Decode(encode(t, payload))
			if err != nil {
				t.Fatalf("fixture must decode: %v", err)
			}
			if err := ValidateRole(role, result, mainEntity); err == nil {
				t.Fatalf("expected %s with no kickback and no no_kickback_reason to be rejected", role)
			}
		})
		t.Run(string(role)+"_no_kickback_with_reason_accepted", func(t *testing.T) {
			payload := validPayload()
			payload["kickbacks"] = []interface{}{}
			payload["no_kickback_reason"] = "External approval pending"
			result, err := Decode(encode(t, payload))
			if err != nil {
				t.Fatalf("fixture must decode: %v", err)
			}
			if err := ValidateRole(role, result, mainEntity); err != nil {
				t.Fatalf("expected %s with a no_kickback_reason to be accepted, got %v", role, err)
			}
		})
		t.Run(string(role)+"_with_kickback_and_no_reason_accepted", func(t *testing.T) {
			result := mustResult(t)
			if err := ValidateRole(role, result, mainEntity); err != nil {
				t.Fatalf("expected %s with a kickback (no reason required) to be accepted, got %v", role, err)
			}
		})
	}
}

func TestValidateRole_ObsoleteRoleRejected(t *testing.T) {
	result := mustResult(t)
	if err := ValidateRole(OutcomeRole("rework"), result, mainEntity); err == nil {
		t.Fatalf("expected obsolete role 'rework' to be rejected")
	}
}

func TestValidateRole_RoleIsExternalInputNotWorkerPayload(t *testing.T) {
	// GateResult has no role/outcome_roles field of its own: the worker cannot
	// select its own validation rules. This is enforced structurally (no such
	// field exists to decode), but this test locks the parser contract: even a
	// worker payload naming a role-shaped field is rejected as unknown.
	payload := validPayload()
	payload["kickbacks"] = []interface{}{}
	payload["role"] = "success"
	if _, err := Decode(encode(t, payload)); err == nil {
		t.Fatalf("expected a worker-supplied role field to be rejected as unknown")
	}
}

func TestValidateRole_MissingMainEntityKeyRejected(t *testing.T) {
	result := mustResult(t)
	if err := ValidateRole(RoleSuccess, result, ""); err == nil {
		t.Fatalf("expected empty main entity key to be rejected")
	}
}

func TestValidateRole_NilResultRejected(t *testing.T) {
	if err := ValidateRole(RoleSuccess, nil, mainEntity); err == nil {
		t.Fatalf("expected nil result to be rejected")
	}
}
