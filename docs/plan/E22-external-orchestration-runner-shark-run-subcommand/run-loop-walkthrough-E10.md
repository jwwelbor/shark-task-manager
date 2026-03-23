# `shark run E10` — Run Loop Walkthrough

**Purpose**: Visualize the exact flow of `shark run` driving an epic from `draft` to `completed`, using E10 ("Advanced Task Intelligence & Context Management") as the example.

**Workflow source**: `shark-templates/.sharkworkflow-short.json` → `epic_workflow`

---

## Design Decisions Applied

1. **`check_or_resume` → `spawn_agent`**: All `in_*` statuses now use `spawn_agent` with resume instructions. `check_or_resume` is reserved as a true breakpoint/pause (not currently used in any workflow).

2. **`cascade` = internal loop**: The controller handles `cascade` internally by iterating child entities and calling `Run()` recursively. No agent is dispatched for cascade.

3. **`ready_for_X` → `in_X` = one agent launch**: When the controller sees `spawn_agent` at a `ready_for_*` status, it advances to `in_*` first (marks work started), then dispatches the agent. If the agent fails or is interrupted, `in_*` has its own `spawn_agent` with a resume instruction — still only one dispatch per attempt.

---

## Controller Pseudocode

```
Run(key):
  info = GetNextStatus(key)
  if info.IsTerminal → return "already_terminal"
  currentStatus = info.CurrentStatus

  loop:
    vars = GeneratePlaceholders(key)
    action = GetStatusActionPopulated(currentStatus, vars)    ← config read, no DB

    switch action.Action:

      case "advance_status":
        result = TransitionStatus(key, firstTransition)       ← same as shark status advance
        currentStatus = result.ToStatus
        if terminal → return "completed"
        continue loop

      case "spawn_agent":
        // Step 1: Advance ready_for → in_ (marks work started)
        result = TransitionStatus(key, firstTransition)       ← same as shark status advance
        // Step 2: Dispatch agent with instruction from the ready_for status
        dispatchResult = dispatcher.Dispatch(action.instruction)
        if dispatchResult.ExitCode != 0 → return "failed"    ← status stays at in_, resumable
        // Step 3: Advance in_ → next ready_for (marks work done)
        result = TransitionStatus(key, firstTransition)       ← same as shark status advance
        currentStatus = result.ToStatus
        if terminal → return "completed"
        continue loop

      case "cascade":
        children = listChildren(key)                          ← shark list <key> --json
        for each child (in execution order):
          if child is terminal → skip
          Run(child.key)                                      ← recursive
        // All children done → advance parent
        result = TransitionStatus(key, firstTransition)
        currentStatus = result.ToStatus
        if terminal → return "completed"
        continue loop

      case "pause":
        return "paused"

      case "archive":
        return "completed"
```

### Resume Case (interrupted at `in_X`)

```
Run(key):
  info = GetNextStatus(key)
  currentStatus = "in_refinement"                             ← picked up where we left off

  loop:
    action = GetStatusActionPopulated("in_refinement", vars)  ← spawn_agent (resume instruction)

    case "spawn_agent":
      // Step 1: Advance in_refinement → ready_for_research?
      //         NO — we DON'T want to skip the work.
      //         The in_ template has resume logic. We dispatch from here.
      //
      // KEY: When currentStatus is already in_*, the controller
      //      dispatches the agent WITHOUT advancing first.
      //      The advance happens AFTER agent success.
      dispatchResult = dispatcher.Dispatch(action.instruction)  ← resume instruction
      if dispatchResult.ExitCode != 0 → return "failed"
      result = TransitionStatus(key, firstTransition)           ← in_ → next ready_for
      currentStatus = result.ToStatus
      continue loop
```

**How the controller distinguishes**: When `spawn_agent` fires, check if the first available transition target starts with `in_` (e.g., `ready_for_X` → `in_X`). If yes, advance first then dispatch. If no (we're already at `in_X` → `ready_for_Y`), dispatch first then advance.

---

## Happy Path: Full Epic Lifecycle

| # | Status | Action | Agent | Model | Controller Behavior |
|---|--------|--------|-------|-------|---------------------|
| 1 | `draft` | `advance_status` | — | — | Advance → `ready_for_refinement`. No agent. |
| 2 | `ready_for_refinement` | `spawn_agent` | business-analyst | opus | Advance → `in_refinement`. Dispatch BA: "Write epic PRD for E10". |
| 3 | *(agent works)* | | | | BA creates PRD docs. Exits 0. Controller advances `in_refinement` → `ready_for_research`. |
| 4 | `ready_for_research` | `spawn_agent` | researcher | sonnet | Advance → `in_research`. Dispatch researcher: "Brownfield analysis for E10". |
| 5 | *(agent works)* | | | | Researcher produces research report. Exits 0. Controller advances `in_research` → `ready_for_design`. |
| 6 | `ready_for_design` | `spawn_agent` | architect | opus | Advance → `in_design`. Dispatch architect: "Architecture + UAT plan for E10". |
| 7 | *(agent works)* | | | | Architect produces design docs. Exits 0. Controller advances `in_design` → `ready_for_decomposition`. |
| 8 | `ready_for_decomposition` | `spawn_agent` | product-manager | sonnet | Advance → `in_decomposition`. Dispatch PM: "Decompose E10 into features". |
| 9 | *(agent works)* | | | | PM creates features. Exits 0. Controller advances `in_decomposition` → `active`. |
| 10 | `active` | `cascade` | — (internal) | — | Controller lists features. For each non-terminal feature: `Run(feature-key)` recursively. When all done → advance `active` → `completed`. |
| 11 | `completed` | `archive` | — | — | Terminal. Return `outcome=completed`. |

**Total agent dispatches**: 4 (one per phase: refinement, research, design, decomposition)
**Total status transitions**: 10 (draft→rfr→ir→rfres→ires→rfd→id→rfdecomp→idecomp→active→completed)

---

## Failure & Resume Scenario

```
Iteration 4: ready_for_research → advance to in_research → dispatch researcher
             Researcher crashes (exit code 1)
             → Controller returns outcome="failed"
             → Status stays at in_research (work was started but not finished)

User re-runs: shark run E10

Iteration 1: GetNextStatus("E10") → status=in_research
             action = spawn_agent (resume instruction from in_research.tmpl)
             → Controller dispatches researcher with resume instruction
             → Researcher checks for existing report, continues work
             → Exits 0 → advance in_research → ready_for_design
             → Loop continues normally from ready_for_design
```

---

## Cascade Detail: `active` Status

When the controller reaches `active` (cascade action), it handles this internally:

```
handleCascade(epicKey):
  features = shark feature list --epic=E10 --json       ← sorted by execution order

  for each feature:
    if feature.status in [completed, cancelled]:
      skip
    else:
      Run(feature.key)                                   ← recursive call
      // This drives the feature through ITS full workflow:
      // draft → ready_for_assessment → ... → active → completed
      // Which in turn cascades into tasks when feature hits "active"

  // All features terminal → advance epic
  TransitionStatus("E10", "completed")
```

**E10 right now** (status=active, 6 features):

| Feature | Status | What cascade does |
|---------|--------|-------------------|
| E10-F01 | completed | Skip |
| E10-F02 | completed | Skip |
| E10-F03 | active (90.9%) | `Run("E10-F03")` → cascade into its tasks |
| E10-F04 | completed | Skip |
| E10-F05 | completed | Skip |
| E10-F06 | draft | `Run("E10-F06")` → drive through full feature workflow |

After F03 and F06 complete → advance E10 to `completed`.

---

## Status Flow Diagram

```
                                  ONE AGENT DISPATCH PER PHASE
                               ┌──────────────────────────────────┐
                               │  advance first    then dispatch  │
                               │  ready_for → in_  agent works    │
                               │  (status change)  (one launch)   │
                               └──────────────────────────────────┘

draft ─[advance_status]─→ ready_for_refinement ─[advance to in_]─→ in_refinement ─[BA works]─→
                                                                                               │
                          ready_for_research ←─────────────────────────────────────────────────┘
                                │
                          [advance to in_]─→ in_research ─[researcher works]─→
                                                                              │
                          ready_for_design ←──────────────────────────────────┘
                                │
                          [advance to in_]─→ in_design ─[architect works]─→
                                                                           │
                          ready_for_decomposition ←────────────────────────┘
                                │
                          [advance to in_]─→ in_decomposition ─[PM works]─→ active
                                                                                │
                                                                          [cascade: run features]
                                                                                │
                                                                           completed
```

**Exception paths** (can happen at any `ready_for_*` or `in_*`):
```
any status → blocked → [pause] → (user unblocks) → resume at ready_for_*
any status → on_hold → [pause] → (user resumes)  → resume at ready_for_*
```

---

## Action Type Reference

| Action | Meaning | Controller Behavior |
|--------|---------|---------------------|
| `advance_status` | Auto-advance, no agent needed | `TransitionStatus()` → loop |
| `spawn_agent` | Dispatch an agent | If at `ready_for_*`: advance first, then dispatch. If at `in_*`: dispatch (resume), then advance on success. |
| `cascade` | Drive child entities | Internal loop: list children, `Run()` each recursively, advance parent when all done. |
| `pause` | Stop, human intervention needed | Return `outcome=paused`. |
| `archive` | Terminal, entity is done | Return `outcome=completed`. |
| `check_or_resume` | True breakpoint (reserved) | Return `outcome=paused`. Not currently used in any workflow. |
