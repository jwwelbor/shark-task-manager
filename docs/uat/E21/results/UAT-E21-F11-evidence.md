# UAT Evidence: E21-F11 Polymorphic Entity Relationships

Collected: 2026-03-21

---

## Scenario 1: Schema and Migration

### CurrentSchemaVersion

File: `internal/db/db.go`, line 393:
```go
const CurrentSchemaVersion = 9
```

### Schema Migration Function

File: `internal/db/db.go`, lines 2847-2881:
```go
func migrateAddEntityRelationships(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS entity_relationships (
            id                INTEGER PRIMARY KEY AUTOINCREMENT,
            from_entity_type  TEXT NOT NULL CHECK(from_entity_type IN (
                                'epic','feature','task','bug','change'
                              )),
            from_entity_id    INTEGER NOT NULL,
            to_entity_type    TEXT NOT NULL CHECK(to_entity_type IN (
                                'epic','feature','task','bug','change'
                              )),
            to_entity_id      INTEGER NOT NULL,
            relationship_type TEXT NOT NULL CHECK(relationship_type IN (
                                'depends_on','blocks','related_to','follows',
                                'spawned_from','duplicates','references','linked_to'
                              )),
            created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(from_entity_type, from_entity_id,
                   to_entity_type,   to_entity_id, relationship_type)
        )`,
		`CREATE INDEX IF NOT EXISTS idx_er_from
             ON entity_relationships(from_entity_type, from_entity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_er_to
             ON entity_relationships(to_entity_type, to_entity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_er_type
             ON entity_relationships(relationship_type)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrateAddEntityRelationships: %w", err)
		}
	}
	return nil
}
```

### Migration Call Site

File: `internal/db/db.go`, lines 756-763:
```go
// Create polymorphic entity_relationships table (E21-F11)
if err := migrateAddEntityRelationships(db); err != nil {
    return fmt.Errorf("failed to migrate entity_relationships table: %w", err)
}
// Migrate data from legacy relationship tables into entity_relationships (E21-F11)
if err := migrateDataToEntityRelationships(db); err != nil {
    return fmt.Errorf("failed to migrate data to entity_relationships: %w", err)
}
```

### Schema Details

- Table: `entity_relationships`
- Columns: `id`, `from_entity_type`, `from_entity_id`, `to_entity_type`, `to_entity_id`, `relationship_type`, `created_at`
- CHECK constraints on `from_entity_type` and `to_entity_type`: `('epic','feature','task','bug','change')`
- CHECK constraint on `relationship_type`: `('depends_on','blocks','related_to','follows','spawned_from','duplicates','references','linked_to')`
- UNIQUE constraint: `(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)`
- 3 indexes: `idx_er_from`, `idx_er_to`, `idx_er_type`

---

## Scenario 2: Repository CRUD Operations

### File: `internal/repository/entity_relationship_repository.go` (245 lines)

#### Public Methods and Signatures

```go
func NewEntityRelationshipRepository(db *DB) *EntityRelationshipRepository
func (r *EntityRelationshipRepository) Create(ctx context.Context, rel *models.EntityRelationship) error
func (r *EntityRelationshipRepository) Delete(ctx context.Context, id int64) error
func (r *EntityRelationshipRepository) DeleteByEntitiesAndType(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error
func (r *EntityRelationshipRepository) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error)
func (r *EntityRelationshipRepository) GetOutgoing(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error)
func (r *EntityRelationshipRepository) GetIncoming(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error)
```

Private helper:
```go
func (r *EntityRelationshipRepository) scanRelationships(rows *sql.Rows) ([]*models.EntityRelationship, error)
```

#### Implementation Notes

- `Create`: Calls `rel.Validate()` before INSERT. Detects UNIQUE constraint failures and returns user-friendly error.
- `Delete`: Checks `RowsAffected` and returns "not found" if 0.
- `DeleteByEntitiesAndType`: Directed delete by from/to entity + type. Checks `RowsAffected`.
- `GetByEntity`: Returns both incoming and outgoing via OR clause.
- `GetOutgoing`/`GetIncoming`: Support optional type filtering via IN clause with dynamic placeholders.

### Test File: `internal/repository/entity_relationship_repository_test.go` (529 lines)

#### Test Functions

| Test Function | Subtests |
|---|---|
| `TestEntityRelationshipRepository_Create` | `happy_path`, `duplicate_detection`, `validation_-_zero_from_entity_id`, `validation_-_self-relationship`, `validation_-_invalid_relationship_type` |
| `TestEntityRelationshipRepository_Delete` | `happy_path`, `not_found` |
| `TestEntityRelationshipRepository_DeleteByEntitiesAndType` | `happy_path`, `not_found` |
| `TestEntityRelationshipRepository_GetByEntity` | `bidirectional_results_for_epic`, `bidirectional_results_for_feature`, `no_results_for_unknown_entity` |
| `TestEntityRelationshipRepository_GetOutgoing` | `without_type_filter`, `with_type_filter_-_single_type`, `with_type_filter_-_multiple_types`, `with_type_filter_-_no_matches` |
| `TestEntityRelationshipRepository_GetIncoming` | `without_type_filter`, `with_type_filter_-_single_type`, `with_type_filter_-_multiple_types`, `with_type_filter_-_no_matches` |

Tests use `cleanupEntityRelationships()` helper for cleanup, `test.GetTestDB()` and `test.SeedTestData()` for setup.

### Repository Test Output (raw)

```
=== RUN   TestEntityRelationshipRepository_Create
=== RUN   TestEntityRelationshipRepository_Create/happy_path
=== RUN   TestEntityRelationshipRepository_Create/duplicate_detection
=== RUN   TestEntityRelationshipRepository_Create/validation_-_zero_from_entity_id
=== RUN   TestEntityRelationshipRepository_Create/validation_-_self-relationship
=== RUN   TestEntityRelationshipRepository_Create/validation_-_invalid_relationship_type
--- PASS: TestEntityRelationshipRepository_Create (0.00s)
=== RUN   TestEntityRelationshipRepository_Delete
=== RUN   TestEntityRelationshipRepository_Delete/happy_path
=== RUN   TestEntityRelationshipRepository_Delete/not_found
--- PASS: TestEntityRelationshipRepository_Delete (0.00s)
=== RUN   TestEntityRelationshipRepository_DeleteByEntitiesAndType
=== RUN   TestEntityRelationshipRepository_DeleteByEntitiesAndType/happy_path
=== RUN   TestEntityRelationshipRepository_DeleteByEntitiesAndType/not_found
--- PASS: TestEntityRelationshipRepository_DeleteByEntitiesAndType (0.00s)
=== RUN   TestEntityRelationshipRepository_GetByEntity
=== RUN   TestEntityRelationshipRepository_GetByEntity/bidirectional_results_for_epic
=== RUN   TestEntityRelationshipRepository_GetByEntity/bidirectional_results_for_feature
=== RUN   TestEntityRelationshipRepository_GetByEntity/no_results_for_unknown_entity
--- PASS: TestEntityRelationshipRepository_GetByEntity (0.00s)
=== RUN   TestEntityRelationshipRepository_GetOutgoing
=== RUN   TestEntityRelationshipRepository_GetOutgoing/without_type_filter
=== RUN   TestEntityRelationshipRepository_GetOutgoing/with_type_filter_-_single_type
=== RUN   TestEntityRelationshipRepository_GetOutgoing/with_type_filter_-_multiple_types
=== RUN   TestEntityRelationshipRepository_GetOutgoing/with_type_filter_-_no_matches
--- PASS: TestEntityRelationshipRepository_GetOutgoing (0.00s)
=== RUN   TestEntityRelationshipRepository_GetIncoming
=== RUN   TestEntityRelationshipRepository_GetIncoming/without_type_filter
=== RUN   TestEntityRelationshipRepository_GetIncoming/with_type_filter_-_single_type
=== RUN   TestEntityRelationshipRepository_GetIncoming/with_type_filter_-_multiple_types
=== RUN   TestEntityRelationshipRepository_GetIncoming/with_type_filter_-_no_matches
--- PASS: TestEntityRelationshipRepository_GetIncoming (0.00s)
PASS
ok  	github.com/jwwelbor/shark-task-manager/internal/repository	0.016s
```

---

## Scenario 3: Cycle Detection

### File: `internal/services/entity_relationship_service.go`, lines 181-232

#### DetectCycle Implementation (DFS Algorithm)

```go
func (s *EntityRelationshipService) DetectCycle(
	ctx context.Context,
	fromType models.EntityType, fromID int64,
	toType models.EntityType, toID int64,
	relType models.EntityRelationshipType,
) (bool, error) {
	// Only check for cyclic relationship types and only when both entities are the same type
	if !models.CyclicRelationshipTypes[relType] || fromType != toType {
		return false, nil
	}

	// DFS from (toType, toID) following same relType edges.
	// If we reach (fromType, fromID), a cycle exists.
	type node struct {
		entityType models.EntityType
		entityID   int64
	}

	visited := map[node]bool{}
	stack := []node{{toType, toID}}
	target := node{fromType, fromID}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current == target {
			return true, nil
		}

		if visited[current] {
			continue
		}
		visited[current] = true

		// Get outgoing edges of same relType
		outgoing, err := s.repo.GetOutgoing(ctx, current.entityType, current.entityID,
			[]models.EntityRelationshipType{relType})
		if err != nil {
			return false, fmt.Errorf("cycle detection query failed: %w", err)
		}

		for _, rel := range outgoing {
			next := node{rel.ToEntityType, rel.ToEntityID}
			if !visited[next] {
				stack = append(stack, next)
			}
		}
	}

	return false, nil
}
```

#### Cycle Detection Trigger

File: `internal/services/entity_relationship_service.go`, lines 91-101:
```go
// Cycle detection for cyclic relationship types within same entity type
if rel.IsCyclic() {
    hasCycle, err := s.DetectCycle(ctx, fromType, fromID, toType, toID, relType)
    if err != nil {
        return nil, fmt.Errorf("cycle detection failed: %w", err)
    }
    if hasCycle {
        return nil, fmt.Errorf("cannot create relationship: would create a cycle (%s(%d) -[%s]-> %s(%d))",
            fromType, fromID, relType, toType, toID)
    }
}
```

#### CyclicRelationshipTypes

File: `internal/models/entity_relationship.go`, lines 51-54:
```go
var CyclicRelationshipTypes = map[EntityRelationshipType]bool{
	EntityRelDependsOn: true,
	EntityRelBlocks:    true,
}
```

Only `depends_on` and `blocks` trigger cycle detection, and only when `fromType == toType`.

### Cycle Detection Service Tests

File: `internal/services/entity_relationship_service_test.go`

| Test Function | What It Tests |
|---|---|
| `TestCreateRelationship_CycleDetected` (line 139) | A depends_on B exists; B depends_on A rejected |
| `TestCreateRelationship_NoCycleCheckForNonCyclicType` (line 168) | `related_to` skips cycle detection (GetOutgoing not called) |
| `TestCreateRelationship_NoCycleCheckForCrossEntityType` (line 198) | Cross-entity `depends_on` skips cycle detection |
| `TestDetectCycle_Simple` (line 371) | A->B exists, B->A cycle detected |
| `TestDetectCycle_Transitive` (line 399) | A->B->C exists, C->A cycle detected |
| `TestDetectCycle_NoCycle` (line 433) | A->B exists, C->A no cycle |
| `TestDetectCycle_NonCyclicRelType` (line 459) | `related_to` returns false immediately |
| `TestDetectCycle_CrossEntityType` (line 473) | Cross-entity type returns false immediately |
| `TestDetectCycle_RepoError` (line 487) | Repo error propagated |

### Service Test Output (raw)

```
=== RUN   TestNewEntityRelationshipService
=== RUN   TestNewEntityRelationshipService/panics_on_nil_repo
=== RUN   TestNewEntityRelationshipService/succeeds_with_valid_repo
--- PASS: TestNewEntityRelationshipService (0.00s)
=== RUN   TestCreateRelationship_Success
--- PASS: TestCreateRelationship_Success (0.00s)
=== RUN   TestCreateRelationship_ValidationError
--- PASS: TestCreateRelationship_ValidationError (0.00s)
=== RUN   TestCreateRelationship_SelfReference
--- PASS: TestCreateRelationship_SelfReference (0.00s)
=== RUN   TestCreateRelationship_CycleDetected
--- PASS: TestCreateRelationship_CycleDetected (0.00s)
=== RUN   TestCreateRelationship_NoCycleCheckForNonCyclicType
--- PASS: TestCreateRelationship_NoCycleCheckForNonCyclicType (0.00s)
=== RUN   TestCreateRelationship_NoCycleCheckForCrossEntityType
--- PASS: TestCreateRelationship_NoCycleCheckForCrossEntityType (0.00s)
=== RUN   TestDeleteRelationship
=== RUN   TestDeleteRelationship/success
=== RUN   TestDeleteRelationship/error_propagation
--- PASS: TestDeleteRelationship (0.00s)
=== RUN   TestUnlinkEntities
=== RUN   TestUnlinkEntities/success
=== RUN   TestUnlinkEntities/error_propagation
--- PASS: TestUnlinkEntities (0.00s)
=== RUN   TestGetRelationships
--- PASS: TestGetRelationships (0.00s)
=== RUN   TestGetOutgoing
--- PASS: TestGetOutgoing (0.00s)
=== RUN   TestGetIncoming
--- PASS: TestGetIncoming (0.00s)
=== RUN   TestDetectCycle_Simple
--- PASS: TestDetectCycle_Simple (0.00s)
=== RUN   TestDetectCycle_Transitive
--- PASS: TestDetectCycle_Transitive (0.00s)
=== RUN   TestDetectCycle_NoCycle
--- PASS: TestDetectCycle_NoCycle (0.00s)
=== RUN   TestDetectCycle_NonCyclicRelType
--- PASS: TestDetectCycle_NonCyclicRelType (0.00s)
=== RUN   TestDetectCycle_CrossEntityType
--- PASS: TestDetectCycle_CrossEntityType (0.00s)
=== RUN   TestDetectCycle_RepoError
--- PASS: TestDetectCycle_RepoError (0.00s)
=== RUN   TestCreateRelationship_RepoError
--- PASS: TestCreateRelationship_RepoError (0.00s)
=== RUN   TestGetRelationships_RepoError
--- PASS: TestGetRelationships_RepoError (0.00s)
PASS
ok  	github.com/jwwelbor/shark-task-manager/internal/services	0.020s
```

All 20 service tests pass.

---

## Scenario 4: Data Migration

### CLI Migration Command

File: `internal/cli/commands/migrate_relationships.go` (122 lines)

Command registration:
```go
var migrateRelationshipsCmd = &cobra.Command{
	Use:   "relationships",
	Short: "Migrate legacy relationship data to entity_relationships table",
	...
}

func init() {
	migrateCmd.AddCommand(migrateRelationshipsCmd)
}
```

Available via: `shark admin migrate relationships`

### Data Migration Function

File: `internal/db/db.go`, lines 2898-3066

Function: `MigrateDataToEntityRelationships(db *sql.DB) (map[string]int64, error)`

5 phases, all using `INSERT OR IGNORE` for idempotency, wrapped in a transaction:

| Phase | Source | Target | Relationship Type |
|---|---|---|---|
| Phase 1 | `task_relationships` | `entity_relationships` | Preserves original type |
| Phase 2 | `tasks.depends_on` JSON column | `entity_relationships` | `depends_on` |
| Phase 3 | `bugs.linked_entity_type`/`linked_entity_key` | `entity_relationships` | `linked_to` |
| Phase 4 | `change_cards.epic_id`/`feature_id`/`related_task_id` | `entity_relationships` | `related_to` |
| Phase 5 | `epic_relationships` + `feature_relationships` | `entity_relationships` | Preserves original type |

Each phase checks `tableExistsInDB()` before running (skips if source table doesn't exist).

The migration also runs automatically during `runMigrations()` in `internal/db/db.go` (line 761-763).

### Deduplication

All INSERT statements use `INSERT OR IGNORE` combined with the UNIQUE constraint `(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)` to prevent duplicate rows.

---

## Scenario 5: CLI Commands

### New Polymorphic Commands

File: `internal/cli/commands/link.go` (294 lines)

Three top-level commands registered on `cli.RootCmd`:

```go
func init() {
	linkCmd.Flags().StringVar(&linkRelType, "type", "", "Relationship type (required)")
	linkCmd.MarkFlagRequired("type")
	unlinkCmd.Flags().StringVar(&linkRelType, "type", "", "Relationship type (required)")
	unlinkCmd.MarkFlagRequired("type")
	cli.RootCmd.AddCommand(linkCmd)
	cli.RootCmd.AddCommand(unlinkCmd)
	cli.RootCmd.AddCommand(linksCmd)
}
```

| Command | Use | Args | Description |
|---|---|---|---|
| `shark link <from-key> <to-key> --type=TYPE` | `linkCmd` | 2 | Create relationship via `EntityRelationshipService.CreateRelationship()` |
| `shark unlink <from-key> <to-key> --type=TYPE` | `unlinkCmd` | 2 | Remove relationship via `EntityRelationshipService.UnlinkEntities()` |
| `shark links <key>` | `linksCmd` | 1 | List all relationships via `EntityRelationshipService.GetRelationships()` |

Entity type auto-detection via `resolveEntityKeyToTypeAndID()` which uses `DetectEntityType()` and `EntityRegistry`.

Supported entity types in `mapDetectedTypeToEntityType()`: epic, feature, task, bug, change/change_card.

`runLinks` enriches output with entity keys by calling `registry.GetRepository(otherType).GetByID()`.

### Legacy Commands (Backward Compatible)

File: `internal/cli/commands/task_link.go` (157 lines)

```go
// Deprecated: Use 'shark link' for cross-entity relationships via the
// entity_relationships table.
var taskLinkCmd = &cobra.Command{
	Use:   "link <task-key>",
	Short: "Create typed relationships between tasks (deprecated: use 'shark link')",
	...
}
```

Registered as `taskCmd.AddCommand(taskLinkCmd)`. Still uses legacy `cli.GetTaskServiceWithDeps().CreateTypedRelationship()` which writes to `task_relationships` table.

File: `internal/cli/commands/task_unlink.go` (165 lines)

```go
// Deprecated: Use 'shark unlink' for cross-entity relationships via the
// entity_relationships table.
var taskUnlinkCmd = &cobra.Command{
	Use:   "unlink <task-key>",
	Short: "Remove typed relationships between tasks (deprecated: use 'shark unlink')",
	...
}
```

Both legacy commands include deprecation notices in Short, Long, and example text directing users to `shark link`/`shark unlink`.

### CLI Test File

File: `internal/cli/commands/link_test.go` (113 lines)

| Test Function | What It Tests |
|---|---|
| `TestMapDetectedTypeToEntityType` | Table-driven: epic, feature, task, bug, change, change_card, unknown, empty, idea |
| `TestMapDetectedTypeToEntityType_AllValidEntityTypes` | Smoke test that all 5 entity types map correctly |
| `TestLinkCommandRelationshipTypeValidation` | Verifies valid types accepted, invalid types rejected |

### CLI Test Output (raw)

```
=== RUN   TestLinkCommandRelationshipTypeValidation
--- PASS: TestLinkCommandRelationshipTypeValidation (0.00s)
=== RUN   TestMapDetectedTypeToEntityType
=== RUN   TestMapDetectedTypeToEntityType/epic
=== RUN   TestMapDetectedTypeToEntityType/feature
=== RUN   TestMapDetectedTypeToEntityType/task
=== RUN   TestMapDetectedTypeToEntityType/bug
=== RUN   TestMapDetectedTypeToEntityType/change
=== RUN   TestMapDetectedTypeToEntityType/change_card
=== RUN   TestMapDetectedTypeToEntityType/unknown_type_returns_error
=== RUN   TestMapDetectedTypeToEntityType/empty_string_returns_error
=== RUN   TestMapDetectedTypeToEntityType/idea_returns_error_(not_a_relationship_entity)
--- PASS: TestMapDetectedTypeToEntityType (0.00s)
=== RUN   TestMapDetectedTypeToEntityType_AllValidEntityTypes
--- PASS: TestMapDetectedTypeToEntityType_AllValidEntityTypes (0.00s)
PASS
```

---

## Scenario 6: Legacy Code Removal

### TaskDependencyService Still Exists

File: `internal/services/task_dependency_service.go` (544 lines)

Contains LEGACY annotations on the struct and all public methods:

```go
// LEGACY: TaskDependencyService operates on the legacy task_relationships table.
// New code should use EntityRelationshipService which operates on the polymorphic
// entity_relationships table and supports cross-entity-type linking.
// This service will be removed once all callers are migrated to EntityRelationshipService.
type TaskDependencyService struct { ... }
```

Methods with LEGACY markers: `AddDependency`, `RemoveDependency`, `ListDependencies`, `UnlinkFile`, `UnlinkRelationships`, `CreateTypedRelationship`, `GetTaskRelationships`, `GetTaskBlockedBy`, `GetTaskBlocks`.

### Old Relationship Repository Files Still Exist

Files present in `internal/repository/`:
- `task_relationship_repository.go`
- `epic_relationship_repository.go`
- `feature_relationship_repository.go`
- `relationship_repositories_test.go`
- `task_relationship_repository_test.go`

### Old Relationship Model Files Still Exist

Files present in `internal/models/`:

**`task_relationship.go`** (63 lines):
```go
// LEGACY: TaskRelationship uses the legacy task_relationships table.
// New code should use EntityRelationship (entity_relationships table) which
// supports polymorphic cross-entity-type linking. This model will be removed
// once all callers are migrated to EntityRelationship.
type TaskRelationship struct { ... }
```

**`epic_relationship.go`** (37 lines):
```go
// LEGACY: EpicRelationship uses the legacy epic_relationships table.
// New code should use EntityRelationship (entity_relationships table) which
// supports polymorphic cross-entity-type linking. This model will be removed
// once all callers are migrated to EntityRelationship.
type EpicRelationship struct { ... }
```

**`feature_relationship.go`** (37 lines):
```go
// LEGACY: FeatureRelationship uses the legacy feature_relationships table.
// New code should use EntityRelationship (entity_relationships table) which
// supports polymorphic cross-entity-type linking. This model will be removed
// once all callers are migrated to EntityRelationship.
type FeatureRelationship struct { ... }
```

### Bug Model Legacy Fields

File: `internal/models/bug.go`, lines 37-44:
```go
// LEGACY: LinkedEntityType is a legacy field for direct entity linking.
// Migrate to entity_relationships table via EntityRelationshipService.
// This field will be removed once all callers are migrated.
LinkedEntityType *string `json:"linked_entity_type,omitempty" db:"linked_entity_type"`
// LEGACY: LinkedEntityKey is a legacy field for direct entity linking.
// Migrate to entity_relationships table via EntityRelationshipService.
// This field will be removed once all callers are migrated.
LinkedEntityKey *string `json:"linked_entity_key,omitempty" db:"linked_entity_key"`
```

### ChangeCard Model Legacy Fields

File: `internal/models/change_card.go`, lines 37-48:
```go
// LEGACY: EpicID is a legacy field for direct entity linking.
// Migrate to entity_relationships table via EntityRelationshipService.
// This field will be removed once all callers are migrated.
EpicID *int64 `json:"epic_id,omitempty" db:"epic_id"`
// LEGACY: FeatureID is a legacy field for direct entity linking.
// Migrate to entity_relationships table via EntityRelationshipService.
// This field will be removed once all callers are migrated.
FeatureID *int64 `json:"feature_id,omitempty" db:"feature_id"`
// LEGACY: RelatedTaskID is a legacy field for direct entity linking.
// Migrate to entity_relationships table via EntityRelationshipService.
// This field will be removed once all callers are migrated.
RelatedTaskID *int64 `json:"related_task_id,omitempty" db:"related_task_id"`
```

---

## Entity Relationship Model

File: `internal/models/entity_relationship.go` (107 lines)

### Type Definitions

```go
type EntityRelationshipType string

const (
	EntityRelDependsOn   EntityRelationshipType = "depends_on"
	EntityRelBlocks      EntityRelationshipType = "blocks"
	EntityRelRelatedTo   EntityRelationshipType = "related_to"
	EntityRelFollows     EntityRelationshipType = "follows"
	EntityRelSpawnedFrom EntityRelationshipType = "spawned_from"
	EntityRelDuplicates  EntityRelationshipType = "duplicates"
	EntityRelReferences  EntityRelationshipType = "references"
	EntityRelLinkedTo    EntityRelationshipType = "linked_to"
)
```

Backward-compatible untyped constants also defined (lines 25-34).

### Struct

```go
type EntityRelationship struct {
	ID               int64                  `json:"id" db:"id"`
	FromEntityType   EntityType             `json:"from_entity_type" db:"from_entity_type"`
	FromEntityID     int64                  `json:"from_entity_id" db:"from_entity_id"`
	ToEntityType     EntityType             `json:"to_entity_type" db:"to_entity_type"`
	ToEntityID       int64                  `json:"to_entity_id" db:"to_entity_id"`
	RelationshipType EntityRelationshipType `json:"relationship_type" db:"relationship_type"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
}
```

### Validate Method

```go
func (er *EntityRelationship) Validate() error {
	if er.FromEntityID == 0 { return fmt.Errorf("from_entity_id must not be zero") }
	if er.ToEntityID == 0 { return fmt.Errorf("to_entity_id must not be zero") }
	if !ValidEntityTypes[er.FromEntityType] { ... }
	if !ValidEntityTypes[er.ToEntityType] { ... }
	if !ValidEntityRelationshipTypeSet[er.RelationshipType] { ... }
	if er.FromEntityType == er.ToEntityType && er.FromEntityID == er.ToEntityID {
		return fmt.Errorf("entity cannot have a relationship with itself")
	}
	return nil
}
```

### IsCyclic Method

```go
func (er *EntityRelationship) IsCyclic() bool {
	return CyclicRelationshipTypes[er.RelationshipType] &&
		er.FromEntityType == er.ToEntityType
}
```

---

## Use Case Traces

### CLI Commands Calling EntityRelationshipService

| CLI Command | File | Service Method Called |
|---|---|---|
| `shark link` | `internal/cli/commands/link.go:147` | `svc.CreateRelationship()` |
| `shark unlink` | `internal/cli/commands/link.go:184` | `svc.UnlinkEntities()` |
| `shark links` | `internal/cli/commands/link.go:221` | `svc.GetRelationships()` |

### EntityRelationshipService Wiring

File: `internal/cli/services_global.go`, lines 443-453:
```go
func GetEntityRelationshipService() *services.EntityRelationshipService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for EntityRelationshipService: %v", err))
	}
	repo := repository.NewEntityRelationshipRepository(db)
	return services.NewEntityRelationshipService(repo)
}
```

Pattern: Creates new instance each call. Depends on `repository.EntityRelationshipRepository` (concrete) which wraps `*repository.DB`.

### EntityRelationshipService Interface

File: `internal/services/entity_relationship_service.go`, lines 13-55:
```go
type EntityRelationshipRepository interface {
	Create(ctx context.Context, rel *models.EntityRelationship) error
	Delete(ctx context.Context, id int64) error
	DeleteByEntitiesAndType(ctx context.Context, fromType models.EntityType, fromID int64, toType models.EntityType, toID int64, relType models.EntityRelationshipType) error
	GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityRelationship, error)
	GetOutgoing(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error)
	GetIncoming(ctx context.Context, entityType models.EntityType, entityID int64, relTypes []models.EntityRelationshipType) ([]*models.EntityRelationship, error)
}
```

### EntityRelationshipService Public Methods

```go
func NewEntityRelationshipService(repo EntityRelationshipRepository) *EntityRelationshipService
func (s *EntityRelationshipService) CreateRelationship(ctx, fromType, fromID, toType, toID, relType) (*models.EntityRelationship, error)
func (s *EntityRelationshipService) DeleteRelationship(ctx, id int64) error
func (s *EntityRelationshipService) UnlinkEntities(ctx, fromType, fromID, toType, toID, relType) error
func (s *EntityRelationshipService) GetRelationships(ctx, entityType, entityID) ([]*models.EntityRelationship, error)
func (s *EntityRelationshipService) GetOutgoing(ctx, entityType, entityID, relTypes) ([]*models.EntityRelationship, error)
func (s *EntityRelationshipService) GetIncoming(ctx, entityType, entityID, relTypes) ([]*models.EntityRelationship, error)
func (s *EntityRelationshipService) DetectCycle(ctx, fromType, fromID, toType, toID, relType) (bool, error)
```

---

## Build Health

### make lint

```
0 issues.
```

### make test

```
FAIL	github.com/jwwelbor/shark-task-manager/internal/templates	0.056s
```

The only failing package is `internal/templates` which fails on epic template tests (unrelated to E21-F11: `TestEpicTemplates_ExistAndRender`, `TestEpicTemplates_AllExist`, etc.). All E21-F11 related packages pass:

- `internal/repository` -- PASS (EntityRelationship tests: 20/20)
- `internal/services` -- PASS (EntityRelationship tests: 20/20)
- `internal/cli/commands` -- PASS (Link tests: 11/11)
