# Product Manager Directive: E07-F20 CLI Command Options Standardization

**Date**: 2026-01-03
**Epic**: E07 - Enhancements
**Feature**: E07-F20 - CLI Command Options Standardization
**Assigned By**: Technical Delivery Director

---

## Directive

You are responsible for coordinating the implementation of feature E07-F20: CLI Command Options Standardization.

## Current State

- **Feature Status**: Draft (0% complete)
- **Task Count**: 19 tasks
- **All Tasks Status**: Draft (need to be transitioned to todo)
- **Design Documentation**: Complete and comprehensive (see artifacts below)

## Task Organization

Tasks are organized by priority groups:

### Priority 8 (Foundation - Tasks 1-4)
Case insensitive key handling:
- T-E07-F20-001: Add key normalization function
- T-E07-F20-002: Update validation functions
- T-E07-F20-003: Update parsing functions
- T-E07-F20-004: Add unit tests
- T-E07-F20-005: Add integration tests

### Priority 7 (Enhancement - Tasks 6-8)
Short task key format (drop T- prefix):
- T-E07-F20-006: Add short task key pattern and normalization
- T-E07-F20-007: Update task commands
- T-E07-F20-008: Add tests for short format

### Priority 6 (UX Improvement - Tasks 9-12)
Positional arguments for create commands:
- T-E07-F20-009: Update feature create command
- T-E07-F20-010: Update task create command
- T-E07-F20-011: Add unit tests for positional arguments
- T-E07-F20-012: Add integration tests for positional arguments

### Priority 5 (Error Handling - Tasks 13-15)
Enhanced error messages:
- T-E07-F20-013: Create error template system
- T-E07-F20-014: Update error messages throughout CLI
- T-E07-F20-015: Test enhanced error messages

### Priority 4 (Documentation - Tasks 16-19)
Documentation updates:
- T-E07-F20-016: Update CLI_REFERENCE.md
- T-E07-F20-017: Update CLAUDE.md
- T-E07-F20-018: Update README.md
- T-E07-F20-019: Create migration guide

## Your Responsibilities

### 1. Task State Management
- Transition all 19 tasks from "draft" → "todo" in shark
- Verify task dependencies and sequencing
- Ensure tasks are dev-ready with clear acceptance criteria

### 2. Developer Coordination
- Dispatch backend developers for implementation tasks (Priority 8-6)
- Coordinate testing specialists for test tasks
- Assign documentation tasks to technical writers

### 3. Progress Monitoring
- Update shark with task progress as work completes
- Track blockers and dependencies
- Coordinate code reviews between tasks

### 4. Quality Assurance
- Ensure all tests pass before marking tasks complete
- Verify backward compatibility throughout implementation
- Coordinate QA validation at each priority level

### 5. Communication
- Report progress back to Tech Director
- Escalate blockers or dependencies
- Update feature status as task groups complete

## Design Documentation

Comprehensive design docs are available at:
- `/home/jwwelbor/projects/shark-task-manager/docs/workflow/artifacts/F20-implementation-guide.md`
- `/home/jwwelbor/projects/shark-task-manager/docs/workflow/artifacts/F20-design-summary.md`
- `/home/jwwelbor/projects/shark-task-manager/docs/workflow/artifacts/F20-cli-ux-specification.md`

## Implementation Approach

Follow the phased approach in the implementation guide:
1. **Week 1**: Case insensitivity (Priority 8 tasks)
2. **Week 2**: Short keys + Positional args (Priority 7-6 tasks)
3. **Week 3**: Enhanced errors (Priority 5 tasks)
4. **Week 4**: Documentation (Priority 4 tasks)

## Success Criteria

Feature is complete when:
- [ ] All 19 tasks moved to "completed" status in shark
- [ ] All unit and integration tests pass
- [ ] Code review completed and approved
- [ ] Backward compatibility verified
- [ ] Documentation updated
- [ ] Feature ready for UAT presentation

## Shark Commands Reference

```bash
# Query feature state
./bin/shark feature get E07-F20 --json

# List tasks by priority
./bin/shark task list E07-F20 --json

# Update task status
./bin/shark task update <task-key> --status=todo

# Monitor progress
./bin/shark status --feature=E07-F20
```

## Next Actions

1. Query shark for complete feature and task details
2. Review all 19 task files for dev-readiness
3. Transition all tasks from "draft" to "todo"
4. Begin dispatching developers for Priority 8 tasks
5. Set up monitoring cadence for progress tracking

---

**Report Completion Back To**: Technical Delivery Director
**Communication**: Update shark continuously, report when E07-F20 is complete
