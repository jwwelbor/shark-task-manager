# E23 OpenTelemetry Observability - Codebase Research Report

**Date:** 2026-03-22
**Epic:** E23 - OpenTelemetry Observability (tracing, metrics, structured logging)
**Researcher:** Researcher Agent

---

## Executive Summary

The shark-task-manager codebase has zero existing telemetry infrastructure. All observability work for E23 is net-new. However, the existing architecture provides excellent foundations: `context.Context` is already threaded throughout the entire CLI → Service → Repository → Database stack, Cobra lifecycle hooks are in place for SDK initialization/shutdown, the config system has a `RawData` escape-hatch for additive JSON fields, and the `sync.Once` singleton pattern for global services is well-established and directly reusable. The implementation is feasible with low risk; the primary risk is test isolation (OTel SDK must be no-op by default in tests).

---

## 1. Existing Implementations Relevant to This Epic

### 1.1 Current Logging — What Must Be Replaced

All diagnostic logging currently uses the `log` standard library package (unstructured, no levels, writes to stderr by default). There is NO existing `slog`, structured logging, or telemetry code anywhere.

**`log.*` calls in `internal/` (23 occurrences):**

| File | Line(s) | Call |
|------|---------|------|
| `internal/config/manager.go` | 58 | `log.Printf("Warning: Invalid last_sync_time format...")` |
| `internal/config/template_helpers.go` | 330, 343, 441, 450, 516, 529 | `log.Printf(...)` (template rendering warnings) |
| `internal/config/orchestrator_action.go` | 182 | `log.Printf(...)` |
| `internal/status/derivation.go` | 30, 46, 64 | `log.Printf(...)` (status derivation warnings) |
| `internal/services/epic_service.go` | 224 | `log.Printf("WARNING: ...")` |
| `internal/services/task_service.go` | 631, 976, 996 | `log.Printf("WARNING: ...")` |
| `internal/services/feature_service.go` | 238, 750 | `log.Printf("WARNING: ...")` |
| `internal/services/entity_service.go` | 58 | `log.Printf(...)` |
| `internal/services/display_service.go` | 449, 634, 668 | `log.Printf(...)` |

**`log.*` calls in `cmd/` (68 occurrences):**

| File | Representative calls |
|------|---------------------|
| `cmd/server/main.go` | `log.Fatal(...)`, `log.Println(...)`, `log.Printf(...)` (6 calls) |
| `cmd/demo/main.go` | Multiple `log.Printf` calls |
| `cmd/cleanup/main.go` | Multiple `log.Printf` calls |
| `cmd/migrate-exec-order/main.go` | Multiple `log.Printf` calls |
| `cmd/backfill-slugs/main.go` | Multiple `log.Printf` calls |

**`fmt.Fprintf(os.Stderr, ...)` diagnostic calls (not user-facing output):**

| File | Line(s) |
|------|---------|
| `internal/cli/cli_error.go` | 42 |
| `internal/cli/epic_helpers.go` | 379, 790 |
| `internal/cli/feature_helpers.go` | 215, 224, 228, 750 |
| `internal/cli/helpers.go` | 820 |
| `internal/cli/root.go` | 406, 432 |
| `internal/config/workflow_parser.go` | 393 |
| `internal/services/bug_service.go` | 164 |
| `internal/services/change_card_service.go` | 169, 289 |
| `internal/taskcreation/creator.go` | 308 |
| `internal/init/profile_service.go` | 113, 127 |
| `internal/db/db.go` | 1064 |

### 1.2 User-Facing Output — Must NOT Be Replaced

The following are presentation functions, not logging. They write to stdout (or use pterm for terminal formatting) and are completely outside the scope of slog migration:

- `cli.Success(message)` — `internal/cli/root.go` line ~411
- `cli.Error(message)` — `internal/cli/root.go` line ~420
- `cli.Warning(message)` — `internal/cli/root.go` line ~432
- `cli.Info(message)` — `internal/cli/root.go` line ~444
- `cli.OutputJSON(data)` — `internal/cli/root.go` line ~367 (writes to stdout)
- `cli.OutputTable(headers, rows)` — uses pterm

### 1.3 Existing Telemetry/Tracing Code

**None.** Grep for `opentelemetry`, `otel`, `tracing`, `jaeger`, `zipkin`, `prometheus`, `slog` all return zero results in the project source.

### 1.4 Configuration System

**`internal/config/config.go`:**
- `Config` struct (line ~31): contains `RawData map[string]interface{}` which preserves all unknown JSON fields
- Pattern: `.sharkconfig.json` is parsed via `json.Unmarshal` into the `Config` struct; unrecognized top-level keys go into `RawData`
- This means adding `"observability": {...}` to `.sharkconfig.json` will not break any existing config loading
- The correct extension is to add an `ObservabilityConfig` struct with a `json:"observability,omitempty"` tag to the `Config` struct

**Current `.sharkconfig.json` structure:**
```json
{
  "color_enabled": true,
  "database": {...},
  "interactive_mode": false,
  "json_output": false,
  "last_sync_time": "...",
  "require_rejection_reason": true,
  "workflow_config": "..."
}
```

No `observability` section exists. Adding it is purely additive.

---

## 2. Patterns and Conventions That Must Be Followed

### 2.1 Global Singleton Pattern (`sync.Once`)

**`internal/cli/db_global.go`** establishes the canonical pattern for global resources:

```go
var (
    globalDB    *repository.DB
    dbInitOnce  sync.Once
    dbInitErr   error
)

func GetDB(ctx context.Context) (*repository.DB, error) {
    dbInitOnce.Do(func() {
        // initialization
    })
    return globalDB, dbInitErr
}

func ResetDB() {
    globalDB = nil
    dbInitOnce = sync.Once{}
    dbInitErr = nil
}
```

The OTel provider global must follow this exact pattern in a new file `internal/cli/observability_global.go`. The `ResetDB()` analog must also be added to `ResetServices()` in `internal/cli/services_global.go`.

### 2.2 Cobra Lifecycle Hooks

**`internal/cli/root.go`** already has the two hooks needed for OTel lifecycle management:

- `PersistentPreRunE` (line ~42): runs before EVERY command — SDK initialization, logger setup
- `PersistentPostRunE` (line ~66): runs after EVERY command, currently calls `CloseDB()` — SDK shutdown/flush must be added here

No new hooks are needed; these existing hooks are the correct injection points.

### 2.3 Service Constructor Injection

All services use constructor injection (`NewTaskService(repo, workflowSvc, ...)`) with no DI framework. Tracing dependencies (a `trace.Tracer`) should be injected the same way — as a parameter to service constructors, defaulting to a no-op tracer when nil.

### 2.4 Context Propagation

Every method in the service and repository layers already accepts `context.Context` as its first parameter. This is documented explicitly in the repository package comment (`internal/repository/task_repository.go` lines 3-22, which lists "Distributed tracing" as a use case). No method signature changes are required to propagate trace context.

### 2.5 CLI Output Separation

**Critical:** Structured logs MUST go to `stderr`. The `cli.OutputJSON()` function writes to `stdout`. Any OTel or slog output that contaminated stdout would break AI agent tooling that parses `--json` output. This is a hard constraint.

### 2.6 Dependency Injection for Tests

The existing test pattern uses `ResetDB()` and `ResetServices()` to tear down global state between tests. OTel global state must participate in the same reset. The no-op provider pattern ensures that tests without explicit OTel setup still compile and run correctly.

---

## 3. Integration Points

### 3.1 SDK Lifecycle (CLI)

| Location | File | Current State | Required Change |
|----------|------|---------------|-----------------|
| SDK init | `internal/cli/root.go:PersistentPreRunE` | Existing hook, only sets verbose | Add OTel provider initialization |
| SDK shutdown | `internal/cli/root.go:PersistentPostRunE` | Existing hook, calls CloseDB() | Add `provider.Shutdown(ctx)` call |
| Global state | `internal/cli/db_global.go` (pattern file) | DB singleton | New file: `internal/cli/observability_global.go` |
| Service reset | `internal/cli/services_global.go:ResetServices()` | Resets service singletons | Add OTel provider reset |

### 3.2 SDK Lifecycle (HTTP Server)

| Location | File | Current State | Required Change |
|----------|------|---------------|-----------------|
| Server init | `cmd/server/main.go` | Minimal stub, `log.Fatal` calls | Initialize OTel provider before `http.ListenAndServe` |
| HTTP middleware | `cmd/server/main.go` | No middleware, bare `http.HandleFunc` | Wrap mux with `otelhttp.NewHandler(mux, "shark-server")` |

### 3.3 Structured Logging Migration

| Package | Files | `log.*` count | `fmt.Fprintf(stderr)` count | Migration Action |
|---------|-------|---------------|----------------------------|-----------------|
| `internal/config/` | manager.go, template_helpers.go, orchestrator_action.go | 8 | 1 | Replace with `slog.Warn/Error` |
| `internal/status/` | derivation.go | 3 | 0 | Replace with `slog.Warn` |
| `internal/services/` | task/feature/epic/entity/display_service.go | 9 | 6 | Replace with `slog.Warn/Error` |
| `internal/cli/` | cli_error.go, helpers.go, root.go, etc. | 0 | 8 | Replace with `slog.Warn/Error` |
| `internal/taskcreation/` | creator.go | 0 | 1 | Replace with `slog.Warn` |
| `internal/init/` | profile_service.go | 0 | 2 | Replace with `slog.Warn` |
| `internal/db/` | db.go | 0 | 1 | Replace with `slog.Warn` |
| `cmd/server/` | main.go | 6 | 0 | Replace with `slog.Info/Fatal` |
| `cmd/` utilities | demo, cleanup, migrate, backfill | ~62 | 0 | Replace with `slog.Info/Warn` |

**Global slog default logger** should be configured once in `PersistentPreRunE` via `slog.SetDefault(logger)` so all packages automatically use structured logging without import changes beyond replacing `log.Printf` with `slog.Warn`/`slog.Info` etc.

### 3.4 Tracing — Service Layer

All service methods already accept `context.Context`. Span injection pattern:

```go
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    ctx, span := s.tracer.Start(ctx, "TaskService.GetTask")
    defer span.End()
    // existing implementation unchanged
}
```

Services that need tracing: `TaskService`, `FeatureService`, `EpicService`, `BugService`, `ChangeCardService`.

### 3.5 Tracing — Repository Layer

All repository methods use `QueryRowContext(ctx, ...)` and `ExecContext(ctx, ...)`. Span injection:

```go
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    ctx, span := otel.Tracer("repository").Start(ctx, "TaskRepository.GetByKey")
    defer span.End()
    // existing query unchanged — ctx already carries the span
}
```

### 3.6 Metrics — Key Measurement Points

| Metric | Location | Type |
|--------|----------|------|
| CLI command duration | `root.go:PersistentPreRunE/PostRunE` | Histogram |
| CLI command invocations | `root.go:PersistentPreRunE` | Counter |
| DB query duration | repository layer | Histogram |
| Task status transitions | `TaskService.AdvanceTaskStatus` | Counter |
| HTTP request duration | `cmd/server/main.go` (via otelhttp) | Histogram (auto) |

### 3.7 Configuration

Extend `internal/config/config.go` with:

```go
type ObservabilityConfig struct {
    Enabled        bool   `json:"enabled"`
    TracingEnabled bool   `json:"tracing_enabled"`
    MetricsEnabled bool   `json:"metrics_enabled"`
    LogLevel       string `json:"log_level"`       // "debug", "info", "warn", "error"
    Exporter       string `json:"exporter"`        // "stdout", "otlp", "noop"
    OTLPEndpoint   string `json:"otlp_endpoint"`   // e.g. "localhost:4317"
    ServiceName    string `json:"service_name"`    // defaults to "shark-task-manager"
}
```

Add to `Config` struct:
```go
Observability ObservabilityConfig `json:"observability,omitempty"`
```

Environment variable overrides (`SHARK_OTEL_ENABLED`, `SHARK_OTEL_EXPORTER`, `OTEL_EXPORTER_OTLP_ENDPOINT`) should be checked in the SDK initialization code in `PersistentPreRunE`.

---

## 4. What Can Be Extended vs. What Needs New Code

### EXTEND (Modify Existing Files)

| File | What to Extend | Why Extension (Not New Code) |
|------|---------------|------------------------------|
| `internal/config/config.go` | Add `ObservabilityConfig` struct and field | `RawData` pattern explicitly designed for additive fields; breaking change would require new file |
| `internal/cli/root.go:PersistentPreRunE` | Add OTel SDK init call | This hook already exists for exactly this purpose; adding a new hook would create ordering ambiguity |
| `internal/cli/root.go:PersistentPostRunE` | Add `provider.Shutdown(ctx)` | Hook already exists, already calls CloseDB() as the shutdown pattern |
| `internal/cli/services_global.go:ResetServices()` | Add OTel provider reset | Function already exists for test teardown; OTel must participate in same reset |
| `internal/services/task_service.go` | Inject `trace.Tracer`, add spans | Constructor injection is the established pattern; new file would duplicate struct definition |
| `internal/services/feature_service.go` | Same as task_service | Same rationale |
| `internal/services/epic_service.go` | Same as task_service | Same rationale |
| `internal/repository/task_repository.go` | Add span creation per method | Context is already passed; file already exists; new file would require interface duplication |
| `cmd/server/main.go` | Add OTel init, otelhttp middleware | Only one server entry point; splitting would add unnecessary indirection |
| All files with `log.Printf` | Replace with `slog.Warn/Info/Error` | Mechanical replacement; new file would not eliminate original log calls |

### NEW CODE (New Files/Packages)

| New File/Package | Reason New Code Is Required |
|------------------|-----------------------------|
| `internal/observability/provider.go` | No existing OTel provider anywhere; must be created. Centralizes SDK lifecycle (TracerProvider, MeterProvider, LoggerProvider, OTLP exporter setup). |
| `internal/observability/logger.go` | No existing slog configuration; must be created. Configures slog handler (JSON to stderr, log level from config). |
| `internal/observability/noop.go` | No existing no-op provider; must be created. Returns no-op providers for test environments and when observability is disabled. |
| `internal/cli/observability_global.go` | Follows db_global.go pattern for global OTel provider singleton. Cannot reuse db_global.go (different concern). |
| `internal/observability/middleware.go` | No existing HTTP middleware anywhere in the project. Wraps `net/http` handler with OTel tracing. |

---

## 5. Technical Risks and Feasibility Assessment

### Risk 1: Test Isolation (HIGH PROBABILITY, MEDIUM IMPACT)
**Description:** OTel SDK initialization involves global state (`otel.SetTracerProvider`, `otel.SetTextMapPropagator`). Tests that execute service or CLI code will inadvertently share OTel state unless the no-op pattern is enforced.

**Mitigation:**
- `internal/observability/noop.go` provides no-op providers that are the default when `ObservabilityConfig.Enabled == false`
- Add OTel provider to `ResetServices()` in `services_global.go` for test cleanup
- Never call `otel.SetTracerProvider` in library code — only in CLI entry point (`PersistentPreRunE`)
- Services should receive `trace.Tracer` via constructor injection (defaults to `trace.NewNoopTracer()` when nil)

**Feasibility:** Fully mitigatable with the no-op pattern.

### Risk 2: stdout Contamination (HIGH PROBABILITY, HIGH IMPACT)
**Description:** Any slog or OTel output to stdout would break the `--json` flag parsing relied on by AI agents and scripts.

**Mitigation:**
- `slog` JSON handler configured with `os.Stderr` explicitly
- OTel stdout exporter also writes to `os.Stderr`
- Code review checklist item: no `log.*` or `slog.*` writes to stdout
- Existing `OutputJSON` already uses stdout — never introduce slog to that code path

**Feasibility:** Fully preventable by enforcing stderr in `internal/observability/logger.go`.

### Risk 3: Import Cycle (LOW PROBABILITY, HIGH IMPACT)
**Description:** `internal/observability/` package importing from `internal/config/` is fine. Risk arises if `internal/cli/` imports `internal/observability/` which imports something that imports `internal/cli/`.

**Mitigation:**
- `internal/observability/` must depend only on `internal/config/` and OTel SDK packages
- CLI packages import observability; observability must never import CLI
- Standard dependency hierarchy: `internal/cli/` → `internal/observability/` → `internal/config/` → nothing internal

**Feasibility:** Preventable by enforcing import hierarchy.

### Risk 4: go.sum / Dependency Size (LOW PROBABILITY, LOW IMPACT)
**Description:** OTel Go SDK is a large dependency tree. Adds ~15-25 transitive dependencies to `go.mod`.

**Mitigation:**
- Use only needed OTel packages: `go.opentelemetry.io/otel`, `otel/trace`, `otel/metric`, `otel/exporters/otlp/otlptrace/otlptracegrpc`, `otel/exporters/stdout/stdouttrace`, `otel/sdk/trace`, `otel/sdk/metric`
- `contrib/instrumentation/net/http/otelhttp` for HTTP middleware
- Build constraints or `//go:build` tags are not needed since the no-op pattern handles disabled mode at runtime

**Feasibility:** Standard Go module management; no unusual risk.

### Risk 5: Performance Impact on CLI Commands (LOW PROBABILITY, LOW IMPACT)
**Description:** CLI commands are short-lived processes. OTel SDK initialization in `PersistentPreRunE` could add latency.

**Mitigation:**
- When `ObservabilityConfig.Enabled == false` (default), skip all initialization; use no-op providers
- When enabled with OTLP, use batch span processor (async export) — no blocking on command exit
- `PersistentPostRunE` calls `Shutdown(ctx)` with a short timeout (e.g., 5 seconds) to flush pending spans

**Feasibility:** Standard OTel best practice; well-documented pattern.

### Overall Feasibility: HIGH
The codebase is well-suited for OTel integration. Context propagation, dependency injection, and lifecycle hooks are already in place. The implementation follows established patterns from `db_global.go` and service constructors. Estimated complexity: **Medium** (not XS/S due to breadth of logging migration, but not L+ because no architectural changes are required).

---

## 6. Recommended Implementation Approach

### Principle: Extend First, New Code Only When Necessary

The implementation should proceed in three phases to maintain stability:

### Phase 1: Foundation (New Package + Config Extension)

1. **Create `internal/observability/` package**
   - `provider.go`: OTel SDK bootstrap (TracerProvider, MeterProvider). Accepts `ObservabilityConfig`. Returns a `Shutdown func(context.Context) error`.
   - `logger.go`: Configures `slog.Default()` with JSON handler writing to `os.Stderr`. Level from `ObservabilityConfig.LogLevel`.
   - `noop.go`: Returns no-op TracerProvider and MeterProvider for disabled/test mode.

2. **Extend `internal/config/config.go`**
   - Add `ObservabilityConfig` struct
   - Add `Observability ObservabilityConfig` field with `json:"observability,omitempty"` tag
   - Default: `Enabled: false` (telemetry off by default)

3. **Create `internal/cli/observability_global.go`**
   - `sync.Once` singleton for the OTel provider
   - `GetObservabilityProvider()` accessor
   - `ResetObservability()` for tests — call from existing `ResetServices()`

### Phase 2: Integration (Lifecycle + Logging Migration)

4. **Extend `internal/cli/root.go`**
   - `PersistentPreRunE`: call `observability.InitProvider(config.Observability)` → stores provider globally
   - `PersistentPostRunE`: call `provider.Shutdown(ctx)` after `CloseDB()`

5. **Migrate `log.*` calls in `internal/` (23 calls)**
   - Mechanical replacement: `log.Printf("Warning: ...")` → `slog.Warn("...", "key", value)`
   - Start with `internal/services/` (highest value for observability)
   - Then `internal/config/`, `internal/status/`, remaining packages

6. **Migrate `fmt.Fprintf(os.Stderr, ...)` diagnostic calls**
   - Replace with appropriate `slog` level calls
   - Do NOT touch `cli.Success/Error/Warning/Info` — these are presentation, not logging

7. **Extend `cmd/server/main.go`**
   - Initialize OTel provider before `http.ListenAndServe`
   - Wrap mux with `otelhttp.NewHandler(mux, "shark-server")`

### Phase 3: Instrumentation (Spans + Metrics)

8. **Service layer spans**
   - Inject `trace.Tracer` into service constructors (default: `trace.NewNoopTracer()`)
   - Add span at entry of key service methods (Create, Get, List, status transitions)
   - Propagate enriched `ctx` to repository calls — spans nest automatically

9. **Repository layer spans**
   - Add `tracer.Start(ctx, "repo.OperationName")` wrapping existing `QueryRowContext`/`ExecContext` calls
   - Use package-level tracer: `otel.Tracer("internal/repository")`

10. **CLI command metrics**
    - In `PersistentPreRunE`: record command start time and command name
    - In `PersistentPostRunE`: record `shark.cli.command.duration` histogram and `shark.cli.command.invocations` counter

11. **Update global service accessors**
    - `GetTaskService()` in `internal/cli/services_global.go`: pass tracer from global OTel provider to service constructor
    - Same for `GetFeatureService()`, `GetEpicService()`, `GetBugService()`, `GetChangeCardService()`

### Implementation Order (Critical Path)

```
Phase 1: internal/observability/ package + config extension
    ↓
Phase 2a: PersistentPreRunE/PostRunE integration
    ↓
Phase 2b: log.* → slog migration (can be parallel per package)
    ↓
Phase 3a: Service layer spans
    ↓
Phase 3b: Repository layer spans
    ↓
Phase 3c: Metrics (CLI command + DB query)
```

Each phase is independently deployable. The no-op default means Phase 1 can ship without any visible behavioral change.

### Key Design Decisions

**1. Disabled by default:** `observability.enabled: false` in config means zero overhead for users who don't configure OTel. The no-op provider is essentially free.

**2. Global `slog.Default()` configuration:** Setting the default slog logger once in `PersistentPreRunE` means all `internal/` packages can migrate from `log.Printf` to `slog.Warn` without any additional wiring. The global slog logger is the correct Go idiom for this.

**3. Constructor injection for tracers, not global `otel.Tracer()`:** Services should receive a `trace.Tracer` in their constructor. This keeps services testable with mock tracers. Repository layer can use `otel.Tracer("package")` since repositories are tested with real DB (not mocked).

**4. Stderr-only for all telemetry output:** Enforced at the `internal/observability/logger.go` level. Never use `os.Stdout` for any log/span/metric output.

**5. No new required runtime dependencies when disabled:** When `Enabled: false`, the OTel SDK is still compiled in (Go doesn't support conditional compilation of this kind cleanly), but no network connections are made, no goroutines spawned, no file I/O performed. The no-op provider handles this.

---

## References

- `internal/cli/db_global.go` — canonical `sync.Once` global pattern to replicate
- `internal/cli/root.go` — `PersistentPreRunE`/`PersistentPostRunE` hooks
- `internal/config/config.go` — `RawData` extensibility pattern
- `internal/cli/services_global.go` — `ResetServices()` teardown pattern
- `internal/repository/task_repository.go` — context propagation documentation (lines 3-22)
- `go.mod` — Go 1.23.4 (supports `log/slog` natively, available since 1.21)
- Epic PRD: `docs/plan/E23-opentelemetry-observability-tracing-metrics-struct/epic.md`
