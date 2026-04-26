---
feature_key: E28-F02-reusable-maintainer-authorization-gate
epic_key: E28
title: Reusable Maintainer Authorization Gate
description: Build the general-purpose `internal/auth/maintainer` package that owns password verification and sudo-style file-backed session caching. Exposes `Gate.Authorize(ctx, pass)` and `Gate.RecordSuccess(ctx)` for any future admin command to consume; F03 is its first client.
order: 2
---

# Reusable Maintainer Authorization Gate

**Feature Key**: E28-F02-reusable-maintainer-authorization-gate

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Thin Description

Deliver a stand-alone, tag-agnostic authorization package at `internal/auth/maintainer` that any sensitive admin command can consume. The package implements the `Gate` interface (`Authorize(ctx, providedPass) error`, `RecordSuccess(ctx) error`), SHA-256 hex digest password comparison, and a sudo-style file-backed cache at `~/.cache/shark/<project-hash>/maintainer.session` honoring `XDG_CACHE_HOME`. Cache window defaults to 60 seconds and is configurable via constructor. Adds the `maintainer` object to `.sharkconfig.json` (`password_hash`, `cache_window_seconds`) wired through `internal/config`, plus the bootstrap helper `shark admin maintainer set-password` that accepts a plaintext password and writes the digest so users never type a hash. Emits an OpenTelemetry span (`maintainer.authorized=true|false`) without leaking the password or stored hash. Has no shark-domain dependencies (no tag types, no entity types) so future admin commands adopt it in three lines.

**Integration points:** new `internal/auth/maintainer/` package, new `internal/config/maintainer.go` + `Config.Maintainer` field, new `shark admin maintainer set-password` command, new `cli.GetMaintainerGate()` accessor in `internal/cli/services_global.go`.

**Architecture refs:** ADR-2 (reusable gate), ADR-3 (cache location), ADR-6 (SHA-256 hashing), §1.3 (service layering), §4.6 (future-consumer contract), §6 (observability rules).

**Execution order:** 2 — depends on nothing; blocks F03 (gate is the first consumer). Can be developed in parallel with F01.

---

*Last Updated*: 2026-04-22
