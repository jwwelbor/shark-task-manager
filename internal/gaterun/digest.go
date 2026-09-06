package gaterun

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// ComputeOperationDigest computes the REQ-F-003 operation digest: SHA-256
// over canonical JSON containing the stable bound identity (entity key,
// entity type, source status, gate) and the validated envelope, with object
// keys lexicographically sorted and arrays left in contract (input) order.
//
// envelope is accepted as opaque already-validated JSON bytes (the accepted
// GateResult envelope, or any other structured payload a future caller
// digests) rather than a typed struct, so this package never needs to import
// internal/gateresult.
func ComputeOperationDigest(entityKey, entityType, sourceStatus, gate string, envelope json.RawMessage) (string, error) {
	var envelopeValue interface{}
	if err := json.Unmarshal(envelope, &envelopeValue); err != nil {
		return "", fmt.Errorf("gaterun: decode envelope for digest: %w", err)
	}

	payload := map[string]interface{}{
		"entity_key":    entityKey,
		"entity_type":   entityType,
		"source_status": sourceStatus,
		"gate":          gate,
		"envelope":      envelopeValue,
	}

	canonical, err := canonicalJSON(payload)
	if err != nil {
		return "", fmt.Errorf("gaterun: canonicalize digest payload: %w", err)
	}

	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// canonicalJSON encodes v deterministically: encoding/json already sorts
// map[string]interface{} keys lexicographically on marshal, and preserves
// array/slice order verbatim, so the only extra care needed is disabling
// HTML-escaping (so byte-identical logical content always digests
// identically regardless of what characters it contains) and trimming the
// trailing newline json.Encoder appends.
func canonicalJSON(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
