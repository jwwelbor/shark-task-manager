# Workflow: /retro-sprint

Post-close sprint retrospective workflow. Reads closed sprint data from shark — detailed summary, velocity history, and per-entity notes — then synthesizes a structured five-section retro report. Recommendations are generated from measurable data thresholds, not free-form advice.

---

## Step 0: Argument Parse

Parse `$ARGUMENTS`:

1. Extract the sprint key (first positional argument).
2. Validate: the key must match the pattern `S` followed by one to three digits (e.g., `S001`, `S42`, `S999`). Case-insensitive (accept `s001` as `S001`).
3. If the key does not match, stop immediately:
   ```
   /retro-sprint only operates on sprints. Got: {KEY}
   ```
   Do not call any shark commands.
4. Parse `--no-write` flag (optional; if present, print the retro report to stdout instead of writing to disk).

---

## Step 1: Verify Sprint Is Closed

```bash
shark get {S###} --json --field=status
```

Parse the returned value as `STATUS`.

**If `STATUS` is `completed` or `archived`**: proceed to Step 2.

**If `STATUS` is anything else** (e.g., `planning`, `active`, `cancelled`): stop immediately with:
```
/retro-sprint requires a completed or archived sprint. {S###} is in status: {STATUS}. Close the sprint first with /shark-rider run-sprint or shark sprint close.
```

Do not pull any further data.

---

## Step 2: Pull Data

Execute the following three data pulls in order. Store all results for use in Step 3 and Step 4.

### 2a. Detailed Sprint Summary

```bash
shark sprint summary {S###} --detailed --json
```

Store the full JSON response as `SUMMARY`. Key fields to extract:

- `planned_count` — total entities planned at sprint start
- `completed_count` — entities that reached terminal status (`completed`)
- `carryover_count` — entities that carried over to the next sprint
- `planned_size` — Σ size of all planned entities
- `completed_size` — Σ size of completed entities
- `carryover` — list of entity keys that carried over (may be empty `[]`)
- `rejected` — list of entity keys that were rejected/blocked without completion (may be empty `[]`)
- `cycle_times_by_phase` — per-phase cycle time data (from `--detailed`); if absent, record `null`
- `agent_allocations` — per-agent-type planned vs. completed data (from `--detailed`); if absent, record `null`

### 2b. Sprint Velocity

```bash
shark sprint velocity --json
```

Store the full JSON response as `VELOCITY`. Key fields to extract:

- `this_sprint_velocity` — velocity (tasks or size points completed) for `{S###}`
- `trailing_average` — average velocity over the preceding N sprints (N defined by shark's velocity config)
- `trailing_sprints` — list of recent sprint velocity data points (for sparkline context)
- `variance_pct` — absolute percentage difference between this sprint and trailing average; compute if not present:
  ```
  variance_pct = abs(this_sprint_velocity - trailing_average) / trailing_average * 100
  ```
  If `trailing_average` is 0 or null (no prior sprints available), set `variance_pct = null` (cannot compute).

### 2c. Carryover and Rejection Notes

For each entity key in `SUMMARY.carryover` and `SUMMARY.rejected` (deduplicated):

```bash
shark notes {entity_key}
```

Store results as `NOTES[entity_key]` — the raw notes output for that entity. If an entity has no notes, store an empty result (do not error).

This step may produce zero calls if both lists are empty.

---

## Step 3: Synthesize Recommendations

Apply the five pattern-match rules below against the collected data. For each rule whose condition is true, generate one recommendation item. Collect all triggered items, then emit the top 3–5 by signal strength (rules listed in signal-strength order — apply all, emit top results).

**Rule 1 — Velocity variance** (signal: high if variance > 50%, medium if 25–50%):
- **Condition**: `variance_pct > 25` (and `variance_pct` is not null)
- **Recommendation**: `"Velocity was {this_sprint_velocity} vs. trailing average of {trailing_average} ({variance_pct:.0f}% variance) — consider re-evaluating sprint capacity before the next sprint."`

**Rule 2 — XL task completion** (signal: high if any completed entity has size ≥ 8):
- **Condition**: any entity in `SUMMARY` completed with `size >= 8`
- **Recommendation**: `"Sprint included {N} entity/entities with size ≥ 8 (XL or larger). Consider splitting XL+ entities into smaller tasks before next sprint planning to reduce cycle-time variance."`

**Rule 3 — High carryover rate** (signal: high if carryover > 30% of planned):
- **Pre-condition guard**: if `SUMMARY.planned_count == 0`, skip Rule 3 entirely (no planned entities → carryover rate undefined).
- **Condition**: `SUMMARY.carryover_count / SUMMARY.planned_count > 0.30`
- **Recommendation**: `"Carryover rate was {carryover_pct:.0f}% ({SUMMARY.carryover_count} of {SUMMARY.planned_count} planned entities) — scope was too aggressive. Consider a capacity buffer of ~{buffer_pct:.0f}% in next sprint planning."`
  - Compute `carryover_pct = carryover_count / planned_count * 100`
  - Compute `buffer_pct = carryover_count / planned_count * 100` rounded up to nearest 5%. Clamp to `[0, 100]`. (Uses only carryover, not rejected entities, since rejected work has no bearing on capacity estimation.)

**Rule 4 — Phase cycle-time imbalance** (signal: high if any phase is > 2× another):
- **Condition**: `SUMMARY.cycle_times_by_phase` is not null AND any phase cycle time exceeds 2× any other phase's cycle time
- Find `SLOWEST_PHASE` (highest cycle time) and `FASTEST_PHASE` (lowest cycle time).
- **Recommendation**: `"Phase '{SLOWEST_PHASE}' cycle time ({slowest_time}) was {ratio:.1f}× longer than '{FASTEST_PHASE}' ({fastest_time}) — investigate the {SLOWEST_PHASE} bottleneck."`

**Rule 5 — Agent-type overallocation** (signal: medium if any agent type allocated > 20% over capacity):
- **Condition**: `SUMMARY.agent_allocations` is not null AND for any agent type: `planned_size > capacity * 1.20`
- Find the most overallocated agent type as `OVER_AGENT`.
- **Recommendation**: `"Agent type '{OVER_AGENT}' was allocated {allocated} size points against a capacity of {capacity} ({overrun_pct:.0f}% over) — rebalance {OVER_AGENT} load in the next sprint."`

**Fallback**: If fewer than 3 rules trigger, add a general observation derived from the data to reach 3 items minimum. Use the highest-signal available data point (e.g., completion rate, velocity trend direction). Do not emit placeholder text like "consider improving processes."

**Never emit** generic advice without a quantitative trigger. Each recommendation must cite a number from the sprint data.

---

## Step 4: Render Markdown Report

Compose the retro report as a single markdown document. Use the structure below exactly — section headers are a fixed contract.

```markdown
# Sprint {S###} Retrospective

**Sprint**: {S###}
**Status**: {STATUS}
**Generated**: {YYYY-MM-DD}

---

## Outcome

| Metric | Planned | Completed | Carryover |
|---|---|---|---|
| Entity count | {planned_count} | {completed_count} | {carryover_count} |
| Σ size points | {planned_size} | {completed_size} | {planned_size - completed_size} |

**Completion rate**: {completed_count}/{planned_count} entities ({completion_pct:.0f}%)

{If carryover_count > 0: "Carryover entities: {comma-separated list of carryover keys}"}
{If rejected is non-empty: "Rejected/blocked without completion: {comma-separated list of rejected keys}"}

---

## Velocity Context

**This sprint**: {this_sprint_velocity} {unit — tasks or size points, whichever shark reports}
**Trailing average** ({trailing_sprints count} sprints): {trailing_average} {unit}
**Delta**: {+/−N} {unit} vs. trailing average ({variance_pct:.0f}% {above/below})

{If trailing_sprints data is available, list last 3–5 sprint velocity values as a simple trend:}
Recent trend: {S(n-3)}: {v1} → {S(n-2)}: {v2} → {S(n-1)}: {v3} → {S###}: {this_sprint_velocity}

{If trailing_average is 0 or null: "Insufficient velocity history for comparison."}

---

## Carryover Analysis

{If carryover list and rejected list are both empty:}
No entities carried over or rejected in this sprint.

{Otherwise, for each entity in the combined carryover + rejected list:}
### {entity_key} — {entity_title if available, else entity_key}

**Disposition**: {carried over | rejected/blocked}

{If NOTES[entity_key] has content:}
**Notes ({count} notes)**:
{For each note in NOTES[entity_key], display:}
- [{note.type}] {note.content} _(added {note.created_at if available})_

{If NOTES[entity_key] is empty:}
No notes recorded for this entity.

---

## Cycle-Time Highlights

{If SUMMARY.cycle_times_by_phase is not null and non-empty:}
| Phase | Avg Cycle Time | Min | Max |
|---|---|---|---|
{For each phase in cycle_times_by_phase:}
| {phase} | {avg} | {min} | {max} |

{Highlight the slowest phase:}
**Slowest phase**: {SLOWEST_PHASE} ({slowest_time} avg) — {ratio:.1f}× the fastest phase ({FASTEST_PHASE}, {fastest_time} avg)

{If cycle_times_by_phase is null or absent:}
Detailed cycle-time data was not available in the sprint summary. Re-run with a shark build that includes `--detailed` phase-level data to populate this section.

---

## Recommendations

{List 3–5 recommendation items from Step 3, as a numbered list:}

1. {Recommendation from Rule N — cite the triggering data value}
2. {Recommendation from Rule N}
3. {Recommendation from Rule N}
{4. and 5. if additional rules triggered}

---

_Retrospective generated by `/retro-sprint` from `shark sprint summary {S###} --detailed --json` and `shark sprint velocity --json` data._
```

---

## Step 5: Write or Print

**If `--no-write` flag is set**:
- Print the full rendered markdown report to stdout.
- Do not touch the filesystem.
- Exit.

**If `--no-write` is NOT set**:

1. Determine the output path: `docs/sprints/{S###}-retro.md` (relative to the project root — the directory where `.sharkconfig.json` or `shark-tasks.db` is located).

2. Check whether `docs/sprints/` directory exists:
   - If not, create it (create the full path `docs/sprints/`).

3. Check whether `docs/sprints/{S###}-retro.md` already exists:
   - **If the file does NOT exist**: write the report directly. Print:
     ```
     Retro report written to docs/sprints/{S###}-retro.md
     ```
   - **If the file EXISTS**: prompt the user:
     ```
     docs/sprints/{S###}-retro.md already exists. Overwrite? (yes/no)
     ```
     - **yes**: overwrite the file. Print confirmation.
     - **no**: exit without writing. Print:
       ```
       Retro report not written. Existing file preserved.
       Use --no-write to print to stdout instead.
       ```

---

## Optional: Archive Offer

After the report is written (or printed with `--no-write`), if `STATUS` is `completed` (not yet `archived`), offer:

```
Sprint {S###} is completed but not yet archived.
Archive now? (yes/no)
```

- **yes**: call `shark sprint archive {S###} --json` and confirm.
- **no**: exit without archiving.

**This prompt is informational and optional.** If the user does not respond or declines, exit cleanly. Do NOT call `shark sprint archive` without explicit user confirmation.

---

## Error Handling

- **`shark sprint summary` returns error**: print the error and exit. Do not attempt to render a partial report.
- **`shark sprint velocity` returns error or no data**: record `trailing_average = null`, set `variance_pct = null`, render the Velocity Context section with "Velocity history unavailable." Do not abort the full retro.
- **`shark notes {entity_key}` returns error for one entity**: record empty notes for that entity, continue with the rest. Log a notice: `Note: could not retrieve notes for {entity_key}.`
- **Sprint key not found** (`shark get` returns not-found): print `Sprint {S###} not found.` and exit.
- **Fewer than 3 recommendation rules triggered**: generate additional observations from the highest-signal data available (see Step 3 Fallback). Never emit fewer than 3 recommendation items as long as any sprint data is available.

---

## Idempotency

This workflow is safe to re-invoke:

- It reads shark data fresh each time. No state is written except the retro file.
- Re-running on the same sprint produces the same report (modulo note additions since the first run).
- Re-running when the retro file already exists triggers the overwrite prompt — the file is never silently overwritten.
- `shark sprint archive` is only called on explicit user confirmation; calling the workflow twice does not double-archive (archived sprints are already in terminal state for this workflow).

---

## Constraints

- All shark calls use `--json` or `--field`.
- `shark sprint summary {S###} --detailed --json` is called exactly once.
- `shark sprint velocity --json` is called exactly once.
- `shark notes` is called at most once per unique entity key in the carryover and rejected lists.
- `shark sprint archive` is only called after explicit user confirmation.
- Recommendations always cite quantitative data from the sprint. No generic placeholders.
- The five section headers (`## Outcome`, `## Velocity Context`, `## Carryover Analysis`, `## Cycle-Time Highlights`, `## Recommendations`) are fixed and must appear in every report, even if a section contains only a "data not available" note.
