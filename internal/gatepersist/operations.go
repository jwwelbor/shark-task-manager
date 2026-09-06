package gatepersist

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
)

// Operation kinds, per architecture.md step 7's suboperation identity list.
const (
	kindGateSummary = "gate_summary"
	kindFinding     = "finding"
	kindSweep       = "sweep"
	kindImpact      = "impact"
	kindKickback    = "kickback"
)

// gateSummaryIdentity is the literal singleton item identity for the one
// gate-summary/evidence note per run.
const gateSummaryIdentity = "gate_summary"

// Note types this package writes. Sweep and impact notes intentionally reuse
// the "reference" type with bounded record_kind metadata (REQ-F-002); no new
// note enum is introduced.
const (
	noteTypeReview        = "review"
	noteTypeReviewFinding = "review-finding"
	noteTypeReference     = "reference"
)

// record_kind values distinguishing "reference" notes without a new note type.
const (
	recordKindSweep  = "remediation_sweep"
	recordKindImpact = "change_impact"
)

// Metadata field names shared by every note this package writes. These are
// the cross-task contract T-E34-F05-002's reconciliation read path (via this
// package's own TargetRecordReader implementation, see reconcile.go) depends
// on to find a suboperation's durable record again after a crash.
const (
	metaRunID           = "run_id"
	metaSuboperationID  = "suboperation_id"
	metaOperationDigest = "operation_digest"
	metaRecordKind      = "record_kind"
	metaGate            = "gate"
	metaContentDigest   = "content_digest"
	metaParentSession   = "parent_session"
	metaOutcomeKey      = "outcome_key"
	metaRole            = "role"
	metaEvidence        = "evidence"
)

// note-only finding metadata fields (REQ-F-002's required finding metadata).
const (
	metaSeverity           = "severity"
	metaClassKey           = "class_key"
	metaClassStatement     = "class_statement"
	metaFingerprint        = "fingerprint"
	metaAffectedIDs        = "affected_ids"
	metaDisposition        = "disposition"
	metaDispositionPointer = "disposition_pointer"
	metaSourceKind         = "source_kind"
	metaSourceKey          = "source_key"
)

// operation is one target write in the REQ-F-002 persistence order: either a
// note on the main entity or a kickback transition on a different entity.
type operation struct {
	kind         string
	itemIdentity string

	// note-write fields (kind != kindKickback)
	noteType string
	content  string
	metadata map[string]interface{}

	// kickback fields (kind == kindKickback)
	kickback *gateresult.Kickback
}

// suboperationID derives this operation's stable ID via gaterun's shared
// contract, given the run's operation digest.
func (op operation) suboperationID(operationDigest string) string {
	return gaterun.DeriveSuboperationID(operationDigest, op.kind, op.itemIdentity)
}

// contentDigest is a bounded, deterministic hash of everything about this
// operation that must not change across a replay under the same
// suboperation ID. A mismatch on resume/reconciliation is a conflicting
// replay and must fail closed (REQ-F-003), never silently apply different
// content under an ID already recorded complete.
func (op operation) contentDigest() string {
	h := sha256.New()
	_, _ = h.Write([]byte(op.kind))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(op.itemIdentity))
	_, _ = h.Write([]byte{0})
	if op.kind == kindKickback {
		_, _ = h.Write([]byte(op.kickback.EntityKey))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(op.kickback.TargetStatus))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(op.kickback.Reason))
	} else {
		_, _ = h.Write([]byte(op.noteType))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(op.content))
		// Metadata order must not affect the digest, so marshal via the
		// canonical (lexicographically key-sorted) JSON encoding.
		encoded, err := json.Marshal(op.metadata)
		if err == nil {
			_, _ = h.Write(encoded)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// buildOperations derives the full REQ-F-002 ordered operation list from a
// validated GateResult: gate-summary, findings, sweeps, impacts, then
// kickbacks. The order and item identities are exactly architecture.md
// step 7's contract.
func buildOperations(result *gateresult.GateResult, summary string) []operation {
	ops := make([]operation, 0, 1+len(result.Findings)+len(result.RemediationSweeps)+len(result.ChangeImpacts)+len(result.Kickbacks))

	ops = append(ops, operation{
		kind:         kindGateSummary,
		itemIdentity: gateSummaryIdentity,
		noteType:     noteTypeReview,
		content:      summary,
		metadata: map[string]interface{}{
			metaRecordKind: kindGateSummary,
		},
	})

	for _, f := range result.Findings {
		ops = append(ops, operation{
			kind:         kindFinding,
			itemIdentity: f.Fingerprint,
			noteType:     noteTypeReviewFinding,
			content:      findingContent(f),
			metadata:     findingMetadata(f),
		})
	}

	for _, s := range result.RemediationSweeps {
		ops = append(ops, operation{
			kind:         kindSweep,
			itemIdentity: s.ClassKey,
			noteType:     noteTypeReference,
			content:      sweepContent(s),
			metadata: map[string]interface{}{
				metaRecordKind:   recordKindSweep,
				metaClassKey:     s.ClassKey,
				"status":         s.Status,
				"matching_count": s.MatchingCount,
				"open_count":     s.OpenCount,
			},
		})
	}

	for _, c := range result.ChangeImpacts {
		ops = append(ops, operation{
			kind:         kindImpact,
			itemIdentity: impactIdentity(c.SourceKind, c.SourceKey),
			noteType:     noteTypeReference,
			content:      impactContent(c),
			metadata: map[string]interface{}{
				metaRecordKind: recordKindImpact,
				metaSourceKind: c.SourceKind,
				metaSourceKey:  c.SourceKey,
				"status":       c.Status,
			},
		})
	}

	for i := range result.Kickbacks {
		k := result.Kickbacks[i]
		ops = append(ops, operation{
			kind:         kindKickback,
			itemIdentity: k.EntityKey,
			kickback:     &k,
		})
	}

	return ops
}

// impactIdentity combines source_kind and source_key into one stable item
// identity. A NUL separator (never valid in either bounded text field, both
// of which are validated non-empty printable identity text) prevents
// ("a","bc") and ("ab","c") from colliding.
func impactIdentity(sourceKind, sourceKey string) string {
	return sourceKind + "\x00" + sourceKey
}

func findingContent(f gateresult.Finding) string {
	return fmt.Sprintf("[%s] %s", f.Severity, f.ClassStatement)
}

func findingMetadata(f gateresult.Finding) map[string]interface{} {
	m := map[string]interface{}{
		metaRecordKind:     kindFinding,
		metaSeverity:       f.Severity,
		metaClassKey:       f.ClassKey,
		metaClassStatement: f.ClassStatement,
		metaFingerprint:    f.Fingerprint,
		metaDisposition:    string(f.Disposition),
	}
	if len(f.AffectedIDs) > 0 {
		m[metaAffectedIDs] = f.AffectedIDs
	}
	if f.DispositionPointer != "" {
		m[metaDispositionPointer] = f.DispositionPointer
	}
	return m
}

func sweepContent(s gateresult.DefectClassSweep) string {
	return fmt.Sprintf("[%s] %s (matching=%d fixed=%d dispositioned=%d open=%d)",
		s.Status, s.ClassStatement, s.MatchingCount, s.FixedCount, s.DispositionedCount, s.OpenCount)
}

func impactContent(c gateresult.ChangeImpactSet) string {
	return fmt.Sprintf("[%s/%s] %s (%s)", c.SourceKind, c.SourceKey, c.ChangeSummary, c.Status)
}

// summaryFrom builds the gate-summary note content from the GateResult's
// bounded summary field plus the observed gate, so the note is
// self-describing without embedding worker prose beyond what was already
// validated bounded text.
func summaryFrom(gate, summary string) string {
	return strings.TrimSpace(fmt.Sprintf("[%s] %s", gate, summary))
}
