// Package portfolio provides set-oriented read models for portfolio advice.
package portfolio

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// ChildStateRow is one feature or task in an epic's descendant snapshot.
type ChildStateRow struct {
	EpicID          int64
	EpicKey         string
	EntityType      models.EntityType
	EntityKey       string
	Title           string
	Status          string
	DirectParentKey string
	ProgressPct     *float64
}

// EpicRelationshipRow is one supported stored relationship between epic IDs.
// Endpoint fields are nil when a stored relationship references a missing epic.
type EpicRelationshipRow struct {
	FromEpicID       int64
	FromKey          *string
	FromStatus       *string
	RelationshipType models.EntityRelationshipType
	ToEpicID         int64
	ToKey            *string
	ToStatus         *string
}

// Snapshot is the complete read model needed to assemble bare-next portfolio
// advice. Production loads it with one database query and decodes the small
// hierarchy locally.
type Snapshot struct {
	Epics         []*models.Epic
	Children      []ChildStateRow
	Relationships []EpicRelationshipRow
	Claims        []*models.EntityClaim
}

// Repository reads the set-oriented data needed to assemble portfolio advice.
type Repository struct {
	db *dbconn.DB
}

// NewRepository creates a portfolio repository.
func NewRepository(db *dbconn.DB) *Repository {
	return &Repository{db: db}
}

// ReadSnapshot loads the complete epic hierarchy, supported epic
// relationships, and claims in one database round trip. epic_display_data is
// the existing epic-to-feature view; task, relationship, and claim JSON are
// attached as small correlated/global projections and decoded locally.
func (r *Repository) ReadSnapshot(ctx context.Context) (Snapshot, error) {
	const query = `
		SELECT e.id,
		       e.key,
		       e.title,
		       e.status,
		       e.priority,
		       e.business_value,
		       e.features_json,
		       (SELECT COALESCE(json_group_array(json_object(
		           'id', t.id,
		           'key', t.key,
		           'title', t.title,
		           'status', t.status,
		           'direct_parent_key', f.key
		       )), '[]')
		        FROM tasks t
		        JOIN features f ON f.id = t.feature_id
		        WHERE f.epic_id = e.id) AS tasks_json,
		       (SELECT COALESCE(json_group_array(json_object(
		           'from_entity_id', er.from_entity_id,
		           'from_key', from_epic.key,
		           'from_status', from_epic.status,
		           'relationship_type', er.relationship_type,
		           'to_entity_id', er.to_entity_id,
		           'to_key', to_epic.key,
		           'to_status', to_epic.status
		       )), '[]')
		        FROM entity_relationships er
		        LEFT JOIN epics from_epic ON from_epic.id = er.from_entity_id
		        LEFT JOIN epics to_epic ON to_epic.id = er.to_entity_id
		        WHERE er.from_entity_type = ?
		          AND er.to_entity_type = ?
		          AND er.relationship_type IN (?, ?, ?)) AS relationships_json,
		       (SELECT COALESCE(json_group_array(json_object(
		           'entity_type', c.entity_type,
		           'entity_key', c.entity_key,
		           'claimed_by', c.claimed_by,
		           'last_heartbeat', c.last_heartbeat,
		           'progress', c.progress
		       )), '[]')
		        FROM entity_claims c) AS claims_json
		FROM epic_display_data e
		ORDER BY e.key ASC
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		models.EntityTypeEpic,
		models.EntityTypeEpic,
		models.EntityRelDependsOn,
		models.EntityRelBlocks,
		models.EntityRelFollows,
	)
	if err != nil {
		return Snapshot{}, fmt.Errorf("query portfolio snapshot: %w", err)
	}
	defer rows.Close()

	snapshot := allocatedSnapshot()
	globalsDecoded := false
	for rows.Next() {
		var (
			epic                          models.Epic
			featuresJSON, tasksJSON       string
			relationshipsJSON, claimsJSON string
		)
		if err := rows.Scan(
			&epic.ID,
			&epic.Key,
			&epic.Title,
			&epic.Status,
			&epic.Priority,
			&epic.BusinessValue,
			&featuresJSON,
			&tasksJSON,
			&relationshipsJSON,
			&claimsJSON,
		); err != nil {
			return Snapshot{}, fmt.Errorf("scan portfolio snapshot epic: %w", err)
		}
		snapshot.Epics = append(snapshot.Epics, &epic)
		if err := appendSnapshotChildren(&snapshot, epic.ID, epic.Key, featuresJSON, tasksJSON); err != nil {
			return Snapshot{}, err
		}
		if !globalsDecoded {
			if err := decodeSnapshotGlobals(&snapshot, relationshipsJSON, claimsJSON); err != nil {
				return Snapshot{}, err
			}
			globalsDecoded = true
		}
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("iterate portfolio snapshot: %w", err)
	}
	sortSnapshot(&snapshot)
	return snapshot, nil
}

type snapshotFeature struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	ProgressPct *float64 `json:"progress_pct"`
}

type snapshotTask struct {
	Key             string `json:"key"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	DirectParentKey string `json:"direct_parent_key"`
}

type snapshotRelationship struct {
	FromEntityID     int64                         `json:"from_entity_id"`
	FromKey          *string                       `json:"from_key"`
	FromStatus       *string                       `json:"from_status"`
	RelationshipType models.EntityRelationshipType `json:"relationship_type"`
	ToEntityID       int64                         `json:"to_entity_id"`
	ToKey            *string                       `json:"to_key"`
	ToStatus         *string                       `json:"to_status"`
}

type snapshotClaim struct {
	EntityType    string   `json:"entity_type"`
	EntityKey     string   `json:"entity_key"`
	ClaimedBy     string   `json:"claimed_by"`
	LastHeartbeat string   `json:"last_heartbeat"`
	Progress      *float64 `json:"progress"`
}

func allocatedSnapshot() Snapshot {
	return Snapshot{
		Epics:         []*models.Epic{},
		Children:      []ChildStateRow{},
		Relationships: []EpicRelationshipRow{},
		Claims:        []*models.EntityClaim{},
	}
}

func appendSnapshotChildren(
	snapshot *Snapshot,
	epicID int64,
	epicKey string,
	featuresJSON string,
	tasksJSON string,
) error {
	var features []snapshotFeature
	if err := json.Unmarshal([]byte(featuresJSON), &features); err != nil {
		return fmt.Errorf("decode portfolio snapshot features for %s: %w", epicKey, err)
	}
	for _, feature := range features {
		snapshot.Children = append(snapshot.Children, ChildStateRow{
			EpicID:          epicID,
			EpicKey:         epicKey,
			EntityType:      models.EntityTypeFeature,
			EntityKey:       feature.Key,
			Title:           feature.Title,
			Status:          feature.Status,
			DirectParentKey: epicKey,
			ProgressPct:     feature.ProgressPct,
		})
	}

	var tasks []snapshotTask
	if err := json.Unmarshal([]byte(tasksJSON), &tasks); err != nil {
		return fmt.Errorf("decode portfolio snapshot tasks for %s: %w", epicKey, err)
	}
	for _, task := range tasks {
		snapshot.Children = append(snapshot.Children, ChildStateRow{
			EpicID:          epicID,
			EpicKey:         epicKey,
			EntityType:      models.EntityTypeTask,
			EntityKey:       task.Key,
			Title:           task.Title,
			Status:          task.Status,
			DirectParentKey: task.DirectParentKey,
		})
	}
	return nil
}

func decodeSnapshotGlobals(snapshot *Snapshot, relationshipsJSON string, claimsJSON string) error {
	var relationships []snapshotRelationship
	if err := json.Unmarshal([]byte(relationshipsJSON), &relationships); err != nil {
		return fmt.Errorf("decode portfolio snapshot relationships: %w", err)
	}
	for _, relationship := range relationships {
		snapshot.Relationships = append(snapshot.Relationships, EpicRelationshipRow{
			FromEpicID:       relationship.FromEntityID,
			FromKey:          relationship.FromKey,
			FromStatus:       relationship.FromStatus,
			RelationshipType: relationship.RelationshipType,
			ToEpicID:         relationship.ToEntityID,
			ToKey:            relationship.ToKey,
			ToStatus:         relationship.ToStatus,
		})
	}

	var claims []snapshotClaim
	if err := json.Unmarshal([]byte(claimsJSON), &claims); err != nil {
		return fmt.Errorf("decode portfolio snapshot claims: %w", err)
	}
	for _, claim := range claims {
		heartbeat, err := parseSnapshotTime(claim.LastHeartbeat)
		if err != nil {
			return fmt.Errorf("decode portfolio snapshot claim %s heartbeat: %w", claim.EntityKey, err)
		}
		snapshot.Claims = append(snapshot.Claims, &models.EntityClaim{
			EntityType:    claim.EntityType,
			EntityKey:     claim.EntityKey,
			ClaimedBy:     claim.ClaimedBy,
			LastHeartbeat: heartbeat,
			Progress:      claim.Progress,
		})
	}
	return nil
}

func parseSnapshotTime(value string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		dbconn.TimeFormat,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05 -0700 MST",
	} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp %q", value)
}

func sortSnapshot(snapshot *Snapshot) {
	sort.Slice(snapshot.Children, func(i, j int) bool {
		left, right := snapshot.Children[i], snapshot.Children[j]
		if left.EpicKey != right.EpicKey {
			return left.EpicKey < right.EpicKey
		}
		if left.EntityType != right.EntityType {
			return left.EntityType < right.EntityType
		}
		return left.EntityKey < right.EntityKey
	})
	sort.Slice(snapshot.Relationships, func(i, j int) bool {
		left, right := snapshot.Relationships[i], snapshot.Relationships[j]
		if pointerString(left.FromKey) != pointerString(right.FromKey) {
			return pointerString(left.FromKey) < pointerString(right.FromKey)
		}
		if left.RelationshipType != right.RelationshipType {
			return left.RelationshipType < right.RelationshipType
		}
		if pointerString(left.ToKey) != pointerString(right.ToKey) {
			return pointerString(left.ToKey) < pointerString(right.ToKey)
		}
		if left.FromEpicID != right.FromEpicID {
			return left.FromEpicID < right.FromEpicID
		}
		return left.ToEpicID < right.ToEpicID
	})
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// ListChildStates returns all feature and task state grouped by verified epic
// ownership. The direct parent distinguishes epic-owned features from
// feature-owned tasks without deriving ownership from entity keys.
func (r *Repository) ListChildStates(ctx context.Context) ([]ChildStateRow, error) {
	const query = `
		SELECT e.id AS epic_id,
		       e.key AS epic_key,
		       ? AS entity_type,
		       f.key AS entity_key,
		       f.title AS title,
		       f.status AS status,
		       e.key AS direct_parent_key,
		       f.progress_pct AS progress_pct
		FROM features f
		JOIN epics e ON e.id = f.epic_id
		UNION ALL
		SELECT e.id AS epic_id,
		       e.key AS epic_key,
		       ? AS entity_type,
		       t.key AS entity_key,
		       t.title AS title,
		       t.status AS status,
		       f.key AS direct_parent_key,
		       CAST(NULL AS REAL) AS progress_pct
		FROM tasks t
		JOIN features f ON f.id = t.feature_id
		JOIN epics e ON e.id = f.epic_id
		ORDER BY epic_key ASC, entity_type ASC, entity_key ASC
	`

	rows, err := r.db.QueryContext(ctx, query, models.EntityTypeFeature, models.EntityTypeTask)
	if err != nil {
		return nil, fmt.Errorf("query portfolio child states: %w", err)
	}
	defer rows.Close()

	result := make([]ChildStateRow, 0)
	for rows.Next() {
		var row ChildStateRow
		var progress sql.NullFloat64
		if err := rows.Scan(
			&row.EpicID,
			&row.EpicKey,
			&row.EntityType,
			&row.EntityKey,
			&row.Title,
			&row.Status,
			&row.DirectParentKey,
			&progress,
		); err != nil {
			return nil, fmt.Errorf("scan portfolio child state: %w", err)
		}
		if progress.Valid {
			row.ProgressPct = &progress.Float64
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portfolio child states: %w", err)
	}
	return result, nil
}

// ListEpicRelationships returns supported directed epic relationships. LEFT
// JOINs preserve rows with a missing endpoint so callers can report dangling
// evidence instead of silently discarding it.
func (r *Repository) ListEpicRelationships(ctx context.Context) ([]EpicRelationshipRow, error) {
	const query = `
		SELECT er.from_entity_id,
		       from_epic.key,
		       from_epic.status,
		       er.relationship_type,
		       er.to_entity_id,
		       to_epic.key,
		       to_epic.status
		FROM entity_relationships er
		LEFT JOIN epics from_epic ON from_epic.id = er.from_entity_id
		LEFT JOIN epics to_epic ON to_epic.id = er.to_entity_id
		WHERE er.from_entity_type = ?
		  AND er.to_entity_type = ?
		  AND er.relationship_type IN (?, ?, ?)
		ORDER BY COALESCE(from_epic.key, '') ASC,
		         er.relationship_type ASC,
		         COALESCE(to_epic.key, '') ASC,
		         er.from_entity_id ASC,
		         er.to_entity_id ASC
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		models.EntityTypeEpic,
		models.EntityTypeEpic,
		models.EntityRelDependsOn,
		models.EntityRelBlocks,
		models.EntityRelFollows,
	)
	if err != nil {
		return nil, fmt.Errorf("query portfolio epic relationships: %w", err)
	}
	defer rows.Close()

	result := make([]EpicRelationshipRow, 0)
	for rows.Next() {
		var row EpicRelationshipRow
		var fromKey, fromStatus, toKey, toStatus sql.NullString
		if err := rows.Scan(
			&row.FromEpicID,
			&fromKey,
			&fromStatus,
			&row.RelationshipType,
			&row.ToEpicID,
			&toKey,
			&toStatus,
		); err != nil {
			return nil, fmt.Errorf("scan portfolio epic relationship: %w", err)
		}
		row.FromKey = nullStringPointer(fromKey)
		row.FromStatus = nullStringPointer(fromStatus)
		row.ToKey = nullStringPointer(toKey)
		row.ToStatus = nullStringPointer(toStatus)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portfolio epic relationships: %w", err)
	}
	return result, nil
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
