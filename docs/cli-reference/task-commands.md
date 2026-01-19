# Task Commands

Commands for managing tasks.

## Quick Reference

- `shark task create` - Create a new task
- `shark task list` - List tasks with filtering
- `shark task get` - Get task details
- `shark task next` - Find next available task
- `shark task start` - Start working on a task
- `shark task complete` - Mark task ready for review
- `shark task approve` - Approve and complete task
- `shark task reopen` - Reopen task for rework
- `shark task block` - Block a task
- `shark task unblock` - Unblock a task
- `shark task next-status` - Transition to next status

See [Task Commands (Full)](task-commands-full.md) for complete documentation of all task commands.

## Common Workflows

### Creating and Starting a Task

```bash
# Create task (positional syntax)
shark task create E07 F01 "Implement JWT validation" --agent=backend --priority=3

# Start the task
shark task start E07-F01-001 --agent="ai-agent-001" --json
```

### Completing a Task

```bash
# Mark complete (ready for review)
shark task complete E07-F01-001 --notes="Implementation complete, all tests passing"

# Approve task
shark task approve E07-F01-001 --notes="LGTM, approved"
```

### Handling Rejections

```bash
# Reopen with rejection reason
shark task reopen E07-F01-001 \
  --rejection-reason="Missing error handling on line 67" \
  --reason-doc="docs/reviews/code-review.md"
```

See [Rejection Reasons](rejection-reasons.md) for detailed documentation.

## Related Documentation

- [Task Commands (Full)](task-commands-full.md) - Complete command reference
- [Rejection Reasons](rejection-reasons.md) - Rejection workflow documentation
- [Orchestrator Actions](orchestrator-actions.md) - API response format
- [Key Formats](key-formats.md) - Case insensitive and short formats
