# Technical Specification: Tech-Debt Entity Implementation (E25-F01)

**Feature**: E25-F01 Tech-Debt Entity Implementation
**Epic**: E25 Add Tech-Debt Entity Type
**Date**: 2026-04-05
**Status**: Draft

---

## 1. Requirements

### 1.1 Functional Requirements

#### FR-01: CRUD Operations (SC-1)

**Create**:
- `shark td create "<title>" [--category=<cat>] [--severity=<sev>] [--effort-estimate=<est>] [--description="..."]` creates a tech-debt entity with auto-generated key TD-### (next available 3-digit number).
- Default category: `code-quality`. Default severity: `medium`. Default status: `identified` (workflow default).
- On success, CLI prints the entity key, title, status, and file path of the created markdown file.
- `--json` flag returns the full entity as JSON.

**Read**:
- `shark td get <key>` displays entity details: key, title, status, category, severity, effort_estimate, description, file_path, created_at, updated_at.
- `shark td get <key> --json` returns JSON representation.
- `shark td get <key> --field <name>` extracts a single field value.

**Update**:
- `shark td update <key> [--title=...] [--category=...] [--severity=...] [--effort-estimate=...] [--description=...]` updates specified fields.
- Does NOT accept `--status` flag. Status changes go through `shark status set` or `shark status advance`.

**Delete**:
- `shark td delete <key>` removes the entity from the database.
- `shark td delete <key> --force` skips confirmation prompt.

**List**:
- `shark td list [--category=<cat>] [--severity=<sev>] [--status=<status>] [--json]` lists tech-debt entities with optional filters.
- Default sort: by key ascending.

#### FR-02: Triage Command (SC-1)

- `shark td triage <key> [--severity=<sev>] [--category=<cat>] [--effort-estimate=<est>]` sets triage fields and advances status to `triaged` (if currently `identified`).
- If already past `identified`, updates fields without changing status.

#### FR-03: Workflow Integration (SC-2)

- Tech-debt entities participate in the configured workflow profile.
- Default workflow statuses: `identified`, `triaged`, `in_progress`, `resolved`, `wont_fix`, `cancelled`.
- Default flow: `identified -> triaged -> in_progress -> resolved`; also `triaged -> wont_fix`, `triaged -> cancelled`, `in_progress -> cancelled`.
- `shark status advance TD-001` advances to next valid status.
- `shark status set TD-001 <status>` sets status directly (with transition validation).
- `shark status options TD-001` shows valid next statuses from current state.
- `shark status history TD-001` shows status change history.
- Invalid transitions are rejected with an error message listing valid options.

#### FR-04: Notes and Context (SC-3)

- `shark td note add <key> --content="..." [--type=<type>]` adds a note. Valid types: comment, decision, blocker, solution, reference, implementation, testing, future, question, rejection, requirement.
- `shark td notes <key>` lists all notes for the entity.
- `shark td context set <key> --field <name> --value <val>` sets a context field.
- `shark td context get <key>` displays all context fields.
- `shark td context clear <key> --field <name>` clears a context field.

#### FR-05: Core Command Auto-Detection (SC-4)

TD-### keys are recognized by all core dispatch commands:
- `shark get TD-001` routes to tech-debt get.
- `shark delete TD-001` routes to tech-debt delete.
- `shark status TD-001` routes to tech-debt status display.
- `shark status advance TD-001` routes to tech-debt status advance.
- `shark status set TD-001 <status>` routes to tech-debt status set.
- `shark list` (with no args) does NOT list tech-debt (standalone entity, not in epic hierarchy). Use `shark td list`.
- `shark update TD-001` routes to tech-debt update.
- `shark history TD-001` routes to tech-debt history.
- `shark notes TD-001` routes to tech-debt notes.
- `shark view TD-001` routes to tech-debt file view.

#### FR-06: Search Integration (SC-5)

- `shark search "<query>"` includes tech-debt entities in results.
- Search matches against title, key, and description fields.
- Results display entity_type as "tech_debt" with the category shown in the severity/tag column.

#### FR-07: Analytics Integration (SC-5)

- `shark analytics` includes tech-debt summary (total count, count by status, count by category).
- Tech-debt entities appear in project-level dashboard statistics.

#### FR-08: JSON Output (SC-8)

- All `shark td` subcommands support `--json` flag for machine-readable output.
- `--field <name>` extracts a single field from JSON output (implies `--json`).
- JSON structure follows the same conventions as Bug and Change-Card entities.

#### FR-09: Database Migration (SC-7)

- `tech_debts` table is created via additive migration (CREATE TABLE IF NOT EXISTS).
- Migration bumps `CurrentSchemaVersion` from 10 to 11.
- Migration is idempotent; safe to run multiple times.
- Existing data in all other tables is untouched.

### 1.2 Non-Functional Requirements

#### NFR-01: Performance

- All CRUD operations complete in < 100ms for databases with up to 999 tech-debt entities.
- List with filters uses indexed columns (status, category, severity) for efficient queries.
- No degradation to existing entity operations (epics, features, tasks, bugs, change-cards).

#### NFR-02: Backward Compatibility

- All existing CLI commands, workflows, and database operations continue to work unchanged.
- All existing tests pass without modification.
- The `tech_debts` table simply does not exist in older schema versions; no ALTER TABLE on existing tables.

#### NFR-03: Migration Safety

- Migration is purely additive (CREATE TABLE, CREATE INDEX, CREATE TRIGGER).
- No modification to existing tables, indexes, or triggers.
- Developer must set `skip_migrations: false` in `.sharkconfig.json` before first shark command post-merge, then reset to `true`.

#### NFR-04: Code Quality

- All new code passes `make fmt && make lint && make test`.
- Follows established patterns from Bug/Change-Card reference implementations.
- No new linting warnings introduced.

---

## 2. Architecture

### 2.1 New Files

#### `internal/models/tech_debt.go`
Model definition for the TechDebt entity.

**Contains**:
- `TechDebtStatus` type (string alias, values determined by workflow config)
- `TechDebtCategory` type with constants: `code-quality`, `architecture`, `dependency`, `testing`, `performance`, `documentation`
- `ValidTechDebtCategories` map for allowlist validation
- `TechDebtSeverity` type with constants: `critical`, `high`, `medium`, `low`
- `ValidTechDebtSeverities` map for allowlist validation
- `TechDebt` struct embedding `BaseEntity` with fields: `Status`, `Category`, `Severity`, `EffortEstimate *string`
- Entity interface methods: `GetEntityType() -> EntityTypeTechDebt`, `GetStatus()`, `SetStatus()`
- `Validate()` method: validates key format, non-empty title, non-empty status, valid category, valid severity
- `techDebtKeyPattern` regex: `^TD-\d{3}$`
- `ValidateTechDebtKey()` function
- `ErrInvalidTechDebtKey` sentinel error

**Pattern**: Copy `internal/models/bug.go` and substitute Bug -> TechDebt, add Category field, add EffortEstimate field, remove LinkedEntity fields.

#### `internal/repository/tech_debt/repository.go`
Full CRUD repository for tech-debt entities.

**Contains**:
- `Repository` struct with `*repository.DB` dependency
- `NewTechDebtRepository(db *repository.DB) *Repository` constructor
- `Create(ctx, *models.TechDebt) error` -- INSERT with auto-key generation (SELECT MAX key + 1)
- `GetByKey(ctx, key string) (*models.TechDebt, error)` -- case-insensitive lookup
- `GetByID(ctx, id int64) (*models.TechDebt, error)`
- `Update(ctx, *models.TechDebt) error` -- UPDATE title, description, category, severity, effort_estimate
- `UpdateStatus(ctx, id int64, status models.TechDebtStatus, notes *string) error`
- `Delete(ctx, key string) error`
- `List(ctx) ([]*models.TechDebt, error)` -- all tech-debt items
- `ListWithFilters(ctx, filters TechDebtFilters) ([]*models.TechDebt, error)` -- filtered list
- `CountByStatus(ctx) (map[string]int, error)` -- for analytics
- `CountByCategory(ctx) (map[string]int, error)` -- for analytics
- `GenerateNextKey(ctx) (string, error)` -- SELECT MAX + format TD-###
- Scan helper: `scanTechDebt(row) (*models.TechDebt, error)`

**Pattern**: Copy `internal/repository/bug/repository.go` and substitute Bug -> TechDebt, add category/effort_estimate columns, remove linked_entity columns.

#### `internal/services/tech_debt_service.go`
Service layer with workflow-aware business logic.

**Contains**:
- `TechDebtRepository` interface (defined at point of use, not in repository package)
- `TechDebtService` struct with dependencies: `repo TechDebtRepository`, `workflowSvc *workflow.Service`, `noteRepo EntityNoteRepository` (optional)
- `NewTechDebtService(repo, workflowSvc, noteRepo) *TechDebtService` constructor (calls `workflowSvc.ForLevel(workflow.LevelTechDebt)`)
- `Create(ctx, CreateTechDebtInput) (*models.TechDebt, error)` -- validate, generate key, create in DB
- `Get(ctx, key string) (*models.TechDebt, error)` -- retrieve by key
- `Update(ctx, key string, updates TechDebtUpdates) (*models.TechDebt, error)` -- partial update
- `Delete(ctx, key string) error`
- `List(ctx, filters TechDebtFilters) ([]*models.TechDebt, error)`
- `AdvanceStatus(ctx, key string) (*models.TechDebt, error)` -- workflow next-status
- `SetStatus(ctx, key string, status string, reason string, force bool) (*models.TechDebt, error)` -- direct status set with transition validation
- `Triage(ctx, key string, input TriageTechDebtInput) (*models.TechDebt, error)` -- set triage fields + advance to triaged
- `GetStatusOptions(ctx, key string) ([]string, error)` -- valid next statuses
- `GetStatusHistory(ctx, key string) ([]StatusHistoryEntry, error)` -- status change log

**Pattern**: Copy `internal/services/bug_service.go` and substitute Bug -> TechDebt, add category/effort_estimate handling, remove linked_entity handling.

#### `internal/services/tech_debt_dto.go`
Data transfer objects for service inputs.

**Contains**:
- `CreateTechDebtInput` struct: Title, Description, Category, Severity, EffortEstimate, FilePath, CreateFile
- `TechDebtUpdates` struct: Title, Description, Category, Severity, EffortEstimate (all pointer types for partial update)
- `TechDebtFilters` struct: Status, Category, Severity (all string, empty = no filter)
- `TriageTechDebtInput` struct: Severity, Category, EffortEstimate

**Pattern**: Copy `internal/services/bug_dto.go`.

#### `internal/services/tech_debt_repo_adapter.go`
Adapter implementing the generic `EntityRepository` interface for entity registry.

**Contains**:
- `TechDebtRepositoryAdapter` struct wrapping the concrete repository
- `NewTechDebtRepositoryAdapter(repo *tech_debt.Repository) *TechDebtRepositoryAdapter`
- Implements: `GetByKey(ctx, key) (models.Entity, error)`, `UpdateStatus(ctx, id int64, status string, agent *string, notes *string) error`

**Pattern**: Copy `internal/services/bug_repo_adapter.go`.

#### `internal/cli/commands/tech_debt.go`
CLI command group `shark td` with all subcommands.

**Contains**:
- `tdCmd` parent command registered with `GroupID: "advanced"`
- `techDebtServicer` interface for test injection
- `tdSvcOverride` var for test mock injection
- `getTechDebtService()` helper (returns override or `cli.GetTechDebtService()`)
- Subcommands:
  - `td create "<title>" [flags]` -- parse args, call service Create, print key + file path
  - `td get <key> [--json] [--field]` -- parse key, call service Get, format output
  - `td list [--category] [--severity] [--status] [--json]` -- parse filters, call service List, format table/JSON
  - `td update <key> [flags]` -- parse updates, call service Update
  - `td delete <key> [--force]` -- confirm, call service Delete
  - `td triage <key> [flags]` -- parse triage fields, call service Triage
  - `td note add <key> --content="..." [--type=...]` -- delegate to generic note add
  - `td notes <key>` -- delegate to generic notes list
  - `td context set/get/clear <key>` -- delegate to generic context commands
- All subcommands support `--json` and `--field` flags
- `init()` registers `tdCmd` under root with `GroupID: "advanced"`

**Pattern**: Copy `internal/cli/commands/bug.go` and substitute bug -> tech_debt/td, add category/effort_estimate flags, remove linked_entity flags.

#### `shark-templates/tech_debt/*.tmpl` (6 files)
Per-status markdown templates for tech-debt entity files.

**Files**:
- `shark-templates/tech_debt/identified.tmpl`
- `shark-templates/tech_debt/triaged.tmpl`
- `shark-templates/tech_debt/in_progress.tmpl`
- `shark-templates/tech_debt/resolved.tmpl`
- `shark-templates/tech_debt/wont_fix.tmpl`
- `shark-templates/tech_debt/cancelled.tmpl`

Each template includes YAML frontmatter with: key, title, status, category, severity, effort_estimate, description, created_at. Body section with markdown headers for description, impact, resolution plan.

**Pattern**: Copy `shark-templates/bug/*.tmpl` and adjust status names and fields.

### 2.2 Modified Files

#### `internal/models/entity_note.go`
- **Add** `EntityTypeTechDebt EntityType = "tech_debt"` constant alongside existing entity type constants
- **Add** `EntityTypeTechDebt: true` entry in `ValidEntityTypes` map
- **Complexity**: XS (2 lines)

#### `internal/models/entity.go`
- **Add** `_ Entity = (*TechDebt)(nil)` compile-time interface check alongside Bug/ChangeCard checks
- **Complexity**: XS (1 line)

#### `internal/keys/validation.go`
- **Add** `techDebtKeyPattern = regexp.MustCompile("^TD-\\d{3}$")` regex
- **Add** `IsTechDebtKey(key string) bool` function
- **Complexity**: XS (10 lines)

#### `internal/workflow/levels.go`
- **Add** `LevelTechDebt = "tech_debt"` constant
- **Complexity**: XS (1 line)

#### `internal/config/workflow/multilevel.go`
- **Add** `TechDebt *WorkflowConfig` field to `MultiLevelWorkflow` struct
- **Add** `case "tech_debt":` branch in `GetWorkflowForLevel()` returning `m.TechDebt`
- **Complexity**: XS (6 lines)

#### `internal/config/workflow/defaults.go`
- **Add** `DefaultTechDebtWorkflow()` function returning `*WorkflowConfig` with:
  - Statuses: identified, triaged, in_progress, resolved, wont_fix, cancelled
  - Status metadata (colors, phases, progress weights, responsibilities)
  - Status flow: identified->triaged->in_progress->resolved; triaged->wont_fix; triaged->cancelled; in_progress->cancelled
  - Default status: identified
  - Terminal statuses: resolved, wont_fix, cancelled
- **Complexity**: S (30 lines)

#### `internal/config/workflow/parser.go`
- **Add** `"tech_debt_workflow": "tech_debt"` to `workflowKeyToLevel` map
- **Add** corresponding field pointer in `entityKeys` map
- **Complexity**: XS (2 lines)

#### `internal/config/workflow/validator.go`
- **Add** `"tech_debt": multi.TechDebt` to entity levels validation map
- **Complexity**: XS (1 line)

#### `internal/config/template/helpers.go`
- **Add** `TechDebtPlaceholders()` function returning placeholder map with: category, severity, effort_estimate
- **Complexity**: S (20 lines)

#### `internal/db/db.go`
- **Bump** `CurrentSchemaVersion` from 10 to 11
- **Add** `migrateTechDebtTable()` function: CREATE TABLE IF NOT EXISTS tech_debts, CREATE indexes, CREATE updated_at trigger
- **Add** call to `migrateTechDebtTable()` in `runMigrations()`
- **Complexity**: M (60 lines)

#### `internal/cli/services_global.go`
- **Add** `GetTechDebtService() *services.TechDebtService` accessor function (pattern: copy GetBugService)
- **Add** tech-debt registration in `GetEntityRegistry()`: `registry.Register(models.EntityTypeTechDebt, adapter)`
- **Add** tech-debt repo to `GetDashboardAnalyticsService()` constructor call
- **Complexity**: S (40 lines)

#### `internal/cli/commands/helpers.go`
- **Add** `IsTechDebtKey(key string) bool` wrapper calling `keys.IsTechDebtKey()`
- **Add** `case "tech_debt":` in `DetectEntityType()` -- must be placed BEFORE task key checks (TD- starts with T)
- **Add** `scopeTechDebt` scope type and detection case in `ParseGetArgs()`
- **Complexity**: S (20 lines)

#### CLI Dispatch Points (each XS, 2-4 lines per file)

| File | Function | Change |
|------|----------|--------|
| `internal/cli/commands/get.go` | `runGet()` | Add `case "tech_debt":` calling tech-debt service get |
| `internal/cli/commands/delete_dispatch.go` | `dispatchDelete()` | Add `case "tech_debt":` calling tech-debt service delete |
| `internal/cli/commands/status_group.go` | `resolveTransitioner()` | Add `case "tech_debt":` returning tech-debt transitioner |
| `internal/cli/commands/status_group.go` | `dispatchNextStatus()` | Add `case "tech_debt":` for next-status dispatch |
| `internal/cli/commands/list.go` | list dispatcher | Add `case "tech_debt":` calling tech-debt service list |
| `internal/cli/commands/update_dispatch.go` | update dispatcher | Add `case "tech_debt":` calling tech-debt service update |
| `internal/cli/commands/note_generic.go` | `entityTypeFromName()` | Add `case "tech_debt":` returning `EntityTypeTechDebt` |
| `internal/cli/commands/context.go` | entity type resolver | Add `case "tech_debt":` returning `EntityTypeTechDebt` |
| `internal/cli/commands/link.go` | entity type resolver | Add `case "tech_debt":` |
| `internal/cli/commands/errors.go` | error suggestions | Add tech_debt suggestions |
| `internal/cli/commands/render_common.go` | render dispatcher | Add `case "tech_debt":` |
| `internal/cli/commands/analytics.go` | analytics dispatcher | Add `case "tech_debt":` |
| `internal/cli/commands/workflow_show_action.go` | workflow action dispatcher | Add `case "tech_debt":` |
| `internal/cli/commands/run.go` | run dispatcher (2 locations) | Add `case "tech_debt":` |

#### `internal/repository/search/repository.go`
- **Add** UNION ALL block for tech_debts table in `SearchAll()` query
- **Add** 3 additional `pattern` args to the args slice
- **Complexity**: S (10 lines)

#### `internal/services/dashboard_analytics_service.go`
- **Add** `TechDebtSummaryRepository` interface with `CountByStatus()` and `CountByCategory()` methods
- **Add** `techDebtRepo` field to service struct
- **Add** `techDebtRepo` parameter to `NewDashboardAnalyticsService()` constructor
- **Add** `GetTechDebtAnalytics()` method
- **Complexity**: S (30 lines)

### 2.3 Data Model

#### `tech_debts` Table Schema

```sql
CREATE TABLE IF NOT EXISTS tech_debts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    key             TEXT NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    slug            TEXT,
    description     TEXT,
    status          TEXT NOT NULL DEFAULT 'identified',
    category        TEXT NOT NULL CHECK (category IN (
        'code-quality', 'architecture', 'dependency',
        'testing', 'performance', 'documentation'
    )) DEFAULT 'code-quality',
    severity        TEXT NOT NULL CHECK (severity IN (
        'critical', 'high', 'medium', 'low'
    )) DEFAULT 'medium',
    effort_estimate TEXT,
    context_data    TEXT,
    file_path       TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### Indexes

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_tech_debts_key ON tech_debts(key);
CREATE INDEX IF NOT EXISTS idx_tech_debts_status ON tech_debts(status);
CREATE INDEX IF NOT EXISTS idx_tech_debts_severity ON tech_debts(severity);
CREATE INDEX IF NOT EXISTS idx_tech_debts_category ON tech_debts(category);
CREATE INDEX IF NOT EXISTS idx_tech_debts_slug ON tech_debts(slug);
```

#### Trigger

```sql
CREATE TRIGGER IF NOT EXISTS tech_debts_updated_at
AFTER UPDATE ON tech_debts
FOR EACH ROW
BEGIN
    UPDATE tech_debts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### 2.4 Key Format

- **Pattern**: `TD-###` (TD followed by hyphen and exactly 3 digits)
- **Regex**: `^TD-\d{3}$`
- **Examples**: TD-001, TD-042, TD-999
- **Auto-generation**: SELECT MAX(CAST(SUBSTR(key, 4) AS INTEGER)) FROM tech_debts, increment by 1, zero-pad to 3 digits
- **Capacity**: 001-999 (999 items per project)
- **Case-insensitive**: `td-001` and `TD-001` both resolve to the same entity

### 2.5 Domain Fields

#### Category (TEXT NOT NULL, DEFAULT 'code-quality')

| Value | Description |
|-------|-------------|
| `code-quality` | Code smells, duplication, poor naming, lack of comments |
| `architecture` | Architectural shortcuts, tight coupling, missing abstractions |
| `dependency` | Outdated dependencies, version conflicts, deprecated libraries |
| `testing` | Missing test coverage, flaky tests, inadequate test infrastructure |
| `performance` | N+1 queries, missing indexes, unoptimized algorithms |
| `documentation` | Missing docs, outdated docs, undocumented APIs |

Validated via `map[TechDebtCategory]bool` allowlist in model layer.

#### Severity (TEXT NOT NULL, DEFAULT 'medium')

| Value | Description |
|-------|-------------|
| `critical` | Actively causing issues, must be resolved immediately |
| `high` | Significant risk, should be resolved soon |
| `medium` | Moderate risk, plan for resolution |
| `low` | Minor issue, resolve when convenient |

Validated via `map[TechDebtSeverity]bool` allowlist in model layer. Same scale as Bug severity for consistency.

#### Effort Estimate (TEXT, nullable)

Free-text field for effort estimation. No validation beyond max length (100 characters). Teams choose their own estimation scale:
- T-shirt sizes: `xs`, `s`, `m`, `l`, `xl`
- Story points: `1`, `2`, `3`, `5`, `8`, `13`
- Time: `2 hours`, `1 day`, `1 sprint`

### 2.6 Workflow Configuration

Default tech-debt workflow registered as `DefaultTechDebtWorkflow()`:

```
identified -> triaged -> in_progress -> resolved
                      -> wont_fix
                      -> cancelled
              in_progress -> cancelled
```

**Status metadata**:

| Status | Color | Phase | Progress Weight | Responsibility |
|--------|-------|-------|-----------------|----------------|
| identified | gray | planning | 0 | none |
| triaged | blue | planning | 10 | none |
| in_progress | yellow | development | 50 | agent |
| resolved | green | done | 100 | none |
| wont_fix | orange | done | 100 | none |
| cancelled | red | done | 100 | none |

---

## 3. Test Plan

### 3.1 Repository Tests (`internal/repository/tech_debt/repository_test.go`)

Uses real database with cleanup (per project testing architecture).

| Test | Description |
|------|-------------|
| TestCreate | Create tech-debt, verify ID set, fields persisted correctly |
| TestCreate_DuplicateKey | Verify UNIQUE constraint rejects duplicate keys |
| TestGetByKey | Retrieve by key, verify all fields |
| TestGetByKey_CaseInsensitive | `td-001` resolves same as `TD-001` |
| TestGetByKey_NotFound | Returns appropriate error for non-existent key |
| TestUpdate | Update title, category, severity, effort_estimate; verify changes persisted |
| TestUpdateStatus | Update status field independently |
| TestDelete | Delete entity, verify not retrievable |
| TestList | List all entities, verify count and ordering |
| TestListWithFilters_Category | Filter by category, verify only matching returned |
| TestListWithFilters_Severity | Filter by severity |
| TestListWithFilters_Status | Filter by status |
| TestListWithFilters_Combined | Multiple filters applied simultaneously |
| TestGenerateNextKey | First key is TD-001; after TD-005 exists, next is TD-006 |
| TestCountByStatus | Verify count aggregation matches actual data |
| TestCountByCategory | Verify count aggregation by category |
| TestDatabaseConstraints_CategoryCheck | Invalid category rejected by CHECK constraint |
| TestDatabaseConstraints_SeverityCheck | Invalid severity rejected by CHECK constraint |

### 3.2 Service Tests (`internal/services/tech_debt_service_test.go`)

Uses mocked repositories (per project testing architecture). No real database.

| Test | Description |
|------|-------------|
| TestCreate_Success | Mock repo Create, verify key generation and field defaults |
| TestCreate_ValidationError | Empty title returns validation error |
| TestCreate_InvalidCategory | Invalid category returns validation error |
| TestGet_Success | Mock repo GetByKey returns entity |
| TestGet_NotFound | Mock repo returns not found error, verify propagation |
| TestUpdate_Success | Mock repo Update, verify partial update applies only specified fields |
| TestDelete_Success | Mock repo Delete, verify key passed correctly |
| TestAdvanceStatus_Success | From identified -> triaged via workflow service |
| TestAdvanceStatus_TerminalState | From resolved -> error (no valid next status) |
| TestSetStatus_ValidTransition | Transition validates via workflow service |
| TestSetStatus_InvalidTransition | Invalid transition returns workflow error |
| TestSetStatus_Force | Force bypasses transition validation |
| TestTriage_FromIdentified | Advances to triaged + sets fields |
| TestTriage_AlreadyTriaged | Updates fields without status change |
| TestList_WithFilters | Verify filters passed to repository |
| TestGetStatusOptions | Returns valid next statuses from workflow |

### 3.3 CLI Tests (`internal/cli/commands/tech_debt_test.go`)

Uses mocked services (per project testing architecture). No real database, no real service.

| Test | Description |
|------|-------------|
| TestTdCreate_ArgsAndFlags | Verify argument parsing for title, category, severity flags |
| TestTdGet_JSONOutput | Verify JSON output format matches expectations |
| TestTdGet_FieldExtraction | `--field status` returns only status value |
| TestTdList_FilterFlags | Verify --category, --severity, --status flags parsed correctly |
| TestTdUpdate_PartialFlags | Only specified flags included in update DTO |
| TestTdDelete_ForceFlag | --force flag skips confirmation |
| TestTdTriage_ArgsAndFlags | Verify triage flag parsing |
| TestDetectEntityType_TDKey | `TD-001` detected as "tech_debt" |
| TestDetectEntityType_TDKey_CaseInsensitive | `td-001` detected as "tech_debt" |
| TestParseGetArgs_TDKey | `TD-001` parsed with scopeTechDebt |

---

## 4. Success Criteria Traceability

| SC | Requirement | Architecture Section | Test Coverage |
|----|-------------|---------------------|---------------|
| SC-1 | CRUD via `shark td` | FR-01, FR-02; New files: model, repo, service, CLI | Repository CRUD tests, Service CRUD tests, CLI arg tests |
| SC-2 | Workflow status transitions | FR-03; Modified: workflow levels, multilevel, defaults, parser, validator | Service advance/set tests |
| SC-3 | Notes and context | FR-04; Modified: entity_note.go (EntityTypeTechDebt) | Inherits from generic note/context infrastructure |
| SC-4 | Core command auto-detection | FR-05; Modified: helpers.go, 14 dispatch files | CLI DetectEntityType test |
| SC-5 | Search and analytics | FR-06, FR-07; Modified: search repo, analytics service | Search UNION test, analytics method test |
| SC-7 | Migration safety | FR-09; Modified: db.go | Idempotent CREATE TABLE IF NOT EXISTS |
| SC-8 | JSON output | FR-08; All CLI commands | CLI JSON output tests |

**Note**: SC-6 (file path display on entity create) is a separate concern affecting existing entity commands. It is included in the architecture section (2.2 Modified Files) but spans beyond this feature's tech-debt scope. Implementation tasks for SC-6 should be created separately.

---

## 5. Implementation Order

### Phase 1: Foundation (no DB changes, no breaking changes)
1. Model (`internal/models/tech_debt.go`) + entity type registration + interface check
2. Key validation (`internal/keys/validation.go`)
3. Workflow: level constant, defaults, multilevel, parser, validator
4. Template helpers + template files

### Phase 2: Data Layer (requires DB migration)
5. Database migration (`internal/db/db.go`) + schema version bump 10 -> 11
6. Repository (`internal/repository/tech_debt/repository.go`)
7. Service DTOs + repo adapter + service

### Phase 3: CLI Integration
8. Service accessor + entity registry (`internal/cli/services_global.go`)
9. Key detection + dispatch in helpers.go
10. All CLI dispatch points (14 files)
11. CLI command group (`internal/cli/commands/tech_debt.go`)

### Phase 4: Cross-Cutting
12. Search integration (UNION ALL in search repo)
13. Analytics integration (dashboard analytics service)

**Quality gate after each phase**: `make fmt && make lint && make test`

---

*Specification by: Architect agent | Date: 2026-04-05*
