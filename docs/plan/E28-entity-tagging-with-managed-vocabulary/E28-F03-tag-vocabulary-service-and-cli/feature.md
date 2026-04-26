---
feature_key: E28-F03-tag-vocabulary-service-and-cli
epic_key: E28
title: Tag Vocabulary Service and CLI
description: Build `TagService` vocabulary admin operations (list, add, rm, rename) backed by `TagRepository`, gated through the F02 MaintainerGate. Ship the `shark tags list|add|rm|rename` CLI command group including `--pass` flag, `--force` on rm, and rename collision handling.
order: 3
---

# Tag Vocabulary Service and CLI

**Feature Key**: E28-F03-tag-vocabulary-service-and-cli

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Thin Description

Deliver the closed-vocabulary admin surface. Adds the `TagRepository` (CRUD over `tags` including case-insensitive name lookup and a `UsageCount(tagID)` query that consults `entity_tags`) under `internal/repository/tag/`, and a `TagService` owning business rules: name normalization (lowercase-ASCII slug per ADR-4), uniqueness checks, rename-collision hard-error (ADR-8), `rm` blocks when in use unless `--force` (ADR-9), atomic rename that updates only `tags.name` so join rows stay immutable. Every mutating service method (`AddTag`, `RemoveTag`, `RenameTag`) invokes `MaintainerGate.Authorize` up front and `RecordSuccess` on success. Ships the thin CLI command group `shark tags list` (open to all users), `shark tags add <name> [--pass]`, `shark tags rm <name> [--pass] [--force]`, `shark tags rename <old> <new> [--pass]` — each a thin wrapper that parses flags, calls the service, and formats output. Errors from the gate surface actionable stderr text that points at `.sharkconfig.json` (SC-3).

**Integration points:** new `internal/repository/tag/tag_repository.go`, new `internal/services/tag_service.go`, new `internal/cli/commands/tags.go`, wired in `services_global.go` via `GetTagService()`.

**Architecture refs:** ADR-2 (gate consumption), ADR-4 (name normalization), ADR-8 (rename semantics), ADR-9 (rm-in-use policy), §1.3, §4.1.

**Execution order:** 3 — depends on F01 (tables exist) and F02 (gate available). Blocks F04 (entity tag attach reuses TagService.ValidateNames and TagRepository.GetByName).

---

*Last Updated*: 2026-04-22
