# Design Patterns

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 2 — Architecture Analysis

## Architectural Patterns

### 1. Clean/Layered Architecture
- **Where**: Entire codebase structure
- **Implementation**: `cmd/` → `internal/cli/` → `internal/services/` → `internal/repository/` → `internal/db/`
- **Why**: Enforces separation of concerns, enables independent testing, supports multiple entry points (CLI, HTTP)

### 2. Repository Pattern
- **Where**: `internal/repository/*.go`
- **Implementation**: Each entity (Epic, Feature, Task, Bug, ChangeCard) has a dedicated repository with CRUD + query methods. Repositories accept `context.Context` and return domain models.
- **Why**: Abstracts database access, enables mock injection for testing, enforces data access as a separate concern

### 3. Service Layer Pattern
- **Where**: `internal/services/*.go`
- **Implementation**: Fat services (TaskService, FeatureService, EpicService) containing all business logic, validation, orchestration, and transaction management
- **Why**: Centralizes business rules, makes them reusable across CLI and HTTP entry points

## Creational Patterns

### 4. Factory (Task Creation)
- **Where**: `internal/taskcreation/creator.go`
- **Implementation**: `Creator` service generates task keys, creates markdown files, and assembles Task models with proper defaults
- **Why**: Complex task creation involves key generation, file creation, template rendering, and database insertion — too much for inline construction

### 5. Singleton (Lazy Initialization)
- **Where**: `internal/cli/db_global.go`, `internal/cli/services_global.go`
- **Implementation**: `sync.Once` for thread-safe lazy DB initialization. Service accessors (`GetTaskService()`) create new service instances but reuse the global DB connection
- **Why**: Expensive resources (DB connection) initialized once; services are cheap to create per-call

### 6. Builder (Configuration)
- **Where**: `internal/config/workflow_parser.go`
- **Implementation**: Workflow configuration assembled from JSON config with status metadata, transition rules, and agent routing
- **Why**: Complex configuration object built step-by-step from multiple sources

## Structural Patterns

### 7. Adapter
- **Where**: `cmd/server/services.go` (repository adapters), `internal/services/*_adapter.go`
- **Implementation**: Adapters convert between service-level interfaces and concrete repository types. Example: `workSessionAdapter` converts `SessionStats` types between packages
- **Why**: Services define interfaces they need; adapters bridge concrete repository implementations

### 8. Facade
- **Where**: `internal/cli/root.go`
- **Implementation**: Root Cobra command aggregates all subcommands into a unified CLI interface with shared global flags and lifecycle hooks
- **Why**: Simplifies CLI entry point, provides consistent flags and error handling

### 9. Dependency Injection (Constructor)
- **Where**: All service constructors (`NewTaskService`, `NewFeatureService`, etc.)
- **Implementation**: Pure Go constructor injection — no DI framework, no reflection. Dependencies declared as constructor parameters, stored as struct fields
- **Why**: Compile-time safety, explicit dependencies, easy mock injection for testing

## Behavioral Patterns

### 10. Strategy (Workflow Profiles)
- **Where**: `internal/workflow/service.go`, `internal/config/`
- **Implementation**: Workflow behavior (valid transitions, agent routing, progress weights) varies by profile (basic/advanced) loaded from configuration. Same `workflow.Service` interface, different runtime behavior
- **Why**: Supports different team sizes and methodologies without code changes

### 11. Command Pattern
- **Where**: `internal/cli/commands/*.go`
- **Implementation**: Each CLI command is a thin wrapper implementing Cobra's `RunE` function — encapsulates a single operation (parse args, call service, format output)
- **Why**: Uniform command structure, consistent error handling, easy to add new commands

### 12. Observer (Database Triggers)
- **Where**: `internal/db/db.go` (trigger definitions)
- **Implementation**: SQLite triggers automatically update `updated_at` timestamps and create `task_history` records on status changes
- **Why**: Ensures audit trail and timestamp consistency without application-level coordination

### 13. Template Method
- **Where**: `internal/templates/`, `shark-templates/`
- **Implementation**: Go `text/template` with 80+ status-specific templates. Base template structure with customizable sections per entity type and status
- **Why**: Consistent entity file format with status-specific guidance content

## Data Patterns

### 14. Unit of Work (Transaction Ownership)
- **Where**: Service layer methods that span multiple repository calls
- **Implementation**: Services begin transactions, coordinate multiple repository operations, and commit/rollback. Repositories accept `*sql.Tx` for participation
- **Why**: Ensures atomicity across related database operations

### 15. Polymorphic Entity
- **Where**: `internal/models/entity.go`
- **Implementation**: Common `Entity` interface implemented by Epic, Feature, Task, Bug, ChangeCard. Polymorphic tables (`entity_history`, `entity_notes`, `entity_documents`) use `entity_type` + `entity_id` columns
- **Why**: Enables generic services (EntityService, NoteService) that work with any entity type

### 16. Dual Key (Slug Architecture)
- **Where**: All entity repositories, `internal/slug/`
- **Implementation**: Each entity has both a numeric key (`E07-F01-001`) and an auto-generated slug (`implement-jwt-token`). Lookup tries numeric first, then slug-qualified match
- **Why**: Human-readable identifiers while maintaining machine-parseable key hierarchy

## Anti-Patterns Present (Legacy)

### Fat Controllers (Being Migrated)
- **Where**: Some older commands in `internal/cli/commands/`
- **Issue**: Business logic directly in CLI handlers instead of services
- **Status**: Active migration to service layer (Epic E15)
- **Impact**: Duplicated logic, untestable without DB, mixed concerns

See also: [System Overview](system-overview.md) | [Components](components.md) | [Dependencies](dependencies.md)
