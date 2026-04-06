# Research Report: Add Tech-Debt Entity Type (E25)

## Executive Summary

The codebase already has two fully implemented standalone entity types — Bug (B###) and ChangeCard (CC-###) — that are structurally identical to what Tech-Debt (TD-###) requires. Every layer needed (model, repository, service, CLI commands, workflow config, search, analytics, entity registry) follows a deeply established pattern that can be replicated with minimal new design. The risk is **low**: this is largely a mechanical duplication and extension of the Bug entity, with a new key format, two new domain fields (category, effort_estimate), and one new workflow level added to existing infrastructure. No existing abstractions need modification beyond additive registration.

---

## Research Questions

1. What existing code can be directly replicated for Tech-Debt?
2. What infrastructure requires additive changes (not new files)?
3. What are the exact integration points that must be wired?
4. What technical risks exist?
5. What is the recommended implementation order?

---

## Methodology

- Read Bug and ChangeCard models, repositories, services, CLI commands in full
- Traced every integration point: EntityRegistry, workflow levels, key detection, status dispatch, search, analytics, template system, parser, multilevel workflow config
- Verified `CurrentSchemaVersion = 10` (must bump to 11)
- Identified all switch statements and maps that enumerate entity types

---

## Findings

### Finding 1: Complete Reference Implementation Exists (Bug Entity)

The Bug entity is an exact structural template for Tech-Debt. Every layer is present and working.

**File paths (lines are approximate entry points):**

| Layer | Bug File | Lines |
|-------|----------|-------|
| Model | `internal/models/bug.go` | 1–87 |
| Model key validation | `internal/models/bug.go` | 47–86 |
| Repository | `internal/repository/bug/repository.go` | 1–360 |
| Repository filters | `internal/repository/bug/repository.go` | 222–228 |
| Service | `internal/services/bug_service.go` | 1–454 |
| Service DTOs | `internal/services/bug_dto.go` | 1–42 |
| Service repo interface | `internal/services/bug_service.go` | 19–29 |
| Repository adapter | `internal/services/bug_repo_adapter.go` | 1–59 |
| CLI commands | `internal/cli/commands/bug.go` | 1–~500 |
| Service accessor | `internal/cli/services_global.go` | 432–464 |

**Implications:** All new Tech-Debt files can be written by copying the Bug counterpart and substituting:
- `Bug` → `TechDebt`
- `BugSeverity` → `TechDebtCategory` + `TechDebtSeverity` (two domain fields instead of one)
- `B###` → `TD-###` (key format)
- `bugs` → `tech_debts` (table name)
- `bug` → `tech_debt` (entity type string)
- `linked_entity_type/linked_entity_key` → `effort_estimate` + optional linked fields (per scope)

---

### Finding 2: EntityType Enum — Additive Extension Required

**File:** `internal/models/entity_note.go`, lines 10–27

```go
const (
    EntityTypeEpic    EntityType = "epic"
    EntityTypeFeature EntityType = "feature"
    EntityTypeTask    EntityType = "task"
    EntityTypeChange  EntityType = "change"
    EntityTypeBug     EntityType = "bug"
)

var ValidEntityTypes = map[EntityType]bool{
    EntityTypeEpic:    true,
    EntityTypeFeature: true,
    EntityTypeTask:    true,
    EntityTypeChange:  true,
    EntityTypeBug:     true,
}
```

**Required change:** Add `EntityTypeTechDebt EntityType = "tech_debt"` constant and register it in `ValidEntityTypes`. This is a 2-line addition.

**Why not a new file:** The `EntityType` type and `ValidEntityTypes` map are defined in `entity_note.go`. The Tech-Debt type must be added there for consistency with Bug and Change.

---

### Finding 3: Entity Interface — Zero Changes Required

**File:** `internal/models/entity.go`, lines 11–86

`BaseEntity` already provides all shared fields (ID, Key, Title, Slug, Description, FilePath, ContextData, CreatedAt, UpdatedAt) and all accessor methods. The compile-time interface check block (lines 79–85) needs one new line:

```go
_ Entity = (*TechDebt)(nil)
```

The `Entity` interface itself does **not** change.

---

### Finding 4: Key Detection — Three Additive Extensions

**Location 1:** `internal/keys/validation.go` — add TD-### regex and `IsTechDebtKey()` function (pattern: Bug is `^B\d+$` at line 20; TD should be `^TD-\d{3}$`)

**Location 2:** `internal/cli/commands/helpers.go` — add `IsTechDebtKey()` wrapper (same pattern as `IsBugKey` at line 55–60) and extend `DetectEntityType()` (line 667) with:
```go
if IsTechDebtKey(normalized) {
    return "tech_debt"
}
```

**Location 3:** `internal/cli/commands/helpers.go`, `ParseGetArgs()` (line 298) — add tech_debt scope type and key detection alongside existing `scopeBug` at line 363–366.

**Pattern reference:** `internal/keys/validation.go` lines 19–22 show exactly how `bugKeyPattern` and `changeKeyPattern` are defined and used.

---

### Finding 5: Workflow Level — Additive Extension Required

**File:** `internal/workflow/levels.go` — add `LevelTechDebt = "tech_debt"`

**File:** `internal/config/workflow/multilevel.go` — `MultiLevelWorkflow` struct (lines 5–24) needs a `TechDebt *WorkflowConfig` field; `GetWorkflowForLevel()` (lines 34–64) needs a `case "tech_debt"` branch

**File:** `internal/config/workflow/defaults.go` — add `DefaultTechDebtWorkflow()` function (pattern: copy `DefaultBugWorkflow()` at line 107–136, adjust statuses for tech-debt lifecycle: `identified → triaged → in_progress → resolved / cancelled / wont_fix`)

**File:** `internal/config/workflow/parser.go` — add `"tech_debt_workflow": "tech_debt"` to `workflowKeyToLevel` map (line 232–238) and corresponding field pointer in `entityKeys` map (line 242–248)

**File:** `internal/config/workflow/validator.go` — add `"tech_debt": multi.TechDebt` to the entity levels map (line 301–302)

---

### Finding 6: Entity Registry — One Additive Registration

**File:** `internal/cli/services_global.go`, lines 79–92 (inside `GetEntityRegistry()`)

Add:
```go
c.registry.Register(models.EntityTypeTechDebt,
    services.NewTechDebtRepositoryAdapter(repository.NewTechDebtRepository(db)))
```

This is pattern-identical to Bug (lines 86–88) and Change (lines 88–90).

---

### Finding 7: Status Dispatch — Four Switch Cases

The following functions dispatch status transitions/advances by entity type. Each needs one `case "tech_debt":` branch:

| Function | File | Lines |
|----------|------|-------|
| `resolveTransitioner()` | `status_group.go` | 175–189 |
| `dispatchNextStatus()` | `status_group.go` | 191–207 |
| `runGet()` dispatcher | `get.go` | 54–79 |
| `delete_dispatch.go` | `delete_dispatch.go` | ~41–55 |

Pattern: Bug cases at lines 182–183 and 200–201 in `status_group.go`.

---

### Finding 8: Database Schema — New Table + Schema Version Bump

**File:** `internal/db/db.go`
- Current: `CurrentSchemaVersion = 10` (line 438)
- Required: Bump to `11`
- Add migration function `migrateTechDebtTable()` following the `migrateBugAndChangeCardTables()` pattern (line 2009–2066)

**Tech-Debt table schema (proposed):**
```sql
CREATE TABLE IF NOT EXISTS tech_debts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    key           TEXT NOT NULL UNIQUE,
    title         TEXT NOT NULL,
    slug          TEXT,
    description   TEXT,
    status        TEXT NOT NULL DEFAULT 'identified',
    category      TEXT NOT NULL DEFAULT 'code-quality',
    severity      TEXT NOT NULL DEFAULT 'medium',
    effort_estimate TEXT,
    context_data  TEXT,
    file_path     TEXT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Indexes required:**
```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_tech_debts_key ON tech_debts(key);
CREATE INDEX IF NOT EXISTS idx_tech_debts_status ON tech_debts(status);
CREATE INDEX IF NOT EXISTS idx_tech_debts_severity ON tech_debts(severity);
CREATE INDEX IF NOT EXISTS idx_tech_debts_category ON tech_debts(category);
CREATE INDEX IF NOT EXISTS idx_tech_debts_slug ON tech_debts(slug);
```

**Trigger required (updated_at):** Same pattern as bugs trigger (line 2057–2066).

**IMPORTANT for developer:** After adding this migration, set `skip_migrations: false` in `.sharkconfig.json` before running the next shark command, then reset to `true`. This is required per the database-critical rule.

---

### Finding 9: Search Integration — UNION SQL Extension

**File:** `internal/repository/search/repository.go`, `SearchAll()` function (line 45–122)

The current UNION SQL covers: epic, feature, task, bug, change_cards (lines 53–84). Add one more:
```sql
UNION ALL

SELECT 'tech_debt' AS entity_type, key, title, CAST(status AS TEXT), CAST(category AS TEXT) AS severity
FROM tech_debts
WHERE title LIKE ? OR key LIKE ?
   OR COALESCE(description, '') LIKE ?
```

Note: The `EntitySearchResult` struct uses `Severity string` (line 37); for tech_debt, populate this field with `category` since severity also exists on tech_debt items. Alternatively, rename to `Tag` in a future refactor. For now, pass `category` for display parity with Bug's severity.

Add 3 more `pattern` args to `args` slice.

---

### Finding 10: Analytics — DashboardAnalyticsService Extension

**File:** `internal/services/dashboard_analytics_service.go`

The service currently handles Bug and ChangeCard analytics (lines 61–113). Add:
- `TechDebtSummaryRepository` interface (parallel to `BugSummaryRepository` at line 10–25)
- `GetTechDebtAnalytics()` method
- Wire in `GetDashboardAnalyticsService()` in `services_global.go`

The Tech-Debt repo should implement `CountByStatus()` and `CountByCategory()` (parallel to `CountBySeverity()` for bugs).

---

### Finding 11: Workflow Config Config Parser — Additive Extension

**File:** `internal/config/workflow/parser.go`, line 236–247

The `workflowKeyToLevel` map and `entityKeys` map need:
```go
"tech_debt_workflow": "tech_debt"
```

This allows `.sharkconfig.json` (or `.sharkworkflow.json`) to contain an optional `tech_debt_workflow` block for project-specific overrides.

---

### Finding 12: Template System — New Template Directory

**File system:** `shark-templates/tech_debt/` directory with per-status `.tmpl` files

**Pattern:** `shark-templates/bug/` has: `blocked.tmpl`, `cancelled.tmpl`, `completed.tmpl`, `draft.tmpl`, `in_development.tmpl`, `on_hold.tmpl`, `ready_for_development.tmpl`

For tech_debt, create templates matching the default workflow statuses: `identified.tmpl`, `triaged.tmpl`, `in_progress.tmpl`, `resolved.tmpl`, `cancelled.tmpl`, `wont_fix.tmpl`

**File:** `internal/config/template/helpers.go` — add `TechDebtPlaceholders()` (pattern: `BugPlaceholders()` at line 215–234). Include `category`, `severity`, `effort_estimate` in the placeholder map.

---

### Finding 13: File Path Display on Entity Create (SC-6)

**Scope:** All entity creation commands currently do not display the file path.

**Affected files:**
- `internal/cli/commands/epic.go` — `runEpicCreate`
- `internal/cli/commands/feature.go` — `runFeatureCreate`
- `internal/cli/commands/create.go` — `runCreate` (unified create)
- `internal/cli/commands/bug.go` — `runBugCreate`
- `internal/cli/commands/change.go` — `runChangeCreate`
- New: `internal/cli/commands/tech_debt.go` — `runTechDebtCreate`

**Approach:** After successful creation, extract `entity.GetFilePath()` and print it. Each creation command already receives the returned entity object. This is a UI-only change requiring no service or repository modifications.

---

### Finding 14: CLI Command File Structure

**Pattern reference:** `internal/cli/commands/bug.go` implements all CRUD and triage subcommands under `bugCmd` parent.

New file: `internal/cli/commands/tech_debt.go`
- Command group: `tdCmd` registered with `GroupID: "advanced"`
- Subcommands: `td create`, `td get`, `td list`, `td update`, `td delete`, `td triage`, `td note`, `td notes`, `td context`
- Service interface: `techDebtServicer` (parallel to `bugServicer` at line 17–26)
- Override var: `tdSvcOverride` for test injection

**Service accessor:** `cli.GetTechDebtService()` in `internal/cli/services_global.go`

---

## Integration Points Summary

| Integration Point | File | Change Type | Complexity |
|---|---|---|---|
| EntityType constant | `models/entity_note.go` | ADD 2 lines | XS |
| Entity interface check | `models/entity.go` | ADD 1 line | XS |
| TechDebt model | NEW `models/tech_debt.go` | NEW file | S |
| Key validation | `keys/validation.go` | ADD func + regex | XS |
| Key detection wrapper | `commands/helpers.go` | ADD func + switch case | XS |
| ParseGetArgs | `commands/helpers.go` | ADD scope type + case | S |
| Workflow level constant | `workflow/levels.go` | ADD 1 line | XS |
| MultiLevelWorkflow struct | `config/workflow/multilevel.go` | ADD field + case | XS |
| Default workflow | `config/workflow/defaults.go` | ADD function | S |
| Workflow parser | `config/workflow/parser.go` | ADD 2 map entries | XS |
| Workflow validator | `config/workflow/validator.go` | ADD 1 map entry | XS |
| DB migration + schema version | `db/db.go` | ADD function + bump | M |
| Repository | NEW `repository/tech_debt/repository.go` | NEW file | M |
| Service repo adapter | NEW `services/tech_debt_repo_adapter.go` | NEW file | S |
| Service DTOs | NEW `services/tech_debt_dto.go` | NEW file | S |
| Service | NEW `services/tech_debt_service.go` | NEW file | M |
| Service accessor | `cli/services_global.go` | ADD function | S |
| Entity registry | `cli/services_global.go` | ADD 2 lines | XS |
| Status dispatch | `commands/status_group.go` | ADD 2 cases | XS |
| Get dispatcher | `commands/get.go` | ADD 1 case | XS |
| Delete dispatcher | `commands/delete_dispatch.go` | ADD 1 case | XS |
| CLI commands | NEW `commands/tech_debt.go` | NEW file | M |
| Search SQL | `repository/search/repository.go` | ADD UNION block | S |
| Analytics | `services/dashboard_analytics_service.go` | ADD method | S |
| Analytics service accessor | `cli/services_global.go` | MODIFY wiring | XS |
| Template helpers | `config/template/helpers.go` | ADD function | S |
| Templates | NEW `shark-templates/tech_debt/*.tmpl` | NEW files | S |
| File path display (SC-6) | 5+ command files | MODIFY output | S |

---

## Extension vs. New Code Analysis

| Component | Extend Existing? | Why? |
|---|---|---|
| Entity interface (`Entity`) | Extend (add compile check) | Interface already supports all needed methods via BaseEntity |
| EntityType enum | Extend (add constant) | Single source of truth in `entity_note.go` |
| Key detection (`DetectEntityType`) | Extend (add case) | Function dispatches all key types; extending avoids parallel logic |
| Status dispatch (`status_group.go`) | Extend (add case) | Existing dispatch table already handles bug/change; one more case |
| Search SQL | Extend (add UNION) | SearchAll is one function querying all tables |
| MultiLevelWorkflow struct | Extend (add field) | All entity workflow configs live in this struct |
| Workflow parser | Extend (add map entry) | Config-driven; one additional key-to-level mapping |
| Analytics service | Extend (add method + repo interface) | Pattern already set by Bug and ChangeCard |
| Template helpers | Extend (add function) | `EntityPlaceholders()` base already handles shared fields |
| DB migration | Extend (new migration function, version bump) | Pattern requires new function added to `runMigrations()` |
| Model, repository, service, CLI | **NEW files** | Domain-specific code; cannot extend Bug without coupling tech-debt to bugs |

The principle: **all infrastructure is extended; all domain logic is new files**.

---

## Technical Risks and Feasibility

### Risk 1: Schema version bump coordination (HIGH likelihood if forgotten, LOW impact otherwise)
- **Description:** `CurrentSchemaVersion` must go from `10` to `11`. If bumped without the matching migration function, or vice versa, the migration will not run on existing databases.
- **Mitigation:** Add migration function and version bump in the same commit. Developer must set `skip_migrations: false` before running any shark command post-merge.

### Risk 2: Switch statement coverage (MEDIUM likelihood, LOW impact)
- **Description:** Multiple switch statements in CLI layer dispatch by entity type. Missing a `tech_debt` case in any one of them causes a silent "unsupported entity type" error at runtime.
- **Mitigation:** Grep for `"bug"` and `"change"` cases across the codebase before finalizing. Key locations: `status_group.go` (2 functions), `get.go` (1 function), `delete_dispatch.go` (1 function), `context_generic.go` (`entityTypeFromName` at line 71).

### Risk 3: `entity_history` entity_type validation (MEDIUM likelihood, LOW impact)
- **Description:** `ValidEntityTypes` map is checked in `EntityNote.Validate()` and `EntityHistory.Validate()`. If `EntityTypeTechDebt` is not added to the map, history and notes writes will fail validation silently.
- **Mitigation:** Add to `ValidEntityTypes` in the same commit as the model.

### Risk 4: Key format collision (LOW likelihood, LOW impact)
- **Description:** TD-### uses `TD-` prefix. The existing `changeKeyPattern` matches `^C\d+$` and `changeCardKeyPattern` matches `^CC-\d{3}$`. `TD-` is distinct from both and from epic/feature/task patterns. No collision exists.
- **Confirmation:** `bugKeyPattern = regexp.MustCompile('^B\d+$')`, `changeKeyPattern = regexp.MustCompile('^C\d+$')`, `changeCardKeyPattern = regexp.MustCompile('^CC-\d{3}$')` — TD-### cannot match any of these.

### Risk 5: Analytics wiring omission (LOW likelihood, LOW impact)
- **Description:** `DashboardAnalyticsService` is wired in `GetDashboardAnalyticsService()`. If tech_debt summary repo is not passed, analytics silently returns empty (service handles nil repo gracefully per line 69–70).
- **Mitigation:** Add tech_debt repo to `NewDashboardAnalyticsService()` constructor at the same time as analytics method.

### Feasibility Assessment
**FEASIBLE — LOW RISK.** All required patterns are established and working. No new architectural abstractions are needed. The implementation is primarily additive with well-defined integration points.

---

## Recommended Implementation Approach

**Phase 1 — Foundation (non-breaking, no DB changes):**
1. `internal/models/tech_debt.go` — model, key validation, severity/category constants, `Validate()`
2. `internal/models/entity_note.go` — add `EntityTypeTechDebt` and register in `ValidEntityTypes`
3. `internal/models/entity.go` — add compile-time interface check
4. `internal/keys/validation.go` — add `IsTechDebtKey()` and regex
5. `internal/workflow/levels.go` — add `LevelTechDebt`
6. `internal/config/workflow/defaults.go` — add `DefaultTechDebtWorkflow()`
7. `internal/config/workflow/multilevel.go` — add `TechDebt` field and case
8. `internal/config/workflow/parser.go` + `validator.go` — add `tech_debt_workflow` map entries
9. `internal/config/template/helpers.go` — add `TechDebtPlaceholders()`
10. `shark-templates/tech_debt/*.tmpl` — per-status templates

**Phase 2 — Data Layer (requires DB migration):**
11. `internal/db/db.go` — add `migrateTechDebtTable()`, bump `CurrentSchemaVersion` to 11
12. `internal/repository/tech_debt/repository.go` — full CRUD repository
13. `internal/services/tech_debt_repo_adapter.go` — EntityRepository adapter
14. `internal/services/tech_debt_dto.go` — CreateTechDebtInput, TechDebtUpdates, TechDebtFilters
15. `internal/services/tech_debt_service.go` — full service with workflow delegation

**Phase 3 — CLI Integration:**
16. `internal/cli/services_global.go` — `GetTechDebtService()` + EntityRegistry registration
17. `internal/cli/commands/helpers.go` — `IsTechDebtKey()`, `DetectEntityType()` case, `ParseGetArgs` scope
18. `internal/cli/commands/get.go` — add `tech_debt` case
19. `internal/cli/commands/delete_dispatch.go` — add `tech_debt` case
20. `internal/cli/commands/status_group.go` — add `tech_debt` cases to dispatch functions
21. `internal/cli/commands/tech_debt.go` — full `td` command group with all subcommands

**Phase 4 — Cross-Cutting Features:**
22. `internal/repository/search/repository.go` — add tech_debts UNION block
23. `internal/services/dashboard_analytics_service.go` — add `GetTechDebtAnalytics()`
24. Entity creation file path display (modify 5+ existing command files for SC-6)

**Quality gate after each phase:** `make fmt && make lint && make test`

---

## References

- Bug model: `/home/jwwel/projects/shark-task-manager/internal/models/bug.go`
- Bug repository: `/home/jwwel/projects/shark-task-manager/internal/repository/bug/repository.go`
- Bug service: `/home/jwwel/projects/shark-task-manager/internal/services/bug_service.go`
- Bug CLI commands: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/bug.go`
- Bug repo adapter: `/home/jwwel/projects/shark-task-manager/internal/services/bug_repo_adapter.go`
- Bug DTO: `/home/jwwel/projects/shark-task-manager/internal/services/bug_dto.go`
- Entity type enum: `/home/jwwel/projects/shark-task-manager/internal/models/entity_note.go` (lines 9–27)
- Entity interface + BaseEntity: `/home/jwwel/projects/shark-task-manager/internal/models/entity.go`
- Key detection: `/home/jwwel/projects/shark-task-manager/internal/keys/validation.go` (lines 1–25)
- DetectEntityType: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/helpers.go` (line 667)
- Status dispatch: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status_group.go` (lines 175–207)
- Workflow levels: `/home/jwwel/projects/shark-task-manager/internal/workflow/levels.go`
- MultiLevelWorkflow: `/home/jwwel/projects/shark-task-manager/internal/config/workflow/multilevel.go`
- Default workflows: `/home/jwwel/projects/shark-task-manager/internal/config/workflow/defaults.go` (lines 107–163)
- Workflow parser: `/home/jwwel/projects/shark-task-manager/internal/config/workflow/parser.go` (lines 232–247)
- DB schema/version: `/home/jwwel/projects/shark-task-manager/internal/db/db.go` (lines 436–438, 749+, 2009+)
- Search repository: `/home/jwwel/projects/shark-task-manager/internal/repository/search/repository.go`
- Entity registry: `/home/jwwel/projects/shark-task-manager/internal/cli/services_global.go` (lines 79–92)
- Services global accessors: `/home/jwwel/projects/shark-task-manager/internal/cli/services_global.go` (lines 399–464)
- Template helpers: `/home/jwwel/projects/shark-task-manager/internal/config/template/helpers.go` (lines 215–260)
- Templates directory: `/home/jwwel/projects/shark-task-manager/shark-templates/bug/` (reference)

---

*Produced by Researcher agent | Date: 2026-04-05*
