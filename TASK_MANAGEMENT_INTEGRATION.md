# Shark Task Management Integration

This document explains the complete integration of shark task management into the Claude Code workflow.

## Overview

The shark CLI has been integrated into the development workflow through three complementary mechanisms:

1. **Slash Commands** - Explicit, discoverable task management commands
2. **Skills** - Centralized knowledge and integration points
3. **Optional Hooks** - Automated reminders and validations

## Quick Start

### Using Slash Commands

The simplest way to use task management:

```bash
# Get next task
/task-next

# Start working on it
/task-start T-E04-F02-001

# [... implement, test, validate ...]

# Complete the task
/task-complete T-E04-F02-001
```

All commands are available via tab completion and appear in `/help` output.

## Available Slash Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `/task-start` | Start a task | `/task-start T-E04-F02-001` |
| `/task-complete` | Complete a task | `/task-complete T-E04-F02-001` |
| `/task-next` | Get next available task | `/task-next` |
| `/task-list` | List tasks with filters | `/task-list --status=todo` |
| `/task-info` | Get task details | `/task-info T-E04-F02-001` |

## Skill Integration

The `shark-task-management` skill provides:

- Complete workflow documentation
- CLI reference
- Best practices
- Troubleshooting guides
- Integration examples

### Location

`.claude/skills/shark-task-management/SKILL.md`

### How Other Skills Use It

The `implementation` skill references task management in all workflows:

```markdown
## Phase 0: Start Task Tracking

Before beginning implementation, start task tracking:

/task-start <task-id>

This updates task status and enables progress tracking.
```

## Optional Hooks

For users who want automation, optional hooks are available:

### Test Completion Reminder

Shows a reminder when tests pass:

```bash
# After running tests successfully
⚠️  All tests passed! Consider completing the task with: /task-complete <task-id>
```

**Enable:** See `.claude/skills/shark-task-management/HOOKS.md`

### Other Available Hooks

- Auto-start tasks when reading task files
- Validate active task before editing implementation files

**All hooks are opt-in and documented in `HOOKS.md`**

## Implementation Workflow Changes

All implementation workflows now include task tracking:

### Before (Old Workflow)

```
1. Read design documentation
2. Implement code
3. Run tests
4. Complete
```

### After (New Workflow)

```
1. Read design documentation
2. Start task: /task-start <task-id>
3. Implement code
4. Run tests
5. Complete task: /task-complete <task-id>
```

## Updated Files

### Slash Commands Created

```
.claude/commands/
├── task-start.md       - Start task tracking
├── task-complete.md    - Complete task tracking
├── task-next.md        - Get next task
├── task-list.md        - List tasks
└── task-info.md        - Get task info
```

### Skills Created

```
.claude/skills/shark-task-management/
├── SKILL.md           - Main skill documentation
├── HOOKS.md           - Optional hooks guide
└── README.md          - Skill overview
```

### Implementation Skill Updated

```
.claude/skills/implementation/
├── SKILL.md                           - References shark-task-management
├── README.md                          - Updated with task workflow
└── workflows/
    ├── implement-api.md              - Added Phase 0: task tracking
    ├── implement-backend.md          - Added Phase 0: task tracking
    ├── implement-frontend.md         - Added Phase 0: task tracking
    ├── implement-database.md         - Added Phase 0: task tracking
    └── implement-tests.md            - Added Phase 0: task tracking
```

### Optional Hooks

```
.claude/hooks/
└── task-completion-reminder.json  - Example hook configuration
```

## Benefits

### For Individual Developers

- **Clear workflow**: Know exactly when to start/complete tasks
- **Explicit control**: You choose when to run commands
- **Automatic tracking**: Status updates in database
- **No surprises**: Commands do exactly what they say

### For Teams

- **Visibility**: See who's working on what
- **Progress tracking**: Accurate status in database
- **Audit trail**: Complete history of task lifecycle
- **Consistency**: Everyone follows same pattern

### For Agents

- **Clear instructions**: Slash commands in all workflows
- **Discoverable**: Available via `/help`
- **Documented**: Complete skill reference
- **Integration points**: Other skills can reference

## Usage Examples

### Standard Development Workflow

```bash
# Morning standup - check your tasks
/task-list --status=in-progress

# Get next task
/task-next

# Review output: T-E04-F02-003 "Implement user service layer"

# Start the task
/task-start T-E04-F02-003

# Implementation work...
# - Read design docs
# - Write code
# - Write tests
# - Run validation gates

# All tests passing, ready to complete
/task-complete T-E04-F02-003

# Get next task
/task-next
```

### With Optional Hooks

```bash
# Tests pass automatically
pytest

# Hook shows reminder:
# ⚠️  All tests passed! Consider completing the task with: /task-complete T-E04-F02-003

# Complete when ready
/task-complete T-E04-F02-003
```

### Team Coordination

```bash
# See what's in progress
/task-list --status=in-progress

# Output:
# T-E04-F02-001  in-progress  Alice    Implement database schema
# T-E04-F02-002  in-progress  Bob      Create API endpoints
# T-E04-F02-003  todo         -        Implement user service

# Start the available task
/task-start T-E04-F02-003
```

## Design Rationale

### Why Slash Commands?

**Explicit > Implicit**
- Users control when to run commands
- No surprising automations
- Clear cause and effect

**Discoverable**
- Appear in `/help` output
- Support tab completion
- Self-documenting

**Project-Specific**
- Commands live in `.claude/commands/`
- Checked into version control
- Shared across team

### Why Skills?

**Knowledge Centralization**
- Single source of truth
- Reusable by other skills
- Progressive disclosure

**Context Efficient**
- Loaded only when needed
- References from other skills
- Avoids duplication

### Why Optional Hooks?

**Power User Features**
- Choose what to enable
- Deterministic automation
- Transparent behavior

**Safe Defaults**
- Off by default
- Well documented
- Can warn vs block

## Migration Guide

### For Existing Workflows

No breaking changes! The implementation skill still works as before, it just now includes task tracking as an optional Phase 0.

**Before:**
```markdown
## Prerequisites
...

## Phase 1: Design Service Interface
...
```

**After:**
```markdown
## Prerequisites
...

## Phase 0: Start Task Tracking (Optional)
/task-start <task-id>

## Phase 1: Design Service Interface
...
```

### Enabling for Your Team

1. **Commit the new files:**
   ```bash
   git add .claude/
   git commit -m "feat: add shark task management integration"
   ```

2. **Team members pull changes:**
   ```bash
   git pull
   ```

3. **Commands available immediately:**
   ```bash
   /help  # Shows new task-* commands
   ```

4. **Optional: Enable hooks individually:**
   - See `.claude/skills/shark-task-management/HOOKS.md`
   - Each user can enable in their `~/.claude/settings.json`

## Troubleshooting

### Commands not showing in /help

Check that files exist:
```bash
ls .claude/commands/task-*.md
```

### Skill not loading

Verify skill structure:
```bash
cat .claude/skills/shark-task-management/SKILL.md | head -5
```

Should show YAML frontmatter with `name` and `description`.

### Shark CLI not found

Install or build shark:
```bash
make install-pm
# or
make pm
```

## Future Enhancements

Potential additions:

- `/task-assign <task-id> <agent>` - Assign tasks to specific agents
- `/task-block <task-id> <reason>` - Mark task as blocked
- `/task-depends <task-id>` - Show task dependencies
- Integration with git commits (link commits to tasks)
- Integration with PRs (link PRs to completed tasks)

## See Also

- [CLI Documentation](docs/CLI.md) - Complete shark CLI reference
- [Implementation Skill](.claude/skills/implementation/SKILL.md) - Updated workflows
- [Shark Task Management Skill](.claude/skills/shark-task-management/SKILL.md) - Complete reference
- [Optional Hooks Guide](.claude/skills/shark-task-management/HOOKS.md) - Automation options

---

**Questions or Issues?**

Open an issue or discuss in team chat.
