# /shark update-docs — Diff-driven architecture-doc refresh

Keeps `docs/architecture/*.md` in sync with the code, updating only what the diff
since the last refresh touched. **Architecture docs only** — not entity specs.

Usage:
```
/shark update-docs
/shark update-docs --allow-dirty    # fold the working-tree diff in too
```

## Procedure

### 1. Clean-worktree guard
```bash
git status --porcelain
```
If the worktree is **dirty** and `--allow-dirty` was NOT passed → **abort** with
guidance ("commit or stash first, or re-run with --allow-dirty"). This prevents
the baseline from advancing past uncommitted code.

### 2. Baseline
Read `docs/architecture/.update-docs-state` (a stored commit SHA).

- **First run** (file absent): do a **full generation** using the bundle's
  doc-gen workflow — resolve the content bundle root and follow
  `<bundle>/skills/research/workflows/project-init.md`'s doc-gen path. Then write
  the baseline (step 4). Stop.

### 3. Diff and revise
```bash
git diff <baseline>..HEAD --stat        # + working tree when --allow-dirty
```
Map changed source paths to the affected `docs/architecture/*.md` and revise
**only those files**. Leave untouched docs alone.

### 4. Restamp
Only **after** edits are produced **and** the worktree was clean at start
(i.e. not `--allow-dirty`), update the baseline to `HEAD`:
```bash
git rev-parse HEAD > docs/architecture/.update-docs-state
```
With `--allow-dirty`, do not advance the baseline (working-tree changes aren't
committed yet) — note this to the user.

## Notes
- If the bundle doc-gen workflow is unavailable on a first run, say so and stop
  rather than guessing a structure.
