---
feature_key: E07-F37-split-config-package-into-focused-sub-packages
epic_key: E07
title: Split config package into focused sub-packages
description: Refactor the monolithic internal/config package (3,365 production LOC, 28 files) into focused sub-packages for better navigation, maintainability, and faster iteration.
complexity_tier: STANDARD
complexity_score: 11/27
migrated_from: CC-007
precedent: E07-F36
---

# Split config package into focused sub-packages

**Feature Key**: E07-F37-split-config-package-into-focused-sub-packages
**Complexity**: STANDARD (11/27)
**Migrated from**: CC-007

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem

The `internal/config` package is a flat, monolithic package containing 3,365 lines of production code across 11 files with 5 distinct domain responsibilities. Developers working on orchestrator actions must navigate through unrelated workflow parsing code. The package serves as core infrastructure imported by 80+ files across CLI, services, workflow, and status packages. This tight coupling makes it difficult to reason about, test in isolation, or modify without risk of unintended side effects.

### Solution

Split `internal/config` into focused sub-packages along its 5 natural domain boundaries, following the proven phased approach established by E07-F36 (repository package split). Use type aliases in a backward-compatibility shim (`aliases.go`) to preserve the existing 52+ exported APIs without requiring changes to all 80+ downstream import sites.

### Impact

- Improved developer navigation: each sub-package has a single responsibility
- Faster iteration: changes to orchestrator actions don't require understanding workflow parsing
- Better test isolation: sub-packages can be tested independently
- Reduced coupling: explicit dependency boundaries between config domains
- Zero breaking changes: type alias shim preserves all existing imports

---

## Current State Analysis

### Package Structure (Flat - 11 production files, 3,365 LOC)

| File | LOC | Domain |
|------|-----|--------|
| `template_helpers.go` | 731 | Template placeholder parsing & enrichment |
| `workflow_parser.go` | 575 | Workflow file loading, multi-level parsing |
| `workflow_validator.go` | 404 | Workflow validation logic |
| `workflow_schema.go` | 290 | WorkflowConfig struct, StatusMetadata |
| `orchestrator_action.go` | 259 | OrchestratorAction struct, validation |
| `action_service.go` | 200 | ActionService interface, implementations |
| `manager.go` | 196 | Config file I/O, last_sync_time |
| `workflow_default.go` | 188 | Default workflow profiles |
| `config.go` | 157 | Config struct, status metadata API |
| `workflow_multilevel.go` | 64 | Multi-level workflow support |
| `validation_error.go` | 28 | Error types |

### Test Coverage: 7,908 LOC (2.3:1 test-to-code ratio)

### External Coupling: 80+ files import `internal/config`

### Public API: 22 exported types, 30+ exported functions

---

## Target Sub-Package Structure

```
internal/config/
├── config.go              # Root: Config struct, Manager, core I/O
├── aliases.go             # Backward-compat type aliases & re-exports
├── workflow/              # WorkflowConfig, StatusMetadata, parsing, validation, defaults
│   ├── schema.go
│   ├── parser.go
│   ├── validator.go
│   ├── defaults.go
│   └── multilevel.go
├── action/                # OrchestratorAction, ActionService, PopulatedAction
│   ├── orchestrator.go
│   └── service.go
├── template/              # Entity placeholder helpers, enrichment interfaces
│   └── helpers.go
└── validation/            # ValidationError, shared validation types
    └── error.go
```

---

## Implementation Phases

### Phase 1: Foundation (Task 001)
Extract leaf-node packages with minimal coupling: `validation/` and `template/`.
- `validation_error.go` (28 LOC) has no internal dependencies
- `template_helpers.go` (731 LOC) depends on `models` only

### Phase 2: Standalone (Task 002)
Extract `action/` sub-package: `orchestrator_action.go` + `action_service.go` (459 LOC).
These depend on workflow schema types but can use interfaces.

### Phase 3: Coupling Analysis (Task 003)
Resolve the `workflow_schema` <-> `workflow_parser` <-> `workflow_validator` three-way coupling before splitting. Define interfaces where needed.

### Phase 4: Core Split (Task 004)
Extract `workflow/` sub-package (1,521 LOC). Create `aliases.go` shim for backward compatibility across all 80+ import sites.

### Phase 5: Cleanup (Task 005)
Update direct imports where beneficial, remove unnecessary aliases, verify all tests pass.

---

## Precedent: E07-F36 (Repository Package Split)

| Factor | E07-F36 | E07-F37 (this) |
|--------|---------|----------------|
| Production LOC | 9,714 | 3,365 |
| Total LOC | 30,867 | ~11,300 |
| Production files | 25 | 11 |
| Import sites | 71 | 80+ |
| Sub-packages created | 15 | 4-5 |
| Complexity tier | COMPLEX (18/27) | STANDARD (11/27) |
| Pattern novelty | First-of-kind | Follows proven pattern |
| Estimated effort | ~0.6-0.75x of E07-F36 |

Key mitigation from E07-F36 reused here: **type alias shim** for zero-breakage backward compatibility.

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Sub-packages compile and tests pass**
- **Given** the config package has been split into sub-packages
- **When** `make fmt && make lint && make test` is run
- **Then** all checks pass with zero failures

**Scenario 2: Backward compatibility preserved**
- **Given** 80+ files import `internal/config`
- **When** no import paths are changed in consuming packages
- **Then** all existing code compiles without modification via type aliases

**Scenario 3: Each sub-package is independently testable**
- **Given** each sub-package has its own test files
- **When** `go test ./internal/config/workflow/...` (etc.) is run
- **Then** tests pass in isolation without requiring the full config package

---

## Out of Scope

1. **Changing the public API** — all 52+ exports are preserved; this is a pure reorganization
2. **Refactoring consuming packages** — downstream code uses aliases, not updated imports
3. **Adding new functionality** — no new features, just structural improvement
4. **Config file format changes** — `.sharkconfig.json` format is unchanged

---

## Dependencies

- **E07-F36 pattern**: Type alias shim approach (completed, proven)
- **No external dependencies**: Pure internal refactoring

---

## Quality Gates

Each phase must pass before proceeding to the next:
```bash
make fmt && make lint && make test
```

---

*Last Updated*: 2026-03-24
