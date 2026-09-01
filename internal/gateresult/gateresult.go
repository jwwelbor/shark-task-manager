// Package gateresult implements the I-02 GateResult v1 model: the worker-owned
// nested payload of the canonical `kind: final` worker-control envelope,
// carrying gate-specific findings, kickbacks, remediation sweeps (I-03), and
// change impacts (I-04). See
// docs/plan/E34-prompt-and-skill-improvements/architecture.md#i-02-gateresult-v1.
//
// This package owns REQ-F-001 (shared model/parser) and REQ-F-004 (gate
// completeness/role validation) for E34-F05. It deliberately does not own:
//   - The outer worker-control envelope (recommended_outcome, EvidenceRef) —
//     that is parent-observed and lives outside this nested payload.
//   - Persistence, replay, or the sidecar result/operation-state protocol
//     (REQ-F-002, REQ-F-003).
//   - Workflow-schema `result_contract`/`outcome_roles` wiring or `shark next`
//     exposure (REQ-F-006).
//   - Rider/core-runner integration (REQ-F-005).
//
// Kickback.TargetStatus is intentionally validated here only as bounded text.
// It is an opaque value that must be validated against the *target* entity's
// configured workflow outcomes before a transition is applied; that check
// belongs to the persistence coordinator (REQ-F-002), not this model — see
// the project convention against hardcoding status names in shared models.
package gateresult

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Bounds shared by every GateResult v1 field, per architecture.md I-02.
const (
	// SchemaVersion1 is the only currently accepted schema_version value.
	SchemaVersion1 = 1

	// SummaryMaxBytes bounds any free-text summary/statement/reason field.
	SummaryMaxBytes = 1000
	// PointerMaxBytes bounds any evidence/reference pointer field.
	PointerMaxBytes = 2048
	// IdentityMaxBytes bounds short stable identifier fields (keys,
	// fingerprints, severities, statuses). Not independently specified by
	// architecture.md; chosen to match the existing Question model's
	// identity-field convention (internal/models/question.go).
	IdentityMaxBytes = 256
	// MaxCollectionItems bounds every array field in the GateResult v1 shape.
	MaxCollectionItems = 100
	// MaxTotalBytes bounds the canonical (re-marshaled) size of the entire
	// nested GateResult object.
	MaxTotalBytes = 256 * 1024
)

// Disposition is the closed set of Finding disposition values.
type Disposition string

const (
	DispositionOpen                 Disposition = "open"
	DispositionFixed                Disposition = "fixed"
	DispositionAlreadyDispositioned Disposition = "already_dispositioned"
	DispositionSeverityConflict     Disposition = "severity_conflict"
	DispositionNotReproducible      Disposition = "not_reproducible"
)

// OutcomeRole is the closed set of parent-owned semantic roles a workflow
// step's configured outcome can carry. The role selects validation rules; it
// never selects a transition (the opaque outcome key does that).
type OutcomeRole string

const (
	RoleSuccess        OutcomeRole = "success"
	RoleRouteRework    OutcomeRole = "route_rework"
	RoleKickbackRework OutcomeRole = "kickback_rework"
	RoleBlocked        OutcomeRole = "blocked"
	RoleHold           OutcomeRole = "hold"
	RoleCancelled      OutcomeRole = "cancelled"
)

// ErrorClass classifies a validation failure without echoing rejected
// content, per REQ-NF-001 ("report validation errors by field and class
// without echoing rejected secrets or entire worker output").
type ErrorClass string

const (
	ErrorClassShape            ErrorClass = "shape"
	ErrorClassSchema           ErrorClass = "schema"
	ErrorClassUnknownField     ErrorClass = "unknown_field"
	ErrorClassBounds           ErrorClass = "bounds"
	ErrorClassDuplicate        ErrorClass = "duplicate"
	ErrorClassForbiddenContent ErrorClass = "forbidden_content"
	ErrorClassRole             ErrorClass = "role"
)

// ValidationError is the one error type this package returns. Field and
// Class are always safe to log; Message never echoes the rejected value.
type ValidationError struct {
	Field   string
	Class   ErrorClass
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("gate result (%s): %s", e.Class, e.Message)
	}
	return fmt.Sprintf("gate result %s (%s): %s", e.Field, e.Class, e.Message)
}

func newValidationError(field string, class ErrorClass, message string) *ValidationError {
	return &ValidationError{Field: field, Class: class, Message: message}
}

// Kickback is one I-02 kickback directive. TargetStatus is opaque; see the
// package doc comment for what this model does and does not validate.
type Kickback struct {
	EntityKey    string `json:"entity_key"`
	TargetStatus string `json:"target_status"`
	Reason       string `json:"reason"`
}

// Finding is one I-02 review/QA finding.
type Finding struct {
	Severity           string      `json:"severity"`
	ClassKey           string      `json:"class_key"`
	ClassStatement     string      `json:"class_statement"`
	Fingerprint        string      `json:"fingerprint"`
	AffectedIDs        []string    `json:"affected_ids,omitempty"`
	EvidencePointers   []string    `json:"evidence_pointers,omitempty"`
	Disposition        Disposition `json:"disposition"`
	DispositionPointer string      `json:"disposition_pointer,omitempty"`
}

// SweepGuard is the I-03 guard describing how a completed sweep's fix class
// is structurally prevented from recurring.
type SweepGuard struct {
	Kind                  string `json:"kind"`
	ImplementationPointer string `json:"implementation_pointer"`
	CounterfactualPointer string `json:"counterfactual_pointer"`
	Status                string `json:"status"`
}

// SweepInstance is one matched site within an I-03 DefectClassSweep.
type SweepInstance struct {
	Fingerprint      string   `json:"fingerprint"`
	SitePointer      string   `json:"site_pointer"`
	Disposition      string   `json:"disposition"`
	EvidencePointers []string `json:"evidence_pointers,omitempty"`
}

// DefectClassSweep is the I-03 DefectClassSweep v1 payload.
type DefectClassSweep struct {
	ClassKey           string          `json:"class_key"`
	ClassStatement     string          `json:"class_statement"`
	SearchScope        []string        `json:"search_scope,omitempty"`
	PriorDesigns       []string        `json:"prior_designs,omitempty"`
	SearchedCount      int             `json:"searched_count"`
	MatchingCount      int             `json:"matching_count"`
	Instances          []SweepInstance `json:"instances,omitempty"`
	FixedCount         int             `json:"fixed_count"`
	DispositionedCount int             `json:"dispositioned_count"`
	OpenCount          int             `json:"open_count"`
	Guard              SweepGuard      `json:"guard"`
	Status             string          `json:"status"`
}

// AffectedArtifact is one artifact invalidated by a change, within an I-04
// ChangeImpactSet.
type AffectedArtifact struct {
	Path            string `json:"path"`
	ArtifactKind    string `json:"artifact_kind"`
	InvalidatedText string `json:"invalidated_text"`
	Disposition     string `json:"disposition"`
	FollowUpKey     string `json:"follow_up_key,omitempty"`
}

// AffectedConsumer is one production caller accounted for by an I-04
// ChangeImpactSet.
type AffectedConsumer struct {
	EntityKey             string   `json:"entity_key"`
	ProductionCallerPath  string   `json:"production_caller_path"`
	AffectedIDs           []string `json:"affected_ids,omitempty"`
	RegressionTestPointer string   `json:"regression_test_pointer"`
}

// SharedName is one owned name and its checked producer/consumer usages,
// within an I-04 ChangeImpactSet.
type SharedName struct {
	Name   string   `json:"name"`
	Usages []string `json:"usages,omitempty"`
}

// ChangeImpactSet is the I-04 ChangeImpactSet v1 payload.
type ChangeImpactSet struct {
	SourceKind        string             `json:"source_kind"`
	SourceKey         string             `json:"source_key"`
	SourcePointer     string             `json:"source_pointer"`
	ChangeSummary     string             `json:"change_summary"`
	AffectedArtifacts []AffectedArtifact `json:"affected_artifacts,omitempty"`
	AffectedConsumers []AffectedConsumer `json:"affected_consumers,omitempty"`
	SharedNames       []SharedName       `json:"shared_names,omitempty"`
	Verification      []string           `json:"verification,omitempty"`
	Status            string             `json:"status"`
}

// GateResult is the I-02 GateResult v1 nested payload. Entity, source status,
// observed gate, outer configured outcome, and outer evidence are
// deliberately absent (they are parent-observed, owned by the outer
// worker-control envelope).
type GateResult struct {
	SchemaVersion     int                `json:"schema_version"`
	Summary           string             `json:"summary"`
	Findings          []Finding          `json:"findings,omitempty"`
	Kickbacks         []Kickback         `json:"kickbacks,omitempty"`
	RemediationSweeps []DefectClassSweep `json:"remediation_sweeps,omitempty"`
	ChangeImpacts     []ChangeImpactSet  `json:"change_impacts,omitempty"`
	NoKickbackReason  string             `json:"no_kickback_reason,omitempty"`
}

// Decode parses and fully structurally validates a candidate GateResult v1
// document. It rejects unknown top-level fields (which catches the outer
// envelope's gate/outcome/evidence/recommended_outcome fields leaking into
// the nested payload — the "second envelope"/alias case), malformed JSON
// shapes, and trailing content after the object.
func Decode(data []byte) (*GateResult, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var result GateResult
	if err := dec.Decode(&result); err != nil {
		if field, ok := unknownFieldName(err); ok {
			return nil, newValidationError(field, ErrorClassUnknownField, "is not a recognized GateResult v1 field")
		}
		return nil, newValidationError("", ErrorClassShape, "must be a well-formed JSON object matching the GateResult v1 shape")
	}
	if dec.More() {
		return nil, newValidationError("", ErrorClassShape, "must contain exactly one JSON object; a second top-level value is not permitted")
	}

	if err := result.Validate(); err != nil {
		return nil, err
	}
	return &result, nil
}

// unknownFieldName extracts the rejected field name from the stdlib
// DisallowUnknownFields error message. The message never contains rejected
// values, only the field name, so surfacing it does not violate REQ-NF-001.
func unknownFieldName(err error) (string, bool) {
	const marker = `json: unknown field "`
	msg := err.Error()
	idx := strings.Index(msg, marker)
	if idx < 0 {
		return "", false
	}
	rest := msg[idx+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

// Validate checks every REQ-F-001 structural invariant: schema version,
// bounded text, bounded/duplicate-free collections, and the aggregate size
// bound. It does not evaluate role-dependent completeness rules; call
// ValidateRole for those (REQ-F-004).
func (r *GateResult) Validate() error {
	if r.SchemaVersion != SchemaVersion1 {
		return newValidationError("schema_version", ErrorClassSchema, "must equal 1")
	}
	if err := boundedText("summary", r.Summary, 1, SummaryMaxBytes); err != nil {
		return err
	}

	if err := boundCollection("findings", len(r.Findings)); err != nil {
		return err
	}
	seenFingerprints := make(map[string]struct{}, len(r.Findings))
	for i, f := range r.Findings {
		if err := f.validate(i); err != nil {
			return err
		}
		if _, exists := seenFingerprints[f.Fingerprint]; exists {
			return newValidationError("findings", ErrorClassDuplicate, "fingerprint must be unique within the result")
		}
		seenFingerprints[f.Fingerprint] = struct{}{}
	}

	if err := boundCollection("kickbacks", len(r.Kickbacks)); err != nil {
		return err
	}
	seenKickbacks := make(map[string]struct{}, len(r.Kickbacks))
	for i, k := range r.Kickbacks {
		if err := k.validate(i); err != nil {
			return err
		}
		if _, exists := seenKickbacks[k.EntityKey]; exists {
			return newValidationError("kickbacks", ErrorClassDuplicate, "entity_key must be unique within the result")
		}
		seenKickbacks[k.EntityKey] = struct{}{}
	}

	if err := boundCollection("remediation_sweeps", len(r.RemediationSweeps)); err != nil {
		return err
	}
	seenSweeps := make(map[string]struct{}, len(r.RemediationSweeps))
	for i, s := range r.RemediationSweeps {
		if err := s.validate(i); err != nil {
			return err
		}
		if _, exists := seenSweeps[s.ClassKey]; exists {
			return newValidationError("remediation_sweeps", ErrorClassDuplicate, "class_key must be unique within the result")
		}
		seenSweeps[s.ClassKey] = struct{}{}
	}

	if err := boundCollection("change_impacts", len(r.ChangeImpacts)); err != nil {
		return err
	}
	seenImpacts := make(map[[2]string]struct{}, len(r.ChangeImpacts))
	for i, c := range r.ChangeImpacts {
		if err := c.validate(i); err != nil {
			return err
		}
		key := [2]string{c.SourceKind, c.SourceKey}
		if _, exists := seenImpacts[key]; exists {
			return newValidationError("change_impacts", ErrorClassDuplicate, "source_kind plus source_key must be unique within the result")
		}
		seenImpacts[key] = struct{}{}
	}

	if r.NoKickbackReason != "" {
		if err := boundedText("no_kickback_reason", r.NoKickbackReason, 1, SummaryMaxBytes); err != nil {
			return err
		}
	}

	encoded, err := json.Marshal(r)
	if err != nil {
		return newValidationError("", ErrorClassShape, "could not be canonically encoded")
	}
	if len(encoded) > MaxTotalBytes {
		return newValidationError("", ErrorClassBounds, "must be at most "+strconv.Itoa(MaxTotalBytes)+" bytes canonically encoded")
	}
	return nil
}

func (f Finding) validate(index int) error {
	prefix := fmt.Sprintf("findings[%d]", index)
	if err := boundedText(prefix+".severity", f.Severity, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".class_key", f.ClassKey, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".class_statement", f.ClassStatement, 1, SummaryMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".fingerprint", f.Fingerprint, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundCollection(prefix+".affected_ids", len(f.AffectedIDs)); err != nil {
		return err
	}
	for i, id := range f.AffectedIDs {
		if err := boundedText(fmt.Sprintf("%s.affected_ids[%d]", prefix, i), id, 1, IdentityMaxBytes); err != nil {
			return err
		}
	}
	if err := boundCollection(prefix+".evidence_pointers", len(f.EvidencePointers)); err != nil {
		return err
	}
	for i, p := range f.EvidencePointers {
		if err := boundedText(fmt.Sprintf("%s.evidence_pointers[%d]", prefix, i), p, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	switch f.Disposition {
	case DispositionOpen, DispositionFixed, DispositionAlreadyDispositioned, DispositionSeverityConflict, DispositionNotReproducible:
	default:
		return newValidationError(prefix+".disposition", ErrorClassSchema, "must be one of open, fixed, already_dispositioned, severity_conflict, not_reproducible")
	}
	requiresPointer := f.Disposition == DispositionAlreadyDispositioned || f.Disposition == DispositionSeverityConflict
	if requiresPointer {
		if err := boundedText(prefix+".disposition_pointer", f.DispositionPointer, 1, PointerMaxBytes); err != nil {
			return newValidationError(prefix+".disposition_pointer", ErrorClassSchema, "is required and must point to a durable decision when disposition is already_dispositioned or severity_conflict")
		}
	} else if f.DispositionPointer != "" {
		if err := boundedText(prefix+".disposition_pointer", f.DispositionPointer, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	return nil
}

func (k Kickback) validate(index int) error {
	prefix := fmt.Sprintf("kickbacks[%d]", index)
	if err := boundedText(prefix+".entity_key", k.EntityKey, 1, IdentityMaxBytes); err != nil {
		return err
	}
	// target_status is opaque here: only bounded-text shape is checked. See
	// the package doc comment — workflow membership validation against the
	// target entity's configured outcomes is deferred to the persistence
	// coordinator (REQ-F-002).
	if err := boundedText(prefix+".target_status", k.TargetStatus, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".reason", k.Reason, 1, SummaryMaxBytes); err != nil {
		return err
	}
	return nil
}

func (s DefectClassSweep) validate(index int) error {
	prefix := fmt.Sprintf("remediation_sweeps[%d]", index)
	if err := boundedText(prefix+".class_key", s.ClassKey, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".class_statement", s.ClassStatement, 1, SummaryMaxBytes); err != nil {
		return err
	}
	if err := boundCollection(prefix+".search_scope", len(s.SearchScope)); err != nil {
		return err
	}
	for i, v := range s.SearchScope {
		if err := boundedText(fmt.Sprintf("%s.search_scope[%d]", prefix, i), v, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	if err := boundCollection(prefix+".prior_designs", len(s.PriorDesigns)); err != nil {
		return err
	}
	for i, v := range s.PriorDesigns {
		if err := boundedText(fmt.Sprintf("%s.prior_designs[%d]", prefix, i), v, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	if s.SearchedCount < 0 || s.MatchingCount < 0 || s.FixedCount < 0 || s.DispositionedCount < 0 || s.OpenCount < 0 {
		return newValidationError(prefix, ErrorClassSchema, "counts must be non-negative")
	}
	if s.MatchingCount > s.SearchedCount {
		return newValidationError(prefix+".matching_count", ErrorClassSchema, "must not exceed searched_count")
	}
	if s.FixedCount+s.DispositionedCount+s.OpenCount != s.MatchingCount {
		return newValidationError(prefix, ErrorClassSchema, "fixed_count plus dispositioned_count plus open_count must equal matching_count")
	}
	if err := boundCollection(prefix+".instances", len(s.Instances)); err != nil {
		return err
	}
	seenFingerprints := make(map[string]struct{}, len(s.Instances))
	for i, inst := range s.Instances {
		if err := inst.validate(prefix, i); err != nil {
			return err
		}
		if _, exists := seenFingerprints[inst.Fingerprint]; exists {
			return newValidationError(prefix+".instances", ErrorClassDuplicate, "fingerprint must be unique within the sweep")
		}
		seenFingerprints[inst.Fingerprint] = struct{}{}
	}
	if err := s.Guard.validate(prefix); err != nil {
		return err
	}
	switch s.Status {
	case "open":
	case "complete":
		if s.OpenCount != 0 {
			return newValidationError(prefix+".status", ErrorClassSchema, "a complete sweep must have open_count 0")
		}
		if len(s.Instances) != s.MatchingCount {
			return newValidationError(prefix+".instances", ErrorClassSchema, "a complete sweep must represent every matching instance exactly once")
		}
		if s.Guard.Status != "verified" {
			return newValidationError(prefix+".guard.status", ErrorClassSchema, "a complete sweep requires a verified guard")
		}
	default:
		return newValidationError(prefix+".status", ErrorClassSchema, "must be open or complete")
	}
	return nil
}

func (i SweepInstance) validate(prefix string, index int) error {
	p := fmt.Sprintf("%s.instances[%d]", prefix, index)
	if err := boundedText(p+".fingerprint", i.Fingerprint, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".site_pointer", i.SitePointer, 1, PointerMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".disposition", i.Disposition, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundCollection(p+".evidence_pointers", len(i.EvidencePointers)); err != nil {
		return err
	}
	for j, ep := range i.EvidencePointers {
		if err := boundedText(fmt.Sprintf("%s.evidence_pointers[%d]", p, j), ep, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	return nil
}

func (g SweepGuard) validate(prefix string) error {
	p := prefix + ".guard"
	if err := boundedText(p+".kind", g.Kind, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".implementation_pointer", g.ImplementationPointer, 1, PointerMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".counterfactual_pointer", g.CounterfactualPointer, 1, PointerMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".status", g.Status, 1, IdentityMaxBytes); err != nil {
		return err
	}
	return nil
}

// ValidateChangeImpactSet validates a standalone I-04 ChangeImpactSet v1
// payload outside of a GateResult (REQ-F-006's `shark impact record` ADR
// adoption boundary). It reuses the exact same field-level bounds and
// required-shape checks GateResult.Validate applies to each entry of its
// change_impacts collection; standalone callers get one validation error
// class, not a second parser.
func ValidateChangeImpactSet(c ChangeImpactSet) error {
	return c.validate(0)
}

func (c ChangeImpactSet) validate(index int) error {
	prefix := fmt.Sprintf("change_impacts[%d]", index)
	// source_kind's literal token set (question/tech_debt/change_card/adr/...)
	// is not yet fixed by any canonical spec text; only bounded-text shape is
	// checked here rather than inventing enum tokens ahead of that decision.
	if err := boundedText(prefix+".source_kind", c.SourceKind, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".source_key", c.SourceKey, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".source_pointer", c.SourcePointer, 1, PointerMaxBytes); err != nil {
		return err
	}
	if err := boundedText(prefix+".change_summary", c.ChangeSummary, 1, SummaryMaxBytes); err != nil {
		return err
	}
	if err := boundCollection(prefix+".affected_artifacts", len(c.AffectedArtifacts)); err != nil {
		return err
	}
	for i, a := range c.AffectedArtifacts {
		if err := a.validate(prefix, i); err != nil {
			return err
		}
	}
	if err := boundCollection(prefix+".affected_consumers", len(c.AffectedConsumers)); err != nil {
		return err
	}
	for i, a := range c.AffectedConsumers {
		if err := a.validate(prefix, i); err != nil {
			return err
		}
	}
	if err := boundCollection(prefix+".shared_names", len(c.SharedNames)); err != nil {
		return err
	}
	for i, sn := range c.SharedNames {
		if err := sn.validate(prefix, i); err != nil {
			return err
		}
	}
	if err := boundCollection(prefix+".verification", len(c.Verification)); err != nil {
		return err
	}
	for i, v := range c.Verification {
		if err := boundedText(fmt.Sprintf("%s.verification[%d]", prefix, i), v, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	switch c.Status {
	case "accounted", "incomplete":
	default:
		return newValidationError(prefix+".status", ErrorClassSchema, "must be accounted or incomplete")
	}
	return nil
}

func (a AffectedArtifact) validate(prefix string, index int) error {
	p := fmt.Sprintf("%s.affected_artifacts[%d]", prefix, index)
	if err := boundedText(p+".path", a.Path, 1, PointerMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".artifact_kind", a.ArtifactKind, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".invalidated_text", a.InvalidatedText, 1, SummaryMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".disposition", a.Disposition, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if a.FollowUpKey != "" {
		if err := boundedText(p+".follow_up_key", a.FollowUpKey, 1, IdentityMaxBytes); err != nil {
			return err
		}
	}
	return nil
}

func (a AffectedConsumer) validate(prefix string, index int) error {
	p := fmt.Sprintf("%s.affected_consumers[%d]", prefix, index)
	if err := boundedText(p+".entity_key", a.EntityKey, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText(p+".production_caller_path", a.ProductionCallerPath, 1, PointerMaxBytes); err != nil {
		return err
	}
	if err := boundCollection(p+".affected_ids", len(a.AffectedIDs)); err != nil {
		return err
	}
	for i, id := range a.AffectedIDs {
		if err := boundedText(fmt.Sprintf("%s.affected_ids[%d]", p, i), id, 1, IdentityMaxBytes); err != nil {
			return err
		}
	}
	if err := boundedText(p+".regression_test_pointer", a.RegressionTestPointer, 1, PointerMaxBytes); err != nil {
		return err
	}
	return nil
}

func (sn SharedName) validate(prefix string, index int) error {
	p := fmt.Sprintf("%s.shared_names[%d]", prefix, index)
	if err := boundedText(p+".name", sn.Name, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundCollection(p+".usages", len(sn.Usages)); err != nil {
		return err
	}
	for i, u := range sn.Usages {
		if err := boundedText(fmt.Sprintf("%s.usages[%d]", p, i), u, 1, PointerMaxBytes); err != nil {
			return err
		}
	}
	return nil
}

func boundCollection(field string, n int) error {
	if n > MaxCollectionItems {
		return newValidationError(field, ErrorClassBounds, "must contain at most "+strconv.Itoa(MaxCollectionItems)+" entries")
	}
	return nil
}
