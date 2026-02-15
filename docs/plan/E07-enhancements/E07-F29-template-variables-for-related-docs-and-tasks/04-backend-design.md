# Backend Design: Template Variables for Related Docs and Tasks

**Feature:** E07-F29
**Version:** 1.0
**Last Updated:** 2026-02-13

## Component Architecture

### Layer Responsibilities

```
┌─────────────────────────────────────────────────┐
│         Service Layer (Business Logic)          │
│  - OrchestratorActionService                    │
│  - Generates instructions with placeholders     │
│  - Calls placeholder factories                  │
└─────────────────────┬───────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│    Template Helper Layer (Placeholder Logic)    │
│  - TaskPlaceholdersWithRelated()                │
│  - FeaturePlaceholdersWithRelated()             │
│  - EpicPlaceholdersWithRelated()                │
│  - formatDocPathsAsCSV()                        │
│  - extractRelatedTasksFromContext()             │
└─────────────────────┬───────────────────────────┘
                      │
        ┌─────────────┼──────────────┬──────────────┐
        ▼             ▼              ▼              ▼
┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐
│ Document   │ │  Feature   │ │   Epic     │ │   Task     │
│ Repository │ │Relationship│ │Relationship│ │ Repository │
│            │ │ Repository │ │ Repository │ │            │
└────────────┘ └────────────┘ └────────────┘ └────────────┘
```

---

## New Components

### 1. FeatureRelationshipRepository

**Location:** `internal/repository/feature_relationship_repository.go`

**Purpose:** CRUD operations for feature-to-feature relationships

**Interface:**
```go
type FeatureRelationshipRepository struct {
    db *DB
}

func NewFeatureRelationshipRepository(db *DB) *FeatureRelationshipRepository

// Core CRUD
func (r *FeatureRelationshipRepository) Create(ctx context.Context, rel *models.FeatureRelationship) error
func (r *FeatureRelationshipRepository) GetByID(ctx context.Context, id int64) (*models.FeatureRelationship, error)
func (r *FeatureRelationshipRepository) Delete(ctx context.Context, id int64) error

// Query methods
func (r *FeatureRelationshipRepository) GetByFeatureID(ctx context.Context, featureID int64) ([]*models.FeatureRelationship, error)
func (r *FeatureRelationshipRepository) GetOutgoing(ctx context.Context, featureID int64, relTypes []string) ([]*models.FeatureRelationship, error)
func (r *FeatureRelationshipRepository) GetIncoming(ctx context.Context, featureID int64, relTypes []string) ([]*models.FeatureRelationship, error)

// Helper for placeholder population
func (r *FeatureRelationshipRepository) ListRelatedFeatures(ctx context.Context, featureID int64) ([]string, error)
```

**Key Method Implementation:**
```go
// ListRelatedFeatures returns feature keys (not full objects) for placeholder population
func (r *FeatureRelationshipRepository) ListRelatedFeatures(
    ctx context.Context,
    featureID int64,
) ([]string, error) {
    query := `
        SELECT DISTINCT f.key
        FROM features f
        INNER JOIN feature_relationships fr ON (
            f.id = fr.to_feature_id OR f.id = fr.from_feature_id
        )
        WHERE fr.from_feature_id = ? OR fr.to_feature_id = ?
        ORDER BY f.key ASC
    `

    rows, err := r.db.QueryContext(ctx, query, featureID, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to list related features: %w", err)
    }
    defer rows.Close()

    var keys []string
    for rows.Next() {
        var key string
        if err := rows.Scan(&key); err != nil {
            return nil, fmt.Errorf("failed to scan feature key: %w", err)
        }
        keys = append(keys, key)
    }

    return keys, rows.Err()
}
```

### 2. EpicRelationshipRepository

**Location:** `internal/repository/epic_relationship_repository.go`

**Same interface pattern as FeatureRelationshipRepository**

**Key difference:** Operates on `epic_relationships` table instead of `feature_relationships`

### 3. Extended Placeholder Functions

**Location:** `internal/config/template_helpers.go`

**New Functions:**

```go
// TaskPlaceholdersWithRelated extends TaskPlaceholders with relationship data
func TaskPlaceholdersWithRelated(
    ctx context.Context,
    task *models.Task,
    docRepo *repository.DocumentRepository,
) (map[string]string, error) {
    // Start with basic placeholders
    placeholders := TaskPlaceholders(task)

    // Add related_docs
    docs, err := docRepo.ListForTask(ctx, task.ID)
    if err != nil {
        log.Printf("WARNING: Failed to fetch related docs for task %s: %v", task.Key, err)
        placeholders["related_docs"] = ""
    } else {
        placeholders["related_docs"] = formatDocPathsAsCSV(docs)
    }

    // Add related_tasks from context_data
    placeholders["related_tasks"] = extractRelatedTasksFromContext(task.ContextData)

    return placeholders, nil
}

// FeaturePlaceholdersWithRelated extends FeaturePlaceholders with relationships
func FeaturePlaceholdersWithRelated(
    ctx context.Context,
    feature *models.Feature,
    docRepo *repository.DocumentRepository,
    featureRelRepo *repository.FeatureRelationshipRepository,
) (map[string]string, error) {
    placeholders := FeaturePlaceholders(feature)

    // Add related_docs
    docs, err := docRepo.ListForFeature(ctx, feature.ID)
    if err != nil {
        log.Printf("WARNING: Failed to fetch related docs for feature %s: %v", feature.Key, err)
        placeholders["related_docs"] = ""
    } else {
        placeholders["related_docs"] = formatDocPathsAsCSV(docs)
    }

    // Add related_features from relationship table
    relatedKeys, err := featureRelRepo.ListRelatedFeatures(ctx, feature.ID)
    if err != nil {
        log.Printf("WARNING: Failed to fetch related features for %s: %v", feature.Key, err)
        placeholders["related_features"] = ""
    } else {
        placeholders["related_features"] = strings.Join(relatedKeys, ",")
    }

    // Add related_epics from context_data
    placeholders["related_epics"] = extractRelatedEpicsFromContext(feature.ContextData)

    return placeholders, nil
}

// EpicPlaceholdersWithRelated extends EpicPlaceholders with relationships
func EpicPlaceholdersWithRelated(
    ctx context.Context,
    epic *models.Epic,
    docRepo *repository.DocumentRepository,
    epicRelRepo *repository.EpicRelationshipRepository,
) (map[string]string, error) {
    placeholders := EpicPlaceholders(epic)

    // Add related_docs
    docs, err := docRepo.ListForEpic(ctx, epic.ID)
    if err != nil {
        log.Printf("WARNING: Failed to fetch related docs for epic %s: %v", epic.Key, err)
        placeholders["related_docs"] = ""
    } else {
        placeholders["related_docs"] = formatDocPathsAsCSV(docs)
    }

    // Add related_epics from relationship table
    relatedKeys, err := epicRelRepo.ListRelatedEpics(ctx, epic.ID)
    if err != nil {
        log.Printf("WARNING: Failed to fetch related epics for %s: %v", epic.Key, err)
        placeholders["related_epics"] = ""
    } else {
        placeholders["related_epics"] = strings.Join(relatedKeys, ",")
    }

    return placeholders, nil
}
```

**Helper Functions:**

```go
// formatDocPathsAsCSV extracts file paths and joins with commas
func formatDocPathsAsCSV(docs []*models.Document) string {
    if len(docs) == 0 {
        return ""
    }

    paths := make([]string, len(docs))
    for i, doc := range docs {
        paths[i] = doc.FilePath
    }

    return strings.Join(paths, ",")
}

// extractRelatedTasksFromContext parses RelatedTasks from context_data JSON
func extractRelatedTasksFromContext(contextData *string) string {
    if contextData == nil || *contextData == "" {
        return ""
    }

    cd, err := models.FromJSON(*contextData)
    if err != nil {
        log.Printf("WARNING: Failed to parse context_data JSON (returning empty): %v", err)
        return ""  // Graceful degradation
    }

    if len(cd.RelatedTasks) == 0 {
        return ""
    }

    return strings.Join(cd.RelatedTasks, ",")
}

// extractRelatedFeaturesFromContext parses RelatedFeatures from context_data JSON
func extractRelatedFeaturesFromContext(contextData *string) string {
    if contextData == nil || *contextData == "" {
        return ""
    }

    cd, err := models.FromJSON(*contextData)
    if err != nil {
        log.Printf("WARNING: Failed to parse context_data JSON (returning empty): %v", err)
        return ""
    }

    if len(cd.RelatedFeatures) == 0 {
        return ""
    }

    return strings.Join(cd.RelatedFeatures, ",")
}

// extractRelatedEpicsFromContext parses RelatedEpics from context_data JSON
func extractRelatedEpicsFromContext(contextData *string) string {
    if contextData == nil || *contextData == "" {
        return ""
    }

    cd, err := models.FromJSON(*contextData)
    if err != nil {
        log.Printf("WARNING: Failed to parse context_data JSON (returning empty): %v", err)
        return ""
    }

    if len(cd.RelatedEpics) == 0 {
        return ""
    }

    return strings.Join(cd.RelatedEpics, ",")
}
```

---

## Modified Components

### 1. OrchestratorActionService

**Changes to use new placeholder functions:**

**Before:**
```go
func (s *OrchestratorActionService) GetActionForTask(task *models.Task, status string) (*OrchestratorAction, error) {
    template := s.config.GetInstructionTemplate(status)
    placeholders := config.TaskPlaceholders(task)
    instruction := config.PopulateTemplate(template, placeholders)
    // ...
}
```

**After:**
```go
func (s *OrchestratorActionService) GetActionForTask(
    ctx context.Context,
    task *models.Task,
    status string,
) (*OrchestratorAction, error) {
    template := s.config.GetInstructionTemplate(status)

    // Use extended placeholder function with document repository
    placeholders, err := config.TaskPlaceholdersWithRelated(ctx, task, s.docRepo)
    if err != nil {
        // Log error but continue with basic placeholders
        log.Printf("WARNING: Using basic placeholders for task %s due to error: %v", task.Key, err)
        placeholders = config.TaskPlaceholders(task)
    }

    instruction := config.PopulateTemplate(template, placeholders)
    // ...
}
```

**Dependency Injection:**
```go
type OrchestratorActionService struct {
    config             *WorkflowConfig
    docRepo            *repository.DocumentRepository     // NEW
    featureRelRepo     *repository.FeatureRelationshipRepository  // NEW (for features)
    epicRelRepo        *repository.EpicRelationshipRepository     // NEW (for epics)
}

func NewOrchestratorActionService(
    config *WorkflowConfig,
    docRepo *repository.DocumentRepository,
    featureRelRepo *repository.FeatureRelationshipRepository,
    epicRelRepo *repository.EpicRelationshipRepository,
) *OrchestratorActionService {
    return &OrchestratorActionService{
        config:         config,
        docRepo:        docRepo,
        featureRelRepo: featureRelRepo,
        epicRelRepo:    epicRelRepo,
    }
}
```

### 2. TaskRepository (GetOrchestratorActionForTask)

**Updated to inject document repository:**

```go
func (r *TaskRepository) GetOrchestratorActionForTask(
    ctx context.Context,
    taskID int64,
) (*config.OrchestratorAction, error) {
    task, err := r.GetByID(ctx, taskID)
    if err != nil {
        return nil, err
    }

    // Create document repository (injected or passed via constructor)
    docRepo := repository.NewDocumentRepository(r.db)

    // Use extended placeholder function
    placeholders, err := config.TaskPlaceholdersWithRelated(ctx, task, docRepo)
    if err != nil {
        log.Printf("WARNING: Failed to get extended placeholders for task %d: %v", taskID, err)
        placeholders = config.TaskPlaceholders(task)
    }

    // Get template and populate
    template := r.workflowConfig.GetInstructionTemplate(string(task.Status))
    instruction := config.PopulateTemplate(template, placeholders)

    return &config.OrchestratorAction{
        Instruction: instruction,
        Status:      string(task.Status),
    }, nil
}
```

---

## Error Handling Patterns

### Graceful Degradation

**All placeholder population errors degrade gracefully:**

```go
// Pattern: Try to fetch related data, fallback to empty string on error
docs, err := docRepo.ListForTask(ctx, task.ID)
if err != nil {
    log.Printf("WARNING: Failed to fetch related docs for task %s: %v", task.Key, err)
    placeholders["related_docs"] = ""  // Empty string, not error
} else {
    placeholders["related_docs"] = formatDocPathsAsCSV(docs)
}
```

**Why:**
- Template population should never fail due to relationship queries
- Missing relationships are valid (task may have no docs)
- Orchestrator actions must always be generated

### Error Logging Strategy

```go
// WARNING level for missing/failed relationship data
log.Printf("WARNING: Failed to fetch related docs for task %s: %v", task.Key, err)

// INFO level for successful operations
log.Printf("INFO: Populated placeholders for task %s with %d related docs", task.Key, len(docs))

// ERROR level for critical failures (never happens in placeholder population)
log.Printf("ERROR: Orchestrator action generation failed: %v", err)
```

---

## Testing Strategy

### Unit Tests

**Template Helper Tests:**
```go
// internal/config/template_helpers_test.go

func TestTaskPlaceholdersWithRelated_WithDocs(t *testing.T) {
    // Mock document repository
    mockDocRepo := &MockDocumentRepository{
        ListForTaskFunc: func(ctx context.Context, taskID int64) ([]*models.Document, error) {
            return []*models.Document{
                {FilePath: "docs/spec.md"},
                {FilePath: "docs/design.md"},
            }, nil
        },
    }

    task := &models.Task{Key: "E07-F29-001", Title: "Test Task"}

    placeholders, err := TaskPlaceholdersWithRelated(context.Background(), task, mockDocRepo)
    assert.NoError(t, err)
    assert.Equal(t, "docs/spec.md,docs/design.md", placeholders["related_docs"])
}

func TestTaskPlaceholdersWithRelated_NoDocs(t *testing.T) {
    mockDocRepo := &MockDocumentRepository{
        ListForTaskFunc: func(ctx context.Context, taskID int64) ([]*models.Document, error) {
            return []*models.Document{}, nil  // Empty array
        },
    }

    task := &models.Task{Key: "E07-F29-001"}

    placeholders, err := TaskPlaceholdersWithRelated(context.Background(), task, mockDocRepo)
    assert.NoError(t, err)
    assert.Equal(t, "", placeholders["related_docs"])  // Empty string for no docs
}

func TestTaskPlaceholdersWithRelated_ErrorDegradesgracefully(t *testing.T) {
    mockDocRepo := &MockDocumentRepository{
        ListForTaskFunc: func(ctx context.Context, taskID int64) ([]*models.Document, error) {
            return nil, errors.New("database connection failed")
        },
    }

    task := &models.Task{Key: "E07-F29-001"}

    placeholders, err := TaskPlaceholdersWithRelated(context.Background(), task, mockDocRepo)
    assert.NoError(t, err)  // No error returned (graceful degradation)
    assert.Equal(t, "", placeholders["related_docs"])  // Empty string fallback
}
```

**Repository Tests (Integration):**
```go
// internal/repository/feature_relationship_repository_test.go

func TestFeatureRelationshipRepository_ListRelatedFeatures(t *testing.T) {
    ctx := context.Background()
    db := test.GetTestDB()
    repo := NewFeatureRelationshipRepository(db)
    featureRepo := NewFeatureRepository(db)

    // Clean up
    _, _ = db.ExecContext(ctx, "DELETE FROM feature_relationships")
    _, _ = db.ExecContext(ctx, "DELETE FROM features")

    // Create test features
    f1 := &models.Feature{Key: "E07-F01", Title: "Feature 1"}
    f2 := &models.Feature{Key: "E07-F05", Title: "Feature 2"}
    f3 := &models.Feature{Key: "E10-F05", Title: "Feature 3"}

    featureRepo.Create(ctx, f1)
    featureRepo.Create(ctx, f2)
    featureRepo.Create(ctx, f3)

    // Create relationships
    repo.Create(ctx, &models.FeatureRelationship{
        FromFeatureID:    f1.ID,
        ToFeatureID:      f2.ID,
        RelationshipType: "depends_on",
    })
    repo.Create(ctx, &models.FeatureRelationship{
        FromFeatureID:    f1.ID,
        ToFeatureID:      f3.ID,
        RelationshipType: "references",
    })

    // List related features for f1
    keys, err := repo.ListRelatedFeatures(ctx, f1.ID)
    assert.NoError(t, err)
    assert.Equal(t, []string{"E07-F05", "E10-F05"}, keys)
}
```

---

## Performance Optimization

### Query Optimization

**Use indexes for all relationship queries:**

```sql
-- Feature relationship query uses composite index
EXPLAIN QUERY PLAN
SELECT DISTINCT f.key
FROM features f
INNER JOIN feature_relationships fr ON (f.id = fr.to_feature_id OR f.id = fr.from_feature_id)
WHERE fr.from_feature_id = ? OR fr.to_feature_id = ?;

-- Result: Uses idx_feature_relationships_from, idx_feature_relationships_to
```

### Caching Strategy (Future)

**Phase 2: Add placeholder cache:**

```go
type PlaceholderCache struct {
    cache map[string]map[string]string
    mu    sync.RWMutex
}

func (c *PlaceholderCache) Get(entityKey string) (map[string]string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    placeholders, exists := c.cache[entityKey]
    return placeholders, exists
}

func (c *PlaceholderCache) Set(entityKey string, placeholders map[string]string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cache[entityKey] = placeholders
}

func (c *PlaceholderCache) Invalidate(entityKey string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.cache, entityKey)
}
```

---

## Summary

**New Components:**
- `FeatureRelationshipRepository` (CRUD for feature relationships)
- `EpicRelationshipRepository` (CRUD for epic relationships)
- Extended placeholder functions (`*PlaceholdersWithRelated`)

**Modified Components:**
- `OrchestratorActionService` (inject document/relationship repos)
- `TaskRepository.GetOrchestratorActionForTask` (use extended placeholders)

**Error Handling:**
- Graceful degradation (empty strings on errors)
- Logging warnings for relationship query failures
- Never fail template population due to missing relationships

**Testing:**
- Unit tests for placeholder functions (mocked repos)
- Integration tests for relationship repositories (real DB)
- Test coverage target: ≥ 80%
