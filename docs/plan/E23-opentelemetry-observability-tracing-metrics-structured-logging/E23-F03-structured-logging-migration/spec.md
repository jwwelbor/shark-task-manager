# E23-F03: Structured Logging Migration — Specification

**Feature**: E23-F03 Structured Logging Migration
**Epic**: E23 OpenTelemetry Observability
**Status**: in_specification
**Date**: 2026-03-22

---

## 1. Overview

This feature performs a mechanical but comprehensive replacement of all ad-hoc `log.*` calls and diagnostic `fmt.Fprintf(os.Stderr, ...)` calls across `internal/` and `cmd/` with their `log/slog` equivalents. It does not change user-facing output (`cli.Success`, `cli.Error`, `cli.OutputJSON`, etc.). After this feature, running shark with observability enabled produces fully structured JSON log lines on stderr; with observability disabled the default handler discards all output, preserving current silent behavior.

See epic PRD and architecture for business context and system-level design decisions.

---

## 2. Requirements

### 2.1 Functional Requirements

**REQ-F-001: Replace all `log.Printf` / `log.Println` calls with `slog.Info` or `slog.Warn`**
- Description: Every `log.Printf(...)` warning/info call in `internal/` and `cmd/` is replaced with the appropriate `slog` level call using structured key-value pairs.
- Priority: Must-Have
- Acceptance Criteria:
  - [ ] `grep -r "log\.Printf\|log\.Println" internal/ cmd/ --include="*.go"` returns zero results (excluding test files and comments)

**REQ-F-002: Replace all `log.Fatal*` calls with `slog.Error` + `os.Exit(1)`**
- Description: `log.Fatalf(...)` and `log.Fatal(...)` calls are replaced with a `slog.Error(msg, "error", err)` call followed by `os.Exit(1)`.
- Priority: Must-Have
- Acceptance Criteria:
  - [ ] `grep -r "log\.Fatal" internal/ cmd/ --include="*.go"` returns zero results (excluding comments)
  - [ ] Each former `log.Fatal*` site calls `os.Exit(1)` after the `slog.Error` call

**REQ-F-003: Replace all `fmt.Fprintf(os.Stderr, ...)` diagnostic calls with `slog.*`**
- Description: Every `fmt.Fprintf(os.Stderr, ...)` that emits warnings or diagnostic information (not user-facing output) is replaced with the appropriate `slog` level call.
- Priority: Must-Have
- Acceptance Criteria:
  - [ ] `grep -r 'fmt\.Fprintf(os\.Stderr' internal/ cmd/ --include="*.go"` returns zero results (excluding user-facing presentation functions)

**REQ-F-004: Preserve semantic intent of each call**
- Description: The slog level chosen must match the severity intent of the original call.
- Priority: Must-Have
- Rules:
  - `WARNING:` / `Warning:` prefix in message → `slog.Warn`
  - `log.Fatal*` (fatal error, process exits) → `slog.Error` + `os.Exit(1)`
  - Error details printed without fatal → `slog.Error` (if error condition) or `slog.Warn` (if recoverable)
  - Informational prints (`log.Println("initialized")`) → `slog.Info`
  - Debug/verbose prints → `slog.Debug`

**REQ-F-005: Use structured key-value fields, not format strings**
- Description: Replace `log.Printf("Failed to get %s: %v", key, err)` with `slog.Warn("Failed to get", "key", key, "error", err)`. Use the conventional `"error"` key for errors.
- Priority: Must-Have
- Acceptance Criteria:
  - [ ] No `fmt.Sprintf(...)` wrapping the message argument of any `slog.*` call added in this migration

**REQ-F-006: No changes to user-facing output functions**
- Description: `cli.Success`, `cli.Error`, `cli.Warning`, `cli.Info`, `cli.OutputJSON`, `cli.OutputTable`, and all display formatting in `internal/cli/commands/` that writes structured output to stdout remain unchanged.
- Priority: Must-Have
- Acceptance Criteria:
  - [ ] Diff of `internal/cli/commands/` shows only changes to `fmt.Fprintf(os.Stderr, ...)` warning calls, not to any stdout-bound presentation logic

**REQ-F-007: Each migrated file adds the `log/slog` import and removes the `log` package import**
- Description: Files that previously imported `"log"` switch to `"log/slog"`. Files that had `fmt.Fprintf(os.Stderr, ...)` calls may also need to remove the `"fmt"` import if it is no longer used (or retain it if still needed for other purposes).
- Priority: Must-Have
- Acceptance Criteria:
  - [ ] `go build ./...` succeeds with no import errors
  - [ ] `make lint` passes with no unused import warnings

### 2.2 Non-Functional Requirements

**REQ-NF-001: Zero behavioral change when observability is disabled**
- Description: When `observability.enabled = false` (the default), `InitLogger` installs a discard handler. All `slog.*` calls therefore produce no output — identical to the current silent behavior.
- Measurement: Manual test: run any shark command without observability config; no new stderr lines appear.

**REQ-NF-002: Build and tests must pass**
- Description: `make fmt && make lint && make test` must pass after all changes.
- Target: Green CI across all existing tests

**REQ-NF-003: No performance regression**
- Description: `slog.*` calls with the discard handler are no-ops at the handler level; no measurable latency added to normal CLI invocations.

### 2.3 Out of Scope

- Adding new log calls or new observability instrumentation (that is E23-F04 and E23-F05)
- Changing `cli.Success` / `cli.Error` / user-facing output — these are presentation, not logging
- Migrating test files (`*_test.go`) — test log calls are acceptable
- `internal/templates/orchestrator_renderer.go:162` comment (`// In production, this would log.Fatalf`) — this is a comment, not a call
- `cmd/server/services.go` lines 19, 37 — these are already comments (`// log.Fatal(...)`)
- `internal/fileops/doc.go:25` — this is a doc comment example, not a call

---

## 3. Files to Migrate

### 3.1 `internal/cli/` — 12 calls

| File | Line | Call | Target |
|------|------|------|--------|
| `internal/cli/commands/epic.go` | 182 | `fmt.Fprintf(os.Stderr, "Failed to list epics: %v\n", err)` | `slog.Error("Failed to list epics", "error", err)` |
| `internal/cli/commands/epic.go` | 245 | `fmt.Fprintf(os.Stderr, "Failed to build epic data: %v\n", err)` | `slog.Error("Failed to build epic data", "error", err)` |
| `internal/cli/commands/epic_helpers.go` | 379 | `fmt.Fprintf(os.Stderr, "Warning: Failed to calculate progress for epic %s: %v\n", epic.Key, err)` | `slog.Warn("Failed to calculate progress for epic", "epic", epic.Key, "error", err)` |
| `internal/cli/commands/epic_helpers.go` | 790 | `fmt.Fprintf(os.Stderr, "Warning: Failed to update progress for feature %s: %v\n", f.Key, recalcErr)` | `slog.Warn("Failed to update progress for feature", "feature", f.Key, "error", recalcErr)` |
| `internal/cli/commands/feature_helpers.go` | 215 | `fmt.Fprintf(os.Stderr, "Warning: Failed to batch fetch status breakdowns: %v\n", err)` | `slog.Warn("Failed to batch fetch status breakdowns", "error", err)` |
| `internal/cli/commands/feature_helpers.go` | 224 | `fmt.Fprintf(os.Stderr, "Warning: Failed to get config path: %v\n", cfgErr)` | `slog.Warn("Failed to get config path", "error", cfgErr)` |
| `internal/cli/commands/feature_helpers.go` | 228 | `fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", cfgErr)` | `slog.Warn("Failed to load config", "error", cfgErr)` |
| `internal/cli/commands/feature_helpers.go` | 750 | `fmt.Fprintf(os.Stderr, "Warning: Failed to get task counts: %v\n", err)` | `slog.Warn("Failed to get task counts", "error", err)` |
| `internal/cli/commands/helpers.go` | 820 | `fmt.Fprintf(os.Stderr, "Details: %v\n", err)` | `slog.Debug("Error details", "error", err)` |
| `internal/cli/root.go` | 58 | `fmt.Fprintf(os.Stderr, "warning: observability init failed: %v\n", err)` | `slog.Warn("observability init failed", "error", err)` |
| `internal/cli/root.go` | 77 | `fmt.Fprintf(os.Stderr, "warning: observability shutdown failed: %v\n", err)` | `slog.Warn("observability shutdown failed", "error", err)` |
| `internal/cli/root.go` | 420 | `fmt.Fprintf(os.Stderr, "Failed to render table: %v\n", err)` | `slog.Error("Failed to render table", "error", err)` |

### 3.2 `internal/config/` — 9 calls

| File | Line | Call | Target |
|------|------|------|--------|
| `internal/config/manager.go` | 58 | `log.Printf("Warning: Invalid last_sync_time format in config: %v", err)` | `slog.Warn("Invalid last_sync_time format in config", "error", err)` |
| `internal/config/orchestrator_action.go` | 182 | `log.Printf("template rendering failed for %s: %v", oa.InstructionTemplate, err)` | `slog.Error("template rendering failed", "template", oa.InstructionTemplate, "error", err)` |
| `internal/config/template_helpers.go` | 330 | `log.Printf("WARNING: Failed to fetch related docs for feature %s: %v", feature.Key, err)` | `slog.Warn("Failed to fetch related docs for feature", "feature", feature.Key, "error", err)` |
| `internal/config/template_helpers.go` | 343 | `log.Printf("WARNING: Failed to fetch related features for %s: %v", feature.Key, err)` | `slog.Warn("Failed to fetch related features", "feature", feature.Key, "error", err)` |
| `internal/config/template_helpers.go` | 441 | `log.Printf("WARNING: Failed to fetch related docs for task %s: %v", task.Key, err)` | `slog.Warn("Failed to fetch related docs for task", "task", task.Key, "error", err)` |
| `internal/config/template_helpers.go` | 450 | `log.Printf("WARNING: Failed to fetch related tasks for task %s: %v", task.Key, err)` | `slog.Warn("Failed to fetch related tasks", "task", task.Key, "error", err)` |
| `internal/config/template_helpers.go` | 516 | `log.Printf("WARNING: Failed to fetch related docs for epic %s: %v", epic.Key, err)` | `slog.Warn("Failed to fetch related docs for epic", "epic", epic.Key, "error", err)` |
| `internal/config/template_helpers.go` | 529 | `log.Printf("WARNING: Failed to fetch related epics for %s: %v", epic.Key, err)` | `slog.Warn("Failed to fetch related epics", "epic", epic.Key, "error", err)` |
| `internal/config/workflow_parser.go` | 393 | `fmt.Fprintf(os.Stderr, "Warning: Failed to load workflow config: %v\n", err)` | `slog.Warn("Failed to load workflow config", "error", err)` |

### 3.3 `internal/services/` — 13 calls

| File | Line | Call | Target |
|------|------|------|--------|
| `internal/services/bug_service.go` | 164 | `fmt.Fprintf(os.Stderr, "warning: failed to write bug file %s: %v\n", filePath, writeErr)` | `slog.Warn("failed to write bug file", "path", filePath, "error", writeErr)` |
| `internal/services/change_card_service.go` | 169 | `fmt.Fprintf(os.Stderr, "warning: failed to write change-card file %s: %v\n", filePath, writeErr)` | `slog.Warn("failed to write change-card file", "path", filePath, "error", writeErr)` |
| `internal/services/change_card_service.go` | 289 | `fmt.Fprintf(os.Stderr, "warning: failed to delete change-card file %s: %v\n", absPath, removeErr)` | `slog.Warn("failed to delete change-card file", "path", absPath, "error", removeErr)` |
| `internal/services/epic_service.go` | 224 | `log.Printf("WARNING: Failed to fetch enrichment data for epic %s: %v", epic.Key, err)` | `slog.Warn("Failed to fetch enrichment data for epic", "epic", epic.Key, "error", err)` |
| `internal/services/feature_service.go` | 238 | `log.Printf("WARNING: Failed to fetch enrichment data for feature %s: %v", feature.Key, err)` | `slog.Warn("Failed to fetch enrichment data for feature", "feature", feature.Key, "error", err)` |
| `internal/services/feature_service.go` | 750 | `log.Printf("warning: auto-reopen of epic %s failed: %v", epic.Key, err)` | `slog.Warn("auto-reopen of epic failed", "epic", epic.Key, "error", err)` |
| `internal/services/task_service.go` | 631 | `log.Printf("WARNING: Failed to fetch enrichment data for task %s: %v", task.Key, err)` | `slog.Warn("Failed to fetch enrichment data for task", "task", task.Key, "error", err)` |
| `internal/services/task_service.go` | 976 | `log.Printf("warning: auto-reopen check for feature %s failed: %v", featureKey, err)` | `slog.Warn("auto-reopen check for feature failed", "feature", featureKey, "error", err)` |
| `internal/services/task_service.go` | 996 | `log.Printf("warning: auto-reopen of feature %s failed: %v", featureKey, err)` | `slog.Warn("auto-reopen of feature failed", "feature", featureKey, "error", err)` |
| `internal/services/entity_service.go` | 58 | `log.Printf("warning: failed to record entity history for %s: %v", entityType, err)` | `slog.Warn("failed to record entity history", "entity_type", entityType, "error", err)` |
| `internal/services/display_service.go` | 449 | `log.Printf("WARNING: Failed to fetch enrichment data for epic %s: %v", epic.Key, err)` | `slog.Warn("Failed to fetch enrichment data for epic", "epic", epic.Key, "error", err)` |
| `internal/services/display_service.go` | 634 | `log.Printf("WARNING: Failed to fetch enrichment data for feature %s: %v", feature.Key, err)` | `slog.Warn("Failed to fetch enrichment data for feature", "feature", feature.Key, "error", err)` |
| `internal/services/display_service.go` | 668 | `log.Printf("WARNING: Failed to fetch enrichment data for task %s: %v", task.Key, err)` | `slog.Warn("Failed to fetch enrichment data for task", "task", task.Key, "error", err)` |

### 3.4 Remaining `internal/` packages — 6 calls

| File | Line | Call | Target |
|------|------|------|--------|
| `internal/status/derivation.go` | 30 | `log.Println("WARN: No workflow config provided to DeriveFeatureStatus, using safe defaults")` | `slog.Warn("No workflow config provided to DeriveFeatureStatus, using safe defaults")` |
| `internal/status/derivation.go` | 46 | `log.Printf("WARN: Status %q not found in workflow config, treating as planning phase", status)` | `slog.Warn("Status not found in workflow config, treating as planning phase", "status", status)` |
| `internal/status/derivation.go` | 64 | `log.Printf("WARN: Unrecognized phase %q for status %q, treating as planning", meta.Phase, status)` | `slog.Warn("Unrecognized phase for status, treating as planning", "phase", meta.Phase, "status", status)` |
| `internal/db/db.go` | 1064 | `fmt.Fprintf(os.Stderr, "Warning: Failed to backup WAL file %s: %v\n", walFile, err)` | `slog.Warn("Failed to backup WAL file", "file", walFile, "error", err)` |
| `internal/init/profile_service.go` | 113 | `fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)` | `slog.Warn("failed to create backup", "error", err)` |
| `internal/init/profile_service.go` | 127 | `fmt.Fprintf(os.Stderr, "Warning: failed to create workflow file backup: %v\n", err)` | `slog.Warn("failed to create workflow file backup", "error", err)` |
| `internal/taskcreation/creator.go` | 308 | `fmt.Fprintf(os.Stderr, "[task-creator] %s\n", msg)` | `slog.Debug("task-creator", "msg", msg)` |

### 3.5 `cmd/server/` — 6 calls

| File | Line | Call | Target |
|------|------|------|--------|
| `cmd/server/main.go` | 15 | `log.Fatal("Failed to initialize database:", err)` | `slog.Error("Failed to initialize database", "error", err); os.Exit(1)` |
| `cmd/server/main.go` | 19 | `log.Println("Database initialized successfully")` | `slog.Info("Database initialized successfully")` |
| `cmd/server/main.go` | 23 | `log.Fatal("Database integrity check failed:", err)` | `slog.Error("Database integrity check failed", "error", err); os.Exit(1)` |
| `cmd/server/main.go` | 25 | `log.Println("Database integrity check passed")` | `slog.Info("Database integrity check passed")` |
| `cmd/server/main.go` | 46 | `log.Printf("Starting server on port %s", port)` | `slog.Info("Starting server", "port", port)` |
| `cmd/server/main.go` | 48 | `log.Fatal("Server failed to start:", err)` | `slog.Error("Server failed to start", "error", err); os.Exit(1)` |

### 3.6 `cmd/` utility binaries — 62 calls

These are in utility/maintenance binaries: `cmd/backfill-slugs`, `cmd/cleanup`, `cmd/create-epic`, `cmd/demo`, `cmd/migrate`, `cmd/migrate-exec-order`, `cmd/test-backfill`, `cmd/test-db`.

Migration rules are identical:
- `log.Fatalf(msg, args...)` → `slog.Error(msg, "error", err); os.Exit(1)` (or other structured args)
- `log.Printf(msg, args...)` → `slog.Warn(msg, ...)` or `slog.Info(msg, ...)` depending on severity
- `log.Fatal(msg, err)` → `slog.Error(msg, "error", err); os.Exit(1)`

---

## 4. Architecture

### 4.1 Pattern

All migrated files switch from `"log"` to `"log/slog"`. The global slog default logger is configured by `observability.InitLogger(cfg)` which is called during cobra's `PersistentPreRunE` (already implemented in E23-F02). For the `cmd/` utility binaries that do not go through cobra, `slog.*` calls will use the default logger (which before `InitLogger` is called is the standard `slog.Default()` — text format on stderr). This is acceptable since those are maintenance tools.

### 4.2 Import Changes Per File

For each file in `internal/` and `cmd/`:

**Files using `log.*` only:** Replace `"log"` import with `"log/slog"`.

**Files using `fmt.Fprintf(os.Stderr, ...)` only:** Remove the `os.Stderr`-bound usage. If `"os"` import is no longer needed (no `os.Exit`, `os.Args`, etc.) remove it; otherwise retain. If `"fmt"` is no longer needed, remove it.

**Files using both:** Replace `"log"` with `"log/slog"`, apply same `"fmt"`/`"os"` cleanup as above.

### 4.3 Conversion Rules (canonical)

| Original | Replacement |
|----------|-------------|
| `log.Printf("WARNING: X for %s: %v", k, err)` | `slog.Warn("X", "key", k, "error", err)` |
| `log.Printf("X for %s: %v", k, err)` | `slog.Info("X", "key", k, "error", err)` (if info) or `slog.Warn(...)` (if error condition) |
| `log.Println("WARN: X")` | `slog.Warn("X")` |
| `log.Println("X")` | `slog.Info("X")` |
| `log.Fatalf("X: %v", err)` | `slog.Error("X", "error", err)` then `os.Exit(1)` on next line |
| `log.Fatal("X:", err)` | `slog.Error("X", "error", err)` then `os.Exit(1)` on next line |
| `fmt.Fprintf(os.Stderr, "Warning: X %s: %v\n", k, err)` | `slog.Warn("X", "key", k, "error", err)` |
| `fmt.Fprintf(os.Stderr, "X: %v\n", err)` | `slog.Error("X", "error", err)` (if fatal-adjacent) or `slog.Warn("X", "error", err)` |
| `fmt.Fprintf(os.Stderr, "[tag] %s\n", msg)` | `slog.Debug("tag", "msg", msg)` |

### 4.4 No New Files Required

This migration touches existing files only. No new packages, interfaces, or types are introduced.

### 4.5 Integration with Existing Code

- `internal/observability/logger.go` — already implemented in E23-F01; this feature is a consumer, not a modifier
- `internal/cli/root.go` PersistentPreRunE — already calls `observability.InitLogger(cfg)` in E23-F02; root.go itself has 3 `fmt.Fprintf` calls that also get migrated here
- `cmd/` binaries — these do not call `InitLogger`; their `slog.*` calls use the stdlib default logger (text on stderr). This is acceptable for maintenance tools.

---

## 5. Acceptance Criteria (Feature-Level)

**Scenario 1: No log.* calls remain**
- Given: All source files under `internal/` and `cmd/`
- When: `grep -r "log\.Print\|log\.Fatal\|log\.Println" internal/ cmd/ --include="*.go"` is run (excluding comments and test files)
- Then: Zero matches returned

**Scenario 2: No fmt.Fprintf(os.Stderr diagnostic calls remain**
- Given: All source files under `internal/` and `cmd/`
- When: `grep -r 'fmt\.Fprintf(os\.Stderr' internal/ cmd/ --include="*.go"` is run
- Then: Zero matches returned

**Scenario 3: Build passes**
- Given: Migration is complete
- When: `make fmt && make lint && make build` is run
- Then: All pass with no errors

**Scenario 4: Existing tests pass**
- Given: Migration is complete
- When: `make test` is run
- Then: All tests pass (no regressions from import changes or behavioral shifts)

**Scenario 5: Silent by default**
- Given: No `.sharkconfig.json` with `observability.enabled: true`
- When: Any shark command is run
- Then: No new lines on stderr (slog discard handler active)

---

## 6. Dependencies

- E23-F01 (Observability Foundation Package) — must be merged; provides `internal/observability/logger.go` with `InitLogger`
- E23-F02 (CLI Lifecycle Integration) — must be merged; `root.go` already calls `InitLogger` so migrated slog calls in root.go will route correctly
