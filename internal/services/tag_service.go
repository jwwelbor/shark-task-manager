package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	tagrepo "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
)

// tagServiceTracerName is the OTel tracer name for TagService.
// Tracer is fetched lazily per-call so tests can install a recording
// provider before invoking service methods.
const tagServiceTracerName = "shark/services/tag"

// TagService owns the vocabulary business rules for the managed tag set.
// CLI and (future) HTTP handlers consume it; it consumes F01 repositories
// and the F02 gate.
//
// REQ-F-001: lives in internal/services/tag_service.go; depends only on
// interfaces from internal/repository/tag (F01) and internal/auth/maintainer
// (F02).
type TagService struct {
	tagRepo       tagrepo.TagRepositoryInterface
	entityTagRepo tagrepo.EntityTagRepositoryInterface
	gate          maintainer.Gate
	tracer        trace.Tracer // optional; nil falls back to global tracer
}

// NewTagService constructs a TagService. All three dependencies are required;
// the constructor panics on nil per the architecture's constructor rules (AC-T1).
func NewTagService(
	tagRepo tagrepo.TagRepositoryInterface,
	entityTagRepo tagrepo.EntityTagRepositoryInterface,
	gate maintainer.Gate,
) *TagService {
	requireNonNil(tagRepo, "TagService requires a non-nil TagRepositoryInterface")
	requireNonNil(entityTagRepo, "TagService requires a non-nil EntityTagRepositoryInterface")
	requireNonNil(gate, "TagService requires a non-nil Gate")
	return &TagService{
		tagRepo:       tagRepo,
		entityTagRepo: entityTagRepo,
		gate:          gate,
	}
}

// SetTracer sets the OpenTelemetry tracer used by all methods. When nil, the
// global OTel tracer is used. Use this in tests to install an in-memory exporter.
func (s *TagService) SetTracer(t trace.Tracer) {
	s.tracer = t
}

// getTracer returns the configured tracer or the OTel global tracer.
func (s *TagService) getTracer() trace.Tracer {
	if s.tracer != nil {
		return s.tracer
	}
	return otel.Tracer(tagServiceTracerName)
}

// ---------------------------------------------------------------------------
// ValidateName
// ---------------------------------------------------------------------------

// ValidateName normalizes raw input (TrimSpace + ToLower) and validates the
// result against models.ValidateTagName. Returns the normalized name on
// success, or a *ValidationError on failure.
//
// This is the single entry point for name validation in the TagService.
// F04's AttachByNames will call this method rather than duplicating the logic.
// (REQ-NF-004, AC-T8)
func (s *TagService) ValidateName(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if err := models.ValidateTagName(normalized); err != nil {
		return "", &ValidationError{
			Field:   "tag name",
			Message: "must match ^[a-z0-9][a-z0-9-]{0,63}$ (lowercase ASCII letters, digits, hyphens; must start with letter or digit; max 64 characters)",
		}
	}
	return normalized, nil
}

// validateTagNameWithField is like ValidateName but lets the caller specify
// the field name (used by RenameTag for "old name"/"new name").
func (s *TagService) validateTagNameWithField(raw, fieldName string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if err := models.ValidateTagName(normalized); err != nil {
		return "", &ValidationError{
			Field:   fieldName,
			Message: "must match ^[a-z0-9][a-z0-9-]{0,63}$ (lowercase ASCII letters, digits, hyphens; must start with letter or digit; max 64 characters)",
		}
	}
	return normalized, nil
}

// ---------------------------------------------------------------------------
// ListTags
// ---------------------------------------------------------------------------

// ListTags returns the full vocabulary ordered by name ascending. Open to all
// users — no gate invocation. (REQ-F-002, AC-1)
func (s *TagService) ListTags(ctx context.Context) ([]*models.Tag, error) {
	ctx, span := s.getTracer().Start(ctx, "tag_service.list_tags")
	defer span.End()

	tags, err := s.tagRepo.List(ctx)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("tag service: list tags: %w", err))
	}
	return tags, nil
}

// ---------------------------------------------------------------------------
// AddTag
// ---------------------------------------------------------------------------

// AddTag authorizes (first), normalizes the name, creates the tag, and calls
// RecordSuccess. Returns the created *models.Tag on success.
//
// Per D1 (spec §2.8): Authorize is called before ValidateName to prevent
// unauthenticated probing of the validator. (REQ-F-002, REQ-F-003, REQ-F-004, AC-2, AC-3)
func (s *TagService) AddTag(ctx context.Context, name, providedPass string) (*models.Tag, error) {
	ctx, span := s.getTracer().Start(ctx, "tag_service.add_tag")
	defer span.End()

	// Step 1: Authorize first (D1 — before any validation).
	if err := s.gate.Authorize(ctx, providedPass); err != nil {
		return nil, recordSpanError(span, err) // UnauthorizedError propagates unwrapped (REQ-F-007)
	}

	// Step 2: Normalize + validate name.
	normalized, err := s.ValidateName(name)
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	span.SetAttributes(attribute.String("tag.name", normalized))

	// Step 3: Create tag.
	created, err := s.tagRepo.Create(ctx, normalized)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagConflict) {
			return nil, recordSpanError(span, &ConflictError{Name: normalized})
		}
		return nil, recordSpanError(span, fmt.Errorf("tag service: add tag %q: %w", normalized, err))
	}

	// Step 4: RecordSuccess is best-effort; errors are logged and swallowed.
	if err := s.gate.RecordSuccess(ctx); err != nil {
		log.Printf("tag service: AddTag: RecordSuccess error (swallowed): %v", err)
	}

	return created, nil
}

// ---------------------------------------------------------------------------
// RemoveTag
// ---------------------------------------------------------------------------

// RemoveTag authorizes, normalizes the name, looks up the tag, enforces the
// in-use policy (ADR-9), deletes, and records success.
//
// Argument order per spec D5: (ctx, name, force, providedPass).
// (REQ-F-005, AC-5, AC-6, AC-7)
func (s *TagService) RemoveTag(ctx context.Context, name string, force bool, providedPass string) error {
	ctx, span := s.getTracer().Start(ctx, "tag_service.remove_tag")
	defer span.End()

	// Step 1: Authorize.
	if err := s.gate.Authorize(ctx, providedPass); err != nil {
		return recordSpanError(span, err)
	}

	// Step 2: Normalize + validate name.
	normalized, err := s.ValidateName(name)
	if err != nil {
		return recordSpanError(span, err)
	}

	span.SetAttributes(
		attribute.String("tag.name", normalized),
		attribute.Bool("tag.force", force),
	)

	// Step 3: Look up tag by name.
	t, err := s.tagRepo.GetByName(ctx, normalized)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagNotFound) {
			return recordSpanError(span, &NotFoundError{Name: normalized})
		}
		return recordSpanError(span, fmt.Errorf("tag service: remove tag %q: lookup: %w", normalized, err))
	}

	// Step 4: Count usages.
	count, err := s.entityTagRepo.CountByTag(ctx, t.ID)
	if err != nil {
		return recordSpanError(span, fmt.Errorf("tag service: remove tag %q: count usages: %w", normalized, err))
	}

	// Step 5: Enforce in-use policy.
	if count > 0 && !force {
		return recordSpanError(span, &TagInUseError{Name: normalized, Count: count})
	}

	// Step 6 & 7: Delete (force=true removes entity_tags rows first; force=false only
	// when count==0, so it's safe to pass force=false here).
	if err := s.tagRepo.Delete(ctx, t.ID, force); err != nil {
		return recordSpanError(span, fmt.Errorf("tag service: remove tag %q: delete: %w", normalized, err))
	}

	// Step 8: RecordSuccess is best-effort.
	if err := s.gate.RecordSuccess(ctx); err != nil {
		log.Printf("tag service: RemoveTag: RecordSuccess error (swallowed): %v", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// RenameTag
// ---------------------------------------------------------------------------

// RenameTag authorizes, normalizes both names, pre-checks collision, renames,
// and records success. Returns the updated *models.Tag on success.
//
// Per ADR-8: collision detected via GetByName pre-check so the error is typed
// *ConflictError rather than relying on the repository's constraint. The
// service MUST NOT call any EntityTagRepository method. (REQ-F-006, AC-8..10)
func (s *TagService) RenameTag(ctx context.Context, oldName, newName, providedPass string) (*models.Tag, error) {
	ctx, span := s.getTracer().Start(ctx, "tag_service.rename_tag")
	defer span.End()

	// Step 1: Authorize.
	if err := s.gate.Authorize(ctx, providedPass); err != nil {
		return nil, recordSpanError(span, err)
	}

	// Step 2: Normalize + validate both names.
	normalizedOld, err := s.validateTagNameWithField(oldName, "old name")
	if err != nil {
		return nil, recordSpanError(span, err)
	}
	normalizedNew, err := s.validateTagNameWithField(newName, "new name")
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	// Same-name check (after normalization).
	if normalizedOld == normalizedNew {
		return nil, recordSpanError(span, &ValidationError{
			Field:   "new name",
			Message: fmt.Sprintf("must differ from old name %q", normalizedOld),
		})
	}

	span.SetAttributes(attribute.String("tag.name", normalizedNew))

	// Step 3: Look up source tag.
	src, err := s.tagRepo.GetByName(ctx, normalizedOld)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagNotFound) {
			return nil, recordSpanError(span, &NotFoundError{Name: normalizedOld})
		}
		return nil, recordSpanError(span, fmt.Errorf("tag service: rename tag: lookup %q: %w", normalizedOld, err))
	}

	// Step 4: Pre-check collision.
	existing, err := s.tagRepo.GetByName(ctx, normalizedNew)
	if err != nil && !errors.Is(err, tagrepo.ErrTagNotFound) {
		return nil, recordSpanError(span, fmt.Errorf("tag service: rename tag: collision check for %q: %w", normalizedNew, err))
	}
	if existing != nil {
		return nil, recordSpanError(span, &ConflictError{Name: normalizedNew})
	}

	// Step 5: Rename.
	updated, err := s.tagRepo.Rename(ctx, src.ID, normalizedNew)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagConflict) {
			return nil, recordSpanError(span, &ConflictError{Name: normalizedNew})
		}
		return nil, recordSpanError(span, fmt.Errorf("tag service: rename tag %q to %q: %w", normalizedOld, normalizedNew, err))
	}

	// Step 7: RecordSuccess is best-effort.
	if err := s.gate.RecordSuccess(ctx); err != nil {
		log.Printf("tag service: RenameTag: RecordSuccess error (swallowed): %v", err)
	}

	return updated, nil
}
