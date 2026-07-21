# /shark-rider query — Default (NL + CLI passthrough)

This is the fallthrough verb for anything that isn't a named verb: a bare entity
key, a direct shark CLI subcommand, or natural-language prose. Answer the
question or run the command — do not invent a workflow.

## Direct CLI passthrough

If the input starts with a shark subcommand, run it verbatim and show the result:

```bash
shark status                 # project dashboard
shark status E01             # epic status with rollups
shark status E01-F02         # feature status with task breakdown
shark list                   # epics
shark list E01               # features in E01
shark list E01 F02           # tasks in E01-F02
shark get  E01-F02-001       # entity details (auto-detected type)
shark view E01-F02-001       # render the markdown file
shark search "auth"          # cross-entity search
shark claims                 # active leases
```

A **bare key** (e.g. `/shark-rider E01-F02-001`) → `shark get <key>`.

## Consult-intent recognizer

Before translating to a query, check whether the request is asking you to *talk to an agent* (not to query project data). Trigger patterns:

| Phrasing pattern | Example |
|-----------------|---------|
| "ask \<agent\> to/about …" | "ask the cx-designer about the onboarding flow" |
| "have \<agent\> look at / review …" | "have the architect review E01-F02" |
| "consult \<agent\> about …" | "consult the qa-engineer about test strategy" |
| "get \<agent\>'s opinion / take on …" | "get the backend-dev's take on the API design" |
| "talk to \<agent\> about …" | "talk to the product-designer about the vision" |

Extract `<agent>` (the role/persona name) and `<referent>` (what to discuss). If `<agent>` resolves to a known shark agent persona, **do not proceed to NL routing** — instead `Read skills/shark-rider/verbs/consult.md` and follow it with `agent=<agent>` and `referent=<referent>`.

**Negative example:** "tell me about the cx-designer" → this is a query *about* an agent, not a request to consult one; fall through to NL routing below.

If the agent name does not resolve to a known persona, fall through to normal NL routing below.

## Natural-language questions

Translate prose into read-only shark queries, then summarize. Examples:

| Ask | Command(s) |
|-----|-----------|
| "show blocked tasks" | `shark task list --blocked` |
| "what's in progress" | `shark task list --status in_<phase>` (resolve phase from the workflow) |
| "status of E01" | `shark status E01` |
| "who's working on what" | `shark claims` |
| "next up for E01-F02" | `shark status E01-F02` + `shark status transitions E01-F02` |

Prefer `--field` for single values; never pipe JSON through `head`/`grep`/`jq`/`python`.

## 2.x command vocabulary (use these, not 1.x)

| Do | Not |
|----|-----|
| `shark status set <key> <status>` | `shark task set-status …` |
| `shark status advance <key> --outcome pass\|fail\|blocked` | `shark status advance --status …` / bare `next-status` |
| `shark status transitions <key>` | `shark status options …` |
| `shark claim / release / heartbeat / claims` | (1.x had no lease model) |

## Mutations

For status changes, notes, context, creates/updates/deletes, follow the patterns
in:
- `context/workflow-and-status.md` — status, outcomes, claim/lease
- `context/entity-crud.md` — create / update / delete
- `context/notes-context-docs.md` — notes, context, related docs

If the user clearly wants to *drive* an entity through its workflow, suggest
`/shark-rider run <key>` instead of advancing by hand.
