# /shark-rider project — Progress-driven project setup

Read `skills/shark-rider/context/project-bootstrap-execution.md` as the
execution map before coordinating child actions.

The `project` namespace coordinates one-time project setup. `bootstrap` is the
resumable coordinator; `product-design` and `brownfield-analysis` remain
directly callable child actions. Shark CLI commands in this file are owned by
this Rider action. A child action owns its own bundle retrieval and CLI
integration; bootstrap does not duplicate either child's adapter.

Usage: `/shark-rider project <bootstrap|product-design|brownfield-analysis> [args]`

## Dispatch

With no activity, print the available activities and stop. Do not silently run
bootstrap. For `product-design` and `brownfield-analysis`, read the matching
Rider action and pass the remaining arguments through unchanged.

Unknown activities stop with:

> Unknown activity: `<x>`. Valid activities: bootstrap, product-design, brownfield-analysis

## Bootstrap coordinator

`/shark-rider project bootstrap` repeats inspection and status derivation on
every entry, then resumes the first required or explicitly selected activity.
It never treats the previous checklist text as authoritative and never
restarts evidence that already exists.

### Stage 0 — Prepare the local Shark instance

1. Inspect `.sharkconfig.json`, the configured database/store, `docs/plan/`,
   and the installed content bundle. Preserve existing configuration and data.
2. If Shark is not initialized, run the safe idempotent path:
   `shark admin init --non-interactive`. If initialization fails, report the
   concrete error and stop before creating misleading setup artifacts.
3. If Shark is initialized, run `shark admin validate`. Do not use `--force`,
   overwrite a store, or delete a database. Repair missing generated support
   files only through the normal idempotent CLI path.
4. Confirm that the research bundle is available through the owning adapter:
   `shark skill get research workflows/bootstrap.md`. If unavailable, report
   the bundle error and stop.

The CLI owns the database, configuration, workflow, routing, and content
retrieval. Rider owns the surrounding coordination and local document craft.

### Stage 1 — Seed or refresh the progress record

Create `docs/product/progress.md` immediately after Stage 0, before dispatching
product or brownfield work. If it is missing, create `docs/product/` and seed
it from `file_templates/progress.md`. If it exists, preserve its Decision Log
and Cross-Epic Integration Map sections while refreshing only derived content.

If the file is malformed, copy it to a timestamped recovery file, report the
problem, and do not destructively rewrite it. If the path is read-only, warn
that the checkpoint was not saved and continue routing from observed evidence.

### Stage 2 — Confirm irreducible project parameters

Derive what is safe to derive, then ask only for values that cannot be known
from the repository or artifacts. Record each answer, approval, reason, or
deferral in the append-only Decision Log:

| Parameter | Allowed values | Meaning |
|---|---|---|
| Repository estate | `greenfield`, `brownfield` | What exists in the repository now; derive from evidence first. |
| Initiative posture | `stack-only`, `new-capability`, `extend`, `modernize`, `replace` | What this initiative intends to do; never infer from estate. |
| Product-design scope | `D01-D07`, `D01-D11`, `D01-D14`, or `custom` | How far bootstrap coordinates before handoff. |
| Brownfield depth | `none`, `lightweight`, `comprehensive` | Whether full analysis is needed beyond the observed baseline. |
| Constraints | hard, soft, unresolved | Source and owner for each constraint or open question. |
| Setup customization | bundle, workflow, or local conventions | Project-specific choices that do not belong in a project entity. |

At this checkpoint, refresh `progress.md` so the selected scope and next
required activity survive an interrupted run.

### Stage 3 — Select the estate branch

Use repository evidence in this order:

1. A build manifest with real dependencies means brownfield with high
   confidence.
2. More than five source files means brownfield with high confidence. One to
   five source files is ambiguous and requires confirmation.
3. More than three meaningful git commits means brownfield with medium
   confidence. A fresh template clone remains greenfield unless confirmed
   otherwise.
4. No decisive signal means greenfield with high confidence.

Record the signals and confidence in the progress metadata and Decision Log.
Estate is evidence about the repository, not a constraint on the initiative.

### Stage 4 — Coordinate discovery

#### Greenfield

For a normal product initiative:

1. Keep architecture input minimal and provisional. Do not generate a final,
   prescriptive foundation before D07.
2. Load and follow the sibling Rider procedure
   `skills/shark-rider/verbs/product-design.md` for the selected scope in this
   same coordination run. Do not emit `/shark-rider project product-design` as
   a nested dispatch: this worker is already inside the Rider host procedure.
   The child procedure owns retrieval of the `product-design` bundle and
   checkpoints each D artifact, then returns a structured child result here.
3. Treat D04 as preliminary feasibility against constraints and proposed
   options. Continue through D07 before the architecture integration gate.
4. Reconcile the proposed target against D01, D04, and D07. The gate runs even
   when D04 says `feasible as described`; that outcome clears provisional state
   only after reconciliation.
5. Ask for approval when the target architecture contains a material human
   choice. Record approval or deferral, then continue to the selected D08-D11
   or D08-D14 scope.

If the user explicitly selects `stack-only`, the complete scaffold may be
generated immediately, but the posture and reason must be recorded. If D07 is
deferred, leave architecture partial or provisional and do not claim setup is
ready for epic handoff.

#### Brownfield

1. Produce or refresh the lightweight observed baseline early enough to inform
   D01 and D04. It describes the current system; it does not approve it as the
   target.
2. Load and follow the sibling Rider procedure
   `skills/shark-rider/verbs/brownfield-analysis.md` only when
   `brownfield_depth: comprehensive` or the risk profile requires it. Do not
   emit a nested `/shark-rider project brownfield-analysis` dispatch. The child
   procedure checkpoints each selected analysis area, then returns its result
   to this coordinator.
3. Feed architecture, integrations, quality, debt, security, and migration
   evidence into product design.
4. Classify constraints as hard, soft, or unresolved with a source and owner.
   Apply `new-capability`, `extend`, `modernize`, or `replace` to choose the
   target and transition strategy; do not declare the current stack fixed by
   default.
5. Run the architecture integration gate and route only genuinely deferred
   remediation to Shark entities.

For a new capability in an existing monorepo, record `estate: brownfield` and
`initiative_posture: new-capability` separately.

### Stage 5 — Resume and hand off

After every child return, inspect evidence again and refresh the progress
record. Select the first missing or partial item in the confirmed scope. A
successful setup handoff requires the selected artifacts, visible blockers and
deferred decisions, and a reconciled or approved architecture state as
required by the initiative posture. Suggest the first epic-planning action;
do not move recurring maintenance into the setup checklist.

### Child action return contract

Every child action must return a result before control comes back here:

```text
CHILD ACTION RESULT
action: product-design | brownfield-analysis
outcome: completed | partial | blocked
artifacts: <durable files written or updated>
checkpoint: saved | read-only-warning | recovery-file
next_step: <first missing artifact or explicit blocker>
stack_feedback: <none or the D04 feedback signal>
```

The coordinator consumes this result, re-inspects the repository, refreshes
`docs/product/progress.md`, and decides the next action. A child action never
starts another host-level Rider dispatch. If D04 requests reconciliation, the
product-design child returns `stack_feedback` and this coordinator re-enters
its own reconciliation stage.

## Progress checkpoint contract

The checklist is derived and advisory. Routing uses current files,
configuration, artifact metadata, and child outcomes—not manually edited
checkboxes. At bootstrap entry, before a pause, before a child dispatch, after
each child artifact/analysis area, after architecture changes, and before
handoff:

1. Inspect current evidence.
2. Preserve frontmatter fields that are informative but not derivable and
   preserve all existing Decision Log and Cross-Epic Integration Map content.
3. Regenerate the derived Setup Checklist and artifact status.
4. Append a Decision Log entry only for a decision, approval, reason, or
   deferral; never append a heartbeat-only entry.
5. Use the skill `Write` operation to regenerate the derived rows from current
   evidence and write the file atomically when supported. The checklist is
   advisory: do not treat checkbox text as routing input, and do not claim a
   save after a write error.

### Frontmatter and maturity

Maintain these advisory fields in `docs/product/progress.md`:

```yaml
type: progress-record
schema_version: 2
estate: greenfield | brownfield
initiative_posture: stack-only | new-capability | extend | modernize | replace
product_design_scope: D01-D07 | D01-D11 | D01-D14 | custom
brownfield_depth: none | lightweight | comprehensive
architecture_state: absent | observed | proposed | provisional | reconciled | approved
stack_summary: <informative value>
artifact_paths: <informative map or list>
last_refreshed: <ISO-8601 timestamp>
```

Read a legacy `track` field as an estate hint during migration only. New
decisions must use `estate`; `track` is not a substitute for posture.

Use these checklist markers:

| Marker | Evidence meaning |
|---|---|
| `[x]` | Required evidence exists and its completion rule is satisfied. |
| `[~]` | Work started, evidence is partial, or an artifact remains provisional. |
| `[ ]` | Required evidence is absent or has not started. |
| `[-]` | Explicitly not applicable or deferred; cite the Decision Log. |

Derived groups must cover local Shark readiness, setup definition, brownfield
evidence, every artifact in the selected D01-D14 scope, architecture
integration, and setup handoff. Keep the Cross-Epic Integration Map as a
separate product-level section owned by epic design.

For architecture maturity, prefer explicit frontmatter or a consistent
metadata header (`provenance` and `maturity`). During migration, recognize
legacy provisional markers broadly, including readiness-3 documents whose
wording is merely `Provisional`; never decide maturity from the exact phrase
`Greenfield — Provisional placeholder` alone.

## Failure and recovery rules

- Missing or unreachable Shark: report the concrete blocker and stop before
  product or architecture generation.
- Missing bundle content: report the owning `shark skill get` failure and stop
  that child action; do not copy retrieval logic into a parent.
- Read-only progress path: warn that the checkpoint was not saved; continue
  from observed evidence without claiming persistence.
- Malformed progress: preserve the original and recover non-destructively.
- Interrupted child: leave completed artifacts marked `[x]`, the active item
  `[~]`, and resume from the next evidence gap on re-entry.
