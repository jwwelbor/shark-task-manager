---
epic_key: E15
title: Service Layer Architecture Refactoring
description: Extract business logic from CLI commands into a proper service/application layer. CLI commands should be thin wrappers (parse args, call service, format output) while services encapsulate all business logic, orchestration, and validation.
---

# Service Layer Architecture Refactoring

**Epic Key**: E15

---

## Overview

This epic transforms Shark's architecture from a fat-controller pattern (where CLI commands contain 40-45% of business logic) to a clean three-layer architecture with dedicated service layer. The refactoring extracts business logic from 43,590 lines of CLI command code into reusable service methods, enabling the HTTP API to share logic with the CLI, improving testability, and eliminating widespread code duplication.

## Epic Components

This epic is documented across multiple interconnected files:

- **[User Personas](./personas.md)** - Developers and AI agents affected by architecture
- **[User Journeys](./user-journeys.md)** - Development workflow scenarios before/after refactoring
- **[Requirements](./requirements.md)** - Functional and non-functional requirements
- **[Success Metrics](./success-metrics.md)** - Measurable outcomes and KPIs
- **[Scope Boundaries](./scope.md)** - In-scope, out-of-scope, and future considerations

---

## Goal

### Problem

The CLI command layer (`internal/cli/commands/`) contains ~40-45% of all business logic across 43,590 lines of code, with the three largest files (`task.go` at 2,671 lines, `feature.go` at 2,090, `epic.go` at 1,739) acting as fat controllers. Commands directly create multiple repositories, orchestrate multi-step operations, implement business rules inline, coordinate transactions, and calculate progress. There is no unified service/application layer between the CLI and the repository. This creates three critical problems:

1. **Business logic is unreusable**: The HTTP API in `cmd/server/` cannot share logic with the CLI, forcing duplication or API/CLI feature parity gaps
2. **Testing requires heavy mocking**: Unit tests must mock entire repository layers rather than testing business logic in isolation
3. **Patterns are duplicated**: Key lookup, error handling, status validation, and progress calculation logic is copy-pasted across 47 command files

### Solution

Introduce a proper service/application layer (`internal/services/`) containing `TaskService`, `FeatureService`, `EpicService`, and supporting services. CLI commands become thin wrappers (200-400 lines) that parse arguments, call service methods, and format output. Repositories become pure data access with no business logic. Existing partial services (`workflow.Service`, `status.CalculationService`, `taskcreation.Creator`) are integrated into the unified service layer.

### Impact

- **CLI command files reduced** from 2,000+ lines to ~200-400 lines each
- **Business logic reusable** across CLI and HTTP API entry points
- **Testability improved**: Services can be unit-tested without CLI/Cobra dependencies
- **Duplication eliminated**: Common patterns centralized in service layer
- **HTTP API parity**: API server can perform all operations the CLI can

---

## Business Value

**Rating**: High

This refactoring is a prerequisite for HTTP API feature parity with the CLI, which blocks integration with AI orchestrators, CI/CD pipelines, and web interfaces. The architecture debt compounds as feature count grows - the current fat-controller pattern makes each new command cost 2-3x more development time due to duplication. Eliminating this debt now reduces future development costs by 40-60% and enables the API-first development model required for Shark's growth.

---

## Current Architecture (Anti-Pattern)

```
CLI Command (2,000+ lines: args + business logic + repo calls + formatting)
    |
    +---> Repository (data access + progress calc + status breakdown)
              |
              +---> Database
```

**Problems:**
- Business logic trapped in command layer
- Repository layer contains business logic (progress calculation, status derivation)
- HTTP API cannot reuse command logic
- Testing requires mocking full repository layer

## Target Architecture (Clean Layers)

```
CLI Command (~200-400 lines: parse args, call service, format output)
    |
    +---> Service Layer (all business logic, orchestration, validation)
              |
              +---> Repository (pure data access)
                        |
                        +---> Database
```

**Benefits:**
- Business logic centralized and reusable
- Repository layer is pure data access
- HTTP API calls same service methods as CLI
- Services unit-testable without Cobra/CLI framework

---

## Quick Reference

**Primary Users**: Shark developers (human and AI), HTTP API consumers, Maintainers

**Key Features**:
- Service interface design for Task/Feature/Epic operations
- Business logic extraction from 47 CLI command files
- Repository cleanup to pure data access
- HTTP API wiring to service layer
- Zero regression test requirement

**Success Criteria**:
- No CLI command exceeds 500 lines (excluding tests)
- All business logic testable without Cobra framework
- HTTP API achieves 100% feature parity with CLI
- Zero behavior regressions in existing tests
- Service layer has >80% test coverage

**Timeline**:
- Phase 1 (Interface Design): Week 1
- Phase 2 (Task/Feature/Epic Service Extraction): Weeks 2-4
- Phase 3 (Repository Cleanup): Week 5
- Phase 4 (CLI Slimming): Week 6
- Phase 5 (HTTP API Wiring): Week 7

---

*Last Updated*: 2026-02-16
