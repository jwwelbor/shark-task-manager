# Project Overview

**Shark Task Manager** is a Go-based CLI tool and HTTP API for managing project tasks, features, and epics with AI-driven development workflows. It uses SQLite for persistence and follows clean architecture principles.

---

## ⚠️ Mandatory Quality Gate

**After ANY Go code changes, ALWAYS run before declaring work complete:**
```bash
make fmt && make lint && make test
```
Fix all failures. No exceptions. See @.claude/rules/development-workflows.md for details.

---

## ⚠️ Critical Warnings

See @.claude/rules/database-critical.md for full details.

**NEVER delete shark-tasks.db** - it's the single source of truth for all project data. Deleting it causes data loss and sync errors.

**DO NOT**:
- Run `make clean` during development (deletes database)
- Use `rm shark*` glob patterns
- Delete database to fix sync errors
- Modify task files during sync operations

---

## Navigation Guide

This project uses modular documentation. Rules are loaded automatically based on which files you're working with:

### Always-Loaded (Base Context)
- **Quick Reference**: @.claude/rules/quickref.md - Build, test, common commands
- **Database Critical**: @.claude/rules/database-critical.md - Critical DB warnings & recovery
- **Development Workflows**: @.claude/rules/development-workflows.md - Task creation, lifecycle, patterns

---

## Key Concepts

### Dual Key Format (Slug Architecture)

Shark supports both numeric and human-readable slugged keys:

**Epics**: `E04` or `E04-epic-name`
**Features**: `E04-F02` or `E04-F02-feature-name` or `F02` or `F02-feature-name`
**Tasks**: `T-E04-F02-001` or `T-E04-F02-001-task-name` or `E04-F02-001` (short format)

All keys are **case insensitive**: `E07`, `e07`, `E07-user-management` all work.

Slugs are auto-generated from titles and both formats work in all commands.

### Task Lifecycle

Shark loads its workflow definitions (statuses, transitions, agent routing)
from per-entity YAML files referenced by `workflow_config` in
`.sharkconfig.json`. `workflow_config` may point at:

- a **directory** of per-entity YAML files (the default, `shark-data/workflow/`
  with `task.yaml`, `feature.yaml`, `epic.yaml`, …), or
- a **master index file** that maps each entity to its workflow file (see
  [Route-Based Workflow Guide](docs/guides/route-based-workflow.md) §3).

`shark admin install-shark-data` materializes the `shark-data/` tree (workflows,
prompts, skills, agents) and leaves your `shark-data/overrides/` subtree
untouched. `shark admin init` only creates the database, `docs/plan/`, and
`.sharkconfig.json` — content is served from the embedded bundle by default.

> A bare Shark 1.x JSON workflow file (e.g. `.sharkworkflow.json`) is **no
> longer a valid `workflow_config` target** — the loader rejects it with a
> migration hint because an explicit JSON target overrides the embedded
> defaults. To migrate with no disk bundle, remove the `workflow_config` line
> from `.sharkconfig.json` or set it to an empty string; if a root
> `.sharkworkflow.json` exists, remove or rename it too before expecting
> embedded defaults. To migrate with editable workflow files, run
> `shark admin install-shark-data`; it extracts the content bundle and rewrites
> deprecated JSON targets to the installed bundle's workflow directory.

---

## Documentation References

- **Architecture Details**: @.claude/rules/architecture.md
- **CLI Reference (Unified)**: @docs/cli-reference/README.md
- **Workflow Profiles Guide**: @docs/guides/workflow-profiles.md
- **Route-Based Workflow (Shark 2.x)**: @docs/guides/route-based-workflow.md
- **Turso Cloud Setup**: @docs/TURSO_QUICKSTART.md
- **Turso Migration Guide**: @docs/TURSO_MIGRATION.md

---

## Rule 1 — Think Before Coding
State assumptions explicitly. If uncertain, ask rather than guess.
Present multiple interpretations when ambiguity exists.
Push back when a simpler approach exists.
Stop when confused. Name what's unclear.

## Rule 2 — Simplicity First
Minimum code that solves the problem. Nothing speculative.
No features beyond what was asked. No abstractions for single-use code.
Test: would a senior engineer say this is overcomplicated? If yes, simplify.

## Rule 3 — Surgical Changes
Touch only what you must. Clean up only your own mess.
Don't "improve" adjacent code, comments, or formatting.
Don't refactor what isn't broken. Match existing style.

## Rule 4 — Goal-Driven Execution
Define success criteria. Loop until verified.
Don't follow steps. Define success and iterate.
Strong success criteria let you loop independently.

## Rule 5 — Use the model only for judgment calls
Use me for: classification, drafting, summarization, extraction.
Do NOT use me for: routing, retries, deterministic transforms.
If code can answer, code answers.

## Rule 7 — Surface conflicts, don't average them
If two patterns contradict, pick one (more recent / more tested).
Explain why. Flag the other for cleanup.
Don't blend conflicting patterns.

## Rule 8 — Read before you write
Before adding code, read exports, immediate callers, shared utilities.
"Looks orthogonal" is dangerous. If unsure why code is structured a way, ask.

## Rule 9 — Tests verify intent, not just behavior
Tests must encode WHY behavior matters, not just WHAT it does.
A test that can't fail when business logic changes is wrong.

## Rule 10 — Checkpoint after every significant step
Summarize what was done, what's verified, what's left.
Don't continue from a state you can't describe back.
If you lose track, stop and restate.

## Rule 11 — Match the codebase's conventions, even if you disagree
Conformance > taste inside the codebase.
If you genuinely think a convention is harmful, surface it. Don't fork silently.

## Rule 12 — Fail loud
"Completed" is wrong if anything was skipped silently.
"Tests pass" is wrong if any were skipped.
Default to surfacing uncertainty, not hiding it.

## Rule 13 - Research Agents
Research agents don't need to run at Opus or Fable level. Launch with Sonnet of Haiku
depending on the complexity of the task.