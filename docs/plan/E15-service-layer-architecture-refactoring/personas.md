# User Personas

**Epic**: [Service Layer Architecture Refactoring](./epic.md)

---

## Overview

This epic primarily impacts **Shark developers** (both human and AI agents who contribute to Shark's codebase) and secondarily impacts **HTTP API consumers** who will benefit from improved feature parity. The refactoring is an internal architecture improvement that enables better maintainability and API capabilities.

---

## Primary Personas

### Persona 1: Shark Core Developer (Human)

**Reference**: Composite of contributors to Shark repository

**Profile**:
- **Role/Title**: Software engineer contributing features to Shark
- **Experience Level**: Intermediate to senior Go developers
- **Key Characteristics**:
  - Reads existing CLI command code to understand how to add new features
  - Struggles with 2,000+ line command files that mix concerns
  - Copies patterns from existing commands, propagating duplication
  - Needs to write unit tests that require heavy repository mocking
  - Frustrated by difficulty understanding business logic flow

**Goals Related to This Epic**:
1. Add new CLI commands without copying 500+ lines of boilerplate
2. Understand business logic without parsing through CLI flag handling code
3. Write unit tests for business logic without mocking entire repositories
4. Reuse existing business logic patterns across features

**Pain Points This Epic Addresses**:
- **Fat controllers**: Commands like `task.go` (2,671 lines) mix argument parsing, business logic, repository orchestration, and output formatting
- **Duplication**: Key lookup, error handling, status validation duplicated across 47 files
- **Poor testability**: Must mock repositories, workflow services, and file writers to test simple business rules
- **Hidden logic**: Cannot find where specific business rules live (scattered across commands)

**Success Looks Like**:
Developer adds a new "task archive" command by creating a 150-line CLI wrapper that calls `TaskService.Archive()` method. The service method contains all business logic, is unit-tested independently, and is automatically available to the HTTP API. Developer spends 2 hours instead of 8 hours copying patterns.

---

### Persona 2: AI Code Agent (Cursor/Claude)

**Reference**: AI coding assistants working on Shark codebase

**Profile**:
- **Role/Title**: AI agent assisting with Shark development
- **Experience Level**: Expert pattern matching, struggles with large context windows
- **Key Characteristics**:
  - Analyzes existing code to suggest implementations
  - Cannot effectively reason about 2,000+ line files (context overflow)
  - Copies patterns without understanding if duplication is necessary
  - Needs clear separation of concerns to suggest correct changes

**Goals Related to This Epic**:
1. Suggest correct patterns for new features without context overflow
2. Identify which file contains specific business logic
3. Refactor business logic without breaking CLI/API integration
4. Generate accurate tests for isolated business logic

**Pain Points This Epic Addresses**:
- **Context overflow**: Cannot load full 2,671-line `task.go` into context with related files
- **Mixed concerns**: Suggests changes to argument parsing when business logic is requested
- **Unclear boundaries**: Cannot determine if code should live in command, service, or repository
- **Test generation**: Generates CLI integration tests when unit tests are requested

**Success Looks Like**:
AI agent is asked "How do I add validation that tasks in 'completed' status cannot be deleted?" Agent responds with a 20-line service method change in `TaskService.Delete()`, not a 200-line CLI command modification. Generated tests are focused on business logic, not CLI framework.

---

### Persona 3: HTTP API Consumer (Integration Developer)

**Reference**: Developers building tools that integrate with Shark via HTTP API

**Profile**:
- **Role/Title**: Backend developer integrating Shark with CI/CD pipelines or dashboards
- **Experience Level**: Intermediate, familiar with REST APIs
- **Key Characteristics**:
  - Expects HTTP API to have same capabilities as CLI
  - Discovers features are CLI-only due to business logic in commands
  - Frustrated by API/CLI feature parity gaps
  - Needs consistent behavior between API and CLI

**Goals Related to This Epic**:
1. Use HTTP API for all operations available in CLI
2. Receive consistent validation errors regardless of entry point (CLI vs API)
3. Integrate Shark with CI/CD without shelling out to CLI commands
4. Build web dashboards that replicate CLI functionality

**Pain Points This Epic Addresses**:
- **Feature parity gaps**: CLI has features (dependency checking, health calculations) not available in API
- **Behavior inconsistencies**: API and CLI implement same operation differently
- **Forced CLI usage**: Must shell out to CLI commands because API is incomplete
- **Unpredictable errors**: Validation differs between API and CLI

**Success Looks Like**:
API consumer builds a web dashboard that calls `/api/tasks/next?agent=backend` and receives identical results to `shark task next --agent=backend`. Dashboard can transition task statuses, check dependencies, and calculate feature health without any CLI commands. API documentation guarantees feature parity.

---

### Persona 4: Shark Maintainer

**Reference**: Primary maintainer reviewing PRs and managing architecture

**Profile**:
- **Role/Title**: Senior engineer responsible for Shark's long-term maintainability
- **Experience Level**: Expert in Go, architecture patterns, and technical debt management
- **Key Characteristics**:
  - Reviews contributions for architecture compliance
  - Rejects PRs that add to duplication problem
  - Struggles to guide contributors on "where code should live"
  - Needs architecture that is self-documenting and enforces good patterns

**Goals Related to This Epic**:
1. Enforce separation of concerns through architecture, not code review comments
2. Reduce PR review time by 50% (less time explaining architecture)
3. Enable contributors to add features without architectural guidance
4. Reduce regression risk from future changes

**Pain Points This Epic Addresses**:
- **Unclear boundaries**: Contributors don't know if code belongs in command, repository, or new layer
- **Review fatigue**: Same architectural feedback given on every PR ("don't put business logic in commands")
- **Regression risk**: Cannot refactor commands without risk of breaking 47 dependent files
- **Onboarding cost**: New contributors take 2-3 PRs to learn "where things go"

**Success Looks Like**:
Maintainer reviews PR adding new feature. Contributor correctly placed business logic in `FeatureService`, CLI wrapper is clean, tests are isolated. Review focuses on feature correctness, not architecture. PR merged in 1 review cycle instead of 3. Future refactoring is isolated to service layer.

---

## Secondary Personas

### Test Engineer
Needs to write comprehensive test suites for business logic. Benefits from service layer testability (mock repositories, not CLI framework). Achieves >80% code coverage with unit tests instead of integration tests.

### DevOps Engineer
Needs to script Shark operations in CI/CD pipelines. Benefits from HTTP API feature parity. Can call API endpoints instead of shelling out to CLI commands.

---

## Persona Validation Notes

These personas are derived from:
- **Shark repository commit history**: Shows duplication patterns and PR review comments about architecture
- **Existing service implementations**: `EpicService`, `FeatureService` demonstrate the target pattern
- **Architecture documentation**: `.claude/rules/architecture.md` shows current state vs. target state
- **HTTP API gaps**: `cmd/server/` has incomplete feature set compared to CLI

Confidence level: **High** for developer personas (directly observed in codebase), **Medium** for API consumer (inferred from incomplete API implementation).

---

*See also*: [User Journeys](./user-journeys.md)
