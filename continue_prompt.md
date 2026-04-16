---
timestamp: 2026-04-14T22:15:00Z
branch: E07-F39-viewer-entity-relationship-service
feature: E07-F39
status: feature completed, changes staged but NOT committed
---

# Resume: E07-F39 — Commit and PR

## What's Done

Feature **E07-F39** ("Remove legacy relationship tables and dual-path query code") is **complete** in shark. Both tasks passed all quality gates (code review, QA, UAT).

Check status: `./bin/shark get E07-F39 --json --field=status` → `completed`

## What Needs to Happen

### Step 1: Commit the staged changes

There are 21 staged files (deletions + modifications from T-E07-F39-001 and T-E07-F39-002) plus 58 untracked files (code_review reports, docs) that should be added selectively.

Key staged changes:
- `D internal/cli/commands/migrate_relationships.go`
- `D internal/models/{task,epic,feature}_relationship.go`
- `D internal/repository/{task,epic,feature}/relationship.go`
- `D internal/repository/relationship_repositories_test.go`
- `D internal/repository/task_relationship_repository_test.go`
- `D internal/repository/task/relationship_test.go`
- `M internal/repository/entityrel/repository.go` — new Feature/EpicKeyAdapters
- `M internal/config/template/helpers.go` — wired new adapters
- `M internal/services/viewer_service.go` — viewer uses EntityRelationshipService
- `M internal/viewer/server/wire.go` — adapter wiring removed
- `M internal/db/db.go` (in git history — bump to schema v12 + DROP TABLE migration)

Untracked files to add selectively:
- `code_review/20260414-*-T-E07-F39-*` — QA/UAT reports for this feature
- `docs/plan/E07-enhancements/E07-F39-*/spec.md`
- `docs/plan/E07-enhancements/E07-F39-*/test-plan.md`
- `docs/plan/E07-enhancements/E07-F39-*/code_review/`
- `task_reviews/E07-F39-task-review.md`
- `docs/workflow/activity.jsonl`

Do NOT add the E27 code_review files — those belong to a different branch.

Commit command:
```bash
git add \
  internal/ \
  docs/plan/E07-enhancements/E07-F39-remove-legacy-relationship-tables-and-dual-path-qu/ \
  code_review/20260414-*-T-E07-F39-* \
  task_reviews/E07-F39-task-review.md \
  docs/workflow/activity.jsonl \
  continue_prompt.md
git commit -m "feat(E07-F39): remove legacy relationship tables and dual-path query code

- Delete task_relationships, feature_relationships, epic_relationships repos and models
- Add DROP TABLE migration (CurrentSchemaVersion 11→12)
- Remove dual-path query fallbacks from dependency.go
- Add EntityRelFeatureKeyAdapter and EntityRelEpicKeyAdapter
- Replace viewer taskRelAdapter with per-task EntityRelationshipService calls
- All tests pass (54 packages), lint clean

IMPORTANT: Set skip_migrations=false in .sharkconfig.json before next Turso command,
then reset to true after migration runs.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

### Step 2: Create PR

```bash
gh pr create \
  --title "feat(E07-F39): remove legacy relationship tables and dual-path query code" \
  --base main \
  --body "..."
```

## Context for Code Review

- **spec.md**: `docs/plan/E07-enhancements/E07-F39-remove-legacy-relationship-tables-and-dual-path-qu/spec.md`
- **test-plan.md**: same directory
- **Task specs**: `docs/plan/.../tasks/T-E07-F39-001.md` and `T-E07-F39-002.md`
- **UAT reports**: `code_review/20260414-210817-T-E07-F39-001-uat.md` and `code_review/20260414-215718-T-E07-F39-002-uat.md`

## Migration Reminder

Schema version bumped 11→12. The migration adds `DROP TABLE IF EXISTS` for `task_relationships`, `feature_relationships`, `epic_relationships`.

On Turso: temporarily set `"skip_migrations": false` in `.sharkconfig.json`, run any `shark` command once, then reset to `true`. See `.claude/rules/database-critical.md`.

## Parallel Execution Notes

This is a single-step commit + PR task — no parallelism needed. Run quality gate once before committing if unsure:
```bash
make fmt && make lint && make test
```
