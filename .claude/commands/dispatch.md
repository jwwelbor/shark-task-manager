---
description: Dispatch tasks using the tech-director agent for parallel execution and coordination
---

# Dispatch Tasks

Launch the TechDirector agent to review, coordinate, and execute multiple development tasks in parallel.

## Usage

```bash
/dispatch
```

Then specify the tasks you want executed.

## What This Does

This command launches the **TechDirector** agent who will:
1. Review all task files you provide
2. Analyze dependencies and determine parallel execution opportunities
3. Dispatch fresh, focused agents for each task
4. Coordinate execution waves (parallel where possible)
5. Commit each agent's work after completion
6. Run quality gates between integration points
7. Provide progress updates throughout

## Example

```
/dispatch

Execute these tasks:
- docs/plan/E04-F05/tasks/T-E04-F05-001.md
- docs/plan/E04-F05/tasks/T-E04-F05-002.md
- docs/plan/E04-F05/tasks/T-E04-F05-003.md
- docs/plan/E04-F05/tasks/T-E04-F05-004.md
- docs/plan/E04-F05/tasks/T-E04-F05-005.md
```

Or more simply:

```
/dispatch

Execute all tasks in docs/plan/E04-F05/tasks/
```

## What Makes This Different

The TechDirector agent:
- **Coordinates, doesn't implement** - Dispatches specialized agents for each task
- **Maximizes parallelism** - Runs independent tasks concurrently
- **Keeps agents focused** - One task per agent, fresh agent per task
- **Handles integration** - Commits work and validates between tasks
- **Manages dependencies** - Launches tasks only when prerequisites complete

This is ideal for executing multiple tasks that are part of a feature implementation.
