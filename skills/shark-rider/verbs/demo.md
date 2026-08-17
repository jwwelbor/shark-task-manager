# /shark-rider demo — Evidence-based demo preparation

Usage:

```
/shark-rider demo <epic-key|feature-key|sprint-key> [--draft]
```

This is an explicit host-local Mode-3 action. It prepares a truthful, portable
walkthrough from documented project evidence; it is not a `shark demo` command,
workflow transition, approval, or replacement for independent UAT.

## Step 0 — Parse and validate the request

Accept exactly one target and the optional `--draft` flag. Reject unknown flags,
additional positional arguments, and a missing target. Resolve the target with:

```bash
shark get <key> --json
```

Only epic, feature, and sprint targets are valid. If the resolved entity is
another type, report the unsupported target and stop; do not guess from the key
shape.
Use the canonical entity key returned by `shark get` for every later command and
artifact path. Reject a key containing a path separator or traversal segment,
and confirm the resolved artifact path remains below `docs/demos/` before the
skill writes or registers it.

## Step 1 — Collect documented state and guidance

Read only the selected scope, its documented requirements/acceptance evidence,
linked documents, notes, and child state using the existing Shark data plane:

```bash
shark get <key> --json
shark list <epic> [feature] --json
# For a sprint target:
shark sprint get <sprint-key> --json
shark sprint backlog <sprint-key> --all --json
# For an epic target:
shark related-docs list --epic=<epic-key> --json
# For a feature target:
shark related-docs list --feature=<feature-key> --json
```

For a sprint target, treat the ordered `shark sprint backlog --all --json`
result as the sprint scope snapshot. Read each assigned item with `shark get
<item-key> --json`, follow its documented parent requirements and linked
guidance, and include completed work as candidate demo material. Keep assigned
work that is incomplete, blocked, or lacks observable evidence visible as
`Not demonstrated / pending integration`; do not silently expand the demo to
unassigned or unrelated work. A completed status selects work for review but is
not evidence that the behavior is demonstrable.

Use only project-documented commands, environments, access, and capture methods
found in that state and its linked guidance. Never infer credentials, endpoints,
deployment steps, or capture tooling.

Read the latest independent assessor verdict separately from the owner decision
and open conditions. For I-01 readiness evidence, preserve the source
`E34-interaction-map.md#i-01-readiness-evidence-shape` and read these nine
fields without changing their meanings: `assessor_verdict`, `owner_decision`,
`open_conditions`, `gate_mode`, `activation_owner`, `closure_key`,
`counterpart_status`, `review_basis`, and `demonstrability_disposition`.
Read counterpart status live from Shark; do not replace an assessor verdict with
an owner decision or completion marker.

## Step 2 — Retrieve and follow the portable craft skill

Retrieve the bundled procedure through its documented interface:

```bash
shark skill get demo-script
```

If it is unavailable, report that the demo-script bundle is unavailable and
stop. Do not copy or substitute a local skill. Pass the collected documented
state and guidance to the retrieved skill so it can create the scenario map and
validate each claim.

Normal mode may put a scenario under **Demonstrated now** only with a source
requirement and documented existing environment/date-scoped evidence. If setup
or capture guidance is absent, report an evidence/decomposition gap instead of
writing an unsupported claim. With `--draft`, the skill may retain an
uncaptured scenario only when it visibly labels the gap. Do not invent commands, credentials, deployments, endpoints, or proof.

Keep the generated artifact's three sections distinct:

- **Demonstrated now** — observable, traced behavior with the required normal-mode evidence.
- **Not demonstrated / pending integration** — missing evidence, `contract-only`,
  or open activation obligations until the activation owner provides closed live
  production-path proof.
- **Accepted risks and overrides** — the independent assessor verdict, separate
  owner decision, open conditions, and accepted risk remain visible. An
  `override-accept` never changes the verdict or turns pending integration into
  demonstrated delivery.

If no complete, observable scenario exists, return the evidence/decomposition
gap and offer `/shark-rider triage` as a candidate. Do not create a backlog item:
triage requires normal deduplication and user confirmation.

## Step 3 — Persist and register a successful artifact

After the skill has successfully created the script at
`docs/demos/<entity-key>/demo-script.md` and retained supporting evidence under
`docs/demos/<entity-key>/evidence/`, register the script with the selected
scope:

```bash
# For an epic target:
shark related-docs add "Demo Script" docs/demos/<entity-key>/demo-script.md --epic=<epic-key>
# For a feature target:
shark related-docs add "Demo Script" docs/demos/<entity-key>/demo-script.md --feature=<feature-key>
# For a sprint target (sprints have no related-document parent option):
shark create note <sprint-key> "Demo script: docs/demos/<sprint-key>/demo-script.md" --type=reference
shark create note <key> "Demo script: docs/demos/<entity-key>/demo-script.md" --type=reference
```

For epic and feature targets, run the related-document and note commands only
after the script is successfully created. For sprint targets, run only the
sprint reference-note command after successful creation. These are discovery
links, not acceptance evidence or lifecycle changes.

## Boundaries

- This procedure does not call claim, status-transition, approval, provisioning, or automatic triage commands.
- It never creates a `shark demo` runtime command, workflow status, entity type,
  schema, API, service, repository, or persistent readiness ledger.
- A demo artifact, completed status, or owner decision is context, not proof;
  this procedure does not repeat, replace, or approve UAT.
