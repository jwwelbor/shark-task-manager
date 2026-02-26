# CX Review: Shark CLI for AI Agent Users

**Date:** 2026-02-25
**Reviewer:** CXDesigner Agent
**Scope:** CLI command taxonomy, flag consistency, AI agent ergonomics
**Evidence Base:** Activity log from wormwoodGM project (250+ real AI agent interactions), CLI reference docs, command implementations

---

## 1. Current State Analysis

### 1.1 Command Surface Area

The current CLI has approximately 40+ distinct command paths. The major categories are:

| Category | Commands | Notes |
|----------|----------|-------|
| Smart dispatchers | `list`, `get`, `status`, `history` | Auto-detect entity type from key format |
| Entity CRUD (noun-first) | `epic create/list/get`, `feature create/list/get`, `task create/list/get` | Duplicates smart dispatchers for list/get |
| Task lifecycle | `task start/complete/approve/reopen/block/unblock` | Task-only workflow commands |
| Status manipulation | `task update --status`, `task next-status`, `task set-status`, `feature update --status`, `epic update --status`, `feature next-status`, `epic next-status`, `feature set-status`, `epic set-status` | Fragmented across 9 commands |
| Context/resume | `epic context`, `feature context`, `epic resume`, `feature resume` | Entity-specific |
| Notes | `epic note`, `feature note`, `task note` | Per-entity |
| Config/admin | `init`, `init update`, `config get/set`, `cloud init/status`, `migrate slugs`, `workflow show/validate` | Setup and maintenance |

**Total unique command paths: ~45+**

### 1.2 Observed Pain Points from Activity Log

The activity log reveals several critical UX failures when real AI agents interact with Shark.

#### Pain Point 1: Status Transition Confusion (CRITICAL)

Agents repeatedly try non-existent commands and fall back through multiple alternatives:

```bash
# Agent tries 3-4 different commands to change status:
shark task set-status T-E18-F05-002 ready_for_development 2>&1 ||
shark task update T-E18-F05-002 --status ready_for_development 2>&1

# Another agent tries yet another pattern:
shark task update T-E18-F05-007 --status ready_for_refinement_tech 2>&1 ||
shark task update T-E18-F05-007 --status refinement_tech 2>&1

# Agent falls back to piping help:
shark task set-status T-E18-F05-006 ready_for_development 2>/dev/null ||
shark task update T-E18-F05-006 --status "ready for development" 2>/dev/null ||
echo "Checking available status commands"; shark task --help 2>/dev/null | head -30
```

**Root cause:** There are at least 4 different ways to change status (`update --status`, `next-status`, `set-status`, `start/complete/approve`), and agents cannot predict which one exists or works.

#### Pain Point 2: Excessive JSON Post-Processing (HIGH)

Agents pipe `--json` output through Python to extract single fields:

```bash
shark task get T-E18-F05-004 --json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('status'))"
shark task get T-E18-F05-001 --json | python3 -c "import json,sys; d=json.load(sys.stdin); print('status:', d['status'])"
shark feature get E18-F05 --json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('path'))"
```

**Root cause:** No `--field` or `--format` flag to extract specific values. Agents need to invoke Python as a JSON parser on every call.

#### Pain Point 3: Defensive Error Handling (HIGH)

Agents wrap nearly every command in error suppression:

```bash
shark feature get E18-F05 --json 2>/dev/null || shark feature get E18-F05
shark task get T-E18-F05-003 --json 2>/dev/null || echo "SHARK_ERROR"
shark task next-status T-E18-F05-001 --preview 2>/dev/null || echo "no preview flag"
```

**Root cause:** Agents do not trust that commands will behave predictably. Error messages go to stderr mixed with output, and exit codes may not be reliable.

#### Pain Point 4: Batch Operations Are Manual Loops (MEDIUM)

Agents write bash for-loops to operate on multiple entities:

```bash
for t in T-E18-F05-001 T-E18-F05-002 ... T-E18-F05-009; do
  shark task update $t --status in_qa
done

for t in 001 002 003 004 005 006 007 008 009; do
  shark task next-status T-E18-F05-$t --status in_refinement_ba --force 2>&1 | tail -1
done
```

**Root cause:** No batch mode. Every status change requires a separate process invocation.

#### Pain Point 5: next-status Command Ergonomics (MEDIUM)

The `next-status` command is the most-used status transition command, but its behavior is confusing:
- `--preview` to see what will happen
- `--force` to skip confirmation
- `--status` to target a specific next status (but why is it called "next" if you specify the target?)

Agents frequently chain: `next-status --preview` then `next-status --force`, doubling the number of invocations.

### 1.3 Command Frequency Analysis

From the activity log, the most common operations by frequency:

1. **Get entity details** (`shark task get ... --json`) - ~40% of all commands
2. **Change status** (`shark task update --status`, `shark task next-status --force`) - ~35%
3. **Get feature overview** (`shark feature get ... --json`) - ~10%
4. **List entities** (`shark list`, `shark feature get`) - ~5%
5. **Create entities** - ~3%
6. **Other** (config, history, context) - ~7%

This tells us: **75% of all AI agent interactions are either reading an entity or changing its status.** The CLI should be optimized for these two operations above all else.

---

## 2. Recommended Command Taxonomy

### 2.1 Design Principles for AI Agent CLI

1. **One obvious way to do everything.** No aliases, no "also available as" alternatives.
2. **Verb-first, auto-detect entity.** Since 75% of operations take an ID, let the ID format drive dispatch.
3. **JSON is the primary output mode.** Default to `--json` or provide `SHARK_OUTPUT=json` env var.
4. **Predictable flag patterns.** Same flags mean the same thing on every command.
5. **Machine-parseable field extraction.** `--field status` returns raw value, no JSON wrapper.
6. **Batch-native.** Accept multiple IDs wherever possible.
7. **Idempotent where possible.** Setting status to its current value should succeed silently.

### 2.2 Proposed Command Tree

```
shark
  # --- Core CRUD (auto-detect from ID) ---
  get <id> [--json] [--field <name>]
  list [parent-id] [--status <s>] [--agent <a>] [--json] [--field <name>]
  create epic "<title>" [--priority N] [--file path]
  create feature <epic-id> "<title>" [--order N] [--file path]
  create task <feature-id> "<title>" [--agent <type>] [--priority N] [--order N] [--file path]
  delete <id> [--force]
  update <id> [--title "..."] [--priority N] [--agent <type>] [--order N]

  # --- Status (the #1 AI agent operation) ---
  advance <id> [--to <status>] [--force] [--notes "..."]
  set-status <id> <status> [--force] [--notes "..."]

  # --- Task-specific workflow shortcuts ---
  next [--agent <type>] [--epic <key>] [--json]
  block <id> --reason "..."
  unblock <id>

  # --- Information ---
  status <id> [--json]
  history <id> [--json]
  context <id> [--json]

  # --- Notes ---
  note <id> "<text>" [--type rejection|comment|review]
  note list <id> [--json]

  # --- Admin (infrequent) ---
  init [--workflow basic|advanced] [--non-interactive]
  config get <key>
  config set <key> <value>
  cloud init --url "..." --auth-token "..."
  cloud status [--json]
  workflow show [--json]
  migrate slugs
```

**Total command paths: ~25** (down from ~45)

### 2.3 Key Design Decisions

#### Decision 1: Merge `get`/`status`/`history`/`context` vs Keep Separate

**Recommendation: Keep separate but add `--field` to `get`.**

Rationale: `get` returns the entity data. `status` returns progress/health rollups. `history` returns changelog. These are different response shapes. However, `get` should include a `valid_transitions` field in its JSON output (it appears to already), eliminating the need for `--preview` on advance.

#### Decision 2: `advance` vs `next-status` vs `update --status`

**Recommendation: Two commands instead of four.**

- `shark advance <id>` - Move to the next logical status in the workflow. No arguments needed for the common case. Equivalent to current `next-status --force`.
- `shark set-status <id> <status>` - Jump to a specific status. Replaces `update --status`, `set-status`, and `next-status --status X --force`.

Remove: `task start`, `task complete`, `task approve`, `task reopen`. These are just `advance` or `set-status` with hardcoded target statuses. The workflow engine already knows what "advance" means.

**Backward compatibility:** Keep old commands as hidden aliases for 2 major versions.

#### Decision 3: `--field` for Targeted Extraction

Add `--field <name>` to `get` and `list`:

```bash
# Instead of:
shark task get T-E18-F05-001 --json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['status'])"

# Write:
shark get T-E18-F05-001 --field status
# Output: ready_for_development

shark get T-E18-F05-001 --field valid_transitions
# Output: in_development,blocked
```

This eliminates the #1 pattern of AI agents piping through Python.

#### Decision 4: Batch Mode

Add multi-ID support to `advance`, `set-status`, and `get`:

```bash
# Instead of:
for t in T-E18-F05-001 T-E18-F05-002 T-E18-F05-003; do
  shark task update $t --status in_qa
done

# Write:
shark set-status T-E18-F05-001 T-E18-F05-002 T-E18-F05-003 in_qa

# Or even:
shark set-status --feature E18-F05 in_qa   # All tasks in feature
```

#### Decision 5: JSON Default for Agent Mode

Add `SHARK_OUTPUT=json` environment variable:

```bash
export SHARK_OUTPUT=json
shark get E18-F05-001  # Returns JSON without --json flag
```

This avoids agents needing to remember `--json` on every single call.

---

## 3. Flag Consistency Matrix

### 3.1 Current Inconsistencies

| Flag | epic create | feature create | task create | Notes |
|------|------------|----------------|-------------|-------|
| Title | `--title="..."` | positional arg 2 | positional arg 3 | Inconsistent |
| Parent | N/A | positional arg 1 (`<epic>`) | positional args 1+2 (`<epic> <feature>`) | Different arity |
| File | `--file` | `--file` | `--file` | Consistent |
| Force | `--force` | `--force` | `--force` | Consistent |
| Priority | `--priority` | N/A | `--priority` | Missing on feature |
| Order | N/A | `--execution-order` | `--order` | Different flag names |
| Agent | N/A | N/A | `--agent` | Task-only |
| JSON | `--json` | `--json` | `--json` | Consistent |

### 3.2 Proposed Consistent Flag Set

**Universal flags (all commands):**
| Flag | Short | Description |
|------|-------|-------------|
| `--json` | `-j` | JSON output |
| `--field` | `-f` | Extract single field (requires --json behavior) |
| `--verbose` | `-v` | Debug logging |
| `--no-color` | | Disable ANSI colors |

**Entity flags (create/update):**
| Flag | Applies to | Description |
|------|-----------|-------------|
| `--title` | all | Entity title (also positional) |
| `--priority` | epic, task | Priority 1-10 |
| `--order` | feature, task | Execution order (replaces `--execution-order`) |
| `--agent` | task | Agent type |
| `--file` | all | Custom file path |
| `--force` | all | Force operation |

**Filter flags (list):**
| Flag | Description |
|------|-------------|
| `--status` | Filter by status |
| `--agent` | Filter by agent type |
| `--all` | Include completed/cancelled (replaces `--show-all`) |

**Status flags (advance/set-status):**
| Flag | Description |
|------|-------------|
| `--to` | Target status (advance only) |
| `--force` | Skip confirmation |
| `--notes` | Transition notes |
| `--reason` | Block reason (block only) |

---

## 4. Create Command Special Cases

Creation is the one operation where auto-detection cannot work (no ID exists yet). The proposed syntax:

```bash
# Epics (no parent)
shark create epic "Epic Title" [--priority N]

# Features (epic parent)
shark create feature E07 "Feature Title" [--order N]

# Tasks (feature parent - accept both formats)
shark create task E07-F01 "Task Title" [--agent backend] [--priority 5] [--order 1]
shark create task E07 F01 "Task Title" [--agent backend]   # 3-arg also works
```

**Key simplification:** Always `shark create <entity-type> [parent] "title" [flags]`. No legacy flag syntax (`--epic=E07 --feature=F01 --title="..."`).

---

## 5. Notes and Relationship Commands

### Notes (Current: `epic note`, `feature note`, `task note`)

**Proposed:** Single `shark note` command with auto-detection:

```bash
shark note E18-F05-001 "Code review feedback"
shark note E18-F05-001 "Failed QA" --type rejection
shark note list E18-F05-001 [--json]
```

### Related Docs

The activity log shows agents trying `shark related-docs add ...` which appears to be a rarely-used command. Keep it as-is but under the admin category.

---

## 6. Binary Split Recommendation

### Should Shark be split into multiple binaries?

**Recommendation: No.**

Rationale:
- AI agents already struggle with one CLI. Adding `shark-config` and `shark-analytics` would triple the command surface they need to learn.
- The admin commands (`init`, `config`, `cloud`, `workflow`, `migrate`) are used once per project setup. They do not pollute the daily workflow namespace.
- A single binary is simpler to install, version, and distribute.

**Alternative: Use subcommand grouping** instead of separate binaries:

```bash
shark admin init [...]
shark admin config get/set
shark admin cloud init/status
shark admin workflow show/validate
shark admin migrate slugs
```

This visually separates admin commands without requiring multiple binaries. Agents doing daily work never need `shark admin`.

---

## 7. Error Handling for AI Agents

### 7.1 Structured Error Output

When `--json` is active, errors should also be JSON:

```json
{
  "error": true,
  "code": "INVALID_TRANSITION",
  "message": "Cannot transition from 'todo' to 'completed'. Valid transitions: in_progress, blocked",
  "entity": "T-E18-F05-001",
  "current_status": "todo",
  "valid_transitions": ["in_progress", "blocked"]
}
```

This eliminates the need for agents to parse stderr text.

### 7.2 Exit Code Consistency

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Operation completed |
| 1 | Not found | Entity does not exist |
| 2 | System error | Database failure, IO error |
| 3 | Invalid state | Workflow violation, validation failure |
| 4 | Conflict | Duplicate key, concurrent modification |

### 7.3 Idempotent Operations

`set-status` should succeed if the entity is already in the target status (exit 0, not exit 3). This eliminates defensive `2>/dev/null` wrappers agents currently use.

---

## 8. Migration Strategy

### Phase 1: Add Without Removing (v-next)

1. Add `shark advance <id>` as alias for `shark task next-status <id> --force`.
2. Add `shark set-status <id> <status>` as unified status setter.
3. Add `--field` flag to `get` and `list`.
4. Add `SHARK_OUTPUT=json` environment variable support.
5. Add JSON error output when `--json` is active.
6. Make `set-status` idempotent.

**Zero breaking changes.** All old commands still work.

### Phase 2: Promote New Commands (v-next+1)

1. Update all documentation to use new command forms.
2. Add deprecation warnings to old command forms (printed to stderr only when not `--json`).
3. Add `shark create` as unified create dispatcher.
4. Add batch mode to `set-status` and `advance`.
5. Move admin commands under `shark admin` subgroup.

### Phase 3: Hide Old Commands (v-next+2)

1. Remove old commands from `--help` output (but keep them functional).
2. Old commands print deprecation notice and delegate to new commands.
3. Update CLAUDE.md and agent instructions to only reference new commands.

### Phase 4: Remove Old Commands (v-next+3)

1. Remove legacy command registrations.
2. Remove `task start/complete/approve/reopen` in favor of `advance`/`set-status`.
3. Remove noun-first `task list/get`, `feature list/get`, `epic list/get` in favor of smart dispatchers.

---

## 9. Summary of Recommendations

### Must-Have (Addresses Critical Pain Points)

| # | Recommendation | Pain Point Addressed | Effort |
|---|---------------|---------------------|--------|
| 1 | Unified `set-status <id> <status>` command | Status transition confusion | M |
| 2 | `--field <name>` flag on `get`/`list` | Python post-processing | S |
| 3 | JSON error output with structured error codes | Defensive error handling | S |
| 4 | Idempotent `set-status` (no error if already at target) | Defensive `2>/dev/null` | S |
| 5 | `SHARK_OUTPUT=json` environment variable | Forgetting `--json` flag | XS |

### Should-Have (Reduces Friction)

| # | Recommendation | Pain Point Addressed | Effort |
|---|---------------|---------------------|--------|
| 6 | `shark advance <id>` (workflow-aware next step) | next-status verbosity | S |
| 7 | Batch mode for status changes | Manual for-loops | M |
| 8 | Unified `shark create epic/feature/task` | Inconsistent create syntax | M |
| 9 | Normalize `--order` flag name (retire `--execution-order`) | Flag inconsistency | XS |

### Nice-to-Have (Polish)

| # | Recommendation | Impact | Effort |
|---|---------------|--------|--------|
| 10 | `shark admin` subgroup for config/init/cloud | Cleaner help output | S |
| 11 | Deprecation warnings on old command forms | Migration guidance | S |
| 12 | Hide noun-first commands from help | Reduced cognitive load | XS |
| 13 | `shark note <id> "text"` unified notes | Consistency | S |

---

## 10. Proposed Ideal Agent Workflow

After implementing the recommendations, a typical AI agent workflow session would look like:

```bash
# Set JSON mode once
export SHARK_OUTPUT=json

# Get next task
TASK=$(shark next --agent developer --field key)

# Read task details
shark get $TASK

# Start working
shark advance $TASK

# Check status after work
shark get $TASK --field status
# Output: in_development

# Mark ready for review
shark advance $TASK --notes "Implementation complete"

# Batch advance all tasks in a feature after code review
shark set-status --feature E18-F05 ready_for_qa

# Check feature progress
shark status E18-F05
```

Compare to the current equivalent:

```bash
# Get next task
TASK=$(shark task next --agent developer --json 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['key'])")

# Read task details
shark task get $TASK --json

# Start working
shark task next-status $TASK --force

# Check status after work
shark task get $TASK --json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('status'))"

# Mark ready for review
shark task next-status $TASK --force --notes "Implementation complete"

# Batch advance all tasks
for t in T-E18-F05-001 T-E18-F05-002 ... T-E18-F05-009; do
  shark task update $t --status ready_for_qa 2>&1 || shark task next-status $t --force --status ready_for_qa 2>&1
done

# Check feature progress
shark feature get E18-F05 --json
```

**Reduction: 60% fewer characters, no Python dependency, no error suppression, no for-loops.**

---

## Appendix A: Activity Log Command Frequency

Extracted from `/home/jwwel/projects/wormwoodGM/docs/workflow/activity.jsonl`:

| Command Pattern | Count | Notes |
|----------------|-------|-------|
| `shark task get ... --json` | ~60 | Most common single command |
| `shark task next-status ... --force` | ~40 | Most common status change |
| `shark task update ... --status` | ~25 | Second most common status change |
| `shark feature get ...` | ~15 | Feature overview |
| `shark task next-status ... --preview` | ~12 | Preview before advance |
| `shark task set-status ...` | ~8 | Attempted command (may not exist) |
| `shark task get ... --json \| python3` | ~30 | JSON field extraction |
| `for t in ...; do shark task ...; done` | ~12 | Batch operations |

## Appendix B: File References

- CLI Reference: `/home/jwwel/projects/shark-task-manager/docs/CLI_REFERENCE.md`
- Quick Reference: `/home/jwwel/projects/shark-task-manager/.claude/rules/quickref.md`
- Command Definitions: `/home/jwwel/projects/shark-task-manager/.claude/rules/cli/commands.md`
- Activity Log: `/home/jwwel/projects/wormwoodGM/docs/workflow/activity.jsonl`
- Command Implementations: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/`
