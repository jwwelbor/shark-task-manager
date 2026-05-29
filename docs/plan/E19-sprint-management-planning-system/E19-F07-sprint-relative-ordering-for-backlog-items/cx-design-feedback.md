# CX Design Feedback: Sprint-Relative Ordering for Backlog Items (E19-F07)

**Audience**: Implementation team (architect + spec writer)
**Date**: 2026-05-15
**Feature status**: in_specification

---

## 1. Current State Analysis

### What exists today

`shark sprint backlog <sprint-key>` groups sprint items by workflow status category (e.g., "draft", "completed") with a flat table per group. Columns are: KEY, TYPE, STATUS, TITLE. No ordering column is shown, no position numbers appear. Within each group, items are displayed in an unspecified database order.

`shark sprint next` selects the single next eligible item using: ExecutionOrder (project-level) → Priority → AssignedAt (FIFO). This produces counterintuitive results when some items have ExecutionOrder set and others do not — the outlier with any ExecutionOrder value always wins over every unordered item, regardless of intent.

`shark sprint plan <sprint-key>` shows unassigned backlog candidates for scoping decisions, not the ordered pull queue of already-assigned items.

### The core UX problem

There is no surface in the current CLI that represents "the sprint pull queue as the PM/orchestrator intended it." The ordering that `sprint next` applies is derived from project-level metadata that was not set with sprint execution in mind. This creates a disconnect between "what I planned for this sprint" and "what the agent gets told to do next."

The two audiences experience this differently:

- **AI agent**: `sprint next` returns the wrong item. No JSON field distinguishes "sprint order" from "project order," so the agent cannot detect or correct the mismatch.
- **Human PM**: `sprint backlog` does not show a pull sequence, so there is no way to visually verify the execution order before starting the sprint.

---

## 2. AI Agent UX Recommendations

### 2.1 What `sprint next` must expose in JSON

The agent's decision to work on the returned item depends entirely on trusting that `sprint next` gave the right answer. To support this trust and to allow the agent to reason about ordering, the JSON response from `sprint next` must include:

```json
{
  "key": "E19-F07-001",
  "title": "...",
  "status": "todo",
  "sprint_order": 2,
  "sprint_key": "S005",
  "execution_order": null,
  "priority": 3,
  "agent_type": "backend",
  "selection_reason": "sprint_order"
}
```

Key additions:
- `sprint_order`: The explicit sprint position (integer, nullable). If null, the item was ordered by fallback signals.
- `selection_reason`: A short enum string indicating which sort factor determined this item's selection. Values: `"sprint_order"`, `"execution_order"`, `"priority"`, `"assigned_at"`. This lets the agent understand whether the selection was intentional or a fallback.
- `sprint_key`: The sprint this item belongs to (already useful for agent context-building).

The agent workflow becomes deterministic: pull `sprint next --json`, check `selection_reason`, begin work. If `selection_reason` is `"assigned_at"` the agent can optionally surface a warning that the sprint has no explicit ordering.

### 2.2 How the agent sets ordering

The agent should be able to set sprint order as part of its planning loop, not just as a human-interactive operation. The recommended surface is:

**During sprint add** (most natural point):
```
shark sprint add S005 E19-F07-001 --at=1
shark sprint add S005 E19-F07-002 --at=2
```

The `--at=<position>` flag places the item at a specific position and shifts existing items accordingly. This matches the feature spec. The agent can iterate through its prioritized candidate list and assign positions explicitly.

**Post-add reorder** (correction or replanning):
```
shark sprint reorder S005 E19-F07-003 --top
shark sprint reorder S005 E19-F07-001 3
```

Both forms should be scriptable (no interactive prompts). All output should support `--json`.

**What the agent does NOT need**: A drag-and-drop mental model, pairwise swaps, or any operation that requires reading back the full ordered list before each write. The `--top`, `--bottom`, and `<position>` forms cover all orchestrator use cases without requiring stateful reasoning.

### 2.3 JSON output for `sprint backlog` — ordered view

When the agent calls `shark sprint backlog <sprint-key> --view=ordered --json`, the response should include `sprint_order` on every item and sort by it ascending (nulls last):

```json
{
  "sprint_key": "S005",
  "sprint_name": "...",
  "view": "ordered",
  "items": [
    {
      "position": 1,
      "sprint_order": 1,
      "key": "E19-F07-001",
      "title": "...",
      "status": "todo",
      "agent_type": "backend",
      "priority": 3,
      "size": 2
    },
    {
      "position": 2,
      "sprint_order": 2,
      "key": "B042",
      "title": "...",
      "status": "todo",
      "agent_type": "frontend",
      "priority": 5,
      "size": 1
    }
  ]
}
```

The `position` field is the sequential rank (1-based, always dense), while `sprint_order` is the stored value. For the agent these are the same in a well-ordered sprint; the distinction matters only when there are gaps (sparse ordering after deletions).

### 2.4 Dependency interaction

The agent must not be blocked by a dependency-unaware ordering. If `sprint next` returns item at position 3 but its dependency (position 1) is still in progress, the agent needs enough information to skip ahead. Two approaches, in order of preference:

**Recommended**: `sprint next` already filters to initial-workflow-status items only. This means items in progress do not appear as candidates. The agent just calls `sprint next` again when stuck and gets the next eligible item. No dependency metadata needed in the response.

**If parallel agent execution is added later**: Include a `blocked_by` array in the `sprint next` JSON response so the agent can skip intelligently. This is out of scope for E19-F07 but the JSON contract should leave room for it (`blocked_by: []` as a placeholder).

### 2.5 Carryover behavior the agent must understand

When the agent calls `shark sprint close S005 --carryover=next --json`, the response should confirm that `sprint_order` was preserved and renumbered:

```json
{
  "sprint_closed": "S005",
  "carryover_sprint": "S006",
  "carryover_count": 3,
  "order_preserved": true
}
```

This allows the agent to skip re-planning the next sprint's order for carryover items. If `order_preserved: false` (carryover=backlog path), the agent knows it must re-assign positions when building the next sprint.

---

## 3. Human Developer UX Recommendations

### 3.1 Sprint backlog display — ordered view

The current backlog display groups by status, which is useful for monitoring. The new ordered view serves a different purpose: it shows the pull queue. These are two distinct mental models and should not be merged.

Recommended: make the ordered view the default for active sprints (status=active), keep the grouped view as `--view=grouped` (or just omit `--view`). Rationale: once a sprint is active, the PM's primary question is "what order will items be worked" not "how many items are in each status." The grouped view is more useful during planning and monitoring.

**Proposed ordered view format**:
```
Pull Queue: S005 (Sprint 24)  —  3/8 complete (37%)

  #   KEY             TYPE    STATUS          AGENT      TITLE
  --  --------------- ------  --------------  ---------  ------------------------
   1  E19-F07-001     task    in_progress     backend    Implement sprint_order col...
   2  B042            bug     todo            frontend   Fix login redirect loop
   3  E19-F07-002     task    todo            backend    Add reorder command
   4  E27-F14-003     task    todo            frontend   Fix Plan tab badges
   5  CC-012          change  todo            human      Update API rate limits
   ~  E19-F07-003     task    todo            backend    Write migration          (unordered)
   ~  B044            bug     todo            qa         Auth token expiry test   (unordered)
```

Design decisions in this format:
- Position numbers (#) are the primary visual signal. Humans scan numbers.
- Items without `sprint_order` show `~` to clearly distinguish them from intentionally ordered items. This makes it obvious when the sprint is only partially ordered.
- Completed items are omitted from the ordered view by default (they are done; ordering no longer matters). Add `--include-completed` flag to show them if needed.
- In-progress items show their current status in the STATUS column — the human can see that position 1 is already being worked.

### 3.2 Sprint backlog display — grouped view (existing behavior, retained)

The grouped view stays as it is for monitoring purposes. It should remain accessible via `shark sprint backlog S005 --view=grouped` and continue to be the default for planning and completed sprints.

### 3.3 `sprint plan` integration

The planning view (`shark sprint plan S005`) shows unassigned backlog candidates. This is where the human makes assignment decisions. After this feature, two ordering-related actions become natural during planning:

1. **Assign with position**: Show an `--at=<N>` hint in the output footer after each assignment:
   ```
   Added E19-F07-001 to S005 (position 5 — end of queue)
   Tip: use --at=<N> on the next add to insert at a specific position
   ```

2. **Reorder reminder**: After `sprint start`, if the sprint has unordered items (sprint_order is null for any), surface a one-line notice:
   ```
   Warning: 3 items have no sprint order. Use `shark sprint reorder` to set pull priority.
   ```
   This is a soft nudge, not a blocker. The sprint can start with partially ordered items.

### 3.4 Mental model for human ordering

The human mental model should be "I am building a numbered list of what to work on, in order." This is equivalent to a physical sticky note ordering on a sprint board.

Key conventions that support this mental model:

- **Position 1 = first to pull**. Lower number = higher sprint priority. This matches natural human counting and the physical sprint board metaphor.
- **Gaps are acceptable, briefly**. When an item is removed from the sprint, don't automatically renumber. The PM may be doing a series of removes and will reorder afterward. Renumber only on explicit `reorder` commands. This prevents surprising position shifts during bulk operations.
- **Unordered items are second class**. Items without a position are displayed after all ordered items and shown with `~`. This motivates PMs to assign positions when they care about ordering.

### 3.5 `sprint reorder` command design

The `reorder` command is the primary human interaction point. It should feel like moving a card on a board.

Natural phrasings a human would try:
- "Move this to the top" → `--top`
- "Move this to the bottom" → `--bottom`
- "Move this to position 3" → `<position>` positional arg
- "Swap this with that" → Not supported (out of scope, and the mental model of numbered positions makes this less needed)

Proposed syntax:
```
shark sprint reorder <sprint> <entity> <position>
shark sprint reorder <sprint> <entity> --top
shark sprint reorder <sprint> <entity> --bottom
```

Output after reorder:
```
Moved B042 to position 1 in S005.

  #   KEY             TITLE
  1   B042            Fix login redirect loop
  2   E19-F07-001     Implement sprint_order column
  3   E19-F07-002     Add reorder command
  ~   E19-F07-003     Write migration (unordered)
```

Showing the abbreviated pull queue after a reorder gives immediate feedback that the move was correct. The human should not need to run a separate `backlog --view=ordered` to verify.

---

## 4. Command Design Proposals

### 4.1 `shark sprint add` with `--at`

```
shark sprint add <sprint-key> <entity-key> [--at=<position>]
```

- `--at` is optional. Default: append to end (max existing sprint_order + 1).
- `--at=1` places at the top, shifting all others down.
- Position is 1-based.
- Error if position > current item count + 1.

Output:
```
Added E19-F07-001 to S005 at position 3.
Sprint now has 8 items (5 ordered, 3 unordered).
```

JSON output:
```json
{
  "sprint_key": "S005",
  "entity_key": "E19-F07-001",
  "sprint_order": 3,
  "total_items": 8,
  "ordered_items": 5
}
```

### 4.2 `shark sprint reorder`

```
shark sprint reorder <sprint-key> <entity-key> <position>
shark sprint reorder <sprint-key> <entity-key> --top
shark sprint reorder <sprint-key> <entity-key> --bottom
```

- `<position>`, `--top`, and `--bottom` are mutually exclusive.
- `--top` is equivalent to `--at=1`.
- `--bottom` assigns `sprint_order = max(existing) + 1` and clears the current position.
- `--json` outputs the updated abbreviated pull queue.

Error cases:
- Entity not in sprint: "E19-F07-999 is not assigned to S005"
- Position out of range: "Position 99 is out of range (sprint has 8 items)"

### 4.3 `shark sprint next` changes

No change to command signature. Changes to output only.

Human output (current): Shows the entity key and title.
Human output (new): Adds position and selection reason.
```
Next: E19-F07-001 — Implement sprint_order column
  Sprint position: 2 of 8  |  Selected by: sprint_order
  Sprint: S005 (Sprint 24)  |  Agent: backend  |  Priority: 3  |  Size: 2
```

JSON output: Add `sprint_order`, `selection_reason` fields (see section 2.1).

### 4.4 `shark sprint backlog` changes

```
shark sprint backlog <sprint-key> [--view=ordered|grouped] [--type=...] [--blocked] [--json]
```

- `--view=ordered` shows the numbered pull queue (section 3.1 format).
- `--view=grouped` shows the existing status-grouped table (current behavior).
- Default for active sprints: `ordered`. Default for planning/completed sprints: `grouped`.

The `--blocked` flag works in both views. In ordered view, blocked items are marked with a `!` prefix:
```
  !2  E27-F14-003     task    blocked         backend    Fix Plan tab badges (blocked 2d)
```

---

## 5. Key Design Decisions

### 5.1 Dense vs. sparse renumbering on reorder

**Decision**: Dense renumbering on every explicit reorder command. Gaps may exist temporarily between adds and removes during bulk operations, but `sprint reorder` always produces a gapless sequence.

**Rationale**: Typical sprints have 5-15 items. The cognitive overhead of managing sparse ordering (e.g., "item is at position 7 but there are only 4 items") outweighs the performance gain of sparse inserts. The feature spec already leans toward dense, and the human mental model of "item 3 of 8" requires it.

### 5.2 Default view for `sprint backlog`

**Decision**: Default to ordered view for active sprints; grouped view for all other sprint states.

**Rationale**: Active sprint = execution mode. The PM's question is "what's next?" Ordered view answers that. Planning sprint = scoping mode. Grouped view by status is more relevant. Completed sprint = retrospective mode. Grouped view by completion status is more relevant. Switching the default based on sprint lifecycle state means the most useful view appears without flags in the common case.

**Risk**: PMs used to the current grouped view may be confused by the new default during active sprints. Mitigate with a header line indicating which view is shown: `Pull Queue: S005` vs. `Backlog: S005`.

### 5.3 Whether `sprint_order` field appears in all backlog JSON output

**Decision**: Include `sprint_order` on every item in all `sprint backlog --json` output, regardless of view. Null for unordered items.

**Rationale**: The agent always needs the raw data. View flags affect display format only. Hiding `sprint_order` in grouped view would require the agent to switch views to get the field, which is unnecessary friction.

### 5.4 `sprint next` selection_reason field

**Decision**: Include `selection_reason` as a required field in `sprint next --json` output.

**Rationale**: Without it, the agent cannot distinguish "this item was returned because it was explicitly prioritized" from "this item was returned because it happened to be inserted first." The field costs nothing to produce (it is a byproduct of the sort algorithm already being run) and gives the agent meaningful signal for logging and anomaly detection.

### 5.5 Carryover order preservation

**Decision**: `--carryover=next` preserves relative order, renumbering from 1 in the receiving sprint. `--carryover=backlog` clears sprint_order.

**Rationale**: When carrying to the next sprint, the PM's deliberate ordering should survive. Renumbering from 1 in the new sprint gives a clean sequence without gaps from completed items. When returning to backlog, sprint-relative order is meaningless outside the sprint context; clearing it avoids stale ordinal data from polluting future planning.

**Edge case**: If the receiving sprint already has ordered items, carryover items should be appended after them (sprint_order = max(existing) + 1, 2, 3...). The PM can then reorder as desired. Do not interleave carryover items with existing items by priority — that would mutate the deliberate ordering of both groups.

### 5.6 No auto-ordering based on priority or dependencies

**Decision**: The system never automatically sets `sprint_order` based on priority, ExecutionOrder, or dependency graphs. Order is always explicit — set by the user via `--at` or `reorder`, or defaulting to FIFO (assigned_at order, sprint_order = max+1).

**Rationale**: Auto-ordering would conflate sprint-level intent with project-level signals, which is the exact problem this feature is solving. The fallback sort in `GetNextTask` (ExecutionOrder → priority → FIFO) already handles unordered items gracefully. Auto-ordering would produce invisible overrides that are hard to detect and reason about. The separation must stay clean.

---

*CX Design Feedback produced by CXDesigner agent for E19-F07.*
*Reference sources: feature.md, sprint_service.go (GetNextTask, GetSprintBacklog, BacklogItemView), sprint.go (command surface), user-journeys.md, personas.md, epic.md.*
