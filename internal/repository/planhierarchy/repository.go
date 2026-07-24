// Package planhierarchy provides one-query direct-child snapshots for
// one-level hierarchy planning (`shark plan <epic|feature>`).
package planhierarchy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// Dependency is one prerequisite and its current workflow status.
type Dependency struct {
	Key    string `json:"key"`
	Status string `json:"status"`
}

// Child is the bounded direct-child evidence needed for selection.
type Child struct {
	Key            string
	Title          string
	Status         string
	EntityType     models.EntityType
	ExecutionOrder *int
	Priority       *int
	Claimed        bool
	Dependencies   []Dependency
}

// Snapshot is one parent and all of its direct children.
type Snapshot struct {
	ParentFound bool
	Children    []Child
}

// Repository reads direct hierarchy children without per-child database calls.
type Repository struct {
	db *dbconn.DB
}

// NewRepository constructs a direct-child snapshot repository.
func NewRepository(db *dbconn.DB) *Repository {
	return &Repository{db: db}
}

// ReadDirectChildren loads exactly one hierarchy edge in one database query.
// Claim expiry is evaluated in SQL using the caller's configured TTL.
func (r *Repository) ReadDirectChildren(
	ctx context.Context,
	parentType, parentKey string,
	claimTTL time.Duration,
	evaluatedAt time.Time,
) (Snapshot, error) {
	switch parentType {
	case string(models.EntityTypeEpic):
		return r.readEpicFeatures(ctx, parentKey, claimTTL, evaluatedAt)
	case string(models.EntityTypeFeature):
		return r.readFeatureTasks(ctx, parentKey, claimTTL, evaluatedAt)
	default:
		return Snapshot{Children: []Child{}}, nil
	}
}

func (r *Repository) readEpicFeatures(
	ctx context.Context,
	epicKey string,
	claimTTL time.Duration,
	evaluatedAt time.Time,
) (Snapshot, error) {
	const query = `
		SELECT e.id,
		       f.key,
		       f.title,
		       f.status,
		       f.execution_order,
		       CASE
		         WHEN c.id IS NULL THEN 0
		         WHEN ? = 1 THEN 1
		         WHEN julianday(c.last_heartbeat) >= julianday(?) THEN 1
		         ELSE 0
		       END AS claimed
		FROM epics e
		LEFT JOIN features f ON f.epic_id = e.id
		LEFT JOIN entity_claims c
		  ON c.entity_type = ? AND c.entity_key = f.key
		WHERE e.key = ?
		ORDER BY f.execution_order IS NULL,
		         f.execution_order ASC,
		         f.created_at ASC,
		         f.key ASC
	`
	rows, err := r.db.QueryContext(
		ctx,
		query,
		claimNeverExpires(claimTTL),
		claimCutoff(claimTTL, evaluatedAt),
		models.EntityTypeFeature,
		epicKey,
	)
	if err != nil {
		return Snapshot{}, fmt.Errorf("query direct features for epic %s: %w", epicKey, err)
	}
	defer rows.Close()

	snapshot := Snapshot{Children: []Child{}}
	for rows.Next() {
		var (
			parentID                int64
			key, title, status      sql.NullString
			executionOrder, claimed sql.NullInt64
		)
		if err := rows.Scan(
			&parentID,
			&key,
			&title,
			&status,
			&executionOrder,
			&claimed,
		); err != nil {
			return Snapshot{}, fmt.Errorf("scan direct feature snapshot for epic %s: %w", epicKey, err)
		}
		snapshot.ParentFound = parentID > 0
		if !key.Valid {
			continue
		}
		snapshot.Children = append(snapshot.Children, Child{
			Key:            key.String,
			Title:          title.String,
			Status:         status.String,
			EntityType:     models.EntityTypeFeature,
			ExecutionOrder: nullableInt(executionOrder),
			Claimed:        claimed.Valid && claimed.Int64 != 0,
			Dependencies:   []Dependency{},
		})
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("iterate direct features for epic %s: %w", epicKey, err)
	}
	return snapshot, nil
}

func (r *Repository) readFeatureTasks(
	ctx context.Context,
	featureKey string,
	claimTTL time.Duration,
	evaluatedAt time.Time,
) (Snapshot, error) {
	const query = `
		SELECT f.id,
		       t.key,
		       t.title,
		       t.status,
		       t.execution_order,
		       t.priority,
		       CASE
		         WHEN c.id IS NULL THEN 0
		         WHEN ? = 1 THEN 1
		         WHEN julianday(c.last_heartbeat) >= julianday(?) THEN 1
		         ELSE 0
		       END AS claimed,
		       COALESCE((
		         SELECT json_group_array(json_object(
		           'key', deps.key,
		           'status', deps.status
		         ))
		         FROM (
		           SELECT dependency.key, dependency.status
		           FROM tasks dependency
		           JOIN json_each(COALESCE(t.depends_on, '[]')) legacy
		             ON dependency.key = legacy.value
		           UNION
		           SELECT dependency.key, dependency.status
		           FROM entity_relationships er
		           JOIN tasks dependency ON dependency.id = er.to_entity_id
		           WHERE er.from_entity_type = ?
		             AND er.from_entity_id = t.id
		             AND er.to_entity_type = ?
		             AND er.relationship_type = ?
		         ) deps
		       ), '[]') AS dependencies_json
		FROM features f
		LEFT JOIN tasks t ON t.feature_id = f.id
		LEFT JOIN entity_claims c
		  ON c.entity_type = ? AND c.entity_key = t.key
		WHERE f.key = ?
		ORDER BY t.execution_order IS NULL,
		         t.execution_order ASC,
		         t.priority ASC,
		         t.created_at ASC,
		         t.key ASC
	`
	rows, err := r.db.QueryContext(
		ctx,
		query,
		claimNeverExpires(claimTTL),
		claimCutoff(claimTTL, evaluatedAt),
		models.EntityTypeTask,
		models.EntityTypeTask,
		models.EntityRelDependsOn,
		models.EntityTypeTask,
		featureKey,
	)
	if err != nil {
		return Snapshot{}, fmt.Errorf("query direct tasks for feature %s: %w", featureKey, err)
	}
	defer rows.Close()

	snapshot := Snapshot{Children: []Child{}}
	for rows.Next() {
		var (
			parentID                         int64
			key, title, status, dependencies sql.NullString
			executionOrder, priority         sql.NullInt64
			claimed                          sql.NullInt64
		)
		if err := rows.Scan(
			&parentID,
			&key,
			&title,
			&status,
			&executionOrder,
			&priority,
			&claimed,
			&dependencies,
		); err != nil {
			return Snapshot{}, fmt.Errorf("scan direct task snapshot for feature %s: %w", featureKey, err)
		}
		snapshot.ParentFound = parentID > 0
		if !key.Valid {
			continue
		}
		child := Child{
			Key:            key.String,
			Title:          title.String,
			Status:         status.String,
			EntityType:     models.EntityTypeTask,
			ExecutionOrder: nullableInt(executionOrder),
			Priority:       nullableInt(priority),
			Claimed:        claimed.Valid && claimed.Int64 != 0,
			Dependencies:   []Dependency{},
		}
		if dependencies.Valid {
			if err := json.Unmarshal([]byte(dependencies.String), &child.Dependencies); err != nil {
				return Snapshot{}, fmt.Errorf(
					"decode dependencies for task %s under feature %s: %w",
					child.Key,
					featureKey,
					err,
				)
			}
		}
		snapshot.Children = append(snapshot.Children, child)
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("iterate direct tasks for feature %s: %w", featureKey, err)
	}
	return snapshot, nil
}

func claimNeverExpires(ttl time.Duration) int {
	if ttl <= 0 {
		return 1
	}
	return 0
}

func claimCutoff(ttl time.Duration, evaluatedAt time.Time) string {
	if ttl <= 0 {
		return dbconn.FormatTime(time.Unix(0, 0))
	}
	return dbconn.FormatTime(evaluatedAt.UTC().Add(-ttl))
}

func nullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}
