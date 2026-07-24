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

- Do not call claim, status-transition, approval, or automatic triage commands.
- Do not create a decision record before the operator has resolved that decision.
- Do not treat a completed status, prior implementation, or a note as proof
  that an architectural choice remains valid.
- Do not invent file paths, code evidence, requirements, acceptance, or
  approvals. Report evidence gaps and leave the decision open.
