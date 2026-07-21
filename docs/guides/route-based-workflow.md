# Route-Based Workflow (Shark 2.x)

> **Status:** Implemented behind the existing engine (Epic E35). The default
> shipped workflows still use the legacy `status_flow` + `status_metadata`
> shape; the route-based `steps:` schema is fully supported and exercised by
> tests, and any workflow file may opt into it. Both shapes coexist — the
> loader derives the legacy maps from `steps:` so every existing reader keeps
> working.

This guide describes the consolidated, route-based workflow model introduced in
Epic E35. The full design rationale is in
[`route-based-workflow-redesign.md`](../plan/route-based-workflow-redesign.md).

---

## 1. One block per step (`steps:`)

The legacy schema split each status across two maps: `status_flow` (the
transition graph) and `status_metadata` (color/phase/weight/agent/action). The
route-based schema merges them into a single per-step block.

```yaml
# workflow/feature.yaml
version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    progress_weight: 0.0
    action: advance_status          # auto, no agent
    outcomes: { pass: refinement }

  qa:                               # was ready_for_qa + in_qa
    phase: qa
    color: cyan
    progress_weight: 0.85
    responsibility: agent
    action: spawn_agent
    agent: qa
    provider: anthropic
    model: sonnet
    skills: [quality]
    prompt: feature/qa.md           # was instruction_template
    aliases: [ready_for_qa, in_qa]  # migration + compat (see §5)
    outcomes:
      pass: approval
      fail: development             # route back
      blocked: on_hold

  on_hold:
    phase: paused
    parking: true                   # resume target computed from history

  completed:
    phase: done
    progress_weight: 1.0
    terminal: true                  # no outcomes
```

Step fields: `phase`, `color`, `description`, `progress_weight`,
`responsibility`, `agent_types`, `action`, `agent`, `provider`, `model`,
`skills`, `prompt`, `outcomes`, `aliases`, `parking`, `terminal`,
`blocks_feature`, `is_planning`, `aggregates_from`, `exclude_from_progress`.

The `ready_for_X` / `in_X` pair collapses: `in_X` becomes the *claim* (§4),
`ready_for_X` becomes the bare phase name.

---

## 2. Outcome routing (`outcomes:`)

Each non-terminal, non-parking step defines an `outcomes:` map. A skill does its
craft and **releases a semantic outcome**; the engine looks up
`step.outcomes[outcome]` and routes there. Skills never name a target status, so
reordering or renaming steps never touches a skill.

**Core vocabulary** — every workable step must define `pass`, `fail`, and
`blocked`. Steps may add extras (`dead-end`, `needs-info`, …).

Release an outcome from the CLI:

```bash
shark status advance E07-F01-001 --outcome pass     # route via outcomes[pass]
shark status advance E07-F01-001 --outcome fail     # route back (auto-authorized)
shark status advance E07-F01-001 --outcome blocked
```

A backward route (e.g. `fail`) is authorized automatically — the configured
route is the authority — and records the outcome as the transition reason. An
unknown outcome lists the valid set. `shark status set <key> <status> --force`
remains the human escape hatch for direct status changes.

If `.sharkconfig.json` enables `advance_guard`, parent-run advances must also
send the claim session id and expected current status:

```bash
shark status advance E07-F01-001 --outcome fail --session "$SID" --from-status qa
```

---

## 3. Master index & bundle-rooted resolution

`workflow_config` in `.sharkconfig.json` may point at a **master index file**
that maps each entity to its workflow file:

```yaml
# shark-data/workflow.yaml  (the index)
entities:
  task:      workflow/task.yaml      # relative to the index's directory
  feature:   workflow/feature.yaml
  epic:      workflow/epic.yaml
  bug:       /shared/bundles/std/workflow/bug.yaml   # absolute = shared bundle
```

- Relative entity paths resolve against the index file's directory (the
  **bundle root**).
- Absolute paths point anywhere on the filesystem — the entire "remote bundle"
  story (shared mount / monorepo / submodule). No fetch, cache, or pinning.
- The bundle root is also the resolution base for prompts/skills/agents;
  `overrides/` layers on top.

`workflow_config` also accepts a directory of per-entity YAML files. It no
longer accepts Shark 1.x JSON workflow files as explicit targets. To use
embedded defaults, remove the field and remove or rename any root
`.sharkworkflow.json`. Or run `shark admin install-shark-data` to extract
editable YAML and set `workflow_config` to the installed bundle's `workflow/`
directory.

---

## 4. Claim / session lease

Status is a pure phase; the **claim is the lease**. Keyed `shark next <root>`
selects only unclaimed dispatchable entities. Bare `shark next` is read-only
portfolio advice and does not select or lease an entity. After keyed selection,
an agent claims the entity before working it. Heartbeats renew the lease, and a
TTL backstop reclaims dead leases.

```bash
shark claim E07-F01-001 --by dev-agent          # prints a session id
shark heartbeat E07-F01-001 --session $SID --progress 0.5 --note "tests passing"
shark release E07-F01-001 --session $SID         # safe session-scoped release (alias: unclaim)
shark claims                                     # list active claims
```

- One claim per entity (atomic single-grab).
- `--force` steals a live claim; expired leases are reclaimed automatically.
- TTL defaults to 15 minutes. Override with `.sharkconfig.json` `claim_ttl_seconds`,
  or `SHARK_CLAIM_TTL_SECONDS` when the config field is absent. Set
  `claim_ttl_seconds` to `0` to disable lease expiry entirely.
- Updates do triple duty: lease renewal + progress + telemetry.

### Dispatch loop (claim → run → release)

The harness dispatch loop becomes:

```
loop:
  entity = shark next <root>        # returns only unclaimed entities
  shark claim <entity> --by <agent> # acquire the lease (session id)
  ... run the agent for the step ...
  shark heartbeat <entity> --session <sid> --progress <p>   # periodically
  shark status advance <entity> --outcome <pass|fail|blocked> --session <sid> --from-status <status>
  shark release <entity> --session <sid>                    # release the lease
  goto loop
```

---

## 5. Status migration & compat (`aliases:`)

Each step's `aliases:` lists the old status names that collapse into it. The
alias map does triple duty:

1. **Input compat shim** — old status names (`ready_for_qa`) are accepted and
   resolved to the new step (`qa`) during the deprecation window, so hooks,
   scripts, and muscle memory keep working.
2. **History-read resolution** — entities parked under an old name resolve on
   read.
3. **One-shot migration** — rewrite the live `status` column:

   ```bash
   shark admin migrate statuses          # dry-run: report what would change
   shark admin migrate statuses --apply  # execute (single transaction)
   ```

   This is **destructive and opt-in** (dry-run by default). `task_history` is
   left untouched — audit trails record what actually happened and alias-resolve
   old names on read.

---

## 6. Validation

`shark admin validate` (and the workflow validator) add route-based checks on
top of the existing reachability/terminal-path rules:

- the `start:` step is defined;
- every workable step defines the core outcomes (`pass`/`fail`/`blocked`);
- every outcome target names a defined step;
- no old-status alias is claimed by two steps.

---

## 7. Compatibility

Nothing is forced to the new shape. The loader projects `steps:` onto the legacy
`status_flow`/`status_metadata`/`special_statuses` maps, so:

- existing status calculations, display, progress, and dispatch keep working;
- a workflow file may be migrated entity-by-entity;
- the live default workflows remain on the legacy shape until explicitly
  switched.

See the [redesign design doc](../plan/route-based-workflow-redesign.md) for the
locked decisions (D1–D7) and the full blast-radius analysis.
