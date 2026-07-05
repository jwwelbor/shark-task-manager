// Package searchindex provides the shared SQL projection for the unified search
// index so migrations and repositories cannot drift.
package searchindex

import (
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Columns is the canonical entity_search_fts column list used by rebuilds and
// incremental index updates.
const Columns = `
	entity_type, entity_id, key, title, body, note_text, metadata_text, status, severity
`

// CreateTableSQL returns the unified FTS5 table DDL.
func CreateTableSQL() string {
	return `
		CREATE VIRTUAL TABLE entity_search_fts USING fts5(
			entity_type UNINDEXED,
			entity_id UNINDEXED,
			key,
			title,
			body,
			note_text,
			metadata_text,
			status UNINDEXED,
			severity UNINDEXED,
			tokenize='porter unicode61'
		);
	`
}

// RebuildSQL returns the full backfill INSERT for every indexed entity type.
func RebuildSQL() string {
	selects := make([]string, 0, len(projections))
	for _, projection := range projections {
		selects = append(selects, projection.selectSQL)
	}
	return "INSERT INTO entity_search_fts (" + Columns + ")\n" + strings.Join(selects, "\nUNION ALL\n")
}

// InsertSQL returns the single-entity INSERT for an indexed entity type.
func InsertSQL(entityType models.EntityType) (string, bool) {
	projection, ok := projectionByType[entityType]
	if !ok {
		return "", false
	}
	return "INSERT INTO entity_search_fts (" + Columns + ")\n" + projection.selectSQL + "\nWHERE " + projection.alias + ".id = ?", true
}

// SupportsEntity reports whether the unified search index stores the entity type.
func SupportsEntity(entityType models.EntityType) bool {
	_, ok := projectionByType[entityType]
	return ok
}

type projection struct {
	entityType models.EntityType
	alias      string
	selectSQL  string
}

var projections = []projection{
	{
		entityType: models.EntityTypeEpic,
		alias:      "e",
		selectSQL: `
			SELECT
				'epic',
				e.id,
				e.key,
				e.title,
				COALESCE(e.description, ''),
				COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'epic' AND entity_id = e.id), ''),
				COALESCE(e.priority || ' ' || e.status, ''),
				COALESCE(e.status, ''),
				''
			FROM epics e`,
	},
	{
		entityType: models.EntityTypeFeature,
		alias:      "f",
		selectSQL: `
			SELECT
				'feature',
				f.id,
				f.key,
				f.title,
				COALESCE(f.description, ''),
				COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'feature' AND entity_id = f.id), ''),
				COALESCE(f.status, ''),
				COALESCE(f.status, ''),
				''
			FROM features f`,
	},
	{
		entityType: models.EntityTypeTask,
		alias:      "t",
		selectSQL: `
			SELECT
				'task',
				t.id,
				t.key,
				t.title,
				COALESCE(t.description, ''),
				COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'task' AND entity_id = t.id), ''),
				COALESCE(t.agent_type || ' ' || t.status, ''),
				COALESCE(t.status, ''),
				''
			FROM tasks t`,
	},
	{
		entityType: models.EntityTypeBug,
		alias:      "b",
		selectSQL: `
			SELECT
				'bug',
				b.id,
				b.key,
				b.title,
				COALESCE(b.description, ''),
				COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'bug' AND entity_id = b.id), ''),
				COALESCE(b.linked_entity_type || ' ' || b.linked_entity_key || ' ' || b.status, b.status, ''),
				COALESCE(b.status, ''),
				COALESCE(b.severity, '')
			FROM bugs b`,
	},
	{
		entityType: models.EntityTypeChange,
		alias:      "c",
		selectSQL: `
			SELECT
				'change',
				c.id,
				c.key,
				c.title,
				COALESCE(c.description, ''),
				COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'change' AND entity_id = c.id), ''),
				COALESCE(c.justification || ' ' || c.impact_analysis || ' ' || c.rollback_plan || ' ' || c.status, c.status, ''),
				COALESCE(c.status, ''),
				''
			FROM change_cards c`,
	},
	{
		entityType: models.EntityTypeTechDebt,
		alias:      "td",
		selectSQL: `
			SELECT
				'tech_debt',
				td.id,
				td.key,
				td.title,
				COALESCE(td.description, ''),
				COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'tech_debt' AND entity_id = td.id), ''),
				COALESCE(td.category || ' ' || td.severity || ' ' || td.effort_estimate || ' ' || td.status, td.status, ''),
				COALESCE(td.status, ''),
				COALESCE(td.severity, '')
			FROM tech_debts td`,
	},
	{
		entityType: models.EntityTypeIdea,
		alias:      "i",
		selectSQL: `
			SELECT
				'idea',
				i.id,
				i.key,
				i.title,
				TRIM(COALESCE(i.description, '') || ' ' || COALESCE(i.notes, '')),
				COALESCE((SELECT GROUP_CONCAT(content, ' ') FROM entity_notes WHERE entity_type = 'idea' AND entity_id = i.id), ''),
				TRIM(COALESCE(i.dependencies, '') || ' ' || COALESCE(i.related_docs, '') || ' ' || COALESCE(i.status, '')),
				COALESCE(i.status, ''),
				''
			FROM ideas i`,
	},
}

var projectionByType = func() map[models.EntityType]projection {
	byType := make(map[models.EntityType]projection, len(projections))
	for _, projection := range projections {
		byType[projection.entityType] = projection
	}
	return byType
}()
