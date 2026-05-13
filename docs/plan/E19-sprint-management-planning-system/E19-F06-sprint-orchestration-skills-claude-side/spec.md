# Spec: E19-F06 — Sprint Orchestration Skills (Claude-side)

**Feature Key**: E19-F06-sprint-orchestration-skills-claude-side
**Epic**: [E19 — Sprint Management & Planning System](../../epic.md)
**Type**: Claude-side skills (no Go code in shark repo)

---

## Context

See epic PRD ([epic.md](../../epic.md)) for problem framing, personas, and journey maps. See [requirements.md](../../requirements.md) for the full sprint command surface this feature consumes.

This feature is **Claude-side only**: it adds slash commands and skill files under `~/.claude/commands/` and `~/.claude/skills/` that wrap the `shark sprint` CLI surface delivered by E19-F01 through E19-F05. No Go code, no shark schema, no migrations. The CLI is a black-box dependency consumed via `shark sprint …` invocations.

The four skills (`/plan-sprint`, `/run-sprint`, `/run-sprint-team`, `/retro-sprint`) and the cross-cutting edits in T-E19-F06-005 collectively let an AI orchestrator (or a human PM via Claude) drive a sprint end-to-end without leaving Claude. Today an orchestrator can drive a single feature via `/run E##-F##` but has no first-class concept of a sprint as a dispatch boundary; this feature closes that gap.

---

## Requirements

Numbering continues the epic's `REQ-F-###` sequence (epic ends at REQ-F-018). All requirements below are **incremental** to the epic — they describe the Claude-side wrappers, not the underlying CLI behavior (which is owned by F01–F05).

### Functional Requirements

#### Skill: `/plan-sprint`

**REQ-F-019**: `/plan-sprint` mode-aware planner skill
- **Description**: A slash command that takes a sprint key and walks an orchestrator (or human via Claude) through scoping the sprint by reading `shark sprint plan` and `shark sprint readiness` and proposing entity assignments.
- **Trace**: Implements the Claude-side surface for journey "Sprint Planning" (epic user-journeys.md, journey 2). Backed by REQ-F-011 (`shark sprint plan`), REQ-F-013 (readiness), REQ-F-004 (assignment), REQ-F-012 (bulk assignment).
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `/plan-sprint S###` is registered in `~/.claude/commands/plan-sprint.md` and resolves to the skill at `~/.claude/skills/sprint-planning/SKILL.md`
  - [ ] Refuses non-sprint keys with: `"/plan-sprint only operates on sprints. Got: {KEY}"`
  - [ ] If sprint is not in `planning` status, prints status and asks user whether to continue (advisory; planning a non-planning sprint is allowed but flagged)
  - [ ] Reads `shark sprint plan {S###} --json` once at start; never re-reads in the same turn unless the user has run a `shark sprint add/remove`
  - [ ] Selects between two **modes** based on the `--mode` flag (default = `interactive`):
    - `interactive` — surfaces the readiness breakdown, lists oversize/unsized entities, and asks the user which backlog items to add. Only invokes `shark sprint add` after explicit user confirmation per item or per group.
    - `auto` — selects backlog entities greedily up to capacity per agent type using readiness factors, then prints the proposed plan and asks for one final confirmation before applying. Never auto-applies without confirmation.
  - [ ] On exit, runs `shark sprint readiness {S###} --json` again and reports the score delta
  - [ ] Does NOT call `shark sprint start` — starting a sprint is an explicit user action handled by `/run-sprint`
  - [ ] All shark calls pass `--json`; never truncates JSON with `head`/`tail`/`grep`

#### Skill: `/run-sprint`

**REQ-F-020**: `/run-sprint` solo sprint pull-loop skill
- **Description**: A slash command that drives an `active` sprint to completion sequentially by repeatedly calling `shark sprint next` and dispatching `/run` against the returned entity, then closing the sprint.
- **Trace**: Implements the Claude-side surface for journey "Sprint Monitoring" (epic user-journeys.md, journey 3). Backed by `shark sprint next` (delivered with E19-F03), REQ-F-006 (close + carryover), and the existing `/run` dispatch loop.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `/run-sprint S###` is registered in `~/.claude/commands/run-sprint.md` and resolves to the skill at `~/.claude/skills/sprint-execution/SKILL.md` (or a `workflows/run-sprint.md` under that skill)
  - [ ] Refuses non-sprint keys with the same message pattern as `/plan-sprint`
  - [ ] If sprint is in `planning` status, asks user whether to call `shark sprint start {S###}` first; if user declines, exits
  - [ ] If sprint is in `completed`/`archived`/`cancelled`, exits with a notice (do not silently no-op)
  - [ ] Loop body: `shark sprint next --json` → if entity returned, dispatch `/run {ENTITY_KEY}` and let it drive the entity to terminal status → loop. If `shark sprint next` returns nothing (empty result), loop exits.
  - [ ] Honors a `--agent=<type>` flag that is passed straight through to `shark sprint next --agent=<type>` so a single Claude session can pull only its own agent type's slice
  - [ ] Honors a `--max-iterations=N` cap (default 50) to prevent runaway loops; exits with a notice when cap is reached and reports the sprint state
  - [ ] On loop exit, prints `shark sprint burndown {S###}` and `shark sprint summary {S###}` (basic, not `--detailed`) for visibility, then asks the user whether to close the sprint
  - [ ] If user confirms close, runs `shark sprint close {S###} --carryover=<value>` where `<value>` is read from a `--carryover` flag (defaults to the sharkconfig default if absent)
  - [ ] Never calls `shark sprint close` without explicit user confirmation
  - [ ] Does not bypass the orchestrator — entity dispatch goes through `/run`, which honors `orchestrator_action.instruction`

#### Skill: `/run-sprint-team`

**REQ-F-021**: `/run-sprint-team` parallel sprint execution skill
- **Description**: A slash command that drives an `active` sprint by grouping its assigned entities into per-feature batches and bootstrapping a Claude Code agent team per batch via the existing `/run-agent-team` skill.
- **Trace**: Implements the Claude-side surface for the "team execution" variant of journey "Sprint Monitoring". Backed by `shark sprint backlog --json`, the existing `/run-agent-team` skill, and the agent-teams primitive (https://code.claude.com/docs/en/agent-teams).
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `/run-sprint-team S###` is registered in `~/.claude/commands/run-sprint-team.md` and resolves under `~/.claude/skills/sprint-execution/`
  - [ ] Refuses non-sprint keys with the same message pattern
  - [ ] Inherits all `/run-agent-team` preconditions (env var `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`, claude version ≥ 2.1.32, branch sanity, clean worktree). Failures abort before any team is spawned.
  - [ ] Groups assigned entities by **feature key** (extracted from task keys `E##-F##-###`). Bugs/change-cards/tech-debt without a feature parent form a single residual group keyed by entity type.
  - [ ] For each feature group, runs `/run-agent-team E##-F##` sequentially (NOT in parallel — only one team can exist per Claude session per the agent-teams docs). Tasks **inside** a feature run in parallel via the team primitive; features themselves run serially.
  - [ ] Honors `--size=N` flag, passed through to each `/run-agent-team` invocation
  - [ ] Honors `--features=E##-F##,E##-F##` flag to restrict execution to a subset of features in the sprint (others are skipped)
  - [ ] Residual groups (bugs, change-cards, tech-debt without feature parents) are **not** dispatched via `/run-agent-team` (which refuses non-feature keys per its own contract). The skill instead falls back to sequential `/run` dispatch for those keys.
  - [ ] Between feature groups, runs `shark sprint burndown {S###}` for visibility
  - [ ] On completion of all groups, behaves like `/run-sprint` post-loop: prints summary, asks user whether to close

#### Skill: `/retro-sprint`

**REQ-F-022**: `/retro-sprint` post-close analysis skill
- **Description**: A slash command that produces a retrospective analysis for a closed or archived sprint by aggregating `shark sprint summary --detailed`, `shark sprint velocity`, and per-entity rejection/blocker notes.
- **Trace**: Implements the Claude-side surface for journey "Sprint Close & Retrospective" (epic user-journeys.md, journey 4). Backed by REQ-F-009 (`shark sprint summary`), REQ-F-007 (`shark sprint velocity`), and existing `shark notes` queries.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `/retro-sprint S###` is registered in `~/.claude/commands/retro-sprint.md` and resolves under `~/.claude/skills/sprint-analytics/`
  - [ ] Refuses non-sprint keys
  - [ ] Refuses sprints that are **not** in `completed` or `archived` status with: `"/retro-sprint requires a completed or archived sprint. {S###} is in status: {status}. Close the sprint first with /run-sprint or shark sprint close."`
  - [ ] Reads, in order: `shark sprint summary {S###} --detailed --json`, `shark sprint velocity --json`, then for each entity in the sprint's carryover and rejection lists: `shark notes {ENTITY_KEY}` to surface blocker/rejection notes
  - [ ] Produces a markdown retro report with sections: **Outcome** (planned vs. completed Σ size and count), **Velocity Context** (this sprint vs. trailing average, with delta), **Carryover Analysis** (what carried, why, from the notes), **Cycle-Time Highlights** (per-phase from `--detailed`), **Recommendations** (3–5 actionable items the skill itself generates from the data, e.g., "average XL-task cycle time was 2× M-task — consider splitting XL tasks before sprint planning")
  - [ ] Writes the retro report to `docs/sprints/{S###}-retro.md` (creating the directory if needed). If the file exists, prompts before overwriting.
  - [ ] Honors `--no-write` flag to print the report to stdout without writing to disk
  - [ ] All shark calls pass `--json`

#### Cross-Cutting Edits

**REQ-F-023**: `/run` bypass for sprint-managed entities
- **Description**: When `/run` is invoked against an entity that is currently assigned to an `active` sprint, the existing skill emits a one-line advisory directing the user to `/run-sprint` instead, then continues normally (does not abort).
- **Trace**: Coordination boundary between the existing per-entity orchestrator and the new sprint orchestrator. Prevents the two from silently competing for the same entity.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `~/.claude/skills/orchestration/workflows/run.md` Step 1 ("Read State") is amended to also check `shark get {KEY} --field sprint --json` (or whichever field exposes the active sprint assignment, depending on what F03 ships)
  - [ ] If the entity is assigned to a sprint in `active` or `planning` status, prints a one-line notice: `"Note: {KEY} is in sprint {S###} ({status}). For sprint-aware execution use /run-sprint {S###}. Continuing per-entity execution."`
  - [ ] The notice is informational only — `/run` execution proceeds unchanged
  - [ ] No notice for entities not in any sprint

**REQ-F-024**: `/run-agent-team` documentation cross-reference
- **Description**: `/run-agent-team` documentation is updated to point at `/run-sprint-team` as the sprint-aware entry point, so users finding `/run-agent-team` first don't miss the sprint orchestration option.
- **Trace**: Discoverability requirement.
- **Priority**: Should-Have
- **Acceptance Criteria**:
  - [ ] `~/.claude/commands/run-agent-team.md` adds a one-line "See also: `/run-sprint-team` for sprint-scoped multi-feature execution" reference near the top
  - [ ] `~/.claude/skills/orchestration/workflows/run-agent-team.md` adds a similar reference in its Usage section
  - [ ] No behavior change to `/run-agent-team` itself

**REQ-F-025**: SKILL.md registration and PIPELINE.md update
- **Description**: The new skills are registered in their parent `SKILL.md` index files, and `~/.claude/PIPELINE.md` is updated to mention the sprint orchestration loop as a downstream extension of the SDLC flow.
- **Trace**: Discoverability and documentation requirement.
- **Priority**: Should-Have
- **Acceptance Criteria**:
  - [ ] Three new skill directories exist with `SKILL.md` files: `sprint-planning/`, `sprint-execution/`, `sprint-analytics/` (under `~/.claude/skills/`)
  - [ ] Each `SKILL.md` carries the standard frontmatter (`name`, `description`) and a single-paragraph "What this is" section that mirrors the style of `orchestration/SKILL.md`
  - [ ] `~/.claude/PIPELINE.md` adds a single row to the "What 'Follow SDLC Flow' Covers" table for sprint orchestration, pointing at the four `/sprint-*` commands. No PDLC diagram changes.

**REQ-F-026**: Product-manager agent reference
- **Description**: The `product-manager` agent prompt is updated to mention the sprint orchestration skills as a viable feature-batch dispatch path.
- **Trace**: Agent prompt update so the PM agent doesn't default to per-entity `/run` when a sprint is active.
- **Priority**: Should-Have
- **Acceptance Criteria**:
  - [ ] `~/.claude/agents/product-manager.md` adds a short subsection (under "PRIMARY: Feature Execution Workflow") titled "When a sprint is active" listing the four `/sprint-*` commands and stating that `/run-sprint` (or `/run-sprint-team`) is the preferred entry point when the feature's tasks are assigned to an `active` sprint.
  - [ ] No other agent prompts changed.

### Non-Functional Requirements

**REQ-NF-007**: Idempotency and re-entry safety
- **Description**: All four skills must be safe to re-invoke at any point — no skill may produce a different shark state on a second invocation against the same sprint at the same lifecycle stage, except as a direct result of work performed by the dispatched agents.
- **Measurement**: Each skill is invoked twice in succession against a fixture sprint; second invocation must not duplicate `shark sprint add`, must not double-close, must not re-create the retro file without confirmation.
- **Target**: Zero duplicate side effects.
- **Justification**: Orchestrator loops crash and resume. Skills must tolerate that without corrupting sprint state.

**REQ-NF-008**: User-confirmation for state-changing operations
- **Description**: Every shark mutation that changes sprint status (`start`, `close`, `archive`, `delete`) or that modifies the assignment set in bulk (`add --bulk*`, `remove`) must be confirmed by the user before execution.
- **Measurement**: Manual review of each skill against this rule.
- **Target**: 100% of mutating-and-irreversible-or-bulk shark calls are gated by an explicit user prompt.
- **Justification**: Sprints represent committed work plans. Accidental mutations are expensive to undo.

**REQ-NF-009**: JSON-only shark consumption
- **Description**: All shark invocations from these skills must use `--json` and consume parsed output; no skill may parse human-readable shark output via regex/grep.
- **Measurement**: Code review of skill files.
- **Target**: 100% of shark calls use `--json` (or `--field`).
- **Justification**: Human-readable output formatting is not a stable contract; only `--json` is. Matches existing `/run` pattern.

**REQ-NF-010**: No Go code, no schema, no CLI changes
- **Description**: This feature must not introduce, modify, or depend on changes to the shark Go codebase, database schema, or CLI command surface beyond what F01–F05 deliver.
- **Measurement**: Tasks for this feature touch only files under `~/.claude/`. No PR for this feature includes diffs to `internal/`, `cmd/`, or `docs/cli-reference/` in the shark repo.
- **Target**: Zero shark-repo changes from this feature's tasks.
- **Justification**: Clean separation of concerns. F01–F05 own the CLI; F06 owns the Claude-side wrappers.

### Acceptance Criteria (Feature-Level)

**Scenario 1: Plan a fresh sprint (interactive mode)**
- **Given** sprint S005 in `planning` status with 12 unassigned eligible backlog entities
- **When** user runs `/plan-sprint S005`
- **Then** the skill displays the readiness breakdown and prompts for backlog selection
- **And** assigns entities only after explicit confirmation
- **And** reports the readiness-score delta on exit
- **And** does not call `shark sprint start`

**Scenario 2: Solo execution of an active sprint**
- **Given** sprint S005 in `active` status with 4 assigned entities
- **When** user runs `/run-sprint S005`
- **Then** the skill loops: `shark sprint next` → `/run {ENTITY}` → `shark sprint next` → ...
- **And** stops when `shark sprint next` returns no entity or `--max-iterations` is hit
- **And** prompts the user before calling `shark sprint close`

**Scenario 3: Team execution across multiple features**
- **Given** sprint S005 in `active` status with assigned entities spanning features E07-F01 and E07-F02 plus standalone bug B003
- **When** user runs `/run-sprint-team S005`
- **Then** the skill checks team-mode preconditions, then runs `/run-agent-team E07-F01` to terminal, then `/run-agent-team E07-F02` to terminal, then dispatches B003 via plain `/run` (since `/run-agent-team` only accepts feature keys)
- **And** prompts the user before closing the sprint

**Scenario 4: Retro on a completed sprint**
- **Given** sprint S004 in `completed` status with `--detailed` summary data and 2 carried-over entities with rejection notes
- **When** user runs `/retro-sprint S004`
- **Then** the skill writes `docs/sprints/S004-retro.md` containing the five required sections, populated from shark JSON
- **And** the Recommendations section contains 3–5 data-driven items, not generic advice

**Scenario 5: Sprint-aware `/run` notice**
- **Given** task E07-F01-001 is assigned to active sprint S005
- **When** user runs `/run E07-F01-001` directly
- **Then** the existing `/run` skill prints the one-line advisory pointing at `/run-sprint S005`
- **And** continues per-entity execution unchanged

### Out of Scope

1. **Multi-sprint orchestration.** Driving more than one sprint in a single Claude session is out of scope; the constraint "only one sprint can be `active`" (epic assumption 2) makes this moot for `/run-sprint*`, and `/plan-sprint S### S###` is explicitly not supported.
2. **Auto-creation of next sprint.** Covered by REQ-F-016 (Could-Have at the epic level). If F02 ships auto-creation, these skills consume it; they do not implement it.
3. **Sprint dashboards / status integration with `shark status`.** Covered by REQ-F-017 at the epic level. These skills do not add UI to `shark status`.
4. **Custom retro templates.** `/retro-sprint` produces one canonical layout. User-customizable retro templates are out of scope.
5. **Modifying `/run-agent-team` execution semantics.** REQ-F-024 only adds a one-line discoverability cross-reference. The execution model (preconditions, dep-graph vs. order modes, single-team-per-session) is unchanged.

---

## Architecture

### Component Inventory

All paths are absolute under `~/.claude/`. Nothing in this feature lives in the shark repo.

| Path | Action | Owner Task |
|---|---|---|
| `~/.claude/commands/plan-sprint.md` | CREATE | T-E19-F06-001 |
| `~/.claude/skills/sprint-planning/SKILL.md` | CREATE | T-E19-F06-001 |
| `~/.claude/skills/sprint-planning/workflows/plan-sprint.md` | CREATE | T-E19-F06-001 |
| `~/.claude/commands/run-sprint.md` | CREATE | T-E19-F06-002 |
| `~/.claude/skills/sprint-execution/SKILL.md` | CREATE | T-E19-F06-002 (created here, extended by 003) |
| `~/.claude/skills/sprint-execution/workflows/run-sprint.md` | CREATE | T-E19-F06-002 |
| `~/.claude/commands/run-sprint-team.md` | CREATE | T-E19-F06-003 |
| `~/.claude/skills/sprint-execution/workflows/run-sprint-team.md` | CREATE | T-E19-F06-003 |
| `~/.claude/commands/retro-sprint.md` | CREATE | T-E19-F06-004 |
| `~/.claude/skills/sprint-analytics/SKILL.md` | CREATE | T-E19-F06-004 |
| `~/.claude/skills/sprint-analytics/workflows/retro-sprint.md` | CREATE | T-E19-F06-004 |
| `~/.claude/skills/orchestration/workflows/run.md` | EDIT (Step 1 amendment) | T-E19-F06-005 |
| `~/.claude/commands/run-agent-team.md` | EDIT (See-also note) | T-E19-F06-005 |
| `~/.claude/skills/orchestration/workflows/run-agent-team.md` | EDIT (See-also note) | T-E19-F06-005 |
| `~/.claude/agents/product-manager.md` | EDIT (sprint subsection) | T-E19-F06-005 |
| `~/.claude/PIPELINE.md` | EDIT (one table row) | T-E19-F06-005 |

### Skill Layout Conventions (Patterns Followed)

The three new skills follow the **existing `orchestration` skill pattern** (`~/.claude/skills/orchestration/`):

```
sprint-planning/
  SKILL.md                       # frontmatter + 1-paragraph "What this is" + pointer to workflows/
  workflows/
    plan-sprint.md               # full workflow body
sprint-execution/
  SKILL.md
  workflows/
    run-sprint.md
    run-sprint-team.md
sprint-analytics/
  SKILL.md
  workflows/
    retro-sprint.md
```

This mirrors `orchestration/SKILL.md` + `orchestration/workflows/run.md` + `orchestration/workflows/run-agent-team.md`. Slash commands under `~/.claude/commands/` are the user-facing thin wrapper and contain the standard `---\ndescription: …\n---` frontmatter, a Usage block, an Arguments block, a "What This Does" section that points at the workflow file, and a CRITICAL INSTRUCTIONS block.

### Skill-by-Skill Design

#### `/plan-sprint` (T-E19-F06-001)

**Slash-command wrapper** (`~/.claude/commands/plan-sprint.md`): same shape as `~/.claude/commands/run.md` — frontmatter, Usage with the `--mode=auto|interactive` and `--no-confirm-reads` examples, refusal pattern for non-`S###` keys, "What This Does" pointing at `sprint-planning/workflows/plan-sprint.md`, and a CRITICAL INSTRUCTIONS list ending with "DO NOT use EnterPlanMode."

**Workflow body** (`sprint-planning/workflows/plan-sprint.md`):

```
Step 0: Argument parse
  - Reject non-S### keys
  - Parse --mode (default: interactive), --max-add (default: unlimited)

Step 1: Read sprint state
  - shark get {S###} --json
  - If status not in {planning, active}, ask user; exit on decline
  - Capture sprint key, status, capacity from output

Step 2: Read plan view
  - shark sprint plan {S###} --json
  - Parse: backlog list, capacity per agent_type, readiness score + factors

Step 3: Branch on --mode
  - interactive:
      Display readiness breakdown
      For each backlog group (by feature, then standalones):
        Show entities; ask user which to add
        For confirmed entities: shark sprint add {S###} {KEY} --json
  - auto:
      Greedy fill: sort backlog by (priority desc, size asc) within each agent type
      Stop adding to an agent type when allocated >= capacity
      Show proposed plan; ask for ONE confirmation
      On confirm: shark sprint add for each (in batches, not bulk — bulk-add
      semantics differ per F03 and we want explicit per-entity adds for
      auditability)

Step 4: Re-read readiness
  - shark sprint readiness {S###} --json
  - Report score delta vs. Step 2

Step 5: Exit
  - DO NOT call shark sprint start
  - Suggest: "Sprint S### ready. To execute: /run-sprint S###"
```

**Key technical decisions**:
- **Why interactive default?** Sprint planning is a human-judgment activity. Auto-mode exists for orchestrator use (e.g., `/loop` or scheduled planning) but defaults to confirm-everything because mistakes are expensive (REQ-NF-008).
- **Why per-entity adds in auto-mode rather than `--bulk`?** Bulk adds collapse audit detail. Sequential adds make rollback feasible (the user can interrupt mid-loop) and keep the shark `sprint_assignments` history granular.

#### `/run-sprint` (T-E19-F06-002)

**Slash-command wrapper** (`~/.claude/commands/run-sprint.md`): same shape as `/run`. CRITICAL INSTRUCTIONS includes "DO NOT skip the user-confirm-before-close gate."

**Workflow body** (`sprint-execution/workflows/run-sprint.md`):

```
Step 0: Argument parse
  - Reject non-S### keys
  - Parse --agent, --max-iterations (default 50), --carryover (optional)

Step 1: Sprint readiness
  - shark get {S###} --json --field=status
  - If planning: ask user to start; on confirm, shark sprint start {S###}
  - If completed/archived/cancelled: exit with notice

Step 2: Pull-loop
  for i in 1..max_iterations:
    NEXT = shark sprint next {--agent=X if set} --json
    if NEXT empty:
      break "no more entities"
    /run {NEXT.key}    # Delegate to existing orchestration skill
    # /run drives the entity to terminal; on return we loop

Step 3: Post-loop reporting
  - shark sprint burndown {S###}    # text chart
  - shark sprint summary {S###}     # basic, not --detailed

Step 4: Close prompt
  - Ask user: "Close sprint S### now? (carryover={carryover or default})"
  - On confirm: shark sprint close {S###} --carryover={value}
  - On decline: exit, report sprint left in active state
```

**Key technical decisions**:
- **Why delegate to `/run` rather than embed dispatch?** `/run` is the canonical per-entity dispatcher and reads `orchestrator_action.instruction`. Re-implementing dispatch here would duplicate logic and drift from the workflow source of truth (which is shark, not Claude). This skill is a *sprint-shaped harness around `/run`*.
- **Why `--max-iterations`?** `shark sprint next` returns an entity even if the previous one bounced back to a non-terminal state (e.g., agent failed and shark left it `in_*`). Without a cap we could loop on the same entity. Cap defaults to 50 — large enough for any realistic sprint, small enough to prevent cost-runaway.
- **Why no auto-close?** Closing a sprint with carryover is a planning decision (which sprint receives the carry-over, or whether items go back to backlog). Per REQ-NF-008, this requires user confirmation.

#### `/run-sprint-team` (T-E19-F06-003)

**Slash-command wrapper** (`~/.claude/commands/run-sprint-team.md`): same shape as `/run-agent-team`. CRITICAL INSTRUCTIONS includes "DO NOT spawn more than one agent team at a time" (a hard limit of the agent-teams primitive).

**Workflow body** (`sprint-execution/workflows/run-sprint-team.md`):

```
Step 0: Argument parse
  - Reject non-S### keys
  - Parse --size (passed through), --features=... (filter)

Step 1: Inherit /run-agent-team preconditions
  - Verify env var, claude version, branch, worktree, no existing team
  - Reuse the precondition list from
    ~/.claude/skills/orchestration/workflows/run-agent-team.md verbatim
    (cite by reference; do not duplicate)

Step 2: Sprint readiness
  - Same as /run-sprint Step 1

Step 3: Group entities by feature
  - shark sprint backlog {S###} --json
  - For each entity, extract feature_key:
      task E##-F##-### → E##-F##
      bug B###, change-card CC-###, tech-debt TD-### with no parent → "standalone"
  - Build groups: {E##-F##: [entities], "standalone": [entities]}
  - If --features filter provided, retain only listed features (and always
    drop standalones unless --features=standalone is included)

Step 4: Sequential per-feature dispatch
  for feature_key in feature_groups (sorted by epic then F##):
    /run-agent-team {feature_key} [--size=N]
    # /run-agent-team handles its own preconditions, plan confirmation,
    # team spawn, and cleanup. We wait for it to return.
    shark sprint burndown {S###}    # progress check between groups

Step 5: Standalone dispatch (if any)
  for entity_key in standalone_group:
    /run {entity_key}              # plain orchestrator dispatch

Step 6: Post-loop (same as /run-sprint Steps 3–4)
```

**Key technical decisions**:
- **Why serial features?** The agent-teams docs say only one team can exist per Claude session. We honor that by serializing feature groups; parallelism happens *within* a feature via the team primitive itself.
- **Why fall back to `/run` for standalones?** `/run-agent-team` rejects non-feature keys (`"only operates on features"`). Bugs, change-cards, and tech-debt items not nested under a feature have no shared file-locking surface; sequential `/run` is the right shape.
- **Why filter via `--features` rather than re-using `--size`?** Two different concerns: `--features` scopes execution; `--size` tunes per-team teammate count. The two are orthogonal and pass through to different downstream commands.

#### `/retro-sprint` (T-E19-F06-004)

**Slash-command wrapper** (`~/.claude/commands/retro-sprint.md`): standard shape, no team-mode preconditions needed (it's read-only).

**Workflow body** (`sprint-analytics/workflows/retro-sprint.md`):

```
Step 0: Argument parse
  - Reject non-S### keys
  - Parse --no-write

Step 1: Verify sprint is closed
  - shark get {S###} --json --field=status
  - If not in {completed, archived}: refuse with message

Step 2: Pull data
  - SUMMARY = shark sprint summary {S###} --detailed --json
  - VELOCITY = shark sprint velocity --json
  - For each entity in SUMMARY.carryover and SUMMARY.rejected (if present):
      NOTES[entity_key] = shark notes {entity_key}

Step 3: Synthesize Recommendations (3-5 items)
  - Pattern-match against the data:
    * "trailing-avg variance > 25%" → "consider re-evaluating capacity"
    * "any size>=8 entity completed" → "split XL+ entities before next sprint"
    * "carryover_count > 30% of planned" → "scope was too aggressive"
    * "phase X cycle time > 2x phase Y" → "investigate {phase X} bottleneck"
    * "agent type allocated > capacity by 20%" → "rebalance {agent type} load"
  - Pick top 3-5 by signal strength; never emit generic placeholders

Step 4: Render markdown
  - Sections (in order):
    # Sprint S### Retrospective
    ## Outcome
    ## Velocity Context
    ## Carryover Analysis
    ## Cycle-Time Highlights
    ## Recommendations
  - Use SUMMARY data to populate Outcome and Cycle-Time
  - Use VELOCITY data to populate Velocity Context
  - Use NOTES to populate Carryover Analysis (cite each note's type/source)

Step 5: Write or print
  - If --no-write: print to stdout
  - Else: docs/sprints/{S###}-retro.md
    If exists: prompt before overwriting
    Create docs/sprints/ dir if missing
```

**Key technical decisions**:
- **Why a fixed five-section layout?** Retros are most useful when consistent across sprints — comparing two retros is easier when their structure matches. Customizable templates are explicitly out of scope.
- **Why generate Recommendations from data patterns rather than asking an LLM to think freely?** Determinism. The same shark data should produce the same recommendations across sessions and across orchestrator runs. Free-form LLM advice is acceptable in the *body* of a recommendation (explaining why) but the *trigger* must be a measurable threshold.
- **Why `docs/sprints/` rather than `docs/plan/`?** Sprints are time-boxed, not plan-hierarchy entities. Mixing sprint retros into `docs/plan/E##/` would obscure the per-epic plan structure. `docs/sprints/{S###}-retro.md` keeps sprint artifacts grouped chronologically.

#### Cross-Cutting Edits (T-E19-F06-005)

**`run.md` Step 1 amendment** (REQ-F-023): The amendment adds, immediately after the existing `shark get {ID} --json` call, a check on the parsed `sprint` field (or whichever F03 names — TBD until F03 spec finalizes; if the field is absent, the amendment falls back to `shark sprint backlog --json` filter on `entity_key`). On a hit, prints the one-line advisory and continues. The amendment is additive and does not change any other dispatch logic.

> **Note**: T-E19-F06-005 must verify the exact field name produced by F03's `shark get` JSON output before editing `run.md`. If the field is named `sprint_assignment` or nested under `assignments`, the workflow text must use that. This is the only TBD in the spec — and it's a downstream-coordination point, not an open architectural question.

**`run-agent-team.md` and `commands/run-agent-team.md` cross-references** (REQ-F-024): A single sentence each, near the Usage section. Example: `> See also: \`/run-sprint-team\` for sprint-scoped multi-feature execution.`

**`product-manager.md` subsection** (REQ-F-026): A new subsection under "PRIMARY: Feature Execution Workflow" titled "When a sprint is active":

```
### When a sprint is active

If the feature you're executing has tasks assigned to an `active` sprint
(check via `shark get {FEATURE} --json --field=sprint` or the sprint's
backlog), prefer:

- `/run-sprint S###` — solo, sequential execution of the whole sprint
- `/run-sprint-team S###` — team execution, one feature at a time
- `/plan-sprint S###` — if the sprint is still in `planning`
- `/retro-sprint S###` — after close

`/run E##-F##` still works for per-feature execution but bypasses the
sprint dispatch loop. Use sprint commands when the user is thinking
in terms of "this iteration."
```

**`PIPELINE.md` table row** (REQ-F-025): Add to the "What 'Follow SDLC Flow' Covers" table:

| Sub-phase | Skill | Output |
|---|---|---|
| Sprint orchestration | `sprint-planning`, `sprint-execution`, `sprint-analytics` (`/plan-sprint`, `/run-sprint`, `/run-sprint-team`, `/retro-sprint`) | Sprint state in shark; `docs/sprints/{S###}-retro.md` |

### Integration with Existing Code

This feature integrates with three existing surfaces:

1. **The `orchestration` skill** (`~/.claude/skills/orchestration/`) — `/run-sprint` and `/run-sprint-team` delegate per-entity dispatch to `/run` rather than re-implementing it. The advisory amendment in `run.md` is the only edit; no semantic change.
2. **The agent-teams primitive** — `/run-sprint-team` invokes `/run-agent-team` once per feature. The primitive's preconditions, lifecycle, and cleanup are unchanged; we just add a sprint-shaped harness around it.
3. **The shark CLI** — all four skills are pure consumers of `shark sprint …` commands and existing `shark get/list/notes` commands. Per REQ-NF-009, every shark call uses `--json`. Per REQ-NF-010, we make zero changes to the shark codebase.

No changes are required to:
- The shark Go codebase (`internal/`, `cmd/`)
- The shark database schema or migrations
- Existing slash commands other than the documented edits in T-E19-F06-005
- Existing agents other than `product-manager.md`
- The PDLC pipeline diagram (only the SDLC table is edited)

### Validation Strategy

This feature is documentation/skill-prompt code, not Go code. Validation is per-skill manual verification:

1. Each new slash command resolves and prints its help/usage
2. Each skill correctly refuses non-`S###` keys
3. Each skill consumes `shark sprint --json` output without breakage on a fixture sprint
4. The advisory amendment in `run.md` fires correctly when an entity is in an active sprint
5. The retro generator produces the five required sections with non-empty Recommendations on a fixture closed sprint

Detailed test plan deferred to the test-planning phase (next workflow status).

---

## Risks & Open Questions

**Risk 1: F03 field name for sprint assignment is undefined.** REQ-F-023 depends on the field that exposes the active sprint on `shark get`. If F03 ships a different shape than expected, T-E19-F06-005 must adapt the amendment. Mitigation: T-E19-F06-005 is sequenced *after* T-E19-F06-001 through 004 and must check the F03 output before editing `run.md`.

**Risk 2: Agent-teams primitive limits.** `/run-sprint-team` assumes one team per session. If the primitive evolves to allow multiple, the serial-features design becomes suboptimal (but not incorrect). Mitigation: revisit when/if the primitive changes.

No truly open questions — all design decisions have rationales above.
