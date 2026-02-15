# Data Design: Template Variables for Related Docs and Tasks

**Feature:** E07-F29
**Version:** 1.0
**Last Updated:** 2026-02-13

## Database Schema

### New Tables

#### 1. feature_relationships

**Purpose:** Store typed relationships between features (depends_on, blocks, related_to, etc.)

```sql
CREATE TABLE feature_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_feature_id INTEGER NOT NULL,
    to_feature_id INTEGER NOT NULL,
    relationship_type TEXT CHECK (relationship_type IN (
        'depends_on',    -- from_feature depends on to_feature completing
        'blocks',        -- from_feature blocks to_feature from proceeding
        'related_to',    -- Features share common code/concerns
        'follows',       -- from_feature naturally follows to_feature
        'spawned_from',  -- from_feature created from UAT/bugs in to_feature
        'duplicates',    -- Features represent duplicate work
        'references'     -- from_feature consults/uses output of to_feature
    )) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_feature_id) REFERENCES features(id) ON DELETE CASCADE,
    FOREIGN KEY (to_feature_id) REFERENCES features(id) ON DELETE CASCADE,
    UNIQUE(from_feature_id, to_feature_id, relationship_type)
);

CREATE INDEX idx_feature_relationships_from ON feature_relationships(from_feature_id);
CREATE INDEX idx_feature_relationships_to ON feature_relationships(to_feature_id);
CREATE INDEX idx_feature_relationships_type ON feature_relationships(relationship_type);
```

**Constraints:**
- UNIQUE on (from_feature_id, to_feature_id, relationship_type) prevents duplicates
- CHECK constraint enforces valid relationship types
- CASCADE DELETE removes relationships when features are deleted
- Indexes on both directions enable efficient bidirectional queries

#### 2. epic_relationships

**Purpose:** Store typed relationships between epics

```sql
CREATE TABLE epic_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_epic_id INTEGER NOT NULL,
    to_epic_id INTEGER NOT NULL,
    relationship_type TEXT CHECK (relationship_type IN (
        'depends_on', 'blocks', 'related_to', 'follows',
        'spawned_from', 'duplicates', 'references'
    )) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    FOREIGN KEY (to_epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    UNIQUE(from_epic_id, to_epic_id, relationship_type)
);

CREATE INDEX idx_epic_relationships_from ON epic_relationships(from_epic_id);
CREATE INDEX idx_epic_relationships_to ON epic_relationships(to_epic_id);
CREATE INDEX idx_epic_relationships_type ON epic_relationships(relationship_type);
```

**Same constraints as feature_relationships**

---

## Existing Schema Extensions

### ContextData Model (internal/models/context_data.go)

**Extension to existing model:**

```go
type ContextData struct {
    Progress                 *ProgressContext             `json:"progress,omitempty"`
    ImplementationDecisions  map[string]string            `json:"implementation_decisions,omitempty"`
    OpenQuestions            []string                     `json:"open_questions,omitempty"`
    Blockers                 []BlockerContext             `json:"blockers,omitempty"`
    AcceptanceCriteriaStatus []AcceptanceCriterionContext `json:"acceptance_criteria_status,omitempty"`
    RelatedTasks             []string                     `json:"related_tasks,omitempty"`     // Existing
    RelatedFeatures          []string                     `json:"related_features,omitempty"`  // NEW
    RelatedEpics             []string                     `json:"related_epics,omitempty"`     // NEW
}
```

**Changes:**
- Add 2 new fields: `RelatedFeatures`, `RelatedEpics`
- Both are `omitempty` for backward compatibility
- Existing JSON without new fields remains valid
- Validation in `Validate()` method unchanged (arrays can be nil/empty)

---

## Data Access Patterns

### 1. Fetch Related Documents for Task

```sql
SELECT d.file_path
FROM documents d
INNER JOIN task_documents td ON d.id = td.document_id
WHERE td.task_id = ?
ORDER BY d.created_at ASC;
```

**Performance:** Uses existing index `idx_task_documents_task_id`
**Expected:** < 10ms for up to 50 documents

### 2. Fetch Related Features for Feature

```sql
SELECT DISTINCT f.key
FROM features f
INNER JOIN feature_relationships fr ON (
    f.id = fr.to_feature_id OR f.id = fr.from_feature_id
)
WHERE fr.from_feature_id = ? OR fr.to_feature_id = ?
ORDER BY f.key ASC;
```

**Performance:** Uses new indexes `idx_feature_relationships_from`, `idx_feature_relationships_to`
**Expected:** < 15ms for up to 20 relationships

### 3. Parse Related Tasks from Context Data

```go
// In-memory JSON parsing (no SQL query)
func extractRelatedTasksFromContext(contextData *string) ([]string, error) {
    if contextData == nil || *contextData == "" {
        return []string{}, nil
    }

    cd, err := models.FromJSON(*contextData)
    if err != nil {
        return []string{}, fmt.Errorf("failed to parse context_data: %w", err)
    }

    return cd.RelatedTasks, nil
}
```

**Performance:** Negligible (in-memory parsing), < 1ms

---

## Data Relationships

### Entity Relationship Diagram

```
┌──────────────┐
│    epics     │
│──────────────│
│ id           │◄─────┐
│ key          │      │
│ title        │      │ CASCADE DELETE
└──────────────┘      │
                      │
                 ┌────┴──────────────────┐
                 │ epic_relationships     │
                 │────────────────────────│
                 │ id                     │
                 │ from_epic_id (FK) ────►│
                 │ to_epic_id (FK) ───────┘
                 │ relationship_type      │
                 │ created_at             │
                 └────────────────────────┘

┌──────────────┐
│  features    │
│──────────────│
│ id           │◄─────┐
│ key          │      │
│ title        │      │ CASCADE DELETE
└──────────────┘      │
                      │
                 ┌────┴──────────────────┐
                 │feature_relationships   │
                 │────────────────────────│
                 │ id                     │
                 │ from_feature_id (FK) ──►
                 │ to_feature_id (FK) ────┘
                 │ relationship_type      │
                 │ created_at             │
                 └────────────────────────┘

┌──────────────┐
│    tasks     │
│──────────────│
│ id           │
│ key          │
│ title        │
│ context_data │──► {"related_tasks": ["E07-F01-001"], ...}
└──────────────┘

┌──────────────┐          ┌──────────────────┐
│  documents   │          │ task_documents   │
│──────────────│◄─────────┤──────────────────│
│ id           │          │ task_id (FK)     │
│ file_path    │          │ document_id (FK) │
└──────────────┘          └──────────────────┘
```

---

## Migration Strategy

### Auto-Migration (internal/db/migrate.go)

**Add to existing migration system:**

```go
// Migration 007: Add feature and epic relationship tables
func migration007FeatureEpicRelationships(db *sql.DB) error {
    // Check if feature_relationships exists
    var tableName string
    err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='feature_relationships'").Scan(&tableName)
    if err != sql.ErrNoRows {
        // Table already exists, skip
        return nil
    }

    // Create feature_relationships table
    _, err = db.Exec(`
        CREATE TABLE feature_relationships (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            from_feature_id INTEGER NOT NULL,
            to_feature_id INTEGER NOT NULL,
            relationship_type TEXT CHECK (relationship_type IN (
                'depends_on', 'blocks', 'related_to', 'follows',
                'spawned_from', 'duplicates', 'references'
            )) NOT NULL,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (from_feature_id) REFERENCES features(id) ON DELETE CASCADE,
            FOREIGN KEY (to_feature_id) REFERENCES features(id) ON DELETE CASCADE,
            UNIQUE(from_feature_id, to_feature_id, relationship_type)
        )
    `)
    if err != nil {
        return fmt.Errorf("failed to create feature_relationships table: %w", err)
    }

    // Create indexes
    _, err = db.Exec("CREATE INDEX idx_feature_relationships_from ON feature_relationships(from_feature_id)")
    // ... other indexes

    // Same for epic_relationships
    // ...

    return nil
}
```

**Backward Compatibility:**
- Migration runs automatically on `InitDB()`
- Safe to run multiple times (checks if table exists)
- No data migration needed (tables start empty)
- Existing databases continue working without new tables

---

## Data Validation

### Relationship Type Validation

**At Database Level:**
```sql
CHECK (relationship_type IN (
    'depends_on', 'blocks', 'related_to', 'follows',
    'spawned_from', 'duplicates', 'references'
))
```

**At Application Level:**
```go
func (r *FeatureRelationship) Validate() error {
    validTypes := map[string]bool{
        "depends_on": true, "blocks": true, "related_to": true,
        "follows": true, "spawned_from": true, "duplicates": true,
        "references": true,
    }

    if !validTypes[r.RelationshipType] {
        return fmt.Errorf("invalid relationship type: %s", r.RelationshipType)
    }

    if r.FromFeatureID == r.ToFeatureID {
        return errors.New("cannot create self-referential relationship")
    }

    if r.FromFeatureID <= 0 || r.ToFeatureID <= 0 {
        return errors.New("feature IDs must be positive integers")
    }

    return nil
}
```

### ContextData Validation

**Existing validation extended:**
```go
func (cd *ContextData) Validate() error {
    // ... existing validation ...

    // NEW: Validate related features (if provided)
    for _, featureKey := range cd.RelatedFeatures {
        if !isValidFeatureKey(featureKey) {
            return fmt.Errorf("invalid feature key in related_features: %s", featureKey)
        }
    }

    // NEW: Validate related epics (if provided)
    for _, epicKey := range cd.RelatedEpics {
        if !isValidEpicKey(epicKey) {
            return fmt.Errorf("invalid epic key in related_epics: %s", epicKey)
        }
    }

    return nil
}

func isValidFeatureKey(key string) bool {
    // Feature keys: E07-F01, F01, E07-F01-slug
    return regexp.MustCompile(`^(E\d+-)?F\d+(-[\w-]+)?$`).MatchString(key)
}

func isValidEpicKey(key string) bool {
    // Epic keys: E07, E07-slug
    return regexp.MustCompile(`^E\d+(-[\w-]+)?$`).MatchString(key)
}
```

---

## Summary

**New Tables:**
- `feature_relationships` (7 relationship types, CASCADE delete, indexes)
- `epic_relationships` (same structure as features)

**Model Extensions:**
- `ContextData.RelatedFeatures` (JSON array)
- `ContextData.RelatedEpics` (JSON array)

**Access Patterns:**
- Document lookup: Uses existing indexes (< 10ms)
- Relationship lookup: New indexes enable fast bidirectional queries (< 15ms)
- Context data parsing: In-memory JSON parsing (< 1ms)

**Migration:**
- Auto-migration adds tables on first run
- Backward compatible (tables optional, context fields optional)
- No data migration required
