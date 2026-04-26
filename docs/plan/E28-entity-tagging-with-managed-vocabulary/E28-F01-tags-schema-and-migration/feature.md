---
feature_key: E28-F01-tags-schema-and-migration
epic_key: E28
title: Tags Schema and Migration
description: Add the two new polymorphic tables (`tags`, `entity_tags`), their indexes, and six per-parent cascade-delete triggers to the database, bump CurrentSchemaVersion 13→14, and add the `idea` member to `models.EntityType`. Foundation feature that all other E28 work depends on.
order: 1
---

# Tags Schema and Migration

**Feature Key**: E28-F01-tags-schema-and-migration

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Thin Description

Deliver the pure-additive database migration that introduces tagging infrastructure: the `tags` vocabulary table (id, name UNIQUE COLLATE NOCASE, timestamps), the `entity_tags` polymorphic join (entity_type CHECK-constrained, entity_id, tag_id FK, UNIQUE triple), the three supporting indexes, and six `entity_tags_cascade_delete_<entity>` triggers mirroring the existing `entity_notes_cascade_delete_*` pattern. Includes the `migrateAddTagsAndEntityTags` migration function, a `CurrentSchemaVersion` bump from 13 to 14, the one-line `models.EntityType` extension adding `EntityTypeIdea`, and the developer callout documenting the one-time `skip_migrations: false` toggle. Verified idempotent on both SQLite and Turso backends. This feature is the foundation on which every other E28 feature depends; it ships with no user-visible CLI surface.

**Integration points:** `internal/db/db.go` (migration + version), `internal/models/entity_type.go` (enum), `internal/models/tag.go` (new domain type + name regex), new cascade triggers attached to `epics`, `features`, `tasks`, `bugs`, `change_cards`, `ideas`.

**Architecture refs:** ADR-1 (two-table polymorphic design), ADR-4 (tag name validation), ADR-10 (idea EntityType), §3.1–§3.5 (schema, triggers, migration strategy).

**Execution order:** 1 — blocks F02, F03, F04, F05, F06.

---

*Last Updated*: 2026-04-22
