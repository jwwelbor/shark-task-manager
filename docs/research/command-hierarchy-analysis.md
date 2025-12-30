# Shark Command Hierarchy Analysis

## Current Structure (16 top-level commands)

```mermaid
graph LR
    shark[shark CLI]

    %% Top-level commands
    shark --> analytics[analytics]
    shark --> config[config]
    shark --> epic[epic]
    shark --> feature[feature]
    shark --> get[get]
    shark --> history[history]
    shark --> init[init]
    shark --> list[list]
    shark --> notes[notes]
    shark --> related-docs[related-docs]
    shark --> search[search]
    shark --> status[status]
    shark --> sync[sync]
    shark --> task[task]
    shark --> validate[validate]

    %% Task subcommands (27!)
    task --> task-approve[approve]
    task --> task-block[block]
    task --> task-blocked-by[blocked-by]
    task --> task-blocks[blocks]
    task --> task-complete[complete]
    task --> task-context[context]
    task --> task-create[create]
    task --> task-criteria[criteria]
    task --> task-delete[delete]
    task --> task-deps[deps]
    task --> task-get[get]
    task --> task-history[history]
    task --> task-link[link]
    task --> task-list[list]
    task --> task-next[next]
    task --> task-note[note]
    task --> task-notes[notes]
    task --> task-reopen[reopen]
    task --> task-resume[resume]
    task --> task-sessions[sessions]
    task --> task-start[start]
    task --> task-timeline[timeline]
    task --> task-unblock[unblock]
    task --> task-unlink[unlink]
    task --> task-update[update]

    style task fill:#ffcccc
    style task-approve fill:#ffe6e6
    style task-block fill:#ffe6e6
    style task-complete fill:#ffe6e6
    style task-start fill:#ffe6e6
```

**Problem:** `task` has **27 subcommands** - way too many!

---

## Proposed: AI-Agent-Focused Structure (13 top-level commands)

Optimized for spec-driven development workflows.

```mermaid
graph LR
    shark[shark CLI]

    %% Core workflow commands (what AI agents do most)
    shark --> work[work]
    shark --> deps[deps]
    shark --> status[status]

    %% CRUD operations
    shark --> epic[epic]
    shark --> feature[feature]
    shark --> task[task]

    %% Query & discovery
    shark --> get[get]
    shark --> list[list]
    shark --> search[search]
    shark --> history[history]

    %% Setup & maintenance
    shark --> init[init]
    shark --> sync[sync]
    shark --> config[config]

    %% Work subcommands (lifecycle)
    work --> work-next[next]
    work --> work-start[start TASK-KEY]
    work --> work-complete[complete TASK-KEY]
    work --> work-approve[approve TASK-KEY]
    work --> work-reopen[reopen TASK-KEY]
    work --> work-block[block TASK-KEY]
    work --> work-unblock[unblock TASK-KEY]
    work --> work-resume[resume TASK-KEY]

    %% Deps subcommands (relationships)
    deps --> deps-add[add TASK1 TASK2]
    deps --> deps-remove[remove TASK1 TASK2]
    deps --> deps-show[show TASK-KEY]
    deps --> deps-tree[tree TASK-KEY]

    %% Task subcommands (reduced to 6)
    task --> task-create[create]
    task --> task-get[get]
    task --> task-list[list]
    task --> task-update[update]
    task --> task-delete[delete]
    task --> task-note[note]

    style work fill:#ccffcc
    style deps fill:#ccccff
    style task fill:#ffffcc
    style status fill:#ffccff
```

---

## Command Count Comparison

| Category | Current | Proposed | Change |
|----------|---------|----------|--------|
| **Top-level commands** | 16 | 13 | -3 |
| **Task subcommands** | 27 | 6 | -21 |
| **Work subcommands** | 0 | 8 | +8 |
| **Deps subcommands** | 0 | 4 | +4 |

---

## AI Agent Workflow Examples

### Current (verbose, nested)
```bash
# 1. Find work
shark task next --json

# 2. Start work
shark task start T-E10-F03-004 --json

# 3. Check what blocks me
shark task blocked-by T-E10-F03-004 --json

# 4. Complete work
shark task complete T-E10-F03-004 --json

# 5. Check project status
shark status --json
```

### Proposed (concise, discoverable)
```bash
# 1. Find work
shark work next --json

# 2. Start work
shark work start T-E10-F03-004 --json

# 3. Check what blocks me
shark deps show T-E10-F03-004 --json

# 4. Complete work
shark work complete T-E10-F03-004 --json

# 5. Check project status
shark status --json
```

---

## Core AI Agent Commands (The Essential 10)

For spec-driven development, AI agents primarily need:

| Command | Purpose | Frequency |
|---------|---------|-----------|
| `shark work next` | Find next task | Very High |
| `shark work start TASK` | Begin work | Very High |
| `shark work complete TASK` | Finish work | Very High |
| `shark work resume TASK` | Get context to resume | High |
| `shark deps show TASK` | Check dependencies | High |
| `shark deps add T1 T2` | Add dependency | Medium |
| `shark task get TASK` | Get task details | High |
| `shark status` | Project overview | Medium |
| `shark task create` | Create new task | Medium |
| `shark sync` | Sync filesystem | Low |

---

## Alternative: Ultra-Minimal Structure

If we want to go **even simpler** for AI agents:

```mermaid
graph LR
    shark[shark CLI]

    %% Just 8 top-level commands
    shark --> next[next]
    shark --> start[start TASK]
    shark --> complete[complete TASK]
    shark --> get[get TASK]
    shark --> list[list]
    shark --> deps[deps]
    shark --> status[status]
    shark --> create[create]

    %% Deps subcommands
    deps --> deps-add[add T1 T2]
    deps --> deps-show[show TASK]

    %% Create subcommands
    create --> create-epic[epic]
    create --> create-feature[feature]
    create --> create-task[task]

    style next fill:#ccffcc
    style start fill:#ccffcc
    style complete fill:#ccffcc
    style deps fill:#ccccff
```

This gives **8 top-level commands** for the most common AI operations.

---

## Recommendation

**For AI-agent spec-driven development:**

Go with the **"work" command group** approach (13 top-level):
- ✅ Keeps top-level manageable (13 vs 16)
- ✅ Groups lifecycle operations logically (`work`)
- ✅ Separates dependency management (`deps`)
- ✅ Reduces cognitive load (27 → 6 task subcommands)
- ✅ Makes AI agent code more readable
- ✅ Matches mental model: "I want to work on tasks" vs "I want to task... something"

### Migration Path

1. Keep old commands as **aliases** for 2-3 releases
2. Add deprecation warnings
3. Update AI agent prompts/documentation
4. Remove aliases in v2.0

Would you like me to implement this restructure?
