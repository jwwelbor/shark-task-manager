# /shark-rider run-sprint-team — Team sprint topology alias

Run the active sprint through the canonical team topology.

Usage: `/shark-rider run-sprint-team S###`

## Procedure

1. Validate the `S###` key, then follow
   `skills/sprint-execution/workflows/run-sprint-team.md`. That workflow reads
   the live sprint and allows delegation only from its configured
   execution-phase status; validation never starts or closes a sprint.

2. After that read-only preflight succeeds, delegate to:

   ```
   /shark-rider run-agent-team --sprint S###
   ```

3. Report the resulting terminal or paused state. Ask the owner whether to
   close the sprint only after the run; never start or close it automatically.

## Result

The team path uses the active backlog and the topology adapter. It does not
group entities by feature, create nested teams, or replace the solo
`/shark-rider run-sprint` workflow.
