# E04-F03: Task Lifecycle Operations - Implementation Tasks

## Overview

This document tracks the implementation tasks (Product Requirement Prompts) for the Task Lifecycle Operations feature. The feature provides CLI commands for managing task lifecycle from creation to completion, including querying, state transitions, blocking, and exception handling.

## Task Breakdown

### Phase 1: Query Operations
**Task**: T-E04-F03-001 - Task Query Operations (list, get, next)
- **Status**: todo
- **Priority**: 1
- **Estimated Time**: 8 hours
- **Dependencies**: T-E04-F01-006 (Database Schema), T-E04-F02-001 (CLI Infrastructure)
- **Agent**: general-purpose

**Scope**:
- Implement `shark task list` command with filtering (status, epic, feature, agent, priority)
- Implement `shark task get <task-key>` command for task details
- Implement `shark task next` command for intelligent task discovery
- Support both human-readable tables and JSON output
- Include dependency validation in next command
- Meet performance targets (<100ms for list, <50ms for next)

**Deliverables**:
- CLI commands: task list, task get, task next
- Service layer: task query service
- Database queries with filters and sorting
- Tests: unit, service, integration

---

### Phase 2: Core State Transitions
**Task**: T-E04-F03-002 - Core State Transition Operations (start, complete, approve)
- **Status**: todo
- **Priority**: 1
- **Estimated Time**: 10 hours
- **Dependencies**: T-E04-F03-001
- **Agent**: general-purpose

**Scope**:
- Implement `shark task start <task-key>` command (todo → in_progress)
- Implement `shark task complete <task-key>` command (in_progress → ready_for_review)
- Implement `shark task approve <task-key>` command (ready_for_review → completed)
- State transition validation (reject invalid transitions)
- History recording for all state changes
- Atomic transactions (database + file operations)
- Feature progress calculation updates

**Deliverables**:
- CLI commands: task start, task complete, task approve
- Service layer: state transition service, state validator, history service
- Transaction management with rollback on failure
- Tests: unit, service, integration

---

### Phase 3: Exception Handling
**Task**: T-E04-F03-003 - Blocking and Exception State Operations (block, unblock, reopen)
- **Status**: todo
- **Priority**: 2
- **Estimated Time**: 6 hours
- **Dependencies**: T-E04-F03-002
- **Agent**: general-purpose

**Scope**:
- Implement `shark task block <task-key> --reason="..."` command
- Implement `shark task unblock <task-key>` command
- Implement `shark task reopen <task-key>` command for rework
- Block from todo or in_progress states
- Unblock returns to todo state
- Reopen returns from ready_for_review to in_progress
- History recording with reasons and notes

**Deliverables**:
- CLI commands: task block, task unblock, task reopen
- Service layer: exception handling service
- Extended state validation for exception transitions
- Tests: unit, service, integration

---

### Phase 4: Integration & Validation
**Task**: T-E04-F03-004 - Integration Testing & Workflow Validation
- **Status**: todo
- **Priority**: 2
- **Estimated Time**: 8 hours
- **Dependencies**: T-E04-F03-001, T-E04-F03-002, T-E04-F03-003
- **Agent**: general-purpose

**Scope**:
- End-to-end workflow tests (complete task lifecycle)
- Exception workflow tests (blocking, unblocking, reopening)
- Transaction rollback tests (failure scenarios)
- Dependency validation tests (next command behavior)
- Concurrent execution tests (multiple agents)
- Performance benchmarks (meet all PRD targets)
- Agent workflow simulation tests (JSON parsing, automation)
- Verify all PRD acceptance criteria met

**Deliverables**:
- Integration test suites for all workflows
- Performance benchmark tests
- Concurrent execution tests
- Agent simulation tests
- Coverage report (>95% target)
- Documentation of test results

---

## Dependencies

### External Dependencies (Other Features)
- **E04-F01**: Database Schema - Provides Task model, SessionFactory, repositories
- **E04-F02**: CLI Infrastructure - Provides Click framework, Rich formatting, error handling
- **E04-F05**: Folder Management - File operations (will be called but may need stubbing initially)

### Internal Dependencies (Between Tasks)
```
T-E04-F01-006, T-E04-F02-001
        ↓
T-E04-F03-001 (Query Operations)
        ↓
T-E04-F03-002 (Core Transitions)
        ↓
T-E04-F03-003 (Exception Handling)
        ↓
T-E04-F03-004 (Integration Testing)
```

## Execution Strategy

### Sequential Execution
Tasks must be executed in order (T-E04-F03-001 → 002 → 003 → 004) because:
1. State transitions (002) depend on query operations (001) to retrieve tasks
2. Exception handling (003) extends state validation from core transitions (002)
3. Integration testing (004) validates all components together

### Estimated Timeline
- **Phase 1**: 8 hours (Query Operations)
- **Phase 2**: 10 hours (Core Transitions)
- **Phase 3**: 6 hours (Exception Handling)
- **Phase 4**: 8 hours (Integration Testing)
- **Total**: 32 hours (~4 days with 1 agent)

### Parallel Opportunities
While tasks are sequential, each task includes parallel work:
- Unit tests can be written alongside implementation
- Documentation can be written during development
- Multiple test files can be created in parallel

## Acceptance Validation

### Feature Complete When
- [ ] All 4 tasks completed (T-E04-F03-001 through T-E04-F03-004)
- [ ] All PRD acceptance criteria validated with tests
- [ ] Performance benchmarks met (list <100ms, next <50ms, transitions <200ms)
- [ ] Integration tests pass with >95% code coverage
- [ ] All commands work with both human and JSON output modes
- [ ] State machine validated (only valid transitions allowed)
- [ ] History recorded for all state changes
- [ ] Atomic transactions verified (rollback on failure)
- [ ] Agent workflow simulation succeeds (next → start → complete automation)

### Key Metrics
- **Commands Implemented**: 9 (list, get, next, start, complete, approve, block, unblock, reopen)
- **Performance Targets**: 3 (<100ms list, <50ms next, <200ms transitions)
- **Test Coverage**: >95%
- **State Transitions**: 7 valid transitions validated
- **Exit Codes**: 4 (0=success, 1=user error, 2=system error, 3=validation error)

## Resources

- **PRD**: [Task Lifecycle Operations PRD](./prd.md)
- **Epic**: [E04 Task Management CLI Core](../epic.md)
- **Database Schema**: [E04-F01 Design Docs](../E04-F01-database-schema/)
- **CLI Framework**: [E04-F02 PRD](../E04-F02-cli-infrastructure/prd.md)

## Status Tracking

| Task | Status | Started | Completed | Agent | Notes |
|------|--------|---------|-----------|-------|-------|
| T-E04-F03-001 | todo | - | - | - | Query operations |
| T-E04-F03-002 | todo | - | - | - | Core state transitions |
| T-E04-F03-003 | todo | - | - | - | Exception handling |
| T-E04-F03-004 | todo | - | - | - | Integration testing |

**Last Updated**: 2025-12-14
