# /shark project — Pre-epic setup namespace

Namespace dispatcher for pre-epic project activities. Each activity is a
distinct operation — they are **not a sequence**; run only the ones you need.

Usage: `/shark project <activity> [args]`

## Activities

| Activity | Description |
|----------|-------------|
| `bootstrap` | Bootstrap architecture docs for a new or brownfield project |
| `brownfield-analysis` | Deep analysis and documentation of an existing codebase |
| `product-design` | Run the product-design (D01–D14) workflow |

## Dispatch

### No arguments — print this menu and stop

When called with no arguments, print the activities table above, explain that
these are **independent activities (not a sequence)**, and stop. Do not
auto-run any activity.

### `bootstrap`

Read and follow:
```
shark skill get research workflows/bootstrap.md
```
If that command fails because the workflow content is unavailable, print:
> `bootstrap` content is not available in this project's bundle
> (`shark skill get research workflows/bootstrap.md` failed). Check the
> installed shark version or `shark_data_path`.

Then stop — do not fall back to a hardcoded procedure.

After the workflow completes, suggest one Shark-native next step:
- If product docs or a clear product vision are still missing, suggest
  `/shark project product-design` for the full D01–D14 flow.
- If product direction already exists, suggest `/shark run <key>` when there
  is a concrete epic, feature, or task ready to drive.

### `brownfield-analysis [path]`

Read `skills/shark/verbs/brownfield-analysis.md` and follow it, passing any
remaining arguments (e.g. the target path) through to that procedure.

### `product-design [args]`

Read `skills/shark/verbs/product-design.md` and follow it, passing any
remaining arguments through to that procedure.

### Unknown activity

If the first argument is not one of the three activities above, print:

> Unknown activity: `<x>`. Valid activities: bootstrap, brownfield-analysis, product-design

Then stop.

---

## Ops and recurring work

One-off activities in this namespace are for project setup. Recurring
operational work (infrastructure maintenance, dependency updates, scheduled
reviews, etc.) belongs as Shark entities — tasks, features, or change-cards —
so that it participates in sprint planning, ownership, and progress tracking.

See `docs/guides/ops-as-entities.md` for the recommended patterns.

---

## Progress record (run after any activity completes)

After any activity above completes successfully, update the project progress
record. This is a best-effort Write craft operation; non-fatal if the file
system is read-only.

### Steps

1. **Check for `docs/product/progress.md`.**
   - If it does not exist, create `docs/product/` directory if needed, then
     seed `docs/product/progress.md` by copying the template at
     `file_templates/progress.md` (repo root). Read that file and write its
     contents verbatim as the initial `docs/product/progress.md`.

2. **Regenerate the Setup Checklist section** by checking the filesystem:
   - `[x]` if `docs/architecture/` directory exists, otherwise `[ ]` — label:
     `Architecture docs (\`docs/architecture/\`) — present` / `— not present`
   - For `docs/architecture/tech-stack.md`:
     - If exists and contains the text "Greenfield — Provisional placeholder": `[ ] Tech stack (\`tech-stack.md\`) — provisional, reconcile after D04`
     - If exists and does NOT contain that text: `[x] Tech stack (\`tech-stack.md\`) — finalized`
     - If not exists: `[ ] Tech stack (\`tech-stack.md\`) — not present`
   - `[x]` if `docs/product/D01-vision-statement.md` exists, otherwise
     `[ ]` — label: `Vision statement (D01-vision-statement.md) — present` / `— not present`
   - `[x]` if `docs/product/D04-feasibility-report.md` exists, otherwise
     `[ ]` — label: `Feasibility report (D04-feasibility-report.md) — present` / `— not present`

   Replace the entire checklist block (between the `<!-- DERIVED -->` comment
   and the next `##` heading) with the freshly computed lines.

3. **Append a timestamped decision-log entry** inside the Decision Log section,
   after the last existing entry (or after the `<!-- APPEND-ONLY -->` comment
   if no entries yet):
   ```
   **YYYY-MM-DD** — <activity>: <one-sentence outcome summary>
   ```

4. Write the updated `docs/product/progress.md`.
