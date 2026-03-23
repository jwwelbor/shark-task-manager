# Maintenance Burden

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 6 — Technical Debt Assessment

## High-Cost Maintenance Areas

### TD-02: Fat Controller Pattern (Medium)

- **Location**: `internal/cli/commands/` (various legacy command files)
- **Issue**: Some older commands contain business logic, direct repository calls, and workflow validation instead of delegating to services
- **Impact**: Duplicated logic, harder to test (requires DB), inconsistent behavior between CLI and HTTP API
- **Cost**: Each new feature requires changes in both command and service layers
- **Status**: Active migration underway (Epic E15)
- **Effort**: Medium — incremental refactoring per command

### TD-06: Large Config Package (Medium)

- **Location**: `internal/config/` (27 files, 14,408 LOC)
- **Issue**: Configuration package is disproportionately large relative to its responsibility
- **Impact**: Complex to navigate, potential for circular dependencies
- **Cost**: Changes to config cascade through many dependents
- **Effort**: Medium — would benefit from splitting into sub-packages

### TD-11: Large Repository Package (Medium)

- **Location**: `internal/repository/` (77 files, 30,133 LOC)
- **Issue**: Single package contains all entity repositories, test files, and adapters
- **Impact**: Long compile times for package changes, difficult to navigate
- **Cost**: Any repository change triggers full package recompilation
- **Effort**: Small — could split into sub-packages per entity type

### TD-07: Duplicate History Tables (Low)

- **Location**: `internal/db/db.go` (schema)
- **Issue**: Both `task_history` (legacy) and `entity_history` (new polymorphic) tables track status changes
- **Impact**: Two places to query history, potential for drift
- **Cost**: Must maintain both codepaths
- **Effort**: Small — consolidate queries to use entity_history only

### TD-08: Sequential Test Execution (Low)

- **Location**: `Makefile` (`-p=1` flag)
- **Issue**: Tests run sequentially to avoid database contention
- **Impact**: Slower test suite (~2-3x slower than parallel execution)
- **Cost**: Developer feedback loop is slower
- **Effort**: Medium — would require per-test database isolation

## Template System Complexity

- **Location**: `shark-templates/` (133 files, 80+ entity templates)
- **Issue**: Large number of status-specific templates with shared patterns
- **Impact**: Adding a new status requires creating templates for each entity type
- **Mitigation**: Partial templates (`_partials/`) reduce duplication
- **Risk**: Template drift between similar statuses

## Documentation Volume

- **Metric**: ~1,955 markdown files in docs/
- **Issue**: Large documentation footprint may become stale
- **Mitigation**: Database is source of truth; docs are output artifacts
- **Risk**: Low — stale docs don't affect functionality

See also: [Summary](summary.md) | [Remediation Plan](remediation-plan.md)
