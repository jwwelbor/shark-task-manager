---
feature_key: E28-F05-tag-based-querying-in-list-and-search
epic_key: E28
title: Tag-Based Querying in List and Search
description: Add the `--tag` repeated filter to `shark list` (at every scope — top-level, per-epic, per-feature, per-entity-type lists) and to `shark search`, implementing the AND conjunction default (ADR-5). Exposes `TagService.EntityIDsByTags` for consumers that need the tagged-entity set.
order: 5
---

# Tag-Based Querying in List and Search

**Feature Key**: E28-F05-tag-based-querying-in-list-and-search

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Thin Description

Deliver the read path that makes tags useful. Adds `TagService.EntityIDsByTags(ctx, entityType, names, op=AND) ([]int64, error)` backed by `EntityTagRepository.FilterByTags` which generates N `EXISTS` sub-clauses joined with AND per ADR-5 (SQLite lacks `unnest`). The `--tag=<name>` repeated CLI flag is added to `shark list` at every scope (epic/feature/task/bug/change/idea, plus the top-level dispatcher) and to `shark search`; commands collect the flag into `[]string` and the relevant list service intersects its existing filter output with the tagged-entity set from `TagService`. Existing filters (status, agent, priority, etc.) compose with `--tag` via AND. The search service treats tag filtering as a post-filter over full-text results rather than reworking the search pipeline (§4.4). Also delivers a `Tags` field populated on `Get` responses for all six entity types (service fills from `EntityTagRepository.ListByEntity`) so detail views show tags alongside other attributes — critical for F06's API DTOs.

**Integration points:** extensions in `internal/services/tag_service.go`, `internal/repository/tag/entity_tag_repository.go`, edits to every list command in `internal/cli/commands/`, `shark search` in `internal/cli/commands/search.go`, `Get<Entity>` service methods to populate `Tags`.

**Architecture refs:** ADR-5 (AND default, EXISTS-per-tag SQL shape), §4.3, §4.4.

**Execution order:** 5 — depends on F04 (rows must be attachable before queries are meaningful). Blocks F06 (viewer API consumes EntityIDsByTags and the per-entity Tags field).

---

*Last Updated*: 2026-04-22
