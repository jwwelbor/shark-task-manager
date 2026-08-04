# /shark-rider walkthrough — Solution and architecture decision walk

Usage:

```
/shark-rider walkthrough <entity-key|docs-path> [overall|section|decision-id]
```

This is an explicit Mode-3 collaboration action. It walks a selected Shark
entity or an authoritative project document through its solution decisions,
one at a time, and writes only the durable records the operator approves. It is not a workflow
transition, approval gate, or automatic backlog-creation command.

## Step 0 — Resolve the target

Require one target and accept the remaining text as an optional scope; default
scope is `overall`. Treat a target containing a path separator or ending in
`.md` as a document path. It must resolve to a regular Markdown file below the
project `docs/` directory; read it in full and retain its canonical relative
path. Otherwise resolve the entity and retain its canonical key, type, file
path, and parent information:

```bash
shark get <key> --json
```

Reject an unresolved key, an unresolved document, or traversal outside
`docs/`. Every artifact path must remain beneath the project `docs/` directory.
Any Shark entity may be discussed; document linking is limited to the entity
types supported by `shark related-docs`.

## Step 1 — Collect the authoritative context

For an entity target, read the returned entity file plus its documented
parent/child context as needed. Read related documents for supported types
using the appropriate flag:

```bash
shark related-docs list --epic=<epic-key> --json
shark related-docs list --feature=<feature-key> --json
shark related-docs list --task=<task-key> --json
shark related-docs list --bug=<bug-key> --json
shark related-docs list --change=<change-key> --json
```

For a document target, discover only entity keys and related records that the
document explicitly names; do not create a Shark association merely because a
topic appears similar. Also inspect the relevant `docs/product/` and
`docs/architecture/` records, including existing ADRs, the product Decision
Log, interaction maps, and cross-epic map. Treat Shark state and its linked
records as the relationship authority when an entity is known; use code and
documentation to ground recommendations, not to infer unrecorded approval.

Review outstanding Question entities as a separate input to every walk.
Outstanding means `open`, `answering`, or `ready_for_resolution`; page each
status with `--limit=100` and increasing `--offset` until a page is short:

```bash
shark question list --status=open --limit=100 --offset=0 --json
shark question list --status=answering --limit=100 --offset=0 --json
shark question list --status=ready_for_resolution --limit=100 --offset=0 --json
```

For a non-Question entity target, also inspect its direct Question blockers:

```bash
shark question blocking-for <entity-key> --limit=100 --offset=0 --json
```

Read each potentially relevant Question with `shark question get <key> --json`
and its linked records with `shark related-docs list --question=<key> --json`.
Prioritize Questions that directly block the target or explicitly name the
target or document. Include every material Question in the decision queue and
report reviewed-but-out-of-scope Questions separately. Do not infer a new
association while reviewing it.

## Step 2 — Retrieve the craft skill and begin the walk

```bash
shark skill get solution-walkthrough
```

If the bundle is unavailable, report that and stop. Follow the retrieved skill
with the collected context. Present the decision queue and a recommended
dependency order, then wait for the operator's confirmation before walking the
first decision. For each decision, investigate first, agree its purpose and
framing, lead with a recommendation, and wait for an approve/amend/redirect
response. A previously documented important choice may be ratified only after
this same review; record it as **Reviewed and confirmed** in the authoritative
source rather than duplicating it.

When the operator approves an answer to an in-scope Question, first write or
update the durable decision record that supplies its evidence pointer. Then use
the Question workflow to record that answer. Read `shark next <question-key>
--json`; continue only when it routes that Question to the walkthrough
operator. Claim the Question under that returned responder identity, record the
approved answer, and release the claim. Continue only when `current_responder`
in the `shark next` JSON matches the authenticated walkthrough operator.
Capture the `session_id` returned by the claim JSON and use that exact value in
the response and release commands:

```bash
shark claim <question-key> --by=<current-responder> --json
shark question respond <question-key> --session=<session-id> --responder=<current-responder> --summary="<approved answer>" --evidence-pointer=<durable-record-path>
shark release <question-key> --session=<session-id>
```

If `current_responder` is not the walkthrough operator,
record the approved decision in its durable source and hand the Question to the
configured responder. Do not infer or impersonate a responder. A response is
not a resolution: do not automatically resolve a Question after recording the
response.

## Step 3 — Persist and link approved outcomes

Use the skill's routing rule: product decisions belong in
`docs/product/progress.md`; shared architecture belongs in
`docs/architecture/adr/`; epic decisions update the epic architecture and maps;
local decisions update the authoritative entity spec/design and, only when
needed, its local decision record. Do not create duplicate ADRs or decision
files when the authoritative record already exists.

After successfully writing a durable entity-local, ADR, or other decision
document, link it to a supported selected entity or an entity explicitly named
by the document:

```bash
shark related-docs add "Decision Record" <path> --epic=<epic-key>
shark related-docs add "Decision Record" <path> --feature=<feature-key>
shark related-docs add "Decision Record" <path> --task=<task-key>
shark related-docs add "Decision Record" <path> --bug=<bug-key>
shark related-docs add "Decision Record" <path> --change=<change-key>
```

For an idea or tech-debt item, add a reference note only after the canonical
document exists. A document-only walk without an evidenced entity association
has no Shark link:

```bash
shark create note <key> "Decision record: <path>" --type=reference
```

Use an accurate title when the artifact is an ADR or product decision rather
than a local decision record. These links are discovery references, not
approval evidence or lifecycle changes.

## Boundaries

- Do not claim or modify a Question except to record an operator-approved
  response through the documented Question workflow.
- Do not call status-transition, approval, or automatic triage commands.
- Do not resolve, withdraw, supersede, or otherwise close a Question as part
  of the walk.
- Do not create a decision record before the operator has resolved that decision.
- Do not treat a completed status, prior implementation, or a note as proof
  that an architectural choice remains valid.
- Do not invent file paths, code evidence, requirements, acceptance, or
  approvals. Report evidence gaps and leave the decision open.
