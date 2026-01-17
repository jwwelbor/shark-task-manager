# Workflow Configuration

Shark supports customizable workflow configuration through `.sharkconfig.json`.

## Configuration Structure

```json
{
  "interactive_mode": false,
  "status_flow": {
    "draft": ["ready_for_refinement", "cancelled"],
    "ready_for_refinement": ["in_refinement", "cancelled"],
    "in_refinement": ["ready_for_development", "draft"],
    "ready_for_development": ["in_development", "cancelled"],
    "in_development": ["ready_for_code_review", "blocked"],
    "ready_for_code_review": ["in_code_review", "in_development"],
    "in_code_review": ["ready_for_qa", "in_development"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["ready_for_approval", "in_development"],
    "ready_for_approval": ["in_approval"],
    "in_approval": ["completed", "ready_for_qa"],
    "completed": [],
    "blocked": ["ready_for_development"],
    "cancelled": []
  },
  "status_metadata": {
    "draft": {
      "color": "gray",
      "description": "Task created but not yet refined",
      "phase": "planning"
    },
    "in_development": {
      "color": "yellow",
      "description": "Code implementation in progress",
      "phase": "development",
      "agent_types": ["developer", "ai-coder"]
    },
    "completed": {
      "color": "green",
      "description": "Task finished and approved",
      "phase": "done"
    }
  },
  "special_statuses": {
    "_start_": ["draft", "ready_for_development"],
    "_complete_": ["completed", "cancelled"]
  }
}
```

## Configuration Options

### interactive_mode
Controls interactive prompts for status transitions (optional, default: `false`)
- `false` (default): Auto-select first transition when multiple options available (ideal for agents/automation)
- `true`: Show interactive prompt for user selection when multiple options available
- See [Interactive Mode Configuration](interactive-mode.md) for detailed documentation

### status_flow
Defines valid transitions between statuses
- Key: Source status
- Value: Array of valid target statuses

### status_metadata
Metadata for each status
- `color`: ANSI color name (red, green, yellow, blue, cyan, magenta, gray, white, orange, purple)
- `description`: Human-readable description
- `phase`: Workflow phase (planning, development, review, qa, approval, done, any)
- `agent_types`: Array of agent types that can work on tasks in this status

### special_statuses
Special status markers
- `_start_`: Valid initial statuses for new tasks
- `_complete_`: Terminal statuses (no transitions out)

## Workflow Phases

Phases are used to order status displays:

1. **planning**: Draft, refinement stages (gray, cyan colors)
2. **development**: Active implementation (yellow colors)
3. **review**: Code review stages (magenta colors)
4. **qa**: Quality assurance (green colors)
5. **approval**: Final approval stages (purple colors)
6. **done**: Terminal states (white/green colors)
7. **any**: Status applicable to any phase (blocked, on_hold)

## Feature Get Display

The `shark feature get` command shows workflow-aware status information:
- Status breakdown ordered by workflow phase
- Statuses colored according to `status_metadata` colors
- Phase column shows which workflow stage each status belongs to
- "All tasks completed!" message when progress reaches 100%

## Example: Simple Workflow

For a simpler workflow with fewer statuses:

```json
{
  "status_flow": {
    "todo": ["in_progress"],
    "in_progress": ["review", "blocked"],
    "review": ["done", "in_progress"],
    "blocked": ["in_progress"],
    "done": []
  },
  "status_metadata": {
    "todo": {"color": "gray", "phase": "planning"},
    "in_progress": {"color": "yellow", "phase": "development"},
    "review": {"color": "magenta", "phase": "review"},
    "blocked": {"color": "red", "phase": "any"},
    "done": {"color": "green", "phase": "done"}
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["done"]
  }
}
```

## Related Documentation

- [Interactive Mode](interactive-mode.md) - Configure interactive prompts
- [Task Commands](task-commands.md) - Task status transitions
- [Feature Commands](feature-commands.md) - Feature display with workflow awareness
- [Configuration](configuration.md) - General configuration commands
