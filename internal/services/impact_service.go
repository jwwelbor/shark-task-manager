package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Sentinel errors for I-04 ChangeImpactSet shape validation
// (test-plan.md TC-010..TC-013). Wrapped with %w so a JSON-parse failure
// can always be distinguished from a missing/invalid-field failure via
// errors.Is, while the wrapping message still names the specific field.
var (
	// ErrImpactMalformedContent indicates content is not valid JSON.
	ErrImpactMalformedContent = errors.New("impact content is not valid JSON")

	// ErrImpactInvalidShape indicates content parsed as JSON but does not
	// satisfy the minimal I-04 shape (missing/empty/wrong-typed field).
	ErrImpactInvalidShape = errors.New("impact content does not match the minimal I-04 shape")
)

// ImpactNoteRecorder is the subset of NoteService's behavior ImpactService
// delegates note creation to. RecordImpact never persists directly and
// introduces no new Shark database column, table, or relationship type
// (spec.md REQ-NF-001) — it only calls this existing note-creation path,
// following note_service.go's resolveEntityID/EntityRegistry precedent
// rather than duplicating it.
type ImpactNoteRecorder interface {
	AddNote(ctx context.Context, entityType models.EntityType, entityKey string, noteType string, content string, createdBy string) (*models.EntityNote, error)
}

// ImpactService validates a minimal I-04 ChangeImpactSet shape and records
// it as a `reference` note via the existing note-creation path (spec.md
// "Key technical decisions" #1-2).
type ImpactService struct {
	noteSvc ImpactNoteRecorder
}

// NewImpactService creates an ImpactService backed by the given note
// recorder (normally *NoteService).
func NewImpactService(noteSvc ImpactNoteRecorder) (*ImpactService, error) {
	if noteSvc == nil {
		return nil, fmt.Errorf("ImpactService: note recorder must not be nil")
	}
	return &ImpactService{noteSvc: noteSvc}, nil
}

// RecordImpact validates that content is minimally-shaped I-04 JSON
// (source_kind, source_key, and a non-empty affected_artifacts array must
// all be present — spec.md "API / interface contracts"), then records it as
// a `reference` note on entityKey via the existing note-creation path.
//
// Validation happens entirely before any repository call: a malformed or
// incomplete payload never reaches entity-key resolution or note
// persistence (test-plan.md TC-010..TC-013).
//
// entityType is resolved by the caller (the CLI command layer already owns
// full key-to-entity-type detection, including tech-debt/idea/sprint keys
// that internal/keys' detector does not cover) — mirroring
// NoteService.AddNote's own signature rather than re-deriving entity type
// here.
func (s *ImpactService) RecordImpact(ctx context.Context, entityType models.EntityType, entityKey string, content string, createdBy string) (*models.EntityNote, error) {
	if err := validateChangeImpactShape(content); err != nil {
		return nil, err
	}

	note, err := s.noteSvc.AddNote(ctx, entityType, entityKey, string(models.NoteTypeReference), content, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to record impact on %s: %w", entityKey, err)
	}
	return note, nil
}

// validateChangeImpactShape enforces the minimal I-04 shape declared in
// spec.md "API / interface contracts": source_kind, source_key, and a
// non-empty affected_artifacts array must all be present. This is
// intentionally not full I-04 schema enforcement.
func validateChangeImpactShape(content string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return fmt.Errorf("%w: %v", ErrImpactMalformedContent, err)
	}

	if !impactFieldPresent(raw, "source_kind") {
		return fmt.Errorf("%w: missing required field source_kind", ErrImpactInvalidShape)
	}
	if !impactFieldPresent(raw, "source_key") {
		return fmt.Errorf("%w: missing required field source_key", ErrImpactInvalidShape)
	}

	artifacts, ok := raw["affected_artifacts"]
	if !ok || isJSONNullRaw(artifacts) {
		return fmt.Errorf("%w: missing required field affected_artifacts", ErrImpactInvalidShape)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(artifacts, &items); err != nil {
		return fmt.Errorf("%w: field affected_artifacts must be a JSON array, got a different type: %v", ErrImpactInvalidShape, err)
	}
	if len(items) == 0 {
		return fmt.Errorf("%w: field affected_artifacts must not be empty", ErrImpactInvalidShape)
	}

	return nil
}

// impactFieldPresent reports whether field is present in raw and not JSON null.
func impactFieldPresent(raw map[string]json.RawMessage, field string) bool {
	v, ok := raw[field]
	return ok && !isJSONNullRaw(v)
}

func isJSONNullRaw(v json.RawMessage) bool {
	return string(v) == "null"
}
