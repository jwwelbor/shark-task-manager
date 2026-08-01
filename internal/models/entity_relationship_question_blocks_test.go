package models

import "testing"

func TestQuestionBlocksIsValidNonCyclicRelationshipType(t *testing.T) {
	rel := &EntityRelationship{
		FromEntityType:   EntityTypeQuestion,
		FromEntityID:     1,
		ToEntityType:     EntityTypeFeature,
		ToEntityID:       2,
		RelationshipType: EntityRelQuestionBlocks,
	}
	if err := rel.Validate(); err != nil {
		t.Fatalf("TC-301 question_blocks Validate() error = %v", err)
	}
	if rel.IsCyclic() {
		t.Fatal("TC-301 question_blocks must not participate in generic cycle detection")
	}
}
