# Implementation Tasks: E16-F01 Core Workflow Engine

## Overview

This folder contains agent-executable tasks that implement the E16-F01 Core Workflow Engine feature. This feature extends Shark's configurable workflow system from task-only to support three independent workflow levels: epic, feature, and task.

## Task Summary

| Task | Title | Dependencies | Execution Order |
|------|-------|-------------|-----------------|
| [T-E16-F01-001](./T-E16-F01-001.md) | Config schema and default workflows | None | 1 |
| [T-E16-F01-002](./T-E16-F01-002.md) | Multi-level config parser and cache | T-001 | 2 |
| [T-E16-F01-003](./T-E16-F01-003.md) | Extend workflow.Service with level awareness | T-002 | 3 |
| [T-E16-F01-004](./T-E16-F01-004.md) | Epic and Feature service layer with transition types | T-003 | 4 |
| [T-E16-F01-005](./T-E16-F01-005.md) | Model and CLI validation refactoring | T-003 | 5 (parallel with T-004) |
| [T-E16-F01-006](./T-E16-F01-006.md) | CLI commands: next-status, update refactor, service accessors | T-004, T-005 | 6 |
| [T-E16-F01-007](./T-E16-F01-007.md) | Workflow validate extension and integration testing | T-006 | 7 |

## Execution Order

```
Wave 1: Config Foundation
  T-001: Config schema, defaults, MultiLevelWorkflow container
  T-002: Multi-level parser and cache (depends on T-001)

Wave 2: Service Extension
  T-003: workflow.Service ForLevel() (depends on T-002)

Wave 3+4: Business Logic & Validation (can run in parallel)
  T-004: EpicService + FeatureService (depends on T-003)
  T-005: Model/CLI validation refactoring (depends on T-003)

Wave 5: CLI Integration
  T-006: next-status commands + update refactoring (depends on T-004, T-005)
  T-007: workflow validate + integration testing (depends on T-006)
```

## Design Documentation

All tasks reference these design documents:
- [Feature PRD](../feature.md) - Requirements and acceptance criteria
- [Technical Architecture](../architecture.md) - Detailed implementation design
- [BA Review](../ba-review.md) - Edge cases, risks, and refined acceptance criteria

## Status Management

Task status is tracked in the database via `shark` CLI:
- `shark task list E16 F01` - View all tasks
- `shark task start <key>` - Start a task
- `shark task complete <key>` - Mark for review
- `shark task next --agent=developer` - Get next available task
