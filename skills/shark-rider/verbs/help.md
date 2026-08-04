# /shark-rider help — Verbs & state-aware next actions

Usage:
```
/shark-rider help            # state-aware: read project state, suggest next actions
/shark-rider help --fast     # static verb list, zero shark calls
/shark-rider help commands   # static Shark command reference, zero shark calls
/shark-rider help <verb>     # static help for a known /shark-rider verb
```

## Static help: `--fast` and `commands`

Treat `commands` as a static alias for `--fast`. Do not run any `shark` command.
Print the verb groups plus the compact command reference below.

```
Getting started:  /shark-rider project bootstrap | product-design | vision "idea"
Day-to-day:       /shark-rider run <key> | triage "desc" | demo <epic-key|feature-key> [--draft] | walkthrough <entity-key|docs-path> [scope] | viewer | status | list <key> | get <key>
Sprint:           /shark-rider plan-sprint <key> | run-sprint <key> | run-agent-team <epic-key|feature-key> | run-sprint-team <sprint-key> | retro-sprint <key>
Maintenance:      /shark-rider update-docs | amend <key> "change" | revalidate <key> | help [commands|<verb>]

Read:             shark status [key] | shark list [epic] [feature] | shark get <key> | shark view <key> | shark search "query"
Create:           shark create epic|feature|task|bug|change|tech-debt|idea|note ...
Workflow:         shark status advance <key> --outcome pass|fail|blocked | shark status set <key> <status>
Leases:           shark claim <key> | shark heartbeat <key> | shark release <key> [--session SID | --force] | shark claims
Notes:            shark create note <key> "text" --type=comment | shark notes search "query"
Select:           shark plan [root|collection] --json (read-only; never claims)
Dispatch:         shark next <key> --json (may normalize workflow status)
Bundle content:   shark skill get <name> [path] | shark agent get <name>
```

## Specific command help

### Known verbs

If the user asks `/shark-rider help <verb>` for a known verb, use the table
below. Print the matching static entry. Do not run any `shark` command. Do not
fall through to state-aware help.

| Verb | Static help |
|------|-------------|
| `project bootstrap` | Bootstrap architecture docs through `shark skill get research workflows/bootstrap.md`. Afterward suggest product design, vision capture, or `/shark-rider run <key>`. |
| `product-design` | Run the bundled product-design D01-D14 methodology through `shark skill get product-design`. |
| `vision` | Turn an idea into a Shark epic through the bundled epic-writing workflow, then offer `/shark-rider run <epic-key>`. |
| `run` | Drive an epic, feature, task, bug, change-card, or tech-debt item through its workflow. Use `/shark-rider run <key>`. |
| `plan` | Recommend an execution shape for `shark plan [root\|collection]`. It does not claim, dispatch, advance, or launch a team. |
| `triage` | Capture, classify, confirm, create, and stop. Use `/shark-rider triage "thing to track"`. |
| `demo` | Prepare an evidence-based demo for an epic or feature. Use `/shark-rider demo <epic-key|feature-key> [--draft]`; it is not a UAT gate or workflow action. |
| `walkthrough` | Walk an entity or authoritative document through solution decisions and ratify reviewed directions. Use `/shark-rider walkthrough <entity-key\|docs-path> [overall\|section\|decision-id]`; it does not change workflow status. |
| `code-review` | Multi-angle review. Flags: `--fix` for safe fixes, `--comment` for GitHub PR comments. |
| `brownfield-analysis` | Deep existing-codebase analysis and documentation. |
| `viewer` | Launch the local web dashboard through `shark web`. |
| `status` | Read project or entity status: `/shark-rider status` or `/shark-rider status <key>`. |
| `list` | List epics, features, tasks, or standalone entity collections: `/shark-rider list`, `/shark-rider list E01`, `/shark-rider list E01 F02`. |
| `get` | Read one entity by key: `/shark-rider get <key>`. |
| `plan-sprint` | Scope a sprint from backlog and readiness data, then ask before adding work. |
| `run-sprint` | Sequential sprint pull-loop using `/shark-rider run` per entity. |
| `run-agent-team` | Run a selected root through the canonical topology adapter. The adapter selects keys and assigns each to an ordinary keyed Rider parent. |
| `run-sprint-team` | Run an active sprint through the canonical topology adapter. It uses the active backlog and retains an explicit owner-only close gate. |
| `retro-sprint` | Post-close sprint retrospective from Shark sprint analytics. |
| `update-docs` | Diff-driven refresh of `docs/architecture/*`. |
| `amend` | Apply a spec change to an entity and rewind it to the appropriate phase. |
| `revalidate` | Audit spec, task readiness, and workflow state before more execution. |
| `help` | This help. Use `/shark-rider help commands` for a static reference. |

### Unknown verbs

Show `/shark-rider help commands` and suggest `/shark-rider <command>`
passthrough for a direct CLI query. Do not run any `shark` command. Do not fall
through to state-aware help.

## Bare `/shark-rider help` (state-aware)

Run bare `shark plan` exactly once:

```bash
shark plan --json
```

Treat the Shark selection as the live workflow and relationship authority. It
carries no worker prompt — it is a bounded selection, not advice to relay
verbatim.

Branch on `action`:

- `select_epic` — report the returned `entity` as the recommended next root
  and offer `/shark-rider run <entity_key>`.
- `parallel_candidates` — report the tied epics from `entities`. Inspect
  `docs/product/progress.md` and `docs/product/cross-epic-integration-map.md`
  when they exist for why-now context, then recommend one of: run one
  candidate sequentially, run independent candidates in parallel via an agent
  team, or report that the tie needs a human decision.
- `pause` — report the `reason` (and any `warnings`) and state the next
  evidence or relationship fix. Do not guess a root.

Stop at the recommendation. The operator must separately invoke
`/shark-rider run <recommended-key>` to start keyed dispatch. Do not call keyed
`shark next <key>` from help. Do not spawn a subagent. Do not claim, advance,
or automatically run any recommended epic.
