# Feature: Cascading Status Calculation

**Feature Key**: E07-F14
**Epic**: E07 - Enhancements
**Status**: Draft
**Priority**: High

## Summary

Automatically calculate Epic and Feature status based on their children (Features and Tasks respectively), with cascading updates when any child status changes.

## Business Value

- Reduces manual status management overhead
- Ensures status accuracy matches actual work state
- Improves dashboard and reporting reliability
- Decreases cognitive load for project managers

## Key Capabilities

1. **Feature Status from Tasks**: Feature status derived from aggregate task statuses
2. **Epic Status from Features**: Epic status derived from aggregate feature statuses
3. **Automatic Cascade**: Status changes propagate up the hierarchy automatically
4. **Manual Override**: Ability to override calculated status when needed
5. **Status Visibility**: Clear indication of calculated vs. manually overridden status

## Status Calculation Summary

### Feature Status Logic
| Task States | Feature Status |
|-------------|----------------|
| No tasks | draft |
| All tasks todo | draft |
| Any task in_progress/ready_for_review/blocked | active |
| All tasks completed/archived | completed |

### Epic Status Logic
| Feature States | Epic Status |
|----------------|-------------|
| No features | draft |
| All features draft | draft |
| Any feature active/blocked | active |
| All features completed/archived | completed |

## Scope

### In Scope
- Database schema changes (status_override column)
- Repository layer recalculation methods
- CLI updates for override and status source display
- Cascade trigger on task/feature status changes
- Cascade trigger on task/feature create/delete
- Manual override with --status=<value> and --status=auto

### Out of Scope
- Status change notifications/webhooks
- Historical status tracking (feature/epic history tables)
- Real-time UI updates (no frontend in shark)
- Cross-project status aggregation

## Dependencies

- Current progress calculation infrastructure (already exists)
- Task status transition logic (already exists)
- Feature and Epic update commands (already exist)

## Risks

| Risk | Mitigation |
|------|------------|
| Performance impact on large projects | Use aggregation queries, benchmark before release |
| Breaking existing workflows | Status override preserves current behavior for manual users |
| Database migration complexity | Migration adds nullable column with default |

## Success Metrics

- Status accuracy: Calculated status matches expected in 100% of test cases
- Performance: Recalculation < 100ms for typical project sizes
- Adoption: Zero breaking changes to existing CLI commands

## Related Documents

- [Full PRD](./prd.md) - Detailed requirements and acceptance criteria
