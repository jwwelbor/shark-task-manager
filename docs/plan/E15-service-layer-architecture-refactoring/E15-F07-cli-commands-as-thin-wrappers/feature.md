---
feature_key: E15-F07-cli-commands-as-thin-wrappers
epic_key: E15
title: CLI Commands as Thin Wrappers
description: Refactor task.go, feature.go, and epic.go from fat controllers (6,858 lines total) to thin wrappers (1,200 lines total). Commands must only parse arguments, call service methods, and format output. All business logic must reside in services.
---

# CLI Commands as Thin Wrappers

**Feature Key**: E15-F07-cli-commands-as-thin-wrappers

---

## Epic

- **Epic PRD**: [Service Layer Architecture Refactoring](../../epic.md)

---

## Goal

### Problem

CLI command files are fat controllers containing 40-45% business logic mixed with argument parsing and output formatting:
- **task.go**: 2,664 lines (business logic: workflow validation, dependency checks, status cascading)
- **feature.go**: 2,254 lines (business logic: progress calculation, health derivation)
- **epic.go**: 1,940 lines (business logic: feature rollups, impediment analysis)

This violates clean architecture and makes commands impossible to test without database.

### Solution

Refactor all CLI commands to thin wrappers (200-400 lines each) with pattern: **parse args → call service → format output**. Create global service accessors (cli.GetTaskService(), cli.GetFeatureService(), cli.GetEpicService()) for easy service access.

### Impact

- **82% code reduction**: 6,858 lines → 1,200 lines (5,658 lines removed)
- **Architecture compliance**: CLI layer has ZERO business logic
- **Testability**: Commands testable with mocked services

---

## User Stories (MoSCoW)

### Must-Have Stories

**Story 1**: As a Shark developer, I want global service accessors so commands can easily access services.

**Acceptance Criteria**:
- [ ] File `internal/cli/services_global.go` exists
- [ ] Functions: GetTaskService(), GetFeatureService(), GetEpicService()
- [ ] Accessors reuse shared dependencies (DB, workflow service)

**Story 2**: As a Shark developer, I want task.go refactored to thin wrapper (≤400 lines).

**Acceptance Criteria**:
- [ ] task.go reduced from 2,664 to ≤400 lines (85% reduction)
- [ ] All task commands call TaskService methods
- [ ] No business logic in task.go

**Story 3**: As a Shark developer, I want feature.go refactored to thin wrapper (≤350 lines).

**Story 4**: As a Shark developer, I want epic.go refactored to thin wrapper (≤300 lines).

---

## Requirements

1. **REQ-F-001**: Create global service accessors (internal/cli/services_global.go)
2. **REQ-F-002**: Refactor task.go (2,664 → ≤400 lines)
3. **REQ-F-003**: Refactor feature.go (2,254 → ≤350 lines)
4. **REQ-F-004**: Refactor epic.go (1,940 → ≤300 lines)

---

## Success Metrics

- **Line reduction**: 82% (6,858 → 1,200 lines)
- **Business logic**: 0% in CLI commands (100% in services)
- **Test regression**: 0 new failures

---

## Dependencies

- **Requires**: E15-F05 (services must be complete)
- **Requires**: E15-F06 (repositories must be pure data access)
- **Blocks**: E15-F11 (integration testing)

---

*Last Updated*: 2026-02-17
