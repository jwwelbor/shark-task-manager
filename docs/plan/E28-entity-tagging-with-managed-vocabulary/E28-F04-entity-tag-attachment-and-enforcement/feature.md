---
feature_key: E28-F04-entity-tag-attachment-and-enforcement
epic_key: E28
title: Entity Tag Attachment and Enforcement
description: Wire `--tag` onto `create` and `update` for all six entity types, add `<entity> tag add|rm` retroactive-tagging subcommands, deliver the `EntityTagRepository` + `TagService.AttachMany/Detach/EnforceRequired` service methods, and enforce `.sharkconfig.json` `tag_required_for` at service layer during create paths.
order: 4
---

# Entity Tag Attachment and Enforcement

**Feature Key**: E28-F04-entity-tag-attachment-and-enforcement

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Thin Description

Deliver the polymorphic attach/detach layer and the per-entity-type enforcement rule. Adds the `EntityTagRepository` (insert, delete, list-by-entity, list-by-tag over `entity_tags`) under `internal/repository/tag/`, plus `TagService.AttachMany(ctx, entityType, entityID, names)`, `DetachOne(ctx, entityType, entityID, name)`, and `EnforceRequired(ctx, entityType, names)` which reads `cfg.TagRequiredFor` and fails creates when a required entity type has zero tags (ADR-7). Integrates the hooks into `TaskService.CreateTask`, `FeatureService.CreateFeature`, `EpicService.CreateEpic`, `BugService.CreateBug`, `ChangeCardService.Create`, `IdeaService.Create` (call `EnforceRequired` pre-insert, `AttachMany` post-insert) and into `*.UpdateXxx` methods (`AttachMany` only — `--tag` on update is additive, never replaces). Adds the `--tag` repeated flag to every create/update CLI and the `shark <entity> tag add <key> <name>` / `shark <entity> tag rm <key> <name>` retroactive subcommands for all six entity types. Unregistered tag names surface the prescribed stderr shape from SC-2 (vocabulary listing + exact `shark tags add` command string).

**Integration points:** new `internal/repository/tag/entity_tag_repository.go`, extensions in `internal/services/{task,feature,epic,bug,change_card,idea}_service.go`, flag + subcommand edits across `internal/cli/commands/{task,feature,epic,bug,change_card,idea}.go`, `Config.TagRequiredFor` wiring from F02's config work.

**Architecture refs:** ADR-5 (entity_tags shape — though query-side is F05), ADR-7 (enforcement location), ADR-10 (idea as EntityType), §4.3 (integration per entity command).

**Execution order:** 4 — depends on F01 (schema) and F03 (TagService + TagRepository). Blocks F05 (attach must exist before query).

---

*Last Updated*: 2026-04-22
