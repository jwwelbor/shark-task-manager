# E17: Personas

> Part of [E17: CLI Simplification for AI Agents](epic.md). See also: [User Journeys](user-journeys.md), [Requirements](requirements.md).

---

## Primary: AI Development Agent

**Name:** DevAgent
**Role:** LLM-based developer using Shark CLI via bash to manage task lifecycle
**Frequency:** 50-100+ CLI invocations per work session
**Technical Level:** High (can write bash, parse JSON) but has no persistent memory of CLI quirks

### Context
- Operates via bash tool calls in a Claude Code, Cursor, or similar AI coding environment
- Must parse CLI output programmatically (JSON preferred)
- Has no memory between sessions -- must re-learn CLI syntax each time from `--help` output and CLAUDE.md
- Cannot handle interactive prompts (stdin is not available)
- Accounts for approximately 70% of all Shark CLI invocations based on wormwoodGM activity logs

### Goals
1. Get assigned task details quickly (`shark get <id>`)
2. Advance task through workflow statuses reliably
3. Check task/feature status for decision-making
4. Create new tasks when needed

### Frustrations (Observed from Activity Logs)
1. **Cannot predict which status command works** - tries `set-status`, `update --status`, `next-status` before finding one that succeeds (see [Journey 1](user-journeys.md#journey-1-ai-agent-daily-task-workflow))
2. **Must use Python to extract single fields** from JSON output (no `--field` flag) -- observed ~30 times in 231-command sample
3. **Wraps every command in `2>/dev/null`** because error output is unpredictable -- observed in 36% of commands
4. **Writes bash for-loops** for batch operations because no batch mode exists -- observed ~12 times
5. **Invents commands that don't exist** (`shark task set-status`, `shark feature update --status`) because the API surface is not intuitive

### Needs
- Predictable, consistent command syntax (addressed by F01, F05, F08)
- Single-field extraction without external tools (addressed by F02)
- JSON output by default or via env var (addressed by F04)
- Structured JSON errors for programmatic handling (addressed by F03)
- Idempotent operations -- no error if already at target state (addressed by F01)
- Batch mode for multi-entity operations (addressed by F07)

---

## Secondary: AI Orchestrator Agent

**Name:** OrchestratorAgent
**Role:** LLM coordinating multiple development agents, managing workflow transitions across features
**Frequency:** 20-50 CLI invocations per orchestration session
**Technical Level:** High, focused on batch operations and status monitoring

### Context
- Coordinates work across multiple tasks and features
- Needs to advance groups of tasks through workflow stages
- Monitors feature-level progress to decide when to move to next phase
- May dispatch work to DevAgents based on task availability
- Accounts for approximately 20% of all Shark CLI invocations

### Goals
1. Batch-advance all tasks in a feature to the next workflow stage
2. Monitor feature progress and health
3. Identify blocked or stalled tasks
4. Assign tasks to appropriate agent types

### Frustrations (Observed from Activity Logs)
1. **No batch operations** - must loop over individual task commands (see [Journey 2](user-journeys.md#journey-2-orchestrator-batch-workflow-transition))
2. **Feature status updates are unreliable** - `shark feature update --status` does not work as expected
3. **Must aggregate task statuses manually** to determine feature health (see [Journey 4](user-journeys.md#journey-4-status-check-and-decision-making))
4. **next-status --preview then next-status --force** doubles command invocations

### Needs
- Batch status changes: `shark status set E18-F05-001 E18-F05-002 E18-F05-003 in_qa` (addressed by F07)
- Feature-level batch: `shark status set --feature E18-F05 in_qa` (addressed by F07)
- Reliable feature status/progress queries (addressed by F06)
- Single command to advance with no preview+confirm dance (addressed by F01)

---

## Tertiary: Human Developer

**Name:** HumanDev
**Role:** Developer occasionally using Shark CLI directly for task management
**Frequency:** 5-20 CLI invocations per day
**Technical Level:** Moderate, expects standard CLI conventions

### Context
- Uses CLI directly from terminal (not through AI agent)
- Expects familiar bash conventions (flags, exit codes, `--help`)
- May use tab completion
- Prefers human-readable table output by default
- Accounts for approximately 10% of all Shark CLI invocations

### Goals
1. Check what tasks are assigned to them
2. Update task status as work progresses
3. Create tasks for discovered work
4. View feature/epic progress

### Frustrations
1. Too many commands to remember (45+ paths)
2. Inconsistent flag names across entities (`--execution-order` vs `--order`)
3. Not clear which command to use for status changes

### Needs
- Clean `--help` output with examples (addressed by F09, F11)
- Consistent flag names (addressed by F05)
- Human-readable default output with tables and colors (preserved by default; JSON only when opted in)
- Tab completion support (out of scope for E17; see [scope.md](scope.md))
