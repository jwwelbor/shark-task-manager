package services

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// QuestionBlock is the compact I-03 handoff for a directly blocked candidate.
// It deliberately contains no Question response, provenance, or relationship data.
type QuestionBlock struct {
	QuestionKey      string `json:"question_key"`
	Summary          string `json:"summary"`
	ResolutionOwner  string `json:"resolution_owner"`
	CurrentResponder string `json:"current_responder"`
}

// QuestionBlockedError is the typed transition-boundary rejection. It keeps
// the candidate identity with the same compact handoff returned by Check.
type QuestionBlockedError struct {
	CandidateType models.EntityType
	CandidateKey  string
	QuestionBlock *QuestionBlock
}

// NewQuestionBlockedError constructs a typed blocked-candidate error.
func NewQuestionBlockedError(candidateType models.EntityType, candidateKey string, block *QuestionBlock) *QuestionBlockedError {
	return &QuestionBlockedError{CandidateType: candidateType, CandidateKey: candidateKey, QuestionBlock: block}
}

// Error implements error without disclosing Question material beyond the
// compact Question key and summary that the handoff already permits.
func (e *QuestionBlockedError) Error() string {
	if e.QuestionBlock == nil {
		return fmt.Sprintf("%s %s is blocked by a Question", e.CandidateType, e.CandidateKey)
	}
	return fmt.Sprintf("%s %s is blocked by Question %s: %s", e.CandidateType, e.CandidateKey, e.QuestionBlock.QuestionKey, e.QuestionBlock.Summary)
}

// QuestionBlockerRelationshipReader reads only incoming typed relationships.
type QuestionBlockerRelationshipReader interface {
	GetIncoming(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error)
}

// QuestionBlockerRegistry resolves the candidate through its registered repository.
type QuestionBlockerRegistry interface {
	GetRepository(entityType models.EntityType) (EntityRepository, error)
}

// QuestionBlockerQuestionReader reads a source Question by its relationship ID.
type QuestionBlockerQuestionReader interface {
	GetQuestionByID(ctx context.Context, id int64) (*models.Question, error)
}

// QualifyQuestionBlock is the one F03 source-Question predicate shared by
// dispatch preflight and focused read surfaces. It deliberately accepts no
// relationship data, so callers remain responsible for direct typed-edge
// selection and deterministic edge ordering.
func QualifyQuestionBlock(question *models.Question) (*QuestionBlock, error) {
	if question == nil {
		return nil, errors.New("source Question is required")
	}
	if !question.Blocking || (question.Status != "open" && question.Status != "answering") {
		return nil, nil
	}
	if err := models.ValidateQuestionKey(question.Key); err != nil {
		return nil, fmt.Errorf("validate source Question identity: %w", err)
	}
	state, err := models.DecodeQuestionState(question.ContextData)
	if err != nil {
		return nil, fmt.Errorf("decode state for %s: %w", question.Key, err)
	}
	if state == nil {
		return nil, fmt.Errorf("decode state for %s: Question workflow state is required", question.Key)
	}
	return &QuestionBlock{
		QuestionKey: question.Key, Summary: question.Summary,
		ResolutionOwner: state.ResolutionOwner, CurrentResponder: state.CurrentResponder(),
	}, nil
}

// QuestionBlocker qualifies direct open blocking Question relationships without
// mutating either side of the relationship.
type QuestionBlocker struct {
	relationships QuestionBlockerRelationshipReader
	registry      QuestionBlockerRegistry
	questions     QuestionBlockerQuestionReader
}

// NewQuestionBlocker constructs the read-only direct Question gate.
func NewQuestionBlocker(relationships QuestionBlockerRelationshipReader, registry QuestionBlockerRegistry, questions QuestionBlockerQuestionReader) (*QuestionBlocker, error) {
	if relationships == nil {
		return nil, fmt.Errorf("QuestionBlocker requires a relationship reader")
	}
	if registry == nil {
		return nil, fmt.Errorf("QuestionBlocker requires an entity registry")
	}
	if questions == nil {
		return nil, fmt.Errorf("QuestionBlocker requires a Question reader")
	}
	return &QuestionBlocker{relationships: relationships, registry: registry, questions: questions}, nil
}

// Check returns the deterministic compact handoff for one directly blocked
// candidate, nil when no source qualifies, or an actionable read/state error.
func (b *QuestionBlocker) Check(ctx context.Context, candidateType models.EntityType, candidateKey string) (*QuestionBlock, error) {
	repository, err := b.registry.GetRepository(candidateType)
	if err != nil {
		return nil, fmt.Errorf("Question blocker resolve %s %s: %w", candidateType, candidateKey, err)
	}
	candidate, err := repository.GetByKey(ctx, candidateKey)
	if err != nil {
		return nil, fmt.Errorf("Question blocker load candidate %s: %w", candidateKey, err)
	}
	if candidate == nil {
		return nil, fmt.Errorf("Question blocker load candidate %s: repository returned no entity", candidateKey)
	}

	edges, err := b.relationships.GetIncoming(ctx, candidateType, candidate.GetID(), []models.EntityRelationshipType{models.EntityRelQuestionBlocks})
	if err != nil {
		return nil, fmt.Errorf("Question blocker load incoming relationships for %s: %w", candidateKey, err)
	}
	type match struct {
		edge  *models.EntityRelationship
		block *QuestionBlock
	}
	matches := make([]match, 0, len(edges))
	for _, edge := range edges {
		if edge == nil || edge.RelationshipType != models.EntityRelQuestionBlocks || edge.FromEntityType != models.EntityTypeQuestion {
			continue
		}
		question, err := b.questions.GetQuestionByID(ctx, edge.FromEntityID)
		if err != nil {
			return nil, fmt.Errorf("Question blocker load source Question %d: %w", edge.FromEntityID, err)
		}
		if question == nil {
			return nil, fmt.Errorf("Question blocker load source Question %d: reader returned no Question", edge.FromEntityID)
		}
		block, err := QualifyQuestionBlock(question)
		if err != nil {
			return nil, fmt.Errorf("Question blocker %w", err)
		}
		if block != nil {
			matches = append(matches, match{edge: edge, block: block})
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].edge.CreatedAt.Equal(matches[j].edge.CreatedAt) {
			return matches[i].edge.ID < matches[j].edge.ID
		}
		return matches[i].edge.CreatedAt.Before(matches[j].edge.CreatedAt)
	})
	return matches[0].block, nil
}
