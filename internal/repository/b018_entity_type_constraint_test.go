// Regression test for B018: entity_notes / entity_relationships / entity_tags
// CHECK constraints exclude idea and tech_debt.
//
// Before the fix, four CHECK constraints (entity_notes.entity_type,
// entity_relationships.from_entity_type, entity_relationships.to_entity_type,
// and entity_tags.entity_type) rejected the 'idea' and 'tech_debt' entity
// types at the SQLite layer with `CHECK constraint failed: entity_type IN
// (...)`. The fix drops these CHECKs in a new migration and relies on the
// existing models.ValidEntityTypes app-layer allowlist for validation —
// matching the bugs.linked_entity_type precedent.
//
// This test creates an idea and a tech_debt row, then exercises all three
// affected polymorphic-association tables against both entity types. Before
// the fix, every INSERT (except entity_tags for idea, which was the only one
// already allowing idea) would fail with a CHECK constraint error. After the
// fix, all INSERTs succeed.
package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestB018_PolymorphicTablesAcceptIdeaAndTechDebt is a regression test for
// B018. It writes notes, relationships, and tags for both 'idea' and
// 'tech_debt' entity types directly through SQL so the CHECK constraints
// (and only the CHECK constraints) are what's being exercised. Each
// sub-INSERT runs against an isolated database created by the standard
// repository-test helper.
//
// All sub-cases must succeed after the fix. On main (before the fix), the
// INSERT statements marked "must succeed after fix" fail with:
//
//	CHECK constraint failed: entity_type IN (...)
func TestB018_PolymorphicTablesAcceptIdeaAndTechDebt(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)

	// ---- Seed parent rows for idea and tech_debt -----------------------
	// We insert directly via SQL so the test does not depend on idea or
	// tech_debt repository code paths — the bug is in the CHECK
	// constraints, not in those repos.

	// ideas.created_date is NOT NULL with no default; supply explicit values.
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	ideaResult, err := database.ExecContext(ctx, `
		INSERT INTO ideas (key, title, description, created_date, status, priority)
		VALUES ('I-2099-12-31-01', 'B018 idea', 'regression', ?, 'new', 5)
	`, now)
	if err != nil {
		t.Fatalf("seed idea: %v", err)
	}
	ideaID, err := ideaResult.LastInsertId()
	if err != nil {
		t.Fatalf("seed idea LastInsertId: %v", err)
	}

	techDebtResult, err := database.ExecContext(ctx, `
		INSERT INTO tech_debts (key, title, slug, description, status, category, severity)
		VALUES ('TD-999', 'B018 tech debt', 'b018-tech-debt', 'regression', 'identified', 'code-quality', 'medium')
	`)
	if err != nil {
		t.Fatalf("seed tech_debt: %v", err)
	}
	techDebtID, err := techDebtResult.LastInsertId()
	if err != nil {
		t.Fatalf("seed tech_debt LastInsertId: %v", err)
	}

	// Seed an epic to use as the "other side" of relationships.
	epicResult, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E99', 'B018 epic', 'regression', 'active', 'high')
	`)
	if err != nil {
		t.Fatalf("seed epic: %v", err)
	}
	epicID, err := epicResult.LastInsertId()
	if err != nil {
		t.Fatalf("seed epic LastInsertId: %v", err)
	}

	// Seed a tag row for entity_tags inserts.
	tagResult, err := database.ExecContext(ctx, `
		INSERT INTO tags (name) VALUES ('b018-regression')
	`)
	if err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	tagID, err := tagResult.LastInsertId()
	if err != nil {
		t.Fatalf("seed tag LastInsertId: %v", err)
	}

	// ---- Sub-cases: every INSERT must succeed after the fix ------------

	cases := []struct {
		name string
		stmt string
		args []any
	}{
		{
			name: "entity_notes accepts idea",
			stmt: `INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
			       VALUES ('idea', ?, 'comment', 'B018 note on idea')`,
			args: []any{ideaID},
		},
		{
			name: "entity_notes accepts tech_debt",
			stmt: `INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
			       VALUES ('tech_debt', ?, 'comment', 'B018 note on tech_debt')`,
			args: []any{techDebtID},
		},
		{
			name: "entity_relationships accepts idea on from side",
			stmt: `INSERT INTO entity_relationships
			         (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
			       VALUES ('idea', ?, 'epic', ?, 'related_to')`,
			args: []any{ideaID, epicID},
		},
		{
			name: "entity_relationships accepts tech_debt on from side",
			stmt: `INSERT INTO entity_relationships
			         (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
			       VALUES ('tech_debt', ?, 'epic', ?, 'related_to')`,
			args: []any{techDebtID, epicID},
		},
		{
			name: "entity_relationships accepts idea on to side",
			stmt: `INSERT INTO entity_relationships
			         (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
			       VALUES ('epic', ?, 'idea', ?, 'spawned_from')`,
			args: []any{epicID, ideaID},
		},
		{
			name: "entity_relationships accepts tech_debt on to side",
			stmt: `INSERT INTO entity_relationships
			         (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
			       VALUES ('epic', ?, 'tech_debt', ?, 'related_to')`,
			args: []any{epicID, techDebtID},
		},
		{
			name: "entity_tags accepts tech_debt",
			stmt: `INSERT INTO entity_tags (entity_type, entity_id, tag_id)
			       VALUES ('tech_debt', ?, ?)`,
			args: []any{techDebtID, tagID},
		},
		// entity_tags already allowed 'idea' before the fix (E28 included it),
		// so we exercise it here only to confirm we did not regress idea support
		// while removing the CHECK clause.
		{
			name: "entity_tags still accepts idea",
			stmt: `INSERT INTO entity_tags (entity_type, entity_id, tag_id)
			       VALUES ('idea', ?, ?)`,
			args: []any{ideaID, tagID},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := database.ExecContext(ctx, tc.stmt, tc.args...)
			if err != nil {
				if strings.Contains(err.Error(), "CHECK constraint failed") {
					t.Fatalf("B018 regression: CHECK constraint still rejects this entity_type — %v", err)
				}
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
