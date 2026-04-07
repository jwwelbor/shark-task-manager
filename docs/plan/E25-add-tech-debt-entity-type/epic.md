---
epic_key: E25
title: Add Tech-Debt Entity Type
description: Introduce a standalone Tech-Debt entity type (TD-###) to Shark Task Manager for tracking code quality issues, architectural improvements, and maintenance work outside the epic/feature/task hierarchy.
---

# Add Tech-Debt Entity Type

**Epic Key**: E25

---

## 1. Problem Statement and Business Justification

Development teams accumulate technical debt continuously -- outdated dependencies, suboptimal data access patterns, missing test coverage, hardcoded configuration, and architectural shortcuts taken under deadline pressure. Currently, Shark Task Manager has no dedicated mechanism to capture, prioritize, and track resolution of these items. Teams are forced to either (a) create tasks under unrelated features, obscuring the true scope of debt, (b) use bugs for items that are not defects, polluting bug severity metrics, or (c) track debt informally in external documents where items are forgotten.

The consequence is that tech debt accumulates invisibly. Without a first-class entity type, teams cannot answer basic questions: How much debt exists? What is the highest-priority item? How much debt was resolved this quarter? Which areas of the codebase carry the most risk?

Introducing a dedicated Tech-Debt entity type (TD-###) provides a structured, queryable, and workflow-managed mechanism to capture debt items, classify them by category and severity, and track them through resolution. This directly mirrors the pattern established by Bugs (B###) and Change-Cards (CC-###) -- standalone entities with their own key format, workflow, and CLI commands -- ensuring consistency in the Shark entity model.

Additionally, a longstanding usability gap exists across all entity creation commands: when a user creates any entity, the CLI does not display the file path where the entity's markdown file was written. This forces users to guess or manually look up the file location. Resolving this as part of this epic provides immediate value across all entity types.

---

## 2. Goals and Success Criteria

### Goals

- Provide a first-class entity type for tracking technical debt separate from bugs, tasks, and change requests.
- Follow the established entity patterns (Bug, Change-Card) for model, repository, service, CLI commands, and workflow integration.
- Improve entity creation UX by displaying file paths on creation for all entity types.

### Success Criteria

| # | Criterion | Measurement |
|---|-----------|-------------|
| SC-1 | Tech-debt entities can be created, read, updated, deleted, and listed via `shark td` CLI commands | All CRUD operations succeed without error; `shark td list`, `shark td get TD-001`, `shark td create`, `shark td update`, `shark td delete` function correctly |
| SC-2 | Tech-debt entities have a dedicated workflow with status transitions managed by `shark status advance TD-001` and `shark status set TD-001 <status>` | Status advance and set commands work; invalid transitions are rejected |
| SC-3 | Tech-debt entities support notes and context via `shark td note add` and `shark td context set` | Notes and context fields persist and are retrievable |
| SC-4 | Tech-debt key format TD-### is recognized by auto-detection in core commands (`shark get TD-001`, `shark status TD-001`, `shark delete TD-001`) | Core commands correctly route TD-### keys to tech-debt operations |
| SC-5 | Tech-debt entities appear in `shark search` results and `shark analytics` output | Search returns matching tech-debt items; analytics includes tech-debt counts |
| SC-6 | All entity creation commands (`shark create epic`, `shark create feature`, `shark create task`, `shark td create`, `shark bug create`, `shark change create`) display the file path of the created entity file | Every creation command outputs the absolute or project-relative file path |
| SC-7 | Tech-debt database table and migrations are applied without data loss to existing databases | Migration runs idempotently; existing epic/feature/task/bug/change-card data is unchanged after migration |
| SC-8 | Tech-debt entities support JSON output via `--json` and `--field` flags consistent with all other entity types | `shark td get TD-001 --json` returns valid JSON; `--field status` extracts a single field |

---

## 3. Scope

### In Scope

- **Tech-Debt model** (`internal/models/tech_debt.go`): struct with BaseEntity, status, category (e.g., code-quality, architecture, dependency, testing, performance, documentation), severity (critical, high, medium, low), optional linked entity fields, and effort estimate field.
- **Tech-Debt key format**: TD-### (TD followed by exactly 3 digits, e.g., TD-001, TD-042). Key validation regex: `^TD-\d{3}$`.
- **Database table**: `tech_debts` table with schema following the bug/change-card pattern. Migration added to `internal/db/db.go` with schema version bump.
- **Repository**: `internal/repository/tech_debt_repository.go` with full CRUD, list with filters, status update, and key-based lookup.
- **Service**: `internal/services/tech_debt_service.go` with workflow-aware status transitions, validation, and CRUD orchestration.
- **CLI commands**: `shark td` command group with subcommands: create, get, list, update, delete, triage, note, notes, context (mirroring `shark bug` command structure).
- **Workflow integration**: Tech-debt entities participate in the configured workflow profile (basic or advanced). Status metadata, flow, and agent routing entries added to workflow templates.
- **Core command auto-detection**: `shark get TD-001`, `shark status TD-001`, `shark delete TD-001`, `shark view TD-001`, `shark history TD-001`, `shark update TD-001` all route to tech-debt operations.
- **Entity type registration**: Add `EntityTypeTechDebt` to `ValidEntityTypes` map, entity note support, entity history support, and entity relationship support.
- **File path display on entity create**: Modify all entity creation commands to print the file path of the created markdown file upon successful creation.
- **Search integration**: Tech-debt entities included in `shark search` results.
- **Template**: Markdown template for tech-debt entity files.

### Out of Scope

- **Tech-debt auto-detection or scanning**: No automated code analysis to discover tech debt. Items are manually created.
- **Tech-debt metrics dashboard**: No dedicated analytics view beyond inclusion in existing `shark analytics`. A dedicated dashboard can be a future epic.
- **Tech-debt to task conversion/promotion**: No `shark td promote` command to convert a tech-debt item into a task. This can be added later, similar to `shark idea promote`.
- **Bulk import of tech-debt items**: No CSV/JSON import mechanism.
- **HTTP API endpoints for tech-debt**: HTTP API handlers will not be added in this epic. The service layer supports future HTTP integration but handlers are deferred.
- **Inter-entity dependency tracking for tech-debt**: Tech-debt items will not participate in the task dependency graph (`shark task deps`). They can be linked via entity relationships but do not block/unblock other entities.
- **Retroactive file path display for existing entities**: The file path display feature applies only to creation commands, not to `get` or `list` commands (those already show file paths in their output).

---

## 4. Constraints and Assumptions

### Constraints

- **Pattern consistency**: The tech-debt entity must follow the same architectural patterns as Bug and Change-Card entities. This means: BaseEntity embedding, Entity interface implementation, dedicated repository, dedicated service, CLI command group, workflow-level configuration, and entity type registration.
- **Migration safety**: The database migration must be additive (CREATE TABLE, not ALTER existing tables) and idempotent. It must bump `CurrentSchemaVersion` in `internal/db/db.go`. Existing data must be untouched.
- **Key format uniqueness**: TD-### must not collide with existing key patterns (E##, E##-F##, E##-F##-###, B###, CC-###). The `TD-` prefix is distinct from all existing prefixes.
- **Backward compatibility**: Adding the tech-debt entity must not break any existing CLI commands, workflows, or database operations. All existing tests must continue to pass.
- **Quality gate**: All new Go code must pass `make fmt && make lint && make test` before the epic is considered complete.

### Assumptions

- The existing BaseEntity pattern and Entity interface provide sufficient shared behavior for tech-debt. No changes to the base abstractions are needed.
- The workflow configuration system (`.sharkconfig.json`) already supports adding new entity-level workflow entries, as demonstrated by Bug and Change-Card.
- The `internal/fileops` package handles file creation for all entity types and can be reused for tech-debt without modification.
- Tech-debt categories (code-quality, architecture, dependency, testing, performance, documentation) are sufficient for initial release. Categories can be extended via configuration in a future iteration.
- The 3-digit key format (TD-001 through TD-999) provides sufficient capacity for any single project. If a project exceeds 999 tech-debt items, key format expansion would be a separate change.

---

## 5. Stakeholder Impact

### Development Teams (Primary Users)

- **Benefit**: A structured way to capture, prioritize, and track technical debt resolution. Teams gain visibility into debt accumulation and resolution velocity.
- **Impact**: Teams must learn one new command group (`shark td`) and one new key format (`TD-###`). The command structure mirrors existing `shark bug` commands, minimizing learning curve.
- **Workflow change**: Teams can now assign tech-debt items to sprints, track status through the workflow, and report on debt resolution as a measurable output.

### Tech Leads and Architects

- **Benefit**: Ability to query tech-debt by category (e.g., `shark td list --category=architecture`) to assess risk areas and plan remediation.
- **Impact**: Tech leads can use `shark analytics` to include tech-debt in project health reporting.

### Product Owners

- **Benefit**: Visibility into the volume and severity of tech debt, enabling informed trade-off decisions between feature work and debt reduction.
- **Impact**: No direct workflow change. Tech-debt entities are visible in project dashboards and analytics alongside other entity types.

### AI Development Agents

- **Benefit**: Agents can create and manage tech-debt items programmatically via `shark td create --json` and advance status via `shark status advance TD-001 --json`.
- **Impact**: Agents gain a new entity type to track in their workflows. The `--json` output format follows the same conventions as all other entities.

### All Users (File Path Display)

- **Benefit**: Every entity creation command now displays the file path where the entity markdown file was created, eliminating the need to guess or look up file locations.
- **Impact**: Minor UX improvement. No workflow change required.

---

## 6. High-Level Acceptance Criteria (UAT Scenarios)

### UAT-1: Tech-Debt CRUD Lifecycle

**Given** the Shark database is initialized and the user has a working `shark` CLI
**When** the user runs `shark td create "Refactor database connection pooling" --category=architecture --severity=high`
**Then** a tech-debt entity is created with key TD-001 (or next available), status set to the workflow default, and the CLI displays the entity key, title, and file path of the created markdown file.

**Given** tech-debt TD-001 exists
**When** the user runs `shark td get TD-001`
**Then** the CLI displays the entity details including key, title, status, category, severity, and file path.

**Given** tech-debt TD-001 exists
**When** the user runs `shark td update TD-001 --severity=critical`
**Then** the severity is updated and confirmed in subsequent `shark td get TD-001` output.

**Given** tech-debt TD-001 exists
**When** the user runs `shark td delete TD-001`
**Then** the entity is removed from the database and no longer appears in `shark td list`.

### UAT-2: Tech-Debt Listing and Filtering

**Given** multiple tech-debt entities exist with different categories and severities
**When** the user runs `shark td list --category=architecture`
**Then** only tech-debt items with category "architecture" are displayed.

**Given** multiple tech-debt entities exist
**When** the user runs `shark td list --json`
**Then** a valid JSON array is returned with all tech-debt entities.

### UAT-3: Tech-Debt Workflow

**Given** tech-debt TD-001 is in the default initial status
**When** the user runs `shark status advance TD-001`
**Then** the status transitions to the next valid status as defined by the workflow profile.

**Given** tech-debt TD-001 is in status "completed"
**When** the user runs `shark status advance TD-001`
**Then** the command returns an error indicating no valid next status (terminal state).

### UAT-4: Core Command Auto-Detection

**Given** tech-debt TD-001 exists
**When** the user runs `shark get TD-001`
**Then** the core `get` command auto-detects the TD-### key format and displays tech-debt details (same output as `shark td get TD-001`).

**Given** tech-debt TD-001 exists
**When** the user runs `shark get TD-001 --field status`
**Then** only the status value is returned as a plain string.

### UAT-5: Notes and Context

**Given** tech-debt TD-001 exists
**When** the user runs `shark td note add TD-001 --content="Found during code review of auth module" --type=comment`
**Then** the note is persisted and visible via `shark td notes TD-001`.

**Given** tech-debt TD-001 exists
**When** the user runs `shark td context set TD-001 --field affected_area --value "internal/repository/"`
**Then** the context field is set and retrievable via `shark td context get TD-001`.

### UAT-6: File Path Display on Entity Create

**Given** a fresh project with no entities
**When** the user runs `shark epic create "New Epic"`
**Then** the success output includes the file path (e.g., "File: docs/plan/E01-new-epic/epic.md").

**Given** a fresh project with epic E01
**When** the user runs `shark td create "Fix N+1 queries" --category=performance --severity=medium`
**Then** the success output includes the file path (e.g., "File: docs/plan/tech-debt/TD-001-fix-n-plus-1-queries.md" or the configured path).

### UAT-7: Search Integration

**Given** tech-debt TD-001 exists with title "Refactor database connection pooling"
**When** the user runs `shark search "database"`
**Then** TD-001 appears in the search results alongside any matching epics, features, tasks, bugs, or change-cards.

### UAT-8: Migration Safety

**Given** an existing database with epics, features, tasks, bugs, and change-cards
**When** the migration for tech-debt runs (via schema version bump)
**Then** the `tech_debts` table is created, all existing data in other tables is unchanged, and `shark epic list`, `shark task list`, `shark bug list`, and `shark change list` return the same results as before.

---

*Last Updated*: 2026-04-05
