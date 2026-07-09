# /shark sync — Filesystem → shark reconciliation (one epic)

Scan one epic's folder under `docs/plan/` and create/update shark entities to
match. The **filesystem is the source of truth** — shark may be empty, stale,
or missing entities entirely. Mode-3 local recipe: this verb does the walking;
the CLI does every read/write.

> **EXPLICIT USER INVOCATION ONLY.** Never run this verb autonomously. If a
> sync looks needed, ask the user first. Premature syncs can overwrite shark
> state, duplicate entities, or reset manually-set statuses — and can touch
> entities under active leases or review (another agent's UAT run, a claimed
> task).
>
> **Not for single entities.** To register one already-authored doc, use
> `shark create <type> "<title>" --key=<KEY> --file=<path>` — additive-only and
> lease-safe. See `context/entity-crud.md` §"Registering an already-authored
> file". Sync is bulk drift reconciliation only.

## Usage

```
/shark sync E14
```

Argument: the epic key (e.g. `E14`).

## Step 1 — Discover the epic folder

```bash
ls -d docs/plan/<KEY>-*/     # e.g. docs/plan/E14-ingestion-spine-poc-.../
```

If no match, check alternate roots (`backend/docs/plan/`, project root). Read
`epic.md` at the folder root — that file is the epic definition.

## Step 2 — Scan the folder structure

```
docs/plan/E##-{slug}/
├── epic.md                          # Epic definition
├── *.md                             # Epic-level docs (requirements, scope, …)
├── E##-F##-{slug}/                  # Feature folders
│   ├── feature.md                   # Feature definition (newer convention)
│   ├── PRD_F##_*.md                 # Feature definition (older convention)
│   ├── *.md                         # Feature-level design docs
│   └── tasks/
│       └── T-E##-F##-###.md         # Task files
```

Discovery rules, in order: epic = `epic.md` at root; features = subdirectories
matching `E##-F##*` (feature doc = `feature.md`, else `PRD_F##_*.md` in the
feature folder or at epic level); tasks = `.md` files under a feature's
`tasks/` (legacy `prps/` dirs count as tasks for backward compatibility).

## Step 3 — Extract metadata per file

Prefer YAML frontmatter when present (`title`, `description`, `status`, `key`,
`epic_key`/`feature_key`/`task_key`, `agent`, `priority`, `execution_order`,
`depends_on`) — **frontmatter is not guaranteed**. Always supplement with
content parsing: title = first `# Heading`; description = text under `## Goal`
/ `## Overview` / `## Purpose` / `## Feature Overview` (first present); key
derived from folder/filename (`T-E14-F01-002.md` → `E14-F01-002`). Status:
only from an explicit frontmatter `status` — never guessed.

## Step 4 — Sync the epic

```bash
shark get E## --json                 # existence check
# missing → create linked to its existing file
shark create epic "Title from epic.md" --key=E## --file="docs/plan/E##-{slug}/epic.md"
# exists but drifted → update (no --status)
shark update E## --title="Title from epic.md"
```

Link `epic.md` and notable epic-level docs:

```bash
shark related-docs add "Epic Definition" docs/plan/E##-{slug}/epic.md --epic=E##
```

## Step 5 — Sync features

Per feature folder: read the feature doc, then

```bash
shark get E##-F## --json
# missing → create linked to the existing doc
shark create feature E## "Feature Title" --key=F## --file="{path-to-feature-doc}"
# exists but drifted
shark update E##-F## --title="Feature Title"
shark related-docs add "Feature PRD" {path-to-feature-doc} --feature=E##-F##
```

Also link design docs found in the feature folder.

## Step 6 — Sync tasks

Per task file: extract metadata, determine key + parent feature, then

```bash
shark get {task-key} --json
shark create task E##-F## "Task Title" --file="{task-file-path}"     # missing
shark update {task-key} --title="Task Title" --file="{task-file-path}"  # drifted
```

Status only when the file's frontmatter explicitly declares one and it differs:

```bash
shark status set {task-key} {status-from-file}
```

Sync `--agent` / `--priority` / `--order` from frontmatter when present.

## Step 7 — Verify and report

```bash
shark get E## --json          # epic with feature/task counts
shark list E## --json         # features
shark list E## F## --json     # spot-check a feature's tasks
```

Report: entities created/updated (keys + titles, task counts per feature),
files that couldn't be parsed or matched, status mismatches resolved.

## Invariants

- **Never delete** shark entities absent from disk — report orphans for the
  user to review.
- **Never set a guessed status** — frontmatter-declared only.
- **File naming varies** — discover what's there; don't assume one convention.
- Features with no tasks yet are normal — sync what exists.
