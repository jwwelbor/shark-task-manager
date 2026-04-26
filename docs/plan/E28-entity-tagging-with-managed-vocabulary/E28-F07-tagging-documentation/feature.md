---
feature_key: E28-F07-tagging-documentation
epic_key: E28
title: Tagging Documentation
description: Write the user-facing documentation that closes SC-10, a new `docs/cli-reference/tags.md` page covering the `shark tags` group and `--pass`, updates to `configuration.md` for `tag_required_for` and the `maintainer` block, `--tag` additions on every `create`/`update`/`list`/`search` reference page, and the migration callout for `skip_migrations, false`.
order: 7
---

# Tagging Documentation

**Feature Key**: E28-F07-tagging-documentation

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Thin Description

Close the epic's documentation acceptance gate (SC-10, UAT-9). Adds a new reference page `docs/cli-reference/tags.md` documenting `shark tags list|add|rm|rename`, the `--pass` flag, the `--force` flag on `rm`, maintainer-gate behavior (cache window, how to set the password via `shark admin maintainer set-password`), and error messages. Updates `docs/cli-reference/configuration.md` to document the new `tag_required_for` array field and the `maintainer` object (`password_hash`, `cache_window_seconds`) with examples and a pointer to the bootstrap helper. Adds `--tag=<name>` documentation (repeated flag, AND semantics on filters, additive semantics on update) to each existing reference page under `docs/cli-reference/` that mentions `create`, `update`, `list`, or `search` — including per-entity pages (`task-commands.md`, `feature-commands.md`, `epic-commands.md`, `bug-commands.md`, `change-commands.md`, and the core-commands page for `shark list`/`shark search`). Adds a migration note to the configuration or initialization docs explaining the one-time `skip_migrations: false` toggle required for v13->v14. Updates `docs/cli-reference/README.md` Command Groups table to reference the new `shark tags` page.

**Integration points:** `docs/cli-reference/tags.md` (new), `docs/cli-reference/configuration.md`, `docs/cli-reference/README.md`, per-entity command reference pages, `docs/cli-reference/core-commands.md` (if present).

**Architecture refs:** Epic §6 (UAT-9), §2 (SC-10), §5.2 (migration developer-callout text reused here for user-facing framing).

**Execution order:** 7 — depends on F03, F04, F05, F06 having finalized their user-facing surface. Parallel with F06 is acceptable provided doc updates lag API/CLI stabilization.

---

*Last Updated*: 2026-04-22
