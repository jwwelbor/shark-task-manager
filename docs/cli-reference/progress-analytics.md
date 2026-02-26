# Progress and Analytics Commands

## Overview

Shark provides two complementary commands for understanding project health and work patterns:

- **`shark progress`** - A dashboard view showing project progress, health indicators, active tasks, and blocked items. Use this to understand *where things stand* at a glance.
- **`shark analytics`** - Session pattern analysis across epics, features, and tasks. Use this to understand *how work is being done* -- session durations, pause frequency, time investment, and agent productivity.

Together, these commands give both a snapshot of current state (progress) and insight into work patterns over time (analytics).

---

## shark progress

Display a progress dashboard showing project progress, health indicators, active tasks, and blocked items.

### Usage

```
shark progress [EPIC] [FEATURE] [flags]
```

### Positional Arguments

| Argument | Description |
|----------|-------------|
| (no args) | Show full project progress dashboard |
| EPIC | Show progress for a specific epic (e.g., `E04`) |
| EPIC FEATURE | Show progress for a specific feature (e.g., `E04 F01` or `E04-F01`) |

Keys are case insensitive and support both numeric and slugged formats.

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--epic` | string | Filter by epic key (flag syntax alternative to positional arg) |
| `--include-archived` | bool | Include archived epics and features in the dashboard |
| `--recent` | string | Recent completion window. Valid values: `24h`, `7d`, `30d`, `90d` |
| `--json` | bool | Output in JSON format (machine-readable) |

### Examples

**Project-wide progress dashboard:**

```bash
shark progress
```

Shows an overview of all epics, their completion status, active tasks, and any blockers across the entire project.

**Epic-scoped progress:**

```bash
# Positional syntax (recommended)
shark progress E05

# Flag syntax (still supported)
shark progress --epic=E05
```

Shows progress for epic E05 including feature breakdowns, task status distribution, and health indicators.

**Feature-scoped progress:**

```bash
# Two-argument positional syntax
shark progress E05 F02

# Combined format
shark progress E05-F02
```

Shows detailed progress for a specific feature including individual task statuses and completion metrics.

**Include recent completions:**

```bash
# Show completions from the last 7 days
shark progress --recent=7d

# Show completions from the last 24 hours for a specific epic
shark progress E05 --recent=24h
```

**Include archived items:**

```bash
shark progress --include-archived
```

**JSON output for scripting or AI agents:**

```bash
shark progress --json
shark progress E05 --json
```

---

## shark analytics

Analyze work session patterns across epics, features, and tasks. Provides insights into session duration patterns, pause frequency, time investment, and agent productivity.

### Usage

```
shark analytics [flags]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--session-duration` | bool | Analyze session duration metrics |
| `--pause-frequency` | bool | Analyze pause frequency patterns |
| `--epic` | string | Filter by epic key |
| `--feature` | string | Filter by feature key |
| `--agent-type` | string | Filter by agent type |
| `--json` | bool | Output in JSON format (machine-readable) |

At least one analysis type flag (`--session-duration` or `--pause-frequency`) should be provided.

### Examples

**Session duration analysis for an epic:**

```bash
shark analytics --session-duration --epic E10
```

Shows how long work sessions last across tasks in epic E10, helping identify whether tasks are being worked on in focused blocks or fragmented sessions.

**Pause frequency analysis for a feature:**

```bash
shark analytics --pause-frequency --epic E10 --feature F05
```

Analyzes how often work is paused or interrupted within a specific feature, useful for identifying tasks that may be blocked or context-switched frequently.

**Agent productivity analysis:**

```bash
shark analytics --session-duration --epic E10 --agent-type backend
```

Filters session duration metrics to a specific agent type, enabling comparison of work patterns across different agent roles (e.g., `backend`, `frontend`, `qa`).

**Combined filters with JSON output:**

```bash
shark analytics --session-duration --epic E10 --feature F05 --json
```

---

## JSON Output

Both commands support the `--json` flag for machine-readable output, which is useful for:

- AI agent integrations that need structured data
- Custom dashboards or reporting scripts
- Piping into tools like `jq` for further processing

```bash
# Extract specific fields with jq
shark progress E05 --json | jq '.completion_percentage'
shark analytics --session-duration --epic E10 --json | jq '.sessions'
```

Both commands also support the global `--field` flag to extract a single field from JSON output:

```bash
shark progress E05 --json --field status
```

## Related Documentation

- [Task Commands](task-commands.md) - Task lifecycle and status transitions
- [Epic Commands](epic-commands.md) - Epic management including progress
- [Feature Commands](feature-commands.md) - Feature management
- [JSON Output Format](json-output.md) - JSON response structures
- [Enhanced JSON Fields](json-api-fields.md) - Progress tracking, health indicators, rollups
