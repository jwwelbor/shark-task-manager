package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	tagrepo "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
)

const tagServiceTracerName = "shark/services/tag"

// TagEnforcementConfig is the narrow contract TagService needs from config —
// it exposes the list of entity-type strings that require at least one tag
// on create. Defined here (not in the config package) so TagService stays
// free of config concerns; *config.Config satisfies it.
type TagEnforcementConfig interface {
	TagRequiredFor() []string
}

// EmptyTagEnforcementConfig is a safe TagEnforcementConfig fallback with an
// always-empty slice. Used by entry-point wiring (CLI, HTTP) when the
// project config cannot be loaded — disables enforcement.
type EmptyTagEnforcementConfig struct{}

func (EmptyTagEnforcementConfig) TagRequiredFor() []string { return nil }

// TagService owns the vocabulary business rules for the managed tag set.
type TagService struct {
	tagRepo       tagrepo.TagRepositoryInterface
	entityTagRepo tagrepo.EntityTagRepositoryInterface
	gate          maintainer.Gate
	cfg           TagEnforcementConfig
	tracer        trace.Tracer // optional; nil falls back to global tracer
}

// NewTagService constructs a TagService. All four dependencies are required;
// the constructor panics on nil.
func NewTagService(
	tagRepo tagrepo.TagRepositoryInterface,
	entityTagRepo tagrepo.EntityTagRepositoryInterface,
	gate maintainer.Gate,
	cfg TagEnforcementConfig,
) *TagService {
	requireNonNil(tagRepo, "TagService requires a non-nil TagRepositoryInterface")
	requireNonNil(entityTagRepo, "TagService requires a non-nil EntityTagRepositoryInterface")
	requireNonNil(gate, "TagService requires a non-nil Gate")
	requireNonNil(cfg, "TagService requires a non-nil TagEnforcementConfig")
	return &TagService{
		tagRepo:       tagRepo,
		entityTagRepo: entityTagRepo,
		gate:          gate,
		cfg:           cfg,
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

const tagNameRuleMessage = "must match ^[a-z0-9][a-z0-9-]{0,63}$ (lowercase ASCII letters, digits, hyphens; must start with letter or digit; max 64 characters)"

// ValidateName normalizes raw input (TrimSpace + ToLower) and validates the
// result against models.ValidateTagName. Returns the normalized name on
// success, or a *ValidationError on failure.
func (s *TagService) ValidateName(raw string) (string, error) {
	return s.validateName(raw, "tag name")
}

func (s *TagService) validateName(raw, field string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if err := models.ValidateTagName(normalized); err != nil {
		return "", &ValidationError{Field: field, Message: tagNameRuleMessage}
	}
	return normalized, nil
}

// ListTags returns the full vocabulary ordered by name ascending. Open to all
// users — no gate invocation.
func (s *TagService) ListTags(ctx context.Context) ([]*models.Tag, error) {
	ctx, span := s.getTracer().Start(ctx, "tag_service.list_tags")
	defer span.End()

	tags, err := s.tagRepo.List(ctx)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("tag service: list tags: %w", err))
	}
	return tags, nil
}

// AddTag authorizes first, normalizes the name, creates the tag, and records
// a success on the gate. Authorize precedes ValidateName so an unauthenticated
// caller cannot probe the validator.
func (s *TagService) AddTag(ctx context.Context, name, providedPass string) (*models.Tag, error) {
	ctx, span := s.getTracer().Start(ctx, "tag_service.add_tag")
	defer span.End()

	if err := s.gate.Authorize(ctx, providedPass); err != nil {
		return nil, recordSpanError(span, err)
	}

	normalized, err := s.ValidateName(name)
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	span.SetAttributes(attribute.String("tag.name", normalized))

	created, err := s.tagRepo.Create(ctx, normalized)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagConflict) {
			return nil, recordSpanError(span, &ConflictError{Name: normalized})
		}
		return nil, recordSpanError(span, fmt.Errorf("tag service: add tag %q: %w", normalized, err))
	}

	if err := s.gate.RecordSuccess(ctx); err != nil {
		log.Printf("tag service: AddTag: RecordSuccess error (swallowed): %v", err)
	}

	return created, nil
}

// RemoveTag authorizes, normalizes the name, looks up the tag, and deletes
// it. When the tag is attached to any entity and force=false, returns
// *TagInUseError; force=true removes associations first.
func (s *TagService) RemoveTag(ctx context.Context, name string, force bool, providedPass string) error {
	ctx, span := s.getTracer().Start(ctx, "tag_service.remove_tag")
	defer span.End()

	if err := s.gate.Authorize(ctx, providedPass); err != nil {
		return recordSpanError(span, err)
	}

	normalized, err := s.ValidateName(name)
	if err != nil {
		return recordSpanError(span, err)
	}

	span.SetAttributes(
		attribute.String("tag.name", normalized),
		attribute.Bool("tag.force", force),
	)

	t, err := s.tagRepo.GetByName(ctx, normalized)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagNotFound) {
			return recordSpanError(span, &NotFoundError{Name: normalized})
		}
		return recordSpanError(span, fmt.Errorf("tag service: remove tag %q: lookup: %w", normalized, err))
	}

	count, err := s.entityTagRepo.CountByTag(ctx, t.ID)
	if err != nil {
		return recordSpanError(span, fmt.Errorf("tag service: remove tag %q: count usages: %w", normalized, err))
	}

	if count > 0 && !force {
		return recordSpanError(span, &TagInUseError{Name: normalized, Count: count})
	}

	if err := s.tagRepo.Delete(ctx, t.ID, force); err != nil {
		return recordSpanError(span, fmt.Errorf("tag service: remove tag %q: delete: %w", normalized, err))
	}

	if err := s.gate.RecordSuccess(ctx); err != nil {
		log.Printf("tag service: RemoveTag: RecordSuccess error (swallowed): %v", err)
	}

	return nil
}

// RenameTag authorizes, normalizes both names, and renames the tag. Entity
// associations are preserved (the repository updates the tag row in place).
// Returns *ConflictError if the new name is already registered.
func (s *TagService) RenameTag(ctx context.Context, oldName, newName, providedPass string) (*models.Tag, error) {
	ctx, span := s.getTracer().Start(ctx, "tag_service.rename_tag")
	defer span.End()

	if err := s.gate.Authorize(ctx, providedPass); err != nil {
		return nil, recordSpanError(span, err)
	}

	normalizedOld, err := s.validateName(oldName, "old name")
	if err != nil {
		return nil, recordSpanError(span, err)
	}
	normalizedNew, err := s.validateName(newName, "new name")
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	if normalizedOld == normalizedNew {
		return nil, recordSpanError(span, &ValidationError{
			Field:   "new name",
			Message: fmt.Sprintf("must differ from old name %q", normalizedOld),
		})
	}

	span.SetAttributes(attribute.String("tag.name", normalizedNew))

	src, err := s.tagRepo.GetByName(ctx, normalizedOld)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagNotFound) {
			return nil, recordSpanError(span, &NotFoundError{Name: normalizedOld})
		}
		return nil, recordSpanError(span, fmt.Errorf("tag service: rename tag: lookup %q: %w", normalizedOld, err))
	}

	existing, err := s.tagRepo.GetByName(ctx, normalizedNew)
	if err != nil && !errors.Is(err, tagrepo.ErrTagNotFound) {
		return nil, recordSpanError(span, fmt.Errorf("tag service: rename tag: collision check for %q: %w", normalizedNew, err))
	}
	if existing != nil {
		return nil, recordSpanError(span, &ConflictError{Name: normalizedNew})
	}

	updated, err := s.tagRepo.Rename(ctx, src.ID, normalizedNew)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagConflict) {
			return nil, recordSpanError(span, &ConflictError{Name: normalizedNew})
		}
		return nil, recordSpanError(span, fmt.Errorf("tag service: rename tag %q to %q: %w", normalizedOld, normalizedNew, err))
	}

	if err := s.gate.RecordSuccess(ctx); err != nil {
		log.Printf("tag service: RenameTag: RecordSuccess error (swallowed): %v", err)
	}

	return updated, nil
}

// AttachMany attaches the named tags to (entityType, entityID). All names
// must be registered in the vocabulary; encountering an unregistered name
// aborts before any Attach call runs. An empty or nil names slice is a
// no-op. Duplicate names in the same call are deduplicated. Does not
// consume the maintainer gate — attaching a registered tag is open.
func (s *TagService) AttachMany(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	names []string,
) error {
	ctx, span := s.getTracer().Start(ctx, "tag_service.attach_many",
		trace.WithAttributes(
			attribute.String("entity.type", string(entityType)),
			attribute.Int64("entity.id", entityID),
			attribute.Int("tag.count", len(names)),
		),
	)
	defer span.End()

	if len(names) == 0 {
		return nil
	}

	resolved := make([]*models.Tag, 0, len(names))
	for _, raw := range names {
		normalized, err := s.ValidateName(raw)
		if err != nil {
			return recordSpanError(span, err)
		}
		t, err := s.tagRepo.GetByName(ctx, normalized)
		if err != nil {
			if errors.Is(err, tagrepo.ErrTagNotFound) {
				return recordSpanError(span, &UnregisteredTagError{Name: normalized})
			}
			return recordSpanError(span, fmt.Errorf("tag service: attach many: lookup %q: %w", normalized, err))
		}
		resolved = append(resolved, t)
	}

	for _, t := range resolved {
		if err := s.entityTagRepo.Attach(ctx, entityType, entityID, t.ID); err != nil {
			return recordSpanError(span, fmt.Errorf("tag service: attach many: attach %q: %w", t.Name, err))
		}
	}
	return nil
}

// DetachOne detaches a single tag from (entityType, entityID). The tag must
// exist in the vocabulary — a name absent from the vocabulary returns
// *NotFoundError (distinct from the attach-path *UnregisteredTagError).
// If no attachment row exists, the underlying Detach is a no-op and this
// returns nil. Does not consume the maintainer gate.
func (s *TagService) DetachOne(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	name string,
) error {
	ctx, span := s.getTracer().Start(ctx, "tag_service.detach_one")
	defer span.End()

	normalized, err := s.ValidateName(name)
	if err != nil {
		return recordSpanError(span, err)
	}
	// Attributes set post-validation so malformed raw input is not leaked into telemetry.
	span.SetAttributes(
		attribute.String("entity.type", string(entityType)),
		attribute.Int64("entity.id", entityID),
		attribute.String("tag.name", normalized),
	)

	t, err := s.tagRepo.GetByName(ctx, normalized)
	if err != nil {
		if errors.Is(err, tagrepo.ErrTagNotFound) {
			return recordSpanError(span, &NotFoundError{Name: normalized})
		}
		return recordSpanError(span, fmt.Errorf("tag service: detach one: lookup %q: %w", normalized, err))
	}

	if err := s.entityTagRepo.Detach(ctx, entityType, entityID, t.ID); err != nil {
		return recordSpanError(span, fmt.Errorf("tag service: detach one %q: %w", normalized, err))
	}
	return nil
}

// TagQuerier is the narrow read-only interface consumed by entity services
// for tag-based filtering (F05). It extends the capabilities of TagAttacher
// with three query methods used by List and Get paths.
//
// Both *TagService and *MockTagService (test double) implement this interface.
type TagQuerier interface {
	TagAttacher
	// EntityIDsByTags returns sorted entity IDs satisfying the AND intersection
	// of the supplied tag names for the given entity type.
	EntityIDsByTags(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)
	// ListTagsForEntity returns the sorted normalized tag names attached to a
	// single entity. Returns an empty non-nil slice when no tags are attached.
	ListTagsForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error)
	// AttachedTagNamesByIDs returns a map from entityID to its sorted tag
	// names. Every input ID appears in the map (empty slice for zero tags).
	AttachedTagNamesByIDs(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)
}

// Compile-time check: *TagService must implement TagQuerier.
var _ TagQuerier = (*TagService)(nil)

// EntityIDsByTags returns the sorted, deduplicated list of entity IDs for
// entityType that satisfy the multi-tag AND intersection of names.
//
// Empty or nil names returns (nil, nil) — callers interpret this as "no tag
// filter". Any unknown name produces *UnregisteredTagError and the method
// issues no EntityTagRepository filter call. Duplicate names are deduplicated.
// Names are normalized via ValidateName.
func (s *TagService) EntityIDsByTags(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
	if len(names) == 0 {
		return nil, nil
	}

	normalizedOp := op.normalize()
	ctx, span := s.getTracer().Start(ctx, "tag_service.entity_ids_by_tags",
		trace.WithAttributes(
			attribute.String("entity.type", string(entityType)),
			attribute.Int("tag.count", len(names)),
			attribute.String("filter.op", string(normalizedOp)),
		),
	)
	defer span.End()

	// Normalize names, dedupe before hitting the repo so duplicate inputs don't
	// trigger duplicate GetByName lookups, then resolve each unique name to its ID.
	seenNames := make(map[string]struct{}, len(names))
	tagIDs := make([]int64, 0, len(names))
	for _, raw := range names {
		normalized, err := s.ValidateName(raw)
		if err != nil {
			return nil, recordSpanError(span, err)
		}
		if _, dup := seenNames[normalized]; dup {
			continue
		}
		seenNames[normalized] = struct{}{}

		t, err := s.tagRepo.GetByName(ctx, normalized)
		if err != nil {
			if errors.Is(err, tagrepo.ErrTagNotFound) {
				return nil, recordSpanError(span, &UnregisteredTagError{Name: normalized})
			}
			return nil, recordSpanError(span, fmt.Errorf("tag service: entity ids by tags: lookup %q: %w", normalized, err))
		}
		tagIDs = append(tagIDs, t.ID)
	}

	ids, err := s.entityTagRepo.FilterEntityIDs(ctx, entityType, tagIDs)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("tag service: entity ids by tags: filter: %w", err))
	}
	return ids, nil
}

// ListTagsForEntity returns the sorted ascending list of normalized tag names
// attached to (entityType, entityID). Returns an empty non-nil slice when no
// tags are attached.
func (s *TagService) ListTagsForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error) {
	ctx, span := s.getTracer().Start(ctx, "tag_service.list_tags_for_entity",
		trace.WithAttributes(
			attribute.String("entity.type", string(entityType)),
			attribute.Int64("entity.id", entityID),
		),
	)
	defer span.End()

	byID, err := s.AttachedTagNamesByIDs(ctx, entityType, []int64{entityID})
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("tag service: list tags for entity %s/%d: %w", entityType, entityID, err))
	}
	return byID[entityID], nil
}

// AttachedTagNamesByIDs returns a map from entityID to its sorted list of
// attached tag names. Every input ID appears in the map (including those with
// zero attachments, mapped to an empty non-nil slice). Empty input returns a
// non-nil empty map.
func (s *TagService) AttachedTagNamesByIDs(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error) {
	if len(entityIDs) == 0 {
		return map[int64][]string{}, nil
	}

	ctx, span := s.getTracer().Start(ctx, "tag_service.attached_tag_names_by_ids",
		trace.WithAttributes(
			attribute.String("entity.type", string(entityType)),
			attribute.Int("entity.count", len(entityIDs)),
		),
	)
	defer span.End()

	rows, err := s.entityTagRepo.ListTagNamesByEntities(ctx, entityType, entityIDs)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("tag service: attached tag names by ids: list: %w", err))
	}

	// Bucket rows into map.
	result := make(map[int64][]string, len(entityIDs))
	for _, row := range rows {
		result[row.EntityID] = append(result[row.EntityID], row.TagName)
	}

	// Ensure every input ID is present in the map.
	for _, id := range entityIDs {
		if _, ok := result[id]; !ok {
			result[id] = []string{}
		}
	}

	// Sort each slice (repository already returns ordered, but defensive sort is cheap).
	for id := range result {
		sort.Strings(result[id])
	}

	return result, nil
}

// EnforceRequired returns *TagRequiredError when the configured
// TagRequiredFor slice contains entityType.String() and names is empty
// (nil or []string{}). Otherwise returns nil. Name contents are not
// validated here — AttachMany handles that. Entries in the configured
// slice are matched with case-sensitive ==; mis-cased entries silently
// disable enforcement for that type (allowed lowercase values: "task",
// "feature", "epic", "bug", "change", "idea").
func (s *TagService) EnforceRequired(
	ctx context.Context,
	entityType models.EntityType,
	names []string,
) error {
	_, span := s.getTracer().Start(ctx, "tag_service.enforce_required",
		trace.WithAttributes(
			attribute.String("entity.type", string(entityType)),
			attribute.Int("tag.count", len(names)),
		),
	)
	defer span.End()

	if len(names) > 0 {
		return nil
	}

	required := s.cfg.TagRequiredFor()
	if len(required) == 0 {
		return nil
	}
	et := string(entityType)
	for _, r := range required {
		if r == et {
			return recordSpanError(span, &TagRequiredError{EntityType: et})
		}
	}
	return nil
}
