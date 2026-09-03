package gaterun

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// DeriveSuboperationID derives the deterministic per-target-write
// suboperation ID from the run's operation digest, the operation kind (e.g.
// "finding", "sweep", "impact", "kickback", "gate_summary"), and the stable
// item identity within that kind (a finding fingerprint, sweep class_key,
// impact source kind/key, kickback target entity key, or the literal
// singleton key for the gate-summary record).
//
// This is the shared contract T-E34-F05-003 must call with identical inputs
// to land the same ID for the same target write — see the task spec's
// "shares the suboperation ID contract" note. The three inputs are combined
// with length-prefixing (not a plain delimiter join) so that no combination
// of (operationDigest, operationKind, itemIdentity) can collide with a
// different combination that happens to concatenate to the same bytes.
func DeriveSuboperationID(operationDigest, operationKind, itemIdentity string) string {
	h := sha256.New()
	writeLengthPrefixed(h, operationDigest)
	writeLengthPrefixed(h, operationKind)
	writeLengthPrefixed(h, itemIdentity)
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}

// writeLengthPrefixed feeds a byte-length prefix followed by s into h, so
// concatenation-based collisions across field boundaries are impossible.
func writeLengthPrefixed(h io.Writer, s string) {
	prefix := fmt.Sprintf("%d:", len(s))
	_, _ = h.Write([]byte(prefix))
	_, _ = h.Write([]byte(s))
}
