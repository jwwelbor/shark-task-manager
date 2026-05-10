# Sprint Commands

Complete reference for sprint lifecycle, planning, capacity, and reporting commands in Shark Task Manager.

---

## Agent Sprint Workflow

This section covers how an agent plans and executes a sprint end-to-end using the Claude-side slash commands. For the raw `shark sprint` CLI reference, see [below](#quick-reference).

### The Four Sprint Slash Commands

| Command | When to use |
|---|---|
| `/plan-sprint S###` | Fill the sprint backlog before starting — proposes assignments, confirms with user, never starts the sprint |
| `/run-sprint S###` | Drive an active sprint solo — pulls one entity at a time via `shark sprint next`, dispatches each via `/run` |
| `/run-sprint-team S###` | Drive an active sprint with a parallel agent team — groups tasks by feature, dispatches each feature group via `/run-agent-team`, standalones via `/run` |
| `/retro-sprint S###` | Post-close retrospective — reads `shark sprint summary --detailed` and velocity data, produces a five-section markdown report |

### Full Lifecycle Walkthrough

The typical agent sprint cycle is: **create → plan → start → execute → close → retro**.

#### 1. Create the sprint

```bash
shark sprint create "Sprint 5" --start=2026-05-12 --end=2026-05-26 --json
# → {"key": "S005", "status": "planning", ...}
```

Set capacity so the planner knows how much to assign per agent type:

```bash
shark sprint capacity set S005 --agent=backend --points=21
shark sprint capacity set S005 --agent=frontend --points=13
```

#### 2. Plan — fill the sprint backlog

Run `/plan-sprint` to see what's eligible and assign work:

```bash
/plan-sprint S005
```

The skill reads `shark sprint plan S005 --json` (backlog + capacity + readiness) and offers two modes:

- **Interactive** (default): presents entities group-by-feature, asks yes/no/pick per group
- **Auto** (`--mode=auto`): greedy-fills capacity by agent type, shows proposed plan, one confirmation

```bash
/plan-sprint S005 --mode=auto          # fill to capacity automatically
/plan-sprint S005 --mode=auto --max-add=10  # cap at 10 new assignments
```

The skill reports a readiness-score delta when it exits and never calls `shark sprint start`. Starting is always an explicit user step.

#### 3. Start the sprint

```bash
shark sprint start S005
```

Or let `/run-sprint` offer to start it if the sprint is still in `planning` status.

#### 4a. Execute — solo (one entity at a time)

```bash
/run-sprint S005
```

The pull-loop:
1. Calls `shark sprint next S005 --json` to get the next unstarted entity
2. Delegates the entity to `/run {KEY}` — which drives it all the way to terminal status via the standard orchestrator loop
3. Loops until `shark sprint next` returns null (backlog drained) or `--max-iterations` is hit
4. Prints a burndown + summary report
5. **Asks before closing** — never auto-closes

Useful flags:
```bash
/run-sprint S005 --agent=backend       # only pull backend-type entities
/run-sprint S005 --max-iterations=5    # stop after 5 entities, leave sprint open
/run-sprint S005 --carryover=next      # pre-set carryover mode for the close prompt
```

#### 4b. Execute — team (parallel agents per feature)

```bash
/run-sprint-team S005
```

The team dispatch loop:
1. Reads `shark sprint backlog S005 --json`
2. Groups tasks by feature key (`E##-F##`); bugs/CCs/TDs go to a standalone list
3. For each feature group: dispatches `/run-agent-team {FEATURE_KEY}` and **waits** before moving on — only one agent team active at a time
4. Dispatches standalones sequentially via `/run`
5. Prints a burndown between each feature group
6. **Asks before closing**

```bash
/run-sprint-team S005 --size=4                        # team of 4 per feature
/run-sprint-team S005 --features=E07-F01,E07-F02      # only those two features
/run-sprint-team S005 --carryover=backlog             # pre-set carryover for close
```

> **Prerequisites for `/run-sprint-team`**: `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` must be set in `~/.claude/settings.json`, Claude Code ≥ 2.1.32, and a clean git branch. The skill checks all preconditions before dispatching anything.

#### 5. Close the sprint

Both `/run-sprint` and `/run-sprint-team` prompt you to close at the end. You can also close manually:

```bash
shark sprint close S005 --carryover=next     # move incomplete work to next sprint
shark sprint close S005 --carryover=backlog  # drop assignments back to the backlog
```

#### 6. Retrospective

```bash
/retro-sprint S005
```

The skill reads `shark sprint summary S005 --detailed --json` and `shark sprint velocity --json`, then produces a five-section markdown retro at `docs/sprints/S005-retro.md`:

- **Outcome** — planned vs. completed entity count and size points
- **Velocity Context** — this sprint vs. trailing average with trend
- **Carryover Analysis** — per-entity notes for everything that carried or was rejected
- **Cycle-Time Highlights** — phase-by-phase breakdown (requires `--detailed`)
- **Recommendations** — 3–5 items, each citing a quantitative threshold from the data (velocity variance, XL task count, carryover rate, phase imbalance, agent overallocation)

Flags:
```bash
/retro-sprint S005 --no-write    # print to stdout instead of writing a file
```

After the report, the skill optionally offers to archive the sprint. You must confirm — it never auto-archives.

### Choosing `/run-sprint` vs. `/run-sprint-team`

| Situation | Use |
|---|---|
| Solo developer, sequential work | `/run-sprint` |
| Multiple features with parallel task work | `/run-sprint-team` |
| Only bugs/CCs/TDs in the sprint (no features) | `/run-sprint` — team skill routes standalones the same way |
| You want a capacity cap or agent-type filter | `/run-sprint --agent=backend` |
| Agent-teams env var not set | `/run-sprint` (team skill requires it) |

### Safety Guarantees

- **`/plan-sprint` never starts a sprint.** Only `shark sprint start` or user confirmation inside `/run-sprint` starts one.
- **No auto-close.** Both execution skills prompt before calling `shark sprint close`.
- **No auto-archive.** `/retro-sprint` prompts before calling `shark sprint archive`.
- **All shark mutation calls confirm first.** `sprint add`, `sprint start`, `sprint close`, `sprint archive` are all gated on explicit user confirmation.
- **All shark calls use `--json`** for machine-readable output and stable parsing.

---

## Overview

Sprints are first-class planning containers in Shark. The sprint command family uses `S###` keys and covers the full workflow:

- create and configure a sprint
- assign and remove work from the sprint
- inspect planning backlog, capacity, and readiness
- manage per-sprint capacity and team defaults
- review sprint velocity, burndown, and summary reports

See the [Sizing Guide](sizing.md) for detailed benchmarks on estimating work for AI agents vs. humans.

## Quick Reference

| Subcommand | Purpose | Notes |
|---|---|---|
| `shark sprint create` | Create a new sprint | Requires `--start` and `--end` |
| `shark sprint get` | View sprint details | Shows dates, status, goal, and metadata |
| `shark sprint list` | List sprints | Optional `--status` filter |
| `shark sprint update` | Update sprint fields | Supports name, goal, and end date |
| `shark sprint delete` | Delete a sprint | Only planning sprints can be deleted |
| `shark sprint start` | Start a sprint | Moves the sprint into `active` |
| `shark sprint close` | Close a sprint with carryover | Supports `--carryover=next|backlog` |
| `shark sprint archive` | Archive a sprint | Moves the sprint to `archived` |
| `shark sprint add` | Assign one entity or many entities | Supports bulk assignment flags |
| `shark sprint remove` | Remove an entity from a sprint | Soft-removes the assignment |
| `shark sprint backlog` | Show assigned work grouped by status | Supports type and blocked filters |
| `shark sprint plan` | Show planning view | Backlog, capacity, and readiness in one view |
| `shark sprint readiness` | Show readiness score | 0-100 score with factor breakdown |
| `shark sprint capacity set` | Set sprint or default capacity | Use `--default` to update config defaults |
| `shark sprint capacity show` | Show capacity utilization | Displays capacity, allocated, remaining, unsized |
| `shark sprint velocity` | Show velocity history | Defaults to the last 5 completed sprints |
| `shark sprint burndown` | Show burndown chart | Uses the active sprint when omitted |
| `shark sprint summary` | Show sprint summary report | Supports `--detailed` |

## Global Flags

All sprint commands support these global flags:

| Flag | Description |
|---|---|
| `--json` | Output machine-readable JSON |
| `--field <name>` | Extract a single field from JSON output |
| `--db <path>` | Override the database path |
| `--config <path>` | Override the config path |
| `--no-color` | Disable colored output |
| `--verbose` / `-v` | Enable verbose logging |

See [Global Flags](global-flags.md) for the shared behavior across the CLI.

## Lifecycle Commands

### `shark sprint create`

Create a new sprint with an auto-generated `S###` key.

**Usage**

```bash
shark sprint create "<name>" --start=YYYY-MM-DD --end=YYYY-MM-DD [--goal=text]
```

**Flags**

| Flag | Description |
|---|---|
| `--start <date>` | Sprint start date in `YYYY-MM-DD` format |
| `--end <date>` | Sprint end date in `YYYY-MM-DD` format |
| `--goal <text>` | Optional sprint goal |

**Examples**

```bash
shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01
shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01 --goal="Complete auth work"
shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01 --json
```

### `shark sprint get`

Show sprint details by key.

**Usage**

```bash
shark sprint get <key>
```

**Examples**

```bash
shark sprint get S024
shark sprint get S024 --json
shark sprint get S024 --field status
```

### `shark sprint list`

List all sprints, optionally filtered by status.

**Usage**

```bash
shark sprint list [--status=<status>]
```

**Flags**

| Flag | Description |
|---|---|
| `--status <status>` | Filter by sprint status |

**Examples**

```bash
shark sprint list
shark sprint list --status=planning
shark sprint list --status=active --json
```

### `shark sprint update`

Update sprint metadata.

**Usage**

```bash
shark sprint update <key> [--name=text] [--goal=text] [--end=YYYY-MM-DD]
```

**Examples**

```bash
shark sprint update S024 --goal="Finish template enrichment"
shark sprint update S024 --name="Sprint 24 (Extended)" --end=2026-04-08
shark sprint update S024 --json
```

### `shark sprint delete`

Delete a sprint.

Only sprints in `planning` status can be deleted. The command asks for confirmation unless `--force` is provided.

**Usage**

```bash
shark sprint delete <key> [--force]
```

**Examples**

```bash
shark sprint delete S024
shark sprint delete S024 --force
shark sprint delete S024 --force --json
```

### `shark sprint start`

Start a sprint.

This transitions the sprint to `active`. Only one sprint can be active at a time.

**Usage**

```bash
shark sprint start <key>
```

**Examples**

```bash
shark sprint start S024
shark sprint start S024 --json
```

### `shark sprint close`

Close a sprint and handle incomplete work with carryover.

If `--carryover` is omitted, Shark reads the default from `sprint_defaults.carryover_behavior` in `.sharkconfig.json`.

**Usage**

```bash
shark sprint close <key> [--carryover=next|backlog]
```

**Carryover modes**

| Mode | Behavior |
|---|---|
| `next` | Move incomplete work to the next planning sprint; create one if needed |
| `backlog` | Remove incomplete assignments so work returns to the backlog |

**Examples**

```bash
shark sprint close S024
shark sprint close S024 --carryover=next
shark sprint close S024 --carryover=backlog
shark sprint close S024 --json
```

### `shark sprint archive`

Archive a sprint after it has been completed.

**Usage**

```bash
shark sprint archive <key>
```

**Examples**

```bash
shark sprint archive S024
shark sprint archive S024 --json
```

## Assignment Commands

### `shark sprint add`

Assign one entity to a sprint, or bulk-assign many entities.

Single-entity assignment accepts either a positional entity key or `--entity`.

Bulk assignment supports:

- `--bulk=<feature-key>` for all eligible tasks in a feature
- `--bulk-bugs` for all eligible bugs
- `--bulk-tech-debt` for all eligible tech-debt items
- `--bulk-changes` for all eligible change-cards

**Usage**

```bash
shark sprint add <sprint-key> <entity-key>
shark sprint add <sprint-key> --entity=<entity-key>
shark sprint add <sprint-key> --bulk=<feature-key>
shark sprint add <sprint-key> --bulk-bugs
shark sprint add <sprint-key> --bulk-tech-debt
shark sprint add <sprint-key> --bulk-changes
```

**Behavior**

- Single adds can emit a capacity warning, but the assignment still succeeds.
- Bulk adds are transactional per bulk group.
- Bulk output summarizes added and skipped counts, plus any capacity warnings.

**Examples**

```bash
shark sprint add S024 E07-F01-001
shark sprint add S024 --entity=B001
shark sprint add S024 --bulk=E07-F34
shark sprint add S024 --bulk-bugs --json
```

### `shark sprint remove`

Remove an entity from a sprint.

**Usage**

```bash
shark sprint remove <sprint-key> <entity-key>
```

**Examples**

```bash
shark sprint remove S024 E07-F01-001
shark sprint remove S024 B001
shark sprint remove S024 CC-001 --json
```

### `shark sprint backlog`

Show all entities assigned to a sprint, grouped by status category.

Use `--type` to focus on one entity type, or `--blocked` to show only blocked items.

**Usage**

```bash
shark sprint backlog <sprint-key> [--type=task|bug|change_card|tech_debt] [--blocked]
```

**Flags**

| Flag | Description |
|---|---|
| `--type <type>` | Filter by entity type |
| `--blocked` | Show only blocked items |

**Examples**

```bash
shark sprint backlog S024
shark sprint backlog S024 --type=task
shark sprint backlog S024 --blocked
shark sprint backlog S024 --json
```

## Planning and Capacity Commands

### `shark sprint plan`

Show the composite planning view for a sprint.

The planning view combines:

- unassigned backlog eligible for sprint assignment
- capacity utilization by agent type
- readiness score with factor breakdown

**Usage**

```bash
shark sprint plan <sprint-key>
```

**Examples**

```bash
shark sprint plan S024
shark sprint plan S024 --json
```

### `shark sprint readiness`

Show the sprint readiness score.

The readiness report includes the overall score, per-factor breakdown, and lists of unsized and oversized entities.

**Usage**

```bash
shark sprint readiness <sprint-key>
```

**Examples**

```bash
shark sprint readiness S024
shark sprint readiness S024 --json
```

### `shark sprint capacity set`

Set capacity for a sprint, or update the team default when `--default` is used.

Without `--default`, the command updates the `sprint_capacity` row for the sprint.

With `--default`, the command updates `sprint_defaults.capacity` in `.sharkconfig.json` and does not touch the database.

**Usage**

```bash
shark sprint capacity set [sprint-key] --agent=<type> --points=<value> [--default]
```

**Flags**

| Flag | Description |
|---|---|
| `--agent <type>` | Agent type to configure, such as `backend`, `frontend`, or `qa` |
| `--points <value>` | Capacity in story points; must be greater than 0 |
| `--default` | Write the value to `.sharkconfig.json` instead of the database |

**Examples**

```bash
shark sprint capacity set S024 --agent=backend --points=21
shark sprint capacity set --default --agent=backend --points=21
shark sprint capacity set --default --agent=frontend --points=13 --json
```

### `shark sprint capacity show`

Show capacity versus allocation for a sprint.

The table includes:

- `capacity_points`
- `allocated_points`
- `remaining`
- `unsized_assigned`

If no capacity rows exist, the command prints an informational message instead of failing.

**Usage**

```bash
shark sprint capacity show <sprint-key>
```

**Examples**

```bash
shark sprint capacity show S024
shark sprint capacity show S024 --json
```

## Reporting Commands

### `shark sprint velocity`

Show velocity history for recent completed sprints.

Velocity is based on completed story-point size, with unsized work reported separately.

**Usage**

```bash
shark sprint velocity [--sprints=<n>]
```

**Flags**

| Flag | Description |
|---|---|
| `--sprints <n>` | Number of completed sprints to include, default `5` |

**Examples**

```bash
shark sprint velocity
shark sprint velocity --sprints=10
shark sprint velocity --json
```

### `shark sprint burndown`

Show a burndown chart for a sprint.

If you omit the sprint key, Shark uses the current active sprint.

**Usage**

```bash
shark sprint burndown [sprint-key]
```

**Examples**

```bash
shark sprint burndown
shark sprint burndown S024
shark sprint burndown S024 --json
```

### `shark sprint summary`

Show a sprint summary report.

The summary is available for completed or archived sprints. Use `--detailed` for cycle-time, size-band, and carryover breakdowns.

**Usage**

```bash
shark sprint summary <sprint-key> [--detailed]
```

**Flags**

| Flag | Description |
|---|---|
| `--detailed` | Include extended retrospective analytics |

**Examples**

```bash
shark sprint summary S024
shark sprint summary S024 --detailed
shark sprint summary S024 --json
```

## Output Notes

- Human-readable planning and capacity tables are optimized for terminal use.
- `--json` is the preferred mode for scripts and orchestrators.
- `--field` works on sprint commands the same way it works on other Shark commands.
- For sprint planning workflows, `shark sprint plan`, `shark sprint readiness`, and `shark sprint capacity show` are the most useful machine-readable entry points.

## Related Documentation

- [Global Flags](global-flags.md)
- [JSON Output Format](json-output.md)
- [Workflow Configuration](workflow-configuration.md)
- [Configuration](configuration.md)
- [Progress and Analytics Commands](progress-analytics.md)
