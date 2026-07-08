# Workflow & Status Management (Shark 2.x)

Shark 2.x is **route-based**: each workflow step declares an `outcomes:` map and
agents release a semantic outcome (`pass` / `fail` / `blocked` / …); the engine
routes to the next step. Status is a pure phase; the **claim is the work lease**.

## Status commands

### Advance by outcome (preferred for workflow)

```bash
shark status advance E01-F02-001 --outcome pass      # route via outcomes[pass]
shark status advance E01-F02-001 --outcome fail      # route back (auto-authorized)
shark status advance E01-F02-001 --outcome blocked   # route to the blocked/parking step
shark status advance E01-F02-001                     # bare: auto-select default next status
```

The JSON response includes `orchestrator_action` with agent-routing instructions.
A backward route (e.g. `fail`) is authorized automatically — the configured route
is the authority — and the outcome is recorded as the transition reason. An
unknown outcome lists the valid set.

### Set (direct status assignment)

```bash
shark status set E01-F02-001 active                  # idempotent
shark status set E01-F02-001 blocked --reason "Missing API key"
shark status set E01-F02-001 <status> --force        # human escape hatch (skips route validation)
```

### Transitions (read-only)

```bash
shark status transitions E01-F02-001    # valid next statuses / outcomes from current state
```

### History

```bash
shark status history E01-F02-001        # chronological changes with timestamps and agents
shark status history E01-F02-001 --json
```

### Dashboard

```bash
shark status            # project-wide dashboard
shark status E01        # epic status with feature rollups
shark status E01-F02    # feature status with task breakdown
```

## Claim / session lease

Status no longer encodes "in progress" via an `in_*` marker — an agent **claims**
the entity to hold the work lease.

```bash
shark claim E01-F02-001 --by dev-agent          # acquire lease → prints a session id
shark claim E01-F02-001 --force                 # steal a live claim
shark heartbeat E01-F02-001 --session $SID --progress 0.5 --note "tests passing"
shark release E01-F02-001 --session $SID         # safe, session-scoped release (alias: unclaim)
shark release E01-F02-001                        # unconditional administrative release
shark claims                                     # list active leases
```

- One claim per entity (atomic single-grab); expired leases are reclaimed
  automatically (TTL default 15 min; `.sharkconfig.json` `claim_ttl_seconds`
  overrides first, then `SHARK_CLAIM_TTL_SECONDS`; `0` disables expiry).
- `shark next <root>` hands out only **unclaimed** entities.

## Orchestrator actions in responses

When reading/advancing an entity, the JSON may include routing instructions:

```json
{
  "task": { "key": "E01-F02-001", "status": "specification" },
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "architect",
    "skills": ["architecture", "specification-writing"],
    "instruction": "…fully-rendered prompt…"
  }
}
```

**Actions**: `spawn_agent`, `advance_status`, `cascade`, `pause`, `archive`.
Always read and follow the `instruction` field when present. The `instruction`
is already resolved from the bundle's prompts/skills/agents — no external skill
is needed.

## Workflow definitions

Workflows are per-entity YAML under the active `workflow_config` target (a
directory of `<entity>.yaml`, or a master index file mapping each entity to its
file). Each step block carries `phase`, `progress_weight`, `action`, optional
`agent`/`model`/`skills`/`prompt`, an `outcomes:` map, and `aliases:` (legacy
status names that resolve to this step). Do not hardcode status names — derive
phases and order from the YAML.

## Feature/epic status

Features and epics use the same commands:

```bash
shark status advance E01-F02 --outcome pass
shark status set E01-F02 active
shark status transitions E01-F02
shark status advance E01 --outcome pass
```

## Batch operations

```bash
for key in E01-F02-001 E01-F02-002 E01-F02-003; do
  shark status advance "$key" --outcome pass
done
```

Do NOT use `--force` to bypass route validation in automation. If a transition
fails, check `shark status transitions KEY` to understand why.
