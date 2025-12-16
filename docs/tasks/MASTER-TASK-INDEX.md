# Master Task Index: E04 Task Management CLI Core

## Overview

This master index tracks all implementation tasks across all features in Epic E04 (Task Management CLI Core).

**Epic Location**: `/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/`
**Total Features**: 7 (planned)
**Implemented Features**: 3 (E04-F01, E04-F02, E04-F04)

## Feature Task Indexes

### E04-F01: Database Schema & Core Data Model
**Status**: Ready for implementation
**Total Tasks**: 6
**Estimated Time**: 48 hours (6 business days)
**Task Index**: [TASK-INDEX.md](./TASK-INDEX.md)

| Task Key | Title | Status | Est. Time |
|----------|-------|--------|-----------|
| T-E04-F01-001 | Database Foundation - Models & Schema | todo | 8h |
| T-E04-F01-002 | Session Management | todo | 6h |
| T-E04-F01-003 | Repository Layer - CRUD Operations | todo | 16h |
| T-E04-F01-004 | Integration Tests & Performance Validation | todo | 8h |
| T-E04-F01-005 | Documentation & Usage Examples | todo | 6h |
| T-E04-F01-006 | Package Export & CLI Integration Prep | todo | 4h |

### E04-F02: CLI Infrastructure & Framework
**Status**: Ready for implementation
**Dependencies**: E04-F01-006 (Database Package Export)
**Total Tasks**: 6
**Estimated Time**: 48 hours (6 business days)
**Task Index**: [TASK-INDEX-E04-F02.md](./TASK-INDEX-E04-F02.md)

| Task Key | Title | Status | Est. Time |
|----------|-------|--------|-----------|
| T-E04-F02-001 | CLI Core Framework & Command Structure | todo | 8h |
| T-E04-F02-002 | Output Formatting System - JSON & Rich Tables | todo | 10h |
| T-E04-F02-003 | Configuration Management & User Defaults | todo | 6h |
| T-E04-F02-004 | Error Handling & Exit Code System | todo | 8h |
| T-E04-F02-005 | Database Context Integration & Session Management | todo | 6h |
| T-E04-F02-006 | Integration Testing, Documentation & Package Finalization | todo | 10h |

### E04-F04: Epic & Feature Queries
**Status**: Ready for implementation
**Dependencies**: E04-F01 (Database), E04-F02 (CLI Framework)
**Total Tasks**: 6
**Estimated Time**: 32 hours (4 business days)
**Task Index**: [TASK-INDEX-E04-F04.md](./TASK-INDEX-E04-F04.md)

| Task Key | Title | Status | Est. Time |
|----------|-------|--------|-----------|
| T-E04-F04-001 | Progress Calculation Service | todo | 6h |
| T-E04-F04-002 | Epic Query Commands | todo | 8h |
| T-E04-F04-003 | Feature Query Commands | todo | 8h |
| T-E04-F04-004 | JSON Output & Filtering | todo | 4h |
| T-E04-F04-005 | Integration Tests & Performance Validation | todo | 4h |
| T-E04-F04-006 | Documentation & Usage Examples | todo | 2h |

## Planned Features (Task Indexes Not Yet Created)

### E04-F03: Task Lifecycle Operations
**Status**: Design pending
**Dependencies**: E04-F01, E04-F02

### E04-F05: Folder Management
**Status**: Design pending
**Dependencies**: E04-F01, E04-F03

### E04-F06: Search & Analytics
**Status**: Design pending
**Dependencies**: E04-F01, E04-F04

### E04-F07: Initialization & Sync
**Status**: Design pending
**Dependencies**: E04-F01, E04-F02

## Task Workflow

All tasks follow this standard workflow:

```
docs/tasks/todo/           → New tasks created here
    ↓
docs/tasks/active/         → Moved when work begins
    ↓
docs/tasks/ready-for-review/ → Moved when implementation complete
    ↓
docs/tasks/completed/      → Moved after successful review
    ↓
docs/tasks/archived/       → Moved when no longer needed
```

## Epic-Level Dependencies

```
E04-F01 (Database Schema) [48h]
    ↓
E04-F02 (CLI Infrastructure) [48h]
    ↓
    ├─→ E04-F04 (Epic & Feature Queries) [32h]
    ├─→ E04-F03 (Task Lifecycle) [pending]
    └─→ E04-F07 (Init & Sync) [pending]
        ↓
    E04-F05 (Folder Management) [pending]
        ↓
    E04-F06 (Search & Analytics) [pending]
```

## Current Status Summary

**Total Tasks Created**: 18 (6 for E04-F01 + 6 for E04-F02 + 6 for E04-F04)
**Total Estimated Time**: 128 hours (16 business days)
**Tasks in Todo**: 18
**Tasks in Active**: 0
**Tasks Completed**: 0

## Next Actions

1. **E04-F01**: Begin implementation with T-E04-F01-001 (Database Foundation)
2. **E04-F02**: Begin implementation with T-E04-F02-001 once E04-F01-006 completes
3. **E04-F04**: Begin implementation once E04-F01 and E04-F02 are complete
4. **Remaining Features**: Create design documents and task indexes for E04-F03, E04-F05, E04-F06, E04-F07

---

**Last Updated**: 2025-12-14
**Epic Status**: In progress (3/7 features have task indexes, foundation features ready)
