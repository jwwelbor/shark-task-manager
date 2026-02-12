---
epic_key: E15
title: Service Layer Architecture Refactoring
description: Extract business logic from CLI commands into a proper service/application layer. CLI commands should be thin wrappers (parse args, call service, format output) while services encapsulate all business logic, orchestration, and validation.
---

# Service Layer Architecture Refactoring

**Epic Key**: E15

---

## Goal

### Problem

The CLI command layer (`internal/cli/commands/`) contains ~40-45% of all business logic, with the three largest files (`task.go` at 2,671 lines, `feature.go` at 2,090, `epic.go` at 1,739) acting as fat controllers. Commands directly create multiple repositories, orchestrate multi-step operations, implement business rules inline, coordinate transactions, and calculate progress. There is no unified service/application layer between the CLI and the repository. This makes business logic unreusable (the HTTP API in `cmd/server/` cannot share it), hard to test (requires mocking entire repository layers), and duplicated across commands.

### Solution

Introduce a proper service/application layer (`internal/services/`) containing `TaskService`, `FeatureService`, `EpicService`, and supporting services. CLI commands become thin wrappers that parse arguments, call service methods, and format output. Repositories become pure data access. Existing partial services (`workflow.Service`, `status.CalculationService`, `taskcreation.Creator`) are integrated into the unified service layer.

### Impact

- CLI command files reduced from 2,000+ lines to ~200-400 lines each
- Business logic reusable across CLI and HTTP API entry points
- Testability improved: services can be unit-tested without CLI/Cobra dependencies
- Duplication eliminated: common patterns (key lookup, error handling, status validation) centralized

---

## Business Value

**Rating**: High

This refactoring enables the HTTP API server to share business logic with the CLI, eliminates duplicated patterns across 47 command files, and makes the codebase significantly more maintainable. As feature count grows, the current fat-controller pattern becomes increasingly costly to develop against and reason about.

---

## Current Architecture

```
CLI Command (2,000+ lines: args + business logic + repo calls + formatting)
    |
    +---> Repository (data access + progress calc + status breakdown)
              |
              +---> Database
```

## Target Architecture

```
CLI Command (~200-400 lines: parse args, call service, format output)
    |
    +---> Service Layer (all business logic, orchestration, validation)
              |
              +---> Repository (pure data access)
                        |
                        +---> Database
```

---

## Key Features

- **F01 - Define Service Interfaces**: Design the `TaskService`, `FeatureService`, `EpicService` interfaces based on current command logic. Identify all operations, inputs, and outputs.
- **F02 - Extract TaskService**: Extract task business logic from `task.go`, `task_deps.go`, `task_next_status.go`, etc. into `internal/services/task_service.go`. Covers lifecycle (start, complete, approve, block, unblock), dependency checking, filtering, and creation orchestration.
- **F03 - Extract FeatureService**: Extract feature business logic from `feature.go` into `internal/services/feature_service.go`. Covers CRUD, progress calculation, status rollups, health indicators, and action items.
- **F04 - Extract EpicService**: Extract epic business logic from `epic.go` into `internal/services/epic_service.go`. Covers CRUD, feature rollups, impediment tracking, and progress aggregation.
- **F05 - Extract QueryService**: Centralize complex filtering/querying logic used across multiple commands into a shared query service.
- **F06 - Slim Down CLI Commands**: Refactor CLI commands to be thin wrappers delegating to services. Verify all existing tests still pass.
- **F07 - Clean Up Repositories**: Remove business logic from repositories (progress calculation, status derivation). Repositories become pure data access.
- **F08 - Wire HTTP API to Services**: Connect the existing HTTP API server (`cmd/server/`) to the new service layer, enabling full API parity with CLI.

---

## Success Criteria

- No CLI command file exceeds 500 lines (excluding tests)
- All business logic is testable without Cobra/CLI framework
- HTTP API server can perform all operations the CLI can
- Zero behavior regressions: all existing tests pass
- Service interfaces are well-defined and documented

---

## Scope Boundaries

**In Scope:**
- Extracting business logic from CLI commands into services
- Cleaning up repository layer to pure data access
- Integrating existing partial services (workflow, status, taskcreation)
- Wiring HTTP API to service layer

**Out of Scope:**
- Adding new features or commands
- Changing database schema
- Changing CLI interface or flags
- Performance optimization (beyond what naturally comes from better architecture)

---

## Risks

- Large refactoring surface area (~39K lines in commands directory)
- Must maintain backward compatibility for all CLI commands
- Incremental approach needed: can't refactor everything at once
- Test coverage must be maintained or improved throughout

## Approach

Incremental, feature-by-feature extraction. Each feature can be developed and merged independently. Start with interface design (F01), then extract one entity at a time (F02-F04), then clean up (F05-F07), then wire the API (F08).

---

## Future Considerations

### Configurable File Naming (Epic/Feature/Task files)

Currently hardcoded:
- Epic files: `epic.md` (in `discovery/folder_scanner.go`, creation commands)
- Feature files: `feature.md` (in `feature.go:1576`, discovery)
- Task key format: `T-E{XX}-F{XX}-{NNN}.md` (in `taskcreation/keygen.go`, regex parsing)

**Epic/Feature filenames** are trivially configurable — only 2-3 references each. Could add `"epic_filename"` and `"feature_filename"` to `.sharkconfig.json` and read at creation + discovery time.

**Task key format** is deeply embedded — key generation, parsing, regex matching, and the entire CLI key normalization layer. Changing this would be a significant refactor. Recommend leaving hardcoded unless there's strong demand.

Consider implementing configurable epic/feature filenames during F06 (Slim Down CLI Commands) when the init/config layer is being reworked. The `internal/patterns` package already has `GenerationFormat` templates that could be wired up for this purpose.

---

*Last Updated*: 2026-02-08
