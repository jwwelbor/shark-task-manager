# Architecture: Add Tech-Debt Entity Type (E25)

**Date**: 2026-04-05
**Status**: Accepted

---

## 1. Component Overview

### What Changes

The Tech-Debt entity type (TD-###) is added as a standalone entity following the exact pattern established by Bug (B###) and Change-Card (CC-###). No existing abstractions are modified -- all changes are additive registrations or new files.

**Summary**: 7 new files, 18 existing files modified (all additive).

### What Stays

- Entity interface (`Entity`), `BaseEntity` struct -- unchanged (only add compile-time check)
- Repository pattern, service pattern, CLI command pattern -- reused as-is
- Workflow system (`workflow.Service`, `MultiLevelWorkflow`) -- extended, not redesigned
- File operations (`fileops` package) -- reused without modification
- Database initialization flow (`InitDB`, `runMigrations`) -- extended with one new migration function

---

## 2. Key Technical Decisions

### ADR-1: Reuse Bug Entity Pattern Exactly

**Context**: Tech-Debt needs CRUD, workflow, notes, context, search, analytics, and core command auto-detection.

**Decision**: Copy the Bug entity implementation across all layers (model, repository, service, CLI) with substitutions for naming, key format, and domain-specific fields. Do not introduce new abstractions, generics, or shared base services.

**Rationale**: The Bug entity was designed as the template for standalone entities. Deviating from this pattern introduces inconsistency and increases maintenance cost. The research report confirms all 26 integration points follow the same dispatch pattern (switch/case on entity type string).

**Consequences**: Some code duplication across entity implementations. This is intentional -- each entity owns its full stack, enabling independent evolution. Shared behavior lives in `BaseEntity` and `EntityService`.

### ADR-2: Key Format TD-### with Regex `^TD-\d{3}$`

**Context**: Need a unique key prefix that does not collide with existing patterns (E##, E##-F##, E##-F##-###, B###, CC-###, C###, I-YYYY-MM-DD-##).

**Decision**: Use `TD-` prefix followed by exactly 3 digits. Regex: `^TD-\d{3}$`. Examples: TD-001, TD-042, TD-999.

**Rationale**: `TD-` is distinct from all existing prefixes. The hyphen after `TD` prevents collision with any future `T`-prefixed entity (task keys use `T-E`). Three digits match Bug (B###) and Change-Card (CC-###) conventions, supporting up to 999 items per project.

**Consequences**: Capacity limited to 999 tech-debt items. If exceeded, a key format expansion (4 digits) would require a separate change.

### ADR-3: Domain Fields -- category, severity, effort_estimate

**Context**: Tech-debt items need classification beyond what BaseEntity provides.

**Decision**: Add three domain-specific fields:
- `category` (TEXT NOT NULL, DEFAULT 'code-quality') -- classifies the type of debt. Valid values: code-quality, architecture, dependency, testing, performance, documentation.
- `severity` (TEXT NOT NULL, DEFAULT 'medium') -- reuses the same severity scale as Bug. Valid values: critical, high, medium, low.
- `effort_estimate` (TEXT, nullable) -- free-text effort estimate (e.g., "2 hours", "1 sprint", "S", "M").

**Rationale**: Category is the primary differentiator from bugs (which have severity but no category). Severity reuses the Bug pattern for consistency. Effort estimate is deliberately free-text to avoid premature standardization (teams use different estimation scales).

**Consequences**: Category validation uses a `map[string]bool` allowlist in the model layer (same pattern as `ValidBugSeverities`). Effort estimate has no validation beyond length limits.

### ADR-4: Default Workflow -- identified/triaged/in_progress/resolved/wont_fix/cancelled

**Context**: Tech-debt items need a lifecycle workflow.

**Decision**: Define a default tech-debt workflow with 6 statuses:
```
identified -> triaged -> in_progress -> resolved
                      -> wont_fix
                      -> cancelled
```

**Rationale**: Mirrors bug workflow structure (reported/triaged/in_fix/resolved/wont_fix) with simpler naming appropriate for debt items. `identified` replaces `reported` because debt is discovered, not reported as a defect. `cancelled` is added because debt items may become irrelevant (e.g., code is deleted).

**Consequences**: Projects using the advanced workflow profile get these statuses by default. Projects can override via `tech_debt_workflow` block in `.sharkworkflow.json`.

### ADR-5: No Linked Entity Fields (Use Entity Relationships)

**Context**: Bug has legacy `linked_entity_type`/`linked_entity_key` fields marked for deprecation. Should Tech-Debt replicate these?

**Decision**: Do NOT add legacy linked entity fields. Tech-debt items link to other entities exclusively through the `entity_relationships` table via `EntityRelationshipService`.

**Rationale**: The Bug model's linked fields are explicitly marked as LEGACY with deprecation comments. Adding them to a new entity would be creating new technical debt while building a tech-debt tracker.

**Consequences**: Tech-debt linking requires the entity relationship system. The `shark td` commands do not have `--linked-entity-type` or `--linked-entity-key` flags. Users link via `shark link TD-001 E07-F01-001`.

### ADR-6: Schema Version Bump 10 to 11

**Context**: Adding a new table requires a database migration.

**Decision**: Bump `CurrentSchemaVersion` from 10 to 11. Add `migrateTechDebtTable()` function following the exact pattern of `migrateBugAndChangeCardTables()`.

**Rationale**: The migration system checks `CurrentSchemaVersion` against the stored version. Without the bump, existing databases never run the new migration.

**Consequences**: Developer must set `skip_migrations: false` in `.sharkconfig.json` before the first shark command after this change, then reset to `true`. This is documented in database-critical.md.

---

## 3. Data Model

### 3.1 TechDebt Model (`internal/models/tech_debt.go`)

```go
type TechDebtStatus string
type TechDebtCategory string
type TechDebtSeverity string

const (
    TechDebtCategoryCodeQuality    TechDebtCategory = "code-quality"
    TechDebtCategoryArchitecture   TechDebtCategory = "architecture"
    TechDebtCategoryDependency     TechDebtCategory = "dependency"
    TechDebtCategoryTesting        TechDebtCategory = "testing"
    TechDebtCategoryPerformance    TechDebtCategory = "performance"
    TechDebtCategoryDocumentation  TechDebtCategory = "documentation"
)

var ValidTechDebtCategories = map[TechDebtCategory]bool{ ... }

// Reuse severity scale from Bug
const (
    TechDebtSeverityCritical TechDebtSeverity = "critical"
    TechDebtSeverityHigh     TechDebtSeverity = "high"
    TechDebtSeverityMedium   TechDebtSeverity = "medium"
    TechDebtSeverityLow      TechDebtSeverity = "low"
)

var ValidTechDebtSeverities = map[TechDebtSeverity]bool{ ... }

type TechDebt struct {
    BaseEntity                          // 9 shared fields + 10 accessor methods
    Status         TechDebtStatus   `json:"status" db:"status"`
    Category       TechDebtCategory `json:"category" db:"category"`
    Severity       TechDebtSeverity `json:"severity" db:"severity"`
    EffortEstimate *string          `json:"effort_estimate,omitempty" db:"effort_estimate"`
}
```

**Entity interface methods**: `GetEntityType() -> EntityTypeTechDebt`, `GetStatus()`, `SetStatus()`, `Validate()`.

**Key validation**: `var techDebtKeyPattern = regexp.MustCompile("^TD-\\d{3}$")`

### 3.2 Database Table Schema

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

**Indexes**:
```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_tech_debts_key ON tech_debts(key);
CREATE INDEX IF NOT EXISTS idx_tech_debts_status ON tech_debts(status);
CREATE INDEX IF NOT EXISTS idx_tech_debts_severity ON tech_debts(severity);
CREATE INDEX IF NOT EXISTS idx_tech_debts_category ON tech_debts(category);
CREATE INDEX IF NOT EXISTS idx_tech_debts_slug ON tech_debts(slug);
```

**Trigger** (updated_at auto-update):
```sql
CREATE TRIGGER IF NOT EXISTS tech_debts_updated_at
AFTER UPDATE ON tech_debts
FOR EACH ROW
BEGIN
    UPDATE tech_debts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

---

## 4. Integration Approach

### 4.1 Key Detection

Add `IsTechDebtKey()` to `internal/keys/validation.go` with regex `^TD-\d{3}$`.

Add wrapper `IsTechDebtKey()` to `internal/cli/commands/helpers.go`.

Extend `DetectEntityType()` in `helpers.go` with:
```go
if IsTechDebtKey(normalized) {
    return "tech_debt"
}
```
This must be placed BEFORE the task key checks (since TD- starts with T, it could potentially confuse slug-based task detection if checked too late).

### 4.2 Workflow Integration

| File | Change |
|------|--------|
| `internal/workflow/levels.go` | Add `LevelTechDebt = "tech_debt"` |
| `internal/config/workflow/multilevel.go` | Add `TechDebt *WorkflowConfig` field; add `case "tech_debt"` in `GetWorkflowForLevel()` |
| `internal/config/workflow/defaults.go` | Add `DefaultTechDebtWorkflow()` function |
| `internal/config/workflow/parser.go` | Add `"tech_debt_workflow": "tech_debt"` to both maps |
| `internal/config/workflow/validator.go` | Add `"tech_debt": multi.TechDebt` to validation map |

### 4.3 Entity Type Registration

| File | Change |
|------|--------|
| `internal/models/entity_note.go` | Add `EntityTypeTechDebt EntityType = "tech_debt"` constant and register in `ValidEntityTypes` |
| `internal/models/entity.go` | Add `_ Entity = (*TechDebt)(nil)` compile-time check |

### 4.4 CLI Dispatch Points (18 switch/case additions)

Every existing switch on entity type needs a `case "tech_debt":` branch. The complete list:

| File | Function/Location | Pattern |
|------|-------------------|---------|
| `commands/get.go` | `runGet()` dispatcher | Add case calling tech-debt service get |
| `commands/delete_dispatch.go` | `dispatchDelete()` | Add case calling tech-debt service delete |
| `commands/status_group.go` | `resolveTransitioner()` | Add case returning tech-debt transitioner |
| `commands/status_group.go` | `dispatchNextStatus()` | Add case returning tech-debt next-status |
| `commands/list.go` | list dispatcher | Add case calling tech-debt service list |
| `commands/update_dispatch.go` | update dispatcher | Add case calling tech-debt service update |
| `commands/note_generic.go` | `entityTypeFromName()` | Add `case "tech_debt":` returning `EntityTypeTechDebt` |
| `commands/context.go` | entity type resolver | Add `case "tech_debt":` returning `EntityTypeTechDebt` |
| `commands/link.go` | entity type resolver | Add tech_debt case |
| `commands/errors.go` | error suggestions | Add tech_debt suggestions |
| `commands/render_common.go` | render dispatcher | Add tech_debt case |
| `commands/analytics.go` | analytics dispatcher | Add tech_debt case |
| `commands/workflow_show_action.go` | workflow action dispatcher | Add tech_debt case |
| `commands/run.go` | run dispatcher (2 locations) | Add tech_debt case |
| `commands/helpers.go` | `DetectEntityType()` | Add IsTechDebtKey check |
| `commands/helpers.go` | `ParseGetArgs()` | Add scopeTechDebt type and case |

### 4.5 Service Layer Wiring

| File | Change |
|------|--------|
| `internal/cli/services_global.go` | Add `GetTechDebtService()` accessor (pattern: copy `GetBugService()`) |
| `internal/cli/services_global.go` | Add tech-debt registration in `GetEntityRegistry()` |
| `internal/cli/services_global.go` | Add tech-debt repo to `GetDashboardAnalyticsService()` |

### 4.6 Search Integration

Extend `SearchAll()` in `internal/repository/search/repository.go` with one additional UNION ALL block:
```sql
UNION ALL

SELECT 'tech_debt' AS entity_type, key, title, CAST(status AS TEXT),
       CAST(category AS TEXT) AS severity
FROM tech_debts
WHERE title LIKE ? OR key LIKE ?
   OR COALESCE(description, '') LIKE ?
```

Add 3 more `pattern` args to the args slice.

Update `entityType` filter comment to include `"tech_debt"`.

### 4.7 Analytics Integration

Extend `DashboardAnalyticsService` in `internal/services/dashboard_analytics_service.go`:
- Add `TechDebtSummaryRepository` interface (pattern: `BugSummaryRepository`)
- Add `techDebtRepo` field to the service struct
- Add `techDebtRepo` parameter to `NewDashboardAnalyticsService()`
- Add `GetTechDebtAnalytics()` method

### 4.8 Template System

Add `TechDebtPlaceholders()` to `internal/config/template/helpers.go` (pattern: `BugPlaceholders()`). Include `category`, `severity`, `effort_estimate` in the placeholder map.

Create `shark-templates/tech_debt/` directory with templates matching default workflow statuses:
- `identified.tmpl`
- `triaged.tmpl`
- `in_progress.tmpl`
- `resolved.tmpl`
- `wont_fix.tmpl`
- `cancelled.tmpl`

### 4.9 File Path Display on Entity Create (SC-6)

Modify these existing creation commands to print `entity.GetFilePath()` after successful creation:
- `internal/cli/commands/epic.go` -- `runEpicCreate()`
- `internal/cli/commands/feature.go` -- `runFeatureCreate()`
- `internal/cli/commands/create.go` -- `runCreate()` (unified create)
- `internal/cli/commands/bug.go` -- `runBugCreate()`
- `internal/cli/commands/change.go` -- `runChangeCreate()`

The new `internal/cli/commands/tech_debt.go` will include file path display from the start.

This is a UI-only change. No service or repository modifications needed.

---

## 5. Migration Strategy

### Database Migration

1. Add `migrateTechDebtTable()` function to `internal/db/db.go`
2. Call it from `runMigrations()` (after existing migrations)
3. Bump `CurrentSchemaVersion` from 10 to 11
4. Migration checks `IF NOT EXISTS` for idempotency
5. Migration is purely additive (CREATE TABLE) -- no ALTER on existing tables

### Developer Instructions

After merging the migration code:
1. Set `skip_migrations: false` in `.sharkconfig.json`
2. Run any shark command (migration auto-applies)
3. Set `skip_migrations: true` in `.sharkconfig.json`

### Backward Compatibility

- All existing tables are untouched
- All existing CLI commands continue to work
- All existing tests continue to pass
- The tech_debts table simply does not exist in older schema versions

---

## 6. File Inventory

### New Files (7)

| File | Purpose | Size Estimate |
|------|---------|---------------|
| `internal/models/tech_debt.go` | Model, key validation, category/severity constants, Validate() | ~100 lines |
| `internal/repository/tech_debt/repository.go` | Full CRUD repository with filters | ~360 lines |
| `internal/services/tech_debt_service.go` | Service with workflow delegation, CRUD orchestration | ~450 lines |
| `internal/services/tech_debt_dto.go` | CreateTechDebtInput, TechDebtUpdates, TechDebtFilters DTOs | ~50 lines |
| `internal/services/tech_debt_repo_adapter.go` | EntityRepository adapter for entity registry | ~60 lines |
| `internal/cli/commands/tech_debt.go` | `shark td` command group with all subcommands | ~500 lines |
| `shark-templates/tech_debt/*.tmpl` (6 files) | Per-status markdown templates | ~30 lines each |

### Modified Files (18)

| File | Change | Complexity |
|------|--------|------------|
| `internal/models/entity_note.go` | Add EntityTypeTechDebt + ValidEntityTypes entry | XS (2 lines) |
| `internal/models/entity.go` | Add compile-time interface check | XS (1 line) |
| `internal/keys/validation.go` | Add techDebtKeyPattern regex + IsTechDebtKey() | XS (10 lines) |
| `internal/workflow/levels.go` | Add LevelTechDebt constant | XS (1 line) |
| `internal/config/workflow/multilevel.go` | Add TechDebt field + switch case | XS (6 lines) |
| `internal/config/workflow/defaults.go` | Add DefaultTechDebtWorkflow() function | S (30 lines) |
| `internal/config/workflow/parser.go` | Add 2 map entries | XS (2 lines) |
| `internal/config/workflow/validator.go` | Add 1 map entry | XS (1 line) |
| `internal/config/template/helpers.go` | Add TechDebtPlaceholders() | S (20 lines) |
| `internal/db/db.go` | Add migrateTechDebtTable() + bump schema version | M (60 lines) |
| `internal/cli/services_global.go` | Add GetTechDebtService() + registry + analytics wiring | S (40 lines) |
| `internal/cli/commands/helpers.go` | Add IsTechDebtKey(), DetectEntityType case, ParseGetArgs scope | S (20 lines) |
| `internal/cli/commands/get.go` | Add tech_debt case | XS (3 lines) |
| `internal/cli/commands/delete_dispatch.go` | Add tech_debt case | XS (3 lines) |
| `internal/cli/commands/status_group.go` | Add tech_debt cases (2 functions) | XS (6 lines) |
| `internal/repository/search/repository.go` | Add tech_debts UNION ALL block + args | S (10 lines) |
| `internal/services/dashboard_analytics_service.go` | Add TechDebtSummaryRepository + method | S (30 lines) |
| `internal/cli/commands/*.go` (5+ files for SC-6) | Add file path display on entity create | S (5 lines each) |

Additional modified files for dispatch (each XS, 2-4 lines):
- `commands/list.go`
- `commands/update_dispatch.go`
- `commands/note_generic.go`
- `commands/context.go`
- `commands/link.go`
- `commands/errors.go`
- `commands/render_common.go`
- `commands/analytics.go`
- `commands/workflow_show_action.go`
- `commands/run.go`

---

## 7. Implementation Order

### Phase 1: Foundation (no DB changes, no breaking changes)

1. Model (`tech_debt.go`) + entity type registration + entity interface check
2. Key validation (`keys/validation.go`)
3. Workflow level + defaults + multilevel + parser + validator
4. Template helpers + template files

### Phase 2: Data Layer (requires DB migration)

5. Database migration (`db.go`) + schema version bump
6. Repository (`tech_debt/repository.go`)
7. Service DTOs + repo adapter + service (`tech_debt_service.go`)

### Phase 3: CLI Integration

8. Service accessor + entity registry registration (`services_global.go`)
9. Key detection + dispatch points in helpers.go
10. All CLI dispatch points (get, delete, status, list, update, context, notes, link, errors, render, analytics, workflow_show_action, run)
11. CLI command group (`tech_debt.go`)

### Phase 4: Cross-Cutting

12. Search integration (search/repository.go)
13. Analytics integration (dashboard_analytics_service.go)
14. File path display on entity create (SC-6, 5+ existing command files)

**Quality gate after each phase**: `make fmt && make lint && make test`

---

## 8. Risk Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Schema version bump forgotten | HIGH if not done together | Migration never runs | Add migration + version bump in same commit |
| Missing switch/case in dispatch | MEDIUM | Silent "unsupported entity type" error | Grep for all `case "bug"` and add parallel `case "tech_debt"` |
| ValidEntityTypes missing registration | MEDIUM | Notes and history writes fail | Add to ValidEntityTypes in same commit as model |
| Key format collision | LOW | False key detection | TD- prefix is distinct from all existing patterns (verified) |
| Analytics wiring omission | LOW | Empty analytics (service is nil-safe) | Wire in same commit as analytics method |

---

*Architecture by: Architect agent | Date: 2026-04-05*
