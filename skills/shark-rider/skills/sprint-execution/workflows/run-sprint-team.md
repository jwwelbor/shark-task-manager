# Workflow: /shark-rider run-sprint-team

Run an active sprint through the canonical team topology without creating a
second sprint dispatcher.

## Prerequisites

Accept one sprint key matching `S###`. The sprint must already be in the
configured execution-phase status (the bundled workflow calls it `active`).
Only the owner may start or close a sprint.

## Procedure

1. Validate the sprint key. If it is invalid, stop without calling Shark:

   ```
   /shark-rider run-sprint-team only operates on sprints. Got: {KEY}
   ```

2. Read the live sprint without changing it:

   ```bash
   shark sprint get {SPRINT_KEY} --json
   ```

   Resolve the configured execution-phase status before comparing: follow the
   project's `workflow_config` to its sprint workflow (or use the embedded
   sprint workflow when no disk workflow is configured), then select the single
   status whose metadata has `phase: execution`. If that status is absent or
   ambiguous, report `Sprint {SPRINT_KEY} execution phase could not be resolved;
   no team run was started.` and stop. Continue only when the sprint's `status`
   equals that resolved execution-phase status (`active` in the bundled
   workflow); never assume a literal status label.

   - If the sprint is not found or the read fails, report `Sprint {SPRINT_KEY}
     is unavailable; no team run was started.` and stop.
   - If its status is `planning`, report `Sprint {SPRINT_KEY} is planning. Only
     the owner may start it; no team run was started.` and stop.
   - If its status is terminal, closing, on hold, or otherwise outside the
     execution phase, report `Sprint {SPRINT_KEY} is not active (status:
     {STATUS}); no team run was started.` and stop.

3. Invoke the topology entrypoint:

   ```
   /shark-rider run-agent-team --sprint {SPRINT_KEY}
   ```

4. Let the canonical Shark Attack `parallel-team.md` procedure use the active
   backlog as the sole selection universe and assign each selected key to an ordinary
   keyed `/shark-rider run` parent. Do not group items by feature, make nested
   teams, construct prompts, or call `shark sprint next` to make a wave.
5. Report the terminal or paused state. If all work is terminal, ask the owner
   whether to close the sprint. Do not start or close it automatically.

## Result

The team alias keeps active-backlog selection, topology, Question routing, and
integration in the canonical adapter while the solo `/shark-rider run-sprint`
workflow remains unchanged.
