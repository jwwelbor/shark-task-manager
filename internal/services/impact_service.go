package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

const (
	impactRecordKind = "change_impact"
	impactNoteType   = "reference"
)

// ImpactNoteWriter is the narrow persistence dependency required to record a
// validated change impact. *NoteService satisfies it directly.
type ImpactNoteWriter interface {
	AddNoteWithMetadata(ctx context.Context, entityType models.EntityType, entityKey string, noteType string, content string, createdBy string, metadata string) (*models.EntityNote, error)
}

// RecordImpactInput contains the parent-asserted identity and parsed I-04
// payload supplied by the CLI boundary.
type RecordImpactInput struct {
	EntityType    models.EntityType
	EntityKey     string
	SourceKind    string
	SourceKey     string
	SourcePointer string
	Impact        gateresult.ChangeImpactSet
}

// ImpactRecord is the durable result of recording an I-04 change impact.
type ImpactRecord struct {
	EntityType    models.EntityType
	EntityKey     string
	SourceKind    string
	SourceKey     string
	SourcePointer string
	Status        string
	NoteID        int64
}

// ImpactService owns I-04 identity reconciliation, validation, metadata
// construction, and note persistence. Transport concerns (flag parsing, file
// reads, and JSON decoding) remain at the command boundary.
type ImpactService struct {
	notes ImpactNoteWriter
}

// NewImpactService creates an ImpactService with the supplied note writer.
func NewImpactService(notes ImpactNoteWriter) (*ImpactService, error) {
	if notes == nil {
		return nil, fmt.Errorf("ImpactService: note writer must not be nil")
	}
	return &ImpactService{notes: notes}, nil
}

// ReconcileAndValidateImpact applies the parent-owned identity contract and
// validates the resulting I-04 payload without touching persistence. Command
// callers use it before resolving a database-backed writer, so malformed
// worker input fails closed before any persistence dependency is initialized.
func ReconcileAndValidateImpact(impact *gateresult.ChangeImpactSet, sourceKind, sourceKey, sourcePointer string) error {
	if err := reconcileImpactIdentity(impact, sourceKind, sourceKey, sourcePointer); err != nil {
		return err
	}
	if err := gateresult.ValidateChangeImpactSet(*impact); err != nil {
		return fmt.Errorf("invalid I-04 ChangeImpactSet: %w", err)
	}
	return nil
}

// Record reconciles, validates, and persists an I-04 change impact.
func (s *ImpactService) Record(ctx context.Context, input RecordImpactInput) (*ImpactRecord, error) {
	if err := ReconcileAndValidateImpact(&input.Impact, input.SourceKind, input.SourceKey, input.SourcePointer); err != nil {
		return nil, err
	}

	content := fmt.Sprintf("[%s/%s] %s (%s)", input.Impact.SourceKind, input.Impact.SourceKey, input.Impact.ChangeSummary, input.Impact.Status)
	metadata := map[string]interface{}{
		"record_kind":    impactRecordKind,
		"source_kind":    input.Impact.SourceKind,
		"source_key":     input.Impact.SourceKey,
		"source_pointer": input.Impact.SourcePointer,
		"status":         input.Impact.Status,
	}
	encodedMeta, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("encode note metadata: %w", err)
	}

	note, err := s.notes.AddNoteWithMetadata(ctx, input.EntityType, input.EntityKey, impactNoteType, content, "", string(encodedMeta))
	if err != nil {
		return nil, fmt.Errorf("persist change-impact note on %s %s: %w", input.EntityType, input.EntityKey, err)
	}
	return &ImpactRecord{input.EntityType, input.EntityKey, input.Impact.SourceKind, input.Impact.SourceKey, input.Impact.SourcePointer, input.Impact.Status, note.ID}, nil
}

func reconcileImpactIdentity(impact *gateresult.ChangeImpactSet, sourceKind, sourceKey, sourcePointer string) error {
	if err := reconcileImpactField("source_kind", sourceKind, &impact.SourceKind); err != nil {
		return err
	}
	if err := reconcileImpactField("source_key", sourceKey, &impact.SourceKey); err != nil {
		return err
	}
	return reconcileImpactField("source_pointer", sourcePointer, &impact.SourcePointer)
}

func reconcileImpactField(field, parentValue string, fileValue *string) error {
	if strings.TrimSpace(*fileValue) == "" {
		*fileValue = parentValue
		return nil
	}
	if *fileValue != parentValue {
		return fmt.Errorf("--%s=%q conflicts with %s=%q in --impact-file; the flag is the parent-asserted identity and must match", strings.ReplaceAll(field, "_", "-"), parentValue, field, *fileValue)
	}
	return nil
}
